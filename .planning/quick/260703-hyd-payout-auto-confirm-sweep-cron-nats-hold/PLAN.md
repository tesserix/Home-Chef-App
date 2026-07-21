---
phase: quick
plan: 260703-payout-auto-confirm-sweep-cron-nats-hold
type: tdd
wave: 1
depends_on: []
mode: quick
issue: "#387 follow-up (auto-confirm sweep + NATS hold events)"
branch: feat/payout-auto-confirm-sweep
autonomous: true
requirements: ["GH-387-followup-auto-confirm", "GH-387-followup-nats-hold-events"]
files_modified:
  - apps/api/services/nats.go
  - apps/api/services/payout_hold.go
  - apps/api/services/payout_auto_confirm_cron.go
  - apps/api/services/cron_temporal.go
  - apps/api/services/payout_hold_test.go
  - apps/api/services/payout_auto_confirm_cron_test.go

must_haves:
  truths:
    - "A delivered, dispute-free hold in awaiting_customer_confirmation past the confirm window is auto-advanced to release_eligible without any customer tap."
    - "A hold with an open OrderIssue is auto-advanced to disputed (never release_eligible) by the sweep."
    - "Every awaiting→release_eligible transition emits payments.hold_release_eligible via the outbox; every →disputed transition emits payments.hold_disputed — from BOTH the #387 confirm endpoints and the sweep."
    - "A hold still inside the confirm window is left untouched by the sweep."
    - "Idempotent/repeat runs do not double-emit events (guarded on RowsAffected>0 + the existing early-return)."
  artifacts:
    - path: apps/api/services/nats.go
      provides: "SubjectHoldReleaseEligible + SubjectHoldDisputed constants (payments.* → PAYMENTS stream)"
      contains: "SubjectHoldReleaseEligible"
    - path: apps/api/services/payout_auto_confirm_cron.go
      provides: "payoutAutoConfirmInterval, runPayoutAutoConfirmScan, StartPayoutAutoConfirmCron"
      contains: "func runPayoutAutoConfirmScan"
    - path: apps/api/services/payout_hold.go
      provides: "Event emission inside applyHoldConfirm on genuine transitions"
      contains: "EnqueueEvent"
  key_links:
    - from: apps/api/services/cron_temporal.go
      to: runPayoutAutoConfirmScan
      via: "cronJobs() row {\"payout-auto-confirm\", ...}"
      pattern: "payout-auto-confirm"
    - from: apps/api/services/payout_auto_confirm_cron.go
      to: ConfirmOrderHold / ConfirmMealPlanDayHold
      via: "per-row call inside the scan"
      pattern: "ConfirmOrderHold|ConfirmMealPlanDayHold"
    - from: apps/api/services/payout_hold.go
      to: EnqueueEvent
      via: "emit inside applyHoldConfirm when res.RowsAffected>0"
      pattern: "SubjectHoldReleaseEligible|SubjectHoldDisputed"
---

<objective>
Ship the #387 payout-hold follow-up, entirely behind the existing escrow flags (no live
behaviour change until they flip):

1. **Auto-confirm sweep cron** — a delivered, dispute-free hold in
   `awaiting_customer_confirmation` that has sat past the confirm window (default 24h,
   `GetCustomerConfirmWindowHours`) is auto-advanced to `release_eligible`; the customer
   never had to tap. Disputed holds auto-advance to `disputed`, never `release_eligible`.

2. **NATS hold events** — `payments.hold_release_eligible` and `payments.hold_disputed`
   emitted via the transactional outbox from inside the hold transition, so BOTH the
   #387 customer-confirm endpoints AND the sweep emit them for free.

Purpose: close the loop so a hold can leave `awaiting_customer_confirmation` without a
manual tap, and give #388 (admin release queue) a durable event to consume.
Output: 2 new subjects, event emission wired into the transition, a new sweep cron
registered in `cronJobs()`, all TDD-covered on the sqlite harness.

