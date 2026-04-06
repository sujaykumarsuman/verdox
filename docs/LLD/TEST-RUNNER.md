# Verdox -- Test Runner Low-Level Design (LLD)

> Go 1.25+ | Docker-in-Docker | Redis 7 job queue | PostgreSQL 17

---

## 1. Architecture Overview

The test runner is embedded in the Go backend process as a pool of worker
goroutines. There is no separate runner service. Workers pull jobs from a
Redis-backed FIFO queue and execute each test run inside an ephemeral Docker
container managed through the Docker Engine API.

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
                                  |        | BRPOP (blocking pop)
                                  |        v
                                  |  +-----+-----------+
                                  |  |   Worker Pool   |
                                  |  |  (goroutines)   |
                                  |  |                 |
                                  |  |  worker-0       |
                                  |  |  worker-1       |
                                  |  |  worker-2       |
                                  |  |  ...            |
                                  |  |  worker-(N-1)   |
                                  |  +-----+-----------+
                                  |        |
             UPDATE test_runs,    |        | Docker API
             INSERT test_results  |        v
                                  |  +-----+-----------+
                                  +--+  Docker Engine  |
                                     |  (DinD)         |
                                     |                 |
                                     |  +-----------+  |
                                     |  | Ephemeral |  |
                                     |  | Container |  |
                                     |  | (one per  |  |
                                     |  |  test run)|  |
                                     |  +-----------+  |
                                     +-----------------+
```

**Data flow summary:**

1. The API handler creates a `test_runs` row with `status = 'queued'` and
   pushes a job payload onto the Redis queue.
2. A worker goroutine picks the job via `BRPOP`, transitions the run to
   `'running'`, and creates an ephemeral Docker container.
3. The worker prepares the local clone (`git fetch --depth 1` + `git checkout
   FETCH_HEAD`), mounts it read-only into the container, runs the test
   command, and streams stdout/stderr back to the worker.
4. The worker parses the test output, writes `test_results` rows, updates
   the `test_runs` status to `'passed'` or `'failed'`, and removes the
   container.

---

## 2. Job Queue Design

The queue uses Redis `LIST` operations for simple, reliable FIFO dispatch.

### Redis Keys

| Key | Type | Purpose |
|-----|------|---------|
| `verdox:jobs:repo:{repo_id}` | LIST | Per-repo FIFO queue of pending job payloads |
| `verdox:jobs:active:{repo_id}` | STRING | Currently active run_id for a repo (exists only while a run is in progress) |
| `verdox:jobs:processing:{worker_id}` | LIST | In-flight job for a specific worker (crash recovery) |
| `verdox:logs:{run_id}` | STRING | Append-only log buffer for a running test (TTL 1 hour) |
| `verdox:logs:{run_id}:stream` | PUB/SUB | Real-time log streaming channel |
| `verdox:cancel:{run_id}` | PUB/SUB | Cancellation signal channel |

### Per-Repo Sequential Execution

Per-repo serialization eliminates checkout conflicts on the local clone. Multiple repos can run in parallel, but a single repo processes one test run at a time.

- **Queue key:** Each repository gets its own queue at `verdox:jobs:repo:{repo_id}`.
- **Active lock:** Before a worker processes a job, it checks `verdox:jobs:active:{repo_id}`.
  - If the key exists: another run is active for that repo. The worker skips this job and tries the next repo's queue (or waits).
  - If the key does not exist: the worker sets `verdox:jobs:active:{repo_id}` = `run_id` **with a TTL of 2x `RUNNER_TIMEOUT`** (default: 60 min) and processes the job.
  - The TTL acts as a safety net: if the worker crashes without releasing the lock, it auto-expires and unblocks the repo's queue. Crash recovery (Section 10) also cleans up stale locks proactively.
- **On completion:** The worker deletes `verdox:jobs:active:{repo_id}`.
- **Cross-repo parallelism:** Runs across different repos execute in parallel, up to `RUNNER_MAX_CONCURRENT`.
- **Same-repo guarantee:** Strictly sequential -- no concurrent checkout conflicts.

### Job Payload

The API handler serializes this JSON and pushes it with `LPUSH`:

```json
{
  "test_run_id": "ddd44444-5555-6666-7777-888899990000",
  "test_suite_id": "aaa11111-2222-3333-4444-555566667777",
  "repo_id": "bbb22222-3333-4444-5555-666677778888",
  "repository_full_name": "owner/repo",
  "local_path": "/var/lib/verdox/repositories/github.com/owner/repo",
  "default_branch": "main",
  "branch": "feature/add-auth",
  "commit_hash": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
  "test_type": "unit",
  "config_path": "./verdox.yaml",
  "timeout_seconds": 300
}
```

| Field | Type | Source | Notes |
|-------|------|--------|-------|
| `test_run_id` | UUID | Generated at enqueue time | Primary key in `test_runs` table |
| `test_suite_id` | UUID | From `test_suites.id` | Used to look up suite configuration |
| `repo_id` | UUID | From `repositories.id` | Used as the per-repo queue key and active-lock key |
| `repository_full_name` | string | From `repositories.github_full_name` | `owner/repo` format for identification |
| `local_path` | string | From `repositories.local_path` | Absolute path to the local clone on the host |
| `default_branch` | string | From `repositories.default_branch` | Default branch of the repo (e.g. `main`), used to reset after a run |
| `branch` | string | From API request body | Branch to fetch and checkout |
| `commit_hash` | string (40 hex chars) | From API request body | Exact commit to checkout |
| `test_type` | enum | From `test_suites.type` | `unit` or `integration` -- affects image and network config |
| `config_path` | string | From `test_suites.config_path` | Path to `verdox.yaml` in the repo (nullable) |
| `timeout_seconds` | int | From `test_suites.timeout_seconds` | Per-suite timeout, default 300 |

**PAT resolution:** Git fetch uses the team's PAT. The worker resolves the PAT via `repositories.team_id` -> `teams.github_pat_encrypted` -> decrypt -> use. This is a team-level credential; any team admin can rotate it without affecting other fields. No single-user dependency exists.

### Permission Model for Triggering Runs

Only certain team roles are authorized to trigger test runs via
`POST /api/v1/suites/:id/run`:

| Role | Can Trigger Runs | Scope |
|------|-----------------|-------|
| `root` | Yes | Any repo across the instance |
| `moderator` | Yes | Any repo across the instance |
| `admin` | Yes | Repos within their team |
| `maintainer` | Yes | Repos within their team |
| `viewer` | No | Cannot trigger runs (read-only access) |

The API handler checks the caller's team role before enqueuing the job.
Unauthorized requests receive `403 Forbidden`.

### Enqueue Operation (API Side)

```go
// internal/queue/queue.go

