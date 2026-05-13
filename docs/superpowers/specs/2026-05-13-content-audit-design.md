# Content Audit & Rewrite Plan — Home Chef Apps

**Date:** 2026-05-13
**Status:** Design (pre-implementation-plan)
**Owner:** Mahesh Sangawar

## 1. Goal

Produce a complete content audit and prioritized rewrite backlog covering every user-facing static-content surface across all 7 Home Chef apps + the Go API's user-facing strings. The audit must apply four expert lenses (technical writer, legal expert, business analyst, brand voice) and produce a sequenced execution plan that downstream phases can consume.

**The deliverable of this effort is a set of four planning artifacts, not rewritten content.** Actual content rewrites become separate `/gsd-plan-phase` runs that consume the backlog.

### Drivers

All four triggers apply equally:
- **Launch readiness** — polish, gap-fill, and consistency before public launch
- **Legal / compliance exposure** — India regulatory landscape (DPDP Act 2023, FSSAI, RBI payment-aggregator, GST) + generic best-practice
- **Brand voice cleanup** — surfaces sound like different products today
- **Conversion / activation** — drop-off in signup, checkout, chef onboarding suspected to be partly copy-driven

### Non-Goals

Explicitly excluded from this effort:

- Actual content rewrites (those are downstream phases CW-01..CW-10)
- Translation or i18n setup (the style guide will note translation-readiness, no translation work happens)
- Drafting new legal documents from scratch — audit + recommendations only; a lawyer drafts the binding text
- Visual design or design-token changes (governed by `.impeccable.md`)
- Data-model changes (e.g., adding `allergens` field to product schema is a separate phase)
- US / EU / UK jurisdictional legal audit (out of scope per user direction)
- SEO growth strategy (we audit existing metadata for clarity; growth strategy is a separate effort)
- Accessibility code changes (prior a11y sweep already completed; this audit covers language accessibility only)

## 2. Surface Area

### Apps in scope

| App | Path | Persona | File count (approx) |
|---|---|---|---|
| Customer Web | `apps/web/` | Customer | 108 TSX/TS |
| Vendor Portal | `apps/vendor-portal/` | Chef | 68 TSX/TS |
| Delivery Portal | `apps/delivery-portal/` | Driver | 48 TSX/TS |
| Admin Portal | `apps/admin-portal/` | Platform admin | 45 TSX/TS |
| Mobile Customer | `apps/mobile-customer/` | Customer | 20 TSX/TS |
| Mobile Vendor | `apps/mobile-vendor/` | Chef | 26 TSX/TS |
| Mobile Delivery | `apps/mobile-delivery/` | Driver | 25 TSX/TS |
| Go API | `apps/api/` | All (via emails / push / errors) | Go handlers + templates |

### Content categories (9 buckets)

| Category | What it covers |
|---|---|
| `legal` | T&C, Privacy, Refund/Cancellation, Cookie, Acceptable Use, Chef Agreement, Driver Agreement, Allergen/Food-safety disclosures |
| `marketing` | Landing/home, About, How-it-works, For-Chefs, For-Drivers, Pricing, FAQ |
| `auth-onboarding` | Sign up, Login, MFA, Password reset, OAuth consent screens, Chef onboarding wizard, Driver verification, Email-OTP |
| `core-ux` | Browse, search, product detail, cart, checkout, order tracking, menu builder, order management, delivery navigation |
| `errors-empty` | Validation errors, system errors (4xx/5xx), empty states, offline states, loading states |
| `transactional` | Email templates, push notifications, SMS, in-app toasts/banners — backend-generated |
| `microcopy` | Tooltips, helper text, placeholders, success confirmations, button labels, modal subtitles |
| `help` | FAQ articles, contact, support flows, in-app hints, "learn more" links |
| `seo-meta` | `<title>`, `<meta description>`, OG tags, structured data, sitemap copy |

## 3. Approach

**Approach B — Content-category × all-apps (horizontal slice), with parallel expert-lens subagents per category.**

Rationale:
- Brand-voice unification requires side-by-side comparison of the same content type across apps
- Legal pages exist in 3+ apps; auditing them together is more efficient and consistent
- Style Guide + Prioritized Backlog deliverables fall naturally out of category-level analysis
- Each category becomes one coherent downstream execution phase

