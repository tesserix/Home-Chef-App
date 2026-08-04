# Chef payout view — one persisted row per chef per order

Owner decisions captured 4 Aug 2026. **Not yet implemented.** D-21 in
`TEST-EXECUTION-2026-08-04.md` is the defect this closes.

## Why

The vendor order screen shows the chef the **customer's** numbers: Subtotal,
Platform fee ₹13.53, CGST ₹9.22, SGST ₹9.22, Total ₹351.97. Three faults:

1. The **platform fee is the product owner's cut** — never the chef's to see.
2. The **Total is the customer's**, and a chef reconciling a payout against it can
   never make it match.
3. The **GST is the customer's combined GST**: ₹18.44 = food ₹16.00 + the
   platform's service GST ₹2.44. The chef's share is ₹16.00. This is the D-09
   pattern (chefs credited the platform's GST) surviving in the display layer —
   D-09 was fixed in the earnings/statement path via `ChefTaxOf`, but this screen
   still renders the combined figure.

Admin (tesserix-home) has no per-order payout view at all, so payouts cannot be
verified against a single authority.

## Decisions

| Question | Decision |
|---|---|
| Food GST | **EXCLUDE** — owner decision, 4 Aug. The platform accounts for the GST; the chef neither charges nor remits it. See the tax note below — this reverses the earlier "include" recommendation and REDUCES chef payouts. |
| Platform fee / service GST / customer total | **Exclude.** Never shown to the chef. |
| Delivery fee | **Include when actually charged.** 3PL is dark, the chef carries the leg, so the fee is theirs. When the fee is 0 (free zone or pickup) the line is **hidden entirely**, not shown as ₹0. |
| Tips | Include the **chef** tip only. Driver tips are never the kitchen's. |
| Penalties | Show on the **order that caused them** (`chefcancel:<orderID>`), while money still moves at the weekly settlement. |

## ⚠ The delivery-fee conflict this creates

`services.ComputeOrderEarnings` (services/earnings.go) currently states:

```go
gross = itemRevenue + Tax + chefTip
// "DeliveryFee ... does NOT enter gross or net (it is the driver's money,
//  settled separately) (#390)"
```

It excludes the delivery fee **unconditionally**, including on `chef_delivery`
orders where the chef IS the driver. Under the decision above that is an
underpayment for every chef who delivers a paid leg.

### ✅ RESOLVED — owner decisions, 4 Aug

**The delivery fee is a pass-through, added AFTER TDS:**

```go
net   = gross − commission − tds + deliveryFee   // chef_delivery only
gross = itemRevenue + chefTip                    // formula itself UNTOUCHED
```

It does not enter `gross`, so the TDS basis is unchanged and **no existing figure
is restated** — `gross` feeds `tds = RateTDS × gross`, and the weekly statement,
FY statement and TDS certificate all scan that same computation.

**`fulfillment_type = chef_delivery` only, from go-live FORWARD.**

**No back-pay.** 10 delivered chef_delivery orders (31 Jul – 4 Aug) charged
₹391.20 in fees the chef was never paid; they stay settled. The 4
platform-carried `delivery` orders with fees (₹175.06) are correctly excluded —
the platform carried those legs.

Implementation must:
- add the fee at the **payout-row layer, NOT inside `ComputeOrderEarnings`**
- gate on `fulfillment_type == chef_delivery` AND `delivery_fee > 0`
- gate on a go-live timestamp so the 10 historical orders are never picked up

## Tax note — the food GST, and why it is the biggest item here

Owner decided the chef does **not** pay or receive the food GST: the platform
accounts for it. Rationale is **CGST s.9(5)** — where "restaurant service" is
supplied *through* an e-commerce operator, the ECO is liable for the GST, not the
supplier (why Swiggy/Zomato remit it themselves). Reinforced by most home chefs
sitting below the ₹20 lakh registration threshold, so they cannot charge or remit
GST at all.

**⚠ This REDUCES chef payouts and contradicts the code as written.**
`ComputeOrderEarnings` today does `gross = itemRevenue + Tax + chefTip` — it
credits the chef the food GST, and `ChefTaxOf` (the D-09 fix, verified live at
₹955.40) exists specifically to give them the food share. Settled statements
already paid it.

So this is not a display change:

- Chef payout falls by the food GST — ~₹16.00 on a ₹320 order, ~5%.
- **Forward-only**, same as the delivery fee. Do NOT restate history.
- `ChefTaxOf` / `ChefAttributableTax` stay as they are for the statement path;
  the payout row simply omits the tax component.

**Not signed off by a CA.** Confirm before go-live: (a) does home-chef food from
a home kitchen fall under 9(5) "restaurant service"; (b) TCS under s.52 and TDS
under s.194-O are platform obligations, and neither appears in the earnings code
beside the existing `RateTDS`. If (a) turns out otherwise this reverses, and
becomes a restatement across every chef.

## Shape

DDL goes in **`apps/api`** (AutoMigrate + post-migrate), NOT `tesserix-k8s`.
The workspace rule "all SQL lives in tesserix-k8s" covers *provisioning*;
`db-schema-bootstrap` only applies SQL to a database it just created, and
`homechef_db` is live.

```
order_chef_payouts
  order_id (unique)   chef_id        currency
  item_revenue        chef_tip       delivery_fee
  commission          commission_gst tds
  penalty             net_payout
  computed_at         source_rev
```

- Written **idempotently at delivery** from `ComputeOrderEarnings`, unique on
  `order_id`, so a retry or re-drive settles once.
- `net_payout` is the number both surfaces render. Neither client recomputes —
  four surfaces doing their own arithmetic is how checkout came to show ₹264.58
  for a receipt that said ₹264.57.

## Surfaces

1. **Vendor app** — replace the PRICING block on `app/orders/[orderId].tsx`.
   Chef sees: food, tip, delivery (only when charged), commission,
   penalty, **"You'll be paid"**. No platform fee, no service GST, no customer
   total.
2. **Admin (tesserix-home)** — `/admin/*` endpoint returning the same row, so
   staff verify per order per chef against one authority.

## Ordering

1. CA sign-off on the 9(5) question in the tax note.
2. Model + migration + idempotent write at delivery, with tests.
3. Chef order response + admin endpoint.
4. Vendor screen, then tesserix-home view.
