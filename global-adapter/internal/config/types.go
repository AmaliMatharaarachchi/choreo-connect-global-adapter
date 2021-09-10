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

package config

import (
	"time"
)

// Config represents the adapter configuration.
// It is created directly from the configuration toml file.
type Config struct {
	Server server
	// Keystore contains the keyFile and Cert File of the global adapter.
	Keystore keystore
	// Trusted Certificates.
	Truststore   truststore
	DataBase     database
	ControlPlane controlPlane
	RedisServer  redisServer
	XdsServer    xdsServer
}

// ControlPlane struct contains configurations related to the API Manager
type controlPlane struct {
	ServiceURL              string        `toml:"serviceUrl"`
	Username                string        `toml:"username"`
	Password                string        `toml:"password"`
	EnvironmentLabels       []string      `toml:"environmentLabels"`
	RetryInterval           time.Duration `toml:"retryInterval"`
	SkipSSLVerification     bool          `toml:"skipSSLVerification"`
	JmsConnectionParameters jmsConnectionParameters
	BrokerConnectionParameters brokerConnectionParameters `toml:"brokerConnectionParameters"`
}

type jmsConnectionParameters struct {
	EventListeningEndpoints []string `toml:"eventListeningEndpoints"`
}

type brokerConnectionParameters struct {
	EventListeningEndpoints []string        `toml:"eventListeningEndpoints"`
	ReconnectInterval      time.Duration `toml:"reconnectInterval"`
	ReconnectRetryCount    int           `toml:"reconnectRetryCount"`
}

type truststore struct {
	Location string
}

type keystore struct {
	PrivateKeyLocation string `toml:"keyPath"`
	PublicKeyLocation  string `toml:"certPath"`
}

// SecurityInfo contains the parameters of endpoint security
type SecurityInfo struct {
	Password         string `json:"password,omitempty"`
	CustomParameters string `json:"customparameters,omitempty"`
	SecurityType     string `json:"Type,omitempty"`
	Enabled          bool   `json:"enabled,omitempty"`
	Username         string `json:"username,omitempty"`
}

type server struct {
	Host               string
	Port               string
	PartitionSize      int
	PartitionThreshold float32
	Users              []User `toml:"users"`
}

// User represents registered GA Users.
type User struct {
	Username string
	Password string
}

type database struct {
	Name             string
	Username         string
	Password         string
	Host             string
	Port             int
	ValidationQuery  string
	PoolOptions      dbPool
	OptionalMetadata databaseOptionalMetadata
}

type dbPool struct {
	MaxActive          int
	MaxWait            int
	TestOnBorrow       bool
	ValidationInterval int
	DefaultAutoCommit  bool
}

type redisServer struct {
	Host               string
	Port               int
	Password           string
	ClientName         string
	DatabaseIndex      int
	ConnectionPoolSize int
	OptionalMetadata   redisOptionalMetadata
}

type databaseOptionalMetadata struct {
	MaxRetryAttempts int
}

type redisOptionalMetadata struct {
	MaxRetryAttempts int
}
type xdsServer struct {
	Host string
	Port string
}