Rejected alternatives: app-by-app (vertical slice) loses cross-app consistency; persona-journey audit fragments shared infrastructure.

## 4. Architecture: 4-phase pipeline

```
Phase 1 — CONTENT-INVENTORY.md        (catalog what exists)
            │
            ▼
Phase 2 — STYLE-GUIDE.md              (define the voice to write against)
            │
            ▼
Phase 3 — AUDIT-FINDINGS.md           (apply 4 lenses × 9 categories)
            │
            ▼
Phase 4 — REWRITE-BACKLOG.md          (prioritized phases for execution)
```

All four artifacts live under `docs/content-audit/` and commit independently.

### Phase 1 — Content Inventory

**Goal:** every piece of user-facing static content has one row in a table.

**Method:**
1. Dispatch 7 parallel `Explore` subagents (one per app) to extract every string, label, page, email template, push payload, and Go error message
2. Dedupe via shared components (`@tesserix/web`, shared layouts)
3. Group into a single `CONTENT-INVENTORY.md` with one section per category and one sub-section per app
4. Spot-check coverage by sampling 5 random routes per app

**Inventory row schema:**

```
surface_id | app | category | route_or_file | audience | word_count |
current_text_excerpt | owns_voice? | last_edited | notes
```

**Estimated size:** ~600–1,500 rows total.

**Excluded from inventory:** server logs, internal admin debug strings, code comments, alt-text on decorative icons (covered by prior a11y sweep), DevTools strings.

### Phase 2 — Style Guide

**Goal:** a single source of truth a future writer (or LLM) can apply to any surface to produce on-brand copy. Anchored to `.impeccable.md`.

**`STYLE-GUIDE.md` structure:**

1. **Voice principles** — 5 short rules with do/don't examples, derived from `.impeccable.md` (Confident · Appetizing · Quietly modern)
2. **Persona tone matrix:**
   - Customer: warmer, sensory verbs, 2nd person
   - Chef/Vendor: functional, time-aware, no marketing fluff
   - Driver: glanceable, imperative, ≤4 words where possible
   - Admin: neutral operator language
3. **Vocabulary list** — 60–100 preferred/banned term pairs (`Sign in` vs `Log in/Login`, `Home chef` vs `Cook/Vendor/Seller`, `Pickup` vs `Pick-up/Pick up`, currency/date/distance formats)
4. **Microcopy patterns** — formulas for buttons, errors, empty states, success toasts, form labels
5. **Legal-page tone** — plain language, short sentences, headings every 200 words, "we"/"you" not "the Company"/"the User", allergen and refund in callout boxes
6. **Numerals & formatting** — tabular figures, currency, 24h internal / 12h customer-facing, relative vs absolute time thresholds
7. **Internationalization readiness flag** — each rule notes translation survival

**Not in the style guide:** visual tokens (`.impeccable.md`), specific page copy (Phase 3+), code conventions.

### Phase 3 — Multi-Lens Audit

**Goal:** every inventory row scored by all four lenses. Output: `AUDIT-FINDINGS.md` keyed by `surface_id`, with per-category detail in `docs/content-audit/findings/*.md`.

**The four lenses (parallel subagents per category):**

| Lens | Agent | What it checks |
|---|---|---|
| Technical Writer | `claude` (general-purpose) with TW brief | Clarity, sentence length, voice-drift, jargon, microcopy formula compliance, language accessibility (Flesch reading ease ≥60 customer-facing, ≥50 vendor/admin), tone-persona match |
| Legal | `claude` with Legal brief (India + generic) | DPDP Act 2023 consent/notice gaps, FSSAI allergen disclosure, RBI payment-aggregator T&C, GST invoice copy, refund clarity, gig-worker terms for drivers, cookie/tracking disclosure, jurisdiction & governing-law clauses, plain-language test |
| Business Analyst | `business-analyst` agent | Conversion-critical CTAs, value-prop clarity, friction language, drop-off zones, missing trust signals, pricing transparency, empty-state CTAs |
| Brand Voice | `claude` with `.impeccable.md` brief | Cross-app consistency, drift from "Confident · Appetizing · Quietly modern", banned vocab/phrasing, terracotta-era legacy copy ("artisanal", "handcrafted with love"), urgency-trick patterns |

