package main

import (
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"

	"github.com/instill-ai/connector-backend/configs"
	"github.com/instill-ai/connector-backend/internal/logger"

	database "github.com/instill-ai/connector-backend/internal/db"
	connectorPB "github.com/instill-ai/protogen-go/connector/v1alpha"
)

var enumRegistry = map[string]map[string]int32{
	"release_stage":                    connectorPB.ReleaseStage_value,
	"connection_type":                  connectorPB.ConnectionType_value,
	"supported_destination_sync_modes": connectorPB.Spec_SupportedDestinationSyncModes_value,
	"auth_flow_type":                   connectorPB.AdvancedAuth_AuthFlowType_value,
}

func main() {

	logger, _ := logger.GetZapLogger()
	defer func() {
		// can't handle the error due to https://github.com/uber-go/zap/issues/880
		_ = logger.Sync()
	}()
	grpc_zap.ReplaceGrpcLoggerV2(logger)

	if err := configs.Init(); err != nil {
		logger.Fatal(err.Error())
	}

	db := database.GetConnection()
	defer database.Close(db)

	srcConnDefs := []*connectorPB.SourceConnectorDefinition{}
	srcDefs := []*connectorPB.ConnectorDefinition{}
	dstConnDefs := []*connectorPB.DestinationConnectorDefinition{}
	dstDefs := []*connectorPB.ConnectorDefinition{}
	dockerImageSpecs := []*connectorPB.DockerImageSpec{}

	if err := loadDefinitionAndDockerImageSpecs(&srcConnDefs, &srcDefs, &dstConnDefs, &dstDefs, &dockerImageSpecs); err != nil {
		logger.Fatal(err.Error())
	}

	for idx, def := range srcDefs {
		spec, err := findDockerImageSpec(def.GetDockerRepository()+":"+def.GetDockerImageTag(), &dockerImageSpecs)
		if err != nil {
			logger.Fatal(err.Error())
		}
		// Create source definition record
		if err := createSourceConnectorDefinition(db, srcConnDefs[idx], def, spec); err != nil {
			logger.Fatal(err.Error())
		}
	}

	for idx, def := range dstDefs {
		if spec, err := findDockerImageSpec(def.GetDockerRepository()+":"+def.GetDockerImageTag(), &dockerImageSpecs); err != nil {
			logger.Fatal(err.Error())
		} else {
			if err := createDestinationConnectorDefinition(db, dstConnDefs[idx], def, spec); err != nil {
				logger.Fatal(err.Error())
			}
		}
	}

}
