# Handoff — Vendor-Payout Prod Readiness (2026-07-05, session 2)

Continuation of the vendor-payout escrow epic (**#403**). Everything is behind escrow flags **`ORDER_PAYOUT_AUTO_RELEASE_ENABLED` / `MEAL_PLAN_ESCROW_ENABLED` (both OFF)** — no live money until they flip. Goal of the next phase: **make the payout control plane prod-ready so the flags can flip ON**, then promote the mobile chef-page redesign to production.

Read first: memory `project_vendor_payout_model_finalized` (full decision record), and this file.

## Shipped + MERGED this session (all on `main`, all flag-gated)
Each went **plan → plan-check → independent verify**, RED-first TDD:
- **#460** (PR #495) — payout-hold races: chef `UpdateOrderStatus` (+delivery.go:503/1430, orders.go:832) now targeted `Updates(map)` not full-row `Save(&order)` (no lost-update of hold cols); `applyHoldConfirm` folds the dispute predicate into the guarded UPDATE (disputed-EXISTS-first, then release-NOT-EXISTS-from-awaiting, + `customer_confirmed_at IS NULL`). Reusable seam: `openOrderIssueSubquery` (NewDB session) in `services/payout_hold.go`.
- **#458** (PR #500) — disputed-hold dead-end: resolve/refund drives `disputed→withheld` (added `disputed` to `WithholdOrReverseOrderHoldForRefund`); `AdminRejectIssue` rewritten to run reject + `ReleaseDisputedOrderHoldIfCleared(tx,orderID)` (disputed→release_eligible, guarded on non-refunded + `NOT EXISTS(pending issue)`) in ONE tx; queue `IncludeDisputed` (visibility-only). ORDER aggregate only.
- **#462** (PR #499, PARTIAL) — exact GST split (`cgst=Round2(fullGST/2); sgst=Round2(fullGST-cgst)`); BulkReleasePayouts distinguishes `errors.Is(ErrHoldNotEligible)`→skipped vs real errors→`failed` + 500-item cap + audit records IDs. **Item 4 (ReleaseDayPayout idempotency) already done by #459.** Remaining items left OPEN on #462 (see gate below).
- **#496** (PR #516) — `ReleaseHold` check-then-act race: `releaseTransition` folds the refund/dispute block into the guarded release UPDATE (`refundedOrderSubquery` via NewDB `Model(&Order{})` for soft-delete parity + `openOrderIssueSubquery`), keyed on the resolved linked order id (order→self, day/group→order_id). `transitionHold` untouched (Withhold/Reverse are refund-driven, must NOT be block-guarded).
- **#501/#502** — CI "Run Tests" flake fixed: `chef_orders_visibility_test.go` seeded bare `time.Now()` but the dashboard filters on `CapacityDay` (IST midnight); sqlite compares naive datetimes so evening-UTC CI runs dropped "today" orders. Fix: anchor the seed to `services.CapacityDay(time.Now()).Add(12h)`.

**Issue hygiene:** #456/#457/#459 were fixed earlier but left open — now **CLOSED** (with PR refs). #461 step 2 was lost when #461 closed — **re-filed as #515**.

## Mobile chef-page redesign (PR #503, MERGED)
Decluttered the customer chef-detail screen into **3 in-page tabs** (`Menu` default / `Weekly plan` / `Reviews`, Airbnb coral underline, deep-linkable via `?tab=`); big coral cards → slim rows; cuisines truncated; **compact one-day weekly menu** (day-selector + Veg/Non-veg filter chips, defaults to IST today) replacing the 7-day scroll. New components: `ChefDetailTabs`, `ChefMenuTab`, `ChefWeeklyPlanTab`, `ChefReviewList`, `ChefActionRow`, `WeeklyMenuDishCard`. Sim-verified (Weekly-plan tab via fixture; deep-links to real chef are clobbered by the root auth-redirect).
- **OTA on `preview` channel** (update group `74e8ec3b`). **iOS `.ipa` + Android `.aab` production builds FINISHED** → auto-submitting to TestFlight/Play.
- **TODO:** QA the default **Menu** + **Reviews** tabs on a real device (TestFlight), then `cd apps/mobile-customer && eas update --channel production --environment production --message "..."` to promote to all prod users. (Did NOT push to production OTA — couldn't verify the default tab on-device.)

## PROD-READINESS GATE — must clear before escrow flags flip ON
1. **#515** (was #461 step 2, P1 security) — remove the `computeLegacy` fallback in `apps/api/middleware/bff_auth.go` + add `SPManagePayouts` on the payout routes. **DEPLOY-GATED:** confirm **tesserix-home #26** (identity-bound BFF signer) is **LIVE in prod** first (else every admin request 401s). #26 merged 2026-07-04; prod rollout UNCONFIRMED. Verify auth-bff is bound-only (it already signs 8-field).
2. **#498** (P2) — drive **day/group** disputed holds on resolve/reject (extend #458 beyond the order aggregate). Needs the day/group refund seam (`RefundDay`/`RefundGroupParticipant`→hold) done together to avoid state-without-seam drift.
3. **#508** (P2, from pen-test #468) — re-guard refund↔payout money seams before enabling flags.
4. **#462 leftovers** — partial-amount reverse (`ReverseTransfer(id, amountPaise)`), queue `Amount` shows gross not captured/net (compute), webhook replay (HMAC-over-body only, no ts/nonce), wallet-only orders bypass hold (add reconciliation check).
5. **W-A** — `CancelGroupOrder` reverse runs OUTSIDE the guarded tx → route through `ReverseHold` (flag-gated TODO in code, safe at launch but fix before ON).
6. **Verify meal-plan-day-gross** — the #390 NET fix was orders; the meal-plan-day per-day transfer (`ReleaseDayPayout`/`RefundDay` seam) may still pay GROSS. Confirm and fix.
7. **TCS §52 (1%)** — NOT modeled in earnings; needs **CA sign-off** before live (with #19).

## Broader payment backlog still open (triage for launch-blocking)
**P0:** #392 (cancellation paths return no money). **P1:** #400 (disputes freeze payouts — order-issue↔escrow), #397 (money-movement audit trail), #398 (escrow reconciliation + stuck-funds sweeps), #394 (meal-plan-day double-refund via disjoint idempotency keyspaces), #395 (payment integrity: DB unique constraints, verify/webhook race, signature parity), #393 (RTO/returned-delivery money handling), #402 (mobile meal-plan charges full advance but shows food-only), #389 (vendor settlement disbursement — audit whether the statement mark-paid ledger backend is complete). **P2:** #396 (integer paise), #401 (unimplemented refund surfaces).

## Go-live sequence (from the original plan)
Clear the gate above → **#218** sandbox sign-off → **#25** live flag switch. **#200/#8** E2E validate the full lifecycle. **#30** RazorpayX automates weekly disbursement later.

## Workflow / gotchas (do not relearn)
- **Money-path = plan → plan-check → independent verify.** Do NOT skip plan-check — it caught real money bugs in every issue this session (soft-delete subquery parity, GST float-equality test, disputed→withheld, atomic dispute predicate). Use the callback-injection RED-test pattern for check-then-act races (see #460/#496 tests in `payout_crossguard_refund_test.go` / `payout_hold_test.go`).
- **`gh auth switch --hostname github.com --user mahesh-sangawar` before EVERY gh write** (it flips to the Civica EMU user).
- **Stacked-PR merge gotcha:** merging a PR with `--delete-branch` **auto-closes its child PRs** (can't reopen — base gone). Merge bottom-up WITHOUT `--delete-branch`, retarget each child to `main` first, delete branches last. (Cost me #497 → recreated as #500 this session.) See memory `reference_stacked_pr_merge_gotcha`.
- **Deploy pipeline:** merge to `main` → CI → GHCR → Kargo/ArgoCD. The `HomeChef API - CI Build` badge shows red from the **Trivy vuln-scan gate** (33 pre-existing dependabot advisories) — image still builds+pushes; the deploy-advance workflow is separate and succeeds. Not a blocker; worth a dependency-bump pass eventually.
- **Mobile sim QA:** the installed Debug app hot-loads JS from **Metro on port 8082** (8081 = Docker); a persisted login session exists. Deep-links to nested routes get clobbered by the root `_layout` auth-redirect to `/(tabs)`, so to see a screen render a **temp fixture route** (`app/dev-*.tsx`) with mock data + `simctl openurl` on the RUNNING app, screenshot, delete it. Sim UDID `9F9E4972-E434-47CD-B7DA-8BE294D41B97`. See memory `reference_customer_sim_live_qa`.
- **EAS builds** are manual-only: `gh workflow run "HomeChef Mobile - EAS Build" --ref main -f app=customer -f platform=all -f profile=production -f submit=true`. `eas update` needs `--environment <channel>` in `--non-interactive`.

## Key payout files
`services/earnings.go` (ComputeOrderEarnings, GST split), `services/commission.go`, `services/payout_hold.go` (applyHoldConfirm, ReleaseDisputedOrderHoldIfCleared, openOrderIssueSubquery, Confirm*Hold), `services/payout_release.go` (ReleaseHold, releaseTransition, transitionHold, refundedOrderSubquery, WithholdOrReverseOrderHoldForRefund, ListPendingPayouts, IncludeDisputed), `services/payout_auto_confirm_cron.go`, `services/payout_reconcile_cron.go`, `services/group_order_payout.go`, `services/meal_plan_escrow.go` (ReleaseDayPayout — check gross), `handlers/payment.go` (chefNetPayout), `handlers/admin_payout.go`, `handlers/order_issue.go`, `handlers/chefs.go`, `middleware/bff_auth.go` (#515). Admin UI in **tesserix-home** (`app/admin/apps/homechef/*`).
