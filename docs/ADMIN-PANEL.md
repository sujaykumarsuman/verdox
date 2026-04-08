# Verdox -- Admin Panel Design Document

> Go 1.26+ | Echo v4 | Next.js 15 | PostgreSQL 17 | Redis 7

This document defines the admin panel for the Verdox platform. It covers access
control, user management, system statistics, UI components, and the required
database migration. All behavior described here is implementation-ready and
references the canonical API endpoints, database schema, and frontend routing
defined in the existing LLD documents.

---

## 1. Admin Access Levels

The admin panel is served at `/admin` within the `DashboardLayout`. Access is
restricted by the `user_role` enum (`root`, `moderator`, `user`) stored in
the `users` table.

| Role | Capabilities |
|------|-------------|
| `root` | Everything: view all users, promote users to moderator, manage all users, deactivate/reactivate any user, view system stats. The root user is bootstrapped from environment variables (`ROOT_EMAIL`, `ROOT_PASSWORD`) on first startup -- there is no database seed. |
| `moderator` | View all users, deactivate/reactivate non-root users, view system stats. Cannot change user roles. |
| `user` | No admin access. The `/admin` route redirects to `/dashboard` via Next.js middleware. API endpoints return `403 FORBIDDEN`. |

### Enforcement layers

1. **Next.js middleware** (`src/middleware.ts`): parses the JWT from the
   `access_token` cookie, reads the `role` claim, and redirects non-admin users
   to `/dashboard`.
2. **AdminPage component**: checks the user role from `AuthContext` and renders
   a 403 fallback message if the role check fails (defense in depth).
3. **API middleware** (Go/Echo): every `/api/v1/admin/*` handler validates the
   JWT role claim server-side and returns `403 FORBIDDEN` for unauthorized
   roles. This is the authoritative check.

### Promote to Moderator (root only)

The user table includes a "Promote to Moderator" action button, visible only
to the `root` user.

- **Visibility:** The button is rendered in the Actions column only when the
  current user's role is `root` and the target user's role is `user`.
- **API:** `PUT /api/v1/admin/users/:id` with body `{ "role": "moderator" }`.
- **Confirmation modal:**

| Property | Value |
|----------|-------|
| Title | Promote to Moderator |
| Description | "Promote {username} to moderator? They will be able to create teams and manage repositories." |
| Confirm button | "Promote" (default variant) |
| Cancel button | "Cancel" |

---

## 2. User Management

### 2.1 User List View

The user list is rendered by the `UserTable` client component and fetches data
from `GET /api/v1/admin/users`.

**Table columns:**

| # | Column | Source field | Notes |
|---|--------|------------|-------|
| 1 | Avatar | `avatar_url` | 32x32 rounded image, fallback to initials |
| 2 | Username | `username` | Sortable |
| 3 | Email | `email` | Sortable |
| 4 | Role | `role` | `RoleDropdown` for root, plain `Badge` for moderator |
| 5 | Status | `is_active` | `StatusToggle` switch (active/inactive) |
| 6 | Created At | `created_at` | Sortable, formatted as relative date |
| 7 | Actions | -- | Role dropdown + status toggle (see sections below) |

**Pagination:**

- Default `per_page`: 20
- Max `per_page`: 100
- 1-based page numbers
- Pagination controls at bottom of table (previous/next + page indicator)
- Query: `GET /api/v1/admin/users?page=N&per_page=20`

**Search:**

- Single text input above the table
- Searches across `username` and `email` (case-insensitive, server-side)
- Debounced at 300ms before sending the request
- Resets to page 1 on new search
- Query parameter: `search=<term>`

**Filters:**

| Filter | Control | Query parameter | Options |
|--------|---------|----------------|---------|
| Role | Dropdown | `role` | All (default), `root`, `moderator`, `user` |
| Status | Dropdown | `status` | All (default), `active`, `inactive` |

**Sorting:**

- Clickable column headers for Username, Email, Created At
- Toggle between `asc` and `desc` on click
- Query parameters: `sort=<field>&order=<asc|desc>`
- Default: `sort=created_at&order=desc`

---

### 2.2 User Actions

#### Change Role (root only)

- **Control:** `RoleDropdown` rendered inline in the table row.
- **Visibility:** Only rendered for users with `role === 'root'`. Moderator
  users see a plain text `Badge` showing the role (read-only).
- **Options:** `user`, `moderator`. The `root` role cannot be assigned via the
  UI -- it is env-based only.
- **Disabled states:**
  - The dropdown for the current user's own row is disabled (cannot change own
    role).
- **Flow:**
  1. root selects a new role from the dropdown.
  2. Confirmation modal appears (see section 2.3).
  3. On confirm, send `PUT /api/v1/admin/users/:id` with body `{ "role": "<new_role>" }`.
  4. On success, update the row in-place and show a success toast.
