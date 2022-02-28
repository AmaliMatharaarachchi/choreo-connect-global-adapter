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
func WakeUpConnection() (isPing bool) {
	conf := config.ReadConfigs()
	isPing = false
	maxAttempts := conf.DataBase.OptionalMetadata.MaxRetryAttempts
	var (
		retryInterval time.Duration = 5
		attempt       int
	)
	defer func() {
		// to handle the panic error while db ping in defer func
		if recover() != nil {
			health.SetDatabaseConnectionStatus(false)
		}
	}()
	defer func() {
		// to handle the panic error while db ping. panic might be related to https://github.com/denisenkom/go-mssqldb/issues/536
		if recover() != nil {
			logger.LoggerServer.Errorf("Panic while DB ping.")
			ConnectToDb()
			isPing = DB.Ping() == nil
			health.SetDatabaseConnectionStatus(isPing)
		}
	}()
	pingError := DB.Ping()
	if pingError == nil {
		isPing = true
	} else {
		for attempt = 1; attempt <= maxAttempts; attempt++ {
			logger.LoggerServer.Infof("Error while initiating the database %v. Retrying to connect to database. "+
				"Attempt %d", pingError, attempt)
			ConnectToDb()
			pingError = DB.Ping()
			if pingError == nil {
				isPing = true
				break
			} else {
				time.Sleep(retryInterval * time.Second)
			}
		}
	}
	health.SetDatabaseConnectionStatus(isPing)
	return isPing
}

// IsAliveConnection check if the db connection is alive
func IsAliveConnection() (islive bool) {
	defer func() {
		// to handle the panic error while db ping. panic might be related to https://github.com/denisenkom/go-mssqldb/issues/536
		recover()
	}()
	islive = DB.Ping() == nil
	return islive
}

// ExecDBQuery Execute db queries after checking/waking up the db connection
func ExecDBQuery(query string, args ...interface{}) (*sql.Rows, error) {
	var err error
	var row *sql.Rows
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
	return row, err
}

// ExecPreparedStatement Execute prepared statement after checking/waking up the db connection
func ExecPreparedStatement(statement string, args ...interface{}) (sql.Result, error) {
	var result sql.Result
	var err error
	var stmt *sql.Stmt
	for {
		if WakeUpConnection() {
			stmt, err = DB.Prepare(statement)
			if err != nil && !IsAliveConnection() {
				// seems like the db connection has dropped. hence retry executing
				continue
			}
			result, err = stmt.Exec(args...)
			if err != nil && !IsAliveConnection() {
				// seems like the db connection has dropped. hence retry executing
				continue
			}
			if err == nil {
				stmt.Close()
			}
			break
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
