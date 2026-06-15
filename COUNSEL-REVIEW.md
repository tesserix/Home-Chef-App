# COUNSEL-REVIEW — Fe3dr legal package

> **Status: template legal package, reviewer-ready. NOT legal sign-off. Generated 15 June 2026.**

This checklist accompanies the legal documents shipped across the three live surfaces — the
fe3dr.com landing site, the customer app, and the vendor (chef) app. Every legal file carries a
top-of-file code comment ("Template legal content — have counsel review before launch") and uses
a single canonical set of values:

- **Operator:** Tesserix Pty Ltd
- **Support / grievance email:** support@fe3dr.com
- **Date line:** Last updated: 11 June 2026

The goal is a package counsel can review cheaply in one pass. Each item below is something counsel
must confirm or correct before launch.

---

## ⚠ TOP PRIORITY — entity vs jurisdiction conflict (must resolve before launch)

The single biggest open question. The operator is named **"Tesserix Pty Ltd"** — an Australian
company suffix. The sunset web SPA (`apps/web/`, not edited — app-only going forward) still carries
the full Australian identifiers: **ACN 694 070 865 / ABN 59 694 070 865, registered in New South
Wales, Australia**. Yet every policy in the live package asserts **India-only operation**: compliance
with India's Digital Personal Data Protection Act 2023 (DPDP), Indian GST invoicing, an Indian-style
Grievance Officer (DPDP §10/§13, IT Intermediary Rules 2021), RBI Payment Aggregator refund
timelines, and FSSAI food-safety law.

This is internally inconsistent and must be reconciled.

- [ ] **(a) Governing law / jurisdiction clause.** The SPA Terms §12 carried a *split* clause (NSW
  Australia for the company; India for platform operations, food safety, payments, tax). The live
  Terms (`apps/web-landing/app/terms/page.tsx` §12) has been softened to India-operations framing
  with a `TODO(counsel)` marker. Counsel must decide on a single, coherent governing-law and
  jurisdiction clause and apply it across landing Terms, the Chef & Vendor Agreement, and the EULA.
- [ ] **(b) DPDP "Data Fiduciary" identity.** Which legal entity is the Data Fiduciary under the
  DPDP Act? An Australian Pty Ltd acting as Data Fiduciary for India-resident Data Principals needs
  confirmation (and likely an India nexus / representative).
- [ ] **(c) GST registration / GSTIN.** Confirm the operating entity's GST registration and GSTIN.
  Indian GST invoicing and TDS (Section 194-O) are asserted in the vendor earnings flow and the
  Chef & Vendor Agreement.
- [ ] **(d) Contracting entity.** Confirm whether an Indian subsidiary or the AU entity actually
  contracts with chefs and customers. This drives every "between you and Tesserix Pty Ltd" line.
- [ ] **(e) Grievance Officer.** Must be an India-resident appointee under DPDP §10/§13 and the IT
  Intermediary Rules 2021. Currently a bracketed `[Grievance Officer Name]` / `[Grievance Officer
  Phone]` placeholder (see landing Privacy §10).

---

## Refund & cancellation vs Razorpay

- [ ] Confirm the **7-working-day** refund window matches the RBI Payment Aggregator Master
  Direction (17 Mar 2020, §8) and current Razorpay settlement reality.
  *Where:* landing Refund policy §4; customer `app/refund.tsx`.
- [ ] Confirm the **original-payment-method** rule and the **escrow / settlement** wording match
  Razorpay's actual flow and the live refund implementation (PROD-READINESS W3 cancel/refund
  endpoints).
- [ ] Confirm the order-stage cancellation matrix (Placed / Accepted / Preparing / Out-for-delivery
  / Delivered / Never-delivered) matches the cancel-button logic actually shipped in the apps.

## Data protection / DPDP

