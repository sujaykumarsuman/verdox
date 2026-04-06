# Verdox System Architecture

> Self-hosted test orchestration platform for teams that want full control over
> their CI testing infrastructure.

---

## 1. System Overview

Verdox is a self-hosted test orchestration platform that enables development
teams to trigger, manage, and analyze test runs against their GitHub
repositories. It is designed as a single-node deployment running entirely within
Docker Compose behind an Nginx reverse proxy.

**Tech Stack:**

- **Backend:** Go 1.25+ with Echo v4 HTTP framework
- **Frontend:** Next.js 15 (App Router, React Server Components)
- **Database:** PostgreSQL 17
- **Cache / Queue:** Redis 7
- **Test Execution:** Docker-in-Docker (DinD) ephemeral containers
- **Reverse Proxy:** Nginx (TLS termination, routing, caching)

**Deployment Model:**

All six services are defined in a single `docker-compose.yml` and communicate
over an internal Docker bridge network (`verdox-network`). Nginx is the only
service that exposes ports to the host, making it suitable for deployment on a
local machine, a VPS, or any Docker-capable server. There is no dependency on
Kubernetes or any external orchestrator.

---

## 2. Component Diagram

```
                         ┌──────────────┐
                         │    Nginx     │ :80 / :443
                         │  (reverse    │
                         │   proxy)     │
                         └──────┬───────┘
                                │
                 ┌──────────────┴──────────────┐
                 │                             │
          ┌──────┴───────┐             ┌───────┴───────┐
          │   Next.js    │ :3000       │   Go API      │ :8080
          │   Frontend   │             │   Backend     │
          │  (SSR/CSR)   │             │  (Echo v4)    │
          └──────────────┘             └───────┬───────┘
                                               │
                          ┌────────────────────┼────────────────────┐
                          │                    │                    │
                   ┌──────┴──────┐      ┌──────┴──────┐     ┌──────┴──────┐
                   │ PostgreSQL  │      │    Redis    │     │    DinD     │
                   │    :5432    │      │    :6379    │     │   Runner    │
                   │             │      │             │     │ (ephemeral) │
                   └─────────────┘      └─────────────┘     └─────────────┘
```

**Request routing summary:**

- `/*` (non-API paths) are proxied to the Next.js frontend on port 3000.
- `/api/*` paths are proxied to the Go backend on port 8080.
- All inter-service communication stays on the internal Docker network.

---

## 3. Service Responsibility Matrix

