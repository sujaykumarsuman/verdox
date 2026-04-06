CREATE TYPE user_role AS ENUM ('root', 'moderator', 'user');
CREATE TYPE team_member_role AS ENUM ('admin', 'maintainer', 'viewer');
CREATE TYPE team_member_status AS ENUM ('pending', 'approved', 'rejected');
CREATE TYPE test_type AS ENUM ('unit', 'integration');
CREATE TYPE test_run_status AS ENUM ('queued', 'running', 'passed', 'failed', 'cancelled');
CREATE TYPE test_result_status AS ENUM ('pass', 'fail', 'skip', 'error');
