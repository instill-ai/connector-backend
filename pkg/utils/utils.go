package utils

import (
	"strings"
	"time"

	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"google.golang.org/protobuf/types/known/structpb"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

const (
	CreateEvent     string = "Create"
	UpdateEvent     string = "Update"
	DeleteEvent     string = "Delete"
	ConnectEvent    string = "Connect"
	DisconnectEvent string = "Disconnect"
	RenameEvent     string = "Rename"
	ExecuteEvent    string = "Execute"
)

func IsAuditEvent(eventName string) bool {
	return strings.HasPrefix(eventName, CreateEvent) ||
		strings.HasPrefix(eventName, UpdateEvent) ||
		strings.HasPrefix(eventName, DeleteEvent) ||
		strings.HasPrefix(eventName, ConnectEvent) ||
		strings.HasPrefix(eventName, DisconnectEvent) ||
		strings.HasPrefix(eventName, RenameEvent) ||
		strings.HasPrefix(eventName, ExecuteEvent)
}

// TODO: billable connectors TBD
func IsBillableEvent(eventName string) bool {
	return false
}

func NewDataPoint(
	ownerUUID string,
	connectorExecuteId string,
	connector *datamodel.Connector,
	pipelineMetadata *structpb.Value,
	startTime time.Time,
) *write.Point {
	return influxdb2.NewPoint(
		"connector.execute",
		map[string]string{},
		map[string]interface{}{
			"pipeline_id":              pipelineMetadata.GetStructValue().GetFields()["id"].GetStringValue(),
			"pipeline_uid":             pipelineMetadata.GetStructValue().GetFields()["uid"].GetStringValue(),
			"pipeline_owner":           pipelineMetadata.GetStructValue().GetFields()["owner"].GetStringValue(),
			"pipeline_trigger_id":      pipelineMetadata.GetStructValue().GetFields()["trigger_id"].GetStringValue(),
			"connector_owner_uid":      ownerUUID,
			"connector_id":             connector.ID,
			"connector_uid":            connector.UID.String(),
			"connector_definition_uid": connector.ConnectorDefinitionUID.String(),
			"connector_execute_id":     connectorExecuteId,
			"execute_time":             startTime.Format(time.RFC3339Nano),
		},
		time.Now(),
	)
}
