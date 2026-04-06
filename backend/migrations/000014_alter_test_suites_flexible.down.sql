ALTER TABLE test_suites DROP CONSTRAINT IF EXISTS chk_execution_mode;

ALTER TABLE test_suites
    DROP COLUMN IF EXISTS env_vars,
    DROP COLUMN IF EXISTS gha_workflow_id,
    DROP COLUMN IF EXISTS test_command,
    DROP COLUMN IF EXISTS docker_image,
    DROP COLUMN IF EXISTS execution_mode;

-- Revert type back to enum (only safe if all values are 'unit' or 'integration')
ALTER TABLE test_suites
    ALTER COLUMN type TYPE test_type USING type::test_type;