| Service | Container Name | Port | Responsibilities |
|---------|---------------|------|------------------|
| **Nginx** | `verdox-nginx` | 80, 443 | TLS termination (Let's Encrypt or self-signed), reverse proxy routing (`/api/*` to backend, all else to frontend), static asset caching with `Cache-Control` headers, security headers (`X-Frame-Options`, `CSP`, `HSTS`), network-level rate limiting via `limit_req` |
| **Frontend (Next.js)** | `verdox-frontend` | 3000 | Server-side rendering and client-side rendering of UI pages, authentication state management (reading JWT from cookies), API consumption via server-side `fetch` and client-side hooks, theme management (light/dark), route protection with middleware guards |
| **Backend (Go/Echo)** | `verdox-backend` | 8080 | REST API for all business operations, authentication and authorization (JWT issuance and validation), team-level PAT-encrypted GitHub access, local repository clone management, per-repo sequential job queue (Redis), test result parsing and storage |
| **PostgreSQL** | `verdox-postgres` | 5432 | Persistent storage for all domain data: users, repositories, teams, team memberships, test runs, individual test results, sessions. Schema migrations managed by the backend on startup |
| **Redis** | `verdox-redis` | 6379 | Session cache (fast JWT session lookups), per-repo sequential job queue (`verdox:jobs:repo:{repo_id}` lists + `verdox:jobs:active:{repo_id}` locks), rate limit counters (sliding window per user/IP) |
| **DinD Runner** | `verdox-runner` | -- | Ephemeral container execution (one container per test run), local clone mounted read-only (`-v {local_path}:/workspace:ro`), test command execution inside the container, stdout/stderr log capture and streaming back to the worker |

---

## 4. Data Flow Diagrams

### 4a. User Authentication Flow

```
 Client                Nginx              Frontend           Backend            PostgreSQL         Redis
   │                     │                    │                  │                    │                │
   │  POST /login        │                    │                  │                    │                │
   │────────────────────>│                    │                  │                    │                │
   │                     │  proxy /api/login  │                  │                    │                │
   │                     │───────────────────────────────────>  │                    │                │
   │                     │                    │                  │                    │                │
   │                     │                    │                  │  SELECT user       │                │
   │                     │                    │                  │  WHERE email=?     │                │
   │                     │                    │                  │───────────────────>│                │
   │                     │                    │                  │                    │                │
   │                     │                    │                  │  user row          │                │
   │                     │                    │                  │<───────────────────│                │
   │                     │                    │                  │                    │                │
   │                     │                    │                  │  bcrypt.Compare    │                │
   │                     │                    │                  │  (password check)  │                │
   │                     │                    │                  │                    │                │
   │                     │                    │                  │  Generate JWT      │                │
   │                     │                    │                  │  (access + refresh)│                │
   │                     │                    │                  │                    │                │
   │                     │                    │                  │  SET session:{id}  │                │
   │                     │                    │                  │──────────────────────────────────> │
   │                     │                    │                  │                    │                │
   │                     │                    │                  │  session cached    │                │
   │                     │                    │                  │<──────────────────────────────────│
   │                     │                    │                  │                    │                │
   │  Set-Cookie: access_token (httpOnly)     │                  │                    │                │
   │  Set-Cookie: refresh_token (httpOnly)    │                  │                    │                │
   │<────────────────────────────────────────────────────────── │                    │                │
   │                     │                    │                  │                    │                │
```

**Key points:**

- Passwords are verified using bcrypt on the backend.
- JWTs are returned as `httpOnly`, `Secure`, `SameSite=Strict` cookies.
- A session record is cached in Redis for fast validation and revocation.

---

### 4b. Repository Add & Clone Flow

```
 Client              Backend API            PostgreSQL          Redis Queue        Worker          Local Disk
   │                    │                       │                   │                 │                │
   │  POST /api/v1/     │                       │                   │                 │                │
   │  repositories      │                       │                   │                 │                │
   │  {github_url}      │                       │                   │                 │                │
   │───────────────────>│                       │                   │                 │                │
   │                    │                       │                   │                 │                │
   │                    │  Validate URL format  │                   │                 │                │
   │                    │  Lookup team PAT      │                   │                 │                │
   │                    │  (decrypt AES-256)    │                   │                 │                │
   │                    │                       │                   │                 │                │
   │                    │  INSERT INTO repos    │                   │                 │                │
   │                    │  (clone_status:       │                   │                 │                │
   │                    │   'pending')          │                   │                 │                │
   │                    │──────────────────────>│                   │                 │                │
   │                    │                       │                   │                 │                │
   │                    │  LPUSH                │                   │                 │                │
   │                    │  verdox:jobs:clone    │                   │                 │                │
   │                    │  {repo_id, url, pat}  │                   │                 │                │
   │                    │──────────────────────────────────────────>│                 │                │
   │                    │                       │                   │                 │                │
   │  202 Accepted      │                       │                   │                 │                │
   │  {repo, status:    │                       │                   │                 │                │
   │   pending}         │                       │                   │                 │                │
   │<───────────────────│                       │                   │                 │                │
   │                    │                       │                   │                 │                │
   │                    │                       │                   │  BRPOP          │                │
   │                    │                       │                   │  verdox:jobs:   │                │
   │                    │                       │                   │  clone          │                │
   │                    │                       │                   │<────────────────│                │
   │                    │                       │                   │                 │                │
   │                    │                       │                   │                 │  git clone     │
   │                    │                       │                   │                 │  --depth 1     │
   │                    │                       │                   │                 │  --branch main │
   │                    │                       │                   │                 │────────────────>
   │                    │                       │                   │                 │                │
   │                    │                       │                   │                 │  UPDATE repos  │
   │                    │                       │                   │                 │  SET clone_    │
   │                    │                       │                   │                 │  status =      │
   │                    │                       │                   │                 │  'ready',      │
   │                    │                       │                   │                 │  local_path    │
   │                    │                       │                   │                 │──────────────> │
   │                    │                       │                   │                 │  (via PG)      │
   │                    │                       │                   │                 │                │
```

**Key points:**

- Users add repos by GitHub URL (e.g., `https://github.com/org/repo`).
- The backend resolves the team's PAT (repo -> team -> `teams.github_pat_encrypted`) and decrypts it (AES-256-GCM) for git authentication.
- Repos are shallow-cloned (`--depth 1`) to `VERDOX_REPO_BASE_PATH/{repo_id}/` on local disk.
- Clone happens asynchronously via a worker -- the API returns 202 immediately.
- Branch listing uses `git ls-remote --heads origin` (works with shallow clones).
- Commit listing uses GitHub API `GET /repos/{owner}/{repo}/commits` (the only GitHub API call).

---

### 4c. Test Run Execution Flow

This is the core workflow of Verdox. It uses a per-repo sequential queue,
local clone mounted read-only, commit-hash caching, and ephemeral DinD containers.

```
 Client        Backend API        PostgreSQL        Redis                Worker Goroutine      Docker (DinD)
   │               │                  │                  │                   │                    │
   │  POST /api/   │                  │                  │                   │                    │
   │  v1/suites/   │                  │                  │                   │                    │
   │  {id}/runs    │                  │                  │                   │                    │
   │  {branch}     │                  │                  │                   │                    │
   │──────────────>│                  │                  │                   │                    │
   │               │                  │                  │                   │                    │
   │               │  Check: existing │                  │                   │                    │
   │               │  run for same    │                  │                   │                    │
   │               │  suite+branch+   │                  │                   │                    │
   │               │  commit_sha?     │                  │                   │                    │
   │               │─────────────────>│                  │                   │                    │
   │               │                  │                  │                   │                    │
   │               │  (if cached and  │                  │                   │                    │
   │               │   !force: return │                  │                   │                    │
   │               │   cached result) │                  │                   │                    │
   │               │                  │                  │                   │                    │
   │               │  INSERT INTO     │                  │                   │                    │
   │               │  test_runs       │                  │                   │                    │
   │               │  (status:queued, │                  │                   │                    │
   │               │   run_number++)  │                  │                   │                    │
   │               │─────────────────>│                  │                   │                    │
   │               │                  │                  │                   │                    │
   │               │  LPUSH           │                  │                   │                    │
   │               │  verdox:jobs:    │                  │                   │                    │
   │               │  repo:{repo_id} │                  │                   │                    │
   │               │  {run_id, ...}   │                  │                   │                    │
   │               │─────────────────────────────────>  │                   │                    │
   │               │                  │                  │                   │                    │
   │  202 Accepted │                  │                  │                   │                    │
   │  {run_id}     │                  │                  │                   │                    │
   │<──────────────│                  │                  │                   │                    │
   │               │                  │                  │                   │                    │
   │               │                  │                  │  SETNX            │                    │
   │               │                  │                  │  verdox:jobs:     │                    │
   │               │                  │                  │  active:{repo_id} │                    │
   │               │                  │                  │<──────────────────│                    │
   │               │                  │                  │                   │                    │
   │               │                  │                  │  (lock acquired)  │                    │
   │               │                  │                  │──────────────────>│                    │
   │               │                  │                  │                   │                    │
   │               │                  │                  │  RPOP             │                    │
   │               │                  │                  │  verdox:jobs:     │                    │
   │               │                  │                  │  repo:{repo_id}   │                    │
   │               │                  │                  │<──────────────────│                    │
   │               │                  │                  │                   │                    │
   │               │                  │                  │                   │  git fetch         │
   │               │                  │                  │                   │  --depth 1 origin  │
   │               │                  │                  │                   │  {branch}          │
   │               │                  │                  │                   │  (on local clone)  │
   │               │                  │                  │                   │                    │
   │               │                  │                  │                   │  git checkout      │
   │               │                  │                  │                   │  FETCH_HEAD        │
   │               │                  │                  │                   │  (detached HEAD)   │
   │               │                  │                  │                   │                    │
   │               │                  │                  │                   │  UPDATE test_runs  │
   │               │                  │                  │                   │  SET status =      │
   │               │                  │                  │                   │  'running'         │
   │               │                  │                  │                   │────────────────>   │
   │               │                  │                  │                   │  (via PostgreSQL)  │
   │               │                  │                  │                   │                    │
   │               │                  │                  │                   │  docker create     │
   │               │                  │                  │                   │  -v local_path:    │
   │               │                  │                  │                   │  /workspace:ro     │
   │               │                  │                  │                   │───────────────────>│
   │               │                  │                  │                   │                    │
   │               │                  │                  │                   │                    │  run test
   │               │                  │                  │                   │                    │  command
   │               │                  │                  │                   │                    │  in /workspace
   │               │                  │                  │                   │                    │
   │               │                  │                  │                   │  stdout/stderr     │
   │               │                  │                  │                   │<───────────────────│
   │               │                  │                  │                   │                    │
   │               │                  │                  │                   │  docker rm         │
   │               │                  │                  │                   │  (cleanup)         │
   │               │                  │                  │                   │───────────────────>│
   │               │                  │                  │                   │                    │
   │               │                  │                  │                   │  Parse test output │
   │               │                  │                  │                   │  INSERT results    │
   │               │                  │                  │                   │  UPDATE test_runs  │
   │               │                  │                  │                   │  SET status, time  │
   │               │                  │                  │                   │────────────────>   │
   │               │                  │                  │                   │  (via PostgreSQL)  │
   │               │                  │                  │                   │                    │
   │               │                  │                  │  DEL active lock  │                    │
   │               │                  │                  │  Process next job │                    │
   │               │                  │                  │  in repo queue    │                    │
   │               │                  │                  │<──────────────────│                    │
   │               │                  │                  │                   │                    │
   │  GET /api/v1/ │                  │                  │                   │                    │
   │  runs/{id}    │                  │                  │                   │                    │
   │  (polling)    │                  │                  │                   │                    │
   │──────────────>│                  │                  │                   │                    │
   │               │  SELECT ...      │                  │                   │                    │
   │               │─────────────────>│                  │                   │                    │
   │               │                  │                  │                   │                    │
   │  200 OK       │                  │                  │                   │                    │
   │  {status:     │                  │                  │                   │                    │
   │   passed,     │                  │                  │                   │                    │
   │   results:[]} │                  │                  │                   │                    │
   │<──────────────│                  │                  │                   │                    │
```

**Step-by-step breakdown:**

| Step | Actor | Action |
|------|-------|--------|
| 1 | Client | Sends `POST /api/v1/suites/{id}/runs` with `branch` (commit SHA resolved from branch HEAD) |
| 2 | Backend API | **Commit-hash cache check:** queries for existing run with same suite + branch + commit SHA. If found and `force` is not set, returns cached result immediately |
| 3 | Backend API | Creates a `test_runs` row with `status = 'queued'`, increments `run_number` per suite+branch |
| 4 | Backend API | Pushes job onto the per-repo Redis queue (`LPUSH verdox:jobs:repo:{repo_id}`) |
| 5 | Backend API | Returns `202 Accepted` with the `run_id` immediately (non-blocking) |
| 6 | Worker goroutine | Attempts to acquire the per-repo active lock (`SETNX verdox:jobs:active:{repo_id}`) |
| 7 | Worker | Pops job from the repo's queue (`RPOP verdox:jobs:repo:{repo_id}`) |
| 8 | Worker | Fetches the target branch on the local clone: `git fetch --depth 1 origin {branch}` |
| 9 | Worker | Checks out `FETCH_HEAD` (detached HEAD — no branch cleanup needed) |
| 10 | Worker | Updates `test_runs` status to `'running'` in PostgreSQL |
| 11 | Worker | Creates ephemeral DinD container with local clone mounted read-only (`-v {local_path}:/workspace:ro`) |
| 12 | Container | Executes the configured test command (e.g., `go test ./...`, `npm test`) in `/workspace` |
| 13 | Worker | Captures stdout/stderr streams, removes ephemeral container |
| 14 | Worker | Parses test output into individual test case results, batch-inserts `test_results` rows |
| 15 | Worker | Updates `test_runs` to `'passed'`/`'failed'`, sets `finished_at` timestamp |
| 16 | Worker | Releases the per-repo active lock (`DEL verdox:jobs:active:{repo_id}`), processes next queued job if any |
| 17 | Frontend | Polls `GET /api/v1/runs/{id}` on an interval and renders results when complete |

**Key design decisions:**

- **Per-repo sequential queue:** Only one test run executes per repo at a time (prevents git checkout conflicts on the shared local clone). Different repos run in parallel up to `RUNNER_MAX_CONCURRENT`.
- **Commit-hash caching:** Avoids redundant runs when the branch HEAD hasn't changed.
- **Read-only mount:** The local clone is mounted as `:ro` into the DinD container — tests cannot modify the source.
- **Detached HEAD:** `git checkout FETCH_HEAD` avoids creating local branch refs that need cleanup.
- **Permission:** Only root, moderator, team admin, or team maintainer can trigger runs.

---

### 4d. Team Permission Check Flow

```
 Request              Auth Middleware         Role Middleware        Team Check            Handler
   │                       │                      │                     │                    │
   │  ANY /api/v1/teams/   │                      │                     │                    │
   │  {team_id}/...        │                      │                     │                    │
   │──────────────────────>│                      │                     │                    │
   │                       │                      │                     │                    │
   │                       │  Extract JWT from    │                     │                    │
   │                       │  cookie / header     │                     │                    │
   │                       │                      │                     │                    │
   │                       │  Validate signature  │                     │                    │
   │                       │  + expiration        │                     │                    │
   │                       │                      │                     │                    │
   │                       │  Check session in    │                     │                    │
   │                       │  Redis (not revoked) │                     │                    │
   │                       │                      │                     │                    │
   │                       │  Set user in context │                     │                    │
   │                       │─────────────────────>│                     │                    │
   │                       │                      │                     │                    │
   │                       │                      │  Check user.role    │                    │
   │                       │                      │  (root/moderator/   │                    │
   │                       │                      │   user)             │                    │
   │                       │                      │                     │                    │
   │                       │                      │  If root/moderator: │                    │
   │                       │                      │  skip team check    │                    │
   │                       │                      │─────────────────────────────────────────>│
   │                       │                      │                     │                    │
   │                       │                      │  If user: check     │                    │
   │                       │                      │  team membership    │                    │
   │                       │                      │  + team role        │                    │
   │                       │                      │────────────────────>│                    │
   │                       │                      │                     │                    │
   │                       │                      │                     │  SELECT FROM       │
   │                       │                      │                     │  team_members      │
   │                       │                      │                     │  WHERE user_id     │
   │                       │                      │                     │  AND team_id       │
   │                       │                      │                     │  AND status =      │
   │                       │                      │                     │  'approved'        │
   │                       │                      │                     │                    │
   │                       │                      │                     │  Check team_role   │
   │                       │                      │                     │  (admin/maintainer │
   │                       │                      │                     │   /viewer) meets   │
   │                       │                      │                     │  required level    │
   │                       │                      │                     │────────────────────>
   │                       │                      │                     │                    │
   │                       │                      │                     │                    │  Execute
   │                       │                      │                     │                    │  handler
   │                       │                      │                     │                    │  logic
   │                       │                      │                     │                    │
```

**Middleware chain:** `AuthMiddleware` -> `RequireRole(roles...)` -> `RequireTeamRole(teamRoles...)` -> `Handler`

- **System roles:** root > moderator > user. Root and moderator bypass team membership checks.
- **Team roles:** admin > maintainer > viewer. Only approved members are checked.
- Write operations (trigger runs, manage members) require team admin or maintainer role.
- Read operations (view results, list suites) require at minimum viewer role.
- If any middleware step fails, the request is rejected with `401 Unauthorized` or `403 Forbidden`.

---

## 5. Network Topology

```
 Host Machine
 ┌──────────────────────────────────────────────────────────────────┐
 │                                                                  │
 │   Ports 80, 443 ──> ┌──────────────────────────────────────┐    │
 │                      │       verdox-network (bridge)        │    │
 │                      │                                      │    │
 │                      │   ┌────────────┐                     │    │
 │                      │   │   Nginx    │ ◄── only service    │    │
 │                      │   │  :80/:443  │     with host port  │    │
 │                      │   └─────┬──────┘     mapping         │    │
 │                      │         │                            │    │
 │                      │    ┌────┴─────┐                      │    │
 │                      │    │          │                      │    │
 │                      │  ┌─┴──────┐ ┌─┴───────┐             │    │
 │                      │  │Next.js │ │ Go API  │             │    │
 │                      │  │ :3000  │ │  :8080  │             │    │
 │                      │  └────────┘ └────┬────┘             │    │
 │                      │                  │                   │    │
 │                      │         ┌────────┼────────┐         │    │
 │                      │         │        │        │         │    │
 │                      │     ┌───┴───┐ ┌──┴──┐ ┌───┴───┐    │    │
 │                      │     │Postgres│ │Redis│ │  DinD │    │    │
 │                      │     │ :5432 │ │:6379│ │Runner │    │    │
 │                      │     └───────┘ └─────┘ └───────┘    │    │
 │                      │                                      │    │
 │                      └──────────────────────────────────────┘    │
 │                                                                  │
 └──────────────────────────────────────────────────────────────────┘
```

**Network rules:**

| Rule | Detail |
|------|--------|
| External access | Only Nginx exposes ports 80 and 443 to the host |
| Internal services | Frontend, backend, PostgreSQL, and Redis have no host port mappings in production |
| DinD isolation | The runner sits on the same bridge network but is restricted: it cannot initiate connections to PostgreSQL or Redis directly. It only communicates back to the backend via a callback API endpoint |
| DNS resolution | Services reference each other by container name (e.g., `postgres://verdox-postgres:5432`) thanks to Docker's built-in DNS |

---

## 6. Communication Protocols

| From | To | Protocol | Details |
|------|----|----------|---------|
| Client (Browser) | Nginx | HTTPS | TLS 1.2+ enforced, HTTP/2 enabled, HSTS header set |
| Nginx | Frontend | HTTP | Reverse proxy to `verdox-frontend:3000`, `X-Forwarded-*` headers set |
| Nginx | Backend | HTTP | Reverse proxy `/api/*` to `verdox-backend:8080`, WebSocket upgrade support for future use |
| Frontend (SSR) | Backend | HTTP | Server-side `fetch()` calls to `http://verdox-backend:8080` (internal Docker DNS, no TLS needed) |
| Backend | PostgreSQL | TCP | `lib/pq` driver, connection pool (`max_open_conns`, `max_idle_conns` configured), SSL mode optional internally |
| Backend | Redis | TCP | `go-redis/v9` client, connection pool, `DB 0` for sessions/cache, `DB 1` for job queue |
| Backend | DinD Runner | Unix socket / TCP | Docker client API via `/var/run/docker.sock` mount or TCP `tcp://verdox-runner:2376` with TLS client certs |
| Backend | GitHub | HTTPS | GitHub REST API v3 only (commit listing), team's PAT decrypted at call time, respects `X-RateLimit-*` headers. Git operations (clone, fetch) use PAT via HTTPS credential embedding |

---

## 7. Scaling Considerations

| Component | Current Design | Scaling Path |
|-----------|---------------|--------------|
| **Frontend (Next.js)** | Single container | Stateless; horizontally scalable by adding instances behind a load balancer. No sticky sessions needed |
| **Backend (Go/Echo)** | Single container | Stateless (JWT-based auth); horizontally scalable. Multiple instances can share the same PostgreSQL and Redis. No sticky sessions required |
| **PostgreSQL** | Single instance, no replicas | Vertical scaling (CPU/RAM) first. Add read replicas for read-heavy workloads. Consider PgBouncer for connection pooling at scale |
| **Redis** | Single instance | Sufficient for expected load (thousands of jobs/day). Add Redis Sentinel for high availability. Consider Redis Cluster only at very high throughput |
| **DinD Runner** | Worker goroutines within backend | Scale by increasing `MAX_CONCURRENT_RUNNERS` env var (controls goroutine pool size). Each goroutine manages one ephemeral container. For multi-node scaling, extract the worker into a separate service |
| **Nginx** | Single instance | Sufficient for single-node. In a multi-node setup, replace with a cloud load balancer or HAProxy in front of multiple Nginx instances |

**Current architecture constraint:** Single-node Docker Compose deployment. This
is intentional -- Verdox targets small-to-medium teams who want simplicity over
distributed system complexity. Migration to Kubernetes is possible but not
planned.

---

## 8. Technology Decision Rationale

| Decision | Choice | Rationale |
|----------|--------|-----------|
| **HTTP framework** | Echo v4 over Gin | Cleaner middleware API with a unified error handling model (`echo.HTTPError`). Equally performant in benchmarks. Better support for custom context and request-scoped values |
| **Test execution** | Docker-in-Docker | Full isolation per test run -- no host contamination, no shared filesystem state. Supports any language or test framework by swapping the base image. Containers are ephemeral and cleaned up after every run |
| **Job queue** | Redis LIST/STREAM | No additional infrastructure required. `BRPOP` provides reliable blocking dequeue. Can upgrade to a dedicated library like Asynq (built on Redis) if features like retries, scheduled jobs, or dead-letter queues are needed |
| **Frontend framework** | Next.js 15 App Router | React Server Components reduce client-side JavaScript bundle size. Built-in file-based routing with layouts simplifies page structure. Strong developer experience with hot reload, TypeScript support, and integrated API routes for BFF patterns |
| **Database** | PostgreSQL 17 | Proven reliability for transactional workloads. Excellent Go driver support (`lib/pq`, `pgx`). JSONB columns allow flexible storage for test metadata and configuration without schema changes. Strong indexing and query planner for analytical queries on test results |
| **Reverse proxy** | Nginx | Industry standard for TLS termination and static asset caching. Simple configuration for path-based routing. Built-in rate limiting via `limit_req_zone`. Low memory footprint |
| **Cache layer** | Redis 7 | Serves triple duty (session cache, job queue, API cache) eliminating the need for separate systems. Redis Streams provide a more robust alternative to LIST-based queues when needed. Sub-millisecond latency for session lookups |
