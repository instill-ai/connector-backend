package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/instill-ai/connector-backend/internal/logger"
	"github.com/instill-ai/connector-backend/internal/resource"
	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/x/checkfield"
	"github.com/instill-ai/x/sterr"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
)

func (h *handler) CreateDestinationConnector(ctx context.Context, req *connectorPB.CreateDestinationConnectorRequest) (*connectorPB.CreateDestinationConnectorResponse, error) {
	resp, err := h.createConnector(ctx, req)
	if err != nil {
		return resp.(*connectorPB.CreateDestinationConnectorResponse), err
	}

	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		return resp.(*connectorPB.CreateDestinationConnectorResponse), err
	}

	return resp.(*connectorPB.CreateDestinationConnectorResponse), nil
}

func (h *handler) ListDestinationConnector(ctx context.Context, req *connectorPB.ListDestinationConnectorRequest) (*connectorPB.ListDestinationConnectorResponse, error) {
	resp, err := h.listConnector(ctx, req)
	return resp.(*connectorPB.ListDestinationConnectorResponse), err
}

func (h *handler) GetDestinationConnector(ctx context.Context, req *connectorPB.GetDestinationConnectorRequest) (*connectorPB.GetDestinationConnectorResponse, error) {
	resp, err := h.getConnector(ctx, req)
	return resp.(*connectorPB.GetDestinationConnectorResponse), err
}

func (h *handler) UpdateDestinationConnector(ctx context.Context, req *connectorPB.UpdateDestinationConnectorRequest) (*connectorPB.UpdateDestinationConnectorResponse, error) {
	resp, err := h.updateConnector(ctx, req)
	return resp.(*connectorPB.UpdateDestinationConnectorResponse), err
}

func (h *handler) DeleteDestinationConnector(ctx context.Context, req *connectorPB.DeleteDestinationConnectorRequest) (*connectorPB.DeleteDestinationConnectorResponse, error) {
	resp, err := h.deleteConnector(ctx, req)
	if err != nil {
		return resp.(*connectorPB.DeleteDestinationConnectorResponse), err
	}

	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		return resp.(*connectorPB.DeleteDestinationConnectorResponse), err
	}
	return resp.(*connectorPB.DeleteDestinationConnectorResponse), nil
}

func (h *handler) LookUpDestinationConnector(ctx context.Context, req *connectorPB.LookUpDestinationConnectorRequest) (*connectorPB.LookUpDestinationConnectorResponse, error) {
	resp, err := h.lookUpConnector(ctx, req)
	return resp.(*connectorPB.LookUpDestinationConnectorResponse), err
}

func (h *handler) ConnectDestinationConnector(ctx context.Context, req *connectorPB.ConnectDestinationConnectorRequest) (*connectorPB.ConnectDestinationConnectorResponse, error) {
	resp, err := h.connectConnector(ctx, req)
	return resp.(*connectorPB.ConnectDestinationConnectorResponse), err
}

func (h *handler) DisconnectDestinationConnector(ctx context.Context, req *connectorPB.DisconnectDestinationConnectorRequest) (*connectorPB.DisconnectDestinationConnectorResponse, error) {
	resp, err := h.disconnectConnector(ctx, req)
	return resp.(*connectorPB.DisconnectDestinationConnectorResponse), err
}

func (h *handler) RenameDestinationConnector(ctx context.Context, req *connectorPB.RenameDestinationConnectorRequest) (*connectorPB.RenameDestinationConnectorResponse, error) {
	resp, err := h.renameConnector(ctx, req)
	return resp.(*connectorPB.RenameDestinationConnectorResponse), err
}

func (h *handler) WriteDestinationConnector(ctx context.Context, req *connectorPB.WriteDestinationConnectorRequest) (*connectorPB.WriteDestinationConnectorResponse, error) {

	logger, _ := logger.GetZapLogger()

	resp := &connectorPB.WriteDestinationConnectorResponse{}

	// Return error if REQUIRED fields are not provided in the requested payload
	if err := checkfield.CheckRequiredFields(req, writeDestinationRequiredFields); err != nil {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] write destination connector error",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Field:       "required fields",
					Description: err.Error(),
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return resp, st.Err()
	}

	dstConnID, err := resource.GetRscNameID(req.GetName())
	if err != nil {
		return resp, err
	}

	ownerRscName, err := resource.GetOwner(ctx)
	if err != nil {
		return resp, err
	}

	var rootFieldName string
	switch req.Task {
	case modelPB.ModelInstance_TASK_UNSPECIFIED:
		rootFieldName = "unspecified_outputs"
	case modelPB.ModelInstance_TASK_CLASSIFICATION:
		rootFieldName = "classification_outputs"
	case modelPB.ModelInstance_TASK_DETECTION:
		rootFieldName = "detection_outputs"
	case modelPB.ModelInstance_TASK_KEYPOINT:
		rootFieldName = "keypoint_outputs"
	}

	batch, ok := req.Data.Fields[rootFieldName]
	if !ok {
		return resp, fmt.Errorf("Task input array is not found in the payload")
	}

	// Validate TaskAirbyteCatalog's JSON schema
	if err := datamodel.ValidateTaskAirbyteCatalog(req.Task, batch); err != nil {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] write destination connector error",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Field:       "data",
					Description: err.Error(),
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return resp, st.Err()
	}

	var syncMode string
	switch req.SyncMode {
	case connectorPB.SupportedSyncModes_SUPPORTED_SYNC_MODES_FULL_REFRESH:
		syncMode = "full_refresh"
	case connectorPB.SupportedSyncModes_SUPPORTED_SYNC_MODES_INCREMENTAL:
		syncMode = "incremental"
	}

	var dstSyncMode string
	switch req.DestinationSyncMode {
	case connectorPB.SupportedDestinationSyncModes_SUPPORTED_DESTINATION_SYNC_MODES_OVERWRITE:
		dstSyncMode = "overwrite"
	case connectorPB.SupportedDestinationSyncModes_SUPPORTED_DESTINATION_SYNC_MODES_APPEND:
		dstSyncMode = "append"
	case connectorPB.SupportedDestinationSyncModes_SUPPORTED_DESTINATION_SYNC_MODES_APPEND_DEDUP:
		dstSyncMode = "append_dedup"
	}

	if err := h.service.WriteDestinationConnector(dstConnID, ownerRscName, req.Task, syncMode, dstSyncMode, batch); err != nil {
		return resp, err
	}

	return resp, nil
}
