package handler

import (
	"context"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

func (h *PrivateHandler) ListDestinationConnectorsAdmin(ctx context.Context, req *connectorPB.ListDestinationConnectorsAdminRequest) (*connectorPB.ListDestinationConnectorsAdminResponse, error) {
	resp, err := h.listConnectors(ctx, req)
	return resp.(*connectorPB.ListDestinationConnectorsAdminResponse), err
}

func (h *PrivateHandler) GetDestinationConnectorAdmin(ctx context.Context, req *connectorPB.GetDestinationConnectorAdminRequest) (*connectorPB.GetDestinationConnectorAdminResponse, error) {
	resp, err := h.getConnector(ctx, req)
	return resp.(*connectorPB.GetDestinationConnectorAdminResponse), err
}

func (h *PrivateHandler) LookUpDestinationConnectorAdmin(ctx context.Context, req *connectorPB.LookUpDestinationConnectorAdminRequest) (*connectorPB.LookUpDestinationConnectorAdminResponse, error) {
	resp, err := h.lookUpConnector(ctx, req)
	return resp.(*connectorPB.LookUpDestinationConnectorAdminResponse), err
}

func (h *privateHandler) CheckDestinationConnector(ctx context.Context, req *connectorPB.CheckDestinationConnectorRequest) (*connectorPB.CheckDestinationConnectorResponse, error) {
	resp, err := h.CheckConnector(ctx, req)
	return resp.(*connectorPB.CheckDestinationConnectorResponse), err
}
