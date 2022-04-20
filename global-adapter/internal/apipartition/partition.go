/*
 *  Copyright (c) 2021, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

/*
	Package apipartition contains logic related to the API labelling , persisting to the DB and
	updating the Redis cache with relavant information.
*/

package apipartition

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"

	msg "github.com/wso2/product-microgateway/adapter/pkg/messaging"

	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/cache"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/config"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/database"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/health"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/synchronizer"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/types"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/xds"
)

// RedisBlockedValue Step quota exceeded/reset org's redis value
const RedisBlockedValue = "blocked"

const featureStepQuotaLimiting = "FEATURE_STEP_QUOTA_LIMITING"

var configs = config.ReadConfigs()
var partitionSize = configs.Server.PartitionSize
var deployAdapterTriggered bool

// IsStepQuotaLimitingEnabled step quota limiting is enabled or not
var IsStepQuotaLimitingEnabled = getStepQuotaLimitingConfig()

// CacheAction is use as enum type for Redis cache update event type
type CacheAction int

// Cache update type
const (
	Add CacheAction = iota
	Remove
)

const (
	gwType                 string = "type"
	gatewayLabel           string = "gatewayLabel"
	envoy                  string = "Envoy"
	authorization          string = "Authorization"
	uuid                   string = "uuid"
	apiID                  string = "apiId"
	clientName             string = "global-adapter"
	productionSandboxLabel string = "Production and Sandbox"
	defaultGatewayLabel    string = "default"
	deleteEvent            string = "delete"
)

// PopulateAPIData - populating API information to Database and redis cache
func PopulateAPIData(apiEventsWithStartupFlag synchronizer.APIEventsWithStartupFlag, stmt *sql.Stmt) {
	apis := apiEventsWithStartupFlag.APIEvents
	var laAPIList []*types.LaAPIEvent
	var cacheObj []string
	var laLabels map[string]map[string]int // map of label hierarchy -> API UUID -> API ID
	var quotaStatus map[string]bool        // map of orgId -> isExceeded

	// load the db data at the startup to speedup the GA startup
	if apiEventsWithStartupFlag.IsStartup {
		laLabels = getAPILALabels()
		quotaStatus = getQuotaStatus()
	}

	for ind := range apis {
		for index := range apis[ind].GatewayLabels {
			// It is required to convert the gateway label to lowercase as the partition name is required for deploying k8s
			// services
			gatewayLabel := strings.ToLower(apis[ind].GatewayLabels[index])

			// when gateway label is "Production and Sandbox" , then gateway label set as "default"
			if gatewayLabel == strings.ToLower(productionSandboxLabel) {
				gatewayLabel = defaultGatewayLabel
			}
			var apiID int
			if apiEventsWithStartupFlag.IsStartup {
				apiID, _ = laLabels[gatewayLabel][apis[ind].UUID]
			} else {
				apiID = insertRecord(&apis[ind], gatewayLabel, stmt)
			}

			if apiID >= 0 {
				label := getLaLabel(gatewayLabel, apiID, partitionSize)
				logger.LoggerAPIPartition.Info("Label for : ", apis[ind].UUID, " and Gateway : ", gatewayLabel, " is ", label)

				isExceeded := false
				if apiEventsWithStartupFlag.IsStartup {
					isExceeded, _ = quotaStatus[apis[ind].OrganizationID]
				} else {
					isExceeded = isQuotaExceededForOrg(apis[ind].OrganizationID)
				}
				cacheKey := getCacheKey(&apis[ind], gatewayLabel)
				cacheValue := RedisBlockedValue
				if !isExceeded {
					cacheValue = getCacheValue(&apis[ind], label)
				}
				logger.LoggerAPIPartition.Debugf("Found cache key : %v and cache value: %v for organisation: %v",
					cacheKey, cacheValue, apis[ind].OrganizationID)
				// Push each key and value to the string array (Ex: "key1","value1","key2","value2")
				if cacheKey != "" {
					cacheObj = append(cacheObj, cacheKey)
					cacheObj = append(cacheObj, cacheValue)
				}

				apiState := types.LaAPIEvent{
					LabelHierarchy:   apis[ind].GatewayLabels[index],
					Label:            label,
					RevisionUUID:     apis[ind].RevisionID,
					APIUUID:          apis[ind].UUID,
					OrganizationUUID: apis[ind].OrganizationID,
				}
				laAPIList = append(laAPIList, &apiState)
			} else {
				logger.LoggerAPIPartition.Errorf("Error while fetching the API label UUID : %v in gateway : %v", apis[ind].UUID, gatewayLabel)
			}
		}
	}

	if len(cacheObj) >= 2 && !apis[0].IsReload {
		rc := cache.GetClient()
		cachingError := cache.SetCacheKeys(cacheObj, rc)
		if cachingError != nil {
			logger.LoggerAPIPartition.Errorf("Error setting cache keys in redis cache error: %s", cachingError.Error())
			return
		}
		logger.LoggerAPIPartition.Infof("Cache keys were successfully updated into redis cache at the startup(y/n) : %v", apiEventsWithStartupFlag.IsStartup)
		cache.PublishUpdatedAPIKeys(cacheObj, rc)
	}
	pushToXdsCache(laAPIList)
	if apiEventsWithStartupFlag.IsStartup {
		logger.LoggerAPIPartition.Info("All artifacts have been loaded to XDS cache in the startup. Hense marking readiness as true")
		health.Startup.SetStatus(true)
	}
}

