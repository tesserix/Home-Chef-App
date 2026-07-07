# #624 — RefundIssueToWallet re-excludes items cancelled AFTER an issue was filed

## Bug
Report an issue naming line A (still live) → amount > AutoApproveCap → issue stays PENDING with
`RequestedAmount` including A → chef cancels A via `CancelOrderItem` (refunds A, records
`order_items.refund_amount`, adds it back into `RemainingRefundable`) → admin later resolves the
pending issue → `RefundIssueToWallet` refunds A's value a SECOND time. The order-level
`RemainingRefundable` cap doesn't bind because the per-line add-back plus other live lines leave
headroom. `AdminResolveIssue`'s pre-cap is the naive `Total − RefundAmount`.

## Fix (mirror the #622 ReportIssue exclusion, at resolve time, under the order lock)
1. `services.RefundIssueToWallet` (services/order_issue.go): under the existing order lock, when
   the issue names affected items, subtract the per-line refund already issued for any affected
   line now `is_cancelled` from the issue's own reported value (`RequestedAmount`) and cap the
   credit to it. Only bites when an affected line was cancelled after report → no change to the
   auto path or admin discretion when nothing was cancelled. New tx helper
   `cancelledAffectedRefundTx` (errors propagate → tx aborts → money-safe, never over-refunds).
2. `handlers.AdminResolveIssue`: align the pre-cap from naive `Total − RefundAmount` onto
   `services.RemainingRefundable`, and map `ErrNothingToRefund` → 409 (the #624 cap can now zero
   the credit when every affected line was already cancelled).

## Tests (RED-first)
- service: affected line cancelled after report → only the still-live affected line is refunded
  (not doubled); all affected cancelled → ErrNothingToRefund; no-cancel → admin discretion kept.
- handler: AdminResolveIssue caps to remaining + returns 409 when nothing is left.
- harness: add `refund_amount` to the handler order_items DDL (RemainingRefundable/helper SELECT it).

## Guardrails
Flag-independent (customer-facing wallet refund; no escrow flag). Money-safe: fail toward
under-refund. Adversarial go-developer verify before merge.
