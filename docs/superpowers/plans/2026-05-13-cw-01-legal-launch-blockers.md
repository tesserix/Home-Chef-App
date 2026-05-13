# CW-01 — Legal Launch-Blockers Execution Plan

> **Phase ID:** CW-01 (Legal launch-blockers)
> **Source:** `docs/content-audit/REWRITE-BACKLOG.md`
> **Findings count:** 711 (legal-flag findings across all categories)
> **Launch blocker:** Yes
> **Review gate:** Lawyer review required before merge of any binding text

**Goal:** Eliminate the regulatory exposure surfaced by the legal lens during the content audit, by adding missing legal pages, fixing contradictions, and aligning consent flows with India regulatory requirements (DPDP Act 2023, FSSAI, RBI PA, GST).

**Architecture:** Six sub-bundles (CW-01a..CW-01f) executed in dependency order. CW-01a (missing pages) ships first because subsequent bundles link to those pages. Every binding-text draft flagged `DRAFT — REQUIRES LAWYER REVIEW BEFORE PUBLICATION` until a real lawyer reviews and signs off.

**Tech Stack:** React 19 / Next.js-style routing (apps/web is Vite + React Router v7), TypeScript, Tailwind, `@tesserix/web` design system.

---

## Sub-bundle structure

| Sub-bundle | Title | Scope | Status this session |
|---|---|---|---|
| CW-01a | Missing legal pages | Create T&C, Privacy, Refund, Cookie pages on apps/web; wire routing; update existing link targets | **Drafting this session** |
| CW-01b | DPDP consent infrastructure | Add granular consent UI to signup, grievance officer disclosure, cookie banner | **Starting this session** |
| CW-01c | FSSAI compliance | Resolve FSSAI Optional/Required contradiction, add license display, add allergen disclosure schema | Future session |
| CW-01d | RBI Payment Aggregator | Refund timeline disclosure, escrow/merchant disclosure on checkout | Future session |
| CW-01e | GST invoicing | Order confirmation email GST line, HSN/SAC, downloadable invoice | Future session |
| CW-01f | Cross-cutting | Brand identity unification (Fe3dr → consistent), grievance officer in footers, governing-law clauses | Future session |

## Session deliverables (what I'm doing now)

### CW-01a — Missing legal pages (this session)

Four new pages on apps/web with DRAFT copy:

1. **`/terms` → `TermsPage.tsx`** — Platform Terms of Service for customers (and chef/driver agreements linked separately)
2. **`/privacy` → `PrivacyPolicyPage.tsx`** — DPDP Act 2023-aligned privacy notice
3. **`/refund` → `RefundPolicyPage.tsx`** — Refund + cancellation policy with explicit timelines
4. **`/cookies` → `CookiePolicyPage.tsx`** — Cookie + tracking disclosure

Each page:
- Lives at `apps/web/src/features/legal/pages/<PageName>.tsx`
- Uses MainLayout shell (header + footer like other pages)
- Renders draft legal content per the audit findings
- Includes a prominent "DRAFT — REQUIRES LAWYER REVIEW" banner until reviewed
- Plain-language tone per `STYLE-GUIDE.md` §5
- Callout boxes for high-stakes clauses (allergens, refunds, dispute)

Routes wired into `apps/web/src/app/routes/index.tsx` inside the public/MainLayout section.

### CW-01b start — DPDP consent (this session)

Limited scope first pass:
1. Update `RegisterPage.tsx` consent text to be granular DPDP-compliant (purpose-specific, not bundled)
2. Add a `<CookieBanner />` shared component, mounted in `MainLayout`
3. Add grievance officer + data fiduciary identity to the footer + legal pages

The full DPDP consent rewrite (sensitive-data consent, withdrawal UI, retention disclosure, children's age gate) is a future session.

## Important notes

- **All copy is DRAFT.** A licensed India advocate must review and sign off before this ships to production. Every page has a "DRAFT — UNDER LEGAL REVIEW" banner that the lawyer's sign-off removes.
- **Brand identity drift** — I'll use `Home Chef` as the canonical brand for this draft pass (per CLAUDE.md project context says "Home Chef Mobile Apps"). The Fe3dr/HomeChef unification decision belongs in CW-01f.
- **No PII or live data** is collected by these pages; they're static informational.
- **Routes mount inside MainLayout** so existing footer link targets light up.

## Out of scope this session

- CW-01c, CW-01d, CW-01e, CW-01f (future sessions)
- Lawyer-engagement workflow
- Vendor T&C / Driver Agreement (covered in CW-03/CW-04)
- Mobile app legal pages (CW-01a covers apps/web; mobile in future session)
- Backend changes (no API contract changes; pages are static React)

## Acceptance for the work being done this session

1. `/terms`, `/privacy`, `/refund`, `/cookies` routes return real pages with substantive draft legal copy (not "TODO" placeholders)
2. Each page has the DRAFT banner
3. Pages link to each other where relevant (e.g., Privacy mentions Cookie Policy)
4. RegisterPage consent text references actual T&C link (not a dead link)
5. MainLayout footer Privacy/Terms links light up
6. CookieBanner component drafted; mounted in MainLayout
7. Grievance officer section appears in Privacy page footer + main footer
