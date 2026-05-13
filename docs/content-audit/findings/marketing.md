# Findings — Marketing

Category: marketing
Lenses applied: technical-writer, legal, business-analyst, brand-voice
Inventory slice size: 22 surfaces
Total findings: 106

## Severity summary

| Severity | TW | Legal | BA | BV | Total |
|---|---|---|---|---|---|
| P0 | 1 | 10 | 12 | 8 | 31 |
| P1 | 6 | 14 | 4 | 4 | 28 |
| P2 | 9 | 5 | 4 | 7 | 25 |
| P3 | 9 | 6 | 2 | 5 | 22 |

## Cross-lens consensus surfaces

Surfaces flagged by 4 lenses (every lens agreed there's a problem):

- **`vp-mkt-register-benefits-list`** — flagged by TW(P0), Legal(P0, P1×2), BA(P0×2, P1), BV(P0) — vendor register benefits: "Zero commission first month / risk-free", "Weekly payouts", "thousands of home chefs", "Never miss an order"
- **`web-mkt-brand-tagline`** — flagged by TW(P3), Legal(P3), BA(P3), BV(P3) — Logo tagline "Homemade Food Delivered" case + brand-statement consolidation
- **`web-mkt-browse-heading`** — flagged by TW(P2), Legal(P2), BA(P2), BV(P2) — "Discover talented home chefs serving authentic homemade food" filler + regulated "authentic" descriptor
- **`web-mkt-hero-variants`** — flagged by TW(P1), Legal(P1), BA(P0), BV(P0, P1) — shared HeroSection hardcodes "500+ / 4.8 / 30min" Stats across every consuming page
- **`web-mkt-landing-become-chef`** — flagged by TW(P2, P3×2), Legal(P1×2), BA(P0, P1), BV(P2, P3) — "Love Cooking? Share Your Talent" + dead /become-chef and /chef-resources routes + income/KYC disclosure gaps
- **`web-mkt-landing-catering-cta`** — flagged by TW(P2, P3), Legal(P1×2), BA(P2), BV(P2) — "Planning an Event? / Perfect for parties..." catering CTA missing scope, DPDP notice, marketplace disclosure
- **`web-mkt-landing-hero-badge`** — flagged by TW(P1), Legal(P0), BA(P0), BV(P0) — hardcoded "500+ Home Chefs Near You" on landing hero badge
- **`web-mkt-landing-how-it-works`** — flagged by TW(P2), Legal(P2, P3), BA(P2), BV(P2) — "Get Delicious Food in 3 Steps" + "place your order securely" + "fresh homemade food delivered to your doorstep" filler
- **`web-mkt-landing-trust-badges`** — flagged by TW(P1), Legal(P0), BA(P0), BV(P0) — hardcoded "500+ Home Chefs / 4.8 Average Rating / 30-45 min Delivery" trust row
- **`web-mkt-landing-why-choose`** — flagged by TW(P1, P2), Legal(P0×2, P1×3, P2), BA(P0×2, P2), BV(P0, P2) — "The Fe3dr Difference" block: "Verified Chefs" FSSAI claim, "Made with Love", "Fast Delivery 30-45 minutes", "Secure Payments", "thousands of happy customers"
- **`web-mkt-layout-footer`** — flagged by TW(P3), Legal(P0×2, P1), BA(P3), BV(P1) — "© Fe3dr" brand inconsistency, missing grievance officer / refund / FSSAI links, brand-blurb voice drift
- **`web-mkt-login-testimonial`** — flagged by TW(P1), Legal(P0, P1), BA(P0), BV(P1) — fabricated "Sarah M., Happy Customer" testimonial on login page
- **`web-mkt-register-benefits`** — flagged by TW(P1), Legal(P0, P1), BA(P0), BV(P0×2) — RegisterPage benefits rail: "Access to 500+ home chefs", "hundreds of home chefs", "Join Fe3dr Today"

Surfaces flagged by 3 lenses (also high-priority rewrites):

- **`web-mkt-catering-request-heading`** — flagged by TW(P3), Legal(P2), BA(P1) — catering wizard "Event Details / Tell us about your event" needs DPDP notice + post-submission expectation copy
- **`web-mkt-hero-how-it-works-cta`** — flagged by TW(P3), BA(P0), BV(P3) — "How It Works" button: title case + dead /how-it-works route
- **`web-mkt-landing-cuisines`** — flagged by TW(P3), Legal(P3), BV(P2) — "Explore Flavors / Discover authentic dishes from around the world" filler + cuisine availability promise
- **`web-mkt-landing-featured-chefs-heading`** — flagged by TW(P3×2), Legal(P1), BV(P0, P3) — "Top Rated / Featured Chefs / Our community's favorite home chefs" methodology + "View All Chefs" case
- **`web-mkt-landing-hero-subtitle`** — flagged by TW(P2), Legal(P2), BV(P2) — "Discover talented home chefs in your neighborhood... right to your doorstep" filler-stack hero subtitle
- **`web-mkt-landing-hero-title`** — flagged by Legal(P1), BA(P1), BV(P1) — "Homemade Food, Delivered Fresh" regulated "Fresh" descriptor + Find Food CTA/destination mismatch + HomePage vs HeroSection drift
- **`web-mkt-landing-search-cta`** — flagged by TW(P2), Legal(P2), BV(P3) — "Find Food" primary CTA: title case + geographic-availability gating

## Technical Writer findings

```yaml
findings:
  # ============================================================
  # AI-SLOP / UNVERIFIED METRICS  (Style Guide Rule 5 — Restraint)
  # ============================================================
  - finding_id: TW-001
    surface_id: web-mkt-landing-hero-badge
    lens: technical-writer
    severity: P1
    issue: "Hardcoded social-proof metric '500+ Home Chefs Near You' in hero badge — unverified number on entry surface, AI-slop placeholder copy."
    evidence_excerpt: "500+ Home Chefs Near You"
    recommendation: "Remove the hardcoded count. Either bind to a live signal (e.g., 'Home chefs in {city}') driven by API, or replace with a non-numeric brand pill like 'Homemade, never restaurant'. Never ship hardcoded metrics in marketing surfaces."
    depends_on: null

  - finding_id: TW-002
    surface_id: web-mkt-landing-trust-badges
    lens: technical-writer
    severity: P1
    issue: "Hardcoded '500+ Home Chefs' + '4.8 Average Rating' + '30-45 min Delivery' trust row. All three are unverified, hardcoded metrics presented as factual claims on the landing page."
    evidence_excerpt: "{ icon: ChefHat, label: '500+ Home Chefs' }, { icon: Star, label: '4.8 Average Rating' }, { icon: Clock, label: '30-45 min Delivery' }"
    recommendation: "Replace with API-backed live stats (chef count, current avg rating, median delivery time computed server-side) or remove the trust-row entirely until real data exists. Hardcoded marketing numbers are P1 because they could become P0 false-advertising claims if the real numbers diverge."
    depends_on: null

  - finding_id: TW-003
    surface_id: web-mkt-landing-why-choose
    lens: technical-writer
    severity: P1
    issue: "Generic AI-slop social proof: 'Join thousands of happy customers enjoying homemade food.' Unverified metric ('thousands') + filler superlative ('happy customers') — violates Rule 5 (restraint) and Rule 1 (confident, not loud)."
    evidence_excerpt: "Join thousands of happy customers enjoying homemade food"
    recommendation: "Rewrite without a metric: 'Made by people who cook for a living, not at scale.' or 'Real cooks. Real kitchens. Real food.' If keeping a count, bind it to live data and pluralise correctly."
    depends_on: null

  - finding_id: TW-004
    surface_id: web-mkt-register-benefits
    lens: technical-writer
    severity: P1
    issue: "Hardcoded 'Access to 500+ home chefs' in registration benefits list — same unverified metric as the homepage hero, repeated on the signup conversion surface."
    evidence_excerpt: "const BENEFITS = ['Access to 500+ home chefs', 'Authentic homemade food', 'Fast & reliable delivery', 'Support local home chefs']"
    recommendation: "Drop the number. Use 'Home chefs near you' or bind to live API count. Keeping '500+' creates a contract that customers can verify on /chefs — if the directory shows fewer, this becomes P0 false advertising."
    depends_on: TW-001

  - finding_id: TW-005
    surface_id: web-mkt-hero-variants
    lens: technical-writer
    severity: P1
    issue: "Shared HeroSection.tsx Stats component hardcodes '500+ Home Chefs', '4.8 Avg Rating', '30min Avg Delivery' — duplicates the landing-page AI-slop in a reusable shared component, multiplying the risk."
    evidence_excerpt: "const stats = [{ icon: ChefHat, value: '500+', label: 'Home Chefs' }, { icon: Star, value: '4.8', label: 'Avg Rating' }, { icon: Clock, value: '30min', label: 'Avg Delivery' }]"
    recommendation: "Remove the Stats() sub-component or refactor it to accept props bound to live data. Shared components should never embed hardcoded marketing numbers — fix once, fix everywhere."
    depends_on: TW-002

  - finding_id: TW-006
    surface_id: web-mkt-login-testimonial
    lens: technical-writer
    severity: P1
    issue: "Hardcoded testimonial from fabricated user 'Sarah M., Happy Customer' on the login page. No source, no real reviewer, generic 'Happy Customer' attribution — textbook AI-slop social proof."
    evidence_excerpt: "\"Fe3dr has changed how I eat. Finally, real homemade food that reminds me of my mom's cooking!\" — Sarah M., Happy Customer"
    recommendation: "Remove until a real verified testimonial exists with full name, photo, and verifiable source (review platform, video). If kept temporarily, mark clearly as 'Illustrative example' — but the better fix is to delete and let the photo carry the page."
    depends_on: null

  - finding_id: TW-007
    surface_id: vp-mkt-register-benefits-list
    lens: technical-writer
    severity: P0
    issue: "Vendor portal register page promises 'Zero commission first month' as a concrete commercial commitment in marketing copy. If the product cannot honor this for every signup (or the policy changes), this is a factually misleading claim — TW P0 by definition (commitment the product can't keep)."
    evidence_excerpt: "{ title: 'Zero commission first month', desc: 'Get started completely risk-free' }"
    recommendation: "Either (a) verify with product/legal that zero-commission-first-month is a binding, currently-active policy and add a footnote linking to terms; or (b) replace with a softer non-binding line like 'Low platform fees' or 'Pricing built for home kitchens'. 'Completely risk-free' is also banned-style absolute language — soften to 'No upfront cost'."
    depends_on: null

  # ============================================================
  # VOICE / RULE-1 DRIFT (loud, generic, filler)
  # ============================================================
  - finding_id: TW-008
    surface_id: web-mkt-landing-hero-subtitle
    lens: technical-writer
    severity: P2
    issue: "Hero subtitle is generic landing-page filler — 'Discover talented home chefs in your neighborhood and enjoy authentic, homemade meals delivered right to your doorstep.' Combines two banned-style flourishes ('talented', 'authentic') and ends with the cliché 'right to your doorstep'."
    evidence_excerpt: "Discover talented home chefs in your neighborhood and enjoy authentic, homemade meals delivered right to your doorstep."
    recommendation: "Tighten to ≤15 words, one verb, one image. e.g. 'Order from home cooks in your neighbourhood. Delivered hot.' Per Rule 3, the photo carries the brand — copy supports, doesn't compete."
    depends_on: null

  - finding_id: TW-009
    surface_id: web-mkt-browse-heading
    lens: technical-writer
    severity: P2
    issue: "Browse page intro duplicates the same 'Discover talented home chefs serving authentic homemade food' filler — repeated cliché stack ('talented' + 'authentic') across surfaces. Within-surface voice drift from the more restrained tone elsewhere."
    evidence_excerpt: "Explore Home Chefs / Discover talented home chefs serving authentic homemade food"
    recommendation: "Replace with action-oriented intro: 'Home chefs near you. Filter by cuisine, diet, or delivery time.' Cuts banned-style adjectives and tells the user what they can do on the page."
    depends_on: TW-008

  - finding_id: TW-010
    surface_id: web-mkt-landing-how-it-works
    lens: technical-writer
    severity: P2
    issue: "'Get Delicious Food in 3 Steps' — 'Delicious' is filler adjective; 'we make it simple' is generic SaaS-speak ('From discovery to delivery, we make it simple to enjoy homemade food'). Voice drifts away from confident-restrained toward promotional-loud."
    evidence_excerpt: "Get Delicious Food in 3 Steps / From discovery to delivery, we make it simple to enjoy homemade food"
    recommendation: "Heading: 'How ordering works'. Sub: 'Three steps from browse to plate.' Removes filler adjective and 'we make it simple' cliché. Title case on heading is fine; banned word 'delicious' is purely decorative here."
    depends_on: null

  - finding_id: TW-011
    surface_id: web-mkt-landing-why-choose
    lens: technical-writer
    severity: P2
    issue: "Feature card 'Made with Love / Every meal is prepared fresh with authentic family recipes' — 'Made with Love' is exactly the banned brand-drift term from style guide Sec 3. 'Authentic family recipes' is filler."
    evidence_excerpt: "Made with Love / Every meal is prepared fresh with authentic family recipes"
    recommendation: "Replace card title with operational: 'Cooked to order' or 'Same-day cooking'. Body: 'Your meal is prepared after you order, never reheated.' Removes banned 'made with love' kitsch and tells the user a concrete fact."
    depends_on: null

  - finding_id: TW-012
    surface_id: web-mkt-landing-become-chef
    lens: technical-writer
    severity: P2
    issue: "'Love Cooking? Share Your Talent' + 'Turn your passion into income' — both phrases are generic creator-economy / gig-app filler. Within-surface drift from restrained brand voice."
    evidence_excerpt: "Love Cooking? Share Your Talent / Turn your passion into income. Join our community of home chefs and start earning by sharing your delicious homemade food."
    recommendation: "Heading: 'Cook from home. Get paid.' Sub: 'Set your menu, your prices, and the days you cook. We handle orders and payouts.' Functional, vendor-tone-appropriate, removes 'passion'/'talent'/'delicious' filler."
    depends_on: null

  - finding_id: TW-013
    surface_id: web-mkt-landing-featured-chefs-heading
    lens: technical-writer
    severity: P3
    issue: "'Our community's favorite home chefs' uses the vague 'community's favourite' generic — better to use a concrete signal like 'highest-rated this month' or just drop the subtitle."
    evidence_excerpt: "Top Rated / Featured Chefs / Our community's favorite home chefs / View All Chefs"
    recommendation: "Sub: 'Highest-rated this week.' (computed from real ratings) or remove the subtitle entirely. 'Community's favourite' is unverifiable filler."
    depends_on: null

  - finding_id: TW-014
    surface_id: web-mkt-landing-cuisines
    lens: technical-writer
    severity: P3
    issue: "'Explore Flavors / Discover authentic dishes from around the world' — 'authentic' again (third occurrence on the same page), 'around the world' is filler. Style guide flags 'authentic' as overused in this codebase."
    evidence_excerpt: "Cuisines / Explore Flavors / Discover authentic dishes from around the world"
    recommendation: "Heading: 'Cuisines'. Sub: 'South Indian, Italian, Japanese, and more.' List the actual options instead of hand-waving at 'around the world'."
    depends_on: null

  - finding_id: TW-015
    surface_id: web-mkt-landing-catering-cta
    lens: technical-writer
    severity: P3
    issue: "'Get catering quotes from multiple home chefs. Perfect for parties, corporate events, and special occasions.' — 'Perfect for' is filler. 'special occasions' is generic."
    evidence_excerpt: "Catering Services / Planning an Event? / Get catering quotes from multiple home chefs. Perfect for parties, corporate events, and special occasions."
    recommendation: "Sub: 'Tell us the date, headcount, and cuisine. Home chefs send quotes within 24 hours.' Replaces fluff with a concrete user-flow promise the product can actually keep."
    depends_on: null

  # ============================================================
  # MICROCOPY FORMULA / BUTTON RULES
  # ============================================================
  - finding_id: TW-016
    surface_id: web-mkt-landing-search-cta
    lens: technical-writer
    severity: P2
    issue: "Primary search CTA 'Find Food' is title-case; style guide Sec 4 button format requires sentence case ('Find food'). Two-word verb-first form is otherwise good."
    evidence_excerpt: "Find Food"
    recommendation: "Use sentence case: 'Find food'. Or be more specific: 'Search chefs'."
    depends_on: null

  - finding_id: TW-017
    surface_id: web-mkt-hero-search
    lens: technical-writer
    severity: P2
    issue: "Shared HeroSection's CTA 'Find Chefs' is title-case (style guide says sentence case). Also inconsistent with HomePage's 'Find Food' — same surface group ships two different verbs ('Find food' vs 'Find chefs') with no semantic difference."
    evidence_excerpt: "Find Chefs"
    recommendation: "Pick one: 'Find chefs' (sentence case) and use it everywhere. Cross-surface inconsistency on a primary CTA is a content-system bug."
    depends_on: TW-016

  - finding_id: TW-018
    surface_id: web-mkt-hero-how-it-works-cta
    lens: technical-writer
    severity: P3
    issue: "'How It Works' button is title case; style guide requires sentence case for buttons."
    evidence_excerpt: "How It Works"
    recommendation: "Change to 'How it works'. Same fix wherever this label appears."
    depends_on: null

  - finding_id: TW-019
    surface_id: web-mkt-landing-catering-cta
    lens: technical-writer
    severity: P2
    issue: "'Request Catering Quote' button is title case (3 words is fine, case is not). Also wordy — 'Request a quote' or 'Get quotes' is shorter and clearer."
    evidence_excerpt: "Request Catering Quote"
    recommendation: "Use 'Get quotes' (2 words, sentence case, verb-first). Context already says 'Catering Services'."
    depends_on: null

  - finding_id: TW-020
    surface_id: web-mkt-landing-become-chef
    lens: technical-writer
    severity: P3
    issue: "'Become a Chef' and 'Learn More' buttons both title-case; 'Learn More' violates style guide ban on standalone 'Learn more →' without topic context."
    evidence_excerpt: "Become a Chef / Learn More"
    recommendation: "'Become a chef' (sentence case). Replace 'Learn More' with a contextual link like 'See chef resources' or 'Read the cook handbook'. Standalone 'Learn more' is banned."
    depends_on: null

  - finding_id: TW-021
    surface_id: web-mkt-landing-featured-chefs-heading
    lens: technical-writer
    severity: P3
    issue: "'View All Chefs' is title case (sentence-case rule)."
    evidence_excerpt: "View All Chefs"
    recommendation: "'View all chefs' — or just 'See all'. Same fix on the 'View All' link in the Cuisines section."
    depends_on: TW-020

  # ============================================================
  # CONSISTENCY / NAMING
  # ============================================================
  - finding_id: TW-022
    surface_id: web-mkt-layout-footer
    lens: technical-writer
    severity: P3
    issue: "Footer brand lead-in 'Connecting you with home chefs for authentic, homemade food delivered to your doorstep.' uses 'authentic' (banned-style filler, recurs on page) and 'delivered to your doorstep' (cliché — second occurrence on home page)."
    evidence_excerpt: "Connecting you with home chefs for authentic, homemade food delivered to your doorstep."
    recommendation: "Tighten: 'Order homemade food from chefs near you.' Removes 'authentic' and the doorstep cliché. Single sentence, plain English, matches Rule 2."
    depends_on: null

  - finding_id: TW-023
    surface_id: web-mkt-brand-tagline
    lens: technical-writer
    severity: P3
    issue: "Logo tagline 'Homemade Food Delivered' is title case but is meant as a tagline string; usable but inconsistent with hero H1 ('Homemade Food, Delivered Fresh'). Two near-identical taglines is sloppy."
    evidence_excerpt: "Homemade Food Delivered"
    recommendation: "Either drop the tagline (logo carries the brand) or align exactly: 'Homemade food, delivered fresh.' — sentence case, comma matches hero. Single source of truth."
    depends_on: null

  # ============================================================
  # SENTENCE LENGTH / READING EASE
  # ============================================================
  - finding_id: TW-024
    surface_id: web-mkt-landing-become-chef
    lens: technical-writer
    severity: P3
    issue: "Body paragraph runs 22 words but stuffs two unrelated ideas into one sentence: 'Turn your passion into income. Join our community of home chefs and start earning by sharing your delicious homemade food.' The second sentence is 18 words; cut 'delicious' (banned-style filler) and split the join+earn idea."
    evidence_excerpt: "Turn your passion into income. Join our community of home chefs and start earning by sharing your delicious homemade food."
    recommendation: "Rewrite per TW-012; new copy keeps sentences under 15 words and removes filler. This finding is the length-rule complement to TW-012's voice-drift fix."
    depends_on: TW-012

  - finding_id: TW-025
    surface_id: web-mkt-catering-request-heading
    lens: technical-writer
    severity: P3
    issue: "'Event Details / Tell us about your event' is acceptable as a step heading but 'Tell us about your event' duplicates the heading meaning — could be a more specific helper telling the user what fields are coming."
    evidence_excerpt: "Event Details / Tell us about your event"
    recommendation: "Heading 'Event details' (sentence case). Helper: 'Date, headcount, cuisine, and budget — about 60 seconds.' Sets expectation for length and content."
    depends_on: null
```

## Legal findings

```yaml
# Legal lens findings — MARKETING category
# Jurisdiction: India + generic best-practice
# Auditor note: NOT a lawyer. Every finding flags depends_on: "needs lawyer review"
#                for human binding-text drafting.

findings:
  # ─────────────────────────────────────────────────────────────────────
  # P0 — Unverified social-proof metrics (ASCI / CP E-Commerce Rules 2020)
  # ─────────────────────────────────────────────────────────────────────
  - finding_id: LEG-MKT-001
    surface_id: web-mkt-landing-hero-badge
    lens: legal
    severity: P0
    issue: "Hardcoded social-proof metric '500+ Home Chefs Near You' rendered without substantiation, geographic qualifier, or 'as of date' — misleading advertising exposure"
    evidence_excerpt: "500+ Home Chefs Near You"
    recommendation: "Either (a) bind the number to a live count from the chefs API with a verifiable 'as-of' timestamp and a geographic qualifier ('Near You' is location-dependent), or (b) remove the metric. If retained, store the substantiation file (count source + date) per ASCI Code Chapter I.1 'all claims, direct or implied, must be capable of substantiation'. Avoid the '+' suffix unless the rounded figure is documented."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 4(11) (no false/misleading advertising); ASCI Code Chapter I.1 (substantiation of claims)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-MKT-002
    surface_id: web-mkt-landing-trust-badges
    lens: legal
    severity: P0
    issue: "Trust badge row asserts three quantified marketing claims ('500+ Home Chefs', '4.8 Average Rating', '30-45 min Delivery') without substantiation, methodology, or 'as of' date"
    evidence_excerpt: "500+ Home Chefs / 4.8 Average Rating / 30-45 min Delivery"
    recommendation: "Each claim needs substantiation per ASCI guidelines: (1) chef count must reference a live data source with timestamp; (2) '4.8 Average Rating' needs sample size disclosure (e.g., 'across 12,400 reviews, as of <date>') and the rating must be re-checked periodically; (3) '30-45 min Delivery' is a performance claim that must be qualified ('typical', 'in select pin codes', 'subject to traffic') or it constitutes a deliverable promise the platform may not be able to honour, exposing it under Rule 4(11)."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 4(11); ASCI Code Chapter I.1 (substantiation); ASCI Guidelines on Misleading Advertising"
    depends_on: "needs lawyer review"

  - finding_id: LEG-MKT-003
    surface_id: web-mkt-register-benefits
    lens: legal
    severity: P0
    issue: "Marketing claim '500+ home chefs' duplicated on registration entry page reinforces unsubstantiated metric at the conversion moment — consumer-decision point under CP E-Comm Rules"
    evidence_excerpt: "Access to 500+ home chefs"
    recommendation: "Same substantiation requirement as LEG-MKT-001/002. The legal exposure is HIGHER here because the claim is presented at the registration conversion moment — i.e., the consumer is forming the contractual relationship in reliance on the claim. Either bind to a live count or remove."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 4(11) and Rule 5(3) (no misrepresentation to induce a transaction); ASCI Code Chapter I.1"
    depends_on: "needs lawyer review"

  # ─────────────────────────────────────────────────────────────────────
  # P0 — Food-safety claims in marketing (FSSAI consistency required)
  # ─────────────────────────────────────────────────────────────────────
  - finding_id: LEG-MKT-004
    surface_id: web-mkt-landing-why-choose
    lens: legal
    severity: P0
    issue: "Marketing card states 'All our home chefs are verified for food safety and quality' — this is a regulated food-safety claim that requires evidence of an actual verification process consistent with FSSAI registration/licensing"
    evidence_excerpt: "Verified Chefs / All our home chefs are verified for food safety and quality"
    recommendation: "Either (a) the platform must document and operate an actual chef-verification programme (FSSAI registration check, food-handler training certificate, kitchen photos, periodic re-verification) and link the consumer to the verification status on each chef profile, or (b) the claim must be removed. The phrase 'verified for food safety' implies regulatory oversight; if FSSAI registration is not in fact collected/displayed per FSS (Licensing & Registration) Regulations 2011, this is an unsubstantiated regulatory claim under Rule 4(11)."
    citation: "FSS (Licensing & Registration) Regulations 2011; FSS Act 2006 §31; Consumer Protection (E-Commerce) Rules 2020 Rule 4(11); ASCI Guidelines on Misleading Advertising"
    depends_on: "needs lawyer review"

  - finding_id: LEG-MKT-005
    surface_id: web-mkt-landing-why-choose
    lens: legal
    severity: P1
    issue: "'Made with Love / Every meal is prepared fresh with authentic family recipes' — 'fresh' and 'authentic' are unqualified food-quality claims that map to FSSAI's restricted descriptors"
    evidence_excerpt: "Made with Love / Every meal is prepared fresh with authentic family recipes"
    recommendation: "Per FSS (Advertising and Claims) Regulations 2018, descriptors like 'fresh' and 'authentic' must not mislead. 'Fresh' specifically has a working definition (not preserved, recently prepared). Either define what 'fresh' means on this platform (e.g., 'prepared within X hours of pickup') in a tooltip / linked policy, or use less regulated alternatives ('home-cooked', 'home-prepared'). 'Authentic family recipes' is harder to substantiate per-chef — consider a softer phrasing."
    citation: "FSS (Advertising and Claims) Regulations 2018 §4 (no misleading descriptors); Consumer Protection (E-Commerce) Rules 2020 Rule 4(11)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-MKT-006
    surface_id: web-mkt-landing-hero-title
    lens: legal
    severity: P1
    issue: "Hero title 'Homemade Food, Delivered Fresh' uses 'Fresh' as the herb-accent marketing word without a substantiation pathway"
    evidence_excerpt: "Homemade Food, Delivered Fresh"
    recommendation: "Same as LEG-MKT-005 — 'Fresh' is a regulated descriptor under FSSAI claims rules. Either define 'fresh' platform-wide (in T&C or a linked claims page) and ensure operational compliance (e.g., max prep-to-delivery time), or replace with a non-regulated descriptor."
    citation: "FSS (Advertising and Claims) Regulations 2018 §4; ASCI Guidelines on Misleading Advertising"
    depends_on: "needs lawyer review"

  - finding_id: LEG-MKT-007
    surface_id: web-mkt-landing-how-it-works
    lens: legal
    severity: P2
    issue: "'Get fresh homemade food delivered to your doorstep' restates 'fresh' as a deliverable promise without definition"
    evidence_excerpt: "Get fresh homemade food delivered to your doorstep"
    recommendation: "Align with the platform-wide 'fresh' definition once established (see LEG-MKT-005). If 'fresh' cannot be operationalised, drop it. Avoid using the same regulated descriptor in 4+ places without a single source of truth."
    citation: "FSS (Advertising and Claims) Regulations 2018 §4; best-practice (claim-consistency across surfaces)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-MKT-008
    surface_id: web-mkt-browse-heading
    lens: legal
    severity: P2
    issue: "Browse-chefs page intro 'Discover talented home chefs serving authentic homemade food' — 'authentic' is a regulated quality descriptor"
    evidence_excerpt: "Discover talented home chefs serving authentic homemade food"
    recommendation: "Per FSS Advertising Claims Regulations, 'authentic' implies origin/recipe verification. Either drop the word ('homemade' alone is sufficient and accurate to the model) or define what 'authentic' verification means."
    citation: "FSS (Advertising and Claims) Regulations 2018 §4 (no implied superiority without substantiation)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-MKT-009
    surface_id: web-mkt-landing-hero-subtitle
    lens: legal
    severity: P2
    issue: "Hero subtitle 'Discover talented home chefs in your neighborhood and enjoy authentic, homemade meals delivered right to your doorstep' compounds 'authentic' with a geographic implication ('in your neighborhood')"
    evidence_excerpt: "Discover talented home chefs in your neighborhood and enjoy authentic, homemade meals delivered right to your doorstep."
    recommendation: "Two issues: (1) 'authentic' — see LEG-MKT-008; (2) 'in your neighborhood' is a geographic promise. If the platform operates in only some pin codes, this claim is misleading to visitors outside coverage. Add a 'check availability in your area' qualifier or geo-detect before showing this claim."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 4(11); FSS (Advertising and Claims) Regulations 2018"
    depends_on: "needs lawyer review"

  # ─────────────────────────────────────────────────────────────────────
  # P0 — Performance / delivery-time claims
  # ─────────────────────────────────────────────────────────────────────
  - finding_id: LEG-MKT-010
    surface_id: web-mkt-landing-why-choose
    lens: legal
    severity: P0
    issue: "'Fast Delivery / Reliable delivery to your doorstep within 30-45 minutes' is a quantified performance promise that may bind the platform under CP E-Commerce Rules and create a guaranteed-delivery expectation"
    evidence_excerpt: "Fast Delivery / Reliable delivery to your doorstep within 30-45 minutes"
    recommendation: "Quantified time-window promises are HIGH-risk: a consumer who experiences a 50-minute delivery has a colourable claim of misleading advertising and possibly a refund right. Recommend (a) qualify with 'typical', 'in eligible areas', 'subject to chef availability and traffic'; (b) tie to a documented SLA with a refund/credit mechanism if breached; or (c) drop the specific window in favour of a softer claim ('delivery to your doorstep'). Note: also restated in trust badges (LEG-MKT-002) — fix once, propagate."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 4(11) and Rule 5(3); Consumer Protection Act 2019 §2(28) (unfair trade practice — false delivery promises); ASCI Code Chapter I.1"
    depends_on: "needs lawyer review"

  - finding_id: LEG-MKT-011
    surface_id: web-mkt-hero-variants
    lens: legal
    severity: P1
    issue: "Shared HeroSection component holds 4 hero variants used across home/chefs/catering/feed pages — claims propagate to every page that imports it. Single source of truth for marketing claims is the weakest link"
    evidence_excerpt: "Homemade Food, Delivered Fresh / Discover Local Chefs / Catering for Every Occasion / Food Inspiration / Browse Local Chefs / Start Exploring / Plan Your Event / Explore Feed"
    recommendation: "Catalogue every claim emitted from this shared component and run each through the same substantiation test (LEG-MKT-001 through -010). Where the live HomePage hero (HomePage.tsx) and the reusable HeroSection diverge in copy ('Delivered Fresh' vs 'Discover Local Chefs'), only one set is in production today — audit which is authoritative and retire the unused variant to reduce drift and legal-claim surface area."
    citation: "best-practice (single source of truth for advertising claims); Consumer Protection (E-Commerce) Rules 2020 Rule 4(11)"
    depends_on: "needs lawyer review"

  # ─────────────────────────────────────────────────────────────────────
  # P0 — Testimonial / endorsement substantiation (ASCI)
  # ─────────────────────────────────────────────────────────────────────
  - finding_id: LEG-MKT-012
    surface_id: web-mkt-login-testimonial
    lens: legal
    severity: P0
    issue: "Testimonial attributed to 'Sarah M., Happy Customer' with brand-superlative claim ('Fe3dr has changed how I eat. Finally, real homemade food…') — no evidence the testimonial is from a real customer; ASCI requires endorsements to be genuine and substantiable"
    evidence_excerpt: "\"Fe3dr has changed how I eat. Finally, real homemade food that reminds me of my mom's cooking!\" — Sarah M., Happy Customer"
    recommendation: "Per ASCI Guidelines for Influencer Advertising and ASCI Code Chapter I.4, testimonials must (a) be from real customers who have actually used the service, (b) be capable of substantiation (the customer can be contacted, the statement is theirs), (c) not be misleading by selective quotation. Action: either source genuine testimonials with consent (and retain consent records), or remove the placeholder testimonial. The current Sarah M. quote on the login page reads as marketing fiction and is HIGH-risk if challenged."
    citation: "ASCI Code Chapter I.4 (testimonials); ASCI Guidelines for Influencer Advertising; Consumer Protection (E-Commerce) Rules 2020 Rule 4(11)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-MKT-013
    surface_id: web-mkt-landing-featured-chefs-heading
    lens: legal
    severity: P1
    issue: "'Top Rated' badge applied to featured-chefs section without disclosed methodology — what makes a chef 'top rated'? Rating threshold, sample size, time window all undefined"
    evidence_excerpt: "Top Rated / Featured Chefs / Our community's favorite home chefs"
    recommendation: "ASCI requires comparative claims like 'top rated' to be substantiated with methodology. Either (a) document the criteria ('rated 4.5+ across at least 20 reviews in the last 90 days') in a tooltip / linked policy, or (b) replace with a non-comparative phrase ('Featured Chefs' alone is fine; 'Top Rated' adds legal exposure with no marketing lift)."
    citation: "ASCI Code Chapter I.1 and Chapter III (comparative advertising — substantiation required); Consumer Protection (E-Commerce) Rules 2020 Rule 4(11)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-MKT-014
    surface_id: web-mkt-landing-why-choose
    lens: legal
    severity: P1
    issue: "'Join thousands of happy customers enjoying homemade food' — unsubstantiated user-count claim ('thousands') and emotional outcome claim ('happy')"
    evidence_excerpt: "Join thousands of happy customers enjoying homemade food"
    recommendation: "ASCI requires substantiation for quantified social-proof claims even when rounded ('thousands'). Either (a) bind to actual user-count data with a freshness date, or (b) drop the quantifier. 'Happy' is a subjective outcome claim — defensible only if backed by survey data (NPS, satisfaction score). Recommend softer phrasing: 'Join the home-cooked food community' (no quantified or outcome claim)."
    citation: "ASCI Code Chapter I.1 (substantiation); Consumer Protection (E-Commerce) Rules 2020 Rule 4(11)"
    depends_on: "needs lawyer review"

  # ─────────────────────────────────────────────────────────────────────
  # P0 — Vendor-side marketing promises (chef onboarding)
  # ─────────────────────────────────────────────────────────────────────
  - finding_id: LEG-MKT-015
    surface_id: vp-mkt-register-benefits-list
    lens: legal
    severity: P0
    issue: "'Zero commission first month / Get started completely risk-free' is a binding financial promise at the chef-onboarding step. If commission terms in the actual chef agreement differ, this constitutes misrepresentation inducing the contract"
    evidence_excerpt: "Zero commission first month / Get started completely risk-free"
    recommendation: "(1) Verify the actual chef T&C / commercial terms enforce zero commission in month 1 — the promise must be operationalised in the contract and billing system. (2) 'Risk-free' is an absolute claim; a chef still bears food-cost risk, time risk, and reputational risk — 'risk-free' is misleading. Replace with 'No platform commission for your first month' (precise and operational). (3) The promise needs an end-date, eligibility criteria, and a clearly stated post-month-1 commission rate (or a link to it) so the chef is not surprised."
    citation: "Indian Contract Act 1872 §17 (misrepresentation); Consumer Protection (E-Commerce) Rules 2020 Rule 4(11); ASCI Code Chapter I.1; best-practice (offer-terms must be operationally enforceable)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-MKT-016
    surface_id: vp-mkt-register-benefits-list
    lens: legal
    severity: P1
    issue: "'Weekly payouts / Get paid directly to your bank account' is a payment-flow commitment that must align with the platform's actual settlement schedule and RBI Payment Aggregator rules"
    evidence_excerpt: "Weekly payouts / Get paid directly to your bank account"
    recommendation: "(1) Verify the platform operates a weekly settlement cycle; if monthly or T+N, the marketing claim is false. (2) RBI's Payment Aggregator Guidelines require disclosure of settlement timelines in the merchant agreement — keep marketing copy aligned with the contractual SLA. (3) Add a qualifier 'subject to KYC completion and order delivery confirmation' since payouts cannot be unconditional. (4) Identify the merchant of record (chef or platform) in the underlying T&C — marketing copy here implies chef-as-merchant which has tax/KYC consequences."
    citation: "RBI Guidelines on Regulation of Payment Aggregators and Payment Gateways (Mar 2020) §8 (settlement timelines, disclosure); Consumer Protection (E-Commerce) Rules 2020 Rule 4(11)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-MKT-017
    surface_id: vp-mkt-register-benefits-list
    lens: legal
    severity: P1
    issue: "'Never miss an order with instant notifications' is an absolute reliability claim ('never', 'instant') that the platform cannot technically guarantee"
    evidence_excerpt: "Real-time order tracking / Never miss an order with instant notifications"
    recommendation: "Absolute reliability claims ('never', 'always', '100%', 'guaranteed') are flagged by ASCI as high-risk. Push notifications fail for many reasons outside platform control (device offline, OS throttling, app uninstalled). Soften to 'Real-time order notifications' — drop 'never miss' and 'instant'."
    citation: "ASCI Guidelines on Misleading Advertising (absolute claims); Consumer Protection (E-Commerce) Rules 2020 Rule 4(11)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-MKT-018
    surface_id: web-mkt-landing-become-chef
    lens: legal
    severity: P1
    issue: "'Turn your passion into income' on the consumer-facing landing page solicits chef enrolment without disclosing earnings model, commission, KYC, FSSAI registration requirement, or platform terms — borderline on income-claim regulation"
    evidence_excerpt: "Turn your passion into income. Join our community of home chefs and start earning by sharing your delicious homemade food."
    recommendation: "(1) Income-related solicitations on consumer surfaces should link to a transparent earnings page with rate cards, payout cycles, and an income disclaimer ('actual earnings vary by orders, pricing, market'). (2) Disclose chef-side obligations up front: FSSAI registration may be required for chefs above the turnover threshold; KYC required for payouts. (3) ASCI scrutinises 'turn your passion into income' framing on gig platforms — add a 'no guaranteed earnings' disclaimer and link to chef onboarding terms before the CTA."
    citation: "FSS (Licensing & Registration) Regulations 2011 (chef-side compliance); RBI Payment Aggregator Guidelines (KYC); ASCI Guidelines on Misleading Advertising (income claims); Consumer Protection (E-Commerce) Rules 2020 Rule 4(11)"
    depends_on: "needs lawyer review"

  # ─────────────────────────────────────────────────────────────────────
  # P0 — Pricing claims (catering, hidden conditions)
  # ─────────────────────────────────────────────────────────────────────
  - finding_id: LEG-MKT-019
    surface_id: web-mkt-landing-catering-cta
    lens: legal
    severity: P1
    issue: "'Get catering quotes from multiple home chefs. Perfect for parties, corporate events, and special occasions.' — no disclosure of who is liable for the catering contract (chef vs platform), no pricing model, no cancellation policy reference"
    evidence_excerpt: "Catering Services / Planning an Event? / Get catering quotes from multiple home chefs. Perfect for parties, corporate events, and special occasions."
    recommendation: "Catering = higher-value transactions with bigger consumer exposure. (1) Marketing copy must indicate that quotes are non-binding and the contract is between the customer and the chef (or platform, whichever is operative) — this affects refund rights, FSSAI licensing scope (caterers above ₹12 lakh turnover need FSSAI Central licence), and tax invoicing. (2) Add a disclosure or link 'See how catering works' near the CTA covering scope, payment terms, cancellation. (3) Avoid 'Perfect for' superlatives unless substantiated."
    citation: "FSS (Licensing & Registration) Regulations 2011 (caterer thresholds); Consumer Protection (E-Commerce) Rules 2020 Rule 5(3) (clear disclosure of seller identity); ASCI Code Chapter I.1"
    depends_on: "needs lawyer review"

  - finding_id: LEG-MKT-020
    surface_id: web-mkt-catering-request-heading
    lens: legal
    severity: P2
    issue: "Catering-request flow heading 'Tell us about your event' onboards the consumer into a high-value transaction without a visible reference to terms of service or catering-specific policies"
    evidence_excerpt: "Event Details / Tell us about your event"
    recommendation: "Before the consumer submits event details (which is personal data + a commercial intent), surface a brief 'How catering quotes work' inline notice with a link to the catering T&C and DPDP notice for the personal data being collected (date, address, guest count, dietary requirements — some of which may be sensitive)."
    citation: "DPDP Act 2023 §5 (notice at point of collection); Consumer Protection (E-Commerce) Rules 2020 Rule 5(3); best-practice"
    depends_on: "needs lawyer review"

  # ─────────────────────────────────────────────────────────────────────
  # P0 — Brand identity confusion ("Fe3dr" vs "HomeChef")
  # ─────────────────────────────────────────────────────────────────────
  - finding_id: LEG-MKT-021
    surface_id: web-mkt-layout-footer
    lens: legal
    severity: P0
    issue: "Footer copyright reads '© <year> Fe3dr' while the project name in CLAUDE.md is 'Home Chef' / 'Home Chef Mobile Apps'. Brand identity inconsistency across marketing surfaces creates trademark, attribution, and contract-formation ambiguity"
    evidence_excerpt: "© {new Date().getFullYear()} Fe3dr. All rights reserved."
    recommendation: "Pick ONE consumer-facing brand name and use it consistently across (a) all marketing pages, (b) T&C and Privacy Policy (the 'data fiduciary' identification under DPDP §5), (c) tax invoices, (d) email-from headers, (e) the legal entity name. Current state — 'Fe3dr' in HomePage hero/why-choose/footer, 'HomeChef' in branding contexts — risks customers being unable to identify who they are contracting with. This is a foundational legal issue (you cannot enforce a contract or comply with DPDP fiduciary-identification requirements if the consumer doesn't know who you are)."
    citation: "DPDP Act 2023 §5 (data fiduciary identification); Trade Marks Act 1999 (brand consistency for trademark enforcement); Consumer Protection (E-Commerce) Rules 2020 Rule 5(1) (seller identification); Companies Act 2013 §12 (use of registered name)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-MKT-022
    surface_id: web-mkt-login-testimonial
    lens: legal
    severity: P1
    issue: "Testimonial references 'Fe3dr' brand name in user-attributed quotation — if 'Fe3dr' is not the legal brand customers register under, the testimonial cannot be authentic"
    evidence_excerpt: "\"Fe3dr has changed how I eat...\""
    recommendation: "Compound issue with LEG-MKT-012 (testimonial authenticity) and LEG-MKT-021 (brand confusion). Resolve brand identity first, then re-verify testimonial copy."
    citation: "ASCI Code Chapter I.4 (testimonials must be genuine); Trade Marks Act 1999"
    depends_on: "needs lawyer review"

  - finding_id: LEG-MKT-023
    surface_id: web-mkt-landing-why-choose
    lens: legal
    severity: P1
    issue: "'The Fe3dr Difference' section header — same brand-identity issue surfaces in marketing copy at the value-proposition headline"
    evidence_excerpt: "The Fe3dr Difference"
    recommendation: "Resolve via brand consolidation (LEG-MKT-021). Until resolved, every 'Fe3dr' string on a marketing surface compounds the trademark and contract-identification exposure."
    citation: "Trade Marks Act 1999; Consumer Protection (E-Commerce) Rules 2020 Rule 5(1)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-MKT-024
    surface_id: web-mkt-register-benefits
    lens: legal
    severity: P1
    issue: "Registration page invites user to 'Join Fe3dr Today' — at the contract-formation moment the user is told they are joining 'Fe3dr', but the underlying T&C and privacy policy must match this name exactly"
    evidence_excerpt: "Join Fe3dr Today"
    recommendation: "Verify (a) T&C identifies 'Fe3dr' as the contracting party (or the legal entity that owns the Fe3dr brand), (b) Privacy Policy identifies 'Fe3dr' as the data fiduciary, (c) post-registration emails come from a Fe3dr-branded domain consistent with this name. If any of (a)/(b)/(c) uses a different name (e.g., 'HomeChef Pvt Ltd' or another entity), the user has a colourable misrepresentation claim."
    citation: "DPDP Act 2023 §5 (data fiduciary identification at notice); Consumer Protection (E-Commerce) Rules 2020 Rule 5(1) (seller identification); Indian Contract Act 1872 §17"
    depends_on: "needs lawyer review"

  - finding_id: LEG-MKT-025
    surface_id: web-mkt-brand-tagline
    lens: legal
    severity: P3
    issue: "Logo tagline 'Homemade Food Delivered' is non-quantified and low-risk on its own, but the shared Logo component is the brand anchor — its tagline must be consistent with whichever brand identity is finalised"
    evidence_excerpt: "Homemade Food Delivered"
    recommendation: "Low-priority. Once brand identity is consolidated (LEG-MKT-021), confirm the Logo tagline does not silently introduce a third brand variant or food-quality claim."
    citation: "best-practice (brand-claim consistency)"
    depends_on: "needs lawyer review"

  # ─────────────────────────────────────────────────────────────────────
  # P0 — Footer: legal-link discoverability + DPDP grievance officer
  # ─────────────────────────────────────────────────────────────────────
  - finding_id: LEG-MKT-026
    surface_id: web-mkt-layout-footer
    lens: legal
    severity: P0
    issue: "Footer links 'Privacy Policy' and 'Terms of Service' are present, but no link to a DPDP grievance-officer contact, no link to FSSAI licence display, no link to a refund/return policy, no 'Cancellation Policy', and no 'Contact Us' for consumer disputes"
    evidence_excerpt: "Privacy Policy / Terms of Service / Help Center / About Us"
    recommendation: "CP E-Commerce Rules 2020 Rule 5 requires e-commerce entities to display: (a) name, principal address, contact details; (b) grievance officer name and contact; (c) ticket/complaint redressal mechanism. DPDP Act §8(9) separately requires a grievance officer for data-protection complaints. Add footer links: 'Grievance Officer', 'Refund & Cancellation Policy', 'Contact Us', 'FSSAI Licence' (platform's own + a link to chef-level disclosure). 'Help Center' is not a legal-redressal substitute."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 4(3) and Rule 5 (mandatory disclosures); DPDP Act 2023 §8(9) (grievance officer for personal data); FSS (Licensing & Registration) Regulations 2011 §2.1.13 (FSSAI licence display)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-MKT-027
    surface_id: web-mkt-layout-footer
    lens: legal
    severity: P1
    issue: "Footer brand description 'Connecting you with home chefs for authentic, homemade food delivered to your doorstep' frames the platform as a marketplace ('connecting you with') but does not disclose whether the platform is a 'marketplace e-commerce entity' or 'inventory e-commerce entity' under CP E-Comm Rules — affects liability allocation"
    evidence_excerpt: "Connecting you with home chefs for authentic, homemade food delivered to your doorstep."
    recommendation: "Per Rule 5(1)(a)/(b), an e-commerce entity must declare whether it is a marketplace (facilitator between chefs and customers) or inventory (selling its own food). This affects: liability for product quality (FSSAI), refund obligations, GST treatment. The current footer copy implies marketplace but doesn't formally declare it; T&C must align. Recommend the About / T&C explicitly state the classification, and the footer or About copy reference it ('Home Chef is a marketplace platform connecting customers with home chefs')."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 5(1)(a) and Rule 6 (marketplace classification, liability allocation)"
    depends_on: "needs lawyer review"

  # ─────────────────────────────────────────────────────────────────────
  # P1 — Marketing page CTAs lacking conditions/qualifiers
  # ─────────────────────────────────────────────────────────────────────
  - finding_id: LEG-MKT-028
    surface_id: web-mkt-landing-search-cta
    lens: legal
    severity: P2
    issue: "Primary CTA 'Find Food' on the home page leads to /chefs without any geographic or availability filter — visitors outside coverage areas may be misled into thinking the service is available everywhere"
    evidence_excerpt: "Find Food"
    recommendation: "(1) Either gate the search by serviceable pin codes (so non-coverage areas get a clear 'we don't operate here yet' message), or (2) qualify the home-page promise with a 'Available in <list of cities>' line near the hero. CP E-Comm Rules treat undisclosed geographic limitations as a form of misleading advertising."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 4(11); best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-MKT-029
    surface_id: web-mkt-landing-search-placeholders
    lens: legal
    severity: P3
    issue: "Search placeholder 'Enter your delivery address...' collects an address (personal data) in an unauthenticated context — DPDP requires notice even for pre-login data collection"
    evidence_excerpt: "Enter your delivery address... / Search dishes or chefs..."
    recommendation: "If the address entered in the search bar is transmitted to backend or persisted (even in a cookie), DPDP §5 requires a notice at point of collection — even for non-account users. Recommend a small 'How we use this' tooltip or a one-line privacy note under the search bar referencing the privacy policy. If the address is purely client-side and ephemeral, document that in a code-comment to defend the position."
    citation: "DPDP Act 2023 §5 (notice at point of collection); best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-MKT-030
    surface_id: web-mkt-hero-search
    lens: legal
    severity: P3
    issue: "Shared HeroSection search ('Enter your delivery address / Find Chefs') replicates the same pre-login data-collection issue across every page using the shared component"
    evidence_excerpt: "Enter your delivery address / Find Chefs"
    recommendation: "Fix at the component level — a single privacy/notice treatment in the shared HeroSection covers all consuming pages."
    citation: "DPDP Act 2023 §5; best-practice (fix shared components once)"
    depends_on: "needs lawyer review"

  # ─────────────────────────────────────────────────────────────────────
  # P2 — Marketing copy referencing operational claims without scope
  # ─────────────────────────────────────────────────────────────────────
  - finding_id: LEG-MKT-031
    surface_id: web-mkt-landing-cuisines
    lens: legal
    severity: P3
    issue: "Cuisine cards (South Indian, Italian, Japanese, North Indian, Mexican, Thai) imply all six cuisines are available in every market — if certain cuisines are not present in some pin codes, this misleads"
    evidence_excerpt: "Cuisines / Explore Flavors / Discover authentic dishes from around the world"
    recommendation: "Either (a) ensure each cuisine card leads to non-empty results in every serviceable area, or (b) gate visibility by location, or (c) softer phrasing 'Cuisines our community cooks' (no implicit availability promise). Also 'authentic dishes from around the world' is a regulated descriptor — see LEG-MKT-008."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 4(11); FSS (Advertising and Claims) Regulations 2018 §4"
    depends_on: "needs lawyer review"

  - finding_id: LEG-MKT-032
    surface_id: web-mkt-landing-why-choose
    lens: legal
    severity: P2
    issue: "'Secure Payments / Safe and secure payment processing for every order' — 'safe', 'secure', 'every' are absolute security claims with regulatory implications under RBI Payment Aggregator rules"
    evidence_excerpt: "Secure Payments / Safe and secure payment processing for every order"
    recommendation: "Absolute security claims ('safe', 'secure', 'every') are flagged risk under ASCI guidelines (no system is absolutely secure). Soften to 'Payments processed by <Razorpay/Stripe>, PCI-DSS compliant' — this is a substantiable factual statement instead of an absolute claim. Also creates an opportunity to disclose the actual payment aggregator, which CP E-Comm Rules 5(3) requires."
    citation: "ASCI Guidelines on Misleading Advertising (absolute claims); RBI Guidelines on Payment Aggregators (disclosure); Consumer Protection (E-Commerce) Rules 2020 Rule 5(3)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-MKT-033
    surface_id: web-mkt-landing-how-it-works
    lens: legal
    severity: P3
    issue: "How-It-Works step 'Order — Select your favorite dishes and place your order securely' duplicates the 'securely' absolute claim from the Why-Choose-Us block"
    evidence_excerpt: "Order / Select your favorite dishes and place your order securely"
    recommendation: "Same fix as LEG-MKT-032 — replace 'securely' with a substantiable factual reference, or drop the adverb (the action 'place your order' is sufficient)."
    citation: "ASCI Guidelines on Misleading Advertising"
    depends_on: "needs lawyer review"

  # ─────────────────────────────────────────────────────────────────────
  # P1 — DPDP notice exposure on marketing-context CTAs
  # ─────────────────────────────────────────────────────────────────────
  - finding_id: LEG-MKT-034
    surface_id: web-mkt-landing-catering-cta
    lens: legal
    severity: P1
    issue: "Catering CTA 'Request Catering Quote' leads to a form that will collect personal data (name, contact, event details) — DPDP requires the notice to be served at or before collection, not buried in a privacy policy link"
    evidence_excerpt: "Request Catering Quote"
    recommendation: "Per DPDP §5(1), a notice in 'clear and plain language' must accompany the request for consent at point of collection. On the catering form page (the link target), surface an inline DPDP-compliant notice: purpose, data fiduciary identity, grievance officer, rights (access/correction/erasure/grievance). Marketing CTA does not need the notice, but the target page must."
    citation: "DPDP Act 2023 §5 (notice); DPDP Act 2023 §6 (consent must be free, specific, informed)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-MKT-035
    surface_id: web-mkt-landing-become-chef
    lens: legal
    severity: P1
    issue: "'Become a Chef' CTA leads chef-onboarding flow that will collect KYC-class personal data (PAN, bank details, FSSAI registration, identity proofs) — DPDP §5 + §9 (sensitive data) require enhanced notice"
    evidence_excerpt: "Become a Chef"
    recommendation: "(1) Chef onboarding collects significantly more sensitive data than customer signup — DPDP requires a proportionately detailed notice including purposes, recipients (e.g., bank verification API, FSSAI authority, KYC provider), retention period, grievance officer. (2) RBI Payment Aggregator rules require chef-side KYC; this must be disclosed up-front, not at the end of the funnel."
    citation: "DPDP Act 2023 §5 and §6 (consent); RBI Guidelines on Payment Aggregators §6 (merchant KYC)"
    depends_on: "needs lawyer review"
```

## Business Analyst findings

```yaml
findings:

  # ── P0: Fake / unverified trust signals ───────────────────────────────────

  - finding_id: BA-001
    surface_id: web-mkt-landing-hero-badge
    lens: business-analyst
    severity: P0
    issue: "Hardcoded '500+ Home Chefs Near You' badge is not sourced from live data"
    evidence_excerpt: "<Badge variant=\"premium\" size=\"lg\">500+ Home Chefs Near You</Badge>"
    recommendation: >
      Replace with a live API-driven count from the chefs endpoint (e.g., `{chefCount}+ Home Chefs Near You`).
      If live count is unavailable at launch, remove the badge entirely rather than ship a fabricated number.
      Expected impact: removes a trust-destroying signal that sophisticated users will immediately cross-check
      by searching on the platform itself.
    metric_hypothesis: "trust score; any visitor who searches and sees fewer than 500 chefs will distrust every other claim on the page and churn"
    depends_on: null

  - finding_id: BA-002
    surface_id: web-mkt-landing-trust-badges
    lens: business-analyst
    severity: P0
    issue: "Three hardcoded trust-badge stats ('500+ Home Chefs', '4.8 Average Rating', '30-45 min Delivery') are static strings with no live-data binding"
    evidence_excerpt: "{ icon: ChefHat, label: '500+ Home Chefs' }, { icon: Star, label: '4.8 Average Rating' }, { icon: Clock, label: '30-45 min Delivery' }"
    recommendation: >
      Bind all three values to live API responses: chef count from `/chefs?count=true`, average rating
      from `/ratings/aggregate`, and actual median delivery time from `/orders/stats`.
      Until live data is available, replace '4.8 Average Rating' with 'Highly Rated Chefs' (no fabricated
      numeral), '500+' with 'Growing community of home chefs', and '30-45 min Delivery' with
      'Fast delivery' (no fabricated SLA).
    metric_hypothesis: "trust score; signup conversion — a fabricated 4.8 rating shown before any orders exist destroys credibility upon scrutiny"
    depends_on: null

  - finding_id: BA-003
    surface_id: web-mkt-hero-variants
    lens: business-analyst
    severity: P0
    issue: "Shared HeroSection Stats component repeats the same hardcoded '500+' chef count and '30min Avg Delivery' on every page variant that renders it"
    evidence_excerpt: "{ icon: ChefHat, value: '500+', label: 'Home Chefs' }, { icon: Clock, value: '30min', label: 'Avg Delivery' }"
    recommendation: >
      Centralise stat values in a single data-fetch hook (e.g., `usePlatformStats()`) shared by HeroSection
      and HomePage trust badges.  Both surfaces will then update from one source when live data is wired.
      This also eliminates the dual-maintenance risk where one gets updated and the other doesn't.
    metric_hypothesis: "trust score — duplicate hardcoded numbers in multiple components doubles exposure to trust erosion"
    depends_on: BA-002

  - finding_id: BA-004
    surface_id: web-mkt-register-benefits
    lens: business-analyst
    severity: P0
    issue: "Customer signup page lists 'Access to 500+ home chefs' as a benefit — same unverified hardcoded number shown on the conversion-critical registration page"
    evidence_excerpt: "const BENEFITS = ['Access to 500+ home chefs', 'Authentic homemade food', 'Fast & reliable delivery', 'Support local home chefs']"
    recommendation: >
      Replace 'Access to 500+ home chefs' with a claim that doesn't depend on a specific count,
      e.g., 'Browse home chefs near you' or 'Home-cooked meals from your neighbourhood'.
      Additionally, 'Fast & reliable delivery' uses a banned phrase (per style guide: no vague promise
      language); replace with 'Delivery to your doorstep'.
    metric_hypothesis: "signup conversion — a false metric on the registration page directly erodes trust at the moment of commitment"
    depends_on: null

  - finding_id: BA-005
    surface_id: web-mkt-login-testimonial
    lens: business-analyst
    severity: P0
    issue: "Fake placeholder testimonial from 'Sarah M., Happy Customer' ships in production on the login page"
    evidence_excerpt: "\"Fe3dr has changed how I eat. Finally, real homemade food that reminds me of my mom's cooking!\" — Sarah M., Happy Customer"
    recommendation: >
      Either (a) replace with a real, attributed testimonial from an actual beta customer with full name,
      city, and date, or (b) remove the testimonial entirely and replace with a product benefit statement.
      The role 'Happy Customer' is a placeholder-level designation that signals the quote is fabricated.
      Fake testimonials are specifically prohibited under ASCI (India) guidelines and erode trust on the
      highest-volume entry point (login is visited every session).
    metric_hypothesis: "trust score; login→signup conversion — users who notice the fabricated testimonial will share screenshots externally, creating reputational damage"
    depends_on: null

  - finding_id: BA-006
    surface_id: web-mkt-landing-why-choose
    lens: business-analyst
    severity: P0
    issue: "Subtitle 'Join thousands of happy customers enjoying homemade food' is an unverified claim with no live data source"
    evidence_excerpt: "Join thousands of happy customers enjoying homemade food"
    recommendation: >
      Remove the quantified claim entirely. Replace with a specific, verifiable benefit statement:
      'Homemade food from chefs who cook the way they cook at home.'
      'Thousands' is a magnitude claim that cannot be substantiated on a new platform and will fail
      a consumer-protection challenge.
    metric_hypothesis: "trust score — unverifiable superlative claims reduce perceived authenticity for comparison-shopping users"
    depends_on: null

  - finding_id: BA-007
    surface_id: vp-mkt-register-benefits-list
    lens: business-analyst
    severity: P0
    issue: "'Join thousands of home chefs earning with Fe3dr' on the vendor registration page is an unverified magnitude claim"
    evidence_excerpt: "Join thousands of home chefs earning with Fe3dr. Your kitchen, your recipes, your rules."
    recommendation: >
      Replace with a factual, non-quantified hook: 'Start selling from your kitchen today. Your recipes,
      your prices, your schedule.' Remove the 'thousands' claim until live chef count data backs it.
    metric_hypothesis: "chef apply rate — inflated claims raise expectations that the platform is already at scale; new chefs who join and see a sparse marketplace will churn faster"
    depends_on: null

  # ── P0: "Coming soon" feature advertised to users collecting sensitive data ─

  - finding_id: BA-008
    surface_id: vp-mkt-register-benefits-list
    lens: business-analyst
    severity: P0
    issue: "'Weekly payouts — Get paid directly to your bank account' is advertised as a live benefit while the payout integration is explicitly marked 'coming soon' in KitchenSetupPage"
    evidence_excerpt: |
      RegisterPage.tsx: { title: 'Weekly payouts', desc: 'Get paid directly to your bank account' }
      KitchenSetupPage.tsx: "Payout integration coming soon. These details are saved locally for now."
    recommendation: >
      The 'Weekly payouts' benefit claim on the registration page must be removed or replaced with a
      truthful status: e.g., 'Payout integration in progress — your bank details will be used when
      payouts go live.' More critically: if the system collects bank account numbers but the payout
      integration is not functional, this is a deceptive data collection practice and a regulatory risk.
      Immediately remove the bank-details form from KitchenSetupPage or add a prominent disclosure:
      'Payouts are not yet active. We are collecting your bank details for when the feature launches.
      You will be notified before your first payout is processed.'
    metric_hypothesis: "chef apply rate; churn; regulatory risk — chefs who submit bank details expecting weekly payouts and receive nothing will churn and file complaints"
    depends_on: null

  # ── P0: Verified Chefs claim contradicts optional FSSAI onboarding ─────────

  - finding_id: BA-009
    surface_id: web-mkt-landing-why-choose
    lens: business-analyst
    severity: P0
    issue: "'Verified Chefs — All our home chefs are verified for food safety and quality' contradicts the vendor onboarding flow where FSSAI is marked as Optional"
    evidence_excerpt: |
      HomePage.tsx: "All our home chefs are verified for food safety and quality"
      StepDocuments.tsx: "FSSAI license is optional — many home chefs start without one and add it later."
    recommendation: >
      Either (a) make FSSAI mandatory in onboarding and then the verified claim becomes defensible,
      or (b) soften the customer-facing claim to 'ID-verified home chefs' (reflecting what is actually
      checked: government ID only). Do not use 'food safety verified' when FSSAI is optional.
      This is simultaneously a trust, legal, and consumer-protection issue — a customer who gets ill
      and checks this claim will have grounds for a complaint.
    metric_hypothesis: "trust score; legal liability — false food-safety verification claim is the highest-magnitude trust risk on the platform"
    depends_on: null

  # ── P0: Dead CTAs — broken conversion paths ───────────────────────────────

  - finding_id: BA-010
    surface_id: web-mkt-landing-become-chef
    lens: business-analyst
    severity: P0
    issue: "Both CTAs in the 'Become a Chef' section link to unregistered routes (/become-chef and /chef-resources) that resolve to the homepage (404 → redirect to /)"
    evidence_excerpt: |
      <Link to="/become-chef">Become a Chef</Link>
      <Link to="/chef-resources">Learn More</Link>
      (Neither route exists in apps/web/src/app/routes/index.tsx)
    recommendation: >
      Register the /become-chef route immediately and redirect it to the vendor portal registration page
      (`https://vendor.fe3dr.com/register` or the equivalent). Remove the 'Learn More' button until
      a /chef-resources page exists; two dead CTAs is worse than one working CTA.
      Until fixed: every chef prospect who clicks 'Become a Chef' lands on the homepage with no
      indication they clicked anything. This is a complete conversion blackhole.
    metric_hypothesis: "chef apply rate — 100% conversion loss on the primary chef acquisition CTA on the homepage"
    depends_on: null

  - finding_id: BA-011
    surface_id: web-mkt-hero-how-it-works-cta
    lens: business-analyst
    severity: P0
    issue: "Secondary hero CTA 'How It Works' links to /how-it-works, a route not registered in the router — visitors are silently redirected to home"
    evidence_excerpt: "<Link to=\"/how-it-works\">How It Works</Link> (not in routes/index.tsx)"
    recommendation: >
      Either register a /how-it-works route pointing to the 'How It Works' section on the homepage
      (via anchor: /#how-it-works) or change the Link to an anchor scroll: `<a href=\"#how-it-works\">`.
      The 'How It Works' button is the secondary consideration path for hesitant visitors — losing them
      here costs a disproportionate share of high-intent signups.
    metric_hypothesis: "signup conversion — secondary CTA clickthrough from hesitant visitors drops to zero; these are typically higher-quality signups who read before committing"
    depends_on: null

  # ── P0: Admin analytics dashboard shipping placeholder charts to operators ─

  - finding_id: BA-012
    surface_id: null
    lens: business-analyst
    severity: P0
    issue: "Admin analytics page ships 'Chart coming soon' placeholder to operators — the feature is not in the marketing inventory but qualifies as a fake analytics surface"
    evidence_excerpt: "AnalyticsPage.tsx (admin-portal): <p className=\"mt-2 text-sm\">Chart coming soon</p>"
    recommendation: >
      Either (a) ship a real chart using the existing data endpoints before releasing this page to
      operators, or (b) hide the analytics nav item entirely until the feature is complete.
      Operators who see 'Chart coming soon' for more than one sprint lose confidence in the platform's
      readiness and escalate with support tickets.
    metric_hypothesis: "operator trust; operator activation — empty dashboard surfaces are the single fastest way to lose operator confidence post-launch"
    depends_on: null

  # ── P1: Conversion-critical clarity issues ────────────────────────────────

  - finding_id: BA-013
    surface_id: web-mkt-landing-hero-title
    lens: business-analyst
    severity: P1
    issue: "Hero search button says 'Find Food' but the search actually routes to /chefs — misalignment between CTA and destination erodes trust"
    evidence_excerpt: |
      Button label: "Find Food"
      Link destination: /chefs (or /chefs?search=...)
      The user is looking for food; they land on a chef browse page
    recommendation: >
      Either change the button label to 'Find Chefs' (matching the destination) or change the
      destination to a dish/menu search results page if one exists. The search inputs also ask for
      'dishes or chefs' — the CTA verb must match the most prominent input's expected result.
      Suggested: 'Search' (neutral, always accurate) or 'Find chefs near you'.
    metric_hypothesis: "hero search conversion — label/destination mismatch increases bounce rate after click; users who expected a food search result and land on a chef list feel misled"
    depends_on: null

  - finding_id: BA-014
    surface_id: web-mkt-landing-become-chef
    lens: business-analyst
    severity: P1
    issue: "'Love Cooking? Share Your Talent' headline in the chef recruitment CTA is weak on differentiation — gives no reason to choose Fe3dr over Swiggy Minis or Zomato Home"
    evidence_excerpt: |
      "Love Cooking? Share Your Talent"
      "Turn your passion into income. Join our community of home chefs and start earning by sharing your delicious homemade food."
    recommendation: >
      Lead with the structural advantage: no ghost kitchen required, no delivery fleet to manage, cook
      your own hours. Suggested headline: 'Cook from home. Set your own menu and hours.'
      Subtext: 'No restaurant overheads. Customers come to you.' This differentiates from gig platforms
      where drivers/cooks are managed assets.
    metric_hypothesis: "chef apply rate — generic passion-economy copy does not differentiate; chefs who compare Fe3dr to Swiggy Minis will not have a reason to prefer it"
    depends_on: BA-010

  - finding_id: BA-015
    surface_id: vp-mkt-register-benefits-list
    lens: business-analyst
    severity: P1
    issue: "'Zero commission first month' benefit claim has no terms attached — creates misleading expectations about ongoing commission rates"
    evidence_excerpt: "{ title: 'Zero commission first month', desc: 'Get started completely risk-free' }"
    recommendation: >
      Add the ongoing commission rate immediately adjacent: e.g., 'Zero commission first month,
      then X% per order.' Without the ongoing rate, chefs who read this feel surprised and misled
      when they see commission charges in month 2. The StepPolicies copy confirms a commission
      exists: 'Fe3dr charges a platform commission on each order' — but the rate is never disclosed
      on the registration marketing surface. Disclose it.
    metric_hypothesis: "chef D30 retention; chef apply rate — hidden ongoing pricing discovered post-signup is the leading cause of early churn in marketplace supplier acquisition"
    depends_on: null

  - finding_id: BA-016
    surface_id: web-mkt-catering-request-heading
    lens: business-analyst
    severity: P1
    issue: "Catering request wizard has no CTA that sets expectations for what happens after submission — 'Request Catering Quotes' is the entry button but the wizard has no completion confirmation copy"
    evidence_excerpt: |
      Heading: "Request Catering Quotes"
      Sub: "Tell us about your event and receive quotes from our home chefs"
      Progress steps: Event Details / Preferences / Location (no step 4 = 'Review' or 'Submitted')
    recommendation: >
      Add a Step 4 confirmation screen: 'Request submitted. Chefs will respond within 24 hours.
      You'll get a notification for each quote.' Set expectations explicitly on when and how quotes
      arrive. The current flow ends at Step 3 (Location) with no success state copy visible in
      the inventory — users will not know if the submission worked.
    metric_hypothesis: "catering quote completion rate — missing post-submission copy causes re-submission attempts and support tickets, reducing effective conversion"
    depends_on: null

  # ── P2: Missed opportunity on medium-traffic surfaces ─────────────────────

  - finding_id: BA-017
    surface_id: web-mkt-browse-heading
    lens: business-analyst
    severity: P2
    issue: "Browse Chefs page heading 'Explore Home Chefs' with sub 'Discover talented home chefs serving authentic homemade food' is generic and adds no browse-aid value"
    evidence_excerpt: "Explore Home Chefs / Discover talented home chefs serving authentic homemade food"
    recommendation: >
      Replace the static sub with a live contextual indicator that helps the user orient:
      '{chefCount} home chefs in {city}' or 'Showing chefs available for delivery today.'
      At minimum, 'talented' is a brand-drift adjective that says nothing; drop it.
      Suggested: 'Home Chefs / Find a home chef near you.'
    metric_hypothesis: "browse→order conversion — context-setting copy (city, count, availability) reduces time-to-first-click on a chef profile by giving users orientation"
    depends_on: null

  - finding_id: BA-018
    surface_id: web-mkt-landing-how-it-works
    lens: business-analyst
    severity: P2
    issue: "'How It Works' step 2 (Order) says only 'Select your favorite dishes and place your order securely' — 'securely' is a hollow reassurance without specifying what payment methods are accepted"
    evidence_excerpt: "description: 'Select your favorite dishes and place your order securely'"
    recommendation: >
      Replace 'securely' with a concrete payment reassurance: 'Pay by UPI, card, or cash on delivery.'
      This converts a vague trust word into an actionable buying signal. Users who don't know if UPI
      is accepted will abandon rather than click through to discover it.
    metric_hypothesis: "how-it-works→signup conversion — specifying payment methods reduces pre-signup payment anxiety, a leading cause of drop-off in Indian food-ordering contexts"
    depends_on: null

  - finding_id: BA-019
    surface_id: web-mkt-landing-why-choose
    lens: business-analyst
    severity: P2
    issue: "'Made with Love' card title uses a banned brand-drift phrase per STYLE-GUIDE.md"
    evidence_excerpt: "{ title: 'Made with Love', description: 'Every meal is prepared fresh with authentic family recipes' }"
    recommendation: >
      Replace 'Made with Love' with a specific and differentiated claim: 'Family recipes, fresh daily'
      or 'Cooked to order, not reheated.' The card description is more compelling than the title —
      promote the description to the title. 'Made with Love' is listed as a banned artisanal-drift
      phrase in the style guide.
    metric_hypothesis: "brand trust; differentiation — generic sentimental language ('love') makes the platform indistinguishable from every restaurant marketing deck"
    depends_on: null

  - finding_id: BA-020
    surface_id: web-mkt-landing-catering-cta
    lens: business-analyst
    severity: P2
    issue: "Catering CTA 'Planning an Event?' is a yes/no question that gates a ₹-significant feature — no minimum guest count, no price range, no social proof is shown"
    evidence_excerpt: |
      "Planning an Event?"
      "Get catering quotes from multiple home chefs. Perfect for parties, corporate events, and special occasions."
      Button: "Request Catering Quote"
    recommendation: >
      Add one trust/scale signal to the catering CTA to set expectations: e.g., 'Quotes for 10–500
      guests. Chefs respond within 24 hours.' This reduces abandonment for users who don't know if
      the feature applies to their event size, and makes the CTA actionable rather than aspirational.
    metric_hypothesis: "catering quote request rate — adding minimum guest scope reduces hesitation from users unsure if the feature fits their event scale"
    depends_on: null

  # ── P3: Minor friction on lower-traffic surfaces ──────────────────────────

  - finding_id: BA-021
    surface_id: web-mkt-layout-footer
    lens: business-analyst
    severity: P3
    issue: "Footer 'For Chefs' column links to /become-chef and /chef-resources — both dead routes (confirmed by router audit); this is a secondary surface but still creates 404-experience"
    evidence_excerpt: |
      MainLayout.tsx footer:
      <Link to=\"/become-chef\">Become a Chef</Link>
      <Link to=\"/chef-resources\">Resources</Link>
    recommendation: >
      Fix after BA-010 is resolved. Until /chef-resources page is built, remove the 'Resources' footer
      link. A broken footer link is a lower-priority trust issue than the hero CTA, but it confirms
      the platform is unfinished to any user who tests links.
    metric_hypothesis: "brand trust — broken footer links signal an incomplete product to users who are doing due diligence before signing up"
    depends_on: BA-010

  - finding_id: BA-022
    surface_id: web-mkt-brand-tagline
    lens: business-analyst
    severity: P3
    issue: "Logo tagline 'Homemade Food Delivered' is shown in the footer but is too generic to differentiate — identical to dozens of food delivery apps"
    evidence_excerpt: "Logo.tsx:91 — optional tagline: \"Homemade Food Delivered\""
    recommendation: >
      If the tagline is shown, sharpen it to something that states the structural differentiator:
      'Home chefs, not restaurants.' or 'From their kitchen to your door.'
      Generic taglines cost nothing to sharpen but improve brand recall at the bottom of the funnel.
    metric_hypothesis: "brand recall — footer tagline is the last impression for users who scrolled without converting; a specific tagline can tip return visits"
    depends_on: null
```

## Brand Voice findings

```yaml
# Brand-Voice lens findings — MARKETING category
# Auditor: brand-voice lens agent
# Scope: 22 marketing-category surfaces across web (landing, hero, auth marketing rails, footer, browse, catering, brand)
#        + vendor-portal register marketing rail
# Date: 2026-05-13
#
# Marketing is the brand's ENTRY surface. Every drift here defines first impressions, so
# severity skews higher than for deep operational screens. AI-slop placeholder metrics
# (the user's explicit concern) are P0 across the board because they erode platform trust.

findings:
  # ─────────────────────────────────────────────────────────────────────────
  # P0 — Unverified social-proof / AI-slop hardcoded metrics
  # (User's stated #1 concern. Every instance is a brand-trust violation
  #  AND a Rule 5 "Restraint over urgency" / Rule 1 "Confident, not loud" violation.)
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BV-001
    surface_id: web-mkt-landing-hero-badge
    lens: brand-voice
    severity: P0
    issue: "Hardcoded '500+ Home Chefs Near You' on landing hero badge — unverified platform claim, almost certainly fake, and the same number is repeated three times across the homepage (badge, trust row, shared HeroSection Stats) signaling AI-slop placeholder content rather than real data."
    evidence_excerpt: "<Badge variant=\"premium\" size=\"lg\" className=\"mb-6\"><Sparkles … /> 500+ Home Chefs Near You </Badge>"
    related_surfaces:
      - web-mkt-landing-trust-badges
      - web-mkt-register-benefits
      - web-mkt-hero-variants
    recommendation: "Replace with a real metric fed from the API (e.g., useQuery returning live chef count for the user's pincode → '127 chefs near you', or scoped 'Chefs in Bengaluru'). If real count is below the brag-threshold, drop the badge entirely — restraint > inflated proof. Never hardcode '500+'."
    depends_on: null

  - finding_id: BV-002
    surface_id: web-mkt-landing-trust-badges
    lens: brand-voice
    severity: P0
    issue: "Three hardcoded 'trust signals' below the hero — '500+ Home Chefs', '4.8 Average Rating', '30-45 min Delivery' — all static strings, none read from real platform data. Classic AI-slop trust-badge pattern, identical to the look critiqued in `.impeccable.md`. The 4.8 rating in particular is a statistical claim with no source."
    evidence_excerpt: "{ icon: ChefHat, label: '500+ Home Chefs' }, { icon: Star, label: '4.8 Average Rating' }, { icon: Clock, label: '30-45 min Delivery' }"
    related_surfaces:
      - web-mkt-landing-hero-badge
      - web-mkt-hero-variants
      - web-mkt-landing-why-choose
    recommendation: "Either bind these to live aggregates (chef count from `/chefs/count?near=…`, rolling 30-day avg rating from `/reviews/aggregate`, p50 delivery time from `/orders/eta-summary`) or delete the row. Hardcoded marketing stats are worse than no stats — they read as untrustworthy on inspection and translate badly to other locales."
    depends_on: null

  - finding_id: BV-003
    surface_id: web-mkt-hero-variants
    lens: brand-voice
    severity: P0
    issue: "Shared `HeroSection` `Stats` component hardcodes the same fabricated metrics as the homepage hero ('500+', '4.8', '30min'), meaning any page that mounts this hero (chefs, catering, feed variants) repeats the unverified claim. Cross-surface amplification of a single AI-slop placeholder."
    evidence_excerpt: "const stats = [ { icon: ChefHat, value: '500+', label: 'Home Chefs' }, { icon: Star, value: '4.8', label: 'Avg Rating' }, { icon: Clock, value: '30min', label: 'Avg Delivery' } ];"
    related_surfaces:
      - web-mkt-landing-hero-badge
      - web-mkt-landing-trust-badges
    recommendation: "Lift the Stats component out of the hero entirely, OR accept a `stats` prop from the parent route loader that fetches real numbers. Default to no-stats when the loader returns null. Same rule as BV-001/002 — no hardcoded numerals in marketing."
    depends_on: BV-001

  - finding_id: BV-004
    surface_id: web-mkt-landing-why-choose
    lens: brand-voice
    severity: P0
    issue: "'Join thousands of happy customers enjoying homemade food' under 'The Fe3dr Difference' headline — textbook unverified social-proof platitude. 'Thousands' has no source, 'happy customers' is unverifiable, and the phrasing is exactly the generic-DTC pattern `.impeccable.md` lists as an anti-reference."
    evidence_excerpt: "<h2 …>The Fe3dr Difference</h2> <p …>Join thousands of happy customers enjoying homemade food</p>"
    related_surfaces:
      - vp-mkt-register-benefits-list
      - web-mkt-register-benefits
    recommendation: "Replace the subhead with substance, not size: 'Verified home kitchens. Real recipes. Delivered today.' or similar. If 'thousands' is genuinely true and disclosable, prove it with a specific number sourced from the analytics service; otherwise remove."
    depends_on: null

  - finding_id: BV-005
    surface_id: web-mkt-register-benefits
    lens: brand-voice
    severity: P0
    issue: "RegisterPage left-rail hardcodes 'Access to 500+ home chefs' in the BENEFITS array — same fabricated count, now on the signup conversion surface. A customer who counts <500 chefs in their pincode will catch the lie."
    evidence_excerpt: "const BENEFITS = [ 'Access to 500+ home chefs', 'Authentic homemade food', 'Fast & reliable delivery', 'Support local home chefs' ];"
    related_surfaces:
      - web-mkt-landing-hero-badge
      - web-mkt-landing-trust-badges
      - web-mkt-hero-variants
    recommendation: "Swap the count for an evergreen claim: 'Home chefs in every metro we serve' or 'Verified neighborhood kitchens'. Generic-quantity claims belong on landing only when the data backs them, never on the signup surface where users are about to compare reality against the promise."
    depends_on: BV-001

  - finding_id: BV-006
    surface_id: web-mkt-register-benefits
    lens: brand-voice
    severity: P0
    issue: "Heading 'Get access to hundreds of home chefs serving authentic, homemade food in your area' — second unverified count on the same surface ('hundreds') and contradicts the badge's '500+' just one row above. Two different fake numbers on the same hero is a credibility break."
    evidence_excerpt: "<p>Get access to hundreds of home chefs serving authentic, homemade food in your area.</p>"
    related_surfaces:
      - web-mkt-register-benefits
      - web-mkt-landing-hero-badge
    recommendation: "Pick one truthful, scoped framing per surface. Suggested: 'Real home kitchens in your neighborhood, serving the food you actually want to eat.' No number, no internal contradiction."
    depends_on: BV-005

  - finding_id: BV-007
    surface_id: vp-mkt-register-benefits-list
    lens: brand-voice
    severity: P0
    issue: "Vendor portal register page promises 'Join thousands of home chefs earning with Fe3dr' AND 'Zero commission first month' as the headline benefits. Both are marketing claims without proof: 'thousands' is unverified social proof identical to BV-004; 'zero commission first month' is an explicit financial promise on an entry surface with no terms link and no expiry mechanism."
    evidence_excerpt: "Join thousands of home chefs earning with Fe3dr. Your kitchen, your recipes, your rules. … BENEFITS = [{ title: 'Zero commission first month', desc: 'Get started completely risk-free' }, …]"
    related_surfaces:
      - web-mkt-landing-why-choose
      - web-mkt-register-benefits
    recommendation: "Remove 'thousands'. The zero-commission claim must either (a) be backed by a live, version-controlled promotion record with explicit start/end dates and link to vendor T&Cs, or (b) be replaced with neutral language ('Transparent commission structure — see plans'). Otherwise this exposes the platform to consumer-protection complaints in addition to brand drift."
    depends_on: null

  - finding_id: BV-008
    surface_id: web-mkt-landing-featured-chefs-heading
    lens: brand-voice
    severity: P0
    issue: "'Our community's favorite home chefs' subhead under 'Featured Chefs / Top Rated' — 'favorite' is a verifiable claim with no underlying ranking source visible. Worse, the featured query uses `sort: 'rating', limit: 6` so a single 5-star review on a new chef may surface them above veterans — the copy promises social validation the algorithm doesn't deliver."
    evidence_excerpt: "<Badge variant=\"premium\">Top Rated</Badge> <h2>Featured Chefs</h2> <p>Our community's favorite home chefs</p>"
    related_surfaces: []
    recommendation: "Either tie 'favorite' to a real signal (e.g., 'Most ordered this week') sourced from order count, or use neutral language like 'Highly rated home chefs near you'. The badge 'Top Rated' should also reflect a documented threshold (≥4.5 with ≥20 reviews, etc.) and that threshold should be visible somewhere."
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # P1 — Brand identity drift on entry surfaces
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BV-009
    surface_id: web-mkt-login-testimonial
    lens: brand-voice
    severity: P1
    issue: "Login page right-rail features a fabricated testimonial: 'Fe3dr has changed how I eat. Finally, real homemade food that reminds me of my mom's cooking!' attributed to 'Sarah M., Happy Customer'. Initial + role pattern is the canonical AI-slop fake-testimonial signature called out in the lens brief. Worse, it sits on the LOGIN page — the user has already decided to use the product, so the testimonial does nothing but erode credibility on entry."
    evidence_excerpt: "\"Fe3dr has changed how I eat. Finally, real homemade food that reminds me of my mom's cooking!\" — Sarah M. / Happy Customer"
    related_surfaces:
      - web-mkt-register-benefits
    recommendation: "Either replace with a real, named, opt-in testimonial sourced from a reviewed customer (with their consent on file) and reviewed by legal, or remove the testimonial card entirely and let the food photography carry the rail. 'Happy Customer' role tag is a tell — real testimonials use the person's actual context (city, dish ordered, etc.)."
    depends_on: null

  - finding_id: BV-010
    surface_id: web-mkt-landing-hero-title
    lens: brand-voice
    severity: P1
    issue: "Two different homepage heroes coexist in the codebase. The live `HomePage.tsx` uses 'Homemade Food, Delivered Fresh' while the shared `HeroSection.tsx` variant `home` uses the same title with a different description ('Connect with talented home chefs in your area. Enjoy authentic, homemade meals prepared with love and delivered to your doorstep.'). Any route mounting the shared hero shows a different brand promise than the homepage. Cross-surface voice drift on the brand's primary headline."
    evidence_excerpt: "HomePage: 'Discover talented home chefs in your neighborhood and enjoy authentic, homemade meals delivered right to your doorstep.' vs HeroSection.home: 'Connect with talented home chefs in your area. Enjoy authentic, homemade meals prepared with love and delivered to your doorstep.'"
    related_surfaces:
      - web-mkt-landing-hero-subtitle
      - web-mkt-hero-variants
    recommendation: "One canonical homepage hero copy. Move the headline + description into a single exported constant (`HOMEPAGE_HERO`) consumed by both `HomePage` and the shared `HeroSection.home` variant. Also: 'prepared with love' is faux-artisanal (BV-013) — pick the cleaner of the two and drop that phrase."
    depends_on: null

  - finding_id: BV-011
    surface_id: web-mkt-hero-variants
    lens: brand-voice
    severity: P1
    issue: "Catering hero variant: 'Catering for Every Occasion / our home chefs bring authentic flavors to your special events.' — 'authentic flavors' + 'special events' is the generic SaaS marketing copy register the style guide bans ('amazing food', 'experience the difference' family). Reads like AI placeholder text written from a brief."
    evidence_excerpt: "catering: { title: 'Catering for', titleHighlight: 'Every Occasion', description: 'From intimate gatherings to large celebrations, our home chefs bring authentic flavors to your special events.', cta: 'Plan Your Event' }"
    related_surfaces:
      - web-mkt-catering-request-heading
      - web-mkt-landing-catering-cta
    recommendation: "Rewrite with specifics that only a home-kitchen marketplace could say: 'Catering from real home kitchens. Birthdays, office lunches, weddings — quotes within a day.' Cut 'authentic flavors' and 'special events'. Same treatment for feed variant ('Food / Inspiration / See what your favorite chefs are cooking today') which is also vague-SaaS."
    depends_on: null

  - finding_id: BV-012
    surface_id: web-mkt-layout-footer
    lens: brand-voice
    severity: P1
    issue: "Footer brand blurb 'Connecting you with home chefs for authentic, homemade food delivered to your doorstep.' is the THIRD variant of the brand description on a single page (hero + register + footer all phrase it differently). Same intent, three voices."
    evidence_excerpt: "Connecting you with home chefs for authentic, homemade food delivered to your doorstep."
    related_surfaces:
      - web-mkt-landing-hero-subtitle
      - web-mkt-brand-tagline
      - web-mkt-register-benefits
    recommendation: "Define a single canonical brand-statement string in `shared/constants/brand.ts` (e.g., `BRAND_STATEMENT_SHORT`, `BRAND_STATEMENT_LONG`). Import it in footer, hero, register, and meta tags. Recommend short form: 'Homemade food, delivered from your neighborhood.' (matches Logo tagline, drops 'authentic')."
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # P2 — Faux-artisanal language / banned vocabulary on marketing surfaces
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BV-013
    surface_id: web-mkt-landing-why-choose
    lens: brand-voice
    severity: P2
    issue: "'Made with Love / Every meal is prepared fresh with authentic family recipes' is exactly the faux-artisanal register STYLE-GUIDE.md section 3 bans ('handcrafted with love', 'lovingly prepared', 'homestyle goodness'). Pairs with a heart icon and paprika tint — visual maximalism reinforces the copy drift."
    evidence_excerpt: "{ icon: Heart, title: 'Made with Love', description: 'Every meal is prepared fresh with authentic family recipes', tone: 'paprika' }"
    related_surfaces:
      - web-mkt-hero-variants
      - vp-mkt-register-benefits-list
    recommendation: "Replace title with a functional benefit ('Cooked when you order', 'Family recipes', or 'No reheats'). Drop the heart icon. The other three cards (Verified Chefs / Secure Payments / Fast Delivery) are functional — this card is the only one with brand-personality drift in the row."
    depends_on: null

  - finding_id: BV-014
    surface_id: web-mkt-landing-hero-subtitle
    lens: brand-voice
    severity: P2
    issue: "Hero description: 'Discover talented home chefs in your neighborhood and enjoy authentic, homemade meals delivered right to your doorstep.' — 'talented', 'authentic', 'right to your doorstep' are all generic-marketing modifiers the style guide says compete with the food photo. Six modifiers in a 22-word sentence is the opposite of 'photo-forward, chrome-light' (Rule 3)."
    evidence_excerpt: "Discover talented home chefs in your neighborhood and enjoy authentic, homemade meals delivered right to your doorstep."
    related_surfaces:
      - web-mkt-hero-variants
      - web-mkt-browse-heading
      - web-mkt-layout-footer
    recommendation: "Cut by half. Suggested: 'Real home kitchens, delivered in your neighborhood.' (8 words). 'Authentic' is a flag — STYLE-GUIDE doesn't ban it explicitly but it appears in 4 different marketing surfaces here, signalling overuse. Consider adding it to the banned list."
    depends_on: null

  - finding_id: BV-015
    surface_id: web-mkt-browse-heading
    lens: brand-voice
    severity: P2
    issue: "'Discover talented home chefs serving authentic homemade food' — repeats the 'talented + authentic + homemade' modifier stack from the homepage hero (BV-014), suggesting these come from a single fill-in-the-blanks template rather than considered per-surface copy. Cross-surface voice monotony."
    evidence_excerpt: "<h1>Explore Home Chefs</h1> <p>Discover talented home chefs serving authentic homemade food</p>"
    related_surfaces:
      - web-mkt-landing-hero-subtitle
    recommendation: "Each surface should sound like it was written for that surface. Browse-page subhead can be more functional: 'Filter by cuisine, distance, or what's available right now.' This is a tool page, not a brand moment."
    depends_on: BV-014

  - finding_id: BV-016
    surface_id: web-mkt-landing-cuisines
    lens: brand-voice
    severity: P2
    issue: "'Cuisines / Explore Flavors / Discover authentic dishes from around the world' — 'around the world' is generic-SaaS travel-blog filler and contradicts the marketplace's actual scope (six hardcoded cuisines: South Indian, Italian, Japanese, North Indian, Mexican, Thai are NOT 'around the world')."
    evidence_excerpt: "<Badge>Cuisines</Badge> <h2>Explore Flavors</h2> <p>Discover authentic dishes from around the world</p>"
    related_surfaces: []
    recommendation: "Subhead should match scope and tone: 'Pick a cuisine, see chefs cooking it tonight.' The six cuisines should also be data-driven (top-6 cuisines by chef count from the API) rather than hardcoded — same root cause as the AI-slop metric findings."
    depends_on: null

  - finding_id: BV-017
    surface_id: web-mkt-landing-how-it-works
    lens: brand-voice
    severity: P2
    issue: "'Get Delicious Food in 3 Steps' uses 'Delicious Food' — generic consumer-marketing modifier the style guide flags as bland ('amazing food', 'best in city'). Title-cases 'Delicious Food' for emphasis where sentence case is the style-guide default."
    evidence_excerpt: "<h2>Get Delicious Food in 3 Steps</h2> <p>From discovery to delivery, we make it simple to enjoy homemade food</p>"
    related_surfaces: []
    recommendation: "Sentence case + sharper noun: 'From discovery to doorstep, in three steps.' Drop 'delicious'. The step labels themselves (Discover / Order / Enjoy) are fine; the subhead is where the drift lives."
    depends_on: null

  - finding_id: BV-018
    surface_id: web-mkt-landing-become-chef
    lens: brand-voice
    severity: P2
    issue: "'Love Cooking? Share Your Talent / Turn your passion into income. Join our community of home chefs and start earning by sharing your delicious homemade food.' — 'passion into income', 'share your talent', 'delicious homemade food' all hit the faux-artisanal + generic-DTC pattern stack. The card sits on `bg-ink` (dark background, herb CTA) which would be fine if the copy weren't shouting two voice violations at once."
    evidence_excerpt: "Love Cooking? Share Your Talent. Turn your passion into income. Join our community of home chefs and start earning by sharing your delicious homemade food."
    related_surfaces:
      - vp-mkt-register-benefits-list
    recommendation: "Functional rewrite: 'Cook from home. Sell to your neighborhood. / Set your menu, your prices, your hours. Weekly payouts, transparent fees, no exclusivity.' Anchors on what the chef actually gets, not vague aspiration. Also: this is consumer-side marketing for a chef CTA — voice should LEAN to the chef tone-matrix row (functional, time-aware) once they click through."
    depends_on: null

  - finding_id: BV-019
    surface_id: web-mkt-landing-catering-cta
    lens: brand-voice
    severity: P2
    issue: "'Catering Services / Planning an Event?' uses Title Case for 'Catering Services' (badge), and the subhead 'Get catering quotes from multiple home chefs. Perfect for parties, corporate events, and special occasions.' has a comma-listed-occasions pattern ('parties, corporate events, special occasions') that reads like an SEO meta description rather than considered marketing copy."
    evidence_excerpt: "<Badge>Catering Services</Badge> <h2>Planning an Event?</h2> <p>Get catering quotes from multiple home chefs. Perfect for parties, corporate events, and special occasions.</p>"
    related_surfaces:
      - web-mkt-hero-variants
      - web-mkt-catering-request-heading
    recommendation: "Drop the badge or sentence-case it ('Catering'). Rewrite subhead: 'Tell us about your event. Home chefs send you quotes within a day.' Specific timeline, concrete outcome."
    depends_on: BV-011

  # ─────────────────────────────────────────────────────────────────────────
  # P3 — Punctuation / capitalization / cross-link drift
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BV-020
    surface_id: web-mkt-landing-search-cta
    lens: brand-voice
    severity: P3
    issue: "Primary search CTA reads 'Find Food' (Title Case) while the shared HeroSection's equivalent CTA is 'Find Chefs' (also Title Case) and ALL other primary buttons in the design system use sentence case. Two issues: capitalization drift from style-guide buttons-rule, and label drift ('food' vs 'chefs') between two heroes that serve the same job."
    evidence_excerpt: "<Link to=\"/chefs?…\">Find Food</Link>  vs  <Button>Find Chefs</Button>"
    related_surfaces:
      - web-mkt-hero-search
      - web-mkt-hero-variants
    recommendation: "Sentence case + decide on a single noun. Recommend 'Find chefs' across both heroes (matches the route /chefs and the rest of the IA). If the homepage hero is genuinely dish-first, 'Find dishes' is acceptable — but pick one per surface and keep it consistent."
    depends_on: null

  - finding_id: BV-021
    surface_id: web-mkt-hero-how-it-works-cta
    lens: brand-voice
    severity: P3
    issue: "Secondary CTA 'How It Works' is Title Case in the shared hero AND in the homepage 'How It Works' section badge — both are violations of the buttons rule (sentence case). Lowercase 'How it works' is the style-guide default."
    evidence_excerpt: "<Link to=\"/how-it-works\">How It Works</Link>"
    related_surfaces:
      - web-mkt-landing-how-it-works
    recommendation: "Rename to 'How it works' everywhere (button, badge, route slug stays /how-it-works). This is a 5-instance global rename — same string appears in MainLayout nav too."
    depends_on: null

  - finding_id: BV-022
    surface_id: web-mkt-landing-featured-chefs-heading
    lens: brand-voice
    severity: P3
    issue: "'View All Chefs' (Title Case) on a ghost-variant button; cuisines section uses 'View All' (also Title Case). Both violate sentence-case button rule. Also: two different labels for what is essentially the same action ('View all chefs' on featured row vs 'View all' on cuisines row)."
    evidence_excerpt: "<Link to=\"/chefs\">View All Chefs <ArrowRight … /></Link>  /  <Link to=\"/chefs\">View All <ArrowRight … /></Link>"
    related_surfaces:
      - web-mkt-landing-cuisines
    recommendation: "Sentence case + unify: 'View all chefs' on both rows (or 'See all' if shorter is wanted). Pair label with section purpose: featured-row link can stay 'View all chefs' while cuisines-row should be 'View all cuisines' (currently misleading — 'View All' without an object next to a cuisine grid)."
    depends_on: null

  - finding_id: BV-023
    surface_id: web-mkt-landing-become-chef
    lens: brand-voice
    severity: P3
    issue: "Become-a-chef section has two CTAs: 'Become a Chef' (Title Case) and 'Learn More' (Title Case). Style guide: sentence case for buttons. 'Learn more →' is allowed by the style guide only when paired with topic context; standalone 'Learn More' is on the banned list."
    evidence_excerpt: "<Link to=\"/become-chef\">Become a Chef</Link>  /  <Link to=\"/chef-resources\">Learn More</Link>"
    related_surfaces: []
    recommendation: "Sentence case both: 'Become a chef' / 'Read chef resources' (or 'See how it works for chefs'). Drop the standalone 'Learn More' — replace with descriptive link text per style-guide Rule 3 vocabulary table."
    depends_on: null

  - finding_id: BV-024
    surface_id: web-mkt-brand-tagline
    lens: brand-voice
    severity: P3
    issue: "Logo tagline 'Homemade Food Delivered' is Title Case, contradicting all other tagline-like strings on the marketing surface which are sentence case or full sentences. Also: it's the cleanest brand statement in the codebase (vs the three verbose footer/hero/register variants) and isn't reused as the canonical short statement (see BV-012)."
    evidence_excerpt: "<span>Homemade Food Delivered</span>"
    related_surfaces:
      - web-mkt-layout-footer
      - web-mkt-landing-hero-subtitle
      - web-mkt-register-benefits
    recommendation: "Sentence case: 'Homemade food, delivered.' (note the comma + period for editorial feel matching `.impeccable.md` 'Confident, not loud'). Promote this exact string to `shared/constants/brand.ts` as `BRAND_STATEMENT_SHORT` and consume from footer + meta tags."
    depends_on: BV-012
```
