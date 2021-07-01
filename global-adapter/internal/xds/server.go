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

package xds

import (
	"fmt"
	"math/rand"
	"time"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	ga_api "github.com/wso2/product-microgateway/adapter/pkg/discovery/api/wso2/discovery/ga"
	wso2_cache "github.com/wso2/product-microgateway/adapter/pkg/discovery/protocol/cache/v3"
)

var (
	apiCache wso2_cache.SnapshotCache
)

const (
	maxRandomInt int    = 999999999
	typeURL      string = "type.googleapis.com/wso2.discovery.ga.Api"
)

// APIInboundEvent is the event accepted by the module to push it to the cache
type APIInboundEvent struct {
	APIUUID       string
	RevisionUUID  string
	Label         string
	IsRemoveEvent bool
}

// IDHash uses ID field as the node hash.
type IDHash struct{}

// ID uses the node ID field
func (IDHash) ID(node *corev3.Node) string {
	if node == nil {
		return "unknown"
	}
	return node.Id
}

var _ wso2_cache.NodeHash = IDHash{}

func init() {
	apiCache = wso2_cache.NewSnapshotCache(false, IDHash{}, nil)
	rand.Seed(time.Now().UnixNano())
}

// GetAPICache returns API Cache
func GetAPICache() wso2_cache.SnapshotCache {
	return apiCache
}

// AddAPIsToCache adds the provided set of APIUUIDs and updates the XDS cache for the provided label.
func AddAPIsToCache() {
	arr := make([]*APIInboundEvent, 3)
	arr[0] = &APIInboundEvent{
		APIUUID:      "xyz1",
		RevisionUUID: "xyz1",
		Label:        "default",
	}
	arr[1] = &APIInboundEvent{
		APIUUID:      "xyz2",
		RevisionUUID: "xyz2",
		Label:        "default2",
	}
	arr[2] = &APIInboundEvent{
		APIUUID:      "xyz3",
		RevisionUUID: "xyz3",
		Label:        "default",
	}

	AddMultipleAPIs(arr)
	time.Sleep(10 * time.Second)
	addSingleAPI("default", "apiID1", "1234")
	time.Sleep(10 * time.Second)
	addSingleAPI("default", "apiID2", "1234")
	time.Sleep(5 * time.Second)
	addSingleAPI("default", "apiID1", "4567")
	time.Sleep(5 * time.Second)
	removeAPI("default", "apiID1")
}

// addSingleAPI adds the API entry to XDS cache
func addSingleAPI(label, apiUUID, revisionUUID string) {
	//debug
	fmt.Printf("Deploy API is triggered for %s:%s under revision: %s\n", label, apiUUID, revisionUUID)
	var newSnapshot wso2_cache.Snapshot
	version := rand.Intn(maxRandomInt)
	api := &ga_api.Api{
		ApiUUID:      apiUUID,
		RevisionUUID: revisionUUID,
	}
	currentSnapshot, err := apiCache.GetSnapshot(label)

	// error occurs if no snapshot is under the provided label
	if err != nil {
		newSnapshot = wso2_cache.NewSnapshot(fmt.Sprint(version), nil, nil, nil, nil, nil, nil,
			nil, nil, nil, nil, nil, []types.Resource{api})
	} else {
		resourceMap := currentSnapshot.GetResources(typeURL)
		resourceMap[apiUUID] = api
		apiResources := convertResourceMapToArray(resourceMap)
		newSnapshot = wso2_cache.NewSnapshot(fmt.Sprint(version), nil, nil, nil, nil, nil, nil,
			nil, nil, nil, nil, nil, apiResources)
	}
	apiCache.SetSnapshot(label, newSnapshot)
	fmt.Printf("API Snaphsot is updated for label %s with the version %d. \n", label, version)
}

// removeAPI removes the API entry from XDS cache
func removeAPI(label, apiUUID string) {
	//debug
	fmt.Printf("Remove API is triggered for %s:%s \n", label, apiUUID)
	var newSnapshot wso2_cache.Snapshot
	version := rand.Intn(maxRandomInt)
	currentSnapshot, err := apiCache.GetSnapshot(label)

	if err != nil {
		fmt.Printf("API : %s is not found within snapshot for label %s \n", apiUUID, label)
		return
	}
	resourceMap := currentSnapshot.GetResources(typeURL)
	delete(resourceMap, apiUUID)
	apiResources := convertResourceMapToArray(resourceMap)
	newSnapshot = wso2_cache.NewSnapshot(fmt.Sprint(version), nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil, apiResources)
	apiCache.SetSnapshot(label, newSnapshot)
	fmt.Printf("API Snaphsot is updated for label %s with the version %d. \n", label, version)
}

// ProcessSingleEvent is triggered when there is a single event needs to be processed(Corresponding to JMS Events)
func ProcessSingleEvent(event *APIInboundEvent) {
	if event.IsRemoveEvent {
		removeAPI(event.Label, event.APIUUID)
	} else {
		addSingleAPI(event.Label, event.APIUUID, event.RevisionUUID)
	}
}

// AddMultipleAPIs adds the multiple APIs entry to XDS cache (used for statup)
func AddMultipleAPIs(apiEventArray []*APIInboundEvent) {

	snapshotMap := make(map[string]*wso2_cache.Snapshot)
	version := rand.Intn(maxRandomInt)
	for _, event := range apiEventArray {
		label := event.Label
		apiUUID := event.APIUUID
		revisionUUID := event.RevisionUUID
		api := &ga_api.Api{
			ApiUUID:      apiUUID,
			RevisionUUID: revisionUUID,
		}

		snapshotEntry, snapshotFound := snapshotMap[label]
		var newSnapshot wso2_cache.Snapshot
		if !snapshotFound {
			newSnapshot = wso2_cache.NewSnapshot(fmt.Sprint(version), nil, nil, nil, nil, nil, nil,
				nil, nil, nil, nil, nil, []types.Resource{api})
			snapshotEntry = &newSnapshot
			snapshotMap[label] = &newSnapshot
		} else {
			// error occurs if no snapshot is under the provided label
			resourceMap := snapshotEntry.GetResources(typeURL)
			resourceMap[apiUUID] = api
			apiResources := convertResourceMapToArray(resourceMap)
			newSnapshot = wso2_cache.NewSnapshot(fmt.Sprint(version), nil, nil, nil, nil, nil, nil,
				nil, nil, nil, nil, nil, apiResources)
			snapshotMap[label] = &newSnapshot
		}
		fmt.Printf("Deploy API is triggered for %s:%s under revision: %s\n", label, apiUUID, revisionUUID)
	}

	for label, snapshotEntry := range snapshotMap {
		apiCache.SetSnapshot(label, *snapshotEntry)
		fmt.Printf("API Snaphsot is updated for label %s with the version %d.\n", label, version)
	}
}

func convertResourceMapToArray(resourceMap map[string]types.Resource) []types.Resource {
	apiResources := []types.Resource{}
	for _, res := range resourceMap {
		apiResources = append(apiResources, res)
	}
	return apiResources
}
