import http from "k6/http";
import { check, group } from "k6";

import { deepEqual } from "./helper.js"

export function CheckList() {

    group("Connector API: List destination connector definitions", () => {

        check(http.request("GET", `${connectorHost}/connector-destination-definitions`), {
            "GET /connector-destination-definitions response status 200": (r) => r.status === 200,
            "GET /connector-destination-definitions response body has destination_definitions array": (r) => Array.isArray(r.json().destination_definitions),
        });

        var allRes = http.request("GET", `${connectorHost}/connector-destination-definitions`)
        check(http.request("GET", `${connectorHost}/connector-destination-definitions?page_size=0`), {
            "GET /connector-destination-definitions?page_size=0 response status 200": (r) => r.status === 200,
            "GET /connector-destination-definitions?page_size=0 response all records": (r) => r.json().destination_definitions.length === allRes.json().destination_definitions.length,
        });

        check(http.request("GET", `${connectorHost}/connector-destination-definitions?page_size=1`), {
            "GET /connector-destination-definitions?page_size=1 response status 200": (r) => r.status === 200,
            "GET /connector-destination-definitions?page_size=1 response body destination_definitions size 1": (r) => r.json().destination_definitions.length === 1,
        });

        var pageRes = http.request("GET", `${connectorHost}/connector-destination-definitions?page_size=1`)
        check(http.request("GET", `${connectorHost}/connector-destination-definitions?page_size=1&page_cursor=${pageRes.json().next_page_cursor}`), {
            [`GET /connector-destination-definitions?page_size=1&page_cursor=${pageRes.json("next_page_cursor")} response status 200`]: (r) => r.status === 200,
            [`GET /connector-destination-definitions?page_size=1&page_cursor=${pageRes.json("next_page_cursor")} response body destination_definitions size 1`]: (r) => r.json().destination_definitions.length === 1,
        });

        check(http.request("GET", `${connectorHost}/connector-destination-definitions?page_size=1&view=DEFINITION_VIEW_BASIC`), {
            "GET /connector-destination-definitions?page_size=1&view=DEFINITION_VIEW_BASIC response status 200": (r) => r.status === 200,
            "GET /connector-destination-definitions?page_size=1&view=DEFINITION_VIEW_BASIC response body destination_definitions has no spec": (r) => r.json().destination_definitions[0].spec === undefined,
        });

        check(http.request("GET", `${connectorHost}/connector-destination-definitions?page_size=1&view=DEFINITION_VIEW_FULL`), {
            "GET /connector-destination-definitions?page_size=1&view=DEFINITION_VIEW_FULL response status 200": (r) => r.status === 200,
            "GET /connector-destination-definitions?page_size=1&view=DEFINITION_VIEW_FULL response body destination_definitions has spec": (r) => r.json().destination_definitions[0].spec !== undefined,
        });

        check(http.request("GET", `${connectorHost}/connector-destination-definitions?page_size=1`), {
            "GET /connector-destination-definitions?page_size=1 response status 200": (r) => r.status === 200,
            "GET /connector-destination-definitions?page_size=1 response body destination_definitions has no spec": (r) => r.json().destination_definitions[0].spec === undefined,
        });
    });
}

export function CheckGet() {
    group("Connector API: Get destination connector definition", () => {
        var allRes = http.request("GET", `${connectorHost}/connector-destination-definitions`)
        var def = allRes.json().destination_definitions[0]
        check(http.request("GET", `${connectorHost}/connector-destination-definitions/${def.destination_definition_id}`), {
            [`GET ${connectorHost}/connector-destination-definitions/${def.destination_definition_id} response status 200`]: (r) => r.status === 200,
            [`GET ${connectorHost}/connector-destination-definitions/${def.destination_definition_id} response has the exact record`]: (r) => deepEqual(r.json().destination_definition, def),
        });

        check(http.request("GET", `${connectorHost}/connector-destination-definitions/${def.destination_definition_id}?view=DEFINITION_VIEW_BASIC`), {
            [`GET /connector-destination-definitions/${def.destination_definition_id}?view=DEFINITION_VIEW_BASIC response status 200`]: (r) => r.status === 200,
            [`GET /connector-destination-definitions/${def.destination_definition_id}?view=DEFINITION_VIEW_BASIC response body destination_definition has no spec`]: (r) => r.json().destination_definition.spec === undefined,
        });

        check(http.request("GET", `${connectorHost}/connector-destination-definitions/${def.destination_definition_id}?view=DEFINITION_VIEW_FULL`), {
            [`GET /connector-destination-definitions/${def.destination_definition_id}?view=DEFINITION_VIEW_FULL response status 200`]: (r) => r.status === 200,
            [`GET /connector-destination-definitions/${def.destination_definition_id}?view=DEFINITION_VIEW_FULL response body destination_definition has spec`]: (r) => r.json().destination_definition.spec !== undefined,
        });

        check(http.request("GET", `${connectorHost}/connector-destination-definitions/${def.destination_definition_id}`), {
            [`GET /connector-destination-definitions/${def.destination_definition_id} response status 200`]: (r) => r.status === 200,
            [`GET /connector-destination-definitions/${def.destination_definition_id} response body destination_definition has no spec`]: (r) => r.json().destination_definition.spec === undefined,
        });
    });
}
