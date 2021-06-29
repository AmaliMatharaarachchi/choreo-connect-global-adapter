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

	logger "github.com/sirupsen/logrus"
)

// TODO : Following properties should fetch from config file
var dbName string = "globalAdapter"
var dbUserName string = "sa"
var dbPassword string = "Test@1234"
var dbPort int = 1433
var dbHost string = "127.0.0.1"

var DB *sql.DB

func ConnectToDb() {
	var err error
	connString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;database=%s", dbHost, dbUserName, dbPassword, dbPort, dbName)
	DB, err = sql.Open("sqlserver", connString)
	if err != nil {
		logger.Fatal("[DB Error] DB connection error - ", err)
		// check the error and retry to connect
	} else {
		logger.Info("Established the DB connection ...")
	}
}

// create table to persist API records
/*
	Table Name : tbl_apis
	Columns :
		api_uuid : UUID of the API. this id return from the CP
		label_hierarchy : Id of the partition hierarchy.String value.
		api_id : Incremental ID. Partitioning will happen base on this value
		org_id : organization which owns the API
	Primary Key : api_uuid,label_hierarchy,api_id
*/

// No need this part
func CreateTable() bool {
	query := `IF NOT EXISTS (SELECT name FROM sys.tables WHERE name='ga_local_adapter_partition') 
				CREATE TABLE ga_local_adapter_partition ( 
					api_uuid VARCHAR(150) NOT NULL, 
					label_hierarchy VARCHAR(50) NOT NULL, 
					api_id INT NOT NULL, 
					org_id VARCHAR(150
				) PRIMARY KEY (api_uuid,label_hierarchy))`

	_, error := DB.Exec(query)
	if error != nil {
		logger.Fatal("[DB Error] Failed to create table ", error)
		return false
	}
	return true
}
