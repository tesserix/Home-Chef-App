-- Rollback the frozen per-order commission rate (#390).
ALTER TABLE orders
    DROP COLUMN IF EXISTS commission_rate;
