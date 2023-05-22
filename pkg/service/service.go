package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/allegro/bigcache"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/gofrs/uuid"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/connector-backend/config"
	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/connector-backend/pkg/logger"
	"github.com/instill-ai/connector-backend/pkg/repository"
	"github.com/instill-ai/x/sterr"

	dockerClient "github.com/docker/docker/client"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	controllerPB "github.com/instill-ai/protogen-go/vdp/controller/v1alpha"
	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

// Service interface
type Service interface {
	GetMgmtPrivateServiceClient() mgmtPB.MgmtPrivateServiceClient

	// ConnectorDefinition
	ListConnectorDefinitions(connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) ([]*datamodel.ConnectorDefinition, int64, string, error)
	GetConnectorDefinitionByID(id string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.ConnectorDefinition, error)
	GetConnectorDefinitionByUID(uid uuid.UUID, isBasicView bool) (*datamodel.ConnectorDefinition, error)

	// Connector common
	CreateConnector(owner *mgmtPB.User, connector *datamodel.Connector) (*datamodel.Connector, error)
	ListConnectors(owner *mgmtPB.User, connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) ([]*datamodel.Connector, int64, string, error)
	GetConnectorByID(id string, owner *mgmtPB.User, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.Connector, error)
	GetConnectorByUID(uid uuid.UUID, owner *mgmtPB.User, isBasicView bool) (*datamodel.Connector, error)
	UpdateConnector(id string, owner *mgmtPB.User, connectorType datamodel.ConnectorType, updatedConnector *datamodel.Connector) (*datamodel.Connector, error)
	UpdateConnectorID(id string, owner *mgmtPB.User, connectorType datamodel.ConnectorType, newID string) (*datamodel.Connector, error)
	UpdateConnectorState(id string, owner *mgmtPB.User, connectorType datamodel.ConnectorType, state datamodel.ConnectorState) (*datamodel.Connector, error)
	DeleteConnector(id string, owner *mgmtPB.User, connectorType datamodel.ConnectorType) error

	ListConnectorsAdmin(connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) ([]*datamodel.Connector, int64, string, error)
	GetConnectorByUIDAdmin(uid uuid.UUID, isBasicView bool) (*datamodel.Connector, error)

	// Source connector custom service
	ReadSourceConnector(id string, owner *mgmtPB.User) ([]byte, error)

	// Destination connector custom service
	WriteDestinationConnector(id string, owner *mgmtPB.User, param datamodel.WriteDestinationConnectorParam) error

	// Shared public/private method for checking connector's connection
	CheckConnectorByUID(connUID uuid.UUID, dockerRepo string, dockerImgTag string) (*connectorPB.Connector_State, error)

	// Controller custom service
	GetResourceState(uid uuid.UUID, connectorType datamodel.ConnectorType) (*connectorPB.Connector_State, error)
	UpdateResourceState(uid uuid.UUID, connectorType datamodel.ConnectorType, state connectorPB.Connector_State, progress *int32) error
	DeleteResourceState(uid uuid.UUID, connectorType datamodel.ConnectorType) error
}

type service struct {
	repository                  repository.Repository
	cache                       *bigcache.BigCache
	mgmtPrivateServiceClient    mgmtPB.MgmtPrivateServiceClient
	pipelinePublicServiceClient pipelinePB.PipelinePublicServiceClient
	controllerClient            controllerPB.ControllerPrivateServiceClient
	dockerClient                *dockerClient.Client
}

// NewService initiates a service instance
func NewService(
	r repository.Repository,
	u mgmtPB.MgmtPrivateServiceClient,
	p pipelinePB.PipelinePublicServiceClient,
	c controllerPB.ControllerPrivateServiceClient,
	d *dockerClient.Client,
) Service {

	logger, _ := logger.GetZapLogger()

	cache, err := bigcache.NewBigCache(bigcache.DefaultConfig(60 * time.Minute))
	if err != nil {
		logger.Error(err.Error())
	}

	return &service{
		repository:                  r,
		cache:                       cache,
		mgmtPrivateServiceClient:    u,
		pipelinePublicServiceClient: p,
		controllerClient:            c,
		dockerClient:                d,
	}
}

