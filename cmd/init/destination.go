package main

import (
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/connector-backend/pkg/logger"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

func createDestinationConnectorDefinition(db *gorm.DB, dstConnDef *connectorPB.DestinationConnectorDefinition, connDef *connectorPB.ConnectorDefinition, spec datatypes.JSON) error {
	logger, _ := logger.GetZapLogger()

	uid, err := uuid.FromString(dstConnDef.GetUid())
	if err != nil {
		return err
	}

	if dstConnDef.GetId() == "" {
		dstConnDef.Id = connDef.GetDockerRepository()[strings.LastIndex(connDef.GetDockerRepository(), "/")+1:]
	}

	dstConnDef.ConnectorDefinition = connDef
	dstConnDef.GetConnectorDefinition().CreateTime = &timestamppb.Timestamp{}
	dstConnDef.GetConnectorDefinition().UpdateTime = &timestamppb.Timestamp{}
	dstConnDef.GetConnectorDefinition().Tombstone = false
	dstConnDef.GetConnectorDefinition().Public = true
	dstConnDef.GetConnectorDefinition().Custom = false

	dstConnDef.GetConnectorDefinition().Spec = &connectorPB.Spec{}
	if err := protojson.Unmarshal(spec, dstConnDef.GetConnectorDefinition().Spec); err != nil {
		logger.Fatal(err.Error())
	}

	if dstConnDef.GetConnectorDefinition().GetResourceRequirements() == nil {
		dstConnDef.GetConnectorDefinition().ResourceRequirements = &structpb.Struct{}
	}

	// Validate JSON Schema before inserting into db
	if err := datamodel.ValidateJSONSchema(datamodel.DstConnDefJSONSchema, dstConnDef, true); err != nil {
		return err
	}

	releaseDate := func() *time.Time {
		releaseDate := dstConnDef.GetConnectorDefinition().GetReleaseDate()
		if releaseDate != nil {
			t := time.Date(int(releaseDate.Year), time.Month(releaseDate.Month), int(releaseDate.Day), 0, 0, 0, 0, time.UTC)
			return &t
		}
		return nil
	}()

	resourceRequirements := func() datatypes.JSON {
		s := dstConnDef.GetConnectorDefinition().GetResourceRequirements()
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
		dstConnDef.GetId(),
		dstConnDef.GetConnectorDefinition().GetTitle(),
		dstConnDef.GetConnectorDefinition().GetDockerRepository(),
		dstConnDef.GetConnectorDefinition().GetDockerImageTag(),
		dstConnDef.GetConnectorDefinition().GetDocumentationUrl(),
		dstConnDef.GetConnectorDefinition().GetIcon(),
		dstConnDef.GetConnectorDefinition().GetTombstone(),
		dstConnDef.GetConnectorDefinition().GetPublic(),
		dstConnDef.GetConnectorDefinition().GetCustom(),
		releaseDate,
		spec,
		resourceRequirements,
		datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION),
		datamodel.ReleaseStage(dstConnDef.GetConnectorDefinition().GetReleaseStage()),
	); err != nil {
		return err
	}

	return nil
}
