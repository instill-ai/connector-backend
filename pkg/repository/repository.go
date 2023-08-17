package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/gogo/status"
	"github.com/jackc/pgx/v5/pgconn"
	"go.einride.tech/aip/filtering"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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
	CreateConnectorResource(ctx context.Context, connector *datamodel.ConnectorResource) error
	ListConnectorResources(ctx context.Context, ownerPermalink string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*datamodel.ConnectorResource, int64, string, error)
	GetConnectorResourceByID(ctx context.Context, id string, ownerPermalink string, isBasicView bool) (*datamodel.ConnectorResource, error)
	GetConnectorResourceByUID(ctx context.Context, uid uuid.UUID, ownerPermalink string, isBasicView bool) (*datamodel.ConnectorResource, error)
	UpdateConnectorResource(ctx context.Context, id string, ownerPermalink string, connector *datamodel.ConnectorResource) error
	DeleteConnectorResource(ctx context.Context, id string, ownerPermalink string) error
	UpdateConnectorResourceID(ctx context.Context, id string, ownerPermalink string, newID string) error
	UpdateConnectorResourceStateByID(ctx context.Context, id string, ownerPermalink string, state datamodel.ConnectorResourceState) error
	UpdateConnectorResourceStateByUID(ctx context.Context, uid uuid.UUID, ownerPermalink string, state datamodel.ConnectorResourceState) error
	UpdateConnectorResourceTaskByID(ctx context.Context, id string, ownerPermalink string, task datamodel.Task) error

	ListConnectorResourcesAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*datamodel.ConnectorResource, int64, string, error)
	GetConnectorResourceByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool) (*datamodel.ConnectorResource, error)
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

