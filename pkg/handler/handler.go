package handler

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/iancoleman/strcase"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/datatypes"

	fieldmask_utils "github.com/mennanov/fieldmask-utils"

	"github.com/instill-ai/connector-backend/internal/logger"
	"github.com/instill-ai/connector-backend/internal/resource"
	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/connector-backend/pkg/service"
	"github.com/instill-ai/x/checkfield"
	"github.com/instill-ai/x/sterr"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	healthcheckPB "github.com/instill-ai/protogen-go/vdp/healthcheck/v1alpha"
)

type handler struct {
	connectorPB.UnimplementedConnectorServiceServer
	service service.Service
}

// NewHandler initiates a handler instance
func NewHandler(s service.Service) connectorPB.ConnectorServiceServer {
	datamodel.InitJSONSchema()
	datamodel.InitTaskAirbyteCatalog()
	return &handler{
		service: s,
	}
}

func (h *handler) Liveness(ctx context.Context, in *connectorPB.LivenessRequest) (*connectorPB.LivenessResponse, error) {
	return &connectorPB.LivenessResponse{
		HealthCheckResponse: &healthcheckPB.HealthCheckResponse{
			Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_SERVING,
		},
	}, nil
}

func (h *handler) Readiness(ctx context.Context, in *connectorPB.ReadinessRequest) (*connectorPB.ReadinessResponse, error) {
	return &connectorPB.ReadinessResponse{
		HealthCheckResponse: &healthcheckPB.HealthCheckResponse{
			Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_SERVING,
		},
	}, nil
}

func (h *handler) listConnectorDefinition(ctx context.Context, req interface{}) (resp interface{}, err error) {

	var pageSize int64
	var pageToken string
	var isBasicView bool

	var connType datamodel.ConnectorType

	switch v := req.(type) {
	case *connectorPB.ListSourceConnectorDefinitionRequest:
		resp = &connectorPB.ListSourceConnectorDefinitionResponse{}
		pageSize = v.GetPageSize()
		pageToken = v.GetPageToken()
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE)
		isBasicView = (v.GetView() == connectorPB.View_VIEW_BASIC) || (v.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
	case *connectorPB.ListDestinationConnectorDefinitionRequest:
		resp = &connectorPB.ListDestinationConnectorDefinitionResponse{}
		pageSize = v.GetPageSize()
		pageToken = v.GetPageToken()
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION)
		isBasicView = (v.GetView() == connectorPB.View_VIEW_BASIC) || (v.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
	}

	dbDefs, totalSize, nextPageToken, err := h.service.ListConnectorDefinition(connType, pageSize, pageToken, isBasicView)
	if err != nil {
		return resp, err
	}

	switch v := resp.(type) {
	case *connectorPB.ListSourceConnectorDefinitionResponse:
		for _, dbDef := range dbDefs {
			v.SourceConnectorDefinitions = append(
				v.SourceConnectorDefinitions,
				DBToPBConnectorDefinition(
					dbDef,
					datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE)).(*connectorPB.SourceConnectorDefinition))
		}
		v.NextPageToken = nextPageToken
		v.TotalSize = totalSize
	case *connectorPB.ListDestinationConnectorDefinitionResponse:
		for _, dbDef := range dbDefs {
			v.DestinationConnectorDefinitions = append(
				v.DestinationConnectorDefinitions,
				DBToPBConnectorDefinition(
					dbDef,
					datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION)).(*connectorPB.DestinationConnectorDefinition))
		}
		v.NextPageToken = nextPageToken
		v.TotalSize = totalSize
	}

	return resp, nil
}

