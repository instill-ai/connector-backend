package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"
	"go.temporal.io/sdk/client"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"

	"github.com/instill-ai/connector-backend/config"
	"github.com/instill-ai/connector-backend/internal/external"
	"github.com/instill-ai/connector-backend/internal/logger"
	"github.com/instill-ai/connector-backend/pkg/handler"
	"github.com/instill-ai/connector-backend/pkg/repository"
	"github.com/instill-ai/connector-backend/pkg/service"
	"github.com/instill-ai/connector-backend/pkg/usage"
	"github.com/instill-ai/x/repo"
	"github.com/instill-ai/x/zapadapter"

	database "github.com/instill-ai/connector-backend/internal/db"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
	usagePB "github.com/instill-ai/protogen-go/vdp/usage/v1alpha"
	usageclient "github.com/instill-ai/usage-client/client"
)

func grpcHandlerFunc(grpcServer *grpc.Server, gwHandler http.Handler, CORSOrigins []string) http.Handler {
	return h2c.NewHandler(
		cors.New(cors.Options{
			AllowedOrigins:   CORSOrigins,
			AllowCredentials: true,
			Debug:            false,
			AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "HEAD"},
		}).Handler(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
					grpcServer.ServeHTTP(w, r)
				} else {
					gwHandler.ServeHTTP(w, r)
				}
			})),
		&http2.Server{},
	)
}

func startReporter(ctx context.Context, usageServiceClient usagePB.UsageServiceClient, r repository.Repository, mu mgmtPB.UserServiceClient) {

	logger, _ := logger.GetZapLogger()

	version, err := repo.ReadReleaseManifest("release-please/manifest.json")
	if err != nil {
		logger.Fatal(err.Error())
	}

	go func() {
		time.Sleep(5 * time.Second)
		usg := usage.NewUsage(r, mu)
		err = usageclient.StartReporter(ctx, usageServiceClient, usagePB.Session_SERVICE_CONNECTOR, config.Config.Server.Edition, version, usg.RetrieveUsageData)
		if err != nil {
			logger.Error(fmt.Sprintf("unable to start reporter: %v\n", err))
		}
	}()
}

func main() {

	if err := config.Init(); err != nil {
		log.Fatal(err.Error())
	}

	logger, _ := logger.GetZapLogger()
	defer func() {
		// can't handle the error due to https://github.com/uber-go/zap/issues/880
		_ = logger.Sync()
	}()
	grpc_zap.ReplaceGrpcLoggerV2(logger)

	db := database.GetConnection()
	defer database.Close(db)

	temporalClient, err := client.Dial(client.Options{
		// ZapAdapter implements log.Logger interface and can be passed
		// to the client constructor using client using client.Options.
		Logger:   zapadapter.NewZapAdapter(logger),
		HostPort: config.Config.Temporal.ClientOptions.HostPort,
	})
	if err != nil {
		logger.Fatal(err.Error())
	}
	defer temporalClient.Close()

	// Create tls based credential.
	var creds credentials.TransportCredentials
	if config.Config.Server.HTTPS.Cert != "" && config.Config.Server.HTTPS.Key != "" {
		creds, err = credentials.NewServerTLSFromFile(config.Config.Server.HTTPS.Cert, config.Config.Server.HTTPS.Key)
		if err != nil {
			logger.Fatal(fmt.Sprintf("failed to create credentials: %v", err))
		}
	}

	// Shared options for the logger, with a custom gRPC code to log level function.
	opts := []grpc_zap.Option{
		grpc_zap.WithDecider(func(fullMethodName string, err error) bool {
			// will not log gRPC calls if it was a call to liveness or readiness and no error was raised
			if err == nil {
				if match, _ := regexp.MatchString("vdp.connector.v1alpha.ConnectorService/.*ness$", fullMethodName); match {
					return false
				}
			}
			// by default everything will be logged
			return true
		}),
	}

	grpcServerOpts := []grpc.ServerOption{
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			streamAppendMetadataInterceptor,
			grpc_zap.StreamServerInterceptor(logger, opts...),
			grpc_recovery.StreamServerInterceptor(recoveryInterceptorOpt()),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			unaryAppendMetadataInterceptor,
			grpc_zap.UnaryServerInterceptor(logger, opts...),
			grpc_recovery.UnaryServerInterceptor(recoveryInterceptorOpt()),
		)),
	}
	if config.Config.Server.HTTPS.Cert != "" && config.Config.Server.HTTPS.Key != "" {
		grpcServerOpts = append(grpcServerOpts, grpc.Creds(creds))
	}

	userServiceClient, userServiceClientConn := external.InitUserServiceClient()
	defer userServiceClientConn.Close()

	pipelineServiceClient, pipelineServiceClientConn := external.InitPipelineServiceClient()
	defer pipelineServiceClientConn.Close()

	repository := repository.NewRepository(db)

	grpcS := grpc.NewServer(grpcServerOpts...)
	reflection.Register(grpcS)

	connectorPB.RegisterConnectorServiceServer(
		grpcS,
		handler.NewHandler(
			service.NewService(
				repository,
				userServiceClient,
				pipelineServiceClient,
				temporalClient,
			)))

	gwS := runtime.NewServeMux(
		runtime.WithForwardResponseOption(httpResponseModifier),
		runtime.WithErrorHandler(errorHandler),
		runtime.WithIncomingHeaderMatcher(customMatcher),
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				UseProtoNames:   true,
				EmitUnpopulated: true,
				UseEnumNumbers:  false,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		}),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start usage reporter
	if !config.Config.Server.DisableUsage {
		usageServiceClient, usageServiceClientConn := external.InitUsageServiceClient()
		defer usageServiceClientConn.Close()
		startReporter(ctx, usageServiceClient, repository, userServiceClient)
	}

	var dialOpts []grpc.DialOption
	if config.Config.Server.HTTPS.Cert != "" && config.Config.Server.HTTPS.Key != "" {
		dialOpts = []grpc.DialOption{grpc.WithTransportCredentials(creds)}
	} else {
		dialOpts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	}

	if err := connectorPB.RegisterConnectorServiceHandlerFromEndpoint(ctx, gwS, fmt.Sprintf(":%v", config.Config.Server.Port), dialOpts); err != nil {
		logger.Fatal(err.Error())
	}

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%v", config.Config.Server.Port),
		Handler: grpcHandlerFunc(grpcS, gwS, config.Config.Server.CORSOrigins),
	}

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 5 seconds.
	quitSig := make(chan os.Signal, 1)
	errSig := make(chan error)
	if config.Config.Server.HTTPS.Cert != "" && config.Config.Server.HTTPS.Key != "" {
		go func() {
			if err := httpServer.ListenAndServeTLS(config.Config.Server.HTTPS.Cert, config.Config.Server.HTTPS.Key); err != nil {
				errSig <- err
			}
		}()
	} else {
		go func() {
			if err := httpServer.ListenAndServe(); err != nil {
				errSig <- err
			}
		}()
	}
	logger.Info("gRPC server is running.")

	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be catch, so don't need add it
	signal.Notify(quitSig, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errSig:
		logger.Error(fmt.Sprintf("Fatal error: %v\n", err))
	case <-quitSig:
		logger.Info("Shutting down server...")
		grpcS.GracefulStop()
	}
}
