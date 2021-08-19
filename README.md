# choreo-connect-global-adapter
Global adapter component for Choreo Connect deployment. This component mediates between the control plane and partitioned adapter data plane.


# How to Build and Run integration tests
For the first time, run with `IntegrationWithResourcs` profile. 
This would generate the choreo-connect docker images and choreo-product-apim
docker image.

```mvn clean install -P IntegrationWithResources```
