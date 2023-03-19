import grpc from 'k6/net/grpc';
import {
  check,
  group
} from 'k6';

import * as constant from "./const.js"
import * as sourceConnector from './grpc-source-connector.js';
import * as destinationConnector from './grpc-destination-connector.js';
import * as sourceConnectorDefinition from './grpc-source-connector-definition.js';
import * as destinationConnectorDefinition from './grpc-destination-connector-definition.js';

const client = new grpc.Client();
client.load(['proto/vdp/connector/v1alpha'], 'connector_public_service.proto');

export let options = {
  setupTimeout: '10s',
  insecureSkipTLSVerify: true,
  thresholds: {
    checks: ["rate == 1.0"],
  },
};

export default function (data) {

  /*
   * Connector API - API CALLS
   */

  // Health check
  group("Connector API: Health check", () => {
    client.connect(constant.connectorGRPCHost, {
      plaintext: true
    });
    const response = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/Liveness', {});
    check(response, {
      'Status is OK': (r) => r && r.status === grpc.StatusOK,
      'Response status is SERVING_STATUS_SERVING': (r) => r && r.message.healthCheckResponse.status === "SERVING_STATUS_SERVING",
    });
  });

  // Source connector Definitions
  sourceConnectorDefinition.CheckList()
  sourceConnectorDefinition.CheckGet()

  // Destination connector Definitions
  destinationConnectorDefinition.CheckList()
  destinationConnectorDefinition.CheckGet()


  // Source connector
  sourceConnector.CheckCreate()
  sourceConnector.CheckList()
  sourceConnector.CheckGet()
  sourceConnector.CheckUpdate()
  sourceConnector.CheckDelete()
  sourceConnector.CheckLookUp()
  sourceConnector.CheckState()
  sourceConnector.CheckRename()

  // Destination connectors
  destinationConnector.CheckCreate()
  destinationConnector.CheckList()
  destinationConnector.CheckGet()
  destinationConnector.CheckUpdate()
  destinationConnector.CheckLookUp()
  destinationConnector.CheckState()
  destinationConnector.CheckRename()
  destinationConnector.CheckWrite()
}

export function teardown(data) {
  client.connect(constant.connectorGRPCHost, {
    plaintext: true
  });
  group("Connector API: Delete all source connector created by this test", () => {
    for (const srcConnector of client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors', {}, {}).message.sourceConnectors) {
      check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector`, {
        name: `source-connectors/${srcConnector.id}`
      }), {
        [`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector ${srcConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
      });
    }
  });

  group("Connector API: Delete all destination connector created by this test", () => {
    for (const desConnector of client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors', {}, {}).message.destinationConnectors) {
      check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
        name: `destination-connectors/${desConnector.id}`
      }), {
        [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${desConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
      });
    }
  });
  client.close();
}