# #629 — RefundDay stamps the linked shell Order so an issue-refund can't double-pay a meal-plan day

## Bug
`RefundDay` (services/meal_plan_escrow.go) refunds a day (wallet key `mealplan-refund:<dayID>`)
but never touches the linked spawned shell `Order`. It stays paid/pending/RefundAmount=0, so a
customer `report-issue` on it passes `ReportIssue` and `RefundIssueToWallet` credits again on the
disjoint `issue:<id>` key → double refund. (Group orders aren't exposed — they flip consolidated
Order.Status→cancelled in-tx.)

## Fix
In `RefundDay`, when `day.OrderID != nil`, stamp the linked Order refunded in the same tx:
`status=refunded` (blocks ReportIssue's status check regardless of amount rounding) +
`refund_amount=perDayGross` (drives RemainingRefundable→0; also the durable block at
`RefundAmount>=Total` even if a later event flips status) + `refunded_at` + reason + initiated_by.

## Why safe
RefundDay only runs on UNDELIVERED days (undelivered/declined/failed) — never `status='delivered'`
— so it can't touch the weekly statement (selects delivered orders), and a legit quality-issue
refund on a DELIVERED day (never RefundDay'd) stays open. Atomic (RefundDay's tx) + idempotent
(RefundTxnID early-return). perDayGross uses the plan snapshot == the shell Order.Total basis.

## Tests (RED-first)
- RefundDay stamps the linked shell order refunded (status + refund_amount=perDayGross + refunded_at).
- nil OrderID still refunds cleanly, no order touched.

## Out of scope (filed follow-up)
Resurrection: the Shadowfax webhook writes shell order status UNCONDITIONALLY (no WHERE guard) +
RefundUndeliveredDays/RefundDeclinedDays don't terminalize MealPlanDay.Status → a refunded shell
whose 3PL delivery is in flight could be flipped to `delivered` and fold into the statement
(reporting overstatement; actual payout still blocked by orderRefundBlocks' refunded_at check).
Pre-existing, not worsened by this diff; the refund_amount stamp keeps the double-refund blocked.
