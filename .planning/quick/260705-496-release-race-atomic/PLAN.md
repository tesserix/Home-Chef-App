# PLAN — #496 ReleaseHold check-then-act race (atomic block)

**Issue:** `ReleaseHold` (services/payout_release.go) reads `releaseBlockedForAgg` (→ `orderRefundBlocks`/`HasOpenOrderIssue`) **outside** the tx, then `transitionHold`'s guarded UPDATE only checks `payout_hold_status IN (release_eligible)`. A refund or `OrderIssue` filed in the read→UPDATE window still commits `released` (double-pay). Same class as #460. Epic #403, flag-gated (no live money).

## Fix — fold the block predicate INTO the release UPDATE (keep the pre-check)
Keep the existing `releaseBlockedForAgg` pre-check (fast 409 + good error + existing tests). ADD an atomic backstop: a new `releaseTransition(db, aggType, id)` that replaces the `transitionHold(...→released...)` call in `ReleaseHold` and evaluates the block **as part of** the guarded UPDATE, so nothing filed after the pre-check can slip through.

```
UPDATE <aggtable> SET payout_hold_status='released'
WHERE id=? AND payout_hold_status='release_eligible'
  AND NOT EXISTS (SELECT 1 FROM orders WHERE id=<orderID> AND (status IN ('refunded','cancelled') OR refunded_at IS NOT NULL))
  AND NOT EXISTS (SELECT 1 FROM order_issues WHERE order_id=<orderID> AND status='pending')   -- openOrderIssueSubquery (#460)
```

- `<orderID>` = the order carrying the block: for the **order** aggregate it's the row id itself; for **meal-plan-day / group-order** it's the row's `order_id`, resolved inside the tx (a stable FK). When `order_id IS NULL` (day/group with no linked order) the two order-keyed predicates are skipped → not blocked (matches current `releaseBlockedForAgg` behavior).
- Predicates key on the **concrete** resolved `orderID` (non-correlated subqueries), evaluated at UPDATE execution → the issue/refund check is atomic with the status flip. Reuses `openOrderIssueSubquery(tx, orderID)` from #460 for the pending-issue clause.
- `RowsAffected == 0` ⇒ either already-actioned OR a block landed in the window ⇒ `ReleaseHold` returns `ErrHoldNotEligible` (unchanged contract). On a genuine flip, emit `hold_released` on the outbox in the same tx (as today).

`transitionHold` stays for Withhold/Reverse (those are DRIVEN by refunds, must NOT be block-guarded). Only the release path changes.

## Tests (TDD, RED-first via callback injection — same technique as #460)
- **NEW race (order):** seed a `release_eligible` order, no issue. A one-shot `Before(gorm:update)` callback inserts a pending `OrderIssue` just before the release UPDATE (after the pre-check, which is SELECT-only). `ReleaseHold` must return `ErrHoldNotEligible`, leave the row `release_eligible`, and emit **0** `hold_released`. Fails on old code (transitionHold flips to `released`), passes on the fold.
- **NEW race (refund):** same shape, callback stamps `refunded_at` before the UPDATE → blocked.
- **Regression (stay green):** `TestCrossguard_ReleaseHold_AllowsClean` (clean → released), `_BlocksRefundedAtOnly` / `_BlocksRefundedStatus` / `_BlocksPendingIssue` (pre-check path), `_Day_BlocksRefundedOrder` / `_Group_BlocksRefundedOrder`, and `TestReleaseHold_FlagOff_AdvancesNoMoney`.

## Verification
`go build ./...`, `go vet`, `go test ./services/ ./handlers/ -count=1`, `go test ./...`. Money-path gates: plan-check + independent verify. Harness: `setupCrossguardDB` (orders + order_issues + outbox + day/group) already has everything.

## Out of scope
Withhold/Reverse race (they're refund-driven, blocking them would be wrong); the broader release/reverse drift (#459, done).
