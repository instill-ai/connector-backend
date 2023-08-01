package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/iancoleman/strcase"
	"go.einride.tech/aip/filtering"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/datatypes"

	fieldmask_utils "github.com/mennanov/fieldmask-utils"
	proto "google.golang.org/protobuf/proto"

	"github.com/instill-ai/connector-backend/internal/resource"
	"github.com/instill-ai/connector-backend/pkg/connector"
	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/connector-backend/pkg/logger"
	"github.com/instill-ai/connector-backend/pkg/repository"
	"github.com/instill-ai/connector-backend/pkg/service"
	"github.com/instill-ai/connector-backend/pkg/utils"
	"github.com/instill-ai/x/checkfield"
	"github.com/instill-ai/x/paginate"
	"github.com/instill-ai/x/sterr"

	custom_otel "github.com/instill-ai/connector-backend/pkg/logger/otel"
	connectorBase "github.com/instill-ai/connector/pkg/base"
	connectorConfigLoader "github.com/instill-ai/connector/pkg/configLoader"
	mgmtPB "github.com/instill-ai/protogen-go/base/mgmt/v1alpha"
	healthcheckPB "github.com/instill-ai/protogen-go/common/healthcheck/v1alpha"
	taskPB "github.com/instill-ai/protogen-go/common/task/v1alpha"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

var tracer = otel.Tracer("connector-backend.public-handler.tracer")

type PublicHandler struct {
	connectorPB.UnimplementedConnectorPublicServiceServer
	service    service.Service
	connectors connectorBase.IConnector
}

// NewPublicHandler initiates a handler instance
func NewPublicHandler(ctx context.Context, s service.Service) connectorPB.ConnectorPublicServiceServer {

	logger, _ := logger.GetZapLogger(ctx)
	return &PublicHandler{
		service:    s,
		connectors: connector.InitConnectorAll(logger),
	}
}

// GetService returns the service
func (h *PublicHandler) GetService() service.Service {
	return h.service
}

// SetService sets the service
func (h *PublicHandler) SetService(s service.Service) {
	h.service = s
}

func (h *PublicHandler) Liveness(ctx context.Context, in *connectorPB.LivenessRequest) (*connectorPB.LivenessResponse, error) {
	return &connectorPB.LivenessResponse{
		HealthCheckResponse: &healthcheckPB.HealthCheckResponse{
			Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_SERVING,
		},
	}, nil
}

func (h *PublicHandler) Readiness(ctx context.Context, in *connectorPB.ReadinessRequest) (*connectorPB.ReadinessResponse, error) {
	return &connectorPB.ReadinessResponse{
		HealthCheckResponse: &healthcheckPB.HealthCheckResponse{
			Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_SERVING,
		},
	}, nil
}

