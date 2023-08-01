import http from "k6/http";
import { check, group } from "k6";

import { connectorPublicHost, pipelinePublicHost } from "./const.js"

import * as constant from "./const.js"
import * as dataConnectorDefinition from './rest-data-connector-definition.js';
import * as dataConnectorPublic from './rest-data-connector-public.js';
import * as dataConnectorPublicWithJwt from './rest-data-connector-public-with-jwt.js';
import * as dataConnectorPrivate from './rest-data-connector-private.js';

export let options = {
  setupTimeout: '300s',
  insecureSkipTLSVerify: true,
  thresholds: {
    checks: ["rate == 1.0"],
  },
};

export function setup() {

  group("Connector API: Pre delete all connector", () => {
    for (const connector of http.request("GET", `${connectorPublicHost}/v1alpha/connectors`).json("connectors")) {
      check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${connector.id}`), {
        [`DELETE /v1alpha/connectors/${connector.id} response status is 204`]: (r) => r.status === 204,
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

    // data connectors
    dataConnectorPrivate.CheckList()
    dataConnectorPrivate.CheckLookUp()

    // data public with jwt-sub
    dataConnectorPublicWithJwt.CheckCreate()
    dataConnectorPublicWithJwt.CheckList()
    dataConnectorPublicWithJwt.CheckGet()
    dataConnectorPublicWithJwt.CheckUpdate()
    dataConnectorPublicWithJwt.CheckLookUp()
    dataConnectorPublicWithJwt.CheckState()
    dataConnectorPublicWithJwt.CheckRename()
    dataConnectorPublicWithJwt.CheckExecute()
    dataConnectorPublicWithJwt.CheckTest()
  }


  // data connector definitions
  dataConnectorDefinition.CheckList()
  dataConnectorDefinition.CheckGet()

  // data connectors
  dataConnectorPublic.CheckCreate()
  dataConnectorPublic.CheckList()
  dataConnectorPublic.CheckGet()
  dataConnectorPublic.CheckUpdate()
  dataConnectorPublic.CheckConnect()
  dataConnectorPublic.CheckLookUp()
  dataConnectorPublic.CheckState()
  dataConnectorPublic.CheckRename()
  dataConnectorPublic.CheckExecute()
  dataConnectorPublic.CheckTest()

}

export function teardown(data) {
  group("Connector API: Delete all pipelines created by this test", () => {
    for (const pipeline of http.request("GET", `${pipelinePublicHost}/v1alpha/pipelines?page_size=100`).json("pipelines")) {
      check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${pipeline.id}`), {
        [`DELETE /v1alpha/pipelines response status is 204`]: (r) => r.status === 204,
      });
    }
  });
}
