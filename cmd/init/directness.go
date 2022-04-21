package main

import (
	"github.com/gofrs/uuid"
	"github.com/instill-ai/connector-backend/internal/util"
	"github.com/instill-ai/connector-backend/pkg/datamodel"
	connectorPB "github.com/instill-ai/protogen-go/connector/v1alpha"
	"gorm.io/gorm"
)

func createSourceDirectnessConnector(db *gorm.DB, srcDef *connectorPB.SourceDefinition) error {

	directnessConnectorWorksapceID, err := uuid.FromString(util.DirectnessConnectorWorksapceID)
	if err != nil {
		return err
	}

	sourceDefinitionID, err := uuid.FromString(srcDef.GetSourceDefinitionId())
	if err != nil {
		return err
	}

	if err := createDirectnessConnector(
		db,
		directnessConnectorWorksapceID,
		sourceDefinitionID,
		srcDef.GetName(),
		srcDef.GetTombstone(),
		[]byte("{}"),
		datamodel.ConnectorTypeSource,
	); err != nil {
		return err
	}

	return nil
}

func createDestinationDirectnessConnector(db *gorm.DB, dstDef *connectorPB.DestinationDefinition) error {

	directnessConnectorWorksapceID, err := uuid.FromString(util.DirectnessConnectorWorksapceID)
	if err != nil {
		return err
	}

	destinationDefinitionID, err := uuid.FromString(dstDef.GetDestinationDefinitionId())
	if err != nil {
		return err
	}

	if err := createDirectnessConnector(
		db,
		directnessConnectorWorksapceID,
		destinationDefinitionID,
		dstDef.GetName(),
		dstDef.GetTombstone(),
		[]byte("{}"),
		datamodel.ConnectorTypeDestination,
	); err != nil {
		return err
	}

	return nil
}
