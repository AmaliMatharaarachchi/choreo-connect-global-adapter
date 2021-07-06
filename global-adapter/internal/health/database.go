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
	"database/sql"
	"time"
)

var (
	databaseConnectionStatusChan  = make(chan bool)
	databaseConnectionEstablished = false
)

// SetDatabaseConnectionStatus sets the given status to the internal channel databaseConnectionStatusChan
func SetDatabaseConnectionStatus(status bool) {
	// check for Database Connection Established, to non block call
	// if called again (somehow) after startup, for extra safe check this value
	if !databaseConnectionEstablished {
		databaseConnectionStatusChan <- status
	}
}

// WaitForDatabaseConnection sleep the current go routine until connected to database
func WaitForDatabaseConnection() {
	dbConnected := false
	if !dbConnected {
		dbConnected = <-databaseConnectionStatusChan
	}
	databaseConnectionEstablished = true
}

// DatabaseConnectRetry retries to connect to the database if there is a connection error
func DatabaseConnectRetry(connString string) (*sql.DB, error) {
	// TODO: (Jayanie) maxAttempt and retryInterval Should be configurable?
	var (
		maxAttempt    int = 5
		retryInterval time.Duration
		attempt       int
		DB            *sql.DB
		err           error
	)
	for attempt = 1; attempt <= maxAttempt; attempt++ {
		DB, err = sql.Open("sqlserver", connString)
		if err == nil {
			return DB, nil
		} 
		pingError := DB.Ping()
		if pingError == nil {
			return DB, nil
		}
		time.Sleep(retryInterval * time.Second)
	}
	return nil, err
}
