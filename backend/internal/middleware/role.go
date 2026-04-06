package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sujaykumarsuman/verdox/backend/pkg/response"
)

// RequireRole returns middleware that enforces the user's global role.
// The role is read from the echo context (set by Auth middleware).
func RequireRole(roles ...string) echo.MiddlewareFunc {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role := GetUserRole(c)
			if !allowed[role] {
				return response.Error(c, http.StatusForbidden, "FORBIDDEN", "Insufficient permissions")
			}
			return next(c)
		}
	}
}
