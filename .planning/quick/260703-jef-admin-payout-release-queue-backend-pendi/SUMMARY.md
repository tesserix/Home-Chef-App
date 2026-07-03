# Admin Payout Release Queue (backend actuator) — Summary

**Issue:** GH #388 — admin payout release queue (backend / actuator)
**Branch:** `feat/payout-admin-release-queue` (no push)
**Scope:** `apps/api` (Go) only. tesserix-home UI is a separate slice.
**One-liner:** Race-safe, flag-gated admin actuator that turns `release_eligible`
payout holds (#387) into real vendor payouts — list / release / withhold / reverse
(single + bulk), driving the existing Razorpay `ReleaseTransfer` / `ReverseTransfer`
seams behind the escrow flags, fully audited.

## What shipped

Backend endpoints under the existing `/admin` group (`bffAuth` + `RequireAdmin`):

| Method | Route | Action |
|--------|-------|--------|
| GET  | `/admin/payouts/pending` | List `release_eligible` holds (awaiting via `?include=awaiting`; `?chefId`, `?before`) |
| POST | `/admin/payouts/:aggType/:id/release` | `release_eligible -> released` + flag-gated release seam |
| POST | `/admin/payouts/:aggType/:id/withhold` | `{release_eligible, awaiting} -> withheld` (mandatory reason) |
| POST | `/admin/payouts/:aggType/:id/reverse` | `{released, release_eligible} -> reversed` + flag-gated reverse seam (mandatory reason) |
| POST | `/admin/payouts/release-bulk` | Bulk release; skips ineligible with a report |

`aggType` in {`order`, `meal-plan-day`}.

## Money-safety design (the point of the slice)

- The status flip is a single **conditional `UPDATE ... WHERE payout_hold_status IN (from...)`**
  guarded by `RowsAffected`. Zero rows -> `ErrHoldNotEligible` -> handler **409**. Two
  concurrent releases: exactly one wins.
- The status flip **commits BEFORE any Razorpay call** (`transitionHold` runs in a
  short `db.Transaction`; the money seam is dispatched post-commit). A non-eligible /
  already-actioned hold can never reach `ReleaseTransfer` / `ReverseTransfer`.
- Release only from `release_eligible`. `withheld` / `reversed` are terminal — excluded
  from the pending queue and from re-release.
- All money seams are **flag-gated** (`ReleaseOrderPayouts` / `ReverseOrderPayouts` on
  `OrderPayoutAutoReleaseEnabled`; `ReleaseDayPayout` / `ReverseTransfer(id, 0)` on
  `MealPlanEscrowActive()`, guarded on `PayoutTransferID != ""` + gateway configured).
  Launch config = both OFF => every action is a **DB-only state advance, zero money moved.**
- Every release/withhold/reverse is audited via `services.LogAudit` (actor + old->new +
  mandatory reason for withhold/reverse). Withhold/reverse enforce a non-empty reason
  (400) at the handler and defensively in the service.

## Partial-failure drift — honest note (must-fix before flags ON)

The money seam is dispatched **after** the tx commits. If that post-commit Razorpay
call fails, the row is left **`released` with the money UNMOVED**, and there is **NO
in-slice re-drive** — nothing here retries the stranded row. The out-of-scope
auto-approve sweep does **not** cover this: it only ever operates on `release_eligible`
rows, so it will never re-drive a `released`-but-unpaid row. This is safe **only**
because both escrow flags are OFF at launch (the seam is a no-op and cannot fail). This
honest caveat is recorded in a code comment at the top of `services/payout_release.go`.
**Before either flag is turned ON, a reconcile/re-drive path for `released`-but-unpaid
holds MUST be built.**

## Files changed

**New**
- `apps/api/services/payout_release_test.go` — RED service tests (list/release/withhold/reverse/getter).
- `apps/api/handlers/admin_payout_test.go` — RED handler tests (200/409/400/audit/bulk).
- `apps/api/services/payout_release.go` — `ListPendingPayouts`, `transitionHold`,
  `ReleaseHold`, `WithholdHold`, `ReverseHold`, `GetPayoutAutoApproveHours`, `ErrHoldNotEligible`.
- `apps/api/handlers/admin_payout.go` — `AdminPayoutHandler` + 5 methods, structured JSON errors, audit.
- `apps/api/migrations/20260703000002_add_payout_withheld_reversed_states.up.sql` / `.down.sql` — doc-only DDL (varchar values, no schema change, reason lives in `audit_logs`).

**Edited**
- `apps/api/models/payout_hold.go` — added `PayoutHoldWithheld` / `PayoutHoldReversed` (terminal).
- `apps/api/services/nats.go` — added `SubjectHoldReleased = "payments.hold_released"`.
- `apps/api/routes/routes.go` — constructed `adminPayoutHandler`; added 5 routes inside the `admin` (RequireAdmin) group.

## Plan-check fixes applied

1. **RED harness DDL:** the cloned sqlite harness (`setupReleaseDB` / `setupPayoutHandlerDB`)
   adds `order_number` on `orders` and `meal_plan_number` on `meal_plans` (both selected by
   `ListPendingPayouts`), plus `chef_id`/`total`/`price` and an `audit_logs` table — so the
   service and handler tests actually pass.
2. **Partial-failure drift:** documented honestly (above) in SUMMARY + a code-comment block;
   did NOT build the reconcile path (listed as must-fix-before-flags-ON follow-up).

## Verification (from `apps/api`)

- `go build ./...` — clean.
- `go vet ./services/ ./handlers/ ./routes/` — clean.
- `gofmt -l services/payout_release.go handlers/admin_payout.go` — empty (formatted).
- `go test ./services/... ./handlers/...` — `ok` both packages.
- `go test ./...` — exit 0, no FAIL/panic (no regressions; confirms no gin route conflict).
- All functions < 50 lines (largest: `transitionHold` 30, `BulkReleasePayouts` 25); `%w`-wrapped errors; structured JSON envelopes; no handler panics.

## Deviations

- **GH #388 issue comment (plan Task 5) NOT posted.** The spawning agent's deliverables
  did not request it and emphasized no external pushes; skipped to avoid an unrequested
  side-effect. Recommend posting a summary comment manually (what shipped + flags-OFF =
  DB-only + follow-ups below).
- `AgeHours` is surfaced on the queue DTO but no SLA-threshold logic is enforced in this
  slice (review DTO only).

## Commits

- `23c9996a` test(388): failing tests for admin payout release queue (RED)
- `28c72b1e` feat(388): add withheld/reversed hold states, hold_released subject, doc migration
- `1baf20bc` feat(388): payout release service (list/release/withhold/reverse, race-safe)
- `f4610ef2` feat(388): admin payout endpoints + routes (list/release/withhold/reverse/bulk)

## Follow-ups (out of scope — do NOT build here)

- **tesserix-home admin UI** — payout queue page (separate repo slice).
- **Auto-approve sweep** — only the `GetPayoutAutoApproveHours` getter (default 0 = disabled) shipped.
- **Drift reconcile/re-drive before flags ON** — retry path for `released`-but-unpaid holds (blocking for enabling either escrow flag).
- **#400 late-dispute freeze** of already-eligible holds.
- **Group-order holds** and refund pairing beyond the raw `ReverseTransfer` call.
- Persisting withhold/reverse **reason as a column** (currently audit-only).
