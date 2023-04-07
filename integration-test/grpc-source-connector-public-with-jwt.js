import grpc from 'k6/net/grpc';
import http from "k6/http";
import {
    sleep,
    check,
    group
} from "k6";
import {
    randomString
} from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js"
import * as helper from "./helper.js"

const client = new grpc.Client();
client.load(['proto/vdp/connector/v1alpha'], 'connector_public_service.proto');

export function CheckCreate() {

    group(`Connector API: Create source connector [with random "jwt-sub" header]`, () => {
        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

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

        // Cannot create source connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector', {
            source_connector: httpSrcConnector
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector create HTTP source response StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
        })

        // Cannot create source connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector', {
            source_connector: gRPCSrcConnector
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector create gRPC source response StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
        })

        client.close();
    });
}

export function CheckList() {

    group(`Connector API: List source connectors [with random "jwt-sub" header]`, () => {
        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        // Cannot list source connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors', {}, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/ListSourceConnectors response StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
        })

        client.close();
    });
}

export function CheckGet() {

    group(`Connector API: Get source connectors by ID [with random "jwt-sub" header]`, () => {
        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var httpSrcConnector = {
            "id": "source-http",
            "source_connector_definition": constant.httpSrcDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        var resHTTP = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector', {
            source_connector: httpSrcConnector
        })

        // Cannot get source connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetSourceConnector', {
            name: `source-connectors/${resHTTP.message.sourceConnector.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/GetSourceConnector name=source-connectors/${resHTTP.message.sourceConnector.id} response StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
        })

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector`, {
            name: `source-connectors/${resHTTP.message.sourceConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector ${resHTTP.message.sourceConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckUpdate() {

    group(`Connector API: Update source connectors [with random "jwt-sub" header]`, () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var gRPCSrcConnector = {
            "id": "source-grpc",
            "source_connector_definition": constant.gRPCSrcDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector', {
            source_connector: gRPCSrcConnector
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector response CreateSourceConnector StatusOK": (r) => r.status === grpc.StatusOK,
        });

        gRPCSrcConnector.connector.description = randomString(20)

        // Cannot update source connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/UpdateSourceConnector', {
            source_connector: gRPCSrcConnector
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/UpdateSourceConnector ${gRPCSrcConnector.id} response StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
        })

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector`, {
            name: `source-connectors/${gRPCSrcConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector ${gRPCSrcConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });

}

export function CheckDelete() {

    group(`Connector API: Delete source connectors [with random "jwt-sub" header]`, () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        // Cannot delete source connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector', {
            name: `source-connectors/source-http`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector source-http response StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
        })

        // Cannot delete destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector', {
            name: `destination-connectors/destination-http`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector destination-http response StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
        })

        client.close();
    });
}

export function CheckLookUp() {

    group(`Connector API: Look up source connectors by UID [with random "jwt-sub" header]`, () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var httpSrcConnector = {
            "id": "source-http",
            "source_connector_definition": constant.httpSrcDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        var resHTTP = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector', {
            source_connector: httpSrcConnector
        })

        // Cannot look up source connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/LookUpSourceConnector', {
            permalink: `source-connectors/${resHTTP.message.sourceConnector.uid}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/LookUpSourceConnector permalink=source-connectors/${resHTTP.message.sourceConnector.uid} response StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
        })

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector`, {
            name: `source-connectors/source-http`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector source-http response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckState() {

    group(`Connector API: Change state source connectors [with random "jwt-sub" header]`, () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var httpSrcConnector = {
            "id": "source-http",
            "source_connector_definition": constant.httpSrcDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        var resHTTP = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector', {
            source_connector: httpSrcConnector
        })

        // Cannot connect source connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectSourceConnector', {
            name: `source-connectors/${resHTTP.message.sourceConnector.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/ConnectSourceConnector ${resHTTP.message.sourceConnector.id} response StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
        })

        // Cannot disconnect source connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/DisconnectSourceConnector', {
            name: `source-connectors/${resHTTP.message.sourceConnector.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/DisconnectSourceConnector ${resHTTP.message.sourceConnector.id} response StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
        })

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector`, {
            name: `source-connectors/source-http`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector source-http response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckRename() {

    group(`Connector API: Rename source connectors [with random "jwt-sub" header]`, () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var httpSrcConnector = {
            "id": "source-http",
            "source_connector_definition": constant.httpSrcDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        var resHTTP = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateSourceConnector', {
            source_connector: httpSrcConnector
        })

        // Cannot rename source connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/RenameSourceConnector', {
            name: `source-connectors/${resHTTP.message.sourceConnector.id}`,
            new_source_connector_id: "some-id-not-http"
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/RenameSourceConnector ${resHTTP.message.sourceConnector.id} response StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
        })

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector`, {
            name: `source-connectors/source-http`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteSourceConnector source-http response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}
