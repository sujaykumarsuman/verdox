package service

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/sujaykumarsuman/verdox/backend/internal/config"
	"github.com/sujaykumarsuman/verdox/backend/internal/dto"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
	"github.com/sujaykumarsuman/verdox/backend/pkg/encryption"
)

var (
	ErrPATNotConfigured = errors.New("no GitHub PAT available (team PAT or service account)")
	ErrDuplicateRepo    = errors.New("repository already added")
	ErrNotTeamMember    = errors.New("user is not a member of this team")
	ErrNotTeamAdmin     = errors.New("user is not an admin of this team")
	ErrForkNotReady     = errors.New("repository fork is not ready")
)

type RepositoryService struct {
	repoRepo       repository.RepositoryRepository
	teamRepo       repository.TeamRepository
	teamMemberRepo repository.TeamMemberRepository
	githubService  *GitHubService
	forkService    *ForkService
	rdb            *redis.Client
	cfg            *config.Config
	log            zerolog.Logger
}

func NewRepositoryService(
	repoRepo repository.RepositoryRepository,
	teamRepo repository.TeamRepository,
	teamMemberRepo repository.TeamMemberRepository,
	githubService *GitHubService,
	forkService *ForkService,
	rdb *redis.Client,
	cfg *config.Config,
	log zerolog.Logger,
) *RepositoryService {
	return &RepositoryService{
		repoRepo:       repoRepo,
		teamRepo:       teamRepo,
		teamMemberRepo: teamMemberRepo,
		githubService:  githubService,
		forkService:    forkService,
		rdb:            rdb,
		cfg:            cfg,
		log:            log,
	}
}

// resolvePAT returns the best available PAT: team PAT if configured, otherwise service account PAT.
func (s *RepositoryService) resolvePAT(ctx context.Context, teamID uuid.UUID) (string, error) {
	team, err := s.teamRepo.GetByID(ctx, teamID)
	if err != nil {
		return "", fmt.Errorf("get team: %w", err)
	}

	// Try team PAT first
	if team.HasPAT() {
		pat, err := encryption.Decrypt(*team.GithubPATEncrypted, team.GithubPATNonce, s.cfg.GithubTokenEncryptionKey)
		if err != nil {
			return "", fmt.Errorf("decrypt team pat: %w", err)
		}
		return pat, nil
	}

	// Fall back to service account PAT
	if s.cfg.ServiceAccountPAT != "" {
		return s.cfg.ServiceAccountPAT, nil
	}

	return "", ErrPATNotConfigured
}

func (s *RepositoryService) AddRepository(ctx context.Context, userID uuid.UUID, req *dto.AddRepositoryRequest) (*dto.RepositoryResponse, error) {
	teamID, err := uuid.Parse(req.TeamID)
	if err != nil {
		return nil, fmt.Errorf("invalid team_id: %w", err)
	}

	// Verify team membership
	isMember, err := s.teamMemberRepo.IsTeamMember(ctx, teamID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotTeamMember
	}

	// Resolve PAT: team PAT → service account PAT fallback
	pat, err := s.resolvePAT(ctx, teamID)
	if err != nil {
		return nil, err
	}

	// Parse GitHub URL
	owner, repoName, err := ParseGitHubURL(req.GithubURL)
	if err != nil {
		return nil, err
	}

	// Fetch repo info from GitHub
	ghInfo, err := s.githubService.GetRepository(ctx, pat, owner, repoName)
	if err != nil {
		return nil, err
	}

	// Check for duplicate — reactivate if previously soft-deleted
	existing, err := s.repoRepo.GetByGithubRepoID(ctx, ghInfo.ID)
	if err == nil && existing != nil {
		if existing.IsActive {
			return nil, ErrDuplicateRepo
		}
		// Reactivate
		existing.IsActive = true
		existing.Description = &ghInfo.Description
		existing.DefaultBranch = ghInfo.DefaultBranch
		if err := s.repoRepo.Reactivate(ctx, existing); err != nil {
			return nil, fmt.Errorf("reactivate repository: %w", err)
		}
		// Auto-fork in background
		if s.forkService != nil && s.forkService.IsConfigured() {
			go func() {
				_ = s.forkService.SetupFork(context.Background(), existing.ID, ghInfo.FullName, ghInfo.DefaultBranch)
			}()
		}
		resp := dto.NewRepositoryResponse(existing, teamID.String())
		return &resp, nil
	}

	// Create repo record
	desc := ghInfo.Description
	repo := &model.Repository{
		GithubRepoID:   ghInfo.ID,
		GithubFullName: ghInfo.FullName,
		Name:           ghInfo.Name,
		Description:    &desc,
		DefaultBranch:  ghInfo.DefaultBranch,
		IsActive:       true,
	}

	if err := s.repoRepo.Create(ctx, repo); err != nil {
		return nil, fmt.Errorf("create repository: %w", err)
	}

	// Create junction row
	if err := s.repoRepo.AddTeamRepository(ctx, teamID, repo.ID, userID); err != nil {
		return nil, fmt.Errorf("add team repository: %w", err)
	}

	// Auto-fork in background
	if s.forkService != nil && s.forkService.IsConfigured() {
		go func() {
			_ = s.forkService.SetupFork(context.Background(), repo.ID, ghInfo.FullName, ghInfo.DefaultBranch)
		}()
	}

	s.log.Info().
		Str("repo", ghInfo.FullName).
		Msg("repository added, fork initiated")

	resp := dto.NewRepositoryResponse(repo, teamID.String())
	return &resp, nil
}

