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
	"source_type":                      connectorPB.SourceDefinition_SourceType_value,
	"destination_type":                 connectorPB.DestinationDefinition_DestinationType_value,
	"supported_destination_sync_modes": connectorPB.SupportedDestinationSyncModes_value,
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

	srcDefs := []*connectorPB.SourceDefinition{}
	dstDefs := []*connectorPB.DestinationDefinition{}
	dockerImageSpecs := []*connectorPB.DockerImageSpec{}

	if err := loadDefinitionAndDockerImageSpecs(&srcDefs, &dstDefs, &dockerImageSpecs); err != nil {
		logger.Fatal(err.Error())
	}

	for _, def := range srcDefs {
		if spec, err := findDockerImageSpec(def.GetDockerRepository()+":"+def.GetDockerImageTag(), &dockerImageSpecs); err != nil {
			logger.Fatal(err.Error())
		} else {
			// Create source definition record
			if err := createSourceConnectorDefinition(db, def, spec); err != nil {
				logger.Fatal(err.Error())
			}

			// Create source directness connector record
			if def.SourceType == connectorPB.SourceDefinition_SOURCE_TYPE_DIRECTNESS {
				if err := createSourceDirectnessConnector(db, def); err != nil {
					logger.Fatal(err.Error())
				}
			}
		}
	}

	for _, def := range dstDefs {
		if spec, err := findDockerImageSpec(def.GetDockerRepository()+":"+def.GetDockerImageTag(), &dockerImageSpecs); err != nil {
			logger.Fatal(err.Error())
		} else {
			if err := createDestinationConnectorDefinition(db, def, spec); err != nil {
				logger.Fatal(err.Error())
			}

			// Create source directness connector record
			if def.DestinationType == connectorPB.DestinationDefinition_DESTINATION_TYPE_DIRECTNESS {
				if err := createDestinationDirectnessConnector(db, def); err != nil {
					logger.Fatal(err.Error())
				}
			}
		}
	}

}
