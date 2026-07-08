-- Payment-id uniqueness backstop (#395·1). DB-level guard against ONE gateway
-- payment being stamped on two orders (the app-logic binding alone can't stop it).
--
-- PARTIAL (WHERE col <> '') because wallet-only / unpaid / Stripe / meal-plan-day
-- shell orders legitimately share the empty default — only real gateway ids must be
-- unique. Group participants (group_order_participants) and catering (catering_requests)
-- live in other tables and are unaffected.
--
-- RUNTIME MECHANISM: like the other payout migrations, these indexes are created at
-- boot by the service's postMigrate DDL block (database/database.go), applied
-- NON-FATALLY there (a pre-existing DUPLICATE payment id logs an ALERT rather than
-- crash-looping the API — dedup then redeploy). This timestamped pair is the auditable
-- production DDL matching convention.

CREATE UNIQUE INDEX IF NOT EXISTS idx_orders_razorpay_order_id ON orders (razorpay_order_id) WHERE razorpay_order_id <> '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_orders_razorpay_payment_id ON orders (razorpay_payment_id) WHERE razorpay_payment_id <> '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_orders_stripe_payment_intent_id ON orders (stripe_payment_intent_id) WHERE stripe_payment_intent_id <> '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_meal_plans_razorpay_order_id ON meal_plans (razorpay_order_id) WHERE razorpay_order_id <> '';
