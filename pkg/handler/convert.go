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
	pbConnector *connectorPB.Connector,
	ownerRscName string,
	connectorDefinition *connectorPB.ConnectorDefinition) *datamodel.Connector {

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

	id = pbConnector.GetId()
	state = datamodel.ConnectorState(pbConnector.GetState())
	tombstone = pbConnector.GetTombstone()
	configuration = pbConnector.GetConfiguration()
	createTime = pbConnector.GetCreateTime().AsTime()
	updateTime = pbConnector.GetUpdateTime().AsTime()

	uid, err = uuid.FromString(pbConnector.GetUid())
	if err != nil {
		logger.Fatal(err.Error())
	}

	description = sql.NullString{
		String: pbConnector.GetDescription(),
		Valid:  true,
	}

	return &datamodel.Connector{
		Owner:                  ownerRscName,
		ID:                     id,
		ConnectorType:          datamodel.ConnectorType(connectorDefinition.ConnectorType),
		Description:            description,
		State:                  state,
		Tombstone:              tombstone,
		ConnectorDefinitionUID: uuid.FromStringOrNil(connectorDefinition.Uid),

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
	owner string,
	connectorDefinitionName string) *connectorPB.Connector {

	logger, _ := logger.GetZapLogger(ctx)

	pbConnector := &connectorPB.Connector{
		Uid:                     dbConnector.UID.String(),
		Name:                    fmt.Sprintf("connectors/%s", dbConnector.ID),
		Id:                      dbConnector.ID,
		ConnectorDefinitionName: connectorDefinitionName,
		ConnectorType:           connectorPB.ConnectorType(dbConnector.ConnectorType),
		Description:             &dbConnector.Description.String,
		State:                   connectorPB.Connector_State(dbConnector.State),
		Tombstone:               dbConnector.Tombstone,
		CreateTime:              timestamppb.New(dbConnector.CreateTime),
		UpdateTime:              timestamppb.New(dbConnector.UpdateTime),

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

	if strings.HasPrefix(owner, "users/") {
		pbConnector.Owner = &connectorPB.Connector_User{User: owner}
	} else if strings.HasPrefix(owner, "organizations/") {
		pbConnector.Owner = &connectorPB.Connector_Org{Org: owner}
	}
	return pbConnector

}
