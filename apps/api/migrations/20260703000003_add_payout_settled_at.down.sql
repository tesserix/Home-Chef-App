-- Rollback the payout settlement tracking additions (#459).
ALTER TABLE orders
    DROP COLUMN IF EXISTS payout_settle_attempts,
    DROP COLUMN IF EXISTS payout_settled_at;

ALTER TABLE meal_plan_days
    DROP COLUMN IF EXISTS payout_settle_attempts,
    DROP COLUMN IF EXISTS payout_settled_at;
