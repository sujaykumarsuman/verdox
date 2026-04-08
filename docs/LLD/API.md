# Verdox -- REST API Specification (LLD)

> Go / Echo v4 | JWT auth | PostgreSQL 17 | Redis 7

---

## 1. Conventions

### Base URL

```
/api/v1
```

### Content-Type

All requests and responses use `application/json` unless stated otherwise.

### Authentication

Authenticated endpoints require one of:

- **Authorization header:** `Bearer <access_token>`
- **httpOnly cookie:** `access_token=<jwt>` (set automatically on login)

Access tokens are JWTs (HS256, 15-minute expiry). Refresh tokens are stored as
httpOnly cookies with a 7-day expiry.

### Pagination

List endpoints accept these query parameters:

| Param      | Type | Default | Max | Description          |
|------------|------|---------|-----|----------------------|
| `page`     | int  | 1       | --  | Page number (1-based)|
| `per_page` | int  | 20      | 100 | Items per page       |

All paginated responses include a `meta` object:

```json
{
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 57,
    "total_pages": 3
  }
}
```

### Sorting

List endpoints accept:

| Param   | Type   | Default      | Description                        |
|---------|--------|--------------|------------------------------------|
| `sort`  | string | `created_at` | Column to sort by                  |
| `order` | string | `desc`       | Sort direction: `asc` or `desc`    |

### Error Response Format

All errors follow this structure:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Human-readable description of the error.",
    "details": {
      "email": "must be a valid email address",
      "password": "must be at least 8 characters"
    }
  }
}
```

The `details` field is present only for validation errors and contains
per-field messages.

### Common Error Codes

| HTTP Status | Code                  | Meaning                                        |
|-------------|-----------------------|------------------------------------------------|
| 400         | `VALIDATION_ERROR`    | Request body or query parameters are invalid   |
| 400         | `BAD_REQUEST`         | Malformed request                              |
| 401         | `UNAUTHORIZED`        | Missing or invalid authentication              |
| 401         | `TOKEN_EXPIRED`       | Access token has expired                       |
| 403         | `FORBIDDEN`           | Authenticated but insufficient permissions     |
| 404         | `NOT_FOUND`           | Resource does not exist                        |
| 409         | `CONFLICT`            | Resource already exists (duplicate)            |
| 422         | `UNPROCESSABLE`       | Semantically invalid request                   |
| 429         | `RATE_LIMITED`         | Too many requests                              |
| 500         | `INTERNAL_ERROR`      | Unexpected server error                        |

### Timestamps

All timestamps use ISO 8601 / RFC 3339 format:

```
2025-01-15T09:30:00Z
```

### UUIDs

All entity IDs are UUID v4:

```
550e8400-e29b-41d4-a716-446655440000
```

### Common Object Shapes

**User object** (never includes `password_hash`):

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "jdoe",
  "email": "jdoe@example.com",
  "role": "user",
  "avatar_url": "https://avatars.githubusercontent.com/u/12345",
  "created_at": "2025-01-15T09:30:00Z",
  "updated_at": "2025-01-15T09:30:00Z"
}
```

**Repository object:**

```json
{
  "id": "661f1a2b-3c4d-5e6f-7a8b-9c0d1e2f3a4b",
  "github_repo_id": 123456789,
  "github_full_name": "octocat/hello-world",
  "name": "hello-world",
  "description": "My first repository",
  "default_branch": "main",
  "is_active": true,
  "created_at": "2025-01-15T09:30:00Z",
  "updated_at": "2025-01-15T09:30:00Z"
}
```

---

## 2. Auth Endpoints

---

### POST /api/v1/auth/signup

**Description:** Register a new user account.

**Auth:** Public

**Rate Limit:** 5 req/min per IP

**Request:**

```
Headers:
  Content-Type: application/json

Body:
```

```json
{
  "username": "jdoe",
  "email": "jdoe@example.com",
  "password": "SecureP@ss1"
}
```

| Field      | Type   | Required | Constraints                                    |
|------------|--------|----------|------------------------------------------------|
| `username` | string | yes      | 3-64 chars, alphanumeric + underscores only    |
| `email`    | string | yes      | Valid email, max 255 chars                     |
| `password` | string | yes      | Min 8 chars, at least 1 upper, 1 lower, 1 digit |

**Response 201:**

```
Headers:
  Set-Cookie: refresh_token=<token>; HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth; Max-Age=604800
```

```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "jdoe",
    "email": "jdoe@example.com",
    "role": "user",
    "avatar_url": null,
    "created_at": "2025-01-15T09:30:00Z",
    "updated_at": "2025-01-15T09:30:00Z"
  },
  "access_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Errors:**

| Status | Code               | When                                      |
|--------|--------------------|-------------------------------------------|
| 400    | `VALIDATION_ERROR` | Missing/invalid fields                    |
| 409    | `CONFLICT`         | Username or email already exists          |
| 429    | `RATE_LIMITED`     | Exceeded 5 req/min                        |

---

### POST /api/v1/auth/login

**Description:** Authenticate and receive tokens.

**Auth:** Public

**Rate Limit:** 5 req/min per IP

**Request:**

```
Headers:
  Content-Type: application/json

Body:
```

```json
{
  "login": "jdoe@example.com",
  "password": "SecureP@ss1"
}
```

| Field      | Type   | Required | Constraints                       |
|------------|--------|----------|-----------------------------------|
| `login`    | string | yes      | Username or email address         |
| `password` | string | yes      | User's password                   |

**Response 200:**

```
Headers:
  Set-Cookie: refresh_token=<token>; HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth; Max-Age=604800
```

```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "jdoe",
    "email": "jdoe@example.com",
    "role": "user",
    "avatar_url": "https://avatars.githubusercontent.com/u/12345",
    "created_at": "2025-01-15T09:30:00Z",
    "updated_at": "2025-01-15T09:30:00Z"
  },
  "access_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Errors:**

| Status | Code               | When                                    |
|--------|--------------------|-----------------------------------------|
| 400    | `VALIDATION_ERROR` | Missing login or password field         |
| 401    | `UNAUTHORIZED`     | Invalid credentials                     |
| 429    | `RATE_LIMITED`     | Exceeded 5 req/min                      |

---

### POST /api/v1/auth/refresh

**Description:** Exchange a valid refresh token for a new access token and rotated refresh token.

**Auth:** Public (refresh token cookie required)

**Rate Limit:** 10 req/min per user

**Request:**

```
Headers:
  Cookie: refresh_token=<token>
```

No request body.

**Response 200:**

```
Headers:
  Set-Cookie: refresh_token=<new_token>; HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth; Max-Age=604800
```

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Errors:**

| Status | Code               | When                                    |
|--------|--------------------|-----------------------------------------|
| 401    | `UNAUTHORIZED`     | Missing, invalid, or expired refresh token |
| 429    | `RATE_LIMITED`     | Exceeded 10 req/min                     |

---

### POST /api/v1/auth/logout

**Description:** Invalidate the current session and clear tokens.

**Auth:** Authenticated

**Rate Limit:** 10 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>
  Cookie: refresh_token=<token>
```

No request body.

**Response 204:**

```
Headers:
  Set-Cookie: refresh_token=; HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth; Max-Age=0
```

No response body.

**Errors:**

| Status | Code               | When                                    |
|--------|--------------------|-----------------------------------------|
| 401    | `UNAUTHORIZED`     | Missing or invalid access token         |

---

### POST /api/v1/auth/forgot-password

**Description:** Request a password reset email. Always returns 200 regardless of whether the email exists (prevents user enumeration).

**Auth:** Public

**Rate Limit:** 3 req/min per IP

**Request:**

```
Headers:
  Content-Type: application/json

Body:
```

```json
{
  "email": "jdoe@example.com"
}
```

| Field   | Type   | Required | Constraints               |
|---------|--------|----------|---------------------------|
| `email` | string | yes      | Valid email, max 255 chars |

**Response 200:**

```json
{
  "message": "If an account with that email exists, a password reset link has been sent."
}
```

**Errors:**

| Status | Code               | When                                    |
|--------|--------------------|-----------------------------------------|
| 400    | `VALIDATION_ERROR` | Missing or invalid email field          |
| 429    | `RATE_LIMITED`     | Exceeded 3 req/min                      |

---

### POST /api/v1/auth/reset-password

**Description:** Set a new password using a reset token received via email.

**Auth:** Public (reset token required)

**Rate Limit:** 5 req/min per IP

**Request:**

```
Headers:
  Content-Type: application/json

Body:
```

```json
{
  "token": "a1b2c3d4e5f6...",
  "new_password": "NewSecureP@ss2"
}
```

| Field          | Type   | Required | Constraints                                    |
|----------------|--------|----------|------------------------------------------------|
| `token`        | string | yes      | Reset token from email link                    |
| `new_password` | string | yes      | Min 8 chars, at least 1 upper, 1 lower, 1 digit |

**Response 200:**

```json
{
  "message": "Password has been reset successfully. Please log in with your new password."
}
```

**Errors:**

| Status | Code               | When                                      |
|--------|--------------------|-------------------------------------------|
| 400    | `VALIDATION_ERROR` | Missing fields or weak password           |
| 401    | `UNAUTHORIZED`     | Invalid or expired reset token            |
| 429    | `RATE_LIMITED`     | Exceeded 5 req/min                        |

---

## 3. Repository Endpoints

---

### GET /api/v1/repositories

**Description:** List repositories accessible to the authenticated user (via team membership). Includes the test suite count and the status of the latest test run per repository.

**Auth:** Authenticated

**Rate Limit:** 30 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>

Query Parameters:
  page      (int, optional, default: 1)
  per_page  (int, optional, default: 20, max: 100)
  sort      (string, optional, default: "created_at", allowed: "created_at", "name", "updated_at")
  order     (string, optional, default: "desc", allowed: "asc", "desc")
  search    (string, optional) -- filter by name (case-insensitive partial match)
```

