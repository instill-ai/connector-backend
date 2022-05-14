import http from "k6/http";
import { check, group } from "k6";

import * as sourceConnectorDefinition from './rest-source-connector-definition.js';
import * as destinationConnectorDefinition from './rest-destination-connector-definition.js';
import * as sourceConnector from './rest-source-connector.js';
import * as destinationConnector from './rest-destination-connector.js';
import * as constant from "./const.js";

const connectorHost = "http://localhost:8082";

export let options = {
  setupTimeout: '300s',
  insecureSkipTLSVerify: true,
  thresholds: {
    checks: ["rate == 1.0"],
  },
};

export function setup() {}

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
  sourceConnector.CheckLookUp()
  sourceConnector.CheckRename()

  // Destination connectors
  destinationConnector.CheckCreate()
  destinationConnector.CheckList()
  destinationConnector.CheckGet()
  destinationConnector.CheckUpdate()
  destinationConnector.CheckLookUp()
  destinationConnector.CheckRename()

}

export function teardown(data) {}
