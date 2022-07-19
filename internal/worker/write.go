package worker

import (
	"bytes"
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
	"github.com/gofrs/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/instill-ai/connector-backend/internal/logger"
	"github.com/instill-ai/connector-backend/internal/resource"
)

// WriteWorkflowParam represents the parameters for WriteWorkflow
type WriteWorkflowParam struct {
	OwnerPermalink           string
	ConnectorPermalink       string
	ImageName                string
	ContainerName            string
	ConfiguredAirbyteCatalog []byte
	AirbyteMessages          []byte
}

// WriteActivityParam represents the parameters for WriteActivity
type WriteActivityParam struct {
	ImageName                string
	ContainerName            string
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
		ConnectorConfig:          dbConnector.Configuration,
		ConfiguredAirbyteCatalog: param.ConfiguredAirbyteCatalog,
		AirbyteMessages:          param.AirbyteMessages,
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
	case exitCodeError:
		logger.Error("connector container exited with errors")
	}

	logger.Info("WriteWorkflow completed")

	return nil
}

func (w *worker) WriteActivity(ctx context.Context, param *WriteActivityParam) (exitCode, error) {

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

	// Write catalog into a container local file (always overwrite)
	catalogFilePath := fmt.Sprintf("%s/connector-data/catalog/%s.json", w.mountTargetVDP, strings.Split(param.ContainerName, ".")[0])
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

	// Configured hostConfig
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
			{
				Type:   w.mountType,
				Source: w.mountSourceAirbyte,
				Target: w.mountTargetAirbyte,
			},
		},
	}

	// Configuration
	config := &container.Config{
		Image:        param.ImageName,
		AttachStderr: true,
		AttachStdin:  true,
		AttachStdout: true,
		Tty:          true,
		OpenStdin:    true,
		Cmd: []string{
			"write",
			"--config", fmt.Sprintf("%s/connector-data/config/%s", w.mountTargetVDP, filepath.Base(configFilePath)),
			"--catalog", fmt.Sprintf("%s/connector-data/catalog/%s", w.mountTargetVDP, filepath.Base(catalogFilePath)),
		},
	}

	// Creating the actual cont
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
	exitCode, err := runWriteContainer(w.dockerClient, &cont, param.AirbyteMessages)
	if err != nil {
		return exitCode, temporal.NewNonRetryableApplicationError("unable to run container", "DockerError", err)
	}

	return exitCode, nil
}

func runWriteContainer(cli *client.Client, cont *container.ContainerCreateCreatedBody, abMsgs []byte) (code exitCode, err error) {

	logger, _ := logger.GetZapLogger()

	// Run the actual container
	if err = cli.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{}); err != nil {
		code = exitCodeError
	}

	resp, err := cli.ContainerAttach(context.Background(), cont.ID, types.ContainerAttachOptions{
		Stdin:  true,
		Stdout: true,
		Stderr: true,
		Stream: true,
	})
	if err != nil {
		code = exitCodeError
	}

	go func() {
		if _, err := io.Copy(os.Stdout, resp.Reader); err != nil {
			logger.Error(err.Error())
		}
	}()

	go func() {
		if _, err := io.Copy(os.Stderr, resp.Reader); err != nil {
			logger.Error(err.Error())
		}
	}()

	// Append Ctrl+D (EOT)
	abMsgs = append(abMsgs, 4)
	go func() {
		if _, err := io.Copy(resp.Conn, bytes.NewReader(abMsgs)); err != nil {
			logger.Error(err.Error())
		}
	}()

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

	switch statusCode {
	case 0:
		code = exitCodeOK
	case 1:
		code = exitCodeError
	}

	return code, nil
}
