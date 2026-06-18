# HomeChef — Money Flow, Holds, Payouts, Cancellations & Refunds

How money is collected, **frozen/held**, **passed through**, **paid to the vendor
(chef) and rider**, and **refunded** across every ordering channel. This is the
single reference for the payment lifecycle and the cancellation/refund policy.

> Currency is INR; amounts move in **paise** at the gateway (`services.ToPaise`/
> `FromPaise`). The business timezone is **IST** for all day/cutoff math.

---

## 1. Rails & accounts

- **Gateway:** Razorpay. Customer charges create a Razorpay **Order**; the
  customer pays via the in-app WebView checkout (`apps/mobile-customer`) and the
  server **verifies** capture (`payment.captured` webhook is the authoritative
  backstop).
- **Razorpay Route (split payments):** each chef and rider has a **linked
  account** (`ChefProfile.RazorpayAccountID`, `DeliveryPartner.RazorpayAccountID`).
  Money is routed to them via **transfers**.
  - **Payment-linked transfer** — attached to a customer charge (`OrderRequest.Transfers`);
    executes on capture; **auto-reversed** if the payment is refunded.
  - **Direct transfer** — `CreateTransfer` from the **platform balance** to a
    linked account, independent of any single payment (used when funds were
    pooled from multiple charges, e.g. group orders / escrow).
- **Hold / freeze:** a transfer created with `OnHold: true` is **frozen** at
  Razorpay — the chef/rider can't withdraw it yet. It is **released** on delivery
  via `ReleaseTransfer` (`on_hold:false`) and **clawed back** on a pre-delivery
  refund via `ReverseTransfer`. (`services/razorpay.go`.)
- **Wallet (store credit):** `Wallet` + `WalletTxn` ledger. Refunds prefer the
  wallet (`CreditWallet`, idempotent on a per-event key); store credit can be
  applied at checkout (`WALLET_CHECKOUT_ENABLED`).
- **Platform revenue:** the service fee + delivery margin stay with the platform;
  **tax (GST)** is collected on the taxable base and remitted by the platform.
  **Tips are never taxed and never commissioned** — 100% pass-through.

---

## 2. Per-channel money lifecycle

### 2.1 Regular à-la-carte order (`handlers/orders.go`, `handlers/payment.go`)
1. **Charge:** customer pays `order.Total` (subtotal + delivery + service + tax + tip − discount − wallet).
2. **Split (on capture):** Route transfers — **chef** gets `subtotal + tax + chefTip` **`OnHold:true`**; the rider's `deliveryFee + driverTip` is settled through the delivery/earnings flow (no rider is assigned at checkout).
3. **Fulfilment:** chef accepts → prepares → ready → dispatch → delivered.
4. **Cancel/refund:** see §3. Refunding the payment **auto-reverses** the chef's payment-linked transfer.

> ⚠️ **Known platform gap:** regular-order chef transfers are created `OnHold:true`
> but there is **no automatic release on delivery** for this channel today — they
> rely on Razorpay's hold settings / manual release. The per-day release mechanism
> built for escrow/group (§2.3/§2.4) is the template to extend here. (Tracked.)

### 2.2 Tips — chefs / riders (`handlers/tips.go`, #45)
- A **separate** Razorpay charge, post-delivery. Route-split **100%** to the chef
  and/or rider with **`OnHold:false`** (delivery already happened → settle
  immediately). No tax, no commission. Idempotent via the tip's `razorpay_order_id`
  + the `payment.captured` webhook. There is no hold and no refund policy (tips are
  voluntary and final).

### 2.3 Tiffin meal-plan / escrow (`services/meal_plan_escrow.go`, #194 — flag `MEAL_PLAN_ESCROW_ENABLED`, default OFF)
1. **Advance capture:** at booking, **one** charge for the full requested total → platform balance (`CreateMealPlanAdvanceOrder` + `VerifyMealPlanAdvance`, stamps `EscrowPaymentID`).
2. **Hold:** on confirm, **one on-hold Route transfer per accepted day** to the chef (`HoldChefPayouts`, stamps `PayoutTransferID`).
3. **Release:** per **delivered day** (`ReleaseDayPayout`, hooked at the delivery-delivered transition for own-fleet + 3PL).
4. **Refund:** declined / expired / rejected / **customer-skipped** days → **wallet** (`RefundDay`, idempotent key `mealplan-refund:<dayID>`; reverses the held transfer first). Full refund on expiry/reject.

