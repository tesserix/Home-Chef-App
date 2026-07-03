# Quick Task Summary: Gate chef "Mark Delivered" by fulfilment type (GH #391)

**Branch:** `fix/chef-delivered-fulfillment-gate`
**Requirement:** GH-391 (HIGH — self-payout theft on courier orders)

## What & why

A chef could mark ANY of their orders `Delivered` via `POST /chef/orders/:orderId/status` —
including 3PL/courier (`delivery`) orders they never delivered. Once escrow release is on, the
delivered transition fires `MarkMealPlanDayDelivered` / `MarkGroupOrderDelivered`, releasing the
held payout to the chef. This slice gates *who may trigger* the delivered transition; it does NOT
touch release semantics or the courier pipeline.

## Files changed

| File | Why |
|------|-----|
| `apps/api/handlers/chefs.go` | Pure gate helper `chefMayMarkDelivered(to, ft)`; wired as the single choke point in `UpdateOrderStatus` before `order.Status = newStatus`; `services.LogAudit` write on every chef-initiated delivered transition. |
| `apps/api/handlers/chef_delivered_gate_test.go` | New — pure gate table test + SQLite-backed HTTP tests for the six behaviours. |

**Untouched (by design):** `apps/api/handlers/delivery.go` courier path + Shadowfax/Borzo webhook.
Payout-release logic (#387 hold state machine) unchanged.

## Choke point & audit
- Gate choke point: `apps/api/handlers/chefs.go:1005` — `if !chefMayMarkDelivered(newStatus, order.FulfillmentType)`
  immediately after `chefCanTransition`, before `order.Status = newStatus` (line 1013). Blocked →
  403 `{"error":"delivery_confirmation_forbidden","message":...}`. Deny-by-default: only explicit
  `chef_delivery`/`pickup` pass; empty/unset (GORM default `delivery`) blocked.
- Audit action string: `"chef.order.delivered"` (entityType `order`, entityID = order id, newValue
  `{"fulfillmentType": ...}`), via existing `services.LogAudit` at `chefs.go:1110`. Actor captured from context.

## Tests
`cd apps/api && go build ./... && go test ./handlers/ ./services/ -count=1`
- build clean; `handlers` ok; `services` ok.
- New: TestChefMayMarkDelivered, TestUpdateOrderStatus_BlocksDeliveryOrder,
  _UnsetFulfillmentBlocked, _AllowsChefDelivery, _AllowsPickup, _WritesAuditOnAllowedDelivered — all pass.
- No tests dropped. Full DB-backed harness wired incl. audit-row + release-not-reached (blocked order stays `ready`) assertions.

## Deviations
None. TDD gates observed: RED (`test:`) → GREEN (`fix:`). Task 3 verification-only (no commit).

## Commits
| Task | Type | SHA | Message |
|------|------|-----|---------|
| 1 (RED) | test | 45d1c9bf | test: RED failing tests for chef delivered fulfilment gate (GH #391) |
| 2 (GREEN) | fix | 8c911b21 | fix: gate chef delivered transition by fulfilment type + audit (GH #391) |

## Follow-ups (out of scope)
- OTP proof-of-delivery for `chef_delivery` orders (backend + customer/vendor apps). Audit trail is interim control.
- #387 payout-hook decoupling — decouple delivered side-effects from payout release.
