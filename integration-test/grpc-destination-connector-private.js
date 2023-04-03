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

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin', {}, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin response destinationConnectors array is 0 length`]: (r) => r.message.destinationConnectors.length === 0,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin response nextPageToken is empty`]: (r) => r.message.nextPageToken === "",
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin response totalSize is 0`]: (r) => r.message.totalSize == 0,
        });

        const numConnectors = 10
        var reqBodies = [];
        for (var i = 0; i < numConnectors; i++) {
            reqBodies[i] = {
                "id": randomString(10),
                "destination_connector_definition": constant.csvDstDefRscName,
                "connector": {
                    "description": randomString(50),
                    "configuration": constant.csvDstConfig
                }
            }
        }

        // Create connectors
        for (const reqBody of reqBodies) {
            var resDstHTTP = clientPublic.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
                destination_connector: reqBody
            })

            check(resDstHTTP, {
                [`vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector x${reqBodies.length} HTTP response StatusOK`]: (r) => r.status === grpc.StatusOK,
            });
        }

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin', {}, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors response has destinationConnectors array`]: (r) => Array.isArray(r.message.destinationConnectors),
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin response has totalSize = ${reqBodies.length}`]: (r) => r.message.totalSize == reqBodies.length,
        });

        var limitedRecords = clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin', {}, {})
        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin', {
            pageSize: 0
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin pageSize=0 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin pageSize=0 response all records`]: (r) => r.message.destinationConnectors.length === limitedRecords.message.destinationConnectors.length,
        });

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin', {
            pageSize: 1
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin pageSize=1 response size 1`]: (r) => r.message.destinationConnectors.length === 1,
        });

        var pageRes = clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin', {
            pageSize: 1
        }, {})

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin', {
            pageSize: 1,
            pageToken: `${pageRes.message.nextPageToken}`
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin pageSize=1 pageToken=${pageRes.message.nextPageToken} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin pageSize=1 pageToken=${pageRes.message.nextPageToken} response size 1`]: (r) => r.message.destinationConnectors.length === 1,
        });

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin', {
            pageSize: 1,
            view: "VIEW_BASIC"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin pageSize=1 view=VIEW_BASIC response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin pageSize=1 view=VIEW_BASIC response destinationConnectors[0].connector.configuration is null`]: (r) => r.message.destinationConnectors[0].connector.configuration === null,
        });

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin', {
            pageSize: 1,
            view: "VIEW_FULL"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin pageSize=1 view=VIEW_FULL response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin pageSize=1 view=VIEW_FULL response destinationConnectors[0].connector.configuration is null`]: (r) => r.message.destinationConnectors[0].connector.configuration !== null,
        });


        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin', {
            pageSize: 1,
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin pageSize=1 response destinationConnectors[0].connector.configuration is null`]: (r) => r.message.destinationConnectors[0].connector.configuration === null,
        });

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin', {
            pageSize: `${limitedRecords.message.totalSize}`,
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin pageSize=${limitedRecords.message.totalSize} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/ListDestinationConnectorsAdmin pageSize=${limitedRecords.message.totalSize} response nextPageToken is empty`]: (r) => r.message.nextPageToken === "",
        });

        // Delete the destination connectors
        for (const reqBody of reqBodies) {
            check(clientPublic.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
                name: `destination-connectors/${reqBody.id}`
            }), {
                [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector x${reqBodies.length} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            });
        }

        clientPrivate.close();
        clientPublic.close();
    });
}

export function CheckGet() {

    group("Connector API: Get destination connectors by ID by admin", () => {

        clientPrivate.connect(constant.connectorGRPCPrivateHost, {
            plaintext: true
        });

        clientPublic.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": constant.csvDstConfig
            }
        }

        var resCSVDst = clientPublic.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: csvDstConnector
        })

        // Check connector state being updated in 120 secs
        var currentTime = new Date().getTime();
        var timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            var res = clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/GetDestinationConnectorAdmin', {
                name: `destination-connectors/${resCSVDst.message.destinationConnector.id}`
            })
            if (res.message.destinationConnector.connector.state === "STATE_CONNECTED") {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/GetDestinationConnectorAdmin', {
            name: `destination-connectors/${resCSVDst.message.destinationConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/GetDestinationConnectorAdmin CSV ${resCSVDst.message.destinationConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/GetDestinationConnectorAdmin CSV ${resCSVDst.message.destinationConnector.id} response connector id`]: (r) => r.message.destinationConnector.id === csvDstConnector.id,
            [`vdp.connector.v1alpha.ConnectorPrivateService/GetDestinationConnectorAdmin CSV ${resCSVDst.message.destinationConnector.id} response connector destinationConnectorDefinition permalink`]: (r) => r.message.destinationConnector.destinationConnectorDefinition === constant.csvDstDefRscName,
        });

        check(clientPublic.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/${resCSVDst.message.destinationConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${resCSVDst.message.destinationConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

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
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": constant.csvDstConfig
            }
        }

        var resCSVDst = clientPublic.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: csvDstConnector
        })

        check(clientPrivate.invoke('vdp.connector.v1alpha.ConnectorPrivateService/LookUpDestinationConnectorAdmin', {
            permalink: `destination_connector/${resCSVDst.message.destinationConnector.uid}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPrivateService/LookUpDestinationConnectorAdmin CSV ${resCSVDst.message.destinationConnector.uid} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPrivateService/LookUpDestinationConnectorAdmin CSV ${resCSVDst.message.destinationConnector.uid} response connector id`]: (r) => r.message.destinationConnector.uid === resCSVDst.message.destinationConnector.uid,
            [`vdp.connector.v1alpha.ConnectorPrivateService/LookUpDestinationConnectorAdmin CSV ${resCSVDst.message.destinationConnector.uid} response connector destinationConnectorDefinition permalink`]: (r) => r.message.destinationConnector.destinationConnectorDefinition === constant.csvDstDefRscName,
        });

        check(clientPublic.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/${csvDstConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        clientPublic.close();
    });
}