**Response 200:**

```json
{
  "data": [
    {
      "id": "661f1a2b-3c4d-5e6f-7a8b-9c0d1e2f3a4b",
      "github_repo_id": 123456789,
      "github_full_name": "octocat/hello-world",
      "name": "hello-world",
      "description": "My first repository",
      "default_branch": "main",
      "is_active": true,
      "suite_count": 3,
      "latest_run_status": "passed",
      "created_at": "2025-01-15T09:30:00Z",
      "updated_at": "2025-01-15T09:30:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 5,
    "total_pages": 1
  }
}
```

**Errors:**

| Status | Code               | When                                    |
|--------|--------------------|-----------------------------------------|
| 400    | `VALIDATION_ERROR` | Invalid query parameters                |
| 401    | `UNAUTHORIZED`     | Missing or invalid access token         |

---

### POST /api/v1/repositories

**Description:** Add a repository by GitHub URL. Validates the repo exists via the team's GitHub PAT (resolved from team_id), creates a repository record, and enqueues a `repo.clone` worker job to clone it to local storage. Only root, moderator, or team admin can add repos.

**Auth:** Authenticated | Role: root, moderator OR Team Role: admin

**Rate Limit:** 5 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>

Body:
```

```json
{
  "github_url": "https://github.com/hashicorp/consul",
  "team_id": "aaa11111-2222-3333-4444-555566667777"
}
```

| Field        | Type   | Required | Constraints                        |
|--------------|--------|----------|------------------------------------|
| `github_url` | string | yes      | Valid GitHub repo URL               |
| `team_id`    | UUID   | yes      | Team to assign the repo to         |

**Response 201:**

```json
{
  "data": {
    "id": "661f1a2b-3c4d-5e6f-7a8b-9c0d1e2f3a4b",
    "github_repo_id": 123456789,
    "github_full_name": "hashicorp/consul",
    "name": "consul",
    "description": "Consul is a distributed service mesh",
    "default_branch": "main",
    "is_active": true,
    "fork_full_name": null,
    "fork_status": "none",
    "fork_synced_at": null,
    "fork_workflow_id": null,
    "fork_head_sha": null,
    "created_at": "2025-01-15T09:30:00Z",
    "updated_at": "2025-01-15T09:30:00Z"
  }
}
```

**Errors:**

| Status | Code               | When                                      |
|--------|--------------------|-------------------------------------------|
| 400    | `VALIDATION_ERROR` | Invalid URL format or missing team_id      |
| 401    | `UNAUTHORIZED`     | Missing or invalid access token           |
| 403    | `FORBIDDEN`        | User is not root/moderator/team admin     |
| 404    | `NOT_FOUND`        | Repo not found on GitHub or no access     |
| 409    | `CONFLICT`         | Repository already added                  |
| 422    | `UNPROCESSABLE`    | Team GitHub PAT not configured or revoked |
| 502    | `BAD_GATEWAY`      | GitHub API unreachable                    |

---

### POST /api/v1/repositories/:id/resync

**Description:** Re-fetch latest data from the remote repository (git fetch --all --prune). Uses the team's GitHub PAT for authentication. Updates branch and commit data.

**Auth:** Authenticated (must be team member with access)

**Rate Limit:** 10 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>

Path Parameters:
  id  (UUID, required) -- Repository ID
```

**Response 200:**

```json
{
  "data": {
    "message": "Repository resynced successfully",
    "default_branch": "main",
    "branch_count": 5
  }
}
```

**Errors:**

| Status | Code               | When                                      |
|--------|--------------------|-------------------------------------------|
| 401    | `UNAUTHORIZED`     | Missing or invalid access token           |
| 404    | `NOT_FOUND`        | Repository not found or not accessible    |
| 422    | `UNPROCESSABLE`    | Repository not in a valid state for resync|

---

### GET /api/v1/repositories/:id

**Description:** Get detailed information for a single repository, including its test suites and recent test runs.

**Auth:** Authenticated (must be team member with access)

**Rate Limit:** 30 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>

Path Parameters:
  id  (UUID, required) -- Repository ID
```

**Response 200:**

```json
{
  "data": {
    "id": "661f1a2b-3c4d-5e6f-7a8b-9c0d1e2f3a4b",
    "github_repo_id": 123456789,
    "github_full_name": "octocat/hello-world",
    "name": "hello-world",
    "description": "My first repository",
    "default_branch": "main",
    "is_active": true,
    "created_at": "2025-01-15T09:30:00Z",
    "updated_at": "2025-01-15T09:30:00Z",
    "suites": [
      {
        "id": "aaa11111-2222-3333-4444-555566667777",
        "name": "Unit Tests",
        "type": "unit",
        "config_path": ".verdox/unit.yml",
        "timeout_seconds": 300,
        "created_at": "2025-01-16T08:00:00Z",
        "updated_at": "2025-01-16T08:00:00Z",
        "latest_run": {
          "id": "bbb22222-3333-4444-5555-666677778888",
          "branch": "main",
          "commit_hash": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
          "status": "passed",
          "started_at": "2025-01-20T14:00:00Z",
          "finished_at": "2025-01-20T14:02:30Z",
          "created_at": "2025-01-20T14:00:00Z"
        }
      },
      {
        "id": "ccc33333-4444-5555-6666-777788889999",
        "name": "Integration Tests",
        "type": "integration",
        "config_path": ".verdox/integration.yml",
        "timeout_seconds": 600,
        "created_at": "2025-01-16T08:30:00Z",
        "updated_at": "2025-01-16T08:30:00Z",
        "latest_run": null
      }
    ],
    "recent_runs": [
      {
        "id": "bbb22222-3333-4444-5555-666677778888",
        "test_suite_id": "aaa11111-2222-3333-4444-555566667777",
        "suite_name": "Unit Tests",
        "branch": "main",
        "commit_hash": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
        "status": "passed",
        "started_at": "2025-01-20T14:00:00Z",
        "finished_at": "2025-01-20T14:02:30Z",
        "created_at": "2025-01-20T14:00:00Z"
      }
    ]
  }
}
```

**Errors:**

| Status | Code           | When                                      |
|--------|----------------|-------------------------------------------|
| 401    | `UNAUTHORIZED` | Missing or invalid access token           |
| 403    | `FORBIDDEN`    | User does not have team access             |
| 404    | `NOT_FOUND`    | Repository does not exist                 |

---

### DELETE /api/v1/repositories/:id

**Description:** Deactivate a repository (soft delete). Sets `is_active = false`. Does not delete database records.

**Auth:** Authenticated (must be team admin)

**Rate Limit:** 10 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>

Path Parameters:
  id  (UUID, required) -- Repository ID
```

**Response 200:**

```json
{
  "data": {
    "id": "661f1a2b-3c4d-5e6f-7a8b-9c0d1e2f3a4b",
    "is_active": false,
    "message": "Repository has been deactivated."
  }
}
```

**Errors:**

| Status | Code           | When                                      |
|--------|----------------|-------------------------------------------|
| 401    | `UNAUTHORIZED` | Missing or invalid access token           |
| 403    | `FORBIDDEN`    | User is not team admin                    |
| 404    | `NOT_FOUND`    | Repository does not exist                 |

---

### GET /api/v1/repositories/:id/branches

**Description:** List branches for a repository. Fetched from the GitHub API and cached in Redis (TTL: 5 minutes).

**Auth:** Authenticated (must be team member with access)

**Rate Limit:** 20 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>

Path Parameters:
  id  (UUID, required) -- Repository ID
```

**Response 200:**

```json
{
  "data": [
    {
      "name": "main",
      "is_default": true,
      "commit_sha": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
    },
    {
      "name": "develop",
      "is_default": false,
      "commit_sha": "f6e5d4c3b2a1f6e5d4c3b2a1f6e5d4c3b2a1f6e5"
    },
    {
      "name": "feature/auth",
      "is_default": false,
      "commit_sha": "1a2b3c4d5e6f1a2b3c4d5e6f1a2b3c4d5e6f1a2b"
    }
  ]
}
```

**Errors:**

| Status | Code             | When                                        |
|--------|------------------|---------------------------------------------|
| 401    | `UNAUTHORIZED`   | Missing or invalid access token             |
| 403    | `FORBIDDEN`      | User does not have team access               |
| 404    | `NOT_FOUND`      | Repository does not exist                   |
| 502    | `BAD_GATEWAY`    | GitHub API unreachable or returned error    |

---

### GET /api/v1/repositories/:id/commits

**Description:** List recent commits for a specific branch. Fetched from the GitHub API and cached in Redis (TTL: 2 minutes).

**Auth:** Authenticated (must be team member with access)

**Rate Limit:** 20 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>

Path Parameters:
  id  (UUID, required) -- Repository ID

Query Parameters:
  branch    (string, required) -- Branch name (e.g., "main")
  page      (int, optional, default: 1)
  per_page  (int, optional, default: 20, max: 100)
```

**Response 200:**

```json
{
  "data": [
    {
      "sha": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
      "message": "feat: add user authentication",
      "author": {
        "name": "John Doe",
        "email": "jdoe@example.com",
        "date": "2025-01-20T14:00:00Z"
      },
      "url": "https://github.com/octocat/hello-world/commit/a1b2c3d4"
    },
    {
      "sha": "f6e5d4c3b2a1f6e5d4c3b2a1f6e5d4c3b2a1f6e5",
      "message": "fix: resolve login redirect issue",
      "author": {
        "name": "Jane Smith",
        "email": "jane@example.com",
        "date": "2025-01-19T16:30:00Z"
      },
      "url": "https://github.com/octocat/hello-world/commit/f6e5d4c3"
    }
  ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 42,
    "total_pages": 3
  }
}
```

**Errors:**

