package repository

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgconn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/x/paginate"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

// DefaultPageSize is the default pagination page size when page size is not assigned
const DefaultPageSize = 10

// MaxPageSize is the maximum pagination page size if the assigned value is over this number
const MaxPageSize = 100

// Repository interface
type Repository interface {
	// ConnectorDefinition
	ListConnectorDefinition(connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) ([]*datamodel.ConnectorDefinition, int64, string, error)
	GetConnectorDefinitionByID(id string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.ConnectorDefinition, error)
	GetConnectorDefinitionByUID(uid uuid.UUID, isBasicView bool) (*datamodel.ConnectorDefinition, error)

	// Connector
	CreateConnector(connector *datamodel.Connector) error
	ListConnector(ownerPermalink string, connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) ([]*datamodel.Connector, int64, string, error)
	GetConnectorByID(id string, ownerPermalink string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.Connector, error)
	GetConnectorByUID(uid uuid.UUID, ownerPermalink string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.Connector, error)
	UpdateConnector(id string, ownerPermalink string, connectorType datamodel.ConnectorType, connector *datamodel.Connector) error
	DeleteConnector(id string, ownerPermalink string, connectorType datamodel.ConnectorType) error
	UpdateConnectorID(id string, ownerPermalink string, connectorType datamodel.ConnectorType, newID string) error
	UpdateConnectorStateByID(id string, ownerPermalink string, connectorType datamodel.ConnectorType, state datamodel.ConnectorState) error
	UpdateConnectorStateByUID(uid uuid.UUID, ownerPermalink string, connectorType datamodel.ConnectorType, state datamodel.ConnectorState) error
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

func (r *repository) ListConnectorDefinition(connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) (connectorDefinitions []*datamodel.ConnectorDefinition, totalSize int64, nextPageToken string, err error) {

	r.db.Model(&datamodel.ConnectorDefinition{}).Where("connector_type = ?", connectorType).Count(&totalSize)

	queryBuilder := r.db.Model(&datamodel.ConnectorDefinition{}).Order("create_time DESC, uid DESC").Where("connector_type = ?", connectorType)

	if pageSize == 0 {
		pageSize = DefaultPageSize
	} else if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	queryBuilder = queryBuilder.Limit(int(pageSize))

	if pageToken != "" {
		createdAt, uid, err := paginate.DecodeToken(pageToken)
		if err != nil {
			return nil, 0, "", status.Errorf(codes.InvalidArgument, "Invalid page token: %s", err.Error())
		}
		queryBuilder = queryBuilder.Where("(create_time,uid) < (?::timestamp, ?)", createdAt, uid)
	}

	if isBasicView {
		queryBuilder.Omit("spec")
	}

	// only using one for all loops, we only need the latest one in the end
	var createTime time.Time
	rows, err := queryBuilder.Rows()
	if err != nil {
		return nil, 0, "", status.Errorf(codes.Internal, err.Error())
	}
	defer rows.Close()
	for rows.Next() {
		var item datamodel.ConnectorDefinition
		if err = r.db.ScanRows(rows, &item); err != nil {
			return nil, 0, "", status.Errorf(codes.Internal, err.Error())
		}
		createTime = item.CreateTime
		connectorDefinitions = append(connectorDefinitions, &item)
	}

	if len(connectorDefinitions) < int(pageSize) {
		return connectorDefinitions, totalSize, "", nil
	}
	if len(connectorDefinitions) > 0 {
		lastUID := (connectorDefinitions)[len(connectorDefinitions)-1].UID
		lastItem := &datamodel.ConnectorDefinition{}
		if result := r.db.Model(&datamodel.ConnectorDefinition{}).
			Where("connector_type = ?", connectorType).
			Order("create_time ASC, uid ASC").Limit(1).Find(lastItem); result.Error != nil {
			return nil, 0, "", status.Errorf(codes.Internal, result.Error.Error())
		}
		if lastItem.UID.String() == lastUID.String() {
			nextPageToken = ""
		} else {
			nextPageToken = paginate.EncodeToken(createTime, lastUID.String())
		}
	}

	return connectorDefinitions, totalSize, nextPageToken, nil

}

func (r *repository) GetConnectorDefinitionByID(id string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.ConnectorDefinition, error) {
	var connectorDefinition datamodel.ConnectorDefinition
	queryBuilder := r.db.Model(&datamodel.ConnectorDefinition{}).Where("id = ? AND connector_type = ?", id, connectorType)
	if isBasicView {
		queryBuilder.Omit("spec")
	}
	if result := queryBuilder.First(&connectorDefinition); result.Error != nil {
		return nil, status.Errorf(codes.NotFound, "[GetConnectorDefinitionByID] The connector with connector_type '%s' and id '%s' you specified is not found", connectorPB.ConnectorType(connectorType), id)
	}
	return &connectorDefinition, nil
}

func (r *repository) GetConnectorDefinitionByUID(uid uuid.UUID, isBasicView bool) (*datamodel.ConnectorDefinition, error) {
	var connectorDefinition datamodel.ConnectorDefinition
	queryBuilder := r.db.Model(&datamodel.ConnectorDefinition{}).Where("uid = ?", uid)
	if isBasicView {
		queryBuilder.Omit("spec")
	}
	if result := queryBuilder.First(&connectorDefinition); result.Error != nil {
		return nil, status.Errorf(codes.NotFound, "[GetConnectorDefinitionByUID] The connector with uid '%s' you specified is not found", uid)
	}
	return &connectorDefinition, nil
}

func (r *repository) CreateConnector(connector *datamodel.Connector) error {
	if result := r.db.Model(&datamodel.Connector{}).Create(connector); result.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(result.Error, &pgErr) {
			if pgErr.Code == "23505" {
				return status.Errorf(codes.AlreadyExists, pgErr.Message)
			}
		}
	}
	return nil
}

