# Verdox -- PostgreSQL Database Schema (LLD)

> PostgreSQL 17 | UUID primary keys | golang-migrate

---

## 1. Entity Relationship Diagram (ASCII)

```
 +-------------+        +------------------+        +----------------+
 |   users     |        |   repositories   |        |   test_suites  |
 |-------------|        |------------------|        |----------------|
 | id (PK)     |<--+    | id (PK)          |<--+    | id (PK)        |
 | username     |   |    | github_repo_id   |   |   | repository_id  |---+
 | email        |   |    | github_full_name |   |   | name           |   |
 | password_hash|   |    | name             |   |   | type           |   |
 | role         |   |    | description      |   |   | config_path    |   |
 | is_active    |   |    | default_branch   |   |   | timeout_seconds|   |
 | is_banned    |   |    | fork_full_name   |   |   | execution_mode |   |
 | avatar_url   |   |    | fork_status      |   |   | env_vars       |   |
 | created_at   |   |    | fork_synced_at   |   |   | workflow_config|   |
 | updated_at   |   |    | is_active        |   |   | created_at     |   |
 +------+-------+   |    | created_at       |   |   | updated_at     |   |
        |            |    | updated_at       |   |   +-------+--------+   |
        |            |    +--------+---------+   |           |            |
        |            |             |              |           |            |
        |            |             |              |    +------v--------+   |
 +------v-------+   |             |              |    |  test_runs    |   |
 |  sessions    |   |             |              |    |---------------|   |
 |--------------|   |             |              |    | id (PK)       |   |
 | id (PK)      |   |             |              |    | test_suite_id |---+
 | user_id (FK)-+---+             |              |    | triggered_by  |--+
 | refresh_     |   |             |              |    | run_number    |  |
 |  token_hash  |   |             |              |    | branch        |  |
 | expires_at   |   |             |              |    | commit_hash   |  |
 | created_at   |   |             |              |    | status        |  |
 +------------- +   |             |              |    | gha_run_id    |  |
                     |             |              |    | summary       |  |
                     |             |              |    | created_at    |  |
                     |             |              |    +------+--------+  |
                     |             |              |           |           |
                     |    +--------v---------+    |    +------v--------+  |
                     |    | team_repositories |   |    | test_results  |  |
                     |    |  (junction)       |   |    |---------------|  |
                     |    |------------------|   |    | id (PK)       |  |
                     |    | id (PK)          |   |    | test_run_id   |  |
                     |    | team_id (FK)-----+---+    | test_name     |  |
                     |    | repository_id(FK)+--+|    | status        |  |
                     |    | added_by (FK)----+--+|    | duration_ms   |  |
                     |    | created_at       |   |    | error_message |  |
                     |    +------------------+   |    | log_output    |  |
                     |                           |    | created_at    |  |
                     |    +------------------+   |    +------+--------+  |
                     |    |  team_members    |   |           |           |
                     |    |  (junction)      |   |    +------v--------+  |
                     |    |------------------|   |    | test_groups   |  |
                     +----+-user_id (FK)     |   |    |---------------|  |
                     |    | team_id (FK)-----+---+    | id (PK)       |  |
                     +----+-invited_by (FK)  |   |    | test_run_id   |  |
                          | id (PK)          |   |    | name          |  |
                          | role             |   |    | status        |  |
                          | status           |   |    | created_at    |  |
                          | created_at       |   |    +------+--------+  |
                          +--------+---------+   |           |           |
                                   |             |    +------v--------+  |
                          +--------v---------+   |    | test_cases    |  |
                          | team_join_requests|   |    |---------------|  |
                          |------------------|   |    | id (PK)       |  |
                          | id (PK)          |   |    | test_group_id |  |
                          | team_id (FK)     |   |    | test_run_id   |  |
                          | user_id (FK)     |   |    | name          |  |
                          | message          |   |    | status        |  |
                          | status           |   |    | duration_ms   |  |
                          | reviewed_by (FK) |   |    | created_at    |  |
                          | role_assigned    |   |    +---------------+  |
                          | created_at       |   |                       |
                          | updated_at       |   |   +-----v-----------------+
                          +------------------+   |   |   teams               |
                                                 |   |------------------------|
 +------------------+    +------------------+    |   | id (PK)                |
 |  ban_reviews     |    |  notifications   |    |   | name                   |
 |------------------|    |------------------|    |   | slug                   |
 | id (PK)          |    | id (PK)          |    +---+ created_by             |
 | user_id (FK)     |    | user_id (FK)     |        | is_discoverable        |
 | ban_reason       |    | type             |        | github_pat_encrypted   |
 | clarification    |    | subject          |        | github_pat_nonce       |
 | status           |    | body             |        | github_pat_set_at      |
 | reviewed_by (FK) |    | is_read          |        | github_pat_set_by (FK) |
 | created_at       |    | sender_id (FK)   |        | github_pat_github_user |
 | reviewed_at      |    | created_at       |        | created_at             |
 +------------------+    +------------------+        | updated_at             |
                                                     | deleted_at             |
                                                     +------------------------+
                                                                     |
                          users.id <---------------------------------+
```

### Relationship summary

