# COUNSEL-REVIEW — Fe3dr legal package

> **Status: template legal package, reviewer-ready. NOT legal sign-off. Generated 15 June 2026; entity reconciled to mark8ly parent precedent, 15 June 2026.**

This checklist accompanies the legal documents shipped across the three live surfaces — the
fe3dr.com landing site, the customer app, and the vendor (chef) app. Every legal file carries a
top-of-file code comment ("Template legal content — have counsel review before launch") and uses
a single canonical set of values:

- **Operator:** Tesserix Pty Ltd (ACN 694 070 865, ABN 59 694 070 865), registered in New South Wales, Australia
- **General support / legal email:** support@fe3dr.com
- **India Grievance Officer:** Samyak Rout, grievance@fe3dr.com *(mailbox provisioned and monitored via Cloudflare Email Routing, confirmed 2 Aug 2026; no phone published — see below)*
- **Governing law:** New South Wales, Australia (with non-excludable consumer-protection forum carve-out)
- **Date line:** Last updated: 11 June 2026

The goal is a package counsel can review cheaply in one pass. Each item below is something counsel
must confirm or correct before launch.

---

## ✓ RESOLVED — operating entity confirmed (was the #1 open item)

The earlier "AU-suffix vs India-only" flag was a **false alarm** — it is a deliberate cross-border
structure, not an inconsistency. **Tesserix Pty Ltd (ACN 694 070 865, ABN 59 694 070 865),
registered in New South Wales, Australia, is the genuine parent.** This was confirmed against the
sibling product **mark8ly** (same parent — see
`/mark8ly/apps/onboarding/app/{privacy,terms,legal}`), which carries the identical entity block and
house structure. Home Chef (Fe3dr) and mark8ly are sibling products of one parent.

Home Chef now mirrors the parent's house structure across every live legal surface:

- **Full entity block** — "Home Chef (Fe3dr) is a product of Tesserix Pty Ltd (ACN 694 070 865,
  ABN 59 694 070 865), registered in New South Wales, Australia" — on every operator mention
  (landing + customer app + vendor app). ACN/ABN are now in user-facing copy, matching mark8ly.
