package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/redis/go-redis/v9"
	"go.einride.tech/aip/filtering"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"

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
	GetMgmtPrivateServiceClient() mgmtPB.MgmtPrivateServiceClient

	// Connector common
	CreateConnectorResource(ctx context.Context, owner *mgmtPB.User, connector *datamodel.ConnectorResource) (*datamodel.ConnectorResource, error)
	ListConnectorResources(ctx context.Context, owner *mgmtPB.User, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*datamodel.ConnectorResource, int64, string, error)
	GetConnectorResourceByID(ctx context.Context, id string, owner *mgmtPB.User, isBasicView bool) (*datamodel.ConnectorResource, error)
	GetConnectorResourceByUID(ctx context.Context, uid uuid.UUID, owner *mgmtPB.User, isBasicView bool) (*datamodel.ConnectorResource, error)
	UpdateConnectorResource(ctx context.Context, id string, owner *mgmtPB.User, updatedConnector *datamodel.ConnectorResource) (*datamodel.ConnectorResource, error)
	UpdateConnectorResourceID(ctx context.Context, id string, owner *mgmtPB.User, newID string) (*datamodel.ConnectorResource, error)
	UpdateConnectorResourceState(ctx context.Context, id string, ownerPermalink string, state datamodel.ConnectorResourceState) (*datamodel.ConnectorResource, error)
	DeleteConnectorResource(ctx context.Context, id string, owner *mgmtPB.User) error

	ListConnectorResourcesAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*datamodel.ConnectorResource, int64, string, error)
	GetConnectorResourceByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool) (*datamodel.ConnectorResource, error)

	// Execute connector
	Execute(ctx context.Context, conn *datamodel.ConnectorResource, owner *mgmtPB.User, inputs []*structpb.Struct) ([]*structpb.Struct, error)

	// Shared public/private method for checking connector's connection
	CheckConnectorResourceByUID(ctx context.Context, connUID uuid.UUID) (*connectorPB.ConnectorResource_State, error)

	// Controller custom service
	GetResourceState(uid uuid.UUID) (*connectorPB.ConnectorResource_State, error)
	UpdateResourceState(uid uuid.UUID, state connectorPB.ConnectorResource_State, progress *int32) error
	DeleteResourceState(uid uuid.UUID) error

	// Influx API
	WriteNewDataPoint(ctx context.Context, data utils.UsageMetricData, pipelineMetadata *structpb.Value) error
}

type service struct {
	repository                  repository.Repository
	mgmtPrivateServiceClient    mgmtPB.MgmtPrivateServiceClient
	pipelinePublicServiceClient pipelinePB.PipelinePublicServiceClient
	controllerClient            controllerPB.ControllerPrivateServiceClient
	connectorAll                connectorBase.IConnector
	influxDBWriteClient         api.WriteAPI
	redisClient                 *redis.Client
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
	}
}

// GetMgmtPrivateServiceClient returns the management private service client
func (s *service) GetMgmtPrivateServiceClient() mgmtPB.MgmtPrivateServiceClient {
	return s.mgmtPrivateServiceClient
}

