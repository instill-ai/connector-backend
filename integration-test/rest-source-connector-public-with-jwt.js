import http from "k6/http";
import { sleep, check, group } from "k6";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import { connectorPublicHost, modelPublicHost, pipelinePublicHost } from "./const.js"

import * as constant from "./const.js"
import * as helper from "./helper.js"

export function CheckCreate() {

    group(`Connector API: Create source connectors [with random "jwt-sub" header]`, () => {

        var httpSrcConnector = {
            "id": "source-http",
            "source_connector_definition": constant.httpSrcDefRscName,
            "connector": {
                "description": "HTTP source",
                "configuration": {},
            }
        }

        var gRPCSrcConnector = {
            "id": "source-grpc",
            "source_connector_definition": constant.gRPCSrcDefRscName,
            "connector": {
                "description": "gRPC source",
                "configuration": {},
            }
        }

        check(http.request("POST",
            `${connectorPublicHost}/v1alpha/source-connectors`,
            JSON.stringify(httpSrcConnector), constant.paramsWithJwt), {
            [`[with random "jwt-sub" header] POST /v1alpha/source-connectors response status for HTTP source connector is 500`]: (r) => r.status === 500,
        });

        // Cannot create grpc source connector of a non-exist user
        check(http.request("POST",
            `${connectorPublicHost}/v1alpha/source-connectors`,
            JSON.stringify(gRPCSrcConnector), constant.paramsWithJwt), {
            [`[with random "jwt-sub" header] POST /v1alpha/source-connectors response status for gRPC source connector is 500`]: (r) => r.status === 500,
        });
    });
}

export function CheckList() {

    group(`Connector API: List source connectors [with random "jwt-sub" header]`, () => {

        // Cannot list source connector of a non-exist user
        check(http.request("GET", `${connectorPublicHost}/v1alpha/source-connectors`, null, constant.paramsWithJwt), {
            [`[with random "jwt-sub" header] GET /v1alpha/source-connectors response status is 500`]: (r) => r.status === 500,
        });
    });
}

export function CheckGet() {

    group(`Connector API: Get source connectors by ID [with random "jwt-sub" header]`, () => {

        var httpSrcConnector = {
            "id": "source-http",
            "source_connector_definition": constant.httpSrcDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        var resHTTP = http.request("POST", `${connectorPublicHost}/v1alpha/source-connectors`,
            JSON.stringify(httpSrcConnector), constant.params)

        // Cannot get a source connector of a non-exist user
        check(http.request("GET", `${connectorPublicHost}/v1alpha/source-connectors/${resHTTP.json().source_connector.id}`, null, constant.paramsWithJwt), {
            [`[with random "jwt-sub" header] GET /v1alpha/source-connectors/${resHTTP.json().source_connector.id} response status is 500`]: (r) => r.status === 500,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/source-connectors/${resHTTP.json().source_connector.id}`), {
            [`DELETE /v1alpha/source-connectors/${resHTTP.json().source_connector.id} response status 204`]: (r) => r.status === 204,
        });

    });
}

export function CheckUpdate() {

    group(`Connector API: Update source connectors [with random "jwt-sub" header]`, () => {

        var gRPCSrcConnector = {
            "id": "source-grpc",
            "source_connector_definition": constant.gRPCSrcDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        check(http.request(
            "POST",
            `${connectorPublicHost}/v1alpha/source-connectors`,
            JSON.stringify(gRPCSrcConnector), constant.params), {
            "POST /v1alpha/source-connectors response status for creating gRPC source connector 201": (r) => r.status === 201,
        });

        gRPCSrcConnector.connector.description = randomString(20)

        // Cannot patch a source connector of a non-exist user
        check(http.request(
            "PATCH",
            `${connectorPublicHost}/v1alpha/source-connectors/${gRPCSrcConnector.id}`,
            JSON.stringify(gRPCSrcConnector), constant.paramsWithJwt), {
            [`[with random "jwt-sub" header] PATCH /v1alpha/source-connectors/${gRPCSrcConnector.id} response status for updating gRPC source connector 500`]: (r) => r.status === 500,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/source-connectors/${gRPCSrcConnector.id}`), {
            [`DELETE /v1alpha/source-connectors/${gRPCSrcConnector.id} response status 204`]: (r) => r.status === 204,
        });

    });

}

export function CheckDelete() {

    group(`Connector API: Delete source connectors [with random "jwt-sub" header]`, () => {

        // Cannot delete source connector of a non-exist user
        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/source-connectors/source-http`, null, constant.paramsWithJwt), {
            [`[with random "jwt-sub" header] DELETE /v1alpha/source-connectors/source-http response status 500`]: (r) => r.status === 500,
        });

        // Cannot delete destination connector of a non-exist user
        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/destination-connectors/destination-http`, null, constant.paramsWithJwt), {
            [`[with random "jwt-sub" header] DELETE /v1alpha/destination-connectors/destination-http response status 500`]: (r) => r.status === 500,
        });
    });
}

