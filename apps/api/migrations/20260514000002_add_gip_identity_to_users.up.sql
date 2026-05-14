-- Add Google Identity Platform (GIP) identity columns to users.
--
-- These columns are added as NULLABLE because existing rows from before the
-- fresh-start cutover have no GIP identity to backfill. The application
-- (apps/auth-bff's /internal/users/upsert endpoint, Task 2.2) only ever
-- INSERTs with these fields populated, so new rows will always carry GIP
-- identity. Old orphaned rows can be hard-deleted in a follow-on cleanup
-- after the cutover soak period.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS gip_uid          VARCHAR(128),
    ADD COLUMN IF NOT EXISTS gip_tenant_id    VARCHAR(64),
    ADD COLUMN IF NOT EXISTS gip_provider     VARCHAR(32),
    ADD COLUMN IF NOT EXISTS auth_pool        VARCHAR(16),
    ADD COLUMN IF NOT EXISTS role             VARCHAR(16),
    ADD COLUMN IF NOT EXISTS last_login_at    TIMESTAMPTZ;

-- Unique index on gip_uid: each GIP user maps to at most one row.
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_gip_uid
    ON users (gip_uid)
    WHERE gip_uid IS NOT NULL;

-- Composite index for lookups by (email, auth_pool).
CREATE INDEX IF NOT EXISTS idx_users_email_pool
    ON users (email, auth_pool);

-- Per-pool email uniqueness (case-insensitive). The same email is allowed
-- in different pools (e.g., a chef who also orders as a customer).
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_per_pool
    ON users (lower(email), auth_pool)
    WHERE email IS NOT NULL AND auth_pool IS NOT NULL;

-- Drop the legacy global email unique constraint if it exists. Names vary
-- by how the original table was created (GORM AutoMigrate vs. explicit
-- migration); try the most likely candidates.
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;
DROP INDEX IF EXISTS users_email_key;
DROP INDEX IF EXISTS idx_users_email;
