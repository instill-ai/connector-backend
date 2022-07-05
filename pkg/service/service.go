package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"go.temporal.io/sdk/client"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/connector-backend/internal/logger"
	"github.com/instill-ai/connector-backend/pkg/datamodel"
	"github.com/instill-ai/connector-backend/pkg/repository"
	"github.com/instill-ai/x/sterr"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

// Service interface
type Service interface {
	// ConnectorDefinition
	ListConnectorDefinition(connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) ([]*datamodel.ConnectorDefinition, int64, string, error)
	GetConnectorDefinitionByID(id string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.ConnectorDefinition, error)
	GetConnectorDefinitionByUID(uid uuid.UUID, isBasicView bool) (*datamodel.ConnectorDefinition, error)

	// Connector common
	CreateConnector(connector *datamodel.Connector) (*datamodel.Connector, error)
	ListConnector(ownerRscName string, connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) ([]*datamodel.Connector, int64, string, error)
	GetConnectorByID(id string, ownerRscName string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.Connector, error)
	GetConnectorByUID(uid uuid.UUID, ownerRscName string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.Connector, error)
	UpdateConnector(id string, ownerRscName string, connectorType datamodel.ConnectorType, updatedConnector *datamodel.Connector) (*datamodel.Connector, error)
	UpdateConnectorID(id string, ownerRscName string, connectorType datamodel.ConnectorType, newID string) (*datamodel.Connector, error)
	UpdateConnectorState(id string, ownerRscName string, connectorType datamodel.ConnectorType, state datamodel.ConnectorState) (*datamodel.Connector, error)
	DeleteConnector(id string, ownerRscName string, connectorType datamodel.ConnectorType) error

	// Source connector custom service
	ReadSourceConnector(id string, ownerRscName string) ([]byte, error)

	// Destination connector custom service
	WriteDestinationConnector(id string, ownerRscName string, task modelPB.ModelInstance_Task, batch *structpb.Value) error
}

type service struct {
	repository            repository.Repository
	userServiceClient     mgmtPB.UserServiceClient
	pipelineServiceClient pipelinePB.PipelineServiceClient
	temporalClient        client.Client
}

// NewService initiates a service instance
func NewService(r repository.Repository, u mgmtPB.UserServiceClient, p pipelinePB.PipelineServiceClient, t client.Client) Service {
	return &service{
		repository:            r,
		userServiceClient:     u,
		pipelineServiceClient: p,
		temporalClient:        t,
	}
}

func (s *service) ListConnectorDefinition(connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) ([]*datamodel.ConnectorDefinition, int64, string, error) {
	return s.repository.ListConnectorDefinition(connectorType, pageSize, pageToken, isBasicView)
}

func (s *service) GetConnectorDefinitionByID(id string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.ConnectorDefinition, error) {
	return s.repository.GetConnectorDefinitionByID(id, connectorType, isBasicView)
}

func (s *service) GetConnectorDefinitionByUID(uid uuid.UUID, isBasicView bool) (*datamodel.ConnectorDefinition, error) {
	return s.repository.GetConnectorDefinitionByUID(uid, isBasicView)
}

