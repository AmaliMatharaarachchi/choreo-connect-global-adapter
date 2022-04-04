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

// Package callbacks is used to intercept the XDS requests/responses
package callbacks

import (
	"context"
	"strings"

	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/xds"
)

// Callbacks is used to debug the xds server related communication.
type Callbacks struct {
}

// Report logs the fetches and requests.
func (cb *Callbacks) Report() {}

// OnStreamOpen prints debug logs
func (cb *Callbacks) OnStreamOpen(_ context.Context, id int64, typ string) error {
	logger.LoggerXdsCallbacks.Debugf("stream %d open for %s\n", id, typ)
	return nil
}

// OnStreamClosed prints debug logs
func (cb *Callbacks) OnStreamClosed(id int64) {
	logger.LoggerXdsCallbacks.Debugf("stream %d closed\n", id)
}

// OnStreamRequest prints debug logs
func (cb *Callbacks) OnStreamRequest(id int64, request *discovery.DiscoveryRequest) error {
	logger.LoggerXdsCallbacks.Debugf("stream request on stream id: %d, from node: %s, version: %s, for type: %s",
		id, request.GetNode(), request.GetVersionInfo(), request.GetTypeUrl())
	if request.ErrorDetail != nil {
		logger.LoggerXdsCallbacks.Errorf("Stream request for type %s on stream id: %d Error: %s", request.GetTypeUrl(), id, request.ErrorDetail.Message)
	}
	_, err := xds.GetAPICache().GetSnapshot(request.GetNode().Id)
	if err != nil && strings.Contains(err.Error(), "no snapshot found for node") {
		// This will be called only after the readiness probe is deployed.
		// Hence, there is no possibility to set empty snapshot for woking adapter (with APIs)
		// (i.e setting snapshot before adding APIs to the cache)
		errSetSnap := xds.SetEmptySnapshot(request.GetNode().Id)
		if errSetSnap != nil {
			logger.LoggerXdsCallbacks.Errorf("Error while setting empty snapshot. error : %s", errSetSnap.Error())
			return nil
		}
		logger.LoggerXdsCallbacks.Infof("Updated empty snapshot into cache as there is no apis for the label : %v", request.GetNode().Id)
	}
	return nil
}

// OnStreamResponse prints debug logs
func (cb *Callbacks) OnStreamResponse(context context.Context, id int64, request *discovery.DiscoveryRequest, response *discovery.DiscoveryResponse) {
	logger.LoggerXdsCallbacks.Debugf("stream response on stream id: %d node: %s for type: %s version: %s",
		id, request.GetNode(), request.GetTypeUrl(), response.GetVersionInfo())
}

// OnFetchRequest prints debug logs
func (cb *Callbacks) OnFetchRequest(_ context.Context, req *discovery.DiscoveryRequest) error {
	logger.LoggerXdsCallbacks.Debugf("fetch request from node: %s, version: %s, for type: %s", req.Node.Id, req.VersionInfo, req.TypeUrl)
	return nil
}

// OnFetchResponse prints debug logs
func (cb *Callbacks) OnFetchResponse(req *discovery.DiscoveryRequest, res *discovery.DiscoveryResponse) {
	logger.LoggerXdsCallbacks.Debugf("fetch response to node: %s, version: %s, for type: %s", req.Node.Id, req.VersionInfo, res.TypeUrl)
}

// OnDeltaStreamOpen is unused.
func (cb *Callbacks) OnDeltaStreamOpen(_ context.Context, id int64, typ string) error {
	return nil
}

// OnDeltaStreamClosed is unused.
func (cb *Callbacks) OnDeltaStreamClosed(id int64) {
}

// OnStreamDeltaResponse is unused.
func (cb *Callbacks) OnStreamDeltaResponse(id int64, req *discovery.DeltaDiscoveryRequest, res *discovery.DeltaDiscoveryResponse) {
}

// OnStreamDeltaRequest is unused.
func (cb *Callbacks) OnStreamDeltaRequest(id int64, req *discovery.DeltaDiscoveryRequest) error {
	return nil
}
