import http from "k6/http";
import {
    check,
    group,
    sleep
} from "k6";
import {
    randomString
} from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import {
    connectorPublicHost,
    connectorPrivateHost
} from "./const.js"

import * as constant from "./const.js"

export function CheckList() {

    group("Connector API: List destination connectors by admin", () => {

        check(http.request("GET", `${connectorPrivateHost}/v1alpha/admin/destination-connectors`), {
            [`GET /v1alpha/admin/destination-connectors response status is 200`]: (r) => r.status === 200,
            [`GET /v1alpha/admin/destination-connectors response destination_connectors array is 0 length`]: (r) => r.json().destination_connectors.length === 0,
            [`GET /v1alpha/admin/destination-connectors response next_page_token is empty`]: (r) => r.json().next_page_token === "",
            [`GET /v1alpha/admin/destination-connectors response total_size is 0`]: (r) => r.json().total_size == 0,
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
            var resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors`,
                JSON.stringify(reqBody), constant.params)
            check(resCSVDst, {
                [`POST /v1alpha/destination-connectors x${reqBodies.length} response status 201`]: (r) => r.status === 201,
            });
        }

        check(http.request("GET", `${connectorPrivateHost}/v1alpha/admin/destination-connectors`), {
            [`GET /v1alpha/admin/destination-connectors response status is 200`]: (r) => r.status === 200,
            [`GET /v1alpha/admin/destination-connectors response has destination_connectors array`]: (r) => Array.isArray(r.json().destination_connectors),
            [`GET /v1alpha/admin/destination-connectors response has total_size = ${numConnectors}`]: (r) => r.json().total_size == numConnectors,
        });

        var limitedRecords = http.request("GET", `${connectorPrivateHost}/v1alpha/admin/destination-connectors`)
        check(http.request("GET", `${connectorPrivateHost}/v1alpha/admin/destination-connectors?page_size=0`), {
            "GET /v1alpha/admin/destination-connectors?page_size=0 response status is 200": (r) => r.status === 200,
            "GET /v1alpha/admin/destination-connectors?page_size=0 response all records": (r) => r.json().destination_connectors.length === limitedRecords.json().destination_connectors.length,
        });

        check(http.request("GET", `${connectorPrivateHost}/v1alpha/admin/destination-connectors?page_size=1`), {
            "GET /v1alpha/admin/destination-connectors?page_size=1 response status is 200": (r) => r.status === 200,
            "GET /v1alpha/admin/destination-connectors?page_size=1 response destination_connectors size 1": (r) => r.json().destination_connectors.length === 1,
        });

        var pageRes = http.request("GET", `${connectorPrivateHost}/v1alpha/admin/destination-connectors?page_size=1`)
        check(http.request("GET", `${connectorPrivateHost}/v1alpha/admin/destination-connectors?page_size=1&page_token=${pageRes.json().next_page_token}`), {
            [`GET /v1alpha/admin/destination-connectors?page_size=1&page_token=${pageRes.json().next_page_token} response status is 200`]: (r) => r.status === 200,
            [`GET /v1alpha/admin/destination-connectors?page_size=1&page_token=${pageRes.json().next_page_token} response destination_connectors size 1`]: (r) => r.json().destination_connectors.length === 1,
        });

        check(http.request("GET", `${connectorPrivateHost}/v1alpha/admin/destination-connectors?page_size=1&view=VIEW_BASIC`), {
            "GET /v1alpha/admin/destination-connectors?page_size=1&view=VIEW_BASIC response status 200": (r) => r.status === 200,
            "GET /v1alpha/admin/destination-connectors?page_size=1&view=VIEW_BASIC response destination_connectors[0].connector.configuration is null": (r) => r.json().destination_connectors[0].connector.configuration === null,
        });

        check(http.request("GET", `${connectorPrivateHost}/v1alpha/admin/destination-connectors?page_size=1&view=VIEW_FULL`), {
            "GET /v1alpha/admin/destination-connectors?page_size=1&view=VIEW_FULL response status 200": (r) => r.status === 200,
            "GET /v1alpha/admin/destination-connectors?page_size=1&view=VIEW_FULL response destination_connectors[0].connector.configuration is not null": (r) => r.json().destination_connectors[0].connector.configuration !== null
        });

        check(http.request("GET", `${connectorPrivateHost}/v1alpha/admin/destination-connectors?page_size=1`), {
            "GET /v1alpha/admin/destination-connectors?page_size=1 response status 200": (r) => r.status === 200,
            "GET /v1alpha/admin/destination-connectors?page_size=1 response destination_connectors[0].connector.configuration is null": (r) => r.json().destination_connectors[0].connector.configuration === null,
        });

        check(http.request("GET", `${connectorPrivateHost}/v1alpha/admin/destination-connectors?page_size=${limitedRecords.json().total_size}`), {
            [`GET /v1alpha/admin/destination-connectors?page_size=${limitedRecords.json().total_size} response status 200`]: (r) => r.status === 200,
            [`GET /v1alpha/admin/destination-connectors?page_size=${limitedRecords.json().total_size} response next_page_token is empty`]: (r) => r.json().next_page_token === ""
        });

        // Delete the destination connectors
        for (const reqBody of reqBodies) {
            check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${reqBody.id}`), {
                [`DELETE /v1alpha/admin/destination-connectors x${reqBodies.length} response status is 204`]: (r) => r.status === 204,
            });
        }
    });
}

