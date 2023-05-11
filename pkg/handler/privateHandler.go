package handler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"google.golang.org/genproto/googleapis/rpc/errdetails"

	"github.com/instill-ai/connector-backend/internal/resource"
	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/connector-backend/pkg/logger"
	"github.com/instill-ai/connector-backend/pkg/service"
	"github.com/instill-ai/x/checkfield"
	"github.com/instill-ai/x/sterr"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

type PrivateHandler struct {
	connectorPB.UnimplementedConnectorPrivateServiceServer
	service service.Service
}

// NewPrivateHandler initiates a handler instance
func NewPrivateHandler(s service.Service) connectorPB.ConnectorPrivateServiceServer {
	datamodel.InitJSONSchema()
	datamodel.InitAirbyteCatalog()
	return &PrivateHandler{
		service: s,
	}
}

// GetService returns the service
func (h *PrivateHandler) GetService() service.Service {
	return h.service
}

// SetService sets the service
func (h *PrivateHandler) SetService(s service.Service) {
	h.service = s
}

func (h *PrivateHandler) listConnectors(ctx context.Context, req interface{}) (resp interface{}, err error) {
	var pageSize int64
	var pageToken string
	var isBasicView bool

	var connType datamodel.ConnectorType

	var connDefColID string

	switch v := req.(type) {
	case *connectorPB.ListSourceConnectorsAdminRequest:
		resp = &connectorPB.ListSourceConnectorsAdminResponse{}
		pageSize = v.GetPageSize()
		pageToken = v.GetPageToken()
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE)
		isBasicView = (v.GetView() == connectorPB.View_VIEW_BASIC) || (v.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
		connDefColID = "source-connector-definitions"
	case *connectorPB.ListDestinationConnectorsAdminRequest:
		resp = &connectorPB.ListDestinationConnectorsAdminResponse{}
		pageSize = v.GetPageSize()
		pageToken = v.GetPageToken()
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION)
		isBasicView = (v.GetView() == connectorPB.View_VIEW_BASIC) || (v.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
		connDefColID = "destination-connector-definitions"
	}

	dbConnectors, totalSize, nextPageToken, err := h.service.ListConnectorsAdmin(connType, pageSize, pageToken, isBasicView)
	if err != nil {
		return resp, err
	}

	var pbConnectors []interface{}
	for idx := range dbConnectors {
		dbConnDef, err := h.service.GetConnectorDefinitionByUID(dbConnectors[idx].ConnectorDefinitionUID, true)
		if err != nil {
			return resp, err
		}
		pbConnectors = append(
			pbConnectors,
			DBToPBConnector(
				dbConnectors[idx],
				connType,
				dbConnectors[idx].Owner,
				fmt.Sprintf("%s/%s", connDefColID, dbConnDef.ID),
			))
	}

	switch v := resp.(type) {
	case *connectorPB.ListSourceConnectorsAdminResponse:
		var pbSrcConns []*connectorPB.SourceConnector
		for _, pbConnector := range pbConnectors {
			pbSrcConns = append(pbSrcConns, pbConnector.(*connectorPB.SourceConnector))
		}
		v.SourceConnectors = pbSrcConns
		v.NextPageToken = nextPageToken
		v.TotalSize = totalSize
	case *connectorPB.ListDestinationConnectorsAdminResponse:
		var pbDstConns []*connectorPB.DestinationConnector
		for _, pbConnector := range pbConnectors {
			pbDstConns = append(pbDstConns, pbConnector.(*connectorPB.DestinationConnector))
		}
		v.DestinationConnectors = pbDstConns
		v.NextPageToken = nextPageToken
		v.TotalSize = totalSize
	}

	return resp, nil

}

