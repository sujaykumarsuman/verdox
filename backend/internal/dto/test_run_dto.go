package dto

import (
	"fmt"
	"time"

	"github.com/sujaykumarsuman/verdox/backend/internal/model"
)

// --- Requests ---

type TriggerRunRequest struct {
	Branch     string `json:"branch" validate:"required,min=1,max=255"`
	CommitHash string `json:"commit_hash" validate:"required,len=40,hexadecimal"`
}

type RunAllRequest struct {
	Branch     string `json:"branch" validate:"required,min=1,max=255"`
	CommitHash string `json:"commit_hash" validate:"required,len=40,hexadecimal"`
}

// --- Responses ---

type TestRunResponse struct {
	ID                  string  `json:"id"`
	TestSuiteID         string  `json:"test_suite_id"`
	TriggeredBy         *string `json:"triggered_by"`
	TriggeredByUsername *string `json:"triggered_by_username,omitempty"`
	RunNumber           int     `json:"run_number"`
	Branch              string  `json:"branch"`
	CommitHash          string  `json:"commit_hash"`
	Status              string  `json:"status"`
	StartedAt           *string `json:"started_at"`
	FinishedAt          *string `json:"finished_at"`
	GHARunID            *int64  `json:"gha_run_id,omitempty"`
	GHARunURL           *string `json:"gha_run_url,omitempty"`
	CreatedAt           string  `json:"created_at"`
}

type TestRunDetailResponse struct {
	TestRunResponse
	SuiteName      string               `json:"suite_name"`
	SuiteType      string               `json:"suite_type"`
	ExecutionMode  string               `json:"execution_mode"`
	RepositoryID   string               `json:"repository_id"`
	RepositoryName string               `json:"repository_name"`
	LogOutput      *string              `json:"log_output,omitempty"`
	Summary        *RunSummary          `json:"summary"`
	SummaryV2      *RunSummaryV2        `json:"summary_v2,omitempty"`
	Results        []TestResultResponse `json:"results"`
	Groups         []TestGroupResponse  `json:"groups,omitempty"`
	ReportID       *string              `json:"report_id,omitempty"`
}

type RunSummary struct {
	Total      int   `json:"total"`
	Passed     int   `json:"passed"`
	Failed     int   `json:"failed"`
	Skipped    int   `json:"skipped"`
	Errors     int   `json:"errors"`
	DurationMs int64 `json:"duration_ms"`
}

type TestResultResponse struct {
	ID           string  `json:"id"`
	TestName     string  `json:"test_name"`
	Status       string  `json:"status"`
	DurationMs   *int    `json:"duration_ms"`
	ErrorMessage *string `json:"error_message"`
	CreatedAt    string  `json:"created_at"`
}

type TestRunListResponse struct {
	Runs []TestRunResponse `json:"runs"`
	Meta PaginationMeta    `json:"meta"`
}

type RunLogsResponse struct {
	RunID string         `json:"run_id"`
	Logs  []TestLogEntry `json:"logs"`
}

type TestLogEntry struct {
	TestName   string  `json:"test_name"`
	Status     string  `json:"status"`
	DurationMs *int    `json:"duration_ms"`
	LogOutput  *string `json:"log_output"`
}

type CancelRunResponse struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type RunAllResponse struct {
	Message string            `json:"message"`
	Runs    []TestRunResponse `json:"runs"`
}

// --- Constructors ---

func NewTestRunResponse(run *model.TestRun) TestRunResponse {
	resp := TestRunResponse{
		ID:          run.ID.String(),
		TestSuiteID: run.TestSuiteID.String(),
		RunNumber:   run.RunNumber,
		Branch:      run.Branch,
		CommitHash:  run.CommitHash,
		Status:      string(run.Status),
		GHARunID:    run.GHARunID,
		CreatedAt:   run.CreatedAt.Format(time.RFC3339),
	}
	if run.TriggeredBy != nil {
		s := run.TriggeredBy.String()
		resp.TriggeredBy = &s
	}
	if run.StartedAt != nil {
		s := run.StartedAt.Format(time.RFC3339)
		resp.StartedAt = &s
	}
	if run.FinishedAt != nil {
		s := run.FinishedAt.Format(time.RFC3339)
		resp.FinishedAt = &s
	}
	return resp
}

// NewTestRunResponseWithGHA adds the GHA run URL computed from repo full name.
func NewTestRunResponseWithGHA(run *model.TestRun, repoFullName string) TestRunResponse {
	resp := NewTestRunResponse(run)
	if run.GHARunID != nil && repoFullName != "" {
		url := fmt.Sprintf("https://github.com/%s/actions/runs/%d", repoFullName, *run.GHARunID)
		resp.GHARunURL = &url
	}
	return resp
}

func NewTestResultResponse(r *model.TestResult) TestResultResponse {
	return TestResultResponse{
		ID:           r.ID.String(),
		TestName:     r.TestName,
		Status:       string(r.Status),
		DurationMs:   r.DurationMs,
		ErrorMessage: r.ErrorMessage,
		CreatedAt:    r.CreatedAt.Format(time.RFC3339),
	}
}
