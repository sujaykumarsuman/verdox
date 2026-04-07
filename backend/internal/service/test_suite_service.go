package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/sujaykumarsuman/verdox/backend/internal/dto"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
)

var (
	ErrNotAdminOrMaintainer = errors.New("admin or maintainer role required")
	ErrSuiteNotFound        = errors.New("test suite not found")
)

type TestSuiteService struct {
	suiteRepo      repository.TestSuiteRepository
	repoRepo       repository.RepositoryRepository
	teamMemberRepo repository.TeamMemberRepository
	forkService    *ForkService
	log            zerolog.Logger
}

func NewTestSuiteService(
	suiteRepo repository.TestSuiteRepository,
	repoRepo repository.RepositoryRepository,
	teamMemberRepo repository.TeamMemberRepository,
	forkService *ForkService,
	log zerolog.Logger,
) *TestSuiteService {
	return &TestSuiteService{
		suiteRepo:      suiteRepo,
		repoRepo:       repoRepo,
		teamMemberRepo: teamMemberRepo,
		forkService:    forkService,
		log:            log,
	}
}

func (s *TestSuiteService) CreateSuite(ctx context.Context, userID, repoID uuid.UUID, req *dto.CreateTestSuiteRequest) (*dto.TestSuiteResponse, error) {
	if err := s.verifyAdminOrMaintainer(ctx, userID, repoID); err != nil {
		return nil, err
	}

	timeout := 300
	if req.TimeoutSeconds != nil {
		timeout = *req.TimeoutSeconds
	}

	execMode := model.ExecutionModeForkGHA
	if req.ExecutionMode != "" {
		execMode = req.ExecutionMode
	}

	var wfConfig model.WorkflowConfigJSON
	if req.WorkflowConfig != nil {
		wfConfig = model.WorkflowConfigJSON(*req.WorkflowConfig)
	}

	suite := &model.TestSuite{
		RepositoryID:   repoID,
		Name:           req.Name,
		Type:           req.Type,
		ExecutionMode:  execMode,
		DockerImage:    req.DockerImage,
		TestCommand:    req.TestCommand,
		GHAWorkflowID:  req.GHAWorkflowID,
		EnvVars:        model.EnvVarsMap(req.EnvVars),
		ConfigPath:     req.ConfigPath,
		TimeoutSeconds: timeout,
		WorkflowConfig: wfConfig,
		WorkflowYAML:   req.WorkflowYAML,
	}

	if err := s.suiteRepo.Create(ctx, suite); err != nil {
		return nil, fmt.Errorf("create test suite: %w", err)
	}

	// Push updated workflow files to fork (async — don't block response)
	if s.forkService != nil {
		go func() {
			if err := s.forkService.RefreshSuiteWorkflows(context.Background(), repoID); err != nil {
				s.log.Error().Err(err).Str("repo_id", repoID.String()).Msg("failed to refresh suite workflows after create")
			}
		}()
	}

	resp := dto.NewTestSuiteResponse(suite)
	return &resp, nil
}

func (s *TestSuiteService) GetSuite(ctx context.Context, userID, suiteID uuid.UUID) (*dto.TestSuiteResponse, error) {
	suite, err := s.suiteRepo.GetByID(ctx, suiteID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrSuiteNotFound
		}
		return nil, fmt.Errorf("get suite: %w", err)
	}

	if err := s.verifyMember(ctx, userID, suite.RepositoryID); err != nil {
		return nil, err
	}

	resp := dto.NewTestSuiteResponse(suite)
	return &resp, nil
}

func (s *TestSuiteService) ListSuites(ctx context.Context, userID, repoID uuid.UUID) ([]dto.TestSuiteResponse, error) {
	if err := s.verifyMember(ctx, userID, repoID); err != nil {
		return nil, err
	}

	suites, err := s.suiteRepo.ListByRepositoryID(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("list test suites: %w", err)
	}

	resp := make([]dto.TestSuiteResponse, len(suites))
	for i := range suites {
		resp[i] = dto.NewTestSuiteResponse(&suites[i])
	}
	return resp, nil
}

