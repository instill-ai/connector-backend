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

func (s *service) startCheckStateWorkflow(ownerRscName string, ownerPermalink string, connRscName string, connPermalink string, connType datamodel.ConnectorType, dockerRepo string, dockerImgTag string) error {

	logger, _ := logger.GetZapLogger()

	workflowOptions := client.StartWorkflowOptions{
		ID:        strings.ReplaceAll(fmt.Sprintf("%s.%s", ownerRscName, connRscName), "/", "."),
		TaskQueue: worker.TaskQueue,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		context.Background(),
		workflowOptions,
		"ConnectorCheckStateWorkflow",
		&worker.CheckStateWorkflowParam{
			OwnerPermalink:     ownerPermalink,
			ConnectorPermalink: connPermalink,
			ConnectorType:      connType,
			ImageName:          fmt.Sprintf("%s:%s", dockerRepo, dockerImgTag),
			ContainerName:      strings.ReplaceAll(fmt.Sprintf("%s.%s", ownerRscName, connRscName), "/", "."),
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return nil
}
