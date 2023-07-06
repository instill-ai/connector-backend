package utils

type OperationType string

const (
	OperationTypeCheck       OperationType = "check"
	OperationTypeWrite       OperationType = "write"
	OperationTypeHealthCheck OperationType = "healthcheck"
)
