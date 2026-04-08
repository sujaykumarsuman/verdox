package runner

import (
	"context"
	"fmt"
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
)

type WorkerPool struct {
	size       int
	queue      *queue.RedisQueue
	executors  map[string]Executor
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
	forkGHAExec *ForkGHAExecutor,
) *WorkerPool {
	executors := map[string]Executor{}
	if forkGHAExec != nil {
		executors[ModeForkGHA] = forkGHAExec
	}

	return &WorkerPool{
		size:      cfg.RunnerMaxConcurrent,
		queue:     q,
		executors: executors,
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
		wp.log.Warn().Msg("shutdown deadline exceeded")
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
		Logger()

	log.Info().Msg("executing test run")

	runRepo := repository.NewTestRunRepository(wp.db)
	resultRepo := repository.NewTestResultRepository(wp.db)

	runID, _ := uuid.Parse(job.TestRunID)

	// Look up the fork_gha executor
	executor, ok := wp.executors[ModeForkGHA]
	if !ok {
		log.Error().Msg("fork_gha executor not available")
		runRepo.UpdateFinished(ctx, runID, model.TestRunStatusFailed)
		saveSingleFailResult(ctx, resultRepo, runID, "Fork GHA executor not configured. Check VERDOX_SERVICE_ACCOUNT_PAT.")
		wp.queue.Ack(ctx, workerID, job)
		return
	}

	// Build ExecutionJob
	execJob := &ExecutionJob{
		RunID:              runID,
		SuiteID:            uuid.MustParse(job.TestSuiteID),
		RepoID:             uuid.MustParse(job.RepoID),
		RepositoryFullName: job.RepositoryFullName,
		DefaultBranch:      job.DefaultBranch,
		Branch:             job.Branch,
		CommitHash:         job.CommitHash,
		SuiteType:          job.SuiteType,
		ExecutionMode:      ModeForkGHA,
		TestCommand:        job.TestCommand,
		GHAWorkflowID:      job.GHAWorkflowID,
		ConfigPath:         job.ConfigPath,
		TimeoutSeconds:     job.TimeoutSeconds,
		EnvVars:            job.EnvVars,
	}

	// Inject fork info from the repository
	if execJob.EnvVars == nil {
		execJob.EnvVars = make(map[string]string)
	}
	var forkFullName, forkHeadSHA *string
	err := wp.db.QueryRowContext(ctx,
		"SELECT fork_full_name, fork_head_sha FROM repositories WHERE id = $1",
		execJob.RepoID,
	).Scan(&forkFullName, &forkHeadSHA)
	if err != nil || forkFullName == nil || *forkFullName == "" {
		log.Error().Err(err).Msg("fork not set up for this repository")
		runRepo.UpdateFinished(ctx, runID, model.TestRunStatusFailed)
		saveSingleFailResult(ctx, resultRepo, runID, "Fork not set up for this repository. Run fork setup first.")
		wp.queue.Ack(ctx, workerID, job)
		return
	}
	execJob.EnvVars["_fork_full_name"] = *forkFullName
	if forkHeadSHA != nil {
		execJob.EnvVars["_fork_head_sha"] = *forkHeadSHA
	}

	// Update run status to running and dispatch
	runRepo.UpdateStarted(ctx, runID)

	var result *ExecutionResult
	if job.IsRerun && job.OriginalGHARunID > 0 {
		// Rerun: call GitHub's rerun API instead of dispatching a new workflow
		if forkExec, ok := executor.(*ForkGHAExecutor); ok {
			result, err = forkExec.Rerun(ctx, execJob, job.OriginalGHARunID)
		} else {
			err = fmt.Errorf("rerun not supported for this executor")
		}
	} else {
		result, err = executor.Execute(ctx, execJob)
	}

	if err != nil || (result != nil && result.Status == "failed") {
		errMsg := "Fork GHA dispatch failed"
		if err != nil {
			errMsg = err.Error()
		} else if result != nil && result.ErrorMsg != "" {
			errMsg = result.ErrorMsg
		}
		runRepo.UpdateFinished(ctx, runID, model.TestRunStatusFailed)
		saveSingleFailResult(ctx, resultRepo, runID, errMsg)
	}
	wp.queue.Ack(ctx, workerID, job)
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
