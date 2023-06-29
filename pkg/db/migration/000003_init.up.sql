BEGIN;

ALTER TYPE valid_connector_type ADD VALUE IF NOT EXISTS 'CONNECTOR_TYPE_AI';
ALTER TYPE valid_connector_type ADD VALUE IF NOT EXISTS 'CONNECTOR_TYPE_BLOCKCHAIN';

CREATE TYPE valid_visibility AS ENUM (
  'VISIBILITY_UNSPECIFIED',
  'VISIBILITY_PRIVATE',
  'VISIBILITY_PUBLIC'
);

ALTER TABLE public.connector ADD COLUMN "visibility" VALID_VISIBILITY DEFAULT 'VISIBILITY_PRIVATE' NOT NULL;

DROP INDEX IF EXISTS unique_owner_id_connector_type_deleted_at;
CREATE UNIQUE INDEX unique_owner_id_deleted_at ON public.connector (owner, id) WHERE delete_time IS NULL;

COMMIT;
