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

	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/config"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/health"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter/internal/logger"

	"github.com/go-redis/redis"
)

var redisClient *redis.Client

// ConnectToRedisServer - for connect redis client with the redis server
func ConnectToRedisServer() *redis.Client {
	conf := config.ReadConfigs()
	clientOptions := getRedisClientOptions(conf)
	rdb := redis.NewClient(clientOptions)
	var isConnected bool = false
	pong, err := rdb.Ping().Result()
	// Check the connection error and Retry
	if err != nil {
		redisClient, isConnected = health.RedisCacheConnectRetry(clientOptions)
	} else {
		logger.LoggerServer.Info("Connected to the redis cluster ", pong)
		isConnected = true
		redisClient = rdb
	}
	health.SetRedisCacheConnectionStatus(isConnected)
	return rdb
}

func getRedisClientOptions(conf *config.Config) *redis.Options {
	return &redis.Options{
		Addr:     fmt.Sprintf("%s:%s", conf.RedisServer.Host, strconv.Itoa(conf.RedisServer.Port)),
		Password: conf.RedisServer.Password,
		DB:       conf.RedisServer.DatabaseIndex,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		OnConnect: func(c *redis.Conn) error {
			name := conf.RedisServer.ClientName
			return c.ClientSetName(name).Err()
		},
		PoolSize:   conf.RedisServer.ConnectionPoolSize,
		MaxRetries: conf.RedisServer.OptionalMetadata.MaxRetryAttempts,
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

// RemoveCacheKey - delete specified keys from cache
func RemoveCacheKey(key string, client *redis.Client) {
	res := client.Del(key)
	if res.Err() != nil {
		logger.LoggerServer.Errorf("Error while deleting the cache object | key : %s , %v", key, res.Err().Error())
	} else {
		logger.LoggerServer.Debug("Cache object removed for key : ", key)
	}
}

// SetCacheKeys - Insert new cache key-values, updaye existing values.
func SetCacheKeys(cacheList []string, client *redis.Client) error {
	res := client.MSet(cacheList)
	if res.Err() != nil {
		logger.LoggerServer.Error("Error while adding the cache object(s) ", res.Err().Error())
	} else {
		logger.LoggerServer.Debugf("Cache updated , total key-values : %d", len(cacheList)/2)
		return nil
	}
	return res.Err()
}

// PublishRedisEvent - Publish an event to Redis Server
func PublishRedisEvent(channel string, client *redis.Client, event string) {
	res := client.Publish(channel, event)
	if res.Err() != nil {
		logger.LoggerServer.Errorf("Error while publishing to redis channel: %s with event: %s, %v", channel, event, res.Err().Error())
	} else {
		logger.LoggerServer.Debug("Published redis event : ", channel)
	}
}
