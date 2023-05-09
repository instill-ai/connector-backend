package handler

import (
	"context"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

func (h *PrivateHandler) ListDestinationConnectorsAdmin(ctx context.Context, req *connectorPB.ListDestinationConnectorsAdminRequest) (*connectorPB.ListDestinationConnectorsAdminResponse, error) {
	resp, err := h.listConnectors(ctx, req)
	return resp.(*connectorPB.ListDestinationConnectorsAdminResponse), err
}

func (h *PrivateHandler) LookUpDestinationConnectorAdmin(ctx context.Context, req *connectorPB.LookUpDestinationConnectorAdminRequest) (*connectorPB.LookUpDestinationConnectorAdminResponse, error) {
	resp, err := h.lookUpConnector(ctx, req)
	return resp.(*connectorPB.LookUpDestinationConnectorAdminResponse), err
}

func (h *PrivateHandler) CheckDestinationConnector(ctx context.Context, req *connectorPB.CheckDestinationConnectorRequest) (*connectorPB.CheckDestinationConnectorResponse, error) {
	resp, err := h.checkConnector(ctx, req)
	return resp.(*connectorPB.CheckDestinationConnectorResponse), err
}