**Per-category run pattern:**

1. Prepare brief: category inventory slice + style guide + lens-specific rubric
2. Dispatch all 4 agents in parallel (one message, four agent calls)
3. Merge into the category's findings file
4. Move to next category

**Finding row schema:**

```
finding_id | surface_id | lens | severity | issue | evidence_excerpt |
recommendation | depends_on (e.g., needs lawyer review)
```

**Severity scale:**

- **P0 — Launch-blocker:** Legal/regulatory exposure (missing DPDP consent, missing allergen disclosure, missing GST line), factually-wrong copy (broken commitment, wrong refund window)
- **P1 — Launch-recommended:** Conversion-critical or trust-eroding (unclear pricing, broken CTA, missing food-safety badge, voice drift on landing/checkout)
- **P2 — Polish:** Microcopy inconsistencies, tone drift on secondary surfaces, error-message formula violations
- **P3 — Nice-to-have:** Tooltip improvements, help-content gaps, SEO description tweaks

**Estimated finding count:** ~400–800 across the inventory.

**Conflict resolution between lenses:** Legal wins on legal pages; Brand Voice wins on marketing; both reconcile via style-guide rule "plain language in legal".

**Out of scope for lenses:** translation quality, code-level a11y, visual design issues.

### Phase 4 — Prioritized Backlog

**Goal:** turn ~400–800 findings into a sequenced list of execution phases. Each phase becomes one `/gsd-plan-phase` later.

**Scoring formula per finding:**

```
score = severity_weight × impact_weight ÷ effort_weight

severity:  P0=8  P1=4  P2=2  P3=1
impact:    customer-facing=3  chef/driver-facing=2  admin/internal=1
effort:    S(<1h)=1  M(1–4h)=2  L(>4h)=4
```

**Phase bundling:** findings bucketed by `category + persona + severity tier`, not by raw score (a 600-item flat list is unusable; 30–60-item bundles are executable).

**Suggested phase sequence:**

| Phase | Bundle | Why this order |
|---|---|---|
| CW-01 | Legal launch-blockers (all P0 legal findings) | Regulatory exposure non-negotiable; lawyer review loop needed |
| CW-02 | Customer signup + checkout copy (P0/P1, customer) | Highest conversion impact; demonstrates style guide in practice |
| CW-03 | Chef onboarding + vendor-portal core verbs (P0/P1, chef) | Supply-side activation; unblocks marketplace growth |
| CW-04 | Driver onboarding + delivery glanceable copy (P0/P1, driver) | Operational safety + driver retention |
| CW-05 | Marketing surfaces (P1, brand/BA) — landing, How-it-works, For-Chefs, For-Drivers | Voice unification across entry points |
| CW-06 | Transactional sweep (P0/P1) — emails, push, SMS, in-app toasts | Backend-generated; touches every persona; often most outdated |
| CW-07 | Errors + empty states unification (P1/P2 across all apps) | High consistency dividend |
| CW-08 | Microcopy polish (P2) — tooltips, helper text, success messages | Long tail; partly LLM-assisted with style guide as prompt |
| CW-09 | Help/FAQ + SEO/meta (P2/P3) | Lower-urgency surface completion |
| CW-10 | Admin-portal copy (P2/P3, internal) | Last because internal users tolerate imperfection |

**Backlog row schema:**

```
phase_id | title | finding_ids[] | personas[] | apps[] |
estimated_LOE | dependencies | launch_blocker (Y/N) |
review_required (lawyer/PM/design) | suggested_agent
```

**Launch gate vs post-launch split:**

- CW-01 through CW-04: must ship before public launch
- CW-05, CW-06: strongly recommended pre-launch
- CW-07 onward: iterative post-launch

**Special — legal-review loop.** CW-01 phases produce draft copy; a human lawyer must review before merge. Each CW-01 phase plan includes a checkpoint to export rewritten clauses for external lawyer review before commit. `review_required: lawyer` is the gate.