func (h *handler) getConnectorDefinition(ctx context.Context, req interface{}) (resp interface{}, err error) {

	var connID string
	var isBasicView bool

	var connType datamodel.ConnectorType

	switch v := req.(type) {
	case *connectorPB.GetSourceConnectorDefinitionRequest:
		resp = &connectorPB.GetSourceConnectorDefinitionResponse{}
		if connID, err = resource.GetRscNameID(v.GetName()); err != nil {
			return resp, err
		}
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE)
		isBasicView = (v.GetView() == connectorPB.View_VIEW_BASIC) || (v.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
	case *connectorPB.GetDestinationConnectorDefinitionRequest:
		resp = &connectorPB.GetDestinationConnectorDefinitionResponse{}
		if connID, err = resource.GetRscNameID(v.GetName()); err != nil {
			return resp, err
		}
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION)
		isBasicView = (v.GetView() == connectorPB.View_VIEW_BASIC) || (v.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
	}

	dbDef, err := h.service.GetConnectorDefinitionByID(connID, connType, isBasicView)
	if err != nil {
		return resp, err
	}

	switch v := resp.(type) {
	case *connectorPB.GetSourceConnectorDefinitionResponse:
		v.SourceConnectorDefinition = DBToPBConnectorDefinition(dbDef, connType).(*connectorPB.SourceConnectorDefinition)
	case *connectorPB.GetDestinationConnectorDefinitionResponse:
		v.DestinationConnectorDefinition = DBToPBConnectorDefinition(dbDef, connType).(*connectorPB.DestinationConnectorDefinition)
	}

	return resp, nil

}

func (h *handler) createConnector(ctx context.Context, req interface{}) (resp interface{}, err error) {

	logger, _ := logger.GetZapLogger()

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
			return resp, st.Err()
		}

		// Validate SourceConnector JSON Schema
		if err := datamodel.ValidateJSONSchema(datamodel.SrcConnJSONSchema, v.GetSourceConnector(), false); err != nil {
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
			return resp, st.Err()
		}

		connID = v.GetSourceConnector().GetId()

		connConfig, err = v.GetSourceConnector().GetConnector().GetConfiguration().MarshalJSON()
		if err != nil {
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
			return resp, st.Err()
		}

		if err := datamodel.ValidateJSONSchemaString(string(b), v.GetSourceConnector().GetConnector().GetConfiguration()); err != nil {
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
			return resp, st.Err()
		}

		// Validate DestinationConnector JSON Schema
		if err := datamodel.ValidateJSONSchema(datamodel.DstConnJSONSchema, v.GetDestinationConnector(), false); err != nil {
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
			return resp, st.Err()
		}

		connID = v.GetDestinationConnector().GetId()

		connConfig, err = v.GetDestinationConnector().GetConnector().GetConfiguration().MarshalJSON()
		if err != nil {
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
			return resp, st.Err()
		}

		if err := datamodel.ValidateJSONSchemaString(string(b), v.GetDestinationConnector().GetConnector().GetConfiguration()); err != nil {
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
		return resp, st.Err()
	}

	ownerRscName, err := resource.GetOwner(ctx)
	if err != nil {
		return resp, err
	}

	dbConnector := &datamodel.Connector{
		ID:                     connID,
		Owner:                  ownerRscName,
		ConnectorDefinitionUID: connDefUID,
		Tombstone:              false,
		Configuration:          connConfig,
		ConnectorType:          connType,
		Description:            connDesc,
	}

	dbConnector, err = h.service.CreateConnector(dbConnector)
	if err != nil {
		return resp, err
	}

	pbConnector := DBToPBConnector(
		dbConnector,
		connType,
		ownerRscName,
		connDefRscName)

	switch v := resp.(type) {
	case *connectorPB.CreateSourceConnectorResponse:
		v.SourceConnector = pbConnector.(*connectorPB.SourceConnector)
	case *connectorPB.CreateDestinationConnectorResponse:
		v.DestinationConnector = pbConnector.(*connectorPB.DestinationConnector)
	}

	return resp, nil
}

