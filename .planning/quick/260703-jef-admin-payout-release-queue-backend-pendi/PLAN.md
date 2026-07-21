---
mode: quick
issue: "GH #388 — admin payout release queue (backend / actuator)"
branch: feat/payout-admin-release-queue
scope: apps/api (Go) only — tesserix-home UI is a separate slice
depends_on: "#387 hold state machine (on main)"
flags_default_off: [OrderPayoutAutoReleaseEnabled, MealPlanEscrowEnabled]
---

<objective>
Build the backend actuator that turns `release_eligible` payout holds into real
vendor payouts. Admin-only endpoints to **list** release-eligible holds and
**release / withhold / reverse** them (single + bulk), advancing the #387 hold
state machine and driving the existing Razorpay `ReleaseTransfer` /
`ReverseTransfer` seams.

Money movement stays behind the existing escrow flags: when
`OrderPayoutAutoReleaseEnabled` (regular orders) and `MealPlanEscrowEnabled`
(meal-plan days) are OFF, every action is a **DB-only state advance with no money
moved**. Manual-first launch: no auto-approve sweep, just the getter.

Purpose: close the loop on #387 — chef payouts held on delivery can now actually
be released by an admin, race-safely and audited.
Output: enum values, doc migration, `services/payout_release.go`,
`handlers/admin_payout.go`, four admin routes, TDD coverage.
</objective>

<grounding>
Re-verified against apps/api on 2026-07-03. Cited locations:

- **Hold enum:** `apps/api/models/payout_hold.go` — `PayoutHoldStatus` (varchar(32)):
  `PayoutHoldNone ""`, `PayoutHoldAwaitingConfirmation "awaiting_customer_confirmation"`,
  `PayoutHoldReleaseEligible "release_eligible"`, `PayoutHoldReleased "released"`,
  `PayoutHoldDisputed "disputed"`. **No `withheld`/`reversed`.**
- **Hold fields:** `models/order.go:168-169` (`PayoutHoldStatus`, `CustomerConfirmedAt`),
  plus `Total float64` (:96), `ChefID` (:70), `RazorpayOrderID` (:155), `DeliveredAt` (:132).
  `models/meal_plan.go:127-128` (`PayoutHoldStatus`, `CustomerConfirmedAt`), plus
  `Price float64` (:113), `OrderID *uuid.UUID` (:116), `PayoutTransferID` (:117),
  `DeliveredAt` (:121); chef via `meal_plans.chef_id`.
- **Order seams (gated `payoutMovementEnabled()` = `OrderPayoutAutoReleaseEnabled`):**
  `services/order_payout.go:47 ReleaseOrderPayouts(orderID uuid.UUID) error`,
  `:83 ReverseOrderPayouts(orderID uuid.UUID) error`. Both no-op when flag OFF,
  when order has no razorpay id, or when `GetRazorpay()==nil`.
- **Meal-plan-day seam (gated `MealPlanEscrowActive()` = `MealPlanEscrowEnabled`):**
  `services/meal_plan_escrow.go:183 ReleaseDayPayout(tx *gorm.DB, day *models.MealPlanDay) error`
  (DB-guarded on `day.PayoutTransferID != ""`, no-op when escrow off).
  **No reverse-day seam** — `RefundDay` reverses AND credits wallet, wrong here.
  Use `GetRazorpay().ReverseTransfer(day.PayoutTransferID, 0)` (`services/razorpay.go:449`),
  guarded on `MealPlanEscrowActive() && day.PayoutTransferID != "" && GetRazorpay()!=nil`.
- **Razorpay client:** `ReleaseTransfer(id)` (razorpay.go:416), `ReverseTransfer(id, amountPaise)`
  (:449, amountPaise 0 = full).
- **Conditional-update race pattern:** `services/payout_hold.go:82 applyHoldConfirm` —
  `tx.Model(m).Where("id=? AND payout_hold_status=?", ...).Updates(...)` then
  `res.RowsAffected==0 → no genuine transition`. Reuse this exact style.
