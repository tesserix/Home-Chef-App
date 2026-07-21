# Handoff — Vendor-Payout Prod Readiness (2026-07-06, session 6)

Continues `HANDOFF-2026-07-05-prod-readiness-session5.md`. Goal unchanged: make the
escrow control plane ready to flip `ORDER_PAYOUT_AUTO_RELEASE_ENABLED` /
`MEAL_PLAN_ESCROW_ENABLED` ON. Both still OFF (no live money). Epic #403.

## Shipped this session (3 PRs, all MERGED to origin/main; HEAD = `e101852f`)

Each: plan → RED-first TDD → GREEN → independent adversarial verify (go-developer).

1. **#549 (P2, the big deferred item) — partial goodwill refund forfeited the chef's
   ENTIRE hold** (PR #566, `5b101084`). ROOT MECHANISM (bigger than the status flip):
   every release-side guard blocks the whole payout on `refunded_at IS NOT NULL`
   (payout_release.go canReleaseOrderHold + guarded UPDATE, payout_reconcile_cron.go,
   payout_hold.go). New `services.ReverseOrderChefTransferPartial` /
   `WithholdOrReverseOrderHoldForPartialRefund` (order_payout_partial.go): claws back
   `min(refundedPaise, chef un-reversed transfer)` from the CHEF transfer only
   (Account==Chef.RazorpayAccountID via chef_profiles join; rider untouched; capped by
   NEW `TransferResponse.AmountReversed` so repeat partials don't over-claw), LEAVES the
   hold releasable. All 3 partial-capable sites (InitiateRefund to-wallet + gateway,
   goodwill RefundOrder) only stamp `refunded_at`/flip status→Refunded when
   `fullRefund := refundAmount >= RemainingRefundable`. Flag-gated (state-only no-op
   while OFF). Persists switched `Model(&order)` → `Model(&models.Order{}).Where(id)`
   (avoid spurious 65-col Chef upsert). **Verifier caught + fixed IN-PR:** goodwill used
   naive `Total−RefundAmount` → misclassify partial-after-per-line-cancel as full;
   wallet idempotency key was per-ORDER → repeat partials silently shorted the customer
   while clawing the chef → new per-instance `refundWalletIdempotencyKey`;
   `releaseCapOnRefund` used the same divergent naive formula → now reuses `fullRefund`.
   **Declined** verifier's "revert claim on partial-gateway persist failure" (would risk
   a double gateway refund; leaving terminal is safer — verifier agreed). Follow-ups:
   **#567** (ToWallet branch no atomic claim → concurrent double-submit over-claws chef),
   **#568** (partial-reverse fires even on failed persist + no per-instance idempotency →
   retry double-claws chef; MUST-FIX-BEFORE-FLAGS-ON, validate under #218).

2. **#395 item 4 — tip/group amount+signature parity** (PR #569, `5fed3b0b`). `VerifyTip`
   + `VerifyGroupShare` now check captured AMOUNT (`payment.Amount < ToPaise(tip.Amount)`
   / `ToPaise(me.ShareAmount)` — unforgeable gateway value, hard gate) + Checkout
   SIGNATURE (enforce-when-present, over the STORED rz order id so the client can't
   forge), mirroring main-order VerifyPayment. Was a LIVE under-amount settle-in-full
   hole (not escrow-gated). **Test infra unblocked** (the session-5 blocker): new exported
   `services.SetRazorpayClient` + `services.NewRazorpayTestClient(baseURL,keyID,keySecret,
   webhookSecret)` (razorpay_inject.go) → handler-package tests drive FetchPayment/
   signature via httptest; same convention as services/redis_testseam.go; **reuse for
   #218 sandbox harnessing**. Verifier: SHIP.

3. **#395 item 5 — mutex-guard signature reads** (PR #570, `e101852f`).
   VerifyPaymentSignature + VerifyWebhookSignature now snapshot the client via new
   `snapshotRazorpayClient()` (reads pointer under razorpayMu; client fields immutable
   post-construction → snapshot lock-free) instead of touching the package-global while
   GetRazorpay/Invalidate/SetRazorpayClient write it. RED→GREEN under `go test -race`.

4. **#395 item 3 — webhook completion skips wallet settlement** (PR #571, `fa852e4a`).
   Wallet-at-checkout settlement (#141: debit applied store credit + chef/driver
   top-ups) ran ONLY in the verify path → a webhook-only completion (client dropped
   before verify) left the credit un-debited (customer keeps money they spent) and the
   top-up unpaid. Extracted shared `settleOrderWallet(order)` called from BOTH verify
   and `handlePaymentCaptured` (reloads Chef + Delivery accounts, filter
   `completed AND wallet_applied > 0`, runs on every delivery so a retry recovers a
   crashed partial). Idempotent (DebitWallet key `wallet-debit:<orderID>`; top-ups
   per-(order,account) via #554). Verifier caught + fixed in-PR: return on genuine debit
   failure (insufficient balance) instead of funding the chef/driver off uncollected
   credit.

5. **#568 — partial chef-transfer claw-back not re-fire idempotent** (PR #572, `5cf2858b`).
   `crossGuardRefundHold(..., persistOK)` runs the PARTIAL claw-back only when the refund
   persist committed; FULL stays unconditional (idempotent + safety net). Closes the
   swallowed-persist-failure + retry double-claw.

6. **#567 — InitiateRefund to-wallet branch had no atomic refund claim** (PR #573,
   `388f3f68`). Hoisted the `completed→refunded` claim (`claimRefundForProcessing`) +
   `revertClaim` to before the to-wallet/gateway split so both serialize concurrent
   double-submits (winner proceeds, duplicate 409). Partial reverts to completed
   (repeatable). Verifier caught + fixed: a partial WALLET persist-failure was stuck at
   `refunded` → now `revertClaim()` (safe — wallet credit idempotent). Deliberately NOT
   applied to the gateway branch (reverting there would double-refund via non-idempotent
   `CreateRefund`). Filed **#574** (gateway calls need idempotency keys — recurring gap).

   Testing note: sqlite :memory: cannot faithfully test true concurrency (a racing
   goroutine 404s on DB contention, not 409) — test the transition helper deterministically
   (2 back-to-back claims, 2nd loses), mirroring `payment_complete_race_test.go`.

**#395 in-repo work is COMPLETE** (items 2/3/4/5 done; item 1 migration-blocked) and the
two #549 follow-ups **#567/#568 are DONE**. Only **item 1** of #395 remains
(migration-blocked: DB unique indexes on payment ids, needs a tesserix-k8s migration);
item 6 ABANDONED (oversell). Pre-existing follow-ups noted on #395 (not filed as issues):
CreateTransfer has no gateway idempotency key (timeout-after-success double-pay);
DirectTopUps not auto-reversed on refund; wallet credit not reserved at apply-time.

## ⚠ Infra gap found
`homechef-api-build.yml:155` runs `go test` **WITHOUT `-race`** (only auth-bff uses it) —
data races (like #395·5) slip through CI. Worth adding `-race` to that workflow.

## Remaining before flags-ON gate
The in-repo money-safety work is essentially DONE. What's left is migration-blocked or
lower-priority:
- **Migration-blocked** (need a **tesserix-k8s** db-schema migration OUTSIDE this repo):
  #547 (`commission_rate` col on day/group), #396 Phase 2 (integer-paise columns),
  #395 item 1 (unique indexes on payment ids). These can only be done partially in-repo
  (GORM tags) until the migration lands.
- **Lower-priority in-repo:** #544 (P3, inert). #395·6 ABANDONED (oversell — re-scope =
  re-reserve on retry).
- **Escrow-gated follow-ups: #567 + #568 now DONE (PRs #573/#572).** Still open: #558 (P3),
  #462 leftovers, #540 (P3 meal-plan tax-basis micro-alignment). (#523 was CLOSED — its
  premise didn't hold: meal-plan income already reports via the spawned per-day orders.)
- **#574 filed (before-flags-ON):** Razorpay/Stripe `CreateRefund` + `CreateTransfer` +
  `ReverseTransfer` carry no gateway-side idempotency key → timeout-after-success double-pay
  AND blocks a safe gateway-branch claim-revert. Thread a client `Idempotency-Key`.
- **Other pre-existing gaps (worth issues before live):** DirectTopUps not auto-reversed on
  refund; wallet credit not reserved at apply-time; add `-race` to `homechef-api-build.yml`.

## Next-session recommendation
No unblocked in-repo money-safety items remain. Next steps are the **go-live sequence**
(below) — chiefly **#218 Razorpay sandbox sign-off**, which validates the whole escrow
transfer surface (hold → release → partial/full reverse) that's been built behind the
flags and is the gate before the #25 live flip. The migration-blocked items need a
tesserix-k8s PR first. Consider filing issues for the pre-existing gaps above.

## Money-path rules (UNCHANGED — caught real defects on #549 again this session)
plan → RED-first TDD → GREEN → independent adversarial go-developer verify (given the
diff + attack points). Fix in-scope gaps in-PR (amend + force-with-lease); file
pre-existing. ABANDON if verify finds a fundamental flaw. Isolated git worktree per
branch (`reference_shared_worktree_head_collision`). `gh auth switch --hostname
github.com --user mahesh-sangawar` before EVERY gh write. sqlite harnesses use raw
CREATE TABLE — patch every harness a query touches; run `go test ./...` before pushing.
Reuse the injectable `RazorpayClient.baseURL` (#559) + `services.SetRazorpayClient` +
httptest to drive gateway seams. Wait for CI green (Build + Run Tests + GitGuardian)
before `--merge --delete-branch`.

## Go-live sequence (after the gate clears)
#218 Razorpay sandbox sign-off → #25 live switch + register live webhook → #200/#8 E2E
lifecycle. #389/#30 disbursement automation later; manual weekly payouts fine interim.
