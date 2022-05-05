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

package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/config"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// HTTPGetHandler function handles get requests
func (s *Server) HTTPGetHandler(w http.ResponseWriter, r *http.Request) {
	buildRequestWithOrganizationID(s.client, w, r, http.MethodGet, nil)
}

// HTTPPostHandler function handles post requests
func (s *Server) HTTPPostHandler(w http.ResponseWriter, r *http.Request) {
	buildRequestWithOrganizationID(s.client, w, r, http.MethodPost, r.Body)
}

// HTTPatchHandler function handles patch requests
func (s *Server) HTTPatchHandler(w http.ResponseWriter, r *http.Request) {
	buildRequestWithOrganizationID(s.client, w, r, http.MethodPatch, r.Body)
}

// NoneOrgIDHTTPGetHandler function handles get requests
func (s *Server) NoneOrgIDHTTPGetHandler(w http.ResponseWriter, r *http.Request) {
	buildRequestWithoutOrganizationID(s.client, w, r, http.MethodGet, nil)
}

func buildRequestWithOrganizationID(client *http.Client, w http.ResponseWriter, r *http.Request, httpMethod string, body io.Reader) {
	conf := config.ReadConfigs()
	var requestURL = conf.ControlPlane.ServiceURL + r.RequestURI
	if conf.PrivateDataPlane.Enabled {
		if isOrganizationIDExists(r) {
			requestURL = replaceURLParameter(requestURL, "organizationId", conf.PrivateDataPlane.OrganizationID)
		} else if conf.PrivateDataPlane.OrganizationID != "" {
			if strings.Contains(requestURL, "?") {
				requestURL += "&organizationId=" + conf.PrivateDataPlane.OrganizationID
			} else {
				requestURL += "?organizationId=" + conf.PrivateDataPlane.OrganizationID
			}
		}
	}
	sendHTTPRequest(client, conf, requestURL, w, httpMethod, body)
}

func buildRequestWithoutOrganizationID(client *http.Client, w http.ResponseWriter, r *http.Request, httpMethod string, body io.Reader) {
	conf := config.ReadConfigs()
	var requestURL = conf.ControlPlane.ServiceURL + r.RequestURI
	if conf.PrivateDataPlane.Enabled {
		if isOrganizationIDExists(r) {
			requestURL = replaceURLParameter(requestURL, "organizationId", "ALL")
		}
	}
	sendHTTPRequest(client, conf, requestURL, w, httpMethod, body)
}

func sendHTTPRequest(client *http.Client, conf *config.Config, requestURL string, w http.ResponseWriter, httpMethod string, body io.Reader) {
	logger.LoggerServer.Debug("Request URL for control plane: ", requestURL)
	req, err := http.NewRequest(httpMethod, requestURL, body)

	if err != nil {
		w.WriteHeader(500)
		logger.LoggerServer.Error("Error occurred while constructing http request to control plane ", err)
		return
	}
	req.SetBasicAuth(conf.ControlPlane.Username, conf.ControlPlane.Password)

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)

	if err != nil {
		w.WriteHeader(500)
		logger.LoggerServer.Error("Error occurred while connecting to Control Plane ", err)
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

func isOrganizationIDExists(r *http.Request) bool {
	r.ParseForm()
	_, ok := r.Form["organizationId"]
	if !ok {
		return false
	}
	return true
}

func replaceURLParameter(requestURL string, queryParamKey string, QueryParamValue string) string {
	logger.LoggerServer.Debugf("Replacing %s query parameter with %s: ", queryParamKey, QueryParamValue)
	newURL, _ := url.Parse(requestURL)
	values, _ := url.ParseQuery(newURL.RawQuery)
	values.Set(queryParamKey, QueryParamValue)
	newURL.RawQuery = values.Encode()
	return newURL.String()
}
