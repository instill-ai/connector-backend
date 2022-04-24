import http from "k6/http";
import { check, group } from "k6";

import { deepEqual } from "./helper.js"

export function CheckList() {

    group("Connector API: List source connector definitions", () => {

        check(http.request("GET", `${connectorHost}/connector-source-definitions`), {
            "GET /connector-source-definitions response status is 200": (r) => r.status === 200,
            "GET /connector-source-definitions response body has source_definitions array": (r) => Array.isArray(r.json().source_definitions),
        });

        var allRes = http.request("GET", `${connectorHost}/connector-source-definitions`)
        check(http.request("GET", `${connectorHost}/connector-source-definitions?page_size=0`), {
            "GET /connector-source-definitions?page_size=0 response status is 200": (r) => r.status === 200,
            "GET /connector-source-definitions?page_size=0 response all records": (r) => r.json().source_definitions.length === allRes.json().source_definitions.length,
        });

        check(http.request("GET", `${connectorHost}/connector-source-definitions?page_size=1`), {
            "GET /connector-source-definitions?page_size=1 response status is 200": (r) => r.status === 200,
            "GET /connector-source-definitions?page_size=1 response body source_definitions size 1": (r) => r.json().source_definitions.length === 1,
        });

        var pageRes = http.request("GET", `${connectorHost}/connector-source-definitions?page_size=1`)
        check(http.request("GET", `${connectorHost}/connector-source-definitions?page_size=1&page_cursor=${pageRes.json().next_page_cursor}`), {
            [`GET /connector-source-definitions?page_size=1&page_cursor=${pageRes.json().next_page_cursor} response status is 200`]: (r) => r.status === 200,
            [`GET /connector-source-definitions?page_size=1&page_cursor=${pageRes.json().next_page_cursor} response body source_definitions size 1`]: (r) => r.json().source_definitions.length === 1,
        });

        check(http.request("GET", `${connectorHost}/connector-source-definitions?page_size=1&view=DEFINITION_VIEW_BASIC`), {
            "GET /connector-source-definitions?page_size=1&view=DEFINITION_VIEW_BASIC response status 200": (r) => r.status === 200,
            "GET /connector-source-definitions?page_size=1&view=DEFINITION_VIEW_BASIC response body source_definitions has no spec": (r) => r.json().source_definitions[0].spec === undefined,
        });

        check(http.request("GET", `${connectorHost}/connector-source-definitions?page_size=1&view=DEFINITION_VIEW_FULL`), {
            "GET /connector-source-definitions?page_size=1&view=DEFINITION_VIEW_FULL response status 200": (r) => r.status === 200,
            "GET /connector-source-definitions?page_size=1&view=DEFINITION_VIEW_FULL response body source_definitions has spec": (r) => r.json().source_definitions[0].spec !== undefined,
        });

        check(http.request("GET", `${connectorHost}/connector-source-definitions?page_size=1`), {
            "GET /connector-source-definitions?page_size=1 response status 200": (r) => r.status === 200,
            "GET /connector-source-definitions?page_size=1 response body source_definitions has no spec": (r) => r.json().source_definitions[0].spec === undefined,
        });
    });
}

export function CheckGet() {
    group("Connector API: Get source connector definition", () => {
        var allRes = http.request("GET", `${connectorHost}/connector-source-definitions`)
        var def = allRes.json().source_definitions[0]
        check(http.request("GET", `${connectorHost}/connector-source-definitions/${def.source_definition_id}`), {
            [`GET ${connectorHost}/connector-source-definitions/${def.source_definition_id} response status is 200`]: (r) => r.status === 200,
            [`GET ${connectorHost}/connector-source-definitions/${def.source_definition_id} response has the exact record`]: (r) => deepEqual(r.json().source_definition, def),
        });

        check(http.request("GET", `${connectorHost}/connector-source-definitions/${def.source_definition_id}?view=DEFINITION_VIEW_BASIC`), {
            [`GET /connector-source-definitions/${def.source_definition_id}?view=DEFINITION_VIEW_BASIC response status 200`]: (r) => r.status === 200,
            [`GET /connector-source-definitions/${def.source_definition_id}?view=DEFINITION_VIEW_BASIC response body source_definition has no spec`]: (r) => r.json().source_definition.spec === undefined,
        });

        check(http.request("GET", `${connectorHost}/connector-source-definitions/${def.source_definition_id}?view=DEFINITION_VIEW_FULL`), {
            [`GET /connector-source-definitions/${def.source_definition_id}?view=DEFINITION_VIEW_FULL response status 200`]: (r) => r.status === 200,
            [`GET /connector-source-definitions/${def.source_definition_id}?view=DEFINITION_VIEW_FULL response body source_definition has spec`]: (r) => r.json().source_definition.spec !== undefined,
        });

        check(http.request("GET", `${connectorHost}/connector-source-definitions/${def.source_definition_id}`), {
            [`GET /connector-source-definitions/${def.source_definition_id} response status 200`]: (r) => r.status === 200,
            [`GET /connector-source-definitions/${def.source_definition_id} response body source_definition has no spec`]: (r) => r.json().source_definition.spec === undefined,
        });
    });
}
