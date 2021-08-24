# Choreo Connect Global Adapter
![Build workflow](https://github.com/wso2-enterprise/choreo-connect-global-adapter/actions/workflows/main.yml/badge.svg)

In order to manage the growing number of APIs straining the envoy routers in Choreo data plane the API partitioning was required. As a method to manage the APIs and handle the API partitioning the Control Plane was introduced with the Global Adapter component to act as the partition manager for APIs. This component interacts with the API event hub to retrieve APIs and API events to handle paritioning and distribution of APIs to the respective partitions. 

## Architecture 
The architecture and the designs are discussed under the [API partitioning](https://github.com/wso2-enterprise/choreo/wiki/Choreo-Architecture-Links) section.
  * [Slide Deck](https://docs.google.com/presentation/d/1atVgz6BjN2FP-O6hYjLKm1WBE0FFBXMRhBovtoZZCpA/edit#slide=id.ga8e32a3ae0_0_1071)
  * [Initial proposal](https://docs.google.com/document/d/1g3ijX4KNdJHw5L4ud_D3CpQ7osUKgG3bZI7QtT41HmU/edit?usp=sharing)
  * [Final Document - WIP](https://docs.google.com/document/d/1zKUjRiQ1PQVDWZYjK9ns4S6ncx4Bfhn8d3fGGgNueKM/edit?usp=sharing) 

The overall architecture for APIM partition is illustrated in the following diagram. 

![API Partitioning Architecture]( Architecture.png "API Partitioning Architecture")

## How to Build and Run integration tests

Please note that the test client methods used within the integration tests are from the choreo-product-apim repository.
If you have built the product-apim repository prior to running integration tests (without building the choreo-product-apim)
it will run into compilation issues as the test classes contain the same package names.

You need to populate the following mongodb related environment variables.

```
mongodb_connection_string
mongodb_group_id
mongodb_cluster_name
mongodb_public_key
mongodb_private_key
```

For the first build, run with `ReleaseWithResources` profile. 
This would generate the choreo-connect docker images and choreo-product-apim
docker image.

```mvn clean install -P ReleaseWithResourcesGen```

---
**NOTE**

###### Prerequisites for running integration tests.

1. JDK 8 distribution should be installed in the host machine.
    - `JAVA_8_HOME` environmental variable should be initialized accordingly.

2. JDK 11 distribution should be installed in the host machine.
    - `JAVA_11_HOME` environmental variable should be initialized accordingly.

3. Docker should be installed and running in the host machine.

4. GIT should be installed in the host machine.
    - GIT credentials needs to be populated prior to the build as it is required to
      clone/pull the chroeo-product-apim

5. Maven should be installed within the host machine and PATH variable
   should be updated accordingly.
    - If the installed maven version is 3.8.x, then the settings.xml needs
      to be configured such that it can use http connections to pull the artifacts.

---


Then onwards, you can use `Release` profile to run integration tests.

```mvn clean install -P Release```
