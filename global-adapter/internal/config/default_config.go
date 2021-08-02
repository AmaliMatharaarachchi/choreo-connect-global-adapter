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

// Configuration object which is populated with default values.
var defaultConfig = &Config{
	Server: server{
		Host:               "0.0.0.0",
		Port:               "9843",
		PartitionSize:      1000,
		PartitionThreshold: 0.9,
		Users: []User{
			{
				Username: "admin",
				Password: "admin",
			},
		},
	},
	Keystore: keystore{
		PrivateKeyLocation: "/home/wso2/security/keystore/mg.key",
		PublicKeyLocation:  "/home/wso2/security/keystore/mg.pem",
	},
	Truststore: truststore{
		Location: "/home/wso2/security/truststore",
	},
	DataBase: database{
		Name:            "choreo-mssql",
		Username:        "db_user",
		Password:        "$env{ga_db_pwd}",
		Host:            "mssql-db",
		Port:            1433,
		ValidationQuery: "select 1",
		PoolOptions: dbPool{
			MaxActive:          50,
			MaxWait:            60000,
			TestOnBorrow:       true,
			ValidationInterval: 30000,
			DefaultAutoCommit:  true,
		},
		OptionalMetadata: databaseOptionalMetadata{
			MaxRetryAttempts: 10,
		},
	},
	ControlPlane: controlPlane{
		ServiceURL:          "https://apim:9443/",
		Username:            "admin",
		Password:            "$env{cp_admin_pwd}",
		EnvironmentLabels:   []string{"Default"},
		RetryInterval:       5,
		SkipSSLVerification: true,
		JmsConnectionParameters: jmsConnectionParameters{
			EventListeningEndpoints: []string{"amqp://admin:$env{cp_admin_pwd}@apim:5672?retries='10'&connectdelay='30'"},
		},
	},
	RedisServer: redisServer{
		Host:               "choreo-dev-redis-cache.redis.cache.windows.net",
		Port:               6380,
		Password:           "$env{redis_host_pwd}",
		ClientName:         "global-adapter",
		DatabaseIndex:      2,
		ConnectionPoolSize: 3,
		OptionalMetadata: redisOptionalMetadata{
			MaxRetryAttempts: 10,
		},
	},
	XdsServer: xdsServer{
		Host: "0.0.0.0",
		Port: "18000",
	},
}
