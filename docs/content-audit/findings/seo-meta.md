# Findings — SEO & Meta

Category: seo-meta
Lenses applied: technical-writer, legal, business-analyst, brand-voice
Inventory slice size: 27 surfaces
Total findings: 77

## Severity summary

| Severity | TW | Legal | BA | BV | Total |
|---|---|---|---|---|---|
| P0 | 1 | 4 | 3 | 2 | 10 |
| P1 | 5 | 7 | 7 | 5 | 24 |
| P2 | 10 | 4 | 7 | 5 | 26 |
| P3 | 11 | 0 | 3 | 3 | 17 |

## Cross-lens consensus surfaces

Surfaces flagged by 4 lenses (every lens agreed there's a problem):

- **`dp-seo-meta-description`** — flagged by TW(P2), Legal(P1), BA(P2), BV(P1) — "Fe3dr Delivery Partner Portal — Manage deliveries, track earnings" — term drift (Partner Portal vs Delivery), gig-worker classification framing, cross-portal voice drift
- **`md-meta-001`** — flagged by TW(P1), Legal(P0), BA(P1), BV(P0) — Mobile-delivery app name "HomeChef Delivery" vs every other surface "Fe3dr" — brand-name fork across App Store + permission prompts
- **`md-meta-003`** — flagged by TW(P1), Legal(P0), BA(P1), BV(P2) — NSCameraUsageDescription "Used to take photos" — terse, brandless, Apple-review-risk, DPDP §5 specific-purpose gap
- **`md-meta-005`** — flagged by TW(P0), Legal(P0), BA(P1), BV(P2) — NSLocationAlwaysAndWhenInUseUsageDescription background-location rationale — missing scope/retention/revocation, surveillance vocabulary, DPDP §6 consent quality
- **`web-seo-meta-description`** — flagged by TW(P2×2, P3), Legal(P1), BA(P1, P2), BV(P1) — "Fe3dr - Connect with home chefs for authentic homemade food delivered to your doorstep" — "authentic" + "doorstep" filler, 88 chars below 150-160 target, brand-first prefix wastes SERP real estate
- **`web-seo-meta-keywords`** — flagged by TW(P3), Legal(P1), BA(P2), BV(P2) — "home chef, food delivery, homemade food, local chef, catering" — deprecated SEO theatre + "catering" implies FSSAI catering license + AI-slop signal
- **`web-seo-og-description`** — flagged by TW(P2, P3), Legal(P1), BA(P1), BV(P1) — "Connect with home chefs for authentic homemade food delivered to your doorstep" — same "authentic"/"doorstep" filler echoed across OG/Twitter cards, identical to meta-description, under-length
- **`web-seo-og-title`** — flagged by TW(P3×2), Legal(P2×2), BA(P0, P1, P3), BV(P1, P3) — "Fe3dr - Homemade Food Delivered" + og:image references missing /og-image.png — Title Case, ASCII hyphen, blank social cards, no og:image:width/height/alt, hardcoded og:url

Surfaces flagged by 3 lenses (also high-priority rewrites):

- **`ap-seo-meta-title`** — flagged by TW(P1, P3), BA(P0), BV(P3) — Duplicate <title>Fe3dr Admin</title> at lines 38 and 43 in admin-portal/index.html — HTML5 spec violation + copy-paste defect pattern
- **`dp-seo-html-title`** — flagged by TW(P1, P2), BA(P0), BV(P3) — Duplicate <title>Fe3dr Delivery</title> at lines 24 and 32 in delivery-portal/index.html + missing OG/Twitter/robots tags
- **`md-meta-002`** — flagged by TW(P2), Legal(P1), BA(P3) — NSFaceIDUsageDescription "Use Face ID to log in quickly" — "log in" vs "sign in" (Sec 3 vocab), "quickly" urgency cue, missing on-device-only DPDP §5(b) reassurance
- **`md-meta-004`** — flagged by TW(P2), Legal(P1), BV(P0) — NSLocationWhenInUseUsageDescription — "HomeChef Delivery" brand drift, missing DPDP §5 retention/grievance/third-party-sharing notice, possessive-heavy voice
- **`md-meta-006`** — flagged by TW(P1), Legal(P0), BV(P2) — Android expo-location plugin rationale differs from iOS Info.plist — cross-platform notice parity gap (DPDP §5), shorter Android string than iOS
- **`vp-seo-meta-description`** — flagged by TW(P2), Legal(P2), BV(P1) — "Fe3dr Vendor Portal - Manage your kitchen, menus, orders, and earnings" — gross-vs-net "earnings" framing, "Vendor Portal" register, no robots noindex on gated portal
- **`vp-seo-meta-keywords`** — flagged by TW(P3), BA(P2), BV(P2) — "vendor portal, home chef, kitchen management, food business, menu management" — "vendor" customer-facing banned vocab + deprecated meta keywords + persona register drift
- **`web-seo-app-name`** — flagged by TW(P3), Legal(P2), BA(P2) — application-name="Fe3dr" / apple-mobile-web-app-title="Fe3dr" — no data-fiduciary identification per DPDP §5(i), bare brand without descriptor, theme_color drift between manifest.json (#f97316 orange) and index.html (#fafaf7 Paper)

Surfaces flagged by 2 lenses:

- **`ap-seo-meta-description`** — flagged by Legal(P1), BA(P3) — "Fe3dr Admin Portal - Manage users, chefs, orders, and platform analytics" — defense-in-depth leak if noindex misconfigured + "users" banned vocab
- **`web-seo-html-title`** — flagged by TW(P2), BA(P2) — "Fe3dr - Homemade Food Delivered" — static SPA root title (no per-route updates), 31 chars below 50-60 SEO target
- **`web-seo-twitter-description`** — flagged by TW(P2, P3), BA(P2) — Identical "authentic homemade food delivered to your doorstep" copy as meta/OG, no twitter:site handle, 78 chars below 100-200 Twitter target

## Technical Writer findings

```yaml
findings:
  # ============================================================
  # WITHIN-SURFACE VOICE DRIFT & BANNED VOCABULARY
  # (Sec 1-3 of style guide)
  # ============================================================
  - finding_id: TW-001
    surface_id: web-seo-meta-description
    lens: technical-writer
    severity: P2
    issue: "Voice drift: 'authentic homemade food delivered to your doorstep' uses two filler/cliché phrases ('authentic', 'delivered to your doorstep') that read as generic food-app boilerplate and violate Rule 3 (chrome-light, supports the photo) and Rule 5 (restraint). 'Authentic' is also adjacent to the banned 'artisanal' family — both signal craft-washing the style guide explicitly avoids."
    evidence_excerpt: "Fe3dr - Connect with home chefs for authentic homemade food delivered to your doorstep"
    recommendation: "Rewrite to confident-not-loud, customer-warmer voice: 'Order homemade food from chefs cooking near you. Same-day pickup and delivery.' Trims the kitsch, keeps SEO keywords (homemade, chefs, delivery), and uses sensory verbs ('cooking', 'pickup')."
    depends_on: null

  - finding_id: TW-002
    surface_id: web-seo-og-description
    lens: technical-writer
    severity: P2
    issue: "Same 'authentic homemade food delivered to your doorstep' filler as meta description, repeated in OG and Twitter cards. Brand-drift adjacent vocabulary on the surface seen most by social sharers."
    evidence_excerpt: "Connect with home chefs for authentic homemade food delivered to your doorstep"
    recommendation: "Match the rewritten meta description so all SEO surfaces share one voice: 'Order homemade food from chefs cooking near you.' Drop 'authentic' and 'doorstep'."
    depends_on: TW-001

  - finding_id: TW-003
    surface_id: web-seo-twitter-description
    lens: technical-writer
    severity: P2
    issue: "Identical filler copy as OG and meta description. Same banned-adjacent vocabulary on the Twitter share card."
    evidence_excerpt: "Connect with home chefs for authentic homemade food delivered to your doorstep"
    recommendation: "Match unified rewrite from TW-001. One SEO description, used across meta/OG/Twitter, all dynamic from one source string."
    depends_on: TW-001

  - finding_id: TW-004
    surface_id: web-seo-meta-description
    lens: technical-writer
    severity: P3
    issue: "Verb 'Connect with' is vague and SaaS-flavoured ('connect' is what dating apps and LinkedIn say). Style guide Customer tone is suggestive/sensory ('Discover', 'Try', 'Order'), not transactional/networking."
    evidence_excerpt: "Connect with home chefs..."
    recommendation: "Replace 'Connect with' with a food-verb: 'Order from', 'Discover', 'Eat from'. Customer tone matrix example: 'Discover chefs cooking in your neighborhood.'"
    depends_on: TW-001

  # ============================================================
  # SEO LENGTH & FORMAT ISSUES
  # (50-60 char title, 150-160 char meta description, OG completeness)
  # ============================================================
  - finding_id: TW-005
    surface_id: web-seo-html-title
    lens: technical-writer
    severity: P2
    issue: "<title> is 31 characters — well below the 50-60 char SEO sweet spot. Wastes SERP real estate on the primary customer-facing entry surface."
    evidence_excerpt: "Fe3dr - Homemade Food Delivered"
    recommendation: "Lengthen to 50-60 chars while keeping editorial restraint: 'Fe3dr — Homemade Food from Local Chefs, Delivered' (52 chars) or 'Fe3dr — Order Homemade Food from Chefs Near You' (49 chars)."
    depends_on: null

  - finding_id: TW-006
    surface_id: web-seo-meta-description
    lens: technical-writer
    severity: P2
    issue: "Meta description is 88 characters — below the 150-160 char SEO target. Truncated value on the highest-traffic public surface; loses keyword density and CTR opportunity."
    evidence_excerpt: "Fe3dr - Connect with home chefs for authentic homemade food delivered to your doorstep"
    recommendation: "Extend to 150-160 chars while fixing voice: 'Order homemade food from chefs cooking in your neighborhood. Browse menus, schedule pickup or delivery, and pay securely on Fe3dr.' (~155 chars)."
    depends_on: TW-001

  - finding_id: TW-007
    surface_id: web-seo-og-title
    lens: technical-writer
    severity: P3
    issue: "OG title (31 chars) is shorter than the recommended 60-90 char OG sweet spot. Social previews under-utilise the headline slot."
    evidence_excerpt: "Fe3dr - Homemade Food Delivered"
    recommendation: "Match the rewritten <title> so both stay in sync: 'Fe3dr — Homemade Food from Local Chefs, Delivered'."
    depends_on: TW-005

  - finding_id: TW-008
    surface_id: web-seo-og-description
    lens: technical-writer
    severity: P3
    issue: "OG description is 78 chars; recommended is 100-200 chars for richer social previews. Currently truncated value."
    evidence_excerpt: "Connect with home chefs for authentic homemade food delivered to your doorstep"
    recommendation: "Extend to 150-200 chars in parallel with TW-006: 'Order homemade food from chefs cooking in your neighborhood. Browse menus, schedule pickup or delivery, and pay securely on Fe3dr.'"
    depends_on: TW-006

  - finding_id: TW-009
    surface_id: web-seo-twitter-title
    lens: technical-writer
    severity: P3
    issue: "Twitter title 31 chars — below 70 char Twitter card limit. Same under-utilisation as OG title."
    evidence_excerpt: "Fe3dr - Homemade Food Delivered"
    recommendation: "Match unified rewritten title from TW-005."
    depends_on: TW-005

  - finding_id: TW-010
    surface_id: web-seo-twitter-description
    lens: technical-writer
    severity: P3
    issue: "Twitter description 78 chars — Twitter allows up to ~200 chars. Same under-utilisation."
    evidence_excerpt: "Connect with home chefs for authentic homemade food delivered to your doorstep"
    recommendation: "Match unified rewritten description from TW-006."
    depends_on: TW-006

  # ============================================================
  # CROSS-SURFACE INCONSISTENCY (Sec 3 — same concept, different name)
  # ============================================================
  - finding_id: TW-011
    surface_id: md-meta-001
    lens: technical-writer
    severity: P1
    issue: "Brand-name drift: mobile-delivery app.json displays 'HomeChef Delivery' as the iOS/Android app name, while every other surface (web, vendor, delivery web portal, admin) uses 'Fe3dr'. Customers/drivers installing the app see a different brand than the marketing surfaces."
    evidence_excerpt: "\"name\": \"HomeChef Delivery\""
    recommendation: "Rename Expo app to 'Fe3dr Delivery' to match `dp-seo-app-name` and `dp-seo-html-title`. Update bundle identifier in a controlled migration if needed, but the display name must match marketing immediately."
    depends_on: null

  - finding_id: TW-012
    surface_id: dp-seo-meta-description
    lens: technical-writer
    severity: P2
    issue: "Term drift: meta description says 'Delivery Partner Portal' while every other delivery surface (html title, app-name, apple title) says 'Delivery'. Pick one — the inventory note already flags this."
    evidence_excerpt: "Fe3dr Delivery Partner Portal - Manage deliveries, track earnings"
    recommendation: "Standardise on driver-facing term 'Driver' or 'Delivery' per Sec 3 vocab. Rewrite: 'Fe3dr Driver — Manage your deliveries and earnings.' Decision: pick a single noun for the driver app and apply it across html title, app-name, meta description, OG."
    depends_on: null

  - finding_id: TW-013
    surface_id: md-meta-006
    lens: technical-writer
    severity: P1
    issue: "Permission-rationale drift between iOS and Android. iOS Info.plist says 'tracks your location in the background to keep customers updated on their delivery status' (16 words); Android expo-location plugin says 'tracks your location during deliveries to keep customers updated' (10 words). Different copy for the same OS-level prompt creates regulatory ambiguity and review-team friction."
    evidence_excerpt: "iOS: 'HomeChef Delivery tracks your location in the background to keep customers updated on their delivery status.' / Android: 'HomeChef Delivery tracks your location during deliveries to keep customers updated.'"
    recommendation: "Unify both to one canonical string, defined once in a constants file and injected into both plist + expo-location plugin: 'Fe3dr Delivery tracks your location during active deliveries to share live ETAs with customers. Tracking stops automatically when your shift ends.' Same wording, both stores, both reviewers."
    depends_on: TW-011

  - finding_id: TW-014
    surface_id: web-seo-app-name
    lens: technical-writer
    severity: P3
    issue: "`apple-mobile-web-app-title` and `application-name` are both just 'Fe3dr' — no descriptor. When customers add the web app to home screen, the icon label is identical to the vendor and delivery PWAs once installed side-by-side (they'd see 'Fe3dr', 'Fe3dr Vendor Portal', 'Fe3dr Delivery'). The bare 'Fe3dr' loses context."
    evidence_excerpt: "apple-mobile-web-app-title: 'Fe3dr' / application-name: 'Fe3dr'"
    recommendation: "Either keep bare 'Fe3dr' as the consumer brand and add descriptors to the other two (already done — vendor + delivery have descriptors), and document that 'Fe3dr' alone always means the consumer app. Style guide: add to Sec 3 vocabulary list 'Fe3dr — consumer storefront; Fe3dr Vendor Portal — chef portal; Fe3dr Delivery — driver portal.'"
    depends_on: null

  # ============================================================
  # MISSING / DUPLICATED SEO TAGS
  # ============================================================
  - finding_id: TW-015
    surface_id: dp-seo-html-title
    lens: technical-writer
    severity: P1
    issue: "Duplicated `<title>` tag inside `<head>` (lines 24 and 32 of apps/delivery-portal/index.html). HTML spec disallows multiple titles; browsers/crawlers use the last one. Bug, not a copy issue, but it ships on the driver portal SEO surface."
    evidence_excerpt: "Line 24: <title>Fe3dr Delivery</title>  / Line 32: <title>Fe3dr Delivery</title>"
    recommendation: "Delete line 32 (the second `<title>` after the no-flash bootstrap script). Same fix already needed in admin-portal/index.html (lines 38 + 43). File a P1 cleanup for both."
    depends_on: null

  - finding_id: TW-016
    surface_id: ap-seo-meta-title
    lens: technical-writer
    severity: P1
    issue: "Duplicated `<title>` tag in apps/admin-portal/index.html (lines 38 and 43). Same bug class as TW-015."
    evidence_excerpt: "Line 38: <title>Fe3dr Admin</title>  / Line 43: <title>Fe3dr Admin</title>"
    recommendation: "Delete the second `<title>` tag. Lint rule to forbid >1 `<title>` in `<head>` should be added to the build."
    depends_on: TW-015

  - finding_id: TW-017
    surface_id: dp-seo-html-title
    lens: technical-writer
    severity: P2
    issue: "delivery-portal/index.html has no Open Graph or Twitter card meta tags, no `<meta name=\"author\">`, and no `<meta name=\"keywords\">`. While the portal is gated, drivers do share links with each other and ops; preview cards on Slack/WhatsApp will fall back to filename, not brand."
    evidence_excerpt: "apps/delivery-portal/index.html (lines 1-38) — only meta description, no og:* or twitter:* tags"
    recommendation: "Add `og:type=website`, `og:title=Fe3dr Delivery`, `og:description=<unified rewrite from TW-012>`, `og:image=/og-image.png`, `og:url=https://delivery.fe3dr.com`, `og:site_name=Fe3dr Delivery`. Also add `<meta name=\"robots\" content=\"noindex, nofollow\">` (matches admin-portal pattern; this is a gated app)."
    depends_on: TW-012

  - finding_id: TW-018
    surface_id: ap-seo-meta-title
    lens: technical-writer
    severity: P3
    issue: "admin-portal correctly has `noindex, nofollow`, but the meta description is still verbose and exposed in the HTML source ('Manage users, chefs, orders, and platform analytics'). Anyone viewing source can see the admin scope. Low-risk but unnecessary surface."
    evidence_excerpt: "Fe3dr Admin Portal - Manage users, chefs, orders, and platform analytics"
    recommendation: "Either drop meta description on admin-portal (noindex'd anyway, so SEO value = 0) or shorten to 'Fe3dr internal admin.' Don't enumerate capabilities."
    depends_on: null

  - finding_id: TW-019
    surface_id: vp-seo-meta-description
    lens: technical-writer
    severity: P2
    issue: "vendor-portal has no `<meta name=\"robots\">` tag. Authenticated portal (gated) should not be indexed — Google indexing login pages leaks the portal's existence and pollutes brand SERPs."
    evidence_excerpt: "apps/vendor-portal/index.html lines 24-27: meta description + keywords + author, no robots directive"
    recommendation: "Add `<meta name=\"robots\" content=\"noindex, nofollow\">` matching admin-portal pattern. Same fix required for delivery-portal per TW-017."
    depends_on: null

  - finding_id: TW-020
    surface_id: vp-seo-og-title
    lens: technical-writer
    severity: P3
    issue: "vendor-portal OG has no `og:image` and no Twitter card tags. Internal links shared in Slack render as text-only previews."
    evidence_excerpt: "Lines 30-34: og:type, og:title, og:description, og:url, og:site_name — no og:image, no twitter:*"
    recommendation: "Add `og:image=/og-image-vendor.png` (or reuse `/og-image.png`) and a minimal `twitter:card=summary` + `twitter:title` + `twitter:description`. Drop og entirely if vendor portal will be `noindex` per TW-019 — but social preview still works for shared links."
    depends_on: TW-019

  - finding_id: TW-021
    surface_id: web-seo-og-title
    lens: technical-writer
    severity: P3
    issue: "OG image tag references `/og-image.png` but has no companion `og:image:width`, `og:image:height`, or `og:image:alt`. WhatsApp/Slack/Twitter often skip the image preview without dimensions; screen readers skip without alt."
    evidence_excerpt: "<meta property=\"og:image\" content=\"/og-image.png\" />"
    recommendation: "Add `og:image:width`, `og:image:height` (typical 1200×630), `og:image:alt` (e.g., 'Plated homemade meal beside the Fe3dr wordmark'), and an absolute URL (`https://fe3dr.com/og-image.png`) per OG spec — relative URLs sometimes fail in legacy crawlers."
    depends_on: null

  # ============================================================
  # KEYWORD-META & LEGACY-SEO HYGIENE
  # ============================================================
  - finding_id: TW-022
    surface_id: web-seo-meta-keywords
    lens: technical-writer
    severity: P3
    issue: "`<meta name=\"keywords\">` is ignored by Google since ~2009 and treated as a spam signal by some crawlers. The string 'home chef, food delivery, homemade food, local chef, catering' adds zero SEO value and bloats HTML."
    evidence_excerpt: "home chef, food delivery, homemade food, local chef, catering"
    recommendation: "Delete the `keywords` meta tag. Move keyword intent into the meta description (which Google does read) and into page H1/H2s. Same fix for vendor-portal (vp-seo-meta-keywords)."
    depends_on: null

  - finding_id: TW-023
    surface_id: vp-seo-meta-keywords
    lens: technical-writer
    severity: P3
    issue: "Same as TW-022 — `keywords` meta on vendor-portal is deprecated SEO theatre."
    evidence_excerpt: "vendor portal, home chef, kitchen management, food business, menu management"
    recommendation: "Delete the tag. Vendor portal is gated and should be noindex anyway (TW-019)."
    depends_on: TW-022

  # ============================================================
  # MOBILE PERMISSION RATIONALES (Driver tone: glanceable, imperative,
  # but Apple-review-safe: must explain purpose specifically)
  # ============================================================
  - finding_id: TW-024
    surface_id: md-meta-003
    lens: technical-writer
    severity: P1
    issue: "NSCameraUsageDescription is 'Used to take photos' — 4 words, no subject, no purpose. Apple App Review consistently rejects vague permission rationales like this; users also can't tell why the app wants camera access. Inventory note already flags this as a P1 candidate."
    evidence_excerpt: "Used to take photos"
    recommendation: "Driver-tone rewrite explaining the specific use: 'Fe3dr Delivery uses your camera to photograph proof-of-delivery and document any pickup or drop-off issues.' Satisfies Apple's specificity rule and tells the driver the genuine reason."
    depends_on: TW-011

  - finding_id: TW-025
    surface_id: md-meta-002
    lens: technical-writer
    severity: P2
    issue: "NSFaceIDUsageDescription 'Use Face ID to log in quickly' is friendly but uses the banned-adjacent urgency cue 'quickly' (Rule 5 — restraint) and doesn't say what 'log in' protects. Style guide also prefers 'Sign in' over 'log in' (Sec 3 vocabulary)."
    evidence_excerpt: "Use Face ID to log in quickly"
    recommendation: "Replace with: 'Use Face ID to sign in to Fe3dr Delivery without entering your password.' Drops the urgency word, swaps the banned 'log in' for 'sign in', explains what Face ID replaces."
    depends_on: TW-011

  - finding_id: TW-026
    surface_id: md-meta-004
    lens: technical-writer
    severity: P2
    issue: "NSLocationWhenInUseUsageDescription is acceptable but uses the brand-drift name 'HomeChef Delivery' (see TW-011). Once the app rebrands to Fe3dr Delivery, this string must follow."
    evidence_excerpt: "HomeChef Delivery needs your location to show your position on the delivery map."
    recommendation: "Rewrite to: 'Fe3dr Delivery uses your location to show your position on the delivery map and route to pickups.' Keeps Apple-safe specificity, fixes brand drift, and adds the pickup-routing purpose (which is the real second use)."
    depends_on: TW-011

  - finding_id: TW-027
    surface_id: md-meta-005
    lens: technical-writer
    severity: P0
    issue: "NSLocationAlwaysAndWhenInUseUsageDescription is the highest-risk permission prompt (background location). Current copy doesn't promise when tracking stops, which Apple explicitly requires for background-location approval per App Store Review Guideline 5.1.5. Inventory note already flags as P0 (drift vs in-app modal)."
    evidence_excerpt: "HomeChef Delivery tracks your location in the background to keep customers updated on their delivery status."
    recommendation: "Rewrite to App-Review-compliant string: 'Fe3dr Delivery tracks your location in the background only during active deliveries to share live ETAs with customers. Tracking stops automatically when you complete a delivery or end your shift.' Specifies start condition, purpose, AND stop condition. Mirror exactly in Android expo-location plugin (TW-013)."
    depends_on: TW-011
```

## Legal findings

```yaml
---
# Legal lens findings — seo-meta slice
# Inventory rows audited: 27
# All findings flagged depends_on: "needs lawyer review"
# Lens scope: India jurisdiction (DPDP 2023, ASCI Code, FSSAI Licensing Regs, CP Rules 2020,
#             Code on Social Security 2020) + jurisdiction-agnostic best-practice.
# Category note: SEO-META is a lighter-touch category for legal lens. Focus areas
#                are (a) brand-name accuracy in OG/Twitter cards, (b) substantiation
#                of marketing-style title/description claims under ASCI, (c) native
#                permission rationale strings as DPDP §5 notice surfaces, and
#                (d) data-fiduciary identification on indexable pages.

- finding_id: LEG-001
  surface_id: md-meta-001
  lens: legal
  severity: P0
  issue: 'Brand-name mismatch between web/portals ("Fe3dr") and mobile-delivery app ("HomeChef Delivery") — same legal entity surfaces under two consumer-facing names across OG tags and app-store listings'
  evidence_excerpt: 'app.json name "HomeChef Delivery" vs web/vendor/admin meta "Fe3dr"'
  recommendation: 'Pick one consumer-facing brand and use it consistently across `<title>`, og:site_name, og:title, twitter:title, application-name, apple-mobile-web-app-title, app.json `name`, and store-listing metadata. If "HomeChef Delivery" is a sub-brand of Fe3dr, disclose the parent/operator on app store listings and in privacy policy. Inconsistent identity creates ASCI Ch.I.1 misleading-representation exposure and undermines DPDP §5(i) data-fiduciary identification.'
  citation: 'ASCI Code Ch.I.1 (truthful representation); DPDP Act 2023 §5(i) (identity of data fiduciary in notice); CP Rules 2020 (misleading advertisement)'
  depends_on: 'needs lawyer review'

- finding_id: LEG-002
  surface_id: web-seo-meta-description
  lens: legal
  severity: P1
  issue: '"authentic homemade food" — origin/quality claim in indexable meta description with no substantiation pathway; "homemade" has no statutory definition under FSSAI and "authentic" is an absolute marketing claim'
  evidence_excerpt: '"Fe3dr - Connect with home chefs for authentic homemade food delivered to your doorstep"'
  recommendation: 'Either (a) define "homemade" / "authentic" on the home page with substantiation (e.g., "prepared in domestic kitchens registered with FSSAI") and link from meta description landing, or (b) soften to a non-claim ("food cooked by home chefs near you"). ASCI Ch.I.1 requires marketing claims to be capable of substantiation; meta description text is indexed and quoted by search engines as a representation.'
  citation: 'ASCI Code Ch.I.1; CP Rules 2020 §4 (misleading advertisement); FSSAI Labelling & Display Regs 2020 (no statutory definition of "homemade")'
  depends_on: 'needs lawyer review'

- finding_id: LEG-003
  surface_id: web-seo-og-description
  lens: legal
  severity: P1
  issue: '"authentic homemade food delivered to your doorstep" — same unsubstantiated origin/quality claim re-emitted in og:description and twitter:description, amplifying reach across social platforms'
  evidence_excerpt: '"Connect with home chefs for authentic homemade food delivered to your doorstep" (og:description, twitter:description, web-seo-og-description, web-seo-twitter-description)'
  recommendation: 'Fix once at source per LEG-002 — OG and Twitter descriptions inherit the same risk and are scraped/cached by Facebook, X, LinkedIn, WhatsApp link previews. Single substantiation paragraph on home page can cover all four surfaces.'
  citation: 'ASCI Code Ch.I.1; CP Rules 2020 §4'
  depends_on: 'needs lawyer review'

- finding_id: LEG-004
  surface_id: web-seo-meta-keywords
  lens: legal
  severity: P1
  issue: '"catering" keyword — implies catering service offering; FSSAI Licensing & Registration Regs 2011 Schedule 2 treats catering as a separate license category. If platform/chefs are not catering-licensed, indexing this keyword is a misleading representation.'
  evidence_excerpt: '"home chef, food delivery, homemade food, local chef, catering"'
  recommendation: 'Confirm whether platform onboards FSSAI catering-licensed chefs. If not, remove "catering" from keywords and any landing pages. If yes, ensure catering chefs surface a distinct catering license number on their profile (separate from home-kitchen FSSAI registration).'
  citation: 'FSSAI Licensing & Registration Regs 2011 Sch. 2 (catering license); ASCI Code Ch.I.1'
  depends_on: 'needs lawyer review'

- finding_id: LEG-005
  surface_id: web-seo-og-title
  lens: legal
  severity: P2
  issue: '"Homemade Food Delivered" — absolute claim used as standalone tagline in og:title, twitter:title, and `<title>`; reads as a definitive promise rather than a service descriptor'
  evidence_excerpt: '"Fe3dr - Homemade Food Delivered"'
  recommendation: 'Reframe as service descriptor ("Home chefs, delivered" or "Order from home chefs near you") to avoid the absolute "homemade" claim. The title appears in browser tabs, search results, and social previews — three high-visibility surfaces of the same representation.'
  citation: 'ASCI Code Ch.I.1; best-practice (avoid absolute claims in indexed surfaces)'
  depends_on: 'needs lawyer review'

- finding_id: LEG-006
  surface_id: dp-seo-meta-description
  lens: legal
  severity: P1
  issue: '"Delivery Partner Portal" — "Partner" framing in indexable meta description has gig-worker classification implications; Code on Social Security 2020 distinguishes "gig worker" / "platform worker" from "partner"/"employee" and the public-facing label should align with the binding driver agreement classification'
  evidence_excerpt: '"Fe3dr Delivery Partner Portal - Manage deliveries, track earnings"'
  recommendation: 'Confirm driver-agreement classification (independent contractor / platform worker / gig worker). Meta description should use the term that matches the binding contract — "partner" carries quasi-employment connotations in Indian labour law context. Inconsistency with the actual agreement is contractual evidence in disputes.'
  citation: 'Code on Social Security 2020 §2(35), §2(60) (gig/platform worker definitions); best-practice (contract-public-statement alignment)'
  depends_on: 'needs lawyer review'

- finding_id: LEG-007
  surface_id: md-meta-003
  lens: legal
  severity: P0
  issue: 'NSCameraUsageDescription "Used to take photos" — fails DPDP §5(a) specific-purpose notice and Apple App Store Review Guideline 5.1.1 (purpose-string clarity). Does not say WHAT photos, for WHAT purpose, WHERE they are sent, or how long retained'
  evidence_excerpt: '"Used to take photos"'
  recommendation: 'Rewrite to name (a) the specific in-app action (e.g., "Capture delivery proof-of-handoff photos"), (b) who sees the photo (chef, customer, support), (c) retention duration, and (d) link to privacy policy. The Info.plist string is the DPDP §5 notice moment for camera data collection — terse text is a regulatory gap.'
  citation: 'DPDP Act 2023 §5(a) and §5(d) (specific purpose, retention notice); Apple App Store Review Guideline 5.1.1; Android Data Safety Section requirements (Play Console policy)'
  depends_on: 'needs lawyer review'

- finding_id: LEG-008
  surface_id: md-meta-002
  lens: legal
  severity: P1
  issue: 'NSFaceIDUsageDescription "Use Face ID to log in quickly" — biometric purpose string omits DPDP §5(b) processing-detail notice (Face ID template stays on device via Secure Enclave; clarify no biometric data leaves the device)'
  evidence_excerpt: '"Use Face ID to log in quickly"'
  recommendation: 'Add the standard "Face ID data stays on your device and is never sent to Fe3dr/HomeChef" reassurance. Biometric data is sensitive personal data under DPDP §2(t) framing; even when Apple keeps it on-device, the data principal needs notice that the platform does not receive it. Reduces consent-quality challenges.'
  citation: 'DPDP Act 2023 §5(b) (nature/extent of processing); IT(Reasonable Security Practices) Rules 2011 r.3 (sensitive personal data)'
  depends_on: 'needs lawyer review'

- finding_id: LEG-009
  surface_id: md-meta-004
  lens: legal
  severity: P1
  issue: 'NSLocationWhenInUseUsageDescription names purpose but omits DPDP §5 elements: retention period, third-party sharing (customer app receives driver position in real-time), grievance-officer/right-to-erase reference'
  evidence_excerpt: '"HomeChef Delivery needs your location to show your position on the delivery map."'
  recommendation: 'iOS purpose strings have a 175-char practical limit; the standard approach is a short purpose + in-app secondary notice with full DPDP §5 disclosure. Confirm the secondary in-app modal exists, surfaces retention duration, names the customer app as a recipient of the live position, and links to grievance officer contact. If absent, the Info.plist string is the only notice and is insufficient.'
  citation: 'DPDP Act 2023 §5(a), §5(c) (rights), §5(e) (grievance redressal officer)'
  depends_on: 'needs lawyer review'

- finding_id: LEG-010
  surface_id: md-meta-005
  lens: legal
  severity: P0
  issue: 'Background-location rationale "tracks your location in the background to keep customers updated on their delivery status" — background tracking is the most sensitive collection moment and the iOS string omits retention period, processing scope (only during active delivery? always?), and DPDP §6 free-revocable-consent reference'
  evidence_excerpt: '"HomeChef Delivery tracks your location in the background to keep customers updated on their delivery status."'
  recommendation: 'Resolve four gaps: (1) scope ("only when a delivery is active"), (2) retention ("location pings retained 30/60/90 days for dispute resolution then deleted"), (3) revocation ("you can turn this off in iOS Settings — your active deliveries will pause"), (4) grievance-officer reference for DPDP rights. Background-location processing under DPDP needs §6 free-given, specific, informed consent; the Info.plist string is the consent moment.'
  citation: 'DPDP Act 2023 §6 (consent quality), §5(a)(d)(e) (purpose, retention, grievance); Apple App Store Review Guideline 5.1.1(iv) (background location justification)'
  depends_on: 'needs lawyer review'

- finding_id: LEG-011
  surface_id: md-meta-006
  lens: legal
  severity: P0
  issue: 'iOS/Android location-rationale drift — iOS NSLocationAlwaysAndWhenInUseUsageDescription (md-meta-005) and Android locationAlwaysAndWhenInUsePermission (md-meta-006) emit different wording for the same processing activity. DPDP §5 notice must be consistent across the channels through which it is given; divergent wording creates "which notice did the data principal consent to" evidentiary ambiguity'
  evidence_excerpt: 'iOS: "...tracks your location in the background to keep customers updated on their delivery status." vs Android: "...tracks your location during deliveries to keep customers updated."'
  recommendation: 'Single canonical purpose string maintained in one config source and synced to both Info.plist and expo-location plugin. Both platforms should read the same text after applying any platform-specific length constraints. Document the canonical version in privacy policy and reference it as the binding notice.'
  citation: 'DPDP Act 2023 §5 (notice contents) and §6 (consent quality); best-practice (cross-platform notice parity)'
  depends_on: 'needs lawyer review'

- finding_id: LEG-012
  surface_id: web-seo-app-name
  lens: legal
  severity: P2
  issue: 'No data-fiduciary identification on indexed surfaces — meta `author="Fe3dr"` and og:site_name="Fe3dr" reference a brand name, not the registered legal entity. DPDP §5(i) requires the notice to identify the data fiduciary; the home page is the entry point of the consent journey'
  evidence_excerpt: 'application-name="Fe3dr", apple-mobile-web-app-title="Fe3dr", og:site_name="Fe3dr"'
  recommendation: 'Either (a) include legal-entity name in the website footer with a clear "Operated by [Legal Entity Pvt Ltd]" line that DPDP notice can reference, or (b) add the legal entity to og:site_name as "Fe3dr (by [Legal Entity Pvt Ltd])". Meta tags themselves do not need to carry the full entity, but the indexed surface they describe must.'
  citation: 'DPDP Act 2023 §5(i); CP (E-commerce) Rules 2020 r.5(3) (entity identification)'
  depends_on: 'needs lawyer review'

- finding_id: LEG-013
  surface_id: ap-seo-meta-description
  lens: legal
  severity: P1
  issue: 'Admin-portal meta description is set even though inventory notes "noindex/nofollow" — if the noindex header/meta is ever misconfigured, the description "Manage users, chefs, orders, and platform analytics" tips off attackers about internal tooling surface'
  evidence_excerpt: '"Fe3dr Admin Portal - Manage users, chefs, orders, and platform analytics"'
  recommendation: 'Strip the descriptive meta from the admin portal entirely (set to empty or generic "Restricted"). Combined with noindex this gives defense-in-depth — if noindex fails, the leaked surface still carries no useful signal. Internal-tooling discoverability is best-practice security/legal hygiene under IT Rules 2021 §3 due-diligence.'
  citation: 'IT Rules 2021 §3 (due diligence); best-practice (defense-in-depth for restricted surfaces)'
  depends_on: 'needs lawyer review'

- finding_id: LEG-014
  surface_id: vp-seo-meta-description
  lens: legal
  severity: P2
  issue: '"Manage your kitchen, menus, orders, and earnings" — "earnings" framing in indexable description may set commercial expectation; if vendor portal also surfaces tax obligations (TDS, GST) the public claim should not imply earnings are net'
  evidence_excerpt: '"Fe3dr Vendor Portal - Manage your kitchen, menus, orders, and earnings"'
  recommendation: 'Either reframe ("Manage your kitchen, menus, orders, and payouts") or keep "earnings" but ensure the in-app earnings screen clearly states gross-vs-net, TDS deductions, GST handling, and platform fees. Public description sets expectation that the product must meet.'
  citation: 'CP Rules 2020 §4 (no misleading representation); ASCI Code Ch.I.1; GST Act provisions on supplier disclosure'
  depends_on: 'needs lawyer review'

- finding_id: LEG-015
  surface_id: web-seo-og-title
  lens: legal
  severity: P2
  issue: 'OG image `/og-image.png` and Twitter image referenced from indexable meta — if image contains food photography sourced from chefs, ensure consent/license to reuse in marketing surfaces is captured at chef onboarding (image-rights chain)'
  evidence_excerpt: 'og:image=/og-image.png, twitter:image=/og-image.png'
  recommendation: 'Confirm whether og-image.png contains photography sourced from onboarded chefs or stock/licensed art. If chef-sourced, chef onboarding T&C must include a marketing-use license; if it is stock, retain license proof. OG image is cached and re-distributed by social platforms — withdrawal is operationally hard, so the up-front license must be solid.'
  citation: 'Copyright Act 1957 §17, §18 (licensing of works); best-practice (image-rights audit trail)'
  depends_on: 'needs lawyer review'

# Audit summary
# - Total findings: 15
# - P0: 4  (LEG-001 brand-mismatch, LEG-007 camera-string, LEG-010 background-location, LEG-011 iOS/Android drift)
# - P1: 7  (LEG-002 homemade claim, LEG-003 OG/Twitter echo, LEG-004 catering keyword, LEG-006 partner framing,
#           LEG-008 Face ID notice, LEG-009 in-use location, LEG-013 admin meta leak)
# - P2: 4  (LEG-005 absolute tagline, LEG-012 data-fiduciary identification, LEG-014 earnings framing,
#           LEG-015 og:image rights)
# - P3: 0
# - Top surfaces by exposure:
#     1. md-meta-* (mobile-delivery app.json permission strings) — 5 findings, DPDP §5/§6 notice surface
#     2. web-seo-* (apps/web/index.html meta tags) — 5 findings, ASCI claim substantiation
#     3. dp-seo-meta-description, vp-seo-meta-description, ap-seo-meta-description — labour/leak/expectation
#     4. Cross-app brand-name consistency (Fe3dr vs HomeChef Delivery)
# - Primary citations relied on:
#     DPDP Act 2023 §5, §6 (notice + consent quality)
#     ASCI Code Ch.I.1 (truthful, substantiable claims)
#     CP Rules 2020 §4 + CP (E-commerce) Rules 2020 r.5 (misleading representation, entity ID)
#     FSSAI Licensing & Registration Regs 2011 Sch. 2 (catering license)
#     Code on Social Security 2020 §2(35), §2(60) (gig/platform worker)
#     Apple App Store Review Guideline 5.1.1 (purpose-string clarity)
#     IT Rules 2021 §3 (due diligence on restricted surfaces)
#     Copyright Act 1957 §17, §18 (image-rights chain)
```

## Business Analyst findings

```yaml
findings:

  # ── P0: Broken conversion path / missing OG asset ─────────────────────────

  - finding_id: BA-SEO-001
    surface_id: web-seo-og-title
    lens: business-analyst
    severity: P0
    issue: "OG and Twitter card images declared but og-image.png is absent from the public directory — all social shares render a blank card"
    evidence_excerpt: |
      <meta property="og:image" content="/og-image.png" />
      <meta name="twitter:image" content="/og-image.png" />
      # apps/web/public/ contains: favicon.svg, logo-icon.svg, manifest.json, offline.html, sw.js
      # og-image.png is NOT present
    recommendation: >
      Create and commit a 1200×630px OG image at apps/web/public/og-image.png.
      The image should show an appetising food photograph with the Fe3dr wordmark in Geist 700,
      no decorative text overlay beyond the tagline. A blank card on social shares means
      every link shared on WhatsApp, Instagram Stories, Twitter/X, and Slack renders with no
      visual — this is a top-of-funnel dead-end for the most common word-of-mouth channel.
    metric_hypothesis: "organic social referral CTR; food-delivery apps are heavily discovery-driven via social shares — a blank OG card drops click-through to near zero"
    depends_on: null

  - finding_id: BA-SEO-002
    surface_id: dp-seo-html-title
    lens: business-analyst
    severity: P0
    issue: "Duplicate <title> tag in delivery-portal/index.html — two identical <title>Fe3dr Delivery</title> elements at lines 24 and 32"
    evidence_excerpt: |
      line 24: <title>Fe3dr Delivery</title>
      line 32: <title>Fe3dr Delivery</title>
    recommendation: >
      Remove the duplicate <title> at line 32. While browsers display only the last title,
      duplicate tags are a parsing error that can confuse crawlers and PWA install prompts.
      Same defect pattern exists in admin-portal/index.html (lines 38 and 43) — fix both together.
    metric_hypothesis: "driver app install rate via browser PWA prompt — malformed head may suppress install banners on some Android browsers"
    depends_on: null

  - finding_id: BA-SEO-003
    surface_id: ap-seo-meta-title
    lens: business-analyst
    severity: P0
    issue: "Duplicate <title> tag in admin-portal/index.html — two identical <title>Fe3dr Admin</title> elements at lines 38 and 43"
    evidence_excerpt: |
      line 38: <title>Fe3dr Admin</title>
      line 43: <title>Fe3dr Admin</title>
    recommendation: >
      Remove the duplicate <title> at line 43. Admin portal likely won't be indexed (noindex/nofollow
      is set correctly), but duplicate tags indicate a copy-paste error pattern that should be caught
      by an HTML lint step in CI before it reappears on indexed surfaces.
    metric_hypothesis: "admin operator trust — while low direct SEO impact, the same pattern that creates this bug on admin can corrupt the customer-facing title if the copy-paste migrates there"
    depends_on: BA-SEO-002

  # ── P1: Conversion-critical clarity / click-through rate ──────────────────

  - finding_id: BA-SEO-004
    surface_id: web-seo-meta-description
    lens: business-analyst
    severity: P1
    issue: "Primary meta description starts with the brand name prefix 'Fe3dr -' which wastes the first 8 characters Google shows in SERPs — benefit-first copy would increase click-through rate"
    evidence_excerpt: "Fe3dr - Connect with home chefs for authentic homemade food delivered to your doorstep"
    recommendation: >
      Rewrite to lead with the customer benefit and defer the brand name to the end or omit it
      (Google shows the page title separately):
      "Order homemade food from home chefs near you. Fresh, authentic meals delivered to your door."
      (86 chars — well within the 155-char limit, adds a second action-oriented sentence that
      the current version lacks). The word 'authentic' is kept but 'doorstep' edges toward
      lifestyle-copy cliche; 'door' is crisper.
    metric_hypothesis: "organic search CTR — SERP descriptions that lead with the user benefit rather than the brand name consistently outperform brand-first patterns by 10-25% CTR in A/B tests"
    depends_on: null

  - finding_id: BA-SEO-005
    surface_id: web-seo-og-title
    lens: business-analyst
    severity: P1
    issue: "OG title and HTML title are identical strings ('Fe3dr - Homemade Food Delivered') — a missed opportunity to write a social-share-optimised title that differs from the SERP title"
    evidence_excerpt: |
      <title>Fe3dr - Homemade Food Delivered</title>
      <meta property="og:title" content="Fe3dr - Homemade Food Delivered" />
      <meta name="twitter:title" content="Fe3dr - Homemade Food Delivered" />
    recommendation: >
      SERP title can remain 'Fe3dr — Homemade Food Delivered' (good keyword density for search).
      Social (OG/Twitter) title should be written for human curiosity, not crawlers:
      "Your neighbourhood has home chefs. Find yours on Fe3dr."
      This is more likely to generate shares and clicks in social contexts where users are
      browsing a feed rather than actively searching.
    metric_hypothesis: "social referral CTR — social titles optimised for curiosity rather than keywords generate more shares and clicks from feed-browsing users"
    depends_on: BA-SEO-001

  - finding_id: BA-SEO-006
    surface_id: web-seo-og-description
    lens: business-analyst
    severity: P1
    issue: "OG description is a near-verbatim copy of the meta description but without the brand prefix — all four social tags (OG title, OG desc, Twitter title, Twitter desc) use two interchangeable strings, making the social card feel templated rather than crafted"
    evidence_excerpt: |
      og:description — "Connect with home chefs for authentic homemade food delivered to your doorstep"
      twitter:description — "Connect with home chefs for authentic homemade food delivered to your doorstep"
      meta description — "Fe3dr - Connect with home chefs for authentic homemade food delivered to your doorstep"
    recommendation: >
      Write a distinct social description that adds information the title doesn't have:
      "Browse menus, read reviews, and order from home chefs cooking in your city. No restaurants.
      Just real food from real kitchens."
      This surfaces differentiating information (menus, reviews, real-kitchens positioning) that
      would not appear in a SERP snippet, giving a social share more persuasive surface area.
    metric_hypothesis: "social share conversion — a description that adds new information beyond the title increases social card click-through versus a redundant paraphrase of the title"
    depends_on: BA-SEO-001

  - finding_id: BA-SEO-007
    surface_id: md-meta-003
    lens: business-analyst
    severity: P1
    issue: "iOS NSCameraUsageDescription 'Used to take photos' is dangerously terse — iOS may reject the app or reduce permission grant rate; this is shown verbatim on the iOS permission sheet"
    evidence_excerpt: '"NSCameraUsageDescription": "Used to take photos"'
    recommendation: >
      iOS App Review guidelines require camera usage strings to explain the specific use case.
      A generic 'Used to take photos' will likely trigger App Store rejection or reduced user
      permission grant rates. Rewrite to:
      "We use your camera to photograph your pickup location as proof of delivery."
      This is honest (drivers confirm deliveries), specific, and passes App Review.
      Impact: camera permission denial means drivers cannot submit delivery proof, breaking
      the delivery confirmation flow and triggering false dispute rates.
    metric_hypothesis: "App Store approval rate; driver delivery confirmation completion rate — vague permission strings fail review and lower user grant rates"
    depends_on: null

  - finding_id: BA-SEO-008
    surface_id: md-meta-005
    lens: business-analyst
    severity: P1
    issue: "iOS NSLocationAlwaysAndWhenInUseUsageDescription differs from the Android expo-location locationAlwaysAndWhenInUsePermission string — platform permission drift means drivers see inconsistent rationale on iOS vs Android"
    evidence_excerpt: |
      iOS:     "HomeChef Delivery tracks your location in the background to keep customers updated on their delivery status."
      Android: "HomeChef Delivery tracks your location during deliveries to keep customers updated."
    recommendation: >
      Align both platforms to a single canonical string that satisfies both iOS App Review and
      Google Play policy:
      "HomeChef Delivery tracks your location during active deliveries so customers see
      live updates. Background tracking stops when you end your shift."
      The added final sentence is critical: Android Play policy requires background-location
      strings to explain when the background access ends. The current Android string omits this
      and risks Play Store rejection. The iOS string is more explicit but still lacks the
      'when it ends' clause.
    metric_hypothesis: "Play Store approval rate; driver onboarding completion rate — permission rejections block the driver from working, directly reducing supply-side capacity"
    depends_on: null

  - finding_id: BA-SEO-009
    surface_id: md-meta-001
    lens: business-analyst
    severity: P1
    issue: "Mobile delivery app display name is 'HomeChef Delivery' while all web/portal surfaces use the 'Fe3dr' brand — a split-brand identity undermines SEO coherence and user recognition when drivers search the App Store"
    evidence_excerpt: |
      app.json — "name": "HomeChef Delivery"
      web/index.html — apple-mobile-web-app-title: "Fe3dr"
      vendor-portal/index.html — application-name: "Fe3dr Vendor Portal"
    recommendation: >
      Decide on one brand name and apply it consistently across all surfaces.
      If 'Fe3dr' is the go-to-market brand, the mobile app should be 'Fe3dr Delivery' to match.
      If 'HomeChef' is a distinct product brand for the native app tier, document it as intentional
      and ensure no crossover leakage in customer-facing copy.
      Inconsistency between web and App Store brand names reduces word-of-mouth referral
      effectiveness: a customer saying 'download Fe3dr' sends a driver searching for
      'HomeChef Delivery' — they won't find it.
    metric_hypothesis: "driver acquisition from customer referral; App Store search visibility — brand name mismatch between web and app store breaks referral chains"
    depends_on: null

  - finding_id: BA-SEO-010
    surface_id: vp-seo-html-title
    lens: business-analyst
    severity: P1
    issue: "Vendor portal has no og:image or Twitter card tags at all — if a chef shares their portal URL (e.g., in a WhatsApp chef group), the share renders with no visual and no social card metadata"
    evidence_excerpt: |
      apps/vendor-portal/index.html — no <meta property="og:image">, no <meta name="twitter:card">
      og:title and og:description are present but og:image is absent
    recommendation: >
      Add a chef-oriented OG image and Twitter card tags to vendor-portal/index.html.
      The image should differ from the customer-facing og-image.png — show a kitchen/chef
      context rather than a food-delivery context. Also add twitter:card = summary_large_image
      and twitter:image pointing to the same asset.
      Chef community word-of-mouth (WhatsApp groups, food blogger communities) is a key
      chef-acquisition channel — link unfurling with no image kills that channel.
    metric_hypothesis: "chef acquisition via community word-of-mouth — blank link unfurls in chef WhatsApp groups suppress organic chef referrals"
    depends_on: null

  # ── P2: Missed opportunity / medium-traffic surface ───────────────────────

  - finding_id: BA-SEO-011
    surface_id: web-seo-html-title
    lens: business-analyst
    severity: P2
    issue: "HTML title is a static string that never updates per-route — users who deep-link to a chef profile or menu page see 'Fe3dr - Homemade Food Delivered' as the tab title and SERP snippet, losing per-page keyword opportunity"
    evidence_excerpt: '<title>Fe3dr - Homemade Food Delivered</title> — apps/web/index.html:69 (static, SPA root)'
    recommendation: >
      Implement dynamic per-route title updates using React Helmet or the built-in
      document.title setter in route components. Priority pages to target:
      - Chef profile: "{Chef Name} | Home Chef in {City} | Fe3dr"
      - Menu page: "{Chef Name}'s Menu | Fe3dr"
      - Browse page: "Home Chefs in {City} | Fe3dr"
      This enables long-tail SEO capture for chef-name searches, which is the most likely
      search a customer makes after hearing about a specific chef by word of mouth.
    metric_hypothesis: "organic search traffic for chef-name and city-level queries — static SPA titles mean Google indexes only the root title for all pages"
    depends_on: null

  - finding_id: BA-SEO-012
    surface_id: web-seo-meta-description
    lens: business-analyst
    severity: P2
    issue: "No structured data (JSON-LD) for LocalBusiness, FoodService, or Restaurant schema — the platform loses eligibility for Google rich results (star ratings, operating hours, order action) in SERPs"
    evidence_excerpt: |
      apps/web/index.html — no <script type="application/ld+json"> found
      apps/web/public/ — no JSON-LD files found
    recommendation: >
      Add JSON-LD structured data for:
      1. WebSite schema on the root URL (enables Sitelinks Searchbox in Google)
      2. FoodService or LocalBusiness schema on each chef profile page (enables star ratings
         and direct-order rich results for 'home chef near me' queries)
      3. BreadcrumbList on browse/category pages
      At minimum, implement WebSite schema in the SPA root and FoodService on chef profile
      pages. This is the highest-leverage unblocked SEO action available.
    metric_hypothesis: "organic SERP CTR from rich results — sites with FoodService structured data that qualify for rich snippets see 20-30% higher CTR than plain SERP listings"
    depends_on: BA-SEO-011

  - finding_id: BA-SEO-013
    surface_id: web-seo-meta-keywords
    lens: business-analyst
    severity: P2
    issue: "Meta keywords tag is present but Google has ignored meta keywords since 2009 — the tag provides zero SEO value and uses engineering budget to maintain a no-op field"
    evidence_excerpt: '<meta name="keywords" content="home chef, food delivery, homemade food, local chef, catering" />'
    recommendation: >
      Remove the meta keywords tag from web/index.html and vendor-portal/index.html.
      The same engineering effort should be redirected to the structured data gap (BA-SEO-012)
      which does provide measurable ranking benefit.
      Note: removing keywords tags has zero negative impact — Google, Bing, and all major
      crawlers explicitly ignore this tag.
    metric_hypothesis: "engineering velocity — removing a maintained no-op frees developer time for structured data implementation which has real CTR impact"
    depends_on: null

  - finding_id: BA-SEO-014
    surface_id: vp-seo-meta-keywords
    lens: business-analyst
    severity: P2
    issue: "Vendor portal meta keywords tag includes 'vendor portal' and 'food business' — 'Vendor' is a banned term per the style guide vocabulary list; customer-facing copy must use 'Home chef'"
    evidence_excerpt: '<meta name="keywords" content="vendor portal, home chef, kitchen management, food business, menu management" />'
    recommendation: >
      Remove the keywords tag (see BA-SEO-013 for rationale). If a keywords-style content
      strategy is ever revisited, replace 'vendor portal' with 'chef portal' or 'home chef
      dashboard' to align with the style guide. 'Food business' is also vague — 'home chef
      business' would be more specific.
    metric_hypothesis: "brand vocabulary consistency — 'Vendor' in meta copy leaks into content strategy documentation and training data, normalising a banned term"
    depends_on: BA-SEO-013

  - finding_id: BA-SEO-015
    surface_id: web-seo-app-name
    lens: business-analyst
    severity: P2
    issue: "PWA manifest.json sets theme_color to '#f97316' (orange) which conflicts with the brand design system's Herb green accent and Paper/Ink palette — the OS chrome shown on Android home screen will be orange, not brand-consistent"
    evidence_excerpt: |
      manifest.json: "theme_color": "#f97316"
      index.html theme-color: "#fafaf7" (Paper, correct)
      Design system: primary accent is Herb green oklch(0.48 0.13 145), never orange
    recommendation: >
      Update manifest.json theme_color to the Herb green hex equivalent (#3d6b3a or nearest)
      to match the brand accent used throughout the app.
      Also update background_color from '#ffffff' to the Paper background '#fafaf7' to
      prevent a jarring white flash on Android when the PWA splashscreen loads.
      The manifest and index.html theme-color tags currently disagree — they should use
      the same value so PWA install banners and OS chrome are consistent.
    metric_hypothesis: "PWA install rate; brand trust — an orange OS chrome on an app that uses green everywhere creates a 'cheap' first impression that reduces install completion"
    depends_on: null

  - finding_id: BA-SEO-016
    surface_id: dp-seo-meta-description
    lens: business-analyst
    severity: P2
    issue: "Delivery portal meta description uses 'Delivery Partner Portal' but the portal title and apple-app-title use 'Fe3dr Delivery' — internal name drift that would confuse a driver googling their portal URL"
    evidence_excerpt: |
      meta description: "Fe3dr Delivery Partner Portal - Manage deliveries, track earnings"
      <title>: "Fe3dr Delivery"
      apple-mobile-web-app-title: "Fe3dr Delivery"
    recommendation: >
      Align meta description to match the title brand:
      "Fe3dr Delivery — manage your deliveries and track your earnings."
      Optionally expand to capture driver-acquisition search intent:
      "Fe3dr Delivery — driver portal for managing pickups, tracking earnings, and navigating
      to customers."
      The current description is accurate but the name mismatch ('Partner Portal' vs 'Delivery')
      is a consistency bug the inventory has already flagged.
    metric_hypothesis: "driver portal search discoverability — drivers Googling 'Fe3dr driver login' will see a SERP snippet that doesn't match the page title, reducing click confidence"
    depends_on: null

  - finding_id: BA-SEO-017
    surface_id: web-seo-twitter-description
    lens: business-analyst
    severity: P2
    issue: "No twitter:site handle declared — Twitter cards will render without an attributed account, reducing the viral coefficient of any shared link because followers cannot click-through to the brand account"
    evidence_excerpt: |
      Present tags: twitter:card, twitter:title, twitter:description, twitter:image
      Missing: twitter:site (e.g., @fe3dr or @homechefapp)
    recommendation: >
      Add <meta name="twitter:site" content="@fe3dr" /> (or whichever handle is active).
      Twitter/X cards with a twitter:site attribution show the account handle below the card,
      enabling one-tap follow. This is a zero-cost top-of-funnel brand awareness signal.
    metric_hypothesis: "social brand follower growth; social referral CTR — unattributed cards miss the attribution link that converts a share viewer into a brand follower"
    depends_on: BA-SEO-001

  # ── P3: Minor / low-traffic surface ──────────────────────────────────────

  - finding_id: BA-SEO-018
    surface_id: web-seo-og-title
    lens: business-analyst
    severity: P3
    issue: "og:url is hardcoded to 'https://fe3dr.com' in all page variants — for an SPA this means every route's shared URL resolves OG metadata to the root, so deep-linked chef or menu pages show the homepage description when shared"
    evidence_excerpt: '<meta property="og:url" content="https://fe3dr.com" />'
    recommendation: >
      Update og:url dynamically per route in each page component, matching the canonical URL.
      This requires moving OG tags out of index.html into per-route React Helmet calls.
      Priority routes: chef profile pages, menu pages, browse-by-city pages.
      Without this, a customer sharing "Chef Priya's Biryani" shows the generic homepage card.
    metric_hypothesis: "social referral conversion — a shared chef-profile link that unfurls with chef-specific title/description converts 3-5x better than a generic homepage card"
    depends_on: BA-SEO-011

  - finding_id: BA-SEO-019
    surface_id: md-meta-002
    lens: business-analyst
    severity: P3
    issue: "iOS Face ID usage description 'Use Face ID to log in quickly' is technically accurate but misses the opportunity to address driver security concerns — drivers may hesitate to enable biometrics without understanding data handling"
    evidence_excerpt: '"NSFaceIDUsageDescription": "Use Face ID to log in quickly"'
    recommendation: >
      Expand to: "Use Face ID to sign in securely. Your biometric data stays on your device
      and is never sent to Fe3dr."
      The second sentence addresses the primary anxiety (data leaving the device) that causes
      drivers to deny biometric permissions. Higher biometric adoption = lower sign-in friction
      = faster shift starts.
    metric_hypothesis: "driver shift start time; biometric permission grant rate — addressing the primary objection in the permission string increases grant rate and reduces sign-in abandonment"
    depends_on: null

  - finding_id: BA-SEO-020
    surface_id: ap-seo-meta-description
    lens: business-analyst
    severity: P3
    issue: "Admin portal has a live meta description despite being noindex/nofollow — the description is reachable by operators bookmarking or sharing the URL internally, and 'Manage users, chefs, orders, and platform analytics' uses 'users' which is a banned generic term per the style guide"
    evidence_excerpt: |
      <meta name="description" content="Fe3dr Admin Portal - Manage users, chefs, orders, and platform analytics" />
      <meta name="robots" content="noindex, nofollow" />
    recommendation: >
      Update the description to use role-specific language consistent with the style guide:
      "Fe3dr Admin — manage customers, home chefs, orders, and platform analytics."
      'users' → 'customers' and 'chefs' (style guide approved terms).
      Low urgency since noindex suppresses SERP exposure, but internal operator bookmarks
      and Slack link-unfurls will show this description.
    metric_hypothesis: "internal operator vocabulary consistency — admin copy that uses banned terms trains internal stakeholders to use those terms in external communications"
    depends_on: null
```

## Brand Voice findings

```yaml
# Brand Voice findings — seo-meta category
# Lens: brand-voice (cross-app, cross-surface voice consistency)
# Inputs: STYLE-GUIDE.md, .impeccable.md, CONTENT-INVENTORY.md seo-meta slice (27 rows)

# ─── P0: Cross-surface brand-name contradiction (worst breaks platform trust) ──

- finding_id: BV-SEO-001
  surface_id: md-meta-001
  lens: brand-voice
  severity: P0
  issue: "Brand-name fork between web and mobile: every web surface and OG/Twitter card says 'Fe3dr', the mobile-delivery app store name and all four iOS/Android permission prompts say 'HomeChef Delivery'. Same product, two brands — customers, drivers, and reviewers see different names in the App Store vs the web landing."
  evidence_excerpt: "\"name\": \"HomeChef Delivery\" (app.json) vs <meta name=\"application-name\" content=\"Fe3dr Delivery\" /> (apps/delivery-portal/index.html)"
  related_surfaces: ["md-meta-002", "md-meta-003", "md-meta-004", "md-meta-005", "md-meta-006", "dp-seo-apple-title", "dp-seo-app-name", "dp-seo-html-title", "dp-seo-meta-description", "web-seo-app-name", "web-seo-og-title", "web-seo-twitter-title", "web-seo-html-title", "vp-seo-html-title", "vp-seo-og-title", "ap-seo-meta-title"]
  recommendation: "Pick one brand name and apply it everywhere. If 'Fe3dr' is the live brand (web + portals + production domains fe3dr.com all use it), rename the Expo app to 'Fe3dr Delivery', update slug/scheme to fe3dr-delivery, rewrite all four Info.plist + expo-location rationale strings to start with 'Fe3dr Delivery' instead of 'HomeChef Delivery'. Cross-surface naming consistency is Rule-0 for brand trust."
  depends_on: null

- finding_id: BV-SEO-002
  surface_id: md-meta-004
  lens: brand-voice
  severity: P0
  issue: "iOS permission rationale strings hardcode the wrong brand ('HomeChef Delivery') on a safety-critical surface (foreground + background location). The first sentence a driver reads in the iOS permission sheet contradicts the Fe3dr brand they signed up under."
  evidence_excerpt: "\"HomeChef Delivery needs your location to show your position on the delivery map.\" / \"HomeChef Delivery tracks your location in the background to keep customers updated on their delivery status.\""
  related_surfaces: ["md-meta-005", "md-meta-006", "dp-seo-meta-description"]
  recommendation: "Rewrite to: 'Fe3dr Delivery shows your position on the delivery map.' / 'Fe3dr Delivery shares your location with the customer so they can track their order in progress.' Same brand name as the rest of the product, plain English, no possessive 'your … your' stacking."
  depends_on: BV-SEO-001

# ─── P1: Persona-tone violation on entry SEO surfaces ─────────────────────────

- finding_id: BV-SEO-003
  surface_id: web-seo-meta-description
  lens: brand-voice
  severity: P1
  issue: "Customer landing meta description uses 'authentic homemade food' — a faux-artisanal trigger word the style guide explicitly bans (alongside 'artisanal', 'handcrafted with love'). 'Authentic' on a customer SERP snippet reads like terracotta-era marketing copy that survived the migration."
  evidence_excerpt: "Fe3dr - Connect with home chefs for authentic homemade food delivered to your doorstep"
  related_surfaces: ["web-seo-og-description", "web-seo-twitter-description"]
  recommendation: "Drop 'authentic'. Rewrite: 'Order home-cooked food from chefs near you. Delivered fresh, from kitchens around the corner.' Confident · Appetizing · Quietly modern — no artisanal trigger words."
  depends_on: null

- finding_id: BV-SEO-004
  surface_id: web-seo-og-description
  lens: brand-voice
  severity: P1
  issue: "OG/Twitter card description is identical to the meta description and carries the same 'authentic homemade' artisanal trigger. Social-share cards are the highest-leverage brand surface (every share = a brand impression) and they currently read as legacy marketing copy."
  evidence_excerpt: "Connect with home chefs for authentic homemade food delivered to your doorstep"
  related_surfaces: ["web-seo-meta-description", "web-seo-twitter-description"]
  recommendation: "Rewrite OG/Twitter description distinct from meta-description (search vs share is a different read context). Suggested: 'Home-cooked food from chefs near you. Browse menus, order, and track delivery.' Plain, action-led, no 'authentic'."
  depends_on: BV-SEO-003

- finding_id: BV-SEO-005
  surface_id: web-seo-og-title
  lens: brand-voice
  severity: P1
  issue: "OG/Twitter/document title uses Title Case ('Fe3dr - Homemade Food Delivered'). Style guide §4 requires sentence case for buttons and labels and bans Title Case 'outside of proper nouns'. Title tag is the loudest brand voice surface on web; Title Case here reads as 2018-era SEO copy, not 2026 confident-restraint."
  evidence_excerpt: "Fe3dr - Homemade Food Delivered"
  related_surfaces: ["web-seo-twitter-title", "web-seo-html-title"]
  recommendation: "Rewrite in sentence case: 'Fe3dr — home-cooked food delivered'. Use em-dash (—), not hyphen, between brand and tagline (typographic convention). Tagline reads as a calm statement, not a SERP keyword stuffing."
  depends_on: null

# ─── P1: Cross-app voice drift — same product surface, four different tones ───

- finding_id: BV-SEO-006
  surface_id: vp-seo-meta-description
  lens: brand-voice
  severity: P1
  issue: "The four portal meta descriptions are written in four different voices: customer ('Connect with home chefs for authentic homemade food'), vendor ('Manage your kitchen, menus, orders, and earnings'), delivery ('Manage deliveries, track earnings'), admin ('Manage users, chefs, orders, and platform analytics'). The vendor and admin descriptions read as four-item Oxford-comma lists; the delivery one drops the comma; the customer one is a marketing sentence. No shared sentence template across the brand."
  evidence_excerpt: "Fe3dr Vendor Portal - Manage your kitchen, menus, orders, and earnings"
  related_surfaces: ["dp-seo-meta-description", "ap-seo-meta-description", "web-seo-meta-description"]
  recommendation: "Pick one template per persona surface. Suggested: '{Brand} {Portal}. {Verb-phrase for the persona's primary job}.' e.g., 'Fe3dr for chefs. Run your kitchen — menus, orders, payouts.' / 'Fe3dr for drivers. Pick up, deliver, get paid.' / 'Fe3dr for admins. Users, chefs, orders, analytics.' Period after brand, single primary verb, no Oxford-comma list of unrelated nouns."
  depends_on: null

- finding_id: BV-SEO-007
  surface_id: dp-seo-meta-description
  lens: brand-voice
  severity: P1
  issue: "Inventory itself flags drift: meta description says 'Delivery Partner Portal' but the apple-mobile-web-app-title says 'Fe3dr Delivery'. Style guide vocabulary section: customer-facing 'Delivery partner', driver-facing 'Driver'. The delivery portal IS driver-facing, so 'Delivery Partner Portal' is the wrong register entirely — it's the customer-facing euphemism leaking into the driver's own product."
  evidence_excerpt: "Fe3dr Delivery Partner Portal - Manage deliveries, track earnings"
  related_surfaces: ["dp-seo-apple-title", "dp-seo-app-name", "dp-seo-html-title"]
  recommendation: "Pick driver-facing register everywhere on the driver portal: 'Fe3dr for drivers — pick-ups, deliveries, payouts.' Drop 'Partner Portal' (it's both wrong-persona and bureaucratic). Driver tone per style guide §2 = telegraphic, imperative, ≤4 words where possible."
  depends_on: null

# ─── P2: AI-slop SEO copy patterns ────────────────────────────────────────────

- finding_id: BV-SEO-008
  surface_id: web-seo-meta-keywords
  lens: brand-voice
  severity: P2
  issue: "The customer meta keywords tag ('home chef, food delivery, homemade food, local chef, catering') is the textbook AI-slop SEO pattern Google has ignored since ~2009. It also lists 'catering' which is not a product surface in the inventory. Keeping it ships a signal that the brand still believes in 2009 SEO theatre."
  evidence_excerpt: "home chef, food delivery, homemade food, local chef, catering"
  related_surfaces: ["vp-seo-meta-keywords"]
  recommendation: "Delete the <meta name=\"keywords\"> tag entirely from web and vendor-portal index.html. Google has ignored it for 15+ years, Bing for nearly as long; keeping it just ships clutter and reads as low-confidence SEO. Style guide Rule 1 — confident, not loud."
  depends_on: null

- finding_id: BV-SEO-009
  surface_id: vp-seo-meta-keywords
  lens: brand-voice
  severity: P2
  issue: "Vendor portal keywords ('vendor portal, home chef, kitchen management, food business, menu management') compounds the same AI-slop SEO pattern AND uses 'vendor' as a customer-facing keyword. Style guide explicitly bans 'Vendor' customer-facing ('Home chef' ✅ / 'Vendor' ❌); shipping it as a search keyword publishes the wrong word to crawlers."
  evidence_excerpt: "vendor portal, home chef, kitchen management, food business, menu management"
  related_surfaces: ["web-seo-meta-keywords"]
  recommendation: "Delete the meta keywords tag. If keeping for legacy reasons (don't), at minimum strip 'vendor portal' and 'food business' — both are anti-brand vocabulary from the style-guide banned list / wrong-persona register."
  depends_on: BV-SEO-008

- finding_id: BV-SEO-010
  surface_id: md-meta-003
  lens: brand-voice
  severity: P2
  issue: "NSCameraUsageDescription is two words ('Used to take photos') with no subject, no context, no brand. Inventory itself flags as 'VERY terse; P1 candidate'. Two words on a permission sheet reads as a placeholder a developer typed in 30 seconds, not as the brand."
  evidence_excerpt: "Used to take photos"
  related_surfaces: ["md-meta-002"]
  recommendation: "Rewrite consistent with the location strings: 'Fe3dr Delivery uses the camera so you can take a proof-of-delivery photo at drop-off.' Subject + action + reason. Same template across all NS*UsageDescription strings so the four prompts read as one brand voice."
  depends_on: BV-SEO-001

- finding_id: BV-SEO-011
  surface_id: md-meta-005
  lens: brand-voice
  severity: P2
  issue: "Background-location rationale is also wordy and possessive-heavy: 'HomeChef Delivery tracks your location in the background to keep customers updated on their delivery status.' Three 'your/their' in one sentence, plus 'tracks' (surveillance vocabulary) where 'shares with the customer' is the actual intent. Driver voice per style guide §2 is telegraphic; this is the opposite."
  evidence_excerpt: "HomeChef Delivery tracks your location in the background to keep customers updated on their delivery status."
  related_surfaces: ["md-meta-004", "md-meta-006"]
  recommendation: "Rewrite shorter and reframed: 'Fe3dr Delivery shares your location with the customer so they can see when you're nearby.' Reframes from 'tracking' (creepy) to 'sharing with the customer' (consensual + customer-positive). Same Fe3dr brand name as the rest of the product."
  depends_on: BV-SEO-001

- finding_id: BV-SEO-012
  surface_id: md-meta-006
  lens: brand-voice
  severity: P2
  issue: "expo-location plugin Android rationale ('tracks your location during deliveries to keep customers updated') differs from the iOS Info.plist string ('tracks your location in the background to keep customers updated on their delivery status'). Inventory flags this drift explicitly. Same permission, same product, two slightly different sentences — Android and iOS users see different brand voices for the same consent."
  evidence_excerpt: "HomeChef Delivery tracks your location during deliveries to keep customers updated."
  related_surfaces: ["md-meta-005"]
  recommendation: "Use the same string across iOS Info.plist and the expo-location plugin Android rationale (the rewritten version from BV-SEO-011): 'Fe3dr Delivery shares your location with the customer so they can see when you're nearby.' Single source of truth for permission rationale across platforms."
  depends_on: BV-SEO-011

# ─── P3: Duplicate / minor punctuation drift ──────────────────────────────────

- finding_id: BV-SEO-013
  surface_id: dp-seo-html-title
  lens: brand-voice
  severity: P3
  issue: "<title>Fe3dr Delivery</title> is duplicated twice in apps/delivery-portal/index.html (line 24 and line 32). Browser uses the last one, but search-engine crawlers and HTML validators flag duplicate <title> as malformed head. Inventory notes the duplication."
  evidence_excerpt: "<title>Fe3dr Delivery</title> ... <title>Fe3dr Delivery</title>"
  related_surfaces: []
  recommendation: "Delete the duplicate <title> on line 32 of apps/delivery-portal/index.html. Same drift exists in apps/admin-portal/index.html (lines 38 and 43) — delete the line-43 duplicate there too. Single <title> per document is HTML5 spec."
  depends_on: null

- finding_id: BV-SEO-014
  surface_id: ap-seo-meta-title
  lens: brand-voice
  severity: P3
  issue: "Duplicate <title>Fe3dr Admin</title> tag also exists in apps/admin-portal/index.html (lines 38 and 43). Same defect as BV-SEO-013 — appears to be a copy-paste from the no-flash theme bootstrap block accidentally including the title."
  evidence_excerpt: "<title>Fe3dr Admin</title> (line 38) ... <title>Fe3dr Admin</title> (line 43)"
  related_surfaces: ["dp-seo-html-title"]
  recommendation: "Delete the duplicate <title> on line 43 of apps/admin-portal/index.html."
  depends_on: BV-SEO-013

- finding_id: BV-SEO-015
  surface_id: web-seo-og-title
  lens: brand-voice
  severity: P3
  issue: "All title-line constructions use ASCII hyphen ' - ' between brand and tagline ('Fe3dr - Homemade Food Delivered', 'Fe3dr Vendor Portal -', 'Fe3dr Delivery Partner Portal -', 'Fe3dr Admin Portal -'). Editorial-luxury brand voice per .impeccable.md and the new design system uses em-dash (—) for typographic separators. Hyphen reads as developer-default; em-dash reads as intentional."
  evidence_excerpt: "Fe3dr - Homemade Food Delivered"
  related_surfaces: ["web-seo-meta-description", "web-seo-twitter-title", "web-seo-html-title", "vp-seo-meta-description", "dp-seo-meta-description", "ap-seo-meta-description"]
  recommendation: "Replace ' - ' with ' — ' (em-dash with hairspaces) in every title and meta-description across all four portal index.html files. Small detail, but the typographic separator is one of the loudest brand-voice signals on web."
  depends_on: null
```
