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

package messaging

import (
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/database"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"
)

// Multiple listeners needs to check whether an organisation is blocked or not when taking actions for the events
func isQuotaExceededForOrg(orgID string) bool {
	var isExceeded bool
	row, err := database.DB.Query(database.QueryIsQuotaExceeded, orgID)
	if err == nil {
		if !row.Next() {
			logger.LoggerMsg.Debugf("Record does not exist for orgId : %s", orgID)
		} else {
			row.Scan(&isExceeded)
			logger.LoggerMsg.Debugf("Step quota limit exceeded : %v for orgId: %s", isExceeded, orgID)
			return isExceeded
		}
	} else {
		logger.LoggerMsg.Errorf("Error when checking whether organisation's quota exceeded or not for orgId : %s. Error: %v", orgID, err)
	}
	return isExceeded
}

func getAPIIdsForOrg(orgID string) ([]string, error) {
	var apiIds []string
	row, err := database.DB.Query(database.QueryGetAPIsbyOrg, orgID)
	if err == nil {
		for row.Next() {
			var apiID string
			row.Scan(&apiID)
			apiIds = append(apiIds, apiID)
		}

		logger.LoggerMsg.Debugf("Found %v APIs for org: %v", len(apiIds), orgID)
		logger.LoggerMsg.Debugf("APIs for org: %v are: %v", orgID, apiIds)
	} else {
		logger.LoggerMsg.Errorf("Error when fetching APIs for orgId : %s. Error: %v", orgID, err)
		return nil, err
	}
	return apiIds, nil
}

// Multiple listeners needs to insert/update organisation's quota exceeded status
func upsertQuotaExceededStatus(orgID string, status bool) error {
	stmt, err := database.DB.Prepare(database.QueryUpsertQuotaStatus)
	if err != nil {
		logger.LoggerMsg.Errorf("Error while preparing quota exceeded query for org: %s. Error: %v", orgID, err)
		return err
	}

	_, error := stmt.Exec(orgID, status)
	if error != nil {
		logger.LoggerMsg.Errorf("Error while upserting quota exceeded status into DB for org: %s. Error: %v", orgID, err)
		return error
	}
	logger.LoggerMsg.Infof("Successfully upserted quota exceeded status into DB for org: %s, status: %v", orgID, status)
	return nil
}