| Status | Code               | When                                      |
|--------|--------------------|-------------------------------------------|
| 400    | `VALIDATION_ERROR` | Missing `branch` query parameter          |
| 401    | `UNAUTHORIZED`     | Missing or invalid access token           |
| 403    | `FORBIDDEN`        | User does not have team access            |
| 404    | `NOT_FOUND`        | Repository or branch does not exist       |
| 502    | `BAD_GATEWAY`      | GitHub API unreachable or returned error  |

---

## 4. Test Suite Endpoints

---

### GET /api/v1/repositories/:id/suites

**Description:** List all test suites for a repository.

**Auth:** Authenticated (must be team member with access)

**Rate Limit:** 30 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>

Path Parameters:
  id  (UUID, required) -- Repository ID

Query Parameters:
  type  (string, optional) -- Filter by type: "unit" or "integration"
```

**Response 200:**

```json
{
  "data": [
    {
      "id": "aaa11111-2222-3333-4444-555566667777",
      "repository_id": "661f1a2b-3c4d-5e6f-7a8b-9c0d1e2f3a4b",
      "name": "Unit Tests",
      "type": "unit",
      "config_path": ".verdox/unit.yml",
      "timeout_seconds": 300,
      "created_at": "2025-01-16T08:00:00Z",
      "updated_at": "2025-01-16T08:00:00Z"
    },
    {
      "id": "ccc33333-4444-5555-6666-777788889999",
      "repository_id": "661f1a2b-3c4d-5e6f-7a8b-9c0d1e2f3a4b",
      "name": "Integration Tests",
      "type": "integration",
      "config_path": ".verdox/integration.yml",
      "timeout_seconds": 600,
      "created_at": "2025-01-16T08:30:00Z",
      "updated_at": "2025-01-16T08:30:00Z"
    }
  ]
}
```

**Errors:**

| Status | Code               | When                                    |
|--------|--------------------|-----------------------------------------|
| 400    | `VALIDATION_ERROR` | Invalid `type` filter value             |
| 401    | `UNAUTHORIZED`     | Missing or invalid access token         |
| 403    | `FORBIDDEN`        | User does not own or have team access   |
| 404    | `NOT_FOUND`        | Repository does not exist               |

---

### POST /api/v1/repositories/:id/suites

**Description:** Create a new test suite for a repository.

**Auth:** Authenticated (must be team admin/maintainer)

**Rate Limit:** 10 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>
  Content-Type: application/json

Path Parameters:
  id  (UUID, required) -- Repository ID

Body:
```

```json
{
  "name": "Unit Tests",
  "type": "unit",
  "config_path": ".verdox/unit.yml",
  "timeout_seconds": 300
}
```

| Field             | Type   | Required | Constraints                                    |
|-------------------|--------|----------|------------------------------------------------|
| `name`            | string | yes      | 1-255 chars                                    |
| `type`            | string | yes      | One of: `unit`, `integration`                  |
| `config_path`     | string | no       | Relative path to config file in repo           |
| `timeout_seconds` | int    | no       | Default: 300. Min: 30, max: 3600               |

**Response 201:**

```json
{
  "data": {
    "id": "aaa11111-2222-3333-4444-555566667777",
    "repository_id": "661f1a2b-3c4d-5e6f-7a8b-9c0d1e2f3a4b",
    "name": "Unit Tests",
    "type": "unit",
    "config_path": ".verdox/unit.yml",
    "timeout_seconds": 300,
    "created_at": "2025-01-16T08:00:00Z",
    "updated_at": "2025-01-16T08:00:00Z"
  }
}
```

**Errors:**

| Status | Code               | When                                      |
|--------|--------------------|-------------------------------------------|
| 400    | `VALIDATION_ERROR` | Missing or invalid fields                 |
| 401    | `UNAUTHORIZED`     | Missing or invalid access token           |
| 403    | `FORBIDDEN`        | User is not team admin/maintainer         |
| 404    | `NOT_FOUND`        | Repository does not exist                 |

---

### PUT /api/v1/suites/:id

**Description:** Update an existing test suite's configuration.

**Auth:** Authenticated (must be team admin/maintainer)

**Rate Limit:** 10 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>
  Content-Type: application/json

Path Parameters:
  id  (UUID, required) -- Test Suite ID

Body:
```

```json
{
  "name": "Unit Tests (Updated)",
  "type": "unit",
  "config_path": ".verdox/unit-v2.yml",
  "timeout_seconds": 600
}
```

| Field             | Type   | Required | Constraints                                    |
|-------------------|--------|----------|------------------------------------------------|
| `name`            | string | no       | 1-255 chars                                    |
| `type`            | string | no       | One of: `unit`, `integration`                  |
| `config_path`     | string | no       | Relative path to config file in repo           |
| `timeout_seconds` | int    | no       | Min: 30, max: 3600                             |

All fields are optional; only provided fields are updated.

**Response 200:**

```json
{
  "data": {
    "id": "aaa11111-2222-3333-4444-555566667777",
    "repository_id": "661f1a2b-3c4d-5e6f-7a8b-9c0d1e2f3a4b",
    "name": "Unit Tests (Updated)",
    "type": "unit",
    "config_path": ".verdox/unit-v2.yml",
    "timeout_seconds": 600,
    "created_at": "2025-01-16T08:00:00Z",
    "updated_at": "2025-01-20T12:00:00Z"
  }
}
```

**Errors:**

| Status | Code               | When                                      |
|--------|--------------------|-------------------------------------------|
| 400    | `VALIDATION_ERROR` | Invalid field values                      |
| 401    | `UNAUTHORIZED`     | Missing or invalid access token           |
| 403    | `FORBIDDEN`        | User is not team admin/maintainer         |
| 404    | `NOT_FOUND`        | Test suite does not exist                 |

---

### DELETE /api/v1/suites/:id

**Description:** Delete a test suite and all its associated test runs and results (cascade delete).

**Auth:** Authenticated (must be team admin)

**Rate Limit:** 10 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>

Path Parameters:
  id  (UUID, required) -- Test Suite ID
```

**Response 204:**

No response body.

**Errors:**

| Status | Code           | When                                       |
|--------|----------------|--------------------------------------------|
| 401    | `UNAUTHORIZED` | Missing or invalid access token            |
| 403    | `FORBIDDEN`    | User is not team admin                     |
| 404    | `NOT_FOUND`    | Test suite does not exist                  |

---

## 5. Test Run Endpoints

---

### POST /api/v1/suites/:id/run

**Description:** Trigger a new test run for a suite. Creates a run with `status: queued` and pushes a job onto the Redis queue. Returns immediately.

**Auth:** Authenticated (must be team member with access)

**Rate Limit:** 10 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>
  Content-Type: application/json

Path Parameters:
  id  (UUID, required) -- Test Suite ID

Body:
```

```json
{
  "branch": "main",
  "commit_hash": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
}
```

| Field         | Type   | Required | Constraints                          |
|---------------|--------|----------|--------------------------------------|
| `branch`      | string | yes      | 1-255 chars, valid branch name       |
| `commit_hash` | string | yes      | Exactly 40 hex characters (SHA-1)    |

**Response 201:**

```json
{
  "data": {
    "id": "ddd44444-5555-6666-7777-888899990000",
    "test_suite_id": "aaa11111-2222-3333-4444-555566667777",
    "triggered_by": "550e8400-e29b-41d4-a716-446655440000",
    "branch": "main",
    "commit_hash": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
    "status": "queued",
    "started_at": null,
    "finished_at": null,
    "created_at": "2025-01-20T14:00:00Z"
  }
}
```

**Errors:**

| Status | Code               | When                                      |
|--------|--------------------|-------------------------------------------|
| 400    | `VALIDATION_ERROR` | Missing or invalid fields                 |
| 401    | `UNAUTHORIZED`     | Missing or invalid access token           |
| 403    | `FORBIDDEN`        | User does not have access to the suite    |
| 404    | `NOT_FOUND`        | Test suite does not exist                 |
| 409    | `CONFLICT`         | A run is already queued/running for this suite+branch+commit |

---

### GET /api/v1/suites/:id/runs

**Description:** List test runs for a specific test suite, ordered newest first.

**Auth:** Authenticated (must be team member with access)

**Rate Limit:** 30 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>

Path Parameters:
  id  (UUID, required) -- Test Suite ID

Query Parameters:
  page      (int, optional, default: 1)
  per_page  (int, optional, default: 20, max: 100)
  status    (string, optional) -- Filter by status: "queued", "running", "passed", "failed", "cancelled"
  branch    (string, optional) -- Filter by branch name
```

**Response 200:**

```json
{
  "data": [
    {
      "id": "ddd44444-5555-6666-7777-888899990000",
      "test_suite_id": "aaa11111-2222-3333-4444-555566667777",
      "triggered_by": "550e8400-e29b-41d4-a716-446655440000",
      "triggered_by_username": "jdoe",
      "branch": "main",
      "commit_hash": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
      "status": "passed",
      "started_at": "2025-01-20T14:00:05Z",
      "finished_at": "2025-01-20T14:02:30Z",
      "created_at": "2025-01-20T14:00:00Z"
    },
    {
      "id": "eee55555-6666-7777-8888-999900001111",
      "test_suite_id": "aaa11111-2222-3333-4444-555566667777",
      "triggered_by": "550e8400-e29b-41d4-a716-446655440000",
      "triggered_by_username": "jdoe",
      "branch": "develop",
      "commit_hash": "f6e5d4c3b2a1f6e5d4c3b2a1f6e5d4c3b2a1f6e5",
      "status": "failed",
      "started_at": "2025-01-19T10:00:03Z",
      "finished_at": "2025-01-19T10:01:45Z",
      "created_at": "2025-01-19T10:00:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 12,
    "total_pages": 1
  }
}
```

**Errors:**

| Status | Code               | When                                    |
|--------|--------------------|-----------------------------------------|
| 400    | `VALIDATION_ERROR` | Invalid query parameters                |
| 401    | `UNAUTHORIZED`     | Missing or invalid access token         |
| 403    | `FORBIDDEN`        | User does not have access to the suite  |
| 404    | `NOT_FOUND`        | Test suite does not exist               |

