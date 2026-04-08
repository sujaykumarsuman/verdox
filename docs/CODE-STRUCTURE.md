# Verdox -- Code Structure & Directory Organization

This document is the canonical reference for how the Verdox codebase is organized. Every directory and file listed here has a single, well-defined responsibility. Follow the conventions described below when adding new code.

---

## 1. Top-Level Project Structure

```
verdox/
├── backend/                  # Go API server (Echo v4, Go 1.26+)
├── frontend/                 # Next.js 15 web app (App Router, TypeScript, Tailwind CSS)
├── docker/                   # Dockerfiles and Docker-specific configs
├── nginx/                    # Nginx reverse-proxy configuration
├── docs/                     # Project documentation (you are here)
├── scripts/                  # Utility and automation scripts (DB seed, CI helpers)
├── docker-compose.yml        # Production Docker Compose manifest
├── docker-compose.dev.yml    # Development overrides (hot reload, debug ports)
├── Makefile                  # Project-wide make targets (see Section 8)
├── .env.example              # Template for required environment variables
├── .gitignore                # Git ignore rules for both backend and frontend
└── README.md                 # Project overview and quick-start instructions
```

**Rules:**

- Production configuration lives in `docker-compose.yml`. Development-only overrides go in `docker-compose.dev.yml` and are applied with `docker compose -f docker-compose.yml -f docker-compose.dev.yml up`.
- The `scripts/` directory is for one-off or CI-related scripts. Application code never imports from it.
- `.env.example` must stay in sync with every environment variable the application reads. Never commit a real `.env` file.

---

## 2. Backend Directory Structure (Go)

The backend follows standard Go project layout conventions with a clean separation between the HTTP transport layer, business logic, and data access.

