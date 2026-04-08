CREATE TABLE test_cases (
    id              UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    test_group_id   UUID               NOT NULL,
    test_run_id     UUID               NOT NULL,
    case_id         VARCHAR(512)       NOT NULL,
    name            VARCHAR(512)       NOT NULL,
    status          test_result_status NOT NULL,
    duration_ms     INTEGER,
    error_message   TEXT,
    stack_trace     TEXT,
    retry_count     INTEGER            NOT NULL DEFAULT 0,
    logs_url        TEXT,
    created_at      TIMESTAMPTZ        NOT NULL DEFAULT now(),

    CONSTRAINT fk_test_cases_group
        FOREIGN KEY (test_group_id)
        REFERENCES test_groups (id)
        ON DELETE CASCADE,
    CONSTRAINT fk_test_cases_run
        FOREIGN KEY (test_run_id)
        REFERENCES test_runs (id)
        ON DELETE CASCADE
);

CREATE INDEX idx_test_cases_group_id ON test_cases (test_group_id);
CREATE INDEX idx_test_cases_run_id   ON test_cases (test_run_id);
CREATE INDEX idx_test_cases_status   ON test_cases (test_run_id, status);
