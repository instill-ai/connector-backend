package datamodel

import (
	"database/sql"
	"database/sql/driver"
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

// BaseStatic contains common columns for all tables with static UUID as primary key
type BaseStatic struct {
	UID        uuid.UUID      `gorm:"type:uuid;primary_key;<-:create"` // allow read and create
	CreateTime time.Time      `gorm:"autoCreateTime:nano"`
	UpdateTime time.Time      `gorm:"autoUpdateTime:nano"`
	DeleteTime gorm.DeletedAt `sql:"index"`
}

// BaseDynamic contains common columns for all tables with dynamic UUID as primary key generated when creating
type BaseDynamic struct {
	UID        uuid.UUID      `gorm:"type:uuid;primary_key;<-:create"` // allow read and create
	CreateTime time.Time      `gorm:"autoCreateTime:nano"`
	UpdateTime time.Time      `gorm:"autoUpdateTime:nano"`
	DeleteTime gorm.DeletedAt `sql:"index"`
}

// BeforeCreate will set a UUID rather than numeric ID.
func (base *BaseDynamic) BeforeCreate(db *gorm.DB) error {
	uuid, err := uuid.NewV4()
	if err != nil {
		return err
	}
	db.Statement.SetColumn("UID", uuid)
	return nil
}

// ConnectorDefinition is the data model of the connector_definition table
type ConnectorDefinition struct {
	BaseStatic
	ID                   string
	Title                string
	DockerRepository     string
	DockerImageTag       string
	DocumentationURL     string
	Icon                 string
	Tombstone            bool
	Public               bool
	Custom               bool
	ReleaseDate          *time.Time
	Spec                 datatypes.JSON `gorm:"type:jsonb"`
	ResourceRequirements datatypes.JSON `gorm:"type:jsonb"`
	ConnectorType        ConnectorType  `sql:"type:valid_connector_type"`
	ReleaseStage         ReleaseStage   `sql:"type:valid_release_stage"`
}

// Connector is the data model of the connector table
type Connector struct {
	BaseDynamic
	ID                     string
	Owner                  string
	ConnectorDefinitionUID uuid.UUID
	Description            sql.NullString
	Tombstone              bool
	Configuration          datatypes.JSON `gorm:"type:jsonb"`
	ConnectorType          ConnectorType  `sql:"type:valid_connector_type"`
	State                  ConnectorState `sql:"type:valid_state_type"`
}

// ConnectorType is an alias type for Protobuf enum ConnectorType
type ConnectorType connectorPB.ConnectorType

// Scan function for custom GORM type ConnectorType
func (c *ConnectorType) Scan(value interface{}) error {
	*c = ConnectorType(connectorPB.ConnectorType_value[value.(string)])
	return nil
}

// Value function for custom GORM type ConnectorType
func (c ConnectorType) Value() (driver.Value, error) {
	return connectorPB.ConnectorType(c).String(), nil
}

// ReleaseStage is an alias type for Protobuf enum ReleaseStage
type ReleaseStage connectorPB.ReleaseStage

// Scan function for custom GORM type ReleaseStage
func (r *ReleaseStage) Scan(value interface{}) error {
	*r = ReleaseStage(connectorPB.ReleaseStage_value[value.(string)])
	return nil
}

// Value function for custom GORM type ReleaseStage
func (r ReleaseStage) Value() (driver.Value, error) {
	return connectorPB.ReleaseStage(r).String(), nil
}

// ConnectorState is an alias type for Protobuf enum ConnectorState
type ConnectorState connectorPB.Connector_State

// Scan function for custom GORM type ConnectorState
func (r *ConnectorState) Scan(value interface{}) error {
	*r = ConnectorState(connectorPB.Connector_State_value[value.(string)])
	return nil
}

// Value function for custom GORM type ConnectorState
func (r ConnectorState) Value() (driver.Value, error) {
	return connectorPB.Connector_State(r).String(), nil
}
