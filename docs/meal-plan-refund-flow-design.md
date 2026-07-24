# Meal-plan / Group-order Refund Workflow — Design

**Status:** Draft for approval · **Scope:** meal-plan day-skip + whole-plan cancel, plus group-order refunds. On-demand à-la-carte orders are explicitly **out** of wallet refunds.

Grounded in the current code: `handlers/meal_plan.go` (SkipMealPlanDay, CancelMealPlan, admin approve/reject-skip), `services/meal_plan_escrow.go` (RefundDay/refundDayAmount/RefundUndeliveredDays, perDayGross/perDaySkipRefund), `services/payout_release.go` (holds), `services/cancellation_order_refund.go` (`runCancellationGatewayRefund` provider switch: wallet | razorpay | stripe), `handlers/payment.go` (order refund `ToWallet`).

---

## 1. Money model (unchanged escrow)

Customer pays the full plan upfront into platform escrow (food + platform fee + GST + delivery). Chef payout is **held per day**, released as each day is served.

**FIRM RULE — the refund base EXCLUDES the platform fee, GST, and delivery.** These are **never** refunded, for both meal plans and group orders. The refundable amount for a day is the **pure food value** net of the platform fee and taxes — i.e. the refund seam operates on `refundableFood(day) = food − platformFee − GST − delivery` (whatever of those apply per day). The platform always keeps its fee + GST + delivery.

**Refund amount = refundableFood(day) × proportion**, where proportion ∈ {1.0 full, 0.5 half, 0.0 none}. The chef keeps `(refundableFood − customer refund)` as prep compensation (their held payout is released for that slice; the refunded slice is clawed back). Every refund path — auto (>12h), chef Full/Half, admin pay — computes off this fee/GST-excluded base; a Full refund is 100% of the **food only**, not the gross the customer paid.

---

## 2. The 12h guardrail (auto vs explicit)

Each day has a cook-start time (`mealPlanDayStartIST` from the chef schedule). Let `lead = cook_start − now`.

- **`lead > 12h` (early):** the chef has not started prep → **AUTO-APPROVE**. Full food refund issued automatically, chef payout for that day reversed. No chef or admin action.
- **`lead ≤ 12h` (late):** the chef may have started prep → **EXPLICIT**. Routed to the chef to decide the proportion (full/half/none); then admin pays.

The 12h threshold is a `platform_settings` key (`mealplan.refund_prep_cutoff_hours`, default 12), so ops can tune it without a deploy.

---

## 3. Per-day refund state machine

```
confirmed
  │  customer requests skip (or the day is part of a whole-plan cancel)
  ▼
[evaluate lead time]
  ├─ lead > 12h ──────────────► auto: full food refund ─► refunded   (destination = wallet, instant)
  └─ lead ≤ 12h ──────────────► skip_pending_chef
                                   │
        chef DECLINES ◄────────────┤ chef reviews
        (day proceeds, no refund)  │
                                   ├─ accept · FULL ─► refund_pending_admin (100% food)
                                   ├─ accept · HALF ─► refund_pending_admin (50% food)
                                   └─ accept · NONE ─► resolved_no_refund   (chef paid 100%, day skipped, customer forfeits)
refund_pending_admin
  ├─ admin pays → WALLET  ─► refunded  (instant; chef gets food − refund)
  └─ admin pays → SOURCE  ─► refunded  (Razorpay refund to card/UPI, ~5–7 business days per RBI)
```

- **Auto path** skips chef+admin entirely (full refund → wallet).
- **NONE** needs no customer payment, so it auto-resolves (chef payout released, day marked skipped, no refund) — no admin pay step.
- **DECLINE** returns the day to `confirmed` (it will be cooked and delivered; customer charged as normal).

**Whole-plan cancel** = run every still-unserved day through the same evaluation: days `>12h` out auto-refund (full → wallet); a day `≤12h` out goes to the chef. The plan flips to `cancelled` once all its days reach a terminal state (refunded / resolved_no_refund / declined-then-served).

---

## 4. Refund destination + eligibility

| Order type | Wallet refund? | Original method? |
|---|---|---|
| **Meal plan** (day skip / plan cancel) | ✅ default (instant) | ✅ admin option (RBI ~5–7 days) |
| **Group order** | ✅ default (instant) | ✅ admin option |
| **On-demand à-la-carte order** | ❌ **blocked** | ✅ only |