func (h *handler) listConnector(ctx context.Context, req interface{}) (resp interface{}, err error) {

	var pageSize int64
	var pageToken string
	var isBasicView bool

	var connType datamodel.ConnectorType

	var connDefColID string

	switch v := req.(type) {
	case *connectorPB.ListSourceConnectorRequest:
		resp = &connectorPB.ListSourceConnectorResponse{}
		pageSize = v.GetPageSize()
		pageToken = v.GetPageToken()
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE)
		isBasicView = (v.GetView() == connectorPB.View_VIEW_BASIC) || (v.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
		connDefColID = "source-connector-definitions"
	case *connectorPB.ListDestinationConnectorRequest:
		resp = &connectorPB.ListDestinationConnectorResponse{}
		pageSize = v.GetPageSize()
		pageToken = v.GetPageToken()
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION)
		isBasicView = (v.GetView() == connectorPB.View_VIEW_BASIC) || (v.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
		connDefColID = "destination-connector-definitions"
	}

	ownerRscName, err := resource.GetOwner(ctx)
	if err != nil {
		return resp, err
	}

	dbConnectors, totalSize, nextPageToken, err := h.service.ListConnector(ownerRscName, connType, pageSize, pageToken, isBasicView)
	if err != nil {
		return resp, err
	}

	var pbConnectors []interface{}
	for _, dbConnector := range dbConnectors {
		dbConnDef, err := h.service.GetConnectorDefinitionByUID(dbConnector.ConnectorDefinitionUID, true)
		if err != nil {
			return resp, err
		}
		pbConnectors = append(
			pbConnectors,
			DBToPBConnector(
				dbConnector,
				connType,
				dbConnector.Owner,
				fmt.Sprintf("%s/%s", connDefColID, dbConnDef.ID),
			))
	}

	switch v := resp.(type) {
	case *connectorPB.ListSourceConnectorResponse:
		var pbSrcConns []*connectorPB.SourceConnector
		for _, pbConnector := range pbConnectors {
			pbSrcConns = append(pbSrcConns, pbConnector.(*connectorPB.SourceConnector))
		}
		v.SourceConnectors = pbSrcConns
		v.NextPageToken = nextPageToken
		v.TotalSize = totalSize
	case *connectorPB.ListDestinationConnectorResponse:
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

func (h *handler) getConnector(ctx context.Context, req interface{}) (resp interface{}, err error) {

	var isBasicView bool

	var connID string
	var connType datamodel.ConnectorType

	var connDefColID string

	switch v := req.(type) {
	case *connectorPB.GetSourceConnectorRequest:
		resp = &connectorPB.GetSourceConnectorResponse{}
		if connID, err = resource.GetRscNameID(v.GetName()); err != nil {
			return resp, err
		}
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE)
		isBasicView = (v.GetView() == connectorPB.View_VIEW_BASIC) || (v.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
		connDefColID = "source-connector-definitions"
	case *connectorPB.GetDestinationConnectorRequest:
		resp = &connectorPB.GetDestinationConnectorResponse{}
		if connID, err = resource.GetRscNameID(v.GetName()); err != nil {
			return resp, err
		}
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION)
		isBasicView = (v.GetView() == connectorPB.View_VIEW_BASIC) || (v.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
		connDefColID = "destination-connector-definitions"
	}

	ownerRscName, err := resource.GetOwner(ctx)
	if err != nil {
		return resp, err
	}

	dbConnector, err := h.service.GetConnectorByID(connID, ownerRscName, connType, isBasicView)
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
	case *connectorPB.GetSourceConnectorResponse:
		v.SourceConnector = pbConnector.(*connectorPB.SourceConnector)
	case *connectorPB.GetDestinationConnectorResponse:
		v.DestinationConnector = pbConnector.(*connectorPB.DestinationConnector)
	}

	return resp, nil
}

func (h *handler) updateConnector(ctx context.Context, req interface{}) (resp interface{}, err error) {

	logger, _ := logger.GetZapLogger()

	var pbConnectorReq interface{}
	var pbConnectorToUpdate interface{}

	var mask fieldmask_utils.Mask

	var connID string
	var connType datamodel.ConnectorType

	var connDefRscName string
	var connDefUID uuid.UUID

	switch v := req.(type) {
	case *connectorPB.UpdateSourceConnectorRequest:
		resp = &connectorPB.UpdateSourceConnectorResponse{}
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
			return resp, st.Err()
		}
		// Set all OUTPUT_ONLY fields to zero value on the requested payload
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
			return resp, st.Err()
		}
		getResp, err := h.GetSourceConnector(
			ctx,
			&connectorPB.GetSourceConnectorRequest{
				Name: v.GetSourceConnector().GetName(),
				View: connectorPB.View_VIEW_FULL.Enum(),
			})
		if err != nil {
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
			return resp, err
		}

		dbConnDef, err := h.service.GetConnectorDefinitionByID(dbConnDefID, connType, true)
		if err != nil {
			return resp, err
		}

		connDefRscName = fmt.Sprintf("source-connector-definitions/%s", dbConnDef.ID)
		connDefUID = dbConnDef.UID

	case *connectorPB.UpdateDestinationConnectorRequest:

		resp = &connectorPB.UpdateDestinationConnectorResponse{}
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
			return resp, st.Err()
		}

		// Set all OUTPUT_ONLY fields to zero value on the requested payload
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
			return resp, st.Err()
		}

		getResp, err := h.GetDestinationConnector(
			ctx,
			&connectorPB.GetDestinationConnectorRequest{
				Name: v.GetDestinationConnector().GetName(),
				View: connectorPB.View_VIEW_FULL.Enum(),
			})
		if err != nil {
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
			return resp, err
		}

		dbConnDef, err := h.service.GetConnectorDefinitionByID(dbConnDefID, connType, true)
		if err != nil {
			return resp, err
		}

		connDefRscName = fmt.Sprintf("destination-connector-definitions/%s", dbConnDef.ID)
		connDefUID = dbConnDef.UID
	}

	ownerRscName, err := resource.GetOwner(ctx)
	if err != nil {
		return resp, err
	}

	// Only the fields mentioned in the field mask will be copied to `pbPipelineToUpdate`, other fields are left intact
	err = fieldmask_utils.StructToStruct(mask, pbConnectorReq, pbConnectorToUpdate)
	if err != nil {
		return resp, err
	}

	dbConnector, err := h.service.UpdateConnector(connID, ownerRscName, connType, PBToDBConnector(pbConnectorToUpdate, connType, ownerRscName, connDefUID))
	if err != nil {
		return resp, err
	}

	pbConnector := DBToPBConnector(
		dbConnector,
		connType,
		ownerRscName,
		connDefRscName)

	switch v := resp.(type) {
	case *connectorPB.UpdateSourceConnectorResponse:
		v.SourceConnector = pbConnector.(*connectorPB.SourceConnector)
	case *connectorPB.UpdateDestinationConnectorResponse:
		v.DestinationConnector = pbConnector.(*connectorPB.DestinationConnector)
	}

	return resp, nil
}

