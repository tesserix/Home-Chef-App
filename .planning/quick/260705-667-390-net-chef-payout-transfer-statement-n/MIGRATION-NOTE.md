# MIGRATION NOTE — #390 Net chef payout == statement

This slice changed the chef settlement basis and froze the commission rate on the
order. Two classes of already-persisted data were computed on the OLD basis and
are intentionally **NOT rewritten** here.

## What changed (the new basis)

- **Chef transfer is now NET.** The Razorpay Route split and the Stripe transfer
  now pay `gross − commission − TDS` (was gross). One helper `chefNetPayout(order)`
  feeds all three payout sites (FSSAI-withhold audit, Route split, Stripe transfer)
  and equals `ComputeOrderEarnings(order).NetPayout` — the statement figure.
- **Gross = itemRevenue + Tax + ChefTip.** The food GST (`Tax`) is the chef's
  income and now enters gross; the delivery fee is the driver's and is **excluded**
  from the chef's gross/net. Previously gross wrongly included delivery and excluded
  tax.
- **Commission rate is frozen** on `orders.commission_rate` at checkout. The verify
  path, weekly statement, TDS certificate, and earnings breakdown all read that
  per-order column; a legacy `0` falls back to the live/default rate.

## (a) Historical WeeklyStatement rows — DO NOT rewrite

Existing `weekly_statements` rows were computed with the OLD (delivery-in-gross,
tax-out) formula. Payout-hold flags were OFF during that window, so **no real money
moved** on that basis — but regenerating or re-displaying an old statement now
yields different gross/net numbers. We do **not** rewrite historical rows in this
slice; the frozen totals on each row remain the record of what was reported for
that week.

## (b) REGULATORY caveat (W1) — already-issued TDS Form-16A certificates

`tds_certificate.go` recomputes gross + TDS **live** from the order rows each time a
certificate is generated. Because gross changed (Tax now in, delivery now out), a
regenerated certificate for **historical** orders will report a **different TDS**
than any Form-16A already issued for that period. This is a **regulatory
follow-up**: do NOT silently regenerate past certificates. Coordinate with finance
before re-issuing any historical TDS document so the deposited-vs-certified figures
stay reconciled at TRACES.

## (c) Legacy orders lack a frozen rate

Orders placed before this migration have `commission_rate = 0`. On statement /
earnings / TDS compute they fall back to the live `payout.commission_rate` (or the
6% default) via `rowRate`. New orders freeze the rate at checkout, so their split is
deterministic across create → verify → statement.

## (d) Out-of-scope follow-ups

- **Meal-plan-day payout (net):** the meal-plan escrow / `RefundDay` path is a
  separate settlement seam. Verify whether per-day meal-plan transfers also pay
  gross and need the same net-of-commission-and-TDS fix. Not touched here.
- **TCS §52:** Tax Collected at Source is CA-pending; not modelled in this slice.
- **Vendor UI copy:** the vendor earnings screens auto-reflect the DTO fields;
  any explanatory copy beyond that (e.g. "net of commission & TDS") is a separate
  content change.