export function CheckLookUp() {

    group(`Connector API: Look up source connectors by UID [with random "jwt-sub" header]`, () => {

        var httpSrcConnector = {
            "id": "source-http",
            "source_connector_definition": constant.httpSrcDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        var resHTTP = http.request("POST", `${connectorPublicHost}/v1alpha/source-connectors`,
            JSON.stringify(httpSrcConnector), constant.params)

        // Cannot look up source connector of a non-exist user
        check(http.request("GET", `${connectorPublicHost}/v1alpha/source-connectors/${resHTTP.json().source_connector.uid}/lookUp`, null, constant.paramsWithJwt), {
            [`[with random "jwt-sub" header] GET /v1alpha/source-connectors/${resHTTP.json().source_connector.uid}/lookUp response status 500`]: (r) => r.status === 500,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/source-connectors/${resHTTP.json().source_connector.id}`), {
            [`DELETE /v1alpha/source-connectors/${resHTTP.json().source_connector.id} response status 204`]: (r) => r.status === 204,
        });

    });
}

export function CheckState() {

    group(`Connector API: Change state source connectors [with random "jwt-sub" header]`, () => {
        var httpSrcConnector = {
            "id": "source-http",
            "source_connector_definition": constant.httpSrcDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        var resHTTP = http.request("POST", `${connectorPublicHost}/v1alpha/source-connectors`,
            JSON.stringify(httpSrcConnector), constant.params)

        // Cannot connect source connector of a non-exist user
        check(http.request("POST", `${connectorPublicHost}/v1alpha/source-connectors/${resHTTP.json().source_connector.id}/connect`, null, constant.paramsWithJwt), {
            [`[with random "jwt-sub" header] POST /v1alpha/source-connectors/${resHTTP.json().source_connector.id}/connect response status 500`]: (r) => r.status === 500,
        });

        // Cannot disconnect source connector of a non-exist user
        check(http.request("POST", `${connectorPublicHost}/v1alpha/source-connectors/${resHTTP.json().source_connector.id}/disconnect`, null, constant.paramsWithJwt), {
            [`[with random "jwt-sub" header] POST /v1alpha/source-connectors/${resHTTP.json().source_connector.id}/disconnect response status 500`]: (r) => r.status === 500,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/source-connectors/${resHTTP.json().source_connector.id}`), {
            [`DELETE /v1alpha/source-connectors/${resHTTP.json().source_connector.id} response status 204`]: (r) => r.status === 204,
        });

    });

}

export function CheckRename() {

    group(`Connector API: Rename source connectors [with random "jwt-sub" header]`, () => {
        var httpSrcConnector = {
            "id": "source-http",
            "source_connector_definition": constant.httpSrcDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        var resHTTP = http.request("POST", `${connectorPublicHost}/v1alpha/source-connectors`,
            JSON.stringify(httpSrcConnector), constant.params)

        // Cannot rename source connector of a non-exist user
        check(http.request("POST", `${connectorPublicHost}/v1alpha/source-connectors/${resHTTP.json().source_connector.id}/rename`,
            JSON.stringify({
                "new_source_connector_id": "some-id-not-http"
            }), constant.paramsWithJwt), {
            [`[with random "jwt-sub" header] POST /v1alpha/source-connectors/${resHTTP.json().source_connector.id}/rename response status 500`]: (r) => r.status === 500,
        });

        check(http.request("DELETE", `${connectorPublicHost}/v1alpha/source-connectors/${httpSrcConnector.id}`), {
            [`DELETE /v1alpha/source-connectors/${httpSrcConnector.id} response status 204`]: (r) => r.status === 204,
        });
    });

}
