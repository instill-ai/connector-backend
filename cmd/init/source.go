package main

import (
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/gofrs/uuid"

	"github.com/instill-ai/connector-backend/internal/logger"
	"github.com/instill-ai/connector-backend/pkg/datamodel"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

func createSourceConnectorDefinition(db *gorm.DB, srcConnDef *connectorPB.SourceConnectorDefinition, connDef *connectorPB.ConnectorDefinition, spec datatypes.JSON) error {
	logger, _ := logger.GetZapLogger()

	uid, err := uuid.FromString(srcConnDef.GetUid())
	if err != nil {
		return err
	}

	if srcConnDef.GetId() == "" {
		srcConnDef.Id = connDef.GetDockerRepository()[strings.LastIndex(connDef.GetDockerRepository(), "/")+1:]
	}

	srcConnDef.ConnectorDefinition = connDef
	srcConnDef.GetConnectorDefinition().CreateTime = &timestamppb.Timestamp{}
	srcConnDef.GetConnectorDefinition().UpdateTime = &timestamppb.Timestamp{}
	srcConnDef.GetConnectorDefinition().Tombstone = false
	srcConnDef.GetConnectorDefinition().Public = true
	srcConnDef.GetConnectorDefinition().Custom = false

	srcConnDef.GetConnectorDefinition().Spec = &connectorPB.Spec{}
	if err := protojson.Unmarshal(spec, srcConnDef.GetConnectorDefinition().Spec); err != nil {
		logger.Fatal(err.Error())
	}

	if srcConnDef.GetConnectorDefinition().GetResourceRequirements() == nil {
		srcConnDef.GetConnectorDefinition().ResourceRequirements = &structpb.Struct{}
	}

	// Validate JSON Schema before inserting into db
	if err := datamodel.ValidateJSONSchema(datamodel.SrcConnDefJSONSchema, srcConnDef, true); err != nil {
		return err
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
		srcConnDef.GetId(),
		srcConnDef.GetConnectorDefinition().GetTitle(),
		srcConnDef.GetConnectorDefinition().GetDockerRepository(),
		srcConnDef.GetConnectorDefinition().GetDockerImageTag(),
		srcConnDef.GetConnectorDefinition().GetDocumentationUrl(),
		srcConnDef.GetConnectorDefinition().GetIcon(),
		srcConnDef.GetConnectorDefinition().GetTombstone(),
		srcConnDef.GetConnectorDefinition().GetPublic(),
		srcConnDef.GetConnectorDefinition().GetCustom(),
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
