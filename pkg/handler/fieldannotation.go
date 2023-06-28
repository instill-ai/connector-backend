package handler

// *RequiredFields are Protobuf message fields with REQUIRED field_behavior annotation
var createRequiredFields = []string{"configuration"}
var lookUpRequiredFields = []string{"permalink"}
var connectRequiredFields = []string{"name"}
var disconnectRequiredFields = []string{"name"}
var renameRequiredFields = []string{"name", "new_connector_id"}

// *ImmutableFields* are Protobuf message fields with IMMUTABLE field_behavior annotation
var immutableFields = []string{"id", "connector_definition"}

// *OutputOnlyFields are Protobuf message fields with OUTPUT_ONLY field_behavior annotation
var outputOnlyFields = []string{"name", "uid", "state", "tombstone", "owner", "create_time", "update_time"}
