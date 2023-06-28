BEGIN;

ALTER TYPE valid_connector_type ADD VALUE IF NOT EXISTS 'CONNECTOR_TYPE_AI';
ALTER TYPE valid_connector_type ADD VALUE IF NOT EXISTS 'CONNECTOR_TYPE_BLOCKCHAIN';

DROP INDEX IF EXISTS unique_owner_id_connector_type_deleted_at;
CREATE UNIQUE INDEX unique_owner_id_deleted_at ON public.connector (owner, id) WHERE delete_time IS NULL;

COMMIT;
