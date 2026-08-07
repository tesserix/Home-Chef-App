# HomeChef — Money Flow, Holds, Payouts, Cancellations & Refunds

How money is collected, **held**, **paid to the vendor (chef) and rider**, and
**refunded** across every ordering channel. This is the single reference for the
payment lifecycle and the cancellation/refund policy.

> Currency is INR; amounts move in **paise** at the gateway (`services.ToPaise`/
> `FromPaise`). The business timezone is **IST** for all day/cutoff math.

---

## 1. Rails & accounts

- **Gateway: Cashfree Payments**, the only gateway a new charge can be created
  on (`services/cashfree.go`, `GetCashfreeFor(mode)`). The customer pays a
  Cashfree **order**; the server verifies the capture and the webhook is the
  authoritative backstop. Stripe Connect exists only for chefs paid outside
  India (`PUT /chef/payment-provider` accepts `cashfree | stripe`).
- **The whole capture lands in the platform merchant account.** There is no
  per-charge routing to the chef. The chef's share is allocated *afterwards*,
  by one of two rails:
  - **Easy Split at release** — `ReleaseOrderSplit`
    (`services/easy_split_release.go`) calls Cashfree's split-after-payment API
    inside the account's **order split delay** (`easy_split_delay_hours`
    setting, Cashfree's own default 24h). One split line, to the chef's
    deterministic vendor id (`EasySplitVendorIDFor` → `hc_<chefID>`).
  - **The payout rail** — the weekly statement → payout batch → Cashfree
    Payouts (`services/payout_disbursement.go`, `payouts/rail.go`). This is the
    fallback for every order the split refuses, and was the only rail before
    Easy Split existed.
