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

// Multiple listeners needs to insert/update organisation's quota exceeded status
func upsertQuotaExceededStatus(orgID string, status bool) error {
	_, err := database.ExecDBQuery(database.QueryUpsertQuotaStatus, orgID, status)
	if err != nil {
		logger.LoggerMsg.Errorf("Error while upserting quota exceeded status into DB for org: %s, status: %v. Error: %v", orgID, status, err)
		return err
	}
	logger.LoggerMsg.Infof("Successfully upserted quota exceeded status into DB for org: %s, status: %v", orgID, status)
	return nil
}

func getAPIEvents(orgUUID string, conf *config.Config) ([]synchronizer.APIEvent, error) {
	// Populate data from configuration file.
	serviceURL := conf.ControlPlane.ServiceURL
	username := conf.ControlPlane.Username
	password := conf.ControlPlane.Password
	environmentLabels := conf.ControlPlane.EnvironmentLabels
	skipSSL := conf.ControlPlane.SkipSSLVerification
	retryInterval := conf.ControlPlane.RetryInterval
	truststoreLocation := conf.Truststore.Location
	requestTimeout := conf.ControlPlane.HTTPClient.RequestTimeOut

	// Create a channel for the byte slice (response from the APIs from control plane).
	c := make(chan sync.SyncAPIResponse)

	// Fetch APIs from control plane and write to the channel c.
	adapter.GetAPIs(c, nil, serviceURL, username, password, environmentLabels, skipSSL, truststoreLocation,
		synchronizer.RuntimeMetaDataEndpoint, false, nil, requestTimeout)

	// Get deployment.json from the channel c.
	deploymentDescriptor, err := synchronizer.GetArtifactDetailsFromChannel(c, serviceURL,
		username, password, skipSSL, truststoreLocation, retryInterval, requestTimeout)

	var apiEvents []synchronizer.APIEvent
	if err != nil {
		logger.LoggerServer.Errorf("Error occurred while reading API artifacts: %v ", err)
		return apiEvents, err
	}

	for _, deployment := range deploymentDescriptor.Data.Deployments {
		if deployment.OrganizationID == orgUUID {
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
	}
	return apiEvents, nil
}

func updateCacheForAPIIds(orgUUID string, redisValue string, conf *config.Config) {
	apiEvents, err := getAPIEvents(orgUUID, conf)
	if err != nil || len(apiEvents) == 0 {
		logger.LoggerMsg.Error("Failed to get API events for step quota event. ", err)
	} else {
		logger.LoggerMsg.Debugf("Updating redis cache for quota exceeded status for %v api events.", len(apiEvents))
		apipartition.UpdateCacheForQuotaExceededStatus(apiEvents, redisValue, orgUUID)
	}
}
