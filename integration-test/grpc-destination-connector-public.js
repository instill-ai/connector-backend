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
            "connector_definition_name": constant.httpDstDefRscName,
            "description": "HTTP source",
            "configuration": {},
        }

        var resDstHTTP = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: httpDstConnector
        })

        check(resDstHTTP, {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector HTTP response StatusOK": (r) => r.status === grpc.StatusOK,
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector HTTP response destinationConnector name": (r) => r.message.connector.name == `connectors/${httpDstConnector.id}`,
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector HTTP response destinationConnector uid": (r) => helper.isUUID(r.message.connector.uid),
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector HTTP response destinationConnector connectorDefinition": (r) => r.message.connector.connectorDefinitionName === constant.httpDstDefRscName,
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector HTTP response destinationConnector owner is UUID": (r) => helper.isValidOwner(r.message.connector.user),
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: httpDstConnector
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector response StatusAlreadyExists": (r) => r.status === grpc.StatusAlreadyExists,
        });

        // destination-grpc
        var gRPCDstConnector = {
            "id": "destination-grpc",
            "connector_definition_name": constant.gRPCDstDefRscName,
            "configuration": {}
        }

        var resDstGRPC = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: gRPCDstConnector
        })

        check(resDstGRPC, {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector gRPC response StatusOK": (r) => r.status === grpc.StatusOK,
        });


        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {}), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector response StatusInvalidArgument": (r) => r.status === grpc.StatusInvalidArgument,
        });

        // destination-csv
        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: csvDstConnector
        })

        check(resCSVDst, {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector CSV response StatusOK": (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/GetConnector CSV ${resCSVDst.message.connector.id} response STATE_CONNECTED`]: (r) => r.message.connector.state === "STATE_CONNECTED",
        });

        // destination-mysql (will end up with STATE_ERROR)
        var mySQLDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.mySQLDstDefRscName,
            "configuration": {
                "host": randomString(10),
                "port": 3306,
                "username": randomString(10),
                "database": randomString(10),
            }
        }

        var resDstMySQL = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector',
            {
                connector: mySQLDstConnector,
            },
            {
                timeout: "600s",
            }
        )

        check(resDstMySQL, {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector MySQL response StatusOK": (r) => r.status === grpc.StatusOK,
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector MySQL response destinationConnector name": (r) => r.message.connector.name == `connectors/${mySQLDstConnector.id}`,
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector MySQL response destinationConnector uid": (r) => helper.isUUID(r.message.connector.uid),
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector MySQL response destinationConnector connectorDefinition": (r) => r.message.connector.connectorDefinitionName === constant.mySQLDstDefRscName,
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector MySQL response destinationConnector owner is UUID": (r) => helper.isValidOwner(r.message.connector.user),
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnector', {
            name: `connectors/${resDstMySQL.message.connector.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector MySQL destination connector ended up STATE_ERROR": (r) => r.message.state === "STATE_ERROR",
        })

        // check JSON Schema failure cases
        var jsonSchemaFailedBodyCSV = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": {} // required destination_path
        }

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: jsonSchemaFailedBodyCSV
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector response status for JSON Schema failed body 400 (destination-csv missing destination_path)": (r) => r.status === grpc.StatusInvalidArgument,
        });

        var jsonSchemaFailedBodyMySQL = {
            "id": randomString(10),
            "connector_definition_name": constant.mySQLDstDefRscName,
            "description": randomString(50),
            "configuration": {
                "host": randomString(10),
                "port": "3306",
                "username": randomString(10),
                "database": randomString(10),
            } // required port integer type
        }

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: jsonSchemaFailedBodyMySQL
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector response status for JSON Schema failed body 400 (destination-mysql port not integer)": (r) => r.status === grpc.StatusInvalidArgument,
        });

        // Delete test records
        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${resDstHTTP.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${resDstHTTP.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${resDstGRPC.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${resDstGRPC.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${resCSVDst.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${resDstMySQL.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${resDstMySQL.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });

}

export function CheckList() {

    group("Connector API: List destination connectors", () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectors', {
            filter: "connector_type=CONNECTOR_TYPE_DESTINATION",
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors response connectors array is 0 length`]: (r) => r.message.connectors.length === 0,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors response nextPageToken is empty`]: (r) => r.message.nextPageToken === "",
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors response totalSize is 0`]: (r) => r.message.totalSize == 0,
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

        // Create connectors
        for (const reqBody of reqBodies) {
            var resDstHTTP = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
                connector: reqBody
            })

            check(resDstHTTP, {
                [`vdp.connector.v1alpha.ConnectorPublicService/CreateConnector x${reqBodies.length} HTTP response StatusOK`]: (r) => r.status === grpc.StatusOK,
            });
        }

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectors', {
            filter: "connector_type=CONNECTOR_TYPE_DESTINATION",
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors response has connectors array`]: (r) => Array.isArray(r.message.connectors),
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors response has totalSize = ${reqBodies.length}`]: (r) => r.message.totalSize == reqBodies.length,
        });

        var limitedRecords = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectors', {
            filter: "connector_type=CONNECTOR_TYPE_DESTINATION",
        }, {})
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectors', {
            filter: "connector_type=CONNECTOR_TYPE_DESTINATION",
            pageSize: 0
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=0 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=0 response all records`]: (r) => r.message.connectors.length === limitedRecords.message.connectors.length,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectors', {
            filter: "connector_type=CONNECTOR_TYPE_DESTINATION",
            pageSize: 1
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 response size 1`]: (r) => r.message.connectors.length === 1,
        });

        var pageRes = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectors', {
            filter: "connector_type=CONNECTOR_TYPE_DESTINATION",
            pageSize: 1
        }, {})

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectors', {
            filter: "connector_type=CONNECTOR_TYPE_DESTINATION",
            pageSize: 1,
            pageToken: `${pageRes.message.nextPageToken}`
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 pageToken=${pageRes.message.nextPageToken} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 pageToken=${pageRes.message.nextPageToken} response size 1`]: (r) => r.message.connectors.length === 1,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectors', {
            filter: "connector_type=CONNECTOR_TYPE_DESTINATION",
            pageSize: 1,
            view: "VIEW_BASIC"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 view=VIEW_BASIC response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 view=VIEW_BASIC response connectors[0].configuration is null`]: (r) => r.message.connectors[0].configuration === null,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 view=VIEW_BASIC response connectors[0].owner is UUID`]: (r) => helper.isValidOwner(r.message.connectors[0].user ),
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectors', {
            filter: "connector_type=CONNECTOR_TYPE_DESTINATION",
            pageSize: 1,
            view: "VIEW_FULL"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 view=VIEW_FULL response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 view=VIEW_FULL response connectors[0].configuration is not null`]: (r) => r.message.connectors[0].configuration !== null,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 view=VIEW_FULL response connectors[0].connectorDefinitionDetail is not null`]: (r) => r.message.connectors[0].connectorDefinitionDetail !== null,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 view=VIEW_FULL response connectors[0].owner is UUID`]: (r) => helper.isValidOwner(r.message.connectors[0].user ),
        });


        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectors', {
            filter: "connector_type=CONNECTOR_TYPE_DESTINATION",
            pageSize: 1,
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 response connectors[0].configuration is null`]: (r) => r.message.connectors[0].configuration === null,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 response connectors[0].owner is UUID`]: (r) => helper.isValidOwner(r.message.connectors[0].user ),
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectors', {
            filter: "connector_type=CONNECTOR_TYPE_DESTINATION",
            pageSize: `${limitedRecords.message.totalSize}`,
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=${limitedRecords.message.totalSize} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=${limitedRecords.message.totalSize} response nextPageToken is empty`]: (r) => r.message.nextPageToken === "",
        });

        // Delete the destination connectors
        for (const reqBody of reqBodies) {
            check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
                name: `connectors/${reqBody.id}`
            }), {
                [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector x${reqBodies.length} response StatusOK`]: (r) => r.status === grpc.StatusOK,
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
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: csvDstConnector
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/GetConnector CSV ${resCSVDst.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/GetConnector CSV ${resCSVDst.message.connector.id} response connector id`]: (r) => r.message.connector.id === csvDstConnector.id,
            [`vdp.connector.v1alpha.ConnectorPublicService/GetConnector CSV ${resCSVDst.message.connector.id} response connector connectorDefinition permalink`]: (r) => r.message.connector.connectorDefinitionName === constant.csvDstDefRscName,
            [`vdp.connector.v1alpha.ConnectorPublicService/GetConnector CSV ${resCSVDst.message.connector.id} response connector owner is UUID`]: (r) => helper.isValidOwner(r.message.connector.user),
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${resCSVDst.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
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
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: csvDstConnector
        })

        var csvDstConnectorUpdate = {
            "id": csvDstConnector.id,
            "name": `connectors/${csvDstConnector.id}`,
            "connector_definition_name": csvDstConnector.connector_definition_name,
            "tombstone": true,
            "description": randomString(50),
            "configuration": {
                destination_path: "/tmp"
            }
        }

        var resCSVDstUpdate = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/UpdateConnector', {
            connector: csvDstConnectorUpdate,
            update_mask: "description,configuration",
        })

        check(resCSVDstUpdate, {
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateConnector ${resCSVDstUpdate.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateConnector ${resCSVDstUpdate.message.connector.id} response connector connectorDefinition`]: (r) => r.message.connector.connectorDefinitionName === constant.csvDstDefRscName,
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateConnector ${resCSVDstUpdate.message.connector.id} response connector description`]: (r) => r.message.connector.description === csvDstConnectorUpdate.description,
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateConnector ${resCSVDstUpdate.message.connector.id} response connector tombstone`]: (r) => r.message.connector.tombstone === false,
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateConnector ${resCSVDstUpdate.message.connector.id} response connector configuration`]: (r) => r.message.connector.configuration.destination_path === csvDstConnectorUpdate.configuration.destination_path,
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateConnector ${resCSVDstUpdate.message.connector.id} response connector owner is UUID`]: (r) => helper.isValidOwner(r.message.connector.user),
        });

        // Try to update with empty description
        csvDstConnectorUpdate = {
            "name": `connectors/${csvDstConnector.id}`,
            "description": "",
        }

        resCSVDstUpdate = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/UpdateConnector', {
            connector: csvDstConnectorUpdate,
            update_mask: "description",
        })

        check(resCSVDstUpdate, {
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateConnector ${resCSVDstUpdate.message.connector.id} with empty description response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateConnector ${resCSVDstUpdate.message.connector.id} with empty description response connector description`]: (r) => r.message.connector.description === csvDstConnectorUpdate.description,
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateConnector ${resCSVDstUpdate.message.connector.id} with empty description response connector owner is UUID`]: (r) => helper.isValidOwner(r.message.connector.user),
        });

        // Try to update with a non-existing name field (which should be ignored because name field is OUTPUT_ONLY)
        csvDstConnectorUpdate = {
            "name": `connectors/${randomString(5)}`,
            "description": randomString(50),
        }

        resCSVDstUpdate = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/UpdateConnector', {
            connector: csvDstConnectorUpdate,
            update_mask: "description",
        })
        check(resCSVDstUpdate, {
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateConnector with non-existing name field response StatusNotFound`]: (r) => r.status === grpc.StatusNotFound,
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${csvDstConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
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
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: csvDstConnector
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/LookUpConnector', {
            permalink: `destination_connector/${resCSVDst.message.connector.uid}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/LookUpConnector CSV ${resCSVDst.message.connector.uid} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/LookUpConnector CSV ${resCSVDst.message.connector.uid} response connector id`]: (r) => r.message.connector.uid === resCSVDst.message.connector.uid,
            [`vdp.connector.v1alpha.ConnectorPublicService/LookUpConnector CSV ${resCSVDst.message.connector.uid} response connector connectorDefinition permalink`]: (r) => r.message.connector.connectorDefinitionName === constant.csvDstDefRscName,
            [`vdp.connector.v1alpha.ConnectorPublicService/LookUpConnector CSV ${resCSVDst.message.connector.uid} response connector owner is UUID`]: (r) => helper.isValidOwner(r.message.connector.user),
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${csvDstConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
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
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: csvDstConnector
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ConnectConnector ${resCSVDst.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/DisconnectConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DisconnectConnector ${resCSVDst.message.connector.id} response at STATE_CONNECTED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ConnectConnector ${resCSVDst.message.connector.id} response at STATE_CONNECTED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ConnectConnector ${resCSVDst.message.connector.id} response at STATE_CONNECTED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/DisconnectConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DisconnectConnector ${resCSVDst.message.connector.id} response at STATE_CONNECTED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/DisconnectConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DisconnectConnector ${resCSVDst.message.connector.id} response at STATE_DISCONNECTED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ConnectConnector ${resCSVDst.message.connector.id} response at STATE_DISCONNECTED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${csvDstConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
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
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: csvDstConnector
        })

        let new_id = `some-id-not-${resCSVDst.message.connector.id}`

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/RenameConnector', {
            name: resCSVDst.message.connector.id,
            new_connector_id: new_id
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/RenameConnector ${resCSVDst.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/RenameConnector ${resCSVDst.message.connector.id} response id is some-id-not-${resCSVDst.message.connector.id}`]: (r) => r.message.connector.id === `some-id-not-${resCSVDst.message.connector.id}`,
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${new_id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${new_id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckExecute() {

    group("Connector API: Write destination connectors", () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var csvDstConnector, resCSVDst, currentTime, timeoutTime

        // Write classification output
        csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": {
                "destination_path": "/local/test-classification"
            },
        }

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: csvDstConnector
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnector', {
            "name": `destination_connector/${resCSVDst.message.connector.id}`,
            "inputs": constant.clsModelOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnector ${resCSVDst.message.connector.id} response (classification) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${resCSVDst.message.connector.id} response (classification) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Write detection output (empty bounding_boxes)
        csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": {
                "destination_path": "/local/test-detection-empty-bounding-boxes"
            },
        }

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: csvDstConnector
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnector', {
            "name": `destination_connector/${resCSVDst.message.connector.id}`,
            "inputs": constant.detectionEmptyModelOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnector ${resCSVDst.message.connector.id} response (detection) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${resCSVDst.message.connector.id} response (detection) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Write detection output (multiple models)
        csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": {
                "destination_path": "/local/test-detection-multi-models"
            },
        }

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: csvDstConnector
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnector', {
            "name": `destination_connector/${resCSVDst.message.connector.id}`,
            "inputs": constant.detectionModelOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnector ${resCSVDst.message.connector.id} response (detection) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${resCSVDst.message.connector.id} response (detection) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Write keypoint output
        csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": {
                "destination_path": "/local/test-keypoint"
            },
        }


        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: csvDstConnector
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnector', {
            "name": `destination_connector/${resCSVDst.message.connector.id}`,
            "inputs": constant.keypointModelOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnector ${resCSVDst.message.connector.id} response (keypoint) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${resCSVDst.message.connector.id} response (keypoint) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Write ocr output
        csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": {
                "destination_path": "/local/test-ocr"
            },
        }


        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: csvDstConnector
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnector', {
            "name": `destination_connector/${resCSVDst.message.connector.id}`,
            "inputs": constant.ocrModelOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnector ${resCSVDst.message.connector.id} response (ocr) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });
        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${resCSVDst.message.connector.id} response (ocr) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Write semantic segmentation output
        csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": {
                "destination_path": "/local/test-semantic-segmentation"
            },
        }

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: csvDstConnector
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnector', {
            "name": `destination_connector/${resCSVDst.message.connector.id}`,
            "inputs": constant.semanticSegModelOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnector ${resCSVDst.message.connector.id} response (semantic-segmentation) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${resCSVDst.message.connector.id} response (semantic-segmentation) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Write instance segmentation output
        csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": {
                "destination_path": "/local/test-instance-segmentation"
            },
        }

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: csvDstConnector
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnector', {
            "name": `destination_connector/${resCSVDst.message.connector.id}`,
            "inputs": constant.instSegModelOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnector ${resCSVDst.message.connector.id} response (instance-segmentation) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${resCSVDst.message.connector.id} response (instance-segmentation) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Write text-to-image output
        csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": {
                "destination_path": "/local/test-text-to-image"
            },
        }

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: csvDstConnector
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnector', {
            "name": `destination_connector/${resCSVDst.message.connector.id}`,
            "inputs": constant.textToImageModelOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnector ${resCSVDst.message.connector.id} response (text-to-image) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${resCSVDst.message.connector.id} response (text-to-image) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Write text-generation output
        csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": {
                "destination_path": "/local/test-text-generation"
            },
        }

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: csvDstConnector
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnector', {
            "name": `destination_connector/${resCSVDst.message.connector.id}`,
            "inputs": constant.textGenerationModelOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnector ${resCSVDst.message.connector.id} response (text-generation) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${resCSVDst.message.connector.id} response (text-generation) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Write unspecified output
        csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": {
                "destination_path": "/local/test-unspecified"
            },
        }

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: csvDstConnector
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnector', {
            "name": `destination_connector/${resCSVDst.message.connector.id}`,
            "inputs": constant.unspecifiedModelOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnector ${resCSVDst.message.connector.id} response (unspecified) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${resCSVDst.message.connector.id} response (unspecified) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckTest() {

    group("Connector API: Test destination connectors by ID", () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: csvDstConnector
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/TestConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/TestConnector CSV ${resCSVDst.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/TestConnector CSV ${resCSVDst.message.connector.id} response connector STATE_CONNECTED`]: (r) => r.message.state === "STATE_CONNECTED",
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${resCSVDst.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}
