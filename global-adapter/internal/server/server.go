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

package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"

	// enable profiling endpoints
	_ "net/http/pprof"

	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/apipartition"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/config"
	healthGA "github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/health"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/messaging"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/synchronizer"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/xds"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/xds/callbacks"
	ga_service "github.com/wso2/product-microgateway/adapter/pkg/discovery/api/wso2/discovery/service/ga"
	wso2_server "github.com/wso2/product-microgateway/adapter/pkg/discovery/protocol/server/v3"
	"github.com/wso2/product-microgateway/adapter/pkg/health"
	healthservice "github.com/wso2/product-microgateway/adapter/pkg/health/api/wso2/health/service"
	"github.com/wso2/product-microgateway/adapter/pkg/tlsutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// TODO: (VirajSalaka) check this is streams per connections or total number of concurrent streams.
const grpcMaxConcurrentStreams = 1000000
const amqpProtocol = "amqp"

// Run functions starts the XDS Server.
func Run(conf *config.Config) {
	logger.LoggerServer.Info("Starting global adapter ....")
	// Checks grpc server health and Waits for grpc server
	go healthGA.WaitForGrpcServer()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Checks control plane health and Waits for Control plane
	go health.WaitForControlPlane()

	var connectionURLList = conf.ControlPlane.BrokerConnectionParameters.EventListeningEndpoints
	if strings.Contains(connectionURLList[0], amqpProtocol) {
		go messaging.ProcessEvents(conf)
	} else {
		messaging.InitiateAndProcessEvents(conf)
	}

	// Consume API events from channel.
	go apipartition.ProcessEventsInDatabase()
	// Consume non API events from channel.
	go GetNonAPIDeployAndRemoveEventsFromChannel()

	// Fetch APIs from control plane.
	synchronizer.FetchAPIsOnStartUp(conf, false)

	enforcerAPIDsSrv := wso2_server.NewServer(ctx, xds.GetAPICache(), &callbacks.Callbacks{})

	var grpcOptions []grpc.ServerOption
	grpcOptions = append(grpcOptions, grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams))

	publicKeyLocation := conf.Keystore.PublicKeyLocation
	privateKeyLocation := conf.Keystore.PrivateKeyLocation
	cert, err := tlsutils.GetServerCertificate(publicKeyLocation, privateKeyLocation)
	caCertPool := tlsutils.GetTrustedCertPool(conf.Truststore.Location)
	if err != nil {
		logger.LoggerServer.Fatal("Error while loading private key public key pair.", err)
	} else {
		grpcOptions = append(grpcOptions, grpc.Creds(
			credentials.NewTLS(&tls.Config{
				Certificates: []tls.Certificate{cert},
				ClientAuth:   tls.RequireAndVerifyClientCert,
				ClientCAs:    caCertPool,
			}),
		))
	}

	grpcServer := grpc.NewServer(grpcOptions...)
	ga_service.RegisterApiGADiscoveryServiceServer(grpcServer, enforcerAPIDsSrv)

	port := conf.XdsServer.Port
	// TODO: (VirajSalaka) Bind host to the listener
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))

	if err != nil {
		logger.LoggerServer.Fatalf("Error while listening on port: %s", port)
	}

	//to enable pprof endpoint
	go func() {
		logger.LoggerServer.Debug("Opening port 6060 for pprof")
		http.ListenAndServe("localhost:6060", nil)
	}()

	// register health service
	healthservice.RegisterHealthServer(grpcServer, &health.Server{})
	logger.LoggerServer.Info("XDS server is starting.")
	// Set the Grpc server health status
	healthGA.SetGrpcServerStatus(true)
	if err = grpcServer.Serve(listener); err != nil {
		// Set the Grpc server health status to false
		healthGA.SetGrpcServerStatus(false)
		logger.LoggerServer.Fatal("Error while starting gRPC server.")
	}

OUTER:
	for {
		select {
		case s := <-sig:
			switch s {
			case os.Interrupt:
				break OUTER
			}
		}
	}
}

// GetNonAPIDeployAndRemoveEventsFromChannel consume non API deploy/remove events from the channel.
func GetNonAPIDeployAndRemoveEventsFromChannel() {
	for d := range messaging.NonAPIDeployAndRemoveEventChannel {
		logger.LoggerServer.Infof("Non api event %s ", d)
	}
}
