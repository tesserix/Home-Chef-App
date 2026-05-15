-- Rollback the marketing-consent additions (CW-01b).
ALTER TABLE users
    DROP COLUMN IF EXISTS marketing_consent_at,
    DROP COLUMN IF EXISTS marketing_consent;
