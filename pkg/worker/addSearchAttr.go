package worker

import (
	"github.com/gofrs/uuid"
	"go.temporal.io/sdk/workflow"

	"github.com/instill-ai/connector-backend/internal/util"
)

func (w *worker) AddSearchAttributeWorkflow(ctx workflow.Context) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("AddSearchAttributeWorkflow started")

	// Upsert search attributes.
	connectorUID, err := uuid.NewV4()
	if err != nil {
		return err
	}

	attributes := map[string]interface{}{
		"Type":          util.OperationTypeHealthCheck,
		"ConnectorUID":  connectorUID.String(),
		"Owner":         "",
	}

	err = workflow.UpsertSearchAttributes(ctx, attributes)
	if err != nil {
		return err
	}

	logger.Info("AddSearchAttributeWorkflow completed")

	return nil
}
