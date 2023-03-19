let proto
let pHost, cHost, mHost
let pPort, cPort, mPort

if (__ENV.MODE == "api-gateway") {
  // api-gateway mode
  proto = "http"
  pHost = cHost = mHost = "api-gateway"
  pPort = cPort = mPort = 8080
} else if (__ENV.MODE == "localhost") {
  // localhost mode for GitHub Actions
  proto = "http"
  pHost = cHost = mHost = "localhost"
  pPort = cPort = mPort = 8080
} else {
  // direct microservice mode
  proto = "http"
  pHost = "pipeline-backend"
  cHost = "connector-backend"
  mHost = "model-backend"
  pPort = 8081
  cPort = 8082
  mPort = 8083
}

export const pipelineHost = `${proto}://${pHost}:${pPort}`;
export const connectorHost = `${proto}://${cHost}:${cPort}`;
export const connectorGRPCHost = `${cHost}:${cPort}`;
export const modelHost = `${proto}://${mHost}:${mPort}`;

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
    "model_instance": "models/dummy-model/instances/v1.0-cpu",
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
    "model_instance": "models/dummy-model/instances/v1.0-cpu",
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
    "model_instance": "models/dummy-model/instances/v2.0-cpu",
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
    "model_instance": "models/dummy-model/instances/v1.0-cpu",
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
    "model_instance": "models/dummy-model/instances/v1.0-cpu",
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
    "model_instance": "models/dummy-model/instances/v1.0-cpu",
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

export const semanticSegModelInstOutputs = [
  {
    "task": "TASK_SEMANTIC_SEGMENTATION",
    "model_instance": "models/dummy-model/instances/v1.0-cpu",
    "task_outputs": [
      {
        "index": "01GB5T5ZK9W9C2VXMWWRYM8WPU",
        "semantic_segmentation": {
          "stuffs": [
            {
              "rle": "2918,12,382,33,...",
              "category": "person"
            },
            {
              "rle": "34,18,230,18,...",
              "category": "sky"
            },
            {
              "rle": "34,18,230,18,...",
              "category": "dog"
            }
          ]
        }
      }
    ]
  }
]

export const instSegModelInstOutputs = [
  {
    "task": "TASK_INSTANCE_SEGMENTATION",
    "model_instance": "models/dummy-model/instances/v1.0-cpu",
    "task_outputs": [
      {
        "index": "01GB5T5ZK9W9C2VXMWWRYM8WPU",
        "instance_segmentation": {
          "objects": [
            {
              "rle": "11,6,35,8,59,10,83,12,107,14,131,16,156,16,180,18,205,18,229,...",
              "score": 0.9996394,
              "bounding_box": {
                "top": 375,
                "left": 166,
                "width": 25,
                "height": 70
              },
              "category": "dog"
            },
            {
              "rle": "11,6,35,8,59,10,83,12,107,14,131,16,156,16,180,18,205,18,229,...",
              "score": 0.9990727,
              "bounding_box": {
                "top": 107,
                "left": 240,
                "width": 27,
                "height": 27
              },
              "category": "car"
            }
          ]
        }
      }
    ]
  }
]

export const textToImageModelInstOutputs = [
  {
    "task": "TASK_TEXT_TO_IMAGE",
    "model_instance": "models/dummy-model/instances/v1.0-cpu",
    "task_outputs": [
      {
        "index": "01GB5T5ZK9W9C2VXMWWRYM8WPU",
        "text_to_image": {
          "images": [
            "/9j/4AAQSkZJRgABAQAAAQABAAD/...oADAMBAAIRAxEAPwD2p76rBDHU2KHMpuE",
            "/9j/4AAQSkZJRgABAQAAAQABAAD/...oADAMBAAIRAxEAPwD2p76rBDHU2KHMpuE",
            "/9j/4AAQSkZJRgABAQAAAQABAAD/...oADAMBAAIRAxEAPwD2p76rBDHU2KHMpuE",
            "/9j/4AAQSkZJRgABAQAAAQABAAD/...oADAMBAAIRAxEAPwD2p76rBDHU2KHMpuE"
          ]
        }
      }
    ]
  }
]

export const textGenerationModelInstOutputs = [
  {
    "task": "TASK_TEXT_GENERATION",
    "model_instance": "models/dummy-model/instances/v1.0-cpu",
    "task_outputs": [
      {
        "index": "01GB5T5ZK9W9C2VXMWWRYM8WPU",
        "text_generation": {
          "text": "The winds of change are blowing strong, bring new beginnings, righting wrongs. The world around us is constantly turning, and with each sunrise, our spirits are yearning..."
        }
      }
    ]
  }
]

export const unspecifiedModelInstOutputs = [
  {
    "task": "TASK_UNSPECIFIED",
    "model_instance": "models/dummy-model/instances/v1.0-cpu",
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