- [ ] Confirm the **data-retention periods** (the SPA asserted: order/payment records per Indian tax
  law, FSSAI traceability for food-safety complaints, 90-day deletion of closed-account data). The
  live policies summarise these without hard numbers — counsel to confirm the numbers before they
  are restored.
  *Where:* landing Privacy §6; customer/vendor `app/privacy.tsx`.
- [ ] Confirm **cross-border transfer** wording (Stripe processing in the US) against DPDP §16.
- [ ] Confirm the **breach-notification** commitment (notify affected users + Data Protection Board
  of India without undue delay).
- [ ] Confirm the **children / 18+** gate is adequate (no parental-consent mechanism exists today).
- [ ] Confirm the **consent + withdrawal** flow and the **15-day Grievance Officer** response
  commitment.

## Marketplace / liability allocation

- [ ] Confirm **intermediary status** under IT Act §79 and the IT Intermediary Rules 2021.
  *Where:* landing Terms §11.
- [ ] Confirm the **chef independent-contractor** characterisation and the **FSSAI / food-safety**
  liability allocation (chef bears food-safety responsibility; platform does not cook/test/inspect).
  *Where:* landing Chef & Vendor Agreement §1, §5, §6; vendor `app/chef-agreement.tsx`.
- [ ] Confirm the **delivery-partner (3PL)** liability allocation. (Per project notes delivery is
  moving to a 3PL/Shadowfax-class model; the policies describe independent delivery partners.)
- [ ] Confirm the **allergen-disclosure** liability wording (chef bears the disclosure duty).
- [ ] Confirm the **review / defamation** wording.
- [ ] Confirm the **liability cap** (platform fee + commission per order) is enforceable and does not
  fall foul of the Consumer Protection Act 2019 / E-Commerce Rules 2020.

## EULA

- [ ] Confirm our own EULA is acceptable to **Apple / Google**, or whether to fall back to **Apple's
  standard Licensed Application End User Licence Agreement**. The current EULA explicitly notes
  Apple's standard EULA "applies as a fallback where this EULA is silent."
  *Where:* landing `app/eula/page.tsx` §4; customer & vendor `app/eula.tsx`.
- [ ] Confirm the **IP-ownership entity** named in the EULA matches the resolved operating entity
  from the TOP PRIORITY item.

## Outstanding placeholders (real values needed before launch)

- [ ] Registered address of the operating entity.
- [ ] CIN (if Indian Pvt Ltd) and/or ACN/ABN (if AU) — a single authoritative entity.
- [ ] GSTIN.
- [ ] Grievance Officer name + phone (currently bracketed `TODO(counsel)` in landing Privacy §10).
- [ ] Confirm the support email **support@fe3dr.com** is monitored and contractually named as the
  grievance/legal contact.

## Known drift deliberately left

- **Sunset web SPA (`apps/web/**`).** The three React-Router legal pages
  (`PrivacyPolicyPage.tsx`, `TermsPage.tsx`, `RefundPolicyPage.tsx`) still carry the old values —
  **13 May 2026** date, **support@homechef.in / grievance@homechef.in / legal@homechef.in** emails,
  and the full **ACN 694 070 865 / ABN 59 694 070 865, NSW Australia** identifiers. These were left
  untouched on purpose: all three web apps are being sunset (the platform is app-only; fe3dr.com is
  landing-only). They are listed here only so counsel is not surprised if they grep the repo.
- **API host fallback `https://api.homechef.app`** in
  `apps/mobile-customer/hooks/useOrderTrackingWS.ts`. This is an infrastructure host fallback URL,
  not user-facing legal-contact copy, so it was out of scope for the legal consistency sweep. Noted
  here for completeness; it should be migrated to the fe3dr.com host as part of infra cleanup, not
  the legal pass.
- **`hello@fe3dr.com` / `chefs@fe3dr.com`** in `apps/web-landing/lib/site.ts` are the general
  marketing/contact and chef-recruitment addresses, intentionally distinct from the legal/grievance
  `support@fe3dr.com`. Left as-is per the sweep scope.
