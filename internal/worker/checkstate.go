package worker

import (
	"context"
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

// CheckStateWorkflowParam represents the parameters for CheckStateWorkflow
type CheckStateWorkflowParam struct {
	OwnerPermalink     string
	ConnectorPermalink string
	ImageName          string
	ContainerName      string
	ConnectorType      datamodel.ConnectorType
}

// CheckStateActivityParam represents the parameters for CheckStateActivity
type CheckStateActivityParam struct {
	ImageName       string
	ContainerName   string
	ConnectorConfig []byte
}

// CheckStateWorkflow is a check-state workflow definition.
func (w *worker) CheckStateWorkflow(ctx workflow.Context, param *CheckStateWorkflowParam) error {

	logger := workflow.GetLogger(ctx)
	logger.Info("CheckStateWorkflow started")

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

	dbConnector, err := w.repository.GetConnectorByUID(connUID, param.OwnerPermalink, param.ConnectorType, false)
	if err != nil {
		return temporal.NewNonRetryableApplicationError("cannot get the connector", "RepositoryError", err)
	}

	var result exitCode
	if err := workflow.ExecuteActivity(ctx, w.CheckStateActivity, &CheckStateActivityParam{
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
		if err := w.repository.UpdateConnectorStateByUID(connUID, param.OwnerPermalink, param.ConnectorType, datamodel.ConnectorState(connectorPB.Connector_STATE_CONNECTED)); err != nil {
			return temporal.NewNonRetryableApplicationError("cannot update connector state by UUID", "RepositoryError", err)
		}
	case exitCodeError:
		logger.Error("connector container exited with errors")
		if err := w.repository.UpdateConnectorStateByUID(connUID, param.OwnerPermalink, param.ConnectorType, datamodel.ConnectorState(connectorPB.Connector_STATE_ERROR)); err != nil {
			return temporal.NewNonRetryableApplicationError("cannot update connector state by UUID", "RepositoryError", err)
		}
	}

	logger.Info("CheckStateWorkflow completed")

	return nil
}

// CheckStateActivity is a check-state activity definition.
func (w *worker) CheckStateActivity(ctx context.Context, param *CheckStateActivityParam) (exitCode, error) {

	logger := activity.GetLogger(ctx)
	logger.Info("Activity", "ImageName", param.ImageName, "ContainerName", param.ContainerName)

	// Write config into a container local file
	configFilePath := fmt.Sprintf("%s/connector-data/config/%s.json", w.mountTargetVDP, strings.ReplaceAll(param.ContainerName, ".check", ""))
	if _, err := os.Stat(configFilePath); err != nil {
		if err := os.MkdirAll(filepath.Dir(configFilePath), os.ModePerm); err != nil {
			return exitCodeUnknown, temporal.NewNonRetryableApplicationError(fmt.Sprintf("unable to create folders for filepath %s", configFilePath), "WriteContainerLocalFileError", err)
		}
		if err := ioutil.WriteFile(configFilePath, param.ConnectorConfig, 0644); err != nil {
			return exitCodeUnknown, temporal.NewNonRetryableApplicationError(fmt.Sprintf("unable to write connector config file %s", configFilePath), "WriteContainerLocalFileError", err)
		}
	}

	pull, err := w.dockerClient.ImagePull(context.Background(), param.ImageName, types.ImagePullOptions{})
	if err != nil {
		return exitCodeUnknown, err
	}
	defer pull.Close()

	if _, err := io.Copy(os.Stdout, pull); err != nil {
		return exitCodeUnknown, err
	}

	// Configured hostConfig:
	// https://godoc.org/github.com/docker/docker/api/types/container#HostConfig
	hostConfig := &container.HostConfig{
		RestartPolicy: container.RestartPolicy{
			Name: "no",
		},
		LogConfig: container.LogConfig{
			Type:   "json-file",
			Config: map[string]string{},
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
		AttachStdin:  true,
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
		return exitCodeUnknown, err
	}

	// Run the container
	exitCode, err := runCheckStateContainer(w.dockerClient, &cont)
	if err != nil {
		return exitCodeUnknown, temporal.NewNonRetryableApplicationError("unable to run container", "DockerError", err)
	}

	return exitCode, nil
}

func runCheckStateContainer(cli *client.Client, cont *container.ContainerCreateCreatedBody) (exitCode, error) {

	// Run the actual container
	if err := cli.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{}); err != nil {
		return exitCodeUnknown, err
	}

	var statusCode int64
	statusCh, errCh := cli.ContainerWait(context.Background(), cont.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return exitCodeUnknown, err
		}
	case status := <-statusCh:
		statusCode = status.StatusCode
	}

	out, err := cli.ContainerLogs(context.Background(), cont.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return exitCodeUnknown, err
	}
	defer out.Close()

	if _, err := stdcopy.StdCopy(os.Stdout, os.Stderr, out); err != nil {
		return exitCodeUnknown, err
	}

	switch statusCode {
	case 0:
		return exitCodeOK, nil
	case 1:
		return exitCodeError, nil
	}

	return exitCodeUnknown, nil
}
