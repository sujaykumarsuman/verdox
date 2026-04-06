package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"

	"github.com/sujaykumarsuman/verdox/backend/internal/config"
	"github.com/sujaykumarsuman/verdox/backend/internal/handler"
	mw "github.com/sujaykumarsuman/verdox/backend/internal/middleware"
	"github.com/sujaykumarsuman/verdox/backend/pkg/logger"
	v "github.com/sujaykumarsuman/verdox/backend/pkg/validator"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Init logger
	log := logger.New(cfg.LogLevel)
	log.Info().Str("env", cfg.AppEnv).Msg("starting verdox server")

	// Connect Postgres
	db, err := sqlx.Connect("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	db.SetMaxOpenConns(cfg.DBMaxOpenConn)
	db.SetMaxIdleConns(cfg.DBMaxIdleConn)
	db.SetConnMaxLifetime(time.Duration(cfg.DBMaxLifetime) * time.Second)
	log.Info().Msg("connected to postgres")

	// Connect Redis
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse redis URL")
	}
	rdb := redis.NewClient(opt)
	defer rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal().Err(err).Msg("failed to connect to redis")
	}
	log.Info().Msg("connected to redis")

	// Bootstrap root user
	bootstrapRootUser(db, cfg, log)

	// Setup Echo
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Validator = v.New()

	// Global middleware
	e.Use(echomw.RequestID())
	e.Use(mw.Recover(log))
	e.Use(mw.Logger(log))
	e.Use(mw.CORS(cfg.CORSOriginsList()))

	// Health endpoints
	healthHandler := handler.NewHealthHandler(db, rdb)
	e.GET("/health", healthHandler.Liveness)
	e.GET("/health/ready", healthHandler.Readiness)

	// Auth routes
	registerAuthRoutes(e, db, rdb, cfg, log)

	// Start server
	port := cfg.AppPort
	if port == "" {
		port = "8080"
	}

	go func() {
		addr := ":" + port
		log.Info().Str("addr", addr).Msg("HTTP server listening")
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server failed")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("server forced to shutdown")
	}
	log.Info().Msg("server stopped")
}
