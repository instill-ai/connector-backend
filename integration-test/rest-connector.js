import http from "k6/http";
import { check, group } from "k6";

import * as constant from "./const.js"
import * as helper from "./helper.js"

export function CheckCreate() {

    group("Connector API: Create connectors", () => {

        var dirHTTPSrcConnector = {
            "connector_definition_id": constant.httpSrcDefinitionId,
            "connector_type": "CONNECTOR_TYPE_SOURCE"
        }

        var dirHTTPDstConnector = {
            "connector_definition_id": constant.httpDstDefinitionId,
            "connector_type": "CONNECTOR_TYPE_DESTINATION"
        }

        var dirGRPCSrcConnector = {
            "connector_definition_id": constant.gRPCSrcDefinitionId,
            "connector_type": "CONNECTOR_TYPE_SOURCE"
        }

        var dirGRPCDstConnector = {
            "connector_definition_id": constant.gRPCDstDefinitionId,
            "connector_type": "CONNECTOR_TYPE_DESTINATION"
        }

        var resSrcHTTP = http.request(
            "POST",
            `${connectorHost}/connectors`,
            JSON.stringify(dirHTTPSrcConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(resSrcHTTP, {
            "POST /connectors response status for creating directness HTTP source connector 201": (r) => r.status === 201,
            "POST /connectors response body connector id": (r) => helper.isUUID(r.json().connector.id),
            "POST /connectors response body connector connector_definition_id": (r) => r.json().connector.connector_definition_id === constant.httpSrcDefinitionId
        });

        check(http.request(
            "POST",
            `${connectorHost}/connectors`,
            JSON.stringify(dirHTTPSrcConnector), {
            headers: { "Content-Type": "application/json" },
        }), {
            "POST /connectors response duplicate directness connector status 409": (r) => r.status === 409
        });

        var resDstHTTP = http.request(
            "POST",
            `${connectorHost}/connectors`,
            JSON.stringify(dirHTTPDstConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(resDstHTTP, {
            "POST /connectors response status for creating directness HTTP destination connector 201": (r) => r.status === 201,
        });

        var resSrcGRPC = http.request(
            "POST",
            `${connectorHost}/connectors`,
            JSON.stringify(dirGRPCSrcConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(resSrcGRPC, {
            "POST /connectors response status for creating directness gRPC source connector 201": (r) => r.status === 201,
        });

        var resDstGRPC = http.request(
            "POST",
            `${connectorHost}/connectors`,
            JSON.stringify(dirGRPCDstConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(resDstGRPC, {
            "POST /connectors response status 201": (r) => r.status === 201,
        });

        // Delete test records
        check(http.request("DELETE", `${connectorHost}/connectors/${resSrcHTTP.json().connector.name}?connector_type=CONNECTOR_TYPE_SOURCE`), {
            [`DELETE /connectors/${resSrcHTTP.json().connector.name}?connector_type=CONNECTOR_TYPE_SOURCE response status 204`]: (r) => r.status === 204,
        });
        check(http.request("DELETE", `${connectorHost}/connectors/${resDstHTTP.json().connector.name}?connector_type=CONNECTOR_TYPE_DESTINATION`), {
            [`DELETE /connectors/${resDstHTTP.json().connector.name}?connector_type=CONNECTOR_TYPE_DESTINATION response status 204`]: (r) => r.status === 204,
        });
        check(http.request("DELETE", `${connectorHost}/connectors/${resSrcGRPC.json().connector.name}?connector_type=CONNECTOR_TYPE_SOURCE`), {
            [`DELETE /connectors/${resSrcGRPC.json().connector.name}?connector_type=CONNECTOR_TYPE_SOURCE response status 204`]: (r) => r.status === 204,
        });
        check(http.request("DELETE", `${connectorHost}/connectors/${resDstGRPC.json().connector.name}?connector_type=CONNECTOR_TYPE_DESTINATION`), {
            [`DELETE /connectors/${resDstGRPC.json().connector.name}?connector_type=CONNECTOR_TYPE_DESTINATION response status 204`]: (r) => r.status === 204,
        });
    });

}

export function CheckUpdate() {

    group("Connector API: Update connectors", () => {

        var csvDstConnector = {
            "connector_definition_id": constant.csvDstDefinitionId,
            "connector_type": "CONNECTOR_TYPE_DESTINATION",
            "name": "origin",
            "description": "this is the original connector",
            "configuration": constant.csvDstConfig
        }

        var res = http.request(
            "POST",
            `${connectorHost}/connectors`,
            JSON.stringify(csvDstConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(res, {
            "POST /connectors response status 201": (r) => r.status === 201,
            "POST /connectors response body connector id": (r) => helper.isUUID(r.json().connector.id),
            "POST /connectors response body connector connector_definition_id": (r) => r.json().connector.connector_definition_id === constant.csvDstDefinitionId
        });

        var csvDstConnectorUpdate = {
            "name": "updated",
            "description": "this is the updated connector",
            "tombstone": true,
            "configuration": constant.csvDstConfig
        }

        csvDstConnectorUpdate.configuration.connection_specification.destination_path = "/tmp"

        check(http.request(
            "PATCH",
            `${connectorHost}/connectors/${res.json().connector.name}?connector_type=CONNECTOR_TYPE_DESTINATION`,
            JSON.stringify(csvDstConnectorUpdate), {
            headers: { "Content-Type": "application/json" },
        }), {
            [`PATCH /connectors/${res.json().connector.name}?connector_type=CONNECTOR_TYPE_DESTINATION response status 200`]: (r) => r.status === 200,
            [`PATCH /connectors/${res.json().connector.name}?connector_type=CONNECTOR_TYPE_DESTINATION response body connector id`]: (r) => helper.isUUID(r.json().connector.id),
            [`PATCH /connectors/${res.json().connector.name}?connector_type=CONNECTOR_TYPE_DESTINATION response body connector connector_definition_id`]: (r) => r.json().connector.connector_definition_id === constant.csvDstDefinitionId,
            [`PATCH /connectors/${res.json().connector.name}?connector_type=CONNECTOR_TYPE_DESTINATION response body connector name`]: (r) => r.json().connector.name === csvDstConnectorUpdate.name,
            [`PATCH /connectors/${res.json().connector.name}?connector_type=CONNECTOR_TYPE_DESTINATION response body connector description`]: (r) => r.json().connector.description === csvDstConnectorUpdate.description,
            [`PATCH /connectors/${res.json().connector.name}?connector_type=CONNECTOR_TYPE_DESTINATION response body connector tombstone`]: (r) => r.json().connector.tombstone === csvDstConnectorUpdate.tombstone,
            [`PATCH /connectors/${res.json().connector.name}?connector_type=CONNECTOR_TYPE_DESTINATION response body connector configuration`]: (r) => r.json().connector.configuration.connection_specification.destination_path === csvDstConnectorUpdate.configuration.connection_specification.destination_path
        });

        check(http.request("DELETE", `${connectorHost}/connectors/${csvDstConnectorUpdate.name}?connector_type=CONNECTOR_TYPE_DESTINATION`), {
            [`DELETE /connectors/${csvDstConnectorUpdate.name}?connector_type=CONNECTOR_TYPE_DESTINATION response status 204`]: (r) => r.status === 204,
        });
    });
}

export function CheckGet() {

    group("Connector API: Get connectors", () => {

        var dirHTTPSrcConnector = {
            "connector_definition_id": constant.httpSrcDefinitionId,
            "connector_type": "CONNECTOR_TYPE_SOURCE"
        }

        var resHTTP = http.request(
            "POST",
            `${connectorHost}/connectors`,
            JSON.stringify(dirHTTPSrcConnector), {
            headers: { "Content-Type": "application/json" },
        })

        var csvDstConnector = {
            "connector_definition_id": constant.csvDstDefinitionId,
            "connector_type": "CONNECTOR_TYPE_DESTINATION",
            "name": "csv-dst-connector",
            "configuration": constant.csvDstConfig
        }

        var resCSV = http.request(
            "POST",
            `${connectorHost}/connectors`,
            JSON.stringify(csvDstConnector), {
            headers: { "Content-Type": "application/json" },
        })

        check(http.request("GET", `${connectorHost}/connectors/${resHTTP.json().connector.name}?connector_type=CONNECTOR_TYPE_SOURCE`), {
            [`GET /connectors/${resHTTP.json().connector.name}?connector_type=CONNECTOR_TYPE_SOURCE response status 200`]: (r) => r.status === 200,
            [`GET /connectors/${resHTTP.json().connector.name}?connector_type=CONNECTOR_TYPE_SOURCE response body connector id`]: (r) => helper.isUUID(r.json().connector.id),
            [`GET /connectors/${resHTTP.json().connector.name}?connector_type=CONNECTOR_TYPE_SOURCE response body connector connector_definition_id`]: (r) => r.json().connector.connector_definition_id === constant.httpSrcDefinitionId,
            [`GET /connectors/${resHTTP.json().connector.name}?connector_type=CONNECTOR_TYPE_SOURCE response body connector full name`]: (r) => r.json().connector.full_name === resHTTP.json().connector.full_name,
        });

        check(http.request("GET", `${connectorHost}/connectors/${resCSV.json().connector.name}?connector_type=CONNECTOR_TYPE_DESTINATION`), {
            [`GET /connectors/${resCSV.json().connector.name}?connector_type=CONNECTOR_TYPE_DESTINATION response status 200`]: (r) => r.status === 200,
            [`GET /connectors/${resCSV.json().connector.name}?connector_type=CONNECTOR_TYPE_DESTINATION response body connector id`]: (r) => helper.isUUID(r.json().connector.id),
            [`GET /connectors/${resCSV.json().connector.name}?connector_type=CONNECTOR_TYPE_DESTINATION response body connector connector_definition_id`]: (r) => r.json().connector.connector_definition_id === constant.csvDstDefinitionId,
            [`GET /connectors/${resCSV.json().connector.name}?connector_type=CONNECTOR_TYPE_DESTINATION response body connector full name`]: (r) => r.json().connector.full_name === resCSV.json().connector.full_name,
        });

        check(http.request("DELETE", `${connectorHost}/connectors/${resHTTP.json().connector.name}?connector_type=CONNECTOR_TYPE_SOURCE`), {
            [`DELETE /connectors/${resHTTP.json().name}?connector_type=CONNECTOR_TYPE_SOURCE response status 204`]: (r) => r.status === 204,
        });

        check(http.request("DELETE", `${connectorHost}/connectors/${resCSV.json().connector.name}?connector_type=CONNECTOR_TYPE_DESTINATION`), {
            [`DELETE /connectors/${resCSV.json().name}?connector_type=CONNECTOR_TYPE_DESTINATION response status 204`]: (r) => r.status === 204,
        });
    });
}
