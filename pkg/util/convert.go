package util

import (
	"fmt"
)

func ConvertConnectorToResourceName(connectorName string) string {

	connectorTypeStr := "connectors"

	resourceName := fmt.Sprintf("resources/%s/types/%s", connectorName, connectorTypeStr)

	return resourceName
}
