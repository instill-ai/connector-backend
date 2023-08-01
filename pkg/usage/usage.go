package usage

import (
	"context"
	"fmt"
	"time"

	"github.com/instill-ai/connector-backend/config"
	"github.com/instill-ai/connector-backend/pkg/connector"
	"github.com/instill-ai/connector-backend/pkg/logger"
	"github.com/instill-ai/connector-backend/pkg/repository"
	"github.com/instill-ai/x/repo"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/types/known/timestamppb"

	connectorData "github.com/instill-ai/connector-data/pkg"
	connectorBase "github.com/instill-ai/connector/pkg/base"
	mgmtPB "github.com/instill-ai/protogen-go/base/mgmt/v1alpha"
	usagePB "github.com/instill-ai/protogen-go/base/usage/v1alpha"
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
	redisClient              *redis.Client
	reporter                 usageReporter.Reporter
	version                  string
	connectorData            connectorBase.IConnector
}

// NewUsage initiates a usage instance
func NewUsage(ctx context.Context, r repository.Repository, ma mgmtPB.MgmtPrivateServiceClient, rc *redis.Client, usc usagePB.UsageServiceClient) Usage {
	logger, _ := logger.GetZapLogger(ctx)

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
		redisClient:              rc,
		reporter:                 reporter,
		version:                  version,
		connectorData:            connectorData.Init(logger, connector.GetConnectorDataOptions()),
	}
}

func (u *usage) RetrieveUsageData() interface{} {

	ctx := context.Background()
	logger, _ := logger.GetZapLogger(ctx)

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

			executeDataList := []*usagePB.ConnectorUsageData_UserUsageData_ConnectorExecuteData{}

			executeCount := u.redisClient.LLen(ctx, fmt.Sprintf("user:%s:execute.execute_uid", user.GetUid())).Val() // O(1)

			if executeCount != 0 {
				for i := int64(0); i < executeCount; i++ {
					// LPop O(1)
					timeStr, _ := time.Parse(time.RFC3339Nano, u.redisClient.LPop(ctx, fmt.Sprintf("user:%s:execute.execute_time", user.GetUid())).Val())
					executeData := &usagePB.ConnectorUsageData_UserUsageData_ConnectorExecuteData{
						ExecuteUid:             u.redisClient.LPop(ctx, fmt.Sprintf("user:%s:execute.execute_uid", user.GetUid())).Val(),
						ExecuteTime:            timestamppb.New(timeStr),
						ConnectorUid:           u.redisClient.LPop(ctx, fmt.Sprintf("user:%s:execute.connector_uid", user.GetUid())).Val(),
						ConnectorDefinitionUid: u.redisClient.LPop(ctx, fmt.Sprintf("user:%s:execute.connector_definition_uid", user.GetUid())).Val(),
						Status:                 mgmtPB.Status(mgmtPB.Status_value[u.redisClient.LPop(ctx, fmt.Sprintf("user:%s:execute.status", user.GetUid())).Val()]),
					}
					executeDataList = append(executeDataList, executeData)
				}
			}

			// Cleanup in case of length mismatch between lists
			u.redisClient.Unlink(
				ctx,
				fmt.Sprintf("user:%s:execute.execute_time", user.GetUid()),
				fmt.Sprintf("user:%s:execute.execute_uid", user.GetUid()),
				fmt.Sprintf("user:%s:execute.connector_uid", user.GetUid()),
				fmt.Sprintf("user:%s:execute.connector_definition_uid", user.GetUid()),
				fmt.Sprintf("user:%s:execute.status", user.GetUid()),
			)

			pbConnectorUsageData = append(pbConnectorUsageData, &usagePB.ConnectorUsageData_UserUsageData{
				UserUid:              user.GetUid(),
				ConnectorExecuteData: executeDataList,
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

	logger, _ := logger.GetZapLogger(ctx)
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
	logger, _ := logger.GetZapLogger(ctx)
	err := usageClient.SingleReporter(ctx, u.reporter, usagePB.Session_SERVICE_CONNECTOR, config.Config.Server.Edition, u.version, u.RetrieveUsageData())
	if err != nil {
		logger.Error(fmt.Sprintf("unable to trigger single reporter: %v\n", err))
	}
}
