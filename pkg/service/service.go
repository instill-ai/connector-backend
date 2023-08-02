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
	CreateConnector(ctx context.Context, owner *mgmtPB.User, connector *datamodel.Connector) (*datamodel.Connector, error)
	ListConnectors(ctx context.Context, owner *mgmtPB.User, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*datamodel.Connector, int64, string, error)
	GetConnectorByID(ctx context.Context, id string, owner *mgmtPB.User, isBasicView bool) (*datamodel.Connector, error)
	GetConnectorByUID(ctx context.Context, uid uuid.UUID, owner *mgmtPB.User, isBasicView bool) (*datamodel.Connector, error)
	UpdateConnector(ctx context.Context, id string, owner *mgmtPB.User, updatedConnector *datamodel.Connector) (*datamodel.Connector, error)
	UpdateConnectorID(ctx context.Context, id string, owner *mgmtPB.User, newID string) (*datamodel.Connector, error)
	UpdateConnectorState(ctx context.Context, id string, ownerPermalink string, state datamodel.ConnectorState) (*datamodel.Connector, error)
	DeleteConnector(ctx context.Context, id string, owner *mgmtPB.User) error

	ListConnectorsAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*datamodel.Connector, int64, string, error)
	GetConnectorByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool) (*datamodel.Connector, error)

	// Execute connector
	Execute(ctx context.Context, conn *datamodel.Connector, owner *mgmtPB.User, inputs []*connectorPB.DataPayload) ([]*connectorPB.DataPayload, error)

	// Shared public/private method for checking connector's connection
	CheckConnectorByUID(ctx context.Context, connUID uuid.UUID) (*connectorPB.Connector_State, error)

	// Controller custom service
	GetResourceState(uid uuid.UUID) (*connectorPB.Connector_State, error)
	UpdateResourceState(uid uuid.UUID, state connectorPB.Connector_State, progress *int32) error
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

func (s *service) CreateConnector(ctx context.Context, owner *mgmtPB.User, connector *datamodel.Connector) (*datamodel.Connector, error) {

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

		if existingConnector, _ := s.GetConnectorByID(ctx, connector.ID, owner, true); existingConnector != nil {
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

	if err := s.repository.CreateConnector(ctx, connector); err != nil {
		return nil, err
	}

	if connDef.GetId() == constant.StartConnectorId || connDef.GetId() == constant.EndConnectorId {
		// User desire state = CONNECTED
		if err := s.repository.UpdateConnectorStateByID(ctx, connector.ID, connector.Owner, datamodel.ConnectorState(connectorPB.Connector_STATE_CONNECTED)); err != nil {
			return nil, err
		}
		if err := s.UpdateResourceState(connector.UID, connectorPB.Connector_STATE_CONNECTED, nil); err != nil {
			return nil, err
		}
	} else {
		// User desire state = DISCONNECTED
		if err := s.repository.UpdateConnectorStateByID(ctx, connector.ID, connector.Owner, datamodel.ConnectorState(connectorPB.Connector_STATE_DISCONNECTED)); err != nil {
			return nil, err
		}
		if err := s.UpdateResourceState(connector.UID, connectorPB.Connector_STATE_DISCONNECTED, nil); err != nil {
			return nil, err
		}
	}

	dbConnector, err := s.repository.GetConnectorByID(ctx, connector.ID, ownerPermalink, false)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil

}

func (s *service) ListConnectors(ctx context.Context, owner *mgmtPB.User, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*datamodel.Connector, int64, string, error) {

	ownerPermalink := GenOwnerPermalink(owner)

	dbConnectors, pageSize, pageToken, err := s.repository.ListConnectors(ctx, ownerPermalink, pageSize, pageToken, isBasicView, filter)
	if err != nil {
		return nil, 0, "", err
	}

	return dbConnectors, pageSize, pageToken, nil
}

func (s *service) ListConnectorsAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*datamodel.Connector, int64, string, error) {

	dbConnectors, pageSize, pageToken, err := s.repository.ListConnectorsAdmin(ctx, pageSize, pageToken, isBasicView, filter)
	if err != nil {
		return nil, 0, "", err
	}

	return dbConnectors, pageSize, pageToken, nil
}

func (s *service) GetConnectorByID(ctx context.Context, id string, owner *mgmtPB.User, isBasicView bool) (*datamodel.Connector, error) {

	ownerPermalink := GenOwnerPermalink(owner)

	dbConnector, err := s.repository.GetConnectorByID(ctx, id, ownerPermalink, isBasicView)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil
}

func (s *service) GetConnectorByUID(ctx context.Context, uid uuid.UUID, owner *mgmtPB.User, isBasicView bool) (*datamodel.Connector, error) {

	ownerPermalink := GenOwnerPermalink(owner)

	dbConnector, err := s.repository.GetConnectorByUID(ctx, uid, ownerPermalink, isBasicView)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil
}

func (s *service) GetConnectorByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool) (*datamodel.Connector, error) {

	dbConnector, err := s.repository.GetConnectorByUIDAdmin(ctx, uid, isBasicView)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil
}

