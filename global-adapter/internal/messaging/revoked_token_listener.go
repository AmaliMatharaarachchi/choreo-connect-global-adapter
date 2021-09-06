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
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"
	msg "github.com/wso2/product-microgateway/adapter/pkg/messaging"
	"encoding/json"
)

func handleTokenRevocation() {
	for delivery := range msg.RevokedTokenChannel {
		logger.LoggerMsg.Infof("Token Revocation Event %s is received", string(delivery.Body))
		writeNonAPIEventToChannel(delivery)
		delivery.Ack(false)
	}
	logger.LoggerMsg.Infof("handle: deliveries channel closed")
}

func handleAzureTokenRevocation() {
	for d := range msg.AzureRevokedTokenChannel {
		logger.LoggerMsg.Info("[TEST][FEATURE_FLAG_REPLACE_EVENT_HUB] message received for " +
			"RevokedTokenChannel = " + string(d))
		var notification msg.EventTokenRevocationNotification
		error := parseRevokedTokenJSONEvent(d, &notification)
		if error != nil {
			logger.LoggerMsg.Errorf("[TEST][FEATURE_FLAG_REPLACE_EVENT_HUB] Error while processing " +
				"the token revocation event %v. Hence dropping the event", error)
			continue
		}
		logger.LoggerMsg.Infof("[TEST][FEATURE_FLAG_REPLACE_EVENT_HUB] Event %s is received",
			notification.Event.PayloadData.Type)
		logger.LoggerMsg.Printf("[TEST][FEATURE_FLAG_REPLACE_EVENT_HUB] RevokedToken: %s, " +
			"Token Type: %s", notification.Event.PayloadData.RevokedToken,
			notification.Event.PayloadData.Type)
	}
}

func parseRevokedTokenJSONEvent(data []byte, notification *msg.EventTokenRevocationNotification) error {
	unmarshalErr := json.Unmarshal(data, &notification)
	if unmarshalErr != nil {
		logger.LoggerMsg.Errorf("Error occurred while unmarshalling revoked token event data %v", unmarshalErr)
	}
	return unmarshalErr
}
