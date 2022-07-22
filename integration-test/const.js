export const pipelineHost = "http://pipeline-backend:8081"
export const connectorHost = "http://connector-backend:8082";
export const modelHost = "http://model-backend:8083"

export const csvDstConfig = {
    "destination_path": "/local"
};

export const csvDstDefRscName = "destination-connector-definitions/destination-csv"
export const csvDstDefRscPermalink = "destination-connector-definitions/8be1cf83-fde1-477f-a4ad-318d23c9f3c6"

export const httpSrcDefRscName = "source-connector-definitions/source-http"
export const httpSrcDefRscPermalink = "source-connector-definitions/f20a3c02-c70e-4e76-8566-7c13ca11d18d"

export const gRPCSrcDefRscName = "source-connector-definitions/source-grpc"
export const gRPCSrcDefRscPermalink = "source-connector-definitions/82ca7d29-a35c-4222-b900-8d6878195e7a"

export const httpDstDefRscName = "destination-connector-definitions/destination-http"
export const httpDstDefRscPermalink = "destination-connector-definitions/909c3278-f7d1-461c-9352-87741bef11d3"

export const gRPCDstDefRscName = "destination-connector-definitions/destination-grpc"
export const gRPCDstDefRscPermalink = "destination-connector-definitions/c0e4a82c-9620-4a72-abd1-18586f2acccd"

export const mySQLDstDefRscName = "destination-connector-definitions/destination-mysql"
export const mySQLDstDefRscPermalink = "destination-connector-definitions/ca81ee7c-3163-4246-af40-094cc31e5e42"

export const detModelOutput = {
    "detection_outputs": [
        {
            "bounding_box_objects": [{ "bounding_box": { "height": 402.58002, "left": 325.7926, "top": 99.084984, "width": 204.18988 }, "category": "dog", "score": 0.980409 }, { "bounding_box": { "height": 242.36627, "left": 133.76924, "top": 195.17859, "width": 207.40651 }, "category": "dog", "score": 0.9009272 }]
        },
        {
            "bounding_box_objects": [{ "bounding_box": { "height": 402.58002, "left": 325.7926, "top": 99.084984, "width": 204.18988 }, "category": "dog", "score": 0.980409 }, { "bounding_box": { "height": 242.36627, "left": 133.76924, "top": 195.17859, "width": 207.40651 }, "category": "dog", "score": 0.9009272 }]
        },
        {
            "bounding_box_objects": [{ "bounding_box": { "height": 402.58002, "left": 325.7926, "top": 99.084984, "width": 204.18988 }, "category": "dog", "score": 0.980409 }, { "bounding_box": { "height": 242.36627, "left": 133.76924, "top": 195.17859, "width": 207.40651 }, "category": "dog", "score": 0.9009272 }]
        }
    ]
}
