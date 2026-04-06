# Verdox -- Frontend Routing & Page Design (LLD)

> Next.js 15 App Router | TypeScript | Tailwind CSS | Design tokens from BRAND-PALETTE.md

---

## 1. Route Map

| Route | Page Component | File Path | Layout | Auth | Description |
|-------|---------------|-----------|--------|------|-------------|
| `/` | LandingPage | `src/app/page.tsx` | RootLayout | Public | Hero section, tagline, CTA buttons |
| `/login` | LoginPage | `src/app/(auth)/login/page.tsx` | AuthLayout | Public (redirect if authed) | Login form |
| `/signup` | SignUpPage | `src/app/(auth)/signup/page.tsx` | AuthLayout | Public (redirect if authed) | Registration form |
| `/forgot-password` | ForgotPasswordPage | `src/app/(auth)/forgot-password/page.tsx` | AuthLayout | Public | Email input for password reset |
| `/reset-password` | ResetPasswordPage | `src/app/(auth)/reset-password/page.tsx` | AuthLayout | Public | New password form (token from query string) |
| `/dashboard` | DashboardPage | `src/app/(dashboard)/dashboard/page.tsx` | DashboardLayout | Protected | Repository list with search and sync |
| `/repositories/[id]` | RepositoryDetailPage | `src/app/(dashboard)/repositories/[id]/page.tsx` | DashboardLayout | Protected | Repo detail with branches, suites, runs |
| `/repositories/[id]/runs/[runId]` | TestRunDetailPage | `src/app/(dashboard)/repositories/[id]/runs/[runId]/page.tsx` | DashboardLayout | Protected | Individual test run results and logs |
| `/teams` | TeamsListPage | `src/app/(dashboard)/teams/page.tsx` | DashboardLayout | Protected | Team list with create button |
| `/teams/[id]` | TeamDetailPage | `src/app/(dashboard)/teams/[id]/page.tsx` | DashboardLayout | Protected | Team repos and members management |
| `/teams/discover` | TeamDiscoveryPage | `src/app/(dashboard)/teams/discover/page.tsx` | DashboardLayout | Protected | Browse and join teams |
| `/teams/[id]/requests` | JoinRequestsPage | `src/app/(dashboard)/teams/[id]/requests/page.tsx` | DashboardLayout | Protected (team admin/maintainer) | Review join requests |
| `/admin` | AdminPage | `src/app/(dashboard)/admin/page.tsx` | DashboardLayout | Protected (root / moderator) | User management and system stats |
| `/settings` | SettingsPage | `src/app/(dashboard)/settings/page.tsx` | DashboardLayout | Protected | User profile and password forms |

---

## 2. Layout Hierarchy

```
RootLayout (src/app/layout.tsx)
|
|-- Providers: ThemeProvider, AuthProvider, Toaster
|-- Fonts: DM Serif Display, DM Sans, JetBrains Mono
|
+-- LandingPage (/)
|
+-- AuthLayout (src/app/(auth)/layout.tsx)
|   |-- Centered card on warm neutral background
|   |-- Verdox logo at top
|   |-- No sidebar, no topbar
|   |
|   +-- LoginPage (/login)
|   +-- SignUpPage (/signup)
|   +-- ForgotPasswordPage (/forgot-password)
|   +-- ResetPasswordPage (/reset-password)
|
+-- DashboardLayout (src/app/(dashboard)/layout.tsx)
    |-- Sidebar (left, 260px expanded / 64px collapsed)
    |-- TopBar (top, 56px height)
    |-- Main content area (scrollable)
    |
    +-- DashboardPage (/dashboard)
    +-- RepositoryDetailPage (/repositories/[id])
    +-- TestRunDetailPage (/repositories/[id]/runs/[runId])
    +-- TeamsListPage (/teams)
    +-- TeamDiscoveryPage (/teams/discover)
    +-- TeamDetailPage (/teams/[id])
    +-- JoinRequestsPage (/teams/[id]/requests)
    +-- AdminPage (/admin)
    +-- SettingsPage (/settings)
```

### 2.1 RootLayout

**File:** `src/app/layout.tsx`

**Responsibilities:**

- HTML shell (`<html>`, `<body>`) with `lang="en"` attribute
- Font loading via `next/font/google`: DM Serif Display (weight 400), DM Sans (weights 400, 500, 600, 700), JetBrains Mono (weights 400, 500)
- `ThemeProvider` from `next-themes` wrapping all children, configured with `attribute="data-theme"`, `defaultTheme="system"`, and `enableSystem={true}`
- `AuthProvider` wrapping all children -- manages access token state, refresh logic, and user context
- `Toaster` from Sonner positioned at `bottom-right` for toast notifications
- Global CSS import (`src/styles/globals.css`) containing Tailwind directives and CSS custom properties from BRAND-PALETTE.md

```tsx
// src/app/layout.tsx (structural outline)
import { DM_Serif_Display, DM_Sans, JetBrains_Mono } from 'next/font/google';
import { ThemeProvider } from 'next-themes';
import { Toaster } from 'sonner';

import { AuthProvider } from '@/lib/auth';
import '@/styles/globals.css';

const dmSerifDisplay = DM_Serif_Display({ weight: '400', subsets: ['latin'], variable: '--font-display' });
const dmSans = DM_Sans({ weight: ['400', '500', '600', '700'], subsets: ['latin'], variable: '--font-body' });
const jetbrainsMono = JetBrains_Mono({ weight: ['400', '500'], subsets: ['latin'], variable: '--font-mono' });

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`${dmSerifDisplay.variable} ${dmSans.variable} ${jetbrainsMono.variable}`}>
      <body className="font-body bg-bg-primary text-text-primary antialiased">
        <ThemeProvider attribute="data-theme" defaultTheme="system" enableSystem>
          <AuthProvider>
            {children}
            <Toaster position="bottom-right" richColors />
          </AuthProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
```

### 2.2 AuthLayout

**File:** `src/app/(auth)/layout.tsx`

**Responsibilities:**

- Full-viewport container with `bg-bg-primary` background
- Vertically and horizontally centered content (`flex items-center justify-center min-h-screen`)
- Verdox logo centered above the form card
- Form card: `bg-bg-secondary`, `border border-border`, `rounded-xl`, `shadow-card`, `max-w-[420px] w-full`, `p-8`
- No sidebar, no topbar -- clean, distraction-free layout

