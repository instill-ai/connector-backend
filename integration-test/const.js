import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

let proto
let pHost, cHost, mHost
let cPrivatePort, cPublicPort, pPublicPort, mPublicPort

if (__ENV.API_GATEWAY_VDP_HOST && !__ENV.API_GATEWAY_VDP_PORT || !__ENV.API_GATEWAY_VDP_HOST && __ENV.API_GATEWAY_VDP_PORT) {
  fail("both API_GATEWAY_VDP_HOST and API_GATEWAY_VDP_PORT should be properly configured.")
}

export const apiGatewayMode = (__ENV.API_GATEWAY_VDP_HOST && __ENV.API_GATEWAY_VDP_PORT);

if (__ENV.API_GATEWAY_PROTOCOL) {
  if (__ENV.API_GATEWAY_PROTOCOL !== "http" && __ENV.API_GATEWAY_PROTOCOL != "https") {
    fail("only allow `http` or `https` for API_GATEWAY_PROTOCOL")
  }
  proto = __ENV.API_GATEWAY_PROTOCOL
} else {
  proto = "http"
}

if (apiGatewayMode) {
  // api gateway mode
  pHost = cHost = __ENV.API_GATEWAY_VDP_HOST
  cPrivatePort = 3082
  pPublicPort = cPublicPort = __ENV.API_GATEWAY_VDP_PORT

  // TODO: remove model-backend dependency
  mHost = "api-gateway-model"
  mPublicPort = 9080
} else {
  // direct microservice mode
  pHost = "pipeline-backend"
  cHost = "connector-backend"
  mHost = "model-backend"
  cPrivatePort = 3082
  cPublicPort = 8082
  pPublicPort = 8081
  mPublicPort = 8083
}

export const connectorPrivateHost = `${proto}://${cHost}:${cPrivatePort}`;
export const connectorPublicHost = `${proto}://${cHost}:${cPublicPort}`;
export const connectorGRPCPrivateHost = `${cHost}:${cPrivatePort}`;
export const connectorGRPCPublicHost = `${cHost}:${cPublicPort}`;
export const pipelinePublicHost = `${proto}://${pHost}:${pPublicPort}`;
export const pipelineGRPCPublicHost = `${pHost}:${pPublicPort}`;
export const modelPublicHost = `${proto}://${mHost}:${mPublicPort}`;

export const csvDstDefRscName = "connector-definitions/airbyte-destination-csv"
export const csvDstDefRscPermalink = "connector-definitions/8be1cf83-fde1-477f-a4ad-318d23c9f3c6"

export const srcDefRscName = "connector-definitions/trigger"
export const srcDefRscPermalink = "connector-definitions/f20a3c02-c70e-4e76-8566-7c13ca11d18d"

export const dstDefRscName = "connector-definitions/response"
export const dstDefRscPermalink = "connector-definitions/909c3278-f7d1-461c-9352-87741bef11d3"

export const mySQLDstDefRscName = "connector-definitions/airbyte-destination-mysql"
export const mySQLDstDefRscPermalink = "connector-definitions/ca81ee7c-3163-4246-af40-094cc31e5e42"

export const csvDstConfig = {
  "destination_path": "/local/test"
};

export const params = {
  headers: {
    "Content-Type": "application/json",
  },
  timeout: "600s",
};

const randomUUID = uuidv4();
export const paramsGRPCWithJwt = {
  metadata: {
    "Content-Type": "application/json",
    "Jwt-Sub": randomUUID,
  },
}

export const paramsHTTPWithJwt = {
  headers: {
    "Content-Type": "application/json",
    "Jwt-Sub": randomUUID,
  },
}

const singleModelPipelineMetadata = {
  "pipeline": {
    "name": "pipelines/dummy-pipeline",
    "recipe": {
      "version": "v1alpha",
      "components": [
        { "id": "s01", "resource_name": "source-connectors/dummy-source" },
        { "id": "m01", "resource_name": "models/dummy-model" },
        { "id": "d01", "resource_name": "destination-connectors/dummy-destination" }
      ]
    }
  }
}

