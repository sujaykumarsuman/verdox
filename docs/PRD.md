# Verdox -- Product Requirements Document

**Version:** 1.0
**Date:** 2026-04-05
**Status:** Draft

---

## Table of Contents

1. [Product Vision & Positioning](#1-product-vision--positioning)
2. [Target Users](#2-target-users)
3. [Tech Stack](#3-tech-stack)
4. [Feature Groups & User Stories](#4-feature-groups--user-stories)
5. [Non-Functional Requirements](#5-non-functional-requirements)
6. [Out of Scope for v1](#6-out-of-scope-for-v1)
7. [Screen Inventory](#7-screen-inventory)
8. [Data Entities](#8-data-entities)

---

## 1. Product Vision & Positioning

**Tagline:** Test Your Services at One Place

Verdox is a self-hosted test orchestration platform that gives engineering teams full control over their testing workflow without vendor lock-in. Teams connect their GitHub repositories, organize test suites (unit and integration), trigger runs, inspect results, and manage access through role-based team permissions -- all from a single dashboard they own and operate.

### Why Verdox?

- **No vendor lock-in.** Run Verdox on your own infrastructure. Your test data, logs, and configuration stay on machines you control.
- **Single pane of glass.** Stop jumping between CI dashboards, terminal windows, and GitHub checks. Verdox consolidates test orchestration into one interface.
- **Team-first access model.** Repositories are assigned to teams. Members are granted roles. Visibility and execution permissions follow organizational structure, not ad-hoc sharing links.
- **Lightweight deployment.** A single `docker compose up` brings up the entire stack -- API server, frontend, database, cache, and test runner -- behind an Nginx reverse proxy.

### Deployment Model

Docker Compose + Nginx reverse proxy. All services (Go API, Next.js frontend, PostgreSQL, Redis) are defined as Compose services. Nginx terminates TLS and routes traffic. Test execution runs on GitHub Actions via fork-based workflow dispatch -- no runner container needed.

---

## 2. Target Users

| Persona | Description | Primary Goals |
|---|---|---|
| **Software Engineer** | Individual contributor writing and maintaining application code. | Connect repos, create test suites, trigger runs, inspect results and logs. |
| **DevOps Engineer** | Responsible for infrastructure, CI/CD pipelines, and platform tooling. | Deploy and operate the Verdox instance, configure runner capacity, monitor system health. |
| **Team Lead / Engineering Manager** | Oversees one or more engineering teams. | Create teams, manage membership and roles, assign repos to teams, review aggregate test health. |
| **Open-Source Maintainer** | Maintains public or internal open-source projects. | Self-host a test dashboard for contributors, review test results per branch/commit, control access. |

---

## 3. Tech Stack

| Layer | Technology | Notes |
|---|---|---|
| Backend API | Go 1.26+, Echo v4 | RESTful JSON API, JWT authentication |
| Frontend | Next.js 15 (App Router), TypeScript, Tailwind CSS | Server and client components, SPA navigation |
| Database | PostgreSQL 17 | Primary data store for all entities |
| Cache / Queue | Redis 7 | Token blocklist, session metadata, job queue, SSE pub/sub |
| Test Execution | GitHub Actions (fork-based) | Tests run on GHA runners via workflow dispatch on Verdox-managed forks |
| Reverse Proxy | Nginx | TLS termination, static asset serving, upstream routing |
| Deployment | Docker Compose | Single-command orchestration of all services |

---

## 4. Feature Groups & User Stories

### 4.1 Authentication

| ID | User Story | Acceptance Criteria |
|---|---|---|
| AUTH-1 | As a new user, I want to sign up with a username, email, and password, so that I can create an account and access the platform. | 1. Sign-up form requires username (3-30 alphanumeric characters), valid email, and password (min 8 characters, at least one uppercase, one lowercase, one digit). 2. Duplicate username or email returns a descriptive error. 3. On success the user is redirected to the login page with a confirmation message. 4. Password is stored as a bcrypt hash; plaintext is never persisted. |
| AUTH-2 | As a registered user, I want to log in with my username or email and password, so that I can access my dashboard and repositories. | 1. Login accepts either username or email plus password. 2. On success the server returns a JWT access token (15-min expiry) and a refresh token (7-day expiry). 3. Three consecutive failed attempts for the same account trigger a 60-second lockout. 4. Invalid credentials return a generic "invalid credentials" message (no enumeration). |
| AUTH-3 | As a logged-in user, I want to log out, so that my session is terminated and my tokens are invalidated. | 1. Logout invalidates the current access token by adding it to the Redis blocklist. 2. The refresh token is revoked in the database. 3. The client is redirected to the login page. 4. Subsequent API calls with the invalidated token return 401. |
| AUTH-4 | As a user who forgot my password, I want to request a password reset, so that I can regain access to my account. | 1. User submits their email address on the forgot-password form. 2. If the email exists, a reset token (valid for 30 minutes) is generated and logged to the server console (email delivery is out of scope for v1). 3. The reset-password page accepts the token and a new password. 4. After reset, all existing sessions for that user are invalidated. |
| AUTH-5 | As an authenticated user, I want my access token to refresh automatically before it expires, so that I am not forced to log in again during active use. | 1. The frontend sends the refresh token to `/api/v1/auth/refresh` when the access token has less than 2 minutes remaining. 2. The server returns a new access token and rotates the refresh token. 3. If the refresh token is expired or revoked, the user is redirected to login. |

### 4.2 Repository Management

| ID | User Story | Acceptance Criteria |
|---|---|---|
| REPO-1 | As a team admin, I want to configure a GitHub PAT for my team, so that all team members can add and access repositories. | 1. Team admin provides a GitHub PAT on the team settings page. 2. The system validates the token by calling the GitHub API (`/user` endpoint). 3. On success the encrypted token is stored in the `teams` table and the GitHub username is recorded on the team. 4. Invalid or expired tokens return an actionable error. 5. Only team admins can set, rotate, or revoke the PAT. 6. A PAT status indicator shows whether the team has a PAT configured and warns if it is expiring soon. |
| REPO-2 | As a team admin, I want to add a repository by URL, so that it is cloned and available for testing. | 1. Team admin enters a GitHub repository URL on the team detail page. 2. The system clones the repository using the team's stored PAT. 3. The cloned repository is stored on disk at the configured `VERDOX_REPO_BASE_PATH`. 4. The repository record is created and assigned to the team. 5. Invalid URLs or inaccessible repositories return an actionable error. |
| REPO-3 | As a user, I want to view my list of connected repositories, so that I can select one to run tests against. | 1. The dashboard shows repo cards with name, owner, language, and last-synced time. 2. Cards include a "Run" button (triggers default suite) and a "Dash" button (navigates to repo detail). 3. List supports search by repo name. 4. Empty state displays a prompt to add a repository. |
| REPO-4 | As a user, I want to browse branches and recent commits for a repository, so that I can choose the correct ref for a test run. | 1. The repo detail page shows a branch selector dropdown populated from the local clone. 2. Selecting a branch displays the 20 most recent commits (hash, message, author, date). 3. The selected branch and commit are used when triggering a test run. |
| REPO-5 | As a user, I want to remove a repository from Verdox, so that it no longer appears in my dashboard. | 1. A "Remove" action is available on the repo detail page. 2. Removing a repo soft-deletes it and disassociates it from all teams. 3. Historical test runs and results for the repo are retained for audit purposes. 4. The action requires confirmation via a dialog. |
| REPO-6 | As a team admin, I want to scan a repository with AI to discover existing tests, so that test suites can be auto-populated. | 1. A "Discover Tests" button is available on the repo detail page (optional feature, requires `VERDOX_OPENAI_API_KEY`). 2. The system scans the repository source code using AI to identify test files, frameworks, and run commands. 3. Discovered tests are presented as suggestions that the user can accept or dismiss. 4. Accepted suggestions create pre-configured test suites. |

### 4.3 Test Execution

| ID | User Story | Acceptance Criteria |
|---|---|---|
| TEST-1 | As a user, I want to create a test suite for a repository, so that I can group related tests under a named configuration. | 1. User provides a suite name, selects the suite type (unit or integration), and associates it with a repository. 2. Suite names must be unique within a repository. 3. The suite is listed on the repo detail page under the appropriate section (Unit Test / Integration Test). |
| TEST-2 | As a user, I want to configure a test suite with a run command and optional environment variables, so that Verdox knows how to execute my tests. | 1. Configuration form accepts a shell command (e.g., `go test ./...`), working directory (relative to repo root), timeout (default 10 minutes, max 60 minutes), and key-value environment variables. 2. Environment variable values are stored encrypted at rest. 3. Configuration can be updated at any time; changes apply to the next run. |
| TEST-3 | As a user, I want to trigger a test run for a suite on a specific branch and commit, so that I can validate my code changes. | 1. The "Run" action starts a new test run and returns a run ID. 2. The system dispatches a GitHub Actions workflow on a Verdox-managed fork of the repository. 3. Only one run per suite can be active at a time; concurrent requests return a 409. 4. The total number of concurrent workflow dispatches is capped (default 5, configurable). |
| TEST-4 | As a user, I want to view the real-time progress of a running test suite, so that I can monitor execution without waiting for completion. | 1. The repo detail page shows a progress bar per suite with pass count / total count. 2. Status updates are delivered via server-sent events (SSE) or polling at 3-second intervals. 3. Possible run statuses: `queued`, `running`, `passed`, `failed`, `cancelled`, `timed_out`. |
| TEST-5 | As a user, I want to view detailed test results with logs after a run completes, so that I can diagnose failures. | 1. The test run detail page lists each individual test case with name and status (passed, failed, skipped, errored). 2. A "Run Logs" button opens the full stdout/stderr output of the run. 3. Failed test rows are highlighted and sorted to the top. 4. Results are retained for 90 days (configurable). |
| TEST-6 | As a user, I want to cancel a running test, so that I can stop a stuck or unnecessary execution. | 1. A "Cancel" button is visible on runs with status `queued` or `running`. 2. Cancellation sends SIGTERM to the runner container, followed by SIGKILL after a 10-second grace period. 3. The run status transitions to `cancelled` and the timestamp is recorded. 4. Partial results collected before cancellation are preserved. |

### 4.4 Team Management

| ID | User Story | Acceptance Criteria |
|---|---|---|
| TEAM-1 | As a team lead, I want to create a team, so that I can group engineers and assign repositories. | 1. Team creation requires a unique team name (3-50 characters) and optional description. 2. The creating user is automatically assigned the `owner` role for that team. 3. Teams are listed on the `/teams` page as cards. |
| TEAM-2 | As a user, I want to request to join a team, so that I can access repositories. | 1. The Team Discovery page (`/teams/discover`) lists all teams with name, description, and member count. 2. Each team card has a "Request to Join" button. 3. Submitting a request creates a pending join request visible to team admins. 4. Duplicate requests for the same team are rejected. 5. The user receives an in-app notification when their request is approved or rejected. |
| TEAM-3 | As a team admin, I want to review and approve/reject join requests, so that I control team membership. | 1. The Join Requests page (`/teams/:id/requests`) lists all pending requests with username, email, and request date. 2. Each request row has "Approve" and "Reject" buttons. 3. Approved members gain the `viewer` role by default. 4. Rejected requests are removed from the pending list and the user is notified. |
| TEAM-4 | As a team lead, I want to assign or unassign repositories to my team, so that team members can run tests on those repos. | 1. The team detail page includes a Repo panel with "+" and "-" controls. 2. Only repos owned by the team lead (or that the lead has access to) can be assigned. 3. Unassigning a repo removes team access but does not delete the repo or its data. 4. Changes take effect immediately. |
| TEAM-5 | As a team lead, I want to manage member roles (owner, admin, maintainer, viewer), so that I can delegate administrative tasks. | 1. Available roles: `owner` (full control, one per team), `admin` (manage members and repos), `maintainer` (run tests and manage suites), `viewer` (view results only). 2. Role changes are made from the Members panel in the team detail page. 3. Only owners can promote a member to admin. 4. An owner cannot demote themselves unless another owner is designated. |

### 4.5 Admin

| ID | User Story | Acceptance Criteria |
|---|---|---|
| ADMIN-1 | As a root user, I want to view all registered users, so that I can audit platform usage. | 1. The admin panel at `/admin` lists all users with username, email, role, status, and registration date. 2. The list supports pagination (20 per page) and search by username or email. 3. Only users with the `root` or `moderator` system role can access this page. |
| ADMIN-2 | As a root user, I want to promote users to moderator, so that they can create teams and manage repos. | 1. System roles: `user`, `moderator`, `root`. 2. Role change takes effect on the user's next API request (token claims are re-evaluated). 3. A root user cannot remove their own root role. 4. Role change is recorded in an audit log. 5. Only the root user can promote to moderator; moderators cannot change system roles. |
| ADMIN-3 | As a root user, I want to deactivate a user account, so that I can revoke access for a departed or malicious user. | 1. Deactivation sets the user status to `inactive`. 2. All active sessions and tokens for the user are immediately invalidated. 3. The user's data is retained but they cannot log in. 4. A deactivated user can be reactivated by a root user. |
| ADMIN-4 | As a root user, I want to view system statistics, so that I can monitor platform health and capacity. | 1. Stats include: total users, active users (last 30 days), total repositories, total test runs (last 7 days), active runs, pass/fail ratio. 2. Data is computed on-demand (no pre-aggregation required for v1). 3. Stats are displayed as summary cards on the admin panel. |

### 4.6 UI/UX

| ID | User Story | Acceptance Criteria |
|---|---|---|
| UX-1 | As a user, I want to toggle between light and dark mode, so that I can use the interface comfortably in any lighting. | 1. A toggle is accessible from the user settings page and/or the top navigation bar. 2. Preference is persisted in `localStorage` and applied on page load. 3. The default follows the operating system preference (`prefers-color-scheme`). 4. All pages and components render correctly in both modes. |
| UX-2 | As a user, I want to receive in-app notifications for important events, so that I stay informed without leaving the platform. | 1. Notifications cover: test run completed, team invitation received, membership approved/rejected. 2. A bell icon in the navigation bar shows an unread count badge. 3. Clicking the bell opens a dropdown with recent notifications (last 20). 4. Each notification links to the relevant page (run detail, team detail). |
| UX-3 | As a user, I want to manage my profile settings, so that I can update my information and preferences. | 1. The settings page allows editing display name and password. 2. Dark mode preference is configurable here. 3. Changes are saved via API and confirmed with a success toast. 4. GitHub PAT management is available on the team settings page (team admin only). |

---

## 5. Non-Functional Requirements

### 5.1 Performance

| Metric | Target |
|---|---|
| API response time (p95) | < 200 ms |
| Frontend Time to Interactive (TTI) | < 3 seconds on a broadband connection |
| SSE / polling latency for run updates | < 5 seconds |

### 5.2 Capacity

| Dimension | Limit |
|---|---|
| Concurrent test runs | 5 (configurable via `MAX_CONCURRENT_RUNS` env var) |
| Repositories per user | 100 |
| Teams per user | 20 |
| Members per team | 50 |
| Test suites per repository | 20 |

### 5.3 Data Retention

| Data Type | Retention |
|---|---|
| Test results and logs | 90 days (configurable via `RESULT_RETENTION_DAYS` env var) |
| Audit logs | 1 year |
| Soft-deleted repositories | 30 days before permanent purge |

### 5.4 Security

- All API endpoints (except signup, login, password reset) require a valid JWT.
- Passwords are hashed with bcrypt (cost factor 12).
- GitHub PATs are encrypted at rest using AES-256-GCM.
- Environment variable values in test suite configs are encrypted at rest.
- CORS is restricted to the configured frontend origin.
- Rate limiting: 100 requests per minute per IP for unauthenticated endpoints; 300 per minute per user for authenticated endpoints.

### 5.5 Browser Support

| Browser | Supported Versions |
|---|---|
| Google Chrome | Latest 2 major versions |
| Mozilla Firefox | Latest 2 major versions |
| Apple Safari | Latest 2 major versions |
| Microsoft Edge | Latest 2 major versions |

### 5.6 Deployment & Operations

- Single `docker compose up -d` brings up all services.
- Health-check endpoints: `GET /api/v1/health` (API), `GET /healthz` (frontend).
- Structured JSON logging to stdout on the API server.
- Graceful shutdown with in-flight request draining (30-second timeout).

---

## 6. Out of Scope for v1

The following capabilities are explicitly excluded from the initial release:

| Exclusion | Rationale |
|---|---|
| CI/CD pipeline creation | Verdox is a test orchestration tool, not a full CI/CD platform. Pipelines are managed externally. |
| Multi-cloud deployment | v1 targets Docker Compose on a single host. Kubernetes and cloud-native deployments are deferred to v2. |
| Test parallelism within a single suite | Each suite runs as a single container. Intra-suite parallelism (sharding) adds runner complexity deferred to v2. |
| Email notifications | v1 supports in-app notifications only. Email/Slack/webhook integrations are planned for v2. |
| Billing and payments | Verdox is self-hosted. There is no SaaS billing model in v1. |
| GitHub OAuth login | v1 uses username/password auth. GitHub OAuth is planned for a future release. |
| GitLab / Bitbucket support | Only GitHub is supported in v1. Other providers may be added later. |
| Test coverage reporting | Verdox shows pass/fail results. Coverage analysis integration is deferred. |
| GitHub webhooks for automatic test triggers | Manual triggers only in v1. Webhook-based auto-triggering is deferred to v2. |

---

## 7. Screen Inventory

| Screen | Route | Key Elements |
|---|---|---|
| Landing Page | `/` | Hero text ("Test Your Services at One Place"), Login button, Sign Up button, feature highlights. |
| Sign Up | `/signup` | Form fields: Username, Email, Password, Confirm Password. Submit button. Link to Login. |
| Login | `/login` | Form fields: Username or Email, Password. Login button. "Forgot Password?" link. Link to Sign Up. |
| Forgot Password | `/forgot-password` | Form field: Email. Submit button. Link back to Login. |
| Reset Password | `/reset-password?token=xxx` | Form fields: New Password, Confirm Password. Submit button. |
| Dashboard | `/dashboard` | Sidebar navigation (Repositories, Teams, Tests). Main area: repository cards grid. Each card shows repo name, owner, language, "Run" button (triggers default suite), "Dash" button (navigates to repo detail). Search bar. Sync Repos button. |
| Repository Detail | `/repositories/:id` | Branch selector dropdown, commit hash display. Unit Test section: suite name, progress bar, pass/total count, "Run" button, "Detail" link. Integration Test section: same layout. List of recent runs. |
| Test Run Detail | `/repositories/:id/runs/:runId` | Header: branch name, commit hash, run number, run status badge. "Run Logs" button (opens full log modal). Table of individual test cases: name, status icon, duration. Failed tests sorted to top. "Cancel" button (if run is active). |
| Teams List | `/teams` | "Create New" button. Grid of team cards (team name, member count, repo count). |
| Team Discovery | `/teams/discover` | Grid of all teams with name, description, and member count. Each card has a "Request to Join" button. Search bar to filter teams. |
| Team Detail | `/teams/:id` | Two-panel layout. Left panel -- Repositories: list of assigned repos with "+" (assign) and "-" (unassign) controls, "Add by URL" form. Right panel -- Members: list of members with role badge, role dropdown for existing members. Link to Join Requests page. GitHub PAT section (admin only): PAT status indicator, set/rotate/revoke PAT form. |
| Join Requests | `/teams/:id/requests` | Table of pending join requests (username, email, request date). "Approve" and "Reject" buttons per row. Empty state when no pending requests. |
| Admin Panel | `/admin` | Users section: paginated table (username, email, role, status, registered date). Role dropdown per user (root can promote to moderator). Activate/Deactivate toggle per user. System stats cards (total users, active users, total repos, runs this week, pass rate). |
| User Settings | `/settings` | Profile section: display name, email (read-only). Password change form. Dark mode toggle. (GitHub PAT management has moved to team settings.) |
| User Menu Dropdown | (overlay, top-right nav) | Menu items: Settings, Admin (visible to root/moderator roles only), Sign Out. |

---

## 8. Data Entities

### 8.1 Entity Relationship Summary

```
users 1---* team_members *---1 teams
teams 1---* team_repositories *---1 repositories
repositories 1---* test_suites
test_suites 1---* test_runs
test_runs 1---* test_results
users 1---* sessions
users 1---* notifications
```

### 8.2 Entity Definitions

#### users

| Column | Type | Constraints | Description |
|---|---|---|---|
| id | UUID | PK | Unique user identifier. |
| username | VARCHAR(30) | UNIQUE, NOT NULL | Login username. |
| email | VARCHAR(255) | UNIQUE, NOT NULL | User email address. |
| password_hash | VARCHAR(255) | NOT NULL | Bcrypt-hashed password. |
| display_name | VARCHAR(100) | | Optional display name. |
| system_role | ENUM | NOT NULL, DEFAULT 'user' | One of: `user`, `moderator`, `root`. |
| status | ENUM | NOT NULL, DEFAULT 'active' | One of: `active`, `inactive`. |
| github_username | VARCHAR(100) | | Linked GitHub username (informational). |
| created_at | TIMESTAMPTZ | NOT NULL | Registration timestamp. |
| updated_at | TIMESTAMPTZ | NOT NULL | Last update timestamp. |

#### repositories

| Column | Type | Constraints | Description |
|---|---|---|---|
| id | UUID | PK | Unique repository identifier. |
| github_id | BIGINT | UNIQUE, NOT NULL | GitHub's numeric repository ID. |
| name | VARCHAR(255) | NOT NULL | Repository name. |
| full_name | VARCHAR(255) | NOT NULL | Full name (owner/repo). |
| owner | VARCHAR(255) | NOT NULL | GitHub owner (user or org). |
| language | VARCHAR(50) | | Primary language reported by GitHub. |
| default_branch | VARCHAR(100) | | Default branch name. |
| clone_url | TEXT | NOT NULL | HTTPS clone URL. |
| is_deleted | BOOLEAN | NOT NULL, DEFAULT false | Soft-delete flag. |
| last_synced_at | TIMESTAMPTZ | | Last sync timestamp. |
| created_at | TIMESTAMPTZ | NOT NULL | Row creation timestamp. |
| updated_at | TIMESTAMPTZ | NOT NULL | Last update timestamp. |

#### teams

| Column | Type | Constraints | Description |
|---|---|---|---|
| id | UUID | PK | Unique team identifier. |
| name | VARCHAR(50) | UNIQUE, NOT NULL | Team name. |
| description | TEXT | | Optional team description. |
| created_by | UUID | FK -> users.id, NOT NULL | User who created the team. |
| github_pat_encrypted | TEXT | | AES-256-GCM encrypted GitHub PAT for the team. |
| github_username | VARCHAR(100) | | GitHub username associated with the team's PAT. |
| pat_expires_at | TIMESTAMPTZ | | Expiry date of the team's GitHub PAT (if known). |
| created_at | TIMESTAMPTZ | NOT NULL | Creation timestamp. |
| updated_at | TIMESTAMPTZ | NOT NULL | Last update timestamp. |

#### team_members

| Column | Type | Constraints | Description |
|---|---|---|---|
| id | UUID | PK | Row identifier. |
| team_id | UUID | FK -> teams.id, NOT NULL | Parent team. |
| user_id | UUID | FK -> users.id, NOT NULL | Member user. |
| role | ENUM | NOT NULL, DEFAULT 'viewer' | One of: `owner`, `admin`, `maintainer`, `viewer`. |
| status | ENUM | NOT NULL, DEFAULT 'pending' | One of: `pending`, `active`, `rejected`. |
| created_at | TIMESTAMPTZ | NOT NULL | Invitation timestamp. |
| updated_at | TIMESTAMPTZ | NOT NULL | Last update timestamp. |

*Unique constraint on (team_id, user_id).*

#### team_repositories

| Column | Type | Constraints | Description |
|---|---|---|---|
| id | UUID | PK | Row identifier. |
| team_id | UUID | FK -> teams.id, NOT NULL | Parent team. |
| repository_id | UUID | FK -> repositories.id, NOT NULL | Assigned repository. |
| assigned_by | UUID | FK -> users.id, NOT NULL | User who assigned the repo. |
| created_at | TIMESTAMPTZ | NOT NULL | Assignment timestamp. |

*Unique constraint on (team_id, repository_id).*

#### test_suites

| Column | Type | Constraints | Description |
|---|---|---|---|
| id | UUID | PK | Unique suite identifier. |
| repository_id | UUID | FK -> repositories.id, NOT NULL | Parent repository. |
| name | VARCHAR(100) | NOT NULL | Suite name (unique per repo). |
| type | ENUM | NOT NULL | One of: `unit`, `integration`. |
| command | TEXT | NOT NULL | Shell command to execute (e.g., `go test ./...`). |
| working_dir | VARCHAR(255) | DEFAULT '.' | Working directory relative to repo root. |
| timeout_seconds | INT | NOT NULL, DEFAULT 600 | Max run duration (600s = 10 min). |
| env_vars_encrypted | TEXT | | JSON object of env vars, encrypted at rest. |
| created_at | TIMESTAMPTZ | NOT NULL | Creation timestamp. |
| updated_at | TIMESTAMPTZ | NOT NULL | Last update timestamp. |

*Unique constraint on (repository_id, name).*

#### test_runs

| Column | Type | Constraints | Description |
|---|---|---|---|
| id | UUID | PK | Unique run identifier. |
| test_suite_id | UUID | FK -> test_suites.id, NOT NULL | Parent suite. |
| triggered_by | UUID | FK -> users.id, NOT NULL | User who triggered the run. |
| run_number | INT | NOT NULL | Sequential run number per suite. |
| branch | VARCHAR(255) | NOT NULL | Git branch name. |
| commit_hash | VARCHAR(40) | NOT NULL | Full SHA of the commit under test. |
| status | ENUM | NOT NULL, DEFAULT 'queued' | One of: `queued`, `running`, `passed`, `failed`, `cancelled`, `timed_out`. |
| total_tests | INT | DEFAULT 0 | Total test case count (populated during/after run). |
| passed_tests | INT | DEFAULT 0 | Count of passed tests. |
| failed_tests | INT | DEFAULT 0 | Count of failed tests. |
| skipped_tests | INT | DEFAULT 0 | Count of skipped tests. |
| log_output | TEXT | | Full stdout/stderr of the run. |
| started_at | TIMESTAMPTZ | | Timestamp when execution began. |
| finished_at | TIMESTAMPTZ | | Timestamp when execution ended. |
| created_at | TIMESTAMPTZ | NOT NULL | Row creation timestamp. |

*Unique constraint on (test_suite_id, run_number).*

#### test_results

| Column | Type | Constraints | Description |
|---|---|---|---|
| id | UUID | PK | Unique result identifier. |
| test_run_id | UUID | FK -> test_runs.id, NOT NULL | Parent run. |
| test_name | VARCHAR(500) | NOT NULL | Fully qualified test name. |
| status | ENUM | NOT NULL | One of: `passed`, `failed`, `skipped`, `errored`. |
| duration_ms | INT | | Execution time in milliseconds. |
| error_message | TEXT | | Error/failure output (if any). |
| created_at | TIMESTAMPTZ | NOT NULL | Row creation timestamp. |

#### sessions

| Column | Type | Constraints | Description |
|---|---|---|---|
| id | UUID | PK | Session identifier. |
| user_id | UUID | FK -> users.id, NOT NULL | Session owner. |
| refresh_token_hash | VARCHAR(255) | NOT NULL | Hashed refresh token. |
| expires_at | TIMESTAMPTZ | NOT NULL | Refresh token expiry. |
| revoked | BOOLEAN | NOT NULL, DEFAULT false | Whether the session has been revoked. |
| user_agent | TEXT | | Client user-agent string. |
| ip_address | VARCHAR(45) | | Client IP at session creation. |
| created_at | TIMESTAMPTZ | NOT NULL | Session creation timestamp. |

#### notifications

| Column | Type | Constraints | Description |
|---|---|---|---|
| id | UUID | PK | Notification identifier. |
| user_id | UUID | FK -> users.id, NOT NULL | Recipient user. |
| type | VARCHAR(50) | NOT NULL | Notification type (e.g., `test_run_completed`, `team_invite`, `membership_approved`). |
| title | VARCHAR(255) | NOT NULL | Short notification title. |
| message | TEXT | | Notification body. |
| link | VARCHAR(500) | | URL to navigate to on click. |
| is_read | BOOLEAN | NOT NULL, DEFAULT false | Read status. |
| created_at | TIMESTAMPTZ | NOT NULL | Creation timestamp. |

---

*End of document.*
