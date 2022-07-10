package main

import (
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"

	"github.com/instill-ai/connector-backend/config"
	"github.com/instill-ai/connector-backend/internal/logger"
	"github.com/instill-ai/connector-backend/pkg/datamodel"

	database "github.com/instill-ai/connector-backend/internal/db"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

var enumRegistry = map[string]map[string]int32{
	"release_stage":                    connectorPB.ReleaseStage_value,
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

	if err := config.Init(); err != nil {
		logger.Fatal(err.Error())
	}

	db := database.GetConnection()
	defer database.Close(db)

	datamodel.InitJSONSchema()

	srcConnDefs := []*connectorPB.SourceConnectorDefinition{}
	srcDefs := []*connectorPB.ConnectorDefinition{}
	dstConnDefs := []*connectorPB.DestinationConnectorDefinition{}
	dstDefs := []*connectorPB.ConnectorDefinition{}
	dockerImageSpecs := []*connectorPB.DockerImageSpec{}

	if err := loadDefinitionAndDockerImageSpecs(&srcConnDefs, &srcDefs, &dstConnDefs, &dstDefs, &dockerImageSpecs); err != nil {
		logger.Fatal(err.Error())
	}

	for idx, def := range srcDefs {

		var imgTag string
		if def.GetDockerImageTag() != "" {
			imgTag = ":" + def.GetDockerImageTag()
		} else {
			imgTag = def.GetDockerImageTag()
		}

		if spec, err := findDockerImageSpec(def.GetDockerRepository()+imgTag, &dockerImageSpecs); err != nil {
			logger.Fatal(err.Error())
		} else {
			// Create source definition record
			if err := createSourceConnectorDefinition(db, srcConnDefs[idx], def, spec); err != nil {
				logger.Fatal(err.Error())
			}
		}
	}

	for idx, def := range dstDefs {
		var imgTag string
		if def.GetDockerImageTag() != "" {
			imgTag = ":" + def.GetDockerImageTag()
		} else {
			imgTag = def.GetDockerImageTag()
		}
		if spec, err := findDockerImageSpec(def.GetDockerRepository()+imgTag, &dockerImageSpecs); err != nil {
			logger.Fatal(err.Error())
		} else {
			// Create destination definition record
			if err := createDestinationConnectorDefinition(db, dstConnDefs[idx], def, spec); err != nil {
				logger.Fatal(err.Error())
			}
		}
	}

}
