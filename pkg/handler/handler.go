package handler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/gogo/status"
	"github.com/iancoleman/strcase"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/datatypes"

	fieldmask_utils "github.com/mennanov/fieldmask-utils"

	"github.com/instill-ai/connector-backend/internal/resource"
	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/connector-backend/pkg/service"
	"github.com/instill-ai/x/checkfield"

	connectorPB "github.com/instill-ai/protogen-go/connector/v1alpha"
)

type handler struct {
	connectorPB.UnimplementedConnectorServiceServer
	service service.Service
}

// NewHandler initiates a handler instance
func NewHandler(s service.Service) connectorPB.ConnectorServiceServer {
	datamodel.InitJSONSchema()
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
		if connID, err = resource.GetNameID(v.GetName()); err != nil {
			return resp, err
		}
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE)
		isBasicView = (v.GetView() == connectorPB.View_VIEW_BASIC) || (v.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
	case *connectorPB.GetDestinationConnectorDefinitionRequest:
		resp = &connectorPB.GetDestinationConnectorDefinitionResponse{}
		if connID, err = resource.GetNameID(v.GetName()); err != nil {
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

	var connID string
	var connDesc sql.NullString
	var connType datamodel.ConnectorType
	var connConfig datatypes.JSON

	var connDefUID uuid.UUID
	var connDefRscName string

	switch v := req.(type) {
	case *connectorPB.CreateSourceConnectorRequest:

		resp = &connectorPB.CreateSourceConnectorResponse{}

		// Validate SourceConnector JSON Schema
		if err := datamodel.ValidateJSONSchema(datamodel.SrcConnJSONSchema, v.GetSourceConnector(), false); err != nil {
			return resp, status.Error(codes.InvalidArgument, err.Error())
		}

		// Set all OUTPUT_ONLY fields to zero value on the requested payload
		if err := checkfield.CheckCreateOutputOnlyFields(v.GetSourceConnector(), outputOnlyFields); err != nil {
			return resp, status.Error(codes.InvalidArgument, err.Error())
		}

		// Return error if REQUIRED fields are not provided in the requested payload
		if err := checkfield.CheckRequiredFields(v.GetSourceConnector(), append(createRequiredFields, sourceImmutableFields...)); err != nil {
			return resp, status.Error(codes.InvalidArgument, err.Error())
		}

		connID = v.GetSourceConnector().GetId()

		connConfig = []byte(v.GetSourceConnector().GetConnector().GetConfiguration())

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
			return resp, status.Error(codes.InvalidArgument, err.Error())
		}

		if err := datamodel.ValidateJSONSchemaString(string(b), v.GetSourceConnector().GetConnector().GetConfiguration()); err != nil {
			return resp, status.Error(codes.InvalidArgument, err.Error())
		}

		connDefUID, err = uuid.FromString(connDefResp.SourceConnectorDefinition.GetUid())
		if err != nil {
			return resp, status.Error(codes.InvalidArgument, err.Error())
		}

		connDefRscName = fmt.Sprintf("source-connector-definitions/%s", connDefResp.SourceConnectorDefinition.GetId())

	case *connectorPB.CreateDestinationConnectorRequest:

		resp = &connectorPB.CreateDestinationConnectorResponse{}

		// Validate DestinationConnector JSON Schema
		if err := datamodel.ValidateJSONSchema(datamodel.DstConnJSONSchema, v.GetDestinationConnector(), false); err != nil {
			return resp, status.Error(codes.InvalidArgument, err.Error())
		}

		// Validate DestinationConnector configuration JSON Schema

		// Set all OUTPUT_ONLY fields to zero value on the requested payload
		if err := checkfield.CheckCreateOutputOnlyFields(v.GetDestinationConnector(), outputOnlyFields); err != nil {
			return resp, status.Error(codes.InvalidArgument, err.Error())
		}

		// Return error if REQUIRED fields are not provided in the requested payload
		if err := checkfield.CheckRequiredFields(v.GetDestinationConnector(), append(createRequiredFields, destinationImmutableFields...)); err != nil {
			return resp, status.Error(codes.InvalidArgument, err.Error())
		}

		connID = v.GetDestinationConnector().GetId()

		connConfig = []byte(v.GetDestinationConnector().GetConnector().GetConfiguration())

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

		// Validate SourceConnector configuration JSON Schema
		connSpec := connDefResp.GetDestinationConnectorDefinition().GetConnectorDefinition().GetSpec().GetConnectionSpecification()
		b, err := protojson.Marshal(connSpec)
		if err != nil {
			return resp, status.Error(codes.InvalidArgument, err.Error())
		}

		if err := datamodel.ValidateJSONSchemaString(string(b), v.GetDestinationConnector().GetConnector().GetConfiguration()); err != nil {
			return resp, status.Error(codes.InvalidArgument, err.Error())
		}

		connDefUID, err = uuid.FromString(connDefResp.DestinationConnectorDefinition.GetUid())
		if err != nil {
			return resp, status.Error(codes.InvalidArgument, err.Error())
		}

		connDefRscName = fmt.Sprintf("destination-connector-definitions/%s", connDefResp.DestinationConnectorDefinition.GetId())
	}

	// Return error if resource ID does not follow RFC-1034
	if err := checkfield.CheckResourceID(connID); err != nil {
		return resp, status.Error(codes.InvalidArgument, err.Error())
	}

	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return resp, err
	}

	dbConnector := &datamodel.Connector{
		ID:                     connID,
		Owner:                  owner,
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
		owner,
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

	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return resp, err
	}

	dbConnectors, totalSize, nextPageToken, err := h.service.ListConnector(owner, connType, pageSize, pageToken, isBasicView)
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
		if connID, err = resource.GetNameID(v.GetName()); err != nil {
			return resp, err
		}
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE)
		isBasicView = (v.GetView() == connectorPB.View_VIEW_BASIC) || (v.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
		connDefColID = "source-connector-definitions"
	case *connectorPB.GetDestinationConnectorRequest:
		resp = &connectorPB.GetDestinationConnectorResponse{}
		if connID, err = resource.GetNameID(v.GetName()); err != nil {
			return resp, err
		}
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION)
		isBasicView = (v.GetView() == connectorPB.View_VIEW_BASIC) || (v.GetView() == connectorPB.View_VIEW_UNSPECIFIED)
		connDefColID = "destination-connector-definitions"
	}

	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return resp, err
	}

	dbConnector, err := h.service.GetConnectorByID(connID, owner, connType, isBasicView)
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
		if !pbUpdateMask.IsValid(v.GetSourceConnector()) {
			return resp, status.Error(codes.InvalidArgument, "The update_mask is invalid")
		}
		// Set all OUTPUT_ONLY fields to zero value on the requested payload
		pbUpdateMask, err = checkfield.CheckUpdateOutputOnlyFields(pbUpdateMask, outputOnlyFields)
		if err != nil {
			return resp, status.Error(codes.InvalidArgument, err.Error())
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
			return resp, status.Error(codes.InvalidArgument, err.Error())
		}

		mask, err = fieldmask_utils.MaskFromProtoFieldMask(pbUpdateMask, strcase.ToCamel)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		if mask.IsEmpty() {
			return &connectorPB.UpdateSourceConnectorResponse{
				SourceConnector: getResp.GetSourceConnector(),
			}, nil
		}

		pbConnectorToUpdate = getResp.GetSourceConnector()

		connID = getResp.GetSourceConnector().GetId()
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE)

		dbConnDefID, err := resource.GetNameID(v.GetSourceConnector().GetSourceConnectorDefinition())
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
		if !pbUpdateMask.IsValid(v.GetDestinationConnector()) {
			return resp, status.Error(codes.InvalidArgument, "The update_mask is invalid")
		}
		// Set all OUTPUT_ONLY fields to zero value on the requested payload
		pbUpdateMask, err = checkfield.CheckUpdateOutputOnlyFields(pbUpdateMask, outputOnlyFields)
		if err != nil {
			return resp, status.Error(codes.InvalidArgument, err.Error())
		}

		getResp, err := h.GetDestinationConnector(
			ctx,
			&connectorPB.GetDestinationConnectorRequest{
				Name: fmt.Sprintf("destination-connectors/%s", v.GetDestinationConnector().GetId()),
				View: connectorPB.View_VIEW_FULL.Enum(),
			})
		if err != nil {
			return resp, err
		}

		// Return error if IMMUTABLE fields are intentionally changed
		if err := checkfield.CheckUpdateImmutableFields(v.GetDestinationConnector(), getResp.GetDestinationConnector(), destinationImmutableFields); err != nil {
			return resp, status.Error(codes.InvalidArgument, err.Error())
		}

		mask, err = fieldmask_utils.MaskFromProtoFieldMask(pbUpdateMask, strcase.ToCamel)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		if mask.IsEmpty() {
			return &connectorPB.UpdateDestinationConnectorResponse{
				DestinationConnector: getResp.GetDestinationConnector(),
			}, nil
		}

		pbConnectorToUpdate = getResp.GetDestinationConnector()

		connID = getResp.GetDestinationConnector().GetId()
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION)

		dbConnDefID, err := resource.GetNameID(v.GetDestinationConnector().GetDestinationConnectorDefinition())
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

	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return resp, err
	}

	// Only the fields mentioned in the field mask will be copied to `pbPipelineToUpdate`, other fields are left intact
	err = fieldmask_utils.StructToStruct(mask, pbConnectorReq, pbConnectorToUpdate)
	if err != nil {
		return resp, err
	}

	dbConnector, err := h.service.UpdateConnector(connID, owner, connType, PBToDBConnector(pbConnectorToUpdate, connType, owner, connDefUID))
	if err != nil {
		return resp, err
	}

	pbConnector := DBToPBConnector(
		dbConnector,
		connType,
		owner,
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
		if connID, err = resource.GetNameID(v.GetName()); err != nil {
			return resp, err
		}
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE)
	case *connectorPB.DeleteDestinationConnectorRequest:
		resp = &connectorPB.DeleteDestinationConnectorResponse{}
		if connID, err = resource.GetNameID(v.GetName()); err != nil {
			return resp, err
		}
		connType = datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION)
	}

	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return resp, err
	}

	if err := h.service.DeleteConnector(connID, owner, connType); err != nil {
		return resp, err
	}

	return resp, nil
}

