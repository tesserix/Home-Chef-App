# Checkout Credits — Wallet + Loyalty at Checkout

**Date:** 2026-07-25
**Status:** Approved, ready for implementation planning
**Scope of this spec:** Pass 1 — on-demand orders only. Group orders, meal-plan
advances and catering deposits are Pass 2, built on the same engine.

---

## Problem

Three defects, one root cause.

**1. Credit pays fees and tax it must never pay.**
`handlers/payment.go:142` plans the funding split against `order.Total`:

```go
plan := services.PlanWalletFunding(totalPaise, balancePaise, requestedWalletPaise, settlements)
```

The ceiling is the whole total, so wallet credit freely consumes the platform
service fee and GST. The owner's rule is that GST and the platform fee are always
settled in real money — the platform remits GST to the government and the service
fee is its revenue; neither may be funded by a liability the platform itself
issued. Nothing in the codebase currently expresses a "redeemable base".

**2. The client decides the money, so the UI and the charge disagree.**
`apps/mobile-customer/app/checkout.tsx:560` computes the applied amount locally:

```ts
const walletApplied = applyWallet ? Math.min(walletBalance, total) : 0;
const payable = Math.max(0, total - walletApplied);
```

and posts that rupee figure to the server, which then independently re-clamps it
against the *live* wallet balance and the *real* `order.Total`. Any drift between
the two — a stale cached balance, a delivery fee recomputed server-side, a
chef-adjusted fee — leaves the server with a non-zero capture while the screen has
already rendered "To pay ₹0". This is the reported bug: an order showing ₹0
outstanding still opened the Razorpay sheet.

The defect is structural. Two independent parties compute the same money from
different inputs, and only one of them charges the card.

**3. Loyalty points cannot be spent at checkout at all, and their caps are fiction.**
Points redeem only to wallet credit, from the loyalty screen, via
`handlers/loyalty.go:RedeemLoyalty`. Separately, `LoyaltyConfig.MaxRedeemPct`
(10% of food subtotal) and `LoyaltyConfig.MonthlyRedeemCap` (₹300 per rolling 30
days) are parsed from platform settings and editable from the admin UI
(`handlers/admin_loyalty.go:76-80`) but are **enforced nowhere in the codebase**.
Their struct comment already says "checkout, Phase 3" — this is that phase.

There is also a discoverability failure: the wallet control is a bare checkbox
wedged between the promo-code field and the "To pay" row (`checkout.tsx:1329`),
which the owner missed entirely while placing a live order.

---

## Money model

### Composition of an order total

From `handlers/orders.go:496-498`:

```
Total = Subtotal + DeliveryFee + ServiceFee + Tax + Tip − Discount
```

with two tax regimes (line 486):

- **Exclusive** — `Tax` is computed on the base and *added* to `Total`.
- **Inclusive** — `Tax` is already embedded inside `Subtotal` and is *not* added
  to `Total`.

`ServiceFee` is what the customer-facing UI labels "Platform fee". `Tax` is
rendered split as CGST + SGST but stored as one field.

### The redeemable ceiling

```
RedeemableCap = max(0, min( Subtotal − Discount + DeliveryFee,
                            Total − ServiceFee − Tax ))
```

Both expressions are computed and the smaller wins. This is deliberate, and each
branch earns its place:

- The **first** branch encodes the product rule directly: credit covers food and
  delivery, nothing else.
- The **second** branch is a structural guarantee. It makes
  `payable ≥ ServiceFee + Tax` hold *by construction*, whatever else `Total` may
  contain now or later. It is what keeps a future component added to `Total` from
  silently becoming redeemable.

Two consequences fall out of taking the minimum, both correct and both tested:

- **Tip is excluded.** A tip is a gratuity destined for the chef or rider, not
  food. The first branch omits it, so it is never redeemable.
- **Inclusive tax is handled.** In inclusive mode the first branch would include
  the tax embedded in `Subtotal`; the second branch is smaller and wins, so GST
  is still paid in cash under either regime.

### Allocation — wallet first

```
walletMax  = min(WalletBalance, RedeemableCap)
wallet     = walletMax                                      ← default
loyaltyMax = min( PointsBalance × RedeemRate,
                  MaxRedeemPct × Subtotal,
                  MonthlyRemaining,
                  RedeemableCap − wallet )
loyalty    = loyaltyMax                                     ← default
payable    = Total − wallet − loyalty
```

**Wallet takes precedence over loyalty.** Wallet credit is a booked liability the
platform has already issued; loyalty points are a capped, expiring promotional
currency. Burning the liability first is the owner's stated preference and the
better balance-sheet outcome. It also means that on a small order the points
simply survive: a ₹340 redeemable order against a ₹500 wallet takes ₹340 from the
wallet, leaves `RedeemableCap − wallet = 0`, and spends zero points.

