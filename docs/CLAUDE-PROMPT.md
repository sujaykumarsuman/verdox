# CLAUDE-PROMPT.md — Verdox

> **Verdox** — Test orchestration platform for GitHub repositories.
> "Test Your Services at One Place."

---

## 🎯 Project Overview

Build **Verdox**, a self-hosted test orchestration platform that connects to GitHub repositories, lets teams organize and run test suites (unit + integration), view results, and manage access via teams with role-based permissions.

**Target deployment:** Docker Compose + Nginx reverse proxy on a local machine or VPS.

---

## 📎 Reference Wireframes

See attached wireframe images (hand-drawn). They define the following screens:

### Screen Inventory (from wireframes)

1. **Landing Page** — Hero: "Test Your Services at One Place", Login / Sign Up buttons, minimal illustration placeholders.
2. **Sign Up** — Fields: Username, Email, Password → Sign Up button.
3. **Login** — Fields: Username/Email, Password, "Forgot Password" link → Login button.
4. **Dashboard (authenticated)** — Left sidebar: Repository, Teams, Test. Top bar: logo, theme toggle (light/dark), notification bell, user avatar+name. Main area: Repositories list — each repo card shows name, "Run a Test" button, "Dash →" link.
5. **Repository Detail** — Breadcrumb: Repository/Consul. Branch selector (main●), commit hash. Sections: Unit Test (progress bar, pass/total count, Run button, Detail→ link), Integration Test (play/run button).
6. **Test Run Detail** — Repo: Consul/Unit, Branch toggle, commit hash. Test Run #2 header, "Run Logs" button. Per-test rows: Test 1 — Pass (play icon), Test 2 — Fail (play icon). Each test has a status icon and can replay/view logs.
7. **Teams List** — "Teams" header, "Create New" button. Team cards with name + enter arrow.
8. **Team Detail** — Team: consul-team. Two panels side by side: **Repo** panel (list repos with +/- to assign/unassign), **Members** panel (list members with role badges — admin/mod, approve ✓ / reject ✗ actions).
9. **Admin Panel** — Admin sidebar, Users section for user management.
10. **User Menu Dropdown** — Clicking avatar shows: Settings, Admin/Mod (if applicable), Sign Out.

---

## 🏗️ Tech Stack

| Layer | Technology |
|-------|-----------|
| **Backend** | Go 1.22+, Echo v4 framework |
| **Frontend** | Next.js 14 (App Router), TypeScript, Tailwind CSS |
| **Database** | PostgreSQL 16 |
| **Cache/Queue** | Redis 7 (session cache + job queue) |
| **Test Runner** | Docker-in-Docker (DinD) — spin up ephemeral containers per test run |
| **Auth** | JWT (access + refresh tokens), bcrypt passwords |
| **GitHub Integration** | GitHub App or OAuth App + GitHub API v3/v4 |
| **Reverse Proxy** | Nginx |
| **Containerization** | Docker, Docker Compose |
| **CI/CD** | GitHub Actions (for Verdox itself) |

---

## 🎨 Brand & Design System

### Palette

| Token | Hex | Usage |
|-------|-----|-------|
| `--accent` | `#1C6D74` | Primary buttons, active states, links |
| `--accent-light` | `#248F98` | Hover states, secondary highlights |
| `--accent-dark` | `#155459` | Pressed states |
| `--bg-primary` | `#FAFAF8` | Page background (light mode) |
| `--bg-secondary` | `#F0EDE6` | Card/panel backgrounds (light mode) |
| `--bg-primary-dark` | `#1A1A1A` | Page background (dark mode) |
| `--bg-secondary-dark` | `#242424` | Card/panel backgrounds (dark mode) |
| `--text-primary` | `#1A1A1A` | Main text (light mode) |
| `--text-secondary` | `#6B6B6B` | Muted text |
| `--text-primary-dark` | `#E8E8E8` | Main text (dark mode) |
| `--success` | `#2D8A4E` | Pass, success states |
| `--danger` | `#C93B3B` | Fail, error states |
| `--warning` | `#D4910A` | Warnings |
| `--border` | `#E2DFD8` | Borders, dividers (light) |
| `--border-dark` | `#333333` | Borders, dividers (dark) |

