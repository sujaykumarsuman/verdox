# Verdox Usage Guide

Welcome to Verdox, a test orchestration platform that helps you connect your GitHub repositories, create and manage test suites, run tests, and collaborate with your team. This guide walks you through everything you need to get started and make the most of the platform.

---

## 1. Getting Started

### Create an Account

1. Navigate to the Verdox landing page at `/`.
2. Click **Sign Up** in the navigation bar.
3. Fill in the registration form:
   - **Username**: 3 to 30 characters.
   - **Email**: A valid email address.
   - **Password**: At least 8 characters, with at least 1 uppercase letter, 1 lowercase letter, and 1 digit.
4. Click **Sign Up**.
5. You will be redirected to the dashboard automatically.

### Log In

1. Navigate to `/login` or click **Login** in the navigation bar.
2. Enter your **username or email** and **password**.
3. Click **Login**.
4. If you have forgotten your password, click the **Forgot Password** link on the login page to initiate a password reset.

### Log Out

Click your user avatar in the top bar and select **Logout**. You will be returned to the landing page.

---

## 2. Dashboard Overview

After logging in, you land on the dashboard (`/dashboard`). The dashboard is your central hub and is organized into three areas:

- **Left Sidebar** -- Navigation links to your key sections:
  - **Repository** -- Your connected GitHub repositories.
  - **Teams** -- Teams you belong to or manage.
  - **Test** -- Quick access to test-related features.

- **Top Bar** -- Always visible across the application:
  - **Verdox logo** -- Click to return to the dashboard.
  - **Theme toggle** -- Switch between light and dark mode (sun/moon icon).
  - **Notification bell** -- View recent notifications.
  - **User avatar** -- Access your profile, settings, admin panel, and logout.

- **Main Area** -- Displays your synced repositories as cards. Each card provides quick actions for running tests and navigating to repository details.

---

## 3. Connecting GitHub

### First-Time Setup (Team Admin)