func (h *PublicHandler) ListConnectorDefinitions(ctx context.Context, req *connectorPB.ListConnectorDefinitionsRequest) (resp *connectorPB.ListConnectorDefinitionsResponse, err error) {
	ctx, span := tracer.Start(ctx, "ListConnectorDefinitions",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	resp = &connectorPB.ListConnectorDefinitionsResponse{}
	pageSize := req.GetPageSize()
	pageToken := req.GetPageToken()
	isBasicView := (req.GetView() == connectorPB.View_VIEW_BASIC) || (req.GetView() == connectorPB.View_VIEW_UNSPECIFIED)

	var connType connectorPB.ConnectorType
	declarations, err := filtering.NewDeclarations([]filtering.DeclarationOption{
		filtering.DeclareStandardFunctions(),
		filtering.DeclareEnumIdent("connector_type", connType.Type()),
	}...)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	prevLastUid := ""

	if pageToken != "" {
		_, prevLastUid, err = paginate.DecodeToken(pageToken)
		if err != nil {
			st, err := sterr.CreateErrorBadRequest(
				fmt.Sprintf("[db] list connector error: %s", err.Error()),
				[]*errdetails.BadRequest_FieldViolation{
					{
						Field:       "page_token",
						Description: fmt.Sprintf("Invalid page token: %s", err.Error()),
					},
				},
			)
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, st.Err()
		}
	}

	if pageSize == 0 {
		pageSize = repository.DefaultPageSize
	} else if pageSize > repository.MaxPageSize {
		pageSize = repository.MaxPageSize
	}

	unfilteredDefs := h.connectors.ListConnectorDefinitions()

	// don't return definition with tombstone = true
	unfilteredDefsRemoveTombstone := []*connectorPB.ConnectorDefinition{}
	for idx := range unfilteredDefs {
		if !unfilteredDefs[idx].Tombstone {
			unfilteredDefsRemoveTombstone = append(unfilteredDefsRemoveTombstone, unfilteredDefs[idx])
		}
	}
	unfilteredDefs = unfilteredDefsRemoveTombstone

	var defs []*connectorPB.ConnectorDefinition
	if filter.CheckedExpr != nil {
		trans := repository.NewTranspiler(filter)
		expr, _ := trans.Transpile()
		typeMap := map[string]bool{}
		for idx := range expr.Vars {
			if idx == 0 {
				typeMap[string(expr.Vars[idx].(protoreflect.Name))] = true
			} else {
				typeMap[string(expr.Vars[idx].([]interface{})[0].(protoreflect.Name))] = true
			}

		}
		for idx := range unfilteredDefs {
			if _, ok := typeMap[unfilteredDefs[idx].ConnectorType.String()]; ok {
				defs = append(defs, unfilteredDefs[idx])
			}
		}

	} else {
		defs = unfilteredDefs
	}

	startIdx := 0
	lastUid := ""
	for idx, def := range defs {
		if def.Uid == prevLastUid {
			startIdx = idx + 1
			break
		}
	}

	page := []*connectorPB.ConnectorDefinition{}
	for i := 0; i < int(pageSize) && startIdx+i < len(defs); i++ {
		def := proto.Clone(defs[startIdx+i]).(*connectorPB.ConnectorDefinition)
		page = append(page, def)
		lastUid = def.Uid
	}

	nextPageToken := ""

	if startIdx+len(page) < len(defs) {
		nextPageToken = paginate.EncodeToken(time.Time{}, lastUid)
	}
	for _, def := range page {
		def.Name = fmt.Sprintf("connector-definitions/%s", def.Id)
		if isBasicView {
			def.Spec = nil
		}
		def.VendorAttributes = nil
		resp.ConnectorDefinitions = append(
			resp.ConnectorDefinitions,
			def)
	}
	resp.NextPageToken = nextPageToken
	resp.TotalSize = int64(len(defs))

	logger.Info("ListConnectorDefinitions")

	return resp, nil
}

func (h *PublicHandler) GetConnectorDefinition(ctx context.Context, req *connectorPB.GetConnectorDefinitionRequest) (resp *connectorPB.GetConnectorDefinitionResponse, err error) {
	ctx, span := tracer.Start(ctx, "GetConnectorDefinition",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	resp = &connectorPB.GetConnectorDefinitionResponse{}

	var connID string

	if connID, err = resource.GetRscNameID(req.GetName()); err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	isBasicView := (req.GetView() == connectorPB.View_VIEW_BASIC) || (req.GetView() == connectorPB.View_VIEW_UNSPECIFIED)

	dbDef, err := h.connectors.GetConnectorDefinitionById(connID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	resp.ConnectorDefinition = proto.Clone(dbDef).(*connectorPB.ConnectorDefinition)
	if isBasicView {
		resp.ConnectorDefinition.Spec = nil
	}
	resp.ConnectorDefinition.VendorAttributes = nil
	resp.ConnectorDefinition.Name = fmt.Sprintf("connector-definitions/%s", resp.ConnectorDefinition.GetId())

	logger.Info("GetConnectorDefinition")
	return resp, nil

}

func (h *PublicHandler) CreateConnector(ctx context.Context, req *connectorPB.CreateConnectorRequest) (resp *connectorPB.CreateConnectorResponse, err error) {

	eventName := "CreateConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string
	var connDesc sql.NullString
	var connConfig datatypes.JSON

	var connDefUID uuid.UUID
	var connDefRscName string

	resp = &connectorPB.CreateConnectorResponse{}
	fmt.Println()
	fmt.Println(req)

	// Set all OUTPUT_ONLY fields to zero value on the requested payload
	if err := checkfield.CheckCreateOutputOnlyFields(req.GetConnector(), outputOnlyFields); err != nil {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] create connector error",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Field:       "OUTPUT_ONLY fields",
					Description: err.Error(),
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return resp, st.Err()
	}

	// Return error if REQUIRED fields are not provided in the requested payload
	if err := checkfield.CheckRequiredFields(req.GetConnector(), append(createRequiredFields, immutableFields...)); err != nil {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] create connector error",
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
		span.SetStatus(1, st.Err().Error())
		return resp, st.Err()
	}

	// TODO
	// Validate DestinationConnector JSON Schema
	configLoader := connectorConfigLoader.InitJSONSchema(logger)
	if err := connectorConfigLoader.ValidateJSONSchema(configLoader.ConnJSONSchema, req.GetConnector(), false); err != nil {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] create connector error",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Field:       "connector",
					Description: err.Error(),
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return resp, st.Err()
	}

	connID = req.GetConnector().GetId()
	if len(connID) > 8 && connID[:8] == "instill-" {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] create connector error",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Field:       "connector",
					Description: "the id can not start with instill-",
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return resp, st.Err()
	}

	connConfig, err = req.GetConnector().GetConfiguration().MarshalJSON()
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	connDesc = sql.NullString{
		String: req.GetConnector().GetDescription(),
		Valid:  len(req.GetConnector().GetDescription()) > 0,
	}

	connDefResp, err := h.GetConnectorDefinition(
		ctx,
		&connectorPB.GetConnectorDefinitionRequest{
			Name: req.GetConnector().GetConnectorDefinitionName(),
			View: connectorPB.View_VIEW_FULL.Enum(),
		})
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	// Validate Connector configuration JSON Schema
	// connSpec := connDefResp.GetConnectorDefinition().GetSpec().GetConnectionSpecification()
	// b, err := protojson.Marshal(connSpec)
	// if err != nil {
	// 	st, err := sterr.CreateErrorResourceInfo(
	// 		codes.Internal,
	// 		"[handler] create connector error",
	// 		"connector-definitions",
	// 		fmt.Sprintf("uid %s", connDefResp.ConnectorDefinition.GetUid()),
	// 		"",
	// 		err.Error(),
	// 	)
	// 	if err != nil {
	// 		logger.Error(err.Error())
	// 	}
	// 	span.SetStatus(1, st.Err().Error())
	// 	return resp, st.Err()
	// }

	// if err := connectorConfigLoader.ValidateJSONSchemaString(string(b), req.GetConnector().GetConfiguration()); err != nil {
	// 	st, err := sterr.CreateErrorBadRequest(
	// 		"[handler] create connector error",
	// 		[]*errdetails.BadRequest_FieldViolation{
	// 			{
	// 				Field:       "connector.configuration",
	// 				Description: err.Error(),
	// 			},
	// 		},
	// 	)
	// 	if err != nil {
	// 		logger.Error(err.Error())
	// 	}
	// 	span.SetStatus(1, st.Err().Error())
	// 	return resp, st.Err()
	// }

	connDefUID, err = uuid.FromString(connDefResp.ConnectorDefinition.GetUid())
	if err != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			"[handler] create connector error",
			"connector-definitions",
			fmt.Sprintf("uid %s", connDefResp.ConnectorDefinition.GetUid()),
			"",
			err.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return resp, st.Err()
	}

	connDefRscName = fmt.Sprintf("connector-definitions/%s", connDefResp.ConnectorDefinition.GetId())

	// Return error if resource ID does not follow RFC-1034
	if err := checkfield.CheckResourceID(connID); err != nil {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] create connector error",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Field:       "id",
					Description: err.Error(),
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return resp, st.Err()
	}

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbConnector := &datamodel.Connector{
		ID:                     connID,
		Owner:                  owner.GetName(),
		ConnectorDefinitionUID: connDefUID,
		Tombstone:              false,
		Configuration:          connConfig,
		ConnectorType:          datamodel.ConnectorType(connDefResp.ConnectorDefinition.GetConnectorType()),
		Description:            connDesc,
		Visibility:             datamodel.ConnectorVisibility(req.Connector.Visibility),
		Task:                   datamodel.Task(taskPB.Task_TASK_UNSPECIFIED),
	}

	dbConnector, err = h.service.CreateConnector(ctx, owner, dbConnector)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		owner,
		eventName,
		custom_otel.SetEventResource(dbConnector),
	)))

	pbConnector := DBToPBConnector(
		ctx,
		dbConnector,
		service.GenOwnerPermalink(owner),
		connDefRscName)

	connector.MaskCredentialFields(h.connectors, connDefResp.ConnectorDefinition.Id, pbConnector.Configuration)
	resp.Connector = pbConnector

	if err != nil {
		return resp, err
	}

	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		return resp, err
	}

	return resp, nil
}

