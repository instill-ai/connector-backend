package service

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/connector-backend/pkg/util"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	controllerPB "github.com/instill-ai/protogen-go/vdp/controller/v1alpha"
)

func (s *service) GetResourceState(connectorUID uuid.UUID, connectorType datamodel.ConnectorType) (*connectorPB.Connector_State, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resourcePermalink := util.ConvertConnectorToResourceName(connectorUID.String(), connectorType)

	resp, err := s.controllerClient.GetResource(ctx, &controllerPB.GetResourceRequest{
		ResourcePermalink: resourcePermalink,
	})

	if err != nil {
		return nil, err
	}

	return resp.Resource.GetConnectorState().Enum(), nil
}

func (s *service) UpdateResourceState(connectorUID uuid.UUID, connectorType datamodel.ConnectorType, state connectorPB.Connector_State, progress *int32) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resourcePermalink := util.ConvertConnectorToResourceName(connectorUID.String(), connectorType)

	_, err := s.controllerClient.UpdateResource(ctx, &controllerPB.UpdateResourceRequest{
		Resource: &controllerPB.Resource{
			ResourcePermalink: resourcePermalink,
			State: &controllerPB.Resource_ConnectorState{
				ConnectorState: state,
			},
			Progress: progress,
		},
		WorkflowId: nil,
	})

	if err != nil {
		return err
	}

	return nil
}

func (s *service) DeleteResourceState(connectorUID uuid.UUID, connectorType datamodel.ConnectorType) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resourcePermalink := util.ConvertConnectorToResourceName(connectorUID.String(), connectorType)

	_, err := s.controllerClient.DeleteResource(ctx, &controllerPB.DeleteResourceRequest{
		ResourcePermalink: resourcePermalink,
	})

	if err != nil {
		return err
	}

	return nil
}
