package handler

import (
	"context"
	"fmt"

	"go.einride.tech/aip/filtering"
	"google.golang.org/genproto/googleapis/rpc/errdetails"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/connector-backend/internal/resource"
	"github.com/instill-ai/connector-backend/pkg/connector"
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

func (h *PrivateHandler) ListConnectorResourcesAdmin(ctx context.Context, req *connectorPB.ListConnectorResourcesAdminRequest) (resp *connectorPB.ListConnectorResourcesAdminResponse, err error) {

	var pageSize int64
	var pageToken string
	var isBasicView bool

	resp = &connectorPB.ListConnectorResourcesAdminResponse{}
	pageSize = req.GetPageSize()
	pageToken = req.GetPageToken()

	isBasicView = (req.GetView() == connectorPB.View_VIEW_BASIC) || (req.GetView() == connectorPB.View_VIEW_UNSPECIFIED)

	var connType connectorPB.ConnectorType
	declarations, err := filtering.NewDeclarations([]filtering.DeclarationOption{
		filtering.DeclareStandardFunctions(),
		filtering.DeclareEnumIdent("connector_type", connType.Type()),
	}...)
	if err != nil {
		return nil, err
	}
	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		return nil, err
	}

	connectorResources, totalSize, nextPageToken, err := h.service.ListConnectorResourcesAdmin(ctx, pageSize, pageToken, isBasicView, filter)
	if err != nil {
		return nil, err
	}

	resp.ConnectorResources = connectorResources
	resp.NextPageToken = nextPageToken
	resp.TotalSize = totalSize

	return resp, nil

}

func (h *PrivateHandler) LookUpConnectorResourceAdmin(ctx context.Context, req *connectorPB.LookUpConnectorResourceAdminRequest) (resp *connectorPB.LookUpConnectorResourceAdminResponse, err error) {

	logger, _ := logger.GetZapLogger(ctx)

	var isBasicView bool

	resp = &connectorPB.LookUpConnectorResourceAdminResponse{}

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
		return nil, st.Err()
	}

	connUID, err := resource.GetRscPermalinkUID(req.GetPermalink())
	if err != nil {
		return nil, err
	}

	isBasicView = (req.GetView() == connectorPB.View_VIEW_BASIC) || (req.GetView() == connectorPB.View_VIEW_UNSPECIFIED)

	connectorResource, err := h.service.GetConnectorResourceByUIDAdmin(ctx, connUID, isBasicView)
	if err != nil {
		return nil, err
	}

	resp.ConnectorResource = connectorResource

	return resp, nil
}

func (h *PrivateHandler) CheckConnectorResource(ctx context.Context, req *connectorPB.CheckConnectorResourceRequest) (resp *connectorPB.CheckConnectorResourceResponse, err error) {

	var isBasicView = true

	resp = &connectorPB.CheckConnectorResourceResponse{}
	connUID, err := resource.GetRscPermalinkUID(req.GetPermalink())
	if err != nil {
		return resp, err
	}

	connectorResource, err := h.service.GetConnectorResourceByUIDAdmin(ctx, connUID, isBasicView)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	if connectorResource.Tombstone {
		resp.State = connectorPB.ConnectorResource_STATE_ERROR
		return resp, nil
	}

	if connectorResource.State == connectorPB.ConnectorResource_STATE_CONNECTED {
		state, err := h.service.CheckConnectorResourceByUID(ctx, uuid.FromStringOrNil(connectorResource.Uid))
		if err != nil {
			return resp, err
		}

		resp.State = *state
		return resp, nil

	} else {
		resp.State = connectorPB.ConnectorResource_STATE_DISCONNECTED
		return resp, nil
	}

}

func (h *PrivateHandler) LookUpConnectorDefinitionAdmin(ctx context.Context, req *connectorPB.LookUpConnectorDefinitionAdminRequest) (resp *connectorPB.LookUpConnectorDefinitionAdminResponse, err error) {

	logger, _ := logger.GetZapLogger(ctx)

	resp = &connectorPB.LookUpConnectorDefinitionAdminResponse{}

	connUID, err := resource.GetRscPermalinkUID(req.GetPermalink())
	if err != nil {
		return resp, err
	}
	isBasicView := (req.GetView() == connectorPB.View_VIEW_BASIC) || (req.GetView() == connectorPB.View_VIEW_UNSPECIFIED)

	// TODO add a service wrapper
	def, err := h.connectors.GetConnectorDefinitionByUid(connUID)
	if err != nil {
		return resp, err
	}
	resp.ConnectorDefinition = def
	if isBasicView {
		resp.ConnectorDefinition.Spec = nil
	}
	resp.ConnectorDefinition.VendorAttributes = nil
	resp.ConnectorDefinition.Name = fmt.Sprintf("connector-definitions/%s", resp.ConnectorDefinition.GetId())

	logger.Info("GetConnectorDefinitionAdmin")

	return resp, nil
}
