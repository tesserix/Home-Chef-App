---
phase: quick/260703-km0-payout-drift-reconcile
plan: 01
type: execute
wave: 1
depends_on: []
branch: feat/payout-drift-reconcile
autonomous: true
requirements: [GH-459]
files_modified:
  - apps/api/models/order.go
  - apps/api/models/meal_plan.go
  - apps/api/migrations/20260703000003_add_payout_settled_at.up.sql
  - apps/api/migrations/20260703000003_add_payout_settled_at.down.sql
  - apps/api/services/payout_release.go
  - apps/api/services/meal_plan_escrow.go
  - apps/api/services/payout_reconcile_cron.go
  - apps/api/services/cron_temporal.go
  - apps/api/services/payout_hold_test.go
  - apps/api/services/payout_reconcile_cron_test.go
must_haves:
  truths:
    - "A released hold whose money seam failed (payout_settled_at NULL) is re-driven and, on seam success, stamped payout_settled_at"
    - "A released hold already stamped payout_settled_at is never re-driven"
    - "A reversed-but-unsettled hold is re-driven and stamped on success"
    - "The reconcile scan is a pure no-op when both escrow flags are off"
    - "Re-driving an already-released meal-plan day transfer does not error"
    - "ReleaseHold/ReverseHold stamp payout_settled_at ONLY after the seam returns nil"
  artifacts:
    - path: apps/api/services/payout_reconcile_cron.go
      provides: "released/reversed-but-unsettled reconcile sweep, flag-gated + bounded"
      contains: "runPayoutReconcileScan"
    - path: apps/api/models/order.go
      provides: "Order.PayoutSettledAt *time.Time (gorm payout_settled_at)"
      contains: "PayoutSettledAt"
    - path: apps/api/models/meal_plan.go
      provides: "MealPlanDay.PayoutSettledAt *time.Time (gorm payout_settled_at)"
      contains: "PayoutSettledAt"
  key_links:
    - from: apps/api/services/payout_release.go
      to: "stampPayoutSettled"
      via: "settleRelease/settleReverse after seam==nil"
      pattern: "stampPayoutSettled"
    - from: apps/api/services/cron_temporal.go
      to: "runPayoutReconcileScan"
      via: "cronJobs() registration"
      pattern: "payout-reconcile"
---

