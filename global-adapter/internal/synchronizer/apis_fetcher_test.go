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

package synchronizer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	sync "github.com/wso2/product-microgateway/adapter/pkg/synchronizer"
)

// TODO: (Jayanie) Uncomment this test case after changing the common pkg ControlPlaneStarted variable
// func TestGetArtifactDetailsFromChannel(t *testing.T) {
// 	serviceURL := "https://apim:9443/"
// 	username := "admin"
// 	password := "admin"
// 	skipSSL := true
// 	truststoreLocation := "/home/wso2/security/truststore"
// 	retryInterval := time.Duration(5)
//  health.ControlPlaneStarted = true
// 	data := sync.DeploymentDescriptor{
// 		Type:    "deployments",
// 		Version: "v4.0.0",
// 		Data: sync.DeploymentData{
// 			Deployments: []sync.APIDeployment{
// 				{
// 					APIFile: "60d2cda9e9cab41a3214fa91-60d2cdade9cab41a3214fa92",
// 					Environments: []sync.GatewayLabel{
// 						{
// 							Name:  "Default",
// 							Vhost: "localhost",
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}

// 	jsonData, _ := json.Marshal(data)

// 	response := sync.SyncAPIResponse{Resp: jsonData, Err: nil, ErrorCode: 400, APIUUID: "123", GatewayLabels: nil}

// 	c := make(chan sync.SyncAPIResponse)

// 	go writeDataToChannel(c, response)

// 	deploymentDescriptor, _ := GetArtifactDetailsFromChannel(c, serviceURL,
// 		username, password, skipSSL, truststoreLocation, retryInterval)

// 	for _, deployment := range deploymentDescriptor.Data.Deployments {
// 		assert.Equal(t, deployment.APIFile, "60d2cda9e9cab41a3214fa91-60d2cdade9cab41a3214fa92",
// 			"APIFile name should be the same")
// 		for _, environment := range deployment.Environments {
// 			assert.Equal(t, environment.Name, "Default", "Environment name should be the same")
// 			assert.Equal(t, environment.Vhost, "localhost", "Vhost of the environment should be the same")
// 		}
// 	}

// }

// // Helper method for TestGetArtifactDetailsFromChannel.
// func writeDataToChannel(c chan sync.SyncAPIResponse, response sync.SyncAPIResponse) {
// 	c <- response
// }

func TestAddAPIEventsToChannel(t *testing.T) {
	deploymentDescriptor := sync.DeploymentDescriptor{
		Type:    "deployments",
		Version: "v4.0.0",
		Data: sync.DeploymentData{
			Deployments: []sync.APIDeployment{
				{
					APIFile: "60d2cda9e9cab41a3214fa91-60d2cdade9cab41a3214fa92",
					Environments: []sync.GatewayLabel{
						{
							Name:  "Default",
							Vhost: "localhost",
						},
						{
							Name:  "Prod",
							Vhost: "localhost",
						},
					},
					Version:    "1.0.0",
					APIContext: "/testorg/myorg/v1",
				},
				{
					APIFile: "50d2cda9e9cab41a3244fa91-70d2cdade9cab41a3214fa98",
					Environments: []sync.GatewayLabel{
						{
							Name:  "Default",
							Vhost: "localhost",
						},
					},
					Version:    "1.0.0",
					APIContext: "/testorg/myorg/v1",
				},
			},
		},
	}
	go AddAPIEventsToChannel(&deploymentDescriptor)
	// Consume API events from channel.
	APIEventsArray := <-APIDeployAndRemoveEventChannel

	// Check UUIDs of API events.
	assert.Equal(t, APIEventsArray[0].UUID, "60d2cda9e9cab41a3214fa91",
		"UUID should be the same in received API event.")
	assert.Equal(t, APIEventsArray[1].UUID, "50d2cda9e9cab41a3244fa91",
		"UUID should be the same in received API event.")

	// Check Revision IDs of API events.
	assert.Equal(t, APIEventsArray[0].RevisionID, "60d2cdade9cab41a3214fa92",
		"RevisionID should be the same in received API event.")
	assert.Equal(t, APIEventsArray[1].RevisionID, "70d2cdade9cab41a3214fa98",
		"RevisionID should be the same in received API event.")

	// Check gateway labels of API events.
	assert.Equal(t, APIEventsArray[0].GatewayLabels[0], "Default",
		"GatewayLabels should be the same in received API event.")
	assert.Equal(t, APIEventsArray[0].GatewayLabels[1], "Prod",
		"GatewayLabels should be the same in received API event.")
	assert.Equal(t, APIEventsArray[1].GatewayLabels[0], "Default",
		"GatewayLabels should be the same in received API event.")
}
