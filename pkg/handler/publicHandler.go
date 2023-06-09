package handler

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/iancoleman/strcase"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/datatypes"

	fieldmask_utils "github.com/mennanov/fieldmask-utils"
	proto "google.golang.org/protobuf/proto"

	"github.com/instill-ai/connector-backend/internal/resource"
	"github.com/instill-ai/connector-backend/pkg/connector"
	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/connector-backend/pkg/logger"
	"github.com/instill-ai/connector-backend/pkg/repository"
	"github.com/instill-ai/connector-backend/pkg/service"
	"github.com/instill-ai/x/checkfield"
	"github.com/instill-ai/x/paginate"
	"github.com/instill-ai/x/sterr"

	custom_otel "github.com/instill-ai/connector-backend/pkg/logger/otel"
	connectorDestination "github.com/instill-ai/connector-destination/pkg"
	connectorSource "github.com/instill-ai/connector-source/pkg"
	connectorBase "github.com/instill-ai/connector/pkg/base"
	connectorConfigLoader "github.com/instill-ai/connector/pkg/configLoader"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	healthcheckPB "github.com/instill-ai/protogen-go/vdp/healthcheck/v1alpha"
)

var tracer = otel.Tracer("connector-backend.public-handler.tracer")

type PublicHandler struct {
	connectorPB.UnimplementedConnectorPublicServiceServer
	service              service.Service
	connectorAll         connectorBase.IConnector
	connectorSource      connectorBase.IConnector
	connectorDestination connectorBase.IConnector
}

