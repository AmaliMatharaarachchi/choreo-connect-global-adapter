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
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/cache"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/config"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/database"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/types"
)

// TODO : Following 2 lines no need in near future.Since no need to fetch API info from the CP
var ehURL string = "https://localhost:9443/internal/data/v1/apis/"
var basicAuth string = "Basic YWRtaW46YWRtaW4="

var configs = config.ReadConfigs()
var partitionSize = configs.Server.PartitionSize

// CacheAction is use as enum type for Redis cache update event type
type CacheAction int

// Cache update type
const (
	Add CacheAction = iota
	Remove
)

const (
	gwType        string = "type"
	gatewayLabel  string = "gatewayLabel"
	envoy         string = "Envoy"
	authorization string = "Authorization"
	uuid          string = "uuid"
	clientName    string = "global-adapter"
)

var apisChan = make(chan []types.LaAPIState)

// PopulateAPIData - populating API infomation to Database and redis cache
func PopulateAPIData(apis []types.API) {
	var laAPIList []types.LaAPIState
	var cacheObj []string

	database.WakeUpConnection()

	for ind := range apis {
		for index := range apis[ind].GwLabel {
			label := InsertRecord(&apis[ind], apis[ind].GwLabel[index], types.APICreate)
			cacheKey := getCacheKey(&apis[ind], &apis[ind].GwLabel[index])

			logger.LoggerServer.Info("Label for : ", apis[ind].UUID, " and Gateway : ", apis[ind].GwLabel[index], " is ", label)

			apiState := types.LaAPIState{LabelHierarchy: apis[ind].GwLabel[index], Label: *label, Revision: apis[ind].RevisionID, EventType: types.APICreate}
			laAPIList = append(laAPIList, apiState)

			// Push each key and value to the string array (Ex: "key1","value1","key2","value2")
			cacheObj = append(cacheObj, *cacheKey)
			cacheObj = append(cacheObj, *label)
		}
	}

	rc := cache.GetClient()
	cache.SetCacheKeys(cacheObj, rc)

	pushToChan(apisChan, laAPIList)
	defer database.CloseDbConnection()

}

func pushToChan(c chan []types.LaAPIState, laAPIList []types.LaAPIState) {
	logger.LoggerServer.Debug("API List : ", len(laAPIList))
	apisChan <- laAPIList
}

// InsertRecord always return the adapter label for the relevant API
// If the API is not in database, that will save to the database and return the label
func InsertRecord(api *types.API, gwLabel string, eventType types.EvetType) *string {
	var adapterLabel *string
	stmt, _ := database.DB.Prepare(database.QueryInsertAPI)
	isExists, apiID := isAPIExists(api.UUID, gwLabel)
	if isExists {
		logger.LoggerServer.Debug("API : ", api.UUID, " has been already persisted to gateway : ", gwLabel)
		*adapterLabel = getLaLabel(gwLabel, *apiID, partitionSize)
	} else {
		for {
			availableID := getAvailableID(&gwLabel)
			if availableID == -1 { // Return -1 due to an error
				logger.LoggerServer.Errorf("Error while getting next available ID | hierarchy : %v", gwLabel)
				break
			} else {
				_, err := stmt.Exec(api.UUID, &gwLabel, availableID)
				if err != nil {
					if strings.Contains(err.Error(), "duplicate key") {
						logger.LoggerServer.Debug(" ID already exists ", err)
						continue
					} else {
						logger.LoggerServer.Error("Error while writing partition information ", err)
					}
					adapterLabel = nil
				} else {
					*adapterLabel = getLaLabel(gwLabel, availableID, partitionSize)
					logger.LoggerServer.Debug("New API record persisted UUID : ", api.UUID, " gatewayLebl : ", gwLabel, " partitionId : ", availableID)
					break
				}
			}
		}
	}
	stmt.Close()

	return adapterLabel
}

// Return a boolean for API existance , int for incremental ID if the API already exists
func isAPIExists(uuid string, labelHierarchy string) (bool, *int) {

	var apiID int
	row, error := database.DB.Query(database.QueryIsAPIExists, uuid, labelHierarchy)
	if error == nil {
		if !row.Next() {
			logger.LoggerServer.Debug("Record does not exist for labelHierarchy : ", labelHierarchy, " and uuid : ", uuid)
		} else {
			row.Scan(&apiID)
			logger.LoggerServer.Debug("API already persisted : ", uuid, " for ", labelHierarchy)
			return true, &apiID
		}
	} else {
		logger.LoggerServer.Error("Error when checking whether the API is exists. uuid : ", uuid, " ", error)
	}

	return false, nil
}

// Function returns the next available inremental ID. For collect the next available ID , there are 2 helper functions.
// 1. getEmptiedId() -  Return if there any emptied ID. Return smallest first ID.If no emptied IDs available , then returns 0.
// 2. getNextIncrementalId() - If getEmptiedId return 0 , then this function returns next incremental ID.
func getAvailableID(hierarchyID *string) int {

	var nextAvailableID int = getEmptiedID(hierarchyID)

	if nextAvailableID == 0 {
		nextAvailableID = getNextIncrementalID(hierarchyID)
	}

	logger.LoggerServer.Debug("Next available ID for hierarchy ", *hierarchyID, " is ", nextAvailableID)
	return nextAvailableID
}

