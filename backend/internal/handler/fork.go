package handler

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sujaykumarsuman/verdox/backend/internal/dto"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
	"github.com/sujaykumarsuman/verdox/backend/internal/service"
	"github.com/sujaykumarsuman/verdox/backend/pkg/response"
)

type ForkHandler struct {
	forkService *service.ForkService
	repoRepo    repository.RepositoryRepository
}

func NewForkHandler(forkService *service.ForkService, repoRepo repository.RepositoryRepository) *ForkHandler {
	return &ForkHandler{forkService: forkService, repoRepo: repoRepo}
}

// SetupFork initiates forking for a repository.
func (h *ForkHandler) SetupFork(c echo.Context) error {
	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid repository ID")
	}

	if !h.forkService.IsConfigured() {
		return response.Error(c, http.StatusServiceUnavailable, "NOT_CONFIGURED", "Service account is not configured for fork-based testing")
	}

	repo, err := h.repoRepo.GetByID(c.Request().Context(), repoID)
	if err != nil {
		return response.Error(c, http.StatusNotFound, "NOT_FOUND", "Repository not found")
	}

	// Run fork setup asynchronously (use Background context — request context is cancelled on response)
	go func() {
		_ = h.forkService.SetupFork(context.Background(), repoID, repo.GithubFullName, repo.DefaultBranch)
	}()

	return response.Success(c, http.StatusAccepted, dto.MessageResponse{
		Message: "Fork setup initiated. This may take a moment.",
	})
}

// SyncFork syncs the fork with upstream by re-pushing the Verdox workflow
// on top of the latest upstream HEAD (maintains a single Verdox commit).
func (h *ForkHandler) SyncFork(c echo.Context) error {
	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid repository ID")
	}

	repo, err := h.repoRepo.GetByID(c.Request().Context(), repoID)
	if err != nil {
		return response.Error(c, http.StatusNotFound, "NOT_FOUND", "Repository not found")
	}

	if repo.ForkFullName == nil || *repo.ForkFullName == "" {
		return response.Error(c, http.StatusBadRequest, "NO_FORK", "Repository has not been forked yet")
	}

	if err := h.forkService.ResyncFork(c.Request().Context(), repoID, *repo.ForkFullName, repo.GithubFullName, repo.DefaultBranch); err != nil {
		return response.Error(c, http.StatusInternalServerError, "SYNC_FAILED", "Failed to sync fork: "+err.Error())
	}

	return response.Success(c, http.StatusOK, dto.MessageResponse{Message: "Fork synced with upstream"})
}

// GetForkStatus returns the fork status for a repository.
func (h *ForkHandler) GetForkStatus(c echo.Context) error {
	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid repository ID")
	}

	repo, err := h.repoRepo.GetByID(c.Request().Context(), repoID)
	if err != nil {
		return response.Error(c, http.StatusNotFound, "NOT_FOUND", "Repository not found")
	}

	result := map[string]interface{}{
		"fork_status":      repo.ForkStatus,
		"fork_full_name":   repo.ForkFullName,
		"fork_synced_at":   repo.ForkSyncedAt,
		"fork_workflow_id": repo.ForkWorkflowID,
	}

	return response.Success(c, http.StatusOK, result)
}
