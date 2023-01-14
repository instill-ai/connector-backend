package worker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

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
	Pipeline                 string
	Indices                  []string
	ConfiguredAirbyteCatalog []byte
	AirbyteMessages          []byte
}

// WriteActivityParam represents the parameters for WriteActivity
type WriteActivityParam struct {
	ImageName                string
	ContainerName            string
	ConfigFileName           string
	CatalogFileName          string
	Pipeline                 string
	Indices                  []string
	ConnectorConfig          []byte
	ConfiguredAirbyteCatalog []byte
	AirbyteMessages          []byte
}

func (w *worker) WriteWorkflow(ctx workflow.Context, param *WriteWorkflowParam) error {

	logger := workflow.GetLogger(ctx)
	logger.Info(fmt.Sprintf("WriteWorkflow started for records %v in pipeline %s", param.Indices, param.Pipeline))

	lao := workflow.LocalActivityOptions{
		ScheduleToCloseTimeout: 10 * time.Minute, // In case if image pulling is required
	}
	ctx = workflow.WithLocalActivityOptions(ctx, lao)

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

	// Local activity is used together with the local cache to flag dispatched container run
	if err := workflow.ExecuteLocalActivity(ctx, w.WriteActivity, &WriteActivityParam{
		ImageName:                param.ImageName,
		ContainerName:            param.ContainerName,
		ConfigFileName:           param.ConfigFileName,
		CatalogFileName:          param.CatalogFileName,
		Pipeline:                 param.Pipeline,
		Indices:                  param.Indices,
		ConnectorConfig:          dbConnector.Configuration,
		ConfiguredAirbyteCatalog: param.ConfiguredAirbyteCatalog,
		AirbyteMessages:          param.AirbyteMessages,
	}).Get(ctx, &result); err != nil {
		return err
	}

	logger.Info("WriteWorkflow completed")

	// Delete the cache entry only after the workflow completed
	if err := w.cache.Delete(param.ContainerName); err != nil {
		logger.Error(err.Error())
	}

	return nil
}

func (w *worker) WriteActivity(ctx context.Context, param *WriteActivityParam) (exitCode, error) {

	logger := activity.GetLogger(ctx)
	logger.Info("Activity", "ImageName", param.ImageName, "ContainerName", param.ContainerName)

	// If there is already a container run dispatched in the previous attempt, return exitCodeOK directly
	if _, err := w.cache.Get(param.ContainerName); err == nil {
		return exitCodeOK, nil
	}

	// Write config into a container local file (always overwrite)
	configFilePath := fmt.Sprintf("%s/connector-data/config/%s.json", w.mountTargetVDP, param.ConfigFileName)
	if err := os.MkdirAll(filepath.Dir(configFilePath), os.ModePerm); err != nil {
		return exitCodeError, temporal.NewNonRetryableApplicationError(fmt.Sprintf("unable to create folders for filepath %s", configFilePath), "WriteContainerLocalFileError", err)
	}
	if err := os.WriteFile(configFilePath, param.ConnectorConfig, 0644); err != nil {
		return exitCodeError, temporal.NewNonRetryableApplicationError(fmt.Sprintf("unable to write connector config file %s", configFilePath), "WriteContainerLocalFileError", err)
	}

	// Write catalog into a container local file (always overwrite)
	catalogFilePath := fmt.Sprintf("%s/connector-data/catalog/%s.json", w.mountTargetVDP, param.CatalogFileName)
	if err := os.MkdirAll(filepath.Dir(catalogFilePath), os.ModePerm); err != nil {
		return exitCodeError, temporal.NewNonRetryableApplicationError(fmt.Sprintf("unable to create folders for filepath %s", catalogFilePath), "WriteContainerLocalFileError", err)
	}
	if err := os.WriteFile(catalogFilePath, param.ConfiguredAirbyteCatalog, 0644); err != nil {
		return exitCodeError, temporal.NewNonRetryableApplicationError(fmt.Sprintf("unable to write connector catalog file %s", catalogFilePath), "WriteContainerLocalFileError", err)
	}

	defer func() {
		// Delete config local file
		if _, err := os.Stat(configFilePath); err == nil {
			if err := os.Remove(configFilePath); err != nil {
				logger.Error("Activity", "ImageName", param.ImageName, "ContainerName", param.ContainerName, "Error", err)
			}
		}

		// Delete catalog local file
		if _, err := os.Stat(catalogFilePath); err == nil {
			if err := os.Remove(catalogFilePath); err != nil {
				logger.Error("Activity", "ImageName", param.ImageName, "ContainerName", param.ContainerName, "Error", err)
			}
		}
	}()

	// Check image existing or otherwise pull image
	runCmd := exec.Command(
		"docker",
		"inspect",
		"--type=image",
		param.ImageName,
	)

	if err := runCmd.Run(); err != nil {

		runCmd = exec.CommandContext(ctx,
			"docker",
			"pull",
			param.ImageName,
		)

		out, err := runCmd.CombinedOutput()
		if err != nil {
			return exitCodeError, err
		}

		logger.Info("Activity", "ImageName", param.ImageName, "ContainerName", param.ContainerName, "Pipeline", param.Pipeline, "Indices", param.Indices, "STDOUT", string(out))

	}

	// Launch airbyte container to write data in to the destination
	runCmd = exec.CommandContext(ctx,
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

	stdin, err := runCmd.StdinPipe()
	if err != nil {
		return exitCodeError, err
	}

	param.AirbyteMessages = append(param.AirbyteMessages, 4)
	if _, err := stdin.Write(param.AirbyteMessages); err != nil {
		return exitCodeError, err
	}

	if err := stdin.Close(); err != nil {
		return exitCodeError, err
	}

	// Set cache flag (empty value is fine since we need only the entry record)
	if err := w.cache.Set(param.ContainerName, []byte{}); err != nil {
		logger.Error(err.Error())
	}

	// Run exec command and get the combined stdout and stderr
	out, err := runCmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return exitCodeError, fmt.Errorf("container run timed out")
	}

	if err != nil {
		return exitCodeError, err
	}

	logger.Info("Activity", "ImageName", param.ImageName, "ContainerName", param.ContainerName, "Pipeline", param.Pipeline, "Indices", param.Indices, "STDOUT", string(out))

	return exitCodeOK, nil

}
