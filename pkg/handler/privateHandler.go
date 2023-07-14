package handler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"go.einride.tech/aip/filtering"
	"google.golang.org/genproto/googleapis/rpc/errdetails"

	"github.com/instill-ai/connector-backend/internal/resource"
	"github.com/instill-ai/connector-backend/pkg/connector"
	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/connector-backend/pkg/logger"
	"github.com/instill-ai/connector-backend/pkg/service"
	"github.com/instill-ai/x/checkfield"
	"github.com/instill-ai/x/sterr"

	connectorBase "github.com/instill-ai/connector/pkg/base"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

type PrivateHandler struct {
	connectorPB.UnimplementedConnectorPrivateServiceServer
	service    service.Service
	connectors connectorBase.IConnector
}

// NewPrivateHandler initiates a handler instance
func NewPrivateHandler(ctx context.Context, s service.Service) connectorPB.ConnectorPrivateServiceServer {
	logger, _ := logger.GetZapLogger(ctx)

	return &PrivateHandler{
		service:    s,
		connectors: connector.InitConnectorAll(logger),
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

func (h *PrivateHandler) ListConnectorsAdmin(ctx context.Context, req *connectorPB.ListConnectorsAdminRequest) (resp *connectorPB.ListConnectorsAdminResponse, err error) {

	var pageSize int64
	var pageToken string
	var isBasicView bool

	var connDefColID string

	resp = &connectorPB.ListConnectorsAdminResponse{}
	pageSize = req.GetPageSize()
	pageToken = req.GetPageToken()

	isBasicView = (req.GetView() == connectorPB.View_VIEW_BASIC) || (req.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
	connDefColID = "connector-definitions"

	var connType connectorPB.ConnectorType
	declarations, err := filtering.NewDeclarations([]filtering.DeclarationOption{
		filtering.DeclareStandardFunctions(),
		filtering.DeclareEnumIdent("connector_type", connType.Type()),
	}...)
	if err != nil {
		return resp, err
	}
	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		return resp, err
	}

	dbConnectors, totalSize, nextPageToken, err := h.service.ListConnectorsAdmin(ctx, pageSize, pageToken, isBasicView, filter)
	if err != nil {
		return resp, err
	}

	var pbConnectors []*connectorPB.Connector
	for idx := range dbConnectors {
		dbConnDef, err := h.connectors.GetConnectorDefinitionByUid(dbConnectors[idx].ConnectorDefinitionUID)
		if err != nil {
			return resp, err
		}
		pbConnector := DBToPBConnector(
			ctx,
			dbConnectors[idx],
			dbConnectors[idx].Owner,
			fmt.Sprintf("%s/%s", connDefColID, dbConnDef.GetId()),
		)
		if !isBasicView {
			pbConnector.ConnectorDefinition = dbConnDef
		}

		pbConnectors = append(
			pbConnectors,
			pbConnector,
		)
	}

	resp.Connectors = pbConnectors
	resp.NextPageToken = nextPageToken
	resp.TotalSize = totalSize

	return resp, nil

}

func (h *PrivateHandler) LookUpConnectorAdmin(ctx context.Context, req *connectorPB.LookUpConnectorAdminRequest) (resp *connectorPB.LookUpConnectorAdminResponse, err error) {

	logger, _ := logger.GetZapLogger(ctx)

	var isBasicView bool

	var connUID uuid.UUID

	var connDefColID string

	resp = &connectorPB.LookUpConnectorAdminResponse{}

	// Return error if REQUIRED fields are not provided in the requested payload
	if err := checkfield.CheckRequiredFields(req, lookUpRequiredFields); err != nil {
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

	connUIDStr, err := resource.GetPermalinkUID(req.GetPermalink())
	if err != nil {
		return resp, err
	}
	connUID, err = uuid.FromString(connUIDStr)
	if err != nil {
		return resp, err
	}

	isBasicView = (req.GetView() == connectorPB.View_VIEW_BASIC) || (req.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
	connDefColID = "connector-definitions"

	dbConnector, err := h.service.GetConnectorByUIDAdmin(ctx, connUID, isBasicView)
	if err != nil {
		return resp, err
	}

	dbConnDef, err := h.connectors.GetConnectorDefinitionByUid(dbConnector.ConnectorDefinitionUID)
	if err != nil {
		return resp, err
	}

	pbConnector := DBToPBConnector(
		ctx,
		dbConnector,
		dbConnector.Owner,
		fmt.Sprintf("%s/%s", connDefColID, dbConnDef.GetId()),
	)

	if !isBasicView {
		connector.MaskCredentialFields(h.connectors, dbConnDef.Id, pbConnector.Configuration)
		pbConnector.ConnectorDefinition = dbConnDef
	}
	resp.Connector = pbConnector

	return resp, nil
}

func (h *PrivateHandler) CheckConnector(ctx context.Context, req *connectorPB.CheckConnectorRequest) (resp *connectorPB.CheckConnectorResponse, err error) {

	var isBasicView = true

	var connUID string

	resp = &connectorPB.CheckConnectorResponse{}
	if connUID, err = resource.GetPermalinkUID(req.GetConnectorPermalink()); err != nil {
		return resp, err
	}

	dbConnector, err := h.service.GetConnectorByUIDAdmin(ctx, uuid.FromStringOrNil(connUID), isBasicView)
	if err != nil {
		return resp, err
	}

	if err != nil {
		return resp, err
	}

	if dbConnector.State == datamodel.ConnectorState(connectorPB.Connector_STATE_CONNECTED) {
		state, err := h.service.CheckConnectorByUID(ctx, dbConnector.UID)
		if err != nil {
			return resp, err
		}

		resp.State = *state
		return resp, nil

	} else {
		resp.State = connectorPB.Connector_STATE_DISCONNECTED
		return resp, nil
	}

}
