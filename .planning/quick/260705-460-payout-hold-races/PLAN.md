# PLAN — #460 Payout-hold race conditions

**Issue:** [P1][payments] Payout-hold race conditions: chef `tx.Save()` lost-update + non-atomic dispute check.
**Epic:** #403. **Flag-gated:** hold columns are plain DB state; money only moves once escrow flags flip — so this is safe to land now, but it is a money-path change → plan → plan-check → independent verify.

## Problem (two races)

### Race 1 — full-row `Save(&order)` lost-update of hold columns
`handlers/chefs.go:UpdateOrderStatus` loads the whole `Order` row at handler start (incl. `payout_hold_status`, `customer_confirmed_at`), then persists with `tx.Save(&order)` (chefs.go:1072). Re-submitting `delivered` on an already-delivered order is an allowed idempotent no-op, so a retry that straddles a concurrent **customer-confirm / auto-confirm sweep / admin-release** commit writes the row's *load-time* hold columns back over the newer values → the hold reverts (e.g. `release_eligible` → `awaiting`). Full-row `Save` is the vector.

Other full-row `Save(&order)` sites on the orders table (audit): `delivery.go:503` + `delivery.go:1430` (PickedUp assignment) and `orders.go:832` (CancelOrder). These operate at lifecycle stages where the hold is provably empty (pickup precedes the delivered-park; cancel is only allowed from pending/accepted), so no live lost-update **today** — but they are the same anti-pattern and become a bug the moment a hold can exist earlier. Harden them too.

### Race 2 — dispute check not atomic with the transition
`ConfirmOrderHold` / `ConfirmMealPlanDayHold` / `ConfirmGroupOrderHold` read `HasOpenOrderIssue(db, …)` **outside** the tx, pass a `disputed bool` into `applyHoldConfirm`, which then does a separate guarded UPDATE. An `OrderIssue` filed in the check→UPDATE window still commits `release_eligible` (money escapes a just-filed dispute).

## Fix

### Race 1 — targeted `Updates` excluding hold columns
Replace each full-row `tx.Save(&order)` on the orders table with a targeted `tx.Model(&models.Order{}).Where("id = ?", order.ID).Updates(map[string]any{…})` writing **only** the columns that handler owns. Hold columns (`payout_hold_status`, `customer_confirmed_at`) and everything else are never in the map, so a concurrent transition survives. Bonus: `Updates(map)` skips GORM association upserts that `Save` can trigger.

- **chefs.go UpdateOrderStatus** (the live bug): map = `{status}` + `fulfillment_type` (only when a carrier was chosen) + the single lifecycle stamp for the transition (`accepted_at`/`prepared_at`/`picked_up_at`/`delivered_at`).
- **delivery.go:503 / :1430** (AcceptDelivery / ManualAssign): map = `{delivery_id, status, picked_up_at, estimated_delivery_time}`.
- **orders.go:832** (CancelOrder): map = `{status, cancelled_at, cancel_reason}`.

Column names verified against `models/order.go` (GORM snake_case). Each map enumerated against every `order.X =` mutation preceding the Save.

### Race 2 — fold the dispute predicate into the guarded UPDATE
Refactor `applyHoldConfirm` to take `issueOrderID *uuid.UUID` (the dispute source) instead of `disputed bool`, and evaluate issue-existence **inside** the UPDATE via `EXISTS` / `NOT EXISTS` subqueries. Run the **disputed (EXISTS) update first**, then the **release_eligible (NOT EXISTS, from `awaiting` only) update**. Ordering makes a dispute that races the confirm **fail safe**: if the issue lands after the disputed-update no-ops, the release-update's `NOT EXISTS` catches it and the row stays `awaiting` (money stays held) rather than releasing.

```
// 1) awaiting|disputed -> disputed   WHEN EXISTS(open issue on issueOrderID)
// 2) awaiting -> release_eligible    WHEN NOT EXISTS(open issue)   (skipped predicate when issueOrderID==nil)
```

Callers pass the source id: `ConfirmOrderHold → &order.ID`, `ConfirmMealPlanDayHold → day.OrderID`, `ConfirmGroupOrderHold → g.OrderID` (both already `*uuid.UUID`; nil = no dispute source → EXISTS update skipped, proceeds to release). Drop the three outside-tx `HasOpenOrderIssue` reads. Keep `HasOpenOrderIssue` (still used by `payout_release.go:282`).

Public signatures of the three `Confirm*Hold` funcs are unchanged → sweep cron (`payout_auto_confirm_cron.go`) and handlers (`handlers/payout_hold.go`) untouched.

## Tests (TDD)
- **Regression (must stay green):** all `payout_hold_test.go` cases (awaiting→release, open-issue→disputed, idempotent, disputed-stays-disputed, meal-plan-day) — the refactor must not change serial behavior.
- **NEW race-1 (chef_delivered_gate_test.go harness):** seed a delivered `chef_delivery` order; via a one-shot GORM `Before(gorm:update)` callback simulate a concurrent customer-confirm landing (`release_eligible` + `customer_confirmed_at`) just before the handler's persist; re-POST `delivered`; assert the hold columns are **preserved** (`release_eligible`). Fails on old `Save`, passes on targeted `Updates`.
- **NEW race-2 (payout_hold_test.go harness):** seed an `awaiting` order; via a one-shot callback insert a pending `OrderIssue` just before the confirm's transition UPDATE; call `ConfirmOrderHold`; assert result is **not** `release_eligible` (→ `disputed`) and **0** release-eligible events. Fails on old bool-precompute, passes on in-UPDATE predicate.

## Verification
- `go build ./...`, `go vet ./...`, `go test ./services/... ./handlers/...` (race-relevant packages), plus `go test ./...` for the api module.
- Independent verifier agent (fresh context) reviews the diff against the issue.

## Out of scope
- #458 disputed-hold resolution (separate issue). This fix only ensures a dispute *reaches* `disputed`; getting *out* of `disputed` is #458.
- W-A CancelGroupOrder reverse-outside-tx (separate, flag-gated).
