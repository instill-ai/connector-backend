package external

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/instill-ai/connector-backend/configs"
	"github.com/instill-ai/connector-backend/internal/logger"

	mgmtPB "github.com/instill-ai/protogen-go/mgmt/v1alpha"
)

// InitUserServiceClient initialises a UserServiceClient instance
func InitUserServiceClient() mgmtPB.UserServiceClient {
	logger, _ := logger.GetZapLogger()

	var clientDialOpts grpc.DialOption
	var creds credentials.TransportCredentials
	var err error
	if configs.Config.MgmtBackend.HTTPS.Cert != "" && configs.Config.MgmtBackend.HTTPS.Key != "" {
		creds, err = credentials.NewServerTLSFromFile(configs.Config.MgmtBackend.HTTPS.Cert, configs.Config.MgmtBackend.HTTPS.Key)
		if err != nil {
			logger.Fatal(err.Error())
		}
		clientDialOpts = grpc.WithTransportCredentials(creds)
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.Dial(fmt.Sprintf("%v:%v", configs.Config.MgmtBackend.Host, configs.Config.MgmtBackend.Port), clientDialOpts)
	if err != nil {
		logger.Fatal(err.Error())
	}

	return mgmtPB.NewUserServiceClient(clientConn)
}
