package usage

import (
	"context"
	"fmt"
	"time"

	"github.com/instill-ai/connector-backend/config"
	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/connector-backend/pkg/logger"
	"github.com/instill-ai/connector-backend/pkg/repository"
	"github.com/instill-ai/x/repo"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
	usagePB "github.com/instill-ai/protogen-go/vdp/usage/v1alpha"
	usageClient "github.com/instill-ai/usage-client/client"
	usageReporter "github.com/instill-ai/usage-client/reporter"
)

// Usage interface
type Usage interface {
	RetrieveUsageData() interface{}
	StartReporter(ctx context.Context)
	TriggerSingleReporter(ctx context.Context)
}

type usage struct {
	repository               repository.Repository
	mgmtPrivateServiceClient mgmtPB.MgmtPrivateServiceClient
	reporter                 usageReporter.Reporter
	version                  string
}

// NewUsage initiates a usage instance
func NewUsage(ctx context.Context, r repository.Repository, ma mgmtPB.MgmtPrivateServiceClient, usc usagePB.UsageServiceClient) Usage {
	logger, _ := logger.GetZapLogger()

	version, err := repo.ReadReleaseManifest("release-please/manifest.json")
	if err != nil {
		logger.Error(err.Error())
		return nil
	}

	reporter, err := usageClient.InitReporter(ctx, usc, usagePB.Session_SERVICE_CONNECTOR, config.Config.Server.Edition, version)
	if err != nil {
		logger.Error(err.Error())
		return nil
	}

	return &usage{
		repository:               r,
		mgmtPrivateServiceClient: ma,
		reporter:                 reporter,
		version:                  version,
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
		userResp, err := u.mgmtPrivateServiceClient.ListUsersAdmin(ctx, &mgmtPB.ListUsersAdminRequest{
			PageSize:  &userPageSizeMax,
			PageToken: &userPageToken,
		})
		if err != nil {
			logger.Error(fmt.Sprintf("[mgmt-backend: ListUser] %s", err))
			break
		}

		// Roll all pipeline resources on a user
		for _, user := range userResp.Users {

			connPageToken := ""

			srcConnConnectedStateNum := int64(0)
			srcConnDisconnectedStateNum := int64(0)
			srcConnDefSet := make(map[string]struct{})
			for {
				dbSrcConns, _, connNextPageToken, err := u.repository.ListConnectors(
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
				dbDstConns, _, connNextPageToken, err := u.repository.ListConnectors(
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

func (u *usage) StartReporter(ctx context.Context) {
	if u.reporter == nil {
		return
	}

	logger, _ := logger.GetZapLogger()
	go func() {
		time.Sleep(5 * time.Second)
		err := usageClient.StartReporter(ctx, u.reporter, usagePB.Session_SERVICE_CONNECTOR, config.Config.Server.Edition, u.version, u.RetrieveUsageData)
		if err != nil {
			logger.Error(fmt.Sprintf("unable to start reporter: %v\n", err))
		}
	}()
}

func (u *usage) TriggerSingleReporter(ctx context.Context) {
	if u.reporter == nil {
		return
	}
	logger, _ := logger.GetZapLogger()
	err := usageClient.SingleReporter(ctx, u.reporter, usagePB.Session_SERVICE_CONNECTOR, config.Config.Server.Edition, u.version, u.RetrieveUsageData())
	if err != nil {
		logger.Error(fmt.Sprintf("unable to trigger single reporter: %v\n", err))
	}
}
