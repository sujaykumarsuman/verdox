# Verdox Branching Strategy

> Git workflow rules for the Verdox repository.

---

## 1. Golden Rule

**Never push directly to `main`.** Every change reaches `main` through a pull
request — no exceptions.

---

## 2. Branch Naming Convention

All branches must follow the pattern:

```
{type}/{short-description}
```

Use kebab-case for the description. Keep it under 50 characters total.

| Prefix | When to use | Example |
|--------|-------------|---------|
| `feat/` | New feature or capability | `feat/team-pat-settings` |
| `fix/` | Bug fix discovered during development | `fix/jwt-refresh-race-condition` |
| `bug/` | Bug reported by a user or found in production | `bug/clone-status-stuck-pending` |
| `hotfix/` | Urgent production fix that needs immediate merge | `hotfix/active-lock-ttl-missing` |
| `chore/` | Tooling, config, dependencies, CI — no app logic change | `chore/upgrade-go-1.25` |
| `docs/` | Documentation only — no code changes | `docs/add-pat-guide` |
| `refactor/` | Code restructuring with no behavior change | `refactor/extract-queue-service` |
| `test/` | Adding or updating tests only | `test/auth-middleware-edge-cases` |
| `perf/` | Performance improvement | `perf/batch-insert-test-results` |

**Invalid branch names** (CI should reject these):

```
main                     # protected
sujay/stuff              # no type prefix
feat/Fix_Some_Thing      # no snake_case or PascalCase
feature/add-thing        # use feat/, not feature/
```

---

## 3. Workflow

```
main (protected)
 │
 ├── feat/team-crud ──────── PR #1 ──► main
 ├── fix/login-redirect ──── PR #2 ──► main
 ├── docs/update-api-spec ── PR #3 ──► main
 └── hotfix/db-migration ─── PR #4 ──► main
```

### Step-by-step

1. **Create branch** from latest `main`:
   ```bash
   git checkout main
   git pull origin main
   git checkout -b feat/team-pat-settings
   ```

2. **Commit often** with clear messages. Follow conventional commit style:
   ```
   feat: add team PAT storage endpoint

   - Encrypt PAT with AES-256-GCM before storing
   - Validate against GitHub API on save
   - Return masked token in status response
   ```

3. **Push and open PR**:
   ```bash
   git push -u origin feat/team-pat-settings
   gh pr create --title "feat: add team PAT storage endpoint" --body "..."
   ```

4. **Review and merge** — squash merge preferred for clean history.

5. **Delete branch** after merge (GitHub auto-delete recommended).

---

## 4. Commit Message Format

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
{type}({scope}): {description}

{optional body}
```

| Type | Purpose |
|------|---------|
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `chore` | Maintenance, deps, config |
| `refactor` | Code restructuring, no behavior change |
| `test` | Adding or updating tests |
| `perf` | Performance improvement |
| `ci` | CI/CD changes |
| `style` | Formatting, whitespace — no logic change |

**Scope** is optional but encouraged — use the module name:

```
feat(auth): add JWT refresh token rotation
fix(runner): release active lock on git fetch failure
chore(deps): bump echo to v4.13
docs(api): update team PAT endpoint spec
```

---

## 5. Branch Protection Rules (GitHub)

Configure these on the `main` branch:

| Rule | Setting |
|------|---------|
| Require pull request before merging | Yes |
| Required approvals | 1 (adjust per team size) |
| Dismiss stale reviews on new push | Yes |
| Require status checks to pass | Yes (CI build + lint) |
| Require branches to be up to date | Yes |
| Restrict direct pushes | Yes — no one pushes to main |
| Allow force pushes | Never |
| Allow deletions | No |
| Require linear history | Yes (squash merge) |

---

## 6. Merge Strategy

**Squash and merge** is the default for all PRs:

- Keeps `main` history linear and clean
- Each PR becomes one commit on `main`
- The squash commit message should follow conventional commit format

**When to use merge commit** (rare):

- Large PRs with meaningful intermediate commits worth preserving
- Release branches merging back to main

**Never rebase onto main** — use squash merge via GitHub UI.

---

## 7. Hotfix Process

For urgent production issues:

1. Branch from `main`: `hotfix/description`
2. Fix, commit, push
3. Open PR with `hotfix` label
4. Fast-track review (single approval sufficient)
5. Squash merge to `main`
6. Deploy immediately

Hotfixes skip the normal review queue but still require a PR — no direct
pushes.

---

## 8. Branch Lifecycle

| State | Action |
|-------|--------|
| Created | Developer creates branch from latest `main` |
| In progress | Developer commits and pushes regularly |
| PR opened | Ready for review — CI runs automatically |
| Approved | Reviewer approves, CI passes |
| Merged | Squash merged to `main`, branch auto-deleted |
| Stale | No commits for 14+ days — developer should close or rebase |

**Stale branch cleanup:** Branches with no activity for 14 days should be
evaluated. Close the PR if abandoned, or rebase and continue if still
relevant.
