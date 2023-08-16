import http from "k6/http";
import { check, group, sleep } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import { connectorPublicHost } from "./const.js"

import * as constant from "./const.js"
import * as helper from "./helper.js"

export function CheckCreate() {

    group("Connector API: Create destination connectors", () => {


        // destination-csv
        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig

        }

        var resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(csvDstConnector), {
            headers: { "Content-Type": "application/json" },
        })

        http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${csvDstConnector.id}/connect`,
            {}, constant.params)

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/watch`), {
            [`GET /v1alpha/connectors/${resCSVDst.json().connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(resCSVDst, {
            "POST /v1alpha/connectors response status 201": (r) => r.status === 201,
        });

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}`), {
            [`GET /v1alpha/connectors/${resCSVDst.json().connector.id} response STATE_CONNECTED`]: (r) => r.json().connector.state === "STATE_CONNECTED",
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

        var resDstMySQL = http.request(
            "POST",
            `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(mySQLDstConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(resDstMySQL, {
            "POST /v1alpha/connectors response status for creating MySQL destination connector 201": (r) => r.status === 201,
            "POST /v1alpha/connectors response connector name": (r) => r.json().connector.name == `connectors/${mySQLDstConnector.id}`,
            "POST /v1alpha/connectors response connector uid": (r) => helper.isUUID(r.json().connector.uid),
            "POST /v1alpha/connectors response connector connector_definition_name": (r) => r.json().connector.connector_definition_name === constant.mySQLDstDefRscName
        });


        // Delete test records
        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resCSVDst.json().connector.id} response status 204`]: (r) => r.status === 204,
        });
        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resDstMySQL.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resDstMySQL.json().connector.id} response status 204`]: (r) => r.status === 204,
        });
    });

}

export function CheckList() {

    group("Connector API: List destination connectors", () => {

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors?filter=connector_type=CONNECTOR_TYPE_DATA`), {
            [`GET /v1alpha/connectors response status is 200`]: (r) => r.status === 200,
            [`GET /v1alpha/connectors response connectors array is 0 length`]: (r) => r.json().connectors.length === 0,
            [`GET /v1alpha/connectors response next_page_token is empty`]: (r) => r.json().next_page_token === "",
            [`GET /v1alpha/connectors response total_size is 0`]: (r) => r.json().total_size == 0,
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
            var resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
                JSON.stringify(reqBody), {
                headers: { "Content-Type": "application/json" },
            })
            check(resCSVDst, {
                [`POST /v1alpha/connectors x${reqBodies.length} response status 201`]: (r) => r.status === 201,
            });
        }

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors?filter=connector_type=CONNECTOR_TYPE_DATA`), {
            [`GET /v1alpha/connectors response status is 200`]: (r) => r.status === 200,
            [`GET /v1alpha/connectors response has connectors array`]: (r) => Array.isArray(r.json().connectors),
            [`GET /v1alpha/connectors response has total_size = ${numConnectors}`]: (r) => r.json().total_size == numConnectors,
        });

        var limitedRecords = http.request("GET", `${connectorPublicHost}/v1alpha/connectors?filter=connector_type=CONNECTOR_TYPE_DATA`)
        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors?filter=connector_type=CONNECTOR_TYPE_DATA&page_size=0`), {
            "GET /v1alpha/connectors?page_size=0 response status is 200": (r) => r.status === 200,
            "GET /v1alpha/connectors?page_size=0 response all records": (r) => r.json().connectors.length === limitedRecords.json().connectors.length,
        });

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors?filter=connector_type=CONNECTOR_TYPE_DATA&page_size=1`), {
            "GET /v1alpha/connectors?page_size=1 response status is 200": (r) => r.status === 200,
            "GET /v1alpha/connectors?page_size=1 response connectors size 1": (r) => r.json().connectors.length === 1,
        });

        var pageRes = http.request("GET", `${connectorPublicHost}/v1alpha/connectors?filter=connector_type=CONNECTOR_TYPE_DATA&page_size=1`)
        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors?filter=connector_type=CONNECTOR_TYPE_DATA&page_size=1&page_token=${pageRes.json().next_page_token}`), {
            [`GET /v1alpha/connectors?page_size=1&page_token=${pageRes.json().next_page_token} response status is 200`]: (r) => r.status === 200,
            [`GET /v1alpha/connectors?page_size=1&page_token=${pageRes.json().next_page_token} response connectors size 1`]: (r) => r.json().connectors.length === 1,
        });

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors?filter=connector_type=CONNECTOR_TYPE_DATA&page_size=1&view=VIEW_BASIC`), {
            "GET /v1alpha/connectors?page_size=1&view=VIEW_BASIC response status 200": (r) => r.status === 200,
            "GET /v1alpha/connectors?page_size=1&view=VIEW_BASIC response connectors[0].configuration is null": (r) => r.json().connectors[0].configuration === null,
        });

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors?filter=connector_type=CONNECTOR_TYPE_DATA&page_size=1&view=VIEW_FULL`), {
            "GET /v1alpha/connectors?page_size=1&view=VIEW_FULL response status 200": (r) => r.status === 200,
            "GET /v1alpha/connectors?page_size=1&view=VIEW_FULL response connectors[0].configuration is not null": (r) => r.json().connectors[0].configuration !== null,
            "GET /v1alpha/connectors?page_size=1&view=VIEW_FULL response connectors[0].connector_definition_detail is not null": (r) => r.json().connectors[0].connector_definition_detail !== null,
        });

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors?filter=connector_type=CONNECTOR_TYPE_DATA&page_size=1`), {
            "GET /v1alpha/connectors?page_size=1 response status 200": (r) => r.status === 200,
            "GET /v1alpha/connectors?page_size=1 response connectors[0].configuration is null": (r) => r.json().connectors[0].configuration === null,
        });

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors?filter=connector_type=CONNECTOR_TYPE_DATA&page_size=${limitedRecords.json().total_size}`), {
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

    group("Connector API: Get destination connectors by ID", () => {

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(csvDstConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/watch`), {
            [`GET /v1alpha/connectors/${resCSVDst.json().connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}`), {
            [`GET /v1alpha/connectors/${resCSVDst.json().connector.id} response status 200`]: (r) => r.status === 200,
            [`GET /v1alpha/connectors/${resCSVDst.json().connector.id} response connector id`]: (r) => r.json().connector.id === csvDstConnector.id,
            [`GET /v1alpha/connectors/${resCSVDst.json().connector.id} response connector connector_definition_name permalink`]: (r) => r.json().connector.connector_definition_name === constant.csvDstDefRscName,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resCSVDst.json().connector.id} response status 204`]: (r) => r.status === 204,
        });
    });
}

