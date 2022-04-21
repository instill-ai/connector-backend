package main

import (
	"strings"
	"time"

	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/protobuf/types/known/structpb"
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

	releaseDate := func(releaseDate *date.Date) *time.Time {
		if releaseDate != nil {
			t := time.Date(int(releaseDate.Year), time.Month(releaseDate.Month), int(releaseDate.Day), 0, 0, 0, 0, time.UTC)
			return &t
		}
		return nil
	}(srcDef.GetReleaseDate())

	resourceRequirements := func(s *structpb.Struct) datatypes.JSON {
		if s != nil {
			if b, err := s.MarshalJSON(); err != nil {
				logger.Fatal(err.Error())
			} else {
				return b
			}
		}
		return []byte("{}")
	}(srcDef.GetResourceRequirements())

	connectorType := datamodel.ConnectorTypeSource

	connectionType := func(enum string) *datamodel.ValidConnectionType {
		if enum == "SOURCE_TYPE_UNSPECIFIED" {
			return nil
		}
		e := datamodel.ValidConnectionType(strings.ReplaceAll(enum, "SOURCE_TYPE", "CONNECTION_TYPE"))
		return &e
	}(srcDef.GetSourceType().String())

	releaseStage := func(enum string) *datamodel.ValidReleaseStage {
		if enum == "RELEASE_STAGE_UNSPECIFIED" {
			return nil
		}
		e := datamodel.ValidReleaseStage(enum)
		return &e
	}(srcDef.GetReleaseStage().String())

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
		connectorType,
		connectionType,
		releaseStage,
	); err != nil {
		return err
	}

	return nil
}