```
backend/
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go         # Configuration loading
│   ├── handler/              # HTTP route handlers (transport layer)
│   │   ├── auth.go           # Authentication endpoints
│   │   ├── repository.go     # Repository management endpoints
│   │   ├── test_suite.go     # Test suite endpoints
│   │   ├── test_run.go       # Test run endpoints
│   │   ├── team.go           # Team management endpoints
│   │   ├── admin.go          # Admin-only endpoints
│   │   ├── user.go           # User profile endpoints
│   │   ├── webhook.go        # Inbound webhook receiver
│   │   ├── join_request.go   # Join request endpoints
│   │   └── discovery.go      # AI discovery endpoints
│   ├── middleware/            # Echo middleware stack
│   │   ├── auth.go           # JWT authentication middleware
│   │   ├── cors.go           # CORS policy middleware
│   │   ├── logger.go         # Structured request logging middleware
│   │   ├── ratelimit.go      # Rate limiting middleware
│   │   └── recover.go        # Panic recovery middleware
│   ├── model/                # Domain data models
│   │   ├── user.go           # User and role structs
│   │   ├── repository.go     # Repository struct
│   │   ├── team.go           # Team and membership structs
│   │   ├── test_suite.go     # Test suite struct
│   │   ├── test_run.go       # Test run struct
│   │   ├── test_result.go    # Individual test result struct
│   │   ├── session.go        # Auth session struct
│   │   ├── join_request.go   # Team join request model
│   │   └── discovery.go      # Test discovery model
│   ├── repository/           # Database access layer
│   │   ├── user_repo.go      # User table queries
│   │   ├── repository_repo.go # Repository table queries
│   │   ├── team_repo.go      # Team and membership table queries
│   │   ├── test_suite_repo.go # Test suite table queries
│   │   ├── test_run_repo.go  # Test run table queries
│   │   ├── test_result_repo.go # Test result table queries
│   │   ├── session_repo.go   # Session table queries
│   │   ├── join_request_repo.go # Join request data access
│   │   └── discovery_repo.go # Discovery data access
│   ├── service/              # Business logic layer
│   │   ├── auth_service.go   # Signup, login, token refresh, password reset logic
│   │   ├── repository_service.go # Repo CRUD, sync, branch/commit operations
│   │   ├── test_service.go   # Test suite CRUD, run triggering, result aggregation
│   │   ├── team_service.go   # Team CRUD, member management, repo assignment
│   │   ├── admin_service.go  # User management, system statistics
│   │   └── github_service.go # GitHub API integration (clone, webhook verification, PAT encrypt/decrypt/validate)
│   ├── runner/               # Fork-based GHA test execution engine
│   │   ├── runner.go         # Worker pool: polls queue, dispatches jobs to executor
│   │   ├── fork_gha_executor.go  # Fork management, workflow dispatch, GHA run tracking
│   │   ├── gha_poller.go     # Polls GitHub Actions API for run completion, downloads artifacts
│   │   ├── executor_interface.go # Executor plugin interface and ExecutionJob struct
│   │   └── parser.go         # Test output parsing for flat result formats
│   ├── queue/                # Job queue abstraction
│   │   └── queue.go          # Redis-backed push/pop, retry, and dead-letter logic
│   ├── analyzer/             # AI-powered test discovery
│   │   ├── scanner.go        # Repo scanning logic
│   │   └── openai.go         # OpenAI API client
│   └── dto/                  # Data Transfer Objects
│       ├── auth_dto.go       # Auth request/response shapes
│       ├── repository_dto.go # Repository request/response shapes
│       ├── test_dto.go       # Test suite/run request/response shapes
│       ├── team_dto.go       # Team request/response shapes
│       └── admin_dto.go      # Admin request/response shapes
├── pkg/                      # Shared utility packages (importable externally)
│   ├── jwt/
│   │   └── jwt.go            # JWT token generation and validation
│   ├── hash/
│   │   └── hash.go           # Bcrypt password hashing and SHA-256 utilities
│   ├── response/
│   │   └── response.go       # Standardized JSON response envelope helpers
│   ├── validator/
│   │   └── validator.go      # Custom validation rules for go-playground/validator
│   └── logger/
│       └── logger.go         # Zerolog initialization and configuration
├── migrations/               # SQL migration files (golang-migrate format)
│   ├── 000001_create_enum_types.up.sql
│   ├── 000001_create_enum_types.down.sql
│   ├── 000002_create_users.up.sql
│   ├── 000002_create_users.down.sql
│   ├── 000003_create_sessions.up.sql
│   ├── 000003_create_sessions.down.sql
│   ├── 000004_create_teams.up.sql
│   ├── 000004_create_teams.down.sql
│   ├── 000005_create_team_members.up.sql
│   ├── 000005_create_team_members.down.sql
│   ├── 000006_create_repositories.up.sql
│   ├── 000006_create_repositories.down.sql
│   ├── 000007_create_team_repositories.up.sql
│   ├── 000007_create_team_repositories.down.sql
│   ├── 000008_create_test_suites.up.sql
│   ├── 000008_create_test_suites.down.sql
│   ├── 000009_create_test_runs.up.sql
│   ├── 000009_create_test_runs.down.sql
│   ├── 000010_create_test_results.up.sql
│   └── 000010_create_test_results.down.sql
├── go.mod                    # Go module definition and dependency versions
├── go.sum                    # Dependency checksum database
└── Makefile                  # Backend-specific make targets
```

### Detailed File Responsibilities

#### `cmd/server/main.go`

Application entry point. Performs four things in order:

1. Loads configuration via `internal/config`.
2. Connects to PostgreSQL and Redis.
3. Registers all Echo routes and middleware.
4. Starts the HTTP server and the background test runner.

No business logic lives here.

#### `internal/config/config.go`

Uses Viper to load configuration from environment variables and optional config files. Exposes a single `Load()` function that returns a typed `Config` struct. All configuration keys are documented in `.env.example`.

#### `internal/handler/` -- HTTP Transport Layer

Each file in this package corresponds to a resource domain. Handlers are responsible for:

- Parsing and validating HTTP requests (binding JSON, reading path params).
- Calling the appropriate service method.
- Serializing the response using `pkg/response`.

Handlers never access the database directly. They never contain business rules.

