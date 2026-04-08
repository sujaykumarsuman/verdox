CREATE TABLE users (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    username        VARCHAR(64) NOT NULL,
    email           VARCHAR(255) NOT NULL,
    password_hash   TEXT         NOT NULL,
    role            user_role    NOT NULL DEFAULT 'user',
    avatar_url      TEXT,
    is_active       BOOLEAN      NOT NULL DEFAULT TRUE,
    is_banned       BOOLEAN      NOT NULL DEFAULT FALSE,
    ban_reason      TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uq_users_username UNIQUE (username),
    CONSTRAINT uq_users_email    UNIQUE (email)
);

CREATE UNIQUE INDEX idx_users_username  ON users (username);
CREATE UNIQUE INDEX idx_users_email     ON users (email);
CREATE INDEX        idx_users_is_active ON users (is_active);
CREATE INDEX        idx_users_is_banned ON users (is_banned);
