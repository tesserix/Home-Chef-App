# Summary — #456: Route group/office-order chef payouts through the payout hold state machine + flag-gate

**Branch:** `feat/group-order-payout-hold` (not pushed)
**Issue:** GH #456 (P0 — live money leak)

## What changed and why

The group/office order released the chef's Route transfer immediately on delivery with
**no escrow-flag gate**, bypassing the entire #387/#388/#459 payout hold machine. This
slice promotes the group order to a first-class payout-hold aggregate and closes the leak.

### Production code
- `models/group_order.go` — added 5 payout-hold columns to `GroupOrder` (`PayoutHoldStatus`, `CustomerConfirmedAt`, `DeliveredAt`, `PayoutSettledAt`, `PayoutSettleAttempts`), mirroring `Order`/`MealPlanDay`.
- `migrations/20260703000004_add_group_order_payout_hold.{up,down}.sql` — auditable DDL. Number bumped 000003→000004 (000003 was already `add_payout_settled_at`).
- `services/group_order_payout.go` — **P0:** flag-gated all 3 money seams on `payoutMovementEnabled()`. Rewrote `MarkGroupOrderDelivered` to **park** the hold `awaiting_customer_confirmation` + stamp `delivered_at` (new `parkGroupOrderOnDelivery` helper) instead of releasing. Made `ReleaseGroupChefPayout` tolerate a nil gateway (matches `ReleaseOrderPayouts`).
- `services/payout_hold.go` — added `SetGroupOrderHoldAwaitingConfirmation` + `ConfirmGroupOrderHold` (reuse `applyHoldConfirm`).
- `services/payout_release.go` — `aggTypeGroupOrder`, `holdModel` branch, `listPendingGroupOrders` (unioned), `releaseMoney`/`reverseMoney` branches. **W-B:** context = `'GRP-'||substr(id,1,8)`, never raw UUID. **W-C:** updated `PendingPayout.Amount` doc.
- `services/payout_auto_confirm_cron.go` — `sweepGroupOrders`. `services/payout_reconcile_cron.go` — `reconcileGroupOrders`.
- `handlers/admin_payout.go` — `parseAggType` + bulk accept `"group-order"`.
- `handlers/payout_hold.go` — `ConfirmGroupOrderReceived` (host-scoped via `loadGroupForParticipant` + `GroupRoleHost`). `routes/routes.go` — `POST /group-orders/:id/confirm-received`.
- `handlers/group_order.go` — `// TODO(#456-followup)` comment at the flag-gated cancel reverse (W-A minimum fix).

### Tests
- `services/group_order_payout_test.go` (new) — 7 tests incl. park-not-release (flag on+off), confirm→eligible/disputed, ReleaseHold+409, pending-queue, sweep+reconcile, no-phantom-consolidated-hold (Q1 regression).
- `services/payout_hold_test.go`, `payout_release_test.go`, `handlers/admin_payout_test.go` — added `group_orders` table to the hand-DDL harnesses (production code unchanged).

## Verification
- `go build ./...` → OK
- `go vet ./services/... ./handlers/...` → OK (clean)
- `gofmt -l` on all touched files → empty (clean); pre-existing unformatted files elsewhere left untouched.
- `go test -race ./services/... ./handlers/...` → ok / ok
- Existing #387/#388/#459 tests (ConfirmOrderHold, ReleaseHold, WithholdHold, ReverseHold, ListPendingPayouts, PayoutAutoConfirm*, PayoutReconcile*, StampPayoutSettled, admin *Payout) → all pass.

## Deviations
- [Rule 3] Migration number 000003→000004 (collision with existing add_payout_settled_at).
- [Rule 1] `ReleaseGroupChefPayout` errored on nil gateway, breaking the reconcile settle-stamp contract; fixed to `return nil` like `ReleaseOrderPayouts`.
- [harness] Added `group_orders` DDL to `admin_payout_test.go` (500 fix; production code unchanged).

## No-leak confirmation
All group money seams gated on `payoutMovementEnabled()`: Hold (placement), Reverse (cancel), Release/Reverse (admin queue). `MarkGroupOrderDelivered` no longer calls Release — it parks. No group delivered/cancel path moves chef money while the escrow flag is off.

## Commit SHAs
- a77a2315 test(payments): failing tests for group-order payout hold aggregate (#456)
- 201da905 fix(payments): flag-gate group-order money seams + add payout hold columns (#456)
- a23db09c fix(payments): park group-order hold on delivery instead of releasing (#456)
- b9a82cba feat(payments): wire group-order aggType through release/reverse + host confirm (#456)
- d8273755 feat(payments): cover group orders in auto-confirm sweep + reconcile (#456)
- 3cdf637d feat(payments): admin payout queue accepts group-order aggType (#456)
- 93ee809f feat(payments): host confirm-received endpoint + guard cancel reverse (#456)
- b45b5e09 test(payments): green group-order payout suite + nil-gateway seam tolerance (#456)

## Follow-ups (out of scope)
- W-A deep fix: route CancelGroupOrder reverse through ReverseHold / inside the guarded tx before flags-ON.
- tesserix-home: "Group order" row label + host confirm UI.
- W-B / W-C done in this slice. Referenced: #457/#458/#460/#461/#462.
