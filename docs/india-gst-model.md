# India GST — how it applies to Fe3dr, and where the code disagrees

Status: **planning input, not advice.** Rates and notifications below are as understood
at the time of writing; GST changes by notification and Council decision, so every figure
here needs confirmation from the platform's CA before it drives money. The open questions
in §7 are the ones that actually need an answer to build against.

Ties to #19 (counsel sign-off) and #396 (money model).

---

## 1. What Fe3dr is, in GST terms

An **e-commerce operator (ECO)**: it owns a digital platform through which another person
(the home chef) supplies restaurant service. That classification drives everything below —
in particular §9(5), which moves the tax liability for the food onto the platform.

Three distinct supplies happen on every order, and they are **not taxed alike**:

| # | Supply | From → To | Rate |
|---|--------|-----------|------|
| 1 | Restaurant service (the food) | chef → customer, *through* the platform | **5%, no ITC** |
| 2 | Platform / convenience fee | platform → customer | **18%** |
| 3 | Commission | platform → chef | **18%** |

Delivery charge is a fourth line whose treatment depends on who performs the delivery — see §4.

## 2. The food: 5%, and the platform pays it

**Rate.** Restaurant service is 5% with no input tax credit (Notification 11/2017-CTR as
amended by 46/2017-CTR). "Restaurant service" expressly covers cloud kitchens and takeaway,
so a home kitchen preparing and supplying food is inside it. The 18%-with-ITC rate applies
only to restaurants in *specified premises* (hotel units above the ₹7,500/day threshold) —
irrelevant here.

Intra-state → CGST 2.5% + SGST 2.5%. Inter-state → IGST 5%.

**Who pays — Section 9(5), CGST Act.** Since 1 Jan 2022, restaurant service supplied
*through* an ECO is notified under §9(5) (Notification 17/2017-CTR as amended by
17/2021-CTR). The ECO discharges the GST **as if it were the supplier**:

- The **platform** issues the tax invoice for the food line, under its own GSTIN.
- The **chef does not charge GST** on those supplies.
- The platform pays that 5% **in cash** — restaurant service carries no ITC, so it cannot be
  set off against input credits.
- The chef still reports the turnover, but as supplies on which the ECO pays tax.

Mechanics are in Circular 167/23/2021-GST.

**Why this matters commercially:** a chef under the registration threshold (§3) needs no
GSTIN to sell on the platform, because the platform is already paying the tax. That is the
single biggest onboarding unlock in this whole document.

**Place of supply** for restaurant service is where the service is *performed* — the
kitchen. So the food line is intra-state whenever the kitchen and the customer are in the
same state, and the platform needs a GST registration **in each state where it has chefs**.
Multi-state expansion is therefore a registration exercise, not just a marketing one.

## 3. Registration thresholds

| Party | Rule |
|---|---|
| Platform (ECO) | **Compulsory registration**, §24(ix) — no turnover threshold, from the first rupee. |
| Chef supplying only through the platform | Exempt from compulsory registration below the normal threshold, because the platform pays under §9(5). |
| Normal threshold | ₹20 lakh for services (₹10 lakh in special-category states). |

Composition scheme for chefs: §10(2)(d) bars anyone supplying through an ECO *required to
collect TCS*. Since §9(5) supplies are outside TCS (§4), whether a chef can still hold
composition is genuinely contested — **CA question**.

## 4. The platform's own charges

**Platform / convenience fee → 18%.** A support service supplied by the platform to the
customer in its own name. Not restaurant service, so not 5%.

**Commission from the chef → 18%.** Already modelled correctly in the codebase
(`services/earnings.go`, `handlers/chef_earnings.go`).

**Delivery charge → depends on who delivers.**

