CREATE TABLE team_join_requests (
    id              UUID                PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID                NOT NULL,
    user_id         UUID                NOT NULL,
    message         TEXT,
    status          team_member_status  NOT NULL DEFAULT 'pending',
    reviewed_by     UUID,
    role_assigned   team_member_role,
    created_at      TIMESTAMPTZ         NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ         NOT NULL DEFAULT now(),

    CONSTRAINT uq_join_requests_team_user UNIQUE (team_id, user_id),

    CONSTRAINT fk_join_requests_team
        FOREIGN KEY (team_id)
        REFERENCES teams (id)
        ON DELETE CASCADE,

    CONSTRAINT fk_join_requests_user
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE,

    CONSTRAINT fk_join_requests_reviewer
        FOREIGN KEY (reviewed_by)
        REFERENCES users (id)
        ON DELETE SET NULL
);

CREATE INDEX idx_join_requests_team_id ON team_join_requests (team_id);
CREATE INDEX idx_join_requests_user_id ON team_join_requests (user_id);
CREATE INDEX idx_join_requests_status  ON team_join_requests (status);
