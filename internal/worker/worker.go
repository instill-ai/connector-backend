package worker

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"go.temporal.io/sdk/workflow"

	"github.com/instill-ai/connector-backend/config"
	"github.com/instill-ai/connector-backend/internal/logger"
	"github.com/instill-ai/connector-backend/pkg/repository"
)

// TaskQueue is the task queue name for connector-backend
const TaskQueue = "connector-backend"

type exitCode int64

const (
	exitCodeUnknown exitCode = iota
	exitCodeOK
	exitCodeError
)

// Worker interface
type Worker interface {
	CheckStateWorkflow(ctx workflow.Context, param *CheckStateWorkflowParam) error
	CheckStateActivity(ctx context.Context, param *CheckStateActivityParam) (exitCode, error)
	WriteDestinationWorkflow(ctx workflow.Context, param *WriteDestinationWorkflowParam) error
	WriteDestinationActivity(ctx context.Context, param *WriteDestinationActivityParam) (exitCode, error)
}

// worker represents resources required to run Temporal workflow and activity
type worker struct {
	repository         repository.Repository
	dockerClient       *client.Client
	mountType          mount.Type
	mountSource        string
	mountTarget        string
	mountSourceAirbyte string
}

// NewWorker initiates a temporal worker for workflow and activity definition
func NewWorker(r repository.Repository) Worker {
	logger, _ := logger.GetZapLogger()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Error("Unable to create docker client")
	}

	var mountType mount.Type
	var mountSource, mountTarget, mountSourceAirbyte string

	switch {
	case config.Config.Server.Debug:
		mountType = mount.TypeBind
		mountSource = fmt.Sprintf("%s/vdp", os.TempDir())
		mountTarget = fmt.Sprintf("%s/vdp", os.TempDir())
		mountSourceAirbyte = fmt.Sprintf("%s/vdp/airbyte", os.TempDir())
	default:
		mountType = mount.TypeVolume
		mountSource = "vdp"
		mountTarget = "/vdp"
		mountSourceAirbyte = "airbyte"
	}

	return &worker{
		repository:         r,
		dockerClient:       cli,
		mountType:          mountType,
		mountSource:        mountSource,
		mountTarget:        mountTarget,
		mountSourceAirbyte: mountSourceAirbyte,
	}
}

// Stop and remove a container
func stopAndRemoveContainer(cli *client.Client, containerName string) error {

	if err := cli.ContainerStop(context.Background(), containerName, nil); err != nil {
		return fmt.Errorf("unable to stop container %s: %s", containerName, err)
	}

	removeOptions := types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}

	if err := cli.ContainerRemove(context.Background(), containerName, removeOptions); err != nil {
		return fmt.Errorf("Unable to remove container %s: %s", containerName, err)
	}

	return nil
}
