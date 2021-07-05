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

package database

import (
	"database/sql"
	"fmt"

	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"
)

// TODO : Following properties should fetch from config file
var dbName string = "globalAdapter"
var dbUserName string = "sa"
var dbPassword string = ""
var dbPort int = 1433
var dbHost string = "127.0.0.1"
var maxRetryAttempts int = 10

const (
	sqlDriver string = "sqlserver"
)

// DB - export MSSQL client
var DB *sql.DB

// ConnectToDb - connecting with the database server
func ConnectToDb() {
	var err error
	connString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;database=%s", dbHost, dbUserName, dbPassword, dbPort, dbName)
	DB, err = sql.Open(sqlDriver, connString)
	if err != nil {
		logger.LoggerServer.Fatal("DB connection error - ", err)
		//TODO:  check the error and retry to connect
	} else {
		logger.LoggerServer.Debug("Established the DB connection ...")
	}
}

// WakeUpConnection - checking whether the databace connection is still alive , if not alive then reconnect to the DB
func WakeUpConnection() bool {
	pingError := DB.Ping()
	retryAttempts := 0
	var isPing bool = false

	for {
		if pingError != nil {
			logger.LoggerServer.Debug("Error while initiating the database ", pingError, ". Retry attempt(s) :", retryAttempts)

			if retryAttempts >= maxRetryAttempts {
				break
			} else {
				ConnectToDb()
			}

			isPing = false
			retryAttempts++
			continue

		} else {
			isPing = true
			break
		}
	}

	return isPing
}

// IsTableExists return true if find the searched table
func IsTableExists(tableName string) bool {
	res, _ := DB.Query(QueryTableExists, tableName)
	if !res.Next() {
		logger.LoggerServer.Debug("Table not exists : ", tableName)
	} else {
		logger.LoggerServer.Debug("Table exists : ", tableName)
		return true
	}

	return false
}

// CloseDbConnection - closgin the database connection
func CloseDbConnection() {
	DB.Close()
}
