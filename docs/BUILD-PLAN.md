# Verdox -- Sprint-Based Implementation Plan

> Phase 0 (documentation) is complete. This plan covers Phase 1 through Phase 7.
>
> **Canonical references:** CODE-STRUCTURE.md, ARCHITECTURE.md, LLD/DATABASE.md,
> LLD/API.md, LLD/AUTH.md, LLD/TEST-RUNNER.md, LLD/FRONTEND-ROUTES.md,
> LLD/GITHUB-INTEGRATION.md, ADMIN-PANEL.md, SECURITY.md, SETUP.md,
> DEPLOYMENT.md, BRANCHING-STRATEGY.md, GITHUB-PAT-GUIDE.md
>
> **Git workflow:** All development follows `BRANCHING-STRATEGY.md` — never
> push to `main`, always use a PR, branch names must use `feat/`, `fix/`,
> `bug/`, `hotfix/`, `chore/`, `docs/`, `refactor/`, `test/`, or `perf/`
> prefixes.

---

## Phase 1 -- Foundation

**Goal:** Project scaffolding, Docker infrastructure, database schema, auth
system, and minimal frontend. By the end of this phase a user can sign up,
log in, see a dashboard shell, and hit health endpoints.

**Gate:** Auth flow works end-to-end. Can sign up, login, see dashboard shell.

### 1.1 Project Scaffolding

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | Initialize Go module | `backend/go.mod` (`github.com/sujaykumarsuman/verdox/backend`) | Go 1.25+. Run `go mod init`. |
| 2 | Install backend dependencies | `backend/go.sum` | Echo v4, sqlx, pgx, redis/v9, golang-jwt/jwt/v5, viper, zerolog, validator/v10, bcrypt, uuid. |
| 3 | Initialize Next.js project | `frontend/` (App Router, TypeScript, Tailwind CSS) | `npx create-next-app@15 frontend --typescript --tailwind --app --src-dir`. |
| 4 | Install frontend dependencies | `frontend/package.json` | next-themes, sonner, clsx, tailwind-merge, lucide-react. |
| 5 | Create root Makefile | `Makefile` | All targets from CODE-STRUCTURE.md Section 8: `dev`, `dev-backend`, `dev-frontend`, `build`, `build-backend`, `build-frontend`, `migrate-up`, `migrate-down`, `migrate-create`, `seed`, `up`, `down`, `logs`, `test`, `test-backend`, `test-frontend`, `lint`. |
| 6 | Create `.env.example` | `.env.example` | `DATABASE_URL`, `REDIS_URL`, `JWT_SECRET`, `JWT_ACCESS_EXPIRY`, `JWT_REFRESH_EXPIRY`, `GITHUB_TOKEN_ENCRYPTION_KEY`, `VERDOX_REPO_BASE_PATH`, `ROOT_EMAIL`, `ROOT_PASSWORD`, `BCRYPT_COST`, `SERVER_PORT`, `FRONTEND_URL`, `CORS_ORIGINS`, `LOG_LEVEL`. |
| 7 | Create `.gitignore` | `.gitignore` | Go binaries, `vendor/`, `node_modules/`, `.next/`, `.env`, `*.exe`, `.DS_Store`, IDE files. |

### 1.2 Docker Infrastructure

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | Write `docker-compose.yml` | `docker-compose.yml` | 6 services: `verdox-nginx` (:80/:443), `verdox-frontend` (:3000), `verdox-backend` (:8080), `verdox-postgres` (:5432), `verdox-redis` (:6379), `verdox-runner`. Internal network `verdox-network`. Healthchecks on postgres and redis. |
| 2 | Write `docker-compose.dev.yml` | `docker-compose.dev.yml` | Volume mounts for hot reload (backend source, frontend source). Expose debug ports (5432, 6379). Use `air` for Go hot reload. |
| 3 | Write backend Dockerfile | `docker/backend.Dockerfile` | Multi-stage: `golang:1.25-alpine` (builder) -> `alpine:3.21` (runtime). CGO_ENABLED=0. Non-root user. ~15 MB final image. |
| 4 | Write frontend Dockerfile | `docker/frontend.Dockerfile` | Multi-stage: `node:22-alpine` (deps) -> `node:22-alpine` (builder, `next build`) -> `node:22-alpine` (runtime, standalone output). Non-root user. |
| 5 | Write runner Dockerfile | `docker/runner.Dockerfile` | Based on DinD image. Pre-install Go, Node, Python runtimes for test execution. |
| 6 | Write Nginx config | `nginx/nginx.conf`, `nginx/conf.d/default.conf` | Proxy rules: `/api/*` -> backend:8080 (strip `/api` prefix), `/webhooks/*` -> backend:8080 (passthrough), `/*` -> frontend:3000. Worker processes, gzip, log format. |

### 1.3 Database Setup

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | Write migration 000001 | `migrations/000001_create_enum_types.{up,down}.sql` | 6 enum types: `user_role`, `team_member_role`, `team_member_status`, `test_type`, `test_run_status`, `test_result_status`. |
| 2 | Write migration 000002 | `migrations/000002_create_users.{up,down}.sql` | `users` table with UUID PK, username, email, password_hash, role, avatar_url, timestamps. Unique constraints on username and email. |
| 3 | Write migration 000003 | `migrations/000003_create_sessions.{up,down}.sql` | `sessions` table with FK to users, refresh_token_hash, expires_at. Indexes on user_id and expires_at. |
| 4 | Write migration 000004 | `migrations/000004_create_repositories.{up,down}.sql` | `repositories` table. Unique on github_repo_id. No owner_id -- repositories are associated to teams via `team_repositories`. |
| 5 | Write migration 000005 | `migrations/000005_create_teams.{up,down}.sql` | `teams` table. Unique on slug. FK to users (created_by). |
| 6 | Write migration 000006 | `migrations/000006_create_team_members.{up,down}.sql` | `team_members` join table. Composite unique (team_id, user_id). FKs to teams, users. Index on user_id. |
| 7 | Write migration 000007 | `migrations/000007_create_team_repositories.{up,down}.sql` | `team_repositories` join table. Composite unique (team_id, repository_id). FKs to teams, repositories, users. |
| 8 | Write migration 000008 | `migrations/000008_create_test_suites.{up,down}.sql` | `test_suites` table. FK to repositories. Index on repository_id. |
| 9 | Write migration 000009 | `migrations/000009_create_test_runs.{up,down}.sql` | `test_runs` table. FKs to test_suites, users. Indexes on suite_id, (suite_id, status), triggered_by. |
| 10 | Write migration 000010 | `migrations/000010_create_test_results.{up,down}.sql` | `test_results` table. FK to test_runs. Index on test_run_id. |
| 11 | Implement `migrate-up`/`migrate-down`/`migrate-create` | `Makefile` targets | Uses golang-migrate CLI against `DB_DSN`. |
| 12 | Root user bootstrap | `cmd/server/main.go` (startup) | Root user bootstrap from `ROOT_EMAIL` / `ROOT_PASSWORD` env vars. On first boot, upsert user with role=root. Bcrypt cost 12 hash. Skip if user already exists. |
| 13 | Database connection pool setup | `backend/internal/config/config.go` | Viper-based config. Expose `Config` struct with DB pool settings (max open, max idle, max lifetime). |

