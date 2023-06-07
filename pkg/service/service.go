package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/connector-backend/pkg/connector"
	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/connector-backend/pkg/logger"
	"github.com/instill-ai/connector-backend/pkg/repository"
	"github.com/instill-ai/x/sterr"

	connectorDestination "github.com/instill-ai/connector-destination/pkg"
	connectorAirbyte "github.com/instill-ai/connector-destination/pkg/airbyte"
	connectorSource "github.com/instill-ai/connector-source/pkg"
	connectorBase "github.com/instill-ai/connector/pkg/base"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	controllerPB "github.com/instill-ai/protogen-go/vdp/controller/v1alpha"
	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

// Service interface
type Service interface {
	GetMgmtPrivateServiceClient() mgmtPB.MgmtPrivateServiceClient

	// Connector common
	CreateConnector(ctx context.Context, owner *mgmtPB.User, connector *datamodel.Connector) (*datamodel.Connector, error)
	ListConnectors(ctx context.Context, owner *mgmtPB.User, connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) ([]*datamodel.Connector, int64, string, error)
	GetConnectorByID(ctx context.Context, id string, owner *mgmtPB.User, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.Connector, error)
	GetConnectorByUID(ctx context.Context, uid uuid.UUID, owner *mgmtPB.User, isBasicView bool) (*datamodel.Connector, error)
	UpdateConnector(ctx context.Context, id string, owner *mgmtPB.User, connectorType datamodel.ConnectorType, updatedConnector *datamodel.Connector) (*datamodel.Connector, error)
	UpdateConnectorID(ctx context.Context, id string, owner *mgmtPB.User, connectorType datamodel.ConnectorType, newID string) (*datamodel.Connector, error)
	UpdateConnectorState(ctx context.Context, id string, owner *mgmtPB.User, connectorType datamodel.ConnectorType, state datamodel.ConnectorState) (*datamodel.Connector, error)
	DeleteConnector(ctx context.Context, id string, owner *mgmtPB.User, connectorType datamodel.ConnectorType) error

	ListConnectorsAdmin(ctx context.Context, connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) ([]*datamodel.Connector, int64, string, error)
	GetConnectorByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool) (*datamodel.Connector, error)

	// Source connector custom service
	ReadSourceConnector(ctx context.Context, id string, owner *mgmtPB.User) ([]byte, error)

	// Destination connector custom service
	WriteDestinationConnector(ctx context.Context, id string, owner *mgmtPB.User, param connectorAirbyte.WriteDestinationConnectorParam) error

	// Shared public/private method for checking connector's connection
	CheckConnectorByUID(ctx context.Context, connUID uuid.UUID) (*connectorPB.Connector_State, error)

	// Controller custom service
	GetResourceState(uid uuid.UUID, connectorType datamodel.ConnectorType) (*connectorPB.Connector_State, error)
	UpdateResourceState(uid uuid.UUID, connectorType datamodel.ConnectorType, state connectorPB.Connector_State, progress *int32) error
	DeleteResourceState(uid uuid.UUID, connectorType datamodel.ConnectorType) error
}

type service struct {
	repository                  repository.Repository
	mgmtPrivateServiceClient    mgmtPB.MgmtPrivateServiceClient
	pipelinePublicServiceClient pipelinePB.PipelinePublicServiceClient
	controllerClient            controllerPB.ControllerPrivateServiceClient
	connectorAll                connectorBase.IConnector
	connectorSource             connectorBase.IConnector
	connectorDestination        connectorBase.IConnector
}

