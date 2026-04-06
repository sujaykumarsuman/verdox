package main

import (
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/sujaykumarsuman/verdox/backend/internal/config"
	"github.com/sujaykumarsuman/verdox/backend/internal/handler"
	mw "github.com/sujaykumarsuman/verdox/backend/internal/middleware"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
	"github.com/sujaykumarsuman/verdox/backend/internal/service"
)

func registerAuthRoutes(e *echo.Echo, db *sqlx.DB, rdb *redis.Client, cfg *config.Config, log zerolog.Logger) {
	// Repositories
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	resetRepo := repository.NewPasswordResetRepository(db)

	// Services
	authService := service.NewAuthService(userRepo, sessionRepo, resetRepo, rdb, cfg, log)

	// Handlers
	authHandler := handler.NewAuthHandler(authService, userRepo, cfg)

	// Auth routes (public, with rate limiting)
	auth := e.Group("/v1/auth")
	auth.POST("/signup", authHandler.Signup, mw.SignupRateLimit(rdb))
	auth.POST("/login", authHandler.Login, mw.LoginRateLimit(rdb))
	auth.POST("/refresh", authHandler.Refresh, mw.RefreshRateLimit(rdb))
	auth.POST("/forgot-password", authHandler.ForgotPassword, mw.ForgotPasswordRateLimit(rdb))
	auth.POST("/reset-password", authHandler.ResetPassword)

	// Authenticated auth routes
	authMiddleware := mw.Auth(cfg.JWTSecret, userRepo, rdb)
	auth.GET("/me", authHandler.Me, authMiddleware)
	auth.POST("/logout", authHandler.Logout, authMiddleware)
}
