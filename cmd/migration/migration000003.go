package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/connector-backend/config"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	database "github.com/instill-ai/connector-backend/pkg/db"
)

type Model struct {
	ID          string    `json:"id,omitempty"`
	UID         uuid.UUID `gorm:"type:uuid;primary_key;<-:create"`
	Owner       string    `json:"owner,omitempty"`
	Visibility  string    `json:"string,omitempty"`
	Task        string    `sql:"type:string"`
	Description string    `sql:"description:string"`
}
type Model_Visibility int32

const (
	// Visibility: UNSPECIFIED, equivalent to PRIVATE.
	Model_VISIBILITY_UNSPECIFIED Model_Visibility = 0
	// Visibility: PRIVATE
	Model_VISIBILITY_PRIVATE Model_Visibility = 1
	// Visibility: PUBLIC
	Model_VISIBILITY_PUBLIC Model_Visibility = 2
)

// BaseStatic contains common columns for all tables with static UUID as primary key
type BaseStatic struct {
	UID        uuid.UUID      `gorm:"type:uuid;primary_key;<-:create"` // allow read and create
	CreateTime time.Time      `gorm:"autoCreateTime:nano"`
	UpdateTime time.Time      `gorm:"autoUpdateTime:nano"`
	DeleteTime gorm.DeletedAt `sql:"index"`
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

func Migrate000003() error {
	fmt.Println("Migrate000003")

	var modelDb *gorm.DB
	databaseConfig := config.Config.Database
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=%s",
		databaseConfig.Host,
		databaseConfig.Username,
		databaseConfig.Password,
		"model",
		databaseConfig.Port,
		databaseConfig.TimeZone,
	)
	var err error
	modelDb, err = gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true, // disables implicit prepared statement usage
	}), &gorm.Config{
		QueryFields:          true, // QueryFields mode will select by all fields’ name for current model
		FullSaveAssociations: true,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})

	if err != nil {
		fmt.Println("no model db, skip")
		return nil
	}

	modelSqlDB, _ := modelDb.DB()

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	modelSqlDB.SetMaxIdleConns(databaseConfig.Pool.IdleConnections)
	// SetMaxOpenConns sets the maximum number of open connections to the database.
	modelSqlDB.SetMaxOpenConns(databaseConfig.Pool.MaxConnections)
	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	modelSqlDB.SetConnMaxLifetime(databaseConfig.Pool.ConnLifeTime)

	var items []Model
	result := modelDb.Unscoped().Model(&Model{})
	if result.Error != nil {
		return result.Error
	}

	rows, err := result.Rows()
	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		var item Model
		if err = modelDb.ScanRows(rows, &item); err != nil {
			return err
		}
		items = append(items, item)

	}

	db := database.GetConnection()
	defer database.Close(db)

	if err != nil {
		panic(err)
	}
	for idx := range items {
		oriConnID := items[idx].ID
		connID := oriConnID
		for {
			var totalSize int64
			db.Unscoped().Model(&Connector{}).Where("owner = ? and id = ?", items[idx].Owner, connID).Count(&totalSize)
			if totalSize > 0 {
				var letters = []byte("abcdefghijklmnopqrstuvwxyz")

				b := make([]byte, 3)
				r := rand.New(rand.NewSource(time.Now().UnixNano()))

				for i := range b {
					b[i] = letters[r.Intn(len(letters))]
				}

				connID = fmt.Sprintf("%s-%s", oriConnID, string(b))
			} else {
				break
			}
		}
		edition := strings.Split(config.Config.Server.Edition, ":")[0]
		serverUrl := ""
		if edition == "local-ce" {
			serverUrl = "http://localhost:9080"
		} else {
			serverUrl = "https://api.instill.tech/model"
		}
		connDefUID := "ddcf42c3-4c30-4c65-9585-25f1c89b2b48"
		m, err := structpb.NewValue(map[string]interface{}{
			"api_key":    "",
			"server_url": serverUrl,
			"model_id":   connID,
		})
		if err != nil {
			panic(err)
		}
		configuration, err := structpb.NewStruct(m.GetStructValue().AsMap())
		if err != nil {
			panic(err)
		}
		configurationJson, err := configuration.MarshalJSON()
		if err != nil {
			panic(err)
		}
		connector := &Connector{
			BaseDynamic: BaseDynamic{
				UID: items[idx].UID,
			},
			ID:                     connID,
			Owner:                  items[idx].Owner,
			Description:            items[idx].Description,
			ConnectorDefinitionUID: uuid.FromStringOrNil(connDefUID),
			Tombstone:              false,
			Configuration:          configurationJson,
			ConnectorType:          "CONNECTOR_TYPE_AI",
			Visibility:             items[idx].Visibility,
			State:                  "STATE_DISCONNECTED",
			Task:                   items[idx].Task,
		}
		if result := db.Model(&Connector{}).Create(connector); result.Error != nil {
			panic(result.Error)
		}

	}

	if modelDb != nil {
		modelSqlDB, _ := modelDb.DB()
		modelSqlDB.Close()
	}
	return nil
}
