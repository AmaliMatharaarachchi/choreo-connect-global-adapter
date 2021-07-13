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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/config"
	sync "github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/synchronizer"
)

func TestAPIRemoveEventsHandling(t *testing.T) {
	conf := config.ReadConfigs()
	apiRemoveEvent := `{"apiId":2,"uuid":"60d2fe0f1702d40469718ba2","name":"HelloWorld","version":"1.0.0","provider":"admin","apiType":"HTTP","gatewayLabels":["Prod"],"associatedApis":[],"context":"/hello/1.0.0","eventId":"8b4bdd1c-fb5a-4c8a-be73-2e1094cd27c2","timeStamp":1625193215839,"type":"REMOVE_API_FROM_GATEWAY","tenantId":0,"tenantDomain":"WSO2"}`
	eventByteArray := []byte(string(apiRemoveEvent))
	go handleAPIDeployAndRemoveEvents(eventByteArray, "REMOVE_API_FROM_GATEWAY", conf)
	event := <-sync.APIDeployAndRemoveEventChannel
	removeEvent, _ := json.Marshal(event)
	actualRemoveEvent := `[{"UUID":"60d2fe0f1702d40469718ba2","RevisionID":"","Context":"/hello/1.0.0","Version":"1.0.0","GatewayLabels":["Prod"],"OrganizationID":"WSO2","IsRemoveEvent":true}]`
	assert.Equal(t, actualRemoveEvent, string(removeEvent), "REMOVE_API_FROM_GATEWAY event is not correctly received")
}