func (h *handler) deleteConnector(ctx context.Context, req interface{}) (resp interface{}, err error) {

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

	ownerRscName, err := resource.GetOwner(ctx)
	if err != nil {
		return resp, err
	}

	if err := h.service.DeleteConnector(connID, ownerRscName, connType); err != nil {
		return resp, err
	}

	return resp, nil
}

func (h *handler) lookUpConnector(ctx context.Context, req interface{}) (resp interface{}, err error) {

	logger, _ := logger.GetZapLogger()

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

	ownerRscName, err := resource.GetOwner(ctx)
	if err != nil {
		return resp, err
	}

	dbConnector, err := h.service.GetConnectorByUID(connUID, ownerRscName, isBasicView)
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
	case *connectorPB.LookUpSourceConnectorResponse:
		v.SourceConnector = pbConnector.(*connectorPB.SourceConnector)
	case *connectorPB.LookUpDestinationConnectorResponse:
		v.DestinationConnector = pbConnector.(*connectorPB.DestinationConnector)
	}

	return resp, nil
}

func (h *handler) connectConnector(ctx context.Context, req interface{}) (resp interface{}, err error) {

	logger, _ := logger.GetZapLogger()

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
			return resp, st.Err()
		}

		connID, err = resource.GetRscNameID(v.GetName())
		if err != nil {
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
			return resp, err
		}

		dbConnDefID, err := resource.GetRscNameID(getResp.GetSourceConnector().GetSourceConnectorDefinition())
		if err != nil {
			return resp, err
		}

		dbConnDef, err := h.service.GetConnectorDefinitionByID(dbConnDefID, connType, true)
		if err != nil {
			return resp, err
		}

		connDefRscName = fmt.Sprintf("source-connector-definitions/%s", dbConnDef.ID)

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
			return resp, st.Err()
		}

		connID, err = resource.GetRscNameID(v.GetName())
		if err != nil {
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
			return resp, err
		}

		dbConnDefID, err := resource.GetRscNameID(getResp.GetDestinationConnector().GetDestinationConnectorDefinition())
		if err != nil {
			return resp, err
		}

		dbConnDef, err := h.service.GetConnectorDefinitionByID(dbConnDefID, connType, true)
		if err != nil {
			return resp, err
		}

		connDefRscName = fmt.Sprintf("destination-connector-definitions/%s", dbConnDef.ID)
	}

	ownerRscName, err := resource.GetOwner(ctx)
	if err != nil {
		return resp, err
	}

	dbConnector, err := h.service.UpdateConnectorState(connID, ownerRscName, connType, datamodel.ConnectorState(connectorPB.Connector_STATE_CONNECTED))
	if err != nil {
		return resp, err
	}

	pbConnector := DBToPBConnector(
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

func (h *handler) disconnectConnector(ctx context.Context, req interface{}) (resp interface{}, err error) {

	logger, _ := logger.GetZapLogger()

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
			return resp, st.Err()
		}

		connID, err = resource.GetRscNameID(v.GetName())
		if err != nil {
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
			return resp, err
		}

		dbConnDefID, err := resource.GetRscNameID(getResp.GetSourceConnector().GetSourceConnectorDefinition())
		if err != nil {
			return resp, err
		}

		dbConnDef, err := h.service.GetConnectorDefinitionByID(dbConnDefID, connType, true)
		if err != nil {
			return resp, err
		}

		connDefRscName = fmt.Sprintf("source-connector-definitions/%s", dbConnDef.ID)

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
			return resp, st.Err()
		}

		connID, err = resource.GetRscNameID(v.GetName())
		if err != nil {
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
			return resp, err
		}

		dbConnDefID, err := resource.GetRscNameID(getResp.GetDestinationConnector().GetDestinationConnectorDefinition())
		if err != nil {
			return resp, err
		}

		dbConnDef, err := h.service.GetConnectorDefinitionByID(dbConnDefID, connType, true)
		if err != nil {
			return resp, err
		}

		connDefRscName = fmt.Sprintf("destination-connector-definitions/%s", dbConnDef.ID)
	}

	ownerRscName, err := resource.GetOwner(ctx)
	if err != nil {
		return resp, err
	}

	dbConnector, err := h.service.UpdateConnectorState(connID, ownerRscName, connType, datamodel.ConnectorState(connectorPB.Connector_STATE_DISCONNECTED))
	if err != nil {
		return resp, err
	}

	pbConnector := DBToPBConnector(
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

func (h *handler) renameConnector(ctx context.Context, req interface{}) (resp interface{}, err error) {

	logger, _ := logger.GetZapLogger()

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
			return resp, st.Err()
		}

		connID, err = resource.GetRscNameID(v.GetName())
		if err != nil {
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
			return resp, err
		}

		dbConnDefID, err := resource.GetRscNameID(getResp.GetSourceConnector().GetSourceConnectorDefinition())
		if err != nil {
			return resp, err
		}

		dbConnDef, err := h.service.GetConnectorDefinitionByID(dbConnDefID, connType, true)
		if err != nil {
			return resp, err
		}

		connDefRscName = fmt.Sprintf("source-connector-definitions/%s", dbConnDef.ID)

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
			return resp, err
		}

		dbConnDefID, err := resource.GetRscNameID(getResp.GetDestinationConnector().GetDestinationConnectorDefinition())
		if err != nil {
			return resp, err
		}

		dbConnDef, err := h.service.GetConnectorDefinitionByID(dbConnDefID, connType, true)
		if err != nil {
			return resp, err
		}

		connDefRscName = fmt.Sprintf("destination-connector-definitions/%s", dbConnDef.ID)
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
		return resp, st.Err()
	}

	ownerRscName, err := resource.GetOwner(ctx)
	if err != nil {
		return resp, err
	}

	dbConnector, err := h.service.UpdateConnectorID(connID, ownerRscName, connType, connNewID)
	if err != nil {
		return resp, err
	}

	pbConnector := DBToPBConnector(
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
