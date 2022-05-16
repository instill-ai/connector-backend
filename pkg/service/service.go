package service

import (
	"github.com/gofrs/uuid"
	"github.com/gogo/status"
	"google.golang.org/grpc/codes"

	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/connector-backend/pkg/repository"

	connectorPB "github.com/instill-ai/protogen-go/connector/v1alpha"
)

// Service interface
type Service interface {
	// ConnectorDefinition
	ListConnectorDefinition(connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) ([]*datamodel.ConnectorDefinition, int64, string, error)
	GetConnectorDefinitionByID(id string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.ConnectorDefinition, error)
	GetConnectorDefinitionByUID(uid uuid.UUID, isBasicView bool) (*datamodel.ConnectorDefinition, error)

	// Connector
	CreateConnector(connector *datamodel.Connector) (*datamodel.Connector, error)
	ListConnector(owner string, connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) ([]*datamodel.Connector, int64, string, error)
	GetConnectorByID(id string, owner string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.Connector, error)
	GetConnectorByUID(uid uuid.UUID, owner string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.Connector, error)
	UpdateConnector(id string, owner string, connectorType datamodel.ConnectorType, updatedConnector *datamodel.Connector) (*datamodel.Connector, error)
	DeleteConnector(id string, owner string, connectorType datamodel.ConnectorType) error
	UpdateConnectorID(id string, owner string, connectorType datamodel.ConnectorType, newID string) (*datamodel.Connector, error)
}

type service struct {
	repository repository.Repository
}

// NewService initiates a service instance
func NewService(r repository.Repository) Service {
	return &service{
		repository: r,
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

	// TODO: validate spec JSON Schema

	if err := s.repository.CreateConnector(connector); err != nil {
		return nil, err
	}

	dbConnector, err := s.GetConnectorByID(connector.ID, connector.Owner, connector.ConnectorType, false)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil

}

func (s *service) ListConnector(owner string, connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) ([]*datamodel.Connector, int64, string, error) {
	return s.repository.ListConnector(owner, connectorType, pageSize, pageToken, isBasicView)
}

func (s *service) GetConnectorByID(id string, owner string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.Connector, error) {
	dbConnector, err := s.repository.GetConnectorByID(id, owner, connectorType, isBasicView)
	if err != nil {
		return nil, err
	}
	return dbConnector, nil
}

func (s *service) GetConnectorByUID(uid uuid.UUID, owner string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.Connector, error) {
	dbConnector, err := s.repository.GetConnectorByUID(uid, owner, connectorType, isBasicView)
	if err != nil {
		return nil, err
	}
	return dbConnector, nil
}

func (s *service) UpdateConnector(id string, owner string, connectorType datamodel.ConnectorType, updatedConnector *datamodel.Connector) (*datamodel.Connector, error) {

	// Validatation: Directness connectors cannot be updated
	existingConnector, _ := s.GetConnectorByID(id, owner, connectorType, true)
	if existingConnector == nil {
		return nil, status.Errorf(codes.NotFound, "Connector id \"%s\" with connector_type \"%s\" is not found", updatedConnector.ID, connectorPB.ConnectorType(updatedConnector.ConnectorType))
	}

	def, err := s.GetConnectorDefinitionByUID(existingConnector.ConnectorDefinitionUID, true)
	if err != nil {
		return nil, err
	}

	if connectorPB.ConnectionType(def.ConnectionType) == connectorPB.ConnectionType_CONNECTION_TYPE_DIRECTNESS {
		return nil, status.Errorf(codes.InvalidArgument, "Directness connector cannot be updated")
	}

	if err := s.repository.UpdateConnector(id, owner, connectorType, updatedConnector); err != nil {
		return nil, err
	}

	dbConnector, err := s.GetConnectorByID(updatedConnector.ID, updatedConnector.Owner, updatedConnector.ConnectorType, false)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil
}

func (s *service) DeleteConnector(id string, owner string, connectorType datamodel.ConnectorType) error {
	return s.repository.DeleteConnector(id, owner, connectorType)
}

func (s *service) UpdateConnectorID(id string, owner string, connectorType datamodel.ConnectorType, newID string) (*datamodel.Connector, error) {
	// Validatation: Directness connectors cannot be updated
	existingConnector, _ := s.GetConnectorByID(id, owner, connectorType, true)
	if existingConnector == nil {
		return nil, status.Errorf(codes.NotFound, "Connector id \"%s\" with connector_type \"%s\" is not found", id, connectorPB.ConnectorType(connectorType))
	}

	def, err := s.GetConnectorDefinitionByUID(existingConnector.ConnectorDefinitionUID, true)
	if err != nil {
		return nil, err
	}

	if connectorPB.ConnectionType(def.ConnectionType) == connectorPB.ConnectionType_CONNECTION_TYPE_DIRECTNESS {
		return nil, status.Errorf(codes.InvalidArgument, "Directness connector cannot be updated")
	}

	if err := s.repository.UpdateConnectorID(id, owner, connectorType, newID); err != nil {
		return nil, err
	}

	dbConnector, err := s.GetConnectorByID(newID, owner, connectorType, false)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil
}
