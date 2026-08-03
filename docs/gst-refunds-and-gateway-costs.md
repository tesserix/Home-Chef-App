# GST on the platform fee, refunds, and what Cashfree actually costs

Status: **decision input, not tax advice.** Every rate here needs the platform's CA to
confirm before it drives money. Companion to `india-gst-model.md`, which covers the
statutory background; this file is the worked arithmetic behind the pricing decision and
the refund consequences that follow from it.

All examples use a **₹500 food order**, priced two ways — pickup and delivery — and carried
through every refund tier.

---

## 1. The problem in one line

The platform fee is charged to the customer at **5%** GST. It is a support service supplied
by the platform in its own name, not restaurant service, so **18%** is due. The difference
is not collected from anyone — the platform absorbs it, silently, on every order.

Second, smaller leak: the GST Cashfree charges on its own fee is currently unreclaimable,
because the platform's only output supply sits at the 5% no-ITC restaurant rate.

---

## 2. Option B — the chosen model

The customer-facing platform fee stays **4.99% all-in**. GST comes *out* of it rather than
being added on top. The customer's total goes slightly **down**; the platform's real margin
goes **up**, because it stops absorbing a 13-point tax gap.

> **Option B does not mean hiding the GST.** A tax invoice must state tax separately
> (CGST Rule 46). "Inclusive" is a *pricing* decision — don't *add* 18% on top — not a
> presentation one. The fee still itemises as net + tax on the invoice.

### Order P — pickup, no delivery

```
Food                                    500.00
Platform fee (net)                       21.14      ← 24.95 all-in
CGST 2.5% on food                        12.50
SGST 2.5% on food                        12.50
CGST 9%  on platform fee                  1.91
SGST 9%  on platform fee                  1.90
                                       ────────
TOTAL                                   549.95
```

### Order D — delivery ₹39

Delivery held at **5%** pending the CA's answer to `india-gst-model.md` §7 Q2 (chef
self-delivery as a composite supply). The live fulfilment mode is chef self-delivery — all
third-party providers are disabled — so the aggregator's 18% answer does not obviously apply.

```
Food                                    500.00
Delivery                                 39.00
Platform fee (net)                       21.14      ← 24.95 all-in
CGST 2.5% + SGST 2.5% on food             25.00
CGST 2.5% + SGST 2.5% on delivery          1.95
CGST 9%   + SGST 9%   on platform fee      3.81
                                       ────────
TOTAL                                   590.90
```

### Against today, on Order P

| | Today | Option B |
|---|---|---|
| Customer pays | 551.20 | **549.95** (−1.25) |
| Platform fee, gross | 24.95 | 24.95 |
| − GST absorbed by the platform | −3.24 | 0 |
| − GST inside the fee | 0 | −3.81 |
| − Cashfree MDR | −10.75 | −10.72 |
| − GST on MDR | −1.93 *(stuck)* | 0 *(reclaimable)* |
| **Platform's real margin** | **9.03** | **10.42** |

Nominal take drops 4.99% → 4.23%. Real take rises, because the 13-point gap and the
stranded input credit both disappear.

---

## 3. MDR — what it is and why it matters here

**MDR = Merchant Discount Rate**: the fee an acquirer or gateway charges a merchant to
process a transaction, as a percentage of transaction value. "Discount" is historical — the
acquirer settles the transaction amount *less* their cut, i.e. at a discount to face value.

For this platform it is Cashfree's **1.95%**.

| | Order P | Order D |
|---|---|---|
| MDR 1.95% of total | 10.72 | 11.52 |
| GST on MDR (18%) | 1.93 | 2.07 |
| Gross gateway cost | 12.65 | 13.59 |
| Less ITC (reclaimable under Option B) | −1.93 | −2.07 |
| **Real cost** | **10.72** | **11.52** |

Four things about MDR that change the numbers:

- **It is charged at capture and not returned on a refund.** A cancelled order is therefore
  never neutral — the gateway fee is sunk regardless of what the customer gets back. This is
  the single most under-modelled cost in the current design.
- **UPI and RuPay debit carry zero MDR by law** (Payments and Settlement Systems Act,
  since January 2020). The real blended rate depends entirely on the payment mix — a
  UPI-heavy customer base pays well under 1.95% on average. Pull the actual figure from
  Cashfree settlement reports before trusting any margin number here.