// NewPublicHandler initiates a handler instance
func NewPublicHandler(ctx context.Context, s service.Service) connectorPB.ConnectorPublicServiceServer {

	logger, _ := logger.GetZapLogger(ctx)
	return &PublicHandler{
		service:              s,
		connectorAll:         connector.InitConnectorAll(logger),
		connectorSource:      connectorSource.Init(logger),
		connectorDestination: connectorDestination.Init(logger, connector.GetConnectorDestinationOptions()),
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

func (h *PublicHandler) listConnectorDefinitions(ctx context.Context, req interface{}) (resp interface{}, err error) {

	ctx, span := tracer.Start(ctx, "ListConnectorDefinitions",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	var pageSize int64
	var pageToken string
	var isBasicView bool

	switch v := req.(type) {
	case *connectorPB.ListSourceConnectorDefinitionsRequest:
		resp = &connectorPB.ListSourceConnectorDefinitionsResponse{}
		pageSize = v.GetPageSize()
		pageToken = v.GetPageToken()
		isBasicView = (v.GetView() == connectorPB.View_VIEW_BASIC) || (v.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
	case *connectorPB.ListDestinationConnectorDefinitionsRequest:
		resp = &connectorPB.ListDestinationConnectorDefinitionsResponse{}
		pageSize = v.GetPageSize()
		pageToken = v.GetPageToken()
		isBasicView = (v.GetView() == connectorPB.View_VIEW_BASIC) || (v.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
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

	// TODO: refactor
	switch v := resp.(type) {
	case *connectorPB.ListSourceConnectorDefinitionsResponse:
		defs := h.connectorSource.ListConnectorDefinitions()

		startIdx := 0
		lastUid := ""
		for idx, def := range defs {
			if def.(*connectorPB.SourceConnectorDefinition).Uid == prevLastUid {
				startIdx = idx + 1
				break
			}
		}

		page := []*connectorPB.SourceConnectorDefinition{}
		for i := 0; i < int(pageSize) && startIdx+i < len(defs); i++ {
			def := proto.Clone(defs[startIdx+i].(*connectorPB.SourceConnectorDefinition)).(*connectorPB.SourceConnectorDefinition)
			page = append(page, def)
			lastUid = def.Uid
		}

		nextPageToken := ""

		if startIdx+len(page) < len(defs) {
			nextPageToken = paginate.EncodeToken(time.Time{}, lastUid)
		}

		for _, def := range page {
			def.Name = fmt.Sprintf("source-connector-definitions/%s", def.Id)
			def.ConnectorDefinition.ReleaseDate = func() *date.Date {
				if def.ConnectorDefinition.ReleaseDate != nil {
					return &date.Date{
						Year:  int32(def.ConnectorDefinition.ReleaseDate.Year),
						Month: int32(def.ConnectorDefinition.ReleaseDate.Month),
						Day:   int32(def.ConnectorDefinition.ReleaseDate.Day),
					}
				}
				return &date.Date{}
			}()
			if isBasicView {
				def.GetConnectorDefinition().Spec = nil
			}
			v.SourceConnectorDefinitions = append(
				v.SourceConnectorDefinitions,
				def)
		}
		v.NextPageToken = nextPageToken
		v.TotalSize = int64(len(defs))
	case *connectorPB.ListDestinationConnectorDefinitionsResponse:
		defs := h.connectorDestination.ListConnectorDefinitions()

		startIdx := 0
		lastUid := ""
		for idx, def := range defs {
			if def.(*connectorPB.DestinationConnectorDefinition).Uid == prevLastUid {
				startIdx = idx + 1
				break
			}
		}

		page := []*connectorPB.DestinationConnectorDefinition{}
		for i := 0; i < int(pageSize) && startIdx+i < len(defs); i++ {
			def := proto.Clone(defs[startIdx+i].(*connectorPB.DestinationConnectorDefinition)).(*connectorPB.DestinationConnectorDefinition)
			page = append(page, def)
			lastUid = def.Uid
		}

		nextPageToken := ""

		if startIdx+len(page) < len(defs) {
			nextPageToken = paginate.EncodeToken(time.Time{}, lastUid)
		}
		for _, def := range page {
			def.Name = fmt.Sprintf("destination-connector-definitions/%s", def.Id)
			def.ConnectorDefinition.ReleaseDate = func() *date.Date {
				if def.ConnectorDefinition.ReleaseDate != nil {
					return &date.Date{
						Year:  int32(def.ConnectorDefinition.ReleaseDate.Year),
						Month: int32(def.ConnectorDefinition.ReleaseDate.Month),
						Day:   int32(def.ConnectorDefinition.ReleaseDate.Day),
					}
				}
				return &date.Date{}
			}()
			if isBasicView {
				def.GetConnectorDefinition().Spec = nil
			}
			v.DestinationConnectorDefinitions = append(
				v.DestinationConnectorDefinitions,
				def)
		}
		v.NextPageToken = nextPageToken
		v.TotalSize = int64(len(defs))
	}

	logger.Info("ListConnectorDefinitions")

	return resp, nil
}

func (h *PublicHandler) getConnectorDefinition(ctx context.Context, req interface{}) (resp interface{}, err error) {

	ctx, span := tracer.Start(ctx, "GetConnectorDefinition",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string
	var isBasicView bool

	switch v := req.(type) {
	case *connectorPB.GetSourceConnectorDefinitionRequest:
		resp = &connectorPB.GetSourceConnectorDefinitionResponse{}
		if connID, err = resource.GetRscNameID(v.GetName()); err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}
		isBasicView = (v.GetView() == connectorPB.View_VIEW_BASIC) || (v.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
	case *connectorPB.GetDestinationConnectorDefinitionRequest:
		resp = &connectorPB.GetDestinationConnectorDefinitionResponse{}
		if connID, err = resource.GetRscNameID(v.GetName()); err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}
		isBasicView = (v.GetView() == connectorPB.View_VIEW_BASIC) || (v.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
	}

	// TODO: refactor
	switch v := resp.(type) {
	case *connectorPB.GetSourceConnectorDefinitionResponse:
		dbDef, err := h.connectorSource.GetConnectorDefinitionById(connID)

		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}
		v.SourceConnectorDefinition = proto.Clone(dbDef.(*connectorPB.SourceConnectorDefinition)).(*connectorPB.SourceConnectorDefinition)
		if isBasicView {
			v.SourceConnectorDefinition.ConnectorDefinition.Spec = nil
		}
		v.SourceConnectorDefinition.Name = fmt.Sprintf("source-connector-definitions/%s", v.SourceConnectorDefinition.GetId())
		v.SourceConnectorDefinition.ConnectorDefinition.ReleaseDate = func() *date.Date {
			if v.SourceConnectorDefinition.ConnectorDefinition.ReleaseDate != nil {
				return &date.Date{
					Year:  int32(v.SourceConnectorDefinition.ConnectorDefinition.ReleaseDate.Year),
					Month: int32(v.SourceConnectorDefinition.ConnectorDefinition.ReleaseDate.Month),
					Day:   int32(v.SourceConnectorDefinition.ConnectorDefinition.ReleaseDate.Day),
				}
			}
			return &date.Date{}
		}()
		logger.Info(v.SourceConnectorDefinition.ConnectorDefinition.Title)
	case *connectorPB.GetDestinationConnectorDefinitionResponse:
		dbDef, err := h.connectorDestination.GetConnectorDefinitionById(connID)
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}
		v.DestinationConnectorDefinition = proto.Clone(dbDef.(*connectorPB.DestinationConnectorDefinition)).(*connectorPB.DestinationConnectorDefinition)
		if isBasicView {
			v.DestinationConnectorDefinition.ConnectorDefinition.Spec = nil
		}
		v.DestinationConnectorDefinition.Name = fmt.Sprintf("destination-connector-definitions/%s", v.DestinationConnectorDefinition.GetId())
		v.DestinationConnectorDefinition.ConnectorDefinition.ReleaseDate = func() *date.Date {
			if v.DestinationConnectorDefinition.ConnectorDefinition.ReleaseDate != nil {
				return &date.Date{
					Year:  int32(v.DestinationConnectorDefinition.ConnectorDefinition.ReleaseDate.Year),
					Month: int32(v.DestinationConnectorDefinition.ConnectorDefinition.ReleaseDate.Month),
					Day:   int32(v.DestinationConnectorDefinition.ConnectorDefinition.ReleaseDate.Day),
				}
			}
			return &date.Date{}
		}()
		logger.Info(v.DestinationConnectorDefinition.ConnectorDefinition.Title)
	}

	return resp, nil

}

func (h *PublicHandler) createConnector(ctx context.Context, req interface{}) (resp interface{}, err error) {

	ctx, span := tracer.Start(ctx, "CreateConnector",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string
	var connDesc sql.NullString
	var connType datamodel.ConnectorType
	var connConfig datatypes.JSON

	var connDefUID uuid.UUID
	var connDefRscName string

	switch v := req.(type) {
	case *connectorPB.CreateSourceConnectorRequest:

		resp = &connectorPB.CreateSourceConnectorResponse{}

		// Set all OUTPUT_ONLY fields to zero value on the requested payload
		if err := checkfield.CheckCreateOutputOnlyFields(v.GetSourceConnector(), outputOnlyFields); err != nil {
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
		if err := checkfield.CheckRequiredFields(v.GetSourceConnector(), append(createRequiredFields, sourceImmutableFields...)); err != nil {
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

		// Validate SourceConnector JSON Schema
		configLoader := connectorConfigLoader.InitJSONSchema(logger)
		if err := connectorConfigLoader.ValidateJSONSchema(configLoader.SrcConnJSONSchema, v.GetSourceConnector(), false); err != nil {
			st, err := sterr.CreateErrorBadRequest(
				"[handler] create connector error",
				[]*errdetails.BadRequest_FieldViolation{
					{
						Field:       "source_connector",
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

		connID = v.GetSourceConnector().GetId()

		connConfig, err = v.GetSourceConnector().GetConnector().GetConfiguration().MarshalJSON()
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}

		connDesc = sql.NullString{
			String: v.SourceConnector.GetConnector().GetDescription(),
			Valid:  len(v.SourceConnector.GetConnector().GetDescription()) > 0,
		}

		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE)

		connDefResp, err := h.GetSourceConnectorDefinition(
			ctx,
			&connectorPB.GetSourceConnectorDefinitionRequest{
				Name: v.GetSourceConnector().GetSourceConnectorDefinition(),
				View: connectorPB.View_VIEW_FULL.Enum(),
			})
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}

		// Validate SourceConnector configuration JSON Schema
		connSpec := connDefResp.GetSourceConnectorDefinition().GetConnectorDefinition().GetSpec().GetConnectionSpecification()
		b, err := protojson.Marshal(connSpec)
		if err != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.Internal,
				"[handler] create connector error",
				"destination-connector-definitions",
				fmt.Sprintf("uid %s", connDefResp.SourceConnectorDefinition.GetUid()),
				"",
				err.Error(),
			)
			if err != nil {
				logger.Error(err.Error())
			}
			span.SetStatus(1, st.Err().Error())
			return resp, st.Err()
		}

		if err := connectorConfigLoader.ValidateJSONSchemaString(string(b), v.GetSourceConnector().GetConnector().GetConfiguration()); err != nil {
			st, err := sterr.CreateErrorBadRequest(
				"[handler] create connector error",
				[]*errdetails.BadRequest_FieldViolation{
					{
						Field:       "source_connector.connector.configuration",
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

		connDefUID, err = uuid.FromString(connDefResp.SourceConnectorDefinition.GetUid())
		if err != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.Internal,
				"[handler] create connector error",
				"source-connector-definitions",
				fmt.Sprintf("uid %s", connDefResp.SourceConnectorDefinition.GetUid()),
				"",
				err.Error(),
			)
			if err != nil {
				logger.Error(err.Error())
			}
			span.SetStatus(1, st.Err().Error())
			return resp, st.Err()
		}

		connDefRscName = fmt.Sprintf("source-connector-definitions/%s", connDefResp.SourceConnectorDefinition.GetId())

	case *connectorPB.CreateDestinationConnectorRequest:

		resp = &connectorPB.CreateDestinationConnectorResponse{}

		// Set all OUTPUT_ONLY fields to zero value on the requested payload
		if err := checkfield.CheckCreateOutputOnlyFields(v.GetDestinationConnector(), outputOnlyFields); err != nil {
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
		if err := checkfield.CheckRequiredFields(v.GetDestinationConnector(), append(createRequiredFields, destinationImmutableFields...)); err != nil {
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

		// Validate DestinationConnector JSON Schema
		configLoader := connectorConfigLoader.InitJSONSchema(logger)
		if err := connectorConfigLoader.ValidateJSONSchema(configLoader.DstConnJSONSchema, v.GetDestinationConnector(), false); err != nil {
			st, err := sterr.CreateErrorBadRequest(
				"[handler] create connector error",
				[]*errdetails.BadRequest_FieldViolation{
					{
						Field:       "destination_connector",
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

		connID = v.GetDestinationConnector().GetId()

		connConfig, err = v.GetDestinationConnector().GetConnector().GetConfiguration().MarshalJSON()
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}

		connDesc = sql.NullString{
			String: v.DestinationConnector.GetConnector().GetDescription(),
			Valid:  len(v.DestinationConnector.GetConnector().GetDescription()) > 0,
		}

		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION)

		connDefResp, err := h.GetDestinationConnectorDefinition(
			ctx,
			&connectorPB.GetDestinationConnectorDefinitionRequest{
				Name: v.GetDestinationConnector().GetDestinationConnectorDefinition(),
				View: connectorPB.View_VIEW_FULL.Enum(),
			})
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}

		// Validate DestinationConnector configuration JSON Schema
		connSpec := connDefResp.GetDestinationConnectorDefinition().GetConnectorDefinition().GetSpec().GetConnectionSpecification()
		b, err := protojson.Marshal(connSpec)
		if err != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.Internal,
				"[handler] create connector error",
				"destination-connector-definitions",
				fmt.Sprintf("uid %s", connDefResp.DestinationConnectorDefinition.GetUid()),
				"",
				err.Error(),
			)
			if err != nil {
				logger.Error(err.Error())
			}
			span.SetStatus(1, st.Err().Error())
			return resp, st.Err()
		}

		if err := connectorConfigLoader.ValidateJSONSchemaString(string(b), v.GetDestinationConnector().GetConnector().GetConfiguration()); err != nil {
			st, err := sterr.CreateErrorBadRequest(
				"[handler] create connector error",
				[]*errdetails.BadRequest_FieldViolation{
					{
						Field:       "destination_connector.connector.configuration",
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

		connDefUID, err = uuid.FromString(connDefResp.DestinationConnectorDefinition.GetUid())
		if err != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.Internal,
				"[handler] create connector error",
				"destination-connector-definitions",
				fmt.Sprintf("uid %s", connDefResp.DestinationConnectorDefinition.GetUid()),
				"",
				err.Error(),
			)
			if err != nil {
				logger.Error(err.Error())
			}
			span.SetStatus(1, st.Err().Error())
			return resp, st.Err()
		}

		connDefRscName = fmt.Sprintf("destination-connector-definitions/%s", connDefResp.DestinationConnectorDefinition.GetId())
	}

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
		ConnectorType:          connType,
		Description:            connDesc,
	}

	dbConnector, err = h.service.CreateConnector(ctx, owner, dbConnector)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		owner,
		true,
		"CreateConnector",
		"request",
		"CreateConnector done",
		false,
		custom_otel.SetEventResource(dbConnector),
	)))

	pbConnector := DBToPBConnector(
		ctx,
		dbConnector,
		connType,
		service.GenOwnerPermalink(owner),
		connDefRscName)

	switch v := resp.(type) {
	case *connectorPB.CreateSourceConnectorResponse:
		v.SourceConnector = pbConnector.(*connectorPB.SourceConnector)
	case *connectorPB.CreateDestinationConnectorResponse:
		v.DestinationConnector = pbConnector.(*connectorPB.DestinationConnector)
	}

	return resp, nil
}

func (h *PublicHandler) listConnectors(ctx context.Context, req interface{}) (resp interface{}, err error) {

	ctx, span := tracer.Start(ctx, "ListConnectors",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	var pageSize int64
	var pageToken string
	var isBasicView bool

	var connType datamodel.ConnectorType

	var connDefColID string

	switch v := req.(type) {
	case *connectorPB.ListSourceConnectorsRequest:
		resp = &connectorPB.ListSourceConnectorsResponse{}
		pageSize = v.GetPageSize()
		pageToken = v.GetPageToken()
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE)
		isBasicView = (v.GetView() == connectorPB.View_VIEW_BASIC) || (v.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
		connDefColID = "source-connector-definitions"
	case *connectorPB.ListDestinationConnectorsRequest:
		resp = &connectorPB.ListDestinationConnectorsResponse{}
		pageSize = v.GetPageSize()
		pageToken = v.GetPageToken()
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION)
		isBasicView = (v.GetView() == connectorPB.View_VIEW_BASIC) || (v.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
		connDefColID = "destination-connector-definitions"
	}

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbConnectors, totalSize, nextPageToken, err := h.service.ListConnectors(ctx, owner, connType, pageSize, pageToken, isBasicView)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	var pbConnectors []interface{}
	for idx := range dbConnectors {
		dbConnDef, err := h.connectorAll.GetConnectorDefinitionByUid(dbConnectors[idx].ConnectorDefinitionUID)
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}
		pbConnectors = append(
			pbConnectors,
			DBToPBConnector(
				ctx,
				dbConnectors[idx],
				connType,
				dbConnectors[idx].Owner,
				fmt.Sprintf("%s/%s", connDefColID, dbConnDef.GetId()),
			))
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		owner,
		false,
		"ListConnectors",
		"request",
		"ListConnectors done",
		false,
	)))

	switch v := resp.(type) {
	case *connectorPB.ListSourceConnectorsResponse:
		var pbSrcConns []*connectorPB.SourceConnector
		for _, pbConnector := range pbConnectors {
			pbSrcConns = append(pbSrcConns, pbConnector.(*connectorPB.SourceConnector))
		}
		v.SourceConnectors = pbSrcConns
		v.NextPageToken = nextPageToken
		v.TotalSize = totalSize
	case *connectorPB.ListDestinationConnectorsResponse:
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

func (h *PublicHandler) getConnector(ctx context.Context, req interface{}) (resp interface{}, err error) {

	ctx, span := tracer.Start(ctx, "GetConnector",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	var isBasicView bool

	var connID string
	var connType datamodel.ConnectorType

	var connDefColID string

	switch v := req.(type) {
	case *connectorPB.GetSourceConnectorRequest:
		resp = &connectorPB.GetSourceConnectorResponse{}
		if connID, err = resource.GetRscNameID(v.GetName()); err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE)
		isBasicView = (v.GetView() == connectorPB.View_VIEW_BASIC) || (v.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
		connDefColID = "source-connector-definitions"
	case *connectorPB.GetDestinationConnectorRequest:
		resp = &connectorPB.GetDestinationConnectorResponse{}
		if connID, err = resource.GetRscNameID(v.GetName()); err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION)
		isBasicView = (v.GetView() == connectorPB.View_VIEW_BASIC) || (v.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
		connDefColID = "destination-connector-definitions"
	}

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbConnector, err := h.service.GetConnectorByID(ctx, connID, owner, connType, isBasicView)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbConnDef, err := h.connectorAll.GetConnectorDefinitionByUid(dbConnector.ConnectorDefinitionUID)

	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	pbConnector := DBToPBConnector(
		ctx,
		dbConnector,
		connType,
		dbConnector.Owner,
		fmt.Sprintf("%s/%s", connDefColID, dbConnDef.GetId()),
	)

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		owner,
		false,
		"GetConnector",
		"request",
		"GetConnector done",
		false,
		custom_otel.SetEventResource(dbConnector),
	)))

	switch v := resp.(type) {
	case *connectorPB.GetSourceConnectorResponse:
		v.SourceConnector = pbConnector.(*connectorPB.SourceConnector)
	case *connectorPB.GetDestinationConnectorResponse:
		v.DestinationConnector = pbConnector.(*connectorPB.DestinationConnector)
	}

	return resp, nil
}

func (h *PublicHandler) updateConnector(ctx context.Context, req interface{}) (resp interface{}, err error) {

	ctx, span := tracer.Start(ctx, "UpdateConnector",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	var pbConnectorReq interface{}
	var pbConnectorToUpdate interface{}

	var mask fieldmask_utils.Mask

	var connID string
	var connType datamodel.ConnectorType

	var connDefRscName string
	var connDefUID uuid.UUID

	owner, ownerErr := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient())

	switch v := req.(type) {
	case *connectorPB.UpdateSourceConnectorRequest:
		resp = &connectorPB.UpdateSourceConnectorResponse{}
		if ownerErr != nil {
			span.SetStatus(1, ownerErr.Error())
			return resp, ownerErr
		}

		pbConnectorReq = v.GetSourceConnector()
		pbUpdateMask := v.GetUpdateMask()

		// configuration filed is type google.protobuf.Struct, which needs to be updated as a whole
		for idx, path := range pbUpdateMask.Paths {
			if strings.Contains(path, "configuration") {
				pbUpdateMask.Paths[idx] = "connector.configuration"
			}
		}

		if !pbUpdateMask.IsValid(v.GetSourceConnector()) {
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
		getResp, err := h.GetSourceConnector(
			ctx,
			&connectorPB.GetSourceConnectorRequest{
				Name: v.GetSourceConnector().GetName(),
				View: connectorPB.View_VIEW_FULL.Enum(),
			})
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}

		// Return error if IMMUTABLE fields are intentionally changed
		if err := checkfield.CheckUpdateImmutableFields(v.GetSourceConnector(), getResp.GetSourceConnector(), sourceImmutableFields); err != nil {
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
			span.SetStatus(1, err.Error())
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
			return &connectorPB.UpdateSourceConnectorResponse{
				SourceConnector: getResp.GetSourceConnector(),
			}, nil
		}

		pbConnectorToUpdate = getResp.GetSourceConnector()

		connID = getResp.GetSourceConnector().GetId()
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE)

		dbConnDefID, err := resource.GetRscNameID(getResp.GetSourceConnector().GetSourceConnectorDefinition())
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}

		dbConnDef, err := h.connectorSource.GetConnectorDefinitionById(dbConnDefID)
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}

		connDefRscName = fmt.Sprintf("source-connector-definitions/%s", dbConnDef.GetId())
		connDefUID = uuid.FromStringOrNil(dbConnDef.GetUid())

	case *connectorPB.UpdateDestinationConnectorRequest:
		resp = &connectorPB.UpdateDestinationConnectorResponse{}
		if ownerErr != nil {
			span.SetStatus(1, ownerErr.Error())
			return resp, ownerErr
		}

		pbConnectorReq = v.GetDestinationConnector()
		pbUpdateMask := v.GetUpdateMask()

		// configuration filed is type google.protobuf.Struct, which needs to be updated as a whole
		for idx, path := range pbUpdateMask.Paths {
			if strings.Contains(path, "configuration") {
				pbUpdateMask.Paths[idx] = "connector.configuration"
			}
		}

		if !pbUpdateMask.IsValid(v.GetDestinationConnector()) {
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

		getResp, err := h.GetDestinationConnector(
			ctx,
			&connectorPB.GetDestinationConnectorRequest{
				Name: v.GetDestinationConnector().GetName(),
				View: connectorPB.View_VIEW_FULL.Enum(),
			})
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}

		// Return error if IMMUTABLE fields are intentionally changed
		if err := checkfield.CheckUpdateImmutableFields(v.GetDestinationConnector(), getResp.GetDestinationConnector(), destinationImmutableFields); err != nil {
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
			return &connectorPB.UpdateDestinationConnectorResponse{
				DestinationConnector: getResp.GetDestinationConnector(),
			}, nil
		}

		pbConnectorToUpdate = getResp.GetDestinationConnector()

		connID = getResp.GetDestinationConnector().GetId()
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION)

		dbConnDefID, err := resource.GetRscNameID(getResp.GetDestinationConnector().GetDestinationConnectorDefinition())
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}

		dbConnDef, err := h.connectorDestination.GetConnectorDefinitionById(dbConnDefID)
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}

		connDefRscName = fmt.Sprintf("destination-connector-definitions/%s", dbConnDef.GetId())
		connDefUID = uuid.FromStringOrNil(dbConnDef.GetUid())

	}

	if ownerErr != nil {
		span.SetStatus(1, err.Error())
		return resp, ownerErr
	}

	// Only the fields mentioned in the field mask will be copied to `pbPipelineToUpdate`, other fields are left intact
	err = fieldmask_utils.StructToStruct(mask, pbConnectorReq, pbConnectorToUpdate)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbConnector, err := h.service.UpdateConnector(ctx, connID, owner, connType, PBToDBConnector(ctx, pbConnectorToUpdate, connType, owner.GetName(), connDefUID))
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		owner,
		true,
		"UpdateConnector",
		"request",
		"UpdateConnector done",
		false,
		custom_otel.SetEventResource(dbConnector),
	)))

	pbConnector := DBToPBConnector(
		ctx,
		dbConnector,
		connType,
		service.GenOwnerPermalink(owner),
		connDefRscName)

	switch v := resp.(type) {
	case *connectorPB.UpdateSourceConnectorResponse:
		v.SourceConnector = pbConnector.(*connectorPB.SourceConnector)
	case *connectorPB.UpdateDestinationConnectorResponse:
		v.DestinationConnector = pbConnector.(*connectorPB.DestinationConnector)
	}

	return resp, nil
}

