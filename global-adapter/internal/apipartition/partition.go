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

package apipartition

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	_ "github.com/denisenkom/go-mssqldb"
	// logger "github.com/sirupsen/logrus"

	"github.com/wso2-enterprise/choreo-connect-global-adapter/internal/cache"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/internal/database"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/internal/logger"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/internal/types"
)

// TODO : Following properties should fetch from config file
var partitionSize int = 2
var ehUrl string = "https://localhost:9443/internal/data/v1/apis/"
var basicAuth string = "Basic YWRtaW46YWRtaW4="

type CacheAction int

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

func PopulateApiData(api *types.API) {

	apisChan := make(chan []types.ApiState)

	var laApiList []types.ApiState
	for index, _ := range api.GwLabel {
		label := InsertRecord(api, api.GwLabel[index], types.API_CREATE)
		logger.LoggerServer.Debug("Label for : ", api.UUID, " and Gateway : ", api.GwLabel[index], " is ", label)

		apiState := types.ApiState{LabelHierarchy: api.GwLabel[index], Label: label, Revision: api.RevisionId, EventType: types.API_CREATE}
		laApiList = append(laApiList, apiState)
		updateRedisCache(api, &api.GwLabel[index], &label, types.API_CREATE)
	}

	apisChan <- laApiList
}

/*
	always return the adapter label for the relevant API
	If the API is not in database, that will save to the database and return the label
*/
func InsertRecord(api *types.API, gwLabel string, eventType types.EvetType) string {
	var adapterLabel string
	stmt, _ := database.DB.Prepare("INSERT INTO ga_local_adapter_partition(api_uuid, label_hierarchy, api_id) VALUES(@p1, @p2, @p3)")
	isExists, apiId := isApiExists(api.UUID, gwLabel)
	if isExists {
		logger.LoggerServer.Debug("API : ", api.UUID, " has been already persisted to gateway : ", gwLabel)
		adapterLabel = getLaLabel(gwLabel, *apiId)
	} else {
		for {
			availableId := getAvailableId(&gwLabel)
			_, err := stmt.Exec(api.UUID, &gwLabel, availableId)
			if err != nil {
				if strings.Contains(err.Error(), "duplicate key") {
					logger.LoggerServer.Debug("[DB Error] ID already exists ", err)
					continue
				} else {
					logger.LoggerServer.Error("[DB Error] error when API persisting ", err)
				}
			} else {
				adapterLabel = getLaLabel(gwLabel, availableId)
				logger.LoggerServer.Debug("New API record persisted UUID : ", api.UUID, " gatewayLebl : ", gwLabel, " partitionId : ", availableId)
				break
			}
		}
	}

	return adapterLabel
}

/*
	Return
	boolean for API existance
	int for incremental ID if the API already exists
*/
func isApiExists(uuid string, labelHierarchy string) (bool, *int) {
	row, error := database.DB.Query("SELECT api_id FROM ga_local_adapter_partition WHERE api_uuid = @p1 and label_hierarchy = @p2", uuid, labelHierarchy)
	var apiId int
	if error == nil {
		if row.Next() {
			row.Scan(&apiId)
			logger.LoggerServer.Debug("API already persisted : ", uuid, " for ", labelHierarchy)
			return true, &apiId
		} else {
			logger.LoggerServer.Debug("Record not exists for labelHierarchy : ", labelHierarchy, " and uuid : ", uuid)
			return false, nil
		}
	} else {
		logger.LoggerServer.Error("Error when checking whether the API is exists. ", error)
		return false, nil
	}
}

func getAvailableId(hierarchyId *string) int {

	var nextAvailableId int = 1
	emptiedId := getEmptiedId(hierarchyId)

	if emptiedId != 0 {
		nextAvailableId = emptiedId
	} else {
		nextAvailableId = getNextIncrementalId(hierarchyId)
	}

	logger.LoggerServer.Debug("Next available ID for hierarchy ", *hierarchyId, " is ", nextAvailableId)
	return nextAvailableId
}

// Observing emptied incremental ID
func getEmptiedId(hierarchyId *string) int {
	var emptiedId int
	query := `SELECT TOP 1 temp.rowId from (
				SELECT 
				ga_pt.api_id ,
				ROW_NUMBER() OVER(ORDER BY ga_pt.api_id ASC) as rowId
				FROM ga_local_adapter_partition ga_pt
				where ga_pt.label_hierarchy = @p1
			) temp where temp.rowId <> temp.api_id`
	stmt, _ := database.DB.Query(query, hierarchyId)
	stmt.Next()
	stmt.Scan(&emptiedId)
	logger.LoggerServer.Debug("First emptied ID for hierarchy : ", *hierarchyId, " is : ", emptiedId)
	return emptiedId
}

