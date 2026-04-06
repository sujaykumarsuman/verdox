package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sujaykumarsuman/verdox/backend/internal/dto"
	mw "github.com/sujaykumarsuman/verdox/backend/internal/middleware"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
	"github.com/sujaykumarsuman/verdox/backend/internal/service"
	"github.com/sujaykumarsuman/verdox/backend/pkg/response"
	v "github.com/sujaykumarsuman/verdox/backend/pkg/validator"
)

type UserHandler struct {
	authService *service.AuthService
	userRepo    repository.UserRepository
}

func NewUserHandler(authService *service.AuthService, userRepo repository.UserRepository) *UserHandler {
	return &UserHandler{authService: authService, userRepo: userRepo}
}

func (h *UserHandler) GetProfile(c echo.Context) error {
	userID := mw.GetUserID(c)

	user, err := h.userRepo.GetByID(c.Request().Context(), userID)
	if err != nil {
		return response.Error(c, http.StatusNotFound, "NOT_FOUND", "User not found")
	}

	return response.Success(c, http.StatusOK, dto.NewUserResponse(user))
}

func (h *UserHandler) UpdateProfile(c echo.Context) error {
	var req dto.UpdateProfileRequest
	if err := v.BindAndValidate(c, &req); err != nil {
		return err
	}

	userID := mw.GetUserID(c)
	user, err := h.authService.UpdateProfile(c.Request().Context(), userID, &req)
	if err != nil {
		if errors.Is(err, service.ErrEmailTaken) {
			return response.Error(c, http.StatusConflict, "CONFLICT", "Email already in use")
		}
		if errors.Is(err, service.ErrUsernameTaken) {
			return response.Error(c, http.StatusConflict, "CONFLICT", "Username already in use")
		}
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update profile")
	}

	return response.Success(c, http.StatusOK, dto.NewUserResponse(user))
}

func (h *UserHandler) ChangePassword(c echo.Context) error {
	var req dto.ChangePasswordRequest
	if err := v.BindAndValidate(c, &req); err != nil {
		return err
	}

	userID := mw.GetUserID(c)
	err := h.authService.ChangePassword(c.Request().Context(), userID, &req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCurrentPassword) {
			return response.Error(c, http.StatusBadRequest, "INVALID_PASSWORD", "Current password is incorrect")
		}
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to change password")
	}

	return response.Success(c, http.StatusOK, dto.MessageResponse{Message: "Password changed successfully"})
}