### 1.4 Backend Core

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | Config loading | `internal/config/config.go` | Viper. Reads `.env` and environment variables. Typed `Config` struct covering all vars from `.env.example`. |
| 2 | Logger setup | `pkg/logger/logger.go` | Zerolog initialization. JSON output. Log level from config. Caller information enabled. |
| 3 | Echo server setup | `cmd/server/main.go` | Load config, connect Postgres (sqlx), connect Redis, register routes, register middleware chain (recover, logger, CORS, rate limit), start HTTP server. Graceful shutdown on SIGINT/SIGTERM. |
| 4 | Middleware: recover | `internal/middleware/recover.go` | Catch panics, return 500 with request ID. |
| 5 | Middleware: logger | `internal/middleware/logger.go` | Log method, path, status, latency, request ID using zerolog. |
| 6 | Middleware: CORS | `internal/middleware/cors.go` | Allowed origins from config (`CORS_ORIGINS`). Allow credentials. Allowed methods/headers. |
| 7 | Standardized response helpers | `pkg/response/response.go` | `Success(c, status, data)`, `Error(c, status, code, message)`, `ValidationError(c, details)`. Standard envelope: `{"data": ...}` or `{"error": {"code": "...", "message": "...", "details": ...}}`. |
| 8 | Custom validators | `pkg/validator/validator.go` | Register with Echo. Custom rules: `strong_password` (min 8, 1 upper, 1 lower, 1 digit), `github_url`. |
| 9 | Health check endpoints | `internal/handler/health.go` (or inline in main.go) | `GET /health` (liveness: always 200), `GET /health/ready` (readiness: ping Postgres + Redis). |

### 1.5 Auth System

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | User model | `internal/model/user.go` | Struct with `db` and `json` tags. Fields: id, username, email, password_hash, role, avatar_url, created_at, updated_at. |
| 2 | User repository | `internal/repository/user_repo.go` | Interface + sqlx implementation. Methods: Create, GetByID, GetByEmail, GetByUsername, Update, Delete, List (with offset/limit/total). |
| 3 | Session model | `internal/model/session.go` | Fields: id, user_id, refresh_token_hash, expires_at, created_at. |
| 4 | Session repository | `internal/repository/session_repo.go` | Methods: Create, GetByID, GetByUserID, DeleteByID, DeleteByUserID, DeleteExpired. |
| 5 | Auth DTOs | `internal/dto/auth_dto.go` | SignupRequest, LoginRequest, RefreshRequest, ForgotPasswordRequest, ResetPasswordRequest, AuthResponse, TokenResponse. Validation tags. |
| 6 | JWT utilities | `pkg/jwt/jwt.go` | GenerateAccessToken (HS256, 15min, claims: user_id, role), GenerateRefreshToken (32 random bytes, hex-encoded), ValidateAccessToken, ExtractClaims. |
| 7 | Hash utilities | `pkg/hash/hash.go` | HashPassword (bcrypt cost 12), CheckPassword, SHA256Hash (for refresh token storage). |
| 8 | Auth service | `internal/service/auth_service.go` | Signup (validate, hash password, insert user, generate tokens, create session, cache in Redis). Login (verify email, check password, generate tokens, create session). Refresh (validate refresh token hash against DB, rotate tokens, update session). Logout (delete session from DB and Redis). ForgotPassword (generate reset token, store hashed). ResetPassword (validate token, update password hash). |
| 9 | Auth middleware | `internal/middleware/auth.go` | Extract JWT from `Authorization: Bearer` header or `verdox_access` cookie. Validate. Load user from DB. Check session exists in Redis. Inject user into Echo context. Return 401 on failure. |
| 10 | Rate limiting middleware | `internal/middleware/ratelimit.go` | Redis sliding window. Per-IP for public endpoints. Per-user for authenticated endpoints. Config: window size, max requests. Auth endpoints: 5 req/min per IP (signup), 10 req/min per IP (login). |
| 11 | Auth handlers | `internal/handler/auth.go` | `POST /api/v1/auth/signup` (201), `POST /api/v1/auth/login` (200), `POST /api/v1/auth/refresh` (200), `POST /api/v1/auth/logout` (200), `POST /api/v1/auth/forgot-password` (200), `POST /api/v1/auth/reset-password` (200). Set refresh token as httpOnly cookie. |
| 12 | Route registration | `cmd/server/main.go` | Wire auth handlers to Echo routes. Apply rate limit middleware to auth group. |

### 1.6 Frontend Foundation

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | Root layout | `src/app/layout.tsx` | HTML head, font loading (DM Serif Display, DM Sans, JetBrains Mono), ThemeProvider (next-themes), AuthProvider, Toaster (sonner). |
| 2 | Global CSS with design tokens | `src/styles/globals.css` | Tailwind directives. CSS custom properties for colors from BRAND-PALETTE.md. Light and dark mode variables. Base resets. |
| 3 | Tailwind config | `tailwind.config.ts` | Extend theme with custom colors (Slate Charcoal, Warm White, Signal Green, Scarlet, Amber, Verdox Indigo), fonts, spacing. |
| 4 | API client | `src/lib/api.ts` | Fetch wrapper. Base URL from env. Automatic `Authorization` header from cookie/context. JSON serialization. Error handling (parse error envelope). Token refresh on 401. |
| 5 | Auth context | `src/lib/auth.tsx` | AuthContext provider. State: user, loading, isAuthenticated. Actions: login, signup, logout, refreshToken. Persist auth state. Auto-refresh before expiry. |
| 6 | Utility helpers | `src/lib/utils.ts` | `cn()` (clsx + twMerge), `formatDate()`, `pluralize()`. |
| 7 | TypeScript types | `src/types/user.ts` | User, UserRole, Session types matching API response shapes. |
| 8 | Base UI: Button | `src/components/ui/button.tsx` | Variants: primary, secondary, ghost, danger. Sizes: sm, md, lg. Loading state with spinner. |
| 9 | Base UI: Input | `src/components/ui/input.tsx` | Label, error state, icon support, disabled state. |
| 10 | Base UI: Card | `src/components/ui/card.tsx` | Container with optional header and footer. |
| 11 | Route protection middleware | `src/middleware.ts` | Next.js middleware. Redirect unauthenticated users to `/login`. Redirect authenticated users away from auth pages to `/dashboard`. Check `verdox_access` cookie existence. |
| 12 | useAuth hook | `src/hooks/use-auth.ts` | Wraps AuthContext. Returns user, loading, isAuthenticated, login, signup, logout. |

