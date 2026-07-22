# Payout Release Governor — Design

**Date:** 2026-07-22
**Status:** Approved
**Supersedes parts of:** ADR 0002 (Automated Vendor Payouts)
**Epic:** #736

## Problem

No chef has ever been paid, and the reason was not a missing payout engine.

Razorpay Route pays a chef by settling their linked account's balance to a bank
account **registered on that linked account**. Our onboarding created linked
accounts carrying identity only — `LinkedAccountRequest` had no bank fields and
nothing ever called Razorpay's product-configuration API. Every account we hold
can therefore receive transfers and has nowhere to pay them out to.

Separately, ADR 0002 specified a rail (RazorpayX push payouts) that the codebase
does not use. Route is already implemented and live: `handlers/payment.go`
creates held per-payee transfers at checkout, and `order_payout.go`,
`meal_plan_escrow.go` and `group_order_payout.go` release and reverse them.

## Decisions

### 1. Rail — Razorpay Route

Route is confirmed as the payout rail, superseding ADR 0002's RazorpayX choice.

The platform never custodies chef funds: the full amount is captured to the
platform account, a transfer moves only the chef's share to their linked
account, and the platform fee is never transferred — it stays in the balance and
settles to the company current account on the platform's own schedule. "Keeping
the platform fee aside" therefore requires no code.

**Consequence — UPI is not payable on this rail.** Route settles by NEFT/IMPS to
a bank account; a linked account has no UPI VPA destination. A chef who
nominates UPI cannot be registered, and is refused at registration rather than
being given an account that looks configured and can never pay out. UPI support
requires a RazorpayX adapter, deferred.

**Consequence — the chef's settlement timing is not ours to tune.** Razorpay
settles linked accounts in 2 working days *irrespective of the platform's own
settlement schedule*. Buying instant settlement for the platform does not speed
up the chef. Chef-facing copy must reflect roughly 2 days after release, not
hours.

### 2. Cadence — 2h maturation, 15-minute sweep

The maturation window drops from 24h to **2h**, and the daily batch is replaced
by a **sweep every 15 minutes**.

Rationale: our hold and Razorpay's 2-working-day settlement are sequential, not
concurrent. A 24h hold made the chef wait roughly three days. A 2h hold keeps
the total inside two days while preserving a window in which a late refund or
fraud signal can still cancel a release.

A 15-minute sweep gives a worst case of 2h15m, immaterial against a
2-working-day settlement, and it self-heals: a missed tick is simply picked up
by the next one, with no per-order timer state and no backfill path.

### 3. Release model — per order, not per batch

The daily batch (one payout per payee per business date) is the wrong shape for
Route, where money moves as per-payment transfers created at checkout. The
release governor decides **per order** whether that order's already-held
transfers are released.

