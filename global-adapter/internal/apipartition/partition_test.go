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
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/types"
)

func TestGetCacheKey(t *testing.T) {
	api := types.API{Context: "/shanakama/worldbank/api/dev", Version: "v1.0", GwLabel: []string{"dev"}}
	cacheKey := *getCacheKey(&api, &api.GwLabel[0])
	assert.Equal(t, cacheKey, "global-adapter#dev#shanakama_/worldbank/api/dev_v1.0", "Invalid cache key")

	api2 := types.API{Context: "/shanakama/worldbank/api/dev", Version: "v1.0.0", GwLabel: []string{"prod"}}
	cacheKey2 := *getCacheKey(&api2, &api2.GwLabel[0])
	assert.Equal(t, cacheKey2, "global-adapter#prod#shanakama_/worldbank/api/dev_v1.0.0", "Invalid cache key")
}

func TestGetLaLabel(t *testing.T) {
	label := getLaLabel("dev", 1, 10)
	assert.Equal(t, label, "devP-1", "Label Invalid")

	label2 := getLaLabel("dev", 12, 5)
	assert.Equal(t, label2, "devP-3", "Label Invalid")
}
