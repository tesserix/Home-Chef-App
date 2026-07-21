---
phase: quick
plan: 01
type: tdd
wave: 1
depends_on: []
autonomous: true
branch: fix/chef-delivered-fulfillment-gate
requirements: [GH-391]
files_modified:
  - apps/api/handlers/chefs.go
  - apps/api/handlers/chef_delivered_gate_test.go
must_haves:
  truths:
    - "A chef cannot mark a `delivery` (3PL) order Delivered — request is rejected 403"
    - "A chef CAN mark a `chef_delivery` order Delivered (200)"
    - "A chef CAN mark a `pickup` order Delivered (200)"
    - "An order with unset FulfillmentType is treated as `delivery` and blocked"
    - "Every chef-initiated Delivered transition writes an AuditLog row (actor, order id, fulfillmentType)"
    - "On a blocked transition, no release side-effect (MarkMealPlanDayDelivered / MarkGroupOrderDelivered) runs and the order stays not-delivered"
  artifacts:
    - path: "apps/api/handlers/chefs.go"
      provides: "chefMayMarkDelivered gate + audit write in UpdateOrderStatus"
      contains: "chefMayMarkDelivered"
    - path: "apps/api/handlers/chef_delivered_gate_test.go"
      provides: "RED tests for the fulfilment gate + audit"
      contains: "TestChefMayMarkDelivered"
  key_links:
    - from: "apps/api/handlers/chefs.go UpdateOrderStatus"
      to: "chefMayMarkDelivered"
      via: "guard clause before order.Status = newStatus"
      pattern: "chefMayMarkDelivered\\("
    - from: "apps/api/handlers/chefs.go UpdateOrderStatus (Delivered case)"
      to: "services.LogAudit"
      via: "audit write on chef-initiated delivered"
      pattern: "services\\.LogAudit\\("
---

<objective>
Close GH #391 (HIGH): a chef can mark ANY order `Delivered` — including 3PL/courier
(`delivery`) orders they never delivered — which (once escrow flags are on) releases
the held payout to themselves.

Fix: gate the chef `→ Delivered` transition by fulfilment type at a single choke point,
and audit-log every chef-initiated delivered transition.

Purpose: prevent self-payout theft on courier orders while preserving the legit
self-delivery (`chef_delivery`) and pickup (`pickup`) handoff paths.

Output:
- A pure, unit-tested gate helper `chefMayMarkDelivered` wired into `UpdateOrderStatus`.
- An audit row on every chef-initiated delivered transition, using the EXISTING
  `services.LogAudit` helper (no new audit machinery).

Branch: `fix/chef-delivered-fulfillment-gate`. Minimal, reviewable diff.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
</execution_context>

<context>
@CLAUDE.md

<grounding>
VERIFIED locations (cite these):

- `apps/api/handlers/chefs.go`
  - `chefOrderTransitions` map (line ~910): `Ready`/`PickedUp`/`Delivering → Delivered`
    allowed for ANY order.
  - `chefCanTransition(from, to)` (line 919): checks only from→to, NOT fulfilment type.
  - `UpdateOrderStatus` (line ~929): transition validated at line 978
    (`if newStatus != priorStatus && !chefCanTransition(...)`) — `order.Status = newStatus`
    is set at line 982. Delivered side-effects fire later at lines 1066-1075
    (`SignalOrderDelivered`, `MarkMealPlanDayDelivered`, `MarkGroupOrderDelivered`).
  - Actor id is available: handler already calls `middleware.GetUserID(c)` (context key `"userID"`).

- `apps/api/models/order.go:57-63`
  - `FulfillmentDelivery = "delivery"` (3PL rider, GORM default `'delivery'`) — chef must NOT mark delivered.
  - `FulfillmentChefDelivery = "chef_delivery"` — chef MAY.
  - `FulfillmentPickup = "pickup"` — chef MAY.
  - Column default is `'delivery'`; treat empty/unset FulfillmentType as `delivery` (blocked) for safety.

