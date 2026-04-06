package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
	"github.com/sujaykumarsuman/verdox/backend/pkg/jwt"
	"github.com/sujaykumarsuman/verdox/backend/pkg/response"
)

type contextKey string

const (
	ContextKeyUserID   contextKey = "user_id"
	ContextKeyUsername contextKey = "username"
	ContextKeyUserRole contextKey = "user_role"
)

func Auth(jwtSecret string, userRepo repository.UserRepository, rdb *redis.Client) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Extract token: cookie first, then Authorization header
			tokenStr := ""

			if cookie, err := c.Cookie("verdox_access"); err == nil {
				tokenStr = cookie.Value
			}

			if tokenStr == "" {
				auth := c.Request().Header.Get("Authorization")
				if strings.HasPrefix(auth, "Bearer ") {
					tokenStr = strings.TrimPrefix(auth, "Bearer ")
				}
			}

			if tokenStr == "" {
				return response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication token")
			}

			// Validate JWT
			claims, err := jwt.ValidateAccessToken(jwtSecret, tokenStr)
			if err != nil {
				return response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired token")
			}

			userID, err := uuid.Parse(claims.Subject)
			if err != nil {
				return response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid token claims")
			}

			// Check session exists in Redis (fast path)
			ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
			defer cancel()

			sessionKey := fmt.Sprintf("session:%s", userID.String())
			exists, _ := rdb.Exists(ctx, sessionKey).Result()

			if exists == 0 {
				// Fallback: check DB
				if _, err := userRepo.GetByID(ctx, userID); err != nil {
					return response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Session not found")
				}
			}

			// Set context values
			c.Set(string(ContextKeyUserID), userID)
			c.Set(string(ContextKeyUsername), claims.Username)
			c.Set(string(ContextKeyUserRole), claims.Role)

			return next(c)
		}
	}
}

func GetUserID(c echo.Context) uuid.UUID {
	if id, ok := c.Get(string(ContextKeyUserID)).(uuid.UUID); ok {
		return id
	}
	return uuid.Nil
}

func GetUserRole(c echo.Context) string {
	if role, ok := c.Get(string(ContextKeyUserRole)).(string); ok {
		return role
	}
	return ""
}
