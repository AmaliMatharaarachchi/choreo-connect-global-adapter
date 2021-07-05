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
 *  distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 *  WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 *
 */

// Package messaging holds the implementation for event listeners functions
package messaging

import (
	"github.com/wso2/product-microgateway/adapter/pkg/health"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/config"
	msg "github.com/wso2/product-microgateway/adapter/pkg/messaging"
)

// ProcessEvents to pass event consumption.
func ProcessEvents(config *config.Config) {
	err := msg.InitiateJMSConnection(config.ControlPlane.JmsConnectionParameters.EventListeningEndpoints)
	health.SetControlPlaneJmsStatus(err == nil)

	// Handle notification events.
	go handleNotification(config)
	// Handle non notification events.
	go handleNonNotification()
}

func handleNonNotification() {
	go handleKMConfiguration()
	go handleThrottleData()
	go handleTokenRevocation()
}
