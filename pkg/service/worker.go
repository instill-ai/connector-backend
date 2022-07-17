package service

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"

	"github.com/instill-ai/connector-backend/internal/logger"
	"github.com/instill-ai/connector-backend/internal/worker"
)

func (s *service) startCheckWorkflow(ownerPermalink string, connPermalink string, dockerRepo string, dockerImgTag string) error {

	logger, _ := logger.GetZapLogger()

	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("%s.check", connPermalink),
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
			OwnerPermalink:     ownerPermalink,
			ConnectorPermalink: connPermalink,
			ImageName:          fmt.Sprintf("%s:%s", dockerRepo, dockerImgTag),
			ContainerName:      fmt.Sprintf("%s.check", connPermalink),
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return nil
}

func (s *service) startWriteWorkflow(ownerPermalink string, connPermalink string, dockerRepo string, dockerImgTag string, cfgAbCatalog []byte, abMsgs []byte) error {

	logger, _ := logger.GetZapLogger()

	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("%s.write", connPermalink),
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
			ConnectorPermalink:       connPermalink,
			ImageName:                fmt.Sprintf("%s:%s", dockerRepo, dockerImgTag),
			ContainerName:            fmt.Sprintf("%s.write", connPermalink),
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

func (s *service) startDeleteWorkflow(connPermalink string) error {

	logger, _ := logger.GetZapLogger()

	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("%s.delete", connPermalink),
		TaskQueue: worker.TaskQueue,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		context.Background(),
		workflowOptions,
		"DeleteWorkflow",
		&worker.DeleteWorkflowParam{
			ContainerLocalFileName: connPermalink,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return nil
}