export function CheckGet() {

    group("Connector API: Get destination connectors by ID by admin", () => {

        var csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": constant.csvDstConfig
            }
        }

        var resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors`,
            JSON.stringify(csvDstConnector), constant.params)

        // Check connector state being updated in 120 secs
        var currentTime = new Date().getTime();
        var timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            var res = http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch`)
            if (res.json().state === "STATE_CONNECTED") {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        check(http.request("GET", `${connectorPrivateHost}/v1alpha/admin/destination-connectors/${resCSVDst.json().destination_connector.id}`), {
            [`GET /v1alpha/admin/destination-connectors/${resCSVDst.json().destination_connector.id} response status 200`]: (r) => r.status === 200,
            [`GET /v1alpha/admin/destination-connectors/${resCSVDst.json().destination_connector.id} response connector id`]: (r) => r.json().destination_connector.id === csvDstConnector.id,
            [`GET /v1alpha/admin/destination-connectors/${resCSVDst.json().destination_connector.id} response connector destination_connector_definition permalink`]: (r) => r.json().destination_connector.destination_connector_definition === constant.csvDstDefRscName,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}`), {
            [`DELETE /v1alpha/admin/destination-connectors/${resCSVDst.json().destination_connector.id} response status 204`]: (r) => r.status === 204,
        });
    });
}

export function CheckLookUp() {

    group("Connector API: Look up destination connectors by UID by admin", () => {

        var csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": constant.csvDstConfig
            }
        }

        var resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors`,
            JSON.stringify(csvDstConnector), constant.params)

        check(http.request("GET", `${connectorPrivateHost}/v1alpha/admin/destination-connectors/${resCSVDst.json().destination_connector.uid}/lookUp`), {
            [`GET /v1alpha/admin/destination-connectors/${resCSVDst.json().destination_connector.uid}/lookUp response status 200`]: (r) => r.status === 200,
            [`GET /v1alpha/admin/destination-connectors/${resCSVDst.json().destination_connector.uid}/lookUp response connector uid`]: (r) => r.json().destination_connector.uid === resCSVDst.json().destination_connector.uid,
            [`GET /v1alpha/admin/destination-connectors/${resCSVDst.json().destination_connector.uid}/lookUp response connector destination_connector_definition`]: (r) => r.json().destination_connector.destination_connector_definition === constant.csvDstDefRscName,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}`), {
            [`DELETE /v1alpha/admin/destination-connectors/${resCSVDst.json().destination_connector.id} response status 204`]: (r) => r.status === 204,
        });

    });
}
