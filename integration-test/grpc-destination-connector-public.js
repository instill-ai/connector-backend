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

const client = new grpc.Client();
client.load(['proto/vdp/connector/v1alpha'], 'connector_public_service.proto');

export function CheckCreate() {

    group("Connector API: Create destination connectors", () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        // destination-http
        var httpDstConnector = {
            "id": "destination-http",
            "destination_connector_definition": constant.httpDstDefRscName,
            "connector": {
                "description": "HTTP source",
                "configuration": {},
            }
        }

        var resDstHTTP = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: httpDstConnector
        })

        check(resDstHTTP, {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector HTTP response StatusOK": (r) => r.status === grpc.StatusOK,
            "vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector HTTP response destinationConnector name": (r) => r.message.destinationConnector.name == `destination-connectors/${httpDstConnector.id}`,
            "vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector HTTP response destinationConnector uid": (r) => helper.isUUID(r.message.destinationConnector.uid),
            "vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector HTTP response destinationConnector destinationConnectorDefinition": (r) => r.message.destinationConnector.destinationConnectorDefinition === constant.httpDstDefRscName
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: httpDstConnector
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector response StatusAlreadyExists": (r) => r.status === grpc.StatusAlreadyExists,
        });

        // destination-grpc
        var gRPCDstConnector = {
            "id": "destination-grpc",
            "destination_connector_definition": constant.gRPCDstDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        var resDstGRPC = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: gRPCDstConnector
        })

        check(resDstGRPC, {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector gRPC response StatusOK": (r) => r.status === grpc.StatusOK,
        });


        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {}), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector response StatusInvalidArgument": (r) => r.status === grpc.StatusInvalidArgument,
        });

        // destination-csv
        var csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": constant.csvDstConfig
            }
        }

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: csvDstConnector
        })

        // Check connector state being updated in 120 secs
        let currentTime = new Date().getTime();
        let timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            var res = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnector', {
                name: `destination_connector/${resCSVDst.message.destinationConnector.id}`
            })
            if (res.message.destinationConnector.connector.state === "STATE_CONNECTED") {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        check(resCSVDst, {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector CSV response StatusOK": (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnector', {
            name: `destination_connector/${resCSVDst.message.destinationConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnector CSV ${resCSVDst.message.destinationConnector.id} response STATE_CONNECTED`]: (r) => r.message.destinationConnector.connector.state === "STATE_CONNECTED",
        });

        // destination-mysql (will end up with STATE_ERROR)
        var mySQLDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.mySQLDstDefRscName,
            "connector": {
                "configuration": {
                    "host": randomString(10),
                    "port": 3306,
                    "username": randomString(10),
                    "database": randomString(10),
                }
            }
        }

        var resDstMySQL = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: mySQLDstConnector
        })

        check(resDstMySQL, {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector MySQL response StatusOK": (r) => r.status === grpc.StatusOK,
            "vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector MySQL response destinationConnector name": (r) => r.message.destinationConnector.name == `destination-connectors/${mySQLDstConnector.id}`,
            "vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector MySQL response destinationConnector uid": (r) => helper.isUUID(r.message.destinationConnector.uid),
            "vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector MySQL response destinationConnector destinationConnectorDefinition": (r) => r.message.destinationConnector.destinationConnectorDefinition === constant.mySQLDstDefRscName
        });

        // Check connector state being updated in 180 secs
        currentTime = new Date().getTime();
        timeoutTime = new Date().getTime() + 180000;
        var pass = false
        while (timeoutTime > currentTime) {
            var res = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnector', {
                name: `destination_connector/${resDstMySQL.message.destinationConnector.id}`
            })
            if (res.message.destinationConnector.connector.state === "STATE_ERROR") {
                pass = true
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        check(null, {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector MySQL destination connector ended up STATE_ERROR": (r) => pass
        })

        // check JSON Schema failure cases
        var jsonSchemaFailedBodyCSV = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": {} // required destination_path
            }
        }

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: jsonSchemaFailedBodyCSV
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector response status for JSON Schema failed body 400 (destination-csv missing destination_path)": (r) => r.status === grpc.StatusInvalidArgument,
        });

        var jsonSchemaFailedBodyMySQL = {
            "id": randomString(10),
            "destination_connector_definition": constant.mySQLDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": {
                    "host": randomString(10),
                    "port": "3306",
                    "username": randomString(10),
                    "database": randomString(10),
                } // required port integer type
            }
        }

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: jsonSchemaFailedBodyMySQL
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector response status for JSON Schema failed body 400 (destination-mysql port not integer)": (r) => r.status === grpc.StatusInvalidArgument,
        });

        // Delete test records
        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/${resDstHTTP.message.destinationConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${resDstHTTP.message.destinationConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/${resDstGRPC.message.destinationConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${resDstGRPC.message.destinationConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/${resCSVDst.message.destinationConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${resCSVDst.message.destinationConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/${resDstMySQL.message.destinationConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${resDstMySQL.message.destinationConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });

}

export function CheckList() {

    group("Connector API: List destination connectors", () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors', {}, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors response destinationConnectors array is 0 length`]: (r) => r.message.destinationConnectors.length === 0,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors response nextPageToken is empty`]: (r) => r.message.nextPageToken === "",
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors response totalSize is 0`]: (r) => r.message.totalSize == 0,
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
            var resDstHTTP = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
                destination_connector: reqBody
            })

            check(resDstHTTP, {
                [`vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector x${reqBodies.length} HTTP response StatusOK`]: (r) => r.status === grpc.StatusOK,
            });
        }

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors', {}, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors response has destinationConnectors array`]: (r) => Array.isArray(r.message.destinationConnectors),
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors response has totalSize = ${reqBodies.length}`]: (r) => r.message.totalSize == reqBodies.length,
        });

        var limitedRecords = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors', {}, {})
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors', {
            pageSize: 0
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors pageSize=0 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors pageSize=0 response all records`]: (r) => r.message.destinationConnectors.length === limitedRecords.message.destinationConnectors.length,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors', {
            pageSize: 1
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors pageSize=1 response size 1`]: (r) => r.message.destinationConnectors.length === 1,
        });

        var pageRes = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors', {
            pageSize: 1
        }, {})

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors', {
            pageSize: 1,
            pageToken: `${pageRes.message.nextPageToken}`
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors pageSize=1 pageToken=${pageRes.message.nextPageToken} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors pageSize=1 pageToken=${pageRes.message.nextPageToken} response size 1`]: (r) => r.message.destinationConnectors.length === 1,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors', {
            pageSize: 1,
            view: "VIEW_BASIC"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors pageSize=1 view=VIEW_BASIC response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors pageSize=1 view=VIEW_BASIC response destinationConnectors[0].connector.configuration is null`]: (r) => r.message.destinationConnectors[0].connector.configuration === null,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors', {
            pageSize: 1,
            view: "VIEW_FULL"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors pageSize=1 view=VIEW_FULL response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors pageSize=1 view=VIEW_FULL response destinationConnectors[0].connector.configuration is null`]: (r) => r.message.destinationConnectors[0].connector.configuration !== null,
        });


        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors', {
            pageSize: 1,
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors pageSize=1 response destinationConnectors[0].connector.configuration is null`]: (r) => r.message.destinationConnectors[0].connector.configuration === null,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors', {
            pageSize: `${limitedRecords.message.totalSize}`,
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors pageSize=${limitedRecords.message.totalSize} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors pageSize=${limitedRecords.message.totalSize} response nextPageToken is empty`]: (r) => r.message.nextPageToken === "",
        });

        // Delete the destination connectors
        for (const reqBody of reqBodies) {
            check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
                name: `destination-connectors/${reqBody.id}`
            }), {
                [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector x${reqBodies.length} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            });
        }

        client.close();
    });
}

