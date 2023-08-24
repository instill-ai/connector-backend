package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/gogo/status"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/redis/go-redis/v9"
	"go.einride.tech/aip/filtering"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/connector-backend/internal/resource"
	"github.com/instill-ai/connector-backend/pkg/connector"
	"github.com/instill-ai/connector-backend/pkg/constant"
	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/connector-backend/pkg/logger"
	"github.com/instill-ai/connector-backend/pkg/repository"
	"github.com/instill-ai/connector-backend/pkg/utils"
	"github.com/instill-ai/x/sterr"

	connectorBase "github.com/instill-ai/connector/pkg/base"
	mgmtPB "github.com/instill-ai/protogen-go/base/mgmt/v1alpha"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	controllerPB "github.com/instill-ai/protogen-go/vdp/controller/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

// Service interface
type Service interface {
	// Connector common
	ListConnectorResources(ctx context.Context, userUid uuid.UUID, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*datamodel.ConnectorResource, int64, string, error)
	CreateUserConnectorResource(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, connector *datamodel.ConnectorResource) (*datamodel.ConnectorResource, error)
	ListUserConnectorResources(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*datamodel.ConnectorResource, int64, string, error)
	GetUserConnectorResourceByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, isBasicView bool) (*datamodel.ConnectorResource, error)
	GetUserConnectorResourceByUID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, uid uuid.UUID, isBasicView bool) (*datamodel.ConnectorResource, error)
	UpdateUserConnectorResourceByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, updatedConnector *datamodel.ConnectorResource) (*datamodel.ConnectorResource, error)
	UpdateUserConnectorResourceIDByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, newID string) (*datamodel.ConnectorResource, error)
	UpdateUserConnectorResourceStateByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, state datamodel.ConnectorResourceState) (*datamodel.ConnectorResource, error)
	DeleteUserConnectorResourceByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string) error

	ListConnectorResourcesAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*datamodel.ConnectorResource, int64, string, error)
	GetConnectorResourceByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool) (*datamodel.ConnectorResource, error)

	// Execute connector
	Execute(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, conn *datamodel.ConnectorResource, inputs []*structpb.Struct) ([]*structpb.Struct, error)

	// Shared public/private method for checking connector's connection
	CheckConnectorResourceByUID(ctx context.Context, connUID uuid.UUID) (*connectorPB.ConnectorResource_State, error)

	// Controller custom service
	GetResourceState(uid uuid.UUID) (*connectorPB.ConnectorResource_State, error)
	UpdateResourceState(uid uuid.UUID, state connectorPB.ConnectorResource_State, progress *int32) error
	DeleteResourceState(uid uuid.UUID) error

	// Influx API
	WriteNewDataPoint(ctx context.Context, data utils.UsageMetricData, pipelineMetadata *structpb.Value) error

	GetRscNamespaceAndNameID(path string) (*resource.Namespace, string, error)
	GetRscNamespaceAndPermalinkUID(path string) (*resource.Namespace, uuid.UUID, error)
	ConvertOwnerPermalinkToName(permalink string) (string, error)
	ConvertOwnerNameToPermalink(name string) (string, error)

	DBToPBConnector(ctx context.Context, dbConnector *datamodel.ConnectorResource, connectorDefinitionName string) (*connectorPB.ConnectorResource, error)
	PBToDBConnector(ctx context.Context, pbConnector *connectorPB.ConnectorResource, connectorDefinition *connectorPB.ConnectorDefinition) (*datamodel.ConnectorResource, error)
	GetUserUid(ctx context.Context) (uuid.UUID, error)
}

type service struct {
	repository                  repository.Repository
	mgmtPrivateServiceClient    mgmtPB.MgmtPrivateServiceClient
	pipelinePublicServiceClient pipelinePB.PipelinePublicServiceClient
	controllerClient            controllerPB.ControllerPrivateServiceClient
	connectorAll                connectorBase.IConnector
	influxDBWriteClient         api.WriteAPI
	redisClient                 *redis.Client
	defaultUserUid              uuid.UUID
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
	defaultUserUid uuid.UUID,
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
		defaultUserUid:              defaultUserUid,
	}
}

