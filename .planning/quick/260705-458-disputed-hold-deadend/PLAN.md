# PLAN — #458 Disputed payout hold is a dead-end

**Issue:** Once a hold reaches `disputed` (via `applyHoldConfirm`), nothing transitions it out. `ListPendingPayouts` never lists disputed rows; `Release/Withhold/ReverseHold` source-states exclude `disputed`; `WithholdOrReverseOrderHoldForRefund` treats `disputed` as a no-op default; `AdminResolveIssue`/`AdminRejectIssue` never write `payout_hold_status`. So a disputed order's held transfer is stuck forever. Epic #403, flag-gated (no live money).

## Root cause (confirmed in code)
- `applyHoldConfirm` (services/payout_hold.go) produces `disputed` for order / meal_plan_day / group_order.
- Resolve path: `AdminResolveIssue` → `RefundIssueToWallet` (services/order_issue.go:188) → `WithholdOrReverseOrderHoldForRefund` — but that switch's `default` case (payout_release.go:436-437) lists `disputed` as a no-op. So an approved refund on a disputed order refunds the wallet but leaves the hold `disputed`.
- Reject path: `AdminRejectIssue` (handlers/order_issue.go:320) only sets the issue → `rejected`; never touches the hold. A rejected dispute leaves the chef's legitimate payout stuck in `disputed` forever.

## Fix (ORDER aggregate — the one whose release/reverse money seam is wired)

### 1. Resolve (refund) → `disputed → withheld`
Add `models.PayoutHoldDisputed` to the withheld branch of `WithholdOrReverseOrderHoldForRefund` (both the `switch` case and the `transitionHold` from-list). A refund on a disputed order now blocks the payout, exactly like an eligible/awaiting order. Automatically covers `AdminResolveIssue` and every other order-refund path via the single crossguard choke point. Consistent with the #457 full-clawback model (partial-amount handling stays #462).

### 2. Reject (no refund) → `disputed → release_eligible`
New service fn `ReleaseDisputedOrderHoldIfCleared(db, orderID)` — one atomic guarded UPDATE, then emit on a genuine change:
```
UPDATE orders SET payout_hold_status='release_eligible'
WHERE id=? AND payout_hold_status='disputed'
  AND status NOT IN ('refunded','cancelled') AND refunded_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM order_issues WHERE order_id=? AND status='pending')
```
Emits `payout.hold_release_eligible` (outbox, same tx) when RowsAffected>0. The guards mean it only releases a genuinely disputed, non-refunded order with **no remaining pending issues** (an order can carry several issues). Wire a best-effort call into `AdminRejectIssue` **after** the issue is marked rejected (so the just-rejected issue isn't counted pending); a hold-drive failure logs and never fails the reject response. Uses the same `NOT EXISTS`/atomic pattern as the #460 fix.

### 3. Surface disputed holds in the queue (visibility)
Add `IncludeDisputed bool` to `PendingFilter`; `pendingStatuses()` appends `disputed` when set. Wire `?include=` in `GetPendingPayouts` to accept `awaiting` and/or `disputed` (comma-tolerant). Disputed rows show with `HasOpenIssue=true` and remain **non-releasable** — `releaseBlockedForAgg` (pending issue) and the `release_eligible`-only source guard both block a release attempt, so surfacing them is safe/read-only.

## Scope boundary (documented, filed as follow-up)
The automatic resolve/reject **drive** covers the **order** aggregate because its release/reverse money seam is wired. Meal-plan-day / group-order disputed holds get **queue visibility** now; driving them on resolve/reject needs the day/group refund seam (a separate #457 follow-up) — moving their hold state without the matching money seam would create drift. Filed as a follow-up.

## Tests (TDD)
- `TestCrossguard_DisputedWithheld` (NEW) — refund on a disputed order → withheld; release then blocked.
- `TestCrossguard_NoopStates` (UPDATE) — remove `disputed` from the no-op set (now driven); keep none/withheld/reversed.
- `TestReleaseDisputedOrderHoldIfCleared_*` (NEW) — disputed + no pending issue → release_eligible + 1 event; disputed + still-pending issue → stays disputed, 0 event; disputed + refunded_at → stays disputed; non-disputed source → no-op.
- `AdminRejectIssue` wiring (NEW, handler harness) — disputed order + one pending issue → reject → hold = release_eligible.
- Queue `IncludeDisputed` (NEW) — a disputed row appears only when the flag is set.
- All existing `payout_crossguard_refund_test.go` / `payout_hold_test.go` / `payout_release_test.go` stay green.

## Verification
- `go build ./...`, `go vet ./services/ ./handlers/`, `go test ./services/ ./handlers/ -count=1`, then `go test ./...`.
- Money-path gates: plan-check (this doc) + independent verify on the diff.