- Legit courier path that stays the ONLY way a `delivery` order reaches Delivered:
  `apps/api/handlers/delivery.go:603-609` (delivery pipeline) + Shadowfax/Borzo webhook.
  DO NOT touch these.

- EXISTING audit write helper (USE THIS — do not invent):
  `apps/api/services/audit.go:23` `func LogAudit(c *gin.Context, action, entityType, entityID string, oldValue, newValue any)`.
  Non-blocking, records actor from `c.Get("userID")`, marshals newValue to JSON.
  Model: `apps/api/models/admin.go:18` `type AuditLog struct{ UserID *uuid.UUID; Action; EntityType; EntityID; NewValue ... }`.

- Test harness to match: `apps/api/handlers/meal_plan_booking_test.go:30-90` — in-memory
  sqlite (`gorm.Open(sqlite.Open(":memory:"))`), raw `CREATE TABLE`, swap `database.DB`,
  restore in `t.Cleanup`; router with `r.Use(func(c){ c.Set("userID", userID) })`.
</grounding>

<choke_point_decision>
Single choke point = immediately AFTER the `chefCanTransition` guard (chefs.go line 981),
BEFORE `order.Status = newStatus` (line 982). Every path that later reaches
`MarkMealPlanDayDelivered` / `MarkGroupOrderDelivered` (lines 1074-1075) passes through
this line, so nothing bypasses the gate. Do NOT scatter the check into the side-effect
switch.
</choke_point_decision>
</context>

<tasks>

