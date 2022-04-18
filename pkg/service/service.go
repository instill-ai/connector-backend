package service

import (
	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/connector-backend/pkg/repository"
)

// Service interface
type Service interface {
	ListDefinitionByConnectorType(pageSize int, pageCursor string, connectorType string) ([]datamodel.ConnectorDefinition, string, error)
	GetDefinition(ID string) (*datamodel.ConnectorDefinition, error)
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

func (s *service) ListDefinitionByConnectorType(pageSize int, pageCursor string, connectorType string) ([]datamodel.ConnectorDefinition, string, error) {
	return s.repository.ListDefinitionByConnectorType(pageSize, pageCursor, connectorType)
}

func (s *service) GetDefinition(ID string) (*datamodel.ConnectorDefinition, error) {
	return s.repository.GetDefinition(ID)
}
