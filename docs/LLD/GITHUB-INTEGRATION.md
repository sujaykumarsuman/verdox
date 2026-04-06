# Verdox -- GitHub Integration (LLD)

> Go 1.25+ | Echo v4 | go-github/v60 | Redis 7 | AES-256-GCM

---

## 1. Integration Strategy

Verdox integrates with GitHub to add repositories, clone them locally, and
use the local clone for branch listing, commit listing, and test execution.
This avoids heavy reliance on the GitHub API after initial setup.

### v1: Personal Access Token (PAT)

For v1, a team admin provides a single GitHub PAT in team settings. This is
the pragmatic choice for self-hosted deployments because:

1. **No app registration required.** A team admin generates a PAT in their
   GitHub settings -- no OAuth app, no client ID/secret, no callback URLs.
2. **Works behind firewalls.** Self-hosted Verdox instances don't need to be
   publicly reachable (no callback endpoint required).
3. **Team-scoped access is sufficient.** One PAT per team covers all
   repositories added by that team. Any team admin can rotate the PAT.

For detailed PAT creation instructions, see `docs/GITHUB-PAT-GUIDE.md`.

### v2 (future): GitHub OAuth or GitHub App

When Verdox needs browser-based token acquisition or organization-level
access, the migration path is:

1. **GitHub OAuth App** -- for automatic token management via browser-based
   authorization flow. Removes the need for users to manually create PATs.
2. **GitHub App** -- for org-level installation tokens, fine-grained
   permissions, and webhook support for push-triggered test runs.

The `teams` table stores the PAT alongside the team record. Adding new
providers (GitLab, Bitbucket) in v2+ can follow the same pattern with
additional provider-specific columns on the `teams` table.

---

## 2. PAT Configuration

### User Flow

1. Team admin navigates to **Team Settings -> GitHub**.
2. Team admin generates a PAT on GitHub with the required scope
   (see `docs/GITHUB-PAT-GUIDE.md` for step-by-step instructions):
   - `repo` -- full access to private and public repositories.
   - `public_repo` -- sufficient if only public repositories are needed.
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

Response (valid):

```json
{
  "status": "success",
  "data": {
    "provider": "github",
    "valid": true,
    "github_username": "sujaykumar",
    "scopes": "repo"
  }
}
```

Response (invalid/revoked):

```json
{
  "status": "success",
  "data": {
    "provider": "github",
    "valid": false,
    "error": "PAT returned 401 from GitHub -- token may be revoked"
  }
}
```

### Validation Logic

The backend validates the PAT by calling the GitHub API before storing it:

```
GET https://api.github.com/user
Authorization: Bearer {pat}
```

- **200 OK** -- PAT is valid. Extract the GitHub username from the response
  for the `label` field.
- **401 Unauthorized** -- PAT is invalid or revoked. Return
  `422 UNPROCESSABLE` to the frontend.

### Database Migration

Add PAT columns to the `teams` table:

```sql
-- migrations/000011_add_team_pat_fields.up.sql
ALTER TABLE teams ADD COLUMN github_pat_encrypted TEXT;              -- AES-256-GCM ciphertext
ALTER TABLE teams ADD COLUMN github_pat_nonce TEXT;                  -- GCM nonce (hex-encoded)
ALTER TABLE teams ADD COLUMN github_pat_set_at TIMESTAMPTZ;          -- when the PAT was set
ALTER TABLE teams ADD COLUMN github_pat_set_by UUID REFERENCES users(id); -- which admin set it
ALTER TABLE teams ADD COLUMN github_pat_github_username VARCHAR(255); -- GitHub username the PAT belongs to
```

```sql
-- migrations/000011_add_team_pat_fields.down.sql
ALTER TABLE teams DROP COLUMN IF EXISTS github_pat_encrypted;
ALTER TABLE teams DROP COLUMN IF EXISTS github_pat_nonce;
ALTER TABLE teams DROP COLUMN IF EXISTS github_pat_set_at;
ALTER TABLE teams DROP COLUMN IF EXISTS github_pat_set_by;
ALTER TABLE teams DROP COLUMN IF EXISTS github_pat_github_username;
```

### Token Encryption (Go outline)

