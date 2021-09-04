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

// Declarations all database query
const (
	QueryTableExists  string = "SELECT name FROM sys.tables WHERE name=@p1"
	QueryInsertAPI    string = "INSERT INTO ga_local_adapter_partition(api_uuid, label_hierarchy, api_id, org_id) VALUES(@p1, @p2, @p3, @p4)"
	QueryIsAPIExists  string = "SELECT api_id FROM ga_local_adapter_partition WHERE api_uuid = @p1 and label_hierarchy = @p2"
	QueryDeleteAPI    string = "DELETE FROM ga_local_adapter_partition WHERE api_uuid = @p1 and label_hierarchy = @p2"
	QueryGetNextIncID string = "SELECT TOP 1 api_id from ga_local_adapter_partition WHERE label_hierarchy = @p1 ORDER BY api_id DESC"
	QueryGetEmptiedID string = `SELECT TOP 1 temp.rowId from (
										SELECT 
										ga_pt.api_id ,
										ROW_NUMBER() OVER(ORDER BY ga_pt.api_id ASC) as rowId
										FROM ga_local_adapter_partition ga_pt
										where ga_pt.label_hierarchy = @p1
									) temp where temp.rowId <> temp.api_id`
	QueryGetPartitionSize          string = "SELECT parition_size FROM la_partition_size"
	QueryDeleteAPIsForOrganization string = "DELETE FROM ga_local_adapter_partition WHERE org_id = @p1"
)
