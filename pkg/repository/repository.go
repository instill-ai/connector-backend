package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"

	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/connector-backend/pkg/logger"
	"github.com/instill-ai/x/paginate"
	"github.com/instill-ai/x/sterr"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

// DefaultPageSize is the default pagination page size when page size is not assigned
const DefaultPageSize = 10

// MaxPageSize is the maximum pagination page size if the assigned value is over this number
const MaxPageSize = 100

// Repository interface
type Repository interface {

	// Connector
	CreateConnector(ctx context.Context, connector *datamodel.Connector) error
	ListConnectors(ctx context.Context, ownerPermalink string, connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) ([]*datamodel.Connector, int64, string, error)
	GetConnectorByID(ctx context.Context, id string, ownerPermalink string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.Connector, error)
	GetConnectorByUID(ctx context.Context, uid uuid.UUID, ownerPermalink string, isBasicView bool) (*datamodel.Connector, error)
	UpdateConnector(ctx context.Context, id string, ownerPermalink string, connectorType datamodel.ConnectorType, connector *datamodel.Connector) error
	DeleteConnector(ctx context.Context, id string, ownerPermalink string, connectorType datamodel.ConnectorType) error
	UpdateConnectorID(ctx context.Context, id string, ownerPermalink string, connectorType datamodel.ConnectorType, newID string) error
	UpdateConnectorStateByID(ctx context.Context, id string, ownerPermalink string, connectorType datamodel.ConnectorType, state datamodel.ConnectorState) error
	UpdateConnectorStateByUID(ctx context.Context, uid uuid.UUID, ownerPermalink string, state datamodel.ConnectorState) error

	ListConnectorsAdmin(ctx context.Context, connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) ([]*datamodel.Connector, int64, string, error)
	GetConnectorByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool) (*datamodel.Connector, error)
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

func (r *repository) CreateConnector(ctx context.Context, connector *datamodel.Connector) error {

	logger, _ := logger.GetZapLogger(ctx)

	if result := r.db.Model(&datamodel.Connector{}).Create(connector); result.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(result.Error, &pgErr) {
			if pgErr.Code == "23505" {
				st, err := sterr.CreateErrorResourceInfo(
					codes.AlreadyExists,
					fmt.Sprintf("[db] create connector error: %s", pgErr.Message),
					"connector",
					fmt.Sprintf("id %s and connector_type %s", connector.ID, connectorPB.ConnectorType(connector.ConnectorType)),
					connector.Owner,
					pgErr.Message,
				)
				if err != nil {
					logger.Error(err.Error())
				}
				return st.Err()
			}
		}
	}
	return nil
}

func (r *repository) ListConnectors(ctx context.Context, ownerPermalink string, connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) (connectors []*datamodel.Connector, totalSize int64, nextPageToken string, err error) {

	logger, _ := logger.GetZapLogger(ctx)

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
			st, err := sterr.CreateErrorBadRequest(
				fmt.Sprintf("[db] list connector error: %s", err.Error()),
				[]*errdetails.BadRequest_FieldViolation{
					{
						Field:       "page_token",
						Description: fmt.Sprintf("Invalid page token: %s", err.Error()),
					},
				},
			)
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, 0, "", st.Err()
		}

		queryBuilder = queryBuilder.Where("(create_time,uid) < (?::timestamp, ?)", createdAt, uid)
	}

	if isBasicView {
		queryBuilder.Omit("configuration")
	}

	var createTime time.Time // only using one for all loops, we only need the latest one in the end
	rows, err := queryBuilder.Rows()
	if err != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[db] list connector error: %s", err.Error()),
			"connector",
			fmt.Sprintf("connector_type %s", connectorPB.ConnectorType(connectorType)),
			ownerPermalink,
			err.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return nil, 0, "", st.Err()
	}
	defer rows.Close()
	for rows.Next() {
		var item datamodel.Connector
		if err = r.db.ScanRows(rows, &item); err != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.Internal,
				fmt.Sprintf("[db] list connector error: %s", err.Error()),
				"connector",
				fmt.Sprintf("connector_type %s", connectorPB.ConnectorType(connectorType)),
				ownerPermalink,
				err.Error(),
			)
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, 0, "", st.Err()
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
			st, err := sterr.CreateErrorResourceInfo(
				codes.Internal,
				fmt.Sprintf("[db] list connector error: %s", err.Error()),
				"connector",
				fmt.Sprintf("connector_type %s", connectorPB.ConnectorType(connectorType)),
				ownerPermalink,
				result.Error.Error(),
			)
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, 0, "", st.Err()
		}
		if lastItem.UID.String() == lastUID.String() {
			nextPageToken = ""
		} else {
			nextPageToken = paginate.EncodeToken(createTime, lastUID.String())
		}
	}

	return connectors, totalSize, nextPageToken, nil
}

