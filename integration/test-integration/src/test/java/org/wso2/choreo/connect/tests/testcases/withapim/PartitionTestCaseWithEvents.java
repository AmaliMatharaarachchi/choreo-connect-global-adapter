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

package org.wso2.choreo.connect.tests.testcases.withapim;

import com.github.dockerjava.zerodep.shaded.org.apache.hc.core5.http.HttpStatus;
import com.google.common.net.HttpHeaders;
import org.testng.Assert;
import org.testng.annotations.BeforeClass;
import org.testng.annotations.Test;
import org.wso2.am.integration.test.utils.bean.APIRequest;
import org.wso2.choreo.connect.tests.apim.ApimBaseTest;
import org.wso2.choreo.connect.tests.apim.dto.AppWithConsumerKey;
import org.wso2.choreo.connect.tests.apim.dto.Application;
import org.wso2.choreo.connect.tests.apim.utils.PublisherUtils;
import org.wso2.choreo.connect.tests.apim.utils.StoreUtils;
import org.wso2.choreo.connect.tests.context.CCTestException;
import org.wso2.choreo.connect.tests.util.HttpResponse;
import org.wso2.choreo.connect.tests.util.HttpsClientRequest;
import org.wso2.choreo.connect.tests.util.TestConstant;
import org.wso2.choreo.connect.tests.util.Utils;
import redis.clients.jedis.DefaultJedisClientConfig;
import redis.clients.jedis.Jedis;
import redis.clients.jedis.JedisClientConfig;

