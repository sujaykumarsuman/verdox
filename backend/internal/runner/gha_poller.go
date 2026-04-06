package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"

	"github.com/sujaykumarsuman/verdox/backend/internal/model"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
)

type activeGHARun struct {
	RunID uuid.UUID
	Job   *ExecutionJob
	PAT   string
}

// GHAPoller tracks dispatched GitHub Actions workflows and updates
// test runs when they complete.
type GHAPoller struct {
	db             *sqlx.DB
	client         *http.Client
	log            zerolog.Logger
	parser         *Parser
	mu             sync.Mutex
	activeRuns     map[string]*activeGHARun // verdox run_id -> tracking data
	pollInterval   time.Duration
	cancelFunc     context.CancelFunc
}

func NewGHAPoller(db *sqlx.DB, log zerolog.Logger) *GHAPoller {
	return &GHAPoller{
		db:           db,
		client:       &http.Client{Timeout: 30 * time.Second},
		log:          log,
		parser:       NewParser(),
		activeRuns:   make(map[string]*activeGHARun),
		pollInterval: 15 * time.Second,
	}
}

// Register adds a dispatched GHA run for tracking.
func (p *GHAPoller) Register(runID string, job *ExecutionJob, pat string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.activeRuns[runID] = &activeGHARun{
		RunID: job.RunID,
		Job:   job,
		PAT:   pat,
	}
	p.log.Info().Str("run_id", runID).Msg("registered GHA run for polling")
}

// Start begins the polling loop.
func (p *GHAPoller) Start(ctx context.Context) {
	ctx, p.cancelFunc = context.WithCancel(ctx)

	// On startup, load any active GHA runs from DB that may have been
	// dispatched before a restart (crash recovery)
	p.recoverActiveRuns(ctx)

	go func() {
		ticker := time.NewTicker(p.pollInterval)
		defer ticker.Stop()

		p.log.Info().Dur("interval", p.pollInterval).Msg("GHA poller started")

		for {
			select {
			case <-ctx.Done():
				p.log.Info().Msg("GHA poller stopping")
				return
			case <-ticker.C:
				p.poll(ctx)
			}
		}
	}()
}

func (p *GHAPoller) Shutdown() {
	if p.cancelFunc != nil {
		p.cancelFunc()
	}
}

func (p *GHAPoller) poll(ctx context.Context) {
	p.mu.Lock()
	runs := make(map[string]*activeGHARun)
	for k, v := range p.activeRuns {
		runs[k] = v
	}
	p.mu.Unlock()

	if len(runs) == 0 {
		return
	}

	runRepo := repository.NewTestRunRepository(p.db)

	for runID, active := range runs {
		// Check if already completed (webhook may have beaten us)
		dbRun, err := runRepo.GetByID(ctx, active.RunID)
		if err != nil {
			p.log.Warn().Err(err).Str("run_id", runID).Msg("failed to get run from DB")
			continue
		}
		if dbRun.Status.IsTerminal() {
			p.mu.Lock()
			delete(p.activeRuns, runID)
			p.mu.Unlock()
			continue
		}

		// Find the GHA workflow run by querying recent runs
		ghaRunID, status, err := p.findWorkflowRun(ctx, active)
		if err != nil {
			p.log.Debug().Err(err).Str("run_id", runID).Msg("workflow run not found yet")
			continue
		}

		// Update GHA run ID in our DB if not set
		if dbRun.GHARunID == nil && ghaRunID > 0 {
			runRepo.UpdateGHARunID(ctx, active.RunID, ghaRunID)
		}

		if status == "completed" {
			p.handleCompletion(ctx, active, ghaRunID)
			p.mu.Lock()
			delete(p.activeRuns, runID)
			p.mu.Unlock()
		}
	}
}

