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
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.testng.Assert;
import org.testng.annotations.AfterClass;
import org.testng.annotations.BeforeClass;
import org.testng.annotations.Test;
import org.wso2.am.integration.test.utils.bean.APIRequest;
import org.wso2.choreo.connect.tests.apim.ApimBaseTest;
import org.wso2.choreo.connect.tests.apim.ApimResourceProcessor;
import org.wso2.choreo.connect.tests.apim.dto.AppWithConsumerKey;
import org.wso2.choreo.connect.tests.apim.dto.Application;
import org.wso2.choreo.connect.tests.apim.utils.PublisherUtils;
import org.wso2.choreo.connect.tests.apim.utils.StoreUtils;
import org.wso2.choreo.connect.tests.common.model.PartitionTestEntry;
import org.wso2.choreo.connect.tests.context.CCTestException;
import org.wso2.choreo.connect.tests.util.HttpsClientRequest;
import org.wso2.choreo.connect.tests.util.HttpResponse;
import org.wso2.choreo.connect.tests.util.PartitionTestUtils;
import org.wso2.choreo.connect.tests.util.TestConstant;
import org.wso2.choreo.connect.tests.util.Utils;
import redis.clients.jedis.Jedis;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class PartitionTestCaseWithEvents extends ApimBaseTest {
    private static final String APP_NAME = "GlobalAdapterEventTest";
    private String applicationId;
    Map <String, String> headers;
    List<PartitionTestEntry> newAPITestEntryList = new ArrayList<>();
    List<PartitionTestEntry> existingAPITestEntryList = new ArrayList<>();
    private Jedis jedis;
    private AppWithConsumerKey appWithConsumerKey;
    private static final Logger log = LoggerFactory.getLogger(PartitionTestCaseWithEvents.class);

    @BeforeClass(alwaysRun = true, description = "initialize setup")
    void setup() throws Exception {
        super.initWithSuperTenant();
        // Populate the API Entries which are going to be added in the middle.
        initializeTestEntryMap();
        // Initialize Jedis (Redis_ Client
        jedis = PartitionTestUtils.createJedisConnection();

        Application app = new Application(APP_NAME, TestConstant.APPLICATION_TIER.UNLIMITED);
        appWithConsumerKey = StoreUtils.createApplicationWithKeys(app, storeRestClient);
        applicationId = appWithConsumerKey.getApplicationId();
    }

    @Test
    public void testAlreadyExistingAPIs() throws Exception {
        for (PartitionTestEntry entry : existingAPITestEntryList) {
            String apiId = ApimResourceProcessor.apiNameToId.get(entry.getApiName());
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
        for (PartitionTestEntry testEntry : existingAPITestEntryList) {
            // Checks against both the router partitions available.
            // If the partition matches, it should return 200 OK, otherwise 404.
            PartitionTestUtils.checkTestEntry(jedis, testEntry, headers, false);
        }

    }

    @Test
    public void testCreateEvents() throws Exception {

        for (PartitionTestEntry entry : newAPITestEntryList) {
            APIRequest apiRequest = PublisherUtils.createSampleAPIRequest(
                    entry.getApiName(), entry.getApiContext(), entry.getApiVersion(), user.getUserName());
            String apiId = PublisherUtils.createAndPublishAPI(apiRequest, publisherRestClient);
            entry.setApiID(apiId);
            StoreUtils.subscribeToAPI(apiId, applicationId, TestConstant.SUBSCRIPTION_TIER.UNLIMITED, storeRestClient);
        }

        String accessToken = StoreUtils.generateUserAccessToken(apimServiceURLHttps,
                appWithConsumerKey.getConsumerKey(), appWithConsumerKey.getConsumerSecret(),
                new String[]{}, user, storeRestClient);

        headers = new HashMap<>();
        headers.put(HttpHeaders.AUTHORIZATION, "Bearer " + accessToken);

        Utils.delay(60000, "Interrupted when waiting for the " +
                "subscriptions to be deployed");
        //Invoke all the added API
        for (PartitionTestEntry testEntry : newAPITestEntryList) {
            // Checks against both the router partitions available.
            // If the partition matches, it should return 200 OK, otherwise 404.
            PartitionTestUtils.checkTestEntry(jedis, testEntry, headers, true);
        }
    }

    @Test
    public void testDeleteAndAddTwoAPIs() throws Exception {
        int testEntryListSize = newAPITestEntryList.size();
        // first delete the 1st API and the last API in list (which belongs to two partitions)
        // now there is a vacant index in each partition.
        PartitionTestEntry testEntryFirst = newAPITestEntryList.get(0);
        PartitionTestEntry testEntryLast = newAPITestEntryList.get(testEntryListSize - 1);

        testEntryFirst = undeployTestEntry(testEntryFirst);
        String testEntryFirstPartition = testEntryFirst.getPartition();
        String testEntryFirstAPIID = testEntryFirst.getApiID();
        testEntryLast = undeployTestEntry(testEntryLast);
        String testEntryLastPartition = testEntryLast.getPartition();
        String testEntryLastAPIID = testEntryLast.getApiID();

        // Since the order is going to be changed, the partitions should be swapped.
        testEntryFirst.setPartition(testEntryLastPartition);
        testEntryLast.setPartition(testEntryFirstPartition);

        PublisherUtils.createAPIRevisionAndDeploy(testEntryLastAPIID, publisherRestClient);
        StoreUtils.subscribeToAPI(testEntryLastAPIID, applicationId, TestConstant.SUBSCRIPTION_TIER.UNLIMITED, storeRestClient);
        testEntryLast.setApiID(testEntryLastAPIID);
        Utils.delay(TestConstant.DEPLOYMENT_WAIT_TIME, "Interrupted while waiting to subscription delete event");

        PublisherUtils.createAPIRevisionAndDeploy(testEntryFirstAPIID, publisherRestClient);
        testEntryFirst.setApiID(testEntryFirstAPIID);
        StoreUtils.subscribeToAPI(testEntryFirstAPIID, applicationId, TestConstant.SUBSCRIPTION_TIER.UNLIMITED, storeRestClient);
        Utils.delay(TestConstant.DEPLOYMENT_WAIT_TIME, "Interrupted while waiting to subscription delete event");

        PartitionTestUtils.checkTestEntry(jedis, testEntryFirst, headers, true);
        PartitionTestUtils.checkTestEntry(jedis, testEntryLast, headers, true);
    }

    @Test
    public void testDeleteEvents() throws Exception{
        StoreUtils.removeAllSubscriptionsForAnApp(applicationId, storeRestClient);
        Utils.delay(TestConstant.DEPLOYMENT_WAIT_TIME, "Interrupted while waiting to subscription delete event");

        for (PartitionTestEntry testEntry : newAPITestEntryList) {
            String endpoint = PartitionTestUtils.PARTITION_ENDPOINT_MAP.get(testEntry.getPartition());
            String invocationUrl = endpoint + testEntry.getApiContext() + "/" + testEntry.getApiVersion()
                    + testEntry.getResourcePath();
            HttpResponse response = HttpsClientRequest.doGet(invocationUrl, headers);
            Assert.assertNotNull(response, "Error occurred while invoking the url " + invocationUrl + " HttpResponse ");
            Assert.assertEquals(response.getResponseCode(), HttpStatus.SC_FORBIDDEN,
                    "Status code mismatched. Endpoint:" + endpoint + " HttpResponse ");
        }
        storeRestClient.removeApplicationById(applicationId);
        for (PartitionTestEntry testEntry : newAPITestEntryList) {
            publisherRestClient.deleteAPI(testEntry.getApiID());
            Utils.delay(TestConstant.DEPLOYMENT_WAIT_TIME, "Interrupted while waiting to API delete event");
            String endpoint = PartitionTestUtils.PARTITION_ENDPOINT_MAP.get(testEntry.getPartition());
            String invocationUrl = endpoint + testEntry.getApiContext() + "/" + testEntry.getApiVersion()
                    + testEntry.getResourcePath();
            PartitionTestUtils.assert404Response(invocationUrl, headers);
        }
    }

    private void initializeTestEntryMap() {
        PartitionTestUtils.addTestEntryToList(existingAPITestEntryList, "API1", "1.0.0", "testOrg1/api1",
                PartitionTestUtils.PARTITION_1);
        PartitionTestUtils.addTestEntryToList(existingAPITestEntryList, "API2", "1.0.0", "testOrg1/api2",
                PartitionTestUtils.PARTITION_1);
        PartitionTestUtils.addTestEntryToList(existingAPITestEntryList, "API3", "1.0.0", "testOrg1/api3",
                PartitionTestUtils.PARTITION_1);
        PartitionTestUtils.addTestEntryToList(existingAPITestEntryList, "API4", "1.0.0", "testOrg1/api4",
                PartitionTestUtils.PARTITION_1);
        PartitionTestUtils.addTestEntryToList(existingAPITestEntryList, "API5", "1.0.0", "testOrg1/api5",
                PartitionTestUtils.PARTITION_1);

        PartitionTestUtils.addTestEntryToList(newAPITestEntryList, "APIEvent1", "1.0.0", "testOrg1/apiEvent1",
                PartitionTestUtils.PARTITION_1);
        PartitionTestUtils.addTestEntryToList(newAPITestEntryList, "APIEvent2", "1.0.0", "testOrg1/apiEvent2",
                PartitionTestUtils.PARTITION_1);
        PartitionTestUtils.addTestEntryToList(newAPITestEntryList, "APIEvent3", "1.0.0", "testOrg1/apiEvent3",
                PartitionTestUtils.PARTITION_1);
        PartitionTestUtils.addTestEntryToList(newAPITestEntryList, "APIEvent4", "1.0.0", "testOrg1/apiEvent4",
                PartitionTestUtils.PARTITION_2);
    }

    private PartitionTestEntry undeployTestEntry(PartitionTestEntry testEntry) throws Exception {
        StoreUtils.removeSubscriptionsForAnAPI(applicationId, testEntry.getApiID(), storeRestClient);
        Utils.delay(TestConstant.DEPLOYMENT_WAIT_TIME, "Interrupted while waiting to subscription delete event");
        PublisherUtils.undeployAndDeleteAPIRevisions(testEntry.getApiID(), publisherRestClient);
        String invocationUrl = PartitionTestUtils.PARTITION_ENDPOINT_MAP.get(testEntry.getPartition()) + testEntry.getApiContext() + "/"
                + testEntry.getApiVersion() + testEntry.getResourcePath();
        Utils.delay(TestConstant.DEPLOYMENT_WAIT_TIME, "Interrupted while waiting to API delete event");
        HttpResponse response = HttpsClientRequest.doGet(invocationUrl, headers);
        Assert.assertNotNull(response, "Error occurred while invoking the endpoint " + invocationUrl + " HttpResponse ");
        Assert.assertEquals(response.getResponseCode(), HttpStatus.SC_NOT_FOUND,
                "Status code mismatched. Endpoint:" + invocationUrl + " HttpResponse ");
        newAPITestEntryList.remove(testEntry);
        return testEntry;
    }

    @AfterClass
    public void afterClass() {
        try {
            StoreUtils.removeAllSubscriptionsForAnApp(applicationId, storeRestClient);
        } catch (CCTestException e) {
            log.error("Error while unsubscribing APIs under application: " + applicationId, e);
        }
    }
}
