# Verdox

Self-hosted test orchestration platform. Trigger, manage, and analyze test runs against GitHub repositories without vendor lock-in.

## Architecture

```
Browser --> Nginx (reverse proxy)
              |
        +-----+-----+
        |           |
    Next.js 15   Go / Echo v4
    (frontend)    (backend)
        |           |
        |     +-----+-----+
        |     |           |
        |  PostgreSQL   Redis
        |     (data)    (queue + SSE)
        |
        +---> GitHub Actions (fork-based test execution)
```

**Test execution model:** Verdox forks target repositories using a service account, pushes workflow files to the fork, and dispatches GitHub Actions runs. Results are ingested via webhook or artifact download.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Frontend | Next.js 15, React 19, TypeScript, Tailwind CSS |
| Backend | Go 1.26, Echo v4, sqlx |
| Database | PostgreSQL 17 |
| Cache/Queue | Redis 7 |
| Proxy | Nginx 1.27 |
| Auth | JWT (HS256) with httpOnly refresh cookies |
| Test Execution | GitHub Actions via fork-based workflow dispatch |

## Features

- **Repository Management** -- Add GitHub repos, browse branches/commits, fork management
- **Test Suites** -- Create suites with custom workflow YAML, template system, AI-powered import from existing GHA workflows
- **Test Runs** -- Trigger runs per branch, real-time status via SSE, cancel/rerun support
- **Hierarchical Results** -- 3-level test result model: runs > groups (jobs) > cases
- **Teams** -- Create teams, invite members (admin/maintainer/viewer roles), assign repos, join requests
- **Admin Panel** -- User management, ban system with appeal flow, bulk notifications, system stats
- **Notifications** -- Real-time via Redis pub/sub + SSE, persisted in DB
- **Dark Mode** -- Full dark/light theme support

## Quick Start

```bash
# Prerequisites: Docker, Docker Compose, Go 1.26+, Node.js 22+, golang-migrate CLI

# Clone and start
git clone https://github.com/sujaykumarsuman/verdox.git
cd verdox
cp .env.example .env.dev   # Edit with your values
make dev                    # Starts postgres, redis, backend (air), frontend (next dev), nginx

# Access
open http://localhost       # Frontend (via nginx)
# Default root user: admin@verdox.local / changeme123
```

## Project Structure

```
verdox/
|-- backend/
|   |-- cmd/server/          # HTTP server entry point, routes
|   |-- internal/
|   |   |-- config/          # Viper-based configuration
|   |   |-- handler/         # HTTP handlers (12 files)
|   |   |-- service/         # Business logic (11 services)
|   |   |-- repository/      # Data access layer (16 repos)
|   |   |-- model/           # Domain entities
|   |   |-- dto/             # Request/response DTOs
|   |   |-- middleware/      # Auth, CORS, rate limiting
|   |   |-- runner/          # Worker pool, fork executor, GHA poller
|   |   |-- queue/           # Redis job queue
|   |   +-- sse/             # Server-sent events publisher
|   |-- migrations/          # 16 SQL migrations (golang-migrate)
|   +-- pkg/                 # Shared utilities (jwt, hash, encryption, logger)
|-- frontend/
|   +-- src/
|       |-- app/             # Next.js app router (auth + dashboard pages)
|       |-- components/      # React components by domain
|       |-- hooks/           # Custom hooks (auth, repos, teams, tests, SSE)
|       |-- lib/             # API client, auth context, query keys
|       +-- types/           # TypeScript type definitions
|-- docker/                  # Dockerfiles (backend, frontend)
|-- nginx/                   # Nginx config (prod SSL + dev HTTP)
|-- docs/                    # Architecture, LLD, guides
+-- docker-compose*.yml      # Production and dev compose files
```

## Makefile Targets

```bash
make dev              # Full stack with hot reload
make dev-backend      # Backend only (air)
make dev-frontend     # Frontend only (next dev)
make up               # Production stack
make down             # Stop and remove containers + volumes
make migrate-up       # Apply pending migrations
make migrate-down     # Rollback last migration
make migrate-create NAME=xyz  # Create new migration pair
make seed             # Bootstrap root user
make test             # Run all tests
make lint             # Run linters
```

## API Overview

| Group | Base Path | Key Endpoints |
|-------|-----------|--------------|
| Auth | `/v1/auth` | signup, login, logout, refresh, forgot/reset password |
| Users | `/v1/users` | profile, password change |
| Teams | `/v1/teams` | CRUD, members, PAT, repos, join requests |
| Repositories | `/v1/repositories` | CRUD, branches, commits, fork, suites, workflows |
| Suites | `/v1/suites` | CRUD, trigger run, list runs |
| Runs | `/v1/runs` | status, logs, cancel, rerun, hierarchy (groups/cases) |
| Reports | `/v1/reports` | Multi-suite batch results |
| Notifications | `/v1/notifications` | list, mark read, unread count |
| Admin | `/v1/admin` | users, teams, stats, ban reviews, mail |
| Webhooks | `/v1/webhooks` | GHA callback, direct ingestion |
| SSE | `/v1/sse` | Real-time event stream |

## Database Schema (16 migrations)

Core tables: `users`, `sessions`, `password_resets`, `repositories`, `teams`, `team_members`, `team_join_requests`, `team_repositories`, `test_suites`, `test_runs`, `test_results`, `test_groups`, `test_cases`, `ban_reviews`, `notifications`

## Build Progress

| Phase | Status |
|-------|--------|
| Phase 0: Documentation | Done |
| Phase 1: Foundation | Done |
| Phase 2: Repository Management | Done |
| Phase 3: Test Execution | Done |
| Phase 4: Teams & Access Control | Done |
| Phase 5: Admin & Polish | Done |
| Phase 6: Deployment & Monitoring | Not started |
| Phase 7: Testing & Hardening | Not started |

## Documentation

See [docs/](docs/) for detailed documentation:
- [Architecture](docs/ARCHITECTURE.md) -- System design and responsibility matrix
- [Build Plan](docs/BUILD-PLAN.md) -- Phase-by-phase task breakdown
- [Status](docs/STATUS.md) -- Implementation progress tracker
- [Setup](docs/SETUP.md) -- Developer environment setup
- [Deployment](docs/DEPLOYMENT.md) -- Docker deployment guide
- [API Reference](docs/LLD/API.md) -- Complete REST API documentation
- [Database Schema](docs/LLD/DATABASE.md) -- Full schema with ER diagram
- [Security](docs/SECURITY.md) -- Security controls and threat model

## License

Private -- All rights reserved.