export function CheckGet() {

    group("Connector API: Get destination connectors by ID", () => {

        client.connect(constant.connectorGRPCPublicHost, {
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

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: csvDstConnector
        })

        // Check connector state being updated in 120 secs
        var currentTime = new Date().getTime();
        var timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            var res = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnector', {
                name: `destination_connector/${resCSVDst.message.destinationConnector.id}`
            })
            if (res.message.destinationConnector.connector.state === "STATE_CONNECTED") {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnector', {
            name: `destination_connector/${resCSVDst.message.destinationConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnector CSV ${resCSVDst.message.destinationConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnector CSV ${resCSVDst.message.destinationConnector.id} response connector id`]: (r) => r.message.destinationConnector.id === csvDstConnector.id,
            [`vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnector CSV ${resCSVDst.message.destinationConnector.id} response connector destinationConnectorDefinition permalink`]: (r) => r.message.destinationConnector.destinationConnectorDefinition === constant.csvDstDefRscName,
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/${resCSVDst.message.destinationConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${resCSVDst.message.destinationConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckUpdate() {

    group("Connector API: Update destination connectors", () => {

        client.connect(constant.connectorGRPCPublicHost, {
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

        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: csvDstConnector
        })

        var csvDstConnectorUpdate = {
            "id": csvDstConnector.id,
            "name": `destination-connectors/${csvDstConnector.id}`,
            "destination_connector_definition": csvDstConnector.destination_connector_definition,
            "connector": {
                "tombstone": true,
                "description": randomString(50),
                "configuration": {
                    destination_path: "/tmp"
                }
            }
        }

        var resCSVDstUpdate = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/UpdateDestinationConnector', {
            destination_connector: csvDstConnectorUpdate,
            update_mask: "connector.description,connector.configuration",
        })

        check(resCSVDstUpdate, {
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateDestinationConnector ${resCSVDstUpdate.message.destinationConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateDestinationConnector ${resCSVDstUpdate.message.destinationConnector.id} response connector connectorDefinition`]: (r) => r.message.destinationConnector.destinationConnectorDefinition === constant.csvDstDefRscName,
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateDestinationConnector ${resCSVDstUpdate.message.destinationConnector.id} response connector description`]: (r) => r.message.destinationConnector.connector.description === csvDstConnectorUpdate.connector.description,
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateDestinationConnector ${resCSVDstUpdate.message.destinationConnector.id} response connector tombstone`]: (r) => r.message.destinationConnector.connector.tombstone === false,
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateDestinationConnector ${resCSVDstUpdate.message.destinationConnector.id} response connector configuration`]: (r) => r.message.destinationConnector.connector.configuration.destination_path === csvDstConnectorUpdate.connector.configuration.destination_path,
        });

        // Try to update with empty description
        csvDstConnectorUpdate = {
            "name": `destination-connectors/${csvDstConnector.id}`,
            "connector": {
                "description": "",
            }
        }

        resCSVDstUpdate = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/UpdateDestinationConnector', {
            destination_connector: csvDstConnectorUpdate,
            update_mask: "connector.description",
        })

        check(resCSVDstUpdate, {
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateDestinationConnector ${resCSVDstUpdate.message.destinationConnector.id} with empty description response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateDestinationConnector ${resCSVDstUpdate.message.destinationConnector.id} with empty description response connector description`]: (r) => r.message.destinationConnector.connector.description === csvDstConnectorUpdate.connector.description,
        });

        // Try to update with a non-existing name field (which should be ignored because name field is OUTPUT_ONLY)
        csvDstConnectorUpdate = {
            "name": `destination-connectors/${randomString(5)}`,
            "connector": {
                "description": randomString(50),
            }
        }

        resCSVDstUpdate = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/UpdateDestinationConnector', {
            destination_connector: csvDstConnectorUpdate,
            update_mask: "connector.description",
        })
        check(resCSVDstUpdate, {
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateDestinationConnector with non-existing name field response StatusNotFound`]: (r) => r.status === grpc.StatusNotFound,
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/${csvDstConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckLookUp() {

    group("Connector API: Look up destination connectors by UID", () => {

        client.connect(constant.connectorGRPCPublicHost, {
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

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: csvDstConnector
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/LookUpDestinationConnector', {
            permalink: `destination_connector/${resCSVDst.message.destinationConnector.uid}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/LookUpDestinationConnector CSV ${resCSVDst.message.destinationConnector.uid} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/LookUpDestinationConnector CSV ${resCSVDst.message.destinationConnector.uid} response connector id`]: (r) => r.message.destinationConnector.uid === resCSVDst.message.destinationConnector.uid,
            [`vdp.connector.v1alpha.ConnectorPublicService/LookUpDestinationConnector CSV ${resCSVDst.message.destinationConnector.uid} response connector destinationConnectorDefinition permalink`]: (r) => r.message.destinationConnector.destinationConnectorDefinition === constant.csvDstDefRscName,
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/${csvDstConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckState() {

    group("Connector API: Change state destination connectors", () => {

        client.connect(constant.connectorGRPCPublicHost, {
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

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: csvDstConnector
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectDestinationConnector', {
            name: `destination_connector/${resCSVDst.message.destinationConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ConnectDestinationConnector ${resCSVDst.message.destinationConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/DisconnectDestinationConnector', {
            name: `destination_connector/${resCSVDst.message.destinationConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DisconnectDestinationConnector ${resCSVDst.message.destinationConnector.id} response at UNSPECIFIED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectDestinationConnector', {
            name: `destination_connector/${resCSVDst.message.destinationConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ConnectDestinationConnector ${resCSVDst.message.destinationConnector.id} response at UNSPECIFIED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Check connector state being updated in 120 secs
        let currentTime = new Date().getTime();
        let timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            var res = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnector', {
                name: `destination_connector/${resCSVDst.message.destinationConnector.id}`
            })
            if (res.message.destinationConnector.connector.state === "STATE_CONNECTED") {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectDestinationConnector', {
            name: `destination_connector/${resCSVDst.message.destinationConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ConnectDestinationConnector ${resCSVDst.message.destinationConnector.id} response at STATE_CONNECTED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/DisconnectDestinationConnector', {
            name: `destination_connector/${resCSVDst.message.destinationConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DisconnectDestinationConnector ${resCSVDst.message.destinationConnector.id} response at STATE_CONNECTED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/DisconnectDestinationConnector', {
            name: `destination_connector/${resCSVDst.message.destinationConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DisconnectDestinationConnector ${resCSVDst.message.destinationConnector.id} response at STATE_DISCONNECTED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectDestinationConnector', {
            name: `destination_connector/${resCSVDst.message.destinationConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ConnectDestinationConnector ${resCSVDst.message.destinationConnector.id} response at STATE_DISCONNECTED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Check connector state being updated in 120 secs
        currentTime = new Date().getTime();
        timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            var res = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnector', {
                name: `destination_connector/${resCSVDst.message.destinationConnector.id}`
            })
            if (res.message.destinationConnector.connector.state === "STATE_CONNECTED") {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/${csvDstConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckRename() {

    group("Connector API: Rename destination connectors", () => {

        client.connect(constant.connectorGRPCPublicHost, {
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

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: csvDstConnector
        })

        let new_id = `some-id-not-${resCSVDst.message.destinationConnector.id}`
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/RenameDestinationConnector', {
            name: resCSVDst.message.destinationConnector.id,
            new_destination_connector_id: new_id
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/RenameDestinationConnector ${resCSVDst.message.destinationConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/RenameDestinationConnector ${resCSVDst.message.destinationConnector.id} response id is some-id-not-${resCSVDst.message.destinationConnector.id}`]: (r) => r.message.destinationConnector.id === `some-id-not-${resCSVDst.message.destinationConnector.id}`,
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/${new_id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${new_id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckWrite() {

    group("Connector API: Write destination connectors", () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var csvDstConnector, resCSVDst, currentTime, timeoutTime

        // Write classification output
        csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": {
                    "destination_path": "/local/test-classification"
                },
            }
        }

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: csvDstConnector
        })

        // Check connector state being updated in 120 secs
        currentTime = new Date().getTime();
        timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            var res = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnector', {
                name: `destination_connector/${resCSVDst.message.destinationConnector.id}`
            })
            if (res.message.destinationConnector.connector.state === "STATE_CONNECTED") {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WriteDestinationConnector', {
            "name": `destination_connector/${resCSVDst.message.destinationConnector.id}`,
            "sync_mode": "SUPPORTED_SYNC_MODES_FULL_REFRESH",
            "destination_sync_mode": "SUPPORTED_DESTINATION_SYNC_MODES_OVERWRITE",
            "pipeline": "pipelines/dummy-pipeline",
            "recipe": {
                "source": "source-connectors/dummy-source",
                "model_instances": [
                    "models/dummy-model/instances/v1.0-cpu"
                ],
                "destination": "destination-connectors/dummy-destination",
            },
            "data_mapping_indices": ["01GB5T5ZK9W9C2VXMWWRYM8WPA"],
            "model_instance_outputs": constant.clsModelInstOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/WriteDestinationConnector ${resCSVDst.message.destinationConnector.id} response (classification) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/${resCSVDst.message.destinationConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${resCSVDst.message.destinationConnector.id} response (classification) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Write detection output (empty bounding_boxes)
        csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": {
                    "destination_path": "/local/test-detection-empty-bounding-boxes"
                },
            }
        }

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: csvDstConnector
        })

        // Check connector state being updated in 120 secs
        currentTime = new Date().getTime();
        timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            var res = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnector', {
                name: `destination_connector/${resCSVDst.message.destinationConnector.id}`
            })
            if (res.message.destinationConnector.connector.state === "STATE_CONNECTED") {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WriteDestinationConnector', {
            "name": `destination_connector/${resCSVDst.message.destinationConnector.id}`,
            "sync_mode": "SUPPORTED_SYNC_MODES_FULL_REFRESH",
            "destination_sync_mode": "SUPPORTED_DESTINATION_SYNC_MODES_OVERWRITE",
            "pipeline": "pipelines/dummy-pipeline",
            "recipe": {
                "source": "source-connectors/dummy-source",
                "model_instances": [
                    "models/dummy-model/instances/v1.0-cpu",
                    "models/dummy-model/instances/v2.0-cpu"
                ],
                "destination": "destination-connectors/dummy-destination",
            },
            "data_mapping_indices": ["01GB5T5ZK9W9C2VXMWWRYM8WPM"],
            "model_instance_outputs": constant.detectionEmptyModelInstOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/WriteDestinationConnector ${resCSVDst.message.destinationConnector.id} response (detection) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/${resCSVDst.message.destinationConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${resCSVDst.message.destinationConnector.id} response (detection) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Write detection output (multiple models)
        csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": {
                    "destination_path": "/local/test-detection-multi-models"
                },
            }
        }

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: csvDstConnector
        })

        // Check connector state being updated in 120 secs
        currentTime = new Date().getTime();
        timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            var res = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnector', {
                name: `destination_connector/${resCSVDst.message.destinationConnector.id}`
            })
            if (res.message.destinationConnector.connector.state === "STATE_CONNECTED") {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WriteDestinationConnector', {
            "name": `destination_connector/${resCSVDst.message.destinationConnector.id}`,
            "sync_mode": "SUPPORTED_SYNC_MODES_FULL_REFRESH",
            "destination_sync_mode": "SUPPORTED_DESTINATION_SYNC_MODES_OVERWRITE",
            "pipeline": "pipelines/dummy-pipeline",
            "recipe": {
                "source": "source-connectors/dummy-source",
                "model_instances": [
                    "models/dummy-model/instances/v1.0-cpu",
                    "models/dummy-model/instances/v2.0-cpu"
                ],
                "destination": "destination-connectors/dummy-destination",
            },
            "data_mapping_indices": ["01GB5T5ZK9W9C2VXMWWRYM8WPM", "01GB5T5ZK9W9C2VXMWWRYM8WPN", "01GB5T5ZK9W9C2VXMWWRYM8WPO"],
            "model_instance_outputs": constant.detectionModelInstOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/WriteDestinationConnector ${resCSVDst.message.destinationConnector.id} response (detection) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/${resCSVDst.message.destinationConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${resCSVDst.message.destinationConnector.id} response (detection) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Write keypoint output
        csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": {
                    "destination_path": "/local/test-keypoint"
                },
            }
        }


        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: csvDstConnector
        })

        // Check connector state being updated in 120 secs
        currentTime = new Date().getTime();
        timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            var res = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnector', {
                name: `destination_connector/${resCSVDst.message.destinationConnector.id}`
            })
            if (res.message.destinationConnector.connector.state === "STATE_CONNECTED") {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WriteDestinationConnector', {
            "name": `destination_connector/${resCSVDst.message.destinationConnector.id}`,
            "sync_mode": "SUPPORTED_SYNC_MODES_FULL_REFRESH",
            "destination_sync_mode": "SUPPORTED_DESTINATION_SYNC_MODES_OVERWRITE",
            "pipeline": "pipelines/dummy-pipeline",
            "recipe": {
                "source": "source-connectors/dummy-source",
                "model_instances": [
                    "models/dummy-model/instances/v1.0-cpu"
                ],
                "destination": "destination-connectors/dummy-destination",
            },
            "data_mapping_indices": ["01GB5T5ZK9W9C2VXMWWRYM8WPA"],
            "model_instance_outputs": constant.keypointModelInstOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/WriteDestinationConnector ${resCSVDst.message.destinationConnector.id} response (keypoint) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/${resCSVDst.message.destinationConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${resCSVDst.message.destinationConnector.id} response (keypoint) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Write ocr output
        csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": {
                    "destination_path": "/local/test-ocr"
                },
            }
        }


        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: csvDstConnector
        })

        // Check connector state being updated in 120 secs
        currentTime = new Date().getTime();
        timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            var res = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnector', {
                name: `destination_connector/${resCSVDst.message.destinationConnector.id}`
            })
            if (res.message.destinationConnector.connector.state === "STATE_CONNECTED") {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WriteDestinationConnector', {
            "name": `destination_connector/${resCSVDst.message.destinationConnector.id}`,
            "sync_mode": "SUPPORTED_SYNC_MODES_FULL_REFRESH",
            "destination_sync_mode": "SUPPORTED_DESTINATION_SYNC_MODES_OVERWRITE",
            "pipeline": "pipelines/dummy-pipeline",
            "recipe": {
                "source": "source-connectors/dummy-source",
                "model_instances": [
                    "models/dummy-model/instances/v1.0-cpu"
                ],
                "destination": "destination-connectors/dummy-destination",
            },
            "data_mapping_indices": ["01GB5T5ZK9W9C2VXMWWRYM8WPA"],
            "model_instance_outputs": constant.ocrModelInstOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/WriteDestinationConnector ${resCSVDst.message.destinationConnector.id} response (ocr) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });
        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/${resCSVDst.message.destinationConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${resCSVDst.message.destinationConnector.id} response (ocr) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Write semantic segmentation output
        csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": {
                    "destination_path": "/local/test-semantic-segmentation"
                },
            }
        }

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: csvDstConnector
        })

        // Check connector state being updated in 120 secs
        currentTime = new Date().getTime();
        timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            var res = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnector', {
                name: `destination_connector/${resCSVDst.message.destinationConnector.id}`
            })
            if (res.message.destinationConnector.connector.state === "STATE_CONNECTED") {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WriteDestinationConnector', {
            "name": `destination_connector/${resCSVDst.message.destinationConnector.id}`,
            "sync_mode": "SUPPORTED_SYNC_MODES_FULL_REFRESH",
            "destination_sync_mode": "SUPPORTED_DESTINATION_SYNC_MODES_OVERWRITE",
            "pipeline": "pipelines/dummy-pipeline",
            "recipe": {
                "source": "source-connectors/dummy-source",
                "model_instances": [
                    "models/dummy-model/instances/v1.0-cpu"
                ],
                "destination": "destination-connectors/dummy-destination",
            },
            "data_mapping_indices": ["01GB5T5ZK9W9C2VXMWWRYM8WPA"],
            "model_instance_outputs": constant.semanticSegModelInstOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/WriteDestinationConnector ${resCSVDst.message.destinationConnector.id} response (semantic-segmentation) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/${resCSVDst.message.destinationConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${resCSVDst.message.destinationConnector.id} response (semantic-segmentation) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Write instance segmentation output
        csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": {
                    "destination_path": "/local/test-instance-segmentation"
                },
            }
        }

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: csvDstConnector
        })

        // Check connector state being updated in 120 secs
        currentTime = new Date().getTime();
        timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            var res = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnector', {
                name: `destination_connector/${resCSVDst.message.destinationConnector.id}`
            })
            if (res.message.destinationConnector.connector.state === "STATE_CONNECTED") {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WriteDestinationConnector', {
            "name": `destination_connector/${resCSVDst.message.destinationConnector.id}`,
            "sync_mode": "SUPPORTED_SYNC_MODES_FULL_REFRESH",
            "destination_sync_mode": "SUPPORTED_DESTINATION_SYNC_MODES_OVERWRITE",
            "pipeline": "pipelines/dummy-pipeline",
            "recipe": {
                "source": "source-connectors/dummy-source",
                "model_instances": [
                    "models/dummy-model/instances/v1.0-cpu"
                ],
                "destination": "destination-connectors/dummy-destination",
            },
            "data_mapping_indices": ["01GB5T5ZK9W9C2VXMWWRYM8WPA"],
            "model_instance_outputs": constant.instSegModelInstOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/WriteDestinationConnector ${resCSVDst.message.destinationConnector.id} response (instance-segmentation) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/${resCSVDst.message.destinationConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${resCSVDst.message.destinationConnector.id} response (instance-segmentation) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Write text-to-image output
        csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": {
                    "destination_path": "/local/test-text-to-image"
                },
            }
        }

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: csvDstConnector
        })

        // Check connector state being updated in 120 secs
        currentTime = new Date().getTime();
        timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            var res = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnector', {
                name: `destination_connector/${resCSVDst.message.destinationConnector.id}`
            })
            if (res.message.destinationConnector.connector.state === "STATE_CONNECTED") {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WriteDestinationConnector', {
            "name": `destination_connector/${resCSVDst.message.destinationConnector.id}`,
            "sync_mode": "SUPPORTED_SYNC_MODES_FULL_REFRESH",
            "destination_sync_mode": "SUPPORTED_DESTINATION_SYNC_MODES_OVERWRITE",
            "pipeline": "pipelines/dummy-pipeline",
            "recipe": {
                "source": "source-connectors/dummy-source",
                "model_instances": [
                    "models/dummy-model/instances/v1.0-cpu"
                ],
                "destination": "destination-connectors/dummy-destination",
            },
            "data_mapping_indices": ["01GB5T5ZK9W9C2VXMWWRYM8WPA"],
            "model_instance_outputs": constant.textToImageModelInstOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/WriteDestinationConnector ${resCSVDst.message.destinationConnector.id} response (text-to-image) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/${resCSVDst.message.destinationConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${resCSVDst.message.destinationConnector.id} response (text-to-image) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Write text-generation output
        csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": {
                    "destination_path": "/local/test-text-generation"
                },
            }
        }

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: csvDstConnector
        })

        // Check connector state being updated in 120 secs
        currentTime = new Date().getTime();
        timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            var res = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnector', {
                name: `destination_connector/${resCSVDst.message.destinationConnector.id}`
            })
            if (res.message.destinationConnector.connector.state === "STATE_CONNECTED") {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WriteDestinationConnector', {
            "name": `destination_connector/${resCSVDst.message.destinationConnector.id}`,
            "sync_mode": "SUPPORTED_SYNC_MODES_FULL_REFRESH",
            "destination_sync_mode": "SUPPORTED_DESTINATION_SYNC_MODES_OVERWRITE",
            "pipeline": "pipelines/dummy-pipeline",
            "recipe": {
                "source": "source-connectors/dummy-source",
                "model_instances": [
                    "models/dummy-model/instances/v1.0-cpu"
                ],
                "destination": "destination-connectors/dummy-destination",
            },
            "data_mapping_indices": ["01GB5T5ZK9W9C2VXMWWRYM8WPA"],
            "model_instance_outputs": constant.textGenerationModelInstOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/WriteDestinationConnector ${resCSVDst.message.destinationConnector.id} response (text-generation) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/${resCSVDst.message.destinationConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${resCSVDst.message.destinationConnector.id} response (text-generation) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Write unspecified output
        csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": {
                    "destination_path": "/local/test-unspecified"
                },
            }
        }

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: csvDstConnector
        })

        // Check connector state being updated in 120 secs
        currentTime = new Date().getTime();
        timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            var res = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnector', {
                name: `destination_connector/${resCSVDst.message.destinationConnector.id}`
            })
            if (res.message.destinationConnector.connector.state === "STATE_CONNECTED") {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WriteDestinationConnector', {
            "name": `destination_connector/${resCSVDst.message.destinationConnector.id}`,
            "sync_mode": "SUPPORTED_SYNC_MODES_FULL_REFRESH",
            "destination_sync_mode": "SUPPORTED_DESTINATION_SYNC_MODES_OVERWRITE",
            "pipeline": "pipelines/dummy-pipeline",
            "recipe": {
                "source": "source-connectors/dummy-source",
                "model_instances": [
                    "models/dummy-model/instances/v1.0-cpu"
                ],
                "destination": "destination-connectors/dummy-destination",
            },
            "data_mapping_indices": ["01GB5T5ZK9W9C2VXMWWRYM8WPA"],
            "model_instance_outputs": constant.unspecifiedModelInstOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/WriteDestinationConnector ${resCSVDst.message.destinationConnector.id} response (unspecified) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/${resCSVDst.message.destinationConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${resCSVDst.message.destinationConnector.id} response (unspecified) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}
