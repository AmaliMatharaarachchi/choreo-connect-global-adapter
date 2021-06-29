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

package types

type API struct {
	APIID            int      `json:"apiId"`
	UUID             string   `json:"uuid"`
	Provider         string   `json:"provider" json:"apiProvider,omitempty"`
	Name             string   `json:"name" json:"apiName,omitempty"`
	Version          string   `json:"version" json:"apiVersion,omitempty"`
	Context          string   `json:"context" json:"apiContext,omitempty"`
	Policy           string   `json:"policy,omitempty"`
	APIType          string   `json:"apiType,omitempty"`
	IsDefaultVersion bool     `json:"isDefaultVersion,omitempty"`
	APIStatus        string   `json:"status,omitempty"`
	TenantID         int32    `json:"tenanId,omitempty"`
	TenantDomain     string   `json:"tenanDomain,omitempty"`
	TimeStamp        int64    `json:"timeStamp,omitempty"`
	GwLabel          []string `json:"gwLabel, omitempty"`
	RevisionId       string   `json:"revuuid,omitempty"`
	Organization     string   `json:"revuuid,omitempty"`
}

type Response struct {
	Count      int    `json:"count,omitempty"`
	List       []API  `json:"list, omitempty"`
	Pagination string `json:"pagination, omitempty"`
}

type EvetType int

const (
	API_CREATE EvetType = iota
	API_DELETE
	DEPLOY_API_IN_GATEWAY
)

type ApiState struct {
	LabelHierarchy string
	Label          string
	Revision       string
	EventType      EvetType
}
