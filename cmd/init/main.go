package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	_ "embed"

	"github.com/gofrs/uuid"
	"go.einride.tech/aip/filtering"
	"go.opentelemetry.io/otel"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/instill-ai/connector-backend/config"
	"github.com/instill-ai/connector-backend/pkg/connector"
	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/connector-backend/pkg/logger"
	"github.com/instill-ai/connector-backend/pkg/repository"

	database "github.com/instill-ai/connector-backend/pkg/db"
	connectorDataAirbyte "github.com/instill-ai/connector-data/pkg/airbyte"
)

type PrebuiltConnector struct {
	Id                     string      `json:"id"`
	Uid                    string      `json:"uid"`
	Owner                  string      `json:"owner"`
	ConnectorDefinitionUid string      `json:"connector_definition_uid"`
	Configuration          interface{} `json:"configuration"`
	Task                   string      `json:"task"`
}

// BaseDynamic contains common columns for all tables with dynamic UUID as primary key generated when creating
type BaseDynamic struct {
	UID        uuid.UUID      `gorm:"type:uuid;primary_key;<-:create"` // allow read and create
	CreateTime time.Time      `gorm:"autoCreateTime:nano"`
	UpdateTime time.Time      `gorm:"autoUpdateTime:nano"`
	DeleteTime gorm.DeletedAt `sql:"index"`
}

// Connector is the data model of the connector table
type Connector struct {
	BaseDynamic
	ID                     string
	Owner                  string
	ConnectorDefinitionUID uuid.UUID
	Description            string
	Tombstone              bool
	Configuration          datatypes.JSON `gorm:"type:jsonb"`
	ConnectorType          string         `sql:"type:string"`
	State                  string         `sql:"type:string"`
	Visibility             string         `sql:"type:string"`
	Task                   string         `sql:"type:string"`
}

//go:embed prebuilt_list.json
var prebuiltJson []byte

func main() {

	if err := config.Init(); err != nil {
		log.Fatal(err.Error())
	}
	ctx, cancel := context.WithCancel(context.Background())
	ctx, span := otel.Tracer("init-tracer").Start(ctx,
		"main",
	)
	defer span.End()
	defer cancel()

	logger, _ := logger.GetZapLogger(ctx)

	db := database.GetConnection()
	defer database.Close(db)

	repository := repository.NewRepository(db)

	airbyte := connectorDataAirbyte.Init(logger, connectorDataAirbyte.ConnectorOptions{
		MountSourceVDP:        config.Config.Connector.Airbyte.MountSource.VDP,
		MountTargetVDP:        config.Config.Connector.Airbyte.MountTarget.VDP,
		MountSourceAirbyte:    config.Config.Connector.Airbyte.MountSource.Airbyte,
		MountTargetAirbyte:    config.Config.Connector.Airbyte.MountTarget.Airbyte,
		ExcludeLocalConnector: config.Config.Connector.Airbyte.ExcludeLocalConnector,
		VDPProtocolPath:       "/etc/vdp/vdp_protocol.yaml",
	})

	// TODO: use pagination
	conns, _, _, err := repository.ListConnectorResourcesAdmin(ctx, 1000, "", false, filtering.Filter{})
	if err != nil {
		panic(err)
	}

	airbyteConnector := airbyte.(*connectorDataAirbyte.Connector)
	var uids []uuid.UUID
	for idx := range conns {
		uid := conns[idx].ConnectorDefinitionUID
		if airbyteConnector.HasUid(uid) {
			uids = append(uids, uid)

		}
	}

	err = airbyteConnector.PreDownloadImage(logger, uids)

	if err != nil {
		panic(err)
	}

	if config.Config.Server.PrebuiltConnector.Enabled {
		fmt.Println("Load PreBuiltConnector")
		var prebuiltConnectors []PrebuiltConnector
		err := json.Unmarshal(prebuiltJson, &prebuiltConnectors)
		if err != nil {
			panic(err)
		}
		for idx := range prebuiltConnectors {
			// TODO: refactor this
			if val, ok := prebuiltConnectors[idx].Configuration.(map[string]interface{})["api_key"]; ok {
				val := val.(string)
				if val[:4] == "<CFG" {
					envVal := os.Getenv(val[1 : len(val)-1])
					if envVal == "" {
						panic(fmt.Sprintf("%s is missing", val))
					}
					prebuiltConnectors[idx].Configuration.(map[string]interface{})["api_key"] = envVal
				}

			}
			if val, ok := prebuiltConnectors[idx].Configuration.(map[string]interface{})["server_url"]; ok {
				val := val.(string)
				if val[:4] == "<CFG" {
					envVal := os.Getenv(val[1 : len(val)-1])
					if envVal == "" {
						panic(fmt.Sprintf("%s is missing", val))
					}
					prebuiltConnectors[idx].Configuration.(map[string]interface{})["server_url"] = envVal
				}

			}
			if val, ok := prebuiltConnectors[idx].Configuration.(map[string]interface{})["api_token"]; ok {
				val := val.(string)
				if val[:4] == "<CFG" {
					envVal := os.Getenv(val[1 : len(val)-1])
					if envVal == "" {
						panic(fmt.Sprintf("%s is missing", val))
					}
					prebuiltConnectors[idx].Configuration.(map[string]interface{})["api_token"] = envVal
				}

			}
			if val, ok := prebuiltConnectors[idx].Configuration.(map[string]interface{})["capture_token"]; ok {
				val := val.(string)
				if val[:4] == "<CFG" {
					envVal := os.Getenv(val[1 : len(val)-1])
					if envVal == "" {
						panic(fmt.Sprintf("%s is missing", val))
					}
					prebuiltConnectors[idx].Configuration.(map[string]interface{})["capture_token"] = envVal
				}
			}

			config, err := json.Marshal(prebuiltConnectors[idx].Configuration)
			if err != nil {
				panic(err)
			}
			connectorType := "CONNECTOR_TYPE_AI"
			if prebuiltConnectors[idx].Id == "instill-number" {
				connectorType = "CONNECTOR_TYPE_BLOCKCHAIN"

			}
			connector := &Connector{
				BaseDynamic: BaseDynamic{
					UID: uuid.FromStringOrNil(prebuiltConnectors[idx].Uid),
				},
				ID:                     prebuiltConnectors[idx].Id,
				Owner:                  prebuiltConnectors[idx].Owner,
				ConnectorDefinitionUID: uuid.FromStringOrNil(prebuiltConnectors[idx].ConnectorDefinitionUid),
				Tombstone:              false,
				Configuration:          config,
				ConnectorType:          connectorType,
				Visibility:             "VISIBILITY_PUBLIC",
				State:                  "STATE_CONNECTED",
				Task:                   prebuiltConnectors[idx].Task,
			}

			if result := db.Model(&Connector{}).Clauses(clause.OnConflict{
				UpdateAll: true,
			}).Create(connector); result.Error != nil {
				panic(result.Error)
			}

		}

	}

	// Set tombstone based on definition
	connectors := connector.InitConnectorAll(logger)
	definitions := connectors.ListConnectorDefinitions()
	for idx := range definitions {
		if definitions[idx].Tombstone {
			db.Unscoped().Model(&datamodel.ConnectorResource{}).Where("connector_definition_uid = ?", definitions[idx].Uid).Update("tombstone", true)
		}
	}

}
