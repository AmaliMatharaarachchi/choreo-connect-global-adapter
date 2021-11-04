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
 */

package apipartition

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/synchronizer"
)

func TestGetCacheKey(t *testing.T) {
	api := synchronizer.APIEvent{Context: "/shanakama/worldbank/api/dev/v1.0", OrganizationID: "bda17a6c-f50d-49b9-b48a-83913b00b459", Version: "v1.0", GatewayLabels: []string{"dev"}}
	cacheKey := getCacheKey(&api, api.GatewayLabels[0])
	assert.Equal(t, cacheKey, "#global-adapter#dev#bda17a6c-f50d-49b9-b48a-83913b00b459#/worldbank/api/dev/v1.0", "Invalid cache key")

	api2 := synchronizer.APIEvent{Context: "/shanakama/worldbank/api/dev/v1.0.0", OrganizationID: "bda17a6c-f50d-49b9-b48a-83913b00b459", Version: "v1.0.0", GatewayLabels: []string{"prod"}}
	cacheKey2 := getCacheKey(&api2, api2.GatewayLabels[0])
	assert.Equal(t, cacheKey2, "#global-adapter#prod#bda17a6c-f50d-49b9-b48a-83913b00b459#/worldbank/api/dev/v1.0.0", "Invalid cache key")
}

func TestGetLaLabel(t *testing.T) {
	label := getLaLabel("dev", 1, 10)
	assert.Equal(t, label, "dev-p1", "Label Invalid")

	label2 := getLaLabel("dev", 12, 5)
	assert.Equal(t, label2, "dev-p3", "Label Invalid")
}
