package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sujaykumarsuman/verdox/backend/internal/config"
	"github.com/sujaykumarsuman/verdox/backend/internal/dto"
	mw "github.com/sujaykumarsuman/verdox/backend/internal/middleware"
	"github.com/sujaykumarsuman/verdox/backend/internal/service"
	"github.com/sujaykumarsuman/verdox/backend/pkg/response"
	v "github.com/sujaykumarsuman/verdox/backend/pkg/validator"
)

type AuthHandler struct {
	authService *service.AuthService
	cfg         *config.Config
}

func NewAuthHandler(authService *service.AuthService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{authService: authService, cfg: cfg}
}

func (h *AuthHandler) Signup(c echo.Context) error {
	var req dto.SignupRequest
	if err := v.BindAndValidate(c, &req); err != nil {
		return err
	}

	resp, refreshToken, err := h.authService.Signup(c.Request().Context(), &req)
	if err != nil {
		if errors.Is(err, service.ErrEmailTaken) {
			return response.Error(c, http.StatusConflict, "CONFLICT", "Email already in use")
		}
		if errors.Is(err, service.ErrUsernameTaken) {
			return response.Error(c, http.StatusConflict, "CONFLICT", "Username already in use")
		}
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create account")
	}

	h.setRefreshCookie(c, refreshToken)
	h.setAccessCookie(c, resp.AccessToken)
	return response.Success(c, http.StatusCreated, resp)
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req dto.LoginRequest
	if err := v.BindAndValidate(c, &req); err != nil {
		return err
	}

	resp, refreshToken, err := h.authService.Login(c.Request().Context(), &req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			return response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid email/username or password")
		}
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to login")
	}

	h.setRefreshCookie(c, refreshToken)
	h.setAccessCookie(c, resp.AccessToken)
	return response.Success(c, http.StatusOK, resp)
}

func (h *AuthHandler) Refresh(c echo.Context) error {
	cookie, err := c.Cookie("verdox_refresh")
	if err != nil || cookie.Value == "" {
		return response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Missing refresh token")
	}

	resp, newRefreshToken, err := h.authService.Refresh(c.Request().Context(), cookie.Value)
	if err != nil {
		if errors.Is(err, service.ErrInvalidRefreshToken) {
			h.clearRefreshCookie(c)
			return response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired refresh token")
		}
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to refresh token")
	}

	h.setRefreshCookie(c, newRefreshToken)
	h.setAccessCookie(c, resp.AccessToken)
	return response.Success(c, http.StatusOK, resp)
}

func (h *AuthHandler) Logout(c echo.Context) error {
	userID := mw.GetUserID(c)
	if err := h.authService.Logout(c.Request().Context(), userID); err != nil {
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to logout")
	}

	h.clearRefreshCookie(c)
	h.clearAccessCookie(c)
	return c.NoContent(http.StatusNoContent)
}

func (h *AuthHandler) ForgotPassword(c echo.Context) error {
	var req dto.ForgotPasswordRequest
	if err := v.BindAndValidate(c, &req); err != nil {
		return err
	}

	_ = h.authService.ForgotPassword(c.Request().Context(), &req)

	// Always return success to prevent email enumeration
	return response.Success(c, http.StatusOK, dto.MessageResponse{
		Message: "If an account with that email exists, a password reset link has been sent.",
	})
}

func (h *AuthHandler) ResetPassword(c echo.Context) error {
	var req dto.ResetPasswordRequest
	if err := v.BindAndValidate(c, &req); err != nil {
		return err
	}

	if err := h.authService.ResetPassword(c.Request().Context(), &req); err != nil {
		if errors.Is(err, service.ErrInvalidResetToken) {
			return response.Error(c, http.StatusBadRequest, "INVALID_TOKEN", "Invalid or expired reset token")
		}
		return response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to reset password")
	}

	return response.Success(c, http.StatusOK, dto.MessageResponse{
		Message: "Password has been reset successfully. Please log in with your new password.",
	})
}

func (h *AuthHandler) setRefreshCookie(c echo.Context, token string) {
	c.SetCookie(&http.Cookie{
		Name:     "verdox_refresh",
		Value:    token,
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   h.cfg.IsProduction(),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   604800, // 7 days
	})
}

func (h *AuthHandler) clearRefreshCookie(c echo.Context) {
	c.SetCookie(&http.Cookie{
		Name:     "verdox_refresh",
		Value:    "",
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   h.cfg.IsProduction(),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}

func (h *AuthHandler) setAccessCookie(c echo.Context, token string) {
	c.SetCookie(&http.Cookie{
		Name:     "verdox_access",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.IsProduction(),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(15 * time.Minute / time.Second),
	})
}

func (h *AuthHandler) clearAccessCookie(c echo.Context) {
	c.SetCookie(&http.Cookie{
		Name:     "verdox_access",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.IsProduction(),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}
