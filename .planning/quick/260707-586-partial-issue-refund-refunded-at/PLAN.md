# #586 — partial issue refund must not forfeit / strand the chef payout

## Root cause (grounded in code, not the issue's framing)

`services.RefundIssueToWallet` (services/order_issue.go) is the ONE refund path #549 never
covered. It unconditionally:
- stamps `order.refunded_at` (order_issue.go:189), even for a **partial** issue refund, and
- runs the FULL hold cross-guard `WithholdOrReverseOrderHoldForRefund` (order_issue.go:208).

Every release-side guard keys "block the WHOLE chef payout" on `refunded_at IS NOT NULL`
(payout_hold.go:166 `ReleaseDisputedOrderHoldIfCleared`; payout_release.go:168 queue,
:394 `orderRefundBlocks`, :456 `refundedOrderSubquery`; payout_reconcile_cron.go:267
`reconcileCancelledOrders`). So a partial issue refund:
1. forfeits the chef's ENTIRE remaining payout (same class as #549, unfixed for this path);
2. makes the delivery-failure **customer-fault** resolver's `ReleaseDisputedOrderHoldIfCleared`
   match 0 rows → **silent strand** (admin gets 200 "resolved", chef never released);
3. worse: `reconcileCancelledOrders` matches `refunded_at IS NOT NULL` + disputed → actively
   drives the hold to **withheld**, contradicting the "pay the chef" ruling.

## Fix — remove the conflation at the SOURCE (mirror #549 exactly)

In `RefundIssueToWallet`, compute `fullRefund := credit >= remaining` under the same locked
read that caps the credit, then:
- stamp `refunded_at` ONLY when `fullRefund`;
- post-tx cross-guard: `fullRefund` → `WithholdOrReverseOrderHoldForRefund` (whole hold, as
  today); else → `WithholdOrReverseOrderHoldForPartialRefund(db, orderID, ToPaise(credit), …)`
  — claw only the refunded portion from the chef transfer, LEAVE the hold releasable.

This makes `refunded_at` stamped by FULL refunds ONLY across ALL paths, so every existing
`refunded_at IS NOT NULL` guard becomes exactly correct with **ZERO guard changes** and **ZERO
new over-pay-via-relaxed-guard risk**. #586's strand + misdirection both dissolve.

## Deliberately NOT doing

The heavier approach prior sessions sketched (relax ~6 guards to `RemainingRefundable<=0` +
a `partial_claw_settled_at` marker column) is unnecessary and riskier. Fixing the source is
smaller, safer, and consistent with the #549 precedent (chef_order_cancel `RefundOrder` +
`crossGuardRefundHold`).

## Known residual (pre-existing, flag-gated, out of scope — note on issue)

A partial issue refund at the `none` hold (prep-time, before any transfer exists) can't claw
(no transfer yet); the chef transfer is later sized on subtotal by `ComputeOrderEarnings`
regardless of issue refunds, so at flags-ON release the chef is paid full-net, over by the
refunded portion. This is a **systemic transfer-SIZING gap** affecting every partial-refunded
order identically (not #586-specific) and is flag-gated OFF. Belongs with the transfer-sizing /
#602 refund-drift reconcile work. Before the fix the chef got NOTHING (clear under-pay + wrong
direction); after, the chef is paid in the correct direction (small over by refunded portion).

## Money-safety invariants preserved

- All new movement flag-gated (partial claw is a no-op while `ORDER_PAYOUT_AUTO_RELEASE_ENABLED`
  OFF). No release guard relaxed.
- Double-refund protection untouched: RefundIssueToWallet's cap = `RemainingRefundable` under the
  order-row lock + the issue-claim; `refunded_at` is NOT a claim key here. A later full refund of
  the remainder now correctly proceeds (was wrongly blocked by the partial's refunded_at), still
  capped at remaining.
- Partial claw runs exactly once (issue-claim → a retry loses the claim → never re-reaches the
  cross-guard → no double-claw, #568).

## Tests (RED-first)

1. order_issue_test.go — partial `RefundIssueToWallet` leaves `refunded_at` NULL; full stamps it.
2. payout_crossguard_refund_test.go — REWRITE `TestCrossguard_AutoRefundDrivesHold` (currently a
   PARTIAL 120-of-250 asserting whole-hold withheld — encodes the bug): partial → hold stays
   release_eligible + still releasable; full → withheld + blocked.
3. delivery_failure_resolve_test.go — #586 end-to-end: partial issue refund, then delivery-failure
   customer-fault resolve → disputed hold RELEASES (was stranded).
4. delivery_failure_reconcile / cron — `reconcileCancelledOrders` does NOT withhold a
   partially-refunded (refunded_at NULL) order.

Then: `go test ./...`, independent adversarial go-developer verify against the diff.