func (h *PublicHandler) ListConnectors(ctx context.Context, req *connectorPB.ListConnectorsRequest) (resp *connectorPB.ListConnectorsResponse, err error) {

	eventName := "ListConnectors"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var pageSize int64
	var pageToken string
	var isBasicView bool

	var connDefColID string

	resp = &connectorPB.ListConnectorsResponse{}
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
		span.SetStatus(1, err.Error())
		return resp, err
	}
	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbConnectors, totalSize, nextPageToken, err := h.service.ListConnectors(ctx, owner, pageSize, pageToken, isBasicView, filter)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	var pbConnectors []*connectorPB.Connector
	for idx := range dbConnectors {
		dbConnDef, err := h.connectors.GetConnectorDefinitionByUid(dbConnectors[idx].ConnectorDefinitionUID)
		if err != nil {
			span.SetStatus(1, err.Error())
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
		connector.MaskCredentialFields(h.connectors, dbConnDef.GetId(), pbConnector.Configuration)
		pbConnectors = append(
			pbConnectors,
			pbConnector,
		)
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		owner,
		eventName,
	)))

	resp.Connectors = pbConnectors
	resp.NextPageToken = nextPageToken
	resp.TotalSize = totalSize

	return resp, nil

}

func (h *PublicHandler) GetConnector(ctx context.Context, req *connectorPB.GetConnectorRequest) (resp *connectorPB.GetConnectorResponse, err error) {
	return h.getConnector(ctx, req, true)
}