| File | Endpoints |
|------|-----------|
| `auth.go` | `POST /auth/signup`, `POST /auth/login`, `POST /auth/logout`, `POST /auth/refresh`, `POST /auth/forgot-password`, `POST /auth/reset-password` |
| `repository.go` | `GET /repositories`, `POST /repositories`, `GET /repositories/:id`, `PUT /repositories/:id`, `DELETE /repositories/:id`, `POST /repositories/:id/sync`, `GET /repositories/:id/branches`, `GET /repositories/:id/commits` |
| `test_suite.go` | `GET /repositories/:id/suites`, `POST /repositories/:id/suites`, `PUT /suites/:id`, `DELETE /suites/:id` |
| `test_run.go` | `POST /suites/:id/runs`, `GET /runs`, `GET /runs/:id`, `GET /runs/:id/logs`, `POST /runs/:id/cancel` |
| `team.go` | `GET /teams`, `POST /teams`, `GET /teams/:id`, `PUT /teams/:id`, `DELETE /teams/:id`, `POST /teams/:id/members`, `DELETE /teams/:id/members/:userId`, `POST /teams/:id/repositories`, `DELETE /teams/:id/repositories/:repoId`, `PUT /teams/:id/pat`, `GET /teams/:id/pat/validate`, `DELETE /teams/:id/pat` |
| `admin.go` | `GET /admin/users`, `PUT /admin/users/:id/role`, `GET /admin/stats` |
| `user.go` | `GET /users/me`, `PUT /users/me`, `PUT /users/me/password` |
| `webhook.go` | `POST /webhooks/github` |

#### `internal/middleware/` -- Echo Middleware

| File | Purpose |
|------|---------|
| `auth.go` | Extracts the JWT from the `Authorization` header, validates it, loads the `User` model, and injects it into the Echo context. Returns 401 on failure. |
| `cors.go` | Configures allowed origins, methods, and headers for CORS preflight requests. |
| `logger.go` | Logs every request with method, path, status, latency, and request ID using zerolog. |
| `ratelimit.go` | Enforces per-IP or per-user rate limits backed by Redis sliding window counters. |
| `recover.go` | Catches panics in downstream handlers and returns a 500 response with a request ID for debugging. |

#### `internal/model/` -- Domain Models

Pure data structs that map 1:1 to database tables. Each struct includes:

- `db` tags for sqlx column mapping.
- `json` tags for serialization.

Models have no methods with side effects. Computed fields (e.g., `FullName()`) are acceptable.

#### `internal/repository/` -- Data Access Layer

One file per database table. Each file defines an interface and its implementation:

```go
type UserRepository interface {
    Create(ctx context.Context, user *model.User) error
    GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
    GetByEmail(ctx context.Context, email string) (*model.User, error)
    Update(ctx context.Context, user *model.User) error
    Delete(ctx context.Context, id uuid.UUID) error
    List(ctx context.Context, offset, limit int) ([]model.User, int, error)
}
```

Repository methods deal exclusively with SQL queries. They never call other repositories, services, or external APIs. Transaction management is handled at the service layer by passing a `*sqlx.Tx`.

#### `internal/service/` -- Business Logic Layer

Services orchestrate operations across multiple repositories, enforce authorization rules, and encapsulate domain logic:

| File | Responsibility |
|------|---------------|
| `auth_service.go` | Hashes passwords, issues/refreshes JWTs, manages sessions, sends password reset emails. |
| `repository_service.go` | Validates ownership, clones repos via GitHub API, syncs branches and commits. |
| `test_service.go` | Creates test suites, enqueues test runs on the Redis queue, aggregates results from individual test cases. |
| `team_service.go` | Enforces membership limits, manages roles within teams, assigns repositories to teams. Manages team-level GitHub PAT (encrypt, decrypt, validate, rotate). |
| `admin_service.go` | Verifies admin role, computes system-wide statistics, manages user roles. |
| `github_service.go` | Authenticates with GitHub, clones repositories, verifies webhook signatures. |

#### `internal/runner/` -- Test Execution Engine

| File | Responsibility |
|------|---------------|
| `runner.go` | Long-running goroutine that polls the Redis queue for pending jobs. Manages concurrency limits and graceful shutdown. |
| `executor.go` | Creates Docker containers from the appropriate test image, mounts the repository code, streams stdout/stderr, enforces timeouts, and cleans up containers on completion or failure. |
| `parser.go` | Parses structured test output. Supports Go (`go test -json`), pytest (`--tb=short`), and Jest (`--json`) formats. Extracts pass/fail/skip counts and individual test case results. |

#### `internal/queue/queue.go`

Redis-backed job queue. Provides:

- `Push(ctx, job)` -- enqueue a new test run job.
- `Pop(ctx)` -- blocking pop from the queue with timeout.
- `Ack(ctx, jobID)` -- mark a job as completed.
- `Fail(ctx, jobID, err)` -- move a job to the dead-letter queue.
- `Retry(ctx, jobID)` -- re-enqueue a failed job.

#### `internal/dto/` -- Data Transfer Objects

Request and response structs used by handlers. DTOs are separate from models to:

- Decouple the API contract from the database schema.
- Include validation tags (`validate:"required,email"`).
- Omit internal fields (e.g., password hash) from responses.

#### `pkg/` -- Shared Utility Packages

Packages under `pkg/` have zero imports from `internal/`. They can be extracted into standalone libraries without modification.

| Package | Purpose |
|---------|---------|
| `pkg/jwt` | Generates HS256 access and refresh tokens. Validates tokens and extracts claims. |
| `pkg/hash` | Wraps bcrypt for password hashing/comparison. Provides SHA-256 helper for token hashing. |
| `pkg/response` | Defines a `JSON(c echo.Context, status int, data interface{})` helper and a standard error response format (`{"error": {"code": "...", "message": "..."}}`). |
| `pkg/validator` | Registers custom validation rules (e.g., `strong_password`, `github_url`) with the go-playground/validator instance. |
| `pkg/logger` | Initializes zerolog with JSON output, log level from config, and caller information. |

#### `migrations/`

