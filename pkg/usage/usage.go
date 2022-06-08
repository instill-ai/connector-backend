package usage

import (
	"context"
	"fmt"

	"github.com/instill-ai/connector-backend/internal/logger"
	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/connector-backend/pkg/repository"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
	usagePB "github.com/instill-ai/protogen-go/vdp/usage/v1alpha"
)

// Usage interface
type Usage interface {
	RetrieveUsageData() interface{}
}

type usage struct {
	repository        repository.Repository
	userServiceClient mgmtPB.UserServiceClient
}

// NewUsage initiates a usage instance
func NewUsage(r repository.Repository, mu mgmtPB.UserServiceClient) Usage {
	return &usage{
		repository:        r,
		userServiceClient: mu,
	}
}

func (u *usage) RetrieveUsageData() interface{} {

	logger, _ := logger.GetZapLogger()
	ctx := context.Background()

	logger.Debug("Retrieve usage data...")

	pbConnectorUsageData := []*usagePB.ConnectorUsageData_UserUsageData{}

	// Roll over all users and update the metrics with the cached uuid
	userPageToken := ""
	userPageSizeMax := int64(repository.MaxPageSize)
	for {
		userResp, err := u.userServiceClient.ListUser(ctx, &mgmtPB.ListUserRequest{
			PageSize:  &userPageSizeMax,
			PageToken: &userPageToken,
		})
		if err != nil {
			logger.Error(fmt.Sprintf("[mgmt-backend: ListUser] %s", err))
		}

		// Roll all pipeline resources on a user
		for _, user := range userResp.Users {

			connPageToken := ""

			srcConnConnectedStateNum := int64(0)
			srcConnDisconnectedStateNum := int64(0)
			srcConnDefSet := make(map[string]struct{})
			for {
				dbSrcConns, _, connNextPageToken, err := u.repository.ListConnector(
					fmt.Sprintf("users/%s", user.GetUid()),
					datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE),
					int64(repository.MaxPageSize),
					connPageToken,
					true)

				if err != nil {
					logger.Error(fmt.Sprintf("%s", err))
				}

				for _, conn := range dbSrcConns {
					if conn.State == datamodel.ConnectorState(connectorPB.Connector_STATE_CONNECTED) {
						srcConnConnectedStateNum++
					}
					if conn.State == datamodel.ConnectorState(connectorPB.Connector_STATE_DISCONNECTED) {
						srcConnDisconnectedStateNum++
					}
					srcConnDef, err := u.repository.GetConnectorDefinitionByUID(conn.ConnectorDefinitionUID, false)
					if err != nil {
						logger.Error(fmt.Sprintf("%s", err))
					}
					srcConnDefSet[srcConnDef.ID] = struct{}{}
				}

				if connNextPageToken == "" {
					break
				} else {
					connPageToken = connNextPageToken
				}
			}

			srcConnDefs := make([]string, 0, len(srcConnDefSet))
			for k := range srcConnDefSet {
				srcConnDefs = append(srcConnDefs, k)
			}

			connPageToken = ""

			dstConnConnectedStateNum := int64(0)
			dstConnDisconnectedStateNum := int64(0)
			dstConnDefSet := make(map[string]struct{})
			for {
				dbDstConns, _, connNextPageToken, err := u.repository.ListConnector(
					fmt.Sprintf("users/%s", user.GetUid()),
					datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION),
					int64(repository.MaxPageSize),
					connPageToken,
					true)

				if err != nil {
					logger.Error(fmt.Sprintf("%s", err))
				}

				for _, conn := range dbDstConns {
					if conn.State == datamodel.ConnectorState(connectorPB.Connector_STATE_CONNECTED) {
						dstConnConnectedStateNum++
					}
					if conn.State == datamodel.ConnectorState(connectorPB.Connector_STATE_DISCONNECTED) {
						dstConnDisconnectedStateNum++
					}
					dstConnDef, err := u.repository.GetConnectorDefinitionByUID(conn.ConnectorDefinitionUID, false)
					if err != nil {
						logger.Error(fmt.Sprintf("%s", err))
					}
					dstConnDefSet[dstConnDef.ID] = struct{}{}
				}

				if connNextPageToken == "" {
					break
				} else {
					connPageToken = connNextPageToken
				}
			}

			dstConnDefs := make([]string, 0, len(dstConnDefSet))
			for k := range dstConnDefSet {
				dstConnDefs = append(dstConnDefs, k)
			}

			pbConnectorUsageData = append(pbConnectorUsageData, &usagePB.ConnectorUsageData_UserUsageData{
				UserUid:                                  user.GetUid(),
				SourceConnectorConnectedStateNum:         srcConnConnectedStateNum,
				SourceConnectorDisconnectedStateNum:      srcConnDisconnectedStateNum,
				SourceConnectorDefinitionIds:             srcConnDefs,
				DestinationConnectorConnectedStateNum:    dstConnConnectedStateNum,
				DestinationConnectorDisconnectedStateNum: dstConnDisconnectedStateNum,
				DestinationConnectorDefinitionIds:        dstConnDefs,
			})

		}

		if userResp.NextPageToken == "" {
			break
		} else {
			userPageToken = userResp.NextPageToken
		}
	}

	logger.Debug("Send retrieved usage data...")

	return &usagePB.SessionReport_ConnectorUsageData{
		ConnectorUsageData: &usagePB.ConnectorUsageData{
			Usages: pbConnectorUsageData,
		},
	}
}
