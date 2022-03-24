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
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/config"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"
	sync "github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/synchronizer"
	msg "github.com/wso2/product-microgateway/adapter/pkg/messaging"
)

const (
	apiEventType       = "API"
	apiLifeCycleChange = "LIFECYCLE_CHANGE"
	deployAPI          = "DEPLOY_API_IN_GATEWAY"
	removeAPI          = "REMOVE_API_FROM_GATEWAY"
)

var (
	// NonAPIDeployAndRemoveEventChannel represents the channel for writing non API deploy and remove events.
	NonAPIDeployAndRemoveEventChannel chan []byte
)

func init() {
	NonAPIDeployAndRemoveEventChannel = make(chan []byte)
}

// Process incoming notifications.
func handleNotification(config *config.Config) {
	for delivery := range msg.NotificationChannel {
		var notification msg.EventNotification
		notificationErr := parseNotificationJSONEvent([]byte(string(delivery.Body)), &notification)
		if notificationErr != nil {
			continue
		}
		logger.LoggerMsg.Infof("Event %s is received", notification.Event.PayloadData.EventType)
		err := processNotificationEvent(config, &notification)
		if err != nil {
			continue
		}
		delivery.Ack(false)
	}
	logger.LoggerMsg.Infof("handle: deliveries channel closed")
}

func handleAzureNotification(config *config.Config) {
	for d := range msg.AzureNotificationChannel {
		logger.LoggerMsg.Infof("message received for NotificationChannel = " + string(d))
		var notification msg.EventNotification
		error := parseNotificationJSONEvent(d, &notification)
		if error != nil {
			continue
		}
		logger.LoggerMsg.Infof("Event %s is received", notification.Event.PayloadData.EventType)
		err := processNotificationEvent(config, &notification)
		if err != nil {
			continue
		}
	}
}

// handle API deploy and remove events.
func handleAPIDeployAndRemoveEvents(data []byte, eventType string, config *config.Config) {
	var apiEvent msg.APIEvent
	// Create an APIEvent array.
	APIEventArray := []sync.APIEvent{}

	apiEventErr := json.Unmarshal([]byte(string(data)), &apiEvent)
	if apiEventErr != nil {
		logger.LoggerMsg.Errorf("Error occurred while unmarshalling API event data %v", apiEventErr)
		return
	}
	// NOTE: GA only propagates API deploy and remove events.
	// Other API events(API create/update/delete) are not propagated as LA is not processing them.

	// Get runtime artifacts for api deploy events.
	if apiEvent.Event.Type == deployAPI {
		// Get runtime artifacts reladed to UUID and GatewayLabels  of the api event.
		apiEventErr = sync.GetArtifactsAndAddToChannel(&apiEvent.UUID, apiEvent.GatewayLabels, config, false, false)
		if apiEventErr != nil {
			logger.LoggerMsg.Errorf("Error occurred while getting runtime artifacts %v", apiEventErr)
			return
		}
	} else if apiEvent.Event.Type == removeAPI {
		// Write remove API Event to the APIDeployAndRemoveEventChannel.
		removeEvent := sync.APIEvent{}
		removeEvent.UUID = apiEvent.UUID
		removeEvent.Context = apiEvent.Context
		removeEvent.Version = apiEvent.Version
		removeEvent.GatewayLabels = apiEvent.GatewayLabels
		// TenantDomain is used to keep organization ID in choreo-scenario
		removeEvent.OrganizationID = apiEvent.TenantDomain
		removeEvent.IsRemoveEvent = true
		if removeEvent.GatewayLabels != nil {
			APIEventArray = append(APIEventArray, removeEvent)
			sync.APIDeployAndRemoveEventChannel <- sync.APIEventsWithStartupFlag{APIEvents: APIEventArray, IsStartup: false}
		}
	}
}

func parseNotificationJSONEvent(data []byte, notification *msg.EventNotification) error {
	unmarshalErr := json.Unmarshal(data, &notification)
	if unmarshalErr != nil {
		logger.LoggerMsg.Errorf("Error occurred while unmarshalling "+
			"notification event data %v. Hence dropping the event", unmarshalErr)
	}
	return unmarshalErr
}

func processNotificationEvent(conf *config.Config, notification *msg.EventNotification) error {
	var eventType string
	var decodedByte, err = base64.StdEncoding.DecodeString(notification.Event.PayloadData.Event)
	if err != nil {
		if _, ok := err.(base64.CorruptInputError); ok {
			logger.LoggerMsg.Errorf("\nbase64 input is corrupt, check the provided key")
		}
		logger.LoggerMsg.Errorf("Error occurred while decoding the notification event %v. "+
			"Hence dropping the event", err)
		return err
	}
	logger.LoggerMsg.Debugf("\n\n[%s]", decodedByte)
	eventType = notification.Event.PayloadData.EventType
	if strings.Contains(eventType, apiEventType) && !(strings.Contains(eventType, apiLifeCycleChange)) {
		handleAPIDeployAndRemoveEvents(decodedByte, eventType, conf)
	} else {
		logger.LoggerMsg.Debugf("Write non API Event %s to the NonAPIDeployAndRemoveEventChannel", decodedByte)
		// Write non API Event to the NonAPIDeployAndRemoveEventChannel
		NonAPIDeployAndRemoveEventChannel <- decodedByte
	}
	return nil
}
