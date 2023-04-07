package service

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/connector-backend/pkg/logger"
	"github.com/instill-ai/connector-backend/pkg/worker"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/types/known/anypb"

	workflowpb "go.temporal.io/api/workflow/v1"
)

func (s *service) startCheckWorkflow(ownerPermalink string, connUID string, dockerRepo string, dockerImgTag string) (string, error) {

	logger, _ := logger.GetZapLogger()

	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("%s.%d.check", connUID, time.Now().UnixNano()),
		TaskQueue: worker.TaskQueue,
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		context.Background(),
		workflowOptions,
		"CheckWorkflow",
		&worker.CheckWorkflowParam{
			OwnerPermalink: ownerPermalink,
			ConnUID:        connUID,
			ImageName:      fmt.Sprintf("%s:%s", dockerRepo, dockerImgTag),
			ContainerName:  workflowOptions.ID,
			ConfigFileName: workflowOptions.ID,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return "", err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return workflowOptions.ID, nil
}

func (s *service) startWriteWorkflow(ownerPermalink string, connUID string,
	dockerRepo string, dockerImgTag string, pipeline string, indices []string, cfgAbCatalog []byte, abMsgs []byte) (string, error) {

	logger, _ := logger.GetZapLogger()

	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("%s.%d.write", connUID, time.Now().UnixNano()),
		TaskQueue: worker.TaskQueue,
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		context.Background(),
		workflowOptions,
		"WriteWorkflow",
		&worker.WriteWorkflowParam{
			OwnerPermalink:           ownerPermalink,
			ConnectorPermalink:       connUID,
			ImageName:                fmt.Sprintf("%s:%s", dockerRepo, dockerImgTag),
			ContainerName:            workflowOptions.ID,
			ConfigFileName:           workflowOptions.ID,
			CatalogFileName:          workflowOptions.ID,
			Pipeline:                 pipeline,
			Indices:                  indices,
			ConfiguredAirbyteCatalog: cfgAbCatalog,
			AirbyteMessages:          abMsgs,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return "", err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return workflowOptions.ID, nil
}

func getOperationFromWorkflowInfo(workflowExecutionInfo *workflowpb.WorkflowExecutionInfo) (*longrunningpb.Operation, *worker.WorkflowParam, *string, error) {
	operation := longrunningpb.Operation{}

	// var state connectorPB.Connector_State
	// if err := converter.GetDefaultDataConverter().FromPayload(
	// 	workflowExecutionInfo.Memo.Fields["result"],
	// 	state,
	// ); err != nil {
	// 	return nil, nil, err
	// }

	switch workflowExecutionInfo.Status {
	case enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		operation = longrunningpb.Operation{
			Done: true,
			Result: &longrunningpb.Operation_Response{
				Response: &anypb.Any{
					Value: workflowExecutionInfo.Memo.GetFields()["Result"].GetData(),
				},
			},
		}
	case enums.WORKFLOW_EXECUTION_STATUS_RUNNING:
	case enums.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW:
		operation = longrunningpb.Operation{
			Done: false,
			Result: &longrunningpb.Operation_Response{
				Response: &anypb.Any{},
			},
		}
	default:
		operation = longrunningpb.Operation{
			Done: true,
			Result: &longrunningpb.Operation_Error{
				Error: &status.Status{
					Code:    int32(workflowExecutionInfo.Status),
					Details: []*anypb.Any{},
					Message: "",
				},
			},
		}
	}

	// Get search attributes that were provided when workflow was started.
	workflowParam := worker.WorkflowParam{}
	operationType := ""
	for k, v := range workflowExecutionInfo.GetSearchAttributes().GetIndexedFields() {
		if k != "ConnectorUID" && k != "Owner" && k != "Type" {
			continue
		}
		var currentVal string
		if err := converter.GetDefaultDataConverter().FromPayload(v, &currentVal); err != nil {
			return nil, nil, nil, err
		}
		if currentVal == "" {
			continue
		}

		if k == "Type" {
			operationType = currentVal
			continue
		}

		if k == "ConnectorUID" {
			workflowParam.ConnectorUID = currentVal
		} else if k == "Owner" {
			workflowParam.Owner = fmt.Sprintf("users/%s", currentVal) // remove prefix users when storing in temporal
		}
	}
	operation.Name = fmt.Sprintf("operations/%s", workflowExecutionInfo.Execution.WorkflowId)
	return &operation, &workflowParam, &operationType, nil
}

func (s *service) GetOperation(workflowId string) (*longrunningpb.Operation, *worker.WorkflowParam, *string, error) {
	workflowExecutionRes, err := s.temporalClient.DescribeWorkflowExecution(context.Background(), workflowId, "")

	if err != nil {
		return nil, nil, nil, err
	}

	return getOperationFromWorkflowInfo(workflowExecutionRes.WorkflowExecutionInfo)
}

func (s *service) SearchAttributeReady() error {
	logger, _ := logger.GetZapLogger()
	id, _ := uuid.NewV4()
	workflowOptions := client.StartWorkflowOptions{
		ID:        id.String(),
		TaskQueue: worker.TaskQueue,
	}

	ctx := context.Background()
	we, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"AddSearchAttributeWorkflow",
	)
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	start := time.Now()
	for {
		if time.Since(start) > 10*time.Second {
			return fmt.Errorf("health workflow timed out")
		}
		workflowExecutionRes, err := s.temporalClient.DescribeWorkflowExecution(ctx, we.GetID(), we.GetRunID())
		if err != nil {
			continue
		}
		if workflowExecutionRes.WorkflowExecutionInfo.Status == enums.WORKFLOW_EXECUTION_STATUS_COMPLETED {
			return nil
		} else if workflowExecutionRes.WorkflowExecutionInfo.Status == enums.WORKFLOW_EXECUTION_STATUS_FAILED {
			return fmt.Errorf("health workflow failed")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