### 1.7 Auth Pages

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | Landing page | `src/app/page.tsx` | Hero section with tagline. CTA buttons: "Get Started" -> /signup, "Sign In" -> /login. Verdox branding. |
| 2 | Auth layout | `src/app/(auth)/layout.tsx` | Centered card on warm neutral background. Verdox logo at top. No sidebar, no topbar. |
| 3 | Login form component | `src/components/auth/login-form.tsx` | Email + password fields. Validation (required, email format). Error display. "Forgot password?" link. Submit calls auth context login. |
| 4 | Login page | `src/app/(auth)/login/page.tsx` | Renders LoginForm. Link to signup. |
| 5 | Signup form component | `src/components/auth/signup-form.tsx` | Username + email + password + confirm password. Password strength indicator. Validation. Submit calls auth context signup. |
| 6 | Signup page | `src/app/(auth)/signup/page.tsx` | Renders SignupForm. Link to login. |
| 7 | Forgot password page | `src/app/(auth)/forgot-password/page.tsx` | Email input. Submit sends POST to forgot-password. Success message. |
| 8 | Reset password page | `src/app/(auth)/reset-password/page.tsx` | New password + confirm. Reads token from query string. Submit sends POST to reset-password. |
| 9 | Dashboard layout | `src/app/(dashboard)/layout.tsx` | Sidebar (260px expanded / 64px collapsed) + TopBar (56px). Main content area. Protected route (wraps with auth check). |
| 10 | Sidebar component | `src/components/layout/sidebar.tsx` | Collapsible. Nav links: Dashboard, Teams, Settings, Admin (conditional on role). Active state highlighting. Verdox logo. |
| 11 | TopBar component | `src/components/layout/topbar.tsx` | Breadcrumbs. User menu dropdown (avatar, Settings, Sign Out). |
| 12 | Dashboard shell | `src/app/(dashboard)/dashboard/page.tsx` | Empty content area with "Welcome to Verdox" placeholder. Will be replaced with repo cards in Phase 2. |

### Phase 1 Confirmation Gate

- [ ] `make dev` starts all 6 services without errors
- [ ] `make migrate-up` applies all 10 migrations successfully
- [ ] Root user bootstrap creates root user from `ROOT_EMAIL`/`ROOT_PASSWORD` env vars on first boot
- [ ] `GET /health` returns 200
- [ ] `GET /health/ready` returns 200 (Postgres + Redis healthy)
- [ ] `POST /api/v1/auth/signup` creates a new user and returns JWT + refresh cookie
- [ ] `POST /api/v1/auth/login` authenticates and returns tokens
- [ ] `POST /api/v1/auth/refresh` rotates tokens
- [ ] `POST /api/v1/auth/logout` invalidates session
- [ ] JWT access token is 15min, HS256
- [ ] Refresh token is httpOnly cookie, 7-day expiry
- [ ] Rate limiting blocks >5 signup requests/min from same IP
- [ ] Frontend `/signup` page renders and submits successfully
- [ ] Frontend `/login` page renders and submits successfully
- [ ] After login, user is redirected to `/dashboard`
- [ ] Dashboard shell renders with sidebar and topbar
- [ ] Unauthenticated access to `/dashboard` redirects to `/login`
- [ ] Authenticated access to `/login` redirects to `/dashboard`

---

## Phase 2 -- Repository Management

**Goal:** GitHub PAT integration, repository addition by URL, local clone,
branch/commit browsing from local clone. Team admins can configure a GitHub PAT
for their team, add repos by URL, and browse cloned repositories.

**Gate:** Can store GitHub PAT, add repo by URL, clone completes, browse branches from local clone.

### 2.1 Team-Level GitHub PAT Integration

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | PAT storage endpoint | `internal/handler/team.go` (extend) | `PUT /api/v1/teams/:id/pat` -- accept GitHub PAT, validate against GitHub API (`GET /user`), encrypt and store on the `teams` table. Requires team admin role. |
| 2 | PAT encryption | `internal/service/github_service.go` (extend) | Encrypt PAT with AES-256-GCM before storing in `teams.github_pat_encrypted` column. Decrypt on use. Key from `GITHUB_TOKEN_ENCRYPTION_KEY` env var. |
| 3 | PAT validation | `internal/service/github_service.go` (extend) | On store, call GitHub API to verify PAT has `repo` scope. Return error if invalid or insufficient scope. |
| 4 | PAT status endpoint | `internal/handler/team.go` (extend) | `GET /api/v1/teams/:id/pat/validate` -- validate the stored PAT is still valid. `DELETE /api/v1/teams/:id/pat` -- revoke the team's PAT. Both require team admin role. |
| 5 | Environment variables | `.env.example` update | `GITHUB_TOKEN_ENCRYPTION_KEY`, `VERDOX_REPO_BASE_PATH`. |

### 2.2 Repository Addition & Clone

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | Repository model | `internal/model/repository.go` | Fields: id, github_full_name, name, description, default_branch, clone_path, clone_status (pending/cloning/ready/failed), is_active, created_at, updated_at. |
| 2 | Repository DTOs | `internal/dto/repository_dto.go` | AddRepoRequest (github_url), UpdateRepoRequest, RepoResponse, RepoListResponse, BranchResponse, CommitResponse. |
| 3 | Repository repository | `internal/repository/repository_repo.go` | Create, GetByID, GetByGitHubFullName, ListByTeam (paginated, sorted), Update, SoftDelete (set is_active=false). |
| 4 | Repository service | `internal/service/repository_service.go` | AddRepo (parse URL, resolve team PAT, fetch metadata via PAT, insert, enqueue clone job), GetDetail, List, Deactivate, GetBranches (from local clone via `git branch -r`), GetCommits (from local clone via `git log`). |
| 5 | Clone worker job | `internal/queue/repo_clone.go` | `repo.clone` job type. Worker clones repo to `VERDOX_REPO_BASE_PATH/<owner>/<repo>` using the team's PAT. Updates clone_status on success/failure. |
| 6 | Repository handlers | `internal/handler/repository.go` | `POST /api/v1/repositories` (add by URL), `GET /api/v1/repositories` (list), `GET /api/v1/repositories/:id` (detail), `PUT /api/v1/repositories/:id` (update), `DELETE /api/v1/repositories/:id` (soft delete), `GET /api/v1/repositories/:id/branches`, `GET /api/v1/repositories/:id/commits`. All require auth middleware. |
| 7 | TypeScript types | `src/types/repository.ts` | Repository, Branch, Commit types. |