func pushToXdsCache(laAPIList []*types.LaAPIEvent) {
	logger.LoggerAPIPartition.Debug("API List : ", len(laAPIList))
	switch n := len(laAPIList); {
	case n == 0:
		return
	case n == 1:
		xds.ProcessSingleEvent(laAPIList[0])
		return
	default:
		xds.AddMultipleAPIs(laAPIList)
		return
	}
}

// insertRecord always return the adapter label for the relevant API
// If the API is not in database, that will save to the database and return the api id
func insertRecord(api *synchronizer.APIEvent, gwLabel string, stmt *sql.Stmt) int {
	apiID := -1
	isNewID := false
	isExists := false
	for {
		isExists, apiID = isAPIExists(api.UUID, gwLabel)
		if isExists {
			logger.LoggerAPIPartition.Debug("API : ", api.UUID, " has been already persisted to gateway : ", gwLabel, " partitionId : ", apiID)
			break
		}
		apiID, isNewID = getAvailableID(gwLabel)
		if apiID < 0 { // Return -1 due to an error
			logger.LoggerAPIPartition.Errorf("Error while getting next available ID | hierarchy for api : %v in gateway : %v, apiId : %v", api.UUID, gwLabel, apiID)
		} else {
			_, err := database.ExecPreparedStatement(database.QueryInsertAPI, stmt, api.UUID, &gwLabel, apiID, api.OrganizationID)
			if err != nil {
				if strings.Contains(err.Error(), "duplicate key") {
					logger.LoggerAPIPartition.Debugf("API : %v in gateway : %v is already exists %v", api.UUID, gwLabel, err.Error())
					continue
				} else {
					logger.LoggerAPIPartition.Errorf("Error while writing partition information for API : %v in gateway : %v , %v",
						api.UUID, gwLabel, err.Error())
					apiID = -1
				}
			} else {
				logger.LoggerAPIPartition.Debugf("New API record persisted UUID : %v gatewayLabel : %v partitionId : %v isNewPartition : %v",
					api.UUID, gwLabel, apiID, isNewID)
				if isNewID {
					// Only if the incremental ID is a new one (instead of occupying avaliable vacant ID), new deployment trigger should happen.
					triggerNewDeploymentIfRequired(apiID, partitionSize, configs.Server.PartitionThreshold)
				}
			}
		}
		break
	}
	return apiID
}

// getAPILALabels get partition info from db
func getAPILALabels() map[string]map[string]int {
	var apiID int
	var labelHierarchy string
	var apiUUID string
	labels := make(map[string]map[string]int) // label hierarchy -> API UUID -> API ID
	row, err := database.ExecDBQuery(database.QueryGetAllLabels)
	if err == nil {
		for {
			if !row.Next() {
				logger.LoggerAPIPartition.Debug("No more partition label records exist for API partitions in database")
				break
			} else {
				row.Scan(&apiUUID, &labelHierarchy, &apiID)
				logger.LoggerAPIPartition.Debugf("API %v found in database with label : %v : label : %v ", apiUUID, labelHierarchy, apiID)
				if _, found := labels[labelHierarchy]; !found {
					labels[labelHierarchy] = make(map[string]int)
				}
				labels[labelHierarchy][apiUUID] = apiID
			}
		}
	} else {
		logger.LoggerAPIPartition.Error("Error when getting api partition label records from database")
	}
	return labels
}

