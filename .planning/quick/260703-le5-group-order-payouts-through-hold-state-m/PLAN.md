---
type: quick
issue: 456
priority: P0
branch: feat/group-order-payout-hold
title: Route group/office-order chef payouts through the payout hold state machine + flag-gate
created: 2026-07-03
files_modified:
  - apps/api/models/group_order.go
  - apps/api/migrations/20260703000003_add_group_order_payout_hold.up.sql
  - apps/api/migrations/20260703000003_add_group_order_payout_hold.down.sql
  - apps/api/services/group_order_payout.go
  - apps/api/services/payout_hold.go
  - apps/api/services/payout_release.go
  - apps/api/services/payout_auto_confirm_cron.go
  - apps/api/services/payout_reconcile_cron.go
  - apps/api/handlers/admin_payout.go
  - apps/api/handlers/group_order.go
  - apps/api/handlers/payout_hold.go
  - apps/api/routes/routes.go
  - apps/api/services/group_order_payout_test.go
  - apps/api/services/payout_hold_test.go
  - apps/api/services/payout_release_test.go
autonomous: true
---

<objective>
GH #456 (P0, live money leak). `services/group_order_payout.go` releases the chef's
group-order Route transfer immediately on delivery with NO escrow-flag gate,
bypassing the entire #387/#388 payout hold machine (no customer confirm, no dispute
check, no admin release queue, no reconcile). `MarkGroupOrderDelivered(orderID)`
looks up the GroupOrder by `order_id`, flips its status to delivered, and calls
`ReleaseGroupChefPayout` directly.

Make the group order a first-class payout-hold aggregate (Option A):
1. Stop the live leak by flag-gating the three money seams on `payoutMovementEnabled()`.
2. Decouple delivery from release — park the group hold `awaiting_customer_confirmation`.
3. Wire a new `aggTypeGroupOrder = "group-order"` through the whole #387/#388 machine
   (hold set/confirm, release/reverse actuator, auto-confirm sweep, reconcile cron,
   admin release queue).
4. Add a host-scoped `POST /group-orders/:id/confirm-received` confirm endpoint.
5. Guard the cancel-path reverse behind the flag.

Purpose: no group-delivered code path may move chef money ungated; the group hold
must obey the same conditional-transition (race-safe) invariants as order/meal-plan-day.
Output: group orders flow money only via the flag-gated admin release queue off
`release_eligible`, exactly like regular orders and meal-plan days.
</objective>

<verification_findings>
Both pre-design questions are answered definitively by the code — grounded, re-verified.

## Q1 — Double-hold check: NO phantom double-hold. The order path is a genuine no-op.
The consolidated Order built at `handlers/group_order.go:757-782` sets `CustomerID:
g.HostID` but NEVER sets `RazorpayOrderID` — participants pay via their own
per-participant `GroupOrderParticipant.RazorpayOrderID` (`models/group_order.go:124`),
not on the consolidated order. So the consolidated order has `razorpay_order_id = ''`.
Consequences (all verified):
- `SetOrderHoldAwaitingConfirmation` early-returns when `RazorpayOrderID == ''`
  (`services/payout_hold.go:40-42`) → consolidated order stays `PayoutHoldNone`.
- It therefore never enters `listPendingOrders` (status stays none, excluded by
  `pendingStatuses()`), and `reconcileOrders` explicitly filters
  `razorpay_order_id <> ''` (`services/payout_reconcile_cron.go:99`) → excluded there too.
- The existing test `TestSetOrderHoldAwaiting_RegularVsConsolidated`
  (`services/payout_hold_test.go:96-100`) already asserts this exact no-op.
DECISION: leave the consolidated order with no `razorpay_order_id` (do NOT add one).
Only the new GroupOrder hold carries the group payout. Add a regression test
(Task 8) asserting the consolidated order stays `PayoutHoldNone` while the group
hold advances — locking the invariant against a phantom double-hold.

