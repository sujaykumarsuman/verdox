ALTER TABLE repositories DROP COLUMN IF EXISTS fork_full_name;
ALTER TABLE repositories DROP COLUMN IF EXISTS fork_status;
ALTER TABLE repositories DROP COLUMN IF EXISTS fork_synced_at;
ALTER TABLE repositories DROP COLUMN IF EXISTS fork_workflow_id;