```tsx
// src/app/(auth)/layout.tsx (structural outline)
import Image from 'next/image';

export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-bg-primary px-4">
      <div className="w-full max-w-[420px]">
        <div className="mb-8 flex justify-center">
          <Image src="/logo.svg" alt="Verdox" width={140} height={40} priority />
        </div>
        <div className="rounded-xl border border-border bg-bg-secondary p-8 shadow-card">
          {children}
        </div>
      </div>
    </div>
  );
}
```

### 2.3 DashboardLayout

**File:** `src/app/(dashboard)/layout.tsx`

**Responsibilities:**

- Two-region layout: fixed Sidebar (left) + main area (right)
- Main area subdivided: fixed TopBar (top) + scrollable content (below)
- Sidebar consumes the full viewport height; content area fills the remaining width
- Sidebar collapses from 260px to 64px on toggle (persisted to localStorage)
- On mobile (below `md` breakpoint), sidebar is hidden behind a hamburger menu and overlays as a drawer

```tsx
// src/app/(dashboard)/layout.tsx (structural outline)
import { Sidebar } from '@/components/layout/sidebar';
import { TopBar } from '@/components/layout/topbar';

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex h-screen overflow-hidden">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
        <TopBar />
        <main className="flex-1 overflow-y-auto bg-bg-primary p-6">
          {children}
        </main>
      </div>
    </div>
  );
}
```

---

## 3. Per-Page Specification

---

### 3.1 Landing Page (`/`)

**Component:** `LandingPage`
**File:** `src/app/page.tsx`
**Rendering:** Server component (static, no API calls)

**API Calls:** None

**Key UI Components:**

| Component | Description |
|-----------|-------------|
| `Logo` | Verdox logo rendered with `next/image` |
| `HeroSection` | DM Serif Display heading ("Test Your Services at One Place"), DM Sans subtitle paragraph, warm neutral background |
| `CTAButtons` | Two buttons side by side: "Login" (secondary variant, links to `/login`), "Sign Up" (primary variant, links to `/signup`) |
| `FeatureCards` | Three cards in a responsive grid (`grid-cols-1 md:grid-cols-3`), each with a placeholder illustration, DM Sans heading, and description text |
| `Footer` | Minimal footer with copyright text |

**Loading State:** Not applicable (static content, no data fetching)

**Empty State:** Not applicable

**Error State:** Not applicable (no data fetching)

**Layout Notes:**

- Full-width page, not wrapped in DashboardLayout
- Hero section spans full viewport height (`min-h-screen`) with centered content
- Features section below the fold with `py-16` vertical spacing
- Accent color (`#1C6D74`) used for the primary CTA button and feature card icons

---

### 3.2 Login Page (`/login`)

**Component:** `LoginPage`
**File:** `src/app/(auth)/login/page.tsx`
**Rendering:** Client component (`'use client'`) -- form interaction and API mutation

**API Calls:**

| Action | Method | Endpoint | Trigger |
|--------|--------|----------|---------|
| Authenticate | POST | `/api/v1/auth/login` | Form submission |

**Key UI Components:**

| Component | Props / Notes |
|-----------|---------------|
| `LoginForm` | Built with React Hook Form + Zod schema validation. Fields: `login` (username or email), `password`. Submit button with loading spinner during API call. |
| `Input` | Two instances: login field (text type, placeholder "Username or email"), password field (password type with show/hide toggle) |
| `Button` | Primary variant, full width, `type="submit"`, shows loading state |
| `Link` | "Don't have an account? Sign up" linking to `/signup` |
| `Link` | "Forgot password?" linking to `/forgot-password` |

**Validation (Zod schema):**

```typescript
const loginSchema = z.object({
  login: z.string().min(1, 'Username or email is required'),
  password: z.string().min(1, 'Password is required'),
});
```

**On Success:**

1. Store access token in AuthContext (memory) and refresh token via httpOnly cookie (set by server)
2. Store user object in AuthContext
3. Redirect to `/dashboard` (or to the return URL from query param `?returnTo=`)

**Loading State:** Submit button shows spinner and "Signing in..." text, inputs disabled

**Empty State:** Not applicable

**Error State:**

- Validation errors displayed inline below each field (red text, `text-danger`)
- API errors (401, 429) displayed as a toast notification (Sonner) or inline error banner above the form

---

### 3.3 Sign Up Page (`/signup`)

**Component:** `SignUpPage`
**File:** `src/app/(auth)/signup/page.tsx`
**Rendering:** Client component (`'use client'`)

**API Calls:**

| Action | Method | Endpoint | Trigger |
|--------|--------|----------|---------|
| Register | POST | `/api/v1/auth/signup` | Form submission |

**Key UI Components:**

| Component | Props / Notes |
|-----------|---------------|
| `SignupForm` | React Hook Form + Zod. Fields: `username`, `email`, `password`. |
| `Input` | Three instances: username (text), email (email type), password (password type with show/hide toggle) |
| `Button` | Primary variant, full width, `type="submit"` |
| `Link` | "Already have an account? Log in" linking to `/login` |

**Validation (Zod schema):**

```typescript
const signupSchema = z.object({
  username: z.string()
    .min(3, 'Username must be at least 3 characters')
    .max(30, 'Username must be at most 30 characters')
    .regex(/^[a-zA-Z0-9_]+$/, 'Only letters, numbers, and underscores allowed'),
  email: z.string().email('Enter a valid email address'),
  password: z.string()
    .min(8, 'Password must be at least 8 characters')
    .regex(/[A-Z]/, 'Must contain at least one uppercase letter')
    .regex(/[a-z]/, 'Must contain at least one lowercase letter')
    .regex(/[0-9]/, 'Must contain at least one digit'),
});
```

**On Success:**

1. Store access token and user in AuthContext
2. Redirect to `/dashboard`

**Loading State:** Submit button shows spinner and "Creating account..." text

**Empty State:** Not applicable

**Error State:**

- Inline validation errors per field
- 409 CONFLICT: "Username or email already exists" banner above the form

---

### 3.4 Forgot Password Page (`/forgot-password`)

**Component:** `ForgotPasswordPage`
**File:** `src/app/(auth)/forgot-password/page.tsx`
**Rendering:** Client component (`'use client'`)

**API Calls:**

| Action | Method | Endpoint | Trigger |
|--------|--------|----------|---------|
| Request reset | POST | `/api/v1/auth/forgot-password` | Form submission |

**Key UI Components:**

| Component | Props / Notes |
|-----------|---------------|
| `Input` | Email field |
| `Button` | Primary variant, "Send Reset Link" |
| `Link` | "Back to login" linking to `/login` |

**Validation:** `email: z.string().email()`

**On Success:** Show a confirmation message ("If an account with that email exists, a password reset link has been sent.") and disable the form. Do not reveal whether the email exists.

