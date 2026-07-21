# #392 — Cancellation paths that return no money (LIVE P0)

**LIVE in prod, NOT escrow-flag-gated.** A paid order cancelled via two paths refunds ₹0.

## Confirmed live exploit (the P0 core)
The **vendor app's reject action** (`apps/mobile-vendor/hooks/useVendorOrders.ts:138`, `useOrderAction`,
`action: 'accepted'|'rejected'`) calls `PUT /chef/orders/:id/status` → `ChefHandler.UpdateOrderStatus`
(`handlers/chefs.go:958`) → on cancel/reject only `services.SignalOrderCancelled` (`chefs.go:1153`) →
`signalOrderSaga` **early-returns when `ORDER_SAGA_ENABLED=false` (default)** → NO refund, NO
`refund_amount`. So: customer pays → chef taps "Reject" → customer gets nothing.

Secondary (latent — customer app uses the correct `/cancel-request` arbitration flow, but the route is
live): **customer `POST /orders/:id/cancel`** (`OrderHandler.CancelOrder`, `handlers/orders.go:802`) flips
status + releases capacity/slots/3PL but issues no refund for a paid `accepted` order.

Accounting bug: chef dedicated cancel (`handlers/chef_order_cancel.go:136`) stamps
`refund_amount = order.Total` unconditionally (not additive; also stamps Total even when the gateway
refund was skipped) → drifts the ledger, trips `reconciliation.go` `refund_mismatch`.

## Existing machinery (reuse, don't reinvent)
- `InitiateRefund` (`handlers/payment.go:674`) is the gold-standard full-refund composition: **atomic
  claim** `completed→refunded` conditional UPDATE (double-refund guard, gateway calls have no idempotency
  key), provider switch (wallet/stripe/razorpay), additive `refund_amount = RefundAmount + delta`,
  `WithholdOrReverseOrderHoldForRefund` cross-guard.
- `ExecuteCancellationRefund` (`services/cancellation_execute.go:23`) — crash-safe tx + sweep backstop,
  but requires a `CancellationRequest` row (one-per-order unique index → collision risk if reused here).
- Chef agreement (`apps/mobile-vendor/app/chef-agreement.tsx:52`): **"If you cancel an order after
  accepting it, the customer receives a full refund."** → chef cancel/reject = FULL refund.

## PLAN-CHECK REVISIONS (2026-07-05) — required before implementation
Plan-check found a **cross-path double-refund blocker**: the `payment_status` claim is blind to the
sibling paths (`ExecuteCancellationRefund` / `chef_order_cancel` never write `payment_status`), so
concurrent `/cancel-request`-vs-reject and the `/cancel`→`ConfirmCancellation` sequence would refund
twice. Required corrections folded in below:
1. **Unify on a shared `refunded_at IS NULL` atomic claim.** EVERY full-refund path claims
   `UPDATE orders SET refunded_at=now WHERE id=? AND refunded_at IS NULL` (RowsAffected==1 ⇒ proceed to
   gateway; 0 ⇒ already refunded, skip). Add this claim to `ExecuteCancellationRefund`
   (`cancellation_execute.go`, before its gateway call at :36) and `chef_order_cancel.go CancelOrder`
   (before :105) as well as the new service. InitiateRefund is already excluded (its `payment_status`
   claim + `remaining` check; the new service also claims `payment_status` so they mutually exclude).
2. **Stripe API is `services.GetStripe().CreateRefund(&StripeRefundRequest{PaymentIntent, Amount:
   ToMinor(amt, currency), ReverseTransfer:true, RefundApplicationFee:true, ...})`** — currency via
   `order.Currency`/`CurrencyForCountry(order.Chef.PayoutCountry)`, mirroring payment.go:917-933. There
   is NO `CreateStripeRefund`.
3. **Partial `WalletApplied` cap**: gateway refund capped to `capture = Total − WalletApplied`; the
   wallet slice re-credited separately with key `refund-wallet:<orderID>` (mirror payment.go:850-863).
4. **refund_amount over-stamp fix** = OMIT the `refund_amount` write when `amountPaise<=0` (the
   `RefundAmount+refundable` rephrase is an algebraic no-op). Leave the prior value untouched on skip.
5. **Reject-status clobber**: treat `rejected` (and `cancelled`) as terminal — the service must NOT
   overwrite `rejected`→`cancelled`. Only set `status=cancelled` when the order isn't already terminal.
