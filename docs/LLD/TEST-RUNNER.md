# Verdox -- Fork-Based Test Runner (LLD)

> Go 1.25+ | GitHub Actions | Redis 7 job queue | PostgreSQL 17

---

## 1. Overview

Tests in Verdox run on GitHub Actions via Verdox-managed forks. A dedicated
service account forks the target repository, pushes a Verdox workflow file
(`verdox-test.yml`), and dispatches the workflow using the `workflow_dispatch`
event. The backend polls GitHub Actions for run completion and downloads
logs/artifacts to store test results.

There is no Docker-in-Docker runner, no privileged container, and no local
compute required for test execution. All test workloads run on GitHub-hosted
runners (or self-hosted runners configured on the fork).

---

## 2. Architecture

```
                                 +-----------+
                                 |  Client   |
                                 +-----+-----+
                                       |
                                       | POST /api/v1/suites/:id/run
                                       v
                              +--------+--------+
                              |   Backend API   |
                              |   (Echo v4)     |
                              +---+--------+----+
                                  |        |
             INSERT test_runs     |        |  LPUSH job payload
             (status: queued)     |        |
                                  v        v
                          +-------+--+ +---+----------+
                          |PostgreSQL | |    Redis     |
                          |          | |              |
                          +-------+--+ +---+----------+
                                  ^        |
                                  |        | Pop job
                                  |        v
                                  |  +-----+-----------+
                                  |  |   Worker Pool   |
                                  |  |  (goroutines)   |
                                  |  +-----+-----------+
                                  |        |
             UPDATE test_runs,    |        | ForkGHAExecutor
             INSERT test_results  |        v
                                  |  +-----+-----------+
                                  +--+ GitHub API      |
                                     | (via service    |
                                     |  account PAT)   |
                                     |                 |
                                     |  Fork repo      |
                                     |  Push workflow   |
                                     |  Dispatch run    |
                                     |  Poll status     |
                                     |  Download logs   |
                                     +-----------------+
                                            |
                                            v
                                     +-----------------+
                                     | GitHub Actions  |
                                     | Runner          |
                                     | (executes tests)|
                                     +-----------------+
```

**Data flow summary:**

1. The API handler creates a `test_runs` row with `status = 'queued'` and
   pushes a job payload onto the Redis queue.
2. A worker goroutine picks the job, and the `ForkGHAExecutor` takes over.
3. The executor ensures a fork exists under the service account, syncs it
   with upstream, pushes the `verdox-test.yml` workflow, and dispatches the
   workflow via `workflow_dispatch`.
4. The `GHAPoller` polls the GitHub Actions API for workflow run completion.
5. On completion, the worker downloads workflow logs/artifacts, parses test
   output, writes `test_results` rows, and updates the `test_runs` status.

---

## 3. Key Components

### 3a. ForkService

Manages the lifecycle of Verdox-managed forks under the service account.

**Responsibilities:**

- Fork a repository under the service account (if not already forked)
- Sync the fork with upstream before each test run
- Push/update the `verdox-test.yml` workflow file in the fork
- Track fork metadata in the `repository_forks` table

```go
type ForkService struct {
    ghClient *github.Client   // authenticated with VERDOX_SERVICE_ACCOUNT_PAT
    db       *sql.DB
}

func (s *ForkService) EnsureFork(ctx context.Context, repo *model.Repository) (*model.Fork, error) {
    // 1. Check if fork already exists in DB
    // 2. If not, call GitHub API: POST /repos/{owner}/{repo}/forks
    // 3. Wait for fork to be ready (GitHub forks are async)
    // 4. Store fork metadata in repository_forks table
    // 5. Return fork info
}

func (s *ForkService) SyncUpstream(ctx context.Context, fork *model.Fork) error {
    // POST /repos/{fork_owner}/{fork_repo}/merge-upstream
    // body: { "branch": "main" }
}

func (s *ForkService) PushWorkflow(ctx context.Context, fork *model.Fork, suite *model.TestSuite) error {
    // Use GitHub Contents API to create/update .github/workflows/verdox-test.yml
    // PUT /repos/{fork_owner}/{fork_repo}/contents/.github/workflows/verdox-test.yml
}
```

### 3b. ForkGHAExecutor

Orchestrates a single test run via GitHub Actions on the fork.

**Responsibilities:**

- Coordinate with ForkService to ensure fork is ready
- Dispatch the workflow via `workflow_dispatch`
- Track the workflow run ID for polling

