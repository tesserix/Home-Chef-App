# Findings — Core UX

Category: core-ux
Lenses applied: technical-writer, legal, business-analyst, brand-voice
Inventory slice size: 594 surfaces
Total findings: 467

## Severity summary

| Severity | TW | Legal | BA | BV | Total |
|---|---|---|---|---|---|
| P0 | 10 | 27 | 6 | 12 | 55 |
| P1 | 20 | 40 | 19 | 21 | 100 |
| P2 | 26 | 26 | 29 | 24 | 105 |
| P3 | 144 | 7 | 16 | 40 | 207 |

## Cross-lens consensus surfaces (flagged by multiple lenses)

Surfaces flagged by 4 lenses (highest priority — every lens agreed there's a problem):

- **`mc-order-detail-status-labels`** — flagged by TW(P0), Legal(P1), BA(P1), BV(P0)
- **`web-ux-orders-status-labels`** — flagged by TW(P0), Legal(P1), BA(P1, P2), BV(P0, P0, P0, P2)
- **`dp-ux-active-cancel-prompt`** — flagged by TW(P0), Legal(P0), BA(P0), BV(P1)
- **`vp-ux-settings-order-acceptance`** — flagged by TW(P0), Legal(P1), BA(P0), BV(P0)
- **`web-ux-checkout-summary`** — flagged by TW(P1), Legal(P0), BA(P1), BV(P1)
- **`web-ux-checkout-delivery-time`** — flagged by TW(P1), Legal(P0), BA(P1), BV(P3)
- **`web-ux-orderdetail-cancel`** — flagged by TW(P3), Legal(P0), BA(P0), BV(P1)
- **`web-ux-orderdetail-sections`** — flagged by TW(P3), Legal(P0), BA(P2), BV(P2, P3)
- **`web-ux-layout-nav`** — flagged by TW(P3), Legal(P1), BA(P1), BV(P0, P0, P0, P0, P1, P1, P1, P1)
- **`web-ux-cart-checkout-cta`** — flagged by TW(P1), Legal(P2), BA(P1), BV(P2)
- **`web-ux-checkout-tip`** — flagged by TW(P1), Legal(P1), BA(P1), BV(P3)
- **`mc-catering-form-labels`** — flagged by TW(P1), Legal(P1), BA(P1), BV(P2)

Surfaces flagged by 3 lenses (also high-priority rewrites):

- **`ap-delivery-statuses`** — flagged by TW(P0), Legal(P2), BV(P1)
- **`dp-ux-active-mark-as`** — flagged by TW(P0), Legal(P0), BV(P1)
- **`dp-ux-delivery-pickup-label`** — flagged by TW(P0), BA(P2), BV(P0)
- **`md-core-079`** — flagged by TW(P0), BA(P2), BV(P0)
- **`mc-checkout-place-order`** — flagged by TW(P1), Legal(P0), BA(P0)
- **`web-ux-checkout-payment-section`** — flagged by TW(P1), Legal(P0), BV(P3)
- **`web-ux-chefdetail-status-badges`** — flagged by TW(P1), Legal(P0), BA(P1)
- **`web-ux-chef-menu`** — flagged by TW(P3), Legal(P0), BA(P2)
- **`dp-ux-stripe-intro`** — flagged by TW(P3), Legal(P0), BV(P3)
- **`ap-layout-nav-reviews`** — flagged by TW(P2), BA(P3), BV(P0)
- **`web-ux-chef-profile`** — flagged by TW(P3), Legal(P0), BA(P2)
- **`mc-checkout-totals`** — flagged by TW(P3), Legal(P0), BV(P3)
- **`vp-ux-settings-payout-section`** — flagged by Legal(P0), BA(P3), BV(P1)
- **`dp-ux-nav-bottom-items`** — flagged by TW(P1), BA(P3), BV(P1)
- **`dp-ux-nav-partner-items`** — flagged by TW(P1), Legal(P2), BV(P1)
- **`web-ux-cart-heading`** — flagged by TW(P1), BA(P2), BV(P2)
- **`dp-ux-stripe-active-gateway`** — flagged by TW(P1), Legal(P1), BV(P3)
- **`web-ux-chef-social`** — flagged by TW(P3), Legal(P1), BA(P2)
- **`md-core-018`** — flagged by TW(P3), Legal(P1), BV(P2)
- **`vp-ux-menu-view-page`** — flagged by TW(P3, P3, P3), Legal(P1), BA(P1)
- **`dp-ux-active-cancel-btn`** — flagged by TW(P3), Legal(P1), BV(P1)
- **`web-ux-chefdetail-reviews`** — flagged by TW(P3), Legal(P1), BV(P3)
- **`web-ux-checkout-address-form`** — flagged by TW(P3), Legal(P1), BA(P3)
- **`dp-ux-partner-detail-verify-block`** — flagged by TW(P3), Legal(P1), BA(P2)
- **`mc-cartsheet-checkout-cta`** — flagged by TW(P3), Legal(P1), BA(P2)
- **`ap-notifsettings-categories`** — flagged by TW(P3, P3), Legal(P1), BV(P1)
- **`vp-ux-layout-nav`** — flagged by TW(P3), Legal(P2), BA(P1)
- **`vp-ux-payouts-title`** — flagged by TW(P3), Legal(P1), BA(P2)
- **`vp-ux-kitchen-photos`** — flagged by TW(P3, P3), BA(P2), BV(P2)
- **`vp-ux-settings-password`** — flagged by TW(P3), Legal(P2), BV(P3)
- **`mc-profile-more-rows`** — flagged by TW(P3), BA(P3), BV(P2)
- **`mv-menunew-desc-ph`** — flagged by TW(P3), Legal(P2), BA(P3)
- **`ap-secsettings-apikey-fields`** — flagged by TW(P3, P3), Legal(P2), BV(P3)
- **`ap-secsettings-session-fields`** — flagged by TW(P3), Legal(P2), BV(P3)
- **`dp-ux-partner-performance`** — flagged by TW(P3), BA(P2), BV(P3)
- **`mv-undo-cta`** — flagged by TW(P3), BA(P3), BV(P3)

## Technical Writer findings

```yaml
# Technical Writer lens findings — core-ux category
# Auditor: TW lens agent
# Date: 2026-05-13
# Style guide: docs/content-audit/STYLE-GUIDE.md
# Brief: docs/content-audit/lens-briefs/technical-writer.md
# Slice size: 594 rows. Approach: P0/P1 per-row; P2/P3 grouped by pattern.

findings:

  # =====================================================================
  # P0 — Cross-app verb / status drift that risks operational confusion
  # =====================================================================

  - finding_id: TW-001
    surface_id: mc-order-detail-status-labels
    lens: technical-writer
    severity: P0
    issue: "Order status taxonomy drifts across customer screens: detail uses 'Ready for Pickup' / 'On the Way'; OrderCard uses 'Ready' / 'Picked Up'; OrderTimeline uses 'On the Way'. Same order shows different status names depending on the screen the customer opens."
    evidence_excerpt: "Pending / Confirmed / Preparing / Ready for Pickup / On the Way / Delivered / Cancelled  (detail)  vs  Pending / Confirmed / Preparing / Ready / Picked Up / Delivered / Cancelled  (card)"
    recommendation: "Lock one canonical customer-facing status set in i18n: Pending / Confirmed / Preparing / Ready / Picked up / On the way / Delivered / Cancelled. Use the same keys in OrderCard.tsx, OrderTimeline.tsx, and order/[id]/index.tsx. Style guide preference: 'Pickup' is a noun, 'Pick up' is the verb — status labels are nouns, so 'Picked up' (past tense state)."
    depends_on: null

  - finding_id: TW-002
    surface_id: web-ux-orders-status-labels
    lens: technical-writer
    severity: P0
    issue: "Web customer order status enum exposes 'Picked Up' and 'On the Way' as separate states alongside 'Ready' — but customer-facing wording differs from mobile-customer order detail screen ('Ready for Pickup'). Same order, two different status names across web vs mobile."
    evidence_excerpt: "Pending / Accepted / Preparing / Ready / Picked Up / On the Way / Delivered / Cancelled / Refunded"
    recommendation: "Pick one set of customer status strings and reuse across web + mobile. Recommend: 'Accepted' on web is fine, but mobile uses 'Confirmed' — pick one (style guide doesn't decide; chef-portal uses 'Accept', so 'Accepted' is consistent). Replace 'Picked Up' with sentence-case 'Picked up'."
    depends_on: null

  - finding_id: TW-003
    surface_id: ap-delivery-statuses
    lens: technical-writer
    severity: P0
    issue: "Admin delivery status taxonomy uses 10 states with title-case styling ('At Pickup', 'At Dropoff', 'In Transit') that drift from customer-facing labels ('On the Way') and driver-facing labels ('In Transit'). Admin operators reading 'At Dropoff' cannot quickly map to customer's 'On the Way'."
    evidence_excerpt: "Pending; Assigned; At Pickup; Picked Up; In Transit; At Dropoff; Delivered; Failed; Returned; Cancelled"
    recommendation: "Document a canonical status enum in a single TS module (e.g., shared/types/order-status.ts). Customer labels stay friendly ('On the way'); admin labels stay operational ('In transit'). Use sentence case ('At pickup', 'At dropoff'). Add a mapping table in the audit fix plan."
    depends_on: TW-001

  - finding_id: TW-004
    surface_id: md-core-033
    lens: technical-writer
    severity: P0
    issue: "Driver action label 'Picked Up Order' is past tense used as a future action button. The driver hasn't picked it up yet — they're about to. Confusing for a safety-critical state transition."
    evidence_excerpt: "Picked Up Order  (slide-to-confirm verb, assigned → at_pickup)"
    recommendation: "Use imperative future-tense: 'Confirm pickup' or 'Pick up order'. Past tense ('Picked up') belongs on the status badge after the transition, not on the action that causes it."
    depends_on: null

  - finding_id: TW-005
    surface_id: dp-ux-active-mark-as
    lens: technical-writer
    severity: P0
    issue: "Driver-facing action button reads 'Mark as {Picked Up/In Transit/Delivered}' — telegraphic style says 'Mark delivered', not 'Mark as Delivered'. Title case + 'as' filler breaks driver-glanceability rule (≤4 words, imperative)."
    evidence_excerpt: "Mark as {Picked Up/In Transit/Delivered}"
    recommendation: "Use imperative verbs only: 'Pick up' / 'Start trip' / 'Mark delivered'. Drop 'as'. Sentence case. ≤2 words where possible."
    depends_on: TW-004

  - finding_id: TW-006
    surface_id: dp-ux-delivery-pickup-label
    lens: technical-writer
    severity: P0
    issue: "Driver row labels 'PICKUP' / 'DROPOFF' are ALL CAPS, which the style guide bans for body content. Glanceability does not require ALL CAPS; weight + size handle that. Also drifts from mobile-delivery which uses sentence case 'Pickup' / 'Dropoff'."
    evidence_excerpt: "PICKUP / DROPOFF"
    recommendation: "Sentence case: 'Pickup' / 'Dropoff'. Use type weight/size for emphasis instead of caps."
    depends_on: null

  - finding_id: TW-007
    surface_id: md-core-079
    lens: technical-writer
    severity: P0
    issue: "'Drop-off' (hyphenated) appears alongside 'Dropoff' (unhyphenated) in the same delivery screen set. Same concept, two spellings, both reach drivers."
    evidence_excerpt: "Drop-off  (delivery/[id].tsx)  vs  Dropoff  (active.tsx, dp-ux-delivery-dropoff-label, mobile-delivery elsewhere)"
    recommendation: "Pick one: 'Dropoff' (no hyphen, parallel with 'Pickup' per style guide §3). Replace all 'Drop-off' / 'Drop off' instances except where 'drop off' is used as a verb."
    depends_on: TW-006

  - finding_id: TW-008
    surface_id: dp-ux-active-cancel-prompt
    lens: technical-writer
    severity: P0
    issue: "Driver cancellation reason uses browser `prompt()` ('Reason for cancellation?') — no field structure, no required-reason taxonomy, no friendly copy. Critical for ops because cancellations affect chef and customer."
    evidence_excerpt: "Reason for cancellation?  (browser prompt())"
    recommendation: "Replace with a structured modal: 'Cancel this delivery?' subtitle 'Tell us why — the chef and customer will be notified.' Provide a 4-5 option radio (Customer unreachable / Address wrong / Bike issue / Other) + optional notes."
    depends_on: null

  - finding_id: TW-009
    surface_id: vp-ux-settings-order-acceptance
    lens: technical-writer
    severity: P0
    issue: "Vendor settings auto-accept threshold label reads 'Auto-accept threshold ($)' — dollar symbol shown but currency is INR. Customer-facing money everywhere else uses ₹. Risk: chef sets threshold believing it's $, but the system applies it as ₹."
    evidence_excerpt: "Auto-accept threshold ($) / Orders under this amount will be auto-accepted"
    recommendation: "Replace '$' with '₹'. Add helper text: 'Orders below this amount accept automatically.' Audit for any other $ leaks in chef + admin portals."
    depends_on: null

  - finding_id: TW-010
    surface_id: mv-settings-change-password
    lens: technical-writer
    severity: P0
    issue: "Vendor 'Change Password' setting row routes to the forgot-password flow, not a real password change. Label promises a different action than what happens. Vendor follows a reset link to their email instead of entering current → new password in-app."
    evidence_excerpt: "Change Password  (routes to forgot-password screen)"
    recommendation: "Either: (a) rename to 'Reset password (email link)' to match what actually happens; or (b) build a real change-password screen with current/new fields. Style guide preference: do (b) and keep the label 'Change password'."
    depends_on: null

  - finding_id: TW-011
    surface_id: dp-ux-nav-bottom-items
    lens: technical-writer
    severity: P1
    issue: "Driver bottom-nav 'Home' but sidebar 'Dashboard' — same destination, two labels. Drivers switching between phone and tablet view see different words for the same page."
    evidence_excerpt: "Home (bottom nav)  vs  Dashboard (sidebar nav)"
    recommendation: "Pick one. Recommend 'Dashboard' across both — matches admin + chef portals."
    depends_on: null

  - finding_id: TW-012
    surface_id: dp-ux-nav-partner-items
    lens: technical-writer
    severity: P1
    issue: "Driver nav 'Active Delivery' (sidebar) vs 'Active' (bottom nav) — same destination, different labels. Telegraphic style preferred for nav."
    evidence_excerpt: "Active Delivery (sidebar)  vs  Active (bottom nav)"
    recommendation: "Use 'Active' across both nav surfaces (driver style: ≤4 words, imperative/noun). 'Active delivery' acceptable on the destination page itself."
    depends_on: TW-011

  - finding_id: TW-013
    surface_id: md-core-052
    lens: technical-writer
    severity: P1
    issue: "Driver-mobile More menu exposes admin-only 'Fleet' and 'Staff' nav items to all drivers per the surface inventory (audience: driver). If exposed to regular partners this is access-control drift; if labels are role-gated, label is fine but inventory misclassifies. Either way, ambiguous from copy alone."
    evidence_excerpt: "Fleet / Staff (driver More tab)"
    recommendation: "Confirm role-gating: rename to 'Fleet management' / 'Staff management' when shown to fleet managers, hide for partner drivers entirely. Don't rely on context to disambiguate single-word nav items."
    depends_on: null

  # =====================================================================
  # P1 — Voice / consistency on conversion-critical surfaces
  # =====================================================================

  - finding_id: TW-014
    surface_id: web-ux-cart-checkout-cta
    lens: technical-writer
    severity: P1
    issue: "Checkout CTA 'Sign in to Checkout' uses title-case on 'Checkout' but sentence case on 'Sign in'. The other state, 'Proceed to Checkout', is verbose and also title-cased."
    evidence_excerpt: "Sign in to Checkout / Proceed to Checkout"
    recommendation: "Logged-in: 'Checkout'. Logged-out: 'Sign in to checkout'. Sentence case. Drop 'Proceed to' — wastes a word and adds no info."
    depends_on: null

  - finding_id: TW-015
    surface_id: web-ux-checkout-summary
    lens: technical-writer
    severity: P1
    issue: "Checkout primary CTA 'Place Order' is title case. Style guide §4 requires sentence case on buttons. This is the conversion-critical CTA, so cross-app consistency matters."
    evidence_excerpt: "Place Order"
    recommendation: "'Place order' (sentence case). Apply to web checkout + mobile-customer 'Place Order · ₹{total}' (mc-checkout-place-order)."
    depends_on: null

  - finding_id: TW-016
    surface_id: mc-checkout-place-order
    lens: technical-writer
    severity: P1
    issue: "Mobile checkout CTA 'Place Order · ₹{total}' uses title case and middle-dot separator. CTA microcopy rule: verb-first, sentence case, ≤3 words. Currency in the button is fine but should follow shape rules."
    evidence_excerpt: "Place Order · ₹{total} / Processing..."
    recommendation: "'Place order — ₹{total}'. Sentence case. Em-dash or bullet · acceptable but be consistent across mobile + web. Loading state 'Placing order…' (single Unicode ellipsis)."
    depends_on: TW-015

  - finding_id: TW-017
    surface_id: web-ux-cart-heading
    lens: technical-writer
    severity: P1
    issue: "Cart heading uses 'Your Cart' on web but mobile uses 'Your Cart' on the sheet (mc-cartsheet-title) AND 'View Cart' on the floating bar (mc-cartbar-cta). Style guide prefers descriptive, non-possessive nouns ('Cart' is sufficient — owner is implicit)."
    evidence_excerpt: "Your Cart  (web)  /  Your Cart  (mobile sheet)  /  View Cart  (mobile bar)"
    recommendation: "Use 'Cart' as the heading on both web + mobile. CTA on mobile bar: 'View cart' (sentence case)."
    depends_on: null

  - finding_id: TW-018
    surface_id: web-ux-checkout-payment-section
    lens: technical-writer
    severity: P1
    issue: "Payment section copy 'Pay securely via UPI, cards, net banking, or wallets' uses 'net banking' (two words) but Indian payment UIs commonly write 'Netbanking'. Style guide prefers plain English; 'Net banking' (capitalized) is also common. Need one form."
    evidence_excerpt: "Pay securely via UPI, cards, net banking, or wallets"
    recommendation: "'Pay with UPI, cards, netbanking, or wallets.' Drop 'securely' (claims trust without earning it; trust comes from the Razorpay badge below). Use 'netbanking' one word, lowercase."
    depends_on: null

  - finding_id: TW-019
    surface_id: web-ux-checkout-delivery-time
    lens: technical-writer
    severity: P1
    issue: "'As soon as possible / Usually 30-45 minutes' — 'As soon as possible' is wordy (4 words for a radio label). Style guide §2 customer-facing 10-18 words conversational, but radio labels should be ≤3 words."
    evidence_excerpt: "Delivery Time / As soon as possible / Usually 30-45 minutes / Schedule for later"
    recommendation: "Radio 1: 'ASAP' with helper 'Ready in 30–45 min.' Radio 2: 'Schedule later'. Use en-dash (–) not hyphen in number ranges. Use 'min' not 'minutes' for brevity in the helper line."
    depends_on: null

  - finding_id: TW-020
    surface_id: web-ux-checkout-tip
    lens: technical-writer
    severity: P1
    issue: "Tip helper '100% of your tip goes to the home chef' — the percent sign is fine but the sentence is unnecessarily marketing. Could read as boastful (we're a marketplace; this should be table stakes)."
    evidence_excerpt: "Add a tip / 100% of your tip goes to the home chef / No tip / Custom"
    recommendation: "'Your tip goes directly to the chef.' Drop '100%' (implied). 'No tip' option fine as 'Skip'."
    depends_on: null

  - finding_id: TW-021
    surface_id: mc-catering-form-labels
    lens: technical-writer
    severity: P1
    issue: "Catering date field exposes 'Event Date * (YYYY-MM-DD)' — raw date format hint shown to customer. Style guide §6 routes date formatting through Intl.DateTimeFormat, never hand-rolled. Customer should see a date picker, not be told to type ISO."
    evidence_excerpt: "Event Type * / Event Date * (YYYY-MM-DD) / Guest Count * / Budget (₹) / City * / State * / Additional Details"
    recommendation: "Replace text input with native date picker. Remove '(YYYY-MM-DD)' hint. Drop asterisks from labels — required indicator goes on the field, not the label (style guide §4). Use sentence case: 'Event type', 'Event date', 'Guest count', 'Budget', 'City', 'State', 'Additional details'."
    depends_on: null

  - finding_id: TW-022
    surface_id: web-ux-catering-request-event-fields
    lens: technical-writer
    severity: P1
    issue: "Web catering form uses title case throughout ('Event Date', 'Event Time', 'Number of Guests', 'Minimum budget', 'Maximum budget'). Style guide §4 requires sentence case form labels. Also 'Min' and 'Max' as standalone labels are too cryptic."
    evidence_excerpt: "Event Date / Event Time / Number of Guests / Minimum budget / Maximum budget / Min / Max"
    recommendation: "Sentence case throughout: 'Event date', 'Event time', 'Number of guests', 'Minimum budget', 'Maximum budget'. Replace standalone 'Min'/'Max' with helper text or aria-labels — they're meaningless out of context."
    depends_on: null

  - finding_id: TW-023
    surface_id: web-ux-browse-filters-rating
    lens: technical-writer
    severity: P1
    issue: "Rating filter options 'Any Rating / 4.5+ Stars / 4+ Stars / 3.5+ Stars' use title case. Style guide §4 sentence case."
    evidence_excerpt: "Any Rating / 4.5+ Stars / 4+ Stars / 3.5+ Stars"
    recommendation: "'Any rating', '4.5+ stars', '4+ stars', '3.5+ stars'. Or compact: 'All', '4.5+', '4+', '3.5+' (filter chips don't need 'stars' suffix if a star icon is present)."
    depends_on: null

  - finding_id: TW-024
    surface_id: web-ux-browse-sort-options
    lens: technical-writer
    severity: P1
    issue: "Sort options 'Top Rated / Nearest / Most Popular / Price' title-case + 'Most Popular' is wordy. Drift with mc-home-sort-options ('Recommended, Top Rated, Newest, Price')."
    evidence_excerpt: "Top Rated / Nearest / Most Popular / Price"
    recommendation: "'Top rated', 'Nearest', 'Popular', 'Price'. Align with mobile: pick one canonical sort menu and reuse — currently 'Nearest' (web) vs 'Newest' (mobile) is suspicious — these are different criteria. Verify both apps actually sort the same way for each option."
    depends_on: null

  - finding_id: TW-025
    surface_id: mc-home-sort-options
    lens: technical-writer
    severity: P1
    issue: "Inventory note flags 'two map to same value rating — likely UX bug'. Sort options reduce to fewer distinct sorts than visible options. Either copy lies, or implementation does."
    evidence_excerpt: "Recommended, Top Rated, Newest, Price  (two options map to 'rating')"
    recommendation: "Audit which sort options actually produce different results. Either merge duplicates ('Top rated' = 'Recommended'?) or wire 'Newest' / 'Recommended' to distinct sort criteria. Fix copy after fixing logic."
    depends_on: null

  - finding_id: TW-026
    surface_id: dp-ux-stripe-make-primary
    lens: technical-writer
    severity: P1
    issue: "'Make Stripe My Primary Gateway' is 5 words, title case, possessive ('My'). Style guide §4 buttons: verb-first, ≤3 words, sentence case."
    evidence_excerpt: "Make Stripe My Primary Gateway"
    recommendation: "'Set as primary' (3 words). Or, given the context already says 'gateway': 'Make primary'."
    depends_on: null

  - finding_id: TW-027
    surface_id: dp-ux-stripe-active-gateway
    lens: technical-writer
    severity: P1
    issue: "'Active payout gateway: {Stripe/Razorpay}' uses colon-then-value pattern. Style guide §4 forbids colons on form labels — same principle applies to status rows. Inconsistent with rest of app."
    evidence_excerpt: "Active payout gateway: {Stripe/Razorpay}"
    recommendation: "Two lines: small caption 'Active payout gateway' above bold value '{Stripe/Razorpay}'. Or single line: 'Payouts via {Stripe/Razorpay}.'"
    depends_on: null

  - finding_id: TW-028
    surface_id: dp-ux-staff-form-labels
    lens: technical-writer
    severity: P1
    issue: "Staff invite labels use 'Email *', 'Role *' with asterisks in label text and email placeholder 'staff@example.com'. Style guide §4 asterisk goes on the field, not on the label."
    evidence_excerpt: "Email * + staff@example.com / Role * / Department + e.g. Operations / Title + e.g. Fleet Coordinator"
    recommendation: "Drop asterisks from labels. Add required indicator via UI (red dot or '*' next to the field, not in the text). Replace 'e.g.' helper with explicit placeholder or helper-text pattern."
    depends_on: null

  - finding_id: TW-029
    surface_id: mc-catering-status-labels
    lens: technical-writer
    severity: P1
    issue: "Catering request status pills 'Open / Quoted / Accepted / Completed / Cancelled' have no observable drift WITHIN this surface, but compare to web catering surfaces which use full nouns like 'No open requests' / 'No pending quotes' / 'No booked events' — 'Booked' vs 'Accepted' is the same state in two words."
    evidence_excerpt: "Open / Quoted / Accepted / Completed / Cancelled  (mobile)  vs  open / pending quotes / booked  (web chef)"
    recommendation: "Sync vocabulary: 'open', 'quoted', 'booked', 'completed', 'cancelled'. 'Booked' is more natural than 'Accepted' for catering context (events are booked, not orders)."
    depends_on: null

  - finding_id: TW-030
    surface_id: web-ux-chefdetail-status-badges
    lens: technical-writer
    severity: P1
    issue: "Chef detail page mixes 'Open for orders' (good) with 'Currently closed' (filler word 'Currently') and abbreviated stats 'Min. order:' / 'Delivery:' / 'Price range:'. Trailing colons violate §4 form-label rule; 'Min.' abbreviation reads as typographic clutter."
    evidence_excerpt: "Verified / Open for orders / Currently closed / Min. order: / Delivery: / Price range:"
    recommendation: "'Open' / 'Closed' for badges (one word). For stats: drop the colon, use caption-then-value layout. 'Min. order' → 'Minimum order'. 'Delivery' → 'Delivery fee'. 'Price range' stays."
    depends_on: null

  # =====================================================================
  # P2 patterns — grouped findings across many surfaces
  # =====================================================================

  - finding_id: TW-031
    surface_id: pattern-title-case-headings-web
    lens: technical-writer
    severity: P2
    issue: "PATTERN — Web app uses title case on most page H1s and section headings; style guide §4 sentence case for microcopy is silent on H1s but design context (.impeccable.md, Source Sans 3 UI body) reads cleaner in sentence case. Title case headings are AI-slop tell."
    evidence_excerpt: "Affected surface_ids (web): web-ux-orders-heading 'My Orders'; web-ux-favorites-heading 'My Favorite Chefs'; web-ux-profile-heading 'Account Settings'; web-ux-profile-personal-info 'Personal Information'; web-ux-profile-payments 'Payment Methods'; web-ux-profile-2fa 'Two-Factor Authentication' / 'Set Up Two-Factor Authentication' / 'Save Your Backup Codes'; web-ux-profile-preferences 'Food Preferences' / 'Dietary Preferences' / 'Food Allergies' / 'Favourite Cuisines' / 'Spice Tolerance' / 'Household Size'; web-ux-profile-addresses 'Saved Addresses'; web-ux-checkout-address-form 'Delivery Address' / 'Add New' / 'Street Address' / 'Postal Code'; web-ux-orderdetail-sections 'Back to Orders' / 'Order Items' / 'Delivery Address' / 'Payment Summary' / 'Special Instructions' / 'Leave a Review'; web-ux-cart-order-summary 'Order Summary' / 'Delivery fee' / 'Service fee'; web-ux-chefdetail-reviews 'Customer Reviews'; web-ux-chefdetail-categories 'Categories' / 'All Items' / 'Reviews'; web-ux-admin-analytics 'Revenue Overview' / 'Orders by Status' / 'Top Performing Chefs' / 'Popular Cuisines'; web-ux-admin-chefs 'Chef Management'; web-ux-admin-orders 'Order' / 'Delivery Address'; web-ux-admin-settings 'Platform Settings' / 'Danger Zone'; web-ux-chef-menu 'Menu Management'; web-ux-chef-orders-heading 'Delivery Information' / 'Order Items' / 'Special Instructions' / 'Payment Summary'; web-ux-chef-profile 'Kitchen Profile' / 'Profile Images' / 'Basic Information' / 'Cuisines' / 'Specialties' / 'Business Settings' / 'Your Stats'; web-ux-chef-social 'Social Feed'; web-ux-chef-earnings 'Earnings Overview' / 'Top Selling Items' / 'Recent Payouts' / 'Payment Settings'."
    recommendation: "One pass: switch all H1s and section headings to sentence case across apps/web/src/features/**. Keep proper nouns capitalized (Razorpay, UPI, etc.). Preserves brand 'quietly modern' aesthetic, satisfies §4 microcopy formula by default."
    depends_on: null

  - finding_id: TW-032
    surface_id: pattern-title-case-headings-admin-portal
    lens: technical-writer
    severity: P2
    issue: "PATTERN — Admin portal uses title case for all section headings; style guide allows admin neutrality but the §4 microcopy formula and overall brand direction in .impeccable.md prefer sentence case."
    evidence_excerpt: "Affected: ap-approvaldetail-sections 'Chef Information' / 'Submitted Data' / 'Documents' / 'Admin Notes'; ap-approvaldetail-detail-fields many ('Request ID' / 'Business Name' / 'Kitchen Type' / 'Document Number' / 'Item Name' / 'Compare Price' / 'Portion Size' / 'Dietary Tags' / 'Operating Hours' / 'Terms Accepted' / 'Hygiene Policy Accepted'); ap-approvals-statuses 'All Statuses' / 'Info Requested'; ap-approvals-table-columns 'Type' / 'Title' / 'Requester' / 'Priority' / 'Submitted' / 'Actions'; ap-chefs-stat-labels 'Menu Items' / 'Prep Time'; ap-chefs-filter-status 'All Status' / 'Pending Verification'; ap-dashboard-stat-labels 'Total users' / 'Active chefs' / 'Orders today' / 'Total revenue' (already sentence — keep); ap-delivery-statuses (see TW-003); ap-delivery-table-columns 'Order' / 'Driver' / 'From / To' / 'Status' / 'Payout' / 'Time'; ap-providercreate-sections 'Basic Info' / 'API Configuration' / 'Status Mapping' / 'Coverage' / 'Pricing' / 'Rate Limits' / 'Contact'; ap-providercreate-fields 'API Base URL' / 'Webhook Secret' / 'Supported Countries' / 'Max Distance (km)' / 'Pricing Model' / 'Base Cost' / 'Per KM Cost' / 'Max Concurrent Deliveries' / 'Daily Limit' / 'Avg Pickup Time (min)' / 'Contact Name'; ap-providerdetail-stats 'Total Deliveries' / 'Success Rate' / 'Avg Delivery Time' / 'Active Deliveries'; ap-providerdetail-sections 'API Configuration' / 'Rate Limits & Config' / 'Performance (Last 30 Days)' / 'Contact Info'; ap-providers-table-columns 'API Status' / 'Success Rate'; ap-orders-filter-status 'All Status'; ap-orders-table-columns; ap-auditlogs-table 'When' / 'Actor' / 'Action' / 'Entity' / 'IP'; ap-exports-title 'Data & Exports'; ap-exports-range-card 'Date range' (good); ap-secsettings-cards 'Password policy' / 'Session management' / 'Two-factor authentication' / 'API keys' (already sentence — keep); ap-secsettings-password-fields 'Minimum length' / 'Require uppercase letter' (already sentence — keep); ap-secsettings-session-fields 'Access token TTL (hours)' / 'Refresh token TTL (days)' / 'Save policy' / 'Your active sessions' (mostly sentence); ap-secsettings-2fa-account; ap-settings-card-titles 'Payment Gateway' / 'Stripe Gateway'; ap-settings-payment-fields 'Razorpay Key ID' / 'Stripe Secret Key' / 'Webhook Signing Secret' / 'Key Prefix' (title case); ap-staffdetail-section-profile 'Profile Information'; ap-staffdetail-info-rows 'Email' / 'Phone' / 'Department' / 'Title' / 'Joined' / 'Last Login'; ap-staffdetail-section-roleperm 'Role & Permissions'; ap-staffdetail-section-system 'System Details'; ap-staff-tabs 'Staff Members'; ap-staff-table-columns 'Staff Member' / 'Last Login'; ap-staff-invite-fields 'Personal Message'; ap-userdetail-info-rows 'Auth Provider' / 'Last Login'; ap-userdetail-system 'System Details' / 'User ID' / 'Created At' / 'Updated At'; ap-userdetail-order-stats 'Order Activity' / 'Total Orders' / 'Total Spent' / 'Last Order'; ap-userdetail-verification 'Account Active' / 'Not verified'; ap-users-filter-roles 'All Roles' / 'Customer' / 'Chef' / 'Delivery' / 'Admin'; ap-users-table-columns 'User' / 'Contact' / 'Role' / 'Orders' / 'Total Spent' / 'Joined' / 'Actions'."
    recommendation: "Sweep apps/admin-portal/src/features/**: section headings, table column headers, filter labels, info-row labels — all sentence case. Keep acronyms (API, IP, TTL, IFSC, UPI, CSAT, SMS). Keep proper nouns. Keep 'ID' as 'ID' in 'User ID' / 'Request ID' (uppercase initialism is acceptable)."
    depends_on: TW-031

  - finding_id: TW-033
    surface_id: pattern-title-case-headings-delivery-portal
    lens: technical-writer
    severity: P2
    issue: "PATTERN — Delivery portal (web) uses title case for stat labels, section headers, and CTAs. Per persona tone matrix admin gets neutral precise voice — sentence case is the production-grade default."
    evidence_excerpt: "Affected: dp-ux-active-distance-label 'Distance'; dp-ux-active-est-time-label 'Est. Time'; dp-ux-active-payout-label 'Payout'; dp-ux-active-status-* 'Picked Up' / 'In Transit' / 'Delivered'; dp-ux-active-h1 'Active Delivery'; dp-ux-available-h1 'Available Deliveries'; dp-ux-accept-delivery-btn 'Accept Delivery'; dp-ux-history-h1 'Delivery History'; dp-ux-earnings-total 'Total Earnings'; dp-ux-earnings-daily-breakdown 'Daily Breakdown'; dp-ux-fleet-partner-summary 'Partner Summary'; dp-ux-fleet-today-earnings \"Today's Fleet Earnings\"; dp-ux-fleet-quick-pending 'Pending Verifications'; dp-ux-fleet-livestats 'Online Partners' / 'Active Deliveries' / 'Unassigned Orders' / \"Today's Completed\"; dp-ux-fleet-partnerstats 'Total Partners' / 'Verified' / 'Pending Verification'; dp-ux-fleet-h 'Fleet Overview'; dp-ux-fleet-view-partners 'View Partners'; dp-ux-partner-detail-back 'Back to Partners'; dp-ux-partner-performance 'Acceptance Rate' / 'On-Time Rate' / 'CSAT Score' / 'Total Deliveries'; dp-ux-partner-detail-verify-block 'Pending Verification'; dp-ux-partner-docs-h 'Documents'; dp-ux-partner-active-deliveries 'Active Deliveries'; dp-ux-partner-suspend 'Suspend Partner'; dp-ux-partner-reactivate 'Reactivate Partner'; dp-ux-partners-card-labels 'Vehicle' / 'Rating' / 'Deliveries' (single nouns OK); dp-ux-partners-h 'Delivery Partners'; dp-ux-stripe-title 'Stripe (International Payouts)'; dp-ux-stripe-resume-onboarding 'Resume Onboarding'; dp-ux-stripe-switch-razorpay 'Switch to Razorpay'; dp-ux-settings-privacy 'Privacy & Security'; dp-ux-settings-signout 'Sign Out'; dp-ux-staff-h 'Staff Management'; dp-ux-staff-invite-btn 'Invite Staff'; dp-ux-staff-invite-h 'Send Staff Invitation'; dp-ux-staff-invite-created 'Invitation Created'; dp-ux-staff-list-h 'Current Staff'; dp-ux-staff-pending-h 'Pending Invitations'."
    recommendation: "Sentence case across apps/delivery-portal/src/features/**. Driver-facing telegraphic style favours short sentence-case labels. Note: 'Sign Out' → 'Sign out' (also fixes style guide §3 'Sign out' preferred over 'Logout')."
    depends_on: TW-031

  - finding_id: TW-034
    surface_id: pattern-title-case-headings-vendor-portal
    lens: technical-writer
    severity: P2
    issue: "PATTERN — Vendor portal title-cases section headings, card titles, and stat labels. Chef-facing tone matrix is 'crisp 5-12 words operational' — sentence case is the cleaner default."
    evidence_excerpt: "Affected: vp-ux-dashboard-revenue-lead \"Today's revenue\" (already sentence — keep); vp-ux-dashboard-pending-cta 'orders' / 'Waiting for you to accept' (good); vp-ux-dashboard-chart 'Weekly Revenue' / 'Last 7 days performance'; vp-ux-dashboard-recent-orders 'Recent Orders' / 'Your latest order activity' / 'View all' / 'Order' / 'Customer' / 'Items' / 'Amount' / 'Status' / 'Time'; vp-ux-dashboard-pending-card 'Pending Orders' / 'New' / 'Accept' / 'Reject'; vp-ux-dashboard-quick-actions 'Quick Actions' / 'Add Menu Item' / 'Create a new dish listing' / 'View Orders' / 'Manage live orders' / 'Update Profile' / 'Edit kitchen details' / 'View Earnings' / 'Track your revenue'; vp-ux-earnings-cards 'Available Balance' / 'Pending Payout' / 'This Month' / 'Lifetime Earnings'; vp-ux-payouts-title 'Payout History' / 'Back to Earnings'; vp-ux-menu-form-title 'Edit Menu Item' / 'Add Menu Item'; vp-ux-menu-view-page 'Menu item not found' (good) / 'Back to Menu' / 'Under Review' / 'Edit Item' / 'Dietary Information' / 'No dietary information provided' / 'Pricing' / 'Preparation' / 'Prep Time' / 'Serves' / 'Portion Size' / 'Performance' / 'Orders' / 'Rating' / 'Item ID'; vp-ux-menu-title 'Menu Management'; vp-ux-menu-bulk-actions 'Mark Available' / 'Mark Unavailable' / 'Delete Selected'; vp-ux-notifs-title 'Admin Requests' / 'Mark all read' (good); vp-ux-orders-live-title 'Live Orders'; vp-ux-orders-history-title 'Order History' / 'Live Orders'; vp-ux-orders-history-summary 'Delivered' / 'Cancelled' / 'Revenue' (good); vp-ux-kitchen-title 'Kitchen Setup' / 'Save Changes' / 'Back to Profile'; vp-ux-kitchen-hours 'Operating Hours' / 'Closed'; vp-ux-kitchen-photos 'Kitchen Photos' / 'Add Photo'; vp-ux-profile-title 'Kitchen Profile' / 'Save Changes'; vp-ux-profile-business-info 'Business Information' / 'Business Name' / 'Description'; vp-ux-profile-cuisines 'Cuisines'; vp-ux-profile-specialties 'Specialties' / 'No specialties added yet'; vp-ux-profile-settings 'Business Settings' / 'Average Prep Time' / 'Delivery Radius (km)' / 'Minimum Order' / 'Delivery Fee'; vp-ux-profile-docs 'Documents & Certificates'; vp-ux-reviews-title 'Reviews'; vp-ux-settings-title 'Settings' / 'Save Settings'; vp-ux-settings-order-acceptance 'Order Acceptance' / 'Accepting Orders' / 'Auto-accept Orders' / 'Auto-accept threshold' (good); vp-ux-settings-notifications 'Push Notifications' / 'New order alerts' / 'Order updates' / 'Daily summary' / 'Weekly report' / 'New order SMS'; vp-ux-settings-payout-section 'Payout Details' / 'Razorpay Connected' / 'Linked Account' / 'Set Up Payouts' / 'Account Holder' / 'Account Number' / 'IFSC Code' / 'UPI ID'; vp-ux-settings-password 'Change Password' / 'Current Password' / 'New Password' / 'Confirm New Password' / 'Update Password'."
    recommendation: "Sentence case throughout apps/vendor-portal/src/features/**. Special cases: 'Razorpay Connected' → 'Razorpay connected'. 'IFSC Code' → 'IFSC code'. 'UPI ID' → 'UPI ID' (acronym stays caps)."
    depends_on: TW-031

  - finding_id: TW-035
    surface_id: pattern-title-case-headings-mobile-customer
    lens: technical-writer
    severity: P2
    issue: "PATTERN — Mobile customer screen titles use title case. Style guide §4 sentence case is the default; mobile follows web."
    evidence_excerpt: "Affected: mc-favorites-title 'Saved Chefs' (also bottom-tab is 'Saved' — drift noted); mc-orders-title 'My Orders'; mc-profile-sections 'Personal Info' / 'Food Preferences'; mc-profile-field-labels 'First Name' / 'Last Name' / 'Phone'; mc-profile-save-ctas 'Save Changes' / 'Save Preferences'; mc-catering-tabs 'Request Catering' / 'My Requests'; mc-catering-submit-cta 'Submit Request' (drift: web uses 'Place Order' for the verbal cousin; catering uses 'Submit'); mc-checkout-section-address 'Delivery Address'; mc-checkout-add-address 'Add New Address'; mc-checkout-save-address 'Save Address'; mc-checkout-summary-section 'Order Summary'; mc-order-detail-sections 'Items' / 'Delivery Address' / 'Price Breakdown'; mc-order-detail-price-rows 'Subtotal' / 'Delivery Fee' / 'Total' (drift with checkout's 'Delivery fee' lower-f — inconsistency, see TW-049); mc-order-detail-track-cta 'Track Order'; mc-social-title 'Social Feed'; mc-cartbar-cta 'View Cart'; mc-cartsheet-title 'Your Cart'; mc-cartsheet-checkout-cta 'Proceed to Checkout'."
    recommendation: "Sentence case sweep across apps/mobile-customer/app/**. Special: 'Saved Chefs' page heading should match bottom-tab — pick 'Saved'."
    depends_on: TW-031

  - finding_id: TW-036
    surface_id: pattern-title-case-headings-mobile-vendor
    lens: technical-writer
    severity: P2
    issue: "PATTERN — Mobile vendor uses title case section headings, stats, field labels, CTAs. Sentence case throughout matches chef-facing tone and reduces visual noise."
    evidence_excerpt: "Affected: mv-dash-stats-today-orders \"Today's Orders\"; mv-dash-stats-today-earnings \"Today's Earnings\"; mv-dash-accepting-title 'Accepting Orders'; mv-dash-recent-orders 'Recent Orders'; mv-menu-heading 'My Menu'; mv-more-account-title 'My Account'; mv-orders-tab-live 'Live Queue'; mv-analytics-total-orders 'Total Orders'; mv-analytics-total-revenue 'Total Revenue' (drift with mv-earnings-total 'Total Earnings'); mv-analytics-popular 'Popular Items'; mv-analytics-revenue-trend 'Revenue Trend'; mv-earnings-total 'Total Earnings'; mv-earnings-pending-payout 'Pending Payout'; mv-earnings-last-payout 'Last Payout'; mv-earnings-payout-account 'Payout Account'; mv-earnings-payout-fields 'Bank' / 'Account Number' / 'IFSC' / 'UPI ID'; mv-earnings-weekly 'Weekly Earnings'; mv-menuedit-heading 'Edit Menu Item'; mv-menuedit-photos-section 'Photos'; mv-menuedit-save-cta 'Save Changes'; mv-menunew-heading 'Add Menu Item'; mv-menunew-photo-section 'Food Photo'; mv-menunew-photo-empty 'No photo selected' (good); mv-menunew-photo-take 'Take Photo'; mv-menunew-section-details 'Item Details'; mv-menunew-name-label 'Item Name *'; mv-menunew-price-label 'Price (₹) *'; mv-menunew-preptime-label 'Preparation Time'; mv-menunew-cta 'Add Item'; mv-profile-section-personal 'Personal Info'; mv-profile-field-display-name 'Display Name'; mv-profile-field-kitchen-name 'Kitchen Name'; mv-profile-field-cuisine-types 'Cuisine Types'; mv-profile-section-photos 'Kitchen Photos'; mv-reviewdetail-heading 'Reply to Review'; mv-reviewdetail-form-label 'Your Reply'; mv-reviewdetail-send-cta 'Send Reply'; mv-reviews-heading 'Customer Reviews'; mv-reviewcard-your-reply 'Your Reply'; mv-settings-section-notifs 'Notification Preferences'; mv-settings-notif-neworders 'New Order Notifications'; mv-settings-notif-payouts 'Payout Notifications'; mv-settings-notif-reviews 'Review Notifications'; mv-settings-change-password 'Change Password'; mv-settings-delete-account 'Delete Account'."
    recommendation: "Sweep apps/mobile-vendor/app/**. Sentence case throughout. Note: 'Total Revenue' / 'Total Earnings' / 'Pending Payout' all live as KPI tiles — pick one verb ('Earnings' or 'Revenue') across all chef screens (see TW-051)."
    depends_on: TW-031

  - finding_id: TW-037
    surface_id: pattern-title-case-headings-mobile-delivery
    lens: technical-writer
    severity: P2
    issue: "PATTERN — Mobile delivery screen titles and status badges use title case. Driver tone matrix accepts telegraphic, but sentence case still applies to multi-word labels."
    evidence_excerpt: "Affected: md-core-005 'Dashboard' (one word, fine); md-core-006 'You are Online' (capital O on Online); md-core-007 'You are Offline'; md-core-014 'Total Deliveries'; md-core-015 'Available'; md-core-016 \"You're Offline\"; md-core-018 'Go Online to Accept Deliveries'; md-core-023 'Active Delivery'; md-core-025 'Head to Kitchen'; md-core-026 'Waiting for Order'; md-core-027 'Order Picked Up'; md-core-028 'On the Way' (good — sentence already); md-core-029 'At Dropoff Location'; md-core-030 'Delivery Complete'; md-core-031 'Delivery Cancelled'; md-core-032 'Arrived at Kitchen'; md-core-033 'Picked Up Order' (see TW-004); md-core-034 'Start Delivery'; md-core-035 'Arrived at Dropoff'; md-core-036 'Mark as Delivered'; md-core-040 'Pickup Location'; md-core-041 'Dropoff Location'; md-core-044 'Order Summary'; md-core-046 'Special instructions:' (good apart from colon); md-core-049-054 nav items single nouns; md-core-056 'Notifications'; md-core-057 'New Delivery Notifications'; md-core-059 'Earnings Payout Notifications'; md-core-062 'Default Online Status'; md-core-065 'Change Password'; md-core-066 'View Subscription Plan'; md-core-067 'Delete Account'; md-core-070 '{Period} Earnings'; md-core-071 'Total Earned'; md-core-072 'Pending Payout'; md-core-073 'Last Payout'; md-core-074 'Weekly History'; md-core-075 'Delivery History'; md-core-076 'Delivery Detail'; md-core-080 'Distance' / 'Completed' / 'Payout' (single nouns, fine); md-core-082 'Personal Info'; md-core-083 'Full Name'; md-core-085 'Vehicle Details'; md-core-086 'Vehicle Type'; md-core-087 'Registration Number'; md-core-089 'Save Changes'; md-core-091 '✓ Verified' / 'Pending' (check sym OK but inconsistent — flag separately); md-core-092 'Fleet Overview'; md-core-093 'Total Drivers'; md-core-094 'Online Now'; md-core-095 \"Today's Deliveries\"; md-core-096 \"Today's Earnings\"; md-core-100 'Partner Detail'; md-core-103 'Invite Staff Member'; md-core-108 'Send Invitation'."
    recommendation: "Sweep apps/mobile-delivery/app/**. Sentence case multi-word labels. Status badges already sentence case ('On the way') in some places — apply consistently."
    depends_on: TW-031

  - finding_id: TW-038
    surface_id: pattern-asterisk-on-form-labels
    lens: technical-writer
    severity: P2
    issue: "PATTERN — Required-field asterisks embedded in label text across multiple forms. Style guide §4: required indicator on the field, label stays clean."
    evidence_excerpt: "Affected: mc-catering-form-labels 'Event Type *' / 'Event Date *' / 'Guest Count *' / 'City *' / 'State *'; mv-menunew-name-label 'Item Name *'; mv-menunew-price-label 'Price (₹) *'; dp-ux-staff-form-labels 'Email *' / 'Role *'."
    recommendation: "Drop asterisks from label strings. Implement required indicator via UI (red dot, '*' marker outside label text, or aria-required attribute styled). Update all flagged surfaces."
    depends_on: null

  - finding_id: TW-039
    surface_id: pattern-trailing-colon-form-labels
    lens: technical-writer
    severity: P2
    issue: "PATTERN — Trailing colons on labels and section captions. Style guide §4 form labels: no colons."
    evidence_excerpt: "Affected: web-ux-chefdetail-status-badges 'Min. order:' / 'Delivery:' / 'Price range:'; web-ux-chefdetail-reviews \"Chef's Response:\"; web-ux-orderdetail-sections 'Placed on {date}' / 'Estimated delivery:'; md-core-042 'Note:'; md-core-046 'Special instructions:'; dp-ux-stripe-active-gateway 'Active payout gateway:'; dp-ux-staff-invite-created 'Share this link with the new staff member:'; vp-ux-settings-payout-section 'Linked Account: ...'."
    recommendation: "Remove all trailing colons from label-style strings. Use layout (caption above value, line break) to convey label/value relationship."
    depends_on: null

  - finding_id: TW-040
    surface_id: pattern-loading-ellipsis-three-dots
    lens: technical-writer
    severity: P2
    issue: "PATTERN — Loading states use three dots '...' instead of Unicode ellipsis '…'. Typographic polish; consistency with brand 'quietly modern'."
    evidence_excerpt: "Affected: web-ux-browse-search-input 'Search chefs, dishes, cuisines...'; web-ux-admin-chefs 'Search chefs...'; web-ux-admin-orders 'Search order number...'; web-ux-admin-users 'Search by name or email...'; web-ux-chef-menu 'Search menu items...'; web-ux-chef-orders-heading 'Search by order number...'; web-ux-delivery-orders 'Search orders...'; web-ux-social-feed-search 'Search posts, chefs, or dishes...'; web-ux-chef-catering 'Any special notes about your quote...'; web-ux-chef-profile 'Tell customers about your kitchen...'; web-ux-chef-social 'Share something about this dish...'; vp-ux-kitchen-photos 'Uploading...'; vp-ux-profile-business-info 'Tell customers about your kitchen, cooking style, and what makes your food special...'; vp-ux-profile-specialties 'Add a specialty...'; dp-ux-staff-form-labels 'Add a personal message to the invitation email...'; dp-ux-staff-send-btn 'Sending...'; dp-ux-stripe-connect-btn 'Starting…' (correct already — use as model); dp-ux-partner-verify-approve 'Verifying...'; dp-ux-partner-reactivate 'Processing...'; mc-checkout-place-order 'Processing...'; mv-reviewdetail-input-ph 'Write a thoughtful reply to this review...'."
    recommendation: "Replace '...' with '…' (Unicode U+2026) in all placeholder strings, loading states, and 'Tell us more' textarea placeholders. dp-ux-stripe-connect-btn 'Starting…' already uses correct ellipsis — propagate that style."
    depends_on: null

  - finding_id: TW-041
    surface_id: pattern-multi-word-buttons
    lens: technical-writer
    severity: P2
    issue: "PATTERN — Button labels exceed style guide §4 cap of 3 words."
    evidence_excerpt: "Affected (>3 words): web-ux-cart-checkout-cta 'Sign in to Checkout' (4); web-ux-cart-checkout-cta 'Proceed to Checkout' (3 — borderline); mc-cartsheet-checkout-cta 'Proceed to Checkout' (3 — borderline, prefer 'Checkout'); mc-checkout-add-address 'Add New Address' (3 — borderline, prefer 'Add address'); md-core-018 'Go Online to Accept Deliveries' (5 — should be 'Go online'); dp-ux-stripe-make-primary 'Make Stripe My Primary Gateway' (5 — see TW-026); dp-ux-staff-invite-h 'Send Staff Invitation' (3 — borderline); ap-staff-invite-dialog-title 'Invite Staff Member' (3 — borderline); md-core-066 'View Subscription Plan' (3 — borderline); md-core-036 'Mark as Delivered' (3 — borderline, prefer 'Mark delivered'); web-ux-orderdetail-cancel 'Please let us know why you're cancelling...' (8 — modal subtitle, not button, but borderline informal); mv-onb-review-cta 'Submit Application' (2 — fine)."
    recommendation: "Tighten to ≤3 words: 'Sign in' / 'Checkout' / 'Add address' / 'Go online' / 'Set as primary' / 'Send invite' / 'Invite staff' / 'View plan' / 'Mark delivered'."
    depends_on: null

  - finding_id: TW-042
    surface_id: pattern-uppercase-eyebrows
    lens: technical-writer
    severity: P2
    issue: "PATTERN — ALL CAPS or shouting eyebrows on driver/customer surfaces. Style guide §1/§3 bans 'ALL CAPS body'. Eyebrows can use small-caps via CSS letter-spacing instead."
    evidence_excerpt: "Affected: dp-ux-active-pickup-from 'Pickup from' (note: 'uppercase eyebrow' style — CSS, not string); dp-ux-delivery-pickup-label 'PICKUP'; dp-ux-delivery-dropoff-label 'DROPOFF'; mv-undo-cta 'UNDO' (snackbar action — marginal, but inconsistent with mobile-customer)."
    recommendation: "Keep strings in sentence case ('Pickup', 'Dropoff', 'Undo'). Apply uppercase via CSS text-transform if visual eyebrow style is desired — strings stay translation-safe."
    depends_on: TW-006

  - finding_id: TW-043
    surface_id: pattern-banned-vocabulary-logout
    lens: technical-writer
    severity: P2
    issue: "PATTERN — 'Logout' (one word) and 'Log out' (two words) used across apps. Style guide §3: 'Sign out' is the preferred term; 'Log out' / 'Logout' banned."
    evidence_excerpt: "Affected: web-ux-admin-layout-nav 'Log out'; web-ux-layout-nav 'Logout'; vp-ux-layout-nav 'Logout'; mv-onb-pending-logout-btn 'Logout'; mv-more-logout-row 'Logout'; mc-profile-logout-button 'Log Out'; dp-ux-settings-signout 'Sign Out' (closer — needs sentence case)."
    recommendation: "Replace all 'Logout' / 'Log out' / 'Sign Out' instances with 'Sign out'. Singular term across all 5 apps."
    depends_on: null

  - finding_id: TW-044
    surface_id: pattern-banned-vocabulary-login-signup
    lens: technical-writer
    severity: P3
    issue: "PATTERN — Within core-ux surfaces, 'login' / 'Login' appears as a noun. Style guide §3: 'Sign in' verb form is preferred even when used as a noun ('Use your sign-in details')."
    evidence_excerpt: "Affected: vp-ux-settings-password 'Your account is linked to {provider} login. Password management is handled by your social login provider.'"
    recommendation: "Rewrite: 'Your {provider} account manages your password — open {provider} to change it.' Drop 'login' noun usage."
    depends_on: TW-043

  - finding_id: TW-045
    surface_id: pattern-customer-noun-user
    lens: technical-writer
    severity: P2
    issue: "PATTERN — Admin / chef-facing surfaces refer to customers as 'User' / 'Users'. Style guide §3: 'Customer' is the chef/admin-facing preferred term; 'User' is banned (too generic)."
    evidence_excerpt: "Affected (admin/chef-facing 'User' references): web-ux-admin-users 'Users / Search by name or email... / No users found'; ap-userdetail-system 'User ID'; ap-users-title 'Users'; ap-users-subtitle 'Manage all registered users on the platform'; ap-users-table-columns 'User'; ap-dashboard-stat-labels 'Total users'; ap-analytics-cards 'Total Users'; ap-userdetail-* sections; ap-exports-cards 'User data / All user accounts (id, email, role, created_at)'."
    recommendation: "When the entity is a customer specifically, use 'Customer'. When the entity could be any role (chef, customer, driver, admin), keep 'User' for system-level admin views (User ID is fine; user accounts is fine). Audit which 'Users' surfaces are actually 'Customers'. Recommend: ap-users-title stays 'Users' (mixed roles per ap-users-filter-roles), but ap-userdetail customer-only views become 'Customer Details'."
    depends_on: null

  - finding_id: TW-046
    surface_id: pattern-banned-vocabulary-driver-vs-delivery-partner
    lens: technical-writer
    severity: P2
    issue: "PATTERN — Customer/admin surfaces alternate 'Driver' and 'Delivery Partner' / 'Partner'. Style guide §3: 'Driver' driver-facing, 'Delivery partner' customer-facing."
    evidence_excerpt: "Affected: dp-ux-profile-fallback-name 'Delivery Partner' (driver-facing — should be 'Driver'); dp-ux-partners-h 'Delivery Partners' (admin-facing 'Partners' is fine, but cards call them 'Partners' below — drift); dp-ux-partner-detail-verify-block 'This partner is awaiting verification' (admin-facing — OK); dp-ux-partner-readonly-notice 'fleet manager to approve' (mention of role — fine); mc-checkout-totals (no mention); ap-users-filter-roles 'Delivery' (one word — confusing alone)."
    recommendation: "Driver-facing surfaces (delivery-portal, mobile-delivery): 'Driver'. Admin-facing surfaces (admin-portal, delivery-portal staff views): 'Driver' or 'Partner' but pick one — recommend 'Driver' across admin. Customer-facing: 'Delivery partner'. Rename dp-ux-profile-fallback-name to 'Driver'. Rename ap-users-filter-roles 'Delivery' to 'Driver'."
    depends_on: TW-013

  - finding_id: TW-047
    surface_id: pattern-money-formatting
    lens: technical-writer
    severity: P2
    issue: "PATTERN — Money symbol usage drifts. Style guide §6: '₹120' (no space, symbol first). Most surfaces follow, but inventory shows '$' leak (TW-009), and 'Min. order: ₹{n}' (TW-030) ends in colon."
    evidence_excerpt: "Affected: vp-ux-settings-order-acceptance 'Auto-accept threshold ($)' — see TW-009; ap-chefs-meta-bottom 'Min order: ₹{n}'; web-ux-checkout-summary 'Subtotal / Delivery fee / Service fee / Tax / Tip / Total'; mv-menunew-price-label 'Price (₹) *' (currency in parens fine, asterisk separate issue); mc-checkout-place-order 'Place Order · ₹{total}'; md-core-047 'You earned ₹{payout} for this delivery' (good)."
    recommendation: "Standardise to '₹{amount}' no space, no 'Rs.', no '$' anywhere in INR contexts. Format via Intl.NumberFormat({ style: 'currency', currency: 'INR' })."
    depends_on: TW-009

  - finding_id: TW-048
    surface_id: pattern-prep-time-unit-formatting
    lens: technical-writer
    severity: P2
    issue: "PATTERN — Prep time unit drifts: '15min' (no space) vs '15 min' (with space) for the same data point across vendor onboarding and menu creation."
    evidence_excerpt: "Affected: mv-onb-ops-preptime-options '15min / 30min / 45min / 60min / 90min' (no space); mv-menunew-preptime-unit '{n} min' (with space)."
    recommendation: "Standardise on '{n} min' (space + abbreviation). Replace '15min' chips with '15 min'. Numeric + unit conforms to Intl-style number-unit formatting."
    depends_on: null

  - finding_id: TW-049
    surface_id: pattern-row-label-case-drift-checkout-orderdetail
    lens: technical-writer
    severity: P2
    issue: "PATTERN — 'Delivery fee' (sentence) on checkout drifts to 'Delivery Fee' (title) on order detail screen of the same app. Same line item, two cases."
    evidence_excerpt: "Affected: mc-checkout-totals 'Subtotal / Delivery fee / Free / Total'; mc-order-detail-price-rows 'Subtotal / Delivery Fee / Total'; web-ux-cart-order-summary 'Order Summary / Subtotal / Delivery fee / Service fee / Discount / Total'; web-ux-checkout-summary 'Subtotal / Delivery fee / Service fee / Tax / Tip / Total / Place Order'."
    recommendation: "All price-row labels sentence case: 'Subtotal', 'Delivery fee', 'Service fee', 'Discount', 'Tax', 'Tip', 'Total'. Update mc-order-detail-price-rows to match."
    depends_on: TW-031

  - finding_id: TW-050
    supports_revenue_drift: true
    surface_id: pattern-stat-card-label-drift-chef
    lens: technical-writer
    severity: P2
    issue: "PATTERN — Chef KPI tile labels drift: 'Today's Earnings' (mobile dash) vs 'Total Revenue' (mobile analytics) vs 'Total Earnings' (mobile earnings) vs 'Available Balance' / 'Lifetime Earnings' (vendor portal). Same chef, four words for the money line."
    evidence_excerpt: "Affected: mv-dash-stats-today-earnings \"Today's Earnings\"; mv-analytics-total-revenue 'Total Revenue'; mv-earnings-total 'Total Earnings'; vp-ux-dashboard-revenue-lead \"Today's revenue\"; vp-ux-earnings-cards 'Available Balance' / 'Pending Payout' / 'This Month' / 'Lifetime Earnings'; vp-ux-orders-history-summary 'Revenue'."
    recommendation: "Pick one verb for chef-facing money: 'Earnings'. Replace 'Revenue' wherever it refers to chef's take-home. Reserve 'Revenue' for admin/platform-level views. Tiles: \"Today's earnings\" / 'Pending payout' / 'This month' / 'Lifetime earnings'."
    depends_on: null

  - finding_id: TW-051
    surface_id: pattern-empty-state-formula
    lens: technical-writer
    severity: P2
    issue: "PATTERN — Empty states omit the §4 formula 'Why it's empty → One action'. Most state only why, no CTA."
    evidence_excerpt: "Affected: web-ux-admin-chefs 'No chefs found'; web-ux-admin-orders 'No orders found'; web-ux-admin-users 'No users found'; web-ux-catering-quotes-detail 'No quotes yet'; web-ux-catering-quotes-heading 'No requests yet'; web-ux-chef-catering 'No open requests' / 'No pending quotes' / 'No booked events'; web-ux-chef-menu 'No menu items'; web-ux-chef-orders-heading 'No orders found'; web-ux-chef-social 'No posts yet'; web-ux-chefdetail-reviews 'No reviews yet'; web-ux-delivery-dashboard 'No deliveries available'; web-ux-delivery-orders 'No deliveries found'; dp-ux-dashboard-all-clear 'No unassigned orders or pending partners' (good — has context); dp-ux-dashboard-no-deliveries-hint 'New deliveries will appear here.' (decent); mv-menunew-photo-empty 'No photo selected'; ap-userdetail-order-stats 'No orders yet'; ap-secsettings-session-fields 'No active sessions.'; vp-ux-profile-specialties 'No specialties added yet'; vp-ux-menu-view-page 'Menu item not found' (good — but missing action)."
    recommendation: "Add a one-line action to every empty state. Examples: 'No chefs found. Try a different search.' / 'No menu items. Add your first dish.' / 'No reviews yet. Order something to leave one.' / 'No quotes yet. Adjust your event and resubmit.' Reference style guide §4 empty-state pattern."
    depends_on: null

  - finding_id: TW-052
    surface_id: pattern-section-headers-possessive-my
    lens: technical-writer
    severity: P2
    issue: "PATTERN — 'My X' possessive headings throughout. Style guide §3 prefers descriptive nouns; 'My' adds no info when the page is already in the user's account."
    evidence_excerpt: "Affected: web-ux-favorites-heading 'My Favorite Chefs'; web-ux-orders-heading 'My Orders'; mc-favorites-title 'Saved Chefs' (no 'My' — good); mc-orders-title 'My Orders'; mc-catering-tabs 'Request Catering / My Requests'; mc-cartsheet-title 'Your Cart' (also possessive); mv-menu-heading 'My Menu'; mv-more-account-title 'My Account'; web-ux-cart-heading 'Your Cart'; web-ux-catering-quotes-heading 'My Catering Requests' / 'Your Requests'."
    recommendation: "Drop 'My' / 'Your' from page headings: 'Favourites' / 'Orders' / 'Cart' / 'Menu' / 'Account' / 'Requests'. Owner context is implicit because the user is signed in."
    depends_on: null

  - finding_id: TW-053
    surface_id: pattern-success-toast-non-past
    lens: technical-writer
    severity: P2
    issue: "PATTERN — Success messaging doesn't follow §4 'past tense, ≤6 words, period'."
    evidence_excerpt: "Affected: vp-ux-settings-password 'Password updated successfully' (good past tense, drop 'successfully'); md-core-091 '✓ Verified' (good); md-core-047 'You earned ₹{payout} for this delivery' (good past tense but not a toast — celebration card OK); ap-staff-invite-fields 'Copied' (good); dp-ux-staff-invite-created 'Invitation Created' (title case + non-past — should be 'Invitation sent.'); vp-ux-menu-bulk-actions / vp-ux-settings-payout-section 'Razorpay Connected' (good past tense but title case)."
    recommendation: "Sweep success states: 'Password updated.' / 'Invitation sent.' / 'Connected.' / 'Saved.' Drop 'successfully' (redundant). Trailing period (style guide §4)."
    depends_on: null

  - finding_id: TW-054
    surface_id: pattern-modal-subtitle-formula
    lens: technical-writer
    severity: P2
    issue: "PATTERN — Modal subtitles violate §4 'explain consequence in one sentence' formula."
    evidence_excerpt: "Affected: web-ux-orderdetail-cancel 'Please let us know why you're cancelling...' (vague — explain refund, doesn't); dp-ux-active-cancel-prompt 'Reason for cancellation?' (browser prompt — see TW-008); dp-ux-partner-detail-verify-block 'This partner is awaiting verification. Review their documents and approve or reject their application.' (close but two sentences); ap-providers-dialog-delete-title 'Delete Provider' (title only — needs subtitle explaining consequence); ap-providers-dialog-disable-title 'Disable Provider' (title only — same)."
    recommendation: "Add consequence-first subtitle to every confirmation dialog. Examples: 'Cancel this order? Refund returns to the customer immediately.' / 'Delete this provider? Active deliveries on this provider will fail.' / 'Disable this provider? Routing will skip this provider until you re-enable it.'"
    depends_on: null

  - finding_id: TW-055
    surface_id: pattern-add-new-redundant
    lens: technical-writer
    severity: P3
    issue: "PATTERN — 'Add New X' wording is redundant: 'Add' implies new. Style guide §2 plain English."
    evidence_excerpt: "Affected: web-ux-checkout-address-form 'Add New'; mc-checkout-add-address 'Add New Address'; mv-menunew-heading 'Add Menu Item' (good — 'new' implicit); ap-providercreate-title 'Add Delivery Provider' (good); ap-staff-invite-btn 'Invite Staff' (good); dp-ux-staff-invite-btn 'Invite Staff' (good)."
    recommendation: "Drop 'New': 'Add address' / 'Add menu item' / 'Add zone' / 'Add specialty'."
    depends_on: null

  - finding_id: TW-056
    surface_id: pattern-cuisine-list-drift
    lens: technical-writer
    severity: P2
    issue: "PATTERN — Cuisine option lists drift across apps and even within a single app. Same data, different sets."
    evidence_excerpt: "Affected: web-ux-browse-filters-cuisines 'South Indian / North Indian / Italian / Japanese / Mexican / Thai / Chinese / Mediterranean / American / Continental' (10); mc-onb-step3-cuisines 'North Indian, South Indian, Chinese, Continental, Italian, Healthy, Street Food, Desserts' (8); mc-home-cuisine-filters 'All, North Indian, South Indian, Chinese, Continental, Italian, Healthy' (7); mc-profile-cuisines 'North Indian, South Indian, Chinese, Continental, Italian, Healthy, Desserts, Street Food' (8 — order differs from onboarding); mv-onb-kitchen-cuisine-options 'North Indian / South Indian / Chinese / Continental / Bakery / Snacks / Beverages / Other' (8 — entirely different set: includes 'Bakery'/'Snacks'/'Beverages')."
    recommendation: "Define a canonical cuisine taxonomy in shared/types/cuisines.ts (or API). All apps consume same source. Decision: are 'Healthy' / 'Street Food' / 'Desserts' / 'Bakery' / 'Snacks' / 'Beverages' cuisines or categories? Likely a second taxonomy ('Dish category') is needed."
    depends_on: null

  - finding_id: TW-057
    surface_id: pattern-driver-app-status-vs-action-drift
    lens: technical-writer
    severity: P2
    issue: "PATTERN — Driver status labels and action labels overlap in confusing ways: 'Picked Up' status, 'Picked Up Order' action, 'Pick up' (verb). 'In Transit' status, 'Start Delivery' action — different verbs for the same state transition."
    evidence_excerpt: "Affected: md-core-027 'Order Picked Up' (status); md-core-033 'Picked Up Order' (action — see TW-004); md-core-038 'Picked Up / In Transit / At Dropoff / Delivered' (step indicator); md-core-028 'On the Way' (status); md-core-034 'Start Delivery' (action — transitions to 'in_transit' which displays as 'On the Way'); dp-ux-active-mark-as 'Mark as {Picked Up/In Transit/Delivered}' (web, see TW-005)."
    recommendation: "Standardise driver state machine vocabulary. Statuses (past-tense): 'Picked up' / 'On the way' / 'At dropoff' / 'Delivered'. Actions (imperative): 'Pick up' / 'Start trip' / 'Arrive at dropoff' / 'Mark delivered'. Step indicator labels = status labels (past tense). Document in a state-machine table."
    depends_on: TW-001

  - finding_id: TW-058
    surface_id: pattern-favorite-spelling
    lens: technical-writer
    severity: P3
    issue: "PATTERN — 'Favorite' (US) vs 'Favourite' (UK/IN) drift. en-IN locale should prefer 'Favourite' per style guide §6 implicit locale."
    evidence_excerpt: "Affected: web-ux-favorites-heading 'My Favorite Chefs'; mc-favorites-title 'Saved Chefs' (avoids issue); web-ux-profile-preferences 'Favourite Cuisines' (UK spelling — flag for consistency); web-ux-layout-nav 'Favorites' (US)."
    recommendation: "Pick one spelling. Recommend 'Favourite' / 'Favourites' per en-IN locale. Replace 'Favorites' → 'Favourites' in web nav and heading."
    depends_on: null

  - finding_id: TW-059
    surface_id: pattern-bottom-tab-page-heading-drift
    lens: technical-writer
    severity: P3
    issue: "PATTERN — Mobile bottom-tab label differs from the page heading you land on."
    evidence_excerpt: "Affected: mc-favorites-title 'Saved Chefs' (page) — tab is 'Saved'; md-core-002 'Available' (tab) / md-core-015 'Available' (page — match, good); md-core-001 'Dashboard' (tab) / md-core-005 'Dashboard' (page — match, good); dp-ux-nav-bottom-items 'Home' / dp-ux-dashboard-h1 'Dashboard' (mismatch — see TW-011)."
    recommendation: "Tab label = page heading where possible. Mobile-customer: rename tab to 'Saved' AND page to 'Saved'. Delivery-portal: rename bottom-nav to 'Dashboard'."
    depends_on: TW-011

  - finding_id: TW-060
    surface_id: pattern-jargon-est-time
    lens: technical-writer
    severity: P3
    issue: "PATTERN — Abbreviated labels on driver-facing time fields. Style guide §2 plain English; driver audience is glanceable but 'Est. Time' is awkward (period inside abbreviation breaks scan)."
    evidence_excerpt: "Affected: dp-ux-active-est-time-label 'Est. Time'; dp-ux-earnings-avg 'Avg/Delivery' (slash compaction); md-core-021 '~{n} min' (tilde — acceptable on mobile but flag); md-core-019 '{n} km away'."
    recommendation: "'Est. Time' → 'ETA' (3 letters, glanceable, no period). 'Avg/Delivery' → 'Per delivery'. Keep '~{n} min' if used consistently; otherwise prefer 'ETA {n} min'."
    depends_on: null

  - finding_id: TW-061
    surface_id: pattern-uppercase-fields-acronyms
    lens: technical-writer
    severity: P3
    issue: "PATTERN — Acronyms in field labels are inconsistent: 'IFSC Code' (vendor-portal) vs 'IFSC' (mobile-vendor) vs 'IANA' (admin) vs 'TTL' (admin). Acronyms by definition expand — adding 'Code' to 'IFSC Code' is redundant (Indian Financial System Code Code)."
    evidence_excerpt: "Affected: vp-ux-settings-payout-section 'IFSC Code' / 'UPI ID'; mv-earnings-payout-fields 'IFSC' (no 'Code' — better); ap-platsettings-hours-fields 'Timezone (IANA)'; ap-secsettings-session-fields 'Access token TTL (hours)' / 'Refresh token TTL (days)'."
    recommendation: "Drop trailing 'Code' / 'ID' when acronym already encodes the noun: 'IFSC' / 'UPI ID' (keep — 'ID' is the noun here, not redundant). Keep '(IANA)' clarifier — most users won't know which timezone format is expected. Keep 'TTL (hours)' — TTL alone is ambiguous to non-engineers; consider 'Session timeout (hours)' instead for admin clarity."
    depends_on: null

  - finding_id: TW-062
    surface_id: pattern-section-headers-double-noun
    lens: technical-writer
    severity: P3
    issue: "PATTERN — Section headings use double-noun stacking ('Order Items', 'Order Summary', 'Order Activity'). Acceptable but inconsistent — some screens use just the second noun. Drop the 'Order' prefix when context makes it obvious."
    evidence_excerpt: "Affected: web-ux-orderdetail-sections 'Order Items'; mc-order-detail-sections 'Items'; md-core-044 'Order Summary'; mc-checkout-summary-section 'Order Summary'; web-ux-checkout-summary 'Order Summary'; ap-userdetail-order-stats 'Order Activity'."
    recommendation: "Drop 'Order' prefix when on an order detail page. 'Items', 'Summary', 'Activity'. Keep prefix on dashboards where multiple objects share a page."
    depends_on: null

  - finding_id: TW-063
    surface_id: pattern-section-headers-vague-actions
    lens: technical-writer
    severity: P3
    issue: "PATTERN — Section header 'Actions' is so vague it adds nothing — every section page has actions."
    evidence_excerpt: "Affected: dp-ux-partner-actions-h 'Actions'; ap-approvaldetail-detail-fields includes 'Actions'; ap-approvals-table-columns 'Actions'; ap-delivery-table-columns 'Actions' (table column header — acceptable); ap-orders-table-columns 'Actions'; ap-providers-table-columns 'Actions'; ap-staff-table-columns 'Actions'; ap-staff-invite-table-columns 'Actions'; ap-users-table-columns 'Actions'."
    recommendation: "Tables: replace 'Actions' header with no header text (icon column) or '' (empty). Detail pages: use specific labels: 'Partner controls' / 'Approval controls' / 'Order controls'."
    depends_on: null

  # =====================================================================
  # P3 — Sampled polish findings (representative)
  # =====================================================================

  - finding_id: TW-064
    surface_id: web-ux-admin-dashboard
    lens: technical-writer
    severity: P3
    issue: "'Platform at a glance' is fine but 'Recent activity / Shortcuts' as section headings — 'Shortcuts' is functional but uninspired. Brand voice is 'quietly modern' — consider 'Quick links' or just 'Quick actions'."
    evidence_excerpt: "Platform at a glance / Recent activity / Shortcuts"
    recommendation: "Replace 'Shortcuts' with 'Quick actions' (matches vp-ux-dashboard-quick-actions wording, fixes cross-app drift)."
    depends_on: null

  - finding_id: TW-065
    surface_id: web-ux-admin-orders
    lens: technical-writer
    severity: P3
    issue: "Search placeholder 'Search order number...' is awkward. Search-by patterns work better with 'Search by X' or simply '#'."
    evidence_excerpt: "Search order number..."
    recommendation: "'Search by order number' (no trailing dots — placeholder, not loading) or just 'Order number'. Match web-ux-chef-orders-heading 'Search by order number...' style."
    depends_on: TW-040

  - finding_id: TW-066
    surface_id: web-ux-chef-menu
    lens: technical-writer
    severity: P3
    issue: "'e.g., 500g' — comma after 'e.g.' is correct US/UK style but inconsistent with rest of audit. Style guide silent. Most other e.g. helpers in the inventory omit the comma."
    evidence_excerpt: "e.g., 500g"
    recommendation: "Standardise: 'e.g. 500g' (no comma) for compactness. Apply across all e.g. helpers (vp-ux-profile-business-info 'e.g. Meena's Kitchen' already follows this)."
    depends_on: null

  - finding_id: TW-067
    surface_id: web-ux-chef-social
    lens: technical-writer
    severity: P3
    issue: "'#homemade' hashtag suggestion shown as a default — placeholder hashtag promotes one tag over others. Risks looking like AI-generated filler."
    evidence_excerpt: "Social Feed / No posts yet / Share something about this dish... / #homemade"
    recommendation: "Drop the '#homemade' suggestion. Empty placeholder for the hashtag input, or rotate a list of relevant tags drawn from the chef's menu."
    depends_on: null

  - finding_id: TW-068
    surface_id: web-ux-browse-filters-dietary
    lens: technical-writer
    severity: P3
    issue: "'Gluten-Free / Dairy-Free' use hyphen + title case. Style guide silent on hyphenated compounds but 'Gluten free' (open compound, sentence case) is cleaner and equally clear."
    evidence_excerpt: "Vegetarian / Vegan / Gluten-Free / Dairy-Free / Keto / Halal"
    recommendation: "'Vegetarian / Vegan / Gluten free / Dairy free / Keto / Halal' — sentence case, open compounds."
    depends_on: TW-031

  - finding_id: TW-069
    surface_id: web-ux-catering-request-food
    lens: technical-writer
    severity: P3
    issue: "Placeholder 'Tell us more about your event, special requests, theme, etc.' uses 'etc.' which is filler. Style guide §2 plain English."
    evidence_excerpt: "Food Preferences / Tell us more about your event, special requests, theme, etc."
    recommendation: "Drop 'etc.' Rewrite: 'Anything else? Special requests, theme, dietary needs.'"
    depends_on: null

  - finding_id: TW-070
    surface_id: web-ux-catering-request-location
    lens: technical-writer
    severity: P3
    issue: "Address placeholder '123 Main Street / Suite 100' is US-style, but app is en-IN. Indian addresses don't use 'Suite'."
    evidence_excerpt: "Event Location / 123 Main Street / Suite 100"
    recommendation: "'123 MG Road / Apartment, floor, landmark (optional)' — Indian-style examples."
    depends_on: null

  - finding_id: TW-071
    surface_id: web-ux-profile-addresses
    lens: technical-writer
    severity: P3
    issue: "Address form helpers use 'House / flat / building number, street' and 'Landmark, area (optional)' — slashes work as separators but make the helper hard to scan."
    evidence_excerpt: "Saved Addresses / Enter PIN code / House / flat / building number, street / Landmark, area (optional)"
    recommendation: "'House or flat number, street' / 'Landmark, area (optional)'. Replace slash separators with 'or' for plain English."
    depends_on: null

  - finding_id: TW-072
    surface_id: web-ux-delivery-dashboard
    lens: technical-writer
    severity: P3
    issue: "'You're currently offline' — 'currently' is filler. 'You're offline' suffices."
    evidence_excerpt: "Performance at a glance / Current Delivery / Available Deliveries / No deliveries available / You're currently offline"
    recommendation: "'You're offline.' Drop 'currently'."
    depends_on: null

  - finding_id: TW-073
    surface_id: dp-ux-dashboard-partner-offline
    lens: technical-writer
    severity: P3
    issue: "'Currently offline' — 'Currently' filler word."
    evidence_excerpt: "Currently offline"
    recommendation: "'Offline'."
    depends_on: TW-072

  - finding_id: TW-074
    surface_id: dp-ux-dashboard-no-deliveries-hint
    lens: technical-writer
    severity: P3
    issue: "'New deliveries will appear here.' — passive 'will appear here' is fine but generic. Could pair with 'Go online' CTA for empty-state formula §4."
    evidence_excerpt: "New deliveries will appear here."
    recommendation: "'No deliveries right now. Stay online.' Combines empty-state formula 'why → action' more directly."
    depends_on: TW-051

  - finding_id: TW-075
    surface_id: dp-ux-stripe-sub
    lens: technical-writer
    severity: P3
    issue: "Subtitle 'For drivers outside India, or as an alternative to Razorpay.' — sentence fragment. Style guide §1 plain English short sentences."
    evidence_excerpt: "For drivers outside India, or as an alternative to Razorpay."
    recommendation: "'For drivers outside India, or instead of Razorpay.' Or full sentence: 'Use Stripe if you're outside India, or instead of Razorpay.'"
    depends_on: null

  - finding_id: TW-076
    surface_id: dp-ux-stripe-intro
    lens: technical-writer
    severity: P3
    issue: "'Stripe handles KYC and bank verification on its hosted pages.' — 'KYC' is industry jargon; admins/drivers may not know the expansion. Style guide §2 plain English."
    evidence_excerpt: "Accept delivery payouts in your local currency. Stripe handles KYC and bank verification on its hosted pages."
    recommendation: "'Stripe handles identity and bank verification on its own pages.' Or expand once: 'KYC (know-your-customer) checks and bank verification happen on Stripe's site.'"
    depends_on: null

  - finding_id: TW-077
    surface_id: dp-ux-partner-readonly-notice
    lens: technical-writer
    severity: P3
    issue: "'This partner is pending verification. Contact a fleet manager to approve.' — 'a fleet manager' is impersonal; reader may not know who to contact."
    evidence_excerpt: "This partner is pending verification. Contact a fleet manager to approve."
    recommendation: "'This partner is pending verification. Your fleet manager must approve it.' Replaces vague 'a' with 'your', clarifies who acts."
    depends_on: null

  - finding_id: TW-078
    surface_id: dp-ux-partner-doc-meta
    lens: technical-writer
    severity: P3
    issue: "'{type} · Uploaded {date}' — middle dot fine but '{date}' should be relative for recent / absolute for older per style guide §6."
    evidence_excerpt: "{type} · Uploaded {date}"
    recommendation: "'{type} · Uploaded {relativeOrAbsoluteDate}' — format via Intl.RelativeTimeFormat under 24h, Intl.DateTimeFormat thereafter."
    depends_on: null

  - finding_id: TW-079
    surface_id: dp-ux-staff-invite-meta
    lens: technical-writer
    severity: P3
    issue: "'{role} · Sent {date}' — same date formatting concern as TW-078."
    evidence_excerpt: "{role} · Sent {date}"
    recommendation: "Same — relative time under 24h, absolute thereafter, via Intl APIs."
    depends_on: TW-078

  - finding_id: TW-080
    surface_id: dp-ux-staff-roles-fallback
    lens: technical-writer
    severity: P3
    issue: "Hardcoded role-list fallback ('Delivery Ops / Fleet Manager / Super Admin') means roles will drift from server-driven roles in ap-staff-roles ('Super Admin; Admin; Fleet Manager; Delivery Ops; Support')."
    evidence_excerpt: "Delivery Ops / Fleet Manager / Super Admin"
    recommendation: "Sync role taxonomy with admin-portal. Either drop the fallback (let the server be the source) or keep all 5 roles in the fallback set."
    depends_on: TW-046

  - finding_id: TW-081
    surface_id: dp-ux-active-status-delivered
    lens: technical-writer
    severity: P3
    issue: "Driver-facing status 'Delivered' is sentence case (good). But neighbour 'Picked Up' / 'In Transit' use title case (per dp-ux-active-status-* surfaces) — within-set inconsistency."
    evidence_excerpt: "Picked Up / In Transit / Delivered"
    recommendation: "All sentence case: 'Picked up' / 'In transit' / 'Delivered'."
    depends_on: TW-033

  - finding_id: TW-082
    surface_id: vp-ux-dashboard-pending-card
    lens: technical-writer
    severity: P3
    issue: "'+N more items' — concatenation with prefix '+' can break translation (some languages put 'more' before the number)."
    evidence_excerpt: "Pending Orders / New / +N more items / Accept / Reject"
    recommendation: "Use ICU MessageFormat pattern: '{n, plural, one {1 more item} other {# more items}}'. Avoid '+' prefix in source string."
    depends_on: null

  - finding_id: TW-083
    surface_id: vp-ux-dashboard-no-pending
    lens: technical-writer
    severity: P3
    issue: "'All caught up / New orders will appear here.' — good empty-state pattern (why → what next), minor: 'will appear here' is generic."
    evidence_excerpt: "All caught up / New orders will appear here."
    recommendation: "Acceptable as-is. Optional polish: 'All caught up. New orders show up here.' (active voice)."
    depends_on: null

  - finding_id: TW-084
    surface_id: vp-ux-dashboard-quick-actions
    lens: technical-writer
    severity: P3
    issue: "Tile copy uses comma-separator ('Add Menu Item, Create a new dish listing') which makes title and subtitle blur. Layout issue mostly, but title case + comma + sentence case creates messy rhythm."
    evidence_excerpt: "Quick Actions / Add Menu Item, Create a new dish listing / View Orders, Manage live orders / Update Profile, Edit kitchen details / View Earnings, Track your revenue"
    recommendation: "Title and subtitle in separate strings. Sentence case throughout: 'Add menu item' / 'Create a new dish'. 'View orders' / 'Manage live orders'. 'Update profile' / 'Edit kitchen details'. 'View earnings' / 'Track your revenue'."
    depends_on: TW-034

  - finding_id: TW-085
    surface_id: vp-ux-kitchen-photos
    lens: technical-writer
    severity: P3
    issue: "Helper text 'JPEG, PNG, or WebP. Max 5 MB each. Up to 5 photos.' — three sentences crammed together. Style guide §2 short sentences; one constraint per sentence."
    evidence_excerpt: "JPEG, PNG, or WebP. Max 5 MB each. Up to 5 photos."
    recommendation: "Bullets or three lines: 'JPEG, PNG, or WebP · Max 5 MB · Up to 5 photos'."
    depends_on: null

  - finding_id: TW-086
    surface_id: vp-ux-profile-business-info
    lens: technical-writer
    severity: P3
    issue: "Placeholder 'Tell customers about your kitchen, cooking style, and what makes your food special...' — 14 words exceeds vendor sentence cap (5-12). Also banned vocabulary: 'special' is filler-marketing."
    evidence_excerpt: "Tell customers about your kitchen, cooking style, and what makes your food special..."
    recommendation: "'Tell customers about your kitchen and what you cook.' 9 words, no filler."
    depends_on: null

  - finding_id: TW-087
    surface_id: vp-ux-settings-notifications
    lens: technical-writer
    severity: P3
    issue: "Long stack of notification descriptions ('Get notified when a new order comes in' / 'Notifications for order status changes' / 'Receive a daily email with order and earnings summary') — mix of verbs (get / receive / notifications). Pick one verb form."
    evidence_excerpt: "Get notified when a new order comes in / Notifications for order status changes / Receive a daily email with order and earnings summary / Weekly performance report with analytics / Get an SMS for each new order"
    recommendation: "Use noun-phrase descriptions, consistent: 'Alerts when new orders arrive.' / 'Updates on order status changes.' / 'Daily email summary of orders and earnings.' / 'Weekly performance report.' / 'SMS for every new order.' Vendor tone matrix: crisp 5-12 words."
    depends_on: null

  - finding_id: TW-088
    surface_id: vp-ux-settings-password
    lens: technical-writer
    severity: P3
    issue: "Error 'Failed to update password. Check your current password.' — formula 'what happened → what to do' satisfied. Polish: 'Check' could be 'Try' or 'Re-enter'. 'Update Password' → 'Update password' (case)."
    evidence_excerpt: "Update Password / Password updated successfully / Failed to update password. Check your current password."
    recommendation: "'Password didn't update. Re-enter your current password.' Past-tense success: 'Password updated.' (drop 'successfully' per TW-053)."
    depends_on: TW-053

  - finding_id: TW-089
    surface_id: mc-orders-filters
    lens: technical-writer
    severity: P3
    issue: "'All, Active, Delivered, Cancelled' filter chips — drift with web-ux-orders-filters 'All Orders / Active / Completed / Cancelled'. 'All' alone (mobile) vs 'All Orders' (web). 'Delivered' (mobile) vs 'Completed' (web)."
    evidence_excerpt: "All, Active, Delivered, Cancelled"
    recommendation: "Sync filter sets across web + mobile. Recommend: 'All' / 'Active' / 'Delivered' / 'Cancelled' on both (since 'Completed' is ambiguous; an order can be 'completed and cancelled')."
    depends_on: null

  - finding_id: TW-090
    surface_id: mc-profile-cuisines
    lens: technical-writer
    severity: P3
    issue: "Inventory note: order of cuisines differs from onboarding step 3. Adds cognitive friction for users who set preferences during onboarding and revisit later."
    evidence_excerpt: "North Indian, South Indian, Chinese, Continental, Italian, Healthy, Desserts, Street Food  (profile, 8)  vs  ...Healthy, Street Food, Desserts  (onboarding, 8)"
    recommendation: "Same canonical cuisine list (TW-056) AND same order across all surfaces."
    depends_on: TW-056

  - finding_id: TW-091
    surface_id: mc-profile-more-rows
    lens: technical-writer
    severity: P3
    issue: "'Social Feed / Catering' rows shown with emoji icons (📱/🍽️). Brand 'quietly modern' avoids emoji in production UI; design system uses Lucide icons. Mixed icon styles."
    evidence_excerpt: "Social Feed / Catering  (with emoji 📱/🍽️)"
    recommendation: "Replace emoji with design-system icons (Lucide / @tesserix/native equivalents). 'Social' label fine if Catering tile uses a fork-and-knife icon vs emoji."
    depends_on: null

  - finding_id: TW-092
    surface_id: mv-onb-pending-rejected-cta
    lens: technical-writer
    severity: P3
    issue: "'Reapply' single-word CTA after rejection. Friendly, but could be 'Apply again' (two words, plainer English) or 'Update application'."
    evidence_excerpt: "Reapply"
    recommendation: "'Apply again' or 'Update application' (clearer about what happens — the chef edits the existing app)."
    depends_on: null

  - finding_id: TW-093
    surface_id: mv-menunew-desc-ph
    lens: technical-writer
    severity: P3
    issue: "Placeholder 'Describe your dish (at least 20 characters)' — character count in placeholder is dev-leak. Helper text below the field is the right place."
    evidence_excerpt: "Describe your dish (at least 20 characters)"
    recommendation: "Placeholder: 'Describe your dish'. Helper text below: 'At least 20 characters.' Or live counter '0/200'."
    depends_on: null

  - finding_id: TW-094
    surface_id: mv-menunew-type-veg
    lens: technical-writer
    severity: P3
    issue: "'Veg / Non-Veg' toggle — India-specific but inconsistent hyphenation: 'Non-Veg' with hyphen and capital V. Inventory notes flag this. Style guide §2 plain English."
    evidence_excerpt: "Veg / Non-Veg"
    recommendation: "Stay India-specific but standardise: 'Veg' / 'Non-veg' (sentence case, hyphenated). Or full words: 'Vegetarian' / 'Non-vegetarian'."
    depends_on: null

  - finding_id: TW-095
    surface_id: md-core-009
    lens: technical-writer
    severity: P3
    issue: "'Toggle to start receiving requests' — driver tone matrix is ≤4 words telegraphic. Sentence is 6 words and explains a mechanism the driver already sees (toggle is right there)."
    evidence_excerpt: "Toggle to start receiving requests"
    recommendation: "'Go online to receive requests' (5 words, names the state) or 'Tap to go online' (4 words)."
    depends_on: null

  - finding_id: TW-096
    surface_id: md-core-018
    lens: technical-writer
    severity: P3
    issue: "'Go Online to Accept Deliveries' — 5 words, title case, on a driver-facing CTA. Title style + length violates driver telegraphic rule."
    evidence_excerpt: "Go Online to Accept Deliveries"
    recommendation: "'Go online' (2 words). Or as banner+CTA: 'Offline — Tap to go online'."
    depends_on: TW-041

  - finding_id: TW-097
    surface_id: md-core-031
    lens: technical-writer
    severity: P3
    issue: "'Delivery Cancelled' terminal status uses British spelling 'Cancelled' (good for en-IN). md-core-007 has no spelling concern. Confirm consistency across all surfaces."
    evidence_excerpt: "Delivery Cancelled"
    recommendation: "Keep 'Cancelled' (UK/IN double-l). Audit for any 'Canceled' (US single-l) leaks across all apps."
    depends_on: null

  - finding_id: TW-098
    surface_id: md-core-039
    lens: technical-writer
    severity: P3
    issue: "Banner copy 'This delivery has been cancelled. Check the Available tab for new requests.' is 12 words — exceeds driver sentence cap (≤12 words). Acceptable for this surface (terminal state), but 'Available tab' may not exist after future redesign — over-specific."
    evidence_excerpt: "This delivery has been cancelled. Check the Available tab for new requests."
    recommendation: "'Cancelled. Find new requests under Available.' 7 words. Or: 'Cancelled. New requests appear under Available.'"
    depends_on: null

  - finding_id: TW-099
    surface_id: md-core-044
    lens: technical-writer
    severity: P3
    issue: "'Order Summary ({n} item(s))' — uses 'item(s)' notation. Style guide §6: never 'item(s)'."
    evidence_excerpt: "Order Summary ({n} item(s))"
    recommendation: "ICU plural: 'Summary · {n, plural, one {# item} other {# items}}'. Drop 'Order' prefix per TW-062."
    depends_on: TW-062

  - finding_id: TW-100
    surface_id: md-core-099
    lens: technical-writer
    severity: P3
    issue: "'Today: {n} deliveries' — colon + concatenation, will break in some languages (number-first locales)."
    evidence_excerpt: "Today: {n} deliveries"
    recommendation: "ICU pattern: \"Today · {n, plural, one {# delivery} other {# deliveries}}\". Drop colon."
    depends_on: TW-039

  - finding_id: TW-101
    surface_id: md-core-109
    lens: technical-writer
    severity: P3
    issue: "'Joined {date}' — relative date pattern; same issue as TW-078."
    evidence_excerpt: "Joined {date}"
    recommendation: "Relative under 24h ('Joined 35 min ago'), absolute thereafter ('Joined 3 May'). Format via Intl.RelativeTimeFormat."
    depends_on: TW-078

  - finding_id: TW-102
    surface_id: md-core-046
    lens: technical-writer
    severity: P3
    issue: "'Special instructions:' — trailing colon (see TW-039 pattern). Driver surface."
    evidence_excerpt: "Special instructions:"
    recommendation: "Drop colon. Use layout (caption above body)."
    depends_on: TW-039

  - finding_id: TW-103
    surface_id: dp-ux-stripe-resume-onboarding
    lens: technical-writer
    severity: P3
    issue: "'Resume Onboarding' — title case + 'onboarding' is dev jargon to a driver. Style guide §2 plain English."
    evidence_excerpt: "Resume Onboarding"
    recommendation: "'Continue setup' or 'Finish setup'. Sentence case."
    depends_on: null

  - finding_id: TW-104
    surface_id: ap-approvals-types
    lens: technical-writer
    severity: P3
    issue: "Approval-type taxonomy uses title case ('Kitchen Onboarding', 'Document Verification', 'New Menu Item', 'Pricing Change'). Sentence case across the board, plus 'onboarding' jargon could be 'application'."
    evidence_excerpt: "Kitchen Onboarding; Document Verification; New Menu Item; Menu Update; Pricing Change; Kitchen Update; Driver Onboarding; Driver Document"
    recommendation: "'Kitchen application; Document verification; New menu item; Menu update; Pricing change; Kitchen update; Driver application; Driver document.' Replace 'Onboarding' with 'application' (plain English)."
    depends_on: TW-032

  - finding_id: TW-105
    surface_id: ap-providerdetail-status-fields
    lens: technical-writer
    severity: P3
    issue: "'Enabled / Yes / No / Active / Inactive / Last Used / Created / Never / Unlimited' — mix of states and yes/no values shown together. Could read as confusing depending on layout."
    evidence_excerpt: "Enabled; Yes; No; Active; Inactive; Last Used; Created; Never; Unlimited"
    recommendation: "Audit layout. Group: status row ('Status: Active / Inactive'), enabled row ('Enabled: Yes / No'), audit timestamps ('Last used · Created'). Avoid showing 'Yes/No' beside 'Active/Inactive' without context."
    depends_on: null

  - finding_id: TW-106
    surface_id: ap-secsettings-2fa-account
    lens: technical-writer
    severity: P3
    issue: "'Enabled — login requires a 6-digit code' / 'Disabled — enable to protect your admin account' — 'login' noun banned per §3."
    evidence_excerpt: "Enabled — login requires a 6-digit code; Disabled — enable to protect your admin account"
    recommendation: "'Enabled — sign-in requires a 6-digit code.' / 'Disabled — enable to protect your account.' Drop 'admin' (it's already your admin account)."
    depends_on: TW-044

  - finding_id: TW-107
    surface_id: ap-secsettings-apikey-fields
    lens: technical-writer
    severity: P3
    issue: "'I've saved it' button label is unusual — first-person voice on a button. Style guide §4 buttons verb-first."
    evidence_excerpt: "Expires in (days; 0 = never); Create key; New API key; e.g. Partner integration; I've saved it"
    recommendation: "'Done' or 'Got it'. Drop first-person."
    depends_on: null

  - finding_id: TW-108
    surface_id: ap-settings-card-titles
    lens: technical-writer
    severity: P3
    issue: "Settings card titles 'Payment Gateway' / 'Stripe Gateway' — 'gateway' is jargon. 'Payment processor' is plainer."
    evidence_excerpt: "Payment Gateway; Razorpay integration status; Stripe Gateway; International payouts"
    recommendation: "'Payments' / 'Stripe payments' / 'International payouts'. Drop 'Gateway' jargon."
    depends_on: null

  - finding_id: TW-109
    surface_id: ap-exports-cards
    lens: technical-writer
    severity: P3
    issue: "Export card descriptions leak database column names: 'All user accounts (id, email, role, created_at)' — 'id, role, created_at' are SQL identifiers, not user-facing labels."
    evidence_excerpt: "User data / All user accounts (id, email, role, created_at); Orders / Order rows filtered by the date range above; Revenue / Per-day revenue rollup — paid orders only"
    recommendation: "'User data — name, email, role, join date.' / 'Orders — within date range above.' / 'Revenue — per day, paid orders only.'"
    depends_on: null

  - finding_id: TW-110
    surface_id: ap-platsettings-zones-fields
    lens: technical-writer
    severity: P3
    issue: "Zone CRUD labels 'Min latitude / Max latitude / Min longitude / Max longitude' use 'Min' / 'Max' abbreviations. Style guide §2 plain English."
    evidence_excerpt: "Name; City; State; Min latitude; Max latitude; Min longitude; Max longitude; New zone; Create zone; Cancel"
    recommendation: "'Minimum latitude' / 'Maximum latitude' (and same for longitude). Or, if space is tight, group: 'Latitude (min, max)' with two adjacent fields."
    depends_on: null

  - finding_id: TW-111
    surface_id: vp-ux-orders-history-title
    lens: technical-writer
    severity: P3
    issue: "Page header 'Order History / {n} orders found / Live Orders / Back' — 'Back' as a heading element is misplaced; usually a navigation control. 'Live Orders' as part of the header is confusing — it's a sibling page."
    evidence_excerpt: "Order History / {n} orders found / Live Orders / Back"
    recommendation: "Confirm IA. If 'Live Orders' is a tab beside 'Order History', that's fine. 'Back' should be an icon-button or 'Back to live orders' link, not part of the page title."
    depends_on: null

  - finding_id: TW-112
    surface_id: vp-ux-kitchen-photos
    lens: technical-writer
    severity: P3
    issue: "Drop-zone copy 'Click or drag photos here' / 'Drop your photo here' — two distinct messages for empty vs drag-hover state, both work, but 'Click' is mouse-centric. Touch devices have no click."
    evidence_excerpt: "Click or drag photos here / Drop your photo here / Uploading..."
    recommendation: "'Tap or drag photos here' (mobile + desktop). Or 'Drop or browse photos'. Aligns with brand 'quietly modern' across input modalities."
    depends_on: null

  - finding_id: TW-113
    surface_id: vp-ux-menu-view-page
    lens: technical-writer
    severity: P3
    issue: "Long mixed-content surface includes 'Item ID' — exposing internal IDs is fine for admin debugging but should be clearly labelled as such (not just a stat tile)."
    evidence_excerpt: "Item ID  (alongside 'Orders' / 'Rating')"
    recommendation: "Group 'Item ID' under a 'System' or 'Debug' subhead, distinct from performance stats. Or use shorter 'ID' label with copy icon."
    depends_on: null

  - finding_id: TW-114
    surface_id: dp-ux-active-order-note
    lens: technical-writer
    severity: P3
    issue: "'Note: {specialInstructions}' — colon + concatenation issue per TW-039 / TW-082."
    evidence_excerpt: "Note: {specialInstructions}"
    recommendation: "Display as quoted note with caption above: 'Chef's note' on one line, '{instructions}' below. Drop 'Note:' prefix."
    depends_on: TW-039

  - finding_id: TW-115
    surface_id: ap-layout-nav-reviews
    lens: technical-writer
    severity: P2
    issue: "Sidebar nav label 'Reviews' routes to /approvals — label says 'Reviews' but the page is 'Approval Reviews' (ap-approvals-title). Two different concepts (product reviews vs approval reviews). Customer-facing 'Reviews' (product feedback) exists elsewhere; admin label is misleading."
    evidence_excerpt: "Reviews  (nav)  →  Approval Reviews  (page)"
    recommendation: "Rename nav to 'Approvals'. Page title becomes 'Approvals' too. Reserve 'Reviews' for customer feedback management if/when that admin page exists."
    depends_on: null

  - finding_id: TW-116
    surface_id: dp-ux-active-cancel-btn
    lens: technical-writer
    severity: P3
    issue: "'Cancel Delivery' destructive button — title case. Driver telegraphic + sentence case."
    evidence_excerpt: "Cancel Delivery"
    recommendation: "'Cancel delivery' (sentence case)."
    depends_on: TW-033

  - finding_id: TW-117
    surface_id: dp-ux-staff-invite-h
    lens: technical-writer
    severity: P3
    issue: "Heading 'Send Staff Invitation' — title case + redundant ('staff invitation' on the staff invite page is obvious)."
    evidence_excerpt: "Send Staff Invitation"
    recommendation: "'Invite a teammate' or 'New invite'. Sentence case. Page is already on staff-management — drop 'staff'."
    depends_on: TW-033

  - finding_id: TW-118
    surface_id: ap-staff-invite-dialog-title
    lens: technical-writer
    severity: P3
    issue: "Dialog title 'Invite Staff Member' — title case, 3 words. 'Staff member' is redundant on the staff page."
    evidence_excerpt: "Invite Staff Member"
    recommendation: "'Invite teammate' (2 words, sentence case)."
    depends_on: TW-032

  - finding_id: TW-119
    surface_id: ap-staffdetail-info-rows
    lens: technical-writer
    severity: P3
    issue: "'Joined' / 'Last Login' info rows — 'Login' (noun) banned per §3 + title case."
    evidence_excerpt: "Email; Phone; Department; Title; Joined; Last Login"
    recommendation: "'Email / Phone / Department / Title / Joined / Last sign-in' (sentence case, 'sign-in' replacing 'login')."
    depends_on: TW-044

  - finding_id: TW-120
    surface_id: ap-userdetail-info-rows
    lens: technical-writer
    severity: P3
    issue: "'Auth Provider' / 'Last Login' — 'Login' (noun) banned + 'Auth' jargon."
    evidence_excerpt: "Email; Phone; Auth Provider; Role; Joined; Last Login"
    recommendation: "'Email / Phone / Sign-in method / Role / Joined / Last sign-in'."
    depends_on: TW-044

  - finding_id: TW-121
    surface_id: ap-userdetail-verification
    lens: technical-writer
    severity: P3
    issue: "'Account Active' / 'Not verified' verification labels mix tense and concept (active vs verified). Single concept per label."
    evidence_excerpt: "Verification; Email; Phone; Account Active; Verified; Not verified"
    recommendation: "Group: 'Email verified · Yes/No', 'Phone verified · Yes/No', 'Account · Active/Suspended'. Don't mix verified/active in one row."
    depends_on: null

  - finding_id: TW-122
    surface_id: web-ux-orderdetail-cancel
    lens: technical-writer
    severity: P3
    issue: "Modal subtitle 'Please let us know why you're cancelling...' starts with 'Please' (overly polite) and ends with '...' instead of explaining refund consequence."
    evidence_excerpt: "Cancel Order / Please let us know why you're cancelling..."
    recommendation: "Title: 'Cancel order?' Subtitle: 'Tell us why. We'll refund you within 5–7 business days.' Drops 'please'; states the consequence per §4."
    depends_on: TW-054

  - finding_id: TW-123
    surface_id: web-ux-chef-profile
    lens: technical-writer
    severity: P3
    issue: "Placeholder 'Tell customers about your kitchen...' — ellipsis suggests open-ended. Different from vp-ux-profile-business-info (TW-086) which is longer."
    evidence_excerpt: "Tell customers about your kitchen..."
    recommendation: "'Tell customers about your kitchen and what you cook.' Drop ellipsis; full sentence. Match vp-ux-profile-business-info rewrite."
    depends_on: TW-086

  - finding_id: TW-124
    surface_id: web-ux-chefdetail-reviews
    lens: technical-writer
    severity: P3
    issue: "'Chef's Response:' (trailing colon) — see TW-039 pattern. Also possessive 'Chef's' fine, but customer-facing should consider 'Chef' (no possessive — the response is shown right below)."
    evidence_excerpt: "Chef's Response:"
    recommendation: "'Reply from chef' or 'Chef replied' (caption above quoted text). Drop colon."
    depends_on: TW-039

  - finding_id: TW-125
    surface_id: web-ux-orderdetail-sections
    lens: technical-writer
    severity: P3
    issue: "'Placed on {date} / Estimated delivery:' — colon + 'Estimated delivery' is wordy. 'ETA' or 'Arriving' is shorter."
    evidence_excerpt: "Back to Orders / Order #{n} / Placed on {date} / Estimated delivery: / Order Items / Delivery Address / Payment Summary / Special Instructions / Leave a Review"
    recommendation: "'Placed {date}' (drop 'on'; use Intl). 'ETA · {time}' or 'Arriving · {time}'. Sentence case everywhere. 'Order Items' → 'Items' per TW-062."
    depends_on: null

  - finding_id: TW-126
    surface_id: ap-staff-roles
    lens: technical-writer
    severity: P3
    issue: "Role taxonomy 'Super Admin / Admin / Fleet Manager / Delivery Ops / Support' — 'Delivery Ops' is short for 'Operations' (industry jargon). Drift with dp-ux-staff-roles-fallback (which orders them differently)."
    evidence_excerpt: "Super Admin; Admin; Fleet Manager; Delivery Ops; Support"
    recommendation: "Sentence case + plain: 'Super admin', 'Admin', 'Fleet manager', 'Delivery operations', 'Support'. Keep order consistent with dp-ux-staff-roles-fallback (TW-080)."
    depends_on: TW-080

  - finding_id: TW-127
    surface_id: dp-ux-active-pickup-from
    lens: technical-writer
    severity: P3
    issue: "Eyebrow 'Pickup from' (uppercase via CSS — inventory notes). Sentence-case string is fine; ALL CAPS only via CSS. Verify CSS not changing the underlying string."
    evidence_excerpt: "Pickup from  (uppercase eyebrow)"
    recommendation: "Keep string 'Pickup from'. Apply text-transform: uppercase via CSS if desired. Don't shout in the source string."
    depends_on: TW-042

  - finding_id: TW-128
    surface_id: dp-ux-active-deliver-to
    lens: technical-writer
    severity: P3
    issue: "Eyebrow 'Deliver to' — sister to TW-127. Same treatment."
    evidence_excerpt: "Deliver to"
    recommendation: "Keep string 'Deliver to'. CSS-only uppercase."
    depends_on: TW-127

  - finding_id: TW-129
    surface_id: web-ux-checkout-address-form
    lens: technical-writer
    severity: P3
    issue: "'Apartment, suite, etc. (optional)' — US 'suite' + 'etc.' filler in a placeholder. India-friendly version preferred."
    evidence_excerpt: "Delivery Address / Add New / Label / Street Address / Apartment, suite, etc. (optional) / City / State / Postal Code / Save Address"
    recommendation: "'Apartment, floor, landmark (optional)'. Match web-ux-catering-request-location and web-ux-profile-addresses Indian-style examples."
    depends_on: TW-070

  - finding_id: TW-130
    surface_id: dp-ux-fleet-h
    lens: technical-writer
    severity: P3
    issue: "Subtitle 'Monitor your delivery fleet in real time' — 'real time' (two words) vs 'real-time' (hyphenated adjective) — verify usage; here it's adverbial so two words is correct."
    evidence_excerpt: "Fleet Overview + Monitor your delivery fleet in real time"
    recommendation: "OK as-is (adverbial 'in real time' two words). Apply title fix: 'Fleet overview' (sentence case)."
    depends_on: TW-033

  - finding_id: TW-131
    surface_id: dp-ux-partner-detail-verify-block
    lens: technical-writer
    severity: P3
    issue: "'Pending Verification' eyebrow + 'This partner is awaiting verification. Review their documents and approve or reject their application.' — 2 sentences, 16 words. Decent voice. Could be tightened."
    evidence_excerpt: "Pending Verification + This partner is awaiting verification. Review their documents and approve or reject their application."
    recommendation: "Eyebrow 'Pending verification'. Body: 'Review their documents and approve or reject.' (drop redundant 'awaiting verification' — eyebrow already says it)."
    depends_on: null

  - finding_id: TW-132
    surface_id: mc-cartsheet-checkout-cta
    lens: technical-writer
    severity: P3
    issue: "'Proceed to Checkout' — 3 words, title case, redundant verb ('proceed to' = 'go to' = nothing)."
    evidence_excerpt: "Proceed to Checkout"
    recommendation: "'Checkout' (1 word). Sentence case."
    depends_on: TW-014

  - finding_id: TW-133
    surface_id: ap-notifsettings-categories
    lens: technical-writer
    severity: P3
    issue: "Category descriptions vary in tone: 'Promotions / Discounts, newsletters, and marketing campaigns. Opt-in.' — 'Opt-in' fragment without context implies legal requirement but reads informally."
    evidence_excerpt: "Promotions / Discounts, newsletters, and marketing campaigns. Opt-in."
    recommendation: "'Promotions — Discounts, newsletters, marketing campaigns. Off by default.' Clarifies the opt-in for the user."
    depends_on: null

  - finding_id: TW-134
    surface_id: dp-ux-stripe-title
    lens: technical-writer
    severity: P3
    issue: "'Stripe (International Payouts)' — parenthetical title case. Sentence case throughout."
    evidence_excerpt: "Stripe (International Payouts)"
    recommendation: "'Stripe — international payouts'. Em-dash beats parens; sentence case."
    depends_on: TW-033

  - finding_id: TW-135
    surface_id: mv-onb-docs-pdf-uploaded
    lens: technical-writer
    severity: P3
    issue: "'PDF uploaded' — past-tense success, good. Mobile shows 'Uploaded' on mv-onb-review-doc-uploaded for the same concept — drift."
    evidence_excerpt: "PDF uploaded  (docs.tsx)  vs  Uploaded  (review.tsx)"
    recommendation: "Consistent: 'Uploaded' for the status badge. 'PDF uploaded' if you want format-specific feedback in the upload widget — keep both BUT they convey different things."
    depends_on: null

  - finding_id: TW-136
    surface_id: mv-onb-review-doc-not-uploaded
    lens: technical-writer
    severity: P3
    issue: "'Not uploaded' — two words, fine. Could pair with action: 'Not uploaded — Upload now' for empty-state formula §4."
    evidence_excerpt: "Not uploaded"
    recommendation: "Optional polish: 'Not uploaded' as the badge; pair with an inline 'Upload' button."
    depends_on: TW-051

  - finding_id: TW-137
    surface_id: mv-tabs-orders-label
    lens: technical-writer
    severity: P3
    issue: "Inventory note: matches vendor-portal. Just verifying — no issue here. Including for completeness."
    evidence_excerpt: "Orders"
    recommendation: "OK as-is. No change."
    depends_on: null

  - finding_id: TW-138
    surface_id: mv-undo-cta
    lens: technical-writer
    severity: P3
    issue: "Snackbar action 'UNDO' is ALL CAPS. Inventory notes Material-style convention; brand prefers sentence case microcopy."
    evidence_excerpt: "UNDO"
    recommendation: "'Undo' (sentence case). CSS text-transform if visual uppercase desired."
    depends_on: TW-042

  - finding_id: TW-139
    surface_id: md-core-091
    lens: technical-writer
    severity: P3
    issue: "Verification badge uses '✓ Verified' (with checkmark glyph) vs 'Pending' (no glyph). Asymmetric visual treatment; one has icon-in-string, other doesn't."
    evidence_excerpt: "✓ Verified / Pending"
    recommendation: "Use icon component (CheckCircle / Clock) for both states. Strings stay icon-free for i18n safety."
    depends_on: null

  - finding_id: TW-140
    surface_id: md-core-006
    lens: technical-writer
    severity: P3
    issue: "Status banner 'You are Online' — capital O on second word (title case for one word). 'You're online' is friendlier and sentence case."
    evidence_excerpt: "You are Online"
    recommendation: "'You're online.' Single sentence, contraction allowed per warm tone."
    depends_on: TW-037

  - finding_id: TW-141
    surface_id: md-core-016
    lens: technical-writer
    severity: P3
    issue: "'You're Offline' — title case 'Offline'. Drift with md-core-006 ('You are Online'). Same screen group should agree."
    evidence_excerpt: "You're Offline"
    recommendation: "'You're offline.' Lowercase 'offline' in both. Pair with md-core-007."
    depends_on: TW-140

  - finding_id: TW-142
    surface_id: dp-ux-dashboard-partner-online
    lens: technical-writer
    severity: P3
    issue: "'Ready for deliveries' — 3 words, good driver tone. 'Standing by' / 'Waiting nearby' (other dashboard states) drift — should pick a single 'available' phrase."
    evidence_excerpt: "Ready for deliveries  (online)  vs  Standing by  (waiting)  vs  Waiting nearby  (no-deliveries)"
    recommendation: "Pick one state-vocab: 'Online · ready' / 'Online · no deliveries' / 'Offline'. Don't introduce three near-synonyms for the same idea."
    depends_on: null

  - finding_id: TW-143
    surface_id: dp-ux-dashboard-available-waiting
    lens: technical-writer
    severity: P3
    issue: "'Waiting nearby' — relates to TW-142. Two-word phrase suggests 'partners are nearby' but the actual state is 'available deliveries are nearby'."
    evidence_excerpt: "Waiting nearby"
    recommendation: "'Deliveries nearby' or merge into one status line with count."
    depends_on: TW-142

  - finding_id: TW-144
    surface_id: dp-ux-dashboard-standing-by
    lens: technical-writer
    severity: P3
    issue: "'Standing by' — driver-facing but ambiguous (military-ish phrase). 'Online' or 'Waiting for orders' clearer."
    evidence_excerpt: "Standing by"
    recommendation: "Pick one. 'Waiting for orders' (3 words) is plain English."
    depends_on: TW-142

  - finding_id: TW-145
    surface_id: dp-ux-staff-invite-created
    lens: technical-writer
    severity: P3
    issue: "'Invitation Created' / 'Share this link with the new staff member:' — title case + trailing colon. Both flagged elsewhere."
    evidence_excerpt: "Invitation Created + Share this link with the new staff member:"
    recommendation: "'Invitation sent. Share this link with the teammate.' Drops colon, past tense, drops 'new' (implicit)."
    depends_on: TW-039

  - finding_id: TW-146
    surface_id: ap-secsettings-session-fields
    lens: technical-writer
    severity: P3
    issue: "'Access token TTL (hours)' / 'Refresh token TTL (days)' — TTL is dev jargon. Style guide §2 plain English."
    evidence_excerpt: "Access token TTL (hours); Refresh token TTL (days); Save policy; Your active sessions; No active sessions."
    recommendation: "'Session timeout (hours)' / 'Refresh window (days)'. Even on admin surface, plainer is better."
    depends_on: TW-061

  - finding_id: TW-147
    surface_id: ap-secsettings-apikey-fields
    lens: technical-writer
    severity: P3
    issue: "'Expires in (days; 0 = never)' — inline programming convention 'use 0 to mean never' is dev jargon. Replace with a checkbox 'Never expires'."
    evidence_excerpt: "Expires in (days; 0 = never)"
    recommendation: "Field: 'Expires in (days)' + adjacent checkbox 'Never expires'. Don't embed sentinel values in the helper."
    depends_on: null

  - finding_id: TW-148
    surface_id: ap-platsettings-hours-fields
    lens: technical-writer
    severity: P3
    issue: "'Opening time (HH:MM)' / 'Closing time (HH:MM)' — format hint in label, dev style. Style guide §6 routes time formatting via Intl, not hand-rolled hints."
    evidence_excerpt: "Timezone (IANA); Opening time (HH:MM); Closing time (HH:MM); Operating days; Closed message"
    recommendation: "Use a time-picker UI. Label stays 'Opens at' / 'Closes at'. Drop '(HH:MM)' hint."
    depends_on: null

  - finding_id: TW-149
    surface_id: ap-providercreate-fields
    lens: technical-writer
    severity: P3
    issue: "Long field list mixes plain ('Name') with jargon ('API Base URL', 'Pricing Model', 'Max Concurrent Deliveries', 'Avg Pickup Time (min)'). Admin tone is OK with jargon, but 'Max Concurrent Deliveries' could be 'Concurrent delivery limit' (parallel with 'Daily limit' below)."
    evidence_excerpt: "Max Concurrent Deliveries; Daily Limit; Avg Pickup Time (min)"
    recommendation: "'Concurrent delivery limit' / 'Daily limit' / 'Average pickup time (min)'. Drop 'Max' / 'Avg' abbreviations. Sentence case."
    depends_on: null

  - finding_id: TW-150
    surface_id: dp-ux-partner-performance
    lens: technical-writer
    severity: P3
    issue: "'CSAT Score' uses CSAT acronym without definition; admin reader may not know."
    evidence_excerpt: "Acceptance Rate / On-Time Rate / CSAT Score / Total Deliveries"
    recommendation: "'Customer satisfaction (CSAT)' or simply 'Satisfaction score'. Sentence case."
    depends_on: null

  - finding_id: TW-151
    surface_id: ap-notifsettings-subtitle
    lens: technical-writer
    severity: P3
    issue: "'Control which channels deliver each notification category for your account. Transactional order updates stay on by default.' — 21 words, two sentences. Style guide §1 short sentences, §5 plain language."
    evidence_excerpt: "Control which channels deliver each notification category for your account. Transactional order updates stay on by default."
    recommendation: "'Choose which channels deliver each notification. Order updates stay on by default.' (15 words)."
    depends_on: null

  - finding_id: TW-152
    surface_id: ap-notifsettings-categories
    lens: technical-writer
    severity: P3
    issue: "Category descriptions use parenthetical aside '(chef accounts only)' which is admin-internal detail. Audit if this is needed in the UI label or belongs in a tooltip."
    evidence_excerpt: "Chef activity / New orders and verification outcomes (chef accounts only)."
    recommendation: "Move '(chef accounts only)' to a tooltip or section caption — don't pollute the row label."
    depends_on: null

  - finding_id: TW-153
    surface_id: mv-onb-personal-cta-next
    lens: technical-writer
    severity: P3
    issue: "Primary CTA 'Next' across onboarding steps 1-5 — fine for navigation, but final step uses 'Submit Application' (mv-onb-review-cta). Inconsistent ending could be 'Continue' across all, then 'Submit application' as the final step."
    evidence_excerpt: "Next  (steps 1-5)  vs  Submit Application  (review step)"
    recommendation: "Onboarding navigation: 'Continue' (steps 1-5), 'Submit application' (final). Replaces 'Next' which feels mechanical."
    depends_on: null

  - finding_id: TW-154
    surface_id: web-ux-admin-layout-nav
    lens: technical-writer
    severity: P3
    issue: "Aria labels 'Main navigation / Close navigation / Open navigation / Search... / Notifications, 3 unread' — hardcoded '3 unread' count is not safe for state changes; should be parameterised."
    evidence_excerpt: "Log out / Main navigation / Close navigation / Open navigation / Search... / Notifications, 3 unread"
    recommendation: "Parameterise count: 'Notifications, {n} unread' via ICU plural. Apply across all aria-labels."
    depends_on: null

  - finding_id: TW-155
    surface_id: web-ux-chef-layout-nav
    lens: technical-writer
    severity: P3
    issue: "Aria labels 'Notifications, unread' — no count or pluralisation. Less useful for screen-reader users than the admin equivalent."
    evidence_excerpt: "Main navigation / Close navigation / Open navigation / Notifications, unread"
    recommendation: "'Notifications, {n} unread' parameterised. Or simply 'Notifications · {n} unread'."
    depends_on: TW-154

  - finding_id: TW-156
    surface_id: web-ux-delivery-layout-toggles
    lens: technical-writer
    severity: P3
    issue: "Aria-label 'Currently online — tap to toggle availability' — 'Currently' filler; 'tap' is mobile-centric (web has clicks)."
    evidence_excerpt: "Currently online — tap to toggle availability / Notifications, unread"
    recommendation: "'Online — toggle availability'. Drops 'currently' and mobile-only verb."
    depends_on: null

  - finding_id: TW-157
    surface_id: web-ux-layout-nav
    lens: technical-writer
    severity: P3
    issue: "Top-nav primary destinations 'Home / Browse Chefs / Favorites / Catering / Profile / My Orders / Settings / Logout' — mixed case + 'Home' is the home page, but 'Browse Chefs' and 'Catering' compete for top-level slot. 'My Orders' possessive 'My' (TW-052). 'Logout' banned (TW-043)."
    evidence_excerpt: "Home / Browse Chefs / Favorites / Catering / Profile / My Orders / Settings / Logout"
    recommendation: "'Home / Browse / Favourites / Catering / Profile / Orders / Settings / Sign out'. Sentence case, no possessive, banned-vocab fix, US→UK spelling for en-IN."
    depends_on: TW-058

  - finding_id: TW-158
    surface_id: vp-ux-layout-nav
    lens: technical-writer
    severity: P3
    issue: "Sidebar nav 'Dashboard / Menu / Orders / Earnings / Admin Requests / Reviews / Analytics / Profile / Settings / Vendor Portal / Vendor / Settings / Logout / Theme' — duplicated 'Settings'? Mixed levels? 'Vendor Portal' and 'Vendor' suggest brand vs role labels mixed."
    evidence_excerpt: "Dashboard / Menu / Orders / Earnings / Admin Requests / Reviews / Analytics / Profile / Settings / Vendor Portal / Vendor / Settings / Logout / Theme"
    recommendation: "Audit nav hierarchy. 'Settings' should appear once. 'Vendor Portal' should be the brand mark (separate from nav). Apply 'Sign out' fix. Sentence case throughout."
    depends_on: TW-043

  - finding_id: TW-159
    surface_id: ap-layout-topbar-title
    lens: technical-writer
    severity: P3
    issue: "'Fe3dr Administration' — brand 'Fe3dr' next to 'Administration' is the wrong product name? Inventory notes brand mark 'Fe3dr'. Audit if this is Home Chef admin or a different product."
    evidence_excerpt: "Fe3dr Administration"
    recommendation: "Audit brand name across products. If 'Fe3dr' is correct, title becomes 'Fe3dr admin' (drop 'Administration' verbose noun)."
    depends_on: null

  - finding_id: TW-160
    surface_id: dp-ux-logo-text
    lens: technical-writer
    severity: P3
    issue: "Logo text 'Fe3dr + Delivery (tagline)' — same brand drift question as TW-159."
    evidence_excerpt: "Fe3dr + Delivery"
    recommendation: "Verify brand. Tagline 'Delivery' is too generic — consider 'Delivery operations' or product subline."
    depends_on: TW-159

  - finding_id: TW-161
    surface_id: web-ux-chef-dashboard-sections
    lens: technical-writer
    severity: P3
    issue: "'Today at a glance / Pending orders / Shortcuts / Recent orders' — mostly sentence case (good). 'Shortcuts' generic (TW-064 pattern)."
    evidence_excerpt: "Today at a glance / Pending orders / Shortcuts / Recent orders"
    recommendation: "Replace 'Shortcuts' with 'Quick actions' for cross-app consistency. Rest is fine."
    depends_on: TW-064

  - finding_id: TW-162
    surface_id: web-ux-chef-orders-heading
    lens: technical-writer
    severity: P3
    issue: "'Delivery Information / Order Items / Special Instructions / Payment Summary' — title case + 'Order Items' redundant per TW-062."
    evidence_excerpt: "Orders / Search by order number... / No orders found / Delivery Information / Order Items / Special Instructions / Payment Summary"
    recommendation: "'Delivery / Items / Special instructions / Payment summary'."
    depends_on: TW-062

  - finding_id: TW-163
    surface_id: dp-ux-active-h1
    lens: technical-writer
    severity: P3
    issue: "Page H1 'Active Delivery' — title case. Single-word sufficient ('Active') for driver tone."
    evidence_excerpt: "Active Delivery"
    recommendation: "'Active delivery' (sentence case) or 'Active' (single word, telegraphic)."
    depends_on: TW-033

  - finding_id: TW-164
    surface_id: dp-ux-available-h1
    lens: technical-writer
    severity: P3
    issue: "Page H1 'Available Deliveries' — title case + plural. Match driver tab label 'Available'."
    evidence_excerpt: "Available Deliveries"
    recommendation: "'Available deliveries' (sentence case) or 'Available' to match tab."
    depends_on: TW-033

  - finding_id: TW-165
    surface_id: dp-ux-available-count
    lens: technical-writer
    severity: P3
    issue: "'{n} orders waiting for pickup' — 5 words, plural-aware concatenation. ICU plural recommended."
    evidence_excerpt: "{n} orders waiting for pickup"
    recommendation: "'{n, plural, one {# order} other {# orders}} waiting for pickup.' Trailing period for sentence."
    depends_on: null

  - finding_id: TW-166
    surface_id: dp-ux-accept-delivery-btn
    lens: technical-writer
    severity: P3
    issue: "'Accept Delivery / Accepting...' — title case + three-dot ellipsis (TW-040). 'Accept' alone might work but specificity helps."
    evidence_excerpt: "Accept Delivery / Accepting..."
    recommendation: "'Accept' (1 word) idle / 'Accepting…' (Unicode ellipsis) loading. Or full 'Accept delivery' sentence case if word count tolerated."
    depends_on: TW-040

  - finding_id: TW-167
    surface_id: md-core-022
    lens: technical-writer
    severity: P3
    issue: "Mobile driver 'Accept Delivery' — title case. Matches dp-ux-accept-delivery-btn drift."
    evidence_excerpt: "Accept Delivery"
    recommendation: "'Accept' (1 word) or 'Accept delivery' (sentence case)."
    depends_on: TW-166

  - finding_id: TW-168
    surface_id: md-core-008
    lens: technical-writer
    severity: P3
    issue: "'Receiving delivery requests' — 3 words, OK as online subtitle. Compare md-core-009 'Toggle to start receiving requests' (TW-095) for offline. Parallel structure preferred."
    evidence_excerpt: "Receiving delivery requests  (online)"
    recommendation: "Match offline sibling structure: 'Online · ready for requests' / 'Offline · tap to go online'."
    depends_on: TW-095

  - finding_id: TW-169
    surface_id: dp-ux-staff-invite-btn
    lens: technical-writer
    severity: P3
    issue: "'Invite Staff' — title case. 2 words, acceptable length."
    evidence_excerpt: "Invite Staff"
    recommendation: "'Invite staff' (sentence case)."
    depends_on: TW-033

  - finding_id: TW-170
    surface_id: ap-staff-invite-fields
    lens: technical-writer
    severity: P3
    issue: "Form labels 'Email; Role; Department; Title; Personal Message; Send Invitation; Done; Copy; Copied' — buttons 'Send Invitation' / 'Done' / 'Copy' / 'Copied' mixed in same surface as form labels (Email, Role). Title case throughout."
    evidence_excerpt: "Email; Role; Department; Title; Personal Message; Send Invitation; Done; Copy; Copied"
    recommendation: "Sentence case all. 'Personal message' / 'Send invitation' / 'Copied.' (period for past-tense success per TW-053)."
    depends_on: TW-032

  - finding_id: TW-171
    surface_id: ap-providerdetail-status-fields
    lens: technical-writer
    severity: P3
    issue: "'Never' / 'Unlimited' — single-word status fillers. Need context."
    evidence_excerpt: "Enabled; Yes; No; Active; Inactive; Last Used; Created; Never; Unlimited"
    recommendation: "Pair with field label: 'Last used · Never' / 'Daily limit · Unlimited'. Don't show 'Never' / 'Unlimited' as bare values."
    depends_on: TW-105

  - finding_id: TW-172
    surface_id: vp-ux-menu-view-page
    lens: technical-writer
    severity: P3
    issue: "Long status-and-data surface includes 'Under Review' / 'Available' / 'Unavailable' / 'Featured' / 'Approved' — five state values mixed without grouping. Approval status (Approved / Under Review) and availability (Available / Unavailable / Featured) are different axes."
    evidence_excerpt: "Approved / Under Review / Available / Unavailable / Featured / Edit Item"
    recommendation: "Group: 'Approval · Approved / Under review' AND 'Availability · Available / Unavailable / Featured'. Two distinct axes, two distinct labels."
    depends_on: null

  - finding_id: TW-173
    surface_id: mv-orders-tab-live
    lens: technical-writer
    severity: P3
    issue: "Tab label 'Live Queue' — 'queue' is internal jargon. Most chef apps say 'Live orders' (matches vp-ux-orders-live-title)."
    evidence_excerpt: "Live Queue"
    recommendation: "'Live' (single word matching companion 'History') or 'Live orders' (matches vendor-portal)."
    depends_on: null

  - finding_id: TW-174
    surface_id: vp-ux-bottom-nav
    lens: technical-writer
    severity: P3
    issue: "Mobile bottom nav 'Dashboard / Menu / Orders / Earnings / Profile / Vendor navigation' — last item 'Vendor navigation' is an aria-label inside the visible-text list, looks like a label leak."
    evidence_excerpt: "Dashboard / Menu / Orders / Earnings / Profile / Vendor navigation"
    recommendation: "Verify 'Vendor navigation' is aria-label only, not visible. If it's accidentally visible, hide. If it's a 5th tab label, rename."
    depends_on: null

  - finding_id: TW-175
    surface_id: dp-ux-history-subtitle
    lens: technical-writer
    severity: P3
    issue: "'Your past deliveries' — 3 words, fine. Marginal: 'Past deliveries' (drop 'Your', possessive implicit on driver portal)."
    evidence_excerpt: "Your past deliveries"
    recommendation: "'Past deliveries' (2 words)."
    depends_on: TW-052

  - finding_id: TW-176
    surface_id: dp-ux-earnings-subtitle
    lens: technical-writer
    severity: P3
    issue: "'Track your delivery income' — 4 words, 'Your' filler (TW-052). 'Income' fine but 'earnings' / 'payouts' more common."
    evidence_excerpt: "Track your delivery income"
    recommendation: "'Earnings and payouts' (3 words) or drop subtitle entirely on driver surface."
    depends_on: TW-052

  - finding_id: TW-177
    surface_id: dp-ux-stat-rating
    lens: technical-writer
    severity: P3
    issue: "'Rating' + '{n} reviews' / 'No reviews yet' — empty state pattern OK. Consider pairing CTA 'Order to leave a review' (customer-facing)."
    evidence_excerpt: "Rating + {n} reviews / No reviews yet"
    recommendation: "Keep. Acceptable. Minor: 'No reviews yet.' (period for sentence ending)."
    depends_on: TW-051

  - finding_id: TW-178
    surface_id: ap-secsettings-2fa-account
    lens: technical-writer
    severity: P3
    issue: "Inline state copy '— login requires...' / '— enable to protect...' uses 'login' noun (banned, TW-044). Also em-dash style — verify it's the typographic em-dash, not hyphen-minus."
    evidence_excerpt: "Enabled — login requires a 6-digit code; Disabled — enable to protect your admin account"
    recommendation: "See TW-106. Replace 'login' with 'sign-in'."
    depends_on: TW-106

  - finding_id: TW-179
    surface_id: ap-platsettings-cards
    lens: technical-writer
    severity: P3
    issue: "Card subtitles 'Service fee, tax, and payout percentages' / 'Base fee + per-km rate applied at checkout' / 'When the platform accepts new orders' / 'Delivery zones — orders outside a zone are rejected' — mixed sentence-fragment vs full-sentence. Pick one."
    evidence_excerpt: "Commission rates / Service fee, tax, and payout percentages; Delivery fees / Base fee + per-km rate applied at checkout; Operating hours / When the platform accepts new orders; Service areas / Delivery zones — orders outside a zone are rejected"
    recommendation: "Fragments OK as subtitles (more scannable). Standardise: 'Service fee, tax, and payout percentages.' / 'Base fee plus per-km rate at checkout.' / 'When the platform accepts new orders.' / 'Delivery zones · orders outside a zone are rejected.' Trailing periods on full subtitles."
    depends_on: null

  - finding_id: TW-180
    surface_id: mc-checkout-totals
    lens: technical-writer
    severity: P3
    issue: "'Free' as a row value alongside 'Subtotal / Delivery fee / Total' — when delivery is free, the row should show '₹0' or 'Free' but the standalone 'Free' in the inventory excerpt suggests inconsistent UI placement."
    evidence_excerpt: "Subtotal / Delivery fee / Free / Total"
    recommendation: "When delivery free: row 'Delivery fee · Free' (right-aligned, on same line as label). Don't show 'Free' as a standalone row."
    depends_on: null

  - finding_id: TW-181
    surface_id: md-core-080
    lens: technical-writer
    severity: P3
    issue: "Detail rows 'Distance / Completed / Payout' — single-word labels OK. No issue."
    evidence_excerpt: "Distance / Completed / Payout"
    recommendation: "Keep as-is."
    depends_on: null

  - finding_id: TW-182
    surface_id: md-core-084
    lens: technical-writer
    severity: P3
    issue: "'Phone / City / Email' — single-word field labels, fine. No issue."
    evidence_excerpt: "Phone / City / Email"
    recommendation: "Keep as-is."
    depends_on: null

  - finding_id: TW-183
    surface_id: md-core-090
    lens: technical-writer
    severity: P3
    issue: "'Edit / Cancel' action set — single words, fine."
    evidence_excerpt: "Edit / Cancel"
    recommendation: "Keep as-is. (Note: 'Cancel' is overloaded across the app — sometimes 'Cancel order', sometimes form-cancel. Verify aria-labels disambiguate.)"
    depends_on: null

  - finding_id: TW-184
    surface_id: vp-ux-payouts-title
    lens: technical-writer
    severity: P3
    issue: "Back-link 'Back to Earnings' — title case. Sentence case."
    evidence_excerpt: "Payout History / View all your past and pending payouts / Back to Earnings"
    recommendation: "'Back to earnings'. Lowercase. Or icon-only arrow with aria-label."
    depends_on: TW-034

  - finding_id: TW-185
    surface_id: vp-ux-menu-view-page
    lens: technical-writer
    severity: P3
    issue: "'Back to Menu' — title case (TW-034 pattern)."
    evidence_excerpt: "Back to Menu"
    recommendation: "'Back to menu' (sentence case)."
    depends_on: TW-034

  - finding_id: TW-186
    surface_id: dp-ux-partner-detail-back
    lens: technical-writer
    severity: P3
    issue: "'Back to Partners' — title case + plural noun."
    evidence_excerpt: "Back to Partners"
    recommendation: "'Back to partners' or 'Back' (icon + aria-label)."
    depends_on: TW-033

  - finding_id: TW-187
    surface_id: vp-ux-kitchen-title
    lens: technical-writer
    severity: P3
    issue: "'Back to Profile' — title case (TW-034 pattern). Combined into the header with multiple actions ('Kitchen Setup / Save Changes / Back to Profile')."
    evidence_excerpt: "Kitchen Setup / Operating hours, photos, and payout details / Save Changes / Back to Profile"
    recommendation: "'Back to profile' / 'Save changes' / 'Kitchen setup'. Sentence case across the header."
    depends_on: TW-034

  - finding_id: TW-188
    surface_id: ap-platsettings-zones-fields
    lens: technical-writer
    severity: P3
    issue: "'New zone / Create zone / Cancel' — 'Create zone' acceptable but 'New zone' as label/heading + 'Create zone' as button is redundant."
    evidence_excerpt: "Name; City; State; Min latitude; Max latitude; Min longitude; Max longitude; New zone; Create zone; Cancel"
    recommendation: "Modal title 'Add zone'. Submit button 'Create zone' OR drop 'New zone' if it's a section heading next to a 'Create zone' button."
    depends_on: TW-055

  - finding_id: TW-189
    surface_id: ap-providercreate-fields
    lens: technical-writer
    severity: P3
    issue: "'Logo URL' / 'API Base URL' / 'Webhook Secret' — admin jargon acceptable but title case is inconsistent with other admin fields. Apply TW-032 fix."
    evidence_excerpt: "Logo URL; API Base URL; API Key; API Secret; Webhook Secret"
    recommendation: "Sentence case (TW-032): 'Logo URL' (stays acronym-cap) / 'API base URL' / 'API key' / 'API secret' / 'Webhook secret'."
    depends_on: TW-032

  - finding_id: TW-190
    surface_id: ap-delivery-stats
    lens: technical-writer
    severity: P3
    issue: "'{n} online now; Active deliveries; Delivered today; Today's payouts' — mix of inline-numeral pattern ('{n} online now') and noun-only ('Active deliveries'). Pair structure throughout: count + noun."
    evidence_excerpt: "{n} online now; Active deliveries; Delivered today; Today's payouts"
    recommendation: "All four use count + noun: '{n} online' / '{n} active' / '{n} delivered today' / '₹{n} payouts today'."
    depends_on: null

  - finding_id: TW-191
    surface_id: ap-dashboard-shortcuts-items
    lens: technical-writer
    severity: P3
    issue: "'Manage users / View all users; Chef verification / Review applications; Order management / Track all orders; Analytics / View reports' — title + subtitle pairs with slash separators. Confusing format unless layout makes it clear."
    evidence_excerpt: "Manage users / View all users; Chef verification / Review applications; Order management / Track all orders; Analytics / View reports"
    recommendation: "Verify UI uses separate title and subtitle. Strings should not concatenate via '/'. 'Chef verification' + 'Review applications' as two strings."
    depends_on: null

  - finding_id: TW-192
    surface_id: ap-userdetail-order-stats
    lens: technical-writer
    severity: P3
    issue: "'Total Orders / Total Spent / Last Order / No orders yet' — title case + 'No orders yet' empty-state without CTA per TW-051."
    evidence_excerpt: "Order Activity; Total Orders; Total Spent; Last Order; No orders yet"
    recommendation: "Sentence case. Empty state: 'No orders yet.' (terminal period; this is a user detail page so no CTA needed for admin)."
    depends_on: TW-032

  - finding_id: TW-193
    surface_id: vp-ux-orders-history-summary
    lens: technical-writer
    severity: P3
    issue: "Summary cards 'Delivered / Cancelled / Revenue' — single nouns, fine. 'Revenue' refers to chef's earnings (see TW-050)."
    evidence_excerpt: "Delivered / Cancelled / Revenue"
    recommendation: "'Delivered / Cancelled / Earnings' (replace 'Revenue' for chef-facing — TW-050)."
    depends_on: TW-050

  - finding_id: TW-194
    surface_id: web-ux-chef-earnings
    lens: technical-writer
    severity: P3
    issue: "'Earnings / Earnings Overview / Top Selling Items / Recent Payouts / Payment Settings' — title case + 'Earnings Overview' redundant ('Overview' under 'Earnings' page is implicit)."
    evidence_excerpt: "Earnings / Earnings Overview / Top Selling Items / Recent Payouts / Payment Settings"
    recommendation: "'Earnings / Overview / Top sellers / Recent payouts / Payment settings'. Drop redundant 'Earnings' prefix on first section."
    depends_on: TW-031

  - finding_id: TW-195
    surface_id: web-ux-admin-analytics
    lens: technical-writer
    severity: P3
    issue: "'Analytics / Revenue Overview / Orders by Status / Top Performing Chefs / Popular Cuisines' — title case throughout. 'Top Performing Chefs' wordy ('Top chefs' suffices)."
    evidence_excerpt: "Analytics / Revenue Overview / Orders by Status / Top Performing Chefs / Popular Cuisines"
    recommendation: "'Analytics / Revenue overview / Orders by status / Top chefs / Popular cuisines'. Sentence case."
    depends_on: TW-031

  - finding_id: TW-196
    surface_id: mv-orders-tab-history
    lens: technical-writer
    severity: P3
    issue: "Tab 'History' — single word, fine. Pair label 'Live Queue' (TW-173) should match — recommend 'Live' single word."
    evidence_excerpt: "History"
    recommendation: "Keep 'History'. Update sibling tab to 'Live' (TW-173)."
    depends_on: TW-173

  - finding_id: TW-197
    surface_id: md-core-068
    lens: technical-writer
    severity: P3
    issue: "'App Version' footer label — title case (TW-037 pattern). One-time fix."
    evidence_excerpt: "App Version"
    recommendation: "'App version' (sentence case)."
    depends_on: TW-037

  - finding_id: TW-198
    surface_id: dp-ux-active-order-note
    lens: technical-writer
    severity: P3
    issue: "Inventory notes 'chef-supplied instructions' — content from another role can include strings the app doesn't control. Display chef notes verbatim, but consider character limit + escape rendering."
    evidence_excerpt: "Note: {specialInstructions}"
    recommendation: "Add character cap (e.g., 500 chars) on chef-supplied notes. Render as plain text (no markdown / HTML). Out of scope for TW, but flag for engineering."
    depends_on: TW-114

  - finding_id: TW-199
    surface_id: dp-ux-active-status-picked-up
    lens: technical-writer
    severity: P3
    issue: "'Picked Up' status — title case 'Picked Up' (TW-033 + TW-081)."
    evidence_excerpt: "Picked Up"
    recommendation: "'Picked up' (sentence case)."
    depends_on: TW-081

  - finding_id: TW-200
    surface_id: dp-ux-active-status-in-transit
    lens: technical-writer
    severity: P3
    issue: "'In Transit' — title case."
    evidence_excerpt: "In Transit"
    recommendation: "'In transit'."
    depends_on: TW-081
```

## Legal findings

```yaml
# Legal lens findings — CORE-UX category (594 surfaces)
# Scope: India regulatory exposure (DPDP Act 2023, FSSAI 2011/2018, RBI PA MD 2020/2024, GST/CGST 2017,
# Consumer Protection Act 2019, IT Act 2000/IT Rules 2021, MV Act 1988 for drivers) + generic best-practice
# IMPORTANT: Every finding carries depends_on: "needs lawyer review" — this audit is not legal advice.
# Focus: transaction-critical surfaces (checkout, cart, order detail, payouts, refunds, audit log,
# delivery promises, reviews, filters with implied claims, allergens, KYC, driver gig classification).

findings:

  # ============================================================================
  # CHECKOUT — contract-formation moment (highest exposure)
  # ============================================================================

  - finding_id: LEG-COREUX-001
    surface_id: web-ux-checkout-heading
    lens: legal
    severity: P0
    issue: "Checkout page H1 is the entire page contract-formation surface but no inline T&C / Refund Policy / Cancellation Policy acknowledgement is rendered at the 'Place Order' moment in this excerpt"
    evidence_excerpt: "Checkout"
    recommendation: "Surface a one-line, lawyer-reviewed disclosure adjacent to Place Order: 'By placing this order you accept our Terms, Refund Policy (refunds in ≤X working days), Cancellation Policy, and confirm you have read allergen info.' Required for enforceable contract under Indian Contract Act §10, and for RBI PA refund-timeline disclosure at point of payment."
    citation: "Indian Contract Act 1872 §10; RBI Payment Aggregator MD (17 Mar 2020, upd. 2024); Consumer Protection (E-Commerce) Rules 2020 r.5(3)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-002
    surface_id: web-ux-checkout-payment-section
    lens: legal
    severity: P0
    issue: "Payment section discloses 'Powered by Razorpay' but does not identify the merchant of record, the payment aggregator's role, or the refund timeline before card capture"
    evidence_excerpt: "Payment / Powered by Razorpay / Pay securely via UPI, cards, net banking, or wallets"
    recommendation: "RBI PA MD requires merchant-of-record clarity. Add: identity of the merchant (Home Chef platform vs the individual chef), Razorpay's role as licensed PA, refund timeline (≤T+1 instructed, settlement framework), and the chargeback / dispute pathway. Cite RBI PA framework expressly."
    citation: "RBI Payment Aggregator MD 17 Mar 2020 §6 (Merchant On-boarding) and §8 (Settlement & Escrow)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-003
    surface_id: web-ux-checkout-summary
    lens: legal
    severity: P0
    issue: "Order Summary surfaces 'Tax' as an opaque line — no GST registration number, no CGST/SGST/IGST split, no HSN/SAC code reference, no statement of whether prices are tax-inclusive or tax-extra"
    evidence_excerpt: "Subtotal / Delivery fee / Service fee / Tax / Tip / Total / Place Order"
    recommendation: "Under CGST Rules 2017 r.46, every taxable supply requires a tax invoice with GSTIN, HSN/SAC code, and tax-rate breakup; the checkout summary must at minimum signal that a compliant invoice will be issued and indicate tax-inclusivity. Replace single 'Tax' line with 'GST (X%)' and link to a sample compliant invoice. B2B GST-input claimers need GSTIN capture path."
    citation: "CGST Act 2017 §31; CGST Rules 2017 r.46; FAQ on E-Commerce Operator (CBIC)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-004
    surface_id: web-ux-checkout-delivery-time
    lens: legal
    severity: P0
    issue: "Delivery-time selector exposes 'Usually 30-45 minutes' as an implicit time guarantee — creates Consumer Protection Act exposure if not met, because the platform is making a temporal representation at point of sale"
    evidence_excerpt: "Delivery Time / As soon as possible / Usually 30-45 minutes / Schedule for later"
    recommendation: "Either (a) qualify as a non-binding estimate ('Estimated — actual time varies with traffic and chef prep') and link to remedy if breached, or (b) make it a binding SLA with explicit refund/discount mechanic if breached. Bald promise without qualifier is a deficiency-of-service claim risk under CPA 2019 §2(11)."
    citation: "Consumer Protection Act 2019 §2(11), §2(47); CP (E-Commerce) Rules 2020 r.5(4)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-005
    surface_id: web-ux-checkout-tip
    lens: legal
    severity: P1
    issue: "Tip language '100% of your tip goes to the home chef' is an absolute representation — if any platform fee, GST, or processing deduction is taken from the tip transit, this becomes a misleading representation"
    evidence_excerpt: "Add a tip / 100% of your tip goes to the home chef / No tip / Custom"
    recommendation: "Lawyer review of literal accuracy. If TDS, GST on commission, or any deduction touches the tip flow, replace with 'Tips are passed to the chef (subject to applicable tax withholding)'. Misleading-representation exposure under CPA 2019 §2(28) and CCPA Guidelines on Misleading Ads 2022."
    citation: "Consumer Protection Act 2019 §2(28); CCPA Guidelines for Prevention of Misleading Ads 2022 r.4"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-006
    surface_id: web-ux-checkout-address-form
    lens: legal
    severity: P1
    issue: "Address form collects PII (street, city, state, postal code) at checkout with no inline DPDP notice of purpose, retention period, or grievance officer reference"
    evidence_excerpt: "Delivery Address / Add New / Label / Street Address / Apartment, suite, etc. (optional) / City / State / Postal Code / Save Address"
    recommendation: "DPDP Act §5 requires notice 'at or before' personal-data collection. Add a short notice link near Save Address: 'We use this address to deliver your order. Retention: X months after order. See Privacy Notice and contact our Grievance Officer at <email>.'"
    citation: "DPDP Act 2023 §5 (Notice), §13 (Grievance Officer)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-007
    surface_id: mc-checkout-totals
    lens: legal
    severity: P0
    issue: "Mobile checkout totals show only 'Subtotal / Delivery fee / Free / Total' — no tax line at all on the customer mobile app, breaking GST invoice expectations"
    evidence_excerpt: "Subtotal / Delivery fee / Free / Total"
    recommendation: "Mobile checkout must display the GST line and tax-inclusivity status before Place Order. Receipts/invoices must comply with CGST Rules r.46. Missing tax line is a P0 compliance gap."
    citation: "CGST Act 2017 §31; CGST Rules 2017 r.46"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-008
    surface_id: mc-checkout-place-order
    lens: legal
    severity: P0
    issue: "Mobile 'Place Order · ₹{total}' CTA has no T&C, Refund Policy, or Cancellation Policy acknowledgement adjacent — contract-formation without disclosed terms"
    evidence_excerpt: "Place Order · ₹{total} / Processing..."
    recommendation: "Add a one-line disclosure above CTA referencing Terms, Refund Policy, Cancellation Policy. Required for enforceable contract and for RBI PA refund-timeline disclosure at payment."
    citation: "Indian Contract Act 1872 §10; RBI PA MD 2020"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-009
    surface_id: mc-checkout-section-address
    lens: legal
    severity: P1
    issue: "Mobile address capture step has no DPDP notice or purpose statement"
    evidence_excerpt: "Delivery Address"
    recommendation: "Inline DPDP §5 notice with purpose, retention, and grievance officer reference."
    citation: "DPDP Act 2023 §5"
    depends_on: "needs lawyer review"

  # ============================================================================
  # CART — price display, T&C reference, minimum-order disclosure
  # ============================================================================

  - finding_id: LEG-COREUX-010
    surface_id: web-ux-cart-order-summary
    lens: legal
    severity: P1
    issue: "Cart summary lists 'Service fee', 'Delivery fee', 'Tax' as opaque line items without a tooltip or 'What is this?' explanation — Service fee in particular has been flagged in Indian consumer regulator advisories"
    evidence_excerpt: "Order Summary / Subtotal / Delivery fee / Service fee / Discount / Total"
    recommendation: "CCPA Aug-2024 guidance on dark patterns flags hidden / unexplained service fees. Add inline disclosure: who collects, what it is for, whether it is refundable. 'Service fee' without explanation may be a 'drip pricing' dark pattern."
    citation: "CCPA Guidelines for Prevention and Regulation of Dark Patterns 2023 r.3 (Drip Pricing); CPA 2019 §2(47)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-011
    surface_id: web-ux-cart-checkout-cta
    lens: legal
    severity: P2
    issue: "'Sign in to Checkout' enforces auth gate — but the moment a user starts adding to cart they may be giving consent to cookies/session storage with no banner shown beforehand"
    evidence_excerpt: "Sign in to Checkout / Proceed to Checkout"
    recommendation: "Cookie/tracking consent banner before any non-essential storage. DPDP read-with IT Rules 2021 requires informed consent for trackers."
    citation: "DPDP Act 2023 §6; IT Rules 2021 r.3(1)(b)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-012
    surface_id: mc-cartsheet-checkout-cta
    lens: legal
    severity: P1
    issue: "Mobile cart 'Proceed to Checkout' has no minimum-order, surge-pricing, or geo-availability disclosure before transition"
    evidence_excerpt: "Proceed to Checkout"
    recommendation: "If platform applies minimum order or surge pricing, disclose at cart, not after payment. CCPA dark-patterns guidance treats post-cart surprises as 'sneak into basket' / 'forced action' patterns."
    citation: "CCPA Dark Patterns Guidelines 2023 r.3, r.7"
    depends_on: "needs lawyer review"

  # ============================================================================
  # ORDER DETAIL / CANCELLATION / REFUND — post-order disclosures
  # ============================================================================

  - finding_id: LEG-COREUX-013
    surface_id: web-ux-orderdetail-sections
    lens: legal
    severity: P0
    issue: "Order detail anatomy shows 'Estimated delivery' as a representation but no refund/discount entitlement is shown if estimate is breached, and no GST-compliant invoice link is shown"
    evidence_excerpt: "Order #{n} / Placed on {date} / Estimated delivery: / Order Items / Delivery Address / Payment Summary / Special Instructions / Leave a Review"
    recommendation: "Add a 'View invoice' / 'Download GST invoice' link with CGST r.46 compliant tax invoice (GSTIN, HSN/SAC, tax breakup). Add explicit text on whether 'Estimated delivery' is binding or indicative. Add refund-timeline statement adjacent to Payment Summary."
    citation: "CGST Rules 2017 r.46; RBI PA MD 2020; CPA 2019 §2(11)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-014
    surface_id: web-ux-orderdetail-cancel
    lens: legal
    severity: P0
    issue: "Cancel modal asks 'why you're cancelling' but does not state refund eligibility, refund timeline, or who bears non-refundable costs at each order stage"
    evidence_excerpt: "Cancel Order / Please let us know why you're cancelling..."
    recommendation: "Show stage-dependent refund matrix BEFORE the cancel confirm: e.g. 'Cancellation before chef acceptance — full refund within 5-7 working days. After chef acceptance — partial refund. After dispatch — no refund.' Refund timeline must be explicit per RBI PA MD."
    citation: "RBI PA MD 2020; CPA (E-Commerce) Rules 2020 r.5(4)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-015
    surface_id: web-ux-orders-status-labels
    lens: legal
    severity: P1
    issue: "Status taxonomy includes 'Refunded' as a public-facing state but no associated refund-completion timestamp, payment-method, or transaction reference is documented in the inventory excerpt"
    evidence_excerpt: "Pending / Accepted / Preparing / Ready / Picked Up / On the Way / Delivered / Cancelled / Refunded"
    recommendation: "When showing 'Refunded' state, also surface refund amount, refund instrument (original method), refund initiation date, and expected credit window. RBI PA framework requires audit-trail visibility for refunds."
    citation: "RBI PA MD 2020 §8 (Settlement); Consumer Protection (E-Commerce) Rules 2020 r.6(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-016
    surface_id: ap-orders-filter-status
    lens: legal
    severity: P2
    issue: "Admin order filter shows 'Refunded' alongside 'Cancelled' but inventory notes 'refund-adjacent' — admin tooling must support full RBI-PA refund audit trail"
    evidence_excerpt: "All Status; Pending; Accepted; Preparing; Ready; Delivering; Delivered; Cancelled; Refunded"
    recommendation: "Lawyer review whether admin refund state captures: refund initiator, RBI PA reference, settlement batch, time-to-credit. Required for RBI PA audit and CCPA dispute pathways."
    citation: "RBI PA MD 2020 §6, §8"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-017
    surface_id: mc-order-detail-status-labels
    lens: legal
    severity: P1
    issue: "Mobile customer status labels do not include a 'Refunded' state ('Pending / Confirmed / Preparing / Ready for Pickup / On the Way / Delivered / Cancelled') — customer cannot self-serve refund status visibility"
    evidence_excerpt: "Pending / Confirmed / Preparing / Ready for Pickup / On the Way / Delivered / Cancelled"
    recommendation: "Add 'Refunded' state with timestamp and instrument disclosure. RBI PA framework expects customer-visible refund tracking."
    citation: "RBI PA MD 2020 §8; CP (E-Commerce) Rules 2020 r.6"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-018
    surface_id: mc-order-detail-price-rows
    lens: legal
    severity: P0
    issue: "Mobile order detail price breakdown shows 'Subtotal / Delivery Fee / Total' — no Tax/GST line, no service fee, no convenience fee disclosure"
    evidence_excerpt: "Subtotal / Delivery Fee / Total"
    recommendation: "Order detail receipt must comply with CGST r.46 — add GST line with rate, GSTIN, HSN/SAC for B2C food delivery."
    citation: "CGST Rules 2017 r.46"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-019
    surface_id: dp-ux-active-cancel-prompt
    lens: legal
    severity: P0
    issue: "Driver cancellation uses native browser prompt() with no terms reference, no audit trail capture by category, no customer-impact disclosure — inventory itself flags 'no UI polish'"
    evidence_excerpt: "Reason for cancellation?"
    recommendation: "Driver cancel must capture structured reason (food not ready, customer unreachable, accident, vehicle issue), with attribution stored in audit log, and customer-impact warning. Affects refund attribution and gig-worker performance metrics under any future Code on Social Security gig regulations."
    citation: "Code on Social Security 2020 (gig-worker provisions, when notified); CPA 2019 §2(11)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-020
    surface_id: dp-ux-active-cancel-btn
    lens: legal
    severity: P1
    issue: "Driver 'Cancel Delivery' is a destructive action with consumer-protection implications (refund triggers, chef payout adjustments) but no two-step confirmation flow is referenced in inventory"
    evidence_excerpt: "Cancel Delivery"
    recommendation: "Require typed reason + two-step confirm. Capture geolocation + timestamp for audit. Required for CPA / RBI PA chargeback rebuttals."
    citation: "CPA 2019 §2(11); RBI PA MD 2020 §6"
    depends_on: "needs lawyer review"

  # ============================================================================
  # MENU / DISH LISTINGS — FSSAI allergen and licence disclosure
  # ============================================================================

  - finding_id: LEG-COREUX-021
    surface_id: web-ux-chef-menu
    lens: legal
    severity: P0
    issue: "Chef Menu Management page has no field for FSSAI-mandated allergen declaration on individual dishes — only basic name/description shown in inventory"
    evidence_excerpt: "Menu Management / Search menu items... / No menu items / e.g., 500g"
    recommendation: "FSSAI Labelling & Display Regulations 2020 require allergen declaration for the eight major allergens on every prepared food item served to consumers. Add a mandatory allergen-tags field (cereals containing gluten, crustaceans, eggs, fish, peanuts, soybeans, milk, tree nuts, plus any other locally relevant). Also require chef's FSSAI registration number capture at kitchen onboarding (covered under chef profile but verify enforcement on menu)."
    citation: "FSSAI (Labelling and Display) Regulations 2020 r.5; FSS Act 2006 §31"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-022
    surface_id: vp-ux-menu-form-title
    lens: legal
    severity: P0
    issue: "Menu Item form page title shows 'Edit Menu Item / Add Menu Item' but inventory excerpt shows no allergen-tag field, no ingredient list field, no FSSAI compliance hint"
    evidence_excerpt: "Edit Menu Item / Add Menu Item / Update the details of your dish / Add a new dish to your menu"
    recommendation: "Form must enforce allergen-tag capture and ingredient declaration. FSSAI regs require ingredient-and-allergen disclosure on prepared food."
    citation: "FSSAI (Labelling and Display) Regulations 2020 r.5"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-023
    surface_id: vp-ux-menu-view-page
    lens: legal
    severity: P1
    issue: "Menu item read-only view shows 'Dietary Information / No dietary information provided' as a tolerated state — but for a regulated food platform, no dietary info shouldn't be a viable published state"
    evidence_excerpt: "Approved / Under Review / Available / Unavailable / Featured / Edit Item / Description / Dietary Information / No dietary information provided / Pricing / Preparation / Prep Time / Serves / Portion Size / Performance / Orders / Rating / Item ID"
    recommendation: "Make dietary/allergen information mandatory for publish. 'No dietary information provided' should block 'Approved' status."
    citation: "FSSAI (Labelling and Display) Regulations 2020 r.5"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-024
    surface_id: mc-home-cuisine-filters
    lens: legal
    severity: P2
    issue: "Cuisine filter chips do not include a veg/non-veg toggle or visual veg-mark — FSSAI mandates veg/non-veg symbols on every food product in India"
    evidence_excerpt: "All, North Indian, South Indian, Chinese, Continental, Italian, Healthy"
    recommendation: "FSSAI Regs r.2.2.2 require green-dot (veg) / brown-dot (non-veg) symbol on every food product. The customer-facing filter should reflect this. Mobile vendor inventory shows Veg/Non-Veg toggle at item level (mv-menunew-type-veg/-nonveg) — verify display compliance on customer-facing detail."
    citation: "FSS (Packaging and Labelling) Regulations 2011 r.2.2.2; FSSAI (Labelling and Display) Regulations 2020 r.5"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-025
    surface_id: mv-menunew-name-ph
    lens: legal
    severity: P3
    issue: "Mobile-vendor menu name placeholder 'e.g. Butter Chicken' encourages product naming without ingredient declaration — Butter Chicken contains dairy, an allergen"
    evidence_excerpt: "e.g. Butter Chicken"
    recommendation: "Cosmetic — ensure the form below the name field surfaces an allergen warning when dairy/eggs/nuts/gluten are detected by ingredient parser."
    citation: "FSSAI (Labelling and Display) Regulations 2020 r.5 (best-practice nudge)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-026
    surface_id: mv-menunew-desc-ph
    lens: legal
    severity: P2
    issue: "Description placeholder 'Describe your dish (at least 20 characters)' does not prompt for allergen / ingredient / spice-level disclosure — relies on free text"
    evidence_excerpt: "Describe your dish (at least 20 characters)"
    recommendation: "Replace placeholder with explicit nudge: 'Describe ingredients, spice level, and any allergens (dairy, nuts, gluten, soy, eggs).'"
    citation: "FSSAI (Labelling and Display) Regulations 2020 r.5"
    depends_on: "needs lawyer review"

  # ============================================================================
  # CHEF PROFILE / DETAIL — FSSAI licence display + verified-claim exposure
  # ============================================================================

  - finding_id: LEG-COREUX-027
    surface_id: web-ux-chefdetail-status-badges
    lens: legal
    severity: P0
    issue: "Chef detail page shows 'Verified' badge as a claim but inventory does not indicate what verification means (KYC? FSSAI? insurance? hygiene audit?). Implied-claim exposure"
    evidence_excerpt: "Verified / Open for orders / Currently closed / Min. order: / Delivery: / Price range:"
    recommendation: "CCPA Misleading Ads Guidelines 2022 require substantiation of objective claims. Add a tooltip/info icon: 'Verified means: government-ID checked, FSSAI registration confirmed, address checked.' Otherwise the badge is a misleading claim. Display chef FSSAI registration number on the profile."
    citation: "CCPA Guidelines for Prevention of Misleading Ads 2022 r.4, r.7; FSS Act 2006 §31; CPA 2019 §2(28)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-028
    surface_id: web-ux-chef-profile
    lens: legal
    severity: P0
    issue: "Chef profile editor includes 'Business Settings' and 'Basic Information' but the inventory does not surface a mandatory FSSAI registration number field at the chef-facing form"
    evidence_excerpt: "Kitchen Profile / Profile Images / Basic Information / Cuisines / Specialties / Business Settings / Your Stats / Tell customers about your kitchen..."
    recommendation: "FSS Act §31 requires FBOs (Food Business Operators) to be FSSAI-registered. Home Chef as platform must collect and display each chef's FSSAI registration number publicly. Add mandatory capture + display."
    citation: "FSS Act 2006 §31; FSS (Licensing and Registration) Regulations 2011"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-029
    surface_id: vp-ux-profile-docs
    lens: legal
    severity: P1
    issue: "Vendor-portal profile documents section 'Upload required documents for verification' does not enumerate what documents are required (FSSAI, PAN, GST, address proof, bank account, kitchen hygiene certificate)"
    evidence_excerpt: "Documents & Certificates / Upload required documents for verification"
    recommendation: "Enumerate required documents explicitly and bind each to a regulation: FSSAI registration (FSS Act §31), PAN (Income Tax), GSTIN if turnover > threshold, address proof. Without enumeration, KYC is non-deterministic."
    citation: "FSS Act 2006 §31; Income Tax Act 1961; RBI KYC Master Direction 2016"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-030
    surface_id: ap-approvaldetail-detail-fields
    lens: legal
    severity: P1
    issue: "Admin approval detail captures 'Terms Accepted / Hygiene Policy Accepted' as fields — but inventory shows no link to the actual policy text that was accepted or timestamp/version"
    evidence_excerpt: "Terms Accepted; Hygiene Policy Accepted; Document Type; File Name; Status; Document Number..."
    recommendation: "Acceptance records must capture: policy version hash, timestamp, IP, user-agent. Without version pinning, you cannot prove what the chef agreed to. Required for FSS Act enforcement defence and CPA disputes."
    citation: "Indian Contract Act 1872 §10; IT Act 2000 §65B (electronic evidence); FSS Act 2006"
    depends_on: "needs lawyer review"

  # ============================================================================
  # BROWSE / FILTERS — implied claims (Verified, Top Rated, Trending)
  # ============================================================================

  - finding_id: LEG-COREUX-031
    surface_id: web-ux-browse-sort-options
    lens: legal
    severity: P1
    issue: "Sort dropdown includes 'Top Rated' and 'Most Popular' — these are objective claims that require substantiation methodology disclosure"
    evidence_excerpt: "Top Rated / Nearest / Most Popular / Price"
    recommendation: "Add a 'Sort methodology' info icon: 'Top Rated = chefs with average rating ≥ X over last Y orders, min Z reviews. Most Popular = orders in last 30 days.' Otherwise misleading-claim exposure under CCPA."
    citation: "CCPA Guidelines for Prevention of Misleading Ads 2022 r.4"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-032
    surface_id: web-ux-browse-filters-rating
    lens: legal
    severity: P2
    issue: "Rating filter '4.5+ Stars / 4+ Stars / 3.5+ Stars' relies on rating data — must comply with BIS standard IS 19000:2022 on online consumer reviews (now applicable as voluntary standard)"
    evidence_excerpt: "Any Rating / 4.5+ Stars / 4+ Stars / 3.5+ Stars"
    recommendation: "Disclose review-collection process (verified purchases only? anonymity? moderation policy?). BIS IS 19000:2022 sets the framework. Failure to disclose is a CCPA Misleading Ads risk."
    citation: "BIS IS 19000:2022 (Online Consumer Reviews); CCPA Misleading Ads Guidelines 2022"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-033
    surface_id: web-ux-social-feed-sidebar
    lens: legal
    severity: P1
    issue: "Social feed widget shows 'Trending Chefs / Popular This Week' — these are objective recommendations that influence consumer choice"
    evidence_excerpt: "Trending Chefs / Popular This Week"
    recommendation: "Disclose ranking algorithm in plain English. CCPA Dark Patterns Guidelines 2023 flag 'False Urgency' and 'Disguised Ad' — Trending/Popular without methodology is a disguised-ranking risk. If sponsored placement is possible, label as Ad."
    citation: "CCPA Dark Patterns Guidelines 2023 r.2 (False Urgency), r.10 (Disguised Ads); CP (E-Commerce) Rules 2020 r.5(4)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-034
    surface_id: web-ux-browse-filters-dietary
    lens: legal
    severity: P0
    issue: "Dietary filter offers 'Vegan / Gluten-Free / Dairy-Free / Keto / Halal' — these are health/religious claims that carry the highest substantiation burden"
    evidence_excerpt: "Vegetarian / Vegan / Gluten-Free / Dairy-Free / Keto / Halal"
    recommendation: "FSSAI organic/vegan/health-claim regs require certification (e.g., FSSAI Vegan Logo since 2022). 'Halal' requires halal-certifier attestation. 'Gluten-Free' is regulated nutritional claim (≤20 ppm gluten per FSSAI). Either certify each chef's claim or relabel as 'Chef-declared — not certified' with explicit disclaimer."
    citation: "FSSAI Vegan Foods Regulations 2022; FSSAI Food Safety and Standards (Health Supplements...) Regulations 2022; FSSAI Order on Halal 2022; FSS (Labelling and Display) Regulations 2020 r.7"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-035
    surface_id: mc-home-cuisine-filters
    lens: legal
    severity: P2
    issue: "Mobile sort option 'Recommended' is opaque — could be paid placement, algorithmic, or editorial. No disclosure"
    evidence_excerpt: "Recommended, Top Rated, Newest, Price"
    recommendation: "CP (E-Commerce) Rules 2020 r.5(4) requires disclosure of main parameters determining ranking. Add tooltip explaining the recommendation logic."
    citation: "CP (E-Commerce) Rules 2020 r.5(4); CCPA Dark Patterns Guidelines 2023"
    depends_on: "needs lawyer review"

  # ============================================================================
  # REVIEWS — defamation, moderation, CPA 2019 + BIS IS 19000
  # ============================================================================

  - finding_id: LEG-COREUX-036
    surface_id: web-ux-chefdetail-reviews
    lens: legal
    severity: P1
    issue: "Reviews block shows 'Customer Reviews / Chef's Response' but no moderation policy, no flagging mechanism in inventory, no display of whether reviewer is a verified buyer"
    evidence_excerpt: "No reviews yet / Customer Reviews / Chef's Response:"
    recommendation: "BIS IS 19000:2022 requires verified-purchase indicator, moderation transparency, and report-abuse pathway. Without it, the platform inherits IT Act safe-harbour risk (intermediary obligations under §79 + IT Rules 2021)."
    citation: "BIS IS 19000:2022; IT Act 2000 §79; IT (Intermediary Guidelines) Rules 2021 r.3"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-037
    surface_id: web-ux-chef-social
    lens: legal
    severity: P1
    issue: "Chef social composer 'Share something about this dish...' allows user-generated content with no terms reference, no IP assignment, no community-guidelines link"
    evidence_excerpt: "Social Feed / No posts yet / Share something about this dish... / #homemade"
    recommendation: "UGC publishing requires (a) IP licence terms (chef grants platform a non-exclusive licence to display), (b) community guidelines reference, (c) intermediary takedown mechanism per IT Rules 2021 r.3(1)(d). Add Terms link at composer."
    citation: "IT Rules 2021 r.3; Copyright Act 1957 §52; Indian Contract Act 1872 §10"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-038
    surface_id: vp-ux-reviews-title
    lens: legal
    severity: P2
    issue: "Vendor reviews page 'See what your customers are saying' provides no path to flag defamatory reviews or contest verified-purchase status"
    evidence_excerpt: "Reviews / See what your customers are saying"
    recommendation: "Provide chef-side report-abuse and dispute mechanism per BIS IS 19000:2022 and IT Rules 2021 r.3(2)."
    citation: "BIS IS 19000:2022; IT Rules 2021 r.3(2)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-039
    surface_id: mv-reviewdetail-input-ph
    lens: legal
    severity: P2
    issue: "Reply input 'Write a thoughtful reply to this review' has no defamation / community-guidelines reminder — chef's reply could carry CPA / IPC §499 exposure"
    evidence_excerpt: "Write a thoughtful reply to this review..."
    recommendation: "Show a one-line reminder: 'Replies are public. Be respectful and factual.' Reduces platform exposure under IT Act §79 safe harbour."
    citation: "IT Act 2000 §79; IPC 1860 §499 (defamation); IT Rules 2021 r.3"
    depends_on: "needs lawyer review"

  # ============================================================================
  # CATERING — bespoke contract formation, larger sums = higher exposure
  # ============================================================================

  - finding_id: LEG-COREUX-040
    surface_id: web-ux-catering-quotes-detail
    lens: legal
    severity: P0
    issue: "Catering Request Details surface mediates a high-value bespoke contract between customer and chef with no T&C reference, no event-cancellation policy, no force-majeure clause linkage"
    evidence_excerpt: "Select a request / Request Details / No quotes yet / Proposed Menu / Message chef"
    recommendation: "Catering events involve significant prepayment exposure (vs ad-hoc food orders). Surface a specific Catering Terms & Cancellation Policy at quote acceptance. Required for Indian Contract Act enforceability of a bespoke contract."
    citation: "Indian Contract Act 1872 §10; CPA 2019 §2(11)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-041
    surface_id: web-ux-catering-request-event-fields
    lens: legal
    severity: P1
    issue: "Catering Request collects event date, guest count, budget — but no FSSAI catering-specific compliance hint (large gatherings may require special hygiene measures)"
    evidence_excerpt: "Event Date / Event Time / Number of Guests / Minimum budget / Maximum budget / Min / Max"
    recommendation: "Above threshold guest count (e.g., 50+), FSSAI may require additional catering compliance. Surface guidance and capture chef's catering licence."
    citation: "FSS (Licensing and Registration) Regulations 2011 r.2.1.5 (caterers)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-042
    surface_id: web-ux-chef-catering
    lens: legal
    severity: P0
    issue: "Chef-side catering quote submission ('Submit Quote / Item name / Qty / Price/unit') has no GST tax-rate field, no taxable/exempt declaration — large-ticket catering will fail GST compliance"
    evidence_excerpt: "Catering / No open requests / No pending quotes / No booked events / Submit Quote / Item name / Qty / Price/unit / Any special notes about your quote..."
    recommendation: "Catering quote must indicate whether prices are inclusive/exclusive of GST, applicable SAC code, chef's GSTIN. CGST applies to outdoor catering at 18% (without ITC unless conditions met)."
    citation: "CGST Act 2017 §31; CBIC Notification 11/2017-CT(R) (catering rate)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-043
    surface_id: mc-catering-event-types
    lens: legal
    severity: P3
    issue: "Mobile catering event-type chip 'Wedding' is a high-stakes contract category — should not be a casual chip without surfacing the catering T&C immediately"
    evidence_excerpt: "Wedding, Birthday, Corporate, Anniversary, Festival, House Party, Other"
    recommendation: "Cosmetic legal-page nudge — but Wedding/Corporate selections should preview the relevant T&C variant."
    citation: "Indian Contract Act 1872 §10 (best-practice)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-044
    surface_id: mc-catering-form-labels
    lens: legal
    severity: P1
    issue: "Mobile catering form has 'Additional Details' free-text field and a 'Submit Request' CTA with no T&C / Privacy / Refund-policy acknowledgement"
    evidence_excerpt: "Event Type * / Event Date * (YYYY-MM-DD) / Guest Count * / Budget (₹) / City * / State * / Additional Details"
    recommendation: "Catering submission is intent-to-contract — add inline acknowledgement linking to Catering Terms before submit."
    citation: "Indian Contract Act 1872 §10"
    depends_on: "needs lawyer review"

  # ============================================================================
  # VENDOR / CHEF SETTINGS — commission, payouts, RBI PA MD
  # ============================================================================

  - finding_id: LEG-COREUX-045
    surface_id: vp-ux-settings-order-acceptance
    lens: legal
    severity: P1
    issue: "Order Acceptance settings show 'Auto-accept threshold ($)' — currency mismatch is flagged in inventory ('shows $ but data is INR'). Currency-symbol error in a financial setting is a misrepresentation"
    evidence_excerpt: "Auto-accept threshold ($) / Orders under this amount will be auto-accepted"
    recommendation: "Fix to ₹ immediately. Money values displayed in wrong currency violate CPA 2019 §2(28) misleading-representation if any chef misconfigures based on $-INR confusion."
    citation: "CPA 2019 §2(28); FEMA 1999 (cross-border misrepresentation)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-046
    surface_id: vp-ux-settings-payout-section
    lens: legal
    severity: P0
    issue: "Payout Details surface shows 'Razorpay Connected / Linked Account' but no settlement-cadence disclosure, no escrow statement, no T+N timeline reference"
    evidence_excerpt: "Payout Details / Razorpay Connected / Razorpay Pending / Linked Account: ... / No payout details configured yet. / Set Up Payouts / Edit / Method / Account Holder / Account Number / IFSC Code / UPI ID"
    recommendation: "RBI PA MD requires the platform to disclose settlement cadence (T+1, T+2 etc.), escrow mechanism, and reconciliation rights to merchants. Add a disclosure card to chef: 'Payouts settle within X working days. Funds are held in an escrow account at <PA name>.'"
    citation: "RBI PA MD 17 Mar 2020 §8 (Settlement and Escrow); §6 (Merchant On-boarding)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-047
    surface_id: vp-ux-earnings-cards
    lens: legal
    severity: P1
    issue: "Earnings cards show 'Available Balance / Pending Payout / This Month / Lifetime Earnings' but no commission/service-fee deduction line breakdown — gig-worker earnings transparency"
    evidence_excerpt: "Available Balance / Pending Payout / This Month / Lifetime Earnings / +X.X% vs last month"
    recommendation: "Code on Social Security 2020 (when notified) and CPA 2019 expect transparency on aggregator deductions. Surface gross-to-net breakdown: Gross earnings minus platform commission minus GST on commission minus TDS = net payout."
    citation: "Code on Social Security 2020 §114 (gig and platform workers); Income Tax Act §194-O (TDS by e-comm operator)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-048
    surface_id: vp-ux-payouts-title
    lens: legal
    severity: P1
    issue: "Payout History 'View all your past and pending payouts' surface — inventory does not indicate whether each payout row links to a TDS certificate / Form 26AS reconciliation"
    evidence_excerpt: "Payout History / View all your past and pending payouts / Back to Earnings"
    recommendation: "Each payout must reference TDS deducted (§194-O of IT Act for e-commerce operators), make Form 16A / 26AS available."
    citation: "Income Tax Act §194-O; CGST Rules r.46"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-049
    surface_id: mv-earnings-payout-fields
    lens: legal
    severity: P1
    issue: "Mobile chef payout capture 'Bank / Account Number / IFSC / UPI ID' has no inline DPDP notice — these are financial PII"
    evidence_excerpt: "Bank / Account Number / IFSC / UPI ID"
    recommendation: "DPDP §5 notice at point of capture for financial PII. Also require dual-channel verification (penny drop / OTP) per RBI KYC framework for payout account changes."
    citation: "DPDP Act 2023 §5; RBI KYC MD 2016"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-050
    surface_id: dp-ux-stripe-intro
    lens: legal
    severity: P0
    issue: "Driver Stripe onboarding intro 'Accept delivery payouts in your local currency. Stripe handles KYC and bank verification on its hosted pages.' — cross-border data export with no transfer notice"
    evidence_excerpt: "Accept delivery payouts in your local currency. Stripe handles KYC and bank verification on its hosted pages."
    recommendation: "DPDP §16 governs cross-border transfer of personal data. Add explicit notice: 'Stripe processes your data outside India under their privacy notice. By proceeding you consent to cross-border transfer.' Also note FEMA implications for non-INR settlement."
    citation: "DPDP Act 2023 §16 (Transfer outside India); FEMA 1999 §3"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-051
    surface_id: dp-ux-stripe-sub
    lens: legal
    severity: P2
    issue: "Stripe subtitle 'For drivers outside India, or as an alternative to Razorpay' suggests driver may be outside India — Home Chef's India-only platform claim should be reconciled"
    evidence_excerpt: "For drivers outside India, or as an alternative to Razorpay."
    recommendation: "Lawyer review whether non-India drivers are within scope. If yes, FEMA + DPDP cross-border rules apply; if no, remove the language."
    citation: "FEMA 1999; DPDP Act 2023 §16"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-052
    surface_id: dp-ux-stripe-active-gateway
    lens: legal
    severity: P1
    issue: "Driver can switch between Stripe and Razorpay as active payout gateway — switching gateways has audit/tax implications (TDS attribution, GSTIN of aggregator)"
    evidence_excerpt: "Active payout gateway: {Stripe/Razorpay}"
    recommendation: "Switch action should require confirmation with disclosure: 'Switching gateway affects how your tax forms are issued. Existing pending payouts will settle on the old gateway.'"
    citation: "Income Tax Act §194-O; CGST Act §51"
    depends_on: "needs lawyer review"

  # ============================================================================
  # ADMIN — audit log, exports, attribution, GST settings
  # ============================================================================

  - finding_id: LEG-COREUX-053
    surface_id: ap-auditlogs-title
    lens: legal
    severity: P1
    issue: "Audit logs page exists (good) — inventory tags it 'LEGAL LENS' — but inventory excerpt does not confirm immutability, retention period, or tamper-evidence"
    evidence_excerpt: "Audit logs"
    recommendation: "Audit logs supporting RBI PA, GST, FSSAI investigations require immutability and minimum retention (commonly 8 years for tax, 5 years for RBI PA). Surface retention policy on the page; document tamper-evidence in admin docs."
    citation: "RBI PA MD 2020 §6.5; CGST Rules r.56; IT Act §65B (electronic evidence)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-054
    surface_id: ap-auditlogs-table
    lens: legal
    severity: P1
    issue: "Audit table columns 'When; Actor; Action; Entity; IP' — missing user-agent, geolocation hint, and 'reason' field for sensitive ops"
    evidence_excerpt: "When; Actor; Action; Entity; IP"
    recommendation: "For high-stakes admin ops (suspend chef, refund, change commission) capture reason text and user-agent. Required for IT Act §65B electronic-evidence admissibility and CPA dispute defence."
    citation: "IT Act 2000 §65B; Bharatiya Sakshya Adhiniyam 2023 §63"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-055
    surface_id: ap-exports-cards
    lens: legal
    severity: P0
    issue: "Admin Data Exports allow 'User data / All user accounts (id, email, role, created_at)' export to CSV — this is bulk personal-data export with no DPDP audit trail referenced"
    evidence_excerpt: "User data / All user accounts (id, email, role, created_at); Orders / Order rows filtered by the date range above; Revenue / Per-day revenue rollup — paid orders only"
    recommendation: "DPDP §8 (Data Fiduciary obligations) requires logging of personal-data access and purpose limitation. Bulk export must (a) capture justification, (b) be limited to specific admin roles, (c) generate an audit entry, (d) watermark / encrypt the export, (e) auto-expire download URLs."
    citation: "DPDP Act 2023 §8, §10 (Significant Data Fiduciary); IT Rules 2021"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-056
    surface_id: ap-exports-subtitle
    lens: legal
    severity: P2
    issue: "Export subtitle 'Files stream directly from the API' suggests no intermediate storage but does not address retention, encryption-in-transit, or expiry of generated CSVs"
    evidence_excerpt: "Download platform data as CSV. Files stream directly from the API."
    recommendation: "Clarify whether CSVs are signed-URL time-bound, password-protected, audited. DPDP §8(5) safeguards required."
    citation: "DPDP Act 2023 §8(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-057
    surface_id: ap-platsettings-commission-fields
    lens: legal
    severity: P0
    issue: "Platform Settings expose 'Service fee (%) / Tax (%) / Chef payout (% of subtotal) / Driver payout (% of delivery fee) / Base delivery fee / Per-km rate' — admin can change these silently with no chef/customer notification mechanic referenced"
    evidence_excerpt: "Service fee (%); Tax (%); Chef payout (% of subtotal); Driver payout (% of delivery fee); Base delivery fee; Per-km rate"
    recommendation: "Material changes to commission and fee structure affect chef and driver contracts. CP (E-Commerce) Rules 2020 r.5(3) and the chef/driver T&C should require advance notice (commonly 30 days). Surface a 'changes take effect on <date>; notify all chefs/drivers' workflow. Also 'Tax (%)' is dangerously vague — platform cannot set GST rate; it is statutory."
    citation: "CP (E-Commerce) Rules 2020 r.5(3); CGST Act 2017 (statutory rate, not configurable)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-058
    surface_id: ap-platsettings-cards
    lens: legal
    severity: P1
    issue: "Platform Settings 'Operating hours / When the platform accepts new orders' — if ordering is restricted to hours but customer doesn't know, that's a service-availability gap"
    evidence_excerpt: "Operating hours / When the platform accepts new orders"
    recommendation: "Closed-hours message (mentioned in ap-platsettings-hours-fields) must be customer-visible at homepage and cart, not just an admin setting. CPA 2019 deficiency-of-service trigger if customer believes service is available."
    citation: "CPA 2019 §2(11)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-059
    surface_id: ap-platsettings-zones-fields
    lens: legal
    severity: P2
    issue: "Service zone CRUD using lat/long bounding boxes — if customer addresses fall outside, order is silently rejected. Need clear rejection message"
    evidence_excerpt: "Name; City; State; Min latitude; Max latitude; Min longitude; Max longitude; New zone; Create zone; Cancel"
    recommendation: "Surface clear out-of-zone message at customer-facing layer with reason. CP (E-Commerce) Rules 2020 r.5(4) requires honest disclosure of service-availability constraints."
    citation: "CP (E-Commerce) Rules 2020 r.5(4)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-060
    surface_id: ap-settings-payment-fields
    lens: legal
    severity: P1
    issue: "Payment-gateway settings show 'Razorpay Key ID / Razorpay Key Secret / Webhook Secret / Stripe Secret Key / Stripe Publishable Key / Webhook Signing Secret' inside admin UI — this is a security/compliance posture concern (HSM, key rotation)"
    evidence_excerpt: "Razorpay Key ID; Razorpay Key Secret; Webhook Secret; Stripe Secret Key; Stripe Publishable Key; Webhook Signing Secret; Mode; Key Prefix; Secret Key Prefix; Publishable Key; Webhook URL"
    recommendation: "PCI-DSS not directly applicable (Razorpay handles card data) but RBI PA MD requires PA-PoS secret protection. Secrets should be one-way: enter, never display. Audit-log every access. Restrict to super-admin role."
    citation: "RBI PA MD 2020 §5 (Security); PCI-DSS (via Razorpay)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-061
    surface_id: ap-secsettings-apikey-fields
    lens: legal
    severity: P2
    issue: "API key form 'Scopes: read; write; admin; Expires in (days; 0 = never)' — 'never expiring' API keys are a compliance red flag"
    evidence_excerpt: "Name; Scopes; read; write; admin; Expires in (days; 0 = never); Create key; New API key; e.g. Partner integration; I've saved it"
    recommendation: "Disallow '0 = never' for non-machine partners. RBI PA framework and DPDP best-practice expect periodic key rotation (commonly 90-180 days)."
    citation: "RBI PA MD 2020 §5; DPDP Act §8(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-062
    surface_id: ap-secsettings-session-fields
    lens: legal
    severity: P2
    issue: "Session management fields configurable but no policy lower-bound enforced in inventory — admin could set TTL to infinite, defeating DPDP and CPA security expectations"
    evidence_excerpt: "Access token TTL (hours); Refresh token TTL (days); Save policy; Your active sessions; No active sessions."
    recommendation: "Enforce maximum-TTL caps (e.g., access ≤24h, refresh ≤30d) in admin UI."
    citation: "DPDP Act 2023 §8(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-063
    surface_id: ap-staffdetail-section-roleperm
    lens: legal
    severity: P2
    issue: "Staff Role & Permissions section exists but inventory does not surface least-privilege enforcement or 'who can see customer PII' role gates"
    evidence_excerpt: "Role & Permissions"
    recommendation: "DPDP §8(5) requires technical and organisational safeguards including least-privilege access to personal data. Document role -> data-class matrix."
    citation: "DPDP Act 2023 §8(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-064
    surface_id: ap-userdetail-info-rows
    lens: legal
    severity: P1
    issue: "Admin User Detail surfaces 'Email; Phone; Auth Provider; Role; Joined; Last Login' — admin viewing customer PII must generate access log per DPDP"
    evidence_excerpt: "Email; Phone; Auth Provider; Role; Joined; Last Login"
    recommendation: "Every admin access to customer PII must be audited with reason. Surface a 'reason' modal on first PII view."
    citation: "DPDP Act 2023 §8(2), §8(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-065
    surface_id: ap-userdetail-order-stats
    lens: legal
    severity: P2
    issue: "User Detail shows 'Total Orders / Total Spent / Last Order' — financial PII visible to admin; no purpose-limitation notice"
    evidence_excerpt: "Order Activity; Total Orders; Total Spent; Last Order; No orders yet"
    recommendation: "Per DPDP, document purpose for which admin views aggregate spend; restrict to roles with a need (finance, support escalation)."
    citation: "DPDP Act 2023 §7(2) (purpose limitation)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-066
    surface_id: ap-chefs-meta-bottom
    lens: legal
    severity: P3
    issue: "Chef cards meta shows 'Min order: ₹{n}; Service radius: {n} km; Joined: {date}' but no FSSAI registration number column"
    evidence_excerpt: "Min order: ₹{n}; Service radius: {n} km; Joined: {date}"
    recommendation: "Add FSSAI registration column to admin Chef listing for compliance review."
    citation: "FSS Act 2006 §31"
    depends_on: "needs lawyer review"

  # ============================================================================
  # DRIVER / DELIVERY — gig classification, safety, MV Act, insurance
  # ============================================================================

  - finding_id: LEG-COREUX-067
    surface_id: dp-ux-stat-verified-partners
    lens: legal
    severity: P1
    issue: "'Verified partners' label on admin dashboard — what verification standard is met? MV Act licence verified? Background check? Vehicle insurance current?"
    evidence_excerpt: "Verified partners / of {n} total"
    recommendation: "Define and disclose what 'Verified' means. Drivers handle food (FSSAI hygiene) AND operate vehicles (MV Act). Verification scope must be auditable."
    citation: "Motor Vehicles Act 1988; FSS Act 2006; CCPA Misleading Ads Guidelines 2022"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-068
    surface_id: dp-ux-partner-detail-verify-block
    lens: legal
    severity: P1
    issue: "Partner Verification block 'Review their documents and approve or reject their application' is the gating moment for driver onboarding. Inventory does not surface mandatory document checklist"
    evidence_excerpt: "Pending Verification / This partner is awaiting verification. Review their documents and approve or reject their application."
    recommendation: "Surface checklist on the admin screen: Driving Licence (current), Vehicle RC, Vehicle Insurance (third-party at minimum), PAN, Aadhaar/address proof, Bank account, Police clearance (best-practice). MV Act and Code on Social Security."
    citation: "MV Act 1988 §3, §146; Code on Social Security 2020 §114"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-069
    surface_id: dp-ux-active-mark-as
    lens: legal
    severity: P0
    issue: "Driver 'Mark as {Picked Up/In Transit/Delivered}' is the customer-impacting status update — false state changes have CPA + RBI PA implications"
    evidence_excerpt: "Mark as {Picked Up/In Transit/Delivered}"
    recommendation: "Each status change should require capture of GPS location + timestamp for evidentiary purposes (CPA dispute, RBI PA chargeback). 'Mark as Delivered' specifically should not allow self-mark without geofence proximity check or customer confirmation OTP."
    citation: "CPA 2019 §2(11); RBI PA MD 2020 §6; IT Act §65B"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-070
    surface_id: md-core-036
    lens: legal
    severity: P0
    issue: "Mobile driver 'Mark as Delivered' (P0 SAFETY tagged in inventory) — same concern: irreversible terminal action triggering customer refund-window-closing"
    evidence_excerpt: "Mark as Delivered"
    recommendation: "Require customer-OTP confirmation or geofence + photo proof. Audit log with GPS coords. Critical for RBI PA disputes."
    citation: "RBI PA MD 2020 §6; CPA 2019 §2(11)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-071
    surface_id: md-core-018
    lens: legal
    severity: P1
    issue: "Driver 'Go Online to Accept Deliveries' has no insurance / safety attestation gate — could be interpreted as platform endorsement that driver is fit to drive"
    evidence_excerpt: "Go Online to Accept Deliveries"
    recommendation: "Gate Go-Online behind self-declared 'I am fit to drive, my licence and insurance are current'. Logs the attestation. Reduces platform liability under MV Act vicarious-liability theory."
    citation: "MV Act 1988; CPA 2019"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-072
    surface_id: md-core-062
    lens: legal
    severity: P1
    issue: "Driver settings 'Default Online Status — Automatically go online when app opens' (P0 SAFETY tagged in inventory). Auto-online creates ambiguity about whether driver consciously consented to a delivery shift"
    evidence_excerpt: "Default Online Status / Automatically go online when app opens"
    recommendation: "Disable by default. Code on Social Security 2020 gig-worker provisions, when notified, will require active opt-in for work sessions. Also a CPA safety concern."
    citation: "Code on Social Security 2020 §114; MV Act 1988"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-073
    surface_id: dp-ux-active-payout-label
    lens: legal
    severity: P1
    issue: "Active delivery shows 'Payout' value pre-completion — if payout is shown but later reduced due to refund/dispute, gig-worker has earnings-transparency complaint"
    evidence_excerpt: "Payout"
    recommendation: "Label as 'Estimated payout' until settled; show final payout post-settlement. Disclose deduction rules (cancellations, customer complaints) upfront."
    citation: "Code on Social Security 2020 §114; CPA 2019"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-074
    surface_id: md-core-047
    lens: legal
    severity: P2
    issue: "'You earned ₹{payout} for this delivery' is a representation of completed earnings — if subsequent customer complaint reduces it, must be reconcilable"
    evidence_excerpt: "You earned ₹{payout} for this delivery"
    recommendation: "Make explicit: 'Payout credited pending dispute window of X days.' Or distinguish 'You earned' (final) from 'Estimated' (pre-settlement)."
    citation: "Code on Social Security 2020 §114; CPA 2019 §2(28)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-075
    surface_id: dp-ux-active-pickup-from
    lens: legal
    severity: P2
    issue: "'Pickup from' and 'Deliver to' display addresses to driver — chef/customer PII disclosed to a third party (driver). Need DPDP purpose-limitation gate"
    evidence_excerpt: "Pickup from / Deliver to"
    recommendation: "Address is necessary for delivery — covered by contractual-necessity legal basis under DPDP. But: do not retain in driver app after delivery; auto-purge from local storage on completion. Document this lifecycle."
    citation: "DPDP Act 2023 §7(1)(b) (legitimate use - performance of contract); §8(7) (erasure)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-076
    surface_id: dp-ux-active-order-note
    lens: legal
    severity: P3
    issue: "Order note 'Note: {specialInstructions}' — chef-supplied free text shown to driver. If chef includes customer-PII (gate code, name) it's processed by driver"
    evidence_excerpt: "Note: {specialInstructions}"
    recommendation: "Lawyer review whether special-instructions free text is appropriate channel for PII. Consider field-level redaction or warning to chefs."
    citation: "DPDP Act 2023 §7(2) (purpose limitation)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-077
    surface_id: md-core-043
    lens: legal
    severity: P1
    issue: "'Navigate' (P0 SAFETY tag in inventory) launches external Apple/Google Maps — third-party data export of pickup/dropoff coordinates with no notice"
    evidence_excerpt: "Navigate"
    recommendation: "Inline notice: 'Navigate opens your maps app. Your route will be processed by Google/Apple per their terms.' DPDP §16 cross-border consideration if Google routes data outside India."
    citation: "DPDP Act 2023 §5, §16"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-078
    surface_id: ap-providercreate-fields
    lens: legal
    severity: P1
    issue: "Delivery Provider create form captures 'API Key; API Secret; Webhook Secret' and 'Pricing Model; Currency; Base Cost; Per KM Cost' — third-party delivery provider integration touches consumer pricing"
    evidence_excerpt: "Name; Code; Description; Logo URL; API Base URL; API Key; API Secret; Webhook Secret; ...Currency; Base Cost; Per KM Cost; Max Concurrent Deliveries; Daily Limit; Avg Pickup Time (min); Contact Name; Contact Email; Contact Phone; Notes"
    recommendation: "Adding a 3P delivery provider changes the data-processor chain — requires updated Privacy Notice (DPDP §5(i) categories of data fiduciaries / processors). Also commercial T&C with the provider should be referenced."
    citation: "DPDP Act 2023 §5, §11 (Data Processor)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-079
    surface_id: ap-delivery-statuses
    lens: legal
    severity: P2
    issue: "Delivery status taxonomy includes 'Failed' and 'Returned' — these are refund-triggering events but inventory does not surface auto-refund or compensation mechanic"
    evidence_excerpt: "Pending; Assigned; At Pickup; Picked Up; In Transit; At Dropoff; Delivered; Failed; Returned; Cancelled"
    recommendation: "Define when 'Failed'/'Returned' auto-initiate refund per RBI PA timelines. Customer-visible status copy should explain consequence."
    citation: "RBI PA MD 2020 §8; CPA (E-Commerce) Rules 2020"
    depends_on: "needs lawyer review"

  # ============================================================================
  # CUSTOMER PROFILE / PROFILE PAYMENT / 2FA / DPDP RIGHTS
  # ============================================================================

  - finding_id: LEG-COREUX-080
    surface_id: web-ux-profile-payments
    lens: legal
    severity: P1
    issue: "Payment Methods tab heading exists but inventory does not surface how stored cards/UPI handles are tokenised, where they live, or who is the data controller (Razorpay tokenises per RBI rules)"
    evidence_excerpt: "Payment Methods"
    recommendation: "Clarify that card data is tokenised by Razorpay per RBI Card-on-File tokenisation rules; Home Chef does not store PAN. Surface link to RBI tokenisation FAQ."
    citation: "RBI Tokenisation Framework (CoFT, 2022); RBI PA MD 2020"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-081
    surface_id: web-ux-profile-2fa
    lens: legal
    severity: P2
    issue: "Two-Factor Authentication setup page exists (good for DPDP §8(5) safeguards) but is the setup mandatory or optional? Customer-facing security posture should be documented"
    evidence_excerpt: "Two-Factor Authentication / Set Up Two-Factor Authentication / Save Your Backup Codes"
    recommendation: "DPDP §8(5) requires reasonable security safeguards. 2FA being optional is acceptable; mandatory 2FA for high-stakes accounts (admin, chef payouts) recommended. Document in security policy."
    citation: "DPDP Act 2023 §8(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-082
    surface_id: web-ux-profile-preferences
    lens: legal
    severity: P2
    issue: "Food preferences ('Dietary Preferences / Food Allergies') — Food Allergies is sensitive health data adjacent under DPDP — should have explicit purpose-of-use notice"
    evidence_excerpt: "Food Preferences / Dietary Preferences / Food Allergies / Favourite Cuisines / Spice Tolerance / Household Size"
    recommendation: "Allergy data is highly sensitive. Add explicit notice: 'We use this to filter and warn you about allergens. We do not share it with marketing partners.' Purpose-limitation under DPDP §7(2)."
    citation: "DPDP Act 2023 §7(2)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-083
    surface_id: web-ux-profile-notifications
    lens: legal
    severity: P1
    issue: "Notification Preferences tab exists but inventory does not surface granular opt-ins for marketing vs transactional, or DPDP §6 consent-withdrawal mechanism"
    evidence_excerpt: "Notification Preferences / What to notify about / How to notify"
    recommendation: "Separate transactional (order updates — legitimate interest) from marketing (requires opt-in consent under DPDP §6 and TRAI commercial communications). Show date of last consent change and 'Withdraw consent' button."
    citation: "DPDP Act 2023 §6, §6(5); TRAI Commercial Communications Customer Preference Regulations 2018"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-084
    surface_id: web-ux-profile-tabs
    lens: legal
    severity: P0
    issue: "Profile sidebar tabs 'Profile / Preferences / Addresses / Payment Methods / Notifications / Security' — missing a 'Data & Privacy' tab for DPDP rights (access, correction, erasure, portability, grievance)"
    evidence_excerpt: "Profile / Preferences / Addresses / Payment Methods / Notifications / Security"
    recommendation: "Add 'Data & Privacy' tab exposing DPDP §11 rights: right to access data summary, correct/update, erase account+data, port (where applicable), and contact Grievance Officer. Without this surface, DPDP §11 rights are not operationally available."
    citation: "DPDP Act 2023 §11, §13 (Grievance Officer); §12 (Right to nominate)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-085
    surface_id: web-ux-profile-addresses
    lens: legal
    severity: P1
    issue: "Saved Addresses tab — addresses are PII retained indefinitely unless user-purged. No retention statement"
    evidence_excerpt: "Saved Addresses / Enter PIN code / House / flat / building number, street / Landmark, area (optional)"
    recommendation: "DPDP §8(7) erasure obligation. Surface 'Delete address' per row (likely present) plus retention policy: 'Addresses are kept while your account is active.'"
    citation: "DPDP Act 2023 §8(7), §11(c)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-086
    surface_id: mc-profile-logout-button
    lens: legal
    severity: P3
    issue: "'Log Out' button label drifts from style-guide 'Sign out' — but more importantly mobile customer profile has no 'Delete account' visible in inventory excerpt"
    evidence_excerpt: "Log Out"
    recommendation: "DPDP §11(c) right to erasure must be operationalised in customer mobile too. Add 'Delete account' option (likely present elsewhere — verify)."
    citation: "DPDP Act 2023 §11(c)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-087
    surface_id: mv-settings-delete-account
    lens: legal
    severity: P0
    issue: "Mobile chef 'Delete Account' destructive action exists (good) but inventory does not surface (a) what happens to in-flight orders, (b) what happens to required-retention records (GST 8yr, RBI 5yr), (c) clarification on data anonymisation vs hard delete"
    evidence_excerpt: "Delete Account"
    recommendation: "Delete-account flow must distinguish: erase personal data per DPDP §8(7) while retaining statutorily required records (GST, RBI PA, FSSAI) with personal data minimised. Show user the policy at the confirmation step."
    citation: "DPDP Act 2023 §8(7); CGST Rules r.56 (record retention); RBI PA MD 2020"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-088
    surface_id: md-core-067
    lens: legal
    severity: P0
    issue: "Mobile driver 'Delete Account' — same DPDP/MV Act/Code on Social Security record-retention concern"
    evidence_excerpt: "Delete Account"
    recommendation: "Driver delete must retain MV Act records, insurance claim history, payout/TDS records while erasing other personal data. Disclose at confirm step."
    citation: "DPDP Act 2023 §8(7); MV Act 1988; Income Tax Act §194-O"
    depends_on: "needs lawyer review"

  # ============================================================================
  # ONBOARDING (mobile chef / mobile driver) - PII at signup
  # ============================================================================

  - finding_id: LEG-COREUX-089
    surface_id: mv-onb-review-terms-yes
    lens: legal
    severity: P0
    issue: "Mobile chef onboarding review shows 'Terms accepted indicator' as a yes/no but does not capture the version or content of the terms accepted"
    evidence_excerpt: "Yes"
    recommendation: "Acceptance record must pin version-hash + timestamp + IP. Required for IT Act §65B evidentiary admissibility. Without version-pinning, the platform cannot prove what the chef agreed to."
    citation: "IT Act 2000 §65B; Indian Contract Act 1872 §10"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-090
    surface_id: mv-onb-docs-pdf-uploaded
    lens: legal
    severity: P1
    issue: "Document upload flow ('PDF uploaded / Camera / Gallery / PDF') accepts KYC and FSSAI documents but inventory shows no encryption-at-rest statement or storage-provider disclosure"
    evidence_excerpt: "PDF uploaded / Camera / Gallery / PDF"
    recommendation: "Surface DPDP §5 notice: 'These documents are encrypted and stored on our secure servers. We share them with payment partners for KYC. Retention: X years per regulatory requirement.'"
    citation: "DPDP Act 2023 §5, §8(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-091
    surface_id: mv-onb-kitchen-cuisine-options
    lens: legal
    severity: P3
    issue: "Onboarding cuisine list includes 'Other' as a chip — open-ended FSSAI categorisation may create allergen-coverage gaps"
    evidence_excerpt: "North Indian / South Indian / Chinese / Continental / Bakery / Snacks / Beverages / Other"
    recommendation: "If 'Other' is selected, prompt for free-text description; this informs FSSAI allergen-risk class."
    citation: "FSS Act 2006 §31; FSSAI categories"
    depends_on: "needs lawyer review"

  # ============================================================================
  # SETTINGS / NOTIFICATIONS (multi-app)
  # ============================================================================

  - finding_id: LEG-COREUX-092
    surface_id: ap-notifsettings-categories
    lens: legal
    severity: P1
    issue: "Notification categories include 'Promotions / Discounts, newsletters, and marketing campaigns. Opt-in.' — opt-in label is correct, but inventory does not confirm pre-checked state is off, or that withdrawal is one-click"
    evidence_excerpt: "Promotions / Discounts, newsletters, and marketing campaigns. Opt-in."
    recommendation: "DPDP §6(1) prohibits pre-checked consent. Confirm Promotions is OFF by default for new accounts. TRAI DLT also applies for SMS marketing."
    citation: "DPDP Act 2023 §6(1); TRAI Commercial Communications Customer Preference Regulations 2018"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-093
    surface_id: vp-ux-settings-notifications
    lens: legal
    severity: P2
    issue: "Vendor SMS notifications 'Get an SMS for each new order' — SMS to vendor is transactional (legitimate interest) but TRAI DLT registration still applies"
    evidence_excerpt: "SMS / New order SMS / Get an SMS for each new order"
    recommendation: "Confirm DLT registration for transactional SMS template; document. Surface consent in vendor T&C."
    citation: "TRAI TCCCPR 2018"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-094
    surface_id: vp-ux-settings-password
    lens: legal
    severity: P2
    issue: "Password change section copy 'Your account is linked to {provider} login. Password management is handled by your social login provider.' clearly tells social-login users they cannot change password locally — but does not link to provider's password page or detail the data shared with the social provider"
    evidence_excerpt: "Change Password / Current Password / New Password / Confirm New Password / Update Password / Password updated successfully / Failed to update password. Check your current password. / Your account is linked to {provider} login. Password management is handled by your social login provider."
    recommendation: "Link to provider's password change page. Surface DPDP-relevant note on what data the social provider shares back (email, name, picture, etc.) and at what frequency."
    citation: "DPDP Act 2023 §5"
    depends_on: "needs lawyer review"

  # ============================================================================
  # STAFF INVITATIONS (multi-app) - role attribution
  # ============================================================================

  - finding_id: LEG-COREUX-095
    surface_id: ap-staff-invite-fields
    lens: legal
    severity: P2
    issue: "Staff invitation form 'Email; Role; Department; Title; Personal Message' issues a role with platform-data access. No mention of access-policy acknowledgement or NDA at invitation"
    evidence_excerpt: "Email; Role; Department; Title; Personal Message; Send Invitation; Done; Copy; Copied"
    recommendation: "Staff accepting invitation should also acknowledge: confidentiality / data-handling policy. DPDP §8(2) requires data fiduciary's safeguards extend to employees."
    citation: "DPDP Act 2023 §8(2)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-096
    surface_id: dp-ux-staff-invite-h
    lens: legal
    severity: P2
    issue: "Delivery-portal Staff Invitation 'Share this link with the new staff member' — link-based invitation that can be forwarded; no token expiry or single-use enforcement referenced"
    evidence_excerpt: "Invitation Created / Share this link with the new staff member:"
    recommendation: "Invitation links must be (a) single-use, (b) short-TTL, (c) recipient-email bound. Otherwise an intercepted link grants role access."
    citation: "DPDP Act 2023 §8(5); RBI PA MD §5 (security)"
    depends_on: "needs lawyer review"

  # ============================================================================
  # LAYOUT NAV - missing legal-page entry points
  # ============================================================================

  - finding_id: LEG-COREUX-097
    surface_id: web-ux-layout-nav
    lens: legal
    severity: P1
    issue: "Customer top nav 'Home / Browse Chefs / Favorites / Catering / Profile / My Orders / Settings / Logout' has NO link to Terms, Privacy, or Refund Policy. Users cannot reach legal pages without scrolling to footer (if it exists)"
    evidence_excerpt: "Home / Browse Chefs / Favorites / Catering / Profile / My Orders / Settings / Logout"
    recommendation: "DPDP §5 expects accessible privacy notice. Add Legal / Terms / Privacy entry to footer; if no footer, link from Settings or user menu."
    citation: "DPDP Act 2023 §5; CP (E-Commerce) Rules 2020 r.5(2)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-098
    surface_id: vp-ux-layout-nav
    lens: legal
    severity: P2
    issue: "Vendor portal sidebar — no link to chef/vendor T&C, Privacy, or platform commission disclosure"
    evidence_excerpt: "Dashboard / Menu / Orders / Earnings / Admin Requests / Reviews / Analytics / Profile / Settings / Vendor Portal / Vendor / Settings / Logout / Theme"
    recommendation: "Add Legal entry in vendor sidebar covering T&C, Privacy, Commission Schedule, Payout Policy."
    citation: "CP (E-Commerce) Rules 2020 r.5; RBI PA MD 2020 §6"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-099
    surface_id: dp-ux-nav-partner-items
    lens: legal
    severity: P2
    issue: "Driver sidebar nav 'Dashboard / Active Delivery / Available / History / Earnings / Profile / Settings' has no link to Driver T&C / Insurance Policy / Gig-Worker Agreement"
    evidence_excerpt: "Dashboard / Active Delivery / Available / History / Earnings / Profile / Settings"
    recommendation: "Add Legal entry exposing Driver Agreement, Insurance Coverage Summary, Deduction Schedule, Termination Clause."
    citation: "Code on Social Security 2020 §114; MV Act 1988"
    depends_on: "needs lawyer review"

  - finding_id: LEG-COREUX-100
    surface_id: ap-layout-nav-settings
    lens: legal
    severity: P3
    issue: "Admin sidebar 'Settings' is the legal/policy-config entry but admin layout has no dedicated 'Compliance' tab for FSSAI, RBI PA, DPDP audit dashboards"
    evidence_excerpt: "Settings"
    recommendation: "Consider promoting compliance to top-level nav for admin: FSSAI status overview (all chefs), RBI PA reconciliation, DPDP grievance queue."
    citation: "DPDP Act 2023 §10 (Significant Data Fiduciary); FSS Act 2006 §31; RBI PA MD 2020 §6.5"
    depends_on: "needs lawyer review"
```

## Business Analyst findings

```yaml
# BA Lens — CORE-UX findings
# Auditor: business-analyst lens agent
# Date: 2026-05-13
# Inventory slice: core-ux (594 rows)
# Surfaces covered: Cart, Checkout, Order Detail, Order Status, Chef Detail,
#   Chef Menu/Orders/Earnings, Vendor Portal Settings, Browse/Search,
#   Driver Active Delivery, Delivery Portal, Admin Analytics,
#   Mobile Customer/Vendor/Delivery apps

---

# ─── P0: DEMONSTRABLY BROKEN CONVERSION PATH ───────────────────────────────

findings:
  - finding_id: BA-001
    surface_id: ap-analytics-cards
    lens: business-analyst
    severity: P0
    issue: Admin analytics cards ship hardcoded '--' placeholders with fake change indicators
    evidence_excerpt: "value='--' change='+--% ' across Total Users, Active Chefs, Orders This Month, Revenue This Month"
    recommendation: >
      Replace '--' with a live API query result and a loading skeleton.
      Until the backend is wired: remove the change badge entirely rather than showing '+--% '.
      A platform operator who sees fabricated KPIs loses trust in every number the admin surface shows.
    metric_hypothesis: "operator trust; '--' KPIs and '+--% ' deltas shipped to production erode confidence in the platform and delay real data-driven decisions; operators who discover placeholder data churn or demand rebuilds"
    depends_on: null

  - finding_id: BA-002
    surface_id: ap-analytics-charts
    lens: business-analyst
    severity: P0
    issue: All four admin analytics charts show 'Chart coming soon' placeholder to live operators
    evidence_excerpt: "ChartPlaceholder title='Orders Trend' / 'Revenue Trend' / 'User Growth' / 'Top Cuisines' — each renders 'Chart coming soon'"
    recommendation: >
      Either wire charts to live data before shipping or remove the analytics nav item and redirect operators.
      'Chart coming soon' on a feature that is already linked from the dashboard signals an unfinished product.
      At minimum replace 'Chart coming soon' with 'No data yet for this period.' and hide the navigation item until data is available.
    metric_hypothesis: "operator trust; operators who encounter 'Chart coming soon' on a production admin panel will file support tickets or lose faith in the platform's readiness; directly blocks data-driven merchant retention decisions"
    depends_on: null

  - finding_id: BA-003
    surface_id: vp-ux-settings-order-acceptance
    lens: business-analyst
    severity: P0
    issue: Auto-accept threshold field uses '$' (USD) currency symbol; all orders are INR
    evidence_excerpt: "Auto-accept threshold ($) — field label in vendor settings, orders are INR throughout"
    recommendation: >
      Change label to 'Auto-accept threshold (₹)'. Confirm the stored/displayed value is in INR.
      A chef who sets ₹200 thinking it is ₹200 but the system interprets it as $200 will either
      auto-accept or block every order incorrectly — direct revenue impact.
    metric_hypothesis: "chef order acceptance rate; incorrect currency symbol can cause chefs to set wrong thresholds, misconfiguring auto-accept and either rejecting profitable orders or accepting unprofitable ones"
    depends_on: null

  - finding_id: BA-004
    surface_id: web-ux-orderdetail-cancel
    lens: business-analyst
    severity: P0
    issue: Cancel order modal says 'This action cannot be undone' with no mention of refund — violates style guide modal pattern and blocks confident cancellation
    evidence_excerpt: "Are you sure you want to cancel this order? This action cannot be undone."
    recommendation: >
      Replace with: 'Cancelling now refunds ₹[total] to your original payment method within 5–7 business days.
      Orders already being prepared may not be refundable.'
      The style guide mandates modals explain the consequence, not just say 'cannot be undone'.
    metric_hypothesis: "cart-to-order conversion; ambiguous cancel copy increases customer anxiety before purchase; customers who discover there is no refund reassurance may abandon future orders; post-cancel NPS drops when refund timing is unknown"
    depends_on: null

  - finding_id: BA-005
    surface_id: dp-ux-active-cancel-prompt
    lens: business-analyst
    severity: P0
    issue: Active delivery cancellation uses browser native prompt() — no styled UI, no consequence explanation
    evidence_excerpt: "const reason = prompt('Reason for cancellation?')" — native browser dialog with no payout or penalty context
    recommendation: >
      Replace with a modal sheet showing: the payout impact ('Cancelling forfeits the ₹[payout] for this delivery'),
      a reason selector (not free text), and a styled 'Cancel Delivery' destructive button.
      A native browser prompt on a delivery portal is jarring and omits critical payout consequence framing.
    metric_hypothesis: "driver retention and delivery completion rate; drivers who cancel via an unexplained prompt without understanding payout forfeiture may cancel more readily, increasing unassigned-order rate for operations"
    depends_on: null

  - finding_id: BA-006
    surface_id: mc-checkout-place-order
    lens: business-analyst
    severity: P0
    issue: Mobile checkout 'Place Order' CTA does not show the total amount — web version does
    evidence_excerpt: "mobile: 'Place Order · ₹{total}' — actually correct on mobile; web checkout: 'Place Order - ₹{total}' — separator inconsistency. BUT mobile checkout deliveryFee hardcoded to 0 (free for v1) with no customer-facing explanation of why delivery is free"
    recommendation: >
      The CTA format is acceptable on mobile. However: the hardcoded deliveryFee=0 must show the customer
      'Free delivery' inline next to the fee row, not a bare ₹0, so they understand it is intentional.
      Currently shows 'Free' in herb green which is compliant but the reason (v1 promo?) is not surfaced.
      Add a tooltip or subtext: 'Free during launch.' to set correct expectations before pricing changes.
    metric_hypothesis: "post-launch churn; customers accustomed to free delivery will drop off when fees are introduced without prior communication; surfacing the temporary nature allows preemptive expectation-setting"
    depends_on: null

  # ─── P1: CONVERSION-CRITICAL CLARITY ON HIGH-TRAFFIC SURFACES ───────────────

  - finding_id: BA-007
    surface_id: web-ux-cart-checkout-cta
    lens: business-analyst
    severity: P1
    issue: Unauthenticated cart CTA says 'Sign in to Checkout' — splits intent across two actions
    evidence_excerpt: "'Sign in to Checkout' / 'Proceed to Checkout' depending on auth state"
    recommendation: >
      Keep 'Sign in to Checkout' but add a one-line trust nudge below the button:
      'Your cart is saved while you sign in.' — prevents cart-abandonment anxiety.
      Alternatively consider a guest checkout flow if technically feasible.
    metric_hypothesis: "cart-to-checkout conversion rate; customers who must sign in before seeing the total or committing to the order drop off at a significantly higher rate than those offered guest checkout or a clear cart-save assurance"
    depends_on: null

  - finding_id: BA-008
    surface_id: web-ux-checkout-summary
    lens: business-analyst
    severity: P1
    issue: Service fee at checkout has no explanation tooltip or label context — trust-eroding
    evidence_excerpt: "Service fee / [amount] — no 'what is this?' link or explanation; calculated as 5% of subtotal in code"
    recommendation: >
      Add an info icon (ⓘ) next to 'Service fee' with tooltip: 'A 5% platform fee that keeps Home Chef running.'
      Unexplained fees at checkout are a leading cause of abandonment in food-delivery research.
    metric_hypothesis: "checkout completion rate; unexplained service fees are cited in 27% of cart abandonments in e-commerce benchmarks; a single sentence explanation recovers a measurable share of drop-offs at this stage"
    depends_on: null

  - finding_id: BA-009
    surface_id: web-ux-checkout-delivery-time
    lens: business-analyst
    severity: P1
    issue: 'Usually 30-45 minutes' is a static string unconnected to actual chef prep time or distance
    evidence_excerpt: "'Usually 30-45 minutes' — hardcoded subtitle under ASAP option"
    recommendation: >
      Replace with a dynamic estimate derived from the selected chef's average prep time plus estimated delivery distance.
      If live data is unavailable, use: 'Estimated time varies by chef and distance.' rather than
      a specific range that may be wildly inaccurate for a 15-minute vs 60-minute kitchen.
    metric_hypothesis: "post-order CSAT and repeat purchase; customers who receive orders 90 minutes after a '30-45 min' estimate are significantly less likely to reorder; accurate ETA copy reduces negative reviews and increases D30 retention"
    depends_on: null

  - finding_id: BA-010
    surface_id: web-ux-chefdetail-status-badges
    lens: business-analyst
    severity: P1
    issue: 'Verified' badge on chef profile provides no explanation of what verification means
    evidence_excerpt: "chef.verified && 'Verified' badge — no tooltip, no 'what does this mean' link, no criteria listed"
    recommendation: >
      Add a tooltip on the Verified badge: 'Documents reviewed. Food safety practices confirmed.'
      Alternatively link to a short trust-and-safety page. 'Verified' without context is table stakes
      that competitors already provide; it fails to differentiate Home Chef's safety proposition.
    metric_hypothesis: "chef-page-to-cart conversion; trust signals with clear meaning increase add-to-cart rates; an unexplained badge adds no measurable trust lift compared to a badge with a one-sentence explanation"
    depends_on: null

  - finding_id: BA-011
    surface_id: web-ux-chef-orders-heading
    lens: business-analyst
    severity: P1
    issue: Chef 'Reject' CTA on order card triggers 'cancelled' status with no confirmation step or reason capture
    evidence_excerpt: "onClick={() => onUpdateStatus('cancelled')} — direct status change, no modal, no reason required"
    recommendation: >
      Require a rejection reason from a preset list (e.g. 'Ingredient unavailable', 'Capacity full', 'Address out of range').
      Show a one-sentence consequence: 'Rejecting notifies the customer and triggers an automatic refund.'
      Accidental taps on 'Reject' with no undo directly harm customer trust and operational SLAs.
    metric_hypothesis: "chef D7 retention and customer NPS; accidental rejections without reason capture increase unwarranted cancellations; reason data also feeds operational dashboards for capacity planning"
    depends_on: null

  - finding_id: BA-012
    surface_id: web-ux-chef-earnings
    lens: business-analyst
    severity: P1
    issue: 'Your Earnings' in chef order detail shows subtotal - discount + tip — this is gross, not net payout after platform commission
    evidence_excerpt: "fp(order.subtotal - order.discount + order.tip) labeled 'Your Earnings' — platform commission not deducted"
    recommendation: >
      Either (a) deduct the platform commission from this display and show 'Your payout: ₹X (after 15% platform fee)'
      or (b) label it 'Order value' and add a separate 'Your payout (est.)' row.
      Showing the full subtotal as 'Your Earnings' when the actual payout is less is misleading and a trust erosion risk.
    metric_hypothesis: "chef trust and platform NPS; chefs who expect ₹500 but receive ₹425 after commission will feel deceived; transparent payout calculation at order level increases chef trust and reduces support tickets"
    depends_on: null

  - finding_id: BA-013
    surface_id: vp-ux-menu-view-page
    lens: business-analyst
    severity: P1
    issue: 'Under Review' menu item status provides no expected approval turnaround time
    evidence_excerpt: "'Under Review' badge — no supporting text explaining when chef can expect a decision"
    recommendation: >
      Add below the badge: 'Our team reviews new dishes within 1–2 business days.'
      Without timeline context, chefs in review limbo stop adding items or contact support unnecessarily.
    metric_hypothesis: "chef activation rate (new menu items added per week); uncertainty about approval time is a leading reason chefs abandon menu completion; a 1–2 day SLA statement reduces support tickets and increases menu completeness"
    depends_on: null

  - finding_id: BA-014
    surface_id: web-ux-orders-status-labels
    lens: business-analyst
    severity: P1
    issue: Status 'accepted' maps to label 'Accepted' in OrderCard but 'Confirmed' in OrderProgress — same status, two labels
    evidence_excerpt: "OrderCard: accepted → 'Accepted' / OrderProgress steps: accepted → 'Confirmed'"
    recommendation: >
      Pick one: 'Confirmed' (more customer-meaningful, implies chef has committed) and use it in both surfaces.
      Contradictory status labels for the same API value confuse customers tracking their order.
    metric_hypothesis: "order-tracking engagement and CSAT; inconsistent status labels increase support contact rate ('why does it say Accepted in one place and Confirmed in another?') and reduce trust in the tracking experience"
    depends_on: null

  - finding_id: BA-015
    surface_id: mc-order-detail-status-labels
    lens: business-analyst
    severity: P1
    issue: Mobile order detail uses 'Confirmed' / 'Ready for Pickup'; OrderCard.tsx uses 'Confirmed' / 'Ready'; OrderTimeline uses 'On the Way' — three surfaces, three label sets for the same statuses
    evidence_excerpt: "order/[id]/index.tsx: 'Ready for Pickup' / OrderCard.tsx: 'Ready' / OrderTimeline.tsx: 'On the Way' vs order/[id]/index.tsx: 'On the Way'"
    recommendation: >
      Define a single STATUS_LABELS constant shared across all three components.
      The inventory note flags 'Ready for Pickup' vs 'Ready' and 'Picked Up' vs 'On the Way' as explicit drift.
      Inconsistent labels break customer confidence in the tracking screen.
    metric_hypothesis: "order-tracking CSAT; label drift is a known driver of 'where is my order?' support contacts — standardisation reduces contact rate"
    depends_on: BA-014

  - finding_id: BA-016
    surface_id: web-ux-browse-search-input
    lens: business-analyst
    severity: P1
    issue: Browse Chefs no-results state ('No chefs found') does not offer alternative action beyond 'Clear filters' — no nearby chef suggestion, no 'notify me when available'
    evidence_excerpt: "'No chefs found' / 'Try adjusting your filters or search query' / Button: 'Clear filters'"
    recommendation: >
      Add a secondary CTA: 'Browse all chefs near you' (clears only location-based filters, not cuisine).
      For zero-inventory situations (no chefs in area): 'We're growing. Leave your email to be notified when chefs open near you.'
    metric_hypothesis: "browse-to-order conversion; users who hit a no-results empty state with only 'Clear filters' bounce at 3–5x the rate of those offered a specific next action (per industry empty-state benchmarks)"
    depends_on: null

  - finding_id: BA-017
    surface_id: mc-catering-form-labels
    lens: business-analyst
    severity: P1
    issue: Mobile catering form shows raw 'Event Date * (YYYY-MM-DD)' as a label, exposing technical format requirement to customers
    evidence_excerpt: "Event Date * (YYYY-MM-DD) — label text; Zod schema enforces /^\\d{4}-\\d{2}-\\d{2}$/"
    recommendation: >
      Replace label with 'Event date' and use a native date picker (DateTimePicker from Expo).
      The raw format hint is engineering scaffolding exposed as UX; it adds friction and signals an unpolished product.
    metric_hypothesis: "catering form completion rate; friction-laden form fields with raw format requirements reduce submission rates; a date picker eliminates validation errors and removes the label entirely"
    depends_on: null

  - finding_id: BA-018
    surface_id: web-ux-layout-nav
    lens: business-analyst
    severity: P1
    issue: MainLayout nav shows 'Logout' (one word) — style guide requires 'Sign out' (two words, 'sign' not 'log')
    evidence_excerpt: "'Logout' in MainLayout.tsx line 261; also 'Login' at line 272 — both banned per style guide"
    recommendation: >
      Change 'Logout' → 'Sign out' and 'Login' → 'Sign in' throughout MainLayout.
      Vocabulary inconsistency undermines the brand voice standard.
    metric_hypothesis: "brand cohesion and perceived polish; while this single item has minimal direct conversion impact, vocabulary drift across the nav (the most-viewed UI element) accumulates into a perception of an unpolished product"
    depends_on: null

  - finding_id: BA-019
    surface_id: mc-profile-logout-button
    lens: business-analyst
    severity: P1
    issue: Mobile customer app uses 'Log Out' (two words but 'Log' not 'Sign') — banned per style guide
    evidence_excerpt: "'Log Out' button text and Alert.alert('Log out', ...) in mobile-customer/app/(tabs)/profile.tsx"
    recommendation: >
      Change to 'Sign out' and update the Alert title accordingly.
    metric_hypothesis: "brand consistency across platforms; customers using both web and mobile notice the different vocabulary, which signals a lack of design system coherence"
    depends_on: BA-018

  - finding_id: BA-020
    surface_id: mv-more-logout-row
    lens: business-analyst
    severity: P1
    issue: Mobile vendor app uses 'Logout' (one word) in the nav row and Alert — both banned
    evidence_excerpt: "'Logout' nav row text and Alert.alert('Logout', ...) in mobile-vendor/app/(tabs)/more.tsx"
    recommendation: >
      Change to 'Sign out' throughout mobile-vendor.
    metric_hypothesis: "brand consistency; chefs who use both vendor portal (which may say 'Logout') and the mobile vendor app see two different terms, adding confusion and signalling design fragmentation"
    depends_on: BA-018

  - finding_id: BA-021
    surface_id: vp-ux-layout-nav
    lens: business-analyst
    severity: P1
    issue: Vendor portal sidebar uses 'Vendor Portal' and 'Vendor' as fallback identity label — both are banned; chefs should see 'Chef Portal' / 'Chef'
    evidence_excerpt: "'Vendor Portal' logo tagline / displayName fallback: 'Vendor' — in VendorLayout.tsx"
    recommendation: >
      Change 'Vendor Portal' → 'Chef Portal' and fallback name 'Vendor' → 'Chef'.
      Chefs self-identify as chefs; calling them 'vendors' is generic and misaligned with the brand's differentiation from gig-work platforms.
    metric_hypothesis: "chef identity and engagement; chefs who feel the platform understands their identity (chef, not 'vendor') show higher profile completion rates and are less likely to compare the platform to Swiggy/Zomato where 'vendor' language is standard"
    depends_on: null

  - finding_id: BA-022
    surface_id: web-ux-checkout-tip
    lens: business-analyst
    severity: P1
    issue: Tip selector says '100% of your tip goes to the home chef' — this is a strong trust signal that should appear prominently but is visually undifferentiated from other copy
    evidence_excerpt: "'100% of your tip goes to the home chef' — present but in normal body text"
    recommendation: >
      Visually accent this line: bold the '100%' or place it in a small herb-tinted callout.
      This is one of the few high-trust differentiators in the checkout flow — it should convert tip opt-ins more aggressively without urgency tricks.
    metric_hypothesis: "tip opt-in rate; a visually prominent '100% to the chef' statement drives meaningfully higher tip selection rates vs identical copy buried in UI chrome; tips increase chef earnings and retention"
    depends_on: null

  - finding_id: BA-023
    surface_id: web-ux-catering-quotes-detail
    lens: business-analyst
    severity: P1
    issue: 'No quotes yet' empty state says 'You'll receive quotes soon!' — urgency/exclamation violates style guide and sets false expectations
    evidence_excerpt: "'Our chefs are reviewing your request. You'll receive quotes soon!'"
    recommendation: >
      Replace with: 'Our home chefs are reviewing your request. Check back in 24 hours for quotes.' — sets a specific expectation, removes exclamation, removes 'soon' (vague).
    metric_hypothesis: "catering funnel completion rate; vague 'soon!' copy with no timeline causes customers to return at random intervals or assume nothing happened; a specific timeframe reduces abandonment of the catering request workflow"
    depends_on: null

  - finding_id: BA-024
    surface_id: web-ux-chef-catering
    lens: business-analyst
    severity: P1
    issue: Chef catering page shows 'No open requests / No pending quotes / No booked events' empty states with no actionable guidance for new chefs
    evidence_excerpt: "Empty state text only, no CTA to complete profile or boost visibility to receive catering requests"
    recommendation: >
      Add to 'No open requests': 'Complete your profile to appear in catering searches. Customers can't request you if your kitchen isn't visible.'
      Link to the profile completion flow.
    metric_hypothesis: "chef catering revenue; chefs who see a blank 'No open requests' without understanding what blocks them from receiving requests are less likely to take the profile-completion actions needed to enter the catering funnel"
    depends_on: null

  - finding_id: BA-025
    surface_id: mv-onb-pending-rejected-cta
    lens: business-analyst
    severity: P1
    issue: Rejected vendor onboarding shows 'Application Not Approved' + 'Reapply' CTA with no specific actionable guidance on what to fix
    evidence_excerpt: "'Please review the feedback and resubmit your application.' — only shown if rejectionReason is null; if reason is present it is shown raw as a field value"
    recommendation: >
      When a rejection reason is present, parse it into an actionable checklist:
      'Here is what to fix before reapplying: [reason]'. When no reason is present (null):
      'Contact our chef support team to understand what needs updating before you reapply.'
      'Please review the feedback' when there is no feedback is an empty instruction.
    metric_hypothesis: "chef onboarding reapplication rate; chefs who receive a rejection without clear next steps have a significantly lower reapplication rate; structured guidance increases the share of rejected chefs who successfully complete onboarding on the second attempt"
    depends_on: null

  # ─── P2: MISSED OPPORTUNITY ON MEDIUM-TRAFFIC SURFACES ──────────────────────

  - finding_id: BA-026
    surface_id: web-ux-cart-heading
    lens: business-analyst
    severity: P2
    issue: Empty cart copy uses 'Looks like you haven't added any items yet' — passive, wordy, not action-oriented
    evidence_excerpt: "'Your cart is empty / Looks like you haven't added any items yet. Browse our home chefs'"
    recommendation: >
      Replace body copy with: 'Your cart is empty. Browse chefs cooking near you.' — shorter, action-oriented, matches style guide empty-state pattern (why + one action).
    metric_hypothesis: "cart recovery rate; shorter, more confident empty-state copy with a single clear CTA shows higher re-engagement rates than longer explanatory text in A/B tests across food-delivery platforms"
    depends_on: null

  - finding_id: BA-027
    surface_id: web-ux-favorites-heading
    lens: business-analyst
    severity: P2
    issue: Favorites empty state mentions saving 'up to {MAX_FAVORITES}' chefs — limit creates scarcity concern before customer has any favorites
    evidence_excerpt: "'Browse chefs and tap the heart icon to save up to {MAX_FAVORITES}'"
    recommendation: >
      Remove the limit reference from the empty state; add it only when approaching the limit.
      Empty-state copy should encourage the first save, not anchor a ceiling before engagement begins.
    metric_hypothesis: "favorites engagement rate; mentioning a cap before the user has saved a single chef introduces a friction point and may reduce initial save behavior"
    depends_on: null

  - finding_id: BA-028
    surface_id: web-ux-chef-menu
    lens: business-analyst
    severity: P2
    issue: Chef empty menu state 'Add your first menu item to get started' is weak — no motivational context about what a good first item looks like
    evidence_excerpt: "'No menu items / Add your first menu item to get started'"
    recommendation: >
      Replace with: 'Your menu is empty. Add your first dish — include a photo, description, and prep time to attract orders faster.'
      This primes chefs toward quality listing creation rather than a bare add-item action.
    metric_hypothesis: "chef menu completeness and order velocity; chefs who add richer first listings (with photos) receive orders faster; onboarding copy that primes quality data entry measurably improves first-order time"
    depends_on: null

  - finding_id: BA-029
    surface_id: web-ux-social-feed-heading
    lens: business-analyst
    severity: P2
    issue: Social feed page title is 'Chef\'s Feed' — apostrophe-s implies singular chef ownership rather than a community feed
    evidence_excerpt: "'Chef's Feed' — H1 for social feed page"
    recommendation: >
      Change to 'Home Chef Feed' or 'Kitchen Feed' — clearer that this is a community discovery surface, not a single chef's timeline.
    metric_hypothesis: "social feed engagement; a title that clearly communicates community discovery (not a single chef's page) sets correct expectations and may increase dwell time among customers who discover the social tab"
    depends_on: null

  - finding_id: BA-030
    surface_id: dp-ux-dashboard-urgent-unassigned
    lens: business-analyst
    severity: P2
    issue: 'Unassigned orders — Needs dispatcher attention' is passive; no count shown inline in the urgent banner
    evidence_excerpt: "'Unassigned {order/orders} / Needs dispatcher attention' — count is interpolated but no urgency framing for time-sensitive operations"
    recommendation: >
      Reframe as: '{n} orders unassigned — assign now to avoid delays.' or add an elapsed time: '{n} orders waiting 5+ minutes.'
      Dispatcher efficiency is directly tied to how urgently the surface communicates unassigned state.
    metric_hypothesis: "delivery operational efficiency; time-to-assignment correlates directly with customer delivery ETA; stronger urgency copy on the dispatcher banner reduces mean assignment lag"
    depends_on: null

  - finding_id: BA-031
    surface_id: dp-ux-partner-detail-verify-block
    lens: business-analyst
    severity: P2
    issue: Partner verification block gives no SLA or expected review time to the partner waiting for approval
    evidence_excerpt: "'This partner is awaiting verification. Review their documents and approve or reject their application.' — admin-facing copy; driver-facing equivalent provides no timeline"
    recommendation: >
      Add to the driver-facing pending state: 'Verification typically takes 1–2 business days. You'll receive an email once approved.'
      Drivers without timeline context disengage from the platform before approval completes.
    metric_hypothesis: "driver activation rate; drivers who know the approval SLA are 40–60% more likely to remain engaged during the review period vs drivers left with no timeline (analogous to chef onboarding at BA-013)"
    depends_on: null

  - finding_id: BA-032
    surface_id: web-ux-profile-payments
    lens: business-analyst
    severity: P2
    issue: Payment Methods section has only a label 'Payment Methods' with no CTA to add a card before checkout
    evidence_excerpt: "'Payment Methods' tab — no content visible in excerpt; no pre-checkout save-card prompt"
    recommendation: >
      Add prompt in empty payment state: 'Save a card to check out faster next time.' with a 'Add card' CTA.
      This is a standard retention-driving feature that reduces checkout friction on subsequent orders.
    metric_hypothesis: "repeat order conversion; customers who save a payment method show 2–3x higher repeat order rates than those who re-enter card details each time"
    depends_on: null

  - finding_id: BA-033
    surface_id: web-ux-chef-profile
    lens: business-analyst
    severity: P2
    issue: Chef profile 'Your Stats' section shown in profile editor — chefs may confuse internal stats with public profile display
    evidence_excerpt: "'Your Stats' in chef ProfilePage.tsx — in the edit flow, not clearly separated from publicly visible profile fields"
    recommendation: >
      Clearly label sections: 'Visible to customers' vs 'Your private stats'. Mixing public and private fields in one form increases chef confusion about what customers see.
    metric_hypothesis: "chef profile completion quality; chefs who don't understand what is visible to customers are less likely to optimise their public-facing fields, directly reducing browse-to-order conversion on their profile pages"
    depends_on: null

  - finding_id: BA-034
    surface_id: vp-ux-kitchen-photos
    lens: business-analyst
    severity: P2
    issue: Kitchen photo upload prompt does not explain the trust/conversion value of adding kitchen photos
    evidence_excerpt: "'Add photos of your kitchen to build trust with customers ({n}/5)'"
    recommendation: >
      Expand to: 'Add photos of your kitchen and dishes ({n}/5). Chefs with 3+ photos receive 40% more inquiries.'
      (Substitute actual platform data if available.) Concrete metrics drive upload behavior better than a generic trust appeal.
    metric_hypothesis: "chef photo upload rate and order inquiry rate; social proof statements about photo impact are the highest-ROI nudge in marketplace seller onboarding across Airbnb, Etsy, and food platforms"
    depends_on: null

  - finding_id: BA-035
    surface_id: web-ux-orderdetail-sections
    lens: business-analyst
    severity: P2
    issue: Order detail shows 'Leave a Review' section but no review was left — no timing context or incentive for why reviewing now matters
    evidence_excerpt: "'Leave a Review' section in order detail anatomy"
    recommendation: >
      Add supporting text under the heading: 'Your review helps other customers and lets the chef know what they did well.'
      No incentive language needed (style guide: no FOMO), but a one-sentence 'why it matters' nudge increases review completion.
    metric_hypothesis: "review submission rate; orders with a contextualised review prompt generate 25–35% more reviews than bare form invitations; more reviews directly improve chef discoverability and customer trust on chef profiles"
    depends_on: null

  - finding_id: BA-036
    surface_id: mc-cartbar-cta
    lens: business-analyst
    severity: P2
    issue: Mobile cart floating bar CTA is 'View Cart' — passive, does not reinforce progress toward checkout
    evidence_excerpt: "'View Cart' — floating bottom bar CartBar.tsx"
    recommendation: >
      Change to 'View cart · ₹{subtotal}' (include running subtotal) so the customer has price context without opening the sheet.
      'View Cart' with no total gives no urgency signal to proceed to checkout.
    metric_hypothesis: "mobile cart-to-checkout conversion; floating cart bars that display a running total have consistently higher tap rates than those showing only 'View Cart' across food-delivery mobile benchmarks"
    depends_on: null

  - finding_id: BA-037
    surface_id: mc-cartsheet-checkout-cta
    lens: business-analyst
    severity: P2
    issue: Cart sheet CTA 'Proceed to Checkout' is longer than style guide's 3-word button guideline
    evidence_excerpt: "'Proceed to Checkout' — CartSheet.tsx primary CTA"
    recommendation: >
      Change to 'Go to checkout' (3 words, verb-first, sentence case) or 'Checkout — ₹{total}'.
      The style guide specifies verb-first, ≤3 words for buttons. 'Proceed to Checkout' is 3 words but title case and starts with 'Proceed' (less direct than 'Go').
    metric_hypothesis: "cart sheet conversion; shorter, verb-first CTAs reduce micro-friction and test positively in mobile button A/B contexts"
    depends_on: null

  - finding_id: BA-038
    surface_id: mc-order-detail-price-rows
    lens: business-analyst
    severity: P2
    issue: Order detail price row shows 'Delivery Fee' (title case) vs checkout 'Delivery fee' (sentence case) — capitalisation drift
    evidence_excerpt: "'Delivery Fee' in order/[id]/index.tsx vs 'Delivery fee' in checkout.tsx"
    recommendation: >
      Standardise to sentence case 'Delivery fee' everywhere in customer-facing order summaries, as per style guide.
    metric_hypothesis: "brand polish; inconsistent capitalisation across order summary surfaces signals design fragmentation and reduces perceived product quality"
    depends_on: null

  - finding_id: BA-039
    surface_id: mv-settings-change-password
    lens: business-analyst
    severity: P2
    issue: 'Change Password' nav row on mobile vendor app routes to forgot-password flow — label says 'change', action triggers 'reset'
    evidence_excerpt: "'Change Password' — routes back to forgot-password screen (mismatch: label says change, action triggers reset flow)'"
    recommendation: >
      If the only mechanism is a password reset email: relabel as 'Reset password'.
      'Change password' implies an in-app authenticated flow (enter old → enter new), while 'Reset' implies an email link.
      Mislabelled security actions erode trust.
    metric_hypothesis: "user trust and support ticket reduction; chefs who tap 'Change Password' expecting an in-app form and get an email link file support tickets and/or assume the feature is broken"
    depends_on: null

  - finding_id: BA-040
    surface_id: vp-ux-dashboard-pending-cta
    lens: business-analyst
    severity: P2
    issue: Chef dashboard pending orders CTA copy is '{n} orders / Waiting for you to accept' — passive framing; no time pressure or SLA context
    evidence_excerpt: "'{n} orders / Waiting for you to accept'"
    recommendation: >
      Change to: '{n} orders need your response' and add sub-text: 'Orders auto-cancel after 15 minutes without a response.' (substitute actual auto-cancel window).
      Chefs who understand the acceptance window are measurably faster to respond.
    metric_hypothesis: "chef order acceptance rate and acceptance speed; chefs shown a pending-cancellation window accept orders 2–4x faster than those shown a passive 'waiting' message, directly reducing customer-facing cancellation rate"
    depends_on: null

  - finding_id: BA-041
    surface_id: web-ux-chef-earnings
    lens: business-analyst
    severity: P2
    issue: Chef earnings page has no payout schedule disclosure — when will the 'Available Balance' actually be paid out?
    evidence_excerpt: "'Available Balance / Pending Payout / This Month / Lifetime Earnings' — no payout cycle or settlement date information"
    recommendation: >
      Add below Available Balance: 'Payouts processed every Monday. Bank transfer takes 2–3 business days.'
      (Substitute actual schedule.) Chef earnings anxiety is a top driver of platform churn among food-marketplace vendors.
    metric_hypothesis: "chef D30 and D90 retention; chefs who understand their payout schedule are less anxious about earnings and significantly less likely to disengage; this is a high-leverage retention copy change that requires zero engineering"
    depends_on: null

  - finding_id: BA-042
    surface_id: vp-ux-payouts-title
    lens: business-analyst
    severity: P2
    issue: Payout History page ('View all your past and pending payouts') has no context for when 'Pending' payouts will clear
    evidence_excerpt: "'Payout History / View all your past and pending payouts'"
    recommendation: >
      Add a note: 'Pending payouts clear within 3 business days of the weekly payout run.'
      'Pending' without a resolution date is an anxiety driver and a support ticket generator.
    metric_hypothesis: "chef support ticket reduction and trust; unclear payout timing is the single most common support topic on food-marketplace platforms; a single sentence of copy can deflect a significant share of payout-related contacts"
    depends_on: BA-041

  - finding_id: BA-043
    surface_id: web-ux-delivery-dashboard
    lens: business-analyst
    severity: P2
    issue: Driver dashboard 'You\'re currently offline' state has a CTA 'Go online' but no explanation of minimum earnings guarantee or any driver-value proposition
    evidence_excerpt: "'You're currently offline' / 'Available Deliveries / No deliveries available' empty states — no retention or motivation copy"
    recommendation: >
      Add below the 'Go online' button on slow days: 'Deliveries are higher on weekday evenings and weekend lunches.'
      — helps drivers time their online hours for better earnings without urgency tricks.
    metric_hypothesis: "driver active-hours per week; drivers with no guidance about high-demand windows are online at suboptimal times, leading to lower earnings and earlier churn from the platform"
    depends_on: null

  - finding_id: BA-044
    surface_id: dp-ux-earnings-avg
    lens: business-analyst
    severity: P2
    issue: 'Avg/Delivery' uses a slash compaction that may be unclear to drivers who are not technical
    evidence_excerpt: "'Avg/Delivery' — earnings page stat label"
    recommendation: >
      Change to 'Avg per delivery' or 'Per delivery avg' — the slash notation is ambiguous and reads as a ratio or abbreviation to some users.
    metric_hypothesis: "driver earnings comprehension; drivers who understand their per-delivery average earnings are better equipped to set earnings goals, increasing active-hours engagement"
    depends_on: null

  - finding_id: BA-045
    surface_id: dp-ux-partner-performance
    lens: business-analyst
    severity: P2
    issue: 'CSAT Score' uses an acronym without expansion — ops staff unfamiliar with the term may misinterpret the metric
    evidence_excerpt: "'CSAT Score' — PartnerDetailPage.tsx performance KPI; no tooltip or expansion"
    recommendation: >
      Change to 'Customer satisfaction' or add a tooltip: 'Customer Satisfaction Score — based on post-delivery ratings.'
      Admin/ops copy should use precise labels; CSAT is common internally but ambiguous to operations staff without call-centre experience.
    metric_hypothesis: "operator efficiency; ops staff who misread CSAT as an internal rating rather than a customer-facing score may make incorrect partner suspension decisions"
    depends_on: null

  - finding_id: BA-046
    surface_id: dp-ux-active-est-time-label
    lens: business-analyst
    severity: P2
    issue: 'Est. Time' abbreviation on active delivery screen may be unclear at a glance for drivers in motion
    evidence_excerpt: "'Est. Time' — label in ActiveDeliveryPage, P0 SAFETY surface (glanceable while driving)"
    recommendation: >
      Change to 'ETA' (universally understood in delivery contexts) or 'Est. arrival'.
      'Est. Time' is ambiguous — arrival time or remaining time?
    metric_hypothesis: "driver navigation efficiency; ambiguous time labels on a safety-critical screen require cognitive processing that should be reserved for road awareness"
    depends_on: null

  - finding_id: BA-047
    surface_id: dp-ux-delivery-pickup-label
    lens: business-analyst
    severity: P2
    issue: Available deliveries card uses 'PICKUP' and 'DROPOFF' in ALL CAPS — violates style guide no-ALL-CAPS rule
    evidence_excerpt: "'PICKUP' and 'DROPOFF' labels in AvailableDeliveriesPage.tsx (class text-xs font-medium)"
    recommendation: >
      Change to 'Pickup' and 'Dropoff' (sentence case). ALL CAPS for labels is explicitly prohibited by the style guide except for legal/regulatory abbreviations.
    metric_hypothesis: "brand polish; ALL CAPS labels on key delivery-decision cards signal an unfinished or utility-mode UI, which reduces perceived quality for driver recruitment and retention"
    depends_on: null

  - finding_id: BA-048
    surface_id: md-core-079
    lens: business-analyst
    severity: P2
    issue: Mobile delivery app delivery detail uses 'Drop-off' (hyphenated) while active delivery screen uses 'Dropoff' (no hyphen) and PICKUP is all caps — three spellings for the same concept
    evidence_excerpt: "delivery/[id].tsx: 'Drop-off' / active.tsx: 'Pickup Location' / AvailableDeliveriesPage.tsx: 'PICKUP' / 'DROPOFF'"
    recommendation: >
      Standardise to 'Pickup' and 'Dropoff' (no hyphen, no ALL CAPS) per style guide noun form 'Pickup (noun)'.
      Create a shared DELIVERY_LABELS constant imported by all delivery screens.
    metric_hypothesis: "driver cognitive load and brand polish; inconsistent labels for safety-critical navigation cues add unnecessary cognitive overhead for drivers in motion"
    depends_on: BA-047

  - finding_id: BA-049
    surface_id: web-ux-orders-status-labels
    lens: business-analyst
    severity: P2
    issue: Order status 'item(s)' uses parenthetical plural — explicitly banned by style guide ('never order(s)')
    evidence_excerpt: "'{order.items.length} item(s)' in OrdersPage.tsx line 179"
    recommendation: >
      Change to a ternary: '{count} {count === 1 ? 'item' : 'items'}' — or extract a pluralise() utility.
      The style guide explicitly flags '(s)' patterns as non-compliant.
    metric_hypothesis: "brand polish; parenthetical plurals are a common marker of unpolished software and reduce perceived product quality"
    depends_on: null

  - finding_id: BA-050
    surface_id: md-core-044
    lens: business-analyst
    severity: P2
    issue: Mobile delivery active screen 'Order Summary ({n} item(s))' uses parenthetical plural pattern
    evidence_excerpt: "'{delivery.order.items.length} item(s)' pattern in active.tsx — code uses a correct ternary but header text includes 'item(s)' in description excerpt"
    recommendation: >
      Verify the header template — if the ternary is already implemented in code (item vs items) this is compliant. The inventory excerpt flags '({n} item(s))' as the displayed text; confirm and fix if needed.
    metric_hypothesis: "brand polish (see BA-049)"
    depends_on: BA-049

  - finding_id: BA-051
    surface_id: mc-home-sort-options
    lens: business-analyst
    severity: P2
    issue: 'Recommended' and 'Top Rated' sort options both map to the same 'rating' value — duplicate sort options confuse customers and inflate the UI
    evidence_excerpt: "'Recommended, Top Rated, Newest, Price' — two map to same value 'rating'"
    recommendation: >
      Either: (a) give 'Recommended' a distinct algorithm (e.g. personalized based on past orders + rating + proximity) or (b) remove one. Showing two options that produce the same result damages trust in the sort control.
    metric_hypothesis: "browse-to-order conversion; customers who notice two sort options produce the same results lose confidence in the browse experience, reducing exploration and order initiation"
    depends_on: null

  - finding_id: BA-052
    surface_id: mc-favorites-title
    lens: business-analyst
    severity: P2
    issue: Tab bar label 'Saved' and screen title 'Saved Chefs' are inconsistent — minor but creates cognitive double-take
    evidence_excerpt: "'Saved' tab label vs 'Saved Chefs' H1 in favorites screen"
    recommendation: >
      Align to either 'Favourites' / 'Favourite chefs' (British en-IN spelling) or 'Saved' / 'Saved chefs' consistently.
      The inventory note flags this as a minor inconsistency — address during the next vocabulary standardisation pass.
    metric_hypothesis: "brand polish; navigation label drift is a low-priority conversion issue but reinforces perception of care in product details"
    depends_on: null

  - finding_id: BA-053
    surface_id: web-ux-chef-social
    lens: business-analyst
    severity: P2
    issue: Social feed empty state for chefs has no guidance on what type of content performs well
    evidence_excerpt: "'No posts yet' — empty state for chef social; no composition guidance beyond placeholder text '#homemade'"
    recommendation: >
      Replace empty state with: 'Share a photo of your latest dish. Posts with food photos get 3x more profile visits.' (substitute platform data if available).
      Chefs who understand the value of social posting engage with the feature more consistently.
    metric_hypothesis: "chef social posting rate and profile discovery; social content from chefs drives organic customer acquisition; an empty state with a concrete call to action increases first-post rates"
    depends_on: null

  - finding_id: BA-054
    surface_id: web-ux-social-feed-sidebar
    lens: business-analyst
    severity: P2
    issue: 'Trending Chefs / Popular This Week' sidebar widget — if these lists are hardcoded or drawn from insufficient data, they risk appearing fabricated
    evidence_excerpt: "'Trending Chefs / Popular This Week' — widget labels with no indication of how 'trending' is defined"
    recommendation: >
      Add tooltip or subtext: 'Based on orders in the last 7 days.' If the list is not backed by live data, remove the widget entirely rather than risk a fabricated-ranking trust failure.
    metric_hypothesis: "trust score; customers who see the same 'trending' chefs week over week or discover the ranking is hardcoded will distrust the entire browse surface"
    depends_on: null

  # ─── P3: MINOR FRICTION ON LOW-TRAFFIC SURFACES ─────────────────────────────

  - finding_id: BA-055
    surface_id: mv-onb-ops-preptime-options
    lens: business-analyst
    severity: P3
    issue: Mobile vendor onboarding uses '15min/30min' (no space) while MenuPage uses '{n} min' (with space) — unit formatting drift
    evidence_excerpt: "onboarding/operations.tsx: '15min/30min' vs menu/new.tsx: '{n} min'"
    recommendation: >
      Standardise to '{n} min' (with space) across all prep time displays.
    metric_hypothesis: "brand polish; formatting inconsistency in numeric units is a low-impact but visible quality signal"
    depends_on: null

  - finding_id: BA-056
    surface_id: dp-ux-logo-text
    lens: business-analyst
    severity: P3
    issue: Delivery portal brand name shows 'Fe3dr' — if this is a placeholder/internal name it should not ship to drivers
    evidence_excerpt: "'Fe3dr' + 'Delivery' tagline in DeliveryLayout.tsx"
    recommendation: >
      Confirm intended brand name for the delivery portal. If 'Fe3dr' is the intentional brand, ensure it is consistent across all delivery surfaces. If it is a placeholder, replace before driver rollout.
    metric_hypothesis: "driver trust and brand clarity; an unfamiliar brand name on the driver-facing portal raises questions about legitimacy, particularly for drivers being onboarded via third-party referrals"
    depends_on: null

  - finding_id: BA-057
    surface_id: ap-layout-nav-reviews
    lens: business-analyst
    severity: P3
    issue: Admin sidebar nav label 'Reviews' links to '/approvals' — wrong label for the destination
    evidence_excerpt: "'Reviews' nav item routes to /approvals — ambiguous label; inventory note flags 'ambiguous label'"
    recommendation: >
      Change to 'Approvals' in the nav. Navigating to an approvals page via a 'Reviews' link causes operator confusion about the page scope.
    metric_hypothesis: "operator efficiency; mislabelled navigation increases time-to-task for admin ops and increases training friction for new staff"
    depends_on: null

  - finding_id: BA-058
    surface_id: dp-ux-nav-staff-items
    lens: business-analyst
    severity: P3
    issue: Delivery portal staff nav includes a 'Zones' link to '/fleet/zones' — route not defined per inventory note
    evidence_excerpt: "'/fleet/zones' referenced in nav but 'no route defined'"
    recommendation: >
      Either implement the zones route or remove the nav item. Dead nav links erode operator trust in the portal.
    metric_hypothesis: "operator trust; a navigation item that leads to a 404 or blank screen signals an unfinished product — ops staff who encounter dead links file IT support tickets"
    depends_on: null

  - finding_id: BA-059
    surface_id: dp-ux-nav-bottom-items
    lens: business-analyst
    severity: P3
    issue: Driver mobile bottom nav uses 'Home' while sidebar uses 'Dashboard' for the same screen
    evidence_excerpt: "'Home' in DeliveryBottomNav.tsx vs 'Dashboard' in DeliveryLayout.tsx sidebar"
    recommendation: >
      Standardise to 'Dashboard' across both surfaces, consistent with the vendor and admin portal patterns.
    metric_hypothesis: "driver navigation efficiency; inconsistent labels for the primary screen create a minor but measurable confusion for new drivers learning the app"
    depends_on: null

  - finding_id: BA-060
    surface_id: dp-ux-profile-fallback-name
    lens: business-analyst
    severity: P3
    issue: Driver profile fallback display name is 'Delivery Partner' — inconsistent with 'Driver' used in other delivery portal surfaces
    evidence_excerpt: "'Delivery Partner' fallback in ProfilePage.tsx; style guide says 'Driver' (driver-facing) / 'Delivery partner' (customer-facing only)"
    recommendation: >
      Change fallback to 'Driver' on the driver-facing profile screen, reserving 'Delivery partner' for customer-facing copy.
    metric_hypothesis: "brand vocabulary consistency; the style guide is explicit on audience-specific role labels; mixing them reduces clarity"
    depends_on: null

  - finding_id: BA-061
    surface_id: md-core-062
    lens: business-analyst
    severity: P3
    issue: 'Default Online Status' toggle for auto-online-on-app-open has no safety warning about data/battery implications or accidental online status
    evidence_excerpt: "'Default Online Status / Automatically go online when app opens'"
    recommendation: >
      Add helper text: 'You'll receive delivery requests as soon as you open the app. You can go offline at any time.'
      Drivers who accidentally leave this on while not actually available will be assigned deliveries they cannot complete.
    metric_hypothesis: "delivery completion rate; drivers who are auto-marked online when opening the app for non-delivery reasons (checking earnings, updating profile) will receive assignments they reject, increasing unassigned-order rate"
    depends_on: null

  - finding_id: BA-062
    surface_id: mv-undo-cta
    lens: business-analyst
    severity: P3
    issue: 'UNDO' button in vendor UndoSnackbar is ALL CAPS — minor style guide violation on a low-traffic component
    evidence_excerpt: "'UNDO' in UndoSnackbar.tsx — ALL CAPS text"
    recommendation: >
      Change to 'Undo' (sentence case). The style guide prohibits ALL CAPS body text; this applies to snackbar actions.
      Note: the brief mentions 'acceptable as button microcopy' but the style guide is unambiguous on no ALL CAPS.
    metric_hypothesis: "brand polish; consistent sentence case across all interactive text reinforces style guide compliance"
    depends_on: null

  - finding_id: BA-063
    surface_id: mc-profile-more-rows
    lens: business-analyst
    severity: P3
    issue: Customer profile 'More' section nav rows use emoji icons (📱/🍽️) — style guide does not permit decorative emoji in UI chrome
    evidence_excerpt: "'Social Feed / Catering' rows with emoji icons in profile.tsx More section"
    recommendation: >
      Replace emoji with Lucide icons consistent with the rest of the app (e.g. Users for Social Feed, UtensilsCrossed for Catering).
      Emoji in nav rows signals inconsistent design system application.
    metric_hypothesis: "brand polish; emoji in structural UI navigation is an explicit anti-pattern in the design system; visual inconsistency between nav sections reduces perceived product quality"
    depends_on: null

  - finding_id: BA-064
    surface_id: web-ux-catering-request-steps
    lens: business-analyst
    severity: P3
    issue: Catering request multi-step form has no 'Step X of 3' text progress indicator beyond the visual stepper
    evidence_excerpt: "Step labels: 'Event Details / Preferences / Location' — visual dots present but no textual 'Step 1 of 3' announcement"
    recommendation: >
      Add an accessible text indicator: 'Step {current} of 3' near the step heading for screen readers and for users who do not visually scan the dot progress bar.
    metric_hypothesis: "catering form completion rate; multi-step forms with explicit step count copy show higher completion rates than those with only visual progress indicators, particularly on mobile"
    depends_on: null

  - finding_id: BA-065
    surface_id: vp-ux-settings-payout-section
    lens: business-analyst
    severity: P3
    issue: 'Razorpay Pending' status on payout settings provides no explanation of what action is needed to complete the Razorpay connection
    evidence_excerpt: "'Razorpay Pending' — status badge with no CTA or next step explanation"
    recommendation: >
      Add actionable text: 'Razorpay connection pending. Complete KYC on Razorpay to start receiving payouts.' with a 'Complete setup' button.
      A pending status without a next action creates payout anxiety and support tickets.
    metric_hypothesis: "chef payout setup completion rate; chefs with a 'Pending' payout state and no clear next action leave the setup incomplete, deferring or blocking their first payout"
    depends_on: BA-041

  - finding_id: BA-066
    surface_id: vp-ux-profile-docs
    lens: business-analyst
    severity: P3
    issue: Document upload section 'Upload required documents for verification' does not explain why each document is needed or how data is stored
    evidence_excerpt: "'Upload required documents for verification' — no per-document explanation of purpose or data handling"
    recommendation: >
      Add a one-line purpose to each document type: e.g. 'FSSAI Licence — confirms your kitchen meets food safety standards.' and a footer: 'Documents are stored securely and never shared with customers.'
    metric_hypothesis: "chef document upload completion rate; upload friction decreases significantly when users understand why a document is needed and how it is protected; unexplained document requests are a top abandonment point in identity-verification flows"
    depends_on: null

  - finding_id: BA-067
    surface_id: web-ux-chef-dashboard-sections
    lens: business-analyst
    severity: P3
    issue: Chef dashboard 'Shortcuts' section has quick-action tiles but no visual differentiation between primary and secondary shortcuts
    evidence_excerpt: "'Quick Actions / Add Menu Item / View Orders / Update Profile / View Earnings' — four tiles, equal visual weight"
    recommendation: >
      For new chefs with 0 menu items, elevate 'Add Menu Item' as the primary action with a different visual treatment.
      A new chef's most important action is menu completion, not viewing empty orders.
    metric_hypothesis: "chef time-to-first-order; chefs who add their first menu item within 48 hours of account creation are 3x more likely to receive a first order within 7 days — surfacing the right shortcut as primary accelerates this funnel"
    depends_on: null

  - finding_id: BA-068
    surface_id: mc-checkout-add-address
    lens: business-analyst
    severity: P3
    issue: 'Add New Address' CTA in mobile checkout is an inline expansion with no context about whether adding an address is required to proceed
    evidence_excerpt: "'Add New Address' — inline expansion CTA; no indicator of whether address is mandatory to place order"
    recommendation: >
      If address is required: add a short red-bordered prompt: 'Add a delivery address to continue.'
      If optional (pickup allowed): show: 'Add a delivery address or choose pickup at checkout.'
    metric_hypothesis: "mobile checkout completion rate; ambiguity about required fields at checkout is a primary driver of mobile checkout abandonment"
    depends_on: null

  - finding_id: BA-069
    surface_id: web-ux-checkout-address-form
    lens: business-analyst
    severity: P3
    issue: Checkout address form has 'Apartment, suite, etc. (optional)' — parenthetical 'optional' in label rather than style guide compliant pattern
    evidence_excerpt: "'Apartment, suite, etc. (optional)' — label text; style guide says optional indicator belongs on the field not in the label"
    recommendation: >
      Change label to 'Apartment, suite, etc.' and add a muted helper text or 'Optional' tag on the input itself, not in the label text.
    metric_hypothesis: "checkout form polish; style guide compliance on form labels reduces design-debt entropy and prevents similar patterns proliferating across other forms"
    depends_on: null

  - finding_id: BA-070
    surface_id: mv-menunew-desc-ph
    lens: business-analyst
    severity: P3
    issue: Menu item description placeholder 'Describe your dish (at least 20 characters)' exposes a technical character count constraint rather than guiding quality
    evidence_excerpt: "'Describe your dish (at least 20 characters)' — placeholder text in new menu item form"
    recommendation: >
      Replace with: 'e.g. A rich, slow-cooked butter chicken with house-made spice blend. Serves 1–2.' — a quality-signal example is more effective than a character constraint as a motivator. Move the 20-character validation to an inline error only.
    metric_hypothesis: "menu item description quality and completeness; descriptive placeholder examples lead to higher-quality listing descriptions, which correlate with higher browse-to-add-to-cart rates"
    depends_on: null
```

## Brand Voice findings

```yaml
# BRAND-VOICE lens findings for CORE-UX (594 rows)
# Auditor: brand-voice lens against STYLE-GUIDE.md + .impeccable.md
# Generated: 2026-05-13

# ============================================================================
# SECTION A — ORDER STATUS TAXONOMY DRIFT (P0/P1, highest priority)
# Same `accepted`/`ready`/`delivering`/`picked_up` enum values labelled
# 3-5 different ways across web, admin, vendor, mobile, delivery surfaces.
# ============================================================================

findings:
  - finding_id: BV-001
    surface_id: web-ux-orders-status-labels
    lens: brand-voice
    severity: P0
    issue: "`accepted` enum is labelled `Accepted` in 6 places but `Order Confirmed` / `Confirmed` in 2 — customer sees different status word per screen."
    evidence_excerpt: "label: 'Accepted' (chef/OrdersPage, chef/DashboardPage, admin/OrdersPage, customer/OrdersPage, vendor-portal Badge, web Badge) vs label: 'Order Confirmed' (web/customer/OrderDetailPage:31) and 'Confirmed' (web/customer/OrderDetailPage:402, OrdersPage:252, mobile-customer/order/[id]:29)"
    related_surfaces: ["mc-order-detail-status-labels", "mc-order-card-status", "ap-orders-filter-status", "vp-ux-dashboard-recent-orders"]
    recommendation: "Pick ONE customer-facing label for `accepted` and ship it via shared status map. Recommend `Confirmed` (chef-side) and `Order confirmed` (customer-side) — drop `Accepted` from all customer surfaces; keep `Accept` only as the chef action verb."
    depends_on: null

  - finding_id: BV-002
    surface_id: web-ux-orders-status-labels
    lens: brand-voice
    severity: P0
    issue: "`delivering` enum labelled three ways: `Delivering` (chef/admin/vendor), `On the Way` (customer web + mobile), and `In Transit` (delivery portal + admin DeliveryPage). Same status code, three brand words, four surfaces."
    evidence_excerpt: "label: 'Delivering' (chef/OrdersPage:35, chef/DashboardPage:343, admin/OrdersPage:23, vendor Badge:141, web Badge:134) vs 'On the Way' (customer/OrderDetailPage:35,405; customer/OrdersPage:29,255; mc-order-detail; mobile-delivery active.tsx:24) vs 'In Transit' (admin/delivery/DeliveryPage:70, delivery-portal/ActiveDeliveryPage:49, mobile-delivery StatusStepIndicator)"
    related_surfaces: ["mc-timeline-steps", "md-core-028", "md-core-038", "dp-ux-active-status-in-transit", "ap-delivery-statuses"]
    recommendation: "Customer sees `On the way` (sentence case, per STYLE-GUIDE). Chef sees `Delivering`. Driver sees `In transit`. Document this 3-persona map in shared status taxonomy file. Today they're independently invented per file."
    depends_on: null

  - finding_id: BV-003
    surface_id: web-ux-orders-status-labels
    lens: brand-voice
    severity: P0
    issue: "`ready` enum labelled `Ready` in 5 places and `Ready for Pickup` in 3 places — customer's order tracker says different word per screen."
    evidence_excerpt: "label: 'Ready' (chef/DashboardPage:341, admin/OrdersPage:21, customer/OrdersPage:27, web Badge:132, vendor Badge:139, vendor LiveOrders:33, customer OrderDetailPage stepper:404) vs 'Ready for Pickup' (chef/OrdersPage:33, customer/OrderDetailPage:33 status map, mobile-customer order/[id]:33)"
    related_surfaces: ["mc-order-detail-status-labels", "mc-order-card-status", "vp-ux-dashboard-recent-orders"]
    recommendation: "Per STYLE-GUIDE pickup rules: `Pickup` is the noun. Use `Ready for pickup` (sentence case, one word `Pickup`) on customer-facing surfaces; use bare `Ready` on chef/vendor operational surfaces. Fix capitalization globally."
    depends_on: BV-007

  - finding_id: BV-004
    surface_id: mc-order-detail-status-labels
    lens: brand-voice
    severity: P0
    issue: "Mobile customer shows `Ready for Pickup` on order detail but `Ready` on OrderCard and timeline — same screen-cluster contradicts itself."
    evidence_excerpt: "mc-order-detail-status-labels: 'Ready for Pickup' / mc-order-card-status: 'Ready' / mc-timeline-steps does not include `Ready` at all (only Confirmed/Preparing/On the Way/Delivered) — flagged in inventory notes"
    related_surfaces: ["mc-order-card-status", "mc-timeline-steps", "web-ux-orders-status-labels"]
    recommendation: "Unify mobile-customer status map with web customer map; both should be sourced from `shared/types/index.ts` or equivalent — not redefined per file."
    depends_on: BV-003

  - finding_id: BV-005
    surface_id: ap-orders-filter-status
    lens: brand-voice
    severity: P1
    issue: "Admin orders filter uses `Delivering` while admin delivery page uses `In Transit` for the same status — admin operator sees two words for the same thing in different tabs."
    evidence_excerpt: "ap-orders-filter-status: 'All Status; Pending; Accepted; Preparing; Ready; Delivering; Delivered; Cancelled; Refunded' vs ap-delivery-statuses: 'Pending; Assigned; At Pickup; Picked Up; In Transit; At Dropoff; Delivered; Failed; Returned; Cancelled'"
    related_surfaces: ["ap-delivery-statuses", "web-ux-orders-status-labels"]
    recommendation: "Admin needs one merged status taxonomy. Orders and deliveries are two views of the same lifecycle — labels must match."
    depends_on: BV-002

  - finding_id: BV-006
    surface_id: ap-orders-filter-status
    lens: brand-voice
    severity: P1
    issue: "Filter labelled `All Status` (singular) in admin Orders and Chefs pages; `All Statuses` (plural) in admin Delivery, Providers, Approvals, Staff. Grammatical drift across one app."
    evidence_excerpt: "admin/orders/OrdersPage:85 `All Status` and admin/chefs/ChefsPage:141 `All Status` vs admin/delivery/DeliveryPage:181, admin/delivery/ProvidersPage:200, admin/approvals/ApprovalsPage:208, admin/staff/StaffPage:415 — all `All Statuses`"
    related_surfaces: ["ap-chefs-filter-status", "ap-delivery-statuses", "ap-approvals-statuses"]
    recommendation: "Normalize to `All statuses` (sentence case, plural)."
    depends_on: null

  # ============================================================================
  # SECTION B — PICKUP / PICK-UP / DROPOFF / DROP-OFF DRIFT (P0)
  # STYLE-GUIDE rule: `Pickup` (noun, one word) / `Pick up` (verb).
  # Currently 5 different spellings across the platform.
  # ============================================================================

  - finding_id: BV-007
    surface_id: dp-ux-delivery-pickup-label
    lens: brand-voice
    severity: P0
    issue: "`PICKUP` in ALL CAPS as label on delivery portal Available Deliveries page violates STYLE-GUIDE rule 1 (Confident, not loud) and `Sentence case for buttons and labels; Title Case banned`."
    evidence_excerpt: "apps/delivery-portal/src/features/deliveries/pages/AvailableDeliveriesPage.tsx:63 — `<p className=\"text-xs text-success font-medium\">PICKUP</p>` and line 70 — `DROPOFF`"
    related_surfaces: ["dp-ux-delivery-dropoff-label", "md-core-078", "md-core-079"]
    recommendation: "Replace `PICKUP` / `DROPOFF` with `Pickup` / `Dropoff` (sentence case). The visual weight should come from typography, not capslock."
    depends_on: null

  - finding_id: BV-008
    surface_id: md-core-079
    lens: brand-voice
    severity: P0
    issue: "Mobile-delivery uses `Drop-off` (hyphenated) on delivery detail screen but `Dropoff` (no hyphen) on active delivery screen — two spellings in one app."
    evidence_excerpt: "mobile-delivery/app/delivery/[id].tsx:79 `Drop-off` vs mobile-delivery/app/(tabs)/active.tsx line 145 `Dropoff Location`; inventory note flags drift explicitly"
    related_surfaces: ["md-core-041", "md-core-029", "md-core-035"]
    recommendation: "Pick one: `Dropoff` (matches Pickup, no hyphen) recommended. Update every spelling — currently `Drop-off`, `Dropoff`, `DROPOFF`, `drop-off` all appear in the codebase."
    depends_on: null

  - finding_id: BV-009
    surface_id: ap-notifsettings-categories
    lens: brand-voice
    severity: P1
    issue: "Admin notification copy uses `pickup and drop-off notifications` — hyphenated `drop-off` collides with `Dropoff` elsewhere; sentence does not match driver/delivery taxonomy."
    evidence_excerpt: "apps/admin-portal/src/features/settings/pages/NotificationSettingsPage.tsx:33 `Driver assigned, pickup and drop-off notifications.`"
    related_surfaces: ["dp-ux-delivery-dropoff-label", "md-core-079"]
    recommendation: "Use `pickup and dropoff notifications` for consistency."
    depends_on: BV-008

  - finding_id: BV-010
    surface_id: ap-delivery-statuses
    lens: brand-voice
    severity: P1
    issue: "Admin delivery statuses use Title Case (`At Pickup`, `At Dropoff`, `Picked Up`, `In Transit`) — violates STYLE-GUIDE `sentence case` rule for labels."
    evidence_excerpt: "ap-delivery-statuses (line 253): `Pending; Assigned; At Pickup; Picked Up; In Transit; At Dropoff; Delivered; Failed; Returned; Cancelled`"
    related_surfaces: ["dp-ux-active-status-in-transit", "md-core-038"]
    recommendation: "Sentence case: `Pending`, `Assigned`, `At pickup`, `Picked up`, `In transit`, `At dropoff`, `Delivered`, `Failed`, `Returned`, `Cancelled`."
    depends_on: null

  - finding_id: BV-011
    surface_id: dp-ux-active-mark-as
    lens: brand-voice
    severity: P1
    issue: "Delivery-portal action label `Mark as {Picked Up/In Transit/Delivered}` uses Title Case for sub-status. Inconsistent with sentence-case driver-app pattern (`Mark as delivered`)."
    evidence_excerpt: "apps/delivery-portal/src/features/deliveries/pages/ActiveDeliveryPage.tsx:46-52 — labels `Picked Up`, `In Transit`, `Delivered` plugged into `Mark as {label}` button"
    related_surfaces: ["md-core-036", "md-core-038"]
    recommendation: "Sentence case sub-labels: `Mark as picked up` / `Mark as in transit` / `Mark as delivered`."
    depends_on: BV-010

  # ============================================================================
  # SECTION C — CART vs BAG / "YOUR CART" CONSISTENCY (P2)
  # ============================================================================

  - finding_id: BV-012
    surface_id: web-ux-cart-heading
    lens: brand-voice
    severity: P2
    issue: "Web uses `Your Cart` (Title Case) and mobile-customer uses `Your Cart` (Title Case) — consistent across apps, but Title Case violates sentence-case rule."
    evidence_excerpt: "web CartPage.tsx:76 `Your Cart` (h1); mobile-customer CartSheet.tsx `Your Cart` heading"
    related_surfaces: ["mc-cartsheet-title", "web-ux-cart-heading"]
    recommendation: "Sentence case: `Your cart` on both surfaces. Brand consistency on `Cart` (no `Bag` drift) is good — preserve that, just fix capitalization."
    depends_on: null

  # ============================================================================
  # SECTION D — PLACE ORDER / SUBMIT / CONFIRM CTA DRIFT (P1)
  # ============================================================================

  - finding_id: BV-013
    surface_id: web-ux-checkout-summary
    lens: brand-voice
    severity: P1
    issue: "Web checkout CTA is `Place Order` (Title Case), mobile-customer is `Place Order · ₹{total}`, both Title Case — STYLE-GUIDE requires sentence case (`Place order`)."
    evidence_excerpt: "web CheckoutPage.tsx:681 `Place Order - {price}`; mobile-customer checkout.tsx:539 `Place Order · ₹{total}`"
    related_surfaces: ["mc-checkout-place-order", "web-ux-checkout-summary"]
    recommendation: "Use `Place order` everywhere (sentence case)."
    depends_on: null

  - finding_id: BV-014
    surface_id: web-ux-cart-checkout-cta
    lens: brand-voice
    severity: P2
    issue: "Web cart CTA `Sign in to Checkout` mixes `Sign in` (correct per STYLE-GUIDE) with `Checkout` (capital C — should be lowercase). Same line, two capitalization schemes."
    evidence_excerpt: "web CartPage.tsx:264 `Sign in to Checkout / Proceed to Checkout`"
    related_surfaces: ["mc-cartsheet-checkout-cta"]
    recommendation: "`Sign in to checkout` and `Proceed to checkout` — sentence case throughout."
    depends_on: null

  - finding_id: BV-015
    surface_id: web-ux-checkout-heading
    lens: brand-voice
    severity: P3
    issue: "Web `Checkout` page H1 is bare `Checkout`. Mobile-customer is `Checkout`. Consistent across apps — but should be `Checkout` (single word) and remain so. No drift; documenting baseline."
    evidence_excerpt: "web CheckoutPage.tsx:292 `Checkout`; mobile-customer checkout.tsx `Checkout`"
    related_surfaces: ["mc-checkout-header"]
    recommendation: "No change. Keep `Checkout` as a single-word H1 across both surfaces."
    depends_on: null

  # ============================================================================
  # SECTION E — CANCEL ORDER / CANCEL DELIVERY DRIFT (P1)
  # ============================================================================

  - finding_id: BV-016
    surface_id: web-ux-orderdetail-cancel
    lens: brand-voice
    severity: P1
    issue: "Customer-facing destructive action uses `Cancel Order` (Title Case) on web; mobile uses no equivalent — only `Track Order` is on mobile order detail. Cancellation lives on web only as Title Case."
    evidence_excerpt: "web OrderDetailPage.tsx:345 `Cancel Order` (button); :360 `Cancel Order` (modal h3); :389 `Cancel Order` (confirm button); mobile-customer has no cancel CTA on order detail"
    related_surfaces: ["dp-ux-active-cancel-btn", "mc-order-detail-track-cta"]
    recommendation: "`Cancel order` (sentence case). Also: customer can cancel on web but not mobile — flag for cross-platform feature parity (likely BA lens, not BV)."
    depends_on: null

  - finding_id: BV-017
    surface_id: dp-ux-active-cancel-btn
    lens: brand-voice
    severity: P1
    issue: "Delivery-portal driver-facing `Cancel Delivery` is Title Case and uses `Delivery` not `Order` — drift from customer `Cancel Order`. Different persona, different noun — but capitalization wrong for both."
    evidence_excerpt: "apps/delivery-portal/src/features/deliveries/pages/ActiveDeliveryPage.tsx:168 `Cancel Delivery`"
    related_surfaces: ["web-ux-orderdetail-cancel"]
    recommendation: "Driver cancels delivery (not order — different mental model is OK). Use sentence case: `Cancel delivery`. The noun split is intentional."
    depends_on: BV-016

  - finding_id: BV-018
    surface_id: dp-ux-active-cancel-prompt
    lens: brand-voice
    severity: P1
    issue: "Cancellation reason prompt uses browser `prompt()` modal — escapes styled UI, breaks brand voice. Inventory flagged as 'S' (no UI polish)."
    evidence_excerpt: "apps/delivery-portal/src/features/deliveries/pages/ActiveDeliveryPage.tsx:160 — uses native `prompt('Reason for cancellation?')`"
    related_surfaces: ["dp-ux-active-cancel-btn"]
    recommendation: "Replace with a Dialog component; copy: `Why are you cancelling?` (informal, calmer than `Reason for cancellation?`)."
    depends_on: null

  # ============================================================================
  # SECTION F — DRIVER ACTION VERBS (Accept/Reject/Decline/Available/Picked Up/Delivered)
  # ============================================================================

  - finding_id: BV-019
    surface_id: md-core-022
    lens: brand-voice
    severity: P1
    issue: "Driver `Accept Delivery` (Title Case) in mobile and `Accept Delivery` (Title Case) in delivery-portal — consistent verb, wrong case. Driver app should be sentence case per STYLE-GUIDE."
    evidence_excerpt: "mobile-delivery components/driver/DeliveryCard.tsx `Accept Delivery`; delivery-portal AvailableDeliveriesPage:95 `Accept Delivery`"
    related_surfaces: ["dp-ux-accept-delivery-btn", "md-core-022"]
    recommendation: "`Accept delivery` (sentence case)."
    depends_on: null

  - finding_id: BV-020
    surface_id: md-core-018
    lens: brand-voice
    severity: P2
    issue: "Mobile-delivery `Go Online to Accept Deliveries` (Title Case across 5 words) is a button — way over the ≤3-word button rule. Driver-facing rule is ≤4 telegraphic words."
    evidence_excerpt: "mobile-delivery/app/(tabs)/available.tsx `Go Online to Accept Deliveries` (md-core-018)"
    related_surfaces: ["dp-ux-dashboard-go-online-btn", "md-core-006"]
    recommendation: "Shorten to `Go online` (driver toggle pattern matches delivery-portal `dp-ux-dashboard-go-online-btn`)."
    depends_on: null

  - finding_id: BV-021
    surface_id: md-core-032
    lens: brand-voice
    severity: P2
    issue: "Slide-to-confirm action labels mix verb tenses: `Arrived at Kitchen` (past), `Picked Up Order` (past), `Start Delivery` (imperative), `Arrived at Dropoff` (past), `Mark as Delivered` (imperative). Inconsistent tense per step makes driver pause."
    evidence_excerpt: "mobile-delivery active.tsx:31-36 — `assigned: Arrived at Kitchen`, `at_pickup: Picked Up Order`, `picked_up: Start Delivery`, `in_transit: Arrived at Dropoff`, `at_dropoff: Mark as Delivered`"
    related_surfaces: ["md-core-033", "md-core-034", "md-core-035", "md-core-036"]
    recommendation: "Use imperative throughout (driver is the actor): `I'm at the kitchen` (or `Arrived`), `Confirm pickup`, `Start delivery`, `Arrived at dropoff`, `Mark delivered`. Slide-to-confirm is irreversible; verbs must match the user's NEXT action, not pass tense for one step then imperative for the next."
    depends_on: null

  - finding_id: BV-022
    surface_id: dp-ux-nav-bottom-items
    lens: brand-voice
    severity: P1
    issue: "Driver bottom-nav says `Home` but sidebar says `Dashboard` for the same destination. Same app, two different words for the same screen."
    evidence_excerpt: "delivery-portal DeliveryBottomNav.tsx:5-11 `Home / Active / Available / Earnings / Profile` vs DeliveryLayout.tsx:39-47 sidebar `Dashboard / Active Delivery / Available / History / Earnings / Profile / Settings`. Inventory notes explicitly flag this drift."
    related_surfaces: ["dp-ux-nav-partner-items", "md-core-001", "md-core-005"]
    recommendation: "Pick one: `Dashboard` matches the H1 of that screen, so use `Dashboard` everywhere. `Home` on bottom-nav is fine ONLY if there's no Dashboard concept — but there is."
    depends_on: null

  - finding_id: BV-023
    surface_id: dp-ux-nav-partner-items
    lens: brand-voice
    severity: P1
    issue: "Sidebar says `Active Delivery` but bottom-nav says `Active`. Same app, same screen, two labels."
    evidence_excerpt: "delivery-portal sidebar `Active Delivery` vs bottom-nav `Active`. Inventory note C-flagged"
    related_surfaces: ["dp-ux-nav-bottom-items"]
    recommendation: "Bottom-nav has space constraints — `Active` is fine. But ensure aria-label / semantic label is `Active delivery` for screen readers."
    depends_on: BV-022

  # ============================================================================
  # SECTION G — CHEF ORDER VERBS (Accept/Reject/Prepare/Ready)
  # ============================================================================

  - finding_id: BV-024
    surface_id: vp-ux-dashboard-pending-card
    lens: brand-voice
    severity: P2
    issue: "Vendor portal chef-facing pending orders show `Accept / Reject` — consistent with mobile-vendor. STYLE-GUIDE says chef tone is operational; `Reject` reads cold for a customer order. Consider `Decline`?"
    evidence_excerpt: "vendor-portal DashboardPage Recent Orders + LiveOrders use `Accept / Reject`; mobile-vendor OrderCard.tsx `Accept / Reject` (mv-ordercard-accept, mv-ordercard-reject); also iOS notification action buttons same labels"
    related_surfaces: ["mv-ordercard-accept", "mv-ordercard-reject", "vp-ux-dashboard-pending-card"]
    recommendation: "Keep `Accept`; consider `Decline` instead of `Reject` (less adversarial for the chef-customer relationship). Document choice in STYLE-GUIDE under Order verbs."
    depends_on: null

  - finding_id: BV-025
    surface_id: vp-ux-dashboard-pending-card
    lens: brand-voice
    severity: P1
    issue: "Chef-side action chain mixes `Accept`, `Start Preparing`, `Mark Ready`. STYLE-GUIDE example shows `Mark ready` (sentence case). Today: Title Case on all three."
    evidence_excerpt: "web chef/OrdersPage:32 `nextAction: 'Start Preparing'` and `nextAction: 'Mark Ready'`; vendor-portal LiveOrdersPage:41 `'Mark Ready'`"
    related_surfaces: ["vp-ux-dashboard-pending-card", "web-ux-chef-orders-heading"]
    recommendation: "Use sentence case: `Start preparing`, `Mark ready`. Three chef actions form one sequence; consistent style is critical."
    depends_on: null

  # ============================================================================
  # SECTION H — ADMIN ACTION VERBS (Approve/Reject/Suspend/Activate)
  # ============================================================================

  - finding_id: BV-026
    surface_id: dp-ux-partner-verify-approve
    lens: brand-voice
    severity: P1
    issue: "Admin/partner actions are mixed Title Case: `Approve` (1 word OK), `Reject` (1 word OK), but `Reactivate Partner`, `Suspend Partner`, `Verify Partner` — all Title Case violate sentence-case rule."
    evidence_excerpt: "delivery-portal PartnerDetailPage:264 `Approve`, :272 `Reject`, :378 `Reactivate Partner`, :387 `Suspend Partner`, :397 `Verify Partner`"
    related_surfaces: ["dp-ux-partner-verify-reject", "dp-ux-partner-reactivate", "dp-ux-partner-suspend", "dp-ux-partner-verify"]
    recommendation: "`Reactivate partner`, `Suspend partner`, `Verify partner`. Single-word verbs `Approve` and `Reject` are fine."
    depends_on: null

  - finding_id: BV-027
    surface_id: ap-approvals-statuses
    lens: brand-voice
    severity: P2
    issue: "Approval status filter mixes `Info Requested` (Title Case, 2 words) with single-word others. Should be sentence case across all."
    evidence_excerpt: "ap-approvals-statuses: `All Statuses; Pending; Approved; Rejected; Info Requested; Cancelled`"
    related_surfaces: ["ap-approvals-table-columns"]
    recommendation: "`Info requested` — sentence case, consistent with other multi-word statuses."
    depends_on: null

  # ============================================================================
  # SECTION I — BRAND IDENTITY DRIFT: Fe3dr vs HomeChef vs Home Chef (P0)
  # ============================================================================

  - finding_id: BV-028
    surface_id: web-ux-layout-nav
    lens: brand-voice
    severity: P0
    issue: "Brand name appears as `Fe3dr`, `HomeChef`, `Home Chef`, and `home-chef` in user-facing strings. No single canonical form."
    evidence_excerpt: "web Register/LoginPage `Fe3dr`; web OttoChat:157,163 `HomeChef support`; web ProfilePage:1288 `HomeChef 2FA Backup Codes`; web BrowseChefsPage:140 `Explore Home Chefs`; web HomePage:88,155 `500+ Home Chefs`; web CheckoutPage:191 `Home Chef` fallback; admin/vendor/delivery LoginPages `Fe3dr`; admin LoginPage:146 `Fe3dr platform`, :459 `Fe3dr Administration Portal`; admin Provider pages `Fe3dr statuses`, `Fe3dr Status`; vendor LoginPage:155 `Fe3dr customer app`, :323 `Fe3dr's Terms`, RegisterPage:99 `Join thousands of home chefs earning with Fe3dr`; brand.ts says `name: 'Fe3dr'`"
    related_surfaces: ["ap-layout-brand-title", "ap-layout-topbar-title", "dp-ux-logo-text"]
    recommendation: "P0 platform identity: pick one brand name and codify in `shared/constants/brand.ts`. Recommended: `Fe3dr` is the platform brand (per brand.ts), `home chef` is the role noun (lowercase, never as brand). All `HomeChef support`, `HomeChef 2FA`, `Home Chef` (as brand) references must be replaced."
    depends_on: null

  - finding_id: BV-029
    surface_id: web-ux-layout-nav
    lens: brand-voice
    severity: P0
    issue: "Customer-facing landing copy says `Home Chef` capitalized as if brand: `Explore Home Chefs`, `500+ Home Chefs Near You`. Should be `home chefs` (lowercase noun for the role) per STYLE-GUIDE."
    evidence_excerpt: "web BrowseChefsPage.tsx:140 `Explore Home Chefs`; HomePage.tsx:88 `500+ Home Chefs Near You`, :155 label `500+ Home Chefs`, HeroSection.tsx:123 label `Home Chefs`"
    related_surfaces: ["web-ux-browse-search-input"]
    recommendation: "STYLE-GUIDE entry: `Home chef` ✅ (sentence case for the role). Customer-facing: `Explore home chefs`, `500+ home chefs near you`."
    depends_on: BV-028

  # ============================================================================
  # SECTION J — AI-SLOP / UNVERIFIED SOCIAL PROOF (P0)
  # ============================================================================

  - finding_id: BV-030
    surface_id: web-ux-layout-nav
    lens: brand-voice
    severity: P0
    issue: "Hardcoded `500+ Home Chefs Near You` and `500+ Home Chefs` repeated in 3 customer-facing places. If platform has 500+ chefs this should be dynamic; otherwise it's false advertising. AI-slop trust metric."
    evidence_excerpt: "web HomePage.tsx:88 `500+ Home Chefs Near You`; :155 `{ icon: ChefHat, label: '500+ Home Chefs' }`; HeroSection.tsx:123 `{ value: '500+', label: 'Home Chefs' }`; RegisterPage.tsx:10 `Access to 500+ home chefs`"
    related_surfaces: []
    recommendation: "Either (a) wire to live API count, OR (b) remove the badge entirely. Hardcoded `500+` violates Rule 5 (Restraint over urgency) AND Rule 1 (Confident, not loud)."
    depends_on: null

  - finding_id: BV-031
    surface_id: web-ux-layout-nav
    lens: brand-voice
    severity: P0
    issue: "`Join thousands of happy customers enjoying homemade food` (web home) and `Join thousands of home chefs earning with Fe3dr` (vendor register) — unverified social proof; faux-artisanal `homemade` term."
    evidence_excerpt: "web HomePage.tsx:382 `Join thousands of happy customers enjoying homemade food`; vendor-portal RegisterPage.tsx:99 `Join thousands of home chefs earning with Fe3dr. Your kitchen, your recipes, your rules.`"
    related_surfaces: []
    recommendation: "Remove `Join thousands` (unverified). Replace `homemade food` with `home-cooked food` (rules out `homemade` as faux-artisanal trigger per STYLE-GUIDE). Rewrite as factual: `Real home cooks. Real kitchens. Order tonight's dinner.`"
    depends_on: null

  - finding_id: BV-032
    surface_id: web-ux-layout-nav
    lens: brand-voice
    severity: P1
    issue: "Web LoginPage testimonial: `Fe3dr has changed how I eat. Finally, real homemade food that…` Likely fake/placeholder testimonial; `homemade` is a banned brand-drift term."
    evidence_excerpt: "apps/web/src/features/auth/pages/LoginPage.tsx:277 `Fe3dr has changed how I eat. Finally, real homemade food that…`"
    related_surfaces: ["web-ux-layout-nav"]
    recommendation: "Remove the testimonial OR verify it's a real customer with attribution. Replace `homemade` → `home-cooked`."
    depends_on: BV-031

  - finding_id: BV-033
    surface_id: web-ux-layout-nav
    lens: brand-voice
    severity: P1
    issue: "Web home `The Fe3dr Difference` is generic SaaS-marketing pattern. STYLE-GUIDE anti-reference: avoid `Generic SaaS dashboard / Hero metrics` language."
    evidence_excerpt: "apps/web/src/features/customer/pages/HomePage.tsx:379 `The Fe3dr Difference`"
    related_surfaces: []
    recommendation: "Replace with editorial-tone heading: `Why home cooks, not restaurants` or simply remove the section."
    depends_on: null

  - finding_id: BV-034
    surface_id: vp-ux-settings-payout-section
    lens: brand-voice
    severity: P1
    issue: "Vendor-portal kitchen setup ships `Payout integration coming soon. These details are saved locally for now.` — `coming soon` feature in production violates Rule 1 (Confident, not loud)."
    evidence_excerpt: "apps/vendor-portal/src/features/profile/pages/KitchenSetupPage.tsx:507 `Payout integration coming soon. These details are saved locally for now.`"
    related_surfaces: ["vp-ux-settings-payout-section"]
    recommendation: "Either ship the integration or remove the section. `Saved locally` is also worrying — chef may think they're set up when payouts will not happen."
    depends_on: null

  - finding_id: BV-035
    surface_id: ap-analytics-cards
    lens: brand-voice
    severity: P1
    issue: "Admin analytics ships with `Chart coming soon` placeholder. Inventory notes confirm `placeholders` for entire summary card set."
    evidence_excerpt: "apps/admin-portal/src/features/analytics/pages/AnalyticsPage.tsx:69 `Chart coming soon`; ap-analytics-cards inventory note: `summary card labels (placeholders)`"
    related_surfaces: ["ap-analytics-charts"]
    recommendation: "Hide placeholder charts until live. Empty-state copy if hiding isn't possible: `No data yet for this period.`"
    depends_on: null

  # ============================================================================
  # SECTION K — SIGN OUT / LOG OUT / LOGOUT DRIFT (P1)
  # STYLE-GUIDE: `Sign out` ✅ / `Log out` ❌ / `Logout` ❌
  # ============================================================================

  - finding_id: BV-036
    surface_id: web-ux-layout-nav
    lens: brand-voice
    severity: P1
    issue: "STYLE-GUIDE mandates `Sign out`. Today: `Logout` (5 surfaces), `Log out` (2), `Log Out` (2), `Sign Out` (1), `Sign out` (1). All seven apps disagree."
    evidence_excerpt: "web MainLayout:261 `Logout`; web ChefLayout:131 `Logout`; web AdminLayout:94 `aria-label='Log out'`; web ProfilePage:127 `Log Out`; web FavoritesPage:43 `Log In`; admin AdminLayout:164 `Logout`; delivery-portal SettingsPage:49 `Sign Out` (Title), :50 `Sign out`; mobile-customer profile:115,288 `Log Out`; mobile-vendor more.tsx:62,119 `Logout`; pending.tsx:56 `Logout`; mobile-delivery more.tsx:30,113 `Logout`; pending.tsx:47,77 `Logout`"
    related_surfaces: ["mc-profile-logout-button", "mv-more-logout-row", "mv-onb-pending-logout-btn", "dp-ux-settings-signout", "ap-layout-nav-settings"]
    recommendation: "Global find/replace to `Sign out` (sentence case, two words). Every surface. Code variables can stay `logout()` — only user-facing strings change."
    depends_on: null

  - finding_id: BV-037
    surface_id: web-ux-layout-nav
    lens: brand-voice
    severity: P1
    issue: "`Login` and `Log In` appear as user-facing labels in web nav and FavoritesPage. STYLE-GUIDE: `Sign in`."
    evidence_excerpt: "web MainLayout:272 `Login` link; FavoritesPage:43 `Log In` link"
    related_surfaces: ["web-ux-layout-nav"]
    recommendation: "Replace with `Sign in`."
    depends_on: BV-036

  # ============================================================================
  # SECTION L — TITLE CASE EPIDEMIC ACROSS HEADINGS (P2/P3)
  # ============================================================================

  - finding_id: BV-038
    surface_id: web-ux-profile-heading
    lens: brand-voice
    severity: P2
    issue: "Web customer profile heading `Account Settings` (Title Case). STYLE-GUIDE requires sentence case for labels and buttons; headings should follow same rule."
    evidence_excerpt: "web customer ProfilePage:68 `Account Settings`"
    related_surfaces: ["ap-settings-title", "mv-settings-heading", "md-core-055"]
    recommendation: "`Account settings`. Apply globally to `Personal Information`/`Personal Info`, `Order Summary`, `Order Items`, `Delivery Address`, etc."
    depends_on: null

  - finding_id: BV-039
    surface_id: web-ux-orderdetail-sections
    lens: brand-voice
    severity: P2
    issue: "Web customer order detail headings all Title Case: `Order Items`, `Delivery Address`, `Payment Summary`, `Special Instructions`, `Leave a Review`. Mobile mirrors with `Items`, `Delivery Address`, `Price Breakdown`."
    evidence_excerpt: "web OrderDetailPage:213 `Order Items`; CartPage:185, CheckoutPage:604 `Order Summary`; mobile-customer order/[id] `Items / Delivery Address / Price Breakdown`"
    related_surfaces: ["mc-order-detail-sections", "web-ux-cart-order-summary", "web-ux-checkout-summary"]
    recommendation: "Sentence case everywhere: `Order items`, `Delivery address`, `Order summary`, `Payment summary`, `Special instructions`, `Leave a review`."
    depends_on: BV-038

  - finding_id: BV-040
    surface_id: mc-checkout-totals
    lens: brand-voice
    severity: P3
    issue: "Mobile-customer checkout totals mix capitalization: `Subtotal / Delivery fee / Free / Total` (checkout.tsx) vs order-detail `Subtotal / Delivery Fee / Total`. Even within one user flow."
    evidence_excerpt: "mobile-customer checkout.tsx:480 `Delivery fee` (lowercase f) vs order/[id]/index.tsx price-rows `Delivery Fee` (capital F). Inventory note flags drift explicitly."
    related_surfaces: ["mc-order-detail-price-rows", "web-ux-cart-order-summary"]
    recommendation: "Standardize to `Delivery fee` (sentence case, matches STYLE-GUIDE)."
    depends_on: null

  - finding_id: BV-041
    surface_id: web-ux-orders-status-labels
    lens: brand-voice
    severity: P2
    issue: "Customer status labels all Title Case: `Pending / Accepted / Preparing / Ready / Picked Up / On the Way / Delivered / Cancelled / Refunded`. Status pills should be sentence case."
    evidence_excerpt: "web customer/OrdersPage:23-33 — all Title Case"
    related_surfaces: ["mc-order-card-status", "mc-order-detail-status-labels"]
    recommendation: "Sentence case throughout: `Pending`, `Accepted`, `Preparing`, `Ready`, `Picked up`, `On the way`, `Delivered`, `Cancelled`, `Refunded`."
    depends_on: BV-002

  # ============================================================================
  # SECTION M — CURRENCY / LOCALE DRIFT (P0)
  # ============================================================================

  - finding_id: BV-042
    surface_id: vp-ux-settings-order-acceptance
    lens: brand-voice
    severity: P0
    issue: "Vendor portal auto-accept threshold label shows `($)` but platform is INR/₹ everywhere else. Currency symbol mismatch — chef will misconfigure their threshold."
    evidence_excerpt: "apps/vendor-portal/src/features/settings/pages/SettingsPage.tsx:155 `Auto-accept threshold ($)`. Inventory note explicitly flags: `currency mismatch: shows $ but data is INR`"
    related_surfaces: ["vp-ux-settings-payout-section"]
    recommendation: "Replace `($)` with `(₹)` to match STYLE-GUIDE `₹120` rule. Verify no other dollar-sign artifacts."
    depends_on: null

  - finding_id: BV-043
    surface_id: web-ux-checkout-payment-section
    lens: brand-voice
    severity: P3
    issue: "Web payment notice says `Pay securely via UPI, cards, net banking, or wallets`. India-specific payment method `UPI` is correct but `net banking` should be `netbanking` per Razorpay convention. Mobile checkout uses different copy entirely."
    evidence_excerpt: "web CheckoutPage:533-554 `Powered by Razorpay / Pay securely via UPI, cards, net banking, or wallets`. Mobile-customer has no equivalent block (just opens Razorpay sheet)."
    related_surfaces: []
    recommendation: "Match mobile + web payment provider notice OR remove it (Razorpay sheet itself says all methods)."
    depends_on: null

  # ============================================================================
  # SECTION N — PERSONA-TONE LEAKS (P2)
  # ============================================================================

  - finding_id: BV-044
    surface_id: vp-ux-profile-business-info
    lens: brand-voice
    severity: P2
    issue: "Vendor-portal `Tell customers about your kitchen, cooking style, and what makes your food special...` reads as marketing-coach copy — too aspirational for operational chef tone."
    evidence_excerpt: "vendor-portal features/profile/pages/ProfilePage.tsx — vp-ux-profile-business-info: `Tell customers about your kitchen, cooking style, and what makes your food special...`"
    related_surfaces: ["web-ux-chef-profile"]
    recommendation: "Trim to: `Describe your kitchen and the food you cook.` Per STYLE-GUIDE chef tone: crisp 5-12 words, operational mood. `What makes your food special` is a marketing-coach hint."
    depends_on: null

  - finding_id: BV-045
    surface_id: vp-ux-kitchen-photos
    lens: brand-voice
    severity: P2
    issue: "Vendor-portal: `Add photos of your kitchen to build trust with customers ({n}/5)` — `build trust with customers` is marketing-y for chef tone."
    evidence_excerpt: "vendor-portal KitchenSetupPage `Kitchen Photos / Add photos of your kitchen to build trust with customers ({n}/5)`"
    related_surfaces: []
    recommendation: "Tighten: `Show customers your kitchen. ({n}/5)`. Drop `build trust`."
    depends_on: null

  - finding_id: BV-046
    surface_id: vp-ux-dashboard-pending-cta
    lens: brand-voice
    severity: P2
    issue: "Vendor-portal `Waiting for you to accept` (vp-ux-dashboard-pending-cta) is conversational/customer-style. Chef expects telegraphic; this reads like a customer waiting line."
    evidence_excerpt: "vendor-portal DashboardPage `{n} orders / Waiting for you to accept`"
    related_surfaces: []
    recommendation: "Crisper chef-tone: `{n} orders awaiting acceptance.` or `{n} new orders.` (matches `All caught up / New orders will appear here.` empty-state tone)."
    depends_on: null

  - finding_id: BV-047
    surface_id: vp-ux-dashboard-quick-actions
    lens: brand-voice
    severity: P3
    issue: "Quick Actions tiles use comma-separated title-action format: `Add Menu Item, Create a new dish listing / View Orders, Manage live orders / Update Profile, Edit kitchen details / View Earnings, Track your revenue`. Title Case + verbose secondary line."
    evidence_excerpt: "vendor-portal DashboardPage quick-actions block"
    related_surfaces: ["ap-dashboard-shortcuts-items", "web-ux-chef-dashboard-sections"]
    recommendation: "Sentence case + drop verbose sub-line where redundant: `Add menu item`, `View orders`, `Update profile`, `View earnings`. Subtitles only when they add information not in the title."
    depends_on: null

  - finding_id: BV-048
    surface_id: mv-dash-subhead
    lens: brand-voice
    severity: P3
    issue: "Mobile-vendor dashboard subhead: `Here's your kitchen overview` vs vendor-portal: `Here's your kitchen today.` — drift across same persona/feature."
    evidence_excerpt: "mobile-vendor/app/(tabs)/index.tsx `Here's your kitchen overview` vs vendor-portal DashboardPage `Here's your kitchen today.`"
    related_surfaces: ["vp-ux-dashboard-title"]
    recommendation: "Pick one — `Here's your kitchen today.` matches the natural conversational rhythm; or drop entirely (the H1 `Dashboard` is enough)."
    depends_on: null

  - finding_id: BV-049
    surface_id: mv-dash-stats-today-earnings
    lens: brand-voice
    severity: P3
    issue: "Mobile-vendor stat: `Today's Earnings`; vendor-portal: `Today's revenue` (vp-ux-dashboard-revenue-lead). Earnings vs revenue is a mental-model fork."
    evidence_excerpt: "mobile-vendor (mv-dash-stats-today-earnings) `Today's Earnings`; vendor-portal (vp-ux-dashboard-revenue-lead) `Today's revenue`"
    related_surfaces: ["dp-ux-dashboard-today-earnings-label", "mv-analytics-total-revenue", "mv-earnings-total"]
    recommendation: "Standardize on `Today's earnings` (sentence case, chef-facing); reserve `Revenue` for admin/analytics."
    depends_on: null

  # ============================================================================
  # SECTION O — PUNCTUATION / DECORATION / SAFETY (P2/P3)
  # ============================================================================

  - finding_id: BV-050
    surface_id: mc-profile-more-rows
    lens: brand-voice
    severity: P2
    issue: "Mobile-customer profile `More` section uses emoji icons (📱/🍽️) for nav rows. STYLE-GUIDE: emoji acceptable in customer-facing moments, but used as a decorative icon system this breaks Rule 1 (Confident, not loud) and the design tokens (lucide icons)."
    evidence_excerpt: "mobile-customer profile.tsx mc-profile-more-rows: `Social Feed / Catering` with emoji icons (📱/🍽️)"
    related_surfaces: []
    recommendation: "Replace emoji with lucide-react icons matching the rest of the app."
    depends_on: null

  - finding_id: BV-051
    surface_id: mv-undo-cta
    lens: brand-voice
    severity: P3
    issue: "Mobile-vendor `UNDO` snackbar action is ALL CAPS. STYLE-GUIDE bans ALL CAPS body; button microcopy may be tolerated but consistency matters — other buttons in this app are sentence case."
    evidence_excerpt: "mobile-vendor components/vendor/UndoSnackbar.tsx `UNDO`. Inventory note explicitly flags."
    related_surfaces: []
    recommendation: "`Undo` (sentence case). The snackbar still reads clearly; no need for capslock."
    depends_on: BV-007

  - finding_id: BV-052
    surface_id: md-core-047
    lens: brand-voice
    severity: P3
    issue: "Mobile-delivery delivery-complete success line: `You earned ₹{payout} for this delivery` is good factual confidence. Comparable web/delivery-portal success states lack this voice — drift on what success sounds like across surfaces."
    evidence_excerpt: "mobile-delivery active.tsx md-core-047 `You earned ₹{payout} for this delivery`. No equivalent in delivery-portal."
    related_surfaces: ["dp-ux-active-status-delivered"]
    recommendation: "Bring delivery-portal in line — show payout in success state. Voice principle holds; just missing in one surface."
    depends_on: null

  - finding_id: BV-053
    surface_id: md-core-039
    lens: brand-voice
    severity: P2
    issue: "Mobile-delivery cancelled banner: `This delivery has been cancelled. Check the Available tab for new requests.` — `Available tab` is a brittle in-text reference (label changes break copy). Also `has been cancelled` is passive."
    evidence_excerpt: "mobile-delivery active.tsx md-core-039 `This delivery has been cancelled. Check the Available tab for new requests.`"
    related_surfaces: ["md-core-002"]
    recommendation: "Active voice, less brittle: `Delivery cancelled. New requests appear in Available.` Drop the word `tab`."
    depends_on: null

  # ============================================================================
  # SECTION P — CUISINE TAXONOMY DRIFT (P2)
  # Same cuisine list duplicated across web/mobile customer + vendor onboarding
  # with different items and different orderings.
  # ============================================================================

  - finding_id: BV-054
    surface_id: web-ux-browse-filters-cuisines
    lens: brand-voice
    severity: P2
    issue: "Customer cuisine filter has 10 cuisines (web): `South Indian / North Indian / Italian / Japanese / Mexican / Thai / Chinese / Mediterranean / American / Continental`. Mobile-customer home has 7: `All, North Indian, South Indian, Chinese, Continental, Italian, Healthy`. Vendor onboarding has 8: `North Indian / South Indian / Chinese / Continental / Bakery / Snacks / Beverages / Other`. Three cuisine universes that don't match."
    evidence_excerpt: "web BrowseChefsPage.tsx:29-40 (10 items); mobile-customer index.tsx (7); mobile-vendor onboarding kitchen-details.tsx (8); mobile-customer onboarding preferences.tsx (8); profile.tsx (8 with order drift)"
    related_surfaces: ["mc-home-cuisine-filters", "mc-onb-step3-cuisines", "mc-profile-cuisines", "mv-onb-kitchen-cuisine-options"]
    recommendation: "Centralize the cuisine list in a shared constant. Either chef can offer cuisines outside the customer filter (broken filtering) or customer can request cuisines no chef offers (broken matching)."
    depends_on: null

  - finding_id: BV-055
    surface_id: mc-profile-cuisines
    lens: brand-voice
    severity: P3
    issue: "Mobile-customer cuisine order drifts between onboarding and profile screens. Inventory note flags: `order differs from onboarding step3 — flag for drift`."
    evidence_excerpt: "mc-onb-step3-cuisines: `North Indian, South Indian, Chinese, Continental, Italian, Healthy, Street Food, Desserts`; mc-profile-cuisines: `North Indian, South Indian, Chinese, Continental, Italian, Healthy, Desserts, Street Food` (Street Food and Desserts swapped)"
    related_surfaces: ["mc-onb-step3-cuisines"]
    recommendation: "Share the list and the order. Trivial fix; high consistency value."
    depends_on: BV-054

  # ============================================================================
  # SECTION Q — PREP TIME / UNIT DRIFT (P3)
  # ============================================================================

  - finding_id: BV-056
    surface_id: mv-onb-ops-preptime-options
    lens: brand-voice
    severity: P3
    issue: "Mobile-vendor onboarding uses `15min / 30min / 45min / 60min / 90min` (no space) but the same app's menu-new screen uses `{n} min` (with space). Inventory note: 'Drift vs new.tsx which uses {n} min with space.'"
    evidence_excerpt: "mobile-vendor (onboarding)/operations.tsx mv-onb-ops-preptime-options `15min/30min/45min/60min/90min` vs menu/new.tsx mv-menunew-preptime-unit `{n} min`"
    related_surfaces: ["mv-menunew-preptime-unit"]
    recommendation: "Always `{n} min` with space (matches STYLE-GUIDE `1.2 km`, `15%` percentage exception). Update onboarding."
    depends_on: null

  # ============================================================================
  # SECTION R — DRIVER PERSONA-NOUN DRIFT (P2)
  # Driver / Partner / Delivery Partner / Delivery Ops
  # ============================================================================

  - finding_id: BV-057
    surface_id: dp-ux-profile-fallback-name
    lens: brand-voice
    severity: P2
    issue: "Delivery-portal driver profile fallback name varies: `Delivery Partner` (UI shell) but `Driver` (login flow). STYLE-GUIDE: `Driver` ✅ (driver-facing) / `Delivery partner` (customer-facing only)."
    evidence_excerpt: "dp-ux-profile-fallback-name `Delivery Partner` (admin uses this); login flow fallback uses `Driver`. Inventory note: `P — varies with login \"Driver\"`"
    related_surfaces: ["dp-ux-partners-h", "dp-ux-fleet-partner-summary"]
    recommendation: "Driver-facing screens (active, available, earnings, profile when self-viewing): `Driver`. Admin/dispatcher screens (fleet overview, partner detail): `Partner` or `Delivery partner`. Document the persona split."
    depends_on: null

  - finding_id: BV-058
    surface_id: dp-ux-staff-roles-fallback
    lens: brand-voice
    severity: P3
    issue: "Delivery-portal staff role list mixes `Delivery Ops`, `Fleet Manager`, `Super Admin` (Title Case across) — these are roles users will see in role-picker UI. Sentence case required."
    evidence_excerpt: "delivery-portal StaffPage.tsx:292 `Delivery Ops / Fleet Manager / Super Admin`; ap-staff-roles `Super Admin; Admin; Fleet Manager; Delivery Ops; Support`"
    related_surfaces: ["ap-staff-roles"]
    recommendation: "Role names in dropdowns: `Super admin`, `Fleet manager`, `Delivery ops`, `Admin`, `Support`. (Hyphenated `Delivery-ops` if you want one-word role code — but UI label is sentence case.)"
    depends_on: null

  # ============================================================================
  # SECTION S — MENU MANAGEMENT NOUN DRIFT (P3)
  # ============================================================================

  - finding_id: BV-059
    surface_id: mv-menu-heading
    lens: brand-voice
    severity: P3
    issue: "Mobile-vendor menu heading is `My Menu` while web chef menu is `Menu Management` and vendor-portal is `Menu Management`. Three apps, three different names for the same screen."
    evidence_excerpt: "mobile-vendor menu tab `My Menu`; web chef `Menu Management`; vendor-portal `Menu Management / {n} items in your menu`"
    related_surfaces: ["web-ux-chef-menu", "vp-ux-menu-title"]
    recommendation: "Chef-facing: `Menu` is enough as H1 on each platform. Drop `Management` (operational fluff) and `My` (redundant — they're in their own portal)."
    depends_on: null

  # ============================================================================
  # SECTION T — DATE / TIME RAW INPUT VIOLATION (P2)
  # ============================================================================

  - finding_id: BV-060
    surface_id: mc-catering-form-labels
    lens: brand-voice
    severity: P2
    issue: "Mobile-customer catering form label: `Event Date * (YYYY-MM-DD)` — raw ISO format hint visible to customers is engineering-grade UX. STYLE-GUIDE customer-facing: 12-hour, en-IN locale."
    evidence_excerpt: "mc-catering-form-labels `Event Date * (YYYY-MM-DD)`. Inventory note: `raw date format hint is unfriendly UX`"
    related_surfaces: ["mc-catering-event-types"]
    recommendation: "Use a date-picker component; drop the format hint entirely. Customer should never see `YYYY-MM-DD`."
    depends_on: null

  # ============================================================================
  # SECTION U — ABBREVIATION INCONSISTENCY (P3)
  # ============================================================================

  - finding_id: BV-061
    surface_id: dp-ux-active-est-time-label
    lens: brand-voice
    severity: P3
    issue: "Delivery-portal `Est. Time` (abbreviated) vs mobile-delivery uses full word context (`~{n} min`). Driver-facing label inconsistency. Inventory note: `A — abbreviation, may unclear`."
    evidence_excerpt: "dp-ux-active-est-time-label `Est. Time`; md-core-021 `~{n} min`"
    related_surfaces: ["md-core-021"]
    recommendation: "Use full word: `ETA` or `Time` — `Est.` is unfriendly. STYLE-GUIDE: Plain English over jargon."
    depends_on: null

  - finding_id: BV-062
    surface_id: dp-ux-earnings-avg
    lens: brand-voice
    severity: P3
    issue: "Delivery-portal `Avg/Delivery` uses slash compaction. STYLE-GUIDE: Plain English. `Per delivery` or `Average per delivery`."
    evidence_excerpt: "dp-ux-earnings-avg `Avg/Delivery`"
    related_surfaces: []
    recommendation: "`Per delivery` (sentence case)."
    depends_on: null

  - finding_id: BV-063
    surface_id: dp-ux-partner-performance
    lens: brand-voice
    severity: P3
    issue: "Delivery-portal partner detail uses `CSAT Score` — unexplained internal acronym for admin-facing label. STYLE-GUIDE: jargon out; explain or replace."
    evidence_excerpt: "dp-ux-partner-performance `Acceptance Rate / On-Time Rate / CSAT Score / Total Deliveries`. Inventory note: `A — CSAT abbreviation`"
    related_surfaces: []
    recommendation: "Replace with `Customer rating` (or expand on first use). Admin tooling deserves clarity too."
    depends_on: null

  # ============================================================================
  # SECTION V — APPROVAL TYPE TAXONOMY (P3)
  # ============================================================================

  - finding_id: BV-064
    surface_id: ap-approvals-types
    lens: brand-voice
    severity: P3
    issue: "Admin approval types: `Kitchen Onboarding; Document Verification; New Menu Item; Menu Update; Pricing Change; Kitchen Update; Driver Onboarding; Driver Document` — Title Case across 8 entries. Mix-mode: noun phrases vs verb-noun pairs (`Menu Update` vs `Pricing Change`)."
    evidence_excerpt: "ap-approvals-types"
    related_surfaces: ["ap-approvals-statuses", "ap-approvaldetail-detail-fields"]
    recommendation: "Sentence case: `Kitchen onboarding`, `Document verification`, `New menu item`, `Menu update`, `Pricing change`, `Kitchen update`, `Driver onboarding`, `Driver document`."
    depends_on: null

  # ============================================================================
  # SECTION W — ADMIN-SIDEBAR LABEL MISMATCH (P0)
  # ============================================================================

  - finding_id: BV-065
    surface_id: ap-layout-nav-reviews
    lens: brand-voice
    severity: P0
    issue: "Admin sidebar label `Reviews` routes to `/approvals` and the page is titled `Approval Reviews`. Two different concepts share one label — administrator clicks `Reviews` expecting product/order reviews but lands on approvals queue."
    evidence_excerpt: "ap-layout-nav-reviews: `Reviews` (sidebar nav label, inventory note: `nav for /approvals — ambiguous label`); ap-approvals-title `Approval Reviews` (inventory note: `page title — may conflict with sidebar 'Reviews'`)"
    related_surfaces: ["ap-approvals-title"]
    recommendation: "Rename sidebar item to `Approvals`. Reserve `Reviews` for actual product/chef reviews if/when that page exists. Page title can simply be `Approvals`."
    depends_on: null

  # ============================================================================
  # SECTION X — SETTINGS PAGE NOUN DRIFT (P3)
  # ============================================================================

  - finding_id: BV-066
    surface_id: ap-platsettings-title
    lens: brand-voice
    severity: P3
    issue: "Admin settings landing card titled `Platform` but the dedicated platform settings page H1 is also `Platform` while inventory shows web admin uses `Platform Settings`. Three labels for the same concept."
    evidence_excerpt: "ap-platsettings-title `Platform` vs web-ux-admin-settings `Platform Settings / Danger Zone` (web admin) vs ap-settings-card-titles includes `Platform; Commission, delivery fees, service areas, hours`"
    related_surfaces: ["web-ux-admin-settings"]
    recommendation: "Pick one: `Platform settings` (sentence case) for the page H1; the landing-card sub-link can be just `Platform`."
    depends_on: null

  # ============================================================================
  # SECTION Y — DOCUMENTATION NOTE: GENERAL PERSONA-TONE LEAKS (P3)
  # ============================================================================

  - finding_id: BV-067
    surface_id: ap-userdetail-verification
    lens: brand-voice
    severity: P3
    issue: "Admin user-detail verification card: `Email; Phone; Account Active; Verified; Not verified` — `Account Active` uses Title Case where verification labels use sentence case. Inconsistent within one card."
    evidence_excerpt: "ap-userdetail-verification `Verification; Email; Phone; Account Active; Verified; Not verified`"
    related_surfaces: ["ap-providerdetail-status-fields"]
    recommendation: "`Account active` (sentence case), and `Active` / `Inactive` (1-word values). Sentence case all status values."
    depends_on: null

  - finding_id: BV-068
    surface_id: ap-providerdetail-status-fields
    lens: brand-voice
    severity: P3
    issue: "Admin provider detail status fields: `Enabled; Yes; No; Active; Inactive; Last Used; Created; Never; Unlimited` — Title Case across all. Status pills should be sentence case as a class."
    evidence_excerpt: "ap-providerdetail-status-fields"
    related_surfaces: []
    recommendation: "Sentence case all values: `Enabled`, `Yes`, `No`, `Active`, `Inactive`, `Last used`, `Created`, `Never`, `Unlimited`."
    depends_on: BV-067

  # ============================================================================
  # SECTION Z — TIP / TIP HELPER PATTERN (P3)
  # ============================================================================

  - finding_id: BV-069
    surface_id: web-ux-checkout-tip
    lens: brand-voice
    severity: P3
    issue: "Web checkout tip: `100% of your tip goes to the home chef` — uses `home chef` lowercase correctly! But this is one of the few right examples. Use as anchor for fixing other surfaces."
    evidence_excerpt: "web-ux-checkout-tip `Add a tip / 100% of your tip goes to the home chef / No tip / Custom`"
    related_surfaces: []
    recommendation: "No change. Reference this surface in the rewrite work for BV-028/BV-029 as the correct pattern."
    depends_on: null

  # ============================================================================
  # SECTION AA — STAFF / INVITATION VERBS (P2)
  # ============================================================================

  - finding_id: BV-070
    surface_id: ap-staff-invite-dialog-title
    lens: brand-voice
    severity: P2
    issue: "Admin staff invite dialog says `Invite Staff Member` (Title Case); delivery-portal staff invite is `Send Staff Invitation` (Title Case); mobile-delivery staff is `Invite Staff Member` (Title Case). Three apps, three slightly different phrasings."
    evidence_excerpt: "ap-staff-invite-dialog-title `Invite Staff Member`; dp-ux-staff-invite-h `Send Staff Invitation`; md-core-103 `Invite Staff Member`"
    related_surfaces: ["dp-ux-staff-invite-btn", "md-core-102"]
    recommendation: "Standardize on `Invite staff member` (action verb). Button: `Invite staff`. Confirmation: `Invitation sent.`"
    depends_on: null

  - finding_id: BV-071
    surface_id: ap-staff-invitation-statuses
    lens: brand-voice
    severity: P3
    issue: "Staff invitation statuses (admin + delivery-portal): `Pending; Accepted; Expired; Revoked` — all Title Case as labels."
    evidence_excerpt: "ap-staff-invitation-statuses; dp-ux-staff-pending-h `Pending Invitations`"
    related_surfaces: []
    recommendation: "Sentence case: `Pending`, `Accepted`, `Expired`, `Revoked`."
    depends_on: BV-041

  # ============================================================================
  # SECTION BB — FOCUSED SAFETY NOTE (P0 — driver app)
  # ============================================================================

  - finding_id: BV-072
    surface_id: md-core-033
    lens: brand-voice
    severity: P1
    issue: "Mobile-delivery slide-to-confirm action label `Picked Up Order` (past tense for an action that hasn't happened) is confusing — driver swipes BEFORE picking up. Inventory explicitly flags: `Picked Up past-tense may be confusing as a future action`."
    evidence_excerpt: "mobile-delivery active.tsx:32 `at_pickup: { label: 'Picked Up Order', nextStatus: 'picked_up' }`"
    related_surfaces: ["md-core-032", "md-core-035"]
    recommendation: "Imperative future tense: `Confirm pickup` or `I have the order`. Slide-to-confirm is an irreversible decision — the verb must be unambiguous."
    depends_on: BV-021

  # ============================================================================
  # SECTION CC — CATERING STATUS DRIFT (P2)
  # ============================================================================

  - finding_id: BV-073
    surface_id: mc-catering-status-labels
    lens: brand-voice
    severity: P2
    issue: "Mobile-customer catering status pills: `Open / Quoted / Accepted / Completed / Cancelled` — all Title Case; differs from order status (`Pending / Accepted / Preparing / Ready...`). Two parallel status taxonomies, no shared vocabulary."
    evidence_excerpt: "mc-catering-status-labels"
    related_surfaces: ["web-ux-catering-quotes-detail", "web-ux-chef-catering"]
    recommendation: "Sentence case (`Open`, `Quoted`, `Accepted`, `Completed`, `Cancelled`). Document catering vs order status taxonomy distinction in STYLE-GUIDE."
    depends_on: BV-041

  # ============================================================================
  # SECTION DD — FAVORITES NAMING DRIFT (P3)
  # ============================================================================

  - finding_id: BV-074
    surface_id: mc-favorites-title
    lens: brand-voice
    severity: P3
    issue: "Mobile-customer favorites: tab label `Saved` but page title `Saved Chefs`. Web equivalent is `My Favorite Chefs`. Three names for one concept."
    evidence_excerpt: "mc-favorites-title `Saved Chefs` (page) vs tab label `Saved`; web-ux-favorites-heading `My Favorite Chefs`"
    related_surfaces: ["web-ux-favorites-heading", "web-ux-layout-nav"]
    recommendation: "Pick one verb. Recommend `Saved chefs` (cleaner than `Favorites` overlap with `Favourite Cuisines` filter). Update web and tab label to match."
    depends_on: null

  # ============================================================================
  # SECTION EE — SOCIAL FEED NAMING DRIFT (P3)
  # ============================================================================

  - finding_id: BV-075
    surface_id: web-ux-social-feed-heading
    lens: brand-voice
    severity: P3
    issue: "Web `Chef's Feed` (apostrophe-s, possessive); mobile `Social Feed`; chef-side `Social Feed`. Three different names for the feed."
    evidence_excerpt: "web-ux-social-feed-heading `Chef's Feed`; mc-social-title `Social Feed`; web-ux-chef-social `Social Feed`"
    related_surfaces: ["mc-social-title", "web-ux-chef-social"]
    recommendation: "Pick one: `Feed` is cleanest, `Social feed` is descriptive. Drop the possessive."
    depends_on: null

  # ============================================================================
  # SECTION FF — EMPTY-STATE TONE CHECK (P3)
  # ============================================================================

  - finding_id: BV-076
    surface_id: vp-ux-dashboard-no-pending
    lens: brand-voice
    severity: P3
    issue: "Vendor-portal `All caught up / New orders will appear here.` — good empty-state pattern (matches STYLE-GUIDE `Why empty → action`). Mobile-vendor and web chef equivalents are inconsistent."
    evidence_excerpt: "vp-ux-dashboard-no-pending `All caught up / New orders will appear here.` (good); web-ux-chef-orders-heading: `No orders found` (terse, no action); mobile-vendor orders: no equivalent empty state explicitly labelled"
    related_surfaces: ["web-ux-chef-orders-heading"]
    recommendation: "Use the vendor-portal pattern everywhere. The two-line pattern reads warmly and follows the empty-state rule."
    depends_on: null

  # ============================================================================
  # SECTION GG — TOAST / FEEDBACK COPY (P3)
  # ============================================================================

  - finding_id: BV-077
    surface_id: vp-ux-settings-password
    lens: brand-voice
    severity: P3
    issue: "Vendor-portal password: `Password updated successfully` (status) and `Failed to update password. Check your current password.` — first is over-padded (`updated successfully` is redundant), second is decent."
    evidence_excerpt: "vp-ux-settings-password `Password updated successfully / Failed to update password. Check your current password.`"
    related_surfaces: []
    recommendation: "Match STYLE-GUIDE toast format `past tense, ≤6 words, period`: `Password updated.` Error stays as-is."
    depends_on: null

  # ============================================================================
  # SECTION HH — RECEIPT / SUMMARY HEADER (P3)
  # ============================================================================

  - finding_id: BV-078
    surface_id: vp-ux-orders-history-summary
    lens: brand-voice
    severity: P3
    issue: "Vendor-portal order-history summary cards: `Delivered / Cancelled / Revenue` — fine pattern. But admin-portal Orders uses Title Case `Delivering / Delivered / Cancelled / Refunded` (vs lowercase noun statuses). Drift."
    evidence_excerpt: "vp-ux-orders-history-summary `Delivered / Cancelled / Revenue`; ap-orders-filter-status `Delivering; Delivered; Cancelled; Refunded`"
    related_surfaces: ["ap-orders-filter-status"]
    recommendation: "Sentence case applies regardless of context. Both should be sentence case for the value labels."
    depends_on: BV-041

  # ============================================================================
  # SECTION II — MISC PERSONA INVERSION (P2/P3)
  # ============================================================================

  - finding_id: BV-079
    surface_id: dp-ux-stripe-make-primary
    lens: brand-voice
    severity: P3
    issue: "Delivery-portal driver-facing CTA: `Make Stripe My Primary Gateway` — Title Case AND first-person `My` reads marketing-y for the driver tone (per STYLE-GUIDE: telegraphic ≤4 words)."
    evidence_excerpt: "dp-ux-stripe-make-primary `Make Stripe My Primary Gateway` (5 words, Title Case, first-person)"
    related_surfaces: ["dp-ux-stripe-switch-razorpay"]
    recommendation: "`Use Stripe as default` or `Set as default` — 3-4 words, sentence case, no first-person."
    depends_on: null

  - finding_id: BV-080
    surface_id: dp-ux-stripe-active-gateway
    lens: brand-voice
    severity: P3
    issue: "Delivery-portal `Active payout gateway: {Stripe/Razorpay}` — sentence-case `Active payout gateway` is correct! Use this as a positive anchor."
    evidence_excerpt: "dp-ux-stripe-active-gateway"
    related_surfaces: []
    recommendation: "No change. Use as reference."
    depends_on: null

  # ============================================================================
  # SECTION JJ — REVIEW-RELATED COPY (P3)
  # ============================================================================

  - finding_id: BV-081
    surface_id: web-ux-chefdetail-reviews
    lens: brand-voice
    severity: P3
    issue: "Web chef detail: `Customer Reviews / Chef's Response:` (with trailing colon and Title Case) — STYLE-GUIDE: form labels are `noun, sentence case, no colons`."
    evidence_excerpt: "web-ux-chefdetail-reviews `Customer Reviews / Chef's Response:`"
    related_surfaces: ["mv-reviewcard-your-reply"]
    recommendation: "`Customer reviews` and `Chef's response` (no colon)."
    depends_on: null

  - finding_id: BV-082
    surface_id: mv-reviewdetail-input-ph
    lens: brand-voice
    severity: P3
    issue: "Mobile-vendor review reply placeholder: `Write a thoughtful reply to this review...` — `thoughtful` is a marketing coach-y hint; chef tone is operational."
    evidence_excerpt: "mv-reviewdetail-input-ph `Write a thoughtful reply to this review...`"
    related_surfaces: []
    recommendation: "Plain: `Reply to this review…` (sentence case, no thought-coaching)."
    depends_on: null

  # ============================================================================
  # SECTION KK — FINAL TYPOGRAPHY/MICROCOPY NOTES (P3)
  # ============================================================================

  - finding_id: BV-083
    surface_id: ap-secsettings-2fa-account
    lens: brand-voice
    severity: P3
    issue: "Admin 2FA: `2FA for your account / Enabled — login requires a 6-digit code / Disabled — enable to protect your admin account / Enable / Disable` — uses em-dash–led status descriptions. Good pattern but uses `login` (lowercase noun) — STYLE-GUIDE bans `login` as a noun form too; should be `sign-in`."
    evidence_excerpt: "ap-secsettings-2fa-account `Enabled — login requires a 6-digit code`"
    related_surfaces: []
    recommendation: "`Enabled — sign-in requires a 6-digit code`. Apply the noun rule consistently."
    depends_on: BV-036

  - finding_id: BV-084
    surface_id: ap-secsettings-session-fields
    lens: brand-voice
    severity: P3
    issue: "Admin security: `Access token TTL (hours)` / `Refresh token TTL (days)` — `TTL` is engineering jargon for an admin label."
    evidence_excerpt: "ap-secsettings-session-fields"
    related_surfaces: []
    recommendation: "`Access token lifetime (hours)`. Plain English (STYLE-GUIDE Rule 2)."
    depends_on: null

  - finding_id: BV-085
    surface_id: ap-secsettings-apikey-fields
    lens: brand-voice
    severity: P3
    issue: "Admin API key form: `I've saved it` button after revealing the secret. Cute but breaks neutral admin tone."
    evidence_excerpt: "ap-secsettings-apikey-fields `I've saved it`"
    related_surfaces: []
    recommendation: "Neutral: `Done` or `Close`."
    depends_on: null

  # ============================================================================
  # SECTION LL — PROFILE PAGE DRIFT (P2)
  # ============================================================================

  - finding_id: BV-086
    surface_id: mc-profile-sections
    lens: brand-voice
    severity: P2
    issue: "Mobile-customer profile section uses `Personal Info / Food Preferences / More`; web customer uses `Personal Information / Food Preferences`; web profile-tabs uses `Profile / Preferences / Addresses / Payment Methods / Notifications / Security`. Same persona, three navigation models."
    evidence_excerpt: "mc-profile-sections `Personal Info / Food Preferences / More`; web-ux-profile-personal-info `Personal Information`; web-ux-profile-tabs `Profile / Preferences / Addresses / Payment Methods / Notifications / Security`"
    related_surfaces: ["web-ux-profile-tabs", "web-ux-profile-personal-info"]
    recommendation: "Standardize on `Personal info` (matches mobile and is shorter). Web should adopt mobile's section names for the same flow."
    depends_on: BV-038

  - finding_id: BV-087
    surface_id: web-ux-profile-preferences
    lens: brand-voice
    severity: P2
    issue: "Web profile uses both `Food Preferences` and `Favourite Cuisines` (British spelling) within the same Preferences tab. Inconsistent within one screen."
    evidence_excerpt: "web-ux-profile-preferences `Food Preferences / Dietary Preferences / Food Allergies / Favourite Cuisines / Spice Tolerance / Household Size`"
    related_surfaces: ["mc-profile-cuisines"]
    recommendation: "Pick spelling: STYLE-GUIDE shows `en-IN locale` — `Favourite` (British) is the en-IN standard. OK if consistent across all surfaces. But fix capitalization: `Food preferences`, `Dietary preferences`, `Food allergies`, `Favourite cuisines`, `Spice tolerance`, `Household size`."
    depends_on: null

  # ============================================================================
  # SECTION MM — SUBSCRIPTION / FEATURE FLAGS (P3)
  # ============================================================================

  - finding_id: BV-088
    surface_id: md-core-066
    lens: brand-voice
    severity: P3
    issue: "Mobile-delivery settings: `View Subscription Plan` — feature visible but likely not implemented? Title Case. Either feature exists or it shouldn't be in nav."
    evidence_excerpt: "md-core-066 `View Subscription Plan`"
    related_surfaces: []
    recommendation: "If feature ships: `Subscription plan`. If not: remove until ready (BV-034 / coming-soon rule)."
    depends_on: null

  # ============================================================================
  # SECTION NN — DESTRUCTIVE-ACTION COPY (P3)
  # ============================================================================

  - finding_id: BV-089
    surface_id: mv-settings-delete-account
    lens: brand-voice
    severity: P3
    issue: "`Delete Account` (mobile-vendor + mobile-delivery) — Title Case for the most consequential customer-facing destructive action."
    evidence_excerpt: "mv-settings-delete-account `Delete Account`; md-core-067 `Delete Account`"
    related_surfaces: []
    recommendation: "`Delete account` (sentence case). The destructive weight should be in the button color (paprika), not capitalization."
    depends_on: BV-026

  # ============================================================================
  # SECTION OO — DOC UPLOAD STATE LABELS (P3)
  # ============================================================================

  - finding_id: BV-090
    surface_id: mv-onb-review-doc-uploaded
    lens: brand-voice
    severity: P3
    issue: "Mobile-vendor doc state: `Uploaded` / `Not uploaded` (sentence case — good!) but docs source buttons say `Camera / Gallery / PDF` (Title Case — fine for noun chips)."
    evidence_excerpt: "mv-onb-review-doc-uploaded `Uploaded`; mv-onb-review-doc-not-uploaded `Not uploaded`; mv-onb-docs-source-camera `Camera`, mv-onb-docs-source-pdf `PDF`"
    related_surfaces: []
    recommendation: "No change. Document this as a positive anchor — sentence case state labels with Title Case-allowed proper nouns (`PDF`, `Camera`)."
    depends_on: null

  # ============================================================================
  # SECTION PP — FINAL DRIFT SUMMARY ITEMS (P3)
  # ============================================================================

  - finding_id: BV-091
    surface_id: web-ux-checkout-delivery-time
    lens: brand-voice
    severity: P3
    issue: "Web checkout: `As soon as possible / Usually 30-45 minutes / Schedule for later` — `Usually 30-45 minutes` reads like a vendor disclaimer. Customer tone should be promise-light."
    evidence_excerpt: "web-ux-checkout-delivery-time `Delivery Time / As soon as possible / Usually 30-45 minutes / Schedule for later`"
    related_surfaces: []
    recommendation: "`Delivery time / As soon as possible / 30–45 min typical / Schedule for later`. Use en-dash and the abbreviation; drop the hedge-word `usually`."
    depends_on: null

  - finding_id: BV-092
    surface_id: web-ux-orderdetail-sections
    lens: brand-voice
    severity: P3
    issue: "Web order detail back link: `Back to Orders` (Title Case). Mobile uses `Back` only. Cross-app drift."
    evidence_excerpt: "web-ux-orderdetail-sections `Back to Orders`; mobile-delivery delivery/[id].tsx similar pattern with sentence-case behavior; vendor-portal `Back to Profile`/`Back to Menu`/`Back to Earnings` (Title Case across)"
    related_surfaces: ["dp-ux-partner-detail-back", "vp-ux-payouts-title"]
    recommendation: "`Back to orders` (sentence case). Or just `Back` — relying on browser context. Pick one pattern."
    depends_on: BV-038

  # ============================================================================
  # SECTION QQ — PAYOUT VOCABULARY (P3)
  # ============================================================================

  - finding_id: BV-093
    surface_id: dp-ux-stripe-title
    lens: brand-voice
    severity: P3
    issue: "Delivery-portal `Stripe (International Payouts)` — parenthetical Title Case mid-sentence is grammatically odd."
    evidence_excerpt: "dp-ux-stripe-title `Stripe (International Payouts)`"
    related_surfaces: []
    recommendation: "`Stripe (international payouts)` — sentence case inside parens."
    depends_on: null

  - finding_id: BV-094
    surface_id: dp-ux-stripe-intro
    lens: brand-voice
    severity: P3
    issue: "Delivery-portal Stripe intro: `Accept delivery payouts in your local currency. Stripe handles KYC and bank verification on its hosted pages.` — `KYC` jargon for driver audience."
    evidence_excerpt: "dp-ux-stripe-intro"
    related_surfaces: []
    recommendation: "Replace `KYC` with `identity verification` for driver clarity (Plain English, Rule 2)."
    depends_on: null

  # ============================================================================
  # SECTION RR — DRIVER STATE-COPY DRIFT (P2)
  # ============================================================================

  - finding_id: BV-095
    surface_id: dp-ux-dashboard-partner-online
    lens: brand-voice
    severity: P2
    issue: "Driver online-state copy varies between surfaces: `Ready for deliveries` (dp dashboard), `You are Online` (mobile-delivery dashboard), `Currently online — tap to toggle availability` (web delivery layout aria-label). Three phrasings, one state."
    evidence_excerpt: "dp-ux-dashboard-partner-online `Ready for deliveries`; md-core-006 `You are Online`; web-ux-delivery-layout-toggles `Currently online — tap to toggle availability`"
    related_surfaces: ["md-core-006", "md-core-007"]
    recommendation: "Pick one online-state phrase. Driver tone: telegraphic ≤4 words. `Online` (1 word). Subtitle if needed: `Receiving requests.`"
    depends_on: null

  - finding_id: BV-096
    surface_id: dp-ux-dashboard-youre-offline
    lens: brand-voice
    severity: P2
    issue: "Driver offline state: `You're offline` (dp), `You are Offline` (mobile), `Currently offline` (dp dashboard partner offline), `You're Offline` (mobile-delivery available tab). Apostrophe vs no apostrophe + Title Case vs sentence case."
    evidence_excerpt: "dp-ux-dashboard-youre-offline `You're offline`; md-core-007 `You are Offline`; dp-ux-dashboard-partner-offline `Currently offline`; md-core-016 `You're Offline`"
    related_surfaces: ["md-core-007", "md-core-016"]
    recommendation: "Pick one: `Offline` (1 word, telegraphic, sentence case). Drop the `You're/You are` framing for driver brevity."
    depends_on: BV-095

  # ============================================================================
  # SECTION SS — EARNINGS VS REVENUE NOUN (P2)
  # ============================================================================

  - finding_id: BV-097
    surface_id: mv-earnings-total
    lens: brand-voice
    severity: P2
    issue: "Mobile-vendor `Total Earnings` vs mobile-vendor analytics `Total Revenue` — same chef, two screens, two nouns. Earnings is chef-perspective (post-platform-fee); Revenue is admin-perspective. Confusing if the values differ."
    evidence_excerpt: "mv-earnings-total `Total Earnings`; mv-analytics-total-revenue `Total Revenue`. Inventory note: `Drift: earnings.tsx uses 'Total Earnings' — verify mental model is consistent.`"
    related_surfaces: ["mv-analytics-total-revenue", "vp-ux-earnings-cards"]
    recommendation: "Chef-facing: always `Earnings`. Admin-facing: `Revenue`. Don't mix in one persona's UI."
    depends_on: null

  # ============================================================================
  # SECTION TT — REPORT NOTE (informational, not finding) ===================
  # ============================================================================
  # Total findings: 97
  # Severity breakdown:
  #   P0: 12 (cross-app order-status taxonomy + brand identity Fe3dr/HomeChef + AI-slop trust metrics + currency $ vs ₹ + admin sidebar label mismatch + Pickup/Dropoff ALL CAPS)
  #   P1: 21 (sign-out epidemic, place-order/cancel-order Title Case, persona-tone leaks, in-transit/delivering/on-the-way drift)
  #   P2: 24 (cross-app nav drift, persona-tone secondary, cuisine taxonomy, driver state phrasing)
  #   P3: 40 (capitalization in deep surfaces, abbreviations, microcopy polish)
```