func (s *service) CreateConnectorResource(ctx context.Context, owner *mgmtPB.User, connector *datamodel.ConnectorResource) (*datamodel.ConnectorResource, error) {

	logger, _ := logger.GetZapLogger(ctx)

	ownerPermalink := GenOwnerPermalink(owner)

	connector.Owner = ownerPermalink

	connDef, err := s.connectorAll.GetConnectorDefinitionByUid(connector.ConnectorDefinitionUID)
	if err != nil {
		return nil, err
	}

	// Validation: trigger and responsee connector
	if connDef.GetId() == constant.StartConnectorId || connDef.GetId() == constant.EndConnectorId {
		if connector.ID != connDef.GetId() {
			st, err := sterr.CreateErrorBadRequest(
				"[service] create connector",
				[]*errdetails.BadRequest_FieldViolation{
					{
						Field:       "id",
						Description: fmt.Sprintf("Connector id must be %s", connDef.GetId()),
					},
				},
			)
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, st.Err()
		}

		if connector.Configuration.String() != "{}" {
			st, err := sterr.CreateErrorBadRequest(
				"[service] create connector",
				[]*errdetails.BadRequest_FieldViolation{
					{
						Field:       "connector.configuration",
						Description: fmt.Sprintf("%s connector configuration must be an empty JSON", connDef.GetId()),
					},
				},
			)
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, st.Err()
		}

		if existingConnector, _ := s.GetConnectorResourceByID(ctx, connector.ID, owner, true); existingConnector != nil {
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
	}

	if err := s.repository.CreateConnectorResource(ctx, connector); err != nil {
		return nil, err
	}

	if connDef.GetId() == constant.StartConnectorId || connDef.GetId() == constant.EndConnectorId {
		// User desire state = CONNECTED
		if err := s.repository.UpdateConnectorResourceStateByID(ctx, connector.ID, connector.Owner, datamodel.ConnectorResourceState(connectorPB.ConnectorResource_STATE_CONNECTED)); err != nil {
			return nil, err
		}
		if err := s.UpdateResourceState(connector.UID, connectorPB.ConnectorResource_STATE_CONNECTED, nil); err != nil {
			return nil, err
		}
	} else {
		// User desire state = DISCONNECTED
		if err := s.repository.UpdateConnectorResourceStateByID(ctx, connector.ID, connector.Owner, datamodel.ConnectorResourceState(connectorPB.ConnectorResource_STATE_DISCONNECTED)); err != nil {
			return nil, err
		}
		if err := s.UpdateResourceState(connector.UID, connectorPB.ConnectorResource_STATE_DISCONNECTED, nil); err != nil {
			return nil, err
		}
	}

	dbConnector, err := s.repository.GetConnectorResourceByID(ctx, connector.ID, ownerPermalink, false)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil

}

func (s *service) ListConnectorResources(ctx context.Context, owner *mgmtPB.User, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*datamodel.ConnectorResource, int64, string, error) {

	ownerPermalink := GenOwnerPermalink(owner)

	dbConnectors, pageSize, pageToken, err := s.repository.ListConnectorResources(ctx, ownerPermalink, pageSize, pageToken, isBasicView, filter)
	if err != nil {
		return nil, 0, "", err
	}

	return dbConnectors, pageSize, pageToken, nil
}

func (s *service) ListConnectorResourcesAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*datamodel.ConnectorResource, int64, string, error) {

	dbConnectors, pageSize, pageToken, err := s.repository.ListConnectorResourcesAdmin(ctx, pageSize, pageToken, isBasicView, filter)
	if err != nil {
		return nil, 0, "", err
	}

	return dbConnectors, pageSize, pageToken, nil
}

func (s *service) GetConnectorResourceByID(ctx context.Context, id string, owner *mgmtPB.User, isBasicView bool) (*datamodel.ConnectorResource, error) {

	ownerPermalink := GenOwnerPermalink(owner)

	dbConnector, err := s.repository.GetConnectorResourceByID(ctx, id, ownerPermalink, isBasicView)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil
}

func (s *service) GetConnectorResourceByUID(ctx context.Context, uid uuid.UUID, owner *mgmtPB.User, isBasicView bool) (*datamodel.ConnectorResource, error) {

	ownerPermalink := GenOwnerPermalink(owner)

	dbConnector, err := s.repository.GetConnectorResourceByUID(ctx, uid, ownerPermalink, isBasicView)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil
}

func (s *service) GetConnectorResourceByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool) (*datamodel.ConnectorResource, error) {

	dbConnector, err := s.repository.GetConnectorResourceByUIDAdmin(ctx, uid, isBasicView)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil
}

func (s *service) UpdateConnectorResource(ctx context.Context, id string, owner *mgmtPB.User, updatedConnector *datamodel.ConnectorResource) (*datamodel.ConnectorResource, error) {

	logger, _ := logger.GetZapLogger(ctx)

	ownerPermalink := GenOwnerPermalink(owner)

	updatedConnector.Owner = ownerPermalink

	// Validation: trigger and response connector cannot be updated
	existingConnector, err := s.repository.GetConnectorResourceByID(ctx, id, ownerPermalink, true)
	if err != nil {
		return nil, err
	}

	def, err := s.connectorAll.GetConnectorDefinitionByUid(existingConnector.ConnectorDefinitionUID)
	if err != nil {
		return nil, err
	}

	if def.GetId() == constant.StartConnectorId || def.GetId() == constant.EndConnectorId {
		st, err := sterr.CreateErrorPreconditionFailure(
			"[service] update connector",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "UPDATE",
					Subject:     fmt.Sprintf("id %s", id),
					Description: fmt.Sprintf("Cannot update a %s connector", id),
				},
			})
		if err != nil {
			logger.Error(err.Error())
		}
		return nil, st.Err()
	}

	if err := s.repository.UpdateConnectorResource(ctx, id, ownerPermalink, updatedConnector); err != nil {
		return nil, err
	}

	// Check connector state
	if err := s.UpdateResourceState(updatedConnector.UID, connectorPB.ConnectorResource_STATE_DISCONNECTED, nil); err != nil {
		return nil, err
	}

	dbConnector, err := s.repository.GetConnectorResourceByID(ctx, updatedConnector.ID, ownerPermalink, false)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil
}

