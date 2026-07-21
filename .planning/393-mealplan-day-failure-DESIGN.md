# Design — #393 meal-plan-day delivery-failure (investigated, ready to execute)

Part of #393. Investigated 2026-07-06 (session 7); paused before implementation due to context limits. This is the design; no code written yet.

## The problem
A failed/returned delivery on a meal-plan-DAY order currently does nothing to the day:
`terminalizeFailedDelivery`/`ReportChefDeliveryFailure` → `services.TerminalizeDeliveryFailure`
→ `RecordDeliveryFailure` which SKIPS non-gateway orders. The per-day fulfilment order
(`generateDayOrder`, meal_plan_fulfillment.go:199) has **no `RazorpayOrderID`** (it's a
shell; `Total`=dayTotal, `Subtotal`=day price), and the day's money lives on the
`meal_plan_days` row (`PayoutHoldStatus`, `PayoutTransferID`). So the day is left stuck at
Prepared/Confirmed (non-terminal) → `allDaysTerminal` never true → **the plan never
completes**, and the day is never resolved (customer not refunded / chef not paid).

## Key facts
- `MealPlanDay`: `OrderID *uuid`, `Status` (varchar(12)), `PayoutHoldStatus`, `PayoutTransferID`, `RefundTxnID`, `Price`.
- Terminal day statuses (`allDaysTerminal`, meal_plan_fulfillment.go:248): Delivered, Skipped, Cancelled, Refunded, Declined.
- `MarkMealPlanDayDelivered(orderID)` (meal_plan_fulfillment.go:280) = the delivered mirror: finds day by order_id, guards terminal states, status→delivered, `SetMealPlanDayHoldAwaitingConfirmation`, emits event.
- `RefundDay(tx, plan, day, reason)` (meal_plan_escrow.go:284) = the day-refund path: reverse the held transfer + wallet-credit `perDayGross` + `reverseRefundedDayHold`. Flag-gated on `MealPlanEscrowActive()`.
- Day hold helpers: `SetMealPlanDayHoldAwaitingConfirmation` (payout_hold.go:55). `aggTypeMealPlanDay = "meal-plan-day"` (payout_release.go:50). `emitHoldEvent(tx, PayoutHoldDisputed, aggType, id)` emits `payout.hold_disputed`.
- Harness: `setupCrossguardDB` has meal_plan_days; `seedCrossDay(t, db, hold, &orderID)` (status='delivered', transfer 'trf_abc123', price 120); `loadDayHold`.

## Slice A — DAY FREEZE (money-safe, ship first; mirrors order slice 1)
1. `models.MealPlanDayFailed = "failed"` (6 chars, fits varchar(12); NEW status VALUE — no migration since the column already accepts strings). **Do NOT add to `allDaysTerminal`** — a failed day is non-terminal (plan waits for resolution).
2. `services.SetMealPlanDayHoldDisputed(tx, dayID)` — guarded UPDATE `payout_hold_status IN (none, awaiting_customer_confirmation) → disputed`; no-op on eligible/released/reversed/withheld (#458 invariant); `emitHoldEvent(tx, PayoutHoldDisputed, aggTypeMealPlanDay, dayID)`. Mirror `SetOrderHoldDisputed`.
3. `services.MarkMealPlanDayFailed(tx, orderID) (bool, error)` — find day by order_id (none → froze=false); guarded UPDATE status→failed WHERE status NOT IN (terminal set + failed); if RowsAffected==0 → froze=false; else `SetMealPlanDayHoldDisputed` + emit `SubjectMealPlanDayFailed` (add `= "meal_plans.day_failed"`); return froze=true. Mirror `MarkMealPlanDayDelivered`.
4. Hook into `services.TerminalizeDeliveryFailure`: after `RecordDeliveryFailure` returns froze=false, try `MarkMealPlanDayFailed(tx, order.ID)`; froze = either. (Gateway order → order freeze; meal-plan-day shell → day freeze.) Still emits the one `delivery.failed` notification on froze.
- Tests: SetMealPlanDayHoldDisputed freezes/no-ops; MarkMealPlanDayFailed freezes+marks+idempotent+not-a-day-order; TerminalizeDeliveryFailure on a day-linked non-gateway order freezes the DAY; allDaysTerminal still false with a `failed` day.
- **Value:** failed days no longer silently stall; the day hold is explicitly disputed (admin-queue visible) + can't be re-parked to awaiting by a late delivered event.

## Slice B — DAY RESOLUTION (follow-up; the intricate part)
The hybrid admin-confirm model (owner policy [[project_rto_money_policy]]) applies to days too:
customer-fault → chef paid + day terminalized; platform/chef-fault → `RefundDay` + day→refunded.
- The #582 order-resolver does NOT fit meal-plan days: opening an OrderIssue on the shell
  per-day order would (a) route to `RefundIssueToWallet(order.Total)` — a SECOND refund
  path alongside `RefundDay` (double-refund risk), and (b) not advance the day STATUS.
- So Slice B needs a **day-specific admin fault-confirm** action: customer-fault → release the
  day hold (disputed→release_eligible) + day status→a paid-terminal (new value or delivered) ;
  platform/chef-fault → `RefundDay` (drives hold→reversed/withheld + wallet refund) + day→refunded.
  Then re-check `allDaysTerminal` → complete the plan.
- Needs care: `RefundDay` is flag-gated (`MealPlanEscrowActive`), sets `RefundTxnID` not
  `Status` — verify who sets day→refunded; the reverseRefundedDayHold drift semantics (#398);
  and per-day-gross vs order.Total. This is escrow money code → its own focused PR + verify.
- Admin UI (tesserix-home) surface for day resolution = part of the UI slice.

## Sequencing
Slice A (freeze, money-safe) → Slice B (resolution, escrow money, own PR) → UI. Same
pattern as orders (slice 1 #580 freeze → #582 resolve). Do NOT rush Slice B.
