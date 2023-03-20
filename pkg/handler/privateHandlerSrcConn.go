package handler

import (
	"context"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

func (h *privateHandler) ListSourceConnectorsAdmin(ctx context.Context, req *connectorPB.ListSourceConnectorsAdminRequest) (*connectorPB.ListSourceConnectorsAdminResponse, error) {
	resp, err := h.listConnectors(ctx, req)
	return resp.(*connectorPB.ListSourceConnectorsAdminResponse), err
}

func (h *privateHandler) GetSourceConnectorAdmin(ctx context.Context, req *connectorPB.GetSourceConnectorAdminRequest) (*connectorPB.GetSourceConnectorAdminResponse, error) {
	resp, err := h.getConnector(ctx, req)
	return resp.(*connectorPB.GetSourceConnectorAdminResponse), err
}

func (h *privateHandler) LookUpSourceConnectorAdmin(ctx context.Context, req *connectorPB.LookUpSourceConnectorAdminRequest) (*connectorPB.LookUpSourceConnectorAdminResponse, error) {
	resp, err := h.lookUpConnector(ctx, req)
	return resp.(*connectorPB.LookUpSourceConnectorAdminResponse), err
}
