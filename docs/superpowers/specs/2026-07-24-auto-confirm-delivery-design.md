# Auto-confirm delivery — durable reminder + auto-confirm workflow

**Date:** 2026-07-24
**Status:** Approved (design)

## Goal

When an order is marked **delivered** but the customer hasn't confirmed receipt,
nudge them, then — if they never act — auto-confirm on their behalf so the order
is marked fulfilled and the chef's payout can proceed. Make it **durable and
reliable** via Temporal (survives restarts/deploys) + NATS (event fan-out).

Concretely, per the product owner:

> "A proper reminder every 10 mins for the next 3 times to ensure the customer
> confirms the delivery, and if still no action then it gets auto-confirmed and
> the delivery is marked completed and the payment release is done accordingly
> automatically."

## The key decision: confirm ≠ release

Auto-confirm does **not** pay the chef directly. The platform already separates
two states on the order:

- `PayoutHoldStatus: awaiting_customer_confirmation → release_eligible` — the
  order is **confirmed / fulfilled**. Set by `ConfirmOrderHold`. Moves **no
  money.**
- The **release sweep + governor** (`payout_release_cron.go` + `payouts.DecideRelease`)
  moves the money later on its own schedule (maturation window, new-chef ramp,
  dispute/refund checks, per-chef automation flag).

So this feature's job is only: **remind → at ~30 min auto-confirm → `release_eligible`.**
The existing sweep releases the money "accordingly automatically." No new money
logic, no new escrow path.

## What already exists (reuse, do not duplicate)

- `services.ConfirmOrderHold(db, order) (PayoutHoldStatus, error)` — the exact
  transition the customer's confirm button uses. Idempotent (guarded on
  `customer_confirmed_at IS NULL`); routes to `disputed` if an OrderIssue is open.
- `handlers.PayoutHoldHandler.ConfirmOrderReceived` — the `POST /v1/orders/:id/confirm-received`
  handler (customer taps "Confirm received").
- `services.SetOrderHoldAwaitingConfirmation(db, orderID)` — called at both
  delivered-transition sites: `handlers/chefs.go` (`UpdateOrderStatus`, self-delivery)
  and `handlers/delivery.go` (3PL `DeliveryDelivered`).
- `services.payout_auto_confirm_cron.go` — an existing **24h** fallback that
  auto-confirms via the same `ConfirmOrderHold`. Registered as Temporal Schedule
  `payout-auto-confirm`.
- Temporal runtime: `temporal.Runtime.Start/Signal`, `TaskQueueOrders`, workflow
  IDs `homechef:<domain>:<id>`, `apitemporal.Activities(ctx, timeout)`. Template:
  `temporal/workflows/order.go` `OrderSagaWorkflow` (timer + signal `select`).
- Producer/gating pattern: `services/temporal_order.go` `StartOrderSaga` /
  `signalOrderSaga` (guarded by `sagaActive()` = runtime present + config flag).
- Push: `services.SendPushNotification(userID, title, body, data)`; durable
  notification activity via `EnqueueNotification` / `NotificationWorkflow`.
- Settings fold: `services/payout_hold.go` `GetCustomerConfirmWindowHours` pattern
  (read `payout.%` keys). NATS outbox: `services.EnqueueEvent`.

## Architecture

```
delivered transition (chefs.go / delivery.go)
  → SetOrderHoldAwaitingConfirmation (existing)
  → StartConfirmReceiptFlow(orderID)         [new producer, gated]
        → Temporal workflow  homechef:confirm:<orderID>  on TaskQueueOrders

ConfirmReceiptWorkflow(orderID):
  for i in 1..maxReminders (default 3):
     selector:
        timer(reminderInterval, default 10m) → fire ReminderActivity(i) → continue
        signal "order.confirmed"             → return (customer confirmed)
        signal "order.disputed"              → return (dispute opened)
  # exhausted all reminders, still no action:
  if AutoConfirmActivity(orderID) reports still-unconfirmed:
        ConfirmOrderHold → release_eligible (or disputed)   [money handled by existing sweep]

ConfirmOrderReceived handler (customer taps):
  → existing ConfirmOrderHold
  → signal workflow "order.confirmed"        [new, best-effort]
```

### Components

1. **Producer** — `services/temporal_confirm.go`
   - `StartConfirmReceiptFlow(orderID uuid.UUID)`: no-op unless
     `confirmFlowActive()` (runtime present + `CONFIRM_RECEIPT_FLOW_ENABLED`,
     default **off**). Starts workflow id `homechef:confirm:<orderID>` on
     `TaskQueueOrders` (idempotent workflow-id reuse).
   - `SignalOrderConfirmed(orderID)` / `SignalOrderDisputed(orderID)`: best-effort
     `runtime.Signal`; swallow "workflow not found" (flow off, or already ended).
   - Called from the two delivered sites (start) and from `ConfirmOrderReceived` +
     the dispute path (signals).

