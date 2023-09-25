package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/gogo/status"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/redis/go-redis/v9"
	"go.einride.tech/aip/filtering"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/connector-backend/internal/resource"
	"github.com/instill-ai/connector-backend/pkg/connector"
	"github.com/instill-ai/connector-backend/pkg/constant"
	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/connector-backend/pkg/logger"
	"github.com/instill-ai/connector-backend/pkg/repository"
	"github.com/instill-ai/connector-backend/pkg/utils"
	"github.com/instill-ai/x/paginate"
	"github.com/instill-ai/x/sterr"

	connectorBase "github.com/instill-ai/component/pkg/base"
	mgmtPB "github.com/instill-ai/protogen-go/base/mgmt/v1alpha"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	controllerPB "github.com/instill-ai/protogen-go/vdp/controller/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

// Service interface
type Service interface {
	ListConnectorDefinitions(ctx context.Context, pageSize int64, pageToken string, view connectorPB.View, filter filtering.Filter) ([]*connectorPB.ConnectorDefinition, int64, string, error)
	GetConnectorResourceByUID(ctx context.Context, userUid uuid.UUID, uid uuid.UUID, view connectorPB.View, credentialMask bool) (*connectorPB.ConnectorResource, error)
	GetConnectorDefinitionByID(ctx context.Context, id string, view connectorPB.View) (*connectorPB.ConnectorDefinition, error)
	GetConnectorDefinitionByUIDAdmin(ctx context.Context, uid uuid.UUID, view connectorPB.View) (*connectorPB.ConnectorDefinition, error)

	// Connector common
	ListConnectorResources(ctx context.Context, userUid uuid.UUID, pageSize int64, pageToken string, view connectorPB.View, filter filtering.Filter, showDeleted bool) ([]*connectorPB.ConnectorResource, int64, string, error)
	CreateUserConnectorResource(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, connectorResource *connectorPB.ConnectorResource) (*connectorPB.ConnectorResource, error)
	ListUserConnectorResources(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pageSize int64, pageToken string, view connectorPB.View, filter filtering.Filter, showDeleted bool) ([]*connectorPB.ConnectorResource, int64, string, error)
	GetUserConnectorResourceByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, view connectorPB.View, credentialMask bool) (*connectorPB.ConnectorResource, error)
	UpdateUserConnectorResourceByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, connectorResource *connectorPB.ConnectorResource) (*connectorPB.ConnectorResource, error)
	UpdateUserConnectorResourceIDByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, newID string) (*connectorPB.ConnectorResource, error)
	UpdateUserConnectorResourceStateByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, state connectorPB.ConnectorResource_State) (*connectorPB.ConnectorResource, error)
	DeleteUserConnectorResourceByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string) error

	ListConnectorResourcesAdmin(ctx context.Context, pageSize int64, pageToken string, view connectorPB.View, filter filtering.Filter, showDeleted bool) ([]*connectorPB.ConnectorResource, int64, string, error)
	GetConnectorResourceByUIDAdmin(ctx context.Context, uid uuid.UUID, view connectorPB.View) (*connectorPB.ConnectorResource, error)

	// Execute connector
	Execute(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, inputs []*structpb.Struct) ([]*structpb.Struct, error)

	// Shared public/private method for checking connector's connection
	CheckConnectorResourceByUID(ctx context.Context, connUID uuid.UUID) (*connectorPB.ConnectorResource_State, error)

	// Controller custom service
	GetResourceState(uid uuid.UUID) (*connectorPB.ConnectorResource_State, error)
	UpdateResourceState(uid uuid.UUID, state connectorPB.ConnectorResource_State, progress *int32) error
	DeleteResourceState(uid uuid.UUID) error

	// Influx API
	WriteNewDataPoint(ctx context.Context, data utils.UsageMetricData, pipelineMetadata *structpb.Value) error

	// Helper functions
	GetRscNamespaceAndNameID(path string) (resource.Namespace, string, error)
	GetRscNamespaceAndPermalinkUID(path string) (resource.Namespace, uuid.UUID, error)
	ConvertOwnerPermalinkToName(permalink string) (string, error)
	ConvertOwnerNameToPermalink(name string) (string, error)
	RemoveCredentialFieldsWithMaskString(dbConnDefID string, config *structpb.Struct)
	GetUser(ctx context.Context) (string, uuid.UUID, error)
}