func (r *repository) ListConnectorsAdmin(ctx context.Context, connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) (connectors []*datamodel.Connector, totalSize int64, nextPageToken string, err error) {

	logger, _ := logger.GetZapLogger(ctx)

	r.db.Model(&datamodel.Connector{}).Where("connector_type = ?", connectorType).Count(&totalSize)

	queryBuilder := r.db.Model(&datamodel.Connector{}).Order("create_time DESC, uid DESC").Where("connector_type = ?", connectorType)

	if pageSize == 0 {
		pageSize = DefaultPageSize
	} else if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	queryBuilder = queryBuilder.Limit(int(pageSize))

	if pageToken != "" {
		createdAt, uid, err := paginate.DecodeToken(pageToken)
		if err != nil {
			st, err := sterr.CreateErrorBadRequest(
				fmt.Sprintf("[db] list connector error: %s", err.Error()),
				[]*errdetails.BadRequest_FieldViolation{
					{
						Field:       "page_token",
						Description: fmt.Sprintf("Invalid page token: %s", err.Error()),
					},
				},
			)
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, 0, "", st.Err()
		}

		queryBuilder = queryBuilder.Where("(create_time,uid) < (?::timestamp, ?)", createdAt, uid)
	}

	if isBasicView {
		queryBuilder.Omit("configuration")
	}

	var createTime time.Time // only using one for all loops, we only need the latest one in the end
	rows, err := queryBuilder.Rows()
	if err != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[db] list connector error: %s", err.Error()),
			"connector",
			fmt.Sprintf("connector_type %s", connectorPB.ConnectorType(connectorType)),
			"admin",
			err.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return nil, 0, "", st.Err()
	}
	defer rows.Close()
	for rows.Next() {
		var item datamodel.Connector
		if err = r.db.ScanRows(rows, &item); err != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.Internal,
				fmt.Sprintf("[db] list connector error: %s", err.Error()),
				"connector",
				fmt.Sprintf("connector_type %s", connectorPB.ConnectorType(connectorType)),
				"admin",
				err.Error(),
			)
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, 0, "", st.Err()
		}
		createTime = item.CreateTime
		connectors = append(connectors, &item)
	}

	if len(connectors) > 0 {
		lastUID := (connectors)[len(connectors)-1].UID
		lastItem := &datamodel.Connector{}
		if result := r.db.Model(&datamodel.Connector{}).
			Where("connector_type = ?", connectorType).
			Order("create_time ASC, uid ASC").Limit(1).Find(lastItem); result.Error != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.Internal,
				fmt.Sprintf("[db] list connector error: %s", err.Error()),
				"connector",
				fmt.Sprintf("connector_type %s", connectorPB.ConnectorType(connectorType)),
				"admin",
				result.Error.Error(),
			)
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, 0, "", st.Err()
		}
		if lastItem.UID.String() == lastUID.String() {
			nextPageToken = ""
		} else {
			nextPageToken = paginate.EncodeToken(createTime, lastUID.String())
		}
	}

	return connectors, totalSize, nextPageToken, nil
}

func (r *repository) GetConnectorByID(ctx context.Context, id string, ownerPermalink string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.Connector, error) {

	logger, _ := logger.GetZapLogger(ctx)

	var connector datamodel.Connector

	queryBuilder := r.db.Model(&datamodel.Connector{}).
		Where("id = ? AND owner = ? AND connector_type = ?", id, ownerPermalink, connectorType)

	if isBasicView {
		queryBuilder.Omit("configuration")
	}

	if result := queryBuilder.First(&connector); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.NotFound,
			fmt.Sprintf("[db] get connector by id error: %s", result.Error.Error()),
			"connector",
			fmt.Sprintf("id %s and connector_type %s", id, connectorPB.ConnectorType(connectorType)),
			ownerPermalink,
			result.Error.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return nil, st.Err()
	}
	return &connector, nil
}

func (r *repository) GetConnectorByUID(ctx context.Context, uid uuid.UUID, ownerPermalink string, isBasicView bool) (*datamodel.Connector, error) {

	logger, _ := logger.GetZapLogger(ctx)

	var connector datamodel.Connector

	queryBuilder := r.db.Model(&datamodel.Connector{}).
		Where("uid = ? AND owner = ?", uid, ownerPermalink)

	if isBasicView {
		queryBuilder.Omit("configuration")
	}

	if result := queryBuilder.First(&connector); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.NotFound,
			fmt.Sprintf("[db] get connector by uid error: %s", result.Error.Error()),
			"connector",
			uid.String(),
			ownerPermalink,
			result.Error.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return nil, st.Err()
	}
	return &connector, nil
}

func (r *repository) GetConnectorByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool) (*datamodel.Connector, error) {

	logger, _ := logger.GetZapLogger(ctx)

	var connector datamodel.Connector

	queryBuilder := r.db.Model(&datamodel.Connector{}).
		Where("uid = ?", uid)

	if isBasicView {
		queryBuilder.Omit("configuration")
	}

	if result := queryBuilder.First(&connector); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.NotFound,
			fmt.Sprintf("[db] get connector by uid error: %s", result.Error.Error()),
			"connector",
			uid.String(),
			"admin",
			result.Error.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return nil, st.Err()
	}
	return &connector, nil
}

