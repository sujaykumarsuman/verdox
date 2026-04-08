# Verdox -- Authentication & Authorization (LLD)

> Go 1.26+ | Echo v4 | bcrypt | golang-jwt/jwt/v5 | Redis 7

---

## 1. Authentication Overview

Verdox uses a stateful JWT authentication model combining short-lived access
tokens with long-lived refresh tokens. Passwords are hashed with bcrypt before
storage. Sessions are tracked in both PostgreSQL (durable) and Redis (fast
lookup).

**Token strategy:**

| Token | Type | Lifetime | Storage | Purpose |
|-------|------|----------|---------|---------|
| Access token | Signed JWT (HS256) | 15 minutes | `Authorization: Bearer <token>` header or `verdox_access` httpOnly cookie | Carries user identity and role claims for stateless request authorization |
| Refresh token | Opaque random bytes | 7 days | `verdox_refresh` httpOnly cookie (client), SHA-256 hash in `sessions` table (server) | Used exclusively to obtain a new access token without re-entering credentials |

**Key principles:**

- Passwords never leave the backend in plaintext. Only the bcrypt hash is stored.
- Refresh tokens are rotated on every use. The previous token is invalidated
  immediately, preventing replay attacks.
- All auth cookies use `httpOnly`, `Secure` (in production), and
  `SameSite=Strict` flags to mitigate XSS and CSRF.
- Session records in Redis allow O(1) revocation checks on every authenticated
  request.

---

## 2. Signup Flow

**Endpoint:** `POST /api/v1/auth/signup`

### Step-by-step

1. Client sends a POST request with a JSON body containing `username`, `email`,
   and `password`.
2. **Validate input:**
   - `username`: 3--30 characters, alphanumeric and underscores only
     (`^[a-zA-Z0-9_]{3,30}$`).
   - `email`: valid email format (RFC 5322 simplified check).
   - `password`: minimum 8 characters, at least 1 uppercase letter, 1 lowercase
     letter, and 1 digit.
3. **Check uniqueness:** query the `users` table for existing rows matching the
   username or email. If either exists, return `409 Conflict` with a specific
   field-level error.
4. **Hash password:** use bcrypt with cost factor 12
   (`bcrypt.GenerateFromPassword([]byte(password), 12)`).
5. **Insert user:** `INSERT INTO users (username, email, password_hash, role)`
   with `role` defaulting to `'user'`. After signup, the user has NO team
   memberships and must discover and request to join a team before they can
   access repos/tests. Post-signup, user is redirected to Team Discovery page.
6. **Generate JWT pair:**
   - Build the access token with claims (see Section 4).
   - Generate a 32-byte cryptographically random refresh token
     (`crypto/rand.Read`), hex-encode it to produce a 64-character string.
7. **Create session:** compute `SHA-256(refresh_token)` and insert a row into
   the `sessions` table with `user_id`, `refresh_token_hash`, and
   `expires_at = now() + 7 days`.
8. **Cache session in Redis:** `SET session:{user_id} <session_id> EX 604800`
   (7 days in seconds). The value is the session UUID so the middleware can
   cross-reference it.
9. **Return response:** access token in the JSON body, refresh token set as an
   httpOnly cookie.

### Request

```json
POST /api/v1/auth/signup
Content-Type: application/json

{
  "username": "sujay",
  "email": "sujay@example.com",
  "password": "Str0ngP@ss"
}
```

### Response (201 Created)

```json
{
  "status": "success",
  "data": {
    "user": {
      "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "username": "sujay",
      "email": "sujay@example.com",
      "role": "user",
      "created_at": "2026-04-05T10:30:00Z"
    },
    "access_token": "eyJhbGciOiJIUzI1NiIs..."
  }
}
```

```
Set-Cookie: verdox_refresh=ae4f...c9b2; HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth; Max-Age=604800
```

### Error Responses

| Status | Condition | Body |
|--------|-----------|------|
| 400 | Validation failure | `{"status":"error","message":"validation failed","errors":[{"field":"password","message":"must contain at least 1 uppercase letter"}]}` |
| 409 | Duplicate username or email | `{"status":"error","message":"username already taken"}` |
| 500 | Internal error | `{"status":"error","message":"internal server error"}` |