// NewService initiates a service instance
func NewService(
	t context.Context,
	r repository.Repository,
	u mgmtPB.MgmtPrivateServiceClient,
	p pipelinePB.PipelinePublicServiceClient,
	c controllerPB.ControllerPrivateServiceClient,
) Service {
	logger, _ := logger.GetZapLogger(t)
	return &service{
		repository:                  r,
		mgmtPrivateServiceClient:    u,
		pipelinePublicServiceClient: p,
		controllerClient:            c,
		connectorAll:                connector.InitConnectorAll(logger),
		connectorSource:             connectorSource.Init(logger),
		connectorDestination:        connectorDestination.Init(logger, connector.GetConnectorDestinationOptions()),
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

	// Validation: HTTP and gRPC connector
	if strings.Contains(connDef.GetId(), "http") || strings.Contains(connDef.GetId(), "grpc") {
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

		if existingConnector, _ := s.GetConnectorByID(ctx, connector.ID, owner, connector.ConnectorType, true); existingConnector != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.AlreadyExists,
				"[service] create connector",
				"connectors",
				fmt.Sprintf("Connector id %s and connector_type %s", connector.ID, connectorPB.ConnectorType(connector.ConnectorType)),
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

	// User desire state = CONNECTED
	if err := s.repository.UpdateConnectorStateByID(ctx, connector.ID, connector.Owner, connector.ConnectorType, datamodel.ConnectorState(connectorPB.Connector_STATE_CONNECTED)); err != nil {
		return nil, err
	}

	// Check connector state and update resource state in etcd
	if state, err := s.CheckConnectorByUID(ctx, connector.UID); err == nil {
		if err := s.UpdateResourceState(connector.UID, connector.ConnectorType, *state, nil); err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}

	dbConnector, err := s.repository.GetConnectorByID(ctx, connector.ID, ownerPermalink, connector.ConnectorType, false)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil

}

func (s *service) ListConnectors(ctx context.Context, owner *mgmtPB.User, connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) ([]*datamodel.Connector, int64, string, error) {

	ownerPermalink := GenOwnerPermalink(owner)

	dbConnectors, pageSize, pageToken, err := s.repository.ListConnectors(ctx, ownerPermalink, connectorType, pageSize, pageToken, isBasicView)
	if err != nil {
		return nil, 0, "", err
	}

	return dbConnectors, pageSize, pageToken, nil
}

func (s *service) ListConnectorsAdmin(ctx context.Context, connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) ([]*datamodel.Connector, int64, string, error) {

	dbConnectors, pageSize, pageToken, err := s.repository.ListConnectorsAdmin(ctx, connectorType, pageSize, pageToken, isBasicView)
	if err != nil {
		return nil, 0, "", err
	}

	return dbConnectors, pageSize, pageToken, nil
}

func (s *service) GetConnectorByID(ctx context.Context, id string, owner *mgmtPB.User, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.Connector, error) {

	ownerPermalink := GenOwnerPermalink(owner)

	dbConnector, err := s.repository.GetConnectorByID(ctx, id, ownerPermalink, connectorType, isBasicView)
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

func (s *service) UpdateConnector(ctx context.Context, id string, owner *mgmtPB.User, connectorType datamodel.ConnectorType, updatedConnector *datamodel.Connector) (*datamodel.Connector, error) {

	logger, _ := logger.GetZapLogger(ctx)

	ownerPermalink := GenOwnerPermalink(owner)

	updatedConnector.Owner = ownerPermalink

	// Validation: HTTP and gRPC connector cannot be updated
	existingConnector, err := s.repository.GetConnectorByID(ctx, id, ownerPermalink, connectorType, true)
	if err != nil {
		return nil, err
	}

	def, err := s.connectorAll.GetConnectorDefinitionByUid(existingConnector.ConnectorDefinitionUID)
	if err != nil {
		return nil, err
	}

	if strings.Contains(def.GetId(), "http") || strings.Contains(def.GetId(), "grpc") {
		st, err := sterr.CreateErrorPreconditionFailure(
			"[service] update connector",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "UPDATE",
					Subject:     fmt.Sprintf("id %s and connector_type %s", id, connectorPB.ConnectorType(connectorType)),
					Description: fmt.Sprintf("Cannot update a %s connector", id),
				},
			})
		if err != nil {
			logger.Error(err.Error())
		}
		return nil, st.Err()
	}

	if err := s.repository.UpdateConnector(ctx, id, ownerPermalink, connectorType, updatedConnector); err != nil {
		return nil, err
	}

	// Check connector state
	if state, err := s.CheckConnectorByUID(ctx, existingConnector.UID); err == nil {
		if err := s.UpdateResourceState(updatedConnector.UID, connectorType, *state, nil); err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}

	dbConnector, err := s.repository.GetConnectorByID(ctx, updatedConnector.ID, ownerPermalink, updatedConnector.ConnectorType, false)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil
}

func (s *service) DeleteConnector(ctx context.Context, id string, owner *mgmtPB.User, connectorType datamodel.ConnectorType) error {
	logger, _ := logger.GetZapLogger(ctx)

	ownerPermalink := GenOwnerPermalink(owner)

	dbConnector, err := s.repository.GetConnectorByID(ctx, id, ownerPermalink, connectorType, false)
	if err != nil {
		return err
	}

	var filter string
	switch {
	case connectorType == datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE):
		filter = fmt.Sprintf("recipe.components.resource_name:\"source-connectors/%s\"", dbConnector.UID)
	case connectorType == datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION):
		filter = fmt.Sprintf("recipe.components.resource_name:\"destination-connectors/%s\"", dbConnector.UID)
	}

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
					Subject:     fmt.Sprintf("id %s and connector_type %s", id, connectorPB.ConnectorType(connectorType)),
					Description: fmt.Sprintf("The connector is still in use by pipeline: %s", strings.Join(pipeIDs, " ")),
				},
			})
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	}

	if err := s.DeleteResourceState(dbConnector.UID, connectorType); err != nil {
		return err
	}

	return s.repository.DeleteConnector(ctx, id, ownerPermalink, connectorType)
}