```go
// pkg/crypto/encrypt.go

package crypto

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/hex"
    "fmt"
    "io"
)

// Encrypt encrypts plaintext using AES-256-GCM with the provided hex-encoded key.
// Returns a hex-encoded string of nonce + ciphertext.
func Encrypt(plaintext string, hexKey string) (string, error) {
    key, err := hex.DecodeString(hexKey)
    if err != nil || len(key) != 32 {
        return "", fmt.Errorf("encryption key must be 32 bytes hex-encoded")
    }

    block, err := aes.NewCipher(key)
    if err != nil {
        return "", fmt.Errorf("creating cipher: %w", err)
    }

    aesGCM, err := cipher.NewGCM(block)
    if err != nil {
        return "", fmt.Errorf("creating GCM: %w", err)
    }

    nonce := make([]byte, aesGCM.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", fmt.Errorf("generating nonce: %w", err)
    }

    ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
    return hex.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a hex-encoded AES-256-GCM ciphertext (nonce prepended).
func Decrypt(hexCiphertext string, hexKey string) (string, error) {
    key, err := hex.DecodeString(hexKey)
    if err != nil || len(key) != 32 {
        return "", fmt.Errorf("encryption key must be 32 bytes hex-encoded")
    }

    ciphertext, err := hex.DecodeString(hexCiphertext)
    if err != nil {
        return "", fmt.Errorf("decoding ciphertext: %w", err)
    }

    block, err := aes.NewCipher(key)
    if err != nil {
        return "", fmt.Errorf("creating cipher: %w", err)
    }

    aesGCM, err := cipher.NewGCM(block)
    if err != nil {
        return "", fmt.Errorf("creating GCM: %w", err)
    }

    nonceSize := aesGCM.NonceSize()
    if len(ciphertext) < nonceSize {
        return "", fmt.Errorf("ciphertext too short")
    }

    nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
    plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", fmt.Errorf("decrypting: %w", err)
    }

    return string(plaintext), nil
}
```

### PAT Handler (Go outline)

```go
// internal/handler/team.go (PAT management methods)

func (h *PATHandler) SaveTeamPAT(c echo.Context) error {
    teamID := c.Param("team_id")

    var req struct {
        Provider string `json:"provider" validate:"required,oneof=github"`
        Token    string `json:"token" validate:"required"`
    }
    if err := c.Bind(&req); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
    }

    userID := c.Get("user_id").(string)

    // 0. Verify caller is a team admin
    if !h.teamService.IsAdmin(c.Request().Context(), teamID, userID) {
        return echo.NewHTTPError(http.StatusForbidden, "only team admins can set the PAT")
    }

    // 1. Validate PAT by calling GitHub API
    ghUser, err := h.githubService.ValidatePAT(c.Request().Context(), req.Token)
    if err != nil {
        return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid GitHub PAT")
    }

    // 2. Encrypt the PAT
    encrypted, nonce, err := crypto.EncryptWithNonce(req.Token, h.cfg.GitHubTokenEncryptionKey)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "encryption failed")
    }

    // 3. Update the teams table
    ghUsername := ghUser.GetLogin()
    now := time.Now().UTC()
    _, err = h.db.ExecContext(c.Request().Context(),
        `UPDATE teams
         SET github_pat_encrypted = $1,
             github_pat_nonce = $2,
             github_pat_set_at = $3,
             github_pat_set_by = $4,
             github_pat_github_username = $5
         WHERE id = $6`,
        encrypted, nonce, now, userID, ghUsername, teamID,
    )
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "failed to store PAT")
    }

    return c.JSON(http.StatusOK, map[string]interface{}{
        "status": "success",
        "data": map[string]interface{}{
            "provider":        req.Provider,
            "github_username": ghUsername,
            "set_at":          now,
            "set_by":          userID,
        },
    })
}
```

### Error Responses

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `VALIDATION_ERROR` | Missing provider or token |
| 401 | `UNAUTHORIZED` | User not authenticated |
| 403 | `FORBIDDEN` | User is not a team admin |
| 422 | `UNPROCESSABLE` | PAT is invalid or revoked by GitHub |
| 429 | `RATE_LIMITED` | PAT validation endpoint rate limit (5/min per user) |

