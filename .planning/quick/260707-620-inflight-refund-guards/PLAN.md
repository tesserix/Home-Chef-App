# #620 — payout release + per-line cancel respect an in-flight/stuck order-level refund

## Part A (release guard) — money-critical, flag-gated
A stuck/in-flight order-level refund sits at payment_status=refunded with refunded_at NULL
(ReserveRefund flips it before persist; #602 leaves it stuck until reconcileStuckRefunds).
The release guards blocked only on status IN(refunded,cancelled)/refunded_at/pending-issue, so a
ReleaseHold could pay the chef the un-clawed amount in that window. Fix: add payment_status=
'refunded' to BOTH guard sites in parity — orderRefundBlocks (Go backstop) + refundedOrderSubquery
(SQL NOT-EXISTS in releaseTransition's atomic UPDATE). A finalized PARTIAL reverts to completed
(releases); a finalized FULL keeps refunded (already blocked).

## Part B (CancelOrderItem order mutex) — live money
CancelOrderItem issued its own gateway refund + wrote refund_amount without checking the order's
payment_status → raced a concurrent ReserveRefund-family order-level refund → two real refunds
past Total. Fix: claimOrderItemForCancel locks the order row FIRST (order-first per #585) and
requires payment_status='completed', else errOrderRefundInProgress → 409.

## Verify outcome
Parts A+B correct/safe (lock-order, false-block, Go/SQL parity, correctness all refuted).
CONFIRMED adjacent gap: CancelOrderItem vs RefundIssueToWallet (issue-refund path — different
mechanism, doesn't flip payment_status) → real over-refund. Pre-existing, design-level (order-level
vs per-line refund reconciliation) → filed #622 (P1 live money), NOT fixed here.

## Deferred (Part C): CancelOrder persist-failure status-stuck (money-safe via reconcileCancelledOrders).