func (r *repository) UpdateConnector(ctx context.Context, id string, ownerPermalink string, connectorType datamodel.ConnectorType, connector *datamodel.Connector) error {

	logger, _ := logger.GetZapLogger(ctx)

	if result := r.db.Model(&datamodel.Connector{}).
		Where("id = ? AND owner = ? AND connector_type = ?", id, ownerPermalink, connectorType).
		Updates(connector); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[db] update connector error: %s", result.Error.Error()),
			"connector",
			fmt.Sprintf("id %s and connector_type %s", id, connectorPB.ConnectorType(connectorType)),
			ownerPermalink,
			result.Error.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	} else if result.RowsAffected == 0 {
		st, err := sterr.CreateErrorResourceInfo(
			codes.NotFound,
			fmt.Sprintf("[db] update connector error: %s", "Not found"),
			"connector",
			fmt.Sprintf("id %s and connector_type %s", id, connectorPB.ConnectorType(connectorType)),
			ownerPermalink,
			"Not found",
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	}
	return nil
}

func (r *repository) DeleteConnector(ctx context.Context, id string, ownerPermalink string, connectorType datamodel.ConnectorType) error {

	logger, _ := logger.GetZapLogger(ctx)

	result := r.db.Model(&datamodel.Connector{}).
		Where("id = ? AND owner = ? AND connector_type = ?", id, ownerPermalink, connectorType).
		Delete(&datamodel.Connector{})

	if result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[db] delete connector error: %s", result.Error.Error()),
			"connector",
			fmt.Sprintf("id %s and connector_type %s", id, connectorPB.ConnectorType(connectorType)),
			ownerPermalink,
			result.Error.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	}

	if result.RowsAffected == 0 {
		st, err := sterr.CreateErrorResourceInfo(
			codes.NotFound,
			fmt.Sprintf("[db] delete connector error: %s", "Not found"),
			"connector",
			fmt.Sprintf("id %s and connector_type %s", id, connectorPB.ConnectorType(connectorType)),
			ownerPermalink,
			"Not found",
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	}

	return nil
}

func (r *repository) UpdateConnectorID(ctx context.Context, id string, ownerPermalink string, connectorType datamodel.ConnectorType, newID string) error {

	logger, _ := logger.GetZapLogger(ctx)

	if result := r.db.Model(&datamodel.Connector{}).
		Where("id = ? AND owner = ? AND connector_type = ?", id, ownerPermalink, connectorType).
		Update("id", newID); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[db] update connector id error: %s", result.Error.Error()),
			"connector",
			fmt.Sprintf("id %s and connector_type %s", id, connectorPB.ConnectorType(connectorType)),
			ownerPermalink,
			result.Error.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	} else if result.RowsAffected == 0 {
		st, err := sterr.CreateErrorResourceInfo(
			codes.NotFound,
			fmt.Sprintf("[db] update connector id error: %s", "Not found"),
			"connector",
			fmt.Sprintf("id %s and connector_type %s", id, connectorPB.ConnectorType(connectorType)),
			ownerPermalink,
			"Not found",
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	}
	return nil
}

func (r *repository) UpdateConnectorStateByID(ctx context.Context, id string, ownerPermalink string, connectorType datamodel.ConnectorType, state datamodel.ConnectorState) error {

	logger, _ := logger.GetZapLogger(ctx)

	if result := r.db.Model(&datamodel.Connector{}).
		Where("id = ? AND owner = ? AND connector_type = ?", id, ownerPermalink, connectorType).
		Update("state", state); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[db] update connector state by id error: %s", result.Error.Error()),
			"connector",
			fmt.Sprintf("id %s and connector_type %s", id, connectorPB.ConnectorType(connectorType)),
			ownerPermalink,
			result.Error.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	} else if result.RowsAffected == 0 {
		st, err := sterr.CreateErrorResourceInfo(
			codes.NotFound,
			fmt.Sprintf("[db] update connector state by id error: %s", "Not found"),
			"connector",
			fmt.Sprintf("id %s and connector_type %s", id, connectorPB.ConnectorType(connectorType)),
			ownerPermalink,
			"Not found",
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	}
	return nil
}

func (r *repository) UpdateConnectorStateByUID(ctx context.Context, uid uuid.UUID, ownerPermalink string, state datamodel.ConnectorState) error {

	logger, _ := logger.GetZapLogger(ctx)

	if result := r.db.Model(&datamodel.Connector{}).
		Where("uid = ? AND owner = ?", uid, ownerPermalink).
		Update("state", state); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[db] update connector state by uid error: %s", result.Error.Error()),
			"connector",
			uid.String(),
			ownerPermalink,
			result.Error.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	} else if result.RowsAffected == 0 {
		st, err := sterr.CreateErrorResourceInfo(
			codes.NotFound,
			fmt.Sprintf("[db] update connector state by uid error: %s", "Not found"),
			"connector",
			uid.String(),
			ownerPermalink,
			"Not found",
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	}

	return nil
}
