-- Level 2: Test groups (packages/features within a suite's run)
CREATE TABLE test_groups (
    id            UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    test_run_id   UUID               NOT NULL,
    group_id      VARCHAR(255)       NOT NULL,
    name          VARCHAR(512)       NOT NULL,
    package       VARCHAR(1024),
    status        test_result_status NOT NULL DEFAULT 'unknown',
    total         INTEGER            NOT NULL DEFAULT 0,
    passed        INTEGER            NOT NULL DEFAULT 0,
    failed        INTEGER            NOT NULL DEFAULT 0,
    skipped       INTEGER            NOT NULL DEFAULT 0,
    duration_ms   INTEGER,
    pass_rate     NUMERIC(5,2),
    sort_order    INTEGER            NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ        NOT NULL DEFAULT now(),

    CONSTRAINT fk_test_groups_run
        FOREIGN KEY (test_run_id)
        REFERENCES test_runs (id)
        ON DELETE CASCADE
);

CREATE INDEX idx_test_groups_run_id ON test_groups (test_run_id);
CREATE UNIQUE INDEX idx_test_groups_run_group ON test_groups (test_run_id, group_id);

-- Level 3: Individual test cases
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
CREATE INDEX idx_test_cases_run_id ON test_cases (test_run_id);
CREATE INDEX idx_test_cases_status ON test_cases (test_run_id, status);

-- Run-level summary JSONB + report grouping
ALTER TABLE test_runs ADD COLUMN summary JSONB;
ALTER TABLE test_runs ADD COLUMN report_id VARCHAR(255);
CREATE INDEX idx_test_runs_report_id ON test_runs (report_id);
