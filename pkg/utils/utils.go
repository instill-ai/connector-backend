package utils

import (
	"fmt"
	"strings"
	"time"

	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/instill-ai/connector-backend/config"
	"github.com/instill-ai/connector-backend/internal/resource"
	"google.golang.org/protobuf/types/known/structpb"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"

	componentBase "github.com/instill-ai/component/pkg/base"
	connector "github.com/instill-ai/connector/pkg"
	connectorAirbyte "github.com/instill-ai/connector/pkg/airbyte"
	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1alpha"
)

const (
	CreateEvent          string = "Create"
	UpdateEvent          string = "Update"
	DeleteEvent          string = "Delete"
	ConnectEvent         string = "Connect"
	DisconnectEvent      string = "Disconnect"
	RenameEvent          string = "Rename"
	ExecuteEvent         string = "Execute"
	credentialMaskString string = "*****MASK*****"
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

type UsageMetricData struct {
	OwnerUID               string
	Status                 mgmtPB.Status
	ConnectorID            string
	ConnectorUID           string
	ConnectorExecuteUID    string
	ConnectorDefinitionUid string
	ExecuteTime            string
	ComputeTimeDuration    float64
}

func NewDataPoint(data UsageMetricData, pipelineMetadata *structpb.Value) *write.Point {
	pipelineOwnerUUID, _ := resource.GetRscPermalinkUID(pipelineMetadata.GetStructValue().GetFields()["owner"].GetStringValue())
	return influxdb2.NewPoint(
		"connector.execute",
		map[string]string{
			"status": data.Status.String(),
		},
		map[string]interface{}{
			"pipeline_id":              pipelineMetadata.GetStructValue().GetFields()["id"].GetStringValue(),
			"pipeline_uid":             pipelineMetadata.GetStructValue().GetFields()["uid"].GetStringValue(),
			"pipeline_release_id":      pipelineMetadata.GetStructValue().GetFields()["release_id"].GetStringValue(),
			"pipeline_release_uid":     pipelineMetadata.GetStructValue().GetFields()["release_uid"].GetStringValue(),
			"pipeline_owner":           pipelineOwnerUUID,
			"pipeline_trigger_id":      pipelineMetadata.GetStructValue().GetFields()["trigger_id"].GetStringValue(),
			"connector_owner_uid":      data.OwnerUID,
			"connector_id":             data.ConnectorID,
			"connector_uid":            data.ConnectorUID,
			"connector_definition_uid": data.ConnectorDefinitionUid,
			"connector_execute_id":     data.ConnectorExecuteUID,
			"execute_time":             data.ExecuteTime,
			"compute_time_duration":    data.ComputeTimeDuration,
		},
		time.Now(),
	)
}

func MaskCredentialFields(connector componentBase.IConnector, defId string, config *structpb.Struct) {
	maskCredentialFields(connector, defId, config, "")
}

func maskCredentialFields(connector componentBase.IConnector, defId string, config *structpb.Struct, prefix string) {

	for k, v := range config.GetFields() {
		key := prefix + k
		if connector.IsCredentialField(defId, key) {
			config.GetFields()[k] = structpb.NewStringValue(credentialMaskString)
		}
		if v.GetStructValue() != nil {
			maskCredentialFields(connector, defId, v.GetStructValue(), fmt.Sprintf("%s.", key))
		}

	}
}

func RemoveCredentialFieldsWithMaskString(connector componentBase.IConnector, defId string, config *structpb.Struct) {
	removeCredentialFieldsWithMaskString(connector, defId, config, "")
}

func removeCredentialFieldsWithMaskString(connector componentBase.IConnector, defId string, config *structpb.Struct, prefix string) {

	for k, v := range config.GetFields() {
		key := prefix + k
		if connector.IsCredentialField(defId, key) {
			if v.GetStringValue() == credentialMaskString {
				delete(config.GetFields(), k)
			}
		}
		if v.GetStructValue() != nil {
			removeCredentialFieldsWithMaskString(connector, defId, v.GetStructValue(), fmt.Sprintf("%s.", key))
		}

	}
}

func GetConnectorOptions() connector.ConnectorOptions {
	return connector.ConnectorOptions{
		Airbyte: connectorAirbyte.ConnectorOptions{
			MountSourceVDP:        config.Config.Connector.Airbyte.MountSource.VDP,
			MountTargetVDP:        config.Config.Connector.Airbyte.MountTarget.VDP,
			MountSourceAirbyte:    config.Config.Connector.Airbyte.MountSource.Airbyte,
			MountTargetAirbyte:    config.Config.Connector.Airbyte.MountTarget.Airbyte,
			ExcludeLocalConnector: config.Config.Connector.Airbyte.ExcludeLocalConnector,
			VDPProtocolPath:       "/etc/vdp/vdp_protocol.yaml",
		},
	}

}