| Relationship                          | Type | Via                          |
| ------------------------------------- | ---- | ---------------------------- |
| users 1 : N sessions                  | 1:N  | sessions.user_id             |
| users 1 : N teams (creator)           | 1:N  | teams.created_by             |
| users 1 : N teams (PAT setter)        | 1:N  | teams.github_pat_set_by      |
| teams N : M users                     | M:N  | team_members                 |
| teams 1 : N team_join_requests        | 1:N  | team_join_requests.team_id   |
| users 1 : N team_join_requests        | 1:N  | team_join_requests.user_id   |
| teams N : M repositories              | M:N  | team_repositories            |
| repositories 1 : N test_suites        | 1:N  | test_suites.repository_id    |
| test_suites 1 : N test_runs           | 1:N  | test_runs.test_suite_id      |
| test_runs 1 : N test_results          | 1:N  | test_results.test_run_id     |
| test_runs 1 : N test_groups           | 1:N  | test_groups.test_run_id      |
| test_groups 1 : N test_cases          | 1:N  | test_cases.test_group_id     |
| test_runs 1 : N test_cases            | 1:N  | test_cases.test_run_id       |
| users 1 : N test_runs (trigger)       | 1:N  | test_runs.triggered_by       |
| users 1 : N ban_reviews               | 1:N  | ban_reviews.user_id          |
| users 1 : N notifications             | 1:N  | notifications.user_id        |

---

## 2. Enum Types

```sql
CREATE TYPE user_role AS ENUM ('root', 'admin', 'moderator', 'user');

CREATE TYPE team_member_role AS ENUM ('admin', 'maintainer', 'viewer');

CREATE TYPE team_member_status AS ENUM ('pending', 'approved', 'rejected');

CREATE TYPE test_run_status AS ENUM ('queued', 'running', 'passed', 'failed', 'cancelled');

CREATE TYPE test_result_status AS ENUM ('pass', 'fail', 'skip', 'error', 'running', 'unknown');

CREATE TYPE notification_type AS ENUM ('system', 'admin_message', 'ban_review', 'test_complete', 'team_invite', 'team_join_request');
```

---

## 3. CREATE TABLE Statements

### 3.1 users

```sql
CREATE TABLE users (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    username        VARCHAR(64)  NOT NULL,
    email           VARCHAR(255) NOT NULL,
    password_hash   TEXT         NOT NULL,
    role            user_role    NOT NULL DEFAULT 'user',
    is_active       BOOLEAN      NOT NULL DEFAULT TRUE,
    is_banned       BOOLEAN      NOT NULL DEFAULT FALSE,
    ban_reason      TEXT,
    avatar_url      TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uq_users_username UNIQUE (username),
    CONSTRAINT uq_users_email    UNIQUE (email)
);
```

### 3.2 sessions

```sql
CREATE TABLE sessions (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            UUID        NOT NULL,
    refresh_token_hash TEXT        NOT NULL,
    expires_at         TIMESTAMPTZ NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT fk_sessions_user
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE
);
```

### 3.3 repositories

```sql
CREATE TABLE repositories (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    github_repo_id    BIGINT       NOT NULL,
    github_full_name  VARCHAR(255) NOT NULL,
    name              VARCHAR(255) NOT NULL,
    description       TEXT,
    default_branch    VARCHAR(255) NOT NULL DEFAULT 'main',
    fork_full_name    VARCHAR(255),
    fork_status       VARCHAR(32)  NOT NULL DEFAULT 'none',
    fork_synced_at    TIMESTAMPTZ,
    fork_workflow_id  VARCHAR(255),
    fork_head_sha     VARCHAR(64),
    is_active         BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uq_repositories_github_repo_id UNIQUE (github_repo_id)
);
```

### 3.4 teams

```sql
CREATE TABLE teams (
    id                       UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name                     VARCHAR(128) NOT NULL,
    slug                     VARCHAR(128) NOT NULL,
    created_by               UUID,
    is_discoverable          BOOLEAN      NOT NULL DEFAULT true,
    github_pat_encrypted     TEXT,
    github_pat_nonce         BYTEA,
    github_pat_set_at        TIMESTAMPTZ,
    github_pat_set_by        UUID,
    github_pat_github_username VARCHAR(255),
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at               TIMESTAMPTZ,          -- soft-delete timestamp; NULL = active

    CONSTRAINT uq_teams_name UNIQUE (name),
    CONSTRAINT uq_teams_slug UNIQUE (slug),

    CONSTRAINT fk_teams_created_by
        FOREIGN KEY (created_by)
        REFERENCES users (id)
        ON DELETE SET NULL,

    CONSTRAINT fk_teams_pat_set_by
        FOREIGN KEY (github_pat_set_by)
        REFERENCES users (id)
        ON DELETE SET NULL
);
```

### 3.5 team_members

```sql
CREATE TABLE team_members (
    id          UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id     UUID               NOT NULL,
    user_id     UUID               NOT NULL,
    role        team_member_role   NOT NULL DEFAULT 'viewer',
    status      team_member_status NOT NULL DEFAULT 'pending',
    invited_by  UUID,
    created_at  TIMESTAMPTZ        NOT NULL DEFAULT now(),

    CONSTRAINT uq_team_members_team_user UNIQUE (team_id, user_id),

    CONSTRAINT fk_team_members_team
        FOREIGN KEY (team_id)
        REFERENCES teams (id)
        ON DELETE CASCADE,

    CONSTRAINT fk_team_members_user
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE,

    CONSTRAINT fk_team_members_invited_by
        FOREIGN KEY (invited_by)
        REFERENCES users (id)
        ON DELETE SET NULL
);
```

### 3.6 team_join_requests

