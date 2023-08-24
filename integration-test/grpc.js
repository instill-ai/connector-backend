import grpc from 'k6/net/grpc';
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

  group("Connector API: Pre delete all connector", () => {
    for (const connector of client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources', {}, {}).message.connectorResources) {
      check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
        name: `${constant.namespace}/connector-resources/${connector.id}`
      }), {
        [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
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

  if (!constant.apiGatewayMode) {
    // data connector private
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

  // data connector Definitions
  dataConnectorDefinition.CheckList()
  dataConnectorDefinition.CheckGet()

  // data connectors
  dataConnectorPublic.CheckCreate()
  dataConnectorPublic.CheckList()
  dataConnectorPublic.CheckGet()
  dataConnectorPublic.CheckUpdate()
  dataConnectorPublic.CheckLookUp()
  dataConnectorPublic.CheckState()
  dataConnectorPublic.CheckRename()
  dataConnectorPublic.CheckExecute()
  dataConnectorPublic.CheckTest()

}

export function teardown(data) {

  group("Connector API: Delete all pipelines created by this test", () => {
    client.connect(constant.pipelineGRPCPublicHost, {
      plaintext: true
    });

    for (const pipeline of client.invoke('vdp.pipeline.v1alpha.PipelinePublicService/ListPipelines', {
      pageSize: 1000
    }, {}).message.pipelines) {
      check(client.invoke(`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline`, {
        name: `pipelines/${pipeline.id}`
      }), {
        [`vdp.pipeline.v1alpha.PipelinePublicService/DeletePipeline response StatusOK`]: (r) => r.status === grpc.StatusOK,
      });
    }

    client.close();
  });

  client.connect(constant.connectorGRPCPublicHost, {
    plaintext: true
  });
  group("Connector API: Delete all connector created by this test", () => {
    for (const connector of client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources', {}, {}).message.connectorResources) {
      check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
        name: `${constant.namespace}/connector-resources/${connector.id}`
      }), {
        [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
      });
    }
  });

  client.close();
}
