package handler

import (
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
		ConnectionType:     connectorPB.ConnectionType(dbSrcDef.ConnectionType),

		ReleaseStage: func() connectorPB.ReleaseStage {
			return connectorPB.ReleaseStage(dbSrcDef.ReleaseStage)
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
			if dbSrcDef.Spec != nil {
				if err := protojson.Unmarshal(dbSrcDef.Spec, &spec); err != nil {
					logger.Fatal(err.Error())
					return nil
				}
				return &spec
			}
			return nil
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
		ConnectionType:          connectorPB.ConnectionType(dbDstDef.ConnectionType),

		ReleaseStage: func() connectorPB.ReleaseStage {
			return connectorPB.ReleaseStage(dbDstDef.ReleaseStage)
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
			if dbDstDef.Spec != nil {
				spec := connectorPB.Spec{}
				if err := protojson.Unmarshal(dbDstDef.Spec, &spec); err != nil {
					logger.Fatal(err.Error())
					return nil
				}
				return &spec
			}
			return nil
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

func convertDBConnectorToPBConnector(dbConnector *datamodel.Connector) *connectorPB.Connector {
	logger, _ := logger.GetZapLogger()

	configuration := connectorPB.Spec{}
	err := protojson.Unmarshal(dbConnector.Configuration, &configuration)
	if err != nil {
		logger.Fatal(err.Error())
	}

	pbConnector := connectorPB.Connector{
		Id:                    dbConnector.BaseDynamic.ID.String(),
		ConnectorDefinitionId: dbConnector.ConnectorDefinitionID.String(),
		Name:                  dbConnector.Name,
		Description:           dbConnector.Description,
		Configuration:         &configuration,
		ConnectorType:         connectorPB.ConnectorType(connectorPB.ConnectorType_value[string(dbConnector.ConnectorType)]),
		Tombstone:             dbConnector.Tombstone,
		OwnerId:               dbConnector.OwnerID.String(),
		FullName:              dbConnector.FullName,
		CreatedAt:             timestamppb.New(dbConnector.CreatedAt),
		UpdateAt:              timestamppb.New(dbConnector.UpdatedAt),
	}

	return &pbConnector
}