const multipleModelOutputsMetadata = {
  "pipeline": {
    "name": "pipelines/dummy-pipeline",
    "recipe": {
      "version": "v1alpha",
      "components": [
        { "id": "s01", "resource_name": "source-connectors/dummy-source" },
        { "id": "m01", "resource_name": "models/dummy-model-1" },
        { "id": "m02", "resource_name": "models/dummy-model-2" },
        { "id": "d01", "resource_name": "destination-connectors/dummy-destination" }
      ]
    }
  }
}

export const clsModelOutputs = [{
  "data_mapping_index": "01GB5T5ZK9W9C2VXMWWRYM8WPA",
  "structured_data": {
    "classification": {
      "category": "person",
      "score": 0.99
    }
  },
  "metadata": singleModelPipelineMetadata
}]



export const detectionModelOutputs = [
  {
    "data_mapping_index": "01GB5T5ZK9W9C2VXMWWRYM8WPM",
    "structured_data": {
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
      },
      "classification": {
        "category": "person",
        "score": 0.99
      }
    },
    "metadata": multipleModelOutputsMetadata
  },
  {
    "data_mapping_index": "01GB5T5ZK9W9C2VXMWWRYM8WPN",
    "structured_data": {
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
      },
      "classification": {
        "category": "person",
        "score": 0.99
      }
    },
    "metadata": multipleModelOutputsMetadata
  },
  {
    "data_mapping_index": "01GB5T5ZK9W9C2VXMWWRYM8WPO",
    "structured_data": {
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
      },
      "classification": {
        "category": "person",
        "score": 0.99
      }
    },
    "metadata": multipleModelOutputsMetadata
  }
]

export const detectionEmptyModelOutputs = [{
  "data_mapping_index": "01GB5T5ZK9W9C2VXMWWRYM8WPM",
  "structured_data": {
    "detection": {
      "objects": []
    }
  },
  "metadata": singleModelPipelineMetadata
}]


export const keypointModelOutputs = [{
  "data_mapping_index": "01GB5T5ZK9W9C2VXMWWRYM8WPT",
  "structured_data": {
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
  },
  "metadata": singleModelPipelineMetadata
}]

export const ocrModelOutputs = [{
  "data_mapping_index": "01GB5T5ZK9W9C2VXMWWRYM8WPU",
  "structured_data": {
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
  },
  "metadata": singleModelPipelineMetadata
}]

export const semanticSegModelOutputs = [{
  "data_mapping_index": "01GB5T5ZK9W9C2VXMWWRYM8WPU",
  "structured_data": {
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
  },
  "metadata": singleModelPipelineMetadata
}]

export const instSegModelOutputs = [{
  "data_mapping_index": "01GB5T5ZK9W9C2VXMWWRYM8WPU",
  "structured_data": {
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
  },
  "metadata": singleModelPipelineMetadata
}]

export const textToImageModelOutputs = [{
  "data_mapping_index": "01GB5T5ZK9W9C2VXMWWRYM8WPU",
  "structured_data": {
    "text_to_image": {
      "images": [
        "/9j/4AAQSkZJRgABAQAAAQABAAD/...oADAMBAAIRAxEAPwD2p76rBDHU2KHMpuE",
        "/9j/4AAQSkZJRgABAQAAAQABAAD/...oADAMBAAIRAxEAPwD2p76rBDHU2KHMpuE",
        "/9j/4AAQSkZJRgABAQAAAQABAAD/...oADAMBAAIRAxEAPwD2p76rBDHU2KHMpuE",
        "/9j/4AAQSkZJRgABAQAAAQABAAD/...oADAMBAAIRAxEAPwD2p76rBDHU2KHMpuE"
      ]
    }
  },
  "metadata": singleModelPipelineMetadata
}]

export const textGenerationModelOutputs = [{
  "data_mapping_index": "01GB5T5ZK9W9C2VXMWWRYM8WPU",
  "structured_data": {
    "text_generation": {
      "text": "The winds of change are blowing strong, bring new beginnings, righting wrongs. The world around us is constantly turning, and with each sunrise, our spirits are yearning..."
    }
  },
  "metadata": singleModelPipelineMetadata
}]

export const unspecifiedModelOutputs = [{
  "data_mapping_index": "01GB5T5ZK9W9C2VXMWWRYM8WPU",
  "structured_data": {
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
  },
  "metadata": singleModelPipelineMetadata
}]
