package middleware

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
	"github.com/sujaykumarsuman/verdox/backend/pkg/response"
)

const ContextKeyTeamRole contextKey = "team_role"

// RequireTeamRole returns middleware that enforces team membership and role.
// Root users bypass all checks. The team ID is read from the ":id" route param.
func RequireTeamRole(teamMemberRepo repository.TeamMemberRepository, roles ...string) echo.MiddlewareFunc {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Root bypasses all team role checks.
			if GetUserRole(c) == "root" {
				c.Set(string(ContextKeyTeamRole), "admin")
				return next(c)
			}

			teamID, err := uuid.Parse(c.Param("id"))
			if err != nil {
				return response.Error(c, http.StatusBadRequest, "INVALID_ID", "Invalid team ID")
			}

			userID := GetUserID(c)
			member, err := teamMemberRepo.GetByTeamAndUser(c.Request().Context(), teamID, userID)
			if err != nil {
				return response.Error(c, http.StatusForbidden, "FORBIDDEN", "Not a member of this team")
			}

			if member.Status != "approved" {
				return response.Error(c, http.StatusForbidden, "FORBIDDEN", "Membership not approved")
			}

			if !allowed[string(member.Role)] {
				return response.Error(c, http.StatusForbidden, "FORBIDDEN", "Insufficient team role")
			}

			c.Set(string(ContextKeyTeamRole), string(member.Role))
			return next(c)
		}
	}
}

// GetTeamRole returns the team role set by RequireTeamRole middleware.
func GetTeamRole(c echo.Context) string {
	if role, ok := c.Get(string(ContextKeyTeamRole)).(string); ok {
		return role
	}
	return ""
}
