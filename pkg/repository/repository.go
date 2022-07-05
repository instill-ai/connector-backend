package repository

import (
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgconn"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"

	"github.com/instill-ai/connector-backend/internal/logger"
	"github.com/instill-ai/connector-backend/pkg/datamodel"
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

	logger, _ := logger.GetZapLogger()

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
			st, err := sterr.CreateErrorBadRequest(
				"[db] list connector definition error",
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
		queryBuilder.Omit("spec")
	}

	// only using one for all loops, we only need the latest one in the end
	var createTime time.Time
	rows, err := queryBuilder.Rows()
	if err != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			"[db] list connector definition error",
			"connector_definition",
			fmt.Sprintf("connector_type %s", connectorPB.ConnectorType(connectorType)),
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
		var item datamodel.ConnectorDefinition
		if err = r.db.ScanRows(rows, &item); err != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.Internal,
				"[db] list connector definition error",
				"connector_definition",
				fmt.Sprintf("connector_type %s", connectorPB.ConnectorType(connectorType)),
				"",
				err.Error(),
			)
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, 0, "", st.Err()
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
			st, err := sterr.CreateErrorResourceInfo(
				codes.Internal,
				"[db] list connector definition error",
				"connector_definition",
				fmt.Sprintf("connector_type %s", connectorPB.ConnectorType(connectorType)),
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

	return connectorDefinitions, totalSize, nextPageToken, nil

}

func (r *repository) GetConnectorDefinitionByID(id string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.ConnectorDefinition, error) {

	logger, _ := logger.GetZapLogger()

	var connectorDefinition datamodel.ConnectorDefinition
	queryBuilder := r.db.Model(&datamodel.ConnectorDefinition{}).Where("id = ? AND connector_type = ?", id, connectorType)
	if isBasicView {
		queryBuilder.Omit("spec")
	}
	if result := queryBuilder.First(&connectorDefinition); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.NotFound,
			"[db] get connector definition by id error",
			"connector_definition",
			fmt.Sprintf("id %s", id),
			"",
			result.Error.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return nil, st.Err()
	}

	return &connectorDefinition, nil
}

func (r *repository) GetConnectorDefinitionByUID(uid uuid.UUID, isBasicView bool) (*datamodel.ConnectorDefinition, error) {

	logger, _ := logger.GetZapLogger()

	var connectorDefinition datamodel.ConnectorDefinition
	queryBuilder := r.db.Model(&datamodel.ConnectorDefinition{}).Where("uid = ?", uid)
	if isBasicView {
		queryBuilder.Omit("spec")
	}
	if result := queryBuilder.First(&connectorDefinition); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.NotFound,
			"[db] get connector definition by uid error",
			"connector_definition",
			fmt.Sprintf("uid %s", uid.String()),
			"",
			result.Error.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return nil, st.Err()
	}
	return &connectorDefinition, nil
}

func (r *repository) CreateConnector(connector *datamodel.Connector) error {

	logger, _ := logger.GetZapLogger()

	if result := r.db.Model(&datamodel.Connector{}).Create(connector); result.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(result.Error, &pgErr) {
			if pgErr.Code == "23505" {
				st, err := sterr.CreateErrorResourceInfo(
					codes.AlreadyExists,
					"[db] create connector error",
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

func (r *repository) ListConnector(ownerPermalink string, connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) (connectors []*datamodel.Connector, totalSize int64, nextPageToken string, err error) {

	logger, _ := logger.GetZapLogger()

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
				"[db] list connector error",
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
			"[db] list connector error",
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
				"[db] list connector error",
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
				"[db] list connector error",
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

func (r *repository) GetConnectorByID(id string, ownerPermalink string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.Connector, error) {

	logger, _ := logger.GetZapLogger()

	var connector datamodel.Connector

	queryBuilder := r.db.Model(&datamodel.Connector{}).
		Where("id = ? AND owner = ? AND connector_type = ?", id, ownerPermalink, connectorType)

	if isBasicView {
		queryBuilder.Omit("configuration")
	}

	if result := queryBuilder.First(&connector); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.NotFound,
			"[db] get connector by id error",
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

func (r *repository) GetConnectorByUID(uid uuid.UUID, ownerPermalink string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.Connector, error) {

	logger, _ := logger.GetZapLogger()

	var connector datamodel.Connector

	queryBuilder := r.db.Model(&datamodel.Connector{}).
		Where("uid = ? AND owner = ? AND connector_type = ?", uid, ownerPermalink, connectorType)

	if isBasicView {
		queryBuilder.Omit("configuration")
	}

	if result := queryBuilder.First(&connector); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.NotFound,
			"[db] get connector by uid error",
			"connector",
			fmt.Sprintf("uid %s and connector_type %s", uid, connectorPB.ConnectorType(connectorType)),
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

func (r *repository) UpdateConnector(id string, ownerPermalink string, connectorType datamodel.ConnectorType, connector *datamodel.Connector) error {

	logger, _ := logger.GetZapLogger()

	if result := r.db.Model(&datamodel.Connector{}).
		Where("id = ? AND owner = ? AND connector_type = ?", id, ownerPermalink, connectorType).
		Updates(connector); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			"[db] update connector error",
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
			"[db] update connector error",
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

func (r *repository) DeleteConnector(id string, ownerPermalink string, connectorType datamodel.ConnectorType) error {

	logger, _ := logger.GetZapLogger()

	result := r.db.Model(&datamodel.Connector{}).
		Where("id = ? AND owner = ? AND connector_type = ?", id, ownerPermalink, connectorType).
		Delete(&datamodel.Connector{})

	if result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			"[db] delete connector error",
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
			"[db] delete connector error",
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

func (r *repository) UpdateConnectorID(id string, ownerPermalink string, connectorType datamodel.ConnectorType, newID string) error {

	logger, _ := logger.GetZapLogger()

	if result := r.db.Model(&datamodel.Connector{}).
		Where("id = ? AND owner = ? AND connector_type = ?", id, ownerPermalink, connectorType).
		Update("id", newID); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			"[db] update connector id error",
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
			"[db] update connector id error",
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

func (r *repository) UpdateConnectorStateByID(id string, ownerPermalink string, connectorType datamodel.ConnectorType, state datamodel.ConnectorState) error {

	logger, _ := logger.GetZapLogger()

	if result := r.db.Model(&datamodel.Connector{}).
		Where("id = ? AND owner = ? AND connector_type = ?", id, ownerPermalink, connectorType).
		Update("state", state); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			"[db] update connector state by id error",
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
			"[db] update connector state by id error",
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

func (r *repository) UpdateConnectorStateByUID(uid uuid.UUID, ownerPermalink string, connectorType datamodel.ConnectorType, state datamodel.ConnectorState) error {

	logger, _ := logger.GetZapLogger()

	if result := r.db.Model(&datamodel.Connector{}).
		Where("uid = ? AND owner = ? AND connector_type = ?", uid, ownerPermalink, connectorType).
		Update("state", state); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			"[db] update connector state by uid error",
			"connector",
			fmt.Sprintf("uid %s and connector_type %s", uid, connectorPB.ConnectorType(connectorType)),
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
			"[db] update connector state by uid error",
			"connector",
			fmt.Sprintf("uid %s and connector_type %s", uid, connectorPB.ConnectorType(connectorType)),
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
