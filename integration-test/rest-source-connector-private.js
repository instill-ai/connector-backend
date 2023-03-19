import http from "k6/http";
import {
    check,
    group
} from "k6";

import {
    connectorHost
} from "./const.js"

import * as constant from "./const.js"

export function CheckList() {

    group("Connector API: List source connectors by admin", () => {

        check(http.request("GET", `${connectorHost}/v1alpha/admin/source-connectors`), {
            [`GET /v1alpha/admin/source-connectors response status is 200`]: (r) => r.status === 200,
            [`GET /v1alpha/admin/source-connectors response source_connectors array is 0 length`]: (r) => r.json().source_connectors.length === 0,
            [`GET /v1alpha/admin/source-connectors response next_page_token is empty`]: (r) => r.json().next_page_token === "",
            [`GET /v1alpha/admin/source-connectors response total_size is 0`]: (r) => r.json().next_page_token == 0,
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
                    headers: {
                        "Content-Type": "application/json"
                    },
                }), {
                [`POST /v1alpha/source-connectors x${reqBodies.length} response status 201`]: (r) => r.status === 201,
            });
        }

        check(http.request("GET", `${connectorHost}/v1alpha/admin/source-connectors`), {
            [`GET /v1alpha/admin/source-connectors response status is 200`]: (r) => r.status === 200,
            [`GET /v1alpha/admin/source-connectors response has source_connectors array`]: (r) => Array.isArray(r.json().source_connectors),
            [`GET /v1alpha/admin/source-connectors response has total_size = ${reqBodies.length}`]: (r) => r.json().total_size == reqBodies.length,
        });

        var limitedRecords = http.request("GET", `${connectorHost}/v1alpha/admin/source-connectors`)
        check(http.request("GET", `${connectorHost}/v1alpha/admin/source-connectors?page_size=0`), {
            "GET /v1alpha/admin/source-connectors?page_size=0 response status is 200": (r) => r.status === 200,
            "GET /v1alpha/admin/source-connectors?page_size=0 response all records": (r) => r.json().source_connectors.length === limitedRecords.json().source_connectors.length,
        });

        check(http.request("GET", `${connectorHost}/v1alpha/admin/source-connectors?page_size=1`), {
            "GET /v1alpha/admin/source-connectors?page_size=1 response status is 200": (r) => r.status === 200,
            "GET /v1alpha/admin/source-connectors?page_size=1 response source_connectors size 1": (r) => r.json().source_connectors.length === 1,
        });

        var pageRes = http.request("GET", `${connectorHost}/v1alpha/admin/source-connectors?page_size=1`)
        check(http.request("GET", `${connectorHost}/v1alpha/admin/source-connectors?page_size=1&page_token=${pageRes.json().next_page_token}`), {
            [`GET /v1alpha/admin/source-connectors?page_size=1&page_token=${pageRes.json().next_page_token} response status is 200`]: (r) => r.status === 200,
            [`GET /v1alpha/admin/source-connectors?page_size=1&page_token=${pageRes.json().next_page_token} response source_connectors size 1`]: (r) => r.json().source_connectors.length === 1,
        });

        check(http.request("GET", `${connectorHost}/v1alpha/admin/source-connectors?page_size=1&view=VIEW_BASIC`), {
            "GET /v1alpha/admin/source-connectors?page_size=1&view=VIEW_BASIC response status 200": (r) => r.status === 200,
            "GET /v1alpha/admin/source-connectors?page_size=1&view=VIEW_BASIC response source_connectors[0]connector.configuration is null": (r) => r.json().source_connectors[0].connector.configuration === null,
        });

        check(http.request("GET", `${connectorHost}/v1alpha/admin/source-connectors?page_size=1&view=VIEW_FULL`), {
            "GET /v1alpha/admin/source-connectors?page_size=1&view=VIEW_FULL response status 200": (r) => r.status === 200,
            "GET /v1alpha/admin/source-connectors?page_size=1&view=VIEW_FULL response source_connectors[0]connector.configuration is not null": (r) => r.json().source_connectors[0].connector.configuration !== null,
            "GET /v1alpha/admin/source-connectors?page_size=1&view=VIEW_BASIC response source_connectors[0]connector.configuration is {}": (r) => Object.keys(r.json().source_connectors[0].connector.configuration).length === 0,
        });

        check(http.request("GET", `${connectorHost}/v1alpha/admin/source-connectors?page_size=1`), {
            "GET /v1alpha/admin/source-connectors?page_size=1 response status 200": (r) => r.status === 200,
            "GET /v1alpha/admin/source-connectors?page_size=1 response source_connectors[0]connector.configuration is null": (r) => r.json().source_connectors[0].connector.configuration === null
        });

        check(http.request("GET", `${connectorHost}/v1alpha/admin/source-connectors?page_size=${limitedRecords.json().total_size}`), {
            [`GET /v1alpha/admin/source-connectors?page_size=${limitedRecords.json().total_size} response status 200`]: (r) => r.status === 200,
            [`GET /v1alpha/admin/source-connectors?page_size=${limitedRecords.json().total_size} response next_page_token is empty`]: (r) => r.json().next_page_token === ""
        });

        // Delete the destination connectors
        for (const reqBody of reqBodies) {
            check(http.request("DELETE", `${connectorHost}/v1alpha/source-connectors/${reqBody.id}`), {
                [`DELETE /v1alpha/admin/source-connectors x${reqBodies.length} response status is 204`]: (r) => r.status === 204,
            });
        }
    });
}