OUT OF SCOPE (follow-ups, do NOT touch):
- Admin release queue consuming `release_eligible` + Razorpay `ReleaseTransfer` → **#388**.
- OrderIssue→hold linkage that freezes an ALREADY `release_eligible` hold → **#400**.
- Mobile UI. Refund/reversal changes. Group-order holds.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
</execution_context>

<context>
@apps/api/services/payout_hold.go
@apps/api/services/payout_hold_test.go
@apps/api/services/meal_plan_cron.go
@apps/api/services/cron_temporal.go
@apps/api/services/nats.go
@apps/api/services/outbox.go

<interfaces>
<!-- Verified against the branch on 2026-07-03. Executor: use these directly. -->

apps/api/services/payout_hold.go:
  // line 79 — refactor target. Currently returns only .Error, no RowsAffected/status awareness.
  func applyHoldConfirm(tx *gorm.DB, model any, id uuid.UUID, disputed bool, now time.Time) error
  // line 96 / 120 — both wrap applyHoldConfirm in db.Transaction; both early-return when
  //   CustomerConfirmedAt != nil (this is what makes sequential re-runs a no-op).
  func ConfirmOrderHold(db *gorm.DB, order *models.Order) (models.PayoutHoldStatus, error)
  func ConfirmMealPlanDayHold(db *gorm.DB, day *models.MealPlanDay) (models.PayoutHoldStatus, error)
  func HasOpenOrderIssue(db *gorm.DB, orderID uuid.UUID) bool          // line 67
  func GetCustomerConfirmWindowHours(db *gorm.DB) int                  // line 181, default 24

apps/api/services/outbox.go:
  // line 68 — enqueue a generic Event onto the outbox WITHIN tx. userID becomes aggregate_id.
  func EnqueueEvent(tx *gorm.DB, subject, eventType string, userID uuid.UUID, data map[string]any) error

apps/api/services/nats.go:
  const SubjectPaymentSuccess = "payments.success"   // line 35 — add the two new payments.* subjects near here
  // setupStreams() already declares PAYMENTS stream over "payments.>" (line 220) —
  //   both new subjects route to it automatically. NO stream change needed.

apps/api/services/cron_temporal.go:
  func cronJobs() []cronJob    // line 25 — returns []cronJob{ {name, interval, run, ticker}, ... }
  // each job: {name string, interval time.Duration, run func(context.Context), ticker func(context.Context)}

Timestamp columns (both verified as real gorm columns):
  models.Order.DeliveredAt         *time.Time  json:"deliveredAt"  (order.go:132; written chefs.go:1003, delivery.go:600) → column delivered_at
  models.MealPlanDay.DeliveredAt   *time.Time  (meal_plan.go:121; written meal_plan_fulfillment.go:214) → column delivered_at
  models.Order.CustomerConfirmedAt / models.MealPlanDay.CustomerConfirmedAt  → column customer_confirmed_at
  Statuses: models.PayoutHoldAwaitingConfirmation, PayoutHoldReleaseEligible, PayoutHoldDisputed, PayoutHoldNone.

OutboxEvent columns (models/outbox.go): id, subject, msg_id (uniqueIndex), aggregate_type,
  aggregate_id, payload, status, next_retry_at, created_at, updated_at.
  Test DDL template (from loyalty_test.go:52):
    CREATE TABLE outbox_events (id TEXT PRIMARY KEY, subject TEXT, msg_id TEXT, aggregate_type TEXT,
      aggregate_id TEXT, payload TEXT, status TEXT, attempts INT, last_error TEXT,
      next_retry_at DATETIME, created_at DATETIME, updated_at DATETIME, published_at DATETIME)
  Assertion pattern used elsewhere:
    db.Raw(`SELECT count(*) FROM outbox_events WHERE subject = ?`, SubjectX).Scan(&n)
</interfaces>
</context>

<tasks>

