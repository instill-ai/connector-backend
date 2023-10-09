package connector

import (
	"fmt"

	"github.com/gofrs/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/connector-backend/config"

	connectorBase "github.com/instill-ai/component/pkg/base"
	connectorAI "github.com/instill-ai/connector-ai/pkg"
	connectorBlockchain "github.com/instill-ai/connector-blockchain/pkg"
	connectorData "github.com/instill-ai/connector-data/pkg"
	connectorDataAirbyte "github.com/instill-ai/connector-data/pkg/airbyte"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

const credentialMaskString = "*****MASK*****"

var connector connectorBase.IConnector

type Connector struct {
	connectorBase.Connector
	connectorUIDMap map[uuid.UUID]connectorBase.IConnector
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

	connector = &Connector{
		Connector:       connectorBase.Connector{Component: connectorBase.Component{Logger: logger}},
		connectorUIDMap: map[uuid.UUID]connectorBase.IConnector{},
	}

	connector.(*Connector).ImportDefinitions(connectorData.Init(logger, GetConnectorDataOptions()))
	connector.(*Connector).ImportDefinitions(connectorBlockchain.Init(logger))
	connector.(*Connector).ImportDefinitions(connectorAI.Init(logger))

	return connector
}

func (c *Connector) ImportDefinitions(con connectorBase.IConnector) {
	for _, v := range con.ListConnectorDefinitions() {
		err := c.AddConnectorDefinition(v)
		if err != nil {
			panic(err)
		}
		c.connectorUIDMap[uuid.FromStringOrNil(v.Uid)] = con
	}
}

func (c *Connector) CreateExecution(defUID uuid.UUID, task string, config *structpb.Struct, logger *zap.Logger) (connectorBase.IExecution, error) {
	return c.connectorUIDMap[defUID].CreateExecution(defUID, task, config, logger)
}

func (c *Connector) Test(defUid uuid.UUID, config *structpb.Struct, logger *zap.Logger) (connectorPB.ConnectorResource_State, error) {
	return c.connectorUIDMap[defUid].Test(defUid, config, logger)
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