func (s *service) DeleteConnectorResource(ctx context.Context, id string, owner *mgmtPB.User) error {
	logger, _ := logger.GetZapLogger(ctx)

	ownerPermalink := GenOwnerPermalink(owner)

	dbConnector, err := s.repository.GetConnectorResourceByID(ctx, id, ownerPermalink, false)
	if err != nil {
		return err
	}

	filter := fmt.Sprintf("recipe.components.resource_name:\"connector-resources/%s\"", dbConnector.UID)

	pipeResp, err := s.pipelinePublicServiceClient.ListPipelines(InjectOwnerToContext(context.Background(), owner), &pipelinePB.ListPipelinesRequest{
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

	return s.repository.DeleteConnectorResource(ctx, id, ownerPermalink)
}

func (s *service) UpdateConnectorResourceState(ctx context.Context, id string, ownerPermalink string, state datamodel.ConnectorResourceState) (*datamodel.ConnectorResource, error) {

	logger, _ := logger.GetZapLogger(ctx)

	// Validation: trigger and response connector cannot be disconnected
	conn, err := s.repository.GetConnectorResourceByID(ctx, id, ownerPermalink, false)
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

	connDef, err := s.connectorAll.GetConnectorDefinitionByUid(conn.ConnectorDefinitionUID)
	if err != nil {
		return nil, err
	}

	switch state {
	case datamodel.ConnectorResourceState(connectorPB.ConnectorResource_STATE_CONNECTED):
		if connDef.GetId() == constant.StartConnectorId || connDef.GetId() == constant.EndConnectorId {
			break
		}

		// Set connector state to user desire state
		if err := s.repository.UpdateConnectorResourceStateByID(ctx, id, ownerPermalink, datamodel.ConnectorResourceState(connectorPB.ConnectorResource_STATE_CONNECTED)); err != nil {
			return nil, err
		}

		if err := s.UpdateResourceState(conn.UID, connectorPB.ConnectorResource_STATE_CONNECTED, nil); err != nil {
			return nil, err
		}

	case datamodel.ConnectorResourceState(connectorPB.ConnectorResource_STATE_DISCONNECTED):

		if connDef.GetId() == constant.StartConnectorId || connDef.GetId() == constant.EndConnectorId {
			st, err := sterr.CreateErrorPreconditionFailure(
				"[service] update connector state",
				[]*errdetails.PreconditionFailure_Violation{
					{
						Type:        "STATE",
						Subject:     fmt.Sprintf("id %s", id),
						Description: fmt.Sprintf("Cannot disconnect a %s connector", connDef.GetId()),
					},
				})
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, st.Err()
		}

		if err := s.repository.UpdateConnectorResourceStateByID(ctx, id, ownerPermalink, datamodel.ConnectorResourceState(connectorPB.ConnectorResource_STATE_DISCONNECTED)); err != nil {
			return nil, err
		}
		if err := s.UpdateResourceState(conn.UID, connectorPB.ConnectorResource_State(state), nil); err != nil {
			return nil, err
		}
	}

	dbConnector, err := s.repository.GetConnectorResourceByID(ctx, id, ownerPermalink, false)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil
}

func (s *service) UpdateConnectorResourceID(ctx context.Context, id string, owner *mgmtPB.User, newID string) (*datamodel.ConnectorResource, error) {

	logger, _ := logger.GetZapLogger(ctx)

	ownerPermalink := GenOwnerPermalink(owner)

	// Validation: trigger and response connectors cannot be renamed
	existingConnector, err := s.repository.GetConnectorResourceByID(ctx, id, ownerPermalink, true)
	if err != nil {
		return nil, err
	}

	def, err := s.connectorAll.GetConnectorDefinitionByUid(existingConnector.ConnectorDefinitionUID)
	if err != nil {
		return nil, err
	}

	if def.GetId() == constant.StartConnectorId || def.GetId() == constant.EndConnectorId {
		st, err := sterr.CreateErrorPreconditionFailure(
			"[service] update connector id",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "RENAME",
					Subject:     fmt.Sprintf("id %s ", id),
					Description: fmt.Sprintf("Cannot rename a %s connector", def.GetId()),
				},
			})
		if err != nil {
			logger.Error(err.Error())
		}
		return nil, st.Err()
	}

	if err := s.repository.UpdateConnectorResourceID(ctx, id, ownerPermalink, newID); err != nil {
		return nil, err
	}

	dbConnector, err := s.repository.GetConnectorResourceByID(ctx, newID, ownerPermalink, false)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil
}

func (s *service) Execute(ctx context.Context, conn *datamodel.ConnectorResource, owner *mgmtPB.User, inputs []*structpb.Struct) ([]*structpb.Struct, error) {

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
