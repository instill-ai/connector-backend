package worker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/pkg/stdcopy"
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
		StartToCloseTimeout: 10 * time.Minute,
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
		return err
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
func (w *worker) CheckActivity(ctx context.Context, param *CheckActivityParam) (exitCode, error) {

	logger := activity.GetLogger(ctx)
	logger.Info("Activity", "ImageName", param.ImageName, "ContainerName", param.ContainerName)

	// Write config into a container local file (always overwrite)
	configFilePath := fmt.Sprintf("%s/connector-data/config/%s.json", w.mountTargetVDP, param.ConfigFileName)
	if err := os.MkdirAll(filepath.Dir(configFilePath), os.ModePerm); err != nil {
		return exitCodeError, temporal.NewNonRetryableApplicationError(fmt.Sprintf("unable to create folders for filepath %s", configFilePath), "WriteContainerLocalFileError", err)
	}
	if err := os.WriteFile(configFilePath, param.ConnectorConfig, 0644); err != nil {
		return exitCodeError, temporal.NewNonRetryableApplicationError(fmt.Sprintf("unable to write connector config file %s", configFilePath), "WriteContainerLocalFileError", err)
	}

	defer func() {
		// Delete config local file
		if _, err := os.Stat(configFilePath); err == nil {
			if err := os.Remove(configFilePath); err != nil {
				logger.Error("Activity", "ImageName", param.ImageName, "ContainerName", param.ContainerName, "Error", err)
			}
		}
	}()

	out, err := w.dockerClient.ImagePull(ctx, param.ImageName, types.ImagePullOptions{})
	if err != nil {
		return exitCodeError, err
	}
	defer out.Close()

	if _, err := io.Copy(os.Stdout, out); err != nil {
		return exitCodeError, err
	}

	resp, err := w.dockerClient.ContainerCreate(ctx,
		&container.Config{
			Image: param.ImageName,
			Tty:   false,
			Cmd: []string{
				"check",
				"--config",
				fmt.Sprintf("%s/connector-data/config/%s", w.mountTargetVDP, filepath.Base(configFilePath)),
			},
		},
		&container.HostConfig{
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeVolume,
					Source: w.mountSourceVDP,
					Target: w.mountTargetVDP,
				},
			},
		},
		nil, nil, param.ContainerName)
	if err != nil {
		return exitCodeError, err
	}

	if err := w.dockerClient.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return exitCodeError, err
	}

	statusCh, errCh := w.dockerClient.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return exitCodeError, err
		}
	case <-statusCh:
	}

	if out, err = w.dockerClient.ContainerLogs(ctx,
		resp.ID,
		types.ContainerLogsOptions{
			ShowStdout: true,
		},
	); err != nil {
		return exitCodeError, err
	}

	if err := w.dockerClient.ContainerRemove(ctx, param.ContainerName,
		types.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		}); err != nil {
		return exitCodeError, err
	}

	var bufStdOut, bufStdErr bytes.Buffer
	if _, err := stdcopy.StdCopy(&bufStdOut, &bufStdErr, out); err != nil {
		return exitCodeError, err
	}

	var teeStdOut io.Reader = strings.NewReader(bufStdOut.String())
	var teeStdErr io.Reader = strings.NewReader(bufStdErr.String())
	teeStdOut = io.TeeReader(teeStdOut, &bufStdOut)
	teeStdErr = io.TeeReader(teeStdErr, &bufStdErr)

	var byteStdOut, byteStdErr []byte
	if byteStdOut, err = io.ReadAll(teeStdOut); err != nil {
		return exitCodeError, err
	}
	if byteStdErr, err = io.ReadAll(teeStdErr); err != nil {
		return exitCodeError, err
	}

	logger.Info("Activity",
		"ImageName", param.ImageName,
		"ContainerName", param.ContainerName,
		"STDOUT", string(byteStdOut),
		"STDERR", string(byteStdErr))

	scanner := bufio.NewScanner(&bufStdOut)
	for scanner.Scan() {

		if err := scanner.Err(); err != nil {
			return exitCodeError, err
		}

		var jsonMsg map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &jsonMsg); err == nil {
			switch jsonMsg["type"] {
			case "CONNECTION_STATUS":
				switch jsonMsg["connectionStatus"].(map[string]interface{})["status"] {
				case "SUCCEEDED":
					return exitCodeOK, nil
				case "FAILED":
					return exitCodeError, nil
				default:
					return exitCodeError, fmt.Errorf("UNKNOWN STATUS")
				}
			}
		}
	}

	return exitCodeError, fmt.Errorf("unable to scan container stdout and find the connection status successfully")

}