## Q2 — Confirm actor: the HOST.
Consolidated `Order.CustomerID = g.HostID` (`handlers/group_order.go:759`). The host
is the payer/receiver. Scope `POST /group-orders/:id/confirm-received` to the host
using the SAME pattern `CancelGroupOrder` already uses
(`loadGroupForParticipant(id, userID)` + `me.Role == models.GroupRoleHost`,
`handlers/group_order.go:857-865`). The dispute check keys on the consolidated
order's `HasOpenOrderIssue(*g.OrderID)` (only when `g.OrderID != nil`), mirroring
`ConfirmMealPlanDayHold`'s `day.OrderID` keying (`services/payout_hold.go:147`).
</verification_findings>

<constraints>
- Go conventions (CLAUDE.md): funcs <50 lines, `%w` wraps, early-return, no panics
  in handlers, `recover()` at cron sweep top, race-safe conditional UPDATEs guarded
  on `RowsAffected`.
- Mirror the existing #387/#388 primitives exactly — do NOT invent a parallel state
  machine. `transitionHold`/`applyHoldConfirm`/`settleRelease`/`settleReverse`/
  `stampPayoutSettled` are already aggType-generic; the work is adding the
  `group-order` case, a `holdModel` branch, a pending-list query, sweep/reconcile
  scans, and the set/confirm seams.
- Money seams stay OFF at launch (`payoutMovementEnabled()` == `OrderPayoutAutoReleaseEnabled`,
  default false) → every new transition is a pure DB advance until the flag flips.
