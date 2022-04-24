package main

import (
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/gofrs/uuid"

	"github.com/instill-ai/connector-backend/internal/logger"
	"github.com/instill-ai/connector-backend/pkg/datamodel"

	connectorPB "github.com/instill-ai/protogen-go/connector/v1alpha"
)

func createDestinationConnectorDefinition(db *gorm.DB, dstDef *connectorPB.DestinationDefinition, spec datatypes.JSON) error {
	logger, _ := logger.GetZapLogger()

	id, err := uuid.FromString(dstDef.GetDestinationDefinitionId())
	if err != nil {
		return err
	}

	releaseDate := func() *time.Time {
		releaseDate := dstDef.GetReleaseDate()
		if releaseDate != nil {
			t := time.Date(int(releaseDate.Year), time.Month(releaseDate.Month), int(releaseDate.Day), 0, 0, 0, 0, time.UTC)
			return &t
		}
		return nil
	}()

	resourceRequirements := func() datatypes.JSON {
		s := dstDef.GetResourceRequirements()
		if s != nil {
			if b, err := s.MarshalJSON(); err != nil {
				logger.Fatal(err.Error())
			} else {
				return b
			}
		}
		return []byte("{}")
	}()

	connectorType := datamodel.ConnectorTypeDestination

	connectionType := func() *datamodel.ValidConnectionType {
		destinationType := dstDef.GetDestinationType().String()
		e := datamodel.ValidConnectionType(strings.ReplaceAll(destinationType, "DESTINATION_TYPE", "CONNECTION_TYPE"))
		return &e
	}()

	releaseStage := func() *datamodel.ValidReleaseStage {
		releaseStage := dstDef.GetReleaseStage().String()
		e := datamodel.ValidReleaseStage(releaseStage)
		return &e
	}()

	if err := createConnectorDefinitionRecord(
		db,
		dstDef.GetName(),
		id,
		dstDef.GetDockerRepository(),
		dstDef.GetDockerImageTag(),
		dstDef.GetDocumentationUrl(),
		dstDef.GetIcon(),
		dstDef.GetTombstone(),
		true, //dstDef.GetPublic(),
		dstDef.GetCustom(),
		releaseDate,
		spec,
		resourceRequirements,
		connectorType,
		connectionType,
		releaseStage,
	); err != nil {
		return err
	}

	return nil
}
