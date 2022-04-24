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

export function setup() {
  constant
}

export default function (data) {
  let res;

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

  // // Destination connector definitions
  destinationDefinition.CheckList()
  destinationDefinition.CheckGet()

  // Connector
  connector.CheckCreate()
  connector.CheckUpdate()
  connector.CheckGet()

}

export function teardown(data) {
  // group("Connector API: Delete all connectors created by this test", () => {
  //   for (const pipeline of http
  //     .request("GET", `${connectorHost}/pipelines`, null, {
  //       headers: genHeader(
  //         "application/json"
  //       ),
  //     })
  //     .json("pipelines")) {
  //     check(pipeline, {
  //       "GET /clients response contents[*] id": (c) => c.id !== undefined,
  //     });

  //     check(
  //       http.request("DELETE", `${connectorHost}/pipelines/${pipeline.name}`, null, {
  //         headers: genHeader("application/json"),
  //       }),
  //       {
  //         [`DELETE /pipelines/${pipeline.name} response status is 204`]: (r) =>
  //           r.status === 204,
  //       }
  //     );
  //   }
  // });
}