// GetMgmtPrivateServiceClient returns the management private service client
func (s *service) GetMgmtPrivateServiceClient() mgmtPB.MgmtPrivateServiceClient {
	return s.mgmtPrivateServiceClient
}

func (s *service) ListConnectorDefinitions(connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) ([]*datamodel.ConnectorDefinition, int64, string, error) {
	return s.repository.ListConnectorDefinitions(connectorType, pageSize, pageToken, isBasicView)
}

func (s *service) GetConnectorDefinitionByID(id string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.ConnectorDefinition, error) {
	return s.repository.GetConnectorDefinitionByID(id, connectorType, isBasicView)
}

func (s *service) GetConnectorDefinitionByUID(uid uuid.UUID, isBasicView bool) (*datamodel.ConnectorDefinition, error) {
	return s.repository.GetConnectorDefinitionByUID(uid, isBasicView)
}

func (s *service) CreateConnector(owner *mgmtPB.User, connector *datamodel.Connector) (*datamodel.Connector, error) {

	logger, _ := logger.GetZapLogger()

	ownerPermalink := GenOwnerPermalink(owner)

	connector.Owner = ownerPermalink

	connDef, err := s.repository.GetConnectorDefinitionByUID(connector.ConnectorDefinitionUID, true)
	if err != nil {
		return nil, err
	}

	// Validation: HTTP and gRPC connector
	if strings.Contains(connDef.ID, "http") || strings.Contains(connDef.ID, "grpc") {
		if connector.ID != connDef.ID {
			st, err := sterr.CreateErrorBadRequest(
				"[service] create connector",
				[]*errdetails.BadRequest_FieldViolation{
					{
						Field:       "id",
						Description: fmt.Sprintf("Connector id must be %s", connDef.ID),
					},
				},
			)
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, st.Err()
		}

		if connector.Configuration.String() != "{}" {
			st, err := sterr.CreateErrorBadRequest(
				"[service] create connector",
				[]*errdetails.BadRequest_FieldViolation{
					{
						Field:       "connector.configuration",
						Description: fmt.Sprintf("%s connector configuration must be an empty JSON", connDef.ID),
					},
				},
			)
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, st.Err()
		}

		if existingConnector, _ := s.GetConnectorByID(connector.ID, owner, connector.ConnectorType, true); existingConnector != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.AlreadyExists,
				"[service] create connector",
				"connectors",
				fmt.Sprintf("Connector id %s and connector_type %s", connector.ID, connectorPB.ConnectorType(connector.ConnectorType)),
				connector.Owner,
				"Already exists",
			)
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, st.Err()
		}
	}

	if err := s.repository.CreateConnector(connector); err != nil {
		return nil, err
	}

	// User desire state = CONNECTED
	if err := s.repository.UpdateConnectorStateByID(connector.ID, connector.Owner, connector.ConnectorType, datamodel.ConnectorState(connectorPB.Connector_STATE_CONNECTED)); err != nil {
		return nil, err
	}

	// Check connector state and update resource state in etcd
	if strings.Contains(connDef.ID, "http") || strings.Contains(connDef.ID, "grpc") {
		// HTTP and gRPC connector is always with STATE_CONNECTED
		if err := s.UpdateResourceState(connector.UID, connector.ConnectorType, connectorPB.Connector_STATE_CONNECTED, nil); err != nil {
			return nil, err
		}
	} else {
		if state, err := s.CheckConnectorByUID(connector.UID, connDef.DockerRepository, connDef.DockerImageTag); err == nil {
			if err := s.UpdateResourceState(connector.UID, connector.ConnectorType, *state, nil); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	dbConnector, err := s.repository.GetConnectorByID(connector.ID, ownerPermalink, connector.ConnectorType, false)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil

}

func (s *service) ListConnectors(owner *mgmtPB.User, connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) ([]*datamodel.Connector, int64, string, error) {

	ownerPermalink := GenOwnerPermalink(owner)

	dbConnectors, pageSize, pageToken, err := s.repository.ListConnectors(ownerPermalink, connectorType, pageSize, pageToken, isBasicView)
	if err != nil {
		return nil, 0, "", err
	}

	return dbConnectors, pageSize, pageToken, nil
}

func (s *service) ListConnectorsAdmin(connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) ([]*datamodel.Connector, int64, string, error) {

	dbConnectors, pageSize, pageToken, err := s.repository.ListConnectorsAdmin(connectorType, pageSize, pageToken, isBasicView)
	if err != nil {
		return nil, 0, "", err
	}

	return dbConnectors, pageSize, pageToken, nil
}

func (s *service) GetConnectorByID(id string, owner *mgmtPB.User, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.Connector, error) {

	ownerPermalink := GenOwnerPermalink(owner)

	dbConnector, err := s.repository.GetConnectorByID(id, ownerPermalink, connectorType, isBasicView)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil
}

func (s *service) GetConnectorByUID(uid uuid.UUID, owner *mgmtPB.User, isBasicView bool) (*datamodel.Connector, error) {

	ownerPermalink := GenOwnerPermalink(owner)

	dbConnector, err := s.repository.GetConnectorByUID(uid, ownerPermalink, isBasicView)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil
}

func (s *service) GetConnectorByUIDAdmin(uid uuid.UUID, isBasicView bool) (*datamodel.Connector, error) {

	dbConnector, err := s.repository.GetConnectorByUIDAdmin(uid, isBasicView)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil
}

func (s *service) UpdateConnector(id string, owner *mgmtPB.User, connectorType datamodel.ConnectorType, updatedConnector *datamodel.Connector) (*datamodel.Connector, error) {

	logger, _ := logger.GetZapLogger()

	ownerPermalink := GenOwnerPermalink(owner)

	updatedConnector.Owner = ownerPermalink

	// Validation: HTTP and gRPC connector cannot be updated
	existingConnector, err := s.repository.GetConnectorByID(id, ownerPermalink, connectorType, true)
	if err != nil {
		return nil, err
	}

	def, err := s.repository.GetConnectorDefinitionByUID(existingConnector.ConnectorDefinitionUID, true)
	if err != nil {
		return nil, err
	}

	if strings.Contains(def.ID, "http") || strings.Contains(def.ID, "grpc") {
		st, err := sterr.CreateErrorPreconditionFailure(
			"[service] update connector",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "UPDATE",
					Subject:     fmt.Sprintf("id %s and connector_type %s", id, connectorPB.ConnectorType(connectorType)),
					Description: fmt.Sprintf("Cannot update a %s connector", id),
				},
			})
		if err != nil {
			logger.Error(err.Error())
		}
		return nil, st.Err()
	}

	if err := s.repository.UpdateConnector(id, ownerPermalink, connectorType, updatedConnector); err != nil {
		return nil, err
	}

	// Check connector state
	if state, err := s.CheckConnectorByUID(existingConnector.UID, def.DockerRepository, def.DockerImageTag); err == nil {
		if err := s.UpdateResourceState(updatedConnector.UID, connectorType, *state, nil); err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}

	dbConnector, err := s.repository.GetConnectorByID(updatedConnector.ID, ownerPermalink, updatedConnector.ConnectorType, false)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil
}

