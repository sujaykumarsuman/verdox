-- Add GHA workflow run tracking and aggregated log output to test_runs.

ALTER TABLE test_runs
    ADD COLUMN gha_run_id  BIGINT,
    ADD COLUMN log_output  TEXT;

CREATE INDEX idx_test_runs_gha_run_id ON test_runs (gha_run_id) WHERE gha_run_id IS NOT NULL;