---

## Root User Bootstrap

- Root user is created on first server startup from environment variables:
  - `ROOT_EMAIL` (required)
  - `ROOT_PASSWORD` (required)
- If user with `ROOT_EMAIL` doesn't exist, create with `role='root'`
- If user exists but `role!='root'`, update to `role='root'`
- Root cannot be created via signup or admin API
- There is exactly one root user at any time

---

## 3. Login Flow

**Endpoint:** `POST /api/v1/auth/login`

### Step-by-step

1. Client sends a POST request with `login` (username or email) and `password`.
2. **Look up user:** query the `users` table by username first; if no match,
   query by email. Use a single query:
   ```sql
   SELECT id, username, email, password_hash, role
     FROM users
    WHERE username = $1 OR email = $1;
   ```
3. **Check account lockout:** query Redis for `lockout:{user_id}`. If the key
   exists and the failed attempt count is >= 10, return `429 Too Many Requests`
   with a `Retry-After` header.
4. **Compare password:** `bcrypt.CompareHashAndPassword(hash, password)`.
   - On failure: increment `lockout:{user_id}` in Redis
     (`INCR lockout:{user_id}`, `EXPIRE lockout:{user_id} 1800` on first
     failure). Return `401 Unauthorized` with a generic message.
   - On success: delete the `lockout:{user_id}` key.
5. **Generate JWT pair:** same as signup (access token + refresh token).
6. **Create session:** insert into `sessions`, cache in Redis.
7. **Return tokens:** access token in body, refresh token in cookie.

### Request

```json
POST /api/v1/auth/login
Content-Type: application/json

{
  "login": "sujay",
  "password": "Str0ngP@ss"
}
```

### Response (200 OK)

```json
{
  "status": "success",
  "data": {
    "user": {
      "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "username": "sujay",
      "email": "sujay@example.com",
      "role": "user",
      "avatar_url": null,
      "created_at": "2026-04-05T10:30:00Z"
    },
    "access_token": "eyJhbGciOiJIUzI1NiIs..."
  }
}
```

```
Set-Cookie: verdox_refresh=ae4f...c9b2; HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth; Max-Age=604800
```

### Error Responses

| Status | Condition | Body |
|--------|-----------|------|
| 400 | Missing or empty fields | `{"status":"error","message":"login and password are required"}` |
| 401 | Invalid credentials | `{"status":"error","message":"invalid credentials"}` |
| 429 | Account locked (10 failed attempts) | `{"status":"error","message":"account temporarily locked, try again later"}` + `Retry-After: 1800` |

---

## 4. JWT Structure

### Access Token

**Algorithm:** HS256 (HMAC-SHA256)
**Signing key:** `JWT_SECRET` environment variable (minimum 32 characters)
**Expiry:** 15 minutes from issuance

**Claims payload:**

```json
{
  "sub": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "username": "sujay",
  "role": "user",
  "iat": 1712345678,
  "exp": 1712346578
}
```

| Claim | Type | Description |
|-------|------|-------------|
| `sub` | string (UUID) | User ID from the `users` table. Used as the primary identity reference |
| `username` | string | Display name for logging and audit. Informational only -- not used for authorization decisions |
| `role` | string (enum) | System role (`root`, `moderator`, `user`). Used by `RequireRole` middleware |
| `iat` | number (Unix) | Issued-at timestamp |
| `exp` | number (Unix) | Expiration timestamp. 15 minutes after `iat` |

**Transport:** sent in the `Authorization: Bearer <token>` header for API
clients, or stored in the `verdox_access` httpOnly cookie for web clients.
The `AuthMiddleware` checks the cookie first, then falls back to the header.

### Go Claims Struct

```go
type Claims struct {
    Username string `json:"username"`
    Role     string `json:"role"`
    jwt.RegisteredClaims
}
```

### Refresh Token

- **Format:** 32 cryptographically random bytes, hex-encoded to a 64-character
  string. Not a JWT -- it carries no claims.
