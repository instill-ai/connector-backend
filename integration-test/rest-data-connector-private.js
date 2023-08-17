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
import * as helper from "./helper.js"

export function CheckList() {

    group("Connector API: List destination connectors by admin", () => {

        check(http.request("GET", `${connectorPrivateHost}/v1alpha/admin/connector-resources?filter=connector_type=CONNECTOR_TYPE_DATA`), {
            [`GET /v1alpha/admin/connector-resources response status is 200`]: (r) => r.status === 200,
            [`GET /v1alpha/admin/connector-resources response connectors array is 0 length`]: (r) => r.json().connector_resources.length === 0,
            [`GET /v1alpha/admin/connector-resources response next_page_token is empty`]: (r) => r.json().next_page_token === "",
            [`GET /v1alpha/admin/connector-resources response total_size is 0`]: (r) => r.json().total_size == 0,
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
            var resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/connector-resources`,
                JSON.stringify(reqBody), constant.params)
            check(resCSVDst, {
                [`POST /v1alpha/connector-resources x${reqBodies.length} response status 201`]: (r) => r.status === 201,
            });
        }

        check(http.request("GET", `${connectorPrivateHost}/v1alpha/admin/connector-resources?filter=connector_type=CONNECTOR_TYPE_DATA`), {
            [`GET /v1alpha/admin/connector-resources response status is 200`]: (r) => r.status === 200,
            [`GET /v1alpha/admin/connector-resources response has connectors array`]: (r) => Array.isArray(r.json().connector_resources),
            [`GET /v1alpha/admin/connector-resources response has total_size = ${numConnectors}`]: (r) => r.json().total_size == numConnectors,
        });

        var limitedRecords = http.request("GET", `${connectorPrivateHost}/v1alpha/admin/connector-resources?filter=connector_type=CONNECTOR_TYPE_DATA`)
        check(http.request("GET", `${connectorPrivateHost}/v1alpha/admin/connector-resources?page_size=0`), {
            "GET /v1alpha/admin/connector-resources?page_size=0 response status is 200": (r) => r.status === 200,
            "GET /v1alpha/admin/connector-resources?page_size=0 response all records": (r) => r.json().connector_resources.length === limitedRecords.json().connector_resources.length,
        });

        check(http.request("GET", `${connectorPrivateHost}/v1alpha/admin/connector-resources?filter=connector_type=CONNECTOR_TYPE_DATA&page_size=1`), {
            "GET /v1alpha/admin/connector-resources?page_size=1 response status is 200": (r) => r.status === 200,
            "GET /v1alpha/admin/connector-resources?page_size=1 response connectors size 1": (r) => r.json().connector_resources.length === 1,
        });

        var pageRes = http.request("GET", `${connectorPrivateHost}/v1alpha/admin/connector-resources?filter=connector_type=CONNECTOR_TYPE_DATA&page_size=1`)
        check(http.request("GET", `${connectorPrivateHost}/v1alpha/admin/connector-resources?page_size=1&page_token=${pageRes.json().next_page_token}`), {
            [`GET /v1alpha/admin/connector-resources?page_size=1&page_token=${pageRes.json().next_page_token} response status is 200`]: (r) => r.status === 200,
            [`GET /v1alpha/admin/connector-resources?page_size=1&page_token=${pageRes.json().next_page_token} response connectors size 1`]: (r) => r.json().connector_resources.length === 1,
        });

        check(http.request("GET", `${connectorPrivateHost}/v1alpha/admin/connector-resources?filter=connector_type=CONNECTOR_TYPE_DATA&page_size=1&view=VIEW_BASIC`), {
            "GET /v1alpha/admin/connector-resources?page_size=1&view=VIEW_BASIC response status 200": (r) => r.status === 200,
            "GET /v1alpha/admin/connector-resources?page_size=1&view=VIEW_BASIC response connectors[0].configuration is null": (r) => r.json().connector_resources[0].configuration === null,
            "GET /v1alpha/admin/connector-resources?page_size=1&view=VIEW_BASIC response connectors[0].owner is UUID": (r) => helper.isValidOwner(r.json().connector_resources[0].user ),
        });

        check(http.request("GET", `${connectorPrivateHost}/v1alpha/admin/connector-resources?filter=connector_type=CONNECTOR_TYPE_DATA&page_size=1&view=VIEW_FULL`), {
            "GET /v1alpha/admin/connector-resources?page_size=1&view=VIEW_FULL response status 200": (r) => r.status === 200,
            "GET /v1alpha/admin/connector-resources?page_size=1&view=VIEW_FULL response connectors[0].configuration is not null": (r) => r.json().connector_resources[0].configuration !== null,
            "GET /v1alpha/admin/connector-resources?page_size=1&view=VIEW_FULL response connectors[0].connector_definition_detail is not null": (r) => r.json().connector_resources[0].connector_definition_detail !== null,
            "GET /v1alpha/admin/connector-resources?page_size=1&view=VIEW_FULL response connectors[0].owner is UUID": (r) => helper.isValidOwner(r.json().connector_resources[0].user ),
        });

        check(http.request("GET", `${connectorPrivateHost}/v1alpha/admin/connector-resources?filter=connector_type=CONNECTOR_TYPE_DATA&page_size=1`), {
            "GET /v1alpha/admin/connector-resources?page_size=1 response status 200": (r) => r.status === 200,
            "GET /v1alpha/admin/connector-resources?page_size=1 response connectors[0].configuration is null": (r) => r.json().connector_resources[0].configuration === null,
            "GET /v1alpha/admin/connector-resources?page_size=1 response connectors[0].owner is UUID": (r) => helper.isValidOwner(r.json().connector_resources[0].user ),
        });

        check(http.request("GET", `${connectorPrivateHost}/v1alpha/admin/connector-resources?filter=connector_type=CONNECTOR_TYPE_DATA&page_size=${limitedRecords.json().total_size}`), {
            [`GET /v1alpha/admin/connector-resources?page_size=${limitedRecords.json().total_size} response status 200`]: (r) => r.status === 200,
            [`GET /v1alpha/admin/connector-resources?page_size=${limitedRecords.json().total_size} response next_page_token is empty`]: (r) => r.json().next_page_token === ""
        });

        // Delete the destination connectors
        for (const reqBody of reqBodies) {
            check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connector-resources/${reqBody.id}`), {
                [`DELETE /v1alpha/admin/connector-resources x${reqBodies.length} response status is 204`]: (r) => r.status === 204,
            });
        }
    });
}