func (h *PublicHandler) getConnector(ctx context.Context, req *connectorPB.GetConnectorRequest, credentialMask bool) (resp *connectorPB.GetConnectorResponse, err error) {

	eventName := "GetConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var isBasicView bool

	var connID string

	var connDefColID string

	resp = &connectorPB.GetConnectorResponse{}
	if connID, err = resource.GetRscNameID(req.GetName()); err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	isBasicView = (req.GetView() == connectorPB.View_VIEW_BASIC) || (req.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
	connDefColID = "connector-definitions"

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbConnector, err := h.service.GetConnectorByID(ctx, connID, owner, isBasicView)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbConnDef, err := h.connectors.GetConnectorDefinitionByUid(dbConnector.ConnectorDefinitionUID)

	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	resp.Connector = DBToPBConnector(
		ctx,
		dbConnector,
		dbConnector.Owner,
		fmt.Sprintf("%s/%s", connDefColID, dbConnDef.GetId()),
	)

	if credentialMask {
		connector.MaskCredentialFields(h.connectors, dbConnDef.GetId(), resp.Connector.Configuration)
	}

	if !isBasicView {
		resp.Connector.ConnectorDefinition = dbConnDef
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		owner,
		eventName,
		custom_otel.SetEventResource(dbConnector),
	)))

	return resp, nil
}