- **NSW governing law** on every Terms / Chef & Vendor Agreement / EULA: laws of New South Wales
  without regard to conflict-of-laws; NSW courts have exclusive jurisdiction **subject to any
  non-excludable consumer-protection forum rules in your jurisdiction** (this carve-out preserves
  Indian consumers' rights); 30-day good-faith resolution → mediation under the Rules of the
  Resolution Institute (Australia), with urgent interlocutory relief preserved.
- **APP & DPDP-aligned posture** wherever data-protection law is named: "Tesserix Pty Ltd handles
  personal data in line with the Australian Privacy Principles (APP) and, for our India operations,
  in alignment with India's Digital Personal Data Protection Act, 2023 (DPDP)." Deliberately
  DPDP-*aligned*, not sole-India-framed and not over-claiming GDPR/UK-GDPR/CCPA (see residual (c)).
- **India Grievance Officer** retained: general support stays at `support@fe3dr.com`; a dedicated
  India grievance line `grievance@fe3dr.com` was added (mirroring mark8ly's `dpo@mark8ly.com`).
  *(Earlier revisions of this document called the address `dpo@fe3dr.com`. That was wrong — the
  code has always published `grievance@fe3dr.com` and `dpo@` appears nowhere in app code.
  Corrected 2 Aug 2026, GH #916.)* As of 2 Aug 2026 the mailbox is **provisioned and monitored**
  and the appointee is **named**, so the `TODO(ops)` markers have been removed.

All India-operational specifics that genuinely apply are **kept**: FSSAI food-safety, Indian GST
invoicing, RBI Payment Aggregator refund timelines, and the India grievance officer.

---

## ⚠ RESIDUAL counsel question (narrowed, India food-commerce specific)

Home Chef differs from mark8ly (a **global SaaS marketplace**) in that it issues **Indian GST tax
invoices**, collects via **Razorpay** (an Indian Payment Aggregator), and is subject to **FSSAI**
and the **Consumer Protection (E-Commerce) Rules, 2020**. This narrows — but does not fully close —
the cross-border question. Counsel must confirm:

- [ ] **(a) GST contracting entity.** Whether issuing Indian GST invoices / the GSTIN must belong to
  an India-registered contracting entity, or whether the AU parent can contract cross-border for
  food delivery the way mark8ly does for SaaS.
- [ ] **(b) NSW jurisdiction enforceability.** Whether NSW exclusive jurisdiction is enforceable
  against Indian consumers for a *food* business, or whether the consumer-protection forum carve-out
  (now in every Terms/Agreement/EULA) is sufficient on its own.
- [ ] **(c) Privacy posture breadth.** Whether to broaden the privacy posture to GDPR/CCPA if the
  app ever serves those regions. Currently India-only, so GDPR/UK-GDPR/CCPA claims were **left out
  deliberately** — over-claiming is worse than silence. Revisit before any non-India launch.

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

Entity identifiers are now **KNOWN**: Tesserix Pty Ltd, **ACN 694 070 865**, **ABN 59 694 070 865**,
NSW Australia (confirmed against mark8ly). Removed from the needed list. Still outstanding:

- [ ] **GSTIN** (India operations) — drives GST invoicing and TDS §194-O in the vendor earnings flow
  and Chef & Vendor Agreement; see residual (a) for whether it must sit on an India-registered entity.
- [x] **India Grievance Officer name** + **provisioned `grievance@fe3dr.com` mailbox** — supplied by
  the owner 2 Aug 2026 (GH #916). Officer: **Samyak Rout**; mailbox provisioned and monitored via
  Cloudflare Email Routing. The bracketed name/phone placeholders in landing Privacy §10 and in the
  web SPA's legal pages are gone, and the `TODO(ops)` / `TODO(counsel)` markers with them.
  **No phone is published:** the owner declined to publish a number, so the phone line and the
  `(Mon–Fri, 10:00–18:00 IST)` hours claim hanging off it were *deleted* rather than blanked —
  advertising hours for a channel nobody answers would be a new false statement. Counsel should
  confirm a DPDP §13 appointee contactable by email alone is acceptable.
- [ ] **Registered street address** of the operating entity — only if a regulator requires more than
  "New South Wales, Australia".
- [ ] Confirm the support email **support@fe3dr.com** is monitored and contractually named as the
  general legal contact, alongside **grievance@fe3dr.com** as the India grievance contact.

## Known drift deliberately left

- **Web SPA (`apps/web/**`) — NOT sunset. Re-classified 2 Aug 2026 (GH #916).** This entry used to
  read "sunset web SPA … left untouched on purpose". That is wrong now: `apps/web/SUNSET.md` opens
  "**ACTIVE again (was paused)**" and the app is built and deployed on every `main` commit as of
  2026-07-27, so its React-Router legal pages and the signed-in `MainLayout` footer are **live legal
  copy**. Current status of the old values: the `*@homechef.in` addresses are **gone** (zero hits;
  the pages publish `grievance@fe3dr.com` / `support@fe3dr.com`), and the Grievance Officer
  placeholders were fixed under #916. Still drifting: the **13 May 2026** date line in
  `PrivacyPolicyPage.tsx`, `TermsPage.tsx`, `CookiePolicyPage.tsx` and `RefundPolicyPage.tsx`, which
  does not match the canonical `LEGAL_LAST_UPDATED`. Treat that as an open defect on a live surface,
  not as accepted drift.
- **API host fallback `https://api.homechef.app`** in
  `apps/mobile-customer/hooks/useOrderTrackingWS.ts`. This is an infrastructure host fallback URL,
  not user-facing legal-contact copy, so it was out of scope for the legal consistency sweep. Noted
  here for completeness; it should be migrated to the fe3dr.com host as part of infra cleanup, not
  the legal pass.
- **`hello@fe3dr.com` / `chefs@fe3dr.com`** in `apps/web-landing/lib/site.ts` are the general
  marketing/contact and chef-recruitment addresses, intentionally distinct from the legal/grievance
  `support@fe3dr.com`. Left as-is per the sweep scope.
