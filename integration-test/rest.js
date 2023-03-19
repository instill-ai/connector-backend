import http from "k6/http";
import { check, group } from "k6";

import { connectorHost } from "./const.js"

import * as sourceConnectorDefinition from './rest-source-connector-definition.js';
import * as destinationConnectorDefinition from './rest-destination-connector-definition.js';
import * as sourceConnector from './rest-source-connector-public.js';
import * as destinationConnector from './rest-destination-connector-public.js';
import * as sourceConnectorAdmin from './rest-source-connector-private.js';
import * as destinationConnectorAdmin from './rest-destination-connector-private.js';

export let options = {
  setupTimeout: '300s',
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
    check(http.request("GET", `${connectorHost}/v1alpha/health/connector`), {
      "GET /health/connector response status is 200": (r) => r.status === 200,
    });
  });

  // Source connector definitions
  sourceConnectorDefinition.CheckList()
  sourceConnectorDefinition.CheckGet()

  // Destination connector definitions
  destinationConnectorDefinition.CheckList()
  destinationConnectorDefinition.CheckGet()

  // Source connectors
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

  // private API do not expose to public.
  if (__ENV.MODE == "private") { 
    // Source connectors
    sourceConnectorAdmin.CheckList()
    sourceConnectorAdmin.CheckGet()
    sourceConnectorAdmin.CheckLookUp()

    // Destination connectors
    destinationConnectorAdmin.CheckList()
    destinationConnectorAdmin.CheckGet()
    destinationConnectorAdmin.CheckLookUp()
  }

}

export function teardown(data) {
  group("Connector API: Delete all source connector created by this test", () => {
    for (const srcConnector of http.request("GET", `${connectorHost}/v1alpha/source-connectors`).json("source_connectors")) {
      check(http.request("DELETE", `${connectorHost}/v1alpha/source-connectors/${srcConnector.id}`), {
        [`DELETE /v1alpha/source-connectors/${srcConnector.id} response status is 204`]: (r) => r.status === 204,
      });
    }
  });

  group("Connector API: Delete all destination connector created by this test", () => {
    for (const desConnector of http.request("GET", `${connectorHost}/v1alpha/destination-connectors`).json("destination_connectors")) {
      check(http.request("DELETE", `${connectorHost}/v1alpha/destination-connectors/${desConnector.id}`), {
        [`DELETE /v1alpha/destination-connectors/${desConnector.id} response status is 204`]: (r) => r.status === 204,
      });
    }
  });
}
