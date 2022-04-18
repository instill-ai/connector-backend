package handler

import (
	"context"

	"github.com/instill-ai/connector-backend/pkg/service"

	connectorPB "github.com/instill-ai/protogen-go/connector/v1alpha"
)

type handler struct {
	connectorPB.UnimplementedConnectorServiceServer
	service service.Service
}

func NewHandler(s service.Service) connectorPB.ConnectorServiceServer {
	return &handler{
		service: s,
	}
}

func (h *handler) Liveness(ctx context.Context, in *connectorPB.LivenessRequest) (*connectorPB.LivenessResponse, error) {
	return &connectorPB.LivenessResponse{
		HealthCheckResponse: &connectorPB.HealthCheckResponse{
			Status: connectorPB.HealthCheckResponse_SERVING_STATUS_SERVING,
		},
	}, nil
}

func (h *handler) Readiness(ctx context.Context, in *connectorPB.ReadinessRequest) (*connectorPB.ReadinessResponse, error) {
	return &connectorPB.ReadinessResponse{
		HealthCheckResponse: &connectorPB.HealthCheckResponse{
			Status: connectorPB.HealthCheckResponse_SERVING_STATUS_SERVING,
		},
	}, nil
}

func (h *handler) ListSourceDefinition(ctx context.Context, in *connectorPB.ListSourceDefinitionRequest) (*connectorPB.ListSourceDefinitionResponse, error) {

	dbSrcDefs, nextPageCursor, err := h.service.ListDefinitionByConnectorType(int(in.PageSize), in.PageCursor, "CONNECTOR_TYPE_SOURCE")
	if err != nil {
		return &connectorPB.ListSourceDefinitionResponse{}, err
	}

	pbSrcDefs := []*connectorPB.SourceDefinition{}
	for _, dbSrcDef := range dbSrcDefs {
		pbSrcDefs = append(pbSrcDefs, convertDBSourceDefinitionToPBSourceDefinition(&dbSrcDef))
	}

	resp := connectorPB.ListSourceDefinitionResponse{
		SourceDefinitions: pbSrcDefs,
		NextPageCursor:    nextPageCursor,
	}

	return &resp, nil
}

func (h *handler) ListDestinationDefinition(ctx context.Context, in *connectorPB.ListDestinationDefinitionRequest) (*connectorPB.ListDestinationDefinitionResponse, error) {

	dbDstDefs, nextPageCursor, err := h.service.ListDefinitionByConnectorType(int(in.PageSize), in.PageCursor, "CONNECTOR_TYPE_DESTINATION")
	if err != nil {
		return &connectorPB.ListDestinationDefinitionResponse{}, err
	}

	pbDstDefs := []*connectorPB.DestinationDefinition{}
	for _, dbDstDef := range dbDstDefs {
		pbDstDefs = append(pbDstDefs, convertDBDestinationDefinitionToPBDestinationDefinition(&dbDstDef))
	}

	resp := connectorPB.ListDestinationDefinitionResponse{
		DestinationDefinitions: pbDstDefs,
		NextPageCursor:         nextPageCursor,
	}

	return &resp, nil
}
