# Pricing Model v2 — Design

**Date:** 2026-08-03
**Status:** Approved by owner (this session); implementation not started
**Drivers:** Money-trail test run `.planning/TEST-EXECUTION-PLAN.md` (defects D-01…D-08), GST
compliance gaps (Section 9(5), Sept 2025 delivery-GST clarification), invisible gateway costs.

## 1. Decisions (owner-approved)

| # | Decision | Choice |
|---|---|---|
| 1 | Take-rate structure | Dual-sided, rebalanced: **₹10 flat customer platform fee + 8% chef commission** (replaces 4.99% customer fee + 6% commission) |
| 2 | Food GST | **Platform collects and remits under Section 9(5)** — chef is paid food price only, never any GST |
| 3 | Commission GST (18%) | **Deducted from chef** (invoice reads "commission + GST"), no longer platform-absorbed |
| 4 | Delivery GST | **Split by who delivers:** chef self-delivery → 5% (composite restaurant supply); 3PL/platform-arranged → 18% (local delivery service under 9(5), per Sept 2025 clarification) |
| 5 | Gateway cost (Cashfree MDR) | **Modeled per order in the ledger, absorbed by platform margin.** No new customer/chef line. Enable existing #885 chef-fault refund levy (2%, no grace) |
| 6 | Refund/cancel policy | **Structure unchanged** (3-path policy proven by test run). Add wallet-first refund destination; retained amount on customer-fault cancel becomes platform fee + its 18% GST |
| 7 | Chef landing | **Communicate + menu reprice window** (~2 weeks, suggested +4–5%, payout simulator in vendor app). No permanent commission giveaway |
| 8 | Engine architecture | **Approach B: single `services/pricing/` engine** (pure functions over integer paise, policy snapshot per order, versioned dispatch) |

## 2. Customer price stack

| Line | Rule | GST |
|---|---|---|
| Food subtotal | Σ(item price × qty) − discount | 5% |
| Delivery fee | existing `QuoteOrderDeliveryFee` logic, pass-through at cost | 5% (self-delivery) / 18% (3PL) |
| Platform fee | ₹10 flat, per order (config `platform_fee_flat_paise`) | 18% |
| Tip | pass-through to chef | none |

```
total = food + delivery + platformFee + gstFood + gstDelivery + gstFee + tip − discount
```

- **Place of supply** (fixes D-03): pickup → chef's state; delivery → drop-address state.
  One shared helper (`PlaceOfSupply(order) → state`) feeds BOTH the customer invoice and the
  chef earnings/settlement math. Intra-state → CGST+SGST (odd paise to SGST, as today);
  inter-state → IGST.
- **Promo discounts:** reduce the food GST base (as today). Chef-funded portion reduces the
  chef's commission base (unchanged, #39).
- **Wallet/loyalty credits:** applied against food+delivery only, never fees/taxes (unchanged).
  Capture = total − credits.

### Worked example (test order: ₹320 food, ₹39.12 self-delivery, ₹20 tip, intra-state)

| | v1 (today) | v2 |
|---|---|---|
| Food | 320.00 | 320.00 |
| Delivery | 39.12 | 39.12 |
| Platform fee | 15.97 (4.99%) | 10.00 |
| GST | 18.75 (5% on food+delivery+fee) | 19.76 (16.00 food + 1.96 delivery + 1.80 fee) |
| Tip | 20.00 | 20.00 |
| **Customer total** | **413.84** | **408.88** (−4.96) |

Same order via 3PL: delivery GST 7.04 → total 413.96.

## 3. Chef settlement stack

```
itemRevenue    = subtotal − chefFundedDiscount        (floored at 0)
commission     = 8% × itemRevenue                     (rate FROZEN on the order at checkout)
commissionGST  = 18% × commission                     (deducted; chef invoice shows commission + GST)
tdsBase        = itemRevenue + chefTip                (food GST is NOT chef income any more)
tds            = 1% × tdsBase                         (Section 194-O)
netPayout      = itemRevenue + chefTip − commission − commissionGST − tds
```

