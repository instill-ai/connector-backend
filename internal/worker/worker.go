package worker

import (
	"context"

	"github.com/docker/docker/client"
	"go.temporal.io/sdk/workflow"

	"github.com/instill-ai/connector-backend/internal/logger"
	"github.com/instill-ai/connector-backend/pkg/repository"
)

// TaskQueue is the task queue name for connector-backend
const TaskQueue = "connector-backend"

// Worker interface
type Worker interface {
	ConnectorCheckStateWorkflow(ctx workflow.Context, param *CheckStateWorkflowParam) error
	ConnectorCheckStateActivity(ctx context.Context, param *CheckStateActivityParam) (exitCode, error)
}

// worker represents resources required to run Temporal workflow and activity
type worker struct {
	repository   repository.Repository
	dockerClient *client.Client
}

// NewWorker initiates a temporal worker for workflow and activity definition
func NewWorker(r repository.Repository) Worker {
	logger, _ := logger.GetZapLogger()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Error("Unable to create docker client")
	}

	return &worker{
		repository:   r,
		dockerClient: cli,
	}
}
