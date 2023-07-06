package utils

import "strings"

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
