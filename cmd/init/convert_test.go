package main

import (
	"bytes"
	"encoding/json"
	"log"
	"testing"
)

func TestConvertJSONKeySnakeCase(t *testing.T) {
	tests := []struct {
		inputJSON    string
		expectedJSON string
	}{
		{
			inputJSON: `
			[
				{
					"dockerImage": "airbyte/destination-snowflake:0.4.24",
					"spec": {
						"advanced_auth": {
							"auth_flow_type": "oauth2.0",
							"predicate_key": [
								"credentials",
								"auth_type"
							],
							"predicate_value": "OAuth2.0"
						},
						"connectionSpecification": {
							"$schema": "http://json-schema.org/draft-07/schema#",
							"additionalProperties": true,
							"title": "Snowflake Destination Spec"
						},
						"documentationUrl": "https://docs.airbyte.io/integrations/destinations/snowflake",
						"supported_destination_sync_modes": [
							"overwrite",
							"append",
							"append_dedup"
						],
						"supportsDBT": true
					}
				}
			]`,
			expectedJSON: `
			[
				{
					"docker_image": "airbyte/destination-snowflake:0.4.24",
					"spec": {
						"advanced_auth": {
							"auth_flow_type": "oauth2.0",
							"predicate_key": [
								"credentials",
								"auth_type"
							],
							"predicate_value": "OAuth2.0"
						},
						"connection_specification": {
							"$schema": "http://json-schema.org/draft-07/schema#",
							"additionalProperties": true,
							"title": "Snowflake Destination Spec"
						},
						"documentation_url": "https://docs.airbyte.io/integrations/destinations/snowflake",
						"supported_destination_sync_modes": [
							"overwrite",
							"append",
							"append_dedup"
						],
						"supports_dbt": true
					}
				}
			]`,
		},
	}
	for _, test := range tests {

		var jsonSliceMap []map[string]interface{}
		if err := json.Unmarshal([]byte(test.inputJSON), &jsonSliceMap); err != nil {
			log.Fatalf("Unmarshal: %v", err)
		}

		ConvertAllJSONKeySnakeCase(jsonSliceMap)

		if b, err := json.Marshal(jsonSliceMap); err != nil {
			log.Fatalf("Marshal: %v", err)
		} else {
			buf := &bytes.Buffer{}
			if err := json.Compact(buf, []byte(test.expectedJSON)); err != nil {
				log.Fatalf("Compact: %v", err)
			}
			if string(b) != buf.String() {
				t.Errorf("\nGot: %v\n Want: %v", string(b), buf.String())
			}
		}
	}
}

func TestConvertJSONEnumValue(t *testing.T) {

	tests := []struct {
		inputJSON    string
		expectedJSON string
	}{
		{
			inputJSON: `
			[
				{
					"dockerImage": "airbyte/destination-snowflake:0.4.24",
					"spec": {
						"advanced_auth": {
							"auth_flow_type": "oauth2.0",
							"predicate_key": [
								"credentials",
								"auth_type"
							],
							"predicate_value": "OAuth2.0"
						},
						"connectionSpecification": {
							"$schema": "http://json-schema.org/draft-07/schema#",
							"additionalProperties": true,
							"title": "Snowflake Destination Spec"
						},
						"documentationUrl": "https://docs.airbyte.io/integrations/destinations/snowflake",
						"supported_destination_sync_modes": [
							"overwrite",
							"append",
							"append_dedup"
						],
						"supportsDBT": true
					}
				}
			]`,
			expectedJSON: `
			[
				{
					"dockerImage": "airbyte/destination-snowflake:0.4.24",
					"spec": {
						"advanced_auth": {
							"auth_flow_type": "AUTH_FLOW_TYPE_OAUTH2_0",
							"predicate_key": [
								"credentials",
								"auth_type"
							],
							"predicate_value": "OAuth2.0"
						},
						"connectionSpecification": {
							"$schema": "http://json-schema.org/draft-07/schema#",
							"additionalProperties": true,
							"title": "Snowflake Destination Spec"
						},
						"documentationUrl": "https://docs.airbyte.io/integrations/destinations/snowflake",
						"supported_destination_sync_modes": [
							"SUPPORTED_DESTINATION_SYNC_MODES_OVERWRITE",
							"SUPPORTED_DESTINATION_SYNC_MODES_APPEND",
							"SUPPORTED_DESTINATION_SYNC_MODES_APPEND_DEDUP"
						],
						"supportsDBT": true
					}
				}
			]`,
		},
	}
	for _, test := range tests {

		var jsonSliceMap []map[string]interface{}
		if err := json.Unmarshal([]byte(test.inputJSON), &jsonSliceMap); err != nil {
			log.Fatalf("Unmarshal: %v", err)
		}

		ConvertAllJSONEnumValueToProtoStyle(enumRegistry, jsonSliceMap)

		if b, err := json.Marshal(jsonSliceMap); err != nil {
			log.Fatalf("Marshal: %v", err)
		} else {
			buf := &bytes.Buffer{}
			if err := json.Compact(buf, []byte(test.expectedJSON)); err != nil {
				log.Fatalf("Compact: %v", err)
			}
			if string(b) != buf.String() {
				t.Errorf("\nGot: %v\n Want: %v", string(b), buf.String())
			}
		}
	}
}