// GetUserPermalink returns the api user
func (s *service) GetUserUid(ctx context.Context) (uuid.UUID, error) {
	// Verify if "jwt-sub" is in the header
	headerUserUId := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
	fmt.Println("headerUserUId", headerUserUId)
	if headerUserUId != "" {
		_, err := uuid.FromString(headerUserUId)
		if err != nil {
			return uuid.Nil, status.Errorf(codes.Unauthenticated, "Unauthorized")
		}
		_, err = s.mgmtPrivateServiceClient.LookUpUserAdmin(context.Background(), &mgmtPB.LookUpUserAdminRequest{Permalink: "users/" + headerUserUId})
		if err != nil {
			return uuid.Nil, status.Errorf(codes.Unauthenticated, "Unauthorized")
		}

		return uuid.FromStringOrNil(headerUserUId), nil
	}

	return s.defaultUserUid, nil
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

func (s *service) GetRscNamespaceAndNameID(path string) (*resource.Namespace, string, error) {
	splits := strings.Split(path, "/")
	if len(splits) < 2 {
		return nil, "", fmt.Errorf("namespace error")
	}
	uidStr, err := s.ConvertOwnerNameToPermalink(splits[1])
	if err != nil {
		return nil, "", fmt.Errorf("namespace error")
	}
	if len(splits) < 4 {
		return &resource.Namespace{
			NsType: resource.NamespaceType(splits[0]),
			NsUid:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
		}, "", nil
	}
	return &resource.Namespace{
		NsType: resource.NamespaceType(splits[0]),
		NsUid:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
	}, splits[3], nil
}

func (s *service) GetRscNamespaceAndPermalinkUID(path string) (*resource.Namespace, uuid.UUID, error) {
	splits := strings.Split(path, "/")
	if len(splits) < 2 {
		return nil, uuid.Nil, fmt.Errorf("namespace error")
	}
	uidStr, err := s.ConvertOwnerNameToPermalink(splits[1])
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("namespace error")
	}
	if len(splits) < 4 {
		return &resource.Namespace{
			NsType: resource.NamespaceType(splits[0]),
			NsUid:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
		}, uuid.Nil, nil
	}
	return &resource.Namespace{
		NsType: resource.NamespaceType(splits[0]),
		NsUid:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
	}, uuid.FromStringOrNil(splits[3]), nil
}

func (s *service) ListConnectorResources(ctx context.Context, userUid uuid.UUID, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*datamodel.ConnectorResource, int64, string, error) {

	userPermalink := resource.UserUidToUserPermalink(userUid)
	return s.repository.ListConnectorResources(ctx, userPermalink, pageSize, pageToken, isBasicView, filter)

}

func (s *service) CreateUserConnectorResource(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, connector *datamodel.ConnectorResource) (*datamodel.ConnectorResource, error) {

	logger, _ := logger.GetZapLogger(ctx)

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	connector.Owner = userPermalink

	if existingConnector, _ := s.repository.GetUserConnectorResourceByID(ctx, ownerPermalink, userPermalink, connector.ID, true); existingConnector != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.AlreadyExists,
			"[service] create connector",
			"connectors",
			fmt.Sprintf("Connector id %s", connector.ID),
			connector.Owner,
			"Already exists",
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return nil, st.Err()
	}

	if err := s.repository.CreateUserConnectorResource(ctx, ownerPermalink, userPermalink, connector); err != nil {
		return nil, err
	}

	// User desire state = DISCONNECTED
	if err := s.repository.UpdateUserConnectorResourceStateByID(ctx, ownerPermalink, userPermalink, connector.ID, datamodel.ConnectorResourceState(connectorPB.ConnectorResource_STATE_DISCONNECTED)); err != nil {
		return nil, err
	}
	if err := s.UpdateResourceState(connector.UID, connectorPB.ConnectorResource_STATE_DISCONNECTED, nil); err != nil {
		return nil, err
	}

	dbConnector, err := s.repository.GetUserConnectorResourceByID(ctx, ownerPermalink, userPermalink, connector.ID, false)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil

}

func (s *service) ListUserConnectorResources(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*datamodel.ConnectorResource, int64, string, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	return s.repository.ListUserConnectorResources(ctx, ownerPermalink, userPermalink, pageSize, pageToken, isBasicView, filter)

}