```go
type ForkGHAExecutor struct {
    forkService *ForkService
    ghClient    *github.Client
    poller      *GHAPoller
}

func (e *ForkGHAExecutor) Execute(ctx context.Context, job *model.JobPayload) (*model.RunResult, error) {
    // 1. EnsureFork (creates fork if needed)
    // 2. SyncUpstream (merge upstream changes)
    // 3. PushWorkflow (push verdox-test.yml with suite config)
    // 4. Dispatch workflow:
    //    POST /repos/{fork_owner}/{repo}/actions/workflows/verdox-test.yml/dispatches
    //    body: { "ref": "{branch}", "inputs": { "run_id": "{run_id}", ... } }
    // 5. Poll for the workflow run to appear and track its run_id
    // 6. Wait for completion via GHAPoller
    // 7. Download logs and artifacts
    // 8. Parse and return results
}
```

### 3c. GHAPoller

Polls GitHub Actions API for workflow run status.

**Responsibilities:**

- Poll `GET /repos/{fork_owner}/{repo}/actions/runs/{run_id}` at intervals
- Detect completion (status: `completed`, conclusion: `success`/`failure`)
- Handle timeouts and cancellations
- Optionally receive webhook callbacks for faster notification

```go
type GHAPoller struct {
    ghClient     *github.Client
    pollInterval time.Duration  // default: 15 seconds
    maxPollTime  time.Duration  // default: 30 minutes
}

func (p *GHAPoller) WaitForCompletion(ctx context.Context, fork *model.Fork, runID int64) (*github.WorkflowRun, error) {
    // Poll loop:
    //   GET /repos/{fork_owner}/{repo}/actions/runs/{run_id}
    //   Check status == "completed"
    //   If completed, return the run with conclusion
    //   If timeout exceeded, cancel the workflow and return error
}
```

### 3d. WorkerPool

Queue consumer that dispatches jobs to the ForkGHAExecutor.

**Responsibilities:**

- Maintain a pool of worker goroutines
- Pop jobs from per-repo Redis queues
- Enforce per-repo sequential execution (one active run per repo)
- Dispatch each job to ForkGHAExecutor

```go
type WorkerPool struct {
    executor *ForkGHAExecutor
    queue    *RedisQueue
    workers  int  // controlled by RUNNER_MAX_CONCURRENT
}

func (p *WorkerPool) Start(ctx context.Context) {
    for i := 0; i < p.workers; i++ {
        go p.runWorker(ctx, i)
    }
}

func (p *WorkerPool) runWorker(ctx context.Context, id int) {
    for {
        job, err := p.queue.Pop(ctx, fmt.Sprintf("worker-%d", id))
        if err != nil || job == nil {
            continue
        }
        result, err := p.executor.Execute(ctx, job)
        // Store results, update test_runs status
        // Release per-repo active lock
    }
}
```

---

## 4. Workflow File (`verdox-test.yml`)

Verdox pushes this workflow file to each fork. It is generated dynamically
based on the test suite configuration.

```yaml
name: Verdox Test Run
on:
  workflow_dispatch:
    inputs:
      run_id:
        description: "Verdox test run ID"
        required: true
      branch:
        description: "Branch to test"
        required: true
      test_command:
        description: "Test command to execute"
        required: true
      webhook_url:
        description: "Callback URL for results"
        required: false

jobs:
  test:
    runs-on: ubuntu-latest
    timeout-minutes: 30
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          ref: ${{ github.event.inputs.branch }}

      - name: Run tests
        id: test
        run: ${{ github.event.inputs.test_command }}

      - name: Notify Verdox (optional webhook)
        if: always() && github.event.inputs.webhook_url != ''
        run: |
          curl -s -X POST "${{ github.event.inputs.webhook_url }}" \
            -H "Content-Type: application/json" \
            -d "{\"run_id\": \"${{ github.event.inputs.run_id }}\", \"status\": \"${{ job.status }}\"}"
```

**Notes:**

- The workflow is pushed to `.github/workflows/verdox-test.yml` on the fork.
- The `workflow_dispatch` event allows Verdox to pass parameters (run ID,
  branch, test command) when dispatching.
- An optional webhook callback provides faster completion notification than
  polling alone.
- The workflow file is regenerated and updated when the test suite
  configuration changes.

---

## 5. Configuration

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `VERDOX_SERVICE_ACCOUNT_PAT` | Yes | -- | GitHub PAT for the Verdox service account. Must have `repo`, `workflow`, and `read:org` scopes |
| `VERDOX_SERVICE_ACCOUNT_USERNAME` | Yes | -- | GitHub username of the service account (e.g., `verdox-bot`) |
| `VERDOX_WEBHOOK_BASE_URL` | No | -- | Base URL for webhook callbacks from GHA runs (e.g., `https://verdox.example.com/api/v1/webhooks/gha`). If not set, polling-only mode is used |
| `RUNNER_MAX_CONCURRENT` | No | `5` | Maximum number of concurrent workflow dispatches |
| `RUNNER_POLL_INTERVAL` | No | `15s` | Interval between GHA status polls |
| `RUNNER_MAX_TIMEOUT` | No | `1800` | Maximum time (seconds) to wait for a GHA run to complete |

