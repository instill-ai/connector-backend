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

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin', {}, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin response sourceConnectors array is 0 length`]: (r) => r.message.sourceConnectors.length === 0,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin response nextPageToken is empty`]: (r) => r.message.nextPageToken === "",
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin response totalSize is 0`]: (r) => r.message.totalSize == 0,
        });

        var reqBodies = [];
        reqBodies[0] = {
            "id": "source-http",
            "source_connector_definition": constant.httpSrcDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        reqBodies[1] = {
            "id": "source-grpc",
            "source_connector_definition": constant.gRPCSrcDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        // Create connectors
        for (const reqBody of reqBodies) {
            check(clientPublic.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector', {
                source_connector: reqBody
            }), {
                [`vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector x${reqBodies.length} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            });
        }

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin', {}, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin response has sourceConnectors array`]: (r) => Array.isArray(r.message.sourceConnectors),
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin response has totalSize = ${reqBodies.length}`]: (r) => r.message.totalSize == reqBodies.length,
        });

        var limitedRecords = clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin', {}, {})
        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin', {
            pageSize: 0
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin pageSize=0 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin pageSize=0 response has sourceConnectors length`]: (r) => r.message.sourceConnectors.length === limitedRecords.message.sourceConnectors.length,
        });

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin', {
            pageSize: 1
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin pageSize=1 response has sourceConnectors length`]: (r) => r.message.sourceConnectors.length === 1,
        });

        var pageRes = clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin', {
            pageSize: 1
        }, {})
        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin', {
            pageSize: 1,
            pageToken: pageRes.message.nextPageToken
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin pageSize=1 pageToken=${pageRes.message.nextPageToken} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin pageSize=1 pageToken=${pageRes.message.nextPageToken} response has sourceConnectors length`]: (r) => r.message.sourceConnectors.length === 1,
        });

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin', {
            pageSize: 1,
            view: "VIEW_BASIC"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin pageSize=1 view=VIEW_BASIC response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin pageSize=1 view=VIEW_BASIC response has sourceConnectors[0].connector.configuration is null`]: (r) => r.message.sourceConnectors[0].connector.configuration === null,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin pageSize=1 view=VIEW_BASIC response has sourceConnectors[0].connector.owner is UUID`]: (r) => helper.isValidOwner(r.message.sourceConnectors[0].connector.user),
        });
        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin', {
            pageSize: 1,
            view: "VIEW_FULL"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin pageSize=1 view=VIEW_FULL response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin pageSize=1 view=VIEW_FULL response has sourceConnectors[0].connector.configuration is not null`]: (r) => r.message.sourceConnectors[0].connector.configuration !== null,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin pageSize=1 view=VIEW_FULL response has sourceConnectors[0].connector.configuration is {}`]: (r) => Object.keys(r.message.sourceConnectors[0].connector.configuration).length === 0,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin pageSize=1 view=VIEW_FULL response has sourceConnectors[0].connector.owner is UUID`]: (r) => helper.isValidOwner(r.message.sourceConnectors[0].connector.user),
        });

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin', {
            pageSize: 1,
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin pageSize=1 response has sourceConnectors[0].connector.configuration is null`]: (r) => r.message.sourceConnectors[0].connector.configuration === null,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin pageSize=1 response has sourceConnectors[0].connector.owner is UUID`]: (r) => helper.isValidOwner(r.message.sourceConnectors[0].connector.user),
        });

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin', {
            pageSize: `${limitedRecords.message.totalSize}`,
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin pageSize=${limitedRecords.message.totalSize} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListSourceConnectorsAdmin pageSize=${limitedRecords.message.totalSize} response nextPageToken is empty`]: (r) => r.message.nextPageToken === "",
        });

        // Delete the destination connectors
        for (const reqBody of reqBodies) {
            check(clientPublic.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector`, {
                name: `source-connectors/${reqBody.id}`
            }), {
                [`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector x${reqBodies.length} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            });
        }
    });

    clientPublic.close();
}

export function CheckGet() {

    group("Connector API: Get source connectors by ID by admin", () => {
        clientPrivate.connect(constant.connectorGRPCPrivateHost, {
            plaintext: true
        });

        clientPublic.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var httpSrcConnector = {
            "id": "source-http",
            "source_connector_definition": constant.httpSrcDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        var resHTTP = clientPublic.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector', {
            source_connector: httpSrcConnector
        })

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/GetSourceConnectorAdmin', {
            name: `source-connectors/${resHTTP.message.sourceConnector.id}`
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/GetSourceConnectorAdmin name=source-connectors/${resHTTP.message.sourceConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/GetSourceConnectorAdmin name=source-connectors/${resHTTP.message.sourceConnector.id} response connector id`]: (r) => r.message.sourceConnector.id === httpSrcConnector.id,
            [`vdp.connector.v1alpha.ConnectorPrivateService/GetSourceConnectorAdmin name=source-connectors/${resHTTP.message.sourceConnector.sourceConnectorDefinition} response connector id`]: (r) => r.message.sourceConnector.sourceConnectorDefinition === constant.httpSrcDefRscName,
            [`vdp.connector.v1alpha.ConnectorPrivateService/GetSourceConnectorAdmin name=source-connectors/${resHTTP.message.sourceConnector.sourceConnectorDefinition} response connector owner is UUID`]: (r) => helper.isValidOwner(r.message.sourceConnector.connector.user),
        });

        check(clientPublic.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector`, {
            name: `source-connectors/${resHTTP.message.sourceConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector ${resHTTP.message.sourceConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        clientPublic.close();
    });
}

export function CheckLookUp() {

    group("Connector API: Look up source connectors by UID by admin", () => {

        clientPrivate.connect(constant.connectorGRPCPrivateHost, {
            plaintext: true
        });

        clientPublic.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var httpSrcConnector = {
            "id": "source-http",
            "source_connector_definition": constant.httpSrcDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        var resHTTP = clientPublic.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector', {
            source_connector: httpSrcConnector
        })

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/LookUpSourceConnectorAdmin', {
            permalink: `source-connectors/${resHTTP.message.sourceConnector.uid}`
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/LookUpSourceConnectorAdmin permalink=source-connectors/${resHTTP.message.sourceConnector.uid} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/LookUpSourceConnectorAdmin permalink=source-connectors/${resHTTP.message.sourceConnector.uid} response connector uid`]: (r) => r.message.sourceConnector.id === httpSrcConnector.id,
            [`vdp.connector.v1alpha.ConnectorPrivateService/LookUpSourceConnectorAdmin permalink=source-connectors/${resHTTP.message.sourceConnector.uid} response connector id`]: (r) => r.message.sourceConnector.sourceConnectorDefinition === constant.httpSrcDefRscName,
            [`vdp.connector.v1alpha.ConnectorPrivateService/LookUpSourceConnectorAdmin permalink=source-connectors/${resHTTP.message.sourceConnector.uid} response connector owner is UUID`]: (r) => helper.isValidOwner(r.message.sourceConnector.connector.user),
        });

        check(clientPublic.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector`, {
            name: `source-connectors/source-http`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector source-http response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        clientPublic.close();
    });
}
