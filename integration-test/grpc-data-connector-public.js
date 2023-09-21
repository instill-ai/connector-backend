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

export function CheckCreate(metadata) {

    group("Connector API: Create destination connectors", () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        // destination-csv
        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
            parent: `${constant.namespace}`,
            connector_resource: csvDstConnector
        }, metadata)

        check(resCSVDst, {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource CSV response StatusOK": (r) => r.status === grpc.StatusOK,
        });
        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        }, metadata)

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/GetUserConnectorResource CSV ${resCSVDst.message.connectorResource.id} response STATE_CONNECTED`]: (r) => r.message.connectorResource.state === "STATE_CONNECTED",
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

        var resDstMySQL = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource',
            {
                parent: `${constant.namespace}`,
                connector_resource: mySQLDstConnector,
            }, metadata
        )
        var resp = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${mySQLDstConnector.id}`
        }, metadata)

        check(resDstMySQL, {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource MySQL response StatusOK": (r) => r.status === grpc.StatusOK,
            "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource MySQL response destinationConnector name": (r) => r.message.connectorResource.name == `${constant.namespace}/connector-resources/${mySQLDstConnector.id}`,
            "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource MySQL response destinationConnector uid": (r) => helper.isUUID(r.message.connectorResource.uid),
            "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource MySQL response destinationConnector connectorDefinition": (r) => r.message.connectorResource.connectorDefinitionName === constant.mySQLDstDefRscName,
            "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource MySQL response destinationConnector owner is UUID": (r) => helper.isValidOwner(r.message.connectorResource.user),
        });

        // TODO: check jsonschema when connect

        // check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchUserConnectorResource', {
        //     name: `${constant.namespace}/connector-resources/${resDstMySQL.message.connectorResource.id}`
        // }), {
        //     "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource MySQL destination connector ended up STATE_ERROR": (r) => r.message.state === "STATE_ERROR",
        // })



        // check JSON Schema failure cases
        // var jsonSchemaFailedBodyCSV = {
        //     "id": randomString(10),
        //     "connector_definition_name": constant.csvDstDefRscName,
        //     "description": randomString(50),
        //     "configuration": {} // required destination_path
        // }

        // check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
        //     connector_resource: jsonSchemaFailedBodyCSV
        // }), {
        //     "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource response status for JSON Schema failed body 400 (destination-csv missing destination_path)": (r) => r.status === grpc.StatusInvalidArgument,
        // });

        // var jsonSchemaFailedBodyMySQL = {
        //     "id": randomString(10),
        //     "connector_definition_name": constant.mySQLDstDefRscName,
        //     "description": randomString(50),
        //     "configuration": {
        //         "host": randomString(10),
        //         "port": "3306",
        //         "username": randomString(10),
        //         "database": randomString(10),
        //     } // required port integer type
        // }

        // check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
        //     connector_resource: jsonSchemaFailedBodyMySQL
        // }), {
        //     "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource response status for JSON Schema failed body 400 (destination-mysql port not integer)": (r) => r.status === grpc.StatusInvalidArgument,
        // });

        // Delete test records
        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${resCSVDst.message.connectorResource.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
            name: `${constant.namespace}/connector-resources/${resDstMySQL.message.connectorResource.id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${resDstMySQL.message.connectorResource.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });

}

export function CheckList(metadata) {

    group("Connector API: List destination connectors", () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources', {
            parent: `${constant.namespace}`,
            filter: "connector_type=CONNECTOR_TYPE_DATA",
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources response connectors array is 0 length`]: (r) => r.message.connectorResources.length === 0,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources response nextPageToken is empty`]: (r) => r.message.nextPageToken === "",
            [`vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources response totalSize is 0`]: (r) => r.message.totalSize == 0,
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
            var resDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
                parent: `${constant.namespace}`,
                connector_resource: reqBody
            }, metadata)
            client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
                name: `${constant.namespace}/connector-resources/${reqBody.id}`
            }, metadata)

            check(resDst, {
                [`vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource x${reqBodies.length} HTTP response StatusOK`]: (r) => r.status === grpc.StatusOK,
            });
        }

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources', {
            parent: `${constant.namespace}`,
            filter: "connector_type=CONNECTOR_TYPE_DATA",
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources response has connectors array`]: (r) => Array.isArray(r.message.connectorResources),
            [`vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources response has totalSize = ${reqBodies.length}`]: (r) => r.message.totalSize == reqBodies.length,
        });

        var limitedRecords = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources', {
            parent: `${constant.namespace}`,
            filter: "connector_type=CONNECTOR_TYPE_DATA",
        }, metadata)
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources', {
            parent: `${constant.namespace}`,
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: 0
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources pageSize=0 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources pageSize=0 response all records`]: (r) => r.message.connectorResources.length === limitedRecords.message.connectorResources.length,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources', {
            parent: `${constant.namespace}`,
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: 1
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources pageSize=1 response size 1`]: (r) => r.message.connectorResources.length === 1,
        });

        var pageRes = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources', {
            parent: `${constant.namespace}`,
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: 1
        }, metadata)

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources', {
            parent: `${constant.namespace}`,
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: 1,
            pageToken: `${pageRes.message.nextPageToken}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources pageSize=1 pageToken=${pageRes.message.nextPageToken} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources pageSize=1 pageToken=${pageRes.message.nextPageToken} response size 1`]: (r) => r.message.connectorResources.length === 1,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources', {
            parent: `${constant.namespace}`,
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: 1,
            view: "VIEW_BASIC"
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources pageSize=1 view=VIEW_BASIC response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources pageSize=1 view=VIEW_BASIC response connectors[0].configuration is null`]: (r) => r.message.connectorResources[0].configuration === null,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources pageSize=1 view=VIEW_BASIC response connectors[0].owner is UUID`]: (r) => helper.isValidOwner(r.message.connectorResources[0].user),
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources', {
            parent: `${constant.namespace}`,
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: 1,
            view: "VIEW_FULL"
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources pageSize=1 view=VIEW_FULL response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources pageSize=1 view=VIEW_FULL response connectors[0].configuration is not null`]: (r) => r.message.connectorResources[0].configuration !== null,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources pageSize=1 view=VIEW_FULL response connectors[0].connectorDefinitionDetail is not null`]: (r) => r.message.connectorResources[0].connectorDefinitionDetail !== null,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources pageSize=1 view=VIEW_FULL response connectors[0].owner is UUID`]: (r) => helper.isValidOwner(r.message.connectorResources[0].user),
        });


        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources', {
            parent: `${constant.namespace}`,
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: 1,
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources pageSize=1 response connectors[0].configuration is null`]: (r) => r.message.connectorResources[0].configuration === null,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources pageSize=1 response connectors[0].owner is UUID`]: (r) => helper.isValidOwner(r.message.connectorResources[0].user),
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources', {
            parent: `${constant.namespace}`,
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: `${limitedRecords.message.totalSize}`,
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources pageSize=${limitedRecords.message.totalSize} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources pageSize=${limitedRecords.message.totalSize} response nextPageToken is empty`]: (r) => r.message.nextPageToken === "",
        });

        // Delete the destination connectors
        for (const reqBody of reqBodies) {
            check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
                name: `${constant.namespace}/connector-resources/${reqBody.id}`
            }, metadata), {
                [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource x${reqBodies.length} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            });
        }

        client.close();
    });
}