- **Outbox event:** `services/outbox.go:68 EnqueueEvent(tx, subject, eventType, userID, data)`;
  subjects in `services/nats.go:39-40` (`SubjectHoldReleaseEligible`, `SubjectHoldDisputed`).
  Add `SubjectHoldReleased = "payments.hold_released"`.
- **Settings getter pattern:** `services/payout_hold.go:204 GetCustomerConfirmWindowHours(db)`
  (`Where("key LIKE ?", "payout.%")`, fold, default). Copy for auto-approve knob.
- **Audit:** `services/audit.go:23 LogAudit(c, action, entityType, entityID, oldValue, newValue)`.
- **Admin route group:** `routes/routes.go:788` —
  `admin := v1.Group("/admin"); admin.Use(bffAuth(bffKey, bffWindow), middleware.RequireAdmin())`.
  Existing handlers on `adminHandler`. Add a new `adminPayoutHandler`.
- **Migration convention:** `apps/api/migrations/<ts>_name.up.sql/.down.sql`; latest
  `20260703000001_add_payout_hold_state.*` — columns are actually created by
  `DB.AutoMigrate` (database/database.go), the SQL pair is auditable DDL only.
- **Test harness:** `services/payout_hold_test.go setupHoldDB` (in-memory sqlite,
  hand-DDL: orders, meal_plan_days, meal_plans, order_issues, platform_settings,
  outbox_events). Handler harness pattern: `handlers/chef_delivered_gate_test.go`
  (swap `database.DB`, gin httptest, restore in `t.Cleanup`).

