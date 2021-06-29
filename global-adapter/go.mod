module github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter

go 1.15

replace github.com/wso2/product-microgateway/adapter => github.com/VirajSalaka/product-microgateway/adapter v0.0.0-20210616092420-815e2dda0d70

require (
	github.com/denisenkom/go-mssqldb v0.10.0
	github.com/envoyproxy/go-control-plane v0.9.8
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/sirupsen/logrus v1.7.0
	github.com/wso2/product-microgateway/adapter v0.0.0
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616 // indirect
	golang.org/x/tools v0.1.3 // indirect
	google.golang.org/grpc v1.33.2
)