export function CheckGet(metadata) {

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

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
            parent: `${constant.namespace}`,
            connector_resource: csvDstConnector
        }, metadata)

        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        }, metadata)

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/GetUserConnectorResource CSV ${resCSVDst.message.connectorResource.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/GetUserConnectorResource CSV ${resCSVDst.message.connectorResource.id} response connector id`]: (r) => r.message.connectorResource.id === csvDstConnector.id,
            [`vdp.connector.v1alpha.ConnectorPublicService/GetUserConnectorResource CSV ${resCSVDst.message.connectorResource.id} response connector connectorDefinition permalink`]: (r) => r.message.connectorResource.connectorDefinitionName === constant.csvDstDefRscName,
            [`vdp.connector.v1alpha.ConnectorPublicService/GetUserConnectorResource CSV ${resCSVDst.message.connectorResource.id} response connector owner is UUID`]: (r) => helper.isValidOwner(r.message.connectorResource.user),
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${resCSVDst.message.connectorResource.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckUpdate(metadata) {

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

        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
            parent: `${constant.namespace}`,
            connector_resource: csvDstConnector
        }, metadata)

        var csvDstConnectorUpdate = {
            "id": csvDstConnector.id,
            "name": `${constant.namespace}/connector-resources/${csvDstConnector.id}`,
            "connector_definition_name": csvDstConnector.connector_definition_name,
            "tombstone": true,
            "description": randomString(50),
            "configuration": {
                destination_path: "/tmp"
            }
        }

        var resCSVDstUpdate = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/UpdateUserConnectorResource', {
            connector_resource: csvDstConnectorUpdate,
            update_mask: "description,configuration",
        }, metadata)

        check(resCSVDstUpdate, {
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateUserConnectorResource ${resCSVDstUpdate.message.connectorResource.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateUserConnectorResource ${resCSVDstUpdate.message.connectorResource.id} response connector connectorDefinition`]: (r) => r.message.connectorResource.connectorDefinitionName === constant.csvDstDefRscName,
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateUserConnectorResource ${resCSVDstUpdate.message.connectorResource.id} response connector description`]: (r) => r.message.connectorResource.description === csvDstConnectorUpdate.description,
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateUserConnectorResource ${resCSVDstUpdate.message.connectorResource.id} response connector tombstone`]: (r) => r.message.connectorResource.tombstone === false,
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateUserConnectorResource ${resCSVDstUpdate.message.connectorResource.id} response connector configuration`]: (r) => r.message.connectorResource.configuration.destination_path === csvDstConnectorUpdate.configuration.destination_path,
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateUserConnectorResource ${resCSVDstUpdate.message.connectorResource.id} response connector owner is UUID`]: (r) => helper.isValidOwner(r.message.connectorResource.user),
        });

        // Try to update with empty description
        csvDstConnectorUpdate = {
            "name": `${constant.namespace}/connector-resources/${csvDstConnector.id}`,
            "description": "",
        }

        resCSVDstUpdate = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/UpdateUserConnectorResource', {
            connector_resource: csvDstConnectorUpdate,
            update_mask: "description",
        }, metadata)

        check(resCSVDstUpdate, {
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateUserConnectorResource ${resCSVDstUpdate.message.connectorResource.id} with empty description response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateUserConnectorResource ${resCSVDstUpdate.message.connectorResource.id} with empty description response connector description`]: (r) => r.message.connectorResource.description === csvDstConnectorUpdate.description,
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateUserConnectorResource ${resCSVDstUpdate.message.connectorResource.id} with empty description response connector owner is UUID`]: (r) => helper.isValidOwner(r.message.connectorResource.user),
        });

        // Try to update with a non-existing name field (which should be ignored because name field is OUTPUT_ONLY)
        csvDstConnectorUpdate = {
            "name": `${constant.namespace}/connector-resources/${randomString(5)}`,
            "description": randomString(50),
        }

        resCSVDstUpdate = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/UpdateUserConnectorResource', {
            connector_resource: csvDstConnectorUpdate,
            update_mask: "description",
        }, metadata)
        check(resCSVDstUpdate, {
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateUserConnectorResource with non-existing name field response StatusNotFound`]: (r) => r.status === grpc.StatusNotFound,
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
            name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckLookUp(metadata) {

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

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
            parent: `${constant.namespace}`,
            connector_resource: csvDstConnector
        }, metadata)

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/LookUpConnectorResource', {
            permalink: `connector-resources/${resCSVDst.message.connectorResource.uid}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/LookUpConnectorResource CSV ${resCSVDst.message.connectorResource.uid} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/LookUpConnectorResource CSV ${resCSVDst.message.connectorResource.uid} response connector id`]: (r) => r.message.connectorResource.uid === resCSVDst.message.connectorResource.uid,
            [`vdp.connector.v1alpha.ConnectorPublicService/LookUpConnectorResource CSV ${resCSVDst.message.connectorResource.uid} response connector connectorDefinition permalink`]: (r) => r.message.connectorResource.connectorDefinitionName === constant.csvDstDefRscName,
            [`vdp.connector.v1alpha.ConnectorPublicService/LookUpConnectorResource CSV ${resCSVDst.message.connectorResource.uid} response connector owner is UUID`]: (r) => helper.isValidOwner(r.message.connectorResource.user),
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
            name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckState(metadata) {

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

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
            parent: `${constant.namespace}`,
            connector_resource: csvDstConnector
        }, metadata)
        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        }, metadata)

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource ${resCSVDst.message.connectorResource.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/DisconnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DisconnectUserConnectorResource ${resCSVDst.message.connectorResource.id} response at STATE_CONNECTED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource ${resCSVDst.message.connectorResource.id} response at STATE_CONNECTED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource ${resCSVDst.message.connectorResource.id} response at STATE_CONNECTED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/DisconnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DisconnectUserConnectorResource ${resCSVDst.message.connectorResource.id} response at STATE_CONNECTED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/DisconnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DisconnectUserConnectorResource ${resCSVDst.message.connectorResource.id} response at STATE_DISCONNECTED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource ${resCSVDst.message.connectorResource.id} response at STATE_DISCONNECTED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
            name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckRename(metadata) {

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

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
            parent: `${constant.namespace}`,
            connector_resource: csvDstConnector
        }, metadata)

        let new_id = `some-id-not-${resCSVDst.message.connectorResource.id}`

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/RenameUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`,
            new_connector_id: new_id
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/RenameUserConnectorResource ${resCSVDst.message.connectorResource.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/RenameUserConnectorResource ${resCSVDst.message.connectorResource.id} response id is some-id-not-${resCSVDst.message.connectorResource.id}`]: (r) => r.message.connectorResource.id === `some-id-not-${resCSVDst.message.connectorResource.id}`,
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
            name: `${constant.namespace}/connector-resources/${new_id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${new_id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckExecute(metadata) {

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

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
            parent: `${constant.namespace}`,
            connector_resource: csvDstConnector
        }, metadata)

        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        }, metadata)

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteUserConnectorResource', {
            "name": `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`,
            "inputs": constant.clsModelOutputs
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteUserConnectorResource ${resCSVDst.message.connectorResource.id} response (classification) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${resCSVDst.message.connectorResource.id} response (classification) StatusOK`]: (r) => r.status === grpc.StatusOK,
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

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
            parent: `${constant.namespace}`,
            connector_resource: csvDstConnector
        }, metadata)
        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        }, metadata)

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteUserConnectorResource', {
            "name": `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`,
            "inputs": constant.detectionEmptyModelOutputs
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteUserConnectorResource ${resCSVDst.message.connectorResource.id} response (detection) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${resCSVDst.message.connectorResource.id} response (detection) StatusOK`]: (r) => r.status === grpc.StatusOK,
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

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
            parent: `${constant.namespace}`,
            connector_resource: csvDstConnector
        }, metadata)
        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        }, metadata)

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteUserConnectorResource', {
            "name": `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`,
            "inputs": constant.detectionModelOutputs
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteUserConnectorResource ${resCSVDst.message.connectorResource.id} response (detection) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${resCSVDst.message.connectorResource.id} response (detection) StatusOK`]: (r) => r.status === grpc.StatusOK,
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


        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
            parent: `${constant.namespace}`,
            connector_resource: csvDstConnector
        }, metadata)
        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        }, metadata)

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteUserConnectorResource', {
            "name": `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`,
            "inputs": constant.keypointModelOutputs
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteUserConnectorResource ${resCSVDst.message.connectorResource.id} response (keypoint) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${resCSVDst.message.connectorResource.id} response (keypoint) StatusOK`]: (r) => r.status === grpc.StatusOK,
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


        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
            parent: `${constant.namespace}`,
            connector_resource: csvDstConnector
        }, metadata)
        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        }, metadata)

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteUserConnectorResource', {
            "name": `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`,
            "inputs": constant.ocrModelOutputs
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteUserConnectorResource ${resCSVDst.message.connectorResource.id} response (ocr) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });
        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${resCSVDst.message.connectorResource.id} response (ocr) StatusOK`]: (r) => r.status === grpc.StatusOK,
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

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
            parent: `${constant.namespace}`,
            connector_resource: csvDstConnector
        }, metadata)
        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        }, metadata)

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteUserConnectorResource', {
            "name": `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`,
            "inputs": constant.semanticSegModelOutputs
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteUserConnectorResource ${resCSVDst.message.connectorResource.id} response (semantic-segmentation) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${resCSVDst.message.connectorResource.id} response (semantic-segmentation) StatusOK`]: (r) => r.status === grpc.StatusOK,
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

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
            parent: `${constant.namespace}`,
            connector_resource: csvDstConnector
        }, metadata)
        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        }, metadata)

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteUserConnectorResource', {
            "name": `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`,
            "inputs": constant.instSegModelOutputs
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteUserConnectorResource ${resCSVDst.message.connectorResource.id} response (instance-segmentation) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${resCSVDst.message.connectorResource.id} response (instance-segmentation) StatusOK`]: (r) => r.status === grpc.StatusOK,
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

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
            parent: `${constant.namespace}`,
            connector_resource: csvDstConnector
        }, metadata)
        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        }, metadata)

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteUserConnectorResource', {
            "name": `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`,
            "inputs": constant.textToImageModelOutputs
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteUserConnectorResource ${resCSVDst.message.connectorResource.id} response (text-to-image) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${resCSVDst.message.connectorResource.id} response (text-to-image) StatusOK`]: (r) => r.status === grpc.StatusOK,
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

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
            parent: `${constant.namespace}`,
            connector_resource: csvDstConnector
        }, metadata)
        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata)

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteUserConnectorResource', {
            "name": `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`,
            "inputs": constant.textGenerationModelOutputs
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteUserConnectorResource ${resCSVDst.message.connectorResource.id} response (text-generation) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${resCSVDst.message.connectorResource.id} response (text-generation) StatusOK`]: (r) => r.status === grpc.StatusOK,
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

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
            parent: `${constant.namespace}`,
            connector_resource: csvDstConnector
        }, metadata)
        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata)

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteUserConnectorResource', {
            "name": `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`,
            "inputs": constant.unspecifiedModelOutputs
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteUserConnectorResource ${resCSVDst.message.connectorResource.id} response (unspecified) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${resCSVDst.message.connectorResource.id} response (unspecified) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckTest(metadata) {

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

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
            parent: `${constant.namespace}`,
            connector_resource: csvDstConnector
        }, metadata)

        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        }, metadata)

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/TestUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/TestUserConnectorResource CSV ${resCSVDst.message.connectorResource.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/TestUserConnectorResource CSV ${resCSVDst.message.connectorResource.id} response connector STATE_CONNECTED`]: (r) => r.message.state === "STATE_CONNECTED",
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, metadata), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${resCSVDst.message.connectorResource.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}
