package model

import (
	"time"

	"github.com/google/uuid"
)

type TestResultStatus string

const (
	TestResultStatusPass    TestResultStatus = "pass"
	TestResultStatusFail    TestResultStatus = "fail"
	TestResultStatusSkip    TestResultStatus = "skip"
	TestResultStatusError   TestResultStatus = "error"
	TestResultStatusRunning TestResultStatus = "running"
	TestResultStatusUnknown TestResultStatus = "unknown"
)

type TestResult struct {
	ID           uuid.UUID        `db:"id" json:"id"`
	TestRunID    uuid.UUID        `db:"test_run_id" json:"test_run_id"`
	TestName     string           `db:"test_name" json:"test_name"`
	Status       TestResultStatus `db:"status" json:"status"`
	DurationMs   *int             `db:"duration_ms" json:"duration_ms"`
	ErrorMessage *string          `db:"error_message" json:"error_message"`
	LogOutput    *string          `db:"log_output" json:"log_output"`
	CreatedAt    time.Time        `db:"created_at" json:"created_at"`
}
