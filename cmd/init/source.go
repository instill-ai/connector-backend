package main

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/gofrs/uuid"

	"github.com/instill-ai/connector-backend/internal/logger"
	"github.com/instill-ai/connector-backend/pkg/datamodel"

	connectorPB "github.com/instill-ai/protogen-go/connector/v1alpha"
)

func createSourceConnectorDefinition(db *gorm.DB, srcDef *connectorPB.SourceDefinition, spec datatypes.JSON) error {
	logger, _ := logger.GetZapLogger()

	id, err := uuid.FromString(srcDef.GetSourceDefinitionId())
	if err != nil {
		return err
	}

	releaseDate := func() *time.Time {
		releaseDate := srcDef.GetReleaseDate()
		if releaseDate != nil {
			t := time.Date(int(releaseDate.Year), time.Month(releaseDate.Month), int(releaseDate.Day), 0, 0, 0, 0, time.UTC)
			return &t
		}
		return nil
	}()

	resourceRequirements := func() datatypes.JSON {
		s := srcDef.GetResourceRequirements()
		if s != nil {
			if b, err := s.MarshalJSON(); err != nil {
				logger.Fatal(err.Error())
			} else {
				return b
			}
		}
		return []byte("{}")
	}()

	if err := createConnectorDefinitionRecord(
		db,
		srcDef.GetName(),
		id,
		srcDef.GetDockerRepository(),
		srcDef.GetDockerImageTag(),
		srcDef.GetDocumentationUrl(),
		srcDef.GetIcon(),
		srcDef.GetTombstone(),
		true, //srcDef.GetPublic(),
		srcDef.GetCustom(),
		releaseDate,
		spec,
		resourceRequirements,
		datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE),
		datamodel.ConnectionType(srcDef.GetConnectionType()),
		datamodel.ReleaseStage(srcDef.GetReleaseStage()),
	); err != nil {
		return err
	}

	return nil
}