<task type="tdd" tdd="true">
  <name>Task 1 (RED): failing tests for hold events + auto-confirm sweep</name>
  <files>apps/api/services/payout_hold_test.go, apps/api/services/payout_auto_confirm_cron_test.go</files>
  <behavior>
    Extend the existing harness `setupHoldDB` in payout_hold_test.go so emission-inside-transition
    can run and be asserted:
      - Add an `outbox_events` table to the DDL slice (use the loyalty_test.go template above).
      - Add `delivered_at DATETIME` to the `orders` CREATE TABLE (meal_plan_days already has it).
    Add assertions to the existing confirm tests (endpoint path proves events emit there too):
      - Test A: `TestConfirmOrderHold_AwaitingToReleaseEligible` → after confirm, exactly ONE
        outbox row with subject = SubjectHoldReleaseEligible.
      - Test B: `TestConfirmOrderHold_OpenIssueDisputes` → exactly ONE outbox row with
        subject = SubjectHoldDisputed, and ZERO with SubjectHoldReleaseEligible.
      - Test C: `TestConfirmOrderHold_Idempotent` → after the second (no-op) confirm, still
        exactly ONE SubjectHoldReleaseEligible row (no double-emit).

    New file payout_auto_confirm_cron_test.go (same package; may reuse setupHoldDB + database.DB
    swap pattern from TestSettleSaga_...):
      - Test D: an awaiting order whose delivered_at is BEFORE now-window → sweep advances it to
        release_eligible; one SubjectHoldReleaseEligible outbox row exists.
      - Test E: an awaiting order whose delivered_at is INSIDE the window (recent) → sweep leaves
        it awaiting_customer_confirmation; no event.
      - Test F: an awaiting order past the window WITH an open pending OrderIssue → sweep advances
        it to disputed (NOT release_eligible); one SubjectHoldDisputed outbox row.
      - Test G: awaiting meal_plan_day past the window (delivered_at set, customer_confirmed_at NULL)
        → advanced to release_eligible + event.
      - Test H: window override — set platform_settings payout.customer_confirm_window_hours=48,
        an order delivered 30h ago stays awaiting (inside 48h); default 24h path (Test D) advances.
      - Test I: run the scan twice → still exactly one event per hold (idempotent, no double-emit).
    Drive the sweep via `runPayoutAutoConfirmScan(context.Background())` with `database.DB` swapped
    to the sqlite db (restore in defer), mirroring TestSettleSaga_RegularOrder_HoldsNoRelease_FlagOn.
  </behavior>
  <action>
    Write the tests above FIRST. They reference symbols that do not exist yet
    (SubjectHoldReleaseEligible, SubjectHoldDisputed, runPayoutAutoConfirmScan) and outbox
    assertions the current code does not satisfy — so the package must fail to compile / tests
    must fail RED. Do NOT implement production code in this task.
    Use tabular figures? N/A — Go test only. Keep helpers small (<50 lines), immutable seed rows.
  </action>
  <verify>
    <automated>cd apps/api &amp;&amp; go test ./services/ -run 'ConfirmOrderHold|PayoutAutoConfirm' 2>&amp;1 | grep -Eiq 'undefined|FAIL|cannot|build failed' &amp;&amp; echo RED-OK</automated>
  </verify>
  <done>Tests compile-fail or fail on missing subjects/sweep + unmet outbox assertions (RED confirmed).</done>
</task>

