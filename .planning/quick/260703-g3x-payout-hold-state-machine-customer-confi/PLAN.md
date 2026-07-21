---
phase: quick
plan: 01
type: execute
wave: 1
depends_on: []
autonomous: true
requirements: [GH-387]
branch: feat/payout-hold-state-machine
files_modified:
  - apps/api/models/payout_hold.go
  - apps/api/models/order.go
  - apps/api/models/meal_plan.go
  - apps/api/migrations/20260703000001_add_payout_hold_state.up.sql
  - apps/api/migrations/20260703000001_add_payout_hold_state.down.sql
  - apps/api/services/payout_hold.go
  - apps/api/services/meal_plan_fulfillment.go
  - apps/api/services/provider.go
  - apps/api/services/shadowfax_webhook.go
  - apps/api/services/temporal_order.go
  - apps/api/handlers/delivery.go
  - apps/api/handlers/chefs.go
  - apps/api/handlers/payout_hold.go
  - apps/api/routes/routes.go
  - apps/api/services/payout_hold_test.go
  - apps/api/handlers/payout_hold_test.go

must_haves:
  truths:
    - "Delivering a regular order via ANY completion path (3PL provider webhook, Shadowfax webhook, Temporal settle saga, own-fleet courier, or chef self-delivery) sets its payout hold to awaiting_customer_confirmation and does NOT release funds — with the escrow flags ON or OFF."
    - "A customer explicitly confirming receipt advances the hold awaiting_customer_confirmation -> release_eligible."
    - "If an open (pending) OrderIssue exists for the order, confirming does NOT reach release_eligible (the hold becomes/stays disputed)."
    - "A disputed or already-released hold can never transition to release_eligible."
    - "Re-confirming an already-confirmed order/day is an idempotent 200 no-op."
    - "Confirming another customer's order/day returns 403/404 with no state change."
    - "GetCustomerConfirmWindowHours returns 24 by default and honours a payout.customer_confirm_window_hours override."
  artifacts:
    - path: apps/api/models/payout_hold.go
      provides: "PayoutHoldStatus enum (none/awaiting_customer_confirmation/release_eligible/released/disputed)"
      contains: "PayoutHoldAwaitingConfirmation"
    - path: apps/api/services/payout_hold.go
      provides: "SetOrderHoldAwaitingConfirmation, SetMealPlanDayHoldAwaitingConfirmation, ConfirmOrderHold, ConfirmMealPlanDayHold, ConfirmTodaysTiffinForCustomer, GetCustomerConfirmWindowHours, HasOpenOrderIssue"
    - path: apps/api/handlers/payout_hold.go
      provides: "ConfirmOrderReceived, ConfirmMealPlanDayReceived, ConfirmTodaysTiffin"
    - path: apps/api/migrations/20260703000001_add_payout_hold_state.up.sql
      provides: "nullable/defaulted hold columns on orders + meal_plan_days"
      contains: "payout_hold_status"
  key_links:
    - from: "regular-order delivery completion (5 sites)"
      to: services.SetOrderHoldAwaitingConfirmation
      via: "replaces services.ReleaseOrderPayouts at ALL delivered transitions: services/provider.go:357, services/shadowfax_webhook.go:82, services/temporal_order.go:102 (SettleOrderPayouts), handlers/delivery.go:615, + new call in handlers/chefs.go delivered case"
      pattern: "SetOrderHoldAwaitingConfirmation"
    - from: apps/api/services/meal_plan_fulfillment.go
      to: services.SetMealPlanDayHoldAwaitingConfirmation
      via: "replaces ReleaseDayPayout inside MarkMealPlanDayDelivered (shared by all completion paths)"
      pattern: "SetMealPlanDayHoldAwaitingConfirmation"
    - from: apps/api/routes/routes.go
      to: handlers.PayoutHoldHandler
      via: "POST /orders/:id/confirm-received, POST /meal-plans/:id/days/:dayId/confirm-received, POST /tiffin/confirm-today"
      pattern: "confirm-received"
    - from: apps/api/services/payout_hold.go
      to: models.OrderIssue
      via: "HasOpenOrderIssue gate before release_eligible"
      pattern: "HasOpenOrderIssue"
---

