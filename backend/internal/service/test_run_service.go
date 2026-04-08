package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/sujaykumarsuman/verdox/backend/internal/config"
	"github.com/sujaykumarsuman/verdox/backend/internal/dto"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
	"github.com/sujaykumarsuman/verdox/backend/internal/queue"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
)

var (
	ErrRunNotFound       = errors.New("test run not found")
	ErrRunConflict       = errors.New("a run for this commit is already queued or running")
	ErrRunNotCancellable = errors.New("run is already in a terminal state")
	ErrRunNotRerunnable  = errors.New("run is not in a failed state or has no GHA run ID")
)

type TestRunService struct {
	runRepo        repository.TestRunRepository
	resultRepo     repository.TestResultRepository
	suiteRepo      repository.TestSuiteRepository
	repoRepo       repository.RepositoryRepository
	teamMemberRepo repository.TeamMemberRepository
	userRepo       repository.UserRepository
	groupRepo      repository.TestGroupRepository
	queue          *queue.RedisQueue
	rdb            *redis.Client
	cfg            *config.Config
	log            zerolog.Logger
}

func NewTestRunService(
	runRepo repository.TestRunRepository,
	resultRepo repository.TestResultRepository,
	suiteRepo repository.TestSuiteRepository,
	repoRepo repository.RepositoryRepository,
	teamMemberRepo repository.TeamMemberRepository,
	userRepo repository.UserRepository,
	groupRepo repository.TestGroupRepository,
	q *queue.RedisQueue,
	rdb *redis.Client,
	cfg *config.Config,
	log zerolog.Logger,
) *TestRunService {
	return &TestRunService{
		runRepo:        runRepo,
		resultRepo:     resultRepo,
		suiteRepo:      suiteRepo,
		repoRepo:       repoRepo,
		teamMemberRepo: teamMemberRepo,
		userRepo:       userRepo,
		groupRepo:      groupRepo,
		queue:          q,
		rdb:            rdb,
		cfg:            cfg,
		log:            log,
	}
}

func (s *TestRunService) TriggerRun(ctx context.Context, userID, suiteID uuid.UUID, req *dto.TriggerRunRequest) (*dto.TestRunResponse, error) {
	suite, err := s.suiteRepo.GetByID(ctx, suiteID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrSuiteNotFound
		}
		return nil, fmt.Errorf("get suite: %w", err)
	}

	repo, err := s.repoRepo.GetByID(ctx, suite.RepositoryID)
	if err != nil {
		return nil, fmt.Errorf("get repo: %w", err)
	}

	// Fork must be ready before running tests
	if repo.ForkStatus != model.ForkStatusReady {
		return nil, ErrForkNotReady
	}

	// Verify team membership (not viewer)
	teamID, err := s.repoRepo.GetTeamIDForRepository(ctx, suite.RepositoryID)
	if err != nil {
		return nil, fmt.Errorf("get team: %w", err)
	}
	member, err := s.teamMemberRepo.GetByTeamAndUser(ctx, teamID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotTeamMember
		}
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if member.Role == model.TeamMemberRoleViewer {
		return nil, ErrNotAdminOrMaintainer
	}

	// Commit-hash caching: check if same suite+branch+commit already ran
	existing, err := s.runRepo.FindTerminalRun(ctx, suiteID, req.Branch, req.CommitHash)
	if err != nil {
		return nil, fmt.Errorf("find terminal run: %w", err)
	}
	if existing != nil {
		resp := dto.NewTestRunResponse(existing)
		return &resp, nil
	}

	// Assign run number
	runNumber, err := s.runRepo.NextRunNumber(ctx, suiteID)
	if err != nil {
		return nil, fmt.Errorf("next run number: %w", err)
	}

	run := &model.TestRun{
		TestSuiteID: suiteID,
		TriggeredBy: &userID,
		RunNumber:   runNumber,
		Branch:      req.Branch,
		CommitHash:  req.CommitHash,
		Status:      model.TestRunStatusQueued,
	}

	if err := s.runRepo.Create(ctx, run); err != nil {
		return nil, fmt.Errorf("create test run: %w", err)
	}

	configPath := ""
	if suite.ConfigPath != nil {
		configPath = *suite.ConfigPath
	}
	dockerImage := ""
	if suite.DockerImage != nil {
		dockerImage = *suite.DockerImage
	}
	testCommand := ""
	if suite.TestCommand != nil {
		testCommand = *suite.TestCommand
	}
	ghaWorkflowID := ""
	if suite.GHAWorkflowID != nil {
		ghaWorkflowID = *suite.GHAWorkflowID
	}
	envVars := make(map[string]string)
	for k, v := range suite.EnvVars {
		envVars[k] = v
	}

	payload := &model.JobPayload{
		TestRunID:          run.ID.String(),
		TestSuiteID:        suite.ID.String(),
		RepoID:             repo.ID.String(),
		RepositoryFullName: repo.GithubFullName,
		DefaultBranch:      repo.DefaultBranch,
		Branch:             req.Branch,
		CommitHash:         req.CommitHash,
		SuiteType:          suite.Type,
		ExecutionMode:      suite.ExecutionMode,
		DockerImage:        dockerImage,
		TestCommand:        testCommand,
		GHAWorkflowID:      ghaWorkflowID,
		ConfigPath:         configPath,
		TimeoutSeconds:     suite.TimeoutSeconds,
		EnvVars:            envVars,
	}

	if err := s.queue.Push(ctx, payload); err != nil {
		return nil, fmt.Errorf("push job: %w", err)
	}

	resp := dto.NewTestRunResponse(run)
	return &resp, nil
}