// getAPILALabelsForOrg get partition info from db for an org
func getAPILALabelsForOrg(orgID string) map[string]map[string]int {
	var apiID int
	var labelHierarchy string
	var apiUUID string
	labels := make(map[string]map[string]int) // label hierarchy -> API UUID -> API ID
	row, err := database.ExecDBQuery(database.QueryGetAllLabelsPerOrg, orgID)
	if err == nil {
		for {
			if !row.Next() {
				logger.LoggerAPIPartition.Debug("No more partition label records exist for API partitions in database")
				break
			} else {
				row.Scan(&apiUUID, &labelHierarchy, &apiID)
				logger.LoggerAPIPartition.Debugf("API %v found in database with label : %v : label : %v ", apiUUID, labelHierarchy, apiID)
				if _, found := labels[labelHierarchy]; !found {
					labels[labelHierarchy] = make(map[string]int)
				}
				labels[labelHierarchy][apiUUID] = apiID
			}
		}
	} else {
		logger.LoggerAPIPartition.Error("Error when getting api partition label records from database")
	}
	return labels
}

// getQuotaStatus get partition info from db
func getQuotaStatus() map[string]bool {
	var orgID string
	var isExceeded bool
	quotaStatus := make(map[string]bool)
	row, err := database.ExecDBQuery(database.QueryQuotaStatus)
	if err == nil {
		for {
			if !row.Next() {
				logger.LoggerAPIPartition.Debug("No more quota status label records exist in database")
				break
			} else {
				row.Scan(&orgID, &isExceeded)
				logger.LoggerAPIPartition.Debugf("Org %v found in database with status : %v", orgID, isExceeded)
				quotaStatus[orgID] = isExceeded
			}
		}
	} else {
		logger.LoggerAPIPartition.Error("Error when getting quota status records from database")
	}
	return quotaStatus
}

// Return a boolean for API existance , int for incremental ID if the API already exists
func isAPIExists(uuid string, labelHierarchy string) (bool, int) {
	var apiID int
	row, err := database.ExecDBQuery(database.QueryIsAPIExists, uuid, labelHierarchy)
	if err == nil {
		if !row.Next() {
			logger.LoggerAPIPartition.Debug("Record does not exist for labelHierarchy : ", labelHierarchy, " and uuid : ", uuid)
		} else {
			row.Scan(&apiID)
			logger.LoggerAPIPartition.Debug("API already persisted : ", uuid, " for ", labelHierarchy)
			return true, apiID
		}
	} else {
		logger.LoggerAPIPartition.Error("Error when checking whether the API is exists. uuid : ", uuid, " ", err)
	}
	return false, apiID
}

// getAvailableID returns the next available inremental ID. For collect the next available ID , there are 2 helper functions.
// 1. getEmptiedId() -  Return if there any emptied ID. Return smallest first ID.If no emptied IDs available , then returns 0.
// 2. getNextIncrementalId() - If getEmptiedId return 0 , then this function returns next incremental ID.
// The second return value is false if the helper method 1 determines the ID. true otherwise.
func getAvailableID(hierarchyID string) (int, bool) {
	var nextAvailableID int = getEmptiedID(hierarchyID)
	newIDAssigned := false
	if nextAvailableID == 0 {
		nextAvailableID = getNextIncrementalID(hierarchyID)
		if nextAvailableID != -1 {
			newIDAssigned = true
		}
	}
	logger.LoggerAPIPartition.Debug("Next available ID for hierarchy ", hierarchyID, " is ", nextAvailableID)
	return nextAvailableID, newIDAssigned
}

// getEmptiedID observing emptied incremental ID
func getEmptiedID(hierarchyID string) int {
	var emptiedID int
	stmt, error := database.ExecDBQuery(database.QueryGetEmptiedID, hierarchyID)
	if error == nil {
		stmt.Next()
		stmt.Scan(&emptiedID)
		logger.LoggerAPIPartition.Debug("The next available id from deleted APIs | hierarchy : ", hierarchyID, " is : ", emptiedID)
	} else {
		logger.LoggerAPIPartition.Error("Error while getting next available id from deleted APIs | hierarchy : ", hierarchyID)
	}

	return emptiedID
}

