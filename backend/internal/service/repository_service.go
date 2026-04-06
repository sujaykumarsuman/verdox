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
	ErrPATNotConfigured = errors.New("team does not have a GitHub PAT configured")
	ErrDuplicateRepo    = errors.New("repository already added")
	ErrCloneNotReady    = errors.New("repository clone is not ready")
	ErrNotTeamMember    = errors.New("user is not a member of this team")
	ErrNotTeamAdmin     = errors.New("user is not an admin of this team")
)

// CloneJob represents a repository clone task for the worker.
type CloneJob struct {
	RepoID uuid.UUID
	TeamID uuid.UUID
}

type RepositoryService struct {
	repoRepo       repository.RepositoryRepository
	teamRepo       repository.TeamRepository
	teamMemberRepo repository.TeamMemberRepository
	githubService  *GitHubService
	rdb            *redis.Client
	cfg            *config.Config
	log            zerolog.Logger
	cloneCh        chan<- CloneJob
}

func NewRepositoryService(
	repoRepo repository.RepositoryRepository,
	teamRepo repository.TeamRepository,
	teamMemberRepo repository.TeamMemberRepository,
	githubService *GitHubService,
	rdb *redis.Client,
	cfg *config.Config,
	log zerolog.Logger,
	cloneCh chan<- CloneJob,
) *RepositoryService {
	return &RepositoryService{
		repoRepo:       repoRepo,
		teamRepo:       teamRepo,
		teamMemberRepo: teamMemberRepo,
		githubService:  githubService,
		rdb:            rdb,
		cfg:            cfg,
		log:            log,
		cloneCh:        cloneCh,
	}
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

	// Load team and verify PAT
	team, err := s.teamRepo.GetByID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("get team: %w", err)
	}
	if !team.HasPAT() {
		return nil, ErrPATNotConfigured
	}

	// Decrypt PAT
	pat, err := encryption.Decrypt(*team.GithubPATEncrypted, team.GithubPATNonce, s.cfg.GithubTokenEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt pat: %w", err)
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
		// Reactivate the soft-deleted repo
		existing.IsActive = true
		existing.CloneStatus = model.CloneStatusPending
		existing.Description = &ghInfo.Description
		existing.DefaultBranch = ghInfo.DefaultBranch
		if err := s.repoRepo.Reactivate(ctx, existing); err != nil {
			return nil, fmt.Errorf("reactivate repository: %w", err)
		}
		s.cloneCh <- CloneJob{RepoID: existing.ID, TeamID: teamID}
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
		CloneStatus:    model.CloneStatusPending,
		IsActive:       true,
	}
	if err := s.repoRepo.Create(ctx, repo); err != nil {
		return nil, fmt.Errorf("create repository: %w", err)
	}

	// Create junction row
	if err := s.repoRepo.AddTeamRepository(ctx, teamID, repo.ID, userID); err != nil {
		return nil, fmt.Errorf("add team repository: %w", err)
	}

	// Enqueue clone job
	s.cloneCh <- CloneJob{RepoID: repo.ID, TeamID: teamID}

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

	// Re-fetch to get updated_at
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

func (s *RepositoryService) RetryClone(ctx context.Context, userID, repoID uuid.UUID) (*dto.RepositoryResponse, error) {
	repo, err := s.repoRepo.GetByID(ctx, repoID)
	if err != nil {
		return nil, err
	}

	if repo.CloneStatus != model.CloneStatusFailed {
		return nil, fmt.Errorf("can only retry failed clones")
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

	// Reset status and re-enqueue
	if err := s.repoRepo.UpdateCloneStatus(ctx, repoID, model.CloneStatusPending); err != nil {
		return nil, fmt.Errorf("reset clone status: %w", err)
	}

	s.cloneCh <- CloneJob{RepoID: repoID, TeamID: teamID}

	repo.CloneStatus = model.CloneStatusPending
	resp := dto.NewRepositoryResponse(repo, teamID.String())
	return &resp, nil
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

	if repo.CloneStatus != model.CloneStatusReady {
		return nil, ErrCloneNotReady
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

	// Get team PAT for ls-remote
	team, err := s.teamRepo.GetByID(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if !team.HasPAT() {
		return nil, ErrPATNotConfigured
	}

	pat, err := encryption.Decrypt(*team.GithubPATEncrypted, team.GithubPATNonce, s.cfg.GithubTokenEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt pat: %w", err)
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

	// Cache result
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

	// Get team PAT
	team, err := s.teamRepo.GetByID(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if !team.HasPAT() {
		return nil, ErrPATNotConfigured
	}

	pat, err := encryption.Decrypt(*team.GithubPATEncrypted, team.GithubPATNonce, s.cfg.GithubTokenEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt pat: %w", err)
	}

	// Fetch commits from GitHub API (shallow clones have no history)
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

	// Cache result
	if data, err := json.Marshal(commits); err == nil {
		s.rdb.Set(ctx, cacheKey, string(data), 2*time.Minute)
	}

	return commits, nil
}

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

	if repo.CloneStatus != model.CloneStatusReady || repo.LocalPath == nil {
		return nil, ErrCloneNotReady
	}

	// Run git fetch
	cmdCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "git", "-C", *repo.LocalPath, "fetch", "--all", "--prune")
	cmd.Env = append(cmd.Environ(), "GIT_TERMINAL_PROMPT=0")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git fetch: %w", err)
	}

	// Invalidate caches
	s.rdb.Del(ctx, fmt.Sprintf("branches:%s", repoID.String()))
	// Delete all commit caches for this repo
	iter := s.rdb.Scan(ctx, 0, fmt.Sprintf("commits:%s:*", repoID.String()), 100).Iterator()
	for iter.Next(ctx) {
		s.rdb.Del(ctx, iter.Val())
	}

	// Get branch count from ls-remote
	branches, _ := s.GetBranches(ctx, userID, repoID)

	return &dto.ResyncResponse{
		Message:       "Repository resynchronized successfully",
		DefaultBranch: repo.DefaultBranch,
		BranchCount:   len(branches),
	}, nil
}
