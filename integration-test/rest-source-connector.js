import http from "k6/http";
import { check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js"
import * as helper from "./helper.js"

export function CheckCreate() {

    group("Connector API: Create source connectors", () => {

        var dirHTTPSrcConnector = {
            "id": "source-http",
            "source_connector_definition": constant.httpSrcDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        var dirGRPCSrcConnector = {
            "id": "source-grpc",
            "source_connector_definition": constant.gRPCSrcDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        var resSrcHTTP = http.request(
            "POST",
            `${connectorHost}/v1alpha/source-connectors`,
            JSON.stringify(dirHTTPSrcConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(resSrcHTTP, {
            "POST /v1alpha/source-connectors response status for creating HTTP source connector 201": (r) => r.status === 201,
            "POST /v1alpha/source-connectors response connector name": (r) => r.json().source_connector.name == `source-connectors/${dirHTTPSrcConnector.id}`,
            "POST /v1alpha/source-connectors response connector uid": (r) => helper.isUUID(r.json().source_connector.uid),
            "POST /v1alpha/source-connectors response connector source_connector_definition": (r) => r.json().source_connector.source_connector_definition === constant.httpSrcDefRscName
        });

        check(http.request(
            "POST",
            `${connectorHost}/v1alpha/source-connectors`,
            JSON.stringify(dirHTTPSrcConnector), {
            headers: { "Content-Type": "application/json" },
        }), {
            "POST /v1alpha/source-connectors response duplicate HTTP source connector status 409": (r) => r.status === 409
        });

        var resSrcGRPC = http.request(
            "POST",
            `${connectorHost}/v1alpha/source-connectors`,
            JSON.stringify(dirGRPCSrcConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(resSrcGRPC, {
            "POST /v1alpha/source-connectors response status for creating gRPC source connector 201": (r) => r.status === 201,
        });

        check(http.request(
            "POST",
            `${connectorHost}/v1alpha/source-connectors`,
            {}, {
            headers: { "Content-Type": "application/json" },
        }), {
            "POST /v1alpha/source-connectors response status for creating empty body 400": (r) => r.status === 400,
        });

        // Delete test records
        check(http.request("DELETE", `${connectorHost}/v1alpha/source-connectors/${resSrcHTTP.json().source_connector.id}`), {
            [`DELETE /v1alpha/source-connectors/${resSrcHTTP.json().source_connector.id} response status 204`]: (r) => r.status === 204,
        });
        check(http.request("DELETE", `${connectorHost}/v1alpha/source-connectors/${resSrcGRPC.json().source_connector.id}`), {
            [`DELETE /v1alpha/source-connectors/${resSrcGRPC.json().source_connector.id} response status 204`]: (r) => r.status === 204,
        });
    });
}

export function CheckList() {

    group("Connector API: List source connectors", () => {

        check(http.request("GET", `${connectorHost}/v1alpha/source-connectors`), {
            [`GET /v1alpha/source-connectors empty db response status is 200`]: (r) => r.status === 200,
            [`GET /v1alpha/source-connectors empty db response has source_connectors array`]: (r) => Array.isArray(r.json().source_connectors),
            [`GET /v1alpha/source-connectors empty db response has total_size 0`]: (r) => r.json().total_size == 0,
            [`GET /v1alpha/source-connectors empty db response has empty next_page_token`]: (r) => r.json().next_page_token == "",
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
            check(http.request(
                "POST",
                `${connectorHost}/v1alpha/source-connectors`,
                JSON.stringify(reqBody), {
                headers: { "Content-Type": "application/json" },
            }), {
                [`POST /v1alpha/source-connectors x${reqBodies.length} response status 201`]: (r) => r.status === 201,
            });
        }

        check(http.request("GET", `${connectorHost}/v1alpha/source-connectors`), {
            [`GET /v1alpha/source-connectors response status is 200`]: (r) => r.status === 200,
            [`GET /v1alpha/source-connectors response has source_connectors array`]: (r) => Array.isArray(r.json().source_connectors),
            [`GET /v1alpha/source-connectors response has total_size = ${reqBodies.length}`]: (r) => r.json().total_size == reqBodies.length,
        });

        var limitedRecords = http.request("GET", `${connectorHost}/v1alpha/source-connectors`)
        check(http.request("GET", `${connectorHost}/v1alpha/source-connectors?page_size=0`), {
            "GET /v1alpha/source-connectors?page_size=0 response status is 200": (r) => r.status === 200,
            "GET /v1alpha/source-connectors?page_size=0 response all records": (r) => r.json().source_connectors.length === limitedRecords.json().source_connectors.length,
        });

        check(http.request("GET", `${connectorHost}/v1alpha/source-connectors?page_size=1`), {
            "GET /v1alpha/source-connectors?page_size=1 response status is 200": (r) => r.status === 200,
            "GET /v1alpha/source-connectors?page_size=1 response source_connectors size 1": (r) => r.json().source_connectors.length === 1,
        });

        var pageRes = http.request("GET", `${connectorHost}/v1alpha/source-connectors?page_size=1`)
        check(http.request("GET", `${connectorHost}/v1alpha/source-connectors?page_size=1&page_token=${pageRes.json().next_page_token}`), {
            [`GET /v1alpha/source-connectors?page_size=1&page_token=${pageRes.json().next_page_token} response status is 200`]: (r) => r.status === 200,
            [`GET /v1alpha/source-connectors?page_size=1&page_token=${pageRes.json().next_page_token} response source_connectors size 1`]: (r) => r.json().source_connectors.length === 1,
        });

        check(http.request("GET", `${connectorHost}/v1alpha/source-connectors?page_size=1&view=VIEW_BASIC`), {
            "GET /v1alpha/source-connectors?page_size=1&view=VIEW_BASIC response status 200": (r) => r.status === 200,
            "GET /v1alpha/source-connectors?page_size=1&view=VIEW_BASIC response source_connectors has no configuration": (r) => r.json().source_connectors[0].connector.configuration === null
        });

        check(http.request("GET", `${connectorHost}/v1alpha/source-connectors?page_size=1&view=VIEW_FULL`), {
            "GET /v1alpha/source-connectors?page_size=1&view=VIEW_FULL response status 200": (r) => r.status === 200,
            "GET /v1alpha/source-connectors?page_size=1&view=VIEW_FULL response source_connectors has configuration": (r) => r.json().source_connectors[0].connector.configuration !== null,
        });

        check(http.request("GET", `${connectorHost}/v1alpha/source-connectors?page_size=1`), {
            "GET /v1alpha/source-connectors?page_size=1 response status 200": (r) => r.status === 200,
            "GET /v1alpha/source-connectors?page_size=1 response source_connectors has no configuration": (r) => r.json().source_connectors[0].connector.configuration === null
        });

        check(http.request("GET", `${connectorHost}/v1alpha/source-connectors?page_size=${limitedRecords.json().total_size}`), {
            [`GET /v1alpha/source-connectors?page_size=${limitedRecords.json().total_size} response status 200`]: (r) => r.status === 200,
            [`GET /v1alpha/source-connectors?page_size=${limitedRecords.json().total_size} response next_page_token empty`]: (r) => r.json().next_page_token === "",
        });

        // Delete the destination connectors
        for (const reqBody of reqBodies) {
            check(http.request("DELETE", `${connectorHost}/v1alpha/source-connectors/${reqBody.id}`), {
                [`DELETE /v1alpha/source-connectors x${reqBodies.length} response status is 204`]: (r) => r.status === 204,
            });
        }
    });
}

export function CheckGet() {

    group("Connector API: Get source connectors by ID", () => {

        var dirHTTPSrcConnector = {
            "id": "source-http",
            "source_connector_definition": constant.httpSrcDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        var resHTTP = http.request("POST", `${connectorHost}/v1alpha/source-connectors`,
            JSON.stringify(dirHTTPSrcConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(http.request("GET", `${connectorHost}/v1alpha/source-connectors/${resHTTP.json().source_connector.id}`), {
            [`GET /v1alpha/source-connectors/${resHTTP.json().source_connector.id} response status 200`]: (r) => r.status === 200,
            [`GET /v1alpha/source-connectors/${resHTTP.json().source_connector.id} response connector id`]: (r) => r.json().source_connector.id === dirHTTPSrcConnector.id,
            [`GET /v1alpha/source-connectors/${resHTTP.json().source_connector.id} response connector source_connector_definition`]: (r) => r.json().source_connector.source_connector_definition === constant.httpSrcDefRscName,
        });

        check(http.request("DELETE", `${connectorHost}/v1alpha/source-connectors/${resHTTP.json().source_connector.id}`), {
            [`DELETE /v1alpha/source-connectors/${resHTTP.json().source_connector.id} response status 204`]: (r) => r.status === 204,
        });

    });
}

export function CheckUpdate() {

    group("Connector API: Update source connectors", () => {

        var dirGRPCSrcConnector = {
            "id": "source-grpc",
            "source_connector_definition": constant.gRPCSrcDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        check(http.request(
            "POST",
            `${connectorHost}/v1alpha/source-connectors`,
            JSON.stringify(dirGRPCSrcConnector), {
            headers: { "Content-Type": "application/json" },
        }), {
            "POST /v1alpha/source-connectors response status for creating gRPC source connector 201": (r) => r.status === 201,
        });

        dirGRPCSrcConnector.connector.description = randomString(20)
        check(http.request(
            "PATCH",
            `${connectorHost}/v1alpha/source-connectors/${dirGRPCSrcConnector.id}`,
            JSON.stringify(dirGRPCSrcConnector), {
            headers: { "Content-Type": "application/json" },
        }), {
            [`PATCH /v1alpha/source-connectors/${dirGRPCSrcConnector.id} response status for updating gRPC source connector 422`]: (r) => r.status === 422,
        });

        check(http.request("DELETE", `${connectorHost}/v1alpha/source-connectors/${dirGRPCSrcConnector.id}`), {
            [`DELETE /v1alpha/source-connectors/${dirGRPCSrcConnector.id} response status 204`]: (r) => r.status === 204,
        });

    });

}

export function CheckDelete() {

    group("Connector API: Delete source connectors", () => {

        check(http.request("POST", `${connectorHost}/v1alpha/source-connectors`,
            JSON.stringify({
                "id": "source-http",
                "source_connector_definition": "source-connector-definitions/source-http",
                "connector": {
                    "configuration": {}
                }
            }), { headers: { "Content-Type": "application/json" } }), {
            "POST /v1alpha/source-connectors response status for creating HTTP source connector 201": (r) => r.status === 201,
        })

        check(http.request("POST", `${connectorHost}/v1alpha/destination-connectors`,
            JSON.stringify({
                "id": "destination-http",
                "destination_connector_definition": "destination-connector-definitions/destination-http",
                "connector": {
                    "configuration": {}
                }
            }), { headers: { "Content-Type": "application/json" } }), {
            "POST /v1alpha/destination-connectors response status for creating HTTP destination connector 201": (r) => r.status === 201,
        })

        check(http.request("POST", `${modelHost}/v1alpha/models`, JSON.stringify({
            "id": "dummy-cls",
            "model_definition": "model-definitions/github",
            "configuration": JSON.stringify({
                "repository": "instill-ai/model-dummy-cls"
            }),
        }), { headers: { "Content-Type": "application/json" } }), {
            "POST /v1alpha/models:multipart task cls response status": (r) => r.status === 201,
        })

        const detSyncRecipe = {
            recipe: {
                source: "source-connectors/source-http",
                model_instances: [`models/dummy-cls/instances/v1.0`],
                destination: "destination-connectors/destination-http"
            },
        };

        // Create a pipeline
        const pipelineID = randomString(5)
        check(http.request("POST", `${pipelineHost}/v1alpha/pipelines`,
            JSON.stringify(Object.assign({
                id: pipelineID,
                description: randomString(10),
            },
                detSyncRecipe
            )), {
            headers: {
                "Content-Type": "application/json",
            },
        }), {
            "POST /v1alpha/pipelines response status is 201": (r) => r.status === 201,
        })

        // Cannot delete source connector due to pipeline occupancy
        check(http.request("DELETE", `${connectorHost}/v1alpha/source-connectors/source-http`), {
            [`DELETE /v1alpha/source-connectors/source-http response status 422`]: (r) => r.status === 422,
            [`DELETE /v1alpha/source-connectors/source-http response error msg not nil`]: (r) => r.json() != {},
        });

        // Cannot delete destination connector due to pipeline occupancy
        check(http.request("DELETE", `${connectorHost}/v1alpha/destination-connectors/destination-http`), {
            [`DELETE /v1alpha/destination-connectors/destination-http response status 422`]: (r) => r.status === 422,
            [`DELETE /v1alpha/destination-connectors/source-http response error msg not nil`]: (r) => r.json() != {},
        });

        check(http.request("DELETE", `${modelHost}/v1alpha/models/dummy-cls`), {
            [`DELETE /v1alpha/models/dummy-cls response status is 204`]: (r) => r.status === 204,
        });

        check(http.request("DELETE", `${pipelineHost}/v1alpha/pipelines/${pipelineID}`), {
            [`DELETE /v1alpha/pipelines/${pipelineID} response status is 204`]: (r) => r.status === 204,
        });

        // Can delete source connector now
        check(http.request("DELETE", `${connectorHost}/v1alpha/source-connectors/source-http`), {
            [`DELETE /v1alpha/source-connectors/source-http response status 204`]: (r) => r.status === 204,
        });

        // Can delete destination connector now
        check(http.request("DELETE", `${connectorHost}/v1alpha/destination-connectors/destination-http`), {
            [`DELETE /v1alpha/destination-connectors/destination-http response status 204`]: (r) => r.status === 204,
        });

    });
}

export function CheckLookUp() {

    group("Connector API: Look up source connectors by UID", () => {

        var dirHTTPSrcConnector = {
            "id": "source-http",
            "source_connector_definition": constant.httpSrcDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        var resHTTP = http.request("POST", `${connectorHost}/v1alpha/source-connectors`,
            JSON.stringify(dirHTTPSrcConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(http.request("GET", `${connectorHost}/v1alpha/source-connectors/${resHTTP.json().source_connector.uid}:lookUp`), {
            [`GET /v1alpha/source-connectors/${resHTTP.json().source_connector.uid}:lookUp response status 200`]: (r) => r.status === 200,
            [`GET /v1alpha/source-connectors/${resHTTP.json().source_connector.uid}:lookUp response connector uid`]: (r) => r.json().source_connector.uid === resHTTP.json().source_connector.uid,
            [`GET /v1alpha/source-connectors/${resHTTP.json().source_connector.uid}:lookUp response connector source_connector_definition`]: (r) => r.json().source_connector.source_connector_definition === constant.httpSrcDefRscName,
        });

        check(http.request("DELETE", `${connectorHost}/v1alpha/source-connectors/${resHTTP.json().source_connector.id}`), {
            [`DELETE /v1alpha/source-connectors/${resHTTP.json().source_connector.id} response status 204`]: (r) => r.status === 204,
        });

    });
}

export function CheckState() {

    group("Connector API: Change state source connectors", () => {
        var dirHTTPSrcConnector = {
            "id": "source-http",
            "source_connector_definition": constant.httpSrcDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        var resHTTP = http.request("POST", `${connectorHost}/v1alpha/source-connectors`,
            JSON.stringify(dirHTTPSrcConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(http.request("POST", `${connectorHost}/v1alpha/source-connectors/${resHTTP.json().source_connector.id}:connect`, null, {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/source-connectors/${resHTTP.json().source_connector.id}:connect response status 200`]: (r) => r.status === 200,
        });

        check(http.request("POST", `${connectorHost}/v1alpha/source-connectors/${resHTTP.json().source_connector.id}:disconnect`, null, {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/source-connectors/${resHTTP.json().source_connector.id}:disconnect response status 422`]: (r) => r.status === 422,
        });

        check(http.request("DELETE", `${connectorHost}/v1alpha/source-connectors/${resHTTP.json().source_connector.id}`), {
            [`DELETE /v1alpha/source-connectors/${resHTTP.json().source_connector.id} response status 204`]: (r) => r.status === 204,
        });

    });

}

export function CheckRename() {

    group("Connector API: Rename source connectors", () => {
        var dirHTTPSrcConnector = {
            "id": "source-http",
            "source_connector_definition": constant.httpSrcDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        var resHTTP = http.request("POST", `${connectorHost}/v1alpha/source-connectors`,
            JSON.stringify(dirHTTPSrcConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(http.request("POST", `${connectorHost}/v1alpha/source-connectors/${resHTTP.json().source_connector.id}:rename`,
            JSON.stringify({
                "new_source_connector_id": "some-id-not-http"
            }), {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/source-connectors/${resHTTP.json().source_connector.id}:rename response status 422`]: (r) => r.status === 422,
        });

        check(http.request("DELETE", `${connectorHost}/v1alpha/source-connectors/${resHTTP.json().source_connector.id}`), {
            [`DELETE /v1alpha/source-connectors/${resHTTP.json().source_connector.id} response status 204`]: (r) => r.status === 204,
        });
    });

}
