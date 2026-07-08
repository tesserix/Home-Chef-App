# COUNSEL-BRIEF — Fe3dr launch legal sign-off (#19)

> **What this is:** a counsel-ready delta on top of [`COUNSEL-REVIEW.md`](./COUNSEL-REVIEW.md). It (A) records an
> automated consistency verification of the shipped legal copy, (B) turns the scattered residuals into a numbered
> questionnaire with the exact code location and a researched *engineering note* per item so counsel can confirm or
> correct in one pass, and (C) lists the real-world facts the owner must supply.
>
> **This document is NOT legal advice and NOT legal sign-off.** The "engineering notes / researched positions" are
> background gathered from the code and public sources to *speed* a qualified reviewer — they must be independently
> verified by Indian counsel + a Chartered Accountant (for GST/TDS/TCS) and, for the AU entity, Australian counsel.
> Sign-off remains an external human step; #19 stays **blocked** until counsel signs and the owner-facts below are filled.

Prepared 8 July 2026, verified against `main` (post-#662). Canonical values, per `apps/web-landing/lib/site.ts`:
Operator **Tesserix Pty Ltd** (ACN 694 070 865, ABN 59 694 070 865), NSW Australia · support `support@fe3dr.com` ·
India grievance `dpo@fe3dr.com` · governing law NSW · date line **11 June 2026**.

---

## Part A — Consistency verification (automated, PASS)

Method: grepped every live legal surface (landing `apps/web-landing/**`, customer `apps/mobile-customer/**`, vendor
`apps/mobile-vendor/**`) for the canonical identifiers, old/leaked values, placeholders, and the date line.

| Check | Result | Evidence |
|---|---|---|
| Entity block present & identical on every operator mention | ✅ PASS | Landing via `LEGAL_OPERATOR_FULL` (`site.ts:34`); customer app privacy/terms/refund/eula; vendor app privacy/terms/refund/eula/chef-agreement — all read "Tesserix Pty Ltd (ACN 694 070 865, ABN 59 694 070 865), registered in New South Wales, Australia" |
| ACN / ABN correct everywhere | ✅ PASS | 694 070 865 / 59 694 070 865 in all surfaces |
| Date line consistent | ✅ PASS | `11 June 2026` via `LEGAL_LAST_UPDATED` (`site.ts:39`) + per-file `LAST_UPDATED` consts |
| Canonical emails via a single source | ✅ PASS | `LEGAL_SUPPORT_EMAIL` / `LEGAL_GRIEVANCE_EMAIL` (`site.ts:28,38`); apps reference the same two addresses |
| No leaked legacy values in live surfaces | ✅ PASS | Zero `homechef.in` / `@homechef.app` / `13 May 2026` / `mark8ly` hits in landing or either app |
| Governing law + consumer-forum carve-out present | ✅ PASS | NSW exclusive jurisdiction "subject to any non-excludable consumer-protection forum rules in your jurisdiction" on Terms/EULA/Chef-Agreement |
| DPDP-*aligned* posture (not over-claiming GDPR/CCPA) | ✅ PASS | "APP … and, for our India operations, in alignment with India's DPDP Act, 2023" |
| Refund copy internally consistent (7 working days, original method, RBI PA Master Direction 17 Mar 2020) | ✅ PASS | `apps/web-landing/app/refund/page.tsx:18,25,38,51,52,72,92`; customer `app/refund.tsx` |

**Deliberate, documented drift (not a defect):** the sunset React web SPA (`apps/web/src/features/legal/*`) still carries
old values (`13 May 2026`, `*@homechef.in`) — it is being retired (app-only platform; fe3dr.com is landing-only), so it
was intentionally left untouched. Noted here only so counsel is not surprised by a repo grep. See COUNSEL-REVIEW "Known drift".

**Conclusion:** the "policy bodies drafted and wired into all apps + landing (code-done 2026-06-15)" claim on #19 **holds**.
The only in-code gaps are the owner-fact placeholders in Part C.

---

## Part B — Counsel questionnaire (confirm / correct each)

For each item: **Code asserts** = what the shipped copy currently states (with location) · **Instrument** = the named law
· **Engineering note** = researched background to speed review (verify independently).

### B1 — Cross-border structure & Indian tax (highest priority)

**Q1. GST contracting entity / GSTIN ownership.** *(COUNSEL-REVIEW residual (a))*
- **Code asserts:** the operator on every invoice/agreement is the AU parent, Tesserix Pty Ltd; a platform **GSTIN** placeholder is required for GST invoicing and TDS §194-O in the vendor earnings flow (`apps/api` earnings/statement + Chef & Vendor Agreement).
- **Instrument:** CGST Act §22/§24 (compulsory registration), §52 (TCS by e-commerce operator), §194-O (TDS), place-of-supply rules.
- **Engineering note (verify with CA/counsel):** a marketplace/aggregator that facilitates taxable *food* supplies in India is an **e-commerce operator** and, unlike a turnover-thresholded ordinary supplier, is subject to **compulsory GST registration regardless of turnover** (§24); a non-resident operator typically must **appoint an authorised representative** in India to register. **TCS §52** (operator collects tax on the net value of supplies made by the chefs through the platform, deposited by the 10th of the following month) appears to apply and is **explicitly NOT yet modeled in the code** (see the TCS §52 TODO in the payout model). This points toward needing an **India GST registration** (and quite possibly an India-registered contracting/Payment-Aggregator-facing entity) for the food-marketplace leg — a materially different answer from mark8ly's pure-SaaS posture. **Ask counsel/CA to confirm:** (i) can the AU parent register directly via an authorised representative, or is an India subsidiary required; (ii) whose GSTIN goes on customer tax invoices vs the chef's own GSTIN; (iii) is TCS §52 in-scope, and must it be live before GA. *This is the single most consequential open item — it can gate the flag flip, not just the legal doc.*

**Q2. NSW jurisdiction enforceability against Indian consumers.** *(residual (b))*
- **Code asserts:** NSW law + NSW courts exclusive jurisdiction, **"subject to any non-excludable consumer-protection forum rules in your jurisdiction"** (Terms/EULA/Chef-Agreement).
- **Instrument:** Consumer Protection Act 2019 + Consumer Protection (E-Commerce) Rules 2020.
- **Engineering note:** Indian consumer-protection law confers non-excludable rights and an Indian forum (District Consumer Disputes Redressal Commission) that a foreign exclusive-jurisdiction clause generally cannot oust for consumer disputes. The carve-out is the correct hedge; **confirm it is sufficient on its own**, or whether Indian courts should be named for Indian-consumer disputes specifically.

**Q3. Privacy posture breadth.** *(residual (c))* — India-only (DPDP/APP), GDPR/CCPA deliberately omitted. **Confirm** this is right for an India-only launch and flag to revisit before any non-India launch. *(Low risk; already the doc's stance.)*

### B2 — Refund & cancellation vs Razorpay / RBI

- **Q4.** Confirm the **7-working-day** refund window and **original-payment-method** rule match the current RBI Payment Aggregator Master Direction and Razorpay's live settlement reality. *Code:* `refund/page.tsx:18,51,52`; customer `refund.tsx:32`.
- **Q5.** Confirm the **order-stage cancellation matrix** (Placed / Accepted / Preparing / Out-for-delivery / Delivered / Never-delivered) matches the cancel-button logic actually shipped. *Code:* `refund/page.tsx:25` + the app cancel flows. *(Cross-check against the escrow/refund endpoints hardened this cycle — see #392/#457/#549.)*
- **Q6.** Confirm the **catering** cancellation tiers (>72h 100% / 48–72h 50% / <48h 0%) and **force-majeure = full refund** wording are acceptable. *Code:* `refund/page.tsx:38,65`.

### B3 — Data protection / DPDP

- **Q7.** **Named Grievance Officer + 15-day response.** DPDP §13 requires a **real, India-resident** grievance officer; the copy commits to a 15-day grievance response. *Owner fact needed — see C2.* *Code:* landing Privacy §10 (`privacy/page.tsx:110-113`), customer/vendor `privacy.tsx`.
- **Q8.** **Retention periods.** Live policies summarise retention (tax records per Indian law; FSSAI traceability; 90-day deletion of closed-account data) **without hard numbers**. Confirm the numbers before they are restored to copy. *Code:* landing Privacy §6.
- **Q9.** **Cross-border transfer (DPDP §16).** Stripe processing in the US. DPDP uses a blacklist (restricted-country) model; confirm the US is permitted and the transfer wording is adequate.
- **Q10.** **Breach notification** to affected users + the Data Protection Board of India without undue delay — confirm wording.
- **Q11.** **Children / 18+ gate.** No parental-consent mechanism exists; the app relies on a hard 18+ gate. Confirm this is adequate under DPDP §9 (verifiable parental consent for minors) given a pure adult-gating approach.

### B4 — Marketplace / liability allocation

- **Q12.** **Intermediary status** (IT Act §79 safe harbour + IT (Intermediary Guidelines) Rules 2021 due-diligence + grievance mechanism). *Code:* landing Terms §11.
- **Q13.** **Chef independent-contractor** characterisation + **FSSAI food-safety** allocation (chef bears food-safety; platform does not cook/test/inspect). *Code:* Chef & Vendor Agreement §1/§5/§6; vendor `chef-agreement.tsx`.
- **Q14.** **Delivery partner (3PL)** liability allocation (independent delivery partners; delivery is moving to a Shadowfax/Borzo-class 3PL).
- **Q15.** **Allergen disclosure** (chef bears the disclosure duty) and **review/defamation** wording.
- **Q16.** **Liability cap** (platform fee + commission per order). **Engineering note:** consumer liability caps are vulnerable under the CPA 2019 "unfair contract" provisions — confirm the cap is enforceable against Indian consumers and does not fall foul of CPA 2019 / E-Commerce Rules 2020.

### B5 — EULA

- **Q17.** Confirm the custom **EULA is acceptable to Apple / Google**, or fall back to Apple's standard LAEULA (the current EULA already notes Apple's standard EULA "applies as a fallback where this EULA is silent"). *Code:* landing `eula/page.tsx` §4; customer/vendor `eula.tsx`.
- **Q18.** Confirm the **IP-ownership entity** named in the EULA (Tesserix Pty Ltd) matches the entity resolved in Q1. *(If Q1 lands on an India entity for the food leg, the IP-owner may still be the AU parent — counsel to reconcile.)*

---

## Part C — Owner facts required before launch (only the owner can supply these)

None of these can be derived from the code or invented — they gate the placeholders currently in the copy.

- [ ] **C1 — Platform GSTIN** (15-char). Drives GST invoices + TDS §194-O. **Depends on Q1** (which entity holds it). Not in user-facing copy yet; needed in the backend invoicing/earnings flow. *(Note: the vendor app's GSTIN field is the **chef's own** GSTIN at onboarding — a different value.)*
- [ ] **C2 — India Grievance Officer: real name + phone.** DPDP §13 requires a named India-resident appointee. Fills `[Grievance Officer Name]` / `[Grievance Officer Phone]` at `apps/web-landing/app/privacy/page.tsx:112-113` (TODO(counsel) at :110).
- [ ] **C3 — Provision `dpo@fe3dr.com` mailbox**, routed to the C2 appointee, monitored with the 15-day/30-day SLA. TODO(ops) at `site.ts:37`, customer `privacy.tsx:72`, vendor `privacy.tsx:61`. *(You are creating the Cloudflare alias — see the email list I sent; `support@` and `dpo@` are the two launch-critical ones.)*
- [ ] **C4 — Confirm `support@fe3dr.com` is monitored** and contractually named as the general legal contact.
- [ ] **C5 — Registered street address** of the operating entity — only if a regulator (FSSAI / GST / consumer) requires more than "New South Wales, Australia".

---

## Part D — Sign-off checklist (counsel ticks; mirrors #19)

- [ ] Counsel sign-off on Privacy / Terms / Refund / Chef & Vendor Agreement / EULA (Part B answered).
- [ ] Residuals resolved: **Q1** GST entity/GSTIN, **Q2** NSW enforceability (the two cross-border items).
- [ ] Owner facts filled: **C1** GSTIN, **C2** Grievance Officer, **C3** dpo mailbox, **C4** support mailbox, **C5** address (if required).
- [ ] TCS §52 in/out-of-scope decision recorded (Q1(iii)) — flag if it gates the escrow flag flip.

**Sources (background research, verify independently):**
[GST framework for e-commerce operators](https://nexpective.in/gst-framework-for-ecommerce-operators/) ·
[GST Council e-commerce FAQ (PDF)](https://gstcouncil.gov.in/sites/default/files/2024-02/faq-e-commerc.pdf) ·
[Section 52 TCS guide](https://gstgyaan.com/section-52-collection-of-tax-at-source-tcs-under-gst) ·
[Compulsory GST registration §24](https://www.taxbuddy.com/blog/compulsory-registration-gst-section-24)
