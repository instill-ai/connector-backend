package main

import (
	"context"
	"log"

	"go.einride.tech/aip/filtering"
	"go.opentelemetry.io/otel"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/connector-backend/config"
	"github.com/instill-ai/connector-backend/pkg/logger"
	"github.com/instill-ai/connector-backend/pkg/repository"

	database "github.com/instill-ai/connector-backend/pkg/db"
	connectorDestinationAirbyte "github.com/instill-ai/connector-destination/pkg/airbyte"
)

func main() {

	if err := config.Init(); err != nil {
		log.Fatal(err.Error())
	}
	ctx, cancel := context.WithCancel(context.Background())
	ctx, span := otel.Tracer("init-tracer").Start(ctx,
		"main",
	)
	defer span.End()
	defer cancel()

	logger, _ := logger.GetZapLogger(ctx)

	db := database.GetConnection()
	defer database.Close(db)

	repository := repository.NewRepository(db)

	airbyte := connectorDestinationAirbyte.Init(logger, connectorDestinationAirbyte.ConnectorOptions{
		MountSourceVDP:     config.Config.Container.MountSource.VDP,
		MountTargetVDP:     config.Config.Container.MountTarget.VDP,
		MountSourceAirbyte: config.Config.Container.MountSource.Airbyte,
		MountTargetAirbyte: config.Config.Container.MountTarget.Airbyte,
		VDPProtocolPath:    "/etc/vdp/vdp_protocol.yaml",
	})

	// TODO: use pagination
	conns, _, _, err := repository.ListConnectorsAdmin(ctx, 1000, "", false, filtering.Filter{})
	if err != nil {
		panic(err)
	}

	airbyteConnector := airbyte.(*connectorDestinationAirbyte.Connector)
	var uids []uuid.UUID
	for idx := range conns {
		uid := conns[idx].ConnectorDefinitionUID
		if airbyteConnector.HasUid(uid) {
			uids = append(uids, uid)
		}
	}

	err = airbyteConnector.PreDownloadImage(logger, uids)

	if err != nil {
		panic(err)
	}
}