func (h *PublicHandler) UpdateConnector(ctx context.Context, req *connectorPB.UpdateConnectorRequest) (resp *connectorPB.UpdateConnectorResponse, err error) {

	eventName := "UpdateConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var mask fieldmask_utils.Mask

	var connID string

	var connDefRscName string

	owner, ownerErr := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient())

	resp = &connectorPB.UpdateConnectorResponse{}
	if ownerErr != nil {
		span.SetStatus(1, ownerErr.Error())
		return resp, ownerErr
	}

	pbConnectorReq := req.GetConnector()
	pbUpdateMask := req.GetUpdateMask()

	// configuration filed is type google.protobuf.Struct, which needs to be updated as a whole
	for idx, path := range pbUpdateMask.Paths {
		if strings.Contains(path, "configuration") {
			pbUpdateMask.Paths[idx] = "configuration"
		}
	}

	if !pbUpdateMask.IsValid(req.GetConnector()) {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] update connector error",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Field:       "update_mask",
					Description: err.Error(),
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return resp, st.Err()
	}

	// Remove all OUTPUT_ONLY fields in the requested payload
	pbUpdateMask, err = checkfield.CheckUpdateOutputOnlyFields(pbUpdateMask, outputOnlyFields)
	if err != nil {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] update connector error",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Field:       "OUTPUT_ONLY fields",
					Description: err.Error(),
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return resp, st.Err()
	}

	getResp, err := h.getConnector(
		ctx,
		&connectorPB.GetConnectorRequest{
			Name: req.GetConnector().GetName(),
			View: connectorPB.View_VIEW_FULL.Enum(),
		}, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	// Return error if IMMUTABLE fields are intentionally changed
	if err := checkfield.CheckUpdateImmutableFields(req.GetConnector(), getResp.GetConnector(), immutableFields); err != nil {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] update connector error",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Field:       "IMMUTABLE fields",
					Description: err.Error(),
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return resp, st.Err()
	}

	mask, err = fieldmask_utils.MaskFromProtoFieldMask(pbUpdateMask, strcase.ToCamel)
	if err != nil {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] update connector error",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Field:       "update_mask",
					Description: err.Error(),
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return resp, st.Err()
	}

	if mask.IsEmpty() {
		return &connectorPB.UpdateConnectorResponse{
			Connector: getResp.GetConnector(),
		}, nil
	}

	pbConnectorToUpdate := getResp.GetConnector()
	if pbConnectorToUpdate.State == connectorPB.Connector_STATE_CONNECTED {
		st, err := sterr.CreateErrorPreconditionFailure(
			"[service] update connector",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "UPDATE",
					Subject:     fmt.Sprintf("id %s", req.Connector.Id),
					Description: fmt.Sprintf("Cannot update a connected %s connector", req.Connector.Id),
				},
			})
		if err != nil {
			logger.Error(err.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return nil, st.Err()
	}

	connID = getResp.GetConnector().GetId()

	dbConnDefID, err := resource.GetRscNameID(getResp.GetConnector().GetConnectorDefinitionName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbConnDef, err := h.connectors.GetConnectorDefinitionById(dbConnDefID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	connDefRscName = fmt.Sprintf("connector-definitions/%s", dbConnDef.GetId())

	if ownerErr != nil {
		span.SetStatus(1, err.Error())
		return resp, ownerErr
	}

	configuration := &structpb.Struct{}
	proto.Merge(configuration, pbConnectorToUpdate.Configuration)

	// Only the fields mentioned in the field mask will be copied to `pbPipelineToUpdate`, other fields are left intact
	err = fieldmask_utils.StructToStruct(mask, pbConnectorReq, pbConnectorToUpdate)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	connector.RemoveCredentialFieldsWithMaskString(h.connectors, dbConnDef.Id, req.Connector.Configuration)
	proto.Merge(configuration, req.Connector.Configuration)
	pbConnectorToUpdate.Configuration = configuration

	dbConnector, err := h.service.UpdateConnector(ctx, connID, owner, PBToDBConnector(ctx, pbConnectorToUpdate, owner.GetName(), dbConnDef))
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	resp.Connector = DBToPBConnector(
		ctx,
		dbConnector,
		service.GenOwnerPermalink(owner),
		connDefRscName)

	connector.MaskCredentialFields(h.connectors, dbConnDef.Id, resp.Connector.Configuration)
	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		owner,
		eventName,
		custom_otel.SetEventResource(dbConnector),
	)))
	return resp, nil
}

