package datamodel

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/instill-ai/connector-backend/internal/logger"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
)

// AirbyteMessage defines the AirbyteMessage protocol  as in
// https://github.com/airbytehq/airbyte/blob/master/airbyte-protocol/protocol-models/src/main/resources/airbyte_protocol/airbyte_protocol.yaml#L13-L49
type AirbyteMessage struct {
	Type   string                `json:"type"`
	Record *AirbyteRecordMessage `json:"record"`
}

// AirbyteRecordMessage defines the RECORD type of AirbyteMessage, AirbyteRecordMessage, protocol as in
// https://github.com/airbytehq/airbyte/blob/master/airbyte-protocol/protocol-models/src/main/resources/airbyte_protocol/airbyte_protocol.yaml#L50-L70
type AirbyteRecordMessage struct {
	Stream    string          `json:"stream"`
	Data      json.RawMessage `json:"data"`
	EmittedAt int64           `json:"emitted_at"`
	Namespace string          `json:"namespace"`
}

// AirbyteCatalog defines the AirbyteCatalog protocol as in:
// https://github.com/airbytehq/airbyte/blob/master/airbyte-protocol/protocol-models/src/main/resources/airbyte_protocol/airbyte_protocol.yaml#L212-L222
type AirbyteCatalog struct {
	Streams []AirbyteStream `json:"streams"`
}

// AirbyteStream defines the AirbyteStream protocol as in:
// https://github.com/airbytehq/airbyte/blob/master/airbyte-protocol/protocol-models/src/main/resources/airbyte_protocol/airbyte_protocol.yaml#L223-L260
type AirbyteStream struct {
	Name                    string          `json:"name"`
	JSONSchema              json.RawMessage `json:"json_schema"`
	SupportedSyncModes      []string        `json:"supported_sync_modes"`
	SourceDefinedCursor     bool            `json:"source_defined_cursor"`
	DefaultCursorField      []string        `json:"default_cursor_field"`
	SourceDefinedPrimaryKey [][]string      `json:"source_defined_primary_key"`
	Namespace               string          `json:"namespace"`
}

// ConfiguredAirbyteCatalog defines the ConfiguredAirbyteCatalog protocol  as in:
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

// TaskAirbyteCatalog stores the pre-defined task AirbyteCatalog
var TaskAirbyteCatalog map[string]*AirbyteCatalog

// InitTaskAirbyteCatalog reads all task AirbyteCatalog files and stores the JSON content in the global TaskAirbyteCatalog variable
func InitTaskAirbyteCatalog() {

	logger, _ := logger.GetZapLogger()

	TaskAirbyteCatalog = make(map[string]*AirbyteCatalog)

	catalogFiles, err := find("config/model/airbytecatalog", ".json")
	if err != nil {
		logger.Fatal(err.Error())
	}

	for _, f := range catalogFiles {
		taskName := strings.TrimSuffix(filepath.Base(f), filepath.Ext(f))
		TaskAirbyteCatalog[taskName] = &AirbyteCatalog{}
		if _, ok := modelPB.ModelInstance_Task_value[taskName]; ok {
			b, err := ioutil.ReadFile(f)
			if err != nil {
				logger.Fatal(err.Error())
			}
			if err := json.Unmarshal(b, TaskAirbyteCatalog[taskName]); err != nil {
				logger.Fatal(err.Error())
			}
		} else {
			logger.Fatal(fmt.Sprintf("%s is not a task type defined in the model protobuf", taskName))
		}
	}
}

// ValidateTaskAirbyteCatalog validates the TaskAirbyteCatalog's JSON schema given the task type and the batch data (i.e., the output from model-backend trigger)
func ValidateTaskAirbyteCatalog(task modelPB.ModelInstance_Task, batch *structpb.Value) error {

	// Load TaskAirbyteCatalog JSON Schema
	jsBytes, err := json.Marshal(TaskAirbyteCatalog[task.String()].Streams[0].JSONSchema)
	if err != nil {
		return err
	}

	sch, err := jsonschema.CompileString("schema.json", string(jsBytes))
	if err != nil {
		return err
	}

	// Check each element in the batch
	for idx, value := range batch.GetListValue().GetValues() {
		b, err := protojson.Marshal(value)
		if err != nil {
			return fmt.Errorf("batch[%d] error: %w", idx, err)
		}

		var v interface{}
		if err := json.Unmarshal(b, &v); err != nil {
			return fmt.Errorf("batch[%d] error: %w", idx, err)
		}

		if err = sch.Validate(v); err != nil {
			switch e := err.(type) {
			case *jsonschema.ValidationError:
				b, err := json.MarshalIndent(e.DetailedOutput(), "", "  ")
				if err != nil {
					return fmt.Errorf("batch[%d] error: %w", idx, err)
				}
				return fmt.Errorf("batch[%d] error: %s", idx, string(b))
			case jsonschema.InvalidJSONTypeError:
				return fmt.Errorf("batch[%d] error: %w", idx, e)
			default:
				return fmt.Errorf("batch[%d] error: %w", idx, e)
			}
		}
	}
	return nil
}

func find(root, ext string) ([]string, error) {
	var a []string
	if err := filepath.WalkDir(root, func(s string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if filepath.Ext(d.Name()) == ext {
			a = append(a, s)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return a, nil
}
