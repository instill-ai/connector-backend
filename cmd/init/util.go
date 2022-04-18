package main

import (
	"encoding/json"
	"io/ioutil"

	"github.com/ghodss/yaml"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/datatypes"

	"github.com/instill-ai/connector-backend/internal/util"

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
			case *[]*connectorPB.SourceDefinition:
				srcDef := connectorPB.SourceDefinition{}
				if err := pj.Unmarshal(b, &srcDef); err != nil {
					return err
				}
				*pb = append(*pb, &srcDef)
			case *[]*connectorPB.DestinationDefinition:
				dstDef := connectorPB.DestinationDefinition{}
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

	util.ConvertAllJSONKeySnakeCase(jsonSliceMap)
	util.ConvertAllJSONEnumValueToProtoStyle(enumRegistry, jsonSliceMap)

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