**Loading State:** Button shows spinner

**Empty State:** Not applicable

**Error State:** Inline validation error for invalid email format

---

### 3.5 Reset Password Page (`/reset-password`)

**Component:** `ResetPasswordPage`
**File:** `src/app/(auth)/reset-password/page.tsx`
**Rendering:** Client component (`'use client'`)

**URL:** `/reset-password?token=<reset_token>`

**API Calls:**

| Action | Method | Endpoint | Trigger |
|--------|--------|----------|---------|
| Reset password | POST | `/api/v1/auth/reset-password` | Form submission |

**Key UI Components:**

| Component | Props / Notes |
|-----------|---------------|
| `Input` | New password field, confirm password field |
| `Button` | Primary variant, "Reset Password" |

**Validation:**

```typescript
const resetSchema = z.object({
  new_password: z.string()
    .min(8, 'Password must be at least 8 characters')
    .regex(/[A-Z]/, 'Must contain at least one uppercase letter')
    .regex(/[a-z]/, 'Must contain at least one lowercase letter')
    .regex(/[0-9]/, 'Must contain at least one digit'),
  confirm_password: z.string(),
}).refine((data) => data.new_password === data.confirm_password, {
  message: 'Passwords do not match',
  path: ['confirm_password'],
});
```

**On Success:** Show success toast, redirect to `/login` after 2 seconds

**Loading State:** Button shows spinner

**Empty State:** If no `token` query param, display "Invalid or missing reset token" with a link to `/forgot-password`

**Error State:** 401 UNAUTHORIZED: "Reset token is invalid or expired" banner

---

### 3.6 Dashboard (`/dashboard`)

**Component:** `DashboardPage`
**File:** `src/app/(dashboard)/dashboard/page.tsx`
**Rendering:** Server component for initial data fetch, client components for search and interactions

**Team Membership Check:** On page load, check if the user has any team memberships. If the user has no team memberships, redirect to `/teams/discover` instead of showing an empty repository list. Repos shown are scoped to user's teams -- user must belong to at least one team to see repositories.

**API Calls:**

| Action | Method | Endpoint | Trigger |
|--------|--------|----------|---------|
| List repos | GET | `/api/v1/repositories?page=N&per_page=20&search=Q` | Page load, pagination, search |
| Add repo | POST | `/api/v1/repositories` | "Add Repository" button → enter GitHub URL |

**Key UI Components:**

| Component | Props / Notes |
|-----------|---------------|
| `SearchBar` | Text input with search icon, debounced (300ms) filtering by repo name |
| `SyncButton` | Secondary variant button, triggers GitHub sync, shows spinner during sync |
| `RepoCard` | Card per repository. Shows: repo name (DM Sans, `type-h3`), latest run status badge (pass/fail/pending), "Run a Test" button (primary, small), "Dash ->" link (ghost button linking to `/repositories/:id`). Card uses `bg-bg-secondary`, `rounded-lg`, `border-border`, `shadow-card`, hover `shadow-md` |
| `Pagination` | Page controls at bottom (Previous / Next buttons, page indicator) |

**Card Grid:** `grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6`

**Loading State:** Grid of 6 `Skeleton` cards matching `RepoCard` dimensions (height ~160px, rounded-lg)

**Empty State:**

- No repos at all: Illustration + "No repositories yet. Sync your GitHub repos to get started." + "Sync Repos" primary button
- No search results: "No repositories match your search." + "Clear search" link

**Error State:** Error boundary wrapping the page. On API failure, shows "Failed to load repositories" message with a "Retry" button.

---

### 3.7 Repository Detail (`/repositories/[id]`)

**Component:** `RepositoryDetailPage`
**File:** `src/app/(dashboard)/repositories/[id]/page.tsx`
**Rendering:** Server component for initial fetch + client components for branch selector, run triggers, and polling

**API Calls:**

| Action | Method | Endpoint | Trigger |
|--------|--------|----------|---------|
| Get repo | GET | `/api/v1/repositories/:id` | Page load |
| List branches | GET | `/api/v1/repositories/:id/branches` | Page load |
| List commits | GET | `/api/v1/repositories/:id/commits?branch=X` | Branch selection |
| List suites | GET | `/api/v1/repositories/:id/suites` | Page load |
| Get suite runs | GET | `/api/v1/suites/:suiteId/runs?per_page=5` | Page load (per suite) |
| Run single suite | POST | `/api/v1/suites/:suiteId/runs` | "Run" button on suite card |
| Run all suites | POST | `/api/v1/repositories/:id/suites/run-all` | "Run All" button |
| Discover tests | POST | `/api/v1/repositories/:id/discover` | "Scan for Tests" button click |
| Get discovery results | GET | `/api/v1/repositories/:id/discovery` | Page load (if previous discovery exists) |

**Key UI Components:**

| Component | Props / Notes |
|-----------|---------------|
| `Breadcrumb` | "Repositories" (link to `/dashboard`) > repo name (current page) |
| `BranchSelector` | Dropdown menu listing branches. Shows current branch with a green dot indicator for the default branch. On change, refreshes commit display and suite run data. |
| `CommitDisplay` | Shows the latest commit hash (truncated to 7 chars, monospace `font-mono text-code`), commit message, and author |
| `SuiteCard` | One card per test suite. Contains: suite name, type badge ("Unit" or "Integration" using `Badge` with neutral variant), `ProgressBar` showing pass/total ratio (green fill for pass, red for fail), pass/total count text (e.g., "42/50"), "Run" button (primary, small), "Detail ->" link (ghost, links to the latest run detail page) |
| `RunAllButton` | Primary button at the top of the suites section, triggers all suites to run |
| `ScanForTestsButton` | Secondary button visible to team admin/maintainer only, and only if `VERDOX_OPENAI_API_KEY` is configured on the server. Triggers `POST /api/v1/repositories/:id/discover`. Shows discovered test suites with "Create Suite" action buttons. Run buttons visible only to team admin/maintainer; viewer sees read-only results. |
| `ProgressBar` | Determinate bar with `bg-success` fill for passed portion and `bg-danger` fill for failed portion, `rounded-full`, height 8px |

**Loading State:**

- Page header: Skeleton for repo name (h1 width) + skeleton for branch selector
- Suite section: 2-3 skeleton cards matching SuiteCard dimensions

**Empty State:**

- No suites: "No test suites configured. Create your first suite to start testing." + "Create Suite" button
- Suite has no runs: "No runs yet" text in the suite card with the "Run" button available

**Error State:** Error boundary. 404: "Repository not found" with "Back to Dashboard" link. Other errors: generic retry message.