- Self-delivery fee is paid to the chef **on top**: not commissioned, no GST withheld from it,
  outside the TDS base (CA to confirm; matches current #390 exclusion).
- Tips: 100% pass-through, never commissioned (INV-6 unchanged); inside TDS base
  (conservative, matches current behaviour; CA question #3).

### Worked example (same order)

| | v1 corrected (food-GST-to-chef, D-02 fixed) | v2 |
|---|---|---|
| Chef gross | 356.00 (320 + 16.00 GST + 20 tip) | 340.00 (320 + 20 tip) |
| Commission | −19.20 (6%) | −25.60 (8%) |
| Commission GST | 0 (absorbed) | −4.61 |
| TDS | −3.56 | −3.40 |
| Net (food+tip) | 333.24 | 306.39 |
| + delivery | 39.12 | 39.12 |
| **Chef receives** | **372.36** | **345.51** |

Chef net on food ex-tip: **89.6%** of subtotal (v2) vs 97.9% (v1) — **−8.3 points ≈ −₹27 on
this order**. Composition: ~5.0 pts food GST that was never legally the chef's, ~2.0 pts
commission increase, ~1.1 pts commission GST, ~0.2 pts TDS base shift. This is the number the
chef comms must state plainly. Softening dial if needed: commission 8%→7% gives chefs back
~1.1 pts at ₹3.20/order platform cost.

## 4. Platform per-order economics (ledger-modeled)

Per order the engine emits a platform-economics record:

```
revenue        = platformFee + commission                       (ex-GST)
ownOutputGST   = 18% × (platformFee + commission)               (funded by customer + chef)
passThroughGST = gstFood + gstDelivery                          (9(5) remittance liability)
gatewayEst     = mdrRate × capturedAmount (+ 18% GST on MDR, input-creditable)
marginEst      = revenue − gatewayEst
```

- `mdrRate` is config (`gateway.mdr_percent`, default 1.95) and snapshotted per order.
- Reconciliation: when Cashfree settlement reports land, actual MDR is written next to the
  estimate. Refund MDR losses (gateways don't return MDR) post as negative margin events.
- Reference order: revenue 35.60 − gateway ≈7.97 → **margin ≈ ₹27.6 (~8.6% of food)** vs
  ≈ ₹23.5 in v1.

## 5. Refunds & cancellations

Structure unchanged; the money that moves is re-based on the v2 snapshot:

| Path | Customer gets | Chef gets | Platform keeps |
|---|---|---|---|
| Chef-fault (reject/cancel) | 100% incl. platform fee + all GST | ₹0 | ₹0 (and eats gateway loss, less #885 levy) |
| Customer cancel pre-accept | total − (platform fee + its 18% GST) | ₹0 | fee + fee-GST |
| Customer cancel post-accept | tiered by chef-declared progress (existing tiers) | retained share via `chef_bonuses` | fee + fee-GST + share |

- **Wallet-first refunds:** default destination = wallet (instant, zero gateway loss); "to
  card" remains an explicit customer choice. Chef-fault refunds present card prominently.
- **#885 gateway-fee levy:** enabled, 2% of refunded amount, chef-fault only, no grace, no
  stacking with `cancel_late`.
- **GST credit notes:** every refund emits per-line credit-note amounts (food/delivery
  pro-rata to the refunded value) that reduce the platform's 9(5) remittance for the period.
- Rail-splitting (`SplitRefundByFunding`), RTO policy (#393), meal-plan refund tiers, and the
  payout-hold machine are unchanged.
- All nine invariants INV-1…INV-9 from the test plan carry over verbatim and become property
  tests on the engine.

## 6. Meal plans, group orders, tips

- **Meal plans** price through the SAME engine (kills D-08): 5% food GST; real per-day
  delivery quote honouring `chef_subscription_configs.delivery_fee` (the ₹2.99 default dies);
  **₹10 platform fee once per plan booking**; 8% commission; per-day apportionment and refund
  tiers unchanged but computed from the v2 snapshot.
- **Group orders:** same stack; platform fee charged once, on the initiator.
- **Tips (D-06):** post-delivery tips move to the Cashfree rail (`tips.go` currently
  Razorpay-only → dead). 100% pass-through preserved.

## 7. Engine architecture

New package `apps/api/services/pricing/`:

- `PriceOrder(PricingInput) → PriceBreakdown` — pure; integer paise in/out; consumes a
  `PolicySnapshot` (platform fee, commission %, all GST rates by line & fulfillment,
  MDR estimate, place-of-supply inputs).
- `SettleOrder(PriceBreakdown snapshot) → Settlement` — pure; chef payout + platform
  economics + remittance lines from the snapshot, never live policy.
- `PolicySnapshot` is serialized onto the order (`orders.pricing_snapshot` JSONB) with
  `orders.pricing_version` (`1` legacy, `2` new). Refund/settlement/statement paths dispatch
  on the version so in-flight v1 orders finish under v1 math.
- Handlers (`orders.go`, `meal_plan.go`, `group_order.go`) become thin adapters.
- `earnings.go` becomes a v1-only shim retained for historical statements; v2 statements read
  the engine.

### Schema additions

- `orders`: `pricing_version` (int, default 1), `pricing_snapshot` (jsonb), `gst_food_paise`,
  `gst_delivery_paise`, `gst_platform_fee_paise`, `gateway_fee_estimate_paise`,
  `place_of_supply_state`.
- New `platform_order_economics` table (order_id, revenue, gateway est/actual, remittance
  lines, margin) — append-only.
- `platform_policy` blob: adds `platformFeeFlatPaise`, `gstFoodPercent`,
  `gstDeliverySelfPercent`, `gstDelivery3plPercent`, `gstPlatformServicePercent`,
  `gatewayMdrPercent`; `PlatformFeePercent` retained for v1 orders only.

## 8. Money representation & rounding

- **Integer paise** for every stored amount, comparison, and engine computation. The
  float-rupee columns remain for display compatibility but are derived, never compared.
- Rounding: **half-up, once per line**, then sum — sums are never re-rounded (prevents ±1p
  drift); refunds round half-up (fixes OBS-11's truncation).
- Wallet goes paise-native: `wallet.balance_paise` + migration backfilling
  `round(balance × 100)`; all debit/credit comparisons in paise (the structural D-07 fix).

## 9. Rollout

1. **Pre-v2 hotfixes (ship now, independent):** D-07 paise wallet + refund-split guard (the
   money-minting exploit), D-01 (populate real delivery fee in chef responses), D-04
   (currency symbol), D-05 (vendor app crash), D-06 (Cashfree tips).
2. **Build the v2 engine** + property tests (INV-1…9) + golden-file tests reproducing every
   money figure in `.planning/TEST-EXECUTION-PLAN.md` §5 under v1, then asserting v2 values.
3. **CA sign-off gate** (checklist §10). No 9(5) money movement before sign-off.
4. **Chef comms + 2-week reprice window:** dated notice, plain statement of the payout change
   (§3 numbers), suggested +4–5% menu adjustment, vendor-app simulator ("your last 10 orders
   under the new model").
5. **Flip:** new orders and new meal-plan bookings stamp `pricing_version=2`. In-flight v1
   orders settle under v1. Meal plans flip the same day (most-wrong today).
6. **Post-flip audit:** re-run the money-trail scenario suite against v2; reconcile first
   weekly statement by hand.

## 10. CA confirmation checklist (design assumes → gate confirms)

1. Section 9(5) applies to unregistered home chefs supplying restaurant service through the
   platform; consequently **TCS u/s 52 does not apply** to those supplies (assumed).
2. Self-delivery at 5% as composite supply vs 3PL at 18% local delivery (scope of the Sept
   2025 clarification) (assumed split).
3. TDS 194-O base: food + tips in, delivery fee out (assumed).
4. Place of supply for pickup = chef's state, for platform fee = customer's state or chef's?
   (assumed: same as order's place of supply).
5. Credit-note mechanics for partial refunds under 9(5) (assumed pro-rata per line).

## 11. Defect resolution map

| Defect | Resolution |
|---|---|
| D-01 "Free delivery" lie | Populate `ChefProfileResponse.DeliveryFee` from self-delivery fields (pre-v2 hotfix) |
| D-02 chef overpaid fee/delivery GST | Structural — chef never receives any GST in v2 |
| D-03 IGST on intra-state pickup | Shared `PlaceOfSupply` helper used by invoice + settlement |
| D-04 "$199.00" message | Use `handlers/currency.go` symbol map (pre-v2 hotfix) |
| D-05 vendor app crash | Pre-v2 hotfix (unrelated to pricing, blocks testing) |
| D-06 dead post-delivery tips | Cashfree tip rail (pre-v2 hotfix) |
| D-07 float wallet minting loop | Paise-native wallet + debit-verified refund split (pre-v2 hotfix); engine is paise-only |
| D-08 meal-plan 8% / ₹2.99 | Meal plans priced by the same engine (v2) |

## 12. Out of scope

- Driver-side economics (3PL settlement is provider-billed; own fleet retired).
- Surge/dynamic pricing, chef-tier commission differentiation.
- Catering and the deferred tiffin-subscription billing model beyond what §6 covers.
- Non-INR currencies (engine takes rates from config; only IN rules are being certified).
