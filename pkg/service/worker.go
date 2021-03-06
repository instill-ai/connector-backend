package service

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"

	"github.com/instill-ai/connector-backend/internal/logger"
	"github.com/instill-ai/connector-backend/internal/worker"
)

func (s *service) startCheckWorkflow(ownerPermalink string, connUID string, dockerRepo string, dockerImgTag string) error {

	logger, _ := logger.GetZapLogger()

	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("%s.%d.check", connUID, time.Now().UnixNano()),
		TaskQueue: worker.TaskQueue,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		context.Background(),
		workflowOptions,
		"CheckWorkflow",
		&worker.CheckWorkflowParam{
			OwnerPermalink: ownerPermalink,
			ConnUID:        connUID,
			ImageName:      fmt.Sprintf("%s:%s", dockerRepo, dockerImgTag),
			ContainerName:  fmt.Sprintf("%s.%d.check", connUID, time.Now().UnixNano()),
			ConfigFileName: fmt.Sprintf("%s.%d", connUID, time.Now().UnixNano()),
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return nil
}

func (s *service) startWriteWorkflow(ownerPermalink string, connUID string, dockerRepo string, dockerImgTag string, cfgAbCatalog []byte, abMsgs []byte) error {

	logger, _ := logger.GetZapLogger()

	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("%s.%d.write", connUID, time.Now().UnixNano()),
		TaskQueue: worker.TaskQueue,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		context.Background(),
		workflowOptions,
		"WriteWorkflow",
		&worker.WriteWorkflowParam{
			OwnerPermalink:           ownerPermalink,
			ConnectorPermalink:       connUID,
			ImageName:                fmt.Sprintf("%s:%s", dockerRepo, dockerImgTag),
			ContainerName:            fmt.Sprintf("%s.%d.write", connUID, time.Now().UnixNano()),
			ConfigFileName:           fmt.Sprintf("%s.%d", connUID, time.Now().UnixNano()),
			CatalogFileName:          fmt.Sprintf("%s.%d", connUID, time.Now().UnixNano()),
			ConfiguredAirbyteCatalog: cfgAbCatalog,
			AirbyteMessages:          abMsgs,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return nil
}
