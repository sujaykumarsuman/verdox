CREATE TABLE team_repositories (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id       UUID        NOT NULL,
    repository_id UUID        NOT NULL,
    added_by      UUID,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_team_repos_team_repo UNIQUE (team_id, repository_id),

    CONSTRAINT fk_team_repos_team
        FOREIGN KEY (team_id)
        REFERENCES teams (id)
        ON DELETE CASCADE,

    CONSTRAINT fk_team_repos_repository
        FOREIGN KEY (repository_id)
        REFERENCES repositories (id)
        ON DELETE CASCADE,

    CONSTRAINT fk_team_repos_added_by
        FOREIGN KEY (added_by)
        REFERENCES users (id)
        ON DELETE SET NULL
);

CREATE UNIQUE INDEX idx_team_repos_team_repo ON team_repositories (team_id, repository_id);
