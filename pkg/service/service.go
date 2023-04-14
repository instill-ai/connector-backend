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

	"github.com/gofrs/uuid"
	"go.temporal.io/sdk/client"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/pkg/stdcopy"
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
	GetConnectorByIDAdmin(id string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.Connector, error)
	GetConnectorByUIDAdmin(uid uuid.UUID, isBasicView bool) (*datamodel.Connector, error)

	// Source connector custom service
	ReadSourceConnector(id string, owner *mgmtPB.User) ([]byte, error)

	// Destination connector custom service
	WriteDestinationConnector(id string, owner *mgmtPB.User, param datamodel.WriteDestinationConnectorParam) error

	// Longrunning operation
	CheckConnectorAdmin(connUID uuid.UUID, dockerRepo string, dockerImgTag string) (*connectorPB.Connector_State, error)

	// Controller custom service
	GetResourceState(connectorName string, connectorType datamodel.ConnectorType) (*connectorPB.Connector_State, error)
	UpdateResourceState(connectorName string, connectorType datamodel.ConnectorType, state connectorPB.Connector_State, progress *int32, workflowId *string) error
	DeleteResourceState(connectorName string, connectorType datamodel.ConnectorType) error
}

type service struct {
	repository                  repository.Repository
	mgmtPrivateServiceClient    mgmtPB.MgmtPrivateServiceClient
	pipelinePublicServiceClient pipelinePB.PipelinePublicServiceClient
	temporalClient              client.Client
	controllerClient            controllerPB.ControllerPrivateServiceClient
	dockerClient                *dockerClient.Client
	mountSourceVDP              string
	mountTargetVDP              string
	mountSourceAirbyte          string
	mountTargetAirbyte          string
}

// NewService initiates a service instance
func NewService(
	r repository.Repository,
	u mgmtPB.MgmtPrivateServiceClient,
	p pipelinePB.PipelinePublicServiceClient,
	t client.Client,
	c controllerPB.ControllerPrivateServiceClient,
	dc *dockerClient.Client,
) Service {
	return &service{
		repository:                  r,
		mgmtPrivateServiceClient:    u,
		pipelinePublicServiceClient: p,
		temporalClient:              t,
		controllerClient:            c,
		dockerClient:                dc,
		mountSourceVDP:              config.Config.Worker.MountSource.VDP,
		mountTargetVDP:              config.Config.Worker.MountTarget.VDP,
		mountSourceAirbyte:          config.Config.Worker.MountSource.Airbyte,
		mountTargetAirbyte:          config.Config.Worker.MountTarget.Airbyte,
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

	ownerRscName := owner.GetName()
	ownerPermalink := "users/" + owner.GetUid()

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
		if err := s.UpdateResourceState(connector.ID, connector.ConnectorType, connectorPB.Connector_STATE_CONNECTED, nil, nil); err != nil {
			return nil, err
		}
	} else {
		if err := s.UpdateResourceState(connector.ID, connector.ConnectorType, connectorPB.Connector_STATE_UNSPECIFIED, nil, nil); err != nil {
			return nil, err
		}
	}

	dbConnector, err := s.repository.GetConnectorByID(connector.ID, ownerPermalink, connector.ConnectorType, false)
	if err != nil {
		return nil, err
	}

	dbConnector.Owner = ownerRscName

	return dbConnector, nil

}