---

### GET /api/v1/runs/:id

**Description:** Get full details of a test run including all individual test results.

**Auth:** Authenticated (must have access to the associated repository)

**Rate Limit:** 30 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>

Path Parameters:
  id  (UUID, required) -- Test Run ID
```

**Response 200:**

```json
{
  "data": {
    "id": "ddd44444-5555-6666-7777-888899990000",
    "test_suite_id": "aaa11111-2222-3333-4444-555566667777",
    "suite_name": "Unit Tests",
    "suite_type": "unit",
    "repository_id": "661f1a2b-3c4d-5e6f-7a8b-9c0d1e2f3a4b",
    "repository_name": "hello-world",
    "triggered_by": "550e8400-e29b-41d4-a716-446655440000",
    "triggered_by_username": "jdoe",
    "branch": "main",
    "commit_hash": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
    "status": "passed",
    "started_at": "2025-01-20T14:00:05Z",
    "finished_at": "2025-01-20T14:02:30Z",
    "created_at": "2025-01-20T14:00:00Z",
    "summary": {
      "total": 15,
      "passed": 13,
      "failed": 1,
      "skipped": 1,
      "errors": 0,
      "duration_ms": 145000
    },
    "results": [
      {
        "id": "fff66666-7777-8888-9999-000011112222",
        "test_name": "TestUserLogin",
        "status": "pass",
        "duration_ms": 120,
        "error_message": null,
        "created_at": "2025-01-20T14:02:30Z"
      },
      {
        "id": "ggg77777-8888-9999-0000-111122223333",
        "test_name": "TestUserSignup",
        "status": "fail",
        "duration_ms": 340,
        "error_message": "expected status 201, got 400: validation error on field 'email'",
        "created_at": "2025-01-20T14:02:30Z"
      },
      {
        "id": "hhh88888-9999-0000-1111-222233334444",
        "test_name": "TestUserDelete",
        "status": "skip",
        "duration_ms": 0,
        "error_message": null,
        "created_at": "2025-01-20T14:02:30Z"
      }
    ]
  }
}
```

**Errors:**

| Status | Code           | When                                      |
|--------|----------------|-------------------------------------------|
| 401    | `UNAUTHORIZED` | Missing or invalid access token           |
| 403    | `FORBIDDEN`    | User does not have access to the repo     |
| 404    | `NOT_FOUND`    | Test run does not exist                   |

---

### GET /api/v1/runs/:id/logs

**Description:** Get log output for a test run. Returns aggregated logs or per-test logs depending on query parameters.

**Auth:** Authenticated (must have access to the associated repository)

**Rate Limit:** 20 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>

Path Parameters:
  id  (UUID, required) -- Test Run ID

Query Parameters:
  test_name  (string, optional) -- Filter logs for a specific test by name
```

**Response 200 (aggregated, no test_name filter):**

```json
{
  "data": {
    "run_id": "ddd44444-5555-6666-7777-888899990000",
    "logs": [
      {
        "test_name": "TestUserLogin",
        "status": "pass",
        "duration_ms": 120,
        "log_output": "=== RUN   TestUserLogin\n--- PASS: TestUserLogin (0.12s)\n"
      },
      {
        "test_name": "TestUserSignup",
        "status": "fail",
        "duration_ms": 340,
        "log_output": "=== RUN   TestUserSignup\n    user_test.go:45: expected status 201, got 400\n--- FAIL: TestUserSignup (0.34s)\n"
      }
    ]
  }
}
```

**Response 200 (single test, with test_name filter):**

```json
{
  "data": {
    "run_id": "ddd44444-5555-6666-7777-888899990000",
    "test_name": "TestUserSignup",
    "status": "fail",
    "duration_ms": 340,
    "log_output": "=== RUN   TestUserSignup\n    user_test.go:45: expected status 201, got 400\n    user_test.go:46: response body: {\"error\":{\"code\":\"VALIDATION_ERROR\"}}\n--- FAIL: TestUserSignup (0.34s)\n"
  }
}
```

**Errors:**

| Status | Code           | When                                      |
|--------|----------------|-------------------------------------------|
| 401    | `UNAUTHORIZED` | Missing or invalid access token           |
| 403    | `FORBIDDEN`    | User does not have access to the repo     |
| 404    | `NOT_FOUND`    | Test run or test name does not exist      |

---

### POST /api/v1/runs/:id/cancel

**Description:** Cancel a test run that is currently queued or running. Sets the status to `cancelled`.

**Auth:** Authenticated (must be the user who triggered the run, or team admin)

**Rate Limit:** 10 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>

Path Parameters:
  id  (UUID, required) -- Test Run ID
```

No request body.

**Response 200:**

```json
{
  "data": {
    "id": "ddd44444-5555-6666-7777-888899990000",
    "status": "cancelled",
    "message": "Test run has been cancelled."
  }
}
```

**Errors:**

| Status | Code               | When                                        |
|--------|--------------------|---------------------------------------------|
| 401    | `UNAUTHORIZED`     | Missing or invalid access token             |
| 403    | `FORBIDDEN`        | User did not trigger the run and lacks admin access |
| 404    | `NOT_FOUND`        | Test run does not exist                     |
| 409    | `CONFLICT`         | Run is already in a terminal state (passed, failed, cancelled) |

---

### POST /api/v1/repositories/:id/run-all

**Description:** Trigger test runs for all active test suites in a repository on the specified branch and commit. Creates one run per suite and enqueues each.

**Auth:** Authenticated (must be team member with access)

**Rate Limit:** 5 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>
  Content-Type: application/json

Path Parameters:
  id  (UUID, required) -- Repository ID

Body:
```

```json
{
  "branch": "main",
  "commit_hash": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
}
```

| Field         | Type   | Required | Constraints                          |
|---------------|--------|----------|--------------------------------------|
| `branch`      | string | yes      | 1-255 chars, valid branch name       |
| `commit_hash` | string | yes      | Exactly 40 hex characters (SHA-1)    |

**Response 201:**

```json
{
  "data": {
    "repository_id": "661f1a2b-3c4d-5e6f-7a8b-9c0d1e2f3a4b",
    "branch": "main",
    "commit_hash": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
    "runs": [
      {
        "id": "ddd44444-5555-6666-7777-888899990000",
        "test_suite_id": "aaa11111-2222-3333-4444-555566667777",
        "suite_name": "Unit Tests",
        "status": "queued",
        "created_at": "2025-01-20T14:00:00Z"
      },
      {
        "id": "eee55555-6666-7777-8888-999900001111",
        "test_suite_id": "ccc33333-4444-5555-6666-777788889999",
        "suite_name": "Integration Tests",
        "status": "queued",
        "created_at": "2025-01-20T14:00:00Z"
      }
    ],
    "total_queued": 2
  }
}
```

**Errors:**

| Status | Code               | When                                      |
|--------|--------------------|-------------------------------------------|
| 400    | `VALIDATION_ERROR` | Missing or invalid fields                 |
| 401    | `UNAUTHORIZED`     | Missing or invalid access token           |
| 403    | `FORBIDDEN`        | User does not have access                 |
| 404    | `NOT_FOUND`        | Repository does not exist                 |
| 422    | `UNPROCESSABLE`    | Repository has no test suites configured  |

---

## 6. Team Endpoints

---

### GET /api/v1/teams

**Description:** List all teams the authenticated user belongs to (as a member with status `approved`).

**Auth:** Authenticated

**Rate Limit:** 30 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>

Query Parameters:
  page      (int, optional, default: 1)
  per_page  (int, optional, default: 20, max: 100)
  sort      (string, optional, default: "created_at", allowed: "created_at", "name")
  order     (string, optional, default: "desc", allowed: "asc", "desc")