import java.net.URI;
import java.net.URISyntaxException;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class PartitionTestCaseWithEvents extends ApimBaseTest {
    private static final String APP_NAME = "GlobalAdapterEventTest";
    private String applicationId;
    Map <String, String> headers;
    List<TestEntry> testEntryList;
    private Map<String, String> partitionEndpointMap;
    private Jedis jedis;

    @BeforeClass(alwaysRun = true, description = "initialize setup")
    void setup() throws Exception {
        super.initWithSuperTenant();
        // Populate the API Entries which are going to be added in the middle.
        initializeEntryMap();
        // Initialize Jedis (Redis_ Client
        jedis = createJedisConnection();
        // Populate partition against endpoint.
        populationPartitionEndpointMap();
    }

    @Test
    public void testCreateEvents() throws Exception {
        //Create, deploy and publish API - only to specifically test events.
        // For other testcases the json is used to create APIs, Apps, Subscriptions
        //Create App. Subscribe.
        Application app = new Application(APP_NAME, TestConstant.APPLICATION_TIER.UNLIMITED);
        AppWithConsumerKey appWithConsumerKey = StoreUtils.createApplicationWithKeys(app, storeRestClient);
        applicationId = appWithConsumerKey.getApplicationId();

        for (TestEntry entry : testEntryList) {
            APIRequest apiRequest = PublisherUtils.createSampleAPIRequest(
                    entry.getApiName(), entry.getApiContext(), entry.getApiVersion(), user.getUserName());
            String apiId = PublisherUtils.createAndPublishAPI(apiRequest, publisherRestClient);
            entry.setApiID(apiId);
            StoreUtils.subscribeToAPI(apiId, applicationId, TestConstant.SUBSCRIPTION_TIER.UNLIMITED, storeRestClient);
        }

        Utils.delay(TestConstant.DEPLOYMENT_WAIT_TIME, "Interrupted when waiting for the " +
                "subscriptions to be deployed");
        String accessToken = StoreUtils.generateUserAccessToken(apimServiceURLHttps,
                appWithConsumerKey.getConsumerKey(), appWithConsumerKey.getConsumerSecret(),
                new String[]{}, user, storeRestClient);

        headers = new HashMap<>();
        headers.put(HttpHeaders.AUTHORIZATION, "Bearer " + accessToken);

        //Invoke all the added API
        for (TestEntry testEntry : testEntryList) {

            // Checks against both the router partitions available.
            // If the partition matches, it should return 200 OK, otherwise 404.
            checkTestEntry(testEntry);
        }
    }

    @Test
    public void testDeleteAndAddTwoAPIs() throws Exception {
        int testEntryListSize = testEntryList.size();
        // first delete the 1st API and the last API in list (which belongs to two partitions)
        // now there is a vacant index in each partition.
        TestEntry testEntryFirst = testEntryList.get(0);
        TestEntry testEntryLast = testEntryList.get(testEntryListSize - 1);

        // TODO: (VirajSalaka) Use Undeploy instead
        testEntryFirst = deleteTestEntry(testEntryFirst);
        String testEntryFirstPartition = testEntryFirst.getPartition();
        testEntryLast = deleteTestEntry(testEntryLast);
        String testEntryLastPartition = testEntryLast.getPartition();

        // Since the order is going to be changed, the partitions should be swapped.
        testEntryFirst.setPartition(testEntryLastPartition);
        testEntryLast.setPartition(testEntryFirstPartition);

        // change the order and redeploy.
        APIRequest apiRequest = PublisherUtils.createSampleAPIRequest(
                testEntryLast.getApiName(), testEntryLast.getApiContext(), testEntryLast.getApiVersion(),
                user.getUserName());
        String apiId = PublisherUtils.createAndPublishAPI(apiRequest, publisherRestClient);
        StoreUtils.subscribeToAPI(apiId, applicationId, TestConstant.SUBSCRIPTION_TIER.UNLIMITED, storeRestClient);
        testEntryLast.setApiID(apiId);
        Utils.delay(TestConstant.DEPLOYMENT_WAIT_TIME, "Interrupted while waiting to subscription delete event");


        apiRequest = PublisherUtils.createSampleAPIRequest(
                testEntryFirst.getApiName(), testEntryFirst.getApiContext(), testEntryFirst.getApiVersion(),
                user.getUserName());
        apiId = PublisherUtils.createAndPublishAPI(apiRequest, publisherRestClient);
        testEntryFirst.setApiID(apiId);
        StoreUtils.subscribeToAPI(apiId, applicationId, TestConstant.SUBSCRIPTION_TIER.UNLIMITED, storeRestClient);
        Utils.delay(TestConstant.DEPLOYMENT_WAIT_TIME, "Interrupted while waiting to subscription delete event");

        checkTestEntry(testEntryFirst);
        checkTestEntry(testEntryLast);
    }

    @Test
    public void testDeleteEvents() throws Exception{
        StoreUtils.removeAllSubscriptionsForAnApp(applicationId, storeRestClient);
        Utils.delay(TestConstant.DEPLOYMENT_WAIT_TIME, "Interrupted while waiting to subscription delete event");

        for (TestEntry testEntry : testEntryList) {
            String endpoint = partitionEndpointMap.get(testEntry.getPartition());
            String invocationUrl = endpoint + testEntry.getApiContext() + "/" + testEntry.getApiVersion()
                    + testEntry.getResourcePath();
            HttpResponse response = HttpsClientRequest.doGet(invocationUrl, headers);
            Assert.assertNotNull(response, "Error occurred while invoking the url " + invocationUrl + " HttpResponse ");
            Assert.assertEquals(response.getResponseCode(), HttpStatus.SC_FORBIDDEN,
                    "Status code mismatched. Endpoint:" + endpoint + " HttpResponse ");
        }
        storeRestClient.removeApplicationById(applicationId);
        for (TestEntry testEntry : testEntryList) {
            publisherRestClient.deleteAPI(testEntry.getApiID());
            Utils.delay(TestConstant.DEPLOYMENT_WAIT_TIME, "Interrupted while waiting to API delete event");
            String endpoint = partitionEndpointMap.get(testEntry.getPartition());
            String invocationUrl = endpoint + testEntry.getApiContext() + "/" + testEntry.getApiVersion()
                    + testEntry.getResourcePath();
            assert404Response(invocationUrl);
        }
    }

    private void initializeEntryMap() {
        testEntryList = new ArrayList<>();
        addTestEntryToList("APIEvent1", "1.0.0", "testOrg1/apiEvent1", "Default-p1");
        addTestEntryToList("APIEvent2", "1.0.0", "testOrg1/apiEvent2", "Default-p1");
        addTestEntryToList("APIEvent3", "1.0.0", "testOrg1/apiEvent3", "Default-p1");
        addTestEntryToList("APIEvent4", "1.0.0", "testOrg1/apiEvent4", "Default-p2");
        addTestEntryToList("APIEvent5", "1.0.0", "testOrg1/apiEvent5", "Default-p2");
    }

    private void assert200Response(String url) throws CCTestException {
        HttpResponse response = HttpsClientRequest.retryGetRequestUntilDeployed(url, headers);
        Assert.assertNotNull(response, "Error occurred while invoking the endpoint " + url + " HttpResponse ");
        Assert.assertEquals(HttpStatus.SC_SUCCESS, response.getResponseCode(),
                "Status code mismatched. Endpoint:" + url + " HttpResponse ");
    }

    private void assert404Response(String url) throws CCTestException {
        HttpResponse response = HttpsClientRequest.doGet(url, headers);
        Assert.assertNotNull(response, "Error occurred while invoking: " + url + " HttpResponse ");
        Assert.assertEquals(HttpStatus.SC_NOT_FOUND, response.getResponseCode(),
                "Status code mismatched. Endpoint:" + url + " HttpResponse ");
    }

    private void addTestEntryToList(String apiName, String apiVersion, String apiContext, String partition) {
        TestEntry testEntry = new TestEntry();
        testEntry.setApiName(apiName);
        testEntry.setApiVersion(apiVersion);
        testEntry.setPartition(partition);
        testEntry.setApiContext(apiContext);
        testEntryList.add(testEntry);
    }

    private void checkRedisEntry(String orgHandle, String apiContext, String apiVersion, String expectedValue) {
        String value = jedis.get(String.format("global-adapter#Default#%s_%s_%s",orgHandle, apiContext, apiVersion));
        Assert.assertEquals(value, expectedValue, " Mismatch found while reading redis entry for " +
                String.format("global-adapter#Default#%s_%s_%s",orgHandle, apiContext, apiVersion));
    }

    public static Jedis createJedisConnection() {
        JedisClientConfig config = DefaultJedisClientConfig.builder().database(0).clientName("global-adapter")
                .socketTimeoutMillis(300000).build();
        try (Jedis jedis = new Jedis(new URI("rediss://redis-host:6379"), config)) {
            jedis.auth("admin");
            return jedis;
        } catch (URISyntaxException e) {
            e.printStackTrace();
        }
        return null;
    }

    public void populationPartitionEndpointMap() {
        partitionEndpointMap = new HashMap<>();
        partitionEndpointMap.put("Default-p1", "https://localhost:9095/");
        partitionEndpointMap.put("Default-p2", "https://localhost:9096/");
    }

    private TestEntry deleteTestEntry(TestEntry testEntry) throws Exception {
        StoreUtils.removeSubscriptionsForAnAPI(applicationId, testEntry.getApiID(), storeRestClient);
        Utils.delay(TestConstant.DEPLOYMENT_WAIT_TIME, "Interrupted while waiting to subscription delete event");
        publisherRestClient.deleteAPI(testEntry.getApiID());
        String invocationUrl = partitionEndpointMap.get(testEntry.getPartition()) + testEntry.getApiContext() + "/"
                + testEntry.getApiVersion() + testEntry.getResourcePath();
        Utils.delay(TestConstant.DEPLOYMENT_WAIT_TIME, "Interrupted while waiting to API delete event");
        HttpResponse response = HttpsClientRequest.doGet(invocationUrl, headers);
        Assert.assertNotNull(response, "Error occurred while invoking the endpoint " + invocationUrl + " HttpResponse ");
        Assert.assertEquals(response.getResponseCode(), HttpStatus.SC_NOT_FOUND,
                "Status code mismatched. Endpoint:" + invocationUrl + " HttpResponse ");
        testEntryList.remove(testEntry);
        return testEntry;
    }

    private void checkTestEntry(TestEntry testEntry) throws Exception {
        String orgHandle = testEntry.getApiContext().substring(0, testEntry.getApiContext().indexOf("/", 1));
        String context = testEntry.getApiContext().substring(testEntry.getApiContext().indexOf("/", 1));
        checkRedisEntry(orgHandle, context, testEntry.getApiVersion(), testEntry.getPartition());
        for (Map.Entry<String, String> mapEntry : partitionEndpointMap.entrySet()) {
            String url = mapEntry.getValue() + testEntry.getApiContext() + "/"
                    + testEntry.getApiVersion() + testEntry.getResourcePath();
            if (testEntry.getPartition().equals(mapEntry.getKey())) {
                assert200Response(url);
            } else {
                assert404Response(url);
            }
        }
    }
}


class TestEntry {
    private String apiName;
    private String apiVersion;
    private String apiContext;
    private String applicationName;
    private String apiID;
    private String applicationID;
    private String partition;
    private String resourcePath = "/pet/findByStatus";

    public String getApiName() {
        return apiName;
    }

    public void setApiName(String apiName) {
        this.apiName = apiName;
    }

    public String getApiVersion() {
        return apiVersion;
    }

    public void setApiVersion(String apiVersion) {
        this.apiVersion = apiVersion;
    }

    public String getApiContext() {
        return apiContext;
    }

    public void setApiContext(String apiContext) {
        this.apiContext = apiContext;
    }

    public String getApplicationName() {
        return applicationName;
    }

    public void setApplicationName(String applicationName) {
        this.applicationName = applicationName;
    }

    public String getApiID() {
        return apiID;
    }

    public void setApiID(String apiID) {
        this.apiID = apiID;
    }

    public String getApplicationID() {
        return applicationID;
    }

    public void setApplicationID(String applicationID) {
        this.applicationID = applicationID;
    }

    public String getPartition() {
        return partition;
    }

    public void setPartition(String partition) {
        this.partition = partition;
    }

    public String getResourcePath() {
        return resourcePath;
    }
}