type service struct {
	repository                  repository.Repository
	mgmtPrivateServiceClient    mgmtPB.MgmtPrivateServiceClient
	pipelinePublicServiceClient pipelinePB.PipelinePublicServiceClient
	controllerClient            controllerPB.ControllerPrivateServiceClient
	connectorAll                connectorBase.IConnector
	influxDBWriteClient         api.WriteAPI
	redisClient                 *redis.Client
	connectors                  connectorBase.IConnector
}

// NewService initiates a service instance
func NewService(
	t context.Context,
	r repository.Repository,
	u mgmtPB.MgmtPrivateServiceClient,
	p pipelinePB.PipelinePublicServiceClient,
	c controllerPB.ControllerPrivateServiceClient,
	rc *redis.Client,
	i api.WriteAPI,
) Service {
	logger, _ := logger.GetZapLogger(t)
	return &service{
		repository:                  r,
		mgmtPrivateServiceClient:    u,
		pipelinePublicServiceClient: p,
		controllerClient:            c,
		connectorAll:                connector.InitConnectorAll(logger),
		redisClient:                 rc,
		influxDBWriteClient:         i,
		connectors:                  connector.InitConnectorAll(logger),
	}
}

// GetUser returns the api user
func (s *service) GetUser(ctx context.Context) (string, uuid.UUID, error) {
	// Verify if "jwt-sub" is in the header
	headerUserUId := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
	if headerUserUId != "" {
		_, err := uuid.FromString(headerUserUId)
		if err != nil {
			return "", uuid.Nil, status.Errorf(codes.Unauthenticated, "Unauthorized")
		}
		userResp, err := s.mgmtPrivateServiceClient.LookUpUserAdmin(context.Background(), &mgmtPB.LookUpUserAdminRequest{Permalink: "users/" + headerUserUId})
		if err != nil {
			return "", uuid.Nil, status.Errorf(codes.Unauthenticated, "Unauthorized")
		}

		return userResp.User.Id, uuid.FromStringOrNil(headerUserUId), nil
	}
	return "", uuid.Nil, status.Errorf(codes.Unauthenticated, "Unauthorized")
}

func (s *service) injectUserToContext(ctx context.Context, userPermalink string) context.Context {
	ctx = metadata.AppendToOutgoingContext(ctx, "Jwt-Sub", strings.Split(userPermalink, "/")[1])
	return ctx
}

func (s *service) ConvertOwnerPermalinkToName(permalink string) (string, error) {
	userResp, err := s.mgmtPrivateServiceClient.LookUpUserAdmin(context.Background(), &mgmtPB.LookUpUserAdminRequest{Permalink: permalink})
	if err != nil {
		return "", fmt.Errorf("ConvertNamespaceToOwnerPath error")
	}
	return fmt.Sprintf("users/%s", userResp.User.Id), nil
}
func (s *service) ConvertOwnerNameToPermalink(name string) (string, error) {
	userResp, err := s.mgmtPrivateServiceClient.GetUserAdmin(context.Background(), &mgmtPB.GetUserAdminRequest{Name: name})
	if err != nil {
		return "", fmt.Errorf("ConvertOwnerNameToPermalink error")
	}
	return fmt.Sprintf("users/%s", *userResp.User.Uid), nil
}

func (s *service) GetRscNamespaceAndNameID(path string) (resource.Namespace, string, error) {
	splits := strings.Split(path, "/")
	if len(splits) < 2 {
		return resource.Namespace{}, "", fmt.Errorf("namespace error")
	}
	uidStr, err := s.ConvertOwnerNameToPermalink(fmt.Sprintf("%s/%s", splits[0], splits[1]))
	if err != nil {
		return resource.Namespace{}, "", fmt.Errorf("namespace error")
	}
	if len(splits) < 4 {
		return resource.Namespace{
			NsType: resource.NamespaceType(splits[0]),
			NsUid:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
		}, "", nil
	}
	return resource.Namespace{
		NsType: resource.NamespaceType(splits[0]),
		NsUid:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
	}, splits[3], nil
}

