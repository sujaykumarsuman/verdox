# Verdox -- GitHub Integration (LLD)

> Go 1.25+ | Echo v4 | go-github/v60 | Redis 7 | AES-256-GCM

---

## 1. Integration Strategy

Verdox integrates with GitHub for two purposes:

1. **Repository management** -- adding repos, listing branches and commits.
2. **Test execution** -- forking repos, pushing workflow files, dispatching
   GitHub Actions workflows, and polling for results.

### PAT Hierarchy

Verdox uses two types of PATs with distinct responsibilities:

| PAT | Configured In | Purpose | Required Scopes |
|-----|--------------|---------|-----------------|
| **Service account PAT** | `.env` (`VERDOX_SERVICE_ACCOUNT_PAT`) | Fork repos, push workflow files, dispatch GHA workflows, poll status, download artifacts | `repo`, `workflow`, `read:org` |
| **Team PAT** (optional) | Team Settings UI (per-team, stored encrypted in DB) | Access private repos the service account cannot see. Used for repo validation and branch/commit listing | `repo` (or fine-grained with Contents read) |

**Resolution order for GitHub API calls:**

1. For fork/workflow operations: always use the service account PAT.
2. For repo validation and branch/commit listing: use the team PAT if
   configured, otherwise fall back to the service account PAT.

### v1: Fork-Based GitHub Actions

For v1, Verdox uses a fork-based approach for test execution:

1. **Service account forks the repo.** The Verdox service account creates a
   fork of the target repository under its own GitHub account.
2. **Verdox pushes a workflow file.** A `verdox-test.yml` workflow is pushed
   to the fork's `.github/workflows/` directory.
3. **Workflow dispatch triggers tests.** The `workflow_dispatch` event is used
   to trigger the workflow with test parameters.
4. **GHA runs tests.** GitHub Actions executes the tests on GitHub-hosted
   runners (or self-hosted runners configured on the fork).
5. **Verdox polls for results.** The backend polls the GitHub Actions API for
   workflow run completion, then downloads logs and artifacts.

This approach requires no Docker-in-Docker, no privileged containers, and no
local compute for test execution.

### v2 (future): GitHub App

When Verdox needs org-level installation tokens, fine-grained permissions,
and automatic token rotation, the migration path is a GitHub App with
installation tokens that replace the service account PAT.

---

## 2. Service Account Setup

### Requirements

1. **Create a dedicated GitHub account** (e.g., `verdox-bot`, `yourorg-ci`).
2. **Add it to the GitHub organization** as a member with read access to
   target repositories.
3. **Generate a classic PAT** with the following scopes:

| Scope | Why |
|-------|-----|
| `repo` | Fork repos, push workflow files, access private repos |
| `workflow` | Dispatch and manage GitHub Actions workflows |
| `read:org` | Read organization membership for private repo access |

4. **Configure in `.env`:**

```
VERDOX_SERVICE_ACCOUNT_PAT=ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
VERDOX_SERVICE_ACCOUNT_USERNAME=verdox-bot
```

---

## 3. Team PAT Configuration (Optional)

Team PATs provide access to private repositories that the service account
cannot see. They are optional -- if the service account has access to all
repos the team uses, no team PAT is needed.

### User Flow

1. Team admin navigates to **Team Settings -> GitHub**.
2. Team admin generates a PAT on GitHub with the `repo` scope
   (see `docs/GITHUB-PAT-GUIDE.md` for step-by-step instructions).
3. Team admin pastes the PAT into the Verdox team settings form and submits.
4. Backend validates the PAT, encrypts it, and stores it on the `teams` table.
5. Any team admin can update or rotate the PAT at any time.

### Sequence

```
Team Admin              Verdox Frontend          Verdox Backend          GitHub
 |                           |                        |                    |
 |  1. Paste PAT             |                        |                    |
 |-------------------------->|                        |                    |
 |                           |  2. PUT /api/v1/teams/  |                    |
 |                           |     :team_id/pat       |                    |
 |                           |----------------------->|                    |
 |                           |                        |                    |
 |                           |            3. Validate PAT                  |
 |                           |                        |  GET /user         |
 |                           |                        |------------------->|
 |                           |                        |  200 OK            |
 |                           |                        |<-------------------|
 |                           |                        |                    |
 |                           |            4. Encrypt (AES-256-GCM)         |
 |                           |            5. Store on teams table          |
 |                           |                        |                    |
 |                           |  6. Return success      |                    |
 |                           |<-----------------------|                    |
 |  7. PAT connected         |                        |                    |
 |<--------------------------|                        |                    |
```