func (s *service) DeleteConnector(id string, owner *mgmtPB.User, connectorType datamodel.ConnectorType) error {
	logger, _ := logger.GetZapLogger()

	ownerPermalink := GenOwnerPermalink(owner)

	dbConnector, err := s.repository.GetConnectorByID(id, ownerPermalink, connectorType, false)
	if err != nil {
		return err
	}

	var filter string
	switch {
	case connectorType == datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE):
		filter = fmt.Sprintf("recipe.components.resource_name:\"source-connectors/%s\"", dbConnector.UID)
	case connectorType == datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION):
		filter = fmt.Sprintf("recipe.components.resource_name:\"destination-connectors/%s\"", dbConnector.UID)
	}

	pipeResp, err := s.pipelinePublicServiceClient.ListPipelines(InjectOwnerToContext(context.Background(), owner), &pipelinePB.ListPipelinesRequest{
		Filter: &filter,
	})
	if err != nil {
		return err
	}

	if len(pipeResp.Pipelines) > 0 {
		var pipeIDs []string
		for _, pipe := range pipeResp.Pipelines {
			pipeIDs = append(pipeIDs, pipe.GetId())
		}
		st, err := sterr.CreateErrorPreconditionFailure(
			"[service] delete connector",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "DELETE",
					Subject:     fmt.Sprintf("id %s and connector_type %s", id, connectorPB.ConnectorType(connectorType)),
					Description: fmt.Sprintf("The connector is still in use by pipeline: %s", strings.Join(pipeIDs, " ")),
				},
			})
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	}

	if err := s.DeleteResourceState(dbConnector.UID, connectorType); err != nil {
		return err
	}

	return s.repository.DeleteConnector(id, ownerPermalink, connectorType)
}

