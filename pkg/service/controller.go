package service

import (
	"context"
	"fmt"
	"time"

	"github.com/instill-ai/connector-backend/pkg/datamodel"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	controllerPB "github.com/instill-ai/protogen-go/vdp/controller/v1alpha"
)

func convertConnectorToResourceName(
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

func (s *service) GetResourceState(connectorName string, connectorType datamodel.ConnectorType) (*connectorPB.Connector_State, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resourceName := convertConnectorToResourceName(connectorName, connectorType)

	resp, err := s.controllerClient.GetResource(ctx, &controllerPB.GetResourceRequest{
		Name: resourceName,
	})

	if err != nil {
		return nil, err
	}

	return resp.Resource.GetConnectorState().Enum(), nil
}

func (s *service) UpdateResourceState(connectorName string, connectorType datamodel.ConnectorType, state connectorPB.Connector_State, progress *int32, workflowId *string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resourceName := convertConnectorToResourceName(connectorName, connectorType)

	_, err := s.controllerClient.UpdateResource(ctx, &controllerPB.UpdateResourceRequest{
		Resource: &controllerPB.Resource{
			Name: resourceName,
			State: &controllerPB.Resource_ConnectorState{
				ConnectorState: state,
			},
			Progress: progress,
		},
		WorkflowId: workflowId,
	})

	if err != nil {
		return err
	}

	return nil
}

func (s *service) DeleteResourceState(connectorName string, connectorType datamodel.ConnectorType) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resourceName := convertConnectorToResourceName(connectorName, connectorType)

	_, err := s.controllerClient.DeleteResource(ctx, &controllerPB.DeleteResourceRequest{
		Name: resourceName,
	})

	if err != nil {
		return err
	}

	return nil
}