---

### 3.8 Test Run Detail (`/repositories/[id]/runs/[runId]`)

**Component:** `TestRunDetailPage`
**File:** `src/app/(dashboard)/repositories/[id]/runs/[runId]/page.tsx`
**Rendering:** Server component for initial data + client components for log viewer and live polling

**API Calls:**

| Action | Method | Endpoint | Trigger |
|--------|--------|----------|---------|
| Get run | GET | `/api/v1/runs/:runId` | Page load |
| Get logs | GET | `/api/v1/runs/:runId/logs` | "Run Logs" button click |
| Cancel run | POST | `/api/v1/runs/:runId/cancel` | "Cancel" button (when running) |
| Poll status | GET | `/api/v1/runs/:runId` | Every 3 seconds while `status === 'running'` |

**Polling Behavior:**

- When run status is `running`, poll `GET /api/v1/runs/:runId` every 3 seconds using `setInterval`
- Stop polling when status changes to `passed`, `failed`, `cancelled`, or `error`
- Use `useEffect` cleanup to clear the interval on unmount
- Update results in place as they arrive

**Key UI Components:**

| Component | Props / Notes |
|-----------|---------------|
| `Breadcrumb` | "Repositories" > repo name (link) > "Run #N" (current) |
| `RunHeader` | Displays: suite name (`type-h2`), branch name badge, commit hash (monospace), run number ("Run #N"), run status badge, total duration, timestamp |
| `RunLogsButton` | Secondary button, opens `LogViewer` modal/drawer |
| `CancelButton` | Danger variant, visible only when status is `running` |
| `ResultRow` | One row per individual test case. Shows: play icon (left), test name, status badge (Pass/Fail/Skip/Error), duration (e.g., "1.2s"), expandable section for per-test log output |
| `LogViewer` | Full-width panel or modal with ANSI-aware terminal output. Uses `font-mono`, `bg-bg-tertiary`, monospace text, auto-scrolls to bottom. |

**Status Badge Colors:**

| Status | Background | Text Color | Hex |
|--------|-----------|------------|-----|
| Pass | `#E6F4EC` | `--success` | `#2D8A4E` |
| Fail | `#FDEAEA` | `--danger` | `#C93B3B` |
| Skip | `#FEF3D9` | `--warning` | `#D4910A` |
| Error | `#FDEAEA` | `--danger` | `#C93B3B` |
| Running | `--accent-subtle` | `--accent` | `#1C6D74` |

**Loading State:**

- RunHeader: Skeleton blocks for suite name, badges, and metadata
- ResultRows: 5 skeleton rows with shimmer animation

**Empty State:** Run has no results yet (still running): "Waiting for results..." with a spinner

**Error State:** 404: "Test run not found" with link back to repository. Other: retry message.

---

### 3.9 Teams List (`/teams`)

**Component:** `TeamsListPage`
**File:** `src/app/(dashboard)/teams/page.tsx`
**Rendering:** Server component

**API Calls:**

| Action | Method | Endpoint | Trigger |
|--------|--------|----------|---------|
| List teams | GET | `/api/v1/teams` | Page load |
| Create team | POST | `/api/v1/teams` | Modal form submission |

**Key UI Components:**

| Component | Props / Notes |
|-----------|---------------|
| Page Header | "Teams" heading (`type-h1`, DM Serif Display) + "Create New" button (primary) |
| `TeamCard` | Card per team. Shows: team name (`type-h3`), member count, enter arrow icon (links to `/teams/:id`). Card styling matches `RepoCard` tokens. |
| `CreateTeamModal` | Modal overlay with form: team name input + description textarea + "Create" button |

**Card Grid:** `grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6`

**Loading State:** Grid of 4 skeleton cards

**Empty State:** "No teams yet. Create your first team." + "Create New" primary button centered

**Error State:** Error boundary with retry

---

### 3.10 Team Detail (`/teams/[id]`)

**Component:** `TeamDetailPage`
**File:** `src/app/(dashboard)/teams/[id]/page.tsx`
**Rendering:** Server component for initial fetch + client components for add/remove interactions

**API Calls:**

| Action | Method | Endpoint | Trigger |
|--------|--------|----------|---------|
| Get team | GET | `/api/v1/teams/:id` | Page load |
| Add repo | POST | `/api/v1/teams/:id/repositories` | "+" button on repo panel |
| Remove repo | DELETE | `/api/v1/teams/:id/repositories/:repoId` | "-" button on repo panel |
| Add member | POST | `/api/v1/teams/:id/members` | Invite form submission |
| Update member role | PUT | `/api/v1/teams/:id/members/:userId` | Role dropdown change |
| Remove member | DELETE | `/api/v1/teams/:id/members/:userId` | Remove button click |
| Save GitHub PAT | PUT | `/api/v1/teams/:id/pat` | PAT form submission (admin only) |
| Validate GitHub PAT | GET | `/api/v1/teams/:id/pat/validate` | "Validate" button click (admin only) |
| Delete GitHub PAT | DELETE | `/api/v1/teams/:id/pat` | "Remove" button click (admin only) |

**Key UI Components:**

| Component | Props / Notes |
|-----------|---------------|
| `Breadcrumb` | "Teams" (link to `/teams`) > team name (current) |
| `RepoPanel` | Left panel (50% width on desktop, full width on mobile). Lists assigned repos with "-" remove buttons. "+" button opens a dropdown to assign unassigned repos. |
| `MemberPanel` | Right panel (50% width on desktop, full width on mobile). Lists members with role badges (`admin`, `maintainer`, `viewer`), approve/reject buttons for pending invitations, and a remove button per member. |
| `TeamPATForm` | GitHub PAT management section, visible only to team admins. Displays PAT status indicator (configured/not configured), expiry warning badge if `pat_expires_at` is within 30 days, token input (password type), "Validate" button (secondary, calls `GET /api/v1/teams/:id/pat/validate`), "Save" button (primary, calls `PUT /api/v1/teams/:id/pat`), "Remove" button (danger). See [GITHUB-PAT-GUIDE.md](../GITHUB-PAT-GUIDE.md) for instructions shown to team admins. |
| `PATStatusBadge` | Inline badge showing PAT status: "Configured" (green), "Not Configured" (red), "Expiring Soon" (amber). Visible to all team members as read-only. |
| `Badge` | Used for role display: admin (accent-subtle bg, accent text), maintainer (warning tint), viewer (neutral) |
| `JoinRequestsLink` | Tab or link labeled "Join Requests" visible to team admin and maintainer. Links to `/teams/[id]/requests`. |
| `ManageRequestsButton` | Secondary button labeled "Manage Requests" linking to `/teams/[id]/requests`, visible to admin/maintainer |

