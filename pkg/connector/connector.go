package connector

import (
	"fmt"

	"github.com/gofrs/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/connector-backend/config"

	connectorAI "github.com/instill-ai/connector-ai/pkg"
	connectorBlockchain "github.com/instill-ai/connector-blockchain/pkg"
	connectorData "github.com/instill-ai/connector-data/pkg"
	connectorDataAirbyte "github.com/instill-ai/connector-data/pkg/airbyte"
	connectorBase "github.com/instill-ai/connector/pkg/base"
)

const credentialMaskString = "*****MASK*****"

type Connector struct {
	connectorBase.BaseConnector
	Data       connectorBase.IConnector
	Blockchain connectorBase.IConnector
	AI         connectorBase.IConnector
}

func GetConnectorDataOptions() connectorData.ConnectorOptions {
	return connectorData.ConnectorOptions{
		Airbyte: connectorDataAirbyte.ConnectorOptions{
			MountSourceVDP:        config.Config.Connector.Airbyte.MountSource.VDP,
			MountTargetVDP:        config.Config.Connector.Airbyte.MountTarget.VDP,
			MountSourceAirbyte:    config.Config.Connector.Airbyte.MountSource.Airbyte,
			MountTargetAirbyte:    config.Config.Connector.Airbyte.MountTarget.Airbyte,
			ExcludeLocalConnector: config.Config.Connector.Airbyte.ExcludeLocalConnector,
			VDPProtocolPath:       "/etc/vdp/vdp_protocol.yaml",
		},
	}

}

func InitConnectorAll(logger *zap.Logger) connectorBase.IConnector {

	connectorData := connectorData.Init(logger, GetConnectorDataOptions())
	connectorBlockchain := connectorBlockchain.Init(logger, connectorBlockchain.ConnectorOptions{})
	connectorAI := connectorAI.Init(logger, connectorAI.ConnectorOptions{})

	connector := &Connector{
		BaseConnector: connectorBase.BaseConnector{},
		Data:          connectorData,
		Blockchain:    connectorBlockchain,
		AI:            connectorAI,
	}

	for _, uid := range connectorData.ListConnectorDefinitionUids() {
		def, err := connectorData.GetConnectorDefinitionByUid(uid)
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
	for _, uid := range connectorAI.ListConnectorDefinitionUids() {
		def, err := connectorAI.GetConnectorDefinitionByUid(uid)
		if err != nil {
			logger.Error(err.Error())
		}
		err = connector.AddConnectorDefinition(uid, def.GetId(), def)
		if err != nil {
			logger.Error(err.Error())
		}
	}
	return connector
}

func (c *Connector) CreateConnection(defUid uuid.UUID, config *structpb.Struct, logger *zap.Logger) (connectorBase.IConnection, error) {
	switch {
	case c.Data.HasUid(defUid):
		return c.Data.CreateConnection(defUid, config, logger)
	case c.Blockchain.HasUid(defUid):
		return c.Blockchain.CreateConnection(defUid, config, logger)
	case c.AI.HasUid(defUid):
		return c.AI.CreateConnection(defUid, config, logger)

	default:
		return nil, fmt.Errorf("no connector uid: %s", defUid)
	}
}

func MaskCredentialFields(connector connectorBase.IConnector, defId string, config *structpb.Struct) {
	maskCredentialFields(connector, defId, config, "")
}

func maskCredentialFields(connector connectorBase.IConnector, defId string, config *structpb.Struct, prefix string) {

	for k, v := range config.GetFields() {
		key := prefix + k
		if connector.IsCredentialField(defId, key) {
			config.GetFields()[k] = structpb.NewStringValue(credentialMaskString)
		}
		if v.GetStructValue() != nil {
			maskCredentialFields(connector, defId, v.GetStructValue(), fmt.Sprintf("%s.", key))
		}

	}
}

func RemoveCredentialFieldsWithMaskString(connector connectorBase.IConnector, defId string, config *structpb.Struct) {
	removeCredentialFieldsWithMaskString(connector, defId, config, "")
}

func removeCredentialFieldsWithMaskString(connector connectorBase.IConnector, defId string, config *structpb.Struct, prefix string) {

	for k, v := range config.GetFields() {
		key := prefix + k
		if connector.IsCredentialField(defId, key) {
			if v.GetStringValue() == credentialMaskString {
				delete(config.GetFields(), k)
			}
		}
		if v.GetStructValue() != nil {
			removeCredentialFieldsWithMaskString(connector, defId, v.GetStructValue(), fmt.Sprintf("%s.", key))
		}

	}
}
