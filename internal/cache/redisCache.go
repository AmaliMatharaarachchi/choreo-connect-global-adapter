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

	logger "github.com/sirupsen/logrus"

	"github.com/go-redis/redis"
)

// TODO : Following properties should fetch from config file
var redisHost string = "choreo-dev-redis-cache.redis.cache.windows.net"
var redisPort int = 6380
var redisPassword string = "kHpoShmMarFcm13E1LuDmDo5+A2rFyAqbG4zmiXPpyk="
var redisClientName = "global-adapter"
var databaseIndex = 2
var redisConnectionPoolSize = 10

var redisClient *redis.Client

func ConnectToRedisServer() *redis.Client {

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

	rdb := redis.NewClient(clientOptions)

	for {
		pong, err := rdb.Ping().Result()
		if err != nil {
			if strings.Contains(err.Error(), "timeout") {
				logger.Debug("Retring to connect with Redis server")
				continue
			} else {
				logger.Error("Failed to connect with redis server using Host : ", redisHost, " and Port : ", redisPort, " Error : ", err)
				break
			}
		} else {
			logger.Debug("Connected to the redis cluster ", pong)
			redisClient = rdb
			break
		}
	}

	return rdb
}

func GetClient() *redis.Client {
	if redisClient == nil {
		logger.Debug("Reconnecting redis client to server")
		ConnectToRedisServer()
		return redisClient
	} else {
		return redisClient
	}
}

func SetCacheKey(key, value *string, client *redis.Client, expiryTime time.Duration) {
	client.Set(*key, *value, expiryTime)
}

func RemoveCacheKey(key *string, client *redis.Client, expiryTime time.Duration) {
	client.Del(*key)
}