func (h *PublicHandler) deleteConnector(ctx context.Context, req interface{}) (resp interface{}, err error) {

	ctx, span := tracer.Start(ctx, "DeleteConnector",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string
	var connType datamodel.ConnectorType

	// Cast all used types and data
	switch v := req.(type) {
	case *connectorPB.DeleteSourceConnectorRequest:
		resp = &connectorPB.DeleteSourceConnectorResponse{}
		if connID, err = resource.GetRscNameID(v.GetName()); err != nil {
			return resp, err
		}
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE)
	case *connectorPB.DeleteDestinationConnectorRequest:
		resp = &connectorPB.DeleteDestinationConnectorResponse{}
		if connID, err = resource.GetRscNameID(v.GetName()); err != nil {
			return resp, err
		}
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION)
	}

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbConnector, err := h.service.GetConnectorByID(ctx, connID, owner, connType, true)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	if err := h.service.DeleteConnector(ctx, connID, owner, connType); err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		owner,
		true,
		"DeleteConnector",
		"request",
		"DeleteConnector done",
		false,
		custom_otel.SetEventResource(dbConnector),
	)))

	return resp, nil
}

func (h *PublicHandler) lookUpConnector(ctx context.Context, req interface{}) (resp interface{}, err error) {

	ctx, span := tracer.Start(ctx, "LookUpConnector",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	var isBasicView bool

	var connUID uuid.UUID
	var connType datamodel.ConnectorType

	var connDefColID string

	switch v := req.(type) {
	case *connectorPB.LookUpSourceConnectorRequest:
		resp = &connectorPB.LookUpSourceConnectorResponse{}

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
			span.SetStatus(1, st.Err().Error())
			return resp, st.Err()
		}

		connUIDStr, err := resource.GetPermalinkUID(v.GetPermalink())
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}
		connUID, err = uuid.FromString(connUIDStr)
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE)
		isBasicView = (v.GetView() == connectorPB.View_VIEW_BASIC) || (v.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
		connDefColID = "source-connector-definitions"
	case *connectorPB.LookUpDestinationConnectorRequest:
		resp = &connectorPB.LookUpDestinationConnectorResponse{}

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
			span.SetStatus(1, st.Err().Error())
			return resp, st.Err()
		}

		connUIDStr, err := resource.GetPermalinkUID(v.GetPermalink())
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}
		connUID, err = uuid.FromString(connUIDStr)
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION)
		isBasicView = (v.GetView() == connectorPB.View_VIEW_BASIC) || (v.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
		connDefColID = "destination-connector-definitions"
	}

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

	dbConnDef, err := h.connectorAll.GetConnectorDefinitionByUid(dbConnector.ConnectorDefinitionUID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		owner,
		false,
		"LookUpConnector",
		"request",
		"LookUpConnector done",
		false,
		custom_otel.SetEventResource(dbConnector),
	)))

	pbConnector := DBToPBConnector(
		ctx,
		dbConnector,
		connType,
		dbConnector.Owner,
		fmt.Sprintf("%s/%s", connDefColID, dbConnDef.GetId()),
	)

	switch v := resp.(type) {
	case *connectorPB.LookUpSourceConnectorResponse:
		v.SourceConnector = pbConnector.(*connectorPB.SourceConnector)
	case *connectorPB.LookUpDestinationConnectorResponse:
		v.DestinationConnector = pbConnector.(*connectorPB.DestinationConnector)
	}

	return resp, nil
}

