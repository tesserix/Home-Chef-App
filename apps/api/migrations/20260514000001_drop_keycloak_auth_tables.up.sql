-- Drop Keycloak-era auth columns from users table.
-- The Go API no longer authenticates users — that responsibility moved to apps/auth-bff
-- backed by Google Identity Platform (GIP). Users are identified by gip_uid
-- (added in the next migration, 20260514000002).
--
-- Fresh-start cutover: existing user rows are intentionally orphaned. They will
-- not have gip_uid populated and won't be reachable by the new auth flow. Users
-- re-register at cutover.

ALTER TABLE users DROP COLUMN IF EXISTS password;
ALTER TABLE users DROP COLUMN IF EXISTS totp_enabled;
ALTER TABLE users DROP COLUMN IF EXISTS totp_verified_at;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified;
ALTER TABLE users DROP COLUMN IF EXISTS auth_provider;
ALTER TABLE users DROP COLUMN IF EXISTS provider_id;

DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS email_verification_tokens;
