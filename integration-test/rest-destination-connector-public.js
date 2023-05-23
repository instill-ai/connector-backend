import http from "k6/http";
import { check, group, sleep } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import { connectorPublicHost } from "./const.js"

import * as constant from "./const.js"
import * as helper from "./helper.js"

export function CheckCreate() {

    group("Connector API: Create destination connectors", () => {

        // destination-http
        var httpDstConnector = {
            "id": "destination-http",
            "destination_connector_definition": constant.httpDstDefRscName,
            "connector": {
                "description": "HTTP source",
                "configuration": {},
            }
        }

        var resDstHTTP = http.request(
            "POST",
            `${connectorPublicHost}/v1alpha/destination-connectors`,
            JSON.stringify(httpDstConnector), constant.params)
        check(resDstHTTP, {
            "POST /v1alpha/destination-connectors response status for creating HTTP destination connector 201": (r) => r.status === 201,
            "POST /v1alpha/destination-connectors response connector name": (r) => r.json().destination_connector.name == `destination-connectors/${httpDstConnector.id}`,
            "POST /v1alpha/destination-connectors response connector uid": (r) => helper.isUUID(r.json().destination_connector.uid),
            "POST /v1alpha/destination-connectors response connector destination_connector_definition": (r) => r.json().destination_connector.destination_connector_definition === constant.httpDstDefRscName,
            "POST /v1alpha/destination-connectors response connector owner is UUID": (r) => helper.isValidOwner(r.json().destination_connector.connector.user),
        });

        check(http.request(
            "POST",
            `${connectorPublicHost}/v1alpha/destination-connectors`,
            JSON.stringify(httpDstConnector), constant.params), {
            "POST /v1alpha/destination-connectors response duplicate HTTP destination connector status 409": (r) => r.status === 409
        });

        // destination-grpc
        var gRPCDstConnector = {
            "id": "destination-grpc",
            "destination_connector_definition": constant.gRPCDstDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        var resDstGRPC = http.request(
            "POST",
            `${connectorPublicHost}/v1alpha/destination-connectors`,
            JSON.stringify(gRPCDstConnector), constant.params)

        check(resDstGRPC, {
            "POST /v1alpha/destination-connectors response status for creating gRPC destination connector 201": (r) => r.status === 201,
        });

        check(http.request(
            "POST",
            `${connectorPublicHost}/v1alpha/destination-connectors`,
            {}, constant.params), {
            "POST /v1alpha/destination-connectors response status for creating empty body 400": (r) => r.status === 400,
        });

        // destination-csv
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

        check(resCSVDst, {
            "POST /v1alpha/destination-connectors response status 201": (r) => r.status === 201,
        });

        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch`), {
            [`GET /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}`), {
            [`GET /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response STATE_CONNECTED`]: (r) => r.json().destination_connector.connector.state === "STATE_CONNECTED",
        });

        // destination-mysql (will end up with STATE_ERROR)
        var mySQLDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.mySQLDstDefRscName,
            "connector": {
                "configuration": {
                    "host": randomString(10),
                    "port": 3306,
                    "username": randomString(10),
                    "database": randomString(10),
                }
            }
        }

        var resDstMySQL = http.request(
            "POST",
            `${connectorPublicHost}/v1alpha/destination-connectors`,
            JSON.stringify(mySQLDstConnector), constant.params)

        check(resDstMySQL, {
            "POST /v1alpha/destination-connectors response status for creating MySQL destination connector 201": (r) => r.status === 201,
            "POST /v1alpha/destination-connectors response connector name": (r) => r.json().destination_connector.name == `destination-connectors/${mySQLDstConnector.id}`,
            "POST /v1alpha/destination-connectors response connector uid": (r) => helper.isUUID(r.json().destination_connector.uid),
            "POST /v1alpha/destination-connectors response connector destination_connector_definition": (r) => r.json().destination_connector.destination_connector_definition === constant.mySQLDstDefRscName,
            "POST /v1alpha/destination-connectors response connector owner is UUID": (r) => helper.isValidOwner(r.json().destination_connector.connector.user),
        });

        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors/${resDstMySQL.json().destination_connector.id}/watch`), {
            [`GET /v1alpha/destination-connectors/${resDstMySQL.json().destination_connector.id}/watch response connector state ended up STATE_ERROR`]: (r) => r.json().state === "STATE_ERROR",
        })

        // check JSON Schema failure cases
        var jsonSchemaFailedBodyCSV = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": {} // required destination_path
            }
        }

        check(http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors`, JSON.stringify(jsonSchemaFailedBodyCSV), constant.params), {
            "POST /v1alpha/destination-connectors response status for JSON Schema failed body 400 (destination-csv missing destination_path)": (r) => r.status === 400,
        });

        var jsonSchemaFailedBodyMySQL = {
            "id": randomString(10),
            "destination_connector_definition": constant.mySQLDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": {
                    "host": randomString(10),
                    "port": "3306",
                    "username": randomString(10),
                    "database": randomString(10),
                } // required port integer type
            }
        }

        check(http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors`, JSON.stringify(jsonSchemaFailedBodyMySQL), constant.params), {
            "POST /v1alpha/destination-connectors response status for JSON Schema failed body 400 (destination-mysql port not integer)": (r) => r.status === 400,
        });

        // Delete test records
        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${resDstHTTP.json().destination_connector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${resDstHTTP.json().destination_connector.id} response status 204`]: (r) => r.status === 204,
        });
        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${resDstGRPC.json().destination_connector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${resDstGRPC.json().destination_connector.id} response status 204`]: (r) => r.status === 204,
        });
        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response status 204`]: (r) => r.status === 204,
        });
        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${resDstMySQL.json().destination_connector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${resDstMySQL.json().destination_connector.id} response status 204`]: (r) => r.status === 204,
        });
    });

}

