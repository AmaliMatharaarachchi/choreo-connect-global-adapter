package startup

import (
	"github.com/wso2-enterprise/choreo-connect-global-adapter/internal/apipartition"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/internal/database"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/internal/logger"
	"github.com/wso2-enterprise/choreo-connect-global-adapter/internal/types"
)

var newPartitionSize = 10

var apiList []types.API = []types.API{
	{UUID: "b2f19c9a-5ee3-4c76-ad55-c5f09b6435b3", GwLabel: []string{"dev", "prod", "stage"}, RevisionId: "", Context: "/shanakama/abc/v1", Version: "1.0.0"},
	{UUID: "480a5f46-33a8-4237-b0cf-f66ffc4fa9b3", GwLabel: []string{"dev"}, RevisionId: "", Context: "/shanakama/worldbank/v1", Version: "1.0.1"},
	{UUID: "56961ec7-75f1-4276-9184-01457f840f8e", GwLabel: []string{"dev"}, RevisionId: "", Context: "/shanakama/etender/api/v1", Version: "V1.0.0"},
	{UUID: "56961ec7-7561-4276-9184-01457f840f8e", GwLabel: []string{"dev"}, RevisionId: "", Context: "/shanakama/gotest/v1", Version: "1.3.0"},
	{UUID: "56971ec7-75f1-4276-9184-01457f840f8e", GwLabel: []string{"prod"}, RevisionId: "", Context: "/shanakama/test", Version: "1"},
	{UUID: "56961ec7-75f1-4276-9184-01457f840f44", GwLabel: []string{"dev"}, RevisionId: "", Context: "/shanakama/default/test/app", Version: "1.11.123"},
}

func Init() {
	database.ConnectToDb()
	pingError := database.DB.Ping()
	if pingError == nil {
		if database.CreateTable() { // no need to merge this condition
			for index, _ := range apiList {
				apipartition.PopulateApiData(&apiList[index])
			}
		}
	} else {
		logger.LoggerServer.Fatal("Error while initiating the database", pingError)
	}
}

func triggerDeploymentAgent() {
	var oldPartitionSize int
	query := "SELECT parition_size FROM la_partition_size"
	result, error := database.DB.Query(query)
	if error != nil {
		logger.LoggerServer.Error("[DB Error] Error when fetching partition size")
	} else {
		if result.Next() {
			result.Scan(&oldPartitionSize)
			if newPartitionSize != oldPartitionSize {
				logger.LoggerServer.Debug("Trigger a LA reboot event")
			} else {
				logger.LoggerServer.Debug("No config changes related to the partition size ")
			}
		}
	}
}
