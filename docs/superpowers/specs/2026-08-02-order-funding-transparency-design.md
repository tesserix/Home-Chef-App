# Order funding transparency — design

**Date:** 2026-08-02
**Status:** approved, ready for implementation plan
**Branch:** `feat/order-funding-transparency`

## The report

A cancelled order showed `₹327 refunded to your card` above a breakdown reading
`Total ₹326.64 / Refunded −₹326.64`, while the Cashfree dashboard showed the
transaction **and** the refund as **₹324.14**.

## What is actually happening

**The money is correct. Only the display is wrong.**

`services/wallet_split.go` charges the gateway `CapturePaise = Total − WalletApplied
− LoyaltyApplied`. This order had **₹2.50** of credit applied, so:

| | |
|---|---|
| Order total | ₹326.64 |
| Credit applied at checkout | ₹2.50 |
| Charged at the gateway | **₹324.14** — matches Cashfree |
| Refunded via the gateway | **₹324.14** — matches Cashfree |
| Returned to wallet | ₹2.50 |

`services/refund_funding_split.go` `SplitRefundByFunding` already divides every
refund pro-rata across wallet, loyalty and card, capping each rail at what it has
not already returned. Nothing is lost and the customer is not short.

## The four defects, all presentational

1. **Rounding.** `apps/mobile-customer/app/order/[id]/index.tsx:500` renders
   `order.refundAmount.toFixed(0)` — "₹327" sits directly above "₹326.64". Two
   numbers for the same thing on one screen.
2. **False destination.** The copy says the amount went to "your original payment
   method". ₹2.50 of it went to the wallet. This sentence is what makes the
   Cashfree figure read as a discrepancy.
3. **The credit is invisible.** The breakdown lists Subtotal, Delivery, Platform
   fee, Tax, Discount, Total — and no credit line. The screen cannot be
   reconciled against a card statement or a gateway dashboard by anyone.
4. **The wallet balance is stale.** `['wallet']` is invalidated by loyalty
   redemption (`hooks/useLoyalty.ts:103`), meal plans (`useMealPlans.ts:307`) and
   report-issue (`useReportIssue.ts:48`) — but **not** by order cancellation,
   refund, or checkout. The wallet moves server-side and the chip never refetches.

Defect 4 is why 1–3 cannot be fixed alone: a "₹2.50 returned to your wallet" line
that sits under a stale balance is worse than saying nothing.

## Scoping fact

`OrderResponse` carries `RefundAmount` and `RefundedAt` but **not** `WalletApplied`,
`LoyaltyApplied`, `LoyaltyPointsSpent`, `WalletRefunded` or `LoyaltyRefunded`. All
five exist on the `Order` model (`models/order.go:162,171,172,177,178`) and are
dropped at the DTO boundary — so the client *cannot* render the credit today.

This is the same dropped-field shape as #901 and #912, inverted: there the client
type omitted fields the server sent; here the response struct omits fields the
model holds. **Unlike #912, this one does need an API change.**

## Design

### 1. API — a computed `funding` object on `OrderResponse`

```json
"funding": {
  "applied":  { "wallet": 0, "loyalty": 2.50, "loyaltyPoints": 320 },
  "refunded": { "gateway": 324.14, "gatewayDestination": "original",
                "wallet": 0, "loyalty": 2.50 }
}
```

Keyed by **source rail**, plus the gateway slice's actual **destination**.

**Why server-computed rather than five raw fields.** The client would otherwise
derive the gateway figure as `refundAmount − walletRefunded − loyaltyRefunded` —
money arithmetic in the client is the exact class of thing that produced the
₹327/₹326.64 mismatch being fixed here. The server computes the gateway slice as
the **remainder**, mirroring `SplitRefundByFunding`'s own "card takes the
remainder" invariant, which is what makes `funding.refunded.gateway` equal the
gateway's own figure *by construction* rather than by coincidence.

