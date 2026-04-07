package model

import (
	"time"

	"github.com/google/uuid"
)

type TestGroup struct {
	ID        uuid.UUID        `db:"id" json:"id"`
	TestRunID uuid.UUID        `db:"test_run_id" json:"test_run_id"`
	GroupID   string           `db:"group_id" json:"group_id"`
	Name      string           `db:"name" json:"name"`
	Package   *string          `db:"package" json:"package"`
	Status    TestResultStatus `db:"status" json:"status"`
	Total     int              `db:"total" json:"total"`
	Passed    int              `db:"passed" json:"passed"`
	Failed    int              `db:"failed" json:"failed"`
	Skipped   int              `db:"skipped" json:"skipped"`
	DurationMs *int            `db:"duration_ms" json:"duration_ms"`
	PassRate  *float64         `db:"pass_rate" json:"pass_rate"`
	SortOrder int              `db:"sort_order" json:"sort_order"`
	CreatedAt time.Time        `db:"created_at" json:"created_at"`
}