### Typography
- **Display/Headings:** "DM Serif Display" (Google Fonts) — warm, editorial feel
- **Body/UI:** "DM Sans" (Google Fonts) — clean, geometric, pairs well
- **Mono (code/logs):** "JetBrains Mono"

### Design Principles
- **Minimal & soothing** — generous whitespace, muted warm backgrounds, no visual clutter
- **Warm neutrals** — off-white/cream tones, not stark white
- **Subtle depth** — light box shadows, no harsh borders. Cards have `border: 1px solid var(--border)` + `box-shadow: 0 1px 3px rgba(0,0,0,0.04)`
- **Smooth transitions** — 200ms ease on hovers, 300ms on page transitions
- **Dark mode support** — toggle in top bar (sun/moon icon from wireframe)
- **Rounded but not bubbly** — `border-radius: 8px` for cards, `6px` for buttons, `4px` for inputs

---

## 📁 Required Documentation (generate BEFORE implementation)

Generate all of the following files under `docs/` directory. Each must be thorough, detailed, and implementation-ready:

```
docs/
├── PRD.md                    # Product Requirements Document — features, user stories, acceptance criteria
├── ARCHITECTURE.md           # System architecture — component diagram, data flow, service boundaries
├── CODE-STRUCTURE.md         # Directory structure for both backend and frontend, file naming conventions
├── BUILD-PLAN.md             # Sprint-based implementation plan with phases, tasks, and confirmation gates
├── LLD/
│   ├── DATABASE.md           # PostgreSQL schema — all tables, indexes, constraints, migrations strategy
│   ├── API.md                # REST API spec — every endpoint, request/response shapes, auth requirements
│   ├── AUTH.md               # Auth flow — signup, login, JWT refresh, password reset, role checks
│   ├── GITHUB-INTEGRATION.md # GitHub App/OAuth setup, webhook handling, repo sync, branch/commit fetch
│   ├── TEST-RUNNER.md        # How tests are executed — Docker-in-Docker, job queue, log streaming, timeouts
│   └── FRONTEND-ROUTES.md    # Next.js route map, page components, layouts, protected routes
├── BRAND-PALETTE.md          # Full design tokens (colors, typography, spacing, shadows, radii) — copy from above + expand
├── ADMIN-PANEL.md            # Admin features — user management, system config, audit logs
├── SECURITY.md               # Auth security, input validation, rate limiting, CORS, secrets management
├── DEPLOYMENT.md             # Docker Compose setup, Nginx config, env vars, local dev setup
├── VPS-DEPLOYMENT.md         # Production VPS deployment — SSL, systemd, backups, domain setup
├── MONITORING.md             # Logging (structured JSON), health checks, metrics, alerting
├── SETUP.md                  # Developer setup guide — prerequisites, clone, configure, run
├── USAGE-GUIDE.md            # End-user guide — how to connect repo, create team, run tests, view results
├── STATUS.md                 # Implementation status tracker — checkboxes per feature/phase
└── CLAUDE-PROMPT.md          # This file (copy it into the repo)
```

### Documentation Quality Bar
- Every doc must be **implementation-ready** — a developer should be able to build from it without asking questions.
- Database schema must include exact SQL `CREATE TABLE` statements.
- API docs must include exact endpoint paths, HTTP methods, request/response JSON examples, and error codes.
- Build plan must have **confirmation gates** — do NOT proceed to next phase without explicit approval.

---

## 🗄️ Data Model (High-Level)

### Core Entities

**users** — id, username, email, password_hash, role (super_admin | admin | user), avatar_url, created_at, updated_at

**repositories** — id, owner_id, github_repo_id, github_full_name, name, description, default_branch, webhook_secret, is_active, created_at, updated_at

**teams** — id, name, slug, created_by, created_at, updated_at

**team_members** — id, team_id, user_id, role (admin | mod | member), status (pending | approved | rejected), invited_by, created_at

**team_repositories** — id, team_id, repository_id, added_by, created_at

