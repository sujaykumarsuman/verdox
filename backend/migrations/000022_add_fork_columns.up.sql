ALTER TABLE repositories ADD COLUMN fork_full_name VARCHAR(255);
ALTER TABLE repositories ADD COLUMN fork_status VARCHAR(32) NOT NULL DEFAULT 'none';
ALTER TABLE repositories ADD COLUMN fork_synced_at TIMESTAMPTZ;
ALTER TABLE repositories ADD COLUMN fork_workflow_id VARCHAR(255);
