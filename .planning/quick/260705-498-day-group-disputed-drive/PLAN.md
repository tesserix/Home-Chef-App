# #498 — Drive meal-plan-day / group-order disputed holds on issue resolve/reject

**Epic #403. P2, flag-gated (money stays held). Extends #458 (order aggregate only) to the day & group aggregates.**

## Problem
`ConfirmMealPlanDayHold` / `ConfirmGroupOrderHold` can move a day/group payout hold to `disputed`
(an open `OrderIssue` on the linked order at confirm time). #458 surfaces those disputed rows in
the admin queue (`IncludeDisputed`) but only the **order** aggregate is driven out of `disputed`
on issue resolve/reject — a disputed day/group hold is stuck in the dead-end.

## Key facts established by code read
- `MealPlanDay` and `GroupOrder` both have `OrderID *uuid.UUID` (indexed), `PayoutHoldStatus`,
  `PayoutTransferID`. `MealPlanDay` also has `RefundTxnID *uuid.UUID`. `GroupOrder` has `Status`
  (`cancelled` terminal); group refunds are per-participant (no group-level refund column).
- The dispute source for a day/group is its **linked order** (`applyHoldConfirm` keys on
  `day.OrderID` / `g.OrderID`). So an `OrderIssue` on order X ⇒ fan out by `WHERE order_id = X`.