- **MDR attracts 18% GST**, and that GST is what becomes reclaimable once the platform fee
  is an 18% output supply.
- ⚠️ **To confirm against the Cashfree contract:** whether a separate refund-processing fee
  applies on top, and whether MDR is genuinely non-returnable on *partial* refunds. The
  codebase currently assumes ~2% as a proxy (`GatewayFeeLevyPercent: 2.0`).

---

## 4. Is the GST refundable, or can it be made non-refundable?

**Neither — GST is not a fee that can be labelled.** It is a tax on a supply, and it follows
the supply:

- **A supply not delivered** → the tax on it was never really due. Refund it to the customer
  and recover it from the government with a **credit note under §34 CGST Act**. Withholding
  it gains the platform nothing: it would be holding tax it still owes, while the customer
  is out of pocket for tax on nothing.
- **A supply delivered** — including a platform fee legitimately retained, or the chef's
  share of food on a partial cancellation → that tax **is** due. Keep it and remit it.

### The rule, in one line

> **Refund the tax on whatever you refund. Keep the tax on whatever you keep.**

The **platform fee** can absolutely be non-refundable — that is a commercial choice, already
made and already disclosed in-app ("The platform fee isn't refundable"). When the fee is
retained, its GST is retained and remitted with it. That disclosure stays accurate and
sufficient under Option B.

⏰ **Deadline.** The credit note must be declared by **30 November following the end of the
financial year** of the original supply (or the annual return date, whichever is earlier).
After that the customer can still be refunded, but the GST becomes the platform's cost.

---

## 5. Refund scenarios

The tier percentage applies to the **food only**. Delivery is all-or-nothing — refunded in
full if no driver was dispatched, nothing once one was, since they are paid regardless. The
platform fee is retained except on chef fault.

### Order P — pickup

| | Chef cancels | Customer, pre-accept | 75% | 50% |
|---|---|---|---|---|
| Food refunded | 500.00 | 500.00 | 375.00 | 250.00 |
| GST on that food | 25.00 | 25.00 | 18.75 | 12.50 |
| Platform fee refunded | 21.14 | — | — | — |
| GST on the fee | 3.81 | — | — | — |
| **Customer gets back** | **549.95** | **525.00** | **393.75** | **262.50** |
| Chef keeps (food) | 0 | 0 | 125.00 | 250.00 |
| Platform keeps (net fee) | 0 | 21.14 | 21.14 | 21.14 |
| Cashfree (sunk) | −10.72 | −10.72 | −10.72 | −10.72 |
| **Platform margin** | **−10.72** | **+10.42** | **+10.42** | **+10.42** |
| Credit note to raise | 28.81 | 25.00 | 18.75 | 12.50 |

### Order D — delivery

| | Chef cancels | Customer, pre-accept | 75%, not dispatched | 50%, dispatched |
|---|---|---|---|---|
| Food refunded | 500.00 | 500.00 | 375.00 | 250.00 |
| GST on that food | 25.00 | 25.00 | 18.75 | 12.50 |
| Delivery refunded | 39.00 | 39.00 | 39.00 | — |
| GST on delivery | 1.95 | 1.95 | 1.95 | — |
| Platform fee + its GST | 24.95 | — | — | — |
| **Customer gets back** | **590.90** | **565.95** | **434.70** | **262.50** |
| Chef keeps (food) | 0 | 0 | 125.00 | 250.00 |
| Driver/chef keeps delivery | 0 | 0 | 0 | 39.00 |
| Platform keeps (net fee) | 0 | 21.14 | 21.14 | 21.14 |
| Cashfree (sunk) | −11.52 | −11.52 | −11.52 | −11.52 |
| **Platform margin** | **−11.52** | **+9.62** | **+9.62** | **+9.62** |
| Credit note to raise | 30.76 | 26.95 | 20.70 | 12.50 |

Every column reconciles exactly to the order total, and no credit note is ever raised
against a supply that actually happened.

---

## 6. The chef-cancel hole — and the mechanism that already exists

The first column of both tables is a guaranteed loss: **₹10.72 (pickup) or ₹11.52
(delivery), every chef cancellation.** Everything is refunded including the platform's own
fee, while Cashfree keeps its cut.

