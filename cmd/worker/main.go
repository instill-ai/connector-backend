package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	dockerclient "github.com/docker/docker/client"

	"github.com/instill-ai/connector-backend/config"
	"github.com/instill-ai/connector-backend/pkg/logger"
	"github.com/instill-ai/connector-backend/pkg/repository"
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

	dc, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		logger.Error(err.Error())
	}
	defer dc.Close()

	clientNamespace, err := client.NewNamespaceClient(client.Options{
		HostPort: config.Config.Temporal.ClientOptions.HostPort,
	})
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to create namespace client: %s", err))
	}
	defer clientNamespace.Close()

	retention := time.Duration(72 * time.Hour)
	if err = clientNamespace.Register(context.Background(), &workflowservice.RegisterNamespaceRequest{
		Namespace:                        connWorker.Namespace,
		Description:                      "For workflows triggered in the connector-backend",
		OwnerEmail:                       "infra@instill.tech",
		WorkflowExecutionRetentionPeriod: &retention,
	}); err != nil {
		if _, ok := err.(*serviceerror.NamespaceAlreadyExists); !ok {
			logger.Error(fmt.Sprintf("Unable to register namespace: %s", err))
		}
	}

	cw := connWorker.NewWorker(repository.NewRepository(db), dc)

	c, err := client.Dial(client.Options{
		// ZapAdapter implements log.Logger interface and can be passed
		// to the client constructor using client using client.Options.
		Logger:    zapadapter.NewZapAdapter(logger),
		HostPort:  config.Config.Temporal.ClientOptions.HostPort,
		Namespace: connWorker.Namespace,
	})

	// Note that Namespace registration using this API takes up to 10 seconds to complete.
	// Ensure to wait for this registration to complete before starting the Workflow Execution against the Namespace.
	time.Sleep(time.Second * 10)

	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to create client: %s", err))
	}
	defer c.Close()

	w := worker.New(c, connWorker.TaskQueue, worker.Options{})

	w.RegisterWorkflow(cw.WriteWorkflow)
	w.RegisterActivity(cw.WriteActivity)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to start worker: %s", err))
	}
}