- **Client storage:** `verdox_refresh` httpOnly cookie with `Secure`,
  `SameSite=Strict`, and `Path=/api/v1/auth` (scoped to auth endpoints only).
- **Server storage:** the raw token is never stored. Instead,
  `SHA-256(refresh_token)` is stored in `sessions.refresh_token_hash`.
- **Expiry:** 7 days. Enforced via both the cookie `Max-Age` and the
  `sessions.expires_at` column.

### Token Generation (Go pseudocode)

```go
func GenerateTokenPair(user *User) (accessToken string, refreshToken string, err error) {
    // Access token
    now := time.Now()
    claims := Claims{
        Username: user.Username,
        Role:     string(user.Role),
        RegisteredClaims: jwt.RegisteredClaims{
            Subject:   user.ID.String(),
            IssuedAt:  jwt.NewNumericDate(now),
            ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    accessToken, err = token.SignedString([]byte(os.Getenv("JWT_SECRET")))
    if err != nil {
        return "", "", fmt.Errorf("signing access token: %w", err)
    }

    // Refresh token (opaque)
    raw := make([]byte, 32)
    if _, err := crypto_rand.Read(raw); err != nil {
        return "", "", fmt.Errorf("generating refresh token: %w", err)
    }
    refreshToken = hex.EncodeToString(raw) // 64 hex chars

    return accessToken, refreshToken, nil
}
```

---

## 5. Token Refresh Flow

**Endpoint:** `POST /api/v1/auth/refresh`

### Step-by-step

1. Client sends a POST request. The refresh token is read from the
   `verdox_refresh` cookie -- no body required.
2. **Hash the received token:** compute `SHA-256(refresh_token)` to get the
   hash for lookup.
3. **Look up session:** check Redis cache first (`GET session:{user_id}`). If
   the session ID is cached, fetch the session row from the database using that
   ID. If not in Redis, query the `sessions` table:
   ```sql
   SELECT id, user_id, expires_at
     FROM sessions
    WHERE refresh_token_hash = $1
      AND expires_at > now();
   ```
4. **Validate:** if no matching session is found, or it has expired, return
   `401 Unauthorized`. Clear the refresh cookie.
5. **Load user:** fetch the user row to get the current role (in case it
   changed since the last token was issued).
6. **Rotate tokens:**
   - Generate a new JWT access token with fresh claims.
   - Generate a new opaque refresh token.
   - **Delete the old session** from both the database and Redis.
   - **Create a new session** with the new refresh token hash.
   - Cache the new session in Redis.
7. **Return:** new access token in the body, new refresh token in a
   replacement `verdox_refresh` cookie.

### Response (200 OK)

```json
{
  "status": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs..."
  }
}
```

```
Set-Cookie: verdox_refresh=b7d1...e8a3; HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth; Max-Age=604800
```

### Error Responses

| Status | Condition | Body |
|--------|-----------|------|
| 401 | Missing, invalid, or expired refresh token | `{"status":"error","message":"invalid or expired refresh token"}` |

### Why Token Rotation Matters

Refresh token rotation means that each refresh token can only be used once.
If an attacker steals a refresh token and the legitimate user refreshes first,
the attacker's stolen token becomes invalid. If the attacker refreshes first,
the legitimate user's next refresh attempt fails, signaling a potential
compromise. The server can then invalidate all sessions for that user as a
precaution.

---

## 6. Logout Flow

**Endpoint:** `POST /api/v1/auth/logout`

### Step-by-step

1. Client sends a POST request with a valid access token (via cookie or
   header).
2. **Extract user_id** from the access token claims (`sub` field).
3. **Delete session** from the `sessions` table:
   ```sql
   DELETE FROM sessions WHERE user_id = $1;
   ```
   This deletes all sessions for the user, effectively logging them out of all
   devices. For single-device logout, delete only the session matching the
   current refresh token hash.
4. **Delete Redis cache:** `DEL session:{user_id}`.
5. **Clear cookies:** set both `verdox_access` and `verdox_refresh` cookies
   with `Max-Age=0` to instruct the browser to remove them.
