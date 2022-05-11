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
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/config"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"
)

const (
	sqlDriver     string = "sqlserver"
	dbIsClosed    string = "is closed"
	dbIsTimeout   string = "timed out"
	dbIsFailedRPC string = "failed to send RPC"
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
		return
	}
	DB.SetMaxOpenConns(conf.DataBase.PoolOptions.MaxActive)
	DB.SetMaxIdleConns(conf.DataBase.PoolOptions.MaxIdle)
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
	logger.LoggerServer.Error("Error while connecting to the database ", pingError)
	return isAlive
}

// ExecDBQuery Execute db queries after checking/waking up the db connection
func ExecDBQuery(query string, args ...interface{}) (*sql.Rows, context.CancelFunc, error) {
	row, cancel, err := execDBQueryWithcontext(query, args...)
	if err != nil && (strings.Contains(err.Error(), dbIsClosed) || strings.Contains(err.Error(), dbIsTimeout) ||
		strings.Contains(err.Error(), dbIsFailedRPC) || !IsAliveConnection()) {
		logger.LoggerServer.Infof("Error while executing DB query hence retrying ... : %v", err.Error())
		for {
			if WakeUpConnection() {
				cancel()
				row, cancel, err = execDBQueryWithcontext(query, args...)
				if err != nil && (strings.Contains(err.Error(), dbIsClosed) || strings.Contains(err.Error(), dbIsFailedRPC) ||
					strings.Contains(err.Error(), dbIsTimeout) || !IsAliveConnection()) {
					// seems like the db connection has dropped. hence retry executing
					logger.LoggerServer.Debugf("Error while executing db query again hence retrying ... : %v", err.Error())
					continue
				}
				logger.LoggerServer.Info("Executing DB query has succeeded.")
				break
			}
		}
	}
	return row, cancel, err
}

func execDBQueryWithcontext(query string, args ...interface{}) (*sql.Rows, context.CancelFunc, error) {
	timeout := time.Duration(config.ReadConfigs().DataBase.OptionalMetadata.QueryTimeout)
	cont, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	rows, err := DB.QueryContext(cont, query, args...)
	return rows, cancel, err
}

// CreatePreparedStatement create prepared statement after checking/waking up the db connection
func CreatePreparedStatement(statement string) (*sql.Stmt, error) {
	stmt, err := createPreparedStatementWithContext(statement)
	if err != nil && (strings.Contains(err.Error(), dbIsClosed) || strings.Contains(err.Error(), dbIsFailedRPC) ||
		strings.Contains(err.Error(), dbIsTimeout) || !IsAliveConnection()) {
		logger.LoggerServer.Infof("Error while creating DB prepared statement hence retrying ... : %v", err.Error())
		for {
			if WakeUpConnection() {
				stmt, err = createPreparedStatementWithContext(statement)
				if err != nil && (strings.Contains(err.Error(), dbIsClosed) || strings.Contains(err.Error(), dbIsFailedRPC) ||
					strings.Contains(err.Error(), dbIsTimeout) || !IsAliveConnection()) {
					// seems like the db connection has dropped. hence retry executing
					logger.LoggerServer.Debugf("Error while creating prepared statement again hence retrying ... : %v", err.Error())
					continue
				}
				logger.LoggerServer.Info("Creating DB prepared statement has succeeded.")
				break
			}
		}
	}
	return stmt, err
}

func createPreparedStatementWithContext(statement string) (*sql.Stmt, error) {
	timeout := time.Duration(config.ReadConfigs().DataBase.OptionalMetadata.QueryTimeout)
	cont, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	defer cancel()
	return DB.PrepareContext(cont, statement)
}

// ExecPreparedStatement Execute prepared statement after checking/waking up the db connection
func ExecPreparedStatement(stmtString string, stmt *sql.Stmt, args ...interface{}) (sql.Result, error) {
	result, err := execPreparedStatementWithContext(stmt, args...)
	if err != nil && (strings.Contains(err.Error(), dbIsClosed) || strings.Contains(err.Error(), dbIsFailedRPC) ||
		strings.Contains(err.Error(), dbIsTimeout) || !IsAliveConnection()) {
		logger.LoggerServer.Infof("Error while executing prepared statement hence retrying ... : %v", err.Error())
		for {
			if WakeUpConnection() {
				result, err = execPreparedStatementWithContext(stmt, args...)
				if err != nil && strings.Contains(err.Error(), dbIsClosed) {
					logger.LoggerServer.Debugf("Closing and creating the prepared statement again ... : %v", stmtString)
					stmt.Close()
					stmt, err = CreatePreparedStatement(stmtString)
					if err != nil {
						logger.LoggerServer.Debugf("Error while creating the prepared statement again ... : %v ", err.Error())
						continue
					}
					logger.LoggerServer.Debugf("Created the prepared statement again ... : %v ", stmtString)
					result, err = execPreparedStatementWithContext(stmt, args...)
				}
				if err != nil && (strings.Contains(err.Error(), dbIsClosed) || strings.Contains(err.Error(), dbIsFailedRPC) ||
					strings.Contains(err.Error(), dbIsTimeout) || !IsAliveConnection()) {
					// seems like the db connection has dropped. hence retry executing
					logger.LoggerServer.Debugf("Error while executing prepared statement again hence retrying ... : %v", err.Error())
					continue
				}
				logger.LoggerServer.Info("Executing DB prepared statement has succeeded.")
				break
			}
		}
	}
	return result, err
}

func execPreparedStatementWithContext(stmt *sql.Stmt, args ...interface{}) (sql.Result, error) {
	timeout := time.Duration(config.ReadConfigs().DataBase.OptionalMetadata.QueryTimeout)
	cont, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	defer cancel()
	return stmt.ExecContext(cont, args...)
}

// IsTableExists return true if find the searched table
func IsTableExists(tableName string) bool {
	res, cancel, _ := ExecDBQuery(QueryTableExists, tableName)
	defer cancel()
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
