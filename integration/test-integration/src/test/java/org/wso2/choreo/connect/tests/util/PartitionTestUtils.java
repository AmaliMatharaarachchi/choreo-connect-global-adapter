/*
 * Copyright (c) 2021, WSO2 Inc. (http://www.wso2.org).
 *
 * WSO2 Inc. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package org.wso2.choreo.connect.tests.util;

import com.github.dockerjava.zerodep.shaded.org.apache.hc.core5.http.HttpStatus;
import org.testng.Assert;
import org.wso2.choreo.connect.tests.common.model.PartitionTestEntry;
import org.wso2.choreo.connect.tests.context.CCTestException;
import redis.clients.jedis.DefaultJedisClientConfig;
import redis.clients.jedis.Jedis;
import redis.clients.jedis.JedisClientConfig;

import java.net.URI;
import java.net.URISyntaxException;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Utility methods used for partitioning related Testcases.
 */
public class PartitionTestUtils {
    public static final Map<String, String> PARTITION_ENDPOINT_MAP;
    public static final String PARTITION_1 = "default-p1";
    public static final String PARTITION_2 = "default-p2";

    static {
        Map<String, String> map = new HashMap<>();
        map.put(PARTITION_1, "https://localhost:9095/");
        map.put(PARTITION_2, "https://localhost:9096/");
        PARTITION_ENDPOINT_MAP = map;
    }

    // Creates the dedis connection
    public static Jedis createJedisConnection() {
        JedisClientConfig config = DefaultJedisClientConfig.builder().database(0).clientName("global-adapter")
                .socketTimeoutMillis(300000).hostnameVerifier((hostname, session) -> true).build();
        try (Jedis jedis = new Jedis(new URI("rediss://localhost:6379"), config)) {
            jedis.auth("admin");
            return jedis;
        } catch (URISyntaxException e) {
            e.printStackTrace();
        }
        return null;
    }

    private static void checkRedisEntry(Jedis jedis, String apiContext, String apiVersion,
                                 String expectedValue) {
        String value = getRedisEntry(jedis, apiContext, apiVersion);
        Assert.assertEquals(value, expectedValue, " Mismatch found while reading redis entry for " +
                String.format("global-adapter#default#/%s/%s", apiContext, apiVersion));
    }

    public static String getRedisEntry(Jedis jedis, String apiContext, String apiVersion) {
        String value = jedis.get(String.format("global-adapter#default#/%s/%s", apiContext, apiVersion));
        return value;
    }

    public static void checkTestEntry(Jedis jedis, PartitionTestEntry testEntry, Map<String,String> headers,
                                boolean verifyInOtherRouter) throws Exception {
        String expectedPartition = String.format("%s/%s/%s", testEntry.getPartition(), testEntry.getApiContext(),
                testEntry.getApiVersion());
        checkRedisEntry(jedis, testEntry.getApiContext(), testEntry.getApiVersion(), expectedPartition);
        for (Map.Entry<String, String> mapEntry : PARTITION_ENDPOINT_MAP.entrySet()) {
            String url = mapEntry.getValue() + testEntry.getApiContext() + "/"
                    + testEntry.getApiVersion() + testEntry.getResourcePath();
            if (testEntry.getPartition().equals(mapEntry.getKey())) {
                assert200Response(url, headers);
            } else if (verifyInOtherRouter){
                assert404Response(url, headers);
            }
        }
    }

    public static void assert200Response(String url, Map<String, String> headers) throws CCTestException {
        HttpResponse response = HttpsClientRequest.retryGetRequestUntilDeployed(url, headers);
        Assert.assertNotNull(response, "Error occurred while invoking the endpoint " + url + " HttpResponse ");
        Assert.assertEquals(HttpStatus.SC_SUCCESS, response.getResponseCode(),
                "Status code mismatched. Endpoint:" + url + " HttpResponse ");
    }

    public static void assert404Response(String url, Map<String, String> headers) throws CCTestException {
        HttpResponse response = HttpsClientRequest.doGet(url, headers);
        Assert.assertNotNull(response, "Error occurred while invoking: " + url + " HttpResponse ");
        Assert.assertEquals(HttpStatus.SC_NOT_FOUND, response.getResponseCode(),
                "Status code mismatched. Endpoint:" + url + " HttpResponse ");
    }

    public static void addTestEntryToList(List<PartitionTestEntry> testEntryList, String apiName, String apiVersion,
                                    String apiContext, String partition) {
        PartitionTestEntry testEntry = new PartitionTestEntry();
        testEntry.setApiName(apiName);
        testEntry.setApiVersion(apiVersion);
        testEntry.setPartition(partition);
        testEntry.setApiContext(apiContext);
        testEntryList.add(testEntry);
    }
}