6. **Return** `204 No Content`.

### Response (204 No Content)

No body.

```
Set-Cookie: verdox_access=; HttpOnly; Secure; SameSite=Strict; Path=/; Max-Age=0
Set-Cookie: verdox_refresh=; HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth; Max-Age=0
```

---

## 7. Password Reset Flow

**Endpoint (request):** `POST /api/v1/auth/forgot-password`
**Endpoint (reset):** `POST /api/v1/auth/reset-password`

### Database Table

A dedicated `password_resets` table is used to track reset tokens:

```sql
CREATE TABLE password_resets (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL,
    token_hash  TEXT        NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT fk_password_resets_user
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE
);

CREATE INDEX idx_password_resets_user_id ON password_resets (user_id);
CREATE INDEX idx_password_resets_token_hash ON password_resets (token_hash);
```

### Step-by-step (Request Reset)

1. Client sends `POST /api/v1/auth/forgot-password` with `{ "email": "..." }`.
2. **Look up user** by email. **Always return 200 OK** regardless of whether
   the email exists. This prevents email enumeration attacks.
3. If the user exists:
   a. Generate a 32-byte cryptographically random token, hex-encode it (64
      characters).
   b. Compute `SHA-256(token)` and store it in `password_resets` with
      `user_id`, `token_hash`, and `expires_at = now() + 1 hour`.
   c. Invalidate any existing unused reset tokens for this user by setting
      `used_at = now()` on all prior rows.
   d. **Log the reset URL** to the server console (email delivery is out of
      scope for v1):
      ```
      [RESET] Password reset URL for user sujay: http://localhost:3000/reset-password?token=ae4f...c9b2
      ```
4. Return a generic success message.

### Request (Forgot Password)

```json
POST /api/v1/auth/forgot-password
Content-Type: application/json

{
  "email": "sujay@example.com"
}
```

### Response (200 OK -- always)

```json
{
  "status": "success",
  "message": "if an account with that email exists, a password reset link has been generated"
}
```

### Step-by-step (Reset Password)

1. Client sends `POST /api/v1/auth/reset-password` with `{ "token": "...", "new_password": "..." }`.
2. **Validate new password** against the same rules as signup (min 8 chars,
   1 uppercase, 1 lowercase, 1 digit).
3. **Hash the token:** compute `SHA-256(token)`.
4. **Look up reset record:**
   ```sql
   SELECT id, user_id, expires_at, used_at
     FROM password_resets
    WHERE token_hash = $1;
   ```
