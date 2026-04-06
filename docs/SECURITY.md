# Verdox -- Security Design

> Self-hosted test orchestration platform.
> Go 1.25+ | Echo v4 | Next.js 15 | PostgreSQL 17 | Redis 7 | Docker-in-Docker

This document defines the security architecture, controls, and operational
procedures for the Verdox platform. Every implementation decision described here
is mandatory -- deviations require explicit justification and review.

---

## 1. Authentication Security

### 1.1 Password Storage

Passwords are hashed using bcrypt with a cost factor of 12 before storage.

```go
hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
```

At cost 12 each hash operation takes approximately 250ms on modern hardware.
This is intentionally slow to make brute-force attacks impractical while
remaining acceptable for interactive login latency. The `password_hash` column
is stored in the `users` table and is never included in any API response or log
output.

### 1.2 Password Requirements

All passwords must satisfy these rules at signup and password change:

| Rule            | Requirement                     |
|-----------------|---------------------------------|
| Minimum length  | 8 characters                    |
| Uppercase       | At least 1 uppercase letter     |
| Lowercase       | At least 1 lowercase letter     |
| Digit           | At least 1 numeric digit        |

Validation is enforced both on the backend (Go validator) and on the frontend
(Zod schema). The backend is the authoritative check -- frontend validation
exists only for user experience.

### 1.3 JWT Signing

| Property        | Value                                           |
|-----------------|-------------------------------------------------|
| Algorithm       | HS256 (HMAC-SHA256)                             |
| Signing key     | `JWT_SECRET` environment variable               |
| Minimum key len | 32 characters                                   |
| Library         | `golang-jwt/jwt/v5`                             |

The server refuses to start if `JWT_SECRET` is shorter than 32 characters. The
secret is never logged, never included in error messages, and never committed to
version control.

### 1.4 Access Token

| Property  | Value                                                      |
|-----------|------------------------------------------------------------|
| Format    | Signed JWT (HS256)                                         |
| Lifetime  | 15 minutes                                                 |
| Delivery  | `Authorization: Bearer <token>` header or `verdox_access` httpOnly cookie |
| Claims    | `sub` (user ID), `username`, `role`, `iat`, `exp`          |

The short lifetime limits the damage window if an access token is compromised.
Clients must use the refresh flow to obtain new access tokens.

### 1.5 Refresh Token

| Property       | Value                                                   |
|----------------|---------------------------------------------------------|
| Format         | Opaque (32 cryptographically random bytes, hex-encoded) |
| Lifetime       | 7 days                                                  |
| Client storage | `verdox_refresh` httpOnly cookie                        |
| Server storage | SHA-256 hash stored in the `sessions` table             |

The raw refresh token is never stored on the server. Only its SHA-256 hash is
persisted, so a database breach does not directly expose valid tokens.

### 1.6 Token Rotation

On every refresh request (`POST /api/v1/auth/refresh`):

1. The server validates the presented refresh token against its stored hash.
2. The old refresh token hash is invalidated (deleted from the `sessions` table).
3. A new refresh token is generated and its hash is stored.
4. A new access token is issued.
5. The new refresh token is set in the `verdox_refresh` cookie.

This rotation ensures that a stolen refresh token can be used at most once. If
the legitimate user and an attacker both attempt to use the same token, the
second attempt fails and the session is flagged.

### 1.7 Cookie Flags

All authentication cookies are set with these flags:

```
Set-Cookie: verdox_refresh=<token>;
    HttpOnly;
    Secure;              // production only (omitted in dev with HTTP)
    SameSite=Strict;
    Path=/api/v1/auth;
    Max-Age=604800       // 7 days
```

| Flag             | Purpose                                               |
|------------------|-------------------------------------------------------|
| `HttpOnly`       | Prevents JavaScript access, mitigating XSS token theft |
| `Secure`         | Cookie transmitted only over HTTPS (production)       |
| `SameSite=Strict`| Cookie not sent on cross-origin requests, mitigating CSRF |
| `Path=/api/v1/auth` | Cookie scoped to auth endpoints only               |

### 1.8 Account Lockout

Failed login attempts are tracked in Redis to prevent brute-force attacks.

| Parameter         | Value                                             |
|-------------------|---------------------------------------------------|
| Threshold         | 10 consecutive failed attempts                    |
| Cooldown period   | 30 minutes                                        |
| Tracking key      | `lockout:{user_id}`                               |
| Storage           | Redis with 1800s TTL                              |