- **API:** `PUT /api/v1/admin/users/:id`
- **Server guard:** The API handler verifies the caller is `root` and
  verifies the target is not the caller.

#### Deactivate User (root, moderator)

- **Control:** `StatusToggle` switch rendered inline in the table row.
- **Disabled states:**
  - The toggle for the current user's own row is disabled (cannot deactivate
    yourself).
  - For moderator callers, the toggle is disabled on rows where the target
    user's role is `root` (moderator cannot deactivate root).
- **Flow:**
  1. Moderator/root flips the toggle from active to inactive.
  2. Confirmation modal appears (see section 2.3).
  3. On confirm, send `PUT /api/v1/admin/users/:id` with body `{ "is_active": false }`.
  4. On success, update the row and show a success toast.
  5. Server-side: all sessions for the deactivated user are deleted from the
     `sessions` table. The user cannot log in until reactivated.
- **API:** `PUT /api/v1/admin/users/:id`

#### Reactivate User (root, moderator)

- Same toggle as deactivation, flipped from inactive to active.
- No confirmation modal required for reactivation.
- Send `PUT /api/v1/admin/users/:id` with body `{ "is_active": true }`.
- On success, the user can log in again.

---

### 2.3 Confirmation Modals

All destructive admin actions require explicit confirmation via the
`ConfirmModal` component.

**Role change modal:**

| Property | Value |
|----------|-------|
| Title | Change User Role |
| Description | "Change {username}'s role from {current_role} to {new_role}? This will affect their permissions immediately." |
| Confirm button | "Change Role" (danger variant) |
| Cancel button | "Cancel" |

**Deactivation modal:**

| Property | Value |
|----------|-------|
| Title | Deactivate User |
| Description | "Deactivate {username}? They will be logged out and unable to sign in until reactivated." |
| Confirm button | "Deactivate" (danger variant) |
| Cancel button | "Cancel" |

---

## 3. System Stats Dashboard

The stats dashboard is displayed as a grid of `StatsCard` components at the top
of the `AdminPage`, above the `UserTable`. Data is fetched from
`GET /api/v1/admin/stats` on page load.

### 3.1 API Response

```json
{
  "data": {
    "total_users": 42,
    "active_users": 38,
    "total_repositories": 15,
    "total_teams": 5,
    "teams_with_pat": 4,
    "total_test_runs": 230,
    "test_runs_today": 12,
    "active_runners": 3,
    "queue_depth": 2,
    "pass_rate_7d": 0.87
  }
}
```

### 3.2 Stat Cards

Cards are rendered in a responsive grid: `grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4`.

| Card | Label | Value | Format | Color coding |
|------|-------|-------|--------|-------------|
| 1 | Total Users | `total_users` (`active_users` active) | "42 (38 active)" | -- |
| 2 | Repositories | `total_repositories` | "15" | -- |
| 3 | Teams | `total_teams` (`teams_with_pat` with PAT) | "5 (4 with PAT)" | -- |
| 4 | Test Runs Today | `test_runs_today` / `total_test_runs` | "12 / 230" | -- |
| 5 | Runners / Queue | `active_runners` / `queue_depth` | "3 running, 2 queued" | Queue > 10: `--semantic-warning` |
| 6 | 7-Day Pass Rate | `pass_rate_7d` | "87%" | >= 90%: `--semantic-success`, 70-89%: `--semantic-warning`, < 70%: `--semantic-error` |

### 3.3 Stats Query (Backend)

The stats endpoint aggregates data across multiple tables in a single handler:

```sql
-- Total and active users
SELECT
  COUNT(*)                          AS total_users,
  COUNT(*) FILTER (WHERE is_active) AS active_users
FROM users;

-- Total repositories
SELECT COUNT(*) AS total_repositories FROM repositories;

-- Total teams and teams with PAT configured
SELECT
  COUNT(*)                                                  AS total_teams,
  COUNT(*) FILTER (WHERE github_pat_encrypted IS NOT NULL)  AS teams_with_pat
FROM teams;

-- Test run stats
SELECT
  COUNT(*)                                                        AS total_test_runs,
  COUNT(*) FILTER (WHERE created_at >= CURRENT_DATE)              AS test_runs_today,
  COUNT(*) FILTER (WHERE status = 'running')                      AS active_runners,
  COUNT(*) FILTER (WHERE status = 'queued')                       AS queue_depth
FROM test_runs;

-- 7-day pass rate
SELECT
  CASE WHEN COUNT(*) = 0 THEN 0
       ELSE ROUND(
         COUNT(*) FILTER (WHERE status = 'passed')::NUMERIC
         / COUNT(*)::NUMERIC, 2
       )
  END AS pass_rate_7d
FROM test_runs
WHERE status IN ('passed', 'failed')
  AND created_at >= NOW() - INTERVAL '7 days';
```