// Return next ID . If error occur from the query level , then return -1.
func getNextIncrementalID(hierarchyID string) int {
	var highestID int
	var nextIncrementalID int
	stmt, error := database.ExecDBQuery(database.QueryGetNextIncID, hierarchyID)
	if error == nil {
		stmt.Next()
		stmt.Scan(&highestID)
		nextIncrementalID = highestID + 1
		logger.LoggerAPIPartition.Debug("Next incremental ID for hierarchy : ", hierarchyID, " is : ", nextIncrementalID)
	} else {
		nextIncrementalID = -1
		logger.LoggerAPIPartition.Error("Error while getting next incremental ID | hierarcy : ", hierarchyID)
	}

	return nextIncrementalID
}

// ProcessEventsInDatabase function can process one event or many API events. If the array length is greater than one, it
// would be the startup scenario. Hence all the APIs would be deployed. Otherwise the events type will be taken into consideration
// and will be processed as delete record or insert record based on the event type. The outcome would be another event, which
// represents the partitionID for a given API.
func ProcessEventsInDatabase() {
	// creating the prepared statement for inserting labels
	insertStmt, error := database.CreatePreparedStatement(database.QueryInsertAPI)
	if error != nil {
		logger.LoggerAPIPartition.Error("Error while creating prepared stmt for inserting record.")
		return
	}
	defer insertStmt.Close()
	deleteStmt, error := database.CreatePreparedStatement(database.QueryDeleteAPI)
	if error != nil {
		logger.LoggerAPIPartition.Error("Error while creating prepared stmt for deleting record.")
		insertStmt.Close()
		return
	}
	defer deleteStmt.Close()
	for d := range synchronizer.APIDeployAndRemoveEventChannel {
		updateFromEvents(d, insertStmt, deleteStmt)
	}
}

// updateFromEvents for update the DB for JMS event
func updateFromEvents(apiEventsWithStartupFlag synchronizer.APIEventsWithStartupFlag, insertStmt *sql.Stmt, deleteStmt *sql.Stmt) {
	logger.LoggerAPIPartition.Debug("Started Processing the API Event")
	apiEvents := apiEventsWithStartupFlag.APIEvents
	apiEventCount := len(apiEvents)
	if apiEventCount == 0 {
		logger.LoggerAPIPartition.Debug("Finished processing as the event count is 0")
	} else if apiEventCount == 1 && apiEvents[0].IsRemoveEvent {
		DeleteAPIRecord(&apiEvents[0], deleteStmt)
		logger.LoggerAPIPartition.Debug("Finished processing the API Delete Event")
	} else {
		// When multiple APIs (> 1) are present, it corresponding to the startup scenario. Hence the IsRemoveEvent flag is not
		// considered.
		PopulateAPIData(apiEventsWithStartupFlag, insertStmt)
		logger.LoggerAPIPartition.Debugf("Finished processing API Events, event count : %v", apiEventCount)
	}
}

// DeleteAPIRecord Funtion accept API uuid as the argument
// When receive an Undeploy event, the API record will delete from the database
// If gwLabels are empty , don`t delete the reord (since it is an "API Update event")
func DeleteAPIRecord(api *synchronizer.APIEvent, deleteStmt *sql.Stmt) {
	if len(api.GatewayLabels) > 0 {
		logger.LoggerAPIPartition.Debug("API undeploy event received : ", api.UUID)

		for index := range api.GatewayLabels {
			gatewayLabel := strings.ToLower(api.GatewayLabels[index])

			// when gateway label is "Production and Sandbox" , then gateway label set as "default"
			if gatewayLabel == strings.ToLower(productionSandboxLabel) {
				gatewayLabel = defaultGatewayLabel
			}

			_, error := database.ExecPreparedStatement(database.QueryDeleteAPI, deleteStmt, api.UUID, gatewayLabel)
			if error != nil {
				logger.LoggerAPIPartition.Errorf("Error while deleting the API UUID : %v in gateway: %v , %v", api.UUID, gatewayLabel, error.Error())
				// break
			} else {
				logger.LoggerAPIPartition.Infof("API deleted from the database : %v in gateway : %v", api.UUID, gatewayLabel)
				updateRedisCache(api, gatewayLabel, nil, types.APIDelete)
				pushToXdsCache([]*types.LaAPIEvent{{
					APIUUID:          api.UUID,
					IsRemoveEvent:    true,
					OrganizationUUID: api.OrganizationID,
					LabelHierarchy:   gatewayLabel,
				}})
			}
		}
	} else {
		logger.LoggerAPIPartition.Debug("API update event received : ", api.UUID)
	}
}

