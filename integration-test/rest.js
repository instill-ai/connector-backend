import http from "k6/http";
import { check, group } from "k6";

import * as sourceConnectorDefinition from './rest-source-connector-definition.js';
import * as destinationConnectorDefinition from './rest-destination-connector-definition.js';
import * as sourceConnector from './rest-source-connector.js';
import * as destinationConnector from './rest-destination-connector.js';

const pipelineHost = "http://pipeline-backend:8081"
const connectorHost = "http://connector-backend:8082";
const modelHost = "http://model-backend:8083"

export let options = {
  setupTimeout: '300s',
  insecureSkipTLSVerify: true,
  thresholds: {
    checks: ["rate == 1.0"],
  },
};

export function setup() { }

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

}

export function teardown(data) { }
