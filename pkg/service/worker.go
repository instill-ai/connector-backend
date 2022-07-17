package service

import (
	"context"
	"fmt"
	"strings"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"

	"github.com/instill-ai/connector-backend/internal/logger"
	"github.com/instill-ai/connector-backend/internal/worker"
	"github.com/instill-ai/connector-backend/pkg/datamodel"
)

func (s *service) startCheckWorkflow(ownerRscName string, ownerPermalink string, connRscName string, connPermalink string, connType datamodel.ConnectorType, dockerRepo string, dockerImgTag string) error {

	logger, _ := logger.GetZapLogger()

	workflowOptions := client.StartWorkflowOptions{
		ID:        strings.ReplaceAll(fmt.Sprintf("%s.%s.check", ownerRscName, connRscName), "/", "."),
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
			ContainerName:      strings.ReplaceAll(fmt.Sprintf("%s.%s.check", ownerRscName, connRscName), "/", "."),
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return nil
}

func (s *service) startWriteWorkflow(ownerRscName string, ownerPermalink string, connRscName string, connPermalink string, dockerRepo string, dockerImgTag string, cfgAbCatalog []byte, abMsgs []byte) error {

	logger, _ := logger.GetZapLogger()

	workflowOptions := client.StartWorkflowOptions{
		ID:        strings.ReplaceAll(fmt.Sprintf("%s.%s.write", ownerRscName, connRscName), "/", "."),
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
			ContainerName:            strings.ReplaceAll(fmt.Sprintf("%s.%s.write", ownerRscName, connRscName), "/", "."),
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

func (s *service) startDeleteWorkflow(ownerRscName string, connRscName string) error {

	logger, _ := logger.GetZapLogger()

	workflowOptions := client.StartWorkflowOptions{
		ID:        strings.ReplaceAll(fmt.Sprintf("%s.%s.delete", ownerRscName, connRscName), "/", "."),
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
			ContainerLocalFileName: strings.ReplaceAll(fmt.Sprintf("%s.%s", ownerRscName, connRscName), "/", "."),
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return nil
}
