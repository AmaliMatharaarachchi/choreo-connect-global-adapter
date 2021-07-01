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

package xds_test

import (
	"testing"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/stretchr/testify/assert"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/xds"
	ga_model "github.com/wso2/product-microgateway/adapter/pkg/discovery/api/wso2/discovery/ga"
)

const (
	typeURL string = "type.googleapis.com/wso2.discovery.ga.Api"
)

func TestAddMultipleAPIs(t *testing.T) {
	initialArray := make([]*xds.APIInboundEvent, 3)
	label1 := "label1"
	label2 := "label2"
	api1 := "api1"
	api2 := "api2"
	api3 := "api3"
	api4 := "api4"
	revision1 := "1"
	revision2 := "2"

	// Test initial Addition
	initialArray[0] = &xds.APIInboundEvent{
		APIUUID:      api1,
		RevisionUUID: revision1,
		Label:        label1,
	}
	initialArray[1] = &xds.APIInboundEvent{
		APIUUID:      api2,
		RevisionUUID: revision1,
		Label:        label2,
	}
	initialArray[2] = &xds.APIInboundEvent{
		APIUUID:      api3,
		RevisionUUID: revision1,
		Label:        label1,
	}
	xds.AddMultipleAPIs(initialArray)
	snapshot1, err := xds.GetAPICache().GetSnapshot(label1)
	assert.Nil(t, err)
	snapshot2, err := xds.GetAPICache().GetSnapshot(label2)
	assert.Nil(t, err)
	assert.NotEmpty(t, snapshot1)
	assert.NotEmpty(t, snapshot2)
	assert.Len(t, snapshot1.GetResources(typeURL), 2)
	assert.Len(t, snapshot2.GetResources(typeURL), 1)

	testResourceContent(t, api1, revision1, snapshot1.GetResources(typeURL))
	testResourceContent(t, api2, revision1, snapshot2.GetResources(typeURL))
	testResourceContent(t, api3, revision1, snapshot1.GetResources(typeURL))

	// Tests the addition of a new API
	apiEvent4 := &xds.APIInboundEvent{
		APIUUID:      api4,
		RevisionUUID: revision1,
		Label:        label1,
	}
	xds.ProcessSingleEvent(apiEvent4)
	snapshot1, _ = xds.GetAPICache().GetSnapshot(label1)
	assert.Len(t, snapshot1.GetResources(typeURL), 3)
	testResourceContent(t, api4, revision1, snapshot1.GetResources(typeURL))

	apiEvent5 := &xds.APIInboundEvent{
		APIUUID:      api1,
		RevisionUUID: revision2,
		Label:        label1,
	}
	xds.ProcessSingleEvent(apiEvent5)
	snapshot1, _ = xds.GetAPICache().GetSnapshot(label1)
	assert.Len(t, snapshot1.GetResources(typeURL), 3)
	testResourceContent(t, api1, revision2, snapshot1.GetResources(typeURL))

	apiEvent6 := &xds.APIInboundEvent{
		APIUUID:       api1,
		Label:         label1,
		IsRemoveEvent: true,
	}
	xds.ProcessSingleEvent(apiEvent6)
	snapshot1, _ = xds.GetAPICache().GetSnapshot(label1)
	assert.Len(t, snapshot1.GetResources(typeURL), 2)
	_, resourceFound := snapshot1.GetResources(typeURL)[api1]
	assert.False(t, resourceFound)
}

func testResourceContent(t *testing.T, apiUUID, revisionUUID string, resourceMap map[string]types.Resource) {
	res, resFound := resourceMap[apiUUID]
	assert.True(t, resFound)
	// api := &ga_model.Api(res)
	switch api := res.(type) {
	case *ga_model.Api:
		assert.Equal(t, apiUUID, api.ApiUUID, "API UUID mismatch")
		assert.Equal(t, revisionUUID, api.RevisionUUID, "RevisionUUID mismatch")
	default:
		t.Error("Unexpected type returned from the resource map.")
	}
}
