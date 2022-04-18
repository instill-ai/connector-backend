package repository

import (
	"errors"
	"time"

	"google.golang.org/grpc/codes"
	"gorm.io/gorm"

	"github.com/gogo/status"
	"github.com/instill-ai/connector-backend/internal/paginate"
	"github.com/instill-ai/connector-backend/pkg/datamodel"
)

// Repository interface
type Repository interface {
	ListDefinitionByConnectorType(pageSize int, pageCursor string, connectorType string) ([]datamodel.ConnectorDefinition, string, error)
	GetDefinition(ID string) (*datamodel.ConnectorDefinition, error)
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

func (r *repository) ListDefinitionByConnectorType(pageSize int, pageCursor string, connectorType string) ([]datamodel.ConnectorDefinition, string, error) {
	queryBuilder := r.db.Model(&datamodel.ConnectorDefinition{}).Where("connector_type = ?", connectorType).Order("created_at DESC")

	if pageSize > 0 {
		queryBuilder = queryBuilder.Limit(pageSize)
	}

	if pageCursor != "" {
		createdAt, uuid, err := paginate.DecodeCursor(pageCursor)
		if err != nil {
			err = errors.New("invalid page cursor")
			return nil, "", err
		}
		queryBuilder = queryBuilder.Where("created_at <= ? AND id > ?", createdAt, uuid)
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
			return nil, "", err
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

func (r *repository) GetDefinition(ID string) (*datamodel.ConnectorDefinition, error) {
	var connectorDefinition datamodel.ConnectorDefinition
	if result := r.db.Model(&datamodel.ConnectorDefinition{}).
		Where("id = ?", ID).
		First(&connectorDefinition); result.Error != nil {
		return nil, status.Errorf(codes.NotFound, "The connector id %s you specified is not found", ID)
	}
	return &connectorDefinition, nil
}