5. **Validate:**
   - If no record found: return `400 Bad Request` ("invalid or expired reset
     token").
   - If `used_at IS NOT NULL`: return `400` (token already used).
   - If `expires_at < now()`: return `400` (token expired).
6. **Update password:** hash the new password with bcrypt (cost 12), then:
   ```sql
   UPDATE users SET password_hash = $1, updated_at = now() WHERE id = $2;
   ```
7. **Mark token as used:**
   ```sql
   UPDATE password_resets SET used_at = now() WHERE id = $1;
   ```
8. **Invalidate all sessions** for this user:
   ```sql
   DELETE FROM sessions WHERE user_id = $1;
   ```
   Also delete the Redis session cache: `DEL session:{user_id}`.
9. Return success. The user must log in again with the new password.

### Request (Reset Password)

```json
POST /api/v1/auth/reset-password
Content-Type: application/json

{
  "token": "ae4f8b2c...64 hex chars...c9b2",
  "new_password": "N3wStr0ng!"
}
```

### Response (200 OK)

```json
{
  "status": "success",
  "message": "password has been reset successfully"
}
```

### Error Responses

| Status | Condition | Body |
|--------|-----------|------|
| 400 | Invalid token, expired, or already used | `{"status":"error","message":"invalid or expired reset token"}` |
| 400 | Password validation failure | `{"status":"error","message":"validation failed","errors":[...]}` |

---

## 8. Role Hierarchy & Authorization

Verdox enforces two layers of authorization: system-wide roles assigned to user
accounts, and team-scoped roles assigned per team membership.

### System Roles (`user_role` enum)

| Role | Source | Can Do |
|------|--------|--------|
| `root` | Bootstrapped from `.env` (`ROOT_EMAIL`, `ROOT_PASSWORD`). Only one root user. Cannot be created via API. | Everything. Create teams, promote users to moderator, manage all users, bypass all team checks. |
| `moderator` | Promoted by root via API. | Create teams, manage repositories (CRUD) without being a team member, view system stats. Cannot manage other moderators or root. |
| `user` | Default role on signup. | Request to join teams. Access is scoped to teams they belong to. |

**Role precedence:** `root` > `moderator` > `user`. A higher role implicitly
has all permissions of lower roles.

### Team Roles (`team_member_role` enum)

| Role | Can Do |
|------|--------|
| `admin` | Full team management: manage members, approve/reject join requests, assign repos, configure test suites, trigger runs, delete team. Creator of team is auto-assigned admin. |
| `maintainer` | Configure test suites per repo, trigger test runs, approve/reject join requests. Cannot manage other members' roles or delete team. |
| `viewer` | Read-only access to team's repos and test runs. View results and logs. Cannot trigger runs or modify anything. |

**Team role precedence:** `admin` > `maintainer` > `viewer`.

### Authorization Matrix

| Action | `root` | `moderator` | `user` | Team `admin` | Team `maintainer` | Team `viewer` |
|--------|:---:|:---:|:---:|:---:|:---:|:---:|
| Create own repos | Y | Y | N | -- | -- | -- |
| Create teams | Y | Y | N | -- | -- | -- |
| View all users | Y | Y | N | -- | -- | -- |
| Change user roles | Y | N | N | -- | -- | -- |
| Deactivate users | Y | N | N | -- | -- | -- |
| View system stats | Y | Y | N | -- | -- | -- |
| Manage team members | Y (bypass) | -- | -- | Y | N | N |
| Approve/reject requests | Y (bypass) | -- | -- | Y | Y | N |
| Assign repos to team | Y (bypass) | -- | -- | Y | N | N |
| Delete team | Y (bypass) | -- | -- | Y | N | N |
| Run tests on team repo | Y (bypass) | -- | -- | Y | Y | N |
| View team repo results | Y (bypass) | -- | -- | Y | Y | Y |

`Y (bypass)` means `root` bypasses all team role checks entirely and is always
authorized.

---

## 9. Middleware Chain

Echo middleware is applied in the order registered. Order matters because each
middleware depends on the context set by the one before it.

### Full Stack (in order)

```
1. RequestID
2. Logger
3. Recover
4. CORS
5. RateLimiter
6. AuthMiddleware        (skip for public routes)
7. RequireRole(...)      (applied per route group)
8. RequireTeamRole(...)  (applied per team route)
```

| # | Middleware | Purpose |
|---|-----------|---------|
| 1 | `RequestID` | Generates a unique `X-Request-Id` header for every request. Used for log correlation and distributed tracing |
| 2 | `Logger` | Logs each request with zerolog: method, path, status, latency, request ID. Uses structured JSON format |
| 3 | `Recover` | Catches panics in downstream handlers, logs the stack trace, and returns `500 Internal Server Error` |
| 4 | `CORS` | Sets `Access-Control-Allow-Origin` to the configured frontend origin. Allows credentials. Restricts methods to GET, POST, PUT, PATCH, DELETE, OPTIONS |
| 5 | `RateLimiter` | Redis-backed sliding window rate limiter. Auth endpoints: 5 requests/minute per IP. General API: 300 requests/minute per user (authenticated) or 100/minute per IP (unauthenticated) |
| 6 | `AuthMiddleware` | JWT extraction, signature validation, expiry check, session verification. Sets user in context |
| 7 | `RequireRole` | System-level role gate. Checks `user.role` against a whitelist of allowed roles |
| 8 | `RequireTeamRole` | Team-level role gate. Checks `team_members.role` for the given team |

### Public Routes (skip AuthMiddleware)

```go
public := e.Group("/api/v1/auth")
public.POST("/signup", authHandler.Signup)
public.POST("/login", authHandler.Login)
public.POST("/refresh", authHandler.Refresh)
public.POST("/forgot-password", authHandler.ForgotPassword)
public.POST("/reset-password", authHandler.ResetPassword)

// Health check
e.GET("/api/v1/health", healthHandler.Check)
```

### Protected Routes

```go
api := e.Group("/api/v1", AuthMiddleware)

// User routes
api.POST("/auth/logout", authHandler.Logout)
api.GET("/me", userHandler.GetProfile)
api.PUT("/me", userHandler.UpdateProfile)

// Repo routes (any authenticated user)
repos := api.Group("/repositories")
repos.GET("", repoHandler.List)
repos.POST("", repoHandler.Create)

// Team routes
teams := api.Group("/teams")
teams.POST("", teamHandler.Create)
teams.GET("/:team_id", RequireTeamRole("admin", "maintainer", "viewer"), teamHandler.Get)
teams.PUT("/:team_id", RequireTeamRole("admin"), teamHandler.Update)
teams.DELETE("/:team_id", RequireTeamRole("admin"), teamHandler.Delete)
teams.POST("/:team_id/members", RequireTeamRole("admin"), teamHandler.AddMember)
teams.PUT("/:team_id/members/:user_id/approve", RequireTeamRole("admin", "maintainer"), teamHandler.ApproveMember)

// Admin routes
admin := api.Group("/admin", RequireRole("root", "moderator"))
admin.GET("/users", adminHandler.ListUsers)
admin.PUT("/users/:id/role", adminHandler.ChangeRole)
admin.PUT("/users/:id/status", adminHandler.ChangeStatus)
admin.GET("/stats", adminHandler.GetStats)
```

### AuthMiddleware Implementation

```go
func AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        // 1. Extract token: check cookie first, then Authorization header
        tokenString := ""
        cookie, err := c.Cookie("verdox_access")
        if err == nil && cookie.Value != "" {
            tokenString = cookie.Value
        } else {
            auth := c.Request().Header.Get("Authorization")
            if strings.HasPrefix(auth, "Bearer ") {
                tokenString = strings.TrimPrefix(auth, "Bearer ")
            }
        }

        if tokenString == "" {
            return echo.NewHTTPError(http.StatusUnauthorized, "missing authentication token")
        }

        // 2. Parse and validate JWT
        claims := &Claims{}
        token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
            if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
            }
            return []byte(os.Getenv("JWT_SECRET")), nil
        })
        if err != nil || !token.Valid {
            return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
        }

        // 3. Verify session exists in Redis (not revoked)
        userID := claims.Subject
        sessionKey := fmt.Sprintf("session:%s", userID)
        exists, err := redisClient.Exists(c.Request().Context(), sessionKey).Result()
        if err != nil || exists == 0 {
            // Fallback: check DB
            var count int
            err = db.QueryRow("SELECT COUNT(*) FROM sessions WHERE user_id = $1 AND expires_at > now()", userID).Scan(&count)
            if err != nil || count == 0 {
                return echo.NewHTTPError(http.StatusUnauthorized, "session revoked")
            }
        }

        // 4. Load user from DB (or cache)
        user, err := userRepo.FindByID(c.Request().Context(), userID)
        if err != nil {
            return echo.NewHTTPError(http.StatusUnauthorized, "user not found")
        }

        // 5. Set user in Echo context
        c.Set("user", user)
        c.Set("user_id", user.ID)
        c.Set("user_role", user.Role)

        return next(c)
    }
}
```

### RequireRole Implementation

```go
func RequireRole(allowedRoles ...string) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            role, ok := c.Get("user_role").(string)
            if !ok {
                return echo.NewHTTPError(http.StatusForbidden, "access denied")
            }

            for _, allowed := range allowedRoles {
                if role == allowed {
                    return next(c)
                }
            }

            return echo.NewHTTPError(http.StatusForbidden, "insufficient permissions")
        }
    }
}
```

### RequireTeamRole Implementation

```go
func RequireTeamRole(allowedRoles ...string) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // 1. Get user from context
            userRole, _ := c.Get("user_role").(string)
            userID, _ := c.Get("user_id").(string)

            // 2. root bypasses all team role checks
            if userRole == "root" {
                return next(c)
            }

            // 3. Get team_id from URL parameter
            teamID := c.Param("team_id")
            if teamID == "" {
                return echo.NewHTTPError(http.StatusBadRequest, "team_id is required")
            }

            // 4. Query team membership
            var memberRole string
            var memberStatus string
            err := db.QueryRow(
                "SELECT role, status FROM team_members WHERE team_id = $1 AND user_id = $2",
                teamID, userID,
            ).Scan(&memberRole, &memberStatus)

            if err != nil {
                return echo.NewHTTPError(http.StatusForbidden, "not a member of this team")
            }

            // 5. Check membership is approved
            if memberStatus != "approved" {
                return echo.NewHTTPError(http.StatusForbidden, "team membership not approved")
            }

            // 6. Check role is in allowed roles
            for _, allowed := range allowedRoles {
                if memberRole == allowed {
                    c.Set("team_role", memberRole)
                    return next(c)
                }
            }

            return echo.NewHTTPError(http.StatusForbidden, "insufficient team permissions")
        }
    }
}
```

---

## 10. Security Considerations

### Password Hashing

- **Algorithm:** bcrypt
- **Cost factor:** 12, tuned to produce ~250ms hash time on modern hardware.
  This makes brute-force attacks impractical while keeping login latency
  acceptable.
- **Implementation:** Go standard library `golang.org/x/crypto/bcrypt`.
  Never use MD5, SHA-1, or plain SHA-256 for password storage.

### JWT Secret Management

- `JWT_SECRET` must be at least 32 characters (256 bits).
- Loaded from environment variables, never committed to source control.
- **Rotation strategy:** deploy a new secret and set `JWT_SECRET_OLD` to the
  previous value. The `AuthMiddleware` attempts validation with the primary
  secret first, then falls back to the old secret. After all existing tokens
  have expired (max 15 minutes), remove `JWT_SECRET_OLD`.

### Refresh Token Security

- **Rotation on every use:** each refresh produces a new token and invalidates
  the old one. This limits the window for stolen token abuse.
- **Server-side hash storage:** only the SHA-256 hash is stored. Even if the
  database is compromised, the raw tokens cannot be recovered.
- **Cookie scope:** `Path=/api/v1/auth` ensures the refresh token is only
  sent to auth endpoints, not to every API request.

### Cookie Security

All auth cookies use the following flags:

| Flag | Value | Purpose |
|------|-------|---------|
| `httpOnly` | `true` | Prevents JavaScript access, mitigating XSS token theft |
| `Secure` | `true` (production) | Ensures cookies are only sent over HTTPS |
| `SameSite` | `Strict` | Prevents the cookie from being sent in cross-site requests, mitigating CSRF |

### Rate Limiting

Redis-backed sliding window counters. Keys expire automatically.

| Endpoint Pattern | Limit | Window | Key |
|-----------------|-------|--------|-----|
| `POST /api/v1/auth/login` | 5 requests | 1 minute | `rate:login:{ip}` |
| `POST /api/v1/auth/signup` | 5 requests | 1 minute | `rate:signup:{ip}` |
| `POST /api/v1/auth/forgot-password` | 3 requests | 1 minute | `rate:forgot:{ip}` |
| `POST /api/v1/auth/refresh` | 10 requests | 1 minute | `rate:refresh:{ip}` |
| Authenticated API (general) | 300 requests | 1 minute | `rate:api:{user_id}` |
| Unauthenticated API (general) | 100 requests | 1 minute | `rate:api:{ip}` |

### Account Lockout

- **Threshold:** 10 consecutive failed login attempts.
- **Cooldown:** 30 minutes (1800 seconds).
- **Implementation:** Redis counter `lockout:{user_id}` with TTL of 1800
  seconds. The counter is incremented on each failed attempt and deleted on
  successful login.
- **Response:** `429 Too Many Requests` with `Retry-After: 1800` header.

### Password Reset Token Security

- **Generation:** 32 cryptographically random bytes via `crypto/rand`.
- **Storage:** SHA-256 hash only (never the raw token).
- **Single-use:** `used_at` column is set on first use. Any subsequent
  attempt with the same token is rejected.
- **Expiry:** 1 hour from generation.
- **Side effect:** all existing sessions for the user are invalidated after
  a successful password reset, forcing re-authentication.

### Timing-Safe Comparisons

- Refresh token hash lookup uses a database query (constant-time at the DB
  layer).
- Password reset token hash comparison uses `crypto/subtle.ConstantTimeCompare`
  after fetching the stored hash, preventing timing side-channel attacks.
- bcrypt comparison is inherently constant-time.

### Session Cleanup

A background goroutine runs every 6 hours to purge expired sessions:

```sql
DELETE FROM sessions WHERE expires_at < now();
```

```go
func StartSessionCleanup(db *sql.DB, interval time.Duration) {
    ticker := time.NewTicker(interval)
    go func() {
        for range ticker.C {
            _, err := db.Exec("DELETE FROM sessions WHERE expires_at < now()")
            if err != nil {
                log.Error().Err(err).Msg("session cleanup failed")
            }
        }
    }()
}
```

Similarly, expired password reset tokens are purged:

```sql
DELETE FROM password_resets WHERE expires_at < now() AND used_at IS NOT NULL;
```

### CORS Configuration

```go
e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
    AllowOrigins:     []string{os.Getenv("FRONTEND_URL")}, // e.g. "https://verdox.example.com"
    AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
    AllowHeaders:     []string{"Authorization", "Content-Type", "X-Request-Id"},
    AllowCredentials: true,
    MaxAge:           86400, // 24 hours
}))
```

---

## Appendix A: Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `JWT_SECRET` | Yes | -- | HMAC-SHA256 signing key. Minimum 32 characters |
| `JWT_SECRET_OLD` | No | -- | Previous signing key during rotation. Cleared after 15 minutes |
| `JWT_ACCESS_TTL` | No | `15m` | Access token lifetime (Go duration format) |
| `JWT_REFRESH_TTL` | No | `168h` | Refresh token / session lifetime (7 days) |
| `BCRYPT_COST` | No | `12` | bcrypt cost factor |
| `ROOT_EMAIL` | Yes | -- | Email for the root user, bootstrapped on first server startup |
| `ROOT_PASSWORD` | Yes | -- | Password for the root user, bootstrapped on first server startup |
| `FRONTEND_URL` | Yes | -- | Allowed CORS origin |
| `REDIS_URL` | Yes | -- | Redis connection string |
| `DATABASE_URL` | Yes | -- | PostgreSQL connection string |

---

## Appendix B: Error Response Format

All auth endpoints return errors in a consistent JSON structure:

```json
{
  "status": "error",
  "message": "human-readable error summary",
  "errors": [
    {
      "field": "password",
      "message": "must be at least 8 characters"
    }
  ]
}
```

- `status`: always `"error"` for error responses.
- `message`: a single-line summary suitable for display.
- `errors`: optional array of field-level validation errors. Only present for
  `400` responses with input validation failures.

---

## Appendix C: Redis Key Reference

| Key Pattern | Type | TTL | Purpose |
|-------------|------|-----|---------|
| `session:{user_id}` | STRING | 7 days | Maps user ID to active session ID for fast lookup |
| `lockout:{user_id}` | STRING (counter) | 30 min | Failed login attempt counter |
| `rate:login:{ip}` | STRING (counter) | 1 min | Login endpoint rate limit |
| `rate:signup:{ip}` | STRING (counter) | 1 min | Signup endpoint rate limit |
| `rate:forgot:{ip}` | STRING (counter) | 1 min | Forgot-password rate limit |
| `rate:refresh:{ip}` | STRING (counter) | 1 min | Token refresh rate limit |
| `rate:api:{user_id}` | STRING (counter) | 1 min | Authenticated API rate limit |
| `rate:api:{ip}` | STRING (counter) | 1 min | Unauthenticated API rate limit |

---

*End of document.*
