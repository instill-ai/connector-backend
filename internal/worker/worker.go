package worker

import (
	"context"

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
	exitCodeOK exitCode = iota
	exitCodeError
)

// Worker interface
type Worker interface {
	CheckWorkflow(ctx workflow.Context, param *CheckWorkflowParam) error
	CheckActivity(ctx context.Context, param *CheckActivityParam) (exitCode, error)
	WriteWorkflow(ctx workflow.Context, param *WriteWorkflowParam) error
	WriteActivity(ctx context.Context, param *WriteActivityParam) (exitCode, error)
}

// worker represents resources required to run Temporal workflow and activity
type worker struct {
	repository         repository.Repository
	dockerClient       *client.Client
	mountSourceVDP     string
	mountTargetVDP     string
	mountSourceAirbyte string
	mountTargetAirbyte string
}

// NewWorker initiates a temporal worker for workflow and activity definition
func NewWorker(r repository.Repository) Worker {
	logger, _ := logger.GetZapLogger()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Error("Unable to create docker client")
	}

	return &worker{
		repository:         r,
		dockerClient:       cli,
		mountSourceVDP:     config.Config.Worker.MountSource.VDP,
		mountTargetVDP:     "/vdp",
		mountSourceAirbyte: config.Config.Worker.MountSource.Airbyte,
		mountTargetAirbyte: "/local",
	}
}