func (q *RedisQueue) Push(ctx context.Context, job *model.JobPayload) error {
    data, err := json.Marshal(job)
    if err != nil {
        return fmt.Errorf("marshal job payload: %w", err)
    }
    key := fmt.Sprintf("verdox:jobs:repo:%s", job.RepoID)
    return q.client.LPush(ctx, key, data).Err()
}
```

### Dequeue Operation (Worker Side)

Workers scan per-repo queues and skip repos that already have an active
run. This guarantees sequential execution within a single repo while
allowing parallel execution across different repos.

```go
func (q *RedisQueue) Pop(ctx context.Context, workerID string) (*model.JobPayload, error) {
    // Discover all per-repo queues.
    repoKeys, err := q.client.Keys(ctx, "verdox:jobs:repo:*").Result()
    if err != nil {
        return nil, fmt.Errorf("scan repo queues: %w", err)
    }
    if len(repoKeys) == 0 {
        time.Sleep(1 * time.Second) // no queues at all, avoid busy loop
        return nil, nil
    }

    for _, key := range repoKeys {
        // Extract repo_id from key "verdox:jobs:repo:{repo_id}".
        repoID := strings.TrimPrefix(key, "verdox:jobs:repo:")
        activeKey := fmt.Sprintf("verdox:jobs:active:%s", repoID)

        // Skip this repo if a run is already active.
        exists, _ := q.client.Exists(ctx, activeKey).Result()
        if exists > 0 {
            continue
        }

        // Try to pop a job from this repo's queue.
        raw, err := q.client.RPop(ctx, key).Result()
        if err == redis.Nil {
            continue // queue empty
        }
        if err != nil {
            return nil, fmt.Errorf("rpop %s: %w", key, err)
        }

        var job model.JobPayload
        if err := json.Unmarshal([]byte(raw), &job); err != nil {
            return nil, fmt.Errorf("unmarshal job payload: %w", err)
        }

        // Claim: set the active lock for this repo with TTL (2x RUNNER_TIMEOUT).
        lockTTL := 2 * time.Duration(q.runnerTimeout) * time.Minute
        q.client.Set(ctx, activeKey, job.TestRunID, lockTTL)

        // Record in this worker's processing list for crash recovery.
        q.client.LPush(ctx, fmt.Sprintf("verdox:jobs:processing:%s", workerID), raw)

        return &job, nil
    }

    // All repos either empty or have an active run. Wait briefly.
    time.Sleep(1 * time.Second)
    return nil, nil
}
```

### Acknowledgement

```go
func (q *RedisQueue) Ack(ctx context.Context, workerID string, job *model.JobPayload) error {
    data, _ := json.Marshal(job)

    // Release the per-repo active lock so the next queued run can proceed.
    q.client.Del(ctx, fmt.Sprintf("verdox:jobs:active:%s", job.RepoID))

    // Remove from the worker's processing list.
    return q.client.LRem(ctx, fmt.Sprintf("verdox:jobs:processing:%s", workerID), 1, data).Err()
}
```

### Reliability Guarantees

| Scenario | Behavior |
|----------|----------|
| Job dequeued successfully | `RPOP` removes from `verdox:jobs:repo:{repo_id}`; worker sets `verdox:jobs:active:{repo_id}` and copies to `verdox:jobs:processing:{worker_id}` |
| Job completed (pass or fail) | Worker calls `Ack()` to delete `verdox:jobs:active:{repo_id}` and remove from processing list |
| Worker crash mid-execution | Processing list retains the job; crash recovery scans and re-queues. Active lock is cleaned up during recovery (see Section 10) |
| Redis restart | Per-repo queues persist (Redis AOF/RDB). In-flight jobs in processing lists are recovered on worker startup |

---

## 3. Worker Pool

### Pool Configuration

| Config | Env Var | Default | Description |
|--------|---------|---------|-------------|
| Pool size | `RUNNER_MAX_CONCURRENT` | 5 | Maximum number of concurrent test runs (across different repos; same repo is always sequential) |
| BRPOP timeout | -- | 5 seconds | How long each worker blocks waiting for a job |
| Graceful shutdown deadline | -- | 60 seconds | Time allowed for in-progress runs to finish |

### Initialization

The worker pool is started by `cmd/server/main.go` alongside the HTTP server:

```go
// internal/runner/runner.go

type WorkerPool struct {
    size       int
    queue      *queue.RedisQueue
    executor   *Executor
    parser     *Parser
    db         *sqlx.DB
    cancelFunc context.CancelFunc
    wg         sync.WaitGroup
}

func NewWorkerPool(cfg config.RunnerConfig, q *queue.RedisQueue, db *sqlx.DB) *WorkerPool {
    return &WorkerPool{
        size:     cfg.MaxConcurrent,
        queue:    q,
        executor: NewExecutor(cfg),
        parser:   NewParser(),
        db:       db,
    }
}

func (wp *WorkerPool) Start(ctx context.Context) {
    ctx, wp.cancelFunc = context.WithCancel(ctx)

    for i := 0; i < wp.size; i++ {
        wp.wg.Add(1)
        go wp.runWorker(ctx, fmt.Sprintf("worker-%d", i))
    }
}
```

### Worker Lifecycle

Each worker is a goroutine that loops indefinitely until the context is
cancelled:

```go
func (wp *WorkerPool) runWorker(ctx context.Context, workerID string) {
    defer wp.wg.Done()

    log.Info().Str("worker", workerID).Msg("worker started")

    for {
        select {
        case <-ctx.Done():
            log.Info().Str("worker", workerID).Msg("worker stopping (context cancelled)")
            return
        default:
        }

        job, err := wp.queue.Pop(ctx, workerID)
        if err != nil {
            log.Error().Err(err).Str("worker", workerID).Msg("failed to pop job")
            continue
        }
        if job == nil {
            continue // BRPOP timed out, loop and try again
        }

        log.Info().
            Str("worker", workerID).
            Str("run_id", job.TestRunID).
            Msg("picked up job")

        wp.executeJob(ctx, workerID, job)
    }
}
```

### Graceful Shutdown

```go
func (wp *WorkerPool) Shutdown() {
    log.Info().Msg("shutting down worker pool")
    wp.cancelFunc() // signal all workers to stop accepting new jobs

    done := make(chan struct{})
    go func() {
        wp.wg.Wait() // wait for in-progress jobs to finish
        close(done)
    }()

    select {
    case <-done:
        log.Info().Msg("all workers stopped gracefully")
    case <-time.After(60 * time.Second):
        log.Warn().Msg("shutdown deadline exceeded, force killing remaining containers")
        wp.executor.ForceKillAll()
    }
}
```

**Shutdown sequence:**

1. `cancelFunc()` is called, causing all workers to exit their loop after
   the current job completes.
2. `wp.wg.Wait()` blocks until every worker goroutine returns.
3. If workers do not finish within 60 seconds, `ForceKillAll()` sends
   SIGKILL to all running containers and removes them.

---

## 4. Execution Flow (Step-by-Step)

This section describes the full lifecycle of a single test run from the
moment a worker picks it up to final cleanup.

### Step 1: Pick Job from Redis Queue

The worker calls `queue.Pop()`, which scans per-repo queues
(`verdox:jobs:repo:{repo_id}`). For each repo, it checks whether an active
run exists (`verdox:jobs:active:{repo_id}`). If no run is active, it pops
a job from that repo's queue, sets the active lock, and copies the raw JSON
to `verdox:jobs:processing:{worker_id}` for crash recovery.

### Step 2: Update Status to Running

```go
// internal/runner/runner.go -- inside executeJob()

now := time.Now().UTC()
_, err := wp.db.ExecContext(ctx,
    `UPDATE test_runs SET status = 'running', started_at = $1 WHERE id = $2`,
    now, job.TestRunID,
)
```

### Step 3: Create Docker Container

The executor creates a container through the Docker Engine API:

```go
// internal/runner/executor.go

