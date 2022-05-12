package handler

import (
	"context"
	"strings"

	"github.com/instill-ai/connector-backend/pkg/datamodel"

	connectorPB "github.com/instill-ai/protogen-go/connector/v1alpha"
)

func (h *handler) ListDestinationConnectorDefinition(ctx context.Context, req *connectorPB.ListDestinationConnectorDefinitionRequest) (*connectorPB.ListDestinationConnectorDefinitionResponse, error) {

	isBasicView := (req.GetView() == connectorPB.View_VIEW_BASIC) || (req.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
	dbDstDefs, nextPageToken, totalSize, err := h.service.ListConnectorDefinition(
		datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION),
		isBasicView,
		req.GetPageSize(),
		req.GetPageToken(),
	)
	if err != nil {
		return &connectorPB.ListDestinationConnectorDefinitionResponse{}, err
	}

	pbDstDefs := []*connectorPB.DestinationConnectorDefinition{}
	for _, dbDstDef := range dbDstDefs {
		pbDstDefs = append(
			pbDstDefs,
			DBToPBConnectorDefinition(dbDstDef, datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION)).(*connectorPB.DestinationConnectorDefinition))
	}

	resp := connectorPB.ListDestinationConnectorDefinitionResponse{
		DestinationConnectorDefinitions: pbDstDefs,
		NextPageToken:                   nextPageToken,
		TotalSize:                       totalSize,
	}

	return &resp, nil
}

func (h *handler) GetDestinationConnectorDefinition(ctx context.Context, req *connectorPB.GetDestinationConnectorDefinitionRequest) (*connectorPB.GetDestinationConnectorDefinitionResponse, error) {

	id := strings.TrimPrefix(req.GetName(), "destination-connector-definitions/")

	isBasicView := (req.GetView() == connectorPB.View_VIEW_BASIC) || (req.GetView() == connectorPB.View_VIEW_UNSPECIFIED)

	dbDstDef, err := h.service.GetConnectorDefinitionByID(id, datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION), isBasicView)
	if err != nil {
		return &connectorPB.GetDestinationConnectorDefinitionResponse{}, err
	}
	pbDstDef := DBToPBConnectorDefinition(dbDstDef, datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION)).(*connectorPB.DestinationConnectorDefinition)
	resp := connectorPB.GetDestinationConnectorDefinitionResponse{
		DestinationConnectorDefinition: pbDstDef,
	}
	return &resp, nil
}
