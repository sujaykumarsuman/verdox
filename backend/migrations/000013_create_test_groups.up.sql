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

CREATE INDEX        idx_test_groups_run_id    ON test_groups (test_run_id);
CREATE UNIQUE INDEX idx_test_groups_run_group ON test_groups (test_run_id, group_id);
