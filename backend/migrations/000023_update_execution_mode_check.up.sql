-- Allow fork_gha execution mode and update default
ALTER TABLE test_suites DROP CONSTRAINT IF EXISTS chk_execution_mode;
ALTER TABLE test_suites ADD CONSTRAINT chk_execution_mode
    CHECK (execution_mode IN ('container', 'gha', 'fork_gha'));
ALTER TABLE test_suites ALTER COLUMN execution_mode SET DEFAULT 'fork_gha';
