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

// ConnectorDefinition is the data model of the connector_definition table
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
	ConnectorType        ValidConnectorType   `sql:"type:valid_connector_type"`
	ConnectionType       *ValidConnectionType `sql:"type:valid_connection_type"`
	ReleaseStage         *ValidReleaseStage   `sql:"type:valid_release_stage"`
}

// Connector is the data model of the connector table
type Connector struct {
	BaseDynamic
	OwnerID               uuid.UUID
	ConnectorDefinitionID uuid.UUID
	Name                  string
	Tombstone             bool
	Configuration         datatypes.JSON     `gorm:"type:jsonb"`
	ConnectorType         ValidConnectorType `sql:"type:valid_connector_type"`

	// Output-only field
	FullName string `gorm:"-"`
}

// ValidConnectorType enumerates the type of connector
type ValidConnectorType string

const (
	// ConnectorTypeUnspecified represents a null connector
	ConnectorTypeUnspecified ValidConnectorType = "CONNECTOR_TYPE_UNSPECIFIED"
	// ConnectorTypeSource represents a source connector
	ConnectorTypeSource ValidConnectorType = "CONNECTOR_TYPE_SOURCE"
	// ConnectorTypeDestination represents a destination connector
	ConnectorTypeDestination ValidConnectorType = "CONNECTOR_TYPE_DESTINATION"
)

// Scan function for custom GORM type ValidConnectorType
func (p *ValidConnectorType) Scan(value interface{}) error {
	switch v := value.(type) {
	case string:
		*p = ValidConnectorType(v)
	case []byte:
		*p = ValidConnectorType(v)
	default:
		return errors.New("Incompatible type for ValidConecctorType")
	}
	return nil
}

// Value function for custom GORM type ValidConnectorType
func (p ValidConnectorType) Value() (driver.Value, error) {
	return string(p), nil
}

// ValidConnectionType enumerates the type of connection
type ValidConnectionType string

const (
	// ConnectionTypeUnspecified represents a null connection
	ConnectionTypeUnspecified ValidConnectionType = "CONNECTION_TYPE_UNSPECIFIED"
	// ConnectionTypeDirectness represents directness connection
	ConnectionTypeDirectness ValidConnectionType = "CONNECTION_TYPE_DIRECTNESS"
	// ConnectionTypeFile represents file connection
	ConnectionTypeFile ValidConnectionType = "CONNECTION_TYPE_FILE"
	// ConnectionTypeAPI represents API connection
	ConnectionTypeAPI ValidConnectionType = "CONNECTION_TYPE_API"
	// ConnectionTypeDatabase represents database connection
	ConnectionTypeDatabase ValidConnectionType = "CONNECTION_TYPE_DATABASE"
	// ConnectionTypeCustom represents custom connection
	ConnectionTypeCustom ValidConnectionType = "CONNECTION_TYPE_CUSTOM"
)

// Scan function for custom GORM type ValidConnectionType
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

// Value function for custom GORM type ValidConnectionType
func (p ValidConnectionType) Value() (driver.Value, error) {
	return string(p), nil
}

// ValidReleaseStage enumerates release stages
type ValidReleaseStage string

const (
	// ReleaseStageUnspecified represents a null release stage
	ReleaseStageUnspecified ValidReleaseStage = "RELEASE_STAGE_UNSPECIFIED"
	// ReleaseStageAlpha represents release stage alpha
	ReleaseStageAlpha ValidReleaseStage = "RELEASE_STAGE_ALPHA"
	// ReleaseStageBeta represents release stage beta
	ReleaseStageBeta ValidReleaseStage = "RELEASE_STAGE_BETA"
	// ReleaseStageGenerallyAvailable represents release stage general available
	ReleaseStageGenerallyAvailable ValidReleaseStage = "RELEASE_STAGE_GENERALLY_AVAILABLE"
	// ReleaseStageCustom represents release stage custom
	ReleaseStageCustom ValidReleaseStage = "RELEASE_STAGE_CUSTOM"
)

// Scan function for custom GORM type ValidReleaseStage
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

// Value function for custom GORM type ValidReleaseStage
func (p ValidReleaseStage) Value() (driver.Value, error) {
	return string(p), nil
}