- Platform-arranged delivery (3PL, or the platform's own fleet): the platform is supplying a
  delivery service in its own name → **18%**. This was litigated against the large
  aggregators and the settled industry position is to charge 18%.
- Chef self-delivery, where delivery is incidental to the meal: arguable as part of the
  composite restaurant-service supply → 5% as the principal supply. **CA question**, and it
  matters here because self-delivery is a first-class fulfilment mode (#699).

**Tips** are a pass-through to the chef or rider and are not consideration for the
platform's supply — correctly excluded from the tax base today.

## 5. Collection obligations beyond output tax

| Obligation | Rate | Applies to |
|---|---|---|
| **GST TCS, §52** | 0.5% (0.25% CGST + 0.25% SGST), reduced from 1% w.e.f. 10 Jul 2024 | Net taxable supplies made through the platform by **registered** suppliers, **excluding** §9(5) supplies. Filed monthly in GSTR-8. |
| **Income-tax TDS, §194-O** | 0.1% of gross sales, reduced from 1% w.e.f. 1 Oct 2024 | Payments to e-commerce participants. No deduction for an individual/HUF participant with gross ≤ ₹5 lakh in the FY who has furnished PAN/Aadhaar — which is most home chefs. |

Neither exists in the codebase today.

## 6. Invoicing

A tax invoice must carry: supplier name, address and GSTIN; invoice number and date;
recipient details; HSN/SAC; taxable value; rate and amount of CGST/SGST/IGST separately;
place of supply; and a signature or digital signature.

- **The food line's invoice comes from the platform** under §9(5), not from the chef.
- E-invoicing (IRN) is a B2B obligation above ₹5 crore aggregate turnover — customer
  invoices here are B2C and outside it. A dynamic QR code applies only above ₹500 crore.
- Credit notes must reverse tax at the same rate and reference the original invoice —
  already implemented for the meal-plan refund path (`services/credit_note.go`, #834).

## 7. Open questions for the CA

1. Is the platform registered in every state where it has chefs, given place of supply for
   restaurant service is the kitchen?
2. Chef self-delivery — composite supply at 5%, or a separate delivery service at 18%?
3. Can a chef under composition supply through the platform, given §9(5) supplies attract no
   TCS?
4. Does the ₹5 lakh §194-O exemption cover the current chef base, or does TDS need building
   before the first chef crosses it?
5. Is the platform fee one supply to the customer, or should part of it be re-characterised
   as commission to the chef? The answer changes who bears the 18%.

---

## 8. What the code does today, and where it diverges

**Order tax** (`apps/api/handlers/orders.go:502-516`): a single rate from the `tax_rates`
table — the India row is `GST 5%, exclusive` (`apps/api/services/tax.go:114`) — applied to
`subtotal + deliveryFee + platformFee − discount`. Split 50/50 into CGST/SGST intra-state,
or IGST inter-state (`apps/api/services/gst.go`), where intra/inter is decided by chef state
vs delivery state.

| Component | Charged today | Should be | Divergence |
|---|---|---|---|
| Food | 5% | 5% | correct |
| Platform fee | 5% | 18% | **under-collected** |
| Delivery fee (platform-arranged) | 5% | 18% | **under-collected** |
| Delivery fee (chef self-delivery) | 5% | 5% or 18% | open (§7 Q2) |
| Commission from chef | 18% | 18% | correct — different code path |
| Tips | untaxed | untaxed | correct |

Worked example, a real pickup order (`HC26080306287649`): food ₹240.00, platform fee ₹11.98.

- Charged: 5% × ₹251.98 = **₹12.60**
- Correct: ₹240 × 5% = ₹12.00, plus ₹11.98 × 18% = ₹2.16 → **₹14.16**
- Shortfall **₹1.56** on a ₹264.57 order (~0.6%), borne by the platform, not the customer.

Two further gaps, both structural rather than arithmetic:

- **Invoice attribution.** `apps/mobile-customer/app/order/[id]/receipt.tsx` and
  `services/invoice_pdf.go` present the chef as the supplier of the food, with the chef's
  GSTIN. Under §9(5) the platform is the supplier of record for that line.
- **Place of supply** is derived from chef state vs delivery state for *every* component. That
  is right for the food (performance-based) but not necessarily for the platform fee, whose
  place of supply follows the recipient.

## 9. Implementation shape, if the answers above hold

Three separable pieces, in rising order of risk. Each is independently shippable.

1. **Display consistency.** One helper produces the itemised tax lines; every surface uses it
   — mobile order detail, `apps/web` order detail and meal-plan pages, admin. Today only
   checkout, the receipt and the invoice PDF itemise. No money moves.
2. **Per-component rates.** Tax becomes a sum over components rather than one rate on one
   base: food at the restaurant rate, platform fee and (per Q2) delivery at the standard
   rate. Needs `tax_rates` to carry more than one row per country, an order-level snapshot of
   each component's rate for refund correctness, and money-conservation tests alongside
   `meal_plan_escrow_fees_test.go`. Refund paths must return tax per component, not
   proportionally.
3. **§9(5) attribution.** The platform becomes the supplier of record for the food line:
   invoice header, GSTIN, receipt copy, credit notes, and the chef-facing statement all
   change. Largest blast radius, and the one most dependent on the CA's answers.

---

## 10. Every other tax this business owes

Sections 1–9 cover only the GST that rides on an order. These are the rest. None of them
appear anywhere in the codebase; most are filings rather than features, but two (§10.2 and
§10.4) become code the moment a chef crosses a threshold.

### 10.1 Income tax on the platform entity

The app names **Zivana Innovations LLP** as the India operator and **Tesserix Pty Ltd (ACN
694 070 865, NSW)** as the parent, so both regimes apply and the intercompany charge between
them is itself a tax event (#31).

| Entity form | Rate |
|---|---|
| LLP | 30% + 12% surcharge above ₹1 crore income + 4% cess. Alternate Minimum Tax 18.5% of adjusted total income. Partners' profit share is exempt in their hands. |
| Domestic company under §115BAA | 22% + 10% surcharge + 4% cess ≈ **25.17% effective**, conditional on forgoing most deductions. MAT does not apply. |
| Domestic company, ordinary | 25% (turnover ≤ ₹400 crore) or 30%, plus surcharge and cess, with MAT at 15%. |

Advance tax is payable quarterly (15 Jun / 15 Sep / 15 Dec / 15 Mar); shortfall carries
interest under §234B/C.

### 10.2 TDS the platform must deduct as a payer

| Section | Rate | On |
|---|---|---|
| **194-O** | 0.1% (cut from 1% w.e.f. 1 Oct 2024) | Gross sales of an e-commerce participant. Exempt for individual/HUF participants with gross ≤ ₹5 lakh in the FY who furnished PAN — which is most home chefs, until one crosses. |
| **194C** | 1% individual/HUF, 2% others | Contractor payments — riders engaged as contractors, if not routed through 194-O. |
| **194H** | 2% (cut from 5% w.e.f. 1 Oct 2024) | Commission or brokerage. |
| **194J** | 10% (2% for technical services) | Professional and technical fees. |
| **194-I** | 10% land/building, 2% plant | Rent. |
| **194Q** | 0.1% | Purchases above ₹50 lakh from one seller. |
| **192** | slab | Salaries. |
| **206AB** | double the normal rate | Payees who have not filed returns — a lookup obligation, not just a rate. |

Rider classification (contractor under 194C vs e-commerce participant under 194-O) changes
both the rate and who files. **CA question.**

### 10.3 GST the platform owes outside the order flow

- **Reverse charge on imported services (OIDAR).** Cloud and SaaS bought from non-residents —
  GCP, Apple Developer Program, Expo/EAS, Sentry, any foreign tool — attract **18% GST under
  reverse charge**, self-invoiced and paid in cash, then claimable as ITC against the
  platform's 18% output supplies (never against the 5% food line, which carries no ITC).
- **Domestic reverse charge** on legal services from advocates, goods transport agency,
  sponsorship, renting of motor vehicles, and renting of commercial property from an
  unregistered landlord to a registered tenant (added Oct 2024).
- **ITC discipline.** The platform's inputs serve both a no-ITC 5% stream and an 18% stream,
  so credit has to be apportioned under Rule 42/43 rather than claimed wholesale.

### 10.4 Withholding on payments out of India

Payments to non-residents attract **§195 TDS** at treaty or Act rates, with **Form 15CA/15CB**
certification per remittance. Under the India–Australia DTAA, royalties and fees for technical
services are capped at 10–15%; a management or platform charge from Tesserix Pty Ltd to
Zivana Innovations LLP will be tested against that, against transfer-pricing arm's-length
rules (Form 3CEB if the aggregate crosses ₹1 crore), and against permanent-establishment risk
if Australian staff effectively run the India operation. This is the engineering-adjacent half
of #31.

**Equalisation levy** — the 2% levy on non-resident e-commerce operators was withdrawn from
1 Aug 2024 and the 6% levy on online advertising from 1 Apr 2025, so ad spend with Google or
Meta should no longer carry it. Confirm before relying on it.

### 10.5 Payroll and state-level

Provident fund (12% employer, mandatory above 20 employees), ESI (3.25% employer, up to
₹21,000 wages), gratuity accrual, **professional tax** (state-levied — Karnataka and
Maharashtra both apply, a few hundred rupees per employee per month), shops and
establishments registration, and stamp duty on executed agreements.

### 10.6 What the chef owes

Useful for onboarding copy, because chefs will ask.

- **Income tax on their own profit**, with presumptive taxation under **§44AD** available —
  6% of turnover received digitally (8% for cash) deemed as income, no books required, up to
  a ₹2 crore (₹3 crore where cash receipts are under 5%) turnover limit.
- **Credit for the 0.1% §194-O deduction**, claimable against their liability — so the
  statement the platform issues has to show it.
- **No GST registration** while under the threshold and supplying only through the platform
  (§3). If they cross it, or sell off-platform, that changes.

### 10.7 Filing calendar

| Return | Cadence |
|---|---|
| GSTR-1 / GSTR-3B | Monthly, or quarterly under QRMP |
| GSTR-8 (TCS) | Monthly, by the 10th |
| GSTR-9 / 9C | Annual, above the turnover thresholds |
| TDS returns (24Q/26Q/27Q) | Quarterly |
| Form 3CEB (transfer pricing) | Annual, with the return |
| Advance tax | Quarterly |
| ROC / LLP annual filings | Annual |