**Two-Panel Layout:** `grid grid-cols-1 lg:grid-cols-2 gap-6`

**Loading State:** Two skeleton panels side by side

**Empty State:**

- Repo panel empty: "No repositories assigned. Use the + button to assign one."
- Member panel empty: "No members yet. Invite team members to collaborate."

**Error State:** 404: "Team not found" with link to `/teams`. 403: "You don't have access to this team."

---

### 3.11 Admin Panel (`/admin`)

**Component:** `AdminPage`
**File:** `src/app/(dashboard)/admin/page.tsx`
**Rendering:** Server component for stats + client component for user table interactions

**Access Control:** Only users with role `root` or `moderator` can access. Middleware redirects others to `/dashboard`. The page itself also checks the user role from AuthContext and renders a 403 message as a fallback.

**API Calls:**

| Action | Method | Endpoint | Trigger |
|--------|--------|----------|---------|
| List users | GET | `/api/v1/admin/users?page=N&per_page=20` | Page load, pagination |
| Update user role | PUT | `/api/v1/admin/users/:id` | Role dropdown change |
| Get stats | GET | `/api/v1/admin/stats` | Page load |

**Key UI Components:**

| Component | Props / Notes |
|-----------|---------------|
| `StatsCards` | Row of stat cards at the top: total users, total repos, total test runs, active teams. Each card: `bg-bg-secondary`, `rounded-lg`, number in `type-h2` (DM Serif Display), label in `type-body-sm` |
| `UserTable` | Data table with columns: username, email, role (dropdown: `user` / `moderator` / `root`), active status (toggle switch), joined date. Sortable by username and joined date. Paginated. "Promote to Moderator" action available to `root` users only. |
| `Dropdown` | Role selector per user row. On change, calls PUT endpoint |
| `Pagination` | Table pagination controls |

**Loading State:**

- Stats: 4 skeleton cards in a row
- Table: Skeleton rows (8 rows) with shimmer

**Empty State:** "No users found." (unlikely in production)

**Error State:** 403: "You do not have permission to access the admin panel." with a "Go to Dashboard" link. Only `root` and `moderator` roles can access. Other: error boundary with retry.

---

### 3.12 Settings Page (`/settings`)

**Component:** `SettingsPage`
**File:** `src/app/(dashboard)/settings/page.tsx`
**Rendering:** Client component (`'use client'`) -- two forms with independent submissions

**API Calls:**

| Action | Method | Endpoint | Trigger |
|--------|--------|----------|---------|
| Get profile | GET | `/api/v1/me` | Page load |
| Update profile | PUT | `/api/v1/me` | Profile form submission |
| Change password | PUT | `/api/v1/me/password` | Password form submission |

**Key UI Components:**

| Component | Props / Notes |
|-----------|---------------|
| `ProfileForm` | React Hook Form. Fields: username (text), email (email), avatar URL (text, optional). "Save Changes" button (primary). |
| `PasswordForm` | React Hook Form. Fields: current password, new password, confirm new password. "Update Password" button (primary). |
| `Divider` | Horizontal rule (`border-border`) separating the form sections |

**Page Layout:** Single column, max-width 640px centered. Profile form on top, password form below, each separated by a divider with `py-8` spacing.

> **Note:** GitHub PAT management has moved to the team settings page.
> Team admins configure the PAT at `/teams/[id]` (see Section 3.10).
> See [GITHUB-PAT-GUIDE.md](../GITHUB-PAT-GUIDE.md) for detailed instructions.

**Validation:**

- Profile: username 3-30 chars, valid email
- Password: current password required, new password min 8 chars with strength rules, confirm must match

**Loading State:** Skeleton placeholders for input fields while `GET /api/v1/me` loads

**Empty State:** Not applicable

**Error State:** Inline validation errors per field. API errors shown as toast notifications.

---

### 3.13 Team Discovery (`/teams/discover`)

**Component:** `TeamDiscoveryPage`
**File:** `src/app/(dashboard)/teams/discover/page.tsx`
**Rendering:** Client component (`'use client'`) -- interactive search and request actions

**Behavior:** Shown after signup when user has no team memberships (redirected from `/dashboard`). Also accessible from sidebar.

**API Calls:**

| Action | Method | Endpoint | Trigger |
|--------|--------|----------|---------|
| List discoverable teams | GET | `/api/v1/teams/discover` | Page load (paginated) |
| Request to join | POST | `/api/v1/teams/:id/join-requests` | "Request to Join" button click |

**Key UI Components:**

| Component | Props / Notes |
|-----------|---------------|
| `SearchBar` | Text input with search icon, filters teams by name |
| `TeamDiscoveryCard` | Card per team. Shows: team name, member count, repo count, "Request to Join" button (primary). After request is sent, button changes to "Pending" (disabled). Hidden if user is already a member. |
| `JoinRequestModal` | Modal overlay triggered on "Request to Join" click. Contains: optional message textarea, "Send Request" button (primary). Submits `POST /api/v1/teams/:id/join-requests` with optional message body. |
| `Pagination` | Page controls at bottom (Previous / Next buttons, page indicator) |

**Loading State:** Grid of 6 skeleton cards matching `TeamDiscoveryCard` dimensions

**Empty State:** "No teams available yet. Contact your administrator."

**Error State:** Error boundary with retry

---

### 3.14 Join Requests (`/teams/[id]/requests`)

**Component:** `JoinRequestsPage`
**File:** `src/app/(dashboard)/teams/[id]/requests/page.tsx`
**Rendering:** Server component for initial fetch + client components for approve/reject interactions

**Access Control:** Visible only to team admin and maintainer. Other team roles are redirected to `/teams/[id]`.

**API Calls:**

| Action | Method | Endpoint | Trigger |
|--------|--------|----------|---------|
| List join requests | GET | `/api/v1/teams/:id/join-requests` | Page load |
| Approve/reject request | PATCH | `/api/v1/teams/:id/join-requests/:rid` | Approve or Reject button click |

**Key UI Components:**

| Component | Props / Notes |
|-----------|---------------|
| `Breadcrumb` | "Teams" (link to `/teams`) > team name (link to `/teams/[id]`) > "Join Requests" (current) |
| `JoinRequestRow` | One row per pending request. Shows: avatar, username, email, optional message, role selector dropdown (viewer/maintainer/admin -- only shown on approval), "Approve" button (primary), "Reject" button (danger). On approve: sends `PATCH /api/v1/teams/:id/join-requests/:rid` with `{ status: "approved", role: "viewer" }` (or selected role). On reject: sends `PATCH` with `{ status: "rejected" }`. |
| `RoleSelector` | Dropdown with options: `viewer`, `maintainer`, `admin`. Defaults to `viewer`. Shown inline in `JoinRequestRow` when approving. |

