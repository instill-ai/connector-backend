import http from "k6/http";
import { check, group, sleep } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import { connectorPublicHost } from "./const.js"

import * as constant from "./const.js"
import * as helper from "./helper.js"

export function CheckCreate() {

    group(`Connector API: Create destination connectors [with random "jwt-sub" header]`, () => {

        // destination-http
        var httpDstConnector = {
            "id": "destination-http",
            "destination_connector_definition": constant.httpDstDefRscName,
            "connector": {
                "description": "HTTP source",
                "configuration": {},
            }
        }

        // destination-grpc
        var gRPCDstConnector = {
            "id": "destination-grpc",
            "destination_connector_definition": constant.gRPCDstDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        // Cannot create http destination connector of a non-exist user
        check(http.request("POST",
            `${connectorPublicHost}/v1alpha/destination-connectors`,
            JSON.stringify(httpDstConnector), constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] POST /v1alpha/destination-connectors response for creating HTTP destination status is 404`]: (r) => r.status === 404,
        });

        // Cannot create grpc destination connector of a non-exist user
        check(http.request("POST",
            `${connectorPublicHost}/v1alpha/destination-connectors`,
            JSON.stringify(gRPCDstConnector), constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] POST /v1alpha/destination-connectors response for creating gRPC destination status is 404`]: (r) => r.status === 404,
        });
    });

}

export function CheckList() {

    group(`Connector API: List destination connectors [with random "jwt-sub" header]`, () => {

        // Cannot list destination connector of a non-exist user
        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors`, null, constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] GET /v1alpha/destination-connectors response status is 404`]: (r) => r.status === 404,
        });
    });
}

export function CheckGet() {

    group(`Connector API: Get destination connectors by ID [with random "jwt-sub" header]`, () => {

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

        // Cannot get a destination connector of a non-exist user
        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}`, null, constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] GET /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response status is 404`]: (r) => r.status === 404,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response status 204`]: (r) => r.status === 204,
        });
    });
}

export function CheckUpdate() {

    group(`Connector API: Update destination connectors [with random "jwt-sub" header]`, () => {

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

        // Cannot patch a destination connector of a non-exist user
        check(http.request(
            "PATCH",
            `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}`,
            JSON.stringify(csvDstConnectorUpdate), constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] PATCH /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response status 404`]: (r) => r.status === 404,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${csvDstConnector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${csvDstConnector.id} response status 204`]: (r) => r.status === 204,
        });
    });
}

export function CheckLookUp() {

    group(`Connector API: Look up destination connectors by UID [with random "jwt-sub" header]`, () => {

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

        // Cannot look up a destination connector of a non-exist user
        check(http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.uid}/lookUp`, null, constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] GET /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.uid}/lookUp response status 404`]: (r) => r.status === 404,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response status 204`]: (r) => r.status === 204,
        });

    });
}

export function CheckState() {

    group(`Connector API: Change state destination connectors [with random "jwt-sub" header]`, () => {
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

        check(http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/disconnect`, null, constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] POST /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/disconnect response at UNSPECIFIED state status 404`]: (r) => r.status === 404,
        });

        check(http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/connect`, null, constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] POST /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/connect response at UNSPECIFIED state status 404`]: (r) => r.status === 404,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response status 204`]: (r) => r.status === 204,
        });

    });

}

export function CheckRename() {

    group(`Connector API: Rename destination connectors [with random "jwt-sub" header]`, () => {

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

        // Cannot rename destination connector of a non-exist user
        check(http.request("POST", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/rename`,
            JSON.stringify({
                "new_destination_connector_id": `some-id-not-${resCSVDst.json().destination_connector.id}`
            }), constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] POST /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/rename response status 404`]: (r) => r.status === 404,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${csvDstConnector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${csvDstConnector.id} response status 204`]: (r) => r.status === 204,
        });
    });
}

export function CheckWrite() {

    group(`Connector API: Write destination connectors [with random "jwt-sub" header]`, () => {

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

        // Check connector state being updated in 120 secs
        currentTime = new Date().getTime();
        timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            var res = http.request("GET", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/watch`)
            if (res.json().state === "STATE_CONNECTED") {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        // Cannot write to destination connector of a non-exist user
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
            }), constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] POST /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}/write response status 404 (classification)`]: (r) => r.status === 404,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id}`), {
            [`DELETE /v1alpha/destination-connectors/${resCSVDst.json().destination_connector.id} response status 204 (classification)`]: (r) => r.status === 204,
        });
    });
}
