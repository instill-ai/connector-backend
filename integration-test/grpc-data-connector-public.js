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

        // destination-csv
        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource', {
            connector_resource: csvDstConnector
        })

        check(resCSVDst, {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource CSV response StatusOK": (r) => r.status === grpc.StatusOK,
        });
        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnectorResource', {
            name: `connector-resources/${csvDstConnector.id}`
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnectorResource', {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetConnectorResource', {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/GetConnectorResource CSV ${resCSVDst.message.connectorResource.id} response STATE_CONNECTED`]: (r) => r.message.connectorResource.state === "STATE_CONNECTED",
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

        var resDstMySQL = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource',
            {
                connector_resource: mySQLDstConnector,
            },
            {
                timeout: "600s",
            }
        )
        var resp = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnectorResource', {
            name: `connector-resources/${mySQLDstConnector.id}`
        })

        check(resDstMySQL, {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource MySQL response StatusOK": (r) => r.status === grpc.StatusOK,
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource MySQL response destinationConnector name": (r) => r.message.connectorResource.name == `connector-resources/${mySQLDstConnector.id}`,
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource MySQL response destinationConnector uid": (r) => helper.isUUID(r.message.connectorResource.uid),
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource MySQL response destinationConnector connectorDefinition": (r) => r.message.connectorResource.connectorDefinitionName === constant.mySQLDstDefRscName,
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource MySQL response destinationConnector owner is UUID": (r) => helper.isValidOwner(r.message.connectorResource.user),
        });

        // TODO: check jsonschema when connect

        // check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnectorResource', {
        //     name: `connector-resources/${resDstMySQL.message.connectorResource.id}`
        // }), {
        //     "vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource MySQL destination connector ended up STATE_ERROR": (r) => r.message.state === "STATE_ERROR",
        // })



        // check JSON Schema failure cases
        // var jsonSchemaFailedBodyCSV = {
        //     "id": randomString(10),
        //     "connector_definition_name": constant.csvDstDefRscName,
        //     "description": randomString(50),
        //     "configuration": {} // required destination_path
        // }

        // check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource', {
        //     connector_resource: jsonSchemaFailedBodyCSV
        // }), {
        //     "vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource response status for JSON Schema failed body 400 (destination-csv missing destination_path)": (r) => r.status === grpc.StatusInvalidArgument,
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

        // check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource', {
        //     connector_resource: jsonSchemaFailedBodyMySQL
        // }), {
        //     "vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource response status for JSON Schema failed body 400 (destination-mysql port not integer)": (r) => r.status === grpc.StatusInvalidArgument,
        // });

        // Delete test records
        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource`, {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource ${resCSVDst.message.connectorResource.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource`, {
            name: `connector-resources/${resDstMySQL.message.connectorResource.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource ${resDstMySQL.message.connectorResource.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });

}

export function CheckList() {

    group("Connector API: List destination connectors", () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources', {
            filter: "connector_type=CONNECTOR_TYPE_DATA",
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources response connectors array is 0 length`]: (r) => r.message.connectorResources.length === 0,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources response nextPageToken is empty`]: (r) => r.message.nextPageToken === "",
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources response totalSize is 0`]: (r) => r.message.totalSize == 0,
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
            var resDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource', {
                connector_resource: reqBody
            })
            client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnectorResource', {
                name: `connector-resources/${reqBody.id}`
            })

            check(resDst, {
                [`vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource x${reqBodies.length} HTTP response StatusOK`]: (r) => r.status === grpc.StatusOK,
            });
        }

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources', {
            filter: "connector_type=CONNECTOR_TYPE_DATA",
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources response has connectors array`]: (r) => Array.isArray(r.message.connectorResources),
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources response has totalSize = ${reqBodies.length}`]: (r) => r.message.totalSize == reqBodies.length,
        });

        var limitedRecords = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources', {
            filter: "connector_type=CONNECTOR_TYPE_DATA",
        }, {})
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources', {
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: 0
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources pageSize=0 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources pageSize=0 response all records`]: (r) => r.message.connectorResources.length === limitedRecords.message.connectorResources.length,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources', {
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: 1
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources pageSize=1 response size 1`]: (r) => r.message.connectorResources.length === 1,
        });

        var pageRes = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources', {
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: 1
        }, {})

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources', {
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: 1,
            pageToken: `${pageRes.message.nextPageToken}`
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources pageSize=1 pageToken=${pageRes.message.nextPageToken} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources pageSize=1 pageToken=${pageRes.message.nextPageToken} response size 1`]: (r) => r.message.connectorResources.length === 1,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources', {
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: 1,
            view: "VIEW_BASIC"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources pageSize=1 view=VIEW_BASIC response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources pageSize=1 view=VIEW_BASIC response connectors[0].configuration is null`]: (r) => r.message.connectorResources[0].configuration === null,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources pageSize=1 view=VIEW_BASIC response connectors[0].owner is UUID`]: (r) => helper.isValidOwner(r.message.connectorResources[0].user),
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources', {
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: 1,
            view: "VIEW_FULL"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources pageSize=1 view=VIEW_FULL response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources pageSize=1 view=VIEW_FULL response connectors[0].configuration is not null`]: (r) => r.message.connectorResources[0].configuration !== null,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources pageSize=1 view=VIEW_FULL response connectors[0].connectorDefinitionDetail is not null`]: (r) => r.message.connectorResources[0].connectorDefinitionDetail !== null,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources pageSize=1 view=VIEW_FULL response connectors[0].owner is UUID`]: (r) => helper.isValidOwner(r.message.connectorResources[0].user),
        });


        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources', {
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: 1,
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources pageSize=1 response connectors[0].configuration is null`]: (r) => r.message.connectorResources[0].configuration === null,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources pageSize=1 response connectors[0].owner is UUID`]: (r) => helper.isValidOwner(r.message.connectorResources[0].user),
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources', {
            filter: "connector_type=CONNECTOR_TYPE_DATA",
            pageSize: `${limitedRecords.message.totalSize}`,
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources pageSize=${limitedRecords.message.totalSize} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectorResources pageSize=${limitedRecords.message.totalSize} response nextPageToken is empty`]: (r) => r.message.nextPageToken === "",
        });

        // Delete the destination connectors
        for (const reqBody of reqBodies) {
            check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource`, {
                name: `connector-resources/${reqBody.id}`
            }), {
                [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource x${reqBodies.length} response StatusOK`]: (r) => r.status === grpc.StatusOK,
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

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource', {
            connector_resource: csvDstConnector
        })

        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnectorResource', {
            name: `connector-resources/${csvDstConnector.id}`
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnectorResource', {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetConnectorResource', {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/GetConnectorResource CSV ${resCSVDst.message.connectorResource.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/GetConnectorResource CSV ${resCSVDst.message.connectorResource.id} response connector id`]: (r) => r.message.connectorResource.id === csvDstConnector.id,
            [`vdp.connector.v1alpha.ConnectorPublicService/GetConnectorResource CSV ${resCSVDst.message.connectorResource.id} response connector connectorDefinition permalink`]: (r) => r.message.connectorResource.connectorDefinitionName === constant.csvDstDefRscName,
            [`vdp.connector.v1alpha.ConnectorPublicService/GetConnectorResource CSV ${resCSVDst.message.connectorResource.id} response connector owner is UUID`]: (r) => helper.isValidOwner(r.message.connectorResource.user),
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource`, {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource ${resCSVDst.message.connectorResource.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
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

        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource', {
            connector_resource: csvDstConnector
        })

        var csvDstConnectorUpdate = {
            "id": csvDstConnector.id,
            "name": `connector-resources/${csvDstConnector.id}`,
            "connector_definition_name": csvDstConnector.connector_definition_name,
            "tombstone": true,
            "description": randomString(50),
            "configuration": {
                destination_path: "/tmp"
            }
        }

        var resCSVDstUpdate = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/UpdateConnectorResource', {
            connector_resource: csvDstConnectorUpdate,
            update_mask: "description,configuration",
        })

        check(resCSVDstUpdate, {
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateConnectorResource ${resCSVDstUpdate.message.connectorResource.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateConnectorResource ${resCSVDstUpdate.message.connectorResource.id} response connector connectorDefinition`]: (r) => r.message.connectorResource.connectorDefinitionName === constant.csvDstDefRscName,
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateConnectorResource ${resCSVDstUpdate.message.connectorResource.id} response connector description`]: (r) => r.message.connectorResource.description === csvDstConnectorUpdate.description,
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateConnectorResource ${resCSVDstUpdate.message.connectorResource.id} response connector tombstone`]: (r) => r.message.connectorResource.tombstone === false,
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateConnectorResource ${resCSVDstUpdate.message.connectorResource.id} response connector configuration`]: (r) => r.message.connectorResource.configuration.destination_path === csvDstConnectorUpdate.configuration.destination_path,
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateConnectorResource ${resCSVDstUpdate.message.connectorResource.id} response connector owner is UUID`]: (r) => helper.isValidOwner(r.message.connectorResource.user),
        });

        // Try to update with empty description
        csvDstConnectorUpdate = {
            "name": `connector-resources/${csvDstConnector.id}`,
            "description": "",
        }

        resCSVDstUpdate = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/UpdateConnectorResource', {
            connector_resource: csvDstConnectorUpdate,
            update_mask: "description",
        })

        check(resCSVDstUpdate, {
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateConnectorResource ${resCSVDstUpdate.message.connectorResource.id} with empty description response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateConnectorResource ${resCSVDstUpdate.message.connectorResource.id} with empty description response connector description`]: (r) => r.message.connectorResource.description === csvDstConnectorUpdate.description,
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateConnectorResource ${resCSVDstUpdate.message.connectorResource.id} with empty description response connector owner is UUID`]: (r) => helper.isValidOwner(r.message.connectorResource.user),
        });

        // Try to update with a non-existing name field (which should be ignored because name field is OUTPUT_ONLY)
        csvDstConnectorUpdate = {
            "name": `connector-resources/${randomString(5)}`,
            "description": randomString(50),
        }

        resCSVDstUpdate = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/UpdateConnectorResource', {
            connector_resource: csvDstConnectorUpdate,
            update_mask: "description",
        })
        check(resCSVDstUpdate, {
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateConnectorResource with non-existing name field response StatusNotFound`]: (r) => r.status === grpc.StatusNotFound,
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource`, {
            name: `connector-resources/${csvDstConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
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

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource', {
            connector_resource: csvDstConnector
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/LookUpConnectorResource', {
            permalink: `destination_connector/${resCSVDst.message.connectorResource.uid}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/LookUpConnectorResource CSV ${resCSVDst.message.connectorResource.uid} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/LookUpConnectorResource CSV ${resCSVDst.message.connectorResource.uid} response connector id`]: (r) => r.message.connectorResource.uid === resCSVDst.message.connectorResource.uid,
            [`vdp.connector.v1alpha.ConnectorPublicService/LookUpConnectorResource CSV ${resCSVDst.message.connectorResource.uid} response connector connectorDefinition permalink`]: (r) => r.message.connectorResource.connectorDefinitionName === constant.csvDstDefRscName,
            [`vdp.connector.v1alpha.ConnectorPublicService/LookUpConnectorResource CSV ${resCSVDst.message.connectorResource.uid} response connector owner is UUID`]: (r) => helper.isValidOwner(r.message.connectorResource.user),
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource`, {
            name: `connector-resources/${csvDstConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
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

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource', {
            connector_resource: csvDstConnector
        })
        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnectorResource', {
            name: `connector-resources/${csvDstConnector.id}`
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnectorResource', {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnectorResource', {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ConnectConnectorResource ${resCSVDst.message.connectorResource.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/DisconnectConnectorResource', {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DisconnectConnectorResource ${resCSVDst.message.connectorResource.id} response at STATE_CONNECTED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnectorResource', {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ConnectConnectorResource ${resCSVDst.message.connectorResource.id} response at STATE_CONNECTED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnectorResource', {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnectorResource', {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ConnectConnectorResource ${resCSVDst.message.connectorResource.id} response at STATE_CONNECTED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnectorResource', {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/DisconnectConnectorResource', {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DisconnectConnectorResource ${resCSVDst.message.connectorResource.id} response at STATE_CONNECTED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/DisconnectConnectorResource', {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DisconnectConnectorResource ${resCSVDst.message.connectorResource.id} response at STATE_DISCONNECTED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnectorResource', {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ConnectConnectorResource ${resCSVDst.message.connectorResource.id} response at STATE_DISCONNECTED state StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnectorResource', {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource`, {
            name: `connector-resources/${csvDstConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
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

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource', {
            connector_resource: csvDstConnector
        })

        let new_id = `some-id-not-${resCSVDst.message.connectorResource.id}`

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/RenameConnectorResource', {
            name: resCSVDst.message.connectorResource.id,
            new_connector_id: new_id
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/RenameConnectorResource ${resCSVDst.message.connectorResource.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/RenameConnectorResource ${resCSVDst.message.connectorResource.id} response id is some-id-not-${resCSVDst.message.connectorResource.id}`]: (r) => r.message.connectorResource.id === `some-id-not-${resCSVDst.message.connectorResource.id}`,
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource`, {
            name: `connector-resources/${new_id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource ${new_id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
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

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource', {
            connector_resource: csvDstConnector
        })

        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnectorResource', {
            name: `connector-resources/${csvDstConnector.id}`
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnectorResource', {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnectorResource', {
            "name": `destination_connector/${resCSVDst.message.connectorResource.id}`,
            "inputs": constant.clsModelOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnectorResource ${resCSVDst.message.connectorResource.id} response (classification) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource`, {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource ${resCSVDst.message.connectorResource.id} response (classification) StatusOK`]: (r) => r.status === grpc.StatusOK,
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

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource', {
            connector_resource: csvDstConnector
        })
        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnectorResource', {
            name: `connector-resources/${csvDstConnector.id}`
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnectorResource', {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnectorResource', {
            "name": `destination_connector/${resCSVDst.message.connectorResource.id}`,
            "inputs": constant.detectionEmptyModelOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnectorResource ${resCSVDst.message.connectorResource.id} response (detection) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource`, {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource ${resCSVDst.message.connectorResource.id} response (detection) StatusOK`]: (r) => r.status === grpc.StatusOK,
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

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource', {
            connector_resource: csvDstConnector
        })
        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnectorResource', {
            name: `connector-resources/${csvDstConnector.id}`
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnectorResource', {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnectorResource', {
            "name": `destination_connector/${resCSVDst.message.connectorResource.id}`,
            "inputs": constant.detectionModelOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnectorResource ${resCSVDst.message.connectorResource.id} response (detection) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource`, {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource ${resCSVDst.message.connectorResource.id} response (detection) StatusOK`]: (r) => r.status === grpc.StatusOK,
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


        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource', {
            connector_resource: csvDstConnector
        })
        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnectorResource', {
            name: `connector-resources/${csvDstConnector.id}`
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnectorResource', {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnectorResource', {
            "name": `destination_connector/${resCSVDst.message.connectorResource.id}`,
            "inputs": constant.keypointModelOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnectorResource ${resCSVDst.message.connectorResource.id} response (keypoint) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource`, {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource ${resCSVDst.message.connectorResource.id} response (keypoint) StatusOK`]: (r) => r.status === grpc.StatusOK,
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


        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource', {
            connector_resource: csvDstConnector
        })
        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnectorResource', {
            name: `connector-resources/${csvDstConnector.id}`
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnectorResource', {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnectorResource', {
            "name": `destination_connector/${resCSVDst.message.connectorResource.id}`,
            "inputs": constant.ocrModelOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnectorResource ${resCSVDst.message.connectorResource.id} response (ocr) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });
        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource`, {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource ${resCSVDst.message.connectorResource.id} response (ocr) StatusOK`]: (r) => r.status === grpc.StatusOK,
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

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource', {
            connector_resource: csvDstConnector
        })
        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnectorResource', {
            name: `connector-resources/${csvDstConnector.id}`
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnectorResource', {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnectorResource', {
            "name": `destination_connector/${resCSVDst.message.connectorResource.id}`,
            "inputs": constant.semanticSegModelOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnectorResource ${resCSVDst.message.connectorResource.id} response (semantic-segmentation) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource`, {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource ${resCSVDst.message.connectorResource.id} response (semantic-segmentation) StatusOK`]: (r) => r.status === grpc.StatusOK,
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

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource', {
            connector_resource: csvDstConnector
        })
        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnectorResource', {
            name: `connector-resources/${csvDstConnector.id}`
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnectorResource', {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnectorResource', {
            "name": `destination_connector/${resCSVDst.message.connectorResource.id}`,
            "inputs": constant.instSegModelOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnectorResource ${resCSVDst.message.connectorResource.id} response (instance-segmentation) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource`, {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource ${resCSVDst.message.connectorResource.id} response (instance-segmentation) StatusOK`]: (r) => r.status === grpc.StatusOK,
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

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource', {
            connector_resource: csvDstConnector
        })
        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnectorResource', {
            name: `connector-resources/${csvDstConnector.id}`
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnectorResource', {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnectorResource', {
            "name": `destination_connector/${resCSVDst.message.connectorResource.id}`,
            "inputs": constant.textToImageModelOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnectorResource ${resCSVDst.message.connectorResource.id} response (text-to-image) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource`, {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource ${resCSVDst.message.connectorResource.id} response (text-to-image) StatusOK`]: (r) => r.status === grpc.StatusOK,
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

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource', {
            connector_resource: csvDstConnector
        })
        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnectorResource', {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnectorResource', {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnectorResource', {
            "name": `destination_connector/${resCSVDst.message.connectorResource.id}`,
            "inputs": constant.textGenerationModelOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnectorResource ${resCSVDst.message.connectorResource.id} response (text-generation) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource`, {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource ${resCSVDst.message.connectorResource.id} response (text-generation) StatusOK`]: (r) => r.status === grpc.StatusOK,
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

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource', {
            connector_resource: csvDstConnector
        })
        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnectorResource', {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnectorResource', {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnectorResource', {
            "name": `destination_connector/${resCSVDst.message.connectorResource.id}`,
            "inputs": constant.unspecifiedModelOutputs
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnectorResource ${resCSVDst.message.connectorResource.id} response (unspecified) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource`, {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource ${resCSVDst.message.connectorResource.id} response (unspecified) StatusOK`]: (r) => r.status === grpc.StatusOK,
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

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnectorResource', {
            connector_resource: csvDstConnector
        })

        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnectorResource', {
            name: `connector-resources/${csvDstConnector.id}`
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/TestConnectorResource', {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/TestConnectorResource CSV ${resCSVDst.message.connectorResource.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/TestConnectorResource CSV ${resCSVDst.message.connectorResource.id} response connector STATE_CONNECTED`]: (r) => r.message.state === "STATE_CONNECTED",
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource`, {
            name: `connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnectorResource ${resCSVDst.message.connectorResource.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}
