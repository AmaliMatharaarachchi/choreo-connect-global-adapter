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

// Package messaging holds the implementation for event listeners functions
package messaging

import (
	"encoding/json"

	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/apipartition"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/config"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/synchronizer"
	"github.com/wso2/product-microgateway/adapter/pkg/adapter"

	msg "github.com/wso2/product-microgateway/adapter/pkg/messaging"
	sync "github.com/wso2/product-microgateway/adapter/pkg/synchronizer"
)

func handleAzurebillingCycleResetEvents(conf *config.Config) {
	for d := range msg.AzureStepQuotaResetChannel {
		logger.LoggerMsg.Info("Message received for AzureStepQuotaResetChannel = " + string(d))
		var resetEvent BillingCycleResetEvent
		error := parseBillingCycleResetJSONEvent(d, &resetEvent)
		if error != nil {
			logger.LoggerMsg.Errorf("Error while processing the billing cycle reset event %v. Hence dropping the event", error)
			continue
		}
		logger.LoggerMsg.Infof("Billing cycle reset event for org ID: %s is received", resetEvent.orgUuid)
		upsertQuotaExceededStatus(resetEvent.orgUuid, false)

		// API IDs for org
		apiIds, err := getApiIdsForOrg(resetEvent.orgUuid)
		if err != nil {
			logger.LoggerMsg.Errorf("Failed to get API IDs for org: %s. Error: %v", resetEvent.orgUuid, error)
			continue
		}

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
		adapter.GetAPIs(c, nil, serviceURL, username, password, environmentLabels, skipSSL, truststoreLocation,
			synchronizer.RuntimeMetaDataEndpoint, false, apiIds)

		// Get deployment.json from the channel c.
		deploymentDescriptor, err := synchronizer.GetArtifactDetailsFromChannel(c, serviceURL,
			username, password, skipSSL, truststoreLocation, retryInterval)

		if err != nil {
			logger.LoggerServer.Fatalf("Error occurred while reading API artifacts: %v ", err)
			continue
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

			apipartition.UpdateCacheForQuotaExceededStatus(apiEvent, "")
		}
	}
}

func parseBillingCycleResetJSONEvent(data []byte, billingCycleResetEvent *BillingCycleResetEvent) error {
	unmarshalErr := json.Unmarshal(data, &billingCycleResetEvent)
	if unmarshalErr != nil {
		logger.LoggerMsg.Errorf("Error occurred while unmarshalling billing cycle reset event data %v", unmarshalErr)
	}
	return unmarshalErr
}