func (s *service) UpdateConnectorState(id string, owner *mgmtPB.User, connectorType datamodel.ConnectorType, state datamodel.ConnectorState) (*datamodel.Connector, error) {

	logger, _ := logger.GetZapLogger()

	ownerPermalink := GenOwnerPermalink(owner)

	// Validation: HTTP and gRPC connector cannot be disconnected
	conn, err := s.repository.GetConnectorByID(id, ownerPermalink, connectorType, true)
	if err != nil {
		return nil, err
	}

	connDef, err := s.repository.GetConnectorDefinitionByUID(conn.ConnectorDefinitionUID, true)
	if err != nil {
		return nil, err
	}

	connState, err := s.GetResourceState(conn.UID, connectorType)
	if err != nil {
		return nil, err
	}

	switch *connState {
	case connectorPB.Connector_STATE_ERROR:
		st, err := sterr.CreateErrorPreconditionFailure(
			"[service] update connector state",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "STATE",
					Subject:     fmt.Sprintf("id %s and connector_type %s", id, connectorPB.ConnectorType(connectorType)),
					Description: "The connector is in STATE_ERROR",
				},
			})
		if err != nil {
			logger.Error(err.Error())
		}
		return nil, st.Err()
	case connectorPB.Connector_STATE_UNSPECIFIED:
		st, err := sterr.CreateErrorPreconditionFailure(
			"[service] update connector state",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "STATE",
					Subject:     fmt.Sprintf("id %s and connector_type %s", id, connectorPB.ConnectorType(connectorType)),
					Description: "The connector is in STATE_UNSPECIFIED",
				},
			})
		if err != nil {
			logger.Error(err.Error())
		}
		return nil, st.Err()
	}

	switch state {
	case datamodel.ConnectorState(connectorPB.Connector_STATE_CONNECTED):

		if strings.Contains(connDef.ID, "http") || strings.Contains(connDef.ID, "grpc") {
			break
		}

		// Set connector state to user desire state
		if err := s.repository.UpdateConnectorStateByID(id, ownerPermalink, connectorType, datamodel.ConnectorState(connectorPB.Connector_STATE_CONNECTED)); err != nil {
			return nil, err
		}

		// Check resource state
		if datamodel.ConnectorState(*connState) != state {
			if state, err := s.CheckConnectorByUID(conn.UID, connDef.DockerRepository, connDef.DockerImageTag); err == nil {
				if err := s.UpdateResourceState(conn.UID, connectorType, *state, nil); err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		}

	case datamodel.ConnectorState(connectorPB.Connector_STATE_DISCONNECTED):

		if strings.Contains(connDef.ID, "http") || strings.Contains(connDef.ID, "grpc") {
			st, err := sterr.CreateErrorPreconditionFailure(
				"[service] update connector state",
				[]*errdetails.PreconditionFailure_Violation{
					{
						Type:        "STATE",
						Subject:     fmt.Sprintf("id %s and connector_type %s", id, connectorPB.ConnectorType(connectorType)),
						Description: fmt.Sprintf("Cannot disconnect a %s connector", connDef.ID),
					},
				})
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, st.Err()
		}

		if err := s.repository.UpdateConnectorStateByID(id, ownerPermalink, connectorType, state); err != nil {
			return nil, err
		}
		if err := s.UpdateResourceState(conn.UID, connectorType, connectorPB.Connector_State(state), nil); err != nil {
			return nil, err
		}
	}

	dbConnector, err := s.repository.GetConnectorByID(id, ownerPermalink, connectorType, false)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil
}

