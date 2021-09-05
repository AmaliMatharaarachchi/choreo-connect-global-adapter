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
	"fmt"
	"strings"

	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/cache"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/config"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/database"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/synchronizer"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/types"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/xds"
)

// RedisBlockedValue Step quota exceeded/reset org's redis value
const RedisBlockedValue = "blocked"

var configs = config.ReadConfigs()
var partitionSize = configs.Server.PartitionSize
var deployAdapterTriggered bool

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
func PopulateAPIData(apis []synchronizer.APIEvent) {
	var laAPIList []*types.LaAPIEvent
	var cacheObj []string

	database.WakeUpConnection()
	defer database.CloseDbConnection()

	for ind := range apis {
		for index := range apis[ind].GatewayLabels {
			gatewayLabel := apis[ind].GatewayLabels[index]

			// when gateway label is "Production and Sandbox" , then gateway label set as "default"
			if gatewayLabel == productionSandboxLabel {
				gatewayLabel = defaultGatewayLabel
			}

			// It is required to convert the gateway label to lowercase as the partition name is required for deploying k8s
			// services
			label := insertRecord(&apis[ind], strings.ToLower(gatewayLabel), types.APICreate)

			if label != "" {
				isExceeded := isQuotaExceededForOrg(apis[ind].OrganizationID)
				cacheKey := getCacheKey(&apis[ind], strings.ToLower(gatewayLabel))
				cacheValue := RedisBlockedValue
				if !isExceeded {
					cacheValue = getCacheValue(&apis[ind], label)
				}

				logger.LoggerAPIPartition.Info("Label for : ", apis[ind].UUID, " and Gateway : ", gatewayLabel, " is ", label)

				apiState := types.LaAPIEvent{
					LabelHierarchy:   gatewayLabel,
					Label:            label,
					RevisionUUID:     apis[ind].RevisionID,
					APIUUID:          apis[ind].UUID,
					OrganizationUUID: apis[ind].OrganizationID,
				}
				laAPIList = append(laAPIList, &apiState)

				// Push each key and value to the string array (Ex: "key1","value1","key2","value2")
				if cacheKey != "" {
					cacheObj = append(cacheObj, cacheKey)
					cacheObj = append(cacheObj, cacheValue)
				}

			} else {
				logger.LoggerAPIPartition.Errorf("Error while fetching the API label UUID : %v ", apis[ind].UUID)
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
		pushToXdsCache(laAPIList)
	}

}

func pushToXdsCache(laAPIList []*types.LaAPIEvent) {
	logger.LoggerAPIPartition.Debug("API List : ", len(laAPIList))
	if len(laAPIList) == 0 {
		return
	}
	if len(laAPIList) > 1 {
		xds.AddMultipleAPIs(laAPIList)
		return
	}
	xds.ProcessSingleEvent(laAPIList[0])
}

// insertRecord always return the adapter label for the relevant API
// If the API is not in database, that will save to the database and return the label
func insertRecord(api *synchronizer.APIEvent, gwLabel string, eventType types.EventType) string {
	var adapterLabel string
	stmt, error := database.DB.Prepare(database.QueryInsertAPI)

	if error != nil {
		logger.LoggerAPIPartition.Errorf("Error while persist the API info for UUID : %v ", &api.UUID)
	} else {
		isExists, apiID := isAPIExists(api.UUID, gwLabel)
		if isExists {
			logger.LoggerAPIPartition.Debug("API : ", api.UUID, " has been already persisted to gateway : ", gwLabel)
			adapterLabel = getLaLabel(gwLabel, *apiID, partitionSize)
		} else {
			for {
				availableID, isNewID := getAvailableID(gwLabel)
				if availableID == -1 { // Return -1 due to an error
					logger.LoggerAPIPartition.Errorf("Error while getting next available ID | hierarchy : %v", gwLabel)
					break
				} else {
					_, err := stmt.Exec(api.UUID, &gwLabel, availableID, api.OrganizationID)
					if err != nil {
						if strings.Contains(err.Error(), "duplicate key") {
							logger.LoggerAPIPartition.Debug(" ID already exists ", err)
							continue
						} else {
							logger.LoggerAPIPartition.Error("Error while writing partition information ", err)
						}
					} else {
						adapterLabel = getLaLabel(gwLabel, availableID, partitionSize)
						logger.LoggerAPIPartition.Debug("New API record persisted UUID : ", api.UUID, " gatewayLabel : ", gwLabel, " partitionId : ", availableID)
						// Only if the incremental ID is a new one (instead of occupying avaliable vacant ID), new deployment trigger should happen.
						if isNewID {
							triggerNewDeploymentIfRequired(availableID, partitionSize, configs.Server.PartitionThreshold)
						}
						break
					}
				}
			}
		}
		stmt.Close()
	}

	return adapterLabel
}

// Return a boolean for API existance , int for incremental ID if the API already exists
func isAPIExists(uuid string, labelHierarchy string) (bool, *int) {

	var apiID int
	row, err := database.DB.Query(database.QueryIsAPIExists, uuid, labelHierarchy)
	if err == nil {
		if !row.Next() {
			logger.LoggerAPIPartition.Debug("Record does not exist for labelHierarchy : ", labelHierarchy, " and uuid : ", uuid)
		} else {
			row.Scan(&apiID)
			logger.LoggerAPIPartition.Debug("API already persisted : ", uuid, " for ", labelHierarchy)
			return true, &apiID
		}
	} else {
		logger.LoggerAPIPartition.Error("Error when checking whether the API is exists. uuid : ", uuid, " ", err)
	}

	return false, nil
}

// Function returns the next available inremental ID. For collect the next available ID , there are 2 helper functions.
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

// Observing emptied incremental ID
func getEmptiedID(hierarchyID string) int {
	var emptiedID int
	stmt, error := database.DB.Query(database.QueryGetEmptiedID, hierarchyID)
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
	stmt, error := database.DB.Query(database.QueryGetNextIncID, hierarchyID)
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
	for d := range synchronizer.APIDeployAndRemoveEventChannel {
		updateFromEvents(d)
	}
}

// for update the DB for JMS event
// TODO : if event is for undeploy or remove task , then API should delete from the DB
func updateFromEvents(apis []synchronizer.APIEvent) {
	// When multiple APIs (> 1) are present, it corresponding to the startup scenario. Hence the IsRemoveEvent flag is not
	// considered.
	if len(apis) > 1 {
		PopulateAPIData(apis)
		return
	}
	if len(apis) == 0 {
		return
	}
	isRemoveEvent := apis[0].IsRemoveEvent
	if isRemoveEvent {
		DeleteAPIRecord(&apis[0])
		return
	}
	PopulateAPIData(apis)
}

// DeleteAPIRecord Funtion accept API uuid as the argument
// When receive an Undeploy event, the API record will delete from the database
// If gwLabels are empty , don`t delete the reord (since it is an "API Update event")
func DeleteAPIRecord(api *synchronizer.APIEvent) bool {
	if len(api.GatewayLabels) > 0 {
		logger.LoggerAPIPartition.Debug("API undeploy event received : ", api.UUID)

		if database.WakeUpConnection() {
			defer database.CloseDbConnection()

			stmt, _ := database.DB.Prepare(database.QueryDeleteAPI)

			for index := range api.GatewayLabels {
				gatewayLabel := api.GatewayLabels[index]

				// when gateway label is "Production and Sandbox" , then gateway label set as "default"
				if gatewayLabel == productionSandboxLabel {
					gatewayLabel = defaultGatewayLabel
				}
				_, error := stmt.Exec(api.UUID, gatewayLabel)
				if error != nil {
					logger.LoggerAPIPartition.Error("Error while deleting the API UUID : ", api.UUID, " ", error)
					// break
				} else {
					logger.LoggerAPIPartition.Info("API deleted from the database : ", api.UUID)
					updateRedisCache(api, strings.ToLower(gatewayLabel), nil, types.APIDelete)
					pushToXdsCache([]*types.LaAPIEvent{{
						APIUUID:          api.UUID,
						IsRemoveEvent:    true,
						OrganizationUUID: api.OrganizationID,
						LabelHierarchy:   strings.ToLower(gatewayLabel),
					}})
					return true
				}
			}
		}
	} else {
		logger.LoggerAPIPartition.Debug("API update event received : ", api.UUID)
	}

	return false
}

// Cache update for undeploy APIs
func updateRedisCache(api *synchronizer.APIEvent, labelHierarchy string, adapterLabel *string, eventType types.EventType) {

	rc := cache.GetClient()
	key := getCacheKey(api, labelHierarchy)
	if key != "" {
		logger.LoggerAPIPartition.Debug("Redis cache updating ")

		switch eventType {
		case types.APIDelete:
			go cache.RemoveCacheKey(key, rc)
			go cache.PublishRedisEvent(key, rc, deleteEvent)
		}
	}

}

func getCacheKey(api *synchronizer.APIEvent, labelHierarchy string) string {
	// apiId : Incremental ID
	// Cache Key pattern : #global-adapter#<environment-label>#<api-context>
	// Cache Value : Partition Label ID ie: dev-p1, prod-p3
	// labelHierarchy : gateway label (dev,prod and etc)

	var cacheKey string

	if api.Context != "" {
		cacheKey = fmt.Sprintf("#%s#%s#%s", clientName, labelHierarchy, api.Context)
	} else {
		logger.LoggerAPIPartition.Error("Unable to get cache key due to empty API Context : ", api.UUID)
	}

	logger.LoggerAPIPartition.Debug(" Generated cache key : ", cacheKey)
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

	logger.LoggerAPIPartition.Debug(" Generated cache Value : ", cacheValue)
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
func UpdateCacheForQuotaExceededStatus(apiEvent synchronizer.APIEvent, cacheValue string) {
	var cacheObj []string
	for index := range apiEvent.GatewayLabels {
		gatewayLabel := apiEvent.GatewayLabels[index]

		// when gateway label is "Production and Sandbox" , then gateway label set as "default"
		if gatewayLabel == productionSandboxLabel {
			gatewayLabel = defaultGatewayLabel
		}

		// It is required to convert the gateway label to lowercase as the partition name is required for deploying k8s
		// services
		isExists, apiID := isAPIExists(apiEvent.UUID, gatewayLabel)
		if isExists {
			logger.LoggerAPIPartition.Debug("API : ", apiEvent.UUID, " has been already persisted to gateway : ", gatewayLabel)
			label := getLaLabel(gatewayLabel, *apiID, partitionSize)

			if label != "" {
				// No need to check if org is blocked
				cacheKey := getCacheKey(&apiEvent, strings.ToLower(gatewayLabel))
				if cacheValue == "" {
					cacheValue = getCacheValue(&apiEvent, label)
				}
				logger.LoggerAPIPartition.Info("Label for : ", apiEvent.UUID, " and Gateway : ", gatewayLabel, " is ", label)

				// Push each key and value to the string array (Ex: "key1","value1","key2","value2")
				if cacheKey != "" {
					logger.LoggerAPIPartition.Debugf("Caching %v -> %v", cacheKey, cacheObj)
					cacheObj = append(cacheObj, cacheKey)
					cacheObj = append(cacheObj, cacheValue)
				}
			} else {
				logger.LoggerAPIPartition.Errorf("Error while fetching the API label UUID : %v ", apiEvent.UUID)
			}
		} else {
			logger.LoggerAPIPartition.Warnf("Couldn't find API for UUID: %s, gatewayLabel: %s", apiEvent.UUID, gatewayLabel)
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
	row, err := database.DB.Query(database.QueryIsQuotaExceeded, orgID)
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
	return false
}
