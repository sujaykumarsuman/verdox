ALTER TABLE users ADD COLUMN is_banned BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX idx_users_is_banned ON users (is_banned);