func (s *service) UpdateConnectorID(id string, owner *mgmtPB.User, connectorType datamodel.ConnectorType, newID string) (*datamodel.Connector, error) {

	logger, _ := logger.GetZapLogger()

	ownerPermalink := GenOwnerPermalink(owner)

	// Validation: HTTP and gRPC connectors cannot be renamed
	existingConnector, err := s.repository.GetConnectorByID(id, ownerPermalink, connectorType, true)
	if err != nil {
		return nil, err
	}

	def, err := s.repository.GetConnectorDefinitionByUID(existingConnector.ConnectorDefinitionUID, true)
	if err != nil {
		return nil, err
	}

	if strings.Contains(def.ID, "http") || strings.Contains(def.ID, "grpc") {
		st, err := sterr.CreateErrorPreconditionFailure(
			"[service] update connector id",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "RENAME",
					Subject:     fmt.Sprintf("id %s and connector_type %s", id, connectorPB.ConnectorType(connectorType)),
					Description: fmt.Sprintf("Cannot rename a %s connector", def.ID),
				},
			})
		if err != nil {
			logger.Error(err.Error())
		}
		return nil, st.Err()
	}

	// if err := s.DeleteResourceState(id, connectorType); err != nil {
	// 	return nil, err
	// }

	// if err := s.UpdateResourceState(newID, connectorType, connectorPB.Connector_State(existingConnector.State), nil, nil); err != nil {
	// 	return nil, err
	// }

	if err := s.repository.UpdateConnectorID(id, ownerPermalink, connectorType, newID); err != nil {
		return nil, err
	}

	dbConnector, err := s.repository.GetConnectorByID(newID, ownerPermalink, connectorType, false)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil
}

func (s *service) ReadSourceConnector(id string, owner *mgmtPB.User) ([]byte, error) {
	// TODO: Implement async source destination
	return nil, nil
}

