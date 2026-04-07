# GitHub PAT Guide for Verdox

> How to create, configure, and maintain GitHub Personal Access Tokens for
> Verdox.

---

## 1. Overview

Verdox uses two types of GitHub PATs:

1. **Service account PAT (required)** -- used by Verdox to fork repositories
   and run tests via GitHub Actions. Configured in `.env`.
2. **Team PAT (optional)** -- used for accessing private repositories that
   the service account cannot see. Configured per-team in the Verdox UI.

**Key principle:** The service account PAT is the primary credential for all
fork and workflow operations. Team PATs are an optional layer for private repo
access. Both are stored securely -- the service account PAT in `.env` (never
exposed), team PATs encrypted with AES-256-GCM in the database.

---

## 2. Service Account PAT (Required)

The service account PAT is used by Verdox for:

- **Forking repositories** under the service account
- **Pushing workflow files** (`verdox-test.yml`) to forks
- **Dispatching GitHub Actions workflows** via `workflow_dispatch`
- **Polling workflow run status** and downloading logs/artifacts
- **Syncing forks** with upstream repositories

### Step-by-Step Setup

1. **Create a dedicated GitHub account** for the service account. Use a clear
   name like `verdox-bot`, `yourorg-verdox-ci`, or `{team}-ci-bot`.

2. **Add the service account to your GitHub organization** as a member. Grant
   it read access to the repositories your teams will test.

3. **Generate a classic PAT** on the service account:
   - Sign in to the service account on GitHub.
   - Navigate to: **Settings** > **Developer settings** > **Personal access tokens** > **Tokens (classic)**.
   - Click **Generate new token (classic)**.
   - **Token name:** `verdox-service-account`
   - **Expiration:** 90 days recommended.
   - **Select the following scopes:**

   | Scope | Required | Why |
   |-------|----------|-----|
   | `repo` | Yes | Fork repositories, push workflow files, access repo contents |
   | `workflow` | Yes | Dispatch and manage GitHub Actions workflows on forks |
   | `read:org` | Yes | Read organization membership to access private repos within orgs |

   - Click **Generate token** and copy it immediately.

4. **Configure in `.env`:**
   ```
   VERDOX_SERVICE_ACCOUNT_PAT=ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
   VERDOX_SERVICE_ACCOUNT_USERNAME=verdox-bot
   ```

5. **Verify** by making a test API call:
   ```bash
   curl -H "Authorization: Bearer ghp_xxxxxxxxxxxx" \
        https://api.github.com/user
   ```
   Expected: `200 OK` with the service account's profile.

### Why a Dedicated Service Account?

- **Not tied to any employee's personal GitHub account** -- survives team
  member departures.
- **Clear audit trail** -- all fork and workflow operations appear under one
  bot identity.
- **Scoped access** -- the service account only has access to repos it needs.
- **Follows GitHub's recommendation** for CI/automation use cases.

### Service Account PAT Rotation

| Trigger | Action |
|---------|--------|
| Token approaching expiry | Generate a new PAT, update `.env`, restart Verdox |
| Security incident | Revoke immediately, generate new PAT, update `.env` |
| Routine rotation | Every 90 days |

**Steps:**
1. Generate a new PAT on the service account with the same scopes.
2. Update `VERDOX_SERVICE_ACCOUNT_PAT` in `.env`.
3. Restart the Verdox backend.
4. Revoke the old PAT in GitHub.

---

## 3. Team PAT (Optional)

The team PAT is an optional, per-team credential for accessing private
repositories that the service account cannot see.

**When is a team PAT needed?**

- The service account does not have access to a private repository.
- The repository is in a different GitHub organization.
- You want to use a different identity for repo validation/listing.

**When is a team PAT NOT needed?**

- The service account has access to all repos the team uses.
- All repositories are public.

### Which GitHub Account Should Own the Team PAT?

#### Recommended: Machine User (Bot Account)

Create a dedicated GitHub account for your team's Verdox integration:

| Step | Action |
|------|--------|
| 1 | Create a new GitHub account (e.g., `acme-verdox-bot` or `{team}-ci-bot`) |
| 2 | Add it to your GitHub organization as a **member** |
| 3 | Grant it **read-only access** to the repositories your team uses in Verdox |
| 4 | Generate a PAT from this account (see Section 4 below) |
| 5 | Enter the PAT in Verdox team settings |

#### Alternative: Team Lead's Personal Account

If creating a machine user is not feasible, a team lead can use their personal
account. Be aware this re-introduces a single-person dependency.

---

## 4. Creating a Fine-Grained Team PAT (Recommended)

Fine-grained PATs offer repository-level scoping and minimal permissions.
GitHub recommends them over classic PATs.

### Step-by-Step

1. **Sign in** to the GitHub account that will own the PAT (ideally a machine
   user).

