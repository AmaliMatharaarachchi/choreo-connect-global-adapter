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
 */

package cache

import (
	"crypto/tls"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"

	"github.com/go-redis/redis"
)

// TODO : Following properties should fetch from config file
var redisHost string = "choreo-dev-redis-cache.redis.cache.windows.net"
var redisPort int = 6380
var redisPassword string = ""
var redisClientName = "global-adapter"
var databaseIndex = 2
var redisConnectionPoolSize = 3
var maxRetryCount int = 20

var redisClient *redis.Client

// ConnectToRedisServer - for connect redis client with the redis server
func ConnectToRedisServer() *redis.Client {
	clientOptions := getRedisClientOptions()
	rdb := redis.NewClient(clientOptions)

	pong, err := rdb.Ping().Result()
	if err != nil {
		if strings.Contains(err.Error(), "timeout") {
			logger.LoggerServer.Info(err, " .Retring to connect with Redis server")
			redisClient = nil
		} else {
			logger.LoggerServer.Error("Failed to connect with redis server using Host : ", redisHost, " and Port : ", redisPort, " Error : ", err)
			redisClient = nil
		}
		redisClient = nil
	} else {
		logger.LoggerServer.Info("Connected to the redis cluster ", pong)
		redisClient = rdb
	}

	return rdb
}

func getRedisClientOptions() *redis.Options {
	return &redis.Options{
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
		PoolSize:   redisConnectionPoolSize,
		MaxRetries: maxRetryCount,
	}
}

// GetClient - returns the connected client reference if it alive , else reconnect and return the reference
func GetClient() *redis.Client {
	if redisClient == nil {
		logger.LoggerServer.Debug("Reconnecting redis client to server")
		return ConnectToRedisServer()
	}
	return redisClient
}

// SetCacheKey - update the cache
func SetCacheKey(key, value *string, client *redis.Client, expiryTime time.Duration) {
	client.Set(*key, *value, expiryTime)
}

// RemoveCacheKey - delete specified keys from cache
func RemoveCacheKey(key *string, client *redis.Client, expiryTime time.Duration) {
	client.Del(*key)
}
