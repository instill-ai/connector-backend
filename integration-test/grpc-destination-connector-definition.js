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

    group("Connector API: List destination connector definitions", () => {

        client.connect(constant.connectorGRPCHost, {
            plaintext: true
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectorDefinitions', {}, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectorDefinitions response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectorDefinitions response destinationConnectorDefinitions array`]: (r) => Array.isArray(r.message.destinationConnectorDefinitions),
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectorDefinitions response totalSize > 0`]: (r) => r.message.totalSize > 0,
        });

        var limitedRecords = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectorDefinitions', {}, {})
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectorDefinitions', {
            pageSize: 0
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectorDefinitions pageSize=0 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectorDefinitions pageSize=0 response destinationConnectorDefinitions length = 1`]: (r) => r.message.destinationConnectorDefinitions.length === limitedRecords.message.destinationConnectorDefinitions.length,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectorDefinitions', {
            pageSize: 1
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectorDefinitions pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectorDefinitions pageSize=1 response destinationConnectorDefinitions length = 1`]: (r) => r.message.destinationConnectorDefinitions.length === 1,
        });

        var pageRes = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectorDefinitions', {
            pageSize: 1
        }, {})
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectorDefinitions', {
            pageSize: 1,
            pageToken: pageRes.message.nextPageToken
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectorDefinitions pageSize=1 pageToken=${pageRes.message.nextPageToken} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectorDefinitions pageSize=1 pageToken=${pageRes.message.nextPageToken} response destinationConnectorDefinitions length = 1`]: (r) => r.message.destinationConnectorDefinitions.length === 1,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectorDefinitions', {
            pageSize: 1,
            view: "VIEW_BASIC"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectorDefinitions pageSize=1 view=VIEW_BASIC response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectorDefinitions pageSize=1 view=VIEW_BASIC response destinationConnectorDefinitions connectorDefinition spec is null`]: (r) => r.message.destinationConnectorDefinitions[0].connectorDefinition.spec === null,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectorDefinitions', {
            pageSize: 1,
            view: "VIEW_FULL"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectorDefinitions pageSize=1 view=VIEW_FULL response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectorDefinitions pageSize=1 view=VIEW_FULL response destinationConnectorDefinitions connectorDefinition spec is not null`]: (r) => r.message.destinationConnectorDefinitions[0].connectorDefinition.spec !== null,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectorDefinitions', {
            pageSize: 1,
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectorDefinitions pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectorDefinitions pageSize=1 response destinationConnectorDefinitions connectorDefinition spec is null`]: (r) => r.message.destinationConnectorDefinitions[0].connectorDefinition.spec === null,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectorDefinitions', {
            pageSize: limitedRecords.message.totalSize,
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectorDefinitions pageSize=${limitedRecords.message.totalSize} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectorDefinitions pageSize=${limitedRecords.message.totalSize} response nextPageToken is null`]: (r) => r.message.nextPageToken === "",
        });

        client.close();
    });
}

export function CheckGet() {
    group("Connector API: Get destination connector definition", () => {
        client.connect(constant.connectorGRPCHost, {
            plaintext: true
        });

        var allRes = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectorDefinitions', {}, {})
        var def = allRes.message.destinationConnectorDefinitions[0]

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnectorDefinition', {
            name: `destination-connector-definitions/${def.id}`
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnectorDefinition id=${def.id}} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnectorDefinition id=${def.id} response has the exact record`]: (r) =>  deepEqual(r.message.destinationConnectorDefinition, def),
            [`vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnectorDefinition id=${def.id} has the non-empty resource name ${def.name}`]: (r) => r.message.destinationConnectorDefinition.name != "",
            [`vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnectorDefinition id=${def.id} has the resource name ${def.name}`]: (r) => r.message.destinationConnectorDefinition.name === def.name,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnectorDefinition', {
            name: `destination-connector-definitions/${def.id}`,
            view: "VIEW_BASIC"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnectorDefinition id=${def.id}} view=VIEW_BASIC response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnectorDefinition id=${def.id} view=VIEW_BASIC response destinationConnectorDefinition.connectorDefinition.spec is null`]: (r) =>  r.message.destinationConnectorDefinition.connectorDefinition.spec === null,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnectorDefinition', {
            name: `destination-connector-definitions/${def.id}`,
            view: "VIEW_FULL"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnectorDefinition id=${def.id}} view=VIEW_FULL response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnectorDefinition id=${def.id} view=VIEW_FULL response destinationConnectorDefinition.connectorDefinition.spec is not null`]: (r) =>  r.message.destinationConnectorDefinition.connectorDefinition.spec !== null,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnectorDefinition', {
            name: `destination-connector-definitions/${def.id}`,
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnectorDefinition id=${def.id}} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnectorDefinition id=${def.id} response destinationConnectorDefinition.connectorDefinition.spec is null`]: (r) =>  r.message.destinationConnectorDefinition.connectorDefinition.spec === null,
        });

        client.close();
    });
}