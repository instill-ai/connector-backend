package util

import (
	"fmt"

	"github.com/instill-ai/connector-backend/pkg/datamodel"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

func ConvertConnectorToResourceName(
	connectorName string,
	connectorType datamodel.ConnectorType) string {
	var connectorTypeStr string
	switch connectorType {
	case datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE):
		connectorTypeStr = "source-connectors"
	case datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION):
		connectorTypeStr = "destination-connectors"
	}

	resourceName := fmt.Sprintf("resources/%s/types/%s", connectorName, connectorTypeStr)

	return resourceName
}
