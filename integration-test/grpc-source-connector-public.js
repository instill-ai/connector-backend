import grpc from 'k6/net/grpc';
import http from "k6/http";
import {
    sleep,
    check,
    group
} from "k6";
import {
    randomString
} from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js"
import * as helper from "./helper.js"

const client = new grpc.Client();
client.load(['proto/vdp/connector/v1alpha'], 'connector_public_service.proto');

export function CheckCreate() {

    group("Connector API: Create source connector", () => {
        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var srcConnector = {
            "id": "trigger",
            "connector_definition_name": constant.srcDefRscName,
            "description": "HTTP source",
            "configuration": {},
        }


        var resSrc = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: srcConnector
        })
        check(resSrc, {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector response StatusOK": (r) => r.status === grpc.StatusOK,
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector response sourceConnector name": (r) => r.message.connector.name == `connectors/${srcConnector.id}`,
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector response sourceConnector uid": (r) => helper.isUUID(r.message.connector.uid),
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector response sourceConnector connectorDefinition": (r) => r.message.connector.connectorDefinitionName === constant.srcDefRscName,
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector response sourceConnector owner is UUID": (r) => helper.isValidOwner(r.message.connector.user),
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: srcConnector
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector response StatusAlreadyExists": (r) => r.status === grpc.StatusAlreadyExists,
        });



        // Delete test records
        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${resSrc.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${resSrc.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckList() {

    group("Connector API: List source connectors", () => {
        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectors', {
            filter: "connector_type=CONNECTOR_TYPE_SOURCE",
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors response connectors array is 0 length`]: (r) => r.message.connectors.length === 0,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors response nextPageToken is empty`]: (r) => r.message.nextPageToken === "",
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors response totalSize is 0`]: (r) => r.message.totalSize == 0,
        });

        var reqBodies = [];
        reqBodies[0] = {
            "id": "trigger",
            "connector_definition_name": constant.srcDefRscName,
            "configuration": {}
        }


        // Create connectors
        for (const reqBody of reqBodies) {
            check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
                connector: reqBody
            }), {
                [`vdp.connector.v1alpha.ConnectorPublicService/CreateConnector x${reqBodies.length} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            });
        }

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectors', {
            filter: "connector_type=CONNECTOR_TYPE_SOURCE",
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors response has connectors array`]: (r) => Array.isArray(r.message.connectors),
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors response has totalSize = ${reqBodies.length}`]: (r) => r.message.totalSize == reqBodies.length,
        });

        var limitedRecords = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectors', {
            filter: "connector_type=CONNECTOR_TYPE_SOURCE",
        }, {})
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectors', {
            filter: "connector_type=CONNECTOR_TYPE_SOURCE",
            pageSize: 0
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=0 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=0 response has connectors length`]: (r) => r.message.connectors.length === limitedRecords.message.connectors.length,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectors', {
            filter: "connector_type=CONNECTOR_TYPE_SOURCE",
            pageSize: 1
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 response has connectors length`]: (r) => r.message.connectors.length === 1,
        });

        var pageRes = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectors', {
            filter: "connector_type=CONNECTOR_TYPE_SOURCE",
            pageSize: 1
        }, {})
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectors', {
            filter: "connector_type=CONNECTOR_TYPE_SOURCE",
            pageSize: 1,
            pageToken: pageRes.message.nextPageToken
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 pageToken=${pageRes.message.nextPageToken} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 pageToken=${pageRes.message.nextPageToken} response has connectors length`]: (r) => r.message.connectors.length === 1,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectors', {
            filter: "connector_type=CONNECTOR_TYPE_SOURCE",
            pageSize: 1,
            view: "VIEW_BASIC"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 view=VIEW_BASIC response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 view=VIEW_BASIC response has connectors[0].configuration is null`]: (r) => r.message.connectors[0].configuration === null,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 view=VIEW_BASIC response has connectors[0].owner is UUID`]: (r) => helper.isValidOwner(r.message.connectors[0].user),
        });
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectors', {
            filter: "connector_type=CONNECTOR_TYPE_SOURCE",
            pageSize: 1,
            view: "VIEW_FULL"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 view=VIEW_FULL response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 view=VIEW_FULL response has connectors[0].configuration is not null`]: (r) => r.message.connectors[0].configuration !== null,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 view=VIEW_FULL response has connectors[0].connectorDefinitionDetail is not null`]: (r) => r.message.connectors[0].connectorDefinitionDetail !== null,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 view=VIEW_FULL response has connectors[0].configuration is {}`]: (r) => Object.keys(r.message.connectors[0].configuration).length === 0,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 view=VIEW_FULL response has connectors[0].owner is UUID`]: (r) => helper.isValidOwner(r.message.connectors[0].user),
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectors', {
            filter: "connector_type=CONNECTOR_TYPE_SOURCE",
            pageSize: 1,
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 response has connectors[0].configuration is null`]: (r) => r.message.connectors[0].configuration === null,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 response has connectors[0].owner is UUID`]: (r) => helper.isValidOwner(r.message.connectors[0].user),
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectors', {
            filter: "connector_type=CONNECTOR_TYPE_SOURCE",
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

    group("Connector API: Get source connectors by ID", () => {
        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var srcConnector = {
            "id": "trigger",
            "connector_definition_name": constant.srcDefRscName,
            "configuration": {}
        }

        var resHTTP = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: srcConnector
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetConnector', {
            name: `connectors/${resHTTP.message.connector.id}`
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/GetConnector name=connectors/${resHTTP.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/GetConnector name=connectors/${resHTTP.message.connector.id} response connector id`]: (r) => r.message.connector.id === srcConnector.id,
            [`vdp.connector.v1alpha.ConnectorPublicService/GetConnector name=connectors/${resHTTP.message.connector.connectorDefinitionName} response connector id`]: (r) => r.message.connector.connectorDefinitionName === constant.srcDefRscName,
            [`vdp.connector.v1alpha.ConnectorPublicService/GetConnector name=connectors/${resHTTP.message.connector.connectorDefinitionName} response connector owner is UUID`]: (r) => helper.isValidOwner(r.message.connector.user),
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${resHTTP.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${resHTTP.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckUpdate() {

    group("Connector API: Update source connectors", () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var srcConnector = {
            "id": "trigger",
            "connector_definition_name": constant.srcDefRscName,
            "configuration": {}
        }

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: srcConnector
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector response CreateConnector StatusOK": (r) => r.status === grpc.StatusOK,
        });

        srcConnector.description = randomString(20)

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/UpdateConnector', {
            connector: srcConnector
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateConnector ${srcConnector.id} response StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${srcConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${srcConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });

}

export function CheckDelete() {

    group("Connector API: Delete source connectors", () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: {
                "id": "trigger",
                "connector_definition_name": "connector-definitions/trigger",
                "configuration": {}
            }
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector response CreateConnector StatusOK": (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: {
                "id": "response",
                "connector_definition_name": "connector-definitions/response",
                "configuration": {}
            }
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector response StatusOK": (r) => r.status === grpc.StatusOK,
        });


        const detSyncRecipe = {
            recipe: {
                "version": "v1alpha",
                "components": [
                    { "id": "s01", "resource_name": "connectors/trigger" },
                    { "id": "d01", "resource_name": "connectors/response" },
                ]
            },
        };

        // Create a pipeline
        const pipelineID = randomString(5)
        check(http.request("POST", `${constant.pipelinePublicHost}/v1alpha/pipelines`,
            JSON.stringify(Object.assign({
                id: pipelineID,
                description: randomString(10),
            },
                detSyncRecipe
            )), constant.params), {
            "POST /v1alpha/pipelines response status is 201": (r) => r.status === 201,
        })

        // Cannot delete source connector due to pipeline occupancy
        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/trigger`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector trigger response StatusFailedPrecondition`]: (r) => r.status === grpc.StatusFailedPrecondition,
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector trigger response message not nil`]: (r) => r.message != {},
        });

        // Cannot delete destination connector due to pipeline occupancy
        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/response`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector response response StatusFailedPrecondition`]: (r) => r.status === grpc.StatusFailedPrecondition,
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector response response message not nil`]: (r) => r.message != {},
        });

        check(http.request("DELETE", `${constant.pipelinePublicHost}/v1alpha/pipelines/${pipelineID}`), {
            [`DELETE /v1alpha/pipelines/${pipelineID} response status is 204`]: (r) => r.status === 204,
        });

        // Can delete source connector now
        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/trigger`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector trigger response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Can delete destination connector now
        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/response`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector response response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });


        client.close();
    });
}