export function CheckLookUp() {

    group("Connector API: Look up destination connectors by UID by admin", () => {

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/connector-resources`,
            JSON.stringify(csvDstConnector), constant.params)

        check(http.request("GET", `${connectorPrivateHost}/v1alpha/admin/connector-resources/${resCSVDst.json().connector_resource.uid}/lookUp`), {
            [`GET /v1alpha/admin/connector-resources/${resCSVDst.json().connector_resource.uid}/lookUp response status 200`]: (r) => r.status === 200,
            [`GET /v1alpha/admin/connector-resources/${resCSVDst.json().connector_resource.uid}/lookUp response connector uid`]: (r) => r.json().connector_resource.uid === resCSVDst.json().connector_resource.uid,
            [`GET /v1alpha/admin/connector-resources/${resCSVDst.json().connector_resource.uid}/lookUp response connector connector_definition_name`]: (r) => r.json().connector_resource.connector_definition_name === constant.csvDstDefRscName,
            [`GET /v1alpha/admin/connector-resources/${resCSVDst.json().connector_resource.uid}/lookUp response connector owner is UUID`]: (r) => helper.isValidOwner(r.json().connector_resource.user),
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connector-resources/${resCSVDst.json().connector_resource.id}`), {
            [`DELETE /v1alpha/admin/connector-resources/${resCSVDst.json().connector_resource.id} response status 204`]: (r) => r.status === 204,
        });

    });
}
