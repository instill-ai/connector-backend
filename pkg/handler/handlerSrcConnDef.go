package handler

import (
	"context"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

func (h *handler) ListSourceConnectorDefinition(ctx context.Context, req *connectorPB.ListSourceConnectorDefinitionRequest) (*connectorPB.ListSourceConnectorDefinitionResponse, error) {
	resp, err := h.listConnectorDefinition(ctx, req)
	return resp.(*connectorPB.ListSourceConnectorDefinitionResponse), err
}

func (h *handler) GetSourceConnectorDefinition(ctx context.Context, req *connectorPB.GetSourceConnectorDefinitionRequest) (*connectorPB.GetSourceConnectorDefinitionResponse, error) {
	resp, err := h.getConnectorDefinition(ctx, req)
	return resp.(*connectorPB.GetSourceConnectorDefinitionResponse), err
}