export function CheckLookUp() {

    group("Connector API: Look up source connectors by UID", () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var srcConnector = {
            "id": "trigger",
            "connector_definition_name": constant.srcDefRscName,
            "configuration": {}
        }

        var resHTTP = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: srcConnector
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/LookUpConnector', {
            permalink: `connectors/${resHTTP.message.connector.uid}`
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/LookUpConnector permalink=connectors/${resHTTP.message.connector.uid} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/LookUpConnector permalink=connectors/${resHTTP.message.connector.uid} response connector uid`]: (r) => r.message.connector.id === srcConnector.id,
            [`vdp.connector.v1alpha.ConnectorPublicService/LookUpConnector permalink=connectors/${resHTTP.message.connector.uid} response connector id`]: (r) => r.message.connector.connectorDefinitionName === constant.srcDefRscName,
            [`vdp.connector.v1alpha.ConnectorPublicService/LookUpConnector permalink=connectors/${resHTTP.message.connector.uid} response connector owner is UUID`]: (r) => helper.isValidOwner(r.message.connector.user),
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/trigger`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector trigger response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckState() {

    group("Connector API: Change state source connectors", () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var srcConnector = {
            "id": "trigger",
            "connector_definition_name": constant.srcDefRscName,
            "configuration": {}
        }

        var resHTTP = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: srcConnector
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnector', {
            name: `connectors/${resHTTP.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ConnectConnector ${resHTTP.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/DisconnectConnector', {
            name: `connectors/${resHTTP.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DisconnectConnector ${resHTTP.message.connector.id} response StatusFailedPrecondition`]: (r) => r.status === grpc.StatusFailedPrecondition,
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/trigger`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector trigger response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckRename() {

    group("Connector API: Rename source connectors", () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var srcConnector = {
            "id": "trigger",
            "connector_definition_name": constant.srcDefRscName,
            "configuration": {}
        }

        var resHTTP = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: srcConnector
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/RenameConnector', {
            name: `connectors/${resHTTP.message.connector.id}`,
            new_connector_id: "some-id-not-http"
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/RenameConnector ${resHTTP.message.connector.id} response StatusFailedPrecondition`]: (r) => r.status === grpc.StatusFailedPrecondition,
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/trigger`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector trigger response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckTest() {

    group("Connector API: Test source connectors by ID", () => {
        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var srcConnector = {
            "id": "trigger",
            "connector_definition_name": constant.srcDefRscName,
            "configuration": {}
        }

        var resHTTP = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: srcConnector
        })

        var testRes = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/TestConnector', {
            name: `connectors/${resHTTP.message.connector.id}`
        }, {})

        check(testRes, {
            [`vdp.connector.v1alpha.ConnectorPublicService/TestConnector name=connectors/${resHTTP.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/TestConnector name=connectors/${resHTTP.message.connector.id} response connector STATE_CONNECTED`]: (r) => r.message.state === "STATE_CONNECTED",
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${resHTTP.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${resHTTP.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}
