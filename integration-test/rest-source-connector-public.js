import http from "k6/http";
import { sleep, check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import { connectorPublicHost, modelPublicHost, pipelinePublicHost } from "./const.js"

import * as constant from "./const.js"
import * as helper from "./helper.js"

export function CheckCreate() {

    group("Connector API: Create source connectors", () => {

        var srcConnector = {
            "id": "trigger",
            "connector_definition_name": constant.srcDefRscName,
            "description": "HTTP source",
            "configuration": {},
        }


        var resSrcHTTP = http.request(
            "POST",
            `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(srcConnector), constant.params)

        check(resSrcHTTP, {
            "POST /v1alpha/connectors response status for creating HTTP source connector 201": (r) => r.status === 201,
            "POST /v1alpha/connectors response connector name": (r) => r.json().connector.name == `connectors/${srcConnector.id}`,
            "POST /v1alpha/connectors response connector uid": (r) => helper.isUUID(r.json().connector.uid),
            "POST /v1alpha/connectors response connector connector_definition_name": (r) => r.json().connector.connector_definition_name === constant.srcDefRscName,
            "POST /v1alpha/connectors response connector owner is UUID": (r) => helper.isValidOwner(r.json().connector.user),
        });

        check(http.request(
            "POST",
            `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(srcConnector), constant.params), {
            "POST /v1alpha/connectors response duplicate HTTP source connector status 409": (r) => r.status === 409
        });


        check(http.request(
            "POST",
            `${connectorPublicHost}/v1alpha/connectors`,
            {}, constant.params), {
            "POST /v1alpha/connectors response status for creating empty body 400": (r) => r.status === 400,
        });

        // Delete test records
        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resSrcHTTP.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resSrcHTTP.json().connector.id} response status 204`]: (r) => r.status === 204,
        });
    });
}

export function CheckList() {

    group("Connector API: List source connectors", () => {

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors?filter=connector_type=CONNECTOR_TYPE_SOURCE`), {
            [`GET /v1alpha/connectors response status is 200`]: (r) => r.status === 200,
            [`GET /v1alpha/connectors response connectors array is 0 length`]: (r) => r.json().connectors.length === 0,
            [`GET /v1alpha/connectors response next_page_token is empty`]: (r) => r.json().next_page_token === "",
            [`GET /v1alpha/connectors response total_size is 0`]: (r) => r.json().next_page_token == 0,
        });

        var reqBodies = [];
        reqBodies[0] = {
            "id": "trigger",
            "connector_definition_name": constant.srcDefRscName,
            "configuration": {}
        }

        // Create connectors
        for (const reqBody of reqBodies) {
            check(http.request(
                "POST",
                `${connectorPublicHost}/v1alpha/connectors`,
                JSON.stringify(reqBody), constant.params), {
                [`POST /v1alpha/connectors x${reqBodies.length} response status 201`]: (r) => r.status === 201,
            });
        }

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors?filter=connector_type=CONNECTOR_TYPE_SOURCE`), {
            [`GET /v1alpha/connectors response status is 200`]: (r) => r.status === 200,
            [`GET /v1alpha/connectors response has connectors array`]: (r) => Array.isArray(r.json().connectors),
            [`GET /v1alpha/connectors response has total_size = ${reqBodies.length}`]: (r) => r.json().total_size == reqBodies.length,
        });

        var limitedRecords = http.request("GET", `${connectorPublicHost}/v1alpha/connectors?filter=connector_type=CONNECTOR_TYPE_SOURCE`)
        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors?filter=connector_type=CONNECTOR_TYPE_SOURCE&page_size=0`), {
            "GET /v1alpha/connectors?page_size=0 response status is 200": (r) => r.status === 200,
            "GET /v1alpha/connectors?page_size=0 response all records": (r) => r.json().connectors.length === limitedRecords.json().connectors.length,
        });

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors?filter=connector_type=CONNECTOR_TYPE_SOURCE&page_size=1`), {
            "GET /v1alpha/connectors?page_size=1 response status is 200": (r) => r.status === 200,
            "GET /v1alpha/connectors?page_size=1 response connectors size 1": (r) => r.json().connectors.length === 1,
        });

        var pageRes = http.request("GET", `${connectorPublicHost}/v1alpha/connectors?filter=connector_type=CONNECTOR_TYPE_SOURCE&page_size=1`)
        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors?page_size=1&page_token=${pageRes.json().next_page_token}`), {
            [`GET /v1alpha/connectors?page_size=1&page_token=${pageRes.json().next_page_token} response status is 200`]: (r) => r.status === 200,
            [`GET /v1alpha/connectors?page_size=1&page_token=${pageRes.json().next_page_token} response connectors size 1`]: (r) => r.json().connectors.length === 1,
        });

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors?filter=connector_type=CONNECTOR_TYPE_SOURCE&page_size=1&view=VIEW_BASIC`), {
            "GET /v1alpha/connectors?page_size=1&view=VIEW_BASIC response status 200": (r) => r.status === 200,
            "GET /v1alpha/connectors?page_size=1&view=VIEW_BASIC response connectors[0]connector.configuration is null": (r) => r.json().connectors[0].configuration === null,
            "GET /v1alpha/connectors?page_size=1&view=VIEW_BASIC response connectors[0]connector.owner is UUID": (r) => helper.isValidOwner(r.json().connectors[0].user),
        });

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors?filter=connector_type=CONNECTOR_TYPE_SOURCE&page_size=1&view=VIEW_FULL`), {
            "GET /v1alpha/connectors?page_size=1&view=VIEW_FULL response status 200": (r) => r.status === 200,
            "GET /v1alpha/connectors?page_size=1&view=VIEW_FULL response connectors[0]connector.configuration is not null": (r) => r.json().connectors[0].configuration !== null,
            "GET /v1alpha/connectors?page_size=1&view=VIEW_FULL response connectors[0]connector.connector_definition_detail is not null": (r) => r.json().connectors[0].connector_definition_detail !== null,
            "GET /v1alpha/connectors?page_size=1&view=VIEW_FULL response connectors[0]connector.configuration is {}": (r) => Object.keys(r.json().connectors[0].configuration).length === 0,
            "GET /v1alpha/connectors?page_size=1&view=VIEW_FULL response connectors[0]connector.owner is UUID": (r) => helper.isValidOwner(r.json().connectors[0].user),
        });

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors?filter=connector_type=CONNECTOR_TYPE_SOURCE&page_size=1`), {
            "GET /v1alpha/connectors?page_size=1 response status 200": (r) => r.status === 200,
            "GET /v1alpha/connectors?page_size=1 response connectors[0]connector.configuration is null": (r) => r.json().connectors[0].configuration === null,
            "GET /v1alpha/connectors?page_size=1&view=VIEW_BASIC response connectors[0]connector.owner is UUID": (r) => helper.isValidOwner(r.json().connectors[0].user),
        });

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors?filter=connector_type=CONNECTOR_TYPE_SOURCE&page_size=${limitedRecords.json().total_size}`), {
            [`GET /v1alpha/connectors?page_size=${limitedRecords.json().total_size} response status 200`]: (r) => r.status === 200,
            [`GET /v1alpha/connectors?page_size=${limitedRecords.json().total_size} response next_page_token is empty`]: (r) => r.json().next_page_token === ""
        });

        // Delete the destination connectors
        for (const reqBody of reqBodies) {
            check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${reqBody.id}`), {
                [`DELETE /v1alpha/connectors x${reqBodies.length} response status is 204`]: (r) => r.status === 204,
            });
        }
    });
}

export function CheckGet() {

    group("Connector API: Get source connectors by ID", () => {

        var srcConnector = {
            "id": "trigger",
            "connector_definition_name": constant.srcDefRscName,
            "configuration": {}
        }

        var resHTTP = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(srcConnector), constant.params)

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors/${resHTTP.json().connector.id}`), {
            [`GET /v1alpha/connectors/${resHTTP.json().connector.id} response status 200`]: (r) => r.status === 200,
            [`GET /v1alpha/connectors/${resHTTP.json().connector.id} response connector id`]: (r) => r.json().connector.id === srcConnector.id,
            [`GET /v1alpha/connectors/${resHTTP.json().connector.id} response connector connector_definition_name`]: (r) => r.json().connector.connector_definition_name === constant.srcDefRscName,
            [`GET /v1alpha/connectors/${resHTTP.json().connector.id} response connector owner is UUID`]: (r) => helper.isValidOwner(r.json().connector.user),
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resHTTP.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resHTTP.json().connector.id} response status 204`]: (r) => r.status === 204,
        });

    });
}

export function CheckUpdate() {

    group("Connector API: Update source connectors", () => {

        var srcConnector = {
            "id": "trigger",
            "connector_definition_name": constant.srcDefRscName,
            "configuration": {}
        }

        check(http.request(
            "POST",
            `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(srcConnector), constant.params), {
            "POST /v1alpha/connectors response status for creating gRPC source connector 201": (r) => r.status === 201,
        });

        http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${srcConnector.id}/connect`,
            {}, constant.params)

        srcConnector.description = randomString(20)

        check(http.request(
            "PATCH",
            `${connectorPublicHost}/v1alpha/connectors/${srcConnector.id}`,
            JSON.stringify(srcConnector), constant.params), {
            [`PATCH /v1alpha/connectors/${srcConnector.id} response status for updating gRPC source connector 422`]: (r) => r.status === 422,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${srcConnector.id}`), {
            [`DELETE /v1alpha/connectors/${srcConnector.id} response status 204`]: (r) => r.status === 204,
        });

    });

}