Implementation:

1. On each failed login, `INCR lockout:{user_id}` and set `EXPIRE` to 1800
   seconds on the first failure.
2. If the counter reaches 10, subsequent login attempts return
   `429 Too Many Requests` with a `Retry-After: 1800` header.
3. On a successful login, the `lockout:{user_id}` key is deleted.

### 1.9 Session Management

| Parameter                | Value                                      |
|--------------------------|--------------------------------------------|
| Max concurrent sessions  | 5 per user                                 |
| Eviction policy          | Oldest session removed when limit exceeded |
| Session storage          | PostgreSQL (`sessions` table) + Redis cache |

When a user logs in and already has 5 active sessions, the oldest session (by
`created_at`) is invalidated: its row is deleted from the `sessions` table and
its Redis cache entry is removed. This prevents unbounded session accumulation
while allowing reasonable multi-device usage.

---

## PAT Security

GitHub Personal Access Tokens (PATs) are stored at the team level. Each team
has a single PAT stored in the `teams` table (`github_pat_encrypted` column),
encrypted at rest with AES-256-GCM using the `GITHUB_TOKEN_ENCRYPTION_KEY`
environment variable. Only team admins can set or rotate the team's PAT.

- PAT values are never returned in API responses -- only metadata
  (`has_pat: true/false`, `pat_expires_at`) is exposed.
- PATs are never logged in any log output.
- Decrypted only when needed for git operations (clone, fetch). The decrypted
  value is not cached and is discarded immediately after use.
- PAT resolution for repository operations: repository -> owning team -> team's PAT.
- DinD test containers never receive the PAT.
- Rate limiting on PAT endpoints: 5 requests per minute per team admin.

---

## 2. Authorization

### 2.1 Role Hierarchies

**System roles** (stored in the `users.role` column):

```
root > moderator > user
```

| Role         | Capabilities                                            |
|--------------|---------------------------------------------------------|
| `root`       | Full platform access: manage all users, teams, settings |
| `moderator`  | Manage users and platform configuration                 |
| `user`       | Standard access: own resources, assigned teams          |

**Team roles** (stored in the `team_members.role` column):

```
admin > maintainer > viewer
```

| Role         | Capabilities                                              |
|--------------|-----------------------------------------------------------|
| `admin`      | Manage team settings, members, and all team resources     |
| `maintainer` | Manage test runs and repositories within the team         |
| `viewer`     | View team resources, trigger test runs on assigned repos  |

### 2.2 Middleware Chain

Every authenticated request passes through the middleware chain in order:

```
Request → AuthMiddleware → RequireRole → RequireTeamRole → Handler
```

| Middleware        | Responsibility                                        |
|-------------------|-------------------------------------------------------|
| `AuthMiddleware`  | Validates the JWT, extracts user claims, rejects expired or malformed tokens |
| `RequireRole`     | Checks the user's system role against the minimum required role for the endpoint |
| `RequireTeamRole` | Checks the user's team role for the target team (loaded from `team_members`), enforces minimum team role |

### 2.3 root Bypass

Users with the `root` system role bypass all team role checks. This
allows platform operators to manage any team without requiring explicit team
membership. The bypass is implemented in `RequireTeamRole` middleware and is
logged as an audit event.

### 2.4 Endpoint Authorization Matrix

| Endpoint Group          | Min System Role | Min Team Role | Notes                    |
|-------------------------|-----------------|---------------|--------------------------|
| `POST /auth/signup`     | Public          | --            | Rate limited             |
| `POST /auth/login`      | Public          | --            | Rate limited             |
| `POST /auth/refresh`    | Public          | --            | Requires valid refresh cookie |
| `POST /auth/logout`     | `user`          | --            |                          |
| `GET /users/me`         | `user`          | --            |                          |
| `PUT /users/me`         | `user`          | --            |                          |
| `GET /repositories`     | `user`          | --            | Returns user's repos     |
| `POST /repositories`    | `user`          | --            |                          |
| `GET /repositories/:id` | `user`          | `viewer`      | Team-scoped              |
| `PUT /repositories/:id` | `user`          | `maintainer`  |                          |
| `DELETE /repositories/:id` | `user`       | `admin`       |                          |
| `POST /test-runs`       | `user`          | `viewer`      |                          |
| `GET /test-runs/:id`    | `user`          | `viewer`      |                          |
| `GET /teams`            | `user`          | --            | Returns user's teams     |
| `POST /teams`           | `user`          | --            | Creator becomes team admin |
| `PUT /teams/:id`        | `user`          | `admin`       |                          |
| `DELETE /teams/:id`     | `user`          | `admin`       |                          |
| `POST /teams/:id/members` | `user`       | `admin`       |                          |
| `PUT /teams/:id/members/:uid` | `user`   | `admin`       | Role changes             |
| `DELETE /teams/:id/members/:uid` | `user`| `admin`       |                          |
| `GET /admin/users`      | `moderator`     | --            |                          |
| `PUT /admin/users/:id/role` | `root`       | --            |                          |
| `PUT /admin/users/:id/deactivate` | `moderator` | --      |                          |
| `POST /webhooks/github` | Public          | --            | Signature-verified       |