```

**Response 200:**

```json
{
  "data": [
    {
      "id": "111aaaaa-bbbb-cccc-dddd-eeee11112222",
      "name": "Consul Team",
      "slug": "consul-team",
      "created_by": "550e8400-e29b-41d4-a716-446655440000",
      "my_role": "admin",
      "member_count": 5,
      "repo_count": 3,
      "created_at": "2025-01-10T08:00:00Z",
      "updated_at": "2025-01-10T08:00:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 2,
    "total_pages": 1
  }
}
```

**Errors:**

| Status | Code           | When                                    |
|--------|----------------|-----------------------------------------|
| 401    | `UNAUTHORIZED` | Missing or invalid access token         |

---

### POST /api/v1/teams

**Description:** Create a new team. The creator is automatically added as team admin with `approved` status. The slug is auto-generated from the name.

**Auth:** Authenticated

**Rate Limit:** 10 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>
  Content-Type: application/json

Body:
```

```json
{
  "name": "Consul Team"
}
```

| Field  | Type   | Required | Constraints                             |
|--------|--------|----------|-----------------------------------------|
| `name` | string | yes      | 2-128 chars, must be unique             |

**Response 201:**

```json
{
  "data": {
    "id": "111aaaaa-bbbb-cccc-dddd-eeee11112222",
    "name": "Consul Team",
    "slug": "consul-team",
    "created_by": "550e8400-e29b-41d4-a716-446655440000",
    "created_at": "2025-01-10T08:00:00Z",
    "updated_at": "2025-01-10T08:00:00Z"
  }
}
```

**Errors:**

| Status | Code               | When                                    |
|--------|--------------------|-----------------------------------------|
| 400    | `VALIDATION_ERROR` | Missing or invalid name                 |
| 401    | `UNAUTHORIZED`     | Missing or invalid access token         |
| 409    | `CONFLICT`         | Team name or generated slug already exists |

---

### GET /api/v1/teams/:id

**Description:** Get team details including members and assigned repositories.

**Auth:** Authenticated (must be team member or root/moderator)

**Rate Limit:** 30 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>

Path Parameters:
  id  (UUID, required) -- Team ID
```

**Response 200:**

```json
{
  "data": {
    "id": "111aaaaa-bbbb-cccc-dddd-eeee11112222",
    "name": "Consul Team",
    "slug": "consul-team",
    "created_by": "550e8400-e29b-41d4-a716-446655440000",
    "created_at": "2025-01-10T08:00:00Z",
    "updated_at": "2025-01-10T08:00:00Z",
    "members": [
      {
        "id": "mmm11111-2222-3333-4444-555566667777",
        "user_id": "550e8400-e29b-41d4-a716-446655440000",
        "username": "jdoe",
        "email": "jdoe@example.com",
        "avatar_url": "https://avatars.githubusercontent.com/u/12345",
        "role": "admin",
        "status": "approved",
        "invited_by": null,
        "created_at": "2025-01-10T08:00:00Z"
      },
      {
        "id": "mmm22222-3333-4444-5555-666677778888",
        "user_id": "660e8400-e29b-41d4-a716-446655440001",
        "username": "jane",
        "email": "jane@example.com",
        "avatar_url": null,
        "role": "viewer",
        "status": "pending",
        "invited_by": "550e8400-e29b-41d4-a716-446655440000",
        "created_at": "2025-01-11T10:00:00Z"
      }
    ],
    "repositories": [
      {
        "id": "661f1a2b-3c4d-5e6f-7a8b-9c0d1e2f3a4b",
        "github_full_name": "octocat/hello-world",
        "name": "hello-world",
        "is_active": true,
        "added_by": "550e8400-e29b-41d4-a716-446655440000",
        "added_at": "2025-01-12T09:00:00Z"
      }
    ]
  }
}
```

**Errors:**

| Status | Code           | When                                        |
|--------|----------------|---------------------------------------------|
| 401    | `UNAUTHORIZED` | Missing or invalid access token             |
| 403    | `FORBIDDEN`    | User is not a team member or platform moderator |
| 404    | `NOT_FOUND`    | Team does not exist                         |

---

### PUT /api/v1/teams/:id

**Description:** Update team details.

**Auth:** Authenticated | Team Role: admin

**Rate Limit:** 10 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>
  Content-Type: application/json

Path Parameters:
  id  (UUID, required) -- Team ID

Body:
```

```json
{
  "name": "Consul Team (Renamed)"
}
```

| Field  | Type   | Required | Constraints                             |
|--------|--------|----------|-----------------------------------------|
| `name` | string | yes      | 2-128 chars                             |

**Response 200:**

```json
{
  "data": {
    "id": "111aaaaa-bbbb-cccc-dddd-eeee11112222",
    "name": "Consul Team (Renamed)",
    "slug": "consul-team-renamed",
    "created_by": "550e8400-e29b-41d4-a716-446655440000",
    "created_at": "2025-01-10T08:00:00Z",
    "updated_at": "2025-01-20T16:00:00Z"
  }
}
```

**Errors:**

| Status | Code               | When                                    |
|--------|--------------------|-----------------------------------------|
| 400    | `VALIDATION_ERROR` | Missing or invalid name                 |
| 401    | `UNAUTHORIZED`     | Missing or invalid access token         |
| 403    | `FORBIDDEN`        | User is not team admin                  |
| 404    | `NOT_FOUND`        | Team does not exist                     |
| 409    | `CONFLICT`         | New name or generated slug already exists |

---

### DELETE /api/v1/teams/:id

**Description:** Soft-delete a team with cascade cleanup. This performs the
following steps in order:

1. Soft-deletes the team by setting `deleted_at` to the current timestamp.
2. Marks all repositories linked via `team_repositories` as inactive
   (`is_active = false` on the `repositories` table).
3. Queues an async background job (`verdox:jobs:cleanup`) to delete local
   repository clones from disk. Disk cleanup is eventual, not immediate.
4. Removes all team memberships (deletes rows from `team_members`).
5. Returns 200 with a confirmation message.

> **Note:** This operation is irreversible for team data (the team cannot be
> restored after soft-delete). Repository data on disk is cleaned up
> asynchronously by the background worker. The `team_repositories` junction
> rows are cascade-deleted by the database when the team is soft-deleted.

**Auth:** Authenticated | Team Role: admin OR Role: root

**Rate Limit:** 5 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>

Path Parameters:
  id  (UUID, required) -- Team ID
```

**Response 200:**

```json
{
  "message": "Team deleted"
}
```

**Errors:**

| Status | Code           | When                                        |
|--------|----------------|---------------------------------------------|
| 401    | `UNAUTHORIZED` | Missing or invalid access token             |
| 403    | `FORBIDDEN`    | User is not team admin or root       |
| 404    | `NOT_FOUND`    | Team does not exist                         |

---

### PUT /api/v1/teams/:id/pat

**Description:** Set or update the team's GitHub Personal Access Token. The PAT is validated against the GitHub API before storing. Encrypted with AES-256-GCM and stored on the `teams` table.

**Auth:** Authenticated | Team Role: admin

**Rate Limit:** 5 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>
  Content-Type: application/json

Path Parameters:
  id  (UUID, required) -- Team ID

Body:
```

```json
{
  "pat": "github_pat_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
}
```

| Field | Type   | Required | Constraints                           |
|-------|--------|----------|---------------------------------------|
| `pat` | string | yes      | Valid GitHub PAT string               |

**Response 200:**

```json
{
  "data": {
    "github_username": "acme-bot",
    "set_at": "2025-01-15T10:30:00Z",
    "set_by": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "jdoe"
    }
  }
}
```

**Errors:**

| Status | Code               | When                                        |
|--------|--------------------|---------------------------------------------|
| 400    | `INVALID_PAT`      | GitHub API rejects the token (invalid/expired) |
| 401    | `UNAUTHORIZED`     | Missing or invalid access token             |
| 403    | `FORBIDDEN`        | User is not team admin                      |
| 404    | `NOT_FOUND`        | Team does not exist                         |
| 422    | `VALIDATION_ERROR` | Missing pat field                           |

---

### GET /api/v1/teams/:id/pat/status

**Description:** Check the team's GitHub PAT configuration status. Does NOT return the actual PAT value.

**Auth:** Authenticated | Team Role: admin, maintainer

**Rate Limit:** 10 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>

Path Parameters:
  id  (UUID, required) -- Team ID
```

**Response 200 (PAT configured):**

```json
{
  "data": {
    "configured": true,
    "github_username": "acme-bot",
    "set_at": "2025-01-15T10:30:00Z",
    "set_by": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "jdoe"
    },
    "masked_token": "github_pat_...XXXX"
  }
}
```

**Response 200 (PAT not configured):**

```json
{
  "data": {
    "configured": false,
    "github_username": null,
    "set_at": null,
    "set_by": null,
    "masked_token": null
  }
}
```

**Errors:**

| Status | Code           | When                                        |
|--------|----------------|---------------------------------------------|
| 401    | `UNAUTHORIZED` | Missing or invalid access token             |
| 403    | `FORBIDDEN`    | User is not team admin or maintainer        |
| 404    | `NOT_FOUND`    | Team does not exist                         |

---

### DELETE /api/v1/teams/:id/pat

**Description:** Remove the team's stored GitHub PAT. Repositories assigned to this team will no longer be able to sync or clone until a new PAT is configured.

**Auth:** Authenticated | Team Role: admin

**Rate Limit:** 5 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>

Path Parameters:
  id  (UUID, required) -- Team ID
```

**Response 204:**

No response body.

**Errors:**

| Status | Code           | When                                        |
|--------|----------------|---------------------------------------------|
| 401    | `UNAUTHORIZED` | Missing or invalid access token             |
| 403    | `FORBIDDEN`    | User is not team admin                      |
| 404    | `NOT_FOUND`    | Team does not exist                         |

---

### POST /api/v1/teams/:id/members

**Description:** Invite a user to the team. The membership is created with `status: pending` until approved.

**Auth:** Authenticated | Team Role: admin, maintainer

**Rate Limit:** 10 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>
  Content-Type: application/json

Path Parameters:
  id  (UUID, required) -- Team ID

Body:
```

```json
{
  "user_id": "660e8400-e29b-41d4-a716-446655440001",
  "role": "viewer"
}
```

| Field     | Type   | Required | Constraints                                  |
|-----------|--------|----------|----------------------------------------------|
| `user_id` | UUID   | yes      | Must be an existing user                     |
| `role`    | string | yes      | One of: `admin`, `maintainer`, `viewer`. Maintainers can only invite as `viewer` |

**Response 201:**

```json
{
  "data": {
    "id": "mmm22222-3333-4444-5555-666677778888",
    "team_id": "111aaaaa-bbbb-cccc-dddd-eeee11112222",
    "user_id": "660e8400-e29b-41d4-a716-446655440001",
    "username": "jane",
    "role": "viewer",
    "status": "pending",
    "invited_by": "550e8400-e29b-41d4-a716-446655440000",
    "created_at": "2025-01-11T10:00:00Z"
  }
}
```

**Errors:**

| Status | Code               | When                                        |
|--------|--------------------|---------------------------------------------|
| 400    | `VALIDATION_ERROR` | Missing or invalid fields                   |
| 401    | `UNAUTHORIZED`     | Missing or invalid access token             |
| 403    | `FORBIDDEN`        | User is not team admin or maintainer, or maintainer tried to assign admin/maintainer role |
| 404    | `NOT_FOUND`        | Team or target user does not exist          |
| 409    | `CONFLICT`         | User is already a member of this team       |

---

### PUT /api/v1/teams/:id/members/:userId

**Description:** Update a team member's role or status (approve/reject invitations).

**Auth:** Authenticated | Team Role: admin, maintainer

**Rate Limit:** 10 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>
  Content-Type: application/json

Path Parameters:
  id      (UUID, required) -- Team ID
  userId  (UUID, required) -- User ID of the member to update

Body:
```

```json
{
  "role": "maintainer",
  "status": "approved"
}
```

