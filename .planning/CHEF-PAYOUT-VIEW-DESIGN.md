# Chef payout — one persisted row per chef per order

**The single reference for what a chef is paid.** Owner decisions, 4 Aug 2026.
Not yet implemented. Closes D-21 in `TEST-EXECUTION-2026-08-04.md`.

## The formula — this is the whole thing

```
chef payout = food subtotal
            + delivery fee   (only when the chef carried the leg AND it was charged)
            + chef tip       (optional, when tipped)
            − penalties      (per policy, when raised)
```

**Nothing else.** No commission deduction, no GST of any kind, no TDS.

Worked against the live order `HC26080411237391`:

| Customer pays | | Chef is paid | |
|---|---:|---|---:|
| Subtotal | 320.00 | Food subtotal | **320.00** |
| Delivery fee | 0.00 | Delivery fee (free zone) | 0.00 |
| Platform fee | 13.53 | Chef tip | 0.00 |
| CGST | 9.22 | Penalties | 0.00 |
| SGST | 9.22 | | |
| **Total** | **351.97** | **Net payable** | **320.00** |

The customer's ₹351.97 is never shown to the chef. The platform keeps the
platform fee and accounts for all GST.

## Why no GST sits with the chef

**CGST s.9(5) + Notification 17/2017-CT(R)** (from 1 Jan 2022): where
**"restaurant service"** is supplied *through* an e-commerce operator, the **ECO
pays the GST** — 5%, no ITC, in cash. **CBIC Circular 164/20/2021-GST** confirms
cloud/central kitchens are "restaurant service"; a home kitchen supplying through
the platform follows the same reasoning.

Consequences:
- fe3dr is liable for the 5% food GST, **not** the chef — regardless of whether
  the chef is registered.
- The chef needs no GST registration for this (Notification 65/2017-CT exempts
  ECO service suppliers below ₹20 lakh; 9(5) supplies don't trigger s.24(ix)).
- **No TCS under s.52** — expressly excluded where the ECO is liable under 9(5).
  The ECO cannot be deemed supplier and collector at once.

The live figures already match: ₹16.00 on a ₹320 subtotal is 5%; ₹2.44 on a
₹13.53 platform fee is 18%. The tax computation is correct — only *who keeps the
food GST* was wrong.

⚠ **Not CA-signed-off.** Confirm home-chef food is "restaurant service" under
9(5) before go-live. Also confirm s.194-O: 0.1% since 1 Oct 2024 (Finance (No.2)
Act 2024), and waived for an individual/HUF participant with gross sales
≤ ₹5 lakh in the FY who has furnished PAN — most chefs — so if it ever applies it
must be conditional, never flat.

## ⚠ Conflicts with the code as written — resolve before building

**1. Commission is stored but is not in the payout.** Orders carry
`commission_rate 0.06`, and `ComputeOrderEarnings` does
`netPayout = gross − commission − tds`. Under the formula above there is **no
commission deduction at all** — the platform's revenue is the customer-paid
platform fee. On a ₹320 order that is a **₹19.20 difference per order**.
Confirm: is the 6% commission dead (superseded by the platform fee), or does it
still apply and the formula above is display-only?

**2. Food GST is currently credited to the chef.** `ComputeOrderEarnings` has
`gross = itemRevenue + Tax + chefTip`, and `ChefTaxOf` (the D-09 fix, verified
live at ₹955.40) exists specifically to give the chef the food share. Excluding
it is a **~5% reduction** and settled statements already paid it.

**3. Commission GST is computed but never deducted.** CGST/SGST/IGST on
commission are calculated, then not subtracted in `netPayout`. Moot if commission
goes away (conflict 1), but flag it either way.

**All changes are FORWARD-ONLY. Do not restate settled statements.**
No back-pay: 10 delivered `chef_delivery` orders (31 Jul – 4 Aug) carry ₹391.20
of delivery fees the chef was never paid; they stay settled. The 4
platform-carried `delivery` orders with fees (₹175.06) are correctly excluded —
the platform carried those legs.

## Schema — the one-stop row

DDL goes in **`apps/api`** (AutoMigrate + post-migrate DDL), **not**
`tesserix-k8s`. That repo's `db-schema-bootstrap` only applies SQL to a database
it just created; `homechef_db` is live.

```
order_chef_payouts
  id             uuid pk
  order_id       uuid  UNIQUE NOT NULL   -- one row per order, ever
  chef_id        uuid  NOT NULL  (indexed)
  currency       text  NOT NULL

  food_amount    numeric(12,2) NOT NULL  -- order subtotal
  delivery_fee   numeric(12,2) NOT NULL  -- 0 unless chef carried a charged leg
  chef_tip       numeric(12,2) NOT NULL  -- chef's tip only, never the driver's
  penalty        numeric(12,2) NOT NULL  -- per policy, positive = deducted
  net_payout     numeric(12,2) NOT NULL  -- food + delivery + tip − penalty

  status         text NOT NULL           -- pending | released | reversed
  computed_at    timestamptz NOT NULL
  created_at / updated_at
```

- `UNIQUE(order_id)` is what makes it idempotent — a retry, a re-drive or a
  concurrent write settles exactly once.
- `net_payout` is the ONLY number either surface renders. **Neither client
  recomputes.** Four surfaces doing their own arithmetic is how checkout came to
  show ₹264.58 for a receipt that said ₹264.57.
- No commission / GST / TDS columns: they are not part of the chef's payout, and
  storing them invites a second, contradictory version of the truth. The
  platform's own take stays derivable from the order row.

## No duplication — the rules

1. **One writer.** A single service function builds the row at delivery.
   Nothing else writes these columns.
2. **One reader shape.** Vendor app and tesserix-home consume the *same*
   serialised payout object from the API. No parallel DTOs.
3. **Penalties are referenced, not copied.** The column carries the amount
   attributed to this order (`chefcancel:<orderID>`); the money still moves at
   the weekly settlement, and the ledger stays the authority for when.
4. **Delivery fee is conditional at write time**, not render time:
   `fulfillment_type == chef_delivery AND delivery_fee > 0`, else 0 — and the UI
   omits the line entirely rather than showing ₹0.

## Surfaces

1. **Vendor app** — replace the PRICING block in `app/orders/[orderId].tsx`.
   Chef sees: food, delivery (only when charged), tip, penalty (only when
   raised), **"You'll be paid ₹X"**. No platform fee, no GST, no customer total.
2. **Admin (tesserix-home)** — `/admin/*` endpoint returning the same object so
   staff verify per order per chef against one authority.

## Build order

1. Resolve the commission question (conflict 1) — it changes the number.
2. CA sign-off on 9(5).
3. Model + migration + single idempotent writer, with tests.
4. Chef order response + admin endpoint (one shared serialiser).
5. Vendor screen, then tesserix-home view.