---

## 4. Audit Log (Future Enhancement)

For v1, all admin actions (role changes, activations, deactivations) are logged
via structured logging using `zerolog`. Each log entry includes:

| Field | Description |
|-------|-------------|
| `actor_id` | UUID of the admin performing the action |
| `actor_role` | Role of the admin |
| `action` | One of: `role_change`, `user_deactivate`, `user_reactivate` |
| `target_id` | UUID of the affected user |
| `details` | JSON object with before/after values (e.g., `{"role_from": "user", "role_to": "moderator"}`) |
| `timestamp` | ISO 8601 timestamp |

### v2 Audit Log (planned)

A future iteration may introduce a dedicated `audit_logs` table:

```sql
CREATE TABLE audit_logs (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id   UUID        NOT NULL REFERENCES users (id),
    action     VARCHAR(64) NOT NULL,
    target_id  UUID,
    details    JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_actor_id   ON audit_logs (actor_id);
CREATE INDEX idx_audit_logs_action     ON audit_logs (action);
CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at);
```

v2 capabilities:
- Searchable admin action history in the admin panel UI
- Filterable by actor, action type, and target user
- Paginated list view with date range selection

---

## 5. UI Components

All components follow the design tokens defined in `BRAND-PALETTE.md`.

### 5.1 Component Tree

```
AdminPage
|-- StatsCards (grid)
|   +-- StatsCard x6
|
+-- UserTable
    |-- SearchInput (debounced, 300ms)
    |-- FilterBar
    |   |-- RoleFilter (dropdown)
    |   +-- StatusFilter (dropdown)
    |
    |-- Table
    |   |-- SortableColumnHeader (Username, Email, Created At)
    |   +-- TableRow (per user)
    |       |-- Avatar
    |       |-- RoleDropdown (root) or Badge (moderator)
    |       +-- StatusToggle
    |
    +-- Pagination
        |-- PreviousButton
        |-- PageIndicator ("Page 1 of 3")
        +-- NextButton

ConfirmModal (shared, rendered via portal)
```

### 5.2 Component Specifications

#### AdminPage

| Property | Value |
|----------|-------|
| File | `src/app/(dashboard)/admin/page.tsx` |
| Rendering | Server component for initial stats fetch, client component for `UserTable` |
| Layout | `StatsCards` grid at top, `UserTable` below, separated by `mb-8` |

#### StatsCard

| Property | Value |
|----------|-------|
| File | `src/components/admin/stats-card.tsx` |
| Props | `icon: LucideIcon`, `label: string`, `value: string`, `trend?: { direction: 'up' | 'down' | 'neutral', color: string }` |
| Styling | `bg-bg-secondary`, `rounded-lg`, `p-4`. Value in `type-h2` (DM Serif Display), label in `type-body-sm` (DM Sans). |

#### UserTable

| Property | Value |
|----------|-------|
| File | `src/components/admin/user-table.tsx` |
| Type | Client component (`'use client'`) |
| State | `page`, `search`, `roleFilter`, `statusFilter`, `sort`, `order` |
| Data fetching | `useSWR` or `useQuery` with query params from state |
| Loading | 8 skeleton rows with shimmer animation |
| Empty | "No users found." centered in table body |

#### RoleDropdown

| Property | Value |
|----------|-------|
| File | `src/components/admin/role-dropdown.tsx` |
| Props | `userId: string`, `currentRole: UserRole`, `disabled: boolean`, `onRoleChange: (role: UserRole) => void` |
| Options | `user`, `moderator` (root cannot be assigned via UI -- it is env-based only) |
| Disabled when | Viewing own row, caller is not root |
| Styling | Compact select, `border-border`, `rounded-md` |

#### StatusToggle

| Property | Value |
|----------|-------|
| File | `src/components/admin/status-toggle.tsx` |
| Props | `userId: string`, `isActive: boolean`, `disabled: boolean`, `onToggle: (isActive: boolean) => void` |
| Disabled when | Viewing own row, moderator targeting a root user |
| Styling | Switch component, active: `bg-semantic-success`, inactive: `bg-semantic-error` |

#### ConfirmModal

| Property | Value |
|----------|-------|
| File | `src/components/ui/confirm-modal.tsx` |
| Props | `open: boolean`, `title: string`, `description: string`, `confirmLabel: string`, `variant: 'danger' | 'default'`, `onConfirm: () => void`, `onCancel: () => void`, `loading: boolean` |
| Danger variant | Confirm button uses `bg-semantic-error` with white text |
| Behavior | Closes on cancel, escape key, or overlay click. Confirm button shows spinner while `loading` is true. |

