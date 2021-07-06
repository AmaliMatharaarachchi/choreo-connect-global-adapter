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

package health

import (
	"crypto/tls"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
)

var (
	redisCacheConnectionStatusChan  = make(chan bool)
	redisCacheConnectionEstablished = false
)

// SetRedisCacheConnectionStatus sets the given status to the internal channel redisCacheConnectionStatusChan
func SetRedisCacheConnectionStatus(status bool) {
	// check for Redis cache Connection Established, to non block call
	// if called again (somehow) after startup, for extra safe check this value
	if !redisCacheConnectionEstablished {
		redisCacheConnectionStatusChan <- status
	}
}

// WaitForRedisCacheConnection sleep the current go routine until connected to redis cache
func WaitForRedisCacheConnection() {
	redisConnected := false
	if !redisConnected {
		redisConnected = <-databaseConnectionStatusChan
	}
	redisCacheConnectionEstablished = true
}

// RedisCacheConnectRetry retries to connect to the redis cache if there is a connection error
func RedisCacheConnectRetry(connString string) (*redis.Client, error) {
	// TODO: get these values from config.toml
	var redisHost string = "choreo-dev-redis-cache.redis.cache.windows.net"
	var redisPort int = 6380
	var redisPassword string = "kHpoShmMarFcm13E1LuDmDo5+A2rFyAqbG4zmiXPpyk="
	var redisClientName = "global-adapter"
	var databaseIndex = 2
	var redisConnectionPoolSize = 10
	// TODO: (Jayanie) maxAttempt and retryInterval Should be configurable?
	var (
		maxAttempt    int = 5
		retryInterval time.Duration
		attempt       int
		err           error
	)
	clientOptions := &redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, strconv.Itoa(redisPort)),
		Password: redisPassword,
		DB:       databaseIndex,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		OnConnect: func(c *redis.Conn) error {
			name := redisClientName
			return c.ClientSetName(name).Err()
		},
		PoolSize: redisConnectionPoolSize,
	}

	for attempt = 1; attempt <= maxAttempt; attempt++ {
		rdb := redis.NewClient(clientOptions)
		_, err := rdb.Ping().Result()
		if err == nil {
			return rdb, nil
		}
		if err != nil {
			if strings.Contains(err.Error(), "timeout") {
				time.Sleep(retryInterval * time.Second)
			} else {
				return nil, err
			}
		}
	}
	return nil, err
}
