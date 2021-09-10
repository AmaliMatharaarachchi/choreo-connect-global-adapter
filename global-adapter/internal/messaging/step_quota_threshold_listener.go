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
	msg "github.com/wso2/product-microgateway/adapter/pkg/messaging"
)

// StepThreshold  Step quota exceeded threshold value
const StepThreshold = 100

// RedisBlockedValue Step quota exceeded/reset org's redis value
const RedisBlockedValue = "blocked"

func handleAzureStepQuotaThresholdEvents(conf *config.Config) {
	for d := range msg.AzureStepQuotaThresholdChannel {
		logger.LoggerMsg.Info("Message received for AzureStepQuotaThresholdChannel for: " + string(d))

		if !apipartition.IsStepQuotaLimitingEnabled() {
			logger.LoggerMsg.Infof("Step quota limiting feature is disabled. Hence not processing event")
			continue
		}

		var thresholdEvent ThresholdReachedEvent
		err := parseStepQuotaThresholdJSONEvent(d, &thresholdEvent)
		if err != nil {
			logger.LoggerMsg.Errorf("Error while processing the step quota exceeded event %v. Hence dropping the event", err)
			continue
		}

		if thresholdEvent.StepUsage < StepThreshold {
			logger.LoggerMsg.Debugf("Step quota threshold not exceeded. Hence ignoring event")
			continue
		}

		logger.LoggerMsg.Debugf("Step quota exceeded event is received for org ID: %s", thresholdEvent.OrgID)

		err = upsertQuotaExceededStatus(thresholdEvent.OrgID, true)
		if err != nil {
			logger.LoggerMsg.Errorf("Failed to upsert quota exceeded status: %v in DB for org: %s. Error: %v", true, thresholdEvent.OrgID, err)
			continue
		}

		// API IDs for org
		apiIds, err := getAPIIdsForOrg(thresholdEvent.OrgID)
		if err != nil {
			logger.LoggerMsg.Errorf("Failed to get API IDs for org: %s. Error: %v", thresholdEvent.OrgID, err)
			continue
		}

		logger.LoggerMsg.Debugf("Found API IDs: %v for org: %s", apiIds, thresholdEvent.OrgID)
		apiEvents, err := getAPIEvents(apiIds, conf)
		if err != nil {
			logger.LoggerMsg.Errorf("Failed to get API event for org: %s. Error: %v", thresholdEvent.OrgID, err)
			continue
		}

		logger.LoggerMsg.Debugf("Got API Events: %v for org: %s. Hence updating redis cache",
			apiEvents, thresholdEvent.OrgID)
		updateCacheForAPIEvents(apiEvents, RedisBlockedValue)

		logger.LoggerMsg.Info("Completed handling Azure step quota threshold event for: " + string(d))
	}
}

func parseStepQuotaThresholdJSONEvent(data []byte, stepQuotaThresholdEvent *ThresholdReachedEvent) error {
	unmarshalErr := json.Unmarshal(data, &stepQuotaThresholdEvent)
	if unmarshalErr != nil {
		logger.LoggerMsg.Errorf("Error occurred while unmarshalling step quota threshold event data %v", unmarshalErr)
	}
	logger.LoggerMsg.Debugf("Successfully parsed step quota threshold Json event. Data: %v", unmarshalErr)
	return unmarshalErr
}
