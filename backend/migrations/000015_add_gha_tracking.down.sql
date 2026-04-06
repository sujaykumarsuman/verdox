DROP INDEX IF EXISTS idx_test_runs_gha_run_id;

ALTER TABLE test_runs
    DROP COLUMN IF EXISTS log_output,
    DROP COLUMN IF EXISTS gha_run_id;
