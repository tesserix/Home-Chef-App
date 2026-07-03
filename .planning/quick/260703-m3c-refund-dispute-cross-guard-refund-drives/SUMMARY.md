# Phase quick Plan m3c-refund-dispute-cross-guard Summary

Cross-guard order refunds against the payout hold state machine (GH #457, P0): a shared
helper drives the hold on every order refund, wired into all five refund choke points, plus
a refunded_at-aware all-aggregate `ReleaseHold` backstop and an open-issue-flagged admin queue.

## What changed and why

### Production code

- **`apps/api/services/payout_release.go`**
  - `WithholdOrReverseOrderHoldForRefund(db, orderID, reason)` — the shared cross-guard.
    Loads only `payout_hold_status`; `release_eligible|awaiting_customer_confirmation` →
    `withheld` (conditional UPDATE), `released` → `reversed` + `settleReverse` (claw-back
    seam, stamp-safe), `none|withheld|reversed|disputed` → no-op. Idempotent/race-safe via
    `transitionHold`'s conditional `WHERE payout_hold_status IN (from)`; best-effort
    `LogSystemAudit(nil, "payout.hold.refund_crossguard", ...)` on a real transition.
  - `orderRefundBlocks` + `releaseBlockedForAgg` — release-side backstop. `ReleaseHold` on
    ANY aggregate (order / meal-plan-day / group-order) now pre-checks the underlying order:
    blocked on `status in (refunded,cancelled)` OR `refunded_at IS NOT NULL` OR pending
    `OrderIssue` → `ErrHoldNotEligible`, before any state moves. A nullable/missing linked
    order → not blocked (backstop fail-open; the five wiring sites already drove the hold —
    also avoids breaking a legitimate release when the linked order row is soft-deleted).
  - `listPendingOrders` — excludes `status in (refunded,cancelled)` AND `refunded_at IS NULL`;
    adds `EXISTS(... order_issues ... status='pending') AS has_open_issue`.
  - `PendingPayout.HasOpenIssue` (+ `pendingRow.HasOpenIssue`, populated in `toPending`).

- **`apps/api/services/order_issue.go`** — `RefundIssueToWallet` now captures the tx result;
  on a real commit (not `ErrNothingToRefund`) it calls the helper best-effort (logged, still
  returns the refund result). One choke point covers auto (`by="system"`) + admin paths.

- **`apps/api/services/temporal_order.go`** — `CompensateOrderRefund` calls the helper
  best-effort after the final Updates. W3 comment: the double-reverse is safe because the
  second gateway reverse is **gateway-rejected + logged non-fatal** (`order_payout.go:108`),
  NOT because the stamp skips it.

- **`apps/api/handlers/chef_order_cancel.go`** — helper wired into `CancelOrder` (in-flight
  full refund) and `RefundOrder` (post-delivery goodwill), both best-effort. Added the
  `TODO(#457-followup)` block (CancelOrderItem safe pre-delivery, handleRefundProcessed
  webhook, RefundGroupParticipant->hold-reverse, day/group queue-exclusion).

- **`apps/api/handlers/payment.go`** — `PaymentHandler.InitiateRefund` wired at BOTH commit
  sites: to-wallet branch (~784) and gateway-persist branch (~938), best-effort with
  `log.Printf` + `services.CaptureBackgroundError`, HTTP response unchanged.

### Tests / harness

- **`apps/api/services/payout_crossguard_refund_test.go`** (new) — 14 tests on an extended
  `setupCrossguardDB` (adds refund columns on orders/order_issues + `wallets`/`wallet_txns`)
  so `Crossguard_AutoRefundDrivesHold` actually executes the `RefundIssueToWallet`->
  `CreditWallet` path (not a vacuous `-run` match — verified it runs and passes).
- **`apps/api/handlers/admin_payout_test.go`** — added an `order_issues` table to
  `setupPayoutHandlerDB` so the new `EXISTS` subquery resolves (production code unchanged).

## Five wiring sites confirmed

1. `RefundIssueToWallet` (order_issue.go, `issue.OrderID`)
2. `CompensateOrderRefund` (temporal_order.go)
3. chef `RefundOrder` (chef_order_cancel.go)
4. chef `CancelOrder` refund path (chef_order_cancel.go)
5. `PaymentHandler.InitiateRefund` (payment.go) — **both** commit sites (to-wallet + gateway-persist)

## Verification

From `apps/api`:
- `go build ./...` — clean
- `go vet ./...` — clean
- `gofmt -l` on all seven touched files — empty (repo has pre-existing gofmt drift in
  unrelated files e.g. `models/menu.go`, `services/campaign.go` — out of scope, not touched)
- `go test -race ./services/... ./handlers/...` — `ok` both packages
- 14/14 `Crossguard_*` tests pass; `AutoRefundDrivesHold` confirmed to actually run
- Prior payout tests (#387/#388/#456/#459) + `RefundIssueToWallet` still green

## Deviations

- **[Rule 1 - correctness] Release guard tolerant of a missing/soft-deleted linked order.**
  `orderRefundBlocks` treats `gorm.ErrRecordNotFound` as not-blocked instead of hard-failing.
  Surfaced by the existing `TestReleaseHold_GroupOrder` (seeds a group order linked to a
  consolidated order that isn't inserted); also correct for production where a linked order
  could be soft-deleted — a backstop must not block a legitimate payout on an unloadable row.
- **[Rule 3 - harness] Added `order_issues` table to the admin payout handler harness** so the
  new pending-queue `EXISTS` subquery resolves. Production code unchanged.

## Commits

- `e9762b8a` test: cross-guard suite + extended harness (RED)
- `871de496` feat: `WithholdOrReverseOrderHoldForRefund` helper (GREEN core)
- `0e5bf5e7` feat: drive the hold from all five order refund paths (GREEN wiring)
- `ed3d8c42` feat: refunded_at-aware all-aggregate ReleaseHold guard + queue exclusion + HasOpenIssue
- `b837d58c` fix: tolerate missing linked order + harness order_issues table + #457 follow-up TODO

## Follow-ups (noted, not wired — `TODO(#457-followup)`)

- `CancelOrderItem` per-line refund (safe: pre-delivery, hold=none)
- `handleRefundProcessed` webhook out-of-band refund (status stays delivered)
- `RefundGroupParticipant` -> hold-reverse parity for the group aggregate
- day/group admin queue-exclusion (ReleaseHold pre-check already backstops all three)
- partial-amount reversal (#462, full-reverse only here); #458/#460/#461/#462
