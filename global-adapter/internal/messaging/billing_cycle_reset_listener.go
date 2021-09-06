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

func handleAzureBillingCycleResetEvents(conf *config.Config) {
	for d := range msg.AzureStepQuotaResetChannel {
		logger.LoggerMsg.Info("Message received for AzureStepQuotaResetChannel = " + string(d))
		var resetEvent BillingCycleResetEvent
		err := parseBillingCycleResetJSONEvent(d, &resetEvent)
		if err != nil {
			logger.LoggerMsg.Errorf("Error while processing the billing cycle reset event %v. Hence dropping the event", err)
			continue
		}
		logger.LoggerMsg.Debugf("Billing cycle reset event for org ID: %s is received", resetEvent.OrgUUID)

		dbErr := upsertQuotaExceededStatus(resetEvent.OrgUUID, false)
		if dbErr != nil {
			logger.LoggerMsg.Errorf("Failed to upsert quota exceeded status: %v in DB for org: %s. Error: %v", false, resetEvent.OrgUUID, dbErr)
			continue
		}

		// API IDs for org
		apiIds, err := getAPIIdsForOrg(resetEvent.OrgUUID)
		if err != nil {
			logger.LoggerMsg.Errorf("Failed to get API IDs for org: %s. Error: %v", resetEvent.OrgUUID, err)
			continue
		}

		logger.LoggerMsg.Debugf("Found API IDs: %v for org: %s", apiIds, resetEvent.OrgUUID)
		apiEvents, err := getAPIEvents(apiIds, conf)
		if err != nil {
			logger.LoggerMsg.Errorf("Failed to get API event for org: %s. Error: %v", resetEvent.OrgUUID, err)
			continue
		}

		logger.LoggerMsg.Debugf("Got API Events: %v for org: %s", apiEvents, resetEvent.OrgUUID)
		for _, apiEvent := range apiEvents {
			apipartition.UpdateCacheForQuotaExceededStatus(apiEvent, "")
		}
	}
}

func parseBillingCycleResetJSONEvent(data []byte, billingCycleResetEvent *BillingCycleResetEvent) error {
	unmarshalErr := json.Unmarshal(data, &billingCycleResetEvent)
	if unmarshalErr != nil {
		logger.LoggerMsg.Errorf("Error occurred while unmarshalling billing cycle reset event data %v", unmarshalErr)
	}
	logger.LoggerMsg.Debugf("Successfully parsed billing cycle reset Json event. Data: %v", unmarshalErr)
	return unmarshalErr
}