func (s *RepositoryService) ListRepositories(ctx context.Context, userID uuid.UUID, teamID string, search string, page, perPage int) (*dto.RepositoryListResponse, error) {
	tid, err := uuid.Parse(teamID)
	if err != nil {
		return nil, fmt.Errorf("invalid team_id: %w", err)
	}

	isMember, err := s.teamMemberRepo.IsTeamMember(ctx, tid, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotTeamMember
	}

	repos, total, err := s.repoRepo.ListByTeamID(ctx, tid, search, page, perPage)
	if err != nil {
		return nil, fmt.Errorf("list repositories: %w", err)
	}

	items := make([]dto.RepositoryResponse, len(repos))
	for i, r := range repos {
		items[i] = dto.NewRepositoryResponse(&r, teamID)
	}

	totalPages := total / perPage
	if total%perPage != 0 {
		totalPages++
	}

	return &dto.RepositoryListResponse{
		Repositories: items,
		Meta: dto.PaginationMeta{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *RepositoryService) GetRepository(ctx context.Context, userID, repoID uuid.UUID) (*dto.RepositoryResponse, error) {
	repo, err := s.repoRepo.GetByID(ctx, repoID)
	if err != nil {
		return nil, err
	}

	teamID, err := s.repoRepo.GetTeamIDForRepository(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("get team for repository: %w", err)
	}

	isMember, err := s.teamMemberRepo.IsTeamMember(ctx, teamID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotTeamMember
	}

	resp := dto.NewRepositoryResponse(repo, teamID.String())
	return &resp, nil
}

func (s *RepositoryService) UpdateRepository(ctx context.Context, userID, repoID uuid.UUID, req *dto.UpdateRepositoryRequest) (*dto.RepositoryResponse, error) {
	repo, err := s.repoRepo.GetByID(ctx, repoID)
	if err != nil {
		return nil, err
	}

	teamID, err := s.repoRepo.GetTeamIDForRepository(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("get team for repository: %w", err)
	}

	isMember, err := s.teamMemberRepo.IsTeamMember(ctx, teamID, userID)
	if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotTeamMember
	}

	if req.Description != nil {
		repo.Description = req.Description
	}

	if err := s.repoRepo.Update(ctx, repo); err != nil {
		return nil, fmt.Errorf("update repository: %w", err)
	}

	repo, err = s.repoRepo.GetByID(ctx, repoID)
	if err != nil {
		return nil, err
	}

	resp := dto.NewRepositoryResponse(repo, teamID.String())
	return &resp, nil
}

func (s *RepositoryService) SoftDeleteRepository(ctx context.Context, userID, repoID uuid.UUID) error {
	teamID, err := s.repoRepo.GetTeamIDForRepository(ctx, repoID)
	if err != nil {
		return fmt.Errorf("get team for repository: %w", err)
	}

	isAdmin, err := s.teamMemberRepo.IsTeamAdmin(ctx, teamID, userID)
	if err != nil {
		return fmt.Errorf("check admin: %w", err)
	}
	if !isAdmin {
		return ErrNotTeamAdmin
	}

	return s.repoRepo.SoftDelete(ctx, repoID)
}

func (s *RepositoryService) GetBranches(ctx context.Context, userID, repoID uuid.UUID) ([]dto.BranchResponse, error) {
	repo, err := s.repoRepo.GetByID(ctx, repoID)
	if err != nil {
		return nil, err
	}

	teamID, err := s.repoRepo.GetTeamIDForRepository(ctx, repoID)
	if err != nil {
		return nil, err
	}

	isMember, err := s.teamMemberRepo.IsTeamMember(ctx, teamID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotTeamMember
	}

	// Check Redis cache
	cacheKey := fmt.Sprintf("branches:%s", repoID.String())
	cached, err := s.rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		var branches []dto.BranchResponse
		if json.Unmarshal([]byte(cached), &branches) == nil {
			return branches, nil
		}
	}

	// Resolve PAT (team → service account fallback)
	pat, err := s.resolvePAT(ctx, teamID)
	if err != nil {
		return nil, err
	}

	// Use git ls-remote to list branches
	remoteURL := fmt.Sprintf("https://x-access-token:%s@github.com/%s.git", pat, repo.GithubFullName)
	cmdCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "git", "ls-remote", "--heads", remoteURL)
	cmd.Env = append(cmd.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-remote: %w", err)
	}

	var branches []dto.BranchResponse
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		sha := parts[0]
		ref := parts[1]
		name := strings.TrimPrefix(ref, "refs/heads/")
		branches = append(branches, dto.BranchResponse{
			Name:      name,
			CommitSHA: sha,
		})
	}

	if data, err := json.Marshal(branches); err == nil {
		s.rdb.Set(ctx, cacheKey, string(data), 5*time.Minute)
	}

	return branches, nil
}

