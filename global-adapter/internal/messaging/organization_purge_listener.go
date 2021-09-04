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
	"encoding/json"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/apipartition"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"
	msg "github.com/wso2/product-microgateway/adapter/pkg/messaging"
)

func handleAzureOrganizationPurge() {
	for d := range msg.AzureOrganizationPurgeChannel {
		logger.LoggerMsg.Info("[TEST][FEATURE_FLAG_REPLACE_EVENT_HUB] message received for " +
			"OrganizationPurge = " + string(d))
		var notification msg.EventOrganizationPurge
		error := parseOrganizationPurgeJSONEvent(d, &notification)
		if error != nil {
			logger.LoggerMsg.Errorf("[TEST][FEATURE_FLAG_REPLACE_EVENT_HUB] Error while processing "+
				"the organization purge event %v. Hence dropping the event", error)
			continue
		}
		logger.LoggerMsg.Infof("[TEST][FEATURE_FLAG_REPLACE_EVENT_HUB] Event %s is received",
			notification.Event.PayloadData.Type)

		apipartition.DeleteApiRecords(notification.Event.PayloadData.Apis, notification.Event.PayloadData.Organization)
	}
}

func parseOrganizationPurgeJSONEvent(data []byte, notification *msg.EventOrganizationPurge) error {
	unmarshalErr := json.Unmarshal(data, &notification)
	if unmarshalErr != nil {
		logger.LoggerMsg.Errorf("Error occurred while unmarshalling revoked token event data %v", unmarshalErr)
	}
	return unmarshalErr
}