### API

**Save PAT (team admin only):**

```
PUT /api/v1/teams/:team_id/pat

{
  "provider": "github",
  "token": "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
}
```

Response:

```json
{
  "status": "success",
  "data": {
    "provider": "github",
    "github_username": "sujaykumar",
    "set_at": "2026-04-06T10:30:00Z",
    "set_by": "user-uuid"
  }
}
```

**Validate stored PAT:**

```
GET /api/v1/teams/:team_id/pat/validate?provider=github
```

### Validation Logic

The backend validates the PAT by calling the GitHub API before storing it:

```
GET https://api.github.com/user
Authorization: Bearer {pat}
```

- **200 OK** -- PAT is valid. Extract the GitHub username from the response.
- **401 Unauthorized** -- PAT is invalid or revoked. Return
  `422 UNPROCESSABLE` to the frontend.

### Error Responses

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `VALIDATION_ERROR` | Missing provider or token |
| 401 | `UNAUTHORIZED` | User not authenticated |
| 403 | `FORBIDDEN` | User is not a team admin |
| 422 | `UNPROCESSABLE` | PAT is invalid or revoked by GitHub |
| 429 | `RATE_LIMITED` | PAT validation endpoint rate limit (5/min per user) |

---

## 4. Fork-Based Workflow

### Fork Lifecycle

When a test run is triggered for a repository, the following fork operations
occur:

1. **Fork creation (first run only):**
   ```
   POST /repos/{owner}/{repo}/forks
   Authorization: Bearer {service_account_pat}
   ```
   The fork is created under the service account's GitHub account. Fork
   metadata is stored in the `repository_forks` table.

2. **Upstream sync (every run):**
   ```
   POST /repos/{fork_owner}/{fork_repo}/merge-upstream
   Authorization: Bearer {service_account_pat}
   Body: { "branch": "{target_branch}" }
   ```
   This ensures the fork has the latest code from upstream before each test
   run.

3. **Workflow push (first run or on config change):**
   ```
   PUT /repos/{fork_owner}/{fork_repo}/contents/.github/workflows/verdox-test.yml
   Authorization: Bearer {service_account_pat}
   ```
   The workflow file is generated from the test suite configuration and pushed
   to the fork. It is only re-pushed if the suite configuration changes
   (tracked via `workflow_sha` in `repository_forks`).

### Workflow Dispatch

Once the fork is synced and the workflow file is in place, Verdox dispatches
the workflow:

```
POST /repos/{fork_owner}/{fork_repo}/actions/workflows/verdox-test.yml/dispatches
Authorization: Bearer {service_account_pat}

{
  "ref": "{branch}",
  "inputs": {
    "run_id": "{verdox_run_id}",
    "branch": "{branch}",
    "test_command": "{test_command}",
    "webhook_url": "{optional_callback_url}"
  }
}
```

### Result Collection

After dispatching, the backend monitors the workflow run:

1. **Poll for workflow run:**
   ```
   GET /repos/{fork_owner}/{fork_repo}/actions/runs?event=workflow_dispatch
   ```
   Match the run by timing and branch to identify the dispatched workflow.

2. **Poll for completion:**
   ```
   GET /repos/{fork_owner}/{fork_repo}/actions/runs/{run_id}
   ```
   Check `status == "completed"` and read `conclusion` (success/failure).

3. **Download logs:**
   ```
   GET /repos/{fork_owner}/{fork_repo}/actions/runs/{run_id}/logs
   ```
   Returns a zip archive of all job logs.

4. **Parse and store results** in the `test_results` table.

---

## 5. Repository Addition

**Endpoint:** `POST /api/v1/repositories`

Users add repositories one at a time by providing a GitHub URL. Only `root`,
`moderator`, or team `admin` roles can add repositories.

### Step-by-Step

1. **Parse the GitHub URL.** Extract `owner` and `repo` from the URL:
   ```
   https://github.com/hashicorp/consul  ->  owner=hashicorp, repo=consul
   ```

