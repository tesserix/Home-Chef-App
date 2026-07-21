# Handoff — Vendor-Payout Prod Readiness (2026-07-05, session 3)

Continues `HANDOFF-2026-07-05-prod-readiness-session2.md`. Goal unchanged: make the escrow control
plane ready to flip `ORDER_PAYOUT_AUTO_RELEASE_ENABLED` / `MEAL_PLAN_ESCROW_ENABLED` ON. Both still
OFF (no live money). Epic #403.

## State at end of session 3 — the whole payout-gate PR queue is now MERGED to main

`origin/main` HEAD ≈ `ca7ee7c0`. This session merged all outstanding payout PRs (they were open awaiting
merge): **#517** (#515 SPManagePayouts gate), **#526** (#392 live P0 refund), **#530** (#508 money-seam
re-read), **#521** (#498 day/group disputed drive), **#522** (#518 meal-plan-day NET payout), **#532**
(#456 W-A group-cancel reverse — needed a rebase + conflict resolution, took #530's tolerant
`!isAlreadyReversedErr` version), **#541** (#534 reconcile stranded cancelled-group holds + park/mark
cancelled-terminal guards + optimistic chef status UPDATE). Also #462 webhook-replay dedup (landed via
the vendor-UI merge #539), plus #439 (CI) and #443 (UPI checkout).

Issues closed this session: #515, #498, #518, #456, #508, #534. **#392 left OPEN** (only 2 of 6 scope
items done by #526 — remaining: Stripe parity, cancellation-fee policy, `refund_amount` stamping,
idempotency keys; see the issue comment). New P3 follow-ups filed: **#540** (meal-plan spawned-order tax
basis vs day-transfer), **#542** (mirror `reconcileCancelledGroups` for orders + meal-plan-days).
**#523 was closed as invalid** — meal-plan income is already reported via the spawned per-day orders, so
adding meal_plan_days to the statement/breakdown/Form-16A would double-count (see the #523 comment).

## Remaining before-flags-ON gate (recommended order)

1. **#394 (P1)** — meal-plan-day double-refund via disjoint idempotency keyspaces. Path A `RefundDay` is
   escrow-gated; unify on `refund-day:<dayID>` or block generic refund on meal-plan-generated day orders.
   Independent of everything — good first pick.
2. **#462 leftovers** — partial-amount `ReverseTransfer(id, amountPaise)`; pending-queue `Amount` shows
   gross not captured/net; wallet-only orders (`razorpay_order_id==''`) bypass the hold → add a
   reconciliation check. (Items 1/5 shipped in #499; webhook-replay dedup already on main.)
3. **#398** — escrow reconciliation / stuck-funds sweeps (failed reversals, address-less paid days strand
   funds forever, transfer drift). Related: **#542** (mirror the cancelled-stranded reconcile for orders +
   meal-plan-days — same class as #534, needed before flags-ON).
4. **#392 remainder**, **#527** (per-line then full-cancel strands remaining live items' money),
   **#524/#396** (ToPaise truncates a paise — migrate to integer paise), **#540**.
5. **#400** remaining — per-day meal-plan issue REPORTING surface + vendor-fault accounting (full-clawback
   vs goodwill at resolve). The dispute-freezes-hold core is DONE (#457/#458/#498/#534).

## Payment integrity before real money
- **#395 (P1)** — DB unique indexes on `orders.razorpay_order_id/payment_id/stripe_payment_intent_id`
  (+`meal_plans.razorpay_order_id`); status-guard the verify path (dup chef notifications on verify/webhook
  race); webhook-only completion skips wallet settlement. Cheap + high-value.
- **#397 (P1)** — money-movement audit trail (refunds/holds/releases/reversals).

## Tiffin / customer GA (gates tiffin, not the payout engine)
- **#402 (P1)** — mobile meal-plan booking shows food-only amount but charges full advance (GST+delivery).
  Customer-trust bug. **#399/#410/#411/#413** — whole-plan cancellation, per-vendor cutoff/skip days,
  mid-week cancellation economics (≥20% penalty), monthly plans + web parity.

## Deferrable (not launch-blocking)
#393 (RTO/returned-delivery money), #401 (unimplemented refund surfaces), #389 (settlement disbursement /
weekly mark-paid ledger — manual-first is fine), #30 (RazorpayX auto-payout).

## Go-live sequence (after the gate clears)
#218 Razorpay sandbox sign-off → #25 live switch + register live webhook → #200/#8 E2E lifecycle.
#389/#30 disbursement automation later.

## Workflow rules (UNCHANGED — they keep catching real double-refund / stranded-money bugs)
- Money-path = **plan → plan-check → RED-first TDD → independent verify**. Do NOT skip plan-check or
  verify. RED-first with callback-injection for check-then-act races (the chef before-update callback seam
  in `handlers/chef_delivered_gate_test.go` is the reusable pattern).
- **Work in an ISOLATED git worktree**, always: `git worktree add -b <branch> <scratchpad>/wt-x origin/main`,
  do all edits/build/test/push with `git -C <worktree>`. A concurrent agent session shares this working
  tree — a plain `git checkout` collides and your commit lands on the wrong branch (bit us twice; see the
  `reference_shared_worktree_head_collision` memory). Clean up the worktree after merge.
- **`gh auth switch --hostname github.com --user mahesh-sangawar` before EVERY gh write** (it silently
  reverts to the Civica EMU account).
- Feature-branch + PR to main. All 6 payout PRs are merged, so new branches off `origin/main` no longer
  conflict with them. Sandbox tests use raw `CREATE TABLE` on sqlite `:memory:` (gorm AutoMigrate emits
  invalid SQLite DDL for uuid-defaulted models); `First`/`Find` use `SELECT *` so a column subset is fine.
  Existing money harnesses: `setupCrossguardDB` (group_orders + meal_plan_days + outbox), `setupHoldDB`,
  `setupChefOrderDB` + `postStatus`, `flagsOff`/`withEscrowFlag`.
- Commit the fix INTO the commit before pushing (an uncommitted working-tree fix cost a CI round on #532).
  Wait for CI green (Build + Run Tests + GitGuardian) before merging.

## Also pending (non-payments)
PRs #528 (@Sam123ben vendor feed) and #357 (dependabot) are open — not ours. #430 (auth logout), #334
(integration review + Playwright E2E). Mobile: promote the chef-page redesign OTA from preview to
production once Menu/Reviews tabs are QA'd on the TestFlight build.