**Defaults are order-sized, not balance-sized.** Each row shows what will actually
be applied to *this* order, never the raw balance. A ₹500 wallet on a ₹760
redeemable order shows ₹500 applied; the same wallet on a ₹340 order shows ₹340
applied. Maximum burn by default, in both directions.

**The 500-point `MinRedeem` floor does not apply at checkout.** It exists to stop
dust transactions on the standalone redeem-to-wallet action, where it remains in
force. At checkout any balance is spendable down to a single point, so the slider
is a clean `0 → max` range and a customer with 180 points sees a usable row.

Loyalty is floored to whole points, so the rupee figure is always an exact
multiple of `RedeemRate`.

### The monthly cap counts both redemption routes

`MonthlyRemaining = MonthlyRedeemCap − (₹ redeemed in the last 30 days)`, where
"redeemed" spans **both** checkout redemptions and standalone redeem-to-wallet
redemptions. One cap, one pool — otherwise a customer could take ₹300 through each
route and draw ₹600 a month against a ₹300 limit.

No query for this exists today, because the cap has never been enforced. It needs
a new helper summing `loyalty_transactions` of the redemption sources over a
rolling 30-day window, keyed by user.

### Credit is INR-only

Wallet and loyalty are rupee instruments settled through Razorpay Route. Orders
routed to Stripe — chefs whose `PaymentProvider` is `stripe`, priced in their
own `PayoutCountry` currency (`handlers/payment.go`) — take **no credit at all**:
the quote returns `walletEnabled: false, loyaltyEnabled: false` with the full
total payable, and `createStripePayment` is untouched. The existing code already
declines to pass wallet down the Stripe path; this makes the reason explicit and
surfaces it to the UI so the card is hidden rather than shown broken.

### Slider interaction rule

Until the customer touches either control, both rows are **auto**: wallet maxed,
loyalty filling whatever remains. The moment they touch either one, both become
**explicit** and hold their values, each still clamped live against the ceiling.

The rule exists to prevent a specific surprise: without it, dragging wallet down
to ₹0 would cause loyalty to silently expand and consume points the customer was
deliberately trying to preserve. Predictable beats clever.

---

## Architecture

### One engine

New file `apps/api/services/checkout_credit.go`. Pure — no DB, no gateway calls —
and working in paise throughout to stay exact, matching the existing convention in
`services/wallet_split.go`.

```go
// CreditInputs is everything the allocation needs, in paise.
type CreditInputs struct {
    SubtotalPaise, DiscountPaise, DeliveryFeePaise int
    ServiceFeePaise, TaxPaise, TotalPaise          int
    WalletBalancePaise                             int
    PointsBalance                                  float64
    RequestedWalletPaise                           *int      // nil = auto (max)
    RequestedPoints                                *float64  // nil = auto (max)
    Cfg                                            LoyaltyConfig
    MonthlyRedeemedPaise                           int
}

// CreditQuote is the authoritative breakdown. The client renders it verbatim
// and never recomputes any field.
type CreditQuote struct {
    RedeemableCapPaise   int
    NonRedeemablePaise   int   // ServiceFee + Tax — always cash
    WalletBalancePaise   int
    WalletAppliedPaise   int
    WalletMaxPaise       int   // slider upper bound
    PointsBalance        float64
    PointsAppliedPoints  float64
    PointsAppliedPaise   int
    PointsMaxPoints      float64 // slider upper bound
    PayablePaise         int
    LoyaltyLimitReason   string  // "balance" | "per_order_cap" | "monthly_cap" | "order_covered"
    WalletEnabled        bool
    LoyaltyEnabled       bool
}

func PlanCheckoutCredit(in CreditInputs) CreditQuote
```

`LoyaltyLimitReason` exists so the UI can explain *why* a row is capped without
reimplementing the cap logic to guess.

### Settlement is unchanged in shape

Loyalty behaves exactly like wallet at the gateway: a platform-funded discount
that reduces the capture but never the chef's or driver's payout.
`services.PlanWalletFunding` already tops up each settlement from the platform
balance when the capture cannot cover it; it simply receives
`wallet + loyalty` as the credit figure instead of `wallet` alone.

One consequence deserves naming explicitly: because `payable ≥ ServiceFee + Tax`
and both are positive on a real order, the **capture is now always non-zero**, so
the full-credit "no gateway payment at all" path (`settleFullWalletOrder`) can no
longer trigger for a normal on-demand order. That is the correct outcome of the
owner's rule, and it means the reported Bug 2 symptom is eliminated by
construction rather than patched. The path is retained for zero-fee
configurations and must keep its tests.

