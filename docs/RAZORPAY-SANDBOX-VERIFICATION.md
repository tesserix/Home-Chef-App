# Razorpay Sandbox Verification Runbook (#218)

The money flows below are **built but flag-gated OFF**. Each moves live settlement
(captures, held Route transfers, releases, reversals, refunds), so before flipping
a flag in production, verify it end-to-end against the **Razorpay sandbox / test
mode** using Route **test linked accounts**. This is a manual sign-off — work
through the relevant section, tick the boxes, then enable the flag.

> Prereqs: Razorpay **Test Mode** keys in the API env (`RAZORPAY_KEY_ID/SECRET`),
> at least one **test linked account** (`acc_…`) on a chef and (for delivery
> release) a rider, the webhook pointed at the test endpoint with
> `RAZORPAY_WEBHOOK_SECRET`, and a test customer. Use Razorpay test cards/UPI.

---

## A. Regular-order payout release — `ORDER_PAYOUT_AUTO_RELEASE_ENABLED` (#217)
- [ ] Place + pay a normal order (test card) → confirm capture; chef transfer exists with `on_hold: true` (Dashboard → Transfers, or `GET /orders/{id}/transfers`).
- [ ] Mark the order delivered (own-fleet status update **and**, separately, a 3PL provider webhook) → confirm the held transfer flips to **released** (`on_hold:false`) and settles to the chef test account.
- [ ] Deliver an order with **no** held transfer (e.g. wallet-only) → confirm no error / no spurious release.
- [ ] Refund a **pre-delivery** order → confirm the payment-linked transfer **auto-reverses** (chef not paid).
- [ ] Idempotency: re-fire the delivered webhook → no double-release.

## B. Tiffin meal-plan escrow — `MEAL_PLAN_ESCROW_ENABLED` (#194)
- [ ] Book a plan → advance Razorpay order created for the full total; pay → `EscrowPaymentID` stamped, funds in platform balance.
- [ ] Chef accept-all → one on-hold transfer per accepted day to the chef; `PayoutTransferID` stamped per day.
- [ ] Cherry-pick + customer approve → declined days refunded to **wallet** (idempotent `mealplan-refund:<dayID>`), accepted days held.
- [ ] Deliver a day → that day's transfer **releases**; mark all days delivered → plan completes.
- [ ] Customer skip / expiry / reject → affected days refunded to wallet; held transfer reversed first.
- [ ] Idempotency: retry verify + delivered hooks → no double capture / release / refund.

## C. Group / office orders — `GROUP_ORDERS_ENABLED` (#46)
- [ ] Host creates → invite link works; 2+ participants join and add items.
- [ ] Host locks → each participant's share computed (pro-rata); capped dishes reserve capacity (a sold-out dish blocks the lock).
- [ ] Each participant pays their share (test card) → all into platform balance; **host-pays** mode: only host pays.
- [ ] All paid → ONE consolidated order created + single delivery; chef's slice held as one transfer.
- [ ] Deliver → chef transfer **releases**.
- [ ] Host cancels (pre-delivery) → held transfer **reversed** + each paid participant refunded to **wallet** (idempotent `grouporder-refund:<participantId>`); reserved capacity released.

## D. Wallet at checkout — `WALLET_CHECKOUT_ENABLED` (#141)
- [ ] Apply store credit at checkout → gateway charges `total − walletApplied`; full-wallet order skips the gateway and is marked paid.
- [ ] Direct-transfer top-ups settle the chef/driver shortfall the capture couldn't cover.

---

## Enabling
Once a section is fully ticked in test mode, set the flag in the prod env
(tesserix-k8s values / external-secret), deploy via ArgoCD, and **watch the first
few live orders** (Dashboard → Transfers/Refunds + the reconciliation cron). Roll
back by flipping the flag off — the gated code is a no-op when off.