<objective>
Close the released/reversed-but-unsettled payout drift (GH #459). Today `ReleaseHold`
and `ReverseHold` (`services/payout_release.go`) commit the `payout_hold_status` flip in
`transitionHold` and then run the money seam (`releaseMoney`/`reverseMoney`) POST-COMMIT.
If the seam fails, the row is left `released`/`reversed` with money unmoved and nothing
re-drives it (the auto-confirm sweep only touches `awaiting_customer_confirmation`). This
is safe today only because both escrow flags are OFF (the seam is a no-op).

Build a durable, idempotent, flag-gated reconcile:
1. Add a `payout_settled_at *time.Time` column to `Order` and `MealPlanDay` that marks
   "the money seam completed" — decoupling status-committed from money-confirmed-moved.
2. Stamp `payout_settled_at = now` ONLY AFTER the seam returns nil, via one shared settle
   helper reused by both the primary path and the reconcile (DRY).
3. Make `ReleaseDayPayout` tolerate an already-released transfer (idempotent re-drive).
4. Add a `payout-reconcile` cron that re-drives `released`/`reversed` holds with
   `payout_settled_at IS NULL`, gated on `payoutMovementEnabled() || MealPlanEscrowActive()`.

Purpose: a stranded row (status committed, money unmoved) is retried until the seam
succeeds, before either escrow flag is turned on — the #388 follow-up the file header
already flags. Output: the reconcile cron + settle helper + the settled-at column.

Scope guard: THIS SLICE IS ONLY the released/reversed-but-unsettled reconcile. Out of
scope (note as follow-ups, do NOT touch here): #456 group-order gate, #457 refund/dispute
cross-guard, #458 disputed dead-end, #460 races, #461 auth, #462 medium cluster.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
</execution_context>

<context>
Branch off latest main: `git checkout main && git pull && git checkout -b feat/payout-drift-reconcile`.

<grounding> (re-verified against apps/api — cite these when implementing)
- `services/payout_release.go`: `transitionHold` (line 186, commits the status flip in a
  short tx), `ReleaseHold` (221 → `releaseMoney`), `releaseMoney` (236), `ReverseHold`
  (275 → `reverseMoney`), `reverseMoney` (294). Seam runs post-commit; both seams already
  no-op when the escrow flag is off / gateway nil.
- `services/order_payout.go`: `payoutMovementEnabled()` (line 32). `ReleaseOrderPayouts`
  (49) is NATURALLY IDEMPOTENT — `FetchOrderTransfers` then release only `OnHold==true`
  transfers (68-73), so re-driving an already-released order is a safe no-op.
  `ReverseOrderPayouts` (83) is best-effort/log-and-continue (already-reversed just logs).
- `services/meal_plan_escrow.go`: `MealPlanEscrowActive()` (33), `ReleaseDayPayout` (183)
  calls `rz.ReleaseTransfer(day.PayoutTransferID)` UNCONDITIONALLY (191) — the one seam
  that is NOT idempotent and must be hardened. No-op already when flag off / transfer id
  empty (184). `reverseMoney` day path (payout_release.go:305) is guarded on flag + id +
  gateway and calls `ReverseTransfer`.
- `services/razorpay.go`: `ReleaseTransfer(transferID)` (416) is a PATCH `on_hold:false`;
  its own doc says releasing an already-released transfer is a no-op ON RAZORPAY'S SIDE.
  `ReverseTransfer(transferID, amountPaise)` (449). `FetchOrderTransfers` (431) is
  order-scoped — THERE IS NO single-transfer `FetchTransfer` on the client. `GetRazorpay()`
  (106) returns a package-global client (not an interface) → not swappable in the sqlite
  harness; when escrow is off it returns nil and the seam no-ops.
- Cron template: `services/payout_auto_confirm_cron.go` — `const payoutAutoConfirmInterval`,
  `const sweepBatchLimit = 500`, `StartPayoutAutoConfirmCron(ctx)` (go func: run once +
  ticker + `ctx.Done()`), `runPayoutAutoConfirmScan` (`recover()`, bounded query,
  log-and-continue per row). Registered in `services/cron_temporal.go` `cronJobs()`
  (line 39, the `payout-auto-confirm` entry). Each job is `{name, interval, run, ticker}`.
- Models: `Order.PayoutHoldStatus` (models/order.go:168), `MealPlanDay.PayoutHoldStatus`
  (models/meal_plan.go:127), `MealPlanDay.PayoutTransferID` (117). Both AutoMigrated
  (database/database.go:170 orders, :219 meal_plan_days) — new struct tags create the
  columns at boot AND in the sqlite harness. Flags: `config.OrderPayoutAutoReleaseEnabled`
  + `config.MealPlanEscrowEnabled` (config/config.go:114/119).
- Migration convention: timestamped `.up.sql`/`.down.sql`. A real new column uses real
  `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` (see 20260703000001) — the doc-only `SELECT 1`
  form (20260703000002) is only for value-only/no-schema changes. Latest is 20260703000002.
- Test harness: `setupHoldDB` (payout_hold_test.go:25) hand-DDLs `orders` + `meal_plan_days`
  (gen_random_uuid() can't run on sqlite). Helpers: `withSweepDB` (swaps `database.DB`),
  `countOutbox`, `loadOrder`, `seedRegularOrder`, `seedAwaitingOrder`. Flag toggling
  pattern: save/restore `config.AppConfig` (see payout_hold_test.go:104-107).
</grounding>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1 (RED): failing reconcile + settle + day-idempotency tests + harness column</name>
  <files>apps/api/services/payout_hold_test.go, apps/api/services/payout_reconcile_cron_test.go</files>
  <behavior>
    Harness: add `payout_settled_at DATETIME` (nullable) to BOTH the `orders` and
    `meal_plan_days` CREATE TABLE statements in `setupHoldDB`. Add a `payout_settle_attempts INTEGER DEFAULT 0`
    column to both as well (Task 4 attempt cap). Add a small helper
    `seedReleasedOrder(t, db, settledAt *time.Time)` that inserts an order with
    `payout_hold_status='released'`, `razorpay_order_id<>''`, and the given (nullable)
    settled_at.

    New file payout_reconcile_cron_test.go — all initially failing:
    - `TestPayoutReconcile_NoopWhenFlagsOff`: seed a `released` order settled_at NULL;
      leave both flags off (default config.AppConfig nil / false); run
      `runPayoutReconcileScan`; assert the row is untouched (settled_at stays NULL) — pure
      no-op gate.
    - `TestPayoutReconcile_ReleasedUnsettledIsDriven`: set config.AppConfig with
      `OrderPayoutAutoReleaseEnabled:true` (GetRazorpay() is nil in the harness, so
      ReleaseOrderPayouts no-ops and returns nil == "seam succeeded"); seed a `released`
      order settled_at NULL; run scan; assert `payout_settled_at` is now stamped
      (NOT NULL); run scan a second time and assert it is NOT re-picked (stamp unchanged).
    - `TestPayoutReconcile_ReleasedAlreadySettledSkipped`: seed a `released` order with
      settled_at already set (past time); flag on; run scan; assert the stamp is unchanged
      (never re-driven).
    - `TestPayoutReconcile_ReversedUnsettledIsDriven`: seed a `reversed` order settled_at
      NULL; flag on; run scan; assert stamped.
    - `TestStampPayoutSettled_ConditionalOnce`: directly call the settle stamp helper twice
      on one id; assert it stamps once and the second call (settled_at already set) leaves
      the original timestamp unchanged (conditional UPDATE on settled_at IS NULL).
    - `TestReleaseDayPayout_IdempotentRedrive`: with escrow flag off, call `ReleaseDayPayout`
      twice on a day that has a `payout_transfer_id` set; assert both return nil (no error).
      (Meaningful gateway-level idempotency is bounded by the harness — GetRazorpay() is nil
      when the flag is off; document this limit in a test comment and assert the no-error
      contract the reconcile depends on.)

    Use the save/restore config.AppConfig pattern and withSweepDB from the existing harness.
  </behavior>
  <action>
    Extend `setupHoldDB` DDL (both tables) with `payout_settled_at DATETIME` and
    `payout_settle_attempts INTEGER DEFAULT 0`. Add `seedReleasedOrder`. Create
    payout_reconcile_cron_test.go with the six tests above. Reference the symbols the later
    tasks will create (`runPayoutReconcileScan`, `stampPayoutSettled`) so the package fails
    to COMPILE — that is the RED state.
  </action>
  <verify>
    <automated>cd apps/api && go test ./services/ -run 'PayoutReconcile|StampPayoutSettled|ReleaseDayPayout_Idempotent' 2>&1 | grep -qiE 'undefined|cannot find|FAIL|build failed' && echo RED-OK</automated>
  </verify>
  <done>Package fails to compile / tests fail because the reconcile + stamp symbols don't exist yet; harness carries payout_settled_at + payout_settle_attempts on both tables.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2 (GREEN-1): settled_at column + migration + shared settle helper wired into ReleaseHold/ReverseHold</name>
  <files>apps/api/models/order.go, apps/api/models/meal_plan.go, apps/api/migrations/20260703000003_add_payout_settled_at.up.sql, apps/api/migrations/20260703000003_add_payout_settled_at.down.sql, apps/api/services/payout_release.go</files>
  <behavior>
    - Model fields: add `PayoutSettledAt *time.Time` with gorm tag `payout_settled_at` to
      Order (next to PayoutHoldStatus, order.go:168) and MealPlanDay (meal_plan.go:127).
      Also add `PayoutSettleAttempts int` gorm tag `payout_settle_attempts;default:0` to both
      (consumed by Task 4). AutoMigrate creates the columns at boot + in the harness.
    - Migration: real ALTER pair (mirror 20260703000001 form):
      `ALTER TABLE orders ADD COLUMN IF NOT EXISTS payout_settled_at TIMESTAMPTZ, ADD COLUMN IF NOT EXISTS payout_settle_attempts INTEGER NOT NULL DEFAULT 0;`
      same for `meal_plan_days`. Down drops both columns on both tables. Header comment
      explains: marks "the money seam completed", so a successful path is released +
      settled_at set; a drift is released + settled_at NULL.
    - Settle helper (single source of truth for "seam succeeded → stamp"):
      `stampPayoutSettled(db, aggType, id) error` — conditional UPDATE via holdModel(aggType):
      `UPDATE ... SET payout_settled_at = now WHERE id = ? AND payout_settled_at IS NULL`.
      `settleRelease(db, aggType, id) error` = `if err := releaseMoney(...); err != nil { return err }; return stampPayoutSettled(...)`.
      `settleReverse(db, aggType, id) error` = same over `reverseMoney`.
      Each helper < 50 lines; wrap errors with `%w`.
    - Wire: `ReleaseHold` returns `settleRelease(db, aggType, id)` instead of the bare
      `releaseMoney(...)` (line 230). `ReverseHold` returns `settleReverse(db, aggType, id)`
      instead of `reverseMoney(...)` (line 288). So a successful primary path is
      released + settled_at stamped; a post-commit seam failure returns the error to the
      caller AND leaves settled_at NULL for the reconcile to pick up.
  </action>
  <verify>
    <automated>cd apps/api && go build ./... && go test ./services/ -run 'StampPayoutSettled|PayoutReconcile_Reversed|PayoutReconcile_ReleasedAlreadySettled' 2>&1 | tail -5</automated>
  </verify>
  <done>Columns + migration exist; stampPayoutSettled/settleRelease/settleReverse compile; ReleaseHold/ReverseHold stamp only after seam==nil; stamp-once + already-settled-skip tests pass.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 3 (GREEN-2): make ReleaseDayPayout idempotent on re-drive</name>
  <files>apps/api/services/meal_plan_escrow.go</files>
  <behavior>
    `ReleaseDayPayout` (line 183) currently calls `rz.ReleaseTransfer` unconditionally and
    returns its error. There is NO single-transfer FetchTransfer on the client, so the
    idempotency is achieved by TWO layers: (a) the reconcile only re-drives rows with
    payout_settled_at IS NULL (never a completed release), and (b) tolerate the gateway
    "already released" path here. Since `ReleaseTransfer` is a PATCH `on_hold:false` that
    Razorpay treats as a no-op on an already-released transfer, keep the call but if it
    returns an error whose message indicates the transfer is already released/settled,
    log it and return nil rather than propagating (so a re-drive of a partially-applied
    release cannot loop forever). Keep the existing flag-off / empty-transfer-id no-op
    guard (184) unchanged. Function stays < 50 lines; `%w` on genuine errors.
    Tradeoff note in the doc comment: string-matching the gateway error is the fragile
    part; the durable guard is payout_settled_at IS NULL (a settled day is never re-driven).
  </action>
  <verify>
    <automated>cd apps/api && go build ./... && go test ./services/ -run 'ReleaseDayPayout_Idempotent' 2>&1 | tail -5</automated>
  </verify>
  <done>ReleaseDayPayout tolerates an already-released transfer (returns nil, logs) and stays a no-op when the escrow flag is off; the idempotency test passes.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 4 (GREEN-3): reconcile cron + registration + attempt cap</name>
  <files>apps/api/services/payout_reconcile_cron.go, apps/api/services/cron_temporal.go</files>
  <behavior>
    New file payout_reconcile_cron.go mirroring payout_auto_confirm_cron.go structure:
    - `const payoutReconcileInterval = 10 * time.Minute`.
    - `const payoutReconcileMaxAttempts = 5` (alert cap).
    - `StartPayoutReconcileCron(ctx)`: go func → run once → ticker → select ctx.Done()/tick.
    - `runPayoutReconcileScan(_ context.Context)`: `recover()` at top; GATE — return
      immediately (pure no-op) unless `payoutMovementEnabled() || MealPlanEscrowActive()`;
      then run four bounded sweeps (Limit `sweepBatchLimit`), log a one-line summary if any
      row was driven.
    - Sweeps (log-and-continue per row; never let one bad row abort the batch):
      * released orders: `payout_hold_status='released' AND payout_settled_at IS NULL AND razorpay_order_id <> '' AND payout_settle_attempts < payoutReconcileMaxAttempts`
        → `settleRelease(database.DB, aggTypeOrder, id)`.
      * released meal_plan_days: same, guard `payout_transfer_id <> ''` → `settleRelease(..., aggTypeMealPlanDay, id)`.
      * reversed orders: `payout_hold_status='reversed' AND payout_settled_at IS NULL AND razorpay_order_id <> '' AND payout_settle_attempts < max` → `settleReverse(..., aggTypeOrder, id)`.
      * reversed meal_plan_days: same with `payout_transfer_id <> ''` → `settleReverse(..., aggTypeMealPlanDay, id)`.
    - Attempt cap: on a settle error, increment `payout_settle_attempts` for that row
      (`UPDATE ... SET payout_settle_attempts = payout_settle_attempts + 1 WHERE id = ?`)
      and, when it reaches the cap, emit an ALERT-level log line so ops can act on a
      permanently-bad transfer. Chosen the DB column (not an in-memory counter) because
      the cron may run as a fresh Temporal activity process each tick (an in-memory map
      would never persist across invocations); the column survives restarts and the
      Temporal per-activity model. Tradeoff: one extra small INT column vs. a counter that
      silently fails to cap under Temporal — documented in the file header.
    - Register in `cronJobs()` (cron_temporal.go, after the payout-auto-confirm entry):
      `{"payout-reconcile", payoutReconcileInterval, runPayoutReconcileScan, StartPayoutReconcileCron},`.
    Keep every func < 50 lines (split per-aggregate sweep helpers if needed, matching
    sweepOrders/sweepMealPlanDays). No panics; `%w` on wrapped errors.
  </action>
  <verify>
    <automated>cd apps/api && go build ./... && go test ./services/ -run 'PayoutReconcile' 2>&1 | tail -8</automated>
  </verify>
  <done>payout-reconcile cron drives released/reversed-but-unsettled holds, is a pure no-op when both flags are off, caps re-drives at 5 attempts with an alert log, and is registered in cronJobs(); all reconcile tests pass.</done>
</task>

<task type="auto">
  <name>Task 5: full verify (build, vet, service tests green)</name>
  <files>apps/api</files>
  <action>
    Run the full service test package and static checks. Confirm no regression in the
    existing payout hold / auto-confirm suites (they share setupHoldDB, which now carries
    the two new columns). Confirm the reconcile suite is green end to end.
  </action>
  <verify>
    <automated>cd apps/api && go build ./... && go vet ./services/ && go test ./services/ -run 'Payout|ReleaseDayPayout|StampPayoutSettled|Reconcile' -count=1 2>&1 | tail -12</automated>
  </verify>
  <done>go build + go vet clean; all Payout*/Reconcile/ReleaseDayPayout/StampPayoutSettled tests pass with -count=1.</done>
</task>

</tasks>

<threat_model>
Key invariant: the reconcile must be idempotent and flag-gated, and it must NEVER move
money the primary path did not already commit to (it only re-drives rows whose status was
already flipped to `released`/`reversed` by an admin/sweep action).

| Threat ID | Category | Component | Disposition | Mitigation |
|-----------|----------|-----------|-------------|------------|
| T-459-01 | Tampering (double-pay) | reconcile re-driving a row whose seam already succeeded | mitigate | Only select `payout_settled_at IS NULL`; `stampPayoutSettled` is a conditional UPDATE on `settled_at IS NULL`; `ReleaseOrderPayouts` re-filters `OnHold==true`; `ReleaseDayPayout` tolerates already-released. A settled row is never re-picked. |
| T-459-02 | Elevation (money moved while dark) | seam firing before escrow verified | mitigate | Whole scan gated on `payoutMovementEnabled() || MealPlanEscrowActive()`; both default OFF → pure DB no-op. Reconcile never invents a status transition — it only re-drives the seam for an already-committed status. |
| T-459-03 | Denial of service (unbounded loop / runaway) | a permanently-failing transfer retried forever | mitigate | `sweepBatchLimit` bounds each scan; `payout_settle_attempts < 5` cap + ALERT log stops the loop and surfaces the bad row to ops. |
| T-459-04 | Denial of service (batch abort) | one bad row aborting the whole sweep | mitigate | `recover()` at scan top + log-and-continue per row (mirrors auto-confirm sweep). |
| T-459-05 | Repudiation | stranded released-but-unpaid row invisible | mitigate | `payout_settled_at` makes "committed vs. actually-paid" auditable; cap-reached rows are ALERT-logged. |
</threat_model>

<verification>
- `cd apps/api && go build ./...` clean.
- `go vet ./services/` clean.
- `go test ./services/ -run 'Payout|Reconcile|ReleaseDayPayout|StampPayoutSettled' -count=1` all green.
- Manual read-through: `ReleaseHold`/`ReverseHold` stamp only after seam==nil; reconcile
  selects only `payout_settled_at IS NULL` rows; scan no-ops when both flags off.
</verification>

<success_criteria>
- `payout_settled_at` (+ `payout_settle_attempts`) exist on Order and MealPlanDay with a
  real ALTER migration pair.
- One shared settle helper stamps `payout_settled_at` after a successful seam, reused by
  both `ReleaseHold`/`ReverseHold` and the reconcile (no duplicated stamp logic).
- `ReleaseDayPayout` re-drive on an already-released transfer does not error.
- `payout-reconcile` cron re-drives `released`/`reversed`-but-unsettled holds, is flag-gated
  (pure no-op when both off), bounded, attempt-capped, and registered in `cronJobs()`.
- All new + existing service tests green; scope limited to #459 (no #456-#462 work).
</success_criteria>

<output>
Follow-ups to file as GitHub issues (do NOT implement here): #456 group-order gate,
#457 refund/dispute cross-guard, #458 disputed dead-end, #460 races, #461 auth,
#462 medium cluster. Also note: adding a real single-transfer `FetchTransfer` to the
Razorpay client would let `ReleaseDayPayout` verify on-hold state instead of string-matching
the gateway error — a cleaner but out-of-scope hardening.
</output>
