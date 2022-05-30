package service

import (
	"github.com/gofrs/uuid"
	"go.temporal.io/sdk/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/connector-backend/pkg/repository"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
)

// Service interface
type Service interface {
	// ConnectorDefinition
	ListConnectorDefinition(connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) ([]*datamodel.ConnectorDefinition, int64, string, error)
	GetConnectorDefinitionByID(id string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.ConnectorDefinition, error)
	GetConnectorDefinitionByUID(uid uuid.UUID, isBasicView bool) (*datamodel.ConnectorDefinition, error)

	// Connector
	CreateConnector(connector *datamodel.Connector) (*datamodel.Connector, error)
	ListConnector(ownerRscName string, connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) ([]*datamodel.Connector, int64, string, error)
	GetConnectorByID(id string, ownerRscName string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.Connector, error)
	GetConnectorByUID(uid uuid.UUID, ownerRscName string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.Connector, error)
	UpdateConnector(id string, ownerRscName string, connectorType datamodel.ConnectorType, updatedConnector *datamodel.Connector) (*datamodel.Connector, error)
	UpdateConnectorID(id string, ownerRscName string, connectorType datamodel.ConnectorType, newID string) (*datamodel.Connector, error)
	UpdateConnectorState(id string, ownerRscName string, connectorType datamodel.ConnectorType, state datamodel.ConnectorState) (*datamodel.Connector, error)
	DeleteConnector(id string, ownerRscName string, connectorType datamodel.ConnectorType) error
}

type service struct {
	repository        repository.Repository
	userServiceClient mgmtPB.UserServiceClient
	temporalClient    client.Client
}

// NewService initiates a service instance
func NewService(r repository.Repository, u mgmtPB.UserServiceClient, t client.Client) Service {
	return &service{
		repository:        r,
		userServiceClient: u,
		temporalClient:    t,
	}
}

func (s *service) ListConnectorDefinition(connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) ([]*datamodel.ConnectorDefinition, int64, string, error) {
	return s.repository.ListConnectorDefinition(connectorType, pageSize, pageToken, isBasicView)
}

func (s *service) GetConnectorDefinitionByID(id string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.ConnectorDefinition, error) {
	return s.repository.GetConnectorDefinitionByID(id, connectorType, isBasicView)
}

func (s *service) GetConnectorDefinitionByUID(uid uuid.UUID, isBasicView bool) (*datamodel.ConnectorDefinition, error) {
	return s.repository.GetConnectorDefinitionByUID(uid, isBasicView)
}

func (s *service) CreateConnector(connector *datamodel.Connector) (*datamodel.Connector, error) {

	var ownerPermalink string
	ownerRscName := connector.Owner
	ownerPermalink, err := s.ownerRscNameToPermalink(ownerRscName)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	connector.Owner = ownerPermalink

	connDef, err := s.repository.GetConnectorDefinitionByUID(connector.ConnectorDefinitionUID, true)
	if err != nil {
		return nil, err
	}

	// Validation: Directness connector
	if connectorPB.ConnectionType(connDef.ConnectionType) == connectorPB.ConnectionType_CONNECTION_TYPE_DIRECTNESS {
		if connector.ID != connDef.ID {
			return nil, status.Errorf(codes.InvalidArgument, "[directness] connector_type %s connector id must be %s", connectorPB.ConnectorType(connector.ConnectorType), connDef.ID)
		}

		if connector.Description.String != "" {
			return nil, status.Errorf(codes.InvalidArgument, "[directness] connector_type %s connector description must be empty", connectorPB.ConnectorType(connector.ConnectorType))
		}

		if connector.Configuration.String() != "{}" {
			return nil, status.Errorf(codes.InvalidArgument, "[directness] connector_type %s connector configuration must be an empty JSON {}", connectorPB.ConnectorType(connector.ConnectorType))
		}

		if existingConnector, _ := s.GetConnectorByID(connector.ID, connector.Owner, connector.ConnectorType, true); existingConnector != nil {
			return nil, status.Errorf(codes.AlreadyExists, "[directness] connector_type %s connector id %s exists already", connectorPB.ConnectorType(connector.ConnectorType), connector.ID)
		}
	}

	if err := s.repository.CreateConnector(connector); err != nil {
		return nil, err
	}

	// Check connector state
	if connectorPB.ConnectionType(connDef.ConnectionType) == connectorPB.ConnectionType_CONNECTION_TYPE_DIRECTNESS {
		// Directness connector is always with STATE_CONNECTED
		if err := s.repository.UpdateConnectorState(connector.ID, connector.Owner, connector.ConnectorType, datamodel.ConnectorState(connectorPB.Connector_STATE_CONNECTED)); err != nil {
			return nil, err
		}
	} else {
		def, err := s.repository.GetConnectorDefinitionByUID(connector.ConnectorDefinitionUID, true)
		if err != nil {
			return nil, err
		}
		if err := s.startCheckStateWorkflow(ownerRscName, ownerPermalink, connector.ID, connector.ConnectorType, def.DockerRepository, def.DockerImageTag); err != nil {
			return nil, err
		}
	}

	dbConnector, err := s.repository.GetConnectorByID(connector.ID, ownerPermalink, connector.ConnectorType, false)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil

}