**test_suites** — id, repository_id, name, type (unit | integration), config_path, timeout_seconds, created_at, updated_at

**test_runs** — id, test_suite_id, triggered_by, branch, commit_hash, status (queued | running | passed | failed | cancelled), started_at, finished_at, created_at

**test_results** — id, test_run_id, test_name, status (pass | fail | skip | error), duration_ms, error_message, log_output, created_at

**sessions** — id, user_id, refresh_token_hash, expires_at, created_at

---

## 🔌 API Endpoints (High-Level)

### Auth
- `POST /api/v1/auth/signup`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/logout`
- `POST /api/v1/auth/forgot-password`
- `POST /api/v1/auth/reset-password`

### Repositories
- `GET /api/v1/repositories` — list user's repos
- `POST /api/v1/repositories/sync` — sync from GitHub
- `GET /api/v1/repositories/:id` — repo detail
- `DELETE /api/v1/repositories/:id`
- `GET /api/v1/repositories/:id/branches`
- `GET /api/v1/repositories/:id/commits?branch=`

### Test Suites
- `GET /api/v1/repositories/:id/suites` — list test suites for repo
- `POST /api/v1/repositories/:id/suites` — create/configure test suite
- `PUT /api/v1/suites/:id` — update suite config
- `DELETE /api/v1/suites/:id`

### Test Runs
- `POST /api/v1/suites/:id/run` — trigger test run (branch + commit)
- `GET /api/v1/suites/:id/runs` — list runs for suite
- `GET /api/v1/runs/:id` — run detail with results
- `GET /api/v1/runs/:id/logs` — stream/fetch logs
- `POST /api/v1/runs/:id/cancel`
- `POST /api/v1/repositories/:id/run-all` — run all suites for a repo

### Teams
- `GET /api/v1/teams`
- `POST /api/v1/teams`
- `GET /api/v1/teams/:id`
- `PUT /api/v1/teams/:id`
- `DELETE /api/v1/teams/:id`
- `POST /api/v1/teams/:id/members` — invite member
- `PUT /api/v1/teams/:id/members/:userId` — update role, approve/reject
- `DELETE /api/v1/teams/:id/members/:userId` — remove member
- `POST /api/v1/teams/:id/repositories` — assign repo to team
- `DELETE /api/v1/teams/:id/repositories/:repoId` — unassign

### Admin
- `GET /api/v1/admin/users` — list all users
- `PUT /api/v1/admin/users/:id` — update role, deactivate
- `GET /api/v1/admin/stats` — system stats

### User
- `GET /api/v1/me` — current user profile
- `PUT /api/v1/me` — update profile
- `PUT /api/v1/me/password` — change password

---

## 🐳 Deployment Architecture

```
                    ┌──────────┐
                    │  Nginx   │ :80/:443
                    └────┬─────┘
                         │
              ┌──────────┴──────────┐
              │                     │
        ┌─────┴─────┐       ┌──────┴──────┐
        │ Next.js   │ :3000 │  Go API     │ :8080
        │ Frontend  │       │  Backend    │
        └───────────┘       └──────┬──────┘
                                   │
                    ┌──────────────┼──────────────┐
                    │              │              │
              ┌─────┴─────┐ ┌─────┴─────┐ ┌─────┴─────┐
              │ PostgreSQL│ │   Redis   │ │  DinD     │
              │   :5432   │ │   :6379   │ │  Runner   │
              └───────────┘ └───────────┘ └───────────┘
