import http from "k6/http";
import { check, group, sleep } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import { connectorPublicHost } from "./const.js"

import * as constant from "./const.js"
import * as helper from "./helper.js"

export function CheckCreate() {

    group(`Connector API: Create destination connectors [with random "jwt-sub" header]`, () => {

        // end
        var httpDstConnector = {
            "id": "end-operator",
            "connector_definition_name": constant.dstDefRscName,
            "description": "HTTP source",
            "configuration": {},

        }


        // Cannot create http destination connector of a non-exist user
        check(http.request("POST",
            `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(httpDstConnector), constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] POST /v1alpha/connectors response for creating HTTP destination status is 404`]: (r) => r.status === 404,
        });

    });

}

export function CheckList() {

    group(`Connector API: List destination connectors [with random "jwt-sub" header]`, () => {

        // Cannot list destination connector of a non-exist user
        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors?filter=connector_type=CONNECTOR_TYPE_DATA`, null, constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] GET /v1alpha/connectors response status is 404`]: (r) => r.status === 404,
        });
    });
}

export function CheckGet() {

    group(`Connector API: Get destination connectors by ID [with random "jwt-sub" header]`, () => {

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(csvDstConnector), constant.params)

        http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${csvDstConnector.id}/connect`,
            {}, constant.params)

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/watch`), {
            [`GET /v1alpha/connectors/${resCSVDst.json().connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        // Cannot get a destination connector of a non-exist user
        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}`, null, constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] GET /v1alpha/connectors/${resCSVDst.json().connector.id} response status is 404`]: (r) => r.status === 404,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resCSVDst.json().connector.id} response status 204`]: (r) => r.status === 204,
        });
    });
}

export function CheckUpdate() {

    group(`Connector API: Update destination connectors [with random "jwt-sub" header]`, () => {

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(csvDstConnector), constant.params)

        var csvDstConnectorUpdate = {
            "id": csvDstConnector.id,
            "connector_definition_name": csvDstConnector.connector_definition_name,
            "tombstone": true,
            "description": randomString(50),
            "configuration": {
                destination_path: "/tmp"
            }
        }

        // Cannot patch a destination connector of a non-exist user
        check(http.request(
            "PATCH",
            `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}`,
            JSON.stringify(csvDstConnectorUpdate), constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] PATCH /v1alpha/connectors/${resCSVDst.json().connector.id} response status 404`]: (r) => r.status === 404,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${csvDstConnector.id}`), {
            [`DELETE /v1alpha/connectors/${csvDstConnector.id} response status 204`]: (r) => r.status === 204,
        });
    });
}

export function CheckLookUp() {

    group(`Connector API: Look up destination connectors by UID [with random "jwt-sub" header]`, () => {

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(csvDstConnector), constant.params)

        // Cannot look up a destination connector of a non-exist user
        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.uid}/lookUp`, null, constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] GET /v1alpha/connectors/${resCSVDst.json().connector.uid}/lookUp response status 404`]: (r) => r.status === 404,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resCSVDst.json().connector.id} response status 204`]: (r) => r.status === 204,
        });

    });
}

export function CheckState() {

    group(`Connector API: Change state destination connectors [with random "jwt-sub" header]`, () => {
        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(csvDstConnector), constant.params)

        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/disconnect`, null, constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] POST /v1alpha/connectors/${resCSVDst.json().connector.id}/disconnect response at UNSPECIFIED state status 404`]: (r) => r.status === 404,
        });

        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/connect`, null, constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] POST /v1alpha/connectors/${resCSVDst.json().connector.id}/connect response at UNSPECIFIED state status 404`]: (r) => r.status === 404,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resCSVDst.json().connector.id} response status 204`]: (r) => r.status === 204,
        });

    });

}

export function CheckRename() {

    group(`Connector API: Rename destination connectors [with random "jwt-sub" header]`, () => {

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(csvDstConnector), constant.params)

        // Cannot rename destination connector of a non-exist user
        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/rename`,
            JSON.stringify({
                "new_connector_id": `some-id-not-${resCSVDst.json().connector.id}`
            }), constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] POST /v1alpha/connectors/${resCSVDst.json().connector.id}/rename response status 404`]: (r) => r.status === 404,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${csvDstConnector.id}`), {
            [`DELETE /v1alpha/connectors/${csvDstConnector.id} response status 204`]: (r) => r.status === 204,
        });
    });
}

export function CheckExecute() {

    group(`Connector API: Write destination connectors [with random "jwt-sub" header]`, () => {

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
            JSON.stringify(csvDstConnector), constant.params)

        http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${csvDstConnector.id}/connect`,
            {}, constant.params)

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/watch`), {
            [`[with random "jwt-sub" header] GET /v1alpha/connectors/${resCSVDst.json().connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        // Cannot write to destination connector of a non-exist user
        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/execute`,
            JSON.stringify({
                "inputs": constant.clsModelOutputs
            }), constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] POST /v1alpha/connectors/${resCSVDst.json().connector.id}/execute response status 404 (classification)`]: (r) => r.status === 404,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resCSVDst.json().connector.id} response status 204 (classification)`]: (r) => r.status === 204,
        });
    });
}

export function CheckTest() {

    group(`Connector API: Test destination connectors by ID [with random "jwt-sub" header]`, () => {

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(csvDstConnector), constant.params)

        http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${csvDstConnector.id}/connect`,
            {}, constant.params)

        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/watch`), {
            [`GET /v1alpha/connectors/${resCSVDst.json().connector.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        // Cannot test destination connector of a non-exist user
        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}/testConnection`, null, constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] POST /v1alpha/connectors/${resCSVDst.json().connector.id}/testConnection response status is 404`]: (r) => r.status === 404,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resCSVDst.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resCSVDst.json().connector.id} response status 204`]: (r) => r.status === 204,
        });
    });
}
