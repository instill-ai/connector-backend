BEGIN;
CREATE TYPE valid_connector_type AS ENUM (
  'CONNECTOR_TYPE_UNSPECIFIED',
  'CONNECTOR_TYPE_SOURCE',
  'CONNECTOR_TYPE_DESTINATION'
);
CREATE TYPE valid_connection_type AS ENUM (
  'CONNECTION_TYPE_UNSPECIFIED',
  'CONNECTION_TYPE_DIRECTNESS',
  'CONNECTION_TYPE_FILE',
  'CONNECTION_TYPE_API',
  'CONNECTION_TYPE_DATABASE',
  'CONNECTION_TYPE_CUSTOM'
);
CREATE TYPE valid_release_stage AS ENUM (
  'RELEASE_STAGE_UNSPECIFIED',
  'RELEASE_STAGE_ALPHA',
  'RELEASE_STAGE_BETA',
  'RELEASE_STAGE_GENERALLY_AVAILABLE',
  'RELEASE_STAGE_CUSTOM'
);
-- conector_definition
CREATE TABLE IF NOT EXISTS public.connector_definition(
  id UUID NOT NULL,
  name VARCHAR(256) NOT NULL,
  docker_repository VARCHAR(256) NOT NULL,
  docker_image_tag VARCHAR(256) NOT NULL,
  documentation_url VARCHAR(1024) NULL,
  icon VARCHAR(256) NULL,
  connector_type VALID_CONNECTOR_TYPE NOT NULL,
  connection_type VALID_CONNECTION_TYPE NOT NULL,
  spec JSONB NOT NULL,
  tombstone BOOL DEFAULT FALSE NOT NULL,
  release_stage VALID_RELEASE_STAGE NOT NULL,
  release_date DATE NULL,
  resource_requirements JSONB NOT NULL,
  public BOOL DEFAULT FALSE NOT NULL,
  custom BOOL DEFAULT FALSE NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
  deleted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NULL,
  CONSTRAINT connector_definition_pkey PRIMARY KEY (id)
);
CREATE INDEX connector_definition_id_created_at_pagination ON public.connector_definition (id, created_at);
-- connector
CREATE TABLE IF NOT EXISTS public.connector(
  id UUID NOT NULL,
  connector_definition_id UUID NOT NULL,
  owner_id UUID NOT NULL,
  name VARCHAR(256) NOT NULL,
  description VARCHAR(1024) NULL,
  configuration JSONB NOT NULL,
  connector_type VALID_CONNECTOR_TYPE NOT NULL,
  tombstone BOOL DEFAULT FALSE NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
  deleted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NULL,
  CONSTRAINT connector_pkey PRIMARY KEY (id)
);
ALTER TABLE public.connector
ADD CONSTRAINT unique_owner_id_name_connector_type_deleted_at UNIQUE (owner_id, name, connector_type, deleted_at);
ALTER TABLE public.connector
ADD CONSTRAINT connector_connector_definition_id_fkey FOREIGN KEY (connector_definition_id) REFERENCES public.connector_definition (id);
CREATE INDEX connector_connector_definition_id_idx ON public.connector (connector_definition_id ASC);
CREATE INDEX connector_id_created_at_pagination ON public.connector (id, created_at);
COMMIT;