2. **Navigate to token settings:**
   ```
   GitHub.com > Settings > Developer settings > Personal access tokens > Fine-grained tokens
   ```
   Direct URL: `https://github.com/settings/personal-access-tokens/new`

3. **Token name:**
   Use a descriptive name: `verdox-{team-name}`

4. **Expiration:**

   | Environment | Expiration | Rationale |
   |-------------|------------|-----------|
   | Production  | 90 days    | Balance between security and rotation frequency |
   | Development | 180 days   | Less sensitive, less rotation overhead |

   Verdox displays a warning in the team dashboard when the PAT is within 14
   days of expiry.

5. **Resource owner:**
   Select the GitHub **organization** that owns the repositories.

6. **Repository access:**
   Select **"Only select repositories"** and pick the specific repositories
   your team will use in Verdox.

7. **Permissions:**
   Set the following permissions and **nothing else**:

   | Category | Permission | Access Level | Why |
   |----------|-----------|--------------|-----|
   | **Contents** | Repository contents | **Read-only** | Required for repo validation and branch/commit listing |
   | **Metadata** | Repository metadata | **Read-only** | Always required (automatically selected) |

   **Permissions NOT needed for the team PAT** (leave unchecked):

   | Permission | Why not needed |
   |-----------|----------------|
   | Actions | Handled by the service account PAT |
   | Workflows | Handled by the service account PAT |
   | Administration | No repo settings changes |
   | Commit statuses | Verdox does not post commit statuses (v1) |

8. **Generate token:** Click **"Generate token"** and copy immediately.

9. **Enter in Verdox:**
   Navigate to your team settings: **Verdox > Teams > {Your Team} > Settings > GitHub Integration**.
   Paste the PAT and click **Save**. Verdox will validate, encrypt, and store it.

---

## 5. Creating a Classic Team PAT (Fallback)

If your GitHub organization does not support fine-grained PATs, you can use a
classic PAT instead.

### Step-by-Step

1. **Navigate to:**
   ```
   GitHub.com > Settings > Developer settings > Personal access tokens > Tokens (classic)
   ```

2. **Note:** `verdox-{team-name}`

3. **Expiration:** Same recommendations as Section 4.

4. **Scopes:** Select **only** the `repo` scope.

   | Scope | Required | Note |
   |-------|----------|------|
   | `repo` | Yes | Grants read/write access to repos. Classic PATs cannot scope to read-only |
   | All others | No | Leave unchecked |

5. **Generate and configure** in Verdox as described in Section 4, steps 8-9.

---

## 6. Configuring the Team PAT in Verdox

### Who Can Set the Team PAT?

Only **team admins** can set, update, or remove the team's GitHub PAT.

| Action | Required Role |
|--------|--------------|
| Set / update PAT | Team admin |
| Remove PAT | Team admin |
| View PAT status (set/not set, expiry) | Team admin, maintainer |
| Use PAT (add repos) | Automatic -- any team member adding repos uses the team PAT transparently |

### PAT Validation

When a PAT is saved, Verdox validates it by making an authenticated request to
the GitHub API:

```
GET https://api.github.com/user
Authorization: Bearer {pat}
```

If the response is `200 OK`, the PAT is valid. Verdox also extracts the GitHub
username for display purposes.

### What Verdox Stores

| Field | Value |
|-------|-------|
| `github_pat_encrypted` | AES-256-GCM encrypted PAT (never stored in plaintext) |
| `github_pat_nonce` | Unique nonce for decryption |
| `github_pat_set_at` | Timestamp of when the PAT was set |
| `github_pat_set_by` | Verdox user ID of the admin who set it |
| `github_pat_github_username` | GitHub username the PAT belongs to (for display) |

---

## 7. PAT Rotation

### When to Rotate

| Trigger | Action |
|---------|--------|
| Token approaching expiry | Verdox shows a warning banner 14 days before expiry |
| Team member departure | If the PAT owner leaves, rotate immediately |
| Security incident | Revoke and regenerate if the token may have been exposed |
| Routine rotation | Rotate every 90 days as a best practice |

### How to Rotate (Team PAT)

1. **Generate a new PAT** in GitHub with the same permissions and repository
   access.
2. **Update in Verdox:** Team Settings > GitHub Integration > paste new PAT > Save.
3. **Revoke the old PAT** in GitHub.

### How to Rotate (Service Account PAT)

1. **Generate a new PAT** on the service account with the same scopes
   (`repo`, `workflow`, `read:org`).
2. **Update `.env`:** Set `VERDOX_SERVICE_ACCOUNT_PAT` to the new token.
3. **Restart the Verdox backend.**
4. **Revoke the old PAT** in GitHub.

---

## 8. Adding Repositories After PAT Setup

Once the service account is configured (and optionally a team PAT), any team
member (admin or maintainer) can add repositories:

