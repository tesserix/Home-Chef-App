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
| Food GST | **Include.** The chef is the supplier of the food and remits it. Matches `ChefTaxOf` and the D-09 fix verified live. |
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

**This must be resolved before the payout row is written**, because the row
persists the answer:

- `gross` feeds **TDS** (`tds = RateTDS × gross`), so adding delivery changes
  TDS withheld.
- Weekly statements, the FY statement and the TDS certificate all scan the same
  computation. Changing it restates figures chefs may already have been
  reconciled against — the exact hazard `ChefTaxOf` was written to avoid.

Recommended: add the delivery fee to the chef's payout **only for
`fulfillment_type = chef_delivery`**, from a stated go-live date forward, leaving
settled history untouched.

## Shape

DDL goes in **`apps/api`** (AutoMigrate + post-migrate), NOT `tesserix-k8s`.
The workspace rule "all SQL lives in tesserix-k8s" covers *provisioning*;
`db-schema-bootstrap` only applies SQL to a database it just created, and
`homechef_db` is live.

```
order_chef_payouts
  order_id (unique)   chef_id        currency
  item_revenue        food_gst       chef_tip
  delivery_fee        commission     commission_gst   tds
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
   Chef sees: food, food GST, tip, delivery (only when charged), commission,
   penalty, **"You'll be paid"**. No platform fee, no service GST, no customer
   total.
2. **Admin (tesserix-home)** — `/admin/*` endpoint returning the same row, so
   staff verify per order per chef against one authority.

## Ordering

1. Settle the delivery-fee/TDS question above.
2. Model + migration + idempotent write at delivery, with tests.
3. Chef order response + admin endpoint.
4. Vendor screen, then tesserix-home view.