```

### Docker Compose Services
- `nginx` — reverse proxy
- `frontend` — Next.js production build
- `backend` — Go binary
- `postgres` — database
- `redis` — cache + job queue
- `runner` — Docker-in-Docker test executor (privileged container)

---

## ⚙️ Implementation Conventions

### Backend (Go)
- **Project layout:** Standard Go project layout (`cmd/`, `internal/`, `pkg/`)
- **Config:** Viper for config, `.env` file for secrets
- **Migrations:** golang-migrate
- **Logging:** zerolog (structured JSON)
- **Error handling:** Custom error types with HTTP status mapping
- **Validation:** go-playground/validator
- **Git:** Feature branches (`feat/`, `fix/`, `chore/`), no direct pushes to main

### Frontend (Next.js)
- **App Router** with layouts for authenticated/unauthenticated states
- **Server Components** where possible, Client Components for interactive parts
- **API calls:** fetch with typed responses, no external HTTP lib
- **State:** React Context for auth, Zustand if needed for complex state
- **Forms:** React Hook Form + Zod validation
- **Icons:** Lucide React
- **Toasts:** Sonner
- **Dark mode:** next-themes with CSS variables

### Shared
- All environment variables documented in `.env.example`
- `Makefile` for common commands (`make dev`, `make build`, `make migrate-up`, `make test`)
- `docker-compose.yml` for full stack
- `docker-compose.dev.yml` override for development (hot reload, volume mounts)

---

## 🚦 Build Plan (Phased)

### Phase 0 — Documentation (THIS PHASE)
Generate all docs listed above. Get approval before proceeding.

### Phase 1 — Foundation
- Project scaffolding (Go + Next.js)
- Docker Compose setup (postgres, redis, nginx)
- Database migrations (all tables)
- Auth system (signup, login, JWT, refresh)
- Landing, Login, Signup pages

**Gate: Auth flow works end-to-end. Can sign up, login, see dashboard shell.**

### Phase 2 — Repository Management
- GitHub OAuth/App integration
- Repository sync, list, detail
- Branch/commit fetching
- Repository detail page with branch selector

**Gate: Can connect GitHub, see repos, select branches.**

### Phase 3 — Test Execution
- Test suite CRUD
- Job queue (Redis-based)
- Docker-in-Docker test runner
- Test run triggering, status tracking
- Log capture and storage
- Test results display (pass/fail per test)

**Gate: Can configure a test suite, run it, see results with logs.**

### Phase 4 — Teams & Access Control
- Team CRUD
- Member management (invite, approve/reject, roles)
- Repo-team assignment
- Permission checks on API endpoints
- Teams UI pages

**Gate: Teams work. Members with different roles see appropriate content.**

### Phase 5 — Admin & Polish
- Admin panel (user management)
- Settings page
- Dark mode toggle
- Notifications (bell icon — in-app)
- Error pages (404, 500)
- Loading states, empty states
- Mobile responsiveness

**Gate: Admin can manage users. Dark mode works. UI is polished.**

### Phase 6 — Deployment & Monitoring
- Production Docker Compose
- Nginx config with SSL placeholder
- Health check endpoints
- Structured logging
- VPS deployment scripts
- Backup strategy docs

**Gate: Can deploy to a VPS with `docker compose up -d`.**

---

## 🛡️ Security Requirements

- Bcrypt password hashing (cost 12)
- JWT access tokens (15min expiry) + refresh tokens (7d, stored hashed in DB)
- Rate limiting on auth endpoints (5 req/min)
- CORS restricted to frontend origin
- Input validation on all endpoints
- SQL injection prevention (parameterized queries only)
- XSS prevention (Next.js default escaping + CSP headers)
- Webhook signature verification for GitHub
- Secrets in `.env`, never in code
- Privileged DinD container isolated on internal Docker network

---

## 📌 Key Decisions

1. **Why DinD for test runner?** Isolation. Each test run gets a fresh container. No host contamination. Supports any language's test framework.
2. **Why Redis for job queue?** Simple, no extra infra. Use Redis lists/streams for job queue. Upgrade to dedicated queue (Asynq) if needed.
3. **Why not GitHub Actions integration?** This is a self-hosted alternative. Users want to run tests on their own infra without CI vendor lock-in.
4. **Why Echo over Gin?** Cleaner middleware API, better error handling patterns, equally performant.

---

## 🚀 First Instruction

**Step 1:** Generate ALL documentation files listed in the "Required Documentation" section above. Each file must be thorough and implementation-ready. Do NOT start any code implementation until all docs are generated and explicitly approved.

**Step 2:** After docs are approved, begin Phase 1 implementation following the BUILD-PLAN.md.

**Confirmation gate protocol:** After completing each phase, stop and list what was done. Wait for explicit "proceed" before starting next phase.
