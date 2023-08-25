package datamodel

import (
	"database/sql"
	"database/sql/driver"
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	taskPB "github.com/instill-ai/protogen-go/common/task/v1alpha"
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

// ConnectorResource is the data model of the connector table
type ConnectorResource struct {
	BaseDynamic
	ID                     string
	Owner                  string
	ConnectorDefinitionUID uuid.UUID
	Description            sql.NullString
	Tombstone              bool
	Configuration          datatypes.JSON              `gorm:"type:jsonb"`
	ConnectorType          ConnectorResourceType       `sql:"type:valid_connector_type"`
	State                  ConnectorResourceState      `sql:"type:valid_state_type"`
	Visibility             ConnectorResourceVisibility `sql:"type:valid_visibility"`
}

func (ConnectorResource) TableName() string {
	return "connector"
}

// ConnectorResourceType is an alias type for Protobuf enum ConnectorType
type ConnectorResourceVisibility connectorPB.ConnectorResource_Visibility

// ConnectorResourceType is an alias type for Protobuf enum ConnectorType
type ConnectorResourceType connectorPB.ConnectorType

// ConnectorType is an alias type for Protobuf enum ConnectorType
type Task taskPB.Task

// Scan function for custom GORM type ConnectorType
func (c *ConnectorResourceType) Scan(value interface{}) error {
	*c = ConnectorResourceType(connectorPB.ConnectorType_value[value.(string)])
	return nil
}

// Value function for custom GORM type ConnectorType
func (c ConnectorResourceType) Value() (driver.Value, error) {
	return connectorPB.ConnectorType(c).String(), nil
}

// ConnectorResourceState is an alias type for Protobuf enum ConnectorState
type ConnectorResourceState connectorPB.ConnectorResource_State

// Scan function for custom GORM type ConnectorState
func (r *ConnectorResourceState) Scan(value interface{}) error {
	*r = ConnectorResourceState(connectorPB.ConnectorResource_State_value[value.(string)])
	return nil
}

// Value function for custom GORM type ConnectorState
func (r ConnectorResourceState) Value() (driver.Value, error) {
	return connectorPB.ConnectorResource_State(r).String(), nil
}

// Scan function for custom GORM type ReleaseStage
func (r *ConnectorResourceVisibility) Scan(value interface{}) error {
	*r = ConnectorResourceVisibility(connectorPB.ConnectorResource_Visibility_value[value.(string)])
	return nil
}

// Value function for custom GORM type ReleaseStage
func (r ConnectorResourceVisibility) Value() (driver.Value, error) {
	return connectorPB.ConnectorResource_Visibility(r).String(), nil
}

// Scan function for custom GORM type Task
func (r *Task) Scan(value interface{}) error {
	*r = Task(taskPB.Task_value[value.(string)])
	return nil
}

// Value function for custom GORM type Task
func (r Task) Value() (driver.Value, error) {
	return taskPB.Task(r).String(), nil
}
