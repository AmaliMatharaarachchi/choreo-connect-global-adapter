#!/bin/sh
# --------------------------------------------------------------------
# Copyright (c) 2021, WSO2 Inc. (http://wso2.com) All Rights Reserved.
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

count=1
napTime=5
retryCount=5
RedisSubscribedStatusfilePath="/redis/redis-subscribe.txt"
LOG_LOCATION="/redis"  
exec >> $LOG_LOCATION/subscribe.log 2>&1

while [ $count -le $retryCount ]; do
  echo "naptime: $napTime"
  sleep $napTime
  #increment naptime to next retry.
  napTime=$((napTime*2))

  if [ -f "$RedisSubscribedStatusfilePath" ]; then
    echo "$RedisSubscribedStatusfilePath exists."

    status=`cat $RedisSubscribedStatusfilePath`

    if [[ -z "$status" ]]  || [[  $status != "subscribed" ]] 
    then
      curl -s -o -X PUT "http://localhost:8080/init-redis-subscribe" > /dev/null 2>&1 & 
    else    
      echo "Already Subscribed. Stopping the retry effort"
      break
    fi
  else 
    echo "No file found in path: $RedisSubscribedStatusfilePath"
    curl -s -o -X PUT "http://localhost:8080/init-redis-subscribe" > /dev/null 2>&1 & 
  fi
  count=$(( count + 1 ))
done

tail -f /dev/null