**Loading State:** Skeleton rows (5 rows) with shimmer animation

**Empty State:** "No pending join requests."

**Error State:** 403: "You do not have permission to manage join requests." 404: "Team not found" with link to `/teams`.

---

## 4. Shared Components

All shared components live under `src/components/` organized by domain.

### 4.1 Layout Components (`src/components/layout/`)

#### Sidebar (`sidebar.tsx`)

| Prop | Type | Description |
|------|------|-------------|
| -- | -- | No external props. Reads current path from `usePathname()` to determine active item. |

**Internal state:**

- `collapsed: boolean` -- toggled by a collapse button, persisted to `localStorage`

**Navigation items:**

| Icon | Label | Href | Visible To |
|------|-------|------|------------|
| `LayoutDashboard` | Dashboard | `/dashboard` | All |
| `GitFork` | Repositories | `/dashboard` | All (same as dashboard) |
| `Users` | Teams | `/teams` | All |
| `Shield` | Admin | `/admin` | `moderator`, `root` only |

**Styling:** Follows Sidebar tokens from BRAND-PALETTE.md. Active item: `bg-accent-subtle`, `text-accent`. Inactive: `text-text-secondary`. Hover: `bg-bg-tertiary`. Collapse transition: `duration-slow`.

#### TopBar (`topbar.tsx`)

| Prop | Type | Description |
|------|------|-------------|
| -- | -- | No external props. Reads user from `useAuth()` hook. |

**Contains:**

- Verdox logo (links to `/dashboard`)
- Breadcrumb trail (populated by page-level metadata)
- ThemeToggle button (sun/moon icon)
- Notification bell icon (placeholder for future notifications)
- User avatar dropdown (opens UserMenu)

**UserMenu items:**

| Label | Action |
|-------|--------|
| Settings | Navigate to `/settings` |
| Admin Panel | Navigate to `/admin` (visible only to moderator/root) |
| Sign Out | Call `POST /api/v1/auth/logout`, clear AuthContext, redirect to `/login` |

#### ThemeToggle (`theme-toggle.tsx`)

| Prop | Type | Description |
|------|------|-------------|
| -- | -- | No external props. Uses `useTheme()` from `next-themes`. |

**Behavior:** Sun icon in dark mode, moon icon in light mode. Toggles `data-theme` attribute on `<html>`. Transition: `duration-fast`.

### 4.2 UI Primitives (`src/components/ui/`)

#### Button (`button.tsx`)

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `variant` | `'primary' \| 'secondary' \| 'ghost' \| 'danger'` | `'primary'` | Visual style per BRAND-PALETTE.md button tokens |
| `size` | `'sm' \| 'md' \| 'lg'` | `'md'` | `sm`: 28px height, `md`: 36px height, `lg`: 44px height |
| `loading` | `boolean` | `false` | Shows spinner icon, disables button |
| `disabled` | `boolean` | `false` | Applies disabled styles |
| `asChild` | `boolean` | `false` | Renders as Slot for wrapping links |

Extends `React.ButtonHTMLAttributes<HTMLButtonElement>`.

#### Input (`input.tsx`)

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `label` | `string` | -- | Label text above the input |
| `error` | `string` | -- | Error message displayed below in `text-danger` |
| `icon` | `React.ReactNode` | -- | Leading icon inside the input |

Extends `React.InputHTMLAttributes<HTMLInputElement>`.

#### Card (`card.tsx`)

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `children` | `React.ReactNode` | -- | Card content |
| `className` | `string` | -- | Additional classes |
| `hoverable` | `boolean` | `false` | Adds hover shadow transition |

Subcomponents: `Card.Header`, `Card.Body`, `Card.Footer`.

#### Badge (`badge.tsx`)

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `variant` | `'success' \| 'danger' \| 'warning' \| 'neutral' \| 'accent'` | `'neutral'` | Color scheme per BRAND-PALETTE.md badge tokens |
| `children` | `React.ReactNode` | -- | Badge text |

#### Modal (`modal.tsx`)

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `open` | `boolean` | -- | Controls visibility |
| `onClose` | `() => void` | -- | Called on overlay click or Escape key |
| `title` | `string` | -- | Modal header text |
| `children` | `React.ReactNode` | -- | Modal body content |

Features: Focus trap, Escape to close, overlay click to close, enter/exit transitions per BRAND-PALETTE.md modal tokens.

#### Table (`table.tsx`)

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `columns` | `Column[]` | -- | Column definitions: `{ key, header, sortable?, render? }` |
| `data` | `T[]` | -- | Row data array |
| `onSort` | `(key: string, order: 'asc' \| 'desc') => void` | -- | Sort handler |
| `loading` | `boolean` | `false` | Shows skeleton rows |

#### ProgressBar (`progress.tsx`)

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `value` | `number` | -- | Current value (0-100) |
| `max` | `number` | `100` | Maximum value |
| `variant` | `'success' \| 'danger' \| 'accent'` | `'accent'` | Fill color |
| `showLabel` | `boolean` | `false` | Shows percentage text |

#### Skeleton (`skeleton.tsx`)

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `className` | `string` | -- | Controls width, height, and border radius |

Renders a `div` with `bg-bg-tertiary animate-pulse rounded-md`.

#### Toast

Provided by Sonner. Invoked via `toast.success()`, `toast.error()`, `toast.info()` functions. Configured globally in RootLayout.

#### Dropdown (`dropdown.tsx`)

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `trigger` | `React.ReactNode` | -- | Element that opens the dropdown |
| `items` | `DropdownItem[]` | -- | `{ label, onClick?, href?, icon?, disabled? }` |
| `align` | `'left' \| 'right'` | `'left'` | Alignment relative to trigger |

### 4.3 Domain Components

#### Repository Domain (`src/components/repository/`)

- **`repo-card.tsx`** -- Used in Dashboard. Props: `repository: Repository`. Renders name, status badge, "Run a Test" button, "Dash ->" link.
- **`branch-selector.tsx`** -- Used in Repository Detail. Props: `branches: Branch[]`, `selected: string`, `onSelect: (branch: string) => void`. Dropdown with search filtering.
- **`commit-list.tsx`** -- Used in Repository Detail. Props: `commits: Commit[]`. Scrollable list showing hash, message, author, and date.

