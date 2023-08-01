BEGIN;

ALTER TYPE valid_connector_type ADD VALUE IF NOT EXISTS 'CONNECTOR_TYPE_DATA';
ALTER TYPE valid_connector_type ADD VALUE IF NOT EXISTS 'CONNECTOR_TYPE_OPERATOR';

ALTER TYPE valid_task ADD VALUE IF NOT EXISTS 'TASK_SPEECH_RECOGNITION';

COMMIT;

UPDATE public.connector SET connector_type = 'CONNECTOR_TYPE_DATA'
                        WHERE (connector_type = 'CONNECTOR_TYPE_DESTINATION'
                            AND connector_definition_uid != '909c3278-f7d1-461c-9352-87741bef11d3'
                            AND connector_definition_uid != 'c0e4a82c-9620-4a72-abd1-18586f2acccd');

-- rename trigger to start-operator
UPDATE public.connector SET id = 'start-operator', connector_type = 'CONNECTOR_TYPE_OPERATOR'
                        WHERE connector_definition_uid = 'f20a3c02-c70e-4e76-8566-7c13ca11d18d';

-- rename response to end-operator
UPDATE public.connector SET id = 'end-operator', connector_type = 'CONNECTOR_TYPE_OPERATOR'
                        WHERE connector_definition_uid = '909c3278-f7d1-461c-9352-87741bef11d3';


COMMIT;