func (h *PublicHandler) connectConnector(ctx context.Context, req interface{}) (resp interface{}, err error) {

	ctx, span := tracer.Start(ctx, "ConnectConnector",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string
	var connType datamodel.ConnectorType

	var connDefRscName string

	switch v := req.(type) {
	case *connectorPB.ConnectSourceConnectorRequest:
		resp = &connectorPB.ConnectSourceConnectorResponse{}

		// Return error if REQUIRED fields are not provided in the requested payload
		if err := checkfield.CheckRequiredFields(v, connectSourceRequiredFields); err != nil {
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

		connID, err = resource.GetRscNameID(v.GetName())
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE)

		getResp, err := h.GetSourceConnector(
			ctx,
			&connectorPB.GetSourceConnectorRequest{
				Name: v.GetName(),
				View: connectorPB.View_VIEW_BASIC.Enum(),
			})
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}

		dbConnDefID, err := resource.GetRscNameID(getResp.GetSourceConnector().GetSourceConnectorDefinition())
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}

		dbConnDef, err := h.connectorSource.GetConnectorDefinitionById(dbConnDefID)
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}

		connDefRscName = fmt.Sprintf("source-connector-definitions/%s", dbConnDef.GetId())

	case *connectorPB.ConnectDestinationConnectorRequest:
		resp = &connectorPB.ConnectDestinationConnectorResponse{}

		// Return error if REQUIRED fields are not provided in the requested payload
		if err := checkfield.CheckRequiredFields(v, connectDestinationRequiredFields); err != nil {
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

		connID, err = resource.GetRscNameID(v.GetName())
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION)

		getResp, err := h.GetDestinationConnector(
			ctx,
			&connectorPB.GetDestinationConnectorRequest{
				Name: v.GetName(),
				View: connectorPB.View_VIEW_BASIC.Enum(),
			})
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}

		dbConnDefID, err := resource.GetRscNameID(getResp.GetDestinationConnector().GetDestinationConnectorDefinition())
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}

		dbConnDef, err := h.connectorDestination.GetConnectorDefinitionById(dbConnDefID)
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}

		connDefRscName = fmt.Sprintf("destination-connector-definitions/%s", dbConnDef.GetId())
	}

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbConnector, err := h.service.UpdateConnectorState(ctx, connID, owner, connType, datamodel.ConnectorState(connectorPB.Connector_STATE_CONNECTED))
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		owner,
		true,
		"ConnectConnector",
		"request",
		"ConnectConnector done",
		false,
		custom_otel.SetEventResource(dbConnector),
	)))

	pbConnector := DBToPBConnector(
		ctx,
		dbConnector,
		connType,
		dbConnector.Owner,
		connDefRscName,
	)

	switch v := resp.(type) {
	case *connectorPB.ConnectSourceConnectorResponse:
		v.SourceConnector = pbConnector.(*connectorPB.SourceConnector)
	case *connectorPB.ConnectDestinationConnectorResponse:
		v.DestinationConnector = pbConnector.(*connectorPB.DestinationConnector)
	}

	return resp, nil
}

