let pHost = __ENV.HOST ? `${__ENV.HOST}` : "pipeline-backend"
let cHost = __ENV.HOST ? `${__ENV.HOST}` : "connector-backend"
let mHost = __ENV.HOST ? `${__ENV.HOST}` : "model-backend"
let pPort = 8081
let cPort = 8082
let mPort = 8083
if (__ENV.HOST == "api-gateway") {
  pHost = cHost = mHost = "localhost"
}
if (__ENV.HOST == "api-gateway") {
  pPort = cPort = mPort = 8000
}

export const pipelineHost = `http://${pHost}:${pPort}`;
export const connectorHost = `http://${cHost}:${cPort}`;
export const modelHost = `http://${mHost}:${mPort}`;

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

export const csvDstConfig = {
  "destination_path": "/local/test"
};

export const clsModelInstOutputs = [
  {
    "task": "TASK_CLASSIFICATION",
    "model_instance": "models/dummy-model/instances/v1.0",
    "task_outputs": [
      {
        "index": "01GB5T5ZK9W9C2VXMWWRYM8WPS",
        "classification": {
          "category": "person",
          "score": 0.99
        }
      }
    ]
  }
]

export const detectionModelInstOutputs = [
  {
    "task": "TASK_DETECTION",
    "model_instance": "models/dummy-model/instances/v1.0",
    "task_outputs": [
      {
        "index": "01GB5T5ZK9W9C2VXMWWRYM8WPM",
        "detection": {
          "objects": [
            {
              "bounding_box": { "height": 0, "left": 0, "top": 99.084984, "width": 204.18988 },
              "category": "dog",
              "score": 0.980409
            },
            {
              "bounding_box": { "height": 242.36627, "left": 133.76924, "top": 195.17859, "width": 207.40651 },
              "category": "dog",
              "score": 0.9009272
            }
          ]
        }
      },
      {
        "index": "01GB5T5ZK9W9C2VXMWWRYM8WPN",
        "detection": {
          "objects": [
            {
              "bounding_box": { "height": 402.58002, "left": 0, "top": 99.084984, "width": 204.18988 },
              "category": "dog",
              "score": 0.980409
            },
            {
              "bounding_box": { "height": 242.36627, "left": 133.76924, "top": 195.17859, "width": 207.40651 },
              "category": "dog",
              "score": 0.9009272
            }
          ]
        }
      },
      {
        "index": "01GB5T5ZK9W9C2VXMWWRYM8WPO",
        "detection": {
          "objects": [
            {
              "bounding_box": { "height": 0, "left": 325.7926, "top": 99.084984, "width": 204.18988 },
              "category": "dog",
              "score": 0.980409
            },
            {
              "bounding_box": { "height": 242.36627, "left": 133.76924, "top": 195.17859, "width": 207.40651 },
              "category": "dog",
              "score": 0.9009272
            }
          ]
        }
      }
    ]
  },
  {
    "task": "TASK_DETECTION",
    "model_instance": "models/dummy-model/instances/v2.0",
    "task_outputs": [
      {
        "index": "01GB5T5ZK9W9C2VXMWWRYM8WPM",
        "detection": {
          "objects": [
            {
              "bounding_box": { "height": 0, "left": 0, "top": 99.084984, "width": 204.18988 },
              "category": "dog",
              "score": 0.980409
            },
            {
              "bounding_box": { "height": 242.36627, "left": 133.76924, "top": 195.17859, "width": 207.40651 },
              "category": "dog",
              "score": 0.9009272
            }
          ]
        }
      },
      {
        "index": "01GB5T5ZK9W9C2VXMWWRYM8WPN",
        "detection": {
          "objects": [
            {
              "bounding_box": { "height": 402.58002, "left": 0, "top": 99.084984, "width": 204.18988 },
              "category": "dog",
              "score": 0.980409
            },
            {
              "bounding_box": { "height": 242.36627, "left": 133.76924, "top": 195.17859, "width": 207.40651 },
              "category": "dog",
              "score": 0.9009272
            }
          ]
        }
      },
      {
        "index": "01GB5T5ZK9W9C2VXMWWRYM8WPO",
        "detection": {
          "objects": [
            {
              "bounding_box": { "height": 0, "left": 325.7926, "top": 99.084984, "width": 204.18988 },
              "category": "dog",
              "score": 0.980409
            },
            {
              "bounding_box": { "height": 242.36627, "left": 133.76924, "top": 195.17859, "width": 207.40651 },
              "category": "dog",
              "score": 0.9009272
            }
          ]
        }
      }
    ]
  }
]