#### Test Domain (`src/components/test/`)

- **`suite-card.tsx`** -- Used in Repository Detail. Props: `suite: TestSuite`, `latestRun?: TestRun`, `onRun: () => void`. Card with progress bar, count, run button, detail link.
- **`run-list.tsx`** -- Used in Repository Detail. Props: `runs: TestRun[]`. List of recent runs with status badges and timestamps.
- **`run-detail.tsx`** -- Used in Test Run Detail. Props: `run: TestRun`. Full run metadata display.
- **`result-row.tsx`** -- Used in Test Run Detail. Props: `result: TestResult`. Expandable row with test name, status icon, duration, and log output.
- **`log-viewer.tsx`** -- Used in Test Run Detail. Props: `logs: string`, `loading: boolean`. ANSI-aware terminal display with auto-scroll.

#### Team Domain (`src/components/team/`)

- **`team-card.tsx`** -- Used in Teams List. Props: `team: Team`. Card with name, member count, link arrow.
- **`member-list.tsx`** -- Used in Team Detail. Props: `members: TeamMember[]`, `onUpdateRole`, `onRemove`, `onApprove`, `onReject`. Table with role badges and action buttons.
- **`repo-assign.tsx`** -- Used in Team Detail. Props: `assignedRepos: Repository[]`, `availableRepos: Repository[]`, `onAssign`, `onUnassign`. List with +/- buttons.
- **`team-discovery-card.tsx`** -- Used in Team Discovery. Props: `name: string`, `slug: string`, `memberCount: number`, `repoCount: number`, `userStatus: 'none' | 'pending' | 'member'`, `onRequestJoin: () => void`. Card with team info and "Request to Join" button.
- **`join-request-row.tsx`** -- Used in Join Requests. Props: `request: JoinRequest`, `onApprove: (role: TeamRole) => void`, `onReject: () => void`. Row with avatar, username, email, message, role selector, approve/reject buttons.

#### Settings Domain (`src/components/settings/`)

- (Empty -- GitHub PAT form has moved to `src/components/team/team-pat-form.tsx`.)

#### Team PAT Component (`src/components/team/team-pat-form.tsx`)

- **`team-pat-form.tsx`** -- Used in Team Detail (admin only). Props: `teamId: string`, `hasPat: boolean`, `patExpiresAt: string | null`, `onSave: (token: string) => void`, `onValidate: () => void`, `onDelete: () => void`. GitHub PAT input with validate button, expiry warning badge, and status indicator.

---

## 5. Auth Protection

### 5.1 Middleware (`src/middleware.ts`)

Next.js middleware runs on the Edge Runtime before every request. It handles route protection and redirects.

```typescript
// src/middleware.ts (implementation outline)
import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

const publicRoutes = ['/', '/login', '/signup', '/forgot-password', '/reset-password'];
const authRoutes = ['/login', '/signup', '/forgot-password', '/reset-password'];
const adminRoutes = ['/admin'];

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const accessToken = request.cookies.get('access_token')?.value;

  // Public routes: no protection needed
  if (publicRoutes.includes(pathname)) {
    // If authenticated user visits auth routes, redirect to dashboard
    if (authRoutes.includes(pathname) && accessToken) {
      return NextResponse.redirect(new URL('/dashboard', request.url));
    }
    return NextResponse.next();
  }

  // Protected routes: require authentication
  if (!accessToken) {
    const loginUrl = new URL('/login', request.url);
    loginUrl.searchParams.set('returnTo', pathname);
    return NextResponse.redirect(loginUrl);
  }

  // Admin routes: require root or moderator role
  // Note: Role is checked from a lightweight cookie or JWT claims.
  // Full role verification happens server-side on API calls.
  if (adminRoutes.some((route) => pathname.startsWith(route))) {
    const userRole = parseRoleFromToken(accessToken);
    if (userRole !== 'root' && userRole !== 'moderator') {
      return NextResponse.redirect(new URL('/dashboard', request.url));
    }
  }

  // Team membership check: users with no team memberships are redirected to /teams/discover
  // (except when already on /teams/discover or /settings)
  if (pathname === '/dashboard' || pathname.startsWith('/repositories')) {
    const hasTeams = parseTeamMembershipFromToken(accessToken);
    if (!hasTeams) {
      return NextResponse.redirect(new URL('/teams/discover', request.url));
    }
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    // Match all routes except static files, _next internals, and API routes
    '/((?!_next/static|_next/image|favicon.ico|logo.svg|api).*)',
  ],
};
```

### 5.2 Client-Side Auth Guard

The `AuthProvider` (`src/lib/auth.tsx`) provides:

| Export | Type | Description |
|--------|------|-------------|
| `AuthProvider` | Component | Context provider wrapping the app. Stores user and access token in state. |
| `useAuth()` | Hook | Returns `{ user, accessToken, login, logout, isLoading }` |

**Token Refresh Logic:**

- On mount, call `POST /api/v1/auth/refresh` to get a fresh access token (refresh token sent via httpOnly cookie automatically)
- Set a timer to refresh the access token when it has less than 2 minutes remaining (based on JWT `exp` claim)
- If refresh fails (401), clear state and redirect to `/login`
- On logout, call `POST /api/v1/auth/logout`, clear all state, redirect to `/login`

### 5.3 API Client (`src/lib/api.ts`)

Centralized fetch wrapper used by all data-fetching hooks:

- Prepends base URL (`/api/v1`) to all paths
- Attaches `Authorization: Bearer <token>` header from AuthContext
- On 401 response: attempts one token refresh, then retries the original request
- On second 401: triggers logout and redirect to `/login`
- Parses JSON responses and throws typed errors for non-2xx status codes

---

## 6. Navigation Flow

