-- Enum expansion must be in its own migration (cannot use new values in same transaction)
ALTER TYPE test_result_status ADD VALUE IF NOT EXISTS 'running';
ALTER TYPE test_result_status ADD VALUE IF NOT EXISTS 'unknown';