func (e *Executor) CreateContainer(ctx context.Context, job *model.JobPayload) (string, error) {
    image := e.selectImage(job)

    resp, err := e.docker.ContainerCreate(ctx, &container.Config{
        Image:      image,
        WorkingDir: "/workspace",
        Env: []string{
            "CI=true",
            fmt.Sprintf("VERDOX_RUN_ID=%s", job.TestRunID),
        },
        Cmd: []string{"sh", "-c", e.buildScript(job)},
    }, &container.HostConfig{
        Binds: []string{
            fmt.Sprintf("%s:/workspace:ro", job.LocalPath), // mount local clone read-only
        },
        Resources: container.Resources{
            NanoCPUs: 2_000_000_000, // 2 CPU cores
            Memory:   2 * 1024 * 1024 * 1024, // 2 GB RAM
        },
        Tmpfs: map[string]string{
            "/tmp": "size=5G",
        },
        NetworkMode: e.networkMode(job),
    }, nil, nil, fmt.Sprintf("verdox-run-%s", job.TestRunID))

    return resp.ID, err
}
```

#### Image Selection

```go
func (e *Executor) selectImage(job *model.JobPayload) string {
    // 1. If verdox.yaml specifies an image, use it.
    if job.ConfigImage != "" {
        return job.ConfigImage
    }

    // 2. Default images by detected language / test type.
    defaults := map[string]string{
        "go":     "golang:1.25-alpine",
        "python": "python:3.12-slim",
        "node":   "node:22-alpine",
    }

    if img, ok := defaults[job.DetectedLanguage]; ok {
        return img
    }

    // 3. Ultimate fallback.
    return e.cfg.DefaultImage // RUNNER_DEFAULT_IMAGE, defaults to alpine:3.21
}
```

#### Container Configuration

| Setting | Value | Rationale |
|---------|-------|-----------|
| Working directory | `/workspace` | Standard mount point for the repository |
| Mount | `-v {local_path}:/workspace:ro` | Local clone mounted read-only to prevent test code from modifying the clone |
| `CI=true` | env var | Signals to test frameworks that they are running in CI |
| `VERDOX_RUN_ID` | env var | Allows test code to correlate with the Verdox run |
| CPU limit | 2 cores | Prevents a single test run from starving other workers |
| Memory limit | 2 GB | Prevents OOM on the host; container is OOM-killed at this threshold |
| Disk | 5 GB tmpfs on `/tmp` | Provides scratch space without polluting the host filesystem |
| Network (unit tests) | `none` | Full network isolation for unit tests |
| Network (integration) | `verdox-runner-net` | Allows access to sidecar services (Postgres, Redis) on the runner network |
| Privileged mode | disabled | Test containers never run in privileged mode |

**Writable workspace:** If tests need write access (e.g., build artifacts,
generated files), the container entrypoint should copy `/workspace` to a
writable location first (e.g., `cp -r /workspace /work && cd /work`) or
use a tmpfs overlay. The read-only mount ensures the local clone is never
modified by test code.

### Step 4: Prepare Workspace

The worker prepares the local clone on the host before creating the
container. No clone happens inside the container -- the local clone is
mounted as a read-only volume.

The initial clone (performed at repo registration time) uses shallow clone
by default: `git clone --depth 1 --branch {default_branch} <url>
{local_path}`. If `full_clone: true` is set in `verdox.yaml`, the initial
clone omits `--depth 1` and fetches the full repository history.

```
Step: Prepare Workspace
1. Verify local clone exists at {local_path} and clone_status = 'cloned'
2. Fetch the target branch:
   - If full_clone is true:  git fetch origin {branch}
   - If full_clone is false (default):  git fetch --depth 1 origin {branch}
3. git checkout FETCH_HEAD (detached HEAD, no branch tracking)
4. Mount {local_path} as /workspace:ro in container
```

The `full_clone` flag is read from `verdox.yaml` in the repository. When
enabled, the fetch retrieves full history for the branch, making `git log`,
`git blame`, and history-dependent tools available inside the container.
When disabled (the default), only the tip commit is fetched for faster
execution and lower disk usage.

**Git credential:** The `git fetch` uses the team's PAT
(`repositories.team_id` -> `teams.github_pat_encrypted` -> decrypt). The
PAT is passed via the credential helper or URL embedding and is never
exposed inside the container. If the PAT is revoked, any team admin can
set a new one -- there is no single-user dependency.

No `git branch -D` cleanup is needed since we use `FETCH_HEAD` (detached
HEAD), not a local branch. After the run completes, the cleanup step
returns the clone to `git checkout {default_branch}`.

This eliminates per-run clone time and GitHub API calls during test
execution. No `GITHUB_TOKEN` is needed in the container environment.

### Step 5: Run Test Command

The test command is determined by priority:

1. **Custom command from verdox.yaml** -- if `config_path` is set and the
   file contains a `command` field, use it.
2. **Default by language:**

| Language | Command | Output Format |
|----------|---------|---------------|
| Go | `go test -v -json ./...` | NDJSON (one JSON object per line) |
| Python | `python -m pytest --tb=short -v` | pytest stdout |
| Node.js | `npm test -- --verbose` | Jest/Mocha stdout |

3. **Fallback** -- `sh -c "echo 'No test command configured'; exit 1"`

### Step 6: Capture Output

The worker attaches to the container's stdout and stderr streams via the
Docker API and reads them in real-time:

```go
func (e *Executor) StreamLogs(ctx context.Context, containerID string, runID string) (string, error) {
    reader, err := e.docker.ContainerLogs(ctx, containerID, container.LogsOptions{
        ShowStdout: true,
        ShowStderr: true,
        Follow:     true,
    })
    if err != nil {
        return "", err
    }
    defer reader.Close()

    var buf bytes.Buffer
    scanner := bufio.NewScanner(reader)

    for scanner.Scan() {
        line := scanner.Text()
        buf.WriteString(line)
        buf.WriteByte('\n')

        // Append to Redis for real-time streaming.
        e.redis.Append(ctx, fmt.Sprintf("verdox:logs:%s", runID), line+"\n")

        // Publish to the streaming channel.
        e.redis.Publish(ctx, fmt.Sprintf("verdox:logs:%s:stream", runID), line)
    }

    return buf.String(), scanner.Err()
}
```

### Step 7: Wait for Completion or Timeout

```go
func (e *Executor) WaitWithTimeout(ctx context.Context, containerID string, timeout time.Duration) (int64, error) {
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    statusCh, errCh := e.docker.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)

    select {
    case err := <-errCh:
        return -1, err
    case status := <-statusCh:
        return status.StatusCode, nil
    case <-ctx.Done():
        // Timeout exceeded. Kill the container.
        _ = e.docker.ContainerKill(context.Background(), containerID, "SIGKILL")
        return -1, fmt.Errorf("timeout exceeded")
    }
}
```

### Step 8: Parse Test Output

The raw stdout/stderr is passed to the parser (see Section 5). The parser
returns a slice of `TestResult` structs.

### Step 9: Write Results to Database

```go
// Batch insert all test results in a single transaction.
tx, err := wp.db.BeginTxx(ctx, nil)
if err != nil {
    return fmt.Errorf("begin tx: %w", err)
}

