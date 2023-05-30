package util

import (
	"encoding/json"
	"fmt"

	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/connector-backend/pkg/logger/otel"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
	"go.opentelemetry.io/otel/trace"
)

func ConvertConnectorToResourceName(
	connectorName string,
	connectorType datamodel.ConnectorType) string {
	var connectorTypeStr string
	switch connectorType {
	case datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE):
		connectorTypeStr = "source-connectors"
	case datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION):
		connectorTypeStr = "destination-connectors"
	}

	resourceName := fmt.Sprintf("resources/%s/types/%s", connectorName, connectorTypeStr)

	return resourceName
}

func ConstructAuditLog(
	span trace.Span,
	user mgmtPB.User,
	connector datamodel.Connector,
	eventName string,
	billable bool,
	metadata string,
) []byte {
	logMessage, _ := json.Marshal(otel.AuditLogMessage{
		ServiceName: "connector-backend",
		TraceInfo: struct {
			TraceId string
			SpanId  string
		}{
			TraceId: span.SpanContext().TraceID().String(),
			SpanId:  span.SpanContext().SpanID().String(),
		},
		UserInfo: struct {
			UserID   string
			UserUUID string
			Token    string
		}{
			UserID:   user.Id,
			UserUUID: *user.Uid,
			Token:    *user.CookieToken,
		},
		EventInfo: struct{ Name string }{
			Name: eventName,
		},
		ResourceInfo: struct {
			ResourceName  string
			ResourceUUID  string
			ResourceState string
			Billable      bool
		}{
			ResourceName:  connector.ID,
			ResourceUUID:  connector.UID.String(),
			ResourceState: connectorPB.Connector_State(connector.State).String(),
			Billable:      billable,
		},
		Metadata: metadata,
	})

	return logMessage
}

func ConstructErrorLog(
	span trace.Span,
	statusCode int,
	errorMessage string,
) []byte {
	logMessage, _ := json.Marshal(otel.ErrorLogMessage{
		ServiceName: "connector-backend",
		TraceInfo: struct {
			TraceId string
			SpanId  string
		}{
			TraceId: span.SpanContext().TraceID().String(),
			SpanId:  span.SpanContext().SpanID().String(),
		},
		StatusCode:   statusCode,
		ErrorMessage: errorMessage,
	})

	return logMessage
}
