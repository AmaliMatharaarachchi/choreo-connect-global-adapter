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
 *
 */

package health

import (
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/config"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"
)

var (
	databaseConnectionStatusChan  = make(chan bool)
	databaseConnectionEstablished = false
)

// SetDatabaseConnectionStatus sets the given status to the internal channel databaseConnectionStatusChan
func SetDatabaseConnectionStatus(status bool) {
	// Check for Database Connection Established, to non block call
	// if called again (somehow) after startup, for extra safe check this value
	if !databaseConnectionEstablished {
		databaseConnectionStatusChan <- status
	}
}

// WaitForDatabaseConnection waits until connected to database
func WaitForDatabaseConnection() {
	conf := config.ReadConfigs()
	dbConnected := false
	for !dbConnected {
		dbConnected = <-databaseConnectionStatusChan
		logger.LoggerHealth.Debugf("Connection status to the database %v at %v:%v returned: %v",
			conf.DataBase.Name, conf.DataBase.Host, conf.DataBase.Port, dbConnected)
	}
	databaseConnectionEstablished = true
	logger.LoggerHealth.Info("Successfully connected to the Database.")
}