func (h *PublicHandler) disconnectConnector(ctx context.Context, req interface{}) (resp interface{}, err error) {

	ctx, span := tracer.Start(ctx, "DisconnectConnector",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string
	var connType datamodel.ConnectorType

	var connDefRscName string

	switch v := req.(type) {
	case *connectorPB.DisconnectSourceConnectorRequest:
		resp = &connectorPB.DisconnectSourceConnectorResponse{}

		// Return error if REQUIRED fields are not provided in the requested payload
		if err := checkfield.CheckRequiredFields(v, disconnectSourceRequiredFields); err != nil {
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

		connID, err = resource.GetRscNameID(v.GetName())
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE)

		getResp, err := h.GetSourceConnector(
			ctx,
			&connectorPB.GetSourceConnectorRequest{
				Name: v.GetName(),
				View: connectorPB.View_VIEW_BASIC.Enum(),
			})
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}

		dbConnDefID, err := resource.GetRscNameID(getResp.GetSourceConnector().GetSourceConnectorDefinition())
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}

		dbConnDef, err := h.connectorSource.GetConnectorDefinitionById(dbConnDefID)
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}

		connDefRscName = fmt.Sprintf("source-connector-definitions/%s", dbConnDef.GetId())

	case *connectorPB.DisconnectDestinationConnectorRequest:
		resp = &connectorPB.DisconnectDestinationConnectorResponse{}

		// Return error if REQUIRED fields are not provided in the requested payload
		if err := checkfield.CheckRequiredFields(v, disconnectDestinationRequiredFields); err != nil {
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

		connID, err = resource.GetRscNameID(v.GetName())
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION)

		getResp, err := h.GetDestinationConnector(
			ctx,
			&connectorPB.GetDestinationConnectorRequest{
				Name: v.GetName(),
				View: connectorPB.View_VIEW_BASIC.Enum(),
			})
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}

		dbConnDefID, err := resource.GetRscNameID(getResp.GetDestinationConnector().GetDestinationConnectorDefinition())
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}

		dbConnDef, err := h.connectorDestination.GetConnectorDefinitionById(dbConnDefID)
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}

		connDefRscName = fmt.Sprintf("destination-connector-definitions/%s", dbConnDef.GetId())
	}

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbConnector, err := h.service.UpdateConnectorState(ctx, connID, owner, connType, datamodel.ConnectorState(connectorPB.Connector_STATE_DISCONNECTED))
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		owner,
		true,
		"DisconnectConnector",
		"request",
		"DisconnectConnector done",
		false,
		custom_otel.SetEventResource(dbConnector),
	)))

	pbConnector := DBToPBConnector(
		ctx,
		dbConnector,
		connType,
		dbConnector.Owner,
		connDefRscName,
	)

	switch v := resp.(type) {
	case *connectorPB.DisconnectSourceConnectorResponse:
		v.SourceConnector = pbConnector.(*connectorPB.SourceConnector)
	case *connectorPB.DisconnectDestinationConnectorResponse:
		v.DestinationConnector = pbConnector.(*connectorPB.DestinationConnector)
	}

	return resp, nil
}