1. Navigate to **Repositories** in the team dashboard.
2. Click **Add Repository**.
3. Paste the GitHub repository URL:
   ```
   https://github.com/your-org/your-repo
   ```
4. Verdox validates the repository using the available PAT (team PAT if
   configured, otherwise service account PAT).
5. The repository is added. Fork creation happens on the first test run.

**Important:** The service account must have access to the repository (or the
team PAT must) for Verdox to validate and later fork it. If the repository is
private, ensure the service account is added as a collaborator or org member.

---

## 9. Troubleshooting

### Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `401 Unauthorized` during fork/dispatch | Service account PAT is revoked or expired | Update `VERDOX_SERVICE_ACCOUNT_PAT` in `.env` and restart |
| `403 Forbidden` on fork | Service account lacks access to the repo | Add the service account as a collaborator or org member |
| `403 Resource not accessible` | Missing required PAT scope | Ensure service account PAT has `repo`, `workflow`, `read:org` scopes |
| `404 Not Found` when adding repo | Repo is private and no PAT has access | Configure a team PAT with access, or add the service account to the repo |
| `422 Validation failed` on dispatch | Workflow file not found on fork | Verdox will auto-push the workflow; if persistent, check fork status |
| Rate limit exceeded (`429`) | Too many GitHub API requests | Wait for rate limit reset; consider distributing load across team PATs |
| `PAT not configured` error | Neither service account nor team PAT has access | Set up `VERDOX_SERVICE_ACCOUNT_PAT` in `.env` |
| Org approval required | Fine-grained PAT needs org admin approval | Ask your GitHub org admin to approve the pending token request |

### Checking PAT Health

**Service account PAT:**
```bash
curl -H "Authorization: Bearer $VERDOX_SERVICE_ACCOUNT_PAT" \
     https://api.github.com/user
```

**Team PAT (via Verdox UI):**
Navigate to **Team Settings > GitHub Integration** to see PAT status.

---

## 10. Security Best Practices

| Practice | Detail |
|----------|--------|
| **Use a dedicated service account** | Never use a personal account for the service account PAT |
| **Use fine-grained PATs for teams** | Scope to specific repos with read-only permissions |
| **Set an expiration** | 90 days recommended for both PAT types |
| **Rotate proactively** | Do not wait for expiry. Rotate when team composition changes |
| **Least-privilege repos** | Only grant access to repos the team actually uses |
| **Never share PATs** | Do not send via Slack, email, or messaging tools |
| **Revoke old tokens** | After rotation, always delete the old PAT from GitHub |
| **Audit access** | Periodically review the service account's repository access |

### What Verdox Guarantees

- **Service account PAT** is stored in `.env` (server-side only, never exposed
  to the frontend or logged).
- **Team PATs** are encrypted at rest with AES-256-GCM.
- PATs are **never logged** -- not in application logs, not in structured
  logs, not in error messages.
- PATs are **never returned** in API responses -- only a masked preview
  (`github_pat_...XXXX`).
- PATs are **decrypted only at the moment of use** -- held in memory briefly,
  then discarded.
- **Fork isolation** -- tests run on a fork owned by the service account.
  The fork cannot push to upstream.

---

## 11. FAQ

**Q: What is the difference between the service account PAT and the team PAT?**
A: The service account PAT is required and handles all fork/workflow operations
(forking repos, pushing workflows, dispatching GHA runs). The team PAT is
optional and is only needed for accessing private repos that the service
account cannot see.

**Q: Can I skip the team PAT entirely?**
A: Yes, if the service account has access to all repositories your teams use.
The team PAT is only needed when the service account cannot access certain
private repos.

**Q: Can different repos in the same team use different PATs?**
A: Not in v1. Each team has one team PAT (optional), and all repos in that
team use it for validation/listing. The service account PAT is shared across
all teams. If you need repos from different GitHub orgs with different access,
create separate Verdox teams.

**Q: What happens if I do not set a team PAT?**
A: The service account PAT is used for everything. If the service account has
access to the repos, no team PAT is needed.

**Q: Does rotating a PAT affect existing forks?**
A: No. Forks persist on GitHub independently of the PAT. The new PAT is used
for future operations (sync, dispatch, poll). No re-fork is needed.

**Q: What if the GitHub org requires approval for fine-grained PATs?**
A: After generating the token, a GitHub org admin must approve it at
`https://github.com/organizations/{org}/settings/personal-access-token-requests`.

**Q: Can I use a GitHub App instead of a PAT?**
A: Not in v1. GitHub App installation tokens (short-lived, auto-rotated) are
planned for v2.

**Q: How do I know when my PAT is about to expire?**
A: Verdox shows a warning banner in the team dashboard 14 days before the team
PAT expires. For the service account PAT, set a calendar reminder to rotate
before expiry.
