-- Rollback for #388 payout hold states ('withheld' / 'reversed').
--
-- Data-only: there is no schema to revert (payout_hold_status is a VARCHAR, the
-- new values need no column/type change, and no columns were added). Any rows
-- left in 'withheld'/'reversed' would need an operational data fix, not a schema
-- migration. The SELECT 1 keeps this a valid, idempotent no-op for the tooling.
SELECT 1;