2. **Workflow** — `temporal/workflows/confirm_receipt.go`
   - `ConfirmReceiptWorkflow(ctx, ConfirmReceiptInput{OrderID, ReminderIntervalSeconds, MaxReminders})`.
   - `workflow.NewSelector` with `NewTimer` + two signal channels (`order.confirmed`,
     `order.disputed`). Loop up to `MaxReminders`; each timer expiry runs
     `ReminderActivity`. Any signal ends the loop and returns.
   - After the loop: call `AutoConfirmActivity`. Return.
   - Reminder count / interval come from the **input** (captured at start from
     settings), so a running workflow is deterministic even if settings change.

3. **Activities** — `temporal/workflows/confirm_receipt.go` (transports wired in
   `cmd/worker/main.go`, same pattern as `SendFunc`/`OrderSettleFunc`):
   - `ReminderActivity(orderID, attempt)` → `ConfirmReminderFunc` →
     `services.SendConfirmReceiptReminder(orderID, attempt)`: loads order+chef,
     skips if already confirmed/disputed/terminal, else sends a push
     ("Did your order from {chef} arrive? Tap to confirm.") with
     `data{order_id, type:"confirm_receipt"}` and stages an outbox event.
   - `AutoConfirmActivity(orderID)` → `AutoConfirmFunc` →
     `services.AutoConfirmOrderReceipt(orderID)`: re-reads the order; if
     `customer_confirmed_at` is still null and not terminal, calls
     `ConfirmOrderHold` (→ `release_eligible`/`disputed`) and stages an outbox
     event `orders.auto_confirmed`. Fully idempotent (guarded UPDATE).

4. **Config** (new `payout.*` keys, read via the settings-fold pattern, with env
   defaults): `payout.confirm_reminder_interval_minutes` (10),
   `payout.confirm_reminder_max_count` (3). Read once in `StartConfirmReceiptFlow`
   and passed as workflow input.

5. **NATS**: new subjects `orders.confirm_reminder` and `orders.auto_confirmed`
   on the existing `ORDERS` stream (`orders.>` already covered). Staged via the
   transactional outbox inside the activities' DB work.

## Reconciliation with the existing 24h cron

`payout-auto-confirm` (24h) stays as the **belt-and-suspenders fallback** for the
Temporal-off path and for any delivered order whose workflow never started
(deploy gaps). Because both paths call the same guarded `ConfirmOrderHold`, an
overlap is a **safe no-op** (the second UPDATE affects 0 rows). When the flow
flag is **on**, the workflow reaches the order first (~30 min ≪ 24h). No change
to the cron is required; it is not a duplicate, it is the backstop.

## Idempotency & edge cases

- **Workflow id keyed per order** → a retried producer call reuses the same run.
- **Customer confirms mid-window** → `ConfirmOrderReceived` already flips the hold;
  the signal ends the workflow early; even without the signal, `AutoConfirmActivity`
  re-reads state and no-ops (guarded UPDATE).
- **Dispute opened** → `ConfirmOrderHold` routes to `disputed`, never
  `release_eligible`; the dispute signal ends the workflow.
- **Order cancelled/refunded after delivery** → `AutoConfirmActivity` treats a
  terminal/refunded order as "nothing to do".
- **Push token missing** → `SendPushNotification` already no-ops; reminders are
  best-effort, auto-confirm still fires.
- **Flow flag off** → producer no-ops; 24h cron still auto-confirms. Safe rollout.

## Testing plan (TDD)

- `services.AutoConfirmOrderReceipt`: confirms an awaiting order → `release_eligible`;
  no-ops on already-confirmed, disputed, terminal, refunded. (unit, sqlite/gorm)
- `services.SendConfirmReceiptReminder`: skips confirmed/terminal; composes push +
  stages outbox for awaiting. (unit, with a stub push sink)
- `ConfirmReceiptWorkflow`: Temporal test env — (a) 3 timers fire 3 reminders then
  auto-confirm; (b) `order.confirmed` signal after reminder 1 → 1 reminder, no
  auto-confirm; (c) `order.disputed` signal → stops, no auto-confirm. Assert
  activity call counts.
- Producer gating: `StartConfirmReceiptFlow` no-ops when flag off / runtime nil.
- Existing dashboard/order tests stay green.

## Files

- **New:** `apps/api/temporal/workflows/confirm_receipt.go`,
  `apps/api/services/temporal_confirm.go`,
  `apps/api/services/confirm_receipt.go` (activity bodies:
  `SendConfirmReceiptReminder`, `AutoConfirmOrderReceipt`), plus tests.
- **Edit:** `cmd/worker/main.go` (register workflow+activities, wire transports),
  `temporal/queues.go` if a signal-name const home is wanted,
  `services/nats.go` (2 subjects), `handlers/chefs.go` + `handlers/delivery.go`
  (call `StartConfirmReceiptFlow` at the delivered sites),
  `handlers/payout_hold.go` (signal `order.confirmed` on manual confirm; signal
  `order.disputed` where a dispute/issue is opened), `config/config.go`
  (`CONFIRM_RECEIPT_FLOW_ENABLED` + reminder defaults).

## Out of scope

- Moving money faster (release timing stays with the existing governor).
- Changing the 24h cron's behavior.
- Mobile changes (the reminder is a push; the existing order screen already has
  the Confirm-received flow).
