# #631 — guard delivery "delivered" writers against resurrecting a refunded/terminal order/day

## Bug
A late/replayed 3PL or own-fleet `delivered` event can flip an already-refunded/cancelled order (or
meal-plan day) back to `delivered`:
- shadowfax_webhook.go:84 wrote `Order.status='delivered'` with only `WHERE id=?` (no guard).
- handlers/delivery.go:623 (own-fleet) did the same via `Model(&delivery.Order).Updates(...)`.
- MarkMealPlanDayDelivered guarded the DAY on STATUS only; RefundUndeliveredDays/RefundDeclinedDays
  refund via RefundDay but leave the day status non-terminal → a delivered event resurrects it.
A resurrected shell Order (status=delivered) folds into the weekly statement (selects delivered).

## Fix (asymmetric-mirror guards, #534/#393/#590 style)
- Both order→delivered writes now guard `status NOT IN [cancelled,refunded,rejected]`
  (`resurrectionTerminalOrderStatuses`, shared services var; inlined in the handler).
- MarkMealPlanDayDelivered adds `refund_txn_id IS NULL` (the durable refunded marker) to its
  guarded UPDATE + an early `day.RefundTxnID != nil` return.

## Money note
The HOLD was already safe: SetOrderHoldAwaitingConfirmation / SetMealPlanDayHoldAwaitingConfirmation
only advance from `none`, so a refunded (withheld/reversed) hold was never re-parked. These guards
close the STATUS resurrection → statement overstatement + day-status consistency.

## Tests (RED-first)
- shadowfax delivered on a refunded order → stays refunded; on an active order → delivered (regression).
- MarkMealPlanDayDelivered on a refunded (refund_txn_id set, status=confirmed) day → not resurrected.
- harness: added refund_txn_id to setupHoldHandlerDB meal_plan_days DDL.
