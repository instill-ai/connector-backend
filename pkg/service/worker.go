package service

import (
	"context"
	"fmt"
	"strings"

	"go.temporal.io/sdk/client"

	"github.com/instill-ai/connector-backend/internal/logger"
	"github.com/instill-ai/connector-backend/internal/worker"
	"github.com/instill-ai/connector-backend/pkg/datamodel"
)

func (s *service) startCheckStateWorkflow(ownerRscName string, ownerPermalink string, connID string, connType datamodel.ConnectorType, dockerRepo string, dockerImgTag string) error {

	logger, _ := logger.GetZapLogger()

	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("%s.%s.%s:%s", ownerRscName, connID, dockerRepo, dockerImgTag),
		TaskQueue: worker.TaskQueue,
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		context.Background(),
		workflowOptions,
		"ConnectorCheckStateWorkflow",
		&worker.CheckStateWorkflowParam{
			ID:             connID,
			ImageName:      fmt.Sprintf("%s:%s", dockerRepo, dockerImgTag),
			ContainerName:  fmt.Sprintf("%s.%s", strings.ReplaceAll(ownerRscName, "/", "."), connID),
			ConnectorType:  connType,
			OwnerPermalink: ownerPermalink,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("Unable to execute workflow: %s", err.Error()))
		return err
	}

	logger.Info(fmt.Sprintf("Started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return nil
}
