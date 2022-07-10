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
	uid uuid.UUID,
	id string,
	title string,
	dockerRepository string,
	dockerImageTag string,
	documentationURL string,
	icon string,
	tombstone bool,
	public bool,
	custom bool,
	releaseDate *time.Time,
	spec datatypes.JSON,
	resourceRequirements datatypes.JSON,
	connectorType datamodel.ConnectorType,
	releaseStage datamodel.ReleaseStage) error {

	connectorDef := datamodel.ConnectorDefinition{
		BaseStatic:           datamodel.BaseStatic{UID: uid},
		ID:                   id,
		Title:                title,
		DockerRepository:     dockerRepository,
		DockerImageTag:       dockerImageTag,
		DocumentationURL:     documentationURL,
		Icon:                 icon,
		Spec:                 spec,
		Tombstone:            tombstone,
		ResourceRequirements: resourceRequirements,
		Public:               public, // Public field is not used in definition yaml. Set it to true by default now.
		Custom:               custom,
		ConnectorType:        connectorType,
		ReleaseDate:          releaseDate,
		ReleaseStage:         releaseStage,
	}

	if result := db.Model(&datamodel.ConnectorDefinition{}).Create(&connectorDef); result.Error != nil {
		return result.Error
	}

	return nil
}
