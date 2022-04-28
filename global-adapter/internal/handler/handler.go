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
	"errors"
	"fmt"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/config"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
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
	var requestURL = conf.ControlPlane.ServiceURL + r.RequestURI
	if isOrganizationIDExists(r) {
		requestURL = replaceURLParameter(requestURL, "organizationId", conf.PrivateDataPlane.OrganizationID)
	} else if conf.PrivateDataPlane.OrganizationID != "" {
		requestURL += "&organizationId=" + conf.PrivateDataPlane.OrganizationID
	}

	req, err := http.NewRequest(httpMethod, requestURL, body)

	if err != nil {
		w.WriteHeader(500)
		logger.LoggerServer.Error("Error occurred while constructing http request to control plane ", err)
		return
	}
	req.SetBasicAuth(conf.ControlPlane.Username, conf.ControlPlane.Password)

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	c := make(chan SyncAPIResponse)

	workerReq := workerRequest{
		Req:                *req,
		SyncAPIRespChannel: c,
	}

	if workerPool == nil {
		logger.LoggerSync.Fatal("WorkerPool is not initiated due to an internal error.")
	}
	// If adding task to the pool cannot be done, the whole thread hangs here.
	workerPool.Enqueue(workerReq)

	data := <-c

	if data.ResponseCode == http.StatusOK {
		bodyString := string(data.Resp)
		fmt.Fprintf(w, "%s", bodyString)
	} else {
		bodyString := string(data.Resp)
		w.WriteHeader(data.ResponseCode)
		fmt.Fprintf(w, "%s", bodyString)
	}
}

// SendRequestToControlPlane is the function triggered to send the request to the control plane.
// It returns true if a response is received from the api manager.
func SendRequestToControlPlane(req *http.Request, c chan SyncAPIResponse, client *http.Client) bool {
	// Make the request
	logger.LoggerSync.Debug("Sending the control plane request")
	resp, err := client.Do(req)

	response := SyncAPIResponse{}

	// In the event of a connection error, the error would not be nil, then return the error
	// If the error is not null, proceed
	if err != nil {
		logger.LoggerSync.Errorf("Error occurred while retrieving APIs from API manager: %v", err)
		response.Err = err
		response.Resp = nil
		c <- response
		return false
	}

	// get the response in the form of a byte slice
	respBytes, err := ioutil.ReadAll(resp.Body)

	// If the reading response gives an error
	if err != nil {
		logger.LoggerSync.Errorf("Error occurred while reading the response: %v", err)
		response.Err = err
		response.ResponseCode = resp.StatusCode
		response.Resp = nil
		c <- response
		return false
	}

	// For successful response, return the byte slice and nil as error
	if resp.StatusCode == http.StatusOK {
		response.Err = nil
		response.ResponseCode = resp.StatusCode
		response.Resp = respBytes
		c <- response
		return true
	}

	// If the response is not successful, create a new error with the response and log it and return
	// Ex: for 401 scenarios, 403 scenarios.
	logger.LoggerSync.Errorf("Failure response: %v", string(respBytes))
	response.Err = errors.New(string(respBytes))
	response.Resp = respBytes
	response.ResponseCode = resp.StatusCode
	c <- response
	return true
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
	keys, ok := r.Form["organizationId"]
	if !ok || len(keys[0]) < 1 {
		return false
	}
	return true
}

func replaceURLParameter(requestURL string, oldValue string, newValue string) string {
	newURL, _ := url.Parse(requestURL)
	values, _ := url.ParseQuery(newURL.RawQuery)
	values.Set(oldValue, newValue)
	newURL.RawQuery = values.Encode()
	return newURL.String()
}
