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

/*
	Package handler contains logic related to authenticate the GA API requests and handle API requests accordingly.
*/

package handler

import (
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"fmt"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/config"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"
	"github.com/wso2/product-microgateway/adapter/pkg/tlsutils"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

// HTTPGetHandler function handles get requests
func HTTPGetHandler(w http.ResponseWriter, r *http.Request) {
	buildRequest(w, r, http.MethodGet, nil)
}

// HTTPPostHandler function handles post requests
func HTTPPostHandler(w http.ResponseWriter, r *http.Request) {
	buildRequest(w, r, http.MethodPost, r.Body)
}

// HTTPatchHandler function handles patch requests
func HTTPatchHandler(w http.ResponseWriter, r *http.Request) {
	buildRequest(w, r, http.MethodPatch, r.Body)
}

func buildRequest(w http.ResponseWriter, r *http.Request, httpMethod string, body io.Reader) {
	conf := config.ReadConfigs()
	skipSSL := conf.ControlPlane.SkipSSLVerification
	requestTimeOut := conf.ControlPlane.HTTPClient.RequestTimeOut
	truststoreLocation := conf.Truststore.Location

	logger.LoggerSync.Debugf("Skip SSL Verification: %v", skipSSL)
	tr := &http.Transport{}
	if !skipSSL {
		caCertPool := tlsutils.GetTrustedCertPool(truststoreLocation)
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: caCertPool},
		}
	} else {
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   requestTimeOut * time.Second,
	}

	var url = conf.ControlPlane.ServiceURL + r.RequestURI
	if conf.FeatureType.OrganizationID != "" {
		url += "&organizationId=" + conf.FeatureType.OrganizationID
	}

	req, err := http.NewRequest(httpMethod, url, body)
	req.SetBasicAuth(conf.ControlPlane.Username, conf.ControlPlane.Password)

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)

	if err != nil {
		w.WriteHeader(500)
		logger.LoggerServer.Error("Error occurred while connecting to API Manager ", err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		w.WriteHeader(resp.StatusCode)
		fmt.Fprintf(w, "%s", bodyString)
	}

	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	bodyString := string(bodyBytes)
	fmt.Fprintf(w, "%s", bodyString)
}

// BasicAuth function authenticates the request.
func BasicAuth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		conf := config.ReadConfigs()
		if ok {
			usernameHash := sha256.Sum256([]byte(username))
			passwordHash := sha256.Sum256([]byte(password))
			expectedUsernameHash := sha256.Sum256([]byte(conf.GAAPIServer.Username))
			expectedPasswordHash := sha256.Sum256([]byte(conf.GAAPIServer.Password))

			usernameMatch := (subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1)
			passwordMatch := (subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1)

			if usernameMatch && passwordMatch {
				next.ServeHTTP(w, r)
				return
			}
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized Access", http.StatusUnauthorized)
	})
}