func (s *service) CreateConnector(connector *datamodel.Connector) (*datamodel.Connector, error) {

	logger, _ := logger.GetZapLogger()

	ownerRscName := connector.Owner
	ownerPermalink, err := s.ownerRscNameToPermalink(ownerRscName)
	if err != nil {
		return nil, err
	}

	connector.Owner = ownerPermalink

	connDef, err := s.repository.GetConnectorDefinitionByUID(connector.ConnectorDefinitionUID, true)
	if err != nil {
		return nil, err
	}

	// Validation: Directness connector
	if connectorPB.ConnectionType(connDef.ConnectionType) == connectorPB.ConnectionType_CONNECTION_TYPE_DIRECTNESS {
		if connector.ID != connDef.ID {
			st, err := sterr.CreateErrorBadRequest(
				"[service] create connector",
				[]*errdetails.BadRequest_FieldViolation{
					{
						Field:       "id",
						Description: fmt.Sprintf("Directness connector with connector_type %s id must be %s", connectorPB.ConnectorType(connector.ConnectorType), connDef.ID),
					},
				},
			)
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, st.Err()
		}

		if connector.Description.String != "" {
			st, err := sterr.CreateErrorBadRequest(
				"[service] create connector",
				[]*errdetails.BadRequest_FieldViolation{
					{
						Field:       "connector.description",
						Description: fmt.Sprintf("Directness connector with connector_type %s description must be an empty", connectorPB.ConnectorType(connector.ConnectorType)),
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
						Description: fmt.Sprintf("Directness connector with connector_type %s configuration must be an empty JSON {}", connectorPB.ConnectorType(connector.ConnectorType)),
					},
				},
			)
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, st.Err()
		}

		if existingConnector, _ := s.GetConnectorByID(connector.ID, connector.Owner, connector.ConnectorType, true); existingConnector != nil {
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

	// Check connector state
	if connectorPB.ConnectionType(connDef.ConnectionType) == connectorPB.ConnectionType_CONNECTION_TYPE_DIRECTNESS {
		// Directness connector is always with STATE_CONNECTED
		if err := s.repository.UpdateConnectorStateByID(connector.ID, connector.Owner, connector.ConnectorType, datamodel.ConnectorState(connectorPB.Connector_STATE_CONNECTED)); err != nil {
			return nil, err
		}
	} else {

		var connCollectionID string
		if connector.ConnectorType == datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE) {
			connCollectionID = "source-connectors"
		} else if connector.ConnectorType == datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION) {
			connCollectionID = "destination-connectors"
		}

		if err := s.startCheckStateWorkflow(
			ownerRscName, ownerPermalink,
			fmt.Sprintf("%s/%s", connCollectionID, connector.ID), fmt.Sprintf("%s/%s", connCollectionID, connector.UID), connector.ConnectorType,
			connDef.DockerRepository, connDef.DockerImageTag); err != nil {
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

func (s *service) ListConnector(ownerRscName string, connectorType datamodel.ConnectorType, pageSize int64, pageToken string, isBasicView bool) ([]*datamodel.Connector, int64, string, error) {

	ownerPermalink, err := s.ownerRscNameToPermalink(ownerRscName)
	if err != nil {
		return nil, 0, "", err
	}

	dbConnectors, pageSize, pageToken, err := s.repository.ListConnector(ownerPermalink, connectorType, pageSize, pageToken, isBasicView)
	if err != nil {
		return nil, 0, "", err
	}

	for _, dbConnector := range dbConnectors {
		dbConnector.Owner = ownerRscName
	}

	return dbConnectors, pageSize, pageToken, nil
}

func (s *service) GetConnectorByID(id string, ownerRscName string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.Connector, error) {

	ownerPermalink, err := s.ownerRscNameToPermalink(ownerRscName)
	if err != nil {
		return nil, err
	}

	dbConnector, err := s.repository.GetConnectorByID(id, ownerPermalink, connectorType, isBasicView)
	if err != nil {
		return nil, err
	}

	dbConnector.Owner = ownerRscName

	return dbConnector, nil
}

func (s *service) GetConnectorByUID(uid uuid.UUID, ownerRscName string, connectorType datamodel.ConnectorType, isBasicView bool) (*datamodel.Connector, error) {

	ownerPermalink, err := s.ownerRscNameToPermalink(ownerRscName)
	if err != nil {
		return nil, err
	}

	dbConnector, err := s.repository.GetConnectorByUID(uid, ownerPermalink, connectorType, isBasicView)
	if err != nil {
		return nil, err
	}

	dbConnector.Owner = ownerRscName

	return dbConnector, nil
}

func (s *service) UpdateConnector(id string, ownerRscName string, connectorType datamodel.ConnectorType, updatedConnector *datamodel.Connector) (*datamodel.Connector, error) {

	logger, _ := logger.GetZapLogger()

	ownerPermalink, err := s.ownerRscNameToPermalink(ownerRscName)
	if err != nil {
		return nil, err
	}

	updatedConnector.Owner = ownerPermalink

	// Validation: Directness connectors cannot be updated
	existingConnector, err := s.repository.GetConnectorByID(id, ownerPermalink, connectorType, true)
	if err != nil {
		return nil, err
	}

	def, err := s.repository.GetConnectorDefinitionByUID(existingConnector.ConnectorDefinitionUID, true)
	if err != nil {
		return nil, err
	}

	if connectorPB.ConnectionType(def.ConnectionType) == connectorPB.ConnectionType_CONNECTION_TYPE_DIRECTNESS {
		st, err := sterr.CreateErrorPreconditionFailure(
			"[service] update connector",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "UPDATE",
					Subject:     fmt.Sprintf("id %s and connector_type %s", id, connectorPB.ConnectorType(connectorType)),
					Description: "Cannot update a directness connector",
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
	var connCollectionID string
	if existingConnector.ConnectorType == datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE) {
		connCollectionID = "source-connectors"
	} else if existingConnector.ConnectorType == datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION) {
		connCollectionID = "destination-connectors"
	}

	if err := s.startCheckStateWorkflow(
		ownerRscName, ownerPermalink,
		fmt.Sprintf("%s/%s", connCollectionID, existingConnector.ID), fmt.Sprintf("%s/%s", connCollectionID, existingConnector.UID), existingConnector.ConnectorType,
		def.DockerRepository, def.DockerImageTag); err != nil {
		return nil, err
	}

	dbConnector, err := s.repository.GetConnectorByID(updatedConnector.ID, ownerPermalink, updatedConnector.ConnectorType, false)
	if err != nil {
		return nil, err
	}

	dbConnector.Owner = ownerRscName

	return dbConnector, nil
}

func (s *service) DeleteConnector(id string, ownerRscName string, connectorType datamodel.ConnectorType) error {

	logger, _ := logger.GetZapLogger()

	ownerPermalink, err := s.ownerRscNameToPermalink(ownerRscName)
	if err != nil {
		return err
	}

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

	pipeResp, err := s.pipelineServiceClient.ListPipeline(context.Background(), &pipelinePB.ListPipelineRequest{
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
					Description: fmt.Sprintf("The connector is still in use by pipeline: %s ", strings.Join(pipeIDs, " ")),
				},
			})
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	}

	return s.repository.DeleteConnector(id, ownerPermalink, connectorType)
}

func (s *service) UpdateConnectorState(id string, ownerRscName string, connectorType datamodel.ConnectorType, state datamodel.ConnectorState) (*datamodel.Connector, error) {

	logger, _ := logger.GetZapLogger()

	ownerPermalink, err := s.ownerRscNameToPermalink(ownerRscName)
	if err != nil {
		return nil, err
	}

	// Validation: Directness connectors cannot be updated
	conn, err := s.repository.GetConnectorByID(id, ownerPermalink, connectorType, true)
	if err != nil {
		return nil, err
	}

	connDef, err := s.repository.GetConnectorDefinitionByUID(conn.ConnectorDefinitionUID, true)
	if err != nil {
		return nil, err
	}

	switch connectorPB.ConnectionType(connDef.ConnectionType) {
	case connectorPB.ConnectionType_CONNECTION_TYPE_DIRECTNESS:
		switch state {
		case datamodel.ConnectorState(connectorPB.Connector_STATE_DISCONNECTED):
			st, err := sterr.CreateErrorPreconditionFailure(
				"[service] update connector state",
				[]*errdetails.PreconditionFailure_Violation{
					{
						Type:        "STATE",
						Subject:     fmt.Sprintf("id %s and connector_type %s", id, connectorPB.ConnectorType(connectorType)),
						Description: "Cannot disconnect a directness connector",
					},
				})
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, st.Err()
		}
	}

	switch conn.State {
	case datamodel.ConnectorState(connectorPB.Connector_STATE_ERROR):
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
	}

	switch state {
	case datamodel.ConnectorState(connectorPB.Connector_STATE_CONNECTED):

		// Set connector state to STATE_UNSPECIFIED when it is set to STATE_CONNECTED from STATE_DISCONNECTED
		if err := s.repository.UpdateConnectorStateByID(id, ownerPermalink, connectorType, datamodel.ConnectorState(connectorPB.Connector_STATE_UNSPECIFIED)); err != nil {
			return nil, err
		}

		var connCollectionID string
		if conn.ConnectorType == datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE) {
			connCollectionID = "source-connectors"
		} else if conn.ConnectorType == datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION) {
			connCollectionID = "destination-connectors"
		}

		if err := s.startCheckStateWorkflow(
			ownerRscName, ownerPermalink,
			fmt.Sprintf("%s/%s", connCollectionID, conn.ID), fmt.Sprintf("%s/%s", connCollectionID, conn.UID), conn.ConnectorType,
			connDef.DockerRepository, connDef.DockerImageTag); err != nil {
			return nil, err
		}

	case datamodel.ConnectorState(connectorPB.Connector_STATE_DISCONNECTED):
		if err := s.repository.UpdateConnectorStateByID(id, ownerPermalink, connectorType, state); err != nil {
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

func (s *service) UpdateConnectorID(id string, ownerRscName string, connectorType datamodel.ConnectorType, newID string) (*datamodel.Connector, error) {

	logger, _ := logger.GetZapLogger()

	ownerPermalink, err := s.ownerRscNameToPermalink(ownerRscName)
	if err != nil {
		return nil, err
	}

	// Validation: Directness connectors cannot be updated
	existingConnector, err := s.repository.GetConnectorByID(id, ownerPermalink, connectorType, true)
	if err != nil {
		return nil, err
	}

	def, err := s.repository.GetConnectorDefinitionByUID(existingConnector.ConnectorDefinitionUID, true)
	if err != nil {
		return nil, err
	}

	if connectorPB.ConnectionType(def.ConnectionType) == connectorPB.ConnectionType_CONNECTION_TYPE_DIRECTNESS {
		st, err := sterr.CreateErrorPreconditionFailure(
			"[service] update connector id",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "RENAME",
					Subject:     fmt.Sprintf("id %s and connector_type %s", id, connectorPB.ConnectorType(connectorType)),
					Description: "Directness connector cannot be renamed",
				},
			})
		if err != nil {
			logger.Error(err.Error())
		}
		return nil, st.Err()
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

func (s *service) ReadSourceConnector(id string, ownerRscName string) ([]byte, error) {
	// TODO: Implement async source destination
	return nil, nil
}

func (s *service) WriteDestinationConnector(id string, ownerRscName string, task modelPB.ModelInstance_Task, batch *structpb.Value) error {

	ownerPermalink, err := s.ownerRscNameToPermalink(ownerRscName)
	if err != nil {
		return err
	}

	conn, err := s.repository.GetConnectorByID(id, ownerPermalink, datamodel.ConnectorType(connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION), true)
	if err != nil {
		return err
	}

	connDef, err := s.repository.GetConnectorDefinitionByUID(conn.ConnectorDefinitionUID, true)
	if err != nil {
		return err
	}

	// Create ConfiguredAirbyteCatalog
	var byteCfgAbCatalog []byte
	cfgCatalog := datamodel.ConfiguredAirbyteCatalog{
		Streams: []datamodel.ConfiguredAirbyteStream{
			{
				Stream:              &datamodel.TaskAirbyteCatalog[task.String()].Streams[0],
				SyncMode:            "full_refresh",
				DestinationSyncMode: "overwrite",
			},
		},
	}

	byteCfgAbCatalog, err = json.Marshal(&cfgCatalog)
	if err != nil {
		return fmt.Errorf("Marshal AirbyteMessage error: %w", err)
	}

	// Create AirbyteMessage RECORD type, i.e., AirbyteRecordMessage in JSON Line format
	var byteAbMsgs []byte
	for idx, value := range batch.GetListValue().GetValues() {

		b, err := protojson.Marshal(value)
		if err != nil {
			return fmt.Errorf("batch[%d] error: %w", idx, err)
		}

		abMsg := datamodel.AirbyteMessage{}
		abMsg.Type = "RECORD"
		abMsg.Record = &datamodel.AirbyteRecordMessage{
			Stream:    datamodel.TaskAirbyteCatalog[task.String()].Streams[0].Name,
			Data:      b,
			EmittedAt: time.Now().Unix(),
		}

		b, err = json.Marshal(&abMsg)
		if err != nil {
			return fmt.Errorf("Marshal AirbyteMessage error: %w", err)
		}
		b = []byte(string(b) + "\n")
		byteAbMsgs = append(byteAbMsgs, b...)
	}

	// Start Temporal worker
	if err := s.startWriteDestinationWorkflow(
		ownerRscName, ownerPermalink,
		fmt.Sprintf("destination-connectors/%s", conn.ID), fmt.Sprintf("destination-connectors/%s", conn.UID),
		connDef.DockerRepository, connDef.DockerImageTag, byteCfgAbCatalog, byteAbMsgs); err != nil {
		return err
	}

	return nil
}
