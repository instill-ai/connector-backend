package main

import (
	"encoding/json"
	"io/ioutil"
	"reflect"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/iancoleman/strcase"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/datatypes"

	connectorPB "github.com/instill-ai/protogen-go/connector/v1alpha"
)

// unmarshalConnectorPB unmarshals a slice of JSON object into a Protobuf Message Go struct element by element
// See: https://github.com/golang/protobuf/issues/675#issuecomment-411182202
func unmarshalConnectorPB(jsonSliceMap interface{}, pb interface{}) error {

	pj := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}

	switch v := jsonSliceMap.(type) {
	case []map[string]interface{}:
		for _, vv := range v {

			b, err := json.Marshal(vv)
			if err != nil {
				return err
			}

			switch pb := pb.(type) {
			case *[]*connectorPB.ConnectorDefinition:
				def := connectorPB.ConnectorDefinition{}
				if err := pj.Unmarshal(b, &def); err != nil {
					return err
				}
				*pb = append(*pb, &def)
			case *[]*connectorPB.SourceConnectorDefinition:
				srcDef := connectorPB.SourceConnectorDefinition{}
				if err := pj.Unmarshal(b, &srcDef); err != nil {
					return err
				}
				*pb = append(*pb, &srcDef)
			case *[]*connectorPB.DestinationConnectorDefinition:
				dstDef := connectorPB.DestinationConnectorDefinition{}
				if err := pj.Unmarshal(b, &dstDef); err != nil {
					return err
				}
				*pb = append(*pb, &dstDef)
			case *[]*connectorPB.DockerImageSpec:
				dockerImgSpec := connectorPB.DockerImageSpec{}
				if err := pj.Unmarshal(b, &dockerImgSpec); err != nil {
					return err
				}
				*pb = append(*pb, &dockerImgSpec)
			}
		}
	}
	return nil
}

func processJSONSliceMap(filename string) ([]map[string]interface{}, error) {

	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	b, err := yaml.YAMLToJSON(yamlFile)
	if err != nil {
		return nil, err
	}

	var jsonSliceMap []map[string]interface{}
	if err := json.Unmarshal(b, &jsonSliceMap); err != nil {
		return nil, err
	}

	ConvertAllJSONKeySnakeCase(jsonSliceMap)
	ConvertAllJSONEnumValueToProtoStyle(enumRegistry, jsonSliceMap)
	ConvertUUIDKey(jsonSliceMap)
	ConvertNameKey(jsonSliceMap)

	return jsonSliceMap, nil
}

func findDockerImageSpec(dockerRepositoryImageTag string, specs *[]*connectorPB.DockerImageSpec) (datatypes.JSON, error) {

	// Search for the docker image corresponding spec
	for _, v := range *specs {
		if dockerRepositoryImageTag == v.GetDockerImage() {
			spec, err := json.Marshal(v.GetSpec())
			if err != nil {
				return nil, err
			}
			return spec, nil
		}
	}

	// If the docker image index cannot be found, return an empty spec
	return []byte("{}"), nil
}

// ConvertAllJSONKeySnakeCase traverses a JSON object to replace all keys to snake_case.
func ConvertAllJSONKeySnakeCase(i interface{}) {
	switch v := i.(type) {
	case map[string]interface{}:
		for k, vv := range v {
			sc := strcase.ToSnake(k)
			if sc != k {
				v[sc] = v[k]
				delete(v, k)
			}
			ConvertAllJSONKeySnakeCase(vv)
		}
	case []map[string]interface{}:
		for _, vv := range v {
			ConvertAllJSONKeySnakeCase(vv)
		}
	}
}

// ConvertAllJSONEnumValueToProtoStyle converts lowercase enum value to the Protobuf naming convention where the enum type is always prefixed and is UPPERCASE snake_case.
// For examples:
// - api in a Protobuf `Enum SourceType` type will be converted to SOURCE_TYPE_API
// - oauth2.0  in a Protobuf `Enum AuthFlowType` type will be converted to AUTH_FLOW_TYPE_OAUTH2_0
func ConvertAllJSONEnumValueToProtoStyle(enumRegistry map[string]map[string]int32, i interface{}) {
	switch v := i.(type) {
	case map[string]interface{}:
		for k, vv := range v {
			if _, ok := enumRegistry[k]; ok {
				for enumKey := range enumRegistry[k] {
					if reflect.TypeOf(vv).Kind() == reflect.Slice { // repeated enum type
						for kk, vvv := range vv.([]interface{}) {
							if strings.ReplaceAll(vvv.(string), ".", "_") == strings.ToLower(strings.TrimPrefix(enumKey, strings.ToUpper(k)+"_")) {
								vv.([]interface{})[kk] = enumKey
							}
						}
					} else {
						if strings.ReplaceAll(vv.(string), ".", "_") == strings.ToLower(strings.TrimPrefix(enumKey, strings.ToUpper(k)+"_")) {
							v[k] = enumKey
						}
					}
				}
			}
			ConvertAllJSONEnumValueToProtoStyle(enumRegistry, vv)
		}
	case []map[string]interface{}:
		for _, vv := range v {
			ConvertAllJSONEnumValueToProtoStyle(enumRegistry, vv)
		}
	}
}

// ConvertUUIDKey converts field key `source_definition_id` and `destination_definition_id` to `uid`
func ConvertUUIDKey(i interface{}) {
	switch v := i.(type) {
	case map[string]interface{}:
		for k, vv := range v {
			if k == "source_definition_id" || k == "destination_definition_id" {
				v["uid"] = v[k]
				delete(v, k)
			}
			ConvertUUIDKey(vv)
		}
	case []map[string]interface{}:
		for _, vv := range v {
			ConvertUUIDKey(vv)
		}
	}
}

// ConvertNameKey converts field key `name` to `title`
func ConvertNameKey(i interface{}) {
	switch v := i.(type) {
	case map[string]interface{}:
		for k, vv := range v {
			if k == "name" {
				v["title"] = v[k]
				delete(v, k)
			}
			ConvertNameKey(vv)
		}
	case []map[string]interface{}:
		for _, vv := range v {
			ConvertNameKey(vv)
		}
	}
}