package datamodel

import (
	"database/sql/driver"
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// BaseStatic contains common columns for all tables with static UUID as primary key
type BaseStatic struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
}

// BaseDynamic contains common columns for all tables with dynamic UUID as primary key generated when creating
type BaseDynamic struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
}

// BeforeCreate will set a UUID rather than numeric ID.
func (base *BaseDynamic) BeforeCreate(db *gorm.DB) error {
	if uuid, err := uuid.NewV4(); err != nil {
		return err
	} else {
		db.Statement.SetColumn("ID", uuid)
	}
	return nil
}

// ConnectorDefinition is the data model of the connection_definition table
type ConnectorDefinition struct {
	BaseStatic
	Name                 string
	DockerRepository     string
	DockerImageTag       string
	DocumentationURL     string
	Icon                 string
	Tombstone            bool
	Public               bool
	Custom               bool
	ReleaseDate          *time.Time
	Spec                 datatypes.JSON       `gorm:"type:jsonb"`
	ResourceRequirements datatypes.JSON       `gorm:"type:jsonb"`
	ConnectorType        ValidConectorType    `sql:"type:valid_connector_type"`
	ConnectionType       *ValidConnectionType `sql:"type:valid_connection_type"`
	ReleaseStage         *ValidReleaseStage   `sql:"type:valid_release_stage"`
}

type ValidConectorType string

const (
	ConnectorTypeSource      ValidConectorType = "CONNECTOR_TYPE_SOURCE"
	ConnectorTypeDestination ValidConectorType = "CONNECTOR_TYPE_DESTINATION"
)

func (p *ValidConectorType) Scan(value interface{}) error {
	switch v := value.(type) {
	case string:
		*p = ValidConectorType(v)
	case []byte:
		*p = ValidConectorType(v)
	default:
		return errors.New("Incompatible type for ValidConecctorType")
	}
	return nil
}

func (p ValidConectorType) Value() (driver.Value, error) {
	return string(p), nil
}

type ValidConnectionType string

const (
	ConnectionTypeDirectness ValidConnectionType = "CONNECTION_TYPE_DIRECTNESS"
	ConnectionTypeFile       ValidConnectionType = "CONNECTION_TYPE_FILE"
	ConnectionTypeAPI        ValidConnectionType = "CONNECTION_TYPE_API"
	ConnectionTypeDatabase   ValidConnectionType = "CONNECTION_TYPE_DATABASE"
	ConnectionTypeCustom     ValidConnectionType = "CONNECTION_TYPE_CUSTOM"
)

func (p *ValidConnectionType) Scan(value interface{}) error {
	switch v := value.(type) {
	case string:
		*p = ValidConnectionType(v)
	case []byte:
		*p = ValidConnectionType(v)
	default:
		return errors.New("Incompatible type for ValidSourceType")
	}
	return nil
}

func (p ValidConnectionType) Value() (driver.Value, error) {
	return string(p), nil
}

type ValidReleaseStage string

const (
	ReleaseStageAlpha              ValidReleaseStage = "RELEASE_STAGE_ALPHA"
	ReleaseStageBeta               ValidReleaseStage = "RELEASE_STAGE_BETA"
	ReleaseStageGenerallyAvailable ValidReleaseStage = "RELEASE_STAGE_GENERALLY_AVAILABLE"
	ReleaseStageCustom             ValidReleaseStage = "RELEASE_STAGE_CUSTOM"
)

func (p *ValidReleaseStage) Scan(value interface{}) error {
	switch v := value.(type) {
	case string:
		*p = ValidReleaseStage(v)
	case []byte:
		*p = ValidReleaseStage(v)
	default:
		return errors.New("Incompatible type for ValidReleaseStage")
	}
	return nil
}

func (p ValidReleaseStage) Value() (driver.Value, error) {
	return string(p), nil
}
