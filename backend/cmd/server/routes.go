package main

import (
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/sujaykumarsuman/verdox/backend/internal/config"
	"github.com/sujaykumarsuman/verdox/backend/internal/handler"
	mw "github.com/sujaykumarsuman/verdox/backend/internal/middleware"
	"github.com/sujaykumarsuman/verdox/backend/internal/queue"
	"github.com/sujaykumarsuman/verdox/backend/internal/repository"
	"github.com/sujaykumarsuman/verdox/backend/internal/service"
)

func registerTeamRoutes(e *echo.Echo, db *sqlx.DB, rdb *redis.Client, cfg *config.Config, log zerolog.Logger) {
	// Repositories
	teamRepo := repository.NewTeamRepository(db)
	memberRepo := repository.NewTeamMemberRepository(db)
	joinReqRepo := repository.NewTeamJoinRequestRepository(db)
	teamRepoRepo := repository.NewTeamRepoAssignmentRepository(db)
	repoRepo := repository.NewRepositoryRepository(db)
	userRepo := repository.NewUserRepository(db)

	// Services
	githubService := service.NewGitHubService(log)
	teamService := service.NewTeamService(teamRepo, memberRepo, joinReqRepo, teamRepoRepo, repoRepo, log)

	// Handler
	teamHandler := handler.NewTeamHandler(teamService, teamRepo, memberRepo, githubService, cfg, log)

	// Middleware
	authMiddleware := mw.Auth(cfg.JWTSecret, userRepo, rdb)
	requireAdmin := mw.RequireTeamRole(memberRepo, "admin")
	requireAdminMaintainer := mw.RequireTeamRole(memberRepo, "admin", "maintainer")
	requireMember := mw.RequireTeamRole(memberRepo, "admin", "maintainer", "viewer")

	teams := e.Group("/v1/teams", authMiddleware)

	// Non-parameterized routes FIRST (before /:id to avoid Echo treating "discover" as an ID)
	teams.GET("", teamHandler.ListMyTeams)
	teams.POST("", teamHandler.CreateTeam)
	teams.GET("/discover", teamHandler.DiscoverTeams)

	// Team-level routes
	teams.GET("/:id", teamHandler.GetTeam, requireMember)
	teams.PUT("/:id", teamHandler.UpdateTeam, requireAdmin)
	teams.DELETE("/:id", teamHandler.DeleteTeam, requireAdmin)

	// PAT routes
	teams.PUT("/:id/pat", teamHandler.SetPAT, requireAdmin)
	teams.GET("/:id/pat/validate", teamHandler.GetPATStatus, requireMember)
	teams.DELETE("/:id/pat", teamHandler.RevokePAT, requireAdmin)

	// Member routes
	teams.POST("/:id/members", teamHandler.InviteMember, requireAdminMaintainer)
	teams.PUT("/:id/members/:userId", teamHandler.UpdateMember, requireAdminMaintainer)
	teams.DELETE("/:id/members/:userId", teamHandler.RemoveMember) // self-removal handled in handler

	// Repo assignment routes
	teams.POST("/:id/repositories", teamHandler.AssignRepo, requireAdminMaintainer)
	teams.DELETE("/:id/repositories/:repoId", teamHandler.UnassignRepo, requireAdminMaintainer)

	// Join request routes
	teams.POST("/:id/join-requests", teamHandler.SubmitJoinRequest)         // any auth user
	teams.GET("/:id/join-requests", teamHandler.ListJoinRequests, requireAdminMaintainer)
	teams.PATCH("/:id/join-requests/:requestId", teamHandler.ReviewJoinRequest, requireAdminMaintainer)
}

