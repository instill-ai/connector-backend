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

        // destination-http
        var httpDstConnector = {
            "id": "destination-http",
            "destination_connector_definition": constant.httpDstDefRscName,
            "connector": {
                "description": "HTTP source",
                "configuration": {},
            }
        }

        // Cannot create http destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: httpDstConnector
        }, constant.paramsWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector HTTP response StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
        })

        // destination-grpc
        var gRPCDstConnector = {
            "id": "destination-grpc",
            "destination_connector_definition": constant.gRPCDstDefRscName,
            "connector": {
                "configuration": {}
            }
        }

        // Cannot create grpc destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: gRPCDstConnector
        }, constant.paramsWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector gRPC response StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
        })

        // destination-csv
        var csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": constant.csvDstConfig
            }
        }

        // Cannot create csv destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: csvDstConnector
        }, constant.paramsWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector CSV response StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
        })

        // destination-mysql (will end up with STATE_ERROR)
        var mySQLDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.mySQLDstDefRscName,
            "connector": {
                "configuration": {
                    "host": randomString(10),
                    "port": 3306,
                    "username": randomString(10),
                    "database": randomString(10),
                }
            }
        }

        // Cannot create MySQL destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: mySQLDstConnector
        }, constant.paramsWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector MySQL response StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
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
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors', {}, constant.paramsWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/ListDestinationConnectors response StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
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
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": constant.csvDstConfig
            }
        }

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: csvDstConnector
        })

        // Check connector state being updated in 120 secs
        var currentTime = new Date().getTime();
        var timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            var res = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnector', {
                name: `destination-connectors/${resCSVDst.message.destinationConnector.id}`
            })
            if (res.message.destinationConnector.connector.state === "STATE_CONNECTED") {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        // Cannot get destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnector', {
            name: `destination-connectors/${resCSVDst.message.destinationConnector.id}`
        }, constant.paramsWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnector CSV ${resCSVDst.message.destinationConnector.id} response StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
        })

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/${resCSVDst.message.destinationConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${resCSVDst.message.destinationConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
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
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": constant.csvDstConfig
            }
        }

        client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: csvDstConnector
        })

        var csvDstConnectorUpdate = {
            "id": csvDstConnector.id,
            "name": `destination-connectors/${csvDstConnector.id}`,
            "destination_connector_definition": csvDstConnector.destination_connector_definition,
            "connector": {
                "tombstone": true,
                "description": randomString(50),
                "configuration": {
                    destination_path: "/tmp"
                }
            }
        }

        // Cannot update destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/UpdateDestinationConnector', {
            destination_connector: csvDstConnectorUpdate,
            update_mask: "connector.description,connector.configuration",
        }, constant.paramsWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/UpdateDestinationConnector ${csvDstConnectorUpdate.id} response StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
        })

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/${csvDstConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
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
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": constant.csvDstConfig
            }
        }

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: csvDstConnector
        })

        // Cannot look up destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/LookUpDestinationConnector', {
            permalink: `destination_connector/${resCSVDst.message.destinationConnector.uid}`
        }, constant.paramsWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/LookUpDestinationConnector CSV ${resCSVDst.message.destinationConnector.id} response StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
        })

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/${csvDstConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
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
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": constant.csvDstConfig
            }
        }

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: csvDstConnector
        })

        // Cannot connect destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectDestinationConnector', {
            name: `destination-connectors/${resCSVDst.message.destinationConnector.id}`
        }, constant.paramsWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/ConnectDestinationConnector ${resCSVDst.message.destinationConnector.id} response StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
        })

        // Cannot disconnect destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/DisconnectDestinationConnector', {
            name: `destination-connectors/${resCSVDst.message.destinationConnector.id}`
        }, constant.paramsWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/DisconnectDestinationConnector ${resCSVDst.message.destinationConnector.id} response at UNSPECIFIED StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
        })

        // Check connector state being updated in 120 secs
        let currentTime = new Date().getTime();
        let timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            var res = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnector', {
                name: `destination-connectors/${resCSVDst.message.destinationConnector.id}`
            })
            if (res.message.destinationConnector.connector.state === "STATE_CONNECTED") {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        // Cannot connect destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectDestinationConnector', {
            name: `destination-connectors/${resCSVDst.message.destinationConnector.id}`
        }, constant.paramsWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/ConnectDestinationConnector ${resCSVDst.message.destinationConnector.id} response at STATE_CONNECTED state StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
        })

        // Cannot disconnect destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/DisconnectDestinationConnector', {
            name: `destination-connectors/${resCSVDst.message.destinationConnector.id}`
        }, constant.paramsWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/DisconnectDestinationConnector ${resCSVDst.message.destinationConnector.id} response at STATE_CONNECTED state StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
        })

        // Cannot connect destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/ConnectDestinationConnector', {
            name: `destination-connectors/${resCSVDst.message.destinationConnector.id}`
        }, constant.paramsWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/ConnectDestinationConnector ${resCSVDst.message.destinationConnector.id} response at STATE_DISCONNECTED state StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
        })

        // Cannot disconnect destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/DisconnectDestinationConnector', {
            name: `destination-connectors/${resCSVDst.message.destinationConnector.id}`
        }, constant.paramsWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/DisconnectDestinationConnector ${resCSVDst.message.destinationConnector.id} response at STATE_DISCONNECTED state StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
        })

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/${csvDstConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
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
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": constant.csvDstConfig
            }
        }

        var resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: csvDstConnector
        })

        let new_id = `some-id-not-${resCSVDst.message.destinationConnector.id}`

        // Cannot rename destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/RenameDestinationConnector', {
            name: resCSVDst.message.destinationConnector.id,
            new_destination_connector_id: new_id
        }, constant.paramsWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/RenameDestinationConnector ${resCSVDst.message.destinationConnector.id} response StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
        })

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/${csvDstConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${csvDstConnector.id} response StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}

