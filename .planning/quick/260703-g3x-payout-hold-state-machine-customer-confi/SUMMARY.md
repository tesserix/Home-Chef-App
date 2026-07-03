# Payout Hold State Machine + Customer Confirmation (#387) — SUMMARY

Backend-only. Delivered orders/days no longer release chef funds directly: on
delivery the hold becomes `awaiting_customer_confirmation`; an explicit customer
confirmation advances it to `release_eligible` unless an open `OrderIssue`
disputes it. `release_eligible` moves NO money — the real Razorpay release stays
behind the admin payout queue (#388). Live behaviour is unchanged (money movement
still gated by the existing escrow flags; setting the hold is plain DB state).

## Files changed & why

**New**
- `apps/api/models/payout_hold.go` — `PayoutHoldStatus` enum (`""`/`awaiting_customer_confirmation`/`release_eligible`/`released`/`disputed`).
- `apps/api/services/payout_hold.go` — the seam: `SetOrderHoldAwaitingConfirmation`, `SetMealPlanDayHoldAwaitingConfirmation`, `HasOpenOrderIssue`, `ConfirmOrderHold`, `ConfirmMealPlanDayHold`, `ConfirmTodaysTiffinForCustomer`, `GetCustomerConfirmWindowHours`. Every transition is a guarded conditional `UPDATE` (idempotent; disputed/released can never flip to release_eligible).
- `apps/api/handlers/payout_hold.go` — `PayoutHoldHandler` with `ConfirmOrderReceived`, `ConfirmMealPlanDayReceived`, `ConfirmTodaysTiffin` (owner-scoped, idempotent, no raw error codes).
- `apps/api/migrations/20260703000001_add_payout_hold_state.{up,down}.sql` — auditable DDL for the two new columns on `orders` + `meal_plan_days` (AutoMigrate is the runtime mechanism).
- `apps/api/services/payout_hold_test.go`, `apps/api/handlers/payout_hold_test.go` — RED-first tests (17 total).

**Model fields** — `PayoutHoldStatus` + `CustomerConfirmedAt` added to `models/order.go` (`Order`) and `models/meal_plan.go` (`MealPlanDay`).

**Decouple — the plan-check fix (5 SetOrderHold sites + meal-plan-day chokepoint):**
- `services/provider.go` — 3PL generic webhook `delivered`: `ReleaseOrderPayouts` -> `SetOrderHoldAwaitingConfirmation`.
- `services/shadowfax_webhook.go` — Shadowfax webhook `delivered`: same swap.
- `services/temporal_order.go` — `SettleOrderPayouts` saga activity: `return ReleaseOrderPayouts(...)` -> `return SetOrderHoldAwaitingConfirmation(database.DB, ...)` (preserves the error-return retry semantics).
- `handlers/delivery.go` — retired own-fleet courier: same swap.
- `handlers/chefs.go` delivered case — ADDED `SetOrderHoldAwaitingConfirmation` (chef self-delivered regular orders now also get a hold; no-op for tiffin/group via the `razorpay_order_id` gate). Existing `MarkMealPlanDayDelivered`/`MarkGroupOrderDelivered` calls left untouched.
- `services/meal_plan_fulfillment.go` — inside `MarkMealPlanDayDelivered` (shared by all completion paths): `ReleaseDayPayout(tx,&day)` -> `SetMealPlanDayHoldAwaitingConfirmation(tx, day.ID)`; status=delivered update and `SubjectMealPlanDayDelivered` event preserved.
- `services/order_payout.go` / `services/meal_plan_escrow.go` — `ReleaseOrderPayouts`/`ReleaseDayPayout` left DEFINED (production-uncalled) with a one-line #388 seam comment.

