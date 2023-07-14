BEGIN;

-- rename source-http to trigger
UPDATE public.connector SET id = 'trigger'
                        WHERE connector_definition_uid = 'f20a3c02-c70e-4e76-8566-7c13ca11d18d';

-- update destination-http to trigger and rename to destination-http-deprecated
UPDATE public.connector SET id = 'source-grpc-deprecated',
                            connector_definition_uid = 'f20a3c02-c70e-4e76-8566-7c13ca11d18d'
                        WHERE connector_definition_uid = '82ca7d29-a35c-4222-b900-8d6878195e7a';

-- rename destination-http to trigger
UPDATE public.connector SET id = 'response'
                        WHERE connector_definition_uid = '909c3278-f7d1-461c-9352-87741bef11d3';

-- update destination-grpc to trigger and rename to destination-grpc-deprecated
UPDATE public.connector SET id = 'destination-grpc-deprecated',
                            connector_definition_uid = '909c3278-f7d1-461c-9352-87741bef11d3'
                        WHERE connector_definition_uid = 'c0e4a82c-9620-4a72-abd1-18586f2acccd';

COMMIT;
