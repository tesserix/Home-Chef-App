# Handoff — Vendor-Payout Prod Readiness (2026-07-05, session 5)

Continues `HANDOFF-2026-07-05-prod-readiness-session4.md`. Goal unchanged: make the escrow
control plane ready to flip `ORDER_PAYOUT_AUTO_RELEASE_ENABLED` / `MEAL_PLAN_ESCROW_ENABLED`
ON. Both still OFF (no live money). Epic #403.

## Shipped this session (all MERGED to origin/main; HEAD = `ad8d1963`)

1. **#554 wallet top-up idempotency** (PR #556) — merged at session start (was in-flight from s4).
2. **#398 (P1) — failed day-refund reverse leaves re-drivable drift, not a settled strand**
   (PR #557). `RefundDay` swallowed a failed `ReverseTransfer` then `markRefundedDayHold`
   stamped the hold terminal (released→reversed+`settled_at`, or parked→withheld) regardless
   of gateway success — a failed claw-back stranded the still-live transfer invisibly (chef
   keeps refunded money). Threaded `reverseOK` into the renamed `reverseRefundedDayHold`: a
   failed reverse now leaves `reversed`+`settled_at=NULL` drift that `reconcileMealPlanDays`
   re-drives (attempt-capped). Already-reversed counts as success.
3. **#397 (P1) — money-movement audit trail for escrow transfer seams** (PR #559). New
   `auditTransferMovement` + `payout.transfer.{hold,release,reverse}` system-actor audit rows,
   wired into all order/day/group hold/release/reverse seams AND the settle-machinery day
   reverse in `reverseMoney` (the verifier caught that gap — the admin-ReverseHold + reconcile
   re-drive path was unaudited). **Landed the injectable `RazorpayClient.baseURL` (handoff
   option b)** so the seams are end-to-end testable via `httptest.Server` — reuse for #395·4,
   #218. Fixed the `user_agent`-column gap in setupCrossguardDB/setupReleaseDB/cancellation harnesses.
4. **#527 (P2) — full cancel refunds remaining live-item money after a per-line cancel**
   (PR #561). `Total − RefundAmount` double-subtracts per-line refunds (Total was already
   reduced). New `services.RemainingRefundable` = `Total − RefundAmount + Σ(cancelled item
   refund_amount)` — a strict no-op without per-line cancels. **Verifier caught a HIGH
   over-refund RACE** (stale caller `order.Total`/`RefundAmount` + fresh per-line query →
   over-credit); fixed by making `RemainingRefundable` read all three in ONE consistent DB
   snapshot (ignores the caller's struct) + refresh money fields in `RefundOrderForCancellation`.
   Wired into `CancelOrder` + `RefundOrderForCancellation` only.
5. **#555 (P2) — dedup order.paid on Stripe verify + full-wallet settle** (PR #562). Extracted
   provider-generic `completeOrderPaymentTx` (guarded `WHERE payment_status <> completed`,
   emit only on the single transition); `completeRazorpayOrderTx` is now a thin wrapper.
   `verifyStripePayment` + `settleFullWalletOrder` no longer double-emit order.paid on a
   re-verify/race.
6. **#563 (P2) — guard the Stripe webhook handlers + block refunded→completed re-stamp**
   (PR #564). Both Stripe webhook handlers were UNCONDITIONAL updates (replay could re-stamp a
   refunded/completed order). New `completionBlockedStatuses = {completed, refunded}`; every
   completion path (`completeOrderPaymentTx`, `handlePaymentCaptured`, `handleStripePayment-
   Succeeded`) now guards `NOT IN (completed, refunded)` — allows `failed→completed` (retry
   after decline) but blocks the refunded re-stamp that would re-enable a payout on refunded
   money at flags-ON. `handleStripePaymentFailed` guarded too.
7. **#560 (P2) — InitiateRefund + order-issue refund use RemainingRefundable** (PR #565). The
   #527 `Total − RefundAmount` strand on two more entry points. `InitiateRefund` uses
   `RemainingRefundable`; `RefundIssueToWallet` keeps its FOR UPDATE lock + adds the per-line
   sum via a new tx-aware `PerLineRefundedTotalTx` (a cross-connection read is a separate DB
   under sqlite `:memory:` / separate snapshot under a Postgres tx — the gotcha this fixed).

## ABANDONED this session (unsafe — do NOT retry naively)

- **#395 item 6 (release capacity on payment.failed)** — implemented, then the adversarial
  verifier caught a **CRITICAL oversell** and I did NOT ship it (branch deleted, never pushed).
  A card decline is not abandonment: `CreateOrderPayment` (payment.go:58) lets a `failed`
  order be re-paid and the retry does NOT re-reserve, so eager release lets another customer
  grab the freed slot while the original still completes on retry → oversell. The stale-order
  cron's TTL release is the SAFE mechanism (current behavior). See the analysis + re-scope
  options on **issue #395** (comment). Correct fix = re-reserve on retry in `CreateOrderPayment`
  (option a) — non-trivial; or shorten the abandonment TTL (option b).

## New follow-ups filed this session
- **#558** — `settleWalletTopUps` dedup key `orderID:account` collides if a chef & driver
  share one Razorpay payout account; key on `orderID:leg:account` (P3, user-requested).
- **#560** — `InitiateRefund` (payment.go:799) + `RefundIssueToWallet` (order_issue.go:128)
  still use the naive `Total − RefundAmount` and strand/block after a per-line cancel (no
  over-refund). Route through `RemainingRefundable`; `order_issue` needs a tx-aware variant
  (reads under FOR UPDATE). P2.
- **#563** — `handleStripePaymentSucceeded` does an UNCONDITIONAL update (can re-stamp a
  refunded order → completed on a webhook replay); mirror of the guard #553/#555 added
  elsewhere. Also noted the `stale_order_cron` unconditional-UPDATE-by-id TOCTOU. P2.

## Remaining before-flags-ON gate (recommended order)

1. **#549 (P2) — partial goodwill refund forfeits the chef's ENTIRE hold** (over-withholding).
   The big remaining money-safety item. **POLICY DECIDED 2026-07-05: chef eats the refunded
   amount** (reverse only the refunded portion from the chef's held transfer, keep the
   remainder releasable). Full ready-to-implement spec on issue #549 (comment): the two
   partial-capable callers (`InitiateRefund` both branches, goodwill `RefundOrder`); item 2 =
   don't flip `Order.Status`/`payment_status`→Refunded on a partial; item 1 = new
   `WithholdOrReverseOrderHoldForPartialRefund` that leaves the hold releasable + runs a
   flag-gated `ReverseOrderChefTransferPartial(orderID, refundedPaise)` (find the transfer whose
   `Account == order.Chef.RazorpayAccountID`, reverse `min(refundedPaise, transferAmount)`).
   State is DB-testable now; the seam is httptest-testable via the #559 baseURL; **gateway
   behavior (partial-reverse an on-hold transfer then release the remainder) needs #218 sandbox**.
   Deferred to a focused session — large + gateway-entangled; no live loss while flags OFF.
2. **#395 remaining** — item 1 (unique indexes — needs a **tesserix-k8s** db-schema-bootstrap
   migration, outside this repo), item 3 (webhook-only wallet settlement), **item 4 (tip/group
   signature+amount parity — now unblockable via the injectable baseURL from #559; best in-repo
   next pick)**, item 5 (mutex-guard signature reads). Items 2 (#553) and 6 (parts) touched;
   #395·6 ABANDONED (unsafe — see above). NOTE: #563 also filed the pre-existing
   `handleStripePaymentSucceeded` unconditional-update fix — that landed in PR #564.
3. **#547 (P3)** — freeze commission rate on day/group. NEEDS a `commission_rate` column on
   `meal_plan_days`/`group_orders` → a **tesserix-k8s migration** (same blocker class as #395·1).
4. **#544 (P3)** — cancel-path status desync (inert; no RazorpayPaymentID on those rows).
5. **#396 Phase 2** — integer-paise columns + largest-remainder allocation + tighten
   reconciliation to exact match (folds in the tips.go un-rounded-sum edge). Also needs
   migrations for the new columns.

Most remaining items are now either the big deferred **#549**, **migration-blocked** (need a
tesserix-k8s db-schema migration outside this repo: #547, #396·2, #395·1), or a lower-priority
in-repo cleanup. **#395 item 4** is the best remaining in-repo money-safety pick.

## Workflow rules (UNCHANGED — caught a swallowed-strand, an over-refund race, AND a CRITICAL
oversell this session — do NOT skip verify)
- Money-path = **plan → RED-first TDD → GREEN → independent adversarial verify** (a go-developer
  subagent given the diff + specific attack points). Fix confirmed in-scope gaps in-PR; file
  pre-existing ones. **If verify finds a fundamental flaw, ABANDON the PR** (as with #395·6) —
  do not ship broken money/inventory code.
- **Isolated git worktree per branch**: `git worktree add -b <branch> <scratchpad>/wt-x
  origin/main`; all ops via `git -C`. Remove after merge. (`reference_shared_worktree_head_collision`.)
- **`gh auth switch --hostname github.com --user mahesh-sangawar` before EVERY gh write.**
- sqlite harnesses use raw `CREATE TABLE` (AutoMigrate emits invalid SQLite DDL). Patch EVERY
  harness that exercises a query when it gains columns; run `go test ./...` before pushing.
- The Postgres capacity SQL (`GREATEST`/`now()`/`ON CONFLICT`) is NOT sqlite-runnable — the
  codebase never unit-tests it; test the orchestration/guard around it with no-item orders.
- Injectable `RazorpayClient.baseURL` (added #559) → point a test client at an `httptest.Server`
  to drive the transfer seams end-to-end without a live gateway (reuse for #395·4, #218).

## Go-live sequence (after the gate clears)
#218 Razorpay sandbox sign-off → #25 live switch + register live webhook → #200/#8 E2E lifecycle.
#389/#30 disbursement automation later; manual weekly payouts fine interim.

## Also pending (non-payments)
PRs #528 (Sam's vendor feed) / #357 (dependabot) — not ours. #430 (auth logout), #334
(integration review + Playwright E2E). Mobile: promote the chef-page redesign OTA to production
once Menu/Reviews tabs are QA'd on TestFlight.
