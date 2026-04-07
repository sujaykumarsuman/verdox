DROP INDEX IF EXISTS idx_test_runs_report_id;
ALTER TABLE test_runs DROP COLUMN IF EXISTS report_id;
ALTER TABLE test_runs DROP COLUMN IF EXISTS summary;

DROP TABLE IF EXISTS test_cases;
DROP TABLE IF EXISTS test_groups;