func (s *service) ListConnector(ownerRscName string, connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) ([]*datamodel.Connector, int64, string, error) {

	var ownerPermalink string
	ownerPermalink, err := s.ownerRscNameToPermalink(ownerRscName)
	if err != nil {
		return nil, 0, "", err
	}

	dbConnectors, pageSize, pageToken, err := s.repository.ListConnector(ownerPermalink, connectorType, pageSize, pageToken, isBasicView)
	if err != nil {
		return nil, 0, "", err
	}

	for _, dbConnector := range dbConnectors {
		dbConnector.Owner = ownerRscName
	}

	return dbConnectors, pageSize, pageToken, nil
}

func (s *service) GetConnectorByID(id string, ownerRscName string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.Connector, error) {

	var ownerPermalink string
	ownerPermalink, err := s.ownerRscNameToPermalink(ownerRscName)
	if err != nil {
		return nil, err
	}

	dbConnector, err := s.repository.GetConnectorByID(id, ownerPermalink, connectorType, isBasicView)
	if err != nil {
		return nil, err
	}

	dbConnector.Owner = ownerRscName

	return dbConnector, nil
}

func (s *service) GetConnectorByUID(uid uuid.UUID, ownerRscName string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.Connector, error) {

	var ownerPermalink string
	ownerPermalink, err := s.ownerRscNameToPermalink(ownerRscName)
	if err != nil {
		return nil, err
	}

	dbConnector, err := s.repository.GetConnectorByUID(uid, ownerPermalink, connectorType, isBasicView)
	if err != nil {
		return nil, err
	}

	dbConnector.Owner = ownerRscName

	return dbConnector, nil
}

func (s *service) UpdateConnector(id string, ownerRscName string, connectorType datamodel.ConnectorType, updatedConnector *datamodel.Connector) (*datamodel.Connector, error) {

	var ownerPermalink string
	ownerPermalink, err := s.ownerRscNameToPermalink(ownerRscName)
	if err != nil {
		return nil, err
	}

	updatedConnector.Owner = ownerPermalink

	// Validation: Directness connectors cannot be updated
	existingConnector, err := s.repository.GetConnectorByID(id, ownerPermalink, connectorType, true)
	if err != nil {
		return nil, err
	}

	def, err := s.repository.GetConnectorDefinitionByUID(existingConnector.ConnectorDefinitionUID, true)
	if err != nil {
		return nil, err
	}

	if connectorPB.ConnectionType(def.ConnectionType) == connectorPB.ConnectionType_CONNECTION_TYPE_DIRECTNESS {
		return nil, status.Errorf(codes.InvalidArgument, "Directness connector cannot be updated")
	}

	if err := s.repository.UpdateConnector(id, ownerPermalink, connectorType, updatedConnector); err != nil {
		return nil, err
	}

	// Check connector state
	if err := s.startCheckStateWorkflow(ownerRscName, ownerPermalink, updatedConnector.ID, updatedConnector.ConnectorType, def.DockerRepository, def.DockerImageTag); err != nil {
		return nil, err
	}

	dbConnector, err := s.repository.GetConnectorByID(updatedConnector.ID, ownerPermalink, updatedConnector.ConnectorType, false)
	if err != nil {
		return nil, err
	}

	dbConnector.Owner = ownerRscName

	return dbConnector, nil
}