func (s *service) GetRscNamespaceAndPermalinkUID(path string) (resource.Namespace, uuid.UUID, error) {
	splits := strings.Split(path, "/")
	if len(splits) < 2 {
		return resource.Namespace{}, uuid.Nil, fmt.Errorf("namespace error")
	}
	uidStr, err := s.ConvertOwnerNameToPermalink(fmt.Sprintf("%s/%s", splits[0], splits[1]))
	if err != nil {
		return resource.Namespace{}, uuid.Nil, fmt.Errorf("namespace error")
	}
	if len(splits) < 4 {
		return resource.Namespace{
			NsType: resource.NamespaceType(splits[0]),
			NsUid:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
		}, uuid.Nil, nil
	}
	return resource.Namespace{
		NsType: resource.NamespaceType(splits[0]),
		NsUid:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
	}, uuid.FromStringOrNil(splits[3]), nil
}

func (s *service) RemoveCredentialFieldsWithMaskString(dbConnDefID string, config *structpb.Struct) {
	connector.RemoveCredentialFieldsWithMaskString(s.connectors, dbConnDefID, config)
}

func (s *service) ListConnectorDefinitions(ctx context.Context, pageSize int64, pageToken string, view connectorPB.View, filter filtering.Filter) ([]*connectorPB.ConnectorDefinition, int64, string, error) {

	logger, _ := logger.GetZapLogger(ctx)

	var err error
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
			return nil, 0, "", st.Err()
		}
	}

	if pageSize == 0 {
		pageSize = repository.DefaultPageSize
	} else if pageSize > repository.MaxPageSize {
		pageSize = repository.MaxPageSize
	}

	unfilteredDefs := s.connectors.ListConnectorDefinitions()

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
			if _, ok := typeMap[unfilteredDefs[idx].Type.String()]; ok {
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

	pageDefs := []*connectorPB.ConnectorDefinition{}

	for _, def := range page {
		def = proto.Clone(def).(*connectorPB.ConnectorDefinition)
		if view == connectorPB.View_VIEW_BASIC {
			def.Spec = nil
		}
		def.VendorAttributes = nil
		pageDefs = append(pageDefs, def)
	}
	return pageDefs, int64(len(defs)), nextPageToken, err

}