func (h *handler) lookUpConnector(ctx context.Context, req interface{}) (resp interface{}, err error) {
	var isBasicView bool

	var connUID uuid.UUID
	var connType datamodel.ConnectorType

	var connDefColID string

	switch v := req.(type) {
	case *connectorPB.LookUpSourceConnectorRequest:
		resp = &connectorPB.LookUpSourceConnectorResponse{}

		// Return error if REQUIRED fields are not provided in the requested payload
		if err := checkfield.CheckRequiredFields(v, lookUpRequiredFields); err != nil {
			return resp, status.Error(codes.InvalidArgument, err.Error())
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
			return resp, status.Error(codes.InvalidArgument, err.Error())
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

	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return resp, err
	}

	dbConnector, err := h.service.GetConnectorByUID(connUID, owner, connType, isBasicView)
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

func (h *handler) renameConnector(ctx context.Context, req interface{}) (resp interface{}, err error) {

	var connID string
	var connNewID string
	var connType datamodel.ConnectorType

	var connDefRscName string

	switch v := req.(type) {
	case *connectorPB.RenameSourceConnectorRequest:
		resp = &connectorPB.RenameSourceConnectorResponse{}

		// Return error if REQUIRED fields are not provided in the requested payload
		if err := checkfield.CheckRequiredFields(v, renameSourceRequiredFields); err != nil {
			return resp, status.Error(codes.InvalidArgument, err.Error())
		}

		connID, err = resource.GetNameID(v.GetName())
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

		dbConnDefID, err := resource.GetNameID(getResp.GetSourceConnector().GetSourceConnectorDefinition())
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
			return resp, status.Error(codes.InvalidArgument, err.Error())
		}

		connID, err = resource.GetNameID(v.GetName())
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

		dbConnDefID, err := resource.GetNameID(getResp.GetDestinationConnector().GetDestinationConnectorDefinition())
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
		return resp, status.Error(codes.InvalidArgument, err.Error())
	}

	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return resp, err
	}

	dbConnector, err := h.service.UpdateConnectorID(connID, owner, connType, connNewID)
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