---

## 3. Repository Addition

**Endpoint:** `POST /api/v1/repositories`

Users add repositories one at a time by providing a GitHub URL. Only `root`,
`moderator`, or team `admin` roles can add repositories.

### Step-by-Step

1. **Parse the GitHub URL.** Extract `owner` and `repo` from the URL:
   ```
   https://github.com/hashicorp/consul  ->  owner=hashicorp, repo=consul
   ```

2. **Retrieve the team's PAT** from the `teams` table via
   `repositories.team_id` -> `teams.github_pat_encrypted`. Decrypt it
   using `GITHUB_TOKEN_ENCRYPTION_KEY`. If no PAT is configured for the
   team, return `422 UNPROCESSABLE`.

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

5. **Enqueue a `repo.clone` worker job** to clone the repository to the
   local filesystem (see Section 4).

6. **Return the created repository:**
   ```json
   {
     "status": "success",
     "data": {
       "id": "uuid",
       "github_full_name": "hashicorp/consul",
       "name": "consul",
       "default_branch": "main",
       "clone_status": "pending",
       "team_id": "uuid"
     }
   }
   ```

### API

```
POST /api/v1/repositories

{
  "github_url": "https://github.com/hashicorp/consul",
  "team_id": "uuid"
}
```

### Database Migration

Add `local_path` and `clone_status` columns to the `repositories` table:

```sql
-- migrations/000012_add_repo_clone_fields.up.sql
ALTER TABLE repositories ADD COLUMN local_path TEXT;
ALTER TABLE repositories ADD COLUMN clone_status VARCHAR(32) NOT NULL DEFAULT 'pending';
-- clone_status values: 'pending', 'cloning', 'cloned', 'failed'
```

```sql
-- migrations/000012_add_repo_clone_fields.down.sql
ALTER TABLE repositories DROP COLUMN IF EXISTS local_path;
ALTER TABLE repositories DROP COLUMN IF EXISTS clone_status;
```

### Error Responses

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `VALIDATION_ERROR` | Invalid GitHub URL format |
| 401 | `UNAUTHORIZED` | User not authenticated |
| 403 | `FORBIDDEN` | User role insufficient (not root/moderator/admin) |
| 404 | `NOT_FOUND` | Repository does not exist on GitHub |
| 409 | `CONFLICT` | Repository already added |
| 422 | `UNPROCESSABLE` | Team PAT not configured, or GitHub returned 403 |

---

## 4. Local Repository Clone

When a repository is added, a worker job clones it to the local filesystem.
All subsequent branch and commit operations use this local clone, minimizing
GitHub API usage.

### Clone Destination

```
{VERDOX_REPO_BASE_PATH}/{remote}/{org}/{repo}
```

Examples:
- Dev default: `./data/repositories/github.com/hashicorp/consul`
- Production: `/var/lib/verdox/repositories/github.com/hashicorp/consul`

### Worker Job: `repo.clone`

```go
func (w *RepoCloneWorker) Execute(ctx context.Context, job *model.Job) error {
    repo, err := w.repoRepo.GetByID(ctx, job.RepoID)
    if err != nil {
        return fmt.Errorf("looking up repo: %w", err)
    }

    // 1. Decrypt the team's PAT
    team, err := w.teamRepo.GetByID(ctx, repo.TeamID)
    if err != nil {
        return fmt.Errorf("looking up team: %w", err)
    }
    pat, err := crypto.Decrypt(team.GitHubPATEncrypted, team.GitHubPATNonce, w.cfg.GitHubTokenEncryptionKey)
    if err != nil {
        return fmt.Errorf("decrypting team PAT: %w", err)
    }

    // 2. Build local path
    localPath := filepath.Join(
        w.cfg.RepoBasePath,
        "github.com",
        repo.GitHubFullName, // e.g. "hashicorp/consul"
    )

    // 3. Set clone_status = 'cloning'
    w.repoRepo.UpdateCloneStatus(ctx, repo.ID, "cloning")

    // 4. Clone the repository
    cloneURL := fmt.Sprintf(
        "https://x-access-token:%s@github.com/%s.git",
        pat, repo.GitHubFullName,
    )
    cmd := exec.CommandContext(ctx, "git", "clone",
        "--depth", "1",
        "--branch", repo.DefaultBranch,
        cloneURL, localPath,
    )
    cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

    if err := cmd.Run(); err != nil {
        w.repoRepo.UpdateCloneStatus(ctx, repo.ID, "failed")
        return fmt.Errorf("git clone failed: %w", err)
    }

    // 5. Update repository record
    w.repoRepo.UpdateLocalPath(ctx, repo.ID, localPath)
    w.repoRepo.UpdateCloneStatus(ctx, repo.ID, "cloned")

    return nil
}
```

