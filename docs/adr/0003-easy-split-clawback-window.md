# ADR 0003 — The claw-back window for Easy Split

- **Status:** Accepted
- **Date:** 2026-08-07
- **Extends:** [ADR 0001](0001-vendor-payout-settlement-model.md), [ADR 0002](0002-automated-vendor-payouts.md)
- **Epic:** [#1076](https://github.com/tesserix/Home-Chef-App/issues/1076) — Cashfree Easy Split payout automation
- **Resolves:** [#1078](https://github.com/tesserix/Home-Chef-App/issues/1078) (blocker B2)

## Context

`BuildOrderSplit` (`services/easy_split.go`) sends `order_splits` on the create-order
call, so the chef's share is allocated at capture. That is regulatorily attractive —
chef money never enters the platform's account — but it steps around every control the
payout engine has. The maturation window (`services/payout_release_cron.go`),
`BlockRefundOpen`, `BlockRecoveryBalance`, `BlockNewChefRamp` and
`BlockAboveReviewThreshold` all run when a *payout* is released. A split order has no
payout to release.

The maturation window exists precisely so that a refund raised in the first hours can be
netted before the chef is paid. Splitting at capture removes it, and #1078 was raised to
decide what replaces it.

The issue framed the options as deferring settlement via `ScheduleOption`, porting the
governor's checks into `BuildOrderSplit`, or accepting the risk. Cashfree's own
documentation offers a fourth that is better than all three.

## What Cashfree actually provides

- **Split delay is on by default.** "By default, order level delay will be enabled and
  the default delay value is `01`" — a T+1 window after a successful payment "within
  which you have to specify the split details along with the vendor ID". Configured
  account-wide under Settings → Easy Split → Feature Configuration; extending it beyond
  the default goes through an account manager.
- **The split can be created after payment.** The Split API takes vendor id and amount
  (or percentage) once the payment has succeeded, rather than at order creation.
- **Not splitting is fail-safe.** "In the scenario where split is not invoked within the
  split delay range, the merchant receives the entire order amount" — it settles to the
  platform on the platform's own cycle, which is exactly the Rail B position we are in
  today.
- **Vendor and merchant legs settle on independent clocks.** `schedule_option` sets the
  vendor's cycle (standard T+2\*; option 6 is instant hourly on working days), and it is
  subject to bank approval rather than self-assignable.
- **Refunds already reach vendor balances.** Proportional debiting of vendor balances is
  on by default, and `refund_splits` overrides it (see #1077 / #1088 / #1089). There is
  also an on-demand balance and a merchant↔vendor transfer API.

Sources: [Order split delay](https://www.cashfree.com/docs/payments/split/settlements/delay/order-level),
[Easy Split overview](https://www.cashfree.com/docs/api-reference/payments/latest/split/easy-split-overview),
[Easy Split FAQ](https://www.cashfree.com/docs/payments/split/faq).

## Decision

**Split after payment, inside the split-delay window, gated by the existing governor.**
Stop sending `order_splits` at capture.

The order captures unsplit. When the governor would release a payout for that order — the
same decision, the same `BlockReason` set, the same maturation window — it calls the Split
API instead of building a platform-funded disbursement. If the governor blocks, we simply
do not call it, the window lapses, and the whole amount settles to the platform to be paid
through the Rail B payout engine exactly as it is now.

This keeps the regulatory benefit (a released chef's money is settled by Cashfree, never
custodied by us) while giving up none of the controls, because the release decision moves
*before* the split rather than being bypassed by it. The failure mode is the status quo
rather than an unrecoverable payment, which is the property worth having.

## Consequences

- The governor becomes the single decision point for both rails. `BuildOrderSplit` is no
  longer a second, weaker gate at checkout and loses its `ACTIVE`-vendor-plus-FSSAI
  check as a *release* decision; those become preconditions on choosing the split rail.
- The maturation window must fit inside the split delay. Our default is 2h against a
  default T+1 — comfortable. Any future maturation window beyond the delay silently
  degrades every order to Rail B, so the two values are coupled and must be asserted
  together in code, not held as folklore.
- Controls that can only fire *after* T+1 (a refund opened days later, a review threshold
  crossed next week) are still outside the window. They are not newly lost — Rail B has
  the same problem once a payout has been disbursed — and they remain covered by
  `refund_splits` debiting the vendor's balance and by recovery against future orders.
- Extending the delay window past T+1 is an account-manager conversation, not a code
  change. Worth having if the review threshold proves to need longer.

## Rejected

- **`ScheduleOption` as the claw-back mechanism.** It sets the vendor's settlement cycle,
  not a merchant-controllable hold, and it is subject to bank approval. It changes when
  the vendor is paid, not whether we can still decide.
- **Porting governor checks into `BuildOrderSplit`.** The checks that matter most are not
  knowable at checkout, and it would leave two implementations of one decision.
- **Accepting the risk with a per-chef rollout.** Unnecessary now that the delay window
  makes the governor's decision available before the money moves.
