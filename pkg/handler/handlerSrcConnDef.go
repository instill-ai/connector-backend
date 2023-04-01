package handler

import (
	"context"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

func (h *PublicHandler) ListSourceConnectorDefinitions(ctx context.Context, req *connectorPB.ListSourceConnectorDefinitionsRequest) (*connectorPB.ListSourceConnectorDefinitionsResponse, error) {
	resp, err := h.listConnectorDefinitions(ctx, req)
	return resp.(*connectorPB.ListSourceConnectorDefinitionsResponse), err
}

func (h *PublicHandler) GetSourceConnectorDefinition(ctx context.Context, req *connectorPB.GetSourceConnectorDefinitionRequest) (*connectorPB.GetSourceConnectorDefinitionResponse, error) {
	resp, err := h.getConnectorDefinition(ctx, req)
	return resp.(*connectorPB.GetSourceConnectorDefinitionResponse), err
}
