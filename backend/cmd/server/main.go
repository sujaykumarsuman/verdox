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
	"github.com/sujaykumarsuman/verdox/backend/internal/queue"
	"github.com/sujaykumarsuman/verdox/backend/internal/runner"
	"github.com/sujaykumarsuman/verdox/backend/internal/service"
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

	// Validate service account configuration
	if cfg.ServiceAccountPAT == "" || cfg.ServiceAccountUsername == "" {
		log.Warn().Msg("VERDOX_SERVICE_ACCOUNT_PAT or VERDOX_SERVICE_ACCOUNT_USERNAME not set — fork-based test execution will not work")
	}

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

	// SSE routes
	registerSSERoutes(e, db, rdb, cfg, log)

	// Notification routes
	registerNotificationRoutes(e, db, rdb, cfg, log)

	// Fork routes
	registerForkRoutes(e, db, rdb, cfg, log)

	// Redis queue for test runs
	redisQueue := queue.NewRedisQueue(rdb, cfg.RunnerMaxTimeout, log)

	// GHA poller for tracking dispatched GitHub Actions workflows
	ghaPoller := runner.NewGHAPoller(db, cfg.ServiceAccountPAT, log)

	// Fork GHA executor (uses service account)
	forkService := service.NewForkService(cfg, db, log)
	var forkGHAExec *runner.ForkGHAExecutor
	if forkService.IsConfigured() {
		forkGHAExec = runner.NewForkGHAExecutor(forkService, cfg, log, ghaPoller.Register)
		log.Info().Msg("fork GHA executor enabled (service account configured)")

		// Recover repos stuck in "forking" state (e.g., from server restart during setup)
		go forkService.RecoverStuckForks(context.Background())
	}

	// User settings & Admin routes
	registerUserRoutes(e, db, rdb, cfg, log)
	registerAdminRoutes(e, db, rdb, cfg, log)

	// Team & Repository routes
	registerTeamRoutes(e, db, rdb, cfg, log)
	registerRepositoryRoutes(e, db, rdb, cfg, log)

	// Test suite & run routes
	registerTestRoutes(e, db, rdb, cfg, log, redisQueue)

	// Webhook routes (no auth)
	registerWebhookRoutes(e, db, log)

	// Hierarchy routes (authenticated read endpoints for groups/cases)
	registerHierarchyRoutes(e, db, rdb, cfg, log)

	// Generate suite routes (if OpenAI API key configured)
	if cfg.OpenAIAPIKey != "" {
		registerGenerateSuiteRoutes(e, db, rdb, cfg, log)
	}

	// Worker pool
	pool := runner.NewWorkerPool(cfg, redisQueue, db, rdb, log, forkGHAExec)
	poolCtx, poolCancel := context.WithCancel(context.Background())
	defer poolCancel()
	pool.Start(poolCtx)

	// Start GHA poller
	pollerCtx, pollerCancel := context.WithCancel(context.Background())
	defer pollerCancel()
	ghaPoller.Start(pollerCtx)

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

	ghaPoller.Shutdown()
	pool.Shutdown()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("server forced to shutdown")
	}
	log.Info().Msg("server stopped")
}