```sql
CREATE TABLE team_join_requests (
    id              UUID                PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID                NOT NULL,
    user_id         UUID                NOT NULL,
    message         TEXT,
    status          team_member_status  NOT NULL DEFAULT 'pending',
    reviewed_by     UUID,
    role_assigned   team_member_role,
    created_at      TIMESTAMPTZ         NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ         NOT NULL DEFAULT now(),

    CONSTRAINT uq_join_requests_team_user UNIQUE (team_id, user_id),

    CONSTRAINT fk_join_requests_team
        FOREIGN KEY (team_id)
        REFERENCES teams (id)
        ON DELETE CASCADE,

    CONSTRAINT fk_join_requests_user
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE,

    CONSTRAINT fk_join_requests_reviewer
        FOREIGN KEY (reviewed_by)
        REFERENCES users (id)
        ON DELETE SET NULL
);

CREATE INDEX idx_join_requests_team_id ON team_join_requests (team_id);
CREATE INDEX idx_join_requests_user_id ON team_join_requests (user_id);
CREATE INDEX idx_join_requests_status ON team_join_requests (status);
```

### 3.7 team_repositories

> **Cascade behavior:** When a team is soft-deleted (`deleted_at` is set),
> all associated repositories should be marked inactive (`is_active = false`).
> This cascade is handled at the application layer, not via database triggers.

```sql
CREATE TABLE team_repositories (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id       UUID        NOT NULL,
    repository_id UUID        NOT NULL,
    added_by      UUID,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_team_repos_team_repo UNIQUE (team_id, repository_id),

    CONSTRAINT fk_team_repos_team
        FOREIGN KEY (team_id)
        REFERENCES teams (id)
        ON DELETE CASCADE,

    CONSTRAINT fk_team_repos_repository
        FOREIGN KEY (repository_id)
        REFERENCES repositories (id)
        ON DELETE CASCADE,

    CONSTRAINT fk_team_repos_added_by
        FOREIGN KEY (added_by)
        REFERENCES users (id)
        ON DELETE SET NULL
);
```

### 3.8 test_suites

```sql
CREATE TABLE test_suites (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id   UUID         NOT NULL,
    name            VARCHAR(255) NOT NULL,
    type            VARCHAR(50)  NOT NULL DEFAULT 'unit',
    config_path     TEXT,
    timeout_seconds INTEGER      NOT NULL DEFAULT 300,
    execution_mode  VARCHAR(20)  NOT NULL DEFAULT 'fork_gha',
    docker_image    VARCHAR(255),
    test_command    TEXT,
    gha_workflow_id VARCHAR(255),
    env_vars        JSONB        NOT NULL DEFAULT '{}',
    workflow_config JSONB        NOT NULL DEFAULT '{}',
    workflow_yaml   TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT fk_test_suites_repository
        FOREIGN KEY (repository_id)
        REFERENCES repositories (id)
        ON DELETE CASCADE
);
```

### 3.9 test_runs

```sql
CREATE TABLE test_runs (
    id            UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    test_suite_id UUID            NOT NULL,
    triggered_by  UUID,
    run_number    INTEGER         NOT NULL DEFAULT 1,
    branch        VARCHAR(255)    NOT NULL,
    commit_hash   CHAR(40)        NOT NULL,
    status        test_run_status NOT NULL DEFAULT 'queued',
    gha_run_id    BIGINT,
    log_output    TEXT,
    summary       JSONB,
    report_id     VARCHAR(255),
    started_at    TIMESTAMPTZ,
    finished_at   TIMESTAMPTZ,
    created_at    TIMESTAMPTZ     NOT NULL DEFAULT now(),

    CONSTRAINT fk_test_runs_suite
        FOREIGN KEY (test_suite_id)
        REFERENCES test_suites (id)
        ON DELETE CASCADE,

    CONSTRAINT fk_test_runs_triggered_by
        FOREIGN KEY (triggered_by)
        REFERENCES users (id)
        ON DELETE SET NULL
);
```

### 3.10 test_results

```sql
CREATE TABLE test_results (
    id            UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    test_run_id   UUID               NOT NULL,
    test_name     VARCHAR(512)       NOT NULL,
    status        test_result_status NOT NULL,
    duration_ms   INTEGER,
    error_message TEXT,
    log_output    TEXT,
    created_at    TIMESTAMPTZ        NOT NULL DEFAULT now(),

    CONSTRAINT fk_test_results_run
        FOREIGN KEY (test_run_id)
        REFERENCES test_runs (id)
        ON DELETE CASCADE
);
```

### 3.11 test_groups

```sql
CREATE TABLE test_groups (
    id            UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    test_run_id   UUID               NOT NULL,
    group_id      VARCHAR(255),
    name          VARCHAR(512),
    package       VARCHAR(1024),
    status        test_result_status DEFAULT 'unknown',
    total         INTEGER,
    passed        INTEGER,
    failed        INTEGER,
    skipped       INTEGER,
    duration_ms   INTEGER,
    pass_rate     NUMERIC(5,2),
    sort_order    INTEGER,
    created_at    TIMESTAMPTZ        NOT NULL DEFAULT now(),

    CONSTRAINT fk_test_groups_run
        FOREIGN KEY (test_run_id)
        REFERENCES test_runs (id)
        ON DELETE CASCADE
);

CREATE INDEX idx_test_groups_run_id ON test_groups (test_run_id);
```

### 3.12 test_cases

