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
CREATE TYPE valid_state AS ENUM (
  'STATE_UNSPECIFIED',
  'STATE_DISCONNECTED',
  'STATE_CONNECTED',
  'STATE_ERROR'
);
-- conector_definition
CREATE TABLE IF NOT EXISTS public.connector_definition(
  "uid" UUID NOT NULL,
  "id" VARCHAR(255) NOT NULL,
  "title" VARCHAR(255) NOT NULL,
  "docker_repository" VARCHAR(255) NOT NULL,
  "docker_image_tag" VARCHAR(255) NOT NULL,
  "documentation_url" VARCHAR(1023) NULL,
  "icon" VARCHAR(255) NULL,
  "connector_type" VALID_CONNECTOR_TYPE DEFAULT 'CONNECTOR_TYPE_UNSPECIFIED' NOT NULL,
  "connection_type" VALID_CONNECTION_TYPE DEFAULT 'CONNECTION_TYPE_UNSPECIFIED' NOT NULL,
  "spec" JSONB NOT NULL,
  "tombstone" BOOL DEFAULT FALSE NOT NULL,
  "release_stage" VALID_RELEASE_STAGE DEFAULT 'RELEASE_STAGE_UNSPECIFIED' NOT NULL,
  "release_date" DATE NULL,
  "resource_requirements" JSONB NOT NULL,
  "public" BOOL DEFAULT FALSE NOT NULL,
  "custom" BOOL DEFAULT FALSE NOT NULL,
  "create_time" TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "update_time" TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "delete_time" TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NULL,
  CONSTRAINT connector_definition_pkey PRIMARY KEY (uid)
);
CREATE INDEX connector_definition_uid_create_time_pagination ON public.connector_definition (uid, create_time);
-- connector
CREATE TABLE IF NOT EXISTS public.connector(
  "uid" UUID NOT NULL,
  "id" VARCHAR(255) NOT NULL,
  "connector_definition_uid" UUID NOT NULL,
  "owner" VARCHAR(255) NOT NULL,
  "description" VARCHAR(1023) NULL,
  "configuration" JSONB NOT NULL,
  "connector_type" VALID_CONNECTOR_TYPE DEFAULT 'CONNECTOR_TYPE_UNSPECIFIED' NOT NULL,
  "state" VALID_STATE DEFAULT 'STATE_UNSPECIFIED' NOT NULL,
  "tombstone" BOOL DEFAULT FALSE NOT NULL,
  "create_time" TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "update_time" TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "delete_time" TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NULL,
  CONSTRAINT connector_pkey PRIMARY KEY (uid)
);
ALTER TABLE public.connector
ADD CONSTRAINT connector_connector_definition_uid_fkey FOREIGN KEY (connector_definition_uid) REFERENCES public.connector_definition (uid);
CREATE INDEX connector_connector_definition_uid_idx ON public.connector (connector_definition_uid ASC);
CREATE UNIQUE INDEX unique_owner_id_connector_type_deleted_at ON public.connector (owner, id, connector_type)
WHERE delete_time IS NULL;
CREATE INDEX connector_uid_create_time_pagination ON public.connector (uid, create_time);
COMMIT;
