package service

import (
	"fmt"
	"regexp"

	"github.com/gofrs/uuid"
	"github.com/gogo/status"
	"google.golang.org/grpc/codes"

	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/connector-backend/pkg/repository"

	connectorPB "github.com/instill-ai/protogen-go/connector/v1alpha"
)

// Service interface
type Service interface {
	ListDefinitionByConnectorType(connectorType datamodel.ConnectorType, view connectorPB.DefinitionView, pageSize int, pageCursor string) ([]datamodel.ConnectorDefinition, string, error)
	GetDefinition(ID uuid.UUID, view connectorPB.DefinitionView) (*datamodel.ConnectorDefinition, error)
	CreateConnector(connector *datamodel.Connector) (*datamodel.Connector, error)
	ListConnector(ownerID uuid.UUID, connectorType datamodel.ConnectorType, pageSize int, pageCursor string) ([]datamodel.Connector, string, error)
	GetConnector(ownerID uuid.UUID, name string, connectorType datamodel.ConnectorType) (*datamodel.Connector, error)
	UpdateConnector(ownerID uuid.UUID, name string, connectorType datamodel.ConnectorType, connector *datamodel.Connector) (*datamodel.Connector, error)
	DeleteConnector(ownerID uuid.UUID, name string, connectorType datamodel.ConnectorType) error
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

func (s *service) ListDefinitionByConnectorType(connectorType datamodel.ConnectorType, view connectorPB.DefinitionView, pageSize int, pageCursor string) ([]datamodel.ConnectorDefinition, string, error) {
	return s.repository.ListDefinitionByConnectorType(connectorType, view, pageSize, pageCursor)
}

func (s *service) GetDefinition(ID uuid.UUID, view connectorPB.DefinitionView) (*datamodel.ConnectorDefinition, error) {
	return s.repository.GetDefinition(ID, view)
}

func (s *service) CreateConnector(connector *datamodel.Connector) (*datamodel.Connector, error) {

	def, err := s.GetDefinition(connector.ConnectorDefinitionID, connectorPB.DefinitionView_DEFINITION_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	// Validation: Directness connector uniqueness and assign its name from definition
	if connectorPB.ConnectionType(def.ConnectionType) == connectorPB.ConnectionType_CONNECTION_TYPE_DIRECTNESS {
		if connector.Name != "" {
			return nil, status.Errorf(codes.FailedPrecondition, "Directness connector name is not configurable")
		}

		if connector.Configuration.String() != "{}" {
			return nil, status.Errorf(codes.FailedPrecondition, "Directness connector configuration is not configurable")
		}

		connector.Name = def.Name
		if existingConnector, _ := s.GetConnector(connector.OwnerID, connector.Name, connector.ConnectorType); existingConnector != nil {
			return nil, status.Errorf(codes.AlreadyExists, "Directness connector name \"%s\" with connector_type \"%s\" exists already", connector.Name, connectorPB.ConnectorType(connector.ConnectorType))
		}
	}

	// Validatation: Required field
	if connector.ConnectorDefinitionID.String() == "" {
		return nil, status.Error(codes.FailedPrecondition, "The required field connector_definition_id is not specified")
	}

	// Validatation: Required field
	if connector.Name == "" {
		return nil, status.Error(codes.FailedPrecondition, "The required field name is not specified")
	}

	// Validatation: Required field
	if connector.ConnectorType == datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_UNSPECIFIED) {
		return nil, status.Error(codes.FailedPrecondition, "The required field connector_type is not specified")
	}

	// Validatation: Required field
	if len(connector.Configuration) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "The required field configuration is not specified")
	}

	// Validatation: Naming rule
	if match, _ := regexp.MatchString("^[A-Za-z0-9][a-zA-Z0-9_.-]*$", connector.Name); !match {
		return nil, status.Error(codes.FailedPrecondition, "The name of connector is invalid")
	}

	// Validation: Name length
	if len(connector.Name) > 100 {
		return nil, status.Error(codes.FailedPrecondition, "The name of connector has more than 100 characters")
	}

	// TODO: validate spec JSON Schema

	if err := s.repository.CreateConnector(connector); err != nil {
		return nil, err
	}

	dbConnector, err := s.GetConnector(connector.OwnerID, connector.Name, connector.ConnectorType)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil

}

func (s *service) ListConnector(ownerID uuid.UUID, connectorType datamodel.ConnectorType, pageSize int, pageCursor string) ([]datamodel.Connector, string, error) {
	return s.repository.ListConnector(ownerID, connectorType, pageSize, pageCursor)
}

func (s *service) GetConnector(ownerID uuid.UUID, name string, connectorType datamodel.ConnectorType) (*datamodel.Connector, error) {
	// Validatation: Required field
	if name == "" {
		return nil, status.Error(codes.FailedPrecondition, "The required field name is not specified")
	}

	// Validatation: Required field
	if connectorType == datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_UNSPECIFIED) {
		return nil, status.Error(codes.FailedPrecondition, "The required field connector_type is not specified")
	}

	dbConnector, err := s.repository.GetConnector(ownerID, name, connectorType)
	if err != nil {
		return nil, err
	}

	//TODO: Use owner_id to query owner_name in mgmt-backend
	dbConnector.FullName = fmt.Sprintf("local-user/%s", dbConnector.Name)

	return dbConnector, nil
}

func (s *service) UpdateConnector(ownerID uuid.UUID, name string, connectorType datamodel.ConnectorType, updatedConnector *datamodel.Connector) (*datamodel.Connector, error) {
	// Validatation: Required field
	if name == "" {
		return nil, status.Error(codes.FailedPrecondition, "The required field name not specify")
	}

	// Validatation: Required field
	if connectorType == datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_UNSPECIFIED) {
		return nil, status.Error(codes.FailedPrecondition, "The required field connector_type is not specified")
	}

	if existingConnector, _ := s.GetConnector(ownerID, name, connectorType); existingConnector == nil {
		return nil, status.Errorf(codes.NotFound, "Directness connector name \"%s\" with connector_type \"%s\" is not found", updatedConnector.Name, connectorPB.ConnectorType(updatedConnector.ConnectorType))
	}

	if err := s.repository.UpdateConnector(ownerID, name, connectorType, updatedConnector); err != nil {
		return nil, err
	}

	dbConnector, err := s.GetConnector(updatedConnector.OwnerID, updatedConnector.Name, updatedConnector.ConnectorType)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil

}

func (s *service) DeleteConnector(ownerID uuid.UUID, name string, connectorType datamodel.ConnectorType) error {

	// Validatation: Required field
	if name == "" {
		return status.Error(codes.FailedPrecondition, "The required field name is not specified")
	}

	// Validatation: Required field
	if connectorType == datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_UNSPECIFIED) {
		return status.Error(codes.FailedPrecondition, "The required field connector_type is not specified")
	}

	return s.repository.DeleteConnector(ownerID, name, connectorType)
}
