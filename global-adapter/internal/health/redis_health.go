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
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/config"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"
)

var (
	redisCacheConnectionStatusChan  = make(chan bool)
	redisCacheConnectionEstablished = false
)

// SetRedisCacheConnectionStatus sets the given status to the internal channel redisCacheConnectionStatusChan
func SetRedisCacheConnectionStatus(status bool) {
	// Check for Redis cache Connection Established, to non block call
	// if called again (somehow) after startup, for extra safe check this value
	if !redisCacheConnectionEstablished {
		redisCacheConnectionStatusChan <- status
	}
}

// WaitForRedisCacheConnection until connected to redis cache
func WaitForRedisCacheConnection() {
	conf := config.ReadConfigs()
	redisConnected := false
	for !redisConnected {
		redisConnected = <-redisCacheConnectionStatusChan
		logger.LoggerHealth.Debugf("Connection status to the Redis cache at %v:%v returned: %v",
			conf.RedisServer.Host, conf.RedisServer.Port, redisConnected)
	}
	redisCacheConnectionEstablished = true
}

// RedisCacheConnectRetry retries to connect to the redis cache if there is a connection error
func RedisCacheConnectRetry(clientOptions *redis.Options) (*redis.Client, bool) {
	conf := config.ReadConfigs()
	maxAttempts := conf.RedisServer.OptionalMetadata.MaxRetryAttempts
	var (
		retryInterval time.Duration = 5
		attempt       int
	)
	for attempt = 1; attempt <= maxAttempts; attempt++ {
		rdb := redis.NewClient(clientOptions)
		_, err := rdb.Ping().Result()
		logger.LoggerHealth.Info(err)
		if err != nil {
			if strings.Contains(err.Error(), "timeout") {
				time.Sleep(retryInterval * time.Second)
			} else {
				return nil, false
			}
		} else {
			return rdb, true
		}
	}
	return nil, false
}
