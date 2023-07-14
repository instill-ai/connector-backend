import http from "k6/http";
import { sleep, check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import { connectorPublicHost, modelPublicHost, pipelinePublicHost } from "./const.js"

import * as constant from "./const.js"
import * as helper from "./helper.js"

export function CheckCreate() {

    group(`Connector API: Create source connectors [with random "jwt-sub" header]`, () => {

        var srcConnector = {
            "id": "trigger",
            "connector_definition_name": constant.srcDefRscName,
            "description": "HTTP source",
            "configuration": {},
        }


        check(http.request("POST",
            `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(srcConnector), constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] POST /v1alpha/connectors response status for HTTP source connector is 404`]: (r) => r.status === 404,
        });

    });
}

export function CheckList() {

    group(`Connector API: List source connectors [with random "jwt-sub" header]`, () => {

        // Cannot list source connector of a non-exist user
        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors?filter=connector_type=CONNECTOR_TYPE_SOURCE`, null, constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] GET /v1alpha/connectors response status is 404`]: (r) => r.status === 404,
        });
    });
}

export function CheckGet() {

    group(`Connector API: Get source connectors by ID [with random "jwt-sub" header]`, () => {

        var srcConnector = {
            "id": "trigger",
            "connector_definition_name": constant.srcDefRscName,
            "configuration": {}
        }

        var resHTTP = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(srcConnector), constant.params)

        // Cannot get a source connector of a non-exist user
        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors/${resHTTP.json().connector.id}`, null, constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] GET /v1alpha/connectors/${resHTTP.json().connector.id} response status is 404`]: (r) => r.status === 404,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resHTTP.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resHTTP.json().connector.id} response status 204`]: (r) => r.status === 204,
        });

    });
}

export function CheckUpdate() {

    group(`Connector API: Update source connectors [with random "jwt-sub" header]`, () => {

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

        srcConnector.description = randomString(20)

        // Cannot patch a source connector of a non-exist user
        check(http.request(
            "PATCH",
            `${connectorPublicHost}/v1alpha/connectors/${srcConnector.id}`,
            JSON.stringify(srcConnector), constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] PATCH /v1alpha/connectors/${srcConnector.id} response status for updating gRPC source connector 404`]: (r) => r.status === 404,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${srcConnector.id}`), {
            [`DELETE /v1alpha/connectors/${srcConnector.id} response status 204`]: (r) => r.status === 204,
        });

    });

}

export function CheckDelete() {

    group(`Connector API: Delete source connectors [with random "jwt-sub" header]`, () => {

        // Cannot delete source connector of a non-exist user
        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/trigger`, null, constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] DELETE /v1alpha/connectors/trigger response status 404`]: (r) => r.status === 404,
        });

        // Cannot delete destination connector of a non-exist user
        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/response`, null, constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] DELETE /v1alpha/connectors/response response status 404`]: (r) => r.status === 404,
        });
    });
}

export function CheckLookUp() {

    group(`Connector API: Look up source connectors by UID [with random "jwt-sub" header]`, () => {

        var srcConnector = {
            "id": "trigger",
            "connector_definition_name": constant.srcDefRscName,
            "configuration": {}
        }

        var resHTTP = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(srcConnector), constant.params)

        // Cannot look up source connector of a non-exist user
        check(http.request("GET", `${connectorPublicHost}/v1alpha/connectors/${resHTTP.json().connector.uid}/lookUp`, null, constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] GET /v1alpha/connectors/${resHTTP.json().connector.uid}/lookUp response status 404`]: (r) => r.status === 404,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resHTTP.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resHTTP.json().connector.id} response status 204`]: (r) => r.status === 204,
        });

    });
}

export function CheckState() {

    group(`Connector API: Change state source connectors [with random "jwt-sub" header]`, () => {
        var srcConnector = {
            "id": "trigger",
            "connector_definition_name": constant.srcDefRscName,
            "configuration": {}
        }

        var resHTTP = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(srcConnector), constant.params)

        // Cannot connect source connector of a non-exist user
        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resHTTP.json().connector.id}/connect`, null, constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] POST /v1alpha/connectors/${resHTTP.json().connector.id}/connect response status 404`]: (r) => r.status === 404,
        });

        // Cannot disconnect source connector of a non-exist user
        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resHTTP.json().connector.id}/disconnect`, null, constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] POST /v1alpha/connectors/${resHTTP.json().connector.id}/disconnect response status 404`]: (r) => r.status === 404,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resHTTP.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resHTTP.json().connector.id} response status 204`]: (r) => r.status === 204,
        });

    });

}

export function CheckRename() {

    group(`Connector API: Rename source connectors [with random "jwt-sub" header]`, () => {
        var srcConnector = {
            "id": "trigger",
            "connector_definition_name": constant.srcDefRscName,
            "configuration": {}
        }

        var resHTTP = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(srcConnector), constant.params)

        // Cannot rename source connector of a non-exist user
        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resHTTP.json().connector.id}/rename`,
            JSON.stringify({
                "new_connector_id": "some-id-not-http"
            }), constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] POST /v1alpha/connectors/${resHTTP.json().connector.id}/rename response status 404`]: (r) => r.status === 404,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${srcConnector.id}`), {
            [`DELETE /v1alpha/connectors/${srcConnector.id} response status 204`]: (r) => r.status === 204,
        });
    });

}

export function CheckTest() {

    group(`Connector API: Test source connectors by ID [with random "jwt-sub" header]`, () => {

        var srcConnector = {
            "id": "trigger",
            "connector_definition_name": constant.srcDefRscName,
            "configuration": {}
        }

        var resHTTP = http.request("POST", `${connectorPublicHost}/v1alpha/connectors`,
            JSON.stringify(srcConnector), constant.params)

        // Cannot test source connector of a non-exist user
        check(http.request("POST", `${connectorPublicHost}/v1alpha/connectors/${resHTTP.json().connector.id}/testConnection`, null, constant.paramsHTTPWithJwt), {
            [`[with random "jwt-sub" header] POST /v1alpha/connectors/${resHTTP.json().connector.id}/testConnection response status is 404`]: (r) => r.status === 404,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/connectors/${resHTTP.json().connector.id}`), {
            [`DELETE /v1alpha/connectors/${resHTTP.json().connector.id} response status 204`]: (r) => r.status === 204,
        });

    });
}
