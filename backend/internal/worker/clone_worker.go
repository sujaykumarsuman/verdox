package worker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"

	"github.com/sujaykumarsuman/verdox/backend/internal/config"
	"github.com/sujaykumarsuman/verdox/backend/internal/model"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
	"github.com/sujaykumarsuman/verdox/backend/internal/service"
	"github.com/sujaykumarsuman/verdox/backend/pkg/encryption"
)

type CloneWorker struct {
	repoRepo repository.RepositoryRepository
	teamRepo repository.TeamRepository
	cfg      *config.Config
	log      zerolog.Logger
}

func NewCloneWorker(
	repoRepo repository.RepositoryRepository,
	teamRepo repository.TeamRepository,
	cfg *config.Config,
	log zerolog.Logger,
) *CloneWorker {
	return &CloneWorker{
		repoRepo: repoRepo,
		teamRepo: teamRepo,
		cfg:      cfg,
		log:      log,
	}
}

// Start processes clone jobs from the channel until the context is cancelled.
func (w *CloneWorker) Start(ctx context.Context, jobs <-chan service.CloneJob) {
	w.log.Info().Msg("clone worker started")
	for {
		select {
		case <-ctx.Done():
			w.log.Info().Msg("clone worker shutting down")
			return
		case job, ok := <-jobs:
			if !ok {
				w.log.Info().Msg("clone worker: job channel closed")
				return
			}
			w.processJob(ctx, job)
		}
	}
}

func (w *CloneWorker) processJob(ctx context.Context, job service.CloneJob) {
	log := w.log.With().
		Str("repo_id", job.RepoID.String()).
		Str("team_id", job.TeamID.String()).
		Logger()

	log.Info().Msg("processing clone job")

	// Load repo
	repo, err := w.repoRepo.GetByID(ctx, job.RepoID)
	if err != nil {
		log.Error().Err(err).Msg("failed to load repository")
		return
	}

	// Load team and decrypt PAT
	team, err := w.teamRepo.GetByID(ctx, job.TeamID)
	if err != nil {
		log.Error().Err(err).Msg("failed to load team")
		_ = w.repoRepo.UpdateCloneStatus(ctx, job.RepoID, model.CloneStatusFailed)
		return
	}

	if !team.HasPAT() {
		log.Error().Msg("team has no PAT configured")
		_ = w.repoRepo.UpdateCloneStatus(ctx, job.RepoID, model.CloneStatusFailed)
		return
	}

	pat, err := encryption.Decrypt(*team.GithubPATEncrypted, team.GithubPATNonce, w.cfg.GithubTokenEncryptionKey)
	if err != nil {
		log.Error().Err(err).Msg("failed to decrypt PAT")
		_ = w.repoRepo.UpdateCloneStatus(ctx, job.RepoID, model.CloneStatusFailed)
		return
	}

	// Compute local path
	localPath := filepath.Join(w.cfg.RepoBasePath, "github.com", repo.GithubFullName)

	// Set status to cloning
	if err := w.repoRepo.UpdateCloneStatus(ctx, job.RepoID, model.CloneStatusCloning); err != nil {
		log.Error().Err(err).Msg("failed to update clone status")
		return
	}

	// Remove existing directory if present (retry after previous failure)
	if _, err := os.Stat(localPath); err == nil {
		if err := os.RemoveAll(localPath); err != nil {
			log.Error().Err(err).Msg("failed to remove existing directory")
			_ = w.repoRepo.UpdateCloneStatus(ctx, job.RepoID, model.CloneStatusFailed)
			return
		}
	}

	// Create parent directory
	parentDir := filepath.Dir(localPath)
	if err := os.MkdirAll(parentDir, 0700); err != nil {
		log.Error().Err(err).Msg("failed to create parent directory")
		_ = w.repoRepo.UpdateCloneStatus(ctx, job.RepoID, model.CloneStatusFailed)
		return
	}

	// Clone
	cloneURL := fmt.Sprintf("https://x-access-token:%s@github.com/%s.git", pat, repo.GithubFullName)
	cloneCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(cloneCtx, "git", "clone", "--depth", "1",
		"--branch", repo.DefaultBranch, cloneURL, localPath)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error().Err(err).Str("output", string(output)).Msg("git clone failed")
		_ = w.repoRepo.UpdateCloneStatus(ctx, job.RepoID, model.CloneStatusFailed)
		return
	}

	// Update repo with local path and ready status
	if err := w.repoRepo.UpdateCloneResult(ctx, job.RepoID, localPath, model.CloneStatusReady); err != nil {
		log.Error().Err(err).Msg("failed to update clone result")
		return
	}

	log.Info().Str("local_path", localPath).Msg("clone completed successfully")
}
