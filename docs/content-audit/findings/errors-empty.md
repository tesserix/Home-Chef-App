# Findings — Errors & Empty States

Category: errors-empty
Lenses applied: technical-writer, legal, business-analyst, brand-voice
Inventory slice size: 282 surfaces
Total findings: 401

## Severity summary

| Severity | TW | Legal | BA | BV | Total |
|---|---|---|---|---|---|
| P0 | 5 | 13 | 6 | 13 | 37 |
| P1 | 13 | 27 | 18 | 14 | 72 |
| P2 | 74 | 29 | 30 | 24 | 157 |
| P3 | 73 | 19 | 16 | 27 | 135 |

## Cross-lens consensus surfaces (flagged by multiple lenses)

Surfaces flagged by 4 lenses (highest priority — every lens agreed there's a problem):

- **`api-error-stripe-connect`** — flagged by TW(P0), Legal(P0), BA(P0), BV(P0) — raw API path (`/chef/stripe/connect`) leaked verbatim to chef users
- **`api-error-payment-config`** — flagged by TW(P0), Legal(P0), BA(P0), BV(P1) — "Stripe gateway not configured by platform admin" leaks internal infra state to customer
- **`api-error-order-cancel-stage`** — flagged by TW(P0), Legal(P0), BA(P1), BV(P1) — "Order cannot be cancelled at this stage" has no refund implication or recovery path
- **`api-error-upload-size`** — flagged by TW(P0), Legal(P2), BA(P2), BV(P0) — "10MB" vs "5 MB" vs "5MB" — three formats of the same unit
- **`api-error-auth-suspended`** — flagged by TW(P1), Legal(P0), BA(P0), BV(P2) — "Account is suspended" with no reason, appeal pathway, or Grievance Officer contact
- **`mc-checkout-errors`** — flagged by TW(P1), Legal(P0), BA(P0), BV(P1, P2) — payment-timeout copy on money-bearing mobile surface
- **`web-err-error-boundary`** — flagged by TW(P1), Legal(P1), BA(P1), BV(P0) — error-boundary fallback drifts across 4 apps; "We've been notified" is unverified trust claim
- **`web-ux-cart-empty`** — flagged by TW(P1), Legal(P3), BA(P1), BV(P0, P2) — cart-empty copy drifts across web/mobile/cart-sheet with chatty "Looks like you haven't…" phrasing
- **`api-error-auth-unauthorized`** — flagged by TW(P1), Legal(P2), BA(P3), BV(P0) — four variants ("Unauthorized" / "unauthorized" / "Authentication required" / "Token required")
- **`api-error-delivery-already-active`** — flagged by TW(P1), Legal(P1), BA(P3), BV(P1) — snake_case slug `no_active_delivery` surfaced as user-facing error
- **`api-error-delivery-online`** — flagged by TW(P1), Legal(P1), BA(P3), BV(P2) — "Partner" admin-vocab leaks into driver-facing errors

Surfaces flagged by 3 lenses (also high-priority rewrites):

- **`api-error-payment-refund`** — flagged by TW(P0), Legal(P0), BA(P2) — refund constraints leak internals; no refund timeline or policy surfaced
- **`ap-analytics-coming-soon`** — flagged by Legal(P1), BA(P0), BV(P0) — "Chart coming soon" placeholder shipped to paying admin operators
- **`web-err-login-generic`** — flagged by TW(P1), BA(P2), BV(P0, P1) — "Something went wrong. Please try again." duplicated verbatim across 6 surfaces
- **`api-error-generic-failed`** — flagged by Legal(P1), BA(P2), BV(P0) — 100+ "Failed to X" variants leak to UI with no translation layer
- **`api-error-validation-required`** — flagged by Legal(P2), BA(P3), BV(P0) — 18 variants of "X is required" with inconsistent field-name casing (`chefId` vs `country`)
- **`api-error-upload-type`** — flagged by Legal(P2), BA(P2), BV(P1) — 5 variants of "Invalid file type" with different allowed lists
- **`api-error-auth-reset-expired`** — flagged by Legal(P2), BA(P2), BV(P2) — three near-duplicate reset-token variants; no "resend reset link" path
- **`api-error-auth-not-verified`** — flagged by Legal(P2), BA(P2), BV(P2) — "Account not verified" vs "Account not verified yet" in same handler
- **`api-error-chef-not-accepting`** — flagged by Legal(P2), BA(P1), BV(P2) — no alternative chef suggestion; dead end for customer
- **`vp-onb-banner-rejected-body`** — flagged by Legal(P0), BA(P1), BV(P3) — chef onboarding rejection lacks due-process language and concrete remedy
- **`web-err-checkout-no-addresses`** — flagged by TW(P1), BA(P2), BV(P3) — empty saved-addresses on checkout uses chatty "don't have any saved...yet"
- **`mc-favorites-empty`** — flagged by Legal(P3), BA(P1), BV(P1, P3) — exclamation mark + emoji on customer favorites empty state
- **`dp-err-access-denied`** — flagged by TW(P1), Legal(P2), BA(P1) — security-evoking "Access denied" for likely role mismatch on driver portal
- **`ap-providerdetail-dialog-bodies`** — flagged by Legal(P1), BA(P3), BV(P2, P3) — three "Are you sure you want to..." confirmation dialogs (banned anti-pattern)
- **`ap-secsettings-apikey-empty`** — flagged by Legal(P3), BA(P3), BV(P3) — "No keys yet." under-informative for first-time API-key admin
- **`md-emp-007`** — flagged by Legal(P3), BA(P2), BV(P2) — fleet-management lock-screen body too long for driver glanceable copy
- **`dp-empty-active-body`** — flagged by Legal(P3), BA(P2), BV(P2) — driver active-delivery empty state too conversational for telegraphic persona

## Technical Writer findings

```yaml
# Technical Writer lens findings — errors-empty category
# Auditor: TW lens agent
# Date: 2026-05-13
# Style guide: docs/content-audit/STYLE-GUIDE.md
# Brief: docs/content-audit/lens-briefs/technical-writer.md
# Slice size: 282 rows. Approach: P0/P1 per-row; P2/P3 grouped by pattern where appropriate.
#
# Formula references (from STYLE-GUIDE.md §4):
#   Errors:        "What happened → What to do." Two sentences max. No blame. No "Oops!".
#   Empty states:  "Why it's empty → One action."
#   Success toasts: past tense, ≤6 words, period.
#   Buttons:       verb-first, ≤3 words, sentence case.

findings:

  # =====================================================================
  # P0 — Factually misleading / commitment-breaking error copy
  # =====================================================================

  - finding_id: TW-001
    surface_id: api-error-payment-config
    lens: technical-writer
    severity: P0
    issue: "Customer-facing payment error leaks an internal infra state ('Stripe gateway not configured', 'Payment gateway not configured by platform admin') that the customer cannot act on. The copy blames the platform admin and gives the customer no recovery path. Violates 'what happened → what to do' and exposes implementation detail."
    evidence_excerpt: "Payment gateway not configured / Stripe gateway not configured / Stripe gateway not configured by platform admin"
    recommendation: "Replace with a customer-safe message + recovery: 'Payments are temporarily unavailable. Try again in a few minutes or contact support.' Log the gateway-specific detail server-side only. Never surface 'platform admin' to a customer."
    depends_on: null

  - finding_id: TW-002
    surface_id: api-error-stripe-connect
    lens: technical-writer
    severity: P0
    issue: "Error string exposes a literal API path to the end user: 'No Stripe account — call /chef/stripe/connect first'. This is developer-console copy reaching a chef's UI. Factually misleading because the chef cannot 'call' an endpoint — they need to tap a button."
    evidence_excerpt: "No Stripe account — call /chef/stripe/connect first / No Stripe account — call /delivery/stripe/connect first"
    recommendation: "Rewrite as user action: 'Connect your Stripe account to receive payouts. Open Settings → Payouts → Connect Stripe.' Remove the API path entirely. Same fix for the delivery variant."
    depends_on: null

  - finding_id: TW-003
    surface_id: api-error-order-cancel-stage
    lens: technical-writer
    severity: P0
    issue: "'Order cannot be cancelled at this stage' tells the customer NO about a refund-bearing action without saying what the stage is, what stage allows cancellation, or whether they will be charged. For a money-bearing flow this is misleading because it implies cancellation is impossible without explaining the time/state window or refund consequence."
    evidence_excerpt: "Order cannot be cancelled at this stage"
    recommendation: "Be specific: 'Orders can be cancelled until the chef starts preparing. Contact support to request a refund.' Cite the cancellation cutoff. Link to refund policy."
    depends_on: null

  - finding_id: TW-004
    surface_id: api-error-payment-refund
    lens: technical-writer
    severity: P0
    issue: "Refund-constraint strings expose internal rules to the user but never communicate the refund timeline or refund policy. 'Can only refund completed payments' and 'Refund amount cannot exceed order total' are accurate but read as developer assertions, not user-facing copy. For a refund-bearing flow this risks commitment ambiguity (is the customer getting a refund? when?)."
    evidence_excerpt: "Can only refund completed payments / Refund amount cannot exceed order total / Failed to process refund / Payment not captured"
    recommendation: "Two fixes: (1) Rephrase to user language — 'We can only refund payments that have completed.' (2) Pair with refund SLA copy: 'Refunds reach your card in 5-7 business days.' Defer the dev-style version to logs."
    depends_on: null

  - finding_id: TW-005
    surface_id: api-error-upload-size
    lens: technical-writer
    severity: P0
    issue: "Four near-identical 'file too large' errors use three different size formats: '10MB', '5 MB', '5MB'. Same code path, different units shown to the same user. This is a numerals-formatting violation (style guide §6 mandates consistent unit formatting) and risks customer confusion about the actual limit."
    evidence_excerpt: "File too large (max 10MB) / File too large. Maximum 5 MB. / Profile photo too large (max 5MB) / Each image must be under 5 MB"
    recommendation: "Lock one format: 'File too large. Max 5 MB.' Use the space (5 MB, not 5MB) consistently. Add a shared constant for upload limits and reference one helper to format size + render the error. Audit fix should also de-duplicate the four variants into one helper."
    depends_on: null

  # =====================================================================
  # P1 — Conversion-critical voice drift / blocking confusion
  # =====================================================================

  - finding_id: TW-006
    surface_id: web-err-login-generic
    lens: technical-writer
    severity: P1
    issue: "'Something went wrong. Please try again.' on the customer login surface is conversion-critical voice drift. It violates the error formula (no 'what happened' — completely opaque) and appears on a login page where bounce risk is high. Same string is duplicated across HomePage, BrowseChefsPage, ChefDetailPage, LoginPage."
    evidence_excerpt: "Something went wrong. Please try again."
    recommendation: "Pick a more specific message when possible. When truly generic, use 'We couldn't sign you in. Try again, or reset your password.' For non-login surfaces use 'We couldn't load this. Refresh to try again.' Drop 'Please' — it's filler. Centralize the message in i18n key so the four duplicates become one."
    depends_on: null

  - finding_id: TW-007
    surface_id: dp-err-generic-auth
    lens: technical-writer
    severity: P1
    issue: "Driver login generic error 'Something went wrong. Please try again.' is too verbose for driver tone (≤4 words target per persona matrix) AND gives the driver no recovery hint on a glance-critical login screen. A driver who can't sign in cannot accept work."
    evidence_excerpt: "Something went wrong. Please try again."
    recommendation: "Driver-targeted rewrite: 'Sign-in failed. Try again.' Keep at 4 words. Use the same string across driver-portal + mobile-delivery to prevent drift."
    depends_on: TW-006

  - finding_id: TW-008
    surface_id: dp-err-access-denied
    lens: technical-writer
    severity: P1
    issue: "'Access denied. Please check your credentials and try again.' on driver login mixes a security-flavored phrase ('Access denied' implies the user is unauthorized) with credential-recovery guidance (implies bad password). The two halves contradict — is the account blocked, or did they fumble the password? Conversion-critical on a sign-in surface."
    evidence_excerpt: "Access denied. Please check your credentials and try again."
    recommendation: "Pick one: if credentials are wrong → 'Invalid email or password. Try again.' If account is blocked → 'This account isn't approved for driver access. Contact support.' Choose based on the API response code; never show 'Access denied' generically."
    depends_on: null

  - finding_id: TW-009
    surface_id: ap-auth-error-access-denied
    lens: technical-writer
    severity: P1
    issue: "Admin login surface shows 'Access denied. Only administrators can sign in to this portal.' This leaks the access policy to anyone reaching the URL, including potential attackers, AND uses 'Access denied' on a login surface where it competes with credential-error messages."
    evidence_excerpt: "Access denied. Only administrators can sign in to this portal."
    recommendation: "Drop the policy explanation from the public error. Use a uniform credential message: 'We couldn't sign you in. Check your email and password.' Log the RBAC denial server-side. (Brand voice + security consideration; defer security framing to Legal lens.)"
    depends_on: TW-008

  - finding_id: TW-010
    surface_id: web-err-error-boundary
    lens: technical-writer
    severity: P1
    issue: "Top-level error boundary copy is 35 words and uses three sentences ('Something broke on our end. We've been notified and we're looking into it...') — violates customer-facing 25-word max sentence rule AND the two-sentence error formula. The 'we've been notified' line is a promise the platform cannot verify in real time."
    evidence_excerpt: "Unexpected error / Something broke on our end. / We've been notified and we're looking into it... / Try again / Refresh page"
    recommendation: "Cut to formula: 'Something broke. Refresh the page, or try again in a moment.' Drop 'we've been notified' (unverifiable commitment). One CTA, not two — 'Refresh page' is the universal recovery; 'Try again' is redundant. Total ≤15 words."
    depends_on: null

  - finding_id: TW-011
    surface_id: web-ux-cart-empty
    lens: technical-writer
    severity: P1
    issue: "Empty cart copy is 28 words across two sentences and uses chatty filler ('Looks like you haven't added any items yet'). Violates customer max-25-word sentence rule and the empty-state formula 'why → one action' (the 'why' here is six redundant words)."
    evidence_excerpt: "Your cart is empty / Looks like you haven't added any items yet. Browse our home chefs..."
    recommendation: "Cut to: 'Your cart is empty. Browse chefs near you.' Total 8 words. Keep the 'Browse chefs' CTA visible separately as the single action."
    depends_on: null

  - finding_id: TW-012
    surface_id: web-ux-favorites-loggedout
    lens: technical-writer
    severity: P1
    issue: "'Log in to see your favorites' uses banned 'Log in' verb (style guide §3: always 'Sign in')."
    evidence_excerpt: "Log in to see your favorites"
    recommendation: "Replace with 'Sign in to see your favorites.' Audit fix should grep for 'Log in' / 'Login' across all surfaces."
    depends_on: null

  - finding_id: TW-013
    surface_id: mc-checkout-errors
    lens: technical-writer
    severity: P1
    issue: "Payment timeout error on customer mobile checkout is 14 words and verbose for a payment failure moment: 'Payment confirmation timed out. Check your order history to confirm status.' The user is in a high-anxiety post-payment state — they want a yes/no, not navigation instructions."
    evidence_excerpt: "Payment confirmation timed out. Check your order history to confirm status."
    recommendation: "Tighten + add an in-place action: 'Payment confirmation timed out. We'll update your order when it clears.' Then auto-route to order history rather than telling the user to navigate. Two sentences max, ≤15 words."
    depends_on: null

  - finding_id: TW-014
    surface_id: api-error-auth-suspended
    lens: technical-writer
    severity: P1
    issue: "'Account is suspended' gives no reason and no contact path. For a sign-in blocker on a money-bearing platform this leaves the user dead-ended on the login screen."
    evidence_excerpt: "Account is suspended"
    recommendation: "Pair with recovery: 'This account is suspended. Contact support@homechef.in to appeal.' Two sentences, formula-compliant, gives the user one action."
    depends_on: null

  - finding_id: TW-015
    surface_id: api-error-delivery-already-active
    lens: technical-writer
    severity: P1
    issue: "Error payload includes a raw snake_case slug ('no_active_delivery') alongside a human string ('You already have an active delivery'). The slug is reaching the user-facing layer per inventory note. Driver tone matrix expects ≤4 words imperative — the slug also violates that."
    evidence_excerpt: "You already have an active delivery / no_active_delivery"
    recommendation: "Keep the slug as an internal code for client-side routing; never display it. Display string: 'You already have a delivery in progress.' (≤8 words, imperative-state)."
    depends_on: null

  - finding_id: TW-016
    surface_id: api-error-delivery-online
    lens: technical-writer
    severity: P1
    issue: "Driver-facing errors use 'Partner' to refer to the driver ('Partner has reached maximum concurrent deliveries', 'Partner is not verified or active') but the driver app addresses the user as 'you'. Voice mismatch on the same surface."
    evidence_excerpt: "Partner has reached maximum concurrent deliveries / Partner is not verified or active"
    recommendation: "Switch to second person: 'You've reached your delivery limit.' / 'Your account isn't verified yet.' Per style guide, 'Partner' is customer-facing-only ('Delivery partner'); driver app uses 'driver' or 'you'."
    depends_on: null

  - finding_id: TW-017
    surface_id: api-error-auth-unauthorized
    lens: technical-writer
    severity: P1
    issue: "Four variants of the same auth-failure concept: 'Unauthorized' / 'unauthorized' / 'Authentication required' / 'Token required'. The lowercase 'unauthorized' is especially poor — looks like raw HTTP status text leaking through."
    evidence_excerpt: "Unauthorized / unauthorized / Authentication required / Token required"
    recommendation: "Pick one user-facing message: 'Sign in to continue.' Use a single error-key (AUTH_REQUIRED) at the API layer; the UI maps to the friendly string. 'Token required' is internal-only; never surface."
    depends_on: null

  - finding_id: TW-018
    surface_id: web-err-checkout-no-addresses
    lens: technical-writer
    severity: P1
    issue: "Empty-state copy on a money-bearing surface is wordy and uses chatty 'don't have any saved...yet' phrasing: 'You don't have any saved addresses yet. Add one to continue.' At checkout, the customer is task-focused; trim to formula."
    evidence_excerpt: "You don't have any saved addresses yet. Add one to continue."
    recommendation: "Cut to: 'No saved addresses. Add a delivery address to continue.' Pair with a prominent 'Add address' button (one action)."
    depends_on: null

  # =====================================================================
  # P2 — Microcopy formula violations grouped by pattern
  # =====================================================================

  - finding_id: TW-019
    surface_id: pattern-error-missing-what-to-do
    lens: technical-writer
    severity: P2
    issue: "Pattern: ~40 error strings give only the 'what happened' half of the error formula — no 'what to do' recovery action. Includes load-failures across vendor-portal, delivery-portal, mobile-vendor, mobile-delivery dashboards, earnings, analytics, profile, menu, reviews, settings."
    evidence_excerpt: "vp-ux-earnings-error 'Unable to load earnings' | vp-ux-menu-error-state 'Failed to load menu' | mv-dash-fail-error 'Failed to load dashboard' | mv-menu-load-error 'Failed to load menu' | mv-analytics-load-error 'Failed to load analytics' | mv-earnings-load-error 'Failed to load earnings' | mv-profile-load-error 'Failed to load profile' | mv-reviews-load-error 'Failed to load reviews' | mv-settings-load-error 'Failed to load settings' | md-err-001..009 'Failed to load X' | ap-users-toast-fail-suspend / -activate"
    recommendation: "Apply formula 'what happened → what to do' as a template: 'Couldn't load {thing}. Tap retry, or check your connection.' Pair every Failed-to-load error with a visible Retry button (most already exist) AND a single helper sentence. Create a shared <ErrorState> component with mandatory `action` prop so the formula is enforced at the type level."
    depends_on: null

  - finding_id: TW-020
    surface_id: pattern-error-failed-to-x
    lens: technical-writer
    severity: P2
    issue: "API surfaces ~100 variants of 'Failed to X' (accept quote, add comment, add favorite, cancel order, create address, etc.). Vague, no remediation, generic. Inventory note explicitly flags this as 'should generally not surface to end users without translation'."
    evidence_excerpt: "Failed to accept quote / Failed to add comment / Failed to cancel order / Failed to create address ... ~100 variants"
    recommendation: "Two-tier approach: (1) API returns a stable error code + technical message. (2) UI maps to one of ~10 user-facing templates: 'Couldn't {verb}. Try again.' Where retry isn't the right action ('cancel order' may need support), the template includes 'Contact support if it keeps failing.' Centralize in `apps/*/src/shared/lib/error-messages.ts`."
    depends_on: TW-019

  - finding_id: TW-021
    surface_id: pattern-error-oops-uhoh
    lens: technical-writer
    severity: P2
    issue: "Style-guide §4 explicitly bans 'Oops!' / 'Uh oh!' patterns. Repo currently shows no 'Oops!' but does use the related blame-deflecting pattern 'Something went wrong / Something broke on our end' across error boundaries and generic toasts."
    evidence_excerpt: "web-err-login-generic / web-err-generic-retry / vp-auth-login-generic-error / dp-err-generic-auth / ap-auth-error-generic / mc-onb-step3-error / vp-err-boundary-fallback 'Something went wrong' / web-err-error-boundary 'Something broke on our end' / mv-onb-review-submit-error-body 'Submission failed. Please try again.'"
    recommendation: "Reserve 'Something went wrong' for cases where the API truly returned nothing — and even then pair with action ('Refresh, or try again in a moment.'). Where the failure is known (network, server 500, validation), use specific copy. Audit fix: replace blanket 'Something went wrong' with `getFriendlyError(error)` helper that returns the right specific string."
    depends_on: TW-006

  - finding_id: TW-022
    surface_id: pattern-empty-no-action
    lens: technical-writer
    severity: P2
    issue: "Pattern: ~25 empty states give 'why it's empty' but not the one action (formula §4). Common across vendor-portal, mobile-vendor, mobile-delivery, admin-portal lists."
    evidence_excerpt: "vp-ux-dashboard-recent-empty 'No recent orders' | vp-ux-reviews-empty 'No reviews yet' | mv-orders-history-empty 'No order history yet' | mv-earnings-no-payout-account 'No payout account configured' | dp-empty-history-body 'Your completed deliveries will appear here.' | ap-chefs-empty 'No chefs found' | ap-orders-empty 'No orders found' | ap-delivery-empty 'No deliveries found' | ap-providers-empty 'No delivery providers found' | ap-users-empty 'No users found' | ap-staff-empty-invites 'No invitations found' | ap-staff-empty-members 'No staff members found' | dp-empty-partner-docs 'No documents uploaded.' | md-emp-005 'No partners found.'"
    recommendation: "For every empty list, define a single primary action below the empty-state text. Templates: 'No orders yet. Orders will appear here as customers place them.' (passive) is fine only when no user action exists; otherwise pair with action ('No reviews yet. Invite customers to review after delivery.'). For admin filtered-empty states, action = 'Clear filters'."
    depends_on: null

  - finding_id: TW-023
    surface_id: pattern-empty-state-chatty
    lens: technical-writer
    severity: P2
    issue: "Pattern: several empty states use the chatty 'Looks like...' / 'It's lonely...' tone explicitly banned in style guide §4."
    evidence_excerpt: "web-ux-cart-empty 'Looks like you haven't added any items yet.' | mc-favorites-empty 'No saved chefs yet / Tap the heart on any chef to save them!' | dp-empty-active-body 'Check available deliveries to pick up a new order.'"
    recommendation: "Strip 'Looks like'. Drop exclamation marks. Convert to formula: 'No favorites yet. Tap the heart on a chef to save them.' (no '!'). 'Check available deliveries' is fine in spirit but use imperative ≤4 words for driver: 'Open the Available tab to start.'"
    depends_on: null

  - finding_id: TW-024
    surface_id: pattern-button-too-long
    lens: technical-writer
    severity: P2
    issue: "Pattern: buttons inside error/empty surfaces exceed the 3-word, sentence-case rule (§4)."
    evidence_excerpt: "vp-ux-menu-empty 'Add Your First Item' (4 words, Title Case) | vp-err-boundary-fallback 'Try Again' (Title Case) | web-err-error-boundary 'Try again' + 'Refresh page' (two CTAs) | dp-empty-history-body / dp-empty-active-body / dp-empty-available-body (CTA implied — actual buttons live elsewhere; check)"
    recommendation: "Rule: verb-first, ≤3 words, sentence case. Fixes: 'Add Your First Item' → 'Add dish' or 'Add item'. 'Try Again' → 'Try again' (already correct words, fix capitalization). For error boundaries, pick ONE primary CTA — 'Refresh page' is the canonical recovery."
    depends_on: null

  - finding_id: TW-025
    surface_id: pattern-please-padding
    lens: technical-writer
    severity: P2
    issue: "Pattern: ~30+ validation errors begin with 'Please' as filler. 'Please' adds no information and inflates length, especially harmful for driver tone matrix (≤4 words target)."
    evidence_excerpt: "web-err-register-validation 'Please fill in all fields' | vp-auth-login-missing-fields 'Please enter your email and password' | vp-onb-validate-required-toast 'Please fill in all required fields' | vp-onb-policies-required-toast 'Please accept all required policies' | dp-err-credentials-empty | dp-err-step1-required | dp-err-payout-bank | dp-err-payout-upi | dp-err-step5-terms | mv-onb-policies-* | api-error-chef-onboarding 'Please complete...' (5 variants) | api-error-tos 'You must accept the terms and conditions'"
    recommendation: "Drop 'Please' from validation errors — they are imperative state messages, not requests. 'Please fill in all required fields' → 'Fill in all required fields.' / 'Required field missing.' Style guide §1 Rule 2 (plain English) prefers the shorter form. Keep 'Please' only when explicit politeness is warranted (e.g., support contact prompts)."
    depends_on: null

  - finding_id: TW-026
    surface_id: pattern-toast-non-past-tense
    lens: technical-writer
    severity: P2
    issue: "Style guide §4 toast formula: past tense, ≤6 words, period. Several error/empty toasts use non-past-tense or run over six words. Cross-checked because failure toasts also fall under the toast format rule."
    evidence_excerpt: "vp-onb-save-failure-toast 'Failed to save your details. Please try again.' (9 words, 2 sentences) | dp-err-step3-upload 'Upload failed. Please try again.' (5 words; OK length, but 'Please' filler) | mv-menuedit-photo-upload-fail 'Failed to upload photo. Please try again.' (7 words) | mv-checkout-errors 'Order creation failed. Please try again.' | api-error-chef-onboarding 'Failed to submit onboarding. Please try again.'"
    recommendation: "Keep toast formula. Failure toasts: 'Couldn't save. Try again.' (4 words) / 'Upload failed.' (2 words). Drop 'Please' (see TW-025). Audit fix: extract a `failureToast(thing: string)` helper that produces 'Couldn't {thing}. Try again.'"
    depends_on: TW-025

  - finding_id: TW-027
    surface_id: pattern-period-inconsistency
    lens: technical-writer
    severity: P2
    issue: "Pattern: errors and empty-state strings inconsistently use trailing periods within the same surface family. Style guide §4 success-toast formula says period; errors are 'two sentences max' implying punctuation should be consistent. Currently some say 'No reviews yet' (no period) and others say 'No documents uploaded.' (period), often in the same app."
    evidence_excerpt: "vp-ux-reviews-empty 'No reviews yet' | vp-ux-dashboard-recent-empty 'No recent orders' | dp-empty-partner-docs 'No documents uploaded.' | dp-empty-partner-notfound 'Partner not found.' | dp-empty-history-body 'Your completed deliveries will appear here.' | api-error-phone-duplicate 'This phone number is already registered with another account.' (period) | api-error-favorites 'Chef is already in your favorites' (no period)"
    recommendation: "Adopt a rule: single-sentence labels and headings → no period. Sentences (≥1 verb, full clause) → period. Apply across the family. Codify in style guide §4 as an addendum and use a lint rule (e.g., regex check in i18n keys)."
    depends_on: null

  - finding_id: TW-028
    surface_id: pattern-emoji-in-empty-states
    lens: technical-writer
    severity: P2
    issue: "Mobile-customer empty states embed emoji ('🍽️', '📸', heart) in copy. Style guide does not explicitly ban emoji but §1 Rule 1 'Confident, not loud' and Rule 4 'One accent per surface' argue against decorative emoji in copy that's already paired with iconography."
    evidence_excerpt: "mc-favorites-empty 'No saved chefs yet / Tap the heart on any chef to save them!' (with heart emoji) | mc-orders-empty (with 🍽️) | mc-catering-empty (with 🍽️) | mc-social-empty (with 📸)"
    recommendation: "Move decorative emoji into the visual icon slot; keep copy emoji-free. Strip the exclamation marks. Verify with Brand Voice lens before final rewrite."
    depends_on: null

  - finding_id: TW-029
    surface_id: pattern-exclamation-overuse
    lens: technical-writer
    severity: P2
    issue: "Style guide §1 Rule 1: '≤1 exclamation per page'. Multiple empty/error strings use exclamation in mundane states."
    evidence_excerpt: "vp-ux-dashboard-pending-empty 'All caught up! / No pending orders right now' | mc-favorites-empty '... Tap the heart on any chef to save them!' | mc-orders-empty '... place your first order!'"
    recommendation: "Drop exclamation marks from empty-state copy unless genuinely celebratory (and even then ≤1 per surface). 'All caught up.' is calmer and on-brand."
    depends_on: TW-023

  - finding_id: TW-030
    surface_id: pattern-title-case-leakage
    lens: technical-writer
    severity: P2
    issue: "Pattern: Title Case bleeds into error/empty headings in vendor-portal and admin-portal where style guide expects sentence case across UI."
    evidence_excerpt: "vp-onb-banner-rejected-title 'Application Rejected' | vp-onb-banner-info-requested-title 'More Information Needed' | dp-empty-active-title 'No Active Delivery' | dp-empty-available-title 'No Available Deliveries' | dp-empty-history-title 'No Deliveries Yet' | mv-onb-docs-required-title 'Documents Required' | mv-onb-policies-policy-required-title 'Policy Required' | mv-onb-personal-validation-alert 'Validation Error' | mv-onb-review-submit-error-title 'Submission Error' | md-emp-006 'Fleet Management' | md-emp-010 'Staff Management'"
    recommendation: "Sentence case everywhere except product/brand names. 'Application Rejected' → 'Application not approved' (softer + sentence case). 'No Active Delivery' → 'No active delivery'. Add a lint rule against repeated capitalised words in i18n keys."
    depends_on: null

  - finding_id: TW-031
    surface_id: pattern-validation-shouty-cap
    lens: technical-writer
    severity: P2
    issue: "Mobile-vendor uses 'Validation Error' / 'Submission Error' as alert titles. These are dev-console titles, not user-facing. Style guide expects neutral noun-phrase headings."
    evidence_excerpt: "mv-onb-personal-validation-alert 'Validation Error' | mv-onb-review-submit-error-title 'Submission Error' | mv-profile-displayname-required (alert title 'Validation')"
    recommendation: "Drop the 'Validation' / 'Submission' labels entirely — the body says enough. If a title is required by the alert API, use the specific issue: 'Check your details' / 'Couldn't submit'."
    depends_on: null

  - finding_id: TW-032
    surface_id: pattern-zod-bare-message
    lens: technical-writer
    severity: P2
    issue: "Zod validation strings come through as bare field rules without form context: 'Name must be at least 2 characters', 'Description must be at most 500 characters', 'Price must be between ₹1 and ₹10,000'. These appear next to the field in most cases so context is OK, but: (a) some are duplicated with slightly different wording in customer vs vendor, (b) some use 'at least' / 'at most' / 'between' inconsistently."
    evidence_excerpt: "vp-ux-menu-form-validation '... Name must be at least 2 characters / Name must be under 100 characters / Description must be at least 10 characters / Description must be under 500 characters' | mv-onb-kitchen-desc-min-err 'Description must be at least 50 characters' / mv-onb-kitchen-desc-max-err 'Description must be at most 500 characters' | mv-menunew-name-err 'Name must be at least 3 characters' (vs vendor-portal 'at least 2')"
    recommendation: "Standardize verbs: 'Name needs at least N characters.' / 'Description must be under N characters.' Pick one set of bounds across customer + chef onboarding (vendor-portal says min=2 chars for name; mobile-vendor says min=3). Audit fix should align bounds across surfaces and pull validation messages from a single zod schema module."
    depends_on: null

  - finding_id: TW-033
    surface_id: pattern-snake-case-leak
    lens: technical-writer
    severity: P2
    issue: "Snake-case slugs reach user-facing copy in at least one place (already flagged in TW-015). General scan didn't find more — but the pattern is to alert authors during the audit fix: any string containing underscores in i18n keys is suspicious."
    evidence_excerpt: "api-error-delivery-already-active includes 'no_active_delivery'"
    recommendation: "Add a lint rule: i18n value strings must not contain `[a-z]_[a-z]` (rough heuristic) outside of email/URL/code-snippet contexts. Catch slug leaks before merge."
    depends_on: TW-015

  - finding_id: TW-034
    surface_id: pattern-payout-method-jargon
    lens: technical-writer
    severity: P2
    issue: "Payout-validation strings contain backend jargon: 'payoutMethod must be...', 'bankAccountNumber, bankIFSC, and bankAccountName are required for bank_transfer'. Field names are camelCase developer identifiers reaching the user."
    evidence_excerpt: "payoutMethod must be 'bank_transfer' or 'upi' / bankAccountNumber, bankIFSC, and bankAccountName are required for bank_transfer / upiId is required for upi payout method"
    recommendation: "Rephrase to label-style: 'Choose bank transfer or UPI.' / 'Account number, IFSC, and account name are required for bank transfer.' / 'Enter your UPI ID.' Never expose camelCase field names."
    depends_on: null

  - finding_id: TW-035
    surface_id: pattern-not-verified-drift
    lens: technical-writer
    severity: P2
    issue: "Two near-identical 'Account not verified' strings in the same handler file — inconsistent ('Account not verified' vs 'Account not verified yet')."
    evidence_excerpt: "Account not verified / Account not verified yet"
    recommendation: "Pick one. 'Yet' implies a future state and is friendlier — use 'Account not verified yet. Check your email for the verification link.' Add the recovery half."
    depends_on: null

  - finding_id: TW-036
    surface_id: pattern-password-incorrect-drift
    lens: technical-writer
    severity: P2
    issue: "Two strings for the same concept: 'Current password is incorrect' / 'Password is incorrect'."
    evidence_excerpt: "Current password is incorrect / Password is incorrect"
    recommendation: "Use one based on context. In change-password forms: 'Current password is incorrect.' In sign-in: deliberately keep credentials vague ('Invalid email or password'). Don't mix in the same handler."
    depends_on: null

  - finding_id: TW-037
    surface_id: pattern-reset-token-drift
    lens: technical-writer
    severity: P2
    issue: "Three near-duplicate variants of password-reset / enrollment token errors."
    evidence_excerpt: "Reset token has expired / Invalid or expired reset token / Invalid or expired enrollment token"
    recommendation: "Single template: 'This {token type} is invalid or has expired. Request a new one.' Pass token type as parameter. Add the recovery action — 'Request a new one' or 'Resend code'."
    depends_on: null

  - finding_id: TW-038
    surface_id: pattern-2fa-vague
    lens: technical-writer
    severity: P2
    issue: "Four 2FA errors are all variants of 'Failed to ...' with zero remediation. 2FA failures are high-friction security touchpoints; users need clarity."
    evidence_excerpt: "Failed to enable 2FA / Failed to start 2FA challenge / Failed to start 2FA enrollment / Invalid or expired challenge"
    recommendation: "Add recovery: 'Couldn't start 2FA. Try again, or contact support.' / 'Code expired. Start over to get a new one.' Two sentences, formula-compliant."
    depends_on: TW-019

  - finding_id: TW-039
    surface_id: pattern-uploaded-file-allowed-list-drift
    lens: technical-writer
    severity: P2
    issue: "Five variants of 'Invalid file type. Allowed: ...' with slightly different allowed lists shown to the user."
    evidence_excerpt: "Invalid file type. Allowed: JPEG, PNG, PDF. / Invalid file type. Allowed: JPEG, PNG, WebP, PDF / Invalid file type. Allowed: JPEG, PNG, WebP. / Invalid image type. Allowed: JPEG, PNG, WebP. / Profile photo must be JPEG or PNG"
    recommendation: "Pull the allowed list from a single source per upload type (document, image, profile photo). Single template: 'This file type isn't supported. Use {list}.' Always include final period. Match the format from TW-005."
    depends_on: TW-005

  - finding_id: TW-040
    surface_id: pattern-image-count-friendly-drift
    lens: technical-writer
    severity: P2
    issue: "Image-count limit errors show four variants — only one includes 'remove one before adding another' (the recovery half)."
    evidence_excerpt: "Maximum 3 images per review / Maximum 4 images per post / Maximum 5 images per menu item / Maximum 5 kitchen photos allowed. Remove one before adding another."
    recommendation: "Apply the kitchen-photos pattern to all: 'Up to {N} {thing} allowed. Remove one before adding another.' Or even tighter: 'Maximum {N} {thing}. Remove one to add a new one.'"
    depends_on: null

  - finding_id: TW-041
    surface_id: pattern-cart-empty-drift
    lens: technical-writer
    severity: P2
    issue: "Same 'cart empty' state is expressed three different ways across surfaces — drift across customer apps."
    evidence_excerpt: "web-ux-cart-empty 'Your cart is empty / Looks like you haven't added any items yet...' (28 words) | mc-checkout-empty-cart 'Your cart is empty.' (period) | mc-cartsheet-empty 'Your cart is empty' (no period)"
    recommendation: "Single i18n key: `empty.cart` = 'Your cart is empty. Browse chefs near you.' Apply in all three places. Decide on period (see TW-027)."
    depends_on: TW-011

  - finding_id: TW-042
    surface_id: pattern-no-X-found-drift
    lens: technical-writer
    severity: P2
    issue: "Same 'no X found' phrasing appears with subtle variations across admin / web / vendor / driver: 'No chefs found', 'No orders found', 'No deliveries found', 'No payouts yet', 'No staff members yet', 'No staff members found', 'No partners found', 'No partners found matching your criteria', 'No staff members. Send an invitation...'"
    evidence_excerpt: "ap-chefs-empty 'No chefs found' | web-err-browse-empty 'No chefs found / Try adjusting your filters or search query' | ap-orders-empty 'No orders found' | ap-staff-empty-members 'No staff members found' | dp-empty-staff 'No staff members yet. Send an invitation to get started.' | dp-empty-partners 'No partners found matching your criteria.'"
    recommendation: "Adopt a binary: (a) Filtered/searched result is empty → 'No {things} match your filters.' + 'Clear filters' CTA. (b) Truly empty list → 'No {things} yet.' + primary action. Codify across admin/vendor/driver. The strongest example is dp-empty-staff which is formula-compliant; clone its structure."
    depends_on: TW-022

  - finding_id: TW-043
    surface_id: pattern-not-found-bare
    lens: technical-writer
    severity: P2
    issue: "Detail pages for missing resources show bare 'X not found' with no recovery action visible in copy. ~42 backend variants of 'X not found' (see api-error-validation-not-found inventory note) and frontend equivalents."
    evidence_excerpt: "web-ux-chefdetail-not-found 'Chef not found' | web-err-orderdetail-not-found 'Order not found / View All Orders' | mc-order-detail-not-found 'Order not found / Go Back' | ap-providerdetail-notfound 'Provider not found' | ap-userdetail-notfound 'User not found' | ap-staffdetail-notfound 'Staff member not found' | dp-empty-partner-notfound 'Partner not found.' | md-emp-004 'Delivery not found.' | md-emp-008 'Partner not found.'"
    recommendation: "Pair every detail-not-found with an explicit recovery CTA in copy: 'Order not found. It may have been removed.' + 'Back to orders' button. Add the recovery half to every variant. Standardize the back-link wording: 'Back to {list}'."
    depends_on: null

  - finding_id: TW-044
    surface_id: pattern-status-transition-vague
    lens: technical-writer
    severity: P2
    issue: "'Invalid status transition' / 'Invalid status' / 'Delivery is in a terminal state' are dev-flavored states reaching user copy."
    evidence_excerpt: "Invalid status transition / Invalid status / Delivery is in a terminal state"
    recommendation: "Translate to action-relevant language: 'This delivery is already {state}.' For drivers: 'Already marked delivered.' Never expose 'terminal state' / 'status transition'."
    depends_on: null

  - finding_id: TW-045
    surface_id: pattern-vendor-customer-noun
    lens: technical-writer
    severity: P2
    issue: "Style guide §3: vendor-portal and admin should use 'chef' (their self-identity) not 'vendor' / 'cook' / 'seller'; customer surfaces should use 'Home chef'; driver app should use 'Driver' / 'Partner' only customer-facing. Scan finds 'vendor' appearing in chef-facing strings."
    evidence_excerpt: "vp-auth-login-access-denied 'This portal is only for vendor accounts. Please use the Fe3dr customer app.'"
    recommendation: "Rewrite: 'This portal is only for chef accounts. Use the customer app to order.' Replace 'Fe3dr' with the correct product name if it's a placeholder. (Brand voice owns the Fe3dr question; defer.)"
    depends_on: null

  - finding_id: TW-046
    surface_id: pattern-go-back-button-form
    lens: technical-writer
    severity: P2
    issue: "Recovery buttons on not-found / error pages use Title Case and inconsistent verbs."
    evidence_excerpt: "web-ux-chefdetail-not-found 'Browse Chefs' (Title Case) | web-err-orderdetail-not-found 'View All Orders' (Title Case, 3 words) | mc-order-detail-not-found 'Go Back' (Title Case)"
    recommendation: "Sentence case, verb-first, ≤3 words: 'Browse chefs' / 'View orders' / 'Back to orders'. Avoid the generic 'Go back' — name the destination."
    depends_on: TW-024

  - finding_id: TW-047
    surface_id: pattern-offline-banner-drift
    lens: technical-writer
    severity: P2
    issue: "Three offline banners across the codebase, three different wordings."
    evidence_excerpt: "web-err-layout-offline 'You're offline. Some features may be unavailable.' | vp-ux-layout-offline 'You're offline. Orders will sync when connected.' | dp-err-offline-banner 'You're offline. Updates will sync when connected.' | ap-layout-offline-banner 'You're offline. Data may not be up to date.'"
    recommendation: "Each is appropriate to its surface — keep specificity. But standardize the format: 'You're offline. {what happens now}.' Across surfaces, ensure the second clause names the consequence (sync, freshness, feature-loss) per audience. Add to style guide §4 as the 'connectivity banner' formula."
    depends_on: null

  - finding_id: TW-048
    surface_id: pattern-confirm-this-action-cannot
    lens: technical-writer
    severity: P2
    issue: "Style guide §4 explicitly cites 'Are you sure? This action cannot be undone.' as ❌ — vague, doesn't explain WHAT. Admin-portal uses this exact pattern in three places."
    evidence_excerpt: "ap-providerdetail-dialog-bodies 'Are you sure you want to delete this delivery provider? This action cannot be undone.' (and two siblings) | ap-providers-dialog-delete-body (same) | mv-menuitem-delete-alert 'Are you sure you want to delete this menu item?'"
    recommendation: "Apply the §4 modal subtitle formula: 'explain consequence in one sentence.' Examples: 'Deleting this provider stops all assignments and removes history.' / 'This menu item will be hidden from your storefront immediately.' Drop 'Are you sure'; the dialog title and the destructive button already imply the question."
    depends_on: null

  - finding_id: TW-049
    surface_id: pattern-staff-confirm-soften
    lens: technical-writer
    severity: P2
    issue: "Staff deactivate/reactivate confirmations use 'Are you sure you want to...' opener (banned by formula) but at least explain the consequence."
    evidence_excerpt: "ap-staffdetail-confirm-deactivate 'Are you sure you want to deactivate this staff member? They will lose access to all portals.' | ap-staffdetail-confirm-reactivate 'Are you sure you want to reactivate this staff member? They will regain access based on their role.'"
    recommendation: "Drop the 'Are you sure' opener: 'Deactivating this staff member removes access to all portals.' / 'Reactivating restores access based on their assigned role.' One sentence each; the destructive button is the confirm."
    depends_on: TW-048

  - finding_id: TW-050
    surface_id: pattern-logout-confirm
    lens: technical-writer
    severity: P2
    issue: "Mobile-vendor logout confirm uses 'Are you sure you want to log out?' which (a) violates banned 'log out' (style guide §3: 'sign out'), (b) uses 'Are you sure' filler, (c) doesn't explain consequence."
    evidence_excerpt: "mv-more-logout-confirm-body 'Are you sure you want to log out?' (Title: 'Logout')"
    recommendation: "Title: 'Sign out'. Body: 'You'll need to sign in again to access orders.' Confirm button: 'Sign out'. Tighter, on-brand, no banned terms."
    depends_on: null

  - finding_id: TW-051
    surface_id: pattern-permissions-required
    lens: technical-writer
    severity: P2
    issue: "Locked-feature explanations on mobile-delivery use a passive, slightly defensive tone: 'Fleet management is available for fleet managers only. Contact your administrator to request access.'"
    evidence_excerpt: "md-emp-007 'Fleet management is available for fleet managers only. Contact your administrator to request access.' | md-emp-011 'Staff management requires manager permissions. Contact your administrator to request access.'"
    recommendation: "Active voice and shorter: 'Only fleet managers can open this. Ask your admin for access.' (12 → 10 words) Use the same 'Ask your admin' pattern across both. Avoids 'request access' which sounds like a button label that doesn't exist."
    depends_on: null

  - finding_id: TW-052
    surface_id: pattern-rejected-banner-soften
    lens: technical-writer
    severity: P2
    issue: "Vendor-portal onboarding rejection banner reads 'Application Rejected' / 'Your previous application was not approved.' 'Rejected' is harsh; the body softens but the title doesn't."
    evidence_excerpt: "vp-onb-banner-rejected-title 'Application Rejected' / vp-onb-banner-rejected-body 'Your previous application was not approved. Please review the feedback below and re-submit.'"
    recommendation: "Title: 'Application not approved' (sentence case, softer). Body: 'Review the feedback below and resubmit.' Drop 'Please' (TW-025), use one word 'resubmit'."
    depends_on: TW-030

  - finding_id: TW-053
    surface_id: pattern-more-info-needed-banner
    lens: technical-writer
    severity: P2
    issue: "Companion banner: 'More Information Needed' (Title Case) + 'The admin team needs additional information before approving your application.' Wordy."
    evidence_excerpt: "vp-onb-banner-info-requested-title 'More Information Needed' / vp-onb-banner-info-requested-body 'The admin team needs additional information before approving your application.'"
    recommendation: "Title: 'More info needed' (sentence case, shorter). Body: 'We need a few more details before approving your application.' 'We' is preferred over 'The admin team' (style guide §5 — though applied here to operational copy)."
    depends_on: TW-030

  - finding_id: TW-054
    surface_id: pattern-failed-to-load-vague-mobile-delivery
    lens: technical-writer
    severity: P2
    issue: "Mobile-delivery has nine 'Failed to load X' strings (md-err-001..009) that all violate driver tone matrix (≤4 words target where possible) AND lack remediation. Driver surfaces are the most glance-critical."
    evidence_excerpt: "md-err-001 'Failed to load dashboard' | md-err-002 'Failed to load delivery detail' | md-err-003 'Failed to load earnings' | md-err-004 'Failed to load delivery history' | md-err-005 'Failed to load delivery detail' | md-err-006 'Failed to load profile' | md-err-007 'Failed to load fleet data' | md-err-008 'Failed to load partner detail' | md-err-009 'Failed to load staff list'"
    recommendation: "Driver-tone fix: 'Couldn't load. Tap to retry.' (4 words) as a shared component. Don't spell out what failed — the screen context is the answer. Pair with a visible retry. Apply TW-019 globally."
    depends_on: TW-019

  - finding_id: TW-055
    surface_id: pattern-bicycle-carrier
    lens: technical-writer
    severity: P2
    issue: "'Please indicate if your bicycle can carry a delivery box' (9 words, 'Please' filler, awkward 'indicate' verb). Driver tone is glanceable imperative."
    evidence_excerpt: "dp-err-step2-bike-carrier 'Please indicate if your bicycle can carry a delivery box'"
    recommendation: "Reword as a yes/no field label or telegraphic: 'Tell us if your bicycle can carry a delivery box.' or as the form question itself ('Can your bicycle carry a delivery box?') — and let the radio/checkbox communicate the action."
    depends_on: TW-025

  - finding_id: TW-056
    surface_id: pattern-stripe-onboarding-jargon
    lens: technical-writer
    severity: P2
    issue: "Chef-facing Stripe errors mix vendor jargon and dev hints. Several variants describe the same situation."
    evidence_excerpt: "Chef has not completed Stripe onboarding / Chef's Stripe account is not ready to accept charges — onboarding may be incomplete / Complete Stripe onboarding before switching to Stripe / Complete Razorpay payout setup before switching to Razorpay"
    recommendation: "User-facing chef voice: 'Finish setting up your Stripe payout account before switching.' / 'Your Stripe account isn't ready to accept payments yet.' One template per scenario; never 'Chef has not completed' (third person about themselves)."
    depends_on: null

  - finding_id: TW-057
    surface_id: pattern-search-min-chars
    lens: technical-writer
    severity: P2
    issue: "'Search query must be at least 2 characters' is fine voice but uses developer noun 'query'."
    evidence_excerpt: "api-error-search-min 'Search query must be at least 2 characters'"
    recommendation: "Replace 'Search query' with 'Search': 'Search must be at least 2 characters.'"
    depends_on: null

  - finding_id: TW-058
    surface_id: pattern-favorites-limit-numeric
    lens: technical-writer
    severity: P2
    issue: "'You can save up to 7 favorite chefs. Remove one first.' uses an arbitrary numeric limit in copy. Functionally fine; tonal nitpick is 'Remove one first' implies a sequence the user doesn't think in. Also reuses across three customer pages."
    evidence_excerpt: "web-err-favorite-max-limit 'You can save up to 7 favorite chefs. Remove one first.'"
    recommendation: "Tighten: 'You've saved the maximum 7 chefs. Remove one to save a new favorite.' Make the second sentence the action verb the user is trying to do (save, not remove)."
    depends_on: null

  - finding_id: TW-059
    surface_id: pattern-coming-soon-leak
    lens: technical-writer
    severity: P2
    issue: "'Chart coming soon' on admin analytics — placeholder copy that exposes unfinished feature work to the user. Style guide §1 Rule 5 'Restraint over urgency' implies we don't tease."
    evidence_excerpt: "ap-analytics-coming-soon 'Chart coming soon'"
    recommendation: "Either ship the chart or hide the section. As a stopgap: 'Analytics for this metric will be available in a future release.' Better: feature-flag and remove from UI when disabled."
    depends_on: null

  - finding_id: TW-060
    surface_id: pattern-locked-toast-onboarding
    lens: technical-writer
    severity: P2
    issue: "Two onboarding-locked toasts in vp-onb use slightly different framings for the same locked state."
    evidence_excerpt: "vp-onb-locked-conflict-toast 'Your kitchen is already under review or live. Returning to the dashboard.' | vp-onb-locked-info-toast 'Your kitchen is already submitted. Manage it from the dashboard.'"
    recommendation: "Pick one verb. 'Your kitchen is already submitted. Manage it from the dashboard.' covers both. If the distinction matters operationally, separate codes — but show the same string."
    depends_on: null

  - finding_id: TW-061
    surface_id: pattern-empty-no-description
    lens: technical-writer
    severity: P2
    issue: "Various 'No description provided' / 'No documents uploaded' / 'No specialties added yet' / 'No days set' / 'None selected' empty inline strings vary in style: some end with period, some don't; some use 'yet', some don't."
    evidence_excerpt: "vp-ux-menu-view-empty-desc 'No description provided' | vp-onb-kitchen-specialties-empty 'No specialties added yet' | vp-onb-review-empty-cuisines 'None selected' | vp-onb-review-empty-days 'No days set' | vp-onb-review-empty-docs 'No documents uploaded' | ap-secsettings-apikey-empty 'No keys yet.' | ap-platsettings-zones-empty 'No zones yet — delivery is available everywhere until a zone is created.'"
    recommendation: "Standardize as 'No {things} yet' (no period when used as a label) with the dash-explanation variant ('No zones yet — delivery is available everywhere...') only when context is needed. Drop the period from labels; keep the period if used as a full sentence."
    depends_on: TW-027

  - finding_id: TW-062
    surface_id: pattern-saved-addresses-yet
    lens: technical-writer
    severity: P2
    issue: "Customer 'No addresses' empty state is duplicated across web checkout and mobile checkout with slightly different wording."
    evidence_excerpt: "web-err-checkout-no-addresses 'You don't have any saved addresses yet. Add one to continue.' | mc-checkout-address-errors (Zod errors covering similar territory)"
    recommendation: "Single template: 'No saved addresses. Add a delivery address to continue.' Identical across web + mobile. See TW-018."
    depends_on: TW-018

  - finding_id: TW-063
    surface_id: pattern-pending-orders-drift
    lens: technical-writer
    severity: P2
    issue: "'No pending orders' appears with three different bodies across vendor-portal and mobile-vendor."
    evidence_excerpt: "vp-ux-dashboard-pending-empty 'All caught up! / No pending orders right now' | vp-ux-orders-live-empty 'No active orders right now. New orders will appear here automatically.' | mv-orders-empty-title 'No pending orders' / mv-orders-empty-body 'New orders will appear here automatically'"
    recommendation: "One template: 'No pending orders. New orders appear here automatically.' Drop 'right now' (filler), drop 'All caught up!' (exclamation), drop the variant in mobile-vendor that omits the second sentence."
    depends_on: TW-029

  - finding_id: TW-064
    surface_id: pattern-rate-limit-friendly
    lens: technical-writer
    severity: P2
    issue: "Chat rate-limit message 'Please wait a moment before sending another message' is friendly but contains 'Please' filler and is wordy for a transient banner."
    evidence_excerpt: "api-error-chat-availability 'Please wait a moment before sending another message'"
    recommendation: "Tighter: 'Slow down — wait a moment before sending another message.' Or simpler: 'You're sending messages too fast. Wait a moment.'"
    depends_on: TW-025

  - finding_id: TW-065
    surface_id: pattern-chat-availability-wordy
    lens: technical-writer
    severity: P2
    issue: "Chat-closed errors carry redundant phrasing across three near-identical variants."
    evidence_excerpt: "api-error-chat-availability 'Chat is not available for completed/cancelled orders' / 'This chat room is closed' / 'This chat room is closed because the order is no longer active'"
    recommendation: "Standardize: 'Chat closed — this order is complete.' / 'Chat closed — this order was cancelled.' Single template with state-specific reason. Drop 'chat room' (use 'chat'). Drop 'no longer active' (vague)."
    depends_on: null

  - finding_id: TW-066
    surface_id: pattern-otto-retry
    lens: technical-writer
    severity: P2
    issue: "'Try again' as a standalone button in OttoChat retry. Per style guide formula, fine — but appears alongside an error toast in same flow that already says 'Something went wrong'. Two redundant CTAs."
    evidence_excerpt: "web-err-otto-retry 'Try again'"
    recommendation: "Keep 'Try again' as the button; ensure paired error message gives the 'what happened' half. E.g., 'Couldn't connect to Otto. Try again.' + button reading 'Try again' (the body provides context, button stays short)."
    depends_on: null

  - finding_id: TW-067
    surface_id: pattern-validation-error-counts
    lens: technical-writer
    severity: P2
    issue: "Mobile-customer catering form (mc-catering-errors) bundles 9 separate Zod messages with 40 total words. Some use 'is required' verb, some use specific verbs ('must be in the future'). Mixed patterns make the form feel chaotic."
    evidence_excerpt: "Event type is required / Date must be YYYY-MM-DD / Event date must be in the future / Guest count must be a number / At least 1 guest required / Budget must be a number / Budget must be positive / City is required / State is required"
    recommendation: "Two formula families: (a) Required: '{Field} is required.' (b) Validation: '{Field} must be {rule}.' Pick a date verb: 'Pick a future date.' (active, friendlier). 'Budget must be positive' → 'Budget must be more than ₹0.'"
    depends_on: null

  - finding_id: TW-068
    surface_id: pattern-zod-india-format
    lens: technical-writer
    severity: P2
    issue: "Several validation strings hard-code India-specific assumptions in the message: 'Enter a valid 10-digit Indian mobile number', 'Enter a valid 6-digit pincode', 'upiId is required'. Fine for v1 but the messages are inconsistent: some say 'Indian', some don't."
    evidence_excerpt: "mc-onb-step1-errors 'Enter a valid 10-digit Indian mobile number' | mc-onb-step2-errors 'Enter a valid 6-digit pincode' | mv-onb-personal-phone-err 'Enter a valid 10-digit Indian mobile number'"
    recommendation: "Drop 'Indian' (the form is India-only at v1). 'Enter a valid 10-digit mobile number.' / 'Enter a valid 6-digit PIN code.' (style guide says 'PIN code' not 'pincode'). Centralize India-format validation messages in shared zod schemas."
    depends_on: null

  - finding_id: TW-069
    surface_id: pattern-mobile-customer-profile-error-alert
    lens: technical-writer
    severity: P2
    issue: "Mobile-customer profile errors use both a title ('Error') and a body — title-less alerts would be friendlier."
    evidence_excerpt: "mc-profile-save-error 'Error / Could not update profile. Please try again. / Could not save preferences.'"
    recommendation: "Drop the 'Error' title; the body is enough. If a title is structurally required, use 'Couldn't save profile' (specific). Drop 'Please' (TW-025). 'Couldn't update profile. Try again.' (5 words)."
    depends_on: TW-031

  - finding_id: TW-070
    surface_id: pattern-mobile-vendor-permission-alerts
    lens: technical-writer
    severity: P2
    issue: "Permission alerts on mobile-vendor use Title Case headings + verbose bodies: 'Permission Required' + 'Camera permission is needed to take photos.'"
    evidence_excerpt: "mv-onb-docs-perm-title 'Permission Required' / mv-onb-docs-camera-perm 'Camera permission is needed to take photos.' / mv-onb-docs-gallery-perm 'Gallery permission is needed to select photos.'"
    recommendation: "Sentence case + tight: 'Camera access needed' / 'We need camera access to take photos.' Drop 'is needed' (passive). Audit fix should consolidate camera+gallery into one helper since both files reuse them."
    depends_on: TW-030

  - finding_id: TW-071
    surface_id: pattern-tos-error
    lens: technical-writer
    severity: P2
    issue: "'You must accept the terms and conditions' on driver onboarding has no link to the actual T&C. The user is being told to accept something but the doc isn't surfaced inline."
    evidence_excerpt: "api-error-tos 'You must accept the terms and conditions'"
    recommendation: "TW-side fix: 'Accept the Terms of Service to continue.' (drops 'You must', shorter). Legal lens owns the link-to-doc requirement; flag for them. Also align casing — 'Terms of Service' (proper noun) vs 'terms and conditions' (lowercase) drift."
    depends_on: null

  - finding_id: TW-072
    surface_id: pattern-payments-policy-missing
    lens: technical-writer
    severity: P2
    issue: "Order-already-paid / payment-not-captured / refund-amount errors don't communicate the refund/charge policy. From a TW lens this is a missing-recovery-half issue."
    evidence_excerpt: "api-error-order-already-paid 'Order already paid' | api-error-payment-refund 'Payment not captured' | 'Refund amount cannot exceed order total'"
    recommendation: "Pair each with policy line: 'Order already paid. View receipt or contact support for a refund.' / 'Payment not captured yet. We'll release the hold within 7 days.' Legal lens owns exact wording on refund SLA; TW recommends paired action exists."
    depends_on: TW-004

  - finding_id: TW-073
    surface_id: pattern-error-boundaries-drift
    lens: technical-writer
    severity: P2
    issue: "Four different error-boundary strings across apps. Same global-failure concept, four wordings."
    evidence_excerpt: "web-err-error-boundary 'Unexpected error / Something broke on our end. / We've been notified...' | vp-err-boundary-fallback 'Something went wrong / An unexpected error occurred while loading this page. / Try Again' | dp-err-boundary-title 'Something went wrong' / dp-err-boundary-body 'An unexpected error occurred while loading this page.' | ap-errorboundary-title 'Something went wrong' / ap-errorboundary-body 'An unexpected error occurred while loading this page.'"
    recommendation: "Pick one template across all four apps: Title 'Something broke' / Body 'Refresh to try again, or come back in a moment.' / Single CTA 'Refresh page'. Drop 'we've been notified' (TW-010). Sentence case throughout."
    depends_on: TW-010

  - finding_id: TW-074
    surface_id: pattern-driver-history-empty-drift
    lens: technical-writer
    severity: P2
    issue: "Driver-portal and mobile-delivery have the same 'no deliveries yet' empty state with slightly different wording."
    evidence_excerpt: "dp-empty-history-title 'No Deliveries Yet' / dp-empty-history-body 'Your completed deliveries will appear here.' | md-emp-003 'No deliveries yet. Your completed deliveries will appear here.'"
    recommendation: "Single template: 'No deliveries yet. Completed deliveries appear here.' Apply across both. Sentence case (TW-030)."
    depends_on: TW-030

  - finding_id: TW-075
    surface_id: pattern-staff-empty-drift
    lens: technical-writer
    severity: P2
    issue: "Admin-portal staff empty state has a clean message that includes the action; delivery-portal staff empty does too; mobile-delivery staff empty does not."
    evidence_excerpt: "ap-staff-empty-members 'No staff members found' (no action) | dp-empty-staff 'No staff members yet. Send an invitation to get started.' (has action) | md-emp-009 'No staff members yet.' (no action)"
    recommendation: "Apply dp-empty-staff's structure everywhere: 'No staff yet. Send an invitation to get started.' Drop 'members' (redundant)."
    depends_on: TW-022

  - finding_id: TW-076
    surface_id: pattern-fleet-empty-drift
    lens: technical-writer
    severity: P2
    issue: "Two 'no partners' empty states, two wordings; both lack a recovery action."
    evidence_excerpt: "dp-empty-partners 'No partners found matching your criteria.' | md-emp-005 'No partners found.'"
    recommendation: "If filtered: 'No partners match your filters.' + 'Clear filters'. If empty: 'No partners yet.' + 'Add partner' CTA. Decide which surface is which and apply."
    depends_on: TW-042

  - finding_id: TW-077
    surface_id: pattern-empty-list-period
    lens: technical-writer
    severity: P2
    issue: "Inconsistency: admin-portal empty states mostly omit periods on single-line labels ('No chefs found'); delivery-portal and mobile-delivery mix them ('No partners found.', 'Delivery not found.')."
    evidence_excerpt: "ap-chefs-empty 'No chefs found' (no period) vs md-emp-005 'No partners found.' (period) vs dp-empty-partner-notfound 'Partner not found.' (period)"
    recommendation: "Apply TW-027: single-line labels → no period. Apply consistently across all four apps."
    depends_on: TW-027

  - finding_id: TW-078
    surface_id: pattern-validation-error-suspend
    lens: technical-writer
    severity: P2
    issue: "Admin user-action failure toasts are bare 'Failed to suspend user' / 'Failed to activate user' — no recovery."
    evidence_excerpt: "ap-users-toast-fail-suspend 'Failed to suspend user' | ap-users-toast-fail-activate 'Failed to activate user'"
    recommendation: "'Couldn't suspend user. Try again.' / 'Couldn't activate user. Try again.' Add the recovery half."
    depends_on: TW-019

  - finding_id: TW-079
    surface_id: pattern-validation-pin-vs-pincode
    lens: technical-writer
    severity: P2
    issue: "Inconsistent terminology: 'pincode', 'PIN code', 'PIN'. Customer-facing should align with Indian official term 'PIN code'."
    evidence_excerpt: "vp-onb-personal-validation '... PIN code is required' | mc-onb-step2-errors 'Enter a valid 6-digit pincode' | mc-checkout-address-errors 'Pincode must be 6 digits'"
    recommendation: "Adopt 'PIN code' everywhere. Update zod schemas + label strings in mobile-customer."
    depends_on: TW-068

  - finding_id: TW-080
    surface_id: pattern-app-name-fe3dr
    lens: technical-writer
    severity: P2
    issue: "Vendor-portal access-denied error mentions 'Fe3dr customer app' — likely a placeholder app name. TW flags the inconsistency; brand voice owns the canonical name."
    evidence_excerpt: "vp-auth-login-access-denied 'This portal is only for vendor accounts. Please use the Fe3dr customer app.'"
    recommendation: "Confirm canonical app name with brand owner; update everywhere. TW-side fix: 'This portal is only for chef accounts. Use the customer app to order.' (drops the brand placeholder, drops 'vendor' per TW-045)."
    depends_on: TW-045

  - finding_id: TW-081
    surface_id: pattern-validation-percentage-vs-percent
    lens: technical-writer
    severity: P2
    issue: "'Percentage discount must be between 0 and 100' uses 'Percentage' as noun in an awkward UI position. Could read more naturally."
    evidence_excerpt: "api-error-validation-numeric 'Percentage discount must be between 0 and 100' / 'rate must be between 0 and 100'"
    recommendation: "'Discount must be 0-100%.' / 'Rate must be 0-100%.' Use the percent symbol for compactness. Drop 'Percentage' / 'rate' as bare lower-case nouns."
    depends_on: null

  - finding_id: TW-082
    surface_id: pattern-validation-interval
    lens: technical-writer
    severity: P2
    issue: "'Invalid interval. Must be monthly, quarterly, or yearly' on subscription form — but the form should be a dropdown that prevents invalid entry, making the error rare."
    evidence_excerpt: "api-error-validation-interval 'Invalid interval. Must be monthly, quarterly, or yearly'"
    recommendation: "If reachable: 'Pick an interval: monthly, quarterly, or yearly.' If unreachable in UI: log only, return generic. Decide based on form design."
    depends_on: null

  - finding_id: TW-083
    surface_id: pattern-validation-min-2-chars
    lens: technical-writer
    severity: P2
    issue: "'Name must be at least 2 characters' min-length applied to chef-onboarding name field — bounded incorrectly given Indian names like 'Om' (2 chars valid). But more importantly, the same field shows min=3 on mobile-vendor."
    evidence_excerpt: "vp-ux-menu-form-validation 'Name must be at least 2 characters' | mv-menunew-name-err 'Name must be at least 3 characters'"
    recommendation: "Align validation bounds across vendor-portal + mobile-vendor (both refer to menu items, not human names — confirm). Use the looser bound (2 chars) unless there's a data reason. Keep messages identical: 'Name needs at least N characters.'"
    depends_on: TW-032

  - finding_id: TW-084
    surface_id: pattern-mobile-vendor-photo-upload-fail
    lens: technical-writer
    severity: P2
    issue: "Three near-identical photo-upload-fail toasts in mv-profile/menuedit, each with slightly different wording."
    evidence_excerpt: "mv-profile-photo-upload-fail 'Failed to upload profile photo.' | mv-profile-kitchenphoto-upload-fail 'Failed to upload kitchen photo.' | mv-menuedit-photo-upload-fail 'Failed to upload photo. Please try again.'"
    recommendation: "Single template: 'Couldn't upload {photo type}. Try again.' E.g., 'Couldn't upload your kitchen photo. Try again.' Drop 'Please' (TW-025), drop 'Failed to' (TW-026)."
    depends_on: TW-026

  - finding_id: TW-085
    surface_id: pattern-mobile-customer-favorites-error
    lens: technical-writer
    severity: P2
    issue: "'Could not remove from favorites. Please try again.' is OK but missing a soft 'we': 'Could not' is impersonal."
    evidence_excerpt: "mc-favorites-error 'Could not remove from favorites. Please try again.'"
    recommendation: "Use 'Couldn't' (informal) or 'We couldn't' (personal). 'Couldn't remove from favorites. Try again.' Drop 'Please' (TW-025)."
    depends_on: TW-025

  - finding_id: TW-086
    surface_id: pattern-checkout-payment-cancelled
    lens: technical-writer
    severity: P2
    issue: "'Payment cancelled' (2 words) appears as a toast — but doesn't follow toast formula (past tense, period, ≤6 words). Past tense ✓, ≤6 words ✓; missing period."
    evidence_excerpt: "web-err-checkout-cancel 'Payment cancelled'"
    recommendation: "Add period: 'Payment cancelled.' Toast formula compliance. Consider expanding for clarity: 'Payment cancelled. Your cart is saved.' (5 words; still formula-compliant + reassures the customer they didn't lose work)."
    depends_on: null

  - finding_id: TW-087
    surface_id: pattern-chef-not-accepting
    lens: technical-writer
    severity: P2
    issue: "'Chef is not accepting orders' has no alternative action for the customer mid-funnel."
    evidence_excerpt: "api-error-chef-not-accepting 'Chef is not accepting orders'"
    recommendation: "Pair with action: 'This chef isn't accepting orders right now. Browse other chefs near you.' Two sentences, formula-compliant, redirects rather than dead-ends."
    depends_on: null

  - finding_id: TW-088
    surface_id: pattern-quote-already-submitted
    lens: technical-writer
    severity: P2
    issue: "'This request is no longer accepting quotes' / 'You have already submitted a quote for this request' / 'Quote deadline has passed' — three near-related error states, all without recovery action."
    evidence_excerpt: "api-error-quotes-catering 'This request is no longer accepting quotes' / 'You have already submitted a quote for this request' / 'Quote deadline has passed'"
    recommendation: "Add recovery: 'This request is no longer accepting quotes. Browse other open requests.' / 'You've already quoted this request. View your quote.' / 'The quote deadline has passed.' For chef-side: each links back to a list of open requests."
    depends_on: null

  - finding_id: TW-089
    surface_id: pattern-promo-not-found
    lens: technical-writer
    severity: P2
    issue: "Promo errors include 'Promo code already exists' / 'Promo code not found' / 'Promotion not found or already activated' — three subtly different concepts."
    evidence_excerpt: "api-error-promo 'Promo code already exists' / 'Promo code not found' / 'Promotion not found or already activated'"
    recommendation: "Distinguish copy clearly. (1) At creation: 'This promo code already exists. Pick a different code.' (2) At redemption: 'Promo code not found.' (3) At activation: 'Promotion not found or already activated.' Three formulas, clear distinction; pair with action where possible."
    depends_on: null

  - finding_id: TW-090
    surface_id: pattern-review-only-delivered
    lens: technical-writer
    severity: P2
    issue: "'Can only review delivered orders' / 'This order has already been reviewed' — fine information but missing recovery."
    evidence_excerpt: "api-error-order-review-rules 'Can only review delivered orders' / 'This order has already been reviewed'"
    recommendation: "Pair with action: 'Reviews are available after delivery.' (active voice) / 'You've already reviewed this order. View your review.'"
    depends_on: null

  - finding_id: TW-091
    surface_id: pattern-invitation-token-drift
    lens: technical-writer
    severity: P2
    issue: "Four invitation-token error variants very close in meaning."
    evidence_excerpt: "api-error-invitation 'Invitation has expired' / 'Invalid or expired invitation' / 'Invalid invitation token' / 'A pending invitation already exists for this email'"
    recommendation: "Standardize: 'This invitation has expired. Ask the admin to resend.' / 'Invitation not valid. Check the link and try again.' / 'A pending invitation already exists for this email.' Keep 'token' out of user copy."
    depends_on: TW-037

  - finding_id: TW-092
    surface_id: pattern-staff-permissions-tone
    lens: technical-writer
    severity: P2
    issue: "Staff permission errors are accurate but blunt: 'You don't have permission to invite staff' / 'Cannot deactivate yourself' / 'Not a staff member'."
    evidence_excerpt: "api-error-staff-permissions 'You don't have permission to invite staff' / 'Only super admins can change staff roles' / 'Only super admins can invite other super admins' / 'Cannot change role of a default super admin' / 'Cannot deactivate a default super admin' / 'Cannot deactivate yourself' / 'Not a staff member' / 'This person is already a staff member'"
    recommendation: "Soften without losing precision (admin tone, but still respectful): 'Only super admins can invite staff. Ask one to send the invite.' / 'You can't deactivate your own account.' / 'This person isn't a staff member.' Drop the bare 'Not a staff member' (sounds like an error code)."
    depends_on: null

  # =====================================================================
  # P3 — Polish / small consistency wins
  # =====================================================================

  - finding_id: TW-093
    surface_id: pattern-currency-formatting
    lens: technical-writer
    severity: P3
    issue: "Error strings mix currency formats: '₹120' (no space, correct), but the price-range error uses '₹1 and ₹10,000' — fine — and other errors mention bare numbers ('between 0 and 100'). No violations of style guide §6 found in error copy specifically, but worth a polish pass."
    evidence_excerpt: "mv-menunew-price-range 'Price must be between ₹1 and ₹10,000' (correct format) | vp-ux-menu-form-validation 'Minimum price is ₹20'"
    recommendation: "Keep as-is; flag this as PASSED for the audit fix. Polish only: align min-price across vendor-portal (₹20) and mobile-vendor (₹1) — those are different rules, confirm intent."
    depends_on: null

  - finding_id: TW-094
    surface_id: pattern-retry-cta-text
    lens: technical-writer
    severity: P3
    issue: "Retry CTAs across surfaces use a mix of 'Retry', 'Try again', 'Try Again', 'Refresh page'."
    evidence_excerpt: "mv-dash-retry-cta 'Retry' | web-err-otto-retry 'Try again' | vp-ux-menu-error-state 'Try Again' | vp-err-boundary-fallback 'Try Again' | web-err-error-boundary 'Try again' + 'Refresh page'"
    recommendation: "Pick one canonical retry: 'Try again' (sentence case). Use 'Retry' only when space is constrained (driver glance contexts). Use 'Refresh page' only for full-page error boundaries."
    depends_on: TW-024

  - finding_id: TW-095
    surface_id: pattern-payouts-history-empty
    lens: technical-writer
    severity: P3
    issue: "Vendor-portal payouts empty state body is wordy: 'Your payout history will appear here once you receive your first payout.' (12 words)."
    evidence_excerpt: "vp-ux-payouts-empty 'No payouts yet / Your payout history will appear here once you receive your first payout.'"
    recommendation: "Tighten: 'No payouts yet. Your first payout will appear here.' (9 words)."
    depends_on: null

  - finding_id: TW-096
    surface_id: pattern-notifications-empty
    lens: technical-writer
    severity: P3
    issue: "Vendor-portal notifications empty is informative but wordy: 'No admin requests yet / Admin requests for document uploads, profile updates, and review feedback will appear here.' (17 words)."
    evidence_excerpt: "vp-ux-notifs-empty 'No admin requests yet / Admin requests for document uploads, profile updates, and review feedback will appear here.'"
    recommendation: "Tighten and lead with the user's stake: 'No admin requests yet. Document and profile requests will appear here.' (10 words)."
    depends_on: null

  - finding_id: TW-097
    surface_id: pattern-orders-empty-customer
    lens: technical-writer
    severity: P3
    issue: "Customer orders-empty body is OK but has redundant phrasing in the filtered case."
    evidence_excerpt: "web-err-orders-empty 'No orders found / You haven't placed any orders yet. / No {filter} orders found.'"
    recommendation: "Two states: (a) Truly empty: 'No orders yet. Browse chefs near you.' (b) Filtered: 'No {filter} orders.' + 'Clear filters' action. Drop the 'No orders found / You haven't placed any orders yet.' redundancy."
    depends_on: TW-042

  - finding_id: TW-098
    surface_id: pattern-favorites-empty-customer
    lens: technical-writer
    severity: P3
    issue: "Web 'No favorites yet' is barebones (3 words); mobile is chatty ('No saved chefs yet / Tap the heart on any chef to save them!'). Cross-surface inconsistency."
    evidence_excerpt: "web-err-favorites-empty 'No favorites yet' | mc-favorites-empty 'No saved chefs yet / Tap the heart on any chef to save them!'"
    recommendation: "Single template: 'No favorites yet. Tap the heart on a chef to save.' Apply across web + mobile. Drop exclamation (TW-029)."
    depends_on: TW-028

  - finding_id: TW-099
    surface_id: pattern-empty-category
    lens: technical-writer
    severity: P3
    issue: "Two surfaces show 'No items in this category' as an empty state — fine, but neither offers an action."
    evidence_excerpt: "web-err-chefdetail-empty-category 'No items in this category' | mc-chef-empty-category 'No items in this category'"
    recommendation: "OK to leave actionless if the user can browse other categories visually (tabs/filters). If category list is hidden, add: 'No items in this category. Try another.'"
    depends_on: null

  - finding_id: TW-100
    surface_id: pattern-cart-minimum
    lens: technical-writer
    severity: P3
    issue: "'Minimum order not met / Add {amount} more to proceed' is fine but 'Add {amount} more to proceed' could be friendlier."
    evidence_excerpt: "web-err-cart-minimum 'Minimum order not met / Add {amount} more to proceed'"
    recommendation: "Tighter: 'Add ₹{amount} more to place your order.' (replaces 'proceed' jargon, names the action 'place order')."
    depends_on: null

  - finding_id: TW-101
    surface_id: pattern-no-posts-yet
    lens: technical-writer
    severity: P3
    issue: "Social feed empty: web shows 'No posts yet' (3 words); mobile shows 'No posts yet / Chefs will share their latest creations here.' (8 words). Inconsistent."
    evidence_excerpt: "web-err-social-feed-empty 'No posts yet' | mc-social-empty 'No posts yet / Chefs will share their latest creations here.'"
    recommendation: "Use the mobile version on both (it gives context). Sentence-case + period: 'No posts yet. Chefs will share their latest creations here.'"
    depends_on: null

  - finding_id: TW-102
    surface_id: pattern-dashboard-recent-activity
    lens: technical-writer
    severity: P3
    issue: "Admin 'Recent activity' empty is two-line: 'No recent activity / Platform events will appear here.' Period inconsistency."
    evidence_excerpt: "ap-dashboard-recent-empty-title 'No recent activity' / ap-dashboard-recent-empty-body 'Platform events will appear here.'"
    recommendation: "Body fine; standardize period rule per TW-027."
    depends_on: TW-027

  - finding_id: TW-103
    surface_id: pattern-audit-logs-empty
    lens: technical-writer
    severity: P3
    issue: "Audit-logs empty is a filtered-empty case but reads like a truly-empty case."
    evidence_excerpt: "ap-auditlogs-empty 'No audit events match these filters.'"
    recommendation: "OK; consider 'No audit events match these filters. Clear filters to see all.' to give the user an action."
    depends_on: TW-042

  - finding_id: TW-104
    surface_id: pattern-data-exports-download-failed
    lens: technical-writer
    severity: P3
    issue: "'Download failed' (2 words) on admin exports — formula-compliant but bare."
    evidence_excerpt: "ap-exports-error 'Download failed'"
    recommendation: "Add recovery: 'Download failed. Try again.' (4 words; satisfies error formula + ≤6 word toast rule)."
    depends_on: null

  - finding_id: TW-105
    surface_id: pattern-platform-settings-zones
    lens: technical-writer
    severity: P3
    issue: "Platform-settings zones empty has a great context-aware message but uses an em-dash style that's inconsistent across the rest of the app."
    evidence_excerpt: "ap-platsettings-zones-empty 'No zones yet — delivery is available everywhere until a zone is created.'"
    recommendation: "Keep the em-dash style as a recommended pattern for explanatory empty states. Apply elsewhere where useful: 'No keys yet — generate one to authenticate API requests.'"
    depends_on: null

  - finding_id: TW-106
    surface_id: pattern-security-no-keys
    lens: technical-writer
    severity: P3
    issue: "Admin API keys empty is bare: 'No keys yet.'"
    evidence_excerpt: "ap-secsettings-apikey-empty 'No keys yet.'"
    recommendation: "Apply TW-105 pattern: 'No API keys yet — generate one to authenticate requests.'"
    depends_on: TW-105

  - finding_id: TW-107
    surface_id: pattern-dashboard-allclear
    lens: technical-writer
    severity: P3
    issue: "Admin dashboard 'No pending chef applications.' is bare; can give value-add context."
    evidence_excerpt: "ap-dashboard-allclear-body 'No pending chef applications.'"
    recommendation: "Add reassurance + a quick action: 'No pending applications. Review past decisions in the chefs list.' (or similar)."
    depends_on: null

  - finding_id: TW-108
    surface_id: pattern-staff-permissions-data
    lens: technical-writer
    severity: P3
    issue: "Admin staff permissions empty has awkward formulation: 'No specific permissions data available for this role.'"
    evidence_excerpt: "ap-staffdetail-perms-empty 'No specific permissions data available for this role.'"
    recommendation: "Reword: 'No specific permissions for this role.' or 'This role uses default permissions.'"
    depends_on: null

  - finding_id: TW-109
    surface_id: pattern-approval-history-empty
    lens: technical-writer
    severity: P3
    issue: "Approval history empty 'No history recorded yet.' is fine but the 'recorded' word is slightly bureaucratic."
    evidence_excerpt: "ap-approvaldetail-history-empty 'No history recorded yet.'"
    recommendation: "Tighter: 'No history yet.' (matches the other 'No X yet' family)."
    depends_on: TW-061

  - finding_id: TW-110
    surface_id: pattern-mobile-onb-step3-error
    lens: technical-writer
    severity: P3
    issue: "Mobile-customer onboarding setup failure alert: 'Setup failed / Something went wrong. Please try again.' — duplicates a generic message in a critical onboarding step."
    evidence_excerpt: "mc-onb-step3-error 'Setup failed / Something went wrong. Please try again.'"
    recommendation: "Drop the duplicate 'Setup failed' title; the body covers it. 'Couldn't finish setup. Try again.' (5 words)."
    depends_on: TW-021

  - finding_id: TW-111
    surface_id: pattern-document-required-alerts-drift
    lens: technical-writer
    severity: P3
    issue: "Mobile-vendor onboarding has multiple 'Required' alerts that mix title and body styles."
    evidence_excerpt: "mv-onb-docs-required-title 'Documents Required' / mv-onb-docs-required-alert 'Please upload both ID proof and FSSAI license to continue.' | mv-onb-policies-terms-required-title 'Terms Required' / mv-onb-policies-terms-required-alert 'Please accept the terms and conditions to continue.' | mv-onb-policies-policy-required-title 'Policy Required' / mv-onb-policies-policy-required-alert 'Please select a cancellation policy.'"
    recommendation: "Sentence case titles, drop 'Required' (covered by body): 'Upload documents' / 'Upload both ID proof and FSSAI license to continue.' / 'Accept terms' / 'Accept the Terms of Service to continue.' Drop 'Please' (TW-025)."
    depends_on: TW-025

  - finding_id: TW-112
    surface_id: pattern-step1-personal-validation
    lens: technical-writer
    severity: P3
    issue: "Vendor-portal onboarding step 1 validation listing is bare ('Full name is required / Phone number is required / Email is required ...'). Fine for inline, but as a toast bundle it's seven sentences."
    evidence_excerpt: "vp-onb-personal-validation 'Full name is required / Phone number is required / Email is required / Address is required / City is required / State is required / PIN code is required'"
    recommendation: "Show inline per-field; if combined toast needed: 'Some required fields are missing.' Then scroll to the first error. Don't list all seven."
    depends_on: null

  - finding_id: TW-113
    surface_id: pattern-checkout-zod
    lens: technical-writer
    severity: P3
    issue: "Mobile-customer checkout address Zod errors duplicate the same patterns from onboarding ('City is required / State is required'). Drift risk — different schemas, same field."
    evidence_excerpt: "mc-checkout-address-errors 'Address line 1 is required / City is required / State is required / Pincode must be 6 digits' | mc-onb-step2-errors 'Address must be at least 5 characters / City is required / State is required / Enter a valid 6-digit pincode'"
    recommendation: "Centralize address-field validation messages in shared schema; reuse across onboarding + checkout. Use 'PIN code' (TW-079). Pick one address-line message ('Address must be at least 5 characters' vs 'Address line 1 is required')."
    depends_on: TW-068

  - finding_id: TW-114
    surface_id: pattern-prep-time-validation
    lens: technical-writer
    severity: P3
    issue: "Vendor-portal menu form prep-time validation uses 'Prep time required / Prep time must be at least 1 minute / Prep time must be under 8 hours'. Inconsistent: 'required' uses elided 'is'; others spell out the bound. Also 'serves count' is awkward."
    evidence_excerpt: "vp-ux-menu-form-validation 'Prep time required / Prep time must be at least 1 minute / Prep time must be under 8 hours / Must serve at least 1 / Must serve under 100 / Serves count is required'"
    recommendation: "Standardize: 'Prep time is required.' / 'Prep time must be at least 1 minute.' / 'Prep time must be under 8 hours.' / 'Serves at least 1 person.' / 'Serves at most 100 people.' / 'Number of servings is required.' Treat 'serves' as the verb, not a noun."
    depends_on: TW-032

  - finding_id: TW-115
    surface_id: pattern-card-empty-mobile-customer
    lens: technical-writer
    severity: P3
    issue: "Mobile-customer cart-sheet says 'Your cart is empty' (4 words, no period); checkout says 'Your cart is empty.' (4 words, period). Same surface family, micro-drift."
    evidence_excerpt: "mc-cartsheet-empty 'Your cart is empty' vs mc-checkout-empty-cart 'Your cart is empty.'"
    recommendation: "Single string + period consistency rule. Suggest 'Your cart is empty.' (period since it's a full sentence). See TW-041."
    depends_on: TW-041

  - finding_id: TW-116
    surface_id: pattern-customer-orders-empty-emoji
    lens: technical-writer
    severity: P3
    issue: "Mobile-customer orders empty uses emoji-in-copy: '🍽️ No orders yet / Browse chefs to place your first order!'"
    evidence_excerpt: "mc-orders-empty 'No orders yet / Browse chefs to place your first order!' (with 🍽️ emoji)"
    recommendation: "Drop emoji from copy (place in visual icon slot). Drop exclamation. 'No orders yet. Browse chefs to place your first order.' (10 words)."
    depends_on: TW-028

  - finding_id: TW-117
    surface_id: pattern-customer-catering-empty-emoji
    lens: technical-writer
    severity: P3
    issue: "Mobile-customer catering empty uses emoji-in-copy: '🍽️ No requests yet / Submit a catering request to get quotes from chefs.'"
    evidence_excerpt: "mc-catering-empty 'No requests yet / Submit a catering request to get quotes from chefs.' (with 🍽️ emoji)"
    recommendation: "Drop emoji. 'No requests yet. Submit a catering request to get quotes from chefs.' (12 words)."
    depends_on: TW-028

  - finding_id: TW-118
    surface_id: pattern-mobile-vendor-photo-delete
    lens: technical-writer
    severity: P3
    issue: "Mobile-vendor photo-delete confirm uses 'Remove this photo?' with title 'Delete Photo'. Verb mismatch (Remove vs Delete) within the same dialog."
    evidence_excerpt: "mv-menuedit-delete-photo-alert 'Remove this photo?' (Title: 'Delete Photo')"
    recommendation: "Align verbs: title 'Remove photo' (sentence case) + body 'This photo will be removed from your listing.' Use 'remove' throughout."
    depends_on: TW-030

  - finding_id: TW-119
    surface_id: pattern-mobile-vendor-reply-fallback
    lens: technical-writer
    severity: P3
    issue: "Two near-identical reply-too-short errors in the same screen: 'Reply must be at least 10 characters' (Zod) and 'Reply is too short' (fallback)."
    evidence_excerpt: "mv-reviewdetail-min-err 'Reply must be at least 10 characters' / mv-reviewdetail-min-fallback 'Reply is too short'"
    recommendation: "Pick one. Style guide prefers specific bound: 'Reply must be at least 10 characters.'"
    depends_on: TW-032

  - finding_id: TW-120
    surface_id: pattern-mobile-vendor-toggle-online-status
    lens: technical-writer
    severity: P3
    issue: "Toast 'Failed to update status' reused in dashboard + active delivery — bare and missing the entity (online status? delivery status? menu status?)."
    evidence_excerpt: "dp-err-toggle-status-failed 'Failed to update status'"
    recommendation: "Be specific per surface: 'Couldn't update online status. Try again.' / 'Couldn't update delivery status. Try again.'"
    depends_on: TW-019

  - finding_id: TW-121
    surface_id: pattern-customer-chef-load-error
    lens: technical-writer
    severity: P3
    issue: "Customer mobile chef-detail load error reads 'Failed to load chef details. Please try again.' — 7 words, 'Please' filler."
    evidence_excerpt: "mc-chef-error-load 'Failed to load chef details. Please try again.'"
    recommendation: "Customer tone: 'Couldn't load chef details. Try again.' (5 words)."
    depends_on: TW-025

  - finding_id: TW-122
    surface_id: pattern-otto-no-recovery
    lens: technical-writer
    severity: P3
    issue: "Otto chat retry button has no associated error message text in inventory — only the button. If the surrounding banner uses 'Something went wrong', combine."
    evidence_excerpt: "web-err-otto-retry 'Try again'"
    recommendation: "Audit fix verifies: paired error text exists and follows formula. If not, add: 'Couldn't reach Otto. Try again.'"
    depends_on: null

  - finding_id: TW-123
    surface_id: pattern-mobile-onboarding-empty-specialties
    lens: technical-writer
    severity: P3
    issue: "Vendor-portal kitchen step shows 'No specialties added yet' — fine, but should pair with the 'Add specialty' action visible nearby. Confirm UI."
    evidence_excerpt: "vp-onb-kitchen-specialties-empty 'No specialties added yet'"
    recommendation: "Verify nearby button exists. If not, add inline link 'Add specialty'."
    depends_on: null

  - finding_id: TW-124
    surface_id: pattern-mobile-vendor-orders-history-empty
    lens: technical-writer
    severity: P3
    issue: "Mobile-vendor 'No order history yet' is bare (4 words); web equivalent has a body."
    evidence_excerpt: "mv-orders-history-empty 'No order history yet' | vp-ux-orders-history-empty 'No orders found / You have no completed or cancelled orders yet. / No orders found for the selected time period. Try a wider date range.'"
    recommendation: "Mobile keeps concision but should add one body line: 'No completed orders yet.' Web is too wordy ('You have no completed or cancelled orders yet' → 'No completed or cancelled orders yet.')."
    depends_on: null

  - finding_id: TW-125
    surface_id: pattern-mobile-vendor-onb-fullname
    lens: technical-writer
    severity: P3
    issue: "Mobile-vendor full-name validation matches the same 'X must be at least N characters' pattern but is missing from the vendor-portal full-name validator (vendor-portal only checks 'Full name is required')."
    evidence_excerpt: "mv-onb-personal-fullname-err 'Full name must be at least 2 characters' (mobile) | vp-onb-personal-validation 'Full name is required' (vendor-portal — no length check)"
    recommendation: "Cross-app consistency: both should require a min length (2 chars) AND use the same message. Define in shared zod schema."
    depends_on: TW-032

  - finding_id: TW-126
    surface_id: pattern-mobile-vendor-policies-required
    lens: technical-writer
    severity: P3
    issue: "'Please accept the terms and conditions to continue.' — 'Please' filler; 'terms and conditions' lowercase but Stripe-Connect strings use 'Terms of Service' capitalized."
    evidence_excerpt: "mv-onb-policies-terms-required-alert 'Please accept the terms and conditions to continue.'"
    recommendation: "'Accept the Terms of Service to continue.' Cap as proper noun. Align across apps (api-error-tos, mv-onb-policies-terms-required-alert, dp-err-step5-terms)."
    depends_on: TW-071

  - finding_id: TW-127
    surface_id: pattern-mobile-vendor-menu-empty
    lens: technical-writer
    severity: P3
    issue: "'No menu items yet. Tap + to add your first item.' — fine, uses formula well, but uses 'Tap +' which depends on a visible UI affordance."
    evidence_excerpt: "mv-menu-empty 'No menu items yet. Tap + to add your first item.'"
    recommendation: "Keep — this is a model empty-state on a mobile surface. Verify the '+' affordance is always visible. If audience misses it, swap to button text: 'Tap Add to start building your menu.'"
    depends_on: null

  - finding_id: TW-128
    surface_id: pattern-mobile-vendor-reviews-empty
    lens: technical-writer
    severity: P3
    issue: "'No reviews yet. Your first review will appear here.' — friendly but slightly passive."
    evidence_excerpt: "mv-reviews-empty 'No reviews yet. Your first review will appear here.'"
    recommendation: "Action-oriented variant: 'No reviews yet. Invite customers to review their orders.' Or keep current as passive baseline if action surface is elsewhere."
    depends_on: TW-022

  - finding_id: TW-129
    surface_id: pattern-vendor-portal-menu-empty
    lens: technical-writer
    severity: P3
    issue: "Vendor-portal menu empty has two branches — empty list and filtered-empty — both wordy."
    evidence_excerpt: "vp-ux-menu-empty 'No menu items yet / Start building your menu by adding your first dish. / Add Your First Item / No items match your search / Try adjusting your search or filter criteria.'"
    recommendation: "Tighten: 'No menu items yet. Add your first dish.' + 'Add dish' button (verb-first, sentence case). Filtered: 'No items match your search.' + 'Clear filters' button. Drop 'Try adjusting...' (vague)."
    depends_on: TW-024

  - finding_id: TW-130
    surface_id: pattern-vendor-portal-menu-error
    lens: technical-writer
    severity: P3
    issue: "'Failed to load menu / Something went wrong while fetching your menu items. / Try Again' — three pieces, redundant."
    evidence_excerpt: "vp-ux-menu-error-state 'Failed to load menu / Something went wrong while fetching your menu items. / Try Again'"
    recommendation: "Single message + action: 'Couldn't load your menu. Try again.' + button 'Try again'. Drop the redundant explanatory line."
    depends_on: TW-021

  - finding_id: TW-131
    surface_id: pattern-vendor-portal-payouts-empty-error
    lens: technical-writer
    severity: P3
    issue: "Payouts page bundles empty + error states in one inventory row with 18 words."
    evidence_excerpt: "vp-ux-payouts-empty 'No payouts yet / Your payout history will appear here once you receive your first payout. / Unable to load payouts / Please try again later.'"
    recommendation: "Split clearly. Empty: 'No payouts yet. Your first payout will appear here.' Error: 'Couldn't load payouts. Try again.' Drop 'Please try again later' (vague — 'later' is non-actionable)."
    depends_on: TW-019

  - finding_id: TW-132
    surface_id: pattern-vendor-portal-earnings-error
    lens: technical-writer
    severity: P3
    issue: "'Unable to load earnings / Please try again later.' — 'Unable to' is more bureaucratic than 'Couldn't'; 'try again later' is non-actionable."
    evidence_excerpt: "vp-ux-earnings-error 'Unable to load earnings / Please try again later.'"
    recommendation: "'Couldn't load earnings. Try again.' (4 words)."
    depends_on: TW-019

  - finding_id: TW-133
    surface_id: pattern-image-validation-toast
    lens: technical-writer
    severity: P3
    issue: "Vendor-portal menu file-upload validation prefixes filename then says 'Invalid type. Allowed: JPEG, PNG, WebP.' — the {name}: prefix is a UI convention that risks overflow on long filenames."
    evidence_excerpt: "vp-ux-menu-form-image-errors '{name}: Invalid type. Allowed: JPEG, PNG, WebP. / {name}: Too large. Max 5 MB.'"
    recommendation: "Truncate filenames in UI (≤30 chars + ellipsis). Keep message tight: 'JPEG, PNG, or WebP only.' / 'Max 5 MB per file.' — and let the filename live separately above."
    depends_on: TW-039

  - finding_id: TW-134
    surface_id: pattern-staff-default-superadmin
    lens: technical-writer
    severity: P3
    issue: "Repeated phrase 'default super admin' is technical jargon for the user."
    evidence_excerpt: "api-error-staff-permissions 'Cannot change role of a default super admin' / 'Cannot deactivate a default super admin'"
    recommendation: "Rephrase user-facing: 'Cannot change role of the primary super admin.' / 'Cannot deactivate the primary super admin.' 'Primary' is clearer than 'default' to non-developers."
    depends_on: TW-092

  - finding_id: TW-135
    surface_id: pattern-no-pending-invites
    lens: technical-writer
    severity: P3
    issue: "Admin staff invitations empty: 'No pending invitations.' Period inconsistency vs other admin empties; otherwise fine."
    evidence_excerpt: "ap-staff-empty-invites 'No invitations found' | dp-empty-pending-invites 'No pending invitations.'"
    recommendation: "Align: 'No pending invitations.' across both. See TW-027."
    depends_on: TW-027

  - finding_id: TW-136
    surface_id: pattern-mobile-onboarding-step3
    lens: technical-writer
    severity: P3
    issue: "Mobile-customer preferences setup-failure: 'Setup failed / Something went wrong. Please try again.'"
    evidence_excerpt: "mc-onb-step3-error 'Setup failed / Something went wrong. Please try again.'"
    recommendation: "Already covered by TW-110; merge."
    depends_on: TW-110

  - finding_id: TW-137
    surface_id: pattern-mobile-customer-checkout-cart-empty
    lens: technical-writer
    severity: P3
    issue: "Mobile-customer checkout empty cart 'Your cart is empty.' — fine but missing recovery action visible in copy."
    evidence_excerpt: "mc-checkout-empty-cart 'Your cart is empty.'"
    recommendation: "Add action: 'Your cart is empty. Browse chefs near you.'"
    depends_on: TW-041

  - finding_id: TW-138
    surface_id: pattern-validation-event-date-future
    lens: technical-writer
    severity: P3
    issue: "Mobile-customer catering: 'Event date must be in the future' — fine but uses formal 'in the future'."
    evidence_excerpt: "mc-catering-errors 'Event date must be in the future'"
    recommendation: "Friendlier: 'Pick a date in the future.' or 'Event must be in the future.'"
    depends_on: TW-067

  - finding_id: TW-139
    surface_id: pattern-validation-event-type
    lens: technical-writer
    severity: P3
    issue: "Mobile-customer catering 'Event type is required' — bare; helper text would solve this."
    evidence_excerpt: "mc-catering-errors 'Event type is required'"
    recommendation: "Pair the field with helper text ('e.g., Birthday, Anniversary, Office party') above the error message. Per style guide §4 form labels."
    depends_on: null

  - finding_id: TW-140
    surface_id: pattern-mobile-customer-profile-validation
    lens: technical-writer
    severity: P3
    issue: "Mobile-customer profile bundles 'First name is required / Last name is required / Invalid phone number' — fine but 'Invalid phone number' is bare vs other apps that say 'Enter a valid 10-digit mobile number'."
    evidence_excerpt: "mc-profile-errors 'First name is required / Last name is required / Invalid phone number'"
    recommendation: "Align with mc-onb-step1-errors: 'Enter a valid 10-digit mobile number.' Replace 'Invalid phone number'."
    depends_on: TW-068

  - finding_id: TW-141
    surface_id: pattern-mobile-vendor-display-name
    lens: technical-writer
    severity: P3
    issue: "Mobile-vendor 'Display name is required.' uses period; vendor-portal 'Full name is required' uses no period. Cross-app drift."
    evidence_excerpt: "mv-profile-displayname-required 'Display name is required.' (period) vs vp-onb-personal-validation 'Full name is required' (no period)"
    recommendation: "Pick period rule (TW-027) and apply."
    depends_on: TW-027

  - finding_id: TW-142
    surface_id: pattern-mobile-vendor-update-failures
    lens: technical-writer
    severity: P3
    issue: "Three near-identical 'Failed to update X. Please try again.' on mobile-vendor."
    evidence_excerpt: "mv-menuedit-update-fail 'Failed to update menu item. Please try again.' | mv-profile-update-fail 'Failed to update profile. Please try again.' | mv-menunew-create-fail 'Failed to create menu item. Please try again.'"
    recommendation: "Helper template: 'Couldn't {update|create} {thing}. Try again.' Drop 'Please' (TW-025), 'Failed to' (TW-026). Implement as a `failureToast(verb, thing)` helper."
    depends_on: TW-026

  - finding_id: TW-143
    surface_id: pattern-mobile-vendor-photo-delete-fail
    lens: technical-writer
    severity: P3
    issue: "'Failed to delete photo.' (3 words, no recovery)."
    evidence_excerpt: "mv-menuedit-delete-photo-fail 'Failed to delete photo.'"
    recommendation: "'Couldn't delete photo. Try again.' (5 words)."
    depends_on: TW-019

  - finding_id: TW-144
    surface_id: pattern-mobile-delivery-tone
    lens: technical-writer
    severity: P3
    issue: "Mobile-delivery 'Accept a delivery from the Available tab to get started.' is 10 words — driver-tone matrix targets ≤4 words. Long for a glance surface."
    evidence_excerpt: "md-emp-002 'Accept a delivery from the Available tab to get started.'"
    recommendation: "Tighten: 'Open Available to start.' (4 words). Pair with the Available tab icon (visual context covers the rest)."
    depends_on: null

  - finding_id: TW-145
    surface_id: pattern-mobile-delivery-available-empty
    lens: technical-writer
    severity: P3
    issue: "'No deliveries available nearby. Pull to refresh.' is 7 words — fine for driver. Period after 'nearby' could feel abrupt; comma might flow better; but rule is fine."
    evidence_excerpt: "md-err-002 'No deliveries available nearby. Pull to refresh.'"
    recommendation: "Keep. Optional polish: 'No deliveries nearby — pull to refresh.' (5 words)."
    depends_on: null

  - finding_id: TW-146
    surface_id: pattern-mobile-delivery-staff-lock
    lens: technical-writer
    severity: P3
    issue: "Mobile-delivery staff/fleet locked screens use 'Contact your administrator to request access.' — formal; long for driver tone."
    evidence_excerpt: "md-emp-007 'Fleet management is available for fleet managers only. Contact your administrator to request access.' | md-emp-011 'Staff management requires manager permissions. Contact your administrator to request access.'"
    recommendation: "Driver-tone: 'Fleet managers only. Ask your admin for access.' / 'Manager permissions needed. Ask your admin for access.' Tighter, same meaning."
    depends_on: TW-051

  - finding_id: TW-147
    surface_id: pattern-csv-vs-comma-list
    lens: technical-writer
    severity: P3
    issue: "File-upload type-error lists use comma-separated formats: 'JPEG, PNG, WebP, PDF'. Customer-facing should use Oxford comma or 'or' — currently ambiguous as 'and' vs 'or'."
    evidence_excerpt: "api-error-upload-type 'Invalid file type. Allowed: JPEG, PNG, WebP, PDF' | vp-ux-menu-form-image-errors 'Allowed: JPEG, PNG, WebP.'"
    recommendation: "Use 'or' for either/or lists: 'Use JPEG, PNG, WebP, or PDF.' Apply across all upload errors. Note: keeping the list short (3-4) is fine; longer lists should use a help-link instead."
    depends_on: TW-039

  - finding_id: TW-148
    surface_id: pattern-mobile-vendor-cuisines-error
    lens: technical-writer
    severity: P3
    issue: "'Select at least one cuisine type' / 'Select at least one cuisine' / 'Select at least one dietary tag' — minor drift on whether to use 'type'."
    evidence_excerpt: "mv-onb-kitchen-cuisines-err 'Select at least one cuisine type' | vp-onb-kitchen-validation 'Select at least one cuisine' | vp-ux-menu-form-validation 'Select at least one dietary tag'"
    recommendation: "Drop 'type': 'Select at least one cuisine.' Apply across apps."
    depends_on: null

  - finding_id: TW-149
    surface_id: pattern-driver-onboarding-prereq
    lens: technical-writer
    severity: P3
    issue: "Five 'Please complete X' variants on driver onboarding say the same thing slightly differently."
    evidence_excerpt: "api-error-chef-onboarding 'Please complete all previous onboarding steps before submitting' / 'Please complete earlier onboarding steps first' / 'Please complete step 1 (personal details) first' / 'Please upload at least one document before submitting' / 'Please indicate if your bicycle can carry a delivery box'"
    recommendation: "Standardize: 'Complete step {N} ({label}) before submitting.' Generate via parameter. Drop 'Please' (TW-025)."
    depends_on: TW-025

  - finding_id: TW-150
    surface_id: pattern-driver-onboarding-state
    lens: technical-writer
    severity: P3
    issue: "Driver application status: 'Your application has already been approved' / 'Your application is already under review' — fine voice but lacks an action."
    evidence_excerpt: "api-error-chef-onboarding 'Your application has already been approved' / 'Your application is already under review'"
    recommendation: "Pair with action: 'Your application is already approved. Open the dashboard to start.' / 'Your application is already under review. We'll notify you when it's decided.'"
    depends_on: null

  - finding_id: TW-151
    surface_id: pattern-validate-vs-zod-summary
    lens: technical-writer
    severity: P3
    issue: "Inventory shows vp-ux-menu-form-validation has 60 words bundling 15 Zod messages — that's expected of a Zod summary but should never display as one toast."
    evidence_excerpt: "vp-ux-menu-form-validation (15 messages, 60 words)"
    recommendation: "Verify inline display: each Zod message attaches to its field, never bundled as toast. If a summary toast is desired: 'Some fields need attention.' + scroll to first error."
    depends_on: TW-112

  - finding_id: TW-152
    surface_id: pattern-vendor-onboarding-min-radius
    lens: technical-writer
    severity: P3
    issue: "Vendor-portal: 'Minimum 1 km radius' (4 words, no period). Style guide expects sentence form."
    evidence_excerpt: "vp-onb-ops-radius-validation 'Minimum 1 km radius'"
    recommendation: "'Service radius must be at least 1 km.' (matches mobile-vendor mv-onb-ops-radius-err format)."
    depends_on: null

  - finding_id: TW-153
    surface_id: pattern-mobile-vendor-radius-range
    lens: technical-writer
    severity: P3
    issue: "Mobile-vendor radius validation: 'Service radius must be between 1 and 50 km' — bound on both ends; vendor-portal only enforces minimum. Inconsistency."
    evidence_excerpt: "mv-onb-ops-radius-err 'Service radius must be between 1 and 50 km' vs vp-onb-ops-radius-validation 'Minimum 1 km radius'"
    recommendation: "Align bounds. If max is 50 km on both, message: 'Service radius must be between 1 and 50 km.' Centralize zod."
    depends_on: TW-32

  - finding_id: TW-154
    surface_id: pattern-driver-step-payout
    lens: technical-writer
    severity: P3
    issue: "Driver onboarding payout errors use bare 'Please fill in all bank details' (5 words, 'Please' filler)."
    evidence_excerpt: "dp-err-payout-bank 'Please fill in all bank details' | dp-err-payout-upi 'Please enter your UPI ID'"
    recommendation: "Drop 'Please': 'Fill in all bank details.' / 'Enter your UPI ID.' Driver tone tighter."
    depends_on: TW-025

  - finding_id: TW-155
    surface_id: pattern-driver-onboarding-step1-required
    lens: technical-writer
    severity: P3
    issue: "Driver onboarding step1 'Please fill in all required fields' (5 words) is reused across step2 — overgeneralized message."
    evidence_excerpt: "dp-err-step1-required 'Please fill in all required fields' (reused step2)"
    recommendation: "Per-field inline validation handles this; if a summary toast is needed: 'Some required fields are missing.' Drop 'Please'."
    depends_on: TW-112

  - finding_id: TW-156
    surface_id: pattern-driver-step-load-review
    lens: technical-writer
    severity: P3
    issue: "'Failed to load review data' on driver step-5 — bare, no recovery."
    evidence_excerpt: "dp-err-step5-load 'Failed to load review data'"
    recommendation: "'Couldn't load your application data. Try again.' (7 words)."
    depends_on: TW-019

  - finding_id: TW-157
    surface_id: pattern-driver-step-submit
    lens: technical-writer
    severity: P3
    issue: "'Failed to submit. Please try again.' on driver step-5 submission."
    evidence_excerpt: "dp-err-step5-submit 'Failed to submit. Please try again.'"
    recommendation: "'Couldn't submit your application. Try again.' (6 words)."
    depends_on: TW-026

  - finding_id: TW-158
    surface_id: pattern-driver-plan-load
    lens: technical-writer
    severity: P3
    issue: "'Failed to load subscription plans' (4 words) on driver step-4."
    evidence_excerpt: "dp-err-step4-plans-load 'Failed to load subscription plans'"
    recommendation: "'Couldn't load plans. Try again.' (5 words)."
    depends_on: TW-019

  - finding_id: TW-159
    surface_id: pattern-driver-plan-select
    lens: technical-writer
    severity: P3
    issue: "'Failed to select plan. Please try again.' (6 words)."
    evidence_excerpt: "dp-err-step4-plan-failed 'Failed to select plan. Please try again.'"
    recommendation: "'Couldn't select that plan. Try again.' (6 words)."
    depends_on: TW-026

  - finding_id: TW-160
    surface_id: pattern-driver-invite-create
    lens: technical-writer
    severity: P3
    issue: "'Failed to create invitation' (3 words, no recovery)."
    evidence_excerpt: "dp-err-invite-create 'Failed to create invitation'"
    recommendation: "'Couldn't create invitation. Try again.' (5 words)."
    depends_on: TW-019

  - finding_id: TW-161
    surface_id: pattern-driver-invite-required
    lens: technical-writer
    severity: P3
    issue: "'Email and role are required' (5 words) — fine but missing period vs other admin tone strings."
    evidence_excerpt: "dp-err-invite-required 'Email and role are required'"
    recommendation: "Add period if used as sentence ('Email and role are required.'). Per TW-027."
    depends_on: TW-027

  - finding_id: TW-162
    surface_id: pattern-driver-partner-status
    lens: technical-writer
    severity: P3
    issue: "Driver fleet admin: 'Failed to update partner status' / 'Failed to verify partner' — bare, no recovery."
    evidence_excerpt: "dp-err-partner-status 'Failed to update partner status' | dp-err-partner-verify 'Failed to verify partner'"
    recommendation: "'Couldn't update partner status. Try again.' / 'Couldn't verify partner. Try again.'"
    depends_on: TW-019

  - finding_id: TW-163
    surface_id: pattern-driver-accept-failed
    lens: technical-writer
    severity: P3
    issue: "Driver mobile 'Failed to accept delivery' (3 words) — high-stakes moment, deserves better recovery copy."
    evidence_excerpt: "dp-err-accept-failed 'Failed to accept delivery'"
    recommendation: "Driver-tone with action: 'Couldn't accept. Try again, or pick another.' (≤6 words, gives a fallback)."
    depends_on: TW-019

  - finding_id: TW-164
    surface_id: pattern-driver-upload-fail
    lens: technical-writer
    severity: P3
    issue: "'Upload failed. Please try again.' / 'File too large. Max {n}MB for {label}' on driver onboarding documents."
    evidence_excerpt: "dp-err-step3-upload 'Upload failed. Please try again.' | dp-err-step3-file-size 'File too large. Max {n}MB for {label}'"
    recommendation: "Drop 'Please' (TW-025): 'Upload failed. Try again.' Standardize size unit per TW-005: 'File too large. Max {n} MB for {label}.'"
    depends_on: TW-005

  - finding_id: TW-165
    surface_id: pattern-driver-referral-invalid
    lens: technical-writer
    severity: P3
    issue: "'Invalid referral code' (3 words) — bare. For driver onboarding where bonuses depend on referrals, deserves clear recovery."
    evidence_excerpt: "dp-err-step1-referral-invalid 'Invalid referral code'"
    recommendation: "'Referral code not recognized. Skip this field if you don't have one.' Helps the driver continue rather than stalling."
    depends_on: null

```

## Legal findings

```yaml
# Legal lens findings — ERRORS-EMPTY category (282 surfaces)
# Scope: India regulatory exposure (DPDP Act 2023, FSSAI 2011/2018, RBI PA MD 2020/2024,
# CGST/GST 2017, Consumer Protection Act 2019 + E-Commerce Rules 2020, IT Act 2000 §43/§43A/§66,
# IT Rules 2021, MV Act 1988 for drivers) + generic best-practice
# IMPORTANT: Every finding carries depends_on: "needs lawyer review" — this audit is not legal advice.
# Focus for errors-empty: information leakage (API paths, snake_case slugs, internal config),
# account-existence disclosure, payment-error categorisation (RBI), refund-timeline communication,
# misleading "try again" copy on permanent failures, PII in error messages (DPDP §8(5)),
# misleading regulatory/empty-state claims, T&C acknowledgement errors without link to T&Cs,
# driver/chef onboarding rejection notices (gig classification + DPDP rights to know),
# permission-denial framing (Consumer Protection E-Commerce Rules §4(5) on transparent redress).

findings:

  # ============================================================================
  # API INFORMATION LEAKAGE — IT Act §43 / DPDP §8(5) reasonable security
  # Internal paths, snake_case slugs, raw config errors surface to end users
  # ============================================================================

  - finding_id: LEG-ERR-001
    surface_id: api-error-stripe-connect
    lens: legal
    severity: P0
    issue: "Error messages leak internal API routes to end users ('call /chef/stripe/connect first', 'call /delivery/stripe/connect first'). This is internal architecture disclosure — a reasonable-security failure under IT Act §43A / DPDP Act §8(5) (Data Fiduciary obligation to maintain reasonable security)."
    evidence_excerpt: "No Stripe account — call /chef/stripe/connect first / No Stripe account — call /delivery/stripe/connect first"
    recommendation: "Strip all internal route paths from user-facing errors. Map server-side error codes to user-friendly text on the client. Surface a remediation in user terms (e.g., 'Finish payout setup before connecting Stripe'). Server logs should retain the technical detail; the API response should not. Cite as reasonable-security gap."
    citation: "IT Act 2000 §43A; DPDP Act 2023 §8(5); CERT-In Directions 28 Apr 2022 (incident-reporting context for poor security hygiene)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-002
    surface_id: api-error-payment-config
    lens: legal
    severity: P0
    issue: "Payment-gateway configuration error leaks internal infrastructure status to the customer ('Payment gateway not configured', 'Stripe gateway not configured by platform admin'). Customer learns the platform has mis-configured a payment processor — undermines trust and exposes operational state. Under RBI PA framework merchant must communicate availability transparently but must not disclose backend implementation."
    evidence_excerpt: "Payment gateway not configured / Stripe gateway not configured / Stripe gateway not configured by platform admin / Stripe gateway not configured"
    recommendation: "Replace with a single user-facing message that does not name vendors or admin state (e.g., 'This chef cannot accept payment right now. Try again later or pick another chef.'). Keep vendor name (Razorpay / Stripe) in logs only. RBI PA MD requires transparency about payment availability, not about internal configuration."
    citation: "RBI Payment Aggregator MD 17 Mar 2020 §7 (Customer Grievance); IT Act 2000 §43A"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-003
    surface_id: api-error-delivery-already-active
    lens: legal
    severity: P1
    issue: "Snake_case slug 'no_active_delivery' is surfaced to the user as an error string. This is an internal enum value leaking through to UI — not a translation key resolved into a sentence. Suggests inconsistent error envelope contract and undermines plain-language obligation."
    evidence_excerpt: "You already have an active delivery / no_active_delivery"
    recommendation: "All user-facing error strings must be translated sentences, not enum slugs. Establish an error-envelope contract: `{ code: 'no_active_delivery', message: '<localised sentence>' }`. Audit all handlers for similar leaks. Cite Consumer Protection (E-Commerce) Rules §4(5) on clear consumer communication."
    citation: "Consumer Protection (E-Commerce) Rules 2020 r.4(5); IT Act 2000 §43A"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-004
    surface_id: api-error-generic-failed
    lens: legal
    severity: P1
    issue: "100+ variants of generic 'Failed to X' errors surface to all user roles without remediation guidance, classification, or contact pathway. Consumer Protection (E-Commerce) Rules §4(5) requires a transparent redress mechanism — opaque 'Failed to fetch X' messages do not meet this standard and may be deemed unfair trade practice if persistent."
    evidence_excerpt: "Failed to accept quote / Failed to create order / Failed to fetch orders / Failed to process refund / Failed to verify partner / [~100 variants]"
    recommendation: "Define an error taxonomy with a finite set of user-facing messages, each carrying: (a) what happened in plain English, (b) one recovery action, (c) a grievance link when applicable. The Grievance Officer (DPDP §10) must be reachable from any persistent failure. Generic 'Failed to X' is acceptable only as last-resort fallback with contact-support CTA."
    citation: "Consumer Protection (E-Commerce) Rules 2020 r.4(5), r.5(3); DPDP Act 2023 §10 (Grievance Officer); CP Act 2019 §2(47) (unfair trade practice)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-005
    surface_id: api-error-validation-required
    lens: legal
    severity: P2
    issue: "Field-name casing leaks internal API contract to users ('chefId is required', 'currencyCode is required', 'stripePaymentIntentId is required'). camelCase identifiers are programmer-facing; consumer-facing errors must use human field names."
    evidence_excerpt: "Email is required / chefId is required / stripePaymentIntentId is required / currencyCode is required"
    recommendation: "Map each API field to a user-facing label. Surface 'Chef is required', not 'chefId is required'. Audit all 18 variants. Plain-language obligation under Consumer Protection (E-Commerce) Rules §4."
    citation: "Consumer Protection (E-Commerce) Rules 2020 r.4(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-006
    surface_id: api-error-validation-invalid-id
    lens: legal
    severity: P2
    issue: "~40 variants of 'Invalid X' errors include internal entity types ('Invalid payload', 'Invalid signature', 'Invalid OAuth provider', 'Invalid applicableTo'). These leak implementation detail and provide zero remediation."
    evidence_excerpt: "Invalid signature / Invalid OAuth provider / Invalid applicableTo / Invalid request body / Invalid payload"
    recommendation: "Replace developer-facing 'Invalid X' messages with user-actionable text. 'Invalid signature' on a webhook should never reach the customer UI — segregate webhook errors from user-API errors. Consumer-facing errors must point to a fix; technical errors belong in logs."
    citation: "Consumer Protection (E-Commerce) Rules 2020 r.4(5); IT Act 2000 §43A"
    depends_on: "needs lawyer review"

  # ============================================================================
  # ACCOUNT-EXISTENCE DISCLOSURE — DPDP / privacy / OWASP A07
  # ============================================================================

  - finding_id: LEG-ERR-007
    surface_id: api-error-auth-invalid-credentials
    lens: legal
    severity: P1
    issue: "Login error 'Invalid credentials' is correctly ambiguous and does not disclose whether email or password is wrong — privacy-safe. Document this as the canonical pattern. However, the codebase ALSO has 'Invalid email or password' variants (web-err-login-validation, web-err-auth-service, dp-err-invalid-credentials, ap-auth-error-invalid-creds, vp-auth-login-invalid-credentials) which read fine but inconsistency across portals creates risk that an account-enumeration variant slips in."
    evidence_excerpt: "Invalid credentials (api) / Invalid email or password (web, vp, dp, ap)"
    recommendation: "Standardise across all portals + API to one ambiguous message. Forbid any login error that distinguishes 'no such user' from 'wrong password'. Document in a security style guide; cite as DPDP §8(5) reasonable-security control against account enumeration. Re-audit forgot-password, register-existing, and OAuth flows."
    citation: "DPDP Act 2023 §8(5); IT Act 2000 §43A; OWASP ASVS V2.5 (best practice)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-008
    surface_id: api-error-phone-duplicate
    lens: legal
    severity: P0
    issue: "Registration error 'This phone number is already registered with another account' confirms account existence to an unauthenticated user — classic account-enumeration vulnerability. An attacker can probe whether a phone is on the platform. DPDP §8(5) reasonable-security failure."
    evidence_excerpt: "This phone number is already registered with another account."
    recommendation: "Replace with privacy-safe pattern: on registration submit, ALWAYS return success state and (if phone exists) send an SMS to the owner explaining a sign-up attempt was made; never confirm existence to the requesting browser. Same applies to email duplicate checks. Cite DPDP §8(5) and Consumer Protection (E-Commerce) Rules §5(3) on consent/privacy."
    citation: "DPDP Act 2023 §8(5); IT Act 2000 §43A; OWASP ASVS V3.2.3"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-009
    surface_id: api-error-favorites
    lens: legal
    severity: P3
    issue: "'Chef is already in your favorites' discloses authenticated-user state — acceptable inside the customer session, but flag for consistency if the favorites API is ever reused unauthenticated."
    evidence_excerpt: "Chef is already in your favorites"
    recommendation: "Verify the favorites endpoint requires session auth before responding. If endpoint is ever reachable unauthenticated, this becomes an enumeration vector for which chefs a known user favorites."
    citation: "DPDP Act 2023 §8(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-010
    surface_id: vp-auth-login-access-denied
    lens: legal
    severity: P1
    issue: "Vendor-portal access-denied error 'This portal is only for vendor accounts. Please use the Fe3dr customer app.' confirms (a) that the credentials are valid and (b) that the account is a customer account. This is partial account-type disclosure to whoever holds the credentials. Acceptable if behind successful authn, but the message reveals account category."
    evidence_excerpt: "This portal is only for vendor accounts. Please use the Fe3dr customer app."
    recommendation: "If returned AFTER successful auth, this is privacy-acceptable but verify. If returned during auth flow (pre-token-issue), do not disclose role. Also: 'Fe3dr' brand reference should be confirmed correct/intentional — looks like a typo or legacy brand name that may mislead users."
    citation: "DPDP Act 2023 §8(5); CP Act 2019 §2(47) on misleading text"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-011
    surface_id: ap-auth-error-access-denied
    lens: legal
    severity: P1
    issue: "Admin-portal access-denied error 'Access denied. Only administrators can sign in to this portal.' confirms credentials are valid for a non-admin role — partial role enumeration. RBAC framing also flagged as 'legal-adjacent' in inventory notes."
    evidence_excerpt: "Access denied. Only administrators can sign in to this portal."
    recommendation: "Either (a) keep admin portal entirely separate so non-admins never reach this page, or (b) return generic 'Invalid credentials' without distinguishing role mismatch from password mismatch. Document admin auth flow in security policy. Cite DPDP §8(5)."
    citation: "DPDP Act 2023 §8(5); IT Act 2000 §43A"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-012
    surface_id: dp-err-access-denied
    lens: legal
    severity: P2
    issue: "Driver-portal 'Access denied. Please check your credentials and try again.' — phrasing is acceptable (does not reveal which credential is wrong) but the verb 'Access denied' implies authorisation failure rather than authentication failure. May mislead a legitimate driver whose credentials are correct but whose account is suspended into trying again instead of contacting support."
    evidence_excerpt: "Access denied. Please check your credentials and try again."
    recommendation: "Differentiate authn failures ('Invalid credentials') from authz / suspension states ('Your driver account is not active. Contact support: <Grievance Officer>'). DPDP §10 requires Grievance Officer contact for account-state issues; this generic error does not point to one."
    citation: "DPDP Act 2023 §10; Consumer Protection (E-Commerce) Rules 2020 r.4(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-013
    surface_id: api-error-auth-suspended
    lens: legal
    severity: P0
    issue: "'Account is suspended' provides no reason, no appeal pathway, no Grievance Officer contact. Suspension is a deprivation of platform access affecting livelihood for chefs and drivers — DPDP §10 (Grievance Officer) AND Consumer Protection (E-Commerce) Rules §5(3) (transparent grievance redress in 1 month) BOTH require redress contact. The driver gig-worker context heightens this (right to procedural fairness)."
    evidence_excerpt: "Account is suspended"
    recommendation: "Suspension errors must surface: (i) brief reason category (e.g., 'safety review', 'document expired'), (ii) appeal link or Grievance Officer contact email, (iii) expected timeline. Cite DPDP §10, CP E-Commerce Rules §5(3), and emerging Social Security Code 2020 §141 (gig-worker grievance) considerations."
    citation: "DPDP Act 2023 §10; Consumer Protection (E-Commerce) Rules 2020 r.5(3); Code on Social Security 2020 §141 (gig workers)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-014
    surface_id: api-error-auth-not-verified
    lens: legal
    severity: P2
    issue: "Two variants 'Account not verified' and 'Account not verified yet' on the same handler — inconsistent. Neither points to the verification path or a Grievance Officer if the verification email never arrived."
    evidence_excerpt: "Account not verified / Account not verified yet"
    recommendation: "Standardise to one message that includes a 'Resend verification' action and a fallback contact. Inconsistent error copy weakens plain-language obligation under Consumer Protection (E-Commerce) Rules §4."
    citation: "Consumer Protection (E-Commerce) Rules 2020 r.4(5); DPDP Act 2023 §10"
    depends_on: "needs lawyer review"

  # ============================================================================
  # SESSION / TOKEN ERRORS — PII / data-handling on reconnect
  # ============================================================================

  - finding_id: LEG-ERR-015
    surface_id: web-err-login-session-expired
    lens: legal
    severity: P2
    issue: "Session-expired banner 'Your session has expired. Please sign in again.' is correct in tone but does not inform the user what happened to any unsaved data, whether cart/draft data persists across re-auth, or how DPDP-relevant session data is handled at expiry. Same pattern repeats across vp-auth-login-session-expired, ap-auth-error-session-expired, dp-err-session-expired."
    evidence_excerpt: "Your session has expired. Please sign in again."
    recommendation: "Surface (or link to) a short data-handling note at the session-expired moment for surfaces that hold drafts (cart, menu draft, catering request): 'Your cart is saved for 24 hours.' DPDP §6(2)(d) requires the data principal to be informed about retention. At minimum, link to Privacy Policy from the expired-session banner."
    citation: "DPDP Act 2023 §6(2)(d) (notice content); §8(3) (retention)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-016
    surface_id: api-error-auth-reset-expired
    lens: legal
    severity: P2
    issue: "Three near-duplicate variants 'Reset token has expired', 'Invalid or expired reset token', 'Invalid or expired enrollment token' — inconsistent across handler. Password-reset and 2FA-enrollment token errors must not allow the requester to distinguish 'no such token' from 'expired token' (subtle enumeration)."
    evidence_excerpt: "Reset token has expired / Invalid or expired reset token / Invalid or expired enrollment token"
    recommendation: "Collapse to one message that does not distinguish 'invalid' from 'expired' — both should read 'This link is no longer valid. Request a new password reset.' Cite DPDP §8(5) and OWASP ASVS V3.2."
    citation: "DPDP Act 2023 §8(5); IT Act 2000 §43A"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-017
    surface_id: api-error-auth-current-pw
    lens: legal
    severity: P2
    issue: "Two variants 'Current password is incorrect' and 'Password is incorrect' on the same handler — inconsistent. Both are acceptable in an authenticated change-password flow."
    evidence_excerpt: "Current password is incorrect / Password is incorrect"
    recommendation: "Standardise. Voice consistency is a plain-language obligation under Consumer Protection rules; not a security issue here."
    citation: "Consumer Protection (E-Commerce) Rules 2020 r.4(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-018
    surface_id: api-error-auth-2fa
    lens: legal
    severity: P1
    issue: "All 2FA errors are vague 'Failed to enable 2FA', 'Failed to start 2FA challenge', 'Invalid or expired challenge' with no remediation. 2FA is a DPDP §8(5) reasonable-security control — silent failure of a security control without operator guidance is a security and audit-trail concern."
    evidence_excerpt: "Failed to enable 2FA / Failed to start 2FA challenge / Failed to start 2FA enrollment / Invalid or expired challenge"
    recommendation: "Each 2FA error must surface (a) plain reason, (b) recovery (re-scan QR, generate new backup code, contact admin), (c) audit-log reference for admin escalation. DPDP §8(5) reasonable-security obligation."
    citation: "DPDP Act 2023 §8(5); IT Act 2000 §43A"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-019
    surface_id: web-err-auth-service
    lens: legal
    severity: P2
    issue: "Fallback errors 'Invalid email or password', 'Registration failed', 'Token refresh failed' thrown client-side when API returns no message — the third one ('Token refresh failed') is a session/security failure that should redirect to login, not surface as a toast. Surfacing a refresh-token failure may confuse user about data state."
    evidence_excerpt: "Invalid email or password / Registration failed / Token refresh failed"
    recommendation: "Token refresh failure should silently redirect to login with session-expired banner — not display 'Token refresh failed' as user-visible text (developer language). DPDP §6(2)(d) notice on retention/expiry."
    citation: "DPDP Act 2023 §6(2)(d), §8(5); Consumer Protection (E-Commerce) Rules 2020 r.4(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-020
    surface_id: dp-err-auth-service-refresh
    lens: legal
    severity: P2
    issue: "Driver-portal surfaces 'Token refresh failed' as user-visible text. Drivers are in motion; developer-language errors during a live shift are unsafe and not consistent with gig-worker procedural fairness expectations."
    evidence_excerpt: "Token refresh failed"
    recommendation: "Suppress developer-language session errors. Driver-app should silently re-auth and surface only 'Signed out — sign in to continue.' Critical because driver-tracking, deliveries, earnings are mid-stream when this fires."
    citation: "Code on Social Security 2020 §141; DPDP Act 2023 §8(5)"
    depends_on: "needs lawyer review"

  # ============================================================================
  # OFFLINE / NETWORK ERRORS — DPDP data-handling on reconnect
  # ============================================================================

  - finding_id: LEG-ERR-021
    surface_id: web-err-layout-offline
    lens: legal
    severity: P2
    issue: "Offline banner 'You're offline. Some features may be unavailable.' does not clarify what happens to inputs (cart additions, order placements) during offline — whether queued locally, what PII is held in browser storage, and whether queued data persists when the device goes back online. DPDP §6(2)(d) requires clarity on processing and retention."
    evidence_excerpt: "You're offline. Some features may be unavailable."
    recommendation: "Add a one-line clarifier when relevant context is at stake (cart, checkout, order tracking): 'Your cart is saved locally and will sync when reconnected.' Link to Privacy Policy section on local storage. DPDP §6 notice obligations."
    citation: "DPDP Act 2023 §6(2)(d), §8(3)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-022
    surface_id: vp-ux-layout-offline
    lens: legal
    severity: P2
    issue: "Vendor offline banner 'You're offline. Orders will sync when connected.' makes an operational promise ('orders will sync') without explaining data integrity guarantees, what happens to a mark-ready action taken offline, or the impact on customer-side timer. If the sync fails or is delayed, customers see stale state — risk of unfair-trade-practice signal."
    evidence_excerpt: "You're offline. Orders will sync when connected."
    recommendation: "Quantify: 'Last sync 7 min ago. Actions taken now will queue and sync when reconnected.' Lawyer to review the operational warranty implied by 'will sync' — set realistic expectations and document divergence behaviour. Consumer Protection §2(47) unfair-trade-practice exposure if customers rely on stale chef state."
    citation: "CP Act 2019 §2(47); CP (E-Commerce) Rules 2020 r.4(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-023
    surface_id: dp-err-offline-banner
    lens: legal
    severity: P1
    issue: "Driver offline banner 'You're offline. Updates will sync when connected.' is a contractual statement during active delivery. If a delivery 'Picked Up' or 'Delivered' status taken offline doesn't sync, customers, chef, and platform all see divergent state — wider exposure than vendor offline. MV Act / consumer-protection ramifications if a delivered order shows undelivered or vice-versa."
    evidence_excerpt: "You're offline. Updates will sync when connected."
    recommendation: "Specify queue size, last sync time, and risk: 'Offline since 12:34. 2 status updates queued. Order tracking on customer side may show stale.' Driver app must also clearly state DPDP retention of location data taken offline. Refer to Motor Vehicles Aggregator Guidelines 2020 §4 on data handling."
    citation: "MV Act 1988; MV Aggregator Guidelines 2020 §4; DPDP Act 2023 §6, §8; CP Act 2019 §2(47)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-024
    surface_id: ap-layout-offline-banner
    lens: legal
    severity: P2
    issue: "Admin offline banner 'You're offline. Data may not be up to date.' is fine for admins, but admins make moderation/suspension decisions; the banner does not warn that actions taken offline (suspend user, approve chef) may race with concurrent server state. Audit-trail integrity (DPDP §8(5)) concern."
    evidence_excerpt: "You're offline. Data may not be up to date."
    recommendation: "For admin destructive actions, gate behind an online check or require explicit 'I understand state may be stale' confirmation. Audit-log entries should mark offline-origin actions for compliance review."
    citation: "DPDP Act 2023 §8(5), §8(7) (audit obligations); IT Rules 2021 r.3 (record-keeping)"
    depends_on: "needs lawyer review"

  # ============================================================================
  # PAYMENT / REFUND / RBI EXPOSURE
  # ============================================================================

  - finding_id: LEG-ERR-025
    surface_id: api-error-payment-refund
    lens: legal
    severity: P0
    issue: "Refund-error responses leak who-can-initiate ('Only the chef or admin can initiate refunds'), processor identity ('No Razorpay payment found for this order', 'No Stripe payment found for this order'), and processor state ('Payment not captured') to the customer. RBI Payment Aggregator MD requires customer-visible refund policy disclosure but does NOT require leakage of which processor or who-can-act. More importantly, NO refund TIMELINE is communicated — RBI PA MD requires explicit refund SLA (T+1 instruction to issuer, customer-visible)."
    evidence_excerpt: "Can only refund completed payments / Refund amount cannot exceed order total / Only the chef or admin can initiate refunds / Failed to process refund / No Razorpay payment found for this order / No Stripe payment found for this order / Payment not captured"
    recommendation: "Refund errors must (a) NOT name processors to the customer, (b) state the SLA explicitly when accepted ('Refund initiated. Funds arrive in your account in 5-7 working days as per RBI guidelines.'), (c) provide Grievance Officer contact when refused. Separate internal error codes from customer-facing copy. Lawyer to draft RBI-compliant refund-error matrix."
    citation: "RBI Payment Aggregator MD 17 Mar 2020 §6, §7; RBI Master Direction on Credit/Debit Cards 2022 (refund timelines); Consumer Protection (E-Commerce) Rules 2020 r.5(3)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-026
    surface_id: web-err-checkout-cancel
    lens: legal
    severity: P1
    issue: "Razorpay dismissal toast says only 'Payment cancelled'. Customer is left without confirmation that no money was captured, no record was created, and what happens next. RBI PA framework requires transparent communication of payment state."
    evidence_excerpt: "Payment cancelled"
    recommendation: "Expand to: 'Payment cancelled. No charge was made. Your cart is saved — try again when ready.' RBI PA MD §7 (Customer Grievance) requires clarity on payment outcome including cancellation."
    citation: "RBI Payment Aggregator MD 17 Mar 2020 §7"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-027
    surface_id: mc-checkout-errors
    lens: legal
    severity: P0
    issue: "Mobile customer checkout error 'Payment confirmation timed out. Check your order history to confirm status.' is a high-risk message: the customer paid (or may have), but the app does not know. Currently directs customer to self-resolve via order history. Under RBI PA MD §7 the platform MUST communicate definitive outcome OR initiate auto-reconciliation with a stated SLA, not push the burden to the customer."
    evidence_excerpt: "Order creation failed. Please try again. / Payment was not completed. Please try again. / Payment confirmation timed out. Check your order history to confirm status. / Dismiss"
    recommendation: "Rewrite with explicit reconciliation promise: 'Payment status pending. We're confirming with your bank. You'll get a confirmation in <X minutes>. If charged but no order created, the amount is auto-refunded within Y days per RBI rules.' Cite RBI PA MD on reconciliation duties and dispute pathways. The 'Order creation failed. Please try again.' is also dangerous — risks double-charge if user retries while first attempt was actually accepted."
    citation: "RBI Payment Aggregator MD 17 Mar 2020 §7; RBI Harmonisation of TAT for Failed Transactions Circular 20 Sep 2019 (T+5 auto-reversal)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-028
    surface_id: api-error-order-already-paid
    lens: legal
    severity: P1
    issue: "'Order already paid' is correct but offers no remediation if customer believes they were not paid. RBI failed-transaction harmonisation circular requires platform to surface dispute/reconciliation path."
    evidence_excerpt: "Order already paid"
    recommendation: "Expand: 'Order already paid. If you don't see the charge on your bank statement, contact our Grievance Officer at <email> within 7 days.' Cite RBI 20-Sep-2019 circular."
    citation: "RBI Harmonisation of TAT Circular 20 Sep 2019; RBI PA MD 17 Mar 2020 §7"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-029
    surface_id: api-error-order-cancel-stage
    lens: legal
    severity: P0
    issue: "'Order cannot be cancelled at this stage' surfaces no refund/no-refund implication, no contact pathway. Customer's right to cancel and entitlement to refund is core to Consumer Protection (E-Commerce) Rules §5 — a flat refusal without explaining refund consequences or escalation is a CP-Rules failure."
    evidence_excerpt: "Order cannot be cancelled at this stage"
    recommendation: "Expand: 'Your order is already being prepared and cannot be cancelled. To dispute or report an issue, contact the chef directly or our support team within 24 hours of delivery.' Reference cancellation/refund policy and Grievance Officer per CP E-Commerce Rules §5(3) and DPDP §10."
    citation: "Consumer Protection (E-Commerce) Rules 2020 r.5; DPDP Act 2023 §10; CP Act 2019 §2(47)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-030
    surface_id: api-error-chef-not-accepting
    lens: legal
    severity: P2
    issue: "'Chef is not accepting orders' offers no alternative or reason. Borderline unfair-trade-practice if a chef is shown in browse results but rejects at order time — customer expectation mismatch."
    evidence_excerpt: "Chef is not accepting orders"
    recommendation: "Expand with 'This chef has paused new orders. Browse other chefs nearby.' Also: ensure browse-time filters surface 'open now' state correctly to avoid the order-time rejection altogether — discover/order mismatch is a CP-Rules §5 misleading-listing risk."
    citation: "Consumer Protection (E-Commerce) Rules 2020 r.5(3); CP Act 2019 §2(47)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-031
    surface_id: api-error-validation-payout-method
    lens: legal
    severity: P1
    issue: "Payout-method validation surfaces 'payoutMethod must be \"bank_transfer\" or \"upi\"' — internal enum values to a chef/driver. Also no tax-context disclaimer (chefs/drivers are responsible for income-tax declaration; KYC under PMLA 2002 applies). NO link to KYC requirements or income-tax obligations."
    evidence_excerpt: "payoutMethod must be 'bank_transfer' or 'upi' / bankAccountNumber, bankIFSC, and bankAccountName are required for bank_transfer / upiId is required for upi payout method"
    recommendation: "Rewrite in human terms: 'Choose Bank Transfer or UPI for payouts.' Add a footer link on the payout-method screen to KYC requirements (PMLA 2002 §12), TDS obligations under Income-Tax Act §194-O (e-commerce TDS @ 1%), and chef tax FAQ. Pure technical error is acceptable internally but must be paired with regulatory context on the screen."
    citation: "PMLA 2002 §12; Income-Tax Act 1961 §194-O; RBI PA MD 17 Mar 2020 §8 (settlement)"
    depends_on: "needs lawyer review"

  # ============================================================================
  # FSSAI / FOOD-SAFETY EXPOSURE IN ERRORS / EMPTY STATES
  # ============================================================================

  - finding_id: LEG-ERR-032
    surface_id: mv-onb-docs-required-alert
    lens: legal
    severity: P0
    issue: "Mobile-vendor onboarding error 'Please upload both ID proof and FSSAI license to continue.' is correct in requiring FSSAI but does not specify which FSSAI category (Registration vs State License vs Central License — turnover-tiered under FSS Act 2006 + Regulations 2011). Chefs may upload the wrong tier."
    evidence_excerpt: "Please upload both ID proof and FSSAI license to continue."
    recommendation: "Surface FSSAI category guidance inline: 'FSSAI Basic Registration (turnover < ₹12L), State License (₹12L-₹20Cr), or Central License (> ₹20Cr). Choose the licence that matches your expected turnover.' Link to FSSAI portal. Without this guidance the platform may admit chefs operating without correct FSSAI tier — FSS Act §63 (penal exposure)."
    citation: "Food Safety and Standards Act 2006 §31, §63; FSS (Licensing & Registration) Regulations 2011 r.2.1"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-033
    surface_id: vp-onb-docs-upload-failed
    lens: legal
    severity: P1
    issue: "FSSAI / KYC document-upload failure says only 'Upload failed' with no retry guidance, no support contact, no statement of data handling for partial uploads. Chef onboarding documents are SENSITIVE PERSONAL DATA under IT Rules 2011 and DPDP §3. Failed upload without retention disclosure is a DPDP §6(2)(d) gap."
    evidence_excerpt: "Upload failed"
    recommendation: "Expand: 'Upload failed. Try again, or contact support if it keeps failing. Failed uploads are not retained.' Link to Privacy Policy section on document retention. DPDP §6 notice + §8(3) retention."
    citation: "DPDP Act 2023 §6(2)(d), §8(3); IT Rules 2011 r.3 (sensitive personal data)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-034
    surface_id: mc-chef-empty-category
    lens: legal
    severity: P3
    issue: "Empty menu-category state 'No items in this category' is innocuous but flag as related-to FSSAI labelling: ensure menu-category names cannot be confused with regulated category claims (e.g., 'Organic', 'Diabetic-Safe', 'Sugar-Free') unless verified by chef — FSS Act §53 misleading advertising."
    evidence_excerpt: "No items in this category"
    recommendation: "Audit allowable category taxonomies for regulated nutrition claims. Add reviewer-side validation on FSS-regulated descriptors. Not a copy fix on this surface, but a category-taxonomy compliance flag."
    citation: "FSS Act 2006 §53; FSS (Advertising & Claims) Regulations 2018"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-035
    surface_id: vp-onb-banner-rejected-body
    lens: legal
    severity: P0
    issue: "Onboarding-rejection banner 'Your previous application was not approved. Please review the feedback below and re-submit.' does NOT specify (a) FSSAI/KYC reason categorisation, (b) appeal rights, (c) Grievance Officer contact, (d) retention period for rejected-application data. DPDP §8(3) retention + §10 Grievance + Consumer Protection (E-Commerce) Rules §4(5) transparent communication."
    evidence_excerpt: "Your previous application was not approved. Please review the feedback below and re-submit."
    recommendation: "Expand: 'Your application was not approved. See feedback below. You can re-submit once resolved. To appeal or ask questions, contact <Grievance Officer email>.' Also state retention of rejected-application data ('Your data will be deleted in 90 days if not re-submitted'). Cite DPDP §8(3), §10."
    citation: "DPDP Act 2023 §8(3), §10; Consumer Protection (E-Commerce) Rules 2020 r.4(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-036
    surface_id: vp-onb-banner-info-requested-body
    lens: legal
    severity: P1
    issue: "'The admin team needs additional information before approving your application.' — no specifics on what's needed, no SLA on response time, no Grievance Officer if the information request itself seems unjustified. DPDP §11 (right to information) is implicated when a Data Fiduciary asks for further personal data."
    evidence_excerpt: "The admin team needs additional information before approving your application."
    recommendation: "Itemise what's requested inline. State the purpose for which each additional item is being collected (DPDP §6(2)(a) notice obligation when personal data is sought). State expected approval SLA after submission."
    citation: "DPDP Act 2023 §6(2)(a), §11; Consumer Protection (E-Commerce) Rules 2020 r.4(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-037
    surface_id: vp-onb-locked-conflict-toast
    lens: legal
    severity: P2
    issue: "'Your kitchen is already under review or live. Returning to the dashboard.' offers no detail and no contact path if the user believes this is wrong. State-confusion at onboarding step is procedurally risky."
    evidence_excerpt: "Your kitchen is already under review or live. Returning to the dashboard."
    recommendation: "Add 'If this seems wrong, contact <Grievance Officer email>.' DPDP §10 Grievance reachability."
    citation: "DPDP Act 2023 §10"
    depends_on: "needs lawyer review"

  # ============================================================================
  # TERMS-AND-CONDITIONS ACKNOWLEDGEMENT ERRORS — contract enforceability
  # ============================================================================

  - finding_id: LEG-ERR-038
    surface_id: api-error-tos
    lens: legal
    severity: P0
    issue: "Driver onboarding error 'You must accept the terms and conditions' is correct in gating but DOES NOT LINK to the actual T&Cs in the error. Click-wrap acceptance for driver/gig worker is contract-formation; a bare error without a link is poor practice and weakens Indian Contract Act §10 enforceability. Inventory note explicitly flags this."
    evidence_excerpt: "You must accept the terms and conditions"
    recommendation: "Always pair acceptance-error with link: 'Open Terms & Conditions, Driver Agreement, and Privacy Policy. You must read and accept these to continue.' Cite Indian Contract Act §10, IT Act §10A (click-wrap enforceability), and DPDP §6 notice for the privacy-policy half."
    citation: "Indian Contract Act 1872 §10; IT Act 2000 §10A; DPDP Act 2023 §6"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-039
    surface_id: dp-err-step5-terms
    lens: legal
    severity: P0
    issue: "Driver web-portal error 'Please accept the terms and conditions' — same as LEG-ERR-038 — does not link to the binding documents. Driver contract-formation moment."
    evidence_excerpt: "Please accept the terms and conditions"
    recommendation: "Pair with links to (a) Platform Terms, (b) Driver Agreement (gig-worker classification), (c) Privacy Policy, (d) Payout/KYC terms. Indian Contract Act §10, IT Act §10A, Social Security Code 2020 §141."
    citation: "Indian Contract Act 1872 §10; IT Act 2000 §10A; Code on Social Security 2020 §141"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-040
    surface_id: mv-onb-policies-terms-required-alert
    lens: legal
    severity: P0
    issue: "Mobile-vendor 'Please accept the terms and conditions to continue.' — chef/vendor contract-formation. Same gap as drivers. Chef contract is more complex (food-safety responsibility split FSS Act §31, GST registration obligation, payout/TDS) yet error does not link to anything."
    evidence_excerpt: "Please accept the terms and conditions to continue."
    recommendation: "Pair with links to Chef Agreement (FSSAI compliance, GST obligations, payout terms), Privacy Policy, allergen-handling addendum. Critical for FSS Act §31 (chef as 'food business operator' responsible for safety) and Income-Tax Act §194-O (e-commerce TDS)."
    citation: "Indian Contract Act 1872 §10; IT Act 2000 §10A; FSS Act 2006 §31; Income-Tax Act 1961 §194-O"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-041
    surface_id: vp-onb-review-policies-pending
    lens: legal
    severity: P1
    issue: "'Policies not accepted / Please go back to Step 5 and accept all required policies.' — points to a step but does not enumerate which policies are pending. Procedural-fairness obligation under Consumer Protection (E-Commerce) Rules §4."
    evidence_excerpt: "Policies not accepted / Please go back to Step 5 and accept all required policies."
    recommendation: "Enumerate pending policies in the error itself with links. Each policy should also have a 'plain-language summary' callout per house style guide §5."
    citation: "Consumer Protection (E-Commerce) Rules 2020 r.4(5); Indian Contract Act 1872 §10"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-042
    surface_id: vp-onb-policies-required-toast
    lens: legal
    severity: P1
    issue: "'Please accept all required policies' — same as above without surface-specific detail. Toast offers zero context."
    evidence_excerpt: "Please accept all required policies"
    recommendation: "Replace toast with inline error listing the unchecked policies, linked. Acceptance-tracking is critical for downstream proof-of-consent under DPDP §6 and IT Act §10A."
    citation: "DPDP Act 2023 §6; IT Act 2000 §10A; Indian Contract Act 1872 §10"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-043
    surface_id: mv-onb-policies-policy-required-alert
    lens: legal
    severity: P1
    issue: "'Please select a cancellation policy.' — chef chooses their own cancellation policy. The policy choice surface must, by Consumer Protection (E-Commerce) Rules §5(3), describe the consequence of each option to the chef (refund obligations, dispute exposure). The error alone is procedural; the screen design needs lawyer review."
    evidence_excerpt: "Please select a cancellation policy."
    recommendation: "On the cancellation-policy selection screen, surface for each option: (i) refund SLA chef commits to, (ii) chef's dispute liability, (iii) customer-side messaging. Cite CP E-Commerce Rules §5(3). Error itself is fine; screen needs the disclosure."
    citation: "Consumer Protection (E-Commerce) Rules 2020 r.5(3); RBI PA MD 17 Mar 2020 §7"
    depends_on: "needs lawyer review"

  # ============================================================================
  # KYC / VERIFICATION ERRORS
  # ============================================================================

  - finding_id: LEG-ERR-044
    surface_id: dp-err-step1-referral-invalid
    lens: legal
    severity: P3
    issue: "'Invalid referral code' — minor; ensure that submitting an invalid code does NOT confirm whether referral programme is active for the user's region (could surface marketing-eligibility data)."
    evidence_excerpt: "Invalid referral code"
    recommendation: "Verify referral code validation does not leak referrer identity or region eligibility. DPDP §8(5) hygiene."
    citation: "DPDP Act 2023 §8(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-045
    surface_id: dp-err-step3-file-size
    lens: legal
    severity: P3
    issue: "'File too large. Max {n}MB for {label}' — fine but ensure consistent size representation across all driver upload surfaces (KYC, vehicle docs, profile). Inconsistent limits across portals may suggest different KYC tiers, confusing drivers and exposing platform to procedural-fairness claims."
    evidence_excerpt: "File too large. Max {n}MB for {label}"
    recommendation: "Standardise driver upload max sizes; surface explicitly per document type. Cite MV Aggregator Guidelines 2020 §4 on driver document handling."
    citation: "MV Aggregator Guidelines 2020 §4; DPDP Act 2023 §6(2)(d)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-046
    surface_id: api-error-upload-type
    lens: legal
    severity: P2
    issue: "5 variants of 'Invalid file type. Allowed: …' with different allowed lists (JPEG/PNG/PDF; JPEG/PNG/WebP/PDF; JPEG/PNG/WebP; etc.) and inconsistent punctuation. For KYC/onboarding uploads this inconsistency means a chef/driver may try a format that worked elsewhere and be rejected — procedural-fairness exposure under Consumer Protection (E-Commerce) Rules §4(5)."
    evidence_excerpt: "Invalid file type. Allowed: JPEG, PNG, PDF. / Invalid file type. Allowed: JPEG, PNG, WebP, PDF / Invalid file type. Allowed: JPEG, PNG, WebP. / Invalid image type. Allowed: JPEG, PNG, WebP. / Profile photo must be JPEG or PNG"
    recommendation: "Define one canonical allowed-format list per upload context (KYC docs, profile photo, menu image, review image, social post) and document. Standardise punctuation. Plain-language obligation."
    citation: "Consumer Protection (E-Commerce) Rules 2020 r.4(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-047
    surface_id: api-error-upload-size
    lens: legal
    severity: P2
    issue: "Inconsistent size formatting: '10MB' vs '5 MB' vs '5MB' vs '5 MB'. Mobile-vendor uses '5 MB' (with space) per locale spec, web uses '5MB' (no space). Style-guide §6 requires consistent unit formatting. For KYC/payout uploads this inconsistency reads as platform sloppiness."
    evidence_excerpt: "File too large (max 10MB) / File too large. Maximum 5 MB. / Profile photo too large (max 5MB) / Each image must be under 5 MB"
    recommendation: "Standardise to '5 MB' (en-IN spacing) everywhere. Plain-language consistency under CP E-Commerce Rules §4(5)."
    citation: "Consumer Protection (E-Commerce) Rules 2020 r.4(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-048
    surface_id: api-error-upload-count
    lens: legal
    severity: P3
    issue: "'Maximum 3 images per review' / 'Maximum 4 images per post' / 'Maximum 5 images per menu item' / 'Maximum 5 kitchen photos allowed. Remove one before adding another.' — only the last has friendly remediation. Procedural consistency."
    evidence_excerpt: "Maximum 3 images per review / Maximum 4 images per post / Maximum 5 images per menu item / Maximum 5 kitchen photos allowed. Remove one before adding another."
    recommendation: "All count limits must include the remediation ('Remove one before adding another'). Cite plain-language obligation."
    citation: "Consumer Protection (E-Commerce) Rules 2020 r.4(5)"
    depends_on: "needs lawyer review"

  # ============================================================================
  # ADMIN / DESTRUCTIVE ACTIONS — audit-trail and procedural fairness
  # ============================================================================

  - finding_id: LEG-ERR-049
    surface_id: ap-providerdetail-dialog-bodies
    lens: legal
    severity: P1
    issue: "Three branched confirmation messages on delivery-provider delete/disable/enable are vague generic 'Are you sure you want to…' without surfacing (a) impact on in-flight deliveries, (b) audit-trail commitment, (c) reversal pathway. DPDP §8(5) audit obligations; CP §5(3) procedural fairness for affected drivers/partners."
    evidence_excerpt: "Are you sure you want to delete this delivery provider? This action cannot be undone.; Are you sure you want to disable this provider? No new deliveries will be assigned to it.; Are you sure you want to enable this provider? It will start receiving delivery assignments."
    recommendation: "Each confirmation should state: (i) impact on in-flight items, (ii) audit-log entry that will be created, (iii) Grievance contact for affected partner if applicable. 'This action cannot be undone' is too vague (style-guide §4 modal-subtitle pattern)."
    citation: "DPDP Act 2023 §8(5); Consumer Protection (E-Commerce) Rules 2020 r.5(3)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-050
    surface_id: ap-providers-dialog-delete-body
    lens: legal
    severity: P1
    issue: "'Are you sure you want to delete this delivery provider? This action cannot be undone.' duplicated across surfaces. 'Cannot be undone' is regulatory exposure if the deletion also purges audit data — DPDP §8(7) and IT Rules 2021 require record retention for grievance lookups. Verify deletion is logical / preserves audit, not physical purge."
    evidence_excerpt: "Are you sure you want to delete this delivery provider? This action cannot be undone."
    recommendation: "Distinguish 'delete' (soft, audit-trail preserved) from 'purge' (hard, GDPR-style right-to-erasure flow under DPDP §13). The copy should match the engineering behaviour. Lawyer to confirm record-retention policy aligns with IT Rules 2021 r.3 and tax-record obligations under Income-Tax Act §44AA (8 years)."
    citation: "DPDP Act 2023 §8(3), §8(7), §13; IT Rules 2021 r.3; Income-Tax Act 1961 §44AA"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-051
    surface_id: ap-staffdetail-confirm-deactivate
    lens: legal
    severity: P1
    issue: "'Are you sure you want to deactivate this staff member? They will lose access to all portals.' — does not surface (a) audit trail of deactivation, (b) access-revocation timestamp, (c) data-retention for deactivated-staff's audit history. IT Rules 2021 r.3 require audit-trail retention for at least 180 days for OSPs / intermediaries."
    evidence_excerpt: "Are you sure you want to deactivate this staff member? They will lose access to all portals."
    recommendation: "Add: '<Staff name>'s actions are retained in the audit log for [X years] per regulatory requirements.' Cite IT Rules 2021 r.3 record-keeping."
    citation: "IT Rules 2021 r.3; DPDP Act 2023 §8(7)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-052
    surface_id: api-error-staff-permissions
    lens: legal
    severity: P2
    issue: "Staff-permission errors expose internal role hierarchy: 'Only super admins can change staff roles', 'Only super admins can invite other super admins', 'Cannot change role of a default super admin', 'Cannot deactivate a default super admin'. Internal role taxonomy leaks to anyone with admin-portal access. Limited risk (admins only) but documents RBAC structure."
    evidence_excerpt: "You don't have permission to invite staff / Only super admins can change staff roles / Only super admins can invite other super admins / Cannot change role of a default super admin / Cannot deactivate a default super admin / Cannot deactivate yourself"
    recommendation: "Acceptable for admin portal but ensure these are NEVER returned to non-admin sessions. Centralise role taxonomy in a single source of truth (also helps DPDP §8(5) reasonable security on RBAC enforcement)."
    citation: "DPDP Act 2023 §8(5); IT Act 2000 §43A"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-053
    surface_id: ap-users-toast-fail-suspend
    lens: legal
    severity: P2
    issue: "'Failed to suspend user' / 'Failed to activate user' — admin destructive actions failing silently with no audit-log reference. If admin retries and second attempt succeeds, audit trail may show duplicate or inconsistent state. DPDP §8(7) audit-log integrity."
    evidence_excerpt: "Failed to suspend user / Failed to activate user"
    recommendation: "Each admin destructive-action failure must capture an audit-log ID surfaced to the admin: 'Failed to suspend user. Audit ID: <X>. Check audit log before retrying.' Cite DPDP §8(7) and IT Rules 2021 r.3."
    citation: "DPDP Act 2023 §8(7); IT Rules 2021 r.3"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-054
    surface_id: ap-auditlogs-empty
    lens: legal
    severity: P2
    issue: "Audit-log empty state 'No audit events match these filters.' is fine. Verify that an empty audit-log table EVER displays this — would indicate no events captured (compliance gap). The empty state should never appear without a filter active."
    evidence_excerpt: "No audit events match these filters."
    recommendation: "Distinguish 'no audit events ever' (compliance failure — should never happen) from 'no events match filter' (normal). The former should never be silent. Cite IT Rules 2021 r.3."
    citation: "IT Rules 2021 r.3; DPDP Act 2023 §8(7)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-055
    surface_id: ap-analytics-coming-soon
    lens: legal
    severity: P1
    issue: "'Chart coming soon' is a placeholder shown to admin users in production. Surfacing unfinished work to operators looks unprofessional and risks an admin acting on stale/incomplete data because 'analytics' implies present-tense capability. Procedural-fairness concern if admin decisions rely on analytics."
    evidence_excerpt: "Chart coming soon"
    recommendation: "Either hide the surface entirely or replace with 'This analytics view is in development. Use Reports > X for current data.' Do not ship 'coming soon' to a paying tenant's admin."
    citation: "Consumer Protection Act 2019 §2(47) (misleading representation, B2B-adjacent); best-practice"
    depends_on: "needs lawyer review"

  # ============================================================================
  # ERROR BOUNDARIES — top-level fallbacks, log-leak risk
  # ============================================================================

  - finding_id: LEG-ERR-056
    surface_id: web-err-error-boundary
    lens: legal
    severity: P1
    issue: "Top-level error fallback 'Unexpected error / Something broke on our end. / We've been notified and we're looking into it... / Try again / Refresh page' makes the operational claim 'We've been notified'. If error reporting (Sentry/etc.) is misconfigured, the claim is FALSE — Consumer Protection §2(47) misleading representation. Also: does not surface a Grievance Officer contact for persistent failures (DPDP §10)."
    evidence_excerpt: "Unexpected error / Something broke on our end. / We've been notified and we're looking into it... / Try again / Refresh page"
    recommendation: "Verify error reporting is wired and the 'We've been notified' claim is factually true. Add Grievance Officer contact for persistent failures: 'Still broken? Contact <email>.' Cite DPDP §10 + CP §2(47)."
    citation: "DPDP Act 2023 §10; CP Act 2019 §2(47)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-057
    surface_id: vp-err-boundary-fallback
    lens: legal
    severity: P1
    issue: "Vendor error boundary 'Something went wrong / An unexpected error occurred while loading this page. / Try Again' — no Grievance contact, no error reference ID. Chefs facing a persistent error during a live cooking shift have no escalation path. CP E-Commerce Rules §5(3) procedural fairness."
    evidence_excerpt: "Something went wrong / An unexpected error occurred while loading this page. / Try Again"
    recommendation: "Surface a unique error ID and a Grievance contact when persistent. Critical for vendor portal because chef revenue depends on uptime."
    citation: "Consumer Protection (E-Commerce) Rules 2020 r.5(3); DPDP Act 2023 §10"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-058
    surface_id: dp-err-boundary-body
    lens: legal
    severity: P0
    issue: "Driver-portal error boundary 'Something went wrong / An unexpected error occurred while loading this page.' — fatal for a driver mid-delivery. No fallback for active-delivery state, no offline-cache reference, no support contact. MV Aggregator Guidelines 2020 require operational safety + grievance redress for gig workers."
    evidence_excerpt: "Something went wrong / An unexpected error occurred while loading this page."
    recommendation: "Error boundary must (a) check for active-delivery state and surface it, (b) provide a phone-number fallback for ops support, (c) instruct driver to NOT leave delivery state — call customer if needed. Cite MV Aggregator Guidelines 2020 §4 + Code on Social Security §141."
    citation: "MV Aggregator Guidelines 2020 §4; Code on Social Security 2020 §141; Consumer Protection (E-Commerce) Rules 2020 r.5(3)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-059
    surface_id: ap-errorboundary-body
    lens: legal
    severity: P1
    issue: "Admin error boundary same generic pattern. Admin actions are evidentiary — an error during a destructive action without an audit reference is an audit-trail risk."
    evidence_excerpt: "Something went wrong / An unexpected error occurred while loading this page."
    recommendation: "Admin error boundary must include action context and audit-log reference if action was attempted. DPDP §8(7) audit-log integrity."
    citation: "DPDP Act 2023 §8(7); IT Rules 2021 r.3"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-060
    surface_id: web-err-generic-retry
    lens: legal
    severity: P2
    issue: "'Something went wrong. Please try again.' duplicated across 5+ surfaces (HomePage, BrowseChefsPage, ChefDetailPage, LoginPage, vp variants, ap variants, dp variants). Generic 'try again' may be MISLEADING if the failure is permanent (e.g., chef permanently deactivated, deleted product). Consumer Protection §2(47) misleading representation; CP E-Commerce Rules §4(5) transparent communication."
    evidence_excerpt: "Something went wrong. Please try again."
    recommendation: "Distinguish transient errors (retry works) from permanent ones (don't retry). For permanent errors surface specific remediation. Never advise 'try again' on a permanent failure — exposes platform to misleading-trade claims for repeated futile attempts."
    citation: "CP Act 2019 §2(47); Consumer Protection (E-Commerce) Rules 2020 r.4(5)"
    depends_on: "needs lawyer review"

  # ============================================================================
  # CHAT / MODERATION ERRORS — IT Rules 2021 + content moderation
  # ============================================================================

  - finding_id: LEG-ERR-061
    surface_id: api-error-chat-availability
    lens: legal
    severity: P1
    issue: "Chat-availability errors include: 'Chat is not available for completed/cancelled orders', 'This chat room is closed', 'File uploads are not allowed in chat'. For an order in dispute the customer/chef may need chat history. IT Rules 2021 r.3 require intermediaries to retain message data for 180 days for legal lookup. The error should clarify that history is retained / accessible to support."
    evidence_excerpt: "Chat is not available for completed/cancelled orders / This chat room is closed / This chat room is closed because the order is no longer active / File uploads are not allowed in chat / Please wait a moment before sending another message / Message must be 500 characters or fewer / You are not a participant in this order"
    recommendation: "When closing a chat add: 'This chat is closed. Your message history is retained for 180 days for disputes. Contact <Grievance Officer> if you need to follow up.' Cite IT Rules 2021 r.3, DPDP §10."
    citation: "IT Rules 2021 r.3; DPDP Act 2023 §10"
    depends_on: "needs lawyer review"

  # ============================================================================
  # CATERING / QUOTES — RBI + Consumer Protection
  # ============================================================================

  - finding_id: LEG-ERR-062
    surface_id: api-error-quotes-catering
    lens: legal
    severity: P1
    issue: "Catering-quote errors include 'This quote has expired', 'Quote deadline has passed', 'This quote cannot be accepted' — no SLA explanation, no refund implication when a quote expires after payment has been initiated, no Grievance contact."
    evidence_excerpt: "This request is no longer accepting quotes / You have already submitted a quote for this request / Quote deadline has passed / This quote cannot be accepted / This quote has expired / Event date must be in the future"
    recommendation: "Quotes are pre-contract offers under Indian Contract Act §2. Errors must clarify: (i) whether any deposit is auto-refunded, (ii) refund SLA, (iii) alternative quote pathway. Cite Indian Contract Act §6 (revocation of offer) and RBI PA MD if payment was tied."
    citation: "Indian Contract Act 1872 §2, §6; RBI PA MD 17 Mar 2020 §7"
    depends_on: "needs lawyer review"

  # ============================================================================
  # EMPTY-STATE CLAIMS — implicit regulatory / quality claims
  # ============================================================================

  - finding_id: LEG-ERR-063
    surface_id: ap-platsettings-zones-empty
    lens: legal
    severity: P2
    issue: "Platform-settings empty state 'No zones yet — delivery is available everywhere until a zone is created.' makes a service-area claim 'available everywhere' that may be false in practice (carrier limitations, FSSAI catchment, MV-permit limits). Misleading representation under CP §2(47) if a customer order is refused after this state."
    evidence_excerpt: "No zones yet — delivery is available everywhere until a zone is created."
    recommendation: "Replace with: 'No delivery zones set. Configure your serviceable zones before going live.' Avoid claims about geographic coverage that may not survive operational reality."
    citation: "CP Act 2019 §2(47); MV Act 1988 (permit limits)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-064
    surface_id: dp-empty-active-body
    lens: legal
    severity: P3
    issue: "'Check available deliveries to pick up a new order.' — neutral. Verify the driver-availability terminology does not conflict with the driver's contractual relationship (independent contractor vs employee classification under Code on Social Security 2020 §2(35) gig-worker definition)."
    evidence_excerpt: "Check available deliveries to pick up a new order."
    recommendation: "Audit driver-app copy globally for language that implies employer-employee relationship vs gig-worker classification. 'Available deliveries' is fine; 'assigned shifts', 'roster' would not be."
    citation: "Code on Social Security 2020 §2(35), §141"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-065
    surface_id: md-emp-007
    lens: legal
    severity: P3
    issue: "'Fleet management is available for fleet managers only. Contact your administrator to request access.' — surfaces internal role taxonomy ('fleet managers', 'administrator') and an access-request pathway. Fine for B2B but ensure no PII leakage if 'administrator' refers to a specific contact."
    evidence_excerpt: "Fleet management is available for fleet managers only. Contact your administrator to request access."
    recommendation: "Verify 'administrator' is generic, not a named contact. DPDP §8(5) hygiene."
    citation: "DPDP Act 2023 §8(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-066
    surface_id: md-emp-011
    lens: legal
    severity: P3
    issue: "'Staff management requires manager permissions. Contact your administrator to request access.' — same as above."
    evidence_excerpt: "Staff management requires manager permissions. Contact your administrator to request access."
    recommendation: "Same as LEG-ERR-065. DPDP §8(5)."
    citation: "DPDP Act 2023 §8(5)"
    depends_on: "needs lawyer review"

  # ============================================================================
  # FAVORITE LIMIT — soft-cap as implicit policy
  # ============================================================================

  - finding_id: LEG-ERR-067
    surface_id: web-err-favorite-max-limit
    lens: legal
    severity: P3
    issue: "'You can save up to 7 favorite chefs. Remove one first.' — duplicated across BrowseChefsPage, ChefDetailPage, HomePage. Limit of 7 is arbitrary; if the limit ever rises, all copies must update. Best practice: centralise via token/config and reference."
    evidence_excerpt: "You can save up to 7 favorite chefs. Remove one first."
    recommendation: "Centralise the constant. Not a legal issue per se but inconsistent limits across surfaces become procedural-fairness signals if not maintained."
    citation: "Consumer Protection (E-Commerce) Rules 2020 r.4(5)"
    depends_on: "needs lawyer review"

  # ============================================================================
  # MISSING ITEMS — empty states that should never be empty
  # ============================================================================

  - finding_id: LEG-ERR-068
    surface_id: ap-staffdetail-perms-empty
    lens: legal
    severity: P2
    issue: "'No specific permissions data available for this role.' — admin sees an empty permissions list. If this ever appears in production, it suggests an RBAC configuration gap. Admin acting on a role with no documented permissions is procedural exposure. DPDP §8(5) requires documented access controls."
    evidence_excerpt: "No specific permissions data available for this role."
    recommendation: "If permissions are missing, this is a system-config error not a normal empty state. Replace empty state with an error surfacing a config issue + admin escalation. DPDP §8(5)."
    citation: "DPDP Act 2023 §8(5); IT Rules 2021 r.3"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-069
    surface_id: ap-secsettings-apikey-empty
    lens: legal
    severity: P3
    issue: "'No keys yet.' for API-key management. Verify that an admin creating their first API key sees a one-time disclosure of (a) key handling responsibility, (b) revocation pathway, (c) audit-log retention. Currently no copy at the empty state on these obligations."
    evidence_excerpt: "No keys yet."
    recommendation: "On empty state, add a small note: 'API keys grant programmatic access. Created keys are shown once. Revoke immediately if leaked.' DPDP §8(5) reasonable-security framing."
    citation: "DPDP Act 2023 §8(5); IT Act 2000 §43A"
    depends_on: "needs lawyer review"

  # ============================================================================
  # CART EMPTY — pre-contract data handling
  # ============================================================================

  - finding_id: LEG-ERR-070
    surface_id: web-ux-cart-empty
    lens: legal
    severity: P3
    issue: "Empty cart copy 'Your cart is empty / Looks like you haven't added any items yet. Browse our home chefs...' is fine. Flag for DPDP §6(2)(d): the cart, even when empty, may carry session/tracking cookies. The cart page is a sensible place to surface a small cookie-policy link."
    evidence_excerpt: "Your cart is empty / Looks like you haven't added any items yet. Browse our home chefs..."
    recommendation: "Not a copy fix for the empty state itself. But verify a cookie/consent banner is visible globally per DPDP §6(2)(d) notice requirements."
    citation: "DPDP Act 2023 §6(2)(d)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-071
    surface_id: web-err-cart-minimum
    lens: legal
    severity: P1
    issue: "'Minimum order not met / Add {amount} more to proceed' — minimum-order rule is a chef-set policy. Customer should see WHO set the minimum (platform vs chef) and reason — particularly for FSSAI catering compliance or chef-side prep economics. Procedural-fairness disclosure."
    evidence_excerpt: "Minimum order not met / Add {amount} more to proceed"
    recommendation: "Clarify: 'This chef requires a minimum order of ₹X. Add ₹Y more to proceed.' Identifying the minimum as chef-set vs platform-set is important for refund/dispute pathways."
    citation: "Consumer Protection (E-Commerce) Rules 2020 r.4(5), r.5"
    depends_on: "needs lawyer review"

  # ============================================================================
  # MOBILE-CUSTOMER SPECIFIC
  # ============================================================================

  - finding_id: LEG-ERR-072
    surface_id: mc-onb-step1-errors
    lens: legal
    severity: P2
    issue: "Mobile-customer onboarding validation 'Enter a valid 10-digit Indian mobile number' is India-specific. Acceptable for India launch but make sure the app refuses sign-ups from outside India (DPDP §3 applicability, RBI KYC, FEMA cross-border data). The validation alone is not an enforcement — back-end must also gate."
    evidence_excerpt: "First name must be at least 2 characters / Last name must be at least 2 characters / Enter a valid 10-digit Indian mobile number"
    recommendation: "Confirm back-end mobile validation and country-of-residence checks. If platform serves only India, surface that on signup. Cite DPDP §3 territorial scope."
    citation: "DPDP Act 2023 §3; RBI KYC Master Direction 2016; FEMA 1999"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-073
    surface_id: mv-onb-personal-phone-err
    lens: legal
    severity: P2
    issue: "Same India-specific mobile pattern enforced on chef onboarding. Chef registration is consumer-identifying AND tax-identifying (TDS under §194-O requires PAN). The mobile validation alone may admit non-resident chefs."
    evidence_excerpt: "Enter a valid 10-digit Indian mobile number"
    recommendation: "Pair mobile validation with PAN capture step for chefs (Income-Tax Act §194-O e-commerce TDS @ 1% requires payee PAN; otherwise TDS @ 5%)."
    citation: "Income-Tax Act 1961 §194-O, §139A; PMLA 2002 §12"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-074
    surface_id: mc-catering-errors
    lens: legal
    severity: P2
    issue: "Catering form validation '… Event date must be in the future / Guest count must be a number / Budget must be a number / Budget must be positive / City is required / State is required' — for large catering events FSSAI bulk-catering rules and GST registration thresholds may apply. The form does not capture or warn about these."
    evidence_excerpt: "Event type is required / Date must be YYYY-MM-DD / Event date must be in the future / Guest count must be a number / At least 1 guest required / Budget must be a number / Budget must be positive / City is required / State is required"
    recommendation: "Above guest count / budget thresholds, surface a one-line FSSAI / GST-applicability note linking to a help page. Cite FSS Act §31 and CGST §22 (threshold for GST registration ₹20L general / ₹10L special category states)."
    citation: "FSS Act 2006 §31; CGST Act 2017 §22"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-075
    surface_id: mc-favorites-empty
    lens: legal
    severity: P3
    issue: "'No saved chefs yet / Tap the heart on any chef to save them!' — innocuous but verify favourites data is covered in Privacy Policy as collected personal data (DPDP §3 inclusion of preference/behavioural data)."
    evidence_excerpt: "No saved chefs yet / Tap the heart on any chef to save them!"
    recommendation: "Privacy Policy must enumerate 'favourites / saved chefs' as collected data. Not a copy fix on this surface; policy gap."
    citation: "DPDP Act 2023 §3, §6(2)"
    depends_on: "needs lawyer review"

  # ============================================================================
  # MENU-FORM VALIDATION — implicit pricing / FSSAI claims
  # ============================================================================

  - finding_id: LEG-ERR-076
    surface_id: vp-ux-menu-form-validation
    lens: legal
    severity: P2
    issue: "Menu-item form validation includes 'Select at least one dietary tag' — dietary-tag taxonomy is FSSAI-relevant (allergens, vegetarian/non-vegetarian, eggetarian, jain). FSS (Packaging and Labelling) Regulations 2011 r.2.4 require accurate veg/non-veg labelling. Validation must enforce mutual-exclusion (a dish cannot be both 'veg' and 'non-veg'). Audit the tag-taxonomy backend for compliance, not the error copy."
    evidence_excerpt: "Select at least one dietary tag (among 14 validation messages)"
    recommendation: "Verify dietary-tag taxonomy is FSSAI-compliant. Surface allergen-disclosure CTA at the dish form. Cite FSS (Packaging & Labelling) Regulations 2011 r.2.4.5 (veg/non-veg mandatory)."
    citation: "FSS (Packaging & Labelling) Regulations 2011 r.2.4; FSS (Advertising & Claims) Regulations 2018"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-077
    surface_id: mv-menunew-price-range
    lens: legal
    severity: P3
    issue: "'Price must be between ₹1 and ₹10,000' — flat range no tax-context. Confirm whether displayed price is inclusive of GST (chefs above ₹20L turnover must register). Style-guide §3 'Tax' wording applies."
    evidence_excerpt: "Price must be between ₹1 and ₹10,000"
    recommendation: "Pair price field with explicit inclusive/exclusive of tax toggle and GSTIN capture (when applicable). Cite CGST §22 + §31 (tax invoice)."
    citation: "CGST Act 2017 §22, §31; CGST Rules 2017 r.46"
    depends_on: "needs lawyer review"

  # ============================================================================
  # CONFIRMATION DIALOG WORDING — modal-subtitle compliance
  # ============================================================================

  - finding_id: LEG-ERR-078
    surface_id: mv-onb-docs-required-title
    lens: legal
    severity: P3
    issue: "'Documents Required' as an alert title — combined with 'Please upload both ID proof and FSSAI license to continue.' body. Surfaces FSSAI dependency clearly which is good. Title-case vs sentence-case inconsistency throughout mobile-vendor alerts."
    evidence_excerpt: "Documents Required / Permission Required / Terms Required / Policy Required / Validation Error / Submission Error"
    recommendation: "Standardise to sentence-case ('Documents required', etc.) per style guide §4 modal-subtitle pattern. Plain-language obligation under CP §4(5)."
    citation: "Consumer Protection (E-Commerce) Rules 2020 r.4(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-079
    surface_id: mv-more-logout-confirm-body
    lens: legal
    severity: P3
    issue: "'Are you sure you want to log out?' — uses 'log out' (style-guide bans this, requires 'sign out'). Voice consistency. Also: at logout, DPDP §6(2)(d) suggests surfacing what local-cache data persists vs purges."
    evidence_excerpt: "Are you sure you want to log out?"
    recommendation: "Rewrite per style guide §3: 'Sign out?' Style-guide §3 bans 'log out'. DPDP §6 notice for the logout-data-handling note."
    citation: "DPDP Act 2023 §6(2)(d); CP (E-Commerce) Rules 2020 r.4(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-080
    surface_id: mv-menuitem-delete-alert
    lens: legal
    severity: P3
    issue: "'Are you sure you want to delete this menu item?' — does not state whether deleting a menu item with active customer orders cancels those orders, refunds them, or what happens. Procedural fairness for in-flight customers."
    evidence_excerpt: "Are you sure you want to delete this menu item?"
    recommendation: "Surface: 'Deleting this item won't affect orders already placed. The item will be unavailable for new orders.' If actual behaviour differs, fix the message. Consumer Protection §2(47) misleading-representation risk."
    citation: "Consumer Protection (E-Commerce) Rules 2020 r.4(5), r.5"
    depends_on: "needs lawyer review"

  # ============================================================================
  # FINAL CROSS-CUTTING
  # ============================================================================

  - finding_id: LEG-ERR-081
    surface_id: api-error-validation-numeric
    lens: legal
    severity: P3
    issue: "Numeric-bounds errors 'overallRating must be between 1 and 5', 'Percentage discount must be between 0 and 100', 'rate must be between 0 and 100' — 'rate' is ambiguous (tax rate? commission rate? interest rate?). 'rate must be between 0 and 100' surfaces an internal field name."
    evidence_excerpt: "overallRating must be between 1 and 5 / Percentage discount must be between 0 and 100 / rate must be between 0 and 100"
    recommendation: "Disambiguate 'rate' in context. Plain-language obligation."
    citation: "Consumer Protection (E-Commerce) Rules 2020 r.4(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-082
    surface_id: api-error-validation-not-found
    lens: legal
    severity: P3
    issue: "~42 'X not found' variants. Mostly safe but flag 'Verification token not found' / 'Invitation token not found' as potential enumeration vectors (attacker can probe valid-vs-invalid token states). Also 'Chef not found or not available' compounds existence + availability — could leak chef-state to an unauthenticated probe."
    evidence_excerpt: "Verification token not found / Invitation token not found / Chef not found or not available / ~40 other variants"
    recommendation: "Token-not-found should be indistinguishable from token-expired (one canonical message). Chef-not-available should be distinct from chef-not-found only when the requester is authenticated and authorised to know. Cite DPDP §8(5)."
    citation: "DPDP Act 2023 §8(5); IT Act 2000 §43A"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-083
    surface_id: api-error-auth-unauthorized
    lens: legal
    severity: P2
    issue: "Four variants 'Unauthorized', 'unauthorized', 'Authentication required', 'Token required' — inconsistent casing AND inconsistent terminology. From a user's perspective these all mean 'sign in', but the inconsistency reads as platform sloppiness AND may make automated grievance-handling harder (Grievance Officer needs to map error code to user-facing complaint)."
    evidence_excerpt: "Unauthorized / unauthorized / Authentication required / Token required"
    recommendation: "Standardise to one user-facing string ('Please sign in to continue.') with one internal code. Plain-language obligation; DPDP §10 Grievance Officer effectiveness."
    citation: "Consumer Protection (E-Commerce) Rules 2020 r.4(5); DPDP Act 2023 §10"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-084
    surface_id: api-error-chef-onboarding
    lens: legal
    severity: P2
    issue: "Chef-onboarding pre-submit checks include 'Your application has already been approved' / 'Your application is already under review'. State-leak through error message — an attacker with stolen credentials learns the account's onboarding state. Limited risk (post-auth) but document. Also note the inventory mislabels this as driver_onboarding.go but it gates chef flow — verify."
    evidence_excerpt: "Please complete all previous onboarding steps before submitting / Please complete earlier onboarding steps first / Please complete step 1 (personal details) first / Please upload at least one document before submitting / Please indicate if your bicycle can carry a delivery box / Your application has already been approved / Your application is already under review"
    recommendation: "Standardise progression-blocking errors. Approval-state errors are post-auth and acceptable, but flag inconsistency ('complete all previous' vs 'complete earlier'). The 'bicycle delivery box' check is driver-specific — clarify which onboarding (chef or driver) the messages belong to. Plain-language."
    citation: "Consumer Protection (E-Commerce) Rules 2020 r.4(5); DPDP Act 2023 §8(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-085
    surface_id: api-error-delivery-online
    lens: legal
    severity: P1
    issue: "Driver-state errors mix 'You must be online to accept deliveries' (driver-facing 'you') with 'Partner has reached maximum concurrent deliveries' and 'Partner is not verified or active' (using 'Partner' terminology). Inconsistent point-of-view AND surfaces internal taxonomy. 'Partner is not verified' is also state-leak through error."
    evidence_excerpt: "You must be online to accept deliveries / Order not available for delivery / Partner has reached maximum concurrent deliveries / Partner is not verified or active"
    recommendation: "Use one POV ('you') in driver-facing surfaces. 'Partner is not verified or active' should never reach a verified driver — if it does, that's a state-machine bug. Cite MV Aggregator Guidelines 2020 §4 and procedural fairness for gig workers."
    citation: "MV Aggregator Guidelines 2020 §4; Code on Social Security 2020 §141; CP (E-Commerce) Rules 2020 r.4(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-086
    surface_id: api-error-order-review-rules
    lens: legal
    severity: P3
    issue: "'Can only review delivered orders' / 'This order has already been reviewed' — fine, but verify the review-system terms (one-review-per-order, edit-window, defamation pathway) are surfaced when a customer first reviews. IT Rules 2021 r.4(2) require intermediaries to enable grievance for unlawful content within 24 hours."
    evidence_excerpt: "Can only review delivered orders / This order has already been reviewed"
    recommendation: "Review-submission flow should surface defamation-policy and report-mechanism. Errors themselves are fine. IT Rules 2021 r.4(2)."
    citation: "IT Rules 2021 r.4(2); DPDP Act 2023 §10"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-087
    surface_id: ap-exports-error
    lens: legal
    severity: P1
    issue: "Admin data-export error 'Download failed' is minimal. Data exports include PII subject to DPDP §11 (right to access). A failed export of a DPDP §11 request must (a) be retried automatically, (b) be logged for audit, (c) surface a Grievance contact if persistent. 'Download failed' alone is insufficient."
    evidence_excerpt: "Download failed"
    recommendation: "Expand: 'Download failed. The request has been re-queued (ID: <X>). Contact <Grievance Officer> if it doesn't complete within 30 days.' DPDP §11 right-of-access timelines."
    citation: "DPDP Act 2023 §11, §10"
    depends_on: "needs lawyer review"

  - finding_id: LEG-ERR-088
    surface_id: api-error-invitation
    lens: legal
    severity: P2
    issue: "'Invitation has expired' / 'Invalid or expired invitation' / 'Invalid invitation token' / 'A pending invitation already exists for this email' — the last one CONFIRMS email existence in the staff/partner system. Privacy enumeration vector. Same pattern as registration phone-duplicate (LEG-ERR-008)."
    evidence_excerpt: "Invitation has expired / Invalid or expired invitation / Invalid invitation token / A pending invitation already exists for this email"
    recommendation: "Replace the 'pending invitation exists' message with a generic 'Invitation sent. If you don't receive an email, contact your admin.' Do not confirm email existence to the requester. DPDP §8(5)."
    citation: "DPDP Act 2023 §8(5); IT Act 2000 §43A"
    depends_on: "needs lawyer review"

  # ============================================================================
  # Closing note
  # ============================================================================
  # Total surfaces audited: 282 ERRORS-EMPTY rows
  # Total findings: 88
  # All findings carry depends_on: "needs lawyer review"
  # No binding text drafted — this is a gap analysis only

```

## Business Analyst findings

```yaml
# Business Analyst findings — errors-empty section
# Lens: conversion, activation, recovery, trust
# Auditor: BA lens agent
# Date: 2026-05-13
# Total inventory rows audited: 282

findings:

  # ─── P0: BROKEN RECOVERY PATHS ──────────────────────────────────────────────

  - finding_id: BA-ERR-001
    surface_id: ap-analytics-coming-soon
    lens: business-analyst
    severity: P0
    issue: "\"Chart coming soon\" placeholder ships to admin operators — fake/unfinished dashboard signal"
    evidence_excerpt: "Chart coming soon"
    recommendation: >
      Either deliver a real chart or hide the section entirely until ready.
      Shipping a visible placeholder to paying operators signals an unfinished product and erodes platform credibility.
      If data is not yet available, show an honest empty state: "Analytics data will appear after your first orders are processed."
    metric_hypothesis: "trust score; admin churn — operators who see 'coming soon' placeholders in a paid tool question the platform's readiness and may abandon before merchant activation"
    depends_on: null

  - finding_id: BA-ERR-002
    surface_id: api-error-payment-config
    lens: business-analyst
    severity: P0
    issue: "Internal payment gateway config error leaked verbatim to end customer — kills checkout conversion"
    evidence_excerpt: "Payment gateway not configured / Stripe gateway not configured / Stripe gateway not configured by platform admin"
    recommendation: >
      Replace all payment-config error messages surfaced to customers with a single generic message:
      "Payment is temporarily unavailable. Try again in a few minutes or contact support."
      The underlying cause (misconfigured gateway) must never reach the customer UI — it signals platform unreadiness and will cause immediate cart abandonment.
      Route the specific config error to admin dashboard alerts instead.
    metric_hypothesis: "cart-to-order conversion; customers who see a backend config message interpret it as a platform fault and do not retry — lost GMV per incident"
    depends_on: null

  - finding_id: BA-ERR-003
    surface_id: api-error-stripe-connect
    lens: business-analyst
    severity: P0
    issue: "Raw API paths exposed in error messages to chef users — signals broken product"
    evidence_excerpt: "No Stripe account — call /chef/stripe/connect first / No Stripe account — call /delivery/stripe/connect first"
    recommendation: >
      Replace with actionable human copy: "Complete your Stripe payout setup before switching payment methods. Go to Earnings > Payout setup."
      API endpoint paths must never appear in user-facing strings — they signal an incomplete integration and destroy trust during the chef activation flow.
    metric_hypothesis: "chef D7 retention; chefs who encounter raw API paths during payout setup interpret the platform as broken and drop out of the activation funnel"
    depends_on: null

  - finding_id: BA-ERR-004
    surface_id: web-err-checkout-cancel
    lens: business-analyst
    severity: P0
    issue: "Payment cancellation produces only \"Payment cancelled\" with no next-step CTA — stranded customer"
    evidence_excerpt: "Payment cancelled"
    recommendation: >
      Replace with a recovery flow: "Payment not completed. Your cart is saved — try again or use a different payment method."
      Add two CTAs: "Try again" (primary) and "Change payment method" (secondary).
      A cancelled Razorpay session is the highest-value recovery moment in the checkout funnel — a bare two-word toast wastes it entirely.
    metric_hypothesis: "cart-to-order conversion; customers who cancel Razorpay and see only a toast with no recovery CTA have no path back to checkout — estimated 30–50% drop on this state"
    depends_on: null

  - finding_id: BA-ERR-005
    surface_id: mc-checkout-errors
    lens: business-analyst
    severity: P0
    issue: "Payment confirmation timeout on mobile presents only a vague message with no clear status resolution path"
    evidence_excerpt: "Payment confirmation timed out. Check your order history to confirm status."
    recommendation: >
      Replace with: "Payment status unclear. Check your orders — if no order appears within 5 minutes, the payment was not charged. [View orders]"
      The current copy leaves the customer uncertain whether money was debited. This is the highest-anxiety state in mobile checkout.
      Add a deep-link CTA directly to the orders tab, not generic instruction.
    metric_hypothesis: "cart-to-order conversion; payment anxiety on mobile; customers who are unsure if they were charged will contact support or dispute charges — both outcomes damage platform economics"
    depends_on: null

  - finding_id: BA-ERR-006
    surface_id: api-error-auth-suspended
    lens: business-analyst
    severity: P0
    issue: "\"Account is suspended\" with no reason or support path — dead end for user"
    evidence_excerpt: "Account is suspended"
    recommendation: >
      Replace with: "Your account has been suspended. Contact support to find out why and how to resolve it. [Contact support]"
      A suspended user who hits a wall with no path forward will post publicly about the experience.
      Include a support link or email. Separately, if the suspension reason is shareable, surface it.
    metric_hypothesis: "trust score; support ticket volume; suspended accounts with no recovery path generate negative reviews and support escalations"
    depends_on: null

  # ─── P1: HIGH-TRAFFIC CONVERSION-CRITICAL ISSUES ────────────────────────────

  - finding_id: BA-ERR-007
    surface_id: web-ux-cart-empty
    lens: business-analyst
    severity: P1
    issue: "Empty cart copy is passive and does not name a specific chef discovery path"
    evidence_excerpt: "Your cart is empty / Looks like you haven't added any items yet. Browse our home chefs..."
    recommendation: >
      Replace with: "Your cart is empty. Browse home chefs near you." with a single CTA "Find a chef" linking directly to the browsing page.
      "Browse our home chefs..." is trailing copy — unclear where to go. Cut the filler phrase "Looks like you haven't added any items yet."
      The style guide empty-state formula is: Why it's empty → One action. The current copy adds a redundant explanation sentence.
    metric_hypothesis: "cart-to-order conversion; empty cart recovery rate — a direct CTA converts better than descriptive text"
    depends_on: null

  - finding_id: BA-ERR-008
    surface_id: web-err-favorites-empty
    lens: business-analyst
    severity: P1
    issue: "Empty favorites state offers no prompt to discover or save a chef — missed activation nudge"
    evidence_excerpt: "No favorites yet"
    recommendation: >
      Replace with: "No saved chefs. Browse home chefs and tap the heart to save your favourites." with CTA "Browse chefs".
      Three-word empty states with no action convert at near-zero. This is the first post-signup engagement hook for new customers.
    metric_hypothesis: "customer D7 retention; favorite-save rate as an activation proxy — customers who save at least one chef have significantly higher repeat order rates"
    depends_on: null

  - finding_id: BA-ERR-009
    surface_id: mc-favorites-empty
    lens: business-analyst
    severity: P1
    issue: "Mobile empty favorites uses a trailing exclamation mark and emoji — violates voice and adds friction language"
    evidence_excerpt: "No saved chefs yet / Tap the heart on any chef to save them!"
    recommendation: >
      Replace with: "No saved chefs. Tap the heart on any chef to save them." Remove the exclamation mark (Rule 1: no urgency tricks).
      The emoji is also a style-guide violation for body/utility copy. The functional guidance is good — just strip the punctuation energy.
    metric_hypothesis: "brand trust; customer activation rate — consistent, calm copy builds trust with the premium-positioning customer"
    depends_on: null

  - finding_id: BA-ERR-010
    surface_id: web-err-orders-empty
    lens: business-analyst
    severity: P1
    issue: "Empty orders state for a filtered view says \"No {filter} orders found\" with no suggestion to clear the filter"
    evidence_excerpt: "No orders found / You haven't placed any orders yet. / No {filter} orders found."
    recommendation: >
      For the filtered variant: "No {filter} orders. Try a different filter or browse all orders." with CTA "View all orders".
      For first-time empty state: "No orders yet. Browse home chefs to place your first order." with CTA "Find a chef".
      The filter-stuck variant is a conversion trap — customers assume no orders exist and leave rather than clearing the filter.
    metric_hypothesis: "repeat order rate; filter abandonment — customers who see a filtered empty state without a clear-filter CTA have a high exit rate from the orders page"
    depends_on: null

  - finding_id: BA-ERR-011
    surface_id: mc-orders-empty
    lens: business-analyst
    severity: P1
    issue: "Mobile orders empty state uses a plate emoji in functional copy — violates style guide"
    evidence_excerpt: "No orders yet / Browse chefs to place your first order!"
    recommendation: >
      Replace with: "No orders yet. Browse home chefs to place your first order." Remove the exclamation mark and the 🍽️ emoji.
      The guidance in the style guide is explicit: no exclamation marks except in genuine celebration. An empty state is not a celebration.
    metric_hypothesis: "brand consistency; new customer activation — voice inconsistency between web and mobile creates a fragmented brand impression"
    depends_on: null

  - finding_id: BA-ERR-012
    surface_id: vp-ux-menu-empty
    lens: business-analyst
    severity: P1
    issue: "New chef empty menu state has a CTA \"Add Your First Item\" but no walkthrough guidance — activation drop-off risk"
    evidence_excerpt: "No menu items yet / Start building your menu by adding your first dish. / Add Your First Item"
    recommendation: >
      The CTA is present and action-oriented — this is the right direction. Strengthen by adding a one-line hint about what happens after:
      "No menu items yet. Add your first dish and publish your menu to start accepting orders." CTA: "Add first dish".
      Chefs who do not add at least one menu item during the first session have near-zero activation. The current copy does not explain the consequence of not acting.
    metric_hypothesis: "chef D1 activation; chef menu-publish rate — chefs need to understand that adding a dish is the prerequisite to going live"
    depends_on: null

  - finding_id: BA-ERR-013
    surface_id: mv-menu-empty
    lens: business-analyst
    severity: P1
    issue: "Mobile vendor empty menu state has no consequence framing — chef does not know why adding a dish matters"
    evidence_excerpt: "No menu items yet. Tap + to add your first item."
    recommendation: >
      Replace with: "No menu items yet. Add a dish to start accepting orders." Remove the gesture hint ("Tap +") — the native affordance is visible.
      Chefs need to know that the menu is the blocker between them and their first order. The current copy sounds like a neutral instruction.
    metric_hypothesis: "chef D1 activation; menu-publish conversion — framing the consequence increases urgency without violating the no-urgency-tricks rule"
    depends_on: null

  - finding_id: BA-ERR-014
    surface_id: vp-ux-payouts-empty
    lens: business-analyst
    severity: P1
    issue: "Empty payouts state gives no guidance on when the first payout should be expected"
    evidence_excerpt: "No payouts yet / Your payout history will appear here once you receive your first payout."
    recommendation: >
      Replace with: "No payouts yet. Payouts are processed every [X] days after order completion. Your first payout will appear here once processed."
      Without a timeline, chefs who have completed orders will contact support thinking payouts are broken.
      If no SLA is defined, use: "Payouts are processed weekly. Your first will appear once an order completes."
    metric_hypothesis: "chef D30 retention; support ticket volume — unclear payout timelines are a top driver of chef churn and support load"
    depends_on: null

  - finding_id: BA-ERR-015
    surface_id: vp-onb-banner-rejected-title
    lens: business-analyst
    severity: P1
    issue: "Application rejection banner uses title case \"Application Rejected\" — cold and administrative, violates voice"
    evidence_excerpt: "Application Rejected"
    recommendation: >
      Replace with: "Application not approved" (sentence case, softer framing).
      Body: "Review the feedback below and resubmit when ready." Remove "Please review the feedback below and re-submit."
      The body already has "please" — the title does not need to be an administrative verdict. A chef hitting rejection is a high-churn moment.
    metric_hypothesis: "chef resubmission rate; chef D30 retention — rejections delivered harshly have lower resubmission rates than those framed with a recovery path"
    depends_on: null

  - finding_id: BA-ERR-016
    surface_id: vp-onb-banner-rejected-body
    lens: business-analyst
    severity: P1
    issue: "Rejection body uses \"was not approved\" passive and \"re-submit\" hyphenated — inconsistent vocabulary and voice"
    evidence_excerpt: "Your previous application was not approved. Please review the feedback below and re-submit."
    recommendation: >
      Replace with: "Your application was not approved. Review the feedback below and resubmit when you're ready."
      Fix: "re-submit" → "resubmit" (no hyphen, per style conventions). Remove "Please" — the chef is already stressed, instructions don't need politeness markers, just clarity.
    metric_hypothesis: "chef resubmission rate — friction in rejection copy reduces the likelihood of resubmission"
    depends_on: null

  - finding_id: BA-ERR-017
    surface_id: api-error-order-cancel-stage
    lens: business-analyst
    severity: P1
    issue: "\"Order cannot be cancelled at this stage\" — no refund implications, no alternative action offered"
    evidence_excerpt: "Order cannot be cancelled at this stage"
    recommendation: >
      Replace with: "This order can't be cancelled — it's already being prepared. Contact support if there's an urgent issue. [Contact support]"
      Customers who attempt to cancel a confirmed order and hit a wall without a support path will dispute the charge.
      The refund policy implication (no refund at this stage) must be implied or stated depending on policy.
    metric_hypothesis: "chargeback rate; customer support ticket volume — customers who cannot cancel and have no alternative become chargebacks"
    depends_on: null

  - finding_id: BA-ERR-018
    surface_id: api-error-chef-not-accepting
    lens: business-analyst
    severity: P1
    issue: "\"Chef is not accepting orders\" — no alternative suggestion, dead end for customer on chef detail page"
    evidence_excerpt: "Chef is not accepting orders"
    recommendation: >
      Replace with: "This chef isn't taking orders right now. Browse other chefs near you." with CTA "Find another chef".
      A customer blocked at the chef detail page with no forward path will abandon the session. The recovery CTA is the entire conversion here.
    metric_hypothesis: "browse-to-order conversion; session abandonment rate — customers who hit availability blocks without a redirect CTA exit at a higher rate"
    depends_on: null

  - finding_id: BA-ERR-019
    surface_id: web-err-error-boundary
    lens: business-analyst
    severity: P1
    issue: "Global error boundary uses \"We've been notified\" — unverified trust claim that may not be true for all error types"
    evidence_excerpt: "Something broke on our end. / We've been notified and we're looking into it... / Try again / Refresh page"
    recommendation: >
      Replace "We've been notified and we're looking into it..." with: "If this keeps happening, contact support."
      "We've been notified" is a false trust signal if error monitoring is not comprehensive. It also contradicts the no-fake-signals principle.
      The ellipsis (...) further weakens confidence. Keep: "Something went wrong on our end." Remove the notification claim.
    metric_hypothesis: "trust score; error recovery rate — false assurances that are not honored erode trust more than honest admissions"
    depends_on: null

  - finding_id: BA-ERR-020
    surface_id: vp-err-boundary-fallback
    lens: business-analyst
    severity: P1
    issue: "Vendor portal error boundary has no support path — chef in active order management hitting this state is stranded"
    evidence_excerpt: "Something went wrong / An unexpected error occurred while loading this page. / Try Again"
    recommendation: >
      Add a support contact path: "Something went wrong. [Try again] If orders are affected, [contact support]."
      A chef who hits an error boundary during active order fulfillment needs an emergency escalation path — the single "Try Again" button is insufficient.
    metric_hypothesis: "chef D7 retention; order fulfilment rate — chefs who lose access during active service and have no recovery path will churn"
    depends_on: null

  - finding_id: BA-ERR-021
    surface_id: dp-err-boundary-title
    lens: business-analyst
    severity: P1
    issue: "Delivery portal error boundary identical to vendor and web — no role-specific recovery for a driver who may have an active delivery"
    evidence_excerpt: "Something went wrong / An unexpected error occurred while loading this page."
    recommendation: >
      For drivers with an active delivery, add: "Something went wrong. If you have an active delivery, call the customer directly. [Contact support]"
      A driver who loses app access mid-delivery has safety and earnings implications. Generic error copy is a liability.
    metric_hypothesis: "driver D7 retention; delivery completion rate — drivers who lose context on active deliveries with no recovery instruction will DNF the order"
    depends_on: null

  - finding_id: BA-ERR-022
    surface_id: dp-err-access-denied
    lens: business-analyst
    severity: P1
    issue: "\"Access denied. Please check your credentials and try again\" — security-evoking language for what is likely a role mismatch"
    evidence_excerpt: "Access denied. Please check your credentials and try again."
    recommendation: >
      Replace with: "This portal is for delivery drivers only. Sign in with your driver account, or sign up as a driver."
      "Access denied" is threat language — it makes a legitimate user feel suspected rather than redirected. The issue is almost certainly a role mismatch, not a security incident.
    metric_hypothesis: "driver signup conversion; driver D1 activation — confusing rejection language causes legitimate applicants to abandon signup"
    depends_on: null

  - finding_id: BA-ERR-023
    surface_id: vp-auth-login-access-denied
    lens: business-analyst
    severity: P1
    issue: "Vendor portal access denied message references \"Fe3dr customer app\" — an incorrect or placeholder app name"
    evidence_excerpt: "This portal is only for vendor accounts. Please use the Fe3dr customer app."
    recommendation: >
      Replace "Fe3dr" with the correct app name. If the brand name is "Home Chef", the message should read:
      "This portal is for chefs only. Sign in to the Home Chef customer app instead."
      "Fe3dr" appears to be a placeholder or internal codename that should never have reached production copy.
    metric_hypothesis: "trust score; misdirected-user conversion — wrong app names in error messages signal unfinished product and damage brand credibility at the first login touchpoint"
    depends_on: null

  - finding_id: BA-ERR-024
    surface_id: vp-ux-orders-live-empty
    lens: business-analyst
    severity: P1
    issue: "Live orders empty state is passive — does not tell a new chef what to do to get their first order"
    evidence_excerpt: "No active orders right now. New orders will appear here automatically."
    recommendation: >
      For a chef with an unpublished menu: "No orders yet. Publish your menu to start accepting orders. [Go to menu]"
      For a chef with a live menu: "No active orders. Orders will appear here when customers place them."
      The current message does not differentiate between a new chef who hasn't published yet (who needs a CTA) and an active chef who is simply waiting (who needs reassurance).
    metric_hypothesis: "chef D1 activation; menu-publish rate — new chefs who see a waiting message without understanding the prerequisite (published menu) do not complete activation"
    depends_on: null

  # ─── P2: MISSED RECOVERY OPPORTUNITIES ON MEDIUM-TRAFFIC SURFACES ────────────

  - finding_id: BA-ERR-025
    surface_id: web-err-browse-empty
    lens: business-analyst
    severity: P2
    issue: "Empty chef search results offers \"Clear filters\" but no fallback to browse without filters"
    evidence_excerpt: "No chefs found / Try adjusting your filters or search query"
    recommendation: >
      Replace "Try adjusting your filters or search query" with: "No chefs match these filters. Clear all filters or browse all chefs near you."
      Add two CTAs: "Clear filters" and "Browse all chefs". The single generic suggestion wastes the recovery moment.
    metric_hypothesis: "browse-to-order conversion; search abandonment rate — customers who cannot find chefs via search need a clear fallback to unfiltered browse"
    depends_on: null

  - finding_id: BA-ERR-026
    surface_id: mc-home-empty
    lens: business-analyst
    severity: P2
    issue: "Mobile home empty state \"No chefs found / Try adjusting your filters\" — same passive fallback as web but worse on mobile"
    evidence_excerpt: "No chefs found / Try adjusting your filters"
    recommendation: >
      Replace with: "No chefs near you right now. Try a wider area or check back later." with CTA "Clear filters".
      On mobile the guidance to "try adjusting filters" requires the user to know which filter to change. A direct "clear all" action is more recoverable on a small screen.
    metric_hypothesis: "mobile browse-to-order conversion — filter abandonment on mobile is higher than web due to interaction cost"
    depends_on: null

  - finding_id: BA-ERR-027
    surface_id: web-ux-favorites-loggedout
    lens: business-analyst
    severity: P2
    issue: "Logged-out favorites state uses \"Log in\" — banned term, should be \"Sign in\""
    evidence_excerpt: "Log in to see your favorites"
    recommendation: >
      Replace with: "Sign in to see your saved chefs." — correct vocabulary per style guide (Sign in, not Log in).
      Also strengthen the value prop: the current message tells users what they can't do, not why they should sign in.
    metric_hypothesis: "signup/sign-in conversion from favorites page — vocabulary consistency reinforces brand trust and prevents confusion between portals"
    depends_on: null

  - finding_id: BA-ERR-028
    surface_id: web-err-checkout-no-addresses
    lens: business-analyst
    severity: P2
    issue: "No saved addresses empty state during checkout is a conversion blocker with a passive instruction"
    evidence_excerpt: "You don't have any saved addresses yet. Add one to continue."
    recommendation: >
      Replace with: "No delivery address saved. Add an address to complete your order." CTA: "Add address".
      The current text is accurate but the CTA is missing from the excerpt — if it's absent in the UI, this is a checkout blocking state with no action button.
      Verify a direct "Add address" button exists inline, not just instruction text.
    metric_hypothesis: "checkout completion rate — address-less users who see instruction text without an inline CTA have higher checkout abandonment"
    depends_on: null

  - finding_id: BA-ERR-029
    surface_id: web-ux-chefdetail-not-found
    lens: business-analyst
    severity: P2
    issue: "Chef not found state has \"Browse Chefs\" CTA which is good — but copy \"Chef not found\" is blunt and offers no explanation"
    evidence_excerpt: "Chef not found / Browse Chefs"
    recommendation: >
      Replace with: "This chef is no longer available. Browse other chefs near you." CTA: "Browse chefs" (sentence case).
      "Chef not found" sounds like a technical error. "No longer available" is honest and less alarming. The CTA case fix (title→sentence) is a vocabulary compliance fix.
    metric_hypothesis: "bounce rate on dead chef links; browse-to-order recovery — a softer 404 with explanation retains the customer in the funnel"
    depends_on: null

  - finding_id: BA-ERR-030
    surface_id: mc-catering-empty
    lens: business-analyst
    severity: P2
    issue: "Catering requests empty state uses 🍽️ emoji in functional copy — violates style guide"
    evidence_excerpt: "No requests yet / Submit a catering request to get quotes from chefs."
    recommendation: >
      Remove the emoji. The functional copy is good ("Submit a catering request to get quotes from chefs") — it explains the action and the value.
      Just enforce the no-emoji-in-body rule. Optionally add a CTA button: "Request catering".
    metric_hypothesis: "catering funnel activation — minor; primarily a brand consistency concern on a low-volume surface"
    depends_on: null

  - finding_id: BA-ERR-031
    surface_id: mc-social-empty
    lens: business-analyst
    severity: P2
    issue: "Social feed empty state uses 📸 emoji in body copy — violates style guide"
    evidence_excerpt: "No posts yet / Chefs will share their latest creations here."
    recommendation: >
      Remove the emoji. The body copy is adequate. Optionally add: "Follow chefs to see their updates here first." to add a behavioral nudge.
    metric_hypothesis: "social feature adoption — minor brand consistency finding"
    depends_on: null

  - finding_id: BA-ERR-032
    surface_id: vp-ux-reviews-empty
    lens: business-analyst
    severity: P2
    issue: "Empty reviews state \"No reviews yet\" — no guidance on how to get the first review"
    evidence_excerpt: "No reviews yet"
    recommendation: >
      Replace with: "No reviews yet. Complete your first order and your customer's review will appear here."
      Reviews are a trust and conversion signal for chefs. An empty three-word state misses the opportunity to explain how reviews are earned and why they matter.
    metric_hypothesis: "chef activation quality; order completion rate — chefs who understand the reviews mechanism are more motivated to fulfil their first orders"
    depends_on: null

  - finding_id: BA-ERR-033
    surface_id: mv-reviews-empty
    lens: business-analyst
    severity: P2
    issue: "Mobile vendor reviews empty state \"No reviews yet. Your first review will appear here.\" — same passive framing as web"
    evidence_excerpt: "No reviews yet. Your first review will appear here."
    recommendation: >
      Replace with: "No reviews yet. Complete your first order — your customer's review will appear here."
      The addition of the prerequisite ("complete your first order") turns a passive wait message into an activation nudge.
    metric_hypothesis: "chef D7 activation; first-order completion rate"
    depends_on: null

  - finding_id: BA-ERR-034
    surface_id: vp-ux-orders-history-empty
    lens: business-analyst
    severity: P2
    issue: "Order history empty state for date range filter gives no indication of what range contains orders"
    evidence_excerpt: "No orders found for the selected time period. Try a wider date range."
    recommendation: >
      Replace with: "No orders in this period. Widen your date range or [view all order history]." Add a CTA "View all history".
      The current message is accurate but provides no escape. Chefs who cannot find their order history will assume the data is lost.
    metric_hypothesis: "chef trust; support ticket volume — chefs who cannot find historical orders contact support unnecessarily"
    depends_on: null

  - finding_id: BA-ERR-035
    surface_id: vp-ux-earnings-error
    lens: business-analyst
    severity: P2
    issue: "Earnings load failure offers no support path — a chef who cannot see their earnings will escalate"
    evidence_excerpt: "Unable to load earnings / Please try again later."
    recommendation: >
      Replace with: "Couldn't load earnings. Try again — if the problem continues, contact support. [Retry] [Contact support]"
      Earnings are the primary retention hook for chefs. A load failure with no support escalation path will generate support tickets and churn.
    metric_hypothesis: "chef D30 retention; support ticket volume — earnings visibility directly drives chef motivation to stay active"
    depends_on: null

  - finding_id: BA-ERR-036
    surface_id: mv-earnings-no-payout-account
    lens: business-analyst
    severity: P2
    issue: "Mobile vendor \"No payout account configured\" — no CTA to configure it, dead end on a revenue-critical screen"
    evidence_excerpt: "No payout account configured"
    recommendation: >
      Replace with: "No payout account set up. Add your bank or UPI details to receive earnings." CTA: "Set up payouts" linking to payout settings.
      A chef who cannot see a path to receiving payment has no motivation to fulfil orders. This state must include an actionable CTA.
    metric_hypothesis: "chef D7 activation; payout setup completion rate — chefs without a payout method configured have near-zero long-term retention"
    depends_on: null

  - finding_id: BA-ERR-037
    surface_id: vp-ux-notifs-empty
    lens: business-analyst
    severity: P2
    issue: "Admin notifications empty state is verbose and unexplained — chef does not know what triggers notifications"
    evidence_excerpt: "No admin requests yet / Admin requests for document uploads, profile updates, and review feedback will appear here."
    recommendation: >
      The explanation is actually good for context. Tighten: "No admin requests. When admin needs action from you — document uploads, profile changes, or review responses — it will appear here."
      The current copy is slightly awkward ("Admin requests for document uploads, profile updates, and review feedback will appear here" reads like a category list).
    metric_hypothesis: "chef compliance rate; onboarding completion — chefs who understand the notification system respond faster to admin requests"
    depends_on: null

  - finding_id: BA-ERR-038
    surface_id: dp-empty-active-body
    lens: business-analyst
    severity: P2
    issue: "Active delivery empty state directs driver to \"check available deliveries\" — good direction but telegram copy not followed"
    evidence_excerpt: "Check available deliveries to pick up a new order."
    recommendation: >
      Replace with: "No active delivery. Check Available tab." (telegraphic ≤4 words per sentence for driver persona per style guide tone matrix).
      The current sentence is 9 words — driver UI should be glanceable and imperative. CTA should link directly to the available tab.
    metric_hypothesis: "driver delivery acceptance rate — telegraphic copy on driver UI reduces cognitive load and increases tab-switch speed"
    depends_on: null

  - finding_id: BA-ERR-039
    surface_id: md-err-002
    lens: business-analyst
    severity: P2
    issue: "Mobile delivery available-deliveries empty state tells driver to \"Pull to refresh\" — gesture instruction in copy is redundant and low-value"
    evidence_excerpt: "No deliveries available nearby. Pull to refresh."
    recommendation: >
      Replace with: "No deliveries nearby. Check again soon." — remove the pull-to-refresh instruction (the pull affordance is standard and the gesture hint adds noise).
      Alternatively, provide an auto-refresh indication: "No deliveries nearby. Checking every 30 seconds."
    metric_hypothesis: "driver engagement; session length — a passive wait message is better than a gesture instruction that implies nothing is happening"
    depends_on: null

  - finding_id: BA-ERR-040
    surface_id: md-emp-007
    lens: business-analyst
    severity: P2
    issue: "Permission denied for fleet management gives admin contact instruction but no action — stranded driver"
    evidence_excerpt: "Fleet management is available for fleet managers only. Contact your administrator to request access."
    recommendation: >
      Replace with: "Fleet management is for fleet managers. Ask your admin for access." — shorter, driver-persona telegraphic style.
      If an in-app contact mechanism exists, link to it. A "contact your administrator" instruction with no link is an incomplete recovery path.
    metric_hypothesis: "driver activation; role-escalation request rate — drivers who cannot reach admins via the app to request access churn silently"
    depends_on: null

  - finding_id: BA-ERR-041
    surface_id: api-error-payment-refund
    lens: business-analyst
    severity: P2
    issue: "Refund error messages expose constraint details with no refund policy or timeline context"
    evidence_excerpt: "Can only refund completed payments / Refund amount cannot exceed order total / Failed to process refund / No Razorpay payment found for this order"
    recommendation: >
      Customer-facing refund errors should be generic and empathetic: "Refund could not be processed. Contact support to resolve this."
      The constraint details ("no Razorpay payment found") must not reach the customer — these are internal diagnostics.
      Separately, every refund-related error should include the expected refund timeline when a refund IS being processed.
    metric_hypothesis: "refund dispute rate; support ticket volume — customers who see confusing refund errors will initiate chargebacks instead of contacting support"
    depends_on: null

  - finding_id: BA-ERR-042
    surface_id: api-error-generic-failed
    lens: business-analyst
    severity: P2
    issue: "~100 \"Failed to X\" API error variants are passed directly to UI with no user-friendly translation layer"
    evidence_excerpt: "Failed to create address / Failed to process refund / Failed to fetch chefs / Failed to send message [and ~95 more variants]"
    recommendation: >
      Establish a UI-layer error translation map: API error codes → friendly copy following the style guide format "What happened → What to do."
      Currently, raw "Failed to X" strings surface verbatim to customers, chefs, and drivers. These strings are developer-facing diagnostics.
      Priority translation targets: any "Failed to" that appears on checkout, payment, order creation, and delivery acceptance flows.
    metric_hypothesis: "checkout and order completion rate; trust score — developer-toned error strings lower consumer confidence at high-stakes moments"
    depends_on: null

  - finding_id: BA-ERR-043
    surface_id: api-error-auth-not-verified
    lens: business-analyst
    severity: P2
    issue: "Two variants of account-not-verified error exist: \"Account not verified\" and \"Account not verified yet\" — inconsistent and no recovery path"
    evidence_excerpt: "Account not verified / Account not verified yet"
    recommendation: >
      Consolidate to one message and add a recovery path: "Your account isn't verified yet. Check your email for the verification link, or [resend it]."
      "Not verified yet" is marginally better (implies it will be fixed) but both lack the critical resend-verification CTA.
      Unverified accounts that cannot self-serve verification will churn rather than contact support.
    metric_hypothesis: "signup completion rate; email verification rate — adding a resend CTA inline with the error message increases verification completion"
    depends_on: null

  - finding_id: BA-ERR-044
    surface_id: web-err-login-generic
    lens: business-analyst
    severity: P2
    issue: "Generic login error \"Something went wrong. Please try again.\" has no diagnostic context or support link"
    evidence_excerpt: "Something went wrong. Please try again."
    recommendation: >
      Keep "Something went wrong. Try again." as the primary message for transient errors.
      Add a secondary line after 2+ failed attempts: "Still having trouble? [Contact support]"
      A persistent "Something went wrong" loop with no escape route causes permanent login abandonment — especially for chefs and drivers who rely on the app for income.
    metric_hypothesis: "login completion rate; D1 reactivation — login failures with no support path result in silent churn"
    depends_on: null

  - finding_id: BA-ERR-045
    surface_id: dp-err-generic-auth
    lens: business-analyst
    severity: P2
    issue: "Delivery portal generic auth error same pattern as web — no driver-specific recovery path"
    evidence_excerpt: "Something went wrong. Please try again."
    recommendation: >
      Same fix as BA-ERR-044 with driver-specific support copy. For the delivery portal, drivers losing access mid-shift need a higher-urgency support path.
      After 2 failures: "Can't sign in? Contact fleet support immediately if you have an active delivery."
    metric_hypothesis: "driver session recovery rate; delivery completion rate — drivers who cannot sign in and have active deliveries need emergency escalation"
    depends_on: BA-ERR-044

  - finding_id: BA-ERR-046
    surface_id: vp-onb-docs-upload-failed
    lens: business-analyst
    severity: P2
    issue: "Document upload failure \"Upload failed\" — no guidance on file format, size, or retry mechanism during onboarding"
    evidence_excerpt: "Upload failed"
    recommendation: >
      Replace with: "Upload failed. Check the file is a JPEG, PNG, or PDF under 10 MB, then try again."
      Two-word error toasts during onboarding are conversion killers. The chef has no idea why the upload failed or what to change.
      This is particularly damaging at the document step — it is a mandatory onboarding gate.
    metric_hypothesis: "chef onboarding completion rate — upload failures at the document step are a top drop-off point when copy provides no remediation"
    depends_on: null

  - finding_id: BA-ERR-047
    surface_id: mv-onb-docs-upload-fail
    lens: business-analyst
    severity: P2
    issue: "Mobile vendor document upload failure same four-word message — same activation blocker as vendor portal"
    evidence_excerpt: "Upload failed. Please try again."
    recommendation: >
      Replace with: "Upload failed. Use a JPEG, PNG, or PDF under 5 MB." — telegraphic and actionable.
      The "Please try again" adds no information. Give the chef the constraint so they can fix it.
    metric_hypothesis: "chef mobile onboarding completion rate — same finding as BA-ERR-046 on a different surface"
    depends_on: BA-ERR-046

  - finding_id: BA-ERR-048
    surface_id: api-error-upload-size
    lens: business-analyst
    severity: P2
    issue: "Inconsistent file size formatting across upload errors: \"10MB\" vs \"5 MB\" vs \"5MB\" — creates user confusion"
    evidence_excerpt: "File too large (max 10MB) / File too large. Maximum 5 MB. / Profile photo too large (max 5MB) / Each image must be under 5 MB"
    recommendation: >
      Standardize to: "File too large. Maximum [X] MB." (space before MB, period-terminated sentence).
      Apply consistently across all upload error points. Inconsistent formatting implies different limits when they may be the same limit.
    metric_hypothesis: "upload completion rate — format inconsistency causes users to re-read and second-guess limits, increasing abandonment"
    depends_on: null

  - finding_id: BA-ERR-049
    surface_id: api-error-upload-type
    lens: business-analyst
    severity: P2
    issue: "Five variants of file-type rejection with slightly different allowed lists — user confusion about what is actually accepted"
    evidence_excerpt: "Invalid file type. Allowed: JPEG, PNG, PDF. / Invalid file type. Allowed: JPEG, PNG, WebP, PDF / Invalid file type. Allowed: JPEG, PNG, WebP. / Invalid image type. Allowed: JPEG, PNG, WebP. / Profile photo must be JPEG or PNG"
    recommendation: >
      Audit which types are genuinely accepted per upload context and create one canonical message per context.
      Example: "Unsupported file type. Use JPEG, PNG, or PDF." (for documents); "Unsupported image. Use JPEG, PNG, or WebP." (for photos).
      Having 5 slightly different lists makes users unsure which formats are universally safe to use.
    metric_hypothesis: "upload completion rate; onboarding completion — type confusion at the document step blocks chef activation"
    depends_on: null

  - finding_id: BA-ERR-050
    surface_id: vp-onb-review-missing-docs
    lens: business-analyst
    severity: P2
    issue: "Missing documents warning \"Please go back to Step 4\" — step references become incorrect if the wizard structure changes"
    evidence_excerpt: "Missing required documents / Please go back to Step 4 and upload: ..."
    recommendation: >
      Replace step number references with section labels: "Missing required documents. Go back to Document uploads and add the required files."
      Use a direct CTA button: "Go to Document uploads" that navigates to the correct step rather than relying on the user to count backward.
    metric_hypothesis: "onboarding completion rate — navigation friction in multi-step wizards is a drop-off cause; direct jump links remove that friction"
    depends_on: null

  - finding_id: BA-ERR-051
    surface_id: vp-onb-review-policies-pending
    lens: business-analyst
    severity: P2
    issue: "Unaccepted policies warning uses step number reference — same fragility as BA-ERR-050"
    evidence_excerpt: "Policies not accepted / Please go back to Step 5 and accept all required policies."
    recommendation: >
      Replace with: "Policies not accepted. Go back to Policies and accept all required agreements." with CTA "Go to Policies".
    metric_hypothesis: "chef onboarding completion rate — same finding as BA-ERR-050"
    depends_on: BA-ERR-050

  - finding_id: BA-ERR-052
    surface_id: api-error-auth-reset-expired
    lens: business-analyst
    severity: P2
    issue: "Three near-duplicate reset token error variants; none include a \"resend reset link\" recovery path"
    evidence_excerpt: "Reset token has expired / Invalid or expired reset token / Invalid or expired enrollment token"
    recommendation: >
      Consolidate to: "This link has expired. Request a new one." with CTA "Resend reset link".
      A reset link expiry error with no resend action is a permanent lock-out for the affected user. The recovery CTA is mandatory.
    metric_hypothesis: "password reset completion rate — expired token errors without a resend CTA result in permanent account loss for that session"
    depends_on: null

  - finding_id: BA-ERR-053
    surface_id: api-error-auth-2fa
    lens: business-analyst
    severity: P2
    issue: "2FA errors are vague with no guidance — admin who loses 2FA access has no recovery path in the error copy"
    evidence_excerpt: "Failed to enable 2FA / Failed to start 2FA challenge / Invalid or expired challenge"
    recommendation: >
      For "Invalid or expired challenge": "This code has expired. Request a new one." with retry CTA.
      For "Failed to enable 2FA": "2FA setup failed. Try again or contact support."
      Admins locked out of 2FA become permanently blocked and represent a high-severity support escalation.
    metric_hypothesis: "admin 2FA adoption rate; admin lockout ticket volume — clear 2FA error recovery reduces lockouts and increases 2FA completion"
    depends_on: null

  - finding_id: BA-ERR-054
    surface_id: dp-err-step4-plans-load
    lens: business-analyst
    severity: P2
    issue: "Subscription plans load failure during driver onboarding \"Failed to load subscription plans\" — no driver can proceed; entire onboarding blocked"
    evidence_excerpt: "Failed to load subscription plans"
    recommendation: >
      Replace with: "Couldn't load subscription plans. Try again — or contact support if this continues. [Retry] [Contact support]"
      Subscription plan selection is a mandatory onboarding step for drivers. A load failure with only a retry button and no support escalation blocks all drivers who experience this.
    metric_hypothesis: "driver onboarding completion rate; support ticket volume — plan-load failures are a critical onboarding blocker requiring an escalation path"
    depends_on: null

  - finding_id: BA-ERR-055
    surface_id: vp-ux-dashboard-pending-empty
    lens: business-analyst
    severity: P2
    issue: "\"All caught up! / No pending orders right now\" — exclamation mark in non-celebratory dashboard state"
    evidence_excerpt: "All caught up! / No pending orders right now"
    recommendation: >
      Replace with: "No pending orders. New orders will appear here automatically."
      "All caught up!" is a faux-celebration on an operational dashboard. A chef with zero orders does not have a reason to feel celebrated — they need clear information.
      Style guide Rule 1: no exclamation marks except in genuine celebration.
    metric_hypothesis: "brand consistency; chef engagement — jarring tone in an operational context signals design inconsistency"
    depends_on: null

  # ─── P3: MINOR FRICTION ON LOW-TRAFFIC SURFACES ──────────────────────────────

  - finding_id: BA-ERR-056
    surface_id: ap-providerdetail-dialog-bodies
    lens: business-analyst
    severity: P3
    issue: "Confirmation dialogs use \"Are you sure you want to...\" phrasing — vague and violates modal subtitle pattern"
    evidence_excerpt: "Are you sure you want to delete this delivery provider? This action cannot be undone."
    recommendation: >
      Replace with consequence-first format per style guide: "Deleting this provider removes all delivery assignments. This cannot be undone."
      "Are you sure?" adds no information. The consequence sentence is the only content that matters.
    metric_hypothesis: "admin error rate; accidental deletion rate — consequence-first dialogs reduce misclick-driven errors"
    depends_on: null

  - finding_id: BA-ERR-057
    surface_id: ap-staffdetail-confirm-deactivate
    lens: business-analyst
    severity: P3
    issue: "Staff deactivation confirmation uses \"Are you sure\" pattern — same pattern violation as BA-ERR-056"
    evidence_excerpt: "Are you sure you want to deactivate this staff member? They will lose access to all portals."
    recommendation: >
      Replace with: "Deactivating this staff member removes their access to all portals immediately."
      Keep the consequence sentence — it is informative. Remove "Are you sure you want to."
    metric_hypothesis: "admin error rate — consistent modal patterns reduce cognitive load in admin operations"
    depends_on: BA-ERR-056

  - finding_id: BA-ERR-058
    surface_id: ap-providers-dialog-delete-body
    lens: business-analyst
    severity: P3
    issue: "Delivery provider delete dialog uses \"Are you sure\" and \"This action cannot be undone\" — both are vague per style guide"
    evidence_excerpt: "Are you sure you want to delete this delivery provider? This action cannot be undone."
    recommendation: >
      Replace with: "Deleting this provider removes all associated delivery zones and configuration permanently."
      "This action cannot be undone" is a banned pattern per the style guide — it explains nothing about what is undone.
    metric_hypothesis: "admin confidence; accidental deletion rate — specific consequences are more informative than generic irreversibility warnings"
    depends_on: BA-ERR-056

  - finding_id: BA-ERR-059
    surface_id: api-error-validation-required
    lens: business-analyst
    severity: P3
    issue: "18 variants of \"X is required\" with inconsistent field name casing (e.g., \"chefId\" vs \"country\") — internal inconsistency"
    evidence_excerpt: "Email is required / chefId is required / country is required / Verification token is required"
    recommendation: >
      Standardize: all required-field error messages should use sentence-case field names readable to the user, not camelCase API field names.
      "chefId is required" should never surface to a customer UI — translate to "Please select a chef" or the equivalent user-facing label.
    metric_hypothesis: "form completion rate; customer support — technical field names in validation errors confuse non-technical users"
    depends_on: null

  - finding_id: BA-ERR-060
    surface_id: api-error-auth-unauthorized
    lens: business-analyst
    severity: P3
    issue: "Four variants of unauthorized error exist including mixed case \"Unauthorized\" and \"unauthorized\" — inconsistency in production"
    evidence_excerpt: "Unauthorized / unauthorized / Authentication required / Token required"
    recommendation: >
      Consolidate to a single user-facing message: "Your session has expired. Sign in again."
      The technical variants ("Token required", "unauthorized") should never reach user-facing copy — they are middleware responses.
      If these strings are surfaced directly, add a UI translation layer.
    metric_hypothesis: "login recovery rate — session expiry messages that reference tokens confuse non-technical users"
    depends_on: null

  - finding_id: BA-ERR-061
    surface_id: ap-secsettings-apikey-empty
    lens: business-analyst
    severity: P3
    issue: "\"No keys yet.\" — full stop with trailing period is inconsistent with most other empty states in admin portal"
    evidence_excerpt: "No keys yet."
    recommendation: >
      Replace with: "No API keys yet. Generate a key to start using the API." with CTA "Generate key".
      The current state is a dead end — an admin who lands here has no path to creating a key.
    metric_hypothesis: "admin API adoption rate — empty states without CTAs on capability-discovery pages slow feature adoption"
    depends_on: null

  - finding_id: BA-ERR-062
    surface_id: vp-onb-review-empty-cuisines
    lens: business-analyst
    severity: P3
    issue: "\"None selected\" for cuisines in onboarding review — does not tell the chef if cuisine selection is required"
    evidence_excerpt: "None selected"
    recommendation: >
      Replace with: "No cuisines selected. Add at least one cuisine type to continue." — make the requirement explicit in context.
      Chefs who see "None selected" at review may not know if it blocks submission.
    metric_hypothesis: "onboarding completion rate — ambiguous required-field states at the review step cause confusion and drop-off"
    depends_on: null

  - finding_id: BA-ERR-063
    surface_id: vp-onb-review-empty-days
    lens: business-analyst
    severity: P3
    issue: "\"No days set\" for operating hours in onboarding review — same ambiguity as BA-ERR-062"
    evidence_excerpt: "No days set"
    recommendation: >
      Replace with: "No operating hours set. Go back to add your availability." with CTA "Set availability".
    metric_hypothesis: "chef onboarding completion rate — same as BA-ERR-062"
    depends_on: BA-ERR-062

  - finding_id: BA-ERR-064
    surface_id: api-error-delivery-online
    lens: business-analyst
    severity: P3
    issue: "\"Partner has reached maximum concurrent deliveries\" — uses \"Partner\" terminology in a driver-facing context"
    evidence_excerpt: "Partner has reached maximum concurrent deliveries / Partner is not verified or active"
    recommendation: >
      Replace "Partner" with "you" in driver-facing contexts: "You've reached the maximum active deliveries. Complete your current delivery first."
      "Partner is not verified or active" → "Your account isn't verified yet. Contact support." — includes a path forward.
    metric_hypothesis: "driver experience quality — role-terminology mismatch signals a backend string was exposed without UI translation"
    depends_on: null

  - finding_id: BA-ERR-065
    surface_id: api-error-delivery-already-active
    lens: business-analyst
    severity: P3
    issue: "Snake_case slug \"no_active_delivery\" exposed as error message variant alongside human-readable text"
    evidence_excerpt: "You already have an active delivery / no_active_delivery"
    recommendation: >
      Remove "no_active_delivery" from any user-facing surface. This is a machine-readable enum that must never appear in UI copy.
      Add a UI translation layer to convert status slugs to human copy before rendering.
    metric_hypothesis: "driver trust; technical perception — status codes in UI copy signal a broken frontend"
    depends_on: null

  - finding_id: BA-ERR-066
    surface_id: mv-more-logout-confirm-body
    lens: business-analyst
    severity: P3
    issue: "Logout confirmation title is \"Logout\" (single word) — should be \"Sign out\" per vocabulary list"
    evidence_excerpt: "Are you sure you want to log out? [Title: Logout]"
    recommendation: >
      Replace title "Logout" with "Sign out". Replace body "Are you sure you want to log out?" with "Sign out of your account?"
      Two vocabulary violations on one dialog (Logout, log out — both banned per style guide). Chef persona copy should also be crisper.
    metric_hypothesis: "vocabulary consistency — minor, but confirms style guide compliance is not applied consistently in mobile"
    depends_on: null

  - finding_id: BA-ERR-067
    surface_id: mv-onb-personal-validation-alert
    lens: business-analyst
    severity: P3
    issue: "\"Validation Error\" as alert title is a technical term, not a user-oriented label"
    evidence_excerpt: "Validation Error"
    recommendation: >
      Replace with: "Check your details" — descriptive, not technical.
      "Validation Error" is developer language. The user does not know what "validation" means in this context.
    metric_hypothesis: "mobile onboarding completion rate — technical terminology in mobile alerts increases confusion at form-completion steps"
    depends_on: null

  - finding_id: BA-ERR-068
    surface_id: mv-onb-review-submit-error-title
    lens: business-analyst
    severity: P3
    issue: "\"Submission Error\" alert title — same technical phrasing as BA-ERR-067"
    evidence_excerpt: "Submission Error"
    recommendation: >
      Replace with: "Couldn't submit" — plain English, non-technical.
    metric_hypothesis: "chef mobile onboarding completion — technical alert titles create anxiety at the final step"
    depends_on: BA-ERR-067

  - finding_id: BA-ERR-069
    surface_id: ap-layout-offline-banner
    lens: business-analyst
    severity: P3
    issue: "Admin offline banner \"Data may not be up to date\" — vague about which operations are affected"
    evidence_excerpt: "You're offline. Data may not be up to date."
    recommendation: >
      Replace with: "You're offline. Approvals and status changes will not save until reconnected."
      The admin portal has real consequences for offline actions (approving chefs, suspending users). The banner should be specific about what cannot be done.
    metric_hypothesis: "admin error rate — vague offline banners lead admins to take actions they believe will save, but do not"
    depends_on: null

  - finding_id: BA-ERR-070
    surface_id: vp-ux-layout-offline
    lens: business-analyst
    severity: P3
    issue: "Vendor offline banner is stronger than others: \"Orders will sync when connected\" — good but inconsistently applied"
    evidence_excerpt: "You're offline. Orders will sync when connected."
    recommendation: >
      This is the best offline banner in the inventory — it explains the sync behavior.
      Propagate this pattern to all other portals: web uses "Some features may be unavailable" (weaker), admin uses "Data may not be up to date" (weaker), delivery uses "Updates will sync when connected" (acceptable).
      Standardize to the vendor-portal pattern as the benchmark.
    metric_hypothesis: "cross-portal brand consistency; operational confidence — users who understand offline sync behavior are less likely to panic or create duplicate actions"
    depends_on: null

```

## Brand Voice findings

```yaml
# Brand-Voice findings — ERRORS-EMPTY category (282 surfaces)
# Lens: cross-surface and cross-app voice consistency vs STYLE-GUIDE.md + .impeccable.md
# Severity: P0 cross-surface contradiction · P1 entry-surface drift · P2 secondary tone drift · P3 punctuation/capitalization

# =============================================================================
# CROSS-APP DRIFT — same error class, different copy across 7 apps + API
# =============================================================================

findings:

  - finding_id: BV-001
    surface_id: web-err-login-generic
    lens: brand-voice
    severity: P0
    issue: "'Something went wrong. Please try again.' generic copy duplicated verbatim across 6 surfaces; vendor-portal, admin-portal, web, delivery-portal, mobile-customer, mobile-vendor — and the duplication is itself the violation (no contextual variant)"
    evidence_excerpt: "Something went wrong. Please try again."
    related_surfaces: ["web-err-generic-retry", "vp-auth-login-generic-error", "ap-auth-error-generic", "dp-err-generic-auth", "mc-onb-step3-error", "mv-onb-review-submit-error-body"]
    recommendation: "Centralize as a single shared string token (e.g., errors.generic) — but require a contextual variant ('Sign-in failed. Try again.', 'Couldn't save changes. Try again.') for every place it appears. Generic 'something went wrong' violates Rule 2 (plain English over jargon) — 'something' is vague."
    depends_on: null

  - finding_id: BV-002
    surface_id: web-err-login-session-expired
    lens: brand-voice
    severity: P3
    issue: "'Your session has expired. Please sign in again.' appears identically in web, vendor-portal, delivery-portal, admin-portal — this one IS consistent and good. Flag as positive baseline, no rewrite."
    evidence_excerpt: "Your session has expired. Please sign in again."
    related_surfaces: ["vp-auth-login-session-expired", "ap-auth-error-session-expired", "dp-err-session-expired"]
    recommendation: "Keep as-is. Centralize as shared token errors.session_expired."
    depends_on: null

  - finding_id: BV-003
    surface_id: web-err-error-boundary
    lens: brand-voice
    severity: P0
    issue: "Error-boundary fallback copy drifts heavily across apps: web uses 'Something broke on our end. We've been notified and we're looking into it...'; vendor-portal uses 'An unexpected error occurred while loading this page.'; admin-portal uses same as VP; mobile apps have no global boundary copy at all"
    evidence_excerpt: "Unexpected error / Something broke on our end. / We've been notified and we're looking into it..."
    related_surfaces: ["vp-err-boundary-fallback", "ap-errorboundary-body", "dp-err-boundary-body"]
    recommendation: "Define one canonical error-boundary string: 'Something broke. We've logged it and are looking into it. Try again or reload.' Same across all 7 apps. The phrase 'unexpected error occurred while loading this page' is bureaucratic — passive, no agent, no recovery suggested."
    depends_on: null

  - finding_id: BV-004
    surface_id: web-err-layout-offline
    lens: brand-voice
    severity: P1
    issue: "Offline banner copy drifts: web 'You're offline. Some features may be unavailable.' / vendor-portal 'You're offline. Orders will sync when connected.' / delivery-portal 'You're offline. Updates will sync when connected.' / admin-portal 'You're offline. Data may not be up to date.' — four variants of the same network state"
    evidence_excerpt: "You're offline. Some features may be unavailable."
    related_surfaces: ["vp-ux-layout-offline", "dp-err-offline-banner", "ap-layout-offline-banner"]
    recommendation: "Define per-persona variants but keep structure identical: customer 'You're offline. We'll sync when you reconnect.' / chef 'Offline. Orders sync when you reconnect.' (shorter — chef in kitchen) / driver 'Offline. Updates sync when connected.' (telegraphic per persona matrix) / admin 'Offline. Data may be stale.' (neutral operator)"
    depends_on: null

  - finding_id: BV-005
    surface_id: vp-ux-orders-live-empty
    lens: brand-voice
    severity: P0
    issue: "'No pending orders' empty state drifts between vendor-portal ('All caught up! / No pending orders right now') and mobile-vendor ('No pending orders / New orders will appear here automatically') — same chef, two different products, two different tones"
    evidence_excerpt: "All caught up! / No pending orders right now"
    related_surfaces: ["mv-orders-empty-title", "mv-orders-empty-body", "vp-ux-dashboard-pending-empty"]
    recommendation: "'All caught up!' violates Rule 1 (no exclamation marks). Keep the structural shorter mobile copy across both: 'No pending orders. New orders appear here automatically.' Same string, both surfaces."
    depends_on: null

  - finding_id: BV-006
    surface_id: web-ux-cart-empty
    lens: brand-voice
    severity: P0
    issue: "Cart-empty copy drifts: web 'Your cart is empty. Looks like you haven't added any items yet. Browse our home chefs...' (28 words) vs mobile-customer 'Your cart is empty.' (4 words) vs CartSheet 'Your cart is empty' (4 words, no period) — same concept, three lengths"
    evidence_excerpt: "Your cart is empty / Looks like you haven't added any items yet. Browse our home chefs..."
    related_surfaces: ["mc-checkout-empty-cart", "mc-cartsheet-empty"]
    recommendation: "Drop 'Looks like you haven't added any items yet' — it's filler that violates Rule 1 (loud) and is the exact 'cuteness' pattern STYLE-GUIDE banned. Standardize: title 'Your cart is empty.' + body 'Browse chefs to start an order.' Same wording on all three surfaces."
    depends_on: null

  - finding_id: BV-007
    surface_id: web-err-browse-empty
    lens: brand-voice
    severity: P2
    issue: "'No chefs found' empty state drifts: web 'No chefs found / Try adjusting your filters or search query' vs mobile-customer 'No chefs found / Try adjusting your filters' (no 'or search query' tail) vs admin-portal 'No chefs found' (no body)"
    evidence_excerpt: "No chefs found / Try adjusting your filters or search query"
    related_surfaces: ["mc-home-empty", "ap-chefs-empty"]
    recommendation: "Customer-facing (web + mobile-customer): same string 'No chefs match your filters. Try widening the search.' Admin: 'No chefs found' (neutral operator tone OK)."
    depends_on: null

  - finding_id: BV-008
    surface_id: web-err-orderdetail-not-found
    lens: brand-voice
    severity: P2
    issue: "'Order not found' uses inconsistent CTA capitalization: web 'View All Orders' (Title Case violates STYLE-GUIDE.md §4 buttons — sentence case required); mobile-customer 'Go Back' (Title Case)"
    evidence_excerpt: "Order not found / View All Orders"
    related_surfaces: ["mc-order-detail-not-found"]
    recommendation: "Sentence case: 'View all orders' / 'Go back'. Same body across surfaces."
    depends_on: null

  - finding_id: BV-009
    surface_id: vp-ux-menu-empty
    lens: brand-voice
    severity: P1
    issue: "Vendor menu-empty CTA 'Add Your First Item' is Title Case — STYLE-GUIDE bans Title Case on buttons (sentence case + verb-first ≤3 words). Mobile equivalent is more conformant: 'Tap + to add your first item.' — but mixed metaphors across surfaces"
    evidence_excerpt: "No menu items yet / Start building your menu by adding your first dish. / Add Your First Item"
    related_surfaces: ["mv-menu-empty"]
    recommendation: "Sentence-case CTA: 'Add first item'. Body unified: 'No menu items. Add your first dish.' (matches STYLE-GUIDE empty-state pattern exactly)."
    depends_on: null

  - finding_id: BV-010
    surface_id: dp-empty-active-title
    lens: brand-voice
    severity: P1
    issue: "Driver app titles use Title Case ('No Active Delivery', 'No Available Deliveries', 'No Deliveries Yet') — violates STYLE-GUIDE sentence-case rule AND violates driver persona-matrix telegraphic ≤4-word imperative tone"
    evidence_excerpt: "No Active Delivery / No Available Deliveries / No Deliveries Yet"
    related_surfaces: ["dp-empty-available-title", "dp-empty-history-title", "md-emp-001"]
    recommendation: "Driver telegraphic: 'No active delivery.' / 'No deliveries available.' / 'No deliveries yet.' Sentence case + period. Drop redundant 'Yet' where the body already says 'will appear here'."
    depends_on: null

  - finding_id: BV-011
    surface_id: md-emp-006
    lens: brand-voice
    severity: P1
    issue: "Mobile-delivery uses Title Case heading 'Fleet Management' and 'Staff Management' on lock screens — Title Case banned, plus 'Management' is generic SaaS-dashboard jargon"
    evidence_excerpt: "Fleet Management / Staff Management"
    related_surfaces: ["md-emp-010"]
    recommendation: "Sentence case + plain: 'Fleet access required.' / 'Staff access required.' Body stays similar."
    depends_on: null

  - finding_id: BV-012
    surface_id: dp-empty-active-body
    lens: brand-voice
    severity: P2
    issue: "Driver app body copy is too conversational for driver persona-matrix telegraphic ≤4-word tone: 'Check available deliveries to pick up a new order.' (9 words) / 'New orders will appear here when they're ready for pickup.' (10 words)"
    evidence_excerpt: "Check available deliveries to pick up a new order."
    related_surfaces: ["dp-empty-available-body", "md-emp-002"]
    recommendation: "Driver-telegraphic rewrite: 'Open Available to pick one up.' (6 words) / 'New orders appear when ready.' (5 words). STYLE-GUIDE persona-matrix says driver = imperative ≤4 words where possible."
    depends_on: null

# =============================================================================
# "OOPS / UH OH / WHOOPS / SOMETHING WENT WRONG" PATTERN
# =============================================================================

  - finding_id: BV-013
    surface_id: web-err-login-generic
    lens: brand-voice
    severity: P1
    issue: "'Something went wrong' pattern appears 6+ times across the codebase — STYLE-GUIDE §4 errors explicitly bans 'Oops!' / 'Uh oh!' family; 'something went wrong' is the same anti-pattern in a different costume (vague, no cause, no recovery)"
    evidence_excerpt: "Something went wrong. Please try again."
    related_surfaces: ["web-err-generic-retry", "vp-auth-login-generic-error", "ap-auth-error-generic", "dp-err-generic-auth"]
    recommendation: "Replace with cause + action per STYLE-GUIDE: 'Sign-in failed. Check your email and password.' / 'Couldn't reach the server. Try again.' / 'Network lost. Reconnect to keep tracking.' (the exact good example from STYLE-GUIDE)"
    depends_on: null

  - finding_id: BV-014
    surface_id: mc-profile-save-error
    lens: brand-voice
    severity: P2
    issue: "Mobile-customer wraps every Alert.alert error with a generic title 'Error' — that's the JS/RN equivalent of 'Oops!'. Customer-facing surface, six instances"
    evidence_excerpt: "Error / Could not update profile. Please try again."
    related_surfaces: ["mc-catering-submit-error", "mv-onb-personal-validation-alert", "mv-onb-review-submit-error-title", "mv-profile-displayname-required"]
    recommendation: "Drop the 'Error' title — the body line is the title. Use 'Couldn't save profile.' as the leading line; if a system alert needs a title, use the cause ('Profile not saved') not the genre. Vendor 'Validation Error' / 'Submission Error' similar — use 'Check your details' / 'Submission failed' instead."
    depends_on: null

  - finding_id: BV-015
    surface_id: mv-onb-docs-required-title
    lens: brand-voice
    severity: P2
    issue: "Mobile-vendor uses Title Case alert titles ('Documents Required', 'Permission Required', 'Terms Required', 'Policy Required', 'Validation Error', 'Submission Error') — Title Case banned by STYLE-GUIDE, AND noun-only titles are vague non-imperative"
    evidence_excerpt: "Documents Required / Permission Required / Terms Required / Policy Required"
    related_surfaces: ["mv-onb-docs-perm-title", "mv-onb-policies-terms-required-title", "mv-onb-policies-policy-required-title", "mv-onb-personal-validation-alert", "mv-onb-review-submit-error-title"]
    recommendation: "Sentence-case + actionable: 'Upload required documents' / 'Camera access needed' / 'Accept terms to continue' / 'Pick a cancellation policy' / 'Check your details' / 'Couldn't submit'."
    depends_on: null

# =============================================================================
# BLAME LANGUAGE — "you forgot", "you haven't"
# =============================================================================

  - finding_id: BV-016
    surface_id: web-ux-cart-empty
    lens: brand-voice
    severity: P2
    issue: "'Looks like you haven't added any items yet' — combines 'looks like' (banned cuteness) with blame phrasing 'you haven't'. STYLE-GUIDE §4 empty-state pattern is 'Why it's empty → One action', not blame"
    evidence_excerpt: "Looks like you haven't added any items yet."
    related_surfaces: []
    recommendation: "Drop entirely. 'Your cart is empty. Browse chefs to start an order.'"
    depends_on: null

  - finding_id: BV-017
    surface_id: web-err-orders-empty
    lens: brand-voice
    severity: P2
    issue: "'You haven't placed any orders yet.' — blame phrasing. STYLE-GUIDE allows 'No orders yet. Browse chefs near you.' (positive, action-led)"
    evidence_excerpt: "No orders found / You haven't placed any orders yet."
    related_surfaces: []
    recommendation: "'No orders yet. Browse chefs to place your first.'"
    depends_on: null

  - finding_id: BV-018
    surface_id: vp-ux-orders-history-empty
    lens: brand-voice
    severity: P2
    issue: "'You have no completed or cancelled orders yet.' — blame-tinged 'you have no' phrasing; also redundant 'orders found / You have no... orders yet' (double-negative wrap)"
    evidence_excerpt: "No orders found / You have no completed or cancelled orders yet."
    related_surfaces: []
    recommendation: "'No order history yet. Completed and cancelled orders appear here.'"
    depends_on: null

  - finding_id: BV-019
    surface_id: web-err-checkout-no-addresses
    lens: brand-voice
    severity: P3
    issue: "'You don't have any saved addresses yet.' — soft blame; STYLE-GUIDE prefers state + action ('No saved addresses. Add one to continue.')"
    evidence_excerpt: "You don't have any saved addresses yet. Add one to continue."
    related_surfaces: []
    recommendation: "'No saved addresses. Add one to continue.'"
    depends_on: null

# =============================================================================
# EMPTY-STATE CUTENESS — banned by lens brief
# =============================================================================

  - finding_id: BV-020
    surface_id: mc-favorites-empty
    lens: brand-voice
    severity: P1
    issue: "'No saved chefs yet / Tap the heart on any chef to save them!' — exclamation mark violates Rule 1 (no exclamations) on a customer empty state; 'them' is grammatical drift ('it' is correct singular antecedent)"
    evidence_excerpt: "No saved chefs yet / Tap the heart on any chef to save them!"
    related_surfaces: []
    recommendation: "'No saved chefs yet. Tap the heart on any chef to save it.' (period, singular pronoun, no exclamation)"
    depends_on: null

  - finding_id: BV-021
    surface_id: mc-orders-empty
    lens: brand-voice
    severity: P1
    issue: "'Browse chefs to place your first order!' — exclamation on customer empty state. Notes column flags 🍽️ emoji-in-copy in same surface; emoji decoration on customer-facing OK per matrix but exclamation isn't"
    evidence_excerpt: "No orders yet / Browse chefs to place your first order!"
    related_surfaces: []
    recommendation: "'No orders yet. Browse chefs to place your first.'"
    depends_on: null

  - finding_id: BV-022
    surface_id: mc-social-empty
    lens: brand-voice
    severity: P3
    issue: "'Chefs will share their latest creations here.' — 'creations' is on the edge of artisanal-banned vocabulary; STYLE-GUIDE bans 'artisanal/handcrafted/curated/foodie'; 'creations' rides the same vibe"
    evidence_excerpt: "Chefs will share their latest creations here."
    related_surfaces: []
    recommendation: "'Chefs will share their latest dishes here.' Plain noun, no faux-artisanal."
    depends_on: null

  - finding_id: BV-023
    surface_id: mv-orders-empty-body
    lens: brand-voice
    severity: P3
    issue: "'New orders will appear here automatically' — 'automatically' is filler; chef persona-matrix says crisp 5-12 words"
    evidence_excerpt: "New orders will appear here automatically"
    related_surfaces: ["vp-ux-orders-live-empty"]
    recommendation: "'New orders appear here.' (4 words)"
    depends_on: null

  - finding_id: BV-024
    surface_id: ap-platsettings-zones-empty
    lens: brand-voice
    severity: P3
    issue: "'No zones yet — delivery is available everywhere until a zone is created.' — em-dash plus passive 'until a zone is created' violates active-voice preference; STYLE-GUIDE §5 'we refund within 7 days' active-voice example applies"
    evidence_excerpt: "No zones yet — delivery is available everywhere until a zone is created."
    related_surfaces: []
    recommendation: "'No zones yet. Without a zone, delivery is available everywhere.'"
    depends_on: null

# =============================================================================
# EMOJI / DECORATIVE IN ERROR-EMPTY CONTEXTS
# =============================================================================

  - finding_id: BV-025
    surface_id: mc-favorites-empty
    lens: brand-voice
    severity: P3
    issue: "Heart emoji in empty-state copy on customer surface — lens brief says customer OK occasional but this is a single-action empty state where photo-forward principle (Rule 3) means the icon does the work, not emoji-in-text"
    evidence_excerpt: "(heart emoji in copy per inventory note)"
    related_surfaces: ["mc-orders-empty", "mc-catering-empty", "mc-social-empty"]
    recommendation: "Remove emoji from copy strings. Use the lucide-react / native icon component above the text instead — keeps copy clean and translation-ready."
    depends_on: null

# =============================================================================
# ANTHROPOMORPHISM / VOICE DRIFT
# =============================================================================

  - finding_id: BV-026
    surface_id: vp-ux-payouts-empty
    lens: brand-voice
    severity: P2
    issue: "'Your payout history will appear here once you receive your first payout.' — 19 words on a chef surface; chef persona-matrix says crisp 5-12 words. 'Once you receive' is also conditional-future heavy"
    evidence_excerpt: "Your payout history will appear here once you receive your first payout."
    related_surfaces: []
    recommendation: "'Payouts appear here after your first payout.' (7 words)"
    depends_on: null

  - finding_id: BV-027
    surface_id: vp-ux-notifs-empty
    lens: brand-voice
    severity: P2
    issue: "'Admin requests for document uploads, profile updates, and review feedback will appear here.' — 14 words listing every category violates chef-tone crispness AND reads like a help-doc entry rather than empty state"
    evidence_excerpt: "Admin requests for document uploads, profile updates, and review feedback will appear here."
    related_surfaces: []
    recommendation: "'No admin requests. They appear here when admin needs something.'"
    depends_on: null

  - finding_id: BV-028
    surface_id: ap-staffdetail-perms-empty
    lens: brand-voice
    severity: P3
    issue: "'No specific permissions data available for this role.' — admin persona allows neutral operator tone but this is bureaucratic ('data available') for what is just 'no permissions set'"
    evidence_excerpt: "No specific permissions data available for this role."
    related_surfaces: []
    recommendation: "'No permissions configured for this role.'"
    depends_on: null

  - finding_id: BV-029
    surface_id: md-emp-007
    lens: brand-voice
    severity: P2
    issue: "Driver lock-screen body 'Fleet management is available for fleet managers only. Contact your administrator to request access.' — 14 words; driver tone says ≤4 telegraphic; 'your administrator' is enterprise-SaaS vocabulary"
    evidence_excerpt: "Fleet management is available for fleet managers only. Contact your administrator to request access."
    related_surfaces: ["md-emp-011"]
    recommendation: "'For fleet managers only. Ask your manager for access.' (10 words, plainer)"
    depends_on: null

# =============================================================================
# BANNED VOCABULARY — STYLE-GUIDE §3
# =============================================================================

  - finding_id: BV-030
    surface_id: ap-providerdetail-dialog-bodies
    lens: brand-voice
    severity: P2
    issue: "'Are you sure you want to delete this delivery provider? This action cannot be undone.' — STYLE-GUIDE §4 modal-subtitle pattern explicitly flags 'Are you sure? This action cannot be undone.' as the ❌ example because it doesn't explain WHAT can't be undone"
    evidence_excerpt: "Are you sure you want to delete this delivery provider? This action cannot be undone."
    related_surfaces: ["ap-providers-dialog-delete-body", "ap-staffdetail-confirm-deactivate", "ap-staffdetail-confirm-reactivate", "mv-menuitem-delete-alert", "mv-menuedit-delete-photo-alert", "mv-more-logout-confirm-body"]
    recommendation: "Replace with consequence-led: 'Delete this provider permanently? Orders already assigned stay; no new ones will route here.' Same pattern across all 7 'Are you sure' surfaces — say what specifically can't be undone, not generic 'cannot be undone'."
    depends_on: null

  - finding_id: BV-031
    surface_id: ap-providerdetail-dialog-bodies
    lens: brand-voice
    severity: P3
    issue: "'Are you sure you want to disable this provider?' / 'Are you sure you want to enable this provider?' — opening with 'Are you sure' is the banned modal pattern from STYLE-GUIDE; also overuse of the same phrase across 7+ confirmations"
    evidence_excerpt: "Are you sure you want to disable this provider? No new deliveries will be assigned to it."
    related_surfaces: ["ap-providers-dialog-disable-body"]
    recommendation: "Lead with the action: 'Disable this provider? No new deliveries will be assigned. Existing deliveries continue.' Modal title carries the question; body explains consequence."
    depends_on: null

  - finding_id: BV-032
    surface_id: api-error-stripe-connect
    lens: brand-voice
    severity: P0
    issue: "'No Stripe account — call /chef/stripe/connect first' leaks an API path to end users; inventory notes flag this. Brand-voice catastrophic — this is plumber-talk to a customer"
    evidence_excerpt: "No Stripe account — call /chef/stripe/connect first"
    related_surfaces: []
    recommendation: "Replace with user-facing: 'Connect Stripe in your payout settings before switching payment methods.' Internal-only error codes stay in API logs."
    depends_on: null

  - finding_id: BV-033
    surface_id: api-error-delivery-already-active
    lens: brand-voice
    severity: P1
    issue: "'no_active_delivery' snake_case slug surfaced as user-facing error string — inventory notes flag this. Snake_case is API-internal vocabulary, never voice."
    evidence_excerpt: "no_active_delivery"
    related_surfaces: []
    recommendation: "Map snake_case to a user-facing message before serialization: 'You don't have an active delivery.' Keep slug as machine-readable code in a separate field."
    depends_on: null

  - finding_id: BV-034
    surface_id: api-error-delivery-online
    lens: brand-voice
    severity: P2
    issue: "'Partner has reached maximum concurrent deliveries' / 'Partner is not verified or active' — 'Partner' is admin-portal vocabulary; driver app refers to driver as 'you'. Same error, two voices, breaks identity per STYLE-GUIDE §3 ('Driver' ✅ in driver-facing, 'Delivery partner' only customer-facing)"
    evidence_excerpt: "Partner has reached maximum concurrent deliveries / Partner is not verified or active"
    related_surfaces: ["dp-err-partner-status", "dp-empty-partner-notfound", "dp-empty-partner-docs", "dp-err-partner-verify", "dp-empty-partners"]
    recommendation: "Driver-facing copy: 'You've hit the max active deliveries.' / 'Your account isn't verified yet.' Admin-facing keeps 'Partner not found' (admin context, neutral operator tone). API should return code + driver-facing message; admin UI builds its own admin-tone string from the code."
    depends_on: null

# =============================================================================
# CAPITALIZATION DRIFT — BUTTON / CTA TITLE CASE
# =============================================================================

  - finding_id: BV-035
    surface_id: vp-ux-menu-error-state
    lens: brand-voice
    severity: P3
    issue: "'Try Again' button label uses Title Case across vendor-portal (menu, dashboard, analytics, earnings, reviews) — STYLE-GUIDE §4 button format is sentence case + ≤3 words verb-first"
    evidence_excerpt: "Try Again"
    related_surfaces: ["vp-err-boundary-fallback", "mv-dash-retry-cta"]
    recommendation: "'Try again' (sentence case). Note mobile-vendor already uses 'Retry' (1 word) — even better. Pick one: 'Retry' (preferred for crispness)."
    depends_on: null

  - finding_id: BV-036
    surface_id: web-ux-chefdetail-not-found
    lens: brand-voice
    severity: P3
    issue: "'Browse Chefs' CTA in Title Case on web 'Chef not found' empty state; same in 'View All Orders'"
    evidence_excerpt: "Chef not found / Browse Chefs"
    related_surfaces: ["web-err-orderdetail-not-found", "web-err-orders-empty"]
    recommendation: "'Browse chefs' / 'View all orders' (sentence case)."
    depends_on: null

  - finding_id: BV-037
    surface_id: vp-onb-banner-rejected-title
    lens: brand-voice
    severity: P3
    issue: "Onboarding banner titles in Title Case: 'Application Rejected' / 'More Information Needed' — chef-facing operational copy; STYLE-GUIDE sentence-case applies"
    evidence_excerpt: "Application Rejected / More Information Needed"
    related_surfaces: ["vp-onb-banner-info-requested-title"]
    recommendation: "'Application rejected' / 'More information needed' — sentence case."
    depends_on: null

# =============================================================================
# PUNCTUATION CONSISTENCY
# =============================================================================

  - finding_id: BV-038
    surface_id: mc-cartsheet-empty
    lens: brand-voice
    severity: P3
    issue: "Period inconsistency: 'Your cart is empty' (no period, CartSheet) vs 'Your cart is empty.' (with period, checkout list) vs full sentence on web. Empty-state titles inconsistent on period usage across codebase"
    evidence_excerpt: "Your cart is empty"
    related_surfaces: ["mc-checkout-empty-cart", "web-ux-cart-empty"]
    recommendation: "Adopt rule: empty-state titles end with period when they read as a sentence ('Your cart is empty.'); fragment titles don't ('No orders yet'). Document in STYLE-GUIDE §4. Then sweep."
    depends_on: null

  - finding_id: BV-039
    surface_id: dp-empty-partners
    lens: brand-voice
    severity: P3
    issue: "Trailing-period drift in delivery-portal: 'No partners found matching your criteria.' (period) vs 'No partners found.' (period) vs 'No Available Deliveries' (no period) — same surface family"
    evidence_excerpt: "No partners found matching your criteria."
    related_surfaces: ["dp-empty-partner-notfound", "dp-empty-partner-docs", "dp-empty-pending-invites", "md-emp-005", "md-emp-008", "md-emp-004"]
    recommendation: "Apply rule from BV-038. Driver-app titles fragmentary (no period), bodies with sentences (period)."
    depends_on: null

  - finding_id: BV-040
    surface_id: api-error-phone-duplicate
    lens: brand-voice
    severity: P3
    issue: "API errors are wildly inconsistent on trailing periods: 'This phone number is already registered with another account.' (period) vs ~140 others without period ('Invalid email or password', 'Chef not found'). Inventory notes flag this"
    evidence_excerpt: "This phone number is already registered with another account."
    related_surfaces: ["api-error-validation-required", "api-error-validation-not-found", "api-error-validation-invalid-id"]
    recommendation: "Adopt rule: API error 'message' field ALWAYS ends with period (writing is complete-sentence). The API isn't the surface; the frontend may strip or keep. Document in STYLE-GUIDE."
    depends_on: null

# =============================================================================
# AI-SLOP / GENERIC PLACEHOLDER COPY
# =============================================================================

  - finding_id: BV-041
    surface_id: ap-analytics-coming-soon
    lens: brand-voice
    severity: P0
    issue: "'Chart coming soon' shipped to production admin surface — inventory notes flag 'placeholder; surfaces unfinished work to users'. This violates Rule 5 (restraint) AND lens brief 'Coming soon features in production'"
    evidence_excerpt: "Chart coming soon"
    related_surfaces: []
    recommendation: "Either ship the chart or hide the section. Never ship 'coming soon' to production. If it must stay visible during a phased rollout, use 'Analytics in beta — full charts arriving next release' with a specific commitment."
    depends_on: null

  - finding_id: BV-042
    surface_id: ap-dashboard-recent-empty-body
    lens: brand-voice
    severity: P3
    issue: "'Platform events will appear here.' — generic SaaS-dashboard placeholder (lens brief 'Generic SaaS dashboard'). No information about what counts as a platform event"
    evidence_excerpt: "Platform events will appear here."
    related_surfaces: []
    recommendation: "'No recent activity. Approvals, suspensions, and refunds appear here.' (specific, useful)"
    depends_on: null

  - finding_id: BV-043
    surface_id: ap-secsettings-apikey-empty
    lens: brand-voice
    severity: P3
    issue: "'No keys yet.' — under-informative; admin tone allows precision but 'No keys yet' could be 'No API keys. Generate one to grant programmatic access.'"
    evidence_excerpt: "No keys yet."
    related_surfaces: []
    recommendation: "'No API keys yet. Generate one to grant programmatic access.'"
    depends_on: null

# =============================================================================
# PERSONA-TONE VIOLATIONS
# =============================================================================

  - finding_id: BV-044
    surface_id: mc-checkout-errors
    lens: brand-voice
    severity: P1
    issue: "'Payment confirmation timed out. Check your order history to confirm status.' — customer-facing on a money-sensitive surface; 17 words for what should be ~10. STYLE-GUIDE customer-matrix says 10-18 words conversational, but this is operational not conversational"
    evidence_excerpt: "Payment confirmation timed out. Check your order history to confirm status."
    related_surfaces: []
    recommendation: "'Payment is still processing. We'll update your order when it confirms.' (warmer + 11 words + no blame on customer)"
    depends_on: null

  - finding_id: BV-045
    surface_id: api-error-payment-config
    lens: brand-voice
    severity: P1
    issue: "'Stripe gateway not configured by platform admin' — leaks internal config state to customer; 'platform admin' is internal vocabulary. Customer doesn't know who that is"
    evidence_excerpt: "Stripe gateway not configured by platform admin"
    related_surfaces: []
    recommendation: "Customer-facing: 'Card payments aren't available right now. Try a different method or contact support.' Log the internal cause separately."
    depends_on: null

  - finding_id: BV-046
    surface_id: api-error-auth-suspended
    lens: brand-voice
    severity: P2
    issue: "'Account is suspended' — no reason, no contact-support guidance. Inventory notes flag this. Suspending a user is a high-trust moment; copy should support recovery path"
    evidence_excerpt: "Account is suspended"
    related_surfaces: []
    recommendation: "'Your account is suspended. Contact support@homechef.in to appeal.' (with actual support address from contact config)"
    depends_on: null

  - finding_id: BV-047
    surface_id: api-error-order-cancel-stage
    lens: brand-voice
    severity: P1
    issue: "'Order cannot be cancelled at this stage' — no refund implication, no explanation of WHICH stage allows cancellation. Customer money is involved"
    evidence_excerpt: "Order cannot be cancelled at this stage"
    related_surfaces: []
    recommendation: "'Your chef has started cooking — cancellation closed. Contact the chef via chat to discuss.' (specific + actionable). Backend should return both code and the stage so frontend can craft per-stage copy."
    depends_on: null

  - finding_id: BV-048
    surface_id: api-error-chef-not-accepting
    lens: brand-voice
    severity: P2
    issue: "'Chef is not accepting orders' — no alternative ('try another chef'); customer surface should always offer a next action per Rule 5 (no urgency tricks but always offer the door)"
    evidence_excerpt: "Chef is not accepting orders"
    related_surfaces: []
    recommendation: "'This chef isn't accepting orders right now. Browse other chefs nearby.' (with a link back to browse)"
    depends_on: null

# =============================================================================
# INCONSISTENT SAME-CONCEPT MESSAGING
# =============================================================================

  - finding_id: BV-049
    surface_id: api-error-auth-unauthorized
    lens: brand-voice
    severity: P0
    issue: "Same auth-failure concept across API: 'Unauthorized' / 'unauthorized' / 'Authentication required' / 'Token required' — four variants. Casing inconsistent within one variant"
    evidence_excerpt: "Unauthorized / unauthorized / Authentication required / Token required"
    related_surfaces: []
    recommendation: "Pick one user-facing string: 'Sign in to continue.' (or 'Session expired. Sign in again.' if it's a refresh). Internal HTTP codes stay 401. Audit handlers/* and consolidate."
    depends_on: null

  - finding_id: BV-050
    surface_id: api-error-auth-not-verified
    lens: brand-voice
    severity: P2
    issue: "'Account not verified' vs 'Account not verified yet' — two variants in same file (handlers/auth.go per inventory). The 'yet' variant is friendlier"
    evidence_excerpt: "Account not verified / Account not verified yet"
    related_surfaces: []
    recommendation: "Keep 'yet' + add action: 'Your account isn't verified yet. Check your email for the verification link.'"
    depends_on: null

  - finding_id: BV-051
    surface_id: api-error-auth-current-pw
    lens: brand-voice
    severity: P2
    issue: "'Current password is incorrect' vs 'Password is incorrect' — two variants of same concept"
    evidence_excerpt: "Current password is incorrect / Password is incorrect"
    related_surfaces: []
    recommendation: "Settle on context: 'Current password is incorrect.' for change-password flow; 'Incorrect email or password.' for sign-in (deliberately vague for security). Document both as canonical."
    depends_on: null

  - finding_id: BV-052
    surface_id: api-error-auth-reset-expired
    lens: brand-voice
    severity: P2
    issue: "'Reset token has expired' / 'Invalid or expired reset token' / 'Invalid or expired enrollment token' — three variants of token-expiry concept"
    evidence_excerpt: "Reset token has expired / Invalid or expired reset token / Invalid or expired enrollment token"
    related_surfaces: []
    recommendation: "Unified: 'This link has expired. Request a new one.' Customers don't think in 'tokens'."
    depends_on: null

  - finding_id: BV-053
    surface_id: api-error-upload-size
    lens: brand-voice
    severity: P0
    issue: "'10MB' vs '5 MB' vs '5MB' — three different formats of the same unit on the same product. Inventory notes flag this. Customer-visible inconsistency on file uploads"
    evidence_excerpt: "File too large (max 10MB) / File too large. Maximum 5 MB. / Profile photo too large (max 5MB)"
    related_surfaces: ["api-error-upload-type", "vp-ux-menu-form-image-errors", "dp-err-step3-file-size"]
    recommendation: "Standardize: '5 MB' (space, capital MB) everywhere. Standardize wrapper: 'File too large. Max 5 MB.' (period, sentence case, no parenthesis)."
    depends_on: null

  - finding_id: BV-054
    surface_id: api-error-upload-type
    lens: brand-voice
    severity: P1
    issue: "5 variants of 'Invalid file type' with slightly different allowed lists ('JPEG, PNG, PDF' vs 'JPEG, PNG, WebP, PDF' vs 'JPEG, PNG, WebP' vs 'JPEG, PNG, WebP' vs 'Profile photo must be JPEG or PNG'). Inventory flags this"
    evidence_excerpt: "Invalid file type. Allowed: JPEG, PNG, PDF."
    related_surfaces: []
    recommendation: "Standardize structure: 'Invalid file type. Allowed: {list}.' — generated from server-side allowlist constant per endpoint. Never hand-roll the variant list."
    depends_on: null

  - finding_id: BV-055
    surface_id: api-error-upload-count
    lens: brand-voice
    severity: P2
    issue: "Upload-count errors inconsistent on remediation: 'Maximum 5 kitchen photos allowed. Remove one before adding another.' (good — adds 'Remove one') vs 'Maximum 3 images per review' (no remediation)"
    evidence_excerpt: "Maximum 3 images per review / Maximum 5 kitchen photos allowed. Remove one before adding another."
    related_surfaces: []
    recommendation: "Apply remediation to all: 'Maximum 3 images per review. Remove one before adding another.' STYLE-GUIDE §4 error pattern: What happened → What to do."
    depends_on: null

  - finding_id: BV-056
    surface_id: web-err-favorite-max-limit
    lens: brand-voice
    severity: P3
    issue: "'You can save up to 7 favorite chefs. Remove one first.' — good remediation pattern (matches BV-055 recommendation). Flag as positive baseline."
    evidence_excerpt: "You can save up to 7 favorite chefs. Remove one first."
    related_surfaces: []
    recommendation: "Keep. Use as template across upload-count and any limit error: '{What's the rule}. {What to do.}'"
    depends_on: null

# =============================================================================
# ALL CAPS / SHOUTY
# =============================================================================

  - finding_id: BV-057
    surface_id: vp-onb-docs-required-title
    lens: brand-voice
    severity: P3
    issue: "Mobile-vendor 'Documents Required' / 'Permission Required' / 'Terms Required' titles read shouty (Title Case + commanding noun); chef-tone matrix says crisp 5-12 words operational, not exclamatory"
    evidence_excerpt: "Documents Required / Permission Required / Terms Required"
    related_surfaces: ["mv-onb-docs-perm-title", "mv-onb-policies-terms-required-title"]
    recommendation: "See BV-015 (already covered). Cross-reference."
    depends_on: BV-015

# =============================================================================
# REDUNDANT / OVERLONG CHEF/DRIVER COPY
# =============================================================================

  - finding_id: BV-058
    surface_id: vp-ux-menu-empty
    lens: brand-voice
    severity: P3
    issue: "'Start building your menu by adding your first dish.' — 9 words but starts with 'Start' (filler verb), 'building your menu' is over-described"
    evidence_excerpt: "Start building your menu by adding your first dish."
    related_surfaces: []
    recommendation: "'Add your first dish to get started.' (7 words) — or just the STYLE-GUIDE example 'No menu items. Add your first dish.' (covered in BV-009)."
    depends_on: BV-009

  - finding_id: BV-059
    surface_id: mv-reviews-empty
    lens: brand-voice
    severity: P3
    issue: "'No reviews yet. Your first review will appear here.' — chef-facing, 'your first review will appear here' is redundant with 'No reviews yet' (both mean: empty list)"
    evidence_excerpt: "No reviews yet. Your first review will appear here."
    related_surfaces: ["vp-ux-reviews-empty"]
    recommendation: "Just 'No reviews yet.' (vendor-portal version is already correct; align mobile-vendor)."
    depends_on: null

  - finding_id: BV-060
    surface_id: dp-empty-staff
    lens: brand-voice
    severity: P3
    issue: "'No staff members yet. Send an invitation to get started.' — admin surface in driver portal; 'to get started' is filler. Admin matrix: precise, no fluff"
    evidence_excerpt: "No staff members yet. Send an invitation to get started."
    related_surfaces: []
    recommendation: "'No staff yet. Send an invitation.' (or align with the admin-portal version 'No staff members found')."
    depends_on: null

# =============================================================================
# SAME-NAME / DIFFERENT-WORDS CONCEPT DRIFT
# =============================================================================

  - finding_id: BV-061
    surface_id: vp-onb-locked-conflict-toast
    lens: brand-voice
    severity: P2
    issue: "Two variants of 'kitchen already submitted' state: 'Your kitchen is already under review or live.' vs 'Your kitchen is already submitted.' — same concept, two toasts"
    evidence_excerpt: "Your kitchen is already under review or live. / Your kitchen is already submitted."
    related_surfaces: ["vp-onb-locked-info-toast"]
    recommendation: "Pick state-precise variant: 'Your kitchen is already {status}.' where status comes from the API (under review / approved / live). Avoid the catchall 'submitted'."
    depends_on: null

  - finding_id: BV-062
    surface_id: api-error-validation-required
    lens: brand-voice
    severity: P0
    issue: "18 variants of 'X is required' across API handlers; inventory notes 'Inconsistent casing on field names (chefId vs country)'. Customer surfaces will see camelCase and human-readable mixed"
    evidence_excerpt: "Email is required / chefId is required / currencyCode is required"
    related_surfaces: []
    recommendation: "API: always use human-readable label, never camelCase: 'Chef is required.' not 'chefId is required.'. Audit all handlers and update binding error formatters. Document in STYLE-GUIDE §3 'Field names in user-facing strings'."
    depends_on: null

  - finding_id: BV-063
    surface_id: api-error-validation-invalid-id
    lens: brand-voice
    severity: P0
    issue: "~40 variants of 'Invalid X' / 'Invalid X ID'; inventory flags this. Customer sees 'Invalid menuItem' — that's a developer-facing string leaked"
    evidence_excerpt: "Invalid menuItem ID / Invalid OAuth provider / Invalid request body"
    related_surfaces: []
    recommendation: "Distinguish: (1) field-format errors stay specific in human label ('Invalid menu item.'); (2) framework errors ('Invalid request body') should never surface — wrap as 'Something looks off with your input. Try again.'"
    depends_on: null

  - finding_id: BV-064
    surface_id: api-error-generic-failed
    lens: brand-voice
    severity: P0
    issue: "100+ 'Failed to X' variants across API; inventory: 'Should generally not surface to end users without translation'. Brand-voice catastrophe — these are operational/dev strings"
    evidence_excerpt: "Failed to fetch chefs / Failed to create order / Failed to upload file"
    related_surfaces: []
    recommendation: "Frontend should NEVER surface 'Failed to {verb}' verbatim from API. Map at the frontend layer: 'Couldn't load chefs. Try again.' / 'We couldn't place your order. Try again or contact support.' API string can stay technical (operational logs); the surfaced message is curated."
    depends_on: null

  - finding_id: BV-065
    surface_id: api-error-quotes-catering
    lens: brand-voice
    severity: P3
    issue: "Catering quote errors generally well-phrased ('This request is no longer accepting quotes' / 'Quote deadline has passed') — positive baseline; flag for inclusion in STYLE-GUIDE examples"
    evidence_excerpt: "This request is no longer accepting quotes"
    related_surfaces: []
    recommendation: "Use as template for state-based errors elsewhere. Pattern: 'This {thing} is no longer {state}.' or 'The {window} has passed.'"
    depends_on: null

# =============================================================================
# PARITY DRIFT — same concept, mobile vs web
# =============================================================================

  - finding_id: BV-066
    surface_id: web-err-chefdetail-empty-category
    lens: brand-voice
    severity: P3
    issue: "'No items in this category' — identical between web and mobile-customer (positive baseline). Flag for reuse pattern."
    evidence_excerpt: "No items in this category"
    related_surfaces: ["mc-chef-empty-category"]
    recommendation: "Keep as canonical shared string. Same word count, no period (fragment)."
    depends_on: null

  - finding_id: BV-067
    surface_id: web-err-browse-load-failed
    lens: brand-voice
    severity: P2
    issue: "'Failed to load chefs. Please try again.' (web) vs 'Failed to load chef details. Please try again.' (mobile-customer chef detail) vs 'Failed to load menu' (mobile-vendor menu — no period, no remediation). Drift across same family"
    evidence_excerpt: "Failed to load chefs. Please try again."
    related_surfaces: ["mc-chef-error-load", "mv-menu-load-error", "mv-dash-fail-error", "mv-analytics-load-error", "mv-earnings-load-error", "md-err-001"]
    recommendation: "Pattern: 'Couldn't load {thing}. Try again.' (sentence case, 'couldn't' over 'failed', period, no 'please'). Apply across all 10+ load-failure surfaces."
    depends_on: null

  - finding_id: BV-068
    surface_id: mc-checkout-errors
    lens: brand-voice
    severity: P2
    issue: "'Order creation failed. Please try again.' vs API-side 'Failed to create order' — same concept, two phrasings (passive vs active 'Failed'). Customer should see warmer variant"
    evidence_excerpt: "Order creation failed. Please try again."
    related_surfaces: []
    recommendation: "'We couldn't place your order. Try again or contact support.' (warmer + actionable). Drop 'Please' — STYLE-GUIDE pattern is no 'please' in errors (consequence/action, not pleading)."
    depends_on: null

  - finding_id: BV-069
    surface_id: vp-ux-payouts-empty
    lens: brand-voice
    severity: P2
    issue: "'Unable to load X / Please try again later.' pattern (vendor-portal earnings/payouts) uses 'Unable to' which is unnecessarily formal; 'try again later' is also weaker than 'Try again'"
    evidence_excerpt: "Unable to load earnings / Please try again later."
    related_surfaces: ["vp-ux-earnings-error"]
    recommendation: "'Couldn't load earnings. Try again.' Drop 'later' (no concrete time = filler)."
    depends_on: null

  - finding_id: BV-070
    surface_id: vp-onb-save-failure-toast
    lens: brand-voice
    severity: P3
    issue: "'Failed to save your details. Please try again.' (VP onboarding) vs 'Failed to save. Please try again.' (DP onboarding) — same concept, drift on 'your details'"
    evidence_excerpt: "Failed to save your details. Please try again."
    related_surfaces: ["dp-err-save-retry"]
    recommendation: "'Couldn't save. Try again.' (shortest, sentence case, no 'please')."
    depends_on: null

# =============================================================================
# MISC — punctuation, format, hyphens
# =============================================================================

  - finding_id: BV-071
    surface_id: vp-onb-banner-rejected-body
    lens: brand-voice
    severity: P3
    issue: "'Your previous application was not approved.' — passive voice. STYLE-GUIDE §5 'Active voice' applies even to non-legal copy"
    evidence_excerpt: "Your previous application was not approved."
    related_surfaces: []
    recommendation: "'We didn't approve your previous application. Review the feedback below and resubmit.' (active, single sentence)."
    depends_on: null

  - finding_id: BV-072
    surface_id: vp-ux-menu-form-image-errors
    lens: brand-voice
    severity: P3
    issue: "'{name}: Invalid type. Allowed: JPEG, PNG, WebP.' format prepends filename with colon — leading filename is OK but the structure '{name}: X. Y.' is jargony"
    evidence_excerpt: "{name}: Invalid type. Allowed: JPEG, PNG, WebP."
    related_surfaces: []
    recommendation: "'{name} — invalid file type. Allowed: JPEG, PNG, WebP.' (em-dash, lowercase 'invalid', no double-colon)."
    depends_on: null

  - finding_id: BV-073
    surface_id: dp-err-step3-file-size
    lens: brand-voice
    severity: P3
    issue: "'File too large. Max {n}MB for {label}' — no space in '{n}MB', conflicts with BV-053 standardization '5 MB'"
    evidence_excerpt: "File too large. Max {n}MB for {label}"
    related_surfaces: []
    recommendation: "'File too large. Max {n} MB for {label}.' (space + period)."
    depends_on: BV-053

  - finding_id: BV-074
    surface_id: api-error-search-min
    lens: brand-voice
    severity: P3
    issue: "'Search query must be at least 2 characters' — 'query' is dev jargon for customer; STYLE-GUIDE plain-English rule applies"
    evidence_excerpt: "Search query must be at least 2 characters"
    related_surfaces: []
    recommendation: "'Search must be at least 2 characters.' (period; 'search' as noun)."
    depends_on: null

  - finding_id: BV-075
    surface_id: api-error-validation-numeric
    lens: brand-voice
    severity: P3
    issue: "'overallRating must be between 1 and 5' — camelCase field name in user-visible message"
    evidence_excerpt: "overallRating must be between 1 and 5"
    related_surfaces: ["api-error-validation-required", "api-error-validation-invalid-id"]
    recommendation: "'Rating must be between 1 and 5.' Audit all numeric-bounds errors for camelCase. See BV-062."
    depends_on: BV-062

# =============================================================================
# POSITIVE BASELINES (flag for STYLE-GUIDE examples)
# =============================================================================

  - finding_id: BV-076
    surface_id: api-error-chef-kitchen-name
    lens: brand-voice
    severity: P3
    issue: "'A kitchen with this name already exists. Please choose a different name.' — friendly, complete, sentence-case. Flag as positive baseline. (Drop 'Please' per BV-068 if applying consistently.)"
    evidence_excerpt: "A kitchen with this name already exists. Please choose a different name."
    related_surfaces: []
    recommendation: "Keep. Use as template. Optionally tighten: 'A kitchen with this name already exists. Choose another.'"
    depends_on: null

  - finding_id: BV-077
    surface_id: api-error-chat-availability
    lens: brand-voice
    severity: P3
    issue: "'Please wait a moment before sending another message' — friendly rate-limit; positive baseline"
    evidence_excerpt: "Please wait a moment before sending another message"
    related_surfaces: []
    recommendation: "Keep. (Adds period: 'Please wait a moment before sending another message.')"
    depends_on: null

  - finding_id: BV-078
    surface_id: api-error-favorites
    lens: brand-voice
    severity: P3
    issue: "'Chef is already in your favorites' — clean, action-implied. Positive baseline"
    evidence_excerpt: "Chef is already in your favorites"
    related_surfaces: []
    recommendation: "Keep. Add period."
    depends_on: null

# =============================================================================
# SUMMARY METADATA
# =============================================================================
# total_findings: 78
# P0: 13 (cross-app drift on canonical errors, AI/dev-string leaks)
# P1: 14 (entry-surface drift, persona violations, banned exclamations on customer entry, all-caps headlines)
# P2: 24 (secondary tone drift, blame language, anthropomorphism, missing remediation)
# P3: 27 (punctuation/capitalization, redundant words, positive baselines)
#
# top_cross_app_drift:
#   1. "Something went wrong. Please try again." — 6 surfaces across 6 apps (BV-001, BV-013)
#   2. Error-boundary fallback copy — 3 distinct variants across 4 apps (BV-003)
#   3. Offline banner — 4 variants across 4 apps (BV-004)
#   4. "No pending orders" empty state — 2 variants between vendor-portal and mobile-vendor (BV-005)
#   5. Cart-empty copy — 3 word counts across web/mobile/cart-sheet (BV-006)
#   6. Title Case CTAs ("Try Again", "Browse Chefs", "View All Orders", "Add Your First Item") — banned by STYLE-GUIDE (BV-008, BV-009, BV-035, BV-036)
#   7. "Are you sure... cannot be undone" — STYLE-GUIDE-banned anti-pattern in 7+ confirmation surfaces (BV-030, BV-031)
#   8. Driver app Title Case headings on glanceable lock screens (BV-010, BV-011)
#   9. API "Failed to X" + "X is required" + "Invalid X" — 150+ dev-string variants leak to UI (BV-062, BV-063, BV-064)
#   10. File-size format "5 MB" vs "5MB" vs "10MB" — customer-visible unit inconsistency (BV-053)
#   11. "Partner" vs "you" terminology — same error, two voices across driver-portal + API (BV-034)
#   12. Trailing-period drift across empty-state titles (BV-038, BV-039, BV-040)

```
