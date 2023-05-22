import http from "k6/http";
import { check, group } from "k6";

import { connectorPublicHost, pipelinePublicHost } from "./const.js"

import * as constant from "./const.js"
import * as sourceConnectorDefinition from './rest-source-connector-definition.js';
import * as destinationConnectorDefinition from './rest-destination-connector-definition.js';
import * as sourceConnectorPublic from './rest-source-connector-public.js';
import * as destinationConnectorPublic from './rest-destination-connector-public.js';
import * as sourceConnectorPublicWithJwt from './rest-source-connector-public-with-jwt.js';
import * as destinationConnectorPublicWithJwt from './rest-destination-connector-public-with-jwt.js';
import * as sourceConnectorPrivate from './rest-source-connector-private.js';
import * as destinationConnectorPrivate from './rest-destination-connector-private.js';

export let options = {
  setupTimeout: '300s',
  insecureSkipTLSVerify: true,
  thresholds: {
    checks: ["rate == 1.0"],
  },
};

export function setup() {

  group("Connector API: Pre delete all source connector", () => {
    for (const srcConnector of http.request("GET", `${connectorPublicHost}/v1alpha/source-connectors`).json("source_connectors")) {
      check(http.request("DELETE", `${connectorPublicHost}/v1alpha/source-connectors/${srcConnector.id}`), {
        [`DELETE /v1alpha/source-connectors/${srcConnector.id} response status is 204`]: (r) => r.status === 204,
      });
    }
  });

  group("Connector API: Pre delete all destination connector", () => {
    for (const desConnector of http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors`).json("destination_connectors")) {
      check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${desConnector.id}`), {
        [`DELETE /v1alpha/destination-connectors/${desConnector.id} response status is 204`]: (r) => r.status === 204,
      });
    }
  });
}

export default function (data) {

  /*
   * Connector API - API CALLS
   */

  // Health check
  group("Connector API: Health check", () => {
    check(http.request("GET", `${connectorPublicHost}/v1alpha/health/connector`), {
      "GET /health/connector response status is 200": (r) => r.status === 200,
    });
  });

  // private API do not expose to public.
  if (!constant.apiGatewayMode) {
    // Source connectors
    sourceConnectorPrivate.CheckList()
    sourceConnectorPrivate.CheckLookUp()

    // Destination connectors
    destinationConnectorPrivate.CheckList()
    destinationConnectorPrivate.CheckLookUp()

    // Source public with jwt-sub
    sourceConnectorPublicWithJwt.CheckCreate()
    sourceConnectorPublicWithJwt.CheckList()
    sourceConnectorPublicWithJwt.CheckGet()
    sourceConnectorPublicWithJwt.CheckUpdate()
    sourceConnectorPublicWithJwt.CheckDelete()
    sourceConnectorPublicWithJwt.CheckLookUp()
    sourceConnectorPublicWithJwt.CheckState()
    sourceConnectorPublicWithJwt.CheckRename()
    sourceConnectorPublicWithJwt.CheckTest()

    // Destination public with jwt-sub
    destinationConnectorPublicWithJwt.CheckCreate()
    destinationConnectorPublicWithJwt.CheckList()
    destinationConnectorPublicWithJwt.CheckGet()
    destinationConnectorPublicWithJwt.CheckUpdate()
    destinationConnectorPublicWithJwt.CheckLookUp()
    destinationConnectorPublicWithJwt.CheckState()
    destinationConnectorPublicWithJwt.CheckRename()
    destinationConnectorPublicWithJwt.CheckWrite()
    destinationConnectorPublicWithJwt.CheckTest()
  }

  // Source connector definitions
  sourceConnectorDefinition.CheckList()
  sourceConnectorDefinition.CheckGet()

  // Destination connector definitions
  destinationConnectorDefinition.CheckList()
  destinationConnectorDefinition.CheckGet()

  // Source connectors
  sourceConnectorPublic.CheckCreate()
  sourceConnectorPublic.CheckList()
  sourceConnectorPublic.CheckGet()
  sourceConnectorPublic.CheckUpdate()
  sourceConnectorPublic.CheckDelete()
  sourceConnectorPublic.CheckLookUp()
  sourceConnectorPublic.CheckState()
  sourceConnectorPublic.CheckRename()
  sourceConnectorPublic.CheckTest()

  // Destination connectors
  destinationConnectorPublic.CheckCreate()
  destinationConnectorPublic.CheckList()
  destinationConnectorPublic.CheckGet()
  destinationConnectorPublic.CheckUpdate()
  destinationConnectorPublic.CheckLookUp()
  destinationConnectorPublic.CheckState()
  destinationConnectorPublic.CheckRename()
  destinationConnectorPublic.CheckWrite()
  destinationConnectorPublic.CheckTest()

}

export function teardown(data) {
  group("Connector API: Delete all pipelines created by this test", () => {
    for (const pipeline of http.request("GET", `${pipelinePublicHost}/v1alpha/pipelines?page_size=100`).json("pipelines")) {
      check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${pipeline.id}`), {
        [`DELETE /v1alpha/pipelines response status is 204`]: (r) => r.status === 204,
      });
    }
  });

  group("Connector API: Delete all source connector created by this test", () => {
    for (const srcConnector of http.request("GET", `${connectorPublicHost}/v1alpha/source-connectors`).json("source_connectors")) {
      check(http.request("DELETE", `${connectorPublicHost}/v1alpha/source-connectors/${srcConnector.id}`), {
        [`DELETE /v1alpha/source-connectors/${srcConnector.id} response status is 204`]: (r) => r.status === 204,
      });
    }
  });

  group("Connector API: Delete all destination connector created by this test", () => {
    for (const desConnector of http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors`).json("destination_connectors")) {
      check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${desConnector.id}`), {
        [`DELETE /v1alpha/destination-connectors/${desConnector.id} response status is 204`]: (r) => r.status === 204,
      });
    }
  });
}
