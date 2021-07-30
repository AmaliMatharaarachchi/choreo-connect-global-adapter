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
     