export function CheckUpdate() {

    group("Connector API: Update destination connectors", () => {

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(csvDstConnector), {
            headers: { "Content-Type": "application/json" },
        })

        var csvDstConnectorUpdate = {
            "id": csvDstConnector.id,
            "connector_definition_name": csvDstConnector.connector_definition_name,
            "tombstone": true,
            "description": randomString(50),
            "configuration": {
                destination_path: "/tmp"
            }
        }

        var resCSVDstUpdate = http.request("PATCH", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}`,
            JSON.stringify(csvDstConnectorUpdate), {
            headers: { "Content-Type": "application/json" },
        })

        check(resCSVDstUpdate, {
            [`PATCH /v1alpha/connectors/${resCSVDst.json().connector.id} response status 200`]: (r) => r.status === 200,
            [`PATCH /v1alpha/connectors/${resCSVDst.json().connector.id} response connector id`]: (r) => r.json().connector.id === csvDstConnectorUpdate.id,
            [`PATCH /v1alpha/connectors/${resCSVDst.json().connector.id} response connector connector_definition_name`]: (r) => r.json().connector.connector_definition_name === constant.csvDstDefRscName,
            [`PATCH /v1alpha/connectors/${resCSVDst.json().connector.id} response connector description`]: (r) => r.json().connector.description === csvDstConnectorUpdate.connector.description,
            [`PATCH /v1alpha/connectors/${resCSVDst.json().connector.id} response connector tombstone`]: (r) => r.json().connector.tombstone === false,
            [`PATCH /v1alpha/connectors/${resCSVDst.json().connector.id} response connector configuration`]: (r) => r.json().connector.configuration.destination_path === csvDstConnectorUpdate.connector.configuration.destination_path
        });

        // Try to update with empty description
        csvDstConnectorUpdate = {
            "configuration": {}
        }

        resCSVDstUpdate = http.request("PATCH", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}`,
            JSON.stringify(csvDstConnectorUpdate), {
            headers: { "Content-Type": "application/json" },
        })

        check(resCSVDstUpdate, {
            [`PATCH /v1alpha/connectors/${resCSVDst.json().connector.id} with empty description response status 200`]: (r) => r.status === 200,
            [`PATCH /v1alpha/connectors/${resCSVDst.json().connector.id} with empty description response empty description`]: (r) => r.json().connector.description === csvDstConnectorUpdate.connector.description,
        })

        // Try to update with a non-existing name field (which should be ignored because name field is OUTPUT_ONLY)
        csvDstConnectorUpdate = {
            "name": `connectors/${randomString(5)}`,
            "description": randomString(50),
        }

        resCSVDstUpdate = http.request("PATCH", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}`,
            JSON.stringify(csvDstConnectorUpdate), {
            headers: { "Content-Type": "application/json" },
        })

        check(resCSVDstUpdate, {
            [`PATCH /v1alpha/connectors/${resCSVDst.json().connector.id} with non-existing name field response status 200`]: (r) => r.status === 200,
        })

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${csvDstConnector.id}`), {
            [`DELETE /v1alpha/connectors/${csvDstConnector.id} response status 204`]: (r) => r.status === 204,
        });
    });
}