func (h *PublicHandler) DeleteConnector(ctx context.Context, req *connectorPB.DeleteConnectorRequest) (resp *connectorPB.DeleteConnectorResponse, err error) {

	eventName := "DeleteConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string

	// Cast all used types and data

	resp = &connectorPB.DeleteConnectorResponse{}
	if connID, err = resource.GetRscNameID(req.GetName()); err != nil {
		return resp, err
	}

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbConnector, err := h.service.GetConnectorByID(ctx, connID, owner, true)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	if err := h.service.DeleteConnector(ctx, connID, owner); err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		owner,
		eventName,
		custom_otel.SetEventResource(dbConnector),
	)))

	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		return resp, err
	}
	return resp, nil
}

func (h *PublicHandler) LookUpConnector(ctx context.Context, req *connectorPB.LookUpConnectorRequest) (resp *connectorPB.LookUpConnectorResponse, err error) {

	eventName := "LookUpConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var isBasicView bool

	var connUID uuid.UUID

	var connDefColID string

	resp = &connectorPB.LookUpConnectorResponse{}

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
		span.SetStatus(1, st.Err().Error())
		return resp, st.Err()
	}

	connUIDStr, err := resource.GetPermalinkUID(req.GetPermalink())
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	connUID, err = uuid.FromString(connUIDStr)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	isBasicView = (req.GetView() == connectorPB.View_VIEW_BASIC) || (req.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
	connDefColID = "connector-definitions"

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbConnector, err := h.service.GetConnectorByUID(ctx, connUID, owner, isBasicView)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbConnDef, err := h.connectors.GetConnectorDefinitionByUid(dbConnector.ConnectorDefinitionUID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		owner,
		eventName,
		custom_otel.SetEventResource(dbConnector),
	)))

	pbConnector := DBToPBConnector(
		ctx,
		dbConnector,
		dbConnector.Owner,
		fmt.Sprintf("%s/%s", connDefColID, dbConnDef.GetId()),
	)
	if !isBasicView {
		pbConnector.ConnectorDefinition = dbConnDef
	}

	resp.Connector = pbConnector

	return resp, nil
}

func (h *PublicHandler) ConnectConnector(ctx context.Context, req *connectorPB.ConnectConnectorRequest) (resp *connectorPB.ConnectConnectorResponse, err error) {

	eventName := "ConnectConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string

	var connDefRscName string

	resp = &connectorPB.ConnectConnectorResponse{}

	// Return error if REQUIRED fields are not provided in the requested payload
	if err := checkfield.CheckRequiredFields(req, connectRequiredFields); err != nil {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] connect connector error",
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
		span.SetStatus(1, st.Err().Error())
		return resp, st.Err()
	}

	connID, err = resource.GetRscNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	getResp, err := h.GetConnector(
		ctx,
		&connectorPB.GetConnectorRequest{
			Name: req.GetName(),
			View: connectorPB.View_VIEW_BASIC.Enum(),
		})
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbConnDefID, err := resource.GetRscNameID(getResp.GetConnector().GetConnectorDefinitionName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbConnDef, err := h.connectors.GetConnectorDefinitionById(dbConnDefID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	connDefRscName = fmt.Sprintf("connector-definitions/%s", dbConnDef.GetId())

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbConnector, err := h.service.GetConnectorByID(ctx, connID, owner, true)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	state, err := h.service.CheckConnectorByUID(ctx, dbConnector.UID)

	if err != nil {
		st, _ := sterr.CreateErrorBadRequest(
			fmt.Sprintf("[handler] connect connector error %v", err),
			[]*errdetails.BadRequest_FieldViolation{},
		)
		span.SetStatus(1, fmt.Sprintf("connect connector error %v", err))
		return resp, st.Err()
	}
	if *state != connectorPB.Connector_STATE_CONNECTED {
		st, _ := sterr.CreateErrorBadRequest(
			"[handler] connect connector error not Connector_STATE_CONNECTED",
			[]*errdetails.BadRequest_FieldViolation{},
		)
		span.SetStatus(1, "connect connector error not Connector_STATE_CONNECTED")
		return resp, st.Err()
	}

	dbConnector, err = h.service.UpdateConnectorState(ctx, connID, service.GenOwnerPermalink(owner), datamodel.ConnectorState(*state))
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		owner,
		eventName,
		custom_otel.SetEventResource(dbConnector),
	)))

	pbConnector := DBToPBConnector(
		ctx,
		dbConnector,
		dbConnector.Owner,
		connDefRscName,
	)

	resp.Connector = pbConnector

	return resp, nil
}

