CREATE TYPE user_role AS ENUM ('root', 'admin', 'moderator', 'user');
CREATE TYPE team_member_role AS ENUM ('admin', 'maintainer', 'viewer');
CREATE TYPE team_member_status AS ENUM ('pending', 'approved', 'rejected');
CREATE TYPE test_run_status AS ENUM ('queued', 'running', 'passed', 'failed', 'cancelled');
CREATE TYPE test_result_status AS ENUM ('pass', 'fail', 'skip', 'error', 'running', 'unknown');
CREATE TYPE notification_type AS ENUM ('system', 'admin_message', 'ban_review', 'test_complete', 'team_invite', 'team_join_request');