// DeleteAPIRecords deletes api records for a certain organization
func DeleteAPIRecords(organizations []msg.Organization) {
	rc := cache.GetClient()
	logger.LoggerAPIPartition.Debugf("APIs undeploy event received for organizations : %v", organizations)

	inClause := prepareInClauseForOrganizationDeletion(organizations)
	sqlQuery := strings.Replace(database.QueryDeleteAPIsForOrganization, "_ORGANIZATIONS_PLACEHOLDER_", inClause, 1)
	_, err := database.ExecDBQuery(sqlQuery)
	if err != nil {
		logger.LoggerAPIPartition.Error("Error while deleting the APIs from database for organizations", err)
	} else {
		logger.LoggerAPIPartition.Info("APIs deleted from the database for organizations")
		for _, organization := range organizations {
			err := cache.RemoveCacheKeysBySubstring(organization.Handle, rc, deleteEvent)
			if err != nil {
				logger.LoggerAPIPartition.Error("Error while deleting the APIs from cache for organization : ", organization.Name, " ", err)
			}
		}
		conf := config.ReadConfigs()
		synchronizer.FetchAllApis(conf, true, false)
	}
}

func prepareInClauseForOrganizationDeletion(organizations []msg.Organization) string {
	str := ""
	str += "("
	for _, organization := range organizations {
		str += "'" + organization.UUID + "'"
		str += ","
	}
	str = str[:len(str)-1] + ")"
	return str
}

// Cache update for undeploy APIs
func updateRedisCache(api *synchronizer.APIEvent, labelHierarchy string, adapterLabel *string, eventType types.EventType) {
	key := getCacheKey(api, labelHierarchy)
	if key != "" {
		logger.LoggerAPIPartition.Debugf("Redis cache updating, cache key : %s", key)
		switch eventType {
		case types.APIDelete:
			rc := cache.GetClient()
			go cache.RemoveCacheKey(key, rc)
			go cache.PublishRedisEvent(key, rc, deleteEvent)
		}
	}

}

func getCacheKey(api *synchronizer.APIEvent, labelHierarchy string) string {
	// apiId : Incremental ID
	// Cache Key pattern : #global-adapter#<environment-label>#<organization-id>#<api-context>
	// api-context should be trimmed out of orgname since organization id is used as the key
	// Cache Value : Partition Label ID ie: dev-p1, prod-p3
	// labelHierarchy : gateway label (dev,prod and etc)

	var cacheKey string

	if api.Context != "" && api.OrganizationID != "" {
		nonOrgContext := api.Context[strings.Index(api.Context[1:], "/")+1:]
		cacheKey = fmt.Sprintf("#%s#%s#%s#%s", clientName, labelHierarchy, api.OrganizationID, nonOrgContext)
	} else {
		logger.LoggerAPIPartition.Error("Unable to get cache key due to empty API Context : ", api.UUID)
	}
	return cacheKey
}

func getCacheValue(api *synchronizer.APIEvent, routerLabel string) string {
	// Cache value pattern : <router-label>/<api-context>

	var cacheValue string

	if api.Context != "" {
		cacheValue = fmt.Sprintf("%s%s", routerLabel, api.Context)
	} else {
		logger.LoggerAPIPartition.Error("Unable to get cache value due to empty API Context : ", api.UUID)
	}
	return cacheValue
}

// Return a label generated against to the gateway label and incremental API ID
func getLaLabel(labelHierarchy string, apiID int, partitionSize int) string {
	var partitionID int = 0
	rem := apiID % partitionSize
	div := apiID / partitionSize

	if rem > 0 {
		partitionID = div + 1
	} else {
		partitionID = div
	}

	return fmt.Sprintf("%s-p%d", labelHierarchy, partitionID)
}

func triggerNewDeploymentIfRequired(incrementalID int, partitionSize int, partitionThreshold float32) {
	remainder := incrementalID % partitionSize
	// If the remainder is 1, it is deployed in the new adapter parition. Hence the deployAdapterTriggered is set to false
	if remainder == 1 {
		deployAdapterTriggered = false
	}
	// deployAdapterTriggered is executed avoid printing multiple log entries if the threshold exceeds.
	if !deployAdapterTriggered && float32(remainder)/float32(partitionSize) > partitionThreshold {
		// Currently, the global adapter prints a log.
		logger.LoggerAPIPartition.Infof("%s percentage of the adapter partition: %d is filled.",
			fmt.Sprintf("%.2f", partitionThreshold*100), (incrementalID/partitionSize)+1)
		deployAdapterTriggered = true
	}
}