<objective>
Introduce a payout **hold state machine** (GH #387) so a delivered order/day no longer releases chef funds directly. On delivery the hold becomes `awaiting_customer_confirmation`; only an explicit customer confirmation advances it to `release_eligible` — and only when no open `OrderIssue` disputes it. Reaching `release_eligible` moves NO money by itself; the actual Razorpay `ReleaseTransfer` is driven later, off `release_eligible`, by the admin payout queue (#388). This is the keystone seam of the escrow control plane.

Purpose: decouple "delivered" from "chef paid" so the customer's confirmation (or, in a follow-up, a timeout) gates release, and disputes can block payout.
Output: hold fields on `Order` + `MealPlanDay`, a small transition/confirm service, customer confirm endpoints, and a settings getter for the (follow-up) auto-confirm window — all behind the existing escrow flags so **live behaviour is unchanged until the flags flip**.
Scope: BACKEND ONLY. No mobile UI. No refund/reversal changes.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
</execution_context>

<context>
Ground truth (all re-verified against the tree on 2026-07-03):

**Flags — both default OFF:**
- `services.MealPlanEscrowActive()` -> `config.AppConfig.MealPlanEscrowEnabled` (services/meal_plan_escrow.go:32-35; field config/config.go:119).
- `payoutMovementEnabled()` -> `config.AppConfig.OrderPayoutAutoReleaseEnabled` (services/order_payout.go:31-34; field config/config.go:114).

**Release paths (what we decouple):**
- Meal-plan day — services/meal_plan_fulfillment.go:202 `MarkMealPlanDayDelivered(orderID)`: inside a tx sets `MealPlanDay.Status=delivered` + `delivered_at`, then calls `ReleaseDayPayout(tx,&day)` (line 221), then emits `SubjectMealPlanDayDelivered`. Called from EVERY completion path (provider.go:355, shadowfax_webhook.go:80, delivery.go:609, chefs.go:1074). Decoupling once inside this function covers them all.
- Regular order — services/order_payout.go:47 `ReleaseOrderPayouts(orderID)`. **REVISED SURVEY: it has FOUR production callers, not one** (grep-verified 2026-07-03):
    1. services/provider.go:357 — generic 3PL provider webhook `delivered` handler. **LIVE (3PL is the active fulfilment model).**
    2. services/shadowfax_webhook.go:82 — Shadowfax webhook `delivered` handler. **LIVE.**
    3. services/temporal_order.go:102 — `SettleOrderPayouts` saga activity (settle-on-delivery, #123). **LIVE (durable path).**
    4. handlers/delivery.go:615 — own-fleet courier pipeline. **RETIRED** (own driver fleet retired 2026-06-11; 3PL only) but still compiled — decouple anyway for safety.
  The chef self-delivery delivered case (handlers/chefs.go:1066-1075) does NOT currently call `ReleaseOrderPayouts` — it only calls `MarkMealPlanDayDelivered` + `MarkGroupOrderDelivered`. So a chef-self-delivered *regular* order never auto-releases today; this plan adds the hold there for consistency (5th SetHold site).
  (Note: services/order_payout_test.go:24 also references `ReleaseOrderPayouts` — that is an EXISTING test of the seam function, which we keep; not a production caller.)
- Group order — services/group_order_payout.go:98 `MarkGroupOrderDelivered` -> `ReleaseGroupChefPayout` (services/group_order_payout.go:51). **NOT flag-gated, and it fires on ALL completion paths (provider/shadowfax/delivery/chefs).** OUT OF SCOPE this slice (no `GroupOrder` hold fields, no group confirm surface). It is ALREADY ungated today, so we change nothing about it and live behaviour is unchanged — but see &lt;out_of_scope&gt;: "no delivered path auto-releases" is therefore NOT fully true (group orders still auto-release). Group-order hold decoupling is an explicit follow-up.

**Settings pattern to copy (NOT `GetCommissionRate` — that symbol does NOT exist):**
services/order_issue.go:66 `GetIssueConfig(db)` and services/loyalty.go:62 `GetLoyaltyConfig(db)` read `models.PlatformSettings` via `db.Where("key LIKE ?", "prefix.%").Find(&settings)` and fold each key with `strconv` + a hardcoded default. Mirror this exactly.

**Open-issue signal:** handlers/order_issue.go:33 `ReportIssue` creates `models.OrderIssue` with `Status = models.IssuePending`. Statuses (models/order_issue.go:37-43): `pending | auto_refunded | resolved | rejected`. "Open" = `IssuePending` (still under review). `auto_refunded`/`resolved` are owned by the refund/reversal path (explicitly out of scope) and must not be re-touched here. NOTE (MINOR): the dispute gate is CONFIRM-TIME only — an OrderIssue filed AFTER a hold already reached `release_eligible` will NOT roll it back to `disputed`. Post-eligible disputes are owned by the admin payout queue #388 (which must re-check open issues before calling ReleaseTransfer). Documented, not fixed here.

**Schema mechanism:** `Order` (database/database.go:170) and `MealPlanDay` (database/database.go:219) are BOTH in the `DB.AutoMigrate(...)` list run by `database.Migrate()` (main.go:95-97). No golang-migrate runner is wired (the migrations/*.sql files are the auditable convention, e.g. 20260515000001_add_marketing_consent_to_users.{up,down}.sql, using `ADD COLUMN IF NOT EXISTS`). => Adding the struct fields is what actually creates the columns (incl. the in-memory sqlite test harness). We ALSO ship the timestamped SQL pair to match repo convention / production auditability.

**Routes:** customer `orders` group routes/routes.go:519-537 (bffAuth, has `report-issue`). Customer `meal-plans` group routes/routes.go:664-673 (bffAuth, already has `/:id/days/:dayId/skip`). Gin/httprouter forbids a static segment sharing a position with a wildcard, so the cross-plan bulk endpoint gets its OWN group `/tiffin` (it cannot live under `/orders/...` or `/meal-plans/...`).

**Test harness:** in-memory sqlite pattern in handlers/meal_plan_booking_test.go (DB setup + AutoMigrate + seed) and pure-logic style in services/meal_plan_fulfillment_test.go. `config.AppConfig` is swappable (save/restore) — see services/meal_plan_fulfillment_test.go TestMealPlanEscrowActive.
</context>

<interfaces>
New enum (create in Task 2):

    // models/payout_hold.go
    type PayoutHoldStatus string
    const (
        PayoutHoldNone                 PayoutHoldStatus = ""                                // pre-delivery / not applicable
        PayoutHoldAwaitingConfirmation PayoutHoldStatus = "awaiting_customer_confirmation"
        PayoutHoldReleaseEligible      PayoutHoldStatus = "release_eligible"
        PayoutHoldReleased             PayoutHoldStatus = "released"  // set later by #388 after ReleaseTransfer
        PayoutHoldDisputed             PayoutHoldStatus = "disputed"
    )

New service signatures (create in Tasks 3-5):

    // services/payout_hold.go
    func SetOrderHoldAwaitingConfirmation(db *gorm.DB, orderID uuid.UUID) error       // no-op unless razorpay_order_id != "" (regular order only)
    func SetMealPlanDayHoldAwaitingConfirmation(tx *gorm.DB, dayID uuid.UUID) error   // called inside the delivery tx
    func HasOpenOrderIssue(db *gorm.DB, orderID uuid.UUID) bool                       // status = pending
    func ConfirmOrderHold(db *gorm.DB, order *models.Order) (models.PayoutHoldStatus, error)  // awaiting -> release_eligible | disputed; idempotent
    func ConfirmMealPlanDayHold(db *gorm.DB, day *models.MealPlanDay) (models.PayoutHoldStatus, error)
    func ConfirmTodaysTiffinForCustomer(db *gorm.DB, customerID uuid.UUID) (int, error)
    func GetCustomerConfirmWindowHours(db *gorm.DB) int                               // key payout.customer_confirm_window_hours, default 24

Existing seams left in place for #388 (do NOT delete): `services.ReleaseOrderPayouts`, `services.ReleaseDayPayout` become defined-but-uncalled-in-production after decoupling (only order_payout_test.go still calls ReleaseOrderPayouts) — #388 will drive the real `ReleaseTransfer` off `release_eligible`.
</interfaces>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1 (RED): Write the failing tests first</name>
  <files>apps/api/services/payout_hold_test.go, apps/api/handlers/payout_hold_test.go</files>
  <behavior>
  Service tests (services/payout_hold_test.go, in-memory sqlite; save/restore config.AppConfig):
    - SetOrderHoldAwaitingConfirmation on a regular order (razorpay_order_id set) -> payout_hold_status = awaiting_customer_confirmation; on a meal-plan/consolidated order (razorpay_order_id == "") -> no-op (stays "").
    - SetMealPlanDayHoldAwaitingConfirmation -> day.payout_hold_status = awaiting_customer_confirmation.
    - **Regular-order NO-RELEASE invariant (BLOCKER 3): with OrderPayoutAutoReleaseEnabled = TRUE, invoking the 3PL/saga completion seam for a regular order (call SettleOrderPayouts or SetOrderHoldAwaitingConfirmation directly against a razorpay-charged order) sets payout_hold_status = awaiting_customer_confirmation and leaves the order UNRELEASED — assert the hold is set and no release side-effect occurred (order has no released marker; transfers untouched). Proves the flag being ON no longer implies release on delivery.**
    - ConfirmOrderHold from awaiting -> release_eligible + stamps CustomerConfirmedAt.
    - ConfirmOrderHold with an open (pending) OrderIssue -> returns disputed, NEVER release_eligible.
    - ConfirmOrderHold is idempotent: a second call on an already-confirmed order -> no-op, returns the same terminal status, does not overwrite CustomerConfirmedAt.
    - ConfirmOrderHold on a disputed hold -> stays disputed (guarded conditional update).
    - GetCustomerConfirmWindowHours -> 24 with no setting; 48 when PlatformSettings key payout.customer_confirm_window_hours = "48".
    - ConfirmMealPlanDayHold parallels ConfirmOrderHold (dispute keyed on day.OrderID).
  Handler/integration tests (handlers/payout_hold_test.go, sqlite + gin router, harness per handlers/meal_plan_booking_test.go):
    - MarkMealPlanDayDelivered on a day that HAS a PayoutTransferID -> day.payout_hold_status = awaiting_customer_confirmation and day.PayoutTransferID is UNCHANGED (release deferred) — assert with MealPlanEscrowEnabled BOTH true and false; day.Status = delivered either way.
    - POST /orders/:id/confirm-received (owner) -> 200, order.CustomerConfirmedAt set, payout_hold_status = release_eligible.
    - Same with an open OrderIssue -> 200 but payout_hold_status = disputed (not release_eligible).
    - POST /orders/:id/confirm-received by a DIFFERENT customer -> 403/404, no state change.
    - Re-POST confirm-received -> idempotent 200, no double transition.
  </behavior>
  <action>
  Author both test files referencing the interfaces above. They will FAIL TO COMPILE (symbols don't exist yet) — that is the RED state for Go TDD. Do NOT stub the production symbols yet. Reuse the sqlite/gin harness helpers from handlers/meal_plan_booking_test.go and the config save/restore idiom from services/meal_plan_fulfillment_test.go. Assert on payout_hold_status / customer_confirmed_at / HTTP status. Keep each test function focused and named for the behaviour.
  </action>
  <verify>
    <automated>cd apps/api && go vet ./services/ ./handlers/ 2>&1 | grep -Eiq 'undefined|undeclared|payout_hold' && echo RED-OK-compile-fails</automated>
  </verify>
  <done>Both test files exist and the packages fail to compile (red) because the production symbols are not yet defined; includes the 3PL/saga regular-order no-release test.</done>
</task>

<task type="auto">
  <name>Task 2: Model enum + fields + migration</name>
  <files>apps/api/models/payout_hold.go, apps/api/models/order.go, apps/api/models/meal_plan.go, apps/api/migrations/20260703000001_add_payout_hold_state.up.sql, apps/api/migrations/20260703000001_add_payout_hold_state.down.sql</files>
  <action>
  1. Create models/payout_hold.go: package models, the `PayoutHoldStatus` type + the five constants from &lt;interfaces&gt; (doc comment starting "PayoutHoldStatus" per Go convention).
  2. models/order.go `Order` struct (after the refund block ~line 161, before CreatedAt) add:
       PayoutHoldStatus    PayoutHoldStatus `gorm:"type:varchar(32);default:''" json:"payoutHoldStatus,omitempty"`
       CustomerConfirmedAt *time.Time       `gorm:"" json:"customerConfirmedAt,omitempty"`
     Comment: hold is independent of Status; release_eligible is consumed by the admin payout queue (#388) and moves no money by itself.
  3. models/meal_plan.go `MealPlanDay` struct (after RefundTxnID ~line 122) add the SAME two fields.
  4. Create the timestamped migration pair (mirror 20260515000001_add_marketing_consent_to_users.{up,down}.sql: header comment + `ADD COLUMN IF NOT EXISTS`). Tables `orders` and `meal_plan_days`.
     up.sql:
       ALTER TABLE orders ADD COLUMN IF NOT EXISTS payout_hold_status VARCHAR(32) NOT NULL DEFAULT '', ADD COLUMN IF NOT EXISTS customer_confirmed_at TIMESTAMPTZ;
       ALTER TABLE meal_plan_days ADD COLUMN IF NOT EXISTS payout_hold_status VARCHAR(32) NOT NULL DEFAULT '', ADD COLUMN IF NOT EXISTS customer_confirmed_at TIMESTAMPTZ;
     down.sql: DROP COLUMN IF EXISTS both columns on both tables.
  In a header comment note that AutoMigrate (database/database.go:170,219) is the runtime mechanism (creates the columns from these struct tags, incl. sqlite tests); the SQL pair is the auditable production DDL matching repo convention.
  Struct-field additions only; no behavioural change yet.
  </action>
  <verify>
    <automated>cd apps/api && go build ./... && grep -q "PayoutHoldAwaitingConfirmation" models/payout_hold.go && grep -q "customer_confirmed_at" migrations/20260703000001_add_payout_hold_state.up.sql && grep -q "DROP COLUMN IF EXISTS" migrations/20260703000001_add_payout_hold_state.down.sql</automated>
  </verify>
  <done>Enum + both structs compile; the RED tests now COMPILE (still failing on behaviour); migration pair present in the correct format.</done>
</task>

<task type="auto">
  <name>Task 3: Transition seam service + decouple ALL delivery completion paths from release</name>
  <files>apps/api/services/payout_hold.go, apps/api/services/meal_plan_fulfillment.go, apps/api/services/provider.go, apps/api/services/shadowfax_webhook.go, apps/api/services/temporal_order.go, apps/api/handlers/delivery.go, apps/api/handlers/chefs.go</files>
  <action>
  Create services/payout_hold.go (package services) with:
    - SetOrderHoldAwaitingConfirmation(db, orderID): select id, razorpay_order_id, payout_hold_status; if razorpay_order_id == "" return nil (meal-plan/group orders settle via their own hold — mirrors order_payout.go scoping); else conditional update `WHERE id=? AND payout_hold_status=''` SET payout_hold_status = awaiting_customer_confirmation (idempotent; RowsAffected==0 is fine). Wrap errors with %w.
    - SetMealPlanDayHoldAwaitingConfirmation(tx, dayID): conditional update on meal_plan_days from '' -> awaiting_customer_confirmation; RowsAffected-tolerant.
  Decouple (no money moves in this slice) — REPLACE `ReleaseOrderPayouts` at ALL FOUR production call sites + ADD the chef site (5 total). Keep the adjacent `MarkMealPlanDayDelivered` / `MarkGroupOrderDelivered` calls untouched:
    - services/provider.go:357 (3PL generic webhook `delivered`): REPLACE `ReleaseOrderPayouts(delivery.OrderID)` with `if err := SetOrderHoldAwaitingConfirmation(database.DB, delivery.OrderID); err != nil { log.Printf(...) }`. Same package -> unqualified call. Update the adjacent comment.
    - services/shadowfax_webhook.go:82 (Shadowfax webhook `delivered`): REPLACE `_ = ReleaseOrderPayouts(order.ID)` with `if err := SetOrderHoldAwaitingConfirmation(database.DB, order.ID); err != nil { log.Printf(...) }`. Update the comment.
    - services/temporal_order.go:102 (`SettleOrderPayouts` saga activity): REPLACE `return ReleaseOrderPayouts(orderID)` with `return SetOrderHoldAwaitingConfirmation(database.DB, orderID)` (preserves the error-return retry semantics of the activity). Update the func doc: it now parks the hold for customer confirmation instead of releasing; the flag-gated release moves to #388.
    - handlers/delivery.go:615 (retired own-fleet courier): REPLACE `services.ReleaseOrderPayouts(delivery.OrderID)` with `if err := services.SetOrderHoldAwaitingConfirmation(database.DB, delivery.OrderID); err != nil { log.Printf(...) }`.
    - handlers/chefs.go delivered case (~line 1066-1075): ADD `if err := services.SetOrderHoldAwaitingConfirmation(database.DB, order.ID); err != nil { log.Printf(...) }` (regular chef-self-delivered orders now also get the hold; no-op for tiffin/group via the razorpay_order_id gate). Do NOT remove the existing MarkMealPlanDayDelivered/MarkGroupOrderDelivered calls.
  Also decouple the meal-plan day: services/meal_plan_fulfillment.go MarkMealPlanDayDelivered (~line 221): REPLACE `if err := ReleaseDayPayout(tx, &day); err != nil { return err }` with `if err := SetMealPlanDayHoldAwaitingConfirmation(tx, day.ID); err != nil { return err }`. Keep the status=delivered update AND the SubjectMealPlanDayDelivered event. Update the func doc ("park the payout in a customer-confirmation hold" instead of "release its held chef payout").
  Leave services.ReleaseOrderPayouts / ReleaseDayPayout DEFINED (production-uncalled) as the #388 seam — add a one-line comment on each pointing to #388. Do NOT touch ReverseOrderPayouts / ReleaseGroupChefPayout / MarkGroupOrderDelivered / refund paths (group-order decoupling is a follow-up).
  Functions &lt;50 lines, early-return, structured errors, no handler panics, immutable-style updates.
  </action>
  <verify>
    <automated>cd apps/api && go build ./... && grep -q "SetMealPlanDayHoldAwaitingConfirmation(tx" services/meal_plan_fulfillment.go && ! grep -REn "ReleaseOrderPayouts\(" services/provider.go services/shadowfax_webhook.go services/temporal_order.go handlers/delivery.go && grep -c "SetOrderHoldAwaitingConfirmation" services/provider.go services/shadowfax_webhook.go services/temporal_order.go handlers/delivery.go handlers/chefs.go && go test ./services/ -run 'HoldAwaiting|NoRelease|ConfirmWindow' 2>&1 | tail -4</automated>
  </verify>
  <done>Build green; ALL FOUR production ReleaseOrderPayouts callers (provider, shadowfax, temporal-saga, delivery) now set the hold, plus the new chefs.go site; meal-plan day parks its hold; ReleaseOrderPayouts/ReleaseDayPayout remain defined (test-only + #388 seam); the 3PL/saga no-release test passes with the flag ON.</done>
</task>

<task type="auto">
  <name>Task 4: Confirm transition logic + endpoints + routes</name>
  <files>apps/api/services/payout_hold.go, apps/api/handlers/payout_hold.go, apps/api/routes/routes.go</files>
  <action>
  Extend services/payout_hold.go:
    - HasOpenOrderIssue(db, orderID) bool: count > 0 `WHERE order_id=? AND status='pending'`.
    - ConfirmOrderHold(db, order): idempotent — if order.CustomerConfirmedAt != nil return (order.PayoutHoldStatus, nil). In a tx: if HasOpenOrderIssue -> conditional update `WHERE id=? AND payout_hold_status IN ('awaiting_customer_confirmation','disputed')` SET payout_hold_status='disputed', customer_confirmed_at=now; return disputed. Else conditional update `WHERE id=? AND payout_hold_status='awaiting_customer_confirmation'` SET payout_hold_status='release_eligible', customer_confirmed_at=now. The WHERE guard IS the safety invariant: released/disputed can never flip to release_eligible. Return the resulting status.
    - ConfirmMealPlanDayHold(db, day): same shape; dispute check via HasOpenOrderIssue(*day.OrderID) (if day.OrderID == nil there is no dispute source -> proceed to release_eligible).
    - ConfirmTodaysTiffinForCustomer(db, customerID): find the customer's MealPlanDays joined to their plan where status='delivered', payout_hold_status='awaiting_customer_confirmation', customer_confirmed_at IS NULL, Date is today (IST). Apply ConfirmMealPlanDayHold to each; return count confirmed.
  Create handlers/payout_hold.go: `PayoutHoldHandler` struct + `NewPayoutHoldHandler()`, methods:
    - ConfirmOrderReceived(c): GetUserID; parse :id; load order `WHERE id=? AND customer_id=?` (404 if not owned/found — mirrors handlers/order_issue.go:42-48); call ConfirmOrderHold; return 200 {payoutHoldStatus, customerConfirmedAt, message}.
    - ConfirmMealPlanDayReceived(c): parse :id (plan) + :dayId; load day joined to plan verifying plan.customer_id == userID (403/404 otherwise); call ConfirmMealPlanDayHold; 200.
    - ConfirmTodaysTiffin(c): GetUserID; call ConfirmTodaysTiffinForCustomer; 200 {confirmed: n}.
  Wire routes/routes.go:
    - orders group (~519): `orders.POST("/:id/confirm-received", payoutHoldHandler.ConfirmOrderReceived) // #387`.
    - meal-plans group (~664, matches the skip pattern): `mealPlans.POST("/:id/days/:dayId/confirm-received", payoutHoldHandler.ConfirmMealPlanDayReceived) // #387`.
    - NEW group after meal-plans: `tiffin := v1.Group("/tiffin"); tiffin.Use(bffAuth(bffKey, bffWindow)); tiffin.POST("/confirm-today", payoutHoldHandler.ConfirmTodaysTiffin) // #387` (own group — cannot share the /orders/:id or /meal-plans/:id wildcard slot).
    - Construct the handler near the other handler constructors (`payoutHoldHandler := handlers.NewPayoutHoldHandler()`).
  Structured JSON errors, no raw error codes to the user, early-return, funcs &lt;50 lines.
  </action>
  <verify>
    <automated>cd apps/api && go build ./... && grep -q "confirm-received" routes/routes.go && grep -q "confirm-today" routes/routes.go && go test ./services/ ./handlers/ -run 'PayoutHold|Confirm' 2>&1 | tail -6</automated>
  </verify>
  <done>All three confirm endpoints wired and owner-scoped; ConfirmOrderHold/ConfirmMealPlanDayHold enforce the awaiting-only transition and the open-issue dispute guard; handler + service confirm tests pass.</done>
</task>

<task type="auto">
  <name>Task 5: Confirm-window settings getter</name>
  <files>apps/api/services/payout_hold.go</files>
  <action>
  Add GetCustomerConfirmWindowHours(db *gorm.DB) int mirroring services/order_issue.go:66 GetIssueConfig exactly: default 24; `db.Where("key LIKE ?", "payout.%").Find(&settings)`; on key `payout.customer_confirm_window_hours` parse with strconv.Atoi and only override on success (ignore parse errors, keep default). Doc-comment that this is consumed by the follow-up auto-confirm sweep; the setting exists now so ops can pre-tune it. Not called by any transition in this slice.
  </action>
  <verify>
    <automated>cd apps/api && go build ./... && go test ./services/ -run 'ConfirmWindow' -v 2>&1 | tail -5</automated>
  </verify>
  <done>Getter returns 24 by default and honours a payout.customer_confirm_window_hours override; window test passes.</done>
</task>

<task type="auto">
  <name>Task 6: Full verify — build, vet, format, whole suite + no-production-caller gate</name>
  <files>apps/api</files>
  <action>
  Run the full backend gate. Fix any residual compile/vet/test failures. Confirm no import cycles (services -> models only), no unused imports, gofmt clean. Confirm the escrow flags are untouched and that with both flags ON no money-movement (ReleaseOrderPayouts/ReleaseDayPayout) is reachable from ANY delivery completion path (provider, shadowfax, temporal saga, delivery, chefs). The ONLY remaining `ReleaseOrderPayouts` references outside order_payout.go must be services/order_payout_test.go. Do NOT expand scope.
  </action>
  <verify>
    <automated>cd apps/api && gofmt -l services/payout_hold.go handlers/payout_hold.go models/payout_hold.go | grep -q . && echo FMT-DIRTY || (go build ./... && go vet ./services/ ./handlers/ ./models/ && (grep -REln "ReleaseOrderPayouts" --include=*.go . | grep -vE "order_payout\.go|order_payout_test\.go" | grep -q . && echo STRAY-PROD-CALLER || echo NO-STRAY-CALLERS) && go test ./services/ ./handlers/ ./models/ 2>&1 | tail -12)</automated>
  </verify>
  <done>gofmt clean; build + vet pass; the only ReleaseOrderPayouts references left are order_payout.go (definition/doc) + order_payout_test.go (seam test); services/handlers/models test packages green including all new PayoutHold/Confirm/no-release tests.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| customer app -> BFF -> /orders,/meal-plans,/tiffin confirm endpoints | untrusted caller asserts "I received this" and drives a money-adjacent state transition |
| 3PL/webhook/saga completion -> hold service | delivery event parks funds in a hold (no release) across all fulfilment paths |
| hold service -> #388 admin queue (future) | only `release_eligible` holds are eligible for a real Razorpay ReleaseTransfer |

## STRIDE Threat Register

**KEY INVARIANT: a disputed or unconfirmed hold must NEVER reach `release_eligible`.** Enforced by a conditional `UPDATE ... WHERE payout_hold_status = 'awaiting_customer_confirmation'` — the only transition that can produce `release_eligible`. Disputed/released rows fail the WHERE and are untouched.

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-387-01 | Elevation of privilege | ConfirmOrderHold / ConfirmMealPlanDayHold | mitigate | Transition to release_eligible is a single conditional update gated on `payout_hold_status='awaiting_customer_confirmation'`; a disputed/released hold can never flip. HasOpenOrderIssue(pending) forces `disputed` before any release. |
| T-387-02 | Tampering | confirm endpoints (:id / :dayId) | mitigate | Ownership enforced: order loaded `WHERE id=? AND customer_id=?` (404); meal-plan day verified against plan.customer_id (403/404). A customer cannot confirm another's order to trigger a payout. |
| T-387-03 | Spoofing / Repudiation | all confirm endpoints | mitigate | Behind bffAuth + middleware.GetUserID; CustomerConfirmedAt is stamped server-side as the audit record of who released the hold. |
| T-387-04 | Tampering (race / replay) | concurrent confirm + duplicate 3PL/webhook/saga delivered events | mitigate | Idempotent: already-confirmed -> no-op 200; SetOrderHoldAwaitingConfirmation and all transitions are conditional updates (RowsAffected), so replayed webhooks / retried saga activities cannot double-advance or re-open a hold. |
| T-387-05 | Information disclosure | non-owned confirm | mitigate | Return 404 for non-owned orders (no existence leak), matching the order-issue handler; response carries only the caller's own hold status. |
| T-387-06 | Denial of service | POST /tiffin/confirm-today | mitigate | Bulk confirm is scoped to the caller's own delivered-today unconfirmed days (join on plan.customer_id); bounded set, no arbitrary fan-out. |
| T-387-07 | Elevation (money movement) | reaching release_eligible | accept | release_eligible moves NO money in this slice; real ReleaseTransfer stays behind #388 + OrderPayoutAutoReleaseEnabled. Even a mistaken release_eligible cannot pay a chef until #388 ships and its flag flips. |
| T-387-08 | Tampering (late dispute) | OrderIssue filed AFTER release_eligible | transfer | The confirm-time gate does not roll an already-eligible hold back to disputed. #388 MUST re-check HasOpenOrderIssue immediately before calling ReleaseTransfer. Documented as #388's responsibility. |
</threat_model>

<verification>
- `go build ./...` and `go vet ./services/ ./handlers/ ./models/` pass.
- `go test ./services/ ./handlers/ ./models/` passes, including the new PayoutHold/Confirm tests: 3PL/saga regular-order sets-hold-no-release (flag ON), meal-plan-day delivery sets-hold-no-release (flag on/off), confirm->release_eligible, dispute-guard, idempotency, ownership 403/404, window default+override.
- Grep gates: delivery.go, provider.go, shadowfax_webhook.go, temporal_order.go no longer call `ReleaseOrderPayouts`; each (plus chefs.go) now calls `SetOrderHoldAwaitingConfirmation`; meal_plan_fulfillment.go calls `SetMealPlanDayHoldAwaitingConfirmation`; routes.go registers `confirm-received` (x2) + `confirm-today`. The only surviving `ReleaseOrderPayouts` references are order_payout.go (def/doc) + order_payout_test.go.
- Escrow flags untouched; with both flags ON, no reachable regular-order delivery path calls Razorpay ReleaseTransfer.
</verification>

<success_criteria>
- Delivering a REGULAR order via any live completion path (3PL provider webhook, Shadowfax webhook, Temporal settle saga) OR the retired courier / chef self-delivery path sets its hold to `awaiting_customer_confirmation` and releases nothing (flags ON or OFF); meal-plan-day delivery does the same.
- `POST /orders/:id/confirm-received` advances awaiting -> `release_eligible`, stamps `CustomerConfirmedAt`, and is idempotent + owner-scoped — and now actually has a hold to advance on 3PL-delivered orders.
- An open (pending) `OrderIssue` at confirm time forces `disputed` and blocks `release_eligible`; disputed/released holds are immutable to the release transition; post-eligible disputes are #388's job.
- Per-day and bulk tiffin confirmation work equivalently.
- `GetCustomerConfirmWindowHours` returns 24 by default, honours the override, and the `payout.customer_confirm_window_hours` setting exists for the follow-up sweep.
- Diff is minimal, cohesive, gofmt-clean; `ReleaseOrderPayouts`/`ReleaseDayPayout` remain as the #388 seam; refund/reversal and group-order semantics unchanged.
</success_criteria>

<out_of_scope>
Tracked as follow-ups (do NOT implement here):
- **Group-order hold decoupling.** `MarkGroupOrderDelivered` -> `ReleaseGroupChefPayout` (services/group_order_payout.go:51) is NOT flag-gated and STILL fires on every completion path in this slice. It is already ungated today so live behaviour is unchanged, BUT this means "no delivered path auto-releases" is NOT fully true after this slice — group/office orders still auto-release. A group hold state machine + a group confirm surface is a separate follow-up.
- Auto-confirm timeout sweep (cron/Temporal) reading GetCustomerConfirmWindowHours to auto-advance stale `awaiting_customer_confirmation` holds — NEXT slice.
- NATS events `payments.hold_release_eligible` / `payments.hold_disputed` — NEXT slice (emit seam is the confirm transition in services/payout_hold.go).
- Admin payout queue that consumes `release_eligible` and calls Razorpay `ReleaseTransfer` off `ReleaseOrderPayouts`/`ReleaseDayPayout` — #388. #388 also owns POST-eligible dispute rollback (re-check HasOpenOrderIssue before ReleaseTransfer).
- ALL mobile app UI (customer confirm screen, vendor payout-state display) — separate slice.
- No changes to refund/reversal semantics (ReverseOrderPayouts, RefundDay, order-issue refunds).
</out_of_scope>

<output>
After completion, create `.planning/quick/260703-g3x-payout-hold-state-machine-customer-confi/SUMMARY.md` recording: fields/enum added, the FIVE SetOrderHoldAwaitingConfirmation call-sites (provider, shadowfax, temporal-saga, delivery, chefs) + the meal-plan-day decouple, the endpoints + routes added, the confirm/dispute invariant, and the follow-up seams (#388 release + post-eligible dispute rollback, auto-confirm sweep, NATS events, group-order hold decoupling).
</output>
