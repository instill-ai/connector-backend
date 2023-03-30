package service

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/client"

	"github.com/instill-ai/connector-backend/pkg/logger"
	"github.com/instill-ai/connector-backend/pkg/worker"
)

func (s *service) startCheckWorkflow(ownerPermalink string, connUID string, dockerRepo string, dockerImgTag string) error {

	logger, _ := logger.GetZapLogger()

	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("%s.%d.check", connUID, time.Now().UnixNano()),
		TaskQueue: worker.TaskQueue,
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		context.Background(),
		workflowOptions,
		"CheckWorkflow",
		&worker.CheckWorkflowParam{
			OwnerPermalink: ownerPermalink,
			ConnUID:        connUID,
			ImageName:      fmt.Sprintf("%s:%s", dockerRepo, dockerImgTag),
			ContainerName:  workflowOptions.ID,
			ConfigFileName: workflowOptions.ID,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return nil
}

func (s *service) startWriteWorkflow(ownerPermalink string, connUID string,
	dockerRepo string, dockerImgTag string, pipeline string, indices []string, cfgAbCatalog []byte, abMsgs []byte) error {

	logger, _ := logger.GetZapLogger()

	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("%s.%d.write", connUID, time.Now().UnixNano()),
		TaskQueue: worker.TaskQueue,
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		context.Background(),
		workflowOptions,
		"WriteWorkflow",
		&worker.WriteWorkflowParam{
			OwnerPermalink:           ownerPermalink,
			ConnectorPermalink:       connUID,
			ImageName:                fmt.Sprintf("%s:%s", dockerRepo, dockerImgTag),
			ContainerName:            workflowOptions.ID,
			ConfigFileName:           workflowOptions.ID,
			CatalogFileName:          workflowOptions.ID,
			Pipeline:                 pipeline,
			Indices:                  indices,
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
