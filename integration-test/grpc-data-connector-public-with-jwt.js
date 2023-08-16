import grpc from 'k6/net/grpc';
import {
    check,
    group,
    sleep
} from "k6";
import {
    randomString
} from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js"
import * as helper from "./helper.js"

const client = new grpc.Client();
client.load(['proto/vdp/connector/v1alpha'], 'connector_public_service.proto');

export function CheckCreate() {

    group(`Connector API: Create destination connectors [with random "jwt-sub" header]`, () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        // destination-csv
        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        // Cannot create csv destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: csvDstConnector
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/CreateConnector CSV response StatusNotFound`]: (r) => r.status === grpc.StatusNotFound,
        })

        // destination-mysql (will end up with STATE_ERROR)
        var mySQLDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.mySQLDstDefRscName,
            "configuration": {
                "host": randomString(10),
                "port": 3306,
                "username": randomString(10),
                "database": randomString(10),
            }
        }

        // Cannot create MySQL destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: mySQLDstConnector
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/CreateConnector MySQL response StatusNotFound`]: (r) => r.status === grpc.StatusNotFound,
        })

        client.close();
    });

}

export function CheckList() {

    group(`Connector API: List destination connectors [with random "jwt-sub" header]`, () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        // Cannot list destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListConnectors', {
            filter: "connector_type=CONNECTOR_TYPE_DATA",
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/ListConnectors response StatusNotFound`]: (r) => r.status === grpc.StatusNotFound,
        })

        client.close();
    });
}

export function CheckGet() {

    group(`Connector API: Get destination connectors by ID [with random "jwt-sub" header]`, () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: csvDstConnector
        })

        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnector', {
            name: `connectors/${csvDstConnector.id}`
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        // Cannot get destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/GetConnector CSV ${resCSVDst.message.connector.id} response StatusNotFound`]: (r) => r.status === grpc.StatusNotFound,
        })

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${resCSVDst.message.connector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckUpdate() {

    group(`Connector API: Update destination connectors [with random "jwt-sub" header]`, () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: csvDstConnector
        })

        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnector', {
            name: `connectors/${csvDstConnector.id}`
        })

        var csvDstConnectorUpdate = {
            "id": csvDstConnector.id,
            "name": `connectors/${csvDstConnector.id}`,
            "connector_definition_name": csvDstConnector.connector_definition_name,
            "tombstone": true,
            "description": randomString(50),
            "configuration": {
                destination_path: "/tmp"
            }
        }

        // Cannot update destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/UpdateConnector', {
            connector: csvDstConnectorUpdate,
            update_mask: "description,configuration",
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/UpdateConnector ${csvDstConnectorUpdate.id} response StatusNotFound`]: (r) => r.status === grpc.StatusNotFound,
        })

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${csvDstConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckLookUp() {

    group(`Connector API: Look up destination connectors by UID [with random "jwt-sub" header]`, () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: csvDstConnector
        })

        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnector', {
            name: `connectors/${csvDstConnector.id}`
        })

        // Cannot look up destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/LookUpConnector', {
            permalink: `connector/${resCSVDst.message.connector.uid}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/LookUpConnector CSV ${resCSVDst.message.connector.id} response StatusNotFound`]: (r) => r.status === grpc.StatusNotFound,
        })

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${csvDstConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckState() {

    group(`Connector API: Change state destination connectors [with random "jwt-sub" header]`, () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: csvDstConnector
        })

        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnector', {
            name: `connectors/${csvDstConnector.id}`
        })

        // Cannot connect destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/ConnectConnector ${resCSVDst.message.connector.id} response StatusNotFound`]: (r) => r.status === grpc.StatusNotFound,
        })

        // Cannot disconnect destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/DisconnectConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/DisconnectConnector ${resCSVDst.message.connector.id} response at UNSPECIFIED StatusNotFound`]: (r) => r.status === grpc.StatusNotFound,
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        // Cannot connect destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/ConnectConnector ${resCSVDst.message.connector.id} response at STATE_CONNECTED state StatusNotFound`]: (r) => r.status === grpc.StatusNotFound,
        })

        // Cannot disconnect destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/DisconnectConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/DisconnectConnector ${resCSVDst.message.connector.id} response at STATE_CONNECTED state StatusNotFound`]: (r) => r.status === grpc.StatusNotFound,
        })

        // Cannot connect destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/ConnectConnector ${resCSVDst.message.connector.id} response at STATE_DISCONNECTED state StatusNotFound`]: (r) => r.status === grpc.StatusNotFound,
        })

        // Cannot disconnect destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/DisconnectConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/DisconnectConnector ${resCSVDst.message.connector.id} response at STATE_DISCONNECTED state StatusNotFound`]: (r) => r.status === grpc.StatusNotFound,
        })

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${csvDstConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckRename() {

    group(`Connector API: Rename destination connectors [with random "jwt-sub" header]`, () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: csvDstConnector
        })

        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnector', {
            name: `connectors/${csvDstConnector.id}`
        })

        let new_id = `some-id-not-${resCSVDst.message.connector.id}`

        // Cannot rename destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/RenameConnector', {
            name: resCSVDst.message.connector.id,
            new_connector_id: new_id
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/RenameConnector ${resCSVDst.message.connector.id} response StatusNotFound`]: (r) => r.status === grpc.StatusNotFound,
        })

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${csvDstConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckExecute() {

    group(`Connector API: Write destination connectors [with random "jwt-sub" header]`, () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

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

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: csvDstConnector
        })

        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnector', {
            name: `connectors/${csvDstConnector.id}`
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateConnector CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        // Cannot write destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnector', {
            "name": `connector/${resCSVDst.message.connector.id}`,
            "inputs": constant.clsModelOutputs
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/ExecuteConnector ${resCSVDst.message.connector.id} response (classification) StatusNotFound`]: (r) => r.status === grpc.StatusNotFound,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${resCSVDst.message.connector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${resCSVDst.message.connector.id} response (classification) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckTest() {

    group(`Connector API: Test destination connectors' connection [with random "jwt-sub" header]`, () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var csvDstConnector = {
            "id": randomString(10),
            "connector_definition_name": constant.csvDstDefRscName,
            "description": randomString(50),
            "configuration": constant.csvDstConfig
        }

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateConnector', {
            connector: csvDstConnector
        })

        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectConnector', {
            name: `connectors/${csvDstConnector.id}`
        })

        // Cannot test destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/TestConnector', {
            name: `connectors/${resCSVDst.message.connector.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/TestConnector CSV ${resCSVDst.message.connector.id} response StatusNotFound`]: (r) => r.status === grpc.StatusNotFound,
        })

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector`, {
            name: `connectors/${csvDstConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}