func (s *service) WriteDestinationConnector(id string, owner *mgmtPB.User, param datamodel.WriteDestinationConnectorParam) error {

	logger, _ := logger.GetZapLogger()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ownerPermalink := GenOwnerPermalink(owner)

	conn, err := s.repository.GetConnectorByID(id, ownerPermalink, datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION), false)
	if err != nil {
		return err
	}

	connDef, err := s.repository.GetConnectorDefinitionByUID(conn.ConnectorDefinitionUID, true)
	if err != nil {
		return err
	}

	// Create ConfiguredAirbyteCatalog
	cfgAbCatalog := datamodel.ConfiguredAirbyteCatalog{
		Streams: []datamodel.ConfiguredAirbyteStream{
			{
				Stream:              &datamodel.TaskOutputAirbyteCatalog.Streams[0],
				SyncMode:            param.SyncMode,
				DestinationSyncMode: param.DstSyncMode,
			},
		},
	}

	byteCfgAbCatalog, err := json.Marshal(&cfgAbCatalog)
	if err != nil {
		return fmt.Errorf("Marshal AirbyteMessage error: %w", err)
	}

	// Create AirbyteMessage RECORD type, i.e., AirbyteRecordMessage in JSON Line format
	var byteAbMsgs []byte

	for _, modelOutput := range param.ModelOutputs {

		for idx, taskOutput := range modelOutput.TaskOutputs {

			b, err := protojson.MarshalOptions{
				UseProtoNames:   true,
				EmitUnpopulated: true,
			}.Marshal(taskOutput)
			if err != nil {
				return fmt.Errorf("task_outputs[%d] error: %w", idx, err)
			}

			dataStruct := structpb.Struct{}
			err = protojson.Unmarshal(b, &dataStruct)
			if err != nil {
				return fmt.Errorf("task_outputs[%d] error: %w", idx, err)
			}

			b, err = protojson.MarshalOptions{
				UseProtoNames:   true,
				EmitUnpopulated: true,
			}.Marshal(param.Recipe)
			if err != nil {
				return fmt.Errorf("task_outputs[%d] error: %w", idx, err)
			}

			recipeStruct := structpb.Struct{}
			err = protojson.Unmarshal(b, &recipeStruct)
			if err != nil {
				return fmt.Errorf("task_outputs[%d] error: %w", idx, err)
			}

			pipelineStruct := structpb.Struct{}
			pipelineStruct.Fields = make(map[string]*structpb.Value)
			pipelineStruct.GetFields()["name"] = structpb.NewStringValue(param.Pipeline)
			pipelineStruct.GetFields()["recipe"] = structpb.NewStructValue(&recipeStruct)

			dataStruct.GetFields()["pipeline"] = structpb.NewStructValue(&pipelineStruct)
			dataStruct.GetFields()["model"] = structpb.NewStringValue(modelOutput.Model)
			dataStruct.GetFields()["index"] = structpb.NewStringValue(param.DataMappingIndices[idx])

			b, err = protojson.Marshal(&dataStruct)
			if err != nil {
				return fmt.Errorf("task_outputs[%d] error: %w", idx, err)
			}

			abMsg := datamodel.AirbyteMessage{}
			abMsg.Type = "RECORD"
			abMsg.Record = &datamodel.AirbyteRecordMessage{
				Stream:    datamodel.TaskOutputAirbyteCatalog.Streams[0].Name,
				Data:      b,
				EmittedAt: time.Now().UnixMilli(),
			}

			b, err = json.Marshal(&abMsg)
			if err != nil {
				return fmt.Errorf("Marshal AirbyteMessage error: %w", err)
			}
			b = []byte(string(b) + "\n")
			byteAbMsgs = append(byteAbMsgs, b...)
		}

	}

	// Remove the last "\n"
	byteAbMsgs = byteAbMsgs[:len(byteAbMsgs)-1]

	imageName := fmt.Sprintf("%s:%s", connDef.DockerRepository, connDef.DockerImageTag)
	containerName := fmt.Sprintf("%s.%d.write", conn.UID, time.Now().UnixNano())
	configFileName := fmt.Sprintf("%s.%d.write", conn.UID, time.Now().UnixNano())
	catalogFileName := fmt.Sprintf("%s.%d.write", conn.UID, time.Now().UnixNano())

	// If there is already a container run dispatched in the previous attempt, return exitCodeOK directly
	if _, err := s.cache.Get(containerName); err == nil {
		return nil
	}

	// Write config into a container local file (always overwrite)
	configFilePath := fmt.Sprintf("%s/connector-data/config/%s.json", config.Config.Container.MountTarget.VDP, configFileName)
	if err := os.MkdirAll(filepath.Dir(configFilePath), os.ModePerm); err != nil {
		return fmt.Errorf(fmt.Sprintf("unable to create folders for filepath %s", configFilePath), "WriteContainerLocalFileError", err)
	}
	if err := os.WriteFile(configFilePath, conn.Configuration, 0644); err != nil {
		return fmt.Errorf(fmt.Sprintf("unable to write connector config file %s", configFilePath), "WriteContainerLocalFileError", err)
	}

	// Write catalog into a container local file (always overwrite)
	catalogFilePath := fmt.Sprintf("%s/connector-data/catalog/%s.json", config.Config.Container.MountTarget.VDP, catalogFileName)
	if err := os.MkdirAll(filepath.Dir(catalogFilePath), os.ModePerm); err != nil {
		return fmt.Errorf(fmt.Sprintf("unable to create folders for filepath %s", catalogFilePath), "WriteContainerLocalFileError", err)
	}
	if err := os.WriteFile(catalogFilePath, byteCfgAbCatalog, 0644); err != nil {
		return fmt.Errorf(fmt.Sprintf("unable to write connector catalog file %s", catalogFilePath), "WriteContainerLocalFileError", err)
	}

	defer func() {
		// Delete config local file
		if _, err := os.Stat(configFilePath); err == nil {
			if err := os.Remove(configFilePath); err != nil {
				logger.Error(fmt.Sprintln("Activity", "ImageName", imageName, "ContainerName", containerName, "Error", err))
			}
		}

		// Delete catalog local file
		if _, err := os.Stat(catalogFilePath); err == nil {
			if err := os.Remove(catalogFilePath); err != nil {
				logger.Error(fmt.Sprintln("Activity", "ImageName", imageName, "ContainerName", containerName, "Error", err))
			}
		}
	}()

	out, err := s.dockerClient.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(os.Stdout, out); err != nil {
		return err
	}

	resp, err := s.dockerClient.ContainerCreate(ctx,
		&container.Config{
			Image:        imageName,
			AttachStdin:  true,
			AttachStdout: true,
			OpenStdin:    true,
			StdinOnce:    true,
			Tty:          true,
			Cmd: []string{
				"write",
				"--config",
				configFilePath,
				"--catalog",
				catalogFilePath,
			},
		},
		&container.HostConfig{
			Mounts: []mount.Mount{
				{
					Type: func() mount.Type {
						if string(config.Config.Container.MountSource.VDP[0]) == "/" {
							return mount.TypeBind
						}
						return mount.TypeVolume
					}(),
					Source: config.Config.Container.MountSource.VDP,
					Target: config.Config.Container.MountTarget.VDP,
				},
				{
					Type: func() mount.Type {
						if string(config.Config.Container.MountSource.VDP[0]) == "/" {
							return mount.TypeBind
						}
						return mount.TypeVolume
					}(),
					Source: config.Config.Container.MountSource.Airbyte,
					Target: config.Config.Container.MountTarget.Airbyte,
				},
			},
		},
		nil, nil, containerName)
	if err != nil {
		return err
	}

	hijackedResp, err := s.dockerClient.ContainerAttach(ctx, resp.ID, types.ContainerAttachOptions{
		Stdout: true,
		Stdin:  true,
		Stream: true,
	})
	if err != nil {
		logger.Error(err.Error())
	}

	// need to append "\n" and "ctrl+D" at the end of the input message
	_, err = hijackedResp.Conn.Write(append(byteAbMsgs, 10, 4))
	if err != nil {
		logger.Error(err.Error())
	}

	if err := s.dockerClient.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	var bufStdOut bytes.Buffer
	if _, err := bufStdOut.ReadFrom(hijackedResp.Reader); err != nil {
		return err
	}

	if err := s.dockerClient.ContainerRemove(ctx, resp.ID,
		types.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		}); err != nil {
		return err
	}

	// Set cache flag (empty value is fine since we need only the entry record)
	if err := s.cache.Set(containerName, []byte{}); err != nil {
		logger.Error(err.Error())
	}

	logger.Info(fmt.Sprintln("Activity",
		"ImageName", imageName,
		"ContainerName", containerName,
		"Pipeline", param.Pipeline,
		"Indices", param.DataMappingIndices,
		"STDOUT", bufStdOut.String()))

	// Delete the cache entry only after the write completed
	if err := s.cache.Delete(containerName); err != nil {
		logger.Error(err.Error())
	}

	return nil
}

