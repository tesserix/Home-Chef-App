# Handoff — Vendor-Payout Prod Readiness (2026-07-05, session 4)

Continues `HANDOFF-2026-07-05-prod-readiness-session3.md`. Goal unchanged: make the escrow control
plane ready to flip `ORDER_PAYOUT_AUTO_RELEASE_ENABLED` / `MEAL_PLAN_ESCROW_ENABLED` ON. Both still
OFF (no live money). Epic #403.

## Shipped this session (all MERGED to main; `origin/main` HEAD = `f21d6630`)

1. **#394 (P1) — meal-plan-day / group double-refund guard** (PR #543). `InitiateRefund` now returns
   **422** for orders spawned by a typed escrow flow (meal-plan day via `MealPlanDay.OrderID`, group via
   `GroupOrder.OrderID`) — the generic `refund:<orderID>` keyspace was disjoint from
   `mealplan-refund:<dayID>` / `grouporder-refund:<id>` and bypassed `Order.RefundAmount`, so the same
   money could be refunded once via each path AND the held direct transfer was never reversed. New
   `services.TypedRefundOrderKind`. The `issue:<id>` flow was already safe (caps at + increments
   `RefundAmount`).
2. **#462 item 3 — chef NET payout in admin queue** (PR #545). `ListPendingPayouts` now surfaces
   `NetPayout` (the actual held-transfer amount: order=`ComputeOrderEarnings.NetPayout`,
   day=`perDayNetPayout`, group=`GroupChefPayout`) alongside the gross `Amount`. Additive/non-breaking.
