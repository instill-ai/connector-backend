import http from "k6/http";
import { check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js"
import * as helper from "./helper.js"

export function CheckCreate() {

    group("Connector API: Create destination connectors", () => {

        var dirHTTPDstConnector = {
            "id": "http",
            "destination_connector_definition": constant.httpDstDefRscName,
            "connector": {
                "configuration": JSON.stringify({})
            }
        }

        var dirGRPCDstConnector = {
            "id": "grpc",
            "destination_connector_definition": constant.gRPCDstDefRscName,
            "connector": {
                "configuration": JSON.stringify({})
            }
        }

        var resDstHTTP = http.request(
            "POST",
            `${connectorHost}/v1alpha/destination-connectors`,
            JSON.stringify(dirHTTPDstConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(resDstHTTP, {
            "POST /v1alpha/destination-connectors response status for creating directness HTTP destination connector 201": (r) => r.status === 201,
            "POST /v1alpha/destination-connectors response connector uid": (r) => helper.isUUID(r.json().destination_connector.uid),
            "POST /v1alpha/destination-connectors response connector destination_connector_definition": (r) => r.json().destination_connector.destination_connector_definition === constant.httpDstDefRscName
        });

        check(http.request(
            "POST",
            `${connectorHost}/v1alpha/destination-connectors`,
            JSON.stringify(dirHTTPDstConnector), {
            headers: { "Content-Type": "application/json" },
        }), {
            "POST /v1alpha/destination-connectors response duplicate directness connector status 409": (r) => r.status === 409
        });

        var resDstGRPC = http.request(
            "POST",
            `${connectorHost}/v1alpha/destination-connectors`,
            JSON.stringify(dirGRPCDstConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(resDstGRPC, {
            "POST /v1alpha/destination-connectors response status for creating directness gRPC destination connector 201": (r) => r.status === 201,
        });

        // Delete test records
        check(http.request("DELETE", `${connectorHost}/v1alpha/destination-connectors/${resDstHTTP.json().destination_connector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${resDstHTTP.json().destination_connector.id} response status 204`]: (r) => r.status === 204,
        });
        check(http.request("DELETE", `${connectorHost}/v1alpha/destination-connectors/${resDstGRPC.json().destination_connector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${resDstGRPC.json().destination_connector.id} response status 204`]: (r) => r.status === 204,
        });
    });

}

export function CheckList() {

    group("Connector API: List destination connectors", () => {

        const numConnectors = 10

        var reqBodies = [];
        for (var i = 0; i < numConnectors; i++) {
            reqBodies[i] = {
                "id": randomString(10),
                "destination_connector_definition": constant.csvDstDefinitionRscName,
                "connector": {
                    "description": randomString(50),
                    "configuration": JSON.stringify(constant.csvDstConfig)
                }
            }
        }

        // Create connectors
        for (const reqBody of reqBodies) {
            check(http.request(
                "POST",
                `${connectorHost}/v1alpha/destination-connectors`,
                JSON.stringify(reqBody), {
                headers: { "Content-Type": "application/json" },
            }), {
                [`POST /v1alpha/destination-connectors x${reqBodies.length} response status 201`]: (r) => r.status === 201,
            });
        }

        check(http.request("GET", `${connectorHost}/v1alpha/destination-connectors`), {
            [`GET /v1alpha/destination-connectors response status is 200`]: (r) => r.status === 200,
            [`GET /v1alpha/destination-connectors response has destination_connectors array`]: (r) => Array.isArray(r.json().destination_connectors),
            [`GET /v1alpha/destination-connectors response has total_size = ${numConnectors}`]: (r) => r.json().total_size == numConnectors,
        });

        var allRes = http.request("GET", `${connectorHost}/v1alpha/destination-connectors`)
        check(http.request("GET", `${connectorHost}/v1alpha/destination-connectors?page_size=0`), {
            "GET /v1alpha/destination-connectors?page_size=0 response status is 200": (r) => r.status === 200,
            "GET /v1alpha/destination-connectors?page_size=0 response all records": (r) => r.json().destination_connectors.length === allRes.json().destination_connectors.length,
        });

        check(http.request("GET", `${connectorHost}/v1alpha/destination-connectors?page_size=1`), {
            "GET /v1alpha/destination-connectors?page_size=1 response status is 200": (r) => r.status === 200,
            "GET /v1alpha/destination-connectors?page_size=1 response destination_connectors size 1": (r) => r.json().destination_connectors.length === 1,
        });

        var pageRes = http.request("GET", `${connectorHost}/v1alpha/destination-connectors?page_size=1`)
        check(http.request("GET", `${connectorHost}/v1alpha/destination-connectors?page_size=1&page_token=${pageRes.json().next_page_token}`), {
            [`GET /v1alpha/destination-connectors?page_size=1&page_token=${pageRes.json().next_page_token} response status is 200`]: (r) => r.status === 200,
            [`GET /v1alpha/destination-connectors?page_size=1&page_token=${pageRes.json().next_page_token} response destination_connectors size 1`]: (r) => r.json().destination_connectors.length === 1,
        });

        check(http.request("GET", `${connectorHost}/v1alpha/destination-connectors?page_size=1&view=VIEW_BASIC`), {
            "GET /v1alpha/destination-connectors?page_size=1&view=VIEW_BASIC response status 200": (r) => r.status === 200,
            "GET /v1alpha/destination-connectors?page_size=1&view=VIEW_BASIC response destination_connectors has no configuration": (r) => JSON.parse(r.json().destination_connectors[0].connector.configuration) === null,
        });

        check(http.request("GET", `${connectorHost}/v1alpha/destination-connectors?page_size=1&view=VIEW_FULL`), {
            "GET /v1alpha/destination-connectors?page_size=1&view=VIEW_FULL response status 200": (r) => r.status === 200,
            "GET /v1alpha/destination-connectors?page_size=1&view=VIEW_FULL response destination_connectors has configuration": (r) => JSON.parse(r.json().destination_connectors[0].connector.configuration) !== null
        });

        check(http.request("GET", `${connectorHost}/v1alpha/destination-connectors?page_size=1`), {
            "GET /v1alpha/destination-connectors?page_size=1 response status 200": (r) => r.status === 200,
            "GET /v1alpha/destination-connectors?page_size=1 response destination_connectors has no configuration": (r) => JSON.parse(r.json().destination_connectors[0].connector.configuration) === null,
        });

        // Delete the destination connectors
        for (const reqBody of reqBodies) {
            check(http.request(
                "DELETE",
                `${connectorHost}/v1alpha/destination-connectors/${reqBody.id}`,
                JSON.stringify(reqBody), {
                headers: {
                    "Content-Type": "application/json",
                },
            }), {
                [`DELETE /v1alpha/destination-connectors x${reqBodies.length} response status is 204`]: (r) => r.status === 204,
            });
        }
    });
}

export function CheckGet() {

    group("Connector API: Get destination connectors by ID", () => {

        var csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefinitionRscName,
            "connector": {
                "description": randomString(50),
                "configuration": JSON.stringify(constant.csvDstConfig)
            }
        }

        var resCSV = http.request("POST", `${connectorHost}/v1alpha/destination-connectors`,
            JSON.stringify(csvDstConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(http.request("GET", `${connectorHost}/v1alpha/destination-connectors/${resCSV.json().destination_connector.id}`), {
            [`GET /v1alpha/destination-connectors/${resCSV.json().destination_connector.id} response status 200`]: (r) => r.status === 200,
            [`GET /v1alpha/destination-connectors/${resCSV.json().destination_connector.id} response connector id`]: (r) => r.json().destination_connector.id === csvDstConnector.id,
            [`GET /v1alpha/destination-connectors/${resCSV.json().destination_connector.id} response connector destination_connector_definition permalink`]: (r) => r.json().destination_connector.destination_connector_definition === constant.csvDstDefinitionRscName,
        });

        check(http.request("DELETE", `${connectorHost}/v1alpha/destination-connectors/${resCSV.json().destination_connector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${resCSV.json().destination_connector.id} response status 204`]: (r) => r.status === 204,
        });
    });
}

export function CheckUpdate() {

    group("Connector API: Update destination connectors", () => {

        var csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefinitionRscName,
            "connector": {
                "description": randomString(50),
                "configuration": JSON.stringify(constant.csvDstConfig)
            }
        }

        var res = http.request(
            "POST",
            `${connectorHost}/v1alpha/destination-connectors`,
            JSON.stringify(csvDstConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(res, {
            "POST /v1alpha/destination-connectors response status 201": (r) => r.status === 201,
        });

        var configUpdate = constant.csvDstConfig
        configUpdate.connection_specification.destination_path = "/tmp"

        var csvDstConnectorUpdate = {
            "id": csvDstConnector.id,
            "destination_connector_definition": csvDstConnector.destination_connector_definition,
            "connector": {
                "tombstone": true,
                "description": randomString(50),
                "configuration": JSON.stringify(configUpdate)
            }
        }

        var resUpdate = http.request(
            "PATCH",
            `${connectorHost}/v1alpha/destination-connectors/${res.json().destination_connector.id}`,
            JSON.stringify(csvDstConnectorUpdate), {
            headers: { "Content-Type": "application/json" },
        })

        check(resUpdate, {
            [`PATCH /v1alpha/destination-connectors/${res.json().destination_connector.id} response status 200`]: (r) => r.status === 200,
            [`PATCH /v1alpha/destination-connectors/${res.json().destination_connector.id} response connector id`]: (r) => r.json().destination_connector.id === csvDstConnectorUpdate.id,
            [`PATCH /v1alpha/destination-connectors/${res.json().destination_connector.id} response connector connector_definition`]: (r) => r.json().destination_connector.destination_connector_definition === constant.csvDstDefinitionRscName,
            [`PATCH /v1alpha/destination-connectors/${res.json().destination_connector.id} response connector description`]: (r) => r.json().destination_connector.connector.description === csvDstConnectorUpdate.connector.description,
            [`PATCH /v1alpha/destination-connectors/${res.json().destination_connector.id} response connector tombstone`]: (r) => r.json().destination_connector.connector.tombstone === false,
            [`PATCH /v1alpha/destination-connectors/${res.json().destination_connector.id} response connector configuration`]: (r) => JSON.parse(r.json().destination_connector.connector.configuration).connection_specification.destination_path === JSON.parse(csvDstConnectorUpdate.connector.configuration).connection_specification.destination_path
        });

        check(http.request("DELETE", `${connectorHost}/v1alpha/destination-connectors/${csvDstConnector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${csvDstConnector.id} response status 204`]: (r) => r.status === 204,
        });
    });
}

export function CheckLookUp() {

    group("Connector API: Look up destination connectors by UID", () => {

        var csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefinitionRscName,
            "connector": {
                "description": randomString(50),
                "configuration": JSON.stringify(constant.csvDstConfig)
            }
        }

        var resCSV = http.request("POST", `${connectorHost}/v1alpha/destination-connectors`,
            JSON.stringify(csvDstConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(http.request("GET", `${connectorHost}/v1alpha/destination-connectors/${resCSV.json().destination_connector.uid}:lookUp`), {
            [`GET /v1alpha/destination-connectors/${resCSV.json().destination_connector.uid}:lookUp response status 200`]: (r) => r.status === 200,
            [`GET /v1alpha/destination-connectors/${resCSV.json().destination_connector.uid}:lookUp response connector uid`]: (r) => r.json().destination_connector.uid === resCSV.json().destination_connector.uid,
            [`GET /v1alpha/destination-connectors/${resCSV.json().destination_connector.uid}:lookUp response connector destination_connector_definition`]: (r) => r.json().destination_connector.destination_connector_definition === constant.csvDstDefinitionRscName,
        });

        check(http.request("DELETE", `${connectorHost}/v1alpha/destination-connectors/${resCSV.json().destination_connector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${resCSV.json().destination_connector.id} response status 204`]: (r) => r.status === 204,
        });

    });

}

export function CheckRename() {

    group("Connector API: Rename destination connectors", () => {
        var csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefinitionRscName,
            "connector": {
                "description": randomString(50),
                "configuration": JSON.stringify(constant.csvDstConfig)
            }
        }

        var resCSV = http.request("POST", `${connectorHost}/v1alpha/destination-connectors`,
            JSON.stringify(csvDstConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(http.request("POST", `${connectorHost}/v1alpha/destination-connectors/${resCSV.json().destination_connector.id}:rename`,
        JSON.stringify({
            "new_destination_connector_id": `some-id-not-${csvDstConnector.id}`
        }), {
            headers: { "Content-Type": "application/json" }
        }),{
            [`POST /v1alpha/destination-connectors/${resCSV.json().destination_connector.id}:rename response status 200`]: (r) => r.status === 200,
            [`POST /v1alpha/destination-connectors/${resCSV.json().destination_connector.id}:rename response id is some-id-not-${csvDstConnector.id}`]: (r) => r.json().destination_connector.id === `some-id-not-${csvDstConnector.id}`,
        });

        check(http.request("DELETE", `${connectorHost}/v1alpha/destination-connectors/some-id-not-${csvDstConnector.id}`), {
            [`DELETE /v1alpha/destination-connectors/some-id-not-${csvDstConnector.id} response status 204`]: (r) => r.status === 204,
        });    });

}