export function CheckGet() {

    group("Connector API: Get source connectors by ID by admin", () => {

        var httpSrcConnector = {
            "id": "source-http",
            "source_connector_definition": constant.httpSrcDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        var resHTTP = http.request("POST", `${connectorHost}/v1alpha/source-connectors`,
            JSON.stringify(httpSrcConnector), {
                headers: {
                    "Content-Type": "application/json"
                },
            })

        check(http.request("GET", `${connectorHost}/v1alpha/admin/source-connectors/${resHTTP.json().source_connector.id}`), {
            [`GET /v1alpha/admin/source-connectors/${resHTTP.json().source_connector.id} response status 200`]: (r) => r.status === 200,
            [`GET /v1alpha/admin/source-connectors/${resHTTP.json().source_connector.id} response connector id`]: (r) => r.json().source_connector.id === httpSrcConnector.id,
            [`GET /v1alpha/admin/source-connectors/${resHTTP.json().source_connector.id} response connector source_connector_definition`]: (r) => r.json().source_connector.source_connector_definition === constant.httpSrcDefRscName,
        });

        check(http.request("DELETE", `${connectorHost}/v1alpha/source-connectors/${resHTTP.json().source_connector.id}`), {
            [`DELETE /v1alpha/admin/source-connectors/${resHTTP.json().source_connector.id} response status 204`]: (r) => r.status === 204,
        });

    });
}

export function CheckLookUp() {

    group("Connector API: Look up source connectors by UID by admin", () => {

        var httpSrcConnector = {
            "id": "source-http",
            "source_connector_definition": constant.httpSrcDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        var resHTTP = http.request("POST", `${connectorHost}/v1alpha/source-connectors`,
            JSON.stringify(httpSrcConnector), {
                headers: {
                    "Content-Type": "application/json"
                },
            })

        check(http.request("GET", `${connectorHost}/v1alpha/admin/source-connectors/${resHTTP.json().source_connector.uid}/lookUp`), {
            [`GET /v1alpha/admin/source-connectors/${resHTTP.json().source_connector.uid}/lookUp response status 200`]: (r) => r.status === 200,
            [`GET /v1alpha/admin/source-connectors/${resHTTP.json().source_connector.uid}/lookUp response connector uid`]: (r) => r.json().source_connector.uid === resHTTP.json().source_connector.uid,
            [`GET /v1alpha/admin/source-connectors/${resHTTP.json().source_connector.uid}/lookUp response connector source_connector_definition`]: (r) => r.json().source_connector.source_connector_definition === constant.httpSrcDefRscName,
        });

        check(http.request("DELETE", `${connectorHost}/v1alpha/source-connectors/${resHTTP.json().source_connector.id}`), {
            [`DELETE /v1alpha/admin/source-connectors/${resHTTP.json().source_connector.id} response status 204`]: (r) => r.status === 204,
        });

    });
}