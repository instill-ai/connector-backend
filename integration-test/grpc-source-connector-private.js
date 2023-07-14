import grpc from 'k6/net/grpc';
import {
    check,
    group
} from "k6";

import * as constant from "./const.js"
import * as helper from "./helper.js"

const clientPrivate = new grpc.Client();
const clientPublic = new grpc.Client();
clientPrivate.load(['proto/vdp/connector/v1alpha'], 'connector_private_service.proto');
clientPublic.load(['proto/vdp/connector/v1alpha'], 'connector_public_service.proto');

export function CheckList() {

    group("Connector API: List source connectors by admin", () => {
        clientPrivate.connect(constant.connectorGRPCPrivateHost, {
            plaintext: true
        });

        clientPublic.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin', {}, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin response connectors array is 0 length`]: (r) => r.message.connectors.length === 0,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin response nextPageToken is empty`]: (r) => r.message.nextPageToken === "",
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin response totalSize is 0`]: (r) => r.message.totalSize == 0,
        });

        var reqBodies = [];
        reqBodies[0] = {
            "id": "trigger",
            "connector_definition_name": constant.srcDefRscName,
            "configuration": {}
        }

        // Create connectors
        for (const reqBody of reqBodies) {
            check(clientPublic.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
                connector: reqBody
            }), {
                [`vdp.connector.v1alpha.ConnectorPublicService/CreateConnector x${reqBodies.length} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            });
        }

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin', {}, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin response has connectors array`]: (r) => Array.isArray(r.message.connectors),
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin response has totalSize = ${reqBodies.length}`]: (r) => r.message.totalSize == reqBodies.length,
        });

        var limitedRecords = clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin', {}, {})
        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin', {
            pageSize: 0
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=0 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=0 response has connectors length`]: (r) => r.message.connectors.length === limitedRecords.message.connectors.length,
        });

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin', {
            pageSize: 1
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=1 response has connectors length`]: (r) => r.message.connectors.length === 1,
        });

        var pageRes = clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin', {
            pageSize: 1
        }, {})
        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin', {
            pageSize: 1,
            pageToken: pageRes.message.nextPageToken
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=1 pageToken=${pageRes.message.nextPageToken} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=1 pageToken=${pageRes.message.nextPageToken} response has connectors length`]: (r) => r.message.connectors.length === 1,
        });

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin', {
            pageSize: 1,
            view: "VIEW_BASIC"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=1 view=VIEW_BASIC response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=1 view=VIEW_BASIC response has connectors[0].configuration is null`]: (r) => r.message.connectors[0].configuration === null,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=1 view=VIEW_BASIC response has connectors[0].owner is UUID`]: (r) => helper.isValidOwner(r.message.connectors[0].user),
        });
        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin', {
            pageSize: 1,
            view: "VIEW_FULL"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=1 view=VIEW_FULL response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=1 view=VIEW_FULL response has connectors[0].configuration is not null`]: (r) => r.message.connectors[0].configuration !== null,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=1 view=VIEW_FULL response has connectors[0].connectorDefinitionDetail is not null`]: (r) => r.message.connectors[0].connectorDefinitionDetail !== null,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=1 view=VIEW_FULL response has connectors[0].configuration is {}`]: (r) => Object.keys(r.message.connectors[0].configuration).length === 0,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=1 view=VIEW_FULL response has connectors[0].owner is UUID`]: (r) => helper.isValidOwner(r.message.connectors[0].user),
        });

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin', {
            pageSize: 1,
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=1 response has connectors[0].configuration is null`]: (r) => r.message.connectors[0].configuration === null,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=1 response has connectors[0].owner is UUID`]: (r) => helper.isValidOwner(r.message.connectors[0].user),
        });

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin', {
            pageSize: `${limitedRecords.message.totalSize}`,
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=${limitedRecords.message.totalSize} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=${limitedRecords.message.totalSize} response nextPageToken is empty`]: (r) => r.message.nextPageToken === "",
        });

        // Delete the destination connectors
        for (const reqBody of reqBodies) {
            check(clientPublic.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
                name: `connectors/${reqBody.id}`
            }), {
                [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector x${reqBodies.length} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            });
        }
    });

    clientPublic.close();
}

export function CheckLookUp() {

    group("Connector API: Look up source connectors by UID by admin", () => {

        clientPrivate.connect(constant.connectorGRPCPrivateHost, {
            plaintext: true
        });

        clientPublic.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var srcConnector = {
            "id": "trigger",
            "connector_definition_name": constant.srcDefRscName,
            "configuration": {}
        }

        var resHTTP = clientPublic.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: srcConnector
        })

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/LookUpConnectorAdmin', {
            permalink: `connectors/${resHTTP.message.connector.uid}`
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/LookUpConnectorAdmin permalink=connectors/${resHTTP.message.connector.uid} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/LookUpConnectorAdmin permalink=connectors/${resHTTP.message.connector.uid} response connector uid`]: (r) => r.message.connector.id === srcConnector.id,
            [`vdp.connector.v1alpha.ConnectorPrivateService/LookUpConnectorAdmin permalink=connectors/${resHTTP.message.connector.uid} response connector id`]: (r) => r.message.connector.connectorDefinitionName === constant.srcDefRscName,
            [`vdp.connector.v1alpha.ConnectorPrivateService/LookUpConnectorAdmin permalink=connectors/${resHTTP.message.connector.uid} response connector owner is UUID`]: (r) => helper.isValidOwner(r.message.connector.user),
        });

        check(clientPublic.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/trigger`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector trigger response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        clientPublic.close();
    });
}
