# W-A — CancelGroupOrder reverse runs outside the guarded tx (route through the hold machine)

**#456 follow-up. Flag-gated (payoutMovementEnabled OFF ⇒ ReverseGroupChefPayout is a no-op). Must-fix
before ORDER_PAYOUT_AUTO_RELEASE_ENABLED flips ON.**

## Problem
`handlers/group_order.go` CancelGroupOrder calls `services.ReverseGroupChefPayout(&g)` **before** and
**outside** the guarded cancel transaction (group_order.go:876), unconditionally:
- It runs even when the in-tx status flip no-ops (`WHERE status NOT IN (cancelled,delivered)`
  RowsAffected==0 → errGroupConflict) — a racing double-cancel could fire the reverse twice / on a
  non-cancel.
- It never updates `payout_hold_status`, so the group hold state is inconsistent with the money seam
  (the admin payout queue / reconcile cron key on `payout_hold_status`).

## Key facts (established by code read)
- `HoldGroupChefPayout` (group_order.go:818, at consolidation/lock) sets `PayoutTransferID` but **NOT**
  `payout_hold_status`. So a **pre-delivery** group cancel has `payout_hold_status='' (none)` yet a
  held transfer. `parkGroupOrderOnDelivery`→`SetGroupOrderHoldAwaitingConfirmation` sets `awaiting`
  only on delivery.
- The group chef transfer is a **DIRECT** transfer (`CreateTransfer`/DirectTransferRequest), NOT a
  payment-linked one. Unlike order Route transfers (auto-reversed by Razorpay on the payment refund),
  a direct transfer is NOT auto-reversed — so on cancel it MUST be explicitly reversed. This is why
  the group path reverses explicitly today (and why order "withhold = leave on-hold" doesn't apply).
- `transitionHold(db, aggType, id, from[], to, emit)` and `settleReverse(db, aggType, id)` are already
  generalized over aggType on main; `reverseMoney`'s group branch calls `ReverseGroupChefPayout` (keys
  on `PayoutTransferID`, flag-gated). So driving the group hold → `reversed` then `settleReverse`
  reverses the held transfer AND stamps `payout_settled_at`.

## Design
New `services.ReverseGroupHoldForCancel(db, groupID, reason) error`:
1. Guarded status transition (its own short tx via `transitionHold`): any non-terminal
   {none, awaiting_customer_confirmation, release_eligible, released, disputed} → `reversed`.
   RowsAffected==0 (already withheld/reversed) → no-op (idempotent; a double-cancel can't double-drive).
2. On a genuine transition, `settleReverse(db, aggTypeGroupOrder, groupID)` → `reverseMoney` reverses
   the held direct transfer (flag-gated) + stamps settled. (Reverses whenever `PayoutTransferID` is set
   — covers the pre-delivery none-with-transfer case the status alone would miss.)

Wire into CancelGroupOrder:
- REMOVE the pre-tx `services.ReverseGroupChefPayout(&g)` (group_order.go:876).
- AFTER the cancel tx commits (err==nil, i.e. the guarded flip genuinely cancelled), call
  `services.ReverseGroupHoldForCancel(database.DB, g.ID, "group order cancelled")` best-effort (log on
  error; the payout-reconcile cron re-drives a reversed-but-unsettled group hold). Now the reverse is:
  conditional on a real cancel, race-safe (guarded transition), and status-consistent.

Note: `transitionHold` opens its own tx, so it must be called with `database.DB` AFTER the cancel tx
commits (not nested inside it) — mirrors how the order refund crossguard is invoked post-commit.

## Interaction with open PRs
- #508 (PR #530) changed `settleReverse`→`settlePayout` (re-read) and made `ReverseGroupChefPayout`
  return an error + tolerate already-reversed. This branch is off main (no #508). `settleReverse` on
  main = `reverseMoney`+stamp; works either way (if #508 merges first, `settleReverse` re-reads the
  `reversed` status we just set → reverseMoney — consistent). Also touches group_order.go's cancel
  block (#508 added error-logging to the `ReverseGroupChefPayout` call I'm removing) → a trivial merge
  conflict; note in the PR that W-A supersedes that line. Recommend merging #530 first.

## Tests (RED-first, sqlite harness)
- ReverseGroupHoldForCancel: none→reversed (pre-delivery held transfer), awaiting→reversed,
  release_eligible→reversed, released→reversed (+settled stamped); already withheld/reversed → no-op.
- Idempotent double-call → single transition, settled once.
- Handler-level (if feasible): CancelGroupOrder drives the group hold to reversed; a conflicting
  double-cancel doesn't double-drive.

## Money-path: plan-check this before implementing (esp. the "reverse from none/any-non-terminal"
## decision and the direct-vs-linked-transfer reasoning). Then RED-first → independent verify.
