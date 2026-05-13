# Findings — Transactional

Category: transactional
Lenses applied: technical-writer, legal, business-analyst, brand-voice
Inventory slice size: 154 surfaces
Total findings: 441

## Severity summary

| Severity | TW | Legal | BA | BV | Total |
|---|---|---|---|---|---|
| P0 | 19 | 19 | 9 | 9 | 56 |
| P1 | 10 | 55 | 16 | 17 | 98 |
| P2 | 91 | 52 | 20 | 34 | 197 |
| P3 | 40 | 28 | 12 | 10 | 90 |

## Cross-lens consensus surfaces (flagged by multiple lenses)

Surfaces flagged by all 4 lenses (highest priority — every lens agreed there's a problem):

- **`api-push-user-welcome`** — flagged by TW(P0), Legal(P0, P1), BA(P0), BV(P0) — first-touch welcome push: brand-name contradiction (HomeChef vs Fe3dr vs Fe3dr HomeChef), GIP/consent gap, duplicate-channel waste, and "amazing/talented" hype banned by style guide
- **`api-email-welcome`** — flagged by TW(P1, P2), Legal(P0, P1), BA(P1), BV(P1, P2) — welcome email leaks marketing hype, fails DPDP §6 consent capture and CAN-SPAM/IT Rules unsubscribe disclosure, and contradicts the push channel's brand string
- **`api-email-account-reminder`** — flagged by TW(P0, P2), Legal(P1), BA(P1), BV(P0) — unbranded inline-HTML account-existence email bypasses emailBase wrapper and triple-names the product ("Fe3dr HomeChef")
- **`api-email-order-confirm`** — flagged by TW(P1, P2, P3), Legal(P0, P1), BA(P0), BV(P1) — order confirmation missing every GST tax-invoice element (GSTIN, HSN/SAC, rate breakup, place-of-supply), no refund/cancellation policy, currency rendered as raw `₹%.2f`
- **`api-email-order-status-cancelled`** — flagged by TW(P0), Legal(P0), BA(P0), BV(P1) — order-cancelled email omits refund timeline (RBI PA MD requires explicit refund SLA), no reason, no support route — money-bearing
- **`api-email-delivery-assigned`** — flagged by TW(P0, P2), Legal(P0, P1), BA(P0), BV(P1) — driver-copy template ("You've been assigned order #X") is routed to customers; leaks pickup address (DPDP §8(5)), wrong audience
- **`api-email-chef-new-order`** — flagged by TW(P0, P1, P2), Legal(P1, P2), BA(P1), BV(P1) — chef new-order email mixes customer-facing hype with operator-critical detail; missing chef-side cancel/decline path and order timer
- **`api-email-chef-verified`** — flagged by TW(P2), Legal(P2), BA(P1), BV(P1) — chef verification success email lacks FSSAI registration-number capture/display and skips next-step CTA to publish menu
- **`api-email-support-created`** — flagged by TW(P0, P2), Legal(P1), BA(P1), BV(P0) — support-ticket-created email unbranded inline HTML, no Grievance Officer disclosure (IT Rules 2021 §3(2)), no SLA
- **`api-email-support-update`** — flagged by TW(P0, P2), Legal(P1), BA(P1), BV(P1) — same unbranded inline-HTML pattern, no Grievance Officer escalation path, no resolution-time commitment
- **`api-success-message-generic`** — flagged by TW(P0, P2, P3), Legal(P1, P2), BA(P2), BV(P1) — generic "Operation successful" copy returned across 30+ API success paths; meaningless to user, leaks no recovery information
- **`api-inapp-approval-approved-chef`** — flagged by TW(P1), Legal(P3), BA(P0), BV(P0) — chef-approved in-app message duplicates the approval push without next-step CTA (publish menu); "Approved" status framing is enum-leak
- **`api-inapp-approval-rejected-chef`** — flagged by TW(P0), Legal(P1), BA(P0), BV(P1) — chef rejection notice missing due-process language (reason, appeal pathway, Grievance Officer), gig-classification risk
- **`api-inapp-chef-responded`** — flagged by TW(P3), Legal(P1), BA(P2), BV(P1) — catering response in-app leaks raw enum, missing accept/decline next-step
- **`api-inapp-order-status-customer`** — flagged by TW(P2), Legal(P2), BA(P2), BV(P2) — customer order-status in-app messages drift across channels (push body vs in-app vs email subject), no refund linkage on cancellation
- **`api-push-chef-new-order-actionable`** — flagged by TW(P1, P2, P3), Legal(P2), BA(P0), BV(P0) — chef new-order push fires twice (duplicate NATS handlers), emoji + exclamation budget blown for vendor persona, no accept/decline action buttons
- **`api-push-delivery-assigned-customer`** — flagged by TW(P2), Legal(P2), BA(P1), BV(P2) — customer push on driver-assigned leaks driver name without driver consent (DPDP §6) and uses inconsistent wording vs in-app/email siblings
- **`api-push-order-cancelled-both`** — flagged by TW(P0), Legal(P1), BA(P1), BV(P0) — single cancellation push fires to both customer and chef with same template; no refund timeline for customer, no relist CTA for chef
- **`web-tx-checkout-payment-toasts`** — flagged by TW(P1), Legal(P1), BA(P1), BV(P1) — checkout payment toasts ("Payment failed" / "Payment successful") missing reason codes, retry path, RBI PA reconciliation reference
- **`web-tx-orderdetail-toasts`** — flagged by TW(P2), Legal(P0, P1), BA(P3), BV(P2) — order-detail action toasts (cancel, reorder) on money-bearing surface lack refund acknowledgement and DPDP-aligned data-retention messaging
- **`web-tx-profile-2fa`** — flagged by TW(P2), Legal(P1, P2), BA(P3), BV(P2) — 2FA enable/disable toasts skip recovery-code disclosure and step-up auth confirmation (RBI/DPDP risk on account takeover)
- **`web-tx-catering-quotes`** — flagged by TW(P2), Legal(P1), BA(P2), BV(P2) — catering-quote toasts lack quote-validity period, GST disclosure, and trader-disclosure (E-Commerce Rules 2020 §5)
- **`web-tx-catering-request-submit`** — flagged by TW(P2), Legal(P2), BA(P1), BV(P2) — catering-request submission misses next-step expectation ("we'll respond in X hours"), no consent capture for data processing
- **`web-tx-delivery-actions`** — flagged by TW(P2), Legal(P2), BA(P2), BV(P1) — chef-side delivery action toasts ("Assigned to driver", "Marked picked up") use enum strings as user copy
- **`vp-onb-policies-ready-banner`** — flagged by TW(P0, P2), Legal(P2), BA(P2), BV(P2) — chef onboarding policies banner uses unverified trust claim ("Your kitchen meets policy"); FSSAI policy linkage missing
- **`vp-onb-submit-success`** — flagged by TW(P0), Legal(P3), BA(P1), BV(P2) — chef onboarding submit-success conflates submission with approval; no SLA on review time, no KYC-refresh cadence disclosure
- **`dp-tx-toast-accept-success`** — flagged by TW(P2), Legal(P2), BA(P2), BV(P2) — driver accept-delivery toast leaks customer address before pickup is confirmed; persona-density mismatch (too long for glanceable driver UI)
- **`dp-tx-toast-step5-submitted`** — flagged by TW(P2), Legal(P3), BA(P2), BV(P2) — driver onboarding submit toast lacks MV Act 1988 license-verification cadence and conflates submission with activation
- **`mc-menuitem-cross-chef`** — flagged by TW(P2), Legal(P3), BA(P2), BV(P2) — cross-chef menu-item add-to-cart toast confuses customer; no resolution path; same surface in 4 places
- **`mc-profile-logout-confirm`** — flagged by TW(P0), Legal(P3), BA(P1), BV(P1) — logout confirmation dialog uses banned "Are you sure?" anti-pattern; "Log out" vs "Logout" vs "Sign out" inconsistency on same screen
- **`ap-secsettings-toasts`** — flagged by TW(P2), Legal(P1), BA(P3), BV(P2) — admin security-settings toasts (API key rotation, password reset) lack audit-trail acknowledgement (DPDP §8 audit log)
- **`ap-staff-invite-success-title`** — flagged by TW(P2), Legal(P3), BA(P3), BV(P2) — admin staff-invite success title uses Title Case while sibling button uses sentence case
- **`md-trx-006`** — flagged by TW(P0), Legal(P3), BA(P1), BV(P1) — mobile delivery transactional surface with brand-name contradiction
- **`md-trx-010`** — flagged by TW(P3), Legal(P0), BA(P2), BV(P0) — mobile delivery transactional surface with DPDP-risk PII exposure
- **`md-trx-011`** — flagged by TW(P2), Legal(P3), BA(P2), BV(P2) — mobile delivery transactional drift across siblings
- **`md-trx-013`** — flagged by TW(P2), Legal(P3), BA(P2), BV(P2) — mobile delivery transactional drift across siblings
- **`mv-reviewdetail-success-title`** — flagged by TW(P3), Legal(P3), BA(P3), BV(P3) — mobile-vendor review-detail success title style drift (all P3 — pattern fix)

Surfaces flagged by 3 lenses (also high-priority rewrites):

- **`md-trx-002`** — flagged by TW(P0), Legal(P0), BV(P0) — mobile delivery transactional copy P0 across three lenses
- **`api-push-order-update-customer-deeplink`** — flagged by TW(P0, P2), Legal(P2), BA(P0) — customer order-update push with broken deeplink + missing fallback
- **`api-email-order-status-delivered`** — flagged by TW(P2), Legal(P1), BV(P2) — delivered-status email missing rating CTA and GST invoice attachment
- **`api-email-password-reset`** — flagged by TW(P2), Legal(P1, P2), BV(P2) — password reset email lacks geographic context (IP/device hint) and expiry-time clarity (IT Rules 2021)
- **`api-email-staff-invite`** — flagged by TW(P2, P3), Legal(P1), BV(P2) — staff-invite email missing tenant context and role-scope disclosure
- **`api-email-verify`** — flagged by TW(P2, P3), Legal(P2), BV(P2) — email-verify body unbranded and missing code-expiry / single-use disclosure
- **`api-inapp-approval-created-admin`** — flagged by TW(P3), Legal(P2), BV(P2) — admin in-app "approval-created" missing SLA/queue-position
- **`api-inapp-chef-new-order`** — flagged by TW(P2), BA(P0), BV(P1) — chef in-app new-order missing accept/decline timer
- **`api-inapp-chef-verified`** — flagged by TW(P2), Legal(P2), BV(P2) — chef-verified in-app lacks publish-menu CTA
- **`api-inapp-delivery-pickedup-customer`** — flagged by TW(P2), Legal(P2), BV(P2) — pickup-confirmed in-app drifts vs push channel
- **`api-inapp-driver-onboarding-admins`** — flagged by TW(P2), Legal(P1), BV(P3) — admin in-app driver-onboarding notification lacks KYC summary
- **`api-inapp-staff-invite`** — flagged by TW(P3), Legal(P1), BV(P2) — staff-invite in-app misses tenant/role context
- **`api-push-approval-approved-chef`** — flagged by TW(P1), BA(P2), BV(P1) — chef-approved push duplicates in-app without distinct value
- **`api-push-delivery-assigned-driver`** — flagged by TW(P2), Legal(P1), BV(P2) — driver-assigned push to driver leaks customer address before driver acceptance
- **`api-push-delivery-pickedup-customer`** — flagged by TW(P2), BA(P2), BV(P1) — picked-up push uses generic copy; missing ETA refresh
- **`ap-approvaldetail-toasts`** — flagged by TW(P2), Legal(P1), BA(P3) — admin approval-detail action toasts missing audit-trail confirmation
- **`ap-providerdetail-toasts`** — flagged by TW(P3), Legal(P2), BV(P2) — provider-detail toasts use "Are you sure?" anti-pattern
- **`mc-catering-submit-success`** — flagged by TW(P2), BA(P2), BV(P2) — catering submit-success on customer mobile lacks response-time SLA
- **`mc-profile-save-success`** — flagged by TW(P3), Legal(P3), BV(P3) — customer profile save toast style drift
- **`md-trx-008`** — flagged by TW(P3), BA(P1), BV(P2) — mobile delivery transactional drift
- **`md-trx-018`** — flagged by TW(P2), Legal(P2), BA(P3) — mobile delivery transactional missing audit linkage
- **`mv-onb-pending-rejected-body`** — flagged by TW(P3), Legal(P2), BA(P1) — mobile-vendor onboarding rejection body lacks reason + appeal path
- **`mv-onb-pending-submitted-title`** — flagged by TW(P2), BA(P1), BV(P2) — mobile-vendor pending-submitted title overstates approval likelihood
- **`vp-onb-personal-avatar-success`** — flagged by TW(P3), Legal(P3), BA(P3) — vendor onboarding avatar success style drift
- **`vp-ux-kitchen-photo-toasts`** — flagged by TW(P2), Legal(P2), BV(P3) — kitchen-photo upload toast inconsistency
- **`vp-ux-profile-doc-toasts`** — flagged by TW(P3), Legal(P1), BA(P2) — chef profile doc upload toasts; missing KYC-cadence reminder
- **`vp-ux-settings-payout-toasts`** — flagged by TW(P3), Legal(P1), BA(P3) — payout-settings toasts missing RBI PA reconciliation reference
- **`web-tx-chefdetail-add-to-cart`** — flagged by TW(P2), Legal(P3), BA(P2) — chef-detail add-to-cart toast drifts vs cart sheet
- **`web-tx-profile-update`** — flagged by TW(P2), Legal(P2), BV(P3) — profile-update toast drift and missing PII-change audit log

Top P0/P1 dual-lens surfaces (severity-weighted):

- **`api-inapp-order-cancelled-both`** — flagged by TW(P0), Legal(P1) — order-cancelled in-app message lacks refund timeline (money-bearing)
- **`api-push-approval-rejected-chef`** — flagged by TW(P0), Legal(P1) — chef-rejected push lacks due-process / appeal language
- **`api-push-order-update-customer`** — flagged by TW(P1, P2), BV(P0) — customer order-update push drift and brand-string contradiction
- **`dp-tx-toast-partner-suspend`** — flagged by Legal(P0), BV(P3) — driver partner-suspend toast lacks reason, appeal, Grievance Officer
- **`md-trx-007`** — flagged by TW(P0), BV(P2) — mobile delivery transactional with P0 TW finding

## Technical Writer findings

```yaml
# Technical Writer lens — TRANSACTIONAL category findings
# Audited 154 rows from CONTENT-INVENTORY.md (transactional section).
# Focus: backend-generated copy (email templates, push, SMS, in-app), plus front-end toast/alert content.

# ============================================================================
# API — BRAND DRIFT & SHARED INFRASTRUCTURE (highest blast radius)
# ============================================================================

findings:

  - finding_id: TW-001
    surface_id: api-email-account-reminder
    lens: technical-writer
    severity: P0
    issue: "Brand drift inside email subject — 'Fe3dr HomeChef' compound name contradicts the single-brand voice used everywhere else"
    evidence_excerpt: "Subject: 'You already have a Fe3dr HomeChef account' / signoff: '— Fe3dr HomeChef'"
    recommendation: "Pick one brand name and use it consistently across every email surface. STYLE-GUIDE.md uses 'Home Chef'; production code uses 'Fe3dr'. Resolve to one and apply repo-wide. As a pure TW fix: replace with 'You already have a Home Chef account' and signoff '— Home Chef'."
    depends_on: null

  - finding_id: TW-002
    surface_id: api-push-user-welcome
    lens: technical-writer
    severity: P0
    issue: "Brand name inside welcome push contradicts welcome email — push says 'HomeChef', email says 'Fe3dr'. A new user receives two products in two minutes"
    evidence_excerpt: "Push title: 'Welcome to HomeChef!' / Email subject: 'Welcome to Fe3dr!'"
    recommendation: "Standardize on 'Home Chef' (per STYLE-GUIDE) in both push title and email subject. Update services/notifications.go::handleUserRegistered and services/email_templates.go::WelcomeEmailHTML in lockstep."
    depends_on: null

  - finding_id: TW-003
    surface_id: api-email-welcome
    lens: technical-writer
    severity: P1
    issue: "Welcome email opens with marketing hype banned by Rule 5 (Restraint over urgency) and Rule 3 (Photo-forward, chrome-light) — 'thrilled', 'amazing', 'talented' stack in one paragraph"
    evidence_excerpt: "We're thrilled to have you. Discover amazing home-cooked meals from talented local chefs right in your neighbourhood."
    recommendation: "Rewrite as: 'Welcome, %s. Discover home chefs cooking in your neighbourhood — order what's available today, track your delivery, and rate the meal when it arrives.' Remove 'thrilled' / 'amazing' / 'talented'."
    depends_on: null

  - finding_id: TW-004
    surface_id: api-email-welcome
    lens: technical-writer
    severity: P2
    issue: "Welcome CTA button uses title-case 'Start Exploring' violating sentence-case rule (Sec 4 Buttons)"
    evidence_excerpt: "<a href='https://fe3dr.com' class='btn'>Start Exploring</a>"
    recommendation: "Change to 'Start exploring' (sentence case)."
    depends_on: null

  - finding_id: TW-005
    surface_id: api-email-welcome
    lens: technical-writer
    severity: P2
    issue: "Exclamation in subject 'Welcome to Fe3dr!' is a Rule 1 violation when the same recipient already gets an exclamation push and an in-app banner — exclamations stack across surfaces"
    evidence_excerpt: "Subject: 'Welcome to Fe3dr!'"
    recommendation: "Drop the exclamation: 'Welcome to Home Chef'."
    depends_on: null

  - finding_id: TW-006
    surface_id: api-email-verify
    lens: technical-writer
    severity: P2
    issue: "Verify-email CTA uses title-case 'Verify Email Address' (3 words is fine; case is wrong)"
    evidence_excerpt: "<a ... class='btn'>Verify Email Address</a>"
    recommendation: "Change to 'Verify email' (also tightens to 2 words)."
    depends_on: null

  - finding_id: TW-007
    surface_id: api-email-verify
    lens: technical-writer
    severity: P3
    issue: "Helper sentence 'Thanks for signing up! Please verify your email address to activate your account and start ordering.' contains an exclamation in transactional context (Rule 1) and is 17 words"
    evidence_excerpt: "Thanks for signing up! Please verify your email address to activate your account and start ordering."
    recommendation: "Rewrite: 'Thanks for signing up. Verify your email to activate your account.' Single sentence, no exclamation."
    depends_on: null

  - finding_id: TW-008
    surface_id: api-email-password-reset
    lens: technical-writer
    severity: P2
    issue: "CTA 'Reset Password' is title-case; should be sentence case per Sec 4"
    evidence_excerpt: "<a ... class='btn'>Reset Password</a>"
    recommendation: "Change to 'Reset password'."
    depends_on: null

  - finding_id: TW-009
    surface_id: api-email-staff-invite
    lens: technical-writer
    severity: P2
    issue: "H2 'You've been invited!' uses exclamation in a transactional admin email (Rule 1)"
    evidence_excerpt: "<h2>You've been invited!</h2>"
    recommendation: "Change to 'You've been invited' — no exclamation."
    depends_on: null

  - finding_id: TW-010
    surface_id: api-email-staff-invite
    lens: technical-writer
    severity: P2
    issue: "CTA 'Accept Invitation' is title-case; should be sentence case"
    evidence_excerpt: "<a ... class='btn'>Accept Invitation</a>"
    recommendation: "Change to 'Accept invitation'."
    depends_on: null

  - finding_id: TW-011
    surface_id: api-email-staff-invite
    lens: technical-writer
    severity: P3
    issue: "Role injected as raw string into body — if role enum is 'fleet_manager' user reads 'has invited you to join the Fe3dr team as fleet_manager' (raw slug leak)"
    evidence_excerpt: "<strong>%s</strong> has invited you to join the Fe3dr team as <strong>%s</strong>"
    recommendation: "Wrap role with a label lookup (e.g. 'a Fleet Manager') and ensure the caller never passes raw enum slugs."
    depends_on: null

  - finding_id: TW-012
    surface_id: api-email-order-confirm
    lens: technical-writer
    severity: P2
    issue: "H2 'Order Confirmed!' uses exclamation against Rule 1; H2 also uses title case while body uses sentence case (within-surface voice drift)"
    evidence_excerpt: "<h2>Order Confirmed!</h2>"
    recommendation: "Change to 'Order confirmed' (no exclamation, sentence case)."
    depends_on: null

  - finding_id: TW-013
    surface_id: api-email-order-confirm
    lens: technical-writer
    severity: P2
    issue: "'Your home chef is preparing your meal!' adds an exclamation and assumes work has started — but order may still be in confirmation queue. Mixes commitment and excitement"
    evidence_excerpt: "Your order #%s has been placed successfully. Your home chef is preparing your meal!"
    recommendation: "Rewrite: 'Your order #%s is placed. Your home chef will start preparing it shortly.' No exclamation, accurate state."
    depends_on: null

  - finding_id: TW-014
    surface_id: api-email-order-confirm
    lens: technical-writer
    severity: P2
    issue: "CTA 'Track Your Order' is title-case and 3 words including possessive — sentence case rule + tighter form"
    evidence_excerpt: "<a ... class='btn'>Track Your Order</a>"
    recommendation: "Change to 'Track order'."
    depends_on: null

  - finding_id: TW-015
    surface_id: api-email-order-confirm
    lens: technical-writer
    severity: P3
    issue: "Total label is plain 'Total' (correct per Sec 3) but column header 'Price' is ambiguous when items have quantity — should be 'Line total' or 'Amount'"
    evidence_excerpt: "<th ...>Price</th> ... ₹%.2f"
    recommendation: "Rename column to 'Amount' so the price row reads as the calculated line total, not a unit price."
    depends_on: null

  - finding_id: TW-016
    surface_id: api-email-order-status-confirmed
    lens: technical-writer
    severity: P2
    issue: "Within-surface voice drift: H2 still says generic 'Order Update' across all 7 statuses while subject is custom, leaving the body bland"
    evidence_excerpt: "<h2>%s Order Update</h2><p>Order <strong>#%s</strong>: %s</p>"
    recommendation: "Make the H2 use the same status label as the subject (e.g. 'Your chef is preparing your meal'). The redundant 'Order #X: ...' line can then be removed."
    depends_on: null

  - finding_id: TW-017
    surface_id: api-email-order-status-preparing
    lens: technical-writer
    severity: P2
    issue: "Status label 'Your chef is preparing your meal' (email) does not match push label 'Your order is being prepared' (notifications.go:1034) or handlers/notifications label 'being prepared'"
    evidence_excerpt: "preparing → 'Your chef is preparing your meal'"
    recommendation: "Consolidate all 3 status-label maps (services/email_templates.go::statusLabels, services/notifications.go::getOrderStatusMessage, handlers/notifications.go::humanReadableOrderStatus) into one canonical map exported from a single package. Use 'Your chef is preparing your meal' as the canonical preparing label (the strongest emotionally-anchored variant)."
    depends_on: null

  - finding_id: TW-018
    surface_id: api-email-order-status-ready
    lens: technical-writer
    severity: P2
    issue: "Label 'Your order is ready for pickup' contradicts customer-facing push 'Your order is ready for pickup/delivery'. A delivery order receives a 'ready for pickup' email"
    evidence_excerpt: "ready → 'Your order is ready for pickup'"
    recommendation: "Split into two labels keyed by order mode: 'Your order is ready for pickup' (pickup orders) and 'Your order is ready for delivery' (delivery orders). The mode is already on the OrderEvent."
    depends_on: null

  - finding_id: TW-019
    surface_id: api-email-order-status-pickedup
    lens: technical-writer
    severity: P2
    issue: "Label uses 'delivery partner' (customer-facing term — good) while push (api-push-order-update-customer-deeplink) says 'picked up by the driver' (banned customer-facing term per Sec 3)"
    evidence_excerpt: "picked_up → 'Your delivery partner has picked up your order'"
    recommendation: "Adopt 'delivery partner' consistently in customer-facing copy. Update handlers/notifications.go::humanReadableOrderStatus 'picked_up' label from 'picked up by the driver' to 'picked up by the delivery partner'."
    depends_on: null

  - finding_id: TW-020
    surface_id: api-email-order-status-onway
    lens: technical-writer
    severity: P2
    issue: "Subject has exclamation 'Your order is on the way!' against Rule 1; sister surface (in-app/push) says 'on its way!' — inconsistent possessive + exclamation in both"
    evidence_excerpt: "Subject: 'Order #%s — Your order is on the way!'"
    recommendation: "Rewrite both surfaces to 'Your order is on its way' (no exclamation, idiomatic possessive). Update the email label and push body together."
    depends_on: null

  - finding_id: TW-021
    surface_id: api-email-order-status-delivered
    lens: technical-writer
    severity: P2
    issue: "Subject 'Your order has been delivered. Enjoy your meal!' compresses two thoughts; exclamation in transactional subject violates Rule 1"
    evidence_excerpt: "Subject: 'Order #%s — Your order has been delivered. Enjoy your meal!'"
    recommendation: "Subject: 'Order #%s — Delivered'. Body H2: 'Order delivered'. Closing paragraph: 'Enjoy your meal.'"
    depends_on: null

  - finding_id: TW-022
    surface_id: api-email-order-status-delivered
    lens: technical-writer
    severity: P2
    issue: "Three different 'enjoy' phrasings across surfaces: email 'Enjoy your meal!', push 'Enjoy your meal!', in-app status 'Enjoy!' (notifications.go:1038). Same product, three voices"
    evidence_excerpt: "email: 'Enjoy your meal!' / push: 'Enjoy your meal!' / in-app status: 'Enjoy!'"
    recommendation: "Use 'Enjoy your meal.' (period, no exclamation) on every surface."
    depends_on: null

  - finding_id: TW-023
    surface_id: api-email-order-status-cancelled
    lens: technical-writer
    severity: P0
    issue: "Cancellation email omits refund timeline and who-to-contact — a P0 transactional gap for a paid order. Body reduces to one sentence with no next step"
    evidence_excerpt: "Subject: 'Order #%s — Your order has been cancelled' / Body: just 'Order #X: Your order has been cancelled' + CTA 'View Order Details'"
    recommendation: "Add: 'Your refund of ₹X has been issued to your original payment method and will appear in 5–7 business days.' Plus contact line: 'Need help? Contact support at support@homechef.in.' (Legal lens owns refund-period accuracy.)"
    depends_on: null

  - finding_id: TW-024
    surface_id: api-email-chef-new-order
    lens: technical-writer
    severity: P1
    issue: "Subject and H2 'New order' fine, but body opens with '🔔 New Order!' violating Rule 1 (exclamation + emoji in transactional). Chef-facing tone should be functional, time-aware (5-12 words crisp)"
    evidence_excerpt: "<h2>🔔 New Order!</h2><p>You have a new order #%s worth ₹%.2f.</p>"
    recommendation: "Rewrite H2 as 'New order #%s' and body as 'A new order worth ₹%.2f is waiting. Accept or decline from your dashboard.' Drop bell emoji."
    depends_on: null

  - finding_id: TW-025
    surface_id: api-email-chef-new-order
    lens: technical-writer
    severity: P0
    issue: "SLA commitment in info-box ('Customers expect a response within 5 minutes') is a factual claim the product may not enforce — no chef-facing timer is exposed on web/mobile vendor portals. False promise to chef"
    evidence_excerpt: "Customers expect a response within 5 minutes. Please accept or decline promptly."
    recommendation: "Either expose the 5-minute auto-reject timer in the vendor portal (BA lens scope) or weaken the email line to 'Accept or decline as soon as you can — customers see real-time chef response times.'"
    depends_on: null

  - finding_id: TW-026
    surface_id: api-email-chef-new-order
    lens: technical-writer
    severity: P2
    issue: "CTA 'View Order' is fine in length but title-case violates Sec 4"
    evidence_excerpt: "<a ... class='btn'>View Order</a>"
    recommendation: "Change to 'View order'."
    depends_on: null

  - finding_id: TW-027
    surface_id: api-email-chef-verified
    lens: technical-writer
    severity: P2
    issue: "Body uses 'Congratulations, %s!' — exclamation in transactional; H2 is also title-case-style ('Your kitchen is verified — Fe3dr')"
    evidence_excerpt: "<h2>Congratulations, %s!</h2>"
    recommendation: "Change H2 to 'Your kitchen is verified, %s' — keeps personalization, removes exclamation, mirrors subject."
    depends_on: null

  - finding_id: TW-028
    surface_id: api-email-chef-verified
    lens: technical-writer
    severity: P2
    issue: "Bulleted next-steps list ends with exclamation 'Start accepting orders!' — third exclamation in this email"
    evidence_excerpt: "<li>Start accepting orders!</li>"
    recommendation: "Change to 'Start accepting orders.' Strip all exclamations from this email per Rule 1 (≤1 per page, none in transactional)."
    depends_on: null

  - finding_id: TW-029
    surface_id: api-email-chef-verified
    lens: technical-writer
    severity: P2
    issue: "CTA 'Go to Your Dashboard' is title-case, contains possessive, 4 words — violates Sec 4 (verb-first, ≤3 words, sentence case)"
    evidence_excerpt: "<a ... class='btn'>Go to Your Dashboard</a>"
    recommendation: "Change to 'Open dashboard'."
    depends_on: null

  - finding_id: TW-030
    surface_id: api-email-chef-verified
    lens: technical-writer
    severity: P2
    issue: "Within-product naming drift: this email says 'kitchen', but the in-app notification (api-inapp-chef-verified) says 'chef profile'. Same event, two nouns"
    evidence_excerpt: "Email: 'Your kitchen has been verified' / In-app: 'Your chef profile has been verified'"
    recommendation: "Pick one term. Recommend 'chef profile' for both because the verification is of the chef record, not the physical kitchen. Update the email and message body together."
    depends_on: null

  - finding_id: TW-031
    surface_id: api-email-delivery-assigned
    lens: technical-writer
    severity: P0
    issue: "Routing bug: function returns driver-oriented copy ('You've been assigned order #...') but is dispatched to the CUSTOMER in services/notifications.go::handleDeliveryAssigned (line 569-581). Customer reads driver copy"
    evidence_excerpt: "Subject: 'Delivery assigned — Order #%s' / Body: '🚗 New Delivery! You\\'ve been assigned order #%s.'"
    recommendation: "Split into two templates: DeliveryAssignedDriverHTML (existing copy, sent to driver) and DeliveryAssignedCustomerHTML ('Your delivery partner %s is on the way to pick up order #%s'). Update sendEmailNotification routing in services/notifications.go:910-914 to dispatch the customer template for the customer-facing event."
    depends_on: null

  - finding_id: TW-032
    surface_id: api-email-delivery-assigned
    lens: technical-writer
    severity: P2
    issue: "H2 '🚗 New Delivery!' uses emoji + exclamation in transactional driver email"
    evidence_excerpt: "<h2>🚗 New Delivery!</h2>"
    recommendation: "Change to 'New delivery — order #%s'. Driver tone per Sec 2 should be telegraphic and imperative; emoji and exclamation undermine that."
    depends_on: null

  - finding_id: TW-033
    surface_id: api-email-delivery-assigned
    lens: technical-writer
    severity: P2
    issue: "CTA 'View Delivery Details' is title-case and 3 words — sentence-case violation"
    evidence_excerpt: "<a ... class='btn'>View Delivery Details</a>"
    recommendation: "Change to 'Open delivery'."
    depends_on: null

  - finding_id: TW-034
    surface_id: api-email-support-created
    lens: technical-writer
    severity: P0
    issue: "Inline HTML bypasses emailBase wrapper — recipient gets no header, no footer, no privacy/terms links, no brand presence. Voice drift vs every other branded email"
    evidence_excerpt: "html := fmt.Sprintf(`<h2>Support Ticket Created</h2><p>Your support ticket <strong>#%s</strong> has been created.</p>...`)"
    recommendation: "Refactor SendSupportTicketCreated to use the emailBase wrapper. Create a SupportTicketCreatedHTML template in email_templates.go that follows the same H2/body/CTA pattern."
    depends_on: null

  - finding_id: TW-035
    surface_id: api-email-support-created
    lens: technical-writer
    severity: P2
    issue: "H2 'Support Ticket Created' uses title case; subject 'Support Ticket #%s — %s' also title case — two voice rules broken in one surface"
    evidence_excerpt: "<h2>Support Ticket Created</h2>"
    recommendation: "After moving into emailBase, change H2 to 'Support ticket created' (sentence case). Subject 'Support ticket #%s — %s'."
    depends_on: null

  - finding_id: TW-036
    surface_id: api-email-support-created
    lens: technical-writer
    severity: P2
    issue: "'Our team will get back to you shortly' is vague — sets no SLA and uses 'get back to you' (filler phrase) instead of a concrete commitment"
    evidence_excerpt: "Our team will get back to you shortly."
    recommendation: "Replace with concrete SLA: 'We respond to most tickets within 24 hours.' If 24h is wrong, use the actual SLA from settings."
    depends_on: null

  - finding_id: TW-037
    surface_id: api-email-support-update
    lens: technical-writer
    severity: P0
    issue: "Same unbranded-email bug as TW-034 — inline HTML, no emailBase wrapper, no privacy/terms footer"
    evidence_excerpt: "html := fmt.Sprintf(`<h2>Support Ticket Update</h2><p>...has been updated to: <strong>%s</strong></p>`)"
    recommendation: "Refactor SendSupportTicketUpdate to use emailBase via a new SupportTicketUpdateHTML template."
    depends_on: null

  - finding_id: TW-038
    surface_id: api-email-support-update
    lens: technical-writer
    severity: P2
    issue: "Status field injected verbatim — if backend uses raw enums (e.g. 'in_progress', 'awaiting_user') the customer sees the slug"
    evidence_excerpt: "Your support ticket #%s has been updated to: %s"
    recommendation: "Map status enum to human-readable labels in the same lookup pattern as the email status labels."
    depends_on: null

  - finding_id: TW-039
    surface_id: api-email-account-reminder
    lens: technical-writer
    severity: P2
    issue: "Inline HTML — no emailBase wrapper, so footer/privacy/terms missing. Same brand drift as support emails"
    evidence_excerpt: "html := fmt.Sprintf(`<p>Hi %s,</p>...<p>— Fe3dr HomeChef</p>`)"
    recommendation: "Move into emailBase with an AccountReminderHTML template."
    depends_on: null

  # ============================================================================
  # API — PUSH / IN-APP STATUS-LABEL DUPLICATION (the 3-map bug)
  # ============================================================================

  - finding_id: TW-040
    surface_id: api-push-order-update-customer-deeplink
    lens: technical-writer
    severity: P0
    issue: "Customer receives TWO pushes for the same status change with different copy. Push-workers queue group sends 'Your order is now picked up by the driver' (handlers/notifications.go:382) while notification-workers sends 'Your order has been picked up by the delivery partner' (services/notifications.go:1031). Both fire on the same NATS event"
    evidence_excerpt: "handlers: 'Your order is now %s' (picked_up → 'picked up by the driver') / services: 'Your order has been picked up by the delivery partner'"
    recommendation: "Either (a) move all push notification dispatch into one queue group with one consumer, or (b) make the push-worker the canonical sender and remove the duplicate path in services/notifications.go::handleOrderUpdated. Then collapse the three status-label maps into one shared lookup. This is the highest-priority TW fix in the audit."
    depends_on: null

  - finding_id: TW-041
    surface_id: api-push-order-update-customer-deeplink
    lens: technical-writer
    severity: P2
    issue: "Driver-facing label 'picked up by the driver' uses banned customer-facing term 'driver' (Sec 3: use 'delivery partner' customer-facing)"
    evidence_excerpt: "labels['picked_up'] = 'picked up by the driver'"
    recommendation: "Change to 'picked up by your delivery partner'. Personalized via possessive."
    depends_on: null

  - finding_id: TW-042
    surface_id: api-push-order-update-customer-deeplink
    lens: technical-writer
    severity: P2
    issue: "'delivering' status mapped to 'on the way to you' but other surfaces use 'on its way' or 'on its way!' — same-word drift across three places"
    evidence_excerpt: "labels['delivering'] = 'on the way to you'"
    recommendation: "Standardize on 'on its way' for all three maps."
    depends_on: null

  - finding_id: TW-043
    surface_id: api-push-order-update-customer
    lens: technical-writer
    severity: P1
    issue: "Push body 'Your order has been confirmed by the chef!' (services/notifications.go:1033) ends in exclamation in transactional push — Rule 1"
    evidence_excerpt: "confirmed → 'Your order has been confirmed by the chef!'"
    recommendation: "Drop the exclamation across all 7 status labels in getOrderStatusMessage."
    depends_on: null

  - finding_id: TW-044
    surface_id: api-push-order-update-customer
    lens: technical-writer
    severity: P2
    issue: "'Your order is on its way!' label keeps an exclamation while sibling labels are flat — within-map tone drift"
    evidence_excerpt: "on_the_way → 'Your order is on its way!'"
    recommendation: "Make all 7 labels declarative with periods. No exclamations."
    depends_on: null

  - finding_id: TW-045
    surface_id: api-inapp-order-status-customer
    lens: technical-writer
    severity: P2
    issue: "Title 'Order Status Updated' differs from push title 'Order Update' for the same event (within-event drift)"
    evidence_excerpt: "In-app title: 'Order Status Updated' / Push title: 'Order Update'"
    recommendation: "Use 'Order update' on both. Title case → sentence case."
    depends_on: null

  - finding_id: TW-046
    surface_id: api-push-order-update-customer-deeplink
    lens: technical-writer
    severity: P2
    issue: "Title 'Order Update' is title-case; should be 'Order update' (sentence case rule applies to push titles too)"
    evidence_excerpt: "Title: 'Order Update'"
    recommendation: "Change push title to 'Order update'."
    depends_on: null

  - finding_id: TW-047
    surface_id: api-push-chef-new-order-actionable
    lens: technical-writer
    severity: P1
    issue: "Chef receives TWO pushes for the same NATS event: 'New Order' + 'You have a new order waiting for your confirmation' (handlers) and 'New Order Received' + 'You have a new order waiting to be prepared!' (services). Two queue groups duplicate copy"
    evidence_excerpt: "handlers: 'New Order' / 'You have a new order waiting for your confirmation' — services: 'New Order Received' / 'You have a new order waiting to be prepared!'"
    recommendation: "Decide which queue group owns the chef push. Suggest the handlers/notifications.go::RegisterPushConsumers path (it's actionable with Accept/Reject buttons) and remove the services push in handleOrderCreated lines 346-352. Then standardize copy: title 'New order', body 'A new order is waiting. Accept or decline.'"
    depends_on: null

  - finding_id: TW-048
    surface_id: api-push-chef-new-order-actionable
    lens: technical-writer
    severity: P2
    issue: "Push title 'New Order' is title-case (Sec 4 says sentence case for buttons; push titles follow the same convention in modern style guides)"
    evidence_excerpt: "Title: 'New Order'"
    recommendation: "Change to 'New order'."
    depends_on: null

  - finding_id: TW-049
    surface_id: api-push-chef-new-order-actionable
    lens: technical-writer
    severity: P3
    issue: "Body 'You have a new order waiting for your confirmation' is 9 words — fine for length but contains weasel-phrasing 'waiting for your confirmation' instead of an action"
    evidence_excerpt: "You have a new order waiting for your confirmation"
    recommendation: "Change to 'A new order is waiting. Accept or decline.' — clearer action."
    depends_on: null

  - finding_id: TW-050
    surface_id: api-inapp-chef-new-order
    lens: technical-writer
    severity: P2
    issue: "Title 'New Order!' with exclamation; sibling in-app/push titles for same event use 'New Order Received' (no exclamation). Three variants of 'new order' copy in three places"
    evidence_excerpt: "Title: 'New Order!' — Message: 'You have a new order to prepare'"
    recommendation: "Consolidate to one title 'New order' across all three surfaces. Drop exclamation."
    depends_on: null

  - finding_id: TW-051
    surface_id: api-inapp-order-created-chef
    lens: technical-writer
    severity: P2
    issue: "Within-event copy drift: DB row saved as 'You have received a new order!' but push body says 'You have a new order waiting to be prepared!' — same event, two messages"
    evidence_excerpt: "DB: 'You have received a new order!' / Push: 'You have a new order waiting to be prepared!'"
    recommendation: "Use one canonical body: 'A new order is waiting. Accept or decline.'"
    depends_on: null

  - finding_id: TW-052
    surface_id: api-inapp-order-delivered-customer
    lens: technical-writer
    severity: P2
    issue: "Customer-facing delivered title 'Order Delivered' is title-case + push body has double exclamation 'Your order has been delivered! Enjoy your meal!' — two exclamations in one push"
    evidence_excerpt: "Title: 'Order Delivered' — Body: 'Your order has been delivered! Enjoy your meal!'"
    recommendation: "Title: 'Order delivered'. Body: 'Your order has been delivered. Enjoy your meal.' No exclamations."
    depends_on: null

  - finding_id: TW-053
    surface_id: api-push-order-delivered-customer
    lens: technical-writer
    severity: P2
    issue: "Same double-exclamation issue as TW-052 on the push variant"
    evidence_excerpt: "Body: 'Your order has been delivered! Enjoy your meal!'"
    recommendation: "Rewrite per TW-052."
    depends_on: null

  - finding_id: TW-054
    surface_id: api-push-delivery-pickedup-customer
    lens: technical-writer
    severity: P2
    issue: "Title 'Order On The Way!' uses title case for prepositions ('On The Way' is wrong even in title case — 'on the' should be lowercase). Also exclamation in transactional"
    evidence_excerpt: "Title: 'Order On The Way!'"
    recommendation: "Change to 'On its way'."
    depends_on: null

  - finding_id: TW-055
    surface_id: api-inapp-delivery-pickedup-customer
    lens: technical-writer
    severity: P2
    issue: "Title 'Order Picked Up' (in-app) does not match push title 'Order On The Way!' (push). Same event, two titles"
    evidence_excerpt: "In-app: 'Order Picked Up' / Push: 'Order On The Way!'"
    recommendation: "Use one title 'On its way' on both surfaces."
    depends_on: null

  - finding_id: TW-056
    surface_id: api-push-delivery-assigned-customer
    lens: technical-writer
    severity: P2
    issue: "Body 'A delivery partner has been assigned to your order and will pick it up soon!' is 15 words (within 25-word customer limit) but ends in exclamation and is passive throughout"
    evidence_excerpt: "A delivery partner has been assigned to your order and will pick it up soon!"
    recommendation: "Rewrite active and short: 'Your delivery partner is on the way to pick up your order.' Use driver name when available (already in event.Data but unused)."
    depends_on: null

  - finding_id: TW-057
    surface_id: api-inapp-delivery-assigned-customer
    lens: technical-writer
    severity: P2
    issue: "In-app body 'A delivery partner has been assigned to your order' is shorter than the push variant (TW-056) — within-event drift"
    evidence_excerpt: "In-app body: 'A delivery partner has been assigned to your order' / Push body: '... and will pick it up soon!'"
    recommendation: "Match TW-056. One canonical body."
    depends_on: null

  - finding_id: TW-058
    surface_id: api-push-delivery-assigned-driver
    lens: technical-writer
    severity: P2
    issue: "Driver push title 'New Delivery Available' is 3 words but title case + driver tone should be telegraphic"
    evidence_excerpt: "Title: 'New Delivery Available' — Body: 'A delivery near you is ready for pickup'"
    recommendation: "Title: 'Delivery available'. Body: 'New delivery ready for pickup nearby.' Driver per Sec 2 = telegraphic ≤4 words where possible."
    depends_on: null

  - finding_id: TW-059
    surface_id: api-push-order-cancelled-both
    lens: technical-writer
    severity: P0
    issue: "Cancellation push body 'Order has been cancelled' is the SAME for both customer and chef despite very different next steps (customer needs refund info, chef needs nothing). Bare wording, no refund window"
    evidence_excerpt: "Title: 'Order Cancelled' — Body: 'Order has been cancelled'"
    recommendation: "Split into customer body ('Your order was cancelled. Refund issued.') and chef body ('Order cancelled. No action needed.'). The Title 'Order cancelled' (sentence case) is fine for both."
    depends_on: null

  - finding_id: TW-060
    surface_id: api-inapp-order-cancelled-both
    lens: technical-writer
    severity: P0
    issue: "Same dual-audience body problem as TW-059 on the in-app side"
    evidence_excerpt: "Title: 'Order Cancelled' — Message: 'Order has been cancelled'"
    recommendation: "Same as TW-059 — split per audience."
    depends_on: null

  - finding_id: TW-061
    surface_id: api-inapp-chef-verified
    lens: technical-writer
    severity: P2
    issue: "Title 'Congratulations!' is bare emoji-replacement-style copy with exclamation in transactional context"
    evidence_excerpt: "Title: 'Congratulations!' — Message: 'Your chef profile has been verified. You can now start accepting orders!'"
    recommendation: "Title: 'Profile verified'. Message: 'Your chef profile is verified. You can start accepting orders.' Drops two exclamations and gives the bell a more useful glance-state."
    depends_on: null

  - finding_id: TW-062
    surface_id: api-email-chef-verified-publish
    lens: technical-writer
    severity: P3
    issue: "Shadowed email title 'Your Chef Profile is Verified!' surfaces only if template lookup fails — but it's title-case with exclamation, so the fallback path leaks bad copy"
    evidence_excerpt: "Email Title: 'Your Chef Profile is Verified!'"
    recommendation: "Even though shadowed, fix the fallback title to 'Your chef profile is verified' for safety."
    depends_on: null

  - finding_id: TW-063
    surface_id: api-inapp-driver-onboarding-admins
    lens: technical-writer
    severity: P2
    issue: "Message 'A new driver from %s has submitted their onboarding application for review.' is 16 words — fine for admin tone but uses gerund 'submitted their onboarding application' (wordy)"
    evidence_excerpt: "A new driver from %s has submitted their onboarding application for review."
    recommendation: "Tighten to: 'New driver application from %s — review pending.' (9 words, admin-direct per Sec 2.)"
    depends_on: null

  # ============================================================================
  # API — APPROVAL FLOW (raw slug exposure)
  # ============================================================================

  - finding_id: TW-064
    surface_id: api-inapp-approval-approved-chef
    lens: technical-writer
    severity: P1
    issue: "Raw enum slug exposed to user: 'Your menu_item has been approved' because approvalType is interpolated verbatim. Banned per Rule 2 (plain English)"
    evidence_excerpt: "Message: 'Your %s has been approved: %s' where %s is 'menu_item' / 'driver_document'"
    recommendation: "Add a label map: approvalTypeLabel['menu_item'] = 'menu item', approvalTypeLabel['driver_document'] = 'document', etc. Apply across all four approval handlers (approved, rejected, info_requested, created)."
    depends_on: null

  - finding_id: TW-065
    surface_id: api-push-approval-approved-chef
    lens: technical-writer
    severity: P1
    issue: "Same raw-slug bug on push body 'Your %s has been approved'. Title also has exclamation 'Request Approved!' while in-app title is bare 'Request Approved'"
    evidence_excerpt: "Title: 'Request Approved!' — Body: 'Your %s has been approved'"
    recommendation: "Fix slug via shared label map (TW-064). Drop exclamation. Title 'Request approved' across both surfaces."
    depends_on: null

  - finding_id: TW-066
    surface_id: api-inapp-approval-info-chef
    lens: technical-writer
    severity: P1
    issue: "Same raw-slug bug; in addition, admin notes appended verbatim — chef sees raw admin language with no scrubbing"
    evidence_excerpt: "Message: 'Admin needs more info about your %s. Notes: %s'"
    recommendation: "Fix slug via shared label map. Wrap notes in a delimiter ('Admin note: …') so the chef knows it's quoted text, not platform copy."
    depends_on: null

  - finding_id: TW-067
    surface_id: api-push-approval-info-chef
    lens: technical-writer
    severity: P2
    issue: "Push body uses same admin notes verbatim — push notifications truncate at ~200 chars on Android and ~178 chars on iOS, so long admin notes will be cut mid-sentence"
    evidence_excerpt: "Body: same as in-app message"
    recommendation: "Push body should be a short template ('Admin needs more info about your menu item.') with notes shown only in the deep-linked in-app screen."
    depends_on: null

  - finding_id: TW-068
    surface_id: api-inapp-approval-rejected-chef
    lens: technical-writer
    severity: P0
    issue: "Triple problem: raw slug, verbatim admin notes, and 'Your menu_item has been rejected. Notes: not good enough' is unprofessional + potential PII leak if admin types names. Title 'Request Rejected' also reads harsh"
    evidence_excerpt: "Title: 'Request Rejected' — Message: 'Your %s has been rejected. Notes: %s'"
    recommendation: "Title: 'Resubmission needed' (softer + actionable, matches mobile-vendor pattern). Message: 'Your %s wasn't approved. Admin note: \"%s\".' Apply slug fix from TW-064."
    depends_on: null

  - finding_id: TW-069
    surface_id: api-push-approval-rejected-chef
    lens: technical-writer
    severity: P0
    issue: "Same as TW-068 on the push path. Title 'Request Rejected' is also a banned harsh-tone match — admin-portal already uses 'Application Not Approved' (mv-onb-pending-rejected-title) which is softer"
    evidence_excerpt: "Title: 'Request Rejected' — Body: same as in-app"
    recommendation: "Align with mobile-vendor pattern: 'Resubmission needed' title, softer body. Slug fix from TW-064."
    depends_on: null

  - finding_id: TW-070
    surface_id: api-inapp-approval-created-admin
    lens: technical-writer
    severity: P3
    issue: "Approval title injected verbatim — if chef-authored title contains profanity or HTML, it flows to all admins unfiltered"
    evidence_excerpt: "Message: 'New approval request pending: %s'"
    recommendation: "Sanitize the title server-side and cap length at 80 chars. Admin notifications are not the place for unmoderated chef-authored text."
    depends_on: null

  # ============================================================================
  # API — GENERIC SUCCESS MESSAGES (43 strings in handlers/*)
  # ============================================================================

  - finding_id: TW-071
    surface_id: api-success-message-generic
    lens: technical-writer
    severity: P2
    issue: "Inconsistent 'successfully' suffix — 16 of 43 messages end with 'successfully', 27 don't. Same domain, two patterns: 'Password changed successfully' vs 'Password has been reset successfully' vs '2FA enabled'"
    evidence_excerpt: "'Password changed successfully' / '2FA disabled' / 'Provider deleted successfully' / 'Promo code deactivated'"
    recommendation: "Strip 'successfully' from every message — the 200 OK status already implies success. Use past-tense bare verb: 'Password changed', 'Provider deleted', 'Promo code deactivated'. Apply repo-wide across handlers/*.go."
    depends_on: null

  - finding_id: TW-072
    surface_id: api-success-message-generic
    lens: technical-writer
    severity: P2
    issue: "Logout message 'Logged out successfully' uses banned 'Log out' compound (per Sec 3 'Sign out' is preferred, 'Log out' is banned)"
    evidence_excerpt: "'Logged out successfully'"
    recommendation: "Change to 'Signed out'."
    depends_on: null

  - finding_id: TW-073
    surface_id: api-success-message-generic
    lens: technical-writer
    severity: P3
    issue: "'If the email exists, a reset link has been sent' is a security-pattern string (intentional) but uses passive 'has been sent' — could be tightened to 'If that email exists, we sent a reset link.'"
    evidence_excerpt: "'If the email exists, a reset link has been sent'"
    recommendation: "Rewrite: 'If that email is registered, we sent a reset link.' Active voice + clearer 'registered' over 'exists'."
    depends_on: null

  - finding_id: TW-074
    surface_id: api-success-message-generic
    lens: technical-writer
    severity: P3
    issue: "Domain inconsistency: 'Chef suspended' (no successfully) / 'Delivery partner suspended' (matches) / 'User suspended' (matches) / 'Staff member deactivated' (uses 'deactivated' not 'suspended'). Four near-synonyms for the same action"
    evidence_excerpt: "'Chef suspended' / 'Staff member deactivated' / 'User suspended'"
    recommendation: "Pick one verb pair per object: chef/user/driver = 'suspended' (account-level), staff = 'deactivated' (access-level — already differentiated). Document in style guide."
    depends_on: null

  - finding_id: TW-075
    surface_id: api-success-message-generic
    lens: technical-writer
    severity: P3
    issue: "'All notifications marked as read' duplicates singular 'Marked as read' and 'Message marked as read' — three near-identical phrasings"
    evidence_excerpt: "'All notifications marked as read' / 'Marked as read' / 'Message marked as read'"
    recommendation: "Consolidate: 'All notifications read' (bulk) and 'Notification read' (single). Drop 'marked as' — implied by the verb."
    depends_on: null

  # ============================================================================
  # API — STRIPE / PAYMENT ERROR LEAKS (referenced in brief)
  # ============================================================================

  - finding_id: TW-076
    surface_id: api-success-message-generic
    lens: technical-writer
    severity: P0
    issue: "API path leak in error message: 'No Stripe account — call /chef/stripe/connect first' (handlers/stripe_connect.go:197) exposes internal route to end-user error UI. Pure ops jargon"
    evidence_excerpt: "{\"error\": \"No Stripe account — call /chef/stripe/connect first\"}"
    recommendation: "Rewrite as user-facing message: 'Connect a Stripe account before requesting a dashboard link.' Move the path-level guidance into developer docs."
    depends_on: null

  # ============================================================================
  # API — INAPP STAFF / CHEF RESPONDED (user-authored content)
  # ============================================================================

  - finding_id: TW-077
    surface_id: api-inapp-chef-responded
    lens: technical-writer
    severity: P3
    issue: "Title built by concatenation 'Chef Responded: ' + approval.Title — title case + colon + verbatim approval title. No length cap visible. Could blow past notification limits"
    evidence_excerpt: "Title: 'Chef Responded: ' + approval.Title"
    recommendation: "Title: 'Chef response' (fixed). Move approval.Title into message body, length-capped at 80 chars."
    depends_on: null

  - finding_id: TW-078
    surface_id: api-inapp-staff-invite
    lens: technical-writer
    severity: P3
    issue: "Title and message are admin-authored strings with no length cap or format validation — risk of poorly-formatted text appearing on the recipient's bell"
    evidence_excerpt: "Title: <invitation.Title> — Message: <req.Message>"
    recommendation: "Provide a default template 'Staff invitation' / 'You're invited to join the team as %role.' Allow admin override but cap length and strip HTML."
    depends_on: null

  # ============================================================================
  # WEB — CUSTOMER & CHEF TOASTS
  # ============================================================================

  - finding_id: TW-079
    surface_id: web-tx-admin-settings
    lens: technical-writer
    severity: P2
    issue: "'Settings saved successfully' / 'Failed to save settings' — 'successfully' redundant on success toast (Sec 4: past tense, ≤6 words, period)"
    evidence_excerpt: "Settings saved successfully / Failed to save settings"
    recommendation: "Success: 'Settings saved.' Error: 'Couldn\\'t save settings. Try again.'"
    depends_on: null

  - finding_id: TW-080
    surface_id: web-tx-admin-user-update
    lens: technical-writer
    severity: P2
    issue: "'User updated successfully' — 'successfully' redundant; also 'User' is banned per Sec 3 (too generic)"
    evidence_excerpt: "User updated successfully"
    recommendation: "Use specific object: 'Member updated' or 'Customer updated' depending on object type. Drop 'successfully'."
    depends_on: null

  - finding_id: TW-081
    surface_id: web-tx-catering-quotes
    lens: technical-writer
    severity: P2
    issue: "'Quote accepted! The chef has been notified.' — 8 words with exclamation in transactional toast (Rule 1 + Sec 4 ≤6 words)"
    evidence_excerpt: "Quote accepted! The chef has been notified."
    recommendation: "Split: success toast 'Quote accepted.' Body shown elsewhere if 'chef notified' needs surfacing."
    depends_on: null

  - finding_id: TW-082
    surface_id: web-tx-catering-quotes
    lens: technical-writer
    severity: P2
    issue: "Error 'Failed to accept quote. Please try again.' uses 'Please try again' filler; doesn't follow what-happened → what-to-do formula"
    evidence_excerpt: "Failed to accept quote. Please try again."
    recommendation: "Rewrite: 'Couldn\\'t accept quote. Retry, or reload the page.' Concrete next action."
    depends_on: null

  - finding_id: TW-083
    surface_id: web-tx-catering-request-submit
    lens: technical-writer
    severity: P2
    issue: "'Catering request submitted successfully!' has exclamation + 'successfully' + 5 words ok but tone wrong for transactional"
    evidence_excerpt: "Catering request submitted successfully!"
    recommendation: "Change to 'Request submitted.'"
    depends_on: null

  - finding_id: TW-084
    surface_id: web-tx-chef-catering
    lens: technical-writer
    severity: P2
    issue: "'Quote submitted successfully' / 'Failed to submit quote' — same 'successfully' redundancy"
    evidence_excerpt: "Quote submitted successfully / Failed to submit quote"
    recommendation: "'Quote sent.' / 'Couldn\\'t send quote. Try again.'"
    depends_on: null

  - finding_id: TW-085
    surface_id: web-tx-chef-menu
    lens: technical-writer
    severity: P2
    issue: "Three 'successfully' violations in the menu CRUD toast set; 'Menu item deleted' (good, no successfully) is the lone consistent variant"
    evidence_excerpt: "Menu item created successfully / Menu item updated successfully / Menu item deleted"
    recommendation: "Drop 'successfully': 'Menu item added.' / 'Menu item updated.' / 'Menu item deleted.' Use 'added' over 'created' (Sec 3 plain English)."
    depends_on: null

  - finding_id: TW-086
    surface_id: web-tx-chef-order-status
    lens: technical-writer
    severity: P3
    issue: "'Order status updated' is fine, but error 'Failed to update order status' is wordy"
    evidence_excerpt: "Order status updated / Failed to update order status"
    recommendation: "Success: 'Status changed.' Error: 'Couldn\\'t change status. Retry.'"
    depends_on: null

  - finding_id: TW-087
    surface_id: web-tx-chef-profile
    lens: technical-writer
    severity: P2
    issue: "'Profile updated successfully' / 'Failed to update profile' — 'successfully' redundancy"
    evidence_excerpt: "Profile updated successfully / Failed to update profile"
    recommendation: "'Profile saved.' / 'Couldn\\'t save profile.'"
    depends_on: null

  - finding_id: TW-088
    surface_id: web-tx-chef-social
    lens: technical-writer
    severity: P3
    issue: "Mixed tense and pattern: 'Post deleted' / 'Post created' / 'Post updated' (past-tense, good) but 'Failed to save post' breaks formula — should be what-happened → what-to-do"
    evidence_excerpt: "Post deleted / Post created / Post updated / Failed to save post"
    recommendation: "Success batch fine; replace 'created' with 'added' for plain English. Error: 'Couldn\\'t save post. Retry.'"
    depends_on: null

  - finding_id: TW-089
    surface_id: web-tx-checkout-payment-toasts
    lens: technical-writer
    severity: P1
    issue: "Payment toast set has voice drift: 'Payment successful!' (exclamation) vs 'Payment gateway is loading' (passive) vs 'Failed to initiate payment' (verb-heavy). Checkout is conversion-critical"
    evidence_excerpt: "Failed to initiate payment / Payment gateway is loading / Payment successful! / Payment verification failed..."
    recommendation: "Standardize: 'Couldn\\'t start payment. Try again.' / 'Loading payment...' / 'Payment received.' / 'Payment couldn\\'t be verified. Contact support.' Drop exclamations."
    depends_on: null

  - finding_id: TW-090
    surface_id: web-tx-chefdetail-add-to-cart
    lens: technical-writer
    severity: P2
    issue: "'Your cart has items from another chef. Clear cart first.' is 11 words across two sentences — close to the limit; 'Clear cart first' implies user must take action via toast, but toast is not actionable"
    evidence_excerpt: "Your cart has items from another chef. Clear cart first."
    recommendation: "Make this a modal (already handled by mc-menuitem-cross-chef on mobile). Toast variant: 'Cart belongs to another chef — open cart to switch.'"
    depends_on: null

  - finding_id: TW-091
    surface_id: web-tx-orderdetail-toasts
    lens: technical-writer
    severity: P2
    issue: "'Payment verification failed — please contact support' uses em-dash + 'please' filler. 6 words good but tone passive"
    evidence_excerpt: "Payment verification failed — please contact support"
    recommendation: "Rewrite: 'Payment couldn\\'t be verified. Contact support.'"
    depends_on: null

  - finding_id: TW-092
    surface_id: web-tx-orderdetail-toasts
    lens: technical-writer
    severity: P2
    issue: "'Order cancelled successfully' redundancy; sibling 'Order number copied' (good) drops successfully — inconsistent within same file"
    evidence_excerpt: "Order cancelled successfully / Order number copied"
    recommendation: "'Order cancelled.' Drop successfully."
    depends_on: null

  - finding_id: TW-093
    surface_id: web-tx-profile-2fa
    lens: technical-writer
    severity: P2
    issue: "Mixed voice in 2FA toast set: 'Two-factor authentication enabled' (formal noun) vs 'Backup codes regenerated' (verb) vs 'Key copied' (terse). 'Backup codes copied to clipboard' is 5 words OK, sibling 'Key copied' is 2 — within-set drift"
    evidence_excerpt: "Two-factor authentication enabled / Two-factor authentication disabled / Backup codes regenerated / Key copied / Backup codes copied to clipboard"
    recommendation: "Standardize on noun-form: 'Two-factor enabled.' / 'Two-factor disabled.' / 'Backup codes regenerated.' / 'Key copied.' / 'Backup codes copied.'"
    depends_on: null

  - finding_id: TW-094
    surface_id: web-tx-profile-password
    lens: technical-writer
    severity: P2
    issue: "Validation 'Passwords do not match' is good; 'Password must be at least 8 characters' could be tightened. Success 'Password updated successfully' has 'successfully' redundancy"
    evidence_excerpt: "Password must be at least 8 characters / Password updated successfully"
    recommendation: "'Password must be 8 characters or more.' / 'Password updated.'"
    depends_on: null

  - finding_id: TW-095
    surface_id: web-tx-profile-update
    lens: technical-writer
    severity: P2
    issue: "Avatar upload errors well-formatted ('File too large. Maximum 5 MB.') — but 'Invalid file type. Use JPEG, PNG or WebP.' is close to 25-word limit and follows what-happened → what-to-do correctly. Sibling 'Profile photo updated' drops 'successfully' while 'Profile updated successfully' keeps it — internal drift"
    evidence_excerpt: "Profile updated successfully / Profile photo updated"
    recommendation: "Make both consistent: 'Profile updated.' / 'Profile photo updated.'"
    depends_on: null

  - finding_id: TW-096
    surface_id: web-tx-profile-preferences-save
    lens: technical-writer
    severity: P3
    issue: "Toast 'Preferences updated' is good; error 'Failed to save preferences' uses 'failed to' filler"
    evidence_excerpt: "Preferences updated / Failed to save preferences"
    recommendation: "'Preferences updated.' / 'Couldn\\'t save preferences.'"
    depends_on: null

  - finding_id: TW-097
    surface_id: web-tx-profile-addresses
    lens: technical-writer
    severity: P2
    issue: "'Please fill in all required fields' uses 'please' filler; toast is not the right surface for field-level validation (should be inline on the field)"
    evidence_excerpt: "Please fill in all required fields"
    recommendation: "Move to inline field-level errors. If toast must exist, use: 'Fill in required fields.' (drop 'please', drop 'all').'"
    depends_on: null

  - finding_id: TW-098
    surface_id: web-tx-delivery-actions
    lens: technical-writer
    severity: P2
    issue: "Driver toasts violate Sec 2 driver tone (telegraphic ≤4 words): 'Delivery accepted!' (exclamation) / 'Delivery completed! Great job!' (8 words, two exclamations, congratulatory)"
    evidence_excerpt: "Delivery accepted! / Delivery completed! Great job!"
    recommendation: "'Accepted.' / 'Delivered.' Telegraphic, no praise — driver tone is operational."
    depends_on: null

  - finding_id: TW-099
    surface_id: web-tx-onboarding-save
    lens: technical-writer
    severity: P2
    issue: "'Your preferences have been saved!' uses possessive + exclamation; 8 words. Validation 'Please fill in the required fields' has 'please' filler and 'the required fields' should be 'required fields'"
    evidence_excerpt: "Please fill in the required fields / Your preferences have been saved! / Failed to save. Please try again."
    recommendation: "Validation: 'Fill in required fields.' Success: 'Preferences saved.' Error: 'Couldn\\'t save. Retry.'"
    depends_on: null

  - finding_id: TW-100
    surface_id: web-tx-social-feed-actions
    lens: technical-writer
    severity: P2
    issue: "'Please log in to like posts' uses banned 'log in' (Sec 3: 'sign in') + 'please' filler"
    evidence_excerpt: "Please log in to like posts / Please log in to save posts"
    recommendation: "'Sign in to like posts.' / 'Sign in to save posts.' Drops 'please'."
    depends_on: null

  # ============================================================================
  # VENDOR PORTAL — CHEF-FACING TOASTS
  # ============================================================================

  - finding_id: TW-101
    surface_id: vp-ux-menu-form-toasts
    lens: technical-writer
    severity: P2
    issue: "'Menu item created successfully' / 'Menu item updated successfully' — 'successfully' redundancy. Also 'At least one image is required' should be inline validation not a toast"
    evidence_excerpt: "Menu item created successfully / Failed to create menu item / At least one image is required / Restored your unsaved draft / Image deleted / Failed to delete image / Some images failed to upload"
    recommendation: "'Menu item added.' / 'Couldn\\'t add menu item.' / 'Add at least one image.' (move to inline) / 'Unsaved draft restored.' / 'Image deleted.' / 'Couldn\\'t delete image.' / 'Some images failed to upload — retry.'"
    depends_on: null

  - finding_id: TW-102
    surface_id: vp-ux-menu-form-category-toasts
    lens: technical-writer
    severity: P3
    issue: "'A category with this name already exists' is 8 words; 'A' opener is filler"
    evidence_excerpt: "Category \"{name}\" created / A category with this name already exists / Failed to create category"
    recommendation: "'Category \"{name}\" added.' / 'Category name already exists.' / 'Couldn\\'t add category.'"
    depends_on: null

  - finding_id: TW-103
    surface_id: vp-ux-menu-toasts
    lens: technical-writer
    severity: P2
    issue: "'N items marked available/unavailable' uses slash for two states — readers will see one or the other but copy must pick. Also 'Failed to update some items' is vague"
    evidence_excerpt: "Availability updated / Failed to update availability / Menu item deleted / Failed to delete item / N items marked available/unavailable / Failed to update some items"
    recommendation: "Use conditional: '{n} items marked available' OR '{n} items marked unavailable' based on action. Generic error: 'Couldn\\'t update {n} items.'"
    depends_on: null

  - finding_id: TW-104
    surface_id: vp-ux-notifs-respond-toasts
    lens: technical-writer
    severity: P3
    issue: "'Please enter a response' is field-validation in a toast — should be inline. 'Response sent to admin' is fine (specific recipient)"
    evidence_excerpt: "Please enter a response / Response sent to admin / Failed to send response"
    recommendation: "Move 'enter a response' to inline. Toasts: 'Response sent.' / 'Couldn\\'t send response.'"
    depends_on: null

  - finding_id: TW-105
    surface_id: vp-onb-personal-avatar-success
    lens: technical-writer
    severity: P3
    issue: "'Profile photo uploaded' is fine — 3 words, past tense, period implied"
    evidence_excerpt: "Profile photo uploaded"
    recommendation: "Keep as-is. (No change needed; flagged for completeness.)"
    depends_on: null

  - finding_id: TW-106
    surface_id: vp-onb-policies-ready-banner
    lens: technical-writer
    severity: P0
    issue: "Banner sets an SLA commitment '24-48 hours' the platform may not honour (admin-portal has no SLA enforcement visible). 28 words across two sentences exceeds vendor sentence limit (20 words)"
    evidence_excerpt: "You're all set to submit your application! / Our team will review your details and you'll be notified within 24-48 hours."
    recommendation: "If 24-48h SLA is enforced via process, keep. Otherwise: 'Our team will review your details and notify you when complete. Most reviews finish within 2 days.' (Removes hard SLA; gives soft guidance.)"
    depends_on: null

  - finding_id: TW-107
    surface_id: vp-onb-policies-ready-banner
    lens: technical-writer
    severity: P2
    issue: "'You're all set to submit your application!' has exclamation in transactional banner"
    evidence_excerpt: "You're all set to submit your application!"
    recommendation: "'You're ready to submit.' (4 words, no exclamation, accurate.)"
    depends_on: null

  - finding_id: TW-108
    surface_id: vp-onb-submit-success
    lens: technical-writer
    severity: P0
    issue: "Same 24-48h SLA in submission toast — duplicates TW-106 risk + lives in two places (single source of truth violation)"
    evidence_excerpt: "Application submitted! We'll review and get back to you within 24-48 hours."
    recommendation: "Centralize SLA text into a single constant. Then rewrite per TW-106."
    depends_on: null

  - finding_id: TW-109
    surface_id: vp-ux-orders-live-toasts
    lens: technical-writer
    severity: P3
    issue: "Same 'Failed to update order status' pattern as web-tx-chef-order-status — duplicated copy across two apps"
    evidence_excerpt: "Order status updated / Failed to update order status"
    recommendation: "Standardize: 'Status changed.' / 'Couldn\\'t change status.' Use shared toast helper."
    depends_on: null

  - finding_id: TW-110
    surface_id: vp-ux-kitchen-photo-toasts
    lens: technical-writer
    severity: P2
    issue: "9-toast set has internal drift: 'Kitchen photo uploaded' (past-tense good) / 'Kitchen setup saved successfully' (successfully redundant) / 'Maximum 5 photos allowed. Remove one first.' (10 words, two sentences ok)"
    evidence_excerpt: "Kitchen photo uploaded / Failed to upload photo / Photo removed / Failed to remove photo / Maximum 5 photos allowed. Remove one first. / File too large. Maximum 5 MB. / Invalid file type. Allowed: JPEG, PNG, WebP. / Kitchen setup saved successfully / Failed to save kitchen setup"
    recommendation: "'Kitchen photo uploaded.' / 'Couldn\\'t upload photo.' / 'Photo removed.' / 'Couldn\\'t remove photo.' / 'Max 5 photos. Remove one to add another.' / 'File too large — max 5 MB.' / 'Use JPEG, PNG, or WebP.' / 'Kitchen setup saved.' / 'Couldn\\'t save kitchen setup.'"
    depends_on: null

  - finding_id: TW-111
    surface_id: vp-ux-profile-toasts
    lens: technical-writer
    severity: P3
    issue: "'Profile photo updated' vs 'Profile updated successfully' — successfully suffix inconsistent within same file (parallels web-tx-profile-update bug)"
    evidence_excerpt: "Profile updated successfully / Profile photo updated / Cover photo updated"
    recommendation: "Drop successfully. 'Profile updated.' / 'Profile photo updated.' / 'Cover photo updated.'"
    depends_on: null

  - finding_id: TW-112
    surface_id: vp-ux-profile-doc-toasts
    lens: technical-writer
    severity: P3
    issue: "'{label} uploaded' is good; 'Upload failed' is bare — doesn't follow what-happened → what-to-do"
    evidence_excerpt: "{label} uploaded / Upload failed"
    recommendation: "'{label} uploaded.' / 'Couldn\\'t upload {label}. Retry.'"
    depends_on: null

  - finding_id: TW-113
    surface_id: vp-ux-settings-payout-toasts
    lens: technical-writer
    severity: P3
    issue: "Same 'successfully' inconsistency: 'Payout details saved' vs 'Settings saved' — actually consistent here, all four drop 'successfully' (good). Errors use 'Failed to' filler"
    evidence_excerpt: "Payout details saved / Failed to save payout details / Settings saved / Failed to save settings"
    recommendation: "Keep success. Errors: 'Couldn\\'t save payout details.' / 'Couldn\\'t save settings.'"
    depends_on: null

  - finding_id: TW-114
    surface_id: vp-ux-settings-stripe-toasts
    lens: technical-writer
    severity: P2
    issue: "'Failed to start Stripe onboarding' uses 'failed to' filler; 'Failed to resume onboarding' is similar. 'Payment provider updated' good"
    evidence_excerpt: "Failed to start Stripe onboarding / Failed to resume onboarding / Payment provider updated / Failed to update provider"
    recommendation: "'Couldn\\'t start Stripe setup.' / 'Couldn\\'t resume setup.' / 'Provider updated.' / 'Couldn\\'t update provider.'"
    depends_on: null

  # ============================================================================
  # DELIVERY PORTAL — DRIVER TOASTS
  # ============================================================================

  - finding_id: TW-115
    surface_id: dp-tx-toast-online
    lens: technical-writer
    severity: P3
    issue: "'You are now online' / 'You are now offline' — 4 words OK for driver (telegraphic limit) but 'You are now' is filler. Driver tone wants 'Online.' / 'Offline.'"
    evidence_excerpt: "You are now online / You are now offline"
    recommendation: "'Online.' / 'Offline.' (1 word, per driver telegraphic rule)"
    depends_on: null

  - finding_id: TW-116
    surface_id: dp-tx-toast-status-updated
    lens: technical-writer
    severity: P3
    issue: "'Status updated' is fine for driver — meets telegraphic rule"
    evidence_excerpt: "Status updated"
    recommendation: "Keep."
    depends_on: null

  - finding_id: TW-117
    surface_id: dp-tx-toast-accept-success
    lens: technical-writer
    severity: P2
    issue: "'Delivery accepted!' exclamation in driver toast — Rule 1 + driver tone is operational"
    evidence_excerpt: "Delivery accepted!"
    recommendation: "'Accepted.' (1 word, no exclamation)"
    depends_on: null

  - finding_id: TW-118
    surface_id: dp-tx-toast-partner-verified
    lens: technical-writer
    severity: P2
    issue: "'Partner verified successfully' — 'successfully' redundancy on admin toast"
    evidence_excerpt: "Partner verified successfully"
    recommendation: "'Partner verified.'"
    depends_on: null

  - finding_id: TW-119
    surface_id: dp-tx-toast-step5-submitted
    lens: technical-writer
    severity: P2
    issue: "'Application submitted successfully!' — exclamation + successfully on driver onboarding"
    evidence_excerpt: "Application submitted successfully!"
    recommendation: "'Application submitted.'"
    depends_on: null

  - finding_id: TW-120
    surface_id: dp-tx-toast-step4-saved
    lens: technical-writer
    severity: P3
    issue: "'Plan selected successfully' — 'successfully' redundancy"
    evidence_excerpt: "Plan selected successfully"
    recommendation: "'Plan selected.'"
    depends_on: null

  - finding_id: TW-121
    surface_id: dp-tx-toast-invite-created
    lens: technical-writer
    severity: P2
    issue: "'Invitation created successfully' — successfully redundancy in admin toast"
    evidence_excerpt: "Invitation created successfully"
    recommendation: "'Invitation created.'"
    depends_on: null

  - finding_id: TW-122
    surface_id: dp-tx-toast-invite-copied
    lens: technical-writer
    severity: P3
    issue: "'Invite URL copied to clipboard' — 'to clipboard' is filler (always to clipboard); 5 words could be 3"
    evidence_excerpt: "Invite URL copied to clipboard"
    recommendation: "'Invite link copied.' (use 'link' for end-user friendliness)"
    depends_on: null

  # ============================================================================
  # ADMIN PORTAL — ADMIN TOASTS
  # ============================================================================

  - finding_id: TW-123
    surface_id: ap-approvaldetail-toasts
    lens: technical-writer
    severity: P2
    issue: "'More information requested from chef' — admin tone says direct ('Direct: Approve, Suspend, Audit'). 'requested from' is passive"
    evidence_excerpt: "More information requested from chef"
    recommendation: "'Info requested from chef.' (4 words, direct, admin tone)"
    depends_on: null

  - finding_id: TW-124
    surface_id: ap-providercreate-toasts
    lens: technical-writer
    severity: P2
    issue: "'Provider created successfully' — successfully redundancy"
    evidence_excerpt: "Provider created successfully / Failed to create provider"
    recommendation: "'Provider added.' / 'Couldn\\'t add provider.'"
    depends_on: null

  - finding_id: TW-125
    surface_id: ap-providerdetail-toasts
    lens: technical-writer
    severity: P3
    issue: "'Connection successful ({ms}ms)' good with metric; 'Connection failed: {err}' raw error injection — admin can see raw stack messages"
    evidence_excerpt: "Connection successful ({ms}ms) / Connection failed: {err}"
    recommendation: "'Connected ({ms}ms).' / 'Connection failed: {err}.' Sanitize err to first sentence only."
    depends_on: null

  - finding_id: TW-126
    surface_id: ap-notifsettings-toasts
    lens: technical-writer
    severity: P3
    issue: "'Preference saved' / 'Failed to save' — error too generic (what preference?)"
    evidence_excerpt: "Preference saved / Failed to save"
    recommendation: "'Preference saved.' / 'Couldn\\'t save preference.'"
    depends_on: null

  - finding_id: TW-127
    surface_id: ap-secsettings-toasts
    lens: technical-writer
    severity: P2
    issue: "Long mixed set: '2FA enabled successfully' (redundancy) / 'Session policy updated' (good) / 'All other sessions revoked. You may need to sign in again.' (12 words two sentences, fine) / 'Invalid code' (good) / 'Enroll failed' (bare)"
    evidence_excerpt: "2FA enabled successfully / 2FA disabled / Enforcement setting updated / Invalid code / Enroll failed"
    recommendation: "'Two-factor enabled.' / 'Two-factor disabled.' / 'Enforcement updated.' / 'Invalid code.' / 'Enrollment failed. Retry.'"
    depends_on: null

  - finding_id: TW-128
    surface_id: ap-settings-payment-feedback
    lens: technical-writer
    severity: P2
    issue: "'Saved, but Razorpay rejected the keys: {err}' injects raw gateway error — admin sees vendor terminology verbatim. Same for Stripe variant"
    evidence_excerpt: "Saved, but Razorpay rejected the keys: {err} / Saved, but Stripe rejected the keys: {err}"
    recommendation: "Wrap err in a label: 'Keys saved but Razorpay rejected them. Reason: {err}.' If err has technical jargon, log it and surface a human summary."
    depends_on: null

  - finding_id: TW-129
    surface_id: ap-staffdetail-toasts
    lens: technical-writer
    severity: P2
    issue: "'Role updated successfully' has redundancy; 'Staff member deactivated' uses 'member' which is fine; 'Failed to deactivate staff member' is wordy"
    evidence_excerpt: "Role updated successfully / Staff member deactivated / Failed to deactivate staff member"
    recommendation: "'Role updated.' / 'Member deactivated.' / 'Couldn\\'t deactivate member.'"
    depends_on: null

  - finding_id: TW-130
    surface_id: ap-staff-invite-success-title
    lens: technical-writer
    severity: P2
    issue: "'Invitation created successfully!' — exclamation + successfully in admin success title"
    evidence_excerpt: "Invitation created successfully!"
    recommendation: "'Invitation created.'"
    depends_on: null

  - finding_id: TW-131
    surface_id: ap-staff-invite-success-body
    lens: technical-writer
    severity: P3
    issue: "'Share this link with the invitee to complete their registration.' — 'invitee' is jargon; 'their registration' is corporate-speak"
    evidence_excerpt: "Share this link with the invitee to complete their registration."
    recommendation: "'Share this link so they can finish signing up.' (Plain English per Rule 2.)"
    depends_on: null

  # ============================================================================
  # MOBILE CUSTOMER — ALERTS
  # ============================================================================

  - finding_id: TW-132
    surface_id: mc-profile-save-success
    lens: technical-writer
    severity: P3
    issue: "Three near-identical save toasts in one component: 'Saved' / 'Profile updated successfully.' / 'Cuisine preferences updated.' — sample size for internal drift"
    evidence_excerpt: "Saved / Profile updated successfully. / Cuisine preferences updated."
    recommendation: "'Saved.' / 'Profile saved.' / 'Cuisines saved.' Three terse, parallel forms."
    depends_on: null

  - finding_id: TW-133
    surface_id: mc-profile-logout-confirm
    lens: technical-writer
    severity: P0
    issue: "Title 'Log out' uses banned compound 'Log out' (Sec 3: 'Sign out'). Button 'Log Out' also has title-case inconsistency"
    evidence_excerpt: "Log out / Are you sure you want to log out? / Cancel / Log Out"
    recommendation: "Title 'Sign out'. Body 'Sign out of Home Chef on this device?' Buttons 'Cancel' / 'Sign out'. Fix both Sec 3 (verb) and case mismatch."
    depends_on: null

  - finding_id: TW-134
    surface_id: mc-catering-submit-success
    lens: technical-writer
    severity: P2
    issue: "'Request Submitted!' — exclamation + title case in transactional alert"
    evidence_excerpt: "Request Submitted! / Chefs will review and send quotes."
    recommendation: "'Request submitted.' / 'Chefs will review and send quotes.'"
    depends_on: null

  - finding_id: TW-135
    surface_id: mc-menuitem-cross-chef
    lens: technical-writer
    severity: P2
    issue: "Title 'Replace Cart?' is title case + question; modal body 'You have items from another chef. Replace cart?' duplicates the question. Modal subtitle should explain consequence (Sec 4)"
    evidence_excerpt: "Replace Cart? / You have items from another chef. Replace cart? / Cancel / Replace"
    recommendation: "Title: 'Replace cart?' Body: 'Your cart belongs to another chef. Adding this item will clear it.' Buttons 'Cancel' / 'Replace cart'."
    depends_on: null

  # ============================================================================
  # MOBILE VENDOR — ALERTS
  # ============================================================================

  - finding_id: TW-136
    surface_id: mv-onb-pending-submitted-title
    lens: technical-writer
    severity: P2
    issue: "'Application Submitted!' uses title case + exclamation as a display title"
    evidence_excerpt: "Application Submitted!"
    recommendation: "'Application submitted'. Drop exclamation; sentence case."
    depends_on: null

  - finding_id: TW-137
    surface_id: mv-onb-pending-submitted-body
    lens: technical-writer
    severity: P1
    issue: "17 words across two sentences — exceeds vendor 20-word limit only marginally, BUT 'within 24-48 hours' duplicates the SLA promise from TW-106/TW-108. Three sources of truth for one SLA"
    evidence_excerpt: "Our team will review your application within 24-48 hours. We will notify you once the review is complete."
    recommendation: "'Our team will review your application and notify you when complete.' (Drop hard SLA; centralize in one place.)"
    depends_on: null

  - finding_id: TW-138
    surface_id: mv-onb-pending-rejected-title
    lens: technical-writer
    severity: P3
    issue: "'Application Not Approved' is title case but the soft phrasing is good — already softer than 'Rejected'"
    evidence_excerpt: "Application Not Approved"
    recommendation: "Sentence case: 'Application not approved'. Keep softer phrasing."
    depends_on: null

  - finding_id: TW-139
    surface_id: mv-onb-pending-rejected-body
    lens: technical-writer
    severity: P3
    issue: "'Please review the feedback and resubmit your application.' uses 'please' filler; 8 words OK"
    evidence_excerpt: "Please review the feedback and resubmit your application."
    recommendation: "'Review the feedback and resubmit.' (5 words, no filler)"
    depends_on: null

  - finding_id: TW-140
    surface_id: mv-reviewdetail-success-title
    lens: technical-writer
    severity: P3
    issue: "'Reply Sent' — title case in alert title"
    evidence_excerpt: "Reply Sent"
    recommendation: "'Reply sent'."
    depends_on: null

  - finding_id: TW-141
    surface_id: mv-undo-accepted
    lens: technical-writer
    severity: P3
    issue: "'Order accepted' / 'Order rejected' good 2-word snackbars — keep"
    evidence_excerpt: "Order accepted / Order rejected"
    recommendation: "Keep."
    depends_on: null

  # ============================================================================
  # MOBILE DELIVERY — DRIVER ALERTS
  # ============================================================================

  - finding_id: TW-142
    surface_id: md-trx-006
    lens: technical-writer
    severity: P0
    issue: "'Logout' as alert title uses banned 'Logout' (one-word) per Sec 3 — should be 'Sign out'"
    evidence_excerpt: "Logout"
    recommendation: "'Sign out'."
    depends_on: null

  - finding_id: TW-143
    surface_id: md-trx-007
    lens: technical-writer
    severity: P0
    issue: "'Are you sure you want to logout?' uses banned compound 'logout' and weak modal subtitle (Sec 4: explain consequence, not 'are you sure')"
    evidence_excerpt: "Are you sure you want to logout?"
    recommendation: "'Sign out of Home Chef Delivery? You'll need to sign in again to receive deliveries.' (Explains consequence.)"
    depends_on: null

  - finding_id: TW-144
    surface_id: md-trx-011
    lens: technical-writer
    severity: P2
    issue: "'Validation' as an alert title is meaningless to a driver — internal dev label leaked to UI"
    evidence_excerpt: "Validation"
    recommendation: "'Missing info' or use no title and put the message body 'Name, phone, and city are required.' directly."
    depends_on: null

  - finding_id: TW-145
    surface_id: md-trx-013
    lens: technical-writer
    severity: P2
    issue: "'Success' as alert title is generic and content-free; same problem as 'Validation'"
    evidence_excerpt: "Success"
    recommendation: "Replace with specific verb: 'Saved' or omit title and use just the body 'Profile updated.'"
    depends_on: null

  - finding_id: TW-146
    surface_id: md-trx-014
    lens: technical-writer
    severity: P3
    issue: "'Profile updated successfully.' — 'successfully' redundancy on driver alert body"
    evidence_excerpt: "Profile updated successfully."
    recommendation: "'Profile updated.'"
    depends_on: null

  - finding_id: TW-147
    surface_id: md-trx-015
    lens: technical-writer
    severity: P3
    issue: "'Failed to update profile. Please try again.' — 'please' filler; matches the same anti-pattern in web/vp"
    evidence_excerpt: "Failed to update profile. Please try again."
    recommendation: "'Couldn\\'t update profile. Retry.'"
    depends_on: null

  - finding_id: TW-148
    surface_id: md-trx-009
    lens: technical-writer
    severity: P2
    issue: "'Visit the web portal to manage your subscription.' is 8 words OK but cross-app dependency is a UX smell — pushes driver out of app for billing"
    evidence_excerpt: "Visit the web portal to manage your subscription."
    recommendation: "Either build native subscription view (BA scope) or rewrite for clarity: 'Manage subscription at fleet.homechef.in.' (Includes the actual URL; drops vague 'web portal'.)"
    depends_on: null

  - finding_id: TW-149
    surface_id: md-trx-010
    lens: technical-writer
    severity: P3
    issue: "'Contact support at support@homechef.in to request account deletion.' is 9 words OK but support email hardcoded in client — single source of truth"
    evidence_excerpt: "Contact support at support@homechef.in to request account deletion."
    recommendation: "Centralize support email in config. Copy itself OK as-is."
    depends_on: null

  - finding_id: TW-150
    surface_id: md-trx-002
    lens: technical-writer
    severity: P0
    issue: "Background location rationale is 31 words — far exceeds driver 12-word telegraphic limit. This is a critical pre-OS-prompt P0 safety string; needs care but also tightening"
    evidence_excerpt: "To keep customers updated on their delivery, HomeChef Delivery needs to track your location while you are on an active delivery — even when the app is in the background."
    recommendation: "Two-line rewrite preserving meaning: 'We track your location during active deliveries so customers see live ETA. Tracking stops automatically when you complete the delivery.' (~22 words combined.) Coordinate with Legal lens — this string must match the iOS Info.plist NSLocationAlwaysAndWhenInUseUsageDescription."
    depends_on: null

  - finding_id: TW-151
    surface_id: md-trx-003
    lens: technical-writer
    severity: P2
    issue: "'Battery usage is minimised by only tracking every 15 seconds.' is 10 words but mixes UK/US spelling ('minimised') — repo otherwise uses US English"
    evidence_excerpt: "Location tracking stops automatically when the delivery is completed. Battery usage is minimised by only tracking every 15 seconds."
    recommendation: "Pick one English variant repo-wide (recommend en-IN aligns with UK 'minimised'). If en-US, change to 'minimized'."
    depends_on: null

  - finding_id: TW-152
    surface_id: md-trx-004
    lens: technical-writer
    severity: P2
    issue: "'Allow Background Location' is title case 3 words — should be sentence case 'Allow background location'"
    evidence_excerpt: "Allow Background Location"
    recommendation: "'Allow background location'."
    depends_on: null

  - finding_id: TW-153
    surface_id: md-trx-005
    lens: technical-writer
    severity: P3
    issue: "'Not Now' is title case — should be sentence case 'Not now'"
    evidence_excerpt: "Not Now"
    recommendation: "'Not now'."
    depends_on: null

  - finding_id: TW-154
    surface_id: md-trx-019
    lens: technical-writer
    severity: P3
    issue: "'Failed to send invitation. Please try again.' uses 'please' filler — same anti-pattern across apps"
    evidence_excerpt: "Failed to send invitation. Please try again."
    recommendation: "'Couldn\\'t send invitation. Retry.'"
    depends_on: null

  - finding_id: TW-155
    surface_id: md-trx-018
    lens: technical-writer
    severity: P2
    issue: "'You do not have permission to invite staff.' is 8 words OK but 'You do not have permission' is jargon-heavy. Driver-friendly form needs simpler phrasing"
    evidence_excerpt: "You do not have permission to invite staff."
    recommendation: "'Only the partner owner can invite staff.' (Explains who can.)"
    depends_on: null

  - finding_id: TW-156
    surface_id: md-trx-016
    lens: technical-writer
    severity: P3
    issue: "'Failed to upload photo. Please try again.' — same 'please' filler pattern"
    evidence_excerpt: "Failed to upload photo. Please try again."
    recommendation: "'Couldn\\'t upload photo. Retry.'"
    depends_on: null

  - finding_id: TW-157
    surface_id: md-trx-008
    lens: technical-writer
    severity: P3
    issue: "Alert title 'Subscription' is bare noun — works as a context label but feels lifeless. Pairs with TW-148 body"
    evidence_excerpt: "Subscription"
    recommendation: "'Manage subscription' or align with body fix from TW-148."
    depends_on: null

  # ============================================================================
  # CROSS-SURFACE: SYSTEMIC PATTERNS (not tied to one surface_id)
  # ============================================================================

  - finding_id: TW-158
    surface_id: api-success-message-generic
    lens: technical-writer
    severity: P2
    issue: "Repository-wide pattern: 27 API success messages and 25+ frontend toasts spell 'successfully' at the end of a past-tense verb — pure redundancy, ~52 places to fix"
    evidence_excerpt: "Across handlers/*.go and apps/*/features/*/pages/*.tsx — count of 'successfully' as toast/api suffix"
    recommendation: "One-time sweep: 'X-ed successfully' → 'X-ed' across the repo. Add a lint rule (custom ESLint + a Go vet script) to catch future additions."
    depends_on: null

  - finding_id: TW-159
    surface_id: api-success-message-generic
    lens: technical-writer
    severity: P2
    issue: "Repository-wide pattern: ~40+ instances of 'Failed to X. Please try again.' frontend toasts — 'please' filler + non-actionable 'try again'"
    evidence_excerpt: "Across vendor/admin/delivery/web apps — count of 'Failed to'/'Please try again' as toast text"
    recommendation: "Sweep: 'Failed to X. Please try again.' → 'Couldn\\'t X. Retry.' Same lint-rule recommendation as TW-158."
    depends_on: null

  - finding_id: TW-160
    surface_id: api-email-order-confirm
    lens: technical-writer
    severity: P1
    issue: "All 8 branded emails hardcode the brand name 'Fe3dr' in subject + footer + URLs. No single source of truth for the brand string. Switching brand requires touching every template"
    evidence_excerpt: "Subject: 'Welcome to Fe3dr!' / 'Verify your email — Fe3dr' / 'Reset your password — Fe3dr' / footer 'Fe3dr by HomeChef' / URLs 'https://fe3dr.com'"
    recommendation: "Extract brand name + URLs into a config struct (BrandName, BrandURL, BrandVendorURL, BrandDeliveryURL). Reference in all templates. Aligns with TW-001/TW-002 single-brand recommendation."
    depends_on: null
```

## Legal findings

```yaml
# Legal lens findings — Home Chef Content Audit — TRANSACTIONAL slice
# Scope: India regulatory exposure (DPDP Act 2023, FSSAI, RBI Payment Aggregator, GST, TRAI/DLT) + generic best-practice
# IMPORTANT: All findings carry depends_on: "needs lawyer review" — this audit is not legal advice.
# Slice rows: 154 transactional inventory entries (web, vendor-portal, delivery-portal, admin-portal, mobile-customer, mobile-vendor, mobile-delivery, api).

findings:
  # ============================================================================
  # ORDER CONFIRMATION EMAILS — GST INVOICE REQUIREMENTS (CGST Act §31, Rule 46)
  # ============================================================================

  - finding_id: LEG-TX-001
    surface_id: api-email-order-confirm
    lens: legal
    severity: P0
    issue: "Order confirmation email is missing every element of a GST tax invoice — no GSTIN, no HSN/SAC code (SAC 996331 for restaurant/food supply), no rate-wise tax breakup, no place-of-supply, no invoice serial number, no chef's legal name/address"
    evidence_excerpt: "\"Order Confirmed! Your order #%s has been placed successfully. Your home chef is preparing your meal!\" — total formatted %.2f with no tax breakdown"
    recommendation: "Where the platform or chef is registered under GST, the order confirmation acts as the tax invoice and must contain every Rule 46 particular. Where the chef is a composition or unregistered supplier, the email must clearly state 'Bill of Supply — no GST charged' and the platform (as e-commerce operator under §52) may still owe TCS disclosures. Split the email template to support: (a) registered-chef invoice, (b) composition/unregistered bill of supply, (c) platform-issued service-fee invoice."
    citation: "CGST Act 2017 §31; CGST Rule 46; §52 (TCS by e-commerce operator); SAC 996331"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-002
    surface_id: api-email-order-confirm
    lens: legal
    severity: P0
    issue: "Currency rendered as raw ₹%.2f with no rounding rule or breakup; no indication of whether price is 'tax inclusive' or 'tax extra'"
    evidence_excerpt: "Currency hardcoded ₹. Total formatted %.2f."
    recommendation: "CGST Rule 46 requires the invoice to disclose taxable value, applicable rate, and tax amount separately. Even on a B2C food order, the customer email should show Subtotal / GST 5% (or 18% if applicable) / Total. Also state 'inclusive of GST' or 'exclusive of GST' on every price line to satisfy the Legal Metrology (Packaged Commodities) and Consumer Protection (E-Commerce) Rules 2020 Rule 6 disclosure duties."
    citation: "CGST Rule 46; Consumer Protection (E-Commerce) Rules 2020 Rule 6(5); Legal Metrology Act 2009"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-003
    surface_id: api-email-order-confirm
    lens: legal
    severity: P0
    issue: "Order confirmation email does not display chef's FSSAI licence number — FSSAI mandates display of the FSSAI licence/registration number on all e-commerce food order communications"
    evidence_excerpt: "MISSING: tax/GST line, chef name, delivery address, ETA, refund/cancellation policy, allergen disclaimer"
    recommendation: "FSSAI Direction dated 8 Feb 2021 (E-Commerce FBO Guidelines) requires the FSSAI licence number of the food business operator to be displayed on bills/receipts/invoices and on the e-commerce platform. Embed the chef's FSSAI number in every order confirmation and order-status email, plus the platform's own FSSAI registration as an e-commerce FBO."
    citation: "FSSAI Direction F. No. 15(31)2020/FoSCoRIS/RCD/FSSAI dated 8 Feb 2021; Food Safety and Standards (Licensing and Registration of Food Businesses) Regulations 2011"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-004
    surface_id: api-email-order-confirm
    lens: legal
    severity: P0
    issue: "No allergen disclosure or 'check label before consuming' callout on the order confirmation — for prepared food sold via e-commerce FBOs, FSSAI requires allergen information to be communicated with the food/order"
    evidence_excerpt: "MISSING: ...allergen disclaimer"
    recommendation: "Append a per-item allergen line (or a standardised allergen footer linking to dish-level disclosure) and a 'If you have allergies, contact the chef before consuming' notice. Track which of the 8 statutorily-recognised allergens (FSSAI Labelling Regulation 2.2.2.4) each item contains."
    citation: "FSS (Labelling and Display) Regulations 2020 Reg. 5(1)(d); FSSAI E-Commerce Direction 2021"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-005
    surface_id: api-email-order-confirm
    lens: legal
    severity: P0
    issue: "No refund/cancellation policy summary on the order confirmation — RBI Payment Aggregator Master Direction and Consumer Protection (E-Commerce) Rules require refund timeline disclosure at the transaction touchpoint"
    evidence_excerpt: "MISSING: ...refund/cancellation policy"
    recommendation: "Add a one-line summary at the bottom of the order confirmation: 'Cancellations before the chef accepts: full refund within 7 working days. After acceptance: see Cancellation Policy.' Link to the full policy. RBI PA framework expects refund timelines stated up-front; CPER 2020 Rule 5(3)(g) requires the e-commerce entity to display its return/refund policy."
    citation: "RBI Payment Aggregator Master Direction 17 Mar 2020 (updated 2024); Consumer Protection (E-Commerce) Rules 2020 Rule 5(3)(g)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-006
    surface_id: api-email-order-confirm
    lens: legal
    severity: P1
    issue: "Order confirmation does not include the delivery address or estimated time — increases dispute exposure when delivery is contested and weakens the audit trail required by CPER 2020"
    evidence_excerpt: "MISSING: ...delivery address, ETA"
    recommendation: "Echo the delivery address and the committed delivery/pickup window into the order confirmation. CPER 2020 Rule 5(3)(h) requires display of an effective grievance redressal mechanism and shipment/delivery details; without the captured address, disputes about misdelivery default against the platform."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 5(3)(h)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-007
    surface_id: api-email-order-confirm
    lens: legal
    severity: P1
    issue: "No grievance-officer contact (name, email, phone, address) in the transactional email footer"
    evidence_excerpt: "MISSING — no footer disclosure observed in template"
    recommendation: "Add a footer block with the grievance officer's name, email, and phone, and the platform's registered address. Required by Consumer Protection (E-Commerce) Rules 2020 Rule 4(5) (grievance officer display) and IT Rules 2021 Rule 3(2). DPDP Act §10 will also require Data Protection Officer contact once the Act fully operationalises."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 4(5); IT (Intermediary Guidelines) Rules 2021 Rule 3(2); DPDP Act 2023 §10"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-008
    surface_id: api-email-order-confirm
    lens: legal
    severity: P1
    issue: "Email does not state the seller of record (chef name) — under CPER 2020 a marketplace must clearly disclose seller details to the buyer"
    evidence_excerpt: "MISSING: ...chef name"
    recommendation: "Inject the chef's legal/display name and registered business address into the confirmation. CPER 2020 Rule 6(5) requires the marketplace to display 'details about the sellers offering goods and services, including the name of their business, geographic address, customer care contact and a ticket number for grievance redressal'."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 6(5)"
    depends_on: "needs lawyer review"

  # ============================================================================
  # ORDER STATUS EMAILS — CONSISTENCY + FSSAI/RBI CARRY-THROUGH
  # ============================================================================

  - finding_id: LEG-TX-009
    surface_id: api-email-order-status-cancelled
    lens: legal
    severity: P0
    issue: "Cancellation email omits refund timeline, refund channel, and grievance pathway — RBI PA rules require explicit refund timeline communicated to the customer at cancellation"
    evidence_excerpt: "Subject: \"Order #%s — Your order has been cancelled\" — NO refund timeline, NO who-to-contact line"
    recommendation: "Cancellation emails MUST state: (a) whether a refund is initiated, (b) the refund timeline (RBI PA framework expects refunds processed within agreed timelines, typically T+7 working days for digital refunds), (c) the refund channel (original payment method), (d) a grievance contact if the refund does not arrive."
    citation: "RBI Payment Aggregator Master Direction 17 Mar 2020 §8; Consumer Protection (E-Commerce) Rules 2020 Rule 5(3)(g)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-010
    surface_id: api-push-order-cancelled-both
    lens: legal
    severity: P1
    issue: "Cancellation push notification body 'Order has been cancelled' is identical for customer and chef and sets no refund expectation"
    evidence_excerpt: "Title: \"Order Cancelled\" — Body: \"Order has been cancelled\". Same body sent to BOTH customer and chef. No refund expectation set."
    recommendation: "Differentiate customer push ('Order cancelled. Refund initiated — typically T+7 days.') from chef push ('Order cancelled by customer.'). Single shared body violates CPER 2020 disclosure duty toward the customer specifically."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 5(3)(g); RBI Payment Aggregator Master Direction"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-011
    surface_id: api-inapp-order-cancelled-both
    lens: legal
    severity: P1
    issue: "In-app cancellation notification has same generic body as push and omits refund/grievance info"
    evidence_excerpt: "Title: \"Order Cancelled\" — Message: \"Order has been cancelled\""
    recommendation: "In-app cancellation notification must include the refund status and link to grievance contact (Consumer Protection (E-Commerce) Rules 2020 Rule 4(5))."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 4(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-012
    surface_id: api-email-order-status-confirmed
    lens: legal
    severity: P1
    issue: "Status emails (confirmed/preparing/ready/picked up/on way/delivered) use one templated wrapper with no FSSAI/grievance footer or invoice link back to the order"
    evidence_excerpt: "OrderStatusUpdateHTML — generic emoji-prefixed bodies"
    recommendation: "Every transactional status email should retain the grievance-officer footer (CPER 2020 Rule 4(5)), an unsubscribe-not-applicable-for-transactional notice (TRAI TCCCPR distinction), and a link to the underlying order/invoice. Currently the wrapper omits these. Also re-confirm the email is correctly classified as 'service/transactional' for DLT purposes if forwarded via SMS."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 4(5); TRAI TCCCPR 2018"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-013
    surface_id: api-email-order-status-delivered
    lens: legal
    severity: P1
    issue: "Delivery confirmation email does not prompt rating/complaint pathway or restate the 7-day refund/dispute window — material for FSSAI food-safety complaint window and consumer-redress timelines"
    evidence_excerpt: "Subject: \"Order #%s — Your order has been delivered. Enjoy your meal!\""
    recommendation: "Add a 'Something wrong with your order? Report it within 24 hours' line that opens a complaints flow. Required to operationalise CPER 2020 Rule 4(5) grievance access and FSSAI food-safety reporting (Food Safety Connect 1800-112-100)."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 4(5); FSSAI Food Safety Connect"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-014
    surface_id: api-push-order-update-customer-deeplink
    lens: legal
    severity: P2
    issue: "Push and email statuses use three different sets of wording for the same lifecycle event ('Your order is being prepared' vs 'Your chef is preparing your meal' vs 'Your order is now being prepared') — inconsistency between channels can be used by counterparties to contest the status timeline"
    evidence_excerpt: "THIRD set of status labels in the codebase, distinct from email's statusLabels map and from getOrderStatusMessage"
    recommendation: "Unify status copy in a single source of truth. From a contract-evidence standpoint, mismatched wording between push/email/in-app can be cited as ambiguity in a dispute; consistent strings strengthen the platform's defence on order timeline."
    citation: "Indian Contract Act 1872 §29 (certainty of terms); evidentiary best-practice"
    depends_on: "needs lawyer review"

  # ============================================================================
  # PUSH/SMS — TRAI TCCCPR + DLT REGISTRATION
  # ============================================================================

  - finding_id: LEG-TX-015
    surface_id: api-push-order-created-chef
    lens: legal
    severity: P1
    issue: "Push notifications are not tagged with their TCCCPR classification (transactional / service / promotional) in the codebase; risk of misclassifying promotional content as transactional"
    evidence_excerpt: "Title: \"New Order Received\" — Body: \"You have a new order waiting to be prepared!\""
    recommendation: "Annotate every push template with its TCCCPR header category. While push notifications themselves are not directly covered by TRAI TCCCPR (which targets SMS/voice), any fallback to SMS through DLT must use a registered template under the correct header category. Mis-categorisation (e.g. sending a promo as transactional) risks DLT scrubbing and operator fines."
    citation: "TRAI Telecom Commercial Communications Customer Preference Regulations 2018; DLT framework"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-016
    surface_id: api-push-user-welcome
    lens: legal
    severity: P0
    issue: "Welcome push ('Welcome to HomeChef!') and welcome email ('Welcome to Fe3dr!') are sent without an explicit consent record for marketing-style messaging — DPDP Act §6 requires consent to be specific, informed, unambiguous; bundling welcome with marketing CTAs ('Discover amazing home-cooked meals near you!') needs separate opt-in"
    evidence_excerpt: "Title: \"Welcome to HomeChef!\" — Message: \"Thank you for joining HomeChef. Discover amazing home-cooked meals near you!\""
    recommendation: "Capture an explicit marketing-comms opt-in at signup (DPDP §6 + TCCCPR 2018 commercial-comms distinction). If the welcome message contains a marketing CTA, it is not purely transactional and must respect the user's preference. Split welcome (transactional, allowed) from any 'discover' nudges (promotional, opt-in only)."
    citation: "DPDP Act 2023 §6(1); TRAI TCCCPR 2018"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-017
    surface_id: api-email-welcome
    lens: legal
    severity: P0
    issue: "Welcome email lacks a DPDP §5 notice of data processing — required to be served at or before consent is sought"
    evidence_excerpt: "\"Welcome to Fe3dr, %s! We're thrilled to have you. Discover amazing home-cooked meals from talented local chefs...\""
    recommendation: "DPDP Act §5 requires a Notice at the time of consent (signup), describing personal data being processed, purposes, the manner of withdrawal, complaint mechanism with the Board, and grievance officer. Either ship this notice during signup (preferred) or attach a summary block at the foot of the welcome email referencing the full Privacy Notice."
    citation: "DPDP Act 2023 §5; §6(3) (withdrawal of consent)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-018
    surface_id: api-email-welcome
    lens: legal
    severity: P1
    issue: "Welcome email contains marketing copy and a 'Start Exploring' CTA but no unsubscribe/preferences link"
    evidence_excerpt: "Lists 4 capabilities (browse, order, track, rate). CTA \"Start Exploring\" → https://fe3dr.com"
    recommendation: "Marketing-tinged emails must include a one-click unsubscribe and a preferences link (DPDP §6(4) withdrawal of consent must be as easy as giving consent). If treated as 'purely transactional', remove the marketing CTAs."
    citation: "DPDP Act 2023 §6(4)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-019
    surface_id: api-push-user-welcome
    lens: legal
    severity: P1
    issue: "Brand drift between welcome push ('HomeChef') and welcome email ('Fe3dr') — exposes the platform to a 'misleading representation' claim under Consumer Protection Act 2019 §2(28)"
    evidence_excerpt: "BRAND DRIFT: says \"HomeChef\", but email welcome says \"Welcome to Fe3dr!\". Two products to a new user"
    recommendation: "Pick one legal trade name. Two brand names appearing to the same customer in transactional channels can be argued as 'misleading representation' under CPA 2019 §2(28); even short of that, it undermines the platform's identification under CPER 2020 Rule 4(1) (e-commerce entity must publish its legal name)."
    citation: "Consumer Protection Act 2019 §2(28); Consumer Protection (E-Commerce) Rules 2020 Rule 4(1)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-020
    surface_id: api-email-account-reminder
    lens: legal
    severity: P1
    issue: "Account-reminder email subject mixes brand names ('Fe3dr HomeChef') — brand drift in transactional security communication"
    evidence_excerpt: "Subject: \"You already have a Fe3dr HomeChef account\" — Brand drift: subject says \"Fe3dr HomeChef\" but other emails just say \"Fe3dr\""
    recommendation: "Standardise sender brand. Security-relevant emails referencing a confused brand reduce phishing-resistance: users cannot distinguish legitimate platform mail from spoof. Pick the legal trade name."
    citation: "Consumer Protection Act 2019 §2(28); IT Act 2000 §43A (reasonable security practices)"
    depends_on: "needs lawyer review"

  # ============================================================================
  # PASSWORD RESET — SECURITY BEST-PRACTICE + DPDP §8 SECURITY SAFEGUARDS
  # ============================================================================

  - finding_id: LEG-TX-021
    surface_id: api-email-password-reset
    lens: legal
    severity: P1
    issue: "Password reset email content not visible in inventory excerpt — verify token expiry stated, link single-use, no plaintext credentials in body"
    evidence_excerpt: "Subject: \"Reset your password — Fe3dr\" — \"...your account is secure\". 1-hour expiry stated."
    recommendation: "Good — 1-hour expiry is stated. Verify the link is single-use (invalidated after first click), the email never contains the new password in plaintext, and the email includes a 'If this wasn't you, please contact security@...' callout. DPDP §8 imposes 'reasonable security safeguards' on Data Fiduciaries; IT Act §43A reinforces."
    citation: "DPDP Act 2023 §8(5); IT Act 2000 §43A; IT (Reasonable Security Practices) Rules 2011"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-022
    surface_id: api-email-password-reset
    lens: legal
    severity: P2
    issue: "Password reset email must not leak whether the account exists — verify wording does not differ between 'account exists' and 'account does not exist' paths"
    evidence_excerpt: "API success-message-generic includes \"If the email exists, a reset link has been sent\" — implies the email path uses neutral phrasing, but verify"
    recommendation: "Keep the API confirmation neutral (already good). Ensure the actual email is only sent if the account exists, and the API response is identical whether or not it does, to avoid account-enumeration attacks. Best-practice security control."
    citation: "OWASP ASVS V2.2; DPDP Act 2023 §8(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-023
    surface_id: api-email-verify
    lens: legal
    severity: P2
    issue: "Email verification link 24-hour expiry is stated, but verify single-use semantics and that the link does not authenticate the user (only verifies the email)"
    evidence_excerpt: "Subject: \"Verify your email — Fe3dr\" — \"...24-hour expiry stated. Fallback URL copy-paste block included.\""
    recommendation: "Verify the verification link does not double as an authentication bypass. DPDP §8 reasonable-security expectations + IT Rules 2011."
    citation: "DPDP Act 2023 §8(5); IT (Reasonable Security Practices) Rules 2011"
    depends_on: "needs lawyer review"

  # ============================================================================
  # STAFF INVITES — CONSENT + LIABILITY CHAIN
  # ============================================================================

  - finding_id: LEG-TX-024
    surface_id: api-email-staff-invite
    lens: legal
    severity: P1
    issue: "Staff invitation email contains 'role injected as string' — risk of admin-authored role label not matching the actual permission set; ambiguous role description creates contract-ambiguity for the invitee"
    evidence_excerpt: "\"You've been invited! %s has invited you to join the Fe3dr team as %s...\". Role injected as string. 7-day invitation expiry stated."
    recommendation: "Render the role from a controlled enum, not free text. Append a one-line plain-English description of the role's powers (e.g. 'as Operations Manager — can view orders and chat with chefs'). Ambiguous role grants under §10 Indian Contract Act + create downstream liability exposure when staff act under unclear authority."
    citation: "Indian Contract Act 1872 §10 + §227 (authority of agents)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-025
    surface_id: api-inapp-staff-invite
    lens: legal
    severity: P1
    issue: "Staff invitation in-app notification title/message are admin-authored fields with no length or format validation — risk of injection of misleading content, profanity, or PII into a notification surface visible to invitees"
    evidence_excerpt: "Title/Message are admin-authored fields — copy is admin-provided, not platform copy. Risk: no length/format validation on admin input"
    recommendation: "Apply a length cap (≤120 chars title, ≤500 chars body), strip HTML, and run admin-authored notification fields through a profanity/PII filter before fanout. The platform is an intermediary under IT Act §79 — admin-injected content is platform-published from the recipient's view, increasing the platform's exposure if the content is defamatory or violates the IT Rules 2021 Rule 3."
    citation: "IT Act 2000 §79; IT (Intermediary Guidelines) Rules 2021 Rule 3"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-026
    surface_id: ap-staff-toasts
    lens: legal
    severity: P3
    issue: "'Invitation revoked' toast is silent on whether prior actions taken by the invitee remain valid — operational ambiguity"
    evidence_excerpt: "Invitation revoked; Failed to revoke invitation; Invitation resent; ..."
    recommendation: "Cosmetic/legal-adjacent: a follow-up flow should confirm that revoking an unaccepted invitation does NOT affect any actions taken (none, since it was unaccepted). Helpful for audit trail."
    citation: "best-practice"
    depends_on: "needs lawyer review"

  # ============================================================================
  # SUPPORT TICKET EMAILS — UNBRANDED, NO GRIEVANCE FOOTER
  # ============================================================================

  - finding_id: LEG-TX-027
    surface_id: api-email-support-created
    lens: legal
    severity: P1
    issue: "Support ticket created email is inline HTML with no branded wrapper, no grievance officer footer, no platform identification — leaves no audit trail of who is responsible to act on the ticket"
    evidence_excerpt: "INLINE in email.go (not in email_templates.go) — does not use emailBase wrapper, so no branded header/footer/privacy links"
    recommendation: "Move support-ticket emails into the branded wrapper. Required so the support communication carries the grievance-officer contact (CPER 2020 Rule 4(5)) and the platform's legal identification (CPER 2020 Rule 4(1)). Currently a ticket email looks indistinguishable from any other system mail."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rules 4(1) and 4(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-028
    surface_id: api-email-support-update
    lens: legal
    severity: P1
    issue: "Support ticket update email is unbranded, no footer/privacy links — same gap as ticket-created email"
    evidence_excerpt: "Same: inline HTML, unbranded, no footer/privacy links"
    recommendation: "Route through emailBase wrapper. Embed grievance-officer + DPDP rights footer. CPER 2020 Rule 4(5)."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 4(5)"
    depends_on: "needs lawyer review"

  # ============================================================================
  # CHEF VERIFICATION + ADMIN-ACTION NOTIFICATIONS — DUE PROCESS
  # ============================================================================

  - finding_id: LEG-TX-029
    surface_id: api-email-chef-verified
    lens: legal
    severity: P2
    issue: "Chef verification email differs in wording from in-app notification ('kitchen' vs 'chef profile') — contract-evidence ambiguity about what exactly is verified"
    evidence_excerpt: "Email subject: \"Your kitchen is verified\" — in-app: \"Your chef profile has been verified\""
    recommendation: "Pick one. From an FSSAI standpoint, 'kitchen' implies premises verification; 'chef profile' implies identity verification. They are not the same regulatory artefact. The email and in-app must describe identical scope of verification, or differentiate clearly."
    citation: "FSSAI Licensing Regulations 2011; Consumer Protection Act 2019 §2(28)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-030
    surface_id: api-inapp-chef-verified
    lens: legal
    severity: P2
    issue: "Chef-verified in-app notification says 'You can now start accepting orders!' — confirm FSSAI licence is verified before this notification fires"
    evidence_excerpt: "Title: \"Congratulations!\" — Message: \"Your chef profile has been verified. You can now start accepting orders!\""
    recommendation: "The 'accept orders' nudge presupposes the chef has a valid FSSAI registration/licence. Verify the verification flow checks FSSAI document validity and licence number authenticity (not just file-uploaded). If the platform allows orders before FSSAI verification, the platform becomes a knowing intermediary for unlicensed food sale (FSS Act §31 — penalty up to ₹5 lakh)."
    citation: "Food Safety and Standards Act 2006 §31; FSSAI E-Commerce FBO Guidelines 2017/2021"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-031
    surface_id: ap-chefs-toast-rejected
    lens: legal
    severity: P1
    issue: "'Chef rejected' admin toast — no record of reason, no notice to chef about appeal/re-application pathway"
    evidence_excerpt: "Chef rejected — 2 words, no context"
    recommendation: "Rejection of a chef application is an adverse action. Even short of statutory due-process duty, best-practice + Consumer Protection (E-Commerce) Rules 2020 Rule 6(4) (treatment of sellers fairly) require: (a) reason captured, (b) notice sent to the chef with reason + appeal pathway. The current toast only confirms the admin action."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 6(4); best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-032
    surface_id: ap-chefs-toast-suspended
    lens: legal
    severity: P0
    issue: "'Chef suspended' toast confirms admin action but does not surface that the chef was notified, given a reason, or offered appeal — gig-worker / platform-worker fairness exposure"
    evidence_excerpt: "Chef suspended — 2 words"
    recommendation: "Suspension is the most serious adverse action available on the chef. Capture a reason code, notify the chef with: (a) reason, (b) effective date, (c) duration, (d) appeal contact, (e) financial impact (pending payouts). Without these, the suspension is challengeable under Indian Contract Act, IT Rules 2021 Rule 4 (grievance redressal), and emerging platform-worker jurisprudence (Code on Social Security 2020 ss. 2(35) + 113-114)."
    citation: "Indian Contract Act 1872; IT (Intermediary Guidelines) Rules 2021 Rule 3-4; Code on Social Security 2020"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-033
    surface_id: ap-users-toast-suspended
    lens: legal
    severity: P1
    issue: "'User suspended' admin toast — DPDP-relevant action (suspension affects access to personal data); needs notice + grievance pathway to the user"
    evidence_excerpt: "User suspended — 2 words, legal-adjacent (suspension)"
    recommendation: "User suspension restricts access to the user's own personal data; under DPDP §11 (right to access/correction) the Data Principal must be able to challenge the restriction. Send a notice to the user with reason, duration, and appeal contact. Toast alone is insufficient evidence of due process."
    citation: "DPDP Act 2023 §11, §13 (grievance redressal); IT Rules 2021 Rule 3"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-034
    surface_id: ap-users-toast-activated
    lens: legal
    severity: P3
    issue: "'User activated' toast — no record of who activated, when, or why; reversal of suspension should be auditable"
    evidence_excerpt: "User activated — 2 words"
    recommendation: "Audit trail best-practice: capture reactivation reason + admin actor + timestamp. Not directly statutory but DPDP §8(6) requires Data Fiduciaries to 'maintain accuracy and completeness' of records of processing."
    citation: "DPDP Act 2023 §8(6); best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-035
    surface_id: api-inapp-approval-rejected-chef
    lens: legal
    severity: P1
    issue: "Approval rejection in-app notification appends raw admin notes verbatim — admin-authored content surfaces to chef without moderation, including potentially defamatory or PII-revealing notes"
    evidence_excerpt: "Title: \"Request Rejected\" — Message: \"Your %s has been rejected. Notes: %s\" — Raw admin notes verbatim — potential PII / raw admin language leak"
    recommendation: "Validate / sanitise admin notes before surfacing. The platform is liable as the publisher of these notes vis-à-vis the chef. Apply length cap, PII redaction, and an admin-side prompt 'These notes will be shown to the chef — keep professional'. IT Rules 2021 Rule 3 due-diligence obligations bite here."
    citation: "IT (Intermediary Guidelines) Rules 2021 Rule 3; IT Act 2000 §79 (intermediary safe harbour)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-036
    surface_id: api-push-approval-rejected-chef
    lens: legal
    severity: P1
    issue: "Approval rejection push notification carries the same raw admin notes — broadcast outside the app via OS-level notification, potentially visible on lock screen"
    evidence_excerpt: "Body: same as in-app message — Same risk"
    recommendation: "Push notifications surface on the OS lock screen and are stored by FCM/APNs. Raw admin notes containing names, contact info, or sharp wording therefore appear on the device's notification shade. Strip notes from the push body; deliver only 'Tap to view details'."
    citation: "DPDP Act 2023 §4 + §6 (lawful processing); IT Rules 2021 Rule 3"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-037
    surface_id: api-inapp-approval-info-chef
    lens: legal
    severity: P2
    issue: "'More Information Needed' notification surfaces admin-authored notes verbatim — same raw-injection issue as rejection"
    evidence_excerpt: "Message: \"Admin needs more info about your %s. Notes: %s\" — Raw slug + admin notes verbatim"
    recommendation: "Sanitise + length-cap admin notes; replace raw type slug ('menu_item') with a translated label ('menu item')."
    citation: "IT (Intermediary Guidelines) Rules 2021 Rule 3; best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-038
    surface_id: api-inapp-approval-approved-chef
    lens: legal
    severity: P3
    issue: "Approval notifications surface raw enum slugs ('Your menu_item has been approved') — undermines the platform's representation that it communicates clearly with chefs/drivers"
    evidence_excerpt: "%s placeholders: approval type (e.g. \"menu_item\"), title (e.g. \"Add Spicy Aloo\"). Raw type slug may surface to user"
    recommendation: "Translate the slug to user-readable form before fanout. Plain-language failure under CPA 2019 §2(28) misleading-representation lens (low risk, but the platform claims to communicate in plain English)."
    citation: "Consumer Protection Act 2019 §2(28); best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-039
    surface_id: api-inapp-chef-responded
    lens: legal
    severity: P1
    issue: "Chef-authored response message is surfaced to ALL admins with no profanity filter, no length cap, no defamation guardrail"
    evidence_excerpt: "Title: \"Chef Responded: \" + approval.Title — Message: req.Response (chef-authored). Surfaces to all admins. Potential abuse vector — no profanity filter, no length cap visible"
    recommendation: "Apply length cap (≤500 chars), strip HTML, and run through a profanity/PII filter. The platform fans the chef's words to multiple admins; if the words are defamatory, the platform's intermediary safe harbour under IT Act §79 weakens if it fails to 'observe due diligence' (IT Rules 2021 Rule 3)."
    citation: "IT Act 2000 §79; IT (Intermediary Guidelines) Rules 2021 Rule 3"
    depends_on: "needs lawyer review"

  # ============================================================================
  # DELIVERY ASSIGNMENT — INSURANCE + GIG-WORKER NOTICE
  # ============================================================================

  - finding_id: LEG-TX-040
    surface_id: api-email-delivery-assigned
    lens: legal
    severity: P0
    issue: "DeliveryAssigned email template is used for BOTH driver and customer (routing bug per inventory note) — customer would receive '🚗 New Delivery! You've been assigned order #...' which makes no sense and misrepresents who has the delivery duty"
    evidence_excerpt: "function is called by SendDeliveryAssigned and used for driver, but services/notifications.go sends the same template via `delivery_assigned` event to the CUSTOMER — likely a routing bug"
    recommendation: "URGENT. A customer receiving a driver-targeted message is misleading representation under CPA 2019 §2(28). Worse, it creates ambiguity in the chain of custody for the order. Fix routing so customers receive only the customer-targeted 'Your delivery partner has been assigned' template."
    citation: "Consumer Protection Act 2019 §2(28); Consumer Protection (E-Commerce) Rules 2020 Rule 4(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-041
    surface_id: api-push-delivery-assigned-driver
    lens: legal
    severity: P1
    issue: "Driver delivery-assigned push contains no insurance/liability reminder, no acceptance-deadline, no terms reference"
    evidence_excerpt: "Title: \"New Delivery Available\" — Body: \"A delivery near you is ready for pickup\""
    recommendation: "Drivers are gig workers under Code on Social Security 2020. Each assignment is effectively a micro-contract. The notification should reference the platform's driver agreement, the insurance status of the trip (if any), and the acceptance window. Without this, the platform's gig-worker disclosure duties under Section 113 (registration of platform workers) read thinly."
    citation: "Code on Social Security 2020 ss. 2(35), 113-114; Motor Vehicles Act 1988 (insurance disclosures)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-042
    surface_id: api-email-delivery-assigned
    lens: legal
    severity: P1
    issue: "Driver-side delivery-assigned email contains no rate breakdown for the trip — gig-worker earnings transparency gap"
    evidence_excerpt: "\"🚗 New Delivery! You've been assigned order #%s. Pickup location: %s\""
    recommendation: "Show the trip fee, base + distance + surge components, and the platform deduction in the assignment notification. Code on Social Security 2020 + emerging gig-worker fairness expectations (e.g. Rajasthan Platform-Based Gig Workers (Registration and Welfare) Act 2023) push toward earnings transparency at every assignment touchpoint."
    citation: "Code on Social Security 2020 §113-114; Rajasthan Platform-Based Gig Workers Act 2023 (state benchmark)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-043
    surface_id: api-push-delivery-assigned-customer
    lens: legal
    severity: P2
    issue: "Customer 'Delivery Partner Assigned' push does not include the driver's first name, vehicle, or contact — undermines the platform's CPER 2020 Rule 6(5) seller-details disclosure spirit, even though strictly that rule applies to sellers, not delivery partners"
    evidence_excerpt: "Driver name NOT included in push body though available in event.Data. Missed personalization"
    recommendation: "Display the driver's first name and vehicle number in the customer notification. Even short of regulatory requirement, this strengthens the platform's safety story, aids dispute resolution, and is consumer-trust best-practice."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 5(3)(h); best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-044
    surface_id: md-trx-001
    lens: legal
    severity: P0
    issue: "Background-location rationale modal title 'Background Location Needed' — IMPORTANT but the modal copy and the iOS Info.plist string DRIFT (modal is longer per inventory note)"
    evidence_excerpt: "P0 SAFETY: pre-OS-prompt rationale modal title; explains why background location needed; longer than iOS Info.plist string (drift)"
    recommendation: "Sync modal copy with Info.plist NSLocationAlwaysAndWhenInUseUsageDescription and Android's manifest permission rationale. Apple's Human Interface Guidelines + DPDP §5 require the purpose stated at consent to match what is actually done with the data. Mismatched strings can be cited as misrepresentation of processing purpose under DPDP §5."
    citation: "DPDP Act 2023 §5(b) (purpose specification); Apple HIG; Google Play Location Policy"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-045
    surface_id: md-trx-002
    lens: legal
    severity: P0
    issue: "Background-location rationale body does not name the driver as Data Principal, does not state retention period, does not state the data fiduciary"
    evidence_excerpt: "\"To keep customers updated on their delivery, HomeChef Delivery needs to track your location while you are on an active delivery — even when the app is in the background.\""
    recommendation: "Add: (a) name of the data fiduciary, (b) retention period for location data, (c) downstream processors (e.g. Google Maps, GCS), (d) the driver's right to withdraw, with practical effect (delivery acceptance disabled). DPDP §5 + §6 require all of this at the consent moment."
    citation: "DPDP Act 2023 §5, §6"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-046
    surface_id: md-trx-003
    lens: legal
    severity: P1
    issue: "Battery / privacy reassurance ('Battery usage is minimised by only tracking every 15 seconds') — operational claim that must be verifiable; if frequency changes silently, this becomes misrepresentation"
    evidence_excerpt: "\"Location tracking stops automatically when the delivery is completed. Battery usage is minimised by only tracking every 15 seconds.\""
    recommendation: "Bind this string to a config constant and document it in the engineering README. If the polling frequency changes for any reason, the rationale string must be updated in the same commit. Misrepresentation of processing facts is a DPDP §5 + Consumer Protection misleading-representation risk."
    citation: "DPDP Act 2023 §5; Consumer Protection Act 2019 §2(28)"
    depends_on: "needs lawyer review"

  # ============================================================================
  # PAYMENT / RAZORPAY / STRIPE TOASTS — RBI PA + CONSUMER PROTECTION
  # ============================================================================

  - finding_id: LEG-TX-047
    surface_id: web-tx-checkout-payment-toasts
    lens: legal
    severity: P1
    issue: "Razorpay/Stripe success/error/cancel toasts (8 strings) — none state the merchant of record, refund timeline, or grievance contact"
    evidence_excerpt: "\"Failed to initiate payment / Payment gateway is loading / Payment successful! / Payment verification failed...\""
    recommendation: "On payment success, the toast is a critical moment of trust — capture the order/transaction ID, the amount, and one-line refund expectation ('Refunds, if needed, take T+7 days.'). On payment verification failure, link to grievance pathway and state 'Amount, if debited, will be auto-reversed within 5-7 working days' (RBI PA framework). Current bare strings leave the customer uncertain about state and recourse."
    citation: "RBI Payment Aggregator Master Direction 17 Mar 2020; Consumer Protection (E-Commerce) Rules 2020 Rule 5(3)(g)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-048
    surface_id: web-tx-orderdetail-toasts
    lens: legal
    severity: P1
    issue: "'Payment verification failed — please contact support' toast does not provide a direct support route or order context"
    evidence_excerpt: "\"Payment confirmed / Payment verification failed — please contact support / Order cancelled successfully / Failed to cancel order / Order number copied\""
    recommendation: "Show the order ID + a 'Contact support' button (deeplink to support flow), not a bare 'please contact support' instruction. Required to satisfy CPER 2020 Rule 4(5) effective grievance redressal."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 4(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-049
    surface_id: web-tx-orderdetail-toasts
    lens: legal
    severity: P0
    issue: "'Order cancelled successfully' toast is silent on refund timeline and refund channel — RBI PA-relevant"
    evidence_excerpt: "\"Order cancelled successfully\""
    recommendation: "Customer-initiated cancellation MUST surface refund expectation: 'Order cancelled. Refund of ₹X to your original payment method within 7 working days.' Bare success without refund info violates RBI PA framework + CPER 2020."
    citation: "RBI Payment Aggregator Master Direction 17 Mar 2020 §8; Consumer Protection (E-Commerce) Rules 2020 Rule 5(3)(g)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-050
    surface_id: ap-settings-payment-feedback
    lens: legal
    severity: P2
    issue: "Admin payment-gateway toasts surface raw vendor error messages ('Saved, but Razorpay rejected the keys: {err}') — internal-only path, low risk; verify {err} cannot contain secret material"
    evidence_excerpt: "Payment gateway keys saved and verified; Stripe keys saved and verified; Saved, but Razorpay rejected the keys: {err}"
    recommendation: "Audit {err} contents to ensure no API key, signing secret, or PII leaks into the admin toast. Even an admin-only surface that logs to the browser console (and any browser-side error tracker) can persist secrets."
    citation: "DPDP Act 2023 §8(5); IT Act 2000 §43A"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-051
    surface_id: ap-secsettings-toasts
    lens: legal
    severity: P1
    issue: "'All other sessions revoked. You may need to sign in again.' — admin-facing security-action toast; ensure user-facing equivalent fires for the affected users"
    evidence_excerpt: "Password rules updated; Session policy updated; Session revoked; All other sessions revoked..."
    recommendation: "Revoking sessions is a DPDP-relevant security action against the affected user. Confirm the corresponding user is notified out-of-band (email/push) of the session revocation, per DPDP §8(5) reasonable security safeguards + good security practice. Toast alone is insufficient evidence of user notice."
    citation: "DPDP Act 2023 §8(5); IT (Reasonable Security Practices) Rules 2011"
    depends_on: "needs lawyer review"

  # ============================================================================
  # 2FA — STRONG-AUTHN UX
  # ============================================================================

  - finding_id: LEG-TX-052
    surface_id: web-tx-profile-2fa
    lens: legal
    severity: P2
    issue: "2FA toasts ('Backup codes copied to clipboard', 'Key copied') — copying TOTP secret to clipboard surfaces a sensitive credential; verify the clipboard auto-clears and that the user is warned"
    evidence_excerpt: "\"Key copied / Backup codes copied to clipboard\""
    recommendation: "Clipboard write of TOTP seed / backup codes is a high-sensitivity action. Auto-clear after 30 seconds (best-practice) and warn the user ('Codes copied — paste into your authenticator now; clipboard will clear in 30s.'). DPDP §8(5) reasonable security safeguards."
    citation: "DPDP Act 2023 §8(5); OWASP ASVS V2.8"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-053
    surface_id: web-tx-profile-2fa
    lens: legal
    severity: P1
    issue: "'Two-factor authentication disabled' toast — confirm a corresponding email is sent to the registered address as a security-action notification"
    evidence_excerpt: "\"Two-factor authentication enabled / ... / Two-factor authentication disabled / ...\""
    recommendation: "Disabling 2FA is a major security event. Send an out-of-band email to the user ('2FA was disabled on your account at {time}. If this wasn't you, secure your account now.') Best-practice + DPDP §8(5) reasonable security."
    citation: "DPDP Act 2023 §8(5); OWASP ASVS V2.6"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-054
    surface_id: ap-secsettings-toasts
    lens: legal
    severity: P1
    issue: "Admin 2FA toasts include '2FA disabled' — confirm admin-side 2FA-off action triggers a corresponding security email to the affected admin"
    evidence_excerpt: "2FA enabled successfully; 2FA disabled; Enforcement setting updated; Invalid code; ..."
    recommendation: "Admin accounts have higher privilege. 2FA-disable on an admin account is a privileged security event and must trigger out-of-band notification + audit log entry. DPDP §8(5)."
    citation: "DPDP Act 2023 §8(5)"
    depends_on: "needs lawyer review"

  # ============================================================================
  # MOBILE — DPDP NOTICE + DATA-DELETION PATHWAY
  # ============================================================================

  - finding_id: LEG-TX-055
    surface_id: md-trx-010
    lens: legal
    severity: P0
    issue: "Account-deletion path is 'Contact support at support@homechef.in' — DPDP §6(4) requires consent withdrawal / right to erasure (§12) to be as easy as giving consent"
    evidence_excerpt: "\"Contact support at support@homechef.in to request account deletion.\" Support email hardcoded — India domain"
    recommendation: "Replace the contact-support pattern with an in-app 'Delete my account' flow. DPDP §12 (right to erasure) + §6(4) require the withdrawal/erasure path to be at least as easy as the signup path. Email-only is not 'as easy as' a one-tap signup. Also Apple App Store Guideline 5.1.1(v) requires in-app account deletion for apps that allow account creation."
    citation: "DPDP Act 2023 §6(4), §12; Apple App Store Review Guidelines §5.1.1(v); Google Play Data Deletion Policy 2024"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-056
    surface_id: md-trx-009
    lens: legal
    severity: P2
    issue: "Subscription management deferred to web ('Visit the web portal to manage your subscription.') — confirm this is acceptable under Apple/Google subscription rules and Indian Auto-Debit framework"
    evidence_excerpt: "\"Visit the web portal to manage your subscription.\""
    recommendation: "If subscription was sold inside the iOS app, it MUST be cancellable inside iOS (Apple Guidelines 3.1.2). If sold via web checkout, web-cancel is OK. Either way, RBI e-Mandate framework requires cancellation as easy as enrolment. Confirm the subscription channel and cancellation path align."
    citation: "RBI e-Mandate framework (2019, updated 2021); Apple App Store Review Guidelines §3.1.2; Google Play Subscriptions Policy"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-057
    surface_id: mv-onb-pending-rejected-reason-prefix
    lens: legal
    severity: P1
    issue: "Driver/chef rejection screens show 'Reason: ...' with admin-authored content — same raw-injection issue as approval-rejected notifications"
    evidence_excerpt: "\"Reason: ...\""
    recommendation: "Sanitise admin-authored rejection reason. Display the reason in a callout box with structure ('What needs to change' / 'How to resubmit') rather than free-text dump. IT Rules 2021 Rule 3 + best-practice."
    citation: "IT (Intermediary Guidelines) Rules 2021 Rule 3"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-058
    surface_id: mv-onb-pending-rejected-body
    lens: legal
    severity: P2
    issue: "'Please review the feedback and resubmit your application' — does not surface appeal contact if the chef disagrees with the rejection"
    evidence_excerpt: "\"Please review the feedback and resubmit your application.\""
    recommendation: "Add a 'Disagree with this decision? Contact appeals@...' line. Captures CPER 2020 Rule 6(4) fair-treatment-of-sellers expectation and reduces social-media-as-grievance-channel risk."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 6(4)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-059
    surface_id: vp-onb-submit-success
    lens: legal
    severity: P3
    issue: "'Application submitted! We'll review and get back to you within 24-48 hours' — sets a 24-48h SLA in a transactional toast; if not met, chef may invoke as broken promise"
    evidence_excerpt: "\"Application submitted! We'll review and get back to you within 24-48 hours.\""
    recommendation: "Either honour the SLA operationally or soften wording ('We aim to respond within 24-48 hours'). Transactional copy is evidence of platform commitments — under Consumer Protection Act 2019 §2(28), a non-honoured SLA stated in transactional channel can be argued as misleading representation."
    citation: "Consumer Protection Act 2019 §2(28); Indian Contract Act 1872 §29"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-060
    surface_id: vp-onb-policies-ready-banner
    lens: legal
    severity: P2
    issue: "'You're all set to submit your application!' banner does not summarise what the chef is consenting to — DPDP §5 notice gap at the moment of submission"
    evidence_excerpt: "\"You're all set to submit your application! / Our team will review your details and you'll be notified within 24-48 hours.\""
    recommendation: "Before submission, show a final-consent summary: what personal documents were uploaded, who can access them, retention period, withdrawal pathway. DPDP §5 requires Notice at or before consent."
    citation: "DPDP Act 2023 §5"
    depends_on: "needs lawyer review"

  # ============================================================================
  # CROSS-CHANNEL CONSISTENCY — CONTRACT-EVIDENCE INTEGRITY
  # ============================================================================

  - finding_id: LEG-TX-061
    surface_id: api-push-chef-new-order-actionable
    lens: legal
    severity: P2
    issue: "Chef receives TWO pushes with slightly different wording for the same 'new order' event (different queue groups) — duplicate transactional fanout is annoying but legally also undermines the platform's record of the notification"
    evidence_excerpt: "DIFFERENT from services/notifications.go::handleOrderCreated push — Both fire on the same NATS event (different queue groups), so the chef gets TWO pushes with slightly different copy"
    recommendation: "De-duplicate the fanout. Two pushes for one event creates ambiguity about 'when did we notify the chef' if a delivery SLA dispute arises later. Pick one path."
    citation: "Indian Contract Act 1872 §29 (certainty); evidentiary best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-062
    surface_id: api-inapp-order-status-customer
    lens: legal
    severity: P2
    issue: "In-app order-status title 'Order Status Updated' vs push title 'Order Update' — minor inconsistency, captured for completeness"
    evidence_excerpt: "Title differs from push (\"Updated\" vs \"Update\"). Map of 7 statuses"
    recommendation: "Unify title strings. Minor on its own but compounding across channels weakens the platform's 'we communicate consistently' representation."
    citation: "best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-063
    surface_id: api-inapp-delivery-pickedup-customer
    lens: legal
    severity: P2
    issue: "Customer in-app + push titles for the same event differ ('Order Picked Up' vs 'Order On The Way!')"
    evidence_excerpt: "Title differs from push title (which says \"Order On The Way!\")"
    recommendation: "Pick one event title. Two-channel drift makes timeline reconstruction harder in a dispute (when did the customer know the order was picked up?)."
    citation: "Indian Contract Act 1872 §29; best-practice"
    depends_on: "needs lawyer review"

  # ============================================================================
  # GENERIC SUCCESS / FAILURE TOASTS — DPDP + GRIEVANCE PATHWAY
  # ============================================================================

  - finding_id: LEG-TX-064
    surface_id: api-success-message-generic
    lens: legal
    severity: P1
    issue: "Generic API success strings — 'Chef rejected', 'Chef suspended', 'User suspended', 'Delivery partner suspended', 'Staff member deactivated', 'Key revoked' — all serious access-removal actions surfaced as 2-3 word confirmations with no audit trail visible in code"
    evidence_excerpt: "43 unique success message strings. Inconsistent 'successfully' suffix"
    recommendation: "For each adverse-action message (suspend, reject, revoke, deactivate, delete), confirm a server-side audit log captures: actor, target, reason code, timestamp, IP. Bare success strings without audit trail expose the platform if a suspended chef/driver/user litigates. DPDP §8(6) requires accuracy & completeness of records; IT Rules 2021 require audit trails for grievance handling."
    citation: "DPDP Act 2023 §8(6); IT (Intermediary Guidelines) Rules 2021 Rule 3-4"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-065
    surface_id: api-success-message-generic
    lens: legal
    severity: P2
    issue: "'Removed from favorites' / 'Address deleted' / 'Image deleted' — DPDP §12 (right to erasure) operations confirmed in 2-3 words; verify hard-delete vs soft-delete semantics align with the platform's stated retention policy"
    evidence_excerpt: "\"Address deleted\" / \"Image deleted\" / \"Removed from favorites\""
    recommendation: "Confirm what 'deleted' actually means downstream — hard delete from DB? Soft delete with retention window? Backup retention? DPDP §12 requires erasure 'unless retention is necessary for the specified purpose or for compliance with any law'. If 'deleted' is soft-delete, the Privacy Notice must say so."
    citation: "DPDP Act 2023 §12; §8(8) (erasure on withdrawal)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-066
    surface_id: web-tx-profile-update
    lens: legal
    severity: P2
    issue: "Profile photo upload error 'Invalid file type. Use JPEG, PNG or WebP.' — confirm uploaded photos are scanned for malware and that EXIF metadata is stripped before storage to avoid PII leakage (GPS coords)"
    evidence_excerpt: "\"File too large. Maximum 5 MB. / Invalid file type. Use JPEG, PNG or WebP. / Profile photo updated\""
    recommendation: "DPDP §8(5) reasonable security safeguards + best-practice. Photos uploaded from mobile commonly contain GPS EXIF metadata; storing this without notice or stripping is excessive data processing under DPDP §4 lawful-purpose limitation."
    citation: "DPDP Act 2023 §4, §8(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-067
    surface_id: vp-ux-profile-doc-toasts
    lens: legal
    severity: P1
    issue: "Chef document upload toast '{label} uploaded' is generic — chef KYC documents (Aadhaar, PAN, FSSAI licence) carry sensitive-personal-data handling obligations and the toast does not surface what was uploaded or to where"
    evidence_excerpt: "{label} uploaded / Upload failed"
    recommendation: "Documents uploaded as part of chef onboarding likely include Aadhaar, PAN, FSSAI licence — these are sensitive-class identifiers. The upload toast should reference the privacy notice and confirm the storage location/encryption status. DPDP §5 + §8(5) require purpose-specific consent + reasonable security for sensitive data. Aadhaar Act §29 + UIDAI regulations impose additional duties if the document is an Aadhaar."
    citation: "DPDP Act 2023 §5, §8(5); Aadhaar Act 2016 §29; UIDAI Aadhaar (Authentication & Offline Verification) Regulations"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-068
    surface_id: dp-tx-toast-step3-uploaded
    lens: legal
    severity: P1
    issue: "Driver document upload toast '{label} uploaded' — same KYC sensitivity as chef; drivers upload DL, RC, Aadhaar/PAN; toast does not state where the data goes or retention"
    evidence_excerpt: "\"{label} uploaded\""
    recommendation: "Same as LEG-TX-067. Driver KYC includes Driving Licence (Motor Vehicles Act validation) + Aadhaar/PAN. Surface a privacy-notice link on the upload screen. Aadhaar Act §29 + DPDP §5 + §8(5)."
    citation: "DPDP Act 2023 §5, §8(5); Aadhaar Act 2016 §29; Motor Vehicles Act 1988"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-069
    surface_id: vp-ux-settings-payout-toasts
    lens: legal
    severity: P1
    issue: "Chef payout-details toast 'Payout details saved' — bank account / UPI / IFSC entry is financial-sensitive data with RBI KYC implications"
    evidence_excerpt: "Payout details saved / Failed to save payout details / Settings saved / Failed to save settings"
    recommendation: "Capturing bank/UPI details makes the chef a beneficiary in a payout flow regulated under RBI PA framework. Confirm KYC of the chef is complete before saving payout details (RBI PA Master Direction §9.1). Surface a one-line privacy notice on the payout screen referencing how the bank data is stored and shared with the payment aggregator."
    citation: "RBI Payment Aggregator Master Direction 17 Mar 2020 §9; DPDP Act 2023 §5, §8(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-070
    surface_id: dp-tx-toast-payout-saved
    lens: legal
    severity: P1
    issue: "Driver payout-details toast 'Payout details saved' — same RBI PA + DPDP concerns as chef payout"
    evidence_excerpt: "\"Payout details saved\""
    recommendation: "Same as LEG-TX-069 for drivers. RBI PA framework KYC."
    citation: "RBI Payment Aggregator Master Direction 17 Mar 2020 §9; DPDP Act 2023 §5"
    depends_on: "needs lawyer review"

  # ============================================================================
  # CATERING, FAVORITES, CART — MISC TRANSACTIONAL EXPOSURE
  # ============================================================================

  - finding_id: LEG-TX-071
    surface_id: web-tx-catering-quotes
    lens: legal
    severity: P1
    issue: "'Quote accepted!' toast does not surface the legal effect of acceptance — accepting a catering quote likely forms a binding contract under Indian Contract Act §7"
    evidence_excerpt: "\"Quote accepted! The chef has been notified. / Failed to accept quote. Please try again. / Quote declined.\""
    recommendation: "On quote acceptance, surface a modal/confirmation step stating 'By accepting, you agree to the catering terms (price ₹X, date Y, refund policy Z).' Toast-only confirmation is thin evidence of contract formation. Indian Contract Act §7 (acceptance must be absolute & unqualified, communicated)."
    citation: "Indian Contract Act 1872 §§4, 7, 10"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-072
    surface_id: web-tx-catering-request-submit
    lens: legal
    severity: P2
    issue: "'Catering request submitted successfully!' — no consent capture for the chef to receive/process the customer's catering brief (which can include guest count, dietary needs, address)"
    evidence_excerpt: "\"Catering request submitted successfully!\""
    recommendation: "DPDP §5 + §6: the customer is consenting at submission that the brief is shared with chefs. Surface a one-line notice. Also if the brief includes special dietary / health information (allergies), that is sensitive personal data warranting purpose-specific consent."
    citation: "DPDP Act 2023 §5, §6"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-073
    surface_id: web-tx-chefdetail-add-to-cart
    lens: legal
    severity: P3
    issue: "'Your cart has items from another chef. Clear cart first.' — operational message, no legal exposure on its own; flagged for completeness because the prior-cart state may include items the customer would prefer to keep"
    evidence_excerpt: "\"Added {item.name} to cart / Your cart has items from another chef. Clear cart first.\""
    recommendation: "Cosmetic legal-adjacent: confirm 'clearing' does not delete a saved-for-later record that would have ongoing personal-data implications under DPDP §12 erasure. Low priority."
    citation: "DPDP Act 2023 §12; best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-074
    surface_id: mc-menuitem-cross-chef
    lens: legal
    severity: P3
    issue: "Mobile cross-chef cart replacement 'Replace Cart? / You have items from another chef. Replace cart?' — same as web; informational only"
    evidence_excerpt: "\"Replace Cart? / You have items from another chef. Replace cart? / Cancel / Replace\""
    recommendation: "Same as LEG-TX-073."
    citation: "best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-075
    surface_id: web-tx-favorites-remove
    lens: legal
    severity: P3
    issue: "'Removed {name} from favorites' — DPDP §12 erasure on user-initiated removal; ensure removal is effective not soft-deleted"
    evidence_excerpt: "\"Removed {name} from favorites\""
    recommendation: "Verify the favorite is removed from underlying storage, not just hidden client-side. DPDP §12."
    citation: "DPDP Act 2023 §12"
    depends_on: "needs lawyer review"

  # ============================================================================
  # DRIVER ONLINE/OFFLINE + ACTIVE DELIVERY — GIG-WORKER STATUS RECORDS
  # ============================================================================

  - finding_id: LEG-TX-076
    surface_id: dp-tx-toast-online
    lens: legal
    severity: P2
    issue: "'You are now online' / 'You are now offline' — driver-status change is a Code-on-Social-Security 'work spell' marker; no audit trail visible from copy alone"
    evidence_excerpt: "\"You are now online / You are now offline\""
    recommendation: "Confirm a server-side log captures every online/offline transition with timestamp + location (if consented). Gig-worker hours-of-work data may become statutorily relevant under Code on Social Security 2020 ss. 113-114 implementation."
    citation: "Code on Social Security 2020 §§113-114"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-077
    surface_id: dp-tx-toast-accept-success
    lens: legal
    severity: P2
    issue: "'Delivery accepted!' — driver accepts a trip-level micro-contract; toast is thin evidence of that contract; reference to terms missing"
    evidence_excerpt: "\"Delivery accepted!\""
    recommendation: "Capture, server-side, the exact trip parameters the driver accepted (fee, distance, ETA, pickup, drop). Surface trip details on acceptance screen, not just a celebratory toast. Indian Contract Act §7 + Code on Social Security gig-worker earnings transparency."
    citation: "Indian Contract Act 1872 §7; Code on Social Security 2020"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-078
    surface_id: web-tx-delivery-actions
    lens: legal
    severity: P2
    issue: "Web driver dashboard toasts 'Delivery completed! Great job!' — celebratory tone but no trip-summary or earnings confirmation"
    evidence_excerpt: "\"Delivery accepted! / Delivery completed! Great job! / Status updated / You are now offline / You are now online\""
    recommendation: "On delivery completion, surface earnings for the trip (base + tip + bonus) and the running daily total. Code on Social Security 2020 + gig-worker earnings transparency expectations."
    citation: "Code on Social Security 2020 §§113-114"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-079
    surface_id: dp-tx-toast-partner-suspend
    lens: legal
    severity: P0
    issue: "'Partner suspended' admin toast — gig-worker adverse-action; same severe due-process gap as chef suspension"
    evidence_excerpt: "\"Partner reactivated / Partner suspended\""
    recommendation: "Same as LEG-TX-032 for chefs. Driver suspension impacts livelihood; must include reason code, written notice, appeal pathway, treatment of pending payouts. Code on Social Security 2020 + emerging platform-worker fairness norms."
    citation: "Code on Social Security 2020 §§113-114; Indian Contract Act 1872; IT (Intermediary Guidelines) Rules 2021 Rule 3"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-080
    surface_id: dp-tx-toast-partner-verified
    lens: legal
    severity: P1
    issue: "'Partner verified successfully' admin toast — confirm verification includes background check, DL validity, motor insurance status; toast says nothing about scope"
    evidence_excerpt: "\"Partner verified successfully\""
    recommendation: "'Verified' must map to a defined set of checks. Document those checks server-side; if a verified driver is later involved in an incident, the platform's defence depends on showing what the verification consisted of. Motor Vehicles Act 1988 driver-licensing + insurance disclosure."
    citation: "Motor Vehicles Act 1988; Code on Social Security 2020 §113"
    depends_on: "needs lawyer review"

  # ============================================================================
  # ADMIN ACTIONS — INVITES, ROLES, ZONES
  # ============================================================================

  - finding_id: LEG-TX-081
    surface_id: ap-staffdetail-toasts
    lens: legal
    severity: P1
    issue: "'Staff member deactivated' / 'Staff member reactivated' — access-removal actions; no surfaced audit trail or notice to the affected staff"
    evidence_excerpt: "Role updated successfully; Failed to update role; Staff member deactivated; Failed to deactivate staff member; Staff member reactivated"
    recommendation: "Deactivation removes access to platform data — DPDP §11 right-of-access of the affected staff must continue to function. Confirm out-of-band notification to deactivated staff with reason + appeal pathway."
    citation: "DPDP Act 2023 §11, §13; IT (Intermediary Guidelines) Rules 2021 Rule 3"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-082
    surface_id: dp-tx-toast-invite-copied
    lens: legal
    severity: P2
    issue: "'Invite URL copied to clipboard' — invite URL is a credential-equivalent token; copying to clipboard is a sensitive moment that should warn the admin not to share over insecure channels"
    evidence_excerpt: "\"Invite URL copied to clipboard\""
    recommendation: "Surface a one-line warning ('Send this link only over a secure channel') or use a more secure invite delivery (email-only)."
    citation: "DPDP Act 2023 §8(5); best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-083
    surface_id: ap-providercreate-toasts
    lens: legal
    severity: P3
    issue: "'Provider created successfully' admin toast — admin-only path, low exposure; flagged for completeness"
    evidence_excerpt: "Provider created successfully; Failed to create provider"
    recommendation: "No legal exposure on its own. Confirm audit log captures the action."
    citation: "best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-084
    surface_id: ap-platsettings-feedback
    lens: legal
    severity: P2
    issue: "'Commission rates updated' / 'Delivery fees updated' — pricing-policy changes; do not surface customer/chef notice from this admin path"
    evidence_excerpt: "Commission rates updated; Delivery fees updated; Operating hours updated; Zone created; Zone deleted"
    recommendation: "Pricing/commission changes impact chefs (revenue share) and customers (final price). Confirm there is a separate flow that notifies affected chefs of commission changes with reasonable advance notice. Indian Contract Act §62 (novation) implications + CPER 2020 Rule 6(4) seller-fairness."
    citation: "Indian Contract Act 1872 §62; Consumer Protection (E-Commerce) Rules 2020 Rule 6(4)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-085
    surface_id: ap-notifsettings-toasts
    lens: legal
    severity: P3
    issue: "Admin notification settings 'Preference saved' — operational, low risk; verify any setting that disables transactional alerts to customers is gated to avoid breaking statutory notice duties"
    evidence_excerpt: "Preference saved; Failed to save"
    recommendation: "Confirm an admin cannot, via this surface, disable a transactional alert that is statutorily required (e.g. cancellation notice, refund initiation). Mark such alerts as non-configurable."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 4(5); best-practice"
    depends_on: "needs lawyer review"

  # ============================================================================
  # CHEF MENU / SOCIAL — FSSAI ALLERGEN + IT INTERMEDIARY DUTY
  # ============================================================================

  - finding_id: LEG-TX-086
    surface_id: web-tx-chef-menu
    lens: legal
    severity: P1
    issue: "Menu CRUD toasts 'Menu item created successfully' — toast says nothing about FSSAI allergen disclosure being mandatory on the menu item; chef can publish without filling allergens"
    evidence_excerpt: "\"Menu item created successfully / Menu item updated successfully / Menu item deleted / Failed to create menu item / Failed to update menu item\""
    recommendation: "FSSAI Labelling Regulation 2.2.2.4 requires the 8 statutory allergens (cereals containing gluten, crustaceans, eggs, fish, peanuts, soybean, milk, tree nuts) to be declared. The menu-item form must require allergen selection before save; the success toast confirms the chef has met that disclosure."
    citation: "FSS (Labelling and Display) Regulations 2020 Reg. 5(1)(d); FSSAI E-Commerce FBO Guidelines 2021"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-087
    surface_id: vp-ux-menu-form-toasts
    lens: legal
    severity: P1
    issue: "Vendor portal menu form toasts — same allergen-disclosure gap as web; 'At least one image is required' but no analogous 'At least one allergen confirmation is required'"
    evidence_excerpt: "Menu item created successfully / Failed to create menu item / Menu item updated successfully / ... / At least one image is required"
    recommendation: "Add 'Confirm allergens' as a required step before menu publish. FSSAI E-Commerce FBO Guidelines + Labelling Regulations."
    citation: "FSS (Labelling and Display) Regulations 2020 Reg. 5(1)(d)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-088
    surface_id: web-tx-chef-social
    lens: legal
    severity: P1
    issue: "Chef social post toasts 'Post created / Post updated / Post deleted' — chef social posts are user-generated content; the platform's IT Act §79 intermediary safe harbour depends on due-diligence; no moderation pathway visible from copy"
    evidence_excerpt: "\"Post deleted / Post created / Post updated / Failed to save post\""
    recommendation: "Surface a 'Posting Guidelines' link near the create-post action. Apply server-side moderation for prohibited content (CSAM, defamation, FSSAI-violating health claims). IT Rules 2021 Rule 3 prescribes specific categories that must be removed within 24-36h of complaint."
    citation: "IT Act 2000 §79; IT (Intermediary Guidelines) Rules 2021 Rule 3"
    depends_on: "needs lawyer review"

  # ============================================================================
  # MOBILE — DPDP & APP-STORE COMPLIANCE
  # ============================================================================

  - finding_id: LEG-TX-089
    surface_id: mc-profile-logout-confirm
    lens: legal
    severity: P3
    issue: "Mobile customer logout confirmation 'Log out / Are you sure you want to log out?' uses banned variant — Style guide says 'Sign out' not 'Log out'"
    evidence_excerpt: "\"Log out / Are you sure you want to log out? / Cancel / Log Out\" — title-case inconsistency"
    recommendation: "Outside Legal lens scope (voice rule). Flag passes through to TW lens. Legal-relevant only: case inconsistency 'Log out' vs button 'Log Out' is a minor consistency issue."
    citation: "STYLE-GUIDE.md §3 (Identity & roles)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-090
    surface_id: mc-profile-save-success
    lens: legal
    severity: P3
    issue: "'Profile updated successfully. / Cuisine preferences updated.' — DPDP §11 right-to-correction surface; verify updates propagate to all systems"
    evidence_excerpt: "\"Saved / Profile updated successfully. / Cuisine preferences updated.\""
    recommendation: "Confirm profile edits propagate to all downstream systems (analytics, marketing lists, payment provider). DPDP §11 right-to-correction requires changes to be reflected wherever the data is held."
    citation: "DPDP Act 2023 §11"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-091
    surface_id: mv-onb-pending-submitted-body
    lens: legal
    severity: P3
    issue: "Mobile vendor pending screen 'Our team will review your application within 24-48 hours' — same SLA commitment as LEG-TX-059"
    evidence_excerpt: "\"Our team will review your application within 24-48 hours. We will notify you once the review is complete.\""
    recommendation: "Same as LEG-TX-059 — either honour the SLA or soften wording."
    citation: "Consumer Protection Act 2019 §2(28)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-092
    surface_id: mv-undo-rejected
    lens: legal
    severity: P1
    issue: "Mobile vendor 'Order rejected' undo snackbar — order rejection by chef triggers customer refund obligation; undo window must be tight + customer must not be misled about state"
    evidence_excerpt: "\"Order rejected\""
    recommendation: "Confirm the undo window does not exceed a few seconds and that no customer-facing notification fires during the undo window. If undo expires, customer must be promptly notified per CPER 2020 + RBI PA refund timeline."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 4(5); RBI PA Master Direction"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-093
    surface_id: mv-reviewdetail-success-body
    lens: legal
    severity: P2
    issue: "'Your reply has been posted.' — chef reply to a customer review is platform-published content; IT Act intermediary duty applies"
    evidence_excerpt: "\"Your reply has been posted.\""
    recommendation: "Apply moderation guardrails on chef replies (no PII about the customer, no defamatory content). IT Rules 2021 Rule 3."
    citation: "IT (Intermediary Guidelines) Rules 2021 Rule 3"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-094
    surface_id: vp-ux-kitchen-photo-toasts
    lens: legal
    severity: P2
    issue: "Kitchen photo upload toasts — kitchen photographs may incidentally show people (cooks, family) who are not the chef; consent for inclusion of third-party identifiable persons not captured at upload"
    evidence_excerpt: "Kitchen photo uploaded / Failed to upload photo / Photo removed / ... / Kitchen setup saved successfully"
    recommendation: "DPDP §6 + §10 (children) require consent for processing personal data of identifiable individuals. Surface a one-line notice 'Only upload photos of your kitchen and personnel who have consented to being shown.' Strip EXIF GPS metadata server-side."
    citation: "DPDP Act 2023 §6, §10"
    depends_on: "needs lawyer review"

  # ============================================================================
  # SOCIAL FEED + UNAUTH INTERACTIONS
  # ============================================================================

  - finding_id: LEG-TX-095
    surface_id: web-tx-social-feed-actions
    lens: legal
    severity: P3
    issue: "'Please log in to like posts / Please log in to save posts' — uses banned variant 'log in' (style guide says 'Sign in')"
    evidence_excerpt: "\"Please log in to like posts / Please log in to save posts\""
    recommendation: "Outside Legal lens — flagged for completeness. Use 'Sign in'."
    citation: "STYLE-GUIDE.md §3"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-096
    surface_id: vp-ux-notifs-respond-toasts
    lens: legal
    severity: P2
    issue: "Chef response 'Response sent to admin' — chef-authored response visible to admins; no length/format guard surfaced from copy"
    evidence_excerpt: "Please enter a response / Response sent to admin / Failed to send response"
    recommendation: "Apply length cap + PII filter on chef responses. Same as LEG-TX-039 chef-responded notification pathway."
    citation: "IT (Intermediary Guidelines) Rules 2021 Rule 3"
    depends_on: "needs lawyer review"

  # ============================================================================
  # GENERIC PROFILE / PREFERENCES UPDATES — DPDP §11 PROPAGATION
  # ============================================================================

  - finding_id: LEG-TX-097
    surface_id: web-tx-profile-update
    lens: legal
    severity: P2
    issue: "Profile update toasts include avatar/file-size errors — confirm rejected files are not retained/logged anywhere with PII (e.g. server-side rejection logging may capture filename or metadata)"
    evidence_excerpt: "\"File too large. Maximum 5 MB. / Invalid file type. Use JPEG, PNG or WebP. / Profile photo updated\""
    recommendation: "DPDP §4 lawful-purpose limitation: rejected files should not be retained beyond the request, and any error log should not capture file metadata containing PII. Verify server-side log scrubbing."
    citation: "DPDP Act 2023 §4, §8(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-098
    surface_id: web-tx-profile-preferences-save
    lens: legal
    severity: P2
    issue: "'Preferences updated' — preferences may include marketing/comms opt-in; confirm DPDP §6(4) withdrawal-as-easy-as-consent semantics"
    evidence_excerpt: "\"Preferences updated / Failed to save preferences\""
    recommendation: "If preferences include marketing comms opt-out, confirm the toast confirms the new state ('Marketing emails: off') rather than a generic 'updated'. DPDP §6(4)."
    citation: "DPDP Act 2023 §6(4)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-099
    surface_id: web-tx-profile-addresses
    lens: legal
    severity: P2
    issue: "Address CRUD toasts — addresses are personal data + may include delivery instructions revealing health/lifestyle (e.g. 'ring extra long, I have hearing impairment'); confirm purpose-limited processing"
    evidence_excerpt: "Please fill in all required fields / Address added / Address updated / Address deleted / Failed to save address / Failed to delete address"
    recommendation: "DPDP §4 + §11: address data must only be used for the stated purpose (delivery). If used for marketing geo-segmentation, separate consent required. Confirm 'Address deleted' is hard-delete from all systems."
    citation: "DPDP Act 2023 §4, §11, §12"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-100
    surface_id: web-tx-onboarding-save
    lens: legal
    severity: P2
    issue: "Customer onboarding 'Your preferences have been saved!' — preferences captured at onboarding may include dietary restrictions (sensitive health-related data)"
    evidence_excerpt: "\"Please fill in the required fields / Your preferences have been saved! / Failed to save. Please try again.\""
    recommendation: "Dietary preferences (allergies, religious diet, medical diet) are sensitive personal data under DPDP §10 implications. Capture purpose-specific consent ('We use these to filter chefs and dishes for you — not for marketing.') and document retention. DPDP §5, §6."
    citation: "DPDP Act 2023 §5, §6"
    depends_on: "needs lawyer review"

  # ============================================================================
  # GENERIC ALERT TITLES / WORDING DRIFT (MOBILE)
  # ============================================================================

  - finding_id: LEG-TX-101
    surface_id: md-trx-011
    lens: legal
    severity: P3
    issue: "Driver app validation alert title 'Validation' — non-descriptive title; offers no user-actionable signal"
    evidence_excerpt: "\"Validation\""
    recommendation: "Replace with descriptive title ('Missing details'). Plain-language failure under STYLE-GUIDE; legal-relevance is minor."
    citation: "STYLE-GUIDE.md §4 (Errors); best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-102
    surface_id: md-trx-013
    lens: legal
    severity: P3
    issue: "Driver app success alert title 'Success' — non-descriptive"
    evidence_excerpt: "\"Success\""
    recommendation: "Use action-specific title ('Profile saved'). Cosmetic; legal-relevance minor."
    citation: "STYLE-GUIDE.md §4"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-103
    surface_id: md-trx-006
    lens: legal
    severity: P3
    issue: "Driver app 'Logout' confirmation uses banned variant — style guide requires 'Sign out'"
    evidence_excerpt: "\"Logout\""
    recommendation: "Outside Legal lens — captured for completeness. Use 'Sign out'."
    citation: "STYLE-GUIDE.md §3"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-104
    surface_id: md-trx-015
    lens: legal
    severity: P3
    issue: "Generic failure body 'Failed to update profile. Please try again.' — leaks no info but provides no diagnostic, no support contact"
    evidence_excerpt: "\"Failed to update profile. Please try again.\""
    recommendation: "Add a 'Contact support' fallback if retry fails — CPER 2020 Rule 4(5) grievance access. Low priority."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 4(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-105
    surface_id: md-trx-018
    lens: legal
    severity: P2
    issue: "'You do not have permission to invite staff.' — 403 error; ensure rejected action is logged for audit (especially if attempted by a previously-privileged user)"
    evidence_excerpt: "\"You do not have permission to invite staff.\""
    recommendation: "Confirm 403 events are server-side audit-logged. DPDP §8(6) records-of-processing accuracy + IT Rules 2021 audit-trail."
    citation: "DPDP Act 2023 §8(6); IT (Intermediary Guidelines) Rules 2021 Rule 3"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-106
    surface_id: api-success-message-generic
    lens: legal
    severity: P2
    issue: "API success string 'If the email exists, a reset link has been sent' — already neutral (good) — but verify rate-limiting on reset endpoint to prevent enumeration via timing"
    evidence_excerpt: "\"If the email exists, a reset link has been sent\""
    recommendation: "Wording is correct. Add rate-limiting (per-IP + per-email) to defeat timing-based enumeration. DPDP §8(5)."
    citation: "DPDP Act 2023 §8(5); OWASP ASVS V2.2"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-107
    surface_id: api-success-message-generic
    lens: legal
    severity: P2
    issue: "API success string 'Email verified successfully' — confirm the verification action carries audit-trail entry tying email to user/session/IP"
    evidence_excerpt: "\"Email verified successfully\""
    recommendation: "DPDP §8(6). Audit-log every identity-event (email verified, phone verified, 2FA enrol/disenrol)."
    citation: "DPDP Act 2023 §8(6); IT (Reasonable Security Practices) Rules 2011"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-108
    surface_id: api-success-message-generic
    lens: legal
    severity: P2
    issue: "API success string 'Logged out successfully' — uses banned 'logged out' (style says 'signed out'); legal-relevance: ensure server-side session is invalidated, not just client-side cookie cleared"
    evidence_excerpt: "\"Logged out successfully\""
    recommendation: "Verify server-side session invalidation (DPDP §8(5) reasonable security). Update copy to 'Signed out'."
    citation: "DPDP Act 2023 §8(5); STYLE-GUIDE.md §3"
    depends_on: "needs lawyer review"

  # ============================================================================
  # IN-APP NOTIFICATIONS / DRIVER ONBOARDING ADMIN FANOUT
  # ============================================================================

  - finding_id: LEG-TX-109
    surface_id: api-inapp-driver-onboarding-admins
    lens: legal
    severity: P1
    issue: "Driver onboarding submitted notification fanned to ALL admins with city of driver — operational visibility for staff; verify all those admins have role/legitimate-interest to see driver location"
    evidence_excerpt: "Title: \"New Driver Application\" — Message: \"A new driver from %s has submitted their onboarding application for review.\" Fanned out to all admins. City substituted"
    recommendation: "DPDP §4 lawful-purpose limitation: not every admin role needs to know about every driver application. Scope the fanout to onboarding-team admins only. Cross-cutting access reduces principle of data-minimisation."
    citation: "DPDP Act 2023 §4, §8"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-110
    surface_id: api-inapp-approval-created-admin
    lens: legal
    severity: P2
    issue: "Approval created notification fanned to all admins with raw 'Approval title injected verbatim' — chef-authored or system-authored content broadcast to all admins"
    evidence_excerpt: "Title: \"New Approval Request\" — Message: \"New approval request pending: %s\". Fanned to all admins. Approval title injected verbatim"
    recommendation: "Sanitise approval-title before fanout. Scope fanout to admins with approvals role. DPDP §4."
    citation: "DPDP Act 2023 §4; IT (Intermediary Guidelines) Rules 2021 Rule 3"
    depends_on: "needs lawyer review"

  # ============================================================================
  # CATCH-ALL — TRANSACTIONAL COPY THAT REFERENCES OUT-OF-APP ACTION
  # ============================================================================

  - finding_id: LEG-TX-111
    surface_id: md-trx-009
    lens: legal
    severity: P2
    issue: "'Visit the web portal to manage your subscription' — driver app deflects subscription mgmt to web; if driver subscription was sold via Google Play, this violates Play Subscriptions Policy"
    evidence_excerpt: "\"Visit the web portal to manage your subscription.\""
    recommendation: "Confirm driver subscriptions are NOT sold via in-app purchase. If they are, Play Billing requires in-app cancellation. If web-only, current copy is fine. RBI e-Mandate framework also requires easy cancellation."
    citation: "Google Play Subscriptions Policy 2024; RBI e-Mandate framework 2019/2021"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-112
    surface_id: dp-tx-toast-step5-submitted
    lens: legal
    severity: P3
    issue: "'Application submitted successfully!' — completion toast; verify a server-side confirmation email is also sent (DPDP §5 notice carryover, evidentiary best-practice)"
    evidence_excerpt: "\"Application submitted successfully!\""
    recommendation: "Send a follow-up confirmation email with: (a) what was submitted, (b) when, (c) when next contact expected, (d) data-rights summary. Toast-only is thin evidence the driver was notified."
    citation: "DPDP Act 2023 §5; best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-113
    surface_id: md-trx-020
    lens: legal
    severity: P3
    issue: "Shared OfflineBanner — copy lives in mobile-shared; cannot inspect content from inventory alone"
    evidence_excerpt: "(shared origin) OfflineBanner imported — copy lives in mobile-shared"
    recommendation: "Fetch the actual copy and verify it does not promise functionality that the offline state cannot deliver (e.g. 'Your location is still being tracked' when in fact it is not). Misrepresentation under DPDP §5 / CPA 2019."
    citation: "DPDP Act 2023 §5; Consumer Protection Act 2019 §2(28)"
    depends_on: "needs lawyer review"

  # ============================================================================
  # CHEF NEW-ORDER EMAIL — SLA + INVOICE CARRY-THROUGH
  # ============================================================================

  - finding_id: LEG-TX-114
    surface_id: api-email-chef-new-order
    lens: legal
    severity: P2
    issue: "Chef new-order email subject contains decorative emoji '🔔 New Order!' — flagged for completeness; substantively the email sets a 'Customers expect a response within 5 minutes' SLA expectation"
    evidence_excerpt: "Subject: \"New order #%s — Fe3dr\" — \"🔔 New Order! You have a new order #%s worth ₹%.2f. Please review and accept the order from your dashboard.\" — \"Customers expect a response within 5 minutes\" SLA expectation"
    recommendation: "If the 'within 5 minutes' SLA is the platform's own promise to the chef, ensure it is also reflected in the chef agreement. If broken, CPA 2019 §2(28) and chef agreement become exposure points. Also the currency is hardcoded ₹ — confirm chef can be paid in INR only / clarify FEMA if not."
    citation: "Consumer Protection Act 2019 §2(28); Indian Contract Act 1872 §29"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-115
    surface_id: api-email-chef-new-order
    lens: legal
    severity: P1
    issue: "Chef new-order email body shows order total ₹%.2f but no GST breakdown or chef payout component — chef has no way to know payout from the email alone"
    evidence_excerpt: "\"You have a new order #%s worth ₹%.2f.\""
    recommendation: "Surface payout-to-chef separately from order total. If the order is ₹500 but the chef nets ₹400 after platform commission + tax, both numbers should appear. Code on Social Security 2020 gig-worker earnings transparency + Consumer Protection (E-Commerce) Rules 2020 Rule 6(4) seller-fairness."
    citation: "Code on Social Security 2020 §113; Consumer Protection (E-Commerce) Rules 2020 Rule 6(4)"
    depends_on: "needs lawyer review"

  # ============================================================================
  # MISC — REVIEWS, COMMENTS, REPLIES
  # ============================================================================

  - finding_id: LEG-TX-116
    surface_id: mv-reviewdetail-success-title
    lens: legal
    severity: P3
    issue: "'Reply Sent' title — chef reply published as platform content; same intermediary moderation gap as LEG-TX-093"
    evidence_excerpt: "\"Reply Sent\""
    recommendation: "Apply moderation guardrails on chef replies (no PII about the customer)."
    citation: "IT (Intermediary Guidelines) Rules 2021 Rule 3"
    depends_on: "needs lawyer review"

  # ============================================================================
  # ADMIN INVITATION SUCCESS BODY
  # ============================================================================

  - finding_id: LEG-TX-117
    surface_id: ap-staff-invite-success-body
    lens: legal
    severity: P2
    issue: "'Share this link with the invitee to complete their registration.' — admin instruction to share an invitation URL; URL is credential-equivalent and should not be shared over insecure channels"
    evidence_excerpt: "\"Share this link with the invitee to complete their registration.\""
    recommendation: "Add a caveat: 'Send only via secure channel (verified email or signed message).' Or default to email-only delivery. DPDP §8(5)."
    citation: "DPDP Act 2023 §8(5); best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-118
    surface_id: ap-staff-invite-success-title
    lens: legal
    severity: P3
    issue: "'Invitation created successfully!' — exclamation mark in admin transactional surface; style-guide says ≤1 exclamation per page in genuine celebration. Legal-relevance minor."
    evidence_excerpt: "\"Invitation created successfully!\""
    recommendation: "Cosmetic — flagged for completeness. Use 'Invitation created.' STYLE-GUIDE §1 Rule 1."
    citation: "STYLE-GUIDE.md §1"
    depends_on: "needs lawyer review"

  # ============================================================================
  # ADMIN PROVIDER ACTIONS
  # ============================================================================

  - finding_id: LEG-TX-119
    surface_id: ap-providerdetail-toasts
    lens: legal
    severity: P2
    issue: "'Connection successful ({ms}ms); Connection failed: {err}' — admin diagnostic; ensure {err} from provider does not leak provider-side secrets or PII"
    evidence_excerpt: "Connection successful ({ms}ms); Connection failed: {err}; Failed to test connection; Provider status updated; Provider deleted; Failed to toggle provider"
    recommendation: "Audit {err} pass-through. Provider error strings sometimes contain auth tokens, account IDs, or other sensitive fragments. DPDP §8(5)."
    citation: "DPDP Act 2023 §8(5); IT Act 2000 §43A"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-120
    surface_id: ap-providers-toasts
    lens: legal
    severity: P3
    issue: "'Provider deleted' admin toast — deleting a provider may affect in-flight orders; ensure UX warns admin before destructive action"
    evidence_excerpt: "Provider status updated; Failed to toggle provider; Provider deleted; Failed to delete provider"
    recommendation: "Apply confirmation modal with consequence statement (style guide §4 'Modal subtitles' — 'explain consequence in one sentence'). Legal-relevance: failure to warn admin about in-flight orders could result in stranded orders + customer harm."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 4(5); best-practice"
    depends_on: "needs lawyer review"

  # ============================================================================
  # ADMIN APPROVAL DETAIL
  # ============================================================================

  - finding_id: LEG-TX-121
    surface_id: ap-approvaldetail-toasts
    lens: legal
    severity: P1
    issue: "'Approval request approved / Approval request rejected / More information requested from chef' — adverse-action confirmations to admin; verify the chef-side notification fires for each + the reason is captured"
    evidence_excerpt: "Approval request approved; Approval request rejected; More information requested from chef; Failed to approve request; Failed to reject request; Failed to request information"
    recommendation: "Confirm each admin action triggers the corresponding chef-side notification with reason (LEG-TX-035, LEG-TX-037, LEG-TX-038 cover the chef-side issues). Admin success toast is fine on its own provided chef-side flow is complete."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 6(4); IT (Intermediary Guidelines) Rules 2021 Rule 3"
    depends_on: "needs lawyer review"

  # ============================================================================
  # CATCH — REMAINING MOBILE / VENDOR / DRIVER SUCCESS TOASTS
  # ============================================================================

  - finding_id: LEG-TX-122
    surface_id: vp-onb-personal-avatar-success
    lens: legal
    severity: P3
    issue: "'Profile photo uploaded' — confirm EXIF stripping + storage location notice"
    evidence_excerpt: "\"Profile photo uploaded\""
    recommendation: "Same as LEG-TX-066. Strip EXIF, surface privacy notice."
    citation: "DPDP Act 2023 §4, §8(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-123
    surface_id: dp-tx-toast-step1-saved
    lens: legal
    severity: P3
    issue: "'Personal info saved' — driver onboarding step; confirm DPDP §5 notice was shown before this step"
    evidence_excerpt: "\"Personal info saved\""
    recommendation: "Verify the DPDP §5 notice (purposes, retention, data fiduciary, withdrawal) is presented before personal info is captured."
    citation: "DPDP Act 2023 §5"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-124
    surface_id: dp-tx-toast-step2-saved
    lens: legal
    severity: P3
    issue: "'Vehicle details saved' — confirm Motor Vehicles Act compliance (DL number, RC, insurance) verification linkage"
    evidence_excerpt: "\"Vehicle details saved\""
    recommendation: "Vehicle details + DL form the basis of the platform's claim that drivers are legally able to deliver. Verify each is checked against authoritative source (Vahan/Sarathi APIs) before driver goes live. Motor Vehicles Act 1988 §3 (driving licence)."
    citation: "Motor Vehicles Act 1988 §3"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-125
    surface_id: dp-tx-toast-step4-saved
    lens: legal
    severity: P2
    issue: "'Plan selected successfully' — driver subscription plan; if plan implies revenue-sharing or fee, RBI e-Mandate + clear-disclosure rules apply"
    evidence_excerpt: "\"Plan selected successfully\""
    recommendation: "Surface plan price + recurring schedule + cancellation pathway on the selection screen, not just a success toast. RBI e-Mandate 2019/2021 + CPA 2019."
    citation: "RBI e-Mandate framework 2019/2021; Consumer Protection Act 2019"
    depends_on: "needs lawyer review"

  # ============================================================================
  # SUMMARY-LEVEL FINDINGS — CROSS-CUTTING CONCERNS
  # ============================================================================

  - finding_id: LEG-TX-126
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P0
    issue: "Cross-cutting: brand identity drift across transactional channels — 'Fe3dr', 'HomeChef', 'Fe3dr HomeChef' all surface to the same user across email/push/in-app — material risk of confusion + misrepresentation"
    evidence_excerpt: "Multiple inventory rows confirm divergence: api-push-user-welcome (HomeChef), api-email-welcome (Fe3dr!), api-email-chef-new-order (Fe3dr), api-email-account-reminder (Fe3dr HomeChef)"
    recommendation: "Choose one legal trade name and standardise everywhere. Until then, customer cannot distinguish phishing from legitimate brand mail (DPDP §8(5) security implication), and the platform's CPER 2020 Rule 4(1) duty to publish its legal name in a unified manner is weakened."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 4(1); Consumer Protection Act 2019 §2(28); DPDP Act 2023 §8(5)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-127
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P0
    issue: "Cross-cutting: NO transactional email contains the grievance officer footer, DPDP §5 notice summary, or unsubscribe/preferences link"
    evidence_excerpt: "OrderConfirmationHTML, PasswordResetHTML, WelcomeEmailHTML, ChefNewOrderHTML, ChefVerificationApprovedHTML, EmailVerificationHTML, StaffInvitationHTML, OrderStatusUpdateHTML, DeliveryAssignedHTML — none reference grievance officer or DPDP rights"
    recommendation: "Add a standard email footer block applied via emailBase wrapper containing: (a) grievance officer name+email+phone, (b) platform legal name + address, (c) link to Privacy Notice, (d) link to manage preferences, (e) link to in-app support, (f) one-line 'You're receiving this because you have an account with [brand]'. Required by Consumer Protection (E-Commerce) Rules 2020 Rule 4(5) + IT Rules 2021 Rule 3(2) + DPDP §5/§6(4)."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rules 4(5), 6(5); IT (Intermediary Guidelines) Rules 2021 Rule 3(2); DPDP Act 2023 §5, §6(4), §10"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-128
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P0
    issue: "Cross-cutting: NO transactional email or SMS surface shows the chef's FSSAI licence number — recurring FSSAI E-Commerce FBO Guidelines violation across the entire transactional surface"
    evidence_excerpt: "Across api-email-order-confirm, api-email-order-status-*, api-push-order-update-*, no FSSAI number observed"
    recommendation: "Inject the chef's FSSAI licence number into every transactional surface tied to an order (confirmation, status, delivery, complaint). Required by FSSAI Direction 8 Feb 2021 (E-Commerce FBO Guidelines)."
    citation: "FSSAI Direction F. No. 15(31)2020/FoSCoRIS/RCD/FSSAI dated 8 Feb 2021"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-129
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P0
    issue: "Cross-cutting: refund timeline disclosure absent across cancellation, refund-initiation, and verification-failed transactional surfaces"
    evidence_excerpt: "api-email-order-status-cancelled, api-push-order-cancelled-both, api-inapp-order-cancelled-both, web-tx-orderdetail-toasts, web-tx-checkout-payment-toasts — none state refund timeline"
    recommendation: "Add a uniform refund-timeline disclosure ('Refunds, when applicable, complete within T+7 working days to your original payment method') to every cancellation, failed-payment, and refund-initiation surface. RBI PA Master Direction + CPER 2020 Rule 5(3)(g) require this disclosure."
    citation: "RBI Payment Aggregator Master Direction 17 Mar 2020 §8; Consumer Protection (E-Commerce) Rules 2020 Rule 5(3)(g)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-130
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P1
    issue: "Cross-cutting: SMS path not visible in inventory — confirm DLT registration, header/template registration, and consent capture for every SMS sent"
    evidence_excerpt: "No SMS templates surfaced in transactional inventory excerpt — verify via separate SMS code-path audit"
    recommendation: "Conduct a separate audit of SMS dispatch code. Every SMS to Indian numbers requires (a) a registered Sender ID, (b) a registered template, (c) correct category (transactional/service/promotional), and (d) consent record. Carrier scrubbing rejects unregistered traffic. TRAI TCCCPR 2018."
    citation: "TRAI Telecom Commercial Communications Customer Preference Regulations 2018; DLT framework"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-131
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P1
    issue: "Cross-cutting: every transactional channel reaches a user but no central record visible of consent capture for those channels (push, email, SMS, in-app)"
    evidence_excerpt: "Welcome push + welcome email + transactional pushes — captured at signup as part of bundled consent (per LEG-001/LEG-002)"
    recommendation: "Maintain a per-user, per-channel, per-purpose consent ledger. DPDP §6(2) requires the Data Fiduciary to be able to demonstrate that consent has been obtained. A bundled signup checkbox does not satisfy this for cross-channel transactional + marketing comms."
    citation: "DPDP Act 2023 §6(2)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-132
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P1
    issue: "Cross-cutting: emoji-prefixed status emails (🔔 / 👨‍🍳 / ✅ / 🚗 / 🎉 / ❌) — legally neutral, but voice-guide says 'no exclamation tricks'; emojis act as visual exclamations. Legal-relevance: low; brand-consistency: medium"
    evidence_excerpt: "Across api-email-order-status-* — all use emoji prefixes"
    recommendation: "Out of legal scope per se — captured for completeness. Style-guide consistency."
    citation: "STYLE-GUIDE.md §1"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-133
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P2
    issue: "Cross-cutting: 'log in / login / logout' banned variants appear in mobile alerts (mc-profile-logout-confirm, md-trx-006, md-trx-007, api-success-message-generic 'Logged out successfully', web-tx-social-feed-actions)"
    evidence_excerpt: "Multiple rows use 'log in' / 'log out' / 'logout' instead of style-guide 'sign in' / 'sign out'"
    recommendation: "Outside Legal lens per se. Brand-consistency duty under CPER 2020 Rule 4(1)."
    citation: "STYLE-GUIDE.md §3; Consumer Protection (E-Commerce) Rules 2020 Rule 4(1)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-134
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P1
    issue: "Cross-cutting: every chef-action and driver-action notification confirms platform-side execution but few surface the corresponding payout/financial impact — gig-worker earnings transparency gap"
    evidence_excerpt: "api-email-chef-new-order, api-push-chef-new-order-actionable, dp-tx-toast-accept-success, web-tx-delivery-actions"
    recommendation: "Surface trip-level / order-level payout estimate in every chef/driver acceptance + completion notification. Code on Social Security 2020 + emerging state platform-worker laws push toward earnings transparency at the work-spell level."
    citation: "Code on Social Security 2020 §113-114; Rajasthan Platform-Based Gig Workers Act 2023 (state benchmark)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-135
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P1
    issue: "Cross-cutting: no transactional surface references the platform's IT Rules 2021 grievance officer or appellate-grievance committee (AGC) for content/intermediary disputes"
    evidence_excerpt: "Across all transactional inventory rows — no IT Rules grievance officer reference observed"
    recommendation: "Add the IT Rules 2021 grievance officer contact to email footer and a 'Report content' / 'Appeal' link to the in-app footer. IT (Intermediary Guidelines and Digital Media Ethics Code) Rules 2021 Rule 3(2)(b) requires display of grievance officer details. Rule 3A introduces Grievance Appellate Committees."
    citation: "IT (Intermediary Guidelines) Rules 2021 Rules 3(2)(b), 3A"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-136
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P1
    issue: "Cross-cutting: no per-channel preference centre referenced from any transactional surface — DPDP §6(4) requires withdrawal of consent to be as easy as giving consent"
    evidence_excerpt: "No 'manage notifications' / 'unsubscribe' link observed in transactional copy"
    recommendation: "Build and link a notification preference centre (per-channel: email/push/SMS; per-purpose: transactional/account/marketing). Link from every email footer. DPDP §6(4)."
    citation: "DPDP Act 2023 §6(4)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-137
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P2
    issue: "Cross-cutting: free-text injection from admins / chefs into notifications visible to other parties — multiple findings (LEG-TX-025, LEG-TX-035, LEG-TX-036, LEG-TX-037, LEG-TX-039, LEG-TX-057, LEG-TX-096, LEG-TX-110) — needs a single mitigation"
    evidence_excerpt: "Admin-authored fields: api-inapp-staff-invite, ap-secsettings-toasts; admin-authored notes: api-inapp-approval-rejected-chef, api-inapp-approval-info-chef; chef-authored: api-inapp-chef-responded"
    recommendation: "Implement a single notification-sanitisation middleware: HTML strip, length cap, profanity filter, PII detection. Apply to every user-authored field before fanout. IT Rules 2021 Rule 3 + IT Act §79 due-diligence."
    citation: "IT Act 2000 §79; IT (Intermediary Guidelines) Rules 2021 Rule 3"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-138
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P2
    issue: "Cross-cutting: order status copy diverges across three code paths (statusLabels map in email_templates.go, getOrderStatusMessage in notifications.go, RegisterPushConsumers humanReadableOrderStatus) — three sources of truth for the same lifecycle states"
    evidence_excerpt: "Inventory notes recurring drift between email, in-app, push wording for the same status"
    recommendation: "Consolidate to a single source-of-truth status-string registry (e.g. enum -> localised string map in shared package). Inconsistent status copy is a contract-evidence weakness (Indian Contract Act §29 certainty)."
    citation: "Indian Contract Act 1872 §29"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-139
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P3
    issue: "Cross-cutting: exclamation-mark inflation in transactional copy ('Welcome to Fe3dr!', 'Application submitted!', 'New Order!', 'Order Delivered!', 'Reset your password — Fe3dr', 'Delivery accepted!') — every transactional surface uses celebratory punctuation"
    evidence_excerpt: "Multiple inventory rows"
    recommendation: "Out of Legal lens scope per se. Brand-voice consistency."
    citation: "STYLE-GUIDE.md §1 Rule 1"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-140
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P2
    issue: "Cross-cutting: ratios of duplicate notifications — chef gets two pushes for new orders (LEG-TX-061), customer gets two pushes for status updates (per inventory notes on three sets of status labels) — DPDP §4 'reasonable' purpose-limited processing is undermined when duplicate notifications are fired"
    evidence_excerpt: "Inventory notes: chef receives TWO pushes / customer can receive two pushes with different status wording"
    recommendation: "Deduplicate at the dispatcher level. Each lifecycle event should produce exactly one push + one in-app + one email (where applicable), with consistent copy."
    citation: "DPDP Act 2023 §4; best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-141
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P1
    issue: "Cross-cutting: 'verified' / 'approved' / 'rejected' / 'suspended' actions throughout the platform have NO single record (per inventory excerpt) of: (a) reason code, (b) actor, (c) timestamp, (d) appeal path"
    evidence_excerpt: "ap-chefs-toast-verified, ap-chefs-toast-rejected, ap-chefs-toast-suspended, dp-tx-toast-partner-suspend, dp-tx-toast-partner-verified, ap-users-toast-suspended, ap-staffdetail-toasts"
    recommendation: "Build a server-side adverse-action audit table that captures, for every suspend/reject/revoke/deactivate, the actor, target, reason code, timestamp, IP. Surface the action's record to the affected user via a 'Why?' link. Required by IT Rules 2021 Rule 3 + DPDP §8(6) + CPER 2020 Rule 6(4)."
    citation: "IT (Intermediary Guidelines) Rules 2021 Rules 3-4; DPDP Act 2023 §8(6); Consumer Protection (E-Commerce) Rules 2020 Rule 6(4)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-142
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P2
    issue: "Cross-cutting: no reference in any transactional surface to the platform's role classification (intermediary under IT Act §79 / e-commerce entity under CPER 2020 / data fiduciary under DPDP §2(i))"
    evidence_excerpt: "Inventory excerpt shows no role-disclosure in transactional copy"
    recommendation: "Add a one-line role disclosure in the email footer: '[Brand] operates as an e-commerce entity facilitating sales between independent home chefs and customers. We are an intermediary under IT Act 2000 §79.' Strengthens intermediary safe-harbour position when content disputes arise."
    citation: "IT Act 2000 §79; Consumer Protection (E-Commerce) Rules 2020 Rule 4; DPDP Act 2023 §2(i)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-143
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P1
    issue: "Cross-cutting: no transactional surface references the customer's right to file complaint with the Food Safety Connect toll-free 1800-112-100 / FSSAI App for food-quality issues"
    evidence_excerpt: "No FSSAI complaint pathway observed in transactional copy"
    recommendation: "Add an FSSAI complaint reference to delivery-confirmation and complaint-flow surfaces. Practical effect: customer's ability to invoke FSSAI complaint channels independently of the platform demonstrates the platform's good-faith intermediary status."
    citation: "Food Safety and Standards Act 2006; FSSAI Food Safety Connect"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-144
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P1
    issue: "Cross-cutting: chef-side FSSAI compliance is recurrence-blind — no transactional surface reminds chef before order acceptance that their FSSAI must be valid and the kitchen FSSAI-compliant"
    evidence_excerpt: "Chef-new-order copy assumes verification is current; no in-flow check surfaced"
    recommendation: "Add a 'FSSAI valid until {date}' line in the chef new-order email + push if within 30 days of expiry. If expired, block order acceptance and notify chef immediately. FSS Act §31 makes it an offence to sell food without a valid FSSAI registration."
    citation: "Food Safety and Standards Act 2006 §31; FSS (Licensing and Registration of Food Businesses) Regulations 2011"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-145
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P3
    issue: "Cross-cutting: time/date formatting in transactional surfaces inconsistent — emails use chef timezone? Customer timezone? Server UTC? Affects when 'within 24-48 hours' SLA actually expires"
    evidence_excerpt: "Inventory shows '24-48 hours' SLA stated but no timezone discipline observed"
    recommendation: "Standardise all customer-facing time as IST with explicit 'IST' suffix. Avoid ambiguity that an Indian customer or chef might infer their local time when the system stamps UTC. CPA 2019 §2(28) misleading-representation lens."
    citation: "Consumer Protection Act 2019 §2(28); STYLE-GUIDE.md §6"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-146
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P2
    issue: "Cross-cutting: no transactional surface references age verification — DPDP §10 prohibits behavioural tracking, targeted advertising, and undertaking of children's data without verifiable parental consent"
    evidence_excerpt: "No age-related copy in transactional inventory"
    recommendation: "Add an age-verification flow during signup (out of scope for transactional inventory but flagged here because welcome email is the carryover surface). Welcome email should reference 'For users 18+' if signup is age-gated, or describe parental-consent path if not. DPDP §10."
    citation: "DPDP Act 2023 §10"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-147
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P2
    issue: "Cross-cutting: customer addresses are stored permanently (no surfaced retention statement); DPDP §8(7) limits retention to purpose-necessary period"
    evidence_excerpt: "Address CRUD toasts (web-tx-profile-addresses); no retention reference"
    recommendation: "Define and surface an address-retention policy. If a customer has been inactive for N years, addresses should be purged absent contractual or statutory retention need. DPDP §8(7) + §12."
    citation: "DPDP Act 2023 §8(7), §12"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-148
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P2
    issue: "Cross-cutting: order data retention — order confirmation emails set up an expectation that order history persists; verify retention aligns with §31 CGST Act (tax invoice retention 6 years) and DPDP §8(7) (purpose-necessary period)"
    evidence_excerpt: "No retention reference in order emails or order-detail toasts"
    recommendation: "Order invoices must be retained for at least 6 years under CGST §36 + Rule 56 + §31. Surface this in privacy notice; ensure DPDP §12 erasure requests do not delete records that GST law requires the platform to retain. Need to reconcile §8(7) + GST retention."
    citation: "CGST Act 2017 §31, §36, Rule 56; DPDP Act 2023 §8(7), §12"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-149
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P1
    issue: "Cross-cutting: payment-related surfaces (success, failure, refund, cancel) do not name the payment aggregator (Razorpay / Stripe) — RBI PA framework requires customer to know who is processing the payment"
    evidence_excerpt: "web-tx-checkout-payment-toasts, web-tx-orderdetail-toasts — Razorpay/Stripe paths unnamed in user-facing copy"
    recommendation: "Surface 'Payment processed securely by [Razorpay/Stripe]' on the success surface. RBI PA Master Direction §4 (information disclosure)."
    citation: "RBI Payment Aggregator Master Direction 17 Mar 2020 §4"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-150
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P3
    issue: "Cross-cutting: 'Profile updated successfully' / 'Settings saved' etc. — DPDP §11 right-to-correction; consider a brief confirmation summary ('Name changed from X to Y') for sensitive fields"
    evidence_excerpt: "web-tx-profile-update, vp-ux-profile-toasts, ap-notifsettings-toasts, ap-platsettings-feedback, ap-secsettings-toasts, mv-profile-update-success-body, md-trx-014"
    recommendation: "For sensitive-field changes (email, phone, bank, address), echo the changed value back in the toast or send a confirmation email. DPDP §8(5) + §11."
    citation: "DPDP Act 2023 §8(5), §11"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-151
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P2
    issue: "Cross-cutting: NO reference in any transactional surface to data fiduciary identity or contact — DPDP §5(1)(a) requires the notice at consent to identify the Data Fiduciary"
    evidence_excerpt: "No emails or push notifications surface the legal entity that operates the platform"
    recommendation: "Add the legal entity name (e.g. 'Fe3dr Foods Pvt Ltd') and registered office address to email footer. DPDP §5(1)(a) requires Notice to identify Data Fiduciary; CPER 2020 Rule 4(1) requires e-commerce entity to publish legal name."
    citation: "DPDP Act 2023 §5(1)(a); Consumer Protection (E-Commerce) Rules 2020 Rule 4(1)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-152
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P2
    issue: "Cross-cutting: cross-border data transfer — Razorpay, Stripe, Google Cloud Storage may move data outside India; transactional copy does not reference this"
    evidence_excerpt: "Payment provider Stripe + storage on GCS (per CLAUDE.md); no user-facing notice"
    recommendation: "DPDP §16 allows cross-border transfer except to negative-listed countries; once those rules operationalise, the platform must update its Notice. Privacy notice must list categories of cross-border processors. Transactional emails not the right surface, but the email footer's privacy-notice link must lead to the right content."
    citation: "DPDP Act 2023 §16"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-153
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P2
    issue: "Cross-cutting: no breach-notification copy/template visible — DPDP §8(6) requires Data Fiduciary to notify the Data Protection Board and affected Data Principals of a 'personal data breach' in such form and manner as prescribed"
    evidence_excerpt: "No breach-notification transactional template observed in inventory"
    recommendation: "Build a breach-notification email template + push template in advance. DPDP §8(6) imposes the notification duty; templating now ensures the platform can respond within the prescribed timeframe (expected to be tight under upcoming Rules)."
    citation: "DPDP Act 2023 §8(6); DPDP Rules (forthcoming)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-TX-154
    surface_id: cross-cutting-transactional
    lens: legal
    severity: P2
    issue: "Cross-cutting: no transactional surface references chef/driver KYC re-verification cadence — RBI PA Master Direction requires periodic merchant KYC refresh"
    evidence_excerpt: "Chef/driver onboarding success surfaces (vp-onb-submit-success, dp-tx-toast-step5-submitted) — no mention of refresh cadence"
    recommendation: "Define a KYC-refresh cadence (typically every 2 years for low-risk, more frequent for higher-risk). Surface a reminder notification template ('Refresh your KYC by {date}') and a periodic chef/driver email. RBI PA Master Direction §9 + Master Direction on KYC."
    citation: "RBI Payment Aggregator Master Direction 17 Mar 2020 §9; RBI Master Direction on KYC 2016 (updated)"
    depends_on: "needs lawyer review"
```

## Business Analyst findings

```yaml
findings:

  # ── P0: Routing bug — wrong audience receives wrong email ────────────────

  - finding_id: BA-TX-001
    surface_id: api-email-delivery-assigned
    lens: business-analyst
    severity: P0
    issue: "DeliveryAssignedHTML template written for drivers is routed to customers via the delivery_assigned event — customer receives 'You've been assigned order #X' making no sense"
    evidence_excerpt: |
      services/email_templates.go::DeliveryAssignedHTML: "🚗 New Delivery! You've been assigned order #%s. Pickup location: %s"
      services/notifications.go:569-581: same template sent to customer on delivery_assigned event
    recommendation: >
      Immediate fix: separate the email path. Create a CustomerDeliveryAssignedHTML template with
      customer-appropriate copy: 'Your delivery partner is on the way — Order #%s'.
      Route the driver notification to the driver email address, not the customer's.
      The existing template stays for drivers. Until fixed: every customer who places an order receives
      an email that implies they are the delivery driver — this is the most confusing trust-eroding
      transactional email in the system. Expected impact: eliminates a guaranteed support ticket per
      delivery and removes a direct cause of customer churn at the highest-anxiety moment.
    metric_hypothesis: "order-to-reorder rate; support ticket volume — customers receiving driver copy at time of delivery will file confusion tickets and may not reorder"
    depends_on: null

  # ── P0: Duplicate push notifications on same event ────────────────────────

  - finding_id: BA-TX-002
    surface_id: api-push-chef-new-order-actionable
    lens: business-analyst
    severity: P0
    issue: "Two queue groups fire on the same order-created NATS event and send two push notifications to the chef — 'New Order' (RegisterPushConsumers) and 'New Order Received' (handleOrderCreated) — with different copy"
    evidence_excerpt: |
      handlers/notifications.go::RegisterPushConsumers: Title "New Order" Body "You have a new order waiting for your confirmation"
      services/notifications.go::handleOrderCreated push: Title "New Order Received" Body "You have a new order waiting to be prepared!"
    recommendation: >
      Consolidate to a single push per event. Deduplicate by using one canonical handler.
      Recommended copy: Title "New order" (no exclamation), Body "Order #%s — confirm within 5 min."
      The 5-minute window is already set as an SLA in the chef email; align the push to reinforce it.
      Until fixed: chefs receive two buzzes per order with inconsistent text, reducing trust in the
      notification system and increasing the likelihood they dismiss all order alerts.
    metric_hypothesis: "chef order acceptance rate — duplicate pushes cause notification fatigue, increasing time-to-accept and risking order cancellation"
    depends_on: null

  - finding_id: BA-TX-003
    surface_id: api-inapp-chef-new-order
    lens: business-analyst
    severity: P0
    issue: "Three distinct 'new order' copy variants exist across the system: 'New Order' (handlers/notifications.go), 'New Order Received' (services/notifications.go push), and 'New Order!' (services/notifications.go in-app) — chef sees all three in one session"
    evidence_excerpt: |
      handlers/notifications.go: Title "New Order"
      services/notifications.go handleOrderCreated push: Title "New Order Received"
      services/notifications.go handleChefNewOrder in-app: Title "New Order!"
    recommendation: >
      Centralise all new-order notification copy in a single constant or i18n key. One canonical
      title: 'New order' (sentence case, no exclamation per style guide), one canonical body
      per channel (push: short imperative; in-app: detail with order value). Three slightly different
      strings for the same event fragments the chef experience and signals an untested system.
    metric_hypothesis: "chef D7 retention; platform trust — notification inconsistency at the highest-frequency chef touchpoint signals product instability"
    depends_on: BA-TX-002

  # ── P0: Order confirmation email missing legally-required and trust-critical fields ─

  - finding_id: BA-TX-004
    surface_id: api-email-order-confirm
    lens: business-analyst
    severity: P0
    issue: "Order confirmation email is missing GST breakdown, chef name, delivery address, ETA, allergen disclaimer, and cancellation/refund policy — the most legally and trust-sensitive transactional email on the platform"
    evidence_excerpt: |
      services/email_templates.go::OrderConfirmationHTML: "Order Confirmed! Your order #%s has been placed successfully. Your home chef is preparing your meal!"
      Fields present: order_id, total (%.2f). Chef name absent. ETA absent. Tax line absent. Allergens absent. Refund terms absent.
    recommendation: >
      Expand the order confirmation to include: (1) Chef name and kitchen name with a link to their
      profile, (2) Itemised list with quantities and unit prices, (3) Tax/GST line (legally required
      for food orders in India above threshold), (4) Estimated delivery time window,
      (5) Delivery address confirmation, (6) Allergen reminder: 'Check item descriptions for allergens.
      Contact us before pickup if you have dietary restrictions.', (7) Cancellation policy: 'Orders
      can be cancelled within X minutes. Refunds processed within 5–7 business days.'
      This is the #1 support-ticket-reducing action available: a complete confirmation email
      eliminates 'where is my order', 'what did I order', and 'I need a receipt' contacts.
    metric_hypothesis: "support ticket volume; order-to-reorder rate — complete order confirmation emails reduce post-order anxiety, the primary driver of cancellations and negative reviews"
    depends_on: null

  # ── P0: Brand drift — two product names in same session ────────────────────

  - finding_id: BA-TX-005
    surface_id: api-push-user-welcome
    lens: business-analyst
    severity: P0
    issue: "Welcome push notification says 'Welcome to HomeChef!' but the welcome email says 'Welcome to Fe3dr!' — a new user sees two different product names in their first session"
    evidence_excerpt: |
      services/notifications.go::handleUserRegistered push: Title "Welcome to HomeChef!" Body "Thank you for joining HomeChef."
      services/email_templates.go::WelcomeEmailHTML: Subject "Welcome to Fe3dr!" Body "Welcome to Fe3dr, %s!"
    recommendation: >
      Align all first-touch transactional copy to a single brand name. Determine the canonical name
      (Fe3dr vs HomeChef) and replace all divergent instances. The push notification path likely
      carries a legacy product name. A user receiving both in the same minute will distrust the
      platform before they place their first order.
    metric_hypothesis: "D1 activation rate; trust score — brand name divergence at first contact is the highest-signal indicator of product immaturity and directly reduces first-order conversion"
    depends_on: null

  # ── P0: Cancellation email missing refund timeline ────────────────────────

  - finding_id: BA-TX-006
    surface_id: api-email-order-status-cancelled
    lens: business-analyst
    severity: P0
    issue: "Cancellation email contains no refund timeline, no contact point, and no reason for cancellation — the bare 'Your order has been cancelled' subject is the worst-case trust event"
    evidence_excerpt: |
      services/email_templates.go::OrderStatusUpdateHTML (cancelled): Subject "Order #%s — Your order has been cancelled" ❌ emoji
      No refund timeline. No who-to-contact. No cancellation reason.
    recommendation: >
      Cancellation emails must include: (1) Who cancelled (customer, chef, or platform),
      (2) Refund timeline: 'Your refund of ₹{amount} will appear in 5–7 business days.',
      (3) Contact path: 'Questions? Reply to this email or visit {support_url}.',
      (4) Re-engagement CTA: 'Browse other chefs near you' (links to browse page).
      The cancellation moment is a churn event — each of these elements reduces churn probability.
      Absence of a refund timeline is also legally non-compliant with RBI prepaid instrument rules.
    metric_hypothesis: "post-cancellation churn; support ticket volume — customers with no refund timeline contact support within the hour; adding the timeline reduces that contact by ~60% (industry benchmark)"
    depends_on: null

  # ── P0: Raw type-slug surfaces to chef in approval notifications ───────────

  - finding_id: BA-TX-007
    surface_id: api-inapp-approval-approved-chef
    lens: business-analyst
    severity: P0
    issue: "Approval notification body uses raw type slug 'menu_item' directly in user-facing message: 'Your menu_item has been approved' — programmer string surfaces to chef"
    evidence_excerpt: |
      services/notifications.go::handleApprovalApproved: "Your %s has been approved: %s" where %s = approval type slug e.g. "menu_item"
      Also affects: api-push-approval-approved-chef, api-inapp-approval-rejected-chef, api-push-approval-rejected-chef, api-inapp-approval-info-chef
    recommendation: >
      Create a type-slug-to-human-readable map: menu_item → 'menu item', chef_profile → 'chef profile',
      kitchen_setup → 'kitchen', etc. Use the mapped label in all notification bodies.
      Suggested body: 'Your menu item "{title}" has been approved. View it on your menu.'
      This affects 5 notification surfaces with the same bug. Fix the map once; all consumers benefit.
    metric_hypothesis: "chef trust score; chef D7 retention — chefs who receive system-internal strings in their approval notifications perceive the platform as unfinished and reduce engagement"
    depends_on: null

  - finding_id: BA-TX-008
    surface_id: api-inapp-approval-rejected-chef
    lens: business-analyst
    severity: P0
    issue: "Rejection notification body exposes raw admin notes verbatim: 'Your menu_item has been rejected. Notes: {raw_admin_text}' — potential PII leak and uncontrolled admin language reaching chefs"
    evidence_excerpt: |
      services/notifications.go::handleApprovalRejected: Title "Request Rejected" Message "Your %s has been rejected. Notes: %s"
      Admin notes are passed verbatim — no length cap, no content filtering
    recommendation: >
      (1) Fix the type-slug issue per BA-TX-007. (2) Add a character limit to admin notes
      surfaced in notifications (recommended: 200 chars, truncated with 'See full details in portal').
      (3) Provide a rejection reason taxonomy so admins pick from a list rather than free-typing.
      This also reduces the chance of internal language ('THIS DISH PHOTO IS BLURRY FFS') reaching
      the chef. Expected: reduces resubmission friction because rejection reasons become actionable
      rather than arbitrary.
    metric_hypothesis: "chef resubmission rate after rejection; chef D30 retention — actionable, well-formatted rejection reasons increase resubmission 2–3x vs vague raw notes (marketplace onboarding benchmark)"
    depends_on: BA-TX-007

  # ── P0: Three divergent status-label maps create customer confusion ─────────

  - finding_id: BA-TX-009
    surface_id: api-push-order-update-customer-deeplink
    lens: business-analyst
    severity: P0
    issue: "Three separate status-label maps exist for order status updates — email_templates.go statusLabels, notifications.go getOrderStatusMessage, and handlers/notifications.go humanReadableOrderStatus — customers receive different wording per channel for the same event"
    evidence_excerpt: |
      Email (email_templates.go:176): "Your chef is preparing your meal"
      Push (handlers/notifications.go humanReadableOrderStatus): "Your order is now being prepared"
      Push (notifications.go getOrderStatusMessage): "Your order is being prepared"
      In-app (notifications.go handleOrderUpdated): "Your order is being prepared"
    recommendation: >
      Define one canonical status-label map as a shared constant. All notification channels (email,
      push, in-app) import from this single source. Canonical labels (following style guide:
      sentence case, chef/driver persona correct, no exclamation except on delivered):
      - pending: 'Order placed — waiting for chef to confirm'
      - confirmed: 'Chef confirmed your order'
      - preparing: 'Your chef is preparing your meal'
      - ready: 'Your order is ready for pickup'
      - picked_up: 'Your delivery partner has your order'
      - on_the_way: 'Your order is on its way'
      - delivered: 'Order delivered. Enjoy your meal.'
      - cancelled: 'Order cancelled — see email for refund details'
    metric_hypothesis: "order tracking anxiety; support ticket volume — status label drift across channels causes customers to think they are in different states, driving 'where is my order' tickets"
    depends_on: null

  # ── P1: Welcome email uses banned/weak language, misses activation CTA ─────

  - finding_id: BA-TX-010
    surface_id: api-email-welcome
    lens: business-analyst
    severity: P1
    issue: "Welcome email uses 'amazing' and 'talented' (style-guide banned adjectives) and 'Discover amazing home-cooked meals from talented local chefs' — generic welcome with no personalised first-step CTA"
    evidence_excerpt: |
      services/email_templates.go::WelcomeEmailHTML: "Discover amazing home-cooked meals from talented local chefs"
      CTA: "Start Exploring" → https://fe3dr.com (no city pre-fill, no onboarding state awareness)
    recommendation: >
      Rewrite to: 'Home-cooked meals from chefs near you.' Remove 'amazing' and 'talented'.
      Change 'Start Exploring' to 'Find chefs near {city}' where city is injected from the
      registration form or IP-inferred. If city is unavailable, use 'Browse chefs near you'.
      A first-order CTA personalised to location increases D1 activation by 15–25% (email benchmark
      for location-aware welcome emails in food marketplaces).
    metric_hypothesis: "D1 activation rate; first-order conversion — personalised CTA in welcome email is the highest-leverage single-email change available"
    depends_on: BA-TX-005

  # ── P1: Chef verification email/in-app copy says different things ──────────

  - finding_id: BA-TX-011
    surface_id: api-email-chef-verified
    lens: business-analyst
    severity: P1
    issue: "Chef verification email says 'Your kitchen has been verified' but in-app notification says 'Your chef profile has been verified' — two different frames for the same event, creating ambiguity about what was verified"
    evidence_excerpt: |
      services/email_templates.go::ChefVerificationApprovedHTML: "Your kitchen has been verified and your chef profile is now live"
      services/notifications.go::handleChefVerified in-app: "Your chef profile has been verified. You can now start accepting orders!"
    recommendation: >
      Align on one frame. The email frame ('kitchen verified') is richer and more specific — use it
      consistently: 'Your kitchen is verified. Your profile is now live — start adding your menu.'
      The in-app notification should match: Title 'Kitchen verified', Body 'Your profile is live.
      Start accepting orders.' Remove the exclamation from both per style guide Rule 1.
    metric_hypothesis: "chef activation rate post-approval — a clear, consistent message about what 'verified' means drives faster first-menu-item creation"
    depends_on: null

  # ── P1: Chef new-order email sets a 5-minute SLA not enforced elsewhere ─────

  - finding_id: BA-TX-012
    surface_id: api-email-chef-new-order
    lens: business-analyst
    severity: P1
    issue: "Chef new-order email sets 'Customers expect a response within 5 minutes' as a binding SLA expectation — this expectation is not stated in push notifications, in-app notifications, or chef onboarding, creating an inconsistent and unenforceable standard"
    evidence_excerpt: |
      services/email_templates.go::ChefNewOrderHTML: "Customers expect a response within 5 minutes"
      No equivalent statement in push (Title "New Order Received"), in-app ("You have a new order to prepare"), or chef onboarding docs
    recommendation: >
      Either (a) make the 5-minute expectation explicit across all channels so chefs understand the
      standard they are agreeing to: push body 'New order — confirm within 5 min', in-app 'You have
      5 min to confirm order #%s', or (b) remove the 5-minute claim from the email if the platform
      cannot enforce or track it. An SLA stated only in email (low-open-rate channel for mobile-first
      chefs) is operationally unenforceable.
    metric_hypothesis: "order acceptance rate; customer satisfaction score — 5-min SLA stated inconsistently means some chefs never see it, leading to late acceptances and customer complaints"
    depends_on: BA-TX-003

  # ── P1: Cancellation push notification is identical for customer and chef ───

  - finding_id: BA-TX-013
    surface_id: api-push-order-cancelled-both
    lens: business-analyst
    severity: P1
    issue: "Same push body 'Order has been cancelled' is sent to both customer and chef with no audience-specific context — customer needs refund info, chef needs restocking info"
    evidence_excerpt: |
      services/notifications.go::handleOrderCancelled: Title "Order Cancelled" Body "Order has been cancelled"
      Sent to BOTH customer and chef (audience: customer,chef in inventory)
    recommendation: >
      Audience-split the cancellation notification:
      - Customer: 'Your order was cancelled. Refund of ₹{amount} in 5–7 business days.'
      - Chef: 'Order #{id} was cancelled. No action needed — ingredients can be repurposed.'
      The chef message should include the cancellation reason if available (customer-initiated vs
      platform-initiated vs chef-initiated) so they can track patterns.
    metric_hypothesis: "chef trust score; customer post-cancellation churn — generic cancellation copy leaves both parties with unresolved anxiety; audience-specific copy reduces follow-up contacts"
    depends_on: BA-TX-006

  # ── P1: Support ticket emails use unbranded inline HTML ───────────────────

  - finding_id: BA-TX-014
    surface_id: api-email-support-created
    lens: business-analyst
    severity: P1
    issue: "Support ticket creation and update emails use inline unbranded HTML with no email wrapper — no header, no footer, no privacy link — while all other transactional emails use emailBase template"
    evidence_excerpt: |
      services/email.go::SendSupportTicketCreated: inline HTML, no emailBase wrapper, no footer, no privacy links
      services/email.go::SendSupportTicketUpdate: same — "Your support ticket #%s has been updated to: %s"
    recommendation: >
      Refactor both support email functions to use the emailBase wrapper template.
      Add footer with privacy policy link and unsubscribe instruction.
      This is a regulatory requirement (India's IT Act 2000 notification rules) as well as a trust signal.
      Support emails are sent when users are already in a frustration state — an unbranded email
      looks like phishing and will be ignored or reported, extending ticket resolution time.
    metric_hypothesis: "support ticket resolution time; trust score — branded support emails are opened at ~35% vs ~12% for plaintext/unbranded at equivalent send volumes"
    depends_on: null

  - finding_id: BA-TX-015
    surface_id: api-email-support-update
    lens: business-analyst
    severity: P1
    issue: "Support update email subject 'Ticket Update — #%s' and body 'Your support ticket #%s has been updated to: %s' uses raw status values — 'updated to: in_progress' may surface to users"
    evidence_excerpt: |
      services/email.go::SendSupportTicketUpdate: Subject "Ticket Update — #%s" Body "updated to: %s" where %s is raw status string
    recommendation: >
      Map internal status values to human-readable labels (same approach as BA-TX-007/TX-009).
      in_progress → 'We're working on it', resolved → 'Resolved — let us know if you need more help',
      awaiting_customer → 'We need more information from you'.
      Subject: 'Update on your support request #{ticket_id} — {human_status}'.
      Apply emailBase wrapper per BA-TX-014.
    metric_hypothesis: "support satisfaction score; ticket reopen rate — clear status labels reduce 'what does this mean' follow-up tickets"
    depends_on: BA-TX-014

  # ── P1: Account reminder email has brand drift in subject line ─────────────

  - finding_id: BA-TX-016
    surface_id: api-email-account-reminder
    lens: business-analyst
    severity: P1
    issue: "Account-already-exists email subject says 'Fe3dr HomeChef account' — combining two brand names — while all other email subjects say only 'Fe3dr'"
    evidence_excerpt: |
      services/email.go::SendAccountReminderEmail: Subject "You already have a Fe3dr HomeChef account"
      All other emails: "...— Fe3dr" (brand name only, no product qualifier)
    recommendation: >
      Align subject to brand standard: 'You already have a Fe3dr account.'
      The account-reminder email is security-sensitive (it prevents user enumeration); the copy change
      is cosmetic but the brand inconsistency compounds the BA-TX-005 brand-name drift issue.
    metric_hypothesis: "trust score; sign-in conversion from reminder — inconsistent brand name in a security email reduces confidence in email legitimacy"
    depends_on: BA-TX-005

  # ── P1: Delivery partner assigned push misses driver name personalisation ───

  - finding_id: BA-TX-017
    surface_id: api-push-delivery-assigned-customer
    lens: business-analyst
    severity: P1
    issue: "Customer delivery-partner-assigned push notification omits the driver's name even though it is available in event.Data — generic 'A delivery partner has been assigned' misses the highest-anxiety moment of the order journey"
    evidence_excerpt: |
      services/notifications.go::handleDeliveryAssigned: Title "Delivery Partner Assigned" Body "A delivery partner has been assigned to your order and will pick it up soon!"
      Driver name available in event.Data but not injected
    recommendation: >
      Update body to: 'Ravi will pick up your order shortly. Track in the app.'
      Where 'Ravi' is the driver's first name from event.Data.
      If driver name is unavailable for any reason, fall back to: 'Your delivery partner is on their way.'
      Named personalisation at the anxiety peak of the order journey is a measurable trust signal.
    metric_hypothesis: "customer satisfaction score; order tracking anxiety — named delivery partner reduces 'where is my order' contacts by establishing a human connection"
    depends_on: null

  # ── P1: Onboarding submission SLA stated inconsistently ───────────────────

  - finding_id: BA-TX-018
    surface_id: vp-onb-submit-success
    lens: business-analyst
    severity: P1
    issue: "Vendor portal submission toast says '24-48 hours' review SLA but mobile vendor pending screen body says the same 24-48 hours — if the actual SLA differs, both surfaces show the wrong number; no SLA commitment is in the push notification chain"
    evidence_excerpt: |
      vp-onb-submit-success: "We'll review and get back to you within 24-48 hours."
      mv-onb-pending-submitted-body: "Our team will review your application within 24-48 hours."
      No SLA stated in api-inapp-driver-onboarding-admins or any chef approval notification to chef
    recommendation: >
      (1) Verify the actual P95 review time and use it consistently. If it is not reliably 24-48 hours,
      state a conservative SLA ('within 3 business days') rather than miss a committed window.
      (2) Add the SLA to the push notification sent when the application is received:
      'Application received. Expect a decision within 24-48 hours.'
      (3) If the SLA is missed, send a proactive delay notification rather than leaving the chef in
      silence — silence after a committed SLA is the primary trigger for support escalation.
    metric_hypothesis: "chef D2/D7 retention; support ticket volume — missing an explicitly stated SLA with no communication is the primary cause of early-chef churn post-application"
    depends_on: null

  # ── P1: Driver logout confirmation uses banned term ───────────────────────

  - finding_id: BA-TX-019
    surface_id: md-trx-006
    lens: business-analyst
    severity: P1
    issue: "Driver app logout confirmation title says 'Logout' (single word) — style guide mandates 'Sign out' as the preferred term and bans 'Login'/'Logout'"
    evidence_excerpt: |
      app/(tabs)/more.tsx: Alert title "Logout"
      Alert body: "Are you sure you want to logout?"
    recommendation: >
      Change title to 'Sign out' and body to 'You'll need to sign in again to go online.'
      (The body currently says 'logout' again — double correction needed.)
      The style guide explicitly lists 'Sign out ✅ / Log out ❌' — this is a vocabulary compliance
      failure on an app-critical screen that drivers see every time they end a shift.
    metric_hypothesis: "brand coherence; driver trust — vocabulary inconsistency on high-frequency screens erodes platform quality perception"
    depends_on: null

  # ── P1: Mobile customer logout confirmation has title-case inconsistency ────

  - finding_id: BA-TX-020
    surface_id: mc-profile-logout-confirm
    lens: business-analyst
    severity: P1
    issue: "Customer app logout confirmation has title-case inconsistency: dialog title 'Log out' (two words, sentence case) vs button 'Log Out' (two words, title case) — and both use 'Log out' which is banned per style guide"
    evidence_excerpt: |
      app/(tabs)/profile.tsx: title "Log out" / button "Log Out"
      Style guide: "Sign out ✅ / Log out ❌"
    recommendation: >
      Change title to 'Sign out' and button to 'Sign out'. Both sentence case per button formatting
      rule (verb-first, sentence case). Remove the redundant 'Cancel' vs 'Log Out' inconsistency.
      Consistent destruction confirmation: Title 'Sign out?', Body 'You'll need to sign in again.',
      Buttons: 'Cancel' / 'Sign out'.
    metric_hypothesis: "brand coherence — vocabulary divergence on a high-frequency screen is disproportionately visible to users and surfaces in app store reviews"
    depends_on: null

  # ── P1: Mobile vendor rejection screen missing empathy copy and next step ───

  - finding_id: BA-TX-021
    surface_id: mv-onb-pending-rejected-body
    lens: business-analyst
    severity: P1
    issue: "Chef rejection screen shows 'Please review the feedback and resubmit your application' but gives no indication of what specifically needs fixing, no empathy acknowledgement, and no deadline for resubmission"
    evidence_excerpt: |
      app/(onboarding)/pending.tsx rejected state:
      Title: "Application Not Approved"
      Reason prefix: "Reason:"
      Body: "Please review the feedback and resubmit your application."
    recommendation: >
      Restructure the rejection screen for maximum resubmission rate:
      Title: 'Not approved this time' (softer than BA-TX-007 raw slug fix, warm not cold)
      Reason box: surfaced prominently with human-readable reason (fix BA-TX-007 first)
      Body: 'Review the notes above and update your application. Most chefs are approved on their
      second try.' (social proof of resubmission success increases attempt rate.)
      CTA button: 'Update and resubmit' (not just a back button).
      A resubmission rate of 40–60% post-rejection is achievable with empathetic copy; cold rejection
      screens drop this to 10–15%.
    metric_hypothesis: "chef resubmission rate; chef supply growth — empathetic rejection copy with clear next step is the single highest-leverage intervention for chef supply acquisition cost"
    depends_on: BA-TX-007

  # ── P1: Checkout payment toasts use banned exclamation pattern ─────────────

  - finding_id: BA-TX-022
    surface_id: web-tx-checkout-payment-toasts
    lens: business-analyst
    severity: P1
    issue: "'Payment successful!' toast uses an exclamation mark — style guide Rule 1 limits exclamations to genuine celebration (≤1 per page) but payment success at checkout is that moment — the actual violation is the inconsistency with 7 other toasts on the same surface that are flat"
    evidence_excerpt: |
      apps/web/src/features/customer/pages/CheckoutPage.tsx:167-280:
      "Payment successful!" (exclamation)
      vs "Failed to initiate payment" / "Payment gateway is loading" (no exclamation)
    recommendation: >
      Payment success is legitimately the celebration moment — 'Payment successful!' can keep the
      exclamation. The issue is that the other toasts are clinical by contrast. Improve the failure
      toasts to be actionable: 'Payment failed — try a different card or UPI.' rather than
      'Failed to initiate payment' (jargon). 'Payment verification failed' → 'We couldn't confirm
      your payment. Your money has not been charged. Try again or contact support.'
      The verification-failed message is particularly high-stakes: if money has been debited but
      verification failed, saying nothing about charge status causes immediate escalation.
    metric_hypothesis: "checkout completion rate; support ticket volume — 'payment verification failed' with no charge-status clarity causes customers to panic and file duplicate tickets"
    depends_on: null

  # ── P1: Catering submission toast uses exclamation in violation of style guide ──

  - finding_id: BA-TX-023
    surface_id: web-tx-catering-request-submit
    lens: business-analyst
    severity: P1
    issue: "'Catering request submitted successfully!' toast fires an exclamation on a form submission — this is not a genuine celebration moment per style guide Rule 1"
    evidence_excerpt: |
      apps/web/src/features/catering/pages/CateringRequestPage.tsx:109-113:
      "Catering request submitted successfully!"
    recommendation: >
      Change to: 'Catering request submitted. Chefs will respond within 24 hours.'
      The second sentence sets expectations (currently missing per BA-016 in marketing findings)
      and turns a generic toast into an actionable next-step indicator.
      Per style guide success toast format: past tense, ≤6 words, period. Expand slightly here
      given the high monetary stakes of a catering booking.
    metric_hypothesis: "catering quote follow-through rate — attaching an expectation to the success toast reduces customer anxiety about whether their request was seen"
    depends_on: null

  # ── P1: Mobile vendor application submission uses exclamation ─────────────

  - finding_id: BA-TX-024
    surface_id: mv-onb-pending-submitted-title
    lens: business-analyst
    severity: P1
    issue: "Mobile vendor pending screen title 'Application Submitted!' uses exclamation — this is a waiting state, not a celebration; the exclamation creates false positivity in an anxious moment"
    evidence_excerpt: |
      app/(onboarding)/pending.tsx: "Application Submitted!" (display serif title)
    recommendation: >
      Change to 'Application received.' (no exclamation). The body already says 'within 24-48 hours'
      which is the relevant information. A submission-received state is relief, not celebration.
      The exclamation raises expectations that are immediately followed by a wait — this creates
      an emotional mismatch that manifests as impatience and support tickets during the review window.
    metric_hypothesis: "chef support ticket rate during review window — mismatched positive punctuation on a waiting screen increases anxiety and drives premature status-check contacts"
    depends_on: null

  # ── P1: Driver app subscription settings bounces to web with no deep link ───

  - finding_id: BA-TX-025
    surface_id: md-trx-008
    lens: business-analyst
    severity: P1
    issue: "Driver app subscription management alerts user 'Visit the web portal to manage your subscription' with no URL, no deep link, and no specific page reference — a dead end for a monetisation-critical flow"
    evidence_excerpt: |
      app/driver-settings.tsx: Alert title "Subscription" Body "Visit the web portal to manage your subscription."
    recommendation: >
      Either (a) implement subscription management natively in the driver app (preferred for a
      monetisation-critical feature), or (b) provide a deep link to the specific subscription
      management page: 'Manage your plan at {url}. Tap to open.' with a direct URL action.
      A modal that says 'go somewhere else with no link' has a near-100% abandonment rate.
      If a driver cannot manage their subscription on mobile, they will not subscribe or renew.
    metric_hypothesis: "driver subscription conversion; driver subscription retention — inability to manage subscription in-app reduces conversion and renewal from mobile-primary drivers"
    depends_on: null

  # ── P2: Account deletion is email-only, hardcoded contact address ──────────

  - finding_id: BA-TX-026
    surface_id: md-trx-010
    lens: business-analyst
    severity: P2
    issue: "Account deletion instructs drivers to email support@homechef.in — this is a hardcoded email address that may change and is not a self-service flow, creating friction at a legally-sensitive moment"
    evidence_excerpt: |
      app/driver-settings.tsx: "Contact support at support@homechef.in to request account deletion."
    recommendation: >
      (1) Replace hardcoded email with a configurable constant or environment variable.
      (2) Add a support ticket creation path in-app rather than requiring email composition.
      (3) Under India's DPDP Act 2023, account deletion is a user right that must be clearly
      accessible — an email-only path may not satisfy the 'easy to exercise' standard.
      Short-term: change the alert to a link that opens the in-app support ticket form
      pre-filled with 'Account deletion request'.
    metric_hypothesis: "regulatory compliance; driver trust — inaccessible deletion path violates DPDP and creates legal exposure; in-app path reduces support overhead"
    depends_on: null

  # ── P2: Driver validation alert uses generic 'Validation' title ──────────

  - finding_id: BA-TX-027
    surface_id: md-trx-011
    lens: business-analyst
    severity: P2
    issue: "Driver profile validation failure alert uses 'Validation' as the title — a technical term that a driver reading this on the road will not understand"
    evidence_excerpt: |
      app/driver-profile.tsx: Alert title "Validation" Body "Name, phone, and city are required."
    recommendation: >
      Change title to 'Missing information' and body to 'Please add your name, phone number, and city.'
      Per style guide error format: 'What happened → What to do.'
      'Validation' is internal jargon; 'Missing information' is immediately actionable.
    metric_hypothesis: "driver onboarding completion rate — jargon in error states creates confusion that increases abandonment at the profile completion step"
    depends_on: null

  # ── P2: Driver profile success alert uses generic 'Success' title ─────────

  - finding_id: BA-TX-028
    surface_id: md-trx-013
    lens: business-analyst
    severity: P2
    issue: "Driver profile save success alert title is 'Success' — generic non-contextual title on a transactional success state"
    evidence_excerpt: |
      app/driver-profile.tsx: Alert title "Success" Body "Profile updated successfully."
    recommendation: >
      Remove the Alert entirely and replace with an in-line toast using the style guide format:
      'Profile saved.' (past tense, ≤6 words, period). A modal alert for a profile save is
      disruptive on a driver app (driver tone: glanceable, telegraphic). Per driver persona:
      ≤4 words where possible. Snackbar/toast is more appropriate than blocking alert.
    metric_hypothesis: "driver UX friction — blocking alerts on routine saves interrupt workflow; snackbar reduces friction at high-frequency touchpoints"
    depends_on: null

  # ── P2: Chef quote success toast uses exclamation ────────────────────────

  - finding_id: BA-TX-029
    surface_id: web-tx-catering-quotes
    lens: business-analyst
    severity: P2
    issue: "'Quote accepted! The chef has been notified.' toast uses an exclamation — for a customer accepting a catering quote worth potentially thousands of rupees, this is an appropriate celebration — but 'The chef has been notified' is redundant filler"
    evidence_excerpt: |
      apps/web/src/features/catering/pages/CateringQuotesPage.tsx:56-69:
      "Quote accepted! The chef has been notified."
    recommendation: >
      'Quote accepted. The chef will contact you to confirm event details.' The exclamation can stay
      on the first sentence. The second sentence should set the next concrete step rather than confirm
      a background notification the customer cannot verify. Removing 'The chef has been notified'
      removes a statement the user cannot verify; replacing with the next user-facing action
      ('will contact you') gives them something to expect.
    metric_hypothesis: "catering booking completion rate — setting the next expected action after quote acceptance reduces customer drop-off between acceptance and event confirmation"
    depends_on: null

  # ── P2: Cross-chef cart conflict message is blunt, no recovery path ────────

  - finding_id: BA-TX-030
    surface_id: web-tx-chefdetail-add-to-cart
    lens: business-analyst
    severity: P2
    issue: "'Your cart has items from another chef. Clear cart first.' is a blocking dead end — it tells the user what they must do but offers no in-context path to do it"
    evidence_excerpt: |
      apps/web/src/features/customer/pages/ChefDetailPage.tsx:365-369:
      "Your cart has items from another chef. Clear cart first."
    recommendation: >
      Convert the toast to an action-confirmation dialog: 'Replace cart?
      You have {itemCount} item(s) from {existingChefName}. Adding this item will clear those.
      [Keep current cart] [Replace with new item]'
      This matches mc-menuitem-cross-chef which already has the dialog approach on mobile.
      The web version should match for consistency. 'Clear cart first' as a toast with no action
      is a conversion dead end — the user must navigate to the cart, clear it, then navigate back.
    metric_hypothesis: "add-to-cart conversion; cross-chef browse rate — in-context cart replacement reduces friction for users exploring multiple chefs"
    depends_on: null

  - finding_id: BA-TX-031
    surface_id: mc-menuitem-cross-chef
    lens: business-analyst
    severity: P2
    issue: "Mobile cross-chef cart dialog says 'Replace Cart?' with 'Replace' as the destructive CTA — the word 'Replace' does not specify what is being replaced or what will survive"
    evidence_excerpt: |
      components/chef/MenuItemCard.tsx:
      Title "Replace Cart?" Body "You have items from another chef. Replace cart?"
      Buttons: "Cancel" / "Replace"
    recommendation: >
      Change to: Title 'Start new cart?', Body 'You have {count} item(s) from {ChefName}.
      Starting a new cart will remove them.', Buttons: 'Keep current cart' / 'Start new cart'.
      The destructive action button should describe what survives (the new item), not what is lost.
      This pattern reduces regret-driven support tickets for customers who tapped 'Replace' without
      understanding they were losing their existing cart.
    metric_hypothesis: "add-to-cart completion; post-order satisfaction — clearer destructive action language reduces regret-driven contacts and potential chargebacks"
    depends_on: null

  # ── P2: Delivery accepted toast uses exclamation ──────────────────────────

  - finding_id: BA-TX-032
    surface_id: dp-tx-toast-accept-success
    lens: business-analyst
    severity: P2
    issue: "'Delivery accepted!' delivery portal toast fires exclamation on an operational action — style guide restricts exclamations to genuine celebration; accepting a delivery job is routine"
    evidence_excerpt: |
      apps/delivery-portal/src/features/deliveries/pages/AvailableDeliveriesPage.tsx:25: "Delivery accepted!"
      Also web-tx-delivery-actions: "Delivery accepted!"
    recommendation: >
      Change to 'Delivery accepted.' (period). Per driver/delivery-portal tone: 'Functional,
      time-aware. Crisp 5-12 words.' The exclamation signals false positivity in an operational
      context and conflicts with the driver persona tone matrix.
    metric_hypothesis: "brand coherence; driver UX — operational toasts with exclamations create noise that drivers learn to ignore, reducing attention to genuinely important alerts"
    depends_on: null

  # ── P2: Driver onboarding step toasts are verbose for a mobile glanceable context ─

  - finding_id: BA-TX-033
    surface_id: dp-tx-toast-step5-submitted
    lens: business-analyst
    severity: P2
    issue: "Driver onboarding submission toast 'Application submitted successfully!' uses adverb-padded language and exclamation on what is the start of a waiting period, not a celebration"
    evidence_excerpt: |
      apps/delivery-portal/src/features/onboarding/components/StepReview.tsx:89: "Application submitted successfully!"
    recommendation: >
      'Application submitted. Expect a decision within 24-48 hours.' (per BA-TX-018 SLA alignment).
      Remove 'successfully' (per style guide: past tense, ≤6 words) and add the SLA expectation
      to set correct waiting expectations. This is the driver equivalent of BA-TX-024.
    metric_hypothesis: "driver review-period support tickets — no SLA on the submission toast drives premature status-check contacts"
    depends_on: BA-TX-018

  # ── P2: Vendor portal policies banner creates false menu-readiness expectation ─

  - finding_id: BA-TX-034
    surface_id: vp-onb-policies-ready-banner
    lens: business-analyst
    severity: P2
    issue: "'You can start setting up your menu in the meantime' banner on the application-submitted screen creates premature menu-building behaviour that may be reset if the application is rejected"
    evidence_excerpt: |
      features/onboarding/components/StepPolicies.tsx:
      "You're all set to submit your application! Our team will review your details and you'll be notified within 24-48 hours. You can start setting up your menu in the meantime."
    recommendation: >
      Add a caveat: 'You can start drafting your menu — it will go live once your application is
      approved.' Without this, chefs who build menus during review and are then rejected will have
      invested time they perceive as wasted, amplifying negative sentiment toward the platform.
      The copy currently implies menu-building is fully productive before approval.
    metric_hypothesis: "chef rejection-to-churn rate — chefs who invest in menu setup and are then rejected feel deceived; the caveat manages expectations and reduces churned-applicant complaints"
    depends_on: null

  # ── P2: Generic document upload toasts use raw label variable ────────────

  - finding_id: BA-TX-035
    surface_id: vp-ux-profile-doc-toasts
    lens: business-analyst
    severity: P2
    issue: "Document upload toast '{label} uploaded' passes through raw template variable — if label is undefined or carries an internal value, the chef sees a blank or jargon toast"
    evidence_excerpt: |
      features/profile/pages/ProfilePage.tsx: "{label} uploaded" / "Upload failed"
      Label comes from a loop — if document type label is missing, toast reads 'undefined uploaded'
    recommendation: >
      Add a fallback: `${label ?? 'Document'} uploaded.`
      Also: 'Upload failed' is a dead-end error message per style guide (What happened → What to do).
      Change to: 'Upload failed. Check your connection and try again.' or 'Upload failed. File must
      be JPEG, PNG, or PDF under 5 MB.'
    metric_hypothesis: "chef onboarding completion rate — dead-end error messages on document upload (a required onboarding step) cause abandonment"
    depends_on: null

  - finding_id: BA-TX-036
    surface_id: dp-tx-toast-step3-uploaded
    lens: business-analyst
    severity: P2
    issue: "Driver document upload toast '{label} uploaded' has the same raw variable interpolation risk as BA-TX-035"
    evidence_excerpt: |
      apps/delivery-portal/src/features/onboarding/components/StepDocuments.tsx:142: "{label} uploaded"
    recommendation: >
      Same fix as BA-TX-035: add fallback `${label ?? 'Document'} uploaded.`
      For the driver portal, the driver persona requires even shorter, clearer feedback:
      'Uploaded.' or '[Document name] uploaded.' Consider inline progress indicator over toast
      for mobile-first driver context.
    metric_hypothesis: "driver onboarding completion rate — document upload errors are the #1 abandonment point in delivery driver onboarding (industry benchmark)"
    depends_on: BA-TX-035

  # ── P2: Status updated toast is content-free ─────────────────────────────

  - finding_id: BA-TX-037
    surface_id: dp-tx-toast-status-updated
    lens: business-analyst
    severity: P2
    issue: "'Status updated' driver delivery toast is the emptiest possible confirmation — no indication of what status was set or what action is expected next"
    evidence_excerpt: |
      apps/delivery-portal/src/features/deliveries/pages/ActiveDeliveryPage.tsx:23: "Status updated"
    recommendation: >
      Inject the new status value: 'Status: Picked up.' / 'Status: Delivered.' etc.
      Per driver persona: imperative, telegraphic. 'Status updated' tells the driver nothing they
      don't already know (they just updated it). The meaningful information is the new state that
      will be communicated to the customer. Knowing the new state helps the driver verify the right
      button was tapped in a time-pressured context.
    metric_hypothesis: "driver error rate; customer status accuracy — drivers who can't confirm status transitions may tap the wrong button; explicit state confirmation reduces status errors"
    depends_on: null

  # ── P2: API success message inconsistency — "successfully" suffix ──────────

  - finding_id: BA-TX-038
    surface_id: api-success-message-generic
    lens: business-analyst
    severity: P2
    issue: "43 API success message strings have inconsistent 'successfully' suffix — 'Password changed successfully' vs 'Logged out successfully' vs 'Preference saved' vs 'Chef verified' — if these surface in UI toasts, the inconsistency is user-visible"
    evidence_excerpt: |
      handlers/*: "Password changed successfully" / "Logged out successfully" / "Preference saved" (no adverb) / "Chef verified" (no adverb) / "Email verified successfully" — mixed pattern
    recommendation: >
      Standardise: either always include 'successfully' or never include it.
      Style guide success toast format is past tense with period and no adverb: 'Password changed.'
      not 'Password changed successfully.' The adverb is implied by the success state.
      Audit which of these 43 strings surface in frontend toasts (vs staying server-side) and apply
      the style guide format to user-visible ones. Create a centralised message constant file.
    metric_hypothesis: "brand coherence — inconsistent toast copy is noticed in consecutive interactions and signals lack of editorial review"
    depends_on: null

  # ── P2: Delivery-completed toast has exclamation and motivational padding ───

  - finding_id: BA-TX-039
    surface_id: web-tx-delivery-actions
    lens: business-analyst
    severity: P2
    issue: "'Delivery completed! Great job!' in the delivery portal adds motivational copy that is inconsistent with the functional driver/operator tone in the style guide"
    evidence_excerpt: |
      apps/web/src/features/delivery/pages/DashboardPage.tsx:76-96: "Delivery completed! Great job!"
    recommendation: >
      Change to 'Delivery completed.' (period). The driver persona tone is 'Functional, time-aware.
      Crisp.' 'Great job!' is consumer-app praise language that reads as condescending in a
      professional driver tool. Operators and drivers who see 'Great job!' after their fifth delivery
      that session will find it hollow.
    metric_hypothesis: "driver trust; brand coherence — patronising motivational copy in a professional tool reduces driver confidence in the platform's understanding of their role"
    depends_on: null

  # ── P2: Push notification title/body length not validated ─────────────────

  - finding_id: BA-TX-040
    surface_id: api-inapp-chef-responded
    lens: business-analyst
    severity: P2
    issue: "Chef-response-to-admin notification title is 'Chef Responded: ' + approval.Title (chef-authored) with no length cap — a chef could submit a 500-char approval title that creates a truncated or broken notification"
    evidence_excerpt: |
      handlers/approval.go::Notification: Title: "Chef Responded: " + approval.Title
      approval.Title is chef-authored; no length validation visible
    recommendation: >
      Add a title truncation: `fmt.Sprintf("Chef responded: %.60s", approval.Title)` (60 chars covers
      most push notification title limits). Add message length validation at the approval creation
      endpoint: max 500 chars for approval titles, max 2000 for admin notes. Also: no profanity
      filter visible — a bad-actor chef can push profanity to all admins via approval titles.
    metric_hypothesis: "admin trust; abuse risk — unvalidated chef-authored strings in push notifications can surface inappropriate content to all admins simultaneously"
    depends_on: null

  # ── P2: Push in-app/push title inconsistency on delivery picked up ────────

  - finding_id: BA-TX-041
    surface_id: api-push-delivery-pickedup-customer
    lens: business-analyst
    severity: P2
    issue: "Push notification for delivery picked up says 'Order On The Way!' but the in-app notification says 'Order Picked Up' — two different status frames for the same event shown in the same session"
    evidence_excerpt: |
      services/notifications.go::handleDeliveryPickedUp push: Title "Order On The Way!"
      services/notifications.go::handleDeliveryPickedUp in-app: Title "Order Picked Up"
    recommendation: >
      Align to the canonical status label from BA-TX-009. Canonical: 'Your delivery partner has your
      order' (both push and in-app). Push title: 'On the way' (telegraphic, no exclamation).
      In-app title: 'On the way'. Body: 'Your order has been picked up and is heading to you.'
    metric_hypothesis: "order tracking clarity — push/in-app title divergence at the same event causes customers to check the app to reconcile two different status descriptions"
    depends_on: BA-TX-009

  # ── P2: Catering mobile success alert misses next-step context ────────────

  - finding_id: BA-TX-042
    surface_id: mc-catering-submit-success
    lens: business-analyst
    severity: P2
    issue: "'Request Submitted! / Chefs will review and send quotes.' mobile catering success alert uses exclamation and vague timing — 'will' without a window is an open-ended commitment"
    evidence_excerpt: |
      app/catering.tsx: Alert title "Request Submitted!" body "Chefs will review and send quotes."
    recommendation: >
      Title: 'Request submitted.' (no exclamation), Body: 'Chefs typically respond within 2-4 hours.
      You'll get a notification for each quote.' This matches the intent of BA-TX-023 (web catering)
      and adds the timing expectation that reduces anxiety post-submission.
    metric_hypothesis: "catering quote follow-through rate — timing expectation reduces 'did my request go through' contacts"
    depends_on: BA-TX-023

  # ── P2: Vendor onboarding document upload success is under-confirmed ────────

  - finding_id: BA-TX-043
    surface_id: mv-onb-docs-upload-success
    lens: business-analyst
    severity: P2
    issue: "'Uploaded successfully' under document preview on mobile vendor onboarding is the only confirmation a chef gets — no summary of which documents are still required, no overall completion progress"
    evidence_excerpt: |
      app/(onboarding)/documents.tsx: "Uploaded successfully" shown under preview
    recommendation: >
      Add an overall progress indicator: 'Government ID uploaded. 1 of 3 documents complete.'
      This turns an isolated success toast into an orientation signal that shows progress toward
      completion. Document upload abandonment in onboarding flows drops significantly when users
      know how many steps remain (standard UX benchmark: ~20% reduction in abandonment).
    metric_hypothesis: "chef onboarding completion rate — progress visibility at each document upload reduces abandonment at the most friction-heavy onboarding step"
    depends_on: null

  # ── P2: Order status push/in-app title inconsistency ─────────────────────

  - finding_id: BA-TX-044
    surface_id: api-inapp-order-status-customer
    lens: business-analyst
    severity: P2
    issue: "In-app order status notification title is 'Order Status Updated' while the push notification title is 'Order Update' — customers who receive both in quick succession see inconsistent status references"
    evidence_excerpt: |
      services/notifications.go::handleOrderUpdated in-app: Title "Order Status Updated"
      services/notifications.go::handleOrderUpdated push: Title "Order Update"
    recommendation: >
      Align both to 'Order update' (sentence case). Refer to BA-TX-009 canonical map.
      The word 'Status' in the in-app title is redundant — 'Updated' already implies status change.
    metric_hypothesis: "notification clarity; support ticket volume — title drift between push and in-app for the same event trains customers to distrust notification accuracy"
    depends_on: BA-TX-009

  # ── P2: Approval title has exclamation inconsistency between push and in-app ─

  - finding_id: BA-TX-045
    surface_id: api-push-approval-approved-chef
    lens: business-analyst
    severity: P2
    issue: "Approval-approved push title says 'Request Approved!' (with exclamation) but in-app notification title says 'Request Approved' (no exclamation) — inconsistency on a high-stakes chef milestone"
    evidence_excerpt: |
      services/notifications.go::handleApprovalApproved push: Title "Request Approved!"
      services/notifications.go::handleApprovalApproved in-app: Title "Request Approved"
    recommendation: >
      Approval of a menu item or profile update is a genuine positive moment — the exclamation is
      appropriate here. Add it consistently to both: 'Approved!' (push, very short) and
      'Request approved' (in-app body, no exclamation per style guide limit of ≤1 per page).
      Per style guide: limit exclamations to genuine celebration; a push title can carry the
      exclamation as the headline moment. In-app should be calm.
    metric_hypothesis: "chef satisfaction at approval milestone — consistent positive confirmation reinforces the effort chefs put into submissions"
    depends_on: BA-TX-007

  # ── P3: Chef profile saved toasts are generic across apps ─────────────────

  - finding_id: BA-TX-046
    surface_id: web-tx-chef-profile
    lens: business-analyst
    severity: P3
    issue: "'Profile updated successfully' toast is the exact same string used by both chef and customer profiles across web and mobile — misses an opportunity to provide context-specific feedback"
    evidence_excerpt: |
      web-tx-chef-profile: "Profile updated successfully"
      web-tx-profile-update: "Profile updated successfully"
      mv-profile-update-success-body: "Profile updated successfully."
      md-trx-014: "Profile updated successfully."
    recommendation: >
      Not a critical change but a quick win: differentiate on high-intent saves.
      Chef profile: 'Profile saved. Customers can see your updates.' — adds value.
      Customer profile: 'Profile saved.' (no need for context, shorter is better).
      The driver profile (md-trx-014) is the highest-priority per persona: 'Saved.' (telegraphic).
    metric_hypothesis: "chef profile completion rate — contextualising saves for chefs ('customers can see your updates') reinforces the value of profile completeness"
    depends_on: null

  # ── P3: Order detail payment confirmation toast is technically ambiguous ────

  - finding_id: BA-TX-047
    surface_id: web-tx-orderdetail-toasts
    lens: business-analyst
    severity: P3
    issue: "'Payment confirmed' toast on OrderDetailPage could refer to the customer confirming payment status manually or to the system confirming payment receipt — ambiguous in a high-money-anxiety context"
    evidence_excerpt: |
      apps/web/src/features/customer/pages/OrderDetailPage.tsx:61-108: "Payment confirmed"
    recommendation: >
      Change to 'Payment verified.' (system action confirmed) or 'We've confirmed your payment.'
      (warmer, clearer actor). The customer is confirming the payment happened — this is the system
      responding that it agrees. 'Payment confirmed' could mean the customer is the one confirming,
      creating ambiguity when there is a payment dispute.
    metric_hypothesis: "payment dispute rate — ambiguous payment confirmation language increases contested chargeback attempts when customers are uncertain about payment status"
    depends_on: null

  # ── P3: 2FA enabled/disabled toasts are cold for a security trust moment ───

  - finding_id: BA-TX-048
    surface_id: web-tx-profile-2fa
    lens: business-analyst
    severity: P3
    issue: "'Two-factor authentication enabled' toast confirms a security upgrade with no follow-up context — no mention of what changed, no backup-codes reminder, no test suggestion"
    evidence_excerpt: |
      apps/web/src/features/customer/pages/ProfilePage.tsx:1200-1284: "Two-factor authentication enabled"
    recommendation: >
      Change to: 'Two-step sign-in enabled. Save your backup codes — you'll need them if you lose
      your phone.' The backup codes are generated at 2FA setup and the toast is the moment to remind
      the user to store them. Users who enable 2FA and are not reminded to save backup codes are the
      #1 source of 'I'm locked out' support tickets.
    metric_hypothesis: "2FA-lockout support ticket volume — backup codes reminder at enable-time reduces lockout contacts by ~40% (auth support benchmark)"
    depends_on: null

  # ── P3: Staff invitation success title uses exclamation on admin surface ────

  - finding_id: BA-TX-049
    surface_id: ap-staff-invite-success-title
    lens: business-analyst
    severity: P3
    issue: "'Invitation created successfully!' uses exclamation on an admin operational surface — style guide Rule 1 restricts exclamations to genuine celebration; admin staff invitations are routine operations"
    evidence_excerpt: |
      apps/admin-portal/src/features/staff/pages/StaffPage.tsx: "Invitation created successfully!"
    recommendation: >
      Change to 'Invitation sent.' Per admin persona tone: 'Neutral operator. Precise. No fluff.'
      The body 'Share this link with the invitee to complete their registration.' is well-written
      and functional — keep it.
    metric_hypothesis: "brand coherence — exclamations on admin operational surfaces signal inconsistent editorial review and reduce admin trust in the tool"
    depends_on: null

  # ── P3: Vendor settings payout 'Payout details saved' with no status feedback ─

  - finding_id: BA-TX-050
    surface_id: vp-ux-settings-payout-toasts
    lens: business-analyst
    severity: P3
    issue: "'Payout details saved' toast confirms bank detail storage but gives no indication of when payouts will be activated or whether the details have been verified — creates false expectation that payouts will start imminently"
    evidence_excerpt: |
      features/settings/pages/SettingsPage.tsx: "Payout details saved"
      Cross-reference: BA-008 from marketing findings — payout integration is 'coming soon'
    recommendation: >
      If payouts are not yet active, the toast must reflect this: 'Bank details saved. Payouts will
      begin once the integration goes live — we'll notify you.' If payouts are active, add
      verification status: 'Bank details saved. First payout scheduled for {next_payout_date}.'
      This connects directly to BA-008 which flags the 'coming soon' payout deception.
    metric_hypothesis: "chef D30 retention; regulatory risk — chefs who save bank details and expect payouts that never arrive will churn and may report deceptive practices"
    depends_on: null

  # ── P3: Vendor portal Stripe onboarding error messages are developer-facing ─

  - finding_id: BA-TX-051
    surface_id: vp-ux-settings-stripe-toasts
    lens: business-analyst
    severity: P3
    issue: "'Failed to start Stripe onboarding' and 'Failed to resume onboarding' are developer-facing error labels surfaced to chefs — a chef does not know what 'Stripe onboarding' means"
    evidence_excerpt: |
      features/settings/pages/SettingsPage.tsx: "Failed to start Stripe onboarding" / "Failed to resume onboarding"
    recommendation: >
      Change to customer-facing language: 'Couldn't start payment setup — try again or contact
      support.' and 'Couldn't resume payment setup. Your progress is saved — try again.'
      'Stripe onboarding' is a technical term from the Stripe API; chefs should hear 'payment setup'
      or 'payment account'. This is especially important post-payment-provider-switch from Razorpay.
    metric_hypothesis: "chef payment setup completion rate — incomprehensible error messages on the payment setup flow cause abandonment of a monetisation-critical step"
    depends_on: null

  # ── P3: Admin approval feedback toasts are terse but legally significant ───

  - finding_id: BA-TX-052
    surface_id: ap-approvaldetail-toasts
    lens: business-analyst
    severity: P3
    issue: "'Approval request approved' and 'Approval request rejected' are passive-tense toasts that confirm admin actions — grammatically they read as if someone else approved/rejected the request the admin is looking at"
    evidence_excerpt: |
      apps/admin-portal/src/features/approvals/pages/ApprovalDetailPage.tsx:
      "Approval request approved" / "Approval request rejected" / "More information requested from chef"
    recommendation: >
      Change to active voice per style guide: 'Request approved.' / 'Request rejected.' /
      'Requested more info from chef.' Per admin persona: 'Direct. Approve, Suspend, Audit.'
      Active voice removes ambiguity about who performed the action in an audit context.
    metric_hypothesis: "admin operational clarity — passive-voice action confirmations create ambiguity in audit trails and reduce admin confidence in system state"
    depends_on: null

  # ── P3: Admin security settings session-revocation toast is technically incomplete ─

  - finding_id: BA-TX-053
    surface_id: ap-secsettings-toasts
    lens: business-analyst
    severity: P3
    issue: "'All other sessions revoked. You may need to sign in again.' uses hedged language 'may need' for a definitive security action — all other sessions ARE revoked, sign-in IS required"
    evidence_excerpt: |
      apps/admin-portal/src/features/settings/pages/SecuritySettingsPage.tsx:
      "All other sessions revoked. You may need to sign in again."
    recommendation: >
      Change to: 'All other sessions ended. Other devices will need to sign in again.'
      'You may need to sign in again' is confusing when the action is 'revoke all OTHER sessions'
      — the current user is not affected. Clarify that it is other devices that need to sign in.
    metric_hypothesis: "admin trust; support ticket volume — 'you may need to sign in again' after a security action causes admins to panic and file tickets expecting to lose their own session"
    depends_on: null

  # ── P3: Chef review reply success uses title-cased alert title ────────────

  - finding_id: BA-TX-054
    surface_id: mv-reviewdetail-success-title
    lens: business-analyst
    severity: P3
    issue: "Mobile vendor review reply success alert title 'Reply Sent' uses title case — style guide button/label convention is sentence case; inconsistent with other success states in the same app"
    evidence_excerpt: |
      app/review/[reviewId].tsx: Alert title "Reply Sent"
    recommendation: >
      Change to 'Reply sent.' (sentence case, period). Replace blocking Alert with a non-blocking
      toast for better mobile UX: 'Reply posted.' (past tense, ≤6 words, period per style guide).
    metric_hypothesis: "brand coherence — title-case inconsistency across alerts and toasts signals copy was written by multiple authors without a shared review"
    depends_on: null

  # ── P3: Driver staff permission error is misplaced in the driver app ────────

  - finding_id: BA-TX-055
    surface_id: md-trx-018
    lens: business-analyst
    severity: P3
    issue: "'You do not have permission to invite staff.' surfaces in the driver app — this is an admin function that should not be reachable by drivers in the first place"
    evidence_excerpt: |
      app/staff.tsx (mobile-delivery): "You do not have permission to invite staff."
    recommendation: >
      The staff invitation UI should not render for users without invite permission.
      The permission error should be caught at the route/component level and hide the UI,
      not display an error on interaction. If the screen must remain reachable, change the copy to:
      'Staff invitations are managed by your fleet administrator.'
      The current message implies the driver attempted something they should not — better UX is
      to never show the action.
    metric_hypothesis: "driver trust; admin security — showing permission-denied errors for hidden features signals incomplete role-based access control implementation"
    depends_on: null

  # ── P3: Mobile customer payment confirming screen has no next-step guidance ─

  - finding_id: BA-TX-056
    surface_id: mc-payment-confirming
    lens: business-analyst
    severity: P3
    issue: "'Confirming payment...' is the entire content of the Razorpay deep-link callback screen — no estimated wait time, no fallback instruction if it hangs"
    evidence_excerpt: |
      app/payment/result.tsx: "Confirming payment..."
    recommendation: >
      Add a timeout handler and fallback copy: 'Confirming payment — this takes a few seconds.'
      After 10 seconds without response: 'Taking longer than expected. If your payment went through,
      you'll receive a confirmation email. Otherwise, nothing was charged.'
      The 'payment confirming' screen is the highest-anxiety moment in the customer journey;
      leaving it as a bare spinner with no guidance on what to do if it freezes causes abandonment
      and duplicate payment attempts.
    metric_hypothesis: "payment completion rate; duplicate payment rate — bare payment-confirming screen with no timeout guidance drives customers to retry payments that were already charged"
    depends_on: null

  # ── P3: Vendor onboarding personal info save toast duplicates subject ────────

  - finding_id: BA-TX-057
    surface_id: vp-onb-personal-avatar-success
    lens: business-analyst
    severity: P3
    issue: "'Profile photo uploaded' vendor onboarding toast is correctly formatted (past tense, no exclamation, period implied) — no copy issue but the inconsistency with dp-tx-toast-step3-uploaded ('{label} uploaded') creates a divergent pattern across onboarding portals"
    evidence_excerpt: |
      features/onboarding/components/StepPersonalInfo.tsx: "Profile photo uploaded"
      vs dp-tx-toast-step3-uploaded: "{label} uploaded" (parametric, no "successfully")
    recommendation: >
      This finding is primarily a consistency flag. The vendor portal version ('Profile photo
      uploaded') is better — adopt it as the canonical pattern for all document/media upload toasts
      across onboarding portals: '{Human-readable document type} uploaded.' No 'successfully'.
      No exclamation. Aligns with style guide success toast format.
    metric_hypothesis: "brand coherence — inconsistent upload confirmation copy across three onboarding portals signals fragmented development"
    depends_on: BA-TX-035
```

## Brand Voice findings

```yaml
# Brand-Voice lens findings — TRANSACTIONAL category
# Lens: cross-surface and cross-app voice consistency
# Source brief: docs/content-audit/lens-briefs/brand-voice.md
# Style guide: docs/content-audit/STYLE-GUIDE.md
# Brand-personality reference: .impeccable.md ("Confident · Appetizing · Quietly modern")
# Inventory rows audited: 154 (transactional)
#
# Headline drift patterns surfaced:
#   1. Brand-identity triple-name leak: "Fe3dr" / "Fe3dr HomeChef" / "HomeChef" co-exist in user-facing strings
#   2. Order-status enum drift: 3+ status-label maps (email subjects, in-app messages, push bodies) — same event, different words
#   3. Subject/in-app/push wording mismatch per event (chef, customer, driver all affected)
#   4. Exclamation-mark budget blown across vendor/driver/admin surfaces (style-guide budget = 0 for these personas)
#   5. Emoji leaking into chef/driver/admin emails ("🔔", "🚗", "✅", "❌", "🎉", "👨‍🍳", "📦", "🍽️") — style-guide allows customer-only
#   6. Title Case vs sentence case inconsistency in alert titles ("Log out" header / "Log Out" button on same screen)
#   7. Raw enum slug leaks ("Your menu_item has been approved", "%s" interpolated with snake_case)
#   8. Unbranded inline-HTML emails (support, account-reminder) bypass emailBase wrapper — breaks brand
#   9. "successfully" suffix toggles on/off across siblings ("Profile updated" vs "Profile updated successfully")
#  10. "Log out" vs "Logout" vs "Sign out" — style-guide requires "Sign out"; all three exist
#  11. Same NATS event fires duplicate pushes with different copy (two different chef "new order" pushes)
#  12. Generic alert titles "Success" / "Validation" — non-brand, sounds like default RN Alert
#  13. CTA tone drift: "Start Exploring" (welcome email) vs sentence-case button rules

findings:

  - finding_id: BV-001
    surface_id: api-push-user-welcome
    lens: brand-voice
    severity: P0
    issue: "Brand identity contradiction: push says 'HomeChef', welcome email says 'Fe3dr', account-reminder email says 'Fe3dr HomeChef' — new user receives three different brand names in their first hour"
    evidence_excerpt: "Title: 'Welcome to HomeChef!' — Message: 'Thank you for joining HomeChef. Discover amazing home-cooked meals near you!'"
    related_surfaces: ["api-email-welcome", "api-email-account-reminder", "api-email-chef-verified", "md-trx-010"]
    recommendation: "Pick one canonical brand name and apply globally. If 'Fe3dr' is the legal brand, push must say 'Welcome to Fe3dr.' (sentence-period, no exclamation per Rule 1). HomeChef-as-brand strings must be removed."
    depends_on: null

  - finding_id: BV-002
    surface_id: api-email-account-reminder
    lens: brand-voice
    severity: P0
    issue: "Brand-name compound 'Fe3dr HomeChef' appears in subject line and body — neither standalone brand is consistent across the system. This is the most visible drift surface (security-related email)."
    evidence_excerpt: "Subject: 'You already have a Fe3dr HomeChef account' — body: 'Someone (possibly you) just tried to create a Fe3dr HomeChef account...'"
    related_surfaces: ["api-email-welcome", "api-push-user-welcome", "md-trx-010"]
    recommendation: "Replace 'Fe3dr HomeChef' with the single canonical brand. Pick one — same name on signup, login, this reminder, the welcome email, and the push."
    depends_on: null

  - finding_id: BV-003
    surface_id: md-trx-002
    lens: brand-voice
    severity: P0
    issue: "Driver-facing safety modal uses 'HomeChef Delivery' brand name while all chef/customer emails say 'Fe3dr'. Driver app does not know the brand."
    evidence_excerpt: "'To keep customers updated on their delivery, HomeChef Delivery needs to track your location while you are on an active delivery...'"
    related_surfaces: ["api-email-welcome", "api-push-user-welcome", "md-trx-010"]
    recommendation: "Replace 'HomeChef Delivery' with the canonical brand. Driver consent strings are legal-adjacent — brand drift here also creates an iOS App Store review risk (the Info.plist string likely uses a different name again)."
    depends_on: null

  - finding_id: BV-004
    surface_id: md-trx-010
    lens: brand-voice
    severity: P0
    issue: "Support email hardcoded as 'support@homechef.in' in driver-facing account-deletion path — domain says HomeChef India, emails say Fe3dr, push says HomeChef. Three brand identities. Driver who emails support sees yet another."
    evidence_excerpt: "'Contact support at support@homechef.in to request account deletion.'"
    related_surfaces: ["api-email-welcome", "api-push-user-welcome", "md-trx-002"]
    recommendation: "Make support email a config token tied to the canonical brand domain. Do not hardcode."
    depends_on: null

  - finding_id: BV-005
    surface_id: api-push-order-update-customer
    lens: brand-voice
    severity: P0
    issue: "Order-status enum drift: three independent label maps for the same 7 statuses (email subject map at email_templates.go, push body via getOrderStatusMessage, and the second push variant via humanReadableOrderStatus). Same status, three different sentences."
    evidence_excerpt: "Email subject: 'Your chef is preparing your meal' / Push: 'Your order is being prepared' / Push alt: 'Your order is now being prepared'"
    related_surfaces: ["api-email-order-status-preparing", "api-email-order-status-confirmed", "api-email-order-status-ready", "api-email-order-status-pickedup", "api-email-order-status-onway", "api-email-order-status-delivered", "api-email-order-status-cancelled", "api-push-order-update-customer-deeplink", "api-inapp-order-status-customer"]
    recommendation: "Single status-label map per status. One sentence per status. Email subject, in-app message, push body all pull from the same map. Suggested phrasing per status: confirmed='Your chef has confirmed the order.' / preparing='Your chef is preparing your meal.' / ready='Your order is ready for pickup.' / pickedup='Your driver has picked up the order.' / onway='Your order is on the way.' / delivered='Delivered. Enjoy your meal.' / cancelled='Your order was cancelled.'"
    depends_on: null

  - finding_id: BV-006
    surface_id: api-push-chef-new-order-actionable
    lens: brand-voice
    severity: P0
    issue: "Chef receives TWO push notifications with different copy on the same NATS event — handlers/notifications.go uses 'New Order / You have a new order waiting for your confirmation' and services/notifications.go::handleOrderCreated uses 'New Order Received / You have a new order waiting to be prepared!'. Two queue groups, two voices."
    evidence_excerpt: "Push A: 'New Order — You have a new order waiting for your confirmation' / Push B: 'New Order Received — You have a new order waiting to be prepared!'"
    related_surfaces: ["api-push-order-created-chef", "api-inapp-chef-new-order", "api-inapp-order-created-chef"]
    recommendation: "One consumer wins. Pick the single canonical chef new-order push and delete the other. Suggested phrasing: 'New order — please confirm.' (sentence-case, period, no exclamation per Rule 1 for chef tone)."
    depends_on: null

  - finding_id: BV-007
    surface_id: api-inapp-chef-new-order
    lens: brand-voice
    severity: P1
    issue: "Three distinct in-app 'new order' notification titles in the codebase: 'New Order!', 'New Order Received', 'New Order'. Same event, three voices."
    evidence_excerpt: "'New Order!' (services/notifications.go::handleChefNewOrder) vs 'New Order Received' (handleOrderCreated) vs 'New Order' (RegisterPushConsumers)"
    related_surfaces: ["api-push-order-created-chef", "api-inapp-order-created-chef", "api-push-chef-new-order-actionable"]
    recommendation: "Consolidate to one: 'New order' (sentence case, no exclamation — chef tone per Style Guide §2 has 0 exclamation budget)."
    depends_on: BV-006

  - finding_id: BV-008
    surface_id: api-email-chef-verified
    lens: brand-voice
    severity: P1
    issue: "Cross-surface noun drift for the same event: email says 'kitchen is verified', in-app says 'chef profile has been verified'. Customer/chef sees two different concepts for one approval."
    evidence_excerpt: "Email subject: 'Your kitchen is verified — Fe3dr' / In-app: 'Your chef profile has been verified.'"
    related_surfaces: ["api-inapp-chef-verified", "api-email-chef-verified-publish", "ap-chefs-toast-verified"]
    recommendation: "Pick one noun. Style Guide §3 prefers 'Chef' chef-facing — recommend 'Your chef profile is verified.' across email subject, in-app title, push, and admin-facing toast."
    depends_on: null

  - finding_id: BV-009
    surface_id: api-email-chef-verified-publish
    lens: brand-voice
    severity: P2
    issue: "NATS-published email title 'Your Chef Profile is Verified!' uses Title Case (banned outside proper nouns per Style Guide §3) and exclamation (Rule 1 / chef tone budget = 0)."
    evidence_excerpt: "Email Title: 'Your Chef Profile is Verified!' — Message: 'Congratulations! Your chef profile has been verified. You can now start accepting orders!'"
    related_surfaces: ["api-email-chef-verified", "api-inapp-chef-verified"]
    recommendation: "'Your chef profile is verified.' Drop the exclamation, sentence-case the title."
    depends_on: BV-008

  - finding_id: BV-010
    surface_id: api-inapp-chef-verified
    lens: brand-voice
    severity: P2
    issue: "Title 'Congratulations!' is loud — Style Guide Rule 1 prohibits urgency/celebration punctuation in chef-facing copy. 'Congratulations' as a body word is fine; as a title it overdoes it."
    evidence_excerpt: "Title: 'Congratulations!' — Message: 'Your chef profile has been verified. You can now start accepting orders!'"
    related_surfaces: ["api-email-chef-verified-publish", "api-email-chef-verified"]
    recommendation: "Title: 'You're verified.' Message: 'Your chef profile is live. You can start accepting orders.' Two periods, zero exclamations."
    depends_on: BV-008

  - finding_id: BV-011
    surface_id: api-email-chef-new-order
    lens: brand-voice
    severity: P1
    issue: "Emoji '🔔' and exclamation 'New Order!' in chef-facing email — style guide chef tone allows zero exclamations and zero decorative emoji. Plus 'New Order!' Title-Cases what should be sentence case."
    evidence_excerpt: "Subject: 'New order #%s — Fe3dr' — body: '🔔 New Order! You have a new order #%s worth ₹%.2f. Please review and accept the order from your dashboard.'"
    related_surfaces: ["api-push-order-created-chef", "api-inapp-chef-new-order", "api-push-chef-new-order-actionable"]
    recommendation: "Drop the bell emoji. Sentence case body heading 'New order' (no exclamation). Subject already sentence-case — keep."
    depends_on: BV-006

  - finding_id: BV-012
    surface_id: api-email-delivery-assigned
    lens: brand-voice
    severity: P1
    issue: "Emoji '🚗' and 'New Delivery!' in driver-facing email — driver tone budget = 0 emoji, 0 exclamations. Worse, this exact template is also sent to the CUSTOMER per the inventory note — customer would see 'You've been assigned order #...' which is wrong-persona copy."
    evidence_excerpt: "Subject: 'Delivery assigned — Order #%s' — body: '🚗 New Delivery! You've been assigned order #%s. Pickup location: %s'"
    related_surfaces: ["api-push-delivery-assigned-driver", "api-push-delivery-assigned-customer", "api-inapp-delivery-assigned-customer"]
    recommendation: "Drop emoji. Driver body: 'New delivery — Order #%s. Pickup: %s.' (telegraphic, sentence case, period). Fix the routing so customer gets the customer template, not this one."
    depends_on: null

  - finding_id: BV-013
    surface_id: api-email-order-status-preparing
    lens: brand-voice
    severity: P2
    issue: "Decorative emoji '👨‍🍳' in customer email — Style Guide allows occasional customer-facing emoji but not as a chrome decoration on every status email. Seven status emails carry seven different emojis; emoji becomes a system, not a moment."
    evidence_excerpt: "Subject: 'Order #%s — Your chef is preparing your meal' — body uses 👨‍🍳 emoji"
    related_surfaces: ["api-email-order-status-confirmed", "api-email-order-status-ready", "api-email-order-status-pickedup", "api-email-order-status-onway", "api-email-order-status-delivered", "api-email-order-status-cancelled"]
    recommendation: "Remove emoji from status emails. The status label and the photo of the dish (where present) do the emotional work. Reserve emoji for the delivered email only — and only one (🎉 is fine on delivery as the celebratory moment, in line with Rule 1's 'genuine celebration' carve-out)."
    depends_on: null

  - finding_id: BV-014
    surface_id: api-email-order-status-onway
    lens: brand-voice
    severity: P2
    issue: "Exclamation 'Your order is on the way!' in customer email subject — Style Guide Rule 1 allows ≤1 exclamation per page customer-facing; this single email has potentially two (subject + body). Plus tracking emoji '🚗' decorative."
    evidence_excerpt: "Subject: 'Order #%s — Your order is on the way!' — 🚗 emoji"
    related_surfaces: ["api-push-order-update-customer", "api-push-delivery-pickedup-customer"]
    recommendation: "Subject: 'Order #%s — On the way.' Body: 'Your driver is on the way. Track in the app.' Period not exclamation."
    depends_on: BV-005

  - finding_id: BV-015
    surface_id: api-email-order-status-delivered
    lens: brand-voice
    severity: P2
    issue: "'Enjoy your meal!' co-exists with in-app push 'Enjoy!' and another in-app status message variant — three sign-offs for one delivery moment. Pick the celebration, not three of them."
    evidence_excerpt: "Subject: 'Order #%s — Your order has been delivered. Enjoy your meal!' — 🎉 emoji"
    related_surfaces: ["api-push-order-delivered-customer", "api-inapp-order-delivered-customer", "api-push-order-update-customer", "web-tx-orderdetail-toasts"]
    recommendation: "Canonical delivered copy across email/push/in-app: 'Delivered. Enjoy your meal.' Period. One emoji 🎉 reserved here only (genuine celebration carve-out from Rule 1)."
    depends_on: BV-005

  - finding_id: BV-016
    surface_id: api-email-order-status-cancelled
    lens: brand-voice
    severity: P1
    issue: "Decorative ❌ emoji on a sensitive event (cancellation) — feels like a Slack reaction, not a brand voice. Plus no refund timeline, no support contact. Voice and content both wrong."
    evidence_excerpt: "Subject: 'Order #%s — Your order has been cancelled' — ❌ emoji"
    related_surfaces: ["api-push-order-cancelled-both", "api-inapp-order-cancelled-both", "web-tx-orderdetail-toasts"]
    recommendation: "Drop emoji. Subject: 'Order #%s — cancelled.' Body must include refund window and contact link (legal-adjacent — see legal lens). Voice should be calm and informative, not decorative."
    depends_on: null

  - finding_id: BV-017
    surface_id: api-email-order-confirm
    lens: brand-voice
    severity: P1
    issue: "'Order Confirmed!' uses Title Case + exclamation; customer-facing budget allows one celebration per page, but the email also has 'Your home chef is preparing your meal!' — two exclamations in one short email."
    evidence_excerpt: "'Order Confirmed! Your order #%s has been placed successfully. Your home chef is preparing your meal!'"
    related_surfaces: ["api-push-order-update-customer-deeplink", "web-tx-orderdetail-toasts", "web-tx-checkout-payment-toasts"]
    recommendation: "Heading: 'Order confirmed.' Body: 'Your order #%s is placed. Your chef is preparing your meal.' Two periods, zero exclamations."
    depends_on: BV-005

  - finding_id: BV-018
    surface_id: api-email-welcome
    lens: brand-voice
    severity: P1
    issue: "Welcome email mixes hype tone with brand drift: 'Welcome to Fe3dr!' subject + 'We're thrilled to have you. Discover amazing home-cooked meals from talented local chefs...' — 'thrilled', 'amazing', 'talented' are bland-marketing tells per brief Anti-references."
    evidence_excerpt: "Subject: 'Welcome to Fe3dr!' — 'Welcome to Fe3dr, %s! We're thrilled to have you. Discover amazing home-cooked meals from talented local chefs...'"
    related_surfaces: ["api-push-user-welcome"]
    recommendation: "Subject: 'Welcome to Fe3dr.' (period). Body: 'Welcome, %s. Discover home chefs cooking near you.' Drop 'thrilled', 'amazing', 'talented' — they read as bland marketing, not the calm-confident voice per .impeccable.md."
    depends_on: BV-001

  - finding_id: BV-019
    surface_id: api-email-welcome
    lens: brand-voice
    severity: P2
    issue: "'Start Exploring' CTA is Title Case — Style Guide §4 requires sentence case for buttons."
    evidence_excerpt: "CTA 'Start Exploring' → https://fe3dr.com"
    related_surfaces: ["api-email-staff-invite", "api-email-chef-verified"]
    recommendation: "'Browse chefs' or 'Get started' (sentence case, ≤3 words, verb-first per Style Guide §4)."
    depends_on: null

  - finding_id: BV-020
    surface_id: api-email-staff-invite
    lens: brand-voice
    severity: P2
    issue: "'You're invited to join Fe3dr' + body 'You've been invited!' — two invitation announcements with two punctuation styles in the same email. Subject is period-less statement, body has exclamation."
    evidence_excerpt: "Subject: 'You're invited to join Fe3dr' — 'You've been invited! %s has invited you to join the Fe3dr team as %s...'"
    related_surfaces: ["ap-staff-invite-success-title", "dp-tx-toast-invite-created", "ap-staff-toasts"]
    recommendation: "Body lead: 'You've been invited.' (period, not exclamation). Match the subject's calm tone."
    depends_on: null

  - finding_id: BV-021
    surface_id: api-email-support-created
    lens: brand-voice
    severity: P0
    issue: "Support emails (created + update) are inline HTML in services/email.go — bypass the emailBase wrapper, so they have no branded header/footer/privacy/unsubscribe. Customer receives an email with no clear sender brand identity."
    evidence_excerpt: "Subject: 'Support Ticket #%s — %s' — 'Support Ticket Created. Your support ticket #%s has been created. Subject: %s. Our team will get back to you shortly.'"
    related_surfaces: ["api-email-support-update", "api-email-account-reminder"]
    recommendation: "Migrate to emailBase template. Match brand header/footer with other transactional emails. The brand-name drift on the rest of the system is compounded when these specific emails arrive unbranded — recipient cannot tell who sent them."
    depends_on: null

  - finding_id: BV-022
    surface_id: api-email-support-update
    lens: brand-voice
    severity: P1
    issue: "Same as BV-021: unbranded inline HTML. Plus 'Support Ticket Update.' as a sentence is sub-grammatical (label, not sentence) — voice reads like an internal ticket dump, not a customer-trust moment."
    evidence_excerpt: "Subject: 'Ticket Update — #%s' — 'Support Ticket Update. Your support ticket #%s has been updated to: %s'"
    related_surfaces: ["api-email-support-created"]
    recommendation: "Use emailBase. Body: 'Your support ticket #%s is now %s.' One sentence, period."
    depends_on: BV-021

  - finding_id: BV-023
    surface_id: api-email-verify
    lens: brand-voice
    severity: P2
    issue: "'Verify your email, %s. Thanks for signing up!' — exclamation again on entry surface (welcome flow). Same as welcome email pattern."
    evidence_excerpt: "Subject: 'Verify your email — Fe3dr' — 'Verify your email, %s. Thanks for signing up! Please verify your email address to activate your account...'"
    related_surfaces: ["api-email-welcome", "api-push-user-welcome"]
    recommendation: "'Thanks for signing up.' Period. Single exclamation budget customer-facing — reserve it for the moment that earns it (Rule 1)."
    depends_on: null

  - finding_id: BV-024
    surface_id: api-email-password-reset
    lens: brand-voice
    severity: P2
    issue: "Subject 'Reset your password — Fe3dr' + body 'We received a request to reset the password for your Fe3dr account...' — repeats brand. Voice is OK but the 'your account is secure' reassurance reads slightly customer-service-script. Acceptable as P3-level note."
    evidence_excerpt: "Subject: 'Reset your password — Fe3dr' — 'Reset your password. We received a request to reset the password for your Fe3dr account...'"
    related_surfaces: ["api-email-verify"]
    recommendation: "'Reset your password. If this wasn't you, your account stays safe — just ignore this email.' (plain English over 'your account is secure' reassurance jargon)."
    depends_on: null

  - finding_id: BV-025
    surface_id: api-inapp-approval-approved-chef
    lens: brand-voice
    severity: P0
    issue: "Raw snake_case enum slug surfaces to chef: 'Your menu_item has been approved' / 'Your delivery_zone has been approved'. Technical leak, not brand voice."
    evidence_excerpt: "Title: 'Request Approved' — Message: 'Your %s has been approved: %s' where %s = 'menu_item' / 'delivery_zone' etc."
    related_surfaces: ["api-push-approval-approved-chef", "api-inapp-approval-info-chef", "api-push-approval-info-chef", "api-inapp-approval-rejected-chef", "api-push-approval-rejected-chef"]
    recommendation: "Add a humanReadable map for approval types: menu_item → 'menu item', delivery_zone → 'delivery zone', kitchen_photo → 'kitchen photo'. Apply before substitution."
    depends_on: null

  - finding_id: BV-026
    surface_id: api-push-approval-approved-chef
    lens: brand-voice
    severity: P1
    issue: "Title 'Request Approved!' has exclamation; in-app 'Request Approved' does not. Same event, two voices. Plus chef-tone budget = 0 exclamations."
    evidence_excerpt: "Push Title: 'Request Approved!' — Body: 'Your %s has been approved'"
    related_surfaces: ["api-inapp-approval-approved-chef"]
    recommendation: "Match in-app: 'Request approved' (sentence case, no exclamation)."
    depends_on: BV-025

  - finding_id: BV-027
    surface_id: api-inapp-approval-rejected-chef
    lens: brand-voice
    severity: P1
    issue: "'Your %s has been rejected. Notes: %s' — appends raw admin notes verbatim. Risks leaking internal language, profanity, or shorthand into a chef notification. Brand voice gets hijacked by whatever the admin typed."
    evidence_excerpt: "Title: 'Request Rejected' — Message: 'Your %s has been rejected. Notes: %s'"
    related_surfaces: ["api-push-approval-rejected-chef", "api-inapp-approval-info-chef", "api-push-approval-info-chef"]
    recommendation: "Either (a) wrap admin notes in a separate field with a label like 'Reviewer comment:' and bound to a sane length, or (b) require admin notes be reviewed/templated before they surface in a transactional notification. Brand voice can't be 'whatever admin wrote'."
    depends_on: BV-025

  - finding_id: BV-028
    surface_id: api-inapp-chef-responded
    lens: brand-voice
    severity: P1
    issue: "'Chef Responded: ' + chef-authored message — admins receive chef text verbatim in a notification title/body. Brand voice cannot guarantee tone consistency. No profanity filter, no length cap per inventory."
    evidence_excerpt: "Title: 'Chef Responded: ' + approval.Title — Message: req.Response (chef-authored)"
    related_surfaces: ["api-inapp-staff-invite", "api-inapp-approval-rejected-chef"]
    recommendation: "Bound chef message to 280 chars in notification, full message visible on click-through. Title: 'Chef replied — %s' (where %s = approval.Title only, not 'Chef Responded:' colon dump). Same applies to staff-invite where admin authors copy."
    depends_on: null

  - finding_id: BV-029
    surface_id: api-push-order-cancelled-both
    lens: brand-voice
    severity: P0
    issue: "Identical push body 'Order has been cancelled' sent to BOTH customer and chef. Two personas with completely different needs (customer: when's my refund? chef: do I keep the prep?) receive the same generic line. Brand voice fails persona matrix entirely."
    evidence_excerpt: "Title: 'Order Cancelled' — Body: 'Order has been cancelled' (same to customer AND chef)"
    related_surfaces: ["api-inapp-order-cancelled-both", "api-email-order-status-cancelled"]
    recommendation: "Persona-split bodies. Customer: 'Your order was cancelled. Refund in 5-7 days.' Chef: 'Order #%s cancelled — no further action needed.' Two routes, two voices, one event."
    depends_on: null

  - finding_id: BV-030
    surface_id: api-push-delivery-pickedup-customer
    lens: brand-voice
    severity: P1
    issue: "Push title 'Order On The Way!' uses Title Case (every word capitalized) + exclamation. Style Guide §3 bans Title Case outside proper nouns. Plus in-app variant title is 'Order Picked Up' — different tense, different voice."
    evidence_excerpt: "Push Title: 'Order On The Way!' — Body: 'Your order has been picked up and is on its way to you!'"
    related_surfaces: ["api-inapp-delivery-pickedup-customer", "api-email-order-status-pickedup", "api-email-order-status-onway"]
    recommendation: "Title: 'On the way' (sentence case). Body: 'Your driver has picked up your order.' One sentence, no exclamation."
    depends_on: BV-005

  - finding_id: BV-031
    surface_id: api-inapp-delivery-pickedup-customer
    lens: brand-voice
    severity: P2
    issue: "In-app title 'Order Picked Up' uses Title Case. Style Guide §3 sentence case for everything except proper nouns."
    evidence_excerpt: "Title: 'Order Picked Up' — Message: 'Your order has been picked up and is on its way!'"
    related_surfaces: ["api-push-delivery-pickedup-customer"]
    recommendation: "'Order picked up' (sentence case)."
    depends_on: BV-030

  - finding_id: BV-032
    surface_id: api-inapp-order-status-customer
    lens: brand-voice
    severity: P2
    issue: "In-app title 'Order Status Updated' vs push title 'Order Update' for the same event. Tense and word count differ — voice drift."
    evidence_excerpt: "In-app title: 'Order Status Updated' vs push title: 'Order Update'"
    related_surfaces: ["api-push-order-update-customer", "api-push-order-update-customer-deeplink"]
    recommendation: "Both: 'Order update' (sentence case, two words). Body carries the specific status."
    depends_on: BV-005

  - finding_id: BV-033
    surface_id: api-push-delivery-assigned-customer
    lens: brand-voice
    severity: P2
    issue: "Push 'A delivery partner has been assigned to your order and will pick it up soon!' is wordy (16 words) and ends with exclamation. In-app variant is 9 words. Two voices for one event."
    evidence_excerpt: "Push: 'Delivery Partner Assigned' / 'A delivery partner has been assigned to your order and will pick it up soon!' vs In-app: 'A delivery partner has been assigned to your order'"
    related_surfaces: ["api-inapp-delivery-assigned-customer", "api-email-delivery-assigned"]
    recommendation: "Push body: 'Your delivery partner is on the way.' (period). Title sentence-case: 'Delivery partner assigned'."
    depends_on: null

  - finding_id: BV-034
    surface_id: api-push-delivery-assigned-driver
    lens: brand-voice
    severity: P2
    issue: "Driver push title 'New Delivery Available' is Title Case; body 'A delivery near you is ready for pickup' is 8 words — Style Guide driver tone targets ≤4 words where possible."
    evidence_excerpt: "Title: 'New Delivery Available' — Body: 'A delivery near you is ready for pickup'"
    related_surfaces: ["api-email-delivery-assigned", "dp-tx-toast-accept-success"]
    recommendation: "Title: 'New delivery' (sentence case, telegraphic). Body: 'Pickup nearby.' (≤4 words per driver matrix)."
    depends_on: null

  - finding_id: BV-035
    surface_id: api-inapp-driver-onboarding-admins
    lens: brand-voice
    severity: P3
    issue: "'A new driver from %s has submitted their onboarding application for review.' — 16 words. Admin tone is precise but this reads slightly verbose."
    evidence_excerpt: "Title: 'New Driver Application' — Message: 'A new driver from %s has submitted their onboarding application for review.'"
    related_surfaces: ["api-inapp-approval-created-admin"]
    recommendation: "Title: 'New driver application' (sentence case). Message: 'A driver from %s submitted for review.' Tighter, admin-direct."
    depends_on: null

  - finding_id: BV-036
    surface_id: api-inapp-approval-created-admin
    lens: brand-voice
    severity: P2
    issue: "Title 'New Approval Request' Title Case. Body 'New approval request pending: %s' duplicates the title word-for-word — title and body should not repeat."
    evidence_excerpt: "Title: 'New Approval Request' — Message: 'New approval request pending: %s'"
    related_surfaces: ["api-inapp-driver-onboarding-admins"]
    recommendation: "Title: 'New approval request'. Body: 'Pending: %s' — short, admin-functional."
    depends_on: null

  - finding_id: BV-037
    surface_id: api-success-message-generic
    lens: brand-voice
    severity: P1
    issue: "43 API success-message strings have inconsistent 'successfully' suffix — some have it ('Password changed successfully'), some don't ('2FA disabled'). Sibling strings on the same handler family drift in shape, breaking systematic voice."
    evidence_excerpt: "'2FA disabled' vs 'Password changed successfully' vs 'Email verified successfully' vs 'Image deleted' vs 'Provider deleted successfully' vs 'Address deleted'"
    related_surfaces: ["web-tx-admin-settings", "vp-ux-profile-toasts", "ap-staffdetail-toasts", "vp-ux-menu-form-toasts"]
    recommendation: "Style Guide §4 success toast: past tense, ≤6 words, period. Drop 'successfully' across all 43 — it's filler. 'Password changed.' 'Email verified.' 'Provider deleted.' Apply same to the frontend toasts that mirror these (web/vp/ap)."
    depends_on: null

  - finding_id: BV-038
    surface_id: web-tx-catering-quotes
    lens: brand-voice
    severity: P2
    issue: "'Quote accepted! The chef has been notified.' — exclamation on a routine action. Style Guide Rule 1: only genuine celebration. Plus 'The chef has been notified' is passive; voice should be active."
    evidence_excerpt: "'Quote accepted! The chef has been notified.'"
    related_surfaces: ["web-tx-catering-request-submit", "web-tx-chef-catering"]
    recommendation: "'Quote accepted. We told the chef.' Period. Active. Tighter."
    depends_on: null

  - finding_id: BV-039
    surface_id: web-tx-catering-request-submit
    lens: brand-voice
    severity: P2
    issue: "'Catering request submitted successfully!' — exclamation + 'successfully' filler. Two voice violations."
    evidence_excerpt: "'Catering request submitted successfully! / Failed to submit request. Please try again.'"
    related_surfaces: ["web-tx-catering-quotes", "vp-onb-submit-success", "dp-tx-toast-step5-submitted"]
    recommendation: "'Request submitted.' Period, ≤6 words."
    depends_on: BV-037

  - finding_id: BV-040
    surface_id: web-tx-checkout-payment-toasts
    lens: brand-voice
    severity: P1
    issue: "'Payment successful!' on a routine customer payment — exclamation budget should be reserved. Plus 'Payment successful' is fine as a status; the bang reads consumer-grade-hype."
    evidence_excerpt: "'Failed to initiate payment / Payment gateway is loading / Payment successful! / Payment verification failed...'"
    related_surfaces: ["web-tx-orderdetail-toasts", "api-email-order-confirm"]
    recommendation: "'Payment confirmed.' (Style Guide §3 prefers 'Paid' / 'Payment successful' but picks one per surface. Pick 'Payment confirmed' across web checkout + order detail + email — consistent verb, no exclamation)."
    depends_on: null

  - finding_id: BV-041
    surface_id: web-tx-delivery-actions
    lens: brand-voice
    severity: P1
    issue: "Driver-portal toasts 'Delivery accepted!' and 'Delivery completed! Great job!' — driver tone budget = 0 exclamations + 0 emoji + ≤4 words. 'Great job!' is gamification cheer (anti-reference: 2018 indie ecommerce / DTC hype)."
    evidence_excerpt: "'Delivery accepted! / Delivery completed! Great job! / Status updated / You are now offline / You are now online'"
    related_surfaces: ["dp-tx-toast-online", "dp-tx-toast-accept-success", "dp-tx-toast-status-updated"]
    recommendation: "'Delivery accepted.' 'Delivery complete.' Drop 'Great job!'. Telegraphic, calm, dignified — driver is at work, not playing a game."
    depends_on: null

  - finding_id: BV-042
    surface_id: dp-tx-toast-accept-success
    lens: brand-voice
    severity: P2
    issue: "'Delivery accepted!' (delivery-portal driver web) duplicates the same exclamation pattern flagged in BV-041. Same string, different file — confirms consistent drift."
    evidence_excerpt: "`Delivery accepted!`"
    related_surfaces: ["web-tx-delivery-actions", "mv-undo-accepted"]
    recommendation: "'Delivery accepted.' Period."
    depends_on: BV-041

  - finding_id: BV-043
    surface_id: dp-tx-toast-step5-submitted
    lens: brand-voice
    severity: P2
    issue: "Driver onboarding submit toast 'Application submitted successfully!' — driver tone, exclamation + 'successfully' filler."
    evidence_excerpt: "`Application submitted successfully!`"
    related_surfaces: ["mv-onb-pending-submitted-title", "vp-onb-submit-success", "web-tx-catering-request-submit"]
    recommendation: "'Application submitted.' Period, ≤6 words."
    depends_on: BV-037

  - finding_id: BV-044
    surface_id: vp-onb-submit-success
    lens: brand-voice
    severity: P2
    issue: "Chef onboarding submit 'Application submitted! We'll review and get back to you within 24-48 hours.' — chef tone, 0-exclamation budget. The SLA line is useful; the exclamation is not."
    evidence_excerpt: "'Application submitted! We'll review and get back to you within 24-48 hours.'"
    related_surfaces: ["vp-onb-policies-ready-banner", "mv-onb-pending-submitted-title", "dp-tx-toast-step5-submitted"]
    recommendation: "'Application submitted. We'll review and reply within 24–48 hours.' (period + en-dash on hour range)."
    depends_on: null

  - finding_id: BV-045
    surface_id: vp-onb-policies-ready-banner
    lens: brand-voice
    severity: P2
    issue: "'You're all set to submit your application!' — exclamation on chef-facing onboarding step. Plus 'You're all set' is generic SaaS-onboarding voice (anti-reference)."
    evidence_excerpt: "'You're all set to submit your application! / Our team will review your details and you'll be notified within 24-48 hours.'"
    related_surfaces: ["vp-onb-submit-success", "vp-onb-personal-avatar-success"]
    recommendation: "'Ready to submit.' Period. Tighter chef tone. Body second sentence is good — keep."
    depends_on: BV-044

  - finding_id: BV-046
    surface_id: mv-onb-pending-submitted-title
    lens: brand-voice
    severity: P2
    issue: "Mobile-vendor 'Application Submitted!' — Title Case + exclamation in chef-facing serif display title. Inventory notes this is a 'lone exclamation in app' — confirms it's a stylistic outlier."
    evidence_excerpt: "'Application Submitted!'"
    related_surfaces: ["vp-onb-submit-success", "dp-tx-toast-step5-submitted"]
    recommendation: "'Application submitted' (sentence case, no exclamation)."
    depends_on: BV-044

  - finding_id: BV-047
    surface_id: mc-catering-submit-success
    lens: brand-voice
    severity: P2
    issue: "Mobile-customer 'Request Submitted! / Chefs will review and send quotes.' — Title Case + exclamation on alert title."
    evidence_excerpt: "'Request Submitted! / Chefs will review and send quotes.'"
    related_surfaces: ["web-tx-catering-request-submit", "vp-onb-submit-success"]
    recommendation: "'Request submitted' (sentence case, no exclamation). Body OK."
    depends_on: BV-044

  - finding_id: BV-048
    surface_id: mc-profile-logout-confirm
    lens: brand-voice
    severity: P1
    issue: "Customer mobile alert: title 'Log out' (sentence case, two words) but button label 'Log Out' (Title Case). Same screen, two casings of the same verb. Plus Style Guide §3 mandates 'Sign out' not 'Log out'."
    evidence_excerpt: "'Log out / Are you sure you want to log out? / Cancel / Log Out'"
    related_surfaces: ["md-trx-006", "md-trx-007"]
    recommendation: "Title: 'Sign out'. Body: 'Sign out of your account?' Button: 'Sign out' (sentence case). One verb, one casing, three matching tokens."
    depends_on: null

  - finding_id: BV-049
    surface_id: md-trx-006
    lens: brand-voice
    severity: P1
    issue: "Driver mobile alert title 'Logout' — single word + wrong verb. Style Guide bans both 'Logout' (one word) and 'Log out' (verb form); requires 'Sign out'."
    evidence_excerpt: "'Logout'"
    related_surfaces: ["md-trx-007", "mc-profile-logout-confirm"]
    recommendation: "'Sign out'. Same on the body 'Are you sure you want to sign out?'."
    depends_on: BV-048

  - finding_id: BV-050
    surface_id: md-trx-007
    lens: brand-voice
    severity: P2
    issue: "Driver mobile alert body 'Are you sure you want to logout?' — uses 'logout' as a verb (banned per Style Guide §3, plus single-word form is wrong)."
    evidence_excerpt: "'Are you sure you want to logout?'"
    related_surfaces: ["md-trx-006", "mc-profile-logout-confirm"]
    recommendation: "'Sign out of your account?' (drops the 'Are you sure' hedge per Style Guide §4 modal-subtitles — explain consequence, not ask permission)."
    depends_on: BV-049

  - finding_id: BV-051
    surface_id: md-trx-011
    lens: brand-voice
    severity: P2
    issue: "Driver alert title 'Validation' — generic system label, not brand voice. Reads like default RN Alert title."
    evidence_excerpt: "'Validation'"
    related_surfaces: ["md-trx-013", "mv-profile-update-success-body"]
    recommendation: "Use a descriptive title: 'Missing details' (matches body 'Name, phone, and city are required.')."
    depends_on: null

  - finding_id: BV-052
    surface_id: md-trx-013
    lens: brand-voice
    severity: P2
    issue: "Driver alert title 'Success' — generic non-brand label. Drop the title entirely or replace with the actual outcome."
    evidence_excerpt: "'Success'"
    related_surfaces: ["md-trx-011", "mv-profile-update-success-body"]
    recommendation: "Drop. Or 'Profile saved' (matches body)."
    depends_on: BV-051

  - finding_id: BV-053
    surface_id: md-trx-008
    lens: brand-voice
    severity: P2
    issue: "Driver alert title 'Subscription' — generic label, not voice."
    evidence_excerpt: "'Subscription'"
    related_surfaces: ["md-trx-009", "md-trx-013"]
    recommendation: "'Manage on web' (matches the body which says 'Visit the web portal to manage your subscription')."
    depends_on: null

  - finding_id: BV-054
    surface_id: mv-profile-update-success-body
    lens: brand-voice
    severity: P2
    issue: "Mobile-vendor alert: title 'Success' (generic) + body 'Profile updated successfully.' — title is non-brand, body has 'successfully' filler."
    evidence_excerpt: "Title: 'Success' / Body: 'Profile updated successfully.'"
    related_surfaces: ["md-trx-013", "md-trx-014", "vp-ux-profile-toasts"]
    recommendation: "Replace alert with a toast: 'Profile updated.' One toast, no generic 'Success' title."
    depends_on: null

  - finding_id: BV-055
    surface_id: mv-reviewdetail-success-title
    lens: brand-voice
    severity: P3
    issue: "'Reply Sent' alert title — Title Case in chef-facing app."
    evidence_excerpt: "Title: 'Reply Sent' / Body: 'Your reply has been posted.'"
    related_surfaces: ["mv-profile-update-success-body"]
    recommendation: "'Reply posted' (sentence case, past tense, matches body verb)."
    depends_on: null

  - finding_id: BV-056
    surface_id: mc-profile-save-success
    lens: brand-voice
    severity: P3
    issue: "Customer mobile: 'Saved / Profile updated successfully. / Cuisine preferences updated.' — sibling toasts inconsistent: one has 'successfully', one doesn't."
    evidence_excerpt: "'Saved / Profile updated successfully. / Cuisine preferences updated.'"
    related_surfaces: ["vp-ux-profile-toasts", "web-tx-profile-update"]
    recommendation: "All three follow Style Guide §4: past tense, ≤6 words, period. 'Saved.' / 'Profile updated.' / 'Preferences updated.'."
    depends_on: BV-037

  - finding_id: BV-057
    surface_id: mc-menuitem-cross-chef
    lens: brand-voice
    severity: P2
    issue: "Cross-chef cart conflict: mobile says 'Replace Cart? / You have items from another chef. Replace cart? / Cancel / Replace'. Web says 'Your cart has items from another chef. Clear cart first.' Two different verbs (Replace / Clear), two different phrasings."
    evidence_excerpt: "Mobile: 'Replace Cart? ... Replace cart? ... Replace' vs Web: 'Your cart has items from another chef. Clear cart first.'"
    related_surfaces: ["web-tx-chefdetail-add-to-cart"]
    recommendation: "Pick one verb across web + mobile. Suggested: 'Replace cart? Your cart has items from another chef.' (active question, 9 words). Buttons: 'Cancel' / 'Replace'. Web rewrites to same. Plus drop 'Replace Cart?' Title Case."
    depends_on: null

  - finding_id: BV-058
    surface_id: web-tx-chef-order-status
    lens: brand-voice
    severity: P3
    issue: "Web chef portal: 'Order status updated' vs vendor-portal: 'Order status updated' (matches) but mobile-vendor uses 'Order accepted' / 'Order rejected' (different shape). Same domain, three different message families."
    evidence_excerpt: "Web chef: 'Order status updated' / VP: 'Order status updated' / MV: 'Order accepted' / 'Order rejected'"
    related_surfaces: ["vp-ux-orders-live-toasts", "mv-undo-accepted", "mv-undo-rejected"]
    recommendation: "On chef-facing toasts, prefer the specific state ('Order accepted' / 'Order rejected' / 'Order marked ready') over generic 'Order status updated'. Migrate web + VP to match MV's specificity."
    depends_on: null

  - finding_id: BV-059
    surface_id: ap-staff-invite-success-title
    lens: brand-voice
    severity: P2
    issue: "Admin-portal 'Invitation created successfully!' — admin tone budget = 0 exclamations + 'successfully' filler. Twice over."
    evidence_excerpt: "'Invitation created successfully!'"
    related_surfaces: ["dp-tx-toast-invite-created", "ap-staff-toasts", "api-success-message-generic"]
    recommendation: "'Invitation created.' Period, no filler. Style Guide §4 admin tone is direct."
    depends_on: BV-037

  - finding_id: BV-060
    surface_id: ap-staff-toasts
    lens: brand-voice
    severity: P3
    issue: "Sibling admin toasts oscillate: 'Invitation sent successfully' / 'Invitation resent' / 'Invitation revoked' — same verb family, inconsistent suffix discipline."
    evidence_excerpt: "'Invitation revoked / Invitation resent / Invitation sent successfully / Invitation created successfully!'"
    related_surfaces: ["ap-staff-invite-success-title", "dp-tx-toast-invite-created", "dp-tx-toast-invite-revoked", "dp-tx-toast-invite-resent"]
    recommendation: "Apply systematically: 'Invitation sent.' / 'Invitation resent.' / 'Invitation revoked.' / 'Invitation created.' All past tense, all period, no 'successfully'."
    depends_on: BV-037

  - finding_id: BV-061
    surface_id: ap-providerdetail-toasts
    lens: brand-voice
    severity: P2
    issue: "'Connection successful ({ms}ms)' uses 'successful' as adjective in admin context; sibling 'Provider deleted' is past tense. Two grammatical shapes for admin operational outcomes."
    evidence_excerpt: "'Connection successful ({ms}ms); Connection failed: {err}; ... Provider status updated; Provider deleted'"
    related_surfaces: ["ap-providers-toasts", "ap-secsettings-toasts"]
    recommendation: "Normalize to past tense + outcome noun: 'Connected ({ms}ms).' / 'Connection failed: {err}.' / 'Provider deleted.' / 'Provider status updated.'."
    depends_on: null

  - finding_id: BV-062
    surface_id: ap-secsettings-toasts
    lens: brand-voice
    severity: P2
    issue: "'2FA enabled successfully' but '2FA disabled' (no suffix) — same handler family, two different shapes. Plus 'All other sessions revoked. You may need to sign in again.' is good voice; rest is inconsistent."
    evidence_excerpt: "'... 2FA enabled successfully; 2FA disabled; Enforcement setting updated; Invalid code; Enroll failed; ...'"
    related_surfaces: ["web-tx-profile-2fa", "api-success-message-generic"]
    recommendation: "'2FA enabled.' / '2FA disabled.' Symmetric. Apply to all paired enable/disable verbs."
    depends_on: BV-037

  - finding_id: BV-063
    surface_id: ap-platsettings-feedback
    lens: brand-voice
    severity: P3
    issue: "Sibling settings-save banners follow consistent past-tense shape ('Commission rates updated', 'Delivery fees updated', 'Operating hours updated', 'Zone created', 'Zone deleted') — but failure pair is generic ('Failed to save', 'Failed to create'). Failure should match the specific noun."
    evidence_excerpt: "Success: 'Commission rates updated; Delivery fees updated; ...' Failure: 'Failed to save; Failed to create'"
    related_surfaces: ["ap-providers-toasts", "ap-staff-toasts"]
    recommendation: "Pair-match failure verbs: 'Failed to update commission rates.' / 'Failed to create zone.' Generic 'Failed to save' loses context."
    depends_on: null

  - finding_id: BV-064
    surface_id: vp-ux-menu-form-category-toasts
    lens: brand-voice
    severity: P3
    issue: "'A category with this name already exists' is fine; sibling 'Category \"{name}\" created' uses quotes around name. Inconsistent — most other VP toasts don't quote interpolated values."
    evidence_excerpt: "'Category \"{name}\" created / A category with this name already exists / Failed to create category'"
    related_surfaces: ["web-tx-favorites-remove", "web-tx-chefdetail-add-to-cart"]
    recommendation: "Drop the quotes: 'Category {name} created.' Or apply quoting systematically across all toast interpolations."
    depends_on: null

  - finding_id: BV-065
    surface_id: dp-tx-toast-partner-suspend
    lens: brand-voice
    severity: P3
    issue: "Cross-app verb drift on a sensitive admin action: delivery-portal admin uses 'Partner suspended'; admin-portal uses 'Chef suspended' / 'User suspended' / 'Delivery partner suspended'. Four different objects ('Partner', 'Chef', 'User', 'Delivery partner') for the same suspend verb."
    evidence_excerpt: "DP: 'Partner reactivated / Partner suspended' vs AP: 'Chef suspended / User suspended' vs API: 'Delivery partner suspended'"
    related_surfaces: ["ap-chefs-toast-suspended", "ap-users-toast-suspended", "api-success-message-generic"]
    recommendation: "Apply the Style Guide noun policy. 'Chef', 'Driver', 'Customer' are the canonical roles. Toasts: 'Chef suspended.' / 'Driver suspended.' / 'Customer suspended.' — never 'Partner' or 'User' (banned in §3)."
    depends_on: null

  - finding_id: BV-066
    surface_id: web-tx-profile-update
    lens: brand-voice
    severity: P3
    issue: "Sibling toasts on the same profile page: 'Profile updated successfully' (with filler) and 'Profile photo updated' (no filler). Same handler family."
    evidence_excerpt: "'Profile updated successfully / Failed to update profile / File too large. Maximum 5 MB. / Invalid file type. Use JPEG, PNG or WebP. / Profile photo updated'"
    related_surfaces: ["vp-ux-profile-toasts", "mc-profile-save-success", "vp-onb-personal-avatar-success"]
    recommendation: "'Profile updated.' (drop 'successfully' — Style Guide §4). Match the photo toast pattern."
    depends_on: BV-037

  - finding_id: BV-067
    surface_id: web-tx-profile-2fa
    lens: brand-voice
    severity: P2
    issue: "2FA toasts mix grand-statement and tiny-detail: 'Two-factor authentication enabled' / 'Two-factor authentication disabled' (verbose) vs 'Key copied' / 'Backup codes copied to clipboard' (one verbose, one telegraphic). Voice rhythm fails."
    evidence_excerpt: "'Failed to initiate 2FA setup / Two-factor authentication enabled / ... / Backup codes regenerated / Key copied / Backup codes copied to clipboard'"
    related_surfaces: ["ap-secsettings-toasts"]
    recommendation: "Pick one length per category. Title abbrev acceptable for known concept: '2FA enabled.' / '2FA disabled.' / 'Backup codes regenerated.' / 'Key copied.' / 'Backup codes copied.'."
    depends_on: BV-066

  - finding_id: BV-068
    surface_id: web-tx-orderdetail-toasts
    lens: brand-voice
    severity: P2
    issue: "'Payment confirmed' (this surface) vs 'Payment successful!' (checkout, BV-040) — same payment outcome, two different verbs in two surfaces the same customer hits within seconds."
    evidence_excerpt: "'Payment confirmed / Payment verification failed — please contact support / Order cancelled successfully / Failed to cancel order / Order number copied'"
    related_surfaces: ["web-tx-checkout-payment-toasts", "api-success-message-generic"]
    recommendation: "Canonical: 'Payment confirmed.' across checkout + order detail + API + email. 'Order cancelled.' (drop 'successfully')."
    depends_on: BV-040

  - finding_id: BV-069
    surface_id: api-inapp-staff-invite
    lens: brand-voice
    severity: P2
    issue: "Title/Message are admin-authored (per inventory note). Brand voice cannot guarantee consistency. Same risk pattern as BV-027 / BV-028."
    evidence_excerpt: "Title: <invitation.Title> e.g. 'Staff Invitation' — Message: <req.Message> (admin-authored)"
    related_surfaces: ["api-inapp-chef-responded", "api-inapp-approval-rejected-chef"]
    recommendation: "Wrap admin-authored copy with a platform-controlled prefix: 'From admin: <message>'. Cap length to 280 chars. Strip newlines and HTML. Brand-voice escape hatch becomes platform liability otherwise."
    depends_on: BV-028

  - finding_id: BV-070
    surface_id: vp-ux-kitchen-photo-toasts
    lens: brand-voice
    severity: P3
    issue: "'Kitchen setup saved successfully' (with filler) alongside sibling 'Photo removed' (no filler). Same family inconsistent."
    evidence_excerpt: "'Kitchen photo uploaded / ... / Kitchen setup saved successfully / Failed to save kitchen setup'"
    related_surfaces: ["vp-ux-profile-toasts", "vp-ux-settings-payout-toasts", "web-tx-profile-update"]
    recommendation: "'Kitchen setup saved.' Drop 'successfully'. Consistent past tense across the file."
    depends_on: BV-037
```