3. **#542 — reconcile stranded refunded/cancelled holds** (PR #548). `reconcileCancelledOrders` +
   `reconcileRefundedDays` mirror `reconcileCancelledGroups` (#534) — recover an order/day left at a
   parked hold with a held transfer after a crashed refund cross-guard. Wired into
   `runPayoutReconcileScan`; runs only when a flag is ON.
4. **#524 / #396 Phase 1 — `ToPaise` rounds** (PR #550). `int(math.Round(amount*100))` instead of
   truncating; matches `ToMinor(_,"INR")` exactly. Stops the 1-paise under-settle on the Route path
   (#518). Phase 2 (integer-paise columns + `services/money` largest-remainder allocation + exact
   reconciliation) stays open in **#396**.
5. **#546 (P1) — group-order chef payout nets commission + TDS** (PR #551). `groupNetPayout` mirrors
   `perDayNetPayout` (`gross=subtotal+tax`, `commission=rate×subtotal`, `tds=RateTDS×gross`).
   `HoldGroupChefPayout` + the admin-queue group `NetPayout` both corrected. Was paying the gross slice
   with zero platform commission/TDS. Verifier note: `group.Tax`/`order.Tax` both include GST on
   delivery+service in the chef gross — pre-existing shared characteristic, not touched here.
6. **#398 stuck-day sweep** (PR #552). `sweepStuckDays` auto-refunds confirmed meal-plan days that
   could never generate an order (customer has no default address) once 24h past their date, and marks
   them refunded so the plan can finish. Claim-first guarded UPDATE + refund-in-tx = money-safe (never
   refunded-and-fulfilled). Also widened `completeFinishedPlans` to scan Confirmed (a fully address-stuck
   plan never flips to Active, so it would never complete). **#398 stays OPEN** — remaining scope below.

7. **#395 item 2 — verify/webhook race status-guard** (PR #553). Extracted `completeRazorpayOrderTx`
   (conditional `WHERE payment_status <> 'completed'` + gate chef-notify AND `order.paid` on the single
   transition). Also killed a pre-existing unconditional every-verify `order.paid` duplicate. **Key
   technique: extracting the race-prone logic into a DB-conditional helper made it deterministically
   unit-testable WITHOUT a live-gateway/razorpay double** — reuse this to unblock the other verify/refund
   items behind the razorpay test wall. **#395 stays OPEN** (only item 2 landed).

Issues CLOSED: #394, #462 (comment — only deferred items 2/6 remain), #542, #524, #546. **#398 and #395
still OPEN** (only their sharpest sub-pieces landed).

## Razorpay test-double wall (important for the remaining money items)
Several remaining gate items touch code behind a LIVE Razorpay client (`GetRazorpay()` is nil in tests,
and `razorpayBaseURL` is a hard-coded const so the client can't be pointed at an httptest server): the
verify-path FetchPayment, `RefundDay`'s `ReverseTransfer`, tip/group `VerifyTip`/`VerifyGroupShare`
signature checks. Two ways forward: (a) **extract the race/logic core into a DB-only helper and test that
directly** (what #553 did — preferred, no prod change), or (b) make `razorpayBaseURL` an injectable field
on `RazorpayClient` (2-line change: `baseURL` field + default in `doRequest`) + a test client pointing at
`httptest.Server`, which unblocks end-to-end testing of the whole money surface. Prefer (a) per-item;
consider (b) if several items need it.

## New follow-ups filed this session
- **#544** — cancel/reject paths (`CancelOrder` / `UpdateOrderStatus` reject / chef `CancelOrder` via
  `RefundOrderForCancellation`) desync `Order.Status` from `MealPlanDay`/`GroupOrder` status and share
  #394's gap. Currently inert (no `RazorpayPaymentID` on those rows) → P3.
- **#546 (P1)** — **group-order chef payout pays `subtotal+tax` GROSS, no commission/TDS deducted**
  (`GroupChefPayout`), unlike orders/meal-plan-days. Real leak once flags flip ON. Blocks flags-ON.
- **#547 (P3)** — meal-plan-day / group-order commission rate isn't frozen (orders freeze
  `Order.CommissionRate`); the pending-queue day net uses the *current* flat rate → few-paise drift if
  the rate changes mid-flight.
- **#549 (P2)** — partial chef "goodwill" refund forfeits the chef's **entire** payout hold
  (over-withholding); pre-existing in `chef_order_cancel.go`/`payment.go` via the #457 cross-guard.

## Remaining before-flags-ON gate (recommended order)

1. **#398 (P1) remaining** — stuck-day sweep is DONE (#552). Left: (a) **failed-reversal retry queue** —
   `RefundDay`/`RefundDeclinedDays`/`RefundUndeliveredDays` swallow a failed `ReverseTransfer` (log +
   continue crediting the wallet); a failed reversal strands the held transfer at `payout_hold_status=""`,
   invisible to the #542 reconcile (`parkedActionableHolds` excludes `""`) — needs durable retry + alert.
   (b) drift-ledger reconcile cron (assert `captured = released + reversed + held + refunded` vs Razorpay).
   (c) sandbox-verify Razorpay auto-reverse (#218). (d) Stripe refund amounts in `reconciliation.go`.
2. **#554 (P1, NEW)** — `settleWalletTopUps` issues Razorpay transfers with NO idempotency key → a
   repeated verify can double-transfer platform funds. Real money double-spend; blocks flags-ON. Sharp.
3. **#395 remaining** — item 2 (verify race) DONE (#553). Left: item 1 (unique indexes, needs tesserix-k8s
   migration), item 3 (webhook-only wallet settlement), item 4 (tip/group signature+amount parity —
   behind the razorpay test wall), item 5 (mutex-guard signature reads), item 6 (capacity release on
   payment.failed — testable, no razorpay). #555 (Stripe/wallet-only `order.paid` double-emit — reuse the
   #553 helper pattern, provider-generic).
3. **#392 remainder** — Stripe parity, cancellation-fee policy, `refund_amount` stamping, idempotency
   keys (2/6 done by #526).
4. **#527** — per-line then full-cancel strands remaining live items' money.
5. **#400 remaining** — per-day meal-plan issue REPORTING surface + vendor-fault accounting
   (full-clawback vs goodwill at resolve). Dispute-freezes-hold core is DONE.

## Payment integrity before real money
- **#395 (P1)** — DB unique indexes on `orders.razorpay_order_id/payment_id/stripe_payment_intent_id`
  (+`meal_plans.razorpay_order_id`); status-guard the verify path; webhook-only completion skips wallet
  settlement. Cheap + high-value.
- **#397 (P1)** — money-movement audit trail. NOTE: `setupCrossguardDB` test harness's `audit_logs`
  table is missing a `user_agent` column, so `LogSystemAudit` best-effort-fails in those tests (benign,
  pre-existing) — worth fixing the harness when touching audit.

## #462 leftovers status (re-audited against main this session)
Items 1 (GST rounding) & 4 (release idempotency) already fixed on main; 5 (`BulkReleasePayouts`)
reworked/gone; **2 (partial `ReverseTransfer`) and 6 (wallet-only reconciliation) remain deferred** —
item 2 is a partial-fault feature tied to #400/#549, item 6 folds into #398. #462 can close once those
land elsewhere.

## Go-live sequence (after the gate clears)
#218 Razorpay sandbox sign-off → #25 live switch + register live webhook → #200/#8 E2E lifecycle.
#389/#30 disbursement automation later.

## Workflow rules (UNCHANGED — kept catching real bugs again this session)
- Money-path = **plan → RED-first TDD → GREEN → independent adversarial verify** (a go-developer
  subagent given the diff + specific attack points). Do NOT skip verify. It caught the day-`Net>Amount`
  doc error (#462), confirmed the partial-refund over-withhold is pre-existing (#542), and the
  tips.go un-rounded-sum edge (#524).
- **Isolated git worktree per branch**, always: `git worktree add -b <branch> <scratchpad>/wt-x
  origin/main`; all ops via `git -C <worktree>`. A plain checkout collides with the concurrent session
  (`reference_shared_worktree_head_collision`). Remove the worktree after merge.
- **`gh auth switch --hostname github.com --user mahesh-sangawar` before EVERY gh write** (reverts to
  Civica EMU otherwise).
- sqlite harnesses use raw `CREATE TABLE`: `setupReleaseDB`/`setupCrossguardDB`/`setupHoldDB`/
  `setupChefOrderDB`+`postStatus`, `flagsOff`. When a query gains columns, patch EVERY harness that
  exercises it (a missed one = "no such column" 500 → CI red; happened once with #462's net columns —
  patched `setupReleaseDB`, `setupCrossguardDB`, `admin_payout_test`). Run `go test ./...` (all
  packages) before pushing to catch missed harnesses.
- Amend the fix INTO the commit before pushing; wait for CI green (Build + Run Tests + GitGuardian)
  before merging.

## Also pending (non-payments)
PRs #528 (@Sam123ben vendor feed) and #357 (dependabot) — not ours. #430 (auth logout), #334
(integration review + Playwright E2E). Mobile: promote the chef-page redesign OTA to production once
Menu/Reviews tabs are QA'd on TestFlight.