```sql
CREATE TABLE test_cases (
    id             UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    test_group_id  UUID               NOT NULL,
    test_run_id    UUID               NOT NULL,
    case_id        VARCHAR(512),
    name           VARCHAR(512),
    status         test_result_status,
    duration_ms    INTEGER,
    error_message  TEXT,
    stack_trace    TEXT,
    retry_count    INTEGER            DEFAULT 0,
    logs_url       TEXT,
    created_at     TIMESTAMPTZ        NOT NULL DEFAULT now(),

    CONSTRAINT fk_test_cases_group
        FOREIGN KEY (test_group_id)
        REFERENCES test_groups (id)
        ON DELETE CASCADE,

    CONSTRAINT fk_test_cases_run
        FOREIGN KEY (test_run_id)
        REFERENCES test_runs (id)
        ON DELETE CASCADE
);

CREATE INDEX idx_test_cases_group_id ON test_cases (test_group_id);
CREATE INDEX idx_test_cases_run_id   ON test_cases (test_run_id);
```

### 3.13 ban_reviews

```sql
CREATE TABLE ban_reviews (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID         NOT NULL,
    ban_reason    TEXT         NOT NULL,
    clarification TEXT         NOT NULL,
    status        VARCHAR(20)  DEFAULT 'pending',
    reviewed_by   UUID,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    reviewed_at   TIMESTAMPTZ,

    CONSTRAINT fk_ban_reviews_user
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE,

    CONSTRAINT fk_ban_reviews_reviewer
        FOREIGN KEY (reviewed_by)
        REFERENCES users (id)
        ON DELETE SET NULL
);

CREATE INDEX idx_ban_reviews_user_id ON ban_reviews (user_id);
CREATE INDEX idx_ban_reviews_status  ON ban_reviews (status);
```

### 3.14 notifications

```sql
CREATE TABLE notifications (
    id             UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID              NOT NULL,
    type           notification_type,
    subject        VARCHAR(255),
    body           TEXT              DEFAULT '',
    is_read        BOOLEAN           DEFAULT false,
    action_type    VARCHAR(64),
    action_payload JSONB,
    sender_id      UUID,
    created_at     TIMESTAMPTZ       NOT NULL DEFAULT now(),

    CONSTRAINT fk_notifications_user
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE,

    CONSTRAINT fk_notifications_sender
        FOREIGN KEY (sender_id)
        REFERENCES users (id)
        ON DELETE SET NULL
);

CREATE INDEX idx_notifications_user_id ON notifications (user_id);
CREATE INDEX idx_notifications_user_read ON notifications (user_id, is_read);
```

---

## 4. Indexes

```sql
-- users
CREATE UNIQUE INDEX idx_users_username ON users (username);
CREATE UNIQUE INDEX idx_users_email    ON users (email);

-- sessions
CREATE INDEX idx_sessions_user_id    ON sessions (user_id);
CREATE INDEX idx_sessions_expires_at ON sessions (expires_at);

-- repositories
CREATE UNIQUE INDEX idx_repositories_github_repo_id ON repositories (github_repo_id);

-- teams
CREATE UNIQUE INDEX idx_teams_slug ON teams (slug);

-- team_members
CREATE UNIQUE INDEX idx_team_members_team_user ON team_members (team_id, user_id);
CREATE INDEX        idx_team_members_user_id   ON team_members (user_id);

-- team_join_requests
CREATE INDEX idx_join_requests_team_id ON team_join_requests (team_id);
CREATE INDEX idx_join_requests_user_id ON team_join_requests (user_id);
CREATE INDEX idx_join_requests_status  ON team_join_requests (status);

-- team_repositories
CREATE UNIQUE INDEX idx_team_repos_team_repo ON team_repositories (team_id, repository_id);

-- test_suites
CREATE INDEX idx_test_suites_repository_id ON test_suites (repository_id);

-- test_runs
CREATE INDEX idx_test_runs_suite_id            ON test_runs (test_suite_id);
CREATE INDEX idx_test_runs_suite_status        ON test_runs (test_suite_id, status);
CREATE INDEX idx_test_runs_triggered_by        ON test_runs (triggered_by);
CREATE INDEX idx_test_runs_suite_branch_commit ON test_runs (test_suite_id, branch, commit_hash);

-- test_results
CREATE INDEX idx_test_results_run_id ON test_results (test_run_id);

-- test_groups
CREATE INDEX idx_test_groups_run_id ON test_groups (test_run_id);

-- test_cases
CREATE INDEX idx_test_cases_group_id ON test_cases (test_group_id);
CREATE INDEX idx_test_cases_run_id   ON test_cases (test_run_id);

-- ban_reviews
CREATE INDEX idx_ban_reviews_user_id ON ban_reviews (user_id);
CREATE INDEX idx_ban_reviews_status  ON ban_reviews (status);

-- notifications
CREATE INDEX idx_notifications_user_id   ON notifications (user_id);
CREATE INDEX idx_notifications_user_read ON notifications (user_id, is_read);
```

> **Note:** The UNIQUE constraints declared inline in the CREATE TABLE
> statements (e.g. `uq_users_username`) already create implicit unique
> indexes. The explicit `CREATE UNIQUE INDEX` statements above are shown
> for completeness; in practice you may omit the duplicate explicit
> indexes and rely on the constraint-backed ones. The non-unique indexes
> (`CREATE INDEX`) are always required separately.

---

## 5. Migration Strategy

