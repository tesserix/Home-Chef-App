# #611 — InitiateRefund partial-aware reserve-under-lock

## Problem
`handlers/payment.go InitiateRefund` (and `handlers/chef_order_cancel.go RefundOrder`) read
`RemainingRefundable` + `order.RefundAmount` UNLOCKED, then `claimRefundForProcessing`
(payment_status='completed' only) in a separate statement, then persist
`refund_amount = order.RefundAmount + amount` (stale read-modify-write). A concurrent partial
(`RefundIssueToWallet` / #549 goodwill) committing between the read and the claim → over-refund
of live customer money + a lost-update clobbering the concurrent partial's increment. #609 fixed
the three FULL paths with `ReserveFullRefund` but that reserves the WHOLE remaining, so it does
not fit `InitiateRefund`'s arbitrary `req.Amount` (partial).

## Fix (mirror #609, partial-aware)
Add `services.ReserveRefund(db, orderID, requested)` → `(amount, priorRefunded, fullRefund, won, err)`:
under a FOR UPDATE row lock, compute `remaining = Round(Total − RefundAmount + PerLineRefundedTotalTx)`,
`amount = min(requested-or-full, remaining)`, and reserve it atomically (guarded UPDATE on
`payment_status='completed'`: flip → refunded [the concurrency mutex] AND
`refund_amount = COALESCE(refund_amount,0)+amount`). Does NOT stamp `refunded_at` — the caller owns
the terminal marker (only a FULL refund stamps it) and the partial revert-to-completed.
`fullRefund = amount >= freshRemaining` (a partial-intent request that exhausts a race-shrunk
remainder is correctly terminal). `priorRefunded` = locked read of refund_amount → the idempotency-key basis.
`ReleaseRefundReservation(db, orderID, amount)` reverts payment_status→completed + decrements refund_amount
on gateway/wallet failure.

### Handler reorder (the non-surgical part)
- Keep the pre-lock UX 400s (negative, remaining<=0, exceeds-remaining) as best-effort; reserve is authoritative.
- Reserve BEFORE the to-wallet/gateway split (where the old claim was).
- Derive `fullRefund` + the capacity-release closure + all idempotency keys from `reserved`/`priorRefunded` (all move AFTER the reserve).
- Drop the `refund_amount = order.RefundAmount + amount` persist writes (reserve already incremented).
- Every old `revertClaim()` → `ReleaseRefundReservation(reserved)` (undo both payment_status AND the increment).
  Gateway PERSIST failure stays no-revert (stuck-then-ops) exactly as today.
- Wallet-split gotcha (#609 memory): release-on-failure uses `reserved` (full), NOT the lowered gateway `refundAmount`.

## Behaviour-preserving
All existing to-wallet handler tests (`payment_partial_refund_test.go`) stay GREEN. Side-improvement:
the wallet-at-checkout split now records the FULL refund in refund_amount (was under-counting by the
wallet portion → latent next-refund over-refund) — strictly safer.

## Tests (RED-first, sqlite can't do true concurrency → deterministic)
- `services/refund_reserve_partial_test.go`: cap vs pre-seeded refund_amount; single-winner (2nd loses);
  partial leaves refunded_at NULL + reserves exact; requested<=0 = full; exceed→cap→full; release reverts both.
- Handler: partial gateway branch via razorpay httptest seam (delivered order → no ReleaseCapacity) —
  over-refund guarded when refund_amount pre-advanced.

## Verify
Adversarial go-developer (fresh) — live customer money, not escrow-flag-gated.
