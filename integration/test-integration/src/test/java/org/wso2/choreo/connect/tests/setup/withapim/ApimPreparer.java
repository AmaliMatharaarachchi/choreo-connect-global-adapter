/*
 * Copyright (c) 2021, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 *
 * WSO2 Inc. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */
package org.wso2.choreo.connect.tests.setup.withapim;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.testng.annotations.AfterSuite;
import org.testng.annotations.BeforeTest;
import org.testng.annotations.Optional;
import org.testng.annotations.Parameters;
import org.wso2.am.integration.clients.publisher.api.ApiException;
import org.wso2.choreo.connect.tests.apim.ApimBaseTest;
import org.wso2.choreo.connect.tests.apim.ApimResourceProcessor;
import org.wso2.choreo.connect.tests.apim.utils.PublisherUtils;
import org.wso2.choreo.connect.tests.apim.utils.StoreUtils;
import org.wso2.choreo.connect.tests.context.CCTestException;
import org.wso2.choreo.connect.tests.context.ChoreoConnectImpl;
import org.wso2.choreo.connect.tests.util.TestConstant;
import org.wso2.choreo.connect.tests.util.Utils;

import java.util.Map;

/**
 * APIs, Apps, Subs created here will be used to test whether
 * resources that existed in APIM were pulled by CC during startup
 * This class must run before CcStartupExecutor
 */
public class ApimPreparer extends ApimBaseTest {
    private static final Logger log = LoggerFactory.getLogger(ApimPreparer.class);
    /**
     * Initialize the clients in the super class and create APIs, Apps, Subscriptions etc.
     */
    @BeforeTest
    @Parameters({"isGASpecific"})
    private void createApiAppSubsEtc(@Optional("") String isGASpecific) throws Exception {
        super.initWithSuperTenant();
        // The tests can be run against the same API Manager instance. Therefore, we clean first
        // in case the tests get interrupted before it ends in the previous run
        StoreUtils.removeAllSubscriptionsAndAppsFromStore(storeRestClient);
        PublisherUtils.removeAllApisFromPublisher(publisherRestClient);
        ApimResourceProcessor apimResourceProcessor = new ApimResourceProcessor();
        apimResourceProcessor.createApisAppsSubs(user.getUserName(), publisherRestClient, storeRestClient,
                isGASpecific);

        if(ChoreoConnectImpl.checkCCInstanceHealth()) {
            //wait till all resources deleted and are redeployed
            Utils.delay(TestConstant.DEPLOYMENT_WAIT_TIME*8, "Interrupted while waiting for DELETE and" +
                    " CREATE events to be deployed");
        }
    }

    @AfterSuite
    public void deleteAPIsApps() {
        for (Map.Entry<String, String> entry :
                ApimResourceProcessor.applicationNameToId.entrySet()) {
            try {
                StoreUtils.removeAllSubscriptionsForAnApp(entry.getValue(), storeRestClient);
            } catch (CCTestException e) {
                log.error("Error while unsubscribing APIs under application: " + entry.getKey(), e);
            }
        }
        for (Map.Entry<String, String> entry : ApimResourceProcessor.apiNameToId.entrySet()) {
            try {
                publisherRestClient.deleteAPI(entry.getValue());
            } catch (ApiException e) {
                log.error("Error while deleting API: " + entry.getKey(), e);
            }
        }
    }
}