- **The money seam for day/group reverse is ALREADY wired**: `reverseMoney` (payout_release.go:594-610)
  reverses the day transfer (`ReverseTransfer(day.PayoutTransferID,0)`, escrow-flag gated) and the
  group transfer (`ReverseGroupChefPayout`). So driving a day/group hold via the existing
  `transitionHold` + `settleReverse` path carries the money seam — NO state-without-seam drift.
  (The issue's "do together with the refund seam" caveat predates that wiring.)
- `refundedOrderSubquery` / `openOrderIssueSubquery` are reusable `NewDB` EXISTS-predicate builders.
- The existing order drives: `ReleaseDisputedOrderHoldIfCleared` (reject) and
  `WithholdOrReverseOrderHoldForRefund` (refund), both order-only.

## Design

### Part A — Reject path: disputed → release_eligible, fan out by order_id
New `services.ReleaseDisputedHoldsForOrderIfCleared(tx, orderID)`:
1. `ReleaseDisputedOrderHoldIfCleared(tx, orderID)` (existing — order self).
2. `SELECT id FROM meal_plan_days WHERE order_id=orderID AND payout_hold_status='disputed'`;
   for each id → single-row guarded transition.
3. `SELECT id FROM group_orders  WHERE order_id=orderID AND payout_hold_status='disputed'`;
   for each id → single-row guarded transition.

Single-row day transition (race-safe; correctness from the guarded UPDATE, the SELECT only enumerates):
```
UPDATE meal_plan_days SET payout_hold_status='release_eligible'
WHERE id=? AND payout_hold_status='disputed'
  AND refund_txn_id IS NULL                              -- day itself not refunded
  AND NOT EXISTS (refundedOrderSubquery(tx, orderID))    -- linked order not refunded/cancelled
  AND NOT EXISTS (openOrderIssueSubquery(tx, orderID))   -- no remaining pending issue on the order
```
emit `hold_release_eligible` on RowsAffected>0. Group transition identical but guard
`status <> 'cancelled'` instead of `refund_txn_id IS NULL`.

Wire into `AdminRejectIssue` (handlers/order_issue.go:357): swap
`ReleaseDisputedOrderHoldIfCleared` → `ReleaseDisputedHoldsForOrderIfCleared` (same tx, so the
just-rejected issue is visible to the NOT EXISTS(pending) guard).

### Part B — Resolve/refund path: → withheld / reversed, fan out by order_id
Extend `WithholdOrReverseOrderHoldForRefund(db, orderID, reason)` so that after driving the order
it also drives linked day/group holds:
- `SELECT id FROM meal_plan_days WHERE order_id=orderID` → `withholdOrReverseHold(db, aggTypeMealPlanDay, id, reason)`
- `SELECT id FROM group_orders  WHERE order_id=orderID` → `withholdOrReverseHold(db, aggTypeGroupOrder, id, reason)`

`withholdOrReverseHold(db, aggType, id, reason)` = the current order-only body generalized over
aggType (load `payout_hold_status`; eligible|awaiting|disputed→withheld; released→reversed then
`settleReverse` — which runs the existing `reverseMoney` seam). Keep `WithholdOrReverseOrderHoldForRefund`
as the entry point (all 5 existing order-refund callers now fan out automatically). Best-effort,
idempotent (`transitionHold` conditional WHERE ⇒ double refund never double-reverses).

### Part C — Direct RefundDay → day-hold drive (close refunded-day double-pay gap)
`RefundDay` (meal_plan_escrow.go:230) reverses the transfer but leaves `payout_hold_status`. The
admin release guard for a day keys on the **linked order**'s refund state, NOT `day.RefundTxnID`, so
a day refunded via skip/expiry/decline (no order refund) can still be released ⇒ double-pay. Fix:
after the wallet refund in `RefundDay`, call `withholdOrReverseHold(tx, aggTypeMealPlanDay, day.ID, reason)`
so the refunded day leaves the releasable set. (No-op when the hold status is none/withheld/reversed.)

`RefundGroupParticipant` is intentionally OUT of scope: refunding one participant does not mean the
chef (paid the whole group slice) should be clawed back. Group-cancel → group-hold reverse is W-A
(#456 follow-up / task #5); note the seam there, don't duplicate here.

## Tests (RED-first, sqlite in-memory like payout_crossguard_refund_test.go)
- `TestReleaseDisputedHoldsForOrder_DrivesLinkedDay` — day disputed + order_id=X, reject clears X's
  issue ⇒ day → release_eligible.
- `TestReleaseDisputedHoldsForOrder_DrivesLinkedGroup` — same for group.
- `TestReleaseDisputedHoldsForOrder_PendingIssueBlocksDay` — a second pending issue on X ⇒ day stays disputed.
- `TestReleaseDisputedHoldsForOrder_RefundedDayBlocked` — day.refund_txn_id set ⇒ stays disputed.
- `TestReleaseDisputedHoldsForOrder_CancelledGroupBlocked` — group cancelled ⇒ stays disputed.
- `TestWithholdOrReverseOrderHoldForRefund_FansOutToDayGroup` — order refund withholds linked day+group.
- `TestWithholdOrReverseOrderHoldForRefund_ReleasedDayReversed` — linked day released → reversed.
- `TestRefundDay_DrivesDayHoldWithheld` — RefundDay on an eligible/awaiting day ⇒ withheld.
- Callback-injection race test: an issue filed concurrently with the reject-drive fails safe (day
  stays disputed) — mirror the #460/#496 pattern.

## Money-path guarantees preserved
- Every transition is a guarded conditional UPDATE; RowsAffected gates the emit (no double-emit).
- All reverse money runs through `settleReverse`→`reverseMoney`, escrow-flag gated (OFF at launch ⇒
  pure DB state advance). `stampPayoutSettled` keeps the reconcile safe.
- Fan-out is by the indexed `order_id`; single-row guarded transitions ⇒ no check-then-act window.

## Files
- `services/payout_hold.go` — `ReleaseDisputedHoldsForOrderIfCleared` + day/group single-row helpers.
- `services/payout_release.go` — generalize to `withholdOrReverseHold(aggType)`, fan-out in `WithholdOrReverseOrderHoldForRefund`.
- `services/meal_plan_escrow.go` — `RefundDay` drives the day hold.
- `handlers/order_issue.go` — reject wire-up.
- `services/*_test.go` — RED-first tests.
