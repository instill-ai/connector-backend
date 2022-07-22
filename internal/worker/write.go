package worker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/gofrs/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/instill-ai/connector-backend/internal/resource"
)

// WriteWorkflowParam represents the parameters for WriteWorkflow
type WriteWorkflowParam struct {
	OwnerPermalink           string
	ConnectorPermalink       string
	ImageName                string
	ContainerName            string
	ConfigFileName           string
	CatalogFileName          string
	ConfiguredAirbyteCatalog []byte
	AirbyteMessages          []byte
}

// WriteActivityParam represents the parameters for WriteActivity
type WriteActivityParam struct {
	ImageName                string
	ContainerName            string
	ConfigFileName           string
	CatalogFileName          string
	ConnectorConfig          []byte
	ConfiguredAirbyteCatalog []byte
	AirbyteMessages          []byte
}

func (w *worker) WriteWorkflow(ctx workflow.Context, param *WriteWorkflowParam) error {

	logger := workflow.GetLogger(ctx)
	logger.Info("WriteWorkflow started")

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 120 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	connUIDStr, err := resource.GetPermalinkUID(param.ConnectorPermalink)
	if err != nil {
		return temporal.NewNonRetryableApplicationError("unable to get the connector UUID", "ParsingError", err)
	}

	connUID, err := uuid.FromString(connUIDStr)
	if err != nil {
		return temporal.NewNonRetryableApplicationError("unable to get the connector UUID", "ParsingError", err)
	}

	dbConnector, err := w.repository.GetConnectorByUID(connUID, param.OwnerPermalink, false)
	if err != nil {
		return temporal.NewNonRetryableApplicationError("cannot get the connector", "RepositoryError", err)
	}

	var result exitCode
	if err := workflow.ExecuteActivity(ctx, w.WriteActivity, &WriteActivityParam{
		ImageName:                param.ImageName,
		ContainerName:            param.ContainerName,
		ConfigFileName:           param.ConfigFileName,
		CatalogFileName:          param.CatalogFileName,
		ConnectorConfig:          dbConnector.Configuration,
		ConfiguredAirbyteCatalog: param.ConfiguredAirbyteCatalog,
		AirbyteMessages:          param.AirbyteMessages,
	}).Get(ctx, &result); err != nil {
		return temporal.NewNonRetryableApplicationError("activity failed", "ActivityError", err)
	}

	logger.Info("WriteWorkflow completed")

	return nil
}

func (w *worker) WriteActivity(ctx context.Context, param *WriteActivityParam) (code exitCode, err error) {

	logger := activity.GetLogger(ctx)
	logger.Info("Activity", "ImageName", param.ImageName, "ContainerName", param.ContainerName)

	// Write config into a container local file (always overwrite)
	configFilePath := fmt.Sprintf("%s/connector-data/config/%s.json", w.mountTargetVDP, param.ConfigFileName)
	if err := os.MkdirAll(filepath.Dir(configFilePath), os.ModePerm); err != nil {
		return exitCodeError, temporal.NewNonRetryableApplicationError(fmt.Sprintf("unable to create folders for filepath %s", configFilePath), "WriteContainerLocalFileError", err)
	}
	if err := ioutil.WriteFile(configFilePath, param.ConnectorConfig, 0644); err != nil {
		return exitCodeError, temporal.NewNonRetryableApplicationError(fmt.Sprintf("unable to write connector config file %s", configFilePath), "WriteContainerLocalFileError", err)
	}

	// Write catalog into a container local file (always overwrite)
	catalogFilePath := fmt.Sprintf("%s/connector-data/catalog/%s.json", w.mountTargetVDP, param.CatalogFileName)
	if err := os.MkdirAll(filepath.Dir(catalogFilePath), os.ModePerm); err != nil {
		return exitCodeError, temporal.NewNonRetryableApplicationError(fmt.Sprintf("unable to create folders for filepath %s", catalogFilePath), "WriteContainerLocalFileError", err)
	}
	if err := ioutil.WriteFile(catalogFilePath, param.ConfiguredAirbyteCatalog, 0644); err != nil {
		return exitCodeError, temporal.NewNonRetryableApplicationError(fmt.Sprintf("unable to write connector catalog file %s", catalogFilePath), "WriteContainerLocalFileError", err)
	}

	// Pull image
	pull, err := w.dockerClient.ImagePull(context.Background(), param.ImageName, types.ImagePullOptions{})
	if err != nil {
		return exitCodeError, err
	}
	defer pull.Close()

	if _, err := io.Copy(os.Stdout, pull); err != nil {
		return exitCodeError, err
	}

	runCmd := exec.Command(
		"docker",
		"run",
		"-i",
		"-v", fmt.Sprintf("%s:%s", w.mountSourceVDP, w.mountTargetVDP),
		"-v", fmt.Sprintf("%s:%s", w.mountSourceAirbyte, w.mountTargetAirbyte),
		"--rm",
		"--restart", "no",
		"--name", param.ContainerName,
		param.ImageName,
		"write",
		"--config", fmt.Sprintf("%s/connector-data/config/%s", w.mountTargetVDP, filepath.Base(configFilePath)),
		"--catalog", fmt.Sprintf("%s/connector-data/catalog/%s", w.mountTargetVDP, filepath.Base(catalogFilePath)),
	)

	runCmd.Stdout = bytes.NewBuffer(nil)

	stdin, err := runCmd.StdinPipe()
	if err != nil {
		logger.Error("Activity", "ImageName", param.ImageName, "ContainerName", param.ContainerName, "Error", err)
		return exitCodeError, err
	}

	param.AirbyteMessages = append(param.AirbyteMessages, 4)
	if _, err := stdin.Write(param.AirbyteMessages); err != nil {
		logger.Error("Activity", "ImageName", param.ImageName, "ContainerName", param.ContainerName, "Error", err)
	}

	if err := stdin.Close(); err != nil {
		logger.Error("Activity", "ImageName", param.ImageName, "ContainerName", param.ContainerName, "Error", err)
	}

	if err := runCmd.Run(); err != nil {
		code = exitCodeError
		logger.Error("Activity", "ImageName", param.ImageName, "ContainerName", param.ContainerName, "Error", err)
	} else {
		code = exitCodeOK
	}

	logger.Info("Activity", "ImageName", param.ImageName, "ContainerName", param.ContainerName, "STDOUT", runCmd.Stdout)

	// Delete config local file
	if _, err := os.Stat(configFilePath); err == nil {
		if err := os.Remove(configFilePath); err != nil {
			logger.Info("Activity", "ImageName", param.ImageName, "ContainerName", param.ContainerName, "Error", err)
			code = exitCodeError
		}
	}

	// Delete catalog local file
	if _, err := os.Stat(catalogFilePath); err == nil {
		if err := os.Remove(catalogFilePath); err != nil {
			logger.Info("Activity", "ImageName", param.ImageName, "ContainerName", param.ContainerName, "Error", err)
			code = exitCodeError
		}
	}

	return code, nil

}
