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

package startup

import (
	"github.com/gorilla/mux"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/config"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/database"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/handler"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/health"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"
	"net/http"
)

// TODO : this should fetch from config file
var partitionSize = 10

const (
	apisTable          string = "ga_local_adapter_partition"
	partitionSizeTable string = "la_partition_size"
)

// Initialize for initialize all startup functions
func Initialize() {
	database.ConnectToDb()
	defer database.CloseDbConnection()
	isDbConnectionAlive := database.WakeUpConnection()
	//should add db connection health check here
	if isDbConnectionAlive {
		health.DBConnection.SetStatus(true)
		if database.IsTableExists(partitionSizeTable) {
			triggerDeploymentAgent()
		} else {
			logger.LoggerServer.Fatal("Table not exists : ", partitionSizeTable)
		}

		if !database.IsTableExists(apisTable) {
			logger.LoggerServer.Fatal("Table not exists : ", apisTable)
		} else {
			return
		}
	} else {
		health.DBConnection.SetStatus(false)
		logger.LoggerServer.Fatal("Error while initiating the database")
	}
}

// InitializeAPIServer for initialize GA API Server
func InitializeAPIServer(conf *config.Config) *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/internal/data/v1/apis/deployed-revisions", handler.BasicAuth(handler.HTTPatchHandler)).Methods(http.MethodGet)
	router.HandleFunc("/internal/data/v1/apis/undeployed-revision", handler.BasicAuth(handler.HTTPPostHandler)).Methods(http.MethodGet)
	router.HandleFunc("/internal/data/v1/runtime-metadata", handler.BasicAuth(handler.HTTPGetHandler)).Methods(http.MethodGet)
	router.HandleFunc("/internal/data/v1/runtime-artifacts", handler.BasicAuth(handler.HTTPGetHandler)).Methods(http.MethodGet)
	router.HandleFunc("/internal/data/v1/retrieve-api-artifacts", handler.BasicAuth(handler.HTTPPostHandler)).Methods(http.MethodPost)
	router.HandleFunc("/internal/data/v1/keymanagers", handler.BasicAuth(handler.HTTPGetHandler)).Methods(http.MethodGet)
	router.HandleFunc("/internal/data/v1/revokedjwt", handler.BasicAuth(handler.HTTPGetHandler)).Methods(http.MethodGet)
	router.HandleFunc("/internal/data/v1/keyTemplates", handler.BasicAuth(handler.HTTPGetHandler)).Methods(http.MethodGet)
	router.HandleFunc("/internal/data/v1/block", handler.BasicAuth(handler.HTTPGetHandler)).Methods(http.MethodGet)
	router.HandleFunc("/internal/data/v1/subscriptions", handler.BasicAuth(handler.HTTPGetHandler)).Methods(http.MethodGet)
	router.HandleFunc("/internal/data/v1/applications", handler.BasicAuth(handler.HTTPGetHandler)).Methods(http.MethodGet)
	router.HandleFunc("/internal/data/v1/application-key-mappings", handler.BasicAuth(handler.HTTPGetHandler)).Methods(http.MethodGet)
	router.HandleFunc("/internal/data/v1/application-policies", handler.BasicAuth(handler.HTTPGetHandler)).Methods(http.MethodGet)
	router.HandleFunc("/internal/data/v1/subscription-policies", handler.BasicAuth(handler.HTTPGetHandler)).Methods(http.MethodGet)
	http.Handle("/", router)

	return router
}

func triggerDeploymentAgent() {
	var previousPartitionSize int
	result, error := database.DB.Query(database.QueryGetPartitionSize)
	if error != nil {
		logger.LoggerServer.Error("[DB Error] Error when fetching partition size")
	} else {
		if result.Next() {
			result.Scan(&previousPartitionSize)
			if partitionSize != previousPartitionSize {
				logger.LoggerServer.Debug("Trigger a LA respawn event")
			} else {
				logger.LoggerServer.Debug("No config changes related to the partition size ")
			}
		}
	}
}