export function CheckList() {

    group("Connector API: List destination connectors", () => {

        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors`), {
            [`GET /v1alpha/destination-connectors response status is 200`]: (r) => r.status === 200,
            [`GET /v1alpha/destination-connectors response destination_connectors array is 0 length`]: (r) => r.json().destination_connectors.length === 0,
            [`GET /v1alpha/destination-connectors response next_page_token is empty`]: (r) => r.json().next_page_token === "",
            [`GET /v1alpha/destination-connectors response total_size is 0`]: (r) => r.json().total_size == 0,
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

        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors`), {
            [`GET /v1alpha/destination-connectors response status is 200`]: (r) => r.status === 200,
            [`GET /v1alpha/destination-connectors response has destination_connectors array`]: (r) => Array.isArray(r.json().destination_connectors),
            [`GET /v1alpha/destination-connectors response has total_size = ${numConnectors}`]: (r) => r.json().total_size == numConnectors,
        });

        var limitedRecords = http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors`)
        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors?page_size=0`), {
            "GET /v1alpha/destination-connectors?page_size=0 response status is 200": (r) => r.status === 200,
            "GET /v1alpha/destination-connectors?page_size=0 response all records": (r) => r.json().destination_connectors.length === limitedRecords.json().destination_connectors.length,
        });

        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors?page_size=1`), {
            "GET /v1alpha/destination-connectors?page_size=1 response status is 200": (r) => r.status === 200,
            "GET /v1alpha/destination-connectors?page_size=1 response destination_connectors size 1": (r) => r.json().destination_connectors.length === 1,
        });

        var pageRes = http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors?page_size=1`)
        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors?page_size=1&page_token=${pageRes.json().next_page_token}`), {
            [`GET /v1alpha/destination-connectors?page_size=1&page_token=${pageRes.json().next_page_token} response status is 200`]: (r) => r.status === 200,
            [`GET /v1alpha/destination-connectors?page_size=1&page_token=${pageRes.json().next_page_token} response destination_connectors size 1`]: (r) => r.json().destination_connectors.length === 1,
        });

        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors?page_size=1&view=VIEW_BASIC`), {
            "GET /v1alpha/destination-connectors?page_size=1&view=VIEW_BASIC response status 200": (r) => r.status === 200,
            "GET /v1alpha/destination-connectors?page_size=1&view=VIEW_BASIC response destination_connectors[0].connector.configuration is null": (r) => r.json().destination_connectors[0].connector.configuration === null,
            "GET /v1alpha/destination-connectors?page_size=1&view=VIEW_BASIC response destination_connectors[0].connector.owner is UUID": (r) => helper.isValidOwner(r.json().destination_connectors[0].connector.user),
        });

        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors?page_size=1&view=VIEW_FULL`), {
            "GET /v1alpha/destination-connectors?page_size=1&view=VIEW_FULL response status 200": (r) => r.status === 200,
            "GET /v1alpha/destination-connectors?page_size=1&view=VIEW_FULL response destination_connectors[0].connector.configuration is not null": (r) => r.json().destination_connectors[0].connector.configuration !== null,
            "GET /v1alpha/destination-connectors?page_size=1&view=VIEW_FULL response destination_connectors[0].connector.owner is UUID": (r) => helper.isValidOwner(r.json().destination_connectors[0].connector.user),
        });

        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors?page_size=1`), {
            "GET /v1alpha/destination-connectors?page_size=1 response status 200": (r) => r.status === 200,
            "GET /v1alpha/destination-connectors?page_size=1 response destination_connectors[0].connector.configuration is null": (r) => r.json().destination_connectors[0].connector.configuration === null,
            "GET /v1alpha/destination-connectors?page_size=1 response destination_connectors[0].connector.owner is UUID": (r) => helper.isValidOwner(r.json().destination_connectors[0].connector.user),
        });

        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors?page_size=${limitedRecords.json().total_size}`), {
            [`GET /v1alpha/destination-connectors?page_size=${limitedRecords.json().total_size} response status 200`]: (r) => r.status === 200,
            [`GET /v1alpha/destination-connectors?page_size=${limitedRecords.json().total_size} response next_page_token is empty`]: (r) => r.json().next_page_token === ""
        });

        // Delete the destination connectors
        for (const reqBody of reqBodies) {
            check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${reqBody.id}`), {
                [`DELETE /v1alpha/destination-connectors x${reqBodies.length} response status is 204`]: (r) => r.status === 204,
            });
        }
    });
}