// Return next ID
func getNextIncrementalId(hierarchyId *string) int {
	var highestId int
	query := `SELECT TOP 1 api_id from ga_local_adapter_partition
	WHERE label_hierarchy = @p1 
	ORDER BY api_id DESC`

	stmt, _ := database.DB.Query(query, hierarchyId)
	stmt.Next()
	stmt.Scan(&highestId)
	nextIncrementalId := highestId + 1
	logger.LoggerServer.Debug("Next incremental ID for hierarchy : ", *hierarchyId, " is : ", nextIncrementalId)
	return nextIncrementalId
}

// for update the DB for JMS event
// TODO : if event is for undeploy or remove task , then API should delete from the DB
func updateFromEvent(api *types.API, eventType types.EvetType) {
	switch eventType {
	case types.API_DELETE:
		DeleteApiRecord(api, "")
	case types.API_CREATE:
		PopulateApiData(api)
	}
}

// When receive an Undeploy event, the API record will delete from the database
// Funtion accept API uuid as the argument
// If gwLabels are empty , don`t delete the reord (since it is an "API Update event")
func DeleteApiRecord(api *types.API, gatewayLabel string) bool {
	if len(api.GwLabel) > 0 {
		logger.LoggerServer.Debug("API delete/undeploy event received : ", api.UUID)
		database.DB.Ping()
		stmt, _ := database.DB.Prepare("DELETE FROM ga_local_adapter_partition WHERE api_uuid = @p1")
		_, error := stmt.Exec(api.UUID)
		if error != nil {
			logger.LoggerServer.Error("[DB Error] error while deleting the API UUID : ", api.UUID, " ", error)
			return false
		} else {
			logger.LoggerServer.Debug("API deleted from the database : ", api.UUID)
			updateRedisCache(api, nil, nil, types.API_DELETE)
			return true
		}
	} else {
		logger.LoggerServer.Debug("API update event received : ", api.UUID)
	}
	return false
}

// TODO - Cache batch update
func updateRedisCache(api *types.API, labelHierarchy *string, adaptaerLabel *string, eventType types.EvetType) {

	rc := cache.GetClient()
	key := getCacheKey(api, labelHierarchy)
	value := adaptaerLabel
	logger.LoggerServer.Debug("Redis cache updating ")

	switch eventType {
	case types.API_CREATE:
		go cache.SetCacheKey(key, value, rc, 0)
	case types.API_DELETE:
		go cache.RemoveCacheKey(key, rc, 0)
	}
}

// TODO : logs and retries
func getCacheKey(api *types.API, labelHierarchy *string) *string {
	/*
		apiId : Incremental ID
		Cache Key pattern : <organization id>_<base path>_<api version>
		Cache Value : Pertion Label ID ie: devP-1, prodP-3
		labelHierarchy : gateway label (dev,prod and etc)
	*/

	var version string
	var basePath string
	var organization string

	if strings.TrimSpace(api.Context) == "" || strings.TrimSpace(api.Version) == "" {
		api := fetchApiInfo(&api.UUID, labelHierarchy)
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

	return &cacheKey
}

/*	TODO :
	No need to fetch API info from /apis endpoint, since API version and context will
	receive from the inital request
*/
func fetchApiInfo(apiUuid, gwLabel *string) *types.API {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	req, _ := http.NewRequest("GET", ehUrl, nil)
	queryParam := req.URL.Query()
	queryParam.Add(uuid, *apiUuid)
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
			return &response.List[0] // TODO
		} else {
			logger.LoggerServer.Error("Error when fetching API info from /apis for API : ", uuid, " .Error : ", err)
			continue
		}
	}
}

// Return a label generated against to the gateway label and incremental API ID
func getLaLabel(labelHierarchy string, apiId int) string {
	var partitionId int = 0
	rem := apiId % partitionSize
	div := apiId / partitionSize

	if rem > 0 {
		partitionId = div + 1
	} else {
		partitionId = div
	}

	return fmt.Sprintf("%sP-%d", labelHierarchy, partitionId)
}
