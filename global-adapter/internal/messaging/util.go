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
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/apipartition"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/config"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/database"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/synchronizer"
	"github.com/wso2/product-microgateway/adapter/pkg/adapter"
	sync "github.com/wso2/product-microgateway/adapter/pkg/synchronizer"
)

// Get all the API IDs for an organisation
func getAPIIdsForOrg(orgID string) ([]string, error) {
	database.WakeUpConnection()
	defer database.CloseDbConnection()

	var apiIds []string
	row, err := database.DB.Query(database.QueryGetAPIsByOrg, orgID)
	if err == nil {
		for row.Next() {
			var apiID string
			row.Scan(&apiID)
			apiIds = append(apiIds, apiID)
		}

		logger.LoggerMsg.Debugf("Found %v APIs for org: %v", len(apiIds), orgID)
	} else {
		logger.LoggerMsg.Errorf("Error when fetching APIs for orgId : %s. Error: %v", orgID, err)
		return nil, err
	}
	return apiIds, nil
}

// Multiple listeners needs to insert/update organisation's quota exceeded status
func upsertQuotaExceededStatus(orgID string, status bool) error {
	database.WakeUpConnection()
	defer database.CloseDbConnection()

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

func getAPIEvents(apiID string, conf *config.Config) ([]synchronizer.APIEvent, error) {
	// Populate data from configuration file.
	serviceURL := conf.ControlPlane.ServiceURL
	username := conf.ControlPlane.Username
	password := conf.ControlPlane.Password
	environmentLabels := conf.ControlPlane.EnvironmentLabels
	skipSSL := conf.ControlPlane.SkipSSLVerification
	retryInterval := conf.ControlPlane.RetryInterval
	truststoreLocation := conf.Truststore.Location

	// Create a channel for the byte slice (response from the APIs from control plane).
	c := make(chan sync.SyncAPIResponse)

	// Fetch APIs from control plane and write to the channel c.
	adapter.GetAPIs(c, &apiID, serviceURL, username, password, environmentLabels, skipSSL, truststoreLocation,
		synchronizer.RuntimeMetaDataEndpoint, false, nil)

	// Get deployment.json from the channel c.
	deploymentDescriptor, err := synchronizer.GetArtifactDetailsFromChannel(c, serviceURL,
		username, password, skipSSL, truststoreLocation, retryInterval)

	var apiEvents []synchronizer.APIEvent
	if err != nil {
		logger.LoggerServer.Fatalf("Error occurred while reading API artifacts: %v ", err)
		return apiEvents, err
	}

	for _, deployment := range deploymentDescriptor.Data.Deployments {
		// Create a new APIEvent.
		apiEvent := synchronizer.APIEvent{}

		// File name is in the format `UUID-revisionID`.
		// UUID and revision id contain 24 characters each.
		apiEvent.UUID = deployment.APIFile[:24]
		// Add the revision ID to the api event.
		apiEvent.RevisionID = deployment.APIFile[25:49]
		// Organization ID is required for the API struct sent over XDS to the local adapter
		apiEvent.OrganizationID = deployment.OrganizationID
		// Read the environments.
		environments := deployment.Environments
		for _, env := range environments {
			// Add the environments as GatewayLabels to the api event.
			apiEvent.GatewayLabels = append(apiEvent.GatewayLabels, env.Name)
		}

		// Add context and version of incoming API events to the apiEvent.
		apiEvent.Context = deployment.APIContext
		apiEvent.Version = deployment.Version

		apiEvents = append(apiEvents, apiEvent)
		logger.LoggerMsg.Debugf("Successfully retrieved API Event: %v", apiEvent)
	}
	return apiEvents, nil
}

func updateCacheForAPIIds(apiIds []string, redisValue string, conf *config.Config) {
	var apiEvents []synchronizer.APIEvent
	var failed bool

	// Retrieve API events from APIM per API Id
	for _, apiID := range apiIds {
		eventsForAPIID, err := getAPIEvents(apiID, conf)
		if err != nil || eventsForAPIID == nil {
			logger.LoggerMsg.Errorf("Failed to get API event for apiID: %s. Error: %v", apiID, err)
			failed = true
			break
		}
		for _, event := range eventsForAPIID {
			apiEvents = append(apiEvents, event)
		}
		logger.LoggerMsg.Debugf("Got API Events: %v for apiID: %s", apiEvents, apiID)
	}

	// If retrieving API events from APIM failed for an apiID, return without continuing for other apiIDs
	if failed {
		return
	}

	database.WakeUpConnection()
	defer database.CloseDbConnection()

	for _, apiEvent := range apiEvents {
		logger.LoggerMsg.Debugf("Found API events. Hence updating redis cache for quota exceeded status")
		apipartition.UpdateCacheForQuotaExceededStatus(apiEvent, redisValue)
	}
}
