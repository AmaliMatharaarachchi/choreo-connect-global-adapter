#  Copyright (c) 2020, WSO2 Inc. (http://www.wso2.com). All Rights Reserved.
#
#  This software is the property of WSO2 Inc. and its suppliers, if any.
#  Dissemination of any information or reproduction of any material contained
#  herein is strictly forbidden, unless permitted by WSO2 in accordance with
#  the WSO2 Commercial License available at http://wso2.com/licenses.
#  For specific language governing the permissions and limitations under
#  this license, please see the license as well as any agreement you’ve
#  entered into with WSO2 governing the purchase of this software and any
#  associated services.

PROJECT_ROOT := $(realpath $(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
PROJECT_NAME := choreo-connect-global-adapter
PROJECT_PKG := github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter
BUILD_DIRECTORY := build
BUILD_ROOT := $(PROJECT_ROOT)/$(BUILD_DIRECTORY)
GO_FILES		= $(shell find . -type f -name '*.go' -not -path "./vendor/*")
GIT_REVISION := $(shell git rev-parse --verify HEAD)

.PHONY: test
test:
	@go test -v -race -covermode=atomic -coverprofile=$(PROJECT_ROOT)/coverage.txt ./cmd/... ./internal/...

coverage: test
	go tool cover -html=$(PROJECT_ROOT)/coverage.txt