### Clone Behavior

- Clone is a **shallow clone** (`--depth 1`) -- only the tip commit of the
  default branch is fetched. This saves disk space and speeds up the initial
  clone for large repositories.
- Clone is a **one-time operation** per repository.
- Subsequent branch/commit data comes from `git fetch` on the local clone,
  or the GitHub API where shallow clone limitations apply (see Section 6).
- Clone timeout: **60 seconds** (configurable).
- If the clone fails (network, disk space), `clone_status` is set to
  `'failed'` and the user can retry via the UI.

### Shallow Clone Limitations

Verdox uses `git clone --depth 1` by default for performance. This means
**git history is NOT available** inside the test container. The local clone
and mounted `/workspace` volume contain only the tip commit of the fetched
branch.

Tests or tools that depend on any of the following **will fail** under a
shallow clone:

- `git log` -- returns only the single fetched commit, not full history.
- `git blame` -- fails with `fatal: no such path` or returns incomplete data.
- `git diff` against historical commits -- fails because ancestor commits
  are not present in the shallow clone.
- Changelog generation tools (e.g., `conventional-changelog`,
  `git-cliff`) -- produce empty or incorrect output because they walk
  commit history.

**Workaround: `full_clone` override.** Set `full_clone: true` in
`verdox.yaml` at the repo level to use a full clone instead:

```yaml
version: 1
full_clone: true    # fetch full git history (default: false)

suites:
  - name: "Unit Tests"
    command: "go test -v -json ./..."
```

When `full_clone: true`:

- The initial clone runs **without** `--depth 1` and fetches the complete
  repository history.
- Subsequent `git fetch` operations for test runs also fetch full history
  (no `--depth 1` flag).
- `git log`, `git blame`, `git diff`, and changelog tools work as expected
  inside the test container.

When `full_clone: false` (the default):

- Shallow clone behavior is used, as described above.
- This is **recommended for most projects** -- it is faster, uses less disk
  space, and is sufficient when tests do not depend on git history.

### Disk Space Management

The `VERDOX_REPO_MAX_DISK_GB` environment variable controls the maximum disk
space allocated for local repository clones (default: 50 GB).

**Eviction behavior:**

- When disk usage for `VERDOX_REPO_BASE_PATH` reaches **90%** of
  `VERDOX_REPO_MAX_DISK_GB`, a background worker begins evicting repos using
  an **LRU policy** based on the last test run time (the repo whose most
  recent test run is oldest is evicted first).
- Evicted repos have their `clone_status` set to `evicted` in the database
  and their local clone directory is deleted from disk.
- On the next test trigger for an evicted repo, Verdox automatically
  re-clones it (transition: `evicted` -> `pending` -> `cloning` -> `ready`).

**Health check integration:**

- The readiness endpoint (`GET /api/v1/health/ready`) reports disk usage for
  `VERDOX_REPO_BASE_PATH`, including current usage in GB, the configured
  maximum, and the usage percentage.
- An alert fires when disk usage exceeds 90% of `VERDOX_REPO_MAX_DISK_GB`
  (see MONITORING.md).

---

## 5. Branch Listing (from local clone)

**Endpoint:** `GET /api/v1/repositories/:id/branches`

### Step-by-Step

1. **Look up the repository** in the database by UUID. Verify the requesting
   user has access. Get the `local_path`.

2. **Check Redis cache.** Look for cached branch data at key
   `branches:{repo_id}` (TTL: 5 minutes). If a cache hit, return the cached
   response immediately.