func (s *service) CheckConnectorByUID(connUID uuid.UUID, dockerRepo string, dockerImgTag string) (*connectorPB.Connector_State, error) {
	logger, _ := logger.GetZapLogger()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbConnector, err := s.repository.GetConnectorByUIDAdmin(connUID, false)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("cannot get the connector, RepositoryError: %v", err))
	}

	if strings.Contains(dbConnector.ID, "http") || strings.Contains(dbConnector.ID, "grpc") {
		return connectorPB.Connector_STATE_CONNECTED.Enum(), nil
	}

	imageName := fmt.Sprintf("%s:%s", dockerRepo, dockerImgTag)
	containerName := fmt.Sprintf("%s.%d.check", connUID, time.Now().UnixNano())
	configFilePath := fmt.Sprintf("%s/connector-data/config/%s.json", config.Config.Container.MountTarget.VDP, containerName)
	// Write config into a container local file
	if err := os.MkdirAll(filepath.Dir(configFilePath), os.ModePerm); err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("unable to create folders for filepath %s", configFilePath), "WriteContainerLocalFileError", err)
	}
	if err := os.WriteFile(configFilePath, dbConnector.Configuration, 0644); err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("unable to write connector config file %s", configFilePath), "WriteContainerLocalFileError", err)
	}

	defer func() {
		// Delete config local file
		if _, err := os.Stat(configFilePath); err == nil {
			if err := os.Remove(configFilePath); err != nil {
				logger.Error(fmt.Sprintf("ImageName: %s, ContainerName: %s, Error: %v", imageName, containerName, err))
			}
		}
	}()

	out, err := s.dockerClient.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		return nil, err
	}
	defer out.Close()

	if _, err := io.Copy(os.Stdout, out); err != nil {
		return nil, err
	}

	resp, err := s.dockerClient.ContainerCreate(ctx,
		&container.Config{
			Image: imageName,
			Tty:   false,
			Cmd: []string{
				"check",
				"--config",
				configFilePath,
			},
		},
		&container.HostConfig{
			Mounts: []mount.Mount{
				{
					Type: func() mount.Type {
						if string(config.Config.Container.MountSource.VDP[0]) == "/" {
							return mount.TypeBind
						}
						return mount.TypeVolume
					}(),
					Source: config.Config.Container.MountSource.VDP,
					Target: config.Config.Container.MountTarget.VDP,
				},
			},
		},
		nil, nil, containerName)
	if err != nil {
		return nil, err
	}

	if err := s.dockerClient.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return nil, err
	}

	statusCh, errCh := s.dockerClient.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return nil, err
		}
	case <-statusCh:
	}

	if out, err = s.dockerClient.ContainerLogs(ctx,
		resp.ID,
		types.ContainerLogsOptions{
			ShowStdout: true,
		},
	); err != nil {
		return nil, err
	}

	if err := s.dockerClient.ContainerRemove(ctx, containerName,
		types.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		}); err != nil {
		return nil, err
	}

	var bufStdOut, bufStdErr bytes.Buffer
	if _, err := stdcopy.StdCopy(&bufStdOut, &bufStdErr, out); err != nil {
		return nil, err
	}

	var teeStdOut io.Reader = strings.NewReader(bufStdOut.String())
	var teeStdErr io.Reader = strings.NewReader(bufStdErr.String())
	teeStdOut = io.TeeReader(teeStdOut, &bufStdOut)
	teeStdErr = io.TeeReader(teeStdErr, &bufStdErr)

	var byteStdOut, byteStdErr []byte
	if byteStdOut, err = io.ReadAll(teeStdOut); err != nil {
		return nil, err
	}
	if byteStdErr, err = io.ReadAll(teeStdErr); err != nil {
		return nil, err
	}

	logger.Info(fmt.Sprintf("ImageName, %s, ContainerName, %s, STDOUT, %v, STDERR, %v", imageName, containerName, byteStdOut, byteStdErr))

	scanner := bufio.NewScanner(&bufStdOut)
	for scanner.Scan() {

		if err := scanner.Err(); err != nil {
			return nil, err
		}

		var jsonMsg map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &jsonMsg); err == nil {
			switch jsonMsg["type"] {
			case "CONNECTION_STATUS":
				switch jsonMsg["connectionStatus"].(map[string]interface{})["status"] {
				case "SUCCEEDED":
					if err := s.UpdateResourceState(dbConnector.UID, dbConnector.ConnectorType, connectorPB.Connector_STATE_CONNECTED, nil); err != nil {
						return nil, err
					}
					return connectorPB.Connector_STATE_CONNECTED.Enum(), nil
				case "FAILED":
					if err := s.UpdateResourceState(dbConnector.UID, dbConnector.ConnectorType, connectorPB.Connector_STATE_ERROR, nil); err != nil {
						return nil, err
					}
					return connectorPB.Connector_STATE_ERROR.Enum(), nil
				default:
					if err := s.UpdateResourceState(dbConnector.UID, dbConnector.ConnectorType, connectorPB.Connector_STATE_ERROR, nil); err != nil {
						return nil, err
					}
					return connectorPB.Connector_STATE_ERROR.Enum(), fmt.Errorf("UNKNOWN STATUS")
				}
			}
		}
	}
	return nil, fmt.Errorf("unable to scan container stdout and find the connection status successfully")
}
