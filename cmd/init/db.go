package main

import (
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/instill-ai/connector-backend/pkg/datamodel"
)

func createConnectorDefinitionRecord(
	db *gorm.DB,
	Name string,
	ID uuid.UUID,
	DockerRepository string,
	DockerImageTag string,
	DocumentationURL string,
	Icon string,
	Tombstone bool,
	Public bool,
	Custom bool,
	ReleaseDate *time.Time,
	Spec datatypes.JSON,
	ResourceRequirements datatypes.JSON,
	ConnectorType datamodel.ValidConectorType,
	ConnectionType *datamodel.ValidConnectionType,
	ReleaseStage *datamodel.ValidReleaseStage) error {

	connectorDef := datamodel.ConnectorDefinition{
		Name:                 Name,
		DockerRepository:     DockerRepository,
		DockerImageTag:       DockerImageTag,
		DocumentationURL:     DocumentationURL,
		Icon:                 Icon,
		Spec:                 Spec,
		Tombstone:            Tombstone,
		ResourceRequirements: ResourceRequirements,
		Public:               true, // Public field is not used in definition yaml. Set it to true by default now.
		Custom:               Custom,
		ConnectorType:        ConnectorType,
		BaseStatic:           datamodel.BaseStatic{ID: ID},
		ReleaseDate:          ReleaseDate,
		ConnectionType:       ConnectionType,
		ReleaseStage:         ReleaseStage,
	}

	if result := db.Model(&datamodel.ConnectorDefinition{}).Create(&connectorDef); result.Error != nil {
		return result.Error
	}

	return nil
}
