package runner

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"

	"github.com/sujaykumarsuman/verdox/backend/internal/dto"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
	"github.com/sujaykumarsuman/verdox/backend/internal/service"
)

// artifactData holds the extracted contents from a verdox-results artifact.
type artifactData struct {
	Results          []ParsedResult
	TestOutputLog    string // raw content of test-output.log
	HierarchicalJSON []byte // raw JSON when payload has "suites" key (schema.json format)
}

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
	serviceAcctPAT string
}

func NewGHAPoller(db *sqlx.DB, serviceAcctPAT string, log zerolog.Logger) *GHAPoller {
	return &GHAPoller{
		db:             db,
		client:         &http.Client{Timeout: 30 * time.Second},
		log:            log,
		parser:         NewParser(),
		activeRuns:     make(map[string]*activeGHARun),
		pollInterval:   15 * time.Second,
		serviceAcctPAT: serviceAcctPAT,
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
	// Match by head_sha — for fork_gha this is the Verdox workflow commit SHA (fork_head_sha),
	// which is the tip of the fork's branch and what GitHub uses for workflow_dispatch runs.
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

	// Return the most recent matching run
	for _, run := range result.WorkflowRuns {
		return run.ID, run.Status, nil
	}

	return 0, "", fmt.Errorf("no workflow runs found for head_sha %s on %s", active.Job.CommitHash, active.Job.RepositoryFullName)
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

	finalStatus := model.TestRunStatusPassed
	if runDetail.Conclusion != "success" {
		finalStatus = model.TestRunStatusFailed
	}

	// Try to download verdox-results artifact (contains verdox-results.json + test-output.log)
	artifact := p.downloadResultsArtifact(ctx, active, ghaRunID)

	// If no test output from artifact, download GHA workflow run logs as fallback
	logOutput := artifact.TestOutputLog
	if logOutput == "" {
		logOutput = p.downloadGHARunLogs(ctx, active.Job.RepositoryFullName, ghaRunID, active.PAT)
	}

	if len(artifact.HierarchicalJSON) > 0 {
		// Hierarchical payload (schema.json format with suites[])
		// Use IngestionService to process into test_groups + test_cases
		if err := p.ingestHierarchicalArtifact(ctx, active.RunID, artifact.HierarchicalJSON); err != nil {
			p.log.Error().Err(err).Msg("failed to ingest hierarchical results, falling back")
		} else {
			// Hierarchical ingestion handles its own status update — skip the one at the end
			if logOutput != "" {
				runRepo.UpdateLogOutput(ctx, active.RunID, logOutput)
			}
			p.log.Info().
				Str("run_id", active.RunID.String()).
				Msg("GHA run completed with hierarchical results")
			return
		}
	}

	if len(artifact.Results) > 0 {
		// We have structured per-test results from the artifact
		modelResults := make([]model.TestResult, len(artifact.Results))
		for i, r := range artifact.Results {
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
		if err := resultRepo.BatchCreate(ctx, modelResults); err != nil {
			p.log.Error().Err(err).Msg("failed to store test results")
		}
	} else if logOutput != "" {
		// No structured results but we have log output — create a summary result entry
		// so the logs endpoint has something to return
		summaryStatus := model.TestResultStatusPass
		if finalStatus == model.TestRunStatusFailed {
			summaryStatus = model.TestResultStatusFail
		}
		summaryResult := model.TestResult{
			TestRunID: active.RunID,
			TestName:  "Test Output",
			Status:    summaryStatus,
			LogOutput: &logOutput,
		}
		// Also set error_message on failures so the UI can show the expandable error
		if finalStatus == model.TestRunStatusFailed {
			summaryResult.ErrorMessage = &logOutput
		}
		if err := resultRepo.BatchCreate(ctx, []model.TestResult{summaryResult}); err != nil {
			p.log.Error().Err(err).Msg("failed to store summary test result")
		}
	}

	// Also store log output on the test_run record for direct access
	if logOutput != "" {
		if err := runRepo.UpdateLogOutput(ctx, active.RunID, logOutput); err != nil {
			p.log.Warn().Err(err).Msg("failed to store run log output")
		}
	}

	runRepo.UpdateFinished(ctx, active.RunID, finalStatus)
	p.log.Info().
		Str("run_id", active.RunID.String()).
		Str("status", string(finalStatus)).
		Int("results", len(artifact.Results)).
		Bool("has_logs", logOutput != "").
		Msg("GHA run completed")
}

func (p *GHAPoller) downloadResultsArtifact(ctx context.Context, active *activeGHARun, ghaRunID int64) artifactData {
	// List artifacts for this workflow run
	url := fmt.Sprintf("https://api.github.com/repos/%s/actions/runs/%d/artifacts",
		active.Job.RepositoryFullName, ghaRunID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return artifactData{}
	}
	req.Header.Set("Authorization", "Bearer "+active.PAT)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := p.client.Do(req)
	if err != nil {
		p.log.Debug().Err(err).Msg("failed to list artifacts")
		return artifactData{}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		p.log.Debug().Int("status", resp.StatusCode).Msg("artifacts list returned non-200")
		return artifactData{}
	}

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

	p.log.Debug().Int64("gha_run_id", ghaRunID).Msg("no verdox-results artifact found")
	return artifactData{}
}

func (p *GHAPoller) downloadAndParseArtifact(ctx context.Context, pat, downloadURL string) artifactData {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return artifactData{}
	}
	req.Header.Set("Authorization", "Bearer "+pat)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := p.client.Do(req)
	if err != nil {
		p.log.Debug().Err(err).Msg("failed to download artifact")
		return artifactData{}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		p.log.Debug().Int("status", resp.StatusCode).Msg("artifact download returned non-200")
		return artifactData{}
	}

	// GitHub artifact downloads are always ZIP files
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		p.log.Debug().Err(err).Msg("failed to read artifact response body")
		return artifactData{}
	}

	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		p.log.Warn().Err(err).Int("bytes", len(data)).Msg("failed to open artifact as ZIP")
		return artifactData{}
	}

	var resultsJSON []byte
	var testOutputLog string

	for _, f := range zipReader.File {
		rc, err := f.Open()
		if err != nil {
			p.log.Debug().Err(err).Str("file", f.Name).Msg("failed to open file in ZIP")
			continue
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}

		switch f.Name {
		case "verdox-results.json":
			resultsJSON = content
		case "test-output.log":
			testOutputLog = string(content)
		}
	}

	p.log.Debug().
		Bool("has_results_json", len(resultsJSON) > 0).
		Bool("has_test_output", testOutputLog != "").
		Int("log_bytes", len(testOutputLog)).
		Msg("extracted artifact contents")

	// Try parsing as full VerdoxResultsFile (with per-test results array)
	if len(resultsJSON) > 0 {
		if results, err := ParseVerdoxResultsJSON(resultsJSON); err == nil && len(results) > 0 {
			// If individual results don't have logs, attach the test-output.log
			// to the first result so there's at least something visible
			if testOutputLog != "" {
				hasAnyLogs := false
				for _, r := range results {
					if r.LogOutput != "" {
						hasAnyLogs = true
						break
					}
				}
				if !hasAnyLogs {
					results[0].LogOutput = testOutputLog
				}
			}
			return artifactData{Results: results, TestOutputLog: testOutputLog}
		}
	}

	// Try hierarchical format (schema.json with "jobs" key)
	if len(resultsJSON) > 0 {
		var probe struct {
			Jobs json.RawMessage `json:"jobs"`
		}
		if err := json.Unmarshal(resultsJSON, &probe); err == nil && len(probe.Jobs) > 0 && string(probe.Jobs) != "null" {
			p.log.Info().Int("bytes", len(resultsJSON)).Msg("detected hierarchical results payload")
			return artifactData{HierarchicalJSON: resultsJSON, TestOutputLog: testOutputLog}
		}
	}

	// Fallback: minimal verdox-results.json format
	// The workflow generates: { verdox_run_id, status, exit_code }
	// which doesn't have the full results array — create a single summary entry
	if len(resultsJSON) > 0 {
		var minimal struct {
			VerdoxRunID string `json:"verdox_run_id"`
			Status      string `json:"status"`   // GHA step outcome: "success", "failure"
			ExitCode    int    `json:"exit_code"`
		}
		if err := json.Unmarshal(resultsJSON, &minimal); err == nil && minimal.VerdoxRunID != "" {
			status := "pass"
			if minimal.Status != "success" || minimal.ExitCode != 0 {
				status = "fail"
			}
			result := ParsedResult{
				TestName:  "Test Output",
				Status:    status,
				LogOutput: testOutputLog,
			}
			return artifactData{Results: []ParsedResult{result}, TestOutputLog: testOutputLog}
		}
	}

	// No parseable results JSON, but we may still have the test output log
	return artifactData{TestOutputLog: testOutputLog}
}

