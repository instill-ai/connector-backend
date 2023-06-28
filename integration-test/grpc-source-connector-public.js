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

        var httpSrcConnector = {
            "id": "source-http",
            "connector_definition": constant.httpSrcDefRscName,
            "description": "HTTP source",
            "configuration": {},
        }

        var gRPCSrcConnector = {
            "id": "source-grpc",
            "connector_definition": constant.gRPCSrcDefRscName,
            "description": "gRPC source",
            "configuration": {},
        }

        var resSrcHTTP = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: httpSrcConnector
        })
        check(resSrcHTTP, {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector response StatusOK": (r) => r.status === grpc.StatusOK,
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector response sourceConnector name": (r) => r.message.connector.name == `connectors/${httpSrcConnector.id}`,
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector response sourceConnector uid": (r) => helper.isUUID(r.message.connector.uid),
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector response sourceConnector connectorDefinition": (r) => r.message.connector.connectorDefinition === constant.httpSrcDefRscName,
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector response sourceConnector owner is UUID": (r) => helper.isValidOwner(r.message.connector.user),
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: httpSrcConnector
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector response StatusAlreadyExists": (r) => r.status === grpc.StatusAlreadyExists,
        });


        var resSrcGRPC = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: gRPCSrcConnector
        })
        check(resSrcGRPC, {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector response gRPCSrcConnector StatusOK": (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {}), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector response StatusInvalidArgument": (r) => r.status === grpc.StatusInvalidArgument,
        });

        // Delete test records
        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${resSrcHTTP.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${resSrcHTTP.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });
        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${resSrcGRPC.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${resSrcGRPC.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
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
            "id": "source-http",
            "connector_definition": constant.httpSrcDefRscName,
            "configuration": {}
        }

        reqBodies[1] = {
            "id": "source-grpc",
            "connector_definition": constant.gRPCSrcDefRscName,
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
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 view=VIEW_BASIC response has connectors[0].owner is UUID`]: (r) => helper.isValidOwner(r.message.connectors[0].user ),
        });
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectors', {
            filter: "connector_type=CONNECTOR_TYPE_SOURCE",
            pageSize: 1,
            view: "VIEW_FULL"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 view=VIEW_FULL response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 view=VIEW_FULL response has connectors[0].configuration is not null`]: (r) => r.message.connectors[0].configuration !== null,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 view=VIEW_FULL response has connectors[0].configuration is {}`]: (r) => Object.keys(r.message.connectors[0].configuration).length === 0,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 view=VIEW_FULL response has connectors[0].owner is UUID`]: (r) => helper.isValidOwner(r.message.connectors[0].user ),
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectors', {
            filter: "connector_type=CONNECTOR_TYPE_SOURCE",
            pageSize: 1,
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 response has connectors[0].configuration is null`]: (r) => r.message.connectors[0].configuration === null,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListConnectors pageSize=1 response has connectors[0].owner is UUID`]: (r) => helper.isValidOwner(r.message.connectors[0].user ),
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

        var httpSrcConnector = {
            "id": "source-http",
            "connector_definition": constant.httpSrcDefRscName,
            "configuration": {}
        }

        var resHTTP = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: httpSrcConnector
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetConnector', {
            name: `connectors/${resHTTP.message.connector.id}`
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/GetConnector name=connectors/${resHTTP.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/GetConnector name=connectors/${resHTTP.message.connector.id} response connector id`]: (r) => r.message.connector.id === httpSrcConnector.id,
            [`vdp.connector.v1alpha.ConnectorPublicService/GetConnector name=connectors/${resHTTP.message.connector.connectorDefinition} response connector id`]: (r) => r.message.connector.connectorDefinition === constant.httpSrcDefRscName,
            [`vdp.connector.v1alpha.ConnectorPublicService/GetConnector name=connectors/${resHTTP.message.connector.connectorDefinition} response connector owner is UUID`]: (r) => helper.isValidOwner(r.message.connector.user),
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

        var gRPCSrcConnector = {
            "id": "source-grpc",
            "connector_definition": constant.gRPCSrcDefRscName,
            "configuration": {}
        }

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: gRPCSrcConnector
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector response CreateConnector StatusOK": (r) => r.status === grpc.StatusOK,
        });

        gRPCSrcConnector.description = randomString(20)

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/UpdateConnector', {
            connector: gRPCSrcConnector
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateConnector ${gRPCSrcConnector.id} response StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${gRPCSrcConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${gRPCSrcConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
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
                "id": "source-http",
                "connector_definition": "connector-definitions/source-http",
                "configuration": {}
            }
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector response CreateConnector StatusOK": (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: {
                "id": "destination-http",
                "connector_definition": "connector-definitions/destination-http",
                "configuration": {}
            }
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector response StatusOK": (r) => r.status === grpc.StatusOK,
        });


        var createClsModelRes = http.request("POST", `${constant.modelPublicHost}/v1alpha/models`, JSON.stringify({
            "id": "dummy-cls",
            "model_definition": "model-definitions/github",
            "configuration": {
                "repository": "instill-ai/model-dummy-cls",
                "tag": "v1.0"
            },
        }), constant.params)
        check(createClsModelRes, {
            "POST /v1alpha/models cls response status is 201": (r) => r.status === 201,
        })
        // Check model creation finished
        let currentTime = new Date().getTime();
        let timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            var res = http.get(`${constant.modelPublicHost}/v1alpha/${createClsModelRes.json().operation.name}`, {
                headers: helper.genHeader(`application/json`),
            })
            if (res.json().operation.done === true) {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        const detSyncRecipe = {
            recipe: {
                "version": "v1alpha",
                "components": [
                    { "id": "s01", "resource_name": "connectors/source-http" },
                    { "id": "m01", "resource_name": "models/dummy-cls" },
                    { "id": "d01", "resource_name": "connectors/destination-http" },
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
            name: `connectors/source-http`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector source-http response StatusFailedPrecondition`]: (r) => r.status === grpc.StatusFailedPrecondition,
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector source-http response message not nil`]: (r) => r.message != {},
        });

        // Cannot delete destination connector due to pipeline occupancy
        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/destination-http`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector destination-http response StatusFailedPrecondition`]: (r) => r.status === grpc.StatusFailedPrecondition,
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector destination-http response message not nil`]: (r) => r.message != {},
        });

        check(http.request("DELETE", `${constant.pipelinePublicHost}/v1alpha/pipelines/${pipelineID}`), {
            [`DELETE /v1alpha/pipelines/${pipelineID} response status is 204`]: (r) => r.status === 204,
        });

        // Can delete source connector now
        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/source-http`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector source-http response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Can delete destination connector now
        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/destination-http`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector destination-http response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Wait for model state to be updated
        currentTime = new Date().getTime();
        timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            var res = client.invoke('vdp.model.v1alpha.ModelPublicService/WatchModel', {
                name: `models/dummy-cls`
            }, {})
            if (res.message.state !== "STATE_UNSPECIFIED") {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        // Can delete model now
        check(http.request("DELETE", `${constant.modelPublicHost}/v1alpha/models/dummy-cls`), {
            [`DELETE /v1alpha/models/dummy-cls response status is 204`]: (r) => r.status === 204,
        });

        client.close();
    });
}

