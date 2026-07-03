-- Add payout hold state to orders and meal_plan_days (#387).
--
-- The payout hold state machine decouples "delivered" from "chef paid": on
-- delivery the hold becomes 'awaiting_customer_confirmation' (no money moves),
-- and only an explicit customer confirmation advances it to 'release_eligible'
-- (consumed later by the admin payout queue, #388). An open OrderIssue forces
-- 'disputed' at confirm time.
--
-- RUNTIME MECHANISM: these columns are actually created at boot by
-- DB.AutoMigrate (database/database.go:170 orders, :219 meal_plan_days) from the
-- new struct tags in models/order.go + models/meal_plan.go (this is also what
-- creates them in the sqlite unit-test harness). This timestamped SQL pair is
-- the auditable production DDL matching repo convention.
--
-- - payout_hold_status: NOT NULL, defaults to '' so legacy rows are 'none'.
-- - customer_confirmed_at: nullable; stamped server-side when the customer
--   confirms receipt (the audit record of who released the hold).

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS payout_hold_status    VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS customer_confirmed_at TIMESTAMPTZ;

ALTER TABLE meal_plan_days
    ADD COLUMN IF NOT EXISTS payout_hold_status    VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS customer_confirmed_at TIMESTAMPTZ;
