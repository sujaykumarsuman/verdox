package handler

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	mw "github.com/sujaykumarsuman/verdox/backend/internal/middleware"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
	"github.com/sujaykumarsuman/verdox/backend/internal/service"
	"github.com/sujaykumarsuman/verdox/backend/pkg/response"

	"github.com/sujaykumarsuman/verdox/backend/internal/dto"
	v "github.com/sujaykumarsuman/verdox/backend/pkg/validator"
)

type TestRunHandler struct {
	runService *service.TestRunService
}

func NewTestRunHandler(runService *service.TestRunService) *TestRunHandler {
	return &TestRunHandler{runService: runService}
}

// Trigger handles POST /v1/suites/:id/run
func (h *TestRunHandler) Trigger(c echo.Context) error {
	suiteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid suite ID")
	}

	var req dto.TriggerRunRequest
	if err := v.BindAndValidate(c, &req); err != nil {
		return err
	}

	userID := mw.GetUserID(c)
	resp, err := h.runService.TriggerRun(c.Request().Context(), userID, suiteID, &req)
	if err != nil {
		return h.mapError(c, err)
	}

	return response.Success(c, http.StatusCreated, resp)
}

// ListBySuite handles GET /v1/suites/:id/runs
func (h *TestRunHandler) ListBySuite(c echo.Context) error {
	suiteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid suite ID")
	}

	page := queryInt(c, "page", 1)
	perPage := queryInt(c, "per_page", 20)
	if perPage > 100 {
		perPage = 100
	}
	status := c.QueryParam("status")
	branch := c.QueryParam("branch")

	userID := mw.GetUserID(c)
	resp, err := h.runService.ListRunsBySuite(c.Request().Context(), userID, suiteID, status, branch, page, perPage)
	if err != nil {
		return h.mapError(c, err)
	}

	return response.Success(c, http.StatusOK, resp)
}

// Get handles GET /v1/runs/:id
func (h *TestRunHandler) Get(c echo.Context) error {
	runID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid run ID")
	}

	userID := mw.GetUserID(c)
	resp, err := h.runService.GetRun(c.Request().Context(), userID, runID)
	if err != nil {
		return h.mapError(c, err)
	}

	return response.Success(c, http.StatusOK, resp)
}

// Logs handles GET /v1/runs/:id/logs
func (h *TestRunHandler) Logs(c echo.Context) error {
	runID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid run ID")
	}

	testName := c.QueryParam("test_name")

	userID := mw.GetUserID(c)
	resp, err := h.runService.GetRunLogs(c.Request().Context(), userID, runID, testName)
	if err != nil {
		return h.mapError(c, err)
	}

	return response.Success(c, http.StatusOK, resp)
}

// Cancel handles POST /v1/runs/:id/cancel
func (h *TestRunHandler) Cancel(c echo.Context) error {
	runID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid run ID")
	}

	userID := mw.GetUserID(c)
	resp, err := h.runService.CancelRun(c.Request().Context(), userID, runID)
	if err != nil {
		return h.mapError(c, err)
	}

	return response.Success(c, http.StatusOK, resp)
}

// Rerun handles POST /v1/runs/:id/rerun
func (h *TestRunHandler) Rerun(c echo.Context) error {
	runID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid run ID")
	}

	userID := mw.GetUserID(c)
	resp, err := h.runService.RerunRun(c.Request().Context(), userID, runID)
	if err != nil {
		return h.mapError(c, err)
	}

	return response.Success(c, http.StatusCreated, resp)
}

// RunAll handles POST /v1/repositories/:id/run-all
func (h *TestRunHandler) RunAll(c echo.Context) error {
	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid repository ID")
	}

	var req dto.RunAllRequest
	if err := v.BindAndValidate(c, &req); err != nil {
		return err
	}

	userID := mw.GetUserID(c)
	resp, err := h.runService.RunAll(c.Request().Context(), userID, repoID, &req)
	if err != nil {
		return h.mapError(c, err)
	}

	return response.Success(c, http.StatusCreated, resp)
}

func (h *TestRunHandler) mapError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return response.Error(c, http.StatusNotFound, "NOT_FOUND", "Resource not found")
	case errors.Is(err, service.ErrRunNotFound):
		return response.Error(c, http.StatusNotFound, "NOT_FOUND", "Test run not found")
	case errors.Is(err, service.ErrSuiteNotFound):
		return response.Error(c, http.StatusNotFound, "NOT_FOUND", "Test suite not found")
	case errors.Is(err, service.ErrRunConflict):
		return response.Error(c, http.StatusConflict, "CONFLICT", "A run for this commit is already queued or running")
	case errors.Is(err, service.ErrRunNotCancellable):
		return response.Error(c, http.StatusConflict, "CONFLICT", "Run is already in a terminal state")
	case errors.Is(err, service.ErrRunNotRerunnable):
		return response.Error(c, http.StatusConflict, "CONFLICT", "Run is not in a failed state or has no GHA run ID")
	case errors.Is(err, service.ErrForkNotReady):
		return response.Error(c, http.StatusUnprocessableEntity, "FORK_NOT_READY", "Repository fork is not ready")
	case errors.Is(err, service.ErrNotTeamMember):
		return response.Error(c, http.StatusForbidden, "FORBIDDEN", "Not a member of this team")
	case errors.Is(err, service.ErrNotTeamAdmin):
		return response.Error(c, http.StatusForbidden, "FORBIDDEN", "Team admin role required")
	case errors.Is(err, service.ErrNotAdminOrMaintainer):
		return response.Error(c, http.StatusForbidden, "FORBIDDEN", "Admin or maintainer role required")
	default:
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
	}
}
