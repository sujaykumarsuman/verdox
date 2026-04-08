CREATE TABLE test_results (
    id            UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    test_run_id   UUID               NOT NULL,
    test_name     VARCHAR(512)       NOT NULL,
    status        test_result_status NOT NULL,
    duration_ms   INTEGER,
    error_message TEXT,
    log_output    TEXT,
    created_at    TIMESTAMPTZ        NOT NULL DEFAULT now(),

    CONSTRAINT fk_test_results_run
        FOREIGN KEY (test_run_id)
        REFERENCES test_runs (id)
        ON DELETE CASCADE
);

CREATE INDEX idx_test_results_run_id ON test_results (test_run_id);
