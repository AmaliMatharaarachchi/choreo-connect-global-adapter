package org.wso2.choreo.connect.tests.testcases.withapim;

import com.google.common.net.HttpHeaders;
import org.testng.annotations.BeforeClass;
import org.testng.annotations.Test;
import org.wso2.choreo.connect.tests.apim.ApimBaseTest;
import org.wso2.choreo.connect.tests.apim.ApimResourceProcessor;
import org.wso2.choreo.connect.tests.apim.dto.AppWithConsumerKey;
import org.wso2.choreo.connect.tests.apim.dto.Application;
import org.wso2.choreo.connect.tests.apim.utils.StoreUtils;
import org.wso2.choreo.connect.tests.common.model.PartitionTestEntry;
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
    private static final String PARTITION_1 = "Default-p1";
    private static final String PARTITION_2 = "Default-p2";
    List<PartitionTestEntry> existingAPITestEntryList = new ArrayList<>();
    private String applicationId;
    Map<String, String> headers;
    private Map<String, String> partitionEndpointMap;
    private AppWithConsumerKey appWithConsumerKey;
    private Jedis jedis;

    @BeforeClass(alwaysRun = true, description = "initialize setup")
    void setup() throws Exception {
        super.initWithSuperTenant();
        partitionEndpointMap =  PartitionTestUtils.PARTITION_ENDPOINT_MAP;
        // Populate the API Entries which are going to be added in the middle.
//        initializeTestEntryMap();
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

        //Invoke all the added API
        for (PartitionTestEntry testEntry : existingAPITestEntryList) {

            // Checks against both the router partitions available.
            // If the partition matches, it should return 200 OK, otherwise 404.
            PartitionTestUtils.checkTestEntry(jedis, testEntry, headers, false);
        }
    }
}
