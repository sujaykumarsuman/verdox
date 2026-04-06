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
)

type WorkerPool struct {
	size       int
	queue      *queue.RedisQueue
	executor   *Executor
	parser     *Parser
	db         *sqlx.DB
	rdb        *redis.Client
	cfg        *config.Config
	log        zerolog.Logger
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup
}

func NewWorkerPool(cfg *config.Config, q *queue.RedisQueue, db *sqlx.DB, rdb *redis.Client, log zerolog.Logger) *WorkerPool {
	return &WorkerPool{
		size:     cfg.RunnerMaxConcurrent,
		queue:    q,
		executor: NewExecutor(cfg, rdb, q, log),
		parser:   NewParser(),
		db:       db,
		rdb:      rdb,
		cfg:      cfg,
		log:      log,
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
		wp.executor.ForceKillAll()
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
			// No work available, sleep briefly
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
		Str("branch", job.Branch).
		Logger()

	log.Info().Msg("executing test run")

	runRepo := repository.NewTestRunRepository(wp.db)
	resultRepo := repository.NewTestResultRepository(wp.db)
	repoRepo := repository.NewRepositoryRepository(wp.db)

	runID, _ := uuid.Parse(job.TestRunID)

	// Always ack + cleanup
	var containerID string
	defer func() {
		if containerID != "" {
			if err := wp.executor.RemoveContainer(ctx, containerID, job.TestRunID); err != nil {
				log.Warn().Err(err).Msg("failed to remove container")
			}
		}

		// Reset git to default branch
		resetCmd := exec.CommandContext(ctx, "git", "checkout", job.DefaultBranch)
		resetCmd.Dir = job.LocalPath
		resetCmd.Run() // best effort

		wp.queue.Ack(ctx, workerID, job)
	}()

	// 1. Update status to running
	if err := runRepo.UpdateStarted(ctx, runID); err != nil {
		log.Error().Err(err).Msg("failed to update run status to running")
		runRepo.UpdateFinished(ctx, runID, model.TestRunStatusFailed)
		return
	}

	// 2. Fetch target branch/commit in local clone
	repo, err := repoRepo.GetByID(ctx, runID)
	_ = repo // repo used implicitly via job.LocalPath

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

	// 3. Load verdox.yaml config
	verdoxCfg, _ := LoadVerdoxConfig(job.LocalPath, job.ConfigPath)
	var suiteCfg *SuiteConfig
	if verdoxCfg != nil {
		// Try to match by suite name — we don't have the suite name in the job payload,
		// so match by type as a fallback
		for i := range verdoxCfg.Suites {
			if verdoxCfg.Suites[i].Type == job.TestType {
				suiteCfg = &verdoxCfg.Suites[i]
				break
			}
		}
	}

	dockerImage := SelectImage(job, suiteCfg, wp.cfg.RunnerDefaultImage)
	testCommand := BuildTestCommand(job, suiteCfg)

	log.Info().Str("image", dockerImage).Str("command", testCommand).Msg("creating container")

	// 4. Create Docker container
	containerID, err = wp.executor.CreateContainer(ctx, job, testCommand, dockerImage)
	if err != nil {
		log.Error().Err(err).Msg("failed to create container")
		runRepo.UpdateFinished(ctx, runID, model.TestRunStatusFailed)
		saveSingleFailResult(ctx, resultRepo, runID, "Docker error: "+err.Error())
		return
	}

	// 5. Start container + subscribe to cancel
	if err := wp.executor.StartContainer(ctx, containerID); err != nil {
		log.Error().Err(err).Msg("failed to start container")
		runRepo.UpdateFinished(ctx, runID, model.TestRunStatusFailed)
		saveSingleFailResult(ctx, resultRepo, runID, "Container start failed: "+err.Error())
		return
	}

	// Cancel listener
	cancelSub := wp.queue.SubscribeCancel(ctx, job.TestRunID)
	defer cancelSub.Close()
	go func() {
		ch := cancelSub.Channel()
		select {
		case <-ch:
			log.Info().Msg("received cancel signal")
			wp.executor.docker.ContainerKill(context.Background(), containerID, "SIGKILL")
		case <-ctx.Done():
		}
	}()

	// 6. Stream logs
	output, err := wp.executor.StreamLogs(ctx, containerID, job.TestRunID)
	if err != nil {
		log.Warn().Err(err).Msg("failed to stream logs")
	}

	// 7. Wait for completion
	timeout := time.Duration(job.TimeoutSeconds) * time.Second
	maxTimeout := time.Duration(wp.cfg.RunnerMaxTimeout) * time.Second
	if timeout > maxTimeout {
		timeout = maxTimeout
	}

	exitCode, err := wp.executor.WaitWithTimeout(ctx, containerID, timeout)
	if err != nil {
		log.Error().Err(err).Msg("container wait failed")
		runRepo.UpdateFinished(ctx, runID, model.TestRunStatusFailed)
		errMsg := "Container execution failed"
		if ctx.Err() != nil {
			errMsg = fmt.Sprintf("Test run exceeded timeout of %d seconds", job.TimeoutSeconds)
		}
		saveSingleFailResult(ctx, resultRepo, runID, errMsg)
		return
	}

	// Check if run was cancelled
	currentRun, _ := runRepo.GetByID(ctx, runID)
	if currentRun != nil && currentRun.Status == model.TestRunStatusCancelled {
		log.Info().Msg("run was cancelled")
		return
	}

	log.Info().Int64("exit_code", exitCode).Msg("container finished")

	// 8. Parse output
	parsed := wp.parser.Parse(output, job.TestType, exitCode)

	// 9. Batch insert results
	results := make([]model.TestResult, len(parsed))
	for i, p := range parsed {
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
		results[i] = r
	}

	if err := resultRepo.BatchCreate(ctx, results); err != nil {
		log.Error().Err(err).Msg("failed to batch insert results")
	}

	// 10. Determine final status
	finalStatus := model.TestRunStatusPassed
	for _, r := range results {
		if r.Status == model.TestResultStatusFail || r.Status == model.TestResultStatusError {
			finalStatus = model.TestRunStatusFailed
			break
		}
	}

	if exitCode != 0 && finalStatus == model.TestRunStatusPassed {
		finalStatus = model.TestRunStatusFailed
	}

	if err := runRepo.UpdateFinished(ctx, runID, finalStatus); err != nil {
		log.Error().Err(err).Msg("failed to update final status")
	}

	log.Info().Str("status", string(finalStatus)).Int("results", len(results)).Msg("test run complete")
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
