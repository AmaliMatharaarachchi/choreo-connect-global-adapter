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
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/database"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/health"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"
)

// TODO : this should fetch from config file
var partitionSize = 10

const (
	apisTable          string = "ga_local_adapter_partition"
	partitionSizeTable string = "la_partition_size"
)

// Initialize for initialize all startup functions
func Initialize() {
	// Checks database connection health and Waits for database connection
	go health.WaitForDatabaseConnection()
	// Checks redis cache connection health and Waits for redis cache connection
	go health.WaitForRedisCacheConnection()
	database.ConnectToDb()
	defer database.CloseDbConnection()
	isDbConnectionAlive := database.WakeUpConnection()

	if isDbConnectionAlive {
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