func (s *service) DeleteConnector(id string, ownerRscName string, connectorType datamodel.ConnectorType) error {
	var ownerPermalink string
	ownerPermalink, err := s.ownerRscNameToPermalink(ownerRscName)
	if err != nil {
		return err
	}
	return s.repository.DeleteConnector(id, ownerPermalink, connectorType)
}

func (s *service) UpdateConnectorState(id string, ownerRscName string, connectorType datamodel.ConnectorType, state datamodel.ConnectorState) (*datamodel.Connector, error) {
	var ownerPermalink string
	ownerPermalink, err := s.ownerRscNameToPermalink(ownerRscName)
	if err != nil {
		return nil, err
	}

	// Validation: Directness connectors cannot be updated
	existingConnector, err := s.repository.GetConnectorByID(id, ownerPermalink, connectorType, true)
	if err != nil {
		return nil, err
	}

	def, err := s.repository.GetConnectorDefinitionByUID(existingConnector.ConnectorDefinitionUID, true)
	if err != nil {
		return nil, err
	}

	if connectorPB.ConnectionType(def.ConnectionType) == connectorPB.ConnectionType_CONNECTION_TYPE_DIRECTNESS {
		return nil, status.Errorf(codes.InvalidArgument, "Directness connector cannot be changed for state due to being always connected")
	}

	if state == datamodel.ConnectorState(connectorPB.Connector_STATE_CONNECTED) && existingConnector.State != datamodel.ConnectorState(connectorPB.Connector_STATE_DISCONNECTED) {
		return nil, status.Errorf(codes.InvalidArgument, "Connector id %s and connector_type %s is not in the disconnected state", id, connectorPB.ConnectorType(connectorType))
	} else if state == datamodel.ConnectorState(connectorPB.Connector_STATE_DISCONNECTED) && existingConnector.State != datamodel.ConnectorState(connectorPB.Connector_STATE_CONNECTED) {
		return nil, status.Errorf(codes.InvalidArgument, "Connector id %s and connector_type %s is not in the connected state", id, connectorPB.ConnectorType(connectorType))
	}

	if state == datamodel.ConnectorState(connectorPB.Connector_STATE_CONNECTED) {
		// Check connector configuration every time when it is set to STATE_CONNECTED from STATE_DISCONNECTED
		if err := s.repository.UpdateConnectorState(id, ownerPermalink, connectorType, datamodel.ConnectorState(connectorPB.Connector_STATE_UNSPECIFIED)); err != nil {
			return nil, err
		}
		if err := s.startCheckStateWorkflow(ownerRscName, ownerPermalink, id, connectorType, def.DockerRepository, def.DockerImageTag); err != nil {
			return nil, err
		}
	} else if state == datamodel.ConnectorState(connectorPB.Connector_STATE_DISCONNECTED) {
		if err := s.repository.UpdateConnectorState(id, ownerPermalink, connectorType, state); err != nil {
			return nil, err
		}
	}

	dbConnector, err := s.repository.GetConnectorByID(id, ownerPermalink, connectorType, false)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil
}

func (s *service) UpdateConnectorID(id string, ownerRscName string, connectorType datamodel.ConnectorType, newID string) (*datamodel.Connector, error) {

	var ownerPermalink string
	ownerPermalink, err := s.ownerRscNameToPermalink(ownerRscName)
	if err != nil {
		return nil, err
	}

	// Validation: Directness connectors cannot be updated
	existingConnector, err := s.repository.GetConnectorByID(id, ownerPermalink, connectorType, true)
	if err != nil {
		return nil, err
	}

	def, err := s.repository.GetConnectorDefinitionByUID(existingConnector.ConnectorDefinitionUID, true)
	if err != nil {
		return nil, err
	}

	if connectorPB.ConnectionType(def.ConnectionType) == connectorPB.ConnectionType_CONNECTION_TYPE_DIRECTNESS {
		return nil, status.Errorf(codes.InvalidArgument, "Directness connector cannot be updated")
	}

	if err := s.repository.UpdateConnectorID(id, ownerPermalink, connectorType, newID); err != nil {
		return nil, err
	}

	dbConnector, err := s.repository.GetConnectorByID(newID, ownerPermalink, connectorType, false)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil
}
