---
phase: quick
plan: m3c-refund-dispute-cross-guard
type: execute
issue: "#457"
priority: P0
branch: feat/refund-payout-hold-crossguard
wave: 1
depends_on: []
autonomous: true
tdd: true
files_modified:
  - apps/api/services/payout_release.go
  - apps/api/services/order_issue.go
  - apps/api/services/temporal_order.go
  - apps/api/handlers/chef_order_cancel.go
  - apps/api/handlers/payment.go
  - apps/api/services/payout_crossguard_refund_test.go
requirements:
  - "GH-457: no refunded/disputed order can be paid out to the chef"
must_haves:
  truths:
    - "Every order-refund entry point drives the payout hold (5 sites): RefundIssueToWallet, CompensateOrderRefund, chef RefundOrder, chef CancelOrder, and the PRIMARY PaymentHandler.InitiateRefund."
    - "Refunding an order whose hold is release_eligible/awaiting withholds it; whose hold is released reverses it (claw-back seam reached when flag ON, state-only when OFF)."
    - "ReleaseHold refuses (ErrHoldNotEligible) any aggregate whose underlying order is refunded, cancelled, has refunded_at set, or has a pending issue — for order, meal-plan-day AND group-order aggregates."
    - "The release-side backstop catches the issue path even though RefundIssueToWallet leaves orders.status='delivered' — because the guard also keys on refunded_at IS NOT NULL."
    - "The admin pending-order queue excludes refunded/cancelled orders AND orders with refunded_at set, and flags rows with an open issue."
    - "Double-refund never double-reverses (idempotent, race-safe conditional UPDATE); a second gateway reverse is gateway-rejected + logged non-fatal."
  artifacts:
    - path: apps/api/services/payout_release.go
      provides: "WithholdOrReverseOrderHoldForRefund helper + all-aggregate ReleaseHold pre-check (refunded_at-aware) + listPendingOrders refund exclusion + PendingPayout.HasOpenIssue"
    - path: apps/api/handlers/payment.go
      provides: "InitiateRefund wired to the hold cross-guard at both commit sites"
    - path: apps/api/services/payout_crossguard_refund_test.go
      provides: "cross-guard TDD suite on an extended sqlite harness (wallet tables seeded so the issue path actually runs)"
  key_links:
    - from: "RefundIssueToWallet / CompensateOrderRefund / chef RefundOrder / chef CancelOrder / PaymentHandler.InitiateRefund"
      to: "WithholdOrReverseOrderHoldForRefund"
      via: "best-effort call after each refund commits"
      pattern: "WithholdOrReverseOrderHoldForRefund"
    - from: "ReleaseHold(order|meal-plan-day|group-order)"
      to: "underlying order status/refunded_at + HasOpenOrderIssue"
      via: "pre-release guard returning ErrHoldNotEligible"
      pattern: "ErrHoldNotEligible"
---

