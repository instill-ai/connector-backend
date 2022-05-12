package handler

import (
	"context"
	"strings"

	"github.com/instill-ai/connector-backend/pkg/datamodel"

	connectorPB "github.com/instill-ai/protogen-go/connector/v1alpha"
)

func (h *handler) ListSourceConnectorDefinition(ctx context.Context, req *connectorPB.ListSourceConnectorDefinitionRequest) (*connectorPB.ListSourceConnectorDefinitionResponse, error) {

	isBasicView := (req.GetView() == connectorPB.View_VIEW_BASIC) || (req.GetView() == connectorPB.View_VIEW_UNSPECIFIED)

	dbSrcDefs, nextPageToken, totalSize, err := h.service.ListConnectorDefinition(
		datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE),
		isBasicView,
		req.GetPageSize(),
		req.GetPageToken(),
	)

	if err != nil {
		return &connectorPB.ListSourceConnectorDefinitionResponse{}, err
	}

	pbSrcDefs := []*connectorPB.SourceConnectorDefinition{}
	for _, dbSrcDef := range dbSrcDefs {
		pbSrcDefs = append(
			pbSrcDefs,
			DBToPBConnectorDefinition(dbSrcDef, datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE)).(*connectorPB.SourceConnectorDefinition))
	}

	resp := connectorPB.ListSourceConnectorDefinitionResponse{
		SourceConnectorDefinitions: pbSrcDefs,
		NextPageToken:              nextPageToken,
		TotalSize:                  totalSize,
	}

	return &resp, nil
}

func (h *handler) GetSourceConnectorDefinition(ctx context.Context, req *connectorPB.GetSourceConnectorDefinitionRequest) (*connectorPB.GetSourceConnectorDefinitionResponse, error) {

	id := strings.TrimPrefix(req.GetName(), "source-connector-definitions/")

	isBasicView := (req.GetView() == connectorPB.View_VIEW_BASIC) || (req.GetView() == connectorPB.View_VIEW_UNSPECIFIED)

	dbSrcDef, err := h.service.GetConnectorDefinitionByID(id, datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE), isBasicView)
	if err != nil {
		return &connectorPB.GetSourceConnectorDefinitionResponse{}, err
	}

	pbSrcDef := DBToPBConnectorDefinition(dbSrcDef, datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE))
	resp := connectorPB.GetSourceConnectorDefinitionResponse{
		SourceConnectorDefinition: pbSrcDef.(*connectorPB.SourceConnectorDefinition),
	}

	return &resp, nil
}
