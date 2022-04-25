package repository

import (
	"time"

	"google.golang.org/grpc/codes"
	"gorm.io/gorm"

	"github.com/gofrs/uuid"
	"github.com/gogo/status"
	"github.com/instill-ai/connector-backend/internal/paginate"
	"github.com/instill-ai/connector-backend/pkg/datamodel"

	connectorPB "github.com/instill-ai/protogen-go/connector/v1alpha"
)

// Repository interface
type Repository interface {
	ListDefinitionByConnectorType(connectorType datamodel.ConnectorType, view connectorPB.DefinitionView, pageSize int, pageCursor string) ([]datamodel.ConnectorDefinition, string, error)
	GetDefinition(ID uuid.UUID, view connectorPB.DefinitionView) (*datamodel.ConnectorDefinition, error)
	CreateConnector(connector *datamodel.Connector) error
	ListConnector(ownerID uuid.UUID, connectorType datamodel.ConnectorType, pageSize int, pageCursor string) ([]datamodel.Connector, string, error)
	GetConnector(ownerID uuid.UUID, name string, connectorType datamodel.ConnectorType) (*datamodel.Connector, error)
	UpdateConnector(ownerID uuid.UUID, name string, connectorType datamodel.ConnectorType, connector *datamodel.Connector) error
	DeleteConnector(ownerID uuid.UUID, name string, connectorType datamodel.ConnectorType) error
}

type repository struct {
	db *gorm.DB
}

// NewRepository initiates a repository instance
func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}

func (r *repository) ListDefinitionByConnectorType(connectorType datamodel.ConnectorType, view connectorPB.DefinitionView, pageSize int, pageCursor string) ([]datamodel.ConnectorDefinition, string, error) {
	queryBuilder := r.db.Model(&datamodel.ConnectorDefinition{}).Order("created_at DESC, id DESC")

	if connectorType != datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_UNSPECIFIED) {
		queryBuilder = queryBuilder.Where("connector_type = ?", connectorType)
	}

	if pageSize > 0 {
		queryBuilder = queryBuilder.Limit(pageSize)
	}

	if pageCursor != "" {
		createdAt, uuid, err := paginate.DecodeCursor(pageCursor)
		if err != nil {
			return nil, "", status.Errorf(codes.InvalidArgument, "Invalid page cursor: %s", err.Error())
		}
		queryBuilder = queryBuilder.Where("(created_at,id) < (?::timestamp, ?)", createdAt, uuid)
	}

	if view != connectorPB.DefinitionView_DEFINITION_VIEW_FULL {
		queryBuilder.Omit("connector.spec")
	}

	var connectorDefinitions []datamodel.ConnectorDefinition
	var createdAt time.Time // only using one for all loops, we only need the latest one in the end
	rows, err := queryBuilder.Rows()
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	for rows.Next() {
		var item datamodel.ConnectorDefinition
		if err = r.db.ScanRows(rows, &item); err != nil {
			return nil, "", status.Errorf(codes.Internal, "Error %v", err.Error())
		}
		createdAt = item.CreatedAt
		connectorDefinitions = append(connectorDefinitions, item)
	}

	if len(connectorDefinitions) > 0 {
		nextPageCursor := paginate.EncodeCursor(createdAt, (connectorDefinitions)[len(connectorDefinitions)-1].ID.String())
		return connectorDefinitions, nextPageCursor, nil
	}

	return nil, "", nil
}

func (r *repository) GetDefinition(ID uuid.UUID, view connectorPB.DefinitionView) (*datamodel.ConnectorDefinition, error) {
	var connectorDefinition datamodel.ConnectorDefinition
	queryBuilder := r.db.Model(&datamodel.ConnectorDefinition{}).Where("id = ?", ID.String())
	if view != connectorPB.DefinitionView_DEFINITION_VIEW_FULL {
		queryBuilder.Omit("connector.spec")
	}
	if result := queryBuilder.First(&connectorDefinition); result.Error != nil {
		return nil, status.Errorf(codes.NotFound, "The connector definition id %s you specified is not found", ID)
	}
	return &connectorDefinition, nil
}

func (r *repository) CreateConnector(connector *datamodel.Connector) error {
	if result := r.db.Model(&datamodel.Connector{}).Create(connector); result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}
	return nil
}

func (r *repository) ListConnector(ownerID uuid.UUID, connectorType datamodel.ConnectorType, pageSize int, pageCursor string) ([]datamodel.Connector, string, error) {
	queryBuilder := r.db.Model(&datamodel.Connector{}).Order("created_at DESC, id DESC")

	if connectorType != datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_UNSPECIFIED) {
		queryBuilder = queryBuilder.Where("owner_id = ? AND connector_type = ?", ownerID, connectorType)
	} else {
		queryBuilder = queryBuilder.Where("owner_id = ?", ownerID)
	}

	if pageSize > 0 {
		queryBuilder = queryBuilder.Limit(pageSize)
	}

	if pageCursor != "" {
		createdAt, uuid, err := paginate.DecodeCursor(pageCursor)
		if err != nil {
			return nil, "", status.Errorf(codes.InvalidArgument, "Invalid page cursor: %s", err.Error())

		}
		queryBuilder = queryBuilder.Where("(created_at,id) < (?::timestamp, ?)", createdAt, uuid)
	}

	var connectors []datamodel.Connector
	var createdAt time.Time // only using one for all loops, we only need the latest one in the end
	rows, err := queryBuilder.Rows()
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	for rows.Next() {
		var item datamodel.Connector
		if err = r.db.ScanRows(rows, &item); err != nil {
			return nil, "", status.Errorf(codes.Internal, "Error %v", err.Error())
		}
		createdAt = item.CreatedAt
		connectors = append(connectors, item)
	}

	if len(connectors) > 0 {
		nextPageCursor := paginate.EncodeCursor(createdAt, (connectors)[len(connectors)-1].ID.String())
		return connectors, nextPageCursor, nil
	}

	return nil, "", nil
}

func (r *repository) GetConnector(ownerID uuid.UUID, name string, connectorType datamodel.ConnectorType) (*datamodel.Connector, error) {
	var connector datamodel.Connector
	if result := r.db.Model(&datamodel.Connector{}).
		Where("owner_id = ? AND name = ? AND connector_type = ?", ownerID, name, connectorType).
		First(&connector); result.Error != nil {
		return nil, status.Errorf(codes.NotFound, "The connector with connector_type \"%s\" and name \"%s\" you specified is not found", connectorPB.ConnectorType(connectorType), name)
	}
	return &connector, nil
}

func (r *repository) UpdateConnector(ownerID uuid.UUID, name string, connectorType datamodel.ConnectorType, connector *datamodel.Connector) error {
	if result := r.db.Model(&datamodel.Connector{}).
		Where("owner_id = ? AND name = ? AND connector_type = ?", ownerID, name, connectorType).
		Updates(connector); result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}
	return nil
}

func (r *repository) DeleteConnector(ownerID uuid.UUID, name string, connectorType datamodel.ConnectorType) error {

	result := r.db.Model(&datamodel.Connector{}).
		Where("owner_id = ? AND name = ? AND connector_type = ?", ownerID, name, connectorType).
		Delete(&datamodel.Connector{})

	if result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	if result.RowsAffected == 0 {
		return status.Errorf(codes.NotFound, "The connector with connector_type \"%v\" and name \"%s\" you specified is not found", connectorPB.ConnectorType(connectorType), name)
	}

	return nil
}