<task type="tdd" tdd="true">
  <name>Task 2 (GREEN): add subjects + emit events inside the hold transition</name>
  <files>apps/api/services/nats.go, apps/api/services/payout_hold.go</files>
  <behavior>
    - nats.go: add near line 35 (with SubjectPaymentSuccess):
        SubjectHoldReleaseEligible = "payments.hold_release_eligible"
        SubjectHoldDisputed        = "payments.hold_disputed"
      (No stream change — PAYMENTS already covers "payments.>".)
    - payout_hold.go: emit the event INSIDE the same tx, only on a genuine transition
      (res.RowsAffected > 0), keyed generically over order vs meal-plan-day.
  </behavior>
  <action>
    Refactor `applyHoldConfirm` (line 79) to run the guarded UPDATE, inspect `res.RowsAffected`,
    and when > 0 emit the matching event within the same `tx`. It needs the aggregate identity, so
    add an `aggType string` parameter (values "order" / "meal_plan_day"); the `id uuid.UUID` is the
    aggregate id. Compute the target status once (disputed → PayoutHoldDisputed + SubjectHoldDisputed
    / eventType "payout.hold_disputed"; else PayoutHoldReleaseEligible + SubjectHoldReleaseEligible /
    "payout.hold_release_eligible"). On RowsAffected>0 call
      EnqueueEvent(tx, subject, eventType, id, map[string]any{"aggregate_type": aggType,
        "aggregate_id": id.String(), "payout_hold_status": string(target)})
    Extract a tiny helper `emitHoldEvent(tx, target, aggType, id, now)` to keep applyHoldConfirm <50
    lines. Update the two callers: `ConfirmOrderHold` passes aggType "order", `ConfirmMealPlanDayHold`
    passes "meal_plan_day". Wrap any error with `%w`. No panics. Emit-once rests on RowsAffected>0
    PLUS the existing `CustomerConfirmedAt != nil` early-return (sequential re-confirm never re-enters).
  </action>
  <verify>
    <automated>cd apps/api &amp;&amp; go test ./services/ -run 'ConfirmOrderHold|ConfirmMealPlanDayHold' 2>&amp;1 | tail -3</automated>
  </verify>
  <done>Confirm-endpoint tests (A/B/C + existing #387 tests) pass; events emit once per real transition.</done>
</task>

<task type="tdd" tdd="true">
  <name>Task 3 (GREEN): auto-confirm sweep cron + registration</name>
  <files>apps/api/services/payout_auto_confirm_cron.go, apps/api/services/cron_temporal.go</files>
  <behavior>
    New sweep, mirroring meal_plan_cron.go structure exactly:
      const payoutAutoConfirmInterval = 15 * time.Minute
      func StartPayoutAutoConfirmCron(ctx context.Context)   // go-routine: run once, then ticker + ctx.Done()
      func runPayoutAutoConfirmScan(ctx context.Context)     // panic-recover; logs advanced counts
    Scan logic:
      hours  := GetCustomerConfirmWindowHours(database.DB)
      cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)
      Orders: WHERE payout_hold_status = PayoutHoldAwaitingConfirmation
              AND delivered_at IS NOT NULL AND delivered_at <= cutoff
              AND customer_confirmed_at IS NULL   (Limit a bounded batch, e.g. 500)
      → for each, call ConfirmOrderHold(database.DB, &order) (applies dispute check + transition + event).
      MealPlanDays: same predicate on delivered_at → call ConfirmMealPlanDayHold(database.DB, &day).
    Register in cronJobs() (cron_temporal.go:26):
      {"payout-auto-confirm", payoutAutoConfirmInterval, runPayoutAutoConfirmScan, StartPayoutAutoConfirmCron},
  </behavior>
  <action>
    Create payout_auto_confirm_cron.go following meal_plan_cron.go's ticker+recover pattern (log
    "payout-auto-confirm-sweep: ..." lines). Keep runPayoutAutoConfirmScan <50 lines by extracting a
    helper per model (e.g. sweepOrders / sweepMealPlanDays) that returns a count. Disputed-vs-release
    is decided entirely inside ConfirmOrderHold/ConfirmMealPlanDayHold (they run HasOpenOrderIssue),
    so the sweep passes NO disputed flag — a disputed hold correctly lands in `disputed` and emits
    hold_disputed. Idempotent + race-safe: the guarded conditional UPDATE + RowsAffected>0 emit gate
    make repeated/concurrent runs safe. Add the cronJobs() row. Errors wrapped with %w; log-and-continue
    per row (one bad row must not abort the batch).
  </action>
  <verify>
    <automated>cd apps/api &amp;&amp; go test ./services/ -run 'PayoutAutoConfirm' 2>&amp;1 | tail -3</automated>
  </verify>
  <done>Sweep tests D–I pass; job registered in cronJobs(); disputed never reaches release_eligible.</done>
