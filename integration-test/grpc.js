import grpc from 'k6/net/grpc';
import http from 'k6/http';
import {
  check,
  group
} from 'k6';

import * as constant from "./const.js"
import * as dataConnectorDefinition from './grpc-data-connector-definition.js';
import * as dataConnectorPublic from './grpc-data-connector-public.js';
import * as dataConnectorPublicWithJwt from './grpc-data-connector-public-with-jwt.js';
import * as dataConnectorPrivate from './grpc-data-connector-private.js';

const client = new grpc.Client();
client.load(['proto/vdp/connector/v1alpha'], 'connector_public_service.proto');
client.load(['proto/vdp/pipeline/v1alpha'], 'pipeline_public_service.proto');

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

  var loginResp = http.request("POST", `${constant.mgmtPublicHost}/v1alpha/auth/login`, JSON.stringify({
    "username": constant.defaultUsername,
    "password": constant.defaultPassword,
  }))

  check(loginResp, {
    [`POST ${constant.mgmtPublicHost}/v1alpha/auth/login response status is 200`]: (
      r
    ) => r.status === 200,
  });

  var metadata = {
    "metadata": {
      "Authorization": `Bearer ${loginResp.json().access_token}`
    },
    "timeout": "600s",
  }


  group("Connector API: Pre delete all connector", () => {
    for (const connector of client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources', {}, {}).message.connectorResources) {
      check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
        name: `${constant.namespace}/connector-resources/${connector.id}`
      }, metadata), {
        [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
      });
    }
  });

  client.close();
  return metadata
}

export default function (metadata) {

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

  if (!constant.apiGatewayMode) {
    // data connector private
    dataConnectorPrivate.CheckList(metadata)
    dataConnectorPrivate.CheckLookUp(metadata)


  } else {

    // data public with jwt-sub
    dataConnectorPublicWithJwt.CheckCreate(metadata)
    dataConnectorPublicWithJwt.CheckList(metadata)
    dataConnectorPublicWithJwt.CheckGet(metadata)
    dataConnectorPublicWithJwt.CheckUpdate(metadata)
    dataConnectorPublicWithJwt.CheckLookUp(metadata)
    dataConnectorPublicWithJwt.CheckState(metadata)
    dataConnectorPublicWithJwt.CheckRename(metadata)
    dataConnectorPublicWithJwt.CheckExecute(metadata)
    dataConnectorPublicWithJwt.CheckTest(metadata)

    // data connector Definitions
    dataConnectorDefinition.CheckList(metadata)
    dataConnectorDefinition.CheckGet(metadata)

    // data connectors
    dataConnectorPublic.CheckCreate(metadata)
    dataConnectorPublic.CheckList(metadata)
    dataConnectorPublic.CheckGet(metadata)
    dataConnectorPublic.CheckUpdate(metadata)
    dataConnectorPublic.CheckLookUp(metadata)
    dataConnectorPublic.CheckState(metadata)
    dataConnectorPublic.CheckRename(metadata)
    dataConnectorPublic.CheckExecute(metadata)
    dataConnectorPublic.CheckTest(metadata)

  }

}

export function teardown(metadata) {

  group("Connector API: Delete all pipelines created by this test", () => {
    client.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true
    });

    for (const pipeline of client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines', {
      pageSize: 1000
    }, metadata).message.pipelines) {
      check(client.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserPipeline`, {
        name: `${constant.namespace}/pipelines/${pipeline.id}`
      }, metadata), {
        [`vdp.pipeline.v1alpha.PipelinePublicService/DeleteUserPipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
      });
    }

    client.close();
  });

  client.connect(constant.connectorGRPCPublicHost, {
    plaintext: true
  });
  group("Connector API: Delete all connector created by this test", () => {
    for (const connector of client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources', {}, metadata).message.connectorResources) {
      check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
        name: `${constant.namespace}/connector-resources/${connector.id}`
      }, metadata), {
        [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
      });
    }
  });

  client.close();
}
