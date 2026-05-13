# Rewrite Backlog — 10 Phases

Sequenced execution backlog. Each phase is a separate `/gsd-plan-phase` invocation later.

Generated: 2026-05-13
Source: AUDIT-FINDINGS.md + findings/*.md
Total findings: 2417

## Launch Gate

- **Must ship pre-launch:** CW-01, CW-02, CW-03, CW-04
- **Strongly recommended pre-launch:** CW-05, CW-06
- **Iterative post-launch OK:** CW-07, CW-08, CW-09, CW-10

## Bundle Rules

Applied in order — first match wins. CW-08 has been extended to absorb persona-polish work (P2/P3 in `auth-onboarding` and `core-ux`) so CW-11 is empty.

| Phase | Rule |
|---|---|
| CW-01 | `category == legal` OR `lens == legal` (any severity) |
| CW-02 | (`audience == customer` OR `audience == multi`) AND `severity` in {P0, P1} AND `category` in {`auth-onboarding`, `core-ux`} |
| CW-03 | `audience == chef` AND `severity` in {P0, P1} AND `category` in {`auth-onboarding`, `core-ux`} |
| CW-04 | `audience == driver` AND `severity` in {P0, P1} AND `category` in {`auth-onboarding`, `core-ux`} |
| CW-05 | `category == marketing` |
| CW-06 | `category == transactional` |
| CW-07 | `category == errors-empty` |
| CW-08 | `category == microcopy` OR (`severity` in {P2, P3} AND `category` in {`auth-onboarding`, `core-ux`}) — **EXTENDED to absorb persona-polish work** |
| CW-09 | `category` in {`help`, `seo-meta`} |
| CW-10 | `audience == admin` OR `app == admin-portal` |
| CW-11 (overflow) | Anything else — should be 0 |

Audience inference: `web-*`/`mc-*` → customer; `vp-*`/`mv-*` → chef; `dp-*`/`md-*` → driver; `ap-*` → admin; `api-*` → multi; `null`/`pattern-*` → multi.

## Phase Index

| phase_id | title | finding_count | personas | apps | est_LOE | launch_blocker | review_required | suggested_agent |
|---|---|---|---|---|---|---|---|---|
| CW-01 | Legal launch-blockers | 711 | all | all | L | Y | lawyer | claude + legal review checkpoint |
| CW-02 | Customer signup + core UX | 53 | customer | web, mobile-customer | L | Y | PM | claude |
| CW-03 | Chef onboarding + vendor verbs | 38 | chef | vendor-portal, mobile-vendor | M | Y | PM | claude |
| CW-04 | Driver onboarding + delivery glanceable | 52 | driver | delivery-portal, mobile-delivery | M | Y | PM + design | claude |
| CW-05 | Marketing surfaces voice unification | 71 | mixed | web, mobile-customer | L | N | PM | claude + brand-voice review |
| CW-06 | Transactional content sweep | 287 | all | api | M | N | PM | claude |
| CW-07 | Errors + empty states unification | 313 | all | all | L | N | none | claude (formula-driven) |
| CW-08 | Microcopy polish + persona UX polish (P2/P3 auth-onboarding + core-ux) | 734 | all | all | XL | N | none | claude |
| CW-09 | Help + SEO/meta | 92 | customer (mostly) | web | M | N | none | claude |
| CW-10 | Admin-portal copy | 66 | admin | admin-portal | M | N | none | claude |
| CW-11 | Overflow | 0 | — | — | — | — | — | — |

## Phase Detail

### CW-01 — Legal launch-blockers

**Finding count:** 711
**Apps:** all
**Personas:** admin, chef, customer, driver, multi
**Dependencies:** Blocks public launch
**Review checkpoint:** Lawyer reviews draft text before final commit
**Estimated LOE:** L (multi-week with lawyer turnaround)
**Acceptance:** Every P0 legal finding has rewritten copy reviewed and approved by external lawyer.

**Severity breakdown:** P0=145, P1=269, P2=189, P3=108

**Sub-bundles within CW-01** (the legal phase is large; bundle by regulatory theme to make lawyer review tractable):

- CW-01a: Missing legal pages (T&C, Privacy, Refund, Cookie) — implement first
- CW-01b: DPDP Act 2023 consent flows (signup, KYC, cookie banner, grievance officer)
- CW-01c: FSSAI compliance (license display, allergens, food-safety claims)
- CW-01d: RBI Payment Aggregator rules (refund timeline, escrow, merchant disclosure)
- CW-01e: GST invoicing (order confirmation, receipts, payout statements)
- CW-01f: Cross-cutting (brand identity unification, grievance officer, footer disclosures, retention policies)

**Top 10 most-flagged surfaces:**
- `cross-cutting-transactional` — 29 findings
- `api-email-order-confirm` — 8 findings
- `api-email-base-footer` — 7 findings
- `vp-legal-policy-tos` — 6 findings
- `mv-onb-policies-terms-body` — 6 findings
- `web-mkt-landing-why-choose` — 6 findings
- `web-auth-register-form-fields` — 6 findings
- `vp-legal-docs-banner-body` — 5 findings
- `vp-legal-policy-hygiene` — 5 findings
- `vp-legal-settings-stripe` — 5 findings

### CW-02 — Customer signup + core UX

**Finding count:** 53
**Apps:** web, mobile-customer, api (multi-audience)
**Personas:** customer
**Dependencies:** Blocks public launch (customer-facing conversion funnel)
**Review checkpoint:** PM signs off; brand-voice review for hero/landing copy
**Estimated LOE:** L
**Acceptance:** All P0/P1 customer-facing auth-onboarding and core-ux findings rewritten and verified in QA.

**Severity breakdown:** P0=17, P1=36

**Top 10 most-flagged surfaces:**
- `web-ux-layout-nav` — 9 findings
- `web-ux-orders-status-labels` — 5 findings
- `mc-order-detail-status-labels` — 3 findings
- `web-ux-checkout-summary` — 3 findings
- `web-auth-register-heading` — 2 findings
- `web-ux-cart-checkout-cta` — 2 findings
- `mc-checkout-place-order` — 2 findings
- `web-ux-checkout-delivery-time` — 2 findings
- `web-ux-checkout-tip` — 2 findings
- `mc-catering-form-labels` — 2 findings

### CW-03 — Chef onboarding + vendor verbs

**Finding count:** 38
**Apps:** vendor-portal, mobile-vendor
**Personas:** chef
**Dependencies:** Blocks public launch (chef supply side cannot onboard)
**Review checkpoint:** PM signs off
**Estimated LOE:** M
**Acceptance:** All P0/P1 chef-facing auth-onboarding and core-ux findings rewritten, vendor verbs consistent across web + mobile.

**Severity breakdown:** P0=8, P1=30

**Top 10 most-flagged surfaces:**
- `vp-auth-register-hero-sub` — 4 findings
- `vendor-portal-onboarding-step-documents` — 3 findings
- `vp-ux-settings-order-acceptance` — 3 findings
- `vendor-portal-register-benefits` — 2 findings
- `vendor-portal-onboarding-step-policies` — 2 findings
- `vp-auth-login-features-list` — 2 findings
- `vendor-portal-register-heading` — 1 findings
- `vendor-portal-register-tos` — 1 findings
- `vendor-portal-login-access-denied` — 1 findings
- `vendor-portal-onboarding-header` — 1 findings

### CW-04 — Driver onboarding + delivery glanceable

**Finding count:** 52
**Apps:** delivery-portal, mobile-delivery
**Personas:** driver
**Dependencies:** Blocks public launch (driver fulfilment side)
**Review checkpoint:** PM + design (glanceable copy density matters in-motion)
**Estimated LOE:** M
**Acceptance:** All P0/P1 driver-facing auth-onboarding and core-ux findings rewritten; status taxonomy unified with API canonical labels.

**Severity breakdown:** P0=18, P1=34

**Top 10 most-flagged surfaces:**
- `dp-ux-active-cancel-prompt` — 3 findings
- `dp-auth-step4-zero-commission` — 2 findings
- `dp-auth-step5-terms` — 2 findings
- `md-auth-002` — 2 findings
- `dp-auth-onboarding-step-labels` — 2 findings
- `delivery-portal-onboarding-status` — 2 findings
- `md-core-033` — 2 findings
- `dp-ux-active-mark-as` — 2 findings
- `dp-ux-delivery-pickup-label` — 2 findings
- `md-core-079` — 2 findings

### CW-05 — Marketing surfaces voice unification

**Finding count:** 71
**Apps:** web (marketing routes), mobile-customer
**Personas:** mixed
**Dependencies:** Strongly recommended pre-launch — fake metric and unverified social-proof claims are DPDP / Consumer Protection Act exposure
**Review checkpoint:** PM + brand-voice review
**Estimated LOE:** L
**Acceptance:** Hardcoded social-proof metrics removed or gated behind live data; tagline/hero copy aligned to brand-voice rubric.

**Severity breakdown:** P0=21, P1=14, P2=19, P3=17

**Top 10 most-flagged surfaces:**
- `web-mkt-landing-why-choose` — 7 findings
- `web-mkt-landing-become-chef` — 7 findings
- `vp-mkt-register-benefits-list` — 5 findings
- `web-mkt-register-benefits` — 4 findings
- `web-mkt-hero-variants` — 4 findings
- `web-mkt-landing-featured-chefs-heading` — 4 findings
- `web-mkt-landing-catering-cta` — 4 findings
- `web-mkt-landing-hero-badge` — 3 findings
- `web-mkt-landing-trust-badges` — 3 findings
- `web-mkt-login-testimonial` — 3 findings

### CW-06 — Transactional content sweep

**Finding count:** 287
**Apps:** api (emails, push, SMS templates)
**Personas:** all
**Dependencies:** Strongly recommended pre-launch — routing bugs send wrong-audience copy
**Review checkpoint:** PM
**Estimated LOE:** M
**Acceptance:** Every transactional template uses canonical `emailBase` wrapper, single brand name, status taxonomy consistent with frontends, no fake metrics.

**Severity breakdown:** P0=37, P1=43, P2=145, P3=62

**Top 10 most-flagged surfaces:**
- `api-success-message-generic` — 10 findings
- `api-email-order-confirm` — 7 findings
- `api-email-welcome` — 6 findings
- `api-email-chef-verified` — 6 findings
- `api-email-chef-new-order` — 5 findings
- `api-email-delivery-assigned` — 5 findings
- `api-email-support-created` — 5 findings
- `api-push-order-update-customer-deeplink` — 5 findings
- `api-push-chef-new-order-actionable` — 5 findings
- `api-email-account-reminder` — 4 findings

### CW-07 — Errors + empty states unification

**Finding count:** 313
**Apps:** all
**Personas:** all
**Dependencies:** Iterative post-launch OK — drive via formula library
**Review checkpoint:** none (formula-driven rewrite once style guide error-pattern is locked)
**Estimated LOE:** L (volume) but each rewrite is S
**Acceptance:** Every error/empty state follows the [what happened] + [what to do] + [where to go] formula from STYLE-GUIDE.md.

**Severity breakdown:** P0=22, P1=43, P2=128, P3=120

**Top 10 most-flagged surfaces:**
- `web-err-login-generic` — 4 findings
- `web-ux-cart-empty` — 4 findings
- `mc-checkout-errors` — 4 findings
- `api-error-payment-config` — 3 findings
- `api-error-stripe-connect` — 3 findings
- `api-error-order-cancel-stage` — 3 findings
- `api-error-upload-size` — 3 findings
- `web-err-error-boundary` — 3 findings
- `api-error-auth-suspended` — 3 findings
- `api-error-delivery-already-active` — 3 findings

### CW-08 — Microcopy polish + persona UX polish (P2/P3 auth-onboarding + core-ux)

**Finding count:** 734 (264 microcopy + 470 persona P2/P3 polish)
**Apps:** all
**Personas:** all
**Dependencies:** Iterative post-launch OK
**Review checkpoint:** none
**Estimated LOE:** XL (volume) but each rewrite is S–M; split execution across sub-bundles
**Acceptance:** Buttons, labels, helpers, tooltips, toasts conform to style-guide vocabulary (sentence case, allowed verbs, banned-vocab eliminated). All P2/P3 auth-onboarding and core-ux findings rewritten per persona conventions (customer / chef / driver).

**Severity breakdown (combined):**
- Microcopy slice: P0=14, P1=46, P2=55, P3=149
- Persona-polish slice: P2=244 (auth-onboarding 168 + core-ux 76), P3=226 (core-ux 162 + auth-onboarding 64)
- Combined: P0=14, P1=46, P2=299, P3=375

**Why this phase is extended:** Earlier the 470 P2/P3 auth-onboarding/core-ux findings fell into a CW-11 overflow bucket because CW-02/03/04 gate on P0/P1. Rather than ship persona polish as a stand-alone overflow phase, CW-08 absorbs them so the post-launch polish work is one coherent bundle. Execution is split into four sub-bundles below to keep each chunk tractable.

**Sub-bundles within CW-08** (execute sub-bundles in parallel; each is independent):

#### CW-08a — Microcopy slice (cross-persona)

**Count:** 264
**Scope:** `category == microcopy` findings. Buttons, labels, helpers, tooltips, toasts across all apps. Formula-driven against STYLE-GUIDE.md vocabulary.
**Severity:** P0=14, P1=46, P2=55, P3=149
**Top 10 surfaces:**
- `web-ux-cart-promo` — 4 findings
- `vp-ux-reviews-reply` — 4 findings
- `dp-mc-portal-footer` — 4 findings
- `web-ux-chef-card-favorite-toast-loggedout` — 3 findings
- `web-ux-orders-search` — 3 findings
- `vp-ux-notifs-sla-line` — 3 findings
- `vp-ux-notifs-respond-form` — 3 findings
- `vp-onb-kitchen-desc-field` — 3 findings
- `vp-onb-ops-min-order` — 3 findings
- `vp-onb-policies-payout-info` — 3 findings

#### CW-08b — Driver persona polish (P2/P3 auth-onboarding + core-ux)

**Count:** 205
**Scope:** `audience == driver` AND `severity` in {P2, P3} AND `category` in {`auth-onboarding`, `core-ux`}.
**Apps:** delivery-portal, mobile-delivery
**Personas:** driver
**Acceptance:** Driver-facing P2/P3 auth-onboarding and core-ux copy aligned with glanceable density rules (44–48px touch targets, short verb-led labels, status taxonomy from STYLE-GUIDE.md).
**Top 10 surfaces:**
- `delivery-portal-onboarding-step-progress` — 3 findings
- `delivery-portal-onboarding-review` — 3 findings
- `dp-ux-partner-performance` — 3 findings
- `md-auth-002` (P2/P3 residuals) — 2 findings
- `md-onb-018` — 2 findings
- `md-onb-109` — 2 findings
- `dp-auth-onboarding-header` — 2 findings
- `dp-auth-status-descriptions` — 2 findings
- `dp-auth-step4-h` — 2 findings
- `dp-auth-step4-secure-pay` — 2 findings

#### CW-08c — Chef persona polish (P2/P3 auth-onboarding + core-ux)

**Count:** 119
**Scope:** `audience == chef` AND `severity` in {P2, P3} AND `category` in {`auth-onboarding`, `core-ux`}.
**Apps:** vendor-portal, mobile-vendor
**Personas:** chef
**Acceptance:** Chef-facing P2/P3 auth-onboarding and core-ux copy aligned with vendor-verb consistency rules and onboarding stepper conventions in STYLE-GUIDE.md.
**Top 10 surfaces:**
- `vp-ux-kitchen-photos` — 4 findings
- `vp-onb-personal-fields` — 3 findings
- `vendor-portal-onboarding-step-kitchen` — 3 findings
- `vendor-portal-onboarding-step-operations` — 3 findings
- `vendor-portal-onboarding-step-policies` — 3 findings
- `vp-ux-menu-view-page` — 3 findings
- `mv-undo-cta` — 3 findings
- `vp-onb-ops-pricing-title` — 2 findings
- `vp-onb-stepper-labels` — 2 findings
- `vp-auth-login-hero-heading` — 2 findings

#### CW-08d — Customer + multi persona polish (P2/P3 auth-onboarding + core-ux)

**Count:** 146 (customer 113 + multi 33)
**Scope:** (`audience == customer` OR `audience == multi`) AND `severity` in {P2, P3} AND `category` in {`auth-onboarding`, `core-ux`}.
**Apps:** web, mobile-customer, cross-cutting
**Personas:** customer, multi
**Acceptance:** Customer-facing P2/P3 auth-onboarding and core-ux copy aligned with customer voice conventions; multi-audience cross-cutting patterns unified to canonical brand surface.
**Top 10 surfaces:**
- `web-auth-onboarding-heading` — 5 findings
- `web-ux-orderdetail-sections` — 4 findings
- `web-auth-login-email-fields` — 3 findings
- `web-auth-login-heading` — 3 findings
- `web-auth-register-heading` — 3 findings
- `web-auth-onboarding-step-address` — 3 findings
- `mc-profile-more-rows` — 3 findings
- `mc-onb-step1-labels` — 2 findings
- `web-auth-login-submit` — 2 findings
- `web-auth-onboarding-step-basic` — 2 findings

### CW-09 — Help + SEO/meta

**Finding count:** 92
**Apps:** web (primarily)
**Personas:** customer (mostly)
**Dependencies:** Iterative post-launch OK; SEO/meta affects organic acquisition once public
**Review checkpoint:** none
**Estimated LOE:** M
**Acceptance:** Help articles and meta-title/description for every public route conform to style-guide; structured data validated.

**Severity breakdown:** P0=6, P1=22, P2=37, P3=27

**Top 10 most-flagged surfaces:**
- `web-seo-og-title` — 7 findings
- `web-seo-meta-description` — 6 findings
- `dp-help-status-contact` — 4 findings
- `web-seo-og-description` — 4 findings
- `dp-seo-html-title` — 4 findings
- `ap-seo-meta-title` — 4 findings
- `web-help-otto-launcher` — 3 findings
- `web-help-otto-empty` — 3 findings
- `dp-help-new-driver` — 3 findings
- `dp-help-new-driver-body` — 3 findings

### CW-10 — Admin-portal copy

**Finding count:** 66
**Apps:** admin-portal
**Personas:** admin
**Dependencies:** Iterative post-launch OK — internal ops only
**Review checkpoint:** none
**Estimated LOE:** M
**Acceptance:** Admin copy (titles, filters, status maps, action buttons) unified to title-case admin convention and consistent with canonical taxonomies.

**Severity breakdown:** P0=4, P1=8, P2=10, P3=44

**Top 10 most-flagged surfaces:**
- `ap-providerdetail-status-fields` — 3 findings
- `ap-secsettings-2fa-account` — 3 findings
- `ap-secsettings-apikey-fields` — 3 findings
- `ap-layout-nav-reviews` — 3 findings
- `ap-notifsettings-categories` — 3 findings
- `ap-auth-login-feature-list` — 2 findings
- `ap-auth-login-heading` — 2 findings
- `ap-auth-login-instruction` — 2 findings
- `ap-auth-login-brand-sub` — 2 findings
- `ap-delivery-statuses` — 2 findings

### CW-11 — Overflow

**Finding count:** 0

The bundle rules now route every finding into CW-01 through CW-10. CW-08's extension (P2/P3 in `auth-onboarding` or `core-ux`) absorbs the 470 findings that previously overflowed. CW-11 is intentionally empty and exists only as a sentinel — if a future ingestion adds findings that don't match any rule, they will land here and surface as a triage signal.

## Bundle distribution validation

Sum of finding counts across all phases: **2417** (target: 2417).

Sum check passes. CW-11 overflow is **0**.

Per-phase counts:

| phase_id | count |
|---|---|
| CW-01 | 711 |
| CW-02 | 53 |
| CW-03 | 38 |
| CW-04 | 52 |
| CW-05 | 71 |
| CW-06 | 287 |
| CW-07 | 313 |
| CW-08 | 734 |
| CW-09 | 92 |
| CW-10 | 66 |
| CW-11 | 0 |
| **Total** | **2417** |

CW-08 sub-bundle counts:

| sub_bundle | count |
|---|---|
| CW-08a (microcopy) | 264 |
| CW-08b (driver P2/P3) | 205 |
| CW-08c (chef P2/P3) | 119 |
| CW-08d (customer + multi P2/P3) | 146 |
| **CW-08 total** | **734** |

Category coverage check — all 9 categories represented:

| category | covered by |
|---|---|
| legal | CW-01 |
| auth-onboarding | CW-02 / CW-03 / CW-04 (P0/P1), CW-08 (P2/P3) |
| core-ux | CW-02 / CW-03 / CW-04 (P0/P1), CW-08 (P2/P3) |
| marketing | CW-05 |
| transactional | CW-06 |
| errors-empty | CW-07 |
| microcopy | CW-08 |
| help | CW-09 |
| seo-meta | CW-09 |

Admin-portal findings are routed to CW-10 by the `audience == admin` OR `app == admin-portal` rule, which runs after the category-keyed rules.

## How to execute a phase

Each phase becomes its own `/gsd-plan-phase` invocation:

```
/gsd-plan-phase CW-01
```

The phase plan loads REWRITE-BACKLOG.md, the relevant findings from `docs/content-audit/findings/`, and `docs/content-audit/STYLE-GUIDE.md`, and produces a per-finding rewrite plan.

For CW-08, plan each sub-bundle independently:

```
/gsd-plan-phase CW-08a   # microcopy slice
/gsd-plan-phase CW-08b   # driver persona polish
/gsd-plan-phase CW-08c   # chef persona polish
/gsd-plan-phase CW-08d   # customer + multi persona polish
```

## Scoring formula (informational)

`score = severity_weight × impact_weight ÷ effort_weight`

- severity: P0=8, P1=4, P2=2, P3=1
- impact: customer-facing=3, chef/driver-facing=2, admin/internal=1
- effort: S(<1h)=1, M(1-4h)=2, L(>4h)=4

Effort is not tracked per finding in the audit and defaults to M(2) when computing scores.
