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
                "configuration": JSON.stringify({})
            }
        }

        var dirGRPCSrcConnector = {
            "id": "source-grpc",
            "source_connector_definition": constant.gRPCSrcDefRscName,
            "connector": {
                "configuration": JSON.stringify({})
            }
        }

        var resSrcHTTP = http.request(
            "POST",
            `${connectorHost}/v1alpha/source-connectors`,
            JSON.stringify(dirHTTPSrcConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(resSrcHTTP, {
            "POST /v1alpha/source-connectors response status for creating directness HTTP source connector 201": (r) => r.status === 201,
            "POST /v1alpha/source-connectors response connector uid": (r) => helper.isUUID(r.json().source_connector.uid),
            "POST /v1alpha/source-connectors response connector source_connector_definition": (r) => r.json().source_connector.source_connector_definition === constant.httpSrcDefRscName
        });

        check(http.request(
            "POST",
            `${connectorHost}/v1alpha/source-connectors`,
            JSON.stringify(dirHTTPSrcConnector), {
            headers: { "Content-Type": "application/json" },
        }), {
            "POST /v1alpha/source-connectors response duplicate directness connector status 409": (r) => r.status === 409
        });

        var resSrcGRPC = http.request(
            "POST",
            `${connectorHost}/v1alpha/source-connectors`,
            JSON.stringify(dirGRPCSrcConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(resSrcGRPC, {
            "POST /v1alpha/source-connectors response status for creating directness gRPC source connector 201": (r) => r.status === 201,
        });

        check(http.request(
            "POST",
            `${connectorHost}/v1alpha/source-connectors`,
            JSON.stringify({}), {
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

        var reqBodies = [];
        reqBodies[0] = {
            "id": "source-http",
            "source_connector_definition": constant.httpSrcDefRscName,
            "connector": {
                "configuration": JSON.stringify({})
            }
        }

        reqBodies[1] = {
            "id": "source-grpc",
            "source_connector_definition": constant.gRPCSrcDefRscName,
            "connector": {
                "configuration": JSON.stringify({})
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

        var allRes = http.request("GET", `${connectorHost}/v1alpha/source-connectors`)
        check(http.request("GET", `${connectorHost}/v1alpha/source-connectors?page_size=0`), {
            "GET /v1alpha/source-connectors?page_size=0 response status is 200": (r) => r.status === 200,
            "GET /v1alpha/source-connectors?page_size=0 response all records": (r) => r.json().source_connectors.length === allRes.json().source_connectors.length,
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
            "GET /v1alpha/source-connectors?page_size=1&view=VIEW_BASIC response source_connectors has no configuration": (r) => JSON.parse(r.json().source_connectors[0].connector.configuration) === null
        });

        check(http.request("GET", `${connectorHost}/v1alpha/source-connectors?page_size=1&view=VIEW_FULL`), {
            "GET /v1alpha/source-connectors?page_size=1&view=VIEW_FULL response status 200": (r) => r.status === 200,
            "GET /v1alpha/source-connectors?page_size=1&view=VIEW_FULL response source_connectors has configuration": (r) => JSON.parse(r.json().source_connectors[0].connector.configuration) !== null,
        });

        check(http.request("GET", `${connectorHost}/v1alpha/source-connectors?page_size=1`), {
            "GET /v1alpha/source-connectors?page_size=1 response status 200": (r) => r.status === 200,
            "GET /v1alpha/source-connectors?page_size=1 response source_connectors has no configuration": (r) => JSON.parse(r.json().source_connectors[0].connector.configuration) === null
        });

        // Delete the destination connectors
        for (const reqBody of reqBodies) {
            check(http.request(
                "DELETE",
                `${connectorHost}/v1alpha/source-connectors/${reqBody.id}`,
                JSON.stringify(reqBody), {
                headers: {
                    "Content-Type": "application/json",
                },
            }), {
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
                "configuration": JSON.stringify({})
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
                "configuration": JSON.stringify({})
            }
        }

        check(http.request(
            "POST",
            `${connectorHost}/v1alpha/source-connectors`,
            JSON.stringify(dirGRPCSrcConnector), {
            headers: { "Content-Type": "application/json" },
        }), {
            "POST /v1alpha/source-connectors response status for creating directness gRPC source connector 201": (r) => r.status === 201,
        });

        dirGRPCSrcConnector.connector.description = randomString(20)
        check(http.request(
            "PATCH",
            `${connectorHost}/v1alpha/source-connectors/${dirGRPCSrcConnector.id}`,
            JSON.stringify(dirGRPCSrcConnector), {
            headers: { "Content-Type": "application/json" },
        }), {
            [`PATCH /v1alpha/source-connectors/${dirGRPCSrcConnector.id} response status for updating directness gRPC source connector 400`]: (r) => r.status === 400,
        });

        check(http.request("DELETE", `${connectorHost}/v1alpha/source-connectors/${dirGRPCSrcConnector.id}`), {
            [`DELETE /v1alpha/source-connectors/${dirGRPCSrcConnector.id} response status 204`]: (r) => r.status === 204,
        });

    });

}

export function CheckLookUp() {

    group("Connector API: Look up source connectors by UID", () => {

        var dirHTTPSrcConnector = {
            "id": "source-http",
            "source_connector_definition": constant.httpSrcDefRscName,
            "connector": {
                "configuration": JSON.stringify({})
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

export function CheckRename() {

    group("Connector API: Rename source connectors", () => {
        var dirHTTPSrcConnector = {
            "id": "source-http",
            "source_connector_definition": constant.httpSrcDefRscName,
            "connector": {
                "configuration": JSON.stringify({})
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
        }),{
            [`POST /v1alpha/source-connectors/${resHTTP.json().source_connector.id}:rename response status 400`]: (r) => r.status === 400,
        });

        check(http.request("DELETE", `${connectorHost}/v1alpha/source-connectors/${resHTTP.json().source_connector.id}`), {
            [`DELETE /v1alpha/source-connectors/${resHTTP.json().source_connector.id} response status 204`]: (r) => r.status === 204,
        });    });

}
