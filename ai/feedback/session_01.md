# Prompt: Refine and Address These Verdox Requirements

Review the following rough plan and refine it based on feasibility, but do not drop any statement and do not change my intent. If a point is phrased as a belief, possibility, or tentative idea, preserve that nuance and evaluate it instead of turning it into a hard requirement.

This is for Verdox. I want a response that turns these notes into a concrete, feasible product and technical direction, while keeping every requirement intact.

Also review this reference document and check whether a similar approach can be used here where relevant:

- `ai/res/docs/live-state-updates-architecture.md`

In your response:

- preserve every point below
- refine the ideas into a practical plan
- call out feasibility, tradeoffs, dependencies, and missing prerequisites
- recommend the best-fit architecture, UX flow, and implementation approach
- be explicit about where GitHub Actions, notifications, email, webhooks, SSE, polling, or similar mechanisms should be used

## 1. Redesign How Tests Are Run

I want to approach a finalised way of how I want to run tests. Review my rough plan below and refine it based on feasibility.

- The application should be deployed with an initial root account.
- This root account should have username `root` and email `<email used for github service account>`.
- The service account is supposed to be just another GitHub account used to manage Verdox-related work, such as:
  - generating PAT
  - forking repo
  - adding Verdox workflows to run Actions
  - syncing upstream of the fork to fetch new changes in the test repo
- I believe this removes the dependency on users having to update the main testing repo and allows testing changes on this fork repo.
- This way, we do not even need to provide a local Docker runner.
- This can be used to run any type of test suite allowed by GitHub Actions runners.
- It will also allow workflows to generate JSON result output artifacts for Verdox to fetch and render in the UI dashboard.
- Test run logs and run history can be maintained in GitHub and would not need to be maintained at Verdox's end.

## 2. Notifications

- Add a notifications page.
- This should contain messages sent by admins and system notifications such as unban requests.
- The page can show a detailed view of a notification message or system notification.
- The notification bell should open a submenu like the profile menu, with a minimal list of unread notifications showing only subjects, and an option to open the full notifications page with message bodies.
- The notifications page should follow a format similar to an email inbox.
- System notification bodies should support functional actions. For example, for ban review requests, approve and deny buttons should be available in the notification body itself even though the admin panel might also have the full list of requests.

## 3. Admin Actions and UI State Update

### Live State and Effect in UI

- Previously attempted in phase 5, I wanted to refine the ban/unban process and UX around it but could not finalise a working approach.
- I had a similar issue resolved in another project and have a reference document around the fix. Please go through `ai/res/docs/live-state-updates-architecture.md`.
- Check whether something similar can be done here to achieve the following:
  - when root/admins ban a user who is active on Verdox, the user should be redirected to the banned page
  - if that helps, have a webhook to notify the banned user to redirect
  - when a review is requested, admins should be notified immediately in notifications
  - maybe have a webhook for each user that can be used to trigger user-specific actions
  - an admin action such as ban taken on a user should force that user into a state where only the banned page and sign-out option are available for as long as the user is banned, with no access to other pages

## 4. Send Mail to Users

- Admins/root should be able to send email to a set of users based on type, individually selected users, all users, or users filtered by status and similar filters.
- The messages should also reflect in the receiver user's notifications, in the mini notification bell box, and in the full notifications page.

## Response Expectation

Do not simplify these requirements by removing details. I want them refined into an actionable prompt and implementation direction without changing the meaning of any point.
