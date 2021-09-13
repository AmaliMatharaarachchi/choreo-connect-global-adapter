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

package synchronizer

// APIEvent is the structure of an API event in GA.
type APIEvent struct {
	UUID           string
	RevisionID     string
	Context        string
	Version        string
	GatewayLabels  []string
	OrganizationID string
	IsRemoveEvent  bool
	IsReload       bool
}

// CpError Control Plane error structure for runtime-artifact
type CpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