### PAT Hierarchy

| PAT | Purpose | Scope |
|-----|---------|-------|
| **Service account PAT** (`VERDOX_SERVICE_ACCOUNT_PAT`) | Forking repos, pushing workflow files, dispatching workflows, polling status, downloading artifacts | `repo`, `workflow`, `read:org` |
| **Team PAT** (optional, per-team in DB) | Accessing private repos that the service account cannot see. Used for repo validation and branch/commit listing | `repo` (or fine-grained with Contents read) |

The service account PAT is the primary credential for all fork and execution
operations. If a repository is private and the service account does not have
access, the team PAT is used to validate the repo and the service account must
be granted access (e.g., added as a collaborator) before tests can run.

---

## 6. Job Queue Design

The queue uses Redis `LIST` operations for simple, reliable FIFO dispatch.

### Redis Keys

| Key | Type | Purpose |
|-----|------|---------|
| `verdox:jobs:repo:{repo_id}` | LIST | Per-repo FIFO queue of pending job payloads |
| `verdox:jobs:active:{repo_id}` | STRING | Currently active run_id for a repo (exists only while a run is in progress) |
| `verdox:gha:run:{run_id}` | STRING | GitHub Actions workflow run ID mapped to Verdox run ID |

### Per-Repo Sequential Execution

Per-repo serialization prevents conflicting workflow dispatches. Multiple repos
can run in parallel, but a single repo processes one test run at a time.

- **Queue key:** Each repository gets its own queue at `verdox:jobs:repo:{repo_id}`.
- **Active lock:** Before a worker processes a job, it checks `verdox:jobs:active:{repo_id}`.
  - If the key exists: another run is active for that repo. The worker skips
    this job and tries the next repo's queue.
  - If the key does not exist: the worker sets the key with the `run_id` and a
    TTL of 2x `RUNNER_MAX_TIMEOUT` (default: 60 min), then processes the job.
- **On completion:** The worker deletes `verdox:jobs:active:{repo_id}`.
- **Cross-repo parallelism:** Runs across different repos execute in parallel,
  up to `RUNNER_MAX_CONCURRENT`.

### Job Payload

```json
{
  "test_run_id": "ddd44444-5555-6666-7777-888899990000",
  "test_suite_id": "aaa11111-2222-3333-4444-555566667777",
  "repo_id": "bbb22222-3333-4444-5555-666677778888",
  "repository_full_name": "owner/repo",
  "branch": "feature/add-auth",
  "commit_hash": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
  "test_command": "go test -v -json ./...",
  "timeout_seconds": 300
}
```

| Field | Type | Source | Notes |
|-------|------|--------|-------|
| `test_run_id` | UUID | Generated at enqueue time | Primary key in `test_runs` table |
| `test_suite_id` | UUID | From `test_suites.id` | Used to look up suite configuration |
| `repo_id` | UUID | From `repositories.id` | Used as the per-repo queue key and active-lock key |
| `repository_full_name` | string | From `repositories.github_full_name` | `owner/repo` format, used for GitHub API calls |
| `branch` | string | From API request body | Branch to test |
| `commit_hash` | string (40 hex) | From API request body | Exact commit SHA to checkout |
| `test_command` | string | From `test_suites.command` | The test command to run in the workflow |
| `timeout_seconds` | int | From `test_suites.timeout_seconds` | Per-suite timeout, default 300 |

---

## 7. Execution Flow

### Step-by-Step

| Step | Actor | Action |
|------|-------|--------|
| 1 | Client | `POST /api/v1/suites/{id}/runs` with `branch` |
| 2 | Backend API | Validates permissions (root/moderator/admin/maintainer) |
| 3 | Backend API | Creates `test_runs` row with `status = 'queued'` |
| 4 | Backend API | Pushes job onto per-repo Redis queue (`LPUSH`) |
| 5 | Backend API | Returns `202 Accepted` with `run_id` |
| 6 | Worker | Pops job from queue, acquires per-repo active lock |
| 7 | ForkService | `EnsureFork` -- forks repo under service account if needed |
| 8 | ForkService | `SyncUpstream` -- merges upstream changes into fork |
| 9 | ForkService | `PushWorkflow` -- creates/updates `verdox-test.yml` on fork |
| 10 | ForkGHAExecutor | Dispatches workflow: `POST /repos/{fork}/actions/workflows/verdox-test.yml/dispatches` |
| 11 | Worker | Updates `test_runs.status = 'running'` |
| 12 | GHAPoller | Polls `GET /repos/{fork}/actions/runs` until workflow completes |
| 13 | Worker | Downloads workflow run logs via `GET /repos/{fork}/actions/runs/{id}/logs` |
| 14 | Worker | Parses test output, batch-inserts `test_results` rows |
| 15 | Worker | Updates `test_runs` to `'passed'`/`'failed'`, sets `finished_at` |
| 16 | Worker | Releases per-repo active lock, processes next queued job |
| 17 | Frontend | Polls `GET /api/v1/runs/{id}` and renders results when complete |

