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
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig,
        }


        // Cannot create http destination connector of a non-exist user
        check(http.request("POST",
            `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources`,
            JSON.stringify(httpDstConnector), constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] POST /v1alpha/${constant.namespace}/connector-resources response for creating HTTP destination status is 401`]: (r) => r.status === 401,
        });

    });

}

export function CheckList() {

    group(`Connector API: List destination connectors [with random "jwt-sub" header]`, () => {

        // Cannot list destination connector of a non-exist user
        check(http.request("GET", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources?filter=connector_type=CONNECTOR_TYPE_DATA`, null, constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] GET /v1alpha/${constant.namespace}/connector-resources response status is 401`]: (r) => r.status === 401,
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

        var resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources`,
            JSON.stringify(csvDstConnector), constant.params)

        http.request("POST", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources/${csvDstConnector.id}/connect`,
            {}, constant.params)

        check(http.request("GET", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id}/watch`), {
            [`GET /v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        // Cannot get a destination connector of a non-exist user
        check(http.request("GET", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id}`, null, constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] GET /v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id} response status is 401`]: (r) => r.status === 401,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id}`), {
            [`DELETE /v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id} response status 204`]: (r) => r.status === 204,
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

        var resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources`,
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
            `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id}`,
            JSON.stringify(csvDstConnectorUpdate), constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] PATCH /v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id} response status 401`]: (r) => r.status === 401,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources/${csvDstConnector.id}`), {
            [`DELETE /v1alpha/${constant.namespace}/connector-resources/${csvDstConnector.id} response status 204`]: (r) => r.status === 204,
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

        var resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources`,
            JSON.stringify(csvDstConnector), constant.params)

        // Cannot look up a destination connector of a non-exist user
        check(http.request("GET", `${connectorPublicHost}/v1alpha/connector-resources/${resCSVDst.json().connector_resource.uid}/lookUp`, null, constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] GET /v1alpha/connector-resources/${resCSVDst.json().connector_resource.uid}/lookUp response status 401`]: (r) => r.status === 401,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id}`), {
            [`DELETE /v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id} response status 204`]: (r) => r.status === 204,
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

        var resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources`,
            JSON.stringify(csvDstConnector), constant.params)

        check(http.request("POST", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id}/disconnect`, null, constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] POST /v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id}/disconnect response at UNSPECIFIED state status 401`]: (r) => r.status === 401,
        });

        check(http.request("POST", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id}/connect`, null, constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] POST /v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id}/connect response at UNSPECIFIED state status 401`]: (r) => r.status === 401,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id}`), {
            [`DELETE /v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id} response status 204`]: (r) => r.status === 204,
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

        var resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources`,
            JSON.stringify(csvDstConnector), constant.params)

        // Cannot rename destination connector of a non-exist user
        check(http.request("POST", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id}/rename`,
            JSON.stringify({
                "new_connector_id": `some-id-not-${resCSVDst.json().connector_resource.id}`
            }), constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] POST /v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id}/rename response status 401`]: (r) => r.status === 401,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources/${csvDstConnector.id}`), {
            [`DELETE /v1alpha/${constant.namespace}/connector-resources/${csvDstConnector.id} response status 204`]: (r) => r.status === 204,
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

        resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources`,
            JSON.stringify(csvDstConnector), constant.params)

        http.request("POST", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources/${csvDstConnector.id}/connect`,
            {}, constant.params)

        check(http.request("GET", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id}/watch`), {
            [`[with random "jwt-sub" header] GET /v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        // Cannot write to destination connector of a non-exist user
        check(http.request("POST", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id}/execute`,
            JSON.stringify({
                "inputs": constant.clsModelOutputs
            }), constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] POST /v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id}/execute response status 401 (classification)`]: (r) => r.status === 401,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id}`), {
            [`DELETE /v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id} response status 204 (classification)`]: (r) => r.status === 204,
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

        var resCSVDst = http.request("POST", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources`,
            JSON.stringify(csvDstConnector), constant.params)

        http.request("POST", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources/${csvDstConnector.id}/connect`,
            {}, constant.params)

        check(http.request("GET", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id}/watch`), {
            [`GET /v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id}/watch response connector state is STATE_CONNECTED`]: (r) => r.json().state === "STATE_CONNECTED",
        })

        // Cannot test destination connector of a non-exist user
        check(http.request("POST", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id}/testConnection`, null, constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] POST /v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id}/testConnection response status is 401`]: (r) => r.status === 401,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id}`), {
            [`DELETE /v1alpha/${constant.namespace}/connector-resources/${resCSVDst.json().connector_resource.id} response status 204`]: (r) => r.status === 204,
        });
    });
}