`GatewayFeeLevyPercent` exists for exactly this — 2% of the amount refunded, deducted from
the chef's next weekly settlement on a chef-fault cancel, through the same raise → grace →
admin waiver → statement-deduction path as the cancellation penalty. It **defaults off**
(`GatewayFeeLevyEnabled: false`), so the platform is absorbing it today.

| | Levy off (today) | Levy on at 2% |
|---|---|---|
| Order P, chef cancel | −10.72 | 2% × 549.95 = 11.00 → **+0.28** |
| Order D, chef cancel | −11.52 | 2% × 590.90 = 11.82 → **+0.30** |

An admin toggle, no deploy. It converts the worst case from a guaranteed loss to roughly
break-even, and puts the cost on the party that caused it.

---

## 7. What breaks under Option B

`ComputeCancellationRefund` apportions tax by **pre-tax value**:

```
taxRefund = totalTax × (foodRefund + deliveryRefund) ÷ (food + delivery + platformFee)
```

This is correct **only while every component shares one rate** — which is true today, and
verified against a real order: pre-accept cancel refunded food ₹320.00 + delivery ₹39.12 +
tax ₹17.95, retained the fee ₹15.97 and its GST ₹0.80, summing to ₹393.84 = order total.

Once the fee sits at 18% and everything else at 5%, value stops being a proxy for tax — the
fee carries a disproportionate share of it.

**Order P** (total tax 28.81, pre-tax base 521.14)

| Tier | Formula gives | Actually attributable | Platform loses |
|---|---|---|---|
| 100% food | 27.64 | 25.00 | −2.64 |
| 75% | 20.73 | 18.75 | −1.98 |
| 50% | 13.82 | 12.50 | −1.32 |

**Order D** (total tax 30.76, pre-tax base 560.14)

| Tier | Formula gives | Actually attributable | Platform loses |
|---|---|---|---|
| 100% food + delivery | 29.60 | 26.95 | −2.65 |
| 75% + delivery | 22.73 | 20.70 | −2.03 |
| 50%, dispatched | 13.73 | 12.50 | −1.23 |

Conservation still holds — the books balance — but the platform hands back tax belonging to
a fee it kept, and **cannot credit-note it**, because that supply did happen. It comes
straight off margin, on top of the gateway hit.

**The fix is small.** The per-component snapshot frozen on the order at checkout
(`tax_food`, `tax_service`, `tax_delivery`) is exactly what the splitter needs: refund
`tax_food × pct`, apply the delivery rule to `tax_delivery`, and never touch `tax_service`
unless the fee itself is being refunded.

---

## 8. Stacked up, on a cancelled Order P

| | Today | Option B, splitter unfixed | Option B, splitter fixed |
|---|---|---|---|
| Customer pre-accept cancel | +9.03 | +7.78 | **+10.42** |
| 75% cancel | +9.03 | +8.44 | **+10.42** |
| Chef cancel, levy off | −12.68 | −10.72 | −10.72 |
| Chef cancel, levy on at 2% | −1.66 | +0.28 | **+0.28** |

---

## 9. Recommended sequence

Order matters. Flipping the rate before fixing the splitter costs ₹1.20–2.65 on every
cancellation in the interval.

1. **Turn the gateway-fee levy on.** Admin toggle, no deploy, largest single improvement.
2. **Fix the refund splitter** to use the per-component tax snapshot instead of apportioning
   by value.
3. **Then set the platform fee to 18%, GST-inclusive.** A value change in `tax_rates`.
4. **Leave delivery at 5%** until the CA answers §7 Q2.

---

## 10. Open questions

1. **§7 Q2 — chef self-delivery.** Composite supply at 5%, or a separate delivery service at
   18%? Self-delivery is the live mode, so this is not academic.
2. **§7 Q5 — whose supply is the platform fee?** If part of it is really commission to the
   chef rather than a service to the customer, the 18% sits on a different party. Changes
   who bears it, not the rate.
3. **Past periods.** Orders to date collected 5% on the fee where 18% was due — roughly
   ₹3.24 per ₹500 order. Whether that is voluntarily disclosed and paid is a filing decision
   for the CA. Worth getting a count of affected orders first.
4. **Real blended MDR.** Every margin figure here assumes 1.95%. With zero-MDR UPI in the
   mix the true average is lower, and every "platform margin" row improves.
5. **Cashfree refund mechanics.** Separate refund fee? MDR returned on partial refunds?
   Both change section 5.