### Flow Diagram

```
User triggers run
       |
       v
  +----+----+
  |  Queue   |  LPUSH to verdox:jobs:repo:{repo_id}
  +----+----+
       |
       v
  +----+----+
  |  Worker  |  Pop from queue, acquire lock
  +----+----+
       |
       v
+------+----------+
| ForkGHAExecutor  |
|                  |
|  1. EnsureFork   |----> GitHub API: POST /repos/{owner}/{repo}/forks
|  2. SyncUpstream |----> GitHub API: POST /repos/{fork}/merge-upstream
|  3. PushWorkflow |----> GitHub API: PUT /repos/{fork}/contents/...
|  4. Dispatch     |----> GitHub API: POST /repos/{fork}/actions/workflows/.../dispatches
+------+-----------+
       |
       v
  +----+-----+
  | GHA Runs  |  GitHub Actions executes tests on runner
  +----+-----+
       |
       v
  +----+-----+
  | GHAPoller |  Polls GET /repos/{fork}/actions/runs/{id}
  +----+-----+
       |
       v
  +----+----------+
  | Results stored |  Download logs, parse, INSERT test_results, UPDATE test_runs
  +---------------+
```

---

## 8. Webhook Callback (Optional)

If `VERDOX_WEBHOOK_BASE_URL` is configured, the workflow includes a callback
step that notifies Verdox when the run completes. This reduces polling latency.

**Endpoint:** `POST /api/v1/webhooks/gha`

```json
{
  "run_id": "ddd44444-5555-6666-7777-888899990000",
  "status": "success"
}
```

When a webhook is received, the backend immediately triggers log/artifact
download and result parsing instead of waiting for the next poll cycle. The
poller acts as a fallback in case the webhook fails to deliver.

---

## 9. Permission Model

Only certain team roles are authorized to trigger test runs via
`POST /api/v1/suites/:id/run`:

| Role | Can Trigger Runs | Scope |
|------|-----------------|-------|
| `root` | Yes | Any repo across the instance |
| `moderator` | Yes | Any repo across the instance |
| `admin` | Yes | Repos within their team |
| `maintainer` | Yes | Repos within their team |
| `viewer` | No | Cannot trigger runs (read-only access) |

---

## 10. Error Handling

| Scenario | Detection | Response |
|----------|-----------|----------|
| Fork creation fails | GitHub API error (e.g., rate limit, permissions) | Set `test_runs.status = 'failed'`, log error, retry eligible |
| Upstream sync fails | GitHub API `409 Conflict` or merge conflict | Set status to `'failed'`, notify user of upstream conflict |
| Workflow dispatch fails | GitHub API `404` (workflow not found) or `422` | Re-push workflow file, retry dispatch once |
| GHA run times out | Poll duration exceeds `RUNNER_MAX_TIMEOUT` | Cancel the workflow run via API, set status to `'timed_out'` |
| GHA run cancelled | Workflow conclusion is `cancelled` | Set status to `'cancelled'` |
| Log download fails | GitHub API error on artifact/log endpoint | Set status to `'failed'`, store partial results if available |
| Service account PAT invalid | `401` from GitHub API | Log critical error, fail the run, alert admin |
| GitHub API rate limit | `403` with `X-RateLimit-Remaining: 0` | Back off until rate limit resets, retry |

---

## 11. Database Schema (Fork Tracking)

```sql
-- Track Verdox-managed forks
CREATE TABLE repository_forks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    fork_owner VARCHAR(255) NOT NULL,       -- service account username
    fork_full_name VARCHAR(512) NOT NULL,   -- e.g., "verdox-bot/repo-name"
    github_fork_id BIGINT,                  -- GitHub's fork ID
    workflow_sha VARCHAR(64),               -- SHA of last pushed workflow file
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(repository_id)
);
```

---

## 12. Future Considerations

- **Self-hosted runners:** Support for organizations with self-hosted GHA
  runners on the fork. The workflow file can be customized to target specific
  runner labels.
- **Workflow caching:** Cache dependencies across runs using GHA's built-in
  cache action to speed up test execution.
- **Parallel test jobs:** Split test suites across multiple GHA jobs for
  faster execution on large test suites.
- **GitHub App integration:** Replace service account PAT with a GitHub App
  installation token for better permission scoping and automatic rotation.
- **Webhook-first mode:** Use GitHub webhooks (workflow_run events) instead
  of polling for faster result notification.