3. **List remote branches** directly from the remote (works with shallow
   clones without requiring a full fetch):
   ```bash
   git -C {local_path} ls-remote --heads origin
   ```
   This queries the remote for all branch refs and their SHAs. Unlike
   `git branch -r`, it does not require fetching all branches into the
   local clone.

4. **Parse the output.** Each line is tab-delimited:
   `{full_sha}\trefs/heads/{branch_name}`. Strip the `refs/heads/` prefix
   and truncate the SHA for display.

5. **Cache the result in Redis:**
   ```go
   branchJSON, _ := json.Marshal(branches)
   s.redis.Set(ctx, fmt.Sprintf("branches:%s", repoID), branchJSON, 5*time.Minute)
   ```

6. **Return the branch list:**
   ```json
   {
     "status": "success",
     "data": [
       {
         "name": "origin/main",
         "commit_sha": "a1b2c3d"
       },
       {
         "name": "origin/develop",
         "commit_sha": "f6e5d4c"
       }
     ]
   }
   ```

### Error Responses

| Status | Code | Condition |
|--------|------|-----------|
| 404 | `NOT_FOUND` | Repository does not exist in Verdox |
| 422 | `UNPROCESSABLE` | Repository not yet cloned (`clone_status != 'cloned'`) |
| 500 | `INTERNAL_ERROR` | Git command failed on local clone |

---

## 6. Commit Listing (from GitHub API)

**Endpoint:** `GET /api/v1/repositories/:id/commits?branch=main`

### Step-by-Step

1. **Validate the `branch` query parameter.** It is required; return
   `400 VALIDATION_ERROR` if missing.

2. **Check Redis cache.** Look for cached commit data at key
   `commits:{repo_id}:{branch}` (TTL: 2 minutes). If a cache hit, return
   immediately.

3. **Fetch commits from the GitHub API.** Since we use a shallow clone
   (`--depth 1`), `git log` only shows the single fetched commit. We call
   the GitHub API to get the full commit history for a branch:
   ```
   GET https://api.github.com/repos/{owner}/{repo}/commits?sha={branch}&per_page=20
   Authorization: Bearer {pat}
   ```
   This is the one case where we still need the GitHub API -- git log on a
   shallow clone only shows the one fetched commit.

4. **Parse the API response.** Each element contains `sha`,
   `commit.message`, `commit.author.name`, and `commit.author.date`.

5. **Cache the result in Redis:**
   ```go
   commitJSON, _ := json.Marshal(commits)
   s.redis.Set(ctx, fmt.Sprintf("commits:%s:%s", repoID, branch), commitJSON, 2*time.Minute)
   ```

6. **Return the commit list:**
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
| 422 | `UNPROCESSABLE` | Repository not yet cloned |
| 500 | `INTERNAL_ERROR` | GitHub API call failed |

---

## 7. Branch Fetch for Test Runs

When a test run is triggered on a non-default branch, the worker fetches
that branch into the local clone before execution.

### Step-by-Step

1. **Fetch only the tip commit** of the specific branch:
   ```bash
   git -C {local_path} fetch --depth 1 origin {branch}
   ```
   This fetches only the tip commit -- no full branch history needed.
   Timeout: 30 seconds.

2. **Worker uses the local clone** for test execution. The test runner
   checks out `FETCH_HEAD` (detached HEAD) and runs tests. See
   TEST-RUNNER LLD for details on the per-repo sequential queue.

3. **No branch cleanup needed.** Since the test runner uses `FETCH_HEAD`
   (detached HEAD) with a per-repo sequential queue, no local branches are
   created and no cleanup is required.

---

## 8. Repository Updates (Re-sync)

A "Re-sync" button in the UI allows users to pull the latest state from the
remote into the local clone.

### Step-by-Step

1. **Fetch all branches and prune stale remote-tracking references:**
   ```bash
   git -C {local_path} fetch --all --prune
   ```

2. **Check if the default branch has changed** by querying the GitHub API:
   ```
   GET https://api.github.com/repos/{owner}/{repo}
   Authorization: Bearer {pat}
   ```
   If `default_branch` differs, update the `repositories` row.

3. **No destructive operations** are performed on the local clone.
   Re-sync is additive only.

---

## 9. Repository Deletion

