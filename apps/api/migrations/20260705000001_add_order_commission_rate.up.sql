-- Freeze the platform commission rate on each order at checkout (#390).
--
-- Before #390 the chef's Route transfer paid gross while the settlement statement
-- computed net-of-commission on the LIVE commission rate. An admin retuning the
-- runtime rate (payout.commission_rate) between checkout and statement generation
-- would make the statement disagree with the transfer already sent. Freezing the
-- rate on the order row makes the split deterministic: verify, statement, TDS
-- certificate, and the earnings breakdown all read this per-order column and fall
-- back to the live rate/default only for legacy rows (commission_rate = 0).
--
-- RUNTIME MECHANISM: like the other payout migrations, this column is actually
-- created at boot by DB.AutoMigrate from the new struct tag in models/order.go
-- (also what creates it in the sqlite unit-test harness). This timestamped SQL
-- pair is the auditable production DDL matching convention.

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS commission_rate DOUBLE PRECISION NOT NULL DEFAULT 0;
