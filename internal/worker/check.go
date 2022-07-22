package worker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
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

	"github.com/instill-ai/connector-backend/pkg/datamodel"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

// CheckWorkflowParam represents the parameters for CheckWorkflow
type CheckWorkflowParam struct {
	OwnerPermalink string
	ConnUID        string
	ImageName      string
	ContainerName  string
	ConfigFileName string
}

// CheckActivityParam represents the parameters for CheckActivity
type CheckActivityParam struct {
	ImageName       string
	ContainerName   string
	ConfigFileName  string
	ConnectorConfig []byte
}

// CheckWorkflow is a check workflow definition.
func (w *worker) CheckWorkflow(ctx workflow.Context, param *CheckWorkflowParam) error {

	logger := workflow.GetLogger(ctx)
	logger.Info("CheckWorkflow started")

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 120 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	connUID, err := uuid.FromString(param.ConnUID)
	if err != nil {
		return temporal.NewNonRetryableApplicationError("unable to get the connector UUID", "ParsingError", err)
	}

	dbConnector, err := w.repository.GetConnectorByUID(connUID, param.OwnerPermalink, false)
	if err != nil {
		return temporal.NewNonRetryableApplicationError("cannot get the connector", "RepositoryError", err)
	}

	var result exitCode
	if err := workflow.ExecuteActivity(ctx, w.CheckActivity, &CheckActivityParam{
		ImageName:       param.ImageName,
		ContainerName:   param.ContainerName,
		ConfigFileName:  param.ConfigFileName,
		ConnectorConfig: dbConnector.Configuration,
	}).Get(ctx, &result); err != nil {
		return temporal.NewNonRetryableApplicationError("activity failed", "ActivityError", err)
	}

	switch result {
	case exitCodeOK:
		if err := w.repository.UpdateConnectorStateByUID(connUID, param.OwnerPermalink, datamodel.ConnectorState(connectorPB.Connector_STATE_CONNECTED)); err != nil {
			return temporal.NewNonRetryableApplicationError("cannot update connector state by UUID", "RepositoryError", err)
		}
	case exitCodeError:
		logger.Error("connector container exited with errors")
		if err := w.repository.UpdateConnectorStateByUID(connUID, param.OwnerPermalink, datamodel.ConnectorState(connectorPB.Connector_STATE_ERROR)); err != nil {
			return temporal.NewNonRetryableApplicationError("cannot update connector state by UUID", "RepositoryError", err)
		}
	}

	logger.Info("CheckWorkflow completed")

	return nil
}

// CheckActivity is a check activity definition.
func (w *worker) CheckActivity(ctx context.Context, param *CheckActivityParam) (code exitCode, err error) {

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
		"--rm",
		"--restart", "no",
		"--name", param.ContainerName,
		param.ImageName,
		"check",
		"--config", fmt.Sprintf("%s/connector-data/config/%s", w.mountTargetVDP, filepath.Base(configFilePath)),
	)

	var out bytes.Buffer
	runCmd.Stdout = &out
	runCmd.Stderr = &out

	buf := bytes.Buffer{}
	tee := io.TeeReader(&out, &buf)

	if err := runCmd.Run(); err != nil {
		logger.Error("Activity", "ImageName", param.ImageName, "ContainerName", param.ContainerName, "Error", err)
	}

	scanner := bufio.NewScanner(tee)
	for scanner.Scan() {

		if err = scanner.Err(); err != nil {
			code = exitCodeError
			break
		}

		var jsonMsg map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &jsonMsg); err == nil {
			switch jsonMsg["type"] {
			case "CONNECTION_STATUS":
				switch jsonMsg["connectionStatus"].(map[string]interface{})["status"] {
				case "SUCCEEDED":
					code = exitCodeOK
				case "FAILED":
					code = exitCodeError
				}
			}
		}

	}

	logger.Info("Activity", "ImageName", param.ImageName, "ContainerName", param.ContainerName, "STDOUT", buf.String())

	// Delete config local file
	if _, err := os.Stat(configFilePath); err == nil {
		if err := os.Remove(configFilePath); err != nil {
			logger.Info("Activity", "ImageName", param.ImageName, "ContainerName", param.ContainerName, "Error", err)
			code = exitCodeError
		}
	}

	return code, nil

}