func (s *service) UpdateConnectorState(ctx context.Context, id string, owner *mgmtPB.User, connectorType datamodel.ConnectorType, state datamodel.ConnectorState) (*datamodel.Connector, error) {

	logger, _ := logger.GetZapLogger(ctx)

	ownerPermalink := GenOwnerPermalink(owner)

	// Validation: HTTP and gRPC connector cannot be disconnected
	conn, err := s.repository.GetConnectorByID(ctx, id, ownerPermalink, connectorType, true)
	if err != nil {
		return nil, err
	}

	connDef, err := s.connectorAll.GetConnectorDefinitionByUid(conn.ConnectorDefinitionUID)
	if err != nil {
		return nil, err
	}

	connState, err := s.GetResourceState(conn.UID, connectorType)
	if err != nil {
		return nil, err
	}

	switch *connState {
	case connectorPB.Connector_STATE_ERROR:
		st, err := sterr.CreateErrorPreconditionFailure(
			"[service] update connector state",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "STATE",
					Subject:     fmt.Sprintf("id %s and connector_type %s", id, connectorPB.ConnectorType(connectorType)),
					Description: "The connector is in STATE_ERROR",
				},
			})
		if err != nil {
			logger.Error(err.Error())
		}
		return nil, st.Err()
	case connectorPB.Connector_STATE_UNSPECIFIED:
		st, err := sterr.CreateErrorPreconditionFailure(
			"[service] update connector state",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "STATE",
					Subject:     fmt.Sprintf("id %s and connector_type %s", id, connectorPB.ConnectorType(connectorType)),
					Description: "The connector is in STATE_UNSPECIFIED",
				},
			})
		if err != nil {
			logger.Error(err.Error())
		}
		return nil, st.Err()
	}

	switch state {
	case datamodel.ConnectorState(connectorPB.Connector_STATE_CONNECTED):

		if strings.Contains(connDef.GetId(), "http") || strings.Contains(connDef.GetId(), "grpc") {
			break
		}

		// Set connector state to user desire state
		if err := s.repository.UpdateConnectorStateByID(ctx, id, ownerPermalink, connectorType, datamodel.ConnectorState(connectorPB.Connector_STATE_CONNECTED)); err != nil {
			return nil, err
		}

		// Check resource state
		if datamodel.ConnectorState(*connState) != state {
			if state, err := s.CheckConnectorByUID(ctx, conn.UID); err == nil {
				if err := s.UpdateResourceState(conn.UID, connectorType, *state, nil); err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		}

	case datamodel.ConnectorState(connectorPB.Connector_STATE_DISCONNECTED):

		if strings.Contains(connDef.GetId(), "http") || strings.Contains(connDef.GetId(), "grpc") {
			st, err := sterr.CreateErrorPreconditionFailure(
				"[service] update connector state",
				[]*errdetails.PreconditionFailure_Violation{
					{
						Type:        "STATE",
						Subject:     fmt.Sprintf("id %s and connector_type %s", id, connectorPB.ConnectorType(connectorType)),
						Description: fmt.Sprintf("Cannot disconnect a %s connector", connDef.GetId()),
					},
				})
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, st.Err()
		}

		if err := s.repository.UpdateConnectorStateByID(ctx, id, ownerPermalink, connectorType, state); err != nil {
			return nil, err
		}
		if err := s.UpdateResourceState(conn.UID, connectorType, connectorPB.Connector_State(state), nil); err != nil {
			return nil, err
		}
	}

	dbConnector, err := s.repository.GetConnectorByID(ctx, id, ownerPermalink, connectorType, false)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil
}

