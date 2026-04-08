CREATE TABLE teams (
    id                       UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name                     VARCHAR(128) NOT NULL,
    slug                     VARCHAR(128) NOT NULL,
    created_by               UUID,
    github_pat_encrypted     TEXT,
    github_pat_nonce         BYTEA,
    github_pat_set_at        TIMESTAMPTZ,
    github_pat_set_by        UUID,
    github_pat_github_username VARCHAR(255),
    is_discoverable          BOOLEAN      NOT NULL DEFAULT true,
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at               TIMESTAMPTZ,

    CONSTRAINT uq_teams_name UNIQUE (name),
    CONSTRAINT uq_teams_slug UNIQUE (slug),

    CONSTRAINT fk_teams_created_by
        FOREIGN KEY (created_by)
        REFERENCES users (id)
        ON DELETE SET NULL,

    CONSTRAINT fk_teams_pat_set_by
        FOREIGN KEY (github_pat_set_by)
        REFERENCES users (id)
        ON DELETE SET NULL
);

CREATE UNIQUE INDEX idx_teams_slug ON teams (slug);