### 2.3 Frontend -- Repository Pages

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | useRepos hook | `src/hooks/use-repos.ts` | Fetch repos, refetch on sync. Caching. |
| 2 | RepoCard component | `src/components/repository/repo-card.tsx` | Shows name, description, default branch, last updated, active status. Links to `/repositories/[id]`. |
| 3 | Dashboard page | `src/app/(dashboard)/dashboard/page.tsx` | Replace placeholder with repo card grid. Search/filter bar. "Add Repository" button. Empty state with CTA when no repos. |
| 4 | BranchSelector component | `src/components/repository/branch-selector.tsx` | Dropdown with search. Fetches branches from local clone via API. Shows current selection. |
| 5 | CommitList component | `src/components/repository/commit-list.tsx` | Scrollable list. Shows SHA (truncated), message, author, date. Loads for selected branch from local clone. |
| 6 | Repository detail page | `src/app/(dashboard)/repositories/[id]/page.tsx` | Repo header (name, description, GitHub link). Clone status indicator. BranchSelector. CommitList. Test suites section (placeholder for Phase 3). |
| 7 | Loading skeletons | `src/components/ui/skeleton.tsx` | Skeleton placeholders for repo cards and repo detail. |

### Phase 2 Confirmation Gate

- [ ] `PUT /api/v1/teams/:id/pat` stores team GitHub PAT encrypted (AES-256-GCM), requires team admin role
- [ ] PAT validation rejects invalid tokens or tokens without `repo` scope
- [ ] `POST /api/v1/repositories` adds a repo by GitHub URL
- [ ] Clone worker job clones repo to `VERDOX_REPO_BASE_PATH`
- [ ] Clone status transitions: pending -> cloning -> ready
- [ ] `GET /api/v1/repositories` returns user's repos with pagination
- [ ] Dashboard shows repo cards in a grid
- [ ] Clicking a repo card navigates to `/repositories/[id]`
- [ ] Repo detail page shows repository metadata and clone status
- [ ] BranchSelector loads branches from local clone
- [ ] CommitList shows commits for the selected branch from local clone
- [ ] Deactivating a repo sets `is_active=false`
- [ ] Empty state shows when user has no repos

---

## Phase 3 -- Test Execution

**Goal:** Test suite CRUD, Redis job queue, Docker-in-Docker runner, test
result parsing, and results display. Users can configure test suites, trigger
runs, and see results with logs.

**Gate:** Can configure a test suite, run it, see results with logs.

### 3.1 Test Suite CRUD

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | TestSuite model | `internal/model/test_suite.go` | Fields: id, repository_id, name, type (unit/integration), config_path, timeout_seconds (default 300), created_at, updated_at. |
| 2 | TestSuite DTOs | `internal/dto/test_dto.go` | CreateSuiteRequest, UpdateSuiteRequest, SuiteResponse. |
| 3 | TestSuite repository | `internal/repository/test_suite_repo.go` | Create, GetByID, ListByRepoID, Update, Delete. |
| 4 | TestSuite service | `internal/service/test_service.go` (suite portion) | Create (validate repo ownership), Update, Delete, List. |
| 5 | TestSuite handlers | `internal/handler/test_suite.go` | `GET /api/v1/repositories/:id/suites` (list), `POST /api/v1/repositories/:id/suites` (create), `PUT /api/v1/suites/:id` (update), `DELETE /api/v1/suites/:id` (delete). Auth required. |
| 6 | Frontend: SuiteCard | `src/components/test/suite-card.tsx` | Suite name, type badge, timeout, config path, "Run" button, edit/delete actions. |
| 7 | Frontend: suite list on repo detail | `src/app/(dashboard)/repositories/[id]/page.tsx` (extend) | Add test suites section below commits. "Add Suite" button. Suite creation form (modal or inline). |

### 3.2 Job Queue

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | Queue implementation | `internal/queue/queue.go` | Redis LIST-backed FIFO. `Push(ctx, job)` -- LPUSH serialized job payload. `Pop(ctx)` -- BRPOP with timeout. `Ack(ctx, jobID)` -- remove from processing set. `Fail(ctx, jobID, err)` -- move to dead-letter queue. `Retry(ctx, jobID)` -- re-enqueue from dead-letter. |
| 2 | Job payload | `internal/queue/queue.go` | Struct: TestRunJobID, TestSuiteID, RepoID, Branch, CommitHash, ConfigPath, Timeout, DockerImage. JSON serialization. |

### 3.3 Test Runner

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | Worker pool | `internal/runner/runner.go` | Configurable worker count (default 3). Long-running goroutines that BRPOP from Redis queue. Graceful shutdown (drain in-progress jobs on SIGTERM). Concurrency control via semaphore. |
| 2 | Container executor | `internal/runner/executor.go` | Docker Engine API client. CreateContainer (image, env, volume mounts, resource limits). Mount local clone read-only (`VERDOX_REPO_BASE_PATH/<owner>/<repo>:ro`) instead of per-run git clone. StartContainer. WaitContainer (with timeout from suite config). CaptureOutput (stdout/stderr streams). RemoveContainer (always, even on failure). Resource limits: 512MB memory, 1 CPU, no network access for test containers. |
| 3 | Output parser | `internal/runner/parser.go` | Parse Go test output (`go test -json`): extract test name, pass/fail/skip, duration. Parse pytest output (`--tb=short`): regex-based extraction. Parse Jest output (`--json`): JSON structure. Return `[]TestResult` structs. |
| 4 | TestRun model | `internal/model/test_run.go` | Fields: id, test_suite_id, triggered_by, branch, commit_hash, run_number (auto-increment per suite: run-1, run-2, ...), status (queued/running/passed/failed/cancelled), started_at, finished_at, created_at. |
| 5 | TestRun repository | `internal/repository/test_run_repo.go` | Create, GetByID, ListBySuiteID (paginated), UpdateStatus, Cancel. |
| 6 | TestResult model | `internal/model/test_result.go` | Fields: id, test_run_id, test_name, status (pass/fail/skip/error), duration_ms, error_message, log_output, created_at. |
| 7 | TestResult repository | `internal/repository/test_result_repo.go` | BulkCreate (batch insert), ListByRunID. |

### 3.4 Test Run API

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | Test run service | `internal/service/test_service.go` (run portion) | TriggerRun (resolve commit hash from local clone, cache commit hash to skip re-runs on same commit, assign run_number, insert test_run with status=queued, push job to Redis queue). Only team admin/maintainer can trigger runs. GetRun. ListRuns. CancelRun (update status, kill container if running). GetLogs (fetch from test_results). |
| 2 | Test run handlers | `internal/handler/test_run.go` | `POST /api/v1/suites/:id/runs` (trigger), `GET /api/v1/runs` (list all, filterable by status), `GET /api/v1/runs/:id` (detail with results), `GET /api/v1/runs/:id/logs` (log output), `POST /api/v1/runs/:id/cancel`. Auth required. |
| 3 | SSE log streaming | `internal/handler/test_run.go` | `GET /api/v1/runs/:id/logs/stream` (optional). Server-Sent Events endpoint. Worker writes log lines to Redis pub/sub, handler streams to client. |

