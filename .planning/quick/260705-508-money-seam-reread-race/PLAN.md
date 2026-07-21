# #508 — Re-guard the money seam against current hold status (re-read before dispatch)

**Epic #403. From pen-test #468 finding #10. Latent — safe today ONLY because both escrow flags are OFF
(the money seams are no-ops). Must-fix before #218/#25 flip the flags ON.**

## The race
`payout_release.go`: `ReleaseHold` → `releaseTransition` (commits `released`) → `settleRelease` →
`releaseMoney` + `stampPayoutSettled`. The money seam runs POST-COMMIT and dispatches on the flip
*this goroutine won*, WITHOUT re-reading the row's current status. A concurrent refund
(`WithholdOrReverseOrderHoldForRefund` → `transitionHold` released→`reversed` → `settleReverse` →
`reverseMoney`) can interleave:
- releaseMoney runs AFTER reverseMoney ⇒ **chef paid after claw-back (money leaked)**.
- or both seams run for the same transfer in the wrong order ⇒ paid / clawed / paid-after-claw.

## Fix (TDD-first)
Dispatch the money seam by the **freshly-read terminal status**, not the won transition. Replace
`settleRelease`/`settleReverse` bodies with one status-dispatched settle:

```
func settlePayout(db, aggType, id):
    status := re-read payout_hold_status   // SELECT ... FOR UPDATE (short tx, read only)
    switch status:
        released  -> releaseMoney; stampPayoutSettled   // only if STILL released
        reversed  -> reverseMoney; stampPayoutSettled    // a refund flipped it -> reverse instead
        default   -> no-op                                // not terminal / already handled
```
- `settleRelease(db,…)` and `settleReverse(db,…)` both delegate to `settlePayout` (keep the names as
  thin wrappers so `ReleaseHold`, `ReverseHold`, `WithholdOrReverseOrderHoldForRefund`, and the
  reconcile cron call sites are untouched). The key change: **releaseMoney only runs if the row is
  STILL `released` at seam time**; if a refund already flipped it to `reversed`, `settlePayout`
  runs `reverseMoney` instead — no chef payment survives a claw-back.
- `stampPayoutSettled` stays the idempotency key (`WHERE payout_settled_at IS NULL`), so the seam is
  stamped once; the reconcile cron (re-drives `released`/`reversed` + `settled_at IS NULL`) is the
  backstop for the residual window between the re-read and the seam.
- Make the reverse seams tolerate an already-reversed gateway response (mirror `isAlreadyReleasedErr`
  in `ReleaseDayPayout`) so a double reverse in the residual window is a safe no-op, not an error.

## Open design question for plan-check
Holding a `FOR UPDATE` lock across the external Razorpay HTTP call is undesirable (long lock on
db-f1-micro). The proposal releases the lock after the status READ, so a refund committing between
the read and the seam is still possible — bounded and covered by the reconcile backstop + idempotent
seams. Confirm this is acceptable vs. a heavier design (e.g. an outbox-driven single settler, or a
`settle_intent` column). Plan-check MUST validate conservation across the interleavings below.

## RED-first concurrency tests (callback-injection, like #460/#496)
Interleave release vs refund on the same aggregate and assert:
1. refund-commits-then-release-settles ⇒ terminal `reversed`, reverseMoney ran, releaseMoney did NOT.
2. release-settles-then-refund ⇒ terminal `reversed`, net money = returned (release then reverse).
3. exactly-once settle stamp; no paid-after-claw.
4. reconcile cron re-drives a row stranded by a seam error and converges to the terminal status.

## Files
- `services/payout_release.go` — `settlePayout` + re-read; `settleRelease`/`settleReverse` wrappers.
- `services/meal_plan_escrow.go` / order reverse seam — tolerate already-reversed.
- `services/payout_reconcile_cron.go` — confirm it dispatches by current status too (it re-drives per
  status already; verify it reads fresh status).
- `services/*_test.go` — concurrency interleave tests.

## Status: PLANNED — needs plan-check before execution (money-path race). Not started.
