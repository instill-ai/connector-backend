BEGIN;
CREATE TYPE valid_connector_type AS ENUM ('CONNECTOR_TYPE_SOURCE', 'CONNECTOR_TYPE_DESTINATION');
CREATE TYPE valid_connection_type AS ENUM (
  'CONNECTION_TYPE_DIRECTNESS',
  'CONNECTION_TYPE_FILE',
  'CONNECTION_TYPE_API',
  'CONNECTION_TYPE_DATABASE',
  'CONNECTION_TYPE_CUSTOM'
);
CREATE TYPE valid_release_stage AS ENUM (
  'RELEASE_STAGE_ALPHA',
  'RELEASE_STAGE_BETA',
  'RELEASE_STAGE_GENERALLY_AVAILABLE',
  'RELEASE_STAGE_CUSTOM'
);
-- conector_definition
create table public.connector_definition(
  id UUID NOT NULL,
  name VARCHAR(256) NOT NULL,
  docker_repository VARCHAR(256) NOT NULL,
  docker_image_tag VARCHAR(256) NOT NULL,
  documentation_url VARCHAR(256) NULL,
  icon VARCHAR(256) NULL,
  connector_type VALID_CONNECTOR_TYPE NOT NULL,
  connection_type VALID_CONNECTION_TYPE NULL,
  spec JSONB NOT NULL,
  tombstone BOOL NOT NULL DEFAULT FALSE,
  release_stage VALID_RELEASE_STAGE NULL,
  release_date date NULL,
  resource_requirements JSONB  NOT NULL,
  public BOOL NOT NULL DEFAULT FALSE,
  custom BOOL NOT NULL DEFAULT FALSE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
  deleted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT connector_definition_pkey PRIMARY KEY (id)
);
CREATE INDEX connector_definition_id_created_at_pagination ON public.connector_definition (id, created_at);
-- connector
CREATE TABLE public.connector(
  id UUID NOT NULL,
  workspace_id UUID NOT NULL,
  connector_definition_id UUID NOT NULL,
  name VARCHAR(256) NOT NULL,
  configuration JSONB NOT NULL,
  connector_type VALID_CONNECTOR_TYPE NOT NULL,
  tombstone BOOL NOT NULL DEFAULT FALSE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
  deleted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT connector_pkey PRIMARY KEY (id)
);
ALTER TABLE public.connector ADD CONSTRAINT actor_actor_definition_id_fkey foreign key (connector_definition_id) references public.connector_definition (id);
CREATE INDEX connector_connector_definition_id_idx on public.connector (connector_definition_id ASC);
COMMIT;
