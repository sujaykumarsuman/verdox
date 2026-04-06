CREATE TABLE test_discoveries (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id   UUID         NOT NULL,
    discovery_json  JSONB        NOT NULL,
    scripts_path    TEXT,
    discovered_by   UUID,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT fk_discoveries_repository
        FOREIGN KEY (repository_id)
        REFERENCES repositories (id)
        ON DELETE CASCADE,

    CONSTRAINT fk_discoveries_user
        FOREIGN KEY (discovered_by)
        REFERENCES users (id)
        ON DELETE SET NULL
);

CREATE INDEX idx_test_discoveries_repo ON test_discoveries (repository_id);