A second consequence is operational, not code: a heavily credit-funded order
leaves a small capture funding a large chef payout, so more of the settlement
flows out of the platform's Razorpay float via `DirectTopUps`. This is existing
behaviour that the cap makes *less* extreme (credit can no longer reach 100% of
the total), but the float headroom should be watched after rollout.

### Endpoints

**New — `POST /v1/payments/order/:orderId/quote`**

```json
{ "useWallet": true, "walletAmount": 300.00,
  "useLoyalty": true, "loyaltyPoints": 1200 }
```

All four fields optional. `useX: true` with no amount means auto (max). Returns
the full `CreditQuote` plus `walletEnabled` / `loyaltyEnabled`. Read-only —
debits nothing, mutates nothing, safe to call on every slider change (debounced).

**It must not sit under the payment rate limiter.** The `/payments` group applies
`middleware.RateLimitByUser(2, 5)` (`routes/routes.go:897`) — two requests per
second, burst five — which is correct for mint/verify/refund but would throttle a
customer simply dragging a slider. `/quote` is read-only and idempotent, so it
gets its own materially more generous per-user limit. Debouncing on the client
reduces the call rate but cannot be relied on as the only defence.

**Changed — `POST /v1/payments/order/:orderId/create`**

Accepts the identical body shape and **recomputes the allocation server-side**
from the live balance and the real order. The client never sends a payable and
its numbers are never trusted. Backwards compatible: an older build posting a
bare `{"walletAmount": N}` is read as `useWallet: true` with that amount, then
clamped by the new cap.

Serving `walletEnabled` / `loyaltyEnabled` from the quote response retires the
build-time `EXPO_PUBLIC_WALLET_CHECKOUT_ENABLED` constant
(`checkout.tsx:92`), which today must be moved in lockstep with the API's
`WALLET_CHECKOUT_ENABLED` env or the client and server disagree about whether the
feature exists at all. After this change the server is the sole authority and a
stale app build cannot desync.

### Feature flags

- `WALLET_CHECKOUT_ENABLED` — default flips `false` → **`true`**
  (`config/config.go:235`).
- `LOYALTY_CHECKOUT_ENABLED` — new, defaults **`true`**.

Both remain overridable per environment, satisfying "always enabled by default
unless turned off". The customer's own per-row toggle choice persists client-side
in the existing Zustand store, so a customer who switches a row off keeps it off
on their next order.

---

## Refunds

### Funding is recorded per order

Four new `orders` columns. Per the workspace rule, the DDL lives in
`tesserix-k8s/charts/apps/db-schema-bootstrap/schemas/homechef/homechef/` — never
in this repo — and the GORM model here is updated to match.

| Column | Type | Purpose |
|---|---|---|
| `loyalty_applied` | `numeric(10,2) default 0` | ₹ of the total funded by points |
| `loyalty_points_spent` | `numeric(12,2) default 0` | points debited |
| `wallet_refunded` | `numeric(10,2) default 0` | ₹ already returned to wallet |
| `loyalty_refunded` | `numeric(10,2) default 0` | ₹ already returned for the loyalty slice |

`wallet_applied` already exists. The card-funded share is derived, never stored:
`Total − wallet_applied − loyalty_applied`.

The two `*_refunded` columns exist so that **repeated partial refunds can never
over-return a source**. Without them, two 60% refunds would each independently
compute 60% of the original wallet slice and return 120% of it.

### Pro-rata allocation

For a refund of `R` on an order funded `(W, L, C)` out of `Total`:

```
wallet_back  = round2(R × W / Total)
loyalty_back = round2(R × L / Total)
card_back    = R − wallet_back − loyalty_back
```

Each source is returned its exact proportion of whatever is being refunded. The
**card slice absorbs the rounding remainder** rather than each slice rounding
independently, which guarantees the three parts sum to exactly `R` — no
half-paise is ever created or destroyed. Every slice is additionally clamped so
that `cumulative_returned ≤ amount_funded` for that source.

This matters because the on-demand refundable base is the **whole total**, not the
food-only base. `services/refundable.go:82` returns `Total − refunded` (adjusted
for per-line cancels), and the cancellation policy then applies a percentage. So a
refund is routinely *smaller* than the credit applied, and the split rule is load-
bearing rather than cosmetic. (The fee-and-GST-exclusive refund base is
meal-plan-specific, in `meal_plan_refund_v2.go`, and is out of scope here.)

### Destinations

