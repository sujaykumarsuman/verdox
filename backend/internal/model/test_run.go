package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SummaryJSON is a JSONB wrapper for run-level summary stats.
type SummaryJSON json.RawMessage

func (s *SummaryJSON) Scan(src interface{}) error {
	if src == nil {
		*s = nil
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("unsupported type for SummaryJSON: %T", src)
	}
	*s = SummaryJSON(data)
	return nil
}

func (s SummaryJSON) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return string(s), nil
}

func (s SummaryJSON) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("null"), nil
	}
	return []byte(s), nil
}

func (s *SummaryJSON) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*s = nil
		return nil
	}
	*s = SummaryJSON(data)
	return nil
}

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
	GHARunID    *int64        `db:"gha_run_id" json:"gha_run_id,omitempty"`
	LogOutput   *string       `db:"log_output" json:"log_output,omitempty"`
	Summary     *SummaryJSON  `db:"summary" json:"summary,omitempty"`
	ReportID    *string       `db:"report_id" json:"report_id,omitempty"`
	CreatedAt   time.Time     `db:"created_at" json:"created_at"`
}