func (r *repository) ListConnector(ownerPermalink string, connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) (connectors []*datamodel.Connector, totalSize int64, nextPageToken string, err error) {

	r.db.Model(&datamodel.Connector{}).Where("owner = ? AND connector_type = ?", ownerPermalink, connectorType).Count(&totalSize)

	queryBuilder := r.db.Model(&datamodel.Connector{}).Order("create_time DESC, uid DESC").Where("owner = ? AND connector_type = ?", ownerPermalink, connectorType)

	if pageSize == 0 {
		pageSize = DefaultPageSize
	} else if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	queryBuilder = queryBuilder.Limit(int(pageSize))

	if pageToken != "" {
		createdAt, uid, err := paginate.DecodeToken(pageToken)
		if err != nil {
			return nil, 0, "", status.Errorf(codes.InvalidArgument, "Invalid page token: %s", err.Error())
		}
		queryBuilder = queryBuilder.Where("(create_time,uid) < (?::timestamp, ?)", createdAt, uid)
	}

	if isBasicView {
		queryBuilder.Omit("configuration")
	}

	var createTime time.Time // only using one for all loops, we only need the latest one in the end
	rows, err := queryBuilder.Rows()
	if err != nil {
		return nil, 0, "", status.Errorf(codes.Internal, err.Error())
	}
	defer rows.Close()
	for rows.Next() {
		var item datamodel.Connector
		if err = r.db.ScanRows(rows, &item); err != nil {
			return nil, 0, "", status.Errorf(codes.Internal, err.Error())
		}
		createTime = item.CreateTime
		connectors = append(connectors, &item)
	}

	if len(connectors) > 0 {
		lastUID := (connectors)[len(connectors)-1].UID
		lastItem := &datamodel.Connector{}
		if result := r.db.Model(&datamodel.Connector{}).
			Where("owner = ? AND connector_type = ?", ownerPermalink, connectorType).
			Order("create_time ASC, uid ASC").Limit(1).Find(lastItem); result.Error != nil {
			return nil, 0, "", status.Errorf(codes.Internal, result.Error.Error())
		}
		if lastItem.UID.String() == lastUID.String() {
			nextPageToken = ""
		} else {
			nextPageToken = paginate.EncodeToken(createTime, lastUID.String())
		}
	}

	return connectors, totalSize, nextPageToken, nil
}

func (r *repository) GetConnectorByID(id string, ownerPermalink string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.Connector, error) {
	var connector datamodel.Connector

	queryBuilder := r.db.Model(&datamodel.Connector{}).
		Where("id = ? AND owner = ? AND connector_type = ?", id, ownerPermalink, connectorType)

	if isBasicView {
		queryBuilder.Omit("configuration")
	}

	if result := queryBuilder.First(&connector); result.Error != nil {
		return nil, status.Errorf(codes.NotFound, "[GetConnectorByID] The connector with connector_type '%s' and id '%s' you specified is not found", connectorPB.ConnectorType(connectorType), id)
	}
	return &connector, nil
}