export function CheckWrite() {

    group(`Connector API: Write destination connectors [with random "jwt-sub" header]`, () => {

        client.connect(constant.connectorGRPCPublicHost, {
            plaintext: true
        });

        var csvDstConnector, resCSVDst, currentTime, timeoutTime

        // Write classification output
        csvDstConnector = {
            "id": randomString(10),
            "destination_connector_definition": constant.csvDstDefRscName,
            "connector": {
                "description": randomString(50),
                "configuration": {
                    "destination_path": "/local/test-classification"
                },
            }
        }

        resCSVDst = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/CreateDestinationConnector', {
            destination_connector: csvDstConnector
        })

        // Check connector state being updated in 120 secs
        currentTime = new Date().getTime();
        timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            var res = client.invoke('vdp.connector.v1alpha.ConnectorPublicService/GetDestinationConnector', {
                name: `destination-connectors/${resCSVDst.message.destinationConnector.id}`
            })
            if (res.message.destinationConnector.connector.state === "STATE_CONNECTED") {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        // Cannot write destination connector of a non-exist user
        check(client.invoke('vdp.connector.v1alpha.ConnectorPublicService/WriteDestinationConnector', {
            "name": `destination_connector/${resCSVDst.message.destinationConnector.id}`,
            "sync_mode": "SUPPORTED_SYNC_MODES_FULL_REFRESH",
            "destination_sync_mode": "SUPPORTED_DESTINATION_SYNC_MODES_OVERWRITE",
            "pipeline": "pipelines/dummy-pipeline",
            "recipe": {
                "source": "source-connectors/dummy-source",
                "model_instances": [
                    "models/dummy-model/instances/v1.0-cpu"
                ],
                "destination": "destination-connectors/dummy-destination",
            },
            "data_mapping_indices": ["01GB5T5ZK9W9C2VXMWWRYM8WPA"],
            "model_instance_outputs": constant.clsModelInstOutputs
        }, constant.paramsWithJwt), {
            [`[with random "jwt-sub" header] vdp.connector.v1alpha.ConnectorPublicService/WriteDestinationConnector ${resCSVDst.message.destinationConnector.id} response (classification) StatusUnknown`]: (r) => r.status === grpc.StatusUnknown,
        });

        // Wait for 1 sec for the connector writing to the destination-csv before deleting it
        sleep(1)

        check(client.invoke(`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector`, {
            name: `destination-connectors/${resCSVDst.message.destinationConnector.id}`
        }), {
            [`vdp.connector.v1alpha.ConnectorPublicService/DeleteDestinationConnector ${resCSVDst.message.destinationConnector.id} response (classification) StatusOK`]: (r) => r.status === grpc.StatusOK,
        });

        client.close();
    });
}