**Tool:** [golang-migrate](https://github.com/golang-migrate/migrate)

**File naming convention:**

```
migrations/
  000001_create_enum_types.up.sql
  000001_create_enum_types.down.sql
  000002_create_users.up.sql
  000002_create_users.down.sql
  000003_create_sessions.up.sql
  000003_create_sessions.down.sql
  000004_create_password_resets.up.sql
  000004_create_password_resets.down.sql
  000005_create_repositories.up.sql
  000005_create_repositories.down.sql
  000006_create_teams.up.sql
  000006_create_teams.down.sql
  000007_create_team_members.up.sql
  000007_create_team_members.down.sql
  000008_create_team_join_requests.up.sql
  000008_create_team_join_requests.down.sql
  000009_create_team_repositories.up.sql
  000009_create_team_repositories.down.sql
  000010_create_test_suites.up.sql
  000010_create_test_suites.down.sql
  000011_create_test_runs.up.sql
  000011_create_test_runs.down.sql
  000012_create_test_results.up.sql
  000012_create_test_results.down.sql
  000013_create_test_groups.up.sql
  000013_create_test_groups.down.sql
  000014_create_test_cases.up.sql
  000014_create_test_cases.down.sql
  000015_create_ban_reviews.up.sql
  000015_create_ban_reviews.down.sql
  000016_create_notifications.up.sql
  000016_create_notifications.down.sql
```

### 000001 -- create_enum_types

**UP** `migrations/000001_create_enum_types.up.sql`

```sql
CREATE TYPE user_role          AS ENUM ('root', 'admin', 'moderator', 'user');
CREATE TYPE team_member_role   AS ENUM ('admin', 'maintainer', 'viewer');
CREATE TYPE team_member_status AS ENUM ('pending', 'approved', 'rejected');
CREATE TYPE test_run_status    AS ENUM ('queued', 'running', 'passed', 'failed', 'cancelled');
CREATE TYPE test_result_status AS ENUM ('pass', 'fail', 'skip', 'error', 'running', 'unknown');
CREATE TYPE notification_type  AS ENUM ('system', 'admin_message', 'ban_review', 'test_complete', 'team_invite', 'team_join_request');
```

**DOWN** `migrations/000001_create_enum_types.down.sql`

```sql
DROP TYPE IF EXISTS notification_type;
DROP TYPE IF EXISTS test_result_status;
DROP TYPE IF EXISTS test_run_status;
DROP TYPE IF EXISTS team_member_status;
DROP TYPE IF EXISTS team_member_role;
DROP TYPE IF EXISTS user_role;
```

---

### 000002 -- create_users

**UP** `migrations/000002_create_users.up.sql`

```sql
CREATE TABLE users (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    username        VARCHAR(64)  NOT NULL,
    email           VARCHAR(255) NOT NULL,
    password_hash   TEXT         NOT NULL,
    role            user_role    NOT NULL DEFAULT 'user',
    is_active       BOOLEAN      NOT NULL DEFAULT TRUE,
    is_banned       BOOLEAN      NOT NULL DEFAULT FALSE,
    ban_reason      TEXT,
    avatar_url      TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uq_users_username UNIQUE (username),
    CONSTRAINT uq_users_email    UNIQUE (email)
);
```

**DOWN** `migrations/000002_create_users.down.sql`

```sql
DROP TABLE IF EXISTS users;
```

---

### 000003 -- create_sessions

**UP** `migrations/000003_create_sessions.up.sql`

```sql
CREATE TABLE sessions (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            UUID        NOT NULL,
    refresh_token_hash TEXT        NOT NULL,
    expires_at         TIMESTAMPTZ NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT fk_sessions_user
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE
);

CREATE INDEX idx_sessions_user_id    ON sessions (user_id);
CREATE INDEX idx_sessions_expires_at ON sessions (expires_at);
```

**DOWN** `migrations/000003_create_sessions.down.sql`

```sql
DROP TABLE IF EXISTS sessions;
```

---

### 000004 -- create_password_resets

**UP** `migrations/000004_create_password_resets.up.sql`

```sql
-- password_resets table (migration 000004)
```

**DOWN** `migrations/000004_create_password_resets.down.sql`

```sql
DROP TABLE IF EXISTS password_resets;
```

---

### 000005 -- create_repositories

**UP** `migrations/000005_create_repositories.up.sql`

```sql
CREATE TABLE repositories (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    github_repo_id    BIGINT       NOT NULL,
    github_full_name  VARCHAR(255) NOT NULL,
    name              VARCHAR(255) NOT NULL,
    description       TEXT,
    default_branch    VARCHAR(255) NOT NULL DEFAULT 'main',
    fork_full_name    VARCHAR(255),
    fork_status       VARCHAR(32)  NOT NULL DEFAULT 'none',
    fork_synced_at    TIMESTAMPTZ,
    fork_workflow_id  VARCHAR(255),
    fork_head_sha     VARCHAR(64),
    is_active         BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uq_repositories_github_repo_id UNIQUE (github_repo_id)
);
```

**DOWN** `migrations/000005_create_repositories.down.sql`

```sql
DROP TABLE IF EXISTS repositories;
```

---

### 000006 -- create_teams

**UP** `migrations/000006_create_teams.up.sql`

```sql
CREATE TABLE teams (
    id                       UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name                     VARCHAR(128) NOT NULL,
    slug                     VARCHAR(128) NOT NULL,
    created_by               UUID,
    is_discoverable          BOOLEAN      NOT NULL DEFAULT true,
    github_pat_encrypted     TEXT,
    github_pat_nonce         BYTEA,
    github_pat_set_at        TIMESTAMPTZ,
    github_pat_set_by        UUID,
    github_pat_github_username VARCHAR(255),
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at               TIMESTAMPTZ,          -- soft-delete timestamp; NULL = active

    CONSTRAINT uq_teams_name UNIQUE (name),
    CONSTRAINT uq_teams_slug UNIQUE (slug),

    CONSTRAINT fk_teams_created_by
        FOREIGN KEY (created_by)
        REFERENCES users (id)
        ON DELETE SET NULL,

    CONSTRAINT fk_teams_pat_set_by
        FOREIGN KEY (github_pat_set_by)
        REFERENCES users (id)
        ON DELETE SET NULL
);
```

**DOWN** `migrations/000006_create_teams.down.sql`

```sql
DROP TABLE IF EXISTS teams;
```

---

### 000007 -- create_team_members

**UP** `migrations/000007_create_team_members.up.sql`

```sql
CREATE TABLE team_members (
    id          UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id     UUID               NOT NULL,
    user_id     UUID               NOT NULL,
    role        team_member_role   NOT NULL DEFAULT 'viewer',
    status      team_member_status NOT NULL DEFAULT 'pending',
    invited_by  UUID,
    created_at  TIMESTAMPTZ        NOT NULL DEFAULT now(),

    CONSTRAINT uq_team_members_team_user UNIQUE (team_id, user_id),

    CONSTRAINT fk_team_members_team
        FOREIGN KEY (team_id)
        REFERENCES teams (id)
        ON DELETE CASCADE,

    CONSTRAINT fk_team_members_user
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE,

    CONSTRAINT fk_team_members_invited_by
        FOREIGN KEY (invited_by)
        REFERENCES users (id)
        ON DELETE SET NULL
);

CREATE INDEX idx_team_members_user_id ON team_members (user_id);
```

**DOWN** `migrations/000007_create_team_members.down.sql`

```sql
DROP TABLE IF EXISTS team_members;
```

---

### 000008 -- create_team_join_requests

**UP** `migrations/000008_create_team_join_requests.up.sql`

```sql
CREATE TABLE team_join_requests (
    id              UUID                PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID                NOT NULL,
    user_id         UUID                NOT NULL,
    message         TEXT,
    status          team_member_status  NOT NULL DEFAULT 'pending',
    reviewed_by     UUID,
    role_assigned   team_member_role,
    created_at      TIMESTAMPTZ         NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ         NOT NULL DEFAULT now(),

    CONSTRAINT uq_join_requests_team_user UNIQUE (team_id, user_id),

    CONSTRAINT fk_join_requests_team
        FOREIGN KEY (team_id)
        REFERENCES teams (id)
        ON DELETE CASCADE,

    CONSTRAINT fk_join_requests_user
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE,

    CONSTRAINT fk_join_requests_reviewer
        FOREIGN KEY (reviewed_by)
        REFERENCES users (id)
        ON DELETE SET NULL
);

CREATE INDEX idx_join_requests_team_id ON team_join_requests (team_id);
CREATE INDEX idx_join_requests_user_id ON team_join_requests (user_id);
CREATE INDEX idx_join_requests_status ON team_join_requests (status);
```

**DOWN** `migrations/000008_create_team_join_requests.down.sql`

```sql
DROP TABLE IF EXISTS team_join_requests;
```

---

### 000009 -- create_team_repositories

**UP** `migrations/000009_create_team_repositories.up.sql`

```sql
CREATE TABLE team_repositories (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id       UUID        NOT NULL,
    repository_id UUID        NOT NULL,
    added_by      UUID,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_team_repos_team_repo UNIQUE (team_id, repository_id),

    CONSTRAINT fk_team_repos_team
        FOREIGN KEY (team_id)
        REFERENCES teams (id)
        ON DELETE CASCADE,

    CONSTRAINT fk_team_repos_repository
        FOREIGN KEY (repository_id)
        REFERENCES repositories (id)
        ON DELETE CASCADE,

    CONSTRAINT fk_team_repos_added_by
        FOREIGN KEY (added_by)
        REFERENCES users (id)
        ON DELETE SET NULL
);
```

**DOWN** `migrations/000009_create_team_repositories.down.sql`

```sql
DROP TABLE IF EXISTS team_repositories;
```

---

### 000010 -- create_test_suites

**UP** `migrations/000010_create_test_suites.up.sql`

```sql
CREATE TABLE test_suites (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id   UUID         NOT NULL,
    name            VARCHAR(255) NOT NULL,
    type            VARCHAR(50)  NOT NULL DEFAULT 'unit',
    config_path     TEXT,
    timeout_seconds INTEGER      NOT NULL DEFAULT 300,
    execution_mode  VARCHAR(20)  NOT NULL DEFAULT 'fork_gha',
    docker_image    VARCHAR(255),
    test_command    TEXT,
    gha_workflow_id VARCHAR(255),
    env_vars        JSONB        NOT NULL DEFAULT '{}',
    workflow_config JSONB        NOT NULL DEFAULT '{}',
    workflow_yaml   TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT fk_test_suites_repository
        FOREIGN KEY (repository_id)
        REFERENCES repositories (id)
        ON DELETE CASCADE
);

CREATE INDEX idx_test_suites_repository_id ON test_suites (repository_id);
```

**DOWN** `migrations/000010_create_test_suites.down.sql`

```sql
DROP TABLE IF EXISTS test_suites;
```

---

### 000011 -- create_test_runs

**UP** `migrations/000011_create_test_runs.up.sql`

```sql
CREATE TABLE test_runs (
    id            UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    test_suite_id UUID            NOT NULL,
    triggered_by  UUID,
    run_number    INTEGER         NOT NULL DEFAULT 1,
    branch        VARCHAR(255)    NOT NULL,
    commit_hash   CHAR(40)        NOT NULL,
    status        test_run_status NOT NULL DEFAULT 'queued',
    gha_run_id    BIGINT,
    log_output    TEXT,
    summary       JSONB,
    report_id     VARCHAR(255),
    started_at    TIMESTAMPTZ,
    finished_at   TIMESTAMPTZ,
    created_at    TIMESTAMPTZ     NOT NULL DEFAULT now(),

    CONSTRAINT fk_test_runs_suite
        FOREIGN KEY (test_suite_id)
        REFERENCES test_suites (id)
        ON DELETE CASCADE,

    CONSTRAINT fk_test_runs_triggered_by
        FOREIGN KEY (triggered_by)
        REFERENCES users (id)
        ON DELETE SET NULL
);

CREATE INDEX idx_test_runs_suite_id            ON test_runs (test_suite_id);
CREATE INDEX idx_test_runs_suite_status        ON test_runs (test_suite_id, status);
CREATE INDEX idx_test_runs_triggered_by        ON test_runs (triggered_by);
CREATE INDEX idx_test_runs_suite_branch_commit ON test_runs (test_suite_id, branch, commit_hash);
```

**DOWN** `migrations/000011_create_test_runs.down.sql`

```sql
DROP TABLE IF EXISTS test_runs;
```

---

### 000012 -- create_test_results

**UP** `migrations/000012_create_test_results.up.sql`

```sql
CREATE TABLE test_results (
    id            UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    test_run_id   UUID               NOT NULL,
    test_name     VARCHAR(512)       NOT NULL,
    status        test_result_status NOT NULL,
    duration_ms   INTEGER,
    error_message TEXT,
    log_output    TEXT,
    created_at    TIMESTAMPTZ        NOT NULL DEFAULT now(),

    CONSTRAINT fk_test_results_run
        FOREIGN KEY (test_run_id)
        REFERENCES test_runs (id)
        ON DELETE CASCADE
);

CREATE INDEX idx_test_results_run_id ON test_results (test_run_id);
```

**DOWN** `migrations/000012_create_test_results.down.sql`

```sql
DROP TABLE IF EXISTS test_results;
```

---

### 000013 -- create_test_groups

**UP** `migrations/000013_create_test_groups.up.sql`

```sql
CREATE TABLE test_groups (
    id            UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    test_run_id   UUID               NOT NULL,
    group_id      VARCHAR(255),
    name          VARCHAR(512),
    package       VARCHAR(1024),
    status        test_result_status DEFAULT 'unknown',
    total         INTEGER,
    passed        INTEGER,
    failed        INTEGER,
    skipped       INTEGER,
    duration_ms   INTEGER,
    pass_rate     NUMERIC(5,2),
    sort_order    INTEGER,
    created_at    TIMESTAMPTZ        NOT NULL DEFAULT now(),

    CONSTRAINT fk_test_groups_run
        FOREIGN KEY (test_run_id)
        REFERENCES test_runs (id)
        ON DELETE CASCADE
);

CREATE INDEX idx_test_groups_run_id ON test_groups (test_run_id);
```

**DOWN** `migrations/000013_create_test_groups.down.sql`

```sql
DROP TABLE IF EXISTS test_groups;
```

---

### 000014 -- create_test_cases

**UP** `migrations/000014_create_test_cases.up.sql`

```sql
CREATE TABLE test_cases (
    id             UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    test_group_id  UUID               NOT NULL,
    test_run_id    UUID               NOT NULL,
    case_id        VARCHAR(512),
    name           VARCHAR(512),
    status         test_result_status,
    duration_ms    INTEGER,
    error_message  TEXT,
    stack_trace    TEXT,
    retry_count    INTEGER            DEFAULT 0,
    logs_url       TEXT,
    created_at     TIMESTAMPTZ        NOT NULL DEFAULT now(),

    CONSTRAINT fk_test_cases_group
        FOREIGN KEY (test_group_id)
        REFERENCES test_groups (id)
        ON DELETE CASCADE,

    CONSTRAINT fk_test_cases_run
        FOREIGN KEY (test_run_id)
        REFERENCES test_runs (id)
        ON DELETE CASCADE
);

CREATE INDEX idx_test_cases_group_id ON test_cases (test_group_id);
CREATE INDEX idx_test_cases_run_id   ON test_cases (test_run_id);
```

**DOWN** `migrations/000014_create_test_cases.down.sql`

```sql
DROP TABLE IF EXISTS test_cases;
```

---

### 000015 -- create_ban_reviews

**UP** `migrations/000015_create_ban_reviews.up.sql`

```sql
CREATE TABLE ban_reviews (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID         NOT NULL,
    ban_reason    TEXT         NOT NULL,
    clarification TEXT         NOT NULL,
    status        VARCHAR(20)  DEFAULT 'pending',
    reviewed_by   UUID,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    reviewed_at   TIMESTAMPTZ,

    CONSTRAINT fk_ban_reviews_user
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE,

    CONSTRAINT fk_ban_reviews_reviewer
        FOREIGN KEY (reviewed_by)
        REFERENCES users (id)
        ON DELETE SET NULL
);

CREATE INDEX idx_ban_reviews_user_id ON ban_reviews (user_id);
CREATE INDEX idx_ban_reviews_status  ON ban_reviews (status);
```

**DOWN** `migrations/000015_create_ban_reviews.down.sql`

```sql
DROP TABLE IF EXISTS ban_reviews;
```

---

### 000016 -- create_notifications

**UP** `migrations/000016_create_notifications.up.sql`

```sql
CREATE TABLE notifications (
    id             UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID              NOT NULL,
    type           notification_type,
    subject        VARCHAR(255),
    body           TEXT              DEFAULT '',
    is_read        BOOLEAN           DEFAULT false,
    action_type    VARCHAR(64),
    action_payload JSONB,
    sender_id      UUID,
    created_at     TIMESTAMPTZ       NOT NULL DEFAULT now(),

    CONSTRAINT fk_notifications_user
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE,

    CONSTRAINT fk_notifications_sender
        FOREIGN KEY (sender_id)
        REFERENCES users (id)
        ON DELETE SET NULL
);

CREATE INDEX idx_notifications_user_id ON notifications (user_id);
CREATE INDEX idx_notifications_user_read ON notifications (user_id, is_read);
```

**DOWN** `migrations/000016_create_notifications.down.sql`

```sql
DROP TABLE IF EXISTS notifications;
```

---

## 6. Seed Data

### Root user bootstrap

The `root` user is **not** seeded via SQL. It is bootstrapped at application
startup from environment variables:

| Env var          | Purpose                        |
| ---------------- | ------------------------------ |
| `ROOT_EMAIL`     | Email for the root account     |
| `ROOT_PASSWORD`  | Password for the root account  |

On first launch the server checks whether a user with `role = 'root'` exists.
If not, it creates one using the values above (bcrypt-hashed password,
`role = 'root'`). No hardcoded seed credentials are stored in the repository.

### Makefile targets

```makefile
# --- Database variables -------------------------------------------------------
MIGRATE       := migrate
DB_DSN        ?= postgres://verdox:verdox@localhost:5432/verdox?sslmode=disable
MIGRATIONS    := migrations

# --- Migrations ---------------------------------------------------------------

.PHONY: migrate-up migrate-down migrate-create

## Run all pending UP migrations
migrate-up:
	$(MIGRATE) -path $(MIGRATIONS) -database "$(DB_DSN)" up

## Roll back the last migration
migrate-down:
	$(MIGRATE) -path $(MIGRATIONS) -database "$(DB_DSN)" down 1

## Create a new migration pair (usage: make migrate-create NAME=add_foo)
migrate-create:
	$(MIGRATE) create -ext sql -dir $(MIGRATIONS) -seq $(NAME)
```

---

## 7. Query Patterns

These are the most common queries the application will issue. Each entry
notes which index it relies on.

### Get user by email (login)

```sql
-- Uses: uq_users_email (unique index on email)
SELECT id, username, email, password_hash, role, avatar_url, created_at, updated_at
  FROM users
 WHERE email = $1;
```

### Get user by ID (auth middleware)

```sql
-- Uses: PK index on users.id
SELECT id, username, email, role, avatar_url, created_at, updated_at
  FROM users
 WHERE id = $1;
```

### List team members with status filter (team detail page)

```sql
-- Uses: uq_team_members_team_user (leading column team_id)
SELECT tm.id,
       tm.user_id,
       u.username,
       u.avatar_url,
       tm.role,
       tm.status,
       tm.created_at
  FROM team_members tm
  JOIN users u ON u.id = tm.user_id
 WHERE tm.team_id = $1
   AND tm.status  = $2
 ORDER BY tm.created_at;
```

### List test runs by suite (run history)

```sql
-- Uses: idx_test_runs_suite_id
SELECT id, branch, commit_hash, status, started_at, finished_at, created_at
  FROM test_runs
 WHERE test_suite_id = $1
 ORDER BY created_at DESC
 LIMIT $2 OFFSET $3;
```

### Get test results for a run (run detail)

```sql
-- Uses: idx_test_results_run_id
SELECT id, test_name, status, duration_ms, error_message, log_output, created_at
  FROM test_results
 WHERE test_run_id = $1
 ORDER BY test_name;
```

### Check team membership (permission check)

```sql
-- Uses: uq_team_members_team_user (exact composite match)
SELECT id, role, status
  FROM team_members
 WHERE team_id = $1
   AND user_id = $2;
```

### Purge expired sessions (scheduled cleanup)

```sql
-- Uses: idx_sessions_expires_at
DELETE FROM sessions
 WHERE expires_at < now();
```

### Count test runs by status for a suite (dashboard widget)

```sql
-- Uses: idx_test_runs_suite_status
SELECT status, count(*) AS cnt
  FROM test_runs
 WHERE test_suite_id = $1
 GROUP BY status;
```

### Check for existing test run by suite+branch+commit (cache lookup)

```sql
-- Uses: idx_test_runs_suite_branch_commit
SELECT id, run_number, status, started_at, finished_at, created_at
  FROM test_runs
 WHERE test_suite_id = $1
   AND branch        = $2
   AND commit_hash   = $3
 ORDER BY run_number DESC
 LIMIT 1;
```

### List pending join requests for a team

```sql
-- Uses: idx_join_requests_team_id, idx_join_requests_status
SELECT jr.id,
       jr.user_id,
       u.username,
       u.avatar_url,
       jr.message,
       jr.status,
       jr.created_at
  FROM team_join_requests jr
  JOIN users u ON u.id = jr.user_id
 WHERE jr.team_id = $1
   AND jr.status  = 'pending'
 ORDER BY jr.created_at;
```

### Get team's GitHub PAT (for cloning / API calls)

```sql
-- Uses: PK index on teams.id
SELECT github_pat_encrypted,
       github_pat_nonce,
       github_pat_set_at,
       github_pat_set_by,
       github_pat_github_username
  FROM teams
 WHERE id = $1;
```

### Resolve PAT for a repository (repo -> team -> PAT)

```sql
-- Uses: uq_team_repos_team_repo, PK on teams.id
SELECT t.github_pat_encrypted,
       t.github_pat_nonce,
       t.github_pat_github_username
  FROM team_repositories tr
  JOIN teams t ON t.id = tr.team_id
 WHERE tr.repository_id = $1
 LIMIT 1;
```
