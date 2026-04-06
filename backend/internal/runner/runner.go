package runner

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/sujaykumarsuman/verdox/backend/internal/config"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
	"github.com/sujaykumarsuman/verdox/backend/internal/queue"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
	"github.com/sujaykumarsuman/verdox/backend/pkg/encryption"
)

type WorkerPool struct {
	size       int
	queue      *queue.RedisQueue
	executors  map[string]Executor
	container  *ContainerExecutor
	db         *sqlx.DB
	rdb        *redis.Client
	cfg        *config.Config
	log        zerolog.Logger
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup
}

func NewWorkerPool(
	cfg *config.Config,
	q *queue.RedisQueue,
	db *sqlx.DB,
	rdb *redis.Client,
	log zerolog.Logger,
	containerExec *ContainerExecutor,
	ghaExec *GHAExecutor,
) *WorkerPool {
	executors := map[string]Executor{
		ModeContainer: containerExec,
	}
	if ghaExec != nil {
		executors[ModeGHA] = ghaExec
	}

	return &WorkerPool{
		size:      cfg.RunnerMaxConcurrent,
		queue:     q,
		executors: executors,
		container: containerExec,
		db:        db,
		rdb:       rdb,
		cfg:       cfg,
		log:       log,
	}
}

func (wp *WorkerPool) Start(ctx context.Context) {
	ctx, wp.cancelFunc = context.WithCancel(ctx)
	for i := 0; i < wp.size; i++ {
		wp.wg.Add(1)
		go wp.runWorker(ctx, fmt.Sprintf("worker-%d", i))
	}
	wp.log.Info().Int("size", wp.size).Msg("worker pool started")
}

func (wp *WorkerPool) Shutdown() {
	wp.log.Info().Msg("shutting down worker pool")
	if wp.cancelFunc != nil {
		wp.cancelFunc()
	}
	done := make(chan struct{})
	go func() { wp.wg.Wait(); close(done) }()
	select {
	case <-done:
		wp.log.Info().Msg("all workers stopped gracefully")
	case <-time.After(60 * time.Second):
		wp.log.Warn().Msg("shutdown deadline exceeded, force killing containers")
		wp.container.ForceKillAll()
	}
}

func (wp *WorkerPool) runWorker(ctx context.Context, workerID string) {
	defer wp.wg.Done()
	log := wp.log.With().Str("worker", workerID).Logger()
	log.Info().Msg("worker started")

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("worker stopping")
			return
		default:
		}

		job, err := wp.queue.Pop(ctx, workerID)
		if err != nil {
			log.Error().Err(err).Msg("pop job failed")
			time.Sleep(1 * time.Second)
			continue
		}
		if job == nil {
			time.Sleep(1 * time.Second)
			continue
		}

		wp.executeJob(ctx, workerID, job, log)
	}
}