`payouts/build.go` (PR #762) is retired. The merged `payouts` core — money,
ledger, rules, payee registry (#754) — is kept: it remains the record of what
each payee is owed and owes, and its payee-agnostic shape still serves drivers
and future payee types.

### 4. Guardrails

Evaluated per order, non-short-circuiting so an admin sees every reason at once
rather than resolving one and immediately hitting the next.

| Guardrail | Blocks when | Default | Overridable |
|---|---|---|---|
| Settlement not activated | chef `RazorpaySettlementStatus != "activated"` | — | No |
| Refund / dispute open | refund in flight, chargeback, or cancellation | — | No |
| Outstanding recovery | chef ledger balance is negative | — | Yes |
| New-chef ramp | fewer than N delivered orders | N = 3 | Yes |
| Value threshold | order total above limit | ₹5,000 | Yes |

The first two are hard blocks. Releasing to an unactivated account strands money
in a balance with no destination and no admin decision can fix that; releasing
against an open refund pays a chef for food the customer is being refunded for,
and the claw-back may land after the money has already settled to their bank.

### 5. Penalties and recovery

Route transfers are per-payment, so a debt cannot be netted across orders after
the fact. Recovery happens **at transfer creation**: `chefNetPayout` deducts the
chef's outstanding ledger balance from the next order's transfer before it is
created, floored at zero, with any remainder carried forward.

Penalties remain ledger debits as ADR 0002 specified. Only the collection
mechanism changes.

### 6. Attribution

No change required. Checkout already creates separate held transfers per payee,
each tagged with the order number and idempotency-keyed on
`(order, leg, account)`:

```go
{Account: chefAccount,   Amount: ToPaise(chefNetPayout(order)), Hold: true,
 Notes: {"purpose": "food_payment",     "order_number": order.OrderNumber}}
{Account: driverAccount, Amount: ToPaise(order.DeliveryFee + order.DriverTip), Hold: true,
 Notes: {"purpose": "delivery_payment", "order_number": order.OrderNumber}}
```

An existing guardrail already blanks the chef account when their FSSAI has
expired.

## Architecture

```
delivery confirmed ──▶ hold matures (2h) ──▶ sweep (≤15 min)
                                                  │
                                  ┌───────────────┴───────────────┐
                               clean                          blocked
                                  │                               │
                          ReleaseTransfer               release queue + reasons
                                  │                               │
                     Razorpay settles (2 working days)   admin releases / withholds
```

### Components

- **`services/route_onboarding.go`** — v2 Razorpay endpoints: create account,
  create stakeholder, request route product, attach settlement account.
  *Delivered.*
- **`services/route_registration.go`** — idempotent orchestration of that
  sequence; reuses an existing account or product rather than minting a second.
  *Delivered.*
- **Release governor** — evaluates the guardrail chain for one order and returns
  release or a set of block reasons. Pure: no database, no gateway, so the
  decision is reproducible from recorded inputs.
- **Sweep** — a Temporal Schedule every 15 minutes that selects matured
  delivered orders, applies the governor, and drives releases.
- **Recovery deduction** — `chefNetPayout` consults the ledger balance.
- **Admin surface** (tesserix-home) — chefs blocked from payout, the release
  queue, and per-chef payout history.

### Runtime settings

All `PlatformSettings` keys, changeable without a deploy:

| Key | Default |
|---|---|
| `payout.maturation_minutes` | `120` |
| `payout.sweep_enabled` | `false` |
| `payout.new_chef_ramp_orders` | `3` |
| `payout.review_above_paise` | `500000` |

`payout.sweep_enabled` defaults off. This moves live settlement, so it is
enabled deliberately after the Razorpay sandbox verification in #218.

## Durability

The sweep runs as a Temporal Schedule alongside the existing 19, so a failed
tick is retried rather than silently skipped.

Each release is idempotent on `(order, transfer)`. Razorpay rejects releasing an
already-released transfer, which is tolerated as a no-op — matching the existing
`isAlreadyReversedErr` treatment on the reversal path — so a re-drive after a
crash is safe.

Release and block events publish to NATS through the existing transactional
outbox, feeding the admin surface and analytics.

## Error handling

- **Gateway failure mid-release.** Return the error so the Temporal activity
  retries. Never stamp an order settled on a partial release; the next sweep
  re-drives it.
- **Partial registration.** `RegisterSettlementAccount` returns what it created
  before failing, so a retry resumes rather than creating a second linked
  account.
- **`needs_clarification`.** A review outcome, not an error. The requirements
  array is persisted verbatim and surfaced in the admin queue with Razorpay's
  resolution URL.
- **Chef never activates.** Transfers stay held indefinitely. This is visible in
  the blocked-chefs view rather than failing silently, because held money is not
  lost money.

## Testing

- **Governor:** table-driven per guardrail, plus a test asserting all reasons
  are reported together rather than the chain short-circuiting.
- **Sweep:** matured/unmatured boundary is inclusive at exactly 2h — an
  off-by-one silently delays every payout by a full cycle.
- **Idempotency:** re-driving a release moves no additional money.
- **Recovery:** a debt larger than the next order's payout floors at zero and
  carries the remainder forward; conservation asserted end to end.
- **Registration:** already covered — sequence order, account and product reuse,
  requirement propagation, missing bank account, partial failure.

## Open items

- Delivery-fee ownership for self-delivery chefs remains unresolved from ADR
  0002 and is unaffected by this design.
- RazorpayX adapter for UPI payees, deferred until there is demand from chefs
  who cannot supply a bank account.
- Whether a chef blocked on activation for a prolonged period should stop
  receiving orders, reusing the #739 payout gate.
