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
	"time"

	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/config"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"
	"github.com/wso2/product-microgateway/adapter/pkg/health"
	msg "github.com/wso2/product-microgateway/adapter/pkg/messaging"
)

const (
	componentName = "ga" /*should not use longer names as subscription name has a limited
			            length of 50 and case-insensitive. Sample unique
	                            subscription name would be ga_41b19c44-f9e0-4b9a-90e3-7599dc1c0545_sub*/
	subscriptionIdleTimeDuration = time.Duration(72 * time.Hour)
)

// InitiateAndProcessEvents to pass event consumption
func InitiateAndProcessEvents(config *config.Config) {
	var err error
	var reconnectRetryCount = config.ControlPlane.BrokerConnectionParameters.ReconnectRetryCount
	var reconnectInterval = config.ControlPlane.BrokerConnectionParameters.ReconnectInterval
	connectionString := config.ControlPlane.BrokerConnectionParameters.EventListeningEndpoints[0]
	subscriptionMetaDataList, err := msg.InitiateBrokerConnectionAndValidate(connectionString, componentName,
		reconnectRetryCount, reconnectInterval*time.Millisecond, subscriptionIdleTimeDuration)
	health.SetControlPlaneBrokerStatus(err == nil)
	if err == nil {
		logger.LoggerMsg.Info("Service bus meta data successfully initialized.")
		msg.InitiateConsumers(connectionString, subscriptionMetaDataList, reconnectInterval*time.Millisecond)
		go handleAzureNotification(config)
		go handleAzureTokenRevocation()
		go handleAzureBillingCycleResetEvents(config)
		go handleAzureStepQuotaThresholdEvents(config)
		go handleAzureOrganizationPurge()
	}
}