for _, result := range results {
    _, err := tx.ExecContext(ctx,
        `INSERT INTO test_results (id, test_run_id, test_name, status, duration_ms, error_message, log_output, created_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7, now())`,
        uuid.New(), job.TestRunID, result.TestName, result.Status,
        result.DurationMs, result.ErrorMessage, result.LogOutput,
    )
    if err != nil {
        tx.Rollback()
        return fmt.Errorf("insert test result: %w", err)
    }
}

if err := tx.Commit(); err != nil {
    return fmt.Errorf("commit results: %w", err)
}
```

### Step 10: Update Test Run Status

```go
finalStatus := "passed"
for _, r := range results {
    if r.Status == "fail" || r.Status == "error" {
        finalStatus = "failed"
        break
    }
}

_, err = wp.db.ExecContext(ctx,
    `UPDATE test_runs SET status = $1, finished_at = $2 WHERE id = $3`,
    finalStatus, time.Now().UTC(), job.TestRunID,
)
```

**Status determination rules:**

| Condition | `test_runs.status` | `error_message` |
|-----------|-------------------|-----------------|
| All tests pass | `passed` | NULL |
| Any test has status `fail` or `error` | `failed` | NULL (individual errors in `test_results`) |
| Container timeout | `failed` | `"Test run exceeded timeout of {N} seconds"` |
| Container exit code != 0, no parseable output | `failed` | `"Container exited with code {X}"` |
| Container OOM killed (exit code 137) | `failed` | `"Out of memory: container killed (exit code 137)"` |
| Docker engine unavailable | `failed` | `"Docker engine unavailable"` |
| Run cancelled by user | `cancelled` | NULL |

### Step 11: Cleanup

```go
// Always runs in a defer block at the start of executeJob().
defer func() {
    removeCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    _ = e.docker.ContainerRemove(removeCtx, containerID, container.RemoveOptions{
        Force:         true,
        RemoveVolumes: true,
    })

    // Return to default branch so the clone is in a clean state for the next run.
    // No branch cleanup needed: we use FETCH_HEAD (detached HEAD), so no local
    // branches accumulate.
    exec.Command("git", "-C", job.LocalPath, "checkout", job.DefaultBranch).Run()

    // Remove the job from the processing list and release the per-repo active lock.
    _ = wp.queue.Ack(removeCtx, workerID, job)

    // Expire the real-time log key.
    _ = e.redis.Expire(removeCtx, fmt.Sprintf("verdox:logs:%s", job.TestRunID), time.Hour)
}()
```

**Post-run reset:** After every test run, the worker returns the local
clone to `git checkout {default_branch}`. Since workspace preparation
checks out `FETCH_HEAD` (detached HEAD) rather than creating a local
branch, there is no branch accumulation and no `git branch -D` cleanup
is needed. The per-repo sequential queue guarantees no other run is
touching this clone concurrently.

---

## Commit-Hash Result Caching

Before queuing a new test run, the API checks if results already exist for
the same suite + branch + commit_hash:

1. **Query:**
   ```sql
   SELECT id, status FROM test_runs
   WHERE test_suite_id = $1 AND branch = $2 AND commit_hash = $3
     AND status IN ('passed', 'failed')
   ORDER BY run_number DESC LIMIT 1
   ```

2. **If a completed run exists for this exact commit:**
   - Return the cached results to the user (no new run queued)
   - Response includes a flag: `"cached": true`

3. **If the user explicitly requests a re-run (`force=true` parameter):**
   - Queue a new run regardless of cache
   - Increment `run_number`: `max(run_number) + 1` for this suite+branch

4. **Run numbering:** Each run for a given suite+branch gets an
   incrementing `run_number` (run-1, run-2, etc.)

5. **Cache invalidation:** When a new commit is pushed to a branch, the
   next run on that branch will have a different `commit_hash`, so it
   naturally bypasses the cache.

---

## 5. Test Output Parsing

The parser (`internal/runner/parser.go`) converts raw test output into
structured `TestResult` records. It selects a parsing strategy based on the
test type and detected output format.

### Go Test (JSON Output)

When the test command includes `-json` (default for Go), `go test` emits
NDJSON -- one JSON object per line:

```json
{"Time":"2025-01-20T14:00:05Z","Action":"run","Test":"TestUserLogin","Package":"./internal/handler"}
{"Time":"2025-01-20T14:00:05Z","Action":"output","Test":"TestUserLogin","Output":"=== RUN   TestUserLogin\n"}
{"Time":"2025-01-20T14:00:05Z","Action":"pass","Test":"TestUserLogin","Elapsed":0.12}
{"Time":"2025-01-20T14:00:06Z","Action":"run","Test":"TestUserSignup","Package":"./internal/handler"}
{"Time":"2025-01-20T14:00:06Z","Action":"fail","Test":"TestUserSignup","Elapsed":0.34}
{"Time":"2025-01-20T14:00:06Z","Action":"skip","Test":"TestSlowIntegration","Elapsed":0.00}
```

**Parsing logic:**

```go
func (p *Parser) ParseGoJSON(output string) ([]TestResult, error) {
    var results []TestResult
    scanner := bufio.NewScanner(strings.NewReader(output))

    // Accumulate output lines per test.
    testOutputs := make(map[string]*strings.Builder)

    for scanner.Scan() {
        var event GoTestEvent
        if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
            continue // skip non-JSON lines (e.g., build output)
        }

        if event.Test == "" {
            continue // package-level event, skip
        }

        switch event.Action {
        case "output":
            if _, ok := testOutputs[event.Test]; !ok {
                testOutputs[event.Test] = &strings.Builder{}
            }
            testOutputs[event.Test].WriteString(event.Output)

        case "pass", "fail", "skip":
            result := TestResult{
                TestName:   event.Test,
                Status:     mapGoAction(event.Action),
                DurationMs: int64(event.Elapsed * 1000),
            }
            if out, ok := testOutputs[event.Test]; ok {
                result.LogOutput = out.String()
            }
            if event.Action == "fail" {
                result.ErrorMessage = extractFailureMessage(result.LogOutput)
            }
            results = append(results, result)
        }
    }

    return results, nil
}
```

**Action mapping:**

| `go test -json` Action | `test_result_status` |
|------------------------|---------------------|
| `"pass"` | `pass` |
| `"fail"` | `fail` |
| `"skip"` | `skip` |

### Python / pytest

pytest does not emit structured JSON by default. The parser scans stdout
for the status line pattern:

```
test_auth.py::test_login PASSED
test_auth.py::test_signup FAILED
test_auth.py::test_admin SKIPPED
test_auth.py::test_edge_case ERROR
```

**Parsing logic:**

```go
var pytestPattern = regexp.MustCompile(
    `^(.+?::[\w\[\]]+)\s+(PASSED|FAILED|SKIPPED|ERROR)`,
)

func (p *Parser) ParsePytest(output string) ([]TestResult, error) {
    var results []TestResult
    scanner := bufio.NewScanner(strings.NewReader(output))

    for scanner.Scan() {
        line := scanner.Text()
        matches := pytestPattern.FindStringSubmatch(line)
        if matches == nil {
            continue
        }

        result := TestResult{
            TestName: matches[1],
            Status:   mapPytestStatus(matches[2]),
        }
        results = append(results, result)
    }

    return results, nil
}
```

**Status mapping:**

| pytest Output | `test_result_status` |
|---------------|---------------------|
| `PASSED` | `pass` |
| `FAILED` | `fail` |
| `SKIPPED` | `skip` |
| `ERROR` | `error` |

### Node.js / Jest

Jest output uses Unicode indicators. The parser matches these patterns:

```
  ✓ should authenticate user (5ms)
  ✗ should reject invalid token (12ms)
  ○ skipped: should handle edge case
