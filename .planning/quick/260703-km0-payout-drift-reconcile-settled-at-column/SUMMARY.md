# Quick Task: Payout Drift Reconcile (settled_at column) — Summary

**GH #459.** Closes the released/reversed-but-unsettled payout drift: a durable, idempotent, flag-gated reconcile that re-drives holds whose money seam failed post-commit, plus a `payout_settled_at` column that decouples "status committed" from "money confirmed moved."

Branch: `feat/payout-drift-reconcile` (not pushed). 5 commits.

## Files changed and why

| File | Change |
|------|--------|
| `apps/api/models/order.go` | Added `PayoutSettledAt *time.Time` (`payout_settled_at`) + `PayoutSettleAttempts int` (`payout_settle_attempts;default:0`). AutoMigrate creates the columns at boot + in the sqlite harness. |
| `apps/api/models/meal_plan.go` | Same two fields on `MealPlanDay`. |
| `apps/api/migrations/20260703000003_add_payout_settled_at.up.sql` | Real `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` pair (orders + meal_plan_days), mirroring `20260703000001`. Header: released + settled_at set = success; released + settled_at NULL = drift. |
| `apps/api/migrations/20260703000003_add_payout_settled_at.down.sql` | Drops both columns on both tables. |
| `apps/api/services/payout_release.go` | Added `stampPayoutSettled` (conditional UPDATE on `settled_at IS NULL`), `settleRelease`, `settleReverse`. `ReleaseHold` returns `settleRelease(...)`, `ReverseHold` returns `settleReverse(...)` — settled_at stamped only after the seam returns nil; a seam failure returns the error and leaves settled_at NULL for the reconcile. |
| `apps/api/services/meal_plan_escrow.go` | `ReleaseDayPayout` tolerates an already-released transfer (`isAlreadyReleasedErr` → log + return nil). Flag-off / empty-transfer-id no-op guard unchanged. |
| `apps/api/services/payout_reconcile_cron.go` | New `payout-reconcile` cron: `recover()`, gated on `payoutMovementEnabled() || MealPlanEscrowActive()` (pure no-op when both off), four bounded sweeps over released/reversed orders (`razorpay_order_id <> ''`) + meal-plan days (`payout_transfer_id <> ''`) with `payout_settled_at IS NULL AND payout_settle_attempts < 5`, driven through shared `settleRelease`/`settleReverse`. Log-and-continue per row; DB-backed attempt cap + ALERT log at cap. |
| `apps/api/services/cron_temporal.go` | Registered `payout-reconcile` in `cronJobs()`. |
| `apps/api/services/payout_hold_test.go` | Harness: added both columns to `orders` + `meal_plan_days` DDL; added `seedReleasedOrder`. |
| `apps/api/services/payout_reconcile_cron_test.go` | New: 6 tests. |
| `apps/api/services/payout_release_test.go`, `apps/api/handlers/admin_payout_test.go`, `apps/api/handlers/chef_delivered_gate_test.go` | Deviation fix — added the two columns to their hand-DDL'd tables. |

## Verification output

- `go build ./...` — clean.
- `go vet ./services/` — clean.
- `gofmt -l` on all 11 touched files — empty (clean). Pre-existing unrelated files (campaign.go, messaging.go, menu.go, etc.) flagged by repo-wide gofmt were not touched — out of scope.
- `go test ./services/... ./handlers/...` — ALL-GREEN, `-count=1`.
- Existing #387/#388 tests confirmed still passing: `TestReleaseHold_*`, `TestReverseHold_*`, `TestReleasePayout_WritesAudit`, `TestBulkRelease_SkipsIneligible`, `TestPayoutAutoConfirm_*`, `TestSettleSaga_RegularOrder_HoldsNoRelease_FlagOn`, `TestConfirmOrderHold_*`, `TestUpdateOrderStatus_*`.

## Deviations from Plan

**1. [Rule 3 — Blocking regression] New model columns broke 5 hand-DDL'd test harnesses.**
- Found during Task 5. Adding the two model fields made GORM write them on a full-struct save; three test files hand-DDL `orders`/`meal_plan_days` without them, so saves failed (500 "Failed to update order"). Broke `TestReleaseHold_*`, `TestReverseHold_*`, `TestReleasePayout_WritesAudit`, `TestBulkRelease_SkipsIneligible`, `TestUpdateOrderStatus_*`.
- Fix: added the columns to each hand-DDL (same pattern the plan prescribed for `setupHoldDB`). No production-code change. Commit c6110ba0.

**2. [Test-detail adjustment] `settledAtOf` reads the model field, not raw SQL.**
- RED helper `db.Raw(...).Scan(&*time.Time)` errored scanning SQL NULL into `*time.Time` on sqlite; switched to `loadOrder(...).PayoutSettledAt`. Test-only.

## Commit SHAs

- `8b8efe50` test: add failing reconcile + settle + day-idempotency tests (RED)
- `c9a2554b` feat: add payout_settled_at column + shared settle helper stamping only after seam==nil
- `a7dd9356` feat: tolerate already-released transfer in ReleaseDayPayout for idempotent re-drive
- `32926d26` feat: add flag-gated payout-reconcile cron with attempt cap and register in cronJobs
- `c6110ba0` test: add payout_settled_at columns to payout hand-DDL test harnesses

## TDD Gate Compliance

RED (test: 8b8efe50) -> GREEN (feat: c9a2554b, a7dd9356, 32926d26). Gate sequence satisfied. No refactor commit needed.

## Follow-ups (NOT implemented here)

- True day idempotency: add a single-transfer `FetchTransfer` to the Razorpay client so `ReleaseDayPayout` verifies on-hold state instead of string-matching the gateway error. Durable guard remains `payout_settled_at IS NULL`.
- Bigger must-fix-before-flags gaps out of scope: #456 group-order gate, #457 refund/dispute cross-guard (the larger blockers before either escrow flag is turned on), plus #458, #460, #461, #462.

## Self-Check: PASSED

- `payout_reconcile_cron.go` (runPayoutReconcileScan), `models/order.go` + `models/meal_plan.go` (PayoutSettledAt), both migration files — FOUND.
- Commits 8b8efe50 / c9a2554b / a7dd9356 / 32926d26 / c6110ba0 — present in git log.
- `cron_temporal.go` cronJobs() contains `payout-reconcile`.
