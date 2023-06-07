package handler

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/connector-backend/pkg/logger"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

// PBToDBConnector converts protobuf data model to db data model
func PBToDBConnector(
	ctx context.Context,
	pbConnector interface{},
	connectorType datamodel.ConnectorType,
	ownerRscName string,
	connectorDefinitionUID uuid.UUID) *datamodel.Connector {

	logger, _ := logger.GetZapLogger(ctx)

	var uid uuid.UUID
	var id string
	var state datamodel.ConnectorState
	var tombstone bool
	var description sql.NullString
	var configuration *structpb.Struct
	var createTime time.Time
	var updateTime time.Time
	var err error

	switch v := pbConnector.(type) {
	case *connectorPB.SourceConnector:
		id = v.GetId()
		state = datamodel.ConnectorState(v.GetConnector().GetState())
		tombstone = v.GetConnector().GetTombstone()
		configuration = v.GetConnector().GetConfiguration()
		createTime = v.GetConnector().GetCreateTime().AsTime()
		updateTime = v.GetConnector().GetUpdateTime().AsTime()

		uid, err = uuid.FromString(v.GetUid())
		if err != nil {
			logger.Fatal(err.Error())
		}

		description = sql.NullString{
			String: v.GetConnector().GetDescription(),
			Valid:  true,
		}
	case *connectorPB.DestinationConnector:
		id = v.GetId()
		state = datamodel.ConnectorState(v.GetConnector().GetState())
		tombstone = v.GetConnector().GetTombstone()
		configuration = v.GetConnector().GetConfiguration()
		createTime = v.GetConnector().GetCreateTime().AsTime()
		updateTime = v.GetConnector().GetUpdateTime().AsTime()

		uid, err = uuid.FromString(v.GetUid())
		if err != nil {
			logger.Fatal(err.Error())
		}

		description = sql.NullString{
			String: v.GetConnector().GetDescription(),
			Valid:  true,
		}
	}

	return &datamodel.Connector{
		Owner:                  ownerRscName,
		ID:                     id,
		ConnectorType:          connectorType,
		Description:            description,
		State:                  state,
		Tombstone:              tombstone,
		ConnectorDefinitionUID: connectorDefinitionUID,

		Configuration: func() []byte {
			if configuration != nil {
				b, err := configuration.MarshalJSON()
				if err != nil {
					logger.Error(err.Error())
				}
				return b
			}
			return []byte{}
		}(),

		BaseDynamic: datamodel.BaseDynamic{
			UID:        uid,
			CreateTime: createTime,
			UpdateTime: updateTime,
		},
	}
}

// DBToPBConnector converts db data model to protobuf data model
func DBToPBConnector(
	ctx context.Context,
	dbConnector *datamodel.Connector,
	connectorType datamodel.ConnectorType,
	owner string,
	connectorDefinition string) interface{} {

	logger, _ := logger.GetZapLogger(ctx)

	connector := &connectorPB.Connector{

		Description: &dbConnector.Description.String,
		State:       connectorPB.Connector_State(dbConnector.State),
		Tombstone:   dbConnector.Tombstone,
		CreateTime:  timestamppb.New(dbConnector.CreateTime),
		UpdateTime:  timestamppb.New(dbConnector.UpdateTime),

		Configuration: func() *structpb.Struct {
			if dbConnector.Configuration != nil {
				str := structpb.Struct{}
				err := str.UnmarshalJSON(dbConnector.Configuration)
				if err != nil {
					logger.Fatal(err.Error())
				}
				return &str
			}
			return nil
		}(),
	}
	if connectorType == datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE) {
		pbConnector := connectorPB.SourceConnector{
			Uid:                       dbConnector.UID.String(),
			Name:                      fmt.Sprintf("source-connectors/%s", dbConnector.ID),
			Id:                        dbConnector.ID,
			SourceConnectorDefinition: connectorDefinition,
			Connector:                 connector,
		}
		if strings.HasPrefix(owner, "users/") {
			pbConnector.GetConnector().Owner = &connectorPB.Connector_User{User: owner}
		} else if strings.HasPrefix(owner, "organizations/") {
			pbConnector.GetConnector().Owner = &connectorPB.Connector_Org{Org: owner}
		}
		return &pbConnector
	} else if connectorType == datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION) {
		pbConnector := connectorPB.DestinationConnector{
			Uid:                            dbConnector.UID.String(),
			Name:                           fmt.Sprintf("destination-connectors/%s", dbConnector.ID),
			Id:                             dbConnector.ID,
			DestinationConnectorDefinition: connectorDefinition,
			Connector:                      connector,
		}
		if strings.HasPrefix(owner, "users/") {
			pbConnector.GetConnector().Owner = &connectorPB.Connector_User{User: owner}
		} else if strings.HasPrefix(owner, "organizations/") {
			pbConnector.GetConnector().Owner = &connectorPB.Connector_Org{Org: owner}
		}
		return &pbConnector
	}
	return nil
}