```

**Parsing logic:**

```go
var jestPassPattern = regexp.MustCompile(`^\s+[✓✔]\s+(.+?)(?:\s+\((\d+)\s*ms\))?\s*$`)
var jestFailPattern = regexp.MustCompile(`^\s+[✗✘×]\s+(.+?)(?:\s+\((\d+)\s*ms\))?\s*$`)
var jestSkipPattern = regexp.MustCompile(`^\s+[○◌]\s+(?:skipped:?\s*)?(.+?)\s*$`)

func (p *Parser) ParseJest(output string) ([]TestResult, error) {
    var results []TestResult
    scanner := bufio.NewScanner(strings.NewReader(output))

    for scanner.Scan() {
        line := scanner.Text()

        if m := jestPassPattern.FindStringSubmatch(line); m != nil {
            results = append(results, TestResult{
                TestName:   strings.TrimSpace(m[1]),
                Status:     "pass",
                DurationMs: parseDuration(m[2]),
            })
        } else if m := jestFailPattern.FindStringSubmatch(line); m != nil {
            results = append(results, TestResult{
                TestName:   strings.TrimSpace(m[1]),
                Status:     "fail",
                DurationMs: parseDuration(m[2]),
            })
        } else if m := jestSkipPattern.FindStringSubmatch(line); m != nil {
            results = append(results, TestResult{
                TestName:   strings.TrimSpace(m[1]),
                Status:     "skip",
            })
        }
    }

    return results, nil
}
```

### Fallback Parser

If the output does not match any known format, or if parsing produces zero
results, the parser falls back to creating a single `test_result` row that
captures the entire output:

```go
func (p *Parser) Fallback(output string, exitCode int64) []TestResult {
    status := "pass"
    if exitCode != 0 {
        status = "fail"
    }

    return []TestResult{{
        TestName:   "full_run",
        Status:     status,
        LogOutput:  output,
    }}
}
```

This ensures every test run produces at least one `test_results` row,
regardless of whether the output was parseable.

---

## 6. Configuration File (verdox.yaml)

Repositories may include a `verdox.yaml` file at the path specified by
`test_suites.config_path` (defaults to repo root). This file allows teams
to customize the test execution environment without modifying the suite
configuration through the Verdox API.

### Schema

```yaml
version: 1
full_clone: false   # optional: fetch full git history (default: false)

suites:
  - name: "Unit Tests"
    type: unit
    image: golang:1.25-alpine
    command: "go test -v -json ./..."
    timeout: 300
    env:
      CGO_ENABLED: "0"

  - name: "Integration Tests"
    type: integration
    image: docker-compose
    command: "docker-compose -f docker-compose.test.yml up --abort-on-container-exit"
    timeout: 600
    services:
      - postgres:17
      - redis:7
    env:
      DATABASE_URL: "postgres://test:test@localhost/test"
```

### Field Reference

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `version` | int | yes | -- | Schema version (currently `1`) |
| `full_clone` | bool | no | `false` | When `true`, clone and fetch operations use full history (no `--depth 1`). Enable this if tests depend on `git log`, `git blame`, `git diff` against history, or changelog generation. See GITHUB-INTEGRATION.md Section 4 for details on shallow clone limitations. |
| `suites` | array | yes | -- | List of suite definitions |
| `suites[].name` | string | yes | -- | Must match the `test_suites.name` in the database |
| `suites[].type` | enum | no | `unit` | `unit` or `integration` |
| `suites[].image` | string | no | auto-detected | Docker image to run the tests in |
| `suites[].command` | string | no | auto-detected | Shell command to execute |
| `suites[].timeout` | int | no | 300 | Timeout in seconds (overrides `test_suites.timeout_seconds`) |
| `suites[].env` | map | no | `{}` | Additional environment variables injected into the container |
| `suites[].services` | array | no | `[]` | Sidecar service images started alongside the test container (integration tests only) |

### Config Loading

The worker reads `verdox.yaml` from the local clone after the workspace
preparation step completes. If the file does not exist or fails to parse, the worker
falls back to defaults without failing the run.

```go
func (e *Executor) loadConfig(workDir string, configPath string) (*VerdoxConfig, error) {
    path := filepath.Join(workDir, configPath)
    data, err := os.ReadFile(path)
    if os.IsNotExist(err) {
        return nil, nil // no config file, use defaults
    }
    if err != nil {
        return nil, fmt.Errorf("read config: %w", err)
    }

    var cfg VerdoxConfig
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        log.Warn().Err(err).Msg("failed to parse verdox.yaml, using defaults")
        return nil, nil
    }
    return &cfg, nil
}
```

---

## 7. Timeout Handling

### Timeout Hierarchy

Timeouts are applied in a specific order of precedence:

```
Global maximum (RUNNER_MAX_TIMEOUT, default 1800s)
  └── Per-suite timeout (test_suites.timeout_seconds, default 300s)
       └── Per-run config override (verdox.yaml suites[].timeout)
```

The effective timeout is always clamped to the global maximum:

```go
func effectiveTimeout(suiteCfg int, yamlCfg int, globalMax int) time.Duration {
    timeout := suiteCfg // from test_suites.timeout_seconds

    if yamlCfg > 0 {
        timeout = yamlCfg // verdox.yaml override
    }

    if timeout > globalMax {
        timeout = globalMax // clamp to global max
    }

    return time.Duration(timeout) * time.Second
}
```

### Timeout Enforcement

The timeout is enforced via `context.WithTimeout` wrapping the Docker
container wait operation:

```go
func (wp *WorkerPool) executeJob(ctx context.Context, workerID string, job *model.JobPayload) {
    timeout := effectiveTimeout(job.TimeoutSeconds, yamlTimeout, wp.cfg.MaxTimeout)

    execCtx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    // ... container create, start ...

    exitCode, err := wp.executor.WaitWithTimeout(execCtx, containerID, timeout)
    if err != nil && strings.Contains(err.Error(), "timeout exceeded") {
        // Kill the container and mark the run as failed.
        wp.db.ExecContext(context.Background(),
            `UPDATE test_runs SET status = 'failed', finished_at = now() WHERE id = $1`,
            job.TestRunID,
        )
        wp.db.ExecContext(context.Background(),
            `INSERT INTO test_results (id, test_run_id, test_name, status, error_message, created_at)
             VALUES ($1, $2, 'full_run', 'error', $3, now())`,
            uuid.New(), job.TestRunID,
            fmt.Sprintf("Test run exceeded timeout of %d seconds", int(timeout.Seconds())),
        )
        return
    }

    // ... normal result processing ...
}
```

---

## 8. Cancellation

### Cancel Flow

```
  Client                     Backend API              Redis                   Worker
    │                            │                      │                       │
    │ POST /api/v1/runs/:id/     │                      │                       │
    │ cancel                     │                      │                       │
    │───────────────────────────>│                      │                       │
    │                            │                      │                       │
    │                            │ SELECT status        │                       │
    │                            │ FROM test_runs       │                       │
    │                            │ WHERE id = :id       │                       │
    │                            │                      │                       │
    │                            │ status == 'queued'?  │                       │
    │                            │──── YES ────────────>│ LREM pending queue    │
    │                            │                      │                       │
    │                            │ UPDATE status =      │                       │
    │                            │ 'cancelled'          │                       │
    │                            │                      │                       │
    │                            │ status == 'running'? │                       │
    │                            │──── YES ────────────>│ PUBLISH               │
    │                            │                      │ verdox:cancel:{id}    │
    │                            │                      │──────────────────────>│
    │                            │                      │                       │
    │                            │                      │                       │ Kill container
    │                            │                      │                       │ (SIGKILL)
    │                            │                      │                       │
    │                            │                      │                       │ UPDATE status =
    │                            │                      │                       │ 'cancelled'
    │                            │                      │                       │
    │ 200 OK                     │                      │                       │
    │ {"status":"cancelled"}     │                      │                       │
    │<───────────────────────────│                      │                       │