---

## 6. Frontend Route Protection

### 6.1 Middleware (`src/middleware.ts`)

The Next.js middleware intercepts all requests to `/admin` routes:

```typescript
const adminRoutes = ['/admin'];

// Inside middleware function:
if (adminRoutes.some((route) => pathname.startsWith(route))) {
  const userRole = parseRoleFromToken(accessToken);
  if (userRole !== 'root' && userRole !== 'moderator') {
    return NextResponse.redirect(new URL('/dashboard', request.url));
  }
}
```

`parseRoleFromToken` decodes the JWT payload (without verification -- the
server verifies on API calls) and reads the `role` claim. This provides a fast
client-side redirect without a round trip.

### 6.2 Sidebar Navigation

The "Admin" nav item in the sidebar is conditionally rendered:

| Icon | Label | Href | Visible To |
|------|-------|------|------------|
| `Shield` | Admin | `/admin` | `moderator`, `root` only |

The sidebar reads the user role from `AuthContext` and omits the admin link
for users with `role === 'user'`.

### 6.3 TopBar User Menu

The "Admin Panel" menu item in the user dropdown is also conditionally
rendered, visible only to `moderator` and `root` users.

### 6.4 Server-Side API Protection

All `/api/v1/admin/*` endpoints enforce role checks in the Echo middleware
chain:

```go
adminGroup := api.Group("/admin")
adminGroup.Use(middleware.RequireRole("root", "moderator"))
```

Individual handlers add further restrictions (e.g., the role-change logic in
`PUT /api/v1/admin/users/:id` requires `root`).

---

## 7. Database Changes

The `users` table currently does not have an `is_active` column. This must be
added via a new migration before the admin panel deactivation feature can work.

### 7.1 Migration

**File:** `migrations/000011_add_users_is_active.up.sql`

```sql
ALTER TABLE users ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE;
CREATE INDEX idx_users_is_active ON users (is_active);
```

**File:** `migrations/000011_add_users_is_active.down.sql`

```sql
DROP INDEX IF EXISTS idx_users_is_active;
ALTER TABLE users DROP COLUMN IF EXISTS is_active;
```

### 7.2 Auth Middleware Update

The auth middleware must check `is_active` when validating a user session. After
resolving the user from the JWT, add:

```go
if !user.IsActive {
    return echo.NewHTTPError(http.StatusForbidden, "account deactivated")
}
```

This check runs on every authenticated request, not just admin endpoints.

### 7.3 Session Invalidation on Deactivation

When a user is deactivated via `PUT /api/v1/admin/users/:id`, the handler must
delete all sessions for that user:

```sql
DELETE FROM sessions WHERE user_id = $1;
```

This ensures the deactivated user is immediately logged out of all devices.

### 7.4 User List Query Update

The `GET /api/v1/admin/users` endpoint must include `is_active` in its
response and support filtering by status:

```sql
SELECT id, username, email, role, avatar_url, is_active, created_at, updated_at
  FROM users
 WHERE ($1 = '' OR username ILIKE '%' || $1 || '%' OR email ILIKE '%' || $1 || '%')
   AND ($2 = '' OR role::TEXT = $2)
   AND ($3 IS NULL OR is_active = $3)
 ORDER BY <sort_column> <order>
 LIMIT $4 OFFSET $5;
```

---

## 8. API Endpoint Summary

Reference for all admin endpoints. Full specifications are in `docs/LLD/API.md`.

| Method | Endpoint | Role Required | Description |
|--------|----------|--------------|-------------|
| `GET` | `/api/v1/admin/users` | `root`, `moderator` | List all users (paginated, searchable, filterable) |
| `PUT` | `/api/v1/admin/users/:id` | `root` (role changes), `moderator` (deactivation only) | Update user role or active status |
| `GET` | `/api/v1/admin/stats` | `root`, `moderator` | System-wide statistics |

### Business Rules (enforced server-side)

| Rule | Enforcement |
|------|-------------|
| Cannot change own role | `PUT` handler checks `caller.id != target.id` |
| Cannot demote last root | `PUT` handler counts remaining root users before role change, returns `409` |
| Cannot deactivate yourself | `PUT` handler checks `caller.id != target.id` |
| Moderator cannot deactivate root | `PUT` handler checks `caller.role` against `target.role` |
| Moderator cannot change roles | `PUT` handler checks `caller.role === 'root'` when `role` field is present |
| Deactivation invalidates sessions | `PUT` handler deletes from `sessions` table on `is_active = false` |