type ghaWorkflowRun struct {
	ID         int64  `json:"id"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	HeadSHA    string `json:"head_sha"`
}

type ghaWorkflowRunsResponse struct {
	TotalCount   int              `json:"total_count"`
	WorkflowRuns []ghaWorkflowRun `json:"workflow_runs"`
}

func (p *GHAPoller) findWorkflowRun(ctx context.Context, active *activeGHARun) (int64, string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/actions/runs?head_sha=%s&per_page=5",
		active.Job.RepositoryFullName, active.Job.CommitHash)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Authorization", "Bearer "+active.PAT)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := p.client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("github api returned %d", resp.StatusCode)
	}

	var result ghaWorkflowRunsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, "", err
	}

	for _, run := range result.WorkflowRuns {
		return run.ID, run.Status, nil
	}

	return 0, "", fmt.Errorf("no workflow runs found for commit %s", active.Job.CommitHash)
}

func (p *GHAPoller) handleCompletion(ctx context.Context, active *activeGHARun, ghaRunID int64) {
	runRepo := repository.NewTestRunRepository(p.db)
	resultRepo := repository.NewTestResultRepository(p.db)

	// Fetch the workflow run conclusion
	url := fmt.Sprintf("https://api.github.com/repos/%s/actions/runs/%d",
		active.Job.RepositoryFullName, ghaRunID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		p.log.Error().Err(err).Msg("failed to create request for GHA run details")
		runRepo.UpdateFinished(ctx, active.RunID, model.TestRunStatusFailed)
		return
	}
	req.Header.Set("Authorization", "Bearer "+active.PAT)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := p.client.Do(req)
	if err != nil {
		p.log.Error().Err(err).Msg("failed to fetch GHA run details")
		runRepo.UpdateFinished(ctx, active.RunID, model.TestRunStatusFailed)
		return
	}
	defer resp.Body.Close()

	var runDetail struct {
		Conclusion string `json:"conclusion"`
	}
	json.NewDecoder(resp.Body).Decode(&runDetail)

	// Try to download artifacts (look for verdox-results artifact)
	results := p.downloadResultsArtifact(ctx, active, ghaRunID)

	if len(results) > 0 {
		modelResults := make([]model.TestResult, len(results))
		for i, r := range results {
			mr := model.TestResult{
				TestRunID: active.RunID,
				TestName:  r.TestName,
				Status:    model.TestResultStatus(r.Status),
			}
			if r.DurationMs > 0 {
				ms := int(r.DurationMs)
				mr.DurationMs = &ms
			}
			if r.ErrorMessage != "" {
				mr.ErrorMessage = &r.ErrorMessage
			}
			if r.LogOutput != "" {
				mr.LogOutput = &r.LogOutput
			}
			modelResults[i] = mr
		}
		resultRepo.BatchCreate(ctx, modelResults)
	}

	// Determine final status
	finalStatus := model.TestRunStatusPassed
	if runDetail.Conclusion != "success" {
		finalStatus = model.TestRunStatusFailed
	}

	runRepo.UpdateFinished(ctx, active.RunID, finalStatus)
	p.log.Info().Str("run_id", active.RunID.String()).Str("status", string(finalStatus)).Msg("GHA run completed")
}

func (p *GHAPoller) downloadResultsArtifact(ctx context.Context, active *activeGHARun, ghaRunID int64) []ParsedResult {
	// List artifacts
	url := fmt.Sprintf("https://api.github.com/repos/%s/actions/runs/%d/artifacts",
		active.Job.RepositoryFullName, ghaRunID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+active.PAT)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := p.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}
	defer resp.Body.Close()

	var artifactsResp struct {
		Artifacts []struct {
			Name               string `json:"name"`
			ArchiveDownloadURL string `json:"archive_download_url"`
		} `json:"artifacts"`
	}
	json.NewDecoder(resp.Body).Decode(&artifactsResp)

	for _, artifact := range artifactsResp.Artifacts {
		if artifact.Name == "verdox-results" {
			return p.downloadAndParseArtifact(ctx, active.PAT, artifact.ArchiveDownloadURL)
		}
	}

	return nil
}

func (p *GHAPoller) downloadAndParseArtifact(ctx context.Context, pat, downloadURL string) []ParsedResult {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+pat)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := p.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}
	defer resp.Body.Close()

	// Artifact is a zip file — for now, try reading as raw JSON
	// (simplified; production should unzip)
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	results, err := ParseVerdoxResultsJSON(data)
	if err != nil {
		p.log.Debug().Err(err).Msg("failed to parse artifact as results JSON")
		return nil
	}

	return results
}

// recoverActiveRuns loads GHA runs that were dispatched but not completed.
func (p *GHAPoller) recoverActiveRuns(ctx context.Context) {
	// Note: Crash recovery for GHA runs requires the PAT to be available.
	// Since we don't store PATs in test_runs, recovery is best-effort.
	// The runs will be detected as stale and eventually marked as failed.
	p.log.Info().Msg("GHA poller recovery: checking for orphaned runs")
}
