package handler

import (
	"strings"

	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/instill-ai/connector-backend/internal/logger"
	"github.com/instill-ai/connector-backend/pkg/datamodel"

	connectorPB "github.com/instill-ai/protogen-go/connector/v1alpha"
)

func convertDBSourceDefinitionToPBSourceDefinition(dbSrcDef *datamodel.ConnectorDefinition) *connectorPB.SourceDefinition {
	logger, _ := logger.GetZapLogger()

	return &connectorPB.SourceDefinition{
		SourceDefinitionId: dbSrcDef.ID.String(),
		Name:               dbSrcDef.Name,
		DockerRepository:   dbSrcDef.DockerRepository,
		DockerImageTag:     dbSrcDef.DockerImageTag,
		DocumentationUrl:   dbSrcDef.DocumentationURL,
		Icon:               dbSrcDef.Icon,
		Tombstone:          dbSrcDef.Tombstone,
		Public:             dbSrcDef.Public,
		Custom:             dbSrcDef.Custom,
		CreatedAt:          timestamppb.New(dbSrcDef.CreatedAt),
		UpdateAt:           timestamppb.New(dbSrcDef.UpdatedAt),

		SourceType: func() connectorPB.SourceDefinition_SourceType {
			if dbSrcDef.ConnectionType != nil {
				return connectorPB.SourceDefinition_SourceType(connectorPB.SourceDefinition_SourceType_value[strings.ReplaceAll(string(*dbSrcDef.ConnectionType), "CONNECTION_TYPE", "SOURCE_TYPE")])
			}
			return connectorPB.SourceDefinition_SourceType(connectorPB.SourceDefinition_SourceType_value["SOURCE_TYPE_UNSPECIFIED"])
		}(),

		ReleaseStage: func() connectorPB.ReleaseStage {
			if dbSrcDef.ReleaseStage != nil {
				return connectorPB.ReleaseStage(connectorPB.ReleaseStage_value[string(*dbSrcDef.ReleaseStage)])
			}
			return connectorPB.ReleaseStage(connectorPB.SourceDefinition_SourceType_value["RELEASE_STAGE_UNSPECIFIED"])
		}(),

		ReleaseDate: func() *date.Date {
			if dbSrcDef.ReleaseDate != nil {
				return &date.Date{
					Year:  int32(dbSrcDef.ReleaseDate.Year()),
					Month: int32(dbSrcDef.ReleaseDate.Month()),
					Day:   int32(dbSrcDef.ReleaseDate.Day()),
				}
			}
			return &date.Date{}
		}(),

		Spec: func() *connectorPB.Spec {
			spec := connectorPB.Spec{}
			if err := protojson.Unmarshal(dbSrcDef.Spec, &spec); err != nil {
				logger.Fatal(err.Error())
				return nil
			}
			return &spec
		}(),

		ResourceRequirements: func() *structpb.Struct {
			s := structpb.Struct{}
			if err := protojson.Unmarshal(dbSrcDef.ResourceRequirements, &s); err != nil {
				logger.Fatal(err.Error())
				return nil
			}
			return &s
		}(),
	}
}

func convertDBDestinationDefinitionToPBDestinationDefinition(dbDstDef *datamodel.ConnectorDefinition) *connectorPB.DestinationDefinition {
	logger, _ := logger.GetZapLogger()

	return &connectorPB.DestinationDefinition{
		DestinationDefinitionId: dbDstDef.ID.String(),
		Name:                    dbDstDef.Name,
		DockerRepository:        dbDstDef.DockerRepository,
		DockerImageTag:          dbDstDef.DockerImageTag,
		DocumentationUrl:        dbDstDef.DocumentationURL,
		Icon:                    dbDstDef.Icon,
		Tombstone:               dbDstDef.Tombstone,
		Public:                  dbDstDef.Public,
		Custom:                  dbDstDef.Custom,
		CreatedAt:               timestamppb.New(dbDstDef.CreatedAt),
		UpdateAt:                timestamppb.New(dbDstDef.UpdatedAt),

		DestinationType: func() connectorPB.DestinationDefinition_DestinationType {
			if dbDstDef.ConnectionType != nil {
				return connectorPB.DestinationDefinition_DestinationType(connectorPB.DestinationDefinition_DestinationType_value[strings.ReplaceAll(string(*dbDstDef.ConnectionType), "CONNECTION_TYPE", "DESTINATION_TYPE")])
			}
			return connectorPB.DestinationDefinition_DestinationType(connectorPB.DestinationDefinition_DestinationType_value["DESTINATION_TYPE_UNSPECIFIED"])
		}(),

		ReleaseStage: func() connectorPB.ReleaseStage {
			if dbDstDef.ReleaseStage != nil {
				return connectorPB.ReleaseStage(connectorPB.ReleaseStage_value[string(*dbDstDef.ReleaseStage)])
			}
			return connectorPB.ReleaseStage(connectorPB.SourceDefinition_SourceType_value["RELEASE_STAGE_UNSPECIFIED"])
		}(),

		ReleaseDate: func() *date.Date {
			if dbDstDef.ReleaseDate != nil {
				return &date.Date{
					Year:  int32(dbDstDef.ReleaseDate.Year()),
					Month: int32(dbDstDef.ReleaseDate.Month()),
					Day:   int32(dbDstDef.ReleaseDate.Day()),
				}
			}
			return &date.Date{}
		}(),

		Spec: func() *connectorPB.Spec {
			spec := connectorPB.Spec{}
			if err := protojson.Unmarshal(dbDstDef.Spec, &spec); err != nil {
				logger.Fatal(err.Error())
				return nil
			}
			return &spec
		}(),

		ResourceRequirements: func() *structpb.Struct {
			s := structpb.Struct{}
			if err := protojson.Unmarshal(dbDstDef.ResourceRequirements, &s); err != nil {
				logger.Fatal(err.Error())
				return nil
			}
			return &s
		}(),
	}
}