### 3.5 Frontend -- Test Pages

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | TypeScript types | `src/types/test.ts` | TestSuite, TestRun, TestResult, RunStatus, TestStatus types. |
| 2 | RunList component | `src/components/test/run-list.tsx` | List of test runs. Columns: branch, commit (truncated), status badge, duration, triggered at. Links to run detail. |
| 3 | Repo detail: run history | `src/app/(dashboard)/repositories/[id]/page.tsx` (extend) | Add recent runs section. "Run All" button to trigger runs for all suites. |
| 4 | ProgressBar component | `src/components/ui/progress.tsx` | Determinate (percentage) and indeterminate (animated) modes. Color based on status. |
| 5 | ResultRow component | `src/components/test/result-row.tsx` | Individual test case row. Pass/fail/skip icon. Test name. Duration. Expandable error message and log output. |
| 6 | LogViewer component | `src/components/test/log-viewer.tsx` | ANSI-color-aware terminal output renderer. Monospace font (JetBrains Mono). Auto-scroll to bottom. Copy button. |
| 7 | Test run detail page | `src/app/(dashboard)/repositories/[id]/runs/[runId]/page.tsx` | Run metadata (branch, commit, status, duration). Summary bar (X passed, Y failed, Z skipped). List of ResultRow components. LogViewer for full output. Cancel button for queued/running. |
| 8 | Polling for active runs | Frontend | Poll `GET /api/v1/runs/:id` every 3 seconds while status is `queued` or `running`. Stop polling when terminal state is reached. |
| 9 | Badge component | `src/components/ui/badge.tsx` | Status variants: success (passed), error (failed), warning (cancelled), neutral (queued), info (running). |

### 3.6 Runner Infrastructure

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | Runner Dockerfile | `docker/runner.Dockerfile` (refine) | DinD base. Install Go 1.25, Node 22, Python 3.12 with pytest. Minimal footprint. |
| 2 | Docker network config | `docker-compose.yml` update | Ensure runner service has access to Docker socket (`/var/run/docker.sock`). Privileged mode for DinD. |
| 3 | Resource limits | `internal/runner/executor.go` | Memory: 512MB. CPU: 1 core. PID limit: 256. No network by default. Configurable per suite via future config. |

### Phase 3 Confirmation Gate

- [ ] Can create a test suite via `POST /api/v1/repositories/:id/suites`
- [ ] Can edit and delete test suites
- [ ] Suite cards appear on repo detail page
- [ ] `POST /api/v1/suites/:id/runs` enqueues a job and creates a test_run with status=queued
- [ ] Only team admin/maintainer can trigger runs (viewer gets 403)
- [ ] Run numbering assigns sequential run-1, run-2, etc. per suite
- [ ] Commit-hash caching skips re-run if same commit was already tested on that suite
- [ ] Worker picks up the job from Redis queue
- [ ] Worker mounts local clone read-only into ephemeral Docker container
- [ ] Test output is captured and parsed into individual test_results
- [ ] `GET /api/v1/runs/:id` returns run detail with nested test_results
- [ ] Test run detail page renders with pass/fail status badges
- [ ] LogViewer displays test output
- [ ] `POST /api/v1/runs/:id/cancel` cancels a queued or running test
- [ ] Polling updates the UI when a run transitions from running to passed/failed
- [ ] Resource limits are enforced (container is killed if it exceeds memory/timeout)
- [ ] Container is always cleaned up, even on failure

---

## Phase 4 -- Teams & Access Control

**Goal:** Team creation and management, member invitations with approval flow,
repository assignment to teams, and role-based permission enforcement.

**Gate:** Teams work. Members with different roles see appropriate content.

