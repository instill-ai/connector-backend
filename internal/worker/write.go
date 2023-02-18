package worker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/gofrs/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/instill-ai/connector-backend/config"
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
			Image:        param.ImageName,
			AttachStdin:  true,
			AttachStdout: true,
			OpenStdin:    true,
			StdinOnce:    true,
			Tty:          true,
			Cmd: []string{
				"write",
				"--config",
				fmt.Sprintf("%s/connector-data/config/%s", w.mountTargetVDP, filepath.Base(configFilePath)),
				"--catalog",
				fmt.Sprintf("%s/connector-data/catalog/%s", w.mountTargetVDP, filepath.Base(catalogFilePath)),
			},
		},
		&container.HostConfig{
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeVolume,
					Source: config.Config.Worker.MountSource.VDP,
					Target: "/vdp",
				},
				{
					Type:   mount.TypeVolume,
					Source: config.Config.Worker.MountSource.Airbyte,
					Target: "/local",
				},
			},
		},
		nil, nil, param.ContainerName)
	if err != nil {
		return exitCodeError, err
	}

	hijackedResp, err := w.dockerClient.ContainerAttach(ctx, resp.ID, types.ContainerAttachOptions{
		Stdout: true,
		Stdin:  true,
		Stream: true,
	})
	if err != nil {
		logger.Error(err.Error())
	}

	// need to append "\n" and "ctrl+D" at the end of the input message
	_, err = hijackedResp.Conn.Write(append(param.AirbyteMessages, 10, 4))
	if err != nil {
		logger.Error(err.Error())
	}

	if err := w.dockerClient.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return exitCodeError, err
	}

	var bufStdOut bytes.Buffer
	if _, err := bufStdOut.ReadFrom(hijackedResp.Reader); err != nil {
		return exitCodeError, err
	}

	if err := w.dockerClient.ContainerRemove(ctx, resp.ID,
		types.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		}); err != nil {
		return exitCodeError, err
	}

	// Set cache flag (empty value is fine since we need only the entry record)
	if err := w.cache.Set(param.ContainerName, []byte{}); err != nil {
		logger.Error(err.Error())
	}

	logger.Info("Activity",
		"ImageName", param.ImageName,
		"ContainerName", param.ContainerName,
		"Pipeline", param.Pipeline,
		"Indices", param.Indices,
		"STDOUT", bufStdOut.String())

	return exitCodeOK, nil

}
