# Monitoring, Logging, and Observability

This document describes the monitoring, logging, and observability strategy for Verdox. It covers structured logging, health checks, application metrics, Docker logging, alerting, and optional log aggregation.

**Stack**: Go 1.25+ with Echo v4, zerolog for structured JSON logging. All services (nginx, frontend, backend, postgres, redis, runner) run in Docker Compose.

---

## 1. Structured Logging

### Format

All backend logs are emitted in JSON format via [zerolog](https://github.com/rs/zerolog). Every HTTP request produces a structured log entry with the following shape:

```json
{
  "level": "info",
  "time": "2024-01-15T10:30:00Z",
  "request_id": "req-uuid",
  "method": "POST",
  "path": "/api/v1/auth/login",
  "status": 200,
  "duration_ms": 45,
  "user_id": "user-uuid",
  "ip": "192.168.1.1",
  "user_agent": "Mozilla/5.0...",
  "message": "request completed"
}
```

The `request_id` field is generated per request by middleware and propagated through all downstream log calls, enabling full request tracing across log entries.

### Log Levels

| Level | Usage | Examples |
|-------|-------|---------|
| `debug` | Development only, verbose detail | SQL queries, cache hits/misses |
| `info` | Normal operations | Request completed, user logged in, test run started |
| `warn` | Recoverable issues | Rate limit approaching, GitHub API quota low, retry attempt |
| `error` | Failures requiring attention | DB connection failed, Docker API error, auth failure |
| `fatal` | Unrecoverable, process exits | Config missing, port in use |

### Per-Domain Log Fields

Each domain adds specific fields to log entries for structured querying and filtering.

**Auth**
- `user_id` -- the user performing the action
- `action` -- one of `login`, `signup`, `logout`, `refresh`
- `success` -- boolean indicating outcome

**Repository**
- `repo_id` -- the repository being operated on
- `action` -- one of `sync`, `delete`
- `repo_count` -- number of repos affected (for batch operations)

**Test Runner**
- `run_id` -- unique identifier for the test run
- `suite_id` -- the test suite being executed
- `status` -- run status (queued, running, passed, failed, cancelled)
- `duration_ms` -- total execution time
- `test_count`, `pass_count`, `fail_count` -- result breakdown

**Team**
- `team_id` -- the team being modified
- `action` -- the team operation performed
- `member_id` -- the user being added/removed/updated
- `role` -- the role assigned or changed

**Admin**
- `actor_id` -- the admin performing the action
- `target_id` -- the entity being acted upon
- `action` -- the administrative action
- `old_value`, `new_value` -- before/after values for auditable changes

### Log Configuration

```go
// Environment-based log level
zerolog.SetGlobalLevel(zerolog.InfoLevel)  // production
zerolog.SetGlobalLevel(zerolog.DebugLevel) // development
```

| Environment | Level | Output | Format |
|-------------|-------|--------|--------|
| Production | `info` | stdout | JSON |
| Development | `debug` | stdout | Console with colors (`zerolog.ConsoleWriter`) |

The log level is configurable at startup via the `LOG_LEVEL` environment variable. The output format is controlled by `LOG_FORMAT`.

---

## 2. Health Check Endpoints

### Liveness Check

```
GET /api/v1/health
```

Returns `200 OK` if the server process is running. This endpoint performs no dependency checks and is suitable for container liveness probes.

**Response (200)**:
```json
{
  "status": "ok",
  "version": "1.0.0",
  "uptime_seconds": 3600
}
```

### Readiness Check

```
GET /api/v1/health/ready
```

Returns `200 OK` only when ALL backend dependencies are reachable. This endpoint is suitable for container readiness probes and load balancer health checks.

**Response (200) -- all dependencies healthy**:
```json
{
  "status": "ok",
  "checks": {
    "database": { "status": "ok", "latency_ms": 2 },
    "redis": { "status": "ok", "latency_ms": 1 },
    "docker": { "status": "ok", "latency_ms": 5 },
    "repo_disk": {
      "status": "ok",
      "usage_gb": 12.4,
      "max_gb": 50,
      "usage_percent": 24.8
    }
  }
}
```

**Response (503) -- one or more dependencies degraded**:
```json
{
  "status": "degraded",
  "checks": {
    "database": { "status": "ok", "latency_ms": 2 },
    "redis": { "status": "error", "message": "connection refused" },
    "docker": { "status": "ok", "latency_ms": 5 },
    "repo_disk": {
      "status": "ok",
      "usage_gb": 12.4,
      "max_gb": 50,
      "usage_percent": 24.8
    }
  }
}
```

**Implementation details**:
- PostgreSQL: executes a `SELECT 1` ping query
- Redis: sends a `PING` command
- Docker: calls the Docker client `Ping()` method
- Repo disk: reads disk usage for `VERDOX_REPO_BASE_PATH` and compares against `VERDOX_REPO_MAX_DISK_GB`
- Each check has a **5-second timeout**; a timeout is reported as an error

### Docker Compose Healthchecks

Each service in `docker-compose.yml` defines a healthcheck so that Docker can track service readiness and enforce dependency ordering via `depends_on` conditions.

| Service | Command | Interval | Timeout | Retries |
|---------|---------|----------|---------|---------|
| `postgres` | `pg_isready -U verdox` | 10s | 5s | 5 |
| `redis` | `redis-cli ping` | 10s | 5s | 5 |
| `backend` | `wget --spider http://localhost:8080/api/v1/health` | 15s | 10s | 3 |
| `frontend` | `wget --spider http://localhost:3000` | 15s | 10s | 3 |
| `nginx` | Process check | 30s | -- | -- |

---

## 3. Application Metrics

Key metrics are tracked via the admin stats endpoint and structured log entries. These metrics can be scraped by external monitoring tools or computed from log aggregation.

### Admin Stats Endpoint

```
GET /api/v1/admin/stats
```

Returns high-level platform statistics:

```json
{
  "total_users": 150,
  "total_repos": 420,
  "total_test_runs": 8500,
  "active_runners": 3,
  "queue_depth": 7
}
```

### Request Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `request_total` | counter | `method`, `path`, `status_code` | Total HTTP requests served |
| `request_duration_ms` | histogram | `method`, `path` | Request latency distribution |
| `request_in_flight` | gauge | -- | Current number of concurrent requests |

### Auth Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `auth_login_total` | counter | `result` (success/failure) | Login attempts |
| `auth_signup_total` | counter | -- | New user registrations |
| `auth_lockout_total` | counter | -- | Account lockout events |

### Test Runner Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `test_runs_total` | counter | `status` (passed/failed/cancelled) | Completed test runs |
| `test_run_duration_seconds` | histogram | -- | End-to-end test run duration |
| `test_results_total` | counter | `status` (pass/fail/skip/error) | Individual test case results |
| `active_runners` | gauge | -- | Currently executing runner containers |
| `queue_depth` | gauge | -- | Test runs waiting in queue |
| `queue_wait_seconds` | histogram | -- | Time from queued to running |

### Repository Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `github_sync_total` | counter | `result` (success/failure) | GitHub sync operations |
| `github_api_calls_total` | counter | -- | Total GitHub API calls made |
| `github_rate_limit_remaining` | gauge | -- | Remaining GitHub API quota |
| `repo_disk_usage_bytes` | gauge | -- | Current disk usage of `VERDOX_REPO_BASE_PATH` in bytes |
| `repo_disk_max_bytes` | gauge | -- | Maximum allowed disk usage (`VERDOX_REPO_MAX_DISK_GB` converted to bytes) |
| `repo_disk_usage_percent` | gauge | -- | Current disk usage as a percentage of `VERDOX_REPO_MAX_DISK_GB` |

---

## 4. Docker Logging

### Container Log Configuration

All services use the `json-file` Docker logging driver with size rotation to prevent unbounded disk usage:

```yaml
# docker-compose.yml logging config per service
logging:
  driver: "json-file"
  options:
    max-size: "10m"
    max-file: "5"
```

Each container's logs are capped at **50 MB** (5 files x 10 MB). Older log files are rotated out automatically.

**Viewing logs**:
```bash
# Follow backend logs
docker compose logs -f backend

# View last 100 lines from all services
docker compose logs --tail=100

# View logs for a specific service since a timestamp
docker compose logs --since="2024-01-15T10:00:00" backend
```

### Nginx Access Log Format

Nginx is configured to emit access logs in JSON format, enabling consistent log parsing across all services:

```nginx
log_format json_combined escape=json
  '{'
    '"time_local":"$time_iso8601",'
    '"remote_addr":"$remote_addr",'
    '"request":"$request",'
    '"status": $status,'
    '"body_bytes_sent":$body_bytes_sent,'
    '"request_time":$request_time,'
    '"http_referrer":"$http_referer",'
    '"http_user_agent":"$http_user_agent"'
  '}';
```

This produces entries like:
```json
{
  "time_local": "2024-01-15T10:30:00+00:00",
  "remote_addr": "192.168.1.1",
  "request": "GET /api/v1/health HTTP/1.1",
  "status": 200,
  "body_bytes_sent": 42,
  "request_time": 0.002,
  "http_referrer": "",
  "http_user_agent": "Mozilla/5.0..."
}
```

---

## 5. Alerting Rules

The following alert conditions should be monitored. These can be implemented via log-based monitoring (e.g., Loki alerting rules), external tools (e.g., Grafana alerting), or simple scripts that poll health endpoints.

| Alert | Condition | Severity | Action |
|-------|-----------|----------|--------|
| Service Down | Health check fails 3 consecutive times | Critical | Restart service, notify admin |
| High Error Rate | >5% 5xx responses in 5 min window | High | Investigate backend logs |
| Queue Backup | `queue_depth` > 20 for 10 min | Medium | Scale runners or investigate stuck jobs |
| DB Connection Pool | >80% pool utilization | Medium | Monitor trend, may need pool size tuning |
| Disk Space | <20% free on data volume | High | Clean up old data, expand disk |
| Repo Disk Quota | `repo_disk_usage_percent` > 90% of `VERDOX_REPO_MAX_DISK_GB` | High | LRU eviction should activate automatically; investigate if repos are not being evicted |
| GitHub Rate Limit | <100 API calls remaining | Low | Reduce sync frequency temporarily |
| Failed Logins | >20 failures per IP in 5 min | Medium | Check for brute force attack, consider IP block |
| Test Runner OOM | Container exit code 137 | Low | Increase runner memory limit in Docker config |

---

## 6. Log Aggregation (Optional)

For production deployments beyond basic `docker compose logs`, two recommended approaches:

### Option A: Loki + Grafana (Lightweight)

Best for smaller deployments and teams already using Grafana.

- **Loki** collects logs directly from the Docker logging driver (via the Loki Docker plugin or Promtail)
- **Grafana** provides dashboards and log exploration via LogQL queries
- Low resource footprint, native Docker integration
- Example LogQL query: `{container="backend"} |= "error" | json | status >= 500`

### Option B: ELK Stack (Full-Featured)

Best for larger deployments needing full-text search and complex analytics.

- **Filebeat** ships container logs to Elasticsearch
- **Elasticsearch** indexes and stores logs
- **Kibana** provides visualization and search
- Higher resource requirements but more powerful querying and retention capabilities

Both options are **optional add-ons** and are not required for basic Verdox deployment. The default `json-file` Docker logging driver with `docker compose logs` is sufficient for development and small-scale production.

---

## 7. Grafana Dashboard Suggestions

If Grafana is deployed (standalone or as part of the Loki stack), the following dashboard panels are recommended:

| Panel | Type | Data Source | Description |
|-------|------|-------------|-------------|
| Request rate by endpoint | Time series | Logs/Metrics | Requests per second grouped by path |
| Response time p50/p95/p99 | Time series | Logs/Metrics | Latency percentiles over time |
| Error rate percentage | Stat panel | Logs/Metrics | Current 5xx rate as a single number |
| Active test runs | Gauge | Admin stats | Currently executing test runs |
| Queue depth over time | Time series | Admin stats / Logs | Pending test runs in queue |
| Test pass/fail ratio | Pie chart | Logs/Metrics | Breakdown of test outcomes |
| User signup/login activity | Bar chart | Logs | Auth events over time |
| GitHub API rate limit remaining | Gauge | Metrics | Current remaining API quota |

---

## 8. Environment Variables

Logging behavior is controlled by two environment variables:

| Variable | Default | Values | Description |
|----------|---------|--------|-------------|
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` | Minimum log level emitted |
| `LOG_FORMAT` | `json` | `json`, `console` | Output format (`console` enables colored human-readable output, intended for development only) |

Example configuration:

```bash
# Production
LOG_LEVEL=info
LOG_FORMAT=json

# Development
LOG_LEVEL=debug
LOG_FORMAT=console
```

These variables are read at backend startup and apply globally to all zerolog output.
