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
 */

package server

import (
	"crypto/tls"
	"github.com/gorilla/mux"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/config"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"
	"github.com/wso2/product-microgateway/adapter/pkg/tlsutils"
	"net/http"
	"time"
)

// RunAPIServer function starts the REST API Server.
func RunAPIServer(conf *config.Config, router *mux.Router) {
	caCertPool := tlsutils.GetTrustedCertPool(conf.Truststore.Location)
	cert, _ := tlsutils.GetServerCertificate(conf.Keystore.PublicKeyLocation, conf.Keystore.PrivateKeyLocation)

	transport := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caCertPool,
	}

	logger.LoggerServer.Info("Starting GA API Server...")
	srv := &http.Server{
		Handler:      router,
		Addr:         conf.GAAPIServer.Host + ":" + conf.GAAPIServer.Port,
		WriteTimeout: 60 * time.Second,
		ReadTimeout:  60 * time.Second,
		TLSConfig:    transport,
	}

	logger.LoggerServer.Fatal(srv.ListenAndServe())
}