export function CheckDelete() {

    group("Connector API: Delete source connectors", () => {

        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify({
                "id": "trigger",
                "connector_definition_name": "connector-definitions/trigger",
                "configuration": {}
            }), constant.params), {
            "POST /v1alpha/connectors response status for creating HTTP source connector 201": (r) => r.status === 201,
        })

        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify({
                "id": "response",
                "connector_definition_name": "connector-definitions/response",
                "configuration": {}
            }), constant.params), {
            "POST /v1alpha/connectors response status for creating HTTP destination connector 201": (r) => r.status === 201,
        })

        let createClsModelRes = http.request("POST", `${modelPublicHost}/v1alpha/models`, JSON.stringify({
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
            var res = http.get(`${modelPublicHost}/v1alpha/${createClsModelRes.json().operation.name}`, {
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
                    { "id": "s01", "resource_name": "connectors/trigger" },
                    { "id": "m01", "resource_name": "models/dummy-cls" },
                    { "id": "d01", "resource_name": "connectors/response" },
                ]
            },
        };

        // Create a pipeline
        const pipelineID = randomString(5)
        check(http.request("POST", `${pipelinePublicHost}/v1alpha/pipelines`,
            JSON.stringify(Object.assign({
                id: pipelineID,
                description: randomString(10),
            },
                detSyncRecipe
            )), constant.params), {
            "POST /v1alpha/pipelines response status is 201": (r) => r.status === 201,
        })

        // Cannot delete source connector due to pipeline occupancy
        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/trigger`), {
            [`DELETE /v1alpha/connectors/trigger response status 422`]: (r) => r.status === 422,
            [`DELETE /v1alpha/connectors/trigger response error msg not nil`]: (r) => r.json() != {},
        });

        // Cannot delete destination connector due to pipeline occupancy
        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/response`), {
            [`DELETE /v1alpha/connectors/response response status 422`]: (r) => r.status === 422,
            [`DELETE /v1alpha/connectors/trigger response error msg not nil`]: (r) => r.json() != {},
        });

        check(http.request("DELETE", `${pipelinePublicHost}/v1alpha/pipelines/${pipelineID}`), {
            [`DELETE /v1alpha/pipelines/${pipelineID} response status is 204`]: (r) => r.status === 204,
        });

        // Can delete source connector now
        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/trigger`), {
            [`DELETE /v1alpha/connectors/trigger response status 204`]: (r) => r.status === 204,
        });

        // Can delete destination connector now
        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/response`), {
            [`DELETE /v1alpha/connectors/response response status 204`]: (r) => r.status === 204,
        });

        // Wait for model state to be updated
        currentTime = new Date().getTime();
        timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            let res = http.get(`${modelPublicHost}/v1alpha/models/dummy-cls/watch`, {
                headers: helper.genHeader(`application/json`),
            })
            if (res.json().state !== "STATE_UNSPECIFIED") {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        // Can delete model now
        check(http.request("DELETE", `${modelPublicHost}/v1alpha/models/dummy-cls`), {
            [`DELETE /v1alpha/models/dummy-cls response status is 204`]: (r) => r.status === 204,
        });

    });
}

