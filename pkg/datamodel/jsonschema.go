package datamodel

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/instill-ai/connector-backend/internal/logger"
)

// SrcConnDefJSONSchema represents the ConnectorDefinition JSON Schema for validating the payload
var SrcConnDefJSONSchema *jsonschema.Schema

// DstConnDefJSONSchema represents the ConnectorDefinition JSON Schema for validating the payload
var DstConnDefJSONSchema *jsonschema.Schema

// SrcConnJSONSchema represents the Connector JSON Schema for validating the payload
var SrcConnJSONSchema *jsonschema.Schema

// DstConnJSONSchema represents the Connector JSON Schema for validating the payload
var DstConnJSONSchema *jsonschema.Schema

// InitJSONSchema initialise JSON Schema instances with the given files
func InitJSONSchema() {

	logger, _ := logger.GetZapLogger()

	compiler := jsonschema.NewCompiler()

	if r, err := os.Open("config/model/connector_definition.json"); err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	} else {
		if err := compiler.AddResource("https://github.com/instill-ai/connector-backend/blob/main/config/model/connector_definition.json", r); err != nil {
			logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
		}
	}

	if r, err := os.Open("config/model/connector.json"); err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	} else {
		if err := compiler.AddResource("https://github.com/instill-ai/connector-backend/blob/main/config/model/connector.json", r); err != nil {
			logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
		}
	}

	var err error
	SrcConnDefJSONSchema, err = compiler.Compile("config/model/source_connector_definition.json")
	if err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	}

	DstConnDefJSONSchema, err = compiler.Compile("config/model/destination_connector_definition.json")
	if err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	}

	SrcConnJSONSchema, err = compiler.Compile("config/model/source_connector.json")
	if err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	}

	DstConnJSONSchema, err = compiler.Compile("config/model/destination_connector.json")
	if err != nil {
		logger.Fatal(fmt.Sprintf("%#v\n", err.Error()))
	}

}

//ValidateJSONSchema validates the Protobuf message data
func ValidateJSONSchema(schema *jsonschema.Schema, msg interface{}, emitUnpopulated bool) error {

	data, err := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: emitUnpopulated}.Marshal(msg.(proto.Message))
	if err != nil {
		return err
	}

	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	if err := schema.Validate(v); err != nil {
		b, _ := json.MarshalIndent(err.(*jsonschema.ValidationError).DetailedOutput(), "", "  ")
		return fmt.Errorf(string(b))
	}

	return nil
}

// ValidateJSONSchemaString validates the string data given a string schema
func ValidateJSONSchemaString(schema string, data string) error {

	sch, err := jsonschema.CompileString("schema.json", schema)
	if err != nil {
		return err
	}

	var v interface{}
	if err := json.Unmarshal([]byte(data), &v); err != nil {
		return err
	}

	if err = sch.Validate(v); err != nil {
		return err
	}

	return nil
}
