DROP INDEX IF EXISTS idx_users_is_banned;

ALTER TABLE users DROP COLUMN IF EXISTS is_banned;
