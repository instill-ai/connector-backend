package handler

import (
	"context"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

func (h *handler) ListDestinationConnectorDefinition(ctx context.Context, req *connectorPB.ListDestinationConnectorDefinitionRequest) (*connectorPB.ListDestinationConnectorDefinitionResponse, error) {
	resp, err := h.listConnectorDefinition(ctx, req)
	return resp.(*connectorPB.ListDestinationConnectorDefinitionResponse), err
}

func (h *handler) GetDestinationConnectorDefinition(ctx context.Context, req *connectorPB.GetDestinationConnectorDefinitionRequest) (*connectorPB.GetDestinationConnectorDefinitionResponse, error) {
	resp, err := h.getConnectorDefinition(ctx, req)
	return resp.(*connectorPB.GetDestinationConnectorDefinitionResponse), err
}