**Endpoint:** `DELETE /api/v1/repositories/:id`

### Soft Delete (default)

- Set `is_active = FALSE` on the `repositories` row.
- Keep the local clone directory for data retention purposes.
- The repository disappears from the UI but data is preserved.

### Hard Delete (admin action)

Triggered with `?hard=true` query parameter. Requires `root` role.

- Delete the `repositories` row from the database.
- Delete the local clone directory from the filesystem:
  ```go
  os.RemoveAll(repo.LocalPath)
  ```

### Error Responses

| Status | Code | Condition |
|--------|------|-----------|
| 403 | `FORBIDDEN` | Hard delete requires root role |
| 404 | `NOT_FOUND` | Repository does not exist |

---

## 10. GitHub API Rate Limiting

Since Verdox uses local clones for branch listing and shallow fetches for
test runs, GitHub API calls are minimal -- limited to initial repo validation,
clone authentication, commit listing (due to shallow clone), and periodic
re-sync.

### Rate Limits

- Authenticated PAT requests: **5,000/hour**.
- Typical usage per repo add: **1-2 API calls** (validate repo + optional
  default branch check).

### Tracking

- Read `X-RateLimit-Remaining` header from GitHub API responses during
  repository addition.
- Log a warning when remaining requests fall below **500**.
- Return a `429 RATE_LIMITED` error if remaining is **0**.

---

## 11. Error Handling

| Scenario | Detection | Response |
|----------|-----------|----------|
| Invalid PAT | 401 from GitHub during validation | Prompt team admin to update PAT |
| Repo not found | 404 from GitHub during repo addition | Return error to user |
| No repo access | 403 from GitHub | Team's PAT needs repo access on GitHub |
| Clone failure (disk/network) | `git clone` exit code != 0 | Set `clone_status='failed'`, allow retry |
| Git fetch timeout | Context deadline exceeded | Return 500, log error |
| Git operations timeout | Clone: 60s, Fetch: 30s | Cancel via context |

---

## 12. Environment Variables

```
VERDOX_REPO_BASE_PATH=./data/repositories         # Dev default
# Production: /var/lib/verdox/repositories

GITHUB_TOKEN_ENCRYPTION_KEY=<32-byte-hex>          # AES-256-GCM key for PAT encryption
# Generate: openssl rand -hex 32
```

No GitHub OAuth credentials (client ID, client secret, callback URL) are
required for the PAT-based approach.

---

## 13. Security Considerations

- **PATs encrypted at rest** with AES-256-GCM. The encryption key is loaded
  from the `GITHUB_TOKEN_ENCRYPTION_KEY` environment variable (32 bytes,
  hex-encoded).
- **PATs never logged, never returned in API responses.** Only metadata is
  returned: `provider`, `label`, `created_at`.
- **Local clone directories** are owned by the application user with mode
  `700` (owner-only access).
- **Git credentials never written to `.git/config`** -- they are passed via
  the clone URL or `GIT_ASKPASS` environment variable, never persisted.
- **Rate limit on PAT validation endpoint** -- 5 requests per minute per
  user to prevent brute-force token scanning.
- **Clone URLs use `x-access-token` scheme** -- the PAT is used as a
  password with the `x-access-token` username, which is GitHub's recommended
  approach for token-based HTTPS authentication.
- **Team-level PAT** is used for all git operations on repositories
  belonging to that team (clone, fetch, ls-remote, API calls for commit
  listing). PAT resolution: repo -> `repositories.team_id` ->
  `teams.github_pat_encrypted` -> decrypt -> use. If the PAT is revoked,
  any team admin can set a new PAT via team settings -- there is no
  single-user dependency.

---

## 14. Future Considerations (v2)

- **GitHub OAuth App** -- for browser-based token acquisition, removing the
  need for users to manually generate and paste PATs.
- **GitHub App** -- for org-level installation tokens, fine-grained
  permissions, and a dedicated bot identity (`verdox[bot]`).
- **Webhook support** -- for push-triggered test runs. Requires Verdox to
  be reachable from GitHub (public URL or tunnel).
- **Additional providers** -- GitLab, Bitbucket. Provider-specific PAT
  columns can be added to the `teams` table following the same pattern.
