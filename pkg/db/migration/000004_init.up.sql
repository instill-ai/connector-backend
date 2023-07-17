BEGIN;

-- rename source-http to trigger
UPDATE public.connector SET id = 'trigger'
                        WHERE connector_definition_uid = 'f20a3c02-c70e-4e76-8566-7c13ca11d18d';

-- rename destination-http to trigger
UPDATE public.connector SET id = 'response'
                        WHERE connector_definition_uid = '909c3278-f7d1-461c-9352-87741bef11d3';

ALTER TYPE valid_task ADD VALUE IF NOT EXISTS 'TASK_IMAGE_TO_IMAGE';
ALTER TYPE valid_task ADD VALUE IF NOT EXISTS 'TASK_TEXT_EMBEDDINGS';

COMMIT;
