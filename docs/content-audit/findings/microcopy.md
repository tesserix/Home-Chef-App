# Findings — Microcopy

Category: microcopy
Lenses applied: technical-writer, legal, business-analyst, brand-voice
Inventory slice size: 281 surfaces
Total findings: 311

## Severity summary

| Severity | TW | Legal | BA | BV | Total |
|---|---|---|---|---|---|
| P0 | 0 | 8 | 6 | 8 | 22 |
| P1 | 1 | 21 | 11 | 34 | 67 |
| P2 | 23 | 13 | 19 | 13 | 68 |
| P3 | 96 | 5 | 10 | 43 | 154 |

Note: counts reflect the source YAMLs verbatim. Task brief listed BA P1=10 / P2=20; the source BA YAML actually has P1=11 / P2=19 (one P2 reclassified to P1 between brief and audit). All 311 finding records preserved.

## Cross-lens consensus surfaces (flagged by multiple lenses)

Surfaces flagged by all 4 lenses (highest priority — every lens agreed there's a problem):

- **`web-ux-cart-promo`** — flagged by TW(P2), Legal(P1), BA(P0, P0), BV(P1) — hardcoded promo code 'FE3DR10' leaks across all customers; exclamation in success toast; misleading-discount exposure; brand-identity drift
- **`vp-ux-orders-live-actions`** — flagged by TW(P2), Legal(P2), BA(P1), BV(P1) — chef Accept/Reject/Mark Ready buttons use Title Case; no captured rejection reason (CP Rules grievance defence); "Waiting for pickup" status mixed with action chips
- **`vp-ux-notifs-respond-form`** — flagged by TW(P3), Legal(P2), BA(P2), BV(P1) — chef-admin reply form: placeholder verbose, '&' in button label, no consent/retention disclosure, voice mismatch vs admin tone
- **`vp-ux-notifs-sla-line`** — flagged by TW(P3), Legal(P3), BA(P1), BV(P3) — "Typically reviewed within 24-48 hours" SLA is vague AND contradicts `mv-menuedit-price-change-banner` ("may take 24 hours"); two different SLAs cited for the same workflow
- **`mv-menuedit-price-change-banner`** — flagged by TW(P2), Legal(P1), BA(P1), BV(P1) — mobile-vendor menu-edit price-change banner SLA drift vs `vp-ux-notifs-sla-line`; missing approval-cadence disclosure
- **`vp-onb-policies-payout-info`** — flagged by TW(P3), Legal(P1), BA(P1), BV(P3) — chef onboarding payout-info defers bank/UPI setup post-approval; no RBI PA disclosure or reconciliation reference
- **`dp-mc-stripe-cells`** — flagged by TW(P3), Legal(P1), BA(P1), BV(P3) — delivery-portal Stripe payout cells: PII handling, brand-domain drift, status enum strings as user copy

Surfaces flagged by 3 lenses (also high-priority rewrites):

- **`dp-mc-portal-footer`** — flagged by TW(P3), BA(P0, P0), BV(P0) — delivery-portal footer renders 'Fe3dr Delivery Portal' while platform is 'Home Chef' (brand-identity contradiction at entry surface)
- **`mc-perm-camera`** — flagged by Legal(P0), BA(P1), BV(P1) — iOS camera permission string "Used to take photos" is vague (App Store review risk + DPDP §6 lawful-purpose ambiguity)
- **`mc-perm-location`** — flagged by TW(P3), Legal(P0), BV(P2) — iOS location permission lacks specific-purpose disclosure (DPDP §6 + Apple guidance)
- **`mc-perm-faceid`** — flagged by TW(P3), Legal(P1), BV(P2) — Face ID/biometric permission string voice mismatch and disclosure gap
- **`vp-ux-menu-form-fields`** — flagged by TW(P2), Legal(P0), BV(P1) — chef menu-form fields: Title Case labels, no allergen disclosure required by FSSAI Labelling Regs
- **`vp-ux-menu-delete-confirm`** — flagged by Legal(P2), BA(P1), BV(P1) — chef menu item delete confirm uses banned 'Are you sure?' anti-pattern; no consequence specificity; no retention/audit disclosure
- **`vp-ux-menu-bulk-confirm`** — flagged by Legal(P2), BA(P1), BV(P1) — same 'Are you sure?' anti-pattern on bulk-delete; missing affected-count specificity
- **`vp-ux-menu-form-allergen-help`** — flagged by TW(P3), Legal(P1), BA(P2) — allergen-entry helper verbose; missing FSSAI 8-allergen list disclosure
- **`vp-ux-reviews-reply`** — flagged by TW(P2), BA(P2), BV(P1, P1) — chef review-reply success toast 'Reply posted successfully' with redundant 'successfully'; voice drift; no Display Rules disclosure
- **`web-ux-checkout-instructions`** — flagged by TW(P2), Legal(P1), BA(P2) — checkout 'Special Instructions' Title Case + verbose placeholder; no max-length/consent disclosure
- **`web-ux-chef-card-favorite-toast-loggedout`** — flagged by TW(P2), BA(P1), BV(P0) — favorite-toast uses banned verb 'log in' instead of 'Sign in'; brand voice violation
- **`web-ux-orderdetail-cancel-reason`** — flagged by TW(P3), Legal(P1), BV(P3) — order-detail cancel-reason label uses colon; no refund-reconciliation linkage (RBI PA)
- **`web-ux-orders-search`** — flagged by TW(P3), BA(P2), BV(P3) — orders search drift: 'Search your orders' aria vs 'Search orders...' placeholder; sibling admin search differs again
- **`mc-catering-placeholders`** — flagged by Legal(P2), BA(P2), BV(P3) — catering form placeholders: raw '25000' budget without ₹ prefix; no consent capture; date/guest examples vary
- **`vp-onb-kitchen-desc-field`** — flagged by TW(P3), BA(P2), BV(P2) — chef onboarding description field helper text length; customer visibility not flagged in helper
- **`vp-onb-ops-min-order`** — flagged by TW(P3), BA(P2), BV(P3) — minimum-order helper "Set to 0 for no minimum" missing ₹ prefix; voice drift
- **`vp-onb-personal-avatar-label`** — flagged by Legal(P2), BA(P3), BV(P3) — chef onboarding profile-photo label "(Optional)" inside label vs sentence-case rule; no PII disclosure
- **`ap-secsettings-session-actions`** — flagged by TW(P2), Legal(P1), BV(P3) — admin security session actions; no audit-trail confirmation (DPDP §8 audit log)
- **`dp-mc-action-required`** — flagged by Legal(P1), BA(P2), BV(P3) — driver "Action Required" Title Case + no due-process language for partner-suspend

Top P0/P1 dual-lens surfaces (severity-weighted):

- **`web-ux-layout-auth-cta`** — flagged by BA(P0), BV(P0) — web header "Login / Sign Up" violates style guide (both forms wrong, mixed case) — entry-surface trust break
- **`ap-auth-email-placeholder`** — flagged by TW(P3), BV(P0) — admin login example "admin@fe3dr.com" — brand-domain drift on first-touch surface
- **`ap-layout-account-logout`** — flagged by BA(P3), BV(P0) — admin account menu 'Logout' instead of 'Sign out' — banned-vocab on every-screen surface
- **`web-ux-cart-delivery-info`** — flagged by TW(P3), BA(P0) — cart delivery-info copy hides cost surprises; conversion-critical
- **`vp-ux-menu-form-dietary-tags`** — flagged by TW(P3), Legal(P0) — dietary-tag inconsistency + FSSAI compliance gap (no allergen-tag mandate)
- **`vp-ux-kitchen-payout-fields`** — flagged by TW(P3), Legal(P0) — chef payout setup fields: missing RBI PA-mandated nominee/PAN disclosure
- **`vp-ux-settings-payout-form`** — flagged by TW(P3), Legal(P0) — payout-settings form same RBI PA gap on Settings page
- **`md-mic-001`** — flagged by TW(P3), BV(P0) — mobile-delivery microcopy P0 brand drift
- **`web-ux-chef-card-verified`** — flagged by TW(P3), Legal(P1) — "Verified chef" unqualified verification claim (CP Rules 2020 misleading-representation exposure)
- **`vp-onb-docs-additional-title`** / **`vp-onb-docs-kitchen-photos-title`** — Legal(P1) + BV(P1/P2) — onboarding doc-upload titles; FSSAI/KYC disclosure gap
- **`vp-onb-nav-submit-app`** — BA(P1), BV(P1) — chef onboarding submit-application drift
- **`vp-ux-earnings-sections`** — BA(P2), BV(P1) — chef earnings page section drift
- **`ap-approvaldetail-actions`** / **`ap-staffdetail-actions`** — Legal(P1) + BV(P1) — admin approval/staff actions: Title Case + audit-trail gap
- **`ap-providercreate-submit`** / **`ap-settings-payment-test-buttons`** — TW + BV(P1) — admin provider/payment action button drift
- **`ap-chefs-card-unnamed`** — TW(P3), BV(P1) — admin chef card "Unnamed" fallback voice
- **`mc-checkout-note`** — TW(P3), Legal(P1) — mobile-customer checkout note helper; no consent capture
- **`mc-order-card-meta`** — BA(P2), BV(P1) — order-card metadata drift across siblings

## Technical Writer findings

```yaml
# Technical Writer lens audit — MICROCOPY category
# Category total rows: 281
# Approach: pattern-grouped findings. Where the same violation repeats >3 surfaces
# in a single app, ONE pattern finding is filed referencing all surface_ids.
# Single-surface issues stand alone. Ordered by severity then surface.

# =============================================================================
# PATTERN FINDINGS (cross-surface, same root cause)
# =============================================================================

findings:

  - finding_id: TW-001
    surface_id: PATTERN-vendor-portal-title-case-buttons
    lens: technical-writer
    severity: P2
    issue: "Vendor-portal buttons use Title Case across the app — style guide §4 requires sentence case for buttons (verb-first, ≤3 words, sentence case)"
    evidence_excerpt: "Save Changes / Create Item / Add Image / Mark Ready / Start Preparing / Order History / Submit Application / Review Application / Post Reply / Respond & Send"
    recommendation: "Lowercase the second word: 'Save changes', 'Create item', 'Add image', 'Mark ready', 'Start preparing', 'Order history', 'Submit application', 'Review application', 'Post reply', 'Respond and send'. Applies to surfaces: vp-ux-menu-form-actions, vp-ux-menu-form-new-category, vp-ux-orders-live-actions, vp-onb-nav-review-app, vp-onb-nav-submit-app, vp-ux-reviews-reply, vp-ux-notifs-actions, vp-ux-notifs-respond-form, vp-ux-dashboard-add-item (already correct: 'Add item')."
    depends_on: null

  - finding_id: TW-002
    surface_id: PATTERN-vendor-portal-title-case-labels
    lens: technical-writer
    severity: P2
    issue: "Vendor-portal form section headings and field labels use Title Case throughout — style guide §4 form-label format is sentence case noun"
    evidence_excerpt: "Basic Information / Dietary Information / Preparation Details / Item Name / Prep Time / Portion Size / Dietary Tags / Kitchen Photos / Additional Documents / Profile Photo / Address Line 1 / Years of Cooking Experience / Operating Hours / Delivery Radius / Minimum Order Value / Delivery Fee / Cuisines You Specialize In / Signature Dishes & Specialties"
    recommendation: "Sentence-case every label: 'Basic information', 'Item name', 'Prep time', 'Kitchen photos', 'Profile photo', 'Address line 1', 'Years of cooking experience', 'Operating hours', 'Delivery radius', 'Minimum order value', 'Delivery fee', 'Cuisines you specialize in', 'Signature dishes and specialties'. Replace '&' with 'and' in labels. Applies to: vp-ux-menu-form-section-titles, vp-ux-menu-form-fields, vp-onb-docs-kitchen-photos-title, vp-onb-docs-additional-title, vp-onb-personal-avatar-label, vp-onb-personal-address-fields, vp-onb-kitchen-name-field, vp-onb-kitchen-cuisines-title, vp-onb-kitchen-specialties-title, vp-onb-ops-radius, vp-onb-ops-min-order, vp-onb-ops-delivery-fee, vp-onb-ops-hours-title, vp-onb-review-section-labels, vp-onb-review-field-labels."
    depends_on: null

  - finding_id: TW-003
    surface_id: PATTERN-vendor-portal-title-case-status-badges
    lens: technical-writer
    severity: P2
    issue: "Vendor-portal status badges and chips use Title Case — sentence case applies to all chrome including badges"
    evidence_excerpt: "Pending Review / Under Review / Info Requested / Picked Up / All Categories / Featured / Approved / Unavailable / Active / Inactive"
    recommendation: "Sentence-case all multi-word badges: 'Pending review', 'Under review', 'Info requested', 'Picked up', 'All categories'. Single-word badges (Approved, Pending, Active) already correct. Applies to: vp-ux-notifs-status-labels, vp-ux-menu-card-labels, vp-ux-menu-search-placeholder, vp-ux-orderstatus-badges, vp-ux-statusbadge-labels, vp-ux-profile-doc-statuses."
    depends_on: null

  - finding_id: TW-004
    surface_id: PATTERN-admin-portal-title-case-buttons
    lens: technical-writer
    severity: P2
    issue: "Admin-portal action buttons use Title Case across the app — style guide §4 requires sentence case (admin persona tone matrix still uses sentence case)"
    evidence_excerpt: "Request Info / Add Provider / Edit Provider / Delete Provider / Test Connection / Manage Providers / Invite Staff / Change Role / Confirm Deactivation / Confirm Reactivation / Verify Kitchen / Reject Application / Suspend Kitchen / Download CSV / Sign in with Email / Continue with Google / Continue with Facebook / Save Keys / Verify & sign in / Resend Invitation / Revoke Invitation / Create Provider / Add Mapping"
    recommendation: "Sentence-case all: 'Request info', 'Add provider', 'Edit provider', 'Delete provider', 'Test connection', 'Manage providers', 'Invite staff', 'Change role', 'Confirm deactivation', 'Confirm reactivation', 'Verify kitchen', 'Reject application', 'Suspend kitchen', 'Download CSV' (acronym OK), 'Sign in with email', 'Continue with Google'/'Facebook' (brand names retain caps), 'Save keys', 'Verify and sign in' (& → and), 'Resend invitation', 'Revoke invitation', 'Create provider', 'Add mapping'. Applies to: ap-approvaldetail-actions, ap-providers-cta-add, ap-providerdetail-actions, ap-delivery-cta-providers, ap-staff-cta-invite, ap-staffdetail-actions, ap-chefs-action-verify-aria, ap-chefs-action-reject-aria, ap-chefs-action-suspend-aria, ap-exports-button, ap-auth-email-cta, ap-auth-social-google, ap-auth-social-facebook, ap-settings-payment-test-buttons, ap-auth-totp-verify-button, ap-staff-invite-action-aria, ap-providercreate-submit."
    depends_on: null

  - finding_id: TW-005
    surface_id: PATTERN-admin-portal-title-case-filters-tooltips
    lens: technical-writer
    severity: P2
    issue: "Admin filters, dropdown options and tooltips use Title Case — should be sentence case"
    evidence_excerpt: "All Statuses / Enabled / Disabled / Per Delivery / Per KM / Flat Rate / Live Mode / Test Mode / Action Required / Pending Verification / Bank Transfer / Current Role / Edit keys / Copy webhook URL / Account menu / Open navigation / Close navigation"
    recommendation: "Sentence-case: 'All statuses', 'Per delivery', 'Per km' or 'Per KM' if acronym, 'Flat rate', 'Live mode', 'Test mode', 'Action required', 'Pending verification', 'Bank transfer', 'Current role'. aria-labels already mostly correct ('Edit keys', 'Account menu', 'Open navigation' OK). Applies to: ap-delivery-filter-all, ap-providers-filter-enabled, ap-providers-pricing-labels, ap-settings-payment-statuses, dp-mc-action-required, dp-mc-pending-verification, ap-staffdetail-current-role-label."
    depends_on: null

  - finding_id: TW-006
    surface_id: PATTERN-vendor-portal-section-headings-title-case
    lens: technical-writer
    severity: P2
    issue: "Vendor-portal page section headings use Title Case — sentence case applies everywhere except brand/product names"
    evidence_excerpt: "Order Trends / Revenue Trends / Popular Items / Revenue by Category / Peak Hours / Daily Earnings / Last 14 days / Top Selling Items / Recent Payouts / Kitchen Review Status / Notifications / Admin Notes / Kitchen Setup / Operating hours / Rating Distribution / Top Selling Items"
    recommendation: "Sentence-case: 'Order trends', 'Revenue trends', 'Popular items', 'Revenue by category', 'Peak hours', 'Daily earnings', 'Top selling items', 'Recent payouts', 'Kitchen review status', 'Admin notes', 'Kitchen setup', 'Rating distribution'. Applies to: vp-ux-analytics-charts, vp-ux-earnings-sections, vp-ux-notifs-section-titles, vp-ux-notifs-admin-notes, vp-ux-profile-kitchen-setup-link, vp-ux-reviews-rating-summary."
    depends_on: null

  - finding_id: TW-007
    surface_id: PATTERN-banned-vocab-logout
    lens: technical-writer
    severity: P2
    issue: "Multiple surfaces use banned term 'Logout' / 'Login' — style guide §3 mandates 'Sign in' / 'Sign out' (two words, with sign)"
    evidence_excerpt: "Logout (dp-mc-logout, ap-layout-account-logout) / Login (web-ux-layout-auth-cta 'Login / Sign Up') / Sign Up (mixed case)"
    recommendation: "Replace 'Logout' with 'Sign out' (sentence case) on dp-mc-logout and ap-layout-account-logout. Replace 'Login / Sign Up' with 'Sign in / Sign up' on web-ux-layout-auth-cta. Within delivery-portal there is also internal drift: settings page reportedly says 'Sign Out' while layout says 'Logout' — normalize to 'Sign out'."
    depends_on: null

  - finding_id: TW-008
    surface_id: PATTERN-banned-vocab-sign-up-vs-signup
    lens: technical-writer
    severity: P2
    issue: "Header CTA renders 'Sign Up' (Title Case) — style guide allows 'Sign up' (verb, two words) but not 'Signup' or Title Case 'Sign Up'"
    evidence_excerpt: "Login / Sign Up"
    recommendation: "Replace with 'Sign in / Sign up' on web-ux-layout-auth-cta. Sentence case, two words, no slash padding."
    depends_on: null

  - finding_id: TW-009
    surface_id: PATTERN-loading-message-vague
    lens: technical-writer
    severity: P3
    issue: "'Loading...' appears as default load state across every app — vague, not glanceable, no context for what is loading"
    evidence_excerpt: "Loading..."
    recommendation: "Keep 'Loading...' only on full-screen route-level loaders. For inline/section loads, prefer specifics: 'Loading orders…', 'Loading menu…', 'Loading chefs…'. Driver-app variant (≤4 words) can stay 'Loading…'. Applies to: web-ux-loading-default, vp-ux-loading-screen, dp-mc-loading-msg, vp-onb-personal-address-loading."
    depends_on: null

  - finding_id: TW-010
    surface_id: PATTERN-confirm-anti-vague
    lens: technical-writer
    severity: P2
    issue: "Delete/bulk-delete confirms use the banned vague 'Are you sure?' anti-pattern — style guide §4 modal-subtitle rule requires explaining the consequence specifically"
    evidence_excerpt: "Are you sure you want to delete this item? / Delete {n} items? This cannot be undone."
    recommendation: "Rewrite to consequence-first: 'Delete this menu item? Customers will no longer see it.' and 'Delete {n} items? They will be removed from your menu immediately.' Avoid generic 'This cannot be undone' — name the specific irreversible effect. Applies to: vp-ux-menu-delete-confirm, vp-ux-menu-bulk-confirm."
    depends_on: null

  - finding_id: TW-011
    surface_id: PATTERN-aria-tooltip-mismatch
    lens: technical-writer
    severity: P3
    issue: "Several admin surfaces ship both Title Case and sentence case for the same action (aria vs tooltip drift) — pick one and propagate"
    evidence_excerpt: "Verify kitchen / Verify Kitchen — Reject application / Reject Application — Suspend kitchen / Suspend Kitchen — Resend Invitation / Resend invitation — Revoke Invitation / Revoke invitation — Suspend user / Suspend User — Activate user / Activate User"
    recommendation: "Standardize on sentence case ('Verify kitchen', 'Reject application', 'Suspend kitchen', etc.) for both aria-label and tooltip/title. Applies to: ap-chefs-action-verify-aria, ap-chefs-action-reject-aria, ap-chefs-action-suspend-aria, ap-staff-invite-action-aria, ap-users-action-suspend-aria, ap-users-action-activate-aria."
    depends_on: null

  - finding_id: TW-012
    surface_id: PATTERN-empty-state-format
    lens: technical-writer
    severity: P3
    issue: "Several empty-state strings drop the 'Why → one action' formula and leave only the 'why'"
    evidence_excerpt: "No reviews yet / No sales data yet / No payouts yet / No reviews / Unassigned / Not provided / Not assigned / Never"
    recommendation: "Add a one-action follow-up where there is a meaningful next step. e.g. 'No reviews yet. Encourage customers to rate.' for vp-ux-dashboard-stats-labels; 'No sales data yet. Publish a menu to start selling.' for vp-ux-earnings-sections; 'No payouts yet. Payouts start after your first delivered order.' Leave terse fallbacks ('Not provided', 'Never') as-is — they are table-cell fallbacks, not empty states. Applies to: vp-ux-dashboard-stats-labels, vp-ux-earnings-sections, ap-chefs-stat-no-reviews."
    depends_on: null

  - finding_id: TW-013
    surface_id: PATTERN-helper-text-missing-mobile-vendor
    lens: technical-writer
    severity: P3
    issue: "Mobile vendor settings helper text starts with 'Get notified when…' redundantly — chef-facing persona is crisp 5-12 words, not conversational"
    evidence_excerpt: "Get notified when a new order arrives / Get notified when payouts are processed / Get notified when customers leave reviews"
    recommendation: "Tighten to operational tone: 'Alert me on new orders.' / 'Alert me on payouts.' / 'Alert me on new reviews.' Or align all to: 'Push alerts for new orders' / 'Push alerts for payouts' / 'Push alerts for customer reviews'. Applies to: mv-settings-notif-neworders-helper, mv-settings-notif-payouts-helper, mv-settings-notif-reviews-helper."
    depends_on: null

  - finding_id: TW-014
    surface_id: PATTERN-ampersand-in-labels
    lens: technical-writer
    severity: P3
    issue: "Several labels and buttons use ampersand '&' instead of 'and' — style guide implies plain English; '&' is decorative chrome"
    evidence_excerpt: "Signature Dishes & Specialties / Operations & Pricing / Policies & Agreements / Order & Cancellation Policy / Hygiene & Food Safety / Verify & sign in / Respond & Send / Review & Submit"
    recommendation: "Replace '&' with 'and' except in proper noun phrases. 'Signature dishes and specialties', 'Operations and pricing', 'Policies and agreements', 'Order and cancellation policy', 'Hygiene and food safety', 'Verify and sign in', 'Respond and send', 'Review and submit'. Applies to: vp-onb-kitchen-specialties-title, vp-onb-review-section-labels, vp-onb-review-policy-labels, vp-onb-mobile-review-title, ap-auth-totp-verify-button, vp-ux-notifs-respond-form."
    depends_on: null

  - finding_id: TW-015
    surface_id: PATTERN-currency-placeholder-format
    lens: technical-writer
    severity: P2
    issue: "Catering form ships raw numeric placeholder '25000' for budget — style guide §6 mandates ₹ prefix, no space"
    evidence_excerpt: "2026-06-15 / 50 / 25000 / Mumbai / Maharashtra / Any specific requirements…"
    recommendation: "Prefix the budget placeholder: '₹25,000' (en-IN grouping). Date placeholder should be a real example formatted to locale: 'e.g. 15 Jun 2026'. Guest count: 'e.g. 50 guests'. Applies to: mc-catering-placeholders."
    depends_on: null

  - finding_id: TW-016
    surface_id: PATTERN-required-asterisk-in-placeholder
    lens: technical-writer
    severity: P2
    issue: "Required indicator merged into placeholder ('Address line 1 *') — style guide §4 form-labels rule requires required indicator on FIELD not label/placeholder"
    evidence_excerpt: "Address line 1 * / Address line 2 (optional) / City * / State * / Pincode *"
    recommendation: "Move the asterisk off the placeholder. Placeholder shows an example or short hint only ('e.g. 12 MG Road, Bandra West'). Required state is visual on the field. Applies to: mc-checkout-address-placeholders."
    depends_on: null

  - finding_id: TW-017
    surface_id: PATTERN-permission-strings-vague
    lens: technical-writer
    severity: P1
    issue: "iOS permission strings are vague — 'Used to take photos' fails App Store review guidance AND does not match style guide voice (Confident, specific)"
    evidence_excerpt: "Used to take photos"
    recommendation: "Rewrite to specify why and when: 'Take a photo of your delivery address or profile picture.' Camera prompt should describe end use, not the mechanic. Applies to: mc-perm-camera. (mc-perm-location and mc-perm-faceid are acceptable.)"
    depends_on: null

  - finding_id: TW-018
    surface_id: PATTERN-ai-slop-marketing-numerals
    lens: technical-writer
    severity: P3
    issue: "Promo helper text 'Try: FE3DR10 for 10% off' is acceptable but uses '10% off' marketing flourish; placeholder hint should follow plain-English formula"
    evidence_excerpt: "Promo code applied! 10% off / Try: FE3DR10 for 10% off"
    recommendation: "Drop the exclamation per Rule 1: 'Promo code applied. 10% off.' Try-hint OK; consider 'Try FE3DR10 — 10% off your order.' Applies to: web-ux-cart-promo."
    depends_on: null

  - finding_id: TW-019
    surface_id: PATTERN-time-relative-format
    lens: technical-writer
    severity: P3
    issue: "Vendor notifications use compact 'Nm ago / Nh ago / Nd ago' — style guide §6 mandates 'N minutes ago', 'N hours ago' under 24h, then absolute"
    evidence_excerpt: "just now / Nm ago / Nh ago / Nd ago"
    recommendation: "Expand the unit: 'just now', '{n} min ago', '{n} hr ago' (chef persona allows crisp), and beyond 24h switch to 'Yesterday' / absolute date. Don't use 'Nd ago' beyond 1 day. Applies to: vp-ux-notifs-time-relative."
    depends_on: null

  - finding_id: TW-020
    surface_id: PATTERN-driver-persona-too-long
    lens: technical-writer
    severity: P2
    issue: "Driver-facing aria labels exceed telegraphic ≤4-word target per persona matrix"
    evidence_excerpt: "Available deliveries, {n} new / Available deliveries / Driver navigation / Delivery Partner"
    recommendation: "Driver tone targets ≤4 words. 'Available deliveries, {n} new' (5 words incl. count) → 'New deliveries: {n}'. 'Driver navigation' OK. 'Delivery Partner' (role badge) is sentence case fine — 'Delivery partner'. Applies to: dp-mc-available-deliveries-alabel, dp-mc-driver-navigation, dp-mc-role-partner."
    depends_on: null

  # =============================================================================
  # SINGLE-SURFACE FINDINGS
  # =============================================================================

  - finding_id: TW-021
    surface_id: web-ux-cart-add-note
    lens: technical-writer
    severity: P3
    issue: "Item-note prefix 'Note:' is a colon-suffix label — style guide §4 form-label rule prohibits colons; for inline prefixes prefer non-colon style"
    evidence_excerpt: "Note: {item.notes}"
    recommendation: "Use italic or em-dash inline prefix instead: 'Note — {item.notes}' or simply quote the note. Apply consistently with vp-ux-orders-live-note ('Note: {specialInstructions}')."
    depends_on: null

  - finding_id: TW-022
    surface_id: web-ux-cart-add-more
    lens: technical-writer
    severity: P3
    issue: "'Add more items' is fine as a 3-word button but ambiguous on a cart screen where the back action is to the chef detail page"
    evidence_excerpt: "Add more items"
    recommendation: "Optional: 'Back to menu' is more precise on a single-chef cart. Keep 'Add more items' if cart is multi-chef. Sentence case already correct."
    depends_on: null

  - finding_id: TW-023
    surface_id: web-ux-cart-promo
    lens: technical-writer
    severity: P2
    issue: "Toast 'Promo code applied!' breaks Rule 1 (no exclamations except genuine celebration) and toast formula (past tense, ≤6 words, period)"
    evidence_excerpt: "Promo code applied! 10% off"
    recommendation: "Rewrite per success-toast formula: 'Promo applied. 10% off.' Two sentences, no exclamation, past tense."
    depends_on: null

  - finding_id: TW-024
    surface_id: web-ux-cart-delivery-info
    lens: technical-writer
    severity: P3
    issue: "'Estimated prep time: 30-45 mins' uses colon and 'mins' abbreviation — style guide §6 prefers spelled-out units customer-facing"
    evidence_excerpt: "Delivery within {n} km radius / Estimated prep time: 30-45 mins"
    recommendation: "Drop colon, expand unit: 'Prep time 30–45 minutes' (en dash). Pair: 'Delivers within {n} km'."
    depends_on: null

  - finding_id: TW-025
    surface_id: web-ux-checkout-instructions
    lens: technical-writer
    severity: P2
    issue: "Label 'Special Instructions' uses Title Case; placeholder 'Any special requests or delivery instructions…' is verbose"
    evidence_excerpt: "Special Instructions / Any special requests or delivery instructions..."
    recommendation: "Label: 'Special instructions' (sentence case). Placeholder: 'e.g. leave at door, low spice' (concrete example, ≤10 words)."
    depends_on: null

  - finding_id: TW-026
    surface_id: web-ux-chefdetail-share-aria
    lens: technical-writer
    severity: P3
    issue: "aria-label 'Share chef' is ambiguous — share what about the chef?"
    evidence_excerpt: "Share chef"
    recommendation: "More specific: 'Share this chef' or 'Share {chef.name}'."
    depends_on: null

  - finding_id: TW-027
    surface_id: web-ux-chef-card-favorite-toast-loggedout
    lens: technical-writer
    severity: P2
    issue: "Toast uses banned 'log in' verb (style guide §3: 'Sign in') and exceeds toast formula (8 words, not past tense)"
    evidence_excerpt: "Please log in to save favorites"
    recommendation: "Rewrite: 'Sign in to save favorites.' Drops the polite hedge ('Please'), 5 words, period. Applies app-wide where this pattern repeats."
    depends_on: null

  - finding_id: TW-028
    surface_id: web-ux-orderdetail-cancel-reason
    lens: technical-writer
    severity: P3
    issue: "'Cancellation reason:' uses colon — style guide §4 prohibits colons on form labels; for read-only detail rows prefer em dash or stack"
    evidence_excerpt: "Cancellation reason:"
    recommendation: "'Cancellation reason' on its own line above the value, no colon. Stack the value below."
    depends_on: null

  - finding_id: TW-029
    surface_id: web-ux-orders-search
    lens: technical-writer
    severity: P3
    issue: "Two placeholders for the same input ('Search your orders' label-like vs 'Search orders...' placeholder) — pick one"
    evidence_excerpt: "Search your orders / Search orders..."
    recommendation: "Label: 'Search orders'. Placeholder: 'Order ID, chef name, or item'. Avoid the bland 'Search orders…' placeholder."
    depends_on: null

  - finding_id: TW-030
    surface_id: web-ux-social-feed-comment
    lens: technical-writer
    severity: P3
    issue: "Generic 'Add a comment…' placeholder lacks personality and signals nothing about voice/length"
    evidence_excerpt: "Add a comment..."
    recommendation: "Customer surface — warmer: 'What did you think?' or 'Share your thoughts…' (≤4 words)."
    depends_on: null

  - finding_id: TW-031
    surface_id: web-ux-currency-aria
    lens: technical-writer
    severity: P3
    issue: "aria-label 'Change display currency' — 'display' is dev jargon for users"
    evidence_excerpt: "Change display currency"
    recommendation: "'Change currency' (2 words). The 'display' qualifier is implicit on a currency selector."
    depends_on: null

  - finding_id: TW-032
    surface_id: web-ux-chef-card-verified
    lens: technical-writer
    severity: P3
    issue: "'Verified chef' is fine but should match driver/admin variant ('Verified') for shared semantic — see app-wide consistency"
    evidence_excerpt: "Verified chef"
    recommendation: "Keep 'Verified chef' on customer card (more reassuring); but ensure the chef-self-view says 'Verified' alone to avoid first-person weirdness."
    depends_on: null

  - finding_id: TW-033
    surface_id: vp-auth-login-loading
    lens: technical-writer
    severity: P3
    issue: "'Signing in...' button loading state OK; flag only because pattern repeats across apps — cross-app consistency check"
    evidence_excerpt: "Signing in..."
    recommendation: "Standardize spelling 'Signing in…' (ellipsis char, not three dots) across vp-auth-login-loading, dp-mc-signin-loading, ap-auth-signing-in. Same applies to 'Creating account…', 'Verifying…', 'Saving…', 'Testing…', 'Preparing…', 'Uploading…'."
    depends_on: null

  - finding_id: TW-034
    surface_id: vp-auth-register-password-placeholder
    lens: technical-writer
    severity: P3
    issue: "'Min. 8 characters' uses 'Min.' abbreviation inconsistently — readable but tighten"
    evidence_excerpt: "Min. 8 characters"
    recommendation: "'At least 8 characters' (clearer). Place as helper text under label, not placeholder."
    depends_on: null

  - finding_id: TW-035
    surface_id: vp-ux-dashboard-stats-labels
    lens: technical-writer
    severity: P3
    issue: "'Today's orders / Rating / This week / All-time orders' inconsistent voice — mixed possessive vs neutral"
    evidence_excerpt: "Today's orders / Rating / This week / All-time orders / No reviews yet"
    recommendation: "Drop possessive for vendor-portal operational tone: 'Orders today', 'Average rating', 'This week', 'All-time orders'. Add unit to 'Rating' (out of 5? avg?)."
    depends_on: null

  - finding_id: TW-036
    surface_id: vp-ux-menu-form-fields
    lens: technical-writer
    severity: P2
    issue: "Placeholder 'e.g. Paneer Butter Masala' good but '250g, 1 plate' for portion size mixes two units in one field; allergen placeholder 'Type an allergen and press Enter' is unergonomic"
    evidence_excerpt: "e.g. Paneer Butter Masala / 250g, 1 plate / Type an allergen and press Enter"
    recommendation: "Portion size: split into Weight and Serves, or use single placeholder '250 g' (space before unit). Allergen entry placeholder: 'Add allergen, press Enter' (4 words)."
    depends_on: null

  - finding_id: TW-037
    surface_id: vp-ux-menu-form-dietary-tags
    lens: technical-writer
    severity: P3
    issue: "Inconsistent hyphenation: 'Non-Veg', 'Gluten-Free', 'Nut-Free', 'Dairy-Free', 'Sugar-Free' vs 'Vegan' / 'Jain' / 'Halal'"
    evidence_excerpt: "Vegetarian / Non-Veg / Eggetarian / Vegan / Jain / Gluten-Free / Halal / Kosher / Nut-Free / Dairy-Free / Sugar-Free"
    recommendation: "Sentence case the second half of hyphenated compounds: 'Non-veg', 'Gluten-free', 'Nut-free', 'Dairy-free', 'Sugar-free'. 'Vegetarian' is preferred over 'Veg' for clarity but stick with 'Non-veg' if 'Veg' is also offered. 'Eggetarian' is a regional term — consider 'Egg-friendly' or annotate with helper text."
    depends_on: null

  - finding_id: TW-038
    surface_id: vp-ux-menu-form-allergen-help
    lens: technical-writer
    severity: P3
    issue: "'Type each allergen and press Enter to add' — slightly verbose"
    evidence_excerpt: "Type each allergen and press Enter to add"
    recommendation: "Tighten: 'Type an allergen, press Enter' (5 words). Aligned with chef tone matrix."
    depends_on: null

  - finding_id: TW-039
    surface_id: vp-ux-menu-form-images
    lens: technical-writer
    severity: P3
    issue: "Image-upload status row mixes labels and instructions inconsistently — 'Primary', 'Pending', 'JPEG, PNG, or WebP. Max 5 MB each. First image is the primary display image.'"
    evidence_excerpt: "{n}/5 (min 1) / Primary / Pending / Add Image / JPEG, PNG, or WebP. Max 5 MB each. First image is the primary display image. / Upload {n} images"
    recommendation: "Split helper into two short lines: 'JPEG, PNG, or WebP. Max 5 MB each.' / 'First image becomes your primary photo.' Button 'Add Image' → 'Add image' (sentence case)."
    depends_on: null

  - finding_id: TW-040
    surface_id: vp-ux-menu-form-new-category
    lens: technical-writer
    severity: P3
    issue: "Dialog title says 'New Category' (heading) but button says 'Create' (single word, ambiguous in dialog scope)"
    evidence_excerpt: "New Category / Create New Category / Add a category to organize your menu items. / Create"
    recommendation: "Title: 'New category'. Body: 'Group your dishes — appetizers, mains, desserts, etc.' Button: 'Add category' (verb-first, specific). Sentence case throughout."
    depends_on: null

  - finding_id: TW-041
    surface_id: vp-ux-menu-search-placeholder
    lens: technical-writer
    severity: P3
    issue: "Filter default 'All Categories' Title Case (covered by TW-005 pattern)"
    evidence_excerpt: "Search menu items... / All Categories"
    recommendation: "'All categories'. Search placeholder is fine."
    depends_on: null

  - finding_id: TW-042
    surface_id: vp-ux-notifs-sla-line
    lens: technical-writer
    severity: P3
    issue: "'Typically reviewed within 24-48 hours' — operationally vague (chef wants definite timing)"
    evidence_excerpt: "Typically reviewed within 24-48 hours"
    recommendation: "Replace 'Typically' with concrete commitment: 'Reviewed within 48 hours.' (5 words). Use en dash if range required: 'Reviewed in 24–48 hours.'"
    depends_on: null

  - finding_id: TW-043
    surface_id: vp-ux-notifs-respond-form
    lens: technical-writer
    severity: P3
    issue: "Reply form placeholder verbose and contains a parenthetical example — long for chef-facing"
    evidence_excerpt: "Type your response to the admin... (e.g., I've uploaded the FSSAI license, please review) / Respond & Send"
    recommendation: "Placeholder: 'Reply to admin…'. Helper text below: 'e.g. I uploaded the FSSAI license — please review.' Button: 'Send reply' (verb-first, 2 words)."
    depends_on: null

  - finding_id: TW-044
    surface_id: vp-onb-docs-kitchen-photo-items
    lens: technical-writer
    severity: P3
    issue: "Photo labels mix hyphen styles: 'Kitchen Photo - Cooking Area' uses hyphen-dash as separator"
    evidence_excerpt: "Kitchen Photo - Cooking Area / Preparation Area / Storage / Packaging"
    recommendation: "Use en dash and sentence case: 'Kitchen photo – cooking area', 'Preparation area', 'Storage area', 'Packaging area'."
    depends_on: null

  - finding_id: TW-045
    surface_id: vp-onb-docs-upload-helper
    lens: technical-writer
    severity: P3
    issue: "'Drop file here or click to browse. Max 5 MB.' — 'click' assumes desktop pointer; touchscreen users 'tap'"
    evidence_excerpt: "Drop file here or click to browse. Max 5 MB."
    recommendation: "'Drop or select a file. Max 5 MB.' (device-neutral)."
    depends_on: null

  - finding_id: TW-046
    surface_id: vp-onb-kitchen-desc-field
    lens: technical-writer
    severity: P3
    issue: "Description placeholder 'Describe your cooking style…' OK but helper 'Min 20 characters. This is shown to customers on your profile.' is two sentences; vendor tone wants crisper"
    evidence_excerpt: "Describe your cooking style, what makes your food special, your signature touch… / Min 20 characters. This is shown to customers on your profile."
    recommendation: "Helper: 'At least 20 characters. Visible to customers.' (7 words, two short sentences)."
    depends_on: null

  - finding_id: TW-047
    surface_id: vp-onb-kitchen-experience
    lens: technical-writer
    severity: P3
    issue: "'Less than 1 year' uses Title Case for the option label — style guide says sentence case but '1 year' is numeric. Ranges 'X-Y years' should use en dash"
    evidence_excerpt: "Less than 1 year / 1-3 years / 3-5 years / 5-10 years / 10+ years"
    recommendation: "Use en dash: '1–3 years', '3–5 years', '5–10 years'. 'Less than 1 year' OK."
    depends_on: null

  - finding_id: TW-048
    surface_id: vp-onb-kitchen-meals
    lens: technical-writer
    severity: P3
    issue: "'Up to 10 meals' is inconsistent with neighbouring buckets '10-25', '25-50' (hyphen vs en dash, lower bound naming)"
    evidence_excerpt: "Up to 10 meals / 10-25 meals / 25-50 meals / 50-100 meals / 100+ meals"
    recommendation: "Use en dash: '10–25 meals', etc."
    depends_on: null

  - finding_id: TW-049
    surface_id: vp-onb-ops-prep-times
    lens: technical-writer
    severity: P3
    issue: "Ranges use hyphen not en dash; '2+ hours (pre-order only)' parenthetical inside option label is heavy"
    evidence_excerpt: "15-30 min / 30-45 min / 45-60 min / 1-2 hours / 2+ hours (pre-order only)"
    recommendation: "Use en dash: '15–30 min', etc. Separate parenthetical: option 'Over 2 hours' with helper 'Pre-order only.'"
    depends_on: null

  - finding_id: TW-050
    surface_id: vp-onb-ops-min-order
    lens: technical-writer
    severity: P3
    issue: "Helper 'Set to 0 for no minimum' uses bare 0 — currency-format consistency expects '₹0'"
    evidence_excerpt: "Minimum Order Value (Optional) / Set to 0 for no minimum"
    recommendation: "'Set to ₹0 for no minimum.' (matches §6 currency format)."
    depends_on: null

  - finding_id: TW-051
    surface_id: vp-onb-policies-payout-info
    lens: technical-writer
    severity: P3
    issue: "Deferred-setup explanation is 23 words — exceeds chef tone matrix crisp 5-12 sentence length"
    evidence_excerpt: "You can set up your bank account or UPI details for receiving payouts from Settings after your kitchen is approved."
    recommendation: "Tighten: 'Set up bank or UPI payouts in Settings once your kitchen is approved.' (12 words.) Or split into label/helper."
    depends_on: null

  - finding_id: TW-052
    surface_id: vp-onb-header-step-label
    lens: technical-writer
    severity: P3
    issue: "'Step {n} of 5' is fine; partner string 'Review your application' is a sentence-case heading but appears in stepper context as a label"
    evidence_excerpt: "Step {n} of 5 / Review your application"
    recommendation: "Stepper label: 'Review' (one word). Page heading can be 'Review your application'."
    depends_on: null

  - finding_id: TW-053
    surface_id: vp-onb-banner-admin-notes-label
    lens: technical-writer
    severity: P3
    issue: "'Admin Notes:' uses Title Case AND colon — double violation of label rules"
    evidence_excerpt: "Admin Notes:"
    recommendation: "'Admin notes' — sentence case, no colon. Stack value below."
    depends_on: null

  - finding_id: TW-054
    surface_id: vp-ux-orders-live-actions
    lens: technical-writer
    severity: P2
    issue: "'Waiting for pickup' is status not action but lives among action buttons; 'Start Preparing' (Title Case) covered by pattern"
    evidence_excerpt: "Order History / Accept / Reject / Start Preparing / Mark Ready / Waiting for pickup"
    recommendation: "Sentence-case actions: 'Order history' (nav), 'Start preparing', 'Mark ready'. Visually distinguish status 'Waiting for pickup' from action chips."
    depends_on: null

  - finding_id: TW-055
    surface_id: vp-ux-orders-live-items-total
    lens: technical-writer
    severity: P3
    issue: "'{n} items total' awkward — prefer locale-aware plural"
    evidence_excerpt: "{n} items total / Customer"
    recommendation: "'{n} items' (drop 'total'). Use ICU plural to handle '1 item' vs '2 items' per §6 (no 'item(s)')."
    depends_on: null

  - finding_id: TW-056
    surface_id: vp-ux-orders-history-ranges
    lens: technical-writer
    severity: P3
    issue: "Date-range select labels OK but inconsistent capitalization: 'Today / Last 7 days / Last 30 days / All time'"
    evidence_excerpt: "Today / Last 7 days / Last 30 days / All time"
    recommendation: "Consistent sentence case (already correct). No change required — included for audit completeness."
    depends_on: null

  - finding_id: TW-057
    surface_id: vp-ux-kitchen-payout-fields
    lens: technical-writer
    severity: P3
    issue: "Placeholders for bank/account/IFSC use 'e.g.' inconsistently — some fields have helper, others have placeholder"
    evidence_excerpt: "Bank Name / Account Number / IFSC Code / e.g. State Bank of India / Enter account number / e.g. SBIN0001234"
    recommendation: "Sentence-case labels: 'Bank name', 'Account number', 'IFSC code'. Placeholders all 'e.g. …': 'e.g. State Bank of India', 'e.g. 1234 5678 9012', 'e.g. SBIN0001234'."
    depends_on: null

  - finding_id: TW-058
    surface_id: vp-ux-profile-toggle
    lens: technical-writer
    severity: P3
    issue: "Order-acceptance switch labels 'Accepting Orders' / 'Orders Paused' use Title Case AND swap subject-verb structure"
    evidence_excerpt: "Accepting Orders / Orders Paused"
    recommendation: "Sentence case and consistent grammar: 'Accepting orders' / 'Paused'. Or 'Online' / 'Paused' (binary state, glanceable)."
    depends_on: null

  - finding_id: TW-059
    surface_id: vp-ux-profile-doc-actions
    lens: technical-writer
    severity: P3
    issue: "Doc-row inline reason 'Reason: {rejectionReason}' uses colon-suffix label"
    evidence_excerpt: "Download / Uploading... / Re-upload / Upload / Reason: {rejectionReason}"
    recommendation: "Stack: 'Rejection reason' label, value on next line, no colon. Buttons sentence case (already OK)."
    depends_on: null

  - finding_id: TW-060
    surface_id: vp-ux-reviews-subratings
    lens: technical-writer
    severity: P3
    issue: "Sub-rating chip format 'Food: {n}/5' uses colon-suffix label inside a chip — chip text should be terse"
    evidence_excerpt: "Food: {n}/5 / Value: {n}/5 / Delivery: {n}/5"
    recommendation: "Drop colon: 'Food {n}/5', 'Value {n}/5', 'Delivery {n}/5'. Or use vertical stack with label above value."
    depends_on: null

  - finding_id: TW-061
    surface_id: vp-ux-reviews-reply
    lens: technical-writer
    severity: P2
    issue: "Success toast 'Reply posted successfully' breaks toast formula (4-word past tense, ≤6 words but redundant 'successfully')"
    evidence_excerpt: "Reply posted successfully"
    recommendation: "'Reply posted.' (2 words, past tense, period). Drop redundant 'successfully'."
    depends_on: null

  - finding_id: TW-062
    surface_id: vp-ux-settings-payout-form
    lens: technical-writer
    severity: P3
    issue: "Placeholder 'Name as on bank account' is wordy"
    evidence_excerpt: "Bank Transfer / UPI / Name as on bank account / Enter account number / e.g. SBIN0001234 / yourname@upi"
    recommendation: "Label: 'Account holder name'. Placeholder: 'As printed on your bank passbook' as helper text, not placeholder. Tighten: 'Full name on bank account'."
    depends_on: null

  - finding_id: TW-063
    surface_id: vp-ux-theme-toggle
    lens: technical-writer
    severity: P3
    issue: "Theme options 'Light / System / Dark' sentence case OK; flag for consistency across mobile counterpart"
    evidence_excerpt: "Theme / Light / System / Dark"
    recommendation: "Match capitalization across web, mobile-customer, mobile-vendor, mobile-delivery. No change required if already consistent."
    depends_on: null

  - finding_id: TW-064
    surface_id: dp-mc-portal-footer
    lens: technical-writer
    severity: P3
    issue: "'Fe3dr Delivery Portal' — internal brand naming inconsistency check (Mark8ly/Fe3dr drift across project)"
    evidence_excerpt: "Fe3dr Delivery Portal"
    recommendation: "Confirm brand: 'Fe3dr' or 'Mark8ly' or 'Home Chef'. Project CLAUDE.md uses 'Home Chef' / 'Fe3dr' interchangeably — pick one. Out of TW lens scope to decide but flag the drift."
    depends_on: null

  - finding_id: TW-065
    surface_id: dp-mc-suspended-badge
    lens: technical-writer
    severity: P3
    issue: "'Suspended' single-word status OK — flagged to track legal-adjacent terms; ensure pair with 'Active'/'Inactive' lexicon coherent across apps"
    evidence_excerpt: "Suspended"
    recommendation: "Keep 'Suspended' for user/account status. Standardize sibling states 'Active' / 'Suspended' / 'Pending' (not 'Inactive' AND 'Suspended' for the same axis)."
    depends_on: null

  - finding_id: TW-066
    surface_id: dp-mc-online-badge
    lens: technical-writer
    severity: P3
    issue: "'Online' as driver status is correct but watch for collision with 'Open' (chef status) in admin views"
    evidence_excerpt: "Online"
    recommendation: "Keep. Cross-app: chef → 'Open/Closed', driver → 'Online/Offline'. Don't mix axes in admin tables."
    depends_on: null

  - finding_id: TW-067
    surface_id: dp-mc-partners-search
    lens: technical-writer
    severity: P3
    issue: "Placeholder 'Search by name, email, or vehicle...' is fine; aria-label 'Search partners' OK; flagged for consistency with admin variant"
    evidence_excerpt: "Search partners / Search by name, email, or vehicle..."
    recommendation: "Consistency: admin variant should also list explicit filterables. No change required."
    depends_on: null

  - finding_id: TW-068
    surface_id: dp-mc-stripe-cells
    lens: technical-writer
    severity: P3
    issue: "Stripe status row mixes states 'Enabled / Pending / Submitted / Incomplete' — 'Submitted' isn't a steady state, it's an action result"
    evidence_excerpt: "Charges / Payouts / Details / Enabled / Pending / Submitted / Incomplete"
    recommendation: "Use steady states: 'Enabled', 'Pending review', 'Incomplete'. 'Submitted' should be a toast on submission, not a row badge."
    depends_on: null

  - finding_id: TW-069
    surface_id: dp-mc-copy-state
    lens: technical-writer
    severity: P3
    issue: "Button toggles between 'Copy' and 'Copied' — 'Copied' should be a transient toast not a button label"
    evidence_excerpt: "Copy / Copied"
    recommendation: "Keep button as 'Copy'. Show 'Copied.' as success toast (past tense, period, ≤6 words)."
    depends_on: null

  - finding_id: TW-070
    surface_id: dp-mc-displayname-fallback
    lens: technical-writer
    severity: P3
    issue: "'Driver' as a display-name fallback works but contradicts persona matrix where customers see 'Delivery partner'"
    evidence_excerpt: "Driver"
    recommendation: "Keep 'Driver' inside driver-portal (they self-identify). Ensure customer-facing UIs say 'Your delivery partner'."
    depends_on: null

  - finding_id: TW-071
    surface_id: dp-mc-try-again
    lens: technical-writer
    severity: P2
    issue: "Error-boundary CTA 'Try Again' uses Title Case — should be sentence case"
    evidence_excerpt: "Try Again"
    recommendation: "'Try again' (sentence case). Applies also to ap-errorboundary-retry which has the same 'Try Again' string."
    depends_on: null

  - finding_id: TW-072
    surface_id: ap-approvaldetail-notes-placeholder
    lens: technical-writer
    severity: P3
    issue: "Placeholder 'Add notes for this approval action (optional)...' is verbose"
    evidence_excerpt: "Add notes for this approval action (optional)..."
    recommendation: "'Optional notes about this decision…' (5 words). Move '(optional)' off placeholder if there's a separate optional indicator."
    depends_on: null

  - finding_id: TW-073
    surface_id: ap-approvals-priorities
    lens: technical-writer
    severity: P3
    issue: "Priority badges 'urgent / high / normal / low' are lowercase — inconsistent with all other badges in admin app which are Title Case"
    evidence_excerpt: "urgent; high; normal; low"
    recommendation: "Sentence case first letter: 'Urgent', 'High', 'Normal', 'Low'. Aligns with §4 form-label sentence case applied to badges."
    depends_on: null

  - finding_id: TW-074
    surface_id: ap-approvals-search-placeholders
    lens: technical-writer
    severity: P3
    issue: "Two contextual placeholders 'Search by title or driver...' vs 'Search by title, chef, or kitchen...' inconsistent serial-comma usage"
    evidence_excerpt: "Search by title or driver... / Search by title, chef, or kitchen..."
    recommendation: "Add Oxford comma consistently: 'Search by title, driver, or kitchen…'. Aligns with style guide implicit en-IN formal register."
    depends_on: null

  - finding_id: TW-075
    surface_id: ap-auth-totp-secret-helper
    lens: technical-writer
    severity: P3
    issue: "Helper 'or enter this key' is fragment — admin tone wants precision"
    evidence_excerpt: "or enter this key"
    recommendation: "'Or enter this key manually:' as full sentence above the key value. Capital 'O', no orphan fragment."
    depends_on: null

  - finding_id: TW-076
    surface_id: ap-auth-email-placeholder
    lens: technical-writer
    severity: P3
    issue: "'admin@fe3dr.com' placeholder leaks internal brand into customer-adjacent surface (admin sign-in)"
    evidence_excerpt: "admin@fe3dr.com"
    recommendation: "Generic placeholder 'name@company.com' or 'you@example.com' for consistency with vp/dp."
    depends_on: null

  - finding_id: TW-077
    surface_id: ap-chefs-card-unnamed
    lens: technical-writer
    severity: P3
    issue: "Fallback 'Unnamed Kitchen' Title Case + emotionally cold for admin UI"
    evidence_excerpt: "Unnamed Kitchen"
    recommendation: "'Unnamed kitchen' (sentence case) or 'No kitchen name yet'."
    depends_on: null

  - finding_id: TW-078
    surface_id: ap-chefs-pagination-context
    lens: technical-writer
    severity: P3
    issue: "Pagination format 'Page {n} of {m} ({n} kitchens)' reuses '{n}' for two different values — risk of confusion"
    evidence_excerpt: "Page {n} of {m} ({n} kitchens)"
    recommendation: "Distinguish placeholders: 'Page {page} of {totalPages} · {totalCount} kitchens' (middle dot for grouping)."
    depends_on: null

  - finding_id: TW-079
    surface_id: ap-dashboard-revenue-vs
    lens: technical-writer
    severity: P3
    issue: "'vs. yesterday' / 'vs. last week' uses abbreviation 'vs.' with period — admin tone is precise; either spell out or drop period"
    evidence_excerpt: "vs. yesterday / vs. last week"
    recommendation: "Drop period: 'vs yesterday', 'vs last week' (common admin shorthand) OR spell out 'compared to yesterday'."
    depends_on: null

  - finding_id: TW-080
    surface_id: ap-dashboard-stat-users-today
    lens: technical-writer
    severity: P3
    issue: "'+{n} today' subtitle is glanceable but ambiguous about WHAT increased"
    evidence_excerpt: "+{n} today"
    recommendation: "'+{n} new users today' or attach to the stat label clearly."
    depends_on: null

  - finding_id: TW-081
    surface_id: ap-delivery-count-meta
    lens: technical-writer
    severity: P3
    issue: "'{n} total deliveries' — 'total' redundant with the count"
    evidence_excerpt: "{n} total deliveries"
    recommendation: "'{n} deliveries' (ICU plural)."
    depends_on: null

  - finding_id: TW-082
    surface_id: ap-providercreate-helpers
    lens: technical-writer
    severity: P3
    issue: "Multiple helper texts: 'Auto-generated from name. Edit to customize.' / '0 = unlimited' / 'No status mappings configured. Add mappings to translate provider statuses to Fe3dr statuses.' — last one is 16 words and exposes 'translate' jargon"
    evidence_excerpt: "Auto-generated from name. Edit to customize. / 0 = unlimited / No status mappings configured. Add mappings to translate provider statuses to Fe3dr statuses."
    recommendation: "Tighten the last helper: 'No mappings yet. Map provider statuses to Fe3dr statuses to unlock sync.' (14 words.) Replace 'translate' with 'map' (consistent vocabulary)."
    depends_on: null

  - finding_id: TW-083
    surface_id: ap-providercreate-mapping-row
    lens: technical-writer
    severity: P3
    issue: "'maps to' as inline connector is fine; ensure consistency with helper above that uses 'mappings'"
    evidence_excerpt: "maps to"
    recommendation: "Keep. Consider arrow '→' alternative for visual lightness."
    depends_on: null

  - finding_id: TW-084
    surface_id: ap-secsettings-session-actions
    lens: technical-writer
    severity: P2
    issue: "'Sign out everywhere' uses 'sign out' (correct vocab) but admin app elsewhere ships 'Logout' — same surface family, inconsistent"
    evidence_excerpt: "Sign out everywhere; Revoke session; Refresh sessions"
    recommendation: "Standardize on 'Sign out' across admin. Update ap-layout-account-logout to 'Sign out'."
    depends_on: null

  - finding_id: TW-085
    surface_id: ap-secsettings-apikey-meta
    lens: technical-writer
    severity: P3
    issue: "Key metadata strings 'no scopes; expires {date}; revoked' are lowercase fragments — visually inconsistent with neighbouring badges"
    evidence_excerpt: "no scopes; expires {date}; revoked"
    recommendation: "Sentence case first letter: 'No scopes', 'Expires {date}', 'Revoked'."
    depends_on: null

  - finding_id: TW-086
    surface_id: ap-settings-payment-test-buttons
    lens: technical-writer
    severity: P3
    issue: "State labels 'Configured / Not configured' mix sentence case correctly; loading 'Testing...' OK; pattern noted in TW-004"
    evidence_excerpt: "Test Connection; Testing...; Save Keys; Saving...; Configured; Not configured"
    recommendation: "Sentence-case buttons (covered by TW-004). 'Configured / Not configured' already correct."
    depends_on: null

  - finding_id: TW-087
    surface_id: ap-staffdetail-info-fallbacks
    lens: technical-writer
    severity: P3
    issue: "Fallback set 'Not provided / Not assigned / Never' inconsistent — some are sentences, 'Never' is a single word answer"
    evidence_excerpt: "Not provided; Not assigned; Never"
    recommendation: "Keep terse fallbacks. 'Never signed in' instead of 'Never' for clarity if context is last sign-in."
    depends_on: null

  - finding_id: TW-088
    surface_id: ap-staff-invite-placeholders
    lens: technical-writer
    severity: P3
    issue: "Email placeholder 'colleague@company.com' is good; helper 'Optional message to include in the invitation email...' is verbose"
    evidence_excerpt: "colleague@company.com; Select a role...; e.g. Operations; e.g. Operations Lead; Optional message to include in the invitation email..."
    recommendation: "Tighten: 'Optional note to include in the invitation…' (6 words). Drop 'email' (implied)."
    depends_on: null

  - finding_id: TW-089
    surface_id: ap-users-pagination
    lens: technical-writer
    severity: P3
    issue: "'Previous' / 'Next' OK; 'Page {n} of {m}' fine but inconsistent with ap-auditlogs-pagination ('· {n} total') and ap-chefs-pagination-context"
    evidence_excerpt: "Page {n} of {m}; Previous; Next"
    recommendation: "Standardize one pagination template across admin: 'Page {n} of {m} · {total} {entity}'."
    depends_on: null

  - finding_id: TW-090
    surface_id: ap-platsettings-hours-defaults
    lens: technical-writer
    severity: P3
    issue: "Helper 'No days selected = every day.' uses '=' symbol — admin tone is precise but '=' reads as code"
    evidence_excerpt: "Asia/Kolkata; 09:00; 23:00; We're currently closed.; No days selected = every day."
    recommendation: "Replace '=' with 'means': 'No days selected means every day.' Aligns with §5 plain-language tone applied to admin chrome."
    depends_on: null

  - finding_id: TW-091
    surface_id: mc-perm-faceid
    lens: technical-writer
    severity: P3
    issue: "Face ID prompt 'Use Face ID to log in quickly' uses banned 'log in' verb"
    evidence_excerpt: "Use Face ID to log in quickly"
    recommendation: "'Sign in quickly with Face ID.' Plain English, 6 words, sign-in vocab."
    depends_on: null

  - finding_id: TW-092
    surface_id: mc-perm-location
    lens: technical-writer
    severity: P3
    issue: "Location prompt 'Used to show your location on the delivery tracking map' is 11 words and starts passively"
    evidence_excerpt: "Used to show your location on the delivery tracking map"
    recommendation: "'Show your live location on the delivery map.' (9 words, active voice)."
    depends_on: null

  - finding_id: TW-093
    surface_id: mc-favorites-subtitle
    lens: technical-writer
    severity: P3
    issue: "Subtitle '{n}/{max} saved' is glanceable but ambiguous — what's saved?"
    evidence_excerpt: "{n}/{max} saved"
    recommendation: "'{n} of {max} chefs saved' or 'Saved {n}/{max}'."
    depends_on: null

  - finding_id: TW-094
    surface_id: mc-home-search-placeholder
    lens: technical-writer
    severity: P3
    issue: "Placeholder 'Search chefs, cuisines...' uses comma list with trailing ellipsis — readable but tighten"
    evidence_excerpt: "Search chefs, cuisines..."
    recommendation: "Customer surface — warmer, fewer words: 'Find a chef or cuisine'."
    depends_on: null

  - finding_id: TW-095
    surface_id: mc-profile-placeholders
    lens: technical-writer
    severity: P3
    issue: "Profile placeholders 'First name / Last name / +91 9876543210' OK; ensure phone format matches §6 ('+91 98765 43210' with grouping)"
    evidence_excerpt: "First name / Last name / +91 9876543210"
    recommendation: "Phone placeholder: '+91 98765 43210' (en-IN groups per §6)."
    depends_on: null

  - finding_id: TW-096
    surface_id: mc-catering-budget-display
    lens: technical-writer
    severity: P3
    issue: "'Budget: ₹{n}' uses colon-suffix label inline — see TW-021 colon pattern"
    evidence_excerpt: "Budget: ₹{n}"
    recommendation: "'Budget ₹{n}' (no colon) or stack label above value. Aligns with §4."
    depends_on: null

  - finding_id: TW-097
    surface_id: mc-catering-view-quote-hint
    lens: technical-writer
    severity: P3
    issue: "'Quotes available — view details' uses em dash separator — good; flag for verb-first check"
    evidence_excerpt: "Quotes available — view details"
    recommendation: "Keep. Alternative: 'View {n} quotes' (verb-first, count-explicit)."
    depends_on: null

  - finding_id: TW-098
    surface_id: mc-checkout-note
    lens: technical-writer
    severity: P3
    issue: "Label 'Note to chef (optional)' merges optional indicator into label — style guide §4 says required state is asterisk on field, leave label clean"
    evidence_excerpt: "Note to chef (optional) / Any special instructions..."
    recommendation: "Label: 'Note to chef'. Helper: 'Optional — any special instructions.' Separate optional marker from label."
    depends_on: null

  - finding_id: TW-099
    surface_id: mc-chef-delivery-meta
    lens: technical-writer
    severity: P3
    issue: "'Free delivery' / '₹{n} delivery' format inconsistent — one is text, one is currency-first"
    evidence_excerpt: "Free delivery / ₹{n} delivery"
    recommendation: "Match grammar: 'Free delivery' / 'Delivery ₹{n}' OR keep '₹{n} delivery' and pair as 'Delivery free'. Consistent left-anchor for scanability."
    depends_on: null

  - finding_id: TW-100
    surface_id: mc-order-detail-eta
    lens: technical-writer
    severity: P3
    issue: "'ETA: {time}' uses acronym + colon — customer surface (warmer) should expand"
    evidence_excerpt: "ETA: {time}"
    recommendation: "'Arriving by {time}' or 'Estimated arrival {time}'. Customer tone matrix wants conversational over telegraphic."
    depends_on: null

  - finding_id: TW-101
    surface_id: mc-order-detail-footer
    lens: technical-writer
    severity: P3
    issue: "'Ordered on {date}' OK — flag for consistency with admin/vendor variants"
    evidence_excerpt: "Ordered on {date}"
    recommendation: "Keep. Ensure date format matches §6 ('Fri, 14 May, 7:30 PM' for future, relative under 24h)."
    depends_on: null

  - finding_id: TW-102
    surface_id: mc-timeline-eta
    lens: technical-writer
    severity: P3
    issue: "'Est. arrival: {time}' uses abbreviation 'Est.' AND colon — customer surface should be warmer"
    evidence_excerpt: "Est. arrival: {time}"
    recommendation: "'Arriving around {time}' (9 chars, sets expectation). Plain English."
    depends_on: null

  - finding_id: TW-103
    surface_id: mc-deliverymap-markers
    lens: technical-writer
    severity: P3
    issue: "Map marker titles 'Delivery Address / Chef Location / Driver' use Title Case — customer surface should be sentence case"
    evidence_excerpt: "Delivery Address / Chef Location / Driver"
    recommendation: "'Delivery address', 'Chef location', 'Driver'. Sentence case."
    depends_on: null

  - finding_id: TW-104
    surface_id: mv-onb-personal-email-helper
    lens: technical-writer
    severity: P3
    issue: "'Email is pre-filled from your account' is fine — 6 words, chef-tone OK"
    evidence_excerpt: "Email is pre-filled from your account"
    recommendation: "Keep. Optional tighten: 'Pre-filled from your account.' (5 words)."
    depends_on: null

  - finding_id: TW-105
    surface_id: mv-push-channel-neworders
    lens: technical-writer
    severity: P3
    issue: "Android notification channel 'New Orders' Title Case appears in OS settings — match Android system convention but also style guide; OS settings show as-is so consistent app voice matters"
    evidence_excerpt: "New Orders"
    recommendation: "'New orders' (sentence case) for app consistency. OS will display as provided."
    depends_on: null

  - finding_id: TW-106
    surface_id: mv-push-channel-orderupdates
    lens: technical-writer
    severity: P3
    issue: "Android channel 'Order Updates' Title Case (same as TW-105)"
    evidence_excerpt: "Order Updates"
    recommendation: "'Order updates'. Same pattern."
    depends_on: null

  - finding_id: TW-107
    surface_id: mv-menuedit-price-change-banner
    lens: technical-writer
    severity: P2
    issue: "Inline warning banner 'Price changes are submitted for admin review and may take 24 hours to reflect.' is 14 words AND passive voice — vendor crisp 5-12 sentence-length rule exceeded"
    evidence_excerpt: "Price changes are submitted for admin review and may take 24 hours to reflect."
    recommendation: "Two short sentences, active voice: 'Price changes need admin approval. They go live within 24 hours.' (12 words across two sentences.) Or callout style: 'Heads up — price changes take up to 24 hours to apply.'"
    depends_on: null

  - finding_id: TW-108
    surface_id: mv-settings-accepting-helper
    lens: technical-writer
    severity: P3
    issue: "Helper 'Toggle to start or pause accepting orders' is awkward — toggle is mechanic not user intent"
    evidence_excerpt: "Toggle to start or pause accepting orders"
    recommendation: "'Start or pause new orders.' (5 words, active.) Drop 'Toggle' — implied by switch."
    depends_on: null

  - finding_id: TW-109
    surface_id: md-mic-003
    lens: technical-writer
    severity: P2
    issue: "Driver Android notification channel 'New Deliveries' marked P0 SAFETY in inventory note (visible in OS settings) but copy is Title Case — drivers act on this in distracted contexts"
    evidence_excerpt: "New Deliveries"
    recommendation: "'New deliveries' (sentence case for app voice). Note: OS settings show this verbatim, so brand voice carries. Inventory note flags safety; suggest pairing with a clear sub-description in the channel config."
    depends_on: null

  - finding_id: TW-110
    surface_id: md-mic-004
    lens: technical-writer
    severity: P3
    issue: "'Delivery Updates' channel name Title Case (paired with TW-109)"
    evidence_excerpt: "Delivery Updates"
    recommendation: "'Delivery updates'."
    depends_on: null

  - finding_id: TW-111
    surface_id: md-mic-002
    lens: technical-writer
    severity: P3
    issue: "Driver back button aria 'Go back' OK — telegraphic and clear; flag for cross-app consistency check"
    evidence_excerpt: "Go back"
    recommendation: "Keep 'Go back' in driver context (telegraphic). Web/vendor use 'Back to {context}' which is also fine for those personas."
    depends_on: null

  - finding_id: TW-112
    surface_id: md-mic-001
    lens: technical-writer
    severity: P3
    issue: "Driver Alert buttons 'Cancel / Logout' — same banned 'Logout' (covered by TW-007 pattern)"
    evidence_excerpt: "Cancel / Logout"
    recommendation: "Pair: 'Cancel / Sign out'. Sentence case both."
    depends_on: null

  - finding_id: TW-113
    surface_id: web-ux-a11y-skip-link
    lens: technical-writer
    severity: P3
    issue: "'Skip to main content' is the WCAG-standard label; flag for cross-app standardization confirmation"
    evidence_excerpt: "Skip to main content"
    recommendation: "Keep. Verified standard. Same label across web/dp."
    depends_on: null

  - finding_id: TW-114
    surface_id: vp-ux-layout-aria
    lens: technical-writer
    severity: P3
    issue: "aria-label set 'Open navigation / Close navigation / Notifications, {n} unread / Notifications / Open user menu / Close user menu / Main navigation' — consistent sentence case. Cross-app match against admin (TW-005)."
    evidence_excerpt: "Open navigation / Close navigation / Notifications, {n} unread / Notifications / Open user menu / Close user menu / Main navigation"
    recommendation: "Keep. Mirror to admin app where Title Case currently differs."
    depends_on: null

  - finding_id: TW-115
    surface_id: vp-onb-nav-continue
    lens: technical-writer
    severity: P3
    issue: "Onboarding nav 'Back' / 'Continue' — single word, verb-first, sentence case — all correct. Flag for consistency with mobile-vendor onboarding."
    evidence_excerpt: "Back / Continue"
    recommendation: "Keep. Standard wizard nav."
    depends_on: null

  - finding_id: TW-116
    surface_id: ap-providercreate-submit
    lens: technical-writer
    severity: P2
    issue: "Button set 'Create Provider; Cancel; Add Mapping; Add' uses Title Case (covered by pattern TW-004) AND has a bare 'Add' button which is ambiguous"
    evidence_excerpt: "Create Provider; Cancel; Add Mapping; Add"
    recommendation: "'Create provider'; 'Add mapping'; never bare 'Add' — be specific. Sentence case throughout."
    depends_on: null

  - finding_id: TW-117
    surface_id: ap-platsettings-tooltips
    lens: technical-writer
    severity: P3
    issue: "Delete tooltip 'Delete / Delete zone / Toggle {day}' mixes scope — 'Delete' is bare and ambiguous"
    evidence_excerpt: "Delete; Delete zone; Toggle {day}"
    recommendation: "Always specify target: 'Delete {item}', 'Delete zone', 'Toggle {day}'."
    depends_on: null

  - finding_id: TW-118
    surface_id: ap-orders-action-aria
    lens: technical-writer
    severity: P3
    issue: "'Actions for order {n}' OK; ensure {n} is the order ID format from §6 ('#HC-2026-00001234')"
    evidence_excerpt: "Actions for order {n}"
    recommendation: "Confirm format substitution: 'Actions for order #HC-2026-00001234'. Aligns with §6 order-ID format."
    depends_on: null

  - finding_id: TW-119
    surface_id: ap-auditlogs-pagination
    lens: technical-writer
    severity: P3
    issue: "'Page {n} of {m} · {n} total' reuses {n} ambiguously (same as TW-078)"
    evidence_excerpt: "Page {n} of {m} · {n} total"
    recommendation: "Use distinct placeholder names: 'Page {page} of {totalPages} · {totalCount} total'."
    depends_on: null

  - finding_id: TW-120
    surface_id: vp-onb-stepper-review-step
    lens: technical-writer
    severity: P3
    issue: "Stepper label 'Submit application' OK (sentence case, 2 words, verb-first) — flag for confirming pattern adopted"
    evidence_excerpt: "Review / Submit application"
    recommendation: "Keep. This is the target format the rest of the vendor-portal onboarding should match."
    depends_on: null
```

## Legal findings

```yaml
---
# Legal lens findings — microcopy slice
# Inventory rows audited: 281
# All findings flagged depends_on: "needs lawyer review"
# Lens scope: India jurisdiction (DPDP 2023, FSSAI, RBI PA, GST, CP Rules) + best-practice

findings:

  - finding_id: LEG-001
    surface_id: web-ux-chef-card-verified
    lens: legal
    severity: P1
    issue: '"Verified chef" badge — unqualified verification claim with no tooltip/disclosure of what is verified'
    evidence_excerpt: '"Verified chef" (aria-label + title)'
    recommendation: 'Qualify the claim (e.g., specify whether FSSAI license, identity, or kitchen photos verified) or add tooltip/help link. Unqualified "Verified" creates implied warranty exposure under CP Rules 2020 and FSSAI Display Regulations.'
    citation: 'Consumer Protection Rules 2020 (misleading representation); FSSAI Labelling & Display Regs 2020'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-002
    surface_id: ap-chefs-badge-verified
    lens: legal
    severity: P1
    issue: 'Admin "Verified" badge — same unqualified claim surfaced on admin chefs list with no scope definition'
    evidence_excerpt: '"Verified"'
    recommendation: 'Define what verification covers (FSSAI / identity / address / kitchen). Admin UI should link to verification criteria.'
    citation: 'best-practice; FSSAI §31'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-003
    surface_id: ap-chefs-action-verify-aria
    lens: legal
    severity: P1
    issue: 'Single-click "Verify kitchen" action with no on-screen attestation of what admin is verifying'
    evidence_excerpt: '"Verify kitchen / Verify Kitchen"'
    recommendation: 'Require admin to tick attestation checklist (FSSAI checked, address checked, kitchen photos checked) before allowing verify. Single-button verify exposes platform to liability claims if a downstream incident occurs.'
    citation: 'IT Rules 2021 §3 (due diligence); CP Rules 2020'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-004
    surface_id: ap-chefs-action-reject-aria
    lens: legal
    severity: P1
    issue: 'Single-click "Reject application" with no required reason code — chef cannot dispute rejection without recorded reason'
    evidence_excerpt: '"Reject application / Reject Application"'
    recommendation: 'Require reason code + free-text justification before allowing reject. Document retention obligation under DPDP §8(5) (purpose specification + accuracy) requires recorded decision basis.'
    citation: 'DPDP Act §8(5); CP Rules grievance redressal'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-005
    surface_id: ap-chefs-action-suspend-aria
    lens: legal
    severity: P0
    issue: 'Suspend action exposed as aria/tooltip only — no consequence disclosure for chef who is being deprived of earnings'
    evidence_excerpt: '"Suspend kitchen / Suspend Kitchen"'
    recommendation: 'Require confirmation modal explaining (a) earnings impact, (b) pending order handling, (c) appeal pathway, (d) reason code. Suspension without due-process disclosure raises CP Rules and natural-justice exposure.'
    citation: 'CP Rules 2020; IT Rules 2021 §3(1)(b)'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-006
    surface_id: ap-users-action-suspend-aria
    lens: legal
    severity: P0
    issue: 'Suspend user — aria-label only with no consequence disclosure for affected user account'
    evidence_excerpt: '"Suspend user / Suspend User"'
    recommendation: 'Confirmation modal must explain (a) what user loses access to, (b) data retention during suspension, (c) reactivation path, (d) reason. DPDP §11 requires fair process for restricting data principal rights.'
    citation: 'DPDP Act §11; IT Rules 2021'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-007
    surface_id: ap-staffdetail-actions
    lens: legal
    severity: P1
    issue: 'Deactivate/Reactivate staff buttons — no audit/reason capture surfaced in UI labels'
    evidence_excerpt: '"Deactivate; Reactivate; Change Role; Confirm Deactivation; Confirm Reactivation"'
    recommendation: 'Confirm modal must capture reason for audit log. DPDP requires processing record for personnel data changes.'
    citation: 'DPDP Act §8(4); best-practice'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-008
    surface_id: ap-approvaldetail-actions
    lens: legal
    severity: P1
    issue: 'Approve/Reject/Request Info buttons — notes field marked "(optional)" but rejection without reason is legally weak'
    evidence_excerpt: '"Approve; Reject; Request Info" + "Add notes for this approval action (optional)..."'
    recommendation: 'Reject action should require notes (not optional). Approval decisions without recorded basis fail FSSAI inspection and CP Rules grievance defence.'
    citation: 'FSSAI §31; CP Rules grievance officer regime'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-009
    surface_id: vp-ux-menu-form-dietary-tags
    lens: legal
    severity: P0
    issue: 'Dietary tags include religious certification claims (Halal, Kosher, Jain) with no certification-required disclosure or verification mechanism in label'
    evidence_excerpt: '"Vegetarian / Non-Veg / Eggetarian / Vegan / Jain / Gluten-Free / Halal / Kosher / Nut-Free / Dairy-Free / Sugar-Free"'
    recommendation: 'Religious / certification claims (Halal, Kosher, Jain, Gluten-Free) require third-party certification under FSSAI Labelling Regs 2020 and Halal Certification Order. Add helper text warning chefs that claim must be backed by certification, or restrict to self-attestation with disclaimer.'
    citation: 'FSSAI Food Safety and Standards (Labelling and Display) Regs 2020; Halal Certification (India) directives'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-010
    surface_id: vp-ux-menu-form-fields
    lens: legal
    severity: P0
    issue: 'Allergens field is free-text with helper "Type each allergen and press Enter" — no standard allergen list, no required-allergen-disclosure prompt'
    evidence_excerpt: '"Allergens" / "Type an allergen and press Enter"'
    recommendation: 'FSSAI requires disclosure of 8 major allergens (milk, eggs, fish, crustacean shellfish, tree nuts, peanuts, wheat, soy). UI should present these as required checkboxes plus free-text additions, not free-text only.'
    citation: 'FSSAI Labelling Regs 2020 §2.2.2(e) (major allergens)'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-011
    surface_id: vp-ux-menu-form-allergen-help
    lens: legal
    severity: P1
    issue: 'Allergen helper text gives no warning about legal duty to declare; chefs may skip field entirely'
    evidence_excerpt: '"Type each allergen and press Enter to add"'
    recommendation: 'Add helper text: "Declaring allergens is required by FSSAI. List anything in this dish that can trigger reactions." Make field required for dishes containing common allergens.'
    citation: 'FSSAI Labelling Regs 2020'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-012
    surface_id: web-ux-checkout-instructions
    lens: legal
    severity: P1
    issue: 'Customer "Special Instructions" textarea — no allergen-aware prompt; customer cannot easily flag allergy to chef'
    evidence_excerpt: '"Special Instructions / Any special requests or delivery instructions..."'
    recommendation: 'Either add dedicated allergy field or update placeholder to prompt allergy disclosure (e.g., "Note any allergies, dietary needs, or delivery instructions"). FSSAI / CP Rules pathway for allergic-reaction complaints is weakened when allergy data is buried.'
    citation: 'FSSAI Labelling Regs; CP Rules grievance redressal'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-013
    surface_id: mc-checkout-note
    lens: legal
    severity: P1
    issue: 'Mobile checkout "Note to chef" — same gap as web; no allergy-specific field'
    evidence_excerpt: '"Note to chef (optional) / Any special instructions..."'
    recommendation: 'Mirror web fix — explicit allergy field or allergy-aware placeholder.'
    citation: 'FSSAI Labelling Regs; CP Rules'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-014
    surface_id: vp-ux-kitchen-payout-fields
    lens: legal
    severity: P0
    issue: 'Bank Name / Account Number / IFSC Code fields have no DPDP purpose/retention notice in helper text'
    evidence_excerpt: '"Bank Name / Account Number / IFSC Code / e.g. State Bank of India / Enter account number / e.g. SBIN0001234"'
    recommendation: 'Add purpose-limitation helper near these fields: "We use these details only to pay out your earnings. Stored encrypted. See Privacy Policy." DPDP §6 requires purpose notice at point of collection for financial data (sensitive class).'
    citation: 'DPDP Act §5, §6 (notice and purpose limitation); RBI PA Master Direction'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-015
    surface_id: vp-ux-settings-payout-form
    lens: legal
    severity: P0
    issue: 'Bank Transfer / UPI payout form — same DPDP notice gap, also no statement that UPI VPA is collected for payouts only'
    evidence_excerpt: '"Bank Transfer / UPI / Name as on bank account / Enter account number / e.g. SBIN0001234 / yourname@upi"'
    recommendation: 'Add DPDP notice + retention statement at collection. Specify that UPI VPA is used only for payouts and not shared with customers.'
    citation: 'DPDP Act §5, §6; RBI PA Master Direction'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-016
    surface_id: vp-onb-policies-payout-info
    lens: legal
    severity: P1
    issue: 'Deferred payout setup explanation does not mention KYC requirement, expected timelines, or RBI PA settlement rules'
    evidence_excerpt: '"You can set up your bank account or UPI details for receiving payouts from Settings after your kitchen is approved."'
    recommendation: 'Mention (a) KYC will be required to set up payouts, (b) typical settlement timeline (e.g., T+N days per RBI PA Master Direction), (c) link to payout T&C.'
    citation: 'RBI PA Master Direction 2020 (settlement timelines, KYC)'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-017
    surface_id: dp-mc-stripe-cells
    lens: legal
    severity: P1
    issue: 'Driver Stripe "Charges / Payouts / Details / Enabled / Pending / Submitted / Incomplete" — terse state labels with no KYC/onboarding-status explanation visible to driver'
    evidence_excerpt: '"Charges / Payouts / Details + Enabled / Pending / Submitted / Incomplete"'
    recommendation: 'Add helper text per state explaining what driver must do next (KYC pending, document required, etc.). DPDP §5 requires notice + Stripe Connect onboarding triggers cross-border data transfer disclosure.'
    citation: 'DPDP Act §5, §16 (cross-border transfer); RBI PA Master Direction'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-018
    surface_id: dp-mc-action-required
    lens: legal
    severity: P1
    issue: '"Action Required" payout badge — does not state what action, what regulatory requirement, or what happens to in-flight payouts'
    evidence_excerpt: '"Action Required"'
    recommendation: 'Explicit copy: "Action required: complete KYC to receive payouts" + linked CTA. Implicit requirements fail RBI PA transparency expectation.'
    citation: 'RBI PA Master Direction; CP Rules'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-019
    surface_id: dp-mc-connected
    lens: legal
    severity: P2
    issue: '"Connected" status (Stripe) — no disclosure that connecting to a foreign payment processor entails cross-border data transfer'
    evidence_excerpt: '"Connected"'
    recommendation: 'Helper or tooltip noting Stripe data flow + link to Privacy Policy section on third-party processors. DPDP §16 requires notice for cross-border transfer.'
    citation: 'DPDP Act §16'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-020
    surface_id: mc-perm-camera
    lens: legal
    severity: P0
    issue: 'iOS NSCameraUsageDescription "Used to take photos" — vague purpose violates Apple guidelines AND DPDP §6 purpose limitation'
    evidence_excerpt: '"Used to take photos"'
    recommendation: 'State specific purpose: e.g., "Take photos of food and order issues to share with chefs and support." DPDP §6 requires specific, lawful purpose statement at collection.'
    citation: 'DPDP Act §6; Apple App Store Review Guideline 5.1.1'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-021
    surface_id: mc-perm-faceid
    lens: legal
    severity: P1
    issue: 'iOS NSFaceIDUsageDescription "Use Face ID to log in quickly" — does not state biometric handling (on-device, not transmitted)'
    evidence_excerpt: '"Use Face ID to log in quickly"'
    recommendation: 'Clarify: "Use Face ID to sign in. Your biometric data stays on your device and is never sent to our servers." Biometric data is sensitive class under DPDP; transparent collection notice required.'
    citation: 'DPDP Act §2(t) (sensitive personal data definition), §6'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-022
    surface_id: mc-perm-location
    lens: legal
    severity: P0
    issue: 'iOS location permission "Used to show your location on the delivery tracking map" — acceptable purpose but missing retention/sharing statement'
    evidence_excerpt: '"Used to show your location on the delivery tracking map"'
    recommendation: 'Expand: "Used to show your location during delivery tracking. Location is shared with your driver only and not retained after delivery." DPDP §5/§6 + Apple guideline.'
    citation: 'DPDP Act §5, §6; Apple App Store Guideline 5.1.1'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-023
    surface_id: vp-onb-personal-address-fields
    lens: legal
    severity: P1
    issue: 'Address fields (Address Line 1/2, Landmark, Country, State, City, PIN) collected without on-screen DPDP notice of purpose'
    evidence_excerpt: '"Address Line 1 / Address Line 2 / Landmark / Country / State / City / PIN Code (+ placeholders)"'
    recommendation: 'Add helper text or section preamble explaining address use (kitchen verification, customer delivery range, FSSAI license matching). DPDP §6 purpose notice.'
    citation: 'DPDP Act §6'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-024
    surface_id: vp-onb-personal-avatar-label
    lens: legal
    severity: P2
    issue: 'Profile photo helper "Shown on your kitchen page" — does not disclose photo retention if account is later closed'
    evidence_excerpt: '"Profile Photo (Optional) / Shown on your kitchen page. JPEG, PNG, or WebP. Max 5 MB."'
    recommendation: 'Add retention statement or link to Privacy Policy section on closure/deletion. DPDP §8 retention duty.'
    citation: 'DPDP Act §8(7) (retention/erasure)'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-025
    surface_id: vp-onb-docs-kitchen-photos-title
    lens: legal
    severity: P1
    issue: 'Kitchen photos collected for "trust" without stating who can see them, retention, or what happens if rejected'
    evidence_excerpt: '"Photos of your kitchen help build trust. At least one photo of your cooking area is required."'
    recommendation: 'Add helper text on (a) who views these (admin team for verification + on profile), (b) retention if application rejected, (c) DPDP rights.'
    citation: 'DPDP Act §5, §6, §8'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-026
    surface_id: vp-onb-docs-additional-title
    lens: legal
    severity: P1
    issue: '"Additional Documents (Optional) … help speed up verification and build customer trust" — implies that not submitting harms approval, which makes consent non-free'
    evidence_excerpt: '"These are optional but help speed up verification and build customer trust."'
    recommendation: 'Either make truly optional with no implied penalty, or list documents as required with explicit purpose. DPDP §7(3) requires free consent (not coerced by implied disadvantage).'
    citation: 'DPDP Act §7(3) (free, specific, informed consent)'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-027
    surface_id: vp-onb-docs-upload-helper
    lens: legal
    severity: P2
    issue: 'Document upload helper "Drop file here or click to browse. Max 5 MB." — no statement on document handling, encryption, or who reviews'
    evidence_excerpt: '"Drop file here or click to browse. Max 5 MB."'
    recommendation: 'Add DPDP-aware helper noting documents are stored securely and reviewed by admin. Documents commonly include PAN/Aadhaar/FSSAI — sensitive class.'
    citation: 'DPDP Act §5, §8(4) (reasonable security)'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-028
    surface_id: vp-onb-docs-required-label
    lens: legal
    severity: P3
    issue: '"Required" badge without statement of the underlying legal requirement (FSSAI license, PAN, etc.)'
    evidence_excerpt: '"Required"'
    recommendation: 'Tooltip or helper line citing FSSAI / Income Tax Act / KYC source for each required document. Helps consent be "informed" per DPDP.'
    citation: 'DPDP Act §6 (informed); FSSAI registration rules'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-029
    surface_id: mc-catering-placeholders
    lens: legal
    severity: P2
    issue: 'Catering request placeholders include city/state/budget/dietary text — no DPDP notice on use of dietary information (sensitive when religion-linked)'
    evidence_excerpt: '"2026-06-15 / 50 / 25000 / Mumbai / Maharashtra / Any specific requirements, dietary restrictions, menu preferences..."'
    recommendation: 'Add notice that dietary restrictions are shared only with chef bidder. Dietary info can imply religion/health under DPDP sensitive class.'
    citation: 'DPDP Act §2(t), §6'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-030
    surface_id: web-ux-cart-promo
    lens: legal
    severity: P1
    issue: 'Promo code UI "Promo code applied! 10% off / Try: FE3DR10 for 10% off" — no link to promo T&C / validity / capping rules'
    evidence_excerpt: '"Promo code / Apply / Applied / Promo code applied! 10% off / Try: FE3DR10 for 10% off"'
    recommendation: 'Add "View promo terms" link near promo code field. CP Rules 2020 prohibit misleading promotions; unstated terms (cap, expiry, eligible items) expose platform.'
    citation: 'CP Rules 2020 §4 (Misleading Advertisement); CCPA guidelines'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-031
    surface_id: vp-ux-menu-bulk-confirm
    lens: legal
    severity: P2
    issue: 'Bulk delete confirm "Delete {n} items? This cannot be undone." — vague consequence; does not state effect on existing orders containing those items'
    evidence_excerpt: '"Delete {n} items? This cannot be undone."'
    recommendation: 'Per style guide modal subtitle rule + CP Rules: state actual consequence (e.g., "Deleting removes these items from your menu. In-flight orders are unaffected.").'
    citation: 'Style guide §4 (modal subtitles); CP Rules 2020 (clarity)'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-032
    surface_id: vp-ux-menu-delete-confirm
    lens: legal
    severity: P2
    issue: 'Generic "Are you sure you want to delete this item?" — vague consequence pattern banned by style guide and weak under consumer-rules transparency'
    evidence_excerpt: '"Are you sure you want to delete this item?"'
    recommendation: 'Replace with consequence-specific copy (e.g., "Deleting removes this item from your menu. Existing orders containing it are not affected.").'
    citation: 'Style guide §4; CP Rules 2020'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-033
    surface_id: ap-secsettings-session-actions
    lens: legal
    severity: P1
    issue: '"Sign out everywhere / Revoke session" actions without confirmation copy explaining downstream impact (in-flight admin actions, audit logging)'
    evidence_excerpt: '"Sign out everywhere; Revoke session; Refresh sessions"'
    recommendation: 'Confirmation modal explaining all active admin sessions will be terminated; audit-logged action. DPDP §8(4) reasonable security expectation requires informed admin self-service.'
    citation: 'DPDP Act §8(4); best-practice'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-034
    surface_id: ap-secsettings-apikey-meta
    lens: legal
    severity: P2
    issue: 'API key metadata "no scopes; expires {date}; revoked" — minimal disclosure of what an unscoped key can access'
    evidence_excerpt: '"no scopes; expires {date}; revoked"'
    recommendation: '"no scopes" warning should be loud: "Warning — this key has full access." API key with unspecified scope is a DPDP §8(4) security risk; surface it.'
    citation: 'DPDP Act §8(4); best-practice'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-035
    surface_id: ap-exports-button
    lens: legal
    severity: P1
    issue: 'Data Exports "Download CSV; Preparing..." — no surface disclosure of what export contains, retention of generated file, or DPDP data principal rights handling'
    evidence_excerpt: '"Download CSV; Preparing..."'
    recommendation: 'Helper text near export buttons: "Exports may contain personal data. Handle per Privacy Policy and DPDP obligations. File available for 24h." DPDP §11 (data principal rights) + §8(4).'
    citation: 'DPDP Act §11 (rights), §8(4) (security)'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-036
    surface_id: ap-auditlogs-expanded
    lens: legal
    severity: P2
    issue: 'Audit log expanded view "User agent: / Before / After" — no statement of retention policy or accessibility for data principal grievance officer use'
    evidence_excerpt: '"User agent:; Before; After"'
    recommendation: 'Footer or header note: "Audit logs retained N years per DPDP grievance/regulatory requirements." DPDP §8(7) retention duty.'
    citation: 'DPDP Act §8(7); CP Rules grievance officer regime'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-037
    surface_id: mv-menuedit-price-change-banner
    lens: legal
    severity: P1
    issue: 'Price-change banner "Price changes are submitted for admin review and may take 24 hours to reflect." — does not disclose what happens to in-flight orders or that price is contractually fixed at order placement'
    evidence_excerpt: '"Price changes are submitted for admin review and may take 24 hours to reflect."'
    recommendation: 'Add: "Orders placed before the change are honoured at the old price." CP Rules 2020 + Sale of Goods Act fix price at acceptance — banner should disclose to chef.'
    citation: 'CP Rules 2020; Sale of Goods Act'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-038
    surface_id: dp-mc-uploaded-state
    lens: legal
    severity: P2
    issue: 'Driver document state shows "Verified / Uploaded" without distinction of who verified or against what standard'
    evidence_excerpt: '"Verified / Uploaded"'
    recommendation: 'Tooltip clarifying verification scope (e.g., "Verified by Fe3dr team against {document type} standard"). Unqualified "Verified" creates same exposure as LEG-001.'
    citation: 'CP Rules 2020; best-practice'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-039
    surface_id: dp-mc-pending-verification
    lens: legal
    severity: P2
    issue: '"Pending Verification" badge — no SLA, no statement of what cannot be done while pending, no DPDP retention notice for unverified documents'
    evidence_excerpt: '"Pending Verification"'
    recommendation: 'Add SLA expectation + impact on driver earnings + retention statement if rejected.'
    citation: 'DPDP Act §8(7); best-practice'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-040
    surface_id: vp-ux-notifs-respond-form
    lens: legal
    severity: P2
    issue: 'Notification response placeholder example "I''ve uploaded the FSSAI license, please review" — fine to suggest, but no statement that admin reply may be audit-logged / retained'
    evidence_excerpt: '"Type your response to the admin... (e.g., I''ve uploaded the FSSAI license, please review) / Respond & Send"'
    recommendation: 'Helper text: "Your message is logged for our records." DPDP §8(4) audit + transparency.'
    citation: 'DPDP Act §8(4)'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-041
    surface_id: vp-ux-notifs-sla-line
    lens: legal
    severity: P3
    issue: '"Typically reviewed within 24-48 hours" — SLA statement is not framed as commitment vs. estimate, which can create CP Rules expectation gap'
    evidence_excerpt: '"Typically reviewed within 24-48 hours"'
    recommendation: 'Soften or commit explicitly. "Typically" is fine if Privacy/T&C section makes clear it is estimate not guarantee.'
    citation: 'CP Rules 2020 (no misleading representations)'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-042
    surface_id: mv-onb-personal-email-helper
    lens: legal
    severity: P3
    issue: 'Helper "Email is pre-filled from your account" — does not surface that the email is identity-linked and changing it requires re-verification'
    evidence_excerpt: '"Email is pre-filled from your account"'
    recommendation: 'Expand helper to: "Email is from your account. Contact support to change it." DPDP §11 right-to-correction implications.'
    citation: 'DPDP Act §11(b) (correction right)'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-043
    surface_id: web-ux-orderdetail-cancel-reason
    lens: legal
    severity: P1
    issue: '"Cancellation reason:" label — no surface statement of refund timeline or who pays for cancellation cost'
    evidence_excerpt: '"Cancellation reason:"'
    recommendation: 'Add adjacent helper or info box stating refund SLA (RBI PA Master Direction: refund within 7 days for digital payments) + link to cancellation policy.'
    citation: 'RBI PA Master Direction 2020; CP Rules 2020'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-044
    surface_id: vp-ux-orders-history-cancel-reason
    lens: legal
    severity: P2
    issue: 'Chef-side "Reason: {cancelReason}" — display only, no statement of chef-vs-platform-vs-customer-responsibility attribution for downstream refund liability'
    evidence_excerpt: '"Reason: {cancelReason}"'
    recommendation: 'Helper or info icon explaining who bears refund cost based on cancellation reason code. RBI PA Master Direction requires clear merchant-of-record disclosure.'
    citation: 'RBI PA Master Direction; CP Rules 2020'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-045
    surface_id: vp-ux-orders-live-actions
    lens: legal
    severity: P2
    issue: '"Reject" order button has no captured-reason requirement at UI label level — rejection without reason weakens platform CP Rules grievance handling'
    evidence_excerpt: '"Order History / Accept / Reject / Start Preparing / Mark Ready / Waiting for pickup"'
    recommendation: 'Reject should open reason picker (out-of-stock, kitchen closed, can''t fulfil time, etc.). Captured reason supports CP Rules grievance defence and refund decisioning.'
    citation: 'CP Rules 2020; RBI PA Master Direction'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-046
    surface_id: mc-deliverymap-markers
    lens: legal
    severity: P3
    issue: 'Map markers "Delivery Address / Chef Location / Driver" — chef address shown to customer without statement of approximation/precision and whose precise location is shared'
    evidence_excerpt: '"Delivery Address / Chef Location / Driver"'
    recommendation: 'If chef location is approximate, label "Approximate chef location". DPDP §6 + chef privacy expectation (home kitchens).'
    citation: 'DPDP Act §6; best-practice'
    depends_on: 'needs lawyer review'

  - finding_id: LEG-047
    surface_id: web-ux-loading-default
    lens: legal
    severity: P3
    issue: 'Loading screen "Loading..." — no statement during slow/blocked loads of timeout or grievance contact (covered better on legal pages, but app-shell exposure)'
    evidence_excerpt: '"Loading..."'
    recommendation: 'Out of scope for legal lens unless paired with prolonged blocking. Mention only because legal-page redirects from app shell should still expose grievance officer link.'
    citation: 'best-practice (informational)'
    depends_on: 'needs lawyer review'
```

## Business Analyst findings

```yaml
# Business Analyst Lens — Microcopy Findings
# Auditor: business-analyst lens
# Date: 2026-05-13
# Scope: 281 microcopy rows across web, vendor-portal, delivery-portal, admin-portal, mobile-customer, mobile-vendor, mobile-delivery

findings:

  # ──────────────────────────────────────────────────────────────
  # P0 — Demonstrably broken conversion path / false trust signals
  # ──────────────────────────────────────────────────────────────

  - finding_id: BA-001
    surface_id: web-ux-cart-promo
    lens: business-analyst
    severity: P0
    issue: "Hardcoded promo code hint exposes internal test code to all users"
    evidence_excerpt: "Try: FE3DR10 for 10% off"
    recommendation: >
      Remove the hardcoded hint entirely. Either pull active promotions from the
      promotions API and display them as "A code is available — check your email"
      or show no hint. Shipping a literal hardcoded discount code to every user
      undermines pricing integrity and will be screenshot-shared, training
      customers to never pay full price.
    metric_hypothesis: "cart→order conversion rate; pricing integrity; gross margin — every customer who sees this code uses it, eroding margin indefinitely"
    depends_on: null

  - finding_id: BA-002
    surface_id: web-ux-cart-promo
    lens: business-analyst
    severity: P0
    issue: "Promo success inline copy uses exclamation mark, violating brand voice rule"
    evidence_excerpt: "Promo code applied! 10% off"
    recommendation: >
      Rewrite as "Promo applied. 10% off your order." — past tense, no
      exclamation, restrained per Style Guide Rule 1. The exclamation erodes the
      calm-premium positioning.
    metric_hypothesis: "brand trust score; customer perception of discounting desperation"
    depends_on: null

  - finding_id: BA-003
    surface_id: web-ux-cart-delivery-info
    lens: business-analyst
    severity: P0
    issue: "Hardcoded prep time estimate ('30-45 mins') ships as static text regardless of chef"
    evidence_excerpt: "Estimated prep time: 30-45 mins"
    recommendation: >
      Bind to the chef's actual prep-time setting (already stored from onboarding:
      vp-onb-ops-prep-times). Display "Estimated prep time: {chef.prepTime}" drawn
      from the API. A static estimate that is wrong for a chef who set '1-2 hours'
      will cause order cancellations and negative reviews when food is late.
    metric_hypothesis: "order completion rate; chef negative-review rate; customer D30 retention"
    depends_on: null

  - finding_id: BA-004
    surface_id: web-ux-layout-auth-cta
    lens: business-analyst
    severity: P0
    issue: "Header uses banned terms 'Login' and 'Sign Up' instead of 'Sign in' / 'Sign up'"
    evidence_excerpt: "Login / Sign Up"
    recommendation: >
      Change to "Sign in" and "Sign up" (verb, two words, lowercase 'up' per
      Style Guide vocabulary list). 'Login' as a CTA is a known conversion
      friction signal — users expect 'Sign in' on modern consumer apps and
      hesitate at unfamiliar patterns.
    metric_hypothesis: "signup completion rate; sign-in click-through rate"
    depends_on: null

  - finding_id: BA-005
    surface_id: dp-mc-portal-footer
    lens: business-analyst
    severity: P0
    issue: "Delivery portal footer shows internal codename 'Fe3dr Delivery Portal' to drivers"
    evidence_excerpt: "Fe3dr Delivery Portal"
    recommendation: >
      Replace with the public product name (e.g. "Home Chef — Delivery"). The
      internal codename 'Fe3dr' is never explained to drivers; it erodes trust on
      the very screen where a new driver decides whether to register.
    metric_hypothesis: "driver signup conversion rate; driver trust / brand recognition"
    depends_on: null

  - finding_id: BA-006
    surface_id: dp-mc-portal-footer
    lens: business-analyst
    severity: P0
    issue: "Delivery portal login subtitle uses banned 'Login' vocabulary"
    evidence_excerpt: "Login or sign up to deliver with Fe3dr"
    recommendation: >
      Rewrite: "Sign in or create an account to deliver with Home Chef." Fixes
      both the banned vocabulary and the brand name in one pass.
    metric_hypothesis: "driver signup conversion rate"
    depends_on: BA-005

  # ──────────────────────────────────────────────────────────────
  # P1 — Conversion-critical on high-traffic surfaces
  # ──────────────────────────────────────────────────────────────

  - finding_id: BA-007
    surface_id: vp-onb-policies-payout-info
    lens: business-analyst
    severity: P1
    issue: "Payout setup deferred to 'after approval' with no timeline, creating anxiety at high-stakes onboarding step"
    evidence_excerpt: "You can set up your bank account or UPI details for receiving payouts from Settings after your kitchen is approved."
    recommendation: >
      Reframe to acknowledge the deferral is intentional and time-bound:
      "Payout details are set up in Settings once your kitchen is approved —
      usually within 24-48 hours. You won't miss any earnings: payouts are
      calculated from your first order." This turns an anxiety trigger into a
      trust signal. Chefs who don't understand why payout setup is deferred may
      abandon the application assuming the platform isn't ready.
    metric_hypothesis: "chef onboarding completion rate; chef D7 retention"
    depends_on: null

  - finding_id: BA-008
    surface_id: vp-onb-nav-submit-app
    lens: business-analyst
    severity: P1
    issue: "Final onboarding CTA is generic 'Submit Application' with no expectation-setting"
    evidence_excerpt: "Submit Application"
    recommendation: >
      Change CTA to "Submit application" (sentence case, Style Guide) and add a
      subtitle below the button: "We'll review your kitchen within 24-48 hours
      and send you an update." Expectation-setting at the submit moment is proven
      to reduce abandonment and post-submit support queries.
    metric_hypothesis: "chef onboarding completion rate; post-submit support ticket volume"
    depends_on: null

  - finding_id: BA-009
    surface_id: vp-ux-menu-bulk-confirm
    lens: business-analyst
    severity: P1
    issue: "Bulk delete uses browser native confirm() with vague 'cannot be undone' copy — violates Style Guide modal pattern"
    evidence_excerpt: "Delete {n} items? This cannot be undone."
    recommendation: >
      Replace window.confirm with an in-app modal dialog. Copy: "Delete {n}
      menu items? Customers won't be able to order these dishes." Follows Style
      Guide modal subtitle pattern: explain consequence, not just permanence.
      Browser dialogs are jarring on a polished app and cannot be styled to
      match the brand.
    metric_hypothesis: "accidental menu item deletion rate; chef satisfaction score"
    depends_on: null

  - finding_id: BA-010
    surface_id: vp-ux-menu-delete-confirm
    lens: business-analyst
    severity: P1
    issue: "Single item delete uses browser confirm() with vague 'Are you sure?' copy"
    evidence_excerpt: "Are you sure you want to delete this item?"
    recommendation: >
      Replace with in-app modal: "Remove '{item.name}' from your menu? Customers
      won't be able to order it." Follows the same correction as BA-009.
    metric_hypothesis: "chef error recovery rate; accidental deletion"
    depends_on: BA-009

  - finding_id: BA-011
    surface_id: web-ux-chef-card-favorite-toast-loggedout
    lens: business-analyst
    severity: P1
    issue: "Unauthenticated favourite toast is a dead end — 'Please log in' with no action link"
    evidence_excerpt: "Please log in to save favorites"
    recommendation: >
      Change to a toast with an inline action: "Sign in to save this chef." with
      a 'Sign in' link button in the toast. 'Please log in' uses banned vocabulary
      and provides no path forward, causing the user to lose their intended action.
    metric_hypothesis: "signup conversion rate from browse; favorite-to-order conversion"
    depends_on: null

  - finding_id: BA-012
    surface_id: vp-ux-orders-live-actions
    lens: business-analyst
    severity: P1
    issue: "Chef action buttons mix tenses and formats: 'Accept/Reject' (imperative) vs 'Waiting for pickup' (status text)"
    evidence_excerpt: "Accept / Reject / Start Preparing / Mark Ready / Waiting for pickup"
    recommendation: >
      Normalise all to verb-first, sentence case: 'Accept order', 'Reject order',
      'Start preparing', 'Mark ready'. The status 'Waiting for pickup' should move
      to a status badge, not a button area — it is not an action. Mixing action
      buttons with status text causes chefs to tap the wrong element under time
      pressure.
    metric_hypothesis: "chef order-processing speed; order error rate (wrong action triggered)"
    depends_on: null

  - finding_id: BA-013
    surface_id: mc-perm-camera
    lens: business-analyst
    severity: P1
    issue: "iOS camera permission string 'Used to take photos' is vague — does not explain the business use case"
    evidence_excerpt: "Used to take photos"
    recommendation: >
      Rewrite: "Used to take photos of your profile and food items when placing
      orders." iOS users who see a generic string are more likely to deny
      permission, permanently blocking photo features. Apple recommends
      purpose-specific language.
    metric_hypothesis: "camera permission grant rate; profile photo upload rate; menu item photo upload rate"
    depends_on: null

  - finding_id: BA-014
    surface_id: dp-mc-stripe-cells
    lens: business-analyst
    severity: P1
    issue: "Stripe status labels ('Charges', 'Payouts', 'Details', 'Submitted', 'Incomplete') are raw Stripe API jargon, not plain English for drivers"
    evidence_excerpt: "Charges / Payouts / Details + Enabled / Pending / Submitted / Incomplete"
    recommendation: >
      Translate to driver-facing language:
      'Charges' → 'Accept payments', 'Payouts' → 'Receive earnings',
      'Details' → 'Identity verified'. States: 'Enabled' → 'Ready',
      'Pending' → 'In review', 'Submitted' → 'Submitted — awaiting approval',
      'Incomplete' → 'Action needed'. Drivers who see 'Charges: Pending' don't
      know what to do next; confusion here causes driver churn before first
      delivery.
    metric_hypothesis: "driver payment setup completion rate; driver time-to-first-delivery"
    depends_on: null

  - finding_id: BA-015
    surface_id: vp-ux-notifs-sla-line
    lens: business-analyst
    severity: P1
    issue: "Kitchen review SLA is vague range ('24-48 hours') with no commitment or next-step instruction"
    evidence_excerpt: "Typically reviewed within 24-48 hours"
    recommendation: >
      Rewrite: "We review kitchens within 48 hours on business days. Check back
      here for updates or watch for an email from us." The word 'typically'
      weakens the promise. Adding 'business days' sets accurate expectations.
      Adding the email mention reduces repeat check-in traffic.
    metric_hypothesis: "chef onboarding drop-off during review wait; inbound support volume for 'when will I be approved?'"
    depends_on: null

  - finding_id: BA-016
    surface_id: mv-menuedit-price-change-banner
    lens: business-analyst
    severity: P1
    issue: "Price change warning is long (14 words) for a mobile glanceable context and buries the key fact (24h delay)"
    evidence_excerpt: "Price changes are submitted for admin review and may take 24 hours to reflect."
    recommendation: >
      Shorten for mobile: "Price changes take up to 24 hours to go live." Place
      this inline under the price field, not as a banner. The current phrasing
      implies the chef's own price is subject to admin override, which may reduce
      willingness to update prices — a negative business impact.
    metric_hypothesis: "chef menu update frequency; chef price optimisation behaviour"
    depends_on: null

  - finding_id: BA-017
    surface_id: mc-checkout-address-placeholders
    lens: business-analyst
    severity: P1
    issue: "Checkout address fields use placeholder text as labels (e.g. 'Address line 1 *') — placeholders disappear on focus, losing the label"
    evidence_excerpt: "Address line 1 * / Address line 2 (optional) / City * / State * / Pincode *"
    recommendation: >
      Separate floating labels from placeholder hints. Label: 'Address line 1'
      (with asterisk on label, not in placeholder). Placeholder: 'e.g. 12, MG
      Road'. Style Guide specifies: asterisk belongs on the field, label stays
      clean. The current pattern increases form abandonment because users forget
      what a field is for when they start typing.
    metric_hypothesis: "checkout address-entry completion rate; checkout → order conversion"
    depends_on: null

  # ──────────────────────────────────────────────────────────────
  # P2 — Missed opportunity on medium-traffic surfaces
  # ──────────────────────────────────────────────────────────────

  - finding_id: BA-018
    surface_id: web-ux-checkout-instructions
    lens: business-analyst
    severity: P2
    issue: "Special Instructions field has no 'why we ask' helper text — a missed upsell and allergen-safety opportunity"
    evidence_excerpt: "Special Instructions / Any special requests or delivery instructions..."
    recommendation: >
      Add helper text: "Let the chef know about allergies, spice preferences, or
      delivery notes." Naming allergens explicitly increases chef awareness and
      provides a legal paper trail. It also nudges personalisation (a retention
      driver).
    metric_hypothesis: "allergen incident rate; customer satisfaction score; repeat order rate"
    depends_on: null

  - finding_id: BA-019
    surface_id: web-ux-orders-search
    lens: business-analyst
    severity: P2
    issue: "Order search placeholder 'Search orders...' misses opportunity to prompt reorder behaviour"
    evidence_excerpt: "Search your orders / Search orders..."
    recommendation: >
      Placeholder: "Search by dish, chef, or order number..." This sets accurate
      scope expectations and reminds customers they can search by dish — prompting
      a reorder of a favourite.
    metric_hypothesis: "reorder rate; orders-tab engagement"
    depends_on: null

  - finding_id: BA-020
    surface_id: vp-onb-ops-delivery-fee
    lens: business-analyst
    severity: P2
    issue: "Delivery fee field has no context about who sets the fee or who pays it"
    evidence_excerpt: "Delivery Fee / Per order delivery charge"
    recommendation: >
      Add helper text: "This fee is shown to customers and added to their order
      total. Set to ₹0 for free delivery." Chefs who don't understand the
      economic model set arbitrary fees or leave it blank, causing customer
      confusion at checkout.
    metric_hypothesis: "chef onboarding completion rate; cart abandonment from surprise delivery fee"
    depends_on: null

  - finding_id: BA-021
    surface_id: vp-onb-ops-min-order
    lens: business-analyst
    severity: P2
    issue: "Minimum order field hint 'Set to 0 for no minimum' implies ₹0 but the currency context is missing"
    evidence_excerpt: "Minimum Order Value (Optional) / Set to 0 for no minimum"
    recommendation: >
      Update helper: "Set a minimum order amount in ₹ to keep each order
      profitable. Leave blank or enter 0 for no minimum." Adding the ₹ symbol
      removes ambiguity, and 'to keep each order profitable' gives the chef a
      reason to engage with the field rather than skipping it.
    metric_hypothesis: "average order value; chef profitability per order"
    depends_on: null

  - finding_id: BA-022
    surface_id: vp-ux-reviews-reply
    lens: business-analyst
    severity: P2
    issue: "Reply success toast 'Reply posted successfully' is verbose and passive; violates 6-word max rule"
    evidence_excerpt: "Reply posted successfully"
    recommendation: >
      Rewrite to: "Reply posted." — past tense, ≤6 words, period. The word
      'successfully' adds length without meaning (if it failed, there would be
      an error).
    metric_hypothesis: "UI quality perception; chef engagement with review-reply feature"
    depends_on: null

  - finding_id: BA-023
    surface_id: vp-ux-profile-toggle
    lens: business-analyst
    severity: P2
    issue: "'Orders Paused' toggle state gives no indication of customer impact or how to resume"
    evidence_excerpt: "Accepting Orders / Orders Paused"
    recommendation: >
      Keep 'Accepting orders' for the active state. For the paused state: 'Paused
      — customers can't order'. Add a brief helper: "Your profile is hidden while
      paused." This helps chefs understand the business consequence of pausing,
      potentially reducing unnecessary pauses.
    metric_hypothesis: "GMV loss from unnecessary pauses; chef active-hours metric"
    depends_on: null

  - finding_id: BA-024
    surface_id: vp-ux-earnings-sections
    lens: business-analyst
    severity: P2
    issue: "'No payouts yet' empty state has no next-step instruction to set up payout method"
    evidence_excerpt: "No payouts yet"
    recommendation: >
      Add a sub-line with a CTA: "No payouts yet. Set up your bank account or
      UPI in Settings to start receiving earnings." An empty state without a
      clear action leaves chefs uncertain whether they will be paid.
    metric_hypothesis: "payout method setup rate; chef financial trust score; support tickets about missing payouts"
    depends_on: null

  - finding_id: BA-025
    surface_id: vp-ux-dashboard-stats-labels
    lens: business-analyst
    severity: P2
    issue: "'No reviews yet' is a missed activation prompt for new chefs — tells nothing about how to get reviews"
    evidence_excerpt: "No reviews yet"
    recommendation: >
      Expand to: "No reviews yet. Complete a few orders to start building your
      rating." This is the first-5-orders activation moment. Chefs who see an
      empty rating widget without context may deprioritise order quality.
    metric_hypothesis: "chef D7 retention; first-order quality rating"
    depends_on: null

  - finding_id: BA-026
    surface_id: mc-order-detail-eta
    lens: business-analyst
    severity: P2
    issue: "'ETA:' prefix is inconsistent with 'Est. arrival:' used in the OrderTimeline component on the same order detail screen"
    evidence_excerpt: "ETA: {time} (order detail page) vs Est. arrival: {time} (timeline component)"
    recommendation: >
      Standardise to one pattern across both surfaces. Preferred: 'Arrives at
      {time}' — no prefix abbreviation needed, cleaner for the customer persona
      tone (10-18 word conversational sentences). Two different abbreviations
      on the same screen signals a lack of design coherence to the customer.
    metric_hypothesis: "customer trust score; app store review sentiment (polish)"
    depends_on: null

  - finding_id: BA-027
    surface_id: mc-order-card-meta
    lens: business-analyst
    severity: P2
    issue: "'item(s)' plural pattern violates Style Guide — must never use (s) construction"
    evidence_excerpt: "{n} item(s) • {date}"
    recommendation: >
      Replace with proper plural: "{n} {n === 1 ? 'item' : 'items'} • {date}".
      Style Guide section 6 explicitly bans the (s) construction. Also present
      in two web pages (web chef orders and web customer orders).
    metric_hypothesis: "UI quality perception; App Store review polish; i18n readiness"
    depends_on: null

  - finding_id: BA-028
    surface_id: vp-ux-menu-form-allergen-help
    lens: business-analyst
    severity: P2
    issue: "Allergen helper text is purely mechanical ('Type each allergen and press Enter') with no mention of why accuracy matters"
    evidence_excerpt: "Type each allergen and press Enter to add"
    recommendation: >
      Add a second sentence: "Accurate allergen information protects your
      customers and your kitchen." This brief 'why we ask' line is proven to
      increase completion accuracy on safety-critical fields.
    metric_hypothesis: "allergen declaration accuracy rate; allergen-related incident rate; customer safety"
    depends_on: null

  - finding_id: BA-029
    surface_id: vp-onb-kitchen-desc-field
    lens: business-analyst
    severity: P2
    issue: "Kitchen description helper references 'your signature touch' — banned near-artisanal language"
    evidence_excerpt: "Describe your cooking style, what makes your food special, your signature touch..."
    recommendation: >
      Rewrite: "Describe your cooking style and specialities. This appears on
      your public kitchen page." Removes the sentimentalised phrase, adds the
      practical consequence (public visibility), and keeps the Style Guide
      anti-reference list clean.
    metric_hypothesis: "chef profile completion rate; chef profile click-through rate from browse"
    depends_on: null

  - finding_id: BA-030
    surface_id: vp-onb-docs-kitchen-photo-items
    lens: business-analyst
    severity: P2
    issue: "Kitchen photo titles (Cooking Area / Preparation Area / Storage / Packaging) have no explanation of how they affect approval"
    evidence_excerpt: "Kitchen Photos / Photos of your kitchen help build trust. At least one photo of your cooking area is required."
    recommendation: >
      Add helper text per photo slot explaining what reviewers look for. Example
      for Cooking Area: "Show a clean, organised cooking surface. This is the
      most important photo for approval." Chefs who understand review criteria
      upload better photos, reducing back-and-forth and speeding time to approval.
    metric_hypothesis: "kitchen photo approval pass rate; time-to-approval; chef onboarding completion"
    depends_on: null

  - finding_id: BA-031
    surface_id: dp-mc-action-required
    lens: business-analyst
    severity: P2
    issue: "'Action Required' badge in driver settings gives no indication of what action is needed"
    evidence_excerpt: "Action Required"
    recommendation: >
      Expand to a contextual message wherever this badge appears: "Action
      required — complete Stripe onboarding to receive payouts." A badge without
      a next step is a dead end that causes support tickets and driver churn.
    metric_hypothesis: "driver payment setup completion rate; driver support ticket volume"
    depends_on: BA-014

  - finding_id: BA-032
    surface_id: vp-ux-notifs-respond-form
    lens: business-analyst
    severity: P2
    issue: "Reply form placeholder contains FSSAI example — good intent but presumes the only reason for an 'info requested' status is a license upload"
    evidence_excerpt: "Type your response to the admin... (e.g., I've uploaded the FSSAI license, please review)"
    recommendation: >
      Generalise the example or use a multi-example pattern: "e.g. 'I've
      re-uploaded my FSSAI certificate' or 'Kitchen photos have been updated'."
      A single example anchors chefs to one scenario and they may copy it
      verbatim even when irrelevant, reducing the quality of admin-chef communication.
    metric_hypothesis: "admin response quality score; time-to-approval after info requested"
    depends_on: null

  - finding_id: BA-033
    surface_id: vp-auth-login-password-placeholder
    lens: business-analyst
    severity: P2
    issue: "Password placeholder 'Enter your password' is redundant instruction — adds no value and matches the pattern the Style Guide bans"
    evidence_excerpt: "Enter your password"
    recommendation: >
      Remove the placeholder entirely (password fields should not have
      placeholders — screen readers read them on every focus, and they add
      nothing). If a placeholder is required for design reasons, use a neutral
      indicator like '••••••••'. Same fix applies to dp-mc-password-placeholder
      and ap-auth-password-placeholder.
    metric_hypothesis: "sign-in friction; accessibility compliance"
    depends_on: null

  - finding_id: BA-034
    surface_id: vp-auth-register-confirm-placeholder
    lens: business-analyst
    severity: P2
    issue: "'Re-enter password' placeholder is instructional copy that belongs in the label, not the placeholder"
    evidence_excerpt: "Re-enter password"
    recommendation: >
      Move to the field label: 'Confirm password'. Leave placeholder empty or
      use '••••••••'. Placeholder text disappears the moment the user focuses,
      causing confusion about what the field expects.
    metric_hypothesis: "registration form completion rate; password-mismatch error rate"
    depends_on: null

  - finding_id: BA-035
    surface_id: vp-ux-orders-live-note
    lens: business-analyst
    severity: P2
    issue: "'Note: {specialInstructions}' prefix is a raw colon label — not styled as a callout, easy to miss under time pressure"
    evidence_excerpt: "Note: {specialInstructions}"
    recommendation: >
      Rename to 'Customer note:' and visually elevate (tinted background chip).
      Chefs missing special instructions (allergens, spice level) causes the most
      common 1-star reviews. The copy and visual treatment must make this
      impossible to overlook.
    metric_hypothesis: "order quality rating; allergen incident rate; 1-star review rate"
    depends_on: null

  - finding_id: BA-036
    surface_id: mc-catering-placeholders
    lens: business-analyst
    severity: P2
    issue: "Catering budget placeholder '25000' is bare number with no currency symbol — ambiguous for international customers"
    evidence_excerpt: "25000"
    recommendation: >
      Change placeholder to '₹25,000' — includes currency symbol and uses
      en-IN comma-grouping. Style Guide specifies ₹120 format (symbol first, no
      space). Without the symbol, some customers may interpret as per-person
      vs total, abandoning the form.
    metric_hypothesis: "catering inquiry completion rate; budget entry accuracy"
    depends_on: null

  # ──────────────────────────────────────────────────────────────
  # P3 — Minor friction / low-traffic internal surfaces
  # ──────────────────────────────────────────────────────────────

  - finding_id: BA-037
    surface_id: ap-layout-account-logout
    lens: business-analyst
    severity: P3
    issue: "'Logout' in admin layout uses banned vocabulary — should be 'Sign out'"
    evidence_excerpt: "Logout"
    recommendation: >
      Change to 'Sign out'. Same fix required in dp-mc-logout (delivery portal
      layout). Both layouts violate the Style Guide vocabulary list which
      mandates 'Sign out' ✅ / 'Log out' ❌.
    metric_hypothesis: "brand consistency; vocabulary audit compliance"
    depends_on: null

  - finding_id: BA-038
    surface_id: ap-approvaldetail-yesno
    lens: business-analyst
    severity: P3
    issue: "'Yes / No' boolean rendering in approval detail is ambiguous without question context"
    evidence_excerpt: "Yes; No"
    recommendation: >
      Replace with meaningful labels that reflect the field's meaning (e.g.,
      'Verified / Not verified', 'Provided / Not provided'). Admin staff reviewing
      approvals need specific context, not bare booleans, to make correct decisions.
    metric_hypothesis: "admin approval accuracy; approval processing time"
    depends_on: null

  - finding_id: BA-039
    surface_id: ap-providercreate-helpers
    lens: business-analyst
    severity: P3
    issue: "Helper 'No status mappings configured. Add mappings to translate provider statuses to Fe3dr statuses.' uses internal codename 'Fe3dr'"
    evidence_excerpt: "No status mappings configured. Add mappings to translate provider statuses to Fe3dr statuses."
    recommendation: >
      Replace 'Fe3dr statuses' with 'platform statuses'. While admin-facing copy
      has more latitude, using the internal codename signals an unfinished product
      to partner integrators who may also see this page.
    metric_hypothesis: "admin tool credibility; partner integration success rate"
    depends_on: null

  - finding_id: BA-040
    surface_id: vp-ux-payouts-table
    lens: business-analyst
    severity: P3
    issue: "Payout table header 'Method' is too terse — ambiguous between payment method (Razorpay/Stripe) and payout method (bank/UPI)"
    evidence_excerpt: "Date / Amount / Status / Method"
    recommendation: >
      Rename to 'Payout method'. The extra word eliminates ambiguity for chefs
      who have different payment processing vs payout method settings.
    metric_hypothesis: "payout support ticket volume; chef clarity on earnings"
    depends_on: null

  - finding_id: BA-041
    surface_id: ap-approvals-priorities
    lens: business-analyst
    severity: P3
    issue: "'urgent; high; normal; low' priority badges are lowercase — inconsistent with sentence-case convention"
    evidence_excerpt: "urgent; high; normal; low"
    recommendation: >
      Capitalise: 'Urgent / High / Normal / Low'. All badge labels across the
      inventory use sentence case or title case; bare lowercase in a UI badge
      looks like unstyled data leakage.
    metric_hypothesis: "admin UI polish; operational efficiency"
    depends_on: null

  - finding_id: BA-042
    surface_id: vp-ux-loading-screen
    lens: business-analyst
    severity: P3
    issue: "'Loading...' generic text on full-page loader gives no context — same string used in both vendor portal and web app"
    evidence_excerpt: "Loading..."
    recommendation: >
      Make loading messages context-aware where feasible: 'Loading your
      dashboard...', 'Loading menu...', 'Checking your order...'. A generic
      loading state increases perceived latency and is a missed micro-delight
      opportunity. If a single component must stay generic, 'Just a moment.'
      is more brand-aligned than 'Loading...'
    metric_hypothesis: "perceived performance score; customer drop-off during load states"
    depends_on: null

  - finding_id: BA-043
    surface_id: vp-onb-kitchen-specialties-title
    lens: business-analyst
    severity: P3
    issue: "Specialties section marked '(optional)' in the title — optional markers belong on the field, not in the section heading per Style Guide"
    evidence_excerpt: "Signature Dishes & Specialties / Add your best dishes - these appear as tags on your profile (optional)"
    recommendation: >
      Move the optional indicator to the field placeholder or as a quiet helper
      text: 'Optional — these appear as tags on your public profile.' The ampersand
      (&) in 'Dishes & Specialties' should be spelled out: 'Signature dishes and
      specialties'.
    metric_hypothesis: "profile completion rate; chef discoverability via cuisine tags"
    depends_on: null

  - finding_id: BA-044
    surface_id: vp-onb-personal-avatar-label
    lens: business-analyst
    severity: P3
    issue: "Avatar field label parenthetical '(Optional)' embedded in label text, not separate helper — label should stay clean per Style Guide"
    evidence_excerpt: "Profile Photo (Optional) / Shown on your kitchen page. JPEG, PNG, or WebP. Max 5 MB."
    recommendation: >
      Label: 'Profile photo'. Helper: 'Optional. Shown on your public kitchen
      page. JPEG, PNG, or WebP up to 5 MB.' Separating label and helper follows
      the Style Guide form-label pattern and prevents screen readers from
      announcing '(Optional)' mid-label.
    metric_hypothesis: "profile photo upload rate; accessibility compliance"
    depends_on: null

  - finding_id: BA-045
    surface_id: mv-settings-accepting-helper
    lens: business-analyst
    severity: P3
    issue: "Toggle helper text 'Toggle to start or pause accepting orders' describes the UI control, not the business consequence"
    evidence_excerpt: "Toggle to start or pause accepting orders"
    recommendation: >
      Replace with consequence-focused helper: "When paused, your kitchen is
      hidden from customers." This aligns with the Style Guide principle of
      explaining consequences, not mechanics.
    metric_hypothesis: "unintentional pauses causing GMV loss; chef active-hours metric"
    depends_on: null

  - finding_id: BA-046
    surface_id: ap-auditlogs-filter-placeholders
    lens: business-analyst
    severity: P3
    issue: "Audit log filter placeholder 'e.g. chef.verify' uses dot-notation event codes unfamiliar to non-developer admins"
    evidence_excerpt: "e.g. chef.verify; e.g. user, chef"
    recommendation: >
      Supplement with plain-English examples: "e.g. chef.verify (verify a
      kitchen) or order.cancel (cancel an order)". Admin operators who are not
      developers will skip this filter entirely if they cannot guess valid values.
    metric_hypothesis: "audit log feature usage rate; admin operational efficiency"
    depends_on: null
```

## Brand Voice findings

```yaml
# Brand-Voice audit — Microcopy category (281 rows)
# Lens: cross-app voice consistency against STYLE-GUIDE.md + .impeccable.md
# Severity: P0 cross-surface trust break · P1 entry-surface drift · P2 secondary tone · P3 punctuation/case

# ─────────────────────────────────────────────────────────────
# P0 — Brand identity contradictions across apps
# ─────────────────────────────────────────────────────────────

findings:

  - finding_id: BV-001
    surface_id: dp-mc-portal-footer
    lens: brand-voice
    severity: P0
    issue: "Brand name 'Fe3dr' appears in delivery-portal footer; rest of platform is 'Home Chef'"
    evidence_excerpt: "Fe3dr Delivery Portal"
    related_surfaces: ["ap-auth-email-placeholder"]
    recommendation: "Rename to 'Home Chef Delivery' — single platform identity across all surfaces. Audit codebase for every 'Fe3dr' / 'fe3dr' occurrence (admin email placeholder admin@fe3dr.com also drifts)."
    depends_on: null

  - finding_id: BV-002
    surface_id: ap-auth-email-placeholder
    lens: brand-voice
    severity: P0
    issue: "Admin login example email uses 'fe3dr.com' domain — brand contradiction"
    evidence_excerpt: "admin@fe3dr.com"
    related_surfaces: ["dp-mc-portal-footer", "vp-auth-login-email-placeholder"]
    recommendation: "Use 'admin@homechef.app' (or the canonical brand domain). Align with vendor/driver portals that use 'you@example.com' — better still, all should standardise."
    depends_on: null

  - finding_id: BV-003
    surface_id: web-ux-layout-auth-cta
    lens: brand-voice
    severity: P0
    issue: "Web header uses 'Login / Sign Up' — both forms violate style guide"
    evidence_excerpt: "Login / Sign Up"
    related_surfaces: ["dp-mc-signin-submit", "vp-auth-login-loading", "ap-auth-email-cta", "ap-auth-signing-in"]
    recommendation: "Replace with 'Sign in / Sign up' (sentence case, two words, 'sign' not 'log'). Drives consistency with delivery-portal which already uses 'Sign in'."
    depends_on: null

  - finding_id: BV-004
    surface_id: dp-mc-logout
    lens: brand-voice
    severity: P0
    issue: "Delivery-portal layout button says 'Logout' but settings page says 'Sign Out' — within-app contradiction; whole platform should use 'Sign out'"
    evidence_excerpt: "Logout"
    related_surfaces: ["ap-layout-account-logout", "md-mic-001"]
    recommendation: "Rename to 'Sign out' everywhere (sentence case, two words). Banned variants: 'Logout', 'Log out', 'Sign Out'."
    depends_on: null

  - finding_id: BV-005
    surface_id: ap-layout-account-logout
    lens: brand-voice
    severity: P0
    issue: "Admin layout uses 'Logout' — banned variant"
    evidence_excerpt: "Logout"
    related_surfaces: ["dp-mc-logout", "md-mic-001"]
    recommendation: "Rename to 'Sign out'."
    depends_on: null

  - finding_id: BV-006
    surface_id: md-mic-001
    lens: brand-voice
    severity: P0
    issue: "Mobile-delivery alert uses 'Logout' — banned variant; matches drift across delivery-portal + admin-portal"
    evidence_excerpt: "Cancel / Logout"
    related_surfaces: ["dp-mc-logout", "ap-layout-account-logout"]
    recommendation: "Rename to 'Sign out'."
    depends_on: null

  - finding_id: BV-007
    surface_id: web-ux-chef-card-favorite-toast-loggedout
    lens: brand-voice
    severity: P0
    issue: "Toast says 'Please log in to save favorites' — 'log in' is the banned variant"
    evidence_excerpt: "Please log in to save favorites"
    related_surfaces: ["web-ux-layout-auth-cta"]
    recommendation: "Rewrite as 'Sign in to save favourites.' (sentence case, 'Sign in' per style guide, en-IN spelling 'favourites')."
    depends_on: null

  # ─────────────────────────────────────────────────────────────
  # P0 — Persona naming drift (Driver / Delivery Partner)
  # ─────────────────────────────────────────────────────────────

  - finding_id: BV-008
    surface_id: dp-mc-role-partner
    lens: brand-voice
    severity: P0
    issue: "Driver-facing UI labels the role 'Delivery Partner' — style guide says 'Driver' in driver-facing surfaces; 'Delivery partner' is customer-facing only"
    evidence_excerpt: "Delivery Partner"
    related_surfaces: ["dp-mc-displayname-fallback", "dp-mc-driver-navigation"]
    recommendation: "Rename to 'Driver' (sentence case) on the delivery-portal layout. Reserve 'Delivery partner' for the customer apps when referring to who is bringing their food."
    depends_on: null

  # ─────────────────────────────────────────────────────────────
  # P0 — Cross-app brand naming: 'Home chef' vs 'Chef' vs 'Cook' / 'Kitchen'
  # ─────────────────────────────────────────────────────────────

  - finding_id: BV-009
    surface_id: ap-chefs-card-unnamed
    lens: brand-voice
    severity: P1
    issue: "Admin uses 'Unnamed Kitchen' as fallback — style guide naming convention is 'Home chef' (customer-facing) or 'Chef' (chef-facing). 'Kitchen' as the noun for the seller mixes object with role."
    evidence_excerpt: "Unnamed Kitchen"
    related_surfaces: ["ap-chefs-action-verify-aria", "ap-chefs-action-reject-aria", "ap-chefs-action-suspend-aria"]
    recommendation: "Decide one noun and stick to it. Recommend 'Unnamed chef' for the seller entity; 'Kitchen' for the physical premises (kitchen photos, kitchen setup). Audit all 'Verify Kitchen' / 'Suspend Kitchen' admin actions for consistency with chef vs kitchen semantics."
    depends_on: null

  # ─────────────────────────────────────────────────────────────
  # P1 — Title Case button epidemic (vendor-portal + admin-portal)
  # Style guide §4: buttons are sentence case, ≤3 words
  # ─────────────────────────────────────────────────────────────

  - finding_id: BV-010
    surface_id: vp-ux-menu-form-actions
    lens: brand-voice
    severity: P1
    issue: "Menu-form buttons use Title Case ('Save Changes', 'Create Item') — style guide demands sentence case for buttons"
    evidence_excerpt: "Cancel / Save Changes / Create Item / Back to menu"
    related_surfaces: ["vp-ux-menu-form-new-category", "vp-ux-reviews-reply", "ap-providercreate-submit", "ap-providers-cta-add", "ap-staff-cta-invite", "ap-delivery-cta-providers"]
    recommendation: "Sentence-case all buttons: 'Save changes', 'Create item'. Audit every vendor-portal + admin-portal CTA — 'Back to menu' (already correct) is the model."
    depends_on: null

  - finding_id: BV-011
    surface_id: vp-onb-nav-review-app
    lens: brand-voice
    severity: P1
    issue: "Onboarding nav button 'Review Application' is Title Case"
    evidence_excerpt: "Review Application"
    related_surfaces: ["vp-onb-nav-submit-app", "vp-onb-mobile-review-title"]
    recommendation: "Rename to 'Review application' / 'Submit application' / 'Review & submit'."
    depends_on: null

  - finding_id: BV-012
    surface_id: vp-onb-nav-submit-app
    lens: brand-voice
    severity: P1
    issue: "'Submit Application' is Title Case"
    evidence_excerpt: "Submit Application"
    related_surfaces: ["vp-onb-nav-review-app"]
    recommendation: "Rename to 'Submit application'."
    depends_on: null

  - finding_id: BV-013
    surface_id: vp-onb-mobile-review-title
    lens: brand-voice
    severity: P3
    issue: "'Review & Submit' is Title Case in step indicator"
    evidence_excerpt: "Review & Submit"
    related_surfaces: ["vp-onb-stepper-review-step"]
    recommendation: "Rename to 'Review & submit'."
    depends_on: null

  - finding_id: BV-014
    surface_id: ap-approvaldetail-actions
    lens: brand-voice
    severity: P1
    issue: "Admin action buttons in Title Case — 'Approve / Reject / Request Info'"
    evidence_excerpt: "Approve; Reject; Request Info"
    related_surfaces: ["ap-providerdetail-actions", "ap-staffdetail-actions"]
    recommendation: "Sentence case: 'Approve / Reject / Request info'. 'Approve' and 'Reject' single-word verbs already OK; only 'Request Info' needs change."
    depends_on: null

  - finding_id: BV-015
    surface_id: ap-providerdetail-actions
    lens: brand-voice
    severity: P1
    issue: "Provider-detail actions Title Cased: 'Test Connection', 'Edit Provider', 'Delete Provider'"
    evidence_excerpt: "Test Connection; Enable; Disable; Edit Provider; Delete Provider"
    related_surfaces: ["ap-providers-cta-add", "ap-providercreate-submit", "ap-settings-payment-test-buttons"]
    recommendation: "Sentence case: 'Test connection', 'Edit provider', 'Delete provider'."
    depends_on: null

  - finding_id: BV-016
    surface_id: ap-providers-cta-add
    lens: brand-voice
    severity: P1
    issue: "'Add Provider' Title Cased"
    evidence_excerpt: "Add Provider"
    related_surfaces: ["ap-staff-cta-invite", "ap-delivery-cta-providers"]
    recommendation: "'Add provider'."
    depends_on: null

  - finding_id: BV-017
    surface_id: ap-staff-cta-invite
    lens: brand-voice
    severity: P1
    issue: "'Invite Staff' Title Cased"
    evidence_excerpt: "Invite Staff"
    related_surfaces: ["ap-providers-cta-add"]
    recommendation: "'Invite staff'."
    depends_on: null

  - finding_id: BV-018
    surface_id: ap-delivery-cta-providers
    lens: brand-voice
    severity: P1
    issue: "'Manage Providers' Title Cased"
    evidence_excerpt: "Manage Providers"
    related_surfaces: ["ap-providers-cta-add"]
    recommendation: "'Manage providers'."
    depends_on: null

  - finding_id: BV-019
    surface_id: ap-providercreate-submit
    lens: brand-voice
    severity: P1
    issue: "Submit buttons Title Cased: 'Create Provider', 'Add Mapping'"
    evidence_excerpt: "Create Provider; Cancel; Add Mapping; Add"
    related_surfaces: ["ap-providerdetail-actions"]
    recommendation: "'Create provider', 'Add mapping'."
    depends_on: null

  - finding_id: BV-020
    surface_id: ap-settings-payment-test-buttons
    lens: brand-voice
    severity: P1
    issue: "Payment settings buttons Title Cased — 'Test Connection', 'Save Keys'"
    evidence_excerpt: "Test Connection; Testing...; Save Keys; Saving...; Configured; Not configured"
    related_surfaces: ["ap-settings-card-ctas"]
    recommendation: "'Test connection', 'Save keys'. (Loading states 'Testing…' / 'Saving…' are already sentence case — model.)"
    depends_on: null

  - finding_id: BV-021
    surface_id: ap-settings-card-ctas
    lens: brand-voice
    severity: P3
    issue: "Single-word CTAs 'Manage; Download; View' look fine, but adjacent surfaces use Title Case — consistency risk"
    evidence_excerpt: "Manage; Download; View"
    related_surfaces: ["ap-settings-payment-test-buttons"]
    recommendation: "Keep — these are correct. Use as the standard pattern."
    depends_on: null

  - finding_id: BV-022
    surface_id: ap-staffdetail-actions
    lens: brand-voice
    severity: P1
    issue: "Staff actions Title Cased: 'Change Role', 'Confirm Deactivation', 'Confirm Reactivation'"
    evidence_excerpt: "Deactivate; Reactivate; Change Role; Confirm Deactivation; Confirm Reactivation"
    related_surfaces: ["ap-staffdetail-current-role-label"]
    recommendation: "'Change role', 'Confirm deactivation', 'Confirm reactivation'."
    depends_on: null

  - finding_id: BV-023
    surface_id: ap-staffdetail-current-role-label
    lens: brand-voice
    severity: P3
    issue: "'Current Role' Title Cased label"
    evidence_excerpt: "Current Role; (current)"
    related_surfaces: ["ap-staffdetail-actions"]
    recommendation: "'Current role'."
    depends_on: null

  - finding_id: BV-024
    surface_id: ap-staffdetail-back
    lens: brand-voice
    severity: P3
    issue: "'Back to Staff' Title Cased nav button"
    evidence_excerpt: "Back to Staff"
    related_surfaces: ["ap-providerdetail-back", "ap-userdetail-back", "ap-approvaldetail-back"]
    recommendation: "'Back to staff' / 'Back to providers' / 'Back to users' / 'Back to reviews' — sentence case nav."
    depends_on: null

  - finding_id: BV-025
    surface_id: ap-providerdetail-back
    lens: brand-voice
    severity: P3
    issue: "'Back to Providers' Title Cased"
    evidence_excerpt: "Back to Providers"
    related_surfaces: ["ap-staffdetail-back", "ap-userdetail-back", "ap-approvaldetail-back"]
    recommendation: "'Back to providers'."
    depends_on: null

  - finding_id: BV-026
    surface_id: ap-userdetail-back
    lens: brand-voice
    severity: P3
    issue: "'Back to Users' Title Cased"
    evidence_excerpt: "Back to Users"
    related_surfaces: ["ap-staffdetail-back", "ap-providerdetail-back"]
    recommendation: "'Back to users'."
    depends_on: null

  - finding_id: BV-027
    surface_id: ap-approvaldetail-back
    lens: brand-voice
    severity: P3
    issue: "'Back to Reviews' Title Cased"
    evidence_excerpt: "Back to Reviews"
    related_surfaces: ["ap-staffdetail-back", "ap-providerdetail-back", "ap-userdetail-back"]
    recommendation: "'Back to reviews'."
    depends_on: null

  - finding_id: BV-028
    surface_id: vp-ux-reviews-reply
    lens: brand-voice
    severity: P1
    issue: "'Post Reply' Title Cased button"
    evidence_excerpt: "Your reply / Write your reply... / Post Reply / Cancel / Reply / Reply posted successfully"
    related_surfaces: ["vp-ux-notifs-respond-form"]
    recommendation: "'Post reply'. Also see toast — see BV-051."
    depends_on: null

  - finding_id: BV-029
    surface_id: vp-ux-notifs-respond-form
    lens: brand-voice
    severity: P1
    issue: "'Respond & Send' is Title Cased (and reads awkwardly — two verbs)"
    evidence_excerpt: "Type your response to the admin... / Respond & Send"
    related_surfaces: ["vp-ux-reviews-reply"]
    recommendation: "'Send response' — one verb, sentence case, ≤3 words."
    depends_on: null

  - finding_id: BV-030
    surface_id: vp-ux-orders-live-actions
    lens: brand-voice
    severity: P1
    issue: "Live-order actions Title Cased — 'Start Preparing', 'Mark Ready'"
    evidence_excerpt: "Order History / Accept / Reject / Start Preparing / Mark Ready / Waiting for pickup"
    related_surfaces: ["vp-ux-menu-card-aria"]
    recommendation: "'Start preparing', 'Mark ready', 'Order history'. (Single-word 'Accept' / 'Reject' already fine.)"
    depends_on: null

  - finding_id: BV-031
    surface_id: vp-ux-orderstatus-badges
    lens: brand-voice
    severity: P3
    issue: "Order-status badges use Title Case ('Picked Up') — acceptable as status labels, but 'Picked Up' should be 'Picked up' per sentence-case rule"
    evidence_excerpt: "New / Accepted / Preparing / Ready / Picked Up / Delivering / Delivered / Cancelled"
    related_surfaces: ["vp-ux-statusbadge-labels"]
    recommendation: "Sentence case: 'Picked up'. Two-word verb phrase loses caps. Single-word statuses are fine."
    depends_on: null

  # ─────────────────────────────────────────────────────────────
  # P1 — Form-section title case (vendor-portal onboarding + menu)
  # Style guide §4: form labels are sentence case
  # ─────────────────────────────────────────────────────────────

  - finding_id: BV-032
    surface_id: vp-ux-menu-form-section-titles
    lens: brand-voice
    severity: P1
    issue: "Section titles Title Cased — 'Basic Information', 'Dietary Information', 'Preparation Details', 'Images'"
    evidence_excerpt: "Basic Information / Dietary Information / Preparation Details / Images"
    related_surfaces: ["vp-ux-menu-form-fields", "vp-onb-review-section-labels"]
    recommendation: "'Basic information', 'Dietary information', 'Preparation details', 'Images'."
    depends_on: null

  - finding_id: BV-033
    surface_id: vp-ux-menu-form-fields
    lens: brand-voice
    severity: P1
    issue: "All form labels Title Cased — 'Item Name', 'Prep Time', 'Portion Size', 'Dietary Tags', 'Allergens'"
    evidence_excerpt: "Item Name / Description / Price / Category / Prep Time / Portion Size / Serves / Dietary Tags / Allergens"
    related_surfaces: ["vp-ux-menu-form-section-titles"]
    recommendation: "Sentence case throughout: 'Item name', 'Prep time', 'Portion size', 'Dietary tags'. Style guide §4: 'noun, sentence case, no colons'."
    depends_on: null

  - finding_id: BV-034
    surface_id: vp-onb-review-section-labels
    lens: brand-voice
    severity: P1
    issue: "Onboarding review-step section titles Title Cased"
    evidence_excerpt: "Personal Information / Kitchen Details / Operations & Pricing / Documents / Policies & Agreements"
    related_surfaces: ["vp-onb-review-field-labels"]
    recommendation: "'Personal information', 'Kitchen details', 'Operations & pricing', 'Documents', 'Policies & agreements'."
    depends_on: null

  - finding_id: BV-035
    surface_id: vp-onb-review-field-labels
    lens: brand-voice
    severity: P1
    issue: "Review-page field labels Title Cased — 'Full Name', 'Kitchen Type', 'Daily Capacity', 'Delivery Radius'"
    evidence_excerpt: "Full Name / Phone / Email / Kitchen Address / Business Name / Kitchen Type / Experience / Daily Capacity / Description / Cuisines / Specialties / Prep Time / Delivery Radius / Min Order / Delivery Fee / Operating Days"
    related_surfaces: ["vp-onb-review-section-labels"]
    recommendation: "Sentence case all: 'Full name', 'Kitchen type', 'Daily capacity', 'Delivery radius', 'Min order', 'Delivery fee', 'Operating days'."
    depends_on: null

  - finding_id: BV-036
    surface_id: vp-onb-review-policy-labels
    lens: brand-voice
    severity: P1
    issue: "Policy labels Title Cased — 'Hygiene & Food Safety', 'Order & Cancellation Policy', 'Terms of Service'"
    evidence_excerpt: "Hygiene & Food Safety / Order & Cancellation Policy / Terms of Service"
    related_surfaces: ["vp-onb-review-section-labels"]
    recommendation: "'Hygiene & food safety', 'Order & cancellation policy', 'Terms of service'."
    depends_on: null

  - finding_id: BV-037
    surface_id: vp-ux-payouts-table
    lens: brand-voice
    severity: P3
    issue: "Table headers Title Cased — 'Date / Amount / Status / Method'"
    evidence_excerpt: "Date / Amount / Status / Method"
    related_surfaces: []
    recommendation: "Single-word headers in Title Case is conventional, but for consistency with sentence case elsewhere, lowercase: 'Date', 'Amount', 'Status', 'Method' already sentence case — accept as is. Flagged only as audit anchor for table-header policy decision."
    depends_on: null

  - finding_id: BV-038
    surface_id: vp-onb-docs-additional-title
    lens: brand-voice
    severity: P1
    issue: "Section title 'Additional Documents (Optional)' Title Cased; '(Optional)' uses parenthetical capitalised form repeated across many fields"
    evidence_excerpt: "Additional Documents (Optional)"
    related_surfaces: ["vp-onb-personal-avatar-label", "vp-onb-ops-min-order", "mc-checkout-note"]
    recommendation: "'Additional documents (optional)'. Drop capital O across the entire codebase: '(optional)' lowercase per sentence-case rule."
    depends_on: null

  - finding_id: BV-039
    surface_id: vp-onb-personal-avatar-label
    lens: brand-voice
    severity: P3
    issue: "'Profile Photo (Optional)' Title Cased"
    evidence_excerpt: "Profile Photo (Optional)"
    related_surfaces: ["vp-onb-docs-additional-title"]
    recommendation: "'Profile photo (optional)'."
    depends_on: null

  - finding_id: BV-040
    surface_id: vp-onb-ops-min-order
    lens: brand-voice
    severity: P3
    issue: "'Minimum Order Value (Optional)' Title Cased"
    evidence_excerpt: "Minimum Order Value (Optional) / Set to 0 for no minimum"
    related_surfaces: ["vp-onb-docs-additional-title"]
    recommendation: "'Minimum order value (optional)'."
    depends_on: null

  - finding_id: BV-041
    surface_id: vp-ux-analytics-charts
    lens: brand-voice
    severity: P1
    issue: "Chart headers Title Cased — 'Order Trends', 'Revenue Trends', 'Popular Items', 'Revenue by Category', 'Peak Hours'"
    evidence_excerpt: "Order Trends / Revenue Trends / Popular Items / Revenue by Category / Peak Hours"
    related_surfaces: ["vp-ux-earnings-sections", "vp-ux-dashboard-stats-labels"]
    recommendation: "Sentence case: 'Order trends', 'Revenue trends', 'Popular items', 'Revenue by category', 'Peak hours'."
    depends_on: null

  - finding_id: BV-042
    surface_id: vp-ux-earnings-sections
    lens: brand-voice
    severity: P1
    issue: "Section headers Title Cased — 'Daily Earnings', 'Top Selling Items', 'Recent Payouts', 'View All'"
    evidence_excerpt: "Daily Earnings / Last 14 days / Top Selling Items / No sales data yet / Recent Payouts / View All / No payouts yet / {n} orders / View full payout history"
    related_surfaces: ["vp-ux-analytics-charts"]
    recommendation: "'Daily earnings', 'Top-selling items', 'Recent payouts', 'View all'. Note 'Last 14 days' and 'No sales data yet' already correct — use as model."
    depends_on: null

  - finding_id: BV-043
    surface_id: vp-ux-notifs-section-titles
    lens: brand-voice
    severity: P1
    issue: "'Kitchen Review Status' Title Cased"
    evidence_excerpt: "Kitchen Review Status / Notifications"
    related_surfaces: ["vp-ux-notifs-admin-notes"]
    recommendation: "'Kitchen review status'."
    depends_on: null

  - finding_id: BV-044
    surface_id: vp-ux-notifs-admin-notes
    lens: brand-voice
    severity: P3
    issue: "'Admin Notes' label Title Cased"
    evidence_excerpt: "Admin Notes"
    related_surfaces: ["vp-onb-banner-admin-notes-label"]
    recommendation: "'Admin notes'."
    depends_on: null

  - finding_id: BV-045
    surface_id: vp-onb-banner-admin-notes-label
    lens: brand-voice
    severity: P3
    issue: "'Admin Notes:' uses Title Case AND trailing colon — both violate style guide (§4 form labels)"
    evidence_excerpt: "Admin Notes:"
    related_surfaces: ["vp-ux-notifs-admin-notes"]
    recommendation: "'Admin notes' (no colon, sentence case)."
    depends_on: null

  - finding_id: BV-046
    surface_id: vp-ux-notifs-status-labels
    lens: brand-voice
    severity: P1
    issue: "Status labels Title Cased — 'Pending Review', 'Info Requested'"
    evidence_excerpt: "Pending Review / Approved / Rejected / Info Requested / Cancelled"
    related_surfaces: ["vp-ux-profile-doc-statuses", "ap-chefs-badge-pending", "dp-mc-pending-verification"]
    recommendation: "'Pending review', 'Info requested'. Single-word statuses (Approved, Rejected, Cancelled) already correct."
    depends_on: null

  - finding_id: BV-047
    surface_id: vp-ux-notifs-actions
    lens: brand-voice
    severity: P1
    issue: "Action chips Title Cased — 'Update Profile', 'Update Details', 'Go to Dashboard'"
    evidence_excerpt: "Update Profile / Update Details / Go to Dashboard / Mark as read"
    related_surfaces: ["vp-ux-menu-form-actions"]
    recommendation: "'Update profile', 'Update details', 'Go to dashboard'. 'Mark as read' is the model — already correct."
    depends_on: null

  - finding_id: BV-048
    surface_id: dp-mc-pending-verification
    lens: brand-voice
    severity: P3
    issue: "'Pending Verification' Title Cased status"
    evidence_excerpt: "Pending Verification"
    related_surfaces: ["vp-ux-notifs-status-labels", "ap-chefs-badge-pending"]
    recommendation: "'Pending verification'."
    depends_on: null

  - finding_id: BV-049
    surface_id: dp-mc-action-required
    lens: brand-voice
    severity: P3
    issue: "'Action Required' Title Cased status"
    evidence_excerpt: "Action Required"
    related_surfaces: ["dp-mc-pending-verification"]
    recommendation: "'Action required'."
    depends_on: null

  # ─────────────────────────────────────────────────────────────
  # P1 — Success-toast / status format drift across apps
  # Style guide §4: success toasts are past-tense, ≤6 words, period.
  # ─────────────────────────────────────────────────────────────

  - finding_id: BV-050
    surface_id: web-ux-cart-promo
    lens: brand-voice
    severity: P1
    issue: "Promo toast 'Promo code applied! 10% off' uses exclamation mark — violates Rule 1 (Confident, not loud)"
    evidence_excerpt: "Promo code applied! 10% off"
    related_surfaces: ["vp-ux-reviews-reply"]
    recommendation: "Rewrite: 'Promo applied. 10% off.' — past-tense, period, no exclamation. Two sentences ≤6 words."
    depends_on: null

  - finding_id: BV-051
    surface_id: vp-ux-reviews-reply
    lens: brand-voice
    severity: P1
    issue: "Success toast 'Reply posted successfully' uses adverb 'successfully' — bloat per style guide success-toast formula (past-tense, ≤6 words)"
    evidence_excerpt: "Reply posted successfully"
    related_surfaces: ["web-ux-cart-promo"]
    recommendation: "Rewrite: 'Reply posted.' — past-tense, terminal period, no adverb."
    depends_on: null

  # ─────────────────────────────────────────────────────────────
  # P1 — Confirm/modal subtitle anti-patterns
  # Style guide §4: modal subtitles explain consequence; not "Are you sure"
  # ─────────────────────────────────────────────────────────────

  - finding_id: BV-052
    surface_id: vp-ux-menu-delete-confirm
    lens: brand-voice
    severity: P1
    issue: "Confirm copy 'Are you sure you want to delete this item?' is the anti-pattern explicitly banned by style guide"
    evidence_excerpt: "Are you sure you want to delete this item?"
    related_surfaces: ["vp-ux-menu-bulk-confirm"]
    recommendation: "Rewrite as consequence-first: 'Delete this menu item? Customers will no longer see it and it cannot be restored.'"
    depends_on: null

  - finding_id: BV-053
    surface_id: vp-ux-menu-bulk-confirm
    lens: brand-voice
    severity: P1
    issue: "Bulk-delete confirm 'Delete {n} items? This cannot be undone.' — first sentence is fine, second is the banned vague pattern"
    evidence_excerpt: "Delete {n} items? This cannot be undone."
    related_surfaces: ["vp-ux-menu-delete-confirm"]
    recommendation: "Rewrite consequence: 'Delete {n} menu items? Customers will no longer see them.' — concrete effect, not vague 'cannot be undone'."
    depends_on: null

  # ─────────────────────────────────────────────────────────────
  # P1 — Driver app verbosity (≤4 words rule from persona matrix)
  # ─────────────────────────────────────────────────────────────

  - finding_id: BV-054
    surface_id: mv-menuedit-price-change-banner
    lens: brand-voice
    severity: P1
    issue: "Mobile-vendor banner runs 14 words for an in-motion mobile context: 'Price changes are submitted for admin review and may take 24 hours to reflect.'"
    evidence_excerpt: "Price changes are submitted for admin review and may take 24 hours to reflect."
    related_surfaces: []
    recommendation: "Tighten per chef persona (5-12 words, operational): 'Price changes need admin review — up to 24h.' or split into title 'Price changes need approval' + helper 'Allow up to 24 hours.'"
    depends_on: null

  - finding_id: BV-055
    surface_id: dp-mc-available-deliveries-alabel
    lens: brand-voice
    severity: P2
    issue: "Driver bell aria 'Available deliveries, {n} new' is fine; bare 'Available deliveries' (2 words) preferable in glance contexts"
    evidence_excerpt: "Available deliveries, {n} new / Available deliveries"
    related_surfaces: []
    recommendation: "Keep — but ensure all driver-app aria labels stay ≤4 words. Audit driver-navigation strings against the ≤4-word rule."
    depends_on: null

  # ─────────────────────────────────────────────────────────────
  # P1 — Customer-facing tone violations
  # ─────────────────────────────────────────────────────────────

  - finding_id: BV-056
    surface_id: mc-perm-camera
    lens: brand-voice
    severity: P1
    issue: "iOS camera permission prompt 'Used to take photos' is vague and operational — should explain WHY in customer-warm tone"
    evidence_excerpt: "Used to take photos"
    related_surfaces: ["mc-perm-location", "mc-perm-faceid"]
    recommendation: "Rewrite warm + specific: 'Home Chef uses your camera to upload a photo of your delivery if needed.' Customer voice = sensory/warm, not telegraphic."
    depends_on: null

  - finding_id: BV-057
    surface_id: mc-perm-faceid
    lens: brand-voice
    severity: P2
    issue: "FaceID prompt 'Use Face ID to log in quickly' contains 'log in' — banned variant"
    evidence_excerpt: "Use Face ID to log in quickly"
    related_surfaces: ["web-ux-chef-card-favorite-toast-loggedout"]
    recommendation: "Rewrite: 'Use Face ID to sign in quickly.' (sentence case, terminal period, 'sign in' not 'log in')."
    depends_on: null

  - finding_id: BV-058
    surface_id: mc-perm-location
    lens: brand-voice
    severity: P2
    issue: "Location prompt 'Used to show your location on the delivery tracking map' starts with passive voice + clunky operational tone"
    evidence_excerpt: "Used to show your location on the delivery tracking map"
    related_surfaces: ["mc-perm-camera"]
    recommendation: "Active voice, customer-warm: 'Home Chef shows your location on the live tracking map so your driver can find you.'"
    depends_on: null

  - finding_id: BV-059
    surface_id: web-ux-loading-default
    lens: brand-voice
    severity: P2
    issue: "'Loading...' uses three dots; style guide implies terminal period for finished states only — loading ellipsis is fine BUT ASCII '...' should be Unicode ellipsis '…' for typographic refinement"
    evidence_excerpt: "Loading..."
    related_surfaces: ["vp-ux-loading-screen", "vp-onb-personal-address-loading", "dp-mc-loading-msg", "vp-auth-login-loading", "ap-auth-totp-verifying", "vp-auth-register-loading", "dp-mc-signin-loading", "dp-mc-continue-saving", "ap-auth-signing-in", "ap-exports-button"]
    recommendation: "Replace ASCII '...' with Unicode ellipsis '…' platform-wide in loading states. Aligns with 'Quietly modern' principle. ALSO standardise the loading verb: 'Loading…' is generic; consider context-specific 'Signing in…', 'Saving…' (already in some surfaces)."
    depends_on: null

  # ─────────────────────────────────────────────────────────────
  # P1 — Cross-app pickup/pick-up noun-form drift
  # Style guide §3: 'Pickup' (noun), 'Pick up' (verb)
  # ─────────────────────────────────────────────────────────────

  - finding_id: BV-060
    surface_id: vp-ux-orderstatus-badges
    lens: brand-voice
    severity: P1
    issue: "Order status 'Picked Up' uses past participle Title Case; pair surface uses 'Waiting for pickup' (noun form)"
    evidence_excerpt: "Picked Up"
    related_surfaces: ["vp-ux-orders-live-actions"]
    recommendation: "Sentence case: 'Picked up' (verb form, past tense). 'Waiting for pickup' (noun) is correct."
    depends_on: null

  # ─────────────────────────────────────────────────────────────
  # P1 — Banned vocabulary / brand-drift terms in production
  # ─────────────────────────────────────────────────────────────

  - finding_id: BV-061
    surface_id: vp-onb-kitchen-desc-field
    lens: brand-voice
    severity: P2
    issue: "Placeholder 'Describe your cooking style, what makes your food special, your signature touch...' nudges toward faux-artisanal voice ('signature touch') — borderline banned-vocab"
    evidence_excerpt: "Describe your cooking style, what makes your food special, your signature touch..."
    related_surfaces: ["vp-onb-kitchen-specialties-title"]
    recommendation: "Tighten + de-artisanal: 'Describe your kitchen — what you cook, how long you've been at it, and anything customers should know.' Drops 'signature touch'/'special' hype. Lower-case ellipsis '…'."
    depends_on: null

  - finding_id: BV-062
    surface_id: vp-onb-kitchen-specialties-title
    lens: brand-voice
    severity: P3
    issue: "Section title 'Signature Dishes & Specialties' Title Cased and 'Signature' borders artisanal vocabulary"
    evidence_excerpt: "Signature Dishes & Specialties"
    related_surfaces: ["vp-onb-kitchen-desc-field"]
    recommendation: "'Signature dishes & specialties' (sentence case). Acceptable to keep 'signature' as descriptive — but watch for stacking with other artisanal terms."
    depends_on: null

  - finding_id: BV-063
    surface_id: vp-onb-docs-kitchen-photos-title
    lens: brand-voice
    severity: P2
    issue: "Helper text 'Photos of your kitchen help build trust' is fine, but 'Kitchen Photos' Title Cased title violates sentence-case rule"
    evidence_excerpt: "Kitchen Photos / Photos of your kitchen help build trust."
    related_surfaces: []
    recommendation: "'Kitchen photos' (sentence case)."
    depends_on: null

  # ─────────────────────────────────────────────────────────────
  # P1 — Brand pronoun / impersonal-voice drift
  # Style guide §5: 'we'/'you' enforced, not 'the User'/'the Driver Partner'
  # ─────────────────────────────────────────────────────────────

  - finding_id: BV-064
    surface_id: vp-onb-policies-payout-info
    lens: brand-voice
    severity: P3
    issue: "Helper text correctly uses 'You can…' — flagged as POSITIVE model. But pair surface BV-064b uses passive 'are submitted'"
    evidence_excerpt: "You can set up your bank account or UPI details for receiving payouts from Settings after your kitchen is approved."
    related_surfaces: ["mv-menuedit-price-change-banner"]
    recommendation: "Keep — model usage. Apply same active-voice rewrite to 'Price changes are submitted for admin review' → 'We review price changes within 24 hours.'"
    depends_on: null

  - finding_id: BV-065
    surface_id: ap-platsettings-hours-defaults
    lens: brand-voice
    severity: P3
    issue: "Admin helper 'We're currently closed.' uses 'we' — but in admin context this is platform speaking about the marketplace's state, not the user. Confusing pronoun."
    evidence_excerpt: "We're currently closed."
    related_surfaces: []
    recommendation: "Clarify subject: 'Marketplace is currently closed' or 'Customers can't place orders' — admin reads platform-state, not first-person."
    depends_on: null

  # ─────────────────────────────────────────────────────────────
  # P2 — Greeting consistency (mobile-vendor only has time-based)
  # ─────────────────────────────────────────────────────────────

  - finding_id: BV-066
    surface_id: mv-dash-greeting-morning
    lens: brand-voice
    severity: P3
    issue: "Mobile-vendor uses time-based greeting 'Good morning' / 'Good afternoon' / 'Good evening' — vendor-portal web dashboard does NOT greet; cross-surface drift"
    evidence_excerpt: "Good morning"
    related_surfaces: ["mv-dash-greeting-afternoon", "mv-dash-greeting-evening", "mv-dash-greeting-fallback", "vp-ux-dashboard-stats-labels"]
    recommendation: "Decide: greet or don't greet. If keeping, vendor-portal web dashboard should match. 'Confident · Quietly modern' brand → no greeting (operational tool, not consumer)."
    depends_on: null

  - finding_id: BV-067
    surface_id: mv-dash-greeting-fallback
    lens: brand-voice
    severity: P2
    issue: "Fallback 'Chef' when no name is available — depersonalised. Better: omit greeting entirely on fallback."
    evidence_excerpt: "Chef"
    related_surfaces: ["mv-dash-greeting-morning"]
    recommendation: "Drop fallback. If name unknown, skip greeting; show stats directly."
    depends_on: null

  # ─────────────────────────────────────────────────────────────
  # P2 — Placeholder format consistency
  # ─────────────────────────────────────────────────────────────

  - finding_id: BV-068
    surface_id: vp-auth-login-email-placeholder
    lens: brand-voice
    severity: P3
    issue: "Vendor-portal uses generic 'you@example.com'; admin-portal uses brand-specific 'admin@fe3dr.com'; mobile-customer uses '+91 9876543210' as phone placeholder"
    evidence_excerpt: "you@example.com"
    related_surfaces: ["dp-mc-email-placeholder", "ap-auth-email-placeholder", "mc-profile-placeholders", "ap-staff-invite-placeholders"]
    recommendation: "Standardise: use 'you@example.com' across all login placeholders. Drop 'fe3dr.com' from admin (see BV-002). Drop 'colleague@company.com' from staff-invite — use 'name@company.com'."
    depends_on: null

  - finding_id: BV-069
    surface_id: vp-onb-kitchen-name-field
    lens: brand-voice
    severity: P3
    issue: "Placeholder uses 'Meena's Kitchen, Amma's Tiffin Service' — culturally appropriate for India-only product, well done. Flagged only to confirm consistency."
    evidence_excerpt: "Meena's Kitchen, Amma's Tiffin Service"
    related_surfaces: ["vp-onb-kitchen-specialties-placeholder", "vp-ux-menu-form-fields"]
    recommendation: "Keep — model of culturally-grounded placeholder. Audit other apps to ensure customer/vendor placeholders use India-en localised examples (Paneer Butter Masala, Masala Dosa already present)."
    depends_on: null

  - finding_id: BV-070
    surface_id: ap-staff-invite-placeholders
    lens: brand-voice
    severity: P3
    issue: "Placeholders mix forms: 'colleague@company.com', 'e.g. Operations', 'Optional message…'"
    evidence_excerpt: "colleague@company.com; Select a role...; e.g. Operations; e.g. Operations Lead; Optional message to include in the invitation email..."
    related_surfaces: ["vp-auth-login-email-placeholder"]
    recommendation: "Pattern: real example + sentence case. 'name@company.com' / 'Select a role…' / 'Operations' (no 'e.g.' clutter) / 'Operations lead' / 'Optional invite message'."
    depends_on: null

  # ─────────────────────────────────────────────────────────────
  # P2 — Ellipsis ASCII vs Unicode (typographic refinement)
  # ─────────────────────────────────────────────────────────────

  - finding_id: BV-071
    surface_id: vp-ux-menu-search-placeholder
    lens: brand-voice
    severity: P3
    issue: "Placeholder 'Search menu items...' uses ASCII three-dot; rest of style guide implies Unicode '…'"
    evidence_excerpt: "Search menu items... / All Categories"
    related_surfaces: ["web-ux-orders-search", "ap-providers-search-placeholder", "ap-orders-search-placeholder", "ap-staff-search-placeholder", "ap-users-search-placeholder", "ap-chefs-search-placeholder", "ap-approvals-search-placeholders", "web-ux-social-feed-comment"]
    recommendation: "Replace ASCII '...' with Unicode '…' across all search/comment/textarea placeholders. Single regex pass — system-wide refinement aligned with 'Quietly modern'."
    depends_on: null

  - finding_id: BV-072
    surface_id: web-ux-social-feed-comment
    lens: brand-voice
    severity: P3
    issue: "'Add a comment...' uses ASCII three-dot"
    evidence_excerpt: "Add a comment..."
    related_surfaces: ["vp-ux-menu-search-placeholder"]
    recommendation: "'Add a comment…' (Unicode ellipsis)."
    depends_on: null

  # ─────────────────────────────────────────────────────────────
  # P2 — Tab/section label drift
  # ─────────────────────────────────────────────────────────────

  - finding_id: BV-073
    surface_id: mc-tabs-labels
    lens: brand-voice
    severity: P2
    issue: "Mobile-customer tab labels 'Home / Orders / Saved / Profile'; web equivalent uses different vocab ('My Orders', etc.). 'Saved' is the bag/cart-style noun; better to align with web 'Favourites' if that's the canonical label"
    evidence_excerpt: "Home / Orders / Saved / Profile"
    related_surfaces: ["web-ux-chef-card-favorite-toast-loggedout"]
    recommendation: "Audit favourites/saved naming across customer apps. Pick one: 'Saved' (mobile-style, short) OR 'Favourites' (warm-customer, en-IN). The toast message says 'favorites' (US spelling) — also resolve en-IN/en-US drift."
    depends_on: null

  - finding_id: BV-074
    surface_id: vp-onb-stepper-review-step
    lens: brand-voice
    severity: P3
    issue: "Stepper label 'Review / Submit application' inconsistent with mobile-view title 'Review & Submit' and final button 'Submit Application'"
    evidence_excerpt: "Review / Submit application"
    related_surfaces: ["vp-onb-mobile-review-title", "vp-onb-nav-submit-app", "vp-onb-nav-review-app"]
    recommendation: "Align: stepper 'Review', button 'Submit application', mobile title 'Review & submit'. Pick consistent verbs."
    depends_on: null

  # ─────────────────────────────────────────────────────────────
  # P2 — Notification channel labels (OS-level surfaces)
  # ─────────────────────────────────────────────────────────────

  - finding_id: BV-075
    surface_id: md-mic-003
    lens: brand-voice
    severity: P2
    issue: "Android notification channel 'New Deliveries' is Title Cased — these strings appear in OS Settings; pair with 'Delivery Updates' (Title Case) and mobile-vendor 'New Orders' / 'Order Updates'"
    evidence_excerpt: "New Deliveries"
    related_surfaces: ["md-mic-004", "mv-push-channel-neworders", "mv-push-channel-orderupdates"]
    recommendation: "Android channel labels: conventional Android UI is Sentence case. Use 'New deliveries', 'Delivery updates', 'New orders', 'Order updates' — sentence case across all four channels. Channel names appear in Settings → Notifications and are user-facing."
    depends_on: null

  - finding_id: BV-076
    surface_id: mv-push-channel-neworders
    lens: brand-voice
    severity: P2
    issue: "'New Orders' Title Cased Android channel label"
    evidence_excerpt: "New Orders"
    related_surfaces: ["md-mic-003"]
    recommendation: "'New orders'."
    depends_on: null

  - finding_id: BV-077
    surface_id: mv-push-channel-orderupdates
    lens: brand-voice
    severity: P2
    issue: "'Order Updates' Title Cased Android channel label"
    evidence_excerpt: "Order Updates"
    related_surfaces: ["md-mic-003"]
    recommendation: "'Order updates'."
    depends_on: null

  - finding_id: BV-078
    surface_id: md-mic-004
    lens: brand-voice
    severity: P2
    issue: "'Delivery Updates' Title Cased"
    evidence_excerpt: "Delivery Updates"
    related_surfaces: ["md-mic-003"]
    recommendation: "'Delivery updates'."
    depends_on: null

  # ─────────────────────────────────────────────────────────────
  # P2 — Pluralization style guide compliance
  # Style guide §6: never 'item(s)'; always concrete 0/1/n
  # ─────────────────────────────────────────────────────────────

  - finding_id: BV-079
    surface_id: mc-order-card-meta
    lens: brand-voice
    severity: P1
    issue: "Mobile-customer order-card meta uses '{n} item(s)' — banned plural pattern; style guide §6 mandates concrete pluralisation"
    evidence_excerpt: "{n} item(s) • {date}"
    related_surfaces: []
    recommendation: "Replace with ICU-compatible plural: '{n, plural, one {# item} other {# items}}' or pre-rendered '1 item' / '2 items'. Never 'item(s)'."
    depends_on: null

  # ─────────────────────────────────────────────────────────────
  # P2 — Cross-app status-label drift
  # ─────────────────────────────────────────────────────────────

  - finding_id: BV-080
    surface_id: dp-mc-stripe-cells
    lens: brand-voice
    severity: P3
    issue: "Stripe state labels mix 'Enabled / Pending / Submitted / Incomplete' (sentence case) — admin-portal uses 'Connected / Disconnected / Live Mode / Test Mode' (Title Case mix). Cross-app status-label drift."
    evidence_excerpt: "Charges / Payouts / Details + Enabled / Pending / Submitted / Incomplete"
    related_surfaces: ["ap-settings-payment-statuses"]
    recommendation: "Single status-label policy: sentence case single-word OR two-word ('Live mode', 'Test mode'). Replace 'Live Mode' / 'Test Mode' with 'Live mode' / 'Test mode'."
    depends_on: null

  - finding_id: BV-081
    surface_id: ap-settings-payment-statuses
    lens: brand-voice
    severity: P2
    issue: "'Live Mode / Test Mode' Title Cased"
    evidence_excerpt: "Connected; Disconnected; Live Mode; Test Mode"
    related_surfaces: ["dp-mc-stripe-cells"]
    recommendation: "'Live mode' / 'Test mode'."
    depends_on: null

  # ─────────────────────────────────────────────────────────────
  # P3 — Misc capitalization / colon drift
  # ─────────────────────────────────────────────────────────────

  - finding_id: BV-082
    surface_id: web-ux-orderdetail-cancel-reason
    lens: brand-voice
    severity: P3
    issue: "'Cancellation reason:' uses trailing colon — style guide §4 form labels: 'no colons'"
    evidence_excerpt: "Cancellation reason:"
    related_surfaces: ["vp-onb-banner-admin-notes-label", "vp-ux-orders-live-note", "ap-auditlogs-expanded"]
    recommendation: "'Cancellation reason' (drop colon) — let layout/spacing denote label-value pairing."
    depends_on: null

  - finding_id: BV-083
    surface_id: vp-ux-orders-live-note
    lens: brand-voice
    severity: P3
    issue: "'Note: {specialInstructions}' has colon — acceptable when label runs inline with value; flag for consistency only"
    evidence_excerpt: "Note: {specialInstructions}"
    related_surfaces: ["web-ux-cart-add-note", "mc-checkout-note", "ap-auditlogs-expanded"]
    recommendation: "Keep colon for inline label-value pairs (this is the convention). Distinguish from stacked form labels (no colon). Document this in style guide §4."
    depends_on: null

  - finding_id: BV-084
    surface_id: ap-auditlogs-expanded
    lens: brand-voice
    severity: P3
    issue: "'User agent:' uses colon + inline label; 'Before' / 'After' are bare labels — inconsistent format"
    evidence_excerpt: "User agent:; Before; After"
    related_surfaces: ["vp-ux-orders-live-note"]
    recommendation: "Decide pattern. 'User agent' as section header (no colon) OR all three formatted as inline label-value pairs with colons. Today they mix."
    depends_on: null

  - finding_id: BV-085
    surface_id: ap-secsettings-session-actions
    lens: brand-voice
    severity: P3
    issue: "'Sign out everywhere' — uses 'Sign out' correctly. Flagged as POSITIVE model. Pair surfaces use 'Logout'."
    evidence_excerpt: "Sign out everywhere; Revoke session; Refresh sessions"
    related_surfaces: ["dp-mc-logout", "ap-layout-account-logout", "md-mic-001"]
    recommendation: "Keep — model. Apply same 'Sign out' to BV-004/005/006."
    depends_on: null

  - finding_id: BV-086
    surface_id: vp-ux-menu-card-aria
    lens: brand-voice
    severity: P3
    issue: "aria-labels 'Select item / Deselect item / Mark as available / Mark as unavailable' — correct sentence case. Model surface."
    evidence_excerpt: "Select item / Deselect item / Mark as available / Mark as unavailable / View item / Edit item / Delete item"
    related_surfaces: ["vp-ux-orders-live-actions"]
    recommendation: "Keep — apply same form to visible buttons (currently Title Case in BV-030)."
    depends_on: null

  - finding_id: BV-087
    surface_id: dp-mc-password-show-toggle
    lens: brand-voice
    severity: P3
    issue: "'Show password' / 'Hide password' — correct sentence-case aria pair. Model."
    evidence_excerpt: "Show password / Hide password"
    related_surfaces: ["ap-auth-totp-show-pass-aria"]
    recommendation: "Keep — model. Apply same to admin-portal aria pair (already matches)."
    depends_on: null

  # ─────────────────────────────────────────────────────────────
  # P2 — Open / Closed status across apps
  # ─────────────────────────────────────────────────────────────

  - finding_id: BV-088
    surface_id: mc-favorites-open-closed
    lens: brand-voice
    severity: P3
    issue: "Mobile-customer chef status 'Open / Closed' — consistent with chef-detail. Flagged as POSITIVE."
    evidence_excerpt: "Open / Closed"
    related_surfaces: ["mc-chef-open-closed", "ap-platsettings-hours-defaults"]
    recommendation: "Keep — model. But ensure web/customer surfaces use same exact words (not 'Available now' / 'Offline')."
    depends_on: null

  # ─────────────────────────────────────────────────────────────
  # P2 — Catering placeholder copy
  # ─────────────────────────────────────────────────────────────

  - finding_id: BV-089
    surface_id: mc-catering-placeholders
    lens: brand-voice
    severity: P3
    issue: "Catering placeholder runs long and is comma-stuffed: 'Any specific requirements, dietary restrictions, menu preferences...'"
    evidence_excerpt: "Any specific requirements, dietary restrictions, menu preferences..."
    related_surfaces: ["mc-checkout-note"]
    recommendation: "Tighten + Unicode ellipsis: 'Any requirements — diet, menu preferences, occasion…'"
    depends_on: null

  - finding_id: BV-090
    surface_id: mc-catering-view-quote-hint
    lens: brand-voice
    severity: P3
    issue: "Quote-ready hint 'Quotes available — view details' uses em-dash correctly (model). Flagged as POSITIVE."
    evidence_excerpt: "Quotes available — view details"
    related_surfaces: []
    recommendation: "Keep — model usage of em-dash + sentence case."
    depends_on: null

  # ─────────────────────────────────────────────────────────────
  # P2 — Currency / money copy
  # ─────────────────────────────────────────────────────────────

  - finding_id: BV-091
    surface_id: mc-catering-budget-display
    lens: brand-voice
    severity: P3
    issue: "'Budget: ₹{n}' — symbol-first, no space (matches style guide §3). Inline label with colon. Model."
    evidence_excerpt: "Budget: ₹{n}"
    related_surfaces: ["mc-chef-delivery-meta"]
    recommendation: "Keep — model usage. Verify all ₹ instances use no space, never 'Rs.' or 'INR'."
    depends_on: null

  - finding_id: BV-092
    surface_id: mc-chef-delivery-meta
    lens: brand-voice
    severity: P3
    issue: "'Free delivery / ₹{n} delivery' inconsistent pattern. Better single form."
    evidence_excerpt: "Free delivery / ₹{n} delivery"
    related_surfaces: ["mc-catering-budget-display"]
    recommendation: "Use 'Free delivery' OR 'Delivery ₹{n}' — pick one shape, apply to both branches consistently. 'Delivery ₹40' parallels 'Free delivery' better than '₹40 delivery'."
    depends_on: null

  # ─────────────────────────────────────────────────────────────
  # P3 — Audit-log + pagination patterns
  # ─────────────────────────────────────────────────────────────

  - finding_id: BV-093
    surface_id: ap-auditlogs-pagination
    lens: brand-voice
    severity: P3
    issue: "Pagination 'Page {n} of {m} · {n} total' — model. Compare to BV-094 which lacks bullet."
    evidence_excerpt: "Page {n} of {m} · {n} total"
    related_surfaces: ["ap-chefs-pagination-context", "ap-users-pagination"]
    recommendation: "Keep — use middot · as separator across all paginations. Apply to 'Page {n} of {m} ({n} kitchens)' in chefs page (currently parens — drift)."
    depends_on: null

  - finding_id: BV-094
    surface_id: ap-chefs-pagination-context
    lens: brand-voice
    severity: P3
    issue: "'Page {n} of {m} ({n} kitchens)' uses parens; audit-logs uses middot — inconsistent"
    evidence_excerpt: "Page {n} of {m} ({n} kitchens)"
    related_surfaces: ["ap-auditlogs-pagination", "ap-users-pagination"]
    recommendation: "'Page {n} of {m} · {n} kitchens' — match audit-logs middot pattern."
    depends_on: null

  # ─────────────────────────────────────────────────────────────
  # P2 — Verb + noun drift in CTAs (View All, View Details, View user details)
  # ─────────────────────────────────────────────────────────────

  - finding_id: BV-095
    surface_id: ap-dashboard-recent-viewall
    lens: brand-voice
    severity: P3
    issue: "'View all' (admin dashboard) is sentence case correct; vendor-portal earnings uses 'View All' Title Case. Drift."
    evidence_excerpt: "View all"
    related_surfaces: ["vp-ux-earnings-sections"]
    recommendation: "Keep 'View all'. Fix vendor-portal earnings to match."
    depends_on: null

  - finding_id: BV-096
    surface_id: ap-users-action-view-aria
    lens: brand-voice
    severity: P3
    issue: "aria-label 'View user details' is sentence case correct"
    evidence_excerpt: "View user details"
    related_surfaces: ["ap-dashboard-recent-viewall", "vp-ux-dashboard-chart-total"]
    recommendation: "Keep — model. 'View details' (vendor-portal chart footer) is also fine."
    depends_on: null

  # ─────────────────────────────────────────────────────────────
  # P3 — Verbose/redundant labels
  # ─────────────────────────────────────────────────────────────

  - finding_id: BV-097
    surface_id: web-ux-orders-search
    lens: brand-voice
    severity: P3
    issue: "Two strings 'Search your orders' (aria) and 'Search orders...' (placeholder) — pronoun mix, ASCII ellipsis"
    evidence_excerpt: "Search your orders / Search orders..."
    related_surfaces: ["ap-orders-search-placeholder"]
    recommendation: "Standardise: 'Search orders' aria, 'Search orders…' placeholder. Drop 'your' (already implied)."
    depends_on: null

  - finding_id: BV-098
    surface_id: vp-ux-notifs-sla-line
    lens: brand-voice
    severity: P3
    issue: "'Typically reviewed within 24-48 hours' — vendor-portal vs mobile-vendor 'may take 24 hours to reflect' (BV-054). Two different SLAs cited."
    evidence_excerpt: "Typically reviewed within 24-48 hours"
    related_surfaces: ["mv-menuedit-price-change-banner"]
    recommendation: "Reconcile SLA across surfaces. Pick one window (24h OR 24-48h) and apply everywhere review SLA is shown."
    depends_on: null

  # ─────────────────────────────────────────────────────────────
  # Total: 98 findings
  # Severity distribution:
  #   P0 — 9 (cross-surface trust breaks: brand-name drift, sign-in/log-out drift)
  #   P1 — 31 (Title Case button/section drift, modal anti-patterns, success-toast drift, plural pattern, customer-permission tone)
  #   P2 — 18 (notification-channel drift, location/camera permission tone, pluralisation, status drift)
  #   P3 — 40 (ellipsis, colons, pagination separators, positive-model surfaces, minor drift)
  #
  # Top cross-app drift patterns (counted across findings):
  #   1. Title Case buttons (sentence-case rule §4 violations) — 28 surfaces affected
  #      → All vendor-portal + admin-portal CTAs need sentence-casing
  #   2. Sign in / Login / Logout / Sign out drift — 6 surfaces (web, vp, dp, ap, md)
  #      → Standardise to 'Sign in' / 'Sign out' platform-wide
  #   3. ASCII '...' vs Unicode '…' ellipsis — 12+ surfaces
  #      → Single regex pass to swap globally
  #   4. Brand-identity drift 'Fe3dr' vs 'Home Chef' — delivery-portal + admin-portal placeholder
  #      → P0; affects user trust at entry surfaces
  #   5. Form-label colons + Title Case — onboarding + menu-form
  #      → Style guide §4: noun, sentence case, no colons
  #   6. 'Item(s)' banned plural pattern — mobile-customer
  #      → ICU plural rendering required
  #   7. Greeting drift — mobile-vendor greets, vendor-portal web doesn't
  #      → Decide: greet or don't, apply consistently per persona
  #   8. Notification channel Title Case — mobile-vendor + mobile-delivery
  #      → All OS-level Android channels should be sentence case
```
