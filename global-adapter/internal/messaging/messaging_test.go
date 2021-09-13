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
	"testing"

	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/config"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"
	sync "github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/synchronizer"
	msg "github.com/wso2/product-microgateway/adapter/pkg/messaging"
)

func TestAPIRemoveEventsHandling(t *testing.T) {
	conf := config.ReadConfigs()
	apiRemoveEvent := `{"apiId":2,"uuid":"60d2fe0f1702d40469718ba2","name":"HelloWorld","version":"1.0.0","provider":"admin","apiType":"HTTP","gatewayLabels":["Prod"],"associatedApis":[],"context":"/hello/1.0.0","eventId":"8b4bdd1c-fb5a-4c8a-be73-2e1094cd27c2","timeStamp":1625193215839,"type":"REMOVE_API_FROM_GATEWAY","tenantId":0,"tenantDomain":"WSO2"}`
	eventByteArray := []byte(string(apiRemoveEvent))
	go handleAPIDeployAndRemoveEvents(eventByteArray, "REMOVE_API_FROM_GATEWAY", conf)
	event := <-sync.APIDeployAndRemoveEventChannel
	removeEvent, _ := json.Marshal(event)
	actualRemoveEvent := `[{"UUID":"60d2fe0f1702d40469718ba2","RevisionID":"","Context":"/hello/1.0.0","Version":"1.0.0","GatewayLabels":["Prod"],"OrganizationID":"WSO2","IsRemoveEvent":true,"IsReload":false}]`
	assert.Equal(t, actualRemoveEvent, string(removeEvent), "REMOVE_API_FROM_GATEWAY event is not correctly received")
}

func TestNotificationChannelSubscriptionAndEventFormat(t *testing.T) {
	logger.LoggerMsg.Infof("[TEST][FEATURE_FLAG_REPLACE_EVENT_HUB] starting test " +
		"TestNotificationChannelSubscriptionAndEventFormat")

	sampleTestEvent := "{\"event\":{\"payloadData\":{\"eventType\":\"API_CREATE\",\"timestamp\":1628490908147," +
		"\"event\":\"eyJhcGlOYW1lIjoiTXlBUEkiLCJhcGlJZCI6MiwidXVpZCI6Ijc4MDhhZjg0LTZiOWEtNGM4Ni05NTNhL" +
		"TRmNDBmMTU3NjcxZiIsImFwaVZlcnNpb24iOiJ2MSIsImFwaUNvbnRleHQiOiIvbXlhcGkvdjEiLCJhcGlQcm92aWRlc" +
		"iI6ImFkbWluIiwiYXBpVHlwZSI6IkhUVFAiLCJhcGlTdGF0dXMiOiJDUkVBVEVEIiwiZXZlbnRJZCI6IjE0NjY3Mz" +
		"A0LTIzZGQtNGI5Zi04YzM5LWExMTAzZDA2ZDA1OCIsInRpbWVTdGFtcCI6MTYyODQ5MDkwODE0NywidHlwZSI" +
		"6IkFQSV9DUkVBVEUiLCJ0ZW5hbnRJZCI6LTEyMzQsInRlbmFudERvbWFpbiI6ImNhcmJvbi5zdXBlciJ9\"}}}"

	var parsedSuccessfully bool
	var notification msg.EventNotification
	go func() {
		msg.AzureNotificationChannel <- []byte(sampleTestEvent)
	}()
	outputData := <-msg.AzureNotificationChannel
	logger.LoggerMsg.Infof("[TEST][FEATURE_FLAG_REPLACE_EVENT_HUB] Event %s is received from channel", outputData)
	error := parseNotificationJSONEvent(outputData, &notification)
	if error != nil {
		logger.LoggerMsg.Info("[TEST][FEATURE_FLAG_REPLACE_EVENT_HUB] Error occurred", error)
	} else {
		parsedSuccessfully = true
		logger.LoggerMsg.Infof("[TEST][FEATURE_FLAG_REPLACE_EVENT_HUB] Event %s is received",
			notification.Event.PayloadData.EventType)
	}
	assert.Equal(t, true, parsedSuccessfully)
}

