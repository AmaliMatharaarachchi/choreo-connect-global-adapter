#!/bin/bash
# --------------------------------------------------------------------
# Copyright (c) 2021, WSO2 Inc. (http://wso2.com).
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
# -----------------------------------------------------------------------

# get script location
script_dir=$(cd -P -- "$(dirname -- "$0")" && pwd -P)
cd $script_dir
#rm -rf target
APIM_DIR="${script_dir}/gen/choreo-apim"

# If the workflow environment variable is available, there is no need to clone or pull the repository
if [[ -z "${IS_WORKFLOW_ENV}" ]]; then
  if [ -d "$APIM_DIR" ]; then
    cd gen/choreo-apim/choreo-product-apim
    git stash save --include-untracked
    git stash drop stash@{0}
    git pull origin main
    if [ $? -eq 0 ]; then
      echo "Pull operation for choreo-product-apim is successful"
    else
      echo "Pull operation for choreo-product-apim is failed"
      exit 1
    fi
  else
    mkdir -p gen/choreo-apim
    cd gen/choreo-apim
  #  TODO: (VirajSalaka) check if it is executed from a workflow, then it is not required to clone
    git clone https://github.com/wso2-enterprise/choreo-product-apim
    if [ $? -eq 0 ]; then
      echo "Clone operation for choreo-product-apim is successful"
    else
      echo "Clone operation for choreo-product-apim is failed"
      exit 1
    fi
    cd choreo-product-apim
  fi
else
 cd $GITHUB_WORKSPACE/integration/integration-artifacts/scripts/gen/choreo-apim/choreo-product-apim
fi

if [[ -z "${JAVA_8_HOME}" ]]; then
  echo "Error: JAVA_8_HOME is not defined."
  exit 1
else
  JAVA_HOME=${JAVA_8_HOME}
  PATH=$JAVA_HOME/bin:$PATH
fi

mvn clean install -Dmaven.test.skip

if [ $? -eq 0 ]; then
    echo "Maven build for choreo-product-apim is successful"
else
  echo "Maven build for choreo-product-apim is failed"
  exit 1
fi

docker build . -t wso2/choreo-product-apim:4.0 --no-cache

if [ $? -eq 0 ]; then
    echo "Docker build for choreo-product-apim is successful"
else
  echo "Docker build for choreo-product-apim is failed"
  exit 1
fi

cd ../../..

if [[ -z "${JAVA_11_HOME}" ]]; then
  echo "Error: JAVA_11_HOME is not defined."
  exit 1
else
  JAVA_HOME=${JAVA_11_HOME}
  PATH=$JAVA_HOME/bin:$PATH
fi

CC_DIR="${script_dir}/gen/choreo-connect"
echo $CC_DIR
# If the workflow environment variable is available, there is no need to clone or pull the repository
if [[ -z "${IS_WORKFLOW_ENV}" ]]; then
  if [ -d "${CC_DIR}" ]; then
    cd gen/choreo-connect/product-microgateway
    git pull origin feature/global-adapter
    if [ $? -eq 0 ]; then
      echo "Pull operation for choreo-connect is successful"
    else
      echo "Pull operation for choreo-connect is failed"
      exit 1
    fi
  else
    cd gen
    mkdir choreo-connect
    cd choreo-connect
    git clone https://github.com/wso2/product-microgateway
    cd product-microgateway
    if [ $? -eq 0 ]; then
      echo "Clone operation for choreo-connect is successful"
    else
      echo "Clone operation for choreo-connect is failed"
      exit 1
    fi
  fi
else
  cd ${GITHUB_WORKSPACE}/integration/integration-artifacts/scripts/gen/choreo-connect/product-microgateway
fi

git checkout feature/global-adapter
mvn clean install -P Release -Dmaven.test.skip -s .maven/settings.xml

if [ $? -eq 0 ]; then
    echo "Maven build for choreo-connect is successful"
else
  echo "Maven build for choreo-connect is failed"
  exit 1
fi
