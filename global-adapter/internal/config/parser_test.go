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

package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReadConfigs(t *testing.T) {
	// Read configuration file.
	os.Setenv("cp_admin_pwd", "admin")
	conf := ReadConfigs()
	controlPlane := conf.ControlPlane
	gaAPIServer := conf.GAAPIServer

	assert.Equal(t, controlPlane.ServiceURL, "https://apim:9443", "Service should be the different")
	assert.Equal(t, controlPlane.Username, "admin", "Usernames should be the same")
	assert.Equal(t, controlPlane.Password, "admin", "Passwords should be the same")
	assert.Equal(t, controlPlane.EnvironmentLabels, []string([]string{"Default", "Prod"}),
		"Environment labels should be the same")
	assert.Equal(t, controlPlane.RetryInterval, time.Duration(5), "RetryInterval should be the same")
	assert.Equal(t, controlPlane.SkipSSLVerification, true, "SkipSSLVerification value should be the same")
	assert.Equal(t, controlPlane.BrokerConnectionParameters.EventListeningEndpoints,
		[]string{"amqp://admin:admin@apim:5672?retries='10'&connectdelay='30'"},
		"EventListeningEndpoints should be the same")

	assert.Equal(t, gaAPIServer.Host, "0.0.0.0", "Host should be the different")
	assert.Equal(t, gaAPIServer.Port, "9745", "Port should be the same")
	assert.Equal(t, gaAPIServer.Username, "admin", "Usernames should be the same")
}
