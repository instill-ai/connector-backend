package main

import (
	"fmt"
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	dockerclient "github.com/docker/docker/client"

	"github.com/instill-ai/connector-backend/config"
	"github.com/instill-ai/connector-backend/pkg/external"
	"github.com/instill-ai/connector-backend/pkg/logger"
	"github.com/instill-ai/connector-backend/pkg/repository"
	"github.com/instill-ai/x/temporal"
	"github.com/instill-ai/x/zapadapter"

	database "github.com/instill-ai/connector-backend/pkg/db"
	connWorker "github.com/instill-ai/connector-backend/pkg/worker"
)

func main() {

	if err := config.Init(); err != nil {
		log.Fatal(err.Error())
	}

	logger, _ := logger.GetZapLogger()
	defer func() {
		// can't handle the error due to https://github.com/uber-go/zap/issues/880
		_ = logger.Sync()
	}()

	db := database.GetConnection()
	defer database.Close(db)

	dockerClient, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		logger.Error(err.Error())
	}
	defer dockerClient.Close()

	controllerClient, controllerClientConn := external.InitControllerPrivateServiceClient()
	if controllerClientConn != nil {
		defer controllerClientConn.Close()
	}

	cw := connWorker.NewWorker(repository.NewRepository(db), dockerClient, controllerClient)

	var temporalClientOptions client.Options
	if config.Config.Temporal.Ca != "" && config.Config.Temporal.Cert != "" && config.Config.Temporal.Key != "" {
		if temporalClientOptions, err = temporal.GetTLSClientOption(
			config.Config.Temporal.HostPort,
			config.Config.Temporal.Namespace,
			zapadapter.NewZapAdapter(logger),
			config.Config.Temporal.Ca,
			config.Config.Temporal.Cert,
			config.Config.Temporal.Key,
			config.Config.Temporal.ServerName,
			true,
		); err != nil {
			logger.Fatal(fmt.Sprintf("Unable to get Temporal client options: %s", err))
		}
	} else {
		if temporalClientOptions, err = temporal.GetClientOption(
			config.Config.Temporal.HostPort,
			config.Config.Temporal.Namespace,
			zapadapter.NewZapAdapter(logger)); err != nil {
			logger.Fatal(fmt.Sprintf("Unable to get Temporal client options: %s", err))
		}
	}

	temporalClient, err := client.Dial(temporalClientOptions)
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to create client: %s", err))
	}
	defer temporalClient.Close()

	w := worker.New(temporalClient, connWorker.TaskQueue, worker.Options{
		MaxConcurrentActivityExecutionSize: 2,
	})

	w.RegisterWorkflow(cw.CheckWorkflow)
	w.RegisterActivity(cw.CheckActivity)
	w.RegisterWorkflow(cw.WriteWorkflow)
	w.RegisterActivity(cw.WriteActivity)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to start worker: %s", err))
	}
}