- sqlite unit-test harness: hand-DDL the `group_orders` table + new columns into
  `setupHoldDB` / `setupReleaseDB` (gen_random_uuid() can't run on sqlite).
- Branch `feat/group-order-payout-hold` (already checked out, off latest main).
</constraints>

<tasks>

## Task 1 (RED): failing tests for the group-order hold aggregate
<files>apps/api/services/group_order_payout_test.go, apps/api/services/payout_hold_test.go, apps/api/services/payout_release_test.go</files>
<action>
Write the failing tests FIRST (they won't compile until Tasks 2-7 land — that is the RED).
Extend the sqlite harnesses: add a `group_orders` table to `setupHoldDB` and
`setupReleaseDB` with columns:
`id TEXT PRIMARY KEY, host_id TEXT, chef_id TEXT, order_id TEXT, status TEXT,
payout_transfer_id TEXT DEFAULT '', payout_hold_status TEXT DEFAULT '',
customer_confirmed_at DATETIME, delivered_at DATETIME, payout_settled_at DATETIME,
payout_settle_attempts INTEGER DEFAULT 0, subtotal REAL DEFAULT 0, tax REAL DEFAULT 0,
currency TEXT DEFAULT 'INR', created_at DATETIME, updated_at DATETIME`.
Add a `seedGroupOrder(t, db, hold)` helper (mirror `seedRegularOrder`).

Tests to add:
- `TestMarkGroupOrderDelivered_ParksHold_NoRelease` — seed a placed group order with a
  `PayoutTransferID`; call `MarkGroupOrderDelivered(orderID)` with the money flag BOTH
  off and on (set `config.AppConfig.OrderPayoutAutoReleaseEnabled`); assert status ==
  delivered, `payout_hold_status == awaiting_customer_confirmation`, `delivered_at`
  stamped, and NO `ReleaseTransfer` reached (GetRazorpay() is nil in tests, so a real
  release would error — assert MarkGroupOrderDelivered returns cleanly and the hold is
  parked, never released).
- `TestConfirmGroupOrderHold_ReleaseEligible` — host confirm on an awaiting group hold
  → `release_eligible`, `customer_confirmed_at` stamped, `payout.hold_release_eligible`
  outbox event emitted once; re-confirm is a no-op (no double emit).
- `TestConfirmGroupOrderHold_Disputed` — open `order_issues` row (status pending) on the
  consolidated `order_id` → confirm lands `disputed`, never `release_eligible`.
- `TestReleaseHold_GroupOrder` — `ReleaseHold(db, "group-order", id)` on a
  `release_eligible` group hold → `released`; re-release → `ErrHoldNotEligible` (409
  path). Flag off → state-only. Flag on but GetRazorpay()==nil → seam reached,
  release path attempted.
- `TestListPendingPayouts_IncludesGroupOrders` — a `release_eligible` group hold shows
  up in `ListPendingPayouts` with `AggType == "group-order"`.
- `TestSweepAndReconcile_CoverGroupOrders` — a stale awaiting group hold is advanced by
  the auto-confirm sweep; a `released`+`settled_at IS NULL` group hold (flag on) is
  re-driven by reconcile.
- `TestNoPhantomConsolidatedOrderHold` (Q1 regression) — deliver a group order; assert
  the consolidated order row stays `PayoutHoldNone` (never enters pending) while the
  group hold parks `awaiting_customer_confirmation`.
</action>
<verify><automated>cd apps/api && go vet ./services/ 2>&1 | grep -i group-order || echo "compile-fails-as-expected(RED)"</automated></verify>
<done>Tests exist and fail to compile (referencing symbols added in Tasks 2-7) — RED established.</done>

## Task 2: model columns + migration + flag-gate the three money seams
<files>apps/api/models/group_order.go, apps/api/migrations/20260703000003_add_group_order_payout_hold.up.sql, apps/api/migrations/20260703000003_add_group_order_payout_hold.down.sql, apps/api/services/group_order_payout.go</files>
<action>
Model (`models/group_order.go`), add to `GroupOrder` (mirror `models/order.go:169-178`):
`PayoutHoldStatus PayoutHoldStatus gorm:"type:varchar(32);not null;default:''"`,
`CustomerConfirmedAt *time.Time`, `DeliveredAt *time.Time`,
`PayoutSettledAt *time.Time`, `PayoutSettleAttempts int gorm:"default:0" json:"-"`.
GroupOrder is already registered in `database/database.go:225` AutoMigrate, so boot
creates the columns; the migration pair is the auditable production DDL (mirror the
header + `ADD COLUMN IF NOT EXISTS` style of
`migrations/20260703000001_add_payout_hold_state.up.sql`). up: add the five columns to
`group_orders`; down: drop them.

**CRITICAL LEAK FIX** — flag-gate the three money seams in `services/group_order_payout.go`
on `payoutMovementEnabled()` (defined `services/order_payout.go:32`). This single change
stops the live leak even before the decoupling lands:
- `HoldGroupChefPayout`: early-return `nil` when `!payoutMovementEnabled()`.
- `ReleaseGroupChefPayout`: early-return `nil` when `!payoutMovementEnabled()`.
- `ReverseGroupChefPayout`: early-return when `!payoutMovementEnabled()`.
Call this line out in the review — it is the P0 stop-the-bleed.
</action>
<verify><automated>cd apps/api && go build ./... && grep -c "payoutMovementEnabled" services/group_order_payout.go</automated></verify>
<done>Columns + migration exist; all three money seams no-op when the flag is off; build passes.</done>

## Task 3: decouple delivery + add SetGroupOrderHoldAwaitingConfirmation
<files>apps/api/services/payout_hold.go, apps/api/services/group_order_payout.go</files>
<action>
In `services/payout_hold.go` add `SetGroupOrderHoldAwaitingConfirmation(tx *gorm.DB,
groupOrderID uuid.UUID)` mirroring `SetOrderHoldAwaitingConfirmation`: conditional
UPDATE `WHERE id = ? AND payout_hold_status = PayoutHoldNone` → set
`awaiting_customer_confirmation`. Idempotent (only advances from the empty state).

Rewrite `MarkGroupOrderDelivered` (`services/group_order_payout.go:98`): inside the
existing guarded delivered transition (`WHERE id = ? AND status <> delivered`), when
`RowsAffected > 0`, STOP calling `ReleaseGroupChefPayout`. Instead, in the same tx:
`Update` set `status=delivered`, stamp `delivered_at = now`, and call
`SetGroupOrderHoldAwaitingConfirmation(tx, g.ID)`. Keep marking `GroupOrder.Status =
delivered` and any existing behavior. Keep <50 lines (extract a helper if needed).
The four callers (`handlers/delivery.go:612`, `handlers/chefs.go:1106`,
`services/provider.go:357`, `services/shadowfax_webhook.go:82`) are unchanged — they
already just call `MarkGroupOrderDelivered(orderID)`.
</action>
<verify><automated>cd apps/api && go build ./... && grep -q "SetGroupOrderHoldAwaitingConfirmation" services/group_order_payout.go && ! grep -q "ReleaseGroupChefPayout(&g)" services/group_order_payout.go && echo ok</automated></verify>
<done>Delivery parks the group hold + stamps delivered_at; no delivered path calls Release directly.</done>

## Task 4: ConfirmGroupOrderHold + aggType release/reverse wiring
<files>apps/api/services/payout_hold.go, apps/api/services/payout_release.go</files>
<action>
`services/payout_release.go`:
- Add const `aggTypeGroupOrder = "group-order"` beside `aggTypeOrder`/`aggTypeMealPlanDay`.
- `holdModel`: add `case aggTypeGroupOrder: return &models.GroupOrder{}, nil`.
- Add `listPendingGroupOrders(db, f)` (mirror `listPendingDays`): `Table("group_orders")`,
  select `id, chef_id, subtotal + tax AS amount, payout_hold_status, delivered_at,
  customer_confirmed_at, id AS context`, `WHERE payout_hold_status IN f.pendingStatuses()`,
  optional `chef_id` / `delivered_at < before`; map via `toPending(aggTypeGroupOrder, rows)`.
  Union it into `ListPendingPayouts` (append alongside orders + days before the sort).
- `releaseMoney`: add `case aggTypeGroupOrder` → load the group order, call
  `ReleaseGroupChefPayout(&g)` (already flag-gated in Task 2).
- `reverseMoney`: add `case aggTypeGroupOrder` → load, call `ReverseGroupChefPayout(&g)`.
`transitionHold`, `settleRelease`, `settleReverse`, `stampPayoutSettled`, `ReleaseHold`,
`WithholdHold`, `ReverseHold` are already aggType-generic — no change beyond the branches above.

`services/payout_hold.go`: add `ConfirmGroupOrderHold(db, g *models.GroupOrder)
(PayoutHoldStatus, error)` mirroring `ConfirmMealPlanDayHold`: idempotent on
`g.CustomerConfirmedAt != nil`; `disputed := g.OrderID != nil &&
HasOpenOrderIssue(db, *g.OrderID)`; wrap `applyHoldConfirm(tx, &models.GroupOrder{},
"group-order", g.ID, disputed, now)` in a tx; reload + copy back
`PayoutHoldStatus`/`CustomerConfirmedAt`.
</action>
<verify><automated>cd apps/api && go build ./... && grep -q "aggTypeGroupOrder" services/payout_release.go && grep -q "ConfirmGroupOrderHold" services/payout_hold.go && echo ok</automated></verify>
<done>Group holds list in the pending queue, release/reverse reach the flag-gated group seams, host-confirm advances awaiting→release_eligible / →disputed via the same guarded UPDATE.</done>

## Task 5: sweep + reconcile cover group orders
<files>apps/api/services/payout_auto_confirm_cron.go, apps/api/services/payout_reconcile_cron.go</files>
<action>
`payout_auto_confirm_cron.go`: add `sweepGroupOrders(cutoff)` mirroring `sweepMealPlanDays`
— select `group_orders` where `payout_hold_status = awaiting_customer_confirmation AND
delivered_at IS NOT NULL AND delivered_at <= cutoff AND customer_confirmed_at IS NULL`,
`Limit(sweepBatchLimit)`, call `ConfirmGroupOrderHold` per row (log-and-continue). Wire
its count into `runPayoutAutoConfirmScan`'s summary log. `recover()` already at scan top.

`payout_reconcile_cron.go`: add `reconcileGroupOrders(status, settle)` mirroring
`reconcileMealPlanDays` — select `group_orders` where `payout_hold_status = ? AND
payout_settled_at IS NULL AND payout_transfer_id <> '' AND payout_settle_attempts <
payoutReconcileMaxAttempts`, `driveSettles(aggTypeGroupOrder, ids, settle)`. Call it for
both `PayoutHoldReleased→settleRelease` and `PayoutHoldReversed→settleReverse` in
`runPayoutReconcileScan`. `bumpSettleAttempt` already resolves the model via `holdModel`
(handles group-order once Task 4's branch lands). Reconcile stays flag-gated by the
existing `!payoutMovementEnabled() && !MealPlanEscrowActive()` early-return.
</action>
<verify><automated>cd apps/api && go build ./... && grep -q "sweepGroupOrders" services/payout_auto_confirm_cron.go && grep -q "reconcileGroupOrders" services/payout_reconcile_cron.go && echo ok</automated></verify>
<done>Stale awaiting group holds auto-confirm; released/reversed-but-unsettled group holds re-drive through the same settle helpers.</done>

## Task 6: admin queue accepts "group-order"
<files>apps/api/handlers/admin_payout.go</files>
<action>
`parseAggType` (`handlers/admin_payout.go:35`): accept `"group-order"` alongside
`"order"` and `"meal-plan-day"`. Do the same in `BulkReleasePayouts`'s inline aggType
validation (`handlers/admin_payout.go:194`). No other handler change — release/withhold/
reverse already dispatch generically by aggType, and the tesserix-home admin release
queue already routes by aggType (a "Group order" row label + confirm UI are OUT OF SCOPE
follow-ups, per the ticket).
</action>
<verify><automated>cd apps/api && go build ./... && grep -c '"group-order"' handlers/admin_payout.go</automated></verify>
<done>Admin release/withhold/reverse (single + bulk) accept the group-order aggType; unknown types still 400.</done>

## Task 7: host confirm endpoint + route + guard cancel reverse
<files>apps/api/handlers/payout_hold.go, apps/api/handlers/group_order.go, apps/api/routes/routes.go</files>
<action>
`handlers/payout_hold.go`: add `ConfirmGroupOrderReceived(c *gin.Context)` — host-scoped.
Parse `:id`; load via the same `loadGroupForParticipant(id, userID)` +
`me.Role == models.GroupRoleHost` guard `CancelGroupOrder` uses
(`handlers/group_order.go:857-865`); 404 if not found, 403 if not host. Call
`services.ConfirmGroupOrderHold(database.DB, &g)`; respond with
`{payoutHoldStatus, customerConfirmedAt, message: confirmMessage(status)}` (reuse the
existing helper — no raw error codes to users, per repo rule). If
`loadGroupForParticipant` is unexported in the group_order handler package (same
`handlers` package — it is), call it directly.

`routes/routes.go`: register inside the existing `groupOrders` group
(`routes/routes.go:692-704`, already `bffAuth`-protected):
`groupOrders.POST("/:id/confirm-received", payoutHoldHandler.ConfirmGroupOrderReceived) // #456`.

Cancel guard: `CancelGroupOrder` calls `services.ReverseGroupChefPayout(&g)` at
`handlers/group_order.go:872` — now flag-gated by Task 2 (no-op while off), which is the
minimum P0 fix. Add a one-line comment noting the deeper "reverse runs outside the tx"
fix is a tracked follow-up (out of scope per the ticket); do NOT restructure the tx here.
</action>
<verify><automated>cd apps/api && go build ./... && grep -q "ConfirmGroupOrderReceived" routes/routes.go && echo ok</automated></verify>
<done>POST /group-orders/:id/confirm-received is host-scoped, idempotent, advances the group hold; cancel reverse is flag-gated.</done>

## Task 8 (GREEN + verify): run the suite
<files>apps/api/services/group_order_payout_test.go, apps/api/services/payout_hold_test.go, apps/api/services/payout_release_test.go</files>
<action>
Run the Task 1 tests — they must now compile and pass. Fix any drift. Confirm the Q1
regression test (`TestNoPhantomConsolidatedOrderHold`) passes: consolidated order stays
`PayoutHoldNone`, group hold advances. Confirm re-release yields `ErrHoldNotEligible`
(the 409 invariant). Run `go test -race ./services/... ./handlers/...`.
</action>
<verify><automated>cd apps/api && go test -race ./services/... ./handlers/... 2>&1 | tail -20</automated></verify>
<done>All new + existing payout/group tests pass under -race; no group-delivered path releases money ungated.</done>

</tasks>

<threat_model>
KEY INVARIANT (STRIDE — Tampering / Elevation of financial state): no group-delivered
code path may release chef money while the escrow flag is off, and the group hold must
obey the same conditional-transition invariants as order/meal-plan-day.

| Threat ID | Category | Component | Disposition | Mitigation |
|-----------|----------|-----------|-------------|------------|
| T-456-01 | Tampering (money) | `MarkGroupOrderDelivered` releasing ungated on delivery | mitigate | Flag-gate the 3 money seams (Task 2) + decouple delivery to park a hold (Task 3). Delivered no longer implies paid. |
| T-456-02 | Elevation | Non-host confirming a group order to force release_eligible | mitigate | Confirm endpoint host-scoped via `loadGroupForParticipant` + `GroupRoleHost` (Task 7). |
| T-456-03 | Tampering (race) | Concurrent/replayed confirm or release double-advancing a hold | mitigate | Reuse `applyHoldConfirm`/`transitionHold` guarded conditional UPDATEs + `RowsAffected` gate; disputed/released can never reach release_eligible; re-release → ErrHoldNotEligible/409. |
| T-456-04 | Repudiation | Bypassing dispute on a group order with an open issue | mitigate | `ConfirmGroupOrderHold` disputes via `HasOpenOrderIssue(*g.OrderID)` when the consolidated order is present. |
| T-456-05 | Tampering | Phantom double-hold (consolidated order + group hold) | accept/verify | Consolidated order has no razorpay_order_id → order hold is a genuine no-op; regression test locks it (Q1, Task 8). |
| T-456-06 | DoS | Cron sweep/reconcile aborting on one bad group row | mitigate | Bounded `sweepBatchLimit`, `recover()` at scan top, log-and-continue per row, attempt-capped reconcile (existing pattern reused). |
</threat_model>

<success_criteria>
- All three money seams in `group_order_payout.go` no-op when `payoutMovementEnabled()` is false (the P0 leak is closed).
- `MarkGroupOrderDelivered` parks `awaiting_customer_confirmation` + stamps `delivered_at`, releases nothing (flag on or off).
- Host confirm → `release_eligible`; open issue on the consolidated order → `disputed`.
- Admin `ReleaseHold("group-order", id)` releases (flag-gated) and 409s on re-release.
- Group orders appear in the pending queue and are covered by sweep + reconcile.
- No phantom consolidated-order hold (consolidated order stays `PayoutHoldNone`).
- `go test -race ./services/... ./handlers/...` green.
</success_criteria>

<output>
Quick-mode: no SUMMARY required. On completion, push `feat/group-order-payout-hold`
and open a PR to main referencing #456 (single-line commits, no signatures). Note the
out-of-scope follow-ups in the PR body: tesserix-home "Group order" row label + confirm
UI; the CancelGroupOrder reverse-outside-tx deep fix; #457/#458/#460/#461/#462.
</output>