func (s *service) ListConnectors(owner *mgmtPB.User, connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) ([]*datamodel.Connector, int64, string, error) {

	ownerRscName := owner.GetName()
	ownerPermalink := "users/" + owner.GetUid()

	dbConnectors, pageSize, pageToken, err := s.repository.ListConnectors(ownerPermalink, connectorType, pageSize, pageToken, isBasicView)
	if err != nil {
		return nil, 0, "", err
	}

	for _, dbConnector := range dbConnectors {
		dbConnector.Owner = ownerRscName
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

	ownerRscName := owner.GetName()
	ownerPermalink := "users/" + owner.GetUid()

	dbConnector, err := s.repository.GetConnectorByID(id, ownerPermalink, connectorType, isBasicView)
	if err != nil {
		return nil, err
	}

	dbConnector.Owner = ownerRscName

	return dbConnector, nil
}

func (s *service) GetConnectorByIDAdmin(id string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.Connector, error) {

	dbConnector, err := s.repository.GetConnectorByIDAdmin(id, connectorType, isBasicView)
	if err != nil {
		return nil, err
	}

	return dbConnector, nil
}

func (s *service) GetConnectorByUID(uid uuid.UUID, owner *mgmtPB.User, isBasicView bool) (*datamodel.Connector, error) {

	ownerRscName := owner.GetName()
	ownerPermalink := "users/" + owner.GetUid()

	dbConnector, err := s.repository.GetConnectorByUID(uid, ownerPermalink, isBasicView)
	if err != nil {
		return nil, err
	}

	dbConnector.Owner = ownerRscName

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

	ownerRscName := owner.GetName()
	ownerPermalink := "users/" + owner.GetUid()

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

	if err := s.UpdateResourceState(id, connectorType, connectorPB.Connector_STATE_UNSPECIFIED, nil, nil); err != nil {
		return nil, err
	}

	dbConnector, err := s.repository.GetConnectorByID(updatedConnector.ID, ownerPermalink, updatedConnector.ConnectorType, false)
	if err != nil {
		return nil, err
	}

	dbConnector.Owner = ownerRscName

	return dbConnector, nil
}

func (s *service) DeleteConnector(id string, owner *mgmtPB.User, connectorType datamodel.ConnectorType) error {
	logger, _ := logger.GetZapLogger()

	ownerPermalink := "users/" + owner.GetUid()

	dbConnector, err := s.repository.GetConnectorByID(id, ownerPermalink, connectorType, false)
	if err != nil {
		return err
	}

	var filter string
	switch {
	case connectorType == datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE):
		filter = fmt.Sprintf("recipe.source:\"%s\"", dbConnector.UID)
	case connectorType == datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION):
		filter = fmt.Sprintf("recipe.destination:\"%s\"", dbConnector.UID)
	}

	pipeResp, err := s.pipelinePublicServiceClient.ListPipelines(context.Background(), &pipelinePB.ListPipelinesRequest{
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

	if err := s.DeleteResourceState(id, connectorType); err != nil {
		return err
	}

	return s.repository.DeleteConnector(id, ownerPermalink, connectorType)
}

func (s *service) UpdateConnectorState(id string, owner *mgmtPB.User, connectorType datamodel.ConnectorType, state datamodel.ConnectorState) (*datamodel.Connector, error) {

	logger, _ := logger.GetZapLogger()

	ownerRscName := owner.GetName()
	ownerPermalink := "users/" + owner.GetUid()

	// Validation: HTTP and gRPC connector cannot be disconnected
	conn, err := s.repository.GetConnectorByID(id, ownerPermalink, connectorType, true)
	if err != nil {
		return nil, err
	}

	connDef, err := s.repository.GetConnectorDefinitionByUID(conn.ConnectorDefinitionUID, true)
	if err != nil {
		return nil, err
	}

	connState, err := s.GetResourceState(id, connectorType)
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

		// Set resource state to STATE_UNSPECIFIED
		if datamodel.ConnectorState(*connState) != state {
			if err := s.UpdateResourceState(id, connectorType, connectorPB.Connector_STATE_UNSPECIFIED, nil, nil); err != nil {
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
		if err := s.UpdateResourceState(id, connectorType, connectorPB.Connector_State(state), nil, nil); err != nil {
			return nil, err
		}
	}

	dbConnector, err := s.repository.GetConnectorByID(id, ownerPermalink, connectorType, false)
	if err != nil {
		return nil, err
	}

	dbConnector.Owner = ownerRscName

	return dbConnector, nil
}

func (s *service) UpdateConnectorID(id string, owner *mgmtPB.User, connectorType datamodel.ConnectorType, newID string) (*datamodel.Connector, error) {

	logger, _ := logger.GetZapLogger()

	ownerRscName := owner.GetName()
	ownerPermalink := "users/" + owner.GetUid()

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

	if err := s.DeleteResourceState(id, connectorType); err != nil {
		return nil, err
	}

	if err := s.UpdateResourceState(newID, connectorType, connectorPB.Connector_State(existingConnector.State), nil, nil); err != nil {
		return nil, err
	}

	if err := s.repository.UpdateConnectorID(id, ownerPermalink, connectorType, newID); err != nil {
		return nil, err
	}

	dbConnector, err := s.repository.GetConnectorByID(newID, ownerPermalink, connectorType, false)
	if err != nil {
		return nil, err
	}

	dbConnector.Owner = ownerRscName

	return dbConnector, nil
}

func (s *service) ReadSourceConnector(id string, owner *mgmtPB.User) ([]byte, error) {
	// TODO: Implement async source destination
	return nil, nil
}

func (s *service) WriteDestinationConnector(id string, owner *mgmtPB.User, param datamodel.WriteDestinationConnectorParam) error {

	ownerPermalink := "users/" + owner.GetUid()

	conn, err := s.repository.GetConnectorByID(id, ownerPermalink, datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION), true)
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

	// Start Temporal worker
	_, err = s.startWriteWorkflow(
		ownerPermalink, conn.UID.String(),
		connDef.DockerRepository, connDef.DockerImageTag,
		param.Pipeline, param.DataMappingIndices,
		byteCfgAbCatalog, byteAbMsgs)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) CheckConnectorAdmin(connUID uuid.UUID, dockerRepo string, dockerImgTag string) (*connectorPB.Connector_State, error) {
	logger, _ := logger.GetZapLogger()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbConnector, err := s.repository.GetConnectorByUIDAdmin(connUID, false)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("cannot get the connector, RepositoryError: %v", err))
	}

	imageName := fmt.Sprintf("%s:%s", dockerRepo, dockerImgTag)
	containerName := fmt.Sprintf("%s.%d.check", connUID, time.Now().UnixNano())
	configFilePath := fmt.Sprintf("%s/connector-data/config/%s.json", s.mountTargetVDP, containerName)
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
						if string(s.mountSourceVDP[0]) == "/" {
							return mount.TypeBind
						}
						return mount.TypeVolume
					}(),
					Source: s.mountSourceVDP,
					Target: s.mountTargetVDP,
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
					return connectorPB.Connector_STATE_CONNECTED.Enum(), nil
				case "FAILED":
					return connectorPB.Connector_STATE_ERROR.Enum(), nil
				default:
					return connectorPB.Connector_STATE_ERROR.Enum(), fmt.Errorf("UNKNOWN STATUS")
				}
			}
		}
	}
	return nil, fmt.Errorf("unable to scan container stdout and find the connection status successfully")
}