export function CheckGet() {

    group("Connector API: Get destination connectors by ID", () => {

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

        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}`), {
            [`GET /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response status 200`]: (r) => r.status === 200,
            [`GET /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response connector id`]: (r) => r.json().destination_connector.id === csvDstConnector.id,
            [`GET /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response connector destination_connector_definition permalink`]: (r) => r.json().destination_connector.destination_connector_definition === constant.csvDstDefRscName,
            [`GET /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response connector owner is UUID`]: (r) => helper.isValidOwner(r.json().destination_connector.connector.user),
        });

        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch`), {
            [`GET /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response status 204`]: (r) => r.status === 204,
        });
    });
}

export function CheckUpdate() {

    group("Connector API: Update destination connectors", () => {

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

        var csvDstConnectorUpdate = {
            "id": csvDstConnector.id,
            "destination_connector_definition": csvDstConnector.destination_connector_definition,
            "connector": {
                "tombstone": true,
                "description": randomString(50),
                "configuration": {
                    destination_path: "/tmp"
                }
            }
        }

        var resCSVDstUpdate = http.request("PATCH", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}`,
            JSON.stringify(csvDstConnectorUpdate), constant.params)

        check(resCSVDstUpdate, {
            [`PATCH /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response status 200`]: (r) => r.status === 200,
            [`PATCH /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response connector id`]: (r) => r.json().destination_connector.id === csvDstConnectorUpdate.id,
            [`PATCH /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response connector connector_definition`]: (r) => r.json().destination_connector.destination_connector_definition === constant.csvDstDefRscName,
            [`PATCH /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response connector description`]: (r) => r.json().destination_connector.connector.description === csvDstConnectorUpdate.connector.description,
            [`PATCH /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response connector tombstone`]: (r) => r.json().destination_connector.connector.tombstone === false,
            [`PATCH /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response connector configuration`]: (r) => r.json().destination_connector.connector.configuration.destination_path === csvDstConnectorUpdate.connector.configuration.destination_path,
            [`PATCH /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response connector owner is UUID`]: (r) => helper.isValidOwner(r.json().destination_connector.connector.user),
        });

        // Try to update with empty description
        csvDstConnectorUpdate = {
            "connector": {
                "description": "",
            }
        }

        resCSVDstUpdate = http.request("PATCH", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}`,
            JSON.stringify(csvDstConnectorUpdate), constant.params)

        check(resCSVDstUpdate, {
            [`PATCH /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} with empty description response status 200`]: (r) => r.status === 200,
            [`PATCH /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} with empty description response empty description`]: (r) => r.json().destination_connector.connector.description === csvDstConnectorUpdate.connector.description,
        })

        // Try to update with a non-existing name field (which should be ignored because name field is OUTPUT_ONLY)
        csvDstConnectorUpdate = {
            "name": `destination-connectors/${randomString(5)}`,
            "connector": {
                "description": randomString(50),
            }
        }

        resCSVDstUpdate = http.request("PATCH", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}`,
            JSON.stringify(csvDstConnectorUpdate), constant.params)

        check(resCSVDstUpdate, {
            [`PATCH /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} with non-existing name field response status 200`]: (r) => r.status === 200,
        })

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${csvDstConnector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${csvDstConnector.id} response status 204`]: (r) => r.status === 204,
        });
    });
}

