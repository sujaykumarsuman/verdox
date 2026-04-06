package runner

import (
	"context"

	"github.com/google/uuid"
)

// ExecutionMode constants.
const (
	ModeContainer = "container"
	ModeGHA       = "gha"
)

// ExecutionJob carries all data needed by an executor to run a test.
type ExecutionJob struct {
	RunID              uuid.UUID
	SuiteID            uuid.UUID
	RepoID             uuid.UUID
	RepositoryFullName string
	LocalPath          string
	DefaultBranch      string
	Branch             string
	CommitHash         string
	SuiteType          string
	ExecutionMode      string
	DockerImage        string
	TestCommand        string
	GHAWorkflowID     string
	ConfigPath         string
	TimeoutSeconds     int
	EnvVars            map[string]string
}

// ExecutionResult carries the outcome of a test execution.
type ExecutionResult struct {
	ExitCode int64
	Output   string         // raw stdout/stderr
	Results  []ParsedResult // parsed test results
	Status   string         // "passed", "failed", "error", "dispatched" (for async GHA)
	ErrorMsg string
}

// Executor is the plugin interface for running tests.
type Executor interface {
	// Execute runs the test job and returns results. For async executors (GHA),
	// it returns immediately with Status="dispatched".
	Execute(ctx context.Context, job *ExecutionJob) (*ExecutionResult, error)

	// Cancel stops a running test.
	Cancel(ctx context.Context, runID string) error

	// Supports returns true if this executor handles the given mode.
	Supports(mode string) bool
}
