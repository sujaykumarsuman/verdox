# GitHub PAT Guide for Verdox

> How to create, configure, and maintain a GitHub Personal Access Token for your
> Verdox team.

---

## 1. Overview

Verdox uses a **team-level GitHub Personal Access Token (PAT)** to clone
repositories and fetch branch updates. Each team in Verdox stores exactly one
PAT. All repositories added to that team use this PAT for git operations
(clone, fetch, ls-remote).

**Key principle:** The PAT belongs to the _team_, not to any individual user.
If the person who originally set the PAT leaves the organization, any team
admin can rotate it without disrupting other team members.

---

## 2. Which GitHub Account Should Own the PAT?

### Recommended: Machine User (Bot Account)

Create a dedicated GitHub account for your team's Verdox integration:

| Step | Action |
|------|--------|
| 1 | Create a new GitHub account (e.g., `acme-verdox-bot` or `{team}-ci-bot`) |
| 2 | Add it to your GitHub organization as a **member** |
| 3 | Grant it **read-only access** to the repositories your team uses in Verdox |
| 4 | Generate a fine-grained PAT from this account (see Section 3 below) |
| 5 | Enter the PAT in Verdox team settings |

**Why a machine user?**

- Not tied to any employee's personal GitHub account
- Survives team member departures — no single-person dependency
- Easy to audit — all Verdox git operations appear under one clear bot identity
- Can be scoped to exactly the repos needed — nothing more
- Follows [GitHub's official recommendation](https://docs.github.com/en/authentication/connecting-to-github-with-ssh/managing-deploy-keys#machine-users) for CI/automation

### Alternative: Team Lead's Personal Account

If creating a machine user isn't feasible (e.g., GitHub seat limits), a team
lead can use their personal account. Be aware this re-introduces the
single-person dependency. If they leave or revoke the PAT, a team admin must
set a new one.

---

## 3. Creating a Fine-Grained PAT (Recommended)

Fine-grained PATs offer repository-level scoping and minimal permissions.
GitHub recommends them over classic PATs.

### Step-by-Step

1. **Sign in** to the GitHub account that will own the PAT (ideally the machine
   user from Section 2).

2. **Navigate to token settings:**
   ```
   GitHub.com → Settings → Developer settings → Personal access tokens → Fine-grained tokens
   ```
   Direct URL: `https://github.com/settings/personal-access-tokens/new`

3. **Token name:**
   Use a descriptive name that identifies the Verdox team:
   ```
   verdox-{team-name}
   ```
   Example: `verdox-backend-team`, `verdox-platform`

4. **Expiration:**
   Set an expiration period. Recommendations:

   | Environment | Expiration | Rationale |
   |-------------|------------|-----------|
   | Production  | 90 days    | Balance between security and rotation frequency |
   | Development | 180 days   | Less sensitive, less rotation overhead |
   | Never       | --         | **Not recommended.** Use only if your org policy allows it and you have automated rotation reminders |

   Verdox displays a warning in the team dashboard when the PAT is within 14
   days of expiry.

5. **Resource owner:**
   Select the GitHub **organization** that owns the repositories. If the repos
   are under a personal account, select that account instead.

   > If the organization requires admin approval for fine-grained PATs, an org
   > admin must approve the token request after creation.

6. **Repository access:**
   Select **"Only select repositories"** and pick the specific repositories
   your team will use in Verdox.

   > **Do not** select "All repositories" unless the team genuinely needs access
   > to every repo in the org. Least-privilege is the goal.

7. **Permissions:**
   Set the following permissions and **nothing else**:

   | Category | Permission | Access Level | Why |
   |----------|-----------|--------------|-----|
   | **Contents** | Repository contents | **Read-only** | Required for `git clone`, `git fetch`, and reading files |
   | **Metadata** | Repository metadata | **Read-only** | Always required (automatically selected by GitHub) |

   That's it — only **2 permissions** needed. Verdox does not push code, create
   branches, manage webhooks, or modify any repository state.

   **Permissions NOT needed** (leave unchecked):

   | Permission | Why not needed |
   |-----------|----------------|
   | Actions | Verdox doesn't interact with GitHub Actions |
   | Administration | No repo settings changes |
   | Commit statuses | Verdox doesn't post commit statuses (v1) |
   | Deployments | No deployment integration |
   | Environments | No environment management |
   | Issues / Pull requests | Verdox doesn't read or create issues/PRs |
   | Pages | No GitHub Pages interaction |
   | Webhooks | Deferred to v2 |
   | Workflows | No workflow dispatch |

8. **Generate token:**
   Click **"Generate token"**. Copy the token immediately — GitHub will not
   show it again.

   ```
   github_pat_XXXXXXXXXXXXXXXXXXXX_XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
   ```

9. **Enter in Verdox:**
   Navigate to your team settings in Verdox:
   ```
   Verdox → Teams → {Your Team} → Settings → GitHub Integration
   ```
   Paste the PAT and click **Save**. Verdox will:
   - Validate the PAT against the GitHub API
   - Encrypt it with AES-256-GCM before storing
   - Never display the full token again (only shows `github_pat_...XXXX`)

---

## 4. Creating a Classic PAT (Fallback)

If your GitHub organization doesn't support fine-grained PATs (e.g., GitHub
Enterprise Server < 3.10), you can use a classic PAT instead.

