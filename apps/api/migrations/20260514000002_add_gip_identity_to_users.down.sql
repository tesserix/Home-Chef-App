-- Rollback the GIP identity additions. Used in coordination with
-- 20260514000001's down migration to fully revert to the pre-GIP schema.

DROP INDEX IF EXISTS idx_users_email_per_pool;
DROP INDEX IF EXISTS idx_users_email_pool;
DROP INDEX IF EXISTS idx_users_gip_uid;

ALTER TABLE users
    DROP COLUMN IF EXISTS gip_uid,
    DROP COLUMN IF EXISTS gip_tenant_id,
    DROP COLUMN IF EXISTS gip_provider,
    DROP COLUMN IF EXISTS auth_pool,
    DROP COLUMN IF EXISTS role,
    DROP COLUMN IF EXISTS last_login_at;

-- Restore the global email unique constraint (best-effort — GORM AutoMigrate
-- creates it as users_email_key by default).
CREATE UNIQUE INDEX IF NOT EXISTS users_email_key ON users (email);
