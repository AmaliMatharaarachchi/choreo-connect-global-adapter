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

/*
 * Package "synchronizer" contains artifacts relate to fetching APIs and
 * API related updates from the control plane event-hub.
 * This file contains functions to retrieve APIs and API updates.
 */

package synchronizer

import (
	"encoding/json"
	"time"

	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"
	"github.com/wso2/product-microgateway/adapter/pkg/health"
	msg "github.com/wso2/product-microgateway/adapter/pkg/messaging"
	sync "github.com/wso2/product-microgateway/adapter/pkg/synchronizer"
)

// RuntimeMetaDataEndpoint represents the endpoint in the control plane
const RuntimeMetaDataEndpoint = "internal/data/v1/runtime-metadata"

// APIDeployAndRemoveEventChannel represents the channel for writing API events.
var APIDeployAndRemoveEventChannel chan []APIEvent

func init() {
	APIDeployAndRemoveEventChannel = make(chan []APIEvent)
}

// GetArtifactDetailsFromChannel retrieve the artifact details from the channel c.
func GetArtifactDetailsFromChannel(c chan sync.SyncAPIResponse, serviceURL string, userName string,
	password string, skipSSL bool, truststoreLocation string,
	retryInterval time.Duration) (*sync.DeploymentDescriptor, error) {

	for i := 0; i < 1; i++ {
		// Read the API details from the channel.
		data := <-c
		if data.Resp != nil {
			// For successfull fetches, data.Resp would return a byte slice with API project(s)
			logger.LoggerSync.Debugf("API project received...")
			health.SetControlPlaneRestAPIStatus(true)
			var deployments sync.DeploymentDescriptor
			err := json.Unmarshal([]byte(string(data.Resp)), &deployments)
			if err != nil {
				logger.LoggerSync.Errorf("Error occured while unmarshalling deplyment data. Error: %v", err.Error())
				return &deployments, err
			}
			return &deployments, nil
		} else if data.ErrorCode == 404 {
			// This condition is checked to prevent GA from crashing when Control Plane doesn't have APIs intially
			// With a 404 http error code the response contains a API Manager error code 900910 hence checking for it
			var error CpError
			unErr := json.Unmarshal([]byte(data.Err.Error()), &error)
			if unErr == nil && error.Code == 900910 {
				logger.LoggerSync.Info("No APIs received from control plane starting global adapter with empty object")
			} else {
				logger.LoggerSync.Fatalf("Error occurred when retrieving APIs from control plane: %v", data.Err)
			}
		} else if data.ErrorCode >= 400 && data.ErrorCode < 500 {
			logger.LoggerSync.Fatalf("Error occurred when retrieving APIs from control plane: %v", data.Err)
		} else {
			// Keep the iteration still until data is received from the control plane.
			i--
			logger.LoggerSync.Errorf("Error occurred while fetching data from control plane: %v", data.Err)
			sync.RetryFetchingAPIs(c, serviceURL, userName, password, skipSSL, truststoreLocation, retryInterval,
				data, RuntimeMetaDataEndpoint, false)
		}
	}
	return &sync.DeploymentDescriptor{}, nil
}

// AddAPIEventsToChannel function updates the api event details and add it to the API event array.
func AddAPIEventsToChannel(deploymentDescriptor *sync.DeploymentDescriptor, incomingAPIEvent *msg.APIEvent) {
	// Create an APIEvent array.
	APIEventArray := []APIEvent{}

	// Loop deployments in deployment descriptor file.
	for _, deployment := range deploymentDescriptor.Data.Deployments {
		// Create a new APIEvent.
		apiEvent := APIEvent{}

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
		if incomingAPIEvent != nil {
			apiEvent.Context = incomingAPIEvent.Context
			apiEvent.Version = incomingAPIEvent.Version
		}
		// Add API Event to array.
		APIEventArray = append(APIEventArray, apiEvent)
	}
	logger.LoggerSync.Debugf("Write API Events %v to the APIDeployAndRemoveEventChannel ", APIEventArray)
	APIDeployAndRemoveEventChannel <- APIEventArray
}