### Step-by-Step

1. **Navigate to:**
   ```
   GitHub.com → Settings → Developer settings → Personal access tokens → Tokens (classic)
   ```
   Direct URL: `https://github.com/settings/tokens/new`

2. **Note:** `verdox-{team-name}`

3. **Expiration:** Same recommendations as Section 3.

4. **Scopes:** Select **only** the `repo` scope.

   | Scope | Required | Note |
   |-------|----------|------|
   | `repo` | Yes | Grants read/write access to repos. Unfortunately, classic PATs cannot scope to read-only repo access |
   | All others | No | Leave unchecked |

   > **Note:** Classic PATs grant broader access than fine-grained PATs. The
   > `repo` scope includes write access, which Verdox never uses but cannot
   > opt out of with classic tokens. This is why fine-grained PATs are
   > recommended.

5. **Generate and configure** in Verdox as described in Section 3, steps 8-9.

---

## 5. Configuring the PAT in Verdox

### Who Can Set the PAT?

Only **team admins** can set, update, or remove the team's GitHub PAT.
Maintainers and viewers cannot access the PAT settings.

| Action | Required Role |
|--------|--------------|
| Set / update PAT | Team admin |
| Remove PAT | Team admin |
| View PAT status (set/not set, expiry) | Team admin, maintainer |
| Use PAT (add repos, trigger runs) | Automatic — any team member adding repos or triggering runs uses the team PAT transparently |

### PAT Validation

When a PAT is saved, Verdox validates it by making an authenticated request to
the GitHub API:

```
GET https://api.github.com/user
Authorization: Bearer {pat}
```

