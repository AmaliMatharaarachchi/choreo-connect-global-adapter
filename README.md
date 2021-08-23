# Choreo Connect Global Adapter
![Build workflow](https://github.com/wso2-enterprise/choreo-connect-global-adapter/actions/workflows/main.yml/badge.svg)

In order to manage the growing number of APIs straining the envoy routers in Choreo data plane the API partitioning was required. As a method to manage the APIs and handle the API partitioning the Control Plane was introduced with the Global Adapter component to act as the partition manager for APIs. This component interacts with the API event hub to retrieve APIs and API events to handle paritioning and distribution of APIs to the respective partitions. 

## Architecture 
The architecture and the designs are discussed under the [API partitioning](https://github.com/wso2-enterprise/choreo/wiki/Choreo-Architecture-Links) section. The overall architectural for APIM partition is 
illustrated in the following diagram. 

![alt text]( Architecture.png "API Partitioning Architecture")