export function CheckLookUp() {

    group("Connector API: Look up destination connectors by UID", () => {

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(csvDstConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.uid}/lookUp`), {
            [`GET /v1alpha/connectors/${resCSVDst.json().connector.uid}/lookUp response status 200`]: (r) => r.status === 200,
            [`GET /v1alpha/connectors/${resCSVDst.json().connector.uid}/lookUp response connector uid`]: (r) => r.json().connector.uid === resCSVDst.json().connector.uid,
            [`GET /v1alpha/connectors/${resCSVDst.json().connector.uid}/lookUp response connector connector_definition_name`]: (r) => r.json().connector.connector_definition_name === constant.csvDstDefRscName,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resCSVDst.json().connector.id} response status 204`]: (r) => r.status === 204,
        });

    });
}

export function CheckState() {

    group("Connector API: Change state destination connectors", () => {
        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(csvDstConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/disconnect`, null, {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/connectors/${resCSVDst.json().connector.id}/disconnect response at UNSPECIFIED state status 200`]: (r) => r.status === 200,
        });

        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/connect`, null, {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/connectors/${resCSVDst.json().connector.id}/connect response at UNSPECIFIED state status 200`]: (r) => r.status === 200,
        });

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/watch`), {
            [`GET /v1alpha/connectors/${resCSVDst.json().connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/connect`, null, {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/connectors/${resCSVDst.json().connector.id}/connect response status 200 (with STATE_CONNECTED)`]: (r) => r.status === 200,
        });

        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/disconnect`, null, {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/connectors/${resCSVDst.json().connector.id}/disconnect response status 200 (with STATE_CONNECTED)`]: (r) => r.status === 200,
        });

        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/disconnect`, null, {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/connectors/${resCSVDst.json().connector.id}/disconnect response status 200 (with STATE_DISCONNECTED)`]: (r) => r.status === 200,
        });

        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/connect`, null, {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/connectors/${resCSVDst.json().connector.id}/connect response status 200 (with STATE_DISCONNECTED)`]: (r) => r.status === 200,
        });

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/watch`), {
            [`GET /v1alpha/connectors/${resCSVDst.json().connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

    });

}

