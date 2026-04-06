-- Convert test_suites.type from enum to VARCHAR for arbitrary suite types.
-- Add execution_mode, docker_image, test_command, gha_workflow_id, env_vars columns.

ALTER TABLE test_suites
    ALTER COLUMN type TYPE VARCHAR(50) USING type::text;

ALTER TABLE test_suites
    ALTER COLUMN type SET DEFAULT 'unit';

ALTER TABLE test_suites
    ADD COLUMN execution_mode VARCHAR(20) NOT NULL DEFAULT 'container',
    ADD COLUMN docker_image   VARCHAR(255),
    ADD COLUMN test_command   TEXT,
    ADD COLUMN gha_workflow_id VARCHAR(255),
    ADD COLUMN env_vars       JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE test_suites
    ADD CONSTRAINT chk_execution_mode CHECK (execution_mode IN ('container', 'gha'));
