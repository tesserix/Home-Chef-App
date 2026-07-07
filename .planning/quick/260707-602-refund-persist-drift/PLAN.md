# #602 — gateway/credit succeeded but persist failed → ledger integrity

## Root cause
`services.ReserveRefund` commits `refund_amount += reserved` + `payment_status→refunded` in its
OWN tx BEFORE moving money. So when the money leg (gateway `CreateRefund` / `CreditWallet`)
SUCCEEDS but the later persist tx FAILS, the true state is "refund happened, `refund_amount`
is already CORRECT, only the terminal write (refund_id / refunded_at / revert-to-completed /
event) didn't land." Two refund paths then RELEASE (decrement `refund_amount`) on that persist
failure — which **erases a refund that actually happened**:
- `InitiateRefund` to-wallet PARTIAL (`payment.go:1013`, `if !fullRefund { releaseReservation() }`)
- chef goodwill `RefundOrder` (`chef_order_cancel.go:560`, unconditional `revertReservation()`)

Because the idempotency keys are a pure function of `(orderID, priorRefunded)` — and the release
reverts `priorRefunded` — the next DISTINCT refund collides the key: razorpay rejects (stuck),
wallet's `applyWalletTxn` (amount-blind, wallet.go:67-72) silently returns the earlier txn →
lost refund + over-claw. (Discovered abandoning #615's naive gateway-branch release.)

The decrement is REQUIRED for a same-amount retry to dedup, yet the decrement is what corrupts
the ledger → unresolvable by releasing. The gateway `CreateRefund` id / wallet txn is the proof
money moved; `refund_amount` already reflects it. So: DON'T release; leave stuck (ledger
correct); a reconcile finalizes the deferred terminal write.

## Slice 1 (money-safety — this PR): stop releasing on PERSIST failure
- `payment.go` to-wallet: drop the partial `releaseReservation()` on persist failure. Leave the
  order stuck (`payment_status=refunded`, `refund_amount` correct) exactly like the gateway
  branch already does. FULL already stays reserved (unchanged).
- `chef_order_cancel.go` RefundOrder: drop the persist-failure `revertReservation()`. Leave stuck.
  Keep the GATEWAY-error `revertReservation()` (money did NOT move → correct to undo).
- The per-line `CancelOrderItem` revert stays (keyed per-LINE, `RefundLineIdempotencyKey`, no
  prior-refunded collision → not part of this bug).
- Stuck-but-ledger-correct is MONEY-SAFE (the reserve durably recorded the refund; a stuck
  `payment_status=refunded` blocks all further refunds so nothing can over-refund). Brings both
  paths to parity with the gateway branch's long-standing safe-stuck behavior.
- Update `TestInitiateRefund_PartialToWallet_PersistFailure_RevertsClaim` → assert stuck +
  `refund_amount` correct (was asserting the buggy revert-to-completed). Add a RefundOrder
  persist-failure test asserting stuck-but-correct.

## Slice 2 (heal — follow-up PR): reconcileStuckRefunds
A stuck mid-refund is UNIQUELY `payment_status='refunded' AND refunded_at IS NULL` (a legit FULL
stamps `refunded_at`; a legit PARTIAL reverts to `completed`). With a staleness guard
(`updated_at < now-grace`, so an in-flight refund's brief reserve→persist window isn't touched),
finalize each: recompute `RemainingRefundable` — `<=0` ⇒ FULL (stamp `refunded_at` + status),
else PARTIAL (revert `payment_status→completed`) + run the partial claw (`crossGuardRefundHold`,
flag-gated). Money-safe: only finalizes what the reserve already committed; the frozen stuck row
can't be mutated by other refunds. Refund_id best-effort (marker or gateway fetch).

## Slice 3 (defense-in-depth — separate PR, audit callers first)
Harden `applyWalletTxn` to compare `existing.Amount` on an idempotency-key match and return a
hard error on mismatch (loud, safe) instead of silently under-crediting. Moot for the refund
paths once slice 1 removes the priorRefunded corruption, but good hygiene for every keyed wallet
credit.

## Rigor
Live customer money. plan → RED-first → adversarial go-developer verify per slice.