export function CheckLookUp() {

    group("Connector API: Look up source connectors by UID", () => {

        var srcConnector = {
            "id": "trigger",
            "connector_definition_name": constant.srcDefRscName,
            "configuration": {}
        }

        var resHTTP = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(srcConnector), constant.params)

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors/${resHTTP.json().connector.uid}/lookUp`), {
            [`GET /v1alpha/connectors/${resHTTP.json().connector.uid}/lookUp response status 200`]: (r) => r.status === 200,
            [`GET /v1alpha/connectors/${resHTTP.json().connector.uid}/lookUp response connector uid`]: (r) => r.json().connector.uid === resHTTP.json().connector.uid,
            [`GET /v1alpha/connectors/${resHTTP.json().connector.uid}/lookUp response connector connector_definition_name`]: (r) => r.json().connector.connector_definition_name === constant.srcDefRscName,
            [`GET /v1alpha/connectors/${resHTTP.json().connector.uid}/lookUp response connector owner is UUID`]: (r) => helper.isValidOwner(r.json().connector.user),
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resHTTP.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resHTTP.json().connector.id} response status 204`]: (r) => r.status === 204,
        });

    });
}

export function CheckState() {

    group("Connector API: Change state source connectors", () => {
        var srcConnector = {
            "id": "trigger",
            "connector_definition_name": constant.srcDefRscName,
            "configuration": {}
        }

        var resHTTP = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(srcConnector), constant.params)

        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resHTTP.json().connector.id}/connect`, null, constant.params), {
            [`POST /v1alpha/connectors/${resHTTP.json().connector.id}/connect response status 200`]: (r) => r.status === 200,
        });

        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resHTTP.json().connector.id}/disconnect`, null, constant.params), {
            [`POST /v1alpha/connectors/${resHTTP.json().connector.id}/disconnect response status 422`]: (r) => r.status === 422,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resHTTP.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resHTTP.json().connector.id} response status 204`]: (r) => r.status === 204,
        });

    });

}

export function CheckRename() {

    group("Connector API: Rename source connectors", () => {
        var srcConnector = {
            "id": "trigger",
            "connector_definition_name": constant.srcDefRscName,
            "configuration": {}
        }

        var resHTTP = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(srcConnector), constant.params)

        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resHTTP.json().connector.id}/rename`,
            JSON.stringify({
                "new_connector_id": "some-id-not-http"
            }), constant.params), {
            [`POST /v1alpha/connectors/${resHTTP.json().connector.id}/rename response status 422`]: (r) => r.status === 422,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resHTTP.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resHTTP.json().connector.id} response status 204`]: (r) => r.status === 204,
        });
    });

}

export function CheckTest() {

    group("Connector API: Test source connectors by ID", () => {

        var srcConnector = {
            "id": "trigger",
            "connector_definition_name": constant.srcDefRscName,
            "configuration": {}
        }

        var resHTTP = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(srcConnector), constant.params)

        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resHTTP.json().connector.id}/testConnection`), {
            [`POST /v1alpha/connectors/${resHTTP.json().connector.id}/testConnection response status 200`]: (r) => r.status === 200,
            [`POST /v1alpha/connectors/${resHTTP.json().connector.id}/testConnection response connector STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resHTTP.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resHTTP.json().connector.id} response status 204`]: (r) => r.status === 204,
        });

    });
}
