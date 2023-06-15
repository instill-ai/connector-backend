package handler

import (
	"context"
	"net/http"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

func (h *PublicHandler) CreateSourceConnector(ctx context.Context, req *connectorPB.CreateSourceConnectorRequest) (*connectorPB.CreateSourceConnectorResponse, error) {
	resp, err := h.createConnector(ctx, req)
	if err != nil {
		return resp.(*connectorPB.CreateSourceConnectorResponse), err
	}

	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		return resp.(*connectorPB.CreateSourceConnectorResponse), err
	}

	return resp.(*connectorPB.CreateSourceConnectorResponse), nil
}

func (h *PublicHandler) ListSourceConnectors(ctx context.Context, req *connectorPB.ListSourceConnectorsRequest) (*connectorPB.ListSourceConnectorsResponse, error) {
	resp, err := h.listConnectors(ctx, req)
	return resp.(*connectorPB.ListSourceConnectorsResponse), err
}

func (h *PublicHandler) GetSourceConnector(ctx context.Context, req *connectorPB.GetSourceConnectorRequest) (*connectorPB.GetSourceConnectorResponse, error) {
	resp, err := h.getConnector(ctx, req)
	return resp.(*connectorPB.GetSourceConnectorResponse), err
}

func (h *PublicHandler) UpdateSourceConnector(ctx context.Context, req *connectorPB.UpdateSourceConnectorRequest) (*connectorPB.UpdateSourceConnectorResponse, error) {
	resp, err := h.updateConnector(ctx, req)
	return resp.(*connectorPB.UpdateSourceConnectorResponse), err
}

func (h *PublicHandler) DeleteSourceConnector(ctx context.Context, req *connectorPB.DeleteSourceConnectorRequest) (*connectorPB.DeleteSourceConnectorResponse, error) {
	resp, err := h.deleteConnector(ctx, req)
	if err != nil {
		return resp.(*connectorPB.DeleteSourceConnectorResponse), err
	}

	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		return resp.(*connectorPB.DeleteSourceConnectorResponse), err
	}
	return resp.(*connectorPB.DeleteSourceConnectorResponse), nil
}

func (h *PublicHandler) LookUpSourceConnector(ctx context.Context, req *connectorPB.LookUpSourceConnectorRequest) (*connectorPB.LookUpSourceConnectorResponse, error) {
	resp, err := h.lookUpConnector(ctx, req)
	return resp.(*connectorPB.LookUpSourceConnectorResponse), err
}

func (h *PublicHandler) ConnectSourceConnector(ctx context.Context, req *connectorPB.ConnectSourceConnectorRequest) (*connectorPB.ConnectSourceConnectorResponse, error) {
	resp, err := h.connectConnector(ctx, req)
	return resp.(*connectorPB.ConnectSourceConnectorResponse), err
}

func (h *PublicHandler) DisconnectSourceConnector(ctx context.Context, req *connectorPB.DisconnectSourceConnectorRequest) (*connectorPB.DisconnectSourceConnectorResponse, error) {
	resp, err := h.disconnectConnector(ctx, req)
	return resp.(*connectorPB.DisconnectSourceConnectorResponse), err
}

func (h *PublicHandler) RenameSourceConnector(ctx context.Context, req *connectorPB.RenameSourceConnectorRequest) (*connectorPB.RenameSourceConnectorResponse, error) {
	resp, err := h.renameConnector(ctx, req)
	return resp.(*connectorPB.RenameSourceConnectorResponse), err
}

func (h *PublicHandler) ExecuteSourceConnector(ctx context.Context, req *connectorPB.ExecuteSourceConnectorRequest) (*connectorPB.ExecuteSourceConnectorResponse, error) {
	resp, err := h.executeConnector(ctx, req)
	return resp.(*connectorPB.ExecuteSourceConnectorResponse), err
}

func (h *PublicHandler) WatchSourceConnector(ctx context.Context, req *connectorPB.WatchSourceConnectorRequest) (*connectorPB.WatchSourceConnectorResponse, error) {
	resp, err := h.watchConnector(ctx, req)
	return resp.(*connectorPB.WatchSourceConnectorResponse), err
}

func (h *PublicHandler) TestSourceConnector(ctx context.Context, req *connectorPB.TestSourceConnectorRequest) (*connectorPB.TestSourceConnectorResponse, error) {
	resp, err := h.testConnector(ctx, req)
	return resp.(*connectorPB.TestSourceConnectorResponse), err
}
