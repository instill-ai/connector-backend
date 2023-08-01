package service

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/connector-backend/pkg/utils"
)

func (s *service) WriteNewDataPoint(ctx context.Context, data utils.UsageMetricData, pipelineMetadata *structpb.Value) {

	s.redisClient.RPush(ctx, fmt.Sprintf("user:%s:execute.execute_time", data.OwnerUID), data.ExecuteTime.Format(time.RFC3339Nano))
	s.redisClient.RPush(ctx, fmt.Sprintf("user:%s:execute.execute_uid", data.OwnerUID), data.ConnectorExecuteUID)
	s.redisClient.RPush(ctx, fmt.Sprintf("user:%s:execute.connector_uid", data.OwnerUID), data.ConnectorUID)
	s.redisClient.RPush(ctx, fmt.Sprintf("user:%s:execute.connector_definition_uid", data.OwnerUID), data.ConnectorDefinitionUid)
	s.redisClient.RPush(ctx, fmt.Sprintf("user:%s:execute.status", data.OwnerUID), data.Status.String())

	s.influxDBWriteClient.WritePoint(utils.NewDataPoint(data, pipelineMetadata))
}
