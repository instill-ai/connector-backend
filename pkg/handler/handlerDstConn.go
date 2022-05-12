package handler

import (
	"context"
	"net/http"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	connectorPB "github.com/instill-ai/protogen-go/connector/v1alpha"
)

func (h *handler) CreateDestinationConnector(ctx context.Context, req *connectorPB.CreateDestinationConnectorRequest) (*connectorPB.CreateDestinationConnectorResponse, error) {
	resp, err := h.createConnector(ctx, req)
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		return resp.(*connectorPB.CreateDestinationConnectorResponse), err
	}
	return resp.(*connectorPB.CreateDestinationConnectorResponse), err
}

func (h *handler) ListDestinationConnector(ctx context.Context, req *connectorPB.ListDestinationConnectorRequest) (*connectorPB.ListDestinationConnectorResponse, error) {
	resp, err := h.listConnector(ctx, req)
	return resp.(*connectorPB.ListDestinationConnectorResponse), err
}

func (h *handler) GetDestinationConnector(ctx context.Context, req *connectorPB.GetDestinationConnectorRequest) (*connectorPB.GetDestinationConnectorResponse, error) {
	resp, err := h.getConnector(ctx, req)
	return resp.(*connectorPB.GetDestinationConnectorResponse), err
}

func (h *handler) UpdateDestinationConnector(ctx context.Context, req *connectorPB.UpdateDestinationConnectorRequest) (*connectorPB.UpdateDestinationConnectorResponse, error) {
	resp, err := h.updateConnector(ctx, req)
	return resp.(*connectorPB.UpdateDestinationConnectorResponse), err
}

func (h *handler) DeleteDestinationConnector(ctx context.Context, req *connectorPB.DeleteDestinationConnectorRequest) (*connectorPB.DeleteDestinationConnectorResponse, error) {
	resp, err := h.deleteConnector(ctx, req)
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		return &connectorPB.DeleteDestinationConnectorResponse{}, err
	}
	return resp.(*connectorPB.DeleteDestinationConnectorResponse), err
}

func (h *handler) LookUpDestinationConnector(ctx context.Context, req *connectorPB.LookUpDestinationConnectorRequest) (*connectorPB.LookUpDestinationConnectorResponse, error) {
	resp, err := h.lookUpConnector(ctx, req)
	return resp.(*connectorPB.LookUpDestinationConnectorResponse), err
}

func (h *handler) RenameDestinationConnector(ctx context.Context, req *connectorPB.RenameDestinationConnectorRequest) (*connectorPB.RenameDestinationConnectorResponse, error) {
	resp, err := h.renameConnector(ctx, req)
	return resp.(*connectorPB.RenameDestinationConnectorResponse), err
}
