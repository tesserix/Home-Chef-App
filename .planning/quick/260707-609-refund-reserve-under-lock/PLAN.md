# #609 — full-refund paths must reserve under a lock (stop over-refund vs a concurrent partial)

## Root cause
Three full-refund paths read the refund amount and claim in TWO steps, unlocked:
- `services/cancellation_order_refund.go` `RefundOrderForCancellation`
- `handlers/chef_order_cancel.go` full `CancelOrder`
- `services/temporal_order.go` `CompensateOrderRefund`

They compute `RemainingRefundable`/`order.Total` from an UNLOCKED read, then a separate claim
`UPDATE ... WHERE payment_status='completed' AND refunded_at IS NULL`, then persist
`refund_amount = order.RefundAmount + amount` (stale read-modify-write). The PARTIAL path
(`RefundIssueToWallet`) already does it right: locks the order `FOR UPDATE`, caps at
`RemainingRefundable`, and increments `refund_amount` atomically. Since #549/#586 a partial
leaves `refunded_at` NULL, so a partial racing a full can:
1. **over-refund** — the full's amount was read before the partial committed (Finding 2, race);
2. **lost-update the ledger** — the full's `order.RefundAmount + amount` clobbers the partial's
   atomic increment (Finding 2, race);
3. **CompensateOrderRefund double-refunds deterministically** — it credits `order.Total` and
   OVERWRITES `refund_amount = order.Total`, ignoring any prior partial (Finding 3; testable).

## Fix — one shared reserve-under-lock helper (mirror the partial path's discipline)
`services/refund_reserve.go`:
- `ReserveFullRefund(db, orderID) (amount, won, err)` — in ONE tx under a row lock: read
  total/refund_amount, compute `remaining = Total − RefundAmount + perLine`, and if payable
  claim+reserve in a single guarded UPDATE (`payment_status→refunded`, stamp `refunded_at`,
  `refund_amount += remaining`). won=false (amount 0) when not completed / nothing left /
  sibling already claimed. A concurrent partial then observes the reserved `refund_amount` and
  caps to 0; a concurrent full loses the payment_status claim.
- `ReleaseFullRefundReservation(db, orderID, amount)` — undo on gateway failure (revert all
  three columns) so a retry can re-refund.

Rewire all three callers to `ReserveFullRefund` (amount) + gateway + `ReleaseFullRefundReservation`
on failure; drop the now-redundant post-gateway `refund_amount` write (reserved atomically).
Preserve every existing behavior: wallet-cap split + idempotency keys (RefundOrderForCancellation),
status ownership (caller sets cancelled/rejected/refunded), capacity release + cross-guard
(chef handler), ReverseOrderPayouts-first (saga).

## Testing (per the repo's concurrency rule — sqlite can't run true concurrent txns)
- Helper deterministic: reserves `remaining` given a pre-existing `refund_amount`; single-winner
  (2 back-to-back, 2nd → RowsAffected 0 → won=false); not-completed / nothing-left → won=false;
  `refunded_at`-set → won=false; Release reverts all three columns.
- `CompensateOrderRefund` prior-partial (DETERMINISTIC over-refund, RED against current): seed
  total=1000 refund_amount=400 → credits 600 (not 1000), refund_amount→1000 (increment).
- Existing `RefundOrderForCancellation` tests stay green (behavior-preserving on the non-race
  paths). Chef full-`CancelOrder` handler isn't sqlite-drivable (ReleaseCapacity = Postgres SQL)
  → covered by the helper tests + code review (same as #576/#601).

## Scope / risk
Pre-existing (since #549), P2, live customer money (not escrow-flag-gated). Rewires 3 critical
refund paths — behavior-preserving for the non-concurrent cases (existing tests protect it), the
reserve-under-lock closes the race by construction. Independent adversarial verify before merge.