// RerunRun re-triggers a failed GHA workflow run via the GitHub rerun API.
func (s *TestRunService) RerunRun(ctx context.Context, userID, runID uuid.UUID) (*dto.TestRunResponse, error) {
	originalRun, err := s.runRepo.GetByID(ctx, runID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrRunNotFound
		}
		return nil, fmt.Errorf("get run: %w", err)
	}

	// Only terminal non-passed runs can be rerun (failed, cancelled, or setup errors)
	if originalRun.Status != model.TestRunStatusFailed && originalRun.Status != model.TestRunStatusCancelled {
		return nil, ErrRunNotRerunnable
	}

	suite, err := s.suiteRepo.GetByID(ctx, originalRun.TestSuiteID)
	if err != nil {
		return nil, fmt.Errorf("get suite: %w", err)
	}

	repo, err := s.repoRepo.GetByID(ctx, suite.RepositoryID)
	if err != nil {
		return nil, fmt.Errorf("get repo: %w", err)
	}
	if repo.ForkStatus != model.ForkStatusReady {
		return nil, ErrForkNotReady
	}

	// Auth check
	teamID, err := s.repoRepo.GetTeamIDForRepository(ctx, suite.RepositoryID)
	if err != nil {
		return nil, fmt.Errorf("get team: %w", err)
	}
	member, err := s.teamMemberRepo.GetByTeamAndUser(ctx, teamID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotTeamMember
		}
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if member.Role == model.TeamMemberRoleViewer {
		return nil, ErrNotAdminOrMaintainer
	}

	// Create new run (same branch/commit, new run number)
	runNumber, err := s.runRepo.NextRunNumber(ctx, suite.ID)
	if err != nil {
		return nil, fmt.Errorf("next run number: %w", err)
	}

	newRun := &model.TestRun{
		TestSuiteID: suite.ID,
		TriggeredBy: &userID,
		RunNumber:   runNumber,
		Branch:      originalRun.Branch,
		CommitHash:  originalRun.CommitHash,
		Status:      model.TestRunStatusQueued,
	}
	if err := s.runRepo.Create(ctx, newRun); err != nil {
		return nil, fmt.Errorf("create rerun: %w", err)
	}

	payload := &model.JobPayload{
		TestRunID:          newRun.ID.String(),
		TestSuiteID:        suite.ID.String(),
		RepoID:             repo.ID.String(),
		RepositoryFullName: repo.GithubFullName,
		DefaultBranch:      repo.DefaultBranch,
		Branch:             originalRun.Branch,
		CommitHash:         originalRun.CommitHash,
		SuiteType:          suite.Type,
		ExecutionMode:      suite.ExecutionMode,
		TimeoutSeconds:     suite.TimeoutSeconds,
	}

	// Always do a fresh workflow_dispatch for reruns. GitHub's rerun API
	// re-runs the exact same workflow run ID, which can trigger the wrong
	// workflow if the fork has multiple workflows (e.g., upstream CI).

	if err := s.queue.Push(ctx, payload); err != nil {
		return nil, fmt.Errorf("push rerun job: %w", err)
	}

	resp := dto.NewTestRunResponse(newRun)
	return &resp, nil
}

