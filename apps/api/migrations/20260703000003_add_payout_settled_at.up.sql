-- Add payout settlement tracking to orders and meal_plan_days (#459).
--
-- The payout hold state machine (#387/#388) flips payout_hold_status to
-- 'released'/'reversed' in a committed tx and only THEN runs the money seam
-- (releaseMoney/reverseMoney) post-commit. If that seam fails, the row is left
-- released/reversed with money UNMOVED and nothing re-drives it. These columns
-- decouple "status committed" from "money confirmed moved":
--
-- - payout_settled_at: stamped (server-side, now) ONLY AFTER the money seam
--   returns nil. A successful path is released + settled_at set; a drift row is
--   released + settled_at NULL — the exact predicate the payout-reconcile cron
--   selects to re-drive the seam.
-- - payout_settle_attempts: bounds those re-drives. On a repeated seam failure the
--   reconcile increments this and, at the cap, ALERT-logs the permanently-bad
--   transfer so ops can act (a DB column, not an in-memory counter, so the cap
--   survives the per-activity Temporal cron process model).
--
-- RUNTIME MECHANISM: like 20260703000001, these columns are actually created at
-- boot by DB.AutoMigrate from the new struct tags in models/order.go +
-- models/meal_plan.go (also what creates them in the sqlite unit-test harness).
-- This timestamped SQL pair is the auditable production DDL matching convention.

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS payout_settled_at      TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS payout_settle_attempts INTEGER NOT NULL DEFAULT 0;

ALTER TABLE meal_plan_days
    ADD COLUMN IF NOT EXISTS payout_settled_at      TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS payout_settle_attempts INTEGER NOT NULL DEFAULT 0;