func (h *PublicHandler) renameConnector(ctx context.Context, req interface{}) (resp interface{}, err error) {

	ctx, span := tracer.Start(ctx, "RenameConnector",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string
	var connNewID string
	var connType datamodel.ConnectorType

	var connDefRscName string

	switch v := req.(type) {
	case *connectorPB.RenameSourceConnectorRequest:
		resp = &connectorPB.RenameSourceConnectorResponse{}

		// Return error if REQUIRED fields are not provided in the requested payload
		if err := checkfield.CheckRequiredFields(v, renameSourceRequiredFields); err != nil {
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

		connID, err = resource.GetRscNameID(v.GetName())
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}
		connNewID = v.GetNewSourceConnectorId()
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE)

		getResp, err := h.GetSourceConnector(
			ctx,
			&connectorPB.GetSourceConnectorRequest{
				Name: v.GetName(),
				View: connectorPB.View_VIEW_BASIC.Enum(),
			})
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}

		dbConnDefID, err := resource.GetRscNameID(getResp.GetSourceConnector().GetSourceConnectorDefinition())
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}

		dbConnDef, err := h.connectorSource.GetConnectorDefinitionById(dbConnDefID)
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}

		connDefRscName = fmt.Sprintf("source-connector-definitions/%s", dbConnDef.GetId())

	case *connectorPB.RenameDestinationConnectorRequest:
		resp = &connectorPB.RenameDestinationConnectorResponse{}

		// Return error if REQUIRED fields are not provided in the requested payload
		if err := checkfield.CheckRequiredFields(v, renameDestinationRequiredFields); err != nil {
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

		connID, err = resource.GetRscNameID(v.GetName())
		if err != nil {
			return resp, err
		}
		connNewID = v.GetNewDestinationConnectorId()
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION)

		getResp, err := h.GetDestinationConnector(
			ctx,
			&connectorPB.GetDestinationConnectorRequest{
				Name: v.GetName(),
				View: connectorPB.View_VIEW_BASIC.Enum(),
			})
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}

		dbConnDefID, err := resource.GetRscNameID(getResp.GetDestinationConnector().GetDestinationConnectorDefinition())
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}

		dbConnDef, err := h.connectorDestination.GetConnectorDefinitionById(dbConnDefID)
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}

		connDefRscName = fmt.Sprintf("destination-connector-definitions/%s", dbConnDef.GetId())
	}

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

	dbConnector, err := h.service.UpdateConnectorID(ctx, connID, owner, connType, connNewID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		owner,
		true,
		"RenameConnector",
		"request",
		"RenameConnector done",
		false,
		custom_otel.SetEventResource(dbConnector),
	)))

	pbConnector := DBToPBConnector(
		ctx,
		dbConnector,
		connType,
		dbConnector.Owner,
		connDefRscName,
	)

	switch v := resp.(type) {
	case *connectorPB.RenameSourceConnectorResponse:
		v.SourceConnector = pbConnector.(*connectorPB.SourceConnector)
	case *connectorPB.RenameDestinationConnectorResponse:
		v.DestinationConnector = pbConnector.(*connectorPB.DestinationConnector)
	}

	return resp, nil
}

