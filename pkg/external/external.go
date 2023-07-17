package external

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"

	"github.com/instill-ai/connector-backend/config"
	"github.com/instill-ai/connector-backend/pkg/logger"

	mgmtPB "github.com/instill-ai/protogen-go/base/mgmt/v1alpha"
	usagePB "github.com/instill-ai/protogen-go/base/usage/v1alpha"
	controllerPB "github.com/instill-ai/protogen-go/vdp/controller/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

// InitMgmtPrivateServiceClient initialises a MgmtPrivateServiceClient instance
func InitMgmtPrivateServiceClient(ctx context.Context) (mgmtPB.MgmtPrivateServiceClient, *grpc.ClientConn) {
	logger, _ := logger.GetZapLogger(ctx)

	var clientDialOpts grpc.DialOption
	var creds credentials.TransportCredentials
	var err error
	if config.Config.MgmtBackend.HTTPS.Cert != "" && config.Config.MgmtBackend.HTTPS.Key != "" {
		creds, err = credentials.NewServerTLSFromFile(config.Config.MgmtBackend.HTTPS.Cert, config.Config.MgmtBackend.HTTPS.Key)
		if err != nil {
			logger.Fatal(err.Error())
		}
		clientDialOpts = grpc.WithTransportCredentials(creds)
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.Dial(fmt.Sprintf("%v:%v", config.Config.MgmtBackend.Host, config.Config.MgmtBackend.PrivatePort), clientDialOpts)
	if err != nil {
		logger.Error(err.Error())
		return nil, nil
	}

	return mgmtPB.NewMgmtPrivateServiceClient(clientConn), clientConn
}

// InitPipelinePublicServiceClient initialises a PipelineServiceClient instance
func InitPipelinePublicServiceClient(ctx context.Context) (pipelinePB.PipelinePublicServiceClient, *grpc.ClientConn) {
	logger, _ := logger.GetZapLogger(ctx)

	var clientDialOpts grpc.DialOption
	var creds credentials.TransportCredentials
	var err error
	if config.Config.PipelineBackend.HTTPS.Cert != "" && config.Config.PipelineBackend.HTTPS.Key != "" {
		creds, err = credentials.NewServerTLSFromFile(config.Config.PipelineBackend.HTTPS.Cert, config.Config.PipelineBackend.HTTPS.Key)
		if err != nil {
			logger.Fatal(err.Error())
		}
		clientDialOpts = grpc.WithTransportCredentials(creds)
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.Dial(fmt.Sprintf("%v:%v", config.Config.PipelineBackend.Host, config.Config.PipelineBackend.PublicPort), clientDialOpts)
	if err != nil {
		logger.Error(err.Error())
		return nil, nil
	}

	return pipelinePB.NewPipelinePublicServiceClient(clientConn), clientConn
}

// InitUsageServiceClient initialises a UsageServiceClient instance
func InitUsageServiceClient(ctx context.Context) (usagePB.UsageServiceClient, *grpc.ClientConn) {
	logger, _ := logger.GetZapLogger(ctx)

	var clientDialOpts grpc.DialOption
	if config.Config.Server.Usage.TLSEnabled {
		tlsConfig := &tls.Config{}
		clientDialOpts = grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.Dial(fmt.Sprintf("%v:%v", config.Config.Server.Usage.Host, config.Config.Server.Usage.Port), clientDialOpts)
	if err != nil {
		logger.Error(err.Error())
		return nil, nil
	}

	return usagePB.NewUsageServiceClient(clientConn), clientConn
}

// InitControllerPrivateServiceClient initialises a ControllerPrivateServiceClient instance
func InitControllerPrivateServiceClient(ctx context.Context) (controllerPB.ControllerPrivateServiceClient, *grpc.ClientConn) {
	logger, _ := logger.GetZapLogger(ctx)

	var clientDialOpts grpc.DialOption
	var creds credentials.TransportCredentials
	var err error
	if config.Config.Controller.HTTPS.Cert != "" && config.Config.Controller.HTTPS.Key != "" {
		creds, err = credentials.NewServerTLSFromFile(config.Config.Controller.HTTPS.Cert, config.Config.Controller.HTTPS.Key)
		if err != nil {
			logger.Fatal(err.Error())
		}
		clientDialOpts = grpc.WithTransportCredentials(creds)
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.Dial(fmt.Sprintf("%v:%v", config.Config.Controller.Host, config.Config.Controller.PrivatePort), clientDialOpts)
	if err != nil {
		logger.Fatal(err.Error())
	}

	return controllerPB.NewControllerPrivateServiceClient(clientConn), clientConn
}

// InitInfluxDBServiceClient initialises a InfluxDBServiceClient instance
func InitInfluxDBServiceClient(ctx context.Context) (influxdb2.Client, api.WriteAPI) {

	logger, _ := logger.GetZapLogger(ctx)

	var creds credentials.TransportCredentials
	var err error

	influxOptions := influxdb2.DefaultOptions()
	if config.Config.Server.Debug {
		influxOptions = influxOptions.SetLogLevel(log.DebugLevel)
	}
	influxOptions = influxOptions.SetFlushInterval(uint(time.Duration(config.Config.InfluxDB.FlushInterval * int(time.Second)).Milliseconds()))

	if config.Config.InfluxDB.HTTPS.Cert != "" && config.Config.InfluxDB.HTTPS.Key != "" {
		// TODO: support TLS
		creds, err = credentials.NewServerTLSFromFile(config.Config.InfluxDB.HTTPS.Cert, config.Config.InfluxDB.HTTPS.Key)
		if err != nil {
			logger.Fatal(err.Error())
		}
		logger.Info(creds.Info().ServerName)
	}

	client := influxdb2.NewClientWithOptions(
		config.Config.InfluxDB.URL,
		config.Config.InfluxDB.Token,
		influxOptions,
	)

	if _, err := client.Ping(ctx); err != nil {
		logger.Warn(err.Error())
	}

	writeAPI := client.WriteAPI(config.Config.InfluxDB.Org, config.Config.InfluxDB.Bucket)

	return client, writeAPI
}