func (s *TestRunService) GetRun(ctx context.Context, userID, runID uuid.UUID) (*dto.TestRunDetailResponse, error) {
	run, err := s.runRepo.GetByID(ctx, runID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrRunNotFound
		}
		return nil, fmt.Errorf("get run: %w", err)
	}

	suite, err := s.suiteRepo.GetByID(ctx, run.TestSuiteID)
	if err != nil {
		return nil, fmt.Errorf("get suite: %w", err)
	}

	repo, err := s.repoRepo.GetByID(ctx, suite.RepositoryID)
	if err != nil {
		return nil, fmt.Errorf("get repo: %w", err)
	}

	// Verify access
	teamID, err := s.repoRepo.GetTeamIDForRepository(ctx, suite.RepositoryID)
	if err != nil {
		return nil, fmt.Errorf("get team: %w", err)
	}
	isMember, err := s.teamMemberRepo.IsTeamMember(ctx, teamID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotTeamMember
	}

	results, err := s.resultRepo.ListByRunID(ctx, run.ID)
	if err != nil {
		return nil, fmt.Errorf("list results: %w", err)
	}

	// Build summary
	summary := &dto.RunSummary{Total: len(results)}
	var totalDuration int64
	for _, r := range results {
		switch r.Status {
		case model.TestResultStatusPass:
			summary.Passed++
		case model.TestResultStatusFail:
			summary.Failed++
		case model.TestResultStatusSkip:
			summary.Skipped++
		case model.TestResultStatusError:
			summary.Errors++
		}
		if r.DurationMs != nil {
			totalDuration += int64(*r.DurationMs)
		}
	}
	summary.DurationMs = totalDuration

	// Build result responses
	resultResps := make([]dto.TestResultResponse, len(results))
	for i := range results {
		resultResps[i] = dto.NewTestResultResponse(&results[i])
	}

	// Use fork full name for GHA URL (runs execute on the fork, not upstream)
	ghaRepoName := repo.GithubFullName
	if repo.ForkFullName != nil && *repo.ForkFullName != "" {
		ghaRepoName = *repo.ForkFullName
	}
	runResp := dto.NewTestRunResponseWithGHA(run, ghaRepoName)

	// Get triggered-by username
	if run.TriggeredBy != nil {
		user, err := s.userRepo.GetByID(ctx, *run.TriggeredBy)
		if err == nil {
			runResp.TriggeredByUsername = &user.Username
		}
	}

	detail := &dto.TestRunDetailResponse{
		TestRunResponse: runResp,
		SuiteName:       suite.Name,
		SuiteType:       suite.Type,
		ExecutionMode:   suite.ExecutionMode,
		RepositoryID:    repo.ID.String(),
		RepositoryName:  repo.Name,
		Summary:         summary,
		Results:         resultResps,
		ReportID:        run.ReportID,
	}

	// If the run has hierarchical data (summary JSONB set), include it
	if run.Summary != nil {
		var summaryV2 dto.RunSummaryV2
		if err := json.Unmarshal([]byte(*run.Summary), &summaryV2); err == nil {
			detail.SummaryV2 = &summaryV2
		}

		// Fetch groups for the hierarchical view
		groups, err := s.groupRepo.ListByRunID(ctx, run.ID)
		if err == nil && len(groups) > 0 {
			groupResps := make([]dto.TestGroupResponse, len(groups))
			for i := range groups {
				groupResps[i] = dto.NewTestGroupResponse(&groups[i])
			}
			detail.Groups = groupResps
		}
	}

	return detail, nil
}

func (s *TestRunService) ListRunsBySuite(ctx context.Context, userID, suiteID uuid.UUID, status, branch string, page, perPage int) (*dto.TestRunListResponse, error) {
	suite, err := s.suiteRepo.GetByID(ctx, suiteID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrSuiteNotFound
		}
		return nil, fmt.Errorf("get suite: %w", err)
	}

	teamID, err := s.repoRepo.GetTeamIDForRepository(ctx, suite.RepositoryID)
	if err != nil {
		return nil, fmt.Errorf("get team: %w", err)
	}
	isMember, err := s.teamMemberRepo.IsTeamMember(ctx, teamID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotTeamMember
	}

	runs, total, err := s.runRepo.ListBySuiteID(ctx, suiteID, status, branch, page, perPage)
	if err != nil {
		return nil, fmt.Errorf("list runs: %w", err)
	}

	runResps := make([]dto.TestRunResponse, len(runs))
	for i := range runs {
		runResps[i] = dto.NewTestRunResponse(&runs[i])
	}

	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}

	return &dto.TestRunListResponse{
		Runs: runResps,
		Meta: dto.PaginationMeta{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *TestRunService) GetRunLogs(ctx context.Context, userID, runID uuid.UUID, testName string) (*dto.RunLogsResponse, error) {
	run, err := s.runRepo.GetByID(ctx, runID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrRunNotFound
		}
		return nil, fmt.Errorf("get run: %w", err)
	}

	suite, err := s.suiteRepo.GetByID(ctx, run.TestSuiteID)
	if err != nil {
		return nil, fmt.Errorf("get suite: %w", err)
	}

	teamID, err := s.repoRepo.GetTeamIDForRepository(ctx, suite.RepositoryID)
	if err != nil {
		return nil, fmt.Errorf("get team: %w", err)
	}
	isMember, err := s.teamMemberRepo.IsTeamMember(ctx, teamID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotTeamMember
	}

	if testName != "" {
		result, err := s.resultRepo.GetByRunIDAndTestName(ctx, runID, testName)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, ErrRunNotFound
			}
			return nil, fmt.Errorf("get result: %w", err)
		}
		return &dto.RunLogsResponse{
			RunID: runID.String(),
			Logs: []dto.TestLogEntry{{
				TestName:   result.TestName,
				Status:     string(result.Status),
				DurationMs: result.DurationMs,
				LogOutput:  result.LogOutput,
			}},
		}, nil
	}

	results, err := s.resultRepo.ListByRunID(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("list results: %w", err)
	}

	logs := make([]dto.TestLogEntry, len(results))
	for i, r := range results {
		logs[i] = dto.TestLogEntry{
			TestName:   r.TestName,
			Status:     string(r.Status),
			DurationMs: r.DurationMs,
			LogOutput:  r.LogOutput,
		}
	}

	// Fallback: if no test results exist but the run has log_output stored
	// directly, create a single entry so the frontend can display something
	if len(logs) == 0 && run.LogOutput != nil && *run.LogOutput != "" {
		logs = []dto.TestLogEntry{{
			TestName:  "Test Output",
			Status:    string(run.Status),
			LogOutput: run.LogOutput,
		}}
	}

	return &dto.RunLogsResponse{RunID: runID.String(), Logs: logs}, nil
}