export function CheckLookUp() {

    group("Connector API: Look up destination connectors by UID", () => {

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

        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.uid}/lookUp`), {
            [`GET /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.uid}/lookUp response status 200`]: (r) => r.status === 200,
            [`GET /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.uid}/lookUp response connector uid`]: (r) => r.json().destination_connector.uid === resCSVDst.json().destination_connector.uid,
            [`GET /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.uid}/lookUp response connector destination_connector_definition`]: (r) => r.json().destination_connector.destination_connector_definition === constant.csvDstDefRscName,
            [`GET /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.uid}/lookUp response connector owner is UUID`]: (r) => helper.isValidOwner(r.json().destination_connector.connector.user),
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response status 204`]: (r) => r.status === 204,
        });

    });
}

export function CheckState() {

    group("Connector API: Change state destination connectors", () => {
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

        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch`), {
            [`GET /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/connect`, null, constant.params), {
            [`POST /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/connect response status 200 (with STATE_CONNECTED)`]: (r) => r.status === 200,
        });

        check(http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/disconnect`, null, constant.params), {
            [`POST /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/disconnect response status 200 (with STATE_CONNECTED)`]: (r) => r.status === 200,
        });

        check(http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/disconnect`, null, constant.params), {
            [`POST /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/disconnect response status 200 (with STATE_DISCONNECTED)`]: (r) => r.status === 200,
        });

        check(http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/connect`, null, constant.params), {
            [`POST /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/connect response status 200 (with STATE_DISCONNECTED)`]: (r) => r.status === 200,
        });

        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch`), {
            [`GET /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response status 204`]: (r) => r.status === 204,
        });

    });

}

export function CheckRename() {

    group("Connector API: Rename destination connectors", () => {

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

        check(http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/rename`,
            JSON.stringify({
                "new_destination_connector_id": `some-id-not-${resCSVDst.json().destination_connector.id}`
            }), constant.params), {
            [`POST /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/rename response status 200`]: (r) => r.status === 200,
            [`POST /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/rename response id is some-id-not-${resCSVDst.json().destination_connector.id}`]: (r) => r.json().destination_connector.id === `some-id-not-${resCSVDst.json().destination_connector.id}`,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/some-id-not-${resCSVDst.json().destination_connector.id}`), {
            [`DELETE /v1alpha/destination-connectors/some-id-not-${resCSVDst.json().destination_connector.id} response status 204`]: (r) => r.status === 204,
        });
    });
}

export function CheckWrite() {

    group("Connector API: Write destination connectors", () => {

        var csvDstConnector, resCSVDst, currentTime, timeoutTime

        // Write classification output
        csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": {
                    "destination_path": "/local/test-classification"
                },
            }
        }

        resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors`,
            JSON.stringify(csvDstConnector), constant.params)

        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch`), {
            [`GET /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/write`,
            JSON.stringify({
                "sync_mode": "SUPPORTED_SYNC_MODES_FULL_REFRESH",
                "destination_sync_mode": "SUPPORTED_DESTINATION_SYNC_MODES_OVERWRITE",
                "pipeline": "pipelines/dummy-pipeline",
                "recipe": {
                    "version": "v1alpha",
                    "components": [
                        {"id": "s01", "resource_name": "source-connectors/dummy-source"},
                        {"id": "m01", "resource_name": "models/dummy-model"},
                        {"id": "d01", "resource_name": "destination-connectors/dummy-destination"},
                    ]
                },
                "data_mapping_indices": ["01GB5T5ZK9W9C2VXMWWRYM8WPA"],
                "model_outputs": constant.clsModelOutputs
            }), {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/write response status 200 (classification)`]: (r) => r.status === 200,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response status 204 (classification)`]: (r) => r.status === 204,
        });

        // Write detection output (empty bounding_boxes)
        csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": {
                    "destination_path": "/local/test-detection-empty-bounding-boxes"
                },
            }
        }

        resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors`,
            JSON.stringify(csvDstConnector), constant.params)

        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch`), {
            [`GET /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/write`,
            JSON.stringify({
                "sync_mode": "SUPPORTED_SYNC_MODES_FULL_REFRESH",
                "destination_sync_mode": "SUPPORTED_DESTINATION_SYNC_MODES_OVERWRITE",
                "pipeline": "pipelines/dummy-pipeline",
                "recipe": {
                    "version": "v1alpha",
                    "components": [
                        {"id": "s01", "resource_name": "source-connectors/dummy-source"},
                        {"id": "m01", "resource_name": "models/dummy-model"},
                        {"id": "d01", "resource_name": "destination-connectors/dummy-destination"},
                    ]
                },
                "data_mapping_indices": ["01GB5T5ZK9W9C2VXMWWRYM8WPM"],
                "model_outputs": constant.detectionEmptyModelOutputs
            }), {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/write response status 200 (detection)`]: (r) => r.status === 200,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response status 204 (detection)`]: (r) => r.status === 204,
        });

        // Write detection output (multiple models)
        csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": {
                    "destination_path": "/local/test-detection-multi-models"
                },
            }
        }

        resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors`,
            JSON.stringify(csvDstConnector), constant.params)

        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch`), {
            [`GET /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/write`,
            JSON.stringify({
                "sync_mode": "SUPPORTED_SYNC_MODES_FULL_REFRESH",
                "destination_sync_mode": "SUPPORTED_DESTINATION_SYNC_MODES_OVERWRITE",
                "pipeline": "pipelines/dummy-pipeline",
                "recipe": {
                    "version": "v1alpha",
                    "components": [
                        {"id": "s01", "resource_name": "source-connectors/dummy-source"},
                        {"id": "m01", "resource_name": "models/dummy-model"},
                        {"id": "m02", "resource_name": "models/dummy-model"},
                        {"id": "d01", "resource_name": "destination-connectors/dummy-destination"},
                    ]
                },
                "data_mapping_indices": ["01GB5T5ZK9W9C2VXMWWRYM8WPM", "01GB5T5ZK9W9C2VXMWWRYM8WPN", "01GB5T5ZK9W9C2VXMWWRYM8WPO"],
                "model_outputs": constant.detectionModelOutputs
            }), {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/write response status 200 (detection)`]: (r) => r.status === 200,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response status 204 (detection)`]: (r) => r.status === 204,
        });

        // Write keypoint output
        csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": {
                    "destination_path": "/local/test-keypoint"
                },
            }
        }

        resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors`,
            JSON.stringify(csvDstConnector), constant.params)

        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch`), {
            [`GET /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/write`,
            JSON.stringify({
                "sync_mode": "SUPPORTED_SYNC_MODES_FULL_REFRESH",
                "destination_sync_mode": "SUPPORTED_DESTINATION_SYNC_MODES_OVERWRITE",
                "pipeline": "pipelines/dummy-pipeline",
                "recipe": {
                    "version": "v1alpha",
                    "components": [
                        {"id": "s01", "resource_name": "source-connectors/dummy-source"},
                        {"id": "m01", "resource_name": "models/dummy-model"},
                        {"id": "d01", "resource_name": "destination-connectors/dummy-destination"},
                    ]
                },
                "data_mapping_indices": ["01GB5T5ZK9W9C2VXMWWRYM8WPA"],
                "model_outputs": constant.keypointModelOutputs
            }), {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/write response status 200 (keypoint)`]: (r) => r.status === 200,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response status 204 (keypoint)`]: (r) => r.status === 204,
        });

        // Write ocr output
        csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": {
                    "destination_path": "/local/test-ocr"
                },
            }
        }

        resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors`,
            JSON.stringify(csvDstConnector), constant.params)

        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch`), {
            [`GET /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/write`,
            JSON.stringify({
                "sync_mode": "SUPPORTED_SYNC_MODES_FULL_REFRESH",
                "destination_sync_mode": "SUPPORTED_DESTINATION_SYNC_MODES_OVERWRITE",
                "pipeline": "pipelines/dummy-pipeline",
                "recipe": {
                    "version": "v1alpha",
                    "components": [
                        {"id": "s01", "resource_name": "source-connectors/dummy-source"},
                        {"id": "m01", "resource_name": "models/dummy-model"},
                        {"id": "d01", "resource_name": "destination-connectors/dummy-destination"},
                    ]
                },
                "data_mapping_indices": ["01GB5T5ZK9W9C2VXMWWRYM8WPA"],
                "model_outputs": constant.ocrModelOutputs
            }), {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/write response status 200 (ocr)`]: (r) => r.status === 200,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response status 204 (ocr)`]: (r) => r.status === 204,
        });

        // Write semantic segmentation output
        csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": {
                    "destination_path": "/local/test-semantic-segmentation"
                },
            }
        }

        resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors`,
            JSON.stringify(csvDstConnector), constant.params)

        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch`), {
            [`GET /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/write`,
            JSON.stringify({
                "sync_mode": "SUPPORTED_SYNC_MODES_FULL_REFRESH",
                "destination_sync_mode": "SUPPORTED_DESTINATION_SYNC_MODES_OVERWRITE",
                "pipeline": "pipelines/dummy-pipeline",
                "recipe": {
                    "version": "v1alpha",
                    "components": [
                        {"id": "s01", "resource_name": "source-connectors/dummy-source"},
                        {"id": "m01", "resource_name": "models/dummy-model"},
                        {"id": "d01", "resource_name": "destination-connectors/dummy-destination"},
                    ]
                },
                "data_mapping_indices": ["01GB5T5ZK9W9C2VXMWWRYM8WPA"],
                "model_outputs": constant.semanticSegModelOutputs
            }), {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/write response status 200 (semantic-segmentation)`]: (r) => r.status === 200,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response status 204 (semantic-segmentation)`]: (r) => r.status === 204,
        });

        // Write instance segmentation output
        csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": {
                    "destination_path": "/local/test-instance-segmentation"
                },
            }
        }

        resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors`,
            JSON.stringify(csvDstConnector), constant.params)

        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch`), {
            [`GET /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/write`,
            JSON.stringify({
                "sync_mode": "SUPPORTED_SYNC_MODES_FULL_REFRESH",
                "destination_sync_mode": "SUPPORTED_DESTINATION_SYNC_MODES_OVERWRITE",
                "pipeline": "pipelines/dummy-pipeline",
                "recipe": {
                    "version": "v1alpha",
                    "components": [
                        {"id": "s01", "resource_name": "source-connectors/dummy-source"},
                        {"id": "m01", "resource_name": "models/dummy-model"},
                        {"id": "d01", "resource_name": "destination-connectors/dummy-destination"},
                    ]
                },
                "data_mapping_indices": ["01GB5T5ZK9W9C2VXMWWRYM8WPA"],
                "model_outputs": constant.instSegModelOutputs
            }), {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/write response status 200 (instance-segmentation)`]: (r) => r.status === 200,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response status 204 (instance-segmentation)`]: (r) => r.status === 204,
        });

        // Write text-to-image output
        csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": {
                    "destination_path": "/local/test-text-to-image"
                },
            }
        }

        resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors`,
            JSON.stringify(csvDstConnector), constant.params)

        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch`), {
            [`GET /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/write`,
            JSON.stringify({
                "sync_mode": "SUPPORTED_SYNC_MODES_FULL_REFRESH",
                "destination_sync_mode": "SUPPORTED_DESTINATION_SYNC_MODES_OVERWRITE",
                "pipeline": "pipelines/dummy-pipeline",
                "recipe": {
                    "version": "v1alpha",
                    "components": [
                        {"id": "s01", "resource_name": "source-connectors/dummy-source"},
                        {"id": "m01", "resource_name": "models/dummy-model"},
                        {"id": "d01", "resource_name": "destination-connectors/dummy-destination"},
                    ]
                },
                "data_mapping_indices": ["01GB5T5ZK9W9C2VXMWWRYM8WPA"],
                "model_outputs": constant.textToImageModelOutputs
            }), {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/write response status 200 (text-to-image)`]: (r) => r.status === 200,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response status 204 (text-to-image)`]: (r) => r.status === 204,
        });

        // Write text-generation output
        csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": {
                    "destination_path": "/local/test-text-generation"
                },
            }
        }

        resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors`,
            JSON.stringify(csvDstConnector), constant.params)

        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch`), {
            [`GET /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/write`,
            JSON.stringify({
                "sync_mode": "SUPPORTED_SYNC_MODES_FULL_REFRESH",
                "destination_sync_mode": "SUPPORTED_DESTINATION_SYNC_MODES_OVERWRITE",
                "pipeline": "pipelines/dummy-pipeline",
                "recipe": {
                    "version": "v1alpha",
                    "components": [
                        {"id": "s01", "resource_name": "source-connectors/dummy-source"},
                        {"id": "m01", "resource_name": "models/dummy-model"},
                        {"id": "d01", "resource_name": "destination-connectors/dummy-destination"},
                    ]
                },
                "data_mapping_indices": ["01GB5T5ZK9W9C2VXMWWRYM8WPA"],
                "model_outputs": constant.textGenerationModelOutputs
            }), {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/write response status 200 (text-generation)`]: (r) => r.status === 200,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response status 204 (text-generation)`]: (r) => r.status === 204,
        });

        // Write unspecified output
        csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": {
                    "destination_path": "/local/test-unspecified"
                },
            }
        }

        resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors`,
            JSON.stringify(csvDstConnector), constant.params)

        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch`), {
            [`GET /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        check(http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/write`,
            JSON.stringify({
                "sync_mode": "SUPPORTED_SYNC_MODES_FULL_REFRESH",
                "destination_sync_mode": "SUPPORTED_DESTINATION_SYNC_MODES_OVERWRITE",
                "pipeline": "pipelines/dummy-pipeline",
                "recipe": {
                    "version": "v1alpha",
                    "components": [
                        {"id": "s01", "resource_name": "source-connectors/dummy-source"},
                        {"id": "m01", "resource_name": "models/dummy-model"},
                        {"id": "d01", "resource_name": "destination-connectors/dummy-destination"},
                    ]
                },
                "data_mapping_indices": ["01GB5T5ZK9W9C2VXMWWRYM8WPA"],
                "model_outputs": constant.unspecifiedModelOutputs
            }), {
            headers: { "Content-Type": "application/json" }
        }), {
            [`POST /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/write response status 200 (unspecified)`]: (r) => r.status === 200,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response status 204 (unspecified)`]: (r) => r.status === 204,
        });
    });
}

export function CheckTest() {

    group("Connector API: Test destination connectors by ID", () => {

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

        check(http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/testConnection`), {
            [`POST /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/testConnection response status 200`]: (r) => r.status === 200,
            [`POST /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/testConnection response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response status 204`]: (r) => r.status === 204,
        });
    });
}