## 5. Execution

| Step | Who | Output |
|---|---|---|
| 1. Inventory sweep | Me + 7 parallel `Explore` subagents (one per app) | `CONTENT-INVENTORY.md` |
| 2. Style guide draft | Me (synthesize `.impeccable.md` + India locale + persona matrix) | `STYLE-GUIDE.md` |
| 3. Lens audits (4 lenses × 9 categories = 36 agent runs) | 4 parallel subagents per category, 9 sequential categories | `AUDIT-FINDINGS.md` + `findings/*.md` |
| 4. Backlog synthesis | Me (apply scoring formula, bundle into 10 phases) | `REWRITE-BACKLOG.md` |
| 5. Hand-off | Invoke `writing-plans` skill on `REWRITE-BACKLOG.md` | Implementation plan for CW-01 + template for remaining phases |

**Repo layout for deliverables:**

```
docs/content-audit/
├── README.md                  # how to read this audit
├── CONTENT-INVENTORY.md       # Phase 1
├── STYLE-GUIDE.md             # Phase 2
├── AUDIT-FINDINGS.md          # Phase 3 (rolled-up)
├── findings/                  # per-category detail
│   ├── legal.md
│   ├── marketing.md
│   ├── auth-onboarding.md
│   ├── core-ux.md
│   ├── errors-empty.md
│   ├── transactional.md
│   ├── microcopy.md
│   ├── help.md
│   └── seo-meta.md
└── REWRITE-BACKLOG.md         # Phase 4
```

**Commits:** ~14 — one per artifact, plus one per category findings file.

**Time/effort estimate (planning stage only):** ~6–10 hours of agent work + synthesis, across 1–2 sessions.

## 6. Risks & Mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| Inventory misses surfaces hidden in dynamic strings (e.g., string interpolation, `t()` calls without literals) | Medium | Sample-check 5 routes per app post-inventory; add a "discovered late" addendum |
| Legal lens hallucinates India regulatory details (DPDP Act / FSSAI specifics) | Medium-High | Mark every legal P0 finding as `review_required: lawyer`; never treat lens output as legally binding |
| 4-lens findings contradict each other on the same surface | Medium | Style-guide tone rules pre-resolve common conflicts; remaining conflicts flagged in `AUDIT-FINDINGS.md` for manual resolution |
| Style guide too generic to be enforceable | Medium | Vocabulary list with concrete preferred/banned pairs is the testable artifact; microcopy formulas are linter-able |
| Backlog scoring produces unintuitive ordering | Low | Sequence is `category + persona + severity` bundled, not raw score; manual review of the 10-phase order before commit |
| Scope creep into actual rewrites mid-audit | Medium | Non-goals section is explicit; any rewrite work spins out into a CW-NN phase via writing-plans |

## 7. Success Criteria

The audit is done when:

1. `CONTENT-INVENTORY.md` rows cover every visible static string across all 7 apps + API-generated content
2. `STYLE-GUIDE.md` is concrete enough that a copywriter or LLM can rewrite any surface and produce demonstrably on-brand output
3. `AUDIT-FINDINGS.md` has at least one finding per high-traffic surface (landing, signup, checkout, T&C, refund policy, key error states); P0 findings all flag legal-review requirement where applicable
4. `REWRITE-BACKLOG.md` is sequenced such that `/gsd-plan-phase CW-01` can be invoked immediately on the first phase
5. The user (and a future writer joining the project) can read the four artifacts in order and understand the full content landscape without reading code

## 8. Open Questions

None as of design freeze. The four scoping questions (motivation, app scope, deliverable shape, legal jurisdiction) are resolved.

Optional refinements that can happen during execution without changing the design:

- Adding allergen-vocabulary subsection to the style guide if FSSAI labeling audit surfaces consistent gaps
- Splitting `transactional` findings into `email` vs `push/SMS` if volume warrants
- Adding a `chef-name capitalization` rule to the style guide if real chef names surface formatting questions

## 9. Next Step

After user approval of this design, invoke the `writing-plans` skill on this spec to produce a step-by-step implementation plan for the audit itself (the 5-step execution pipeline above).
