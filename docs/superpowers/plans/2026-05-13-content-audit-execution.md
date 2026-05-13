# Content Audit Execution Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Execute the 5-step content-audit pipeline defined in `docs/superpowers/specs/2026-05-13-content-audit-design.md`, producing 4 planning artifacts (CONTENT-INVENTORY.md, STYLE-GUIDE.md, AUDIT-FINDINGS.md + per-category findings, REWRITE-BACKLOG.md) under `docs/content-audit/`.

**Architecture:** Documentation-driven workflow with orchestrated agent dispatch. Phase 1 parallelizes 7 `Explore` subagents (one per app) to inventory content. Phase 2 is manual synthesis from `.impeccable.md`. Phase 3 dispatches 4 lens subagents per category in parallel (4 × 9 = 36 runs, batched by category). Phase 4 is manual scoring + bundling. Every artifact commits independently. ~14 commits total.

**Tech Stack:** Markdown documentation, git, subagent dispatch via Agent tool. No code changes outside `docs/content-audit/` and `docs/superpowers/`.

---

## File Structure

```
docs/content-audit/
├── README.md                          # Index + how to read the audit
├── INVENTORY-SCHEMA.md                # Inventory row schema reference
├── CONTENT-INVENTORY.md               # Phase 1 deliverable
├── STYLE-GUIDE.md                     # Phase 2 deliverable
├── lens-briefs/                       # Reusable prompts for Phase 3 lens agents
│   ├── technical-writer.md
│   ├── legal.md
│   ├── business-analyst.md
│   └── brand-voice.md
├── AUDIT-FINDINGS.md                  # Phase 3 rolled-up index
├── findings/                          # Phase 3 per-category detail
│   ├── legal.md
│   ├── marketing.md
│   ├── auth-onboarding.md
│   ├── core-ux.md
│   ├── errors-empty.md
│   ├── transactional.md
│   ├── microcopy.md
│   ├── help.md
│   └── seo-meta.md
└── REWRITE-BACKLOG.md                 # Phase 4 deliverable
```

**File responsibilities:**

- `README.md` — entry point; explains the four artifacts and how to consume the backlog
- `INVENTORY-SCHEMA.md` — schema reference for inventory rows; cited from `CONTENT-INVENTORY.md`
- `CONTENT-INVENTORY.md` — every user-facing string catalogued, organized by category then app
- `STYLE-GUIDE.md` — voice principles, persona tone matrix, vocabulary list, microcopy formulas, legal tone, formatting conventions
- `lens-briefs/*.md` — reusable system prompts for each lens; same brief reused across all 9 categories
- `AUDIT-FINDINGS.md` — index: one row per finding with `finding_id, surface_id, lens, severity, link_to_category_file`
- `findings/*.md` — full prose for each finding (issue, evidence excerpt, recommendation, depends_on)
- `REWRITE-BACKLOG.md` — 10 CW-NN phase bundles with scoring, dependencies, launch-gate flags

---

## Task 0: Scaffold docs/content-audit/ directory

**Files:**
- Create: `docs/content-audit/README.md` (stub)
- Create: `docs/content-audit/findings/.gitkeep`
- Create: `docs/content-audit/lens-briefs/.gitkeep`

- [ ] **Step 1: Create directory tree**

Run:
```bash
mkdir -p docs/content-audit/findings docs/content-audit/lens-briefs
```

- [ ] **Step 2: Write stub README.md**

Create `docs/content-audit/README.md`:

