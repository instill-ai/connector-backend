export const csvDstConfig = {
  "supports_incremental": true,
  "connection_specification": {
    "destination_path": "/local"
  },
  "supported_destination_sync_modes": [2, 1]
};

export const csvDstDefinitionRscName = "destination-connector-definitions/destination-csv"
export const csvDstDefinitionRscPermalink = "destination-connector-definitions/8be1cf83-fde1-477f-a4ad-318d23c9f3c6"

export const httpSrcDefRscName = "source-connector-definitions/http"
export const httpSrcDefRscPermalink = "source-connector-definitions/f20a3c02-c70e-4e76-8566-7c13ca11d18d"

export const gRPCSrcDefRscName = "source-connector-definitions/grpc"
export const gRPCSrcDefRscPermalink = "source-connector-definitions/82ca7d29-a35c-4222-b900-8d6878195e7a"

export const httpDstDefRscName = "destination-connector-definitions/http"
export const httpDstDefRscPermalink = "destination-connector-definitions/909c3278-f7d1-461c-9352-87741bef11d3"

export const gRPCDstDefRscName = "destination-connector-definitions/grpc"
export const gRPCDstDefRscPermalink = "destination-connector-definitions/c0e4a82c-9620-4a72-abd1-18586f2acccd"
