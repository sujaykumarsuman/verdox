package runner

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/sujaykumarsuman/verdox/backend/internal/config"
	"github.com/sujaykumarsuman/verdox/backend/internal/service"
)

// ForkGHAExecutor runs tests on a Verdox-managed fork via GitHub Actions.
// It uses the service account PAT instead of team-level PATs.
type ForkGHAExecutor struct {
	forkService    *service.ForkService
	cfg            *config.Config
	log            zerolog.Logger
	registerPoller func(runID string, job *ExecutionJob, pat string)
}

func NewForkGHAExecutor(
	forkService *service.ForkService,
	cfg *config.Config,
	log zerolog.Logger,
	registerPoller func(string, *ExecutionJob, string),
) *ForkGHAExecutor {
	return &ForkGHAExecutor{
		forkService:    forkService,
		cfg:            cfg,
		log:            log,
		registerPoller: registerPoller,
	}
}

func (e *ForkGHAExecutor) Supports(mode string) bool {
	return mode == ModeForkGHA
}

func (e *ForkGHAExecutor) Execute(ctx context.Context, job *ExecutionJob) (*ExecutionResult, error) {
	if !e.forkService.IsConfigured() {
		return &ExecutionResult{Status: "failed", ErrorMsg: "Service account PAT not configured"}, nil
	}

	forkName := job.EnvVars["_fork_full_name"]
	if forkName == "" {
		return &ExecutionResult{Status: "failed", ErrorMsg: "Fork not set up for this repository"}, nil
	}

	// Ensure Verdox workflow files exist on the target branch.
	// PushSuiteWorkflows creates a single Verdox commit on top of the
	// upstream branch HEAD, adding all suite workflow files. This is needed
	// because workflow_dispatch requires the workflow file to exist on the
	// dispatched ref.
	branchHeadSHA, err := e.forkService.EnsureBranchWorkflows(ctx, job.RepoID, job.Branch)
	if err != nil {
		return &ExecutionResult{
			Status:   "failed",
			ErrorMsg: "Failed to push workflows to branch: " + err.Error(),
		}, nil
	}

	// Dispatch this suite's specific workflow on the target branch
	workflowID := fmt.Sprintf("verdox-%s.yml", job.SuiteID.String())

	inputs := map[string]string{
		"verdox_run_id": job.RunID.String(),
		"branch":        job.Branch,
		"commit_hash":   job.CommitHash,
	}
	if job.TestCommand != "" {
		inputs["test_command"] = job.TestCommand
	}
	if e.cfg.WebhookBaseURL != "" {
		inputs["callback_url"] = fmt.Sprintf("%s/v1/webhooks/gha/%s", e.cfg.WebhookBaseURL, job.RunID.String())
	}

	if err := e.forkService.DispatchWorkflow(ctx, forkName, workflowID, job.Branch, inputs); err != nil {
		return &ExecutionResult{
			Status:   "failed",
			ErrorMsg: "Fork GHA dispatch failed: " + err.Error(),
		}, nil
	}

	e.log.Info().
		Str("run_id", job.RunID.String()).
		Str("fork", forkName).
		Str("branch", job.Branch).
		Str("verdox_commit", branchHeadSHA[:7]).
		Msg("fork GHA workflow dispatched")

	// Register with poller — use the Verdox commit SHA on the target branch
	// (GHA sets head_sha to the tip of the dispatched ref)
	if e.registerPoller != nil {
		pollerJob := *job
		pollerJob.RepositoryFullName = forkName
		pollerJob.CommitHash = branchHeadSHA
		e.registerPoller(job.RunID.String(), &pollerJob, e.cfg.ServiceAccountPAT)
	}

	return &ExecutionResult{Status: "dispatched"}, nil
}

// Rerun triggers a re-execution of a previously completed GHA workflow run.
func (e *ForkGHAExecutor) Rerun(ctx context.Context, job *ExecutionJob, originalGHARunID int64) (*ExecutionResult, error) {
	if !e.forkService.IsConfigured() {
		return &ExecutionResult{Status: "failed", ErrorMsg: "Service account PAT not configured"}, nil
	}

	forkName := job.EnvVars["_fork_full_name"]
	if forkName == "" {
		return &ExecutionResult{Status: "failed", ErrorMsg: "Fork not set up for this repository"}, nil
	}

	// Call GitHub's rerun API
	if err := e.forkService.RerunWorkflow(ctx, forkName, originalGHARunID); err != nil {
		return &ExecutionResult{
			Status:   "failed",
			ErrorMsg: "GHA rerun failed: " + err.Error(),
		}, nil
	}

	e.log.Info().
		Str("run_id", job.RunID.String()).
		Str("fork", forkName).
		Int64("original_gha_run_id", originalGHARunID).
		Msg("fork GHA workflow rerun triggered")

	// Register with poller — use the fork name and commit hash
	// The poller will find the new GHA run by head_sha
	if e.registerPoller != nil {
		pollerJob := *job
		pollerJob.RepositoryFullName = forkName
		e.registerPoller(job.RunID.String(), &pollerJob, e.cfg.ServiceAccountPAT)
	}

	return &ExecutionResult{Status: "dispatched"}, nil
}

func (e *ForkGHAExecutor) Cancel(ctx context.Context, runID string) error {
	e.log.Warn().Str("run_id", runID).Msg("fork GHA run cancellation not yet implemented")
	return nil
}