func (s *service) GetConnectorResourceByUID(ctx context.Context, userUid uuid.UUID, uid uuid.UUID, view connectorPB.View, credentialMask bool) (*connectorPB.ConnectorResource, error) {

	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbConnectorResource, err := s.repository.GetConnectorResourceByUID(ctx, userPermalink, uid, view == connectorPB.View_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	return s.convertDatamodelToProto(ctx, dbConnectorResource, view, credentialMask)
}

func (s *service) GetConnectorDefinitionByID(ctx context.Context, id string, view connectorPB.View) (*connectorPB.ConnectorDefinition, error) {

	def, err := s.connectors.GetConnectorDefinitionById(id)
	if err != nil {
		return nil, err
	}
	def = proto.Clone(def).(*connectorPB.ConnectorDefinition)
	if view == connectorPB.View_VIEW_BASIC {
		def.Spec = nil
	}
	def.VendorAttributes = nil

	return def, nil
}
func (s *service) GetConnectorDefinitionByUIDAdmin(ctx context.Context, uid uuid.UUID, view connectorPB.View) (*connectorPB.ConnectorDefinition, error) {

	def, err := s.connectors.GetConnectorDefinitionByUid(uid)
	if err != nil {
		return nil, err
	}
	def = proto.Clone(def).(*connectorPB.ConnectorDefinition)
	if view == connectorPB.View_VIEW_BASIC {
		def.Spec = nil
	}
	def.VendorAttributes = nil

	return def, nil
}

func (s *service) ListConnectorResources(ctx context.Context, userUid uuid.UUID, pageSize int64, pageToken string, view connectorPB.View, filter filtering.Filter, showDeleted bool) ([]*connectorPB.ConnectorResource, int64, string, error) {

	userPermalink := resource.UserUidToUserPermalink(userUid)

	dbConnectorResources, totalSize, nextPageToken, err := s.repository.ListConnectorResources(ctx, userPermalink, pageSize, pageToken, view == connectorPB.View_VIEW_BASIC, filter, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}

	pbConnectorResources, err := s.convertDatamodelArrayToProtoArray(ctx, dbConnectorResources, view, true)
	return pbConnectorResources, totalSize, nextPageToken, err

}

func (s *service) CreateUserConnectorResource(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, connectorResource *connectorPB.ConnectorResource) (*connectorPB.ConnectorResource, error) {

	logger, _ := logger.GetZapLogger(ctx)

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	connDefResp, err := s.connectors.GetConnectorDefinitionById(strings.Split(connectorResource.ConnectorDefinitionName, "/")[1])
	if err != nil {
		return nil, err
	}

	connDefUID, err := uuid.FromString(connDefResp.GetUid())
	if err != nil {
		return nil, err
	}

	connConfig, err := connectorResource.GetConfiguration().MarshalJSON()
	if err != nil {

		return nil, err
	}

	connDesc := sql.NullString{
		String: connectorResource.GetDescription(),
		Valid:  len(connectorResource.GetDescription()) > 0,
	}

	dbConnectorResourceToCreate := &datamodel.ConnectorResource{
		ID:                     connectorResource.Id,
		Owner:                  resource.UserUidToUserPermalink(userUid),
		ConnectorDefinitionUID: connDefUID,
		Tombstone:              false,
		Configuration:          connConfig,
		ConnectorType:          datamodel.ConnectorResourceType(connDefResp.GetType()),
		Description:            connDesc,
		Visibility:             datamodel.ConnectorResourceVisibility(connectorResource.Visibility),
	}

	if existingConnector, _ := s.repository.GetUserConnectorResourceByID(ctx, ownerPermalink, userPermalink, dbConnectorResourceToCreate.ID, true); existingConnector != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.AlreadyExists,
			"[service] create connector",
			"connectors",
			fmt.Sprintf("Connector id %s", dbConnectorResourceToCreate.ID),
			dbConnectorResourceToCreate.Owner,
			"Already exists",
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return nil, st.Err()
	}

	if err := s.repository.CreateUserConnectorResource(ctx, ownerPermalink, userPermalink, dbConnectorResourceToCreate); err != nil {
		return nil, err
	}

	// User desire state = DISCONNECTED
	if err := s.repository.UpdateUserConnectorResourceStateByID(ctx, ownerPermalink, userPermalink, dbConnectorResourceToCreate.ID, datamodel.ConnectorResourceState(connectorPB.ConnectorResource_STATE_DISCONNECTED)); err != nil {
		return nil, err
	}
	if err := s.UpdateResourceState(dbConnectorResourceToCreate.UID, connectorPB.ConnectorResource_STATE_DISCONNECTED, nil); err != nil {
		return nil, err
	}

	dbConnectorResource, err := s.repository.GetUserConnectorResourceByID(ctx, ownerPermalink, userPermalink, dbConnectorResourceToCreate.ID, false)
	if err != nil {
		return nil, err
	}

	return s.convertDatamodelToProto(ctx, dbConnectorResource, connectorPB.View_VIEW_FULL, true)

}

func (s *service) ListUserConnectorResources(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pageSize int64, pageToken string, view connectorPB.View, filter filtering.Filter, showDeleted bool) ([]*connectorPB.ConnectorResource, int64, string, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbConnectorResources, totalSize, nextPageToken, err := s.repository.ListUserConnectorResources(ctx, ownerPermalink, userPermalink, pageSize, pageToken, view == connectorPB.View_VIEW_BASIC, filter, showDeleted)

	if err != nil {
		return nil, 0, "", err
	}

	pbConnectorResources, err := s.convertDatamodelArrayToProtoArray(ctx, dbConnectorResources, view, true)
	return pbConnectorResources, totalSize, nextPageToken, err

}

func (s *service) ListConnectorResourcesAdmin(ctx context.Context, pageSize int64, pageToken string, view connectorPB.View, filter filtering.Filter, showDeleted bool) ([]*connectorPB.ConnectorResource, int64, string, error) {

	dbConnectorResources, totalSize, nextPageToken, err := s.repository.ListConnectorResourcesAdmin(ctx, pageSize, pageToken, view == connectorPB.View_VIEW_BASIC, filter, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}

	pbConnectorResources, err := s.convertDatamodelArrayToProtoArray(ctx, dbConnectorResources, view, true)
	return pbConnectorResources, totalSize, nextPageToken, err
}

func (s *service) GetUserConnectorResourceByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, view connectorPB.View, credentialMask bool) (*connectorPB.ConnectorResource, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbConnectorResource, err := s.repository.GetUserConnectorResourceByID(ctx, ownerPermalink, userPermalink, id, view == connectorPB.View_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	return s.convertDatamodelToProto(ctx, dbConnectorResource, view, credentialMask)
}

func (s *service) GetConnectorResourceByUIDAdmin(ctx context.Context, uid uuid.UUID, view connectorPB.View) (*connectorPB.ConnectorResource, error) {

	dbConnectorResource, err := s.repository.GetConnectorResourceByUIDAdmin(ctx, uid, view == connectorPB.View_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	return s.convertDatamodelToProto(ctx, dbConnectorResource, view, true)
}

func (s *service) UpdateUserConnectorResourceByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, connectorResource *connectorPB.ConnectorResource) (*connectorPB.ConnectorResource, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	dbConnectorResourceToUpdate, err := s.convertProtoToDatamodel(ctx, connectorResource)
	if err != nil {
		return nil, err
	}
	dbConnectorResourceToUpdate.Owner = ownerPermalink

	if err := s.repository.UpdateUserConnectorResourceByID(ctx, ownerPermalink, userPermalink, id, dbConnectorResourceToUpdate); err != nil {
		return nil, err
	}

	// Check connector state
	if err := s.UpdateResourceState(dbConnectorResourceToUpdate.UID, connectorPB.ConnectorResource_STATE_DISCONNECTED, nil); err != nil {
		return nil, err
	}

	dbConnectorResource, err := s.repository.GetUserConnectorResourceByID(ctx, ownerPermalink, userPermalink, dbConnectorResourceToUpdate.ID, false)
	if err != nil {
		return nil, err
	}

	return s.convertDatamodelToProto(ctx, dbConnectorResource, connectorPB.View_VIEW_FULL, true)

}

func (s *service) DeleteUserConnectorResourceByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string) error {
	logger, _ := logger.GetZapLogger(ctx)

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	dbConnector, err := s.repository.GetUserConnectorResourceByID(ctx, ownerPermalink, userPermalink, id, false)
	if err != nil {
		return err
	}

	filter := fmt.Sprintf("recipe.components.resource_name:\"connector-resources/%s\"", dbConnector.UID)

	pipeResp, err := s.pipelinePublicServiceClient.ListPipelines(s.injectUserToContext(context.Background(), ownerPermalink), &pipelinePB.ListPipelinesRequest{
		Filter: &filter,
	})
	if err != nil {
		return err
	}

	if len(pipeResp.Pipelines) > 0 {
		var pipeIDs []string
		for _, pipe := range pipeResp.Pipelines {
			pipeIDs = append(pipeIDs, pipe.GetId())
		}
		st, err := sterr.CreateErrorPreconditionFailure(
			"[service] delete connector",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "DELETE",
					Subject:     fmt.Sprintf("id %s", id),
					Description: fmt.Sprintf("The connector is still in use by pipeline: %s", strings.Join(pipeIDs, " ")),
				},
			})
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	}

	if err := s.DeleteResourceState(dbConnector.UID); err != nil {
		return err
	}

	return s.repository.DeleteUserConnectorResourceByID(ctx, ownerPermalink, userPermalink, id)
}

func (s *service) UpdateUserConnectorResourceStateByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, state connectorPB.ConnectorResource_State) (*connectorPB.ConnectorResource, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	// Validation: trigger and response connector cannot be disconnected
	conn, err := s.repository.GetUserConnectorResourceByID(ctx, ownerPermalink, userPermalink, id, false)
	if err != nil {
		return nil, err
	}

	if conn.Tombstone {
		st, _ := sterr.CreateErrorPreconditionFailure(
			"[service] update connector state",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "STATE",
					Subject:     fmt.Sprintf("id %s", id),
					Description: "the connector definition is deprecated, you can not use anymore",
				},
			})
		return nil, st.Err()
	}

	switch state {
	case connectorPB.ConnectorResource_STATE_CONNECTED:

		// Set connector state to user desire state
		if err := s.repository.UpdateUserConnectorResourceStateByID(ctx, ownerPermalink, userPermalink, id, datamodel.ConnectorResourceState(connectorPB.ConnectorResource_STATE_CONNECTED)); err != nil {
			return nil, err
		}

		if err := s.UpdateResourceState(conn.UID, connectorPB.ConnectorResource_STATE_CONNECTED, nil); err != nil {
			return nil, err
		}

	case connectorPB.ConnectorResource_STATE_DISCONNECTED:

		if err := s.repository.UpdateUserConnectorResourceStateByID(ctx, ownerPermalink, userPermalink, id, datamodel.ConnectorResourceState(connectorPB.ConnectorResource_STATE_DISCONNECTED)); err != nil {
			return nil, err
		}
		if err := s.UpdateResourceState(conn.UID, connectorPB.ConnectorResource_State(state), nil); err != nil {
			return nil, err
		}
	}

	dbConnectorResource, err := s.repository.GetUserConnectorResourceByID(ctx, ownerPermalink, userPermalink, id, false)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	return s.convertDatamodelToProto(ctx, dbConnectorResource, connectorPB.View_VIEW_FULL, true)
}

func (s *service) UpdateUserConnectorResourceIDByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, newID string) (*connectorPB.ConnectorResource, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	if err := s.repository.UpdateUserConnectorResourceIDByID(ctx, ownerPermalink, userPermalink, id, newID); err != nil {
		return nil, err
	}

	dbConnectorResource, err := s.repository.GetUserConnectorResourceByID(ctx, ownerPermalink, userPermalink, newID, false)
	if err != nil {
		return nil, err
	}

	return s.convertDatamodelToProto(ctx, dbConnectorResource, connectorPB.View_VIEW_FULL, true)

}

func (s *service) Execute(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, inputs []*structpb.Struct) ([]*structpb.Struct, error) {

	logger, _ := logger.GetZapLogger(ctx)
	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	dbConnectorResource, err := s.repository.GetUserConnectorResourceByID(ctx, ownerPermalink, userPermalink, id, false)
	if err != nil {
		return nil, err
	}

	configuration := func() *structpb.Struct {
		if dbConnectorResource.Configuration != nil {
			str := structpb.Struct{}
			err := str.UnmarshalJSON(dbConnectorResource.Configuration)
			if err != nil {
				logger.Fatal(err.Error())
			}
			return &str
		}
		return nil
	}()

	con, err := s.connectorAll.CreateExecution(dbConnectorResource.ConnectorDefinitionUID, configuration, logger)
	if err != nil {
		return nil, err
	}

	return con.Execute(inputs)
}

func (s *service) CheckConnectorResourceByUID(ctx context.Context, connUID uuid.UUID) (*connectorPB.ConnectorResource_State, error) {

	logger, _ := logger.GetZapLogger(ctx)

	dbConnector, err := s.repository.GetConnectorResourceByUIDAdmin(ctx, connUID, false)
	if err != nil {
		return connectorPB.ConnectorResource_STATE_ERROR.Enum(), nil
	}

	configuration := func() *structpb.Struct {
		if dbConnector.Configuration != nil {
			str := structpb.Struct{}
			err := str.UnmarshalJSON(dbConnector.Configuration)
			if err != nil {
				logger.Fatal(err.Error())
			}
			return &str
		}
		return nil
	}()

	state, err := s.connectorAll.Test(dbConnector.ConnectorDefinitionUID, configuration, logger)
	if err != nil {
		return connectorPB.ConnectorResource_STATE_ERROR.Enum(), nil
	}

	switch state {
	case connectorPB.ConnectorResource_STATE_CONNECTED:
		if err := s.UpdateResourceState(dbConnector.UID, connectorPB.ConnectorResource_STATE_CONNECTED, nil); err != nil {
			return connectorPB.ConnectorResource_STATE_ERROR.Enum(), nil
		}
		return connectorPB.ConnectorResource_STATE_CONNECTED.Enum(), nil
	case connectorPB.ConnectorResource_STATE_ERROR:
		if err := s.UpdateResourceState(dbConnector.UID, connectorPB.ConnectorResource_STATE_ERROR, nil); err != nil {
			return connectorPB.ConnectorResource_STATE_ERROR.Enum(), nil
		}
		return connectorPB.ConnectorResource_STATE_ERROR.Enum(), nil
	default:
		if err := s.UpdateResourceState(dbConnector.UID, connectorPB.ConnectorResource_STATE_ERROR, nil); err != nil {
			return connectorPB.ConnectorResource_STATE_ERROR.Enum(), nil
		}
		return connectorPB.ConnectorResource_STATE_ERROR.Enum(), nil
	}

}
