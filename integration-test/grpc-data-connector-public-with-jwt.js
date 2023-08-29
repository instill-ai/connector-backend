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
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
            parent: `${constant.namespace}`,
            connector_resource: csvDstConnector
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource CSV response StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
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
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
            parent: `${constant.namespace}`,
            connector_resource: mySQLDstConnector
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource MySQL response StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
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
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources', {
            parent: `${constant.namespace}`,
            filter: "connector_type=CONNECTOR_TYPE_DATA",
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/ListUserConnectorResources response StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
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

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
            parent: `${constant.namespace}`,
            connector_resource: csvDstConnector
        })

        // client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
        //     name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        // })

        // check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchUserConnectorResource', {
        //     name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        // }), {
        //     "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        // })

        // Cannot get destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/GetUserConnectorResource CSV ${resCSVDst.message.connectorResource.id} response StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${resCSVDst.message.connectorResource.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
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

        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
            parent: `${constant.namespace}`,
            connector_resource: csvDstConnector
        })

        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        })

        var csvDstConnectorUpdate = {
            "id": csvDstConnector.id,
            "name": `${constant.namespace}/connector-resources/${csvDstConnector.id}`,
            "connector_definition_name": csvDstConnector.connector_definition_name,
            "tombstone": true,
            "description": randomString(50),
            "configuration": {
                destination_path: "/tmp"
            }
        }

        // Cannot update destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/UpdateUserConnectorResource', {
            connector_resource: csvDstConnectorUpdate,
            update_mask: "description,configuration",
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/UpdateUserConnectorResource ${csvDstConnectorUpdate.id} response StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
            name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
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

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
            parent: `${constant.namespace}`,
            connector_resource: csvDstConnector
        })

        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        })

        // Cannot look up destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/LookUpConnectorResource', {
            permalink: `connector-resources/${resCSVDst.message.connectorResource.uid}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/LookUpConnectorResource CSV ${resCSVDst.message.connectorResource.id} response StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
            name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
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

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
            parent: `${constant.namespace}`,
            connector_resource: csvDstConnector
        })

        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        })

        // Cannot connect destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource ${resCSVDst.message.connectorResource.id} response StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        // Cannot disconnect destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/DisconnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/DisconnectUserConnectorResource ${resCSVDst.message.connectorResource.id} response at UNSPECIFIED StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        // Cannot connect destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource ${resCSVDst.message.connectorResource.id} response at STATE_CONNECTED state StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        // Cannot disconnect destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/DisconnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/DisconnectUserConnectorResource ${resCSVDst.message.connectorResource.id} response at STATE_CONNECTED state StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        // Cannot connect destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource ${resCSVDst.message.connectorResource.id} response at STATE_DISCONNECTED state StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        // Cannot disconnect destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/DisconnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/DisconnectUserConnectorResource ${resCSVDst.message.connectorResource.id} response at STATE_DISCONNECTED state StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
            name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
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

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
            parent: `${constant.namespace}`,
            connector_resource: csvDstConnector
        })

        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        })

        let new_id = `some-id-not-${resCSVDst.message.connectorResource.id}`

        // Cannot rename destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/RenameUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/resCSVDst.message.connectorResource.id`,
            new_connector_id: new_id
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/RenameUserConnectorResource ${resCSVDst.message.connectorResource.id} response StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
            name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
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

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
            parent: `${constant.namespace}`,
            connector_resource: csvDstConnector
        })

        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        })

        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WatchUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            "vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource CSV destination connector STATE_CONNECTED": (r) => r.message.state === "STATE_CONNECTED",
        })

        // Cannot write destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ExecuteUserConnectorResource', {
            "name": `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`,
            "inputs": constant.clsModelOutputs
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/ExecuteUserConnectorResource ${resCSVDst.message.connectorResource.id} response (classification) StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${resCSVDst.message.connectorResource.id} response (classification) StatusOK`]: (r) => r.status === grpc.StatusOK,
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

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateUserConnectorResource', {
            parent: `${constant.namespace}`,
            connector_resource: csvDstConnector
        })

        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        })

        // Cannot test destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/TestUserConnectorResource', {
            name: `${constant.namespace}/connector-resources/${resCSVDst.message.connectorResource.id}`
        }, constant.paramsGRPCWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/TestUserConnectorResource CSV ${resCSVDst.message.connectorResource.id} response StatusUnauthenticated`]: (r) => r.status === grpc.StatusUnauthenticated,
        })

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource`, {
            name: `${constant.namespace}/connector-resources/${csvDstConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteUserConnectorResource ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}