func (s *TestSuiteService) UpdateSuite(ctx context.Context, userID, suiteID uuid.UUID, req *dto.UpdateTestSuiteRequest) (*dto.TestSuiteResponse, error) {
	suite, err := s.suiteRepo.GetByID(ctx, suiteID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrSuiteNotFound
		}
		return nil, fmt.Errorf("get suite: %w", err)
	}

	if err := s.verifyAdminOrMaintainer(ctx, userID, suite.RepositoryID); err != nil {
		return nil, err
	}

	if req.Name != nil {
		suite.Name = *req.Name
	}
	if req.Type != nil {
		suite.Type = *req.Type
	}
	if req.ExecutionMode != nil {
		suite.ExecutionMode = *req.ExecutionMode
	}
	if req.DockerImage != nil {
		suite.DockerImage = req.DockerImage
	}
	if req.TestCommand != nil {
		suite.TestCommand = req.TestCommand
	}
	if req.GHAWorkflowID != nil {
		suite.GHAWorkflowID = req.GHAWorkflowID
	}
	if req.EnvVars != nil {
		suite.EnvVars = model.EnvVarsMap(req.EnvVars)
	}
	if req.ConfigPath != nil {
		suite.ConfigPath = req.ConfigPath
	}
	if req.TimeoutSeconds != nil {
		suite.TimeoutSeconds = *req.TimeoutSeconds
	}

	if err := s.suiteRepo.Update(ctx, suite); err != nil {
		return nil, fmt.Errorf("update test suite: %w", err)
	}

	// Push updated workflow files to fork (async)
	if s.forkService != nil {
		go func() {
			if err := s.forkService.RefreshSuiteWorkflows(context.Background(), suite.RepositoryID); err != nil {
				s.log.Error().Err(err).Str("suite_id", suiteID.String()).Msg("failed to refresh suite workflows after update")
			}
		}()
	}

	resp := dto.NewTestSuiteResponse(suite)
	return &resp, nil
}

func (s *TestSuiteService) DeleteSuite(ctx context.Context, userID, suiteID uuid.UUID) error {
	suite, err := s.suiteRepo.GetByID(ctx, suiteID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrSuiteNotFound
		}
		return fmt.Errorf("get suite: %w", err)
	}

	// Delete requires admin
	teamID, err := s.repoRepo.GetTeamIDForRepository(ctx, suite.RepositoryID)
	if err != nil {
		return fmt.Errorf("get team: %w", err)
	}
	isAdmin, err := s.teamMemberRepo.IsTeamAdmin(ctx, teamID, userID)
	if err != nil {
		return fmt.Errorf("check admin: %w", err)
	}
	if !isAdmin {
		return ErrNotTeamAdmin
	}

	if err := s.suiteRepo.Delete(ctx, suiteID); err != nil {
		return fmt.Errorf("delete test suite: %w", err)
	}

	// Push updated workflow files to fork (removes deleted suite's workflow)
	if s.forkService != nil {
		go func() {
			if err := s.forkService.RefreshSuiteWorkflows(context.Background(), suite.RepositoryID); err != nil {
				s.log.Error().Err(err).Str("suite_id", suiteID.String()).Msg("failed to refresh suite workflows after delete")
			}
		}()
	}

	return nil
}

// verifyAdminOrMaintainer checks that the user is an admin or maintainer of the repo's team.
func (s *TestSuiteService) verifyAdminOrMaintainer(ctx context.Context, userID, repoID uuid.UUID) error {
	teamID, err := s.repoRepo.GetTeamIDForRepository(ctx, repoID)
	if err != nil {
		return fmt.Errorf("get team: %w", err)
	}

	member, err := s.teamMemberRepo.GetByTeamAndUser(ctx, teamID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotTeamMember
		}
		return fmt.Errorf("check membership: %w", err)
	}

	if member.Role != model.TeamMemberRoleAdmin && member.Role != model.TeamMemberRoleMaintainer {
		return ErrNotAdminOrMaintainer
	}

	return nil
}

// verifyMember checks that the user is any member of the repo's team.
func (s *TestSuiteService) verifyMember(ctx context.Context, userID, repoID uuid.UUID) error {
	teamID, err := s.repoRepo.GetTeamIDForRepository(ctx, repoID)
	if err != nil {
		return fmt.Errorf("get team: %w", err)
	}

	isMember, err := s.teamMemberRepo.IsTeamMember(ctx, teamID, userID)
	if err != nil {
		return fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return ErrNotTeamMember
	}
	return nil
}