- **Why split at release and not at capture (ADR-0003, #1091):** splitting on
  capture allocated the chef's share before the maturation window and before
  `BlockRefundOpen` / `BlockRecoveryBalance` / `BlockNewChefRamp` /
  `BlockAboveReviewThreshold` had said anything — a split order had no payout
  left to release, so none of the governor's blocks ever ran.
- **Hold / freeze** is ours, not the gateway's: a `payout_hold_status` column on
  the order / meal-plan day / group order (`models/payout_hold.go`), driven by
  guarded conditional UPDATEs in `services/payout_hold.go`. The states are
  `none → awaiting_customer_confirmation → release_eligible → released |
  withheld | reversed | disputed`. Money moves only in the release seam
  (`releaseMoney`, `services/payout_release.go`), *after* the state flip has
  committed.
- **Wallet (store credit):** `Wallet` + `WalletTxn` ledger. Refunds prefer the
  wallet (`CreditWallet`, idempotent on a per-event key); store credit can be
  applied at checkout (`WALLET_CHECKOUT_ENABLED`, now default ON). Credit the
  gateway never saw is excluded from every split basis.
- **Platform revenue:** commission on the food subtotal, the service fee and the
  delivery margin stay with the platform; **tax (GST)** is collected on the
  taxable base and remitted by the platform. **Tips are never taxed and never
  commissioned** — 100% pass-through.

### 1.1 The release path, end to end

1. **Delivered.** The delivered transition parks the hold:
   `SetOrderHoldAwaitingConfirmation` (own-fleet `handlers/delivery.go`, chef
   self-delivery `handlers/chefs.go`, 3PL `services/shadowfax_webhook.go` /
   `services/provider.go`, Temporal `services/temporal_order.go`). Delivered no
   longer means paid.
2. **Confirmed.** The customer confirms, or the confirm window lapses and the
   auto-confirm sweep advances it (`services/payout_auto_confirm_cron.go`,
   `GetCustomerConfirmWindowHours`, default 24h). An order with an open issue
   goes to `disputed` instead. → `release_eligible`.
3. **Matured.** The 15-minute release sweep
   (`services/payout_release_cron.go`) picks up `release_eligible` holds older
   than `payout.maturation_minutes` (default 2h) and asks the governor
   (`payouts/governor.go`) whether they may go. Blocked orders stay held and
   surface in the admin queue with every reason.
4. **Released.** `ReleaseHold` flips the status, then `releaseMoney` calls
   `ReleaseOrderSplit`. Only the **order** aggregate moves money here; a
   meal-plan day and a group order are paid on the statement path, so releasing
   their hold is a state change alone.
5. **Split or statement.** `ReleaseOrderSplit` returns true when Cashfree
   accepted the split; it stamps `orders.gateway_split_paise`, and that column
   is what keeps the order **off** the weekly statement
   (`services/statement.go`, `statement_catchup.go`). Return false and the order
   settles on the statement path instead.

`BuildOrderSplitWithReason` computes the share as
`ToPaise(ChefNetPayoutFor(order)) − platform_fee_flat_minor`, capped at the
capture. It refuses — and the reason is written to the audit log
(`order.payout.easy_split_skipped`, #1084) — when:

| Skip reason | Meaning |
|---|---|
| `not_enabled_for_chef` | `easy_split_enabled`, or the chef's own `easy_split_mode` override, says no |
| `credit_funded` | wallet/loyalty part-funded the order, so the capture no longer covers the share |
| `vendor_not_active` | the chef has no `ACTIVE` Cashfree vendor registration |
| `fssai_expired` | licence lapsed → payout withheld, money stays at the platform |
| `platform_fee_unreadable` | the fee setting will not parse — refuse rather than guess |
| `share_below_fee` | the computed share rounds to nothing |

`EasySplitWindowFits` additionally asserts that the maturation window fits
inside the split delay; if it does not, every order matures after Cashfree has
already settled and the rail degrades to the statement path — logged, not
silent.

### 1.2 The payout rail (the fallback, and the only rail for days/groups)

`WeeklyStatement` (Mon–Sun IST, one immutable row per chef per week, unique on
`(chef_id, week_start)`) → `PrepareStatementBatch` → `ExecuteBatch` →
`payouts.Rail.Disburse`. The batch is committed `executing` **before** the rail
is called, because a disbursement can fail without telling you whether money
moved: on `ErrRailAmbiguous` the caller must never retry `Disburse`, only ask
`GetPayoutByReference` with the same idempotency key. `EnsureBeneficiary` is
idempotent on a deterministic beneficiary id. Auto-disbursement is gated by
`payout_auto_disburse_enabled` and capped by `payout_auto_disburse_max_minor`,
per chef via `ChefAutoDisburseEnabled`.

Recovery (what the chef owes the platform) is netted off the gross by
`ApplyRecoveryDeduction` (`services/payout_recovery.go`) on this rail.

---

## 2. Per-channel money lifecycle

### 2.1 Regular à-la-carte order (`handlers/orders.go`, `handlers/payment.go`)
1. **Charge:** the customer pays `order.Total` (subtotal + delivery + service +
   tax + tip − discount − wallet − loyalty) to the **platform** merchant
   account. The order inherits the chef's current gateway at creation so verify
   and refund later read the same provider.
2. **No split at capture.** The chef's and rider's shares are allocated later.
3. **Fulfilment:** chef accepts → prepares → ready → dispatch → delivered, which
   parks the payout hold (§1.1).
4. **Payout:** confirm → maturation → governor → `ReleaseOrderSplit`, else the
   weekly statement.
5. **Cancel/refund:** see §3. A refund on a split order carries explicit
   `refund_splits` so the chef's share is debited back (§3.1).

The release seam is gated by **`ORDER_PAYOUT_AUTO_RELEASE_ENABLED` (default
OFF)** — `payoutMovementEnabled()`, `services/order_payout.go`. With it off the
hold state machine still runs in full; only the money seam no-ops.

### 2.2 Tips — chefs / riders (`handlers/tips.go`, #45)
A **separate Cashfree charge**, post-delivery, and the one place the split
**does** happen at capture — delivery has already happened, so there is nothing
left to hold. One gateway order carries one split line for 100% of the tip to
the chef's vendor: chef and rider are the same party today, and a second charge
would pay the gateway fee twice to reach the same account. The two legs stay
independently traceable on our side (`chef_amount` / `rider_amount`), which is
what the two tip-received notifications are raised from. No tax, no commission,
no hold, no refund policy — tips are voluntary and final. Idempotent on the
deterministic `tip-<tipID>` gateway order id plus the webhook.

### 2.3 Tiffin meal-plan (`services/meal_plan_escrow.go`, #194 — flag `MEAL_PLAN_ESCROW_ENABLED`, default OFF)
1. **Advance capture:** at booking, **one** charge for the full requested total
   to the platform (`CreateMealPlanAdvanceOrder` + `VerifyMealPlanAdvance`,
   stamps `EscrowPaymentID`).
2. **Hold per day:** each accepted day carries its own `payout_hold_status`;
   the delivered transition parks it via `SetMealPlanDayHoldAwaitingConfirmation`
   (`services/meal_plan_fulfillment.go`).
3. **Release per day:** `payout_mealplan_release_cron.go` — the day becomes
   `release_eligible` on customer confirm or auto-confirm (~24h), then matures a
   further ~24h **measured from `customer_confirmed_at`, not from delivery**, so
   a day is released roughly two days after it was delivered. The chef is paid
   for it on the **statement** path; the release is a state change plus
   `payout_settled_at`.
4. **Refund:** declined / expired / rejected / customer-skipped days → wallet
   (`RefundDay`, idempotent key `mealplan-refund:<dayID>`), with the day's hold
   withheld or reversed by the cross-guard. Full refund on expiry/reject.

### 2.4 Group / office orders (`handlers/group_order.go`, `services/group_order_payout.go`, #46 — flag `GROUP_ORDERS_ENABLED`, default OFF)
1. **Per-participant charge:** each participant pays their split share (their
   items + pro-rata delivery/service/tax) into the platform balance. Host-pays
   mode: only the host pays the full total.
2. **Consolidate:** once everyone required has paid, **one** consolidated
   `Order` is created (single delivery to one drop) and fed to the normal chef +
   dispatch pipeline.
3. **Hold:** on delivery, `MarkGroupOrderDelivered` → `parkGroupOrderOnDelivery`
   → `SetGroupOrderHoldAwaitingConfirmation` parks **one** hold for the whole
   group. The amount it carries is `groupNetPayout` — gross (`subtotal + tax`)
   less commission on the subtotal and §194-O TDS on the gross — the same basis
   a regular order and a meal-plan day settle on.
4. **Release:** confirm → maturation → released; the chef is paid on the
   **statement** path.
5. **Refund (host cancels):** `ReverseGroupHoldForCancel` drives the hold to
   `reversed` (self-guarded on `status == cancelled`, so it never touches a
   delivered group), and every paid participant is refunded to wallet
   (`RefundGroupParticipant`, key `grouporder-refund:<participantId>`).

### 2.5 Capacity caps (#48) — *not money, but interacts with refunds*
Per-dish daily caps reserve a **slot** at order time (atomic, oversell-safe).
The slot is **released** whenever the order is cancelled/rejected/refunded
**before delivery**, so the dish becomes orderable again — see §3. Caps reset
per IST day.

---

## 3. Cancellation & refund policy matrix

| Who / when | Allowed states | Money | Capacity (#48) |
|---|---|---|---|
| **Customer cancels** (`orders.go CancelOrder`) | `pending`, `accepted` | Refund to wallet or source (§3.2); 3PL delivery cancelled | **Released** |
| **Chef cancels whole order** (`chef_order_cancel.go CancelOrder`) | mid-prep (`cancellableStatuses`) | Full refund; idempotent on `refund_id` | **Released** |
| **Chef cancels one line** (`CancelOrderItem`) | mid-prep | Line refund (subtotal + pro-rata tax); order totals recomputed | **Released** (that line) |
| **Chef status → cancelled** (`chefs.go UpdateOrderStatus`) | not terminal | (refund handled by the cancel/refund paths) | **Released** (first transition) |
| **Admin/chef refund** (`payment.go InitiateRefund`) | `PaymentCompleted` | Wallet or gateway; partial or full | **Released** only on a **full, pre-delivery** refund |
| **Chef goodwill refund** (`chef_order_cancel.go RefundOrder`) | `delivered` | Partial/full goodwill | **Not** released (food was made) |
| **Delivery cancelled** (`delivery.go`) | — | Order returns to `ready` for re-dispatch | **Not** released (still being delivered) |
| **Meal-plan day** skip/decline/expire/reject (#194) | pre-delivery | Wallet refund (idempotent per day) | n/a (separate inventory) |
| **Group order** host-cancel (#46) | pre-delivery | Reverse the group hold + wallet refund each paid participant | n/a |

Every refund path also runs the payout cross-guard
`WithholdOrReverseOrderHoldForRefund` (#457): `release_eligible` /
`awaiting_customer_confirmation` / `disputed` → `withheld`; `released` →
`reversed`; `none` / `withheld` / `reversed` → no-op. It fans out to any
meal-plan day or group order carrying its hold on the same order, so one refund
cannot leave the chef holding money the customer got back. The transitions are
conditional UPDATEs, so a double refund never double-reverses.

### 3.1 Refunding a split order (`services/easy_split_refund.go`)
Once the split has happened the chef's net share has left the platform, so a
refund the platform bears alone is money it never held. Cashfree debits vendors
proportionally *by default*, but that is an account-level setting that can be
turned off — so `BuildRefundSplits` states the vendor's share explicitly, which
"replaces the proportional default" and makes the outcome identical under either
setting. The basis is `gatewayCapturePaise` (total minus wallet and loyalty the
gateway never saw), and multi-leg rounding is **differenced against the running
cumulative total** rather than re-floored per leg, so N partial refunds sum to
exactly the vendor's share and the sub-paise remainder falls on the platform,
never the chef. Returns nil — meaning "no `refund_splits` field", the
full-capture behaviour — when the order never split, the chef has no vendor, or
the share is already fully reversed.

### 3.2 Refund destination
Default is the **store-credit wallet** (instant, idempotent). A gateway refund
to source is possible only when `Order.GatewayRefundable()` is true, i.e. the
order carries a `gateway_order_id` this platform can still act on
(`models/payment_provider.go`); everything else settles to wallet. Handlers
branch on `GatewayRefundable()` rather than on a provider name.

---

## 4. Idempotency & reconciliation

- **Charges/captures:** guarded by `WHERE … payment_status <> completed` (webhook
  and verify converge once); the deterministic gateway order id anchors
  gateway-side dedup.
- **Splits:** `ReleaseOrderSplit` passes the order's own id as the split
  idempotency key, returns early when `gateway_split_paise > 0`, and stamps that
  column with a conditional `WHERE COALESCE(gateway_split_paise,0) = 0`. A
  re-drive after a lost response is safe — Cashfree reports the repeat as
  already-processed.
- **Wallet refunds:** one idempotency key per event (`refund:<orderID>:<leg>`,
  `mealplan-refund:<dayID>`, `grouporder-refund:<participantId>`) — a retry is a
  no-op.
- **Hold transitions:** every one is a conditional UPDATE guarded on the current
  status, with `RowsAffected` deciding whether the follow-on seam and the NATS
  event fire.
- **Payout batches:** keyed by the batch's `IdempotencyKey`; `executing` is a
  state a batch cannot leave by being cancelled, only by consulting
  `GetPayoutByReference`.
- **Reconcile crons** re-drive unsettled releases and sweep drift
  (`payout_reconcile_cron.go`, `reconciliation_cron.go`,
  `order_payment_reconcile_cron.go` — the last of which skips any order whose
  provider is not `IsKnownProvider`, so a retired-gateway row is never asked
  about on a rail we no longer operate).

---

## 5. Feature flags & go-live posture

Environment flags (`config/config.go`):

| Flag | Default | Gates | Before enabling in prod |
|---|---|---|---|
| `WALLET_CHECKOUT_ENABLED` | **ON** | Applying store credit at checkout (#141) | — live |
| `LOYALTY_CHECKOUT_ENABLED` | **ON** | Applying loyalty points at checkout | — live |
| `MEAL_PLAN_ESCROW_ENABLED` | OFF | Tiffin advance capture / hold / release / refund (#194) | Sandbox-verify capture → hold → release → refund; wire the customer advance-checkout |
| `GROUP_ORDERS_ENABLED` | OFF | Group/office orders end-to-end (#46) | Sandbox-verify multi-payer charge → consolidate → hold → release → refund |
| `ORDER_PAYOUT_AUTO_RELEASE_ENABLED` | OFF | The money seam on a released order hold (#217) | Sandbox-verify delivered → confirm → mature → split |

Runtime settings (the `settings` table, changeable without a deploy):

| Setting | Default | Meaning |
|---|---|---|
| `easy_split_enabled` | off | The split rail, platform-wide; a chef's `easy_split_mode` overrides it either way |
| `easy_split_delay_hours` | 24 (Cashfree's own) | The account's agreed order-split delay — recorded here, never requested |
| `platform_fee_flat_minor` | 0 | Flat fee deducted from the chef's split share |
| `payout.customer_confirm_window_hours` | 24 | How long an `awaiting_customer_confirmation` hold waits before auto-confirm |
| `payout.maturation_minutes` | 120 | How long a `release_eligible` hold waits before the sweep releases it |
| `payout.review_above_paise` | 500000 | Order value above which a human looks; explicit `0` disables |
| `payout.new_chef_ramp_orders` | 3 | Delivered-order count below which a chef's payouts are held; explicit `0` disables |
| `payout_auto_disburse_enabled` | off | Auto-execution of statement payout batches |
| `payout_auto_disburse_max_minor` | — | Per-batch auto-disburse cap |

Sign-off runs against the **Cashfree sandbox** — Easy Split vendor registration,
split-after-payment, refund splits, and a Cashfree Payouts beneficiary +
disbursement — before any of these is flipped on.

Tips (#45) and capacity (#48) are **not** flagged — tips are a standard charge
(reusing the proven checkout) and caps move no money.

---

## 6. Open / tracked gaps (see GitHub issues)

1. **Cashfree-sandbox sign-off (#218)** for the escrow, group and order-release
   flows before their flags are flipped.
2. **Statement sunset for fully-split chefs (#1087)** — a chef settled entirely
   through Easy Split still has a statement generated (with the split orders
   excluded); collapsing that to nothing is not implemented.
3. **Recovery line on the weekly statement (#1092)** — `ApplyRecoveryDeduction`
   nets recovery off the payout, but the statement does not yet show it as its
   own line.
4. **Group orders consume the à-la-carte cap** (reserve at lock, release on
   cancel, #219). Meal-plan orders book weekly-menu cells (separate inventory) —
   per-cell caps remain a distinct follow-up.
5. **Escrow vs. UPI-Autopay model decision** for tiffin (#1/#2) is still open.
