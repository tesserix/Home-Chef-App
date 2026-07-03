-- Add payout hold state to group_orders (#456).
--
-- The group/office order is promoted to a first-class payout-hold aggregate
-- alongside orders and meal_plan_days (#387/#388/#459). Previously the group chef
-- payout was released immediately on delivery with NO escrow-flag gate, bypassing
-- the entire hold machine (no customer confirm, no dispute check, no admin release
-- queue, no reconcile). These columns let the group hold obey the same conditional-
-- transition invariants:
--
-- - payout_hold_status: parked at 'awaiting_customer_confirmation' on delivery,
--   advanced to 'release_eligible' on host confirm, then driven to
--   'released'/'reversed'/'withheld' by the admin payout queue / sweep / reconcile.
-- - customer_confirmed_at: stamped when the host confirms receipt.
-- - delivered_at: stamped on delivery (the SLA/auto-confirm clock).
-- - payout_settled_at / payout_settle_attempts: decouple status-committed from
--   money-confirmed-moved (the payout-reconcile drift predicate + attempt cap).
--
-- RUNTIME MECHANISM: like 20260703000001/000003, these columns are actually created
-- at boot by DB.AutoMigrate from the new struct tags in models/group_order.go (also
-- what creates them in the sqlite unit-test harness). This timestamped SQL pair is
-- the auditable production DDL matching convention.

ALTER TABLE group_orders
    ADD COLUMN IF NOT EXISTS payout_hold_status     VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS customer_confirmed_at  TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS delivered_at           TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS payout_settled_at      TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS payout_settle_attempts INTEGER NOT NULL DEFAULT 0;
