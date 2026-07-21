# Handoff — Vendor-Payout Prod Readiness (2026-07-05, session 2 continued)

Continues `HANDOFF-2026-07-05-prod-readiness.md`. Goal: make the escrow control plane ready to flip
`ORDER_PAYOUT_AUTO_RELEASE_ENABLED` / `MEAL_PLAN_ESCROW_ENABLED` ON. Both still OFF (no live money).

## Shipped this session (PRs OPEN, awaiting Sam's merge)
All went plan → plan-check → RED-first TDD → independent verify (both reviews PASS clean):
- **#515 → PR #517** — `/admin/payouts/*` gated on new `SPManagePayouts` staff-permission (super-admin
  only). Part 1 (remove `computeLegacy`) was ALREADY done by Sam (65af3095, on main+deploy); deploy-gate
  confirmed (tesserix-home #26 live: main HEAD = 6d4b0f1 since 2026-07-04).
- **#498 → PR #521** — day/group disputed-hold drive: reject fan-out (disputed→release_eligible by
  order_id) + refund fan-out (→withheld/reversed) + refunded-day double-pay gap closed in `RefundDay`
  (state-only `markRefundedDayHold`). Also delivers the per-day-meal-plan-issue freeze part of #400.
- **#518 → PR #522** — meal-plan-day chef payout now pays NET (was GROSS: `d.Price`, no commission/TDS).
  `perDayNetPayout` mirrors `ComputeOrderEarnings`. Filed follow-ups: **#523** (meal-plan earnings/TDS
  invisible to weekly statement + Form 16A — before-flags-ON gate) and **#524** (`ToPaise` truncates).

## Remaining gate items — PLANNED, not started (each needs plan-check + RED-first + verify)
- **#508** — money-seam re-read race. Plan written: `.planning/quick/260705-508-money-seam-reread-race/PLAN.md`.
  `releaseMoney`/`reverseMoney` dispatch on the flip the goroutine won without re-reading current status;
  a concurrent refund flipping released→reversed can interleave → paid-after-claw. Fix: `settlePayout`
  re-reads terminal status and dispatches by it; reconcile backstop; idempotent seams. **Issue explicitly
  says must-fix before #218/#25.** Hardest remaining item — do it fresh with concurrency tests.
- **W-A** — `CancelGroupOrder` (`handlers/group_order.go:874`) calls `ReverseGroupChefPayout` OUTSIDE the
  guarded tx and doesn't drive the group `payout_hold_status`. Route the reverse through `ReverseHold`
  inside the cancel tx (flag-gated TODO already in code). Independently re-confirmed by the #498 verify.
- **#462 leftovers** — partial-amount reverse (`ReverseTransfer(id, amountPaise)`); pending-queue `Amount`
  shows gross not captured/net; provider/delivery webhook replay (add ts window / (provider,event-id)
  dedup — security, worth doing regardless of flags); wallet-only orders (`razorpay_order_id==''`) bypass
  the hold pipeline — add a reconciliation check.

## TRIAGE of the broader payment backlog — launch-blocking assessment

**CRITICAL distinction:** the escrow flags gate the hold→release→transfer seams. But several bugs live in
the REGULAR Razorpay refund/cancellation paths that run REGARDLESS of the flags — those are live in prod
NOW and are more urgent than the flag flip.

### LIVE NOW (not escrow-gated) — fix before/independent of the flag flip
- **#392 (P0) — customer self-cancel of a PAID order issues NO refund.** `OrderHandler.CancelOrder`
  (`handlers/orders.go:801`) allows cancel at `accepted` (already paid) but never refunds. Also the chef
  status-cancel path (`chefs.go:929` → saga, `ORDER_SAGA_ENABLED=false` default) refunds nothing.
  **Highest-impact live money bug. Top priority — not gated by escrow flags.**
- **#395 (P1) — payment integrity.** No DB unique indexes on `orders.razorpay_order_id/payment_id/
  stripe_payment_intent_id` (+`meal_plans.razorpay_order_id`); verify-path not status-guarded (duplicate
  chef notifications on verify/webhook race); webhook-only completion skips wallet settlement. DB
  constraints are cheap + high-value. Do before launch.
- **#397 (P1) — money-movement audit trail.** Needed for traceability before real money moves.

### ESCROW-FLAG-GATED — must-fix before flags ON
- **#508** (money-seam race), **W-A**, **#462 leftovers**, **#523** (meal-plan statement/TDS reporting),
  **#394** (P1: meal-plan day double-refund via disjoint idempotency keyspaces — path A `RefundDay` is
  escrow-gated; unify on `refund-day:<dayID>` or block generic refund on meal-plan-generated day orders),
  **#398** (escrow reconciliation: best-effort reverse → double-pay; paid days stuck with no default
  address strand funds forever; no transfer/hold reconciliation).
- **#400** — the "dispute freezes the hold" core is now DONE (#457/#458/#498 across order/day/group).
  Remaining: per-day meal-plan issue REPORTING surface (`ReportIssue` accept mealPlanDayId), vendor-fault
  accounting (full-clawback vs goodwill at resolve), support-ticket→order-issue bridge. Enhancements.

### MEAL-PLAN CUSTOMER LAUNCH (not payout-engine, but gates tiffin GA)
- **#402 (P1)** — mobile meal-plan booking shows food-only amount but charges full advance (GST+delivery).
  Customer-trust bug; fix before tiffin GA.

### DEFERRABLE (not launch-blocking)
- **#393** (RTO/returned-delivery money) — edge case. **#389** (statement mark-paid ledger) — weekly
  disburse is manual-first; the mark-paid UI already exists in tesserix-home (audit its backend);
  needed for record-keeping, not the flag flip. **#396/#524** (integer paise), **#401** (unimplemented
  refund surfaces) — P2.

## Recommended go-live sequence
1. **#392** (live P0 refund bug) — independent of everything, do first.
2. **#508 + W-A + #462 leftovers** (escrow seam correctness) → **#523** (statement/TDS reporting) →
   **#394 + #398** (double-refund + reconciliation) — the escrow-ON gate.
3. **#395 + #397** (payment integrity + audit) — before real money.
4. **#402** for tiffin customer GA.
5. Then #218 sandbox sign-off → #25 flag flip → #200/#8 E2E lifecycle. #389/#30 disbursement automation later.

## Workflow reminders (unchanged)
- Money-path = plan → plan-check → independent verify. RED-first callback-injection for check-then-act races.
- `gh auth switch --hostname github.com --user mahesh-sangawar` before every gh write.
- Stacked PRs: merge bottom-up WITHOUT `--delete-branch`, retarget children to main first.
- Repo was 2 commits stale at session start — always `git fetch` + reconcile local main vs origin first.
