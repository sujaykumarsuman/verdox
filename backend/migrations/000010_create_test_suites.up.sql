CREATE TABLE test_suites (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id   UUID         NOT NULL,
    name            VARCHAR(255) NOT NULL,
    type            VARCHAR(50)  NOT NULL DEFAULT 'unit',
    execution_mode  VARCHAR(20)  NOT NULL DEFAULT 'fork_gha',
    docker_image    VARCHAR(255),
    test_command    TEXT,
    gha_workflow_id VARCHAR(255),
    env_vars        JSONB        NOT NULL DEFAULT '{}'::jsonb,
    config_path     TEXT,
    timeout_seconds INTEGER      NOT NULL DEFAULT 300,
    workflow_config JSONB        NOT NULL DEFAULT '{}',
    workflow_yaml   TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT fk_test_suites_repository
        FOREIGN KEY (repository_id)
        REFERENCES repositories (id)
        ON DELETE CASCADE,

    CONSTRAINT chk_execution_mode CHECK (execution_mode IN ('fork_gha'))
);

CREATE INDEX idx_test_suites_repository_id ON test_suites (repository_id);