| Field    | Type   | Required | Constraints                                      |
|----------|--------|----------|--------------------------------------------------|
| `role`   | string | no       | One of: `admin`, `maintainer`, `viewer`. Only team admins can promote to admin/maintainer |
| `status` | string | no       | One of: `pending`, `approved`, `rejected`        |

At least one field must be provided.

**Response 200:**

```json
{
  "data": {
    "id": "mmm22222-3333-4444-5555-666677778888",
    "team_id": "111aaaaa-bbbb-cccc-dddd-eeee11112222",
    "user_id": "660e8400-e29b-41d4-a716-446655440001",
    "username": "jane",
    "role": "maintainer",
    "status": "approved",
    "invited_by": "550e8400-e29b-41d4-a716-446655440000",
    "created_at": "2025-01-11T10:00:00Z"
  }
}
```

**Errors:**

| Status | Code               | When                                          |
|--------|--------------------|-----------------------------------------------|
| 400    | `VALIDATION_ERROR` | No fields provided or invalid values          |
| 401    | `UNAUTHORIZED`     | Missing or invalid access token               |
| 403    | `FORBIDDEN`        | Insufficient team role (maintainer tried to set role to admin) |
| 404    | `NOT_FOUND`        | Team or member does not exist                 |

---

### DELETE /api/v1/teams/:id/members/:userId

**Description:** Remove a member from the team. Members can also remove themselves (leave the team). Team admins cannot remove the last admin.

**Auth:** Authenticated | Team Role: admin, maintainer (or self-removal)

**Rate Limit:** 10 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>

Path Parameters:
  id      (UUID, required) -- Team ID
  userId  (UUID, required) -- User ID of the member to remove
```

**Response 204:**

No response body.

**Errors:**

| Status | Code               | When                                          |
|--------|--------------------|-----------------------------------------------|
| 401    | `UNAUTHORIZED`     | Missing or invalid access token               |
| 403    | `FORBIDDEN`        | User lacks permission to remove the member    |
| 404    | `NOT_FOUND`        | Team or member does not exist                 |
| 409    | `CONFLICT`         | Cannot remove the last admin from the team    |

---

### POST /api/v1/teams/:id/repositories

**Description:** Assign a repository to a team. The repository must be an existing, active repository.

**Auth:** Authenticated | Team Role: admin, maintainer

**Rate Limit:** 10 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>
  Content-Type: application/json

Path Parameters:
  id  (UUID, required) -- Team ID

Body:
```

```json
{
  "repository_id": "661f1a2b-3c4d-5e6f-7a8b-9c0d1e2f3a4b"
}
```

| Field           | Type | Required | Constraints                      |
|-----------------|------|----------|----------------------------------|
| `repository_id` | UUID | yes      | Must be an existing, active repo |

**Response 201:**

```json
{
  "data": {
    "id": "ttr11111-2222-3333-4444-555566667777",
    "team_id": "111aaaaa-bbbb-cccc-dddd-eeee11112222",
    "repository_id": "661f1a2b-3c4d-5e6f-7a8b-9c0d1e2f3a4b",
    "repository_name": "hello-world",
    "github_full_name": "octocat/hello-world",
    "added_by": "550e8400-e29b-41d4-a716-446655440000",
    "created_at": "2025-01-12T09:00:00Z"
  }
}
```

**Errors:**

| Status | Code               | When                                        |
|--------|--------------------|---------------------------------------------|
| 400    | `VALIDATION_ERROR` | Missing or invalid repository_id            |
| 401    | `UNAUTHORIZED`     | Missing or invalid access token             |
| 403    | `FORBIDDEN`        | User is not team admin or maintainer               |
| 404    | `NOT_FOUND`        | Team or repository does not exist           |
| 409    | `CONFLICT`         | Repository is already assigned to this team |

---

### DELETE /api/v1/teams/:id/repositories/:repoId

**Description:** Unassign a repository from a team.

**Auth:** Authenticated | Team Role: admin, maintainer

**Rate Limit:** 10 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>

Path Parameters:
  id      (UUID, required) -- Team ID
  repoId  (UUID, required) -- Repository ID
```

**Response 204:**

No response body.

**Errors:**

| Status | Code           | When                                          |
|--------|----------------|-----------------------------------------------|
| 401    | `UNAUTHORIZED` | Missing or invalid access token               |
| 403    | `FORBIDDEN`    | User is not team admin or maintainer                 |
| 404    | `NOT_FOUND`    | Team or team-repository assignment not found  |

---

## 7. Admin Endpoints

---

### GET /api/v1/admin/users

**Description:** List all users in the system. Supports search by username or email.

**Auth:** Authenticated | Role: root, moderator

**Rate Limit:** 30 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>

Query Parameters:
  page      (int, optional, default: 1)
  per_page  (int, optional, default: 20, max: 100)
  sort      (string, optional, default: "created_at", allowed: "created_at", "username", "email", "role")
  order     (string, optional, default: "desc", allowed: "asc", "desc")
  search    (string, optional) -- case-insensitive search across username and email
  role      (string, optional) -- filter by role: "root", "moderator", "user"
```

**Response 200:**

```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "jdoe",
      "email": "jdoe@example.com",
      "role": "user",
      "avatar_url": "https://avatars.githubusercontent.com/u/12345",
      "created_at": "2025-01-15T09:30:00Z",
      "updated_at": "2025-01-15T09:30:00Z"
    },
    {
      "id": "00000000-0000-0000-0000-000000000001",
      "username": "admin",
      "email": "admin@verdox.local",
      "role": "root",
      "avatar_url": null,
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-01T00:00:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 45,
    "total_pages": 3
  }
}
```

**Errors:**

| Status | Code               | When                                    |
|--------|--------------------|-----------------------------------------|
| 400    | `VALIDATION_ERROR` | Invalid query parameters                |
| 401    | `UNAUTHORIZED`     | Missing or invalid access token         |
| 403    | `FORBIDDEN`        | User is not root or moderator        |

---

### PUT /api/v1/admin/users/:id

**Description:** Update a user's role or active status. Role changes (promoting/demoting users) can only be performed by a root. Moderator users can only deactivate non-moderator accounts.

**Auth:** Authenticated | Role: root (for role changes), moderator (for deactivation only)

**Rate Limit:** 10 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>
  Content-Type: application/json

Path Parameters:
  id  (UUID, required) -- User ID

Body:
```

```json
{
  "role": "moderator",
  "is_active": true
}
```

| Field       | Type    | Required | Constraints                                          |
|-------------|---------|----------|------------------------------------------------------|
| `role`      | string  | no       | One of: `root`, `moderator`, `user`. root only |
| `is_active` | boolean | no       | Deactivate or reactivate a user account              |

At least one field must be provided.

**Response 200:**

```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "jdoe",
    "email": "jdoe@example.com",
    "role": "moderator",
    "avatar_url": "https://avatars.githubusercontent.com/u/12345",
    "created_at": "2025-01-15T09:30:00Z",
    "updated_at": "2025-01-20T16:00:00Z"
  }
}
```

**Errors:**

| Status | Code               | When                                          |
|--------|--------------------|-----------------------------------------------|
| 400    | `VALIDATION_ERROR` | No fields provided or invalid values          |
| 401    | `UNAUTHORIZED`     | Missing or invalid access token               |
| 403    | `FORBIDDEN`        | Moderator tried to change roles; or tried to modify root account |
| 404    | `NOT_FOUND`        | User does not exist                           |
| 409    | `CONFLICT`         | Cannot deactivate the last root        |

---

### GET /api/v1/admin/stats

**Description:** Get system-wide statistics for the admin dashboard.

**Auth:** Authenticated | Role: root, moderator

**Rate Limit:** 30 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>
```

No query parameters.

**Response 200:**

```json
{
  "data": {
    "total_users": 45,
    "total_repos": 120,
    "total_test_runs": 1580,
    "active_runners": 3,
    "queue_depth": 7,
    "runs_by_status": {
      "queued": 7,
      "running": 3,
      "passed": 1420,
      "failed": 140,
      "cancelled": 10
    },
    "recent_activity": {
      "runs_last_24h": 42,
      "runs_last_7d": 280
    }
  }
}
```

**Errors:**

| Status | Code           | When                                    |
|--------|----------------|-----------------------------------------|
| 401    | `UNAUTHORIZED` | Missing or invalid access token         |
| 403    | `FORBIDDEN`    | User is not root or moderator        |

---

## 8. User Endpoints

---

### GET /api/v1/me

**Description:** Get the authenticated user's profile.

**Auth:** Authenticated

**Rate Limit:** 30 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>
```

**Response 200:**

```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "jdoe",
    "email": "jdoe@example.com",
    "role": "user",
    "avatar_url": "https://avatars.githubusercontent.com/u/12345",
    "created_at": "2025-01-15T09:30:00Z",
    "updated_at": "2025-01-15T09:30:00Z"
  }
}
```

**Errors:**

| Status | Code           | When                                    |
|--------|----------------|-----------------------------------------|
| 401    | `UNAUTHORIZED` | Missing or invalid access token         |

---

### PUT /api/v1/me

**Description:** Update the authenticated user's profile. Only the provided fields are updated.

**Auth:** Authenticated

**Rate Limit:** 10 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>
  Content-Type: application/json

Body:
```

```json
{
  "username": "john_doe",
  "email": "john.doe@example.com",
  "avatar_url": "https://avatars.githubusercontent.com/u/99999"
}
```

| Field        | Type   | Required | Constraints                                    |
|--------------|--------|----------|------------------------------------------------|
| `username`   | string | no       | 3-64 chars, alphanumeric + underscores only    |
| `email`      | string | no       | Valid email, max 255 chars                     |
| `avatar_url` | string | no       | Valid URL or null to remove                    |

At least one field must be provided.

**Response 200:**

```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "john_doe",
    "email": "john.doe@example.com",
    "role": "user",
    "avatar_url": "https://avatars.githubusercontent.com/u/99999",
    "created_at": "2025-01-15T09:30:00Z",
    "updated_at": "2025-01-20T18:00:00Z"
  }
}
```

