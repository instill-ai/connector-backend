package worker

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// DeleteWorkflowParam represents the parameters for DeleteWorkflow
type DeleteWorkflowParam struct {
	ContainerLocalFileName string
}

// DeleteActivityParam represents the parameters for DeleteActivity
type DeleteActivityParam struct {
	ContainerLocalFileName string
}

func (w *worker) DeleteWorkflow(ctx workflow.Context, param *DeleteWorkflowParam) error {

	logger := workflow.GetLogger(ctx)
	logger.Info("DeleteWorkflow started")

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 120 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var result exitCode
	if err := workflow.ExecuteActivity(ctx, w.DeleteActivity, &DeleteActivityParam{
		ContainerLocalFileName: param.ContainerLocalFileName,
	}).Get(ctx, &result); err != nil {
		return temporal.NewNonRetryableApplicationError("activity failed", "ActivityError", err)
	}

	switch result {
	case exitCodeOK:
	case exitCodeError:
		logger.Error("connector container exited with errors")
	}

	logger.Info("DeleteWorkflow completed")

	return nil
}

func (w *worker) DeleteActivity(ctx context.Context, param *DeleteActivityParam) (exitCode, error) {

	logger := activity.GetLogger(ctx)
	logger.Info("Activity", "ContainerLocalFileName", param.ContainerLocalFileName)

	// Delete config local file
	configFilePath := fmt.Sprintf("%s/connector-data/config/%s.json", w.mountTargetVDP, param.ContainerLocalFileName)
	if _, err := os.Stat(configFilePath); err == nil {
		if err := os.Remove(configFilePath); err != nil {
			return exitCodeError, temporal.NewNonRetryableApplicationError(fmt.Sprintf("unable to delete config file %s", configFilePath), "DeleteContainerLocalFileError", err)
		}
	}

	// Delete catalog local file
	catalogFilePath := fmt.Sprintf("%s/connector-data/catalog/%s.json", w.mountTargetVDP, param.ContainerLocalFileName)
	if _, err := os.Stat(catalogFilePath); err == nil {
		if err := os.Remove(catalogFilePath); err != nil {
			return exitCodeError, temporal.NewNonRetryableApplicationError(fmt.Sprintf("unable to delete catalog file %s", catalogFilePath), "DeleteContainerLocalFileError", err)
		}
	}

	return exitCodeOK, nil
}
