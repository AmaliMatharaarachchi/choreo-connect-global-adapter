/*
 *  Copyright (c) 2021, WSO2 Inc. (http://www.wso2.org)
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

// Package logger contains the package references for log messages
// If a new package is introduced, the corresponding logger reference is need to be created as well.
package logger

import (
	"github.com/sirupsen/logrus"
	"github.com/wso2/product-microgateway/adapter/pkg/logging"
)

/* loggers should be initiated only for the main packages
 ********** Don't initiate loggers for sub packages ****************

When you add a new logger instance add the related package name as a constant
*/

// package name constants
const (
	pkgServer       = "github.com/wso2-enterprise/choreo-connnect-global-adapter/global-adapter/internal/server"
	pkgXds          = "github.com/wso2-enterprise/choreo-connnect-global-adapter/global-adapter/internal/xds"
	pkgXdsCallbacks = "github.com/wso2-enterprise/choreo-connnect-global-adapter/global-adapter/internal/xds/callbacks"
	pkgSync         = "github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/synchronizer"
	pkgMsg          = "github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/messaging"
	pkgHealth       = "github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/messaging"
	pkgAPIPartition = "github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/apipartition"
)

// logger package references
var (
	LoggerServer       *logrus.Logger
	LoggerXds          *logrus.Logger
	LoggerXdsCallbacks *logrus.Logger
	LoggerSync         *logrus.Logger
	LoggerMsg          *logrus.Logger
	LoggerHealth       *logrus.Logger
	LoggerAPIPartition *logrus.Logger
)

func init() {
	UpdateLoggers()
}

// UpdateLoggers initializes the logger package references
func UpdateLoggers() {
	LoggerServer = logging.InitPackageLogger(pkgServer)
	LoggerXds = logging.InitPackageLogger(pkgXds)
	LoggerXdsCallbacks = logging.InitPackageLogger(pkgXdsCallbacks)
	LoggerSync = logging.InitPackageLogger(pkgSync)
	LoggerMsg = logging.InitPackageLogger(pkgMsg)
	LoggerHealth = logging.InitPackageLogger(pkgHealth)
	LoggerAPIPartition = logging.InitPackageLogger(pkgAPIPartition)
	logrus.Info("Updated loggers")
}
