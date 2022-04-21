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
	name string,
	id uuid.UUID,
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
	connectorType datamodel.ValidConectorType,
	connectionType *datamodel.ValidConnectionType,
	releaseStage *datamodel.ValidReleaseStage) error {

	connectorDef := datamodel.ConnectorDefinition{
		BaseStatic:           datamodel.BaseStatic{ID: id},
		Name:                 name,
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
		ConnectionType:       connectionType,
		ReleaseStage:         releaseStage,
	}

	if result := db.Model(&datamodel.ConnectorDefinition{}).Create(&connectorDef); result.Error != nil {
		return result.Error
	}

	return nil
}

func createDirectnessConnector(
	db *gorm.DB,
	workspaceID uuid.UUID,
	connectorDefinitionID uuid.UUID,
	name string,
	tombstone bool,
	configuration datatypes.JSON,
	connectorType datamodel.ValidConectorType) error {

	directnessConnector := datamodel.Connector{
		WorkspaceID:           workspaceID,
		ConnectorDefinitionID: connectorDefinitionID,
		Name:                  name,
		Tombstone:             tombstone,
		Configuration:         configuration,
		ConnectorType:         connectorType,
	}

	if result := db.Model(&datamodel.Connector{}).Create(&directnessConnector); result.Error != nil {
		return result.Error
	}

	return nil
}