func (s *service) UpdateConnectorID(ctx context.Context, id string, owner *mgmtPB.User, connectorType datamodel.ConnectorType, newID string) (*datamodel.Connector, error) {

	logger, _ := logger.GetZapLogger(ctx)

	ownerPermalink := GenOwnerPermalink(owner)

	// Validation: HTTP and gRPC connectors cannot be renamed
	existingConnector, err := s.repository.GetConnectorByID(ctx, id, ownerPermalink, connectorType, true)
	if err != nil {
		return nil, err
	}

	def, err := s.connectorAll.GetConnectorDefinitionByUid(existingConnector.ConnectorDefinitionUID)
	if err != nil {
		return nil, err
	}

	if strings.Contains(def.GetId(), "http") || strings.Contains(def.GetId(), "grpc") {
		st, err := sterr.CreateErrorPreconditionFailure(
			"[service] update connector id",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "RENAME",
					Subject:     fmt.Sprintf("id %s and connector_type %s", id, connectorPB.ConnectorType(connectorType)),
					Description: fmt.Sprintf("Cannot rename a %s connector", def.GetId()),
				},
			})
		if err != nil {
			logger.Error(err.Error())
		}
		return nil, st.Err()
	}

	// if err := s.DeleteResourceState(id, connectorType); err != nil {
	// 	return nil, err
	// }

	// if err := s.UpdateResourceState(newID, connectorType, connectorPB.Connector_State(existingConnector.State), nil, nil); err != nil {
	// 	return nil, err
	// }

	if err := s.repository.UpdateConnectorID(ctx, id, ownerPermalink, connectorType, newID); err != nil {
		return nil, err
	}

	dbConnector, err := s.repository.GetConnectorByID(ctx, newID, ownerPermalink, connectorType, false)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil
}

func (s *service) ReadSourceConnector(ctx context.Context, id string, owner *mgmtPB.User) ([]byte, error) {
	// TODO: Implement async source destination
	return nil, nil
}

func (s *service) WriteDestinationConnector(ctx context.Context, id string, owner *mgmtPB.User, param connectorAirbyte.WriteDestinationConnectorParam) error {

	logger, _ := logger.GetZapLogger(ctx)

	ownerPermalink := GenOwnerPermalink(owner)

	conn, err := s.repository.GetConnectorByID(ctx, id, ownerPermalink, datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION), false)
	if err != nil {
		return err
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

	con, err := s.connectorDestination.CreateConnection(conn.ConnectorDefinitionUID, configuration, logger)
	if err != nil {
		return err
	}
	_, err = con.Execute(param)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) CheckConnectorByUID(ctx context.Context, connUID uuid.UUID) (*connectorPB.Connector_State, error) {

	logger, _ := logger.GetZapLogger(ctx)

	dbConnector, err := s.repository.GetConnectorByUIDAdmin(ctx, connUID, false)
	if err != nil {
		return connectorPB.Connector_STATE_UNSPECIFIED.Enum(), fmt.Errorf(fmt.Sprintf("cannot get the connector, RepositoryError: %v", err))
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
		return connectorPB.Connector_STATE_UNSPECIFIED.Enum(), err
	}

	state, err := con.Test()
	if err != nil {
		return connectorPB.Connector_STATE_UNSPECIFIED.Enum(), err
	}

	switch state {
	case connectorPB.Connector_STATE_CONNECTED:
		if err := s.UpdateResourceState(dbConnector.UID, dbConnector.ConnectorType, connectorPB.Connector_STATE_CONNECTED, nil); err != nil {
			return connectorPB.Connector_STATE_UNSPECIFIED.Enum(), err
		}
		return connectorPB.Connector_STATE_CONNECTED.Enum(), nil
	case connectorPB.Connector_STATE_ERROR:
		if err := s.UpdateResourceState(dbConnector.UID, dbConnector.ConnectorType, connectorPB.Connector_STATE_ERROR, nil); err != nil {
			return connectorPB.Connector_STATE_UNSPECIFIED.Enum(), err
		}
		return connectorPB.Connector_STATE_ERROR.Enum(), nil
	default:
		if err := s.UpdateResourceState(dbConnector.UID, dbConnector.ConnectorType, connectorPB.Connector_STATE_ERROR, nil); err != nil {
			return connectorPB.Connector_STATE_UNSPECIFIED.Enum(), err
		}
		return connectorPB.Connector_STATE_ERROR.Enum(), fmt.Errorf("UNKNOWN STATUS")
	}

}
