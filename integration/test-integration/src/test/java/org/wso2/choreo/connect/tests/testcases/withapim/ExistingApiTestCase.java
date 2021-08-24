/*
 * Copyright (c) 2021, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
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
import io.netty.handler.codec.http.HttpHeaderNames;
import org.testng.annotations.BeforeClass;
import org.testng.annotations.Test;
import org.wso2.choreo.connect.mockbackend.ResponseConstants;
import org.wso2.choreo.connect.tests.apim.ApimBaseTest;
import org.wso2.choreo.connect.tests.apim.ApimResourceProcessor;
import org.wso2.choreo.connect.tests.apim.utils.StoreUtils;
import org.wso2.choreo.connect.tests.context.CCTestException;
import org.wso2.choreo.connect.tests.util.TestConstant;
import org.wso2.choreo.connect.tests.util.Utils;

import java.net.MalformedURLException;
import java.util.HashMap;
import java.util.Map;

public class ExistingApiTestCase extends ApimBaseTest {
    private static final String VHOST_API_ENDPOINT = "testOrg1/vhostApi1/1.0.0/pet/findByStatus";

    @BeforeClass(alwaysRun = true, description = "initialize setup")
    void setup() throws Exception {
        super.initWithSuperTenant();
    }

    @Test
    public void testExistingApiWithSubscriptions() throws CCTestException {
        String applicationId = ApimResourceProcessor.applicationNameToId.get(VhostApimTestCase.APPLICATION_NAME);
        String accessToken = StoreUtils.generateUserAccessToken(apimServiceURLHttps, applicationId,
                user, storeRestClient);

        Map<String, String> requestHeaders = new HashMap<>();
        requestHeaders.put(TestConstant.AUTHORIZATION_HEADER, "Bearer " + accessToken);
        requestHeaders.put(HttpHeaderNames.HOST.toString(), "localhost");
        try {
            VhostApimTestCase.testInvokeAPI(Utils.getServiceURLHttps(VHOST_API_ENDPOINT), requestHeaders, HttpStatus.SC_SUCCESS, ResponseConstants.RESPONSE_BODY);
        } catch (MalformedURLException e) {
            throw new CCTestException("Error while calling the vhostAPIEndpoint");
        }
    }
}
