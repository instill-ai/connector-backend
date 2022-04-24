package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/connector-backend/pkg/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"

	connectorPB "github.com/instill-ai/protogen-go/connector/v1alpha"
)

type handler struct {
	connectorPB.UnimplementedConnectorServiceServer
	service service.Service
}

// NewHandler initiates a handler instance
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

func (h *handler) ListSourceDefinition(ctx context.Context, req *connectorPB.ListSourceDefinitionRequest) (*connectorPB.ListSourceDefinitionResponse, error) {

	dbSrcDefs, nextPageCursor, err := h.service.ListDefinitionByConnectorType(datamodel.ConnectorTypeSource, req.View, int(req.PageSize), req.PageCursor)
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

func (h *handler) GetSourceDefinition(ctx context.Context, req *connectorPB.GetSourceDefinitionRequest) (*connectorPB.GetSourceDefinitionResponse, error) {
	defUUID, err := uuid.FromString(req.Id)
	if err != nil {
		return &connectorPB.GetSourceDefinitionResponse{}, err
	}

	dbSrcDef, err := h.service.GetDefinition(defUUID, req.View)
	if err != nil {
		return &connectorPB.GetSourceDefinitionResponse{}, err
	}

	pbSrcDef := convertDBSourceDefinitionToPBSourceDefinition(dbSrcDef)
	resp := connectorPB.GetSourceDefinitionResponse{
		SourceDefinition: pbSrcDef,
	}
	return &resp, nil
}

func (h *handler) ListDestinationDefinition(ctx context.Context, req *connectorPB.ListDestinationDefinitionRequest) (*connectorPB.ListDestinationDefinitionResponse, error) {

	dbDstDefs, nextPageCursor, err := h.service.ListDefinitionByConnectorType(datamodel.ConnectorTypeDestination, req.View, int(req.PageSize), req.PageCursor)
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

func (h *handler) GetDestinationDefinition(ctx context.Context, req *connectorPB.GetDestinationDefinitionRequest) (*connectorPB.GetDestinationDefinitionResponse, error) {
	defUUID, err := uuid.FromString(req.Id)
	if err != nil {
		return &connectorPB.GetDestinationDefinitionResponse{}, err
	}

	dbDstDef, err := h.service.GetDefinition(defUUID, req.View)
	if err != nil {
		return &connectorPB.GetDestinationDefinitionResponse{}, err
	}
	pbDstDef := convertDBDestinationDefinitionToPBDestinationDefinition(dbDstDef)
	resp := connectorPB.GetDestinationDefinitionResponse{
		DestinationDefinition: pbDstDef,
	}
	return &resp, nil
}

func (h *handler) CreateConnector(ctx context.Context, req *connectorPB.CreateConnectorRequest) (*connectorPB.CreateConnectorResponse, error) {

	ownerID, err := getOwnerID(ctx)
	if err != nil {
		return &connectorPB.CreateConnectorResponse{}, err
	}

	connectiondefUUID, err := uuid.FromString(req.ConnectorDefinitionId)
	if err != nil {
		return &connectorPB.CreateConnectorResponse{}, err
	}

	configuration, err := protojson.Marshal(req.Configuration)
	if err != nil {
		return &connectorPB.CreateConnectorResponse{}, err
	}

	dbConnector := &datamodel.Connector{
		OwnerID:               ownerID,
		ConnectorDefinitionID: connectiondefUUID,
		Name:                  req.Name,
		Tombstone:             false,
		Configuration:         configuration,
		ConnectorType:         datamodel.ValidConnectorType(req.ConnectorType.String()),
	}

	dbConnector, err = h.service.CreateConnector(dbConnector)
	if err != nil {
		return &connectorPB.CreateConnectorResponse{}, err
	}

	pbConnector := convertDBConnectorToPBConnector(dbConnector)
	resp := connectorPB.CreateConnectorResponse{
		Connector: pbConnector,
	}

	// We need to manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		return &connectorPB.CreateConnectorResponse{}, err
	}

	return &resp, nil
}

func (h *handler) ListConnector(ctx context.Context, req *connectorPB.ListConnectorRequest) (*connectorPB.ListConnectorResponse, error) {
	ownerID, err := getOwnerID(ctx)
	if err != nil {
		return &connectorPB.ListConnectorResponse{}, err
	}

	dbConnectors, nextPageCursor, err := h.service.ListConnector(ownerID, datamodel.ValidConnectorType(req.ConnectorType.String()), int(req.PageSize), req.PageCursor)
	if err != nil {
		return &connectorPB.ListConnectorResponse{}, err
	}

	pbConnectors := []*connectorPB.Connector{}
	for _, dbConnector := range dbConnectors {
		pbConnectors = append(pbConnectors, convertDBConnectorToPBConnector(&dbConnector))
	}

	resp := connectorPB.ListConnectorResponse{
		Connectors:     pbConnectors,
		NextPageCursor: nextPageCursor,
	}
	return &resp, nil
}

func (h *handler) GetConnector(ctx context.Context, req *connectorPB.GetConnectorRequest) (*connectorPB.GetConnectorResponse, error) {
	ownerID, err := getOwnerID(ctx)
	if err != nil {
		return &connectorPB.GetConnectorResponse{}, err
	}

	dbConnector, err := h.service.GetConnector(
		ownerID,
		req.GetName(),
		datamodel.ValidConnectorType(req.GetConnectorType().String()))
	if err != nil {
		return &connectorPB.GetConnectorResponse{}, err
	}

	pbConnector := convertDBConnectorToPBConnector(dbConnector)
	resp := connectorPB.GetConnectorResponse{
		Connector: pbConnector,
	}

	return &resp, nil
}

func (h *handler) UpdateConnector(ctx context.Context, req *connectorPB.UpdateConnectorRequest) (*connectorPB.UpdateConnectorResponse, error) {

	ownerID, err := getOwnerID(ctx)
	if err != nil {
		return &connectorPB.UpdateConnectorResponse{}, err
	}

	dbConnector := &datamodel.Connector{
		OwnerID:       ownerID,
		ConnectorType: datamodel.ValidConnectorType(req.ConnectorType.String()),
	}

	if req.FieldMask != nil && len(req.FieldMask.Paths) > 0 {
		for _, field := range req.FieldMask.Paths {
			switch field {
			case "name":
				dbConnector.Name = req.ConnectorPatch.Name
			case "tombstone":
				dbConnector.Tombstone = req.ConnectorPatch.Tombstone
			}
			if strings.HasPrefix(field, "configuration") {
				configuration, err := protojson.Marshal(req.ConnectorPatch.Configuration)
				if err != nil {
					return &connectorPB.UpdateConnectorResponse{}, err
				}
				dbConnector.Configuration = configuration
			}
		}
	}

	dbConnector, err = h.service.UpdateConnector(ownerID, req.GetName(), datamodel.ValidConnectorType(req.GetConnectorType().String()), dbConnector)
	if err != nil {
		return nil, err
	}

	pbConnector := convertDBConnectorToPBConnector(dbConnector)
	resp := connectorPB.UpdateConnectorResponse{
		Connector: pbConnector,
	}

	return &resp, nil
}

func (h *handler) DeleteConnector(ctx context.Context, req *connectorPB.DeleteConnectorRequest) (*connectorPB.DeleteConnectorResponse, error) {
	ownerID, err := getOwnerID(ctx)
	if err != nil {
		return &connectorPB.DeleteConnectorResponse{}, err
	}

	if err := h.service.DeleteConnector(ownerID, req.GetName(), datamodel.ValidConnectorType(req.GetConnectorType().String())); err != nil {
		return &connectorPB.DeleteConnectorResponse{}, err
	}

	// We need to manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		return &connectorPB.DeleteConnectorResponse{}, err
	}

	return &connectorPB.DeleteConnectorResponse{}, nil
}