**Why `gatewayDestination` is not optional.** `models/cancellation_request.go:51`
defines `RefundDestination` as `"wallet"` (instant) or `"original"` (gateway) —
**the customer picks**. A hardcoded "To your card" label is therefore wrong for
anyone who chose instant wallet. The server states where the slice went; the
client only chooses wording.

`funding` is omitted entirely when no credit was applied and nothing was refunded.
The common case renders exactly as today, with no empty UI.

### 2. Order detail rendering

```
Applied at checkout
  Loyalty points (320 pts)      -₹2.50
Total                          ₹326.64

Refunded
  To your card                 -₹324.14    ← "To your wallet" when destination=wallet
  To your wallet                -₹2.50
    incl. ₹2.50 from loyalty points, returned as wallet credit
```

Loyalty is **named explicitly**, including that points return as wallet credit and
not as points (`refund_funding_split.go:19` — owner decision). This makes an
unfavourable policy visible at the moment of refund, which is the point: it
pre-empts the "where are my points?" support ticket rather than deferring it.

Also: `toFixed(0)` → the app's currency formatter, and the summary line stops
claiming the full amount reached the card.

### 3. Wallet freshness

Invalidate `['wallet']` on order cancellation, refund, and checkout success. The
other three money-moving paths already do it; these were simply missed.

### 4. Edge cases

- **Full-wallet order** (`CapturePaise == 0`) — render no gateway line at all
  rather than "₹0.00".
- **Partial refunds** — each rail is already capped at what it has not returned,
  so repeated partials still sum to `refundAmount` and never over-return.
- **Zero-credit, never refunded** — `funding` omitted entirely; screen unchanged
  from today.
- **Zero-credit but refunded** — `funding` IS present, with `gateway ==
  refundAmount` and both credit rails zero. The refund block renders a single
  "To your card" line. This is deliberate: it is the case where the app's number
  and the gateway's number agree, and showing it makes that agreement checkable
  rather than assumed.
- **Gateway destination = wallet** — the gateway slice is labelled "To your
  wallet", and may merge visually with the wallet-rail line; keep them as separate
  rows so the amounts still reconcile against the gateway dashboard.

### 5. Testing

**Go.** Table test asserting per-rail reconciliation: `applied − refunded` per rail
is never negative, the parts sum exactly to `refundAmount`, and the gateway slice
equals the captured amount for a full refund. Cases: zero-credit, wallet-only,
loyalty-only, both, full-wallet (`gateway == 0`), and a two-partial-refund sequence.

**Client.** `packages/mobile-shared` vitest is the only working mobile bed, so the
line-composition logic goes there as a pure function taking `funding` and returning
rendered rows — the same shape as `subscriptionRowSummary` (#907). The screen then
holds no money logic. **No new dependency and no lockfile change** (#897).

## Out of scope

- Checkout and receipt surfaces. The same confusion starts at checkout, but this
  change is scoped to order detail; the shared pure function is deliberately
  written so those can adopt it without rework.
- The refund policy itself, and the points-return-as-wallet decision.
- `#396` integer-paise migration. This design reads existing `float64` fields and
  neither helps nor hinders it.

## Verification

- `go build ./...`, `go vet ./...`, `gofmt -l .` unchanged against a pre-change
  baseline (currently 21 pre-existing files), `go test ./...` green.
- `tsc --noEmit -p apps/mobile-customer` measured against a stashed baseline.
  Note a fresh worktree reports 0 rather than 1 because `.expo/types/router.d.ts`
  is gitignored — trust CI's own tsc job.
- `pnpm --filter @homechef/mobile-shared test` — baseline 126 pass / 4 pre-existing
  fail (all 4 in `auth-screens.test.ts`; `resolve-auth-error.test.ts` fails at
  collection and contributes 0 failed tests).
- Device check: cancel an order that used credit, confirm the split lines match the
  Cashfree dashboard and the wallet chip updates without a restart.