func (s *RepositoryService) GetCommits(ctx context.Context, userID, repoID uuid.UUID, branch string) ([]dto.CommitResponse, error) {
	repo, err := s.repoRepo.GetByID(ctx, repoID)
	if err != nil {
		return nil, err
	}

	teamID, err := s.repoRepo.GetTeamIDForRepository(ctx, repoID)
	if err != nil {
		return nil, err
	}

	isMember, err := s.teamMemberRepo.IsTeamMember(ctx, teamID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotTeamMember
	}

	// Check Redis cache
	cacheKey := fmt.Sprintf("commits:%s:%s", repoID.String(), branch)
	cached, err := s.rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		var commits []dto.CommitResponse
		if json.Unmarshal([]byte(cached), &commits) == nil {
			return commits, nil
		}
	}

	// Resolve PAT (team → service account fallback)
	pat, err := s.resolvePAT(ctx, teamID)
	if err != nil {
		return nil, err
	}

	// Fetch commits from GitHub API
	owner, repoName, err := ParseGitHubURL("https://github.com/" + repo.GithubFullName)
	if err != nil {
		return nil, err
	}

	ghCommits, err := s.githubService.GetCommits(ctx, pat, owner, repoName, branch, 30)
	if err != nil {
		return nil, err
	}

	commits := make([]dto.CommitResponse, len(ghCommits))
	for i, c := range ghCommits {
		commits[i] = dto.CommitResponse{
			SHA:     c.SHA,
			Message: c.Message,
			Author:  c.Author,
			Date:    c.Date,
		}
	}

	if data, err := json.Marshal(commits); err == nil {
		s.rdb.Set(ctx, cacheKey, string(data), 2*time.Minute)
	}

	return commits, nil
}

// ResyncRepository syncs the fork with upstream and invalidates caches.
func (s *RepositoryService) ResyncRepository(ctx context.Context, userID, repoID uuid.UUID) (*dto.ResyncResponse, error) {
	repo, err := s.repoRepo.GetByID(ctx, repoID)
	if err != nil {
		return nil, err
	}

	teamID, err := s.repoRepo.GetTeamIDForRepository(ctx, repoID)
	if err != nil {
		return nil, err
	}

	isMember, err := s.teamMemberRepo.IsTeamMember(ctx, teamID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotTeamMember
	}

	// Re-push workflow on top of latest upstream HEAD (maintains single Verdox commit)
	if repo.ForkFullName != nil && *repo.ForkFullName != "" && s.forkService != nil {
		if err := s.forkService.ResyncFork(ctx, repoID, *repo.ForkFullName, repo.GithubFullName, repo.DefaultBranch); err != nil {
			s.log.Warn().Err(err).Str("fork", *repo.ForkFullName).Msg("fork resync failed")
		}
	}

	// Invalidate caches
	s.rdb.Del(ctx, fmt.Sprintf("branches:%s", repoID.String()))
	iter := s.rdb.Scan(ctx, 0, fmt.Sprintf("commits:%s:*", repoID.String()), 100).Iterator()
	for iter.Next(ctx) {
		s.rdb.Del(ctx, iter.Val())
	}

	branches, _ := s.GetBranches(ctx, userID, repoID)

	return &dto.ResyncResponse{
		Message:       "Repository resynchronized successfully",
		DefaultBranch: repo.DefaultBranch,
		BranchCount:   len(branches),
	}, nil
}