// downloadGHARunLogs fetches the workflow run logs directly from the GitHub
// Actions API. This serves as a fallback when no artifact is available.
func (p *GHAPoller) downloadGHARunLogs(ctx context.Context, repoFullName string, ghaRunID int64, pat string) string {
	url := fmt.Sprintf("https://api.github.com/repos/%s/actions/runs/%d/logs",
		repoFullName, ghaRunID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+pat)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := p.client.Do(req)
	if err != nil {
		p.log.Debug().Err(err).Msg("failed to download GHA run logs")
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		p.log.Debug().Int("status", resp.StatusCode).Msg("GHA run logs download returned non-200")
		return ""
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	// GHA run logs are returned as a ZIP containing one .txt file per job step
	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		p.log.Debug().Err(err).Msg("failed to open GHA run logs ZIP")
		return ""
	}

	var allLogs strings.Builder
	for _, f := range zipReader.File {
		rc, err := f.Open()
		if err != nil {
			continue
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}
		allLogs.WriteString(fmt.Sprintf("=== %s ===\n", f.Name))
		allLogs.Write(content)
		allLogs.WriteString("\n")
	}

	result := allLogs.String()
	p.log.Debug().Int("log_bytes", len(result)).Msg("downloaded GHA run logs")
	return result
}