func (s *TestRunService) CancelRun(ctx context.Context, userID, runID uuid.UUID) (*dto.CancelRunResponse, error) {
	run, err := s.runRepo.GetByID(ctx, runID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrRunNotFound
		}
		return nil, fmt.Errorf("get run: %w", err)
	}

	if run.Status.IsTerminal() {
		return nil, ErrRunNotCancellable
	}

	suite, err := s.suiteRepo.GetByID(ctx, run.TestSuiteID)
	if err != nil {
		return nil, fmt.Errorf("get suite: %w", err)
	}

	repo, err := s.repoRepo.GetByID(ctx, suite.RepositoryID)
	if err != nil {
		return nil, fmt.Errorf("get repo: %w", err)
	}

	// Verify: must be triggerer or team admin
	teamID, err := s.repoRepo.GetTeamIDForRepository(ctx, suite.RepositoryID)
	if err != nil {
		return nil, fmt.Errorf("get team: %w", err)
	}
	isAdmin, _ := s.teamMemberRepo.IsTeamAdmin(ctx, teamID, userID)
	isTriggerer := run.TriggeredBy != nil && *run.TriggeredBy == userID
	if !isAdmin && !isTriggerer {
		return nil, ErrNotTeamAdmin
	}

	switch run.Status {
	case model.TestRunStatusQueued:
		if err := s.queue.RemoveByRunID(ctx, repo.ID.String(), runID.String()); err != nil {
			s.log.Warn().Err(err).Msg("failed to remove job from queue")
		}
		if err := s.runRepo.UpdateFinished(ctx, runID, model.TestRunStatusCancelled); err != nil {
			return nil, fmt.Errorf("update status: %w", err)
		}
	case model.TestRunStatusRunning:
		if err := s.queue.PublishCancel(ctx, runID.String()); err != nil {
			return nil, fmt.Errorf("publish cancel: %w", err)
		}
	}

	return &dto.CancelRunResponse{
		ID:      runID.String(),
		Status:  string(model.TestRunStatusCancelled),
		Message: "Run cancellation requested",
	}, nil
}

func (s *TestRunService) RunAll(ctx context.Context, userID, repoID uuid.UUID, req *dto.RunAllRequest) (*dto.RunAllResponse, error) {
	repo, err := s.repoRepo.GetByID(ctx, repoID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("get repo: %w", err)
	}

	if repo.ForkStatus != model.ForkStatusReady {
		return nil, ErrForkNotReady
	}

	teamID, err := s.repoRepo.GetTeamIDForRepository(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("get team: %w", err)
	}
	member, err := s.teamMemberRepo.GetByTeamAndUser(ctx, teamID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotTeamMember
		}
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if member.Role == model.TeamMemberRoleViewer {
		return nil, ErrNotAdminOrMaintainer
	}

	suites, err := s.suiteRepo.ListByRepositoryID(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("list suites: %w", err)
	}

	if len(suites) == 0 {
		return nil, fmt.Errorf("no test suites configured for this repository")
	}

	triggerReq := &dto.TriggerRunRequest{
		Branch:     req.Branch,
		CommitHash: req.CommitHash,
	}

	var runs []dto.TestRunResponse
	for _, suite := range suites {
		runResp, err := s.TriggerRun(ctx, userID, suite.ID, triggerReq)
		if err != nil {
			s.log.Warn().Err(err).Str("suite_id", suite.ID.String()).Msg("failed to trigger run for suite")
			continue
		}
		runs = append(runs, *runResp)
	}

	return &dto.RunAllResponse{
		Message: fmt.Sprintf("Triggered %d runs", len(runs)),
		Runs:    runs,
	}, nil
}
