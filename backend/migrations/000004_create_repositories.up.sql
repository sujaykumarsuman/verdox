CREATE TABLE repositories (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    github_repo_id   BIGINT       NOT NULL,
    github_full_name VARCHAR(255) NOT NULL,
    name             VARCHAR(255) NOT NULL,
    description      TEXT,
    default_branch   VARCHAR(255) NOT NULL DEFAULT 'main',
    local_path       TEXT,
    clone_status     VARCHAR(32)  NOT NULL DEFAULT 'pending',
    is_active        BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uq_repositories_github_repo_id UNIQUE (github_repo_id),
    CONSTRAINT chk_clone_status CHECK (clone_status IN ('pending', 'cloning', 'ready', 'failed', 'evicted'))
);

CREATE UNIQUE INDEX idx_repositories_github_repo_id ON repositories (github_repo_id);
