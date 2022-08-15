package handler

// *RequiredFields are Protobuf message fields with REQUIRED field_behavior annotation
var createRequiredFields = []string{"connector", "connector.configuration"}
var lookUpRequiredFields = []string{"permalink"}
var connectSourceRequiredFields = []string{"name"}
var disconnectSourceRequiredFields = []string{"name"}
var connectDestinationRequiredFields = []string{"name"}
var disconnectDestinationRequiredFields = []string{"name"}
var renameSourceRequiredFields = []string{"name", "new_source_connector_id"}
var renameDestinationRequiredFields = []string{"name", "new_destination_connector_id"}
var writeDestinationRequiredFields = []string{"name", "sync_mode", "destination_sync_mode", "pipeline", "recipe", "indices", "model_instance_outputs"}

// *ImmutableFields* are Protobuf message fields with IMMUTABLE field_behavior annotation
var destinationImmutableFields = []string{"id", "destination_connector_definition"}
var sourceImmutableFields = []string{"id", "source_connector_definition"}

// *OutputOnlyFields are Protobuf message fields with OUTPUT_ONLY field_behavior annotation
var outputOnlyFields = []string{"name", "uid", "connector.state", "connector.tombstone", "connector.owner", "connector.create_time", "connector.update_time"}
