package model

import (
	"time"

	"github.com/google/uuid"
)

type TestCase struct {
	ID           uuid.UUID        `db:"id" json:"id"`
	TestGroupID  uuid.UUID        `db:"test_group_id" json:"test_group_id"`
	TestRunID    uuid.UUID        `db:"test_run_id" json:"test_run_id"`
	CaseID       string           `db:"case_id" json:"case_id"`
	Name         string           `db:"name" json:"name"`
	Status       TestResultStatus `db:"status" json:"status"`
	DurationMs   *int             `db:"duration_ms" json:"duration_ms"`
	ErrorMessage *string          `db:"error_message" json:"error_message"`
	StackTrace   *string          `db:"stack_trace" json:"stack_trace"`
	RetryCount   int              `db:"retry_count" json:"retry_count"`
	LogsURL      *string          `db:"logs_url" json:"logs_url"`
	CreatedAt    time.Time        `db:"created_at" json:"created_at"`
}
