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
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/gofrs/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/instill-ai/connector-backend/internal/resource"
	"github.com/instill-ai/connector-backend/pkg/datamodel"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

// CheckWorkflowParam represents the parameters for CheckWorkflow
type CheckWorkflowParam struct {
	OwnerPermalink     string
	ConnectorPermalink string
	ImageName          string
	ContainerName      string
}

// CheckActivityParam represents the parameters for CheckActivity
type CheckActivityParam struct {
	ImageName       string
	ContainerName   string
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
	if err := workflow.ExecuteActivity(ctx, w.CheckActivity, &CheckActivityParam{
		ImageName:       param.ImageName,
		ContainerName:   param.ContainerName,
		ConnectorConfig: dbConnector.Configuration,
	}).Get(ctx, &result); err != nil {
		if err := stopAndRemoveContainer(w.dockerClient, param.ContainerName); err != nil {
			logger.Error(fmt.Sprintf("unable to stop and remove container: %s", err))
		}
		return temporal.NewNonRetryableApplicationError("activity failed", "ActivityError", err)
	}

	if err := stopAndRemoveContainer(w.dockerClient, param.ContainerName); err != nil {
		logger.Error(fmt.Sprintf("unable to stop and remove container: %s", err))
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
	configFilePath := fmt.Sprintf("%s/connector-data/config/%s.json", w.mountTargetVDP, strings.Split(param.ContainerName, ".")[0])
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

	// Configured hostConfig:
	// https://godoc.org/github.com/docker/docker/api/types/container#HostConfig
	hostConfig := &container.HostConfig{
		RestartPolicy: container.RestartPolicy{
			Name: "no",
		},
		Mounts: []mount.Mount{
			{
				Type:   w.mountType,
				Source: w.mountSourceVDP,
				Target: w.mountTargetVDP,
			},
		},
	}

	// Configuration
	// https://godoc.org/github.com/docker/docker/api/types/container#Config
	config := &container.Config{
		Image:        param.ImageName,
		Tty:          false,
		AttachStdin:  false,
		AttachStdout: true,
		AttachStderr: true,
		Cmd: []string{
			"check",
			"--config", fmt.Sprintf("%s/connector-data/config/%s", w.mountTargetVDP, filepath.Base(configFilePath))},
	}

	// Creating the actual container. This is "nil,nil,nil" in every example.
	cont, err := w.dockerClient.ContainerCreate(
		context.Background(),
		config,
		hostConfig,
		nil,
		nil,
		param.ContainerName,
	)

	if err != nil {
		return exitCodeError, err
	}

	// Run the container
	exitCode, err := runCheckContainer(w.dockerClient, &cont)
	if err != nil {
		return exitCodeError, temporal.NewNonRetryableApplicationError("unable to run container", "DockerError", err)
	}

	return exitCode, nil
}

func runCheckContainer(cli *client.Client, cont *container.ContainerCreateCreatedBody) (code exitCode, err error) {

	// Run the actual container
	if err = cli.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{}); err != nil {
		code = exitCodeError
	}

	var statusCode int64
	statusCh, errCh := cli.ContainerWait(context.Background(), cont.ID, container.WaitConditionNotRunning)
	select {
	case err = <-errCh:
		if err != nil {
			code = exitCodeError
		}
	case status := <-statusCh:
		statusCode = status.StatusCode
	}

	out, err := cli.ContainerLogs(context.Background(), cont.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		code = exitCodeError
	}
	defer out.Close()

	var buf bytes.Buffer
	tee := io.TeeReader(out, &buf)

	if _, err = stdcopy.StdCopy(os.Stdout, os.Stderr, tee); err != nil {
		code = exitCodeError
	}

	switch statusCode {
	case 0:
		scanner := bufio.NewScanner(&buf)
		for scanner.Scan() {
			// header := [8]byte{STREAM_TYPE, 0, 0, 0, SIZE1, SIZE2, SIZE3, SIZE4}
			// STREAM_TYPE can be:
			// 0: stdin (is written on stdout)
			// 1: stdout
			// 2: stderr
			// SIZE1, SIZE2, SIZE3, SIZE4 are the four bytes of the uint32 size encoded as big endian.
			if scanner.Bytes()[0] == 1 {
				var jsonMsg map[string]interface{}
				if err := json.Unmarshal(scanner.Bytes()[8:], &jsonMsg); err == nil {
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
		}
		if err = scanner.Err(); err != nil {
			code = exitCodeError
		}
	case 1:
		code = exitCodeError
	}

	return code, nil
}