</task>

<task type="auto">
  <name>Task 4 (VERIFY): full package build + test + gofmt</name>
  <files>apps/api/services/</files>
  <action>Run the full services test suite + vet + gofmt to confirm no regression across #387 tests
    and the new work. Confirm functions stay <50 lines and no panics were introduced outside recover.</action>
  <verify>
    <automated>cd apps/api &amp;&amp; gofmt -l services/payout_hold.go services/payout_auto_confirm_cron.go services/nats.go services/cron_temporal.go &amp;&amp; go vet ./services/ &amp;&amp; go test ./services/ 2>&amp;1 | tail -5</automated>
  </verify>
  <done>gofmt clean (no files listed), go vet clean, `go test ./services/` passes green.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries
| Boundary | Description |
|----------|-------------|
| cron → hold state machine | Automated actor advances holds with no human in the loop; must honour the same dispute guard as the human confirm path. |
| transition → outbox → NATS | Emitted events drive #388 money movement downstream; a spurious/duplicate event is a financial-integrity risk. |

## STRIDE Threat Register
| Threat ID | Category | Component | Disposition | Mitigation |
|-----------|----------|-----------|-------------|-----------|
| T-387f-01 | Elevation / Tampering | sweep advancing a disputed hold to release_eligible | mitigate | Sweep passes NO disputed flag; ConfirmOrderHold/ConfirmMealPlanDayHold run HasOpenOrderIssue and the release UPDATE is guarded on `payout_hold_status = 'awaiting_customer_confirmation'` only — a disputed row fails the WHERE. Tests F asserts disputed lands `disputed`, C+G+H assert the guard. **KEY INVARIANT.** |
| T-387f-02 | Repudiation / Tampering | duplicate hold events (double release downstream) | mitigate | Emit only on `res.RowsAffected > 0` inside the tx + existing `CustomerConfirmedAt != nil` early-return; each outbox row carries a unique MsgID (uniqueIndex + JetStream dedup). Tests C + I assert exactly-once on re-run. |
| T-387f-03 | Denial of Service | unbounded sweep batch on a large backlog | mitigate | Bounded batch (Limit ~500) per scan; 15m interval; panic-recover; log-and-continue per row so one bad row cannot abort the batch. |
| T-387f-04 | Information disclosure | event payload leaking PII | accept | Payload carries only aggregate_type/id + hold status (no customer PII); PAYMENTS stream is internal. |
</threat_model>

<verification>
- `cd apps/api && go test ./services/` passes (all #387 tests + A–I).
- `gofmt -l` lists none of the touched files; `go vet ./services/` clean.
- Sweep advances past-window dispute-free holds → release_eligible + `payments.hold_release_eligible`.
- Sweep past-window + open issue → disputed + `payments.hold_disputed`, never release_eligible.
- Inside-window holds untouched; window default 24 honoured, override respected.
- Confirm endpoints emit the same events (Tests A/B); no double-emit on re-run (Tests C/I).
</verification>

<success_criteria>
- Two new subjects on the PAYMENTS stream; no stream config change needed.
- Event emission lives inside the transition (one place) → endpoints AND sweep both emit.
- `payout-auto-confirm` job registered in cronJobs(); behaviour still flag-gated (no live money moves).
- Minimal cohesive diff on branch feat/payout-auto-confirm-sweep; funcs <50 lines; %w; no stray panics.
</success_criteria>

<output>
After completion, create `.planning/quick/260703-hyd-payout-auto-confirm-sweep-cron-nats-hold/SUMMARY.md`.
</output>