func (h *PublicHandler) DisconnectConnector(ctx context.Context, req *connectorPB.DisconnectConnectorRequest) (resp *connectorPB.DisconnectConnectorResponse, err error) {

	eventName := "DisconnectConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string

	var connDefRscName string

	resp = &connectorPB.DisconnectConnectorResponse{}

	// Return error if REQUIRED fields are not provided in the requested payload
	if err := checkfield.CheckRequiredFields(req, disconnectRequiredFields); err != nil {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] disconnect connector error",
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
		span.SetStatus(1, st.Err().Error())
		return resp, st.Err()
	}

	connID, err = resource.GetRscNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	getResp, err := h.GetConnector(
		ctx,
		&connectorPB.GetConnectorRequest{
			Name: req.GetName(),
			View: connectorPB.View_VIEW_BASIC.Enum(),
		})
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbConnDefID, err := resource.GetRscNameID(getResp.GetConnector().GetConnectorDefinitionName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbConnDef, err := h.connectors.GetConnectorDefinitionById(dbConnDefID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	connDefRscName = fmt.Sprintf("connector-definitions/%s", dbConnDef.GetId())

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbConnector, err := h.service.UpdateConnectorState(ctx, connID, service.GenOwnerPermalink(owner), datamodel.ConnectorState(connectorPB.Connector_STATE_DISCONNECTED))
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		owner,
		eventName,
		custom_otel.SetEventResource(dbConnector),
	)))

	pbConnector := DBToPBConnector(
		ctx,
		dbConnector,
		dbConnector.Owner,
		connDefRscName,
	)

	resp.Connector = pbConnector

	return resp, nil
}

func (h *PublicHandler) RenameConnector(ctx context.Context, req *connectorPB.RenameConnectorRequest) (resp *connectorPB.RenameConnectorResponse, err error) {

	eventName := "RenameConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string
	var connNewID string

	var connDefRscName string

	resp = &connectorPB.RenameConnectorResponse{}

	// Return error if REQUIRED fields are not provided in the requested payload
	if err := checkfield.CheckRequiredFields(req, renameRequiredFields); err != nil {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] rename connector error",
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
		span.SetStatus(1, st.Err().Error())
		return resp, st.Err()
	}

	connID, err = resource.GetRscNameID(req.GetName())
	if err != nil {
		return resp, err
	}
	connNewID = req.GetNewConnectorId()
	if len(connNewID) > 8 && connNewID[:8] == "instill-" {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] create connector error",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Field:       "connector",
					Description: "the id can not start with instill-",
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return resp, st.Err()
	}

	getResp, err := h.GetConnector(
		ctx,
		&connectorPB.GetConnectorRequest{
			Name: req.GetName(),
			View: connectorPB.View_VIEW_BASIC.Enum(),
		})
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbConnDefID, err := resource.GetRscNameID(getResp.GetConnector().GetConnectorDefinitionName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbConnDef, err := h.connectors.GetConnectorDefinitionById(dbConnDefID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	connDefRscName = fmt.Sprintf("connector-definitions/%s", dbConnDef.GetId())

	// Return error if resource ID does not follow RFC-1034
	if err := checkfield.CheckResourceID(connNewID); err != nil {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] rename connector error",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Field:       "id",
					Description: err.Error(),
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return resp, st.Err()
	}

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbConnector, err := h.service.UpdateConnectorID(ctx, connID, owner, connNewID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		owner,
		eventName,
		custom_otel.SetEventResource(dbConnector),
	)))

	resp.Connector = DBToPBConnector(
		ctx,
		dbConnector,
		dbConnector.Owner,
		connDefRscName,
	)

	return resp, nil
}

