package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	mw "github.com/sujaykumarsuman/verdox/backend/internal/middleware"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
	"github.com/sujaykumarsuman/verdox/backend/internal/service"
	"github.com/sujaykumarsuman/verdox/backend/pkg/response"

	"github.com/sujaykumarsuman/verdox/backend/internal/dto"
	v "github.com/sujaykumarsuman/verdox/backend/pkg/validator"
)

type RepositoryHandler struct {
	repoService *service.RepositoryService
}

func NewRepositoryHandler(repoService *service.RepositoryService) *RepositoryHandler {
	return &RepositoryHandler{repoService: repoService}
}

// Create handles POST /v1/repositories
func (h *RepositoryHandler) Create(c echo.Context) error {
	var req dto.AddRepositoryRequest
	if err := v.BindAndValidate(c, &req); err != nil {
		return err
	}

	userID := mw.GetUserID(c)
	resp, err := h.repoService.AddRepository(c.Request().Context(), userID, &req)
	if err != nil {
		return h.mapError(c, err)
	}

	return response.Success(c, http.StatusCreated, resp)
}

// List handles GET /v1/repositories
func (h *RepositoryHandler) List(c echo.Context) error {
	teamID := c.QueryParam("team_id")
	search := c.QueryParam("search")
	page := queryInt(c, "page", 1)
	perPage := queryInt(c, "per_page", 20)
	if perPage > 100 {
		perPage = 100
	}

	userID := mw.GetUserID(c)

	var resp *dto.RepositoryListResponse
	var err error

	if teamID == "" {
		// No team_id: list all repos the user has access to
		resp, err = h.repoService.ListAllRepositories(c.Request().Context(), userID, search, page, perPage)
	} else {
		resp, err = h.repoService.ListRepositories(c.Request().Context(), userID, teamID, search, page, perPage)
	}

	if err != nil {
		return h.mapError(c, err)
	}

	return response.Success(c, http.StatusOK, resp)
}

// Get handles GET /v1/repositories/:id
func (h *RepositoryHandler) Get(c echo.Context) error {
	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid repository ID")
	}

	userID := mw.GetUserID(c)
	resp, err := h.repoService.GetRepository(c.Request().Context(), userID, repoID)
	if err != nil {
		return h.mapError(c, err)
	}

	return response.Success(c, http.StatusOK, resp)
}

// Update handles PUT /v1/repositories/:id
func (h *RepositoryHandler) Update(c echo.Context) error {
	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid repository ID")
	}

	var req dto.UpdateRepositoryRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	userID := mw.GetUserID(c)
	resp, err := h.repoService.UpdateRepository(c.Request().Context(), userID, repoID, &req)
	if err != nil {
		return h.mapError(c, err)
	}

	return response.Success(c, http.StatusOK, resp)
}

// Delete handles DELETE /v1/repositories/:id
func (h *RepositoryHandler) Delete(c echo.Context) error {
	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid repository ID")
	}

	userID := mw.GetUserID(c)
	if err := h.repoService.SoftDeleteRepository(c.Request().Context(), userID, repoID); err != nil {
		return h.mapError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// ListBranches handles GET /v1/repositories/:id/branches
func (h *RepositoryHandler) ListBranches(c echo.Context) error {
	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid repository ID")
	}

	userID := mw.GetUserID(c)
	branches, err := h.repoService.GetBranches(c.Request().Context(), userID, repoID)
	if err != nil {
		return h.mapError(c, err)
	}

	return response.Success(c, http.StatusOK, branches)
}

// ListCommits handles GET /v1/repositories/:id/commits
func (h *RepositoryHandler) ListCommits(c echo.Context) error {
	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid repository ID")
	}

	branch := c.QueryParam("branch")
	if branch == "" {
		return response.Error(c, http.StatusBadRequest, "MISSING_PARAM", "branch query parameter is required")
	}

	userID := mw.GetUserID(c)
	commits, err := h.repoService.GetCommits(c.Request().Context(), userID, repoID, branch)
	if err != nil {
		return h.mapError(c, err)
	}

	return response.Success(c, http.StatusOK, commits)
}

// Resync handles POST /v1/repositories/:id/resync
func (h *RepositoryHandler) Resync(c echo.Context) error {
	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid repository ID")
	}

	userID := mw.GetUserID(c)
	resp, err := h.repoService.ResyncRepository(c.Request().Context(), userID, repoID)
	if err != nil {
		return h.mapError(c, err)
	}

	return response.Success(c, http.StatusOK, resp)
}

func (h *RepositoryHandler) mapError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return response.Error(c, http.StatusNotFound, "NOT_FOUND", "Resource not found")
	case errors.Is(err, service.ErrInvalidPAT):
		return response.Error(c, http.StatusUnprocessableEntity, "INVALID_PAT", "GitHub PAT is invalid")
	case errors.Is(err, service.ErrRepoNotFound):
		return response.Error(c, http.StatusNotFound, "REPO_NOT_FOUND", "GitHub repository not found")
	case errors.Is(err, service.ErrNoRepoAccess):
		return response.Error(c, http.StatusForbidden, "NO_ACCESS", "PAT does not have access to this repository. Grant the service account access, or add a team PAT with access to this repo.")
	case errors.Is(err, service.ErrDuplicateRepo):
		return response.Error(c, http.StatusConflict, "DUPLICATE", "Repository already added")
	case errors.Is(err, service.ErrPATNotConfigured):
		return response.Error(c, http.StatusUnprocessableEntity, "PAT_NOT_CONFIGURED", "Team does not have a GitHub PAT configured")
	case errors.Is(err, service.ErrForkNotReady):
		return response.Error(c, http.StatusUnprocessableEntity, "FORK_NOT_READY", "Repository fork is not ready")
	case errors.Is(err, service.ErrNotTeamMember):
		return response.Error(c, http.StatusForbidden, "FORBIDDEN", "Not a member of this team")
	case errors.Is(err, service.ErrNotTeamAdmin):
		return response.Error(c, http.StatusForbidden, "FORBIDDEN", "Team admin role required")
	default:
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
	}
}

func queryInt(c echo.Context, key string, defaultVal int) int {
	s := c.QueryParam(key)
	if s == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return defaultVal
	}
	return n
}
