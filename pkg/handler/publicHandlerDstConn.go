package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"go.opentelemetry.io/otel/trace"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/instill-ai/connector-backend/internal/resource"
	"github.com/instill-ai/connector-backend/pkg/logger"
	"github.com/instill-ai/x/checkfield"
	"github.com/instill-ai/x/sterr"

	connectorAirbyte "github.com/instill-ai/connector-destination/pkg/airbyte"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

func (h *PublicHandler) CreateDestinationConnector(ctx context.Context, req *connectorPB.CreateDestinationConnectorRequest) (*connectorPB.CreateDestinationConnectorResponse, error) {
	resp, err := h.createConnector(ctx, req)
	if err != nil {
		return resp.(*connectorPB.CreateDestinationConnectorResponse), err
	}

	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		return resp.(*connectorPB.CreateDestinationConnectorResponse), err
	}

	return resp.(*connectorPB.CreateDestinationConnectorResponse), nil
}

func (h *PublicHandler) ListDestinationConnectors(ctx context.Context, req *connectorPB.ListDestinationConnectorsRequest) (*connectorPB.ListDestinationConnectorsResponse, error) {
	resp, err := h.listConnectors(ctx, req)
	return resp.(*connectorPB.ListDestinationConnectorsResponse), err
}

func (h *PublicHandler) GetDestinationConnector(ctx context.Context, req *connectorPB.GetDestinationConnectorRequest) (*connectorPB.GetDestinationConnectorResponse, error) {
	resp, err := h.getConnector(ctx, req)
	return resp.(*connectorPB.GetDestinationConnectorResponse), err
}

func (h *PublicHandler) UpdateDestinationConnector(ctx context.Context, req *connectorPB.UpdateDestinationConnectorRequest) (*connectorPB.UpdateDestinationConnectorResponse, error) {
	resp, err := h.updateConnector(ctx, req)
	return resp.(*connectorPB.UpdateDestinationConnectorResponse), err
}

func (h *PublicHandler) DeleteDestinationConnector(ctx context.Context, req *connectorPB.DeleteDestinationConnectorRequest) (*connectorPB.DeleteDestinationConnectorResponse, error) {
	resp, err := h.deleteConnector(ctx, req)
	if err != nil {
		return resp.(*connectorPB.DeleteDestinationConnectorResponse), err
	}

	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		return resp.(*connectorPB.DeleteDestinationConnectorResponse), err
	}
	return resp.(*connectorPB.DeleteDestinationConnectorResponse), nil
}

func (h *PublicHandler) LookUpDestinationConnector(ctx context.Context, req *connectorPB.LookUpDestinationConnectorRequest) (*connectorPB.LookUpDestinationConnectorResponse, error) {
	resp, err := h.lookUpConnector(ctx, req)
	return resp.(*connectorPB.LookUpDestinationConnectorResponse), err
}

func (h *PublicHandler) ConnectDestinationConnector(ctx context.Context, req *connectorPB.ConnectDestinationConnectorRequest) (*connectorPB.ConnectDestinationConnectorResponse, error) {
	resp, err := h.connectConnector(ctx, req)
	return resp.(*connectorPB.ConnectDestinationConnectorResponse), err
}

func (h *PublicHandler) DisconnectDestinationConnector(ctx context.Context, req *connectorPB.DisconnectDestinationConnectorRequest) (*connectorPB.DisconnectDestinationConnectorResponse, error) {
	resp, err := h.disconnectConnector(ctx, req)
	return resp.(*connectorPB.DisconnectDestinationConnectorResponse), err
}

func (h *PublicHandler) RenameDestinationConnector(ctx context.Context, req *connectorPB.RenameDestinationConnectorRequest) (*connectorPB.RenameDestinationConnectorResponse, error) {
	resp, err := h.renameConnector(ctx, req)
	return resp.(*connectorPB.RenameDestinationConnectorResponse), err
}

func (h *PublicHandler) WriteDestinationConnector(ctx context.Context, req *connectorPB.WriteDestinationConnectorRequest) (*connectorPB.WriteDestinationConnectorResponse, error) {

	ctx, span := tracer.Start(ctx, "WriteDestinationConnector",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

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

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient())
	if err != nil {
		return resp, err
	}

	// Validate batch outputs of each model
	for _, modelOutput := range req.ModelOutputs {
		taskOutputs := modelOutput.TaskOutputs
		if len(req.DataMappingIndices) != len(taskOutputs) {
			return resp, fmt.Errorf("indices list size %d and data list size %d are not equal", len(req.DataMappingIndices), len(taskOutputs))
		}

		// Validate TaskAirbyteCatalog's JSON schema
		if err := connectorAirbyte.ValidateAirbyteCatalog(taskOutputs); err != nil {
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

	if err := h.service.WriteDestinationConnector(ctx, dstConnID, owner,
		connectorAirbyte.WriteDestinationConnectorParam{
			SyncMode:           syncMode,
			DstSyncMode:        dstSyncMode,
			Pipeline:           req.Pipeline,
			Recipe:             req.Recipe,
			DataMappingIndices: req.DataMappingIndices,
			ModelOutputs:       req.ModelOutputs,
		}); err != nil {
		return resp, err
	}

	return resp, nil
}

func (h *PublicHandler) WatchDestinationConnector(ctx context.Context, req *connectorPB.WatchDestinationConnectorRequest) (*connectorPB.WatchDestinationConnectorResponse, error) {
	resp, err := h.watchConnector(ctx, req)
	return resp.(*connectorPB.WatchDestinationConnectorResponse), err
}

func (h *PublicHandler) TestDestinationConnector(ctx context.Context, req *connectorPB.TestDestinationConnectorRequest) (*connectorPB.TestDestinationConnectorResponse, error) {
	resp, err := h.testConnector(ctx, req)
	return resp.(*connectorPB.TestDestinationConnectorResponse), err
}