<task type="tdd" tdd="true">
  <name>Task 1 (RED): Write failing tests for the fulfilment gate + audit</name>
  <files>apps/api/handlers/chef_delivered_gate_test.go</files>
  <behavior>
    Pure gate helper `chefMayMarkDelivered(to models.OrderStatus, ft models.FulfillmentType) bool`
    (decision only — no DB), table test `TestChefMayMarkDelivered`:
      - to=Delivered, ft="delivery"        → false (blocked)
      - to=Delivered, ft="" (unset)        → false (normalizes to delivery, blocked)
      - to=Delivered, ft="chef_delivery"   → true
      - to=Delivered, ft="pickup"          → true
      - to=Ready,     ft="delivery"        → true (gate only applies to Delivered)
    Non-Delivered transitions are never affected by the gate.

    DB-backed HTTP tests (sqlite harness mirroring meal_plan_booking_test.go), on the
    real `UpdateOrderStatus` handler, seed chef_profiles + orders (+ audit_logs, meal_plans/
    group tables only if the delivered side-effects touch them — otherwise a blocked-path
    order never reaches them):
      - TestUpdateOrderStatus_BlocksDeliveryOrder: order status=ready, fulfillment_type='delivery',
        POST status=delivered → 403 (or 422), structured JSON `{"error":...,"message":...}`;
        assert order row status is STILL 'ready' (release path not reached).
      - TestUpdateOrderStatus_AllowsChefDelivery: fulfillment_type='chef_delivery' → 200, order 'delivered'.
      - TestUpdateOrderStatus_AllowsPickup: fulfillment_type='pickup' → 200, order 'delivered'.
      - TestUpdateOrderStatus_UnsetFulfillmentBlocked: fulfillment_type='' → 403, order stays 'ready'.
      - TestUpdateOrderStatus_WritesAuditOnAllowedDelivered: chef_delivery → 200 AND an
        audit_logs row exists with the chef user id, entity_id = order id, and newValue
        containing the fulfillmentType.
  </behavior>
  <action>
    Create `apps/api/handlers/chef_delivered_gate_test.go` (package `handlers`).
    Copy the sqlite harness shape from `meal_plan_booking_test.go` (in-memory DB, raw
    CREATE TABLE for chef_profiles, orders, audit_logs; swap `database.DB`; `t.Cleanup`
    restore). Router uses `r.Use(func(c *gin.Context){ c.Set("userID", chefUserID); c.Next() })`
    and `r.POST("/chef/orders/:orderId/status", (&ChefHandler{}).UpdateOrderStatus)`.
    Keep the orders CREATE TABLE minimal — include the columns UpdateOrderStatus reads/writes
    (id, order_number, chef_id, customer_id, status, fulfillment_type, timestamps). If the
    handler's delivered side-effects require additional tables to no-op cleanly, add those
    tables empty so `MarkMealPlanDayDelivered`/`MarkGroupOrderDelivered` return no-op.
    Reference the audit column names from `models/admin.go` (user_id, action, entity_type,
    entity_id, new_value, created_at).
    Write ONLY tests here. They MUST fail to compile/pass now (helper + gate don't exist).
  </action>
  <verify>
    <automated>cd apps/api && go test ./handlers/ -run 'ChefMayMarkDelivered|UpdateOrderStatus_(Blocks|Allows|Unset|WritesAudit)' 2>&1 | grep -E 'undefined|FAIL|cannot' </automated>
  </verify>
  <done>Test file compiles against a stubbed helper OR fails as RED (undefined chefMayMarkDelivered / assertions fail). Tests express all six behaviors above.</done>
</task>

<task type="tdd" tdd="true">
  <name>Task 2 (GREEN): Add fulfilment gate + audit write to UpdateOrderStatus</name>
  <files>apps/api/handlers/chefs.go</files>
  <action>
    1. Add a pure helper near `chefCanTransition` (after line 926):

       ```go
       // chefMayMarkDelivered reports whether a chef is permitted to move an order
       // to Delivered given its fulfilment type. 3PL `delivery` orders reach Delivered
       // ONLY via the courier pipeline / webhook (handlers/delivery.go) — a chef marking
       // one delivered would release the held payout to themselves without delivering it
       // (GH #391). chef_delivery and pickup are legit chef-side handoffs. Empty/unset
       // fulfilment normalizes to delivery (GORM default) and is blocked for safety.
       func chefMayMarkDelivered(to models.OrderStatus, ft models.FulfillmentType) bool {
           if to != models.OrderStatusDelivered {
               return true // gate applies only to the Delivered transition
           }
           return ft == models.FulfillmentChefDelivery || ft == models.FulfillmentPickup
       }
       ```

    2. Wire the guard as the choke point — immediately AFTER the `chefCanTransition`
       block (after line 981), BEFORE `order.Status = newStatus`:

       ```go
       if !chefMayMarkDelivered(newStatus, order.FulfillmentType) {
           c.JSON(http.StatusForbidden, gin.H{
               "error":   "delivery_confirmation_forbidden",
               "message": "This order is delivered by a courier — it can only be marked delivered by the delivery service.",
           })
           return
       }
       ```

    3. Audit every chef-initiated delivered transition. In the existing
       `case models.OrderStatusDelivered:` block (lines 1066-1075), after the release
       side-effects, add:

       ```go
       services.LogAudit(c, "chef.order.delivered", "order", order.ID.String(), nil,
           map[string]any{"fulfillmentType": string(order.FulfillmentType)})
       ```

       `LogAudit` reads the actor from `c.Get("userID")` (already set), so the chef user id
       is captured. Do NOT change any release logic (#387 owns the hold state machine).

    Keep functions <50 lines, early-return, no panics, structured JSON error envelope.
  </action>
  <verify>
    <automated>cd apps/api && go build ./... && go test ./handlers/ -run 'ChefMayMarkDelivered|UpdateOrderStatus_(Blocks|Allows|Unset|WritesAudit)' -count=1</automated>
  </verify>
  <done>All six behaviors pass. Blocked `delivery`/unset returns 403 and order stays not-delivered; chef_delivery/pickup return 200; audit row written on allowed delivered transition. `go build ./...` clean.</done>
</task>

<task type="auto">
  <name>Task 3: Full-suite regression + confirm courier path untouched</name>
  <files>apps/api/handlers/chefs.go, apps/api/handlers/chef_delivered_gate_test.go</files>
  <action>
    Run the handler package test suite to confirm no regression in existing order/chef flows.
    Grep-confirm the courier delivered path (`apps/api/handlers/delivery.go:603-609`) and the
    Shadowfax/Borzo webhook were NOT modified — a `delivery` order must still reach Delivered
    via that pipeline. Confirm the diff is limited to the two files above and the branch is
    `fix/chef-delivered-fulfillment-gate`.
  </action>
  <verify>
    <automated>cd apps/api && go test ./handlers/ -count=1 && git diff --name-only main -- apps/api | grep -vE 'handlers/chefs.go|handlers/chef_delivered_gate_test.go' | grep -c . | grep -qx 0 && echo DIFF_SCOPED_OK</automated>
  </verify>
  <done>Handler suite green; diff limited to chefs.go + the new test file; delivery.go / webhook unchanged; on branch fix/chef-delivered-fulfillment-gate.</done>
</task>

</tasks>

<threat_model>
## Trust Boundary
Chef (authenticated vendor) → chef order-status API (`POST /chef/orders/:orderId/status`).
The chef is authenticated but is NOT trusted to assert physical delivery of a courier order.

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-391-01 | Elevation of Privilege | `UpdateOrderStatus` chef→Delivered on `delivery` order | mitigate | `chefMayMarkDelivered` guard rejects 403 at the single choke point before `order.Status` is set — a chef cannot trigger `MarkMealPlanDayDelivered`/`MarkGroupOrderDelivered` (payout release) on a 3PL order. |
| T-391-02 | Repudiation | Chef marks own order delivered, later denies it | mitigate | `services.LogAudit("chef.order.delivered", ...)` records actor user id, order id, fulfillmentType on every chef-initiated delivered transition. |
| T-391-03 | Tampering (bypass) | Unset/empty FulfillmentType slips through as "not delivery" | mitigate | Gate treats only explicit `chef_delivery`/`pickup` as allowed; empty/unset (GORM default `delivery`) is blocked by default (deny-by-default). |
| T-391-04 | Spoofing | Chef marks ANOTHER chef's order delivered | accept (pre-mitigated) | Existing ownership scope: order loaded with `WHERE id = ? AND chef_id = ?` (chefs.go). Unchanged by this fix. |
| T-391-05 | EoP (residual) | Self-delivery (`chef_delivery`) chef marks delivered without proof | accept — FOLLOW-UP | OTP proof-of-delivery is OUT OF SCOPE (separate slice: backend OTP issue/verify + customer/vendor app). Audit trail (T-391-02) is the interim control. |

## Explicitly OUT OF SCOPE (follow-up slices — do NOT implement here)
- OTP proof-of-delivery for `chef_delivery` orders (backend + customer app + vendor app).
- Decoupling delivery hooks from payout release — owned by #387's hold state machine.
- No changes to release logic; this plan only gates WHO can trigger the Delivered transition.
</threat_model>

<verification>
- `cd apps/api && go build ./...` clean.
- `cd apps/api && go test ./handlers/ -count=1` green.
- Blocked path (`delivery` / unset) → 403 structured JSON, order stays not-delivered, no release side-effect reached.
- Allowed paths (`chef_delivery`, `pickup`) → 200, order delivered, audit row written.
- Diff limited to `apps/api/handlers/chefs.go` + `apps/api/handlers/chef_delivered_gate_test.go`.
- `apps/api/handlers/delivery.go` courier path + webhook unchanged.
</verification>

<success_criteria>
- Chef cannot mark a `delivery` (3PL) order Delivered (403); `chef_delivery` and `pickup` still work.
- Unset FulfillmentType is blocked (deny-by-default).
- Every chef-initiated Delivered transition is audit-logged with actor, order id, fulfillmentType.
- No change to payout-release logic or the courier delivered pipeline.
- Minimal, reviewable diff on branch `fix/chef-delivered-fulfillment-gate`.
</success_criteria>

<output>
After completion, create `.planning/quick/260703-fqs-gate-chef-mark-delivered-to-self-deliver/SUMMARY.md`
noting: gate choke point line, audit action string, and the OTP + #387 decoupling follow-ups.
</output>
