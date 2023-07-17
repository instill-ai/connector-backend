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

// Connector is the data model of the connector table
type Connector struct {
	BaseDynamic
	ID                     string
	Owner                  string
	ConnectorDefinitionUID uuid.UUID
	Description            sql.NullString
	Tombstone              bool
	Configuration          datatypes.JSON      `gorm:"type:jsonb"`
	ConnectorType          ConnectorType       `sql:"type:valid_connector_type"`
	State                  ConnectorState      `sql:"type:valid_state_type"`
	Visibility             ConnectorVisibility `sql:"type:valid_visibility"`
	Task                   Task                `sql:"type:valid_task"`
}

// ConnectorType is an alias type for Protobuf enum ConnectorType
type ConnectorVisibility connectorPB.Connector_Visibility

// ConnectorType is an alias type for Protobuf enum ConnectorType
type ConnectorType connectorPB.ConnectorType

// ConnectorType is an alias type for Protobuf enum ConnectorType
type Task connectorPB.Task

// Scan function for custom GORM type ConnectorType
func (c *ConnectorType) Scan(value interface{}) error {
	*c = ConnectorType(connectorPB.ConnectorType_value[value.(string)])
	return nil
}

// Value function for custom GORM type ConnectorType
func (c ConnectorType) Value() (driver.Value, error) {
	return connectorPB.ConnectorType(c).String(), nil
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

// Scan function for custom GORM type ReleaseStage
func (r *ConnectorVisibility) Scan(value interface{}) error {
	*r = ConnectorVisibility(connectorPB.Connector_Visibility_value[value.(string)])
	return nil
}

// Value function for custom GORM type ReleaseStage
func (r ConnectorVisibility) Value() (driver.Value, error) {
	return connectorPB.Connector_Visibility(r).String(), nil
}

// Scan function for custom GORM type Task
func (r *Task) Scan(value interface{}) error {
	*r = Task(connectorPB.Task_value[value.(string)])
	return nil
}

// Value function for custom GORM type Task
func (r Task) Value() (driver.Value, error) {
	return connectorPB.Task(r).String(), nil
}