func registerRepositoryRoutes(e *echo.Echo, db *sqlx.DB, rdb *redis.Client, cfg *config.Config, log zerolog.Logger, cloneCh chan<- service.CloneJob) {
	repoRepo := repository.NewRepositoryRepository(db)
	teamRepo := repository.NewTeamRepository(db)
	teamMemberRepo := repository.NewTeamMemberRepository(db)
	userRepo := repository.NewUserRepository(db)

	githubService := service.NewGitHubService(log)
	repoService := service.NewRepositoryService(repoRepo, teamRepo, teamMemberRepo, githubService, rdb, cfg, log, cloneCh)
	repoHandler := handler.NewRepositoryHandler(repoService)

	authMiddleware := mw.Auth(cfg.JWTSecret, userRepo, rdb)
	repos := e.Group("/v1/repositories", authMiddleware)
	repos.POST("", repoHandler.Create)
	repos.GET("", repoHandler.List)
	repos.GET("/:id", repoHandler.Get)
	repos.PUT("/:id", repoHandler.Update)
	repos.DELETE("/:id", repoHandler.Delete)
	repos.GET("/:id/branches", repoHandler.ListBranches)
	repos.GET("/:id/commits", repoHandler.ListCommits)
	repos.POST("/:id/resync", repoHandler.Resync)
	repos.POST("/:id/retry-clone", repoHandler.RetryClone)
}

func registerTestRoutes(e *echo.Echo, db *sqlx.DB, rdb *redis.Client, cfg *config.Config, log zerolog.Logger, q *queue.RedisQueue) {
	suiteRepo := repository.NewTestSuiteRepository(db)
	runRepo := repository.NewTestRunRepository(db)
	resultRepo := repository.NewTestResultRepository(db)
	repoRepo := repository.NewRepositoryRepository(db)
	teamMemberRepo := repository.NewTeamMemberRepository(db)
	userRepo := repository.NewUserRepository(db)

	suiteService := service.NewTestSuiteService(suiteRepo, repoRepo, teamMemberRepo, log)
	runService := service.NewTestRunService(runRepo, resultRepo, suiteRepo, repoRepo, teamMemberRepo, userRepo, q, rdb, cfg, log)

	suiteHandler := handler.NewTestSuiteHandler(suiteService)
	runHandler := handler.NewTestRunHandler(runService)

	authMiddleware := mw.Auth(cfg.JWTSecret, userRepo, rdb)

	// Suite routes under /v1/repositories/:id/suites
	repos := e.Group("/v1/repositories", authMiddleware)
	repos.GET("/:id/suites", suiteHandler.List)
	repos.POST("/:id/suites", suiteHandler.Create)
	repos.POST("/:id/run-all", runHandler.RunAll)

	// Suite-level routes under /v1/suites
	suites := e.Group("/v1/suites", authMiddleware)
	suites.PUT("/:id", suiteHandler.Update)
	suites.DELETE("/:id", suiteHandler.Delete)
	suites.POST("/:id/run", runHandler.Trigger)
	suites.GET("/:id/runs", runHandler.ListBySuite)

	// Run-level routes under /v1/runs
	runs := e.Group("/v1/runs", authMiddleware)
	runs.GET("/:id", runHandler.Get)
	runs.GET("/:id/logs", runHandler.Logs)
	runs.POST("/:id/cancel", runHandler.Cancel)
}

func registerWebhookRoutes(e *echo.Echo, db *sqlx.DB) {
	runRepo := repository.NewTestRunRepository(db)
	resultRepo := repository.NewTestResultRepository(db)
	webhookHandler := handler.NewWebhookHandler(runRepo, resultRepo)

	// No auth — run_id UUID acts as bearer token
	webhooks := e.Group("/v1/webhooks")
	webhooks.POST("/gha/:run_id", webhookHandler.GHACallback)
}

func registerDiscoveryRoutes(e *echo.Echo, db *sqlx.DB, rdb *redis.Client, cfg *config.Config, log zerolog.Logger) {
	repoRepo := repository.NewRepositoryRepository(db)
	userRepo := repository.NewUserRepository(db)

	discoveryService := service.NewDiscoveryService(repoRepo, cfg, log)
	discoveryHandler := handler.NewDiscoveryHandler(discoveryService)

	authMiddleware := mw.Auth(cfg.JWTSecret, userRepo, rdb)
	repos := e.Group("/v1/repositories", authMiddleware)
	repos.POST("/:id/discover", discoveryHandler.Discover)
	repos.GET("/:id/discovery", discoveryHandler.GetDiscovery)
}

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
