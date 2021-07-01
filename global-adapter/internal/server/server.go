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
	"fmt"
	"net"
	"os"
	"os/signal"

	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/xds"
	ga_service "github.com/wso2/product-microgateway/adapter/pkg/discovery/api/wso2/discovery/service/ga"
	wso2_server "github.com/wso2/product-microgateway/adapter/pkg/discovery/protocol/server/v3"
	"google.golang.org/grpc"
)

// TODO: (VirajSalaka) check this is streams per connections or total number of concurrent streams.
const grpcMaxConcurrentStreams = 1000000

// Run functions starts the XDS Server.
func Run() {
	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	enforcerAPIDsSrv := wso2_server.NewServer(ctx, xds.GetAPICache(), &xds.Callbacks{})

	var grpcOptions []grpc.ServerOption
	grpcOptions = append(grpcOptions, grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams))

	// publicKeyLocation := "/home/wso2/security/mg.pem"
	// privateKeyLocation := "/home/wso2/security/mg.key"
	// cert, err := tlsutils.GetServerCertificate(publicKeyLocation, privateKeyLocation)
	// if err != nil {
	// 	fmt.Println("Error while loading certs")
	// } else {
	// 	grpcOptions = append(grpcOptions, grpc.Creds(
	// 		credentials.NewTLS(&tls.Config{
	// 			Certificates: []tls.Certificate{cert},
	// 		}),
	// 	))
	// }

	grpcServer := grpc.NewServer(grpcOptions...)
	ga_service.RegisterApiGADiscoveryServiceServer(grpcServer, enforcerAPIDsSrv)

	go xds.AddAPIsToCache()
	port := 18002
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		fmt.Println("Error while opening port.")
	}
	fmt.Println("server started.")
	if err = grpcServer.Serve(listener); err != nil {
		fmt.Println("Error while starting gRPC server.")
	}
	fmt.Println("server started.")

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