SQL migration files managed by [golang-migrate](https://github.com/golang-migrate/migrate). Each migration is a pair of `.up.sql` and `.down.sql` files:

| Migration | Purpose |
|-----------|---------|
| `000001` | Creates enum types (`user_role`, `run_status`, `test_status`) |
| `000002` | Creates `users` table |
| `000003` | Creates `sessions` table |
| `000004` | Creates `teams` table |
| `000005` | Creates `team_members` join table |
| `000006` | Creates `repositories` table |
| `000007` | Creates `team_repositories` join table |
| `000008` | Creates `test_suites` table |
| `000009` | Creates `test_runs` table |
| `000010` | Creates `test_results` table |

**Migration naming rule:** `NNNNNN_description.{up,down}.sql` where `NNNNNN` is a zero-padded sequence number.

---

## 3. Frontend Directory Structure (Next.js)

The frontend uses the Next.js 15 App Router with route groups for layout isolation.

```
frontend/
├── src/
│   ├── app/                  # Next.js App Router (file-system routing)
│   │   ├── layout.tsx        # Root layout: providers, fonts, theme wrapper
│   │   ├── page.tsx          # Landing page (/)
│   │   ├── (auth)/           # Auth route group (no sidebar, centered layout)
│   │   │   ├── layout.tsx    # Auth-specific layout (centered card on gradient bg)
│   │   │   ├── login/
│   │   │   │   └── page.tsx  # Login page (/login)
│   │   │   ├── signup/
│   │   │   │   └── page.tsx  # Signup page (/signup)
│   │   │   ├── forgot-password/
│   │   │   │   └── page.tsx  # Forgot password page (/forgot-password)
│   │   │   └── reset-password/
│   │   │       └── page.tsx  # Reset password page (/reset-password?token=xxx)
│   │   ├── (dashboard)/      # Dashboard route group (sidebar + topbar layout)
│   │   │   ├── layout.tsx    # Dashboard layout: Sidebar + TopBar + main content area
│   │   │   ├── dashboard/
│   │   │   │   └── page.tsx  # Repository list dashboard (/dashboard)
│   │   │   ├── repositories/
│   │   │   │   └── [id]/
│   │   │   │       ├── page.tsx      # Repository detail (/repositories/:id)
│   │   │   │       └── runs/
│   │   │   │           └── [runId]/
│   │   │   │               └── page.tsx  # Test run detail (/repositories/:id/runs/:runId)
│   │   │   ├── teams/
│   │   │   │   ├── page.tsx          # Teams list (/teams)
│   │   │   │   ├── discover/
│   │   │   │   │   └── page.tsx      # Team discovery page (/teams/discover)
│   │   │   │   └── [id]/
│   │   │   │       ├── page.tsx      # Team detail (/teams/:id)
│   │   │   │       └── requests/
│   │   │   │           └── page.tsx  # Join requests page (/teams/:id/requests)
│   │   │   ├── admin/
│   │   │   │   └── page.tsx          # Admin panel (/admin)
│   │   │   └── settings/
│   │   │       └── page.tsx          # User settings (/settings)
│   │   └── api/              # Next.js API routes (BFF pattern, if needed)
│   ├── components/           # Shared UI components
│   │   ├── ui/               # Base UI primitives (design system atoms)
│   │   │   ├── button.tsx    # Button with variants: primary, secondary, ghost, danger
│   │   │   ├── input.tsx     # Text input with label, error state, and icon support
│   │   │   ├── card.tsx      # Container card with optional header and footer
│   │   │   ├── badge.tsx     # Status badge (success, warning, error, neutral)
│   │   │   ├── modal.tsx     # Dialog overlay with focus trap and escape-to-close
│   │   │   ├── dropdown.tsx  # Dropdown menu with keyboard navigation
│   │   │   ├── table.tsx     # Data table with sortable columns and pagination
│   │   │   ├── progress.tsx  # Progress bar (determinate and indeterminate)
│   │   │   ├── toast.tsx     # Toast notification (success, error, info)
│   │   │   └── skeleton.tsx  # Loading skeleton placeholder
│   │   ├── layout/           # Layout shell components
│   │   │   ├── sidebar.tsx   # Collapsible sidebar with nav links and active state
│   │   │   ├── topbar.tsx    # Top navigation bar with user menu and breadcrumbs
│   │   │   └── theme-toggle.tsx # Dark/light mode toggle switch
│   │   ├── auth/             # Authentication components
│   │   │   ├── login-form.tsx       # Login form with validation and error display
│   │   │   ├── signup-form.tsx      # Signup form with password strength indicator
│   │   │   └── protected-route.tsx  # Route guard: redirects to /login if unauthenticated
│   │   ├── repository/       # Repository domain components
│   │   │   ├── repo-card.tsx        # Repository summary card for the dashboard grid
│   │   │   ├── branch-selector.tsx  # Branch dropdown with search
│   │   │   └── commit-list.tsx      # Scrollable commit history list
│   │   ├── test/             # Test domain components
│   │   │   ├── suite-card.tsx       # Test suite summary with run button
│   │   │   ├── run-list.tsx         # List of test runs with status and duration
│   │   │   ├── run-detail.tsx       # Single run detail with result breakdown
│   │   │   ├── result-row.tsx       # Individual test case row (pass/fail/skip icon)
│   │   │   └── log-viewer.tsx       # ANSI-aware terminal log output viewer
│   │   ├── team/             # Team domain components
│   │   │   ├── team-card.tsx        # Team summary card with member count
│   │   │   ├── member-list.tsx      # Team member table with role badges
│   │   │   ├── repo-assign.tsx      # Repository assignment multi-select
│   │   │   ├── team-discovery-card.tsx  # Team discovery card
│   │   │   ├── join-request-row.tsx # Join request row component
│   │   │   └── team-pat-form.tsx    # Team-level GitHub PAT form (admin only)
│   │   └── settings/         # Settings domain components
│   │       └── (empty -- PAT form moved to team/)
│   ├── lib/                  # Client-side utilities
│   │   ├── api.ts            # Fetch wrapper: base URL, auth headers, error handling
│   │   ├── auth.tsx          # AuthContext provider: login, logout, token refresh
│   │   └── utils.ts          # General helpers: cn(), formatDate(), pluralize()
│   ├── hooks/                # Custom React hooks
│   │   ├── use-auth.ts       # Accesses AuthContext, returns user + auth actions
│   │   ├── use-repos.ts      # Fetches and caches repository data
│   │   └── use-teams.ts      # Fetches and caches team data
│   ├── types/                # TypeScript type definitions
│   │   ├── user.ts           # User, UserRole, Session types
│   │   ├── repository.ts     # Repository, Branch, Commit types
│   │   ├── test.ts           # TestSuite, TestRun, TestResult, RunStatus types
│   │   └── team.ts           # Team, TeamMember, TeamRole types
│   └── styles/
│       └── globals.css       # Tailwind directives, CSS custom properties, base resets
├── public/                   # Static assets served at /
│   ├── logo.svg              # Verdox logo (used in sidebar and landing)
│   └── favicon.ico           # Browser tab icon
├── next.config.ts            # Next.js configuration (rewrites, env exposure)
├── tailwind.config.ts        # Tailwind theme: colors, fonts, spacing, breakpoints
├── tsconfig.json             # TypeScript compiler options and path aliases
├── package.json              # Dependencies and npm scripts
└── .eslintrc.json            # ESLint rules (extends next/core-web-vitals)
```

### Route Group Explanation

Next.js route groups (directories wrapped in parentheses) share a layout without affecting the URL path:

- **`(auth)/`** -- Pages that use a minimal, centered layout with no sidebar. The URL is `/login`, not `/(auth)/login`.
- **`(dashboard)/`** -- Pages that use the full dashboard layout with sidebar and topbar. The URL is `/dashboard`, not `/(dashboard)/dashboard`.

### Component Organization Rules

1. **`ui/`** components are generic primitives. They accept props, render markup, and have no knowledge of the Verdox domain. They never call API functions.
2. **Domain components** (`repository/`, `test/`, `team/`) compose `ui/` primitives with domain-specific data. They may call hooks but never call `lib/api.ts` directly.
3. **Hooks** call `lib/api.ts` and manage data fetching, caching, and mutation state.
4. **`lib/api.ts`** is the single point of contact with the backend. All API calls go through it.

---

## 4. Docker & Infrastructure Files

```
docker/
├── backend.Dockerfile        # Multi-stage Go build: golang:1.26-alpine -> alpine:3.21
└── frontend.Dockerfile       # Multi-stage Node build: node:22-alpine -> node:22-alpine

nginx/
├── nginx.conf                # Global Nginx settings (worker_processes, log format)
└── conf.d/
    └── default.conf          # Server block: TLS, proxy_pass rules, static caching
```

### Dockerfile Build Stages

**`backend.Dockerfile`**

| Stage | Base Image | Purpose |
|-------|-----------|---------|
| `builder` | `golang:1.26-alpine` | Downloads dependencies, compiles the binary with CGO disabled. |
| `runtime` | `alpine:3.21` | Copies only the compiled binary. Adds CA certificates and a non-root user. Final image is ~15 MB. |

**`frontend.Dockerfile`**

| Stage | Base Image | Purpose |
|-------|-----------|---------|
| `deps` | `node:22-alpine` | Installs `node_modules` from `package-lock.json`. |
| `builder` | `node:22-alpine` | Runs `next build` to produce the `.next` standalone output. |
| `runtime` | `node:22-alpine` | Copies the standalone build and `public/` assets. Runs as a non-root user. |

### Nginx Proxy Rules

`nginx/conf.d/default.conf` defines the following routing:

| Path | Upstream | Notes |
|------|----------|-------|
| `/api/*` | `http://backend:8080` | Strips the `/api` prefix before proxying. |
| `/webhooks/*` | `http://backend:8080` | Passes through without prefix stripping. |
| `/*` (everything else) | `http://frontend:3000` | Serves the Next.js application. |

---

## 5. File Naming Conventions

| Context | Convention | Example |
|---------|-----------|---------|
| Go source files | `snake_case` | `auth_service.go` |
| Go packages | lowercase, single word | `handler`, `service`, `model` |
| Go test files | `snake_case` with `_test` suffix | `auth_service_test.go` |
| TypeScript / TSX files | `kebab-case` | `repo-card.tsx` |
| CSS files | `kebab-case` | `globals.css` |
| SQL migrations | zero-padded number + `snake_case` | `000001_create_users.up.sql` |
| Environment variables | `SCREAMING_SNAKE_CASE` | `DATABASE_URL` |
| Docker Compose services | lowercase with hyphens | `verdox-backend` |
| Docker image tags | lowercase with hyphens | `verdox-backend:latest` |

---

## 6. Import Ordering

### Go

Group imports with a blank line between each group, in this order:

```go
import (
    // 1. Standard library
    "context"
    "fmt"
    "net/http"

    // 2. External dependencies
    "github.com/labstack/echo/v4"
    "github.com/rs/zerolog/log"

    // 3. Internal packages
    "github.com/sujaykumarsuman/verdox/backend/internal/model"
    "github.com/sujaykumarsuman/verdox/backend/internal/service"
)
```

### TypeScript

Order imports top to bottom, with a blank line between groups:

```typescript
// 1. React and Next.js
import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';

// 2. External libraries
import { clsx } from 'clsx';

// 3. Components
import { Button } from '@/components/ui/button';
import { RepoCard } from '@/components/repository/repo-card';

// 4. Hooks
import { useRepos } from '@/hooks/use-repos';

// 5. Lib utilities
import { api } from '@/lib/api';

// 6. Types
import type { Repository } from '@/types/repository';

// 7. Styles (rare -- usually only in layout.tsx)
import '@/styles/globals.css';
```

---

## 7. Package Responsibility Rules

These rules enforce a strict dependency direction and prevent circular imports.

```
handler -> service -> repository -> database
   |          |
   v          v
  dto       model
   |          ^
   v          |
  pkg --------+
```

### Dependency Rules

| Package | May Import | Must Not Import |
|---------|-----------|-----------------|
| `handler` | `service`, `dto`, `model`, `pkg` | `repository`, `queue`, `runner` |
| `service` | `repository`, `model`, `dto`, `pkg`, `queue`, other services | `handler` |
| `repository` | `model`, `pkg` | `handler`, `service`, `dto` |
| `model` | standard library only | any internal package |
| `dto` | `model` (for embedding/conversion), standard library | `handler`, `service`, `repository` |
| `pkg` | standard library, external libraries | any `internal` package |
| `runner` | `service`, `queue`, `model`, `pkg` | `handler` |
| `queue` | `model`, `pkg` | `handler`, `service`, `repository` |

### Key Principles

- **handler** is a thin translation layer. It converts HTTP requests to service calls and service responses to HTTP responses. No business logic.
- **service** is where decisions happen. Authorization checks, validation beyond struct tags, multi-step orchestration, and transaction boundaries all live here.
- **repository** is a data gateway. One method per query. No joins across unrelated tables (compose at the service layer instead).
- **model** is pure data. Structs with tags. No database connections, no HTTP imports, no side effects.
- **dto** defines the API contract. It may embed or reference model types but exists independently so the API shape can evolve without changing the database schema.
- **pkg** is self-contained. It can be copied into another project and must work without any Verdox-specific code.

---

## 8. Root Makefile Targets

```makefile
# ============================================================
# Development
# ============================================================
dev:                 ## Run full stack with docker-compose.dev.yml (hot reload enabled)
dev-backend:         ## Run backend only with air (hot reload)
dev-frontend:        ## Run frontend only with next dev (hot reload)

# ============================================================
# Build
# ============================================================
build:               ## Build all Docker images for production
build-backend:       ## Build backend Docker image
build-frontend:      ## Build frontend Docker image

# ============================================================
# Database
# ============================================================
migrate-up:          ## Run all pending database migrations
migrate-down:        ## Rollback the last applied migration
migrate-create:      ## Create a new migration pair (usage: make migrate-create NAME=add_index)
seed:                ## Seed the database with initial/demo data

# ============================================================
# Docker
# ============================================================
up:                  ## Start the production stack (detached)
down:                ## Stop and remove all containers
logs:                ## Tail logs for all services (Ctrl+C to stop)

# ============================================================
# Testing
# ============================================================
test:                ## Run all tests (backend + frontend)
test-backend:        ## Run Go tests with race detector (go test -race ./...)
test-frontend:       ## Run frontend tests (jest / vitest)
lint:                ## Run all linters (golangci-lint + eslint)
```

Each target is self-documented with `##` comments so `make help` can auto-generate a usage summary.

---

## Quick Reference: Where Does This Code Go?

| I need to... | Put it in... |
|--------------|-------------|
| Add a new API endpoint | `internal/handler/` (new file or existing domain file) |
| Add business logic | `internal/service/` |
| Add a database query | `internal/repository/` |
| Add a new database table | `migrations/` (new migration pair) |
| Define a request/response shape | `internal/dto/` |
| Add a reusable Go utility | `pkg/` (only if it has no `internal` imports) |
| Add a new UI page | `src/app/(dashboard)/` or `src/app/(auth)/` |
| Add a reusable UI component | `src/components/ui/` (generic) or `src/components/<domain>/` |
| Add a new React hook | `src/hooks/` |
| Add a TypeScript type | `src/types/` |
| Add a new Docker service | `docker/` (Dockerfile) + `docker-compose.yml` |
| Add a new Nginx route | `nginx/conf.d/default.conf` |