2. **Retrieve the PAT** for API validation. Use the team PAT if configured,
   otherwise fall back to the service account PAT.

3. **Validate the repository** by calling the GitHub API:
   ```
   GET https://api.github.com/repos/{owner}/{repo}
   Authorization: Bearer {pat}
   ```
   - **200 OK** -- repo exists and the PAT has access.
   - **404 Not Found** -- repo does not exist or PAT has no access.
   - **403 Forbidden** -- PAT lacks permissions on the repo.

4. **Create the `repositories` row** with metadata from the GitHub API
   response (repo ID, full name, default branch, description).

5. **Return the created repository:**
   ```json
   {
     "status": "success",
     "data": {
       "id": "uuid",
       "github_full_name": "hashicorp/consul",
       "name": "consul",
       "default_branch": "main",
       "team_id": "uuid"
     }
   }
   ```

Note: Fork creation is deferred to the first test run trigger. No local clone
is created.

### Error Responses

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `VALIDATION_ERROR` | Invalid GitHub URL format |
| 401 | `UNAUTHORIZED` | User not authenticated |
| 403 | `FORBIDDEN` | User role insufficient (not root/moderator/admin) |
| 404 | `NOT_FOUND` | Repository does not exist on GitHub |
| 409 | `CONFLICT` | Repository already added |
| 422 | `UNPROCESSABLE` | PAT not configured and service account has no access |

---

## 6. Branch Listing

**Endpoint:** `GET /api/v1/repositories/:id/branches`

### Step-by-Step

1. **Look up the repository** in the database by UUID. Verify the requesting
   user has access.

2. **Check Redis cache.** Look for cached branch data at key
   `branches:{repo_id}` (TTL: 5 minutes). If a cache hit, return the cached
   response immediately.

3. **List branches from the GitHub API:**
   ```
   GET https://api.github.com/repos/{owner}/{repo}/branches?per_page=100
   Authorization: Bearer {pat}
   ```
   PAT resolution: team PAT if configured, otherwise service account PAT.

4. **Cache the result in Redis** (TTL: 5 minutes).

5. **Return the branch list:**
   ```json
   {
     "status": "success",
     "data": [
       { "name": "main", "commit_sha": "a1b2c3d" },
       { "name": "develop", "commit_sha": "f6e5d4c" }
     ]
   }
   ```

### Error Responses

| Status | Code | Condition |
|--------|------|-----------|
| 404 | `NOT_FOUND` | Repository does not exist in Verdox |
| 500 | `INTERNAL_ERROR` | GitHub API call failed |

---

## 7. Commit Listing

**Endpoint:** `GET /api/v1/repositories/:id/commits?branch=main`

### Step-by-Step

1. **Validate the `branch` query parameter.** It is required; return
   `400 VALIDATION_ERROR` if missing.

2. **Check Redis cache.** Look for cached commit data at key
   `commits:{repo_id}:{branch}` (TTL: 2 minutes).

3. **Fetch commits from the GitHub API:**
   ```
   GET https://api.github.com/repos/{owner}/{repo}/commits?sha={branch}&per_page=20
   Authorization: Bearer {pat}
   ```

4. **Cache the result in Redis** (TTL: 2 minutes).

5. **Return the commit list:**
   ```json
   {
     "status": "success",
     "data": [
       {
         "sha": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
         "message": "fix: resolve null pointer in test parser",
         "author": "Sujay Kumar",
         "date": "2026-04-05T10:30:00+05:30"
       }
     ]
   }
   ```

### Error Responses

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `VALIDATION_ERROR` | Missing `branch` query parameter |
| 404 | `NOT_FOUND` | Repository or branch does not exist |
| 500 | `INTERNAL_ERROR` | GitHub API call failed |

---

## 8. Repository Deletion

**Endpoint:** `DELETE /api/v1/repositories/:id`

### Soft Delete (default)

- Set `is_active = FALSE` on the `repositories` row.
- The associated fork is not deleted (can be cleaned up manually or by a
  background job).
- The repository disappears from the UI but data is preserved.

### Hard Delete (admin action)

Triggered with `?hard=true` query parameter. Requires `root` role.

