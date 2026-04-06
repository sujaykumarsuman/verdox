package handler

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sujaykumarsuman/verdox/backend/internal/dto"
	mw "github.com/sujaykumarsuman/verdox/backend/internal/middleware"
	"github.com/sujaykumarsuman/verdox/backend/internal/service"
	"github.com/sujaykumarsuman/verdox/backend/pkg/response"
)

type AdminHandler struct {
	adminService *service.AdminService
}

func NewAdminHandler(adminService *service.AdminService) *AdminHandler {
	return &AdminHandler{adminService: adminService}
}

func (h *AdminHandler) ListUsers(c echo.Context) error {
	var req dto.AdminUserListRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid query parameters")
	}

	result, err := h.adminService.ListUsers(c.Request().Context(), &req)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list users")
	}

	return response.Success(c, http.StatusOK, result)
}

func (h *AdminHandler) UpdateUser(c echo.Context) error {
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
	}

	var req dto.UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	if req.Role == nil && req.IsActive == nil && req.IsBanned == nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "No fields to update")
	}

	callerID := mw.GetUserID(c)
	callerRole := mw.GetUserRole(c)

	err = h.adminService.UpdateUser(c.Request().Context(), callerID, callerRole, targetID, &req)
	if err != nil {
		return h.mapError(c, err)
	}

	return response.Success(c, http.StatusOK, dto.MessageResponse{Message: "User updated successfully"})
}

func (h *AdminHandler) GetStats(c echo.Context) error {
	stats, err := h.adminService.GetStats(c.Request().Context())
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get stats")
	}

	return response.Success(c, http.StatusOK, stats)
}

func (h *AdminHandler) ListPendingBanReviews(c echo.Context) error {
	result, err := h.adminService.ListPendingBanReviews(c.Request().Context())
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list ban reviews")
	}
	return response.Success(c, http.StatusOK, result)
}

func (h *AdminHandler) ReviewBan(c echo.Context) error {
	reviewID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid review ID")
	}

	var req dto.ReviewBanDecisionRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	if req.Status != "approved" && req.Status != "denied" {
		return response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "Status must be 'approved' or 'denied'")
	}

	callerID := mw.GetUserID(c)
	err = h.adminService.ReviewBan(c.Request().Context(), callerID, reviewID, req.Status)
	if err != nil {
		if errors.Is(err, service.ErrReviewNotFound) {
			return response.Error(c, http.StatusNotFound, "NOT_FOUND", "Ban review not found")
		}
		if errors.Is(err, service.ErrReviewAlreadyProcessed) {
			return response.Error(c, http.StatusConflict, "ALREADY_PROCESSED", "This review has already been processed")
		}
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to process review")
	}

	action := "denied"
	if req.Status == "approved" {
		action = "approved and user unbanned"
	}
	return response.Success(c, http.StatusOK, dto.MessageResponse{Message: "Ban review " + action})
}

func (h *AdminHandler) GetUserTeams(c echo.Context) error {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
	}

	teams, err := h.adminService.GetUserTeams(c.Request().Context(), userID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get user teams")
	}

	return response.Success(c, http.StatusOK, teams)
}

func (h *AdminHandler) UpdateUserTeams(c echo.Context) error {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
	}

	var req dto.UpdateUserTeamsRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	teamIDs := make([]uuid.UUID, 0, len(req.TeamIDs))
	for _, idStr := range req.TeamIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid team ID: "+idStr)
		}
		teamIDs = append(teamIDs, id)
	}

	callerID := mw.GetUserID(c)
	err = h.adminService.UpdateUserTeams(c.Request().Context(), callerID, userID, teamIDs)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update user teams")
	}

	return response.Success(c, http.StatusOK, dto.MessageResponse{Message: "User teams updated"})
}

func (h *AdminHandler) ListAllTeams(c echo.Context) error {
	teams, err := h.adminService.ListAllTeams(c.Request().Context())
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list teams")
	}

	return response.Success(c, http.StatusOK, teams)
}

func (h *AdminHandler) mapError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, service.ErrSelfDemotion):
		return response.Error(c, http.StatusBadRequest, "SELF_DEMOTION", err.Error())
	case errors.Is(err, service.ErrSelfDeactivation):
		return response.Error(c, http.StatusBadRequest, "SELF_DEACTIVATION", err.Error())
	case errors.Is(err, service.ErrLastRoot):
		return response.Error(c, http.StatusConflict, "LAST_ROOT", err.Error())
	case errors.Is(err, service.ErrModeratorCannotChangeRoles):
		return response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error())
	case errors.Is(err, service.ErrModeratorCannotDeactivateRoot):
		return response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error())
	case errors.Is(err, service.ErrCannotAssignRoot):
		return response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error())
	case errors.Is(err, service.ErrSelfBan):
		return response.Error(c, http.StatusBadRequest, "SELF_BAN", err.Error())
	case errors.Is(err, service.ErrCannotBanRoot):
		return response.Error(c, http.StatusForbidden, "FORBIDDEN", err.Error())
	case errors.Is(err, service.ErrBanReasonRequired):
		return response.Error(c, http.StatusBadRequest, "BAN_REASON_REQUIRED", err.Error())
	default:
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update user")
	}
}
