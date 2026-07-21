# #8 — Full order-lifecycle E2E (on-device runbook)

One scripted run on a real device (or the booted sim) + a real chef account, covering the money/fulfillment paths that need live infra (Razorpay, NATS, FCM push, invoice PDF, email). The **pure money invariants** (per-line refund → recompute → whole cancel → earnings) are already automated in `apps/api/handlers/order_lifecycle_money_test.go` — this runbook covers the rest.

Prereqs: Razorpay in **test mode** is fine for the refund timing checks (live is #25); a chef account that self-delivers or offers pickup; customer `demo@test.com` / `Test@123`; API deployed with the fulfillment branches (#369) so refunds/cancel endpoints exist.

## Steps

1. **Browse → order → push → chef accepts**
   - Customer: open a chef, add 2 distinct items, checkout, pay (test card/UPI).
   - [ ] Chef device gets a **new-order push** (FCM). If not, check #14 APNs + the device FCM token.
   - Chef: **Accept** the order. [ ] Customer order screen flips to *Accepted/Preparing*.

2. **Per-line "can't fulfill" → partial refund in-flight**
   - Chef: on the mid-prep order, mark **one line** as can't-fulfill (`POST /chef/orders/:id/items/:itemId/cancel`).
   - [ ] Refund initiated at the gateway in **<10s** (Razorpay dashboard → Refunds shows the line's amount incl. its tax share).
   - [ ] Order subtotal/tax/total **recompute** (customer + chef both show the reduced total); the **other item continues**.
   - Expected math for a ₹600+₹400 order (tax 5%): cancelling the ₹400 line refunds **₹420** and the order drops to subtotal ₹600 / tax ₹30.

3. **Whole-order cancel → full refund + NATS event**
   - Chef: cancel the whole order with a reason (`POST /chef/orders/:id/cancel`).
   - [ ] Remaining balance refunded in **<10s**.
   - [ ] `order.cancelled` **NATS event** published (check the subscriber logs / audit trail).
   - [ ] Customer sees *Cancelled* + refund summary; no raw error codes.

4. **Post-delivery goodwill refund → earnings**
   - On a **separate delivered** order, chef issues a goodwill refund (`POST /chef/orders/:id/refund`, partial amount).
   - [ ] Refund processes; the amount appears in the chef's **refund history**.
   - [ ] Chef **earnings / weekly statement** reflect the reduced net payout for that order.

5. **Invoice PDF + auto-email**
   - [ ] Invoice **PDF downloads** for a completed order (`GET /chef/orders/:id/invoice`), GSTIN in the expected format (see #19).
   - [ ] Customer receives the invoice **email within ~5 min** of delivery.

6. **Earnings reconciliation**
   - [ ] Refund history + weekly-statement entries are correct and tie out to steps 2–4.

## Driving the sim (from the sim-QA notes)
- iPhone 17 Pro sim `9F9E4972-E434-47CD-B7DA-8BE294D41B97`; Metro on **8082**; customer app built from source against prod.
- `idb` at `/Users/Mahesh.Sangawar/Library/Python/3.9/bin/idb`: `idb ui describe-all` (coords), `idb ui tap X Y`, `idb ui text "…"` (⚠️ flaky with `@` — split + screenshot-verify), `idb ui key 42` (backspace). Screenshot: `xcrun simctl io <UDID> screenshot f.png`.
- The chef side is easiest on the **vendor app** (build it on the sim too) or via direct API calls with a chef bearer token.

## Coverage
- **Automated (CI):** line-refund math, tax-share, order recompute, all-cancelled fees/tip retention, refund conservation (Σ line refunds = subtotal + tax), earnings-reflect-refund → `handlers/order_lifecycle_money_test.go` + `handlers/chef_order_totals_test.go` + `handlers/chef_earnings_test.go`.
- **Manual (this runbook):** gateway refund timing, NATS event, FCM push, invoice PDF, auto-email, weekly-statement reconciliation.