```markdown
# Home Chef Content Audit

Comprehensive audit of all user-facing static content across the 7 Home Chef apps and the Go API.

Design spec: [../superpowers/specs/2026-05-13-content-audit-design.md](../superpowers/specs/2026-05-13-content-audit-design.md)
Execution plan: [../superpowers/plans/2026-05-13-content-audit-execution.md](../superpowers/plans/2026-05-13-content-audit-execution.md)

## Artifacts

| File | Status | Purpose |
|---|---|---|
| `INVENTORY-SCHEMA.md` | TODO | Schema reference for inventory rows |
| `CONTENT-INVENTORY.md` | TODO | Every user-facing string, by category × app |
| `STYLE-GUIDE.md` | TODO | Voice, vocabulary, microcopy formulas |
| `lens-briefs/` | TODO | Reusable prompts for audit lenses |
| `AUDIT-FINDINGS.md` | TODO | Rolled-up index of all findings |
| `findings/*.md` | TODO | Per-category finding detail |
| `REWRITE-BACKLOG.md` | TODO | 10-phase prioritized execution sequence |

## How to read this audit

1. Skim `STYLE-GUIDE.md` for the voice the audit measures against
2. Read `REWRITE-BACKLOG.md` for the sequenced execution phases (CW-01..CW-10)
3. Drill into specific findings via `AUDIT-FINDINGS.md` → `findings/<category>.md`
4. Use `CONTENT-INVENTORY.md` to confirm coverage of a specific surface
```

- [ ] **Step 3: Verify structure**

Run: `find docs/content-audit -type f -o -type d | sort`

Expected output (8 lines):
```
docs/content-audit
docs/content-audit/README.md
docs/content-audit/findings
docs/content-audit/findings/.gitkeep
docs/content-audit/lens-briefs
docs/content-audit/lens-briefs/.gitkeep
```

- [ ] **Step 4: Commit**

```bash
git add docs/content-audit/
git commit -m "docs(content-audit): scaffold directory and stub README"
```

---

## Task 1: Define inventory schema

**Files:**
- Create: `docs/content-audit/INVENTORY-SCHEMA.md`

- [ ] **Step 1: Write the schema document**

Create `docs/content-audit/INVENTORY-SCHEMA.md`:

```markdown
# Inventory Row Schema

Every row in `CONTENT-INVENTORY.md` follows this shape.

## Columns

| Column | Type | Description | Example |
|---|---|---|---|
| `surface_id` | string | Stable ID across phases. Format: `<app>-<category>-<slug>` | `web-marketing-landing-hero` |
| `app` | enum | One of: `web`, `vendor-portal`, `delivery-portal`, `admin-portal`, `mobile-customer`, `mobile-vendor`, `mobile-delivery`, `api` | `web` |
| `category` | enum | One of the 9 buckets (see below) | `marketing` |
| `route_or_file` | string | Route path or source-file path (with line range when relevant) | `apps/web/src/app/routes/HomePage.tsx:42-78` |
| `audience` | enum | Primary persona seeing this surface | `customer` |
| `word_count` | int | Approx word count of visible copy | 47 |
| `current_text_excerpt` | string | First 200 chars of visible text (for grep/identification) | `"Discover home chefs near you..."` |
| `shared_component_origin` | string \| null | Upstream source if this string comes from a shared package | `@tesserix/web Button` or `null` |
| `last_edited` | ISO date \| null | Git blame date of last change to this string | `2026-04-22` |
| `notes` | string | Free-text: special handling, known issues, related surfaces | `"Hero CTA also embedded in mobile-customer onboarding"` |

## Categories (9 buckets)

| Category | Scope |
|---|---|
| `legal` | T&C, Privacy, Refund/Cancellation, Cookie, Acceptable Use, Chef Agreement, Driver Agreement, Allergen/Food-safety disclosures |
| `marketing` | Landing/home, About, How-it-works, For-Chefs, For-Drivers, Pricing, FAQ |
| `auth-onboarding` | Sign up, Login, MFA, Password reset, OAuth consent, Chef onboarding wizard, Driver verification, Email-OTP |
| `core-ux` | In-app verbs and labels: browse, search, cart, checkout, order tracking, menu builder, order management, delivery navigation |
| `errors-empty` | Validation errors, system errors (4xx/5xx), empty states, offline states, loading states |
| `transactional` | Backend-generated: email templates, push notifications, SMS, in-app toasts/banners |
| `microcopy` | Tooltips, helper text, placeholders, success confirmations, button labels, modal subtitles |
| `help` | FAQ articles, contact, support flows, in-app hints, "learn more" links |
| `seo-meta` | `<title>`, `<meta description>`, OG tags, structured data, sitemap copy |

## Audience values

`customer`, `chef`, `driver`, `admin`, `multi` (if surface is shared across personas)

## Surface ID conventions

- Lowercase, kebab-case
- Prefix with app shortcode: `web-`, `vp-`, `dp-`, `ap-`, `mc-`, `mv-`, `md-`, `api-`
- Followed by category: `legal-`, `mkt-`, `auth-`, `ux-`, `err-`, `tx-`, `mc-`, `help-`, `seo-`
- Then a slug identifying the surface

Examples:
- `web-mkt-landing-hero`
- `vp-auth-chef-onboarding-step-3`
- `api-tx-order-confirmed-email`
- `mc-err-network-offline-banner`

## What is NOT in the inventory

- Server log strings
- Internal admin debug strings
- Code comments
- Alt-text on decorative icons (covered by prior a11y sweep)
- DevTools strings
- Test fixture strings
```

- [ ] **Step 2: Verify the file is well-formed**

Run: `wc -l docs/content-audit/INVENTORY-SCHEMA.md`
Expected: ~70-90 lines.

- [ ] **Step 3: Commit**

```bash
git add docs/content-audit/INVENTORY-SCHEMA.md
git commit -m "docs(content-audit): define inventory row schema"
```

---

## Task 2: Dispatch 7 parallel Explore agents to extract content per app

**Files:**
- Create: `docs/content-audit/CONTENT-INVENTORY.md` (after all 7 agents return)

- [ ] **Step 1: Dispatch all 7 Explore agents in a SINGLE message**

Use the Agent tool with `subagent_type: Explore`, dispatching 7 in parallel in one assistant turn. One per app. Use this prompt template, substituting `<APP_PATH>` and `<APP_SHORTCODE>`:

```
Inventory every user-facing static-content string in <APP_PATH>. Output as a markdown table conforming to the schema in docs/content-audit/INVENTORY-SCHEMA.md.

Categories to cover (rows go into whichever applies):
- legal, marketing, auth-onboarding, core-ux, errors-empty, transactional, microcopy, help, seo-meta

For each user-visible string longer than 2 words, produce one row with:
surface_id | app | category | route_or_file | audience | word_count | current_text_excerpt | shared_component_origin | last_edited | notes

Surface_id format: <APP_SHORTCODE>-<category-prefix>-<slug>
Category prefixes: legal-, mkt-, auth-, ux-, err-, tx-, mc-, help-, seo-

For shared_component_origin, look for usages of @tesserix/web, @tesserix/native, packages/ui, or local shared components — if the string lives in a shared component, record the package + component name; otherwise null.

For last_edited, use `git log -1 --format=%ad --date=short -- <file>` per file.

EXCLUDE:
- Strings under 3 words (buttons OK, but skip single-word labels like "OK", "Cancel" — those will be inventoried as part of the shared-component pass)
- Code comments, console logs, test fixtures
- Alt text on decorative icons (already covered by a11y sweep)
- Internal debug strings

Return the markdown table ready to paste into CONTENT-INVENTORY.md. Sort rows by category, then by route_or_file.

Report under 500 words on what you found, separately from the table.
```

The 7 dispatches:
1. `<APP_PATH>=apps/web/` `<APP_SHORTCODE>=web`
2. `<APP_PATH>=apps/vendor-portal/` `<APP_SHORTCODE>=vp`
3. `<APP_PATH>=apps/delivery-portal/` `<APP_SHORTCODE>=dp`
4. `<APP_PATH>=apps/admin-portal/` `<APP_SHORTCODE>=ap`
5. `<APP_PATH>=apps/mobile-customer/` `<APP_SHORTCODE>=mc`
6. `<APP_PATH>=apps/mobile-vendor/` `<APP_SHORTCODE>=mv`
7. `<APP_PATH>=apps/mobile-delivery/` `<APP_SHORTCODE>=md`

For the Go API (8th, separate because content type differs), run a second `Explore` dispatch with this adjusted prompt:

```
Inventory every user-facing static string emitted by the Go API in apps/api/:
- Email template subjects + bodies (look for email rendering / templates)
- Push notification payloads (title, body)
- SMS templates
- API error message strings returned to clients (gin.H{"error": "..."} bodies, especially user-facing ones)
- Validation messages
- Webhook event copy

Skip:
- Internal log strings (log.Printf, logrus calls without user-visible payload)
- DB error wrappers that get translated before client
- Code comments

Output schema same as above; app=api; surface_id prefix=api-; category prefix depends (tx- for emails/push/sms, err- for error strings).
```

- [ ] **Step 2: Wait for all 8 agents to return, then merge into CONTENT-INVENTORY.md**

Compose `docs/content-audit/CONTENT-INVENTORY.md` with this structure:

```markdown
# Content Inventory

Schema: see [INVENTORY-SCHEMA.md](./INVENTORY-SCHEMA.md)

Generated: <today's ISO date>
Total surfaces: <count>

## Coverage Summary

| App | Surfaces | Categories present |
|---|---|---|
| web | <n> | <list> |
| vendor-portal | <n> | <list> |
| ... | | |

## By Category

### Legal

<merged table rows from all 8 agents, filtered to category=legal, sorted by app>

### Marketing

<merged table rows for marketing>

[... repeat for each of 9 categories ...]
```

- [ ] **Step 3: Spot-check coverage**

Pick 5 random routes from `apps/web/src/app/routes/` (use `ls` to list, pick 5). For each route, visually verify that the visible strings appear as rows in `CONTENT-INVENTORY.md`. If any string is missing, re-dispatch the `web` Explore agent with a more specific scope.

Run for each picked route:
```bash
grep -c "<route-file-stem>" docs/content-audit/CONTENT-INVENTORY.md
```

Expected: >0 for each. If 0, that route was missed — add a follow-up step to rerun.

- [ ] **Step 4: Commit**

```bash
git add docs/content-audit/CONTENT-INVENTORY.md
git commit -m "docs(content-audit): inventory all user-facing strings across 7 apps + API"
```

---

## Task 3: Draft the Style Guide

**Files:**
- Create: `docs/content-audit/STYLE-GUIDE.md`

- [ ] **Step 1: Read .impeccable.md for canonical voice principles**

Run: `cat .impeccable.md` (or read it via the Read tool) — the entire file. The style guide must not contradict it.

- [ ] **Step 2: Write STYLE-GUIDE.md with all 7 sections**

Create `docs/content-audit/STYLE-GUIDE.md`:

```markdown
# Home Chef Style Guide

Voice and content conventions for every user-facing string. Aligned with `.impeccable.md` (visual design source of truth). The audit measures every surface against this guide.

## 1. Voice Principles

From `.impeccable.md`: **Confident · Appetizing · Quietly modern.**

### Five rules

**Rule 1 — Confident, not loud.** No urgency tricks. No exclamation marks except in genuine celebration (≤1 per page).

- ✅ "Order placed. Your chef is preparing it now."
- ❌ "🎉 ORDER PLACED! Get ready for AMAZING food! 🍽️"

**Rule 2 — Plain English over jargon.** Choose the shorter, more common word when it carries the same meaning.

- ✅ "We use this to send you order updates."
- ❌ "Pursuant to our communication policy, we utilize this data..."

**Rule 3 — Photo-forward, chrome-light.** UI chrome shrinks; food and faces carry the brand. Copy supports the photo, doesn't compete with it.

- ✅ Hero: "Tonight's dinner, from a kitchen near you." (over photo)
- ❌ Hero: "DISCOVER THE BEST HOME-COOKED MEALS IN YOUR CITY WITH OUR REVOLUTIONARY PLATFORM" (overrides photo)

**Rule 4 — One accent per surface.** One CTA in herb green. One emphasized word per headline.

- ✅ "Order from **home chefs**, not restaurants."
- ❌ "Order from **home chefs**, not **restaurants**, on the **best** platform."

**Rule 5 — Restraint over urgency.** No countdown timers, no "Only 2 left!", no FOMO patterns. Trust the user.

- ✅ "Available today" / "Available tomorrow"
- ❌ "🔥 Selling fast! Order now before it's gone!"

## 2. Persona Tone Matrix

Same voice principles, dialed per audience.

| Persona | Tone | Sentence length | Verb mood | Example |
|---|---|---|---|---|
| Customer | Warmer, sensory | Conversational (10-18 words) | Suggestive ("Discover", "Try") | "Discover chefs cooking in your neighborhood." |
| Chef / Vendor | Functional, time-aware | Crisp (5-12 words) | Operational ("Mark ready", "Pause menu") | "5 orders queued. Earliest pickup: 7:15 PM." |
| Driver | Glanceable, imperative | Telegraphic (≤4 words where possible) | Imperative ("Pick up", "Confirm") | "Pick up at 7 PM. 1.2 km." |
| Admin | Neutral operator | Precise (any length, no fluff) | Direct ("Approve", "Suspend", "Audit") | "Suspend vendor — requires reason code." |

## 3. Vocabulary List

Preferred terms ✅ / banned variants ❌. When a banned variant is in production code, file a finding.

### Identity & roles

- `Sign in` ✅ / `Log in` ❌ / `Login` ❌ (always two words, "sign")
- `Sign out` ✅ / `Log out` ❌
- `Sign up` ✅ (verb) / `Signup` ❌
- `Home chef` ✅ (customer-facing) / `Cook` ❌ / `Vendor` ❌ / `Seller` ❌
- `Chef` ✅ (chef-facing, in their portal) — they self-identify as chefs
- `Driver` ✅ (driver-facing) / `Delivery partner` (customer-facing only)
- `Customer` ✅ (internal/chef-facing) / `User` ❌ (too generic)

### Order verbs

- `Place an order` ✅ / `Purchase` ❌ / `Buy` ❌ (food, not commerce)
- `Order` (noun) ✅ / `Transaction` ❌ (customer side)
- `Pickup` (noun) ✅ / `Pick-up` ❌ / `Pick up` ❌ (when noun)
- `Pick up` (verb, two words) ✅
- `Delivery` ✅ / `Shipping` ❌

### Money & status

- `₹120` ✅ (no space, currency before amount) / `120 ₹` ❌ / `Rs. 120` ❌ / `INR 120` ❌
- `Total` ✅ / `Grand Total` ❌ (over-emphasized)
- `Subtotal` ✅ / `Sub-total` ❌
- `Tax` ✅ / `GST` ✅ when GST line specifically (legally required)
- `Free delivery` ✅ / `Delivery: Free` ❌
- `Paid` ✅ / `Payment successful` ✅ (UI verb forms — pick one per surface)

### Time & dates

- Today: `Today, 7:30 PM` ✅
- Tomorrow: `Tomorrow, 7:30 PM` ✅
- Future: `Fri, 14 May, 7:30 PM` ✅ (en-IN locale, 12-hour clock customer-facing)
- Past relative under 24h: `35 minutes ago` ✅
- Past >24h: `Yesterday` / `2 days ago` / absolute date thereafter

### Banned brand-drift terms

- `Artisanal` ❌
- `Handcrafted with love` ❌
- `Curated` ❌ (overused; use "selected" or just describe directly)
- `Foodie` ❌
- `Hurry!` / `Limited time!` / `Only X left!` ❌ (urgency tricks)
- `Click here` ❌ (use descriptive link text)
- `Learn more →` ✅ when paired with topic context, never standalone

## 4. Microcopy Patterns (formulas)

### Buttons

Format: **verb-first, ≤3 words, sentence case**.

| Action | ✅ | ❌ |
|---|---|---|
| Submit order | `Place order` | `PLACE ORDER`, `Click here to order`, `Submit` |
| Save draft | `Save draft` | `Save it for later`, `Save Draft` |
| Confirm pickup | `Confirm pickup` | `Confirm Pick-Up`, `I picked it up` |

### Errors

Format: **What happened → What to do.** Two sentences max. No blame. No "Oops!" "Uh oh!" exclamation patterns.

- ✅ "Card declined. Try a different payment method."
- ✅ "Network lost. Reconnect to keep tracking."
- ❌ "Oops! Something went wrong! 😢 Please try again later."

### Empty states

Format: **Why it's empty → One action.**

- ✅ "No orders yet. Browse chefs near you."
- ✅ "No menu items. Add your first dish."
- ❌ "Looks like nothing here!" / "It's lonely in here..."

### Success toasts

Format: **past tense, ≤6 words, period.**

- ✅ "Order placed."
- ✅ "Menu published."
- ❌ "Yay! Your order is on its way! 🎉"

### Form labels

Format: **noun, sentence case, no colons.** Helper text in muted tone, under the input.

- Label: `Delivery address`
- Helper: `We'll send your driver here.`
- ❌ `DELIVERY ADDRESS:` / `Address*` (required indicator is asterisk on field, label stays clean)

### Modal subtitles

Format: **explain consequence in one sentence.**

- ✅ "Cancelling this order refunds the customer immediately."
- ❌ "Are you sure? This action cannot be undone." (vague, doesn't explain WHAT can't be undone)

## 5. Legal-Page Tone

Plain language even in T&C. Plain language is a legal feature, not a tradeoff.

- **Short sentences** — average 15-20 words, max 25
- **Headings every ~200 words** — never let a wall of text exceed 200 words without a heading
- **"We" / "you"** — never "the Company", "the User", "the Service Provider"
- **Allergen / refund clauses get callout boxes** — not buried in paragraphs
- **Defined terms in bold first-use** — `**Order**`, `**Pickup Window**`, `**Refund Period**`
- **Active voice** — "We refund within 7 days" not "Refunds are processed within 7 days"
- **One idea per paragraph** — split walls of legal prose
- **Plain-language summary callout at top of every legal page** — "Here's what this page covers, in one paragraph"

Example transformation:

❌ Before:
> "The Company shall not be liable for any damages, losses, or expenses arising from the consumption of food items procured through the Service, including but not limited to allergic reactions, foodborne illness, or any other adverse health consequences, except to the extent such liability cannot be excluded under applicable law."

✅ After:
> "**Allergens.** Home chefs label allergens on every dish. If you have allergies, check the label before ordering and contact the chef with questions. We aren't responsible for allergic reactions unless we're legally required to be (e.g., we hid allergen info from you)."

## 6. Numerals & Formatting

- **Tabular figures everywhere money / IDs / ETAs appear** — `font-feature-settings: "tnum"`
- **Currency** — `₹120` (no space, symbol first) — en-IN locale primary
- **Time** — 24-hour internal-facing (admin, vendor portal scheduling), 12-hour customer-facing
- **Relative time threshold** — under 24h relative ("in 35 min", "2 hours ago"); 24h+ absolute ("Tomorrow, 7:30 PM", "Fri, 14 May")
- **Phone numbers** — `+91 98765 43210` (en-IN format)
- **Order IDs** — `#HC-2026-00001234` — prefix, year, zero-padded
- **Distance** — kilometres customer-facing (`1.2 km`); metres only when <1 km (`850 m`)
- **Plurals** — always: `1 order` / `0 orders` / `2 orders` (never "order(s)")
- **Percentages** — no space: `15%` (never `15 %`)

## 7. Internationalization Readiness

Every rule above survives translation IF:

- Button-length rules allow 30% slack (Hindi/Tamil run longer)
- Sentence templates don't depend on English word order (avoid concatenation like `"Order #" + id + " by " + chef`)
- Plurals use ICU MessageFormat-compatible patterns (or whatever i18n lib is chosen)
- Date formatting goes through `Intl.DateTimeFormat`, never hand-rolled
- Currency formatting goes through `Intl.NumberFormat({ style: 'currency', currency: 'INR' })`

Translation work is **out of scope for this audit**, but every string the audit flags for rewrite must be translation-ready when rewritten.
```

- [ ] **Step 3: Sanity-check against .impeccable.md**

Run:
```bash
grep -iE "(terracotta|cream|playfair|artisanal|bone)" docs/content-audit/STYLE-GUIDE.md
```

Expected: zero matches (or only matches inside "banned" lists). If the style guide accidentally introduces a contradiction with `.impeccable.md` (e.g., re-introduces the terracotta-era vocabulary), fix it.

- [ ] **Step 4: Commit**

```bash
git add docs/content-audit/STYLE-GUIDE.md
git commit -m "docs(content-audit): style guide — voice, persona tone, vocabulary, microcopy formulas"
```

---

## Task 4: Write the 4 lens briefs

**Files:**
- Create: `docs/content-audit/lens-briefs/technical-writer.md`
- Create: `docs/content-audit/lens-briefs/legal.md`
- Create: `docs/content-audit/lens-briefs/business-analyst.md`
- Create: `docs/content-audit/lens-briefs/brand-voice.md`

Each brief is the system prompt used by its lens agent for every category. Writing them once makes Phase 3 dispatch-friendly.

- [ ] **Step 1: Write technical-writer brief**

Create `docs/content-audit/lens-briefs/technical-writer.md`:

```markdown
# Technical Writer Lens Brief

You are auditing user-facing content for clarity and craft. You read every string against the style guide and flag deviations.

## Inputs you will receive
- A slice of `CONTENT-INVENTORY.md` (one category's rows)
- `STYLE-GUIDE.md` (the rules)

## What to flag

- **Voice drift** — strings that don't match the persona tone matrix (Sec 2 of style guide)
- **Banned vocabulary** — every match from the banned list (Sec 3)
- **Microcopy formula violations** — buttons >3 words, errors that don't follow "what happened → what to do", empty states that don't follow "why → one action", success toasts >6 words or non-past-tense (Sec 4)
- **Sentence length** — customer-facing >25 words; vendor-facing >20; driver-facing >12
- **Reading ease** — flag passages where Flesch reading ease drops below 60 (customer-facing) or 50 (vendor/admin)
- **Jargon** — uncommon words where a common synonym exists
- **Ambiguity** — strings that could be read two ways
- **Inconsistency** — same concept named differently across surfaces (e.g., "Cart" in one place, "Bag" in another)
- **Missing helper text** — form labels with no explanatory helper where the field is non-obvious

## Output format

For each finding, output:

```yaml
- finding_id: TW-<NNN>   # sequential per category
  surface_id: <from inventory>
  lens: technical-writer
  severity: P0|P1|P2|P3
  issue: "<one-line summary>"
  evidence_excerpt: "<the actual offending text, ≤200 chars>"
  recommendation: "<concrete rewrite, or a rule-based fix instruction>"
  depends_on: null
```

## Severity guide (TW lens)

- **P0** — Factually misleading copy (commitment the product can't keep, wrong refund window, wrong price format that breaks invoicing)
- **P1** — Conversion-critical voice drift (landing hero, checkout CTA, signup error)
- **P2** — Microcopy formula violation, vocabulary banned-list match, length-rule violation
- **P3** — Polish: tooltip improvements, helper-text additions, minor inconsistencies

## Out of scope for TW lens
- Legal sufficiency (Legal lens owns this)
- Conversion psychology (BA lens owns this)
- Brand-voice cross-app drift detection (Brand Voice lens owns this — TW flags *within-surface* voice drift)
- Visual design issues
```

- [ ] **Step 2: Write legal brief**

Create `docs/content-audit/lens-briefs/legal.md`:

```markdown
# Legal Lens Brief

You are auditing content for legal/regulatory exposure under India jurisdiction + generic best-practice. You are NOT a lawyer; every finding flags `depends_on: needs lawyer review` for human binding-text drafting.

## Inputs you will receive
- A slice of `CONTENT-INVENTORY.md`
- `STYLE-GUIDE.md` (so your recommendations match house tone)

## What to flag

### DPDP Act 2023 (India data privacy)
- Missing or weak consent notice on signup / first-data-collection moment
- Missing data fiduciary identification
- Missing notice of rights (access, correction, erasure, grievance)
- Missing grievance-officer contact
- Bundled consent (one checkbox for multiple unrelated purposes)
- Pre-checked consent boxes
- Missing children's data handling (under 18)
- Missing data retention statement
- Cookie/tracking disclosure missing or buried

### FSSAI (Food Safety and Standards Authority of India)
- Missing FSSAI license number on chef profile / order receipt
- Missing allergen disclosure on product listings
- Missing "this dish contains" disclaimers
- Missing food-handling responsibility split (chef vs platform)
- Missing food complaint pathway

### RBI Payment Aggregator rules
- Missing T&C clarity on payment flow (who is the merchant of record)
- Refund timeline not explicit (≤7 days for digital, must be stated)
- Missing escrow/settlement disclosure if applicable
- KYC requirements for chefs (financial onboarding)

### GST (India tax)
- Invoice missing GST number, HSN/SAC code, GST breakup line
- "Tax inclusive" vs "Tax extra" not clearly stated on price display
- B2B invoice flow not differentiated from B2C

### Gig-worker terms (drivers)
- Driver agreement classification (independent contractor vs employee)
- Insurance disclosure (who pays, what's covered)
- Earnings transparency (rate calculation, deductions)
- Termination clause clarity

### Generic best-practice (jurisdiction-agnostic)
- T&C: governing law clause missing/wrong
- T&C: dispute resolution / arbitration clause unclear
- T&C: limitation of liability — overbroad waivers (likely unenforceable)
- Privacy: third-party data sharing not disclosed (Razorpay, GCS, etc.)
- Cookie: no consent banner / no granular controls
- Accessibility: legal-page content not in accessible format (covered by a11y sweep, but flag if missing alt-text on legal page diagrams)
- Plain-language test — if you can't summarize a clause in one sentence, it fails

## Output format

```yaml
- finding_id: LEG-<NNN>
  surface_id: <from inventory>
  lens: legal
  severity: P0|P1|P2|P3
  issue: "<one-line summary>"
  evidence_excerpt: "<actual text or 'missing entirely'>"
  recommendation: "<what to add/change, plus citation>"
  citation: "DPDP Act §X" | "FSSAI §Y" | "best-practice" | etc.
  depends_on: "needs lawyer review"   # ALWAYS for legal findings
```

## Severity guide (Legal lens)

- **P0** — Regulatory exposure: missing required disclosure (DPDP consent, FSSAI license, GST line), unenforceable clause, jurisdictional ambiguity that breaks the contract
- **P1** — Best-practice gap: privacy policy missing third-party disclosure, cookie banner missing granular controls, refund timeline implicit not explicit
- **P2** — Plain-language failure on legal page (audit-style readability issue with legal-stakes content)
- **P3** — Cosmetic legal-page issues (heading structure, glossary missing)

## Out of scope for Legal lens
- Tax math / pricing logic (engineering concern)
- Drafting binding text (lawyer's job — your output is "what's missing", not "here's the new clause")
- Non-India jurisdictions (US/EU/UK explicitly out of scope for this audit)
```

- [ ] **Step 3: Write business-analyst brief**

Create `docs/content-audit/lens-briefs/business-analyst.md`:

```markdown
# Business Analyst Lens Brief

You are auditing content for conversion, activation, and trust impact. Your lens is: would this copy move the business metric in the right direction?

## Inputs you will receive
- A slice of `CONTENT-INVENTORY.md`
- `STYLE-GUIDE.md`

## What to flag

### Conversion-critical CTAs
- Landing hero CTA: verb clarity, value-prop alignment
- Signup CTA: friction language, social-proof presence
- Checkout CTA: trust signals, payment-safety reassurance
- Chef-apply CTA: clear next step, expectation-setting
- Empty-state CTAs: do they suggest the obvious next action?

### Value-proposition clarity
- Landing/home: who is this for, what's the offer, why now
- For-Chefs / For-Drivers: differentiation from gig competitors (Swiggy/Zomato/Dunzo)
- Pricing: transparent, no hidden fees language

### Friction language
- Mandatory fields not flagged as such
- "Required" / "Optional" inconsistency
- Multi-step forms missing progress indication copy
- Re-entry friction (forms that don't explain why a field is needed)

### Drop-off zones
- Onboarding wizard step copy — does each step have a clear "why am I here, what next"
- Verification steps (email OTP, phone OTP, ID upload) — friction language
- Payment-method selection — trust-eroding copy

### Trust signals
- Reviews count visibility
- Food-safety / FSSAI badge prominence
- Chef profile completeness signals
- Order-tracking transparency

### Pricing transparency
- All-in pricing vs hidden delivery fees
- "Starting at" patterns without ceiling
- Tax-inclusive vs tax-extra clarity (also a Legal flag, but BA cares about trust impact)

### Empty-state opportunity loss
- Empty cart: "browse chefs near you" vs blank
- No favorites yet: prompt to favorite
- New chef no menu: walkthrough to add first dish

### Engagement / retention copy
- Welcome flows: first 5 customer / first 5 chef / first 5 driver
- Re-engagement push notification copy

## Output format

```yaml
- finding_id: BA-<NNN>
  surface_id: <from inventory>
  lens: business-analyst
  severity: P0|P1|P2|P3
  issue: "<one-line summary>"
  evidence_excerpt: "<actual text>"
  recommendation: "<concrete copy change + expected metric impact>"
  metric_hypothesis: "<which metric this affects, e.g., 'signup completion', 'cart→order conversion', 'chef D7 retention'>"
  depends_on: null
```

## Severity guide (BA lens)

- **P0** — Demonstrably broken conversion path (e.g., checkout CTA says "Submit" instead of "Pay ₹X")
- **P1** — Conversion-critical voice/clarity issue on a high-traffic surface (landing, signup, checkout, chef-apply)
- **P2** — Missed opportunity on medium-traffic surface (empty states, re-engagement push)
- **P3** — Minor friction on low-traffic surface (admin internal copy, deep settings)

## Out of scope for BA lens
- Legal sufficiency (Legal lens)
- Voice / vocabulary detail (TW lens)
- Visual design / layout
- Pricing strategy itself (you audit the COPY around prices, not the prices)
```

- [ ] **Step 4: Write brand-voice brief**

Create `docs/content-audit/lens-briefs/brand-voice.md`:

```markdown
# Brand Voice Lens Brief

You are auditing for **cross-surface and cross-app voice consistency** against `.impeccable.md` + `STYLE-GUIDE.md`. Your lens is: does this whole product sound like one brand?

## Inputs you will receive
- A slice of `CONTENT-INVENTORY.md` (the category's rows ACROSS all 7 apps + API)
- `STYLE-GUIDE.md`
- `.impeccable.md` (source of truth for voice principles)

## What to flag

### Cross-app drift
- Same concept named differently across apps (e.g., "Cart" in web, "Bag" in mobile)
- Same action labeled differently (e.g., "Place order" vs "Confirm order" vs "Submit")
- Different greeting tones in welcome emails per persona (when they should share voice DNA)

### Persona-tone violations
- Customer-facing copy that's too operational (sounds like vendor portal)
- Vendor-facing copy that's marketing-y (sounds like customer landing)
- Driver-facing copy that's verbose (>4 words where ≤4 would do)

### Brand-personality drift
- "Confident · Appetizing · Quietly modern" — flag copy that's the opposite:
  - **Loud** ("HURRY!", "AMAZING DEAL", "🎉🎉🎉")
  - **Bland** ("Welcome to our platform", "Your order has been received")
  - **Trend-chasing** ("vibes", "no cap", emoji-as-bullet patterns from 2022 DTC)
  - **Faux-artisanal** ("handcrafted with love", "lovingly prepared", "homestyle goodness")

### Anti-references (from `.impeccable.md`)
- Terracotta-era legacy copy that survived the migration ("artisanal", "cream-and-terra", "warm-cozy", "Playfair Display"-era headlines)
- Swiggy/Zomato-style red-saturated urgency ("Order now!", "Last few left!")
- AI-slop maximalism (cyan-on-dark dark-mode headlines, gradient promises)
- Generic SaaS dashboard ("Hero metrics", "Onboarding journey", "Stakeholder")

### Voice consistency checks
- Pronouns: "we"/"you" enforced everywhere (not "the Company"/"the User"/"the Driver Partner")
- Punctuation: exclamation-mark budget (1 per page customer-facing, 0 vendor/driver)
- Emoji: customer-facing OK in occasional moments; vendor/driver/admin = 0 emoji
- Capitalization: sentence case for buttons and labels; Title Case banned outside of proper nouns

## Output format

```yaml
- finding_id: BV-<NNN>
  surface_id: <from inventory>
  lens: brand-voice
  severity: P0|P1|P2|P3
  issue: "<one-line summary>"
  evidence_excerpt: "<actual text>"
  related_surfaces: ["<other surface_ids showing the same drift>"]
  recommendation: "<rewrite or rule>"
  depends_on: null
```

## Severity guide (Brand Voice lens)

- **P0** — Cross-surface contradiction that breaks brand trust (e.g., "Cart" on web, "Bag" on mobile — customer thinks it's a different product)
- **P1** — Personality drift on entry surfaces (landing, signup, welcome email, push notifications)
- **P2** — Inconsistent persona tone on secondary surfaces
- **P3** — Punctuation / capitalization drift in deep surfaces

## Out of scope for Brand Voice lens
- Within-surface microcopy formulas (TW lens — you focus on cross-surface)
- Legal compliance (Legal lens)
- Conversion psychology (BA lens)
- Visual brand identity (`.impeccable.md` and design system handle this)
```

- [ ] **Step 5: Commit**

```bash
git add docs/content-audit/lens-briefs/
git commit -m "docs(content-audit): lens briefs for technical-writer, legal, BA, brand-voice"
```

---

## Tasks 5–13: Per-category audit (9 categories × 4 lenses each)

These 9 tasks share the same structure. Each task audits one category by dispatching 4 lens agents in parallel against the inventory slice for that category.

**Important reuse:** the lens briefs in `docs/content-audit/lens-briefs/*.md` are loaded into each agent's system prompt; the per-category prompt only varies the inventory slice and any category-specific notes.

### Task 5: Audit `legal` category

**Files:**
- Create: `docs/content-audit/findings/legal.md`

- [ ] **Step 1: Extract the `legal` slice from CONTENT-INVENTORY.md**

Save the inventory rows where `category == legal` into a working clipboard or temp string. You'll paste this into each lens agent's prompt.

Run (sanity check):
```bash
grep -E "^\| .*\| legal \|" docs/content-audit/CONTENT-INVENTORY.md | wc -l
```

Expected: >0 rows.

- [ ] **Step 2: Dispatch 4 lens agents in PARALLEL (one message, four Agent calls)**

Each agent invocation uses subagent_type `general-purpose` (claude) except BA which uses `business-analyst`. Each gets:
1. Its lens brief (paste full contents of the corresponding `docs/content-audit/lens-briefs/*.md`)
2. The style guide (paste full contents of `docs/content-audit/STYLE-GUIDE.md`)
3. The inventory slice for `legal`
4. Instruction: "Output findings in the YAML format defined in your brief. No prose outside the YAML."

Template (substitute `<LENS>` and `<LENS_BRIEF_CONTENT>` for each of the four):

```
You are the <LENS> lens for the Home Chef content audit.

# Your brief
<paste docs/content-audit/lens-briefs/<lens>.md verbatim>

# Style guide
<paste docs/content-audit/STYLE-GUIDE.md verbatim>

# Inventory slice — category: legal
<paste the legal rows from CONTENT-INVENTORY.md>

# Task
Audit every row above. Output one YAML block per finding, in the format defined in your brief. No prose outside the YAML.
```

- [ ] **Step 3: Merge 4 agent outputs into findings/legal.md**

Create `docs/content-audit/findings/legal.md`:

```markdown
# Findings — Legal

Schema: see `../lens-briefs/<lens>.md` for severity definitions per lens.

Category: legal
Lenses applied: technical-writer, legal, business-analyst, brand-voice
Inventory slice size: <N> surfaces

## Technical Writer findings

<paste TW agent's YAML output>

## Legal findings

<paste Legal agent's YAML output>

## Business Analyst findings

<paste BA agent's YAML output>

## Brand Voice findings

<paste BV agent's YAML output>
```

- [ ] **Step 4: Sanity-check**

Run:
```bash
grep -cE "^- finding_id:" docs/content-audit/findings/legal.md
```

Expected: ≥1 per lens × number of inventory rows audited (varies). For `legal`, likely 30–80 total findings.

For Legal lens specifically, run:
```bash
grep -A1 "lens: legal" docs/content-audit/findings/legal.md | grep -c "needs lawyer review"
```

Expected: equals the number of `lens: legal` findings (every Legal finding must flag `depends_on: needs lawyer review`).

- [ ] **Step 5: Commit**

```bash
git add docs/content-audit/findings/legal.md
git commit -m "docs(content-audit): findings — legal (4 lenses)"
```

---

### Task 6: Audit `marketing` category

Identical structure to Task 5, but for the `marketing` category.

**Files:** Create `docs/content-audit/findings/marketing.md`

- [ ] **Step 1: Extract `marketing` inventory slice**

Run:
```bash
grep -E "^\| .*\| marketing \|" docs/content-audit/CONTENT-INVENTORY.md | wc -l
```

- [ ] **Step 2: Dispatch 4 lens agents in parallel for `marketing`**

Use the same prompt template from Task 5 Step 2, substituting category name to `marketing` and pasting the marketing inventory slice.

Category-specific note to add to all 4 prompts:
> Marketing surfaces span the customer landing page, "How it works", "For Chefs", "For Drivers", "About", "Pricing", and the public FAQ. Audit cross-app consistency carefully — `apps/web/` and `apps/mobile-customer/` likely have parallel hero copy that should match.

- [ ] **Step 3: Merge into findings/marketing.md** (same structure as Task 5)

- [ ] **Step 4: Sanity-check**

```bash
grep -cE "^- finding_id:" docs/content-audit/findings/marketing.md
```

Expected: 40–100 findings.

- [ ] **Step 5: Commit**

```bash
git add docs/content-audit/findings/marketing.md
git commit -m "docs(content-audit): findings — marketing (4 lenses)"
```

---

### Task 7: Audit `auth-onboarding` category

**Files:** Create `docs/content-audit/findings/auth-onboarding.md`

- [ ] **Step 1: Extract `auth-onboarding` inventory slice**

```bash
grep -E "^\| .*\| auth-onboarding \|" docs/content-audit/CONTENT-INVENTORY.md | wc -l
```

- [ ] **Step 2: Dispatch 4 lens agents in parallel**

Category-specific note:
> Auth-onboarding covers Sign up, Login, MFA, Password reset, OAuth consent screens, Chef onboarding wizard, Driver verification, Email-OTP. This is the highest-conversion-leverage category — BA lens should be most aggressive here. Legal lens should especially scrutinize DPDP consent moments at signup and the Driver agreement onboarding step.

- [ ] **Step 3: Merge into findings/auth-onboarding.md**

- [ ] **Step 4: Sanity-check**
```bash
grep -cE "^- finding_id:" docs/content-audit/findings/auth-onboarding.md
```
Expected: 40–80 findings.

- [ ] **Step 5: Commit**
```bash
git add docs/content-audit/findings/auth-onboarding.md
git commit -m "docs(content-audit): findings — auth-onboarding (4 lenses)"
```

---

### Task 8: Audit `core-ux` category

**Files:** Create `docs/content-audit/findings/core-ux.md`

- [ ] **Step 1: Extract slice**
```bash
grep -E "^\| .*\| core-ux \|" docs/content-audit/CONTENT-INVENTORY.md | wc -l
```

- [ ] **Step 2: Dispatch 4 lens agents in parallel**

Category-specific note:
> Core-UX is the largest category — verbs and labels across browse, search, cart, checkout, order tracking (customer), menu builder, order management (vendor), and delivery navigation (driver). Brand-voice lens should focus here on cross-app verb consistency (e.g., "Pickup" capitalization, "Order placed" vs "Order confirmed" semantics across personas).

- [ ] **Step 3: Merge into findings/core-ux.md**

- [ ] **Step 4: Sanity-check**
```bash
grep -cE "^- finding_id:" docs/content-audit/findings/core-ux.md
```
Expected: 80–180 findings (largest category).

- [ ] **Step 5: Commit**
```bash
git add docs/content-audit/findings/core-ux.md
git commit -m "docs(content-audit): findings — core-ux (4 lenses)"
```

---

### Task 9: Audit `errors-empty` category

**Files:** Create `docs/content-audit/findings/errors-empty.md`

- [ ] **Step 1: Extract slice**
```bash
grep -E "^\| .*\| errors-empty \|" docs/content-audit/CONTENT-INVENTORY.md | wc -l
```

- [ ] **Step 2: Dispatch 4 lens agents in parallel**

Category-specific note:
> Errors-empty is highest-value for consistency dividend. TW lens should aggressively flag every error message that doesn't match the "what happened → what to do" formula. BA lens should flag empty states that don't include a recovery CTA.

- [ ] **Step 3: Merge into findings/errors-empty.md**

- [ ] **Step 4: Sanity-check**
```bash
grep -cE "^- finding_id:" docs/content-audit/findings/errors-empty.md
```
Expected: 50–120 findings.

- [ ] **Step 5: Commit**
```bash
git add docs/content-audit/findings/errors-empty.md
git commit -m "docs(content-audit): findings — errors-empty (4 lenses)"
```

---

### Task 10: Audit `transactional` category

**Files:** Create `docs/content-audit/findings/transactional.md`

- [ ] **Step 1: Extract slice**
```bash
grep -E "^\| .*\| transactional \|" docs/content-audit/CONTENT-INVENTORY.md | wc -l
```

- [ ] **Step 2: Dispatch 4 lens agents in parallel**

Category-specific note:
> Transactional is backend-generated content — emails, push, SMS, in-app toasts/banners. These often skip design review; voice drift is high. Legal lens: flag any transactional message that should include FSSAI disclosure (order confirmation emails commonly miss this). Brand Voice lens: this is the most cross-persona category (same notification system serves customer, chef, driver).

- [ ] **Step 3: Merge into findings/transactional.md**

- [ ] **Step 4: Sanity-check**
```bash
grep -cE "^- finding_id:" docs/content-audit/findings/transactional.md
```
Expected: 40–100 findings.

- [ ] **Step 5: Commit**
```bash
git add docs/content-audit/findings/transactional.md
git commit -m "docs(content-audit): findings — transactional (4 lenses)"
```

---

### Task 11: Audit `microcopy` category

**Files:** Create `docs/content-audit/findings/microcopy.md`

- [ ] **Step 1: Extract slice**
```bash
grep -E "^\| .*\| microcopy \|" docs/content-audit/CONTENT-INVENTORY.md | wc -l
```

- [ ] **Step 2: Dispatch 4 lens agents in parallel**

Category-specific note:
> Microcopy is tooltips, helper text, placeholders, success confirmations, button labels (those not already captured in auth/checkout), modal subtitles. TW lens dominates here; Legal lens almost no findings expected; BA lens focused on form-friction microcopy; Brand-Voice on cross-app consistency.

- [ ] **Step 3: Merge into findings/microcopy.md**

- [ ] **Step 4: Sanity-check**
```bash
grep -cE "^- finding_id:" docs/content-audit/findings/microcopy.md
```
Expected: 30–100 findings (mostly P2/P3).

- [ ] **Step 5: Commit**
```bash
git add docs/content-audit/findings/microcopy.md
git commit -m "docs(content-audit): findings — microcopy (4 lenses)"
```

---

### Task 12: Audit `help` category

**Files:** Create `docs/content-audit/findings/help.md`

- [ ] **Step 1: Extract slice**
```bash
grep -E "^\| .*\| help \|" docs/content-audit/CONTENT-INVENTORY.md | wc -l
```

- [ ] **Step 2: Dispatch 4 lens agents in parallel**

Category-specific note:
> Help covers FAQ articles, contact, support flows, in-app hints, "learn more" links. Legal lens: any FAQ entry covering refunds/cancellation/allergens overlaps with legal exposure — flag inconsistencies between FAQ answers and the binding T&C / Refund policy.

- [ ] **Step 3: Merge into findings/help.md**

- [ ] **Step 4: Sanity-check**
```bash
grep -cE "^- finding_id:" docs/content-audit/findings/help.md
```
Expected: 20–60 findings.

- [ ] **Step 5: Commit**
```bash
git add docs/content-audit/findings/help.md
git commit -m "docs(content-audit): findings — help (4 lenses)"
```

---

### Task 13: Audit `seo-meta` category

**Files:** Create `docs/content-audit/findings/seo-meta.md`

- [ ] **Step 1: Extract slice**
```bash
grep -E "^\| .*\| seo-meta \|" docs/content-audit/CONTENT-INVENTORY.md | wc -l
```

- [ ] **Step 2: Dispatch 4 lens agents in parallel**

Category-specific note:
> SEO-meta is `<title>`, `<meta description>`, OG tags, structured data, sitemap. BA lens dominates (these are top-of-funnel). TW lens: 50-60 char title limit, 150-160 char meta description limit. Legal lens almost no findings expected.

- [ ] **Step 3: Merge into findings/seo-meta.md**

- [ ] **Step 4: Sanity-check**
```bash
grep -cE "^- finding_id:" docs/content-audit/findings/seo-meta.md
```
Expected: 15–50 findings.

- [ ] **Step 5: Commit**
```bash
git add docs/content-audit/findings/seo-meta.md
git commit -m "docs(content-audit): findings — seo-meta (4 lenses)"
```

---

## Task 14: Roll up AUDIT-FINDINGS.md index

**Files:**
- Create: `docs/content-audit/AUDIT-FINDINGS.md`

- [ ] **Step 1: Extract finding metadata from every findings/*.md**

Run:
```bash
for f in docs/content-audit/findings/*.md; do
  grep -E "^- finding_id:|^  surface_id:|^  lens:|^  severity:|^  issue:" "$f"
done
```

This gives you the raw rows. Reshape into a table.

- [ ] **Step 2: Write AUDIT-FINDINGS.md as a sortable index**

Create `docs/content-audit/AUDIT-FINDINGS.md`:

```markdown
# Audit Findings — Rolled-Up Index

Per-finding detail lives in `findings/<category>.md`. This index is for sorting, filtering, and dashboard views.

Generated: <ISO date>
Total findings: <N>

## Summary by severity

| Severity | Count |
|---|---|
| P0 | <n> |
| P1 | <n> |
| P2 | <n> |
| P3 | <n> |

## Summary by lens

| Lens | Count |
|---|---|
| technical-writer | <n> |
| legal | <n> |
| business-analyst | <n> |
| brand-voice | <n> |

## Summary by category

| Category | TW | Legal | BA | BV | Total |
|---|---|---|---|---|---|
| legal | <n> | <n> | <n> | <n> | <n> |
| marketing | <n> | <n> | <n> | <n> | <n> |
| ... | | | | | |

## Index

| finding_id | surface_id | category | lens | severity | issue | detail |
|---|---|---|---|---|---|---|
| TW-001 | web-mkt-landing-hero | marketing | technical-writer | P1 | <issue> | [findings/marketing.md#tw-001](findings/marketing.md#tw-001) |
| LEG-001 | web-legal-tnc | legal | legal | P0 | <issue> | [findings/legal.md#leg-001](findings/legal.md#leg-001) |
| ... | | | | | | |
```

- [ ] **Step 3: Verify the index is complete**

Run:
```bash
ROLLUP_COUNT=$(grep -cE "^\| (TW|LEG|BA|BV)-" docs/content-audit/AUDIT-FINDINGS.md)
DETAIL_COUNT=$(grep -hE "^- finding_id:" docs/content-audit/findings/*.md | wc -l)
echo "rollup=$ROLLUP_COUNT  detail=$DETAIL_COUNT"
```

Expected: rollup == detail. If mismatched, an audit finding is missing from the index — find and add it.

- [ ] **Step 4: Commit**

```bash
git add docs/content-audit/AUDIT-FINDINGS.md
git commit -m "docs(content-audit): rolled-up findings index"
```

---

## Task 15: Score findings + bundle into 10 CW-NN phases

**Files:**
- Create: `docs/content-audit/REWRITE-BACKLOG.md`

- [ ] **Step 1: Compute score per finding**

Apply the scoring formula from the design spec:

```
score = severity_weight × impact_weight ÷ effort_weight

severity:  P0=8  P1=4  P2=2  P3=1
impact:    customer-facing=3  chef/driver-facing=2  admin/internal=1
effort:    S(<1h)=1  M(1–4h)=2  L(>4h)=4
```

For each finding in `AUDIT-FINDINGS.md`, estimate effort (S/M/L) from the recommendation, determine impact from `audience` in the inventory, and compute score. Add a `score` and `effort` column to a working spreadsheet/table — does not need to be checked in, just used to bundle.

- [ ] **Step 2: Bundle findings into 10 CW-NN phases**

Bundles (from the design spec's suggested sequence). Rules are applied **in order**; each finding goes to the **first** rule it matches. The final catch-all rules guarantee every finding lands in some bundle.

| Phase | Bundle rule |
|---|---|
| CW-01 | `category == legal` OR `lens == legal` — at any severity. Legal findings always go here regardless of persona. |
| CW-02 | (`audience == customer` OR `audience == multi`) AND `severity` in {P0, P1} AND `category` in {`auth-onboarding`, `core-ux`} |
| CW-03 | `audience == chef` AND `severity` in {P0, P1} AND `category` in {`auth-onboarding`, `core-ux`} |
| CW-04 | `audience == driver` AND `severity` in {P0, P1} AND `category` in {`auth-onboarding`, `core-ux`} |
| CW-05 | `category == marketing` |
| CW-06 | `category == transactional` |
| CW-07 | `category == errors-empty` |
| CW-08 | `category == microcopy` |
| CW-09 | `category` in {`help`, `seo-meta`} |
| CW-10 | `audience == admin` OR `app == admin-portal` (catches anything not yet matched, scoped to admin) |
| CW-11 (overflow) | Anything still unmatched lands here for manual triage before backlog publication |

The CW-11 bundle should ideally be empty after rule application. If non-empty, manually reassign each finding to the most appropriate CW-NN before publishing.

Each finding belongs to exactly one bundle (use the FIRST matching rule).

- [ ] **Step 3: Write REWRITE-BACKLOG.md**

Create `docs/content-audit/REWRITE-BACKLOG.md`:

```markdown
# Rewrite Backlog — 10 Phases

Sequenced execution backlog. Each phase is a separate `/gsd-plan-phase` invocation later.

Generated: <ISO date>
Source: `AUDIT-FINDINGS.md` + `findings/*.md`

## Launch Gate

- **Must ship pre-launch:** CW-01, CW-02, CW-03, CW-04
- **Strongly recommended pre-launch:** CW-05, CW-06
- **Iterative post-launch OK:** CW-07, CW-08, CW-09, CW-10

## Phase Index

| phase_id | title | finding_count | personas | apps | est_LOE | launch_blocker | review_required | suggested_agent |
|---|---|---|---|---|---|---|---|---|
| CW-01 | Legal launch-blockers | <n> | all | all | L | Y | lawyer | claude + legal review checkpoint |
| CW-02 | Customer signup + checkout | <n> | customer | web, mobile-customer | M | Y | PM | claude |
| CW-03 | Chef onboarding + vendor verbs | <n> | chef | vendor-portal, mobile-vendor | M | Y | PM | claude |
| CW-04 | Driver onboarding + delivery glanceable | <n> | driver | delivery-portal, mobile-delivery | M | Y | PM + design | claude |
| CW-05 | Marketing surfaces voice unification | <n> | customer + chef + driver | web, mobile-customer | L | N | PM | claude + brand-voice review |
| CW-06 | Transactional content sweep | <n> | all | api | M | N | PM | claude |
| CW-07 | Errors + empty states unification | <n> | all | all | L | N | none | claude (formula-driven) |
| CW-08 | Microcopy polish | <n> | all | all | L | N | none | claude (LLM-assisted with style guide) |
| CW-09 | Help + SEO/meta | <n> | customer (mostly) | web | M | N | none | claude |
| CW-10 | Admin-portal copy | <n> | admin | admin-portal | S-M | N | none | claude |

## Phase Detail

### CW-01 — Legal launch-blockers

**Findings included:** <comma-separated finding_ids>
**Apps:** web, vendor-portal, delivery-portal, mobile-customer, mobile-vendor, mobile-delivery, api
**Dependencies:** None upstream; blocks public launch
**Review checkpoint:** Lawyer must review draft text before final commit
**Estimated LOE:** L (>4h, possibly multi-week with lawyer turnaround)
**Acceptance:** Every P0 legal finding has rewritten copy reviewed and approved by external lawyer.

[... repeat for each of CW-02 through CW-10 ...]

## How to execute a phase

Each CW-NN phase becomes its own `/gsd-plan-phase`:

```
/gsd-plan-phase CW-01
```

The phase plan loads `REWRITE-BACKLOG.md` (this file), the relevant findings from `findings/<category>.md`, and the style guide, and produces a rewrite plan per finding.

## How to update the backlog

If new findings appear during execution (e.g., a CW-02 rewrite surfaces a related issue), add the finding to the appropriate `findings/<category>.md`, regenerate the `AUDIT-FINDINGS.md` index, and add to the appropriate CW-NN bundle here.
```

- [ ] **Step 4: Sanity-check bundle coverage**

Run:
```bash
BACKLOG_IDS=$(grep -oE "(TW|LEG|BA|BV)-[0-9]+" docs/content-audit/REWRITE-BACKLOG.md | sort -u | wc -l)
INDEX_IDS=$(grep -oE "(TW|LEG|BA|BV)-[0-9]+" docs/content-audit/AUDIT-FINDINGS.md | sort -u | wc -l)
echo "backlog=$BACKLOG_IDS  index=$INDEX_IDS"
```

Expected: backlog == index. Every finding must belong to exactly one CW-NN phase.

- [ ] **Step 5: Commit**

```bash
git add docs/content-audit/REWRITE-BACKLOG.md
git commit -m "docs(content-audit): rewrite backlog — 10 sequenced phases (CW-01..CW-10)"
```

---

## Task 16: Update README and verify success criteria

**Files:**
- Modify: `docs/content-audit/README.md`

- [ ] **Step 1: Rewrite README.md with final structure**

Replace the stub README with:

```markdown
# Home Chef Content Audit

Comprehensive audit of all user-facing static content across the 7 Home Chef apps + Go API. Produces a sequenced execution backlog for downstream rewrite phases (CW-01..CW-10).

Design spec: [../superpowers/specs/2026-05-13-content-audit-design.md](../superpowers/specs/2026-05-13-content-audit-design.md)
Execution plan: [../superpowers/plans/2026-05-13-content-audit-execution.md](../superpowers/plans/2026-05-13-content-audit-execution.md)

## Artifacts

| File | Status | Purpose |
|---|---|---|
| [INVENTORY-SCHEMA.md](./INVENTORY-SCHEMA.md) | DONE | Inventory row schema reference |
| [CONTENT-INVENTORY.md](./CONTENT-INVENTORY.md) | DONE | Every user-facing string, organized by category × app |
| [STYLE-GUIDE.md](./STYLE-GUIDE.md) | DONE | Voice, persona tone, vocabulary, microcopy formulas |
| [lens-briefs/](./lens-briefs/) | DONE | Reusable prompts for the four audit lenses |
| [AUDIT-FINDINGS.md](./AUDIT-FINDINGS.md) | DONE | Rolled-up index of all findings (sortable) |
| [findings/](./findings/) | DONE | Per-category finding detail |
| [REWRITE-BACKLOG.md](./REWRITE-BACKLOG.md) | DONE | 10-phase prioritized execution sequence |

## Reading order

1. **`STYLE-GUIDE.md`** — the voice the audit measures against
2. **`REWRITE-BACKLOG.md`** — the sequenced execution phases (CW-01..CW-10)
3. **`AUDIT-FINDINGS.md`** — drill into specific findings by lens / severity
4. **`findings/<category>.md`** — full detail for any specific finding
5. **`CONTENT-INVENTORY.md`** — coverage reference for a specific surface

## Headline numbers

- Surfaces inventoried: <N>
- Findings: <N> (<P0>:P0, <P1>:P1, <P2>:P2, <P3>:P3)
- Launch-blockers: <N> findings across CW-01..CW-04
- Lawyer review required: <N> findings (all CW-01 plus selected from CW-02..CW-04)

## Next step

Run `/gsd-plan-phase CW-01` to start the first execution phase (legal launch-blockers).
```

- [ ] **Step 2: Verify success criteria from the design spec**

Walk through each success criterion from `docs/superpowers/specs/2026-05-13-content-audit-design.md` Section 7:

| Criterion | Met? | Evidence |
|---|---|---|
| 1. CONTENT-INVENTORY.md covers every visible static string across 7 apps + API | ✓ | Spot-check from Task 2 Step 3 |
| 2. STYLE-GUIDE.md concrete enough for copywriter/LLM rewrites | ✓ | 7 sections, 60+ vocab pairs, 5 microcopy formulas |
| 3. AUDIT-FINDINGS.md has ≥1 finding per high-traffic surface; P0 legal flags lawyer review | ✓ | Sanity-checks from Tasks 5–13 |
| 4. REWRITE-BACKLOG.md sequenced so `/gsd-plan-phase CW-01` is immediately invocable | ✓ | Phase Detail section enumerates finding_ids per phase |
| 5. Four artifacts readable in order without reading code | ✓ | README reading-order section |

Document any "not met" with a follow-up task at the end of the README.

- [ ] **Step 3: Commit**

```bash
git add docs/content-audit/README.md
git commit -m "docs(content-audit): finalize README with artifact index and success criteria"
```

- [ ] **Step 4: Final verification — git log review**

Run:
```bash
git log --oneline -20 -- docs/content-audit/
```

Expected: ~14 commits covering directory scaffold, schema, inventory, style guide, 4 lens briefs, 9 category findings, audit-findings index, rewrite backlog, README finalization.

---

## Execution sequencing notes

- **Task 0 → Task 1 → Task 2 → Task 3 → Task 4** are sequential. Each depends on the previous.
- **Tasks 5–13** can run in any order once Tasks 0–4 are complete; they don't depend on each other.
  - For maximum parallelism, dispatch all 9 category audits across multiple sessions, or run them sequentially in one session (the per-category lens dispatches inside each task are themselves parallel-4).
- **Task 14** depends on Tasks 5–13 all complete.
- **Task 15** depends on Task 14.
- **Task 16** depends on Task 15.

## Stop conditions

If during execution you encounter:

- **Lens agents return contradictory recommendations on the same surface** — flag in `findings/<category>.md` as "lens conflict; needs human resolution" and continue. Do NOT silently pick one lens's answer.
- **Inventory agent returns suspiciously few rows for an app** (e.g., apps/web returns <100 surfaces) — re-run with a more specific scope before proceeding. The inventory is the foundation; under-counting cascades.
- **Legal lens hallucinates a regulation that doesn't exist** — every Legal finding cites a section number; spot-check 3 random citations against actual DPDP Act / FSSAI texts. If hallucinations are widespread, switch the Legal lens prompt to "flag potential exposure areas" instead of citing specific sections.
- **A category has zero findings from all four lenses** — likely an inventory or prompt failure. Re-dispatch.

## Out of scope (reminder)

- Actual content rewrites (those are CW-01..CW-10)
- Translation / i18n setup
- Drafting new legal docs from scratch (lawyer's job; audit produces gap recommendations)
- Visual design changes
- Data-model changes (e.g., adding `allergens` field is a separate phase)
- US/EU/UK jurisdictional audit
- SEO growth strategy
