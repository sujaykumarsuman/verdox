package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/sujaykumarsuman/verdox/backend/internal/config"
)

// GHAExecutor triggers GitHub Actions workflows and returns immediately.
// Completion tracking is handled by GHAPoller.
type GHAExecutor struct {
	client         *http.Client
	cfg            *config.Config
	log            zerolog.Logger
	registerPoller func(runID string, job *ExecutionJob, pat string) // callback to register with poller
}

func NewGHAExecutor(cfg *config.Config, log zerolog.Logger, registerPoller func(string, *ExecutionJob, string)) *GHAExecutor {
	return &GHAExecutor{
		client:         &http.Client{Timeout: 30 * time.Second},
		cfg:            cfg,
		log:            log,
		registerPoller: registerPoller,
	}
}

func (e *GHAExecutor) Supports(mode string) bool {
	return mode == ModeGHA
}

// Execute dispatches a GitHub Actions workflow and returns immediately with Status="dispatched".
// The PAT must be passed in via job.EnvVars["_verdox_pat"] (injected by the worker pool before calling).
func (e *GHAExecutor) Execute(ctx context.Context, job *ExecutionJob) (*ExecutionResult, error) {
	pat := job.EnvVars["_verdox_pat"]
	if pat == "" {
		return &ExecutionResult{Status: "failed", ErrorMsg: "GitHub PAT not available for GHA dispatch"}, nil
	}

	// Build workflow dispatch payload
	inputs := map[string]string{
		"verdox_run_id": job.RunID.String(),
		"branch":        job.Branch,
		"commit_hash":   job.CommitHash,
	}
	if e.cfg.WebhookBaseURL != "" {
		inputs["callback_url"] = fmt.Sprintf("%s/v1/webhooks/gha/%s", e.cfg.WebhookBaseURL, job.RunID.String())
	}

	payload := map[string]interface{}{
		"ref":    job.Branch,
		"inputs": inputs,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return &ExecutionResult{Status: "failed", ErrorMsg: "Marshal dispatch payload: " + err.Error()}, nil
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/actions/workflows/%s/dispatches",
		job.RepositoryFullName, job.GHAWorkflowID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return &ExecutionResult{Status: "failed", ErrorMsg: "Create request: " + err.Error()}, nil
	}
	req.Header.Set("Authorization", "Bearer "+pat)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return &ExecutionResult{Status: "failed", ErrorMsg: "Dispatch workflow: " + err.Error()}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return &ExecutionResult{
			Status:   "failed",
			ErrorMsg: fmt.Sprintf("GitHub API returned %d: %s", resp.StatusCode, string(respBody)),
		}, nil
	}

	e.log.Info().
		Str("run_id", job.RunID.String()).
		Str("workflow", job.GHAWorkflowID).
		Str("repo", job.RepositoryFullName).
		Msg("GHA workflow dispatched successfully")

	// Register with poller for completion tracking
	if e.registerPoller != nil {
		e.registerPoller(job.RunID.String(), job, pat)
	}

	return &ExecutionResult{Status: "dispatched"}, nil
}

func (e *GHAExecutor) Cancel(ctx context.Context, runID string) error {
	// GHA cancellation would require finding the workflow run ID and calling
	// POST /repos/{owner}/{repo}/actions/runs/{run_id}/cancel
	// For now, the poller will detect completion regardless.
	e.log.Warn().Str("run_id", runID).Msg("GHA run cancellation not yet implemented")
	return nil
}
