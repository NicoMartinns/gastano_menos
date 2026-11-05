-- Existing accounts retain access. No default administrator or credentials.
ALTER TABLE users
    ADD COLUMN role VARCHAR(10) NOT NULL DEFAULT 'USER',
    ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN session_version INTEGER NOT NULL DEFAULT 0,
    ADD CONSTRAINT chk_users_role CHECK (role IN ('USER', 'ADMIN')),
    ADD CONSTRAINT chk_users_session_version CHECK (session_version >= 0);