export function CheckRename() {

    group("Connector API: Rename destination connectors", () => {

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(csvDstConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/rename`,
            JSON.stringify({
                "new_connector_id": `some-id-not-${resCSVDst.json().connector.id}`
            }), {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/connectors/${resCSVDst.json().connector.id}/rename response status 200`]: (r) => r.status === 200,
            [`POST /v1alpha/connectors/${resCSVDst.json().connector.id}/rename response id is some-id-not-${resCSVDst.json().connector.id}`]: (r) => r.json().connector.id === `some-id-not-${resCSVDst.json().connector.id}`,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/some-id-not-${resCSVDst.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/some-id-not-${resCSVDst.json().connector.id} response status 204`]: (r) => r.status === 204,
        });
    });
}

export function CheckExecute() {

    group("Connector API: Write destination connectors", () => {

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

        resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(csvDstConnector), {
            headers: { "Content-Type": "application/json" },
        })
        http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${csvDstConnector.id}/connect`, {})

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/watch`), {
            [`GET /v1alpha/connectors/${resCSVDst.json().connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/execute`,
            JSON.stringify({
                "inputs": constant.clsModelOutputs
            }), {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/connectors/${resCSVDst.json().connector.id}/execute response status 200 (classification)`]: (r) => r.status === 200,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resCSVDst.json().connector.id} response status 204 (classification)`]: (r) => r.status === 204,
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

        resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(csvDstConnector), {
            headers: { "Content-Type": "application/json" },
        })
        http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${csvDstConnector.id}/connect`, {})

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/watch`), {
            [`GET /v1alpha/connectors/${resCSVDst.json().connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/execute`,
            JSON.stringify({
                "inputs": constant.detectionEmptyModelOutputs
            }), {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/connectors/${resCSVDst.json().connector.id}/execute response status 200 (detection)`]: (r) => r.status === 200,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resCSVDst.json().connector.id} response status 204 (detection)`]: (r) => r.status === 204,
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

        resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(csvDstConnector), {
            headers: { "Content-Type": "application/json" },
        })
        http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${csvDstConnector.id}/connect`, {})

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/watch`), {
            [`GET /v1alpha/connectors/${resCSVDst.json().connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/execute`,
            JSON.stringify({
                "inputs": constant.detectionModelOutputs
            }), {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/connectors/${resCSVDst.json().connector.id}/execute response status 200 (detection)`]: (r) => r.status === 200,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resCSVDst.json().connector.id} response status 204 (detection)`]: (r) => r.status === 204,
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

        resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(csvDstConnector), {
            headers: { "Content-Type": "application/json" },
        })
        http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${csvDstConnector.id}/connect`, {})

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/watch`), {
            [`GET /v1alpha/connectors/${resCSVDst.json().connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/execute`,
            JSON.stringify({
                "inputs": constant.keypointModelOutputs
            }), {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/connectors/${resCSVDst.json().connector.id}/execute response status 200 (keypoint)`]: (r) => r.status === 200,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resCSVDst.json().connector.id} response status 204 (keypoint)`]: (r) => r.status === 204,
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

        resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(csvDstConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/watch`), {
            [`GET /v1alpha/connectors/${resCSVDst.json().connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/execute`,
            JSON.stringify({
                "inputs": constant.ocrModelOutputs
            }), {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/connectors/${resCSVDst.json().connector.id}/execute response status 200 (ocr)`]: (r) => r.status === 200,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resCSVDst.json().connector.id} response status 204 (ocr)`]: (r) => r.status === 204,
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

        resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(csvDstConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/watch`), {
            [`GET /v1alpha/connectors/${resCSVDst.json().connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/write`,
            JSON.stringify({
                "inputs": constant.instSegModelOutputs
            }), {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/connectors/${resCSVDst.json().connector.id}/write response status 200 (semantic-segmentation)`]: (r) => r.status === 200,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resCSVDst.json().connector.id} response status 204 (semantic-segmentation)`]: (r) => r.status === 204,
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

        resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(csvDstConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/watch`), {
            [`GET /v1alpha/connectors/${resCSVDst.json().connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/write`,
            JSON.stringify({
                "inputs": constant.instSegModelOutputs
            }), {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/connectors/${resCSVDst.json().connector.id}/write response status 200 (instance-segmentation)`]: (r) => r.status === 200,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resCSVDst.json().connector.id} response status 204 (instance-segmentation)`]: (r) => r.status === 204,
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

        resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(csvDstConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/watch`), {
            [`GET /v1alpha/connectors/${resCSVDst.json().connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/write`,
            JSON.stringify({
                "inputs": constant.textGenerationModelOutputs
            }), {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/connectors/${resCSVDst.json().connector.id}/write response status 200 (text-to-image)`]: (r) => r.status === 200,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resCSVDst.json().connector.id} response status 204 (text-to-image)`]: (r) => r.status === 204,
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

        resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(csvDstConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/watch`), {
            [`GET /v1alpha/connectors/${resCSVDst.json().connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/write`,
            JSON.stringify({
                "inputs": constant.textGenerationModelOutputs
            }), {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/connectors/${resCSVDst.json().connector.id}/write response status 200 (text-generation)`]: (r) => r.status === 200,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resCSVDst.json().connector.id} response status 204 (text-generation)`]: (r) => r.status === 204,
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

        resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(csvDstConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/watch`), {
            [`GET /v1alpha/connectors/${resCSVDst.json().connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/write`,
            JSON.stringify({
                "inputs": constant.unspecifiedModelOutputs
            }), {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/connectors/${resCSVDst.json().connector.id}/write response status 200 (unspecified)`]: (r) => r.status === 200,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resCSVDst.json().connector.id} response status 204 (unspecified)`]: (r) => r.status === 204,
        });
    });
}
