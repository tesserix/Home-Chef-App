# Audit Findings — Rolled-Up Index

Per-finding detail lives in `findings/<category>.md`. This index is for sorting, filtering, and dashboard views.

Generated: 2026-05-13
Total findings: 2417

## Summary by severity

| Severity | Count |
|---|---|
| P0 | 292 |
| P1 | 545 |
| P2 | 827 |
| P3 | 753 |

## Summary by lens

| Lens | Count |
|---|---|
| technical-writer | 949 |
| legal | 617  (all flagged "needs lawyer review") |
| business-analyst | 378 |
| brand-voice | 473 |

## Summary by category

| Category | TW | Legal | BA | BV | Total | P0 count |
|---|---|---|---|---|---|---|
| legal | 48 | 60 | 23 | 23 | 154 | 25 |
| marketing | 25 | 35 | 22 | 24 | 106 | 30 |
| auth-onboarding | 192 | 106 | 60 | 60 | 418 | 55 |
| core-ux | 200 | 100 | 70 | 97 | 467 | 55 |
| errors-empty | 165 | 88 | 70 | 78 | 401 | 35 |
| transactional | 160 | 154 | 57 | 70 | 441 | 56 |
| microcopy | 120 | 47 | 46 | 98 | 311 | 22 |
| help | 12 | 12 | 10 | 8 | 42 | 4 |
| seo-meta | 27 | 15 | 20 | 15 | 77 | 10 |
| **Total** | 949 | 617 | 378 | 473 | **2417** | **292** |

## Top P0 findings — cross-category executive summary

Below are the major P0 clusters identified across the 292 P0 findings. Each cluster pulls the most exposed surfaces from the per-category files. Numbers in parentheses indicate the count of P0 findings that match cluster keywords (some findings appear in more than one cluster).

### 1. AI-slop / fake unverified metrics (26 P0)

Marketing landing, hero, trust badges and conversion entry pages assert hardcoded social-proof metrics ("500+ Home Chefs Near You", "4.8 Average Rating", "30-45 min Delivery") and a fabricated testimonial attributed to "Sarah M., Happy Customer" without substantiation, geographic qualifier, methodology, or sample size. These claims appear in landing hero (`web-mkt-landing-hero-badge`), trust badge row (`web-mkt-landing-trust-badges`), registration entry (`web-mkt-register-benefits`), login carousel (`web-mkt-login-testimonial`), and vendor signup (`vp-mkt-register-benefits-list`). Under DPDP §5 / Consumer Protection Act 2019 §2(47), unsubstantiated commercial claims are misleading-advertising violations. Remediation: either gate every claim behind live data + "as of {date}" or remove. Highest-risk surfaces: the testimonial (fabricated identity) and "Zero commission first month" promise (binding financial commitment with no contract).

### 2. Brand identity drift — Fe3dr / HomeChef / "HomeChef Delivery" / "Fe3dr by HomeChef" (35 P0)

The platform surfaces 3-4 different brand names across customer web, vendor portal, delivery portal, mobile delivery app, and transactional emails/pushes. Examples: web app says "HomeChef" in support chat, delivery-portal says "Fe3dr" in support email + footer, mobile-delivery `Info.plist` says "HomeChef Delivery", transactional email footer says "Fe3dr by HomeChef" compound, welcome push says "HomeChef" while welcome email says "Fe3dr". Hottest surfaces: `api-email-base-footer`, `api-push-user-welcome`, `api-email-account-reminder`, `md-meta-001` (iOS app name), `web-mkt-layout-footer`, `vp-legal-policy-tos`, `mv-settings-delete-contact-email` (homechef.app vs fe3dr.com support email). One driver hitting both web and mobile cannot tell it is the same company. P0 brand-trust hit on every primary entry surface.

### 3. DPDP Act 2023 consent infrastructure missing across signup and data-collection layer (41 P0)

The entire signup flow (`web-auth-register-form-fields`, `web-auth-register-submit`, `web-auth-register-heading`, `vp-legal-docs-banner-body`, `vp-legal-docs-aadhaar`, `mv-settings-delete-confirm-body`, mobile onboarding equivalents) collects personal data (name, email, password, phone, Aadhaar, PAN, bank, location) with no DPDP §6 consent notice at the data-collection moment, no granular opt-in (bundled "by signing up" clause covers T&C + Privacy in one act), no children's-data screening / age gate, no grievance-officer contact disclosed, no rights notice (access, correction, erasure, nomination, grievance redressal), and no third-party-sharing disclosure for OAuth. This is a foundational compliance gap — not a copy fix. Remediation requires a Privacy Notice / DPN component reused across every data-collection surface.

### 4. FSSAI compliance gaps — Optional/Required contradiction, license number missing, allergen disclosure (19 P0)

FSSAI license is labeled "(Optional)" during chef onboarding (`vp-legal-docs-fssai`) but "Required" on the profile-level `DOCUMENT_TYPES` map (`vp-legal-profile-doc-types`) — internally inconsistent contract. The 7-item food-safety self-attestation (`vp-legal-policies-compliance-items`) is load-bearing for FSSAI compliance but uses vague language ("I maintain proper hygiene") that is not measurable. The customer-facing menu (`vp-ux-menu-form-title`, `web-ux-chef-menu`, `vp-ux-menu-form-fields`) collects allergens as free-text with no standard allergen list and no required-disclosure enforcement — violates FSSAI Labelling and Display Regulations 2020. Landing page (`web-mkt-landing-why-choose`) makes a "verified for food safety" claim without showing FSSAI license number, inspection date, or methodology. Religious certification claims (Halal, Kosher, Jain) appear as dietary tags with no certification-required disclosure (`vp-ux-menu-form-dietary-tags`).

### 5. Routing bugs — wrong-audience email/push, duplicate sends (14 P0)

Transactional layer has multiple routing defects that send the wrong copy to the wrong recipient. `api-email-delivery-assigned` returns driver-oriented copy ("You've been assigned order #...") but is dispatched to the CUSTOMER. `api-push-order-update-customer-deeplink` sends the customer TWO pushes for the same status change with different copy (push-workers queue group). `api-email-account-reminder` and `api-email-support-created`/`api-email-support-update` bypass the `emailBase` wrapper so recipients get unbranded inline HTML with no footer, no privacy/terms links, no grievance-officer contact. These are functional bugs masquerading as content issues — fix requires code change, not just copy change.

### 6. Dead conversion paths — broken links at the conversion moment (6 P0)

`/become-chef` route is referenced from landing hero CTA (`web-mkt-hero-how-it-works-cta`, `web-mkt-landing-become-chef`) but does not exist — clicking 404s. `delivery-portal-register-redirect` immediately redirects, so the visible "Register" entry point dead-ends. Mobile-delivery onboarding ToS link (`md-leg-001`) and Privacy Policy link (`md-leg-002`) are styled as tappable but have "no navigation wired in code". Signup page (`web-legal-register-terms`) links to "Terms of Service" and "Privacy Policy" but no such pages exist in `apps/web` — both 404. Every one of these is a conversion-funnel kill on the highest-intent surface.

### 7. "Saved locally" and "Coming soon" features collecting real data (15 P0)

Vendor banking/payout warning (`vp-legal-kitchen-payout-warning`) says "saved locally for now" — factually misleading since the data flows to the API. Settings danger zone (`vp-legal-settings-danger`) shows a "Demo mode" deactivation toast that ships in production builds. Marketing surfaces continue to advertise "Coming soon" features that are quietly already live or quietly never shipping. Customer-facing forms collect Aadhaar, PAN, bank account, IFSC, and location while showing copy that suggests the data is not yet leaving the device — a DPDP §5 purpose-limitation violation if the data is in fact persisted server-side.

### 8. Order status taxonomy fragmentation — 3+ label maps in API and frontends (33 P0)

Customer screens, admin screens, vendor portal, and driver app each render a different label map for the same underlying order/delivery status enum. Examples: customer detail uses "Ready for Pickup" / "On the Way"; `OrderCard` uses "Ready" / "Picked Up"; web orders enum (`web-ux-orders-status-labels`) exposes "Picked Up" and "On the Way" as separate states; admin (`ap-delivery-statuses`) uses 10 title-case states ("At Pickup", "At Dropoff", "In Transit"); driver action button reads "Mark as Picked Up/In Transit/Delivered" (`dp-ux-active-mark-as`); driver row labels are ALL CAPS ("PICKUP"/"DROPOFF", `dp-ux-delivery-pickup-label`); "Drop-off" and "Dropoff" both appear in the same delivery view (`md-core-079`). Three+ label maps in the API + per-app overrides. Customer who refreshes between two screens sees a different status word for the same state.

### 9. Aadhaar / PAN collection without UIDAI offline-eKYC or DPDP-compliant masking (13 P0)

Aadhaar and PAN are collected across vendor onboarding (`vp-legal-docs-aadhaar`, `vp-legal-docs-pan`), driver onboarding (`dp-auth-step3-doc-labels`, `md-onb-042`), and mobile vendor docs slot (`mv-onb-docs-id-slot`). The flow uses free-form text/image upload — not UIDAI offline-eKYC, not Aadhaar Vault. Helper text is vague ("For identity verification") with no purpose limitation, no retention period, no masking statement, no sharing-with-payment-gateway disclosure (despite Stripe Connect KYC). Aadhaar Act §29 and the Aadhaar (Targeted Delivery) Regulations require offline-eKYC or licensed Aadhaar Authentication User Agency status for any e-KYC use. As shipped, the platform is collecting Aadhaar through an unlicensed flow.

### 10. Missing legal disclosures at checkout/order/payout — grievance officer, refund timeline, GST invoice (36 P0)

Checkout (`web-ux-checkout-payment-section`, `web-legal-checkout-terms-note`) collects payment without DPDP §6 consent for payment-gateway data sharing, with no refund/cancellation/GST disclosure inline. Cancellation email (`api-email-order-status-cancelled`) omits refund timeline and who-to-contact — a P0 transactional gap for a paid order. Refund-constraint error strings (`api-error-payment-refund`) expose internal rules to the user but never communicate refund policy. Across emails, push, and admin/portal surfaces there is no published grievance-officer contact, no DPDP §13 escalation path, no GST invoice line on order receipts (`vp-legal-policy-tos` for commission/payout disclosure also missing). Support touchpoints (`web-help-otto-empty`, `dp-help-status-contact`) point to an AI bot or a single support email with no SLA — does not satisfy Consumer Protection (E-Commerce) Rules 2020 §5(3) grievance-redressal requirements.

## Index of all findings (sortable)

One row per finding. Sorted by category, then severity (P0 → P3), then finding_id. Paginated by category to keep each table manageable; jump via the per-category anchors below.

Jump to: [legal](#index--legal) · [marketing](#index--marketing) · [auth-onboarding](#index--auth-onboarding) · [core-ux](#index--core-ux) · [errors-empty](#index--errors-empty) · [transactional](#index--transactional) · [microcopy](#index--microcopy) · [help](#index--help) · [seo-meta](#index--seo-meta)

### Index — legal

154 findings. Detail: [findings/legal.md](findings/legal.md)

| finding_id | surface_id | category | lens | severity | issue (≤80 chars) | detail link |
|---|---|---|---|---|---|---|
| BA-001 | vp-legal-kitchen-payout-warning | legal | business-analyst | P0 | "Saved locally" on payout bank-detail form is a trust-destroying production leak | [findings/legal.md](findings/legal.md) |
| BA-002 | vp-legal-settings-danger | legal | business-analyst | P0 | "Demo mode" toast on Deactivate Account button leaks placeholder copy to produc… | [findings/legal.md](findings/legal.md) |
| BA-003 | mv-onb-policies-terms-body | legal | business-analyst | P0 | Mobile-vendor terms use brand name "HomeChef" while web onboarding uses "Fe3dr"… | [findings/legal.md](findings/legal.md) |
| BV-001 | api-email-base-footer | legal | brand-voice | P0 | Brand-name drift in legal footer of every transactional email — 'Fe3dr by HomeC… | [findings/legal.md](findings/legal.md) |
| BV-002 | mv-onb-policies-terms-body | legal | brand-voice | P0 | Mobile vendor onboarding uses 'HomeChef' as brand name and links to 'homechef.i… | [findings/legal.md](findings/legal.md) |
| BV-003 | mv-settings-delete-contact-email | legal | brand-voice | P0 | Account-deletion support email uses 'support@homechef.app' — third brand-name v… | [findings/legal.md](findings/legal.md) |
| BV-004 | vp-legal-register-footer | legal | brand-voice | P0 | Chef sees 'Vendor Terms' on register page but 'Terms of Service' on login page… | [findings/legal.md](findings/legal.md) |
| BV-005 | vp-legal-policy-tos | legal | brand-voice | P0 | Onboarding step uses 'Fe3dr Platform Terms of Service' — a third name for the s… | [findings/legal.md](findings/legal.md) |
| LEG-001 | web-legal-register-terms | legal | legal | P0 | Signup links to 'Terms of Service' and 'Privacy Policy' but no such pages exist… | [findings/legal.md](findings/legal.md) |
| LEG-002 | web-legal-register-terms | legal | legal | P0 | Bundled consent — single 'by signing up' clause covers both T&C and Privacy in… | [findings/legal.md](findings/legal.md) |
| LEG-004 | md-leg-001 | legal | legal | P0 | Driver onboarding shows 'Terms of Service' link but inventory notes 'no navigat… | [findings/legal.md](findings/legal.md) |
| LEG-005 | md-leg-002 | legal | legal | P0 | Driver onboarding shows 'Privacy Policy' link with no navigation wired — driver… | [findings/legal.md](findings/legal.md) |
| LEG-006 | vp-legal-docs-fssai | legal | legal | P0 | FSSAI license labeled '(Optional)' during chef onboarding — but FSSAI registrat… | [findings/legal.md](findings/legal.md) |
| LEG-007 | vp-legal-profile-doc-types | legal | legal | P0 | FSSAI labeled 'Required' on Profile DOCUMENT_TYPES but 'Optional' on onboarding… | [findings/legal.md](findings/legal.md) |
| LEG-009 | vp-legal-policies-compliance-items | legal | legal | P0 | 7-item food-safety self-attestation list is load-bearing for FSSAI compliance,… | [findings/legal.md](findings/legal.md) |
| LEG-013 | vp-legal-policy-tos | legal | legal | P0 | Vendor TOS clause commits chef to 'commission' and 'weekly payouts' without dis… | [findings/legal.md](findings/legal.md) |
| LEG-017 | vp-legal-docs-banner-body | legal | legal | P0 | Claim 'Your documents are encrypted and stored securely' is a content claim tha… | [findings/legal.md](findings/legal.md) |
| LEG-018 | vp-legal-docs-banner-body | legal | legal | P0 | Claim 'They are only used for verification' is a purpose-limitation promise but… | [findings/legal.md](findings/legal.md) |
| LEG-019 | vp-legal-docs-aadhaar | legal | legal | P0 | Aadhaar Card collected with only 'For identity verification' framing — Aadhaar… | [findings/legal.md](findings/legal.md) |
| LEG-025 | vp-legal-kitchen-payout-warning | legal | legal | P0 | 'Payout integration coming soon. These details are saved locally for now' — col… | [findings/legal.md](findings/legal.md) |
| LEG-029 | ap-chefs-reject-reason | legal | legal | P0 | Hardcoded rejection reason 'Rejected by admin' sent to chef with no specific re… | [findings/legal.md](findings/legal.md) |
| LEG-038 | vp-legal-settings-danger | legal | legal | P0 | 'Account deactivation is not available in demo mode' — production-shipped strin… | [findings/legal.md](findings/legal.md) |
| LEG-040 | mv-settings-delete-confirm-body | legal | legal | P0 | 'To delete your account, please contact our support team' — under DPDP §12, del… | [findings/legal.md](findings/legal.md) |
| LEG-046 | vp-legal-register-footer | legal | legal | P0 | 'You agree to Fe3dr's Vendor Terms' — but inventory notes 'no link target' for… | [findings/legal.md](findings/legal.md) |
| TW-021 | vp-legal-kitchen-payout-warning | legal | technical-writer | P0 | Banking-data collection warning says 'saved locally for now' — factually mislea… | [findings/legal.md](findings/legal.md) |
| BA-004 | api-email-base-footer | legal | business-analyst | P1 | Email footer uses hybrid brand "Fe3dr by HomeChef" — customers receive an ident… | [findings/legal.md](findings/legal.md) |
| BA-005 | vp-legal-docs-aadhaar | legal | business-analyst | P1 | Aadhaar collection described only as "For identity verification" — no trust fra… | [findings/legal.md](findings/legal.md) |
| BA-006 | vp-legal-profile-doc-types | legal | business-analyst | P1 | FSSAI License labelled "Required" on ProfilePage but "Optional" during onboardi… | [findings/legal.md](findings/legal.md) |
| BA-007 | web-legal-checkout-terms-note | legal | business-analyst | P1 | Checkout terms acknowledgment has no hyperlink to Terms of Service — legal cons… | [findings/legal.md](findings/legal.md) |
| BA-008 | vp-legal-policy-tos | legal | business-analyst | P1 | TOS acceptance checkbox embeds commission disclosure as inline prose with no li… | [findings/legal.md](findings/legal.md) |
| BA-009 | vp-legal-policy-cancellation | legal | business-analyst | P1 | Cancellation policy agreement mentions rating/visibility penalties as consequen… | [findings/legal.md](findings/legal.md) |
| BA-010 | vp-legal-docs-banner-body | legal | business-analyst | P1 | "Encrypted and stored securely" is an unqualified trust claim that cannot be su… | [findings/legal.md](findings/legal.md) |
| BA-011 | ap-chefs-reject-reason | legal | business-analyst | P1 | Hardcoded rejection reason "Rejected by admin" is sent to chef via API — vague… | [findings/legal.md](findings/legal.md) |
| BA-012 | mv-settings-delete-contact-email | legal | business-analyst | P1 | Account deletion flow sends mobile chef to email support@homechef.app — domain… | [findings/legal.md](findings/legal.md) |
| BV-006 | md-leg-001 | legal | brand-voice | P1 | Driver onboarding review screen shows only 'Terms of Service' / 'Privacy Policy… | [findings/legal.md](findings/legal.md) |
| BV-007 | mv-onb-policies-checkbox | legal | brand-voice | P1 | Generic 'terms and conditions' label — fifth document-name variant. Voice princ… | [findings/legal.md](findings/legal.md) |
| BV-008 | mv-onb-policies-terms-body | legal | brand-voice | P1 | Legal block addresses chef in third person ('vendor', 'accounts') instead of 'y… | [findings/legal.md](findings/legal.md) |
| BV-010 | vp-legal-settings-danger | legal | brand-voice | P1 | Toast leaks dev-environment language to chef in production: 'Account deactivati… | [findings/legal.md](findings/legal.md) |
| BV-011 | vp-legal-kitchen-payout-warning | legal | brand-voice | P1 | Banking-data collection helper text leaks implementation state and uses legally… | [findings/legal.md](findings/legal.md) |
| BV-015 | vp-legal-profile-doc-types | legal | brand-voice | P1 | Same document labeled inconsistently across two surfaces of the same vendor-por… | [findings/legal.md](findings/legal.md) |
| BV-017 | ap-chefs-reject-reason | legal | brand-voice | P1 | Hardcoded rejection reason 'Rejected by admin' is sent to the chef as legal-gra… | [findings/legal.md](findings/legal.md) |
| LEG-003 | web-legal-checkout-terms-note | legal | legal | P1 | Order placement references T&C only, not Refund Policy or Cancellation Policy —… | [findings/legal.md](findings/legal.md) |
| LEG-008 | vp-legal-docs-food-safety | legal | legal | P1 | Food Safety Training Certificate marked optional — FoSTaC training is mandated… | [findings/legal.md](findings/legal.md) |
| LEG-010 | vp-legal-policies-compliance-confirm | legal | legal | P1 | Single tick-box attestation for 7 distinct compliance items — combined consent… | [findings/legal.md](findings/legal.md) |
| LEG-011 | vp-legal-policy-hygiene | legal | legal | P1 | Hygiene policy commits chef to 'periodic kitchen checks' by Fe3dr — but does no… | [findings/legal.md](findings/legal.md) |
| LEG-012 | vp-legal-policy-cancellation | legal | legal | P1 | 'Repeated late cancellations may affect my kitchen rating and visibility' — pen… | [findings/legal.md](findings/legal.md) |
| LEG-014 | mv-onb-policies-terms-body | legal | legal | P1 | Vendor terms reference URL 'homechef.in/vendor-terms' but inventory notes URL i… | [findings/legal.md](findings/legal.md) |
| LEG-015 | mv-onb-policies-terms-body | legal | legal | P1 | Mobile vendor terms claim platform may 'suspend accounts that receive repeated… | [findings/legal.md](findings/legal.md) |
| LEG-016 | mv-onb-policies-checkbox | legal | legal | P1 | Single checkbox 'I accept the terms and conditions' bundles entire vendor agree… | [findings/legal.md](findings/legal.md) |
| LEG-020 | vp-legal-docs-aadhaar | legal | legal | P1 | Aadhaar collection alongside PAN and bank details — no separate, prominent Aadh… | [findings/legal.md](findings/legal.md) |
| LEG-021 | vp-legal-docs-pan | legal | legal | P1 | PAN Card framed as 'Required for tax purposes and payouts' but no disclosure of… | [findings/legal.md](findings/legal.md) |
| LEG-022 | vp-legal-docs-banner-title | legal | legal | P1 | 'Documents help us verify your kitchen and enable payouts' — does not identify… | [findings/legal.md](findings/legal.md) |
| LEG-024 | vp-legal-docs-cheque | legal | legal | P1 | Cancelled cheque collected with no disclosure of how bank details are stored, w… | [findings/legal.md](findings/legal.md) |
| LEG-026 | vp-legal-settings-stripe | legal | legal | P1 | Stripe Connect block routes chef KYC through Stripe's hosted onboarding but doe… | [findings/legal.md](findings/legal.md) |
| LEG-027 | vp-legal-settings-stripe | legal | legal | P1 | Gateway switch text 'Make Stripe My Primary Gateway / Switch Back to Razorpay'… | [findings/legal.md](findings/legal.md) |
| LEG-028 | vp-legal-policy-tos | legal | legal | P1 | 'Payouts will be processed weekly' — RBI PA Master Direction defines specific s… | [findings/legal.md](findings/legal.md) |
| LEG-032 | ap-secsettings-2fa-exempt | legal | legal | P1 | 2FA-exempt email list defines a carve-out for service accounts — without policy… | [findings/legal.md](findings/legal.md) |
| LEG-039 | vp-legal-settings-danger | legal | legal | P1 | 'Deactivating your account will hide your kitchen from customers and pause all… | [findings/legal.md](findings/legal.md) |
| LEG-041 | mv-settings-delete-contact-email | legal | legal | P1 | Support email 'support@homechef.app' — inconsistent with brand 'Fe3dr' (see LEG… | [findings/legal.md](findings/legal.md) |
| LEG-042 | api-email-base-footer | legal | legal | P1 | Email footer says 'Fe3dr by HomeChef' — brand identity inconsistency across pro… | [findings/legal.md](findings/legal.md) |
| LEG-043 | api-email-base-footer | legal | legal | P1 | Footer links to '/privacy' and '/terms' hardcoded — same risk as LEG-001 (pages… | [findings/legal.md](findings/legal.md) |
| LEG-045 | vp-legal-login-footer | legal | legal | P1 | Sign-in screen footer 'By continuing, you agree to Fe3dr's Terms of Service and… | [findings/legal.md](findings/legal.md) |
| LEG-049 | mv-onb-policies-cancel-option-no | legal | legal | P1 | Cancellation option 'No cancellations after order accepted' — may conflict with… | [findings/legal.md](findings/legal.md) |
| LEG-057 | web-legal-checkout-terms-note | legal | legal | P1 | Missing GST disclosure at checkout — under CGST Act and standard e-commerce pra… | [findings/legal.md](findings/legal.md) |
| LEG-058 | vp-legal-policies-compliance-items | legal | legal | P1 | 'label allergens and ingredients' is item #7 in compliance checklist but no enf… | [findings/legal.md](findings/legal.md) |
| TW-007 | vp-legal-docs-banner-body | legal | technical-writer | P1 | Two sentences, 26 words total — exceeds vendor-facing max of 20 words per sente… | [findings/legal.md](findings/legal.md) |
| TW-018 | vp-legal-policy-hygiene | legal | technical-writer | P1 | 52-word policy block uses formal first-person legal voice ('I commit to maintai… | [findings/legal.md](findings/legal.md) |
| TW-019 | vp-legal-policy-cancellation | legal | technical-writer | P1 | 56-word block in formal 'I' voice. Buries operational consequences in legal pro… | [findings/legal.md](findings/legal.md) |
| TW-020 | vp-legal-policy-tos | legal | technical-writer | P1 | 46-word TOS block in 'I' voice commits chef to commission + weekly payouts. Two… | [findings/legal.md](findings/legal.md) |
| TW-022 | vp-legal-profile-doc-types | legal | technical-writer | P1 | FSSAI marked 'Required' here but 'Optional' on onboarding (vp-legal-docs-fssai)… | [findings/legal.md](findings/legal.md) |
| TW-025 | vp-legal-settings-danger | legal | technical-writer | P1 | 'Account deactivation is not available in demo mode' shipped to vendor portal s… | [findings/legal.md](findings/legal.md) |
| TW-030 | ap-chefs-reject-reason | legal | technical-writer | P1 | 'Rejected by admin' is the default reason sent to a chef when no specific reaso… | [findings/legal.md](findings/legal.md) |
| TW-045 | mv-onb-policies-terms-body | legal | technical-writer | P1 | 60-word block with two very long sentences (one is ~38 words). Sec 5 says max 2… | [findings/legal.md](findings/legal.md) |
| TW-048 | api-email-base-footer | legal | technical-writer | P1 | 'Fe3dr by HomeChef · Authentic homemade food, delivered · fe3dr.com · Privacy ·… | [findings/legal.md](findings/legal.md) |
| BA-013 | vp-legal-register-footer | legal | business-analyst | P2 | "Vendor Terms" linked in chef registration footer points to no destination — de… | [findings/legal.md](findings/legal.md) |
| BA-014 | vp-legal-policies-compliance-confirm | legal | business-analyst | P2 | Kitchen Compliance Checklist confirmation is a silent passive attestation — no… | [findings/legal.md](findings/legal.md) |
| BA-015 | vp-legal-settings-stripe | legal | business-analyst | P2 | "Make Stripe My Primary Gateway" and "Switch Back to Razorpay" are high-stakes… | [findings/legal.md](findings/legal.md) |
| BA-016 | md-leg-001 | legal | business-analyst | P2 | Driver onboarding Terms of Service and Privacy Policy are styled as links but h… | [findings/legal.md](findings/legal.md) |
| BA-017 | web-mkt-brand-copyright | legal | business-analyst | P2 | Web footer copyright reads "Fe3dr" while email footer reads "Fe3dr by HomeChef"… | [findings/legal.md](findings/legal.md) |
| BA-018 | vp-legal-login-footer | legal | business-analyst | P2 | Chef portal login-page legal footer has no clickable links — terms and privacy… | [findings/legal.md](findings/legal.md) |
| BA-022 | mv-onb-policies-checkbox | legal | business-analyst | P2 | Mobile-vendor policies checkbox reads "I accept the terms and conditions" — gen… | [findings/legal.md](findings/legal.md) |
| BA-023 | vp-legal-policy-hygiene | legal | business-analyst | P2 | "Fe3dr may conduct periodic kitchen checks" buried inside the hygiene policy ch… | [findings/legal.md](findings/legal.md) |
| BV-009 | vp-legal-docs-banner-title | legal | brand-voice | P2 | Mixed voice: 'Documents help us verify your kitchen and enable payouts' — uses… | [findings/legal.md](findings/legal.md) |
| BV-012 | vp-legal-policies-compliance-title | legal | brand-voice | P2 | Title Case in legal heading 'Kitchen Compliance Checklist' violates STYLE-GUIDE… | [findings/legal.md](findings/legal.md) |
| BV-013 | ap-auth-footer-legal | legal | brand-voice | P2 | Title Case in admin-portal footer 'Fe3dr Administration Portal · Internal Use O… | [findings/legal.md](findings/legal.md) |
| BV-018 | vp-legal-policy-hygiene | legal | brand-voice | P2 | Hygiene policy mixes warm self-pledge tone ('I commit to maintaining a clean an… | [findings/legal.md](findings/legal.md) |
| BV-020 | api-email-base-footer | legal | brand-voice | P2 | Tagline 'Authentic homemade food, delivered' uses 'authentic' — adjacent to ban… | [findings/legal.md](findings/legal.md) |
| BV-023 | vp-legal-docs-banner-body | legal | brand-voice | P2 | Privacy promise is grammatically passive and over-stated for brand voice. 'Your… | [findings/legal.md](findings/legal.md) |
| LEG-023 | vp-legal-docs-identity-title | legal | legal | P2 | 'These are required to verify your identity and set up payouts' — broad single-… | [findings/legal.md](findings/legal.md) |
| LEG-030 | ap-approvaldetail-notes-helper | legal | legal | P2 | 'These notes will be recorded in the history and visible to other admins' — doe… | [findings/legal.md](findings/legal.md) |
| LEG-033 | ap-settings-payment-helpers | legal | legal | P2 | 'Keys are stored securely in GCP Secret Manager' — specific technical claim tha… | [findings/legal.md](findings/legal.md) |
| LEG-036 | ap-staffdetail-role-descriptions | legal | legal | P2 | Role descriptions are one-line summaries ('Full access to everything', 'Read-on… | [findings/legal.md](findings/legal.md) |
| LEG-044 | web-mkt-brand-copyright | legal | legal | P2 | Copyright © {year} Fe3dr — no entity name beyond brand; for footer copyright to… | [findings/legal.md](findings/legal.md) |
| LEG-050 | mv-onb-policies-cancel-option-1h | legal | legal | P2 | 'Up to 1 hour before prep start' — consumer-facing implication of this option n… | [findings/legal.md](findings/legal.md) |
| LEG-051 | mv-onb-policies-cancel-option-30m | legal | legal | P2 | 'Up to 30 mins before prep start' — same as LEG-050, no consumer-side preview,… | [findings/legal.md](findings/legal.md) |
| LEG-053 | api-email-base-footer | legal | legal | P2 | No allergen disclaimer in email order confirmations — FSSAI requires allergen i… | [findings/legal.md](findings/legal.md) |
| LEG-055 | vp-legal-policies-agreements-title | legal | legal | P2 | 'Please read and accept each policy to complete your registration' — implies us… | [findings/legal.md](findings/legal.md) |
| LEG-056 | vp-legal-policy-tos | legal | legal | P2 | Vendor TOS uses 'I' first-person ('I have read and agree') — style-guide says l… | [findings/legal.md](findings/legal.md) |
| LEG-059 | vp-legal-policy-hygiene | legal | legal | P2 | 'I commit to maintaining a clean and hygienic kitchen at all times' — open-ende… | [findings/legal.md](findings/legal.md) |
| LEG-060 | vp-legal-policy-cancellation | legal | legal | P2 | 'Accept or reject orders within 5 minutes' — chef-side SLA; platform doesn't di… | [findings/legal.md](findings/legal.md) |
| TW-002 | web-legal-checkout-terms-note | legal | technical-writer | P2 | Checkout legal acknowledgment is inconsistent with signup — references only 'Te… | [findings/legal.md](findings/legal.md) |
| TW-004 | vp-legal-login-footer | legal | technical-writer | P2 | Vendor portal login uses 'Terms of Service' but vendor portal register uses 'Ve… | [findings/legal.md](findings/legal.md) |
| TW-005 | vp-legal-register-footer | legal | technical-writer | P2 | References 'Vendor Terms' but, per inventory note, has no link target. Either d… | [findings/legal.md](findings/legal.md) |
| TW-010 | vp-legal-docs-aadhaar | legal | technical-writer | P2 | For one of India's most sensitive IDs, helper text is just 'For identity verifi… | [findings/legal.md](findings/legal.md) |
| TW-011 | vp-legal-docs-fssai | legal | technical-writer | P2 | FSSAI is labeled 'Optional' here but is labeled 'Required' on the profile-level… | [findings/legal.md](findings/legal.md) |
| TW-014 | vp-legal-policies-compliance-title | legal | technical-writer | P2 | Second sentence is 18 words and uses 'maintain quality standards' — vague phras… | [findings/legal.md](findings/legal.md) |
| TW-015 | vp-legal-policies-compliance-items | legal | technical-writer | P2 | Compliance checklist items are inconsistent in grammatical form — mix of noun p… | [findings/legal.md](findings/legal.md) |
| TW-023 | vp-legal-settings-stripe | legal | technical-writer | P2 | 75-word block is a wall of mixed concepts: positioning ('for chefs outside Indi… | [findings/legal.md](findings/legal.md) |
| TW-024 | vp-legal-settings-stripe | legal | technical-writer | P2 | 'Stripe needs more information before you can accept payments.' — vague pronoun… | [findings/legal.md](findings/legal.md) |
| TW-032 | ap-secsettings-2fa-enforce | legal | technical-writer | P2 | 25-word combined string is at the customer-facing max but applied to admin (whe… | [findings/legal.md](findings/legal.md) |
| TW-033 | ap-secsettings-2fa-exempt | legal | technical-writer | P2 | 35-word helper text exceeds even customer-facing 25-word cap. Contains nested p… | [findings/legal.md](findings/legal.md) |
| TW-034 | ap-secsettings-apikey-warn | legal | technical-writer | P2 | Warning is one-time-secret disclosure but uses em-dash mid-sentence and lacks a… | [findings/legal.md](findings/legal.md) |
| TW-035 | ap-settings-payment-helpers | legal | technical-writer | P2 | Helper text is duplicated verbatim across two payment fields — Sec 3 inconsiste… | [findings/legal.md](findings/legal.md) |
| TW-037 | mv-onb-policies-checkbox | legal | technical-writer | P2 | 'I accept the terms and conditions' uses 'terms and conditions' — banned generi… | [findings/legal.md](findings/legal.md) |
| TW-039 | mv-settings-delete-confirm-body | legal | technical-writer | P2 | 'To delete your account, please contact our support team. This action cannot be… | [findings/legal.md](findings/legal.md) |
| TW-041 | mv-settings-delete-contact-email | legal | technical-writer | P2 | Email is 'support@homechef.app' but inventory notes brand drift — other surface… | [findings/legal.md](findings/legal.md) |
| TW-046 | md-leg-001 | legal | technical-writer | P2 | 'Terms of Service' on driver onboarding review screen — note says no navigation… | [findings/legal.md](findings/legal.md) |
| BA-019 | ap-auth-restricted-heading | legal | business-analyst | P3 | "Restricted Access" heading on admin login is security-framing language that cr… | [findings/legal.md](findings/legal.md) |
| BA-020 | vp-legal-docs-cheque | legal | business-analyst | P3 | Cancelled Cheque described as "optional, can add later" with no explanation of… | [findings/legal.md](findings/legal.md) |
| BA-021 | ap-secsettings-2fa-exempt | legal | business-analyst | P3 | Placeholder email "service@fe3dr.com" is hardcoded in the 2FA exempt list texta… | [findings/legal.md](findings/legal.md) |
| BV-014 | ap-auth-restricted-body | legal | brand-voice | P3 | Admin-portal restricted-access message uses 'administrators' rather than 'admin… | [findings/legal.md](findings/legal.md) |
| BV-016 | vp-legal-docs-fssai | legal | brand-voice | P3 | Inconsistent em-dash vs hyphen in onboarding description ('If you have one - gi… | [findings/legal.md](findings/legal.md) |
| BV-019 | vp-legal-policies-compliance-items | legal | brand-voice | P3 | Compliance items mix sentence styles within a single 7-item attestation list. T… | [findings/legal.md](findings/legal.md) |
| BV-021 | web-mkt-brand-copyright | legal | brand-voice | P3 | Copyright line says only 'Fe3dr' while email footer says 'Fe3dr by HomeChef'. W… | [findings/legal.md](findings/legal.md) |
| BV-022 | mv-onb-policies-cancel-option-no | legal | brand-voice | P3 | Cancellation option labels mix noun-phrase ('No cancellations after order accep… | [findings/legal.md](findings/legal.md) |
| LEG-031 | ap-auditlogs-subtitle | legal | legal | P3 | Audit log subtitle states 'IP, user agent' captured — admin's own personal data… | [findings/legal.md](findings/legal.md) |
| LEG-034 | ap-secsettings-2fa-enforce | legal | legal | P3 | 2FA enforcement toggle — no record of when/by whom the policy was last changed… | [findings/legal.md](findings/legal.md) |
| LEG-035 | ap-secsettings-apikey-warn | legal | legal | P3 | 'Copy this key now — you won't see it again' — operationally correct but doesn'… | [findings/legal.md](findings/legal.md) |
| LEG-037 | ap-auth-restricted-body | legal | legal | P3 | 'New registrations are not allowed' — informational but combined with 'Restrict… | [findings/legal.md](findings/legal.md) |
| LEG-047 | ap-auth-footer-legal | legal | legal | P3 | 'Fe3dr Administration Portal · Internal Use Only' — clarifies portal scope but… | [findings/legal.md](findings/legal.md) |
| LEG-048 | mv-onb-policies-cancel-label | legal | legal | P3 | 'Cancellation Policy *' — asterisk on label conflicts with style guide which sa… | [findings/legal.md](findings/legal.md) |
| LEG-052 | mv-settings-delete-contact-action | legal | legal | P3 | 'Contact Support' as the only action for account deletion — button text doesn't… | [findings/legal.md](findings/legal.md) |
| LEG-054 | vp-legal-docs-pan | legal | legal | P3 | PAN placeholder 'ABCDE1234F' — using a realistic-looking PAN as placeholder cou… | [findings/legal.md](findings/legal.md) |
| TW-001 | web-legal-register-terms | legal | technical-writer | P3 | Signup legal acknowledgment uses 'signing up' verb form, OK, but missing descri… | [findings/legal.md](findings/legal.md) |
| TW-003 | web-mkt-brand-copyright | legal | technical-writer | P3 | Copyright footer uses 'All rights reserved.' — vestigial legal boilerplate that… | [findings/legal.md](findings/legal.md) |
| TW-006 | vp-legal-docs-banner-title | legal | technical-writer | P3 | Header conflates two distinct purposes ('verify your kitchen' AND 'enable payou… | [findings/legal.md](findings/legal.md) |
| TW-008 | vp-legal-docs-identity-title | legal | technical-writer | P3 | Heading uses Title Case ('Identity Documents') in a UI that elsewhere uses sent… | [findings/legal.md](findings/legal.md) |
| TW-009 | vp-legal-docs-pan | legal | technical-writer | P3 | Helper 'Required for tax purposes and payouts' uses vague phrase 'tax purposes'… | [findings/legal.md](findings/legal.md) |
| TW-012 | vp-legal-docs-food-safety | legal | technical-writer | P3 | Helper 'Any food handling or safety certification (optional)' ends with parenth… | [findings/legal.md](findings/legal.md) |
| TW-013 | vp-legal-docs-cheque | legal | technical-writer | P3 | Label is 'Cancelled Cheque / Bank Proof' — dual label is ambiguous (is it one d… | [findings/legal.md](findings/legal.md) |
| TW-016 | vp-legal-policies-compliance-confirm | legal | technical-writer | P3 | Attestation gate uses 'By proceeding' — passive framing. Plain-language test pr… | [findings/legal.md](findings/legal.md) |
| TW-017 | vp-legal-policies-agreements-title | legal | technical-writer | P3 | Title 'Agreements & Policies' is Title Case. Helper 'Please read and accept eac… | [findings/legal.md](findings/legal.md) |
| TW-026 | ap-approvaldetail-notes-helper | legal | technical-writer | P3 | Helper text is clear and active-voice — borderline within admin tone. Could be… | [findings/legal.md](findings/legal.md) |
| TW-027 | ap-auth-restricted-heading | legal | technical-writer | P3 | 'Restricted Access' is Title Case; admin pages should match the sentence-case c… | [findings/legal.md](findings/legal.md) |
| TW-028 | ap-auth-restricted-body | legal | technical-writer | P3 | 'Only pre-authorized administrators can sign in. New registrations are not allo… | [findings/legal.md](findings/legal.md) |
| TW-029 | ap-auth-footer-legal | legal | technical-writer | P3 | 'Fe3dr Administration Portal · Internal Use Only' — 'Administration' is jargon… | [findings/legal.md](findings/legal.md) |
| TW-031 | ap-auditlogs-subtitle | legal | technical-writer | P3 | Subtitle is admin-precise and works. Minor: 'before/after values' is jargon-adj… | [findings/legal.md](findings/legal.md) |
| TW-036 | ap-staffdetail-role-descriptions | legal | technical-writer | P3 | Role descriptions are short noun phrases, mostly parallel, but 'Full access to… | [findings/legal.md](findings/legal.md) |
| TW-038 | mv-onb-policies-cancel-label | legal | technical-writer | P3 | 'Cancellation Policy *' — Title Case label and uses asterisk inline (Sec 4 form… | [findings/legal.md](findings/legal.md) |
| TW-040 | mv-settings-delete-contact-action | legal | technical-writer | P3 | 'Contact Support' button — Title Case. Sec 4 button formula: verb-first, ≤3 wor… | [findings/legal.md](findings/legal.md) |
| TW-042 | mv-onb-policies-cancel-option-no | legal | technical-writer | P3 | 'No cancellations after order accepted' — fine but ambiguous tense ('order acce… | [findings/legal.md](findings/legal.md) |
| TW-043 | mv-onb-policies-cancel-option-1h | legal | technical-writer | P3 | 'Up to 1 hour before prep start' — terse, parallel with -30m, OK. Minor: 'prep… | [findings/legal.md](findings/legal.md) |
| TW-044 | mv-onb-policies-cancel-option-30m | legal | technical-writer | P3 | 'Up to 30 mins before prep start' — uses 'mins' abbreviation; elsewhere app use… | [findings/legal.md](findings/legal.md) |
| TW-047 | md-leg-002 | legal | technical-writer | P3 | 'Privacy Policy' on driver onboarding review screen — assuming link wires, OK.… | [findings/legal.md](findings/legal.md) |

### Index — marketing

106 findings. Detail: [findings/marketing.md](findings/marketing.md)

| finding_id | surface_id | category | lens | severity | issue (≤80 chars) | detail link |
|---|---|---|---|---|---|---|
| BA-001 | web-mkt-landing-hero-badge | marketing | business-analyst | P0 | Hardcoded '500+ Home Chefs Near You' badge is not sourced from live data | [findings/marketing.md](findings/marketing.md) |
| BA-002 | web-mkt-landing-trust-badges | marketing | business-analyst | P0 | Three hardcoded trust-badge stats ('500+ Home Chefs', '4.8 Average Rating', '30… | [findings/marketing.md](findings/marketing.md) |
| BA-003 | web-mkt-hero-variants | marketing | business-analyst | P0 | Shared HeroSection Stats component repeats the same hardcoded '500+' chef count… | [findings/marketing.md](findings/marketing.md) |
| BA-004 | web-mkt-register-benefits | marketing | business-analyst | P0 | Customer signup page lists 'Access to 500+ home chefs' as a benefit — same unve… | [findings/marketing.md](findings/marketing.md) |
| BA-005 | web-mkt-login-testimonial | marketing | business-analyst | P0 | Fake placeholder testimonial from 'Sarah M., Happy Customer' ships in productio… | [findings/marketing.md](findings/marketing.md) |
| BA-006 | web-mkt-landing-why-choose | marketing | business-analyst | P0 | Subtitle 'Join thousands of happy customers enjoying homemade food' is an unver… | [findings/marketing.md](findings/marketing.md) |
| BA-007 | vp-mkt-register-benefits-list | marketing | business-analyst | P0 | 'Join thousands of home chefs earning with Fe3dr' on the vendor registration pa… | [findings/marketing.md](findings/marketing.md) |
| BA-008 | vp-mkt-register-benefits-list | marketing | business-analyst | P0 | 'Weekly payouts — Get paid directly to your bank account' is advertised as a li… | [findings/marketing.md](findings/marketing.md) |
| BA-009 | web-mkt-landing-why-choose | marketing | business-analyst | P0 | 'Verified Chefs — All our home chefs are verified for food safety and quality'… | [findings/marketing.md](findings/marketing.md) |
| BA-010 | web-mkt-landing-become-chef | marketing | business-analyst | P0 | Both CTAs in the 'Become a Chef' section link to unregistered routes (/become-c… | [findings/marketing.md](findings/marketing.md) |
| BA-011 | web-mkt-hero-how-it-works-cta | marketing | business-analyst | P0 | Secondary hero CTA 'How It Works' links to /how-it-works, a route not registere… | [findings/marketing.md](findings/marketing.md) |
| BA-012 | null | marketing | business-analyst | P0 | Admin analytics page ships 'Chart coming soon' placeholder to operators — the f… | [findings/marketing.md](findings/marketing.md) |
| BV-001 | web-mkt-landing-hero-badge | marketing | brand-voice | P0 | Hardcoded '500+ Home Chefs Near You' on landing hero badge — unverified platfor… | [findings/marketing.md](findings/marketing.md) |
| BV-002 | web-mkt-landing-trust-badges | marketing | brand-voice | P0 | Three hardcoded 'trust signals' below the hero — '500+ Home Chefs', '4.8 Averag… | [findings/marketing.md](findings/marketing.md) |
| BV-003 | web-mkt-hero-variants | marketing | brand-voice | P0 | Shared `HeroSection` `Stats` component hardcodes the same fabricated metrics as… | [findings/marketing.md](findings/marketing.md) |
| BV-004 | web-mkt-landing-why-choose | marketing | brand-voice | P0 | 'Join thousands of happy customers enjoying homemade food' under 'The Fe3dr Dif… | [findings/marketing.md](findings/marketing.md) |
| BV-005 | web-mkt-register-benefits | marketing | brand-voice | P0 | RegisterPage left-rail hardcodes 'Access to 500+ home chefs' in the BENEFITS ar… | [findings/marketing.md](findings/marketing.md) |
| BV-006 | web-mkt-register-benefits | marketing | brand-voice | P0 | Heading 'Get access to hundreds of home chefs serving authentic, homemade food… | [findings/marketing.md](findings/marketing.md) |
| BV-007 | vp-mkt-register-benefits-list | marketing | brand-voice | P0 | Vendor portal register page promises 'Join thousands of home chefs earning with… | [findings/marketing.md](findings/marketing.md) |
| BV-008 | web-mkt-landing-featured-chefs-heading | marketing | brand-voice | P0 | 'Our community's favorite home chefs' subhead under 'Featured Chefs / Top Rated… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-001 | web-mkt-landing-hero-badge | marketing | legal | P0 | Hardcoded social-proof metric '500+ Home Chefs Near You' rendered without subst… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-002 | web-mkt-landing-trust-badges | marketing | legal | P0 | Trust badge row asserts three quantified marketing claims ('500+ Home Chefs', '… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-003 | web-mkt-register-benefits | marketing | legal | P0 | Marketing claim '500+ home chefs' duplicated on registration entry page reinfor… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-004 | web-mkt-landing-why-choose | marketing | legal | P0 | Marketing card states 'All our home chefs are verified for food safety and qual… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-010 | web-mkt-landing-why-choose | marketing | legal | P0 | 'Fast Delivery / Reliable delivery to your doorstep within 30-45 minutes' is a… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-012 | web-mkt-login-testimonial | marketing | legal | P0 | Testimonial attributed to 'Sarah M., Happy Customer' with brand-superlative cla… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-015 | vp-mkt-register-benefits-list | marketing | legal | P0 | 'Zero commission first month / Get started completely risk-free' is a binding f… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-021 | web-mkt-layout-footer | marketing | legal | P0 | Footer copyright reads '© <year> Fe3dr' while the project name in CLAUDE.md is… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-026 | web-mkt-layout-footer | marketing | legal | P0 | Footer links 'Privacy Policy' and 'Terms of Service' are present, but no link t… | [findings/marketing.md](findings/marketing.md) |
| TW-007 | vp-mkt-register-benefits-list | marketing | technical-writer | P0 | Vendor portal register page promises 'Zero commission first month' as a concret… | [findings/marketing.md](findings/marketing.md) |
| BA-013 | web-mkt-landing-hero-title | marketing | business-analyst | P1 | Hero search button says 'Find Food' but the search actually routes to /chefs —… | [findings/marketing.md](findings/marketing.md) |
| BA-014 | web-mkt-landing-become-chef | marketing | business-analyst | P1 | 'Love Cooking? Share Your Talent' headline in the chef recruitment CTA is weak… | [findings/marketing.md](findings/marketing.md) |
| BA-015 | vp-mkt-register-benefits-list | marketing | business-analyst | P1 | 'Zero commission first month' benefit claim has no terms attached — creates mis… | [findings/marketing.md](findings/marketing.md) |
| BA-016 | web-mkt-catering-request-heading | marketing | business-analyst | P1 | Catering request wizard has no CTA that sets expectations for what happens afte… | [findings/marketing.md](findings/marketing.md) |
| BV-009 | web-mkt-login-testimonial | marketing | brand-voice | P1 | Login page right-rail features a fabricated testimonial: 'Fe3dr has changed how… | [findings/marketing.md](findings/marketing.md) |
| BV-010 | web-mkt-landing-hero-title | marketing | brand-voice | P1 | Two different homepage heroes coexist in the codebase. The live `HomePage.tsx`… | [findings/marketing.md](findings/marketing.md) |
| BV-011 | web-mkt-hero-variants | marketing | brand-voice | P1 | Catering hero variant: 'Catering for Every Occasion / our home chefs bring auth… | [findings/marketing.md](findings/marketing.md) |
| BV-012 | web-mkt-layout-footer | marketing | brand-voice | P1 | Footer brand blurb 'Connecting you with home chefs for authentic, homemade food… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-005 | web-mkt-landing-why-choose | marketing | legal | P1 | 'Made with Love / Every meal is prepared fresh with authentic family recipes' —… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-006 | web-mkt-landing-hero-title | marketing | legal | P1 | Hero title 'Homemade Food, Delivered Fresh' uses 'Fresh' as the herb-accent mar… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-011 | web-mkt-hero-variants | marketing | legal | P1 | Shared HeroSection component holds 4 hero variants used across home/chefs/cater… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-013 | web-mkt-landing-featured-chefs-heading | marketing | legal | P1 | 'Top Rated' badge applied to featured-chefs section without disclosed methodolo… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-014 | web-mkt-landing-why-choose | marketing | legal | P1 | 'Join thousands of happy customers enjoying homemade food' — unsubstantiated us… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-016 | vp-mkt-register-benefits-list | marketing | legal | P1 | 'Weekly payouts / Get paid directly to your bank account' is a payment-flow com… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-017 | vp-mkt-register-benefits-list | marketing | legal | P1 | 'Never miss an order with instant notifications' is an absolute reliability cla… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-018 | web-mkt-landing-become-chef | marketing | legal | P1 | 'Turn your passion into income' on the consumer-facing landing page solicits ch… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-019 | web-mkt-landing-catering-cta | marketing | legal | P1 | 'Get catering quotes from multiple home chefs. Perfect for parties, corporate e… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-022 | web-mkt-login-testimonial | marketing | legal | P1 | Testimonial references 'Fe3dr' brand name in user-attributed quotation — if 'Fe… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-023 | web-mkt-landing-why-choose | marketing | legal | P1 | 'The Fe3dr Difference' section header — same brand-identity issue surfaces in m… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-024 | web-mkt-register-benefits | marketing | legal | P1 | Registration page invites user to 'Join Fe3dr Today' — at the contract-formatio… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-027 | web-mkt-layout-footer | marketing | legal | P1 | Footer brand description 'Connecting you with home chefs for authentic, homemad… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-034 | web-mkt-landing-catering-cta | marketing | legal | P1 | Catering CTA 'Request Catering Quote' leads to a form that will collect persona… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-035 | web-mkt-landing-become-chef | marketing | legal | P1 | 'Become a Chef' CTA leads chef-onboarding flow that will collect KYC-class pers… | [findings/marketing.md](findings/marketing.md) |
| TW-001 | web-mkt-landing-hero-badge | marketing | technical-writer | P1 | Hardcoded social-proof metric '500+ Home Chefs Near You' in hero badge — unveri… | [findings/marketing.md](findings/marketing.md) |
| TW-002 | web-mkt-landing-trust-badges | marketing | technical-writer | P1 | Hardcoded '500+ Home Chefs' + '4.8 Average Rating' + '30-45 min Delivery' trust… | [findings/marketing.md](findings/marketing.md) |
| TW-003 | web-mkt-landing-why-choose | marketing | technical-writer | P1 | Generic AI-slop social proof: 'Join thousands of happy customers enjoying homem… | [findings/marketing.md](findings/marketing.md) |
| TW-004 | web-mkt-register-benefits | marketing | technical-writer | P1 | Hardcoded 'Access to 500+ home chefs' in registration benefits list — same unve… | [findings/marketing.md](findings/marketing.md) |
| TW-005 | web-mkt-hero-variants | marketing | technical-writer | P1 | Shared HeroSection.tsx Stats component hardcodes '500+ Home Chefs', '4.8 Avg Ra… | [findings/marketing.md](findings/marketing.md) |
| TW-006 | web-mkt-login-testimonial | marketing | technical-writer | P1 | Hardcoded testimonial from fabricated user 'Sarah M., Happy Customer' on the lo… | [findings/marketing.md](findings/marketing.md) |
| BA-017 | web-mkt-browse-heading | marketing | business-analyst | P2 | Browse Chefs page heading 'Explore Home Chefs' with sub 'Discover talented home… | [findings/marketing.md](findings/marketing.md) |
| BA-018 | web-mkt-landing-how-it-works | marketing | business-analyst | P2 | 'How It Works' step 2 (Order) says only 'Select your favorite dishes and place… | [findings/marketing.md](findings/marketing.md) |
| BA-019 | web-mkt-landing-why-choose | marketing | business-analyst | P2 | 'Made with Love' card title uses a banned brand-drift phrase per STYLE-GUIDE.md | [findings/marketing.md](findings/marketing.md) |
| BA-020 | web-mkt-landing-catering-cta | marketing | business-analyst | P2 | Catering CTA 'Planning an Event?' is a yes/no question that gates a ₹-significa… | [findings/marketing.md](findings/marketing.md) |
| BV-013 | web-mkt-landing-why-choose | marketing | brand-voice | P2 | 'Made with Love / Every meal is prepared fresh with authentic family recipes' i… | [findings/marketing.md](findings/marketing.md) |
| BV-014 | web-mkt-landing-hero-subtitle | marketing | brand-voice | P2 | Hero description: 'Discover talented home chefs in your neighborhood and enjoy… | [findings/marketing.md](findings/marketing.md) |
| BV-015 | web-mkt-browse-heading | marketing | brand-voice | P2 | 'Discover talented home chefs serving authentic homemade food' — repeats the 't… | [findings/marketing.md](findings/marketing.md) |
| BV-016 | web-mkt-landing-cuisines | marketing | brand-voice | P2 | 'Cuisines / Explore Flavors / Discover authentic dishes from around the world'… | [findings/marketing.md](findings/marketing.md) |
| BV-017 | web-mkt-landing-how-it-works | marketing | brand-voice | P2 | 'Get Delicious Food in 3 Steps' uses 'Delicious Food' — generic consumer-market… | [findings/marketing.md](findings/marketing.md) |
| BV-018 | web-mkt-landing-become-chef | marketing | brand-voice | P2 | 'Love Cooking? Share Your Talent / Turn your passion into income. Join our comm… | [findings/marketing.md](findings/marketing.md) |
| BV-019 | web-mkt-landing-catering-cta | marketing | brand-voice | P2 | 'Catering Services / Planning an Event?' uses Title Case for 'Catering Services… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-007 | web-mkt-landing-how-it-works | marketing | legal | P2 | 'Get fresh homemade food delivered to your doorstep' restates 'fresh' as a deli… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-008 | web-mkt-browse-heading | marketing | legal | P2 | Browse-chefs page intro 'Discover talented home chefs serving authentic homemad… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-009 | web-mkt-landing-hero-subtitle | marketing | legal | P2 | Hero subtitle 'Discover talented home chefs in your neighborhood and enjoy auth… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-020 | web-mkt-catering-request-heading | marketing | legal | P2 | Catering-request flow heading 'Tell us about your event' onboards the consumer… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-028 | web-mkt-landing-search-cta | marketing | legal | P2 | Primary CTA 'Find Food' on the home page leads to /chefs without any geographic… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-032 | web-mkt-landing-why-choose | marketing | legal | P2 | 'Secure Payments / Safe and secure payment processing for every order' — 'safe'… | [findings/marketing.md](findings/marketing.md) |
| TW-008 | web-mkt-landing-hero-subtitle | marketing | technical-writer | P2 | Hero subtitle is generic landing-page filler — 'Discover talented home chefs in… | [findings/marketing.md](findings/marketing.md) |
| TW-009 | web-mkt-browse-heading | marketing | technical-writer | P2 | Browse page intro duplicates the same 'Discover talented home chefs serving aut… | [findings/marketing.md](findings/marketing.md) |
| TW-010 | web-mkt-landing-how-it-works | marketing | technical-writer | P2 | 'Get Delicious Food in 3 Steps' — 'Delicious' is filler adjective; 'we make it… | [findings/marketing.md](findings/marketing.md) |
| TW-011 | web-mkt-landing-why-choose | marketing | technical-writer | P2 | Feature card 'Made with Love / Every meal is prepared fresh with authentic fami… | [findings/marketing.md](findings/marketing.md) |
| TW-012 | web-mkt-landing-become-chef | marketing | technical-writer | P2 | 'Love Cooking? Share Your Talent' + 'Turn your passion into income' — both phra… | [findings/marketing.md](findings/marketing.md) |
| TW-016 | web-mkt-landing-search-cta | marketing | technical-writer | P2 | Primary search CTA 'Find Food' is title-case; style guide Sec 4 button format r… | [findings/marketing.md](findings/marketing.md) |
| TW-017 | web-mkt-hero-search | marketing | technical-writer | P2 | Shared HeroSection's CTA 'Find Chefs' is title-case (style guide says sentence… | [findings/marketing.md](findings/marketing.md) |
| TW-019 | web-mkt-landing-catering-cta | marketing | technical-writer | P2 | 'Request Catering Quote' button is title case (3 words is fine, case is not). A… | [findings/marketing.md](findings/marketing.md) |
| BA-021 | web-mkt-layout-footer | marketing | business-analyst | P3 | Footer 'For Chefs' column links to /become-chef and /chef-resources — both dead… | [findings/marketing.md](findings/marketing.md) |
| BA-022 | web-mkt-brand-tagline | marketing | business-analyst | P3 | Logo tagline 'Homemade Food Delivered' is shown in the footer but is too generi… | [findings/marketing.md](findings/marketing.md) |
| BV-020 | web-mkt-landing-search-cta | marketing | brand-voice | P3 | Primary search CTA reads 'Find Food' (Title Case) while the shared HeroSection'… | [findings/marketing.md](findings/marketing.md) |
| BV-021 | web-mkt-hero-how-it-works-cta | marketing | brand-voice | P3 | Secondary CTA 'How It Works' is Title Case in the shared hero AND in the homepa… | [findings/marketing.md](findings/marketing.md) |
| BV-022 | web-mkt-landing-featured-chefs-heading | marketing | brand-voice | P3 | 'View All Chefs' (Title Case) on a ghost-variant button; cuisines section uses… | [findings/marketing.md](findings/marketing.md) |
| BV-023 | web-mkt-landing-become-chef | marketing | brand-voice | P3 | Become-a-chef section has two CTAs: 'Become a Chef' (Title Case) and 'Learn Mor… | [findings/marketing.md](findings/marketing.md) |
| BV-024 | web-mkt-brand-tagline | marketing | brand-voice | P3 | Logo tagline 'Homemade Food Delivered' is Title Case, contradicting all other t… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-025 | web-mkt-brand-tagline | marketing | legal | P3 | Logo tagline 'Homemade Food Delivered' is non-quantified and low-risk on its ow… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-029 | web-mkt-landing-search-placeholders | marketing | legal | P3 | Search placeholder 'Enter your delivery address...' collects an address (person… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-030 | web-mkt-hero-search | marketing | legal | P3 | Shared HeroSection search ('Enter your delivery address / Find Chefs') replicat… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-031 | web-mkt-landing-cuisines | marketing | legal | P3 | Cuisine cards (South Indian, Italian, Japanese, North Indian, Mexican, Thai) im… | [findings/marketing.md](findings/marketing.md) |
| LEG-MKT-033 | web-mkt-landing-how-it-works | marketing | legal | P3 | How-It-Works step 'Order — Select your favorite dishes and place your order sec… | [findings/marketing.md](findings/marketing.md) |
| TW-013 | web-mkt-landing-featured-chefs-heading | marketing | technical-writer | P3 | 'Our community's favorite home chefs' uses the vague 'community's favourite' ge… | [findings/marketing.md](findings/marketing.md) |
| TW-014 | web-mkt-landing-cuisines | marketing | technical-writer | P3 | 'Explore Flavors / Discover authentic dishes from around the world' — 'authenti… | [findings/marketing.md](findings/marketing.md) |
| TW-015 | web-mkt-landing-catering-cta | marketing | technical-writer | P3 | 'Get catering quotes from multiple home chefs. Perfect for parties, corporate e… | [findings/marketing.md](findings/marketing.md) |
| TW-018 | web-mkt-hero-how-it-works-cta | marketing | technical-writer | P3 | 'How It Works' button is title case; style guide requires sentence case for but… | [findings/marketing.md](findings/marketing.md) |
| TW-020 | web-mkt-landing-become-chef | marketing | technical-writer | P3 | 'Become a Chef' and 'Learn More' buttons both title-case; 'Learn More' violates… | [findings/marketing.md](findings/marketing.md) |
| TW-021 | web-mkt-landing-featured-chefs-heading | marketing | technical-writer | P3 | 'View All Chefs' is title case (sentence-case rule). | [findings/marketing.md](findings/marketing.md) |
| TW-022 | web-mkt-layout-footer | marketing | technical-writer | P3 | Footer brand lead-in 'Connecting you with home chefs for authentic, homemade fo… | [findings/marketing.md](findings/marketing.md) |
| TW-023 | web-mkt-brand-tagline | marketing | technical-writer | P3 | Logo tagline 'Homemade Food Delivered' is title case but is meant as a tagline… | [findings/marketing.md](findings/marketing.md) |
| TW-024 | web-mkt-landing-become-chef | marketing | technical-writer | P3 | Body paragraph runs 22 words but stuffs two unrelated ideas into one sentence:… | [findings/marketing.md](findings/marketing.md) |
| TW-025 | web-mkt-catering-request-heading | marketing | technical-writer | P3 | 'Event Details / Tell us about your event' is acceptable as a step heading but… | [findings/marketing.md](findings/marketing.md) |

### Index — auth-onboarding

418 findings. Detail: [findings/auth-onboarding.md](findings/auth-onboarding.md)

| finding_id | surface_id | category | lens | severity | issue (≤80 chars) | detail link |
|---|---|---|---|---|---|---|
| BA-001 | web-auth-register-benefits | auth-onboarding | business-analyst | P0 | Hardcoded '500+ home chefs' claim on customer register page — fake metric | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-002 | vendor-portal-register-benefits | auth-onboarding | business-analyst | P0 | Hardcoded 'Join thousands of home chefs' on vendor register page — fake social… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-003 | web-auth-login-testimonial | auth-onboarding | business-analyst | P0 | Fake testimonial ('Sarah M., Happy Customer') hardcoded on login page hero panel | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-040 | delivery-portal-register-redirect | auth-onboarding | business-analyst | P0 | Driver registration page is a blank redirect to login — drivers cannot self-reg… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-001 | web-auth-register-heading | auth-onboarding | brand-voice | P0 | Marketing-toned 'Access to 500+ home chefs' bullet on a customer register page… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-002 | vp-auth-register-hero-sub | auth-onboarding | brand-voice | P0 | 'Join thousands of home chefs earning with Fe3dr' — unverified 'thousands' clai… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-003 | vp-auth-register-hero-sub | auth-onboarding | brand-voice | P0 | Marketing bullets on chef register page — 'Zero commission first month' and 'We… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-004 | dp-auth-step4-zero-commission | auth-onboarding | brand-voice | P0 | Two-paragraph 'We don't take any commission' block inside driver onboarding ste… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-006 | web-auth-login-email-toggle | auth-onboarding | brand-voice | P0 | 'Sign in with email' vs 'Sign in with Email' vs 'Sign in with Email' — capitali… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-007 | web-auth-register-email-toggle | auth-onboarding | brand-voice | P0 | Register-with-email toggle wording drifts: 'Sign up with email' (web) vs 'Regis… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-008 | dp-auth-login-subtitle-driver | auth-onboarding | brand-voice | P0 | 'Driver login or sign up' and 'Login or sign up to deliver with Fe3dr' mix bann… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-011 | dp-auth-login-title | auth-onboarding | brand-voice | P0 | Brand wordmark on delivery login is 'Fe3dr Delivery' — sub-brand applied to log… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-013 | vp-onb-personal-title | auth-onboarding | brand-voice | P0 | Same chef onboarding step labelled differently across mobile-vendor and vendor-… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-018 | dp-auth-onboarding-step-labels | auth-onboarding | brand-voice | P0 | Driver step labels Title Case on delivery-portal ('Personal Info', 'Vehicle Det… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-021 | dp-auth-status-h | auth-onboarding | brand-voice | P0 | Status page sub-heading is 'Delivery Partner Onboarding' on delivery-portal — d… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-022 | dp-auth-mode-driver-sub | auth-onboarding | brand-voice | P0 | 'I'm a Driver' / 'I'm Staff' role-chooser uses Title Case + capitalised role —… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-037 | dp-auth-status-titles | auth-onboarding | brand-voice | P0 | Same status labelled differently across surfaces: 'Under Review' (dp web title)… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-001 | web-auth-register-form-fields | auth-onboarding | legal | P0 | Signup form collects personal data (name, email, password) with no DPDP §6 cons… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-002 | web-auth-register-submit | auth-onboarding | legal | P0 | Account creation submit button has no checkbox-gated affirmative-action consent… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-003 | web-auth-register-form-fields | auth-onboarding | legal | P0 | No children's-data screening at signup; platform has no age gate | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-005 | web-auth-register-heading | auth-onboarding | legal | P0 | No grievance-officer contact disclosed at signup or in the visible consent foot… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-006 | web-auth-register-heading | auth-onboarding | legal | P0 | No disclosure of user rights (access, correction, erasure, nomination, grievanc… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-008 | web-auth-register-social | auth-onboarding | legal | P0 | OAuth signup (Google/Facebook) does not disclose what data is shared with third… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-012 | web-auth-onboarding-step-basic | auth-onboarding | legal | P0 | Phone number collected post-signup without separate purpose-specific consent (t… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-021 | vp-auth-register-login-link | auth-onboarding | legal | P0 | Vendor register page T&C disclosure says 'By registering, you agree...' — impli… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-022 | vp-onb-personal-blurb | auth-onboarding | legal | P0 | 'This information helps verify your identity' is KYC framing without disclosure… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-023 | vp-onb-personal-fields | auth-onboarding | legal | P0 | Vendor full name, phone, email collected as KYC without explicit DPDP consent l… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-030 | dp-auth-step3-doc-labels | auth-onboarding | legal | P0 | Aadhaar collected as document with no UIDAI offline e-KYC / consent framework d… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-031 | dp-auth-step3-doc-labels | auth-onboarding | legal | P0 | PAN Card collection with no Income Tax §139A purpose disclosure or Rule 114B co… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-035 | dp-auth-step-payout-bank-fields | auth-onboarding | legal | P0 | Bank account collection (Account Holder Name, Account Number, IFSC) with no RBI… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-038 | dp-auth-step1-dob | auth-onboarding | legal | P0 | Date of Birth collected with no age-validation gate — driver could be a minor | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-042 | dp-auth-step4-zero-commission | auth-onboarding | legal | P0 | Multiple repeated 'no commission' / '100%' promises in subscription marketing b… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-047 | dp-auth-step5-terms | auth-onboarding | legal | P0 | 'I agree to the Terms & Conditions, Privacy Policy' — links not wired (per inve… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-048 | dp-auth-step5-terms | auth-onboarding | legal | P0 | Bundled consent: 'Terms + Privacy Policy + accuracy attestation' is three oblig… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-052 | dp-auth-status-rejection-reason | auth-onboarding | legal | P0 | 'Reason for rejection: {rejectionReason}' — no appeal pathway shown, no DPDP §1… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-057 | ap-secsettings-2fa-disable | auth-onboarding | legal | P0 | Disabling 2FA is highly-sensitive privileged action — no co-approval or audit-t… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-059 | mc-auth-register-host | auth-onboarding | legal | P0 | Mobile customer register screen has zero local strings — all copy in shared scr… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-063 | mv-onb-docs-sub | auth-onboarding | legal | P0 | 'Upload your identity and FSSAI documents' — both are critical regulatory docum… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-064 | mv-onb-docs-fssai-slot | auth-onboarding | legal | P0 | 'FSSAI License' upload slot — if marked optional or accepted-as-blank, platform… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-065 | mv-onb-docs-id-slot | auth-onboarding | legal | P0 | 'ID Proof' is non-specific — must specify accepted IDs (PAN required for payout… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-074 | md-onb-042 | auth-onboarding | legal | P0 | 'Upload Documents' mobile screen — same Aadhaar/PAN/license concerns as dp-auth… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-075 | md-onb-044 | auth-onboarding | legal | P0 | 'Driving License' slot — Motor Vehicles Act compliance and DL data treatment | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-076 | md-onb-045 | auth-onboarding | legal | P0 | Mobile 'ID Proof' slot — generic label same concern as mv-onb-docs-id-slot (LEG… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-081 | md-onb-059 | auth-onboarding | legal | P0 | 'Payout Details' mobile screen — same RBI PA MD concerns as dp-auth-step-payout… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-082 | md-onb-063 | auth-onboarding | legal | P0 | Mobile bank account collection without on-device security disclosure (autofill,… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-083 | md-onb-079 | auth-onboarding | legal | P0 | 'Your payout details are encrypted and never stored on this device' — binding f… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-086 | md-onb-082 | auth-onboarding | legal | P0 | Mobile 'Choose Your Plan' subscription screen — same dark-pattern subscription-… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-087 | md-onb-093 | auth-onboarding | legal | P0 | Mobile review screen + Terms-acceptance gate parallels dp-auth-step5-terms (LEG… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-088 | md-onb-103 | auth-onboarding | legal | P0 | 'I accept the Terms of Service and Privacy Policy' — single checkbox bundles tw… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-089 | md-onb-103 | auth-onboarding | legal | P0 | T&C and Privacy Policy strings in mobile review have no hyperlink behavior conf… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-100 | dp-auth-login-title | auth-onboarding | legal | P0 | Across web, vendor-portal, delivery-portal, and three mobile apps — there is NO… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-103 | vp-onb-stepper-labels | auth-onboarding | legal | P0 | Across vendor-portal AND mobile-vendor onboarding — there is no visible disclos… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-104 | dp-auth-onboarding-header | auth-onboarding | legal | P0 | Cross-driver-onboarding: no disclosure of platform-side insurance arrangement (… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-120 | md-auth-002 | auth-onboarding | technical-writer | P0 | User-facing error 'Google sign-in failed: no ID token' is a developer-shaped er… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-121 | md-auth-003 | auth-onboarding | technical-writer | P0 | 'Apple sign-in failed: no identity token' — same developer-shaped error pattern… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-004 | web-auth-register-heading | auth-onboarding | business-analyst | P1 | Register page H1 'Create your account' is utility language — no value prop for… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-011 | vendor-portal-register-heading | auth-onboarding | business-analyst | P1 | 'Register your kitchen' as H1 is process-oriented, not value-oriented — no inco… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-012 | vendor-portal-register-benefits | auth-onboarding | business-analyst | P1 | 'Zero commission first month' benefit is buried 3rd item on the panel with no m… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-013 | vendor-portal-register-tos | auth-onboarding | business-analyst | P1 | Vendor registration TOS consent 'By registering, you agree to Fe3dr's Vendor Te… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-017 | vendor-portal-login-access-denied | auth-onboarding | business-analyst | P1 | Access-denied error 'This portal is only for vendor accounts. Please use the Fe… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-018 | vendor-portal-onboarding-header | auth-onboarding | business-analyst | P1 | Wizard header copy 'Set Up Your Kitchen' + 'Step 3 of 5' gives no estimated com… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-020 | vendor-portal-onboarding-wizard | auth-onboarding | business-analyst | P1 | Mobile step indicator shows numeric fraction '3/6' only — no step label on mobi… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-021 | vendor-portal-onboarding-step-personal | auth-onboarding | business-analyst | P1 | Personal Info step has 8 required fields visible at once (full name, phone, ema… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-029 | vendor-portal-onboarding-step-documents | auth-onboarding | business-analyst | P1 | Documents step introduces PAN card, Aadhaar, kitchen photos — no upfront framin… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-030 | vendor-portal-onboarding-step-documents | auth-onboarding | business-analyst | P1 | Trust framing for Aadhaar collection is minimal — 'For identity verification' p… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-031 | vendor-portal-onboarding-step-documents | auth-onboarding | business-analyst | P1 | Cancelled cheque optional document description 'For setting up direct payouts t… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-033 | vendor-portal-onboarding-step-policies | auth-onboarding | business-analyst | P1 | 7-item compliance checklist (COMPLIANCE_ITEMS) displayed as a read-only list wi… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-034 | vendor-portal-onboarding-step-policies | auth-onboarding | business-analyst | P1 | Terms of Service policy checkbox description says 'Fe3dr charges a platform com… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-038 | vendor-portal-onboarding-review | auth-onboarding | business-analyst | P1 | Submit Application toast 'Application submitted! We'll review and get back to y… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-039 | vendor-portal-onboarding-rejection | auth-onboarding | business-analyst | P1 | Rejection banner heading 'Application Rejected' is blunt — no empathetic framin… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-041 | delivery-portal-onboarding-header | auth-onboarding | business-analyst | P1 | 'Become a Delivery Partner' as wizard H1 with subline 'Complete your profile to… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-044 | delivery-portal-onboarding-subscription | auth-onboarding | business-analyst | P1 | Subscription step headline 'Choose Your Plan' appears at step 4 of 5 — driver h… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-049 | delivery-portal-onboarding-status | auth-onboarding | business-analyst | P1 | Status page 'Application Submitted' state shows no next action — driver is left… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-050 | delivery-portal-onboarding-status | auth-onboarding | business-analyst | P1 | Rejection state heading 'Application Needs Changes' with description 'Your appl… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-053 | delivery-portal-onboarding-payout | auth-onboarding | business-analyst | P1 | Payout Details step has no security framing for bank account / IFSC submission… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-005 | dp-auth-step4-billing-howto | auth-onboarding | brand-voice | P1 | Driver-facing 'How billing works' is a 50-word dense block with 4-step list ins… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-009 | ap-auth-login-heading | auth-onboarding | brand-voice | P1 | 'Admin Sign In' uses Title Case for a heading + the canonical verb is one word… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-012 | vp-auth-login-features-list | auth-onboarding | brand-voice | P1 | 'This portal is only for vendor accounts. Please use the Fe3dr customer app.' —… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-014 | vp-onb-kitchen-title | auth-onboarding | brand-voice | P1 | Same chef onboarding step labelled 'Kitchen Information' (vp StepKitchenDetails… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-015 | vp-onb-ops-pricing-title | auth-onboarding | brand-voice | P1 | Web chef step is 'Delivery & Pricing' / 'Operations' (stepper) but mobile-vendo… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-016 | vp-onb-header-title | auth-onboarding | brand-voice | P1 | Web chef onboarding header is 'Set Up Your Kitchen' (Title Case) while no mobil… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-017 | vp-onb-stepper-labels | auth-onboarding | brand-voice | P1 | Web chef stepper uses Title Case ('Personal Info', 'Kitchen Details', 'Operatio… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-019 | dp-auth-step3-h | auth-onboarding | brand-voice | P1 | Step heading 'Documents & Photos' (delivery-portal) vs 'Upload Documents' (mobi… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-020 | dp-auth-onboarding-header | auth-onboarding | brand-voice | P1 | Driver onboarding entry header is 'Become a Delivery Partner' on web (delivery-… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-023 | vp-auth-login-hero-heading | auth-onboarding | brand-voice | P1 | Chef login hero 'Grow your home kitchen business' is marketing-tone on a return… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-024 | vp-auth-login-features-list | auth-onboarding | brand-voice | P1 | 5-bullet feature list ('Real-time order management', 'Earnings & analytics dash… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-025 | ap-auth-login-feature-list | auth-onboarding | brand-voice | P1 | Admin portal login left panel has 6 sales-style feature bullets ('User & role m… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-028 | mv-onb-personal-heading | auth-onboarding | brand-voice | P1 | Mobile-vendor onboarding screens use Title Case headings ('Personal Information… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-036 | dp-auth-status-descriptions | auth-onboarding | brand-voice | P1 | Application-waiting copy drifts across driver and chef surfaces. Driver: 'Our t… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-038 | md-onb-109 | auth-onboarding | brand-voice | P1 | 'Application Not Approved' followed by 'Unfortunately your application was not… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-041 | md-auth-002 | auth-onboarding | brand-voice | P1 | Developer-shaped error 'Google sign-in failed: no ID token' / 'Apple sign-in fa… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-045 | dp-auth-step5-terms | auth-onboarding | brand-voice | P1 | Driver consent string 'I agree to the Terms & Conditions, Privacy Policy, and c… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-052 | dp-auth-step5-submit | auth-onboarding | brand-voice | P1 | Final submit CTA drift: driver web 'Submit Application' / driver mobile 'Submit… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-004 | web-auth-register-heading | auth-onboarding | legal | P1 | Signup heading does not identify the legal entity (data fiduciary) collecting t… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-007 | web-auth-register-form-fields | auth-onboarding | legal | P1 | Password collected and stored; no statement of security practices (encryption a… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-009 | web-auth-register-social | auth-onboarding | legal | P1 | OAuth buttons identical for sign-up and login — different consent expectations;… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-011 | web-auth-login-email-fields | auth-onboarding | legal | P1 | 'Forgot password?' flow not inventoried for legal exposure — password reset ema… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-013 | web-auth-onboarding-step-address | auth-onboarding | legal | P1 | Delivery address collected with no retention statement or purpose limitation | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-014 | web-auth-onboarding-step-preferences | auth-onboarding | legal | P1 | Dietary preferences and allergies collected — potentially sensitive health data… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-016 | web-auth-onboarding-heading | auth-onboarding | legal | P1 | 'Skip' link on onboarding wizard implies users can bypass data collection — mus… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-018 | vp-auth-login-features-list | auth-onboarding | legal | P1 | Pre-contract feature promise 'Document verification & compliance' implies platf… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-019 | vp-auth-register-hero-sub | auth-onboarding | legal | P1 | 'Join thousands of home chefs earning' is a verifiable factual claim — must be… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-024 | vp-onb-kitchen-blurb | auth-onboarding | legal | P1 | 'Tell customers about your kitchen and what makes your food special' — chef-aut… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-025 | vp-onb-personal-address-blurb | auth-onboarding | legal | P1 | Kitchen address collection does not disclose this becomes the FSSAI-registered… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-026 | vp-onb-stepper-labels | auth-onboarding | legal | P1 | Stepper labels show 'Documents' generically — KYC documents required by RBI/PML… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-027 | vp-onb-ops-pricing-title | auth-onboarding | legal | P1 | 'You can change these anytime' — pricing/prep-time promise without reference to… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-028 | dp-auth-mode-driver-cta | auth-onboarding | legal | P1 | 'I'm a Driver' / 'Login or sign up to deliver with Fe3dr' — driver-employment c… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-029 | dp-auth-google-button | auth-onboarding | legal | P1 | Driver OAuth signup (Google) — no separate driver-specific consent for KYC data… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-032 | dp-auth-step3-doc-labels | auth-onboarding | legal | P1 | 'Police Verification' document slot — uploading police verification certificate… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-034 | dp-auth-step3-section-headers | auth-onboarding | legal | P1 | 'Vehicle Photos — All Angles / Optional' framing for photos that the platform l… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-036 | dp-auth-step-payout-upi | auth-onboarding | legal | P1 | UPI ID collection with no NPCI-mandated VPA verification disclosure | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-037 | dp-auth-step1-emergency-name | auth-onboarding | legal | P1 | Emergency contact name/phone collected — third-party personal data with no cons… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-041 | dp-auth-step4-h | auth-onboarding | legal | P1 | 'A small subscription to keep Fe3dr running — you keep every rupee you earn' —… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-043 | dp-auth-step4-trial | auth-onboarding | legal | P1 | '{n}-day free trial / Try everything free. No card needed.' — auto-conversion t… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-044 | dp-auth-step4-threshold | auth-onboarding | legal | P1 | 'No charge until you earn {amount}' creates conditional billing promise — billi… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-045 | dp-auth-step4-billing-howto | auth-onboarding | legal | P1 | 4-step billing explanation is dense legal-info copy presented inside onboarding… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-046 | dp-auth-step4-secure-pay | auth-onboarding | legal | P1 | 'Payments are securely processed via {gateway}' — gateway is a sub-processor; D… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-049 | dp-auth-step5-plan-summary | auth-onboarding | legal | P1 | 'Plan selected. Payments handled securely via Razorpay.' — gateway hardcoded; i… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-051 | dp-auth-status-descriptions | auth-onboarding | legal | P1 | 'Our team is currently reviewing your application. This usually takes 1-2 busin… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-055 | ap-auth-login-instruction | auth-onboarding | legal | P1 | Admin login has no security warning notice (unauthorized access, monitoring) —… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-056 | ap-auth-totp-enroll-help | auth-onboarding | legal | P1 | 'Your organization requires 2FA for admins' — TOTP is correct security practice… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-058 | mc-auth-login-title | auth-onboarding | legal | P1 | Mobile customer login 'Welcome back' — local title override but underlying shar… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-061 | mc-onb-step1-labels | auth-onboarding | legal | P1 | Mobile-customer collects 'Phone Number' as part of onboarding — TRAI commercial… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-062 | mv-auth-login-title | auth-onboarding | legal | P1 | Mobile vendor login title 'Welcome back' — same shared-screen audit gap as mobi… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-066 | mv-onb-policies-heading | auth-onboarding | legal | P1 | 'Policies / Review and accept terms' is bare-minimum policy-acceptance UI; deep… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-067 | mv-onb-review-section-policies | auth-onboarding | legal | P1 | Review screen shows 'Policies' section but inventory doesn't confirm read-only… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-068 | mv-onb-review-field-labels | auth-onboarding | legal | P1 | 'Terms Accepted / Cancellation Policy' shown as a review row — but does not lin… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-071 | md-auth-004 | auth-onboarding | legal | P1 | 'Biometric authentication failed' — biometric data is sensitive; DPDP §8(4) acc… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-077 | md-onb-046 | auth-onboarding | legal | P1 | 'Vehicle RC (optional)' — RC is regulatory document; making it optional in onbo… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-078 | md-onb-053 | auth-onboarding | legal | P1 | 'Upload PDF' as document-upload — PDF can carry embedded metadata/PII; no scrub… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-079 | md-onb-054 | auth-onboarding | legal | P1 | 'Camera permission is required to capture documents' — DPDP requires purpose-sp… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-080 | md-onb-058 | auth-onboarding | legal | P1 | 'Please upload Driving License and ID Proof to continue' — gating requires DL b… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-084 | md-onb-070 | auth-onboarding | legal | P1 | IFSC Code collection — needs RBI/IFSC validation, no validation disclosure | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-085 | md-onb-076 | auth-onboarding | legal | P1 | UPI ID collection — same NPCI VPA verification concern as dp-auth-step-payout-u… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-090 | md-onb-109 | auth-onboarding | legal | P1 | 'Application Not Approved' + 'Unfortunately your application was not approved a… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-091 | md-onb-115 | auth-onboarding | legal | P1 | 'Estimated review time: 24–48 hours' — same SLA-promise concern as dp-auth-stat… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-093 | md-onb-014 | auth-onboarding | legal | P1 | Emergency contact phone '10-digit mobile number' — same third-party-consent con… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-096 | md-onb-038 | auth-onboarding | legal | P1 | 'Driving License Number' field — manual entry vs DigiLocker pull | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-097 | md-onb-035 | auth-onboarding | legal | P1 | 'Vehicle Registration Number' — regex-only check (e.g. MH12AB1234); no MV Act v… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-101 | web-auth-register-form-fields | auth-onboarding | legal | P1 | No CAPTCHA/bot-protection disclosure on signup — bot signup creates compliance… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-102 | web-auth-register-form-fields | auth-onboarding | legal | P1 | Email verification flow not inventoried — confirming email controls account-tak… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-105 | web-auth-register-heading | auth-onboarding | legal | P1 | Across all six apps — no language-choice disclosure or vernacular T&C availabil… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-106 | web-auth-register-form-fields | auth-onboarding | legal | P1 | Account deletion / DPDP §12 erasure right — no visible path from signup/onboard… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-023 | vp-auth-register-hero-sub | auth-onboarding | technical-writer | P1 | AI-SLOP: 'Join thousands of home chefs earning with Fe3dr.' Unverified social-p… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-065 | dp-auth-step4-zero-commission | auth-onboarding | technical-writer | P1 | Block: 'We don't take any commission from your earnings… 100%… We're here to he… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-075 | dp-auth-step5-terms | auth-onboarding | technical-writer | P1 | 'I agree to the Terms & Conditions, Privacy Policy, and confirm that all inform… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-084 | ap-auth-login-feature-list | auth-onboarding | technical-writer | P1 | Six-feature sales-style bullet list on an INTERNAL admin login page is voice-dr… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-122 | md-auth-004 | auth-onboarding | technical-writer | P1 | 'Biometric authentication failed' — missing 'what to do' per §4 errors rule. Dr… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-155 | md-onb-018 | auth-onboarding | technical-writer | P1 | DOB placeholder 'MM/DD/YYYY' is US format. INCONSISTENT with India-only validat… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-175 | md-onb-103 | auth-onboarding | technical-writer | P1 | 'I accept the Terms of Service and Privacy Policy' — link wiring not confirmed… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-179 | dp-auth-step-payout-h | auth-onboarding | technical-writer | P1 | ORPHAN component: inventory notes the payout step file is present but 'not wire… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-181 | vp-auth-register-hero-sub | auth-onboarding | technical-writer | P1 | AI-SLOP IN ENTRY SURFACES — comprehensive scan: vp-auth-register-hero-sub ('Joi… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-182 | dp-auth-onboarding-step-labels | auth-onboarding | technical-writer | P1 | STEPPER COUNT DRIFT: delivery-portal web has 5 steps (Personal Info / Vehicle D… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-005 | web-auth-register-benefits | auth-onboarding | business-analyst | P2 | Benefits panel copy is generic — 'Authentic homemade food' and 'Support local h… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-006 | web-auth-register-email-form | auth-onboarding | business-analyst | P2 | 5-field email registration form (First name, Last name, Email, Password, Confir… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-007 | web-auth-register-email-form | auth-onboarding | business-analyst | P2 | Password confirmation field adds friction but no user-visible strength indicato… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-008 | web-auth-register-tos | auth-onboarding | business-analyst | P2 | Terms of Service consent is a tiny 12px text block — no explicit 'data use' rea… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-010 | web-auth-login-error | auth-onboarding | business-analyst | P2 | Generic auth error message 'Something went wrong. Please try again.' on non-ses… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-014 | vendor-portal-register-email-cta | auth-onboarding | business-analyst | P2 | Email registration CTA button reads 'Register with Email' — inconsistent with '… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-015 | vendor-portal-login-heading | auth-onboarding | business-analyst | P2 | Vendor login subline 'Sign in to manage your menu, orders, and earnings' is tas… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-016 | vendor-portal-login-email-cta | auth-onboarding | business-analyst | P2 | Email login button reads 'Sign in with Email' — sentence-cased but capitalises… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-019 | vendor-portal-onboarding-stepper | auth-onboarding | business-analyst | P2 | Step 5 label 'Policies & Review' combines two distinct concerns in one step lab… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-022 | vendor-portal-onboarding-step-personal | auth-onboarding | business-analyst | P2 | Address Line 2 and Landmark are separate optional fields — creates clutter alon… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-023 | vendor-portal-onboarding-step-kitchen | auth-onboarding | business-analyst | P2 | Description field hint 'Min 20 characters. This is shown to customers on your p… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-024 | vendor-portal-onboarding-step-kitchen | auth-onboarding | business-analyst | P2 | 'Years of Cooking Experience' select defaults to empty 'Select experience' — no… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-026 | vendor-portal-onboarding-step-operations | auth-onboarding | business-analyst | P2 | Delivery Radius field hint 'How far can you deliver or allow pickup from' is am… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-028 | vendor-portal-onboarding-step-operations | auth-onboarding | business-analyst | P2 | Operating Hours grid shows 7 days with individual time pickers — all days start… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-032 | vendor-portal-onboarding-step-documents | auth-onboarding | business-analyst | P2 | PAN number text field validates only client-side; no inline format mask or vali… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-035 | vendor-portal-onboarding-step-policies | auth-onboarding | business-analyst | P2 | Platform TOS policy checkbox links 'Terms of Service, Privacy Policy, and Vendo… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-036 | vendor-portal-onboarding-step-policies | auth-onboarding | business-analyst | P2 | Payout Details card says 'You can set up your bank account or UPI details... af… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-037 | vendor-portal-onboarding-review | auth-onboarding | business-analyst | P2 | Review step missing documents show technical slug names 'pan_card', 'aadhaar_ca… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-042 | delivery-portal-onboarding-step-progress | auth-onboarding | business-analyst | P2 | Step 4 is labelled 'Plan' in the StepProgress component — label is ambiguous; d… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-043 | delivery-portal-onboarding-step-progress | auth-onboarding | business-analyst | P2 | StepProgress has no estimated time per step or total time indication | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-045 | delivery-portal-onboarding-subscription | auth-onboarding | business-analyst | P2 | 'No card needed' on trial CTA — trial card-skip promise is good but undermined… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-046 | delivery-portal-onboarding-subscription | auth-onboarding | business-analyst | P2 | Subscription 'how billing works' list item 2 says 'we won't charge you until yo… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-047 | delivery-portal-onboarding-review | auth-onboarding | business-analyst | P2 | Review step Terms checkbox is plain text 'Terms & Conditions' and 'Privacy Poli… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-048 | delivery-portal-onboarding-review | auth-onboarding | business-analyst | P2 | Subscription Plan review card shows static copy 'Plan selected. Payments handle… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-051 | delivery-portal-onboarding-status | auth-onboarding | business-analyst | P2 | 'Auto-refreshing every 30 seconds' message is an internal system detail exposed… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-052 | delivery-portal-onboarding-status | auth-onboarding | business-analyst | P2 | Support contact for rejected driver shows only an email address — no expected r… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-054 | delivery-portal-onboarding-payout | auth-onboarding | business-analyst | P2 | Payout form CTA button says 'Continue' — no confirmation of what the driver is… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-055 | vendor-portal-register-heading | auth-onboarding | business-analyst | P2 | Vendor portal uses 'vendor' terminology in UI — style guide mandates 'chef' for… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-057 | delivery-portal-onboarding-step-progress | auth-onboarding | business-analyst | P2 | Payout step is listed in StepProgress labels but StepPayoutDetails is not in th… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-059 | vendor-portal-onboarding-step-policies | auth-onboarding | business-analyst | P2 | Completion state copy 'You're all set to submit your application!' uses an excl… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-026 | ap-auth-login-instruction | auth-onboarding | brand-voice | P2 | 'Access the administration dashboard with your internal credentials' — verbose,… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-029 | mv-onb-personal-fullname-label | auth-onboarding | brand-voice | P2 | Mobile-vendor field labels Title Case ('Full Name *', 'Phone Number *', 'Email… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-030 | mc-onb-step1-labels | auth-onboarding | brand-voice | P2 | Mobile-customer onboarding field labels Title Case ('First Name', 'Last Name',… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-031 | vp-onb-personal-fields | auth-onboarding | brand-voice | P2 | Email/Phone field labels drift: 'Email Address' (vp + mv), 'Email' (web auth),… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-032 | mc-onb-step1-subtitle | auth-onboarding | brand-voice | P2 | Customer onboarding step 1 subtitle 'We need a few details to set up your accou… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-033 | mc-onb-step2-subtitle | auth-onboarding | brand-voice | P2 | 'Where should we deliver your orders?' phrases first-person plural correctly bu… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-034 | web-auth-onboarding-heading | auth-onboarding | brand-voice | P2 | Web onboarding header 'Complete Your Profile' (Title Case) drifts from chef hea… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-042 | md-onb-081 | auth-onboarding | brand-voice | P2 | Generic 'Failed to save payout details. Please try again.' pattern repeated acr… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-043 | vp-auth-register-subheading | auth-onboarding | brand-voice | P2 | 'Create your vendor account to start selling home-cooked meals' uses 'vendor ac… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-044 | vp-auth-login-register-link | auth-onboarding | brand-voice | P2 | 'Want to start selling? Register as a vendor' — 'Register' is banned ('Sign up'… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-048 | md-onb-018 | auth-onboarding | brand-voice | P2 | Date placeholder 'MM/DD/YYYY' on driver mobile DOB field — inconsistent with In… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-050 | mv-onb-pending-title | auth-onboarding | brand-voice | P2 | 'Application Submitted!' on chef mobile pending screen uses an exclamation mark… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-055 | web-auth-register-heading | auth-onboarding | brand-voice | P2 | 'Join Fe3dr Today' heading on register left panel — 'Today' adds artificial urg… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-056 | vp-auth-login-features-list | auth-onboarding | brand-voice | P2 | Document verification & compliance' bullet phrasing for chef login is enterpris… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-058 | mv-onb-kitchen-desc-ph | auth-onboarding | brand-voice | P2 | 'Describe your kitchen, specialties, and cooking style (min 50 characters)' — 1… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-059 | dp-auth-step4-h | auth-onboarding | brand-voice | P2 | 'A small subscription to keep Fe3dr running — you keep every rupee you earn.' —… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-010 | web-auth-login-social | auth-onboarding | legal | P2 | Login OAuth flow has no T&C/Privacy disclosure visible (unlike register page fo… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-015 | web-auth-onboarding-step-preferences | auth-onboarding | legal | P2 | 'Household Size' collected — household composition is profile-enriching data, n… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-017 | vp-auth-login-hero-heading | auth-onboarding | legal | P2 | Marketing claim on auth screen ('Grow your home kitchen business') is pre-contr… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-020 | vp-auth-register-hero-sub | auth-onboarding | legal | P2 | 'Your kitchen, your recipes, your rules' contradicts platform T&C which inevita… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-033 | dp-auth-step3-bicycle-hint | auth-onboarding | legal | P2 | 'Since you're using a bicycle, driving license, vehicle RC, and insurance are n… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-039 | dp-auth-step1-referral | auth-onboarding | legal | P2 | Referral code field — pre-onboarding promise of referral reward must match Vend… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-040 | dp-auth-step1-referral-valid | auth-onboarding | legal | P2 | 'Referred by {referrerName}' shows referrer's name — DPDP §5/§6 concern about d… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-050 | dp-auth-onboarding-header | auth-onboarding | legal | P2 | 'Become a Delivery Partner' — 'Delivery Partner' is gig-economy terminology tha… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-054 | ap-auth-login-feature-list | auth-onboarding | legal | P2 | Admin login left-pane features list contains sales-style copy on internal porta… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-060 | mc-onb-step3-title | auth-onboarding | legal | P2 | Mobile customer preferences subtitle 'Select your favourite cuisines to get per… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-069 | md-auth-002 | auth-onboarding | legal | P2 | Developer-shaped error 'Google sign-in failed: no ID token' could surface to us… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-070 | md-auth-003 | auth-onboarding | legal | P2 | 'Apple sign-in failed: no identity token' — same as above; Apple has additional… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-092 | md-onb-116 | auth-onboarding | legal | P2 | 'We'll notify you once your application is approved' — notification commitment… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-094 | md-onb-018 | auth-onboarding | legal | P2 | DOB placeholder 'MM/DD/YYYY' inconsistent with India-only validation context —… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-002 | web-auth-login-email-fields | auth-onboarding | technical-writer | P2 | 'Forgot password?' is fine as a link but should be styled and worded as one phr… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-004 | web-auth-login-heading | auth-onboarding | technical-writer | P2 | 'Don't have an account? Sign up' uses inline question + CTA. Style guide prefer… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-006 | web-auth-register-form-fields | auth-onboarding | technical-writer | P2 | Helper text 'Min. 8 characters' uses abbreviation. Style guide prefers plain En… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-008 | web-auth-onboarding-step-address | auth-onboarding | technical-writer | P2 | Title 'Delivery Address' uses Title Case. Style guide requires sentence case fo… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-009 | web-auth-onboarding-step-address | auth-onboarding | technical-writer | P2 | Placeholder 'House / flat / building number, street' is conversational and long… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-011 | web-auth-onboarding-step-basic | auth-onboarding | technical-writer | P2 | Title 'Basic Information' is Title Case; sentence case per style guide. Also ge… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-013 | web-auth-onboarding-step-preferences | auth-onboarding | technical-writer | P2 | Five sub-section titles use Title Case: 'Dietary Preferences', 'Food Allergies'… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-014 | web-auth-onboarding-heading | auth-onboarding | technical-writer | P2 | 'Complete Your Profile' Title Case; should be sentence case. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-017 | vp-auth-forgot-redirect | auth-onboarding | technical-writer | P2 | Interim 'Redirecting to password reset...' uses three-dot ellipsis. Also no exp… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-018 | vp-auth-login-hero-heading | auth-onboarding | technical-writer | P2 | Hero heading 'Grow your home kitchen business' is marketing-flavoured but accep… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-019 | vp-auth-login-features-list | auth-onboarding | technical-writer | P2 | Five features list uses ampersand inconsistently and runs long: 'Document verif… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-021 | vp-auth-login-email-btn | auth-onboarding | technical-writer | P2 | 'Sign in with Email' uses Title Case on 'Email'. Inconsistent with web ('Sign i… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-025 | vp-auth-register-email-btn | auth-onboarding | technical-writer | P2 | 'Register with Email' Title Case on 'Email'. Inconsistent with 'Sign up with em… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-026 | vp-onb-kitchen-title | auth-onboarding | technical-writer | P2 | 'Kitchen Information' Title Case → sentence case per style guide. Also 'Informa… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-028 | vp-onb-ops-pricing-title | auth-onboarding | technical-writer | P2 | 'Delivery & Pricing' uses ampersand and Title Case. Also section heading + help… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-029 | vp-onb-personal-title | auth-onboarding | technical-writer | P2 | 'Personal Details' Title Case. Field is KYC-collection but title doesn't signal… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-030 | vp-onb-personal-fields | auth-onboarding | technical-writer | P2 | All four field labels use Title Case: 'Full Name', 'Phone Number', 'Email Addre… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-031 | vp-onb-personal-address-title | auth-onboarding | technical-writer | P2 | 'Kitchen Address' Title Case. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-033 | vp-onb-header-title | auth-onboarding | technical-writer | P2 | 'Set Up Your Kitchen' Title Case. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-034 | vp-onb-stepper-labels | auth-onboarding | technical-writer | P2 | Stepper labels mix Title Case ('Personal Info', 'Kitchen Details', 'Operations'… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-035 | dp-auth-mode-staff-cta | auth-onboarding | technical-writer | P2 | Button label 'I'm Staff' uses Title Case. Driver-facing should be glanceable bu… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-036 | dp-auth-mode-staff-sub | auth-onboarding | technical-writer | P2 | Sub 'Fleet managers & delivery operations' — ampersand. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-037 | dp-auth-email-button | auth-onboarding | technical-writer | P2 | 'Sign in with Email' Title Case 'Email'. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-038 | dp-auth-mode-driver-cta | auth-onboarding | technical-writer | P2 | 'I'm a Driver' Title Case 'Driver'. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-040 | dp-auth-login-subtitle-driver | auth-onboarding | technical-writer | P2 | 'Driver login or sign up' — 'login' banned vocabulary, also drift from sibling… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-041 | dp-auth-step1-h | auth-onboarding | technical-writer | P2 | 'Personal Information' Title Case. Driver-facing — should be telegraphic AND se… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-042 | dp-auth-step1-city | auth-onboarding | technical-writer | P2 | Label 'City *' includes asterisk inline with label. Style guide §4: label stays… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-043 | dp-auth-step1-emergency-name | auth-onboarding | technical-writer | P2 | 'Emergency Contact Name *' — Title Case + inline asterisk. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-044 | dp-auth-step1-emergency-phone | auth-onboarding | technical-writer | P2 | 'Emergency Phone *' Title Case + inline asterisk. Placeholder '+91 9876543210'… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-045 | dp-auth-step1-dob | auth-onboarding | technical-writer | P2 | 'Date of Birth' Title Case. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-046 | dp-auth-step1-vehicle-type | auth-onboarding | technical-writer | P2 | 'Vehicle Type *' Title Case + asterisk. Options 'Scooter/Motorcycle' uses slash… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-048 | dp-auth-onboarding-step-labels | auth-onboarding | technical-writer | P2 | Step labels 'Personal Info / Vehicle Details / Documents / Plan / Review' — Tit… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-049 | dp-auth-step2-h-vehicle | auth-onboarding | technical-writer | P2 | Heading 'Bicycle Details / Vehicle Details' + sub 'Tell us about your bicycle/v… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-050 | dp-auth-step2-carrier-q | auth-onboarding | technical-writer | P2 | 'Can your bicycle carry a delivery box / bag? *' — 11 words, exceeds driver lim… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-051 | dp-auth-step2-carrier-warning | auth-onboarding | technical-writer | P2 | 'You can still proceed, but you'll need to attach a carrier or use a delivery b… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-052 | dp-auth-step2-motor-fields | auth-onboarding | technical-writer | P2 | All four labels Title Case: 'Vehicle Make', 'Vehicle Model', 'Vehicle Year', 'V… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-053 | dp-auth-step2-reg-number | auth-onboarding | technical-writer | P2 | 'Vehicle Registration Number *' — Title Case + inline asterisk. Long label for… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-054 | dp-auth-step2-license | auth-onboarding | technical-writer | P2 | 'Driving License Number *' — Title Case + asterisk. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-055 | dp-auth-step3-section-headers | auth-onboarding | technical-writer | P2 | Section headers Title Case: 'Personal Documents', 'Vehicle Documents', 'Vehicle… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-056 | dp-auth-step3-h | auth-onboarding | technical-writer | P2 | 'Documents & Photos' Title Case + ampersand. Sub 'Upload your documents and veh… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-057 | dp-auth-step3-bicycle-hint | auth-onboarding | technical-writer | P2 | 'Since you're using a bicycle, driving license, vehicle RC, and insurance are n… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-060 | dp-auth-step-payout-h | auth-onboarding | technical-writer | P2 | 'Payout Details' Title Case. Sub 'How would you like to receive your earnings?'… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-061 | dp-auth-step-payout-methods | auth-onboarding | technical-writer | P2 | 'Payout Method' Title Case. 'Bank Transfer' Title Case. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-062 | dp-auth-step-payout-bank-fields | auth-onboarding | technical-writer | P2 | All three labels Title Case + asterisk inline: 'Account Holder Name *', 'Accoun… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-063 | dp-auth-step-payout-upi | auth-onboarding | technical-writer | P2 | 'UPI ID *' inline asterisk. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-064 | dp-auth-step4-h | auth-onboarding | technical-writer | P2 | 'Choose Your Plan' Title Case. Subhead 'A small subscription to keep Fe3dr runn… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-067 | dp-auth-step4-threshold | auth-onboarding | technical-writer | P2 | 'No charge until you earn {amount}' + 'We only bill after you start earning wel… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-068 | dp-auth-step4-billing-howto | auth-onboarding | technical-writer | P2 | 'How billing works:' + 4-step ordered list, 50 words total. Trailing colon on h… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-069 | dp-auth-step4-secure-pay | auth-onboarding | technical-writer | P2 | 'Payments are securely processed via {gateway}. You can change your plan or can… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-070 | dp-auth-step4-submit | auth-onboarding | technical-writer | P2 | 'Start with {interval} Plan' — 'Plan' Title Case + interpolated word: 'Start wi… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-071 | dp-auth-step5-h | auth-onboarding | technical-writer | P2 | 'Review & Submit' ampersand + Title Case. Sub 'Review your information before s… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-072 | dp-auth-step5-sections | auth-onboarding | technical-writer | P2 | Section headers Title Case: 'Personal Information', 'Vehicle Details', 'Documen… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-074 | dp-auth-step5-plan-summary | auth-onboarding | technical-writer | P2 | 'Plan selected. Payments handled securely via Razorpay.' + 'Billing starts only… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-076 | dp-auth-step5-submit | auth-onboarding | technical-writer | P2 | 'Submit Application' Title Case + 2 words OK. Loading 'Submitting...' three-dot… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-077 | dp-auth-onboarding-header | auth-onboarding | technical-writer | P2 | 'Become a Delivery Partner' Title Case. 'Delivery Partner' is style-guide-appro… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-078 | dp-auth-status-h | auth-onboarding | technical-writer | P2 | 'Application Status' + 'Delivery Partner Onboarding' — both Title Case. Driver-… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-079 | dp-auth-status-titles | auth-onboarding | technical-writer | P2 | Status titles Title Case: 'Application Needs Changes', 'Under Review', 'Applica… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-080 | dp-auth-status-descriptions | auth-onboarding | technical-writer | P2 | 'Our team is currently reviewing your application. This usually takes 1-2 busin… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-081 | dp-auth-status-badges | auth-onboarding | technical-writer | P2 | Status badges Title Case: 'Rejected', 'In Review', 'Submitted'. Badge labels ac… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-082 | dp-auth-status-fix-resubmit | auth-onboarding | technical-writer | P2 | 'Fix & Resubmit' ampersand + Title Case. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-085 | ap-auth-login-heading | auth-onboarding | technical-writer | P2 | 'Admin Sign In' Title Case. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-087 | ap-auth-totp-verify-heading | auth-onboarding | technical-writer | P2 | 'Enter your 6-digit code' is acceptable. 4 words. Admin-direct. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-088 | ap-auth-totp-verify-help | auth-onboarding | technical-writer | P2 | 'Open your authenticator app and enter the code shown for Fe3dr Admin.' — 13 wo… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-090 | ap-auth-totp-enroll-help | auth-onboarding | technical-writer | P2 | 'Your organization requires 2FA for admins. Scan this QR with Google Authentica… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-091 | ap-secsettings-2fa-enroll | auth-onboarding | technical-writer | P2 | Multi-string concatenated entry: 'Scan with your authenticator; Or type this ke… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-092 | ap-secsettings-2fa-disable | auth-onboarding | technical-writer | P2 | 'Disable 2FA; Requires your password and a current 6-digit code.; Password; Dis… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-094 | mc-onb-step2-labels | auth-onboarding | technical-writer | P2 | Field labels 'Address / City / State / Pincode' — single words OK, but 'Pincode… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-097 | mc-onb-step1-labels | auth-onboarding | technical-writer | P2 | 'First Name / Last Name / Phone Number' Title Case. Web equivalent (TW-011 area… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-100 | mv-onb-docs-heading | auth-onboarding | technical-writer | P2 | 'Documents' is fine as a screen heading. 'Upload your identity and FSSAI docume… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-101 | mv-onb-docs-id-slot | auth-onboarding | technical-writer | P2 | 'ID Proof' Title Case. Inconsistent with delivery's 'ID Proof' (also Title Case… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-102 | mv-onb-docs-fssai-slot | auth-onboarding | technical-writer | P2 | 'FSSAI License' Title Case. FSSAI is acronym (uppercase OK). 'License' should b… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-103 | mv-onb-kitchen-heading | auth-onboarding | technical-writer | P2 | 'Kitchen Details' Title Case. Sub 'Tell us about your kitchen' OK. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-104 | mv-onb-kitchen-business-label | auth-onboarding | technical-writer | P2 | 'Business Name *' Title Case + inline asterisk. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-106 | mv-onb-kitchen-cuisines-label | auth-onboarding | technical-writer | P2 | 'Cuisine Types *' Title Case + asterisk. 'Types' is filler. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-107 | mv-onb-kitchen-desc-label | auth-onboarding | technical-writer | P2 | 'Description *' is fine — single word — but Title Case + asterisk. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-108 | mv-onb-kitchen-desc-ph | auth-onboarding | technical-writer | P2 | 'Describe your kitchen, specialties, and cooking style (min 50 characters)' — 1… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-110 | mv-onb-ops-hours-label | auth-onboarding | technical-writer | P2 | 'Operating Hours' Title Case. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-111 | mv-onb-ops-preptime-label | auth-onboarding | technical-writer | P2 | 'Prep Time' Title Case. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-112 | mv-onb-ops-radius-label | auth-onboarding | technical-writer | P2 | 'Service Radius (km)' Title Case. '(km)' qualifier in label OK as unit indicato… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-114 | mv-onb-personal-heading | auth-onboarding | technical-writer | P2 | 'Personal Information' Title Case. Matches mobile-delivery's same wrong pattern. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-115 | mv-onb-personal-fullname-label | auth-onboarding | technical-writer | P2 | 'Full Name *' Title Case + inline asterisk. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-116 | mv-onb-personal-phone-label | auth-onboarding | technical-writer | P2 | 'Phone Number *' Title Case + asterisk. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-117 | mv-onb-personal-email-label | auth-onboarding | technical-writer | P2 | 'Email Address' Title Case. Also 'Address' is redundant — 'Email' is sufficient. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-118 | mv-onb-review-section-personal | auth-onboarding | technical-writer | P2 | 'Personal Information' (review section label, inventory notes 'uppercase styled… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-119 | mv-onb-review-field-labels | auth-onboarding | technical-writer | P2 | Review labels mix Title Case: 'Full Name / Phone / Email / Business Name / Cuis… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-123 | md-auth-005 | auth-onboarding | technical-writer | P2 | 'No saved session found. Please log in with email.' — 'log in' is banned vocabu… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-124 | md-onb-002 | auth-onboarding | technical-writer | P2 | 'Driver Onboarding' Title Case (header fallback title). | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-125 | md-onb-042 | auth-onboarding | technical-writer | P2 | 'Upload Documents' Title Case. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-127 | md-onb-044 | auth-onboarding | technical-writer | P2 | 'Driving License' Title Case + American spelling. en-IN should be 'licence'. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-129 | md-onb-049 | auth-onboarding | technical-writer | P2 | 'Uploading...' three-dot ellipsis. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-130 | md-onb-053 | auth-onboarding | technical-writer | P2 | Alert title 'Permission Required' Title Case. Body 'Camera permission is requir… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-131 | md-onb-055 | auth-onboarding | technical-writer | P2 | Alert title 'Upload Error' Title Case + generic. Body 'Upload failed. Please tr… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-132 | md-onb-057 | auth-onboarding | technical-writer | P2 | Alert title 'Required Documents' Title Case. Body 'Please upload Driving Licens… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-133 | md-onb-059 | auth-onboarding | technical-writer | P2 | 'Payout Details' Title Case. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-135 | md-onb-061 | auth-onboarding | technical-writer | P2 | 'Bank Account' Title Case (tab label). | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-136 | md-onb-063 | auth-onboarding | technical-writer | P2 | 'Account Number' Title Case. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-137 | md-onb-065 | auth-onboarding | technical-writer | P2 | 'Confirm Account Number' Title Case. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-138 | md-onb-070 | auth-onboarding | technical-writer | P2 | 'IFSC Code' Title Case. IFSC is acronym (uppercase OK), 'Code' should be senten… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-139 | md-onb-073 | auth-onboarding | technical-writer | P2 | 'Bank Name' Title Case. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-140 | md-onb-079 | auth-onboarding | technical-writer | P2 | 'Your payout details are encrypted and never stored on this device. They are us… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-142 | md-onb-109 | auth-onboarding | technical-writer | P2 | 'Application Not Approved' Title Case + euphemism. Inventory note: 'Rejected st… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-143 | md-onb-110 | auth-onboarding | technical-writer | P2 | 'Unfortunately your application was not approved at this time.' — 9 words. 'Unf… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-144 | md-onb-113 | auth-onboarding | technical-writer | P2 | 'Application Under Review' Title Case. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-145 | md-onb-114 | auth-onboarding | technical-writer | P2 | 'Your application has been submitted and is being reviewed by our team.' — 12 w… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-146 | md-onb-115 | auth-onboarding | technical-writer | P2 | 'Estimated review time: 24–48 hours' — colon after label (§4 form labels no col… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-147 | md-onb-116 | auth-onboarding | technical-writer | P2 | 'We'll notify you once your application is approved. This page checks for updat… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-149 | md-onb-003 | auth-onboarding | technical-writer | P2 | 'Personal Information' Title Case (matches mobile-vendor, vendor-portal, delive… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-150 | md-onb-005 | auth-onboarding | technical-writer | P2 | 'City' OK single word. Other fields TW-151 onwards. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-151 | md-onb-008 | auth-onboarding | technical-writer | P2 | 'Vehicle Type' Title Case. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-152 | md-onb-010 | auth-onboarding | technical-writer | P2 | 'Emergency Contact Name' Title Case. Inconsistent with delivery-portal (which a… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-154 | md-onb-013 | auth-onboarding | technical-writer | P2 | 'Emergency Contact Phone' Title Case. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-156 | md-onb-016 | auth-onboarding | technical-writer | P2 | 'Date of Birth' Title Case. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-159 | md-onb-021 | auth-onboarding | technical-writer | P2 | 'Vehicle Details' Title Case. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-160 | md-onb-022 | auth-onboarding | technical-writer | P2 | 'Vehicle type:' — trailing colon violates §4 form labels rule. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-161 | md-onb-023 | auth-onboarding | technical-writer | P2 | 'Vehicle Make' Title Case. Also 'Vehicle' prefix redundant in vehicle screen. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-162 | md-onb-026 | auth-onboarding | technical-writer | P2 | 'Vehicle Model' Title Case + redundant prefix. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-163 | md-onb-029 | auth-onboarding | technical-writer | P2 | 'Vehicle Year' Title Case + redundant prefix. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-164 | md-onb-032 | auth-onboarding | technical-writer | P2 | 'Vehicle Color' Title Case + redundant prefix + American spelling. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-165 | md-onb-035 | auth-onboarding | technical-writer | P2 | 'Vehicle Registration Number' Title Case + verbose. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-166 | md-onb-038 | auth-onboarding | technical-writer | P2 | 'Driving License Number' Title Case + American spelling 'License'. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-168 | md-onb-082 | auth-onboarding | technical-writer | P2 | 'Choose Your Plan' Title Case. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-169 | md-onb-083 | auth-onboarding | technical-writer | P2 | 'Select a subscription plan that works best for you' — 8 words. Driver-OK. 'tha… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-171 | md-onb-088 | auth-onboarding | technical-writer | P2 | Alert title 'Select a Plan' Title Case + identical to button text — confusing.… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-172 | md-onb-093 | auth-onboarding | technical-writer | P2 | 'Review Your Application' Title Case. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-176 | md-onb-104 | auth-onboarding | technical-writer | P2 | Alert title 'Terms Required' Title Case. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-177 | md-onb-107 | auth-onboarding | technical-writer | P2 | Alert title 'Submission Error' Title Case + generic. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-178 | md-onb-095 | auth-onboarding | technical-writer | P2 | Review section headers Title Case: 'Personal Info', 'Vehicle Details', 'Documen… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-180 | md-onb-018 | auth-onboarding | technical-writer | P2 | CROSS-APP DATE FORMAT DRIFT: mobile-delivery uses 'MM/DD/YYYY' (US) for DOB pla… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-183 | web-auth-onboarding-heading | auth-onboarding | technical-writer | P2 | CUSTOMER ONBOARDING STEP COUNT: web shows 'Step {n} of 3', mobile-customer show… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-009 | web-auth-login-heading | auth-onboarding | business-analyst | P3 | 'Welcome back' has no function pointer — nothing below it explains what the use… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-025 | vendor-portal-onboarding-step-kitchen | auth-onboarding | business-analyst | P3 | 'Meals You Can Prepare Daily' select — option '100+ meals' has no capacity impl… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-027 | vendor-portal-onboarding-step-operations | auth-onboarding | business-analyst | P3 | Delivery Fee field label says 'Delivery Fee' with no explanation of who sets it… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-056 | vendor-portal-onboarding-wizard | auth-onboarding | business-analyst | P3 | Wizard submit toast uses an escaped apostrophe in inline string: 'We\\'ll revie… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-058 | web-auth-login-divider | auth-onboarding | business-analyst | P3 | Social/email divider text is uppercase 'Or' on customer login, lowercase 'or' o… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BA-060 | delivery-portal-onboarding-review | auth-onboarding | business-analyst | P3 | Delivery portal review step submit success toast says 'Application submitted su… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-010 | web-auth-login-heading | auth-onboarding | brand-voice | P3 | 'Welcome back' is consistent across web, vendor-portal, mobile-customer, mobile… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-027 | ap-auth-login-brand-sub | auth-onboarding | brand-voice | P3 | 'Manage users, chefs, orders, and analytics for the Fe3dr platform.' — slight r… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-035 | web-auth-onboarding-heading | auth-onboarding | brand-voice | P3 | Step counter format drift: web onboarding 'Step {n} of 3' inline with mdash sep… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-039 | dp-auth-google-button | auth-onboarding | brand-voice | P3 | Mobile shared LoginScreen 'Continue with Google' + 'Continue with Apple' is can… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-040 | web-auth-login-submit | auth-onboarding | brand-voice | P3 | Loading state strings drift: 'Signing in...' (web) vs 'Submitting...' (driver r… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-046 | vp-onb-personal-fields | auth-onboarding | brand-voice | P3 | 'Pre-filled from your login' hint — mixes verb tense and is unclear. Better: 'F… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-047 | web-auth-login-email-fields | auth-onboarding | brand-voice | P3 | Password placeholder 'Enter your password' is redundant with 'Password' label.… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-049 | mc-onb-step3-subtitle | auth-onboarding | brand-voice | P3 | British spelling 'favourite cuisines to get personalised recommendations' on mo… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-051 | vp-onb-personal-blurb | auth-onboarding | brand-voice | P3 | Two-sentence helper 'Tell us about yourself. This information helps verify your… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-053 | md-onb-084 | auth-onboarding | brand-voice | P3 | 'Recommended' badge on subscription plans — fine voice-wise but flagged because… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-054 | vp-auth-forgot-redirect | auth-onboarding | brand-voice | P3 | 'Redirecting to password reset…' interim screen — stale Keycloak reference per… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-057 | dp-auth-step4-secure-pay | auth-onboarding | brand-voice | P3 | 'Payments are securely processed via {gateway}.' — passive voice, slightly trus… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| BV-060 | web-auth-register-heading | auth-onboarding | brand-voice | P3 | Confirmed: same '500+' AI-slop appears at HomePage line 88, HeroSection line 12… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-053 | dp-auth-status-auto-refresh | auth-onboarding | legal | P3 | 'Auto-refreshing every 30 seconds' — auto-refresh of status page is benign, but… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-072 | md-auth-005 | auth-onboarding | legal | P3 | 'No saved session found. Please log in with email.' — benign UX message; no leg… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-073 | md-onb-001 | auth-onboarding | legal | P3 | 'Step 1 of 6' header stepper — cosmetic; no legal concern except confirming use… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-095 | md-onb-015 | auth-onboarding | legal | P3 | 'Enter a valid 10-digit Indian mobile number' — explicit India-only validation;… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-098 | vp-auth-forgot-redirect | auth-onboarding | legal | P3 | 'Redirecting to password reset...' has stale Keycloak code comment per inventor… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| LEG-099 | dp-auth-login-title | auth-onboarding | legal | P3 | 'Fe3dr Delivery' brand wordmark — confirm trademark protection and consistency | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-001 | web-auth-login-email-fields | auth-onboarding | technical-writer | P3 | Placeholder 'Enter your password' restates the label. Adds no value, fights tra… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-003 | web-auth-login-submit | auth-onboarding | technical-writer | P3 | Loading state uses ellipsis 'Signing in...' which is acceptable, but inconsiste… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-005 | web-auth-register-heading | auth-onboarding | technical-writer | P3 | Heading 'Create your account' is fine; sub-link 'Already have an account? Sign… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-007 | web-auth-register-form-fields | auth-onboarding | technical-writer | P3 | Placeholder 'Re-enter password' on confirm-password field — confirm-password he… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-010 | web-auth-onboarding-step-address | auth-onboarding | technical-writer | P3 | 'Landmark, area (optional)' — '(optional)' in placeholder is anti-pattern. Opti… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-012 | web-auth-onboarding-step-basic | auth-onboarding | technical-writer | P3 | Placeholder example phone '+91 98765 43210' is good but inconsistent across sur… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-015 | web-auth-onboarding-heading | auth-onboarding | technical-writer | P3 | Step progress 'Step {n} of 3' is fine. 'Skip' as a link is ambiguous — skip thi… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-016 | web-auth-onboarding-steps | auth-onboarding | technical-writer | P3 | Step labels 'Basic Info / Your details / Preferences / Food & dietary / Address… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-020 | vp-auth-login-google-btn | auth-onboarding | technical-writer | P3 | 'Continue with Google' is 3 words, verb-first — within style. Cross-app consist… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-022 | vp-auth-login-register-link | auth-onboarding | technical-writer | P3 | 'Want to start selling? Register as a vendor' uses 'vendor' which style guide b… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-024 | vp-auth-register-subheading | auth-onboarding | technical-writer | P3 | Subheading 'Create your vendor account to start selling home-cooked meals' uses… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-027 | vp-onb-kitchen-blurb | auth-onboarding | technical-writer | P3 | 'Tell customers about your kitchen and what makes your food special.' 11 words… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-032 | vp-onb-personal-address-blurb | auth-onboarding | technical-writer | P3 | Helper 'Where your kitchen is located. This is used for delivery radius and cus… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-039 | dp-auth-mode-driver-sub | auth-onboarding | technical-writer | P3 | 'Login or sign up to deliver with Fe3dr' — 'Login' is banned (style guide: alwa… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-047 | dp-auth-step1-referral | auth-onboarding | technical-writer | P3 | Placeholder 'Enter referral code (optional)' duplicates the qualifier '(optiona… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-058 | dp-auth-step3-doc-labels | auth-onboarding | technical-writer | P3 | Long doc list mixes Title Case. Helper for 'Number Plate (clear photo)' — quali… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-059 | dp-auth-step3-size-hints | auth-onboarding | technical-writer | P3 | Hints 'JPEG or PNG only, max 5MB' / 'JPEG, PNG, WebP, or PDF, max 10MB' — inven… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-066 | dp-auth-step4-trial | auth-onboarding | technical-writer | P3 | '{n}-day free trial' + 'Try everything free. No card needed.' — second clause i… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-073 | dp-auth-step5-edit | auth-onboarding | technical-writer | P3 | Mixed CTAs: 'Edit' / 'Change' for the same action on review rows. Inconsistency. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-083 | dp-auth-status-auto-refresh | auth-onboarding | technical-writer | P3 | 'Auto-refreshing every 30 seconds' is informational and acceptable. Inconsisten… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-086 | ap-auth-login-instruction | auth-onboarding | technical-writer | P3 | 'Access the administration dashboard with your internal credentials' is jargon… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-089 | ap-auth-totp-enroll-heading | auth-onboarding | technical-writer | P3 | 'Set up 2FA to continue' acceptable but slightly awkward. Could be tighter. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-093 | mc-onb-step2-subtitle | auth-onboarding | technical-writer | P3 | 'Where should we deliver your orders?' is fine. 6 words. Customer-conversationa… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-095 | mc-onb-step2-placeholders | auth-onboarding | technical-writer | P3 | 'House no., street, area' uses abbreviation 'no.'. Helper text on cramped mobil… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-096 | mc-onb-step3-subtitle | auth-onboarding | technical-writer | P3 | 'Select your favourite cuisines to get personalised recommendations.' — 9 words… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-098 | mc-onb-step1-placeholders | auth-onboarding | technical-writer | P3 | Placeholders 'Enter your first name / Enter your last name / 10-digit mobile nu… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-099 | mv-onb-step1-title | auth-onboarding | technical-writer | P3 | Stack header titles 'Step 1 of 6' through 'Step 6 of 6' — verbose for mobile gl… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-105 | mv-onb-kitchen-business-ph | auth-onboarding | technical-writer | P3 | 'Your kitchen / business name' — slash separates synonyms but reads awkward. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-109 | mv-onb-ops-heading | auth-onboarding | technical-writer | P3 | 'Operations' single word — good. Sub 'Set your working hours and service detail… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-113 | mv-onb-ops-radius-ph | auth-onboarding | technical-writer | P3 | '1–50 km' uses en-dash for range — correct per style guide §6 (distances). | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-126 | md-onb-043 | auth-onboarding | technical-writer | P3 | 'Please upload clear photos or PDFs of your documents' 9 words. 'Please' adds p… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-128 | md-onb-046 | auth-onboarding | technical-writer | P3 | 'Vehicle RC (optional)' — '(optional)' qualifier in label violates §4. Use no a… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-134 | md-onb-060 | auth-onboarding | technical-writer | P3 | 'Choose how you would like to receive your earnings' — 8 words. Driver-OK. Coul… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-141 | md-onb-081 | auth-onboarding | technical-writer | P3 | 'Failed to save payout details. Please try again.' — follows what happened → wh… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-148 | md-onb-001 | auth-onboarding | technical-writer | P3 | Step header 'Step 1 of 6' is mobile-glanceable, driver-OK. Matches mobile-vendo… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-153 | md-onb-011 | auth-onboarding | technical-writer | P3 | Placeholder 'Full name of emergency contact' restates label phrasing. Use examp… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-157 | md-onb-017 | auth-onboarding | technical-writer | P3 | '(optional)' qualifier in label. See TW-010, TW-128. Optional fields should hav… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-158 | md-onb-020 | auth-onboarding | technical-writer | P3 | 'Failed to save personal info. Please try again.' — what happened + what to do.… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-167 | md-onb-041 | auth-onboarding | technical-writer | P3 | 'Failed to save vehicle details. Please try again.' — 'Please' filler. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-170 | md-onb-086 | auth-onboarding | technical-writer | P3 | 'Up to {n} deliveries/month' uses '/month' suffix — fine. Pluralisation when n=… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-173 | md-onb-094 | auth-onboarding | technical-writer | P3 | 'Please review your details before submitting' — 6 words, OK. 'Please' is fille… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-174 | md-onb-099 | auth-onboarding | technical-writer | P3 | Summary row value '{n}/{total} required + RC' is cryptic. 'RC' is acronym, need… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-184 | mc-auth-register-host | auth-onboarding | technical-writer | P3 | Mobile-customer register screen uses shared RegisterScreen from @homechef/mobil… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-185 | mc-auth-login-title | auth-onboarding | technical-writer | P3 | Mobile-customer login title 'Welcome back' is consistent across portals (web, v… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-186 | dp-auth-login-title | auth-onboarding | technical-writer | P3 | Brand wordmark 'Fe3dr Delivery' is consistent with admin-portal brand block. Br… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-187 | dp-auth-login-subtitle-choose | auth-onboarding | technical-writer | P3 | 'Sign in to get started' — inventory note V: '"Sign in" alone is enough'. 5 wor… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-188 | dp-auth-login-subtitle-staff | auth-onboarding | technical-writer | P3 | 'Staff & fleet manager login' — ampersand + 'login' banned vocabulary. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-189 | ap-auth-login-brand-heading | auth-onboarding | technical-writer | P3 | 'Platform Administration' is Title Case + abstract. Admin tone is direct/precis… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-190 | ap-auth-login-brand-sub | auth-onboarding | technical-writer | P3 | 'Manage users, chefs, orders, and analytics for the Fe3dr platform.' — 10 words… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-191 | ap-auth-login-portal-label | auth-onboarding | technical-writer | P3 | 'Admin Portal' Title Case. Logo sub-label. | [findings/auth-onboarding.md](findings/auth-onboarding.md) |
| TW-192 | md-onb-080 | auth-onboarding | technical-writer | P3 | Inventory annotation says 'Select a Plan' alert appears in subscription screen… | [findings/auth-onboarding.md](findings/auth-onboarding.md) |

### Index — core-ux

467 findings. Detail: [findings/core-ux.md](findings/core-ux.md)

| finding_id | surface_id | category | lens | severity | issue (≤80 chars) | detail link |
|---|---|---|---|---|---|---|
| BA-001 | ap-analytics-cards | core-ux | business-analyst | P0 | Admin analytics cards ship hardcoded '--' placeholders with fake change indicat… | [findings/core-ux.md](findings/core-ux.md) |
| BA-002 | ap-analytics-charts | core-ux | business-analyst | P0 | All four admin analytics charts show 'Chart coming soon' placeholder to live op… | [findings/core-ux.md](findings/core-ux.md) |
| BA-003 | vp-ux-settings-order-acceptance | core-ux | business-analyst | P0 | Auto-accept threshold field uses '$' (USD) currency symbol; all orders are INR | [findings/core-ux.md](findings/core-ux.md) |
| BA-004 | web-ux-orderdetail-cancel | core-ux | business-analyst | P0 | Cancel order modal says 'This action cannot be undone' with no mention of refun… | [findings/core-ux.md](findings/core-ux.md) |
| BA-005 | dp-ux-active-cancel-prompt | core-ux | business-analyst | P0 | Active delivery cancellation uses browser native prompt() — no styled UI, no co… | [findings/core-ux.md](findings/core-ux.md) |
| BA-006 | mc-checkout-place-order | core-ux | business-analyst | P0 | Mobile checkout 'Place Order' CTA does not show the total amount — web version… | [findings/core-ux.md](findings/core-ux.md) |
| BV-001 | web-ux-orders-status-labels | core-ux | brand-voice | P0 | `accepted` enum is labelled `Accepted` in 6 places but `Order Confirmed` / `Con… | [findings/core-ux.md](findings/core-ux.md) |
| BV-002 | web-ux-orders-status-labels | core-ux | brand-voice | P0 | `delivering` enum labelled three ways: `Delivering` (chef/admin/vendor), `On th… | [findings/core-ux.md](findings/core-ux.md) |
| BV-003 | web-ux-orders-status-labels | core-ux | brand-voice | P0 | `ready` enum labelled `Ready` in 5 places and `Ready for Pickup` in 3 places —… | [findings/core-ux.md](findings/core-ux.md) |
| BV-004 | mc-order-detail-status-labels | core-ux | brand-voice | P0 | Mobile customer shows `Ready for Pickup` on order detail but `Ready` on OrderCa… | [findings/core-ux.md](findings/core-ux.md) |
| BV-007 | dp-ux-delivery-pickup-label | core-ux | brand-voice | P0 | `PICKUP` in ALL CAPS as label on delivery portal Available Deliveries page viol… | [findings/core-ux.md](findings/core-ux.md) |
| BV-008 | md-core-079 | core-ux | brand-voice | P0 | Mobile-delivery uses `Drop-off` (hyphenated) on delivery detail screen but `Dro… | [findings/core-ux.md](findings/core-ux.md) |
| BV-028 | web-ux-layout-nav | core-ux | brand-voice | P0 | Brand name appears as `Fe3dr`, `HomeChef`, `Home Chef`, and `home-chef` in user… | [findings/core-ux.md](findings/core-ux.md) |
| BV-029 | web-ux-layout-nav | core-ux | brand-voice | P0 | Customer-facing landing copy says `Home Chef` capitalized as if brand: `Explore… | [findings/core-ux.md](findings/core-ux.md) |
| BV-030 | web-ux-layout-nav | core-ux | brand-voice | P0 | Hardcoded `500+ Home Chefs Near You` and `500+ Home Chefs` repeated in 3 custom… | [findings/core-ux.md](findings/core-ux.md) |
| BV-031 | web-ux-layout-nav | core-ux | brand-voice | P0 | `Join thousands of happy customers enjoying homemade food` (web home) and `Join… | [findings/core-ux.md](findings/core-ux.md) |
| BV-042 | vp-ux-settings-order-acceptance | core-ux | brand-voice | P0 | Vendor portal auto-accept threshold label shows `($)` but platform is INR/₹ eve… | [findings/core-ux.md](findings/core-ux.md) |
| BV-065 | ap-layout-nav-reviews | core-ux | brand-voice | P0 | Admin sidebar label `Reviews` routes to `/approvals` and the page is titled `Ap… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-001 | web-ux-checkout-heading | core-ux | legal | P0 | Checkout page H1 is the entire page contract-formation surface but no inline T&… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-002 | web-ux-checkout-payment-section | core-ux | legal | P0 | Payment section discloses 'Powered by Razorpay' but does not identify the merch… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-003 | web-ux-checkout-summary | core-ux | legal | P0 | Order Summary surfaces 'Tax' as an opaque line — no GST registration number, no… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-004 | web-ux-checkout-delivery-time | core-ux | legal | P0 | Delivery-time selector exposes 'Usually 30-45 minutes' as an implicit time guar… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-007 | mc-checkout-totals | core-ux | legal | P0 | Mobile checkout totals show only 'Subtotal / Delivery fee / Free / Total' — no… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-008 | mc-checkout-place-order | core-ux | legal | P0 | Mobile 'Place Order · ₹{total}' CTA has no T&C, Refund Policy, or Cancellation… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-013 | web-ux-orderdetail-sections | core-ux | legal | P0 | Order detail anatomy shows 'Estimated delivery' as a representation but no refu… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-014 | web-ux-orderdetail-cancel | core-ux | legal | P0 | Cancel modal asks 'why you're cancelling' but does not state refund eligibility… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-018 | mc-order-detail-price-rows | core-ux | legal | P0 | Mobile order detail price breakdown shows 'Subtotal / Delivery Fee / Total' — n… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-019 | dp-ux-active-cancel-prompt | core-ux | legal | P0 | Driver cancellation uses native browser prompt() with no terms reference, no au… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-021 | web-ux-chef-menu | core-ux | legal | P0 | Chef Menu Management page has no field for FSSAI-mandated allergen declaration… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-022 | vp-ux-menu-form-title | core-ux | legal | P0 | Menu Item form page title shows 'Edit Menu Item / Add Menu Item' but inventory… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-027 | web-ux-chefdetail-status-badges | core-ux | legal | P0 | Chef detail page shows 'Verified' badge as a claim but inventory does not indic… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-028 | web-ux-chef-profile | core-ux | legal | P0 | Chef profile editor includes 'Business Settings' and 'Basic Information' but th… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-034 | web-ux-browse-filters-dietary | core-ux | legal | P0 | Dietary filter offers 'Vegan / Gluten-Free / Dairy-Free / Keto / Halal' — these… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-040 | web-ux-catering-quotes-detail | core-ux | legal | P0 | Catering Request Details surface mediates a high-value bespoke contract between… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-042 | web-ux-chef-catering | core-ux | legal | P0 | Chef-side catering quote submission ('Submit Quote / Item name / Qty / Price/un… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-046 | vp-ux-settings-payout-section | core-ux | legal | P0 | Payout Details surface shows 'Razorpay Connected / Linked Account' but no settl… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-050 | dp-ux-stripe-intro | core-ux | legal | P0 | Driver Stripe onboarding intro 'Accept delivery payouts in your local currency.… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-055 | ap-exports-cards | core-ux | legal | P0 | Admin Data Exports allow 'User data / All user accounts (id, email, role, creat… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-057 | ap-platsettings-commission-fields | core-ux | legal | P0 | Platform Settings expose 'Service fee (%) / Tax (%) / Chef payout (% of subtota… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-069 | dp-ux-active-mark-as | core-ux | legal | P0 | Driver 'Mark as {Picked Up/In Transit/Delivered}' is the customer-impacting sta… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-070 | md-core-036 | core-ux | legal | P0 | Mobile driver 'Mark as Delivered' (P0 SAFETY tagged in inventory) — same concer… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-084 | web-ux-profile-tabs | core-ux | legal | P0 | Profile sidebar tabs 'Profile / Preferences / Addresses / Payment Methods / Not… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-087 | mv-settings-delete-account | core-ux | legal | P0 | Mobile chef 'Delete Account' destructive action exists (good) but inventory doe… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-088 | md-core-067 | core-ux | legal | P0 | Mobile driver 'Delete Account' — same DPDP/MV Act/Code on Social Security recor… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-089 | mv-onb-review-terms-yes | core-ux | legal | P0 | Mobile chef onboarding review shows 'Terms accepted indicator' as a yes/no but… | [findings/core-ux.md](findings/core-ux.md) |
| TW-001 | mc-order-detail-status-labels | core-ux | technical-writer | P0 | Order status taxonomy drifts across customer screens: detail uses 'Ready for Pi… | [findings/core-ux.md](findings/core-ux.md) |
| TW-002 | web-ux-orders-status-labels | core-ux | technical-writer | P0 | Web customer order status enum exposes 'Picked Up' and 'On the Way' as separate… | [findings/core-ux.md](findings/core-ux.md) |
| TW-003 | ap-delivery-statuses | core-ux | technical-writer | P0 | Admin delivery status taxonomy uses 10 states with title-case styling ('At Pick… | [findings/core-ux.md](findings/core-ux.md) |
| TW-004 | md-core-033 | core-ux | technical-writer | P0 | Driver action label 'Picked Up Order' is past tense used as a future action but… | [findings/core-ux.md](findings/core-ux.md) |
| TW-005 | dp-ux-active-mark-as | core-ux | technical-writer | P0 | Driver-facing action button reads 'Mark as {Picked Up/In Transit/Delivered}' —… | [findings/core-ux.md](findings/core-ux.md) |
| TW-006 | dp-ux-delivery-pickup-label | core-ux | technical-writer | P0 | Driver row labels 'PICKUP' / 'DROPOFF' are ALL CAPS, which the style guide bans… | [findings/core-ux.md](findings/core-ux.md) |
| TW-007 | md-core-079 | core-ux | technical-writer | P0 | 'Drop-off' (hyphenated) appears alongside 'Dropoff' (unhyphenated) in the same… | [findings/core-ux.md](findings/core-ux.md) |
| TW-008 | dp-ux-active-cancel-prompt | core-ux | technical-writer | P0 | Driver cancellation reason uses browser `prompt()` ('Reason for cancellation?')… | [findings/core-ux.md](findings/core-ux.md) |
| TW-009 | vp-ux-settings-order-acceptance | core-ux | technical-writer | P0 | Vendor settings auto-accept threshold label reads 'Auto-accept threshold ($)' —… | [findings/core-ux.md](findings/core-ux.md) |
| TW-010 | mv-settings-change-password | core-ux | technical-writer | P0 | Vendor 'Change Password' setting row routes to the forgot-password flow, not a… | [findings/core-ux.md](findings/core-ux.md) |
| BA-007 | web-ux-cart-checkout-cta | core-ux | business-analyst | P1 | Unauthenticated cart CTA says 'Sign in to Checkout' — splits intent across two… | [findings/core-ux.md](findings/core-ux.md) |
| BA-008 | web-ux-checkout-summary | core-ux | business-analyst | P1 | Service fee at checkout has no explanation tooltip or label context — trust-ero… | [findings/core-ux.md](findings/core-ux.md) |
| BA-009 | web-ux-checkout-delivery-time | core-ux | business-analyst | P1 | 'Usually 30-45 minutes' is a static string unconnected to actual chef prep time… | [findings/core-ux.md](findings/core-ux.md) |
| BA-010 | web-ux-chefdetail-status-badges | core-ux | business-analyst | P1 | 'Verified' badge on chef profile provides no explanation of what verification m… | [findings/core-ux.md](findings/core-ux.md) |
| BA-011 | web-ux-chef-orders-heading | core-ux | business-analyst | P1 | Chef 'Reject' CTA on order card triggers 'cancelled' status with no confirmatio… | [findings/core-ux.md](findings/core-ux.md) |
| BA-012 | web-ux-chef-earnings | core-ux | business-analyst | P1 | 'Your Earnings' in chef order detail shows subtotal - discount + tip — this is… | [findings/core-ux.md](findings/core-ux.md) |
| BA-013 | vp-ux-menu-view-page | core-ux | business-analyst | P1 | 'Under Review' menu item status provides no expected approval turnaround time | [findings/core-ux.md](findings/core-ux.md) |
| BA-014 | web-ux-orders-status-labels | core-ux | business-analyst | P1 | Status 'accepted' maps to label 'Accepted' in OrderCard but 'Confirmed' in Orde… | [findings/core-ux.md](findings/core-ux.md) |
| BA-015 | mc-order-detail-status-labels | core-ux | business-analyst | P1 | Mobile order detail uses 'Confirmed' / 'Ready for Pickup'; OrderCard.tsx uses '… | [findings/core-ux.md](findings/core-ux.md) |
| BA-016 | web-ux-browse-search-input | core-ux | business-analyst | P1 | Browse Chefs no-results state ('No chefs found') does not offer alternative act… | [findings/core-ux.md](findings/core-ux.md) |
| BA-017 | mc-catering-form-labels | core-ux | business-analyst | P1 | Mobile catering form shows raw 'Event Date * (YYYY-MM-DD)' as a label, exposing… | [findings/core-ux.md](findings/core-ux.md) |
| BA-018 | web-ux-layout-nav | core-ux | business-analyst | P1 | MainLayout nav shows 'Logout' (one word) — style guide requires 'Sign out' (two… | [findings/core-ux.md](findings/core-ux.md) |
| BA-019 | mc-profile-logout-button | core-ux | business-analyst | P1 | Mobile customer app uses 'Log Out' (two words but 'Log' not 'Sign') — banned pe… | [findings/core-ux.md](findings/core-ux.md) |
| BA-020 | mv-more-logout-row | core-ux | business-analyst | P1 | Mobile vendor app uses 'Logout' (one word) in the nav row and Alert — both bann… | [findings/core-ux.md](findings/core-ux.md) |
| BA-021 | vp-ux-layout-nav | core-ux | business-analyst | P1 | Vendor portal sidebar uses 'Vendor Portal' and 'Vendor' as fallback identity la… | [findings/core-ux.md](findings/core-ux.md) |
| BA-022 | web-ux-checkout-tip | core-ux | business-analyst | P1 | Tip selector says '100% of your tip goes to the home chef' — this is a strong t… | [findings/core-ux.md](findings/core-ux.md) |
| BA-023 | web-ux-catering-quotes-detail | core-ux | business-analyst | P1 | 'No quotes yet' empty state says 'You'll receive quotes soon!' — urgency/exclam… | [findings/core-ux.md](findings/core-ux.md) |
| BA-024 | web-ux-chef-catering | core-ux | business-analyst | P1 | Chef catering page shows 'No open requests / No pending quotes / No booked even… | [findings/core-ux.md](findings/core-ux.md) |
| BA-025 | mv-onb-pending-rejected-cta | core-ux | business-analyst | P1 | Rejected vendor onboarding shows 'Application Not Approved' + 'Reapply' CTA wit… | [findings/core-ux.md](findings/core-ux.md) |
| BV-005 | ap-orders-filter-status | core-ux | brand-voice | P1 | Admin orders filter uses `Delivering` while admin delivery page uses `In Transi… | [findings/core-ux.md](findings/core-ux.md) |
| BV-006 | ap-orders-filter-status | core-ux | brand-voice | P1 | Filter labelled `All Status` (singular) in admin Orders and Chefs pages; `All S… | [findings/core-ux.md](findings/core-ux.md) |
| BV-009 | ap-notifsettings-categories | core-ux | brand-voice | P1 | Admin notification copy uses `pickup and drop-off notifications` — hyphenated `… | [findings/core-ux.md](findings/core-ux.md) |
| BV-010 | ap-delivery-statuses | core-ux | brand-voice | P1 | Admin delivery statuses use Title Case (`At Pickup`, `At Dropoff`, `Picked Up`,… | [findings/core-ux.md](findings/core-ux.md) |
| BV-011 | dp-ux-active-mark-as | core-ux | brand-voice | P1 | Delivery-portal action label `Mark as {Picked Up/In Transit/Delivered}` uses Ti… | [findings/core-ux.md](findings/core-ux.md) |
| BV-013 | web-ux-checkout-summary | core-ux | brand-voice | P1 | Web checkout CTA is `Place Order` (Title Case), mobile-customer is `Place Order… | [findings/core-ux.md](findings/core-ux.md) |
| BV-016 | web-ux-orderdetail-cancel | core-ux | brand-voice | P1 | Customer-facing destructive action uses `Cancel Order` (Title Case) on web; mob… | [findings/core-ux.md](findings/core-ux.md) |
| BV-017 | dp-ux-active-cancel-btn | core-ux | brand-voice | P1 | Delivery-portal driver-facing `Cancel Delivery` is Title Case and uses `Deliver… | [findings/core-ux.md](findings/core-ux.md) |
| BV-018 | dp-ux-active-cancel-prompt | core-ux | brand-voice | P1 | Cancellation reason prompt uses browser `prompt()` modal — escapes styled UI, b… | [findings/core-ux.md](findings/core-ux.md) |
| BV-019 | md-core-022 | core-ux | brand-voice | P1 | Driver `Accept Delivery` (Title Case) in mobile and `Accept Delivery` (Title Ca… | [findings/core-ux.md](findings/core-ux.md) |
| BV-022 | dp-ux-nav-bottom-items | core-ux | brand-voice | P1 | Driver bottom-nav says `Home` but sidebar says `Dashboard` for the same destina… | [findings/core-ux.md](findings/core-ux.md) |
| BV-023 | dp-ux-nav-partner-items | core-ux | brand-voice | P1 | Sidebar says `Active Delivery` but bottom-nav says `Active`. Same app, same scr… | [findings/core-ux.md](findings/core-ux.md) |
| BV-025 | vp-ux-dashboard-pending-card | core-ux | brand-voice | P1 | Chef-side action chain mixes `Accept`, `Start Preparing`, `Mark Ready`. STYLE-G… | [findings/core-ux.md](findings/core-ux.md) |
| BV-026 | dp-ux-partner-verify-approve | core-ux | brand-voice | P1 | Admin/partner actions are mixed Title Case: `Approve` (1 word OK), `Reject` (1… | [findings/core-ux.md](findings/core-ux.md) |
| BV-032 | web-ux-layout-nav | core-ux | brand-voice | P1 | Web LoginPage testimonial: `Fe3dr has changed how I eat. Finally, real homemade… | [findings/core-ux.md](findings/core-ux.md) |
| BV-033 | web-ux-layout-nav | core-ux | brand-voice | P1 | Web home `The Fe3dr Difference` is generic SaaS-marketing pattern. STYLE-GUIDE… | [findings/core-ux.md](findings/core-ux.md) |
| BV-034 | vp-ux-settings-payout-section | core-ux | brand-voice | P1 | Vendor-portal kitchen setup ships `Payout integration coming soon. These detail… | [findings/core-ux.md](findings/core-ux.md) |
| BV-035 | ap-analytics-cards | core-ux | brand-voice | P1 | Admin analytics ships with `Chart coming soon` placeholder. Inventory notes con… | [findings/core-ux.md](findings/core-ux.md) |
| BV-036 | web-ux-layout-nav | core-ux | brand-voice | P1 | STYLE-GUIDE mandates `Sign out`. Today: `Logout` (5 surfaces), `Log out` (2), `… | [findings/core-ux.md](findings/core-ux.md) |
| BV-037 | web-ux-layout-nav | core-ux | brand-voice | P1 | `Login` and `Log In` appear as user-facing labels in web nav and FavoritesPage.… | [findings/core-ux.md](findings/core-ux.md) |
| BV-072 | md-core-033 | core-ux | brand-voice | P1 | Mobile-delivery slide-to-confirm action label `Picked Up Order` (past tense for… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-005 | web-ux-checkout-tip | core-ux | legal | P1 | Tip language '100% of your tip goes to the home chef' is an absolute representa… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-006 | web-ux-checkout-address-form | core-ux | legal | P1 | Address form collects PII (street, city, state, postal code) at checkout with n… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-009 | mc-checkout-section-address | core-ux | legal | P1 | Mobile address capture step has no DPDP notice or purpose statement | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-010 | web-ux-cart-order-summary | core-ux | legal | P1 | Cart summary lists 'Service fee', 'Delivery fee', 'Tax' as opaque line items wi… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-012 | mc-cartsheet-checkout-cta | core-ux | legal | P1 | Mobile cart 'Proceed to Checkout' has no minimum-order, surge-pricing, or geo-a… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-015 | web-ux-orders-status-labels | core-ux | legal | P1 | Status taxonomy includes 'Refunded' as a public-facing state but no associated… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-017 | mc-order-detail-status-labels | core-ux | legal | P1 | Mobile customer status labels do not include a 'Refunded' state ('Pending / Con… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-020 | dp-ux-active-cancel-btn | core-ux | legal | P1 | Driver 'Cancel Delivery' is a destructive action with consumer-protection impli… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-023 | vp-ux-menu-view-page | core-ux | legal | P1 | Menu item read-only view shows 'Dietary Information / No dietary information pr… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-029 | vp-ux-profile-docs | core-ux | legal | P1 | Vendor-portal profile documents section 'Upload required documents for verifica… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-030 | ap-approvaldetail-detail-fields | core-ux | legal | P1 | Admin approval detail captures 'Terms Accepted / Hygiene Policy Accepted' as fi… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-031 | web-ux-browse-sort-options | core-ux | legal | P1 | Sort dropdown includes 'Top Rated' and 'Most Popular' — these are objective cla… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-033 | web-ux-social-feed-sidebar | core-ux | legal | P1 | Social feed widget shows 'Trending Chefs / Popular This Week' — these are objec… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-036 | web-ux-chefdetail-reviews | core-ux | legal | P1 | Reviews block shows 'Customer Reviews / Chef's Response' but no moderation poli… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-037 | web-ux-chef-social | core-ux | legal | P1 | Chef social composer 'Share something about this dish...' allows user-generated… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-041 | web-ux-catering-request-event-fields | core-ux | legal | P1 | Catering Request collects event date, guest count, budget — but no FSSAI cateri… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-044 | mc-catering-form-labels | core-ux | legal | P1 | Mobile catering form has 'Additional Details' free-text field and a 'Submit Req… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-045 | vp-ux-settings-order-acceptance | core-ux | legal | P1 | Order Acceptance settings show 'Auto-accept threshold ($)' — currency mismatch… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-047 | vp-ux-earnings-cards | core-ux | legal | P1 | Earnings cards show 'Available Balance / Pending Payout / This Month / Lifetime… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-048 | vp-ux-payouts-title | core-ux | legal | P1 | Payout History 'View all your past and pending payouts' surface — inventory doe… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-049 | mv-earnings-payout-fields | core-ux | legal | P1 | Mobile chef payout capture 'Bank / Account Number / IFSC / UPI ID' has no inlin… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-052 | dp-ux-stripe-active-gateway | core-ux | legal | P1 | Driver can switch between Stripe and Razorpay as active payout gateway — switch… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-053 | ap-auditlogs-title | core-ux | legal | P1 | Audit logs page exists (good) — inventory tags it 'LEGAL LENS' — but inventory… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-054 | ap-auditlogs-table | core-ux | legal | P1 | Audit table columns 'When; Actor; Action; Entity; IP' — missing user-agent, geo… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-058 | ap-platsettings-cards | core-ux | legal | P1 | Platform Settings 'Operating hours / When the platform accepts new orders' — if… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-060 | ap-settings-payment-fields | core-ux | legal | P1 | Payment-gateway settings show 'Razorpay Key ID / Razorpay Key Secret / Webhook… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-064 | ap-userdetail-info-rows | core-ux | legal | P1 | Admin User Detail surfaces 'Email; Phone; Auth Provider; Role; Joined; Last Log… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-067 | dp-ux-stat-verified-partners | core-ux | legal | P1 | 'Verified partners' label on admin dashboard — what verification standard is me… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-068 | dp-ux-partner-detail-verify-block | core-ux | legal | P1 | Partner Verification block 'Review their documents and approve or reject their… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-071 | md-core-018 | core-ux | legal | P1 | Driver 'Go Online to Accept Deliveries' has no insurance / safety attestation g… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-072 | md-core-062 | core-ux | legal | P1 | Driver settings 'Default Online Status — Automatically go online when app opens… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-073 | dp-ux-active-payout-label | core-ux | legal | P1 | Active delivery shows 'Payout' value pre-completion — if payout is shown but la… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-077 | md-core-043 | core-ux | legal | P1 | 'Navigate' (P0 SAFETY tag in inventory) launches external Apple/Google Maps — t… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-078 | ap-providercreate-fields | core-ux | legal | P1 | Delivery Provider create form captures 'API Key; API Secret; Webhook Secret' an… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-080 | web-ux-profile-payments | core-ux | legal | P1 | Payment Methods tab heading exists but inventory does not surface how stored ca… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-083 | web-ux-profile-notifications | core-ux | legal | P1 | Notification Preferences tab exists but inventory does not surface granular opt… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-085 | web-ux-profile-addresses | core-ux | legal | P1 | Saved Addresses tab — addresses are PII retained indefinitely unless user-purge… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-090 | mv-onb-docs-pdf-uploaded | core-ux | legal | P1 | Document upload flow ('PDF uploaded / Camera / Gallery / PDF') accepts KYC and… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-092 | ap-notifsettings-categories | core-ux | legal | P1 | Notification categories include 'Promotions / Discounts, newsletters, and marke… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-097 | web-ux-layout-nav | core-ux | legal | P1 | Customer top nav 'Home / Browse Chefs / Favorites / Catering / Profile / My Ord… | [findings/core-ux.md](findings/core-ux.md) |
| TW-011 | dp-ux-nav-bottom-items | core-ux | technical-writer | P1 | Driver bottom-nav 'Home' but sidebar 'Dashboard' — same destination, two labels… | [findings/core-ux.md](findings/core-ux.md) |
| TW-012 | dp-ux-nav-partner-items | core-ux | technical-writer | P1 | Driver nav 'Active Delivery' (sidebar) vs 'Active' (bottom nav) — same destinat… | [findings/core-ux.md](findings/core-ux.md) |
| TW-013 | md-core-052 | core-ux | technical-writer | P1 | Driver-mobile More menu exposes admin-only 'Fleet' and 'Staff' nav items to all… | [findings/core-ux.md](findings/core-ux.md) |
| TW-014 | web-ux-cart-checkout-cta | core-ux | technical-writer | P1 | Checkout CTA 'Sign in to Checkout' uses title-case on 'Checkout' but sentence c… | [findings/core-ux.md](findings/core-ux.md) |
| TW-015 | web-ux-checkout-summary | core-ux | technical-writer | P1 | Checkout primary CTA 'Place Order' is title case. Style guide §4 requires sente… | [findings/core-ux.md](findings/core-ux.md) |
| TW-016 | mc-checkout-place-order | core-ux | technical-writer | P1 | Mobile checkout CTA 'Place Order · ₹{total}' uses title case and middle-dot sep… | [findings/core-ux.md](findings/core-ux.md) |
| TW-017 | web-ux-cart-heading | core-ux | technical-writer | P1 | Cart heading uses 'Your Cart' on web but mobile uses 'Your Cart' on the sheet (… | [findings/core-ux.md](findings/core-ux.md) |
| TW-018 | web-ux-checkout-payment-section | core-ux | technical-writer | P1 | Payment section copy 'Pay securely via UPI, cards, net banking, or wallets' use… | [findings/core-ux.md](findings/core-ux.md) |
| TW-019 | web-ux-checkout-delivery-time | core-ux | technical-writer | P1 | 'As soon as possible / Usually 30-45 minutes' — 'As soon as possible' is wordy… | [findings/core-ux.md](findings/core-ux.md) |
| TW-020 | web-ux-checkout-tip | core-ux | technical-writer | P1 | Tip helper '100% of your tip goes to the home chef' — the percent sign is fine… | [findings/core-ux.md](findings/core-ux.md) |
| TW-021 | mc-catering-form-labels | core-ux | technical-writer | P1 | Catering date field exposes 'Event Date * (YYYY-MM-DD)' — raw date format hint… | [findings/core-ux.md](findings/core-ux.md) |
| TW-022 | web-ux-catering-request-event-fields | core-ux | technical-writer | P1 | Web catering form uses title case throughout ('Event Date', 'Event Time', 'Numb… | [findings/core-ux.md](findings/core-ux.md) |
| TW-023 | web-ux-browse-filters-rating | core-ux | technical-writer | P1 | Rating filter options 'Any Rating / 4.5+ Stars / 4+ Stars / 3.5+ Stars' use tit… | [findings/core-ux.md](findings/core-ux.md) |
| TW-024 | web-ux-browse-sort-options | core-ux | technical-writer | P1 | Sort options 'Top Rated / Nearest / Most Popular / Price' title-case + 'Most Po… | [findings/core-ux.md](findings/core-ux.md) |
| TW-025 | mc-home-sort-options | core-ux | technical-writer | P1 | Inventory note flags 'two map to same value rating — likely UX bug'. Sort optio… | [findings/core-ux.md](findings/core-ux.md) |
| TW-026 | dp-ux-stripe-make-primary | core-ux | technical-writer | P1 | 'Make Stripe My Primary Gateway' is 5 words, title case, possessive ('My'). Sty… | [findings/core-ux.md](findings/core-ux.md) |
| TW-027 | dp-ux-stripe-active-gateway | core-ux | technical-writer | P1 | 'Active payout gateway: {Stripe/Razorpay}' uses colon-then-value pattern. Style… | [findings/core-ux.md](findings/core-ux.md) |
| TW-028 | dp-ux-staff-form-labels | core-ux | technical-writer | P1 | Staff invite labels use 'Email *', 'Role *' with asterisks in label text and em… | [findings/core-ux.md](findings/core-ux.md) |
| TW-029 | mc-catering-status-labels | core-ux | technical-writer | P1 | Catering request status pills 'Open / Quoted / Accepted / Completed / Cancelled… | [findings/core-ux.md](findings/core-ux.md) |
| TW-030 | web-ux-chefdetail-status-badges | core-ux | technical-writer | P1 | Chef detail page mixes 'Open for orders' (good) with 'Currently closed' (filler… | [findings/core-ux.md](findings/core-ux.md) |
| BA-026 | web-ux-cart-heading | core-ux | business-analyst | P2 | Empty cart copy uses 'Looks like you haven't added any items yet' — passive, wo… | [findings/core-ux.md](findings/core-ux.md) |
| BA-027 | web-ux-favorites-heading | core-ux | business-analyst | P2 | Favorites empty state mentions saving 'up to {MAX_FAVORITES}' chefs — limit cre… | [findings/core-ux.md](findings/core-ux.md) |
| BA-028 | web-ux-chef-menu | core-ux | business-analyst | P2 | Chef empty menu state 'Add your first menu item to get started' is weak — no mo… | [findings/core-ux.md](findings/core-ux.md) |
| BA-029 | web-ux-social-feed-heading | core-ux | business-analyst | P2 | Social feed page title is 'Chef\'s Feed' — apostrophe-s implies singular chef o… | [findings/core-ux.md](findings/core-ux.md) |
| BA-030 | dp-ux-dashboard-urgent-unassigned | core-ux | business-analyst | P2 | 'Unassigned orders — Needs dispatcher attention' is passive; no count shown inl… | [findings/core-ux.md](findings/core-ux.md) |
| BA-031 | dp-ux-partner-detail-verify-block | core-ux | business-analyst | P2 | Partner verification block gives no SLA or expected review time to the partner… | [findings/core-ux.md](findings/core-ux.md) |
| BA-032 | web-ux-profile-payments | core-ux | business-analyst | P2 | Payment Methods section has only a label 'Payment Methods' with no CTA to add a… | [findings/core-ux.md](findings/core-ux.md) |
| BA-033 | web-ux-chef-profile | core-ux | business-analyst | P2 | Chef profile 'Your Stats' section shown in profile editor — chefs may confuse i… | [findings/core-ux.md](findings/core-ux.md) |
| BA-034 | vp-ux-kitchen-photos | core-ux | business-analyst | P2 | Kitchen photo upload prompt does not explain the trust/conversion value of addi… | [findings/core-ux.md](findings/core-ux.md) |
| BA-035 | web-ux-orderdetail-sections | core-ux | business-analyst | P2 | Order detail shows 'Leave a Review' section but no review was left — no timing… | [findings/core-ux.md](findings/core-ux.md) |
| BA-036 | mc-cartbar-cta | core-ux | business-analyst | P2 | Mobile cart floating bar CTA is 'View Cart' — passive, does not reinforce progr… | [findings/core-ux.md](findings/core-ux.md) |
| BA-037 | mc-cartsheet-checkout-cta | core-ux | business-analyst | P2 | Cart sheet CTA 'Proceed to Checkout' is longer than style guide's 3-word button… | [findings/core-ux.md](findings/core-ux.md) |
| BA-038 | mc-order-detail-price-rows | core-ux | business-analyst | P2 | Order detail price row shows 'Delivery Fee' (title case) vs checkout 'Delivery… | [findings/core-ux.md](findings/core-ux.md) |
| BA-039 | mv-settings-change-password | core-ux | business-analyst | P2 | 'Change Password' nav row on mobile vendor app routes to forgot-password flow —… | [findings/core-ux.md](findings/core-ux.md) |
| BA-040 | vp-ux-dashboard-pending-cta | core-ux | business-analyst | P2 | Chef dashboard pending orders CTA copy is '{n} orders / Waiting for you to acce… | [findings/core-ux.md](findings/core-ux.md) |
| BA-041 | web-ux-chef-earnings | core-ux | business-analyst | P2 | Chef earnings page has no payout schedule disclosure — when will the 'Available… | [findings/core-ux.md](findings/core-ux.md) |
| BA-042 | vp-ux-payouts-title | core-ux | business-analyst | P2 | Payout History page ('View all your past and pending payouts') has no context f… | [findings/core-ux.md](findings/core-ux.md) |
| BA-043 | web-ux-delivery-dashboard | core-ux | business-analyst | P2 | Driver dashboard 'You\'re currently offline' state has a CTA 'Go online' but no… | [findings/core-ux.md](findings/core-ux.md) |
| BA-044 | dp-ux-earnings-avg | core-ux | business-analyst | P2 | 'Avg/Delivery' uses a slash compaction that may be unclear to drivers who are n… | [findings/core-ux.md](findings/core-ux.md) |
| BA-045 | dp-ux-partner-performance | core-ux | business-analyst | P2 | 'CSAT Score' uses an acronym without expansion — ops staff unfamiliar with the… | [findings/core-ux.md](findings/core-ux.md) |
| BA-046 | dp-ux-active-est-time-label | core-ux | business-analyst | P2 | 'Est. Time' abbreviation on active delivery screen may be unclear at a glance f… | [findings/core-ux.md](findings/core-ux.md) |
| BA-047 | dp-ux-delivery-pickup-label | core-ux | business-analyst | P2 | Available deliveries card uses 'PICKUP' and 'DROPOFF' in ALL CAPS — violates st… | [findings/core-ux.md](findings/core-ux.md) |
| BA-048 | md-core-079 | core-ux | business-analyst | P2 | Mobile delivery app delivery detail uses 'Drop-off' (hyphenated) while active d… | [findings/core-ux.md](findings/core-ux.md) |
| BA-049 | web-ux-orders-status-labels | core-ux | business-analyst | P2 | Order status 'item(s)' uses parenthetical plural — explicitly banned by style g… | [findings/core-ux.md](findings/core-ux.md) |
| BA-050 | md-core-044 | core-ux | business-analyst | P2 | Mobile delivery active screen 'Order Summary ({n} item(s))' uses parenthetical… | [findings/core-ux.md](findings/core-ux.md) |
| BA-051 | mc-home-sort-options | core-ux | business-analyst | P2 | 'Recommended' and 'Top Rated' sort options both map to the same 'rating' value… | [findings/core-ux.md](findings/core-ux.md) |
| BA-052 | mc-favorites-title | core-ux | business-analyst | P2 | Tab bar label 'Saved' and screen title 'Saved Chefs' are inconsistent — minor b… | [findings/core-ux.md](findings/core-ux.md) |
| BA-053 | web-ux-chef-social | core-ux | business-analyst | P2 | Social feed empty state for chefs has no guidance on what type of content perfo… | [findings/core-ux.md](findings/core-ux.md) |
| BA-054 | web-ux-social-feed-sidebar | core-ux | business-analyst | P2 | 'Trending Chefs / Popular This Week' sidebar widget — if these lists are hardco… | [findings/core-ux.md](findings/core-ux.md) |
| BV-012 | web-ux-cart-heading | core-ux | brand-voice | P2 | Web uses `Your Cart` (Title Case) and mobile-customer uses `Your Cart` (Title C… | [findings/core-ux.md](findings/core-ux.md) |
| BV-014 | web-ux-cart-checkout-cta | core-ux | brand-voice | P2 | Web cart CTA `Sign in to Checkout` mixes `Sign in` (correct per STYLE-GUIDE) wi… | [findings/core-ux.md](findings/core-ux.md) |
| BV-020 | md-core-018 | core-ux | brand-voice | P2 | Mobile-delivery `Go Online to Accept Deliveries` (Title Case across 5 words) is… | [findings/core-ux.md](findings/core-ux.md) |
| BV-021 | md-core-032 | core-ux | brand-voice | P2 | Slide-to-confirm action labels mix verb tenses: `Arrived at Kitchen` (past), `P… | [findings/core-ux.md](findings/core-ux.md) |
| BV-024 | vp-ux-dashboard-pending-card | core-ux | brand-voice | P2 | Vendor portal chef-facing pending orders show `Accept / Reject` — consistent wi… | [findings/core-ux.md](findings/core-ux.md) |
| BV-027 | ap-approvals-statuses | core-ux | brand-voice | P2 | Approval status filter mixes `Info Requested` (Title Case, 2 words) with single… | [findings/core-ux.md](findings/core-ux.md) |
| BV-038 | web-ux-profile-heading | core-ux | brand-voice | P2 | Web customer profile heading `Account Settings` (Title Case). STYLE-GUIDE requi… | [findings/core-ux.md](findings/core-ux.md) |
| BV-039 | web-ux-orderdetail-sections | core-ux | brand-voice | P2 | Web customer order detail headings all Title Case: `Order Items`, `Delivery Add… | [findings/core-ux.md](findings/core-ux.md) |
| BV-041 | web-ux-orders-status-labels | core-ux | brand-voice | P2 | Customer status labels all Title Case: `Pending / Accepted / Preparing / Ready… | [findings/core-ux.md](findings/core-ux.md) |
| BV-044 | vp-ux-profile-business-info | core-ux | brand-voice | P2 | Vendor-portal `Tell customers about your kitchen, cooking style, and what makes… | [findings/core-ux.md](findings/core-ux.md) |
| BV-045 | vp-ux-kitchen-photos | core-ux | brand-voice | P2 | Vendor-portal: `Add photos of your kitchen to build trust with customers ({n}/5… | [findings/core-ux.md](findings/core-ux.md) |
| BV-046 | vp-ux-dashboard-pending-cta | core-ux | brand-voice | P2 | Vendor-portal `Waiting for you to accept` (vp-ux-dashboard-pending-cta) is conv… | [findings/core-ux.md](findings/core-ux.md) |
| BV-050 | mc-profile-more-rows | core-ux | brand-voice | P2 | Mobile-customer profile `More` section uses emoji icons (📱/🍽️) for nav rows. ST… | [findings/core-ux.md](findings/core-ux.md) |
| BV-053 | md-core-039 | core-ux | brand-voice | P2 | Mobile-delivery cancelled banner: `This delivery has been cancelled. Check the… | [findings/core-ux.md](findings/core-ux.md) |
| BV-054 | web-ux-browse-filters-cuisines | core-ux | brand-voice | P2 | Customer cuisine filter has 10 cuisines (web): `South Indian / North Indian / I… | [findings/core-ux.md](findings/core-ux.md) |
| BV-057 | dp-ux-profile-fallback-name | core-ux | brand-voice | P2 | Delivery-portal driver profile fallback name varies: `Delivery Partner` (UI she… | [findings/core-ux.md](findings/core-ux.md) |
| BV-060 | mc-catering-form-labels | core-ux | brand-voice | P2 | Mobile-customer catering form label: `Event Date * (YYYY-MM-DD)` — raw ISO form… | [findings/core-ux.md](findings/core-ux.md) |
| BV-070 | ap-staff-invite-dialog-title | core-ux | brand-voice | P2 | Admin staff invite dialog says `Invite Staff Member` (Title Case); delivery-por… | [findings/core-ux.md](findings/core-ux.md) |
| BV-073 | mc-catering-status-labels | core-ux | brand-voice | P2 | Mobile-customer catering status pills: `Open / Quoted / Accepted / Completed /… | [findings/core-ux.md](findings/core-ux.md) |
| BV-086 | mc-profile-sections | core-ux | brand-voice | P2 | Mobile-customer profile section uses `Personal Info / Food Preferences / More`;… | [findings/core-ux.md](findings/core-ux.md) |
| BV-087 | web-ux-profile-preferences | core-ux | brand-voice | P2 | Web profile uses both `Food Preferences` and `Favourite Cuisines` (British spel… | [findings/core-ux.md](findings/core-ux.md) |
| BV-095 | dp-ux-dashboard-partner-online | core-ux | brand-voice | P2 | Driver online-state copy varies between surfaces: `Ready for deliveries` (dp da… | [findings/core-ux.md](findings/core-ux.md) |
| BV-096 | dp-ux-dashboard-youre-offline | core-ux | brand-voice | P2 | Driver offline state: `You're offline` (dp), `You are Offline` (mobile), `Curre… | [findings/core-ux.md](findings/core-ux.md) |
| BV-097 | mv-earnings-total | core-ux | brand-voice | P2 | Mobile-vendor `Total Earnings` vs mobile-vendor analytics `Total Revenue` — sam… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-011 | web-ux-cart-checkout-cta | core-ux | legal | P2 | 'Sign in to Checkout' enforces auth gate — but the moment a user starts adding… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-016 | ap-orders-filter-status | core-ux | legal | P2 | Admin order filter shows 'Refunded' alongside 'Cancelled' but inventory notes '… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-024 | mc-home-cuisine-filters | core-ux | legal | P2 | Cuisine filter chips do not include a veg/non-veg toggle or visual veg-mark — F… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-026 | mv-menunew-desc-ph | core-ux | legal | P2 | Description placeholder 'Describe your dish (at least 20 characters)' does not… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-032 | web-ux-browse-filters-rating | core-ux | legal | P2 | Rating filter '4.5+ Stars / 4+ Stars / 3.5+ Stars' relies on rating data — must… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-035 | mc-home-cuisine-filters | core-ux | legal | P2 | Mobile sort option 'Recommended' is opaque — could be paid placement, algorithm… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-038 | vp-ux-reviews-title | core-ux | legal | P2 | Vendor reviews page 'See what your customers are saying' provides no path to fl… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-039 | mv-reviewdetail-input-ph | core-ux | legal | P2 | Reply input 'Write a thoughtful reply to this review' has no defamation / commu… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-051 | dp-ux-stripe-sub | core-ux | legal | P2 | Stripe subtitle 'For drivers outside India, or as an alternative to Razorpay' s… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-056 | ap-exports-subtitle | core-ux | legal | P2 | Export subtitle 'Files stream directly from the API' suggests no intermediate s… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-059 | ap-platsettings-zones-fields | core-ux | legal | P2 | Service zone CRUD using lat/long bounding boxes — if customer addresses fall ou… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-061 | ap-secsettings-apikey-fields | core-ux | legal | P2 | API key form 'Scopes: read; write; admin; Expires in (days; 0 = never)' — 'neve… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-062 | ap-secsettings-session-fields | core-ux | legal | P2 | Session management fields configurable but no policy lower-bound enforced in in… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-063 | ap-staffdetail-section-roleperm | core-ux | legal | P2 | Staff Role & Permissions section exists but inventory does not surface least-pr… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-065 | ap-userdetail-order-stats | core-ux | legal | P2 | User Detail shows 'Total Orders / Total Spent / Last Order' — financial PII vis… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-074 | md-core-047 | core-ux | legal | P2 | 'You earned ₹{payout} for this delivery' is a representation of completed earni… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-075 | dp-ux-active-pickup-from | core-ux | legal | P2 | 'Pickup from' and 'Deliver to' display addresses to driver — chef/customer PII… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-079 | ap-delivery-statuses | core-ux | legal | P2 | Delivery status taxonomy includes 'Failed' and 'Returned' — these are refund-tr… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-081 | web-ux-profile-2fa | core-ux | legal | P2 | Two-Factor Authentication setup page exists (good for DPDP §8(5) safeguards) bu… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-082 | web-ux-profile-preferences | core-ux | legal | P2 | Food preferences ('Dietary Preferences / Food Allergies') — Food Allergies is s… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-093 | vp-ux-settings-notifications | core-ux | legal | P2 | Vendor SMS notifications 'Get an SMS for each new order' — SMS to vendor is tra… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-094 | vp-ux-settings-password | core-ux | legal | P2 | Password change section copy 'Your account is linked to {provider} login. Passw… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-095 | ap-staff-invite-fields | core-ux | legal | P2 | Staff invitation form 'Email; Role; Department; Title; Personal Message' issues… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-096 | dp-ux-staff-invite-h | core-ux | legal | P2 | Delivery-portal Staff Invitation 'Share this link with the new staff member' —… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-098 | vp-ux-layout-nav | core-ux | legal | P2 | Vendor portal sidebar — no link to chef/vendor T&C, Privacy, or platform commis… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-099 | dp-ux-nav-partner-items | core-ux | legal | P2 | Driver sidebar nav 'Dashboard / Active Delivery / Available / History / Earning… | [findings/core-ux.md](findings/core-ux.md) |
| TW-031 | pattern-title-case-headings-web | core-ux | technical-writer | P2 | PATTERN — Web app uses title case on most page H1s and section headings; style… | [findings/core-ux.md](findings/core-ux.md) |
| TW-032 | pattern-title-case-headings-admin-portal | core-ux | technical-writer | P2 | PATTERN — Admin portal uses title case for all section headings; style guide al… | [findings/core-ux.md](findings/core-ux.md) |
| TW-033 | pattern-title-case-headings-delivery-portal | core-ux | technical-writer | P2 | PATTERN — Delivery portal (web) uses title case for stat labels, section header… | [findings/core-ux.md](findings/core-ux.md) |
| TW-034 | pattern-title-case-headings-vendor-portal | core-ux | technical-writer | P2 | PATTERN — Vendor portal title-cases section headings, card titles, and stat lab… | [findings/core-ux.md](findings/core-ux.md) |
| TW-035 | pattern-title-case-headings-mobile-customer | core-ux | technical-writer | P2 | PATTERN — Mobile customer screen titles use title case. Style guide §4 sentence… | [findings/core-ux.md](findings/core-ux.md) |
| TW-036 | pattern-title-case-headings-mobile-vendor | core-ux | technical-writer | P2 | PATTERN — Mobile vendor uses title case section headings, stats, field labels,… | [findings/core-ux.md](findings/core-ux.md) |
| TW-037 | pattern-title-case-headings-mobile-delivery | core-ux | technical-writer | P2 | PATTERN — Mobile delivery screen titles and status badges use title case. Drive… | [findings/core-ux.md](findings/core-ux.md) |
| TW-038 | pattern-asterisk-on-form-labels | core-ux | technical-writer | P2 | PATTERN — Required-field asterisks embedded in label text across multiple forms… | [findings/core-ux.md](findings/core-ux.md) |
| TW-039 | pattern-trailing-colon-form-labels | core-ux | technical-writer | P2 | PATTERN — Trailing colons on labels and section captions. Style guide §4 form l… | [findings/core-ux.md](findings/core-ux.md) |
| TW-040 | pattern-loading-ellipsis-three-dots | core-ux | technical-writer | P2 | PATTERN — Loading states use three dots '...' instead of Unicode ellipsis '…'.… | [findings/core-ux.md](findings/core-ux.md) |
| TW-041 | pattern-multi-word-buttons | core-ux | technical-writer | P2 | PATTERN — Button labels exceed style guide §4 cap of 3 words. | [findings/core-ux.md](findings/core-ux.md) |
| TW-042 | pattern-uppercase-eyebrows | core-ux | technical-writer | P2 | PATTERN — ALL CAPS or shouting eyebrows on driver/customer surfaces. Style guid… | [findings/core-ux.md](findings/core-ux.md) |
| TW-043 | pattern-banned-vocabulary-logout | core-ux | technical-writer | P2 | PATTERN — 'Logout' (one word) and 'Log out' (two words) used across apps. Style… | [findings/core-ux.md](findings/core-ux.md) |
| TW-045 | pattern-customer-noun-user | core-ux | technical-writer | P2 | PATTERN — Admin / chef-facing surfaces refer to customers as 'User' / 'Users'.… | [findings/core-ux.md](findings/core-ux.md) |
| TW-046 | pattern-banned-vocabulary-driver-vs-delivery-partner | core-ux | technical-writer | P2 | PATTERN — Customer/admin surfaces alternate 'Driver' and 'Delivery Partner' / '… | [findings/core-ux.md](findings/core-ux.md) |
| TW-047 | pattern-money-formatting | core-ux | technical-writer | P2 | PATTERN — Money symbol usage drifts. Style guide §6: '₹120' (no space, symbol f… | [findings/core-ux.md](findings/core-ux.md) |
| TW-048 | pattern-prep-time-unit-formatting | core-ux | technical-writer | P2 | PATTERN — Prep time unit drifts: '15min' (no space) vs '15 min' (with space) fo… | [findings/core-ux.md](findings/core-ux.md) |
| TW-049 | pattern-row-label-case-drift-checkout-orderdetail | core-ux | technical-writer | P2 | PATTERN — 'Delivery fee' (sentence) on checkout drifts to 'Delivery Fee' (title… | [findings/core-ux.md](findings/core-ux.md) |
| TW-050 | pattern-stat-card-label-drift-chef | core-ux | technical-writer | P2 | PATTERN — Chef KPI tile labels drift: 'Today's Earnings' (mobile dash) vs 'Tota… | [findings/core-ux.md](findings/core-ux.md) |
| TW-051 | pattern-empty-state-formula | core-ux | technical-writer | P2 | PATTERN — Empty states omit the §4 formula 'Why it's empty → One action'. Most… | [findings/core-ux.md](findings/core-ux.md) |
| TW-052 | pattern-section-headers-possessive-my | core-ux | technical-writer | P2 | PATTERN — 'My X' possessive headings throughout. Style guide §3 prefers descrip… | [findings/core-ux.md](findings/core-ux.md) |
| TW-053 | pattern-success-toast-non-past | core-ux | technical-writer | P2 | PATTERN — Success messaging doesn't follow §4 'past tense, ≤6 words, period'. | [findings/core-ux.md](findings/core-ux.md) |
| TW-054 | pattern-modal-subtitle-formula | core-ux | technical-writer | P2 | PATTERN — Modal subtitles violate §4 'explain consequence in one sentence' form… | [findings/core-ux.md](findings/core-ux.md) |
| TW-056 | pattern-cuisine-list-drift | core-ux | technical-writer | P2 | PATTERN — Cuisine option lists drift across apps and even within a single app.… | [findings/core-ux.md](findings/core-ux.md) |
| TW-057 | pattern-driver-app-status-vs-action-drift | core-ux | technical-writer | P2 | PATTERN — Driver status labels and action labels overlap in confusing ways: 'Pi… | [findings/core-ux.md](findings/core-ux.md) |
| TW-115 | ap-layout-nav-reviews | core-ux | technical-writer | P2 | Sidebar nav label 'Reviews' routes to /approvals — label says 'Reviews' but the… | [findings/core-ux.md](findings/core-ux.md) |
| BA-055 | mv-onb-ops-preptime-options | core-ux | business-analyst | P3 | Mobile vendor onboarding uses '15min/30min' (no space) while MenuPage uses '{n}… | [findings/core-ux.md](findings/core-ux.md) |
| BA-056 | dp-ux-logo-text | core-ux | business-analyst | P3 | Delivery portal brand name shows 'Fe3dr' — if this is a placeholder/internal na… | [findings/core-ux.md](findings/core-ux.md) |
| BA-057 | ap-layout-nav-reviews | core-ux | business-analyst | P3 | Admin sidebar nav label 'Reviews' links to '/approvals' — wrong label for the d… | [findings/core-ux.md](findings/core-ux.md) |
| BA-058 | dp-ux-nav-staff-items | core-ux | business-analyst | P3 | Delivery portal staff nav includes a 'Zones' link to '/fleet/zones' — route not… | [findings/core-ux.md](findings/core-ux.md) |
| BA-059 | dp-ux-nav-bottom-items | core-ux | business-analyst | P3 | Driver mobile bottom nav uses 'Home' while sidebar uses 'Dashboard' for the sam… | [findings/core-ux.md](findings/core-ux.md) |
| BA-060 | dp-ux-profile-fallback-name | core-ux | business-analyst | P3 | Driver profile fallback display name is 'Delivery Partner' — inconsistent with… | [findings/core-ux.md](findings/core-ux.md) |
| BA-061 | md-core-062 | core-ux | business-analyst | P3 | 'Default Online Status' toggle for auto-online-on-app-open has no safety warnin… | [findings/core-ux.md](findings/core-ux.md) |
| BA-062 | mv-undo-cta | core-ux | business-analyst | P3 | 'UNDO' button in vendor UndoSnackbar is ALL CAPS — minor style guide violation… | [findings/core-ux.md](findings/core-ux.md) |
| BA-063 | mc-profile-more-rows | core-ux | business-analyst | P3 | Customer profile 'More' section nav rows use emoji icons (📱/🍽️) — style guide d… | [findings/core-ux.md](findings/core-ux.md) |
| BA-064 | web-ux-catering-request-steps | core-ux | business-analyst | P3 | Catering request multi-step form has no 'Step X of 3' text progress indicator b… | [findings/core-ux.md](findings/core-ux.md) |
| BA-065 | vp-ux-settings-payout-section | core-ux | business-analyst | P3 | 'Razorpay Pending' status on payout settings provides no explanation of what ac… | [findings/core-ux.md](findings/core-ux.md) |
| BA-066 | vp-ux-profile-docs | core-ux | business-analyst | P3 | Document upload section 'Upload required documents for verification' does not e… | [findings/core-ux.md](findings/core-ux.md) |
| BA-067 | web-ux-chef-dashboard-sections | core-ux | business-analyst | P3 | Chef dashboard 'Shortcuts' section has quick-action tiles but no visual differe… | [findings/core-ux.md](findings/core-ux.md) |
| BA-068 | mc-checkout-add-address | core-ux | business-analyst | P3 | 'Add New Address' CTA in mobile checkout is an inline expansion with no context… | [findings/core-ux.md](findings/core-ux.md) |
| BA-069 | web-ux-checkout-address-form | core-ux | business-analyst | P3 | Checkout address form has 'Apartment, suite, etc. (optional)' — parenthetical '… | [findings/core-ux.md](findings/core-ux.md) |
| BA-070 | mv-menunew-desc-ph | core-ux | business-analyst | P3 | Menu item description placeholder 'Describe your dish (at least 20 characters)'… | [findings/core-ux.md](findings/core-ux.md) |
| BV-015 | web-ux-checkout-heading | core-ux | brand-voice | P3 | Web `Checkout` page H1 is bare `Checkout`. Mobile-customer is `Checkout`. Consi… | [findings/core-ux.md](findings/core-ux.md) |
| BV-040 | mc-checkout-totals | core-ux | brand-voice | P3 | Mobile-customer checkout totals mix capitalization: `Subtotal / Delivery fee /… | [findings/core-ux.md](findings/core-ux.md) |
| BV-043 | web-ux-checkout-payment-section | core-ux | brand-voice | P3 | Web payment notice says `Pay securely via UPI, cards, net banking, or wallets`.… | [findings/core-ux.md](findings/core-ux.md) |
| BV-047 | vp-ux-dashboard-quick-actions | core-ux | brand-voice | P3 | Quick Actions tiles use comma-separated title-action format: `Add Menu Item, Cr… | [findings/core-ux.md](findings/core-ux.md) |
| BV-048 | mv-dash-subhead | core-ux | brand-voice | P3 | Mobile-vendor dashboard subhead: `Here's your kitchen overview` vs vendor-porta… | [findings/core-ux.md](findings/core-ux.md) |
| BV-049 | mv-dash-stats-today-earnings | core-ux | brand-voice | P3 | Mobile-vendor stat: `Today's Earnings`; vendor-portal: `Today's revenue` (vp-ux… | [findings/core-ux.md](findings/core-ux.md) |
| BV-051 | mv-undo-cta | core-ux | brand-voice | P3 | Mobile-vendor `UNDO` snackbar action is ALL CAPS. STYLE-GUIDE bans ALL CAPS bod… | [findings/core-ux.md](findings/core-ux.md) |
| BV-052 | md-core-047 | core-ux | brand-voice | P3 | Mobile-delivery delivery-complete success line: `You earned ₹{payout} for this… | [findings/core-ux.md](findings/core-ux.md) |
| BV-055 | mc-profile-cuisines | core-ux | brand-voice | P3 | Mobile-customer cuisine order drifts between onboarding and profile screens. In… | [findings/core-ux.md](findings/core-ux.md) |
| BV-056 | mv-onb-ops-preptime-options | core-ux | brand-voice | P3 | Mobile-vendor onboarding uses `15min / 30min / 45min / 60min / 90min` (no space… | [findings/core-ux.md](findings/core-ux.md) |
| BV-058 | dp-ux-staff-roles-fallback | core-ux | brand-voice | P3 | Delivery-portal staff role list mixes `Delivery Ops`, `Fleet Manager`, `Super A… | [findings/core-ux.md](findings/core-ux.md) |
| BV-059 | mv-menu-heading | core-ux | brand-voice | P3 | Mobile-vendor menu heading is `My Menu` while web chef menu is `Menu Management… | [findings/core-ux.md](findings/core-ux.md) |
| BV-061 | dp-ux-active-est-time-label | core-ux | brand-voice | P3 | Delivery-portal `Est. Time` (abbreviated) vs mobile-delivery uses full word con… | [findings/core-ux.md](findings/core-ux.md) |
| BV-062 | dp-ux-earnings-avg | core-ux | brand-voice | P3 | Delivery-portal `Avg/Delivery` uses slash compaction. STYLE-GUIDE: Plain Englis… | [findings/core-ux.md](findings/core-ux.md) |
| BV-063 | dp-ux-partner-performance | core-ux | brand-voice | P3 | Delivery-portal partner detail uses `CSAT Score` — unexplained internal acronym… | [findings/core-ux.md](findings/core-ux.md) |
| BV-064 | ap-approvals-types | core-ux | brand-voice | P3 | Admin approval types: `Kitchen Onboarding; Document Verification; New Menu Item… | [findings/core-ux.md](findings/core-ux.md) |
| BV-066 | ap-platsettings-title | core-ux | brand-voice | P3 | Admin settings landing card titled `Platform` but the dedicated platform settin… | [findings/core-ux.md](findings/core-ux.md) |
| BV-067 | ap-userdetail-verification | core-ux | brand-voice | P3 | Admin user-detail verification card: `Email; Phone; Account Active; Verified; N… | [findings/core-ux.md](findings/core-ux.md) |
| BV-068 | ap-providerdetail-status-fields | core-ux | brand-voice | P3 | Admin provider detail status fields: `Enabled; Yes; No; Active; Inactive; Last… | [findings/core-ux.md](findings/core-ux.md) |
| BV-069 | web-ux-checkout-tip | core-ux | brand-voice | P3 | Web checkout tip: `100% of your tip goes to the home chef` — uses `home chef` l… | [findings/core-ux.md](findings/core-ux.md) |
| BV-071 | ap-staff-invitation-statuses | core-ux | brand-voice | P3 | Staff invitation statuses (admin + delivery-portal): `Pending; Accepted; Expire… | [findings/core-ux.md](findings/core-ux.md) |
| BV-074 | mc-favorites-title | core-ux | brand-voice | P3 | Mobile-customer favorites: tab label `Saved` but page title `Saved Chefs`. Web… | [findings/core-ux.md](findings/core-ux.md) |
| BV-075 | web-ux-social-feed-heading | core-ux | brand-voice | P3 | Web `Chef's Feed` (apostrophe-s, possessive); mobile `Social Feed`; chef-side `… | [findings/core-ux.md](findings/core-ux.md) |
| BV-076 | vp-ux-dashboard-no-pending | core-ux | brand-voice | P3 | Vendor-portal `All caught up / New orders will appear here.` — good empty-state… | [findings/core-ux.md](findings/core-ux.md) |
| BV-077 | vp-ux-settings-password | core-ux | brand-voice | P3 | Vendor-portal password: `Password updated successfully` (status) and `Failed to… | [findings/core-ux.md](findings/core-ux.md) |
| BV-078 | vp-ux-orders-history-summary | core-ux | brand-voice | P3 | Vendor-portal order-history summary cards: `Delivered / Cancelled / Revenue` —… | [findings/core-ux.md](findings/core-ux.md) |
| BV-079 | dp-ux-stripe-make-primary | core-ux | brand-voice | P3 | Delivery-portal driver-facing CTA: `Make Stripe My Primary Gateway` — Title Cas… | [findings/core-ux.md](findings/core-ux.md) |
| BV-080 | dp-ux-stripe-active-gateway | core-ux | brand-voice | P3 | Delivery-portal `Active payout gateway: {Stripe/Razorpay}` — sentence-case `Act… | [findings/core-ux.md](findings/core-ux.md) |
| BV-081 | web-ux-chefdetail-reviews | core-ux | brand-voice | P3 | Web chef detail: `Customer Reviews / Chef's Response:` (with trailing colon and… | [findings/core-ux.md](findings/core-ux.md) |
| BV-082 | mv-reviewdetail-input-ph | core-ux | brand-voice | P3 | Mobile-vendor review reply placeholder: `Write a thoughtful reply to this revie… | [findings/core-ux.md](findings/core-ux.md) |
| BV-083 | ap-secsettings-2fa-account | core-ux | brand-voice | P3 | Admin 2FA: `2FA for your account / Enabled — login requires a 6-digit code / Di… | [findings/core-ux.md](findings/core-ux.md) |
| BV-084 | ap-secsettings-session-fields | core-ux | brand-voice | P3 | Admin security: `Access token TTL (hours)` / `Refresh token TTL (days)` — `TTL`… | [findings/core-ux.md](findings/core-ux.md) |
| BV-085 | ap-secsettings-apikey-fields | core-ux | brand-voice | P3 | Admin API key form: `I've saved it` button after revealing the secret. Cute but… | [findings/core-ux.md](findings/core-ux.md) |
| BV-088 | md-core-066 | core-ux | brand-voice | P3 | Mobile-delivery settings: `View Subscription Plan` — feature visible but likely… | [findings/core-ux.md](findings/core-ux.md) |
| BV-089 | mv-settings-delete-account | core-ux | brand-voice | P3 | `Delete Account` (mobile-vendor + mobile-delivery) — Title Case for the most co… | [findings/core-ux.md](findings/core-ux.md) |
| BV-090 | mv-onb-review-doc-uploaded | core-ux | brand-voice | P3 | Mobile-vendor doc state: `Uploaded` / `Not uploaded` (sentence case — good!) bu… | [findings/core-ux.md](findings/core-ux.md) |
| BV-091 | web-ux-checkout-delivery-time | core-ux | brand-voice | P3 | Web checkout: `As soon as possible / Usually 30-45 minutes / Schedule for later… | [findings/core-ux.md](findings/core-ux.md) |
| BV-092 | web-ux-orderdetail-sections | core-ux | brand-voice | P3 | Web order detail back link: `Back to Orders` (Title Case). Mobile uses `Back` o… | [findings/core-ux.md](findings/core-ux.md) |
| BV-093 | dp-ux-stripe-title | core-ux | brand-voice | P3 | Delivery-portal `Stripe (International Payouts)` — parenthetical Title Case mid… | [findings/core-ux.md](findings/core-ux.md) |
| BV-094 | dp-ux-stripe-intro | core-ux | brand-voice | P3 | Delivery-portal Stripe intro: `Accept delivery payouts in your local currency.… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-025 | mv-menunew-name-ph | core-ux | legal | P3 | Mobile-vendor menu name placeholder 'e.g. Butter Chicken' encourages product na… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-043 | mc-catering-event-types | core-ux | legal | P3 | Mobile catering event-type chip 'Wedding' is a high-stakes contract category —… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-066 | ap-chefs-meta-bottom | core-ux | legal | P3 | Chef cards meta shows 'Min order: ₹{n}; Service radius: {n} km; Joined: {date}'… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-076 | dp-ux-active-order-note | core-ux | legal | P3 | Order note 'Note: {specialInstructions}' — chef-supplied free text shown to dri… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-086 | mc-profile-logout-button | core-ux | legal | P3 | 'Log Out' button label drifts from style-guide 'Sign out' — but more importantl… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-091 | mv-onb-kitchen-cuisine-options | core-ux | legal | P3 | Onboarding cuisine list includes 'Other' as a chip — open-ended FSSAI categoris… | [findings/core-ux.md](findings/core-ux.md) |
| LEG-COREUX-100 | ap-layout-nav-settings | core-ux | legal | P3 | Admin sidebar 'Settings' is the legal/policy-config entry but admin layout has… | [findings/core-ux.md](findings/core-ux.md) |
| TW-044 | pattern-banned-vocabulary-login-signup | core-ux | technical-writer | P3 | PATTERN — Within core-ux surfaces, 'login' / 'Login' appears as a noun. Style g… | [findings/core-ux.md](findings/core-ux.md) |
| TW-055 | pattern-add-new-redundant | core-ux | technical-writer | P3 | PATTERN — 'Add New X' wording is redundant: 'Add' implies new. Style guide §2 p… | [findings/core-ux.md](findings/core-ux.md) |
| TW-058 | pattern-favorite-spelling | core-ux | technical-writer | P3 | PATTERN — 'Favorite' (US) vs 'Favourite' (UK/IN) drift. en-IN locale should pre… | [findings/core-ux.md](findings/core-ux.md) |
| TW-059 | pattern-bottom-tab-page-heading-drift | core-ux | technical-writer | P3 | PATTERN — Mobile bottom-tab label differs from the page heading you land on. | [findings/core-ux.md](findings/core-ux.md) |
| TW-060 | pattern-jargon-est-time | core-ux | technical-writer | P3 | PATTERN — Abbreviated labels on driver-facing time fields. Style guide §2 plain… | [findings/core-ux.md](findings/core-ux.md) |
| TW-061 | pattern-uppercase-fields-acronyms | core-ux | technical-writer | P3 | PATTERN — Acronyms in field labels are inconsistent: 'IFSC Code' (vendor-portal… | [findings/core-ux.md](findings/core-ux.md) |
| TW-062 | pattern-section-headers-double-noun | core-ux | technical-writer | P3 | PATTERN — Section headings use double-noun stacking ('Order Items', 'Order Summ… | [findings/core-ux.md](findings/core-ux.md) |
| TW-063 | pattern-section-headers-vague-actions | core-ux | technical-writer | P3 | PATTERN — Section header 'Actions' is so vague it adds nothing — every section… | [findings/core-ux.md](findings/core-ux.md) |
| TW-064 | web-ux-admin-dashboard | core-ux | technical-writer | P3 | 'Platform at a glance' is fine but 'Recent activity / Shortcuts' as section hea… | [findings/core-ux.md](findings/core-ux.md) |
| TW-065 | web-ux-admin-orders | core-ux | technical-writer | P3 | Search placeholder 'Search order number...' is awkward. Search-by patterns work… | [findings/core-ux.md](findings/core-ux.md) |
| TW-066 | web-ux-chef-menu | core-ux | technical-writer | P3 | 'e.g., 500g' — comma after 'e.g.' is correct US/UK style but inconsistent with… | [findings/core-ux.md](findings/core-ux.md) |
| TW-067 | web-ux-chef-social | core-ux | technical-writer | P3 | '#homemade' hashtag suggestion shown as a default — placeholder hashtag promote… | [findings/core-ux.md](findings/core-ux.md) |
| TW-068 | web-ux-browse-filters-dietary | core-ux | technical-writer | P3 | 'Gluten-Free / Dairy-Free' use hyphen + title case. Style guide silent on hyphe… | [findings/core-ux.md](findings/core-ux.md) |
| TW-069 | web-ux-catering-request-food | core-ux | technical-writer | P3 | Placeholder 'Tell us more about your event, special requests, theme, etc.' uses… | [findings/core-ux.md](findings/core-ux.md) |
| TW-070 | web-ux-catering-request-location | core-ux | technical-writer | P3 | Address placeholder '123 Main Street / Suite 100' is US-style, but app is en-IN… | [findings/core-ux.md](findings/core-ux.md) |
| TW-071 | web-ux-profile-addresses | core-ux | technical-writer | P3 | Address form helpers use 'House / flat / building number, street' and 'Landmark… | [findings/core-ux.md](findings/core-ux.md) |
| TW-072 | web-ux-delivery-dashboard | core-ux | technical-writer | P3 | 'You're currently offline' — 'currently' is filler. 'You're offline' suffices. | [findings/core-ux.md](findings/core-ux.md) |
| TW-073 | dp-ux-dashboard-partner-offline | core-ux | technical-writer | P3 | 'Currently offline' — 'Currently' filler word. | [findings/core-ux.md](findings/core-ux.md) |
| TW-074 | dp-ux-dashboard-no-deliveries-hint | core-ux | technical-writer | P3 | 'New deliveries will appear here.' — passive 'will appear here' is fine but gen… | [findings/core-ux.md](findings/core-ux.md) |
| TW-075 | dp-ux-stripe-sub | core-ux | technical-writer | P3 | Subtitle 'For drivers outside India, or as an alternative to Razorpay.' — sente… | [findings/core-ux.md](findings/core-ux.md) |
| TW-076 | dp-ux-stripe-intro | core-ux | technical-writer | P3 | 'Stripe handles KYC and bank verification on its hosted pages.' — 'KYC' is indu… | [findings/core-ux.md](findings/core-ux.md) |
| TW-077 | dp-ux-partner-readonly-notice | core-ux | technical-writer | P3 | 'This partner is pending verification. Contact a fleet manager to approve.' — '… | [findings/core-ux.md](findings/core-ux.md) |
| TW-078 | dp-ux-partner-doc-meta | core-ux | technical-writer | P3 | '{type} · Uploaded {date}' — middle dot fine but '{date}' should be relative fo… | [findings/core-ux.md](findings/core-ux.md) |
| TW-079 | dp-ux-staff-invite-meta | core-ux | technical-writer | P3 | '{role} · Sent {date}' — same date formatting concern as TW-078. | [findings/core-ux.md](findings/core-ux.md) |
| TW-080 | dp-ux-staff-roles-fallback | core-ux | technical-writer | P3 | Hardcoded role-list fallback ('Delivery Ops / Fleet Manager / Super Admin') mea… | [findings/core-ux.md](findings/core-ux.md) |
| TW-081 | dp-ux-active-status-delivered | core-ux | technical-writer | P3 | Driver-facing status 'Delivered' is sentence case (good). But neighbour 'Picked… | [findings/core-ux.md](findings/core-ux.md) |
| TW-082 | vp-ux-dashboard-pending-card | core-ux | technical-writer | P3 | '+N more items' — concatenation with prefix '+' can break translation (some lan… | [findings/core-ux.md](findings/core-ux.md) |
| TW-083 | vp-ux-dashboard-no-pending | core-ux | technical-writer | P3 | 'All caught up / New orders will appear here.' — good empty-state pattern (why… | [findings/core-ux.md](findings/core-ux.md) |
| TW-084 | vp-ux-dashboard-quick-actions | core-ux | technical-writer | P3 | Tile copy uses comma-separator ('Add Menu Item, Create a new dish listing') whi… | [findings/core-ux.md](findings/core-ux.md) |
| TW-085 | vp-ux-kitchen-photos | core-ux | technical-writer | P3 | Helper text 'JPEG, PNG, or WebP. Max 5 MB each. Up to 5 photos.' — three senten… | [findings/core-ux.md](findings/core-ux.md) |
| TW-086 | vp-ux-profile-business-info | core-ux | technical-writer | P3 | Placeholder 'Tell customers about your kitchen, cooking style, and what makes y… | [findings/core-ux.md](findings/core-ux.md) |
| TW-087 | vp-ux-settings-notifications | core-ux | technical-writer | P3 | Long stack of notification descriptions ('Get notified when a new order comes i… | [findings/core-ux.md](findings/core-ux.md) |
| TW-088 | vp-ux-settings-password | core-ux | technical-writer | P3 | Error 'Failed to update password. Check your current password.' — formula 'what… | [findings/core-ux.md](findings/core-ux.md) |
| TW-089 | mc-orders-filters | core-ux | technical-writer | P3 | 'All, Active, Delivered, Cancelled' filter chips — drift with web-ux-orders-fil… | [findings/core-ux.md](findings/core-ux.md) |
| TW-090 | mc-profile-cuisines | core-ux | technical-writer | P3 | Inventory note: order of cuisines differs from onboarding step 3. Adds cognitiv… | [findings/core-ux.md](findings/core-ux.md) |
| TW-091 | mc-profile-more-rows | core-ux | technical-writer | P3 | 'Social Feed / Catering' rows shown with emoji icons (📱/🍽️). Brand 'quietly mod… | [findings/core-ux.md](findings/core-ux.md) |
| TW-092 | mv-onb-pending-rejected-cta | core-ux | technical-writer | P3 | 'Reapply' single-word CTA after rejection. Friendly, but could be 'Apply again'… | [findings/core-ux.md](findings/core-ux.md) |
| TW-093 | mv-menunew-desc-ph | core-ux | technical-writer | P3 | Placeholder 'Describe your dish (at least 20 characters)' — character count in… | [findings/core-ux.md](findings/core-ux.md) |
| TW-094 | mv-menunew-type-veg | core-ux | technical-writer | P3 | 'Veg / Non-Veg' toggle — India-specific but inconsistent hyphenation: 'Non-Veg'… | [findings/core-ux.md](findings/core-ux.md) |
| TW-095 | md-core-009 | core-ux | technical-writer | P3 | 'Toggle to start receiving requests' — driver tone matrix is ≤4 words telegraph… | [findings/core-ux.md](findings/core-ux.md) |
| TW-096 | md-core-018 | core-ux | technical-writer | P3 | 'Go Online to Accept Deliveries' — 5 words, title case, on a driver-facing CTA.… | [findings/core-ux.md](findings/core-ux.md) |
| TW-097 | md-core-031 | core-ux | technical-writer | P3 | 'Delivery Cancelled' terminal status uses British spelling 'Cancelled' (good fo… | [findings/core-ux.md](findings/core-ux.md) |
| TW-098 | md-core-039 | core-ux | technical-writer | P3 | Banner copy 'This delivery has been cancelled. Check the Available tab for new… | [findings/core-ux.md](findings/core-ux.md) |
| TW-099 | md-core-044 | core-ux | technical-writer | P3 | 'Order Summary ({n} item(s))' — uses 'item(s)' notation. Style guide §6: never… | [findings/core-ux.md](findings/core-ux.md) |
| TW-100 | md-core-099 | core-ux | technical-writer | P3 | 'Today: {n} deliveries' — colon + concatenation, will break in some languages (… | [findings/core-ux.md](findings/core-ux.md) |
| TW-101 | md-core-109 | core-ux | technical-writer | P3 | 'Joined {date}' — relative date pattern; same issue as TW-078. | [findings/core-ux.md](findings/core-ux.md) |
| TW-102 | md-core-046 | core-ux | technical-writer | P3 | 'Special instructions:' — trailing colon (see TW-039 pattern). Driver surface. | [findings/core-ux.md](findings/core-ux.md) |
| TW-103 | dp-ux-stripe-resume-onboarding | core-ux | technical-writer | P3 | 'Resume Onboarding' — title case + 'onboarding' is dev jargon to a driver. Styl… | [findings/core-ux.md](findings/core-ux.md) |
| TW-104 | ap-approvals-types | core-ux | technical-writer | P3 | Approval-type taxonomy uses title case ('Kitchen Onboarding', 'Document Verific… | [findings/core-ux.md](findings/core-ux.md) |
| TW-105 | ap-providerdetail-status-fields | core-ux | technical-writer | P3 | 'Enabled / Yes / No / Active / Inactive / Last Used / Created / Never / Unlimit… | [findings/core-ux.md](findings/core-ux.md) |
| TW-106 | ap-secsettings-2fa-account | core-ux | technical-writer | P3 | 'Enabled — login requires a 6-digit code' / 'Disabled — enable to protect your… | [findings/core-ux.md](findings/core-ux.md) |
| TW-107 | ap-secsettings-apikey-fields | core-ux | technical-writer | P3 | 'I've saved it' button label is unusual — first-person voice on a button. Style… | [findings/core-ux.md](findings/core-ux.md) |
| TW-108 | ap-settings-card-titles | core-ux | technical-writer | P3 | Settings card titles 'Payment Gateway' / 'Stripe Gateway' — 'gateway' is jargon… | [findings/core-ux.md](findings/core-ux.md) |
| TW-109 | ap-exports-cards | core-ux | technical-writer | P3 | Export card descriptions leak database column names: 'All user accounts (id, em… | [findings/core-ux.md](findings/core-ux.md) |
| TW-110 | ap-platsettings-zones-fields | core-ux | technical-writer | P3 | Zone CRUD labels 'Min latitude / Max latitude / Min longitude / Max longitude'… | [findings/core-ux.md](findings/core-ux.md) |
| TW-111 | vp-ux-orders-history-title | core-ux | technical-writer | P3 | Page header 'Order History / {n} orders found / Live Orders / Back' — 'Back' as… | [findings/core-ux.md](findings/core-ux.md) |
| TW-112 | vp-ux-kitchen-photos | core-ux | technical-writer | P3 | Drop-zone copy 'Click or drag photos here' / 'Drop your photo here' — two disti… | [findings/core-ux.md](findings/core-ux.md) |
| TW-113 | vp-ux-menu-view-page | core-ux | technical-writer | P3 | Long mixed-content surface includes 'Item ID' — exposing internal IDs is fine f… | [findings/core-ux.md](findings/core-ux.md) |
| TW-114 | dp-ux-active-order-note | core-ux | technical-writer | P3 | 'Note: {specialInstructions}' — colon + concatenation issue per TW-039 / TW-082. | [findings/core-ux.md](findings/core-ux.md) |
| TW-116 | dp-ux-active-cancel-btn | core-ux | technical-writer | P3 | 'Cancel Delivery' destructive button — title case. Driver telegraphic + sentenc… | [findings/core-ux.md](findings/core-ux.md) |
| TW-117 | dp-ux-staff-invite-h | core-ux | technical-writer | P3 | Heading 'Send Staff Invitation' — title case + redundant ('staff invitation' on… | [findings/core-ux.md](findings/core-ux.md) |
| TW-118 | ap-staff-invite-dialog-title | core-ux | technical-writer | P3 | Dialog title 'Invite Staff Member' — title case, 3 words. 'Staff member' is red… | [findings/core-ux.md](findings/core-ux.md) |
| TW-119 | ap-staffdetail-info-rows | core-ux | technical-writer | P3 | 'Joined' / 'Last Login' info rows — 'Login' (noun) banned per §3 + title case. | [findings/core-ux.md](findings/core-ux.md) |
| TW-120 | ap-userdetail-info-rows | core-ux | technical-writer | P3 | 'Auth Provider' / 'Last Login' — 'Login' (noun) banned + 'Auth' jargon. | [findings/core-ux.md](findings/core-ux.md) |
| TW-121 | ap-userdetail-verification | core-ux | technical-writer | P3 | 'Account Active' / 'Not verified' verification labels mix tense and concept (ac… | [findings/core-ux.md](findings/core-ux.md) |
| TW-122 | web-ux-orderdetail-cancel | core-ux | technical-writer | P3 | Modal subtitle 'Please let us know why you're cancelling...' starts with 'Pleas… | [findings/core-ux.md](findings/core-ux.md) |
| TW-123 | web-ux-chef-profile | core-ux | technical-writer | P3 | Placeholder 'Tell customers about your kitchen...' — ellipsis suggests open-end… | [findings/core-ux.md](findings/core-ux.md) |
| TW-124 | web-ux-chefdetail-reviews | core-ux | technical-writer | P3 | 'Chef's Response:' (trailing colon) — see TW-039 pattern. Also possessive 'Chef… | [findings/core-ux.md](findings/core-ux.md) |
| TW-125 | web-ux-orderdetail-sections | core-ux | technical-writer | P3 | 'Placed on {date} / Estimated delivery:' — colon + 'Estimated delivery' is word… | [findings/core-ux.md](findings/core-ux.md) |
| TW-126 | ap-staff-roles | core-ux | technical-writer | P3 | Role taxonomy 'Super Admin / Admin / Fleet Manager / Delivery Ops / Support' —… | [findings/core-ux.md](findings/core-ux.md) |
| TW-127 | dp-ux-active-pickup-from | core-ux | technical-writer | P3 | Eyebrow 'Pickup from' (uppercase via CSS — inventory notes). Sentence-case stri… | [findings/core-ux.md](findings/core-ux.md) |
| TW-128 | dp-ux-active-deliver-to | core-ux | technical-writer | P3 | Eyebrow 'Deliver to' — sister to TW-127. Same treatment. | [findings/core-ux.md](findings/core-ux.md) |
| TW-129 | web-ux-checkout-address-form | core-ux | technical-writer | P3 | 'Apartment, suite, etc. (optional)' — US 'suite' + 'etc.' filler in a placehold… | [findings/core-ux.md](findings/core-ux.md) |
| TW-130 | dp-ux-fleet-h | core-ux | technical-writer | P3 | Subtitle 'Monitor your delivery fleet in real time' — 'real time' (two words) v… | [findings/core-ux.md](findings/core-ux.md) |
| TW-131 | dp-ux-partner-detail-verify-block | core-ux | technical-writer | P3 | 'Pending Verification' eyebrow + 'This partner is awaiting verification. Review… | [findings/core-ux.md](findings/core-ux.md) |
| TW-132 | mc-cartsheet-checkout-cta | core-ux | technical-writer | P3 | 'Proceed to Checkout' — 3 words, title case, redundant verb ('proceed to' = 'go… | [findings/core-ux.md](findings/core-ux.md) |
| TW-133 | ap-notifsettings-categories | core-ux | technical-writer | P3 | Category descriptions vary in tone: 'Promotions / Discounts, newsletters, and m… | [findings/core-ux.md](findings/core-ux.md) |
| TW-134 | dp-ux-stripe-title | core-ux | technical-writer | P3 | 'Stripe (International Payouts)' — parenthetical title case. Sentence case thro… | [findings/core-ux.md](findings/core-ux.md) |
| TW-135 | mv-onb-docs-pdf-uploaded | core-ux | technical-writer | P3 | 'PDF uploaded' — past-tense success, good. Mobile shows 'Uploaded' on mv-onb-re… | [findings/core-ux.md](findings/core-ux.md) |
| TW-136 | mv-onb-review-doc-not-uploaded | core-ux | technical-writer | P3 | 'Not uploaded' — two words, fine. Could pair with action: 'Not uploaded — Uploa… | [findings/core-ux.md](findings/core-ux.md) |
| TW-137 | mv-tabs-orders-label | core-ux | technical-writer | P3 | Inventory note: matches vendor-portal. Just verifying — no issue here. Includin… | [findings/core-ux.md](findings/core-ux.md) |
| TW-138 | mv-undo-cta | core-ux | technical-writer | P3 | Snackbar action 'UNDO' is ALL CAPS. Inventory notes Material-style convention;… | [findings/core-ux.md](findings/core-ux.md) |
| TW-139 | md-core-091 | core-ux | technical-writer | P3 | Verification badge uses '✓ Verified' (with checkmark glyph) vs 'Pending' (no gl… | [findings/core-ux.md](findings/core-ux.md) |
| TW-140 | md-core-006 | core-ux | technical-writer | P3 | Status banner 'You are Online' — capital O on second word (title case for one w… | [findings/core-ux.md](findings/core-ux.md) |
| TW-141 | md-core-016 | core-ux | technical-writer | P3 | 'You're Offline' — title case 'Offline'. Drift with md-core-006 ('You are Onlin… | [findings/core-ux.md](findings/core-ux.md) |
| TW-142 | dp-ux-dashboard-partner-online | core-ux | technical-writer | P3 | 'Ready for deliveries' — 3 words, good driver tone. 'Standing by' / 'Waiting ne… | [findings/core-ux.md](findings/core-ux.md) |
| TW-143 | dp-ux-dashboard-available-waiting | core-ux | technical-writer | P3 | 'Waiting nearby' — relates to TW-142. Two-word phrase suggests 'partners are ne… | [findings/core-ux.md](findings/core-ux.md) |
| TW-144 | dp-ux-dashboard-standing-by | core-ux | technical-writer | P3 | 'Standing by' — driver-facing but ambiguous (military-ish phrase). 'Online' or… | [findings/core-ux.md](findings/core-ux.md) |
| TW-145 | dp-ux-staff-invite-created | core-ux | technical-writer | P3 | 'Invitation Created' / 'Share this link with the new staff member:' — title cas… | [findings/core-ux.md](findings/core-ux.md) |
| TW-146 | ap-secsettings-session-fields | core-ux | technical-writer | P3 | 'Access token TTL (hours)' / 'Refresh token TTL (days)' — TTL is dev jargon. St… | [findings/core-ux.md](findings/core-ux.md) |
| TW-147 | ap-secsettings-apikey-fields | core-ux | technical-writer | P3 | 'Expires in (days; 0 = never)' — inline programming convention 'use 0 to mean n… | [findings/core-ux.md](findings/core-ux.md) |
| TW-148 | ap-platsettings-hours-fields | core-ux | technical-writer | P3 | 'Opening time (HH:MM)' / 'Closing time (HH:MM)' — format hint in label, dev sty… | [findings/core-ux.md](findings/core-ux.md) |
| TW-149 | ap-providercreate-fields | core-ux | technical-writer | P3 | Long field list mixes plain ('Name') with jargon ('API Base URL', 'Pricing Mode… | [findings/core-ux.md](findings/core-ux.md) |
| TW-150 | dp-ux-partner-performance | core-ux | technical-writer | P3 | 'CSAT Score' uses CSAT acronym without definition; admin reader may not know. | [findings/core-ux.md](findings/core-ux.md) |
| TW-151 | ap-notifsettings-subtitle | core-ux | technical-writer | P3 | 'Control which channels deliver each notification category for your account. Tr… | [findings/core-ux.md](findings/core-ux.md) |
| TW-152 | ap-notifsettings-categories | core-ux | technical-writer | P3 | Category descriptions use parenthetical aside '(chef accounts only)' which is a… | [findings/core-ux.md](findings/core-ux.md) |
| TW-153 | mv-onb-personal-cta-next | core-ux | technical-writer | P3 | Primary CTA 'Next' across onboarding steps 1-5 — fine for navigation, but final… | [findings/core-ux.md](findings/core-ux.md) |
| TW-154 | web-ux-admin-layout-nav | core-ux | technical-writer | P3 | Aria labels 'Main navigation / Close navigation / Open navigation / Search... /… | [findings/core-ux.md](findings/core-ux.md) |
| TW-155 | web-ux-chef-layout-nav | core-ux | technical-writer | P3 | Aria labels 'Notifications, unread' — no count or pluralisation. Less useful fo… | [findings/core-ux.md](findings/core-ux.md) |
| TW-156 | web-ux-delivery-layout-toggles | core-ux | technical-writer | P3 | Aria-label 'Currently online — tap to toggle availability' — 'Currently' filler… | [findings/core-ux.md](findings/core-ux.md) |
| TW-157 | web-ux-layout-nav | core-ux | technical-writer | P3 | Top-nav primary destinations 'Home / Browse Chefs / Favorites / Catering / Prof… | [findings/core-ux.md](findings/core-ux.md) |
| TW-158 | vp-ux-layout-nav | core-ux | technical-writer | P3 | Sidebar nav 'Dashboard / Menu / Orders / Earnings / Admin Requests / Reviews /… | [findings/core-ux.md](findings/core-ux.md) |
| TW-159 | ap-layout-topbar-title | core-ux | technical-writer | P3 | 'Fe3dr Administration' — brand 'Fe3dr' next to 'Administration' is the wrong pr… | [findings/core-ux.md](findings/core-ux.md) |
| TW-160 | dp-ux-logo-text | core-ux | technical-writer | P3 | Logo text 'Fe3dr + Delivery (tagline)' — same brand drift question as TW-159. | [findings/core-ux.md](findings/core-ux.md) |
| TW-161 | web-ux-chef-dashboard-sections | core-ux | technical-writer | P3 | 'Today at a glance / Pending orders / Shortcuts / Recent orders' — mostly sente… | [findings/core-ux.md](findings/core-ux.md) |
| TW-162 | web-ux-chef-orders-heading | core-ux | technical-writer | P3 | 'Delivery Information / Order Items / Special Instructions / Payment Summary' —… | [findings/core-ux.md](findings/core-ux.md) |
| TW-163 | dp-ux-active-h1 | core-ux | technical-writer | P3 | Page H1 'Active Delivery' — title case. Single-word sufficient ('Active') for d… | [findings/core-ux.md](findings/core-ux.md) |
| TW-164 | dp-ux-available-h1 | core-ux | technical-writer | P3 | Page H1 'Available Deliveries' — title case + plural. Match driver tab label 'A… | [findings/core-ux.md](findings/core-ux.md) |
| TW-165 | dp-ux-available-count | core-ux | technical-writer | P3 | '{n} orders waiting for pickup' — 5 words, plural-aware concatenation. ICU plur… | [findings/core-ux.md](findings/core-ux.md) |
| TW-166 | dp-ux-accept-delivery-btn | core-ux | technical-writer | P3 | 'Accept Delivery / Accepting...' — title case + three-dot ellipsis (TW-040). 'A… | [findings/core-ux.md](findings/core-ux.md) |
| TW-167 | md-core-022 | core-ux | technical-writer | P3 | Mobile driver 'Accept Delivery' — title case. Matches dp-ux-accept-delivery-btn… | [findings/core-ux.md](findings/core-ux.md) |
| TW-168 | md-core-008 | core-ux | technical-writer | P3 | 'Receiving delivery requests' — 3 words, OK as online subtitle. Compare md-core… | [findings/core-ux.md](findings/core-ux.md) |
| TW-169 | dp-ux-staff-invite-btn | core-ux | technical-writer | P3 | 'Invite Staff' — title case. 2 words, acceptable length. | [findings/core-ux.md](findings/core-ux.md) |
| TW-170 | ap-staff-invite-fields | core-ux | technical-writer | P3 | Form labels 'Email; Role; Department; Title; Personal Message; Send Invitation;… | [findings/core-ux.md](findings/core-ux.md) |
| TW-171 | ap-providerdetail-status-fields | core-ux | technical-writer | P3 | 'Never' / 'Unlimited' — single-word status fillers. Need context. | [findings/core-ux.md](findings/core-ux.md) |
| TW-172 | vp-ux-menu-view-page | core-ux | technical-writer | P3 | Long status-and-data surface includes 'Under Review' / 'Available' / 'Unavailab… | [findings/core-ux.md](findings/core-ux.md) |
| TW-173 | mv-orders-tab-live | core-ux | technical-writer | P3 | Tab label 'Live Queue' — 'queue' is internal jargon. Most chef apps say 'Live o… | [findings/core-ux.md](findings/core-ux.md) |
| TW-174 | vp-ux-bottom-nav | core-ux | technical-writer | P3 | Mobile bottom nav 'Dashboard / Menu / Orders / Earnings / Profile / Vendor navi… | [findings/core-ux.md](findings/core-ux.md) |
| TW-175 | dp-ux-history-subtitle | core-ux | technical-writer | P3 | 'Your past deliveries' — 3 words, fine. Marginal: 'Past deliveries' (drop 'Your… | [findings/core-ux.md](findings/core-ux.md) |
| TW-176 | dp-ux-earnings-subtitle | core-ux | technical-writer | P3 | 'Track your delivery income' — 4 words, 'Your' filler (TW-052). 'Income' fine b… | [findings/core-ux.md](findings/core-ux.md) |
| TW-177 | dp-ux-stat-rating | core-ux | technical-writer | P3 | 'Rating' + '{n} reviews' / 'No reviews yet' — empty state pattern OK. Consider… | [findings/core-ux.md](findings/core-ux.md) |
| TW-178 | ap-secsettings-2fa-account | core-ux | technical-writer | P3 | Inline state copy '— login requires...' / '— enable to protect...' uses 'login'… | [findings/core-ux.md](findings/core-ux.md) |
| TW-179 | ap-platsettings-cards | core-ux | technical-writer | P3 | Card subtitles 'Service fee, tax, and payout percentages' / 'Base fee + per-km… | [findings/core-ux.md](findings/core-ux.md) |
| TW-180 | mc-checkout-totals | core-ux | technical-writer | P3 | 'Free' as a row value alongside 'Subtotal / Delivery fee / Total' — when delive… | [findings/core-ux.md](findings/core-ux.md) |
| TW-181 | md-core-080 | core-ux | technical-writer | P3 | Detail rows 'Distance / Completed / Payout' — single-word labels OK. No issue. | [findings/core-ux.md](findings/core-ux.md) |
| TW-182 | md-core-084 | core-ux | technical-writer | P3 | 'Phone / City / Email' — single-word field labels, fine. No issue. | [findings/core-ux.md](findings/core-ux.md) |
| TW-183 | md-core-090 | core-ux | technical-writer | P3 | 'Edit / Cancel' action set — single words, fine. | [findings/core-ux.md](findings/core-ux.md) |
| TW-184 | vp-ux-payouts-title | core-ux | technical-writer | P3 | Back-link 'Back to Earnings' — title case. Sentence case. | [findings/core-ux.md](findings/core-ux.md) |
| TW-185 | vp-ux-menu-view-page | core-ux | technical-writer | P3 | 'Back to Menu' — title case (TW-034 pattern). | [findings/core-ux.md](findings/core-ux.md) |
| TW-186 | dp-ux-partner-detail-back | core-ux | technical-writer | P3 | 'Back to Partners' — title case + plural noun. | [findings/core-ux.md](findings/core-ux.md) |
| TW-187 | vp-ux-kitchen-title | core-ux | technical-writer | P3 | 'Back to Profile' — title case (TW-034 pattern). Combined into the header with… | [findings/core-ux.md](findings/core-ux.md) |
| TW-188 | ap-platsettings-zones-fields | core-ux | technical-writer | P3 | 'New zone / Create zone / Cancel' — 'Create zone' acceptable but 'New zone' as… | [findings/core-ux.md](findings/core-ux.md) |
| TW-189 | ap-providercreate-fields | core-ux | technical-writer | P3 | 'Logo URL' / 'API Base URL' / 'Webhook Secret' — admin jargon acceptable but ti… | [findings/core-ux.md](findings/core-ux.md) |
| TW-190 | ap-delivery-stats | core-ux | technical-writer | P3 | '{n} online now; Active deliveries; Delivered today; Today's payouts' — mix of… | [findings/core-ux.md](findings/core-ux.md) |
| TW-191 | ap-dashboard-shortcuts-items | core-ux | technical-writer | P3 | 'Manage users / View all users; Chef verification / Review applications; Order… | [findings/core-ux.md](findings/core-ux.md) |
| TW-192 | ap-userdetail-order-stats | core-ux | technical-writer | P3 | 'Total Orders / Total Spent / Last Order / No orders yet' — title case + 'No or… | [findings/core-ux.md](findings/core-ux.md) |
| TW-193 | vp-ux-orders-history-summary | core-ux | technical-writer | P3 | Summary cards 'Delivered / Cancelled / Revenue' — single nouns, fine. 'Revenue'… | [findings/core-ux.md](findings/core-ux.md) |
| TW-194 | web-ux-chef-earnings | core-ux | technical-writer | P3 | 'Earnings / Earnings Overview / Top Selling Items / Recent Payouts / Payment Se… | [findings/core-ux.md](findings/core-ux.md) |
| TW-195 | web-ux-admin-analytics | core-ux | technical-writer | P3 | 'Analytics / Revenue Overview / Orders by Status / Top Performing Chefs / Popul… | [findings/core-ux.md](findings/core-ux.md) |
| TW-196 | mv-orders-tab-history | core-ux | technical-writer | P3 | Tab 'History' — single word, fine. Pair label 'Live Queue' (TW-173) should matc… | [findings/core-ux.md](findings/core-ux.md) |
| TW-197 | md-core-068 | core-ux | technical-writer | P3 | 'App Version' footer label — title case (TW-037 pattern). One-time fix. | [findings/core-ux.md](findings/core-ux.md) |
| TW-198 | dp-ux-active-order-note | core-ux | technical-writer | P3 | Inventory notes 'chef-supplied instructions' — content from another role can in… | [findings/core-ux.md](findings/core-ux.md) |
| TW-199 | dp-ux-active-status-picked-up | core-ux | technical-writer | P3 | 'Picked Up' status — title case 'Picked Up' (TW-033 + TW-081). | [findings/core-ux.md](findings/core-ux.md) |
| TW-200 | dp-ux-active-status-in-transit | core-ux | technical-writer | P3 | 'In Transit' — title case. | [findings/core-ux.md](findings/core-ux.md) |

### Index — errors-empty

401 findings. Detail: [findings/errors-empty.md](findings/errors-empty.md)

| finding_id | surface_id | category | lens | severity | issue (≤80 chars) | detail link |
|---|---|---|---|---|---|---|
| BA-ERR-001 | ap-analytics-coming-soon | errors-empty | business-analyst | P0 | "Chart coming soon" placeholder ships to admin operators — fake/unfinished dash… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-002 | api-error-payment-config | errors-empty | business-analyst | P0 | Internal payment gateway config error leaked verbatim to end customer — kills c… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-003 | api-error-stripe-connect | errors-empty | business-analyst | P0 | Raw API paths exposed in error messages to chef users — signals broken product | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-004 | web-err-checkout-cancel | errors-empty | business-analyst | P0 | Payment cancellation produces only "Payment cancelled" with no next-step CTA —… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-005 | mc-checkout-errors | errors-empty | business-analyst | P0 | Payment confirmation timeout on mobile presents only a vague message with no cl… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-006 | api-error-auth-suspended | errors-empty | business-analyst | P0 | "Account is suspended" with no reason or support path — dead end for user | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-001 | web-err-login-generic | errors-empty | brand-voice | P0 | 'Something went wrong. Please try again.' generic copy duplicated verbatim acro… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-003 | web-err-error-boundary | errors-empty | brand-voice | P0 | Error-boundary fallback copy drifts heavily across apps: web uses 'Something br… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-005 | vp-ux-orders-live-empty | errors-empty | brand-voice | P0 | 'No pending orders' empty state drifts between vendor-portal ('All caught up! /… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-006 | web-ux-cart-empty | errors-empty | brand-voice | P0 | Cart-empty copy drifts: web 'Your cart is empty. Looks like you haven't added a… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-032 | api-error-stripe-connect | errors-empty | brand-voice | P0 | 'No Stripe account — call /chef/stripe/connect first' leaks an API path to end… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-041 | ap-analytics-coming-soon | errors-empty | brand-voice | P0 | 'Chart coming soon' shipped to production admin surface — inventory notes flag… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-049 | api-error-auth-unauthorized | errors-empty | brand-voice | P0 | Same auth-failure concept across API: 'Unauthorized' / 'unauthorized' / 'Authen… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-053 | api-error-upload-size | errors-empty | brand-voice | P0 | '10MB' vs '5 MB' vs '5MB' — three different formats of the same unit on the sam… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-062 | api-error-validation-required | errors-empty | brand-voice | P0 | 18 variants of 'X is required' across API handlers; inventory notes 'Inconsiste… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-063 | api-error-validation-invalid-id | errors-empty | brand-voice | P0 | ~40 variants of 'Invalid X' / 'Invalid X ID'; inventory flags this. Customer se… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-064 | api-error-generic-failed | errors-empty | brand-voice | P0 | 100+ 'Failed to X' variants across API; inventory: 'Should generally not surfac… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-001 | api-error-stripe-connect | errors-empty | legal | P0 | Error messages leak internal API routes to end users ('call /chef/stripe/connec… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-002 | api-error-payment-config | errors-empty | legal | P0 | Payment-gateway configuration error leaks internal infrastructure status to the… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-008 | api-error-phone-duplicate | errors-empty | legal | P0 | Registration error 'This phone number is already registered with another accoun… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-013 | api-error-auth-suspended | errors-empty | legal | P0 | 'Account is suspended' provides no reason, no appeal pathway, no Grievance Offi… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-025 | api-error-payment-refund | errors-empty | legal | P0 | Refund-error responses leak who-can-initiate ('Only the chef or admin can initi… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-027 | mc-checkout-errors | errors-empty | legal | P0 | Mobile customer checkout error 'Payment confirmation timed out. Check your orde… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-029 | api-error-order-cancel-stage | errors-empty | legal | P0 | 'Order cannot be cancelled at this stage' surfaces no refund/no-refund implicat… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-032 | mv-onb-docs-required-alert | errors-empty | legal | P0 | Mobile-vendor onboarding error 'Please upload both ID proof and FSSAI license t… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-035 | vp-onb-banner-rejected-body | errors-empty | legal | P0 | Onboarding-rejection banner 'Your previous application was not approved. Please… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-038 | api-error-tos | errors-empty | legal | P0 | Driver onboarding error 'You must accept the terms and conditions' is correct i… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-039 | dp-err-step5-terms | errors-empty | legal | P0 | Driver web-portal error 'Please accept the terms and conditions' — same as LEG-… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-040 | mv-onb-policies-terms-required-alert | errors-empty | legal | P0 | Mobile-vendor 'Please accept the terms and conditions to continue.' — chef/vend… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-058 | dp-err-boundary-body | errors-empty | legal | P0 | Driver-portal error boundary 'Something went wrong / An unexpected error occurr… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-001 | api-error-payment-config | errors-empty | technical-writer | P0 | Customer-facing payment error leaks an internal infra state ('Stripe gateway no… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-002 | api-error-stripe-connect | errors-empty | technical-writer | P0 | Error string exposes a literal API path to the end user: 'No Stripe account — c… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-003 | api-error-order-cancel-stage | errors-empty | technical-writer | P0 | 'Order cannot be cancelled at this stage' tells the customer NO about a refund-… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-004 | api-error-payment-refund | errors-empty | technical-writer | P0 | Refund-constraint strings expose internal rules to the user but never communica… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-005 | api-error-upload-size | errors-empty | technical-writer | P0 | Four near-identical 'file too large' errors use three different size formats: '… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-007 | web-ux-cart-empty | errors-empty | business-analyst | P1 | Empty cart copy is passive and does not name a specific chef discovery path | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-008 | web-err-favorites-empty | errors-empty | business-analyst | P1 | Empty favorites state offers no prompt to discover or save a chef — missed acti… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-009 | mc-favorites-empty | errors-empty | business-analyst | P1 | Mobile empty favorites uses a trailing exclamation mark and emoji — violates vo… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-010 | web-err-orders-empty | errors-empty | business-analyst | P1 | Empty orders state for a filtered view says "No {filter} orders found" with no… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-011 | mc-orders-empty | errors-empty | business-analyst | P1 | Mobile orders empty state uses a plate emoji in functional copy — violates styl… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-012 | vp-ux-menu-empty | errors-empty | business-analyst | P1 | New chef empty menu state has a CTA "Add Your First Item" but no walkthrough gu… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-013 | mv-menu-empty | errors-empty | business-analyst | P1 | Mobile vendor empty menu state has no consequence framing — chef does not know… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-014 | vp-ux-payouts-empty | errors-empty | business-analyst | P1 | Empty payouts state gives no guidance on when the first payout should be expect… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-015 | vp-onb-banner-rejected-title | errors-empty | business-analyst | P1 | Application rejection banner uses title case "Application Rejected" — cold and… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-016 | vp-onb-banner-rejected-body | errors-empty | business-analyst | P1 | Rejection body uses "was not approved" passive and "re-submit" hyphenated — inc… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-017 | api-error-order-cancel-stage | errors-empty | business-analyst | P1 | "Order cannot be cancelled at this stage" — no refund implications, no alternat… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-018 | api-error-chef-not-accepting | errors-empty | business-analyst | P1 | "Chef is not accepting orders" — no alternative suggestion, dead end for custom… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-019 | web-err-error-boundary | errors-empty | business-analyst | P1 | Global error boundary uses "We've been notified" — unverified trust claim that… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-020 | vp-err-boundary-fallback | errors-empty | business-analyst | P1 | Vendor portal error boundary has no support path — chef in active order managem… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-021 | dp-err-boundary-title | errors-empty | business-analyst | P1 | Delivery portal error boundary identical to vendor and web — no role-specific r… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-022 | dp-err-access-denied | errors-empty | business-analyst | P1 | "Access denied. Please check your credentials and try again" — security-evoking… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-023 | vp-auth-login-access-denied | errors-empty | business-analyst | P1 | Vendor portal access denied message references "Fe3dr customer app" — an incorr… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-024 | vp-ux-orders-live-empty | errors-empty | business-analyst | P1 | Live orders empty state is passive — does not tell a new chef what to do to get… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-004 | web-err-layout-offline | errors-empty | brand-voice | P1 | Offline banner copy drifts: web 'You're offline. Some features may be unavailab… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-009 | vp-ux-menu-empty | errors-empty | brand-voice | P1 | Vendor menu-empty CTA 'Add Your First Item' is Title Case — STYLE-GUIDE bans Ti… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-010 | dp-empty-active-title | errors-empty | brand-voice | P1 | Driver app titles use Title Case ('No Active Delivery', 'No Available Deliverie… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-011 | md-emp-006 | errors-empty | brand-voice | P1 | Mobile-delivery uses Title Case heading 'Fleet Management' and 'Staff Managemen… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-013 | web-err-login-generic | errors-empty | brand-voice | P1 | 'Something went wrong' pattern appears 6+ times across the codebase — STYLE-GUI… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-020 | mc-favorites-empty | errors-empty | brand-voice | P1 | 'No saved chefs yet / Tap the heart on any chef to save them!' — exclamation ma… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-021 | mc-orders-empty | errors-empty | brand-voice | P1 | 'Browse chefs to place your first order!' — exclamation on customer empty state… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-033 | api-error-delivery-already-active | errors-empty | brand-voice | P1 | 'no_active_delivery' snake_case slug surfaced as user-facing error string — inv… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-044 | mc-checkout-errors | errors-empty | brand-voice | P1 | 'Payment confirmation timed out. Check your order history to confirm status.' —… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-045 | api-error-payment-config | errors-empty | brand-voice | P1 | 'Stripe gateway not configured by platform admin' — leaks internal config state… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-047 | api-error-order-cancel-stage | errors-empty | brand-voice | P1 | 'Order cannot be cancelled at this stage' — no refund implication, no explanati… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-054 | api-error-upload-type | errors-empty | brand-voice | P1 | 5 variants of 'Invalid file type' with slightly different allowed lists ('JPEG,… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-003 | api-error-delivery-already-active | errors-empty | legal | P1 | Snake_case slug 'no_active_delivery' is surfaced to the user as an error string… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-004 | api-error-generic-failed | errors-empty | legal | P1 | 100+ variants of generic 'Failed to X' errors surface to all user roles without… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-007 | api-error-auth-invalid-credentials | errors-empty | legal | P1 | Login error 'Invalid credentials' is correctly ambiguous and does not disclose… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-010 | vp-auth-login-access-denied | errors-empty | legal | P1 | Vendor-portal access-denied error 'This portal is only for vendor accounts. Ple… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-011 | ap-auth-error-access-denied | errors-empty | legal | P1 | Admin-portal access-denied error 'Access denied. Only administrators can sign i… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-018 | api-error-auth-2fa | errors-empty | legal | P1 | All 2FA errors are vague 'Failed to enable 2FA', 'Failed to start 2FA challenge… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-023 | dp-err-offline-banner | errors-empty | legal | P1 | Driver offline banner 'You're offline. Updates will sync when connected.' is a… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-026 | web-err-checkout-cancel | errors-empty | legal | P1 | Razorpay dismissal toast says only 'Payment cancelled'. Customer is left withou… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-028 | api-error-order-already-paid | errors-empty | legal | P1 | 'Order already paid' is correct but offers no remediation if customer believes… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-031 | api-error-validation-payout-method | errors-empty | legal | P1 | Payout-method validation surfaces 'payoutMethod must be "bank_transfer" or "upi… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-033 | vp-onb-docs-upload-failed | errors-empty | legal | P1 | FSSAI / KYC document-upload failure says only 'Upload failed' with no retry gui… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-036 | vp-onb-banner-info-requested-body | errors-empty | legal | P1 | 'The admin team needs additional information before approving your application.… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-041 | vp-onb-review-policies-pending | errors-empty | legal | P1 | 'Policies not accepted / Please go back to Step 5 and accept all required polic… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-042 | vp-onb-policies-required-toast | errors-empty | legal | P1 | 'Please accept all required policies' — same as above without surface-specific… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-043 | mv-onb-policies-policy-required-alert | errors-empty | legal | P1 | 'Please select a cancellation policy.' — chef chooses their own cancellation po… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-049 | ap-providerdetail-dialog-bodies | errors-empty | legal | P1 | Three branched confirmation messages on delivery-provider delete/disable/enable… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-050 | ap-providers-dialog-delete-body | errors-empty | legal | P1 | 'Are you sure you want to delete this delivery provider? This action cannot be… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-051 | ap-staffdetail-confirm-deactivate | errors-empty | legal | P1 | 'Are you sure you want to deactivate this staff member? They will lose access t… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-055 | ap-analytics-coming-soon | errors-empty | legal | P1 | 'Chart coming soon' is a placeholder shown to admin users in production. Surfac… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-056 | web-err-error-boundary | errors-empty | legal | P1 | Top-level error fallback 'Unexpected error / Something broke on our end. / We'v… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-057 | vp-err-boundary-fallback | errors-empty | legal | P1 | Vendor error boundary 'Something went wrong / An unexpected error occurred whil… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-059 | ap-errorboundary-body | errors-empty | legal | P1 | Admin error boundary same generic pattern. Admin actions are evidentiary — an e… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-061 | api-error-chat-availability | errors-empty | legal | P1 | Chat-availability errors include: 'Chat is not available for completed/cancelle… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-062 | api-error-quotes-catering | errors-empty | legal | P1 | Catering-quote errors include 'This quote has expired', 'Quote deadline has pas… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-071 | web-err-cart-minimum | errors-empty | legal | P1 | 'Minimum order not met / Add {amount} more to proceed' — minimum-order rule is… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-085 | api-error-delivery-online | errors-empty | legal | P1 | Driver-state errors mix 'You must be online to accept deliveries' (driver-facin… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-087 | ap-exports-error | errors-empty | legal | P1 | Admin data-export error 'Download failed' is minimal. Data exports include PII… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-006 | web-err-login-generic | errors-empty | technical-writer | P1 | 'Something went wrong. Please try again.' on the customer login surface is conv… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-007 | dp-err-generic-auth | errors-empty | technical-writer | P1 | Driver login generic error 'Something went wrong. Please try again.' is too ver… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-008 | dp-err-access-denied | errors-empty | technical-writer | P1 | 'Access denied. Please check your credentials and try again.' on driver login m… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-009 | ap-auth-error-access-denied | errors-empty | technical-writer | P1 | Admin login surface shows 'Access denied. Only administrators can sign in to th… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-010 | web-err-error-boundary | errors-empty | technical-writer | P1 | Top-level error boundary copy is 35 words and uses three sentences ('Something… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-011 | web-ux-cart-empty | errors-empty | technical-writer | P1 | Empty cart copy is 28 words across two sentences and uses chatty filler ('Looks… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-012 | web-ux-favorites-loggedout | errors-empty | technical-writer | P1 | 'Log in to see your favorites' uses banned 'Log in' verb (style guide §3: alway… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-013 | mc-checkout-errors | errors-empty | technical-writer | P1 | Payment timeout error on customer mobile checkout is 14 words and verbose for a… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-014 | api-error-auth-suspended | errors-empty | technical-writer | P1 | 'Account is suspended' gives no reason and no contact path. For a sign-in block… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-015 | api-error-delivery-already-active | errors-empty | technical-writer | P1 | Error payload includes a raw snake_case slug ('no_active_delivery') alongside a… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-016 | api-error-delivery-online | errors-empty | technical-writer | P1 | Driver-facing errors use 'Partner' to refer to the driver ('Partner has reached… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-017 | api-error-auth-unauthorized | errors-empty | technical-writer | P1 | Four variants of the same auth-failure concept: 'Unauthorized' / 'unauthorized'… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-018 | web-err-checkout-no-addresses | errors-empty | technical-writer | P1 | Empty-state copy on a money-bearing surface is wordy and uses chatty 'don't hav… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-025 | web-err-browse-empty | errors-empty | business-analyst | P2 | Empty chef search results offers "Clear filters" but no fallback to browse with… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-026 | mc-home-empty | errors-empty | business-analyst | P2 | Mobile home empty state "No chefs found / Try adjusting your filters" — same pa… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-027 | web-ux-favorites-loggedout | errors-empty | business-analyst | P2 | Logged-out favorites state uses "Log in" — banned term, should be "Sign in" | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-028 | web-err-checkout-no-addresses | errors-empty | business-analyst | P2 | No saved addresses empty state during checkout is a conversion blocker with a p… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-029 | web-ux-chefdetail-not-found | errors-empty | business-analyst | P2 | Chef not found state has "Browse Chefs" CTA which is good — but copy "Chef not… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-030 | mc-catering-empty | errors-empty | business-analyst | P2 | Catering requests empty state uses 🍽️ emoji in functional copy — violates style… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-031 | mc-social-empty | errors-empty | business-analyst | P2 | Social feed empty state uses 📸 emoji in body copy — violates style guide | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-032 | vp-ux-reviews-empty | errors-empty | business-analyst | P2 | Empty reviews state "No reviews yet" — no guidance on how to get the first revi… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-033 | mv-reviews-empty | errors-empty | business-analyst | P2 | Mobile vendor reviews empty state "No reviews yet. Your first review will appea… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-034 | vp-ux-orders-history-empty | errors-empty | business-analyst | P2 | Order history empty state for date range filter gives no indication of what ran… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-035 | vp-ux-earnings-error | errors-empty | business-analyst | P2 | Earnings load failure offers no support path — a chef who cannot see their earn… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-036 | mv-earnings-no-payout-account | errors-empty | business-analyst | P2 | Mobile vendor "No payout account configured" — no CTA to configure it, dead end… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-037 | vp-ux-notifs-empty | errors-empty | business-analyst | P2 | Admin notifications empty state is verbose and unexplained — chef does not know… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-038 | dp-empty-active-body | errors-empty | business-analyst | P2 | Active delivery empty state directs driver to "check available deliveries" — go… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-039 | md-err-002 | errors-empty | business-analyst | P2 | Mobile delivery available-deliveries empty state tells driver to "Pull to refre… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-040 | md-emp-007 | errors-empty | business-analyst | P2 | Permission denied for fleet management gives admin contact instruction but no a… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-041 | api-error-payment-refund | errors-empty | business-analyst | P2 | Refund error messages expose constraint details with no refund policy or timeli… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-042 | api-error-generic-failed | errors-empty | business-analyst | P2 | ~100 "Failed to X" API error variants are passed directly to UI with no user-fr… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-043 | api-error-auth-not-verified | errors-empty | business-analyst | P2 | Two variants of account-not-verified error exist: "Account not verified" and "A… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-044 | web-err-login-generic | errors-empty | business-analyst | P2 | Generic login error "Something went wrong. Please try again." has no diagnostic… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-045 | dp-err-generic-auth | errors-empty | business-analyst | P2 | Delivery portal generic auth error same pattern as web — no driver-specific rec… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-046 | vp-onb-docs-upload-failed | errors-empty | business-analyst | P2 | Document upload failure "Upload failed" — no guidance on file format, size, or… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-047 | mv-onb-docs-upload-fail | errors-empty | business-analyst | P2 | Mobile vendor document upload failure same four-word message — same activation… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-048 | api-error-upload-size | errors-empty | business-analyst | P2 | Inconsistent file size formatting across upload errors: "10MB" vs "5 MB" vs "5M… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-049 | api-error-upload-type | errors-empty | business-analyst | P2 | Five variants of file-type rejection with slightly different allowed lists — us… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-050 | vp-onb-review-missing-docs | errors-empty | business-analyst | P2 | Missing documents warning "Please go back to Step 4" — step references become i… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-051 | vp-onb-review-policies-pending | errors-empty | business-analyst | P2 | Unaccepted policies warning uses step number reference — same fragility as BA-E… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-052 | api-error-auth-reset-expired | errors-empty | business-analyst | P2 | Three near-duplicate reset token error variants; none include a "resend reset l… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-053 | api-error-auth-2fa | errors-empty | business-analyst | P2 | 2FA errors are vague with no guidance — admin who loses 2FA access has no recov… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-054 | dp-err-step4-plans-load | errors-empty | business-analyst | P2 | Subscription plans load failure during driver onboarding "Failed to load subscr… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-055 | vp-ux-dashboard-pending-empty | errors-empty | business-analyst | P2 | "All caught up! / No pending orders right now" — exclamation mark in non-celebr… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-007 | web-err-browse-empty | errors-empty | brand-voice | P2 | 'No chefs found' empty state drifts: web 'No chefs found / Try adjusting your f… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-008 | web-err-orderdetail-not-found | errors-empty | brand-voice | P2 | 'Order not found' uses inconsistent CTA capitalization: web 'View All Orders' (… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-012 | dp-empty-active-body | errors-empty | brand-voice | P2 | Driver app body copy is too conversational for driver persona-matrix telegraphi… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-014 | mc-profile-save-error | errors-empty | brand-voice | P2 | Mobile-customer wraps every Alert.alert error with a generic title 'Error' — th… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-015 | mv-onb-docs-required-title | errors-empty | brand-voice | P2 | Mobile-vendor uses Title Case alert titles ('Documents Required', 'Permission R… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-016 | web-ux-cart-empty | errors-empty | brand-voice | P2 | 'Looks like you haven't added any items yet' — combines 'looks like' (banned cu… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-017 | web-err-orders-empty | errors-empty | brand-voice | P2 | 'You haven't placed any orders yet.' — blame phrasing. STYLE-GUIDE allows 'No o… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-018 | vp-ux-orders-history-empty | errors-empty | brand-voice | P2 | 'You have no completed or cancelled orders yet.' — blame-tinged 'you have no' p… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-026 | vp-ux-payouts-empty | errors-empty | brand-voice | P2 | 'Your payout history will appear here once you receive your first payout.' — 19… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-027 | vp-ux-notifs-empty | errors-empty | brand-voice | P2 | 'Admin requests for document uploads, profile updates, and review feedback will… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-029 | md-emp-007 | errors-empty | brand-voice | P2 | Driver lock-screen body 'Fleet management is available for fleet managers only.… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-030 | ap-providerdetail-dialog-bodies | errors-empty | brand-voice | P2 | 'Are you sure you want to delete this delivery provider? This action cannot be… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-034 | api-error-delivery-online | errors-empty | brand-voice | P2 | 'Partner has reached maximum concurrent deliveries' / 'Partner is not verified… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-046 | api-error-auth-suspended | errors-empty | brand-voice | P2 | 'Account is suspended' — no reason, no contact-support guidance. Inventory note… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-048 | api-error-chef-not-accepting | errors-empty | brand-voice | P2 | 'Chef is not accepting orders' — no alternative ('try another chef'); customer… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-050 | api-error-auth-not-verified | errors-empty | brand-voice | P2 | 'Account not verified' vs 'Account not verified yet' — two variants in same fil… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-051 | api-error-auth-current-pw | errors-empty | brand-voice | P2 | 'Current password is incorrect' vs 'Password is incorrect' — two variants of sa… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-052 | api-error-auth-reset-expired | errors-empty | brand-voice | P2 | 'Reset token has expired' / 'Invalid or expired reset token' / 'Invalid or expi… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-055 | api-error-upload-count | errors-empty | brand-voice | P2 | Upload-count errors inconsistent on remediation: 'Maximum 5 kitchen photos allo… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-061 | vp-onb-locked-conflict-toast | errors-empty | brand-voice | P2 | Two variants of 'kitchen already submitted' state: 'Your kitchen is already und… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-067 | web-err-browse-load-failed | errors-empty | brand-voice | P2 | 'Failed to load chefs. Please try again.' (web) vs 'Failed to load chef details… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-068 | mc-checkout-errors | errors-empty | brand-voice | P2 | 'Order creation failed. Please try again.' vs API-side 'Failed to create order'… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-069 | vp-ux-payouts-empty | errors-empty | brand-voice | P2 | 'Unable to load X / Please try again later.' pattern (vendor-portal earnings/pa… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-005 | api-error-validation-required | errors-empty | legal | P2 | Field-name casing leaks internal API contract to users ('chefId is required', '… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-006 | api-error-validation-invalid-id | errors-empty | legal | P2 | ~40 variants of 'Invalid X' errors include internal entity types ('Invalid payl… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-012 | dp-err-access-denied | errors-empty | legal | P2 | Driver-portal 'Access denied. Please check your credentials and try again.' — p… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-014 | api-error-auth-not-verified | errors-empty | legal | P2 | Two variants 'Account not verified' and 'Account not verified yet' on the same… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-015 | web-err-login-session-expired | errors-empty | legal | P2 | Session-expired banner 'Your session has expired. Please sign in again.' is cor… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-016 | api-error-auth-reset-expired | errors-empty | legal | P2 | Three near-duplicate variants 'Reset token has expired', 'Invalid or expired re… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-017 | api-error-auth-current-pw | errors-empty | legal | P2 | Two variants 'Current password is incorrect' and 'Password is incorrect' on the… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-019 | web-err-auth-service | errors-empty | legal | P2 | Fallback errors 'Invalid email or password', 'Registration failed', 'Token refr… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-020 | dp-err-auth-service-refresh | errors-empty | legal | P2 | Driver-portal surfaces 'Token refresh failed' as user-visible text. Drivers are… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-021 | web-err-layout-offline | errors-empty | legal | P2 | Offline banner 'You're offline. Some features may be unavailable.' does not cla… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-022 | vp-ux-layout-offline | errors-empty | legal | P2 | Vendor offline banner 'You're offline. Orders will sync when connected.' makes… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-024 | ap-layout-offline-banner | errors-empty | legal | P2 | Admin offline banner 'You're offline. Data may not be up to date.' is fine for… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-030 | api-error-chef-not-accepting | errors-empty | legal | P2 | 'Chef is not accepting orders' offers no alternative or reason. Borderline unfa… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-037 | vp-onb-locked-conflict-toast | errors-empty | legal | P2 | 'Your kitchen is already under review or live. Returning to the dashboard.' off… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-046 | api-error-upload-type | errors-empty | legal | P2 | 5 variants of 'Invalid file type. Allowed: …' with different allowed lists (JPE… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-047 | api-error-upload-size | errors-empty | legal | P2 | Inconsistent size formatting: '10MB' vs '5 MB' vs '5MB' vs '5 MB'. Mobile-vendo… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-052 | api-error-staff-permissions | errors-empty | legal | P2 | Staff-permission errors expose internal role hierarchy: 'Only super admins can… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-053 | ap-users-toast-fail-suspend | errors-empty | legal | P2 | 'Failed to suspend user' / 'Failed to activate user' — admin destructive action… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-054 | ap-auditlogs-empty | errors-empty | legal | P2 | Audit-log empty state 'No audit events match these filters.' is fine. Verify th… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-060 | web-err-generic-retry | errors-empty | legal | P2 | 'Something went wrong. Please try again.' duplicated across 5+ surfaces (HomePa… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-063 | ap-platsettings-zones-empty | errors-empty | legal | P2 | Platform-settings empty state 'No zones yet — delivery is available everywhere… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-068 | ap-staffdetail-perms-empty | errors-empty | legal | P2 | 'No specific permissions data available for this role.' — admin sees an empty p… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-072 | mc-onb-step1-errors | errors-empty | legal | P2 | Mobile-customer onboarding validation 'Enter a valid 10-digit Indian mobile num… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-073 | mv-onb-personal-phone-err | errors-empty | legal | P2 | Same India-specific mobile pattern enforced on chef onboarding. Chef registrati… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-074 | mc-catering-errors | errors-empty | legal | P2 | Catering form validation '… Event date must be in the future / Guest count must… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-076 | vp-ux-menu-form-validation | errors-empty | legal | P2 | Menu-item form validation includes 'Select at least one dietary tag' — dietary-… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-083 | api-error-auth-unauthorized | errors-empty | legal | P2 | Four variants 'Unauthorized', 'unauthorized', 'Authentication required', 'Token… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-084 | api-error-chef-onboarding | errors-empty | legal | P2 | Chef-onboarding pre-submit checks include 'Your application has already been ap… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-088 | api-error-invitation | errors-empty | legal | P2 | 'Invitation has expired' / 'Invalid or expired invitation' / 'Invalid invitatio… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-019 | pattern-error-missing-what-to-do | errors-empty | technical-writer | P2 | Pattern: ~40 error strings give only the 'what happened' half of the error form… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-020 | pattern-error-failed-to-x | errors-empty | technical-writer | P2 | API surfaces ~100 variants of 'Failed to X' (accept quote, add comment, add fav… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-021 | pattern-error-oops-uhoh | errors-empty | technical-writer | P2 | Style-guide §4 explicitly bans 'Oops!' / 'Uh oh!' patterns. Repo currently show… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-022 | pattern-empty-no-action | errors-empty | technical-writer | P2 | Pattern: ~25 empty states give 'why it's empty' but not the one action (formula… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-023 | pattern-empty-state-chatty | errors-empty | technical-writer | P2 | Pattern: several empty states use the chatty 'Looks like...' / 'It's lonely...'… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-024 | pattern-button-too-long | errors-empty | technical-writer | P2 | Pattern: buttons inside error/empty surfaces exceed the 3-word, sentence-case r… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-025 | pattern-please-padding | errors-empty | technical-writer | P2 | Pattern: ~30+ validation errors begin with 'Please' as filler. 'Please' adds no… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-026 | pattern-toast-non-past-tense | errors-empty | technical-writer | P2 | Style guide §4 toast formula: past tense, ≤6 words, period. Several error/empty… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-027 | pattern-period-inconsistency | errors-empty | technical-writer | P2 | Pattern: errors and empty-state strings inconsistently use trailing periods wit… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-028 | pattern-emoji-in-empty-states | errors-empty | technical-writer | P2 | Mobile-customer empty states embed emoji ('🍽️', '📸', heart) in copy. Style guid… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-029 | pattern-exclamation-overuse | errors-empty | technical-writer | P2 | Style guide §1 Rule 1: '≤1 exclamation per page'. Multiple empty/error strings… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-030 | pattern-title-case-leakage | errors-empty | technical-writer | P2 | Pattern: Title Case bleeds into error/empty headings in vendor-portal and admin… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-031 | pattern-validation-shouty-cap | errors-empty | technical-writer | P2 | Mobile-vendor uses 'Validation Error' / 'Submission Error' as alert titles. The… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-032 | pattern-zod-bare-message | errors-empty | technical-writer | P2 | Zod validation strings come through as bare field rules without form context: '… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-033 | pattern-snake-case-leak | errors-empty | technical-writer | P2 | Snake-case slugs reach user-facing copy in at least one place (already flagged… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-034 | pattern-payout-method-jargon | errors-empty | technical-writer | P2 | Payout-validation strings contain backend jargon: 'payoutMethod must be...', 'b… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-035 | pattern-not-verified-drift | errors-empty | technical-writer | P2 | Two near-identical 'Account not verified' strings in the same handler file — in… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-036 | pattern-password-incorrect-drift | errors-empty | technical-writer | P2 | Two strings for the same concept: 'Current password is incorrect' / 'Password i… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-037 | pattern-reset-token-drift | errors-empty | technical-writer | P2 | Three near-duplicate variants of password-reset / enrollment token errors. | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-038 | pattern-2fa-vague | errors-empty | technical-writer | P2 | Four 2FA errors are all variants of 'Failed to ...' with zero remediation. 2FA… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-039 | pattern-uploaded-file-allowed-list-drift | errors-empty | technical-writer | P2 | Five variants of 'Invalid file type. Allowed: ...' with slightly different allo… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-040 | pattern-image-count-friendly-drift | errors-empty | technical-writer | P2 | Image-count limit errors show four variants — only one includes 'remove one bef… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-041 | pattern-cart-empty-drift | errors-empty | technical-writer | P2 | Same 'cart empty' state is expressed three different ways across surfaces — dri… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-042 | pattern-no-X-found-drift | errors-empty | technical-writer | P2 | Same 'no X found' phrasing appears with subtle variations across admin / web /… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-043 | pattern-not-found-bare | errors-empty | technical-writer | P2 | Detail pages for missing resources show bare 'X not found' with no recovery act… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-044 | pattern-status-transition-vague | errors-empty | technical-writer | P2 | 'Invalid status transition' / 'Invalid status' / 'Delivery is in a terminal sta… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-045 | pattern-vendor-customer-noun | errors-empty | technical-writer | P2 | Style guide §3: vendor-portal and admin should use 'chef' (their self-identity)… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-046 | pattern-go-back-button-form | errors-empty | technical-writer | P2 | Recovery buttons on not-found / error pages use Title Case and inconsistent ver… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-047 | pattern-offline-banner-drift | errors-empty | technical-writer | P2 | Three offline banners across the codebase, three different wordings. | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-048 | pattern-confirm-this-action-cannot | errors-empty | technical-writer | P2 | Style guide §4 explicitly cites 'Are you sure? This action cannot be undone.' a… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-049 | pattern-staff-confirm-soften | errors-empty | technical-writer | P2 | Staff deactivate/reactivate confirmations use 'Are you sure you want to...' ope… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-050 | pattern-logout-confirm | errors-empty | technical-writer | P2 | Mobile-vendor logout confirm uses 'Are you sure you want to log out?' which (a)… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-051 | pattern-permissions-required | errors-empty | technical-writer | P2 | Locked-feature explanations on mobile-delivery use a passive, slightly defensiv… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-052 | pattern-rejected-banner-soften | errors-empty | technical-writer | P2 | Vendor-portal onboarding rejection banner reads 'Application Rejected' / 'Your… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-053 | pattern-more-info-needed-banner | errors-empty | technical-writer | P2 | Companion banner: 'More Information Needed' (Title Case) + 'The admin team need… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-054 | pattern-failed-to-load-vague-mobile-delivery | errors-empty | technical-writer | P2 | Mobile-delivery has nine 'Failed to load X' strings (md-err-001..009) that all… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-055 | pattern-bicycle-carrier | errors-empty | technical-writer | P2 | 'Please indicate if your bicycle can carry a delivery box' (9 words, 'Please' f… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-056 | pattern-stripe-onboarding-jargon | errors-empty | technical-writer | P2 | Chef-facing Stripe errors mix vendor jargon and dev hints. Several variants des… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-057 | pattern-search-min-chars | errors-empty | technical-writer | P2 | 'Search query must be at least 2 characters' is fine voice but uses developer n… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-058 | pattern-favorites-limit-numeric | errors-empty | technical-writer | P2 | 'You can save up to 7 favorite chefs. Remove one first.' uses an arbitrary nume… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-059 | pattern-coming-soon-leak | errors-empty | technical-writer | P2 | 'Chart coming soon' on admin analytics — placeholder copy that exposes unfinish… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-060 | pattern-locked-toast-onboarding | errors-empty | technical-writer | P2 | Two onboarding-locked toasts in vp-onb use slightly different framings for the… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-061 | pattern-empty-no-description | errors-empty | technical-writer | P2 | Various 'No description provided' / 'No documents uploaded' / 'No specialties a… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-062 | pattern-saved-addresses-yet | errors-empty | technical-writer | P2 | Customer 'No addresses' empty state is duplicated across web checkout and mobil… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-063 | pattern-pending-orders-drift | errors-empty | technical-writer | P2 | 'No pending orders' appears with three different bodies across vendor-portal an… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-064 | pattern-rate-limit-friendly | errors-empty | technical-writer | P2 | Chat rate-limit message 'Please wait a moment before sending another message' i… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-065 | pattern-chat-availability-wordy | errors-empty | technical-writer | P2 | Chat-closed errors carry redundant phrasing across three near-identical variant… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-066 | pattern-otto-retry | errors-empty | technical-writer | P2 | 'Try again' as a standalone button in OttoChat retry. Per style guide formula,… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-067 | pattern-validation-error-counts | errors-empty | technical-writer | P2 | Mobile-customer catering form (mc-catering-errors) bundles 9 separate Zod messa… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-068 | pattern-zod-india-format | errors-empty | technical-writer | P2 | Several validation strings hard-code India-specific assumptions in the message:… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-069 | pattern-mobile-customer-profile-error-alert | errors-empty | technical-writer | P2 | Mobile-customer profile errors use both a title ('Error') and a body — title-le… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-070 | pattern-mobile-vendor-permission-alerts | errors-empty | technical-writer | P2 | Permission alerts on mobile-vendor use Title Case headings + verbose bodies: 'P… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-071 | pattern-tos-error | errors-empty | technical-writer | P2 | 'You must accept the terms and conditions' on driver onboarding has no link to… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-072 | pattern-payments-policy-missing | errors-empty | technical-writer | P2 | Order-already-paid / payment-not-captured / refund-amount errors don't communic… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-073 | pattern-error-boundaries-drift | errors-empty | technical-writer | P2 | Four different error-boundary strings across apps. Same global-failure concept,… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-074 | pattern-driver-history-empty-drift | errors-empty | technical-writer | P2 | Driver-portal and mobile-delivery have the same 'no deliveries yet' empty state… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-075 | pattern-staff-empty-drift | errors-empty | technical-writer | P2 | Admin-portal staff empty state has a clean message that includes the action; de… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-076 | pattern-fleet-empty-drift | errors-empty | technical-writer | P2 | Two 'no partners' empty states, two wordings; both lack a recovery action. | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-077 | pattern-empty-list-period | errors-empty | technical-writer | P2 | Inconsistency: admin-portal empty states mostly omit periods on single-line lab… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-078 | pattern-validation-error-suspend | errors-empty | technical-writer | P2 | Admin user-action failure toasts are bare 'Failed to suspend user' / 'Failed to… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-079 | pattern-validation-pin-vs-pincode | errors-empty | technical-writer | P2 | Inconsistent terminology: 'pincode', 'PIN code', 'PIN'. Customer-facing should… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-080 | pattern-app-name-fe3dr | errors-empty | technical-writer | P2 | Vendor-portal access-denied error mentions 'Fe3dr customer app' — likely a plac… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-081 | pattern-validation-percentage-vs-percent | errors-empty | technical-writer | P2 | 'Percentage discount must be between 0 and 100' uses 'Percentage' as noun in an… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-082 | pattern-validation-interval | errors-empty | technical-writer | P2 | 'Invalid interval. Must be monthly, quarterly, or yearly' on subscription form… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-083 | pattern-validation-min-2-chars | errors-empty | technical-writer | P2 | 'Name must be at least 2 characters' min-length applied to chef-onboarding name… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-084 | pattern-mobile-vendor-photo-upload-fail | errors-empty | technical-writer | P2 | Three near-identical photo-upload-fail toasts in mv-profile/menuedit, each with… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-085 | pattern-mobile-customer-favorites-error | errors-empty | technical-writer | P2 | 'Could not remove from favorites. Please try again.' is OK but missing a soft '… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-086 | pattern-checkout-payment-cancelled | errors-empty | technical-writer | P2 | 'Payment cancelled' (2 words) appears as a toast — but doesn't follow toast for… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-087 | pattern-chef-not-accepting | errors-empty | technical-writer | P2 | 'Chef is not accepting orders' has no alternative action for the customer mid-f… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-088 | pattern-quote-already-submitted | errors-empty | technical-writer | P2 | 'This request is no longer accepting quotes' / 'You have already submitted a qu… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-089 | pattern-promo-not-found | errors-empty | technical-writer | P2 | Promo errors include 'Promo code already exists' / 'Promo code not found' / 'Pr… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-090 | pattern-review-only-delivered | errors-empty | technical-writer | P2 | 'Can only review delivered orders' / 'This order has already been reviewed' — f… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-091 | pattern-invitation-token-drift | errors-empty | technical-writer | P2 | Four invitation-token error variants very close in meaning. | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-092 | pattern-staff-permissions-tone | errors-empty | technical-writer | P2 | Staff permission errors are accurate but blunt: 'You don't have permission to i… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-056 | ap-providerdetail-dialog-bodies | errors-empty | business-analyst | P3 | Confirmation dialogs use "Are you sure you want to..." phrasing — vague and vio… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-057 | ap-staffdetail-confirm-deactivate | errors-empty | business-analyst | P3 | Staff deactivation confirmation uses "Are you sure" pattern — same pattern viol… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-058 | ap-providers-dialog-delete-body | errors-empty | business-analyst | P3 | Delivery provider delete dialog uses "Are you sure" and "This action cannot be… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-059 | api-error-validation-required | errors-empty | business-analyst | P3 | 18 variants of "X is required" with inconsistent field name casing (e.g., "chef… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-060 | api-error-auth-unauthorized | errors-empty | business-analyst | P3 | Four variants of unauthorized error exist including mixed case "Unauthorized" a… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-061 | ap-secsettings-apikey-empty | errors-empty | business-analyst | P3 | "No keys yet." — full stop with trailing period is inconsistent with most other… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-062 | vp-onb-review-empty-cuisines | errors-empty | business-analyst | P3 | "None selected" for cuisines in onboarding review — does not tell the chef if c… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-063 | vp-onb-review-empty-days | errors-empty | business-analyst | P3 | "No days set" for operating hours in onboarding review — same ambiguity as BA-E… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-064 | api-error-delivery-online | errors-empty | business-analyst | P3 | "Partner has reached maximum concurrent deliveries" — uses "Partner" terminolog… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-065 | api-error-delivery-already-active | errors-empty | business-analyst | P3 | Snake_case slug "no_active_delivery" exposed as error message variant alongside… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-066 | mv-more-logout-confirm-body | errors-empty | business-analyst | P3 | Logout confirmation title is "Logout" (single word) — should be "Sign out" per… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-067 | mv-onb-personal-validation-alert | errors-empty | business-analyst | P3 | "Validation Error" as alert title is a technical term, not a user-oriented label | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-068 | mv-onb-review-submit-error-title | errors-empty | business-analyst | P3 | "Submission Error" alert title — same technical phrasing as BA-ERR-067 | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-069 | ap-layout-offline-banner | errors-empty | business-analyst | P3 | Admin offline banner "Data may not be up to date" — vague about which operation… | [findings/errors-empty.md](findings/errors-empty.md) |
| BA-ERR-070 | vp-ux-layout-offline | errors-empty | business-analyst | P3 | Vendor offline banner is stronger than others: "Orders will sync when connected… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-002 | web-err-login-session-expired | errors-empty | brand-voice | P3 | 'Your session has expired. Please sign in again.' appears identically in web, v… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-019 | web-err-checkout-no-addresses | errors-empty | brand-voice | P3 | 'You don't have any saved addresses yet.' — soft blame; STYLE-GUIDE prefers sta… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-022 | mc-social-empty | errors-empty | brand-voice | P3 | 'Chefs will share their latest creations here.' — 'creations' is on the edge of… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-023 | mv-orders-empty-body | errors-empty | brand-voice | P3 | 'New orders will appear here automatically' — 'automatically' is filler; chef p… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-024 | ap-platsettings-zones-empty | errors-empty | brand-voice | P3 | 'No zones yet — delivery is available everywhere until a zone is created.' — em… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-025 | mc-favorites-empty | errors-empty | brand-voice | P3 | Heart emoji in empty-state copy on customer surface — lens brief says customer… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-028 | ap-staffdetail-perms-empty | errors-empty | brand-voice | P3 | 'No specific permissions data available for this role.' — admin persona allows… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-031 | ap-providerdetail-dialog-bodies | errors-empty | brand-voice | P3 | 'Are you sure you want to disable this provider?' / 'Are you sure you want to e… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-035 | vp-ux-menu-error-state | errors-empty | brand-voice | P3 | 'Try Again' button label uses Title Case across vendor-portal (menu, dashboard,… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-036 | web-ux-chefdetail-not-found | errors-empty | brand-voice | P3 | 'Browse Chefs' CTA in Title Case on web 'Chef not found' empty state; same in '… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-037 | vp-onb-banner-rejected-title | errors-empty | brand-voice | P3 | Onboarding banner titles in Title Case: 'Application Rejected' / 'More Informat… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-038 | mc-cartsheet-empty | errors-empty | brand-voice | P3 | Period inconsistency: 'Your cart is empty' (no period, CartSheet) vs 'Your cart… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-039 | dp-empty-partners | errors-empty | brand-voice | P3 | Trailing-period drift in delivery-portal: 'No partners found matching your crit… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-040 | api-error-phone-duplicate | errors-empty | brand-voice | P3 | API errors are wildly inconsistent on trailing periods: 'This phone number is a… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-042 | ap-dashboard-recent-empty-body | errors-empty | brand-voice | P3 | 'Platform events will appear here.' — generic SaaS-dashboard placeholder (lens… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-043 | ap-secsettings-apikey-empty | errors-empty | brand-voice | P3 | 'No keys yet.' — under-informative; admin tone allows precision but 'No keys ye… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-056 | web-err-favorite-max-limit | errors-empty | brand-voice | P3 | 'You can save up to 7 favorite chefs. Remove one first.' — good remediation pat… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-057 | vp-onb-docs-required-title | errors-empty | brand-voice | P3 | Mobile-vendor 'Documents Required' / 'Permission Required' / 'Terms Required' t… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-058 | vp-ux-menu-empty | errors-empty | brand-voice | P3 | 'Start building your menu by adding your first dish.' — 9 words but starts with… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-059 | mv-reviews-empty | errors-empty | brand-voice | P3 | 'No reviews yet. Your first review will appear here.' — chef-facing, 'your firs… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-060 | dp-empty-staff | errors-empty | brand-voice | P3 | 'No staff members yet. Send an invitation to get started.' — admin surface in d… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-065 | api-error-quotes-catering | errors-empty | brand-voice | P3 | Catering quote errors generally well-phrased ('This request is no longer accept… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-066 | web-err-chefdetail-empty-category | errors-empty | brand-voice | P3 | 'No items in this category' — identical between web and mobile-customer (positi… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-070 | vp-onb-save-failure-toast | errors-empty | brand-voice | P3 | 'Failed to save your details. Please try again.' (VP onboarding) vs 'Failed to… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-071 | vp-onb-banner-rejected-body | errors-empty | brand-voice | P3 | 'Your previous application was not approved.' — passive voice. STYLE-GUIDE §5 '… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-072 | vp-ux-menu-form-image-errors | errors-empty | brand-voice | P3 | '{name}: Invalid type. Allowed: JPEG, PNG, WebP.' format prepends filename with… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-073 | dp-err-step3-file-size | errors-empty | brand-voice | P3 | 'File too large. Max {n}MB for {label}' — no space in '{n}MB', conflicts with B… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-074 | api-error-search-min | errors-empty | brand-voice | P3 | 'Search query must be at least 2 characters' — 'query' is dev jargon for custom… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-075 | api-error-validation-numeric | errors-empty | brand-voice | P3 | 'overallRating must be between 1 and 5' — camelCase field name in user-visible… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-076 | api-error-chef-kitchen-name | errors-empty | brand-voice | P3 | 'A kitchen with this name already exists. Please choose a different name.' — fr… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-077 | api-error-chat-availability | errors-empty | brand-voice | P3 | 'Please wait a moment before sending another message' — friendly rate-limit; po… | [findings/errors-empty.md](findings/errors-empty.md) |
| BV-078 | api-error-favorites | errors-empty | brand-voice | P3 | 'Chef is already in your favorites' — clean, action-implied. Positive baseline | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-009 | api-error-favorites | errors-empty | legal | P3 | 'Chef is already in your favorites' discloses authenticated-user state — accept… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-034 | mc-chef-empty-category | errors-empty | legal | P3 | Empty menu-category state 'No items in this category' is innocuous but flag as… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-044 | dp-err-step1-referral-invalid | errors-empty | legal | P3 | 'Invalid referral code' — minor; ensure that submitting an invalid code does NO… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-045 | dp-err-step3-file-size | errors-empty | legal | P3 | 'File too large. Max {n}MB for {label}' — fine but ensure consistent size repre… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-048 | api-error-upload-count | errors-empty | legal | P3 | 'Maximum 3 images per review' / 'Maximum 4 images per post' / 'Maximum 5 images… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-064 | dp-empty-active-body | errors-empty | legal | P3 | 'Check available deliveries to pick up a new order.' — neutral. Verify the driv… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-065 | md-emp-007 | errors-empty | legal | P3 | 'Fleet management is available for fleet managers only. Contact your administra… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-066 | md-emp-011 | errors-empty | legal | P3 | 'Staff management requires manager permissions. Contact your administrator to r… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-067 | web-err-favorite-max-limit | errors-empty | legal | P3 | 'You can save up to 7 favorite chefs. Remove one first.' — duplicated across Br… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-069 | ap-secsettings-apikey-empty | errors-empty | legal | P3 | 'No keys yet.' for API-key management. Verify that an admin creating their firs… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-070 | web-ux-cart-empty | errors-empty | legal | P3 | Empty cart copy 'Your cart is empty / Looks like you haven't added any items ye… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-075 | mc-favorites-empty | errors-empty | legal | P3 | 'No saved chefs yet / Tap the heart on any chef to save them!' — innocuous but… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-077 | mv-menunew-price-range | errors-empty | legal | P3 | 'Price must be between ₹1 and ₹10,000' — flat range no tax-context. Confirm whe… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-078 | mv-onb-docs-required-title | errors-empty | legal | P3 | 'Documents Required' as an alert title — combined with 'Please upload both ID p… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-079 | mv-more-logout-confirm-body | errors-empty | legal | P3 | 'Are you sure you want to log out?' — uses 'log out' (style-guide bans this, re… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-080 | mv-menuitem-delete-alert | errors-empty | legal | P3 | 'Are you sure you want to delete this menu item?' — does not state whether dele… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-081 | api-error-validation-numeric | errors-empty | legal | P3 | Numeric-bounds errors 'overallRating must be between 1 and 5', 'Percentage disc… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-082 | api-error-validation-not-found | errors-empty | legal | P3 | ~42 'X not found' variants. Mostly safe but flag 'Verification token not found'… | [findings/errors-empty.md](findings/errors-empty.md) |
| LEG-ERR-086 | api-error-order-review-rules | errors-empty | legal | P3 | 'Can only review delivered orders' / 'This order has already been reviewed' — f… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-093 | pattern-currency-formatting | errors-empty | technical-writer | P3 | Error strings mix currency formats: '₹120' (no space, correct), but the price-r… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-094 | pattern-retry-cta-text | errors-empty | technical-writer | P3 | Retry CTAs across surfaces use a mix of 'Retry', 'Try again', 'Try Again', 'Ref… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-095 | pattern-payouts-history-empty | errors-empty | technical-writer | P3 | Vendor-portal payouts empty state body is wordy: 'Your payout history will appe… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-096 | pattern-notifications-empty | errors-empty | technical-writer | P3 | Vendor-portal notifications empty is informative but wordy: 'No admin requests… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-097 | pattern-orders-empty-customer | errors-empty | technical-writer | P3 | Customer orders-empty body is OK but has redundant phrasing in the filtered cas… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-098 | pattern-favorites-empty-customer | errors-empty | technical-writer | P3 | Web 'No favorites yet' is barebones (3 words); mobile is chatty ('No saved chef… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-099 | pattern-empty-category | errors-empty | technical-writer | P3 | Two surfaces show 'No items in this category' as an empty state — fine, but nei… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-100 | pattern-cart-minimum | errors-empty | technical-writer | P3 | 'Minimum order not met / Add {amount} more to proceed' is fine but 'Add {amount… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-101 | pattern-no-posts-yet | errors-empty | technical-writer | P3 | Social feed empty: web shows 'No posts yet' (3 words); mobile shows 'No posts y… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-102 | pattern-dashboard-recent-activity | errors-empty | technical-writer | P3 | Admin 'Recent activity' empty is two-line: 'No recent activity / Platform event… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-103 | pattern-audit-logs-empty | errors-empty | technical-writer | P3 | Audit-logs empty is a filtered-empty case but reads like a truly-empty case. | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-104 | pattern-data-exports-download-failed | errors-empty | technical-writer | P3 | 'Download failed' (2 words) on admin exports — formula-compliant but bare. | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-105 | pattern-platform-settings-zones | errors-empty | technical-writer | P3 | Platform-settings zones empty has a great context-aware message but uses an em-… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-106 | pattern-security-no-keys | errors-empty | technical-writer | P3 | Admin API keys empty is bare: 'No keys yet.' | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-107 | pattern-dashboard-allclear | errors-empty | technical-writer | P3 | Admin dashboard 'No pending chef applications.' is bare; can give value-add con… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-108 | pattern-staff-permissions-data | errors-empty | technical-writer | P3 | Admin staff permissions empty has awkward formulation: 'No specific permissions… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-109 | pattern-approval-history-empty | errors-empty | technical-writer | P3 | Approval history empty 'No history recorded yet.' is fine but the 'recorded' wo… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-110 | pattern-mobile-onb-step3-error | errors-empty | technical-writer | P3 | Mobile-customer onboarding setup failure alert: 'Setup failed / Something went… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-111 | pattern-document-required-alerts-drift | errors-empty | technical-writer | P3 | Mobile-vendor onboarding has multiple 'Required' alerts that mix title and body… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-112 | pattern-step1-personal-validation | errors-empty | technical-writer | P3 | Vendor-portal onboarding step 1 validation listing is bare ('Full name is requi… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-113 | pattern-checkout-zod | errors-empty | technical-writer | P3 | Mobile-customer checkout address Zod errors duplicate the same patterns from on… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-114 | pattern-prep-time-validation | errors-empty | technical-writer | P3 | Vendor-portal menu form prep-time validation uses 'Prep time required / Prep ti… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-115 | pattern-card-empty-mobile-customer | errors-empty | technical-writer | P3 | Mobile-customer cart-sheet says 'Your cart is empty' (4 words, no period); chec… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-116 | pattern-customer-orders-empty-emoji | errors-empty | technical-writer | P3 | Mobile-customer orders empty uses emoji-in-copy: '🍽️ No orders yet / Browse che… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-117 | pattern-customer-catering-empty-emoji | errors-empty | technical-writer | P3 | Mobile-customer catering empty uses emoji-in-copy: '🍽️ No requests yet / Submit… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-118 | pattern-mobile-vendor-photo-delete | errors-empty | technical-writer | P3 | Mobile-vendor photo-delete confirm uses 'Remove this photo?' with title 'Delete… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-119 | pattern-mobile-vendor-reply-fallback | errors-empty | technical-writer | P3 | Two near-identical reply-too-short errors in the same screen: 'Reply must be at… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-120 | pattern-mobile-vendor-toggle-online-status | errors-empty | technical-writer | P3 | Toast 'Failed to update status' reused in dashboard + active delivery — bare an… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-121 | pattern-customer-chef-load-error | errors-empty | technical-writer | P3 | Customer mobile chef-detail load error reads 'Failed to load chef details. Plea… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-122 | pattern-otto-no-recovery | errors-empty | technical-writer | P3 | Otto chat retry button has no associated error message text in inventory — only… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-123 | pattern-mobile-onboarding-empty-specialties | errors-empty | technical-writer | P3 | Vendor-portal kitchen step shows 'No specialties added yet' — fine, but should… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-124 | pattern-mobile-vendor-orders-history-empty | errors-empty | technical-writer | P3 | Mobile-vendor 'No order history yet' is bare (4 words); web equivalent has a bo… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-125 | pattern-mobile-vendor-onb-fullname | errors-empty | technical-writer | P3 | Mobile-vendor full-name validation matches the same 'X must be at least N chara… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-126 | pattern-mobile-vendor-policies-required | errors-empty | technical-writer | P3 | 'Please accept the terms and conditions to continue.' — 'Please' filler; 'terms… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-127 | pattern-mobile-vendor-menu-empty | errors-empty | technical-writer | P3 | 'No menu items yet. Tap + to add your first item.' — fine, uses formula well, b… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-128 | pattern-mobile-vendor-reviews-empty | errors-empty | technical-writer | P3 | 'No reviews yet. Your first review will appear here.' — friendly but slightly p… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-129 | pattern-vendor-portal-menu-empty | errors-empty | technical-writer | P3 | Vendor-portal menu empty has two branches — empty list and filtered-empty — bot… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-130 | pattern-vendor-portal-menu-error | errors-empty | technical-writer | P3 | 'Failed to load menu / Something went wrong while fetching your menu items. / T… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-131 | pattern-vendor-portal-payouts-empty-error | errors-empty | technical-writer | P3 | Payouts page bundles empty + error states in one inventory row with 18 words. | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-132 | pattern-vendor-portal-earnings-error | errors-empty | technical-writer | P3 | 'Unable to load earnings / Please try again later.' — 'Unable to' is more burea… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-133 | pattern-image-validation-toast | errors-empty | technical-writer | P3 | Vendor-portal menu file-upload validation prefixes filename then says 'Invalid… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-134 | pattern-staff-default-superadmin | errors-empty | technical-writer | P3 | Repeated phrase 'default super admin' is technical jargon for the user. | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-135 | pattern-no-pending-invites | errors-empty | technical-writer | P3 | Admin staff invitations empty: 'No pending invitations.' Period inconsistency v… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-136 | pattern-mobile-onboarding-step3 | errors-empty | technical-writer | P3 | Mobile-customer preferences setup-failure: 'Setup failed / Something went wrong… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-137 | pattern-mobile-customer-checkout-cart-empty | errors-empty | technical-writer | P3 | Mobile-customer checkout empty cart 'Your cart is empty.' — fine but missing re… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-138 | pattern-validation-event-date-future | errors-empty | technical-writer | P3 | Mobile-customer catering: 'Event date must be in the future' — fine but uses fo… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-139 | pattern-validation-event-type | errors-empty | technical-writer | P3 | Mobile-customer catering 'Event type is required' — bare; helper text would sol… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-140 | pattern-mobile-customer-profile-validation | errors-empty | technical-writer | P3 | Mobile-customer profile bundles 'First name is required / Last name is required… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-141 | pattern-mobile-vendor-display-name | errors-empty | technical-writer | P3 | Mobile-vendor 'Display name is required.' uses period; vendor-portal 'Full name… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-142 | pattern-mobile-vendor-update-failures | errors-empty | technical-writer | P3 | Three near-identical 'Failed to update X. Please try again.' on mobile-vendor. | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-143 | pattern-mobile-vendor-photo-delete-fail | errors-empty | technical-writer | P3 | 'Failed to delete photo.' (3 words, no recovery). | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-144 | pattern-mobile-delivery-tone | errors-empty | technical-writer | P3 | Mobile-delivery 'Accept a delivery from the Available tab to get started.' is 1… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-145 | pattern-mobile-delivery-available-empty | errors-empty | technical-writer | P3 | 'No deliveries available nearby. Pull to refresh.' is 7 words — fine for driver… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-146 | pattern-mobile-delivery-staff-lock | errors-empty | technical-writer | P3 | Mobile-delivery staff/fleet locked screens use 'Contact your administrator to r… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-147 | pattern-csv-vs-comma-list | errors-empty | technical-writer | P3 | File-upload type-error lists use comma-separated formats: 'JPEG, PNG, WebP, PDF… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-148 | pattern-mobile-vendor-cuisines-error | errors-empty | technical-writer | P3 | 'Select at least one cuisine type' / 'Select at least one cuisine' / 'Select at… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-149 | pattern-driver-onboarding-prereq | errors-empty | technical-writer | P3 | Five 'Please complete X' variants on driver onboarding say the same thing sligh… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-150 | pattern-driver-onboarding-state | errors-empty | technical-writer | P3 | Driver application status: 'Your application has already been approved' / 'Your… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-151 | pattern-validate-vs-zod-summary | errors-empty | technical-writer | P3 | Inventory shows vp-ux-menu-form-validation has 60 words bundling 15 Zod message… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-152 | pattern-vendor-onboarding-min-radius | errors-empty | technical-writer | P3 | Vendor-portal: 'Minimum 1 km radius' (4 words, no period). Style guide expects… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-153 | pattern-mobile-vendor-radius-range | errors-empty | technical-writer | P3 | Mobile-vendor radius validation: 'Service radius must be between 1 and 50 km' —… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-154 | pattern-driver-step-payout | errors-empty | technical-writer | P3 | Driver onboarding payout errors use bare 'Please fill in all bank details' (5 w… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-155 | pattern-driver-onboarding-step1-required | errors-empty | technical-writer | P3 | Driver onboarding step1 'Please fill in all required fields' (5 words) is reuse… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-156 | pattern-driver-step-load-review | errors-empty | technical-writer | P3 | 'Failed to load review data' on driver step-5 — bare, no recovery. | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-157 | pattern-driver-step-submit | errors-empty | technical-writer | P3 | 'Failed to submit. Please try again.' on driver step-5 submission. | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-158 | pattern-driver-plan-load | errors-empty | technical-writer | P3 | 'Failed to load subscription plans' (4 words) on driver step-4. | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-159 | pattern-driver-plan-select | errors-empty | technical-writer | P3 | 'Failed to select plan. Please try again.' (6 words). | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-160 | pattern-driver-invite-create | errors-empty | technical-writer | P3 | 'Failed to create invitation' (3 words, no recovery). | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-161 | pattern-driver-invite-required | errors-empty | technical-writer | P3 | 'Email and role are required' (5 words) — fine but missing period vs other admi… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-162 | pattern-driver-partner-status | errors-empty | technical-writer | P3 | Driver fleet admin: 'Failed to update partner status' / 'Failed to verify partn… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-163 | pattern-driver-accept-failed | errors-empty | technical-writer | P3 | Driver mobile 'Failed to accept delivery' (3 words) — high-stakes moment, deser… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-164 | pattern-driver-upload-fail | errors-empty | technical-writer | P3 | 'Upload failed. Please try again.' / 'File too large. Max {n}MB for {label}' on… | [findings/errors-empty.md](findings/errors-empty.md) |
| TW-165 | pattern-driver-referral-invalid | errors-empty | technical-writer | P3 | 'Invalid referral code' (3 words) — bare. For driver onboarding where bonuses d… | [findings/errors-empty.md](findings/errors-empty.md) |

### Index — transactional

441 findings. Detail: [findings/transactional.md](findings/transactional.md)

| finding_id | surface_id | category | lens | severity | issue (≤80 chars) | detail link |
|---|---|---|---|---|---|---|
| BA-TX-001 | api-email-delivery-assigned | transactional | business-analyst | P0 | DeliveryAssignedHTML template written for drivers is routed to customers via th… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-002 | api-push-chef-new-order-actionable | transactional | business-analyst | P0 | Two queue groups fire on the same order-created NATS event and send two push no… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-003 | api-inapp-chef-new-order | transactional | business-analyst | P0 | Three distinct 'new order' copy variants exist across the system: 'New Order' (… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-004 | api-email-order-confirm | transactional | business-analyst | P0 | Order confirmation email is missing GST breakdown, chef name, delivery address,… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-005 | api-push-user-welcome | transactional | business-analyst | P0 | Welcome push notification says 'Welcome to HomeChef!' but the welcome email say… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-006 | api-email-order-status-cancelled | transactional | business-analyst | P0 | Cancellation email contains no refund timeline, no contact point, and no reason… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-007 | api-inapp-approval-approved-chef | transactional | business-analyst | P0 | Approval notification body uses raw type slug 'menu_item' directly in user-faci… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-008 | api-inapp-approval-rejected-chef | transactional | business-analyst | P0 | Rejection notification body exposes raw admin notes verbatim: 'Your menu_item h… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-009 | api-push-order-update-customer-deeplink | transactional | business-analyst | P0 | Three separate status-label maps exist for order status updates — email_templat… | [findings/transactional.md](findings/transactional.md) |
| BV-001 | api-push-user-welcome | transactional | brand-voice | P0 | Brand identity contradiction: push says 'HomeChef', welcome email says 'Fe3dr',… | [findings/transactional.md](findings/transactional.md) |
| BV-002 | api-email-account-reminder | transactional | brand-voice | P0 | Brand-name compound 'Fe3dr HomeChef' appears in subject line and body — neither… | [findings/transactional.md](findings/transactional.md) |
| BV-003 | md-trx-002 | transactional | brand-voice | P0 | Driver-facing safety modal uses 'HomeChef Delivery' brand name while all chef/c… | [findings/transactional.md](findings/transactional.md) |
| BV-004 | md-trx-010 | transactional | brand-voice | P0 | Support email hardcoded as 'support@homechef.in' in driver-facing account-delet… | [findings/transactional.md](findings/transactional.md) |
| BV-005 | api-push-order-update-customer | transactional | brand-voice | P0 | Order-status enum drift: three independent label maps for the same 7 statuses (… | [findings/transactional.md](findings/transactional.md) |
| BV-006 | api-push-chef-new-order-actionable | transactional | brand-voice | P0 | Chef receives TWO push notifications with different copy on the same NATS event… | [findings/transactional.md](findings/transactional.md) |
| BV-021 | api-email-support-created | transactional | brand-voice | P0 | Support emails (created + update) are inline HTML in services/email.go — bypass… | [findings/transactional.md](findings/transactional.md) |
| BV-025 | api-inapp-approval-approved-chef | transactional | brand-voice | P0 | Raw snake_case enum slug surfaces to chef: 'Your menu_item has been approved' /… | [findings/transactional.md](findings/transactional.md) |
| BV-029 | api-push-order-cancelled-both | transactional | brand-voice | P0 | Identical push body 'Order has been cancelled' sent to BOTH customer and chef.… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-001 | api-email-order-confirm | transactional | legal | P0 | Order confirmation email is missing every element of a GST tax invoice — no GST… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-002 | api-email-order-confirm | transactional | legal | P0 | Currency rendered as raw ₹%.2f with no rounding rule or breakup; no indication… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-003 | api-email-order-confirm | transactional | legal | P0 | Order confirmation email does not display chef's FSSAI licence number — FSSAI m… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-004 | api-email-order-confirm | transactional | legal | P0 | No allergen disclosure or 'check label before consuming' callout on the order c… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-005 | api-email-order-confirm | transactional | legal | P0 | No refund/cancellation policy summary on the order confirmation — RBI Payment A… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-009 | api-email-order-status-cancelled | transactional | legal | P0 | Cancellation email omits refund timeline, refund channel, and grievance pathway… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-016 | api-push-user-welcome | transactional | legal | P0 | Welcome push ('Welcome to HomeChef!') and welcome email ('Welcome to Fe3dr!') a… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-017 | api-email-welcome | transactional | legal | P0 | Welcome email lacks a DPDP §5 notice of data processing — required to be served… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-032 | ap-chefs-toast-suspended | transactional | legal | P0 | 'Chef suspended' toast confirms admin action but does not surface that the chef… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-040 | api-email-delivery-assigned | transactional | legal | P0 | DeliveryAssigned email template is used for BOTH driver and customer (routing b… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-044 | md-trx-001 | transactional | legal | P0 | Background-location rationale modal title 'Background Location Needed' — IMPORT… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-045 | md-trx-002 | transactional | legal | P0 | Background-location rationale body does not name the driver as Data Principal,… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-049 | web-tx-orderdetail-toasts | transactional | legal | P0 | 'Order cancelled successfully' toast is silent on refund timeline and refund ch… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-055 | md-trx-010 | transactional | legal | P0 | Account-deletion path is 'Contact support at support@homechef.in' — DPDP §6(4)… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-079 | dp-tx-toast-partner-suspend | transactional | legal | P0 | 'Partner suspended' admin toast — gig-worker adverse-action; same severe due-pr… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-126 | cross-cutting-transactional | transactional | legal | P0 | Cross-cutting: brand identity drift across transactional channels — 'Fe3dr', 'H… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-127 | cross-cutting-transactional | transactional | legal | P0 | Cross-cutting: NO transactional email contains the grievance officer footer, DP… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-128 | cross-cutting-transactional | transactional | legal | P0 | Cross-cutting: NO transactional email or SMS surface shows the chef's FSSAI lic… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-129 | cross-cutting-transactional | transactional | legal | P0 | Cross-cutting: refund timeline disclosure absent across cancellation, refund-in… | [findings/transactional.md](findings/transactional.md) |
| TW-001 | api-email-account-reminder | transactional | technical-writer | P0 | Brand drift inside email subject — 'Fe3dr HomeChef' compound name contradicts t… | [findings/transactional.md](findings/transactional.md) |
| TW-002 | api-push-user-welcome | transactional | technical-writer | P0 | Brand name inside welcome push contradicts welcome email — push says 'HomeChef'… | [findings/transactional.md](findings/transactional.md) |
| TW-023 | api-email-order-status-cancelled | transactional | technical-writer | P0 | Cancellation email omits refund timeline and who-to-contact — a P0 transactiona… | [findings/transactional.md](findings/transactional.md) |
| TW-025 | api-email-chef-new-order | transactional | technical-writer | P0 | SLA commitment in info-box ('Customers expect a response within 5 minutes') is… | [findings/transactional.md](findings/transactional.md) |
| TW-031 | api-email-delivery-assigned | transactional | technical-writer | P0 | Routing bug: function returns driver-oriented copy ('You've been assigned order… | [findings/transactional.md](findings/transactional.md) |
| TW-034 | api-email-support-created | transactional | technical-writer | P0 | Inline HTML bypasses emailBase wrapper — recipient gets no header, no footer, n… | [findings/transactional.md](findings/transactional.md) |
| TW-037 | api-email-support-update | transactional | technical-writer | P0 | Same unbranded-email bug as TW-034 — inline HTML, no emailBase wrapper, no priv… | [findings/transactional.md](findings/transactional.md) |
| TW-040 | api-push-order-update-customer-deeplink | transactional | technical-writer | P0 | Customer receives TWO pushes for the same status change with different copy. Pu… | [findings/transactional.md](findings/transactional.md) |
| TW-059 | api-push-order-cancelled-both | transactional | technical-writer | P0 | Cancellation push body 'Order has been cancelled' is the SAME for both customer… | [findings/transactional.md](findings/transactional.md) |
| TW-060 | api-inapp-order-cancelled-both | transactional | technical-writer | P0 | Same dual-audience body problem as TW-059 on the in-app side | [findings/transactional.md](findings/transactional.md) |
| TW-068 | api-inapp-approval-rejected-chef | transactional | technical-writer | P0 | Triple problem: raw slug, verbatim admin notes, and 'Your menu_item has been re… | [findings/transactional.md](findings/transactional.md) |
| TW-069 | api-push-approval-rejected-chef | transactional | technical-writer | P0 | Same as TW-068 on the push path. Title 'Request Rejected' is also a banned hars… | [findings/transactional.md](findings/transactional.md) |
| TW-076 | api-success-message-generic | transactional | technical-writer | P0 | API path leak in error message: 'No Stripe account — call /chef/stripe/connect… | [findings/transactional.md](findings/transactional.md) |
| TW-106 | vp-onb-policies-ready-banner | transactional | technical-writer | P0 | Banner sets an SLA commitment '24-48 hours' the platform may not honour (admin-… | [findings/transactional.md](findings/transactional.md) |
| TW-108 | vp-onb-submit-success | transactional | technical-writer | P0 | Same 24-48h SLA in submission toast — duplicates TW-106 risk + lives in two pla… | [findings/transactional.md](findings/transactional.md) |
| TW-133 | mc-profile-logout-confirm | transactional | technical-writer | P0 | Title 'Log out' uses banned compound 'Log out' (Sec 3: 'Sign out'). Button 'Log… | [findings/transactional.md](findings/transactional.md) |
| TW-142 | md-trx-006 | transactional | technical-writer | P0 | 'Logout' as alert title uses banned 'Logout' (one-word) per Sec 3 — should be '… | [findings/transactional.md](findings/transactional.md) |
| TW-143 | md-trx-007 | transactional | technical-writer | P0 | 'Are you sure you want to logout?' uses banned compound 'logout' and weak modal… | [findings/transactional.md](findings/transactional.md) |
| TW-150 | md-trx-002 | transactional | technical-writer | P0 | Background location rationale is 31 words — far exceeds driver 12-word telegrap… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-010 | api-email-welcome | transactional | business-analyst | P1 | Welcome email uses 'amazing' and 'talented' (style-guide banned adjectives) and… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-011 | api-email-chef-verified | transactional | business-analyst | P1 | Chef verification email says 'Your kitchen has been verified' but in-app notifi… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-012 | api-email-chef-new-order | transactional | business-analyst | P1 | Chef new-order email sets 'Customers expect a response within 5 minutes' as a b… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-013 | api-push-order-cancelled-both | transactional | business-analyst | P1 | Same push body 'Order has been cancelled' is sent to both customer and chef wit… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-014 | api-email-support-created | transactional | business-analyst | P1 | Support ticket creation and update emails use inline unbranded HTML with no ema… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-015 | api-email-support-update | transactional | business-analyst | P1 | Support update email subject 'Ticket Update — #%s' and body 'Your support ticke… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-016 | api-email-account-reminder | transactional | business-analyst | P1 | Account-already-exists email subject says 'Fe3dr HomeChef account' — combining… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-017 | api-push-delivery-assigned-customer | transactional | business-analyst | P1 | Customer delivery-partner-assigned push notification omits the driver's name ev… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-018 | vp-onb-submit-success | transactional | business-analyst | P1 | Vendor portal submission toast says '24-48 hours' review SLA but mobile vendor… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-019 | md-trx-006 | transactional | business-analyst | P1 | Driver app logout confirmation title says 'Logout' (single word) — style guide… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-020 | mc-profile-logout-confirm | transactional | business-analyst | P1 | Customer app logout confirmation has title-case inconsistency: dialog title 'Lo… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-021 | mv-onb-pending-rejected-body | transactional | business-analyst | P1 | Chef rejection screen shows 'Please review the feedback and resubmit your appli… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-022 | web-tx-checkout-payment-toasts | transactional | business-analyst | P1 | 'Payment successful!' toast uses an exclamation mark — style guide Rule 1 limit… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-023 | web-tx-catering-request-submit | transactional | business-analyst | P1 | 'Catering request submitted successfully!' toast fires an exclamation on a form… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-024 | mv-onb-pending-submitted-title | transactional | business-analyst | P1 | Mobile vendor pending screen title 'Application Submitted!' uses exclamation —… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-025 | md-trx-008 | transactional | business-analyst | P1 | Driver app subscription management alerts user 'Visit the web portal to manage… | [findings/transactional.md](findings/transactional.md) |
| BV-007 | api-inapp-chef-new-order | transactional | brand-voice | P1 | Three distinct in-app 'new order' notification titles in the codebase: 'New Ord… | [findings/transactional.md](findings/transactional.md) |
| BV-008 | api-email-chef-verified | transactional | brand-voice | P1 | Cross-surface noun drift for the same event: email says 'kitchen is verified',… | [findings/transactional.md](findings/transactional.md) |
| BV-011 | api-email-chef-new-order | transactional | brand-voice | P1 | Emoji '🔔' and exclamation 'New Order!' in chef-facing email — style guide chef… | [findings/transactional.md](findings/transactional.md) |
| BV-012 | api-email-delivery-assigned | transactional | brand-voice | P1 | Emoji '🚗' and 'New Delivery!' in driver-facing email — driver tone budget = 0 e… | [findings/transactional.md](findings/transactional.md) |
| BV-016 | api-email-order-status-cancelled | transactional | brand-voice | P1 | Decorative ❌ emoji on a sensitive event (cancellation) — feels like a Slack rea… | [findings/transactional.md](findings/transactional.md) |
| BV-017 | api-email-order-confirm | transactional | brand-voice | P1 | 'Order Confirmed!' uses Title Case + exclamation; customer-facing budget allows… | [findings/transactional.md](findings/transactional.md) |
| BV-018 | api-email-welcome | transactional | brand-voice | P1 | Welcome email mixes hype tone with brand drift: 'Welcome to Fe3dr!' subject + '… | [findings/transactional.md](findings/transactional.md) |
| BV-022 | api-email-support-update | transactional | brand-voice | P1 | Same as BV-021: unbranded inline HTML. Plus 'Support Ticket Update.' as a sente… | [findings/transactional.md](findings/transactional.md) |
| BV-026 | api-push-approval-approved-chef | transactional | brand-voice | P1 | Title 'Request Approved!' has exclamation; in-app 'Request Approved' does not.… | [findings/transactional.md](findings/transactional.md) |
| BV-027 | api-inapp-approval-rejected-chef | transactional | brand-voice | P1 | 'Your %s has been rejected. Notes: %s' — appends raw admin notes verbatim. Risk… | [findings/transactional.md](findings/transactional.md) |
| BV-028 | api-inapp-chef-responded | transactional | brand-voice | P1 | 'Chef Responded: ' + chef-authored message — admins receive chef text verbatim… | [findings/transactional.md](findings/transactional.md) |
| BV-030 | api-push-delivery-pickedup-customer | transactional | brand-voice | P1 | Push title 'Order On The Way!' uses Title Case (every word capitalized) + excla… | [findings/transactional.md](findings/transactional.md) |
| BV-037 | api-success-message-generic | transactional | brand-voice | P1 | 43 API success-message strings have inconsistent 'successfully' suffix — some h… | [findings/transactional.md](findings/transactional.md) |
| BV-040 | web-tx-checkout-payment-toasts | transactional | brand-voice | P1 | 'Payment successful!' on a routine customer payment — exclamation budget should… | [findings/transactional.md](findings/transactional.md) |
| BV-041 | web-tx-delivery-actions | transactional | brand-voice | P1 | Driver-portal toasts 'Delivery accepted!' and 'Delivery completed! Great job!'… | [findings/transactional.md](findings/transactional.md) |
| BV-048 | mc-profile-logout-confirm | transactional | brand-voice | P1 | Customer mobile alert: title 'Log out' (sentence case, two words) but button la… | [findings/transactional.md](findings/transactional.md) |
| BV-049 | md-trx-006 | transactional | brand-voice | P1 | Driver mobile alert title 'Logout' — single word + wrong verb. Style Guide bans… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-006 | api-email-order-confirm | transactional | legal | P1 | Order confirmation does not include the delivery address or estimated time — in… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-007 | api-email-order-confirm | transactional | legal | P1 | No grievance-officer contact (name, email, phone, address) in the transactional… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-008 | api-email-order-confirm | transactional | legal | P1 | Email does not state the seller of record (chef name) — under CPER 2020 a marke… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-010 | api-push-order-cancelled-both | transactional | legal | P1 | Cancellation push notification body 'Order has been cancelled' is identical for… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-011 | api-inapp-order-cancelled-both | transactional | legal | P1 | In-app cancellation notification has same generic body as push and omits refund… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-012 | api-email-order-status-confirmed | transactional | legal | P1 | Status emails (confirmed/preparing/ready/picked up/on way/delivered) use one te… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-013 | api-email-order-status-delivered | transactional | legal | P1 | Delivery confirmation email does not prompt rating/complaint pathway or restate… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-015 | api-push-order-created-chef | transactional | legal | P1 | Push notifications are not tagged with their TCCCPR classification (transaction… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-018 | api-email-welcome | transactional | legal | P1 | Welcome email contains marketing copy and a 'Start Exploring' CTA but no unsubs… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-019 | api-push-user-welcome | transactional | legal | P1 | Brand drift between welcome push ('HomeChef') and welcome email ('Fe3dr') — exp… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-020 | api-email-account-reminder | transactional | legal | P1 | Account-reminder email subject mixes brand names ('Fe3dr HomeChef') — brand dri… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-021 | api-email-password-reset | transactional | legal | P1 | Password reset email content not visible in inventory excerpt — verify token ex… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-024 | api-email-staff-invite | transactional | legal | P1 | Staff invitation email contains 'role injected as string' — risk of admin-autho… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-025 | api-inapp-staff-invite | transactional | legal | P1 | Staff invitation in-app notification title/message are admin-authored fields wi… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-027 | api-email-support-created | transactional | legal | P1 | Support ticket created email is inline HTML with no branded wrapper, no grievan… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-028 | api-email-support-update | transactional | legal | P1 | Support ticket update email is unbranded, no footer/privacy links — same gap as… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-031 | ap-chefs-toast-rejected | transactional | legal | P1 | 'Chef rejected' admin toast — no record of reason, no notice to chef about appe… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-033 | ap-users-toast-suspended | transactional | legal | P1 | 'User suspended' admin toast — DPDP-relevant action (suspension affects access… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-035 | api-inapp-approval-rejected-chef | transactional | legal | P1 | Approval rejection in-app notification appends raw admin notes verbatim — admin… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-036 | api-push-approval-rejected-chef | transactional | legal | P1 | Approval rejection push notification carries the same raw admin notes — broadca… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-039 | api-inapp-chef-responded | transactional | legal | P1 | Chef-authored response message is surfaced to ALL admins with no profanity filt… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-041 | api-push-delivery-assigned-driver | transactional | legal | P1 | Driver delivery-assigned push contains no insurance/liability reminder, no acce… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-042 | api-email-delivery-assigned | transactional | legal | P1 | Driver-side delivery-assigned email contains no rate breakdown for the trip — g… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-046 | md-trx-003 | transactional | legal | P1 | Battery / privacy reassurance ('Battery usage is minimised by only tracking eve… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-047 | web-tx-checkout-payment-toasts | transactional | legal | P1 | Razorpay/Stripe success/error/cancel toasts (8 strings) — none state the mercha… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-048 | web-tx-orderdetail-toasts | transactional | legal | P1 | 'Payment verification failed — please contact support' toast does not provide a… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-051 | ap-secsettings-toasts | transactional | legal | P1 | 'All other sessions revoked. You may need to sign in again.' — admin-facing sec… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-053 | web-tx-profile-2fa | transactional | legal | P1 | 'Two-factor authentication disabled' toast — confirm a corresponding email is s… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-054 | ap-secsettings-toasts | transactional | legal | P1 | Admin 2FA toasts include '2FA disabled' — confirm admin-side 2FA-off action tri… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-057 | mv-onb-pending-rejected-reason-prefix | transactional | legal | P1 | Driver/chef rejection screens show 'Reason: ...' with admin-authored content —… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-064 | api-success-message-generic | transactional | legal | P1 | Generic API success strings — 'Chef rejected', 'Chef suspended', 'User suspende… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-067 | vp-ux-profile-doc-toasts | transactional | legal | P1 | Chef document upload toast '{label} uploaded' is generic — chef KYC documents (… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-068 | dp-tx-toast-step3-uploaded | transactional | legal | P1 | Driver document upload toast '{label} uploaded' — same KYC sensitivity as chef;… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-069 | vp-ux-settings-payout-toasts | transactional | legal | P1 | Chef payout-details toast 'Payout details saved' — bank account / UPI / IFSC en… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-070 | dp-tx-toast-payout-saved | transactional | legal | P1 | Driver payout-details toast 'Payout details saved' — same RBI PA + DPDP concern… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-071 | web-tx-catering-quotes | transactional | legal | P1 | 'Quote accepted!' toast does not surface the legal effect of acceptance — accep… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-080 | dp-tx-toast-partner-verified | transactional | legal | P1 | 'Partner verified successfully' admin toast — confirm verification includes bac… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-081 | ap-staffdetail-toasts | transactional | legal | P1 | 'Staff member deactivated' / 'Staff member reactivated' — access-removal action… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-086 | web-tx-chef-menu | transactional | legal | P1 | Menu CRUD toasts 'Menu item created successfully' — toast says nothing about FS… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-087 | vp-ux-menu-form-toasts | transactional | legal | P1 | Vendor portal menu form toasts — same allergen-disclosure gap as web; 'At least… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-088 | web-tx-chef-social | transactional | legal | P1 | Chef social post toasts 'Post created / Post updated / Post deleted' — chef soc… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-092 | mv-undo-rejected | transactional | legal | P1 | Mobile vendor 'Order rejected' undo snackbar — order rejection by chef triggers… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-109 | api-inapp-driver-onboarding-admins | transactional | legal | P1 | Driver onboarding submitted notification fanned to ALL admins with city of driv… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-115 | api-email-chef-new-order | transactional | legal | P1 | Chef new-order email body shows order total ₹%.2f but no GST breakdown or chef… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-121 | ap-approvaldetail-toasts | transactional | legal | P1 | 'Approval request approved / Approval request rejected / More information reque… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-130 | cross-cutting-transactional | transactional | legal | P1 | Cross-cutting: SMS path not visible in inventory — confirm DLT registration, he… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-131 | cross-cutting-transactional | transactional | legal | P1 | Cross-cutting: every transactional channel reaches a user but no central record… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-132 | cross-cutting-transactional | transactional | legal | P1 | Cross-cutting: emoji-prefixed status emails (🔔 / 👨‍🍳 / ✅ / 🚗 / 🎉 / ❌) — legally… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-134 | cross-cutting-transactional | transactional | legal | P1 | Cross-cutting: every chef-action and driver-action notification confirms platfo… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-135 | cross-cutting-transactional | transactional | legal | P1 | Cross-cutting: no transactional surface references the platform's IT Rules 2021… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-136 | cross-cutting-transactional | transactional | legal | P1 | Cross-cutting: no per-channel preference centre referenced from any transaction… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-141 | cross-cutting-transactional | transactional | legal | P1 | Cross-cutting: 'verified' / 'approved' / 'rejected' / 'suspended' actions throu… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-143 | cross-cutting-transactional | transactional | legal | P1 | Cross-cutting: no transactional surface references the customer's right to file… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-144 | cross-cutting-transactional | transactional | legal | P1 | Cross-cutting: chef-side FSSAI compliance is recurrence-blind — no transactiona… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-149 | cross-cutting-transactional | transactional | legal | P1 | Cross-cutting: payment-related surfaces (success, failure, refund, cancel) do n… | [findings/transactional.md](findings/transactional.md) |
| TW-003 | api-email-welcome | transactional | technical-writer | P1 | Welcome email opens with marketing hype banned by Rule 5 (Restraint over urgenc… | [findings/transactional.md](findings/transactional.md) |
| TW-024 | api-email-chef-new-order | transactional | technical-writer | P1 | Subject and H2 'New order' fine, but body opens with '🔔 New Order!' violating R… | [findings/transactional.md](findings/transactional.md) |
| TW-043 | api-push-order-update-customer | transactional | technical-writer | P1 | Push body 'Your order has been confirmed by the chef!' (services/notifications.… | [findings/transactional.md](findings/transactional.md) |
| TW-047 | api-push-chef-new-order-actionable | transactional | technical-writer | P1 | Chef receives TWO pushes for the same NATS event: 'New Order' + 'You have a new… | [findings/transactional.md](findings/transactional.md) |
| TW-064 | api-inapp-approval-approved-chef | transactional | technical-writer | P1 | Raw enum slug exposed to user: 'Your menu_item has been approved' because appro… | [findings/transactional.md](findings/transactional.md) |
| TW-065 | api-push-approval-approved-chef | transactional | technical-writer | P1 | Same raw-slug bug on push body 'Your %s has been approved'. Title also has excl… | [findings/transactional.md](findings/transactional.md) |
| TW-066 | api-inapp-approval-info-chef | transactional | technical-writer | P1 | Same raw-slug bug; in addition, admin notes appended verbatim — chef sees raw a… | [findings/transactional.md](findings/transactional.md) |
| TW-089 | web-tx-checkout-payment-toasts | transactional | technical-writer | P1 | Payment toast set has voice drift: 'Payment successful!' (exclamation) vs 'Paym… | [findings/transactional.md](findings/transactional.md) |
| TW-137 | mv-onb-pending-submitted-body | transactional | technical-writer | P1 | 17 words across two sentences — exceeds vendor 20-word limit only marginally, B… | [findings/transactional.md](findings/transactional.md) |
| TW-160 | api-email-order-confirm | transactional | technical-writer | P1 | All 8 branded emails hardcode the brand name 'Fe3dr' in subject + footer + URLs… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-026 | md-trx-010 | transactional | business-analyst | P2 | Account deletion instructs drivers to email support@homechef.in — this is a har… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-027 | md-trx-011 | transactional | business-analyst | P2 | Driver profile validation failure alert uses 'Validation' as the title — a tech… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-028 | md-trx-013 | transactional | business-analyst | P2 | Driver profile save success alert title is 'Success' — generic non-contextual t… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-029 | web-tx-catering-quotes | transactional | business-analyst | P2 | 'Quote accepted! The chef has been notified.' toast uses an exclamation — for a… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-030 | web-tx-chefdetail-add-to-cart | transactional | business-analyst | P2 | 'Your cart has items from another chef. Clear cart first.' is a blocking dead e… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-031 | mc-menuitem-cross-chef | transactional | business-analyst | P2 | Mobile cross-chef cart dialog says 'Replace Cart?' with 'Replace' as the destru… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-032 | dp-tx-toast-accept-success | transactional | business-analyst | P2 | 'Delivery accepted!' delivery portal toast fires exclamation on an operational… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-033 | dp-tx-toast-step5-submitted | transactional | business-analyst | P2 | Driver onboarding submission toast 'Application submitted successfully!' uses a… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-034 | vp-onb-policies-ready-banner | transactional | business-analyst | P2 | 'You can start setting up your menu in the meantime' banner on the application-… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-035 | vp-ux-profile-doc-toasts | transactional | business-analyst | P2 | Document upload toast '{label} uploaded' passes through raw template variable —… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-036 | dp-tx-toast-step3-uploaded | transactional | business-analyst | P2 | Driver document upload toast '{label} uploaded' has the same raw variable inter… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-037 | dp-tx-toast-status-updated | transactional | business-analyst | P2 | 'Status updated' driver delivery toast is the emptiest possible confirmation —… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-038 | api-success-message-generic | transactional | business-analyst | P2 | 43 API success message strings have inconsistent 'successfully' suffix — 'Passw… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-039 | web-tx-delivery-actions | transactional | business-analyst | P2 | 'Delivery completed! Great job!' in the delivery portal adds motivational copy… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-040 | api-inapp-chef-responded | transactional | business-analyst | P2 | Chef-response-to-admin notification title is 'Chef Responded: ' + approval.Titl… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-041 | api-push-delivery-pickedup-customer | transactional | business-analyst | P2 | Push notification for delivery picked up says 'Order On The Way!' but the in-ap… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-042 | mc-catering-submit-success | transactional | business-analyst | P2 | 'Request Submitted! / Chefs will review and send quotes.' mobile catering succe… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-043 | mv-onb-docs-upload-success | transactional | business-analyst | P2 | 'Uploaded successfully' under document preview on mobile vendor onboarding is t… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-044 | api-inapp-order-status-customer | transactional | business-analyst | P2 | In-app order status notification title is 'Order Status Updated' while the push… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-045 | api-push-approval-approved-chef | transactional | business-analyst | P2 | Approval-approved push title says 'Request Approved!' (with exclamation) but in… | [findings/transactional.md](findings/transactional.md) |
| BV-009 | api-email-chef-verified-publish | transactional | brand-voice | P2 | NATS-published email title 'Your Chef Profile is Verified!' uses Title Case (ba… | [findings/transactional.md](findings/transactional.md) |
| BV-010 | api-inapp-chef-verified | transactional | brand-voice | P2 | Title 'Congratulations!' is loud — Style Guide Rule 1 prohibits urgency/celebra… | [findings/transactional.md](findings/transactional.md) |
| BV-013 | api-email-order-status-preparing | transactional | brand-voice | P2 | Decorative emoji '👨‍🍳' in customer email — Style Guide allows occasional custom… | [findings/transactional.md](findings/transactional.md) |
| BV-014 | api-email-order-status-onway | transactional | brand-voice | P2 | Exclamation 'Your order is on the way!' in customer email subject — Style Guide… | [findings/transactional.md](findings/transactional.md) |
| BV-015 | api-email-order-status-delivered | transactional | brand-voice | P2 | 'Enjoy your meal!' co-exists with in-app push 'Enjoy!' and another in-app statu… | [findings/transactional.md](findings/transactional.md) |
| BV-019 | api-email-welcome | transactional | brand-voice | P2 | 'Start Exploring' CTA is Title Case — Style Guide §4 requires sentence case for… | [findings/transactional.md](findings/transactional.md) |
| BV-020 | api-email-staff-invite | transactional | brand-voice | P2 | 'You're invited to join Fe3dr' + body 'You've been invited!' — two invitation a… | [findings/transactional.md](findings/transactional.md) |
| BV-023 | api-email-verify | transactional | brand-voice | P2 | 'Verify your email, %s. Thanks for signing up!' — exclamation again on entry su… | [findings/transactional.md](findings/transactional.md) |
| BV-024 | api-email-password-reset | transactional | brand-voice | P2 | Subject 'Reset your password — Fe3dr' + body 'We received a request to reset th… | [findings/transactional.md](findings/transactional.md) |
| BV-031 | api-inapp-delivery-pickedup-customer | transactional | brand-voice | P2 | In-app title 'Order Picked Up' uses Title Case. Style Guide §3 sentence case fo… | [findings/transactional.md](findings/transactional.md) |
| BV-032 | api-inapp-order-status-customer | transactional | brand-voice | P2 | In-app title 'Order Status Updated' vs push title 'Order Update' for the same e… | [findings/transactional.md](findings/transactional.md) |
| BV-033 | api-push-delivery-assigned-customer | transactional | brand-voice | P2 | Push 'A delivery partner has been assigned to your order and will pick it up so… | [findings/transactional.md](findings/transactional.md) |
| BV-034 | api-push-delivery-assigned-driver | transactional | brand-voice | P2 | Driver push title 'New Delivery Available' is Title Case; body 'A delivery near… | [findings/transactional.md](findings/transactional.md) |
| BV-036 | api-inapp-approval-created-admin | transactional | brand-voice | P2 | Title 'New Approval Request' Title Case. Body 'New approval request pending: %s… | [findings/transactional.md](findings/transactional.md) |
| BV-038 | web-tx-catering-quotes | transactional | brand-voice | P2 | 'Quote accepted! The chef has been notified.' — exclamation on a routine action… | [findings/transactional.md](findings/transactional.md) |
| BV-039 | web-tx-catering-request-submit | transactional | brand-voice | P2 | 'Catering request submitted successfully!' — exclamation + 'successfully' fille… | [findings/transactional.md](findings/transactional.md) |
| BV-042 | dp-tx-toast-accept-success | transactional | brand-voice | P2 | 'Delivery accepted!' (delivery-portal driver web) duplicates the same exclamati… | [findings/transactional.md](findings/transactional.md) |
| BV-043 | dp-tx-toast-step5-submitted | transactional | brand-voice | P2 | Driver onboarding submit toast 'Application submitted successfully!' — driver t… | [findings/transactional.md](findings/transactional.md) |
| BV-044 | vp-onb-submit-success | transactional | brand-voice | P2 | Chef onboarding submit 'Application submitted! We'll review and get back to you… | [findings/transactional.md](findings/transactional.md) |
| BV-045 | vp-onb-policies-ready-banner | transactional | brand-voice | P2 | 'You're all set to submit your application!' — exclamation on chef-facing onboa… | [findings/transactional.md](findings/transactional.md) |
| BV-046 | mv-onb-pending-submitted-title | transactional | brand-voice | P2 | Mobile-vendor 'Application Submitted!' — Title Case + exclamation in chef-facin… | [findings/transactional.md](findings/transactional.md) |
| BV-047 | mc-catering-submit-success | transactional | brand-voice | P2 | Mobile-customer 'Request Submitted! / Chefs will review and send quotes.' — Tit… | [findings/transactional.md](findings/transactional.md) |
| BV-050 | md-trx-007 | transactional | brand-voice | P2 | Driver mobile alert body 'Are you sure you want to logout?' — uses 'logout' as… | [findings/transactional.md](findings/transactional.md) |
| BV-051 | md-trx-011 | transactional | brand-voice | P2 | Driver alert title 'Validation' — generic system label, not brand voice. Reads… | [findings/transactional.md](findings/transactional.md) |
| BV-052 | md-trx-013 | transactional | brand-voice | P2 | Driver alert title 'Success' — generic non-brand label. Drop the title entirely… | [findings/transactional.md](findings/transactional.md) |
| BV-053 | md-trx-008 | transactional | brand-voice | P2 | Driver alert title 'Subscription' — generic label, not voice. | [findings/transactional.md](findings/transactional.md) |
| BV-054 | mv-profile-update-success-body | transactional | brand-voice | P2 | Mobile-vendor alert: title 'Success' (generic) + body 'Profile updated successf… | [findings/transactional.md](findings/transactional.md) |
| BV-057 | mc-menuitem-cross-chef | transactional | brand-voice | P2 | Cross-chef cart conflict: mobile says 'Replace Cart? / You have items from anot… | [findings/transactional.md](findings/transactional.md) |
| BV-059 | ap-staff-invite-success-title | transactional | brand-voice | P2 | Admin-portal 'Invitation created successfully!' — admin tone budget = 0 exclama… | [findings/transactional.md](findings/transactional.md) |
| BV-061 | ap-providerdetail-toasts | transactional | brand-voice | P2 | 'Connection successful ({ms}ms)' uses 'successful' as adjective in admin contex… | [findings/transactional.md](findings/transactional.md) |
| BV-062 | ap-secsettings-toasts | transactional | brand-voice | P2 | '2FA enabled successfully' but '2FA disabled' (no suffix) — same handler family… | [findings/transactional.md](findings/transactional.md) |
| BV-067 | web-tx-profile-2fa | transactional | brand-voice | P2 | 2FA toasts mix grand-statement and tiny-detail: 'Two-factor authentication enab… | [findings/transactional.md](findings/transactional.md) |
| BV-068 | web-tx-orderdetail-toasts | transactional | brand-voice | P2 | 'Payment confirmed' (this surface) vs 'Payment successful!' (checkout, BV-040)… | [findings/transactional.md](findings/transactional.md) |
| BV-069 | api-inapp-staff-invite | transactional | brand-voice | P2 | Title/Message are admin-authored (per inventory note). Brand voice cannot guara… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-014 | api-push-order-update-customer-deeplink | transactional | legal | P2 | Push and email statuses use three different sets of wording for the same lifecy… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-022 | api-email-password-reset | transactional | legal | P2 | Password reset email must not leak whether the account exists — verify wording… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-023 | api-email-verify | transactional | legal | P2 | Email verification link 24-hour expiry is stated, but verify single-use semanti… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-029 | api-email-chef-verified | transactional | legal | P2 | Chef verification email differs in wording from in-app notification ('kitchen'… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-030 | api-inapp-chef-verified | transactional | legal | P2 | Chef-verified in-app notification says 'You can now start accepting orders!' —… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-037 | api-inapp-approval-info-chef | transactional | legal | P2 | 'More Information Needed' notification surfaces admin-authored notes verbatim —… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-043 | api-push-delivery-assigned-customer | transactional | legal | P2 | Customer 'Delivery Partner Assigned' push does not include the driver's first n… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-050 | ap-settings-payment-feedback | transactional | legal | P2 | Admin payment-gateway toasts surface raw vendor error messages ('Saved, but Raz… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-052 | web-tx-profile-2fa | transactional | legal | P2 | 2FA toasts ('Backup codes copied to clipboard', 'Key copied') — copying TOTP se… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-056 | md-trx-009 | transactional | legal | P2 | Subscription management deferred to web ('Visit the web portal to manage your s… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-058 | mv-onb-pending-rejected-body | transactional | legal | P2 | 'Please review the feedback and resubmit your application' — does not surface a… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-060 | vp-onb-policies-ready-banner | transactional | legal | P2 | 'You're all set to submit your application!' banner does not summarise what the… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-061 | api-push-chef-new-order-actionable | transactional | legal | P2 | Chef receives TWO pushes with slightly different wording for the same 'new orde… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-062 | api-inapp-order-status-customer | transactional | legal | P2 | In-app order-status title 'Order Status Updated' vs push title 'Order Update' —… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-063 | api-inapp-delivery-pickedup-customer | transactional | legal | P2 | Customer in-app + push titles for the same event differ ('Order Picked Up' vs '… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-065 | api-success-message-generic | transactional | legal | P2 | 'Removed from favorites' / 'Address deleted' / 'Image deleted' — DPDP §12 (righ… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-066 | web-tx-profile-update | transactional | legal | P2 | Profile photo upload error 'Invalid file type. Use JPEG, PNG or WebP.' — confir… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-072 | web-tx-catering-request-submit | transactional | legal | P2 | 'Catering request submitted successfully!' — no consent capture for the chef to… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-076 | dp-tx-toast-online | transactional | legal | P2 | 'You are now online' / 'You are now offline' — driver-status change is a Code-o… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-077 | dp-tx-toast-accept-success | transactional | legal | P2 | 'Delivery accepted!' — driver accepts a trip-level micro-contract; toast is thi… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-078 | web-tx-delivery-actions | transactional | legal | P2 | Web driver dashboard toasts 'Delivery completed! Great job!' — celebratory tone… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-082 | dp-tx-toast-invite-copied | transactional | legal | P2 | 'Invite URL copied to clipboard' — invite URL is a credential-equivalent token;… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-084 | ap-platsettings-feedback | transactional | legal | P2 | 'Commission rates updated' / 'Delivery fees updated' — pricing-policy changes;… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-093 | mv-reviewdetail-success-body | transactional | legal | P2 | 'Your reply has been posted.' — chef reply to a customer review is platform-pub… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-094 | vp-ux-kitchen-photo-toasts | transactional | legal | P2 | Kitchen photo upload toasts — kitchen photographs may incidentally show people… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-096 | vp-ux-notifs-respond-toasts | transactional | legal | P2 | Chef response 'Response sent to admin' — chef-authored response visible to admi… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-097 | web-tx-profile-update | transactional | legal | P2 | Profile update toasts include avatar/file-size errors — confirm rejected files… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-098 | web-tx-profile-preferences-save | transactional | legal | P2 | 'Preferences updated' — preferences may include marketing/comms opt-in; confirm… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-099 | web-tx-profile-addresses | transactional | legal | P2 | Address CRUD toasts — addresses are personal data + may include delivery instru… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-100 | web-tx-onboarding-save | transactional | legal | P2 | Customer onboarding 'Your preferences have been saved!' — preferences captured… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-105 | md-trx-018 | transactional | legal | P2 | 'You do not have permission to invite staff.' — 403 error; ensure rejected acti… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-106 | api-success-message-generic | transactional | legal | P2 | API success string 'If the email exists, a reset link has been sent' — already… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-107 | api-success-message-generic | transactional | legal | P2 | API success string 'Email verified successfully' — confirm the verification act… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-108 | api-success-message-generic | transactional | legal | P2 | API success string 'Logged out successfully' — uses banned 'logged out' (style… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-110 | api-inapp-approval-created-admin | transactional | legal | P2 | Approval created notification fanned to all admins with raw 'Approval title inj… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-111 | md-trx-009 | transactional | legal | P2 | 'Visit the web portal to manage your subscription' — driver app deflects subscr… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-114 | api-email-chef-new-order | transactional | legal | P2 | Chef new-order email subject contains decorative emoji '🔔 New Order!' — flagged… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-117 | ap-staff-invite-success-body | transactional | legal | P2 | 'Share this link with the invitee to complete their registration.' — admin inst… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-119 | ap-providerdetail-toasts | transactional | legal | P2 | 'Connection successful ({ms}ms); Connection failed: {err}' — admin diagnostic;… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-125 | dp-tx-toast-step4-saved | transactional | legal | P2 | 'Plan selected successfully' — driver subscription plan; if plan implies revenu… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-133 | cross-cutting-transactional | transactional | legal | P2 | Cross-cutting: 'log in / login / logout' banned variants appear in mobile alert… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-137 | cross-cutting-transactional | transactional | legal | P2 | Cross-cutting: free-text injection from admins / chefs into notifications visib… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-138 | cross-cutting-transactional | transactional | legal | P2 | Cross-cutting: order status copy diverges across three code paths (statusLabels… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-140 | cross-cutting-transactional | transactional | legal | P2 | Cross-cutting: ratios of duplicate notifications — chef gets two pushes for new… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-142 | cross-cutting-transactional | transactional | legal | P2 | Cross-cutting: no reference in any transactional surface to the platform's role… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-146 | cross-cutting-transactional | transactional | legal | P2 | Cross-cutting: no transactional surface references age verification — DPDP §10… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-147 | cross-cutting-transactional | transactional | legal | P2 | Cross-cutting: customer addresses are stored permanently (no surfaced retention… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-148 | cross-cutting-transactional | transactional | legal | P2 | Cross-cutting: order data retention — order confirmation emails set up an expec… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-151 | cross-cutting-transactional | transactional | legal | P2 | Cross-cutting: NO reference in any transactional surface to data fiduciary iden… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-152 | cross-cutting-transactional | transactional | legal | P2 | Cross-cutting: cross-border data transfer — Razorpay, Stripe, Google Cloud Stor… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-153 | cross-cutting-transactional | transactional | legal | P2 | Cross-cutting: no breach-notification copy/template visible — DPDP §8(6) requir… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-154 | cross-cutting-transactional | transactional | legal | P2 | Cross-cutting: no transactional surface references chef/driver KYC re-verificat… | [findings/transactional.md](findings/transactional.md) |
| TW-004 | api-email-welcome | transactional | technical-writer | P2 | Welcome CTA button uses title-case 'Start Exploring' violating sentence-case ru… | [findings/transactional.md](findings/transactional.md) |
| TW-005 | api-email-welcome | transactional | technical-writer | P2 | Exclamation in subject 'Welcome to Fe3dr!' is a Rule 1 violation when the same… | [findings/transactional.md](findings/transactional.md) |
| TW-006 | api-email-verify | transactional | technical-writer | P2 | Verify-email CTA uses title-case 'Verify Email Address' (3 words is fine; case… | [findings/transactional.md](findings/transactional.md) |
| TW-008 | api-email-password-reset | transactional | technical-writer | P2 | CTA 'Reset Password' is title-case; should be sentence case per Sec 4 | [findings/transactional.md](findings/transactional.md) |
| TW-009 | api-email-staff-invite | transactional | technical-writer | P2 | H2 'You've been invited!' uses exclamation in a transactional admin email (Rule… | [findings/transactional.md](findings/transactional.md) |
| TW-010 | api-email-staff-invite | transactional | technical-writer | P2 | CTA 'Accept Invitation' is title-case; should be sentence case | [findings/transactional.md](findings/transactional.md) |
| TW-012 | api-email-order-confirm | transactional | technical-writer | P2 | H2 'Order Confirmed!' uses exclamation against Rule 1; H2 also uses title case… | [findings/transactional.md](findings/transactional.md) |
| TW-013 | api-email-order-confirm | transactional | technical-writer | P2 | 'Your home chef is preparing your meal!' adds an exclamation and assumes work h… | [findings/transactional.md](findings/transactional.md) |
| TW-014 | api-email-order-confirm | transactional | technical-writer | P2 | CTA 'Track Your Order' is title-case and 3 words including possessive — sentenc… | [findings/transactional.md](findings/transactional.md) |
| TW-016 | api-email-order-status-confirmed | transactional | technical-writer | P2 | Within-surface voice drift: H2 still says generic 'Order Update' across all 7 s… | [findings/transactional.md](findings/transactional.md) |
| TW-017 | api-email-order-status-preparing | transactional | technical-writer | P2 | Status label 'Your chef is preparing your meal' (email) does not match push lab… | [findings/transactional.md](findings/transactional.md) |
| TW-018 | api-email-order-status-ready | transactional | technical-writer | P2 | Label 'Your order is ready for pickup' contradicts customer-facing push 'Your o… | [findings/transactional.md](findings/transactional.md) |
| TW-019 | api-email-order-status-pickedup | transactional | technical-writer | P2 | Label uses 'delivery partner' (customer-facing term — good) while push (api-pus… | [findings/transactional.md](findings/transactional.md) |
| TW-020 | api-email-order-status-onway | transactional | technical-writer | P2 | Subject has exclamation 'Your order is on the way!' against Rule 1; sister surf… | [findings/transactional.md](findings/transactional.md) |
| TW-021 | api-email-order-status-delivered | transactional | technical-writer | P2 | Subject 'Your order has been delivered. Enjoy your meal!' compresses two though… | [findings/transactional.md](findings/transactional.md) |
| TW-022 | api-email-order-status-delivered | transactional | technical-writer | P2 | Three different 'enjoy' phrasings across surfaces: email 'Enjoy your meal!', pu… | [findings/transactional.md](findings/transactional.md) |
| TW-026 | api-email-chef-new-order | transactional | technical-writer | P2 | CTA 'View Order' is fine in length but title-case violates Sec 4 | [findings/transactional.md](findings/transactional.md) |
| TW-027 | api-email-chef-verified | transactional | technical-writer | P2 | Body uses 'Congratulations, %s!' — exclamation in transactional; H2 is also tit… | [findings/transactional.md](findings/transactional.md) |
| TW-028 | api-email-chef-verified | transactional | technical-writer | P2 | Bulleted next-steps list ends with exclamation 'Start accepting orders!' — thir… | [findings/transactional.md](findings/transactional.md) |
| TW-029 | api-email-chef-verified | transactional | technical-writer | P2 | CTA 'Go to Your Dashboard' is title-case, contains possessive, 4 words — violat… | [findings/transactional.md](findings/transactional.md) |
| TW-030 | api-email-chef-verified | transactional | technical-writer | P2 | Within-product naming drift: this email says 'kitchen', but the in-app notifica… | [findings/transactional.md](findings/transactional.md) |
| TW-032 | api-email-delivery-assigned | transactional | technical-writer | P2 | H2 '🚗 New Delivery!' uses emoji + exclamation in transactional driver email | [findings/transactional.md](findings/transactional.md) |
| TW-033 | api-email-delivery-assigned | transactional | technical-writer | P2 | CTA 'View Delivery Details' is title-case and 3 words — sentence-case violation | [findings/transactional.md](findings/transactional.md) |
| TW-035 | api-email-support-created | transactional | technical-writer | P2 | H2 'Support Ticket Created' uses title case; subject 'Support Ticket #%s — %s'… | [findings/transactional.md](findings/transactional.md) |
| TW-036 | api-email-support-created | transactional | technical-writer | P2 | 'Our team will get back to you shortly' is vague — sets no SLA and uses 'get ba… | [findings/transactional.md](findings/transactional.md) |
| TW-038 | api-email-support-update | transactional | technical-writer | P2 | Status field injected verbatim — if backend uses raw enums (e.g. 'in_progress',… | [findings/transactional.md](findings/transactional.md) |
| TW-039 | api-email-account-reminder | transactional | technical-writer | P2 | Inline HTML — no emailBase wrapper, so footer/privacy/terms missing. Same brand… | [findings/transactional.md](findings/transactional.md) |
| TW-041 | api-push-order-update-customer-deeplink | transactional | technical-writer | P2 | Driver-facing label 'picked up by the driver' uses banned customer-facing term… | [findings/transactional.md](findings/transactional.md) |
| TW-042 | api-push-order-update-customer-deeplink | transactional | technical-writer | P2 | 'delivering' status mapped to 'on the way to you' but other surfaces use 'on it… | [findings/transactional.md](findings/transactional.md) |
| TW-044 | api-push-order-update-customer | transactional | technical-writer | P2 | 'Your order is on its way!' label keeps an exclamation while sibling labels are… | [findings/transactional.md](findings/transactional.md) |
| TW-045 | api-inapp-order-status-customer | transactional | technical-writer | P2 | Title 'Order Status Updated' differs from push title 'Order Update' for the sam… | [findings/transactional.md](findings/transactional.md) |
| TW-046 | api-push-order-update-customer-deeplink | transactional | technical-writer | P2 | Title 'Order Update' is title-case; should be 'Order update' (sentence case rul… | [findings/transactional.md](findings/transactional.md) |
| TW-048 | api-push-chef-new-order-actionable | transactional | technical-writer | P2 | Push title 'New Order' is title-case (Sec 4 says sentence case for buttons; pus… | [findings/transactional.md](findings/transactional.md) |
| TW-050 | api-inapp-chef-new-order | transactional | technical-writer | P2 | Title 'New Order!' with exclamation; sibling in-app/push titles for same event… | [findings/transactional.md](findings/transactional.md) |
| TW-051 | api-inapp-order-created-chef | transactional | technical-writer | P2 | Within-event copy drift: DB row saved as 'You have received a new order!' but p… | [findings/transactional.md](findings/transactional.md) |
| TW-052 | api-inapp-order-delivered-customer | transactional | technical-writer | P2 | Customer-facing delivered title 'Order Delivered' is title-case + push body has… | [findings/transactional.md](findings/transactional.md) |
| TW-053 | api-push-order-delivered-customer | transactional | technical-writer | P2 | Same double-exclamation issue as TW-052 on the push variant | [findings/transactional.md](findings/transactional.md) |
| TW-054 | api-push-delivery-pickedup-customer | transactional | technical-writer | P2 | Title 'Order On The Way!' uses title case for prepositions ('On The Way' is wro… | [findings/transactional.md](findings/transactional.md) |
| TW-055 | api-inapp-delivery-pickedup-customer | transactional | technical-writer | P2 | Title 'Order Picked Up' (in-app) does not match push title 'Order On The Way!'… | [findings/transactional.md](findings/transactional.md) |
| TW-056 | api-push-delivery-assigned-customer | transactional | technical-writer | P2 | Body 'A delivery partner has been assigned to your order and will pick it up so… | [findings/transactional.md](findings/transactional.md) |
| TW-057 | api-inapp-delivery-assigned-customer | transactional | technical-writer | P2 | In-app body 'A delivery partner has been assigned to your order' is shorter tha… | [findings/transactional.md](findings/transactional.md) |
| TW-058 | api-push-delivery-assigned-driver | transactional | technical-writer | P2 | Driver push title 'New Delivery Available' is 3 words but title case + driver t… | [findings/transactional.md](findings/transactional.md) |
| TW-061 | api-inapp-chef-verified | transactional | technical-writer | P2 | Title 'Congratulations!' is bare emoji-replacement-style copy with exclamation… | [findings/transactional.md](findings/transactional.md) |
| TW-063 | api-inapp-driver-onboarding-admins | transactional | technical-writer | P2 | Message 'A new driver from %s has submitted their onboarding application for re… | [findings/transactional.md](findings/transactional.md) |
| TW-067 | api-push-approval-info-chef | transactional | technical-writer | P2 | Push body uses same admin notes verbatim — push notifications truncate at ~200… | [findings/transactional.md](findings/transactional.md) |
| TW-071 | api-success-message-generic | transactional | technical-writer | P2 | Inconsistent 'successfully' suffix — 16 of 43 messages end with 'successfully',… | [findings/transactional.md](findings/transactional.md) |
| TW-072 | api-success-message-generic | transactional | technical-writer | P2 | Logout message 'Logged out successfully' uses banned 'Log out' compound (per Se… | [findings/transactional.md](findings/transactional.md) |
| TW-079 | web-tx-admin-settings | transactional | technical-writer | P2 | 'Settings saved successfully' / 'Failed to save settings' — 'successfully' redu… | [findings/transactional.md](findings/transactional.md) |
| TW-080 | web-tx-admin-user-update | transactional | technical-writer | P2 | 'User updated successfully' — 'successfully' redundant; also 'User' is banned p… | [findings/transactional.md](findings/transactional.md) |
| TW-081 | web-tx-catering-quotes | transactional | technical-writer | P2 | 'Quote accepted! The chef has been notified.' — 8 words with exclamation in tra… | [findings/transactional.md](findings/transactional.md) |
| TW-082 | web-tx-catering-quotes | transactional | technical-writer | P2 | Error 'Failed to accept quote. Please try again.' uses 'Please try again' fille… | [findings/transactional.md](findings/transactional.md) |
| TW-083 | web-tx-catering-request-submit | transactional | technical-writer | P2 | 'Catering request submitted successfully!' has exclamation + 'successfully' + 5… | [findings/transactional.md](findings/transactional.md) |
| TW-084 | web-tx-chef-catering | transactional | technical-writer | P2 | 'Quote submitted successfully' / 'Failed to submit quote' — same 'successfully'… | [findings/transactional.md](findings/transactional.md) |
| TW-085 | web-tx-chef-menu | transactional | technical-writer | P2 | Three 'successfully' violations in the menu CRUD toast set; 'Menu item deleted'… | [findings/transactional.md](findings/transactional.md) |
| TW-087 | web-tx-chef-profile | transactional | technical-writer | P2 | 'Profile updated successfully' / 'Failed to update profile' — 'successfully' re… | [findings/transactional.md](findings/transactional.md) |
| TW-090 | web-tx-chefdetail-add-to-cart | transactional | technical-writer | P2 | 'Your cart has items from another chef. Clear cart first.' is 11 words across t… | [findings/transactional.md](findings/transactional.md) |
| TW-091 | web-tx-orderdetail-toasts | transactional | technical-writer | P2 | 'Payment verification failed — please contact support' uses em-dash + 'please'… | [findings/transactional.md](findings/transactional.md) |
| TW-092 | web-tx-orderdetail-toasts | transactional | technical-writer | P2 | 'Order cancelled successfully' redundancy; sibling 'Order number copied' (good)… | [findings/transactional.md](findings/transactional.md) |
| TW-093 | web-tx-profile-2fa | transactional | technical-writer | P2 | Mixed voice in 2FA toast set: 'Two-factor authentication enabled' (formal noun)… | [findings/transactional.md](findings/transactional.md) |
| TW-094 | web-tx-profile-password | transactional | technical-writer | P2 | Validation 'Passwords do not match' is good; 'Password must be at least 8 chara… | [findings/transactional.md](findings/transactional.md) |
| TW-095 | web-tx-profile-update | transactional | technical-writer | P2 | Avatar upload errors well-formatted ('File too large. Maximum 5 MB.') — but 'In… | [findings/transactional.md](findings/transactional.md) |
| TW-097 | web-tx-profile-addresses | transactional | technical-writer | P2 | 'Please fill in all required fields' uses 'please' filler; toast is not the rig… | [findings/transactional.md](findings/transactional.md) |
| TW-098 | web-tx-delivery-actions | transactional | technical-writer | P2 | Driver toasts violate Sec 2 driver tone (telegraphic ≤4 words): 'Delivery accep… | [findings/transactional.md](findings/transactional.md) |
| TW-099 | web-tx-onboarding-save | transactional | technical-writer | P2 | 'Your preferences have been saved!' uses possessive + exclamation; 8 words. Val… | [findings/transactional.md](findings/transactional.md) |
| TW-100 | web-tx-social-feed-actions | transactional | technical-writer | P2 | 'Please log in to like posts' uses banned 'log in' (Sec 3: 'sign in') + 'please… | [findings/transactional.md](findings/transactional.md) |
| TW-101 | vp-ux-menu-form-toasts | transactional | technical-writer | P2 | 'Menu item created successfully' / 'Menu item updated successfully' — 'successf… | [findings/transactional.md](findings/transactional.md) |
| TW-103 | vp-ux-menu-toasts | transactional | technical-writer | P2 | 'N items marked available/unavailable' uses slash for two states — readers will… | [findings/transactional.md](findings/transactional.md) |
| TW-107 | vp-onb-policies-ready-banner | transactional | technical-writer | P2 | 'You're all set to submit your application!' has exclamation in transactional b… | [findings/transactional.md](findings/transactional.md) |
| TW-110 | vp-ux-kitchen-photo-toasts | transactional | technical-writer | P2 | 9-toast set has internal drift: 'Kitchen photo uploaded' (past-tense good) / 'K… | [findings/transactional.md](findings/transactional.md) |
| TW-114 | vp-ux-settings-stripe-toasts | transactional | technical-writer | P2 | 'Failed to start Stripe onboarding' uses 'failed to' filler; 'Failed to resume… | [findings/transactional.md](findings/transactional.md) |
| TW-117 | dp-tx-toast-accept-success | transactional | technical-writer | P2 | 'Delivery accepted!' exclamation in driver toast — Rule 1 + driver tone is oper… | [findings/transactional.md](findings/transactional.md) |
| TW-118 | dp-tx-toast-partner-verified | transactional | technical-writer | P2 | 'Partner verified successfully' — 'successfully' redundancy on admin toast | [findings/transactional.md](findings/transactional.md) |
| TW-119 | dp-tx-toast-step5-submitted | transactional | technical-writer | P2 | 'Application submitted successfully!' — exclamation + successfully on driver on… | [findings/transactional.md](findings/transactional.md) |
| TW-121 | dp-tx-toast-invite-created | transactional | technical-writer | P2 | 'Invitation created successfully' — successfully redundancy in admin toast | [findings/transactional.md](findings/transactional.md) |
| TW-123 | ap-approvaldetail-toasts | transactional | technical-writer | P2 | 'More information requested from chef' — admin tone says direct ('Direct: Appro… | [findings/transactional.md](findings/transactional.md) |
| TW-124 | ap-providercreate-toasts | transactional | technical-writer | P2 | 'Provider created successfully' — successfully redundancy | [findings/transactional.md](findings/transactional.md) |
| TW-127 | ap-secsettings-toasts | transactional | technical-writer | P2 | Long mixed set: '2FA enabled successfully' (redundancy) / 'Session policy updat… | [findings/transactional.md](findings/transactional.md) |
| TW-128 | ap-settings-payment-feedback | transactional | technical-writer | P2 | 'Saved, but Razorpay rejected the keys: {err}' injects raw gateway error — admi… | [findings/transactional.md](findings/transactional.md) |
| TW-129 | ap-staffdetail-toasts | transactional | technical-writer | P2 | 'Role updated successfully' has redundancy; 'Staff member deactivated' uses 'me… | [findings/transactional.md](findings/transactional.md) |
| TW-130 | ap-staff-invite-success-title | transactional | technical-writer | P2 | 'Invitation created successfully!' — exclamation + successfully in admin succes… | [findings/transactional.md](findings/transactional.md) |
| TW-134 | mc-catering-submit-success | transactional | technical-writer | P2 | 'Request Submitted!' — exclamation + title case in transactional alert | [findings/transactional.md](findings/transactional.md) |
| TW-135 | mc-menuitem-cross-chef | transactional | technical-writer | P2 | Title 'Replace Cart?' is title case + question; modal body 'You have items from… | [findings/transactional.md](findings/transactional.md) |
| TW-136 | mv-onb-pending-submitted-title | transactional | technical-writer | P2 | 'Application Submitted!' uses title case + exclamation as a display title | [findings/transactional.md](findings/transactional.md) |
| TW-144 | md-trx-011 | transactional | technical-writer | P2 | 'Validation' as an alert title is meaningless to a driver — internal dev label… | [findings/transactional.md](findings/transactional.md) |
| TW-145 | md-trx-013 | transactional | technical-writer | P2 | 'Success' as alert title is generic and content-free; same problem as 'Validati… | [findings/transactional.md](findings/transactional.md) |
| TW-148 | md-trx-009 | transactional | technical-writer | P2 | 'Visit the web portal to manage your subscription.' is 8 words OK but cross-app… | [findings/transactional.md](findings/transactional.md) |
| TW-151 | md-trx-003 | transactional | technical-writer | P2 | 'Battery usage is minimised by only tracking every 15 seconds.' is 10 words but… | [findings/transactional.md](findings/transactional.md) |
| TW-152 | md-trx-004 | transactional | technical-writer | P2 | 'Allow Background Location' is title case 3 words — should be sentence case 'Al… | [findings/transactional.md](findings/transactional.md) |
| TW-155 | md-trx-018 | transactional | technical-writer | P2 | 'You do not have permission to invite staff.' is 8 words OK but 'You do not hav… | [findings/transactional.md](findings/transactional.md) |
| TW-158 | api-success-message-generic | transactional | technical-writer | P2 | Repository-wide pattern: 27 API success messages and 25+ frontend toasts spell… | [findings/transactional.md](findings/transactional.md) |
| TW-159 | api-success-message-generic | transactional | technical-writer | P2 | Repository-wide pattern: ~40+ instances of 'Failed to X. Please try again.' fro… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-046 | web-tx-chef-profile | transactional | business-analyst | P3 | 'Profile updated successfully' toast is the exact same string used by both chef… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-047 | web-tx-orderdetail-toasts | transactional | business-analyst | P3 | 'Payment confirmed' toast on OrderDetailPage could refer to the customer confir… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-048 | web-tx-profile-2fa | transactional | business-analyst | P3 | 'Two-factor authentication enabled' toast confirms a security upgrade with no f… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-049 | ap-staff-invite-success-title | transactional | business-analyst | P3 | 'Invitation created successfully!' uses exclamation on an admin operational sur… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-050 | vp-ux-settings-payout-toasts | transactional | business-analyst | P3 | 'Payout details saved' toast confirms bank detail storage but gives no indicati… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-051 | vp-ux-settings-stripe-toasts | transactional | business-analyst | P3 | 'Failed to start Stripe onboarding' and 'Failed to resume onboarding' are devel… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-052 | ap-approvaldetail-toasts | transactional | business-analyst | P3 | 'Approval request approved' and 'Approval request rejected' are passive-tense t… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-053 | ap-secsettings-toasts | transactional | business-analyst | P3 | 'All other sessions revoked. You may need to sign in again.' uses hedged langua… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-054 | mv-reviewdetail-success-title | transactional | business-analyst | P3 | Mobile vendor review reply success alert title 'Reply Sent' uses title case — s… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-055 | md-trx-018 | transactional | business-analyst | P3 | 'You do not have permission to invite staff.' surfaces in the driver app — this… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-056 | mc-payment-confirming | transactional | business-analyst | P3 | 'Confirming payment...' is the entire content of the Razorpay deep-link callbac… | [findings/transactional.md](findings/transactional.md) |
| BA-TX-057 | vp-onb-personal-avatar-success | transactional | business-analyst | P3 | 'Profile photo uploaded' vendor onboarding toast is correctly formatted (past t… | [findings/transactional.md](findings/transactional.md) |
| BV-035 | api-inapp-driver-onboarding-admins | transactional | brand-voice | P3 | 'A new driver from %s has submitted their onboarding application for review.' —… | [findings/transactional.md](findings/transactional.md) |
| BV-055 | mv-reviewdetail-success-title | transactional | brand-voice | P3 | 'Reply Sent' alert title — Title Case in chef-facing app. | [findings/transactional.md](findings/transactional.md) |
| BV-056 | mc-profile-save-success | transactional | brand-voice | P3 | Customer mobile: 'Saved / Profile updated successfully. / Cuisine preferences u… | [findings/transactional.md](findings/transactional.md) |
| BV-058 | web-tx-chef-order-status | transactional | brand-voice | P3 | Web chef portal: 'Order status updated' vs vendor-portal: 'Order status updated… | [findings/transactional.md](findings/transactional.md) |
| BV-060 | ap-staff-toasts | transactional | brand-voice | P3 | Sibling admin toasts oscillate: 'Invitation sent successfully' / 'Invitation re… | [findings/transactional.md](findings/transactional.md) |
| BV-063 | ap-platsettings-feedback | transactional | brand-voice | P3 | Sibling settings-save banners follow consistent past-tense shape ('Commission r… | [findings/transactional.md](findings/transactional.md) |
| BV-064 | vp-ux-menu-form-category-toasts | transactional | brand-voice | P3 | 'A category with this name already exists' is fine; sibling 'Category "{name}"… | [findings/transactional.md](findings/transactional.md) |
| BV-065 | dp-tx-toast-partner-suspend | transactional | brand-voice | P3 | Cross-app verb drift on a sensitive admin action: delivery-portal admin uses 'P… | [findings/transactional.md](findings/transactional.md) |
| BV-066 | web-tx-profile-update | transactional | brand-voice | P3 | Sibling toasts on the same profile page: 'Profile updated successfully' (with f… | [findings/transactional.md](findings/transactional.md) |
| BV-070 | vp-ux-kitchen-photo-toasts | transactional | brand-voice | P3 | 'Kitchen setup saved successfully' (with filler) alongside sibling 'Photo remov… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-026 | ap-staff-toasts | transactional | legal | P3 | 'Invitation revoked' toast is silent on whether prior actions taken by the invi… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-034 | ap-users-toast-activated | transactional | legal | P3 | 'User activated' toast — no record of who activated, when, or why; reversal of… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-038 | api-inapp-approval-approved-chef | transactional | legal | P3 | Approval notifications surface raw enum slugs ('Your menu_item has been approve… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-059 | vp-onb-submit-success | transactional | legal | P3 | 'Application submitted! We'll review and get back to you within 24-48 hours' —… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-073 | web-tx-chefdetail-add-to-cart | transactional | legal | P3 | 'Your cart has items from another chef. Clear cart first.' — operational messag… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-074 | mc-menuitem-cross-chef | transactional | legal | P3 | Mobile cross-chef cart replacement 'Replace Cart? / You have items from another… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-075 | web-tx-favorites-remove | transactional | legal | P3 | 'Removed {name} from favorites' — DPDP §12 erasure on user-initiated removal; e… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-083 | ap-providercreate-toasts | transactional | legal | P3 | 'Provider created successfully' admin toast — admin-only path, low exposure; fl… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-085 | ap-notifsettings-toasts | transactional | legal | P3 | Admin notification settings 'Preference saved' — operational, low risk; verify… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-089 | mc-profile-logout-confirm | transactional | legal | P3 | Mobile customer logout confirmation 'Log out / Are you sure you want to log out… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-090 | mc-profile-save-success | transactional | legal | P3 | 'Profile updated successfully. / Cuisine preferences updated.' — DPDP §11 right… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-091 | mv-onb-pending-submitted-body | transactional | legal | P3 | Mobile vendor pending screen 'Our team will review your application within 24-4… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-095 | web-tx-social-feed-actions | transactional | legal | P3 | 'Please log in to like posts / Please log in to save posts' — uses banned varia… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-101 | md-trx-011 | transactional | legal | P3 | Driver app validation alert title 'Validation' — non-descriptive title; offers… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-102 | md-trx-013 | transactional | legal | P3 | Driver app success alert title 'Success' — non-descriptive | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-103 | md-trx-006 | transactional | legal | P3 | Driver app 'Logout' confirmation uses banned variant — style guide requires 'Si… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-104 | md-trx-015 | transactional | legal | P3 | Generic failure body 'Failed to update profile. Please try again.' — leaks no i… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-112 | dp-tx-toast-step5-submitted | transactional | legal | P3 | 'Application submitted successfully!' — completion toast; verify a server-side… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-113 | md-trx-020 | transactional | legal | P3 | Shared OfflineBanner — copy lives in mobile-shared; cannot inspect content from… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-116 | mv-reviewdetail-success-title | transactional | legal | P3 | 'Reply Sent' title — chef reply published as platform content; same intermediar… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-118 | ap-staff-invite-success-title | transactional | legal | P3 | 'Invitation created successfully!' — exclamation mark in admin transactional su… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-120 | ap-providers-toasts | transactional | legal | P3 | 'Provider deleted' admin toast — deleting a provider may affect in-flight order… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-122 | vp-onb-personal-avatar-success | transactional | legal | P3 | 'Profile photo uploaded' — confirm EXIF stripping + storage location notice | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-123 | dp-tx-toast-step1-saved | transactional | legal | P3 | 'Personal info saved' — driver onboarding step; confirm DPDP §5 notice was show… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-124 | dp-tx-toast-step2-saved | transactional | legal | P3 | 'Vehicle details saved' — confirm Motor Vehicles Act compliance (DL number, RC,… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-139 | cross-cutting-transactional | transactional | legal | P3 | Cross-cutting: exclamation-mark inflation in transactional copy ('Welcome to Fe… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-145 | cross-cutting-transactional | transactional | legal | P3 | Cross-cutting: time/date formatting in transactional surfaces inconsistent — em… | [findings/transactional.md](findings/transactional.md) |
| LEG-TX-150 | cross-cutting-transactional | transactional | legal | P3 | Cross-cutting: 'Profile updated successfully' / 'Settings saved' etc. — DPDP §1… | [findings/transactional.md](findings/transactional.md) |
| TW-007 | api-email-verify | transactional | technical-writer | P3 | Helper sentence 'Thanks for signing up! Please verify your email address to act… | [findings/transactional.md](findings/transactional.md) |
| TW-011 | api-email-staff-invite | transactional | technical-writer | P3 | Role injected as raw string into body — if role enum is 'fleet_manager' user re… | [findings/transactional.md](findings/transactional.md) |
| TW-015 | api-email-order-confirm | transactional | technical-writer | P3 | Total label is plain 'Total' (correct per Sec 3) but column header 'Price' is a… | [findings/transactional.md](findings/transactional.md) |
| TW-049 | api-push-chef-new-order-actionable | transactional | technical-writer | P3 | Body 'You have a new order waiting for your confirmation' is 9 words — fine for… | [findings/transactional.md](findings/transactional.md) |
| TW-062 | api-email-chef-verified-publish | transactional | technical-writer | P3 | Shadowed email title 'Your Chef Profile is Verified!' surfaces only if template… | [findings/transactional.md](findings/transactional.md) |
| TW-070 | api-inapp-approval-created-admin | transactional | technical-writer | P3 | Approval title injected verbatim — if chef-authored title contains profanity or… | [findings/transactional.md](findings/transactional.md) |
| TW-073 | api-success-message-generic | transactional | technical-writer | P3 | 'If the email exists, a reset link has been sent' is a security-pattern string… | [findings/transactional.md](findings/transactional.md) |
| TW-074 | api-success-message-generic | transactional | technical-writer | P3 | Domain inconsistency: 'Chef suspended' (no successfully) / 'Delivery partner su… | [findings/transactional.md](findings/transactional.md) |
| TW-075 | api-success-message-generic | transactional | technical-writer | P3 | 'All notifications marked as read' duplicates singular 'Marked as read' and 'Me… | [findings/transactional.md](findings/transactional.md) |
| TW-077 | api-inapp-chef-responded | transactional | technical-writer | P3 | Title built by concatenation 'Chef Responded: ' + approval.Title — title case +… | [findings/transactional.md](findings/transactional.md) |
| TW-078 | api-inapp-staff-invite | transactional | technical-writer | P3 | Title and message are admin-authored strings with no length cap or format valid… | [findings/transactional.md](findings/transactional.md) |
| TW-086 | web-tx-chef-order-status | transactional | technical-writer | P3 | 'Order status updated' is fine, but error 'Failed to update order status' is wo… | [findings/transactional.md](findings/transactional.md) |
| TW-088 | web-tx-chef-social | transactional | technical-writer | P3 | Mixed tense and pattern: 'Post deleted' / 'Post created' / 'Post updated' (past… | [findings/transactional.md](findings/transactional.md) |
| TW-096 | web-tx-profile-preferences-save | transactional | technical-writer | P3 | Toast 'Preferences updated' is good; error 'Failed to save preferences' uses 'f… | [findings/transactional.md](findings/transactional.md) |
| TW-102 | vp-ux-menu-form-category-toasts | transactional | technical-writer | P3 | 'A category with this name already exists' is 8 words; 'A' opener is filler | [findings/transactional.md](findings/transactional.md) |
| TW-104 | vp-ux-notifs-respond-toasts | transactional | technical-writer | P3 | 'Please enter a response' is field-validation in a toast — should be inline. 'R… | [findings/transactional.md](findings/transactional.md) |
| TW-105 | vp-onb-personal-avatar-success | transactional | technical-writer | P3 | 'Profile photo uploaded' is fine — 3 words, past tense, period implied | [findings/transactional.md](findings/transactional.md) |
| TW-109 | vp-ux-orders-live-toasts | transactional | technical-writer | P3 | Same 'Failed to update order status' pattern as web-tx-chef-order-status — dupl… | [findings/transactional.md](findings/transactional.md) |
| TW-111 | vp-ux-profile-toasts | transactional | technical-writer | P3 | 'Profile photo updated' vs 'Profile updated successfully' — successfully suffix… | [findings/transactional.md](findings/transactional.md) |
| TW-112 | vp-ux-profile-doc-toasts | transactional | technical-writer | P3 | '{label} uploaded' is good; 'Upload failed' is bare — doesn't follow what-happe… | [findings/transactional.md](findings/transactional.md) |
| TW-113 | vp-ux-settings-payout-toasts | transactional | technical-writer | P3 | Same 'successfully' inconsistency: 'Payout details saved' vs 'Settings saved' —… | [findings/transactional.md](findings/transactional.md) |
| TW-115 | dp-tx-toast-online | transactional | technical-writer | P3 | 'You are now online' / 'You are now offline' — 4 words OK for driver (telegraph… | [findings/transactional.md](findings/transactional.md) |
| TW-116 | dp-tx-toast-status-updated | transactional | technical-writer | P3 | 'Status updated' is fine for driver — meets telegraphic rule | [findings/transactional.md](findings/transactional.md) |
| TW-120 | dp-tx-toast-step4-saved | transactional | technical-writer | P3 | 'Plan selected successfully' — 'successfully' redundancy | [findings/transactional.md](findings/transactional.md) |
| TW-122 | dp-tx-toast-invite-copied | transactional | technical-writer | P3 | 'Invite URL copied to clipboard' — 'to clipboard' is filler (always to clipboar… | [findings/transactional.md](findings/transactional.md) |
| TW-125 | ap-providerdetail-toasts | transactional | technical-writer | P3 | 'Connection successful ({ms}ms)' good with metric; 'Connection failed: {err}' r… | [findings/transactional.md](findings/transactional.md) |
| TW-126 | ap-notifsettings-toasts | transactional | technical-writer | P3 | 'Preference saved' / 'Failed to save' — error too generic (what preference?) | [findings/transactional.md](findings/transactional.md) |
| TW-131 | ap-staff-invite-success-body | transactional | technical-writer | P3 | 'Share this link with the invitee to complete their registration.' — 'invitee'… | [findings/transactional.md](findings/transactional.md) |
| TW-132 | mc-profile-save-success | transactional | technical-writer | P3 | Three near-identical save toasts in one component: 'Saved' / 'Profile updated s… | [findings/transactional.md](findings/transactional.md) |
| TW-138 | mv-onb-pending-rejected-title | transactional | technical-writer | P3 | 'Application Not Approved' is title case but the soft phrasing is good — alread… | [findings/transactional.md](findings/transactional.md) |
| TW-139 | mv-onb-pending-rejected-body | transactional | technical-writer | P3 | 'Please review the feedback and resubmit your application.' uses 'please' fille… | [findings/transactional.md](findings/transactional.md) |
| TW-140 | mv-reviewdetail-success-title | transactional | technical-writer | P3 | 'Reply Sent' — title case in alert title | [findings/transactional.md](findings/transactional.md) |
| TW-141 | mv-undo-accepted | transactional | technical-writer | P3 | 'Order accepted' / 'Order rejected' good 2-word snackbars — keep | [findings/transactional.md](findings/transactional.md) |
| TW-146 | md-trx-014 | transactional | technical-writer | P3 | 'Profile updated successfully.' — 'successfully' redundancy on driver alert body | [findings/transactional.md](findings/transactional.md) |
| TW-147 | md-trx-015 | transactional | technical-writer | P3 | 'Failed to update profile. Please try again.' — 'please' filler; matches the sa… | [findings/transactional.md](findings/transactional.md) |
| TW-149 | md-trx-010 | transactional | technical-writer | P3 | 'Contact support at support@homechef.in to request account deletion.' is 9 word… | [findings/transactional.md](findings/transactional.md) |
| TW-153 | md-trx-005 | transactional | technical-writer | P3 | 'Not Now' is title case — should be sentence case 'Not now' | [findings/transactional.md](findings/transactional.md) |
| TW-154 | md-trx-019 | transactional | technical-writer | P3 | 'Failed to send invitation. Please try again.' uses 'please' filler — same anti… | [findings/transactional.md](findings/transactional.md) |
| TW-156 | md-trx-016 | transactional | technical-writer | P3 | 'Failed to upload photo. Please try again.' — same 'please' filler pattern | [findings/transactional.md](findings/transactional.md) |
| TW-157 | md-trx-008 | transactional | technical-writer | P3 | Alert title 'Subscription' is bare noun — works as a context label but feels li… | [findings/transactional.md](findings/transactional.md) |

### Index — microcopy

311 findings. Detail: [findings/microcopy.md](findings/microcopy.md)

| finding_id | surface_id | category | lens | severity | issue (≤80 chars) | detail link |
|---|---|---|---|---|---|---|
| BA-001 | web-ux-cart-promo | microcopy | business-analyst | P0 | Hardcoded promo code hint exposes internal test code to all users | [findings/microcopy.md](findings/microcopy.md) |
| BA-002 | web-ux-cart-promo | microcopy | business-analyst | P0 | Promo success inline copy uses exclamation mark, violating brand voice rule | [findings/microcopy.md](findings/microcopy.md) |
| BA-003 | web-ux-cart-delivery-info | microcopy | business-analyst | P0 | Hardcoded prep time estimate ('30-45 mins') ships as static text regardless of… | [findings/microcopy.md](findings/microcopy.md) |
| BA-004 | web-ux-layout-auth-cta | microcopy | business-analyst | P0 | Header uses banned terms 'Login' and 'Sign Up' instead of 'Sign in' / 'Sign up' | [findings/microcopy.md](findings/microcopy.md) |
| BA-005 | dp-mc-portal-footer | microcopy | business-analyst | P0 | Delivery portal footer shows internal codename 'Fe3dr Delivery Portal' to drive… | [findings/microcopy.md](findings/microcopy.md) |
| BA-006 | dp-mc-portal-footer | microcopy | business-analyst | P0 | Delivery portal login subtitle uses banned 'Login' vocabulary | [findings/microcopy.md](findings/microcopy.md) |
| BV-001 | dp-mc-portal-footer | microcopy | brand-voice | P0 | Brand name 'Fe3dr' appears in delivery-portal footer; rest of platform is 'Home… | [findings/microcopy.md](findings/microcopy.md) |
| BV-002 | ap-auth-email-placeholder | microcopy | brand-voice | P0 | Admin login example email uses 'fe3dr.com' domain — brand contradiction | [findings/microcopy.md](findings/microcopy.md) |
| BV-003 | web-ux-layout-auth-cta | microcopy | brand-voice | P0 | Web header uses 'Login / Sign Up' — both forms violate style guide | [findings/microcopy.md](findings/microcopy.md) |
| BV-004 | dp-mc-logout | microcopy | brand-voice | P0 | Delivery-portal layout button says 'Logout' but settings page says 'Sign Out' —… | [findings/microcopy.md](findings/microcopy.md) |
| BV-005 | ap-layout-account-logout | microcopy | brand-voice | P0 | Admin layout uses 'Logout' — banned variant | [findings/microcopy.md](findings/microcopy.md) |
| BV-006 | md-mic-001 | microcopy | brand-voice | P0 | Mobile-delivery alert uses 'Logout' — banned variant; matches drift across deli… | [findings/microcopy.md](findings/microcopy.md) |
| BV-007 | web-ux-chef-card-favorite-toast-loggedout | microcopy | brand-voice | P0 | Toast says 'Please log in to save favorites' — 'log in' is the banned variant | [findings/microcopy.md](findings/microcopy.md) |
| BV-008 | dp-mc-role-partner | microcopy | brand-voice | P0 | Driver-facing UI labels the role 'Delivery Partner' — style guide says 'Driver'… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-005 | ap-chefs-action-suspend-aria | microcopy | legal | P0 | Suspend action exposed as aria/tooltip only — no consequence disclosure for che… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-006 | ap-users-action-suspend-aria | microcopy | legal | P0 | Suspend user — aria-label only with no consequence disclosure for affected user… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-009 | vp-ux-menu-form-dietary-tags | microcopy | legal | P0 | Dietary tags include religious certification claims (Halal, Kosher, Jain) with… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-010 | vp-ux-menu-form-fields | microcopy | legal | P0 | Allergens field is free-text with helper "Type each allergen and press Enter" —… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-014 | vp-ux-kitchen-payout-fields | microcopy | legal | P0 | Bank Name / Account Number / IFSC Code fields have no DPDP purpose/retention no… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-015 | vp-ux-settings-payout-form | microcopy | legal | P0 | Bank Transfer / UPI payout form — same DPDP notice gap, also no statement that… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-020 | mc-perm-camera | microcopy | legal | P0 | iOS NSCameraUsageDescription "Used to take photos" — vague purpose violates App… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-022 | mc-perm-location | microcopy | legal | P0 | iOS location permission "Used to show your location on the delivery tracking ma… | [findings/microcopy.md](findings/microcopy.md) |
| BA-007 | vp-onb-policies-payout-info | microcopy | business-analyst | P1 | Payout setup deferred to 'after approval' with no timeline, creating anxiety at… | [findings/microcopy.md](findings/microcopy.md) |
| BA-008 | vp-onb-nav-submit-app | microcopy | business-analyst | P1 | Final onboarding CTA is generic 'Submit Application' with no expectation-setting | [findings/microcopy.md](findings/microcopy.md) |
| BA-009 | vp-ux-menu-bulk-confirm | microcopy | business-analyst | P1 | Bulk delete uses browser native confirm() with vague 'cannot be undone' copy —… | [findings/microcopy.md](findings/microcopy.md) |
| BA-010 | vp-ux-menu-delete-confirm | microcopy | business-analyst | P1 | Single item delete uses browser confirm() with vague 'Are you sure?' copy | [findings/microcopy.md](findings/microcopy.md) |
| BA-011 | web-ux-chef-card-favorite-toast-loggedout | microcopy | business-analyst | P1 | Unauthenticated favourite toast is a dead end — 'Please log in' with no action… | [findings/microcopy.md](findings/microcopy.md) |
| BA-012 | vp-ux-orders-live-actions | microcopy | business-analyst | P1 | Chef action buttons mix tenses and formats: 'Accept/Reject' (imperative) vs 'Wa… | [findings/microcopy.md](findings/microcopy.md) |
| BA-013 | mc-perm-camera | microcopy | business-analyst | P1 | iOS camera permission string 'Used to take photos' is vague — does not explain… | [findings/microcopy.md](findings/microcopy.md) |
| BA-014 | dp-mc-stripe-cells | microcopy | business-analyst | P1 | Stripe status labels ('Charges', 'Payouts', 'Details', 'Submitted', 'Incomplete… | [findings/microcopy.md](findings/microcopy.md) |
| BA-015 | vp-ux-notifs-sla-line | microcopy | business-analyst | P1 | Kitchen review SLA is vague range ('24-48 hours') with no commitment or next-st… | [findings/microcopy.md](findings/microcopy.md) |
| BA-016 | mv-menuedit-price-change-banner | microcopy | business-analyst | P1 | Price change warning is long (14 words) for a mobile glanceable context and bur… | [findings/microcopy.md](findings/microcopy.md) |
| BA-017 | mc-checkout-address-placeholders | microcopy | business-analyst | P1 | Checkout address fields use placeholder text as labels (e.g. 'Address line 1 *'… | [findings/microcopy.md](findings/microcopy.md) |
| BV-009 | ap-chefs-card-unnamed | microcopy | brand-voice | P1 | Admin uses 'Unnamed Kitchen' as fallback — style guide naming convention is 'Ho… | [findings/microcopy.md](findings/microcopy.md) |
| BV-010 | vp-ux-menu-form-actions | microcopy | brand-voice | P1 | Menu-form buttons use Title Case ('Save Changes', 'Create Item') — style guide… | [findings/microcopy.md](findings/microcopy.md) |
| BV-011 | vp-onb-nav-review-app | microcopy | brand-voice | P1 | Onboarding nav button 'Review Application' is Title Case | [findings/microcopy.md](findings/microcopy.md) |
| BV-012 | vp-onb-nav-submit-app | microcopy | brand-voice | P1 | 'Submit Application' is Title Case | [findings/microcopy.md](findings/microcopy.md) |
| BV-014 | ap-approvaldetail-actions | microcopy | brand-voice | P1 | Admin action buttons in Title Case — 'Approve / Reject / Request Info' | [findings/microcopy.md](findings/microcopy.md) |
| BV-015 | ap-providerdetail-actions | microcopy | brand-voice | P1 | Provider-detail actions Title Cased: 'Test Connection', 'Edit Provider', 'Delet… | [findings/microcopy.md](findings/microcopy.md) |
| BV-016 | ap-providers-cta-add | microcopy | brand-voice | P1 | 'Add Provider' Title Cased | [findings/microcopy.md](findings/microcopy.md) |
| BV-017 | ap-staff-cta-invite | microcopy | brand-voice | P1 | 'Invite Staff' Title Cased | [findings/microcopy.md](findings/microcopy.md) |
| BV-018 | ap-delivery-cta-providers | microcopy | brand-voice | P1 | 'Manage Providers' Title Cased | [findings/microcopy.md](findings/microcopy.md) |
| BV-019 | ap-providercreate-submit | microcopy | brand-voice | P1 | Submit buttons Title Cased: 'Create Provider', 'Add Mapping' | [findings/microcopy.md](findings/microcopy.md) |
| BV-020 | ap-settings-payment-test-buttons | microcopy | brand-voice | P1 | Payment settings buttons Title Cased — 'Test Connection', 'Save Keys' | [findings/microcopy.md](findings/microcopy.md) |
| BV-022 | ap-staffdetail-actions | microcopy | brand-voice | P1 | Staff actions Title Cased: 'Change Role', 'Confirm Deactivation', 'Confirm Reac… | [findings/microcopy.md](findings/microcopy.md) |
| BV-028 | vp-ux-reviews-reply | microcopy | brand-voice | P1 | 'Post Reply' Title Cased button | [findings/microcopy.md](findings/microcopy.md) |
| BV-029 | vp-ux-notifs-respond-form | microcopy | brand-voice | P1 | 'Respond & Send' is Title Cased (and reads awkwardly — two verbs) | [findings/microcopy.md](findings/microcopy.md) |
| BV-030 | vp-ux-orders-live-actions | microcopy | brand-voice | P1 | Live-order actions Title Cased — 'Start Preparing', 'Mark Ready' | [findings/microcopy.md](findings/microcopy.md) |
| BV-032 | vp-ux-menu-form-section-titles | microcopy | brand-voice | P1 | Section titles Title Cased — 'Basic Information', 'Dietary Information', 'Prepa… | [findings/microcopy.md](findings/microcopy.md) |
| BV-033 | vp-ux-menu-form-fields | microcopy | brand-voice | P1 | All form labels Title Cased — 'Item Name', 'Prep Time', 'Portion Size', 'Dietar… | [findings/microcopy.md](findings/microcopy.md) |
| BV-034 | vp-onb-review-section-labels | microcopy | brand-voice | P1 | Onboarding review-step section titles Title Cased | [findings/microcopy.md](findings/microcopy.md) |
| BV-035 | vp-onb-review-field-labels | microcopy | brand-voice | P1 | Review-page field labels Title Cased — 'Full Name', 'Kitchen Type', 'Daily Capa… | [findings/microcopy.md](findings/microcopy.md) |
| BV-036 | vp-onb-review-policy-labels | microcopy | brand-voice | P1 | Policy labels Title Cased — 'Hygiene & Food Safety', 'Order & Cancellation Poli… | [findings/microcopy.md](findings/microcopy.md) |
| BV-038 | vp-onb-docs-additional-title | microcopy | brand-voice | P1 | Section title 'Additional Documents (Optional)' Title Cased; '(Optional)' uses… | [findings/microcopy.md](findings/microcopy.md) |
| BV-041 | vp-ux-analytics-charts | microcopy | brand-voice | P1 | Chart headers Title Cased — 'Order Trends', 'Revenue Trends', 'Popular Items',… | [findings/microcopy.md](findings/microcopy.md) |
| BV-042 | vp-ux-earnings-sections | microcopy | brand-voice | P1 | Section headers Title Cased — 'Daily Earnings', 'Top Selling Items', 'Recent Pa… | [findings/microcopy.md](findings/microcopy.md) |
| BV-043 | vp-ux-notifs-section-titles | microcopy | brand-voice | P1 | 'Kitchen Review Status' Title Cased | [findings/microcopy.md](findings/microcopy.md) |
| BV-046 | vp-ux-notifs-status-labels | microcopy | brand-voice | P1 | Status labels Title Cased — 'Pending Review', 'Info Requested' | [findings/microcopy.md](findings/microcopy.md) |
| BV-047 | vp-ux-notifs-actions | microcopy | brand-voice | P1 | Action chips Title Cased — 'Update Profile', 'Update Details', 'Go to Dashboard' | [findings/microcopy.md](findings/microcopy.md) |
| BV-050 | web-ux-cart-promo | microcopy | brand-voice | P1 | Promo toast 'Promo code applied! 10% off' uses exclamation mark — violates Rule… | [findings/microcopy.md](findings/microcopy.md) |
| BV-051 | vp-ux-reviews-reply | microcopy | brand-voice | P1 | Success toast 'Reply posted successfully' uses adverb 'successfully' — bloat pe… | [findings/microcopy.md](findings/microcopy.md) |
| BV-052 | vp-ux-menu-delete-confirm | microcopy | brand-voice | P1 | Confirm copy 'Are you sure you want to delete this item?' is the anti-pattern e… | [findings/microcopy.md](findings/microcopy.md) |
| BV-053 | vp-ux-menu-bulk-confirm | microcopy | brand-voice | P1 | Bulk-delete confirm 'Delete {n} items? This cannot be undone.' — first sentence… | [findings/microcopy.md](findings/microcopy.md) |
| BV-054 | mv-menuedit-price-change-banner | microcopy | brand-voice | P1 | Mobile-vendor banner runs 14 words for an in-motion mobile context: 'Price chan… | [findings/microcopy.md](findings/microcopy.md) |
| BV-056 | mc-perm-camera | microcopy | brand-voice | P1 | iOS camera permission prompt 'Used to take photos' is vague and operational — s… | [findings/microcopy.md](findings/microcopy.md) |
| BV-060 | vp-ux-orderstatus-badges | microcopy | brand-voice | P1 | Order status 'Picked Up' uses past participle Title Case; pair surface uses 'Wa… | [findings/microcopy.md](findings/microcopy.md) |
| BV-079 | mc-order-card-meta | microcopy | brand-voice | P1 | Mobile-customer order-card meta uses '{n} item(s)' — banned plural pattern; sty… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-001 | web-ux-chef-card-verified | microcopy | legal | P1 | "Verified chef" badge — unqualified verification claim with no tooltip/disclosu… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-002 | ap-chefs-badge-verified | microcopy | legal | P1 | Admin "Verified" badge — same unqualified claim surfaced on admin chefs list wi… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-003 | ap-chefs-action-verify-aria | microcopy | legal | P1 | Single-click "Verify kitchen" action with no on-screen attestation of what admi… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-004 | ap-chefs-action-reject-aria | microcopy | legal | P1 | Single-click "Reject application" with no required reason code — chef cannot di… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-007 | ap-staffdetail-actions | microcopy | legal | P1 | Deactivate/Reactivate staff buttons — no audit/reason capture surfaced in UI la… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-008 | ap-approvaldetail-actions | microcopy | legal | P1 | Approve/Reject/Request Info buttons — notes field marked "(optional)" but rejec… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-011 | vp-ux-menu-form-allergen-help | microcopy | legal | P1 | Allergen helper text gives no warning about legal duty to declare; chefs may sk… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-012 | web-ux-checkout-instructions | microcopy | legal | P1 | Customer "Special Instructions" textarea — no allergen-aware prompt; customer c… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-013 | mc-checkout-note | microcopy | legal | P1 | Mobile checkout "Note to chef" — same gap as web; no allergy-specific field | [findings/microcopy.md](findings/microcopy.md) |
| LEG-016 | vp-onb-policies-payout-info | microcopy | legal | P1 | Deferred payout setup explanation does not mention KYC requirement, expected ti… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-017 | dp-mc-stripe-cells | microcopy | legal | P1 | Driver Stripe "Charges / Payouts / Details / Enabled / Pending / Submitted / In… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-018 | dp-mc-action-required | microcopy | legal | P1 | "Action Required" payout badge — does not state what action, what regulatory re… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-021 | mc-perm-faceid | microcopy | legal | P1 | iOS NSFaceIDUsageDescription "Use Face ID to log in quickly" — does not state b… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-023 | vp-onb-personal-address-fields | microcopy | legal | P1 | Address fields (Address Line 1/2, Landmark, Country, State, City, PIN) collecte… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-025 | vp-onb-docs-kitchen-photos-title | microcopy | legal | P1 | Kitchen photos collected for "trust" without stating who can see them, retentio… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-026 | vp-onb-docs-additional-title | microcopy | legal | P1 | "Additional Documents (Optional) … help speed up verification and build custome… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-030 | web-ux-cart-promo | microcopy | legal | P1 | Promo code UI "Promo code applied! 10% off / Try: FE3DR10 for 10% off" — no lin… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-033 | ap-secsettings-session-actions | microcopy | legal | P1 | "Sign out everywhere / Revoke session" actions without confirmation copy explai… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-035 | ap-exports-button | microcopy | legal | P1 | Data Exports "Download CSV; Preparing..." — no surface disclosure of what expor… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-037 | mv-menuedit-price-change-banner | microcopy | legal | P1 | Price-change banner "Price changes are submitted for admin review and may take… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-043 | web-ux-orderdetail-cancel-reason | microcopy | legal | P1 | "Cancellation reason:" label — no surface statement of refund timeline or who p… | [findings/microcopy.md](findings/microcopy.md) |
| TW-017 | PATTERN-permission-strings-vague | microcopy | technical-writer | P1 | iOS permission strings are vague — 'Used to take photos' fails App Store review… | [findings/microcopy.md](findings/microcopy.md) |
| BA-018 | web-ux-checkout-instructions | microcopy | business-analyst | P2 | Special Instructions field has no 'why we ask' helper text — a missed upsell an… | [findings/microcopy.md](findings/microcopy.md) |
| BA-019 | web-ux-orders-search | microcopy | business-analyst | P2 | Order search placeholder 'Search orders...' misses opportunity to prompt reorde… | [findings/microcopy.md](findings/microcopy.md) |
| BA-020 | vp-onb-ops-delivery-fee | microcopy | business-analyst | P2 | Delivery fee field has no context about who sets the fee or who pays it | [findings/microcopy.md](findings/microcopy.md) |
| BA-021 | vp-onb-ops-min-order | microcopy | business-analyst | P2 | Minimum order field hint 'Set to 0 for no minimum' implies ₹0 but the currency… | [findings/microcopy.md](findings/microcopy.md) |
| BA-022 | vp-ux-reviews-reply | microcopy | business-analyst | P2 | Reply success toast 'Reply posted successfully' is verbose and passive; violate… | [findings/microcopy.md](findings/microcopy.md) |
| BA-023 | vp-ux-profile-toggle | microcopy | business-analyst | P2 | 'Orders Paused' toggle state gives no indication of customer impact or how to r… | [findings/microcopy.md](findings/microcopy.md) |
| BA-024 | vp-ux-earnings-sections | microcopy | business-analyst | P2 | 'No payouts yet' empty state has no next-step instruction to set up payout meth… | [findings/microcopy.md](findings/microcopy.md) |
| BA-025 | vp-ux-dashboard-stats-labels | microcopy | business-analyst | P2 | 'No reviews yet' is a missed activation prompt for new chefs — tells nothing ab… | [findings/microcopy.md](findings/microcopy.md) |
| BA-026 | mc-order-detail-eta | microcopy | business-analyst | P2 | 'ETA:' prefix is inconsistent with 'Est. arrival:' used in the OrderTimeline co… | [findings/microcopy.md](findings/microcopy.md) |
| BA-027 | mc-order-card-meta | microcopy | business-analyst | P2 | 'item(s)' plural pattern violates Style Guide — must never use (s) construction | [findings/microcopy.md](findings/microcopy.md) |
| BA-028 | vp-ux-menu-form-allergen-help | microcopy | business-analyst | P2 | Allergen helper text is purely mechanical ('Type each allergen and press Enter'… | [findings/microcopy.md](findings/microcopy.md) |
| BA-029 | vp-onb-kitchen-desc-field | microcopy | business-analyst | P2 | Kitchen description helper references 'your signature touch' — banned near-arti… | [findings/microcopy.md](findings/microcopy.md) |
| BA-030 | vp-onb-docs-kitchen-photo-items | microcopy | business-analyst | P2 | Kitchen photo titles (Cooking Area / Preparation Area / Storage / Packaging) ha… | [findings/microcopy.md](findings/microcopy.md) |
| BA-031 | dp-mc-action-required | microcopy | business-analyst | P2 | 'Action Required' badge in driver settings gives no indication of what action i… | [findings/microcopy.md](findings/microcopy.md) |
| BA-032 | vp-ux-notifs-respond-form | microcopy | business-analyst | P2 | Reply form placeholder contains FSSAI example — good intent but presumes the on… | [findings/microcopy.md](findings/microcopy.md) |
| BA-033 | vp-auth-login-password-placeholder | microcopy | business-analyst | P2 | Password placeholder 'Enter your password' is redundant instruction — adds no v… | [findings/microcopy.md](findings/microcopy.md) |
| BA-034 | vp-auth-register-confirm-placeholder | microcopy | business-analyst | P2 | 'Re-enter password' placeholder is instructional copy that belongs in the label… | [findings/microcopy.md](findings/microcopy.md) |
| BA-035 | vp-ux-orders-live-note | microcopy | business-analyst | P2 | 'Note: {specialInstructions}' prefix is a raw colon label — not styled as a cal… | [findings/microcopy.md](findings/microcopy.md) |
| BA-036 | mc-catering-placeholders | microcopy | business-analyst | P2 | Catering budget placeholder '25000' is bare number with no currency symbol — am… | [findings/microcopy.md](findings/microcopy.md) |
| BV-055 | dp-mc-available-deliveries-alabel | microcopy | brand-voice | P2 | Driver bell aria 'Available deliveries, {n} new' is fine; bare 'Available deliv… | [findings/microcopy.md](findings/microcopy.md) |
| BV-057 | mc-perm-faceid | microcopy | brand-voice | P2 | FaceID prompt 'Use Face ID to log in quickly' contains 'log in' — banned variant | [findings/microcopy.md](findings/microcopy.md) |
| BV-058 | mc-perm-location | microcopy | brand-voice | P2 | Location prompt 'Used to show your location on the delivery tracking map' start… | [findings/microcopy.md](findings/microcopy.md) |
| BV-059 | web-ux-loading-default | microcopy | brand-voice | P2 | 'Loading...' uses three dots; style guide implies terminal period for finished… | [findings/microcopy.md](findings/microcopy.md) |
| BV-061 | vp-onb-kitchen-desc-field | microcopy | brand-voice | P2 | Placeholder 'Describe your cooking style, what makes your food special, your si… | [findings/microcopy.md](findings/microcopy.md) |
| BV-063 | vp-onb-docs-kitchen-photos-title | microcopy | brand-voice | P2 | Helper text 'Photos of your kitchen help build trust' is fine, but 'Kitchen Pho… | [findings/microcopy.md](findings/microcopy.md) |
| BV-067 | mv-dash-greeting-fallback | microcopy | brand-voice | P2 | Fallback 'Chef' when no name is available — depersonalised. Better: omit greeti… | [findings/microcopy.md](findings/microcopy.md) |
| BV-073 | mc-tabs-labels | microcopy | brand-voice | P2 | Mobile-customer tab labels 'Home / Orders / Saved / Profile'; web equivalent us… | [findings/microcopy.md](findings/microcopy.md) |
| BV-075 | md-mic-003 | microcopy | brand-voice | P2 | Android notification channel 'New Deliveries' is Title Cased — these strings ap… | [findings/microcopy.md](findings/microcopy.md) |
| BV-076 | mv-push-channel-neworders | microcopy | brand-voice | P2 | 'New Orders' Title Cased Android channel label | [findings/microcopy.md](findings/microcopy.md) |
| BV-077 | mv-push-channel-orderupdates | microcopy | brand-voice | P2 | 'Order Updates' Title Cased Android channel label | [findings/microcopy.md](findings/microcopy.md) |
| BV-078 | md-mic-004 | microcopy | brand-voice | P2 | 'Delivery Updates' Title Cased | [findings/microcopy.md](findings/microcopy.md) |
| BV-081 | ap-settings-payment-statuses | microcopy | brand-voice | P2 | 'Live Mode / Test Mode' Title Cased | [findings/microcopy.md](findings/microcopy.md) |
| LEG-019 | dp-mc-connected | microcopy | legal | P2 | "Connected" status (Stripe) — no disclosure that connecting to a foreign paymen… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-024 | vp-onb-personal-avatar-label | microcopy | legal | P2 | Profile photo helper "Shown on your kitchen page" — does not disclose photo ret… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-027 | vp-onb-docs-upload-helper | microcopy | legal | P2 | Document upload helper "Drop file here or click to browse. Max 5 MB." — no stat… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-029 | mc-catering-placeholders | microcopy | legal | P2 | Catering request placeholders include city/state/budget/dietary text — no DPDP… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-031 | vp-ux-menu-bulk-confirm | microcopy | legal | P2 | Bulk delete confirm "Delete {n} items? This cannot be undone." — vague conseque… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-032 | vp-ux-menu-delete-confirm | microcopy | legal | P2 | Generic "Are you sure you want to delete this item?" — vague consequence patter… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-034 | ap-secsettings-apikey-meta | microcopy | legal | P2 | API key metadata "no scopes; expires {date}; revoked" — minimal disclosure of w… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-036 | ap-auditlogs-expanded | microcopy | legal | P2 | Audit log expanded view "User agent: / Before / After" — no statement of retent… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-038 | dp-mc-uploaded-state | microcopy | legal | P2 | Driver document state shows "Verified / Uploaded" without distinction of who ve… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-039 | dp-mc-pending-verification | microcopy | legal | P2 | "Pending Verification" badge — no SLA, no statement of what cannot be done whil… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-040 | vp-ux-notifs-respond-form | microcopy | legal | P2 | Notification response placeholder example "I've uploaded the FSSAI license, ple… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-044 | vp-ux-orders-history-cancel-reason | microcopy | legal | P2 | Chef-side "Reason: {cancelReason}" — display only, no statement of chef-vs-plat… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-045 | vp-ux-orders-live-actions | microcopy | legal | P2 | "Reject" order button has no captured-reason requirement at UI label level — re… | [findings/microcopy.md](findings/microcopy.md) |
| TW-001 | PATTERN-vendor-portal-title-case-buttons | microcopy | technical-writer | P2 | Vendor-portal buttons use Title Case across the app — style guide §4 requires s… | [findings/microcopy.md](findings/microcopy.md) |
| TW-002 | PATTERN-vendor-portal-title-case-labels | microcopy | technical-writer | P2 | Vendor-portal form section headings and field labels use Title Case throughout… | [findings/microcopy.md](findings/microcopy.md) |
| TW-003 | PATTERN-vendor-portal-title-case-status-badges | microcopy | technical-writer | P2 | Vendor-portal status badges and chips use Title Case — sentence case applies to… | [findings/microcopy.md](findings/microcopy.md) |
| TW-004 | PATTERN-admin-portal-title-case-buttons | microcopy | technical-writer | P2 | Admin-portal action buttons use Title Case across the app — style guide §4 requ… | [findings/microcopy.md](findings/microcopy.md) |
| TW-005 | PATTERN-admin-portal-title-case-filters-tooltips | microcopy | technical-writer | P2 | Admin filters, dropdown options and tooltips use Title Case — should be sentenc… | [findings/microcopy.md](findings/microcopy.md) |
| TW-006 | PATTERN-vendor-portal-section-headings-title-case | microcopy | technical-writer | P2 | Vendor-portal page section headings use Title Case — sentence case applies ever… | [findings/microcopy.md](findings/microcopy.md) |
| TW-007 | PATTERN-banned-vocab-logout | microcopy | technical-writer | P2 | Multiple surfaces use banned term 'Logout' / 'Login' — style guide §3 mandates… | [findings/microcopy.md](findings/microcopy.md) |
| TW-008 | PATTERN-banned-vocab-sign-up-vs-signup | microcopy | technical-writer | P2 | Header CTA renders 'Sign Up' (Title Case) — style guide allows 'Sign up' (verb,… | [findings/microcopy.md](findings/microcopy.md) |
| TW-010 | PATTERN-confirm-anti-vague | microcopy | technical-writer | P2 | Delete/bulk-delete confirms use the banned vague 'Are you sure?' anti-pattern —… | [findings/microcopy.md](findings/microcopy.md) |
| TW-015 | PATTERN-currency-placeholder-format | microcopy | technical-writer | P2 | Catering form ships raw numeric placeholder '25000' for budget — style guide §6… | [findings/microcopy.md](findings/microcopy.md) |
| TW-016 | PATTERN-required-asterisk-in-placeholder | microcopy | technical-writer | P2 | Required indicator merged into placeholder ('Address line 1 *') — style guide §… | [findings/microcopy.md](findings/microcopy.md) |
| TW-020 | PATTERN-driver-persona-too-long | microcopy | technical-writer | P2 | Driver-facing aria labels exceed telegraphic ≤4-word target per persona matrix | [findings/microcopy.md](findings/microcopy.md) |
| TW-023 | web-ux-cart-promo | microcopy | technical-writer | P2 | Toast 'Promo code applied!' breaks Rule 1 (no exclamations except genuine celeb… | [findings/microcopy.md](findings/microcopy.md) |
| TW-025 | web-ux-checkout-instructions | microcopy | technical-writer | P2 | Label 'Special Instructions' uses Title Case; placeholder 'Any special requests… | [findings/microcopy.md](findings/microcopy.md) |
| TW-027 | web-ux-chef-card-favorite-toast-loggedout | microcopy | technical-writer | P2 | Toast uses banned 'log in' verb (style guide §3: 'Sign in') and exceeds toast f… | [findings/microcopy.md](findings/microcopy.md) |
| TW-036 | vp-ux-menu-form-fields | microcopy | technical-writer | P2 | Placeholder 'e.g. Paneer Butter Masala' good but '250g, 1 plate' for portion si… | [findings/microcopy.md](findings/microcopy.md) |
| TW-054 | vp-ux-orders-live-actions | microcopy | technical-writer | P2 | 'Waiting for pickup' is status not action but lives among action buttons; 'Star… | [findings/microcopy.md](findings/microcopy.md) |
| TW-061 | vp-ux-reviews-reply | microcopy | technical-writer | P2 | Success toast 'Reply posted successfully' breaks toast formula (4-word past ten… | [findings/microcopy.md](findings/microcopy.md) |
| TW-071 | dp-mc-try-again | microcopy | technical-writer | P2 | Error-boundary CTA 'Try Again' uses Title Case — should be sentence case | [findings/microcopy.md](findings/microcopy.md) |
| TW-084 | ap-secsettings-session-actions | microcopy | technical-writer | P2 | 'Sign out everywhere' uses 'sign out' (correct vocab) but admin app elsewhere s… | [findings/microcopy.md](findings/microcopy.md) |
| TW-107 | mv-menuedit-price-change-banner | microcopy | technical-writer | P2 | Inline warning banner 'Price changes are submitted for admin review and may tak… | [findings/microcopy.md](findings/microcopy.md) |
| TW-109 | md-mic-003 | microcopy | technical-writer | P2 | Driver Android notification channel 'New Deliveries' marked P0 SAFETY in invent… | [findings/microcopy.md](findings/microcopy.md) |
| TW-116 | ap-providercreate-submit | microcopy | technical-writer | P2 | Button set 'Create Provider; Cancel; Add Mapping; Add' uses Title Case (covered… | [findings/microcopy.md](findings/microcopy.md) |
| BA-037 | ap-layout-account-logout | microcopy | business-analyst | P3 | 'Logout' in admin layout uses banned vocabulary — should be 'Sign out' | [findings/microcopy.md](findings/microcopy.md) |
| BA-038 | ap-approvaldetail-yesno | microcopy | business-analyst | P3 | 'Yes / No' boolean rendering in approval detail is ambiguous without question c… | [findings/microcopy.md](findings/microcopy.md) |
| BA-039 | ap-providercreate-helpers | microcopy | business-analyst | P3 | Helper 'No status mappings configured. Add mappings to translate provider statu… | [findings/microcopy.md](findings/microcopy.md) |
| BA-040 | vp-ux-payouts-table | microcopy | business-analyst | P3 | Payout table header 'Method' is too terse — ambiguous between payment method (R… | [findings/microcopy.md](findings/microcopy.md) |
| BA-041 | ap-approvals-priorities | microcopy | business-analyst | P3 | 'urgent; high; normal; low' priority badges are lowercase — inconsistent with s… | [findings/microcopy.md](findings/microcopy.md) |
| BA-042 | vp-ux-loading-screen | microcopy | business-analyst | P3 | 'Loading...' generic text on full-page loader gives no context — same string us… | [findings/microcopy.md](findings/microcopy.md) |
| BA-043 | vp-onb-kitchen-specialties-title | microcopy | business-analyst | P3 | Specialties section marked '(optional)' in the title — optional markers belong… | [findings/microcopy.md](findings/microcopy.md) |
| BA-044 | vp-onb-personal-avatar-label | microcopy | business-analyst | P3 | Avatar field label parenthetical '(Optional)' embedded in label text, not separ… | [findings/microcopy.md](findings/microcopy.md) |
| BA-045 | mv-settings-accepting-helper | microcopy | business-analyst | P3 | Toggle helper text 'Toggle to start or pause accepting orders' describes the UI… | [findings/microcopy.md](findings/microcopy.md) |
| BA-046 | ap-auditlogs-filter-placeholders | microcopy | business-analyst | P3 | Audit log filter placeholder 'e.g. chef.verify' uses dot-notation event codes u… | [findings/microcopy.md](findings/microcopy.md) |
| BV-013 | vp-onb-mobile-review-title | microcopy | brand-voice | P3 | 'Review & Submit' is Title Case in step indicator | [findings/microcopy.md](findings/microcopy.md) |
| BV-021 | ap-settings-card-ctas | microcopy | brand-voice | P3 | Single-word CTAs 'Manage; Download; View' look fine, but adjacent surfaces use… | [findings/microcopy.md](findings/microcopy.md) |
| BV-023 | ap-staffdetail-current-role-label | microcopy | brand-voice | P3 | 'Current Role' Title Cased label | [findings/microcopy.md](findings/microcopy.md) |
| BV-024 | ap-staffdetail-back | microcopy | brand-voice | P3 | 'Back to Staff' Title Cased nav button | [findings/microcopy.md](findings/microcopy.md) |
| BV-025 | ap-providerdetail-back | microcopy | brand-voice | P3 | 'Back to Providers' Title Cased | [findings/microcopy.md](findings/microcopy.md) |
| BV-026 | ap-userdetail-back | microcopy | brand-voice | P3 | 'Back to Users' Title Cased | [findings/microcopy.md](findings/microcopy.md) |
| BV-027 | ap-approvaldetail-back | microcopy | brand-voice | P3 | 'Back to Reviews' Title Cased | [findings/microcopy.md](findings/microcopy.md) |
| BV-031 | vp-ux-orderstatus-badges | microcopy | brand-voice | P3 | Order-status badges use Title Case ('Picked Up') — acceptable as status labels,… | [findings/microcopy.md](findings/microcopy.md) |
| BV-037 | vp-ux-payouts-table | microcopy | brand-voice | P3 | Table headers Title Cased — 'Date / Amount / Status / Method' | [findings/microcopy.md](findings/microcopy.md) |
| BV-039 | vp-onb-personal-avatar-label | microcopy | brand-voice | P3 | 'Profile Photo (Optional)' Title Cased | [findings/microcopy.md](findings/microcopy.md) |
| BV-040 | vp-onb-ops-min-order | microcopy | brand-voice | P3 | 'Minimum Order Value (Optional)' Title Cased | [findings/microcopy.md](findings/microcopy.md) |
| BV-044 | vp-ux-notifs-admin-notes | microcopy | brand-voice | P3 | 'Admin Notes' label Title Cased | [findings/microcopy.md](findings/microcopy.md) |
| BV-045 | vp-onb-banner-admin-notes-label | microcopy | brand-voice | P3 | 'Admin Notes:' uses Title Case AND trailing colon — both violate style guide (§… | [findings/microcopy.md](findings/microcopy.md) |
| BV-048 | dp-mc-pending-verification | microcopy | brand-voice | P3 | 'Pending Verification' Title Cased status | [findings/microcopy.md](findings/microcopy.md) |
| BV-049 | dp-mc-action-required | microcopy | brand-voice | P3 | 'Action Required' Title Cased status | [findings/microcopy.md](findings/microcopy.md) |
| BV-062 | vp-onb-kitchen-specialties-title | microcopy | brand-voice | P3 | Section title 'Signature Dishes & Specialties' Title Cased and 'Signature' bord… | [findings/microcopy.md](findings/microcopy.md) |
| BV-064 | vp-onb-policies-payout-info | microcopy | brand-voice | P3 | Helper text correctly uses 'You can…' — flagged as POSITIVE model. But pair sur… | [findings/microcopy.md](findings/microcopy.md) |
| BV-065 | ap-platsettings-hours-defaults | microcopy | brand-voice | P3 | Admin helper 'We're currently closed.' uses 'we' — but in admin context this is… | [findings/microcopy.md](findings/microcopy.md) |
| BV-066 | mv-dash-greeting-morning | microcopy | brand-voice | P3 | Mobile-vendor uses time-based greeting 'Good morning' / 'Good afternoon' / 'Goo… | [findings/microcopy.md](findings/microcopy.md) |
| BV-068 | vp-auth-login-email-placeholder | microcopy | brand-voice | P3 | Vendor-portal uses generic 'you@example.com'; admin-portal uses brand-specific… | [findings/microcopy.md](findings/microcopy.md) |
| BV-069 | vp-onb-kitchen-name-field | microcopy | brand-voice | P3 | Placeholder uses 'Meena's Kitchen, Amma's Tiffin Service' — culturally appropri… | [findings/microcopy.md](findings/microcopy.md) |
| BV-070 | ap-staff-invite-placeholders | microcopy | brand-voice | P3 | Placeholders mix forms: 'colleague@company.com', 'e.g. Operations', 'Optional m… | [findings/microcopy.md](findings/microcopy.md) |
| BV-071 | vp-ux-menu-search-placeholder | microcopy | brand-voice | P3 | Placeholder 'Search menu items...' uses ASCII three-dot; rest of style guide im… | [findings/microcopy.md](findings/microcopy.md) |
| BV-072 | web-ux-social-feed-comment | microcopy | brand-voice | P3 | 'Add a comment...' uses ASCII three-dot | [findings/microcopy.md](findings/microcopy.md) |
| BV-074 | vp-onb-stepper-review-step | microcopy | brand-voice | P3 | Stepper label 'Review / Submit application' inconsistent with mobile-view title… | [findings/microcopy.md](findings/microcopy.md) |
| BV-080 | dp-mc-stripe-cells | microcopy | brand-voice | P3 | Stripe state labels mix 'Enabled / Pending / Submitted / Incomplete' (sentence… | [findings/microcopy.md](findings/microcopy.md) |
| BV-082 | web-ux-orderdetail-cancel-reason | microcopy | brand-voice | P3 | 'Cancellation reason:' uses trailing colon — style guide §4 form labels: 'no co… | [findings/microcopy.md](findings/microcopy.md) |
| BV-083 | vp-ux-orders-live-note | microcopy | brand-voice | P3 | 'Note: {specialInstructions}' has colon — acceptable when label runs inline wit… | [findings/microcopy.md](findings/microcopy.md) |
| BV-084 | ap-auditlogs-expanded | microcopy | brand-voice | P3 | 'User agent:' uses colon + inline label; 'Before' / 'After' are bare labels — i… | [findings/microcopy.md](findings/microcopy.md) |
| BV-085 | ap-secsettings-session-actions | microcopy | brand-voice | P3 | 'Sign out everywhere' — uses 'Sign out' correctly. Flagged as POSITIVE model. P… | [findings/microcopy.md](findings/microcopy.md) |
| BV-086 | vp-ux-menu-card-aria | microcopy | brand-voice | P3 | aria-labels 'Select item / Deselect item / Mark as available / Mark as unavaila… | [findings/microcopy.md](findings/microcopy.md) |
| BV-087 | dp-mc-password-show-toggle | microcopy | brand-voice | P3 | 'Show password' / 'Hide password' — correct sentence-case aria pair. Model. | [findings/microcopy.md](findings/microcopy.md) |
| BV-088 | mc-favorites-open-closed | microcopy | brand-voice | P3 | Mobile-customer chef status 'Open / Closed' — consistent with chef-detail. Flag… | [findings/microcopy.md](findings/microcopy.md) |
| BV-089 | mc-catering-placeholders | microcopy | brand-voice | P3 | Catering placeholder runs long and is comma-stuffed: 'Any specific requirements… | [findings/microcopy.md](findings/microcopy.md) |
| BV-090 | mc-catering-view-quote-hint | microcopy | brand-voice | P3 | Quote-ready hint 'Quotes available — view details' uses em-dash correctly (mode… | [findings/microcopy.md](findings/microcopy.md) |
| BV-091 | mc-catering-budget-display | microcopy | brand-voice | P3 | 'Budget: ₹{n}' — symbol-first, no space (matches style guide §3). Inline label… | [findings/microcopy.md](findings/microcopy.md) |
| BV-092 | mc-chef-delivery-meta | microcopy | brand-voice | P3 | 'Free delivery / ₹{n} delivery' inconsistent pattern. Better single form. | [findings/microcopy.md](findings/microcopy.md) |
| BV-093 | ap-auditlogs-pagination | microcopy | brand-voice | P3 | Pagination 'Page {n} of {m} · {n} total' — model. Compare to BV-094 which lacks… | [findings/microcopy.md](findings/microcopy.md) |
| BV-094 | ap-chefs-pagination-context | microcopy | brand-voice | P3 | 'Page {n} of {m} ({n} kitchens)' uses parens; audit-logs uses middot — inconsis… | [findings/microcopy.md](findings/microcopy.md) |
| BV-095 | ap-dashboard-recent-viewall | microcopy | brand-voice | P3 | 'View all' (admin dashboard) is sentence case correct; vendor-portal earnings u… | [findings/microcopy.md](findings/microcopy.md) |
| BV-096 | ap-users-action-view-aria | microcopy | brand-voice | P3 | aria-label 'View user details' is sentence case correct | [findings/microcopy.md](findings/microcopy.md) |
| BV-097 | web-ux-orders-search | microcopy | brand-voice | P3 | Two strings 'Search your orders' (aria) and 'Search orders...' (placeholder) —… | [findings/microcopy.md](findings/microcopy.md) |
| BV-098 | vp-ux-notifs-sla-line | microcopy | brand-voice | P3 | 'Typically reviewed within 24-48 hours' — vendor-portal vs mobile-vendor 'may t… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-028 | vp-onb-docs-required-label | microcopy | legal | P3 | "Required" badge without statement of the underlying legal requirement (FSSAI l… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-041 | vp-ux-notifs-sla-line | microcopy | legal | P3 | "Typically reviewed within 24-48 hours" — SLA statement is not framed as commit… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-042 | mv-onb-personal-email-helper | microcopy | legal | P3 | Helper "Email is pre-filled from your account" — does not surface that the emai… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-046 | mc-deliverymap-markers | microcopy | legal | P3 | Map markers "Delivery Address / Chef Location / Driver" — chef address shown to… | [findings/microcopy.md](findings/microcopy.md) |
| LEG-047 | web-ux-loading-default | microcopy | legal | P3 | Loading screen "Loading..." — no statement during slow/blocked loads of timeout… | [findings/microcopy.md](findings/microcopy.md) |
| TW-009 | PATTERN-loading-message-vague | microcopy | technical-writer | P3 | 'Loading...' appears as default load state across every app — vague, not glance… | [findings/microcopy.md](findings/microcopy.md) |
| TW-011 | PATTERN-aria-tooltip-mismatch | microcopy | technical-writer | P3 | Several admin surfaces ship both Title Case and sentence case for the same acti… | [findings/microcopy.md](findings/microcopy.md) |
| TW-012 | PATTERN-empty-state-format | microcopy | technical-writer | P3 | Several empty-state strings drop the 'Why → one action' formula and leave only… | [findings/microcopy.md](findings/microcopy.md) |
| TW-013 | PATTERN-helper-text-missing-mobile-vendor | microcopy | technical-writer | P3 | Mobile vendor settings helper text starts with 'Get notified when…' redundantly… | [findings/microcopy.md](findings/microcopy.md) |
| TW-014 | PATTERN-ampersand-in-labels | microcopy | technical-writer | P3 | Several labels and buttons use ampersand '&' instead of 'and' — style guide imp… | [findings/microcopy.md](findings/microcopy.md) |
| TW-018 | PATTERN-ai-slop-marketing-numerals | microcopy | technical-writer | P3 | Promo helper text 'Try: FE3DR10 for 10% off' is acceptable but uses '10% off' m… | [findings/microcopy.md](findings/microcopy.md) |
| TW-019 | PATTERN-time-relative-format | microcopy | technical-writer | P3 | Vendor notifications use compact 'Nm ago / Nh ago / Nd ago' — style guide §6 ma… | [findings/microcopy.md](findings/microcopy.md) |
| TW-021 | web-ux-cart-add-note | microcopy | technical-writer | P3 | Item-note prefix 'Note:' is a colon-suffix label — style guide §4 form-label ru… | [findings/microcopy.md](findings/microcopy.md) |
| TW-022 | web-ux-cart-add-more | microcopy | technical-writer | P3 | 'Add more items' is fine as a 3-word button but ambiguous on a cart screen wher… | [findings/microcopy.md](findings/microcopy.md) |
| TW-024 | web-ux-cart-delivery-info | microcopy | technical-writer | P3 | 'Estimated prep time: 30-45 mins' uses colon and 'mins' abbreviation — style gu… | [findings/microcopy.md](findings/microcopy.md) |
| TW-026 | web-ux-chefdetail-share-aria | microcopy | technical-writer | P3 | aria-label 'Share chef' is ambiguous — share what about the chef? | [findings/microcopy.md](findings/microcopy.md) |
| TW-028 | web-ux-orderdetail-cancel-reason | microcopy | technical-writer | P3 | 'Cancellation reason:' uses colon — style guide §4 prohibits colons on form lab… | [findings/microcopy.md](findings/microcopy.md) |
| TW-029 | web-ux-orders-search | microcopy | technical-writer | P3 | Two placeholders for the same input ('Search your orders' label-like vs 'Search… | [findings/microcopy.md](findings/microcopy.md) |
| TW-030 | web-ux-social-feed-comment | microcopy | technical-writer | P3 | Generic 'Add a comment…' placeholder lacks personality and signals nothing abou… | [findings/microcopy.md](findings/microcopy.md) |
| TW-031 | web-ux-currency-aria | microcopy | technical-writer | P3 | aria-label 'Change display currency' — 'display' is dev jargon for users | [findings/microcopy.md](findings/microcopy.md) |
| TW-032 | web-ux-chef-card-verified | microcopy | technical-writer | P3 | 'Verified chef' is fine but should match driver/admin variant ('Verified') for… | [findings/microcopy.md](findings/microcopy.md) |
| TW-033 | vp-auth-login-loading | microcopy | technical-writer | P3 | 'Signing in...' button loading state OK; flag only because pattern repeats acro… | [findings/microcopy.md](findings/microcopy.md) |
| TW-034 | vp-auth-register-password-placeholder | microcopy | technical-writer | P3 | 'Min. 8 characters' uses 'Min.' abbreviation inconsistently — readable but tigh… | [findings/microcopy.md](findings/microcopy.md) |
| TW-035 | vp-ux-dashboard-stats-labels | microcopy | technical-writer | P3 | 'Today's orders / Rating / This week / All-time orders' inconsistent voice — mi… | [findings/microcopy.md](findings/microcopy.md) |
| TW-037 | vp-ux-menu-form-dietary-tags | microcopy | technical-writer | P3 | Inconsistent hyphenation: 'Non-Veg', 'Gluten-Free', 'Nut-Free', 'Dairy-Free', '… | [findings/microcopy.md](findings/microcopy.md) |
| TW-038 | vp-ux-menu-form-allergen-help | microcopy | technical-writer | P3 | 'Type each allergen and press Enter to add' — slightly verbose | [findings/microcopy.md](findings/microcopy.md) |
| TW-039 | vp-ux-menu-form-images | microcopy | technical-writer | P3 | Image-upload status row mixes labels and instructions inconsistently — 'Primary… | [findings/microcopy.md](findings/microcopy.md) |
| TW-040 | vp-ux-menu-form-new-category | microcopy | technical-writer | P3 | Dialog title says 'New Category' (heading) but button says 'Create' (single wor… | [findings/microcopy.md](findings/microcopy.md) |
| TW-041 | vp-ux-menu-search-placeholder | microcopy | technical-writer | P3 | Filter default 'All Categories' Title Case (covered by TW-005 pattern) | [findings/microcopy.md](findings/microcopy.md) |
| TW-042 | vp-ux-notifs-sla-line | microcopy | technical-writer | P3 | 'Typically reviewed within 24-48 hours' — operationally vague (chef wants defin… | [findings/microcopy.md](findings/microcopy.md) |
| TW-043 | vp-ux-notifs-respond-form | microcopy | technical-writer | P3 | Reply form placeholder verbose and contains a parenthetical example — long for… | [findings/microcopy.md](findings/microcopy.md) |
| TW-044 | vp-onb-docs-kitchen-photo-items | microcopy | technical-writer | P3 | Photo labels mix hyphen styles: 'Kitchen Photo - Cooking Area' uses hyphen-dash… | [findings/microcopy.md](findings/microcopy.md) |
| TW-045 | vp-onb-docs-upload-helper | microcopy | technical-writer | P3 | 'Drop file here or click to browse. Max 5 MB.' — 'click' assumes desktop pointe… | [findings/microcopy.md](findings/microcopy.md) |
| TW-046 | vp-onb-kitchen-desc-field | microcopy | technical-writer | P3 | Description placeholder 'Describe your cooking style…' OK but helper 'Min 20 ch… | [findings/microcopy.md](findings/microcopy.md) |
| TW-047 | vp-onb-kitchen-experience | microcopy | technical-writer | P3 | 'Less than 1 year' uses Title Case for the option label — style guide says sent… | [findings/microcopy.md](findings/microcopy.md) |
| TW-048 | vp-onb-kitchen-meals | microcopy | technical-writer | P3 | 'Up to 10 meals' is inconsistent with neighbouring buckets '10-25', '25-50' (hy… | [findings/microcopy.md](findings/microcopy.md) |
| TW-049 | vp-onb-ops-prep-times | microcopy | technical-writer | P3 | Ranges use hyphen not en dash; '2+ hours (pre-order only)' parenthetical inside… | [findings/microcopy.md](findings/microcopy.md) |
| TW-050 | vp-onb-ops-min-order | microcopy | technical-writer | P3 | Helper 'Set to 0 for no minimum' uses bare 0 — currency-format consistency expe… | [findings/microcopy.md](findings/microcopy.md) |
| TW-051 | vp-onb-policies-payout-info | microcopy | technical-writer | P3 | Deferred-setup explanation is 23 words — exceeds chef tone matrix crisp 5-12 se… | [findings/microcopy.md](findings/microcopy.md) |
| TW-052 | vp-onb-header-step-label | microcopy | technical-writer | P3 | 'Step {n} of 5' is fine; partner string 'Review your application' is a sentence… | [findings/microcopy.md](findings/microcopy.md) |
| TW-053 | vp-onb-banner-admin-notes-label | microcopy | technical-writer | P3 | 'Admin Notes:' uses Title Case AND colon — double violation of label rules | [findings/microcopy.md](findings/microcopy.md) |
| TW-055 | vp-ux-orders-live-items-total | microcopy | technical-writer | P3 | '{n} items total' awkward — prefer locale-aware plural | [findings/microcopy.md](findings/microcopy.md) |
| TW-056 | vp-ux-orders-history-ranges | microcopy | technical-writer | P3 | Date-range select labels OK but inconsistent capitalization: 'Today / Last 7 da… | [findings/microcopy.md](findings/microcopy.md) |
| TW-057 | vp-ux-kitchen-payout-fields | microcopy | technical-writer | P3 | Placeholders for bank/account/IFSC use 'e.g.' inconsistently — some fields have… | [findings/microcopy.md](findings/microcopy.md) |
| TW-058 | vp-ux-profile-toggle | microcopy | technical-writer | P3 | Order-acceptance switch labels 'Accepting Orders' / 'Orders Paused' use Title C… | [findings/microcopy.md](findings/microcopy.md) |
| TW-059 | vp-ux-profile-doc-actions | microcopy | technical-writer | P3 | Doc-row inline reason 'Reason: {rejectionReason}' uses colon-suffix label | [findings/microcopy.md](findings/microcopy.md) |
| TW-060 | vp-ux-reviews-subratings | microcopy | technical-writer | P3 | Sub-rating chip format 'Food: {n}/5' uses colon-suffix label inside a chip — ch… | [findings/microcopy.md](findings/microcopy.md) |
| TW-062 | vp-ux-settings-payout-form | microcopy | technical-writer | P3 | Placeholder 'Name as on bank account' is wordy | [findings/microcopy.md](findings/microcopy.md) |
| TW-063 | vp-ux-theme-toggle | microcopy | technical-writer | P3 | Theme options 'Light / System / Dark' sentence case OK; flag for consistency ac… | [findings/microcopy.md](findings/microcopy.md) |
| TW-064 | dp-mc-portal-footer | microcopy | technical-writer | P3 | 'Fe3dr Delivery Portal' — internal brand naming inconsistency check (Mark8ly/Fe… | [findings/microcopy.md](findings/microcopy.md) |
| TW-065 | dp-mc-suspended-badge | microcopy | technical-writer | P3 | 'Suspended' single-word status OK — flagged to track legal-adjacent terms; ensu… | [findings/microcopy.md](findings/microcopy.md) |
| TW-066 | dp-mc-online-badge | microcopy | technical-writer | P3 | 'Online' as driver status is correct but watch for collision with 'Open' (chef… | [findings/microcopy.md](findings/microcopy.md) |
| TW-067 | dp-mc-partners-search | microcopy | technical-writer | P3 | Placeholder 'Search by name, email, or vehicle...' is fine; aria-label 'Search… | [findings/microcopy.md](findings/microcopy.md) |
| TW-068 | dp-mc-stripe-cells | microcopy | technical-writer | P3 | Stripe status row mixes states 'Enabled / Pending / Submitted / Incomplete' — '… | [findings/microcopy.md](findings/microcopy.md) |
| TW-069 | dp-mc-copy-state | microcopy | technical-writer | P3 | Button toggles between 'Copy' and 'Copied' — 'Copied' should be a transient toa… | [findings/microcopy.md](findings/microcopy.md) |
| TW-070 | dp-mc-displayname-fallback | microcopy | technical-writer | P3 | 'Driver' as a display-name fallback works but contradicts persona matrix where… | [findings/microcopy.md](findings/microcopy.md) |
| TW-072 | ap-approvaldetail-notes-placeholder | microcopy | technical-writer | P3 | Placeholder 'Add notes for this approval action (optional)...' is verbose | [findings/microcopy.md](findings/microcopy.md) |
| TW-073 | ap-approvals-priorities | microcopy | technical-writer | P3 | Priority badges 'urgent / high / normal / low' are lowercase — inconsistent wit… | [findings/microcopy.md](findings/microcopy.md) |
| TW-074 | ap-approvals-search-placeholders | microcopy | technical-writer | P3 | Two contextual placeholders 'Search by title or driver...' vs 'Search by title,… | [findings/microcopy.md](findings/microcopy.md) |
| TW-075 | ap-auth-totp-secret-helper | microcopy | technical-writer | P3 | Helper 'or enter this key' is fragment — admin tone wants precision | [findings/microcopy.md](findings/microcopy.md) |
| TW-076 | ap-auth-email-placeholder | microcopy | technical-writer | P3 | 'admin@fe3dr.com' placeholder leaks internal brand into customer-adjacent surfa… | [findings/microcopy.md](findings/microcopy.md) |
| TW-077 | ap-chefs-card-unnamed | microcopy | technical-writer | P3 | Fallback 'Unnamed Kitchen' Title Case + emotionally cold for admin UI | [findings/microcopy.md](findings/microcopy.md) |
| TW-078 | ap-chefs-pagination-context | microcopy | technical-writer | P3 | Pagination format 'Page {n} of {m} ({n} kitchens)' reuses '{n}' for two differe… | [findings/microcopy.md](findings/microcopy.md) |
| TW-079 | ap-dashboard-revenue-vs | microcopy | technical-writer | P3 | 'vs. yesterday' / 'vs. last week' uses abbreviation 'vs.' with period — admin t… | [findings/microcopy.md](findings/microcopy.md) |
| TW-080 | ap-dashboard-stat-users-today | microcopy | technical-writer | P3 | '+{n} today' subtitle is glanceable but ambiguous about WHAT increased | [findings/microcopy.md](findings/microcopy.md) |
| TW-081 | ap-delivery-count-meta | microcopy | technical-writer | P3 | '{n} total deliveries' — 'total' redundant with the count | [findings/microcopy.md](findings/microcopy.md) |
| TW-082 | ap-providercreate-helpers | microcopy | technical-writer | P3 | Multiple helper texts: 'Auto-generated from name. Edit to customize.' / '0 = un… | [findings/microcopy.md](findings/microcopy.md) |
| TW-083 | ap-providercreate-mapping-row | microcopy | technical-writer | P3 | 'maps to' as inline connector is fine; ensure consistency with helper above tha… | [findings/microcopy.md](findings/microcopy.md) |
| TW-085 | ap-secsettings-apikey-meta | microcopy | technical-writer | P3 | Key metadata strings 'no scopes; expires {date}; revoked' are lowercase fragmen… | [findings/microcopy.md](findings/microcopy.md) |
| TW-086 | ap-settings-payment-test-buttons | microcopy | technical-writer | P3 | State labels 'Configured / Not configured' mix sentence case correctly; loading… | [findings/microcopy.md](findings/microcopy.md) |
| TW-087 | ap-staffdetail-info-fallbacks | microcopy | technical-writer | P3 | Fallback set 'Not provided / Not assigned / Never' inconsistent — some are sent… | [findings/microcopy.md](findings/microcopy.md) |
| TW-088 | ap-staff-invite-placeholders | microcopy | technical-writer | P3 | Email placeholder 'colleague@company.com' is good; helper 'Optional message to… | [findings/microcopy.md](findings/microcopy.md) |
| TW-089 | ap-users-pagination | microcopy | technical-writer | P3 | 'Previous' / 'Next' OK; 'Page {n} of {m}' fine but inconsistent with ap-auditlo… | [findings/microcopy.md](findings/microcopy.md) |
| TW-090 | ap-platsettings-hours-defaults | microcopy | technical-writer | P3 | Helper 'No days selected = every day.' uses '=' symbol — admin tone is precise… | [findings/microcopy.md](findings/microcopy.md) |
| TW-091 | mc-perm-faceid | microcopy | technical-writer | P3 | Face ID prompt 'Use Face ID to log in quickly' uses banned 'log in' verb | [findings/microcopy.md](findings/microcopy.md) |
| TW-092 | mc-perm-location | microcopy | technical-writer | P3 | Location prompt 'Used to show your location on the delivery tracking map' is 11… | [findings/microcopy.md](findings/microcopy.md) |
| TW-093 | mc-favorites-subtitle | microcopy | technical-writer | P3 | Subtitle '{n}/{max} saved' is glanceable but ambiguous — what's saved? | [findings/microcopy.md](findings/microcopy.md) |
| TW-094 | mc-home-search-placeholder | microcopy | technical-writer | P3 | Placeholder 'Search chefs, cuisines...' uses comma list with trailing ellipsis… | [findings/microcopy.md](findings/microcopy.md) |
| TW-095 | mc-profile-placeholders | microcopy | technical-writer | P3 | Profile placeholders 'First name / Last name / +91 9876543210' OK; ensure phone… | [findings/microcopy.md](findings/microcopy.md) |
| TW-096 | mc-catering-budget-display | microcopy | technical-writer | P3 | 'Budget: ₹{n}' uses colon-suffix label inline — see TW-021 colon pattern | [findings/microcopy.md](findings/microcopy.md) |
| TW-097 | mc-catering-view-quote-hint | microcopy | technical-writer | P3 | 'Quotes available — view details' uses em dash separator — good; flag for verb-… | [findings/microcopy.md](findings/microcopy.md) |
| TW-098 | mc-checkout-note | microcopy | technical-writer | P3 | Label 'Note to chef (optional)' merges optional indicator into label — style gu… | [findings/microcopy.md](findings/microcopy.md) |
| TW-099 | mc-chef-delivery-meta | microcopy | technical-writer | P3 | 'Free delivery' / '₹{n} delivery' format inconsistent — one is text, one is cur… | [findings/microcopy.md](findings/microcopy.md) |
| TW-100 | mc-order-detail-eta | microcopy | technical-writer | P3 | 'ETA: {time}' uses acronym + colon — customer surface (warmer) should expand | [findings/microcopy.md](findings/microcopy.md) |
| TW-101 | mc-order-detail-footer | microcopy | technical-writer | P3 | 'Ordered on {date}' OK — flag for consistency with admin/vendor variants | [findings/microcopy.md](findings/microcopy.md) |
| TW-102 | mc-timeline-eta | microcopy | technical-writer | P3 | 'Est. arrival: {time}' uses abbreviation 'Est.' AND colon — customer surface sh… | [findings/microcopy.md](findings/microcopy.md) |
| TW-103 | mc-deliverymap-markers | microcopy | technical-writer | P3 | Map marker titles 'Delivery Address / Chef Location / Driver' use Title Case —… | [findings/microcopy.md](findings/microcopy.md) |
| TW-104 | mv-onb-personal-email-helper | microcopy | technical-writer | P3 | 'Email is pre-filled from your account' is fine — 6 words, chef-tone OK | [findings/microcopy.md](findings/microcopy.md) |
| TW-105 | mv-push-channel-neworders | microcopy | technical-writer | P3 | Android notification channel 'New Orders' Title Case appears in OS settings — m… | [findings/microcopy.md](findings/microcopy.md) |
| TW-106 | mv-push-channel-orderupdates | microcopy | technical-writer | P3 | Android channel 'Order Updates' Title Case (same as TW-105) | [findings/microcopy.md](findings/microcopy.md) |
| TW-108 | mv-settings-accepting-helper | microcopy | technical-writer | P3 | Helper 'Toggle to start or pause accepting orders' is awkward — toggle is mecha… | [findings/microcopy.md](findings/microcopy.md) |
| TW-110 | md-mic-004 | microcopy | technical-writer | P3 | 'Delivery Updates' channel name Title Case (paired with TW-109) | [findings/microcopy.md](findings/microcopy.md) |
| TW-111 | md-mic-002 | microcopy | technical-writer | P3 | Driver back button aria 'Go back' OK — telegraphic and clear; flag for cross-ap… | [findings/microcopy.md](findings/microcopy.md) |
| TW-112 | md-mic-001 | microcopy | technical-writer | P3 | Driver Alert buttons 'Cancel / Logout' — same banned 'Logout' (covered by TW-00… | [findings/microcopy.md](findings/microcopy.md) |
| TW-113 | web-ux-a11y-skip-link | microcopy | technical-writer | P3 | 'Skip to main content' is the WCAG-standard label; flag for cross-app standardi… | [findings/microcopy.md](findings/microcopy.md) |
| TW-114 | vp-ux-layout-aria | microcopy | technical-writer | P3 | aria-label set 'Open navigation / Close navigation / Notifications, {n} unread… | [findings/microcopy.md](findings/microcopy.md) |
| TW-115 | vp-onb-nav-continue | microcopy | technical-writer | P3 | Onboarding nav 'Back' / 'Continue' — single word, verb-first, sentence case — a… | [findings/microcopy.md](findings/microcopy.md) |
| TW-117 | ap-platsettings-tooltips | microcopy | technical-writer | P3 | Delete tooltip 'Delete / Delete zone / Toggle {day}' mixes scope — 'Delete' is… | [findings/microcopy.md](findings/microcopy.md) |
| TW-118 | ap-orders-action-aria | microcopy | technical-writer | P3 | 'Actions for order {n}' OK; ensure {n} is the order ID format from §6 ('#HC-202… | [findings/microcopy.md](findings/microcopy.md) |
| TW-119 | ap-auditlogs-pagination | microcopy | technical-writer | P3 | 'Page {n} of {m} · {n} total' reuses {n} ambiguously (same as TW-078) | [findings/microcopy.md](findings/microcopy.md) |
| TW-120 | vp-onb-stepper-review-step | microcopy | technical-writer | P3 | Stepper label 'Submit application' OK (sentence case, 2 words, verb-first) — fl… | [findings/microcopy.md](findings/microcopy.md) |

### Index — help

42 findings. Detail: [findings/help.md](findings/help.md)

| finding_id | surface_id | category | lens | severity | issue (≤80 chars) | detail link |
|---|---|---|---|---|---|---|
| LEG-HLP-001 | web-help-otto-empty | help | legal | P0 | Otto AI chat invites customers to ask about 'order status, delivery, refunds, c… | [findings/help.md](findings/help.md) |
| LEG-HLP-006 | dp-help-status-contact | help | legal | P0 | Onboarding status help block 'Need Help? Contact us at support@fe3dr.com' is th… | [findings/help.md](findings/help.md) |
| LEG-HLP-008 | dp-help-new-driver-body | help | legal | P0 | Driver-recruitment help text 'Sign up with your email or Google account. You'll… | [findings/help.md](findings/help.md) |
| LEG-HLP-011 | ap-settings-payment-test-cards | help | legal | P0 | Admin help block 'Test Cards; Visa; Mastercard; UPI; 3D Secure' lists sandbox/t… | [findings/help.md](findings/help.md) |
| BA-HLP-001 | dp-help-new-driver | help | business-analyst | P1 | "New Driver?" sign-up hint has no CTA button — driver activation pathway dead-e… | [findings/help.md](findings/help.md) |
| BA-HLP-002 | dp-help-status-contact | help | business-analyst | P1 | Support email domain "fe3dr.com" contradicts HomeChef brand — driver who Google… | [findings/help.md](findings/help.md) |
| BA-HLP-003 | dp-help-new-driver-body | help | business-analyst | P1 | "Login or sign up to deliver with Fe3dr" on choose-mode screen mixes brand name… | [findings/help.md](findings/help.md) |
| BV-HLP-001 | web-help-otto-launcher | help | brand-voice | P1 | Support is exposed under two names on the SAME customer app: the launcher aria-… | [findings/help.md](findings/help.md) |
| BV-HLP-002 | dp-help-status-contact | help | brand-voice | P1 | Cross-app brand-name contradiction. The web app brands itself 'HomeChef' in the… | [findings/help.md](findings/help.md) |
| LEG-HLP-002 | web-help-otto-sender | help | legal | P1 | Sender label 'Otto · AI assistant / Support' is the only place the user is told… | [findings/help.md](findings/help.md) |
| LEG-HLP-004 | web-help-otto-chat-header | help | legal | P1 | Chat header strings ('HomeChef support chat', 'HomeChef support', 'Close suppor… | [findings/help.md](findings/help.md) |
| LEG-HLP-007 | dp-help-settings-help | help | legal | P1 | Driver settings tile 'Help & Support / Get help with your deliveries' is the st… | [findings/help.md](findings/help.md) |
| LEG-HLP-009 | dp-help-staff-access-body | help | legal | P1 | 'Only pre-authorized fleet managers and delivery operations staff can sign in h… | [findings/help.md](findings/help.md) |
| LEG-HLP-012 | web-help-otto-chat-header | help | legal | P1 | 'Connecting...' state in the Otto chat header is the only visible signal of sup… | [findings/help.md](findings/help.md) |
| BA-HLP-004 | dp-help-settings-help | help | business-analyst | P2 | "Help & Support" settings entry is a dead-end button with no destination — tapp… | [findings/help.md](findings/help.md) |
| BA-HLP-005 | web-help-otto-empty | help | business-analyst | P2 | Otto empty-state prompt mixes four topic categories with no intake routing — cu… | [findings/help.md](findings/help.md) |
| BA-HLP-006 | web-help-otto-chat-header | help | business-analyst | P2 | Chat header labels bot as "HomeChef support" without signalling AI — customers… | [findings/help.md](findings/help.md) |
| BA-HLP-007 | dp-help-status-contact | help | business-analyst | P2 | Onboarding status page offers only email contact — no SLA promise, no chat, no… | [findings/help.md](findings/help.md) |
| BV-HLP-003 | dp-help-new-driver | help | brand-voice | P2 | 'New Driver?' uses Title Case for a non-proper-noun. Style guide enforces sente… | [findings/help.md](findings/help.md) |
| BV-HLP-004 | dp-help-new-driver-body | help | brand-voice | P2 | Driver-facing help body is 18 words across two sentences. Style-guide persona m… | [findings/help.md](findings/help.md) |
| BV-HLP-005 | dp-help-staff-access-body | help | brand-voice | P2 | Same persona-tone drift as BV-HLP-004 on the staff-access help card. 17 words,… | [findings/help.md](findings/help.md) |
| BV-HLP-006 | web-help-otto-input | help | brand-voice | P2 | 'Connecting...' as the message-input placeholder is bland infrastructure chatte… | [findings/help.md](findings/help.md) |
| LEG-HLP-003 | web-help-otto-launcher | help | legal | P2 | Floating launcher aria-label 'Open support chat' positions the AI chatbot as th… | [findings/help.md](findings/help.md) |
| LEG-HLP-005 | web-help-otto-input | help | legal | P2 | Input placeholder 'Type a message...' and send button 'Send message' collect fr… | [findings/help.md](findings/help.md) |
| TW-004 | dp-help-new-driver | help | technical-writer | P2 | Title 'New Driver?' uses Title Case + interrogative tone on what is a labeled i… | [findings/help.md](findings/help.md) |
| TW-005 | dp-help-new-driver-body | help | technical-writer | P2 | Body sentence is 18 words — driver persona max is 12 per style-guide §2 tone ma… | [findings/help.md](findings/help.md) |
| TW-006 | dp-help-staff-access-title | help | technical-writer | P2 | Title Case 'Staff Access' violates style-guide §4: 'noun, sentence case'. Same… | [findings/help.md](findings/help.md) |
| TW-007 | dp-help-staff-access-body | help | technical-writer | P2 | Body is 17 words — over the 12-word ceiling for driver-facing copy. While 'Staf… | [findings/help.md](findings/help.md) |
| TW-008 | dp-help-status-contact | help | technical-writer | P2 | Title 'Need Help?' (Title Case + question mark) again breaks sentence-case noun… | [findings/help.md](findings/help.md) |
| TW-009 | dp-help-settings-help | help | technical-writer | P2 | Title 'Help & Support' uses Title Case (style-guide §4 requires sentence case).… | [findings/help.md](findings/help.md) |
| TW-011 | ap-settings-payment-test-cards | help | technical-writer | P2 | Section label 'Test Cards' is Title Case (violates §4 sentence-case rule for la… | [findings/help.md](findings/help.md) |
| BA-HLP-008 | dp-help-staff-access-body | help | business-analyst | P3 | "Contact an admin for access" on staff login has no email, link, or next step —… | [findings/help.md](findings/help.md) |
| BA-HLP-009 | ap-settings-payment-test-cards | help | business-analyst | P3 | Test-card reference panel in admin settings has no "test mode only" warning lab… | [findings/help.md](findings/help.md) |
| BA-HLP-010 | web-help-otto-launcher | help | business-analyst | P3 | Floating chat launcher aria-label is "Open support chat" — no product name or A… | [findings/help.md](findings/help.md) |
| BV-HLP-007 | web-help-otto-sender | help | brand-voice | P3 | Two sender labels appear back-to-back: 'Otto · AI assistant' (humanised bot nam… | [findings/help.md](findings/help.md) |
| BV-HLP-008 | web-help-otto-empty | help | brand-voice | P3 | Em-dash separator + comma-separated topic list ('Ask anything — order status, d… | [findings/help.md](findings/help.md) |
| LEG-HLP-010 | dp-help-staff-access-title | help | legal | P3 | 'Staff Access' heading on a public-facing login page reveals the existence of a… | [findings/help.md](findings/help.md) |
| TW-001 | web-help-otto-launcher | help | technical-writer | P3 | Inventory captures aria-label and chat strings at lines 146-260 of OttoChat.tsx… | [findings/help.md](findings/help.md) |
| TW-002 | web-help-otto-empty | help | technical-writer | P3 | Empty-state copy 'Ask anything — order status, delivery, refunds, chef availabi… | [findings/help.md](findings/help.md) |
| TW-003 | web-help-otto-input | help | technical-writer | P3 | Loading-state 'Connecting...' uses three ASCII dots, not Unicode ellipsis. Same… | [findings/help.md](findings/help.md) |
| TW-010 | dp-help-settings-help | help | technical-writer | P3 | Inconsistency: settings row title 'Help & Support' (Title Case, ampersand) vs s… | [findings/help.md](findings/help.md) |
| TW-012 | ap-settings-payment-test-cards | help | technical-writer | P3 | Missing helper text. 'Test cards' label drops the user into a list of card numb… | [findings/help.md](findings/help.md) |

### Index — seo-meta

77 findings. Detail: [findings/seo-meta.md](findings/seo-meta.md)

| finding_id | surface_id | category | lens | severity | issue (≤80 chars) | detail link |
|---|---|---|---|---|---|---|
| BA-SEO-001 | web-seo-og-title | seo-meta | business-analyst | P0 | OG and Twitter card images declared but og-image.png is absent from the public… | [findings/seo-meta.md](findings/seo-meta.md) |
| BA-SEO-002 | dp-seo-html-title | seo-meta | business-analyst | P0 | Duplicate <title> tag in delivery-portal/index.html — two identical <title>Fe3d… | [findings/seo-meta.md](findings/seo-meta.md) |
| BA-SEO-003 | ap-seo-meta-title | seo-meta | business-analyst | P0 | Duplicate <title> tag in admin-portal/index.html — two identical <title>Fe3dr A… | [findings/seo-meta.md](findings/seo-meta.md) |
| BV-SEO-001 | md-meta-001 | seo-meta | brand-voice | P0 | Brand-name fork between web and mobile: every web surface and OG/Twitter card s… | [findings/seo-meta.md](findings/seo-meta.md) |
| BV-SEO-002 | md-meta-004 | seo-meta | brand-voice | P0 | iOS permission rationale strings hardcode the wrong brand ('HomeChef Delivery')… | [findings/seo-meta.md](findings/seo-meta.md) |
| LEG-001 | md-meta-001 | seo-meta | legal | P0 | Brand-name mismatch between web/portals ("Fe3dr") and mobile-delivery app ("Hom… | [findings/seo-meta.md](findings/seo-meta.md) |
| LEG-007 | md-meta-003 | seo-meta | legal | P0 | NSCameraUsageDescription "Used to take photos" — fails DPDP §5(a) specific-purp… | [findings/seo-meta.md](findings/seo-meta.md) |
| LEG-010 | md-meta-005 | seo-meta | legal | P0 | Background-location rationale "tracks your location in the background to keep c… | [findings/seo-meta.md](findings/seo-meta.md) |
| LEG-011 | md-meta-006 | seo-meta | legal | P0 | iOS/Android location-rationale drift — iOS NSLocationAlwaysAndWhenInUseUsageDes… | [findings/seo-meta.md](findings/seo-meta.md) |
| TW-027 | md-meta-005 | seo-meta | technical-writer | P0 | NSLocationAlwaysAndWhenInUseUsageDescription is the highest-risk permission pro… | [findings/seo-meta.md](findings/seo-meta.md) |
| BA-SEO-004 | web-seo-meta-description | seo-meta | business-analyst | P1 | Primary meta description starts with the brand name prefix 'Fe3dr -' which wast… | [findings/seo-meta.md](findings/seo-meta.md) |
| BA-SEO-005 | web-seo-og-title | seo-meta | business-analyst | P1 | OG title and HTML title are identical strings ('Fe3dr - Homemade Food Delivered… | [findings/seo-meta.md](findings/seo-meta.md) |
| BA-SEO-006 | web-seo-og-description | seo-meta | business-analyst | P1 | OG description is a near-verbatim copy of the meta description but without the… | [findings/seo-meta.md](findings/seo-meta.md) |
| BA-SEO-007 | md-meta-003 | seo-meta | business-analyst | P1 | iOS NSCameraUsageDescription 'Used to take photos' is dangerously terse — iOS m… | [findings/seo-meta.md](findings/seo-meta.md) |
| BA-SEO-008 | md-meta-005 | seo-meta | business-analyst | P1 | iOS NSLocationAlwaysAndWhenInUseUsageDescription differs from the Android expo-… | [findings/seo-meta.md](findings/seo-meta.md) |
| BA-SEO-009 | md-meta-001 | seo-meta | business-analyst | P1 | Mobile delivery app display name is 'HomeChef Delivery' while all web/portal su… | [findings/seo-meta.md](findings/seo-meta.md) |
| BA-SEO-010 | vp-seo-html-title | seo-meta | business-analyst | P1 | Vendor portal has no og:image or Twitter card tags at all — if a chef shares th… | [findings/seo-meta.md](findings/seo-meta.md) |
| BV-SEO-003 | web-seo-meta-description | seo-meta | brand-voice | P1 | Customer landing meta description uses 'authentic homemade food' — a faux-artis… | [findings/seo-meta.md](findings/seo-meta.md) |
| BV-SEO-004 | web-seo-og-description | seo-meta | brand-voice | P1 | OG/Twitter card description is identical to the meta description and carries th… | [findings/seo-meta.md](findings/seo-meta.md) |
| BV-SEO-005 | web-seo-og-title | seo-meta | brand-voice | P1 | OG/Twitter/document title uses Title Case ('Fe3dr - Homemade Food Delivered').… | [findings/seo-meta.md](findings/seo-meta.md) |
| BV-SEO-006 | vp-seo-meta-description | seo-meta | brand-voice | P1 | The four portal meta descriptions are written in four different voices: custome… | [findings/seo-meta.md](findings/seo-meta.md) |
| BV-SEO-007 | dp-seo-meta-description | seo-meta | brand-voice | P1 | Inventory itself flags drift: meta description says 'Delivery Partner Portal' b… | [findings/seo-meta.md](findings/seo-meta.md) |
| LEG-002 | web-seo-meta-description | seo-meta | legal | P1 | "authentic homemade food" — origin/quality claim in indexable meta description… | [findings/seo-meta.md](findings/seo-meta.md) |
| LEG-003 | web-seo-og-description | seo-meta | legal | P1 | "authentic homemade food delivered to your doorstep" — same unsubstantiated ori… | [findings/seo-meta.md](findings/seo-meta.md) |
| LEG-004 | web-seo-meta-keywords | seo-meta | legal | P1 | "catering" keyword — implies catering service offering; FSSAI Licensing & Regis… | [findings/seo-meta.md](findings/seo-meta.md) |
| LEG-006 | dp-seo-meta-description | seo-meta | legal | P1 | "Delivery Partner Portal" — "Partner" framing in indexable meta description has… | [findings/seo-meta.md](findings/seo-meta.md) |
| LEG-008 | md-meta-002 | seo-meta | legal | P1 | NSFaceIDUsageDescription "Use Face ID to log in quickly" — biometric purpose st… | [findings/seo-meta.md](findings/seo-meta.md) |
| LEG-009 | md-meta-004 | seo-meta | legal | P1 | NSLocationWhenInUseUsageDescription names purpose but omits DPDP §5 elements: r… | [findings/seo-meta.md](findings/seo-meta.md) |
| LEG-013 | ap-seo-meta-description | seo-meta | legal | P1 | Admin-portal meta description is set even though inventory notes "noindex/nofol… | [findings/seo-meta.md](findings/seo-meta.md) |
| TW-011 | md-meta-001 | seo-meta | technical-writer | P1 | Brand-name drift: mobile-delivery app.json displays 'HomeChef Delivery' as the… | [findings/seo-meta.md](findings/seo-meta.md) |
| TW-013 | md-meta-006 | seo-meta | technical-writer | P1 | Permission-rationale drift between iOS and Android. iOS Info.plist says 'tracks… | [findings/seo-meta.md](findings/seo-meta.md) |
| TW-015 | dp-seo-html-title | seo-meta | technical-writer | P1 | Duplicated `<title>` tag inside `<head>` (lines 24 and 32 of apps/delivery-port… | [findings/seo-meta.md](findings/seo-meta.md) |
| TW-016 | ap-seo-meta-title | seo-meta | technical-writer | P1 | Duplicated `<title>` tag in apps/admin-portal/index.html (lines 38 and 43). Sam… | [findings/seo-meta.md](findings/seo-meta.md) |
| TW-024 | md-meta-003 | seo-meta | technical-writer | P1 | NSCameraUsageDescription is 'Used to take photos' — 4 words, no subject, no pur… | [findings/seo-meta.md](findings/seo-meta.md) |
| BA-SEO-011 | web-seo-html-title | seo-meta | business-analyst | P2 | HTML title is a static string that never updates per-route — users who deep-lin… | [findings/seo-meta.md](findings/seo-meta.md) |
| BA-SEO-012 | web-seo-meta-description | seo-meta | business-analyst | P2 | No structured data (JSON-LD) for LocalBusiness, FoodService, or Restaurant sche… | [findings/seo-meta.md](findings/seo-meta.md) |
| BA-SEO-013 | web-seo-meta-keywords | seo-meta | business-analyst | P2 | Meta keywords tag is present but Google has ignored meta keywords since 2009 —… | [findings/seo-meta.md](findings/seo-meta.md) |
| BA-SEO-014 | vp-seo-meta-keywords | seo-meta | business-analyst | P2 | Vendor portal meta keywords tag includes 'vendor portal' and 'food business' —… | [findings/seo-meta.md](findings/seo-meta.md) |
| BA-SEO-015 | web-seo-app-name | seo-meta | business-analyst | P2 | PWA manifest.json sets theme_color to '#f97316' (orange) which conflicts with t… | [findings/seo-meta.md](findings/seo-meta.md) |
| BA-SEO-016 | dp-seo-meta-description | seo-meta | business-analyst | P2 | Delivery portal meta description uses 'Delivery Partner Portal' but the portal… | [findings/seo-meta.md](findings/seo-meta.md) |
| BA-SEO-017 | web-seo-twitter-description | seo-meta | business-analyst | P2 | No twitter:site handle declared — Twitter cards will render without an attribut… | [findings/seo-meta.md](findings/seo-meta.md) |
| BV-SEO-008 | web-seo-meta-keywords | seo-meta | brand-voice | P2 | The customer meta keywords tag ('home chef, food delivery, homemade food, local… | [findings/seo-meta.md](findings/seo-meta.md) |
| BV-SEO-009 | vp-seo-meta-keywords | seo-meta | brand-voice | P2 | Vendor portal keywords ('vendor portal, home chef, kitchen management, food bus… | [findings/seo-meta.md](findings/seo-meta.md) |
| BV-SEO-010 | md-meta-003 | seo-meta | brand-voice | P2 | NSCameraUsageDescription is two words ('Used to take photos') with no subject,… | [findings/seo-meta.md](findings/seo-meta.md) |
| BV-SEO-011 | md-meta-005 | seo-meta | brand-voice | P2 | Background-location rationale is also wordy and possessive-heavy: 'HomeChef Del… | [findings/seo-meta.md](findings/seo-meta.md) |
| BV-SEO-012 | md-meta-006 | seo-meta | brand-voice | P2 | expo-location plugin Android rationale ('tracks your location during deliveries… | [findings/seo-meta.md](findings/seo-meta.md) |
| LEG-005 | web-seo-og-title | seo-meta | legal | P2 | "Homemade Food Delivered" — absolute claim used as standalone tagline in og:tit… | [findings/seo-meta.md](findings/seo-meta.md) |
| LEG-012 | web-seo-app-name | seo-meta | legal | P2 | No data-fiduciary identification on indexed surfaces — meta `author="Fe3dr"` an… | [findings/seo-meta.md](findings/seo-meta.md) |
| LEG-014 | vp-seo-meta-description | seo-meta | legal | P2 | "Manage your kitchen, menus, orders, and earnings" — "earnings" framing in inde… | [findings/seo-meta.md](findings/seo-meta.md) |
| LEG-015 | web-seo-og-title | seo-meta | legal | P2 | OG image `/og-image.png` and Twitter image referenced from indexable meta — if… | [findings/seo-meta.md](findings/seo-meta.md) |
| TW-001 | web-seo-meta-description | seo-meta | technical-writer | P2 | Voice drift: 'authentic homemade food delivered to your doorstep' uses two fill… | [findings/seo-meta.md](findings/seo-meta.md) |
| TW-002 | web-seo-og-description | seo-meta | technical-writer | P2 | Same 'authentic homemade food delivered to your doorstep' filler as meta descri… | [findings/seo-meta.md](findings/seo-meta.md) |
| TW-003 | web-seo-twitter-description | seo-meta | technical-writer | P2 | Identical filler copy as OG and meta description. Same banned-adjacent vocabula… | [findings/seo-meta.md](findings/seo-meta.md) |
| TW-005 | web-seo-html-title | seo-meta | technical-writer | P2 | <title> is 31 characters — well below the 50-60 char SEO sweet spot. Wastes SER… | [findings/seo-meta.md](findings/seo-meta.md) |
| TW-006 | web-seo-meta-description | seo-meta | technical-writer | P2 | Meta description is 88 characters — below the 150-160 char SEO target. Truncate… | [findings/seo-meta.md](findings/seo-meta.md) |
| TW-012 | dp-seo-meta-description | seo-meta | technical-writer | P2 | Term drift: meta description says 'Delivery Partner Portal' while every other d… | [findings/seo-meta.md](findings/seo-meta.md) |
| TW-017 | dp-seo-html-title | seo-meta | technical-writer | P2 | delivery-portal/index.html has no Open Graph or Twitter card meta tags, no `<me… | [findings/seo-meta.md](findings/seo-meta.md) |
| TW-019 | vp-seo-meta-description | seo-meta | technical-writer | P2 | vendor-portal has no `<meta name="robots">` tag. Authenticated portal (gated) s… | [findings/seo-meta.md](findings/seo-meta.md) |
| TW-025 | md-meta-002 | seo-meta | technical-writer | P2 | NSFaceIDUsageDescription 'Use Face ID to log in quickly' is friendly but uses t… | [findings/seo-meta.md](findings/seo-meta.md) |
| TW-026 | md-meta-004 | seo-meta | technical-writer | P2 | NSLocationWhenInUseUsageDescription is acceptable but uses the brand-drift name… | [findings/seo-meta.md](findings/seo-meta.md) |
| BA-SEO-018 | web-seo-og-title | seo-meta | business-analyst | P3 | og:url is hardcoded to 'https://fe3dr.com' in all page variants — for an SPA th… | [findings/seo-meta.md](findings/seo-meta.md) |
| BA-SEO-019 | md-meta-002 | seo-meta | business-analyst | P3 | iOS Face ID usage description 'Use Face ID to log in quickly' is technically ac… | [findings/seo-meta.md](findings/seo-meta.md) |
| BA-SEO-020 | ap-seo-meta-description | seo-meta | business-analyst | P3 | Admin portal has a live meta description despite being noindex/nofollow — the d… | [findings/seo-meta.md](findings/seo-meta.md) |
| BV-SEO-013 | dp-seo-html-title | seo-meta | brand-voice | P3 | <title>Fe3dr Delivery</title> is duplicated twice in apps/delivery-portal/index… | [findings/seo-meta.md](findings/seo-meta.md) |
| BV-SEO-014 | ap-seo-meta-title | seo-meta | brand-voice | P3 | Duplicate <title>Fe3dr Admin</title> tag also exists in apps/admin-portal/index… | [findings/seo-meta.md](findings/seo-meta.md) |
| BV-SEO-015 | web-seo-og-title | seo-meta | brand-voice | P3 | All title-line constructions use ASCII hyphen ' - ' between brand and tagline (… | [findings/seo-meta.md](findings/seo-meta.md) |
| TW-004 | web-seo-meta-description | seo-meta | technical-writer | P3 | Verb 'Connect with' is vague and SaaS-flavoured ('connect' is what dating apps… | [findings/seo-meta.md](findings/seo-meta.md) |
| TW-007 | web-seo-og-title | seo-meta | technical-writer | P3 | OG title (31 chars) is shorter than the recommended 60-90 char OG sweet spot. S… | [findings/seo-meta.md](findings/seo-meta.md) |
| TW-008 | web-seo-og-description | seo-meta | technical-writer | P3 | OG description is 78 chars; recommended is 100-200 chars for richer social prev… | [findings/seo-meta.md](findings/seo-meta.md) |
| TW-009 | web-seo-twitter-title | seo-meta | technical-writer | P3 | Twitter title 31 chars — below 70 char Twitter card limit. Same under-utilisati… | [findings/seo-meta.md](findings/seo-meta.md) |
| TW-010 | web-seo-twitter-description | seo-meta | technical-writer | P3 | Twitter description 78 chars — Twitter allows up to ~200 chars. Same under-util… | [findings/seo-meta.md](findings/seo-meta.md) |
| TW-014 | web-seo-app-name | seo-meta | technical-writer | P3 | `apple-mobile-web-app-title` and `application-name` are both just 'Fe3dr' — no… | [findings/seo-meta.md](findings/seo-meta.md) |
| TW-018 | ap-seo-meta-title | seo-meta | technical-writer | P3 | admin-portal correctly has `noindex, nofollow`, but the meta description is sti… | [findings/seo-meta.md](findings/seo-meta.md) |
| TW-020 | vp-seo-og-title | seo-meta | technical-writer | P3 | vendor-portal OG has no `og:image` and no Twitter card tags. Internal links sha… | [findings/seo-meta.md](findings/seo-meta.md) |
| TW-021 | web-seo-og-title | seo-meta | technical-writer | P3 | OG image tag references `/og-image.png` but has no companion `og:image:width`,… | [findings/seo-meta.md](findings/seo-meta.md) |
| TW-022 | web-seo-meta-keywords | seo-meta | technical-writer | P3 | `<meta name="keywords">` is ignored by Google since ~2009 and treated as a spam… | [findings/seo-meta.md](findings/seo-meta.md) |
| TW-023 | vp-seo-meta-keywords | seo-meta | technical-writer | P3 | Same as TW-022 — `keywords` meta on vendor-portal is deprecated SEO theatre. | [findings/seo-meta.md](findings/seo-meta.md) |

## Validation

Run from repo root to verify row count:

```bash
grep -cE "^\| (TW|LEG|BA|BV)-" docs/content-audit/AUDIT-FINDINGS.md
```

Expected: 2417 (one row per finding in the index).
