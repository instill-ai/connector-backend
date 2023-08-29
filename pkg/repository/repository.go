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

const VisibilityPublic = datamodel.ConnectorResourceVisibility(connectorPB.ConnectorResource_VISIBILITY_PUBLIC)

// Repository interface
type Repository interface {

	// List all connector resources visible to the user
	ListConnectorResources(ctx context.Context, userPermalink string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*datamodel.ConnectorResource, int64, string, error)
	GetConnectorResourceByUID(ctx context.Context, userPermalink string, uid uuid.UUID, isBasicView bool) (*datamodel.ConnectorResource, error)

	// Operations for resources under {ownerPermalink} namespace, view by {userPermalink}
	CreateUserConnectorResource(ctx context.Context, ownerPermalink string, userPermalink string, connector *datamodel.ConnectorResource) error
	ListUserConnectorResources(ctx context.Context, ownerPermalink string, userPermalink string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*datamodel.ConnectorResource, int64, string, error)
	GetUserConnectorResourceByID(ctx context.Context, ownerPermalink string, userPermalink string, id string, isBasicView bool) (*datamodel.ConnectorResource, error)
	UpdateUserConnectorResourceByID(ctx context.Context, ownerPermalink string, userPermalink string, id string, connector *datamodel.ConnectorResource) error
	DeleteUserConnectorResourceByID(ctx context.Context, ownerPermalink string, userPermalink string, id string) error
	UpdateUserConnectorResourceIDByID(ctx context.Context, ownerPermalink string, userPermalink string, id string, newID string) error
	UpdateUserConnectorResourceStateByID(ctx context.Context, ownerPermalink string, userPermalink string, id string, state datamodel.ConnectorResourceState) error

	// Operations Admin
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

func (r *repository) listConnectorResources(ctx context.Context, where string, whereArgs []interface{}, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) (connectors []*datamodel.ConnectorResource, totalSize int64, nextPageToken string, err error) {

	var expr *clause.Expr
	if expr, err = r.transpileFilter(filter); err != nil {
		return nil, 0, "", status.Errorf(codes.Internal, err.Error())
	}
	if expr != nil {
		if len(whereArgs) == 0 {
			where = "(?)"
			whereArgs = append(whereArgs, expr)
		} else {
			where = fmt.Sprintf("((%s) AND ?)", where)
			whereArgs = append(whereArgs, expr)
		}
	}

	logger, _ := logger.GetZapLogger(ctx)

	r.db.Model(&datamodel.ConnectorResource{}).Where(where, whereArgs...).Count(&totalSize)

	queryBuilder := r.db.Model(&datamodel.ConnectorResource{}).Order("create_time DESC, uid DESC").Where(where, whereArgs...)

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
			"",
			"",
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
				"",
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

		if result := r.db.Model(&datamodel.ConnectorResource{}).
			Where(where, whereArgs...).
			Order("create_time ASC, uid ASC").Limit(1).Find(lastItem); result.Error != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.Internal,
				fmt.Sprintf("[db] listConnectorResources: %s", err.Error()),
				"connector",
				"",
				"",
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

func (r *repository) ListConnectorResourcesAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) (connectors []*datamodel.ConnectorResource, totalSize int64, nextPageToken string, err error) {
	return r.listConnectorResources(ctx, "", []interface{}{}, pageSize, pageToken, isBasicView, filter)
}

func (r *repository) ListConnectorResources(ctx context.Context, userPermalink string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) (connectors []*datamodel.ConnectorResource, totalSize int64, nextPageToken string, err error) {

	return r.listConnectorResources(ctx,
		"(owner = ? OR visibility = ?)",
		[]interface{}{userPermalink, VisibilityPublic},
		pageSize, pageToken, isBasicView, filter)

}

func (r *repository) ListUserConnectorResources(ctx context.Context, ownerPermalink string, userPermalink string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) (connectors []*datamodel.ConnectorResource, totalSize int64, nextPageToken string, err error) {

	return r.listConnectorResources(ctx,
		"(owner = ? AND (visibility = ? OR ? = ?))",
		[]interface{}{ownerPermalink, VisibilityPublic, ownerPermalink, userPermalink},
		pageSize, pageToken, isBasicView, filter)

}

func (r *repository) CreateUserConnectorResource(ctx context.Context, ownerPermalink string, userPermalink string, connector *datamodel.ConnectorResource) error {

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

func (r *repository) getUserConnectorResource(ctx context.Context, where string, whereArgs []interface{}, isBasicView bool) (*datamodel.ConnectorResource, error) {
	logger, _ := logger.GetZapLogger(ctx)

	var connector datamodel.ConnectorResource

	queryBuilder := r.db.Model(&datamodel.ConnectorResource{}).Where(where, whereArgs...)

	if isBasicView {
		queryBuilder.Omit("configuration")
	}

	if result := queryBuilder.First(&connector); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.NotFound,
			fmt.Sprintf("[db] getUserConnectorResource error: %s", result.Error.Error()),
			"connector",
			"",
			"",
			result.Error.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return nil, st.Err()
	}
	return &connector, nil
}

func (r *repository) GetUserConnectorResourceByID(ctx context.Context, ownerPermalink string, userPermalink string, id string, isBasicView bool) (*datamodel.ConnectorResource, error) {

	return r.getUserConnectorResource(ctx,
		"(id = ? AND (owner = ? AND (visibility = ? OR ? = ?)))",
		[]interface{}{id, ownerPermalink, VisibilityPublic, ownerPermalink, userPermalink},
		isBasicView)
}

func (r *repository) GetConnectorResourceByUID(ctx context.Context, userPermalink string, uid uuid.UUID, isBasicView bool) (*datamodel.ConnectorResource, error) {

	// TODO: ACL
	return r.getUserConnectorResource(ctx,
		"(uid = ? AND (visibility = ? OR owner = ?))",
		[]interface{}{uid, VisibilityPublic, userPermalink},
		isBasicView)

}

func (r *repository) GetConnectorResourceByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool) (*datamodel.ConnectorResource, error) {
	return r.getUserConnectorResource(ctx,
		"(uid = ?)",
		[]interface{}{uid},
		isBasicView)
}

