# choreo-connect-global-adapter
Global adapter component for Choreo Connect deployment. This component mediates between the control plane and partitioned adapter data plane.


# How to Build and Run integration tests

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

```mvn clean install -P ReleaseWithResources```

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
