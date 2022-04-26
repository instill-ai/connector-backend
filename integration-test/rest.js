import http from "k6/http";
import { check, group } from "k6";

import * as sourceDefinition from './rest-source-definition.js';
import * as destinationDefinition from './rest-destination-definition.js';
import * as connector from './rest-connector.js';
import * as constant from "./const.js";

const connectorHost = "http://localhost:8080";

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
    check(http.request("GET", `${connectorHost}/health/connector`), {
      "GET /health/connector response status is 200": (r) => r.status === 200,
    });
  });

  // Source connector definitions
  sourceDefinition.CheckList()
  sourceDefinition.CheckGet()

  // Destination connector definitions
  destinationDefinition.CheckList()
  destinationDefinition.CheckGet()

  // Connector
  connector.CheckCreate()
  connector.CheckUpdate()
  connector.CheckGet()

}

export function teardown(data) {}