<objective>
Cross-guard order refunds against the payout hold state machine (GH #457, P0). Today
refund and payout-hold are independent: an order can be refunded to the customer AND
released to the chef. Root causes (all re-verified against `apps/api`):

- `HasOpenOrderIssue` (services/payout_hold.go:81) counts only `status='pending'`, so an
  auto-refunded/resolved issue stops blocking → the hold advances release_eligible → released.
- `RefundIssueToWallet` (order_issue.go:167) sets `refunded_at`/`refund_amount` but NOT
  `orders.status` (stays `delivered`) — so a status-only guard never catches the issue path.
- `listPendingOrders` (services/payout_release.go:120) excludes nothing.
- `ReleaseHold`/`transitionHold` never re-check disputes/refunds at release time; and FIVE refund
  paths (`RefundIssueToWallet`, `CompensateOrderRefund`, chef `RefundOrder`, chef `CancelOrder`,
  and the PRIMARY `PaymentHandler.InitiateRefund` at payment.go:643 — route
  `POST /payments/order/:orderId/refund`, chef/admin-callable, reachable on an already-`released`
  order, no transfer claw-back) never touch `payout_hold_status`.
- The double-pay hole also exists for meal-plan-day and group-order holds whose UNDERLYING order
  (`meal_plan_days.order_id` / `group_orders.order_id`, both nullable) is refunded.

Fix, defense-in-depth: (1) a shared helper that drives the hold on every order refund;
(2) wire it into ALL FIVE ORDER refund choke points; (3) a release-side belt-and-suspenders guard
that is refunded_at-aware and covers all three aggregates; (4) surface an open-issue flag + refund
exclusion on the admin queue.

Purpose: a chef must never be paid for money the customer got back.
</objective>

<context>
@apps/api/services/payout_release.go
@apps/api/services/payout_hold.go
@apps/api/services/order_issue.go
@apps/api/services/temporal_order.go
@apps/api/handlers/chef_order_cancel.go
@apps/api/handlers/order_issue.go
@apps/api/handlers/payment.go
@apps/api/services/order_payout.go
@apps/api/services/payout_release_test.go
@apps/api/services/audit.go

<interfaces>
Reuse — do NOT fork the hold machine, add branches:

services/payout_release.go:
```go
const (aggTypeOrder = "order"; aggTypeMealPlanDay = "meal-plan-day"; aggTypeGroupOrder = "group-order")
var ErrHoldNotEligible = errors.New("payout hold not in an eligible state")
func transitionHold(db *gorm.DB, aggType string, id uuid.UUID, from []models.PayoutHoldStatus, to models.PayoutHoldStatus, emit bool) (bool, error)
func settleReverse(db *gorm.DB, aggType string, id uuid.UUID) error   // reverseMoney + stampPayoutSettled
func ReleaseHold(db *gorm.DB, aggType string, id uuid.UUID) error
func listPendingOrders(db *gorm.DB, f PendingFilter) ([]PendingPayout, error)
type PendingPayout struct { AggType string; ID uuid.UUID; ChefID uuid.UUID; Amount float64; HoldStatus models.PayoutHoldStatus; /* ... */ Context string }
type pendingRow struct { ID, ChefID string; Amount float64; PayoutHoldStatus string; /* ... */ Context string }
func toPending(aggType string, rows []pendingRow) []PendingPayout
```

services/payout_hold.go: `func HasOpenOrderIssue(db *gorm.DB, orderID uuid.UUID) bool // status='pending' only`

models: PayoutHoldNone/AwaitingConfirmation/ReleaseEligible/Released/Disputed/Withheld/Reversed;
`OrderStatusRefunded="refunded"`, `OrderStatusCancelled="cancelled"`;
`MealPlanDay.OrderID *uuid.UUID` (meal_plan.go:116, nullable), `GroupOrder.OrderID *uuid.UUID` (group_order.go:76, nullable).

The FIVE refund choke points to wire:
```go
services/order_issue.go:103    RefundIssueToWallet(db, issue *OrderIssue, amount, by string, resolvedBy *uuid.UUID) error   // issue.OrderID; auto handlers/order_issue.go:168 + admin :312; sets refunded_at but NOT status
services/temporal_order.go     CompensateOrderRefund(ctx, orderID, reason) error   // already calls ReverseOrderPayouts; still never flips payout_hold_status
handlers/chef_order_cancel.go  (*ChefOrderCancelHandler).RefundOrder   // goodwill, delivered, ~:343
handlers/chef_order_cancel.go  (*ChefOrderCancelHandler).CancelOrder   // in-flight full refund, ~:142
handlers/payment.go:643        (*PaymentHandler).InitiateRefund   // PRIMARY endpoint, route routes.go:783; gate "Can only refund completed payments"; TWO commit sites: to-wallet (~payment.go:784) + gateway-persist (~:938, sets status+payment_status=refunded)
```

Audit without gin.Context: `services.LogSystemAudit(nil, action, entityType, entityID, old, new)` — nil-guarded, writes via database.DB, best-effort.

Test harness (services/payout_release_test.go): `setupReleaseDB(t)` DDLs orders/meal_plan_days/meal_plans/group_orders/order_issues/... and swaps `database.DB = db`; helpers `seedOrderHold`, `loadDayHold`. Money seams hit `GetRazorpay()==nil` (no-op); flags OFF ⇒ pure state advance.
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1 (RED): cross-guard suite + extended harness</name>
  <files>apps/api/services/payout_crossguard_refund_test.go</files>
  <behavior>
    First EXTEND the harness so the issue-path test really runs (W5): add a local `setupCrossguardDB(t)`
    that reuses setupReleaseDB's DDL plus the columns/tables RefundIssueToWallet + CreditWallet touch —
    orders: `refund_amount`, `refund_reason`, `refund_initiated_by` (already partly present, add any
    missing); order_issues: `customer_id`, `reason`, `requested_amount`, `refund_amount`, `resolved_at`,
    `resolved_by`, `refund_txn_id`; and `wallets` + `wallet_transactions` tables matching what CreditWallet
    inserts/reads. If wiring CreditWallet's tables proves heavy, instead add a narrower issue-path test that
    stubs the credit yet still calls WithholdOrReverseOrderHoldForRefund on an auto_refunded issue's order —
    do NOT rely on a `-run` filter that passes vacuously with zero matched tests.

    Add `loadOrderHold(db, id)` and `seedPendingIssue(db, orderID)` helpers. Tests (all must FAIL / not
    compile against current code):

    Direct helper:
    - Crossguard_EligibleWithheld: hold=release_eligible → helper → 'withheld'; then ReleaseHold(order) → ErrHoldNotEligible.
    - Crossguard_AwaitingWithheld: hold=awaiting → helper → 'withheld'.
    - Crossguard_ReleasedReversed: hold=released → helper → 'reversed' (state-only; flags OFF). Assert payout_settled_at stamped once.
    - Crossguard_NoopStates: hold in {none,withheld,reversed,disputed} → no-op, nil.
    - Crossguard_Idempotent: two helper calls on a released order → still 'reversed', single transition.

    Wiring / issue path:
    - Crossguard_AutoRefundDrivesHold: order hold=release_eligible + pending issue → RefundIssueToWallet(db, issue, amt, "system", nil) → hold='withheld' AND ReleaseHold(order) → ErrHoldNotEligible. (Proves the pending-only HasOpenOrderIssue gap is closed AND that best-effort wiring fires.)

    Release-side backstop (refunded_at-aware — BLOCKER 2):
    - Crossguard_ReleaseHold_BlocksRefundedAtOnly: order status STILL 'delivered' but refunded_at set (the issue-path shape), hold=release_eligible → ReleaseHold(order) → ErrHoldNotEligible.
    - Crossguard_ReleaseHold_BlocksRefundedStatus: status='refunded' → ErrHoldNotEligible.
    - Crossguard_ReleaseHold_BlocksPendingIssue: delivered, pending issue, no refunded_at → ErrHoldNotEligible.
    - Crossguard_ReleaseHold_AllowsClean: delivered, no issue, no refunded_at → nil, hold='released'.

    All-aggregate pre-check (W4):
    - Crossguard_ReleaseHold_Day_BlocksRefundedOrder: meal_plan_day hold=release_eligible whose linked order (meal_plan_days.order_id) is refunded / refunded_at set → ReleaseHold(meal-plan-day) → ErrHoldNotEligible; a day with order_id NULL releases fine.
    - Crossguard_ReleaseHold_Group_BlocksRefundedOrder: group_order hold=release_eligible whose linked order (group_orders.order_id) is refunded → ReleaseHold(group-order) → ErrHoldNotEligible; NULL order_id releases fine.

    Queue:
    - Crossguard_ListPendingOrders_ExcludesRefunded: release_eligible orders with status refunded / cancelled / refunded_at-set / clean-delivered → only clean-delivered listed.
    - Crossguard_ListPendingPayouts_FlagsOpenIssue: two clean eligible orders, one with a pending issue → PendingPayout.HasOpenIssue true only for that row.
  </behavior>
  <action>Create the test file + extended harness. Confirm RED (compile failure on missing helper/fields is valid RED). No production code here.</action>
  <verify>
    <automated>cd apps/api &amp;&amp; go test ./services/ -run Crossguard 2>&1 | grep -qiE "undefined|FAIL|cannot" &amp;&amp; echo RED-OK</automated>
  </verify>
  <done>Suite + harness exist; every test fails/does-not-compile; AutoRefundDrivesHold actually executes (not vacuous).</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2 (GREEN core): WithholdOrReverseOrderHoldForRefund + reverse-on-released</name>
  <files>apps/api/services/payout_release.go</files>
  <behavior>
    - release_eligible | awaiting_customer_confirmation → transitionHold(..to withheld).
    - released → transitionHold(..to reversed) then settleReverse (claw back; stampPayoutSettled keeps payout_settled_at reconcile-safe).
    - none | withheld | reversed | disputed → no-op, nil.
    - Idempotent + race-safe via conditional UPDATE (RowsAffected); audit-logged with reason.
  </behavior>
  <action>
    Add `func WithholdOrReverseOrderHoldForRefund(db *gorm.DB, orderID uuid.UUID, reason string) error`
    (keep &lt;50 lines). Select only `payout_hold_status`; switch:
      - ReleaseEligible / AwaitingConfirmation → transitionHold(db, aggTypeOrder, orderID, []{those two}, PayoutHoldWithheld, false)
      - Released → ok,err := transitionHold(db, aggTypeOrder, orderID, []{Released}, Reversed, false); if ok && err==nil { err = settleReverse(db, aggTypeOrder, orderID) }
      - default → return nil
    On a real transition, best-effort `services.LogSystemAudit(nil, "payout.hold.refund_crossguard", "order",
    orderID.String(), oldStatus, map[string]any{"to": newStatus, "reason": reason})`. Wrap errors `%w` with a
    "payout-crossguard:" prefix; return the wrapped load error if the order row is missing (callers log best-effort).
  </action>
  <verify>
    <automated>cd apps/api &amp;&amp; go test ./services/ -run 'Crossguard_(EligibleWithheld|AwaitingWithheld|ReleasedReversed|NoopStates|Idempotent)' -count=1</automated>
  </verify>
  <done>Helper matches the state table; 5 direct-helper tests pass; go vet clean.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 3 (GREEN wiring): drive the hold from ALL FIVE order refund paths</name>
  <files>apps/api/services/order_issue.go, apps/api/services/temporal_order.go, apps/api/handlers/chef_order_cancel.go, apps/api/handlers/payment.go</files>
  <behavior>Every order refund commit drives the hold exactly once, best-effort + loudly logged; a hold-transition failure never fails the refund (release-side guard + reconcile are the backstop).</behavior>
  <action>
    - RefundIssueToWallet (order_issue.go): capture the tx result into `err`; on `err == nil` (real credit committed,
      NOT ErrNothingToRefund) call `WithholdOrReverseOrderHoldForRefund(db, issue.OrderID, "order issue refund: "+string(issue.Reason))`;
      log.Printf on its error but still `return err`. Single choke point covers auto (:168) + admin (:312).
    - CompensateOrderRefund (temporal_order.go): after the final Updates succeed, call the helper best-effort; log on error,
      return the Updates error unchanged. Add comment: the earlier ReverseOrderPayouts already clawed the Route split; the
      helper's released→reversed path may call ReverseOrderPayouts again, which is SAFE because the second reverse is
      GATEWAY-REJECTED and logged non-fatal (order_payout.go:108 — "already reversed errors on Razorpay; logged, not fatal"),
      NOT because stampPayoutSettled skips it. (W3 rationale correction.)
    - RefundOrder (chef_order_cancel.go ~:343): after the state save succeeds, before the audit log, call the helper
      best-effort with req.Reason; log on failure; do not change the 200.
    - CancelOrder refund path (chef_order_cancel.go ~:142): after the cancelled+refund Updates succeed, call the helper
      best-effort with the cancel reason.
    - InitiateRefund (payment.go:643) — the PRIMARY endpoint, reachable on a `released` order (only gates on
      payment_status==completed): add the helper at BOTH commit sites —
        (a) to-wallet branch, after the wallet-refund tx commits (~payment.go:784, near the LogSystemAudit "order.refund.to_wallet");
        (b) gateway-persist branch, after the tx that sets status=refunded commits (~:938).
      Both best-effort: `WithholdOrReverseOrderHoldForRefund(database.DB, order.ID, req.Reason)`; log.Printf +
      services.CaptureBackgroundError on error; never change the HTTP response.
    All call sites are order-scoped (helper uses aggTypeOrder); a few lines each; no new panics; `%w` on wrapped errors.
  </action>
  <verify>
    <automated>cd apps/api &amp;&amp; go build ./... &amp;&amp; go test ./services/ -run 'Crossguard_AutoRefundDrivesHold' -count=1</automated>
  </verify>
  <done>All FIVE refund paths call the helper best-effort; W3 comment corrected; auto-refund test proves the gap is closed; build green.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 4 (GREEN release-side): refunded_at-aware, all-aggregate ReleaseHold guard + queue exclusion + HasOpenIssue</name>
  <files>apps/api/services/payout_release.go</files>
  <behavior>
    - ReleaseHold on ANY aggregate (order, meal-plan-day, group-order) whose underlying order is refunded/cancelled OR has refunded_at set OR has a pending issue → ErrHoldNotEligible (no state change).
    - listPendingOrders never returns refunded/cancelled orders NOR orders with refunded_at set.
    - PendingPayout carries HasOpenIssue for order rows.
  </behavior>
  <action>
    - Add `HasOpenIssue bool `json:"hasOpenIssue"`` to PendingPayout and `HasOpenIssue bool` to pendingRow.
    - listPendingOrders: add `.Where("status NOT IN ?", []string{string(models.OrderStatusRefunded), string(models.OrderStatusCancelled)})`
      AND `.Where("refunded_at IS NULL")` (BLOCKER 2 backstop — the issue path leaves status='delivered' but sets refunded_at).
      Add `EXISTS(SELECT 1 FROM order_issues oi WHERE oi.order_id = orders.id AND oi.status='pending') AS has_open_issue`
      to the Select; set it in toPending for order rows (day/group default false).
    - Shared guard `orderRefundBlocks(db, orderID uuid.UUID) (bool, error)`: load order `status`,`refunded_at`; return true if
      status in {refunded,cancelled} OR refunded_at != nil OR HasOpenOrderIssue(db, orderID).
    - `releaseBlockedForAgg(db, aggType, id) (bool, error)` (W4):
        order → orderRefundBlocks(db, id)
        meal-plan-day → load meal_plan_days.order_id; nil → (false,nil); else orderRefundBlocks(db, *orderID)
        group-order → load group_orders.order_id; nil → (false,nil); else orderRefundBlocks(db, *orderID)
    - ReleaseHold: BEFORE transitionHold, call releaseBlockedForAgg for the aggType; if blocked → return ErrHoldNotEligible.
      Keep ReleaseHold &lt;50 lines via the extracted helpers. (Queue exclusion for day/group aggregates stays a lighter
      follow-up — the ReleaseHold pre-check is the hard backstop for all three; note it in success criteria.)
  </action>
  <verify>
    <automated>cd apps/api &amp;&amp; go test ./services/ -run 'Crossguard_(ReleaseHold_|ListPendingOrders_ExcludesRefunded|ListPendingPayouts_FlagsOpenIssue)' -count=1</automated>
  </verify>
  <done>ReleaseHold blocks refunded/refunded_at/disputed underlying orders across all three aggregates; queue excludes refunded/cancelled/refunded_at; HasOpenIssue populated.</done>
</task>

<task type="auto">
  <name>Task 5 (VERIFY): full suite + vet + build</name>
  <files>apps/api/services/payout_crossguard_refund_test.go</files>
  <action>
    Run the whole services + handlers suites (nothing regressed — payout_hold/payout_release/order_issue/reconcile/payment
    tests still green), vet, build. Confirm every Crossguard test passes AND actually executed (grep the run count — no
    vacuous zero-match). W6 (note only, do not wire): `CancelOrderItem` per-line refund is safe (pre-delivery, hold=none)
    and the `handleRefundProcessed` webhook out-of-band refund (status stays delivered) is a follow-up edge case — leave a
    `// TODO(#457-followup)` comment listing both, plus RefundGroupParticipant→hold-reverse as the deferred group item.
  </action>
  <verify>
    <automated>cd apps/api &amp;&amp; go vet ./... &amp;&amp; go test ./services/ ./handlers/ -count=1 &amp;&amp; go build ./...</automated>
  </verify>
  <done>All services + handlers tests pass, vet clean, build green.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| customer refund → payout ledger | Any refund (customer got money back) must invalidate the chef payout for that order across all five refund entry points. |
| admin release → hold machine | Admin release must not pay out a refunded/disputed order — for order, meal-plan-day, or group-order aggregates. |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-457-01 | Elevation/Integrity | refund vs. payout hold | mitigate | ALL FIVE order-refund paths (incl. the primary InitiateRefund at both commit sites) drive the hold via one shared helper; no refund leaves the hold untouched. |
| T-457-02 | Integrity (double-spend) | ReleaseHold at release time | mitigate | Refunded_at-aware, all-aggregate pre-check rejects refunded/cancelled/refunded_at/pending-issue underlying orders (ErrHoldNotEligible), closing both the issue-path status-gap and the late-dispute gap. |
| T-457-03 | Tampering (race/replay) | concurrent refund + release / double refund | mitigate | transitionHold conditional `WHERE payout_hold_status IN (from)` + RowsAffected; a second gateway reverse is gateway-rejected + logged non-fatal (order_payout.go:108); settleReverse stamp is idempotent. |
| T-457-04 | Information disclosure | admin queue | mitigate | Queue excludes refunded/cancelled/refunded_at orders and flags open-issue rows so an admin cannot release a disputed/refunded order. |
| T-457-05 | Repudiation | why the hold moved | mitigate | LogSystemAudit records "payout.hold.refund_crossguard" with old/new status + reason on every real transition. |

Key invariant: **no refunded/disputed order (or any aggregate whose underlying order is refunded) can be released; a refund always drives the hold; all transitions are idempotent and race-safe.**
</threat_model>

<verification>
- `cd apps/api && go test ./services/ ./handlers/ -count=1` — full suites green incl. the new cross-guard suite and untouched payout_hold/payout_release/order_issue/reconcile/payment tests.
- `go vet ./...` clean; `go build ./...` green.
- Manual trace: RefundIssueToWallet / CompensateOrderRefund / RefundOrder / CancelOrder / InitiateRefund(×2 commit sites) each call WithholdOrReverseOrderHoldForRefund exactly once, best-effort.
</verification>

<success_criteria>
- FIVE wiring sites confirmed (RefundIssueToWallet, CompensateOrderRefund, chef RefundOrder, chef CancelOrder, PaymentHandler.InitiateRefund at both commit sites).
- ReleaseHold refunded_at backstop closes the issue path (status stays 'delivered'); all-aggregate pre-check (order + meal-plan-day + group-order) blocks release on a refunded underlying order.
- Auto-refund (RefundIssueToWallet, by="system") drives the hold — the AutoRefundDrivesHold test actually runs (extended harness), not vacuously.
- Admin queue excludes refunded/cancelled/refunded_at orders and shows HasOpenIssue.
- Idempotent double-refund does not double-reverse; W3 rationale (gateway-rejected second reverse) is in the code comment.
- Generic hold infra reused (branches added, not forked); funcs &lt;50 lines; `%w` wrapping; no handler panics; race-safe conditional updates.
- Follow-ups (noted, NOT wired): RefundGroupParticipant→hold-reverse parity; day/group queue-exclusion (ReleaseHold pre-check already covers them); CancelOrderItem per-line refund (safe: pre-delivery hold=none); handleRefundProcessed webhook out-of-band refund; partial-amount reversal (#462, full-reverse only here); #458/#460/#461/#462.
</success_criteria>

<output>
After completion, create `.planning/quick/260703-m3c-refund-dispute-cross-guard-refund-drives/SUMMARY.md`.
</output>