```

### API Handler

```go
// internal/handler/test_run.go

func (h *TestRunHandler) Cancel(c echo.Context) error {
    runID := c.Param("id")

    var run model.TestRun
    err := h.db.GetContext(c.Request().Context(), &run,
        `SELECT id, status FROM test_runs WHERE id = $1`, runID)
    if err != nil {
        return response.NotFound(c, "Test run not found")
    }

    switch run.Status {
    case "queued":
        // Remove from the repo's pending queue.
        h.queue.RemoveByRunID(c.Request().Context(), runID)

        // Update status directly.
        h.db.ExecContext(c.Request().Context(),
            `UPDATE test_runs SET status = 'cancelled', finished_at = now() WHERE id = $1`, runID)

    case "running":
        // Signal the worker to kill the container.
        h.redis.Publish(c.Request().Context(),
            fmt.Sprintf("verdox:cancel:%s", runID), "cancel")

    default:
        return response.Conflict(c, "Run is already in a terminal state")
    }

    return response.JSON(c, http.StatusOK, map[string]string{
        "id":      runID,
        "status":  "cancelled",
        "message": "Test run has been cancelled.",
    })
}
```

### Worker-Side Cancel Listener

Each worker subscribes to the cancel channel for the run it is currently
executing:

```go
func (wp *WorkerPool) listenForCancel(ctx context.Context, runID string, containerID string) {
    sub := wp.redis.Subscribe(ctx, fmt.Sprintf("verdox:cancel:%s", runID))
    defer sub.Close()

    ch := sub.Channel()
    select {
    case <-ch:
        log.Info().Str("run_id", runID).Msg("received cancel signal")
        _ = wp.executor.docker.ContainerKill(context.Background(), containerID, "SIGKILL")
    case <-ctx.Done():
        // Context cancelled (job finished naturally or worker shutting down).
    }
}
```

This goroutine is launched at the start of `executeJob()` and runs
concurrently with the container wait. When the container is killed, the
wait returns with a non-zero exit code, and the worker detects the
cancellation.

---

## 9. Log Streaming

### Write Path (During Execution)

As the worker reads stdout/stderr from the container, each line is:

1. **Appended** to the Redis key `verdox:logs:{run_id}` (string append
   operation). This key has a TTL of 1 hour.
2. **Published** to the Redis pub/sub channel `verdox:logs:{run_id}:stream`
   for real-time subscribers.

```go
// Called from executor.StreamLogs() for each line of output.
func (e *Executor) appendLogLine(ctx context.Context, runID string, line string) {
    key := fmt.Sprintf("verdox:logs:%s", runID)
    e.redis.Append(ctx, key, line+"\n")
    e.redis.Expire(ctx, key, time.Hour)
    e.redis.Publish(ctx, fmt.Sprintf("verdox:logs:%s:stream", runID), line)
}
```

### Read Path -- Full Log Fetch (Completed Runs)

`GET /api/v1/runs/:id/logs` returns logs from the `test_results` table
after the run has finished. This is the permanent storage location:

```sql
SELECT test_name, status, duration_ms, log_output
FROM test_results
WHERE test_run_id = $1
ORDER BY test_name;
```

### Read Path -- Streaming (In-Progress Runs)

For runs with `status = 'running'`, the endpoint serves an SSE (Server-Sent
Events) stream backed by Redis pub/sub:

```go
// internal/handler/test_run.go

func (h *TestRunHandler) StreamLogs(c echo.Context) error {
    runID := c.Param("id")

    c.Response().Header().Set("Content-Type", "text/event-stream")
    c.Response().Header().Set("Cache-Control", "no-cache")
    c.Response().Header().Set("Connection", "keep-alive")

    // First, send any buffered log lines from Redis.
    buffered, _ := h.redis.Get(c.Request().Context(), fmt.Sprintf("verdox:logs:%s", runID)).Result()
    if buffered != "" {
        fmt.Fprintf(c.Response(), "data: %s\n\n", buffered)
        c.Response().Flush()
    }

    // Then subscribe to the real-time stream.
    sub := h.redis.Subscribe(c.Request().Context(), fmt.Sprintf("verdox:logs:%s:stream", runID))
    defer sub.Close()

    ch := sub.Channel()
    for {
        select {
        case msg := <-ch:
            fmt.Fprintf(c.Response(), "data: %s\n\n", msg.Payload)
            c.Response().Flush()
        case <-c.Request().Context().Done():
            return nil
        }
    }
}
```

### Log Lifecycle

| Phase | Storage | Access Method |
|-------|---------|---------------|
| Running | Redis `verdox:logs:{run_id}` (buffer) + pub/sub (stream) | SSE streaming endpoint |
| Completed (< 1 hour) | Redis buffer + `test_results.log_output` | Full fetch from DB; Redis as fallback |
| Completed (> 1 hour) | `test_results.log_output` only | Full fetch from DB |

---

## 10. Error Handling

### Error Classification and Recovery

| Error Scenario | Detection | Action | `test_runs.status` | `error_message` |
|----------------|-----------|--------|-------------------|-----------------|
| Docker daemon unavailable | Docker API connection refused | Skip execution, fail immediately | `failed` | `"Docker engine unavailable"` |
| Image pull failure | Docker API error on `ImagePull` | Retry once after 5s; if still fails, abort | `failed` | `"Failed to pull image {image}: {error}"` |
| Container OOM kill | Exit code 137 | Fail with specific message | `failed` | `"Out of memory: container killed (exit code 137)"` |
| Container timeout | `context.DeadlineExceeded` | Kill container, fail | `failed` | `"Test run exceeded timeout of {N} seconds"` |
| Non-zero exit code | Exit code != 0, not 137 | Parse output, report individual failures | `failed` | `"Container exited with code {X}"` (only if no parseable results) |
| Disk full | Docker API error or container stderr | Fail gracefully | `failed` | `"Disk quota exceeded"` |
| Workspace preparation failure | `git fetch` or `git checkout` fails, or local clone missing | Fail immediately | `failed` | `"Failed to prepare workspace: {error}"` |
| PAT authentication failure | `git fetch` exit code 128 with "Authentication failed" or "could not read Username" | Fail immediately, do not retry | `failed` | `"GitHub authentication failed. The team's PAT may be expired or revoked. A team admin should update the PAT in team settings."` |
| PAT missing | Worker receives job but team has no PAT configured | Fail immediately | `failed` | `"No GitHub PAT configured for this team"` |
| Config parse failure | YAML parse error | Log warning, continue with defaults | -- (not a failure) | -- |
| Worker crash | Process restart | Scan processing lists for stale jobs | see below | see below |

### PAT Authentication Failure During Git Operations

If `git fetch` returns **exit code 128** with an error message containing
`"Authentication failed"` or `"could not read Username"`, the worker
handles this as a non-retryable infrastructure error:

1. **Mark the test run as `failed`.**
2. **Set `error_message`** to: `"GitHub authentication failed. The team's
   PAT may be expired or revoked. A team admin should update the PAT in
   team settings."`