### 4.1 Team CRUD

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | Team model | `internal/model/team.go` | Team struct: id, name, slug, description, is_discoverable, created_by, created_at, updated_at. TeamMember struct: id, team_id, user_id, role (admin/maintainer/viewer), status (pending/approved/rejected), invited_by, created_at. TeamRepository struct: id, team_id, repository_id, added_by, created_at. TeamJoinRequest struct: id, team_id, user_id, message, status (pending/approved/rejected), reviewed_by, created_at, updated_at (`team_join_requests` table). |
| 2 | Team DTOs | `internal/dto/team_dto.go` | CreateTeamRequest, UpdateTeamRequest, InviteMemberRequest, TeamResponse, TeamDetailResponse, MemberResponse. |
| 3 | Team repository | `internal/repository/team_repo.go` | Create, GetByID, GetBySlug, List (paginated), Update, Delete. ListMembers (by team, with status filter). AddMember, UpdateMember (role, status), RemoveMember. AddRepository, RemoveRepository, ListRepositories. |
| 4 | Team service | `internal/service/team_service.go` | CreateTeam (auto-add creator as admin). InviteMember (check existing membership, create with status=pending). ApproveMember, RejectMember (require team admin/maintainer role). UpdateMemberRole (require team admin). RemoveMember. AssignRepo (check repo ownership). UnassignRepo. ListTeams, GetTeamDetail. RequestJoin (create join request for discoverable teams). ApproveJoinRequest, RejectJoinRequest. Enforce: creator is automatically team admin. |
| 5 | RequireTeamRole middleware | `internal/middleware/auth.go` (extend) | Middleware factory: `RequireTeamRole(roles ...string)`. Reads `:id` param, queries team_members for current user. Checks role (admin/maintainer/viewer) and approved status. Returns 403 if insufficient. root bypasses. |
| 6 | Team handlers | `internal/handler/team.go` | `POST /api/v1/teams` (create), `GET /api/v1/teams` (list user's teams), `GET /api/v1/teams/:id` (detail), `PUT /api/v1/teams/:id` (update), `DELETE /api/v1/teams/:id` (delete, admin only), `POST /api/v1/teams/:id/members` (invite), `PUT /api/v1/teams/:id/members/:userId` (approve/reject/role change), `DELETE /api/v1/teams/:id/members/:userId` (remove), `POST /api/v1/teams/:id/repositories` (assign), `DELETE /api/v1/teams/:id/repositories/:repoId` (unassign), `POST /api/v1/teams/:id/join-requests` (request to join), `PUT /api/v1/teams/:id/join-requests/:requestId` (approve/reject join request), `GET /api/v1/teams/discover` (list discoverable teams). |
| 7 | Team deletion cascade | `internal/service/team_service.go`, `internal/handler/team.go` | Implement team soft-delete with cascade: mark repos inactive (`is_active = false`), queue disk cleanup job for local clones (`verdox:jobs:cleanup`), remove memberships. |

### 4.2 Frontend -- Team Pages

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | TypeScript types | `src/types/team.ts` | Team, TeamMember, TeamRole, TeamMemberStatus types. |
| 2 | useTeams hook | `src/hooks/use-teams.ts` | Fetch teams, members, repos. Mutation helpers for invite/approve/assign. |
| 3 | TeamCard component | `src/components/team/team-card.tsx` | Team name, member count, repo count, created date. Links to `/teams/[id]`. |
| 4 | Teams list page | `src/app/(dashboard)/teams/page.tsx` | Grid of TeamCards. "Create Team" button. Create team modal (name input, auto-generates slug). "Discover Teams" link. Empty state. |
| 5 | Modal component | `src/components/ui/modal.tsx` | Dialog overlay. Focus trap. Escape to close. Backdrop click to close. Title, body, footer slots. |
| 6 | MemberList component | `src/components/team/member-list.tsx` | Table: avatar, username, role badge, status, joined date. Actions: approve/reject (for pending), change role (dropdown), remove. Conditional rendering based on current user's team role. |
| 7 | RepoAssign component | `src/components/team/repo-assign.tsx` | Multi-select of user's repos. Shows currently assigned repos. Add/remove buttons. |
| 8 | Team detail page | `src/app/(dashboard)/teams/[id]/page.tsx` | Tab or panel layout. Members panel (MemberList + invite form). Repositories panel (RepoAssign). Team settings (name, delete) for team admins. |
| 9 | Dropdown component | `src/components/ui/dropdown.tsx` | Keyboard-navigable dropdown menu. Used for role selection and user menu. |
| 10 | Table component | `src/components/ui/table.tsx` | Sortable columns. Pagination controls. Used in member list and admin panel. |
| 11 | Team discovery page | `src/app/(dashboard)/teams/discover/page.tsx` | List of discoverable teams with "Request to Join" button. Shows team name, description, member count. Pending request indicator. |

### 4.3 Permission Enforcement

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | API permission matrix | All handlers | Verify: root can do everything. Team admin can manage members and repos. Team maintainer can approve/reject members and join requests. Team viewer can view only. User can only see teams they belong to. |
| 2 | Frontend conditional rendering | All team components | Hide action buttons based on user's team role. Show invite form only for admin/maintainer. Show role change dropdown only for admin. Show delete team only for team admin or root. |
| 3 | Permission test scenarios | _(Deferred to Phase 7)_ | Moved to Phase 7 -- Testing & Hardening. |

### Phase 4 Confirmation Gate

- [ ] `POST /api/v1/teams` creates a team with the creator as admin member
- [ ] `POST /api/v1/teams/:id/members` invites a user with status=pending
- [ ] Team admin can approve/reject pending members
- [ ] Team admin can change member roles (admin/maintainer/viewer)
- [ ] Team admin can remove members
- [ ] `POST /api/v1/teams/:id/repositories` assigns a repo to a team
- [ ] `DELETE /api/v1/teams/:id/repositories/:repoId` unassigns a repo
- [ ] Team viewer cannot invite, approve, or change roles (403)
- [ ] Team maintainer can approve/reject but cannot change roles (403 on role change)
- [ ] `POST /api/v1/teams/:id/join-requests` creates a join request for discoverable teams
- [ ] Team admin/maintainer can approve/reject join requests
- [ ] `GET /api/v1/teams/discover` lists discoverable teams
- [ ] Team discovery page renders with "Request to Join" button
- [ ] root bypasses all team permission checks
- [ ] Non-member cannot see team detail (403)
- [ ] Teams list page shows only teams the user belongs to
- [ ] Team detail page renders members and repos
- [ ] UI hides action buttons appropriately based on role

---

## Phase 5 -- Admin & Polish

**Goal:** Admin panel for user management, user settings page (including PAT
management), dark mode, AI test discovery (optional), and UI polish (loading
states, error pages, responsive design).

**Gate:** Admin can manage users. PAT settings work. Dark mode works. UI is polished.

### 5.1 Admin Panel

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | Migration 000011 | `migrations/000011_add_users_is_active.{up,down}.sql` | `ALTER TABLE users ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE`. Index on is_active. |
| 2 | Update user model | `internal/model/user.go` | Add `IsActive bool` field. |
| 3 | Update auth middleware | `internal/middleware/auth.go` | After loading user, check `is_active`. If false, return 401 with `ACCOUNT_DEACTIVATED` code. Delete sessions on deactivation. |
| 4 | Admin DTOs | `internal/dto/admin_dto.go` | AdminUserListRequest (search, role filter, status filter, pagination), UpdateUserRoleRequest, StatsResponse. |
| 5 | Admin service | `internal/service/admin_service.go` | ListUsers (search by username/email, filter by role, filter by is_active, paginated). UpdateUserRole (root only, prevent self-demotion). PromoteToMaintainer (root only). DeactivateUser (set is_active=false, delete all sessions). ReactivateUser. GetStats (total users, active users, total repos, total runs by status, runs last 7 days). |
| 6 | Admin handlers | `internal/handler/admin.go` | `GET /api/v1/admin/users` (list), `PUT /api/v1/admin/users/:id/role` (change role), `PUT /api/v1/admin/users/:id` (activate/deactivate), `GET /api/v1/admin/stats`. All require root or admin role. Role changes require root. |
| 7 | RequireRole middleware | `internal/middleware/auth.go` (extend) | `RequireRole(roles ...string)` middleware factory. Checks user.role against allowed roles. Returns 403 if not matched. |
| 8 | StatsCards component | Frontend | Cards showing: total users, active users, total repos, total test runs, pass rate percentage. |
| 9 | UserTable component | Frontend | Searchable, filterable table. Columns: avatar, username, email, role (dropdown to change), is_active (toggle), created_at. Role change with confirmation modal. Deactivate with confirmation modal. |
| 10 | Admin page | `src/app/(dashboard)/admin/page.tsx` | StatsCards row at top. UserTable below. Only visible to root/admin (sidebar link conditional). |

### 5.2 User Settings

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | User handlers | `internal/handler/user.go` | `GET /api/v1/users/me` (current user profile), `PUT /api/v1/users/me` (update username, email, avatar_url), `PUT /api/v1/users/me/password` (change password, requires current_password). |
| 2 | User service | `internal/service/auth_service.go` (extend) | UpdateProfile (validate unique username/email). ChangePassword (verify current password, hash new password, invalidate other sessions). |
| 3 | Settings page | `src/app/(dashboard)/settings/page.tsx` | Profile form (username, email, avatar URL). Password form (current password, new password, confirm). Save buttons. Success/error toasts. (GitHub PAT management is on the team settings page -- team admin only.) |

### 5.3 UI Polish

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | Dark mode toggle | `src/components/layout/theme-toggle.tsx` | Sun/moon icon toggle. Uses next-themes. Persists preference to localStorage. Respects system preference on first visit. |
| 2 | User menu dropdown | `src/components/layout/topbar.tsx` (extend) | Avatar + name. Dropdown: Settings link, Admin link (if admin/root), Sign Out button. |
| 3 | Notification bell | `src/components/layout/topbar.tsx` (extend) | Bell icon with unread count badge. Placeholder panel for v1 (no backend yet). Shows recent test run completions. |
| 4 | Error pages | `src/app/not-found.tsx`, `src/app/error.tsx` | 404: "Page not found" with illustration and link to dashboard. 500: "Something went wrong" with retry button. |
| 5 | Loading skeletons | All pages | Skeleton placeholders for: repo cards, repo detail, run list, run detail, team cards, team detail, admin user table, settings forms. |
| 6 | Empty states | All list pages | Illustrated empty states with CTAs: "No repositories yet -- Sync from GitHub", "No test suites -- Create your first suite", "No teams -- Create a team". |
| 7 | Toast notifications | Throughout frontend | Success: "Test run triggered", "Team created", "Settings saved". Error: API error messages. Info: "Syncing repositories...". Using Sonner. |
| 8 | Mobile responsiveness | All pages | Sidebar collapses to icon-only on mobile. Repo card grid: 1 col on mobile, 2 on tablet, 3 on desktop. Tables scroll horizontally. Modal is full-screen on mobile. |

### 5.4 AI Test Discovery (Optional)

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | AI discovery service | `internal/service/ai_discovery_service.go` | Scan cloned repo for test files. Use OpenAI API to suggest test suite configurations. Behind `VERDOX_OPENAI_API_KEY` env var -- disabled if key not set. |
| 2 | Discovery endpoint | `internal/handler/repository.go` (extend) | `POST /api/v1/repositories/:id/discover` -- trigger AI scan of cloned repo. Returns suggested test suites. |
| 3 | Frontend: discovery UI | `src/components/repository/test-discovery.tsx` | "Discover Tests" button on repo detail page. Shows AI-suggested suites with accept/dismiss actions. Only visible when `VERDOX_OPENAI_API_KEY` is configured. |

> **Note:** Webhook integration (GitHub push events triggering test runs) is deferred to v2.

### Phase 5 Confirmation Gate

- [ ] Migration 000011 adds `is_active` column to users
- [ ] `GET /api/v1/admin/users` returns paginated users with search and filters
- [ ] root can change user roles via `PUT /api/v1/admin/users/:id/role`
- [ ] root can promote user to maintainer
- [ ] root can deactivate a user; deactivated user's sessions are deleted
- [ ] Deactivated user cannot login or use existing tokens (401 ACCOUNT_DEACTIVATED)
- [ ] root can reactivate a user
- [ ] `GET /api/v1/admin/stats` returns system statistics
- [ ] Admin page renders with stats cards and user table
- [ ] Admin page is only accessible to root/admin roles
- [ ] Settings page loads current user data
- [ ] Settings page updates username/email successfully
- [ ] Settings page changes password (requires current password)
- [ ] Team settings PAT section allows team admins to store/update/revoke the team's GitHub PAT
- [ ] Dark mode toggle works and persists across page loads
- [ ] Dark mode colors apply correctly across all pages
- [ ] 404 page renders for unknown routes
- [ ] Error boundary catches runtime errors and shows 500 page
- [ ] Loading skeletons appear during data fetches
- [ ] Empty states render when lists are empty
- [ ] Toast notifications fire for success/error actions
- [ ] Layout is usable on mobile (375px width)
- [ ] AI test discovery suggests suites when `VERDOX_OPENAI_API_KEY` is set (optional)

---

## Phase 6 -- Deployment & Monitoring

**Goal:** Production-ready deployment configuration, monitoring, documentation
finalization, and CI/CD pipeline. The system can be deployed to a VPS with a
single `docker compose up -d`.

**Gate:** Can deploy to a VPS with `docker compose up -d`.

### 6.1 Production Configuration

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | Finalize docker-compose.yml | `docker-compose.yml` | Review all service configs. Set restart policies (`unless-stopped`). Set memory limits. Remove debug ports. Pin image versions. |
| 2 | SSL/TLS in Nginx | `nginx/conf.d/default.conf` | Let's Encrypt certificate paths. HTTP -> HTTPS redirect. TLS 1.2+. Strong cipher suite. OCSP stapling. |
| 3 | Security headers | `nginx/conf.d/default.conf` | `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `X-XSS-Protection: 1; mode=block`, `Strict-Transport-Security`, `Content-Security-Policy`, `Referrer-Policy: strict-origin-when-cross-origin`. |
| 4 | Production .env template | `.env.example` update | Document all production-required variables. Mark optional vs required. Add comments for secure value generation. |
| 5 | Postgres production config | `docker-compose.yml` | Persistent volume mount. `shared_buffers`, `work_mem`, `max_connections` tuning for VPS (2-4GB RAM). Backup considerations. |
| 6 | Redis production config | `docker-compose.yml` | Persistent volume (AOF). Memory limit. Eviction policy (allkeys-lru). |

### 6.2 Monitoring

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | Structured logging audit | All backend handlers | Verify every handler logs: request received, success/failure, duration. Use zerolog fields: `user_id`, `resource_id`, `action`. JSON output in production. |
| 2 | Health check verification | `internal/handler/health.go` | `/health` (liveness, always 200). `/health/ready` (readiness, checks Postgres + Redis + Docker socket). |
| 3 | Docker Compose healthchecks | `docker-compose.yml` | Postgres: `pg_isready`. Redis: `redis-cli ping`. Backend: `curl /health`. Frontend: `curl /`. Nginx: `curl /health`. Define intervals, timeouts, retries. |
| 4 | Log rotation | `docker-compose.yml` | Docker logging driver: `json-file`, `max-size: 10m`, `max-file: 5` for all services. |
| 5 | Graceful shutdown | `cmd/server/main.go` | Signal handling (SIGINT, SIGTERM). Drain HTTP connections (30s timeout). Wait for in-progress test runs. Close DB pool. Close Redis connection. |

### 6.3 Documentation

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | Update SETUP.md | `docs/SETUP.md` | Verify all instructions still work. Update any changed env vars. Test from a clean clone. |
| 2 | Update DEPLOYMENT.md | `docs/DEPLOYMENT.md` | Finalize VPS deployment steps. SSL setup instructions. Backup procedure. Monitoring recommendations. |
| 3 | Verify GITHUB-PAT-GUIDE.md | `docs/GITHUB-PAT-GUIDE.md` | Ensure PAT creation and maintenance instructions are current. Referenced from SETUP.md, USAGE-GUIDE.md, and team settings UI. |
| 4 | VPS deployment test | Manual | Deploy to a clean VPS. Run through DEPLOYMENT.md steps. Fix any issues found. |

### 6.4 CI/CD (GitHub Actions)

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | Lint and test workflow | `.github/workflows/ci.yml` | Trigger on push and PR to main. Steps: checkout, setup Go 1.25, `golangci-lint run`, `go test -race ./...`, setup Node 22, `npm ci`, `npm run lint`, `npm test`. |
| 2 | Build Docker images workflow | `.github/workflows/build.yml` | Trigger on push to main. Build backend, frontend, runner images. Push to GitHub Container Registry (ghcr.io). Tag with commit SHA and `latest`. |
| 3 | Deploy workflow (optional) | `.github/workflows/deploy.yml` | Trigger manually or on release tag. SSH to VPS. Pull latest images. Run `docker compose up -d`. Verify health endpoints. |

### Phase 6 Confirmation Gate

- [ ] `docker compose up -d` starts all services in production mode
- [ ] Nginx serves HTTPS with valid certificate (or self-signed for testing)
- [ ] Security headers present on all responses
- [ ] Health endpoints respond: `/health` (200), `/health/ready` (200)
- [ ] All services pass Docker healthchecks
- [ ] Logs are structured JSON
- [ ] Log rotation is configured (max 10MB, 5 files per service)
- [ ] Graceful shutdown completes within 30 seconds
- [ ] Backend handles SIGTERM without dropping in-flight requests
- [ ] CI workflow passes: lint + test (backend and frontend)
- [ ] Docker images build successfully in CI
- [ ] Can deploy to a clean VPS following DEPLOYMENT.md
- [ ] Application works end-to-end on VPS (signup, sync, run tests, view results)

---

## Phase 7 -- Testing & Hardening

**Goal:** Comprehensive test coverage and reliability improvements.

**Gate:** All tests pass. Core flows have integration test coverage.

### 7.1 Backend Unit Tests

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | Auth service tests | `internal/service/auth_service_test.go` | Signup, login, token refresh, password reset. |
| 2 | Repository service tests | `internal/service/repository_service_test.go` | Add, clone, fetch. |
| 3 | Team service tests | `internal/service/team_service_test.go` | Create, join request, approve/reject. |
| 4 | Test runner service tests | `internal/service/test_service_test.go` | Queue, execute, parse. |
| 5 | Admin service tests | `internal/service/admin_service_test.go` | User management, stats, role changes. |

### 7.2 Backend Integration Tests

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | Database integration tests | `internal/repository/*_test.go` | With test PostgreSQL instance. |
| 2 | Redis integration tests | `internal/queue/*_test.go` | With test Redis instance. |
| 3 | API endpoint tests | `internal/handler/*_test.go` | Using httptest. |

### 7.3 Frontend Tests

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | Component unit tests | `src/components/**/*.test.tsx` | React Testing Library. |
| 2 | Page-level tests | `src/app/**/*.test.tsx` | Key page rendering and interaction. |
| 3 | Form validation tests | `src/components/**/*.test.tsx` | Auth forms, settings forms, team forms. |

### 7.4 End-to-End Flow Tests

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | Core user flow | E2E test | Signup -> join team -> add repo -> run test -> view results. |
| 2 | Admin flow | E2E test | Promote moderator, manage users. |
| 3 | Team management flow | E2E test | Create, join, approve. |

### 7.5 CI Pipeline

| # | Task | Deliverable(s) | Notes |
|---|------|----------------|-------|
| 1 | GitHub Actions workflow | `.github/workflows/ci.yml` | Lint + test on PR. |
| 2 | Docker build verification | `.github/workflows/build.yml` | Verify images build successfully. |
| 3 | Migration verification | CI step | Run migrate-up/migrate-down in CI. |

### Phase 7 Confirmation Gate

- [ ] Backend unit tests pass
- [ ] Integration tests pass with real DB/Redis
- [ ] Frontend component tests pass
- [ ] At least one E2E flow test works
- [ ] CI pipeline runs on PR

---

## Cross-Phase Concerns

These practices apply across every phase and are not deferred.

### Testing

- Testing is deferred to Phase 7. During Phases 1-6, focus on working software. Manual testing during development is sufficient.
- Test file naming: `*_test.go` (Go), `*.test.tsx` / `*.test.ts` (frontend).
- Target: 70%+ backend coverage, meaningful (not exhaustive) frontend coverage.

### Migration Discipline

- Never modify a migration that has been applied. Create a new migration instead.
- Keep migrations backward-compatible during development (additive changes only when possible).
- Test both `up` and `down` for every migration before committing.

### Environment Variables

- Keep `.env.example` updated every time a new variable is introduced.
- Document the variable's purpose, default value, and whether it is required.
- Never commit real secrets to version control.

### Git Workflow

- Work on feature branches, one per task group (e.g., `feat/auth-system`, `feat/repo-management`).
- Each PR should be reviewable in isolation.
- Write descriptive commit messages (imperative mood, reference the phase).
- Merge to `main` via PR after self-review.

### Error Handling

- Backend: use the standardized error response format from `pkg/response`. Never leak stack traces in production.
- Frontend: catch API errors and display user-friendly toast messages. Log detailed errors to console in development only.

### Security

- All user input is validated before processing (backend validation tags + custom validators).
- SQL queries use parameterized statements (sqlx named params or positional `$1`).
- Passwords are bcrypt-hashed (cost 12), never stored or logged in plaintext.
- JWT secrets are minimum 256 bits, loaded from environment variables.
- CORS is configured per-environment (strict in production).
- Rate limiting is applied to all public endpoints.

---

## Phase Summary

| Phase | Focus | Key Deliverables | Endpoints |
|-------|-------|-----------------|-----------|
| 1 | Foundation | Scaffolding, Docker, DB, Auth, Dashboard shell | 8 (auth + health) |
| 2 | Repositories | GitHub PAT, repo addition + clone, branch/commit browsing | 8 (repo endpoints) |
| 3 | Test Execution | Suite CRUD, job queue, DinD runner, results | 9 (suite + run endpoints) |
| 4 | Teams | Team CRUD, members, repo assignment, permissions | 9 (team endpoints) |
| 5 | Admin & Polish | Admin panel, settings + PAT, dark mode, AI discovery | 7 (admin + user + discovery) |
| 6 | Deployment | Production config, monitoring, CI/CD | 0 (infra only) |
| 7 | Testing & Hardening | Unit tests, integration tests, E2E tests, CI pipeline | 0 (tests only) |
| **Total** | | | **41 endpoints** |
