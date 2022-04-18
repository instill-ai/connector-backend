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

func createDestinationConnectorDefinition(db *gorm.DB, dstDef *connectorPB.DestinationDefinition, spec datatypes.JSON) error {
	logger, _ := logger.GetZapLogger()

	ID, err := uuid.FromString(dstDef.GetDestinationDefinitionId())
	if err != nil {
		return err
	}

	releaseDate := func(releaseDate *date.Date) *time.Time {
		if releaseDate != nil {
			t := time.Date(int(releaseDate.Year), time.Month(releaseDate.Month), int(releaseDate.Day), 0, 0, 0, 0, time.UTC)
			return &t
		}
		return nil
	}(dstDef.GetReleaseDate())

	resourceRequirements := func(s *structpb.Struct) datatypes.JSON {
		if s != nil {
			if b, err := s.MarshalJSON(); err != nil {
				logger.Fatal(err.Error())
			} else {
				return b
			}
		}
		return []byte("{}")
	}(dstDef.GetResourceRequirements())

	connectorType := datamodel.ConnectorTypeDestination

	connectionType := func(enum string) *datamodel.ValidConnectionType {
		if enum == "DESTINATION_TYPE_UNSPECIFIED" {
			return nil
		}
		e := datamodel.ValidConnectionType(strings.ReplaceAll(enum, "DESTINATION_TYPE", "CONNECTION_TYPE"))
		return &e
	}(dstDef.GetDestinationType().String())

	releaseStage := func(enum string) *datamodel.ValidReleaseStage {
		if enum == "RELEASE_STAGE_UNSPECIFIED" {
			return nil
		}
		e := datamodel.ValidReleaseStage(enum)
		return &e
	}(dstDef.GetReleaseStage().String())

	if err := createConnectorDefinitionRecord(
		db,
		dstDef.GetName(),
		ID,
		dstDef.GetDockerRepository(),
		dstDef.GetDockerImageTag(),
		dstDef.GetDocumentationUrl(),
		dstDef.GetIcon(),
		dstDef.GetTombstone(),
		dstDef.GetPublic(),
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
