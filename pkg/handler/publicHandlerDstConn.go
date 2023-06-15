package handler

import (
	"context"
	"net/http"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

func (h *PublicHandler) CreateDestinationConnector(ctx context.Context, req *connectorPB.CreateDestinationConnectorRequest) (*connectorPB.CreateDestinationConnectorResponse, error) {
	resp, err := h.createConnector(ctx, req)
	if err != nil {
		return resp.(*connectorPB.CreateDestinationConnectorResponse), err
	}

	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		return resp.(*connectorPB.CreateDestinationConnectorResponse), err
	}

	return resp.(*connectorPB.CreateDestinationConnectorResponse), nil
}

func (h *PublicHandler) ListDestinationConnectors(ctx context.Context, req *connectorPB.ListDestinationConnectorsRequest) (*connectorPB.ListDestinationConnectorsResponse, error) {
	resp, err := h.listConnectors(ctx, req)
	return resp.(*connectorPB.ListDestinationConnectorsResponse), err
}

func (h *PublicHandler) GetDestinationConnector(ctx context.Context, req *connectorPB.GetDestinationConnectorRequest) (*connectorPB.GetDestinationConnectorResponse, error) {
	resp, err := h.getConnector(ctx, req)
	return resp.(*connectorPB.GetDestinationConnectorResponse), err
}

func (h *PublicHandler) UpdateDestinationConnector(ctx context.Context, req *connectorPB.UpdateDestinationConnectorRequest) (*connectorPB.UpdateDestinationConnectorResponse, error) {
	resp, err := h.updateConnector(ctx, req)
	return resp.(*connectorPB.UpdateDestinationConnectorResponse), err
}

func (h *PublicHandler) DeleteDestinationConnector(ctx context.Context, req *connectorPB.DeleteDestinationConnectorRequest) (*connectorPB.DeleteDestinationConnectorResponse, error) {
	resp, err := h.deleteConnector(ctx, req)
	if err != nil {
		return resp.(*connectorPB.DeleteDestinationConnectorResponse), err
	}

	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		return resp.(*connectorPB.DeleteDestinationConnectorResponse), err
	}
	return resp.(*connectorPB.DeleteDestinationConnectorResponse), nil
}

func (h *PublicHandler) LookUpDestinationConnector(ctx context.Context, req *connectorPB.LookUpDestinationConnectorRequest) (*connectorPB.LookUpDestinationConnectorResponse, error) {
	resp, err := h.lookUpConnector(ctx, req)
	return resp.(*connectorPB.LookUpDestinationConnectorResponse), err
}

func (h *PublicHandler) ConnectDestinationConnector(ctx context.Context, req *connectorPB.ConnectDestinationConnectorRequest) (*connectorPB.ConnectDestinationConnectorResponse, error) {
	resp, err := h.connectConnector(ctx, req)
	return resp.(*connectorPB.ConnectDestinationConnectorResponse), err
}

func (h *PublicHandler) DisconnectDestinationConnector(ctx context.Context, req *connectorPB.DisconnectDestinationConnectorRequest) (*connectorPB.DisconnectDestinationConnectorResponse, error) {
	resp, err := h.disconnectConnector(ctx, req)
	return resp.(*connectorPB.DisconnectDestinationConnectorResponse), err
}

func (h *PublicHandler) RenameDestinationConnector(ctx context.Context, req *connectorPB.RenameDestinationConnectorRequest) (*connectorPB.RenameDestinationConnectorResponse, error) {
	resp, err := h.renameConnector(ctx, req)
	return resp.(*connectorPB.RenameDestinationConnectorResponse), err
}

func (h *PublicHandler) ExecuteDestinationConnector(ctx context.Context, req *connectorPB.ExecuteDestinationConnectorRequest) (*connectorPB.ExecuteDestinationConnectorResponse, error) {
	resp, err := h.executeConnector(ctx, req)
	return resp.(*connectorPB.ExecuteDestinationConnectorResponse), err
}

func (h *PublicHandler) WatchDestinationConnector(ctx context.Context, req *connectorPB.WatchDestinationConnectorRequest) (*connectorPB.WatchDestinationConnectorResponse, error) {
	resp, err := h.watchConnector(ctx, req)
	return resp.(*connectorPB.WatchDestinationConnectorResponse), err
}

func (h *PublicHandler) TestDestinationConnector(ctx context.Context, req *connectorPB.TestDestinationConnectorRequest) (*connectorPB.TestDestinationConnectorResponse, error) {
	resp, err := h.testConnector(ctx, req)
	return resp.(*connectorPB.TestDestinationConnectorResponse), err
}
