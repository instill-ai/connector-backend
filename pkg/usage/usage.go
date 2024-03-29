package usage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/instill-ai/connector-backend/config"

	"github.com/instill-ai/connector-backend/pkg/constant"
	"github.com/instill-ai/connector-backend/pkg/logger"
	"github.com/instill-ai/connector-backend/pkg/repository"
	"github.com/instill-ai/connector-backend/pkg/utils"
	"github.com/instill-ai/x/repo"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/types/known/timestamppb"

	componentBase "github.com/instill-ai/component/pkg/base"
	connector "github.com/instill-ai/connector/pkg"
	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1alpha"
	usagePB "github.com/instill-ai/protogen-go/core/usage/v1alpha"
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
	connectorData            componentBase.IConnector
}

// NewUsage initiates a usage instance
func NewUsage(ctx context.Context, r repository.Repository, ma mgmtPB.MgmtPrivateServiceClient, rc *redis.Client, usc usagePB.UsageServiceClient) Usage {
	logger, _ := logger.GetZapLogger(ctx)

	version, err := repo.ReadReleaseManifest("release-please/manifest.json")
	if err != nil {
		logger.Error(err.Error())
		return nil
	}

	var defaultOwnerUID string
	if resp, err := ma.GetUserAdmin(ctx, &mgmtPB.GetUserAdminRequest{Name: constant.DefaultUserID}); err == nil {
		defaultOwnerUID = resp.GetUser().GetUid()
	} else {
		logger.Error(err.Error())
	}

	reporter, err := usageClient.InitReporter(ctx, usc, usagePB.Session_SERVICE_CONNECTOR, config.Config.Server.Edition, version, defaultOwnerUID)
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
		connectorData:            connector.Init(logger, utils.GetConnectorOptions()),
	}
}

func (u *usage) RetrieveUsageData() interface{} {

	ctx := context.Background()
	logger, _ := logger.GetZapLogger(ctx)

	logger.Debug("Retrieve usage data...")

	pbConnectorUsageData := []*usagePB.ConnectorUsageData_UserUsageData{}

	// Roll over all users and update the metrics with the cached uuid
	userPageToken := ""
	userPageSizeMax := int32(repository.MaxPageSize)

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

			executeCount := u.redisClient.LLen(ctx, fmt.Sprintf("user:%s:connector.execute_data", user.GetUid())).Val() // O(1)

			if executeCount != 0 {
				for i := int64(0); i < executeCount; i++ {
					// LPop O(1)
					strData := u.redisClient.LPop(ctx, fmt.Sprintf("user:%s:connector.execute_data", user.GetUid())).Val()

					executeData := &utils.UsageMetricData{}
					if err := json.Unmarshal([]byte(strData), executeData); err != nil {
						logger.Warn("Usage data might be corrupted")
					}

					executeTime, _ := time.Parse(time.RFC3339Nano, executeData.ExecuteTime)

					executeDataList = append(
						executeDataList,
						&usagePB.ConnectorUsageData_UserUsageData_ConnectorExecuteData{
							ExecuteUid:             executeData.ConnectorExecuteUID,
							ExecuteTime:            timestamppb.New(executeTime),
							ConnectorUid:           executeData.ConnectorUID,
							ConnectorDefinitionUid: executeData.ConnectorDefinitionUid,
							Status:                 executeData.Status,
						},
					)
				}
			}

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

	var defaultOwnerUID string
	if resp, err := u.mgmtPrivateServiceClient.GetUserAdmin(ctx, &mgmtPB.GetUserAdminRequest{Name: constant.DefaultUserID}); err == nil {
		defaultOwnerUID = resp.GetUser().GetUid()
	} else {
		logger.Error(err.Error())
		return
	}

	go func() {
		time.Sleep(5 * time.Second)
		err := usageClient.StartReporter(ctx, u.reporter, usagePB.Session_SERVICE_CONNECTOR, config.Config.Server.Edition, u.version, defaultOwnerUID, u.RetrieveUsageData)
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

	var defaultOwnerUID string
	if resp, err := u.mgmtPrivateServiceClient.GetUserAdmin(ctx, &mgmtPB.GetUserAdminRequest{Name: constant.DefaultUserID}); err == nil {
		defaultOwnerUID = resp.GetUser().GetUid()
	} else {
		logger.Error(err.Error())
		return
	}

	err := usageClient.SingleReporter(ctx, u.reporter, usagePB.Session_SERVICE_CONNECTOR, config.Config.Server.Edition, u.version, defaultOwnerUID, u.RetrieveUsageData())
	if err != nil {
		logger.Error(fmt.Sprintf("unable to trigger single reporter: %v\n", err))
	}
}