// UpdateCacheForQuotaExceededStatus Updates redis cache on billing cycle reset or quota exceeded status
func UpdateCacheForQuotaExceededStatus(apiEvents []synchronizer.APIEvent, cacheValue string, orgUUID string) {
	var cacheObj []string
	laLabels := getAPILALabelsForOrg(orgUUID)
	for _, apiEvent := range apiEvents {
		for index := range apiEvent.GatewayLabels {
			gatewayLabel := apiEvent.GatewayLabels[index]

			// when gateway label is "Production and Sandbox" , then gateway label set as "default"
			if gatewayLabel == productionSandboxLabel {
				gatewayLabel = defaultGatewayLabel
			}

			// It is required to convert the gateway label to lowercase as the partition name is required for deploying k8s
			// services
			apiID, isExists := laLabels[gatewayLabel][apiEvent.UUID]
			if isExists {
				logger.LoggerAPIPartition.Debug("API : ", apiEvent.UUID, " has been already persisted to gateway : ", gatewayLabel)
				label := getLaLabel(gatewayLabel, apiID, partitionSize)

				if label != "" {
					// No need to check if org is blocked. If yes,func will be called with "blocked" for cacheValue
					cacheKey := getCacheKey(&apiEvent, strings.ToLower(gatewayLabel))

					if cacheValue == "" {
						cacheValue = getCacheValue(&apiEvent, label)
					}
					logger.LoggerAPIPartition.Infof("Found cache key:%v, cache value:%v, label:%v, for apiEvent:%v",
						cacheKey, cacheValue, label, apiEvent.UUID)

					// Push each key and value to the string array (Ex: "key1","value1","key2","value2")
					if cacheKey != "" {
						cacheObj = append(cacheObj, cacheKey)
						cacheObj = append(cacheObj, cacheValue)
					}
				} else {
					logger.LoggerAPIPartition.Errorf("Error while fetching the API label UUID : %v ", apiEvent.UUID)
				}
			} else {
				logger.LoggerAPIPartition.Errorf("Couldn't find API for UUID: %s, gatewayLabel: %s , orgUUID: %s in database", apiEvent.UUID, gatewayLabel, orgUUID)
			}
		}
	}
	if len(cacheObj) >= 2 {
		rc := cache.GetClient()
		cachingError := cache.SetCacheKeys(cacheObj, rc)
		if cachingError != nil {
			return
		}
		cache.PublishUpdatedAPIKeys(cacheObj, rc)
	}
}

func isQuotaExceededForOrg(orgID string) bool {
	var isExceeded bool
	if IsStepQuotaLimitingEnabled {
		logger.LoggerMsg.Debugf("'%s' enabled. Hence checking quota exceeded for org: %s",
			featureStepQuotaLimiting, orgID)
		row, err := database.ExecDBQuery(database.QueryIsQuotaExceeded, orgID)
		if err == nil {
			if !row.Next() {
				logger.LoggerMsg.Debugf("Record does not exist for orgId : %s", orgID)
			} else {
				row.Scan(&isExceeded)
				logger.LoggerMsg.Debugf("Step quota limit exceeded : %v for orgId: %s", isExceeded, orgID)
				return isExceeded
			}
		} else {
			logger.LoggerMsg.Errorf("Error when checking whether organisation's quota exceeded or not for orgId : %s. Error: %v", orgID, err)
		}
	}
	return false
}

// getStepQuotaLimitingConfig Check if quota limiting feature is enabled
func getStepQuotaLimitingConfig() bool {
	featureStepQuotaLimitingEnvValue := os.Getenv(featureStepQuotaLimiting)
	if featureStepQuotaLimitingEnvValue != "" {
		enabled, err := strconv.ParseBool(featureStepQuotaLimitingEnvValue)
		if err == nil {
			logger.LoggerMsg.Debugf("'%s' is enabled.", featureStepQuotaLimiting)
			return enabled
		}
		logger.LoggerMsg.Errorf("Error occurred while parsing %s environment value. Error: %v",
			featureStepQuotaLimitingEnvValue, err)
	}
	// Disabled by default
	return false
}
