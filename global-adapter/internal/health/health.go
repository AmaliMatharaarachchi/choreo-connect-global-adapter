/*
 *  Copyright (c) 2022, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
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

package health

import (
	"context"
	"sync"

	logger "github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"
	healthservice "github.com/wso2/product-microgateway/adapter/pkg/health/api/wso2/health/service"
)

var (
	serviceHealthStatus = make(map[string]bool)
	healthStatuses      = map[bool]string{
		true:  "HEALTHY",
		false: "UNHEALTHY",
	}
	mutexForHealthUpdate sync.Mutex
)

// Criterias for set health states
const (
	DBConnection criteria = "ga.db.con"
	GRPCService  criteria = "ga.grpc"
	Startup      criteria = "ga.startup"
)

type criteria string

// SetStatus sets the health state of the service
func (c criteria) SetStatus(isHealthy bool) {
	mutexForHealthUpdate.Lock()
	defer mutexForHealthUpdate.Unlock()
	logger.LoggerHealth.Infof("Update health status of service \"%s\" as %s", c, healthStatuses[isHealthy])
	serviceHealthStatus[string(c)] = isHealthy
}

// Server represents the Health GRPC server
type Server struct {
	healthservice.UnimplementedHealthServer
}

// Check responds the health check client with health status of the Adapter
func (s Server) Check(ctx context.Context, request *healthservice.HealthCheckRequest) (*healthservice.HealthCheckResponse, error) {
	logger.LoggerHealth.Debugf("Querying health state for Global Adapter.. service:%q", request.Service)
	logger.LoggerHealth.Debugf("Internal health state map: %v", serviceHealthStatus)

	if request.Service == "" {
		// overall health of the server
		isHealthy := true
		for _, ok := range serviceHealthStatus {
			isHealthy = isHealthy && ok
		}

		if isHealthy {
			logger.LoggerHealth.Debug("Responding health state of GA as HEALTHY")
			return &healthservice.HealthCheckResponse{Status: healthservice.HealthCheckResponse_SERVING}, nil
		}
		logger.LoggerHealth.Debug("Responding health state of GA as NOT_HEALTHY")
		return &healthservice.HealthCheckResponse{Status: healthservice.HealthCheckResponse_NOT_SERVING}, nil
	}

	if request.Service == "readiness" {
		if isHealthy, ok := serviceHealthStatus[string(Startup)]; ok {
			if isHealthy {
				logger.LoggerHealth.Debugf("Responding health state of GA service \"%s\" as HEALTHY", request.Service)
				return &healthservice.HealthCheckResponse{Status: healthservice.HealthCheckResponse_SERVING}, nil
			}
			logger.LoggerHealth.Debugf("Responding health state of GA service \"%s\" as NOT_HEALTHY", request.Service)
			return &healthservice.HealthCheckResponse{Status: healthservice.HealthCheckResponse_NOT_SERVING}, nil
		}
	}

	if request.Service == "liveness" {
		if st1, ok1 := serviceHealthStatus[string(DBConnection)]; ok1 {
			if st2, ok2 := serviceHealthStatus[string(GRPCService)]; ok2 {
				if st1 && st2 {
					logger.LoggerHealth.Debugf("Responding health state of GA service \"%s\" as HEALTHY", request.Service)
					return &healthservice.HealthCheckResponse{Status: healthservice.HealthCheckResponse_SERVING}, nil
				}
				logger.LoggerHealth.Debugf("Responding health state of GA service \"%s\" as NOT_HEALTHY", request.Service)
				return &healthservice.HealthCheckResponse{Status: healthservice.HealthCheckResponse_NOT_SERVING}, nil
			}
		}
	}

	// no component found
	logger.LoggerHealth.Debugf("Responding health state of GA service \"%s\" as UNKNOWN", request.Service)
	return &healthservice.HealthCheckResponse{Status: healthservice.HealthCheckResponse_UNKNOWN}, nil
}
