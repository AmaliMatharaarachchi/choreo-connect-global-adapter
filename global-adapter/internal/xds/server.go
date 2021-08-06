/*
 *  Copyright (c) 2021, WSO2 Inc. (http://www.wso2.org).
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
	"strings"
	"time"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"
	internal_types "github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/types"
	ga_api "github.com/wso2/product-microgateway/adapter/pkg/discovery/api/wso2/discovery/ga"
	wso2_cache "github.com/wso2/product-microgateway/adapter/pkg/discovery/protocol/cache/v3"
)

var (
	apiCache wso2_cache.SnapshotCache
	// The labels with partition IDs are stored here. <LabelHirerarchy>-P:<partition_ID>
	// TODO: (VirajSalaka) change the implementation of the snapshot library to provide the same information.
	introducedLabels map[string]bool
)

const (
	maxRandomInt int    = 999999999
	typeURL      string = "type.googleapis.com/wso2.discovery.ga.Api"
)

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
	introducedLabels = make(map[string]bool, 1)
}

// GetAPICache returns API Cache
func GetAPICache() wso2_cache.SnapshotCache {
	return apiCache
}

// addSingleAPI adds the API entry to XDS cache. Label should contain the paritionID along with label hierarchy.
func addSingleAPI(label, apiUUID, revisionUUID, organizationUUID string) {
	logger.LoggerXds.Infof("Deploy API is triggered for %s:%s under revision: %s", label, apiUUID, revisionUUID)
	var newSnapshot wso2_cache.Snapshot
	version := rand.Intn(maxRandomInt)
	api := &ga_api.Api{
		ApiUUID:          apiUUID,
		RevisionUUID:     revisionUUID,
		OrganizationUUID: organizationUUID,
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
	introducedLabels[label] = true
	logger.LoggerXds.Infof("API Snapshot is updated for label %s with the version %d.", label, version)
}

// removeAPI removes the API entry from XDS cache
func removeAPI(labelHierarchy, apiUUID string) {
	logger.LoggerXds.Infof("Remove API is triggered for %s", apiUUID)
	var newSnapshot wso2_cache.Snapshot
	version := rand.Intn(maxRandomInt)
	for label := range introducedLabels {
		// If the label does not starts with label hierarchy, that means the API is not required to be removed from
		// that specific environment.
		if !strings.HasPrefix(label, labelHierarchy) {
			continue
		}
		currentSnapshot, err := apiCache.GetSnapshot(label)
		if err != nil {
			// non reachable code as the implementation iterates over available labels
			continue
		}

		resourceMap := currentSnapshot.GetResources(typeURL)
		_, apiFound := resourceMap[apiUUID]
		// If the API is found, then the xds cache is updated and returned.
		if apiFound {
			logger.LoggerXds.Debugf("API : %s is found within snapshot for label %s", apiUUID, label)
			delete(resourceMap, apiUUID)
			apiResources := convertResourceMapToArray(resourceMap)
			newSnapshot = wso2_cache.NewSnapshot(fmt.Sprint(version), nil, nil, nil, nil, nil, nil,
				nil, nil, nil, nil, nil, apiResources)
			apiCache.SetSnapshot(label, newSnapshot)
			logger.LoggerXds.Infof("API Snaphsot is updated for label %s with the version %d.", label, version)
			return
		}
	}
	logger.LoggerXds.Errorf("API : %s is not found within snapshot for label hierarchy %s", apiUUID, labelHierarchy)
}

// ProcessSingleEvent is triggered when there is a single event needs to be processed(Corresponding to JMS Events)
func ProcessSingleEvent(event *internal_types.LaAPIEvent) {
	if event.IsRemoveEvent {
		removeAPI(event.LabelHierarchy, event.APIUUID)
	} else {
		addSingleAPI(event.Label, event.APIUUID, event.RevisionUUID, event.OrganizationUUID)
	}
}

// AddMultipleAPIs adds the multiple APIs entry to XDS cache (used for statup)
func AddMultipleAPIs(apiEventArray []*internal_types.LaAPIEvent) {

	snapshotMap := make(map[string]*wso2_cache.Snapshot)
	version := rand.Intn(maxRandomInt)
	for _, event := range apiEventArray {
		label := event.Label
		apiUUID := event.APIUUID
		revisionUUID := event.RevisionUUID
		organizationUUID := event.OrganizationUUID
		api := &ga_api.Api{
			ApiUUID:          apiUUID,
			RevisionUUID:     revisionUUID,
			OrganizationUUID: organizationUUID,
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
		logger.LoggerXds.Infof("Deploy API is triggered for %s:%s under revision: %s in startup", label, apiUUID, revisionUUID)
	}

	for label, snapshotEntry := range snapshotMap {
		apiCache.SetSnapshot(label, *snapshotEntry)
		introducedLabels[label] = true
		logger.LoggerXds.Infof("API Snaphsot is updated for label %s with the version %d.", label, version)
	}
}

func convertResourceMapToArray(resourceMap map[string]types.Resource) []types.Resource {
	apiResources := []types.Resource{}
	for _, res := range resourceMap {
		apiResources = append(apiResources, res)
	}
	return apiResources
}