func (s *service) UpdateConnector(ctx context.Context, id string, owner *mgmtPB.User, updatedConnector *datamodel.Connector) (*datamodel.Connector, error) {

	logger, _ := logger.GetZapLogger(ctx)

	ownerPermalink := GenOwnerPermalink(owner)

	updatedConnector.Owner = ownerPermalink

	// Validation: trigger and response connector cannot be updated
	existingConnector, err := s.repository.GetConnectorByID(ctx, id, ownerPermalink, true)
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

	if err := s.repository.UpdateConnector(ctx, id, ownerPermalink, updatedConnector); err != nil {
		return nil, err
	}

	// Check connector state
	if err := s.UpdateResourceState(updatedConnector.UID, connectorPB.Connector_STATE_DISCONNECTED, nil); err != nil {
		return nil, err
	}

	dbConnector, err := s.repository.GetConnectorByID(ctx, updatedConnector.ID, ownerPermalink, false)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil
}

func (s *service) DeleteConnector(ctx context.Context, id string, owner *mgmtPB.User) error {
	logger, _ := logger.GetZapLogger(ctx)

	ownerPermalink := GenOwnerPermalink(owner)

	dbConnector, err := s.repository.GetConnectorByID(ctx, id, ownerPermalink, false)
	if err != nil {
		return err
	}

	filter := fmt.Sprintf("recipe.components.resource_name:\"connectors/%s\"", dbConnector.UID)

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

	return s.repository.DeleteConnector(ctx, id, ownerPermalink)
}

func (s *service) UpdateConnectorState(ctx context.Context, id string, ownerPermalink string, state datamodel.ConnectorState) (*datamodel.Connector, error) {

	logger, _ := logger.GetZapLogger(ctx)

	// Validation: trigger and response connector cannot be disconnected
	conn, err := s.repository.GetConnectorByID(ctx, id, ownerPermalink, false)
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
	case datamodel.ConnectorState(connectorPB.Connector_STATE_CONNECTED):
		if connDef.GetId() == constant.StartConnectorId || connDef.GetId() == constant.EndConnectorId {
			break
		}

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

		taskName, _ := con.GetTask()
		if err != nil {
			return nil, err
		}

		// Set connector state to user desire state
		if err := s.repository.UpdateConnectorStateByID(ctx, id, ownerPermalink, datamodel.ConnectorState(connectorPB.Connector_STATE_CONNECTED)); err != nil {
			return nil, err
		}

		if err := s.repository.UpdateConnectorTaskByID(ctx, id, ownerPermalink, datamodel.Task(taskName)); err != nil {
			return nil, err
		}
		if err := s.UpdateResourceState(conn.UID, connectorPB.Connector_STATE_CONNECTED, nil); err != nil {
			return nil, err
		}

	case datamodel.ConnectorState(connectorPB.Connector_STATE_DISCONNECTED):

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

		if err := s.repository.UpdateConnectorStateByID(ctx, id, ownerPermalink, datamodel.ConnectorState(connectorPB.Connector_STATE_DISCONNECTED)); err != nil {
			return nil, err
		}
		if err := s.UpdateResourceState(conn.UID, connectorPB.Connector_State(state), nil); err != nil {
			return nil, err
		}
	}

	dbConnector, err := s.repository.GetConnectorByID(ctx, id, ownerPermalink, false)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil
}

func (s *service) UpdateConnectorID(ctx context.Context, id string, owner *mgmtPB.User, newID string) (*datamodel.Connector, error) {

	logger, _ := logger.GetZapLogger(ctx)

	ownerPermalink := GenOwnerPermalink(owner)

	// Validation: trigger and response connectors cannot be renamed
	existingConnector, err := s.repository.GetConnectorByID(ctx, id, ownerPermalink, true)
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

	if err := s.repository.UpdateConnectorID(ctx, id, ownerPermalink, newID); err != nil {
		return nil, err
	}

	dbConnector, err := s.repository.GetConnectorByID(ctx, newID, ownerPermalink, false)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil
}

func (s *service) Execute(ctx context.Context, conn *datamodel.Connector, owner *mgmtPB.User, inputs []*connectorPB.DataPayload) ([]*connectorPB.DataPayload, error) {

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

func (s *service) CheckConnectorByUID(ctx context.Context, connUID uuid.UUID) (*connectorPB.Connector_State, error) {

	logger, _ := logger.GetZapLogger(ctx)

	dbConnector, err := s.repository.GetConnectorByUIDAdmin(ctx, connUID, false)
	if err != nil {
		return connectorPB.Connector_STATE_ERROR.Enum(), nil
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
		return connectorPB.Connector_STATE_ERROR.Enum(), nil
	}

	state, err := con.Test()
	if err != nil {
		return connectorPB.Connector_STATE_ERROR.Enum(), nil
	}

	switch state {
	case connectorPB.Connector_STATE_CONNECTED:
		if err := s.UpdateResourceState(dbConnector.UID, connectorPB.Connector_STATE_CONNECTED, nil); err != nil {
			return connectorPB.Connector_STATE_ERROR.Enum(), nil
		}
		return connectorPB.Connector_STATE_CONNECTED.Enum(), nil
	case connectorPB.Connector_STATE_ERROR:
		if err := s.UpdateResourceState(dbConnector.UID, connectorPB.Connector_STATE_ERROR, nil); err != nil {
			return connectorPB.Connector_STATE_ERROR.Enum(), nil
		}
		return connectorPB.Connector_STATE_ERROR.Enum(), nil
	default:
		if err := s.UpdateResourceState(dbConnector.UID, connectorPB.Connector_STATE_ERROR, nil); err != nil {
			return connectorPB.Connector_STATE_ERROR.Enum(), nil
		}
		return connectorPB.Connector_STATE_ERROR.Enum(), nil
	}

}
