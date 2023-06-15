package main

import (
	"context"

	"go.opentelemetry.io/otel"

	"github.com/instill-ai/connector-backend/config"
	"github.com/instill-ai/connector-backend/pkg/logger"

	connectorDestinationAirbyte "github.com/instill-ai/connector-destination/pkg/airbyte"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	ctx, span := otel.Tracer("init-tracer").Start(ctx,
		"main",
	)
	defer span.End()
	defer cancel()

	logger, _ := logger.GetZapLogger(ctx)

	airbyte := connectorDestinationAirbyte.Init(logger, connectorDestinationAirbyte.ConnectorOptions{
		MountSourceVDP:     config.Config.Container.MountSource.VDP,
		MountTargetVDP:     config.Config.Container.MountTarget.VDP,
		MountSourceAirbyte: config.Config.Container.MountSource.Airbyte,
		MountTargetAirbyte: config.Config.Container.MountTarget.Airbyte,
		VDPProtocolPath:    "/etc/vdp/vdp_protocol.yaml",
	})

	err := airbyte.(*connectorDestinationAirbyte.Connector).PreDownloadImage(logger)

	if err != nil {
		panic(err)
	}
}
