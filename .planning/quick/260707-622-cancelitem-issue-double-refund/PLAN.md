# #622 — CancelOrderItem × RefundIssueToWallet double-refund (both automatic directions)

Root: per-line cancel (CancelOrderItem) and order-level issue refund (RefundIssueToWallet)
both refund customer money pre-delivery via disjoint mechanisms; issue-refunds never flip
payment_status so #620's mutex misses them.

## Forward (issue-refunded line → later per-line cancel)
claimOrderItemForCancel (under the order lock) now blocks: (a) order.refunded_at != nil (full
issue refund keeps payment_status=completed), (b) lineRefundedViaIssue — a resolved/auto_refunded
issue with refund_amount>0 naming the line → errLineAlreadyRefunded/409.

## Reverse-automatic (line cancelled → issue reported → auto-refund)
ReportIssue now skips is_cancelled items when building affectedSubtotals/validAffected → the
cancelled line's value is excluded from RequestedAmount → auto-refund can't return it twice.

## Verify
go-developer verify (2 passes; 1st API-errored): forward fix correct+tested; identified the
reverse as a fully-AUTOMATIC double-refund on ordinary multi-item orders (minority line auto-
refunds under the cap) → fixed via the ReportIssue exclusion (verifier's prescribed fix).

## Residual (filed #624): report-pending → cancel → admin-resolve. RefundIssueToWallet should
re-exclude items cancelled after the report at resolve time. Human-mediated (admin approves) →
lower risk → follow-up.
