package worker

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
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

// CheckStateWorkflowParam represents the parameters for ConnectorCheckStateWorkflow
type CheckStateWorkflowParam struct {
	OwnerPermalink     string
	ConnectorPermalink string
	ImageName          string
	ContainerName      string
	ConnectorType      datamodel.ConnectorType
}

// CheckStateActivityParam represents the parameters for ConnectorCheckStateActivity
type CheckStateActivityParam struct {
	ImageName     string
	ContainerName string
	Config        []byte
}

type exitCode int64

const (
	exitCodeUnknown exitCode = iota
	exitCodeOK
	exitCodeError
)

// ConnectorCheckStateWorkflow is a check-state workflow definition.
func (w *worker) ConnectorCheckStateWorkflow(ctx workflow.Context, param *CheckStateWorkflowParam) error {

	logger := workflow.GetLogger(ctx)
	logger.Info("ConnectorCheckStateWorkflow started")

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
	if err := workflow.ExecuteActivity(ctx, w.ConnectorCheckStateActivity, &CheckStateActivityParam{
		ImageName:     param.ImageName,
		ContainerName: param.ContainerName,
		Config:        dbConnector.Configuration,
	}).Get(ctx, &result); err != nil {
		if err := stopAndRemoveContainer(w.dockerClient, param.ContainerName); err != nil {
			return temporal.NewNonRetryableApplicationError("unable to stop container", "DockerError", err)
		}
		return temporal.NewNonRetryableApplicationError("activity failed", "ActivityError", err)
	}

	switch result {
	case exitCodeOK:
		if err := w.repository.UpdateConnectorStateByUID(connUID, param.OwnerPermalink, param.ConnectorType, datamodel.ConnectorState(connectorPB.Connector_STATE_CONNECTED)); err != nil {
			return temporal.NewNonRetryableApplicationError("cannot update connector state by UUID", "RepositoryError", err)
		}
	case exitCodeError:
		if err := w.repository.UpdateConnectorStateByUID(connUID, param.OwnerPermalink, param.ConnectorType, datamodel.ConnectorState(connectorPB.Connector_STATE_ERROR)); err != nil {
			return temporal.NewNonRetryableApplicationError("cannot update connector state by UUID", "RepositoryError", err)
		}
	}

	logger.Info("ConnectorCheckStateWorkflow completed")

	return nil
}

// ConnectorCheckStateActivity is a check-state activity definition.
func (w *worker) ConnectorCheckStateActivity(ctx context.Context, param *CheckStateActivityParam) (exitCode, error) {

	logger := activity.GetLogger(ctx)
	logger.Info("Activity", "ImageName", param.ImageName, "ContainerName", param.ContainerName)

	// Write config into a container local file
	configFilePath := fmt.Sprintf("/tmp/vdp-data/connector-config/%s.json", param.ContainerName)
	if _, err := os.Stat(configFilePath); err != nil {
		if err := os.MkdirAll(filepath.Dir(configFilePath), os.ModePerm); err != nil {
			return exitCodeUnknown, temporal.NewNonRetryableApplicationError(fmt.Sprintf("unable to create folders for filepath %s", configFilePath), "ConfigFilePreconditionError", err)
		}
		if err := ioutil.WriteFile(configFilePath, param.Config, 0644); err != nil {
			return exitCodeUnknown, temporal.NewNonRetryableApplicationError(fmt.Sprintf("unable to write connector config file %s", configFilePath), "ConfigFilePreconditionError", err)
		}
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return exitCodeUnknown, temporal.NewNonRetryableApplicationError("unable to create docker client", "DockerError", err)
	}

	// Run the container
	exitCode, err := runContainer(cli, param.ImageName, param.ContainerName, configFilePath)
	if err != nil {
		if err := stopAndRemoveContainer(cli, param.ContainerName); err != nil {
			return exitCodeUnknown, temporal.NewNonRetryableApplicationError("unable to stop container", "DockerError", err)
		}
		return exitCodeUnknown, temporal.NewNonRetryableApplicationError("unable to run container", "DockerError", err)
	}

	// Stops and removes the container
	if err := stopAndRemoveContainer(cli, param.ContainerName); err != nil {
		return exitCodeUnknown, temporal.NewNonRetryableApplicationError("unable to remove container", "DockerError", err)
	}

	return exitCode, nil
}

func runContainer(cli *client.Client, imageName string, containerName string, configFilePath string) (exitCode, error) {

	pull, err := cli.ImagePull(context.Background(), imageName, types.ImagePullOptions{})
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
			Name: "always",
		},
		LogConfig: container.LogConfig{
			Type:   "json-file",
			Config: map[string]string{},
		},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: "/tmp/vdp-data/connector-config",
				Target: "/config",
			},
		},
	}

	// Configuration
	// https://godoc.org/github.com/docker/docker/api/types/container#Config
	config := &container.Config{
		Image:        imageName,
		Tty:          false,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{"check", "--config", fmt.Sprintf("/config/%s", filepath.Base(configFilePath))},
	}

	// Creating the actual container. This is "nil,nil,nil" in every example.
	cont, err := cli.ContainerCreate(
		context.Background(),
		config,
		hostConfig,
		nil,
		nil,
		containerName,
	)

	if err != nil {
		return exitCodeUnknown, err
	}

	// Run the actual container
	err = cli.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	if err != nil {
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

// Stop and remove a container
func stopAndRemoveContainer(client *client.Client, containerName string) error {

	if err := client.ContainerStop(context.Background(), containerName, nil); err != nil {
		return fmt.Errorf("unable to stop container %s: %s", containerName, err)
	}

	removeOptions := types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}

	if err := client.ContainerRemove(context.Background(), containerName, removeOptions); err != nil {
		return fmt.Errorf("Unable to remove container %s: %s", containerName, err)
	}

	return nil
}
