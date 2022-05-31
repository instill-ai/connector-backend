package main

import (
	"fmt"
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/instill-ai/connector-backend/config"
	"github.com/instill-ai/connector-backend/internal/logger"
	"github.com/instill-ai/connector-backend/pkg/repository"
	"github.com/instill-ai/x/zapadapter"

	database "github.com/instill-ai/connector-backend/internal/db"
	connWorker "github.com/instill-ai/connector-backend/internal/worker"
)

func main() {

	logger, _ := logger.GetZapLogger()
	defer func() {
		// can't handle the error due to https://github.com/uber-go/zap/issues/880
		_ = logger.Sync()
	}()

	if err := config.Init(); err != nil {
		logger.Fatal(err.Error())
	}

	db := database.GetConnection()
	defer database.Close(db)

	cw := connWorker.NewWorker(repository.NewRepository(db))

	c, err := client.NewClient(client.Options{
		// ZapAdapter implements log.Logger interface and can be passed
		// to the client constructor using client using client.Options.
		Logger:   zapadapter.NewZapAdapter(logger),
		HostPort: config.Config.Temporal.ClientOptions.HostPort,
	})

	if err != nil {
		log.Fatalln("Unable to create client", err)
	}
	defer c.Close()

	w := worker.New(c, connWorker.TaskQueue, worker.Options{})

	w.RegisterWorkflow(cw.ConnectorCheckStateWorkflow)
	w.RegisterActivity(cw.ConnectorCheckStateActivity)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to start worker %s", err))
	}
}