func TestTokenRevocationChannelSubscriptionAndEventFormat(t *testing.T) {
	logger.LoggerMsg.Infof("[TEST][FEATURE_FLAG_REPLACE_EVENT_HUB] starting test " +
		"TestTokenRevocationChannelSubscriptionAndEventFormat")

	sampleTestEvent := "{\"event\":{\"payloadData\":{\"eventId\":\"444d2f9b-57d8-4245-bef2-3f8d824741c3\"," +
		"\"revokedToken\":\"fc8ee897-b3d9-3bb6-a9ca-f4aeb036e5c0\",\"ttl\":\"5000\",\"expiryTime\":1628175421481," +
		"\"type\":\"Default\",\"tenantId\":-1234}}}"
	var parsedSuccessfully bool
	var notification msg.EventTokenRevocationNotification
	go func() {
		msg.AzureRevokedTokenChannel <- []byte(sampleTestEvent)
	}()
	outputData := <-msg.AzureRevokedTokenChannel
	logger.LoggerMsg.Infof("[TEST][FEATURE_FLAG_REPLACE_EVENT_HUB] Event %s is received from channel", outputData)
	error := parseRevokedTokenJSONEvent(outputData, &notification)
	if error != nil {
		logger.LoggerMsg.Info("[TEST][FEATURE_FLAG_REPLACE_EVENT_HUB] Error occurred", error)
	} else {
		parsedSuccessfully = true
		logger.LoggerMsg.Infof("[TEST][FEATURE_FLAG_REPLACE_EVENT_HUB] Event %s is received",
			notification.Event.PayloadData.Type)
		logger.LoggerMsg.Info("[TEST][FEATURE_FLAG_REPLACE_EVENT_HUB] Revoked token value is ",
			notification.Event.PayloadData.RevokedToken)
	}
	assert.Equal(t, true, parsedSuccessfully)
}

func TestBillingCycleResetChannelSubscriptionAndEventFormat(t *testing.T) {
	logger.LoggerMsg.Infof("Starting test " + "TestBillingCycleResetChannelSubscriptionAndEventFormat")

	sampleTestEvent := "{\"orgUuid\":\"13550d86-1c68-4b3d-b756-eb55efda3eb5\",\"orgHandle\":\"flamboyantfrog\",\"monthOfYear\":10}"
	var parsedSuccessfully bool
	var resetEvent BillingCycleResetEvent
	go func() {
		msg.AzureStepQuotaResetChannel <- []byte(sampleTestEvent)
	}()
	outputData := <-msg.AzureStepQuotaResetChannel
	logger.LoggerMsg.Infof("Event %s is received from channel", outputData)
	err := parseBillingCycleResetJSONEvent(outputData, &resetEvent)
	if err != nil {
		logger.LoggerMsg.Errorf("Error occurred while parsing BillingCycleResetEvent. Error: %v", err)
	} else {
		parsedSuccessfully = true
		logger.LoggerMsg.Infof("Event is received for the org ID %s for the billing cycle reset month: %v",
			resetEvent.OrgUUID, resetEvent.MonthOfYear)
	}
	assert.Equal(t, true, parsedSuccessfully)
}

func TestStepQuotaThresholdChannelSubscriptionAndEventFormat(t *testing.T) {
	logger.LoggerMsg.Infof("Starting test TestStepQuotaThresholdChannelSubscriptionAndEventFormat")

	sampleTestEvent := "{\"orgId\":\"13550d86-1c68-4b3d-b756-eb55efda3eb5\",\"orgHandle\":\"flamboyantfrog\"," +
		"\"tierName\":\"free-tier\",\"threshold\":100,\"step_usage\":100721}"
	var parsedSuccessfully bool
	var thresholdEvent ThresholdReachedEvent
	go func() {
		msg.AzureStepQuotaThresholdChannel <- []byte(sampleTestEvent)
	}()
	outputData := <-msg.AzureStepQuotaThresholdChannel
	logger.LoggerMsg.Infof("Event %s is received from channel", outputData)
	err := parseStepQuotaThresholdJSONEvent(outputData, &thresholdEvent)
	if err != nil {
		logger.LoggerMsg.Errorf("Error occurred while parsing ThresholdReachedEvent. Error: %v", err)
	} else {
		parsedSuccessfully = true
		logger.LoggerMsg.Infof("Event is received for the org ID: %s. Step quota usage is: %v", thresholdEvent.OrgID,
			thresholdEvent.StepUsage)
	}
	assert.Equal(t, true, parsedSuccessfully)
}

func parseRevokedTokenJSONEvent(data []byte, notification *msg.EventTokenRevocationNotification) error {
	unmarshalErr := json.Unmarshal(data, &notification)
	if unmarshalErr != nil {
		logger.LoggerMsg.Errorf("Error occurred while unmarshalling revoked token event data %v", unmarshalErr)
	}
	return unmarshalErr
}