export const detectionEmptyModelInstOutputs = [
  {
    "task": "TASK_DETECTION",
    "model_instance": "models/dummy-model/instances/v1.0",
    "task_outputs": [
      {
        "index": "01GB5T5ZK9W9C2VXMWWRYM8WPM",
        "detection": {
          "objects": []
        }
      },
    ]
  }
]

export const keypointModelInstOutputs = [
  {
    "task": "TASK_KEYPOINT",
    "model_instance": "models/dummy-model/instances/v1.0",
    "task_outputs": [
      {
        "index": "01GB5T5ZK9W9C2VXMWWRYM8WPT",
        "keypoint": {
          "objects": [
            {
              "keypoints": [{ "x": 10, "y": 100, "v": 0.6 }, { "x": 11, "y": 101, "v": 0.2 }],
              "score": 0.99
            },
            {
              "keypoints": [{ "x": 20, "y": 10, "v": 0.6 }, { "x": 12, "y": 120, "v": 0.7 }],
              "score": 0.99
            },
          ]
        }
      }
    ]
  }
]

export const ocrModelInstOutputs = [
  {
    "task": "TASK_OCR",
    "model_instance": "models/dummy-model/instances/v1.0",
    "task_outputs": [
      {
        "index": "01GB5T5ZK9W9C2VXMWWRYM8WPU",
        "ocr": {
          "objects": [
            {
              "bounding_box": { "height": 402.58002, "left": 0, "top": 99.084984, "width": 204.18988 },
              "text": "some text",
              "score": 0.99
            },
            {
              "bounding_box": { "height": 242.36627, "left": 133.76924, "top": 195.17859, "width": 207.40651 },
              "text": "some text",
              "score": 0.99
            },
          ],
        }
      }
    ]
  }
]

export const instSegModelInstOutputs = [
  {
    "task": "TASK_INSTANCE_SEGMENTATION",
    "model_instance": "models/dummy-model/instances/v1.0",
    "task_outputs": [
      {
        "index": "01GB5T5ZK9W9C2VXMWWRYM8WPU",
        "instance_segmentation": {
          "objects": [
            {
              "rle": "11,6,35,8,59,10,83,12,107,14,131,16,156,16,180,18,205,18,229,19,254,19,278,21,303,21,328,21,353,21,377,23,402,23,427,23,452,23,477,23,501,24,526,24,551,549,1101,24,1126,24,1151,24,1176,24,1201,24,1226,23,1251,23,1276,23,1301,23,1326,23,1351,22,1377,21,1402,21,1427,20,1452,20,1478,19,1503,18,1528,18,1554,16,1580,15,1605,14,1631,13,1657,11,1684,9,1710,6,1735,6",
              "score": 0.9996394,
              "bounding_box": {
                "top": 375,
                "left": 166,
                "width": 25,
                "height": 70
              },
              "category": "stomata"
            },
            {
              "rle": "29,4,55,7,82,9,109,10,136,12,164,12,192,12,220,12,248,12,276,12,304,12,332,12,360,11,388,11,416,11,443,12,472,10,500,10,528,10,556,9,585,8,613,7,641,7,669,6,697,5,726,3",
              "score": 0.9990727,
              "bounding_box": {
                "top": 107,
                "left": 240,
                "width": 27,
                "height": 27
              },
              "category": "stomata"
            }
          ]
        }
      }
    ]
  }
]

export const unspecifiedModelInstOutputs = [
  {
    "task": "TASK_UNSPECIFIED",
    "model_instance": "models/dummy-model/instances/v1.0",
    "task_outputs": [
      {
        "index": "01GB5T5ZK9W9C2VXMWWRYM8WPV",
        "unspecified": {
          "raw_outputs": [
            {
              "name": "some unspecified model output",
              "data_type": "INT8",
              "shape": [3, 3, 3],
              "data": [1, 2, 3, 4, 5, 6, 7]
            },
          ],
        }
      }
    ]
  }
]
