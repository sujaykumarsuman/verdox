-- PostgreSQL does not support removing enum values directly.
-- Users with role 'admin' must be reassigned before rollback.
UPDATE users SET role = 'moderator' WHERE role = 'admin';