**Design decisions (grounded):**
- **No new columns.** Status stays `varchar(32)`; new enum values need no DDL. The
  withhold/reverse **reason** is recorded in the audit `newValue` payload (and the
  outbox event data), not a new column — keeps the diff minimal and matches
  "columns already exist". Migration pair is doc-only (mirrors #387).
- **Order of operations (race-safe):** conditional status update happens FIRST in a
  short tx (the race guard — only one actor's `RowsAffected` is 1); the money seam
  is dispatched AFTER commit. Seams are idempotent (Razorpay release/reverse of an
  already-actioned transfer is a no-op / logged) and are hard no-ops while flags are
  OFF, so post-commit dispatch is safe. Meal-plan-day money seam is called with
  `database.DB` as its `tx` arg (it accepts `*gorm.DB`, DB-guarded internally).
- **Amount shown** = `Order.Total` / `MealPlanDay.Price` (gross the customer paid);
  the actual Razorpay transfer split is gateway-side. Queue DTO is for review, not
  reconciliation.
</grounding>

<tasks>

## Task 1 (RED): Failing service + handler tests

<files>
apps/api/services/payout_release_test.go (new)
apps/api/handlers/admin_payout_test.go (new)
</files>

<action>
Write the tests first against the intended API (they will not compile until Tasks
2-4 land — that is the RED state).

**Service tests** (`payout_release_test.go`) — clone `setupHoldDB` from
`services/payout_hold_test.go` (add nothing new; the 6 tables already cover this).
Set `config.AppConfig = &config.Config{...}` per case and restore in `t.Cleanup`.
Seed rows with helper inserts (mirror `seedRegularOrder`; add a meal-plan-day seed
with `payout_transfer_id`, `price`, `payout_hold_status`).

- `TestListPendingPayouts_ReturnsEligibleOnly`: seed orders + days in
  `release_eligible`, `awaiting_customer_confirmation`, `disputed`, `withheld`.
  `ListPendingPayouts(db, filters{})` returns only the two `release_eligible` rows
  (order + day) with Amount, AgeHours (from DeliveredAt), HoldStatus, aggType.
  With `filters{IncludeAwaiting: true}` it also returns the awaiting rows; never
  disputed/withheld/reversed/released.
- `TestReleaseHold_FlagOff_AdvancesNoMoney`: flag OFF, order `release_eligible` →
  `ReleaseHold(db,"order",id)` returns nil, row is `released`, a
  `payments.hold_released` outbox row is staged (`countOutbox`). No panic (rz nil).
- `TestReleaseHold_FlagOn_ReachesSeam_NoCrash`: `OrderPayoutAutoReleaseEnabled=true`,
  `GetRazorpay()==nil` (unconfigured) → `ReleaseOrderPayouts` no-ops cleanly; row
  still `released`. Proves the seam is wired without a live gateway.
- `TestReleaseHold_ReRelease_NotEligible`: second `ReleaseHold` on an
  already-`released` row returns `ErrHoldNotEligible` (conditional-update no-op),
  no second outbox row.
- `TestWithholdHold_EligibleToWithheld`: `release_eligible` →
  `WithholdHold(db,"order",id,"suspected fraud")` sets `withheld`; row excluded
  from a subsequent `ListPendingPayouts`.
- `TestReverseHold_ReleasedToReversed`: `released` →
  `ReverseHold(db,"order",id,"chargeback")` sets `reversed`.
- `TestReverseHold_MealPlanDay`: seed day `release_eligible` with transfer id;
  `ReverseHold(db,"meal-plan-day",id,"bad food")` sets `reversed` (flag OFF → no money).
- `TestGetPayoutAutoApproveHours_DefaultZero`: no setting → 0; with
  `payout.auto_approve_after_hours=48` → 48.

**Handler tests** (`admin_payout_test.go`) — mirror `chef_delivered_gate_test.go`:
in-memory sqlite, hand-DDL the same tables **plus an `audit_logs` table**
(`id TEXT PRIMARY KEY, user_id TEXT, action TEXT, entity_type TEXT, entity_id TEXT,
old_value TEXT, new_value TEXT, ip_address TEXT, correlation_id TEXT, created_at DATETIME`),
swap `database.DB`, restore in `t.Cleanup`. Build a gin router with the handler
methods bound directly (no bffAuth/RequireAdmin in-test — those are asserted via a
separate route-wiring check, see Task 4 verify).

- `TestGetPendingPayouts_200`: seeded eligible rows → `GET` returns JSON list.
- `TestReleasePayout_Conflict`: already-`released` row → `POST .../release` → 409.
- `TestWithholdPayout_MissingReason_400`: empty reason → 400.
- `TestReversePayout_MissingReason_400`: empty reason → 400.
- `TestReleasePayout_WritesAudit`: successful release writes one `audit_logs` row
  with action `payout.released`.
- `TestBulkRelease_SkipsIneligible`: mix of eligible + already-released ids →
  response reports released count + skipped list; only eligible advanced.
</action>

<verify>
`cd apps/api && go test ./services/ ./handlers/ -run 'PayoutRelease|Payout' 2>&1`
compiles-fails or fails on missing symbols (RED confirmed). Do NOT proceed until
the tests exist and express every acceptance check above.
</verify>

<done>Both test files exist; running them fails only because production code is absent (RED).</done>

---

## Task 2: Enum values + doc migration + event subject

<files>
apps/api/models/payout_hold.go (edit — add 2 enum consts)
apps/api/services/nats.go (edit — add SubjectHoldReleased)
apps/api/migrations/20260703000002_add_payout_withheld_reversed_states.up.sql (new)
apps/api/migrations/20260703000002_add_payout_withheld_reversed_states.down.sql (new)
</files>

<action>
1. In `models/payout_hold.go`, append to the const block, with doc comments
   matching the file's style:
   - `PayoutHoldWithheld PayoutHoldStatus = "withheld"` — admin blocked the payout
     (open concern / suspected fraud); terminal, no money moved.
   - `PayoutHoldReversed PayoutHoldStatus = "reversed"` — admin clawed a
     released/eligible payout back to the platform (refund/chargeback); terminal.
2. In `services/nats.go` add `SubjectHoldReleased = "payments.hold_released"` next
   to the existing hold subjects (line ~39), with a trailing comment.
3. Migration pair — **doc-only** (mirror the #387 migration prose): the values are
   plain `varchar(32)` strings, so there is genuinely no DDL. The `.up.sql` is a
   comment block explaining that `withheld`/`reversed` are new
   `payout_hold_status` values written by the admin payout queue (#388), that no
   column/type change is required (varchar, not a PG enum), and that no new columns
   are added (the withhold/reverse reason is captured in `audit_logs`). Include a
   harmless idempotent no-op statement (`SELECT 1;`) so the file is valid SQL for
   the migrate tooling. `.down.sql`: comment noting rollback is data-only (no schema
   to revert) + `SELECT 1;`.
</action>

<verify>
`cd apps/api && go build ./... 2>&1` clean. `grep -n 'PayoutHoldWithheld\|PayoutHoldReversed' models/payout_hold.go` shows both. `grep -n 'SubjectHoldReleased' services/nats.go` present. Both migration files exist and are non-empty.
</verify>

<done>Enum values, event subject, and doc migration pair committed; build green.</done>

---

## Task 3: Service layer — `services/payout_release.go`

<files>
apps/api/services/payout_release.go (new)
</files>

<action>
Package `services`. Funcs <50 lines, wrap with `%w`, no globals beyond
`database`/`config`/`GetRazorpay` already used package-wide.

Define:
```
var ErrHoldNotEligible = errors.New("payout hold not in an eligible state")
```

**Unified DTO + filters:**
```
type PendingPayout struct {
    AggType             string    // "order" | "meal-plan-day"
    ID                  uuid.UUID
    ChefID              uuid.UUID
    Amount              float64   // Order.Total / MealPlanDay.Price
    HoldStatus          models.PayoutHoldStatus
    DeliveredAt         *time.Time
    AgeHours            float64   // now - DeliveredAt (SLA clock vs 24h)
    CustomerConfirmedAt *time.Time
    Context             string    // order number / meal-plan number for the queue row
}
type PendingFilter struct {
    IncludeAwaiting bool      // ?include=awaiting → also surface awaiting_customer_confirmation
    ChefID          uuid.UUID // optional
    Before          *time.Time
}
```

`ListPendingPayouts(db *gorm.DB, f PendingFilter) ([]PendingPayout, error)`:
- Statuses = `{release_eligible}`, plus `awaiting_customer_confirmation` when
  `f.IncludeAwaiting`. Never disputed/withheld/reversed/released.
- Query orders (`payout_hold_status IN ?`, optional `chef_id`, optional
  `delivered_at < ?`) → map to DTO (Amount=Total, Context=order number).
- Query meal_plan_days joined to meal_plans for `chef_id`/plan number → map
  (Amount=Price). Merge, sort by AgeHours desc. Compute AgeHours from DeliveredAt.

**Conditional transition helper** (mirror `applyHoldConfirm`):
```
func transitionHold(db, aggType string, id, from []PayoutHoldStatus,
    to PayoutHoldStatus, extra map[string]any) (bool, error)
```
- Pick model by aggType (`&models.Order{}` / `&models.MealPlanDay{}`; unknown
  aggType → error).
- In `db.Transaction`: `tx.Model(m).Where("id = ? AND payout_hold_status IN ?", id, from)
  .Updates(merge({"payout_hold_status": to}, extra))`. If `RowsAffected == 0` →
  return `(false, nil)`. For the release case, stage the event
  (`EnqueueEvent(tx, SubjectHoldReleased, "payout.hold_released", id, {aggregate_type,
  aggregate_id, payout_hold_status})`) inside the same tx. Return `(true, nil)`.

**`ReleaseHold(db, aggType string, id uuid.UUID) error`:**
- `ok := transitionHold(release_eligible → released, emit hold_released)`;
  `if !ok → return ErrHoldNotEligible`.
- Post-commit money seam: `order` → `ReleaseOrderPayouts(id)`; `meal-plan-day` →
  load the day, `ReleaseDayPayout(database.DB, &day)`. Wrap seam errors with `%w`.

**`WithholdHold(db, aggType string, id uuid.UUID, reason string) error`:**
- `reason` must be non-empty (caller/handler validates; guard here too →
  `fmt.Errorf("withhold reason required")`).
- `transitionHold({release_eligible, awaiting_customer_confirmation} → withheld)`;
  `!ok → ErrHoldNotEligible`. No event, no money. (Reason travels to audit at the
  handler.)

**`ReverseHold(db, aggType string, id uuid.UUID, reason string) error`:**
- Require non-empty reason.
- `transitionHold({released, release_eligible} → reversed)`; `!ok → ErrHoldNotEligible`.
- Post-commit money seam: `order` → `ReverseOrderPayouts(id)`; `meal-plan-day` →
  load day, if `MealPlanEscrowActive() && day.PayoutTransferID != "" && GetRazorpay()!=nil`
  → `GetRazorpay().ReverseTransfer(day.PayoutTransferID, 0)` (0 = full).

**`GetPayoutAutoApproveHours(db *gorm.DB) int`:** copy `GetCustomerConfirmWindowHours`
exactly but key `payout.auto_approve_after_hours`, **default 0** (disabled =
manual-first). Add a doc comment: consumed by a follow-up auto-approve sweep (not
wired in this slice).
</action>

<verify>
`cd apps/api && go build ./... && go test ./services/ -run PayoutRelease 2>&1` — all service tests GREEN.
</verify>

<done>Service compiles; every service test from Task 1 passes.</done>

---

## Task 4: Handlers + routes — `handlers/admin_payout.go`

<files>
apps/api/handlers/admin_payout.go (new)
apps/api/routes/routes.go (edit — construct handler + add 4 routes in admin group)
</files>

<action>
`AdminPayoutHandler` struct + `NewAdminPayoutHandler()` (factory singleton pattern
like the other handlers). Methods parse/validate → call service → structured JSON
error envelope `{"error": code, "message": ...}`. No panics.

Shared helpers:
- `parseAggType(c) (string, bool)` — accept only `order` | `meal-plan-day`; else
  400 `invalid_agg_type`.
- Map service errors: `ErrHoldNotEligible` → **409** `hold_not_eligible`; other
  errors → 500 `payout_action_failed`.
- Reason binding: `type reasonReq struct { Reason string \`json:"reason"\` }`;
  empty (trimmed) → 400 `reason_required` for withhold/reverse.

Methods:
- `GetPendingPayouts(c)` — build `PendingFilter` from query
  (`include=awaiting`, `chefId`, `before`) → `ListPendingPayouts` → 200 list.
- `ReleasePayout(c)` — aggType+`:id` → `ReleaseHold`; on success
  `LogAudit(c, "payout.released", aggType, id, oldStatus, gin.H{"status":"released"})`
  → 200.
- `WithholdPayout(c)` — bind reason → `WithholdHold`; audit `payout.withheld`
  with `gin.H{"status":"withheld","reason":reason}` → 200.
- `ReversePayout(c)` — bind reason → `ReverseHold`; audit `payout.reversed`
  with `gin.H{"status":"reversed","reason":reason}` → 200.
- `BulkReleasePayouts(c)` — body `{ items: [{aggType,id}], before?: ts }` (if
  `before` given and items empty, resolve eligible ids via `ListPendingPayouts`).
  Loop `ReleaseHold`; collect `released []` and `skipped [{id,reason}]`
  (`ErrHoldNotEligible` → skipped, not fatal). One summary audit
  `payout.bulk_released`. Return `{released: n, releasedIds, skipped}`.

**Routes** — in `routes/routes.go` inside the existing `admin` group
(after wallet/review lines, ~line 810), construct
`adminPayoutHandler := handlers.NewAdminPayoutHandler()` where the other admin
handlers are built, then:
```
admin.GET("/payouts/pending", adminPayoutHandler.GetPendingPayouts)
admin.POST("/payouts/:aggType/:id/release", adminPayoutHandler.ReleasePayout)
admin.POST("/payouts/:aggType/:id/withhold", adminPayoutHandler.WithholdPayout)
admin.POST("/payouts/:aggType/:id/reverse", adminPayoutHandler.ReversePayout)
admin.POST("/payouts/release-bulk", adminPayoutHandler.BulkReleasePayouts)
```
The group already applies `bffAuth(...) + middleware.RequireAdmin()`, so admin-only
is inherited — do not re-add per-route.
</action>

<verify>
`cd apps/api && go build ./... && go test ./handlers/ -run Payout 2>&1` GREEN.
Route wiring: `grep -n '/payouts/' routes/routes.go` shows all 5 routes inside the
admin group; confirm they sit between the `admin := v1.Group("/admin")` line and
its closing `}` (i.e. under `RequireAdmin`).
</verify>

<done>Handlers + routes compile; handler tests pass; routes live under the admin/RequireAdmin group.</done>

---

## Task 5: Full verify + tidy

<files>(no new files)</files>

<action>
Run the whole api test suite for regressions, `go vet`, and `gofmt`. Confirm no
function exceeds ~50 lines (split helpers if so). Confirm no money-moving call
exists outside the guarded seams. Update the GH #388 issue with a comment: what
shipped (list/release/withhold/reverse + bulk), flags-OFF = DB-only, and the
OUT-OF-SCOPE follow-ups below.
</action>

<verify>
`cd apps/api && gofmt -l services/payout_release.go handlers/admin_payout.go` prints nothing;
`go vet ./services/ ./handlers/ ./routes/ 2>&1` clean;
`go test ./... 2>&1 | tail -20` — suite green (no regressions).
</verify>

<done>Whole suite green, formatted, vetted; issue updated.</done>

</tasks>

<out_of_scope>
Track as #388 follow-ups (do NOT build here):
- **tesserix-home admin UI** (payout queue page) — separate repo slice.
- **Auto-approve sweep** that auto-releases eligible holds past N hours — only the
  `GetPayoutAutoApproveHours` getter (default 0 = disabled) ships now.
- **#400 late-dispute freeze** of already-eligible holds.
- **Group-order holds** and **refund pairing** beyond the raw `ReverseTransfer` call.
- Persisting withhold/reverse **reason as a column** (currently audit-only).
</out_of_scope>

<threat_model>
STRIDE — admin payout actuator (trust boundary: admin BFF → payout mutation → Razorpay).

| ID | Category | Component | Disposition | Mitigation |
|----|----------|-----------|-------------|------------|
| T-388-01 | Elevation | release/withhold/reverse endpoints | mitigate | Routes live under the existing `admin` group (`bffAuth` + `middleware.RequireAdmin()`, routes.go:788); no per-route auth needed, non-admin → 401/403. |
| T-388-02 | Tampering | double/racing release of one hold | mitigate | Only `release_eligible` can be released; the transition is a single conditional `UPDATE ... WHERE payout_hold_status IN (...)` — `RowsAffected==0` → `ErrHoldNotEligible` (409). Two concurrent releases: exactly one wins. Same guard for withhold/reverse allowed-source sets. |
| T-388-03 | Repudiation | admin denies acting on a payout | mitigate | Every action writes an `audit_logs` row via `LogAudit(c, "payout.{released,withheld,reversed,bulk_released}", ...)` with actor, entity id, old→new status, and the mandatory reason for withhold/reverse. |
| T-388-04 | Info-disclosure / Financial | money moves while gateway unverified | mitigate | `ReleaseOrderPayouts`/`ReverseOrderPayouts` are hard no-ops when `OrderPayoutAutoReleaseEnabled` is OFF; `ReleaseDayPayout`/`ReverseTransfer` no-op when `MealPlanEscrowEnabled` is OFF or no transfer id. Launch config = both OFF → DB state advances, **zero money moved**. |
| T-388-05 | DoS / partial-fail | bulk release, or seam fails mid-way | accept | Bulk skips ineligible (reported, not fatal). Post-commit seam failure leaves status `released` but is retry-safe: Razorpay release/reverse are idempotent and can be re-driven by the (out-of-scope) reconcile/sweep. Reverse is mandatory-reason to bound abuse. |
</threat_model>

<success_criteria>
- Pending queue lists only `release_eligible` orders + meal-plan days (awaiting via
  `?include=awaiting`), with amount + age; excludes disputed/withheld/reversed/released.
- Release advances `release_eligible → released`, stages `payments.hold_released`,
  writes audit; re-release → 409. Flag OFF = no money; flag ON reaches the seam.
- Withhold (`→ withheld`) and reverse (`→ reversed`) require a reason (400 without),
  are audited, and drive the reverse seam; withheld rows drop out of the queue.
- Bulk release advances all eligible, skips ineligible with a report.
- Non-admin blocked; all funcs <50 lines, `%w`-wrapped, structured JSON errors, no
  handler panics; full api suite green.
</success_criteria>
