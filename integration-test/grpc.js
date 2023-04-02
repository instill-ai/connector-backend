import grpc from 'k6/net/grpc';
import {
  check,
  group
} from 'k6';

import * as constant from "./const.js"
import * as sourceConnectorDefinition from './grpc-source-connector-definition.js';
import * as destinationConnectorDefinition from './grpc-destination-connector-definition.js';
import * as sourceConnectorPublic from './grpc-source-connector-public.js';
import * as destinationConnectorPublic from './grpc-destination-connector-public.js';
import * as sourceConnectorPrivate from './grpc-source-connector-private.js';
import * as destinationConnectorPrivate from './grpc-destination-connector-private.js';

const client = new grpc.Client();
client.load(['proto/vdp/connector/v1alpha'], 'connector_public_service.proto');

export let options = {
  setupTimeout: '10s',
  insecureSkipTLSVerify: true,
  thresholds: {
    checks: ["rate == 1.0"],
  },
};

export function setup() {
  client.connect(constant.connectorGRPCPublicHost, {
    plaintext: true
  });

  group("Connector API: Pre delete all source connector", () => {
    for (const srcConnector of client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors', {}, {}).message.sourceConnectors) {
      check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector`, {
        name: `source-connectors/${srcConnector.id}`
      }), {
        [`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector ${srcConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
      });
    }
  });

  group("Connector API: Pre delete all destination connector", () => {
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

export default function (data) {

  /*
   * Connector API - API CALLS
   */

  // Health check
  group("Connector API: Health check", () => {
    client.connect(constant.connectorGRPCPublicHost, {
      plaintext: true
    });
    const response = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/Liveness', {});
    check(response, {
      'Status is OK': (r) => r && r.status === grpc.StatusOK,
      'Response status is SERVING_STATUS_SERVING': (r) => r && r.message.healthCheckResponse.status === "SERVING_STATUS_SERVING",
    });
    client.close();
  });

  if (__ENV.MODE != "api-gateway" && __ENV.MODE != "localhost") {
    // Source connector private
    sourceConnectorPrivate.CheckList()
    sourceConnectorPrivate.CheckGet()
    sourceConnectorPrivate.CheckLookUp()

    // Destination connector Admin
    destinationConnectorPrivate.CheckList()
    destinationConnectorPrivate.CheckGet()
    destinationConnectorPrivate.CheckLookUp()
  }

  // Source connector Definitions
  sourceConnectorDefinition.CheckList()
  sourceConnectorDefinition.CheckGet()

  // Destination connector Definitions
  destinationConnectorDefinition.CheckList()
  destinationConnectorDefinition.CheckGet()

  // Source connector
  sourceConnectorPublic.CheckCreate()
  sourceConnectorPublic.CheckList()
  sourceConnectorPublic.CheckGet()
  sourceConnectorPublic.CheckUpdate()
  sourceConnectorPublic.CheckDelete()
  sourceConnectorPublic.CheckLookUp()
  sourceConnectorPublic.CheckState()
  sourceConnectorPublic.CheckRename()

  // Destination connectors
  destinationConnectorPublic.CheckCreate()
  destinationConnectorPublic.CheckList()
  destinationConnectorPublic.CheckGet()
  destinationConnectorPublic.CheckUpdate()
  destinationConnectorPublic.CheckLookUp()
  destinationConnectorPublic.CheckState()
  destinationConnectorPublic.CheckRename()
  destinationConnectorPublic.CheckWrite()

}

export function teardown(data) {
  client.connect(constant.connectorGRPCPublicHost, {
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
