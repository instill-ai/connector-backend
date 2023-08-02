package service

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/connector-backend/config"
	"github.com/instill-ai/connector-backend/pkg/utils"
)

func (s *service) WriteNewDataPoint(ctx context.Context, data utils.UsageMetricData, pipelineMetadata *structpb.Value) error {

	if config.Config.Server.Usage.Enabled {

		bData, err := json.Marshal(data)
		if err != nil {
			return err
		}

		s.redisClient.RPush(ctx, fmt.Sprintf("user:%s:connector.execute_data", data.OwnerUID), string(bData))
	}

	s.influxDBWriteClient.WritePoint(utils.NewDataPoint(data, pipelineMetadata))

	return nil
}
