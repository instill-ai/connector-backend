package worker

import (
	"context"
	"time"

	"github.com/allegro/bigcache"
	"github.com/docker/docker/client"
	"github.com/instill-ai/connector-backend/config"
	"github.com/instill-ai/connector-backend/pkg/logger"
	"github.com/instill-ai/connector-backend/pkg/repository"
	"go.temporal.io/sdk/workflow"

	controllerPB "github.com/instill-ai/protogen-go/vdp/controller/v1alpha"
)

// TaskQueue is the Temporal task queue name for connector-backend
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
	cache              *bigcache.BigCache
	dockerClient       *client.Client
	controllerClient   controllerPB.ControllerPrivateServiceClient
	mountSourceVDP     string
	mountTargetVDP     string
	mountSourceAirbyte string
	mountTargetAirbyte string
}

type WorkflowParam struct {
	ConnectorUID string
	Owner        string
}

// NewWorker initiates a temporal worker for workflow and activity definition
func NewWorker(r repository.Repository, dc *client.Client, c controllerPB.ControllerPrivateServiceClient) Worker {

	logger, _ := logger.GetZapLogger()

	cache, err := bigcache.NewBigCache(bigcache.DefaultConfig(60 * time.Minute))
	if err != nil {
		logger.Error(err.Error())
	}

	return &worker{
		repository:         r,
		cache:              cache,
		dockerClient:       dc,
		controllerClient:   c,
		mountSourceVDP:     config.Config.Worker.MountSource.VDP,
		mountTargetVDP:     config.Config.Worker.MountTarget.VDP,
		mountSourceAirbyte: config.Config.Worker.MountSource.Airbyte,
		mountTargetAirbyte: config.Config.Worker.MountTarget.Airbyte,
	}
}