3. **Release the per-repo active lock** (`verdox:jobs:active:{repo_id}`)
   so subsequent runs for the same repo are not blocked.
4. **Log at WARN level** with `team_id` and `repo_id` for operational
   visibility.
5. **Do NOT retry.** PAT expiry or revocation requires human intervention
   (a team admin must rotate the PAT in team settings). Retrying would
   produce the same failure and waste queue capacity.

```go
// Inside executeJob(), after git fetch:
if exitCode == 128 && (strings.Contains(stderr, "Authentication failed") ||
    strings.Contains(stderr, "could not read Username")) {
    wp.db.ExecContext(ctx,
        `UPDATE test_runs SET status = 'failed', finished_at = now(),
         error_message = 'GitHub authentication failed. The team''s PAT may be expired or revoked. A team admin should update the PAT in team settings.'
         WHERE id = $1`, job.TestRunID)
    log.Warn().
        Str("run_id", job.TestRunID).
        Str("repo_id", job.RepoID).
        Str("team_id", teamID).
        Msg("git fetch failed: PAT authentication error")
    // Release lock and return -- do not retry.
    wp.queue.Ack(ctx, workerID, job)
    return
}
```

**Missing PAT (pre-queue guard):** The API handler checks for a configured
PAT before enqueuing a job. If the team has no PAT, the API returns
`422 UNPROCESSABLE` with `"No GitHub PAT configured for this team"`. If
this check is somehow bypassed and the job reaches the worker, the worker
fails immediately with the same message and does not retry.

### Crash Recovery (On Startup)

On backend startup, the worker pool performs two recovery passes:

**Pass 1: Redis processing list scan.** Scan all
`verdox:jobs:processing:*` keys for jobs that were in-flight when the
previous process died. For each stale job:

1. Clean up the per-repo active lock (`verdox:jobs:active:{repo_id}`).
2. Check the run's status in PostgreSQL.
3. If `status = 'running'` and `started_at` is older than
   `RUNNER_MAX_TIMEOUT`: mark as `failed` with
   `error_message = "Worker crashed or timed out during execution"`.
4. If `status = 'running'` and within the timeout window: re-queue the
   job for another attempt (set `status = 'queued'`, clear `started_at`).
5. Clear the processing list for the dead worker.

**Pass 2: Database scan for orphaned runs.** Scan the `test_runs` table
directly for any rows with `status = 'running'` and `started_at` older
than `RUNNER_MAX_TIMEOUT`. This catches runs that may have been missed by
the Redis scan (e.g., if the Redis processing list was lost during a Redis
restart). Mark them as `failed` with
`error_message = "Worker crashed or timed out during execution"` and clean
up their active locks.

```go
func (wp *WorkerPool) RecoverStalledJobs(ctx context.Context) error {
    // --- Pass 1: Redis processing list scan ---
    keys, err := wp.redis.Keys(ctx, "verdox:jobs:processing:*").Result()
    if err != nil {
        return err
    }

    for _, key := range keys {
        jobs, err := wp.redis.LRange(ctx, key, 0, -1).Result()
        if err != nil {
            continue
        }

        for _, raw := range jobs {
            var job model.JobPayload
            if err := json.Unmarshal([]byte(raw), &job); err != nil {
                continue
            }

            // Clean up the per-repo active lock left behind by the crash.
            wp.redis.Del(ctx, fmt.Sprintf("verdox:jobs:active:%s", job.RepoID))

            // Check if the run is still marked as 'running' in the DB.
            var run struct {
                Status    string    `db:"status"`
                StartedAt time.Time `db:"started_at"`
            }
            err := wp.db.GetContext(ctx, &run,
                `SELECT status, started_at FROM test_runs WHERE id = $1`, job.TestRunID)
            if err != nil {
                continue
            }

            if run.Status == "running" {
                elapsed := time.Since(run.StartedAt)

                if elapsed > time.Duration(wp.cfg.MaxTimeout)*time.Second {
                    // Exceeded max timeout -- mark as failed.
                    wp.db.ExecContext(ctx,
                        `UPDATE test_runs SET status = 'failed', finished_at = now(),
                         error_message = 'Worker crashed or timed out during execution'
                         WHERE id = $1`,
                        job.TestRunID)
                    log.Warn().Str("run_id", job.TestRunID).Msg("stale job marked as failed (timeout)")
                } else {
                    // Re-queue the job for another attempt.
                    wp.queue.Push(ctx, &job)
                    wp.db.ExecContext(ctx,
                        `UPDATE test_runs SET status = 'queued', started_at = NULL WHERE id = $1`,
                        job.TestRunID)
                    log.Info().Str("run_id", job.TestRunID).Msg("stale job re-queued")
                }
            }
        }

        // Clear the processing list for this worker.
        wp.redis.Del(ctx, key)
    }

    // --- Pass 2: Database scan for orphaned runs ---
    maxTimeout := time.Duration(wp.cfg.MaxTimeout) * time.Second
    cutoff := time.Now().UTC().Add(-maxTimeout)

    var orphanedRuns []struct {
        ID     string `db:"id"`
        RepoID string `db:"repo_id"`
    }
    wp.db.SelectContext(ctx, &orphanedRuns,
        `SELECT tr.id, ts.repo_id
         FROM test_runs tr
         JOIN test_suites ts ON tr.test_suite_id = ts.id
         WHERE tr.status = 'running' AND tr.started_at < $1`, cutoff)

    for _, run := range orphanedRuns {
        wp.db.ExecContext(ctx,
            `UPDATE test_runs SET status = 'failed', finished_at = now(),
             error_message = 'Worker crashed or timed out during execution'
             WHERE id = $1`,
            run.ID)
        wp.redis.Del(ctx, fmt.Sprintf("verdox:jobs:active:%s", run.RepoID))
        log.Warn().Str("run_id", run.ID).Msg("orphaned run marked as failed (DB scan)")
    }

    return nil
}
```

### Periodic Stale Run Health Check

In addition to startup recovery, a background goroutine runs every
**5 minutes** to catch stale runs that may have been missed -- for
example, if a worker goroutine deadlocks or the container wait hangs
without triggering the timeout. This provides defense-in-depth beyond
the startup-only scan.