func (h *PublicHandler) WatchConnector(ctx context.Context, req *connectorPB.WatchConnectorRequest) (resp *connectorPB.WatchConnectorResponse, err error) {

	eventName := "WatchConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string

	resp = &connectorPB.WatchConnectorResponse{}
	if connID, err = resource.GetRscNameID(req.GetName()); err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		logger.Info(string(custom_otel.NewLogMessage(
			span,
			logUUID.String(),
			owner,
			eventName,
			custom_otel.SetErrorMessage(err.Error()),
		)))
		return resp, err
	}

	dbConnector, err := h.service.GetConnectorByID(ctx, connID, owner, true)
	if err != nil {
		span.SetStatus(1, err.Error())
		logger.Info(string(custom_otel.NewLogMessage(
			span,
			logUUID.String(),
			owner,
			eventName,
			custom_otel.SetErrorMessage(err.Error()),
		)))
		return resp, err
	}

	state, err := h.service.GetResourceState(dbConnector.UID)

	if err != nil {
		span.SetStatus(1, err.Error())
		logger.Info(string(custom_otel.NewLogMessage(
			span,
			logUUID.String(),
			owner,
			eventName,
			custom_otel.SetErrorMessage(err.Error()),
			custom_otel.SetEventResource(dbConnector),
		)))
		state = connectorPB.Connector_STATE_ERROR.Enum()
	}

	resp.State = *state

	return resp, nil
}

func (h *PublicHandler) TestConnector(ctx context.Context, req *connectorPB.TestConnectorRequest) (resp *connectorPB.TestConnectorResponse, err error) {

	eventName := "TestConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string

	resp = &connectorPB.TestConnectorResponse{}
	if connID, err = resource.GetRscNameID(req.GetName()); err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbConnector, err := h.service.GetConnectorByID(ctx, connID, owner, true)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	state, err := h.service.CheckConnectorByUID(ctx, dbConnector.UID)

	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		owner,
		eventName,
		custom_otel.SetEventResource(dbConnector),
	)))

	resp.State = *state

	return resp, nil
}

func (h *PublicHandler) ExecuteConnector(ctx context.Context, req *connectorPB.ExecuteConnectorRequest) (resp *connectorPB.ExecuteConnectorResponse, err error) {

	startTime := time.Now()
	eventName := "ExecuteConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	resp = &connectorPB.ExecuteConnectorResponse{}
	connID, err := resource.GetRscNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	connector, err := h.service.GetConnectorByID(ctx, connID, owner, false)
	if err != nil {
		return resp, err
	}
	if connector.Tombstone {
		st, _ := sterr.CreateErrorPreconditionFailure(
			"ExecuteConnector",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "STATE",
					Subject:     fmt.Sprintf("id %s", connID),
					Description: "the connector definition is deprecated, you can not use it anymore",
				},
			})
		return resp, st.Err()
	}

	dataPoint := utils.UsageMetricData{
		OwnerUID:               *owner.Uid,
		ConnectorID:            connector.ID,
		ConnectorUID:           connector.UID.String(),
		ConnectorExecuteUID:    logUUID.String(),
		ConnectorDefinitionUid: connector.ConnectorDefinitionUID.String(),
		ExecuteTime:            startTime.Format(time.RFC3339Nano),
	}

	if outputs, err := h.service.Execute(ctx, connector, owner, req.GetInputs()); err != nil {
		span.SetStatus(1, err.Error())
		dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
		dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
		_ = h.service.WriteNewDataPoint(ctx, dataPoint, req.GetInputs()[0].Metadata.GetFields()["pipeline"])
		return nil, err
	} else {
		resp.Outputs = outputs
		logger.Info(string(custom_otel.NewLogMessage(
			span,
			logUUID.String(),
			owner,
			eventName,
		)))
		dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
		dataPoint.Status = mgmtPB.Status_STATUS_COMPLETED
		if err := h.service.WriteNewDataPoint(ctx, dataPoint, req.GetInputs()[0].Metadata.GetFields()["pipeline"]); err != nil {
			logger.Warn("usage and metric data write fail")
		}
	}
	return resp, nil

}
