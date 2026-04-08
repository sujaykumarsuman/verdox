CREATE TABLE test_runs (
    id            UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    test_suite_id UUID            NOT NULL,
    triggered_by  UUID,
    run_number    INTEGER         NOT NULL DEFAULT 1,
    branch        VARCHAR(255)    NOT NULL,
    commit_hash   CHAR(40)        NOT NULL,
    status        test_run_status NOT NULL DEFAULT 'queued',
    started_at    TIMESTAMPTZ,
    finished_at   TIMESTAMPTZ,
    gha_run_id    BIGINT,
    log_output    TEXT,
    summary       JSONB,
    report_id     VARCHAR(255),
    created_at    TIMESTAMPTZ     NOT NULL DEFAULT now(),

    CONSTRAINT fk_test_runs_suite
        FOREIGN KEY (test_suite_id)
        REFERENCES test_suites (id)
        ON DELETE CASCADE,

    CONSTRAINT fk_test_runs_triggered_by
        FOREIGN KEY (triggered_by)
        REFERENCES users (id)
        ON DELETE SET NULL
);

CREATE INDEX idx_test_runs_suite_id            ON test_runs (test_suite_id);
CREATE INDEX idx_test_runs_suite_status        ON test_runs (test_suite_id, status);
CREATE INDEX idx_test_runs_triggered_by        ON test_runs (triggered_by);
CREATE INDEX idx_test_runs_suite_branch_commit ON test_runs (test_suite_id, branch, commit_hash);
CREATE INDEX idx_test_runs_gha_run_id          ON test_runs (gha_run_id) WHERE gha_run_id IS NOT NULL;
CREATE INDEX idx_test_runs_report_id           ON test_runs (report_id);