func (s *service) ListConnectorResourcesAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*datamodel.ConnectorResource, int64, string, error) {

	return s.repository.ListConnectorResourcesAdmin(ctx, pageSize, pageToken, isBasicView, filter)

}

func (s *service) GetUserConnectorResourceByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, isBasicView bool) (*datamodel.ConnectorResource, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	return s.repository.GetUserConnectorResourceByID(ctx, ownerPermalink, userPermalink, id, isBasicView)

}

func (s *service) GetUserConnectorResourceByUID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, uid uuid.UUID, isBasicView bool) (*datamodel.ConnectorResource, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	return s.repository.GetUserConnectorResourceByUID(ctx, ownerPermalink, userPermalink, uid, isBasicView)

}

func (s *service) GetConnectorResourceByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool) (*datamodel.ConnectorResource, error) {

	return s.repository.GetConnectorResourceByUIDAdmin(ctx, uid, isBasicView)

}

func (s *service) UpdateUserConnectorResourceByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, updatedConnector *datamodel.ConnectorResource) (*datamodel.ConnectorResource, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	updatedConnector.Owner = ownerPermalink

	if err := s.repository.UpdateUserConnectorResourceByID(ctx, ownerPermalink, userPermalink, id, updatedConnector); err != nil {
		return nil, err
	}

	// Check connector state
	if err := s.UpdateResourceState(updatedConnector.UID, connectorPB.ConnectorResource_STATE_DISCONNECTED, nil); err != nil {
		return nil, err
	}

	return s.repository.GetUserConnectorResourceByID(ctx, ownerPermalink, userPermalink, updatedConnector.ID, false)

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

func (s *service) UpdateUserConnectorResourceStateByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, state datamodel.ConnectorResourceState) (*datamodel.ConnectorResource, error) {

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
	case datamodel.ConnectorResourceState(connectorPB.ConnectorResource_STATE_CONNECTED):

		// Set connector state to user desire state
		if err := s.repository.UpdateUserConnectorResourceStateByID(ctx, ownerPermalink, userPermalink, id, datamodel.ConnectorResourceState(connectorPB.ConnectorResource_STATE_CONNECTED)); err != nil {
			return nil, err
		}

		if err := s.UpdateResourceState(conn.UID, connectorPB.ConnectorResource_STATE_CONNECTED, nil); err != nil {
			return nil, err
		}

	case datamodel.ConnectorResourceState(connectorPB.ConnectorResource_STATE_DISCONNECTED):

		if err := s.repository.UpdateUserConnectorResourceStateByID(ctx, ownerPermalink, userPermalink, id, datamodel.ConnectorResourceState(connectorPB.ConnectorResource_STATE_DISCONNECTED)); err != nil {
			return nil, err
		}
		if err := s.UpdateResourceState(conn.UID, connectorPB.ConnectorResource_State(state), nil); err != nil {
			return nil, err
		}
	}

	dbConnector, err := s.repository.GetUserConnectorResourceByID(ctx, ownerPermalink, userPermalink, id, false)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil
}

func (s *service) UpdateUserConnectorResourceIDByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, newID string) (*datamodel.ConnectorResource, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	if err := s.repository.UpdateUserConnectorResourceIDByID(ctx, ownerPermalink, userPermalink, id, newID); err != nil {
		return nil, err
	}

	return s.repository.GetUserConnectorResourceByID(ctx, ownerPermalink, userPermalink, newID, false)

}

func (s *service) Execute(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, conn *datamodel.ConnectorResource, inputs []*structpb.Struct) ([]*structpb.Struct, error) {

	logger, _ := logger.GetZapLogger(ctx)

	configuration := func() *structpb.Struct {
		if conn.Configuration != nil {
			str := structpb.Struct{}
			err := str.UnmarshalJSON(conn.Configuration)
			if err != nil {
				logger.Fatal(err.Error())
			}
			return &str
		}
		return nil
	}()

	con, err := s.connectorAll.CreateConnection(conn.ConnectorDefinitionUID, configuration, logger)
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

	con, err := s.connectorAll.CreateConnection(dbConnector.ConnectorDefinitionUID, configuration, logger)

	if err != nil {
		return connectorPB.ConnectorResource_STATE_ERROR.Enum(), nil
	}

	state, err := con.Test()
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