func (r *repository) GetConnectorByUID(uid uuid.UUID, ownerPermalink string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.Connector, error) {
	var connector datamodel.Connector

	queryBuilder := r.db.Model(&datamodel.Connector{}).
		Where("uid = ? AND owner = ? AND connector_type = ?", uid, ownerPermalink, connectorType)

	if isBasicView {
		queryBuilder.Omit("configuration")
	}

	if result := queryBuilder.First(&connector); result.Error != nil {
		return nil, status.Errorf(codes.NotFound, "[GetConnectorByUID] The connector with connector_type '%s' and uid '%s' you specified is not found", connectorPB.ConnectorType(connectorType), uid)
	}
	return &connector, nil
}

func (r *repository) UpdateConnector(id string, ownerPermalink string, connectorType datamodel.ConnectorType, connector *datamodel.Connector) error {
	if result := r.db.Model(&datamodel.Connector{}).
		Where("id = ? AND owner = ? AND connector_type = ?", id, ownerPermalink, connectorType).
		Updates(connector); result.Error != nil {
		return status.Errorf(codes.Internal, result.Error.Error())
	} else if result.RowsAffected == 0 {
		return status.Errorf(codes.NotFound, "[UpdateConnector] The connector with connector_type '%s' and id '%s' you specified is not found", connectorPB.ConnectorType(connectorType), id)
	}
	return nil
}

func (r *repository) DeleteConnector(id string, ownerPermalink string, connectorType datamodel.ConnectorType) error {

	result := r.db.Model(&datamodel.Connector{}).
		Where("id = ? AND owner = ? AND connector_type = ?", id, ownerPermalink, connectorType).
		Delete(&datamodel.Connector{})

	if result.Error != nil {
		return status.Error(codes.Internal, result.Error.Error())
	}

	if result.RowsAffected == 0 {
		return status.Errorf(codes.NotFound, "[DeleteConnector] The connector with connector_type '%s' and id '%s' you specified is not found", connectorPB.ConnectorType(connectorType), id)
	}

	return nil
}

func (r *repository) UpdateConnectorID(id string, ownerPermalink string, connectorType datamodel.ConnectorType, newID string) error {
	if result := r.db.Model(&datamodel.Connector{}).
		Where("id = ? AND owner = ? AND connector_type = ?", id, ownerPermalink, connectorType).
		Update("id", newID); result.Error != nil {
		return status.Errorf(codes.Internal, result.Error.Error())
	} else if result.RowsAffected == 0 {
		return status.Errorf(codes.NotFound, "[UpdateConnectorID] The connector with connector_type '%s' and id '%s' you specified is not found", connectorPB.ConnectorType(connectorType), id)
	}
	return nil
}

func (r *repository) UpdateConnectorStateByID(id string, ownerPermalink string, connectorType datamodel.ConnectorType, state datamodel.ConnectorState) error {
	if result := r.db.Model(&datamodel.Connector{}).
		Where("id = ? AND owner = ? AND connector_type = ?", id, ownerPermalink, connectorType).
		Update("state", state); result.Error != nil {
		return status.Errorf(codes.Internal, result.Error.Error())
	} else if result.RowsAffected == 0 {
		return status.Errorf(codes.NotFound, "[UpdateConnectorStateByID] The connector with connector_type '%s' and id '%s' you specified is not found", connectorPB.ConnectorType(connectorType), id)
	}
	return nil
}

func (r *repository) UpdateConnectorStateByUID(uid uuid.UUID, ownerPermalink string, connectorType datamodel.ConnectorType, state datamodel.ConnectorState) error {
	if result := r.db.Model(&datamodel.Connector{}).
		Where("uid = ? AND owner = ? AND connector_type = ?", uid, ownerPermalink, connectorType).
		Update("state", state); result.Error != nil {
		return status.Errorf(codes.Internal, result.Error.Error())
	} else if result.RowsAffected == 0 {
		return status.Errorf(codes.NotFound, "[UpdateConnectorStateByUID] The connector with connector_type '%s' and uuid '%s' you specified is not found", connectorPB.ConnectorType(connectorType), uid)
	}
	return nil
}
