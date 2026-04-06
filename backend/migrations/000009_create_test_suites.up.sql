CREATE TABLE test_suites (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id   UUID         NOT NULL,
    name            VARCHAR(255) NOT NULL,
    type            test_type    NOT NULL DEFAULT 'unit',
    config_path     TEXT,
    timeout_seconds INTEGER      NOT NULL DEFAULT 300,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT fk_test_suites_repository
        FOREIGN KEY (repository_id)
        REFERENCES repositories (id)
        ON DELETE CASCADE
);

CREATE INDEX idx_test_suites_repository_id ON test_suites (repository_id);
