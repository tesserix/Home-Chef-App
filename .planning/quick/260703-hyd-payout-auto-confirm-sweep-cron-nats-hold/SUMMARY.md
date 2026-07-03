# Payout Auto-Confirm Sweep + NATS Hold Events — Summary

**Issue:** #387 follow-up (auto-confirm sweep + NATS hold events)
**Branch:** feat/payout-auto-confirm-sweep (stacked on the #387 hold-state-machine branch)
**Date:** 2026-07-03

## What shipped

Closed the loop so a payout hold can leave awaiting_customer_confirmation without a customer
tap, and gave #388 a durable event to consume — all behind the existing escrow flags (no live
money moves).

1. Two NATS subjects on the existing PAYMENTS stream (payments.>), no stream change:
   - payments.hold_release_eligible
   - payments.hold_disputed
2. Event emission inside the transition (applyHoldConfirm), gated on RowsAffected > 0, so BOTH
   the #387 customer-confirm endpoints AND the new sweep emit for free, exactly once.
3. Auto-confirm sweep cron (payout-auto-confirm, 15m interval) advancing delivered, dispute-free
   awaiting holds past the confirm window to release_eligible, and disputed ones to disputed.

## Files changed

- apps/api/services/nats.go — added SubjectHoldReleaseEligible + SubjectHoldDisputed constants
  near SubjectPaymentSuccess. No stream config change (PAYMENTS already covers payments.>).
- apps/api/services/payout_hold.go — refactored applyHoldConfirm to run the guarded UPDATE,
  inspect res.RowsAffected, and on a genuine transition emit the matching event within the same
  tx via new emitHoldEvent helper. Added aggType string param ("order"/"meal_plan_day"); updated
  both callers (ConfirmOrderHold / ConfirmMealPlanDayHold).
- apps/api/services/payout_auto_confirm_cron.go — NEW. payoutAutoConfirmInterval (15m),
  StartPayoutAutoConfirmCron (ticker + ctx.Done + panic-recover, mirrors meal_plan_cron.go),
  runPayoutAutoConfirmScan, and sweepOrders/sweepMealPlanDays helpers (bounded batch of 500,
  log-and-continue per row).
- apps/api/services/cron_temporal.go — registered {"payout-auto-confirm", ...} in cronJobs().
- apps/api/services/payout_hold_test.go — added outbox_events table + delivered_at column to
  setupHoldDB DDL; added countOutbox helper; added outbox assertions to the three existing
  confirm tests (A/B/C).
- apps/api/services/payout_auto_confirm_cron_test.go — NEW. Tests D–I for the sweep
  (past-window release, inside-window untouched, open-issue->disputed, meal-plan day,
  48h override respected, idempotent across two runs).

## Invariants honored

- Disputed never reaches release_eligible (T-387f-01): sweep passes NO disputed flag;
  ConfirmOrderHold/ConfirmMealPlanDayHold run HasOpenOrderIssue, and the release UPDATE is
  guarded on payout_hold_status = 'awaiting_customer_confirmation' — a disputed row fails the WHERE.
- Exactly-once emit (T-387f-02): emit only on res.RowsAffected > 0 + existing
  CustomerConfirmedAt != nil early-return; each outbox row carries a unique MsgID (JetStream dedup).
- Bounded sweep (T-387f-03): Limit 500 per scan, 15m interval, panic-recover, log-and-continue.
- release_eligible moves no money here (that is #388). No Razorpay / ReleaseTransfer calls added.

## Verification

- gofmt -l (touched files): clean (no output)
- go build ./...: BUILD-OK
- go vet ./services/: VET-OK
- go test ./services/... ./handlers/...:
    ok  github.com/homechef/api/services  0.708s
    ok  github.com/homechef/api/handlers  0.817s

All 15 hold/sweep tests pass, including every existing #387 confirm test
(TestConfirmOrderHold_*, TestConfirmMealPlanDayHold, TestSettleSaga_*) after the shared
applyHoldConfirm change.

## Deviations

None. Plan executed as written. Minor: added a named sweepBatchLimit = 500 const for the
bounded batch called out in the threat model (T-387f-03).

## Commits

- 677ab1a3 test: add failing tests for hold events + auto-confirm sweep (#387 follow-up)
- c4f000bd feat: emit hold events inside the confirm transition + add payout hold subjects (#387)
- accb8716 feat: add payout auto-confirm sweep cron + register in cronJobs (#387)

## Follow-ups (out of scope)

- #388 — admin release queue consuming release_eligible + Razorpay ReleaseTransfer.
- #400 — OrderIssue->hold linkage that freezes an ALREADY release_eligible hold.
- Group-order holds; mobile UI; refund/reversal changes.