func (h *PrivateHandler) lookUpConnector(ctx context.Context, req interface{}) (resp interface{}, err error) {

	logger, _ := logger.GetZapLogger()

	var isBasicView bool

	var connUID uuid.UUID
	var connType datamodel.ConnectorType

	var connDefColID string

	switch v := req.(type) {
	case *connectorPB.LookUpSourceConnectorAdminRequest:
		resp = &connectorPB.LookUpSourceConnectorAdminResponse{}

		// Return error if REQUIRED fields are not provided in the requested payload
		if err := checkfield.CheckRequiredFields(v, lookUpRequiredFields); err != nil {
			st, err := sterr.CreateErrorBadRequest(
				"[handler] lookup connector error",
				[]*errdetails.BadRequest_FieldViolation{
					{
						Field:       "REQUIRED fields",
						Description: err.Error(),
					},
				},
			)
			if err != nil {
				logger.Error(err.Error())
			}
			return resp, st.Err()
		}

		connUIDStr, err := resource.GetPermalinkUID(v.GetPermalink())
		if err != nil {
			return resp, err
		}
		connUID, err = uuid.FromString(connUIDStr)
		if err != nil {
			return resp, err
		}
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE)
		isBasicView = (v.GetView() == connectorPB.View_VIEW_BASIC) || (v.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
		connDefColID = "source-connector-definitions"
	case *connectorPB.LookUpDestinationConnectorAdminRequest:
		resp = &connectorPB.LookUpDestinationConnectorAdminResponse{}

		// Return error if REQUIRED fields are not provided in the requested payload
		if err := checkfield.CheckRequiredFields(v, lookUpRequiredFields); err != nil {
			st, err := sterr.CreateErrorBadRequest(
				"[handler] lookup connector error",
				[]*errdetails.BadRequest_FieldViolation{
					{
						Field:       "REQUIRED fields",
						Description: err.Error(),
					},
				},
			)
			if err != nil {
				logger.Error(err.Error())
			}
			return resp, st.Err()
		}

		connUIDStr, err := resource.GetPermalinkUID(v.GetPermalink())
		if err != nil {
			return resp, err
		}
		connUID, err = uuid.FromString(connUIDStr)
		if err != nil {
			return resp, err
		}
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION)
		isBasicView = (v.GetView() == connectorPB.View_VIEW_BASIC) || (v.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
		connDefColID = "destination-connector-definitions"
	}

	dbConnector, err := h.service.GetConnectorByUIDAdmin(connUID, isBasicView)
	if err != nil {
		return resp, err
	}

	dbConnDef, err := h.service.GetConnectorDefinitionByUID(dbConnector.ConnectorDefinitionUID, true)
	if err != nil {
		return resp, err
	}

	pbConnector := DBToPBConnector(
		dbConnector,
		connType,
		dbConnector.Owner,
		fmt.Sprintf("%s/%s", connDefColID, dbConnDef.ID),
	)

	switch v := resp.(type) {
	case *connectorPB.LookUpSourceConnectorAdminResponse:
		v.SourceConnector = pbConnector.(*connectorPB.SourceConnector)
	case *connectorPB.LookUpDestinationConnectorAdminResponse:
		v.DestinationConnector = pbConnector.(*connectorPB.DestinationConnector)
	}

	return resp, nil
}

func (h *PrivateHandler) checkConnector(ctx context.Context, req interface{}) (resp interface{}, err error) {
	var isBasicView = true

	var connUID string

	switch v := req.(type) {
	case *connectorPB.CheckSourceConnectorRequest:
		resp = &connectorPB.CheckSourceConnectorResponse{}
		if connUID, err = resource.GetPermalinkUID(v.GetSourceConnectorPermalink()); err != nil {
			return resp, err
		}
	case *connectorPB.CheckDestinationConnectorRequest:
		resp = &connectorPB.CheckDestinationConnectorResponse{}
		if connUID, err = resource.GetPermalinkUID(v.GetDestinationConnectorPermalink()); err != nil {
			return resp, err
		}
	}

	dbConnector, err := h.service.GetConnectorByUIDAdmin(uuid.FromStringOrNil(connUID), isBasicView)
	if err != nil {
		return resp, err
	}

	dbConnDef, err := h.service.GetConnectorDefinitionByUID(dbConnector.ConnectorDefinitionUID, true)
	if err != nil {
		return resp, err
	}

	if err != nil {
		return resp, err
	}

	wfId, err := h.service.CheckConnectorByUID(dbConnector.UID.String(), dbConnector.Owner, dbConnDef)

	if err != nil {
		return resp, err
	}

	switch v := resp.(type) {
	case *connectorPB.CheckSourceConnectorResponse:
		v.WorkflowId = *wfId
	case *connectorPB.CheckDestinationConnectorResponse:
		v.WorkflowId = *wfId
	}

	return resp, nil
}
