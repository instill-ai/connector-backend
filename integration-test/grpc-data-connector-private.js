import grpc from 'k6/net/grpc';
import {
    check,
    group,
    sleep
} from "k6";
import {
    randomString
} from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js"
import * as helper from "./helper.js"

const clientPrivate = new grpc.Client();
const clientPublic = new grpc.Client();
clientPrivate.load(['proto/vdp/connector/v1alpha'], 'connector_private_service.proto');
clientPublic.load(['proto/vdp/connector/v1alpha'], 'connector_public_service.proto');

export function CheckList(metadata) {

    group("Connector API: List data connector-resources by admin", () => {

        clientPrivate.connect(constant.connectorGRPCPrivateHost, {
            plaintext: true
        });

        clientPublic.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin', {}, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin response connector-resources array is 0 length`]: (r) => r.message.connectorResources.length === 0,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin response nextPageToken is empty`]: (r) => r.message.nextPageToken === "",
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin response totalSize is 0`]: (r) => r.message.totalSize == 0,
        });

        const numConnectors = 10
        var reqBodies = [];
        for (var i = 0; i < numConnectors; i++) {
            reqBodies[i] = {
                "id": randomString(10),
                "connector_definition_name": constant.csvDstDefRscName,
                "description": randomString(50),
                "configuration": constant.csvDstConfig
            }
        }

        // Create connector_resources
        for (const reqBody of reqBodies) {
            var resDst = clientPublic.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
                parent: `${constant.namespace}`,
                connector_resource: reqBody
            }, metadata)
            clientPublic.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
                name: `${constant.namespace}/connector-resources/${resDst.message.connectorResource.id}`
            }, metadata)

            check(resDst, {
                [`vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource x${reqBodies.length} HTTP response StatusOK`]: (r) => r.status === grpc.StatusOK,
            });
        }

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin', {}, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResourceAdmin response has connectors array`]: (r) => Array.isArray(r.message.connectorResources),
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin response has totalSize = ${reqBodies.length}`]: (r) => r.message.totalSize == reqBodies.length,
        });

        var limitedRecords = clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin', {}, {})
        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin', {
            pageSize: 0
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin pageSize=0 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin pageSize=0 response all records`]: (r) => r.message.connectorResources.length === limitedRecords.message.connectorResources.length,
        });

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin', {
            pageSize: 1
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin pageSize=1 response size 1`]: (r) => r.message.connectorResources.length === 1,
        });

        var pageRes = clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin', {
            pageSize: 1
        }, {})

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin', {
            pageSize: 1,
            pageToken: `${pageRes.message.nextPageToken}`
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin pageSize=1 pageToken=${pageRes.message.nextPageToken} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin pageSize=1 pageToken=${pageRes.message.nextPageToken} response size 1`]: (r) => r.message.connectorResources.length === 1,
        });

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin', {
            pageSize: 1,
            view: "VIEW_BASIC"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin pageSize=1 view=VIEW_BASIC response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin pageSize=1 view=VIEW_BASIC response connectors[0].configuration is null`]: (r) => r.message.connectorResources[0].configuration === null,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin pageSize=1 view=VIEW_BASIC response connectors[0].owner is UUID`]: (r) => helper.isValidOwner(r.message.connectorResources[0].user),
        });

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin', {
            pageSize: 1,
            view: "VIEW_FULL"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin pageSize=1 view=VIEW_FULL response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin pageSize=1 view=VIEW_FULL response connectors[0].configuration is not null`]: (r) => r.message.connectorResources[0].configuration !== null,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin pageSize=1 view=VIEW_FULL response connectors[0].connectorDefinitionDetail is not null`]: (r) => r.message.connectorResources[0].connectorDefinitionDetail !== null,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin pageSize=1 view=VIEW_FULL response connectors[0].owner is UUID`]: (r) => helper.isValidOwner(r.message.connectorResources[0].user),
        });


        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin', {
            pageSize: 1,
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin pageSize=1 response connectors[0].configuration is null`]: (r) => r.message.connectorResources[0].configuration === null,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin pageSize=1 response connectors[0].owner is UUID`]: (r) => helper.isValidOwner(r.message.connectorResources[0].user),
        });

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin', {
            pageSize: `${limitedRecords.message.totalSize}`,
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin pageSize=${limitedRecords.message.totalSize} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorResourcesAdmin pageSize=${limitedRecords.message.totalSize} response nextPageToken is empty`]: (r) => r.message.nextPageToken === "",
        });

        // Delete the data connectors
        for (const reqBody of reqBodies) {
            check(clientPublic.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
                name: `${constant.namespace}/connector-resources/${reqBody.id}`
            }, metadata), {
                [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource x${reqBodies.length} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            });
        }

        clientPrivate.close();
        clientPublic.close();
    });
}

export function CheckLookUp(metadata) {

    group("Connector API: Look up data connectors by UID by admin", () => {

        clientPrivate.connect(constant.connectorGRPCPrivateHost, {
            plaintext: true
        });

        clientPublic.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = clientPublic.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
            parent: `${constant.namespace}`,
            connector_resource: csvDstConnector
        }, metadata)

        clientPublic.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        }, metadata)

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/LookUpConnectorResourceAdmin', {
            permalink: `connector-resources/${resCSVDst.message.connectorResource.uid}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/LookUpConnectorResourceAdmin CSV ${resCSVDst.message.connectorResource.uid} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/LookUpConnectorResourceAdmin CSV ${resCSVDst.message.connectorResource.uid} response connector id`]: (r) => r.message.connectorResource.uid === resCSVDst.message.connectorResource.uid,
            [`vdp.connector.v1alpha.ConnectorPrivateService/LookUpConnectorResourceAdmin CSV ${resCSVDst.message.connectorResource.uid} response connector connectorDefinition permalink`]: (r) => r.message.connectorResource.connectorDefinitionName === constant.csvDstDefRscName,
            [`vdp.connector.v1alpha.ConnectorPrivateService/LookUpConnectorResourceAdmin CSV ${resCSVDst.message.connectorResource.uid} response connector owner is UUID`]: (r) => helper.isValidOwner(r.message.connectorResource.user),
        });

        check(clientPublic.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
            name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        clientPublic.close();
    });
}