func (wp *WorkerPool) executeJob(ctx context.Context, workerID string, job *model.JobPayload, log zerolog.Logger) {
	log = log.With().
		Str("run_id", job.TestRunID).
		Str("repo", job.RepositoryFullName).
		Str("mode", job.ExecutionMode).
		Logger()

	log.Info().Msg("executing test run")

	runRepo := repository.NewTestRunRepository(wp.db)
	resultRepo := repository.NewTestResultRepository(wp.db)

	runID, _ := uuid.Parse(job.TestRunID)

	// Look up the executor
	executor, ok := wp.executors[job.ExecutionMode]
	if !ok {
		log.Error().Str("mode", job.ExecutionMode).Msg("unsupported execution mode")
		runRepo.UpdateFinished(ctx, runID, model.TestRunStatusFailed)
		saveSingleFailResult(ctx, resultRepo, runID, "Unsupported execution mode: "+job.ExecutionMode)
		wp.queue.Ack(ctx, workerID, job)
		return
	}

	// Build ExecutionJob
	execJob := &ExecutionJob{
		RunID:              runID,
		SuiteID:            uuid.MustParse(job.TestSuiteID),
		RepoID:             uuid.MustParse(job.RepoID),
		RepositoryFullName: job.RepositoryFullName,
		LocalPath:          job.LocalPath,
		DefaultBranch:      job.DefaultBranch,
		Branch:             job.Branch,
		CommitHash:         job.CommitHash,
		SuiteType:          job.SuiteType,
		ExecutionMode:      job.ExecutionMode,
		DockerImage:        job.DockerImage,
		TestCommand:        job.TestCommand,
		GHAWorkflowID:     job.GHAWorkflowID,
		ConfigPath:         job.ConfigPath,
		TimeoutSeconds:     job.TimeoutSeconds,
		EnvVars:            job.EnvVars,
	}

	// For GHA mode, inject PAT into env vars
	if job.ExecutionMode == ModeGHA {
		pat, err := wp.resolveTeamPAT(ctx, runID)
		if err != nil {
			log.Error().Err(err).Msg("failed to resolve PAT for GHA dispatch")
			runRepo.UpdateFinished(ctx, runID, model.TestRunStatusFailed)
			saveSingleFailResult(ctx, resultRepo, runID, "PAT resolution failed: "+err.Error())
			wp.queue.Ack(ctx, workerID, job)
			return
		}
		if execJob.EnvVars == nil {
			execJob.EnvVars = make(map[string]string)
		}
		execJob.EnvVars["_verdox_pat"] = pat

		// Update run status to running
		runRepo.UpdateStarted(ctx, runID)

		// GHA dispatch — non-blocking, returns immediately
		result, err := executor.Execute(ctx, execJob)
		if err != nil || result.Status == "failed" {
			errMsg := "GHA dispatch failed"
			if err != nil {
				errMsg = err.Error()
			} else if result.ErrorMsg != "" {
				errMsg = result.ErrorMsg
			}
			runRepo.UpdateFinished(ctx, runID, model.TestRunStatusFailed)
			saveSingleFailResult(ctx, resultRepo, runID, errMsg)
		}
		// GHA job dispatched — ack immediately, poller will track completion
		wp.queue.Ack(ctx, workerID, job)
		return
	}

	// Container mode — synchronous execution
	defer func() {
		// Reset git to default branch
		resetCmd := exec.CommandContext(ctx, "git", "checkout", job.DefaultBranch)
		resetCmd.Dir = job.LocalPath
		resetCmd.Run()

		wp.queue.Ack(ctx, workerID, job)
	}()

	// Update run status to running
	if err := runRepo.UpdateStarted(ctx, runID); err != nil {
		log.Error().Err(err).Msg("failed to update run status to running")
		runRepo.UpdateFinished(ctx, runID, model.TestRunStatusFailed)
		return
	}

	// Prepare workspace: git fetch + checkout
	fetchCmd := exec.CommandContext(ctx, "git", "fetch", "--depth", "1", "origin", job.CommitHash)
	fetchCmd.Dir = job.LocalPath
	fetchOutput, err := fetchCmd.CombinedOutput()
	if err != nil {
		log.Error().Err(err).Str("output", string(fetchOutput)).Msg("git fetch failed")
		runRepo.UpdateFinished(ctx, runID, model.TestRunStatusFailed)
		saveSingleFailResult(ctx, resultRepo, runID, "Git fetch failed: "+string(fetchOutput))
		return
	}

	checkoutCmd := exec.CommandContext(ctx, "git", "checkout", "FETCH_HEAD")
	checkoutCmd.Dir = job.LocalPath
	checkoutOutput, err := checkoutCmd.CombinedOutput()
	if err != nil {
		log.Error().Err(err).Str("output", string(checkoutOutput)).Msg("git checkout failed")
		runRepo.UpdateFinished(ctx, runID, model.TestRunStatusFailed)
		saveSingleFailResult(ctx, resultRepo, runID, "Git checkout failed: "+string(checkoutOutput))
		return
	}

	// Execute in container
	result, err := executor.Execute(ctx, execJob)
	if err != nil {
		log.Error().Err(err).Msg("container execution failed")
		runRepo.UpdateFinished(ctx, runID, model.TestRunStatusFailed)
		saveSingleFailResult(ctx, resultRepo, runID, "Execution failed: "+err.Error())
		return
	}

	// Check if run was cancelled
	currentRun, _ := runRepo.GetByID(ctx, runID)
	if currentRun != nil && currentRun.Status == model.TestRunStatusCancelled {
		log.Info().Msg("run was cancelled")
		return
	}

	// Batch insert results
	if len(result.Results) > 0 {
		modelResults := make([]model.TestResult, len(result.Results))
		for i, p := range result.Results {
			r := model.TestResult{
				TestRunID: runID,
				TestName:  p.TestName,
				Status:    model.TestResultStatus(p.Status),
			}
			if p.DurationMs > 0 {
				ms := int(p.DurationMs)
				r.DurationMs = &ms
			}
			if p.ErrorMessage != "" {
				r.ErrorMessage = &p.ErrorMessage
			}
			if p.LogOutput != "" {
				r.LogOutput = &p.LogOutput
			}
			modelResults[i] = r
		}

		if err := resultRepo.BatchCreate(ctx, modelResults); err != nil {
			log.Error().Err(err).Msg("failed to batch insert results")
		}
	}

	// Update final status
	finalStatus := model.TestRunStatusPassed
	if result.Status == "failed" {
		finalStatus = model.TestRunStatusFailed
	}

	if err := runRepo.UpdateFinished(ctx, runID, finalStatus); err != nil {
		log.Error().Err(err).Msg("failed to update final status")
	}

	log.Info().Str("status", string(finalStatus)).Int("results", len(result.Results)).Msg("test run complete")
}

// resolveTeamPAT fetches and decrypts the team's GitHub PAT for a given run.
func (wp *WorkerPool) resolveTeamPAT(ctx context.Context, runID uuid.UUID) (string, error) {
	runRepo := repository.NewTestRunRepository(wp.db)
	run, err := runRepo.GetByID(ctx, runID)
	if err != nil {
		return "", fmt.Errorf("get run: %w", err)
	}

	suiteRepo := repository.NewTestSuiteRepository(wp.db)
	suite, err := suiteRepo.GetByID(ctx, run.TestSuiteID)
	if err != nil {
		return "", fmt.Errorf("get suite: %w", err)
	}

	repoRepo := repository.NewRepositoryRepository(wp.db)
	teamID, err := repoRepo.GetTeamIDForRepository(ctx, suite.RepositoryID)
	if err != nil {
		return "", fmt.Errorf("get team: %w", err)
	}

	teamRepo := repository.NewTeamRepository(wp.db)
	team, err := teamRepo.GetByID(ctx, teamID)
	if err != nil {
		return "", fmt.Errorf("get team: %w", err)
	}

	if team.GithubPATEncrypted == nil || team.GithubPATNonce == nil {
		return "", fmt.Errorf("team has no PAT configured")
	}

	pat, err := encryption.Decrypt(*team.GithubPATEncrypted, team.GithubPATNonce, wp.cfg.GithubTokenEncryptionKey)
	if err != nil {
		return "", fmt.Errorf("decrypt PAT: %w", err)
	}

	return pat, nil
}

func saveSingleFailResult(ctx context.Context, repo repository.TestResultRepository, runID uuid.UUID, errMsg string) {
	results := []model.TestResult{{
		TestRunID:    runID,
		TestName:     "setup",
		Status:       model.TestResultStatusError,
		ErrorMessage: &errMsg,
	}}
	repo.BatchCreate(ctx, results)
}
