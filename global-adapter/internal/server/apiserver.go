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

const internalAPIContextV1 = "/internal/data/v1/"

// Server is a wrapped http server with a http client
type Server struct {
	client *http.Client
}

// New function defines the new server structure with a http client.
func New(conf *config.Config) *Server {
	skipSSL := conf.ControlPlane.SkipSSLVerification
	requestTimeOut := conf.ControlPlane.HTTPClient.RequestTimeOut
	truststoreLocation := conf.Truststore.Location

	logger.LoggerSync.Debug("Skip SSL Verification:", skipSSL)
	tr := &http.Transport{}
	if !skipSSL {
		caCertPool := tlsutils.GetTrustedCertPool(truststoreLocation)
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: caCertPool},
			MaxConnsPerHost: conf.ControlPlane.MaxConnectionsPerHost,
		}
	} else {
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			MaxConnsPerHost: conf.ControlPlane.MaxConnectionsPerHost,
		}
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   requestTimeOut * time.Second,
	}

	srv := &Server{
		client: client,
	}
	return srv
}

// RunAPIServer function initializes the GA API server.
func (s *Server) RunAPIServer(conf *config.Config) {
	router := mux.NewRouter()
	router.HandleFunc(internalAPIContextV1+"apis/deployed-revisions", BasicAuth(s.HTTPatchHandler)).Methods(http.MethodPatch)
	router.HandleFunc(internalAPIContextV1+"apis/undeployed-revision", BasicAuth(s.HTTPPostHandler)).Methods(http.MethodPost)
	router.HandleFunc(internalAPIContextV1+"runtime-metadata", BasicAuth(s.HTTPGetHandler)).Methods(http.MethodGet)
	router.HandleFunc(internalAPIContextV1+"runtime-artifacts", BasicAuth(s.HTTPGetHandler)).Methods(http.MethodGet)
	router.HandleFunc(internalAPIContextV1+"retrieve-api-artifacts", BasicAuth(s.HTTPPostHandler)).Methods(http.MethodPost)
	router.HandleFunc(internalAPIContextV1+"keymanagers", BasicAuth(s.NoneOrgIDHTTPGetHandler)).Methods(http.MethodGet)
	router.HandleFunc(internalAPIContextV1+"revokedjwt", BasicAuth(s.NoneOrgIDHTTPGetHandler)).Methods(http.MethodGet)
	router.HandleFunc(internalAPIContextV1+"keyTemplates", BasicAuth(s.NoneOrgIDHTTPGetHandler)).Methods(http.MethodGet)
	router.HandleFunc(internalAPIContextV1+"block", BasicAuth(s.NoneOrgIDHTTPGetHandler)).Methods(http.MethodGet)
	router.HandleFunc(internalAPIContextV1+"subscriptions", BasicAuth(s.HTTPGetHandler)).Methods(http.MethodGet)
	router.HandleFunc(internalAPIContextV1+"applications", BasicAuth(s.HTTPGetHandler)).Methods(http.MethodGet)
	router.HandleFunc(internalAPIContextV1+"application-key-mappings", BasicAuth(s.HTTPGetHandler)).Methods(http.MethodGet)
	router.HandleFunc(internalAPIContextV1+"application-policies", BasicAuth(s.NoneOrgIDHTTPGetHandler)).Methods(http.MethodGet)
	router.HandleFunc(internalAPIContextV1+"subscription-policies", BasicAuth(s.NoneOrgIDHTTPGetHandler)).Methods(http.MethodGet)
	router.HandleFunc(internalAPIContextV1+"api-policies", BasicAuth(s.NoneOrgIDHTTPGetHandler)).Methods(http.MethodGet)
	router.HandleFunc(internalAPIContextV1+"global-policies", BasicAuth(s.NoneOrgIDHTTPGetHandler)).Methods(http.MethodGet)

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
