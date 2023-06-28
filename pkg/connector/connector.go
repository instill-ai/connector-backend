package connector

import (
	"fmt"

	"github.com/gofrs/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/connector-backend/config"

	// connectorAI "github.com/instill-ai/connector-ai/pkg"
	connectorBlockchain "github.com/instill-ai/connector-blockchain/pkg"
	connectorDestination "github.com/instill-ai/connector-destination/pkg"
	connectorDestinationAirbyte "github.com/instill-ai/connector-destination/pkg/airbyte"
	connectorSource "github.com/instill-ai/connector-source/pkg"
	connectorBase "github.com/instill-ai/connector/pkg/base"
)

type Connector struct {
	connectorBase.BaseConnector
	Destination connectorBase.IConnector
	Source      connectorBase.IConnector
	Blockchain  connectorBase.IConnector
	AI          connectorBase.IConnector
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
	}

}

func InitConnectorAll(logger *zap.Logger) connectorBase.IConnector {
	connectorSource := connectorSource.Init(logger)
	connectorDestination := connectorDestination.Init(logger, GetConnectorDestinationOptions())
	connectorBlockchain := connectorBlockchain.Init(logger, connectorBlockchain.ConnectorOptions{})
	// connectorAI := connectorAI.Init(logger, connectorAI.ConnectorOptions{})

	connector := &Connector{
		BaseConnector: connectorBase.BaseConnector{},
		Destination:   connectorDestination,
		Source:        connectorSource,
		Blockchain:    connectorBlockchain,
		// AI: 	  	   connectorAI,
	}

	for _, uid := range connectorDestination.ListConnectorDefinitionUids() {
		def, err := connectorDestination.GetConnectorDefinitionByUid(uid)
		if err != nil {
			logger.Error(err.Error())
		}
		err = connector.AddConnectorDefinition(uid, def.GetId(), def)
		if err != nil {
			logger.Error(err.Error())
		}
	}
	for _, uid := range connectorSource.ListConnectorDefinitionUids() {
		def, err := connectorSource.GetConnectorDefinitionByUid(uid)
		if err != nil {
			logger.Error(err.Error())
		}
		err = connector.AddConnectorDefinition(uid, def.GetId(), def)
		if err != nil {
			logger.Error(err.Error())
		}
	}
	for _, uid := range connectorBlockchain.ListConnectorDefinitionUids() {
		def, err := connectorBlockchain.GetConnectorDefinitionByUid(uid)
		if err != nil {
			logger.Error(err.Error())
		}
		err = connector.AddConnectorDefinition(uid, def.GetId(), def)
		if err != nil {
			logger.Error(err.Error())
		}
	}
	// for _, uid := range connectorAI.ListConnectorDefinitionUids() {
	// 	def, err := connectorAI.GetConnectorDefinitionByUid(uid)
	// 	if err != nil {
	// 		logger.Error(err.Error())
	// 	}
	// 	err = connector.AddConnectorDefinition(uid, def.GetId(), def)
	// 	if err != nil {
	// 		logger.Error(err.Error())
	// 	}
	// }
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
