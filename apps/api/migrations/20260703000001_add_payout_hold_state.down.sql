-- Rollback the payout hold state additions (#387).
ALTER TABLE orders
    DROP COLUMN IF EXISTS customer_confirmed_at,
    DROP COLUMN IF EXISTS payout_hold_status;

ALTER TABLE meal_plan_days
    DROP COLUMN IF EXISTS customer_confirmed_at,
    DROP COLUMN IF EXISTS payout_hold_status;
