CREATE TABLE team_members (
    id          UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id     UUID               NOT NULL,
    user_id     UUID               NOT NULL,
    role        team_member_role   NOT NULL DEFAULT 'viewer',
    status      team_member_status NOT NULL DEFAULT 'pending',
    invited_by  UUID,
    created_at  TIMESTAMPTZ        NOT NULL DEFAULT now(),

    CONSTRAINT uq_team_members_team_user UNIQUE (team_id, user_id),

    CONSTRAINT fk_team_members_team
        FOREIGN KEY (team_id)
        REFERENCES teams (id)
        ON DELETE CASCADE,

    CONSTRAINT fk_team_members_user
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE,

    CONSTRAINT fk_team_members_invited_by
        FOREIGN KEY (invited_by)
        REFERENCES users (id)
        ON DELETE SET NULL
);

CREATE UNIQUE INDEX idx_team_members_team_user ON team_members (team_id, user_id);
CREATE INDEX        idx_team_members_user_id   ON team_members (user_id);