- **Wallet slice** → credited straight back to the wallet, automatically, no
  prompt. Instant.
- **Loyalty slice** → credited to the **wallet as rupees**, not restored as
  points. Owner decision.
- **Card slice** → enters the existing RBI choice flow unchanged: the customer
  picks wallet (instant) or original payment method (5–7 days), with the
  guideline text already shipped.

**The monthly redemption cap is *not* released on refund.** This is the guardrail
on the loyalty-to-wallet conversion: without it, a redeem-then-cancel loop would
be an unlimited pump converting capped, expiring points into uncapped permanent
wallet credit. With the cap held, the conversion is bounded to `MonthlyRedeemCap`
(₹300) per customer per rolling 30 days — no more than they could have spent
legitimately — and cancellation fees make the loop lossy. Bounded and
self-limiting; no further machinery needed.

`ReverseOrderLoyalty` in `services/loyalty_refund.go` is untouched. It claws back
points *earned* on a refunded order, which is a separate concern from returning
points *spent*, and the two must not be conflated.

---

## Checkout UI

`apps/mobile-customer/app/checkout.tsx`. The bare checkbox at line 1329 is
replaced by a titled card placed **above** Price Details, not buried beneath the
promo field:

```
Pay with your credits
──────────────────────────────────────────────
  Wallet credit                    ₹2226.00  ●
  Balance ₹2226.00
  ●────────────────────────────────●  −₹2226.00
  0                           2226

  Loyalty points                  1,240 pts  ●
  Worth ₹62.00 · up to ₹258 on this order
  ●────────────────────────────────●    −₹62.00
  0                           1,240
──────────────────────────────────────────────
  Credits applied                    −₹2288.00
  Fees & taxes are always paid separately
  (₹274.18)

  To pay                               ₹766.18
```

Each row carries a toggle, a slider, and a tap-to-type numeric field. Every
figure — including `To pay` — comes from the quote endpoint; the screen performs
no money arithmetic of its own.

Per `.impeccable.md`: persimmon accent used only for the active slider track and
applied amounts, hairline separators rather than bordered cards, tabular numerals
on every figure, 44px touch targets, `cubic-bezier(0.22, 1, 0.36, 1)` easing with
no bounce, and `prefers-reduced-motion` honoured.

Slider input is debounced before hitting `/quote`; the applied figures update
optimistically from the last known ceiling and reconcile when the response lands.
A failed quote call falls back to the last good quote and disables Place Order
rather than guessing a payable.

---

## Testing

**Property test — the core invariant.** Across randomised inputs spanning both tax
regimes, zero and non-zero tips, discounts exceeding subtotal, zero balances and
balances far exceeding the total:

```
payable ≥ ServiceFee + Tax          (always)
wallet + loyalty ≤ RedeemableCap    (always)
wallet + loyalty + payable == Total (always, to the paise)
```

**Table tests — allocation.** Wallet-first precedence; the ₹340-order/₹500-wallet
case leaving points untouched; each of the four loyalty ceilings binding in turn
with the right `LoyaltyLimitReason`; whole-point flooring; `MinRedeem` waived at
checkout but still enforced on redeem-to-wallet.

**Table tests — refund pro-rata.** Rounding remainder lands on the card slice and
the parts sum to `R` exactly; repeated partial refunds never exceed a source's
funded amount; a refund on a fully-credit-funded order returns zero to the card;
the monthly cap stays consumed after a loyalty-slice refund.

**Regression for Bug 2, written first and failing first.** A test that drives the
old client-computed path — a client-supplied `walletAmount` that disagrees with
what the server would compute — and asserts the server's response payable is
authoritative and that no Razorpay order is minted when the payable is zero. It
must fail against current `main` before the fix lands.

**Cap-enforcement regression.** `MaxRedeemPct` and `MonthlyRedeemCap` are proven
to actually bind — they never have before.

**End-to-end on the Android emulator against prod data**, following the wallet E2E
approach already used for the refund-v2 work: place an order part-funded by wallet
and loyalty, confirm the Razorpay sheet opens for exactly the fees-and-tax
remainder, then cancel it and confirm each source is returned its pro-rata share.

---

## Out of scope for Pass 1

Group-order shares (`handlers/group_order.go:PayGroupShare`), meal-plan escrow
advances (`handlers/meal_plan.go`), and catering deposits
(`handlers/catering.go:CreateDeposit`) each mint their own Razorpay order on a
separate path. They consume the same engine in Pass 2, once it has been proven in
production on the on-demand path. Catering additionally needs a product decision
about what "food price" means for a percentage deposit against a quote, which is
not settled here.