**Errors:**

| Status | Code               | When                                    |
|--------|--------------------|-----------------------------------------|
| 400    | `VALIDATION_ERROR` | Invalid field values                    |
| 401    | `UNAUTHORIZED`     | Missing or invalid access token         |
| 409    | `CONFLICT`         | Username or email already taken         |

---

### PUT /api/v1/me/password

**Description:** Change the authenticated user's password. Requires current password for verification.

**Auth:** Authenticated

**Rate Limit:** 5 req/min per user

**Request:**

```
Headers:
  Authorization: Bearer <access_token>
  Content-Type: application/json

Body:
```

```json
{
  "current_password": "OldSecureP@ss1",
  "new_password": "NewSecureP@ss2"
}
```

| Field              | Type   | Required | Constraints                                    |
|--------------------|--------|----------|------------------------------------------------|
| `current_password` | string | yes      | Must match the user's current password         |
| `new_password`     | string | yes      | Min 8 chars, at least 1 upper, 1 lower, 1 digit. Must differ from current password |

**Response 200:**

```json
{
  "message": "Password has been changed successfully."
}
```

**Errors:**

| Status | Code               | When                                      |
|--------|--------------------|-------------------------------------------|
| 400    | `VALIDATION_ERROR` | Missing fields or weak new password       |
| 401    | `UNAUTHORIZED`     | Missing or invalid access token, or current password is incorrect |
| 422    | `UNPROCESSABLE`    | New password is the same as current       |
| 429    | `RATE_LIMITED`     | Exceeded 5 req/min                        |

---

## 9. Discovery & Webhook Endpoints

---

### POST /api/v1/repositories/:id/discover

**Description:** Trigger AI-powered test discovery on a cloned repository. Requires `VERDOX_OPENAI_API_KEY` to be configured. Scans repo contents, identifies test files, classifies test types, and generates run scripts.

**Auth:** Authenticated | Team Role: admin, maintainer

**Rate Limit:** 3 req/min per user

**Response 202:**

```json
{
  "data": {
    "message": "Test discovery started",
    "repository_id": "661f1a2b-3c4d-5e6f-7a8b-9c0d1e2f3a4b"
  }
}
```

**Errors:**

| Status | Code               | When                                        |
|--------|--------------------|---------------------------------------------|
| 401    | `UNAUTHORIZED`     | Missing or invalid access token             |
| 403    | `FORBIDDEN`        | User lacks admin/maintainer role on team    |
| 422    | `UNPROCESSABLE`    | Repository not cloned yet                   |
| 503    | `SERVICE_UNAVAIL`  | `VERDOX_OPENAI_API_KEY` not configured      |

---

### GET /api/v1/repositories/:id/discovery

**Description:** Get the latest AI test discovery results for a repository.

**Auth:** Authenticated | Team Role: admin, maintainer, viewer

**Rate Limit:** 30 req/min per user

**Response 200:**

```json
{
  "data": {
    "id": "fff66666-7777-8888-9999-000011112222",
    "repository_id": "661f1a2b-3c4d-5e6f-7a8b-9c0d1e2f3a4b",
    "discovery_json": {
      "test_files": [
        { "path": "pkg/api/handler_test.go", "type": "unit", "framework": "go-test" },
        { "path": "tests/integration_test.go", "type": "integration", "framework": "go-test" }
      ],
      "suggested_suites": [
        { "name": "Unit Tests", "type": "unit", "command": "go test -v -json ./pkg/..." },
        { "name": "Integration Tests", "type": "integration", "command": "go test -v -json ./tests/..." }
      ]
    },
    "scripts_path": ".verdox/scripts/",
    "created_at": "2025-01-16T12:00:00Z"
  }
}
```

**Errors:**

| Status | Code               | When                                        |
|--------|--------------------|---------------------------------------------|
| 404    | `NOT_FOUND`        | No discovery results exist for this repo    |

---

### GET /api/v1/teams/discover

**Description:** List all teams available for discovery. Any authenticated user can call this to browse teams they might want to join. Returns basic team info without sensitive details.

**Auth:** Authenticated

**Rate Limit:** 30 req/min per user

**Response 200:**

```json
{
  "data": [
    {
      "id": "ccc33333-4444-5555-6666-777788889999",
      "name": "consul-team",
      "slug": "consul-team",
      "member_count": 5,
      "repo_count": 2,
      "created_at": "2025-01-10T08:00:00Z",
      "user_status": null
    },
    {
      "id": "ddd44444-5555-6666-7777-888899990000",
      "name": "vault-team",
      "slug": "vault-team",
      "member_count": 3,
      "repo_count": 1,
      "created_at": "2025-01-12T10:00:00Z",
      "user_status": "pending"
    }
  ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 2,
    "total_pages": 1
  }
}
```

> `user_status` is `null` if user has no relationship, `"pending"` if they have a join request, or `"approved"` if already a member.

---

### POST /api/v1/teams/:id/join-requests

**Description:** Submit a request to join a team. User must not already be a member or have a pending request.

**Auth:** Authenticated

**Rate Limit:** 10 req/min per user

**Request:**

```json
{
  "message": "I'd like to join to work on the Consul test suite"
}
```

| Field     | Type   | Required | Constraints        |
|-----------|--------|----------|--------------------|
| `message` | string | no       | Max 500 characters |

**Response 201:**

```json
{
  "data": {
    "id": "eee55555-6666-7777-8888-999900001111",
    "team_id": "ccc33333-4444-5555-6666-777788889999",
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "message": "I'd like to join to work on the Consul test suite",
    "status": "pending",
    "created_at": "2025-01-20T14:00:00Z"
  }
}
```

**Errors:**

| Status | Code               | When                                        |
|--------|--------------------|---------------------------------------------|
| 401    | `UNAUTHORIZED`     | Missing or invalid access token             |
| 404    | `NOT_FOUND`        | Team not found                              |
| 409    | `CONFLICT`         | Already a member or pending request exists  |

---

### GET /api/v1/teams/:id/join-requests

**Description:** List join requests for a team. Only team admin and maintainer can view.

**Auth:** Authenticated | Team Role: admin, maintainer

**Rate Limit:** 30 req/min per user

**Query Parameters:**

| Param    | Type   | Default   | Description                         |
|----------|--------|-----------|-------------------------------------|
| `status` | string | `pending` | Filter: `pending`, `approved`, `rejected`, `all` |

**Response 200:**

```json
{
  "data": [
    {
      "id": "eee55555-6666-7777-8888-999900001111",
      "user": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "username": "sujay",
        "email": "sujay@example.com",
        "avatar_url": null
      },
      "message": "I'd like to join to work on the Consul test suite",
      "status": "pending",
      "created_at": "2025-01-20T14:00:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 1,
    "total_pages": 1
  }
}
```

---

### PATCH /api/v1/teams/:id/join-requests/:requestId

**Description:** Approve or reject a join request. When approving, a `role` must be provided — the user is added as a team member with that role.

**Auth:** Authenticated | Team Role: admin, maintainer

**Rate Limit:** 10 req/min per user

**Request:**

```json
{
  "status": "approved",
  "role": "viewer"
}
```

| Field    | Type   | Required | Constraints                            |
|----------|--------|----------|----------------------------------------|
| `status` | string | yes      | `"approved"` or `"rejected"`           |
| `role`   | string | yes*     | `"viewer"`, `"maintainer"`, `"admin"`. Required when `status` = `"approved"` |

**Response 200:**

```json
{
  "data": {
    "id": "eee55555-6666-7777-8888-999900001111",
    "status": "approved",
    "role_assigned": "viewer",
    "reviewed_by": "aaa11111-2222-3333-4444-555566667777",
    "updated_at": "2025-01-20T15:00:00Z"
  }
}
```

**Side effect:** On approval, a `team_members` row is created with the assigned role and status `approved`.

**Errors:**

| Status | Code               | When                                        |
|--------|--------------------|---------------------------------------------|
| 400    | `VALIDATION_ERROR` | Missing role when approving                 |
| 403    | `FORBIDDEN`        | User lacks admin/maintainer role on team    |
| 404    | `NOT_FOUND`        | Join request not found                      |
| 409    | `CONFLICT`         | Request already processed                   |

---

> **Note:** GitHub webhooks (POST /api/v1/webhooks/github) are planned for v2. In v1, test runs are triggered manually by team admins/maintainers.

---

## 10. Endpoint Summary