- **Guard to add:** the order-refund `ToWallet` path (`handlers/payment.go`) rejects wallet for a standalone à-la-carte order — allowed only when the order is a meal-plan shell or a group order.
- Wallet refund reuses `CreditWallet(WalletSourceRefund)` → dual-writes to the ledger (shadow) into the `user_wallet_refund` bucket.
- Original-method refund reuses the existing `runCancellationGatewayRefund` provider switch (`razorpay` → `rz.Refund`); customer told "5–7 business days."

---

## 5. Actors & surfaces

- **Customer (mobile):** requests skip (day) / cancel (plan); sees the outcome + timeline ("₹X to your wallet now" or "₹X to your card in 5–7 days"); for a late request, sees "pending your chef's review."
- **Chef (vendor app):** a review queue for late (≤12h) skip/cancel requests → Accept (Full / Half / None) or Decline, with the reason (prep started).
- **Admin (web + tesserix-home):** a pay queue for `refund_pending_admin` items → choose destination (Wallet / Original) → execute. Maker/checker + audit (reuse the existing cancel-requests admin pattern).

---

## 6. Scenarios (end-to-end, exhaustive)

1. **Skip, >12h** → auto full refund → wallet → chef payout reversed → day `refunded`.
2. **Skip, ≤12h, chef Full** → pending admin → admin pays wallet/source → full refund → chef reversed.
3. **Skip, ≤12h, chef Half** → admin pays → 50% refund to customer, 50% payout released to chef.
4. **Skip, ≤12h, chef None** → resolved, no customer refund, chef paid 100%, day skipped.
5. **Skip, ≤12h, chef Decline** → day back to `confirmed`, cooked + delivered, customer charged.
6. **Cancel plan, all days >12h** → all auto full refunds → wallet → plan `cancelled`.
7. **Cancel plan, mixed** → early days auto-refund; imminent day → chef decides; plan cancels when all resolved.
8. **Admin pays to original** → Razorpay refund, RBI 5–7 days messaging.
9. **On-demand order refund** → wallet blocked, source only.
10. **Chef non-response** (late request) → after a chef-response window (`platform_settings`, default 24h) it **escalates to admin** (admin can decide full/half/none + pay). Never auto-refunds a late day (chef may have cooked).
11. **Concurrency/idempotency** → reuse existing per-day locks + `dayRefundKey` idempotency; a day resolves exactly once.
12. **Money conservation** → refunded slice clawed back from chef; `escrow-ledger` reconcile + `wallet-ledger` reconcile both stay balanced.

---

## 7. Bug fixed as part of this

Cancelling a plan whose days are `skip_req` currently 500s (the refund path doesn't handle days already mid-skip). The new unified flow evaluates each day by its **current** state, so a plan cancel over `skip_pending_chef` / `refund_pending_admin` days resolves them correctly instead of erroring.

---

## 8. Open decisions (my recommended defaults — confirm or adjust)

1. **NONE outcome:** customer forfeits, chef paid 100%, day skipped (not delivered). *(Alt: day still delivered.)* — **Recommend forfeit-skipped.**
2. **Chef non-response window:** 24h → escalate to admin (never auto-refund a late day). — **Recommend 24h.**
3. **Destination default:** wallet (instant) pre-selected for admin; original is a deliberate choice. Auto-approved (>12h) always → wallet. — **Recommend wallet-default.**
4. **Whole-plan cancel with a late day:** the plan stays `confirmed`/partially-resolving until the chef handles the late day, rather than blocking the cancel. — **Recommend non-blocking, per-day resolution.**
5. **Group orders:** same wallet-eligibility; the chef-decision step applies only if a group order has a comparable prep window — otherwise admin-only. — **Recommend meal-plan gets the full flow first; group-order = wallet-eligibility + admin destination choice, no chef-proportion step initially.**

---

## 9. Rollout

Gated behind `MEALPLAN_REFUND_FLOW_V2_ENABLED` (default off) so the current behaviour is unchanged until validated. Backend + tests first (money-critical, TDD), then vendor UI, then admin UI, then customer copy, then enable + deploy.
