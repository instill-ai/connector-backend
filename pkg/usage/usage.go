package usage

import (
	"context"
	"fmt"
	"time"

	"github.com/instill-ai/connector-backend/config"
	"github.com/instill-ai/connector-backend/pkg/connector"
	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/connector-backend/pkg/logger"
	"github.com/instill-ai/connector-backend/pkg/repository"
	"github.com/instill-ai/x/repo"
	"go.einride.tech/aip/filtering"

	connectorData "github.com/instill-ai/connector-data/pkg"
	connectorBase "github.com/instill-ai/connector/pkg/base"
	mgmtPB "github.com/instill-ai/protogen-go/base/mgmt/v1alpha"
	usagePB "github.com/instill-ai/protogen-go/base/usage/v1alpha"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
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
	connectorData            connectorBase.IConnector
}

// NewUsage initiates a usage instance
func NewUsage(ctx context.Context, r repository.Repository, ma mgmtPB.MgmtPrivateServiceClient, usc usagePB.UsageServiceClient) Usage {
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

	var connType connectorPB.ConnectorType
	declarations, err := filtering.NewDeclarations([]filtering.DeclarationOption{
		filtering.DeclareStandardFunctions(),
		filtering.DeclareEnumIdent("connector_type", connType.Type()),
	}...)
	if err != nil {
		logger.Error(fmt.Sprintf("%s", err))
	}

	var dstParser filtering.Parser
	dstParser.Init("connector_type=CONNECTOR_TYPE_DATA")
	dstParsedExpr, err := dstParser.Parse()
	if err != nil {
		logger.Error(fmt.Sprintf("%s", err))
	}
	var dstChecker filtering.Checker
	dstChecker.Init(dstParsedExpr.Expr, dstParsedExpr.SourceInfo, declarations)
	dstCheckedExpr, err := dstChecker.Check()
	if err != nil {
		logger.Error(fmt.Sprintf("%s", err))
	}

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
			dstConnConnectedStateNum := int64(0)
			dstConnDisconnectedStateNum := int64(0)
			dstConnDefSet := make(map[string]struct{})
			for {
				dbDstConns, _, connNextPageToken, err := u.repository.ListConnectors(
					ctx,
					fmt.Sprintf("users/%s", user.GetUid()),
					int64(repository.MaxPageSize),
					connPageToken,
					true,
					filtering.Filter{
						CheckedExpr: dstCheckedExpr,
					},
				)

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
					dstConnDef, err := u.connectorData.GetConnectorDefinitionByUid(conn.ConnectorDefinitionUID)
					if err != nil {
						logger.Error(fmt.Sprintf("%s", err))
					}
					dstConnDefSet[dstConnDef.GetId()] = struct{}{}
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