A **team admin** must configure a GitHub Personal Access Token (PAT) for the team before repositories can be added. Navigate to your **team detail page** and find the **GitHub PAT** section. Enter a token with `repo` scope. Generate one at [github.com/settings/tokens](https://github.com/settings/tokens).

For detailed instructions on creating and maintaining a GitHub PAT, see [GITHUB-PAT-GUIDE.md](./GITHUB-PAT-GUIDE.md).

All team members can see whether a PAT is configured (status indicator on the team page), but only team admins can set, rotate, or revoke it.

### Adding Repositories

1. The team must have a GitHub PAT configured before adding repositories.
2. Add a repository by entering its GitHub URL (e.g., `https://github.com/owner/repo`). Verdox clones it locally using the team's PAT.
3. Only team admins, maintainers, and root can add repositories.
4. Added repositories appear as cards on the dashboard.
5. Each repository card shows:
   - **Repository name**
   - **Run a Test** button -- Start a test run directly from the dashboard.
   - **Dash -->** link -- Navigate to the full repository detail page.

---

## 4. Repository Management

### Viewing a Repository

1. Click **Dash -->** on a repository card, or click the repository name directly.
2. You will be taken to the repository detail page (`/repositories/:id`), where you can see:
   - **Branch selector** -- A dropdown to pick the active branch. Defaults to `main`.
   - **Commit hash** -- The latest commit on the selected branch.
   - **Test suites** -- All suites configured for this repository, listed by type (Unit Test, Integration Test).

### Browsing Branches and Commits

- Use the **branch dropdown** to switch between branches. The commit hash and associated data update automatically when you change branches.
- Recent commits are displayed with their **SHA**, **commit message**, and **author**, so you can identify exactly which code will be tested.

---

## 5. Test Suites

Test suites define what tests to run and how to run them. Each suite belongs to a single repository and has a type (Unit or Integration).

### Creating a Test Suite

1. On the repository detail page, click **Add Suite** (or configure suites through a `verdox.yaml` file in your repository).
2. Fill in the suite details:
   - **Name** -- A descriptive name, such as "Unit Tests" or "API Integration Tests".
   - **Type** -- Choose **Unit** or **Integration**.
   - **Config Path** -- Path to a test configuration file within the repository (optional).
   - **Timeout** -- Maximum run time in seconds. The default is 300 seconds (5 minutes).
3. Click **Create**.

The new suite will appear on the repository detail page, ready to run.

### Editing a Suite

1. Click the **settings icon** on the suite card.
2. Update any of the following: name, type, config path, or timeout.
3. Click **Save** to apply your changes.

### Deleting a Suite

1. Click the **delete icon** on the suite card.
2. Confirm the deletion when prompted.

Note: Deleting a suite also permanently deletes all associated test runs and their results.

---

## 6. Running Tests

### Run a Single Suite

1. On the repository detail page, find the suite you want to run.
2. Click the **Run** button on the suite card.
3. Select a **branch** and **commit** (defaults to the currently selected branch and its latest commit).
4. The test run begins. Its status will display as **Queued** and then transition to **Running** as execution starts.

### Run All Suites

1. At the top of the repository detail page, click **Run All**.
2. All suites configured for the repository will start on the selected branch and commit.
3. Each suite runs independently and reports its own status.

### Automatic Runs (Webhooks)

When webhooks are configured for your repository, pushing to a branch automatically triggers test runs for any matching suites. This lets you integrate Verdox into your CI workflow without manual intervention.

---

## 7. Viewing Test Results

### Test Run Overview

On the repository detail page, each suite card displays a summary of its most recent run:

- **Progress bar** -- Shows the pass count relative to the total number of tests.
- **Status badge** -- Indicates the current state of the run:
  - **Passed** (green) -- All tests passed.
  - **Failed** (red) -- One or more tests failed.
  - **Running** (blue) -- Tests are currently executing.
  - **Queued** (gray) -- Waiting for a runner.
- **Detail -->** link -- Click to view the full results.

### Test Run Detail Page

1. Click **Detail -->** on a suite to open the test run detail page (`/repositories/:id/runs/:runId`).
2. The detail page includes:
   - **Run header** -- Suite name, branch, commit hash, and run number (for example, Run #2).
   - **Run Logs** button -- View the full output from the test execution.
   - **Per-test results** -- Each individual test displayed as a row with:
     - **Test name**
     - **Status** -- Pass, Fail, Skip, or Error (see table below).
     - **Duration** -- How long the test took.
     - **Expand icon** -- Click to view the logs for that specific test.

### Understanding Statuses

| Status    | Meaning                                    | Color  |
|-----------|--------------------------------------------|--------|
| Pass      | Test passed successfully                   | Green  |
| Fail      | Test assertion failed                      | Red    |
| Skip      | Test was skipped                           | Yellow |
| Error     | Test errored (not an assertion failure)    | Red    |
| Queued    | Waiting to run                             | Gray   |
| Running   | Currently executing                        | Blue   |
| Cancelled | Manually cancelled                         | Gray   |

### Cancelling a Run

While a test run is in the **Queued** or **Running** state, you can cancel it:

1. Click the **cancel button** next to the run.
2. The status changes to **Cancelled** and execution stops.

---

## 8. Team Management

Teams let you collaborate with other Verdox users. You can share repositories with your team and control access through roles.

### Creating a Team

1. Navigate to the **Teams** page from the left sidebar (`/teams`).
2. Click **Create New**.
3. Enter a name for your team.
4. Click **Create**.

You automatically become the **admin** of the new team.

### Joining a Team

1. Browse available teams on the **Team Discovery** page.
2. Click **Request to Join** on any team. You can include an optional message.
3. Team admins and maintainers review join requests and approve or reject them with a role assignment.

### Managing Members

- **Approve or Reject** -- Team admins and maintainers see **approve** (checkmark) and **reject** (X) buttons next to pending join requests. When approving, they assign a role to the new member.
- **Change Role** -- Admins can click the role badge next to a member's name and select a new role.
- **Remove** -- Click the **remove** button to remove a member from the team.

### Team Roles

| Role       | Capabilities                                          |
|------------|-------------------------------------------------------|
| Admin      | Manage members, assign repositories, delete the team  |
| Maintainer | Approve and reject join requests                      |
| Viewer     | View team repositories, run tests                     |

### Assigning Repositories to a Team

1. On the team detail page, find the **Repo** panel.
2. Click **+** to assign a repository.
3. Select from the list of your repositories.
4. Once assigned, all team members can access and run tests on that repository.
5. To unassign a repository, click **-** next to the repository name.

---

## 9. User Settings

1. Click your **avatar** in the top bar and select **Settings** (or navigate to `/settings`).
2. Available settings:
   - **Profile** -- Update your username, email address, and avatar image.
   - **Password** -- Change your password. You must enter your current password along with the new one.
   - **Theme** -- Toggle between dark and light mode. This is the same control available via the sun/moon icon in the top bar.

> **Note:** GitHub PAT configuration is managed at the team level. Team admins can set, rotate, or revoke the PAT from the team detail page.

---

## 10. Dark Mode

Verdox supports both light and dark themes.

- Click the **sun/moon icon** in the top bar to switch themes.
- Your preference is saved and persists across sessions, so you will always see your preferred theme when you log back in.
- You can also change the theme from the **Settings** page.

---

## 11. Admin Panel (Admins Only)

If your account has the **root** or **moderator** role, you have access to the admin panel.

1. Click your **avatar** in the top bar and select **Admin**, or navigate directly to `/admin`.
2. The admin panel includes:
   - **Users** -- View a list of all registered users. Search by username or email to find specific accounts.
   - **Promote to Moderator** (root only) -- Root can promote users to the moderator role.
   - **Deactivate Users** -- Moderators can view users and deactivate accounts (cannot change roles).
   - **System Stats** -- View platform-wide statistics including total users, repositories, test runs, and active runners.

---

## 12. Keyboard Shortcuts (Future)

Keyboard shortcuts for power users are planned for a future release. This section will be updated when they become available.

---

## 13. FAQ

**Q: How many repositories can I connect?**
A: You can connect up to 100 repositories per user account.

**Q: How many tests can run at the same time?**
A: Up to 5 concurrent test runs by default. This limit is configurable by your platform admin.

**Q: How long are test results kept?**
A: Test results are retained for 90 days by default. Your admin can adjust this retention period.

**Q: What test frameworks are supported?**
A: Verdox supports Go test, pytest, and Jest out of the box. You can also configure custom test commands through a `verdox.yaml` file in your repository.

**Q: Can I use private repositories?**
A: Yes. A team admin must configure a GitHub PAT with `repo` scope in the team settings, and Verdox can access both public and private repositories on behalf of all team members.

**Q: How do I add a repository?**
A: Team admins can add repos by URL. The repo is cloned locally to the server.

---

For additional help, reach out to your Verdox administrator or visit the project repository for technical documentation.
