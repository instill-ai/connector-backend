package handler

import (
	"context"
	"net/http"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

func (h *publicHandler) CreateSourceConnector(ctx context.Context, req *connectorPB.CreateSourceConnectorRequest) (*connectorPB.CreateSourceConnectorResponse, error) {
	resp, err := h.createConnector(ctx, req)
	if err != nil {
		return resp.(*connectorPB.CreateSourceConnectorResponse), err
	}

	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		return resp.(*connectorPB.CreateSourceConnectorResponse), err
	}

	return resp.(*connectorPB.CreateSourceConnectorResponse), nil
}

func (h *publicHandler) ListSourceConnectors(ctx context.Context, req *connectorPB.ListSourceConnectorsRequest) (*connectorPB.ListSourceConnectorsResponse, error) {
	resp, err := h.listConnector(ctx, req)
	return resp.(*connectorPB.ListSourceConnectorsResponse), err
}

func (h *publicHandler) GetSourceConnector(ctx context.Context, req *connectorPB.GetSourceConnectorRequest) (*connectorPB.GetSourceConnectorResponse, error) {
	resp, err := h.getConnector(ctx, req)
	return resp.(*connectorPB.GetSourceConnectorResponse), err
}

func (h *publicHandler) UpdateSourceConnector(ctx context.Context, req *connectorPB.UpdateSourceConnectorRequest) (*connectorPB.UpdateSourceConnectorResponse, error) {
	resp, err := h.updateConnector(ctx, req)
	return resp.(*connectorPB.UpdateSourceConnectorResponse), err
}

func (h *publicHandler) DeleteSourceConnector(ctx context.Context, req *connectorPB.DeleteSourceConnectorRequest) (*connectorPB.DeleteSourceConnectorResponse, error) {
	resp, err := h.deleteConnector(ctx, req)
	if err != nil {
		return resp.(*connectorPB.DeleteSourceConnectorResponse), err
	}

	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		return resp.(*connectorPB.DeleteSourceConnectorResponse), err
	}
	return resp.(*connectorPB.DeleteSourceConnectorResponse), nil
}

func (h *publicHandler) LookUpSourceConnector(ctx context.Context, req *connectorPB.LookUpSourceConnectorRequest) (*connectorPB.LookUpSourceConnectorResponse, error) {
	resp, err := h.lookUpConnector(ctx, req)
	return resp.(*connectorPB.LookUpSourceConnectorResponse), err
}

func (h *publicHandler) ConnectSourceConnector(ctx context.Context, req *connectorPB.ConnectSourceConnectorRequest) (*connectorPB.ConnectSourceConnectorResponse, error) {
	resp, err := h.connectConnector(ctx, req)
	return resp.(*connectorPB.ConnectSourceConnectorResponse), err
}

func (h *publicHandler) DisconnectSourceConnector(ctx context.Context, req *connectorPB.DisconnectSourceConnectorRequest) (*connectorPB.DisconnectSourceConnectorResponse, error) {
	resp, err := h.disconnectConnector(ctx, req)
	return resp.(*connectorPB.DisconnectSourceConnectorResponse), err
}

func (h *publicHandler) RenameSourceConnector(ctx context.Context, req *connectorPB.RenameSourceConnectorRequest) (*connectorPB.RenameSourceConnectorResponse, error) {
	resp, err := h.renameConnector(ctx, req)
	return resp.(*connectorPB.RenameSourceConnectorResponse), err
}

func (h *publicHandler) ReadSourceConnector(context.Context, *connectorPB.ReadSourceConnectorRequest) (*connectorPB.ReadSourceConnectorResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReadSourceConnector not implemented")
}
