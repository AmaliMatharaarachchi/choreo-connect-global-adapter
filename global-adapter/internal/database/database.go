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
	"time"

	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/config"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/health"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"
)

const (
	sqlDriver string = "sqlserver"
)

// DB - export MSSQL client
var DB *sql.DB

// ConnectToDb - connecting with the database server
func ConnectToDb() {
	var err error
	conf := config.ReadConfigs()
	connString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;database=%s", conf.DataBase.Host, conf.DataBase.Username, conf.DataBase.Password, conf.DataBase.Port, conf.DataBase.Name)
	DB, err = sql.Open(sqlDriver, connString)
	if err != nil {
		logger.LoggerServer.Debugf("DB connection error: %v", err)
	}
}

// WakeUpConnection - checking whether the databace connection is still alive , if not alive then reconnect to the DB
func WakeUpConnection() (isAlive bool) {
	conf := config.ReadConfigs()
	maxAttempts := conf.DataBase.OptionalMetadata.MaxRetryAttempts
	var (
		retryInterval time.Duration = 5
		attempt       int
	)
	isAlive = IsAliveConnection()
	if !isAlive {
		for attempt = 1; attempt <= maxAttempts; attempt++ {
			logger.LoggerServer.Infof("Error while connecting to the database. Retrying to connect to database. "+
				"Attempt %d", attempt)
			ConnectToDb()
			isAlive = IsAliveConnection()
			if isAlive {
				break
			}
			time.Sleep(retryInterval * time.Second)
		}
		logger.LoggerServer.Infof("DB connection liveness is %v", isAlive)
	}
	health.SetDatabaseConnectionStatus(isAlive)
	return isAlive
}

// IsAliveConnection check if the db connection is alive
func IsAliveConnection() (isAlive bool) {
	defer func() {
		// to handle the panic error while db ping. panic might be related to https://github.com/denisenkom/go-mssqldb/issues/536
		recover()
	}()
	pingError := DB.Ping()
	if pingError == nil {
		return true
	}
	logger.LoggerServer.Error("Error while connecting to the database", pingError)
	return isAlive
}

// ExecDBQuery Execute db queries after checking/waking up the db connection
func ExecDBQuery(query string, args ...interface{}) (*sql.Rows, error) {
	row, err := DB.Query(query, args...)
	if err != nil && !IsAliveConnection() {
		logger.LoggerServer.Error("Error while executing DB query", err)
		for {
			if WakeUpConnection() {
				row, err = DB.Query(query, args...)
				if err != nil && !IsAliveConnection() {
					// seems like the db connection has dropped. hence retry executing
					continue
				}
				break
			}
		}
	}
	return row, err
}

// CreatePreparedStatement create prepared statement after checking/waking up the db connection
func CreatePreparedStatement(statement string) (*sql.Stmt, error) {
	stmt, err := DB.Prepare(statement)
	if err != nil && !IsAliveConnection() {
		logger.LoggerServer.Error("Error while creating prepared statement", err)
		for {
			if WakeUpConnection() {
				stmt, err = DB.Prepare(statement)
				if err != nil && !IsAliveConnection() {
					// seems like the db connection has dropped. hence retry executing
					continue
				}
				break
			}
		}
	}
	return stmt, err
}

// ExecPreparedStatement Execute prepared statement after checking/waking up the db connection
func ExecPreparedStatement(stmt *sql.Stmt, args ...interface{}) (sql.Result, error) {
	result, err := stmt.Exec(args...)
	if err != nil && !IsAliveConnection() {
		logger.LoggerServer.Error("Error while executing prepared statement", err)
		for {
			if WakeUpConnection() {
				result, err = stmt.Exec(args...)
				if err != nil && !IsAliveConnection() {
					// seems like the db connection has dropped. hence retry executing
					continue
				}
				break
			}
		}
	}
	return result, err
}

// IsTableExists return true if find the searched table
func IsTableExists(tableName string) bool {
	res, _ := ExecDBQuery(QueryTableExists, tableName)
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
