package handler

import (
	"context"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

func (h *publicHandler) ListSourceConnectorDefinitions(ctx context.Context, req *connectorPB.ListSourceConnectorDefinitionsRequest) (*connectorPB.ListSourceConnectorDefinitionsResponse, error) {
	resp, err := h.listConnectorDefinition(ctx, req)
	return resp.(*connectorPB.ListSourceConnectorDefinitionsResponse), err
}

func (h *publicHandler) GetSourceConnectorDefinition(ctx context.Context, req *connectorPB.GetSourceConnectorDefinitionRequest) (*connectorPB.GetSourceConnectorDefinitionResponse, error) {
	resp, err := h.getConnectorDefinition(ctx, req)
	return resp.(*connectorPB.GetSourceConnectorDefinitionResponse), err
}