### 2.5 Frontend Route Protection

Next.js middleware (`middleware.ts`) intercepts route transitions and checks
the user's JWT claims before rendering protected pages. If the token is missing
or the role is insufficient, the user is redirected to `/login`.

This is a UX convenience only. **All security enforcement happens server-side.**
A user who bypasses the frontend middleware will still be rejected by the
backend middleware chain.

### 2.6 Server-Side Enforcement

No client-side role check is trusted for security purposes. The frontend may
conditionally show or hide UI elements based on role, but the backend always
re-validates permissions on every request. This ensures that API calls made
directly (e.g., via curl) are subject to the same authorization rules.

---

## 3. Input Validation

### 3.1 Per-Field Validation Rules

| Field         | Rules                                                             |
|---------------|-------------------------------------------------------------------|
| `username`    | 3--30 characters, regex `^[a-zA-Z0-9_]+$`, not in reserved list (`admin`, `root`, `system`, `superadmin`, `api`, `www`) |
| `email`       | Valid RFC 5322 format, max 255 characters, normalized to lowercase |
| `password`    | Min 8 characters, at least 1 uppercase, 1 lowercase, 1 digit     |
| `team name`   | 1--128 characters, leading and trailing whitespace trimmed        |
| `repo name`   | 1--255 characters                                                 |
| `branch`      | 1--255 characters, validated against the GitHub API response      |
| `commit_hash` | Exactly 40 hexadecimal characters (`^[0-9a-f]{40}$`)             |
| UUID params   | Valid UUID v4 format, validated before any database query          |

### 3.2 Backend Validation

The backend uses `go-playground/validator` with custom validators registered at
startup:

```go
validate := validator.New()
validate.RegisterValidation("username", validateUsername)
validate.RegisterValidation("commit_hash", validateCommitHash)
```

All handler functions call `validate.Struct()` on the bound request struct
before executing any business logic. Validation errors are returned as
structured JSON with per-field messages and a `400 VALIDATION_ERROR` status.

### 3.3 Frontend Validation

The frontend uses Zod schemas integrated with React Hook Form. Each form has a
corresponding Zod schema that mirrors the backend rules:

```typescript
const signupSchema = z.object({
  username: z.string().min(3).max(30).regex(/^[a-zA-Z0-9_]+$/),
  email: z.string().email().max(255),
  password: z.string().min(8)
    .regex(/[A-Z]/, "must contain an uppercase letter")
    .regex(/[a-z]/, "must contain a lowercase letter")
    .regex(/[0-9]/, "must contain a digit"),
});
```

### 3.4 Sanitization

