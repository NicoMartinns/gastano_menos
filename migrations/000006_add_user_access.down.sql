ALTER TABLE users
    DROP CONSTRAINT chk_users_session_version,
    DROP CONSTRAINT chk_users_role,
    DROP COLUMN session_version,
    DROP COLUMN is_active,
    DROP COLUMN role;