```
                                 +------------------+
                                 |   Landing Page   |
                                 |       (/)        |
                                 +--------+---------+
                                          |
                            +-------------+-------------+
                            |                           |
                     +------v------+             +------v------+
                     |    Login    |             |   Sign Up   |
                     |   (/login)  |             |  (/signup)  |
                     +------+------+             +------+------+
                            |                           |
                            |     +----------------+    |
                            |     | Forgot Password|    |
                            +---->| (/forgot-      |    |
                            |     |  password)     |    |
                            |     +-------+--------+    |
                            |             |             |
                            |     +-------v--------+    |
                            |     | Reset Password |    |
                            |     | (/reset-       |    |
                            |     |  password)     |    |
                            |     +-------+--------+    |
                            |             |             |
                            +------+------+-------------+
                                   |
                                   v
                     +-------------+-------------+
                     |        Dashboard          |
                     |       (/dashboard)        |
                     +---+-------+-------+---+---+
                         |       |       |   |
              +----------+   +--+--+    +--+--+
              |              |     |    |     |
     +--------v--------+    |     |    |  +--v-----------+
     | Repository Detail|    |     |    |  |   Settings   |
     | (/repositories/  |    |     |    |  |  (/settings) |
     |  [id])           |    |     |    |  +--------------+
     +--------+---------+    |     |    |
              |              |     |    +-------+
     +--------v---------+   |     |            |
     | Test Run Detail   |   |     |   +--------v--------+
     | (/repositories/   |   |     |   |  Admin Panel    |
     |  [id]/runs/       |   |     |   |  (/admin)       |
     |  [runId])         |   |     |   |  (admin only)   |
     +-------------------+   |     |   +-----------------+
                             |     |
                    +--------v-+  +v-----------+
                    | Teams List|  | Team Detail |
                    | (/teams)  +->| (/teams/   |
                    +-----------+  |  [id])     |
                                   +------------+
```

### Navigation Summary

| From | To | Trigger |
|------|----|---------|
| Landing | Login | "Login" CTA button |
| Landing | Sign Up | "Sign Up" CTA button |
| Login | Dashboard | Successful authentication |
| Login | Sign Up | "Don't have an account?" link |
| Login | Forgot Password | "Forgot password?" link |
| Sign Up | Dashboard | Successful registration |
| Sign Up | Login | "Already have an account?" link |
| Forgot Password | Login | "Back to login" link |
| Reset Password | Login | Successful password reset (auto-redirect) |
| Dashboard | Repository Detail | Click "Dash ->" on RepoCard |
| Repository Detail | Test Run Detail | Click "Detail ->" on SuiteCard run |
| Repository Detail | Dashboard | Breadcrumb "Repositories" link |
| Test Run Detail | Repository Detail | Breadcrumb repo name link |
| Dashboard | Teams List | Sidebar "Teams" link |
| Teams List | Team Detail | Click on TeamCard |
| Team Detail | Teams List | Breadcrumb "Teams" link |
| Dashboard | Team Discovery | Redirect when user has no team memberships |
| Dashboard | Admin Panel | Sidebar "Admin" link (moderator/root only) |
| Team Discovery | Team Detail | After join request is approved and user has team membership |
| Team Detail | Join Requests | "Join Requests" tab/link or "Manage Requests" button (admin/maintainer) |
| Join Requests | Team Detail | Breadcrumb team name link |
| Any page | Settings | TopBar user menu "Settings" |
| Any page | Login | TopBar user menu "Sign Out" |

---

## 7. Data Fetching Strategy

### Server Components vs Client Components

| Pattern | When to Use | Example Pages |
|---------|-------------|---------------|
| Server component with `fetch()` | Initial page data that benefits from SSR (faster first paint, SEO for landing) | Landing, Dashboard (initial load), Teams List |
| Client component with hooks | Interactive forms, real-time polling, user-triggered mutations | Login, Sign Up, Settings, Test Run Detail (polling) |
| Hybrid (server initial + client interactive) | Pages with both static data and interactive sections | Repository Detail, Team Detail, Admin Panel |

### Caching and Revalidation

- Server-side fetches use Next.js `fetch()` with `{ next: { revalidate: 60 } }` for list data (repos, teams)
- Mutations (POST, PUT, DELETE) call `revalidatePath()` or `revalidateTag()` to invalidate cached data
- Client-side interactive data uses `useEffect` + state or SWR-like patterns (custom hooks in `src/hooks/`)
- Test run polling uses raw `setInterval` with cleanup -- no caching for actively polling data

### Error Handling Pattern

Each page uses a Next.js `error.tsx` file alongside `page.tsx`:

```
src/app/(dashboard)/repositories/[id]/
  page.tsx        # Main page component
  loading.tsx     # Loading UI (skeleton)
  error.tsx       # Error boundary
```

The `error.tsx` component receives the error and a `reset` function. It displays a user-friendly message with a "Try Again" button that calls `reset()`.

---

## 8. TypeScript Types

All frontend types are defined in `src/types/` and mirror the API response shapes:

### `src/types/user.ts`

```typescript
export type UserRole = 'user' | 'moderator' | 'root';

export interface User {
  id: string;
  username: string;
  email: string;
  role: UserRole;
  avatar_url: string | null;
  created_at: string;
  updated_at: string;
}
```

### `src/types/repository.ts`

```typescript
export interface Repository {
  id: string;
  github_repo_id: number;
  github_full_name: string;
  name: string;
  description: string;
  default_branch: string;
  is_active: boolean;
  suite_count?: number;
  latest_run_status?: RunStatus;
  created_at: string;
  updated_at: string;
}

export interface Branch {
  name: string;
  is_default: boolean;
}

export interface Commit {
  sha: string;
  message: string;
  author: string;
  date: string;
}
```

### `src/types/test.ts`

```typescript
export type RunStatus = 'pending' | 'running' | 'passed' | 'failed' | 'cancelled' | 'error';
export type TestStatus = 'pass' | 'fail' | 'skip' | 'error';
export type SuiteType = 'unit' | 'integration';

export interface TestSuite {
  id: string;
  repository_id: string;
  name: string;
  type: SuiteType;
  command: string;
  created_at: string;
  updated_at: string;
}

export interface TestRun {
  id: string;
  suite_id: string;
  branch: string;
  commit_sha: string;
  run_number: number;
  status: RunStatus;
  total: number;
  passed: number;
  failed: number;
  skipped: number;
  duration_ms: number;
  started_at: string;
  finished_at: string | null;
  created_at: string;
}

export interface TestResult {
  id: string;
  run_id: string;
  test_name: string;
  status: TestStatus;
  duration_ms: number;
  error_message: string | null;
  output: string | null;
}
```

### `src/types/team.ts`

```typescript
export type TeamRole = 'admin' | 'maintainer' | 'viewer';
export type MemberStatus = 'active' | 'pending';

export interface Team {
  id: string;
  name: string;
  description: string;
  member_count: number;
  repo_count: number;
  created_at: string;
  updated_at: string;
}

export interface TeamMember {
  user_id: string;
  username: string;
  email: string;
  avatar_url: string | null;
  role: TeamRole;
  status: MemberStatus;
  joined_at: string;
}
```

### `src/types/api.ts`

```typescript
export interface PaginationMeta {
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
}

export interface PaginatedResponse<T> {
  data: T[];
  meta: PaginationMeta;
}

export interface ApiError {
  error: {
    code: string;
    message: string;
    details?: Record<string, string>;
  };
}
```
