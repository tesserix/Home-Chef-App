-- Rollback the group_orders payout hold additions (#456).
ALTER TABLE group_orders
    DROP COLUMN IF EXISTS payout_settle_attempts,
    DROP COLUMN IF EXISTS payout_settled_at,
    DROP COLUMN IF EXISTS delivered_at,
    DROP COLUMN IF EXISTS customer_confirmed_at,
    DROP COLUMN IF EXISTS payout_hold_status;
