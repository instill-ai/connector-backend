package handler

import (
	"context"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

func (h *publicHandler) ListDestinationConnectorDefinitions(ctx context.Context, req *connectorPB.ListDestinationConnectorDefinitionsRequest) (*connectorPB.ListDestinationConnectorDefinitionsResponse, error) {
	resp, err := h.listConnectorDefinition(ctx, req)
	return resp.(*connectorPB.ListDestinationConnectorDefinitionsResponse), err
}

func (h *publicHandler) GetDestinationConnectorDefinition(ctx context.Context, req *connectorPB.GetDestinationConnectorDefinitionRequest) (*connectorPB.GetDestinationConnectorDefinitionResponse, error) {
	resp, err := h.getConnectorDefinition(ctx, req)
	return resp.(*connectorPB.GetDestinationConnectorDefinitionResponse), err
}
