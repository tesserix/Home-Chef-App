-- Add DPDP marketing-consent columns to users.
--
-- CW-01b: the RegisterPage on apps/web exposes an optional "Send me occasional
-- updates" checkbox. This migration adds the storage so /internal/users/upsert
-- (apps/auth-bff -> apps/api) can persist that consent on first sign-in, and
-- so we can produce an auditable timestamp for DPDP §6 disclosures.
--
-- - marketing_consent: boolean flag, defaults to FALSE so legacy/back-filled
--   rows are treated as "no consent" (safe default per DPDP).
-- - marketing_consent_at: nullable timestamp captured the moment consent was
--   granted. Null means the user has never opted in.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS marketing_consent    BOOLEAN     NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS marketing_consent_at TIMESTAMPTZ;