**Routes** (`routes/routes.go`): `POST /orders/:id/confirm-received`, `POST /meal-plans/:id/days/:dayId/confirm-received`, and a dedicated `POST /tiffin/confirm-today` group (a static segment can't share the `/orders/:id` or `/meal-plans/:id` wildcard slot). Handler constructed as `payoutHoldHandler := handlers.NewPayoutHoldHandler()`.

## Confirm / dispute invariant

Only one `UPDATE` produces `release_eligible`, gated on
`payout_hold_status = 'awaiting_customer_confirmation'`. An open (pending)
`OrderIssue` forces `disputed` at confirm time. Idempotent: an already-confirmed
order (`CustomerConfirmedAt != nil`) returns its terminal status without
re-stamping. Ownership: order loaded `WHERE id=? AND customer_id=?` (404 for
non-owner); meal-plan day verified against `plan.customer_id`.

## Verification output

```
=== go test ===
ok  github.com/homechef/api/services
ok  github.com/homechef/api/handlers
ok  github.com/homechef/api/models

=== go vet ./services/ ./handlers/ ./models/ ===
clean (exit 0)

=== gofmt -l (payout_hold.go / handlers / models / routes) ===
(empty — clean)

=== SetOrderHoldAwaitingConfirmation call-site counts ===
services/provider.go:1  services/shadowfax_webhook.go:1  services/temporal_order.go:1
handlers/delivery.go:1  handlers/chefs.go:1

=== grep gate: surviving ReleaseOrderPayouts references ===
services/order_payout.go        (definition + #388 seam)
services/order_payout_test.go   (existing seam test)
```

17 new tests pass: SetOrderHold regular-vs-consolidated, SetMealPlanDayHold,
SettleSaga-holds-no-release (flag ON), MarkMealPlanDayDelivered parks-no-release
(escrow on+off), Confirm->release_eligible, open-issue->disputed, idempotent,
disputed-stays-disputed, window default+override, ConfirmMealPlanDayHold, owner
200, non-owner 403/404, handler idempotent.

## Deviations

- **routes.go gofmt alignment** (Rule 3, trivial): the longer `confirm-received`
  route line shifted the trailing `// #238` comment alignment in the orders block;
  ran `gofmt -w routes/routes.go`. No logic change.
- **Grep-gate comment reword**: an early doc comment in `payout_hold.go` named
  `ReleaseOrderPayouts`/`ReleaseDayPayout`, which tripped the filename-based grep
  gate. Reworded to reference "the release/reverse seams in order_payout.go /
  meal_plan_escrow.go" so the only surviving literal references are the two
  expected files. No behaviour change.
- No unforeseen compile breaks in callers. Scope held to the plan.

## Follow-ups (out of scope, tracked)

- **#388 admin payout queue**: consume `release_eligible` and drive the real
  Razorpay `ReleaseTransfer` off `ReleaseOrderPayouts`/`ReleaseDayPayout`. #388
  also owns POST-eligible dispute rollback — it MUST re-check `HasOpenOrderIssue`
  immediately before `ReleaseTransfer` (the confirm-time gate does not roll an
  already-eligible hold back to `disputed`).
- **Auto-confirm timeout sweep** (cron/Temporal) reading `GetCustomerConfirmWindowHours`
  to auto-advance stale `awaiting_customer_confirmation` holds.
- **NATS events** `payments.hold_release_eligible` / `payments.hold_disputed` —
  emit seam is the confirm transition in `services/payout_hold.go`.
- **Group-order hold decoupling**: `MarkGroupOrderDelivered -> ReleaseGroupChefPayout`
  is NOT flag-gated and STILL fires on every completion path — group/office orders
  still auto-release, so "no delivered path auto-releases" is not fully true yet.
  Needs `GroupOrder` hold fields + a group confirm surface.
- **Mobile UI**: customer confirm screen + vendor payout-state display.
- No changes to refund/reversal semantics (`ReverseOrderPayouts`, `RefundDay`,
  order-issue refunds) or group-order release in this slice.

## Commit SHAs (branch feat/payout-hold-state-machine)

- 85a932ab test: add failing payout hold state machine tests (#387)
- 98835071 feat: add payout hold status enum, order/day fields, migration (#387)
- d8f9f213 feat: park payout hold on delivery across all completion paths (#387)
- 72194f26 feat: add customer confirm endpoints + hold transition logic (#387)
- 28a7bb35 feat: add GetCustomerConfirmWindowHours settings getter (#387)
- 5a93c20f refactor: reword hold seam comment + gofmt routes alignment (#387)
