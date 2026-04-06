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

type TestSuiteHandler struct {
	suiteService *service.TestSuiteService
}

func NewTestSuiteHandler(suiteService *service.TestSuiteService) *TestSuiteHandler {
	return &TestSuiteHandler{suiteService: suiteService}
}

// Create handles POST /v1/repositories/:id/suites
func (h *TestSuiteHandler) Create(c echo.Context) error {
	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid repository ID")
	}

	var req dto.CreateTestSuiteRequest
	if err := v.BindAndValidate(c, &req); err != nil {
		return err
	}

	userID := mw.GetUserID(c)
	resp, err := h.suiteService.CreateSuite(c.Request().Context(), userID, repoID, &req)
	if err != nil {
		return h.mapError(c, err)
	}

	return response.Success(c, http.StatusCreated, resp)
}

// List handles GET /v1/repositories/:id/suites
func (h *TestSuiteHandler) List(c echo.Context) error {
	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid repository ID")
	}

	userID := mw.GetUserID(c)
	resp, err := h.suiteService.ListSuites(c.Request().Context(), userID, repoID)
	if err != nil {
		return h.mapError(c, err)
	}

	return response.Success(c, http.StatusOK, resp)
}

// Update handles PUT /v1/suites/:id
func (h *TestSuiteHandler) Update(c echo.Context) error {
	suiteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid suite ID")
	}

	var req dto.UpdateTestSuiteRequest
	if err := v.BindAndValidate(c, &req); err != nil {
		return err
	}

	userID := mw.GetUserID(c)
	resp, err := h.suiteService.UpdateSuite(c.Request().Context(), userID, suiteID, &req)
	if err != nil {
		return h.mapError(c, err)
	}

	return response.Success(c, http.StatusOK, resp)
}

// Delete handles DELETE /v1/suites/:id
func (h *TestSuiteHandler) Delete(c echo.Context) error {
	suiteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid suite ID")
	}

	userID := mw.GetUserID(c)
	if err := h.suiteService.DeleteSuite(c.Request().Context(), userID, suiteID); err != nil {
		return h.mapError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *TestSuiteHandler) mapError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return response.Error(c, http.StatusNotFound, "NOT_FOUND", "Resource not found")
	case errors.Is(err, service.ErrSuiteNotFound):
		return response.Error(c, http.StatusNotFound, "NOT_FOUND", "Test suite not found")
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
