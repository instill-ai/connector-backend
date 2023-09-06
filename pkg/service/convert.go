package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/instill-ai/connector-backend/pkg/connector"
	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/connector-backend/pkg/logger"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

// convertProtoToDatamodel converts protobuf data model to db data model
func (s *service) convertProtoToDatamodel(
	ctx context.Context,
	pbConnectorResource *connectorPB.ConnectorResource,
) (*datamodel.ConnectorResource, error) {

	logger, _ := logger.GetZapLogger(ctx)

	var uid uuid.UUID
	var id string
	var state datamodel.ConnectorResourceState
	var tombstone bool
	var description sql.NullString
	var configuration *structpb.Struct
	var createTime time.Time
	var updateTime time.Time
	var err error

	id = pbConnectorResource.GetId()
	state = datamodel.ConnectorResourceState(pbConnectorResource.GetState())
	tombstone = pbConnectorResource.GetTombstone()
	configuration = pbConnectorResource.GetConfiguration()
	createTime = pbConnectorResource.GetCreateTime().AsTime()
	updateTime = pbConnectorResource.GetUpdateTime().AsTime()

	connectorDefinition, err := s.connectors.GetConnectorDefinitionById(strings.Split(pbConnectorResource.ConnectorDefinitionName, "/")[1])
	if err != nil {
		return nil, err
	}

	uid = uuid.FromStringOrNil(pbConnectorResource.GetUid())
	if err != nil {
		return nil, err
	}

	description = sql.NullString{
		String: pbConnectorResource.GetDescription(),
		Valid:  true,
	}

	var owner string

	switch pbConnectorResource.Owner.(type) {
	case *connectorPB.ConnectorResource_User:
		owner, err = s.ConvertOwnerNameToPermalink(pbConnectorResource.GetUser())
		if err != nil {
			return nil, err
		}
	case *connectorPB.ConnectorResource_Org:
		return nil, fmt.Errorf("org not supported")
	}

	return &datamodel.ConnectorResource{
		Owner:                  owner,
		ID:                     id,
		ConnectorType:          datamodel.ConnectorResourceType(connectorDefinition.Type),
		Description:            description,
		State:                  state,
		Tombstone:              tombstone,
		ConnectorDefinitionUID: uuid.FromStringOrNil(connectorDefinition.Uid),
		Visibility:             datamodel.ConnectorResourceVisibility(pbConnectorResource.Visibility),

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
	}, nil
}

// convertDatamodelToProto converts db data model to protobuf data model
func (s *service) convertDatamodelToProto(
	ctx context.Context,
	dbConnectorResource *datamodel.ConnectorResource,
	view connectorPB.View,
	credentialMask bool,
) (*connectorPB.ConnectorResource, error) {

	logger, _ := logger.GetZapLogger(ctx)

	owner, err := s.ConvertOwnerPermalinkToName(dbConnectorResource.Owner)
	if err != nil {
		return nil, err
	}
	dbConnDef, err := s.connectors.GetConnectorDefinitionByUid(dbConnectorResource.ConnectorDefinitionUID)
	if err != nil {
		return nil, err
	}
	pbConnectorResource := &connectorPB.ConnectorResource{
		Uid:                     dbConnectorResource.UID.String(),
		Name:                    fmt.Sprintf("%s/connector-resources/%s", owner, dbConnectorResource.ID),
		Id:                      dbConnectorResource.ID,
		ConnectorDefinitionName: dbConnDef.GetName(),

		Type:        connectorPB.ConnectorType(dbConnectorResource.ConnectorType),
		Description: &dbConnectorResource.Description.String,
		State:       connectorPB.ConnectorResource_State(dbConnectorResource.State),
		Tombstone:   dbConnectorResource.Tombstone,
		CreateTime:  timestamppb.New(dbConnectorResource.CreateTime),
		UpdateTime:  timestamppb.New(dbConnectorResource.UpdateTime),
		DeleteTime:  timestamppb.New(dbConnectorResource.DeleteTime.Time),
		Visibility:  connectorPB.ConnectorResource_Visibility(dbConnectorResource.Visibility),

		Configuration: func() *structpb.Struct {
			if dbConnectorResource.Configuration != nil {
				str := structpb.Struct{}
				err := str.UnmarshalJSON(dbConnectorResource.Configuration)
				if err != nil {
					logger.Fatal(err.Error())
				}
				return &str
			}
			return nil
		}(),
	}

	if strings.HasPrefix(owner, "users/") {
		pbConnectorResource.Owner = &connectorPB.ConnectorResource_User{User: owner}
	} else if strings.HasPrefix(owner, "organizations/") {
		pbConnectorResource.Owner = &connectorPB.ConnectorResource_Org{Org: owner}
	}
	if view != connectorPB.View_VIEW_BASIC {
		if credentialMask {
			connector.MaskCredentialFields(s.connectors, dbConnDef.Id, pbConnectorResource.Configuration)
		}
		if view == connectorPB.View_VIEW_FULL {
			pbConnectorResource.ConnectorDefinition = dbConnDef
		}
	}

	return pbConnectorResource, nil

}

func (s *service) convertDatamodelArrayToProtoArray(
	ctx context.Context,
	dbConnectorResources []*datamodel.ConnectorResource,
	view connectorPB.View,
	credentialMask bool,
) ([]*connectorPB.ConnectorResource, error) {

	var err error
	pbConnectorResources := make([]*connectorPB.ConnectorResource, len(dbConnectorResources))
	for idx := range dbConnectorResources {
		pbConnectorResources[idx], err = s.convertDatamodelToProto(
			ctx,
			dbConnectorResources[idx],
			view,
			credentialMask,
		)
		if err != nil {
			return nil, err
		}

	}

	return pbConnectorResources, nil

}