If the response is `200 OK`, the PAT is valid. Verdox also extracts the GitHub
username and stores it alongside the PAT for display purposes (e.g., "PAT set
by @acme-verdox-bot").

If validation fails, the PAT is rejected with a descriptive error:
- `401` → Token is invalid or revoked
- `403` → Token doesn't have required permissions
- Network error → GitHub API unreachable

### What Verdox Stores

| Field | Value |
|-------|-------|
| `github_pat_encrypted` | AES-256-GCM encrypted PAT (never stored in plaintext) |
| `github_pat_nonce` | Unique nonce for decryption |
| `github_pat_set_at` | Timestamp of when the PAT was set |
| `github_pat_set_by` | Verdox user ID of the admin who set it |
| `github_pat_github_username` | GitHub username the PAT belongs to (for display) |

The encryption key is derived from the `GITHUB_TOKEN_ENCRYPTION_KEY` environment
variable set during Verdox deployment.

---

## 6. PAT Rotation

### When to Rotate

| Trigger | Action |
|---------|--------|
| Token approaching expiry | Verdox shows a warning banner 14 days before expiry |
| Team member departure | If the PAT owner (machine user or person) leaves, rotate immediately |
| Security incident | Revoke and regenerate if the token may have been exposed |
| Routine rotation | Rotate every 90 days as a best practice |

### How to Rotate

1. **Generate a new PAT** in GitHub (Section 3 or 4) with the same permissions
   and repository access.

2. **Update in Verdox:** Team Settings → GitHub Integration → paste new PAT →
   Save.

3. **Revoke the old PAT** in GitHub:
   ```
   GitHub.com → Settings → Developer settings → Personal access tokens → Delete old token
   ```

**Zero-downtime rotation:** Verdox uses the PAT at the moment of each git
operation. Updating the PAT in Verdox takes effect immediately — the next
clone or fetch will use the new token. There is no need to re-clone existing
repositories.

### Rotation Checklist

```
[ ] Generate new PAT in GitHub with same permissions
[ ] Update PAT in Verdox team settings
[ ] Verify: add a new repo or trigger a test run to confirm
[ ] Revoke old PAT in GitHub
[ ] Update any internal documentation noting the rotation date
```

---

## 7. Adding Repositories After PAT Setup

Once the team PAT is configured, any team member (admin or maintainer) can add
repositories:

1. Navigate to **Repositories** in the team dashboard
2. Click **Add Repository**
3. Paste the GitHub repository URL:
   ```
   https://github.com/your-org/your-repo
   ```
4. Verdox uses the team's PAT to:
   - Validate access to the repository
   - Shallow clone (`git clone --depth 1`) to local storage
   - Set `clone_status` to `ready` when complete

**Important:** The PAT must have access to the repository being added. If
using a fine-grained PAT with "Only select repositories", the repo must be
in the selected list. If it's not, either:
- Update the PAT's repository access in GitHub, or
- Generate a new PAT that includes the additional repository

---

## 8. Troubleshooting

### Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `401 Unauthorized` during clone/fetch | PAT is revoked or expired | Team admin must set a new PAT in team settings |
| `403 Forbidden` on a specific repo | PAT doesn't have access to that repo | Update the PAT's repository scope in GitHub to include the repo |
| `403 Resource not accessible by fine-grained PAT` | Missing required permission | Ensure the PAT has `Contents: Read-only` and `Metadata: Read-only` |
| Clone times out | Repository is very large | Expected for first clone; subsequent fetches are fast (`--depth 1`) |
| `404 Not Found` when adding repo | Repo is private and PAT doesn't have access, or URL is wrong | Verify the URL and PAT repository access |
| Rate limit exceeded (`429`) | Too many GitHub API requests | Fine-grained PATs allow 5,000 requests/hour — wait or check for runaway processes |
| `PAT not configured` error | Team has no PAT set | Team admin must configure a PAT in team settings before adding repos |
| Org approval required | Fine-grained PAT needs org admin approval | Ask your GitHub org admin to approve the pending token request |

### Checking PAT Health

In Verdox, navigate to **Team Settings → GitHub Integration** to see:

- Whether a PAT is configured
- GitHub username associated with the PAT
- When the PAT was last set
- Which Verdox admin set it

To test the PAT manually (outside Verdox):

```bash
curl -H "Authorization: Bearer github_pat_XXXX" \
     https://api.github.com/user
```

Expected response: `200 OK` with the GitHub user profile JSON.

To test repo access:

```bash
curl -H "Authorization: Bearer github_pat_XXXX" \
     https://api.github.com/repos/{owner}/{repo}
```

Expected response: `200 OK` with the repository metadata.

---

## 9. Security Best Practices

| Practice | Detail |
|----------|--------|
| **Use fine-grained PATs** | Scope to specific repos with read-only permissions. Classic PATs grant broader access than needed |
| **Use a machine user** | Decouple from personal accounts. One bot per team |
| **Set an expiration** | 90 days recommended. Never use "No expiration" in production |
| **Rotate proactively** | Don't wait for expiry. Rotate when team composition changes |
| **Least-privilege repos** | Only select the repos the team actually uses — don't select "All repositories" |
| **Never share PATs** | Don't send via Slack, email, or any messaging tool. Paste directly into the Verdox UI |
| **Revoke old tokens** | After rotation, always delete the old PAT from GitHub |
| **Audit access** | Periodically review the machine user's repository access in GitHub |
| **Monitor rate limits** | Fine-grained PATs have 5,000 req/hr. Verdox's usage is well within this, but monitor if running many repos |

### What Verdox Guarantees

- PATs are **encrypted at rest** with AES-256-GCM
- PATs are **never logged** — not in application logs, not in structured logs, not in error messages
- PATs are **never returned** in API responses after initial storage — only a masked preview (`github_pat_...XXXX`)
- PATs are **never exposed to DinD containers** — local clones are mounted read-only; git operations happen on the host side via the worker
- PATs are **decrypted only at the moment of use** — held in memory briefly for the git operation, then discarded

---

## 10. FAQ

**Q: Can different repos in the same team use different PATs?**
A: Not in v1. Each team has one PAT, and all repos in that team use it. If you
need repos from different GitHub orgs with different access, create separate
Verdox teams.

**Q: What happens if I don't set a PAT for my team?**
A: You won't be able to add repositories. Verdox requires a valid PAT before
any repository operations.

**Q: Can viewers see the PAT?**
A: No. Only team admins can access PAT settings. Maintainers can see whether
a PAT is configured (set/not set) but cannot view or modify it.

**Q: Does rotating the PAT require re-cloning repos?**
A: No. Existing clones on disk remain valid. The new PAT is used only for
future git operations (fetch, ls-remote). No re-clone needed.

**Q: What if the GitHub org requires approval for fine-grained PATs?**
A: After generating the token, a GitHub org admin must approve it at
`https://github.com/organizations/{org}/settings/personal-access-token-requests`.
The token won't work until approved.

**Q: Can I use a GitHub App instead of a PAT?**
A: Not in v1. GitHub App installation tokens (short-lived, auto-rotated) are
planned for v2. For now, use a fine-grained PAT with a machine user.

**Q: How do I know when my PAT is about to expire?**
A: Verdox shows a warning banner in the team dashboard 14 days before the PAT
expires. Team admins also see the expiry status in team settings.
