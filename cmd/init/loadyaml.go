package main

import (
	"fmt"

	connectorPB "github.com/instill-ai/protogen-go/connector/v1alpha"
)

const (
	seedDir = "configs/init/%s/seed/%s"
)

func loadDefinitionAndDockerImageSpecs(
	srcConnDefs *[]*connectorPB.SourceConnectorDefinition,
	srcDefs *[]*connectorPB.ConnectorDefinition,
	dstConnDefs *[]*connectorPB.DestinationConnectorDefinition,
	dstDefs *[]*connectorPB.ConnectorDefinition,
	dockerImageSpecs *[]*connectorPB.DockerImageSpec) error {

	sourceDefsFiles := []string{
		fmt.Sprintf(seedDir, "instill", "source_definitions.yaml"),
	}

	destinationDefsFiles := []string{
		fmt.Sprintf(seedDir, "instill", "destination_definitions.yaml"),
		fmt.Sprintf(seedDir, "airbyte", "destination_definitions.yaml"),
	}

	specsFiles := []string{
		fmt.Sprintf(seedDir, "instill", "source_specs.yaml"),
		fmt.Sprintf(seedDir, "instill", "destination_specs.yaml"),
		fmt.Sprintf(seedDir, "airbyte", "destination_specs.yaml"),
	}

	for _, filename := range sourceDefsFiles {
		if jsonSliceMap, err := processJSONSliceMap(filename); err == nil {
			if err := unmarshalConnectorPB(jsonSliceMap, srcConnDefs); err != nil {
				return err
			}
			if err := unmarshalConnectorPB(jsonSliceMap, srcDefs); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	for _, filename := range destinationDefsFiles {
		if jsonSliceMap, err := processJSONSliceMap(filename); err == nil {
			if err := unmarshalConnectorPB(jsonSliceMap, dstConnDefs); err != nil {
				return err
			}
			if err := unmarshalConnectorPB(jsonSliceMap, dstDefs); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	for _, filename := range specsFiles {
		if jsonSliceMap, err := processJSONSliceMap(filename); err == nil {
			if err := unmarshalConnectorPB(jsonSliceMap, dockerImageSpecs); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}