```go
func (wp *WorkerPool) StartStaleRunChecker(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            wp.checkStaleRuns(ctx)
        }
    }
}

func (wp *WorkerPool) checkStaleRuns(ctx context.Context) {
    maxTimeout := time.Duration(wp.cfg.MaxTimeout) * time.Second
    cutoff := time.Now().UTC().Add(-maxTimeout)

    var staleRuns []struct {
        ID     string `db:"id"`
        RepoID string `db:"repo_id"`
    }
    err := wp.db.SelectContext(ctx, &staleRuns,
        `SELECT tr.id, ts.repo_id
         FROM test_runs tr
         JOIN test_suites ts ON tr.test_suite_id = ts.id
         WHERE tr.status = 'running' AND tr.started_at < $1`, cutoff)
    if err != nil {
        log.Error().Err(err).Msg("stale run check: query failed")
        return
    }

    for _, run := range staleRuns {
        wp.db.ExecContext(ctx,
            `UPDATE test_runs SET status = 'failed', finished_at = now(),
             error_message = 'Worker crashed or timed out during execution'
             WHERE id = $1 AND status = 'running'`,
            run.ID)
        wp.redis.Del(ctx, fmt.Sprintf("verdox:jobs:active:%s", run.RepoID))
        log.Warn().
            Str("run_id", run.ID).
            Str("repo_id", run.RepoID).
            Msg("stale run detected and marked as failed (periodic check)")
    }

    if len(staleRuns) > 0 {
        log.Info().Int("count", len(staleRuns)).Msg("stale run check completed")
    }
}
```

The `StartStaleRunChecker` goroutine is launched alongside the worker pool
in `cmd/server/main.go`:

```go
go wp.StartStaleRunChecker(ctx)
```

**Why both startup and periodic checks?** Startup recovery handles the
common case (process crash/restart). The periodic check handles edge
cases where a worker goroutine is stuck but the process is still running
-- the TTL on `verdox:jobs:active:{repo_id}` will eventually expire, but
the DB row would remain in `'running'` state indefinitely without this
check.

---

## 11. Resource Management

### Container Cleanup

Every `executeJob()` call wraps container removal in a `defer` block.
This guarantees cleanup regardless of success, failure, panic, or timeout:

```go
defer func() {
    ctx := context.Background() // fresh context, not the cancelled one
    _ = e.docker.ContainerRemove(ctx, containerID, container.RemoveOptions{
        Force:         true,
        RemoveVolumes: true,
    })
}()
```

### Image Cache

On startup, the worker pool pre-pulls common base images to avoid pull
latency on the first run:

```go
func (wp *WorkerPool) PrePullImages(ctx context.Context) {
    images := []string{
        "golang:1.25-alpine",
        "python:3.12-slim",
        "node:22-alpine",
        "alpine:3.21",
    }

    for _, img := range images {
        log.Info().Str("image", img).Msg("pre-pulling image")
        reader, err := wp.executor.docker.ImagePull(ctx, img, image.PullOptions{})
        if err != nil {
            log.Warn().Err(err).Str("image", img).Msg("failed to pre-pull image")
            continue
        }
        io.Copy(io.Discard, reader)
        reader.Close()
    }
}
```

### Periodic Disk Cleanup

A background goroutine runs every 30 minutes to remove dangling resources:

```go
func (wp *WorkerPool) StartCleanupLoop(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            // Remove stopped containers with the "verdox-run-" prefix.
            containers, _ := wp.executor.docker.ContainerList(ctx, container.ListOptions{
                All:     true,
                Filters: filters.NewArgs(filters.Arg("name", "verdox-run-")),
            })
            for _, c := range containers {
                if c.State == "exited" || c.State == "dead" {
                    wp.executor.docker.ContainerRemove(ctx, c.ID, container.RemoveOptions{
                        Force:         true,
                        RemoveVolumes: true,
                    })
                }
            }

            // Prune dangling images.
            wp.executor.docker.ImagesPrune(ctx, filters.NewArgs(filters.Arg("dangling", "true")))
        }
    }
}
```

### Queue Depth Monitoring

The admin stats endpoint exposes current queue depth for operational
visibility:

```go
// Exposed via GET /api/v1/admin/stats
func (h *AdminHandler) Stats(c echo.Context) error {
    // Sum pending jobs across all per-repo queues.
    var pendingCount int64
    repoKeys, _ := h.redis.Keys(c.Request().Context(), "verdox:jobs:repo:*").Result()
    for _, key := range repoKeys {
        count, _ := h.redis.LLen(c.Request().Context(), key).Result()
        pendingCount += count
    }

    // Count currently running jobs across all workers.
    runningCount := 0
    keys, _ := h.redis.Keys(c.Request().Context(), "verdox:jobs:processing:*").Result()
    for _, key := range keys {
        count, _ := h.redis.LLen(c.Request().Context(), key).Result()
        runningCount += int(count)
    }

    return response.JSON(c, http.StatusOK, map[string]interface{}{
        "queue_pending":  pendingCount,
        "queue_running":  runningCount,
        "worker_pool":    h.cfg.MaxConcurrent,
    })
}
```

---

## 12. Environment Variables

All runner-related configuration is read from environment variables by
`internal/config/config.go`:

```
RUNNER_MAX_CONCURRENT=5               # Max concurrent test runs (worker pool size)
RUNNER_MAX_TIMEOUT=1800               # Global max timeout in seconds (30 minutes)
RUNNER_DOCKER_HOST=unix:///var/run/docker.sock  # Docker daemon socket path
RUNNER_NETWORK=verdox-runner-net      # Docker network for integration test containers
RUNNER_DEFAULT_IMAGE=alpine:3.21      # Fallback image when language/config not detected
VERDOX_REPO_BASE_PATH=./data/repositories  # Base path for local repository clones
```

### Config Struct

```go
// internal/config/config.go

type RunnerConfig struct {
    MaxConcurrent int    `env:"RUNNER_MAX_CONCURRENT" envDefault:"5"`
    MaxTimeout    int    `env:"RUNNER_MAX_TIMEOUT"    envDefault:"1800"`
    DockerHost    string `env:"RUNNER_DOCKER_HOST"    envDefault:"unix:///var/run/docker.sock"`
    Network       string `env:"RUNNER_NETWORK"        envDefault:"verdox-runner-net"`
    DefaultImage  string `env:"RUNNER_DEFAULT_IMAGE"  envDefault:"alpine:3.21"`
    RepoBasePath  string `env:"VERDOX_REPO_BASE_PATH" envDefault:"./data/repositories"`
}
```

### Variable Reference

| Variable | Type | Default | Constraints | Description |
|----------|------|---------|-------------|-------------|
| `RUNNER_MAX_CONCURRENT` | int | `5` | min: 1, max: 50 | Number of worker goroutines. Each goroutine handles one test run at a time. Runs across different repos execute in parallel up to this limit; same-repo runs are always sequential. Higher values require more CPU and memory on the host. |
| `RUNNER_MAX_TIMEOUT` | int | `1800` | min: 60, max: 3600 | Absolute upper bound for any test run, in seconds. Overrides per-suite and per-config timeouts if they exceed this value. |
| `RUNNER_DOCKER_HOST` | string | `unix:///var/run/docker.sock` | valid URI | Docker daemon endpoint. Use `tcp://host:2376` for remote Docker with TLS. |
| `RUNNER_NETWORK` | string | `verdox-runner-net` | valid Docker network name | Docker network attached to integration test containers. Unit test containers use `none` (no network). |
| `RUNNER_DEFAULT_IMAGE` | string | `alpine:3.21` | valid Docker image reference | Used when no language is detected and no image is specified in `verdox.yaml`. |
| `VERDOX_REPO_BASE_PATH` | string | `./data/repositories` | valid directory path | Base path where local repository clones are stored. Each repo is stored at `{base_path}/github.com/{owner}/{repo}`. |