6. Observations (follow-ups, note in PR): pre-existing per-line→full-cancel under-refund
   (`Total−RefundAmount≈0` strands remaining live items); no sweep backstop (accept, rely on
   reconciliation refund_mismatch); Razorpay CreateRefund has no idempotency key (ambiguous-gateway-
   failure retry risk — defer, log reverted claims).

## Design — one shared service, wired into the two broken paths

New `services.RefundOrderForCancellation(order *models.Order, initiatedBy, reason string) error` (new
file `services/cancellation_order_refund.go`), modeled on `InitiateRefund`'s atomic-claim pattern (NOT
CancellationRequest-based — avoids the unique-index collision):
1. `PaymentStatus != PaymentCompleted` → return nil (never charged; a pending/unpaid reject has nothing
   to refund).
2. Already refunded (`RefundedAt != nil` || `PaymentStatus == PaymentRefunded`) → return nil (idempotent).
3. `refundable = Total − RefundAmount`; `amountPaise = int(roundPaise(refundable))`.
4. **Atomic claim**: conditional UPDATE `payment_status: completed→refunded WHERE payment_status='completed'`;
   `RowsAffected==0` → someone else claimed → return nil (no double refund). `revertClaim` on gateway error.
5. Gateway refund by provider: razorpay `CreateRefund(RazorpayPaymentID, amountPaise)`; stripe
   `CreateStripeRefund(StripePaymentIntentID, …)`; wallet/`WalletApplied` → `CreditWallet` key
   `refund:<orderID>`. If `amountPaise<=0` skip the gateway (just flip state).
6. Stamp additive: `refund_amount = RefundAmount + refundable`, `refund_id`, `refunded_at`,
   `refund_reason`, `refund_initiated_by`, `status = cancelled` (only if not already terminal).
7. Post-commit best-effort `WithholdOrReverseOrderHoldForRefund(db, order.ID, reason)`.

### Wire-in
- **chefs.go `UpdateOrderStatus`** cancel/reject branch: call `RefundOrderForCancellation(order,
  "chef", "chef "+status)` synchronously (in addition to / instead of the saga signal when saga is off).
  Keep `SignalOrderCancelled` when saga IS enabled (avoid double-refund: the service is idempotent via
  the atomic claim, so even if both fire, only one refunds — but gate to avoid noise).
- **orders.go `CancelOrder`**: after the existing cancel tx, call `RefundOrderForCancellation(order,
  "customer", "customer cancellation")`. Latent path; makes it safe for any legacy/direct caller.
- **chef_order_cancel.go:136**: change `"refund_amount": order.Total` → `order.RefundAmount + refundable`
  and only stamp when `amountPaise>0` (or set it to the actual cumulative). Additive parity with the
  other paths. (Keep this path otherwise; it already refunds correctly.)

## Scope OUT (follow-ups, note in PR)
- Cancellation-fee/tier POLICY on these direct paths (the tiered `ComputeCancellationRefund` is the
  vendor-arbitration `/cancel-request` flow; direct chef/customer cancels are full-refund by the chef
  agreement). #392's "fee policy via PlatformSettings" is a separate product decision.
- Stripe parity for the chef DEDICATED cancel path (`chef_order_cancel.go` still 422s Stripe) — the new
  shared service DOES handle Stripe, so the status path gets Stripe refunds; unifying the dedicated path
  is a follow-up.
- Razorpay refund idempotency keys on chef_order_cancel.go (the atomic claim in the new service covers
  the status/customer paths).

## Tests (RED-first, sqlite harness like payout_crossguard_refund_test.go / cancellation tests)
- `RefundOrderForCancellation`: paid razorpay order → refund_amount=Total, status refunded/cancelled,
  hold withheld; already-refunded → no-op; unpaid → no-op; partial-prior-refund → additive to Total;
  wallet/`WalletApplied` path; double-call idempotent (atomic claim → one refund).
- Handler-level: chef `UpdateOrderStatus`→rejected on a paid order refunds (saga off); customer
  `CancelOrder` on a paid accepted order refunds.
- chef_order_cancel over-stamp fixed: prior per-line refund then full cancel → refund_amount == Total,
  not Total+prior.

## Money-path: plan-check this before implementing (esp. the atomic-claim vs CancellationRequest choice
## and the chef-reject full-refund semantics). Then RED-first → independent verify.
