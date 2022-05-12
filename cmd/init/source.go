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

func createSourceConnectorDefinition(db *gorm.DB, srcConnDef *connectorPB.SourceConnectorDefinition, connDef *connectorPB.ConnectorDefinition, spec datatypes.JSON) error {
	logger, _ := logger.GetZapLogger()

	uid, err := uuid.FromString(srcConnDef.GetUid())
	if err != nil {
		return err
	}

	id := srcConnDef.GetId()
	if id == "" {
		id = connDef.GetDockerRepository()[strings.LastIndex(connDef.GetDockerRepository(), "/")+1:]
		if id == "" {
			// Only directness connector ends up this
			id = strings.ToLower(connDef.GetTitle())
		}
	}

	releaseDate := func() *time.Time {
		releaseDate := connDef.GetReleaseDate()
		if releaseDate != nil {
			t := time.Date(int(releaseDate.Year), time.Month(releaseDate.Month), int(releaseDate.Day), 0, 0, 0, 0, time.UTC)
			return &t
		}
		return nil
	}()

	resourceRequirements := func() datatypes.JSON {
		s := connDef.GetResourceRequirements()
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
		uid,
		id,
		connDef.GetTitle(),
		connDef.GetDockerRepository(),
		connDef.GetDockerImageTag(),
		connDef.GetDocumentationUrl(),
		connDef.GetIcon(),
		connDef.GetTombstone(),
		true, //srcDef.GetPublic(),
		connDef.GetCustom(),
		releaseDate,
		spec,
		resourceRequirements,
		datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE),
		datamodel.ConnectionType(connDef.GetConnectionType()),
		datamodel.ReleaseStage(connDef.GetReleaseStage()),
	); err != nil {
		return err
	}

	return nil
}