export function CheckLookUp() {

    group("Connector API: Look up source connectors by UID", () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var httpSrcConnector = {
            "id": "source-http",
            "connector_definition": constant.httpSrcDefRscName,
            "configuration": {}
        }

        var resHTTP = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: httpSrcConnector
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/LookUpConnector', {
            permalink: `connectors/${resHTTP.message.connector.uid}`
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/LookUpConnector permalink=connectors/${resHTTP.message.connector.uid} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/LookUpConnector permalink=connectors/${resHTTP.message.connector.uid} response connector uid`]: (r) => r.message.connector.id === httpSrcConnector.id,
            [`vdp.connector.v1alpha.ConnectorPublicService/LookUpConnector permalink=connectors/${resHTTP.message.connector.uid} response connector id`]: (r) => r.message.connector.connectorDefinition === constant.httpSrcDefRscName,
            [`vdp.connector.v1alpha.ConnectorPublicService/LookUpConnector permalink=connectors/${resHTTP.message.connector.uid} response connector owner is UUID`]: (r) => helper.isValidOwner(r.message.connector.user),
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/source-http`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector source-http response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckState() {

    group("Connector API: Change state source connectors", () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var httpSrcConnector = {
            "id": "source-http",
            "connector_definition": constant.httpSrcDefRscName,
            "configuration": {}
        }

        var resHTTP = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: httpSrcConnector
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
            name: `connectors/source-http`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector source-http response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckRename() {

    group("Connector API: Rename source connectors", () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var httpSrcConnector = {
            "id": "source-http",
            "connector_definition": constant.httpSrcDefRscName,
            "configuration": {}
        }

        var resHTTP = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: httpSrcConnector
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/RenameConnector', {
            name: `connectors/${resHTTP.message.connector.id}`,
            new_connector_id: "some-id-not-http"
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/RenameConnector ${resHTTP.message.connector.id} response StatusFailedPrecondition`]: (r) => r.status === grpc.StatusFailedPrecondition,
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/source-http`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector source-http response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckTest() {

    group("Connector API: Test source connectors by ID", () => {
        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var httpSrcConnector = {
            "id": "source-http",
            "connector_definition": constant.httpSrcDefRscName,
            "configuration": {}
        }

        var resHTTP = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: httpSrcConnector
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