- Delete the `repositories` row from the database.
- Delete the `repository_forks` row if it exists.
- Optionally delete the fork on GitHub via `DELETE /repos/{fork_owner}/{fork_repo}`.

### Error Responses

| Status | Code | Condition |
|--------|------|-----------|
| 403 | `FORBIDDEN` | Hard delete requires root role |
| 404 | `NOT_FOUND` | Repository does not exist |

---

## 9. GitHub API Rate Limiting

With the fork-based approach, GitHub API usage is higher than the previous
clone-based approach because branch listing, commit listing, fork operations,
and workflow management all use the API.

### Rate Limits

- Authenticated PAT requests: **5,000/hour** per PAT.
- Service account PAT handles fork/workflow operations.
- Team PAT (if configured) handles repo validation and branch/commit listing,
  distributing the load across two tokens.

### Tracking

- Read `X-RateLimit-Remaining` header from all GitHub API responses.
- Log a warning when remaining requests fall below **500**.
- Return a `429 RATE_LIMITED` error if remaining is **0**.
- Back off and retry when rate-limited.

---

## 10. Token Encryption

PATs (team PATs) are encrypted at rest using AES-256-GCM.

### Go Implementation

```go
// pkg/crypto/encrypt.go

func Encrypt(plaintext string, hexKey string) (string, error) {
    // AES-256-GCM encryption
    // Returns hex-encoded nonce + ciphertext
}

func Decrypt(hexCiphertext string, hexKey string) (string, error) {
    // AES-256-GCM decryption
    // Extracts nonce from prepended bytes
}
```

The encryption key is loaded from the `GITHUB_TOKEN_ENCRYPTION_KEY` environment
variable (32 bytes, hex-encoded). Generate with: `openssl rand -hex 32`.

---

## 11. Error Handling

| Scenario | Detection | Response |
|----------|-----------|----------|
| Invalid service account PAT | 401 from GitHub during fork/dispatch | Log critical error, fail the run, alert admin |
| Invalid team PAT | 401 from GitHub during validation | Prompt team admin to update PAT |
| Repo not found | 404 from GitHub during repo addition | Return error to user |
| No repo access | 403 from GitHub | PAT needs repo access on GitHub |
| Fork creation fails | GitHub API error | Set run status to 'failed', log error |
| Workflow dispatch fails | 404 or 422 from GitHub | Re-push workflow, retry once |
| GHA run timeout | Poll duration exceeded | Cancel workflow, set status to 'timed_out' |
| GitHub API rate limit | 403 with X-RateLimit-Remaining: 0 | Back off, retry after reset |

---

## 12. Environment Variables

```
VERDOX_SERVICE_ACCOUNT_PAT=ghp_xxxxxxxxxxxx   # Service account PAT (required)
VERDOX_SERVICE_ACCOUNT_USERNAME=verdox-bot     # Service account GitHub username (required)
VERDOX_WEBHOOK_BASE_URL=https://...            # Optional webhook callback URL
GITHUB_TOKEN_ENCRYPTION_KEY=<32-byte-hex>      # AES-256-GCM key for team PAT encryption
```

---

## 13. Security Considerations

- **Service account PAT** is stored in `.env` (server-side only, never exposed
  to the frontend or logged).
- **Team PATs** are encrypted at rest with AES-256-GCM. Never logged, never
  returned in API responses.
- **Fork isolation:** Tests run on a fork, not the original repository. The
  fork is owned by the service account and cannot push to upstream.
- **Workflow file is Verdox-managed:** The `verdox-test.yml` file is generated
  and pushed by Verdox. Users cannot inject arbitrary workflow commands -- the
  test command is the only configurable part, and it runs in a standard GHA
  step.
- **Rate limit on PAT validation endpoint** -- 5 requests per minute per user.

---

## 14. Future Considerations

- **GitHub App** -- for org-level installation tokens, fine-grained
  permissions, and a dedicated bot identity (`verdox[bot]`).
- **Webhook-first mode** -- use GitHub workflow_run webhooks instead of
  polling for faster result notification.
- **Additional providers** -- GitLab, Bitbucket. Provider-specific
  integration modules can follow the same fork-based pattern.
- **Self-hosted runners** -- support for teams that want to run tests on
  their own infrastructure via self-hosted GHA runners on the fork.
