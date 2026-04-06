package model

import (
	"time"

	"github.com/google/uuid"
)

type TestRunStatus string

const (
	TestRunStatusQueued    TestRunStatus = "queued"
	TestRunStatusRunning   TestRunStatus = "running"
	TestRunStatusPassed    TestRunStatus = "passed"
	TestRunStatusFailed    TestRunStatus = "failed"
	TestRunStatusCancelled TestRunStatus = "cancelled"
)

func (s TestRunStatus) IsTerminal() bool {
	return s == TestRunStatusPassed || s == TestRunStatusFailed || s == TestRunStatusCancelled
}

type TestRun struct {
	ID          uuid.UUID     `db:"id" json:"id"`
	TestSuiteID uuid.UUID     `db:"test_suite_id" json:"test_suite_id"`
	TriggeredBy *uuid.UUID    `db:"triggered_by" json:"triggered_by"`
	RunNumber   int           `db:"run_number" json:"run_number"`
	Branch      string        `db:"branch" json:"branch"`
	CommitHash  string        `db:"commit_hash" json:"commit_hash"`
	Status      TestRunStatus `db:"status" json:"status"`
	StartedAt   *time.Time    `db:"started_at" json:"started_at"`
	FinishedAt  *time.Time    `db:"finished_at" json:"finished_at"`
	CreatedAt   time.Time     `db:"created_at" json:"created_at"`
}