- All string inputs are trimmed of leading and trailing whitespace.
- Email addresses are normalized to lowercase before storage and comparison.
- HTML is never interpreted from user input. All user-provided text is rendered
  as plain text in the frontend (React's default JSX escaping).

---

## 4. Rate Limiting

### 4.1 Implementation

Rate limiting uses a Redis-based sliding window algorithm. Each request
increments a counter in a sorted set keyed by the combination of endpoint group
and identifier (IP or user ID). Entries older than the window are pruned on
each request.

### 4.2 Rate Limit Table

| Endpoint Group                        | Limit  | Window | Key      |
|---------------------------------------|--------|--------|----------|
| Auth (signup, login, forgot-password) | 5 req  | 1 min  | IP       |
| Auth (refresh)                        | 10 req | 1 min  | User ID  |
| API (authenticated)                   | 100 req| 1 min  | User ID  |
| Webhooks                              | 50 req | 1 min  | IP       |
| Admin                                 | 30 req | 1 min  | User ID  |

### 4.3 Response Behavior

Every API response includes rate limit headers:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 87
X-RateLimit-Reset: 1712345700
```

When the limit is exceeded, the server responds with:

```
HTTP/1.1 429 Too Many Requests
Retry-After: 23
Content-Type: application/json

{
  "error": {
    "code": "RATE_LIMITED",
    "message": "Too many requests. Try again in 23 seconds."
  }
}
```

The `Retry-After` header contains the number of seconds until the client can
retry.

---

## 5. CORS Configuration

CORS is configured via Echo's built-in middleware, restricting cross-origin
requests to the known frontend origin only.

```go
middleware.CORSWithConfig(middleware.CORSConfig{
    AllowOrigins:     []string{config.FrontendURL}, // e.g., "http://localhost:3000"
    AllowMethods:     []string{
        http.MethodGet,
        http.MethodPost,
        http.MethodPut,
        http.MethodDelete,
        http.MethodOptions,
    },
    AllowHeaders:     []string{
        echo.HeaderAuthorization,
        echo.HeaderContentType,
        "X-Request-ID",
    },
    AllowCredentials: true,
    MaxAge:           86400, // 24 hours preflight cache
})
```

| Setting            | Value                         | Rationale                      |
|--------------------|-------------------------------|--------------------------------|
| `AllowOrigins`     | Frontend URL only             | No wildcard; restricts to known client |
| `AllowCredentials` | `true`                        | Required for httpOnly cookie auth |
| `MaxAge`           | 86400 (24 hours)              | Reduces preflight request volume |
| `AllowHeaders`     | `Authorization`, `Content-Type`, `X-Request-ID` | Minimal set for API usage |

**Production note:** When the frontend is served from a different domain, update
`FRONTEND_URL` in the environment configuration. Never set `AllowOrigins` to
`*` as it is incompatible with `AllowCredentials: true` and would weaken the
security posture.

---

## 6. SQL Injection Prevention

### 6.1 Parameterized Queries

All database queries use parameterized statements with positional placeholders.
String concatenation is never used to build SQL queries.

```go
// Correct -- parameterized query
row := db.QueryRowContext(ctx,
    "SELECT id, username, email FROM users WHERE id = $1", userID)

// NEVER -- string concatenation (this pattern is prohibited)
// query := fmt.Sprintf("SELECT * FROM users WHERE id = '%s'", userID)
```

### 6.2 Repository Layer Enforcement

The `internal/repository/` package is the only layer that executes SQL. Handlers
and services never construct SQL directly. This boundary makes it practical to
audit all queries in a single location.

### 6.3 Named Parameters

For queries with many parameters, `sqlx` named parameters improve readability
without sacrificing safety:

```go
_, err := db.NamedExecContext(ctx,
    `INSERT INTO users (username, email, password_hash, role)
     VALUES (:username, :email, :password_hash, :role)`, user)
```

### 6.4 Prepared Statements

Frequently executed queries (login lookup, session validation, test run
insertion) use prepared statements created at application startup. This
provides both parameterization safety and a minor performance benefit from
query plan caching.

---

## 7. XSS Prevention

### 7.1 React Auto-Escaping

Next.js (React) auto-escapes all values embedded in JSX by default. User input
rendered via `{variable}` is escaped before insertion into the DOM. The codebase
does not use `dangerouslySetInnerHTML` for any user-provided content.

### 7.2 Content-Security-Policy

The following CSP header restricts the sources from which the browser may load
resources:

```
Content-Security-Policy:
    default-src 'self';
    script-src 'self' 'unsafe-inline' 'unsafe-eval';
    style-src 'self' 'unsafe-inline' https://fonts.googleapis.com;
    font-src 'self' https://fonts.gstatic.com;
    img-src 'self' https://avatars.githubusercontent.com data:;
    connect-src 'self' https://api.github.com;
```

| Directive     | Value                                          | Rationale                              |
|---------------|------------------------------------------------|----------------------------------------|
| `default-src` | `'self'`                                       | Baseline: only same-origin resources   |
| `script-src`  | `'self' 'unsafe-inline' 'unsafe-eval'`         | Next.js requires inline scripts and eval for SSR/hydration |
| `style-src`   | `'self' 'unsafe-inline' https://fonts.googleapis.com` | Styled-components and Google Fonts |
| `font-src`    | `'self' https://fonts.gstatic.com`             | Google Fonts CDN                       |
| `img-src`     | `'self' https://avatars.githubusercontent.com data:` | GitHub avatars and inline images  |
| `connect-src` | `'self' https://api.github.com`                | API calls to backend and GitHub        |

### 7.3 Additional Security Headers

```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Referrer-Policy: strict-origin-when-cross-origin
```

| Header                    | Value                               | Purpose                            |
|---------------------------|-------------------------------------|-------------------------------------|
| `X-Content-Type-Options`  | `nosniff`                           | Prevents MIME-type sniffing         |
| `X-Frame-Options`         | `DENY`                              | Prevents clickjacking via iframes   |
| `Referrer-Policy`         | `strict-origin-when-cross-origin`   | Limits referrer leakage to external sites |

### 7.4 Log Output Rendering

Test run log output (stdout/stderr from containers) is displayed in `<pre>` tags
with proper escaping via React's default behavior. The `dangerouslySetInnerHTML`
API is never used for log content. Any HTML entities in logs are rendered as
literal text, not interpreted.

---

## 8. CSRF Protection

### 8.1 Primary Defense: SameSite Cookies

All authentication cookies use `SameSite=Strict`, which instructs the browser
to omit cookies on all cross-origin requests. This provides strong CSRF
protection for cookie-based authentication.

### 8.2 Secondary Defense: Authorization Header

API clients that use the `Authorization: Bearer <token>` header for
authentication are inherently protected from CSRF. The browser will not
automatically attach an `Authorization` header to cross-origin requests,
unlike cookies.

### 8.3 State-Changing Request Policy

No state-changing request (POST, PUT, DELETE) relies on cookie-only
authentication. All such requests are validated against either:

- The `Authorization` header, or
- A combination of httpOnly cookie plus `SameSite=Strict` enforcement.

### 8.4 Origin Header Validation

The backend validates the `Origin` header on all POST, PUT, and DELETE requests
against the configured `FRONTEND_URL`. Requests with a mismatched or missing
`Origin` header (except same-origin requests) are rejected with
`403 Forbidden`.

---

## 9. Webhook Security (Planned for v2)

> **Note:** Webhook-based triggers are deferred to v2. The current architecture
> uses a PAT-based model where Verdox clones repositories using team-level
> Personal Access Tokens. The webhook security design below is retained for
> future implementation.

### 9.1 Signature Verification

GitHub webhook payloads are verified using HMAC-SHA256 signatures. Each
repository has its own `webhook_secret` stored in the database.

### 9.2 Verification Flow

1. Read the raw request body (do not parse JSON first).
2. Extract the `X-Hub-Signature-256` header. It contains `sha256={signature}`.
3. Compute `HMAC-SHA256(webhook_secret, raw_body)` and hex-encode the result.
4. Compare the computed signature to the header value using timing-safe
   comparison.
5. If the comparison fails, reject the request with `401 Unauthorized`.

```go
mac := hmac.New(sha256.New, []byte(webhookSecret))
mac.Write(payload)
expected := mac.Sum(nil)

sig, _ := hex.DecodeString(headerSig) // strip "sha256=" prefix first
if !hmac.Equal(sig, expected) {
    return echo.NewHTTPError(http.StatusUnauthorized, "invalid webhook signature")
}
```

### 9.3 Timing-Safe Comparison

The comparison uses `crypto/subtle.ConstantTimeCompare` (or the equivalent
`hmac.Equal`) to prevent timing side-channel attacks that could be used to
forge signatures byte by byte.

### 9.4 Rejection Logging

All webhook events are logged with the following fields:

- Event type (`X-GitHub-Event` header)
- Delivery ID (`X-GitHub-Delivery` header)
- Repository full name
- Verification result (accepted or rejected)
- Rejection reason (missing header, invalid format, signature mismatch)

> Webhooks are planned for v2. In v1, test runs are triggered manually.

---

## 10. Docker-in-Docker (DinD) Security

### 10.1 Architecture

The Verdox test runner uses Docker-in-Docker to execute test suites. The
architecture consists of two layers:

```
Host Docker daemon
  └── verdox-runner (DinD daemon container) ← privileged
        └── test-container-1               ← NOT privileged
        └── test-container-2               ← NOT privileged
```

The DinD daemon container (`verdox-runner`) runs in privileged mode because
Docker requires elevated capabilities to run a Docker daemon inside a
container. The test containers spawned by the inner Docker daemon are
unprivileged.

### 10.2 Resource Limits

Every test container is constrained to prevent resource exhaustion:

| Resource | Limit         | Docker Flag            |
|----------|---------------|------------------------|
| CPU      | 2 cores       | `--cpus=2`             |
| Memory   | 2 GB          | `--memory=2g`          |
| Disk     | 5 GB tmpfs    | `--tmpfs /tmp:size=5g` |
| PIDs     | 256 processes | `--pids-limit=256`     |

These limits prevent a single misbehaving test from consuming all resources on
the host and affecting other test runs or platform services.

### 10.3 Network Isolation

Test containers are placed on an isolated Docker network with no access to:

- The host network or host services.
- Other platform services (PostgreSQL, Redis, the Go backend).
- Other test containers running concurrently.

The isolated network is created per test run and removed after completion.

### 10.4 Filesystem Restrictions

| Restriction             | Implementation                                |
|-------------------------|-----------------------------------------------|
| No host volume mounts   | Test containers cannot mount host directories |
| Read-only root FS       | `--read-only` flag where applicable           |
| Writable tmpfs only     | Writable directories backed by tmpfs with size limits |
| Read-only repo clone    | The local repository clone is mounted into the test container as read-only (`-v /path/to/repo:/workspace:ro`) to prevent test code from modifying source files |

### 10.5 Container Lifecycle

- Containers are removed immediately after the test run completes or times out
  (`defer` pattern in Go with `docker rm -f`).
- A cleanup goroutine runs periodically to remove any orphaned containers that
  were not properly cleaned up (e.g., due to a crash).
- Container names include the test run UUID for traceability.

### 10.6 Image Whitelist

Only pre-approved base images can be used for test containers. The list is
configurable via environment variable or configuration file. Attempting to use a
non-whitelisted image results in a rejected test run with a descriptive error.

### 10.7 Docker Socket Isolation

The Docker socket inside the DinD container is never exposed to test containers.
Test containers cannot spawn additional containers, access the Docker API, or
escalate privileges through the Docker daemon.

- Local repository clone mounted read-only (-v path:/workspace:ro) into test containers

---

## 11. Secrets Management

### 11.1 Required Secrets

The following secrets are required for Verdox to operate:

```
JWT_SECRET=<min 32 chars, used for HS256 JWT signing>
DATABASE_URL=postgres://user:pass@host:5432/verdox?sslmode=disable
REDIS_URL=redis://host:6379
ROOT_EMAIL=<email for the bootstrapped root user>
ROOT_PASSWORD=<password for the bootstrapped root user>
GITHUB_TOKEN_ENCRYPTION_KEY=<32-byte hex string for PAT encryption>
VERDOX_REPO_BASE_PATH=<local path where repositories are cloned>
VERDOX_OPENAI_API_KEY=<optional, for AI-powered features>
```

### 11.2 Development Environment

In development, secrets are stored in a `.env` file at the project root. This
file is excluded from version control via `.gitignore`. A `.env.example` file
with placeholder values is committed to the repository to document the required
variables.

### 11.3 Production Environment

In production, secrets should be injected via one of:

- Docker secrets (preferred for Docker Compose deployments).
- Environment variables set by the deployment platform.
- A secrets manager (e.g., HashiCorp Vault) if available.

The `.env` file approach is not recommended for production because the file
persists on disk and may be inadvertently exposed.

### 11.4 Code and Image Exclusion

Secrets are never:

- Hard-coded in source code.
- Baked into Docker images at build time.
- Logged by the application (zerolog is configured to redact known secret
  field names).
- Included in error messages or API responses.

### 11.5 Secret Rotation

**JWT_SECRET rotation:**

1. Deploy the new secret as `JWT_SECRET`.
2. All existing access tokens (15-minute lifetime) expire naturally within 15
   minutes.
3. Refresh tokens remain valid because they are verified by hash lookup, not
   by JWT signature.

**GITHUB_TOKEN_ENCRYPTION_KEY rotation:**

1. Deploy the new key.
2. Run the `scripts/rotate-encryption-key.sh` script to re-encrypt all stored
   PATs with the new key.
3. Remove the old key from the environment.

**Database and Redis credentials:**

1. Update credentials on the database/Redis server.
2. Update `DATABASE_URL` and `REDIS_URL` in the environment.
3. Restart the backend service.

### 11.6 PAT Security

GitHub Personal Access Tokens (PATs) are stored at the team level on the
`teams` table and treated as highly sensitive credentials. The following
controls apply:

- **Encrypted at rest:** PATs are encrypted using AES-256-GCM before storage
  in the `teams.github_pat_encrypted` column. The encryption key is
  `GITHUB_TOKEN_ENCRYPTION_KEY`.
- **Team-scoped:** Each team has exactly one PAT. Only team admins can
  set, rotate, or revoke the team's PAT.
- **Never returned in API responses:** API endpoints that return team or
  repository data never include the PAT value. The presence of a PAT is
  indicated by a boolean field (e.g., `has_pat: true`), never by the token
  itself.
- **Never logged:** PAT values are excluded from all log output. The `zerolog`
  configuration redacts any field named `pat`, `token`, or
  `github_access_token`.
- **Decrypted only when needed:** PATs are decrypted in memory only at the
  moment they are needed for git clone or git pull operations. The decrypted
  value is not cached and is discarded immediately after use.
- **DinD containers never receive the PAT:** The PAT is used only by the
  backend worker for git operations. Test containers have no access to it.
- **PAT resolution path:** repository -> team (via `team_repositories`) ->
  `teams.github_pat_encrypted`.
- See [GITHUB-PAT-GUIDE.md](./GITHUB-PAT-GUIDE.md) for detailed PAT creation
  and maintenance instructions.

---

## 12. Security Headers (Nginx)

The Nginx reverse proxy adds the following security headers to all responses:

```nginx
add_header X-Content-Type-Options "nosniff" always;
add_header X-Frame-Options "DENY" always;
add_header X-XSS-Protection "1; mode=block" always;
add_header Referrer-Policy "strict-origin-when-cross-origin" always;
add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' https://avatars.githubusercontent.com data:; connect-src 'self' https://api.github.com;" always;
add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
add_header Permissions-Policy "camera=(), microphone=(), geolocation=()" always;
```

| Header                         | Value                              | Purpose                                 |
|--------------------------------|------------------------------------|-----------------------------------------|
| `X-Content-Type-Options`       | `nosniff`                          | Prevents MIME-type sniffing attacks      |
| `X-Frame-Options`              | `DENY`                             | Blocks all iframe embedding (clickjacking) |
| `X-XSS-Protection`             | `1; mode=block`                    | Legacy XSS filter for older browsers    |
| `Referrer-Policy`              | `strict-origin-when-cross-origin`  | Sends full referrer same-origin, origin-only cross-origin |
| `Content-Security-Policy`      | (see Section 7.2)                  | Restricts resource loading sources      |
| `Strict-Transport-Security`    | `max-age=31536000; includeSubDomains` | Enforces HTTPS for 1 year, including subdomains |
| `Permissions-Policy`           | `camera=(), microphone=(), geolocation=()` | Disables browser features not needed by the application |

**Note:** `X-XSS-Protection` is a legacy header. Modern browsers rely on CSP
instead. It is included for defense-in-depth with older browser versions.

---

## 13. Dependency Security

### 13.1 Go Dependencies

| Practice             | Implementation                                    |
|----------------------|---------------------------------------------------|
| Integrity check      | `go mod verify` runs in CI on every build         |
| Automated updates    | Dependabot configured for weekly Go module updates |
| Vulnerability scan   | `govulncheck` runs in CI                          |

### 13.2 Node.js Dependencies

| Practice             | Implementation                                    |
|----------------------|---------------------------------------------------|
| Audit                | `npm audit` runs in CI on every build             |
| Automated updates    | Dependabot configured for weekly npm updates      |
| Lock file            | `package-lock.json` committed and used for installs (`npm ci`) |

### 13.3 Docker Images

| Practice             | Implementation                                    |
|----------------------|---------------------------------------------------|
| Image pinning        | Production Dockerfiles pin images to specific digests (not tags) |
| Vulnerability scan   | `trivy` scans all images in CI before deployment  |
| Minimal base images  | Alpine-based images used where possible            |

### 13.4 Update Schedule

Dependencies are reviewed and updated monthly. Critical security patches are
applied as soon as they are identified, outside the regular schedule.

---

## 14. Audit Logging

### 14.1 Log Format

All security-relevant events are logged as structured JSON using `zerolog`.
Each log entry includes:

```json
{
  "level": "info",
  "time": "2026-04-05T10:30:00Z",
  "event": "auth.login.success",
  "user_id": "a1b2c3d4-...",
  "ip": "203.0.113.42",
  "user_agent": "Mozilla/5.0 ...",
  "request_id": "req_abc123"
}
```

### 14.2 Logged Events

**Authentication events:**

| Event                    | Log Level | Fields                              |
|--------------------------|-----------|-------------------------------------|
| Login success            | `info`    | `user_id`, `ip`, `user_agent`       |
| Login failure            | `warn`    | `login` (username/email), `ip`, `reason` |
| Signup                   | `info`    | `user_id`, `username`, `ip`         |
| Password reset request   | `info`    | `email`, `ip`                       |
| Token refresh            | `info`    | `user_id`, `session_id`             |
| Logout                   | `info`    | `user_id`, `session_id`             |

**Authorization events:**

| Event                    | Log Level | Fields                              |
|--------------------------|-----------|-------------------------------------|
| Access denied (role)     | `warn`    | `user_id`, `path`, `required_role`, `actual_role` |
| Access denied (team role)| `warn`    | `user_id`, `team_id`, `required_role`, `actual_role` |
| root bypass              | `info`    | `user_id`, `team_id`, `action`      |

**Admin events:**

| Event                    | Log Level | Fields                              |
|--------------------------|-----------|-------------------------------------|
| Role change              | `warn`    | `admin_id`, `target_user_id`, `old_role`, `new_role` |
| User deactivation        | `warn`    | `admin_id`, `target_user_id`        |

**Webhook events:**

| Event                    | Log Level | Fields                              |
|--------------------------|-----------|-------------------------------------|
| Webhook received         | `info`    | `delivery_id`, `event_type`, `repo` |
| Webhook verified         | `info`    | `delivery_id`, `repo`               |
| Webhook rejected         | `warn`    | `delivery_id`, `reason`, `ip`       |

**Test runner events:**

| Event                    | Log Level | Fields                              |
|--------------------------|-----------|-------------------------------------|
| Container started        | `info`    | `test_run_id`, `image`, `container_id` |
| Container completed      | `info`    | `test_run_id`, `exit_code`, `duration` |
| Container timeout        | `warn`    | `test_run_id`, `timeout_seconds`    |
| Container OOM killed     | `warn`    | `test_run_id`, `memory_limit`       |

---

## 15. Incident Response

### 15.1 Detection

All `401 Unauthorized` and `403 Forbidden` responses are logged with full
request details (path, method, IP, user agent, user ID if available). These logs
are the primary signal for detecting unauthorized access attempts.

### 15.2 Alert Thresholds

| Condition                                      | Action                           |
|------------------------------------------------|----------------------------------|
| >10 failed logins from a single IP in 5 minutes | Alert admin, auto-block IP for 1 hour |
| >3 webhook signature failures per hour         | Alert admin, log source IP       |
| Repeated rate limit violations from same source | Escalate to temporary IP ban (1 hour) |
| Any `root` role assignment                       | Immediate admin notification     |

### 15.3 Rate Limit Escalation

When a client repeatedly exceeds rate limits (more than 5 distinct 429
responses within 10 minutes), the system escalates to a temporary IP ban:

1. The IP is added to a Redis blocklist (`blocked_ip:{ip}`) with a 1-hour TTL.
2. All requests from the blocked IP receive `403 Forbidden` without processing.
3. The block is logged as an audit event.
4. The block expires automatically after 1 hour.

### 15.4 Admin Notifications

Security events that trigger alerts are surfaced as in-app notifications to
users with the `moderator` or `root` role. The notification includes:

- Event type and severity.
- Timestamp and source IP.
- Affected resource (user account, repository, webhook).
- Recommended action.

### 15.5 Post-Incident Procedures

After a security incident is identified:

1. **Contain:** Block the source IP or deactivate the compromised account.
2. **Assess:** Review audit logs to determine the scope of the incident.
3. **Remediate:** Rotate affected secrets, invalidate compromised sessions.
4. **Document:** Record the incident timeline, root cause, and remediation
   steps.
5. **Improve:** Update alerting thresholds or add new controls to prevent
   recurrence.