// Observing emptied incremental ID
func getEmptiedID(hierarchyID *string) int {
	var emptiedID int
	stmt, error := database.DB.Query(database.QueryGetEmptiedID, hierarchyID)
	if error != nil {
		stmt.Next()
		stmt.Scan(&emptiedID)
		logger.LoggerServer.Debug("The next available id from deleted APIs | hierarchy : ", *hierarchyID, " is : ", emptiedID)
	} else {
		logger.LoggerServer.Error("Error while getting next available id from deleted APIs | hierarchy : ", *hierarchyID)
	}

	return emptiedID
}

// Return next ID . If error occur from the query level , then return -1.
func getNextIncrementalID(hierarchyID *string) int {
	var highestID int
	var nextIncrementalID int
	stmt, error := database.DB.Query(database.QueryGetNextIncID, hierarchyID)
	if error == nil {
		stmt.Next()
		stmt.Scan(&highestID)
		nextIncrementalID = highestID + 1
		logger.LoggerServer.Debug("Next incremental ID for hierarchy : ", *hierarchyID, " is : ", nextIncrementalID)
	} else {
		nextIncrementalID = -1
		logger.LoggerServer.Error("Error while getting next incremental ID | hierarcy : ", hierarchyID)
	}

	return nextIncrementalID
}

// for update the DB for JMS event
// TODO : if event is for undeploy or remove task , then API should delete from the DB
func updateFromEvent(api *types.API, eventType types.EvetType) {
	switch eventType {
	case types.APIDelete:
		DeleteAPIRecord(api)
	case types.APICreate:
		PopulateAPIData([]types.API{*api})
	}
}

// DeleteAPIRecord Funtion accept API uuid as the argument
// When receive an Undeploy event, the API record will delete from the database
// If gwLabels are empty , don`t delete the reord (since it is an "API Update event")
func DeleteAPIRecord(api *types.API) bool {
	database.WakeUpConnection()
	if len(api.GwLabel) > 0 {
		logger.LoggerServer.Debug("API undeploy event received : ", api.UUID)

		if database.WakeUpConnection() {
			stmt, _ := database.DB.Prepare(database.QueryDeleteAPI)

			for index := range api.GwLabel {
				_, error := stmt.Exec(api.UUID, api.GwLabel[index])
				if error != nil {
					logger.LoggerServer.Error("Error while deleting the API UUID : ", api.UUID, " ", error)
					// break
				} else {
					logger.LoggerServer.Debug("API deleted from the database : ", api.UUID)
					updateRedisCache(api, &api.GwLabel[index], nil, types.APIDelete)
					return true
				}
			}
		}
	} else {
		logger.LoggerServer.Debug("API update event received : ", api.UUID)
	}

	defer database.CloseDbConnection()

	return false
}

// Cache update for undeploy APIs
func updateRedisCache(api *types.API, labelHierarchy *string, adapterLabel *string, eventType types.EvetType) {

	rc := cache.GetClient()
	key := getCacheKey(api, labelHierarchy)
	logger.LoggerServer.Debug("Redis cache updating ")

	switch eventType {
	case types.APIDelete:
		go cache.RemoveCacheKey(key, rc, 0)
	}
}

func getCacheKey(api *types.API, labelHierarchy *string) *string {
	// apiId : Incremental ID
	// Cache Key pattern : <organization id>_<base path>_<api version>
	// Cache Value : Pertion Label ID ie: devP-1, prodP-3
	// labelHierarchy : gateway label (dev,prod and etc)

	var version string
	var basePath string
	var organization string

	if strings.TrimSpace(api.Context) == "" || strings.TrimSpace(api.Version) == "" {
		api := fetchAPIInfo(&api.UUID, labelHierarchy) // deprecated
		if api != nil {
			version = api.Version
			basePath = api.Context
			organization = api.Organization
		}
	} else {
		version = api.Version
		basePath = "/" + strings.SplitN(api.Context, "/", 3)[2]
		organization = strings.Split(api.Context, "/")[1]
	}

	cacheKey := fmt.Sprintf(clientName+"#%s#%s_%s_%s", *labelHierarchy, organization, basePath, version)
	logger.LoggerServer.Debug(" Generated cache key : ", cacheKey)

	return &cacheKey
}

//	TODO : No need to fetch API info from /apis endpoint, since API version and context will fetch from the inital request in future
func fetchAPIInfo(apiUUID, gwLabel *string) *types.API {

	var apiInfo *types.API

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	req, _ := http.NewRequest("GET", ehURL, nil)
	queryParam := req.URL.Query()
	queryParam.Add(uuid, *apiUUID)
	queryParam.Add(gatewayLabel, *gwLabel)
	req.URL.RawQuery = queryParam.Encode()
	client := &http.Client{
		Transport: tr,
	}
	req.Header.Set(authorization, basicAuth)

	for {
		resp, err := client.Do(req)
		if err == nil {
			responseData, _ := ioutil.ReadAll(resp.Body)
			var response types.Response
			json.Unmarshal(responseData, &response)
			apiInfo = &response.List[0]
			break
		} else {
			logger.LoggerServer.Error("Error when fetching API info from /apis for API : ", uuid, " .Error : ", err)
			continue
		}
	}

	return apiInfo
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

	return fmt.Sprintf("%sP-%d", labelHierarchy, partitionID)
}