| Method | Path                                       | Auth             | Rate Limit   | Description                          |
|--------|--------------------------------------------|------------------|--------------|--------------------------------------|
| POST   | `/api/v1/auth/signup`                      | Public           | 5/min (IP)   | Register new user                    |
| POST   | `/api/v1/auth/login`                       | Public           | 5/min (IP)   | Authenticate user                    |
| POST   | `/api/v1/auth/refresh`                     | Refresh Cookie   | 10/min       | Refresh access token                 |
| POST   | `/api/v1/auth/logout`                      | Authenticated    | 10/min       | Invalidate session                   |
| POST   | `/api/v1/auth/forgot-password`             | Public           | 3/min (IP)   | Request password reset               |
| POST   | `/api/v1/auth/reset-password`              | Public + Token   | 5/min (IP)   | Reset password with token            |
| GET    | `/api/v1/repositories`                     | Authenticated    | 30/min       | List accessible repositories         |
| POST   | `/api/v1/repositories`                     | root/mod/team admin | 5/min      | Add repository by GitHub URL         |
| POST   | `/api/v1/repositories/:id/resync`          | Authenticated    | 10/min       | Re-fetch from remote                 |
| GET    | `/api/v1/repositories/:id`                 | Authenticated    | 30/min       | Repository detail with suites        |
| DELETE | `/api/v1/repositories/:id`                 | Authenticated    | 10/min       | Deactivate repository (soft delete)  |
| GET    | `/api/v1/repositories/:id/branches`        | Authenticated    | 20/min       | List branches (local clone, cached)  |
| GET    | `/api/v1/repositories/:id/commits`         | Authenticated    | 20/min       | List commits for branch (local)      |
| POST   | `/api/v1/repositories/:id/discover`        | Team: admin, maint | 3/min      | AI test discovery                    |
| GET    | `/api/v1/repositories/:id/discovery`       | Authenticated    | 30/min       | Get discovery results                |
| GET    | `/api/v1/repositories/:id/suites`          | Authenticated    | 30/min       | List test suites for repo            |
| POST   | `/api/v1/repositories/:id/suites`          | Authenticated    | 10/min       | Create test suite                    |
| PUT    | `/api/v1/suites/:id`                       | Authenticated    | 10/min       | Update test suite                    |
| DELETE | `/api/v1/suites/:id`                       | Authenticated    | 10/min       | Delete test suite                    |
| POST   | `/api/v1/suites/:id/run`                   | Team: admin, maint | 10/min     | Trigger test run                     |
| GET    | `/api/v1/suites/:id/runs`                  | Authenticated    | 30/min       | List runs for suite                  |
| GET    | `/api/v1/runs/:id`                         | Authenticated    | 30/min       | Run detail with results              |
| GET    | `/api/v1/runs/:id/logs`                    | Authenticated    | 20/min       | Get run logs                         |
| POST   | `/api/v1/runs/:id/cancel`                  | Authenticated    | 10/min       | Cancel queued/running run            |
| POST   | `/api/v1/repositories/:id/run-all`         | Team: admin, maint | 5/min      | Run all suites for repo              |
| GET    | `/api/v1/teams`                            | Authenticated    | 30/min       | List user's teams                    |
| GET    | `/api/v1/teams/discover`                   | Authenticated    | 30/min       | Browse all teams for discovery       |
| POST   | `/api/v1/teams`                            | Authenticated    | 10/min       | Create team                          |
| GET    | `/api/v1/teams/:id`                        | Authenticated    | 30/min       | Team detail with members and repos   |
| PUT    | `/api/v1/teams/:id`                        | Team: admin      | 10/min       | Update team                          |
| DELETE | `/api/v1/teams/:id`                        | Team: admin / root | 5/min        | Delete team                          |
| PUT    | `/api/v1/teams/:id/pat`                    | Team: admin      | 5/min        | Set/update team GitHub PAT           |
| GET    | `/api/v1/teams/:id/pat/status`             | Team: admin, maint | 10/min     | Check team PAT status                |
| DELETE | `/api/v1/teams/:id/pat`                    | Team: admin      | 5/min        | Remove team GitHub PAT               |
| POST   | `/api/v1/teams/:id/members`                | Team: admin, maintainer | 10/min       | Invite team member                   |
| PUT    | `/api/v1/teams/:id/members/:userId`        | Team: admin, maintainer | 10/min       | Update member role/status            |
| DELETE | `/api/v1/teams/:id/members/:userId`        | Team: admin, maintainer | 10/min       | Remove team member                   |
| POST   | `/api/v1/teams/:id/repositories`           | Team: admin, maintainer | 10/min       | Assign repo to team                  |
| DELETE | `/api/v1/teams/:id/repositories/:repoId`   | Team: admin, maintainer | 10/min       | Unassign repo from team              |
| POST   | `/api/v1/teams/:id/join-requests`          | Authenticated    | 10/min       | Request to join team                 |
| GET    | `/api/v1/teams/:id/join-requests`          | Team: admin, maint | 30/min     | List join requests                   |
| PATCH  | `/api/v1/teams/:id/join-requests/:rid`     | Team: admin, maint | 10/min     | Approve/reject join request          |
| GET    | `/api/v1/admin/users`                      | Role: root, moderator | 30/min  | List all users                       |
| PUT    | `/api/v1/admin/users/:id`                  | Role: root       | 10/min       | Update user role/status              |
| GET    | `/api/v1/admin/stats`                      | Role: root, moderator | 30/min  | System statistics                    |
| GET    | `/api/v1/me`                               | Authenticated    | 30/min       | Current user profile                 |
| PUT    | `/api/v1/me`                               | Authenticated    | 10/min       | Update profile                       |
| PUT    | `/api/v1/me/password`                      | Authenticated    | 5/min        | Change password                      |

---

## 11. Go/Echo Implementation Notes

### Middleware Chain

Every request passes through middleware in this order:

```
Request
  -> RecoveryMiddleware      (panic recovery, returns 500)
  -> RequestIDMiddleware     (adds X-Request-ID header)
  -> LoggerMiddleware        (structured JSON logging via zerolog)
  -> CORSMiddleware          (restricted to frontend origin)
  -> RateLimitMiddleware     (Redis-backed sliding window)
  -> AuthMiddleware          (JWT validation, sets user in context) [protected routes only]
  -> RoleMiddleware          (checks user.role) [admin routes only]
  -> TeamRoleMiddleware      (checks team_members.role) [team routes only]
  -> Handler
```

### Route Groups

```go
// Public routes
auth := v1.Group("/auth")
auth.POST("/signup", authHandler.Signup)
auth.POST("/login", authHandler.Login)
auth.POST("/refresh", authHandler.Refresh)
auth.POST("/forgot-password", authHandler.ForgotPassword)
auth.POST("/reset-password", authHandler.ResetPassword)

// Webhook (public, signature-verified)
v1.POST("/webhooks/github", webhookHandler.GitHub)

// Authenticated routes
auth.POST("/logout", authHandler.Logout, authMiddleware)

me := v1.Group("/me", authMiddleware)
me.GET("", userHandler.GetProfile)
me.PUT("", userHandler.UpdateProfile)
me.PUT("/password", userHandler.ChangePassword)

repos := v1.Group("/repositories", authMiddleware)
repos.GET("", repoHandler.List)
repos.POST("", repoHandler.Add)  // Add repo by URL, uses team PAT (root/moderator/team admin)
repos.POST("/:id/resync", repoHandler.Resync)
repos.GET("/:id", repoHandler.Get)
repos.DELETE("/:id", repoHandler.Delete)
repos.GET("/:id/branches", repoHandler.ListBranches)
repos.GET("/:id/commits", repoHandler.ListCommits)
repos.POST("/:id/discover", discoveryHandler.Trigger)
repos.GET("/:id/discovery", discoveryHandler.Get)
repos.GET("/:id/suites", suiteHandler.List)
repos.POST("/:id/suites", suiteHandler.Create)
repos.POST("/:id/run-all", runHandler.RunAll)

suites := v1.Group("/suites", authMiddleware)
suites.PUT("/:id", suiteHandler.Update)
suites.DELETE("/:id", suiteHandler.Delete)
suites.POST("/:id/run", runHandler.Trigger)
suites.GET("/:id/runs", runHandler.ListBySuite)

runs := v1.Group("/runs", authMiddleware)
runs.GET("/:id", runHandler.Get)
runs.GET("/:id/logs", runHandler.Logs)
runs.POST("/:id/cancel", runHandler.Cancel)

teams := v1.Group("/teams", authMiddleware)
teams.GET("", teamHandler.List)
teams.POST("", teamHandler.Create)
teams.GET("/:id", teamHandler.Get)
teams.PUT("/:id", teamHandler.Update, teamAdminMiddleware)
teams.DELETE("/:id", teamHandler.Delete, teamAdminOrRootMiddleware)
teams.PUT("/:id/pat", teamPatHandler.Set, teamAdminMiddleware)
teams.GET("/:id/pat/status", teamPatHandler.Status, teamAdminMaintainerMiddleware)
teams.DELETE("/:id/pat", teamPatHandler.Delete, teamAdminMiddleware)
teams.POST("/:id/members", teamHandler.InviteMember, teamAdminMaintainerMiddleware)
teams.PUT("/:id/members/:userId", teamHandler.UpdateMember, teamAdminMaintainerMiddleware)
teams.DELETE("/:id/members/:userId", teamHandler.RemoveMember, teamAdminMaintainerMiddleware)
teams.POST("/:id/repositories", teamHandler.AssignRepo, teamAdminMaintainerMiddleware)
teams.DELETE("/:id/repositories/:repoId", teamHandler.UnassignRepo, teamAdminMaintainerMiddleware)
teams.POST("/:id/join-requests", joinRequestHandler.Create)
teams.GET("/:id/join-requests", joinRequestHandler.List, teamAdminMaintainerMiddleware)
teams.PATCH("/:id/join-requests/:requestId", joinRequestHandler.Review, teamAdminMaintainerMiddleware)
teams.GET("/discover", teamHandler.Discover)  // No team role check needed

// Admin routes
admin := v1.Group("/admin", authMiddleware, adminRoleMiddleware)
admin.GET("/users", adminHandler.ListUsers)
admin.PUT("/users/:id", adminHandler.UpdateUser)
admin.GET("/stats", adminHandler.Stats)
```

### Rate Limiting Strategy

Rate limits are enforced via Redis using a sliding window algorithm:

- **Key format:** `rate:{endpoint_group}:{identifier}` where identifier is user ID (authenticated) or IP address (public).
- **Window:** 60 seconds.
- **Headers returned on every response:**
  - `X-RateLimit-Limit`: Maximum requests per window
  - `X-RateLimit-Remaining`: Remaining requests in current window
  - `X-RateLimit-Reset`: Unix timestamp when the window resets
- **429 response** includes a `Retry-After` header (seconds until reset).

### Webhook Signature Verification

The GitHub webhook handler verifies payloads using HMAC-SHA256:

1. Read the raw request body.
2. Compute `HMAC-SHA256(webhook_secret, body)`.
3. Compare the computed hash with the `X-Hub-Signature-256` header value using `hmac.Equal()` (constant-time comparison).
4. If the signature does not match, return `401 UNAUTHORIZED`.
5. Look up the repository by `github_repo_id` from the payload.
6. Process the event based on the `X-GitHub-Event` header value.
