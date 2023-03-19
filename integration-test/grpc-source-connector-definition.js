import grpc from 'k6/net/grpc';
import {
    check,
    group
} from "k6";

import * as constant from "./const.js"

import {
    deepEqual
} from "./helper.js"

const client = new grpc.Client();
client.load(['proto/vdp/connector/v1alpha'], 'connector_public_service.proto');

export function CheckList() {

    group("Connector API: List source connector definitions", () => {

        client.connect(constant.connectorGRPCHost, {
            plaintext: true
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectorDefinitions', {}, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectorDefinitions response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectorDefinitions response sourceConnectorDefinitions array`]: (r) => Array.isArray(r.message.sourceConnectorDefinitions),
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectorDefinitions response totalSize > 0`]: (r) => r.message.totalSize > 0,
        });

        var limitedRecords = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectorDefinitions', {}, {})
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectorDefinitions', {
            pageSize: 0
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectorDefinitions pageSize=0 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectorDefinitions pageSize=0 response sourceConnectorDefinitions length = 1`]: (r) => r.message.sourceConnectorDefinitions.length === limitedRecords.message.sourceConnectorDefinitions.length,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectorDefinitions', {
            pageSize: 1
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectorDefinitions pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectorDefinitions pageSize=1 response sourceConnectorDefinitions length = 1`]: (r) => r.message.sourceConnectorDefinitions.length === 1,
        });

        var pageRes = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectorDefinitions', {
            pageSize: 1
        }, {})
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectorDefinitions', {
            pageSize: 1,
            pageToken: pageRes.message.nextPageToken
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectorDefinitions pageSize=1 pageToken=${pageRes.message.nextPageToken} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectorDefinitions pageSize=1 pageToken=${pageRes.message.nextPageToken} response sourceConnectorDefinitions length = 1`]: (r) => r.message.sourceConnectorDefinitions.length === 1,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectorDefinitions', {
            pageSize: 1,
            view: "VIEW_BASIC"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectorDefinitions pageSize=1 view=VIEW_BASIC response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectorDefinitions pageSize=1 view=VIEW_BASIC response sourceConnectorDefinitions connectorDefinition spec is null`]: (r) => r.message.sourceConnectorDefinitions[0].connectorDefinition.spec === null,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectorDefinitions', {
            pageSize: 1,
            view: "VIEW_FULL"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectorDefinitions pageSize=1 view=VIEW_FULL response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectorDefinitions pageSize=1 view=VIEW_FULL response sourceConnectorDefinitions connectorDefinition spec is not null`]: (r) => r.message.sourceConnectorDefinitions[0].connectorDefinition.spec !== null,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectorDefinitions', {
            pageSize: 1,
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectorDefinitions pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectorDefinitions pageSize=1 response sourceConnectorDefinitions connectorDefinition spec is null`]: (r) => r.message.sourceConnectorDefinitions[0].connectorDefinition.spec === null,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectorDefinitions', {
            pageSize: limitedRecords.message.totalSize,
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectorDefinitions pageSize=${limitedRecords.message.totalSize} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectorDefinitions pageSize=${limitedRecords.message.totalSize} response nextPageToken is null`]: (r) => r.message.nextPageToken === "",
        });

        client.close();
    });
}

export function CheckGet() {
    group("Connector API: Get source connector definition", () => {
        client.connect(constant.connectorGRPCHost, {
            plaintext: true
        });

        var allRes = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectorDefinitions', {}, {})
        var def = allRes.message.sourceConnectorDefinitions[0]

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetSourceConnectorDefinition', {
            name: `source-connector-definitions/${def.id}`
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/GetSourceConnectorDefinition id=${def.id}} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/GetSourceConnectorDefinition id=${def.id} response has the exact record`]: (r) =>  deepEqual(r.message.sourceConnectorDefinition, def),
            [`vdp.connector.v1alpha.ConnectorPublicService/GetSourceConnectorDefinition id=${def.id} has the non-empty resource name ${def.name}`]: (r) => r.message.sourceConnectorDefinition.name != "",
            [`vdp.connector.v1alpha.ConnectorPublicService/GetSourceConnectorDefinition id=${def.id} has the resource name ${def.name}`]: (r) => r.message.sourceConnectorDefinition.name === def.name,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetSourceConnectorDefinition', {
            name: `source-connector-definitions/${def.id}`,
            view: "VIEW_BASIC"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/GetSourceConnectorDefinition id=${def.id}} view=VIEW_BASIC response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/GetSourceConnectorDefinition id=${def.id} view=VIEW_BASIC response sourceConnectorDefinition.connectorDefinition.spec is null`]: (r) =>  r.message.sourceConnectorDefinition.connectorDefinition.spec === null,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetSourceConnectorDefinition', {
            name: `source-connector-definitions/${def.id}`,
            view: "VIEW_FULL"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/GetSourceConnectorDefinition id=${def.id}} view=VIEW_FULL response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/GetSourceConnectorDefinition id=${def.id} view=VIEW_FULL response sourceConnectorDefinition.connectorDefinition.spec is not null`]: (r) =>  r.message.sourceConnectorDefinition.connectorDefinition.spec !== null,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetSourceConnectorDefinition', {
            name: `source-connector-definitions/${def.id}`,
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/GetSourceConnectorDefinition id=${def.id}} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/GetSourceConnectorDefinition id=${def.id} response sourceConnectorDefinition.connectorDefinition.spec is null`]: (r) =>  r.message.sourceConnectorDefinition.connectorDefinition.spec === null,
        });

        client.close();
    });
}