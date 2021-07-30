module github.com/wso2-enterprise/choreo-connect-global-adapter/global-adapter

go 1.15

replace github.com/wso2/product-microgateway/adapter => github.com/VirajSalaka/product-microgateway/adapter v0.0.0-20210719081110-8c7462a03dc2

require (
	github.com/DATA-DOG/go-sqlmock v1.5.0
	github.com/denisenkom/go-mssqldb v0.10.0
	github.com/envoyproxy/go-control-plane v0.9.8
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/gorilla/mux v1.8.0
	github.com/onsi/gomega v1.14.0 // indirect
	github.com/pelletier/go-toml v1.8.1
	github.com/sirupsen/logrus v1.7.0
	github.com/streadway/amqp v1.0.0
	github.com/stretchr/testify v1.6.1
	github.com/wso2/product-microgateway/adapter v0.0.0
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616 // indirect
	golang.org/x/tools v0.1.4 // indirect
	google.golang.org/genproto v0.0.0-20201201144952-b05cb90ed32e
	google.golang.org/grpc v1.33.2
)
