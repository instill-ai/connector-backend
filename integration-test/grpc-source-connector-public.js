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
            "source_connector_definition": constant.httpSrcDefRscName,
            "connector": {
                "description": "HTTP source",
                "configuration": {},
            }
        }

        var gRPCSrcConnector = {
            "id": "source-grpc",
            "source_connector_definition": constant.gRPCSrcDefRscName,
            "connector": {
                "description": "gRPC source",
                "configuration": {},
            }
        }

        var resSrcHTTP = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector', {
            source_connector: httpSrcConnector
        })
        check(resSrcHTTP, {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector response StatusOK": (r) => r.status === grpc.StatusOK,
            "vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector response sourceConnector name": (r) => r.message.sourceConnector.name == `source-connectors/${httpSrcConnector.id}`,
            "vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector response sourceConnector uid": (r) => helper.isUUID(r.message.sourceConnector.uid),
            "vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector response sourceConnector sourceConnectorDefinition": (r) => r.message.sourceConnector.sourceConnectorDefinition === constant.httpSrcDefRscName
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector', {
            source_connector: httpSrcConnector
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector response StatusAlreadyExists": (r) => r.status === grpc.StatusAlreadyExists,
        });


        var resSrcGRPC = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector', {
            source_connector: gRPCSrcConnector
        })

        check(resSrcGRPC, {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector response gRPCSrcConnector StatusOK": (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector', {}), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector response StatusInvalidArgument": (r) => r.status === grpc.StatusInvalidArgument,
        });

        // Delete test records
        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector`, {
            name: `source-connectors/${resSrcHTTP.message.sourceConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector ${resSrcHTTP.message.sourceConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });
        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector`, {
            name: `source-connectors/${resSrcGRPC.message.sourceConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector ${resSrcGRPC.message.sourceConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckList() {

    group("Connector API: List source connectors", () => {
        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors', {}, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors response sourceConnectors array is 0 length`]: (r) => r.message.sourceConnectors.length === 0,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors response nextPageToken is empty`]: (r) => r.message.nextPageToken === "",
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors response totalSize is 0`]: (r) => r.message.totalSize == 0,
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
            check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector', {
                source_connector: reqBody
            }), {
                [`vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector x${reqBodies.length} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            });
        }

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors', {}, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors response has sourceConnectors array`]: (r) => Array.isArray(r.message.sourceConnectors),
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors response has totalSize = ${reqBodies.length}`]: (r) => r.message.totalSize == reqBodies.length,
        });

        var limitedRecords = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors', {}, {})
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors', {
            pageSize: 0
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors pageSize=0 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors pageSize=0 response has sourceConnectors length`]: (r) => r.message.sourceConnectors.length === limitedRecords.message.sourceConnectors.length,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors', {
            pageSize: 1
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors pageSize=1 response has sourceConnectors length`]: (r) => r.message.sourceConnectors.length === 1,
        });

        var pageRes = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors', {
            pageSize: 1
        }, {})
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors', {
            pageSize: 1,
            pageToken: pageRes.message.nextPageToken
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors pageSize=1 pageToken=${pageRes.message.nextPageToken} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors pageSize=1 pageToken=${pageRes.message.nextPageToken} response has sourceConnectors length`]: (r) => r.message.sourceConnectors.length === 1,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors', {
            pageSize: 1,
            view: "VIEW_BASIC"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors pageSize=1 view=VIEW_BASIC response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors pageSize=1 view=VIEW_BASIC response has sourceConnectors[0].connector.configuration is null`]: (r) => r.message.sourceConnectors[0].connector.configuration === null,
        });
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors', {
            pageSize: 1,
            view: "VIEW_FULL"
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors pageSize=1 view=VIEW_FULL response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors pageSize=1 view=VIEW_FULL response has sourceConnectors[0].connector.configuration is not null`]: (r) => r.message.sourceConnectors[0].connector.configuration !== null,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors pageSize=1 view=VIEW_FULL response has sourceConnectors[0].connector.configuration is {}`]: (r) => Object.keys(r.message.sourceConnectors[0].connector.configuration).length === 0,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors', {
            pageSize: 1,
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors pageSize=1 response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors pageSize=1 response has sourceConnectors[0].connector.configuration is null`]: (r) => r.message.sourceConnectors[0].connector.configuration === null,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors', {
            pageSize: `${limitedRecords.message.totalSize}`,
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors pageSize=${limitedRecords.message.totalSize} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors pageSize=${limitedRecords.message.totalSize} response nextPageToken is empty`]: (r) => r.message.nextPageToken === "",
        });

        // Delete the destination connectors
        for (const reqBody of reqBodies) {
            check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector`, {
                name: `source-connectors/${reqBody.id}`
            }), {
                [`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector x${reqBodies.length} response StatusOK`]: (r) => r.status === grpc.StatusOK,
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
            "source_connector_definition": constant.httpSrcDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        var resHTTP = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector', {
            source_connector: httpSrcConnector
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetSourceConnector', {
            name: `source-connectors/${resHTTP.message.sourceConnector.id}`
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/GetSourceConnector name=source-connectors/${resHTTP.message.sourceConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/GetSourceConnector name=source-connectors/${resHTTP.message.sourceConnector.id} response connector id`]: (r) => r.message.sourceConnector.id === httpSrcConnector.id,
            [`vdp.connector.v1alpha.ConnectorPublicService/GetSourceConnector name=source-connectors/${resHTTP.message.sourceConnector.sourceConnectorDefinition} response connector id`]: (r) => r.message.sourceConnector.sourceConnectorDefinition === constant.httpSrcDefRscName,
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector`, {
            name: `source-connectors/${resHTTP.message.sourceConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector ${resHTTP.message.sourceConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
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
            "source_connector_definition": constant.gRPCSrcDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector', {
            source_connector: gRPCSrcConnector
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector response CreateSourceConnector StatusOK": (r) => r.status === grpc.StatusOK,
        });

        gRPCSrcConnector.connector.description = randomString(20)

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/UpdateSourceConnector', {
            source_connector: gRPCSrcConnector
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/UpdateSourceConnector ${gRPCSrcConnector.id} response StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector`, {
            name: `source-connectors/${gRPCSrcConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector ${gRPCSrcConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });

}

export function CheckDelete() {

    group("Connector API: Delete source connectors", () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector', {
            source_connector: {
                "id": "source-http",
                "source_connector_definition": "source-connector-definitions/source-http",
                "connector": {
                    "configuration": {}
                }
            }
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector response CreateSourceConnector StatusOK": (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: {
                "id": "destination-http",
                "destination_connector_definition": "destination-connector-definitions/destination-http",
                "connector": {
                    "configuration": {}
                }
            }
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector response StatusOK": (r) => r.status === grpc.StatusOK,
        });


        var createClsModelRes = http.request("POST", `${constant.modelPublicHost}/v1alpha/models`, JSON.stringify({
            "id": "dummy-cls",
            "model_definition": "model-definitions/github",
            "configuration": {
                "repository": "instill-ai/model-dummy-cls"
            },
        }), constant.params)
        check(createClsModelRes, {
            "POST /v1alpha/models cls response status": (r) => r.status === 201,
        })
        // Check model creation finished
        let currentTime = new Date().getTime();
        let timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            let res = http.get(`${constant.modelPublicHost}/v1alpha/${createClsModelRes.json().operation.name}`, constant.params)
            if (res.json().operation.done === true) {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        const detSyncRecipe = {
            recipe: {
                source: "source-connectors/source-http",
                model_instances: [`models/dummy-cls/instances/v1.0-cpu`],
                destination: "destination-connectors/destination-http"
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
        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector`, {
            name: `source-connectors/source-http`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector source-http response StatusFailedPrecondition`]: (r) => r.status === grpc.StatusFailedPrecondition,
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector source-http response message not nil`]: (r) => r.message !== {},
        });

        // Cannot delete destination connector due to pipeline occupancy
        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/destination-http`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector destination-http response StatusFailedPrecondition`]: (r) => r.status === grpc.StatusFailedPrecondition,
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector destination-http response message not nil`]: (r) => r.message !== {},
        });

        // Cannot delete model due to pipeline occupancy
        check(http.request("DELETE", `${constant.modelPublicHost}/v1alpha/models/dummy-cls`), {
            [`DELETE /v1alpha/models/dummy-cls response status is 204`]: (r) => r.status === 422,
            [`DELETE /v1alpha/models/dummy-cls response error msg not nil`]: (r) => r.json() != {},
        });

        check(http.request("DELETE", `${constant.pipelinePublicHost}/v1alpha/pipelines/${pipelineID}`), {
            [`DELETE /v1alpha/pipelines/${pipelineID} response status is 204`]: (r) => r.status === 204,
        });

        // Can delete source connector now
        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector`, {
            name: `source-connectors/source-http`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector source-http response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        // Can delete destination connector now
        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/destination-http`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector destination-http response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

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
            "source_connector_definition": constant.httpSrcDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        var resHTTP = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector', {
            source_connector: httpSrcConnector
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/LookUpSourceConnector', {
            permalink: `source-connectors/${resHTTP.message.sourceConnector.uid}`
        }, {}), {
            [`vdp.connector.v1alpha.ConnectorPublicService/LookUpSourceConnector permalink=source-connectors/${resHTTP.message.sourceConnector.uid} response StatusOK`]: (r) => r.status === grpc.StatusOK,
            [`vdp.connector.v1alpha.ConnectorPublicService/LookUpSourceConnector permalink=source-connectors/${resHTTP.message.sourceConnector.uid} response connector uid`]: (r) => r.message.sourceConnector.id === httpSrcConnector.id,
            [`vdp.connector.v1alpha.ConnectorPublicService/LookUpSourceConnector permalink=source-connectors/${resHTTP.message.sourceConnector.uid} response connector id`]: (r) => r.message.sourceConnector.sourceConnectorDefinition === constant.httpSrcDefRscName,
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector`, {
            name: `source-connectors/source-http`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector source-http response StatusOK`]: (r) => r.status === grpc.StatusOK,
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
            "source_connector_definition": constant.httpSrcDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        var resHTTP = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector', {
            source_connector: httpSrcConnector
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectSourceConnector', {
            name: `source-connectors/${resHTTP.message.sourceConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/ConnectSourceConnector ${resHTTP.message.sourceConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/DisconnectSourceConnector', {
            name: `source-connectors/${resHTTP.message.sourceConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DisconnectSourceConnector ${resHTTP.message.sourceConnector.id} response StatusFailedPrecondition`]: (r) => r.status === grpc.StatusFailedPrecondition,
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector`, {
            name: `source-connectors/source-http`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector source-http response StatusOK`]: (r) => r.status === grpc.StatusOK,
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
            "source_connector_definition": constant.httpSrcDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        var resHTTP = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector', {
            source_connector: httpSrcConnector
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/RenameSourceConnector', {
            name: `source-connectors/${resHTTP.message.sourceConnector.id}`,
            new_source_connector_id: "some-id-not-http"
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/RenameSourceConnector ${resHTTP.message.sourceConnector.id} response StatusFailedPrecondition`]: (r) => r.status === grpc.StatusFailedPrecondition,
        });

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector`, {
            name: `source-connectors/source-http`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector source-http response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}
