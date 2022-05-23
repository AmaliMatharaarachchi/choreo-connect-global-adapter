package org.wso2.choreo.connect.tests.testcases.withapim;

import com.google.common.net.HttpHeaders;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.testng.Assert;
import org.testng.annotations.AfterClass;
import org.testng.annotations.BeforeClass;
import org.testng.annotations.Test;
import org.wso2.choreo.connect.tests.apim.ApimBaseTest;
import org.wso2.choreo.connect.tests.apim.ApimResourceProcessor;
import org.wso2.choreo.connect.tests.apim.dto.AppWithConsumerKey;
import org.wso2.choreo.connect.tests.apim.dto.Application;
import org.wso2.choreo.connect.tests.apim.utils.StoreUtils;
import org.wso2.choreo.connect.tests.common.model.PartitionTestEntry;
import org.wso2.choreo.connect.tests.context.CCTestException;
import org.wso2.choreo.connect.tests.util.PartitionTestUtils;
import org.wso2.choreo.connect.tests.util.TestConstant;
import org.wso2.choreo.connect.tests.util.Utils;
import redis.clients.jedis.Jedis;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class PartitionTestCaseStartup extends ApimBaseTest {

    private static final String APP_NAME = "GlobalAdapterEventTest";
    private static final Logger log = LoggerFactory.getLogger(PartitionTestCaseStartup.class);

    List<PartitionTestEntry> existingAPITestEntryList = new ArrayList<>();
    private String applicationId;
    Map<String, String> headers;
    private AppWithConsumerKey appWithConsumerKey;
    private Jedis jedis;

    @BeforeClass(alwaysRun = true, description = "initialize setup")
    void setup() throws Exception {
        super.initWithSuperTenant();
        // Populate the API Entries which are going to be added in the middle.
        initializeTestEntryMap();
        // Initialize Jedis Client
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

        // Population PartitionID (During startup, it is not possible to point out which API is deployed in
        // which partition. Hence the partitionID should be populated from the redis.
        // To check if the partitioning has happened properly, we need to depend on whether at least one API is assigned
        // to the partition_2
        int partition2AssignedCount = 0;
        for (PartitionTestEntry testEntry : existingAPITestEntryList) {        
            // Checks against both the router partitions available.
            // If the partition matches, it should return 200 OK, otherwise 404.
            String currentPartitionContext = PartitionTestUtils.getRedisEntry(jedis, testEntry.getOrgName(),
                    testEntry.getApiContext(), testEntry.getApiVersion());
            testEntry.setPartition(currentPartitionContext.split("/")[0]);

            if (String.format("%s/%s/%s/%s", PartitionTestUtils.PARTITION_2, testEntry.getOrgName(),
                    testEntry.getApiContext(), testEntry.getApiVersion()).equals(currentPartitionContext)) {
                partition2AssignedCount++;
            }
        }
        Assert.assertEquals(1, partition2AssignedCount, "Partition assignment mismatch.");

        //Invoke all the added API
        for (PartitionTestEntry testEntry : existingAPITestEntryList) {

            // Checks against both the router partitions available.
            // If the partition matches, it should return 200 OK, otherwise 404.
            PartitionTestUtils.checkTestEntry(jedis, testEntry, headers, true);
        }
    }

    private void initializeTestEntryMap() {
        PartitionTestUtils.addTestEntryToList(existingAPITestEntryList, "API1", "testOrg1", "1.0.0",
                "api1", "");
        PartitionTestUtils.addTestEntryToList(existingAPITestEntryList, "API2", "testOrg1", "1.0.0",
                "api2", "");
        PartitionTestUtils.addTestEntryToList(existingAPITestEntryList, "API3", "testOrg1", "1.0.0",
                "api3", "");
        PartitionTestUtils.addTestEntryToList(existingAPITestEntryList, "API4", "testOrg1", "1.0.0",
                "api4", "");
        PartitionTestUtils.addTestEntryToList(existingAPITestEntryList, "API5", "testOrg1", "1.0.0",
                "api5", "");
        PartitionTestUtils.addTestEntryToList(existingAPITestEntryList, "API6", "testOrg1", "1.0.0",
                "api6", "");
        PartitionTestUtils.addTestEntryToList(existingAPITestEntryList, "API7", "testOrg1", "1.0.0",
                "api7", "");
        PartitionTestUtils.addTestEntryToList(existingAPITestEntryList, "API8", "testOrg1", "1.0.0",
                "api8", "");
        PartitionTestUtils.addTestEntryToList(existingAPITestEntryList, "API9", "testOrg1", "1.0.0",
                "api9", "");
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