func (r *repository) CreateConnectorResource(ctx context.Context, connector *datamodel.ConnectorResource) error {

	logger, _ := logger.GetZapLogger(ctx)

	if result := r.db.Model(&datamodel.ConnectorResource{}).Create(connector); result.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(result.Error, &pgErr) {
			if pgErr.Code == "23505" {
				st, err := sterr.CreateErrorResourceInfo(
					codes.AlreadyExists,
					fmt.Sprintf("[db] create connector error: %s", pgErr.Message),
					"connector",
					fmt.Sprintf("id %s", connector.ID),
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

func (r *repository) ListConnectorResources(ctx context.Context, ownerPermalink string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) (connectors []*datamodel.ConnectorResource, totalSize int64, nextPageToken string, err error) {

	logger, _ := logger.GetZapLogger(ctx)

	var expr *clause.Expr
	if expr, err = r.transpileFilter(filter); err != nil {
		return nil, 0, "", status.Errorf(codes.Internal, err.Error())
	}

	if expr == nil {
		r.db.Model(&datamodel.ConnectorResource{}).Where("owner = ? or visibility = ?", ownerPermalink, datamodel.ConnectorResourceVisibility(connectorPB.ConnectorResource_VISIBILITY_PUBLIC)).Count(&totalSize)
	} else {
		r.db.Model(&datamodel.ConnectorResource{}).Where("(owner = ? or visibility = ?) and ?", ownerPermalink, datamodel.ConnectorResourceVisibility(connectorPB.ConnectorResource_VISIBILITY_PUBLIC), expr).Count(&totalSize)
	}

	queryBuilder := r.db.Model(&datamodel.ConnectorResource{}).Order("create_time DESC, uid DESC").Where("owner = ? or visibility = ?", ownerPermalink, datamodel.ConnectorResourceVisibility(connectorPB.ConnectorResource_VISIBILITY_PUBLIC))

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

	if expr != nil {
		queryBuilder.Where("(?)", expr)
	}

	var createTime time.Time // only using one for all loops, we only need the latest one in the end
	rows, err := queryBuilder.Rows()
	if err != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[db] list connector error: %s", err.Error()),
			"connector",
			"",
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
		var item datamodel.ConnectorResource
		if err = r.db.ScanRows(rows, &item); err != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.Internal,
				fmt.Sprintf("[db] list connector error: %s", err.Error()),
				"connector",
				"",
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
		lastItem := &datamodel.ConnectorResource{}
		if expr == nil {
			if result := r.db.Model(&datamodel.ConnectorResource{}).
				Where("owner = ? or visibility = ?", ownerPermalink, datamodel.ConnectorResourceVisibility(connectorPB.ConnectorResource_VISIBILITY_PUBLIC)).
				Order("create_time ASC, uid ASC").Limit(1).Find(lastItem); result.Error != nil {
				st, err := sterr.CreateErrorResourceInfo(
					codes.Internal,
					fmt.Sprintf("[db] list connector error: %s", err.Error()),
					"connector",
					"",
					ownerPermalink,
					result.Error.Error(),
				)
				if err != nil {
					logger.Error(err.Error())
				}
				return nil, 0, "", st.Err()
			}
		} else {
			if result := r.db.Model(&datamodel.ConnectorResource{}).
				Where("(owner = ? or visibility = ?) and ?", ownerPermalink, datamodel.ConnectorResourceVisibility(connectorPB.ConnectorResource_VISIBILITY_PUBLIC), expr).
				Order("create_time ASC, uid ASC").Limit(1).Find(lastItem); result.Error != nil {
				st, err := sterr.CreateErrorResourceInfo(
					codes.Internal,
					fmt.Sprintf("[db] list connector error: %s", err.Error()),
					"connector",
					"",
					ownerPermalink,
					result.Error.Error(),
				)
				if err != nil {
					logger.Error(err.Error())
				}
				return nil, 0, "", st.Err()
			}
		}

		if lastItem.UID.String() == lastUID.String() {
			nextPageToken = ""
		} else {
			nextPageToken = paginate.EncodeToken(createTime, lastUID.String())
		}
	}

	return connectors, totalSize, nextPageToken, nil
}

func (r *repository) ListConnectorResourcesAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) (connectors []*datamodel.ConnectorResource, totalSize int64, nextPageToken string, err error) {

	logger, _ := logger.GetZapLogger(ctx)
	var expr *clause.Expr
	if expr, err = r.transpileFilter(filter); err != nil {
		return nil, 0, "", status.Errorf(codes.Internal, err.Error())
	}

	if expr == nil {
		r.db.Model(&datamodel.ConnectorResource{}).Count(&totalSize)
	} else {
		r.db.Model(&datamodel.ConnectorResource{}).Where("?", expr).Count(&totalSize)
	}

	queryBuilder := r.db.Model(&datamodel.ConnectorResource{}).Order("create_time DESC, uid DESC")

	if pageSize == 0 {
		pageSize = DefaultPageSize
	} else if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	if expr != nil {
		queryBuilder.Clauses(expr)
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
			"",
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
		var item datamodel.ConnectorResource
		if err = r.db.ScanRows(rows, &item); err != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.Internal,
				fmt.Sprintf("[db] list connector error: %s", err.Error()),
				"connector",
				"",
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
		lastItem := &datamodel.ConnectorResource{}
		if expr == nil {
			if result := r.db.Model(&datamodel.ConnectorResource{}).
				Order("create_time ASC, uid ASC").Limit(1).Find(lastItem); result.Error != nil {
				st, err := sterr.CreateErrorResourceInfo(
					codes.Internal,
					fmt.Sprintf("[db] list connector error: %s", err.Error()),
					"connector",
					"",
					"admin",
					result.Error.Error(),
				)
				if err != nil {
					logger.Error(err.Error())
				}
				return nil, 0, "", st.Err()
			}
		} else {
			if result := r.db.Model(&datamodel.ConnectorResource{}).
				Where("?", expr).
				Order("create_time ASC, uid ASC").Limit(1).Find(lastItem); result.Error != nil {
				st, err := sterr.CreateErrorResourceInfo(
					codes.Internal,
					fmt.Sprintf("[db] list connector error: %s", err.Error()),
					"connector",
					"",
					"admin",
					result.Error.Error(),
				)
				if err != nil {
					logger.Error(err.Error())
				}
				return nil, 0, "", st.Err()
			}
		}

		if lastItem.UID.String() == lastUID.String() {
			nextPageToken = ""
		} else {
			nextPageToken = paginate.EncodeToken(createTime, lastUID.String())
		}
	}

	return connectors, totalSize, nextPageToken, nil
}

func (r *repository) GetConnectorResourceByID(ctx context.Context, id string, ownerPermalink string, isBasicView bool) (*datamodel.ConnectorResource, error) {

	logger, _ := logger.GetZapLogger(ctx)

	var connector datamodel.ConnectorResource

	queryBuilder := r.db.Model(&datamodel.ConnectorResource{}).
		Where("id = ? AND (owner = ? or visibility = ?)", id, ownerPermalink, datamodel.ConnectorResourceVisibility(connectorPB.ConnectorResource_VISIBILITY_PUBLIC))

	if isBasicView {
		queryBuilder.Omit("configuration")
	}

	if result := queryBuilder.First(&connector); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.NotFound,
			fmt.Sprintf("[db] get connector by id error: %s", result.Error.Error()),
			"connector",
			"",
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

func (r *repository) GetConnectorResourceByUID(ctx context.Context, uid uuid.UUID, ownerPermalink string, isBasicView bool) (*datamodel.ConnectorResource, error) {

	logger, _ := logger.GetZapLogger(ctx)

	var connector datamodel.ConnectorResource

	queryBuilder := r.db.Model(&datamodel.ConnectorResource{}).
		Where("uid = ? AND (owner = ? or visibility = ?)", uid, ownerPermalink, datamodel.ConnectorResourceVisibility(connectorPB.ConnectorResource_VISIBILITY_PUBLIC))

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

func (r *repository) GetConnectorResourceByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool) (*datamodel.ConnectorResource, error) {

	logger, _ := logger.GetZapLogger(ctx)

	var connector datamodel.ConnectorResource

	queryBuilder := r.db.Model(&datamodel.ConnectorResource{}).
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

func (r *repository) UpdateConnectorResource(ctx context.Context, id string, ownerPermalink string, connector *datamodel.ConnectorResource) error {

	logger, _ := logger.GetZapLogger(ctx)

	if result := r.db.Model(&datamodel.ConnectorResource{}).
		Where("id = ? AND owner = ?", id, ownerPermalink).
		Updates(connector); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[db] update connector error: %s", result.Error.Error()),
			"connector",
			"",
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
			"",
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

func (r *repository) DeleteConnectorResource(ctx context.Context, id string, ownerPermalink string) error {

	logger, _ := logger.GetZapLogger(ctx)

	result := r.db.Model(&datamodel.ConnectorResource{}).
		Where("id = ? AND owner = ? ", id, ownerPermalink).
		Delete(&datamodel.ConnectorResource{})

	if result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[db] delete connector error: %s", result.Error.Error()),
			"connector",
			"",
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
			"",
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

func (r *repository) UpdateConnectorResourceID(ctx context.Context, id string, ownerPermalink string, newID string) error {

	logger, _ := logger.GetZapLogger(ctx)

	if result := r.db.Model(&datamodel.ConnectorResource{}).
		Where("id = ? AND owner = ? ", id, ownerPermalink).
		Update("id", newID); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[db] update connector id error: %s", result.Error.Error()),
			"connector",
			"",
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
			"",
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

func (r *repository) UpdateConnectorResourceStateByID(ctx context.Context, id string, ownerPermalink string, state datamodel.ConnectorResourceState) error {

	logger, _ := logger.GetZapLogger(ctx)

	if result := r.db.Model(&datamodel.ConnectorResource{}).
		Where("id = ? AND owner = ?", id, ownerPermalink).
		Update("state", state); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[db] update connector state by id error: %s", result.Error.Error()),
			"connector",
			"",
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
			"",
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

func (r *repository) UpdateConnectorResourceTaskByID(ctx context.Context, id string, ownerPermalink string, task datamodel.Task) error {
	// TODO: refactor this
	logger, _ := logger.GetZapLogger(ctx)

	if result := r.db.Model(&datamodel.ConnectorResource{}).
		Where("id = ? AND owner = ?", id, ownerPermalink).
		Update("task", task); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[db] update connector task by id error: %s", result.Error.Error()),
			"connector",
			"",
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
			fmt.Sprintf("[db] update connector task by id error: %s", "Not found"),
			"connector",
			"",
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

func (r *repository) UpdateConnectorResourceStateByUID(ctx context.Context, uid uuid.UUID, ownerPermalink string, state datamodel.ConnectorResourceState) error {

	logger, _ := logger.GetZapLogger(ctx)

	if result := r.db.Model(&datamodel.ConnectorResource{}).
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

// TranspileFilter transpiles a parsed AIP filter expression to GORM DB clauses
func (r *repository) transpileFilter(filter filtering.Filter) (*clause.Expr, error) {
	return (&Transpiler{
		filter: filter,
	}).Transpile()
}