// ingestHierarchicalArtifact processes a schema.json hierarchical payload
// from the downloaded artifact. It uses the IngestionService to create
// test_groups and test_cases, then updates the run summary.
func (p *GHAPoller) ingestHierarchicalArtifact(ctx context.Context, runID uuid.UUID, payload []byte) error {
	var hierarchical dto.HierarchicalPayload
	if err := json.Unmarshal(payload, &hierarchical); err != nil {
		return fmt.Errorf("unmarshal hierarchical payload: %w", err)
	}

	suiteRepo := repository.NewTestSuiteRepository(p.db)
	runRepo := repository.NewTestRunRepository(p.db)
	groupRepo := repository.NewTestGroupRepository(p.db)
	caseRepo := repository.NewTestCaseRepository(p.db)
	repoRepo := repository.NewRepositoryRepository(p.db)

	ingestionSvc := service.NewIngestionService(suiteRepo, runRepo, groupRepo, caseRepo, repoRepo, p.log)
	return ingestionSvc.IngestForRun(ctx, runID, &hierarchical)
}

// recoverActiveRuns loads GHA runs that were dispatched but not completed.
// This handles backend restarts and Redis flushes — any run with status
// 'running' and a gha_run_id is re-registered for polling.
func (p *GHAPoller) recoverActiveRuns(ctx context.Context) {
	p.log.Info().Msg("GHA poller recovery: checking for orphaned runs")

	var orphans []struct {
		RunID          uuid.UUID `db:"id"`
		GHARunID       *int64    `db:"gha_run_id"`
		CommitHash     string    `db:"commit_hash"`
		Branch         string    `db:"branch"`
		SuiteID        uuid.UUID `db:"test_suite_id"`
		ForkFullName   *string   `db:"fork_full_name"`
		RepoID         uuid.UUID `db:"repo_id"`
		RepoFullName   string    `db:"repo_full_name"`
		DefaultBranch  string    `db:"default_branch"`
	}

	query := `
		SELECT tr.id, tr.gha_run_id, tr.commit_hash, tr.branch, tr.test_suite_id,
		       r.fork_full_name, r.id AS repo_id, r.github_full_name AS repo_full_name,
		       r.default_branch
		FROM test_runs tr
		JOIN test_suites ts ON ts.id = tr.test_suite_id
		JOIN repositories r ON r.id = ts.repository_id
		WHERE tr.status = 'running' AND tr.gha_run_id IS NOT NULL
	`

	if err := p.db.SelectContext(ctx, &orphans, query); err != nil {
		p.log.Error().Err(err).Msg("GHA poller recovery: failed to query orphaned runs")
		return
	}

	if len(orphans) == 0 {
		p.log.Info().Msg("GHA poller recovery: no orphaned runs found")
		return
	}

	p.mu.Lock()
	for _, o := range orphans {
		forkName := o.RepoFullName
		if o.ForkFullName != nil && *o.ForkFullName != "" {
			forkName = *o.ForkFullName
		}

		runIDStr := o.RunID.String()
		p.activeRuns[runIDStr] = &activeGHARun{
			RunID: o.RunID,
			Job: &ExecutionJob{
				RunID:              o.RunID,
				SuiteID:            o.SuiteID,
				RepoID:             o.RepoID,
				RepositoryFullName: forkName,
				DefaultBranch:      o.DefaultBranch,
				Branch:             o.Branch,
				CommitHash:         o.CommitHash,
			},
			PAT: p.serviceAcctPAT,
		}
		p.log.Info().Str("run_id", runIDStr).Int64("gha_run_id", *o.GHARunID).Msg("GHA poller recovery: re-registered orphaned run")
	}
	p.mu.Unlock()
}
