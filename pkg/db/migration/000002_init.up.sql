BEGIN;

ALTER TABLE public.connector DROP CONSTRAINT connector_connector_definition_uid_fkey;
DROP TABLE IF EXISTS public.connector_definition;

COMMIT;
