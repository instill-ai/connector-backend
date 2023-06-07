package connector

import (
	"fmt"

	"github.com/gofrs/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/connector-backend/config"
	connectorDestination "github.com/instill-ai/connector-destination/pkg"
	connectorDestinationAirbyte "github.com/instill-ai/connector-destination/pkg/airbyte"
	connectorDestinationNumbers "github.com/instill-ai/connector-destination/pkg/numbers"
	connectorSource "github.com/instill-ai/connector-source/pkg"
	connectorBase "github.com/instill-ai/connector/pkg/base"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

type Connector struct {
	connectorBase.BaseConnector
	Destination connectorBase.IConnector
	Source      connectorBase.IConnector
}

func GetConnectorDestinationOptions() connectorDestination.ConnectorOptions {
	return connectorDestination.ConnectorOptions{
		Airbyte: connectorDestinationAirbyte.ConnectorOptions{
			MountSourceVDP:     config.Config.Container.MountSource.VDP,
			MountTargetVDP:     config.Config.Container.MountTarget.VDP,
			MountSourceAirbyte: config.Config.Container.MountSource.Airbyte,
			MountTargetAirbyte: config.Config.Container.MountTarget.Airbyte,
			VDPProtocolPath:    "/etc/vdp/vdp_protocol.yaml",
		},
		Numbers: connectorDestinationNumbers.ConnectorOptions{
			APIToken: "",
		},
	}

}

func InitConnectorAll(logger *zap.Logger) connectorBase.IConnector {
	connectorDestination := connectorDestination.Init(logger, GetConnectorDestinationOptions())
	connectorSource := connectorSource.Init(logger)

	connector := &Connector{
		BaseConnector: connectorBase.BaseConnector{},
		Destination:   connectorDestination,
		Source:        connectorSource,
	}

	for _, uid := range connectorDestination.ListConnectorDefinitionUids() {
		def, err := connectorDestination.GetConnectorDefinitionByUid(uid)
		if err != nil {
			logger.Error(err.Error())
		}
		err = connector.AddConnectorDefinition(uid, def.(*connectorPB.DestinationConnectorDefinition).GetId(), def)
		if err != nil {
			logger.Error(err.Error())
		}
	}
	for _, uid := range connectorSource.ListConnectorDefinitionUids() {
		def, err := connectorSource.GetConnectorDefinitionByUid(uid)
		if err != nil {
			logger.Error(err.Error())
		}
		err = connector.AddConnectorDefinition(uid, def.(*connectorPB.SourceConnectorDefinition).GetId(), def)
		if err != nil {
			logger.Error(err.Error())
		}
	}
	return connector
}

func (c *Connector) CreateConnection(defUid uuid.UUID, config *structpb.Struct, logger *zap.Logger) (connectorBase.IConnection, error) {
	switch {
	case c.Destination.HasUid(defUid):
		return c.Destination.CreateConnection(defUid, config, logger)
	case c.Source.HasUid(defUid):
		return c.Source.CreateConnection(defUid, config, logger)

	default:
		return nil, fmt.Errorf("no connector uid: %s", defUid)
	}
}
