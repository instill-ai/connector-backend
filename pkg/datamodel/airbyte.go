package datamodel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/ghodss/yaml"
	"github.com/instill-ai/connector-backend/internal/logger"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"google.golang.org/protobuf/encoding/protojson"

	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

// AirbyteMessage defines the AirbyteMessage protocol  as in
// https://github.com/airbytehq/airbyte/blob/master/airbyte-protocol/protocol-models/src/main/resources/airbyte_protocol/airbyte_protocol.yaml#L13-L49
type AirbyteMessage struct {
	Type   string                `json:"type"`
	Record *AirbyteRecordMessage `json:"record"`
}

// AirbyteRecordMessage defines the RECORD type of AirbyteMessage, AirbyteRecordMessage, protocol as in (without namespace field)
// https://github.com/airbytehq/airbyte/blob/master/airbyte-protocol/protocol-models/src/main/resources/airbyte_protocol/airbyte_protocol.yaml#L50-L70
type AirbyteRecordMessage struct {
	Stream    string          `json:"stream"`
	Data      json.RawMessage `json:"data"`
	EmittedAt int64           `json:"emitted_at"`
}

// AirbyteCatalog defines the AirbyteCatalog protocol as in:
// https://github.com/airbytehq/airbyte/blob/master/airbyte-protocol/protocol-models/src/main/resources/airbyte_protocol/airbyte_protocol.yaml#L212-L222
type AirbyteCatalog struct {
	Streams []AirbyteStream `json:"streams"`
}

// AirbyteStream defines the AirbyteStream protocol as in (without namespace field):
// https://github.com/airbytehq/airbyte/blob/master/airbyte-protocol/protocol-models/src/main/resources/airbyte_protocol/airbyte_protocol.yaml#L223-L260
type AirbyteStream struct {
	Name                    string          `json:"name"`
	JSONSchema              json.RawMessage `json:"json_schema"`
	SupportedSyncModes      []string        `json:"supported_sync_modes"`
	SourceDefinedCursor     bool            `json:"source_defined_cursor"`
	DefaultCursorField      []string        `json:"default_cursor_field"`
	SourceDefinedPrimaryKey [][]string      `json:"source_defined_primary_key"`
}

// ConfiguredAirbyteCatalog defines the ConfiguredAirbyteCatalog protocol as in:
// https://github.com/airbytehq/airbyte/blob/master/airbyte-protocol/protocol-models/src/main/resources/airbyte_protocol/airbyte_protocol.yaml#L261-L271
type ConfiguredAirbyteCatalog struct {
	Streams []ConfiguredAirbyteStream `json:"streams"`
}

// ConfiguredAirbyteStream defines the ConfiguredAirbyteStream protocol  as in:
// https://github.com/airbytehq/airbyte/blob/master/airbyte-protocol/protocol-models/src/main/resources/airbyte_protocol/airbyte_protocol.yaml#L272-L299
type ConfiguredAirbyteStream struct {
	Stream              *AirbyteStream `json:"stream"`
	SyncMode            string         `json:"sync_mode"`
	CursorField         []string       `json:"cursor_field"`
	DestinationSyncMode string         `json:"destination_sync_mode"`
	PrimaryKey          []string       `json:"primary_key"`
}

// WriteDestinationConnectorParam stores the parameters for WriteDestinationConnector service per model instance
type WriteDestinationConnectorParam struct {
	Task               modelPB.ModelInstance_Task
	SyncMode           string
	DstSyncMode        string
	Pipeline           string
	ModelInstance      string
	Recipe             *pipelinePB.Recipe
	DataMappingIndices []string
	BatchOutputs       []*pipelinePB.BatchOutput
}

// TaskOutputAirbyteCatalog stores the pre-defined task AirbyteCatalog
var TaskOutputAirbyteCatalog AirbyteCatalog

var sch *jsonschema.Schema

// InitAirbyteCatalog reads all task AirbyteCatalog files and stores the JSON content in the global TaskAirbyteCatalog variable
func InitAirbyteCatalog() {

	logger, _ := logger.GetZapLogger()

	yamlFile, err := ioutil.ReadFile("/usr/local/vdp/vdp_protocol.yaml")
	if err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	}

	jsonSchemaBytes, err := yaml.YAMLToJSON(yamlFile)
	if err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	}

	compiler := jsonschema.NewCompiler()

	err = compiler.AddResource("vdp_protocol.json", bytes.NewReader(jsonSchemaBytes))
	if err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	}

	sch, err = compiler.Compile("vdp_protocol.json")
	if err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	}

	// Initialise TaskOutputAirbyteCatalog.Streams[0]
	TaskOutputAirbyteCatalog.Streams = []AirbyteStream{
		{
			JSONSchema:          jsonSchemaBytes,
			SupportedSyncModes:  []string{"full_refresh", "incremental"},
			SourceDefinedCursor: false,
		},
	}

}

// ValidateAirbyteCatalog validates the TaskAirbyteCatalog's JSON schema given the task type and the batch data (i.e., the output from model-backend trigger)
func ValidateAirbyteCatalog(batchOutputs []*pipelinePB.BatchOutput) error {

	// Check each element in the batch
	for idx, batchOutput := range batchOutputs {

		b, err := protojson.MarshalOptions{
			UseProtoNames:   true,
			EmitUnpopulated: true,
		}.Marshal(batchOutput)

		if err != nil {
			return fmt.Errorf("batch_outputs[%d] error: %w", idx, err)
		}

		var v interface{}
		if err := json.Unmarshal(b, &v); err != nil {
			return fmt.Errorf("batch_outputs[%d] error: %w", idx, err)
		}

		if err = sch.Validate(v); err != nil {
			switch e := err.(type) {
			case *jsonschema.ValidationError:
				b, err := json.MarshalIndent(e.DetailedOutput(), "", "  ")
				if err != nil {
					return fmt.Errorf("batch_outputs[%d] error: %w", idx, err)
				}
				return fmt.Errorf("batch_outputs[%d] error: %s", idx, string(b))
			case jsonschema.InvalidJSONTypeError:
				return fmt.Errorf("batch_outputs[%d] error: %w", idx, e)
			default:
				return fmt.Errorf("batch_outputs[%d] error: %w", idx, e)
			}
		}
	}
	return nil
}
