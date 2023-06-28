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

export function CheckList() {

    group("Connector API: List destination connectors by admin", () => {

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

        const numConnectors = 10
        var reqBodies = [];
        for (var i = 0; i < numConnectors; i++) {
            reqBodies[i] = {
                "id": randomString(10),
                "connector_definition": constant.csvDstDefRscName,
                "description": randomString(50),
                "configuration": constant.csvDstConfig
            }
        }

        // Create connectors
        for (const reqBody of reqBodies) {
            var resDstHTTP = clientPublic.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
                connector: reqBody
            })

            check(resDstHTTP, {
                [`vdp.connector.v1alpha.ConnectorPublicService/CreateConnector x${reqBodies.length} HTTP response StatusOK`]: (r) => r.status === grpc.StatusOK,
            });
        }

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin', {}, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors response has connectors array`]: (r) => Array.isArray(r.message.connectors),
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin response has totalSize = ${reqBodies.length}`]: (r) => r.message.totalSize == reqBodies.length,
        });

        var limitedRecords = clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin', {}, {})
        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin', {
            pageSize: 0
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=0 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=0 response all records`]: (r) => r.message.connectors.length === limitedRecords.message.connectors.length,
        });

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin', {
            pageSize: 1
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=1 response size 1`]: (r) => r.message.connectors.length === 1,
        });

        var pageRes = clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin', {
            pageSize: 1
        }, {})

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin', {
            pageSize: 1,
            pageToken: `${pageRes.message.nextPageToken}`
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=1 pageToken=${pageRes.message.nextPageToken} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=1 pageToken=${pageRes.message.nextPageToken} response size 1`]: (r) => r.message.connectors.length === 1,
        });

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin', {
            pageSize: 1,
            view: "VIEW_BASIC"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=1 view=VIEW_BASIC response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=1 view=VIEW_BASIC response connectors[0].configuration is null`]: (r) => r.message.connectors[0].configuration === null,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=1 view=VIEW_BASIC response connectors[0].owner is UUID`]: (r) => helper.isValidOwner(r.message.connectors[0].user ),
        });

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin', {
            pageSize: 1,
            view: "VIEW_FULL"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=1 view=VIEW_FULL response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=1 view=VIEW_FULL response connectors[0].configuration is null`]: (r) => r.message.connectors[0].configuration !== null,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=1 view=VIEW_FULL response connectors[0].owner is UUID`]: (r) => helper.isValidOwner(r.message.connectors[0].user ),
        });


        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin', {
            pageSize: 1,
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=1 response connectors[0].configuration is null`]: (r) => r.message.connectors[0].configuration === null,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListConnectorsAdmin pageSize=1 response connectors[0].owner is UUID`]: (r) => helper.isValidOwner(r.message.connectors[0].user ),
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

        clientPrivate.close();
        clientPublic.close();
    });
}

export function CheckLookUp() {

    group("Connector API: Look up destination connectors by UID by admin", () => {

        clientPrivate.connect(constant.connectorGRPCPrivateHost, {
            plaintext: true
        });

        clientPublic.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = clientPublic.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: csvDstConnector
        })

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/LookUpConnectorAdmin', {
            permalink: `destination_connector/${resCSVDst.message.connector.uid}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/LookUpConnectorAdmin CSV ${resCSVDst.message.connector.uid} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/LookUpConnectorAdmin CSV ${resCSVDst.message.connector.uid} response connector id`]: (r) => r.message.connector.uid === resCSVDst.message.connector.uid,
            [`vdp.connector.v1alpha.ConnectorPrivateService/LookUpConnectorAdmin CSV ${resCSVDst.message.connector.uid} response connector connectorDefinition permalink`]: (r) => r.message.connector.connectorDefinition === constant.csvDstDefRscName,
            [`vdp.connector.v1alpha.ConnectorPrivateService/LookUpConnectorAdmin CSV ${resCSVDst.message.connector.uid} response connector owner is UUID`]: (r) => helper.isValidOwner(r.message.connector.user),
        });

        check(clientPublic.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${csvDstConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        clientPublic.close();
    });
}