func (h *PublicHandler) watchConnector(ctx context.Context, req interface{}) (resp interface{}, err error) {

	ctx, span := tracer.Start(ctx, "WatchConnector",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string
	var connType datamodel.ConnectorType

	switch v := req.(type) {
	case *connectorPB.WatchSourceConnectorRequest:
		resp = &connectorPB.WatchSourceConnectorResponse{}
		if connID, err = resource.GetRscNameID(v.GetName()); err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE)
	case *connectorPB.WatchDestinationConnectorRequest:
		resp = &connectorPB.WatchDestinationConnectorResponse{}
		if connID, err = resource.GetRscNameID(v.GetName()); err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION)
	}

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		logger.Info(string(custom_otel.NewLogMessage(
			span,
			owner,
			true,
			"WatchConnector",
			"request",
			"WatchConnector error",
			false,
			custom_otel.SetErrorMessage(err.Error()),
		)))
		return resp, err
	}

	dbConnector, err := h.service.GetConnectorByID(ctx, connID, owner, connType, true)
	if err != nil {
		span.SetStatus(1, err.Error())
		logger.Info(string(custom_otel.NewLogMessage(
			span,
			owner,
			true,
			"WatchConnector",
			"request",
			"WatchConnector error",
			false,
			custom_otel.SetErrorMessage(err.Error()),
		)))
		return resp, err
	}

	state, err := h.service.GetResourceState(dbConnector.UID, connType)

	if err != nil {
		span.SetStatus(1, err.Error())
		logger.Info(string(custom_otel.NewLogMessage(
			span,
			owner,
			true,
			"WatchConnector",
			"request",
			"WatchConnector error",
			false,
			custom_otel.SetErrorMessage(err.Error()),
			custom_otel.SetEventResource(dbConnector),
		)))
		return resp, err
	}

	switch v := resp.(type) {
	case *connectorPB.WatchSourceConnectorResponse:
		v.State = *state
	case *connectorPB.WatchDestinationConnectorResponse:
		v.State = *state
	}

	return resp, nil
}

func (h *PublicHandler) testConnector(ctx context.Context, req interface{}) (resp interface{}, err error) {

	ctx, span := tracer.Start(ctx, "TestConnector",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string
	var connType datamodel.ConnectorType

	switch v := req.(type) {
	case *connectorPB.TestSourceConnectorRequest:
		resp = &connectorPB.TestSourceConnectorResponse{}
		if connID, err = resource.GetRscNameID(v.GetName()); err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE)
	case *connectorPB.TestDestinationConnectorRequest:
		resp = &connectorPB.TestDestinationConnectorResponse{}
		if connID, err = resource.GetRscNameID(v.GetName()); err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION)
	}

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbConnector, err := h.service.GetConnectorByID(ctx, connID, owner, connType, true)
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
		owner,
		true,
		"TestConnector",
		"request",
		"TestConnector done",
		false,
		custom_otel.SetEventResource(dbConnector),
	)))

	switch v := resp.(type) {
	case *connectorPB.TestSourceConnectorResponse:
		v.State = *state
	case *connectorPB.TestDestinationConnectorResponse:
		v.State = *state
	}

	return resp, nil
}
