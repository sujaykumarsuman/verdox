package dto

import (
	"time"

	"github.com/sujaykumarsuman/verdox/backend/internal/model"
)

// --- Ingest Payload DTOs (matches schema.json) ---

type HierarchicalPayload struct {
	Repo      string          `json:"repo" validate:"required"`
	RunID     string          `json:"run_id"`
	Branch    string          `json:"branch"`
	CommitSHA string          `json:"commit_sha"`
	Timestamp string          `json:"timestamp"`
	Summary   *PayloadSummary `json:"summary"`
	Jobs      []PayloadJob    `json:"jobs" validate:"required,min=1"`
}

type PayloadSummary struct {
	TotalJobs       int     `json:"total_jobs"`
	TotalTests      int     `json:"total_tests"`
	TotalCases      int     `json:"total_cases"`
	Passed          int     `json:"passed"`
	Failed          int     `json:"failed"`
	Skipped         int     `json:"skipped"`
	DurationSeconds float64 `json:"duration_seconds"`
	PassRate        float64 `json:"pass_rate"`
}

type PayloadJob struct {
	JobID  string        `json:"job_id" validate:"required"`
	Name   string        `json:"name" validate:"required"`
	Type   string        `json:"type" validate:"required"`
	Status string        `json:"status"`
	Stats  *PayloadStats `json:"stats"`
	Tests  []PayloadTest `json:"tests" validate:"required"`
}

type PayloadTest struct {
	TestID  string        `json:"test_id" validate:"required"`
	Name    string        `json:"name" validate:"required"`
	Package string        `json:"package"`
	Status  string        `json:"status"`
	Stats   *PayloadStats `json:"stats"`
	Cases   []PayloadCase `json:"cases" validate:"required"`
}

type PayloadCase struct {
	CaseID          string  `json:"case_id" validate:"required"`
	Name            string  `json:"name" validate:"required"`
	Status          string  `json:"status" validate:"required"`
	DurationSeconds float64 `json:"duration_seconds"`
	ErrorMessage    string  `json:"error_message"`
	StackTrace      string  `json:"stack_trace"`
	RetryCount      int     `json:"retry_count"`
	LogsURL         string  `json:"logs_url"`
}

type PayloadStats struct {
	Total           int     `json:"total"`
	Passed          int     `json:"passed"`
	Failed          int     `json:"failed"`
	Skipped         int     `json:"skipped"`
	DurationSeconds float64 `json:"duration_seconds"`
	PassRate        float64 `json:"pass_rate"`
}

// --- API Response DTOs ---

type TestGroupResponse struct {
	ID         string  `json:"id"`
	GroupID    string  `json:"group_id"`
	Name       string  `json:"name"`
	Package    *string `json:"package"`
	Status     string  `json:"status"`
	Total      int     `json:"total"`
	Passed     int     `json:"passed"`
	Failed     int     `json:"failed"`
	Skipped    int     `json:"skipped"`
	DurationMs *int    `json:"duration_ms"`
	PassRate   *float64 `json:"pass_rate"`
	CreatedAt  string  `json:"created_at"`
}

type TestCaseResponse struct {
	ID           string  `json:"id"`
	CaseID       string  `json:"case_id"`
	Name         string  `json:"name"`
	Status       string  `json:"status"`
	DurationMs   *int    `json:"duration_ms"`
	ErrorMessage *string `json:"error_message"`
	StackTrace   *string `json:"stack_trace"`
	RetryCount   int     `json:"retry_count"`
	LogsURL      *string `json:"logs_url"`
	CreatedAt    string  `json:"created_at"`
}

type RunHierarchyResponse struct {
	TestRunResponse
	SuiteName      string              `json:"suite_name"`
	SuiteType      string              `json:"suite_type"`
	RepositoryID   string              `json:"repository_id"`
	RepositoryName string              `json:"repository_name"`
	Summary        *RunSummaryV2       `json:"summary"`
	Groups         []TestGroupResponse `json:"groups"`
}

type RunSummaryV2 struct {
	TotalJobs   int     `json:"total_jobs"`
	TotalCases  int     `json:"total_cases"`
	Passed      int     `json:"passed"`
	Failed      int     `json:"failed"`
	Skipped     int     `json:"skipped"`
	DurationMs  int64   `json:"duration_ms"`
	PassRate    float64 `json:"pass_rate"`
}

type GroupCasesResponse struct {
	Cases []TestCaseResponse `json:"cases"`
	Meta  PaginationMeta     `json:"meta"`
}

type ReportResponse struct {
	ReportID string            `json:"report_id"`
	Runs     []TestRunResponse `json:"runs"`
}

type IngestResponse struct {
	ReportID string   `json:"report_id"`
	RunIDs   []string `json:"run_ids"`
	Message  string   `json:"message"`
}

// --- Constructors ---

func NewTestGroupResponse(g *model.TestGroup) TestGroupResponse {
	return TestGroupResponse{
		ID:         g.ID.String(),
		GroupID:    g.GroupID,
		Name:       g.Name,
		Package:    g.Package,
		Status:     string(g.Status),
		Total:      g.Total,
		Passed:     g.Passed,
		Failed:     g.Failed,
		Skipped:    g.Skipped,
		DurationMs: g.DurationMs,
		PassRate:   g.PassRate,
		CreatedAt:  g.CreatedAt.Format(time.RFC3339),
	}
}

func NewTestCaseResponse(c *model.TestCase) TestCaseResponse {
	return TestCaseResponse{
		ID:           c.ID.String(),
		CaseID:       c.CaseID,
		Name:         c.Name,
		Status:       string(c.Status),
		DurationMs:   c.DurationMs,
		ErrorMessage: c.ErrorMessage,
		StackTrace:   c.StackTrace,
		RetryCount:   c.RetryCount,
		LogsURL:      c.LogsURL,
		CreatedAt:    c.CreatedAt.Format(time.RFC3339),
	}
}