func (r *repository) UpdateUserConnectorResourceByID(ctx context.Context, ownerPermalink string, userPermalink string, id string, connector *datamodel.ConnectorResource) error {

	logger, _ := logger.GetZapLogger(ctx)

	if result := r.db.Model(&datamodel.ConnectorResource{}).
		Where("(id = ? AND owner = ? AND ? = ? )", id, ownerPermalink, ownerPermalink, userPermalink).
		Updates(connector); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[db] update connector error: %s", result.Error.Error()),
			"connector",
			"",
			"",
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
			"",
			"Not found",
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	}
	return nil
}

func (r *repository) DeleteUserConnectorResourceByID(ctx context.Context, ownerPermalink string, userPermalink string, id string) error {

	logger, _ := logger.GetZapLogger(ctx)

	result := r.db.Model(&datamodel.ConnectorResource{}).
		Where("(id = ? AND owner = ? AND ? = ?)", id, ownerPermalink, ownerPermalink, userPermalink).
		Delete(&datamodel.ConnectorResource{})

	if result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[db] delete connector error: %s", result.Error.Error()),
			"connector",
			"",
			"",
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
			"",
			"Not found",
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	}

	return nil
}

func (r *repository) UpdateUserConnectorResourceIDByID(ctx context.Context, ownerPermalink string, userPermalink string, id string, newID string) error {

	logger, _ := logger.GetZapLogger(ctx)

	if result := r.db.Model(&datamodel.ConnectorResource{}).
		Where("(id = ? AND owner = ? AND ? = ?)", id, ownerPermalink, ownerPermalink, userPermalink).
		Update("id", newID); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[db] update connector id error: %s", result.Error.Error()),
			"connector",
			"",
			"",
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
			"",
			"Not found",
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	}
	return nil
}

func (r *repository) UpdateUserConnectorResourceStateByID(ctx context.Context, ownerPermalink string, userPermalink string, id string, state datamodel.ConnectorResourceState) error {

	logger, _ := logger.GetZapLogger(ctx)

	if result := r.db.Model(&datamodel.ConnectorResource{}).
		Where("(id = ? AND owner = ? AND ? = ?)", id, ownerPermalink, ownerPermalink, userPermalink).
		Update("state", state); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[db] update connector state by id error: %s", result.Error.Error()),
			"connector",
			"",
			"",
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
			"",
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
