package main

import (
	"context"
	"log"

	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"go.opentelemetry.io/otel"

	"github.com/instill-ai/connector-backend/config"
	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/connector-backend/pkg/logger"

	database "github.com/instill-ai/connector-backend/pkg/db"
	custom_otel "github.com/instill-ai/connector-backend/pkg/logger/otel"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

var enumRegistry = map[string]map[string]int32{
	"release_stage":                    connectorPB.ReleaseStage_value,
	"supported_destination_sync_modes": connectorPB.SupportedDestinationSyncModes_value,
	"auth_flow_type":                   connectorPB.AdvancedAuth_AuthFlowType_value,
}

func main() {

	if err := config.Init(); err != nil {
		log.Fatal(err.Error())
	}

	// setup tracing and metrics
	ctx, cancel := context.WithCancel(context.Background())

	if tp, err := custom_otel.SetupTracing(ctx, "connector-backend-init"); err != nil {
		panic(err)
	} else {
		defer tp.Shutdown(ctx)
	}

	if mp, err := custom_otel.SetupMetrics(ctx, "connector-backend-init"); err != nil {
		panic(err)
	} else {
		defer mp.Shutdown(ctx)
	}

	ctx, span := otel.Tracer("init-tracer").Start(ctx,
		"main",
	)
	defer span.End()
	defer cancel()

	logger, _ := logger.GetZapLogger(ctx)
	defer func() {
		// can't handle the error due to https://github.com/uber-go/zap/issues/880
		_ = logger.Sync()
	}()
	grpc_zap.ReplaceGrpcLoggerV2(logger)

	db := database.GetConnection()
	defer database.Close(db)

	datamodel.InitJSONSchema(ctx)

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
			if err := createSourceConnectorDefinition(ctx, db, srcConnDefs[idx], def, spec); err != nil {
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
			if err := createDestinationConnectorDefinition(ctx, db, dstConnDefs[idx], def, spec); err != nil {
				logger.Fatal(err.Error())
			}
		}
	}
}