### 2.4 Group / office orders (`handlers/group_order.go`, `services/group_order_payout.go`, #46 — flag `GROUP_ORDERS_ENABLED`, default OFF)
1. **Per-participant charge:** each participant pays **their split share** (their items + pro-rata delivery/service/tax) into the **platform balance** (no per-charge transfer). Host-pays mode: only the host pays the full total.
2. **Consolidate:** when everyone required has paid, **one** consolidated `Order` is created (single delivery to one drop) and fed to the normal chef + dispatch pipeline.
3. **Hold:** the chef's slice (`subtotal + tax`) is held as **one** on-hold Route transfer at placement (`HoldGroupChefPayout`, stamps `GroupOrder.PayoutTransferID`).
4. **Release:** on delivery (`ReleaseGroupChefPayout`, hooked at delivery-delivered, own-fleet + 3PL).
5. **Refund (host cancels):** the held chef transfer is **reversed** and **every paid participant is refunded to wallet** (`RefundGroupParticipant`, idempotent key `grouporder-refund:<participantId>`).

### 2.5 Capacity caps (#48) — *not money, but interacts with refunds*
Per-dish daily caps reserve a **slot** at order time (atomic, oversell-safe). The
slot is **released** whenever the order is cancelled/rejected/refunded **before
delivery** (so the dish becomes orderable again) — see §3. Caps reset per IST day.

---

## 3. Cancellation & refund policy matrix

| Who / when | Allowed states | Money | Capacity (#48) |
|---|---|---|---|
| **Customer cancels** (`orders.go CancelOrder`) | `pending`, `accepted` | Refund per provider (wallet or source); 3PL delivery cancelled | **Released** |
| **Chef cancels whole order** (`chef_order_cancel.go CancelOrder`) | mid-prep (`cancellableStatuses`) | Full Razorpay refund; idempotent on `refund_id` | **Released** |
| **Chef cancels one line** (`CancelOrderItem`) | mid-prep | Line refund (subtotal + pro-rata tax); order totals recomputed | **Released** (that line) |
| **Chef status → cancelled** (`chefs.go UpdateOrderStatus`) | not terminal | (refund handled by the cancel/refund paths) | **Released** (first transition) |
| **Admin/chef refund** (`payment.go InitiateRefund`) | `PaymentCompleted` | Wallet or gateway; partial or full | **Released** only on a **full, pre-delivery** refund |
| **Chef goodwill refund** (`chef_order_cancel.go RefundOrder`) | `delivered` | Partial/full goodwill | **Not** released (food was made) |
| **Delivery cancelled** (`delivery.go`) | — | Order returns to `ready` for re-dispatch | **Not** released (still being delivered) |
| **Meal-plan day** skip/decline/expire/reject (#194) | pre-delivery | Wallet refund (idempotent per day) | n/a (separate inventory) |
| **Group order** host-cancel (#46) | pre-delivery | Reverse held payout + wallet refund each paid participant | n/a |

**Refund destination:** default is **store-credit wallet** (instant, idempotent);
gateway-to-source refunds are used where required. Refunding a payment-linked
charge **auto-reverses** the chef's transfer; pooled/escrow payouts are reversed
explicitly (`ReverseTransfer`).

---

## 4. Idempotency & reconciliation

- **Charges/captures:** guarded by `WHERE … payment_status <> completed` (webhook
  + verify converge once); the Razorpay **receipt** (order number / `GRP-…` /
  `TIP-…`) anchors gateway-side dedup.
- **Wallet refunds:** unique idempotency key per event (`refund:<orderID>`,
  `mealplan-refund:<dayID>`, `grouporder-refund:<participantId>`) — a retry is a
  no-op.
- **Transfers:** release/reverse are DB-guarded on the stored transfer id.
- **Reconciliation cron** sweeps stuck/holds and drift.

---

## 5. Feature flags & go-live posture

| Flag | Default | Gates | Before enabling in prod |
|---|---|---|---|
| `MEAL_PLAN_ESCROW_ENABLED` | OFF | Tiffin advance capture / hold / release / refund (#194) | Sandbox-verify capture → hold → release → refund; wire the customer advance-checkout |
| `GROUP_ORDERS_ENABLED` | OFF | Group/office orders end-to-end (#46) | Sandbox-verify the multi-payer charge → consolidate → hold → release → refund |
| `WALLET_CHECKOUT_ENABLED` | OFF | Applying store credit at checkout (#141) | Verify direct-transfer top-ups in sandbox |

Tips (#45) and capacity (#48) are **not** flagged — tips are a standard charge
(reuses the proven checkout) and caps move no money.

---

## 6. Open / tracked gaps (see GitHub issues)

1. **Regular-order held transfers are never auto-released on delivery** — pre-existing platform gap; extend the escrow/group release hook to this channel.
2. **Group + escrow money flows need Razorpay-sandbox sign-off** before flipping their flags.
3. **Group & meal-plan orders bypass the à-la-carte daily cap** — capacity is enforced only on the regular `CreateOrder` path + an add-time sold-out guard for group carts; full reserve-at-lock/confirm with its own release is a follow-up.
4. **Escrow vs. UPI-Autopay model decision** for tiffin (#1/#2) is still open.
