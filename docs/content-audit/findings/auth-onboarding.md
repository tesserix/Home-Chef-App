# Findings — Auth-Onboarding

Category: auth-onboarding
Lenses applied: technical-writer, legal, business-analyst, brand-voice
Inventory slice size: ~301 surfaces
Total findings: 418

## Severity summary

| Severity | TW | Legal | BA | BV | Total |
|---|---|---|---|---|---|
| P0 | 2 | 36 | 4 | 13 | 55 |
| P1 | 10 | 50 | 20 | 18 | 98 |
| P2 | 129 | 14 | 30 | 16 | 189 |
| P3 | 51 | 6 | 6 | 13 | 76 |

## Cross-lens consensus surfaces

Surfaces flagged by all 4 lenses (highest priority — every lens agreed there's a problem):

- **`web-auth-register-heading`** — flagged by TW(P3), Legal(P1, P0, P0, P1), BA(P1), BV(P0, P2, P3) — customer signup heading + DPDP §5/§6 consent notice gaps + AI-slop "500+ home chefs" social-proof

Surfaces flagged by 3 lenses (also high-priority rewrites):

- **`ap-auth-login-feature-list`** — flagged by TW(P1), Legal(P2), BV(P1) — admin-portal login feature bullets
- **`ap-auth-login-instruction`** — flagged by TW(P3), Legal(P1), BV(P2) — admin-portal login instruction copy
- **`dp-auth-login-title`** — flagged by TW(P3), Legal(P3, P0), BV(P0) — delivery-portal login title + onboarding entry copy
- **`dp-auth-onboarding-header`** — flagged by TW(P2), Legal(P2, P0), BV(P1) — delivery-portal onboarding header
- **`dp-auth-status-descriptions`** — flagged by TW(P2), Legal(P1), BV(P1) — delivery-portal status descriptions
- **`dp-auth-step4-billing-howto`** — flagged by TW(P2), Legal(P1), BV(P1) — delivery-portal billing how-to step
- **`dp-auth-step4-h`** — flagged by TW(P2), Legal(P1), BV(P2) — delivery-portal step 4 heading
- **`dp-auth-step4-secure-pay`** — flagged by TW(P2), Legal(P1), BV(P3) — delivery-portal "secure pay" claim
- **`dp-auth-step4-zero-commission`** — flagged by TW(P1), Legal(P0), BV(P0) — delivery-portal "zero commission" claim (unverified pricing promise)
- **`dp-auth-step5-terms`** — flagged by TW(P1), Legal(P0, P0), BV(P1) — delivery-portal terms acceptance step
- **`mc-onb-step1-labels`** — flagged by TW(P2), Legal(P1), BV(P2) — mobile-customer onboarding step 1 labels
- **`md-auth-002`** — flagged by TW(P0), Legal(P2), BV(P1) — mobile-delivery auth surface
- **`md-onb-018`** — flagged by TW(P1, P2), Legal(P2), BV(P2) — mobile-delivery onboarding surface 018
- **`md-onb-109`** — flagged by TW(P2), Legal(P1), BV(P1) — mobile-delivery onboarding surface 109
- **`vp-auth-forgot-redirect`** — flagged by TW(P2), Legal(P3), BV(P3) — vendor-portal forgot-password redirect copy
- **`vp-auth-login-features-list`** — flagged by TW(P2), Legal(P1), BV(P1, P1, P2) — vendor-portal login feature bullets
- **`vp-auth-login-hero-heading`** — flagged by TW(P2), Legal(P2), BV(P1) — vendor-portal login hero heading
- **`vp-auth-register-hero-sub`** — flagged by TW(P1, P1), Legal(P1, P2), BV(P0, P0) — vendor-portal register hero sub
- **`vp-onb-ops-pricing-title`** — flagged by TW(P2), Legal(P1), BV(P1) — vendor-portal onboarding ops pricing title
- **`vp-onb-personal-fields`** — flagged by TW(P2), Legal(P0), BV(P2, P3) — vendor-portal onboarding personal-data fields (DPDP §5 notice gap)
- **`vp-onb-stepper-labels`** — flagged by TW(P2), Legal(P1, P0), BV(P1) — vendor-portal onboarding stepper labels
- **`web-auth-login-email-fields`** — flagged by TW(P3, P2), Legal(P1), BV(P3) — customer login email/password fields
- **`web-auth-login-heading`** — flagged by TW(P2), BA(P3), BV(P3) — customer login heading
- **`web-auth-onboarding-heading`** — flagged by TW(P2, P3, P2), Legal(P1), BV(P2, P3) — customer onboarding heading

## Technical Writer findings

```yaml
# Technical Writer lens findings — auth-onboarding category
# Auditor: TW lens agent
# Date: 2026-05-13
# Style guide: docs/content-audit/STYLE-GUIDE.md
# Brief: docs/content-audit/lens-briefs/technical-writer.md

findings:

  # ===========================================================
  # WEB (customer-facing auth + onboarding)
  # ===========================================================

  - finding_id: TW-001
    surface_id: web-auth-login-email-fields
    lens: technical-writer
    severity: P3
    issue: "Placeholder 'Enter your password' restates the label. Adds no value, fights translation slack."
    evidence_excerpt: "Password / Enter your password"
    recommendation: "Remove the placeholder. Empty placeholder is preferred for password fields (security best practice — discourages auto-suggest UI hints)."
    depends_on: null

  - finding_id: TW-002
    surface_id: web-auth-login-email-fields
    lens: technical-writer
    severity: P2
    issue: "'Forgot password?' is fine as a link but should be styled and worded as one phrase. Currently mixed-case sentence + question mark. Style guide prefers descriptive link text, sentence case, no trailing punctuation on inline links."
    evidence_excerpt: "Forgot password?"
    recommendation: "'Reset password' (verb-first, no question mark)."
    depends_on: null

  - finding_id: TW-003
    surface_id: web-auth-login-submit
    lens: technical-writer
    severity: P3
    issue: "Loading state uses ellipsis 'Signing in...' which is acceptable, but inconsistent with other web surfaces that use 'Signing in' (no ellipsis) — see vendor-portal."
    evidence_excerpt: "Sign in / Signing in..."
    recommendation: "Standardise loading copy across portals: 'Signing in…' with single Unicode ellipsis (…) not three dots. Apply consistently."
    depends_on: null

  - finding_id: TW-004
    surface_id: web-auth-login-heading
    lens: technical-writer
    severity: P2
    issue: "'Don't have an account? Sign up' uses inline question + CTA. Style guide prefers single-clause statements for sub-links."
    evidence_excerpt: "Don't have an account? Sign up"
    recommendation: "'New here? Create an account.' OR keep as-is but ensure 'Sign up' is the link target and verb is consistent (style guide allows 'Sign up' as verb)."
    depends_on: null

  - finding_id: TW-005
    surface_id: web-auth-register-heading
    lens: technical-writer
    severity: P3
    issue: "Heading 'Create your account' is fine; sub-link 'Already have an account? Sign in' mirrors the inverse problem from TW-004 — inline question + CTA."
    evidence_excerpt: "Already have an account? Sign in"
    recommendation: "'Returning? Sign in.' Keep parallel structure with login page sub-link."
    depends_on: null

  - finding_id: TW-006
    surface_id: web-auth-register-form-fields
    lens: technical-writer
    severity: P2
    issue: "Helper text 'Min. 8 characters' uses abbreviation. Style guide prefers plain English."
    evidence_excerpt: "Min. 8 characters"
    recommendation: "'At least 8 characters.' Reads naturally, translates cleanly."
    depends_on: null

  - finding_id: TW-007
    surface_id: web-auth-register-form-fields
    lens: technical-writer
    severity: P3
    issue: "Placeholder 'Re-enter password' on confirm-password field — confirm-password helper text should explain WHY ('to make sure they match'), not just restate."
    evidence_excerpt: "Confirm password / Re-enter password"
    recommendation: "Remove placeholder; add helper text under input: 'We'll check it matches.'"
    depends_on: null

  - finding_id: TW-008
    surface_id: web-auth-onboarding-step-address
    lens: technical-writer
    severity: P2
    issue: "Title 'Delivery Address' uses Title Case. Style guide requires sentence case for form labels and section titles."
    evidence_excerpt: "Delivery Address"
    recommendation: "'Delivery address' (sentence case throughout)."
    depends_on: null

  - finding_id: TW-009
    surface_id: web-auth-onboarding-step-address
    lens: technical-writer
    severity: P2
    issue: "Placeholder 'House / flat / building number, street' is conversational and longer than the field allows visually. Helper text territory."
    evidence_excerpt: "House / flat / building number, street"
    recommendation: "Placeholder: 'Flat, building, street'. Helper text under input: 'Where should we deliver? Include floor or landmark if helpful.'"
    depends_on: null

  - finding_id: TW-010
    surface_id: web-auth-onboarding-step-address
    lens: technical-writer
    severity: P3
    issue: "'Landmark, area (optional)' — '(optional)' in placeholder is anti-pattern. Optional fields should drop the asterisk requirement marker; helper text shouldn't carry the '(optional)' tag."
    evidence_excerpt: "Landmark, area (optional)"
    recommendation: "Label: 'Landmark or area'. Remove '(optional)' qualifier — required indicator (asterisk) is the only signal of required vs optional. See style guide §4 Form labels."
    depends_on: null

  - finding_id: TW-011
    surface_id: web-auth-onboarding-step-basic
    lens: technical-writer
    severity: P2
    issue: "Title 'Basic Information' is Title Case; sentence case per style guide. Also generic — 'Basic Information' tells the user nothing about what step they're in."
    evidence_excerpt: "Basic Information / Tell us a bit about yourself"
    recommendation: "Title: 'About you'. Subtitle: 'A few details so we can personalise your home page.' Clarifies why we're asking."
    depends_on: null

  - finding_id: TW-012
    surface_id: web-auth-onboarding-step-basic
    lens: technical-writer
    severity: P3
    issue: "Placeholder example phone '+91 98765 43210' is good but inconsistent across surfaces (some show '+91 9876543210' without space). Style guide specifies '+91 98765 43210'."
    evidence_excerpt: "+91 98765 43210"
    recommendation: "Keep this format. Audit other surfaces (dp-auth-step1-emergency-phone, md-onb-014) and align to this format."
    depends_on: null

  - finding_id: TW-013
    surface_id: web-auth-onboarding-step-preferences
    lens: technical-writer
    severity: P2
    issue: "Five sub-section titles use Title Case: 'Dietary Preferences', 'Food Allergies', 'Favourite Cuisines', 'Spice Tolerance', 'Household Size'. Style guide requires sentence case."
    evidence_excerpt: "Dietary Preferences / Food Allergies / Favourite Cuisines / Spice Tolerance / Household Size"
    recommendation: "Sentence case all: 'Dietary preferences', 'Food allergies', 'Favourite cuisines', 'Spice tolerance', 'Household size'."
    depends_on: null

  - finding_id: TW-014
    surface_id: web-auth-onboarding-heading
    lens: technical-writer
    severity: P2
    issue: "'Complete Your Profile' Title Case; should be sentence case."
    evidence_excerpt: "Complete Your Profile"
    recommendation: "'Complete your profile'."
    depends_on: null

  - finding_id: TW-015
    surface_id: web-auth-onboarding-heading
    lens: technical-writer
    severity: P3
    issue: "Step progress 'Step {n} of 3' is fine. 'Skip' as a link is ambiguous — skip this step or skip onboarding entirely?"
    evidence_excerpt: "Step {n} of 3 / Skip"
    recommendation: "Replace 'Skip' with 'Skip for now' or 'Do this later' — clarifies intent. Style guide microcopy §4: buttons ≤3 words, but link text should be unambiguous."
    depends_on: null

  - finding_id: TW-016
    surface_id: web-auth-onboarding-steps
    lens: technical-writer
    severity: P3
    issue: "Step labels 'Basic Info / Your details / Preferences / Food & dietary / Address / Delivery location' — descriptions are clearer than titles. 'Basic Info' is filler. 'Address' + 'Delivery location' is redundant."
    evidence_excerpt: "Basic Info / Your details / Preferences / Food & dietary / Address / Delivery location"
    recommendation: "Titles: 'About you', 'Tastes', 'Address'. Descriptions: 'Name and phone', 'Cuisines you love', 'Where to deliver'. Tighter, no Title Case."
    depends_on: null

  # ===========================================================
  # VENDOR-PORTAL (chef-facing auth + onboarding wizard)
  # ===========================================================

  - finding_id: TW-017
    surface_id: vp-auth-forgot-redirect
    lens: technical-writer
    severity: P2
    issue: "Interim 'Redirecting to password reset...' uses three-dot ellipsis. Also no explanation of duration or fallback."
    evidence_excerpt: "Redirecting to password reset..."
    recommendation: "'Redirecting…' (single ellipsis char). If redirect can fail, add helper: 'If nothing happens in 5 seconds, click here.'"
    depends_on: null

  - finding_id: TW-018
    surface_id: vp-auth-login-hero-heading
    lens: technical-writer
    severity: P2
    issue: "Hero heading 'Grow your home kitchen business' is marketing-flavoured but acceptable. Subhead 'Manage menus, track orders, view earnings - all from one dashboard.' uses hyphen instead of em-dash. Sentence length 11 words — within chef range (5-12). Hyphen-spacing issue is the main flaw."
    evidence_excerpt: "Manage menus, track orders, view earnings - all from one dashboard."
    recommendation: "Replace ' - ' with em-dash ' — '. Final: 'Manage menus, track orders, view earnings — all from one dashboard.'"
    depends_on: null

  - finding_id: TW-019
    surface_id: vp-auth-login-features-list
    lens: technical-writer
    severity: P2
    issue: "Five features list uses ampersand inconsistently and runs long: 'Document verification & compliance' is vendor-side jargon. 'Customer reviews & ratings' uses ampersand mid-sentence — style guide prefers 'and' in prose."
    evidence_excerpt: "Real-time order management / Earnings & analytics dashboard / Menu management with categories / Customer reviews & ratings / Document verification & compliance"
    recommendation: "Use 'and' (not '&'): 'Earnings and analytics', 'Reviews and ratings', 'Verification and compliance'. Trim 'Real-time order management' → 'Live order tracking'."
    depends_on: null

  - finding_id: TW-020
    surface_id: vp-auth-login-google-btn
    lens: technical-writer
    severity: P3
    issue: "'Continue with Google' is 3 words, verb-first — within style. Cross-app consistency: web uses same phrase. OK as-is."
    evidence_excerpt: "Continue with Google"
    recommendation: "No change — flagging only for cross-app voice consistency check (vendor + web + delivery all use this exact phrase, which is correct)."
    depends_on: null

  - finding_id: TW-021
    surface_id: vp-auth-login-email-btn
    lens: technical-writer
    severity: P2
    issue: "'Sign in with Email' uses Title Case on 'Email'. Inconsistent with web ('Sign in with email')."
    evidence_excerpt: "Sign in with Email"
    recommendation: "'Sign in with email' — sentence case, matches web."
    depends_on: null

  - finding_id: TW-022
    surface_id: vp-auth-login-register-link
    lens: technical-writer
    severity: P3
    issue: "'Want to start selling? Register as a vendor' uses 'vendor' which style guide bans for customer-facing copy but allows chef-facing. However 'Register' is colder than the brand voice — also 'selling' is a vocabulary term for the chef portal (chefs sell). Acceptable, but could be warmer."
    evidence_excerpt: "Want to start selling? Register as a vendor"
    recommendation: "'New to Fe3dr? Register your kitchen.' Warmer, mirrors hero CTA on register page, replaces 'vendor' (cold) with 'kitchen' (concrete)."
    depends_on: null

  - finding_id: TW-023
    surface_id: vp-auth-register-hero-sub
    lens: technical-writer
    severity: P1
    issue: "AI-SLOP: 'Join thousands of home chefs earning with Fe3dr.' Unverified social-proof metric, exactly the pattern called out in the TW brief and Style Rule 5. No way to verify 'thousands'."
    evidence_excerpt: "Join thousands of home chefs earning with Fe3dr. Your kitchen, your recipes, your rules."
    recommendation: "Replace with concrete, verifiable: 'Start earning from your kitchen. Your recipes, your hours, your terms.' Removes unverified 'thousands' claim."
    depends_on: null

  - finding_id: TW-024
    surface_id: vp-auth-register-subheading
    lens: technical-writer
    severity: P3
    issue: "Subheading 'Create your vendor account to start selling home-cooked meals' uses 'vendor account' — chef-facing should say 'chef' (they self-identify as chefs per style guide vocabulary)."
    evidence_excerpt: "Create your vendor account to start selling home-cooked meals"
    recommendation: "'Create your chef account to start selling home-cooked meals.'"
    depends_on: null

  - finding_id: TW-025
    surface_id: vp-auth-register-email-btn
    lens: technical-writer
    severity: P2
    issue: "'Register with Email' Title Case on 'Email'. Inconsistent with 'Sign up with email' on web."
    evidence_excerpt: "Register with Email"
    recommendation: "'Sign up with email' — sentence case, and align verb across portals ('register' vs 'sign up' inconsistency)."
    depends_on: null

  - finding_id: TW-026
    surface_id: vp-onb-kitchen-title
    lens: technical-writer
    severity: P2
    issue: "'Kitchen Information' Title Case → sentence case per style guide. Also 'Information' is filler."
    evidence_excerpt: "Kitchen Information"
    recommendation: "'About your kitchen' (sentence case, more conversational)."
    depends_on: null

  - finding_id: TW-027
    surface_id: vp-onb-kitchen-blurb
    lens: technical-writer
    severity: P3
    issue: "'Tell customers about your kitchen and what makes your food special.' 11 words — within chef range. Word 'special' is vague."
    evidence_excerpt: "Tell customers about your kitchen and what makes your food special."
    recommendation: "'Tell customers about your kitchen and what you cook best.' Tighter, concrete."
    depends_on: null

  - finding_id: TW-028
    surface_id: vp-onb-ops-pricing-title
    lens: technical-writer
    severity: P2
    issue: "'Delivery & Pricing' uses ampersand and Title Case. Also section heading + helper merged. Style guide prefers 'and' over '&'."
    evidence_excerpt: "Delivery & Pricing / Set your preparation time, delivery range, and pricing. You can change these anytime."
    recommendation: "Heading: 'Delivery and pricing'. Helper kept (good copy — explains reversibility)."
    depends_on: null

  - finding_id: TW-029
    surface_id: vp-onb-personal-title
    lens: technical-writer
    severity: P2
    issue: "'Personal Details' Title Case. Field is KYC-collection but title doesn't signal that."
    evidence_excerpt: "Personal Details"
    recommendation: "'Personal details' (sentence case). Acceptable as-is for tone; KYC framing belongs in helper text (already present in vp-onb-personal-blurb)."
    depends_on: null

  - finding_id: TW-030
    surface_id: vp-onb-personal-fields
    lens: technical-writer
    severity: P2
    issue: "All four field labels use Title Case: 'Full Name', 'Phone Number', 'Email Address'. Style guide requires sentence case."
    evidence_excerpt: "Full Name / Phone Number / Email Address / Pre-filled from your login"
    recommendation: "'Full name', 'Phone number', 'Email address'. Helper 'Pre-filled from your login' is good."
    depends_on: null

  - finding_id: TW-031
    surface_id: vp-onb-personal-address-title
    lens: technical-writer
    severity: P2
    issue: "'Kitchen Address' Title Case."
    evidence_excerpt: "Kitchen Address"
    recommendation: "'Kitchen address'."
    depends_on: null

  - finding_id: TW-032
    surface_id: vp-onb-personal-address-blurb
    lens: technical-writer
    severity: P3
    issue: "Helper 'Where your kitchen is located. This is used for delivery radius and customer visibility.' is two sentences. Sentence 2 is passive ('is used for')."
    evidence_excerpt: "Where your kitchen is located. This is used for delivery radius and customer visibility."
    recommendation: "'We use this to calculate delivery range and show your kitchen to nearby customers.' Active voice, single sentence, plain English."
    depends_on: null

  - finding_id: TW-033
    surface_id: vp-onb-header-title
    lens: technical-writer
    severity: P2
    issue: "'Set Up Your Kitchen' Title Case."
    evidence_excerpt: "Set Up Your Kitchen"
    recommendation: "'Set up your kitchen'."
    depends_on: null

  - finding_id: TW-034
    surface_id: vp-onb-stepper-labels
    lens: technical-writer
    severity: P2
    issue: "Stepper labels mix Title Case ('Personal Info', 'Kitchen Details', 'Operations', 'Documents', 'Policies & Review'). Ampersand in last step."
    evidence_excerpt: "Personal Info / Kitchen Details / Operations / Documents / Policies & Review"
    recommendation: "'Personal info', 'Kitchen', 'Operations', 'Documents', 'Policies and review'. Sentence case + 'and'."
    depends_on: null

  # ===========================================================
  # DELIVERY-PORTAL (driver-facing — TELEGRAPHIC ≤4 words)
  # ===========================================================

  - finding_id: TW-035
    surface_id: dp-auth-mode-staff-cta
    lens: technical-writer
    severity: P2
    issue: "Button label 'I'm Staff' uses Title Case. Driver-facing should be glanceable but sentence case."
    evidence_excerpt: "I'm Staff"
    recommendation: "'I'm staff' (sentence case). Already telegraphic — within driver range."
    depends_on: null

  - finding_id: TW-036
    surface_id: dp-auth-mode-staff-sub
    lens: technical-writer
    severity: P2
    issue: "Sub 'Fleet managers & delivery operations' — ampersand."
    evidence_excerpt: "Fleet managers & delivery operations"
    recommendation: "'Fleet managers and ops staff'. Or simpler: 'Manage fleet and operations'."
    depends_on: null

  - finding_id: TW-037
    surface_id: dp-auth-email-button
    lens: technical-writer
    severity: P2
    issue: "'Sign in with Email' Title Case 'Email'."
    evidence_excerpt: "Sign in with Email"
    recommendation: "'Sign in with email' — sentence case, aligns with web."
    depends_on: null

  - finding_id: TW-038
    surface_id: dp-auth-mode-driver-cta
    lens: technical-writer
    severity: P2
    issue: "'I'm a Driver' Title Case 'Driver'."
    evidence_excerpt: "I'm a Driver"
    recommendation: "'I'm a driver' (sentence case)."
    depends_on: null

  - finding_id: TW-039
    surface_id: dp-auth-mode-driver-sub
    lens: technical-writer
    severity: P3
    issue: "'Login or sign up to deliver with Fe3dr' — 'Login' is banned (style guide: always 'Sign in')."
    evidence_excerpt: "Login or sign up to deliver with Fe3dr"
    recommendation: "'Sign in or sign up to deliver with Fe3dr.' Or shorter for driver: 'Sign in to start delivering.'"
    depends_on: null

  - finding_id: TW-040
    surface_id: dp-auth-login-subtitle-driver
    lens: technical-writer
    severity: P2
    issue: "'Driver login or sign up' — 'login' banned vocabulary, also drift from sibling subtitle which uses 'Sign in'."
    evidence_excerpt: "Driver login or sign up"
    recommendation: "'Driver sign in or sign up.'"
    depends_on: null

  - finding_id: TW-041
    surface_id: dp-auth-step1-h
    lens: technical-writer
    severity: P2
    issue: "'Personal Information' Title Case. Driver-facing — should be telegraphic AND sentence case."
    evidence_excerpt: "Personal Information"
    recommendation: "'About you' (sentence case, telegraphic, parallel with chef onboarding suggestion)."
    depends_on: null

  - finding_id: TW-042
    surface_id: dp-auth-step1-city
    lens: technical-writer
    severity: P2
    issue: "Label 'City *' includes asterisk inline with label. Style guide §4: label stays clean, asterisk goes on the field (visual indicator only)."
    evidence_excerpt: "City *"
    recommendation: "Label: 'City'. Render required marker as a visual asterisk via CSS, not inline in string. (This recommendation applies to ALL dp-auth-step1-*, step2, payout fields with '*' suffix.)"
    depends_on: null

  - finding_id: TW-043
    surface_id: dp-auth-step1-emergency-name
    lens: technical-writer
    severity: P2
    issue: "'Emergency Contact Name *' — Title Case + inline asterisk."
    evidence_excerpt: "Emergency Contact Name *"
    recommendation: "'Emergency contact name'. Strip asterisk from string."
    depends_on: null

  - finding_id: TW-044
    surface_id: dp-auth-step1-emergency-phone
    lens: technical-writer
    severity: P2
    issue: "'Emergency Phone *' Title Case + inline asterisk. Placeholder '+91 9876543210' missing space."
    evidence_excerpt: "Emergency Phone * / +91 9876543210"
    recommendation: "Label: 'Emergency phone'. Placeholder: '+91 98765 43210' (en-IN format per style guide §6)."
    depends_on: null

  - finding_id: TW-045
    surface_id: dp-auth-step1-dob
    lens: technical-writer
    severity: P2
    issue: "'Date of Birth' Title Case."
    evidence_excerpt: "Date of Birth"
    recommendation: "'Date of birth'."
    depends_on: null

  - finding_id: TW-046
    surface_id: dp-auth-step1-vehicle-type
    lens: technical-writer
    severity: P2
    issue: "'Vehicle Type *' Title Case + asterisk. Options 'Scooter/Motorcycle' uses slash — chip labels should be one word where possible."
    evidence_excerpt: "Vehicle Type * / Bicycle / Scooter/Motorcycle / Car"
    recommendation: "Label: 'Vehicle type'. Options: 'Bicycle', 'Scooter or motorcycle', 'Car'. Spell out 'or'."
    depends_on: null

  - finding_id: TW-047
    surface_id: dp-auth-step1-referral
    lens: technical-writer
    severity: P3
    issue: "Placeholder 'Enter referral code (optional)' duplicates the qualifier '(optional)' — see TW-010 rule. Also 'Validate' button label is acceptable but 'Check' is shorter for driver-glanceable."
    evidence_excerpt: "Referral Code + Enter referral code (optional) + Validate"
    recommendation: "Label: 'Referral code'. Placeholder: empty or 'e.g. SARAH123'. Button: 'Check'."
    depends_on: null

  - finding_id: TW-048
    surface_id: dp-auth-onboarding-step-labels
    lens: technical-writer
    severity: P2
    issue: "Step labels 'Personal Info / Vehicle Details / Documents / Plan / Review' — Title Case."
    evidence_excerpt: "Personal Info / Vehicle Details / Documents / Plan / Review"
    recommendation: "'Personal', 'Vehicle', 'Documents', 'Plan', 'Review'. Sentence case + shorter (driver glanceable)."
    depends_on: null

  - finding_id: TW-049
    surface_id: dp-auth-step2-h-vehicle
    lens: technical-writer
    severity: P2
    issue: "Heading 'Bicycle Details / Vehicle Details' + sub 'Tell us about your bicycle/vehicle' — Title Case + slash."
    evidence_excerpt: "Bicycle Details / Vehicle Details + Tell us about your bicycle/vehicle"
    recommendation: "Heading: 'Your bicycle' / 'Your vehicle'. Subhead: 'Tell us about your ride.'"
    depends_on: null

  - finding_id: TW-050
    surface_id: dp-auth-step2-carrier-q
    lens: technical-writer
    severity: P2
    issue: "'Can your bicycle carry a delivery box / bag? *' — 11 words, exceeds driver limit (≤12 OK but borderline). Slash + inline asterisk."
    evidence_excerpt: "Can your bicycle carry a delivery box / bag? *"
    recommendation: "'Can your bicycle carry a delivery bag?' — drop 'box' (redundant), tighter, no slash, no inline asterisk."
    depends_on: null

  - finding_id: TW-051
    surface_id: dp-auth-step2-carrier-warning
    lens: technical-writer
    severity: P2
    issue: "'You can still proceed, but you'll need to attach a carrier or use a delivery backpack before starting deliveries.' — 18 words. Driver limit ≤12 words/sentence."
    evidence_excerpt: "You can still proceed, but you'll need to attach a carrier or use a delivery backpack before starting deliveries."
    recommendation: "'Continue for now. Attach a carrier or use a delivery backpack before your first delivery.' Two short sentences."
    depends_on: null

  - finding_id: TW-052
    surface_id: dp-auth-step2-motor-fields
    lens: technical-writer
    severity: P2
    issue: "All four labels Title Case: 'Vehicle Make', 'Vehicle Model', 'Vehicle Year', 'Vehicle Color'. Also 'Vehicle' prefix is redundant within step labeled 'Vehicle details'."
    evidence_excerpt: "Vehicle Make / Vehicle Model / Vehicle Year / Vehicle Color"
    recommendation: "Drop 'Vehicle' prefix. Labels: 'Make', 'Model', 'Year', 'Colour'. Sentence case + UK spelling for consistency with mobile-customer (mc-onb-step3 uses 'favourite/personalised')."
    depends_on: null

  - finding_id: TW-053
    surface_id: dp-auth-step2-reg-number
    lens: technical-writer
    severity: P2
    issue: "'Vehicle Registration Number *' — Title Case + inline asterisk. Long label for driver."
    evidence_excerpt: "Vehicle Registration Number *"
    recommendation: "'Registration number'. Drop 'Vehicle' (redundant), drop asterisk from string."
    depends_on: null

  - finding_id: TW-054
    surface_id: dp-auth-step2-license
    lens: technical-writer
    severity: P2
    issue: "'Driving License Number *' — Title Case + asterisk."
    evidence_excerpt: "Driving License Number *"
    recommendation: "'Driving licence number' (UK spelling for en-IN per style guide). Drop asterisk."
    depends_on: null

  - finding_id: TW-055
    surface_id: dp-auth-step3-section-headers
    lens: technical-writer
    severity: P2
    issue: "Section headers Title Case: 'Personal Documents', 'Vehicle Documents', 'Vehicle Photos — All Angles', 'Optional'."
    evidence_excerpt: "Personal Documents / Vehicle Documents / Vehicle Photos — All Angles / Optional"
    recommendation: "'Personal documents', 'Vehicle documents', 'Vehicle photos', 'Optional uploads'. 'All Angles' is documentation-style — drop."
    depends_on: null

  - finding_id: TW-056
    surface_id: dp-auth-step3-h
    lens: technical-writer
    severity: P2
    issue: "'Documents & Photos' Title Case + ampersand. Sub 'Upload your documents and vehicle photos for verification' is fine."
    evidence_excerpt: "Documents & Photos + Upload your documents and vehicle photos for verification"
    recommendation: "Heading: 'Documents and photos'. Subhead OK as-is, but 12 words — borderline driver length."
    depends_on: null

  - finding_id: TW-057
    surface_id: dp-auth-step3-bicycle-hint
    lens: technical-writer
    severity: P2
    issue: "'Since you're using a bicycle, driving license, vehicle RC, and insurance are not required.' — 14 words, exceeds driver limit ≤12. Also passive 'are not required'."
    evidence_excerpt: "Since you're using a bicycle, driving license, vehicle RC, and insurance are not required."
    recommendation: "'You ride a bicycle — no licence, RC, or insurance needed.' 11 words, em-dash, active."
    depends_on: null

  - finding_id: TW-058
    surface_id: dp-auth-step3-doc-labels
    lens: technical-writer
    severity: P3
    issue: "Long doc list mixes Title Case. Helper for 'Number Plate (clear photo)' — qualifier in parentheses is helper-text pattern violation."
    evidence_excerpt: "Profile Photo / Driving License / Aadhaar Card / PAN Card / Vehicle RC / Insurance / Vehicle — Front View / Back View / Left Side / Right Side / Top View / Number Plate (clear photo) / Bicycle — Front/Back/Left/Right View / Police Verification"
    recommendation: "Sentence case all: 'Profile photo', 'Driving licence', 'Aadhaar card', 'PAN card', 'Vehicle RC', 'Insurance', 'Front view', 'Back view', etc. Move '(clear photo)' qualifier out of label to helper text under the upload slot: 'Make sure the number plate is readable.'"
    depends_on: null

  - finding_id: TW-059
    surface_id: dp-auth-step3-size-hints
    lens: technical-writer
    severity: P3
    issue: "Hints 'JPEG or PNG only, max 5MB' / 'JPEG, PNG, WebP, or PDF, max 10MB' — inventory note says NOT rendered. Defined but unused → dead string. Also format inconsistent (one excludes WebP/PDF, the other includes them)."
    evidence_excerpt: "JPEG or PNG only, max 5MB / JPEG, PNG, WebP, or PDF, max 10MB"
    recommendation: "Wire these hints into the UI as helper text under each upload slot. Standardise: 'JPEG, PNG up to 5 MB' / 'JPEG, PNG, PDF up to 10 MB'. Note: 'MB' with space per ISO."
    depends_on: null

  - finding_id: TW-060
    surface_id: dp-auth-step-payout-h
    lens: technical-writer
    severity: P2
    issue: "'Payout Details' Title Case. Sub 'How would you like to receive your earnings?' is fine (8 words, conversational for driver — borderline)."
    evidence_excerpt: "Payout Details + How would you like to receive your earnings?"
    recommendation: "'Payout details'. Sub: 'How should we pay you?' (5 words — telegraphic, driver-friendly)."
    depends_on: null

  - finding_id: TW-061
    surface_id: dp-auth-step-payout-methods
    lens: technical-writer
    severity: P2
    issue: "'Payout Method' Title Case. 'Bank Transfer' Title Case."
    evidence_excerpt: "Payout Method + Bank Transfer / UPI"
    recommendation: "'Payout method'. Options: 'Bank transfer', 'UPI'."
    depends_on: null

  - finding_id: TW-062
    surface_id: dp-auth-step-payout-bank-fields
    lens: technical-writer
    severity: P2
    issue: "All three labels Title Case + asterisk inline: 'Account Holder Name *', 'Account Number *', 'IFSC Code *'."
    evidence_excerpt: "Account Holder Name * / Account Number * / IFSC Code *"
    recommendation: "'Account holder name', 'Account number', 'IFSC code'. Strip asterisks."
    depends_on: null

  - finding_id: TW-063
    surface_id: dp-auth-step-payout-upi
    lens: technical-writer
    severity: P2
    issue: "'UPI ID *' inline asterisk."
    evidence_excerpt: "UPI ID *"
    recommendation: "'UPI ID'. Strip asterisk."
    depends_on: null

  - finding_id: TW-064
    surface_id: dp-auth-step4-h
    lens: technical-writer
    severity: P2
    issue: "'Choose Your Plan' Title Case. Subhead 'A small subscription to keep Fe3dr running — you keep every rupee you earn.' is 14 words — exceeds driver limit ≤12. 'rupee' is hardcoded (not i18n-ready)."
    evidence_excerpt: "Choose Your Plan + A small subscription to keep Fe3dr running — you keep every rupee you earn."
    recommendation: "Heading: 'Choose your plan'. Subhead: 'Small subscription. You keep every rupee.' (6 words.) 'Rupee' is acceptable for en-IN; flag for i18n later if Tamil/Hindi/global rollout."
    depends_on: null

  - finding_id: TW-065
    surface_id: dp-auth-step4-zero-commission
    lens: technical-writer
    severity: P1
    issue: "Block: 'We don't take any commission from your earnings… 100%… We're here to help you succeed, not to make profit from your hard work.' — 38 words in a single block. Driver limit ≤12 per sentence. Tone drifts marketing ('here to help you succeed') — banned brand-drift per Rule 5 'restraint over urgency'. Also 'not to make profit' is grammatically off ('make a profit')."
    evidence_excerpt: "We don't take any commission from your earnings + Every delivery fee and every tip goes directly to your account — 100%. We only charge a small subscription fee to keep the platform running and help you get more orders. We're here to help you succeed, not to make profit from your hard work."
    recommendation: "Replace with three short sentences: 'No commission. Every fee and tip is yours, 100%. We only charge a small subscription to run the platform.' 19 words total. Drops marketing flourish, fixes grammar, hits driver telegraphic style."
    depends_on: null

  - finding_id: TW-066
    surface_id: dp-auth-step4-trial
    lens: technical-writer
    severity: P3
    issue: "'{n}-day free trial' + 'Try everything free. No card needed.' — second clause is good, telegraphic. 'No card needed' is concrete trust signal."
    evidence_excerpt: "{n}-day free trial + Try everything free. No card needed."
    recommendation: "Keep. Minor: ensure pluralisation works for n=1 ('1-day free trial' is unidiomatic — consider '{n} days free' for plurals)."
    depends_on: null

  - finding_id: TW-067
    surface_id: dp-auth-step4-threshold
    lens: technical-writer
    severity: P2
    issue: "'No charge until you earn {amount}' + 'We only bill after you start earning well.' — 'earning well' is vague and editorial."
    evidence_excerpt: "No charge until you earn {amount} + We only bill after you start earning well."
    recommendation: "'No charge until you earn {amount}.' Drop the second sentence — it restates the first. If kept, replace with concrete: 'Billing starts after your first {amount} in earnings.'"
    depends_on: null

  - finding_id: TW-068
    surface_id: dp-auth-step4-billing-howto
    lens: technical-writer
    severity: P2
    issue: "'How billing works:' + 4-step ordered list, 50 words total. Trailing colon on heading violates style guide §4 (form labels no colons; same rule applies to in-content headings)."
    evidence_excerpt: "How billing works: + 4-step ordered list explaining trial, threshold, deduction, cancel-anytime"
    recommendation: "Heading: 'How billing works' (no colon). Confirm each list item is ≤12 words (driver limit). Cannot verify without reading the actual 4 items — flag for follow-up audit of those strings."
    depends_on: null

  - finding_id: TW-069
    surface_id: dp-auth-step4-secure-pay
    lens: technical-writer
    severity: P2
    issue: "'Payments are securely processed via {gateway}. You can change your plan or cancel anytime from your dashboard.' — 17 words, two sentences. Passive 'are processed'. Borderline length for driver."
    evidence_excerpt: "Payments are securely processed via {gateway}. You can change your plan or cancel anytime from your dashboard."
    recommendation: "'{gateway} handles payments securely. Change or cancel anytime from your dashboard.' Active voice, 11 words."
    depends_on: null

  - finding_id: TW-070
    surface_id: dp-auth-step4-submit
    lens: technical-writer
    severity: P2
    issue: "'Start with {interval} Plan' — 'Plan' Title Case + interpolated word: 'Start with Monthly Plan' reads odd. Loading state 'Saving...' uses three-dot ellipsis."
    evidence_excerpt: "Start with {interval} Plan / Saving..."
    recommendation: "'Start {interval} plan' or 'Begin {interval} plan'. Loading: 'Saving…' (Unicode ellipsis)."
    depends_on: null

  - finding_id: TW-071
    surface_id: dp-auth-step5-h
    lens: technical-writer
    severity: P2
    issue: "'Review & Submit' ampersand + Title Case. Sub 'Review your information before submitting' is fine."
    evidence_excerpt: "Review & Submit + Review your information before submitting"
    recommendation: "'Review and submit'. Sub OK."
    depends_on: null

  - finding_id: TW-072
    surface_id: dp-auth-step5-sections
    lens: technical-writer
    severity: P2
    issue: "Section headers Title Case: 'Personal Information', 'Vehicle Details', 'Documents ({n} uploaded)', 'Subscription Plan'."
    evidence_excerpt: "Personal Information / Vehicle Details / Documents ({n} uploaded) / Subscription Plan"
    recommendation: "'Personal info', 'Vehicle', 'Documents ({n})', 'Plan'. Sentence case + tighter, glanceable."
    depends_on: null

  - finding_id: TW-073
    surface_id: dp-auth-step5-edit
    lens: technical-writer
    severity: P3
    issue: "Mixed CTAs: 'Edit' / 'Change' for the same action on review rows. Inconsistency."
    evidence_excerpt: "Edit / Change"
    recommendation: "Pick one. 'Edit' is shorter and standard. Apply consistently to all review-row actions."
    depends_on: null

  - finding_id: TW-074
    surface_id: dp-auth-step5-plan-summary
    lens: technical-writer
    severity: P2
    issue: "'Plan selected. Payments handled securely via Razorpay.' + 'Billing starts only after you reach the minimum earnings threshold.' — Razorpay name hardcoded (inventory note flags this; gateway varies). Second sentence 11 words, OK. 'minimum earnings threshold' is jargon."
    evidence_excerpt: "Plan selected. Payments handled securely via Razorpay. + Billing starts only after you reach the minimum earnings threshold."
    recommendation: "'Plan selected. {gateway} handles payments securely.' (Use template var, not hardcoded 'Razorpay'.) Second sentence: 'Billing starts after you reach the earnings threshold.' Drop 'minimum' (redundant)."
    depends_on: null

  - finding_id: TW-075
    surface_id: dp-auth-step5-terms
    lens: technical-writer
    severity: P1
    issue: "'I agree to the Terms & Conditions, Privacy Policy, and confirm that all information provided is accurate and up to date.' — 19 words but more critically: 'Terms & Conditions' and 'Privacy Policy' are NOT linked per inventory note ('no link wired'). Style guide forbids click-here patterns AND requires proper link text. Legal-page text without links is non-compliant."
    evidence_excerpt: "I agree to the Terms & Conditions, Privacy Policy, and confirm that all information provided is accurate and up to date."
    recommendation: "Wire 'Terms and Conditions' and 'Privacy Policy' as proper links. Rewrite: 'I agree to the [Terms and Conditions] and [Privacy Policy]. I confirm the information above is accurate.' Two clear sentences."
    depends_on: null

  - finding_id: TW-076
    surface_id: dp-auth-step5-submit
    lens: technical-writer
    severity: P2
    issue: "'Submit Application' Title Case + 2 words OK. Loading 'Submitting...' three-dot ellipsis."
    evidence_excerpt: "Submitting... / Submit Application"
    recommendation: "'Submit application' (sentence case). Loading: 'Submitting…' (Unicode ellipsis)."
    depends_on: null

  - finding_id: TW-077
    surface_id: dp-auth-onboarding-header
    lens: technical-writer
    severity: P2
    issue: "'Become a Delivery Partner' Title Case. 'Delivery Partner' is style-guide-approved customer-facing term, but this is driver-facing (driver pages use 'Driver')."
    evidence_excerpt: "Become a Delivery Partner"
    recommendation: "'Become a driver' (sentence case, chef-/driver-self-identity)."
    depends_on: null

  - finding_id: TW-078
    surface_id: dp-auth-status-h
    lens: technical-writer
    severity: P2
    issue: "'Application Status' + 'Delivery Partner Onboarding' — both Title Case. Driver-facing — should be brief."
    evidence_excerpt: "Application Status + Delivery Partner Onboarding"
    recommendation: "'Application status' + 'Driver onboarding' (sentence case, driver self-identity)."
    depends_on: null

  - finding_id: TW-079
    surface_id: dp-auth-status-titles
    lens: technical-writer
    severity: P2
    issue: "Status titles Title Case: 'Application Needs Changes', 'Under Review', 'Application Submitted'."
    evidence_excerpt: "Application Needs Changes / Under Review / Application Submitted"
    recommendation: "'Application needs changes', 'Under review', 'Application submitted' (sentence case)."
    depends_on: null

  - finding_id: TW-080
    surface_id: dp-auth-status-descriptions
    lens: technical-writer
    severity: P2
    issue: "'Our team is currently reviewing your application. This usually takes 1-2 business days.' — 13 + 7 = 20 words for the longest. Inconsistent dash style: '1-2' should be en-dash '1–2'. Driver SLA wording mixes 'currently reviewing' (filler — 'currently' is redundant)."
    evidence_excerpt: "Our team is currently reviewing your application. This usually takes 1-2 business days. / Your application has been received and will be reviewed shortly."
    recommendation: "'We're reviewing your application. Usually 1–2 business days.' (en-dash for range per style guide). Third state: 'Application received. We'll review it shortly.' Tighter, active."
    depends_on: null

  - finding_id: TW-081
    surface_id: dp-auth-status-badges
    lens: technical-writer
    severity: P2
    issue: "Status badges Title Case: 'Rejected', 'In Review', 'Submitted'. Badge labels acceptable as Title Case across UI conventions, but inconsistent with sentence-case rule. Style guide is silent on badges specifically — flag for editorial decision."
    evidence_excerpt: "Rejected / In Review / Submitted"
    recommendation: "If badges follow Title Case as a system, document the exception. Otherwise: 'Rejected', 'In review', 'Submitted'."
    depends_on: null

  - finding_id: TW-082
    surface_id: dp-auth-status-fix-resubmit
    lens: technical-writer
    severity: P2
    issue: "'Fix & Resubmit' ampersand + Title Case."
    evidence_excerpt: "Fix & Resubmit"
    recommendation: "'Fix and resubmit' OR shorter 'Fix issues'."
    depends_on: null

  - finding_id: TW-083
    surface_id: dp-auth-status-auto-refresh
    lens: technical-writer
    severity: P3
    issue: "'Auto-refreshing every 30 seconds' is informational and acceptable. Inconsistent with mobile-delivery wording (md-onb-116 uses 'checks for updates automatically every 30 seconds')."
    evidence_excerpt: "Auto-refreshing every 30 seconds"
    recommendation: "Standardise across portals: 'Checks for updates every 30 seconds.' Apply same wording on mobile and web delivery status pages."
    depends_on: null

  # ===========================================================
  # ADMIN-PORTAL
  # ===========================================================

  - finding_id: TW-084
    surface_id: ap-auth-login-feature-list
    lens: technical-writer
    severity: P1
    issue: "Six-feature sales-style bullet list on an INTERNAL admin login page is voice-drift. Admin tone is 'neutral operator, direct'. Bullets read like marketing collateral: 'Platform analytics & insights', 'Revenue tracking & payouts', 'Content moderation tools'."
    evidence_excerpt: "User & role management; Chef verification & approvals; Order monitoring & refunds; Platform analytics & insights; Revenue tracking & payouts; Content moderation tools"
    recommendation: "Remove the feature list entirely from internal admin login OR replace with terse functional sentence: 'Admin tools for users, chefs, orders, and platform health.' (Internal users don't need a marketing pitch.)"
    depends_on: null

  - finding_id: TW-085
    surface_id: ap-auth-login-heading
    lens: technical-writer
    severity: P2
    issue: "'Admin Sign In' Title Case."
    evidence_excerpt: "Admin Sign In"
    recommendation: "'Admin sign in' (sentence case)."
    depends_on: null

  - finding_id: TW-086
    surface_id: ap-auth-login-instruction
    lens: technical-writer
    severity: P3
    issue: "'Access the administration dashboard with your internal credentials' is jargon ('administration dashboard', 'internal credentials'). Admin tone is direct, but this is procedural — could be plainer."
    evidence_excerpt: "Access the administration dashboard with your internal credentials"
    recommendation: "'Sign in with your internal credentials.' Drops the redundant 'access the administration dashboard' clause (sign-in implies it)."
    depends_on: null

  - finding_id: TW-087
    surface_id: ap-auth-totp-verify-heading
    lens: technical-writer
    severity: P2
    issue: "'Enter your 6-digit code' is acceptable. 4 words. Admin-direct."
    evidence_excerpt: "Enter your 6-digit code"
    recommendation: "Keep. Minor: ensure no Title Case drift in production."
    depends_on: null

  - finding_id: TW-088
    surface_id: ap-auth-totp-verify-help
    lens: technical-writer
    severity: P2
    issue: "'Open your authenticator app and enter the code shown for Fe3dr Admin.' — 13 words. Acceptable for admin. 'Fe3dr Admin' is brand name OK."
    evidence_excerpt: "Open your authenticator app and enter the code shown for Fe3dr Admin."
    recommendation: "Keep. Possibly tighten: 'Open your authenticator app and enter the Fe3dr Admin code.' (11 words.)"
    depends_on: null

  - finding_id: TW-089
    surface_id: ap-auth-totp-enroll-heading
    lens: technical-writer
    severity: P3
    issue: "'Set up 2FA to continue' acceptable but slightly awkward. Could be tighter."
    evidence_excerpt: "Set up 2FA to continue"
    recommendation: "'Set up 2FA' (3 words, drops 'to continue' which the workflow implies)."
    depends_on: null

  - finding_id: TW-090
    surface_id: ap-auth-totp-enroll-help
    lens: technical-writer
    severity: P2
    issue: "'Your organization requires 2FA for admins. Scan this QR with Google Authenticator (or similar), then enter the generated code.' — 22 words total, sentence 2 is 17 words. American spelling 'organization' inconsistent with British/Indian-English elsewhere."
    evidence_excerpt: "Your organization requires 2FA for admins. Scan this QR with Google Authenticator (or similar), then enter the generated code."
    recommendation: "'2FA is required for admins. Scan the QR with Google Authenticator or similar, then enter the code.' 17 words, 'organisation' if standardising on UK/IN. Removes 'this' (deictic ambiguity) — 'the QR' is clearer."
    depends_on: null

  - finding_id: TW-091
    surface_id: ap-secsettings-2fa-enroll
    lens: technical-writer
    severity: P2
    issue: "Multi-string concatenated entry: 'Scan with your authenticator; Or type this key manually:; Verify & enable; 6-digit code' — colons after labels violate §4. Ampersand in CTA."
    evidence_excerpt: "Scan with your authenticator; Or type this key manually:; Verify & enable; 6-digit code"
    recommendation: "'Scan with your authenticator.' / 'Or type this key manually' (no colon). CTA: 'Verify and enable' or shorter 'Enable 2FA'. Label '6-digit code' OK."
    depends_on: null

  - finding_id: TW-092
    surface_id: ap-secsettings-2fa-disable
    lens: technical-writer
    severity: P2
    issue: "'Disable 2FA; Requires your password and a current 6-digit code.; Password; Disabling...; Cancel' — confirmation pattern OK, but 'Disabling...' uses three-dot ellipsis. Helper sentence ends with period, but is mid-line in the inventory; verify rendering."
    evidence_excerpt: "Disable 2FA; Requires your password and a current 6-digit code.; Password; Disabling...; Cancel"
    recommendation: "Loading 'Disabling…' (Unicode ellipsis). Heading 'Disable 2FA' OK. Helper sentence 'Requires your password and a current 6-digit code' OK."
    depends_on: null

  # ===========================================================
  # MOBILE-CUSTOMER
  # ===========================================================

  - finding_id: TW-093
    surface_id: mc-onb-step2-subtitle
    lens: technical-writer
    severity: P3
    issue: "'Where should we deliver your orders?' is fine. 6 words. Customer-conversational. Note inconsistency with web: web onboarding step says 'Delivery Address' (cold), mobile says 'Your delivery address' (warmer)."
    evidence_excerpt: "Where should we deliver your orders?"
    recommendation: "Keep mobile copy; rewrite web (see TW-008) to match. Cross-app voice consistency."
    depends_on: null

  - finding_id: TW-094
    surface_id: mc-onb-step2-labels
    lens: technical-writer
    severity: P2
    issue: "Field labels 'Address / City / State / Pincode' — single words OK, but 'Pincode' could be 'PIN code' (style guide is silent but en-IN convention is 'PIN'). Web onboarding uses 'PIN code' (web-auth-onboarding-step-address: 'Enter PIN code'). Inconsistency."
    evidence_excerpt: "Address / City / State / Pincode"
    recommendation: "'PIN code' (uppercase PIN, space). Apply consistently across web + mobile."
    depends_on: null

  - finding_id: TW-095
    surface_id: mc-onb-step2-placeholders
    lens: technical-writer
    severity: P3
    issue: "'House no., street, area' uses abbreviation 'no.'. Helper text on cramped mobile is fine, but 'House number, street' is plain English."
    evidence_excerpt: "House no., street, area / Enter your city / Enter your state / 6-digit pincode"
    recommendation: "'Flat, building, street' (matches web TW-009 recommendation). '6-digit PIN code' (uppercase). 'Enter your city/state' placeholders restate label — remove."
    depends_on: null

  - finding_id: TW-096
    surface_id: mc-onb-step3-subtitle
    lens: technical-writer
    severity: P3
    issue: "'Select your favourite cuisines to get personalised recommendations.' — 9 words. UK spelling 'favourite/personalised' OK for en-IN. Inventory note flags potential drift with American spelling on web. Confirm consistency."
    evidence_excerpt: "Select your favourite cuisines to get personalised recommendations."
    recommendation: "Audit: web uses 'Favourite Cuisines' (Title Case but UK spelling) — consistent. Keep mobile copy. Final fix: sentence case label 'Favourite cuisines'."
    depends_on: null

  - finding_id: TW-097
    surface_id: mc-onb-step1-labels
    lens: technical-writer
    severity: P2
    issue: "'First Name / Last Name / Phone Number' Title Case. Web equivalent (TW-011 area) likely also Title Case — confirm cross-app."
    evidence_excerpt: "First Name / Last Name / Phone Number"
    recommendation: "'First name', 'Last name', 'Phone number' (sentence case)."
    depends_on: null

  - finding_id: TW-098
    surface_id: mc-onb-step1-placeholders
    lens: technical-writer
    severity: P3
    issue: "Placeholders 'Enter your first name / Enter your last name / 10-digit mobile number' — first two restate label."
    evidence_excerpt: "Enter your first name / Enter your last name / 10-digit mobile number"
    recommendation: "Drop 'Enter your...' placeholders (label is enough). Keep '10-digit mobile number' as format hint."
    depends_on: null

  # ===========================================================
  # MOBILE-VENDOR (chef-facing, mirrors vendor-portal)
  # ===========================================================

  - finding_id: TW-099
    surface_id: mv-onb-step1-title
    lens: technical-writer
    severity: P3
    issue: "Stack header titles 'Step 1 of 6' through 'Step 6 of 6' — verbose for mobile glanceable. Inventory note: 'verbose for glanceable mobile chef context but standard'. Drift risk with vendor-portal (5 steps vs 6 in mobile)."
    evidence_excerpt: "Step 1 of 6 / Step 2 of 6 / Step 3 of 6 / Step 4 of 6 / Step 5 of 6 / Step 6 of 6"
    recommendation: "Reconcile step count: mobile-vendor has 6, vendor-portal has 5 (Personal Info / Kitchen Details / Operations / Documents / Policies & Review). Confirm intentional and align stepper UX between web and mobile chef portals. Either both 5 or both 6 — currently INCONSISTENT."
    depends_on: null

  - finding_id: TW-100
    surface_id: mv-onb-docs-heading
    lens: technical-writer
    severity: P2
    issue: "'Documents' is fine as a screen heading. 'Upload your identity and FSSAI documents' subhead is good but 'identity' could be 'ID'."
    evidence_excerpt: "Documents / Upload your identity and FSSAI documents"
    recommendation: "Keep heading. Subhead: 'Upload your ID and FSSAI documents.' (Tighter, chef-functional.)"
    depends_on: null

  - finding_id: TW-101
    surface_id: mv-onb-docs-id-slot
    lens: technical-writer
    severity: P2
    issue: "'ID Proof' Title Case. Inconsistent with delivery's 'ID Proof' (also Title Case, also wrong)."
    evidence_excerpt: "ID Proof"
    recommendation: "'ID proof' (sentence case). Apply across mobile-vendor + mobile-delivery."
    depends_on: null

  - finding_id: TW-102
    surface_id: mv-onb-docs-fssai-slot
    lens: technical-writer
    severity: P2
    issue: "'FSSAI License' Title Case. FSSAI is acronym (uppercase OK). 'License' should be sentence case 'licence' (en-IN spelling)."
    evidence_excerpt: "FSSAI License"
    recommendation: "'FSSAI licence' (UK/IN spelling, sentence case)."
    depends_on: null

  - finding_id: TW-103
    surface_id: mv-onb-kitchen-heading
    lens: technical-writer
    severity: P2
    issue: "'Kitchen Details' Title Case. Sub 'Tell us about your kitchen' OK."
    evidence_excerpt: "Kitchen Details / Tell us about your kitchen"
    recommendation: "'Kitchen details' or 'About your kitchen' (matching web suggestion TW-026)."
    depends_on: null

  - finding_id: TW-104
    surface_id: mv-onb-kitchen-business-label
    lens: technical-writer
    severity: P2
    issue: "'Business Name *' Title Case + inline asterisk."
    evidence_excerpt: "Business Name *"
    recommendation: "'Business name'. Strip asterisk."
    depends_on: null

  - finding_id: TW-105
    surface_id: mv-onb-kitchen-business-ph
    lens: technical-writer
    severity: P3
    issue: "'Your kitchen / business name' — slash separates synonyms but reads awkward."
    evidence_excerpt: "Your kitchen / business name"
    recommendation: "'e.g. Meena's Kitchen' (concrete example placeholder, friendlier)."
    depends_on: null

  - finding_id: TW-106
    surface_id: mv-onb-kitchen-cuisines-label
    lens: technical-writer
    severity: P2
    issue: "'Cuisine Types *' Title Case + asterisk. 'Types' is filler."
    evidence_excerpt: "Cuisine Types *"
    recommendation: "'Cuisines'. Strip asterisk."
    depends_on: null

  - finding_id: TW-107
    surface_id: mv-onb-kitchen-desc-label
    lens: technical-writer
    severity: P2
    issue: "'Description *' is fine — single word — but Title Case + asterisk."
    evidence_excerpt: "Description *"
    recommendation: "'Description'. Strip asterisk."
    depends_on: null

  - finding_id: TW-108
    surface_id: mv-onb-kitchen-desc-ph
    lens: technical-writer
    severity: P2
    issue: "'Describe your kitchen, specialties, and cooking style (min 50 characters)' — 11 words, verbose for placeholder. Inventory note flags it. Chef may skim. '(min 50 characters)' belongs in helper text under input, not placeholder."
    evidence_excerpt: "Describe your kitchen, specialties, and cooking style (min 50 characters)"
    recommendation: "Placeholder: 'What do you cook? What makes your food special?' Helper text under input: 'At least 50 characters.'"
    depends_on: null

  - finding_id: TW-109
    surface_id: mv-onb-ops-heading
    lens: technical-writer
    severity: P3
    issue: "'Operations' single word — good. Sub 'Set your working hours and service details' 7 words, OK for chef."
    evidence_excerpt: "Operations / Set your working hours and service details"
    recommendation: "Keep. Minor: 'Set your hours and service area' (more concrete than 'service details')."
    depends_on: null

  - finding_id: TW-110
    surface_id: mv-onb-ops-hours-label
    lens: technical-writer
    severity: P2
    issue: "'Operating Hours' Title Case."
    evidence_excerpt: "Operating Hours"
    recommendation: "'Operating hours' or 'Hours' (drop redundant qualifier — already in 'Operations' section)."
    depends_on: null

  - finding_id: TW-111
    surface_id: mv-onb-ops-preptime-label
    lens: technical-writer
    severity: P2
    issue: "'Prep Time' Title Case."
    evidence_excerpt: "Prep Time"
    recommendation: "'Prep time'."
    depends_on: null

  - finding_id: TW-112
    surface_id: mv-onb-ops-radius-label
    lens: technical-writer
    severity: P2
    issue: "'Service Radius (km)' Title Case. '(km)' qualifier in label OK as unit indicator."
    evidence_excerpt: "Service Radius (km)"
    recommendation: "'Service radius (km)'."
    depends_on: null

  - finding_id: TW-113
    surface_id: mv-onb-ops-radius-ph
    lens: technical-writer
    severity: P3
    issue: "'1–50 km' uses en-dash for range — correct per style guide §6 (distances)."
    evidence_excerpt: "1–50 km"
    recommendation: "Keep. Good example of correct dash + space + unit formatting."
    depends_on: null

  - finding_id: TW-114
    surface_id: mv-onb-personal-heading
    lens: technical-writer
    severity: P2
    issue: "'Personal Information' Title Case. Matches mobile-delivery's same wrong pattern."
    evidence_excerpt: "Personal Information"
    recommendation: "'About you' (matches web/vendor-portal suggestion TW-011, TW-041)."
    depends_on: null

  - finding_id: TW-115
    surface_id: mv-onb-personal-fullname-label
    lens: technical-writer
    severity: P2
    issue: "'Full Name *' Title Case + inline asterisk."
    evidence_excerpt: "Full Name *"
    recommendation: "'Full name'. Strip asterisk."
    depends_on: null

  - finding_id: TW-116
    surface_id: mv-onb-personal-phone-label
    lens: technical-writer
    severity: P2
    issue: "'Phone Number *' Title Case + asterisk."
    evidence_excerpt: "Phone Number *"
    recommendation: "'Phone number'. Strip asterisk."
    depends_on: null

  - finding_id: TW-117
    surface_id: mv-onb-personal-email-label
    lens: technical-writer
    severity: P2
    issue: "'Email Address' Title Case. Also 'Address' is redundant — 'Email' is sufficient."
    evidence_excerpt: "Email Address"
    recommendation: "'Email'."
    depends_on: null

  - finding_id: TW-118
    surface_id: mv-onb-review-section-personal
    lens: technical-writer
    severity: P2
    issue: "'Personal Information' (review section label, inventory notes 'uppercase styled'). If styled uppercase via CSS, source string should still be sentence case."
    evidence_excerpt: "Personal Information"
    recommendation: "Source: 'Personal info'. CSS handles visual uppercase."
    depends_on: null

  - finding_id: TW-119
    surface_id: mv-onb-review-field-labels
    lens: technical-writer
    severity: P2
    issue: "Review labels mix Title Case: 'Full Name / Phone / Email / Business Name / Cuisines / Description / Open Days / Prep Time / Service Radius / ID Proof / FSSAI License / Terms Accepted / Cancellation Policy'."
    evidence_excerpt: "Full Name / Phone / Email / Business Name / Cuisines / Description / Open Days / Prep Time / Service Radius / ID Proof / FSSAI License / Terms Accepted / Cancellation Policy"
    recommendation: "All sentence case: 'Full name', 'Phone', 'Email', 'Business name', 'Cuisines', 'Description', 'Open days', 'Prep time', 'Service radius', 'ID proof', 'FSSAI licence', 'Terms accepted', 'Cancellation policy'."
    depends_on: null

  # ===========================================================
  # MOBILE-DELIVERY
  # ===========================================================

  - finding_id: TW-120
    surface_id: md-auth-002
    lens: technical-writer
    severity: P0
    issue: "User-facing error 'Google sign-in failed: no ID token' is a developer-shaped error (mentions internal 'ID token' concept). Style guide §4: errors follow 'what happened → what to do'. Currently no 'what to do'. If this surfaces uncaught, it's misleading copy."
    evidence_excerpt: "Google sign-in failed: no ID token"
    recommendation: "Catch in UI layer. Replace with: 'Google sign-in didn't complete. Try again or use email.' (Driver-friendly, action-oriented.)"
    depends_on: null

  - finding_id: TW-121
    surface_id: md-auth-003
    lens: technical-writer
    severity: P0
    issue: "'Apple sign-in failed: no identity token' — same developer-shaped error pattern as TW-120."
    evidence_excerpt: "Apple sign-in failed: no identity token"
    recommendation: "Catch in UI. Replace with: 'Apple sign-in didn't complete. Try again or use email.'"
    depends_on: null

  - finding_id: TW-122
    surface_id: md-auth-004
    lens: technical-writer
    severity: P1
    issue: "'Biometric authentication failed' — missing 'what to do' per §4 errors rule. Driver may not know what to try next."
    evidence_excerpt: "Biometric authentication failed"
    recommendation: "'Biometric sign-in failed. Try your password or face recognition.' (What happened → what to do.)"
    depends_on: null

  - finding_id: TW-123
    surface_id: md-auth-005
    lens: technical-writer
    severity: P2
    issue: "'No saved session found. Please log in with email.' — 'log in' is banned vocabulary (always 'Sign in')."
    evidence_excerpt: "No saved session found. Please log in with email."
    recommendation: "'No saved session. Please sign in with email.' Drops 'found' (redundant), uses 'sign in'."
    depends_on: null

  - finding_id: TW-124
    surface_id: md-onb-002
    lens: technical-writer
    severity: P2
    issue: "'Driver Onboarding' Title Case (header fallback title)."
    evidence_excerpt: "Driver Onboarding"
    recommendation: "'Driver onboarding'."
    depends_on: null

  - finding_id: TW-125
    surface_id: md-onb-042
    lens: technical-writer
    severity: P2
    issue: "'Upload Documents' Title Case."
    evidence_excerpt: "Upload Documents"
    recommendation: "'Upload documents'."
    depends_on: null

  - finding_id: TW-126
    surface_id: md-onb-043
    lens: technical-writer
    severity: P3
    issue: "'Please upload clear photos or PDFs of your documents' 9 words. 'Please' adds politeness but is filler in driver-glanceable. 'clear photos' is good (instruction)."
    evidence_excerpt: "Please upload clear photos or PDFs of your documents"
    recommendation: "'Upload clear photos or PDFs of your documents.' Drops 'Please' (driver-direct)."
    depends_on: null

  - finding_id: TW-127
    surface_id: md-onb-044
    lens: technical-writer
    severity: P2
    issue: "'Driving License' Title Case + American spelling. en-IN should be 'licence'."
    evidence_excerpt: "Driving License"
    recommendation: "'Driving licence' (UK/IN spelling, sentence case)."
    depends_on: null

  - finding_id: TW-128
    surface_id: md-onb-046
    lens: technical-writer
    severity: P3
    issue: "'Vehicle RC (optional)' — '(optional)' qualifier in label violates §4. Use no asterisk for optional fields."
    evidence_excerpt: "Vehicle RC (optional)"
    recommendation: "'Vehicle RC' (no qualifier). Required indicator is asterisk on required fields ONLY; optional fields have no marker."
    depends_on: null

  - finding_id: TW-129
    surface_id: md-onb-049
    lens: technical-writer
    severity: P2
    issue: "'Uploading...' three-dot ellipsis."
    evidence_excerpt: "Uploading..."
    recommendation: "'Uploading…' (Unicode ellipsis)."
    depends_on: null

  - finding_id: TW-130
    surface_id: md-onb-053
    lens: technical-writer
    severity: P2
    issue: "Alert title 'Permission Required' Title Case. Body 'Camera permission is required to capture documents.' uses passive voice and 7 words, driver-OK."
    evidence_excerpt: "Permission Required + Camera permission is required to capture documents."
    recommendation: "Title: 'Camera access needed'. Body: 'We need camera access to capture your documents.' Active voice."
    depends_on: null

  - finding_id: TW-131
    surface_id: md-onb-055
    lens: technical-writer
    severity: P2
    issue: "Alert title 'Upload Error' Title Case + generic. Body 'Upload failed. Please try again.' OK (what happened + what to do)."
    evidence_excerpt: "Upload Error + Upload failed. Please try again."
    recommendation: "Title: 'Upload failed'. Body: 'Try again, or check your connection.' (Adds diagnostic hint.)"
    depends_on: null

  - finding_id: TW-132
    surface_id: md-onb-057
    lens: technical-writer
    severity: P2
    issue: "Alert title 'Required Documents' Title Case. Body 'Please upload Driving License and ID Proof to continue.' — 'Driving License' should be 'driving licence', 'ID Proof' should be 'ID proof' (sentence case in running text)."
    evidence_excerpt: "Required Documents + Please upload Driving License and ID Proof to continue."
    recommendation: "Title: 'Missing documents'. Body: 'Upload your driving licence and ID proof to continue.'"
    depends_on: null

  - finding_id: TW-133
    surface_id: md-onb-059
    lens: technical-writer
    severity: P2
    issue: "'Payout Details' Title Case."
    evidence_excerpt: "Payout Details"
    recommendation: "'Payout details'."
    depends_on: null

  - finding_id: TW-134
    surface_id: md-onb-060
    lens: technical-writer
    severity: P3
    issue: "'Choose how you would like to receive your earnings' — 8 words. Driver-OK. Could be tighter."
    evidence_excerpt: "Choose how you would like to receive your earnings"
    recommendation: "'How should we pay you?' 5 words, telegraphic, conversational."
    depends_on: null

  - finding_id: TW-135
    surface_id: md-onb-061
    lens: technical-writer
    severity: P2
    issue: "'Bank Account' Title Case (tab label)."
    evidence_excerpt: "Bank Account"
    recommendation: "'Bank account' or just 'Bank' (tab label, telegraphic)."
    depends_on: null

  - finding_id: TW-136
    surface_id: md-onb-063
    lens: technical-writer
    severity: P2
    issue: "'Account Number' Title Case."
    evidence_excerpt: "Account Number"
    recommendation: "'Account number'."
    depends_on: null

  - finding_id: TW-137
    surface_id: md-onb-065
    lens: technical-writer
    severity: P2
    issue: "'Confirm Account Number' Title Case."
    evidence_excerpt: "Confirm Account Number"
    recommendation: "'Confirm account number'."
    depends_on: null

  - finding_id: TW-138
    surface_id: md-onb-070
    lens: technical-writer
    severity: P2
    issue: "'IFSC Code' Title Case. IFSC is acronym (uppercase OK), 'Code' should be sentence case."
    evidence_excerpt: "IFSC Code"
    recommendation: "'IFSC code'."
    depends_on: null

  - finding_id: TW-139
    surface_id: md-onb-073
    lens: technical-writer
    severity: P2
    issue: "'Bank Name' Title Case."
    evidence_excerpt: "Bank Name"
    recommendation: "'Bank name'."
    depends_on: null

  - finding_id: TW-140
    surface_id: md-onb-079
    lens: technical-writer
    severity: P2
    issue: "'Your payout details are encrypted and never stored on this device. They are used solely for processing your earnings.' — 19 words, sentence 1 is 12 words (driver borderline), sentence 2 is 7. Passive 'are encrypted' / 'are used'. 'solely' is formal."
    evidence_excerpt: "Your payout details are encrypted and never stored on this device. They are used solely for processing your earnings."
    recommendation: "'We encrypt your payout details. We only use them to pay you.' 13 words, active, plain English."
    depends_on: null

  - finding_id: TW-141
    surface_id: md-onb-081
    lens: technical-writer
    severity: P3
    issue: "'Failed to save payout details. Please try again.' — follows what happened → what to do pattern. OK, but 'Please' is filler for driver."
    evidence_excerpt: "Failed to save payout details. Please try again."
    recommendation: "'Couldn't save payout details. Try again.' Drops 'Please', conversational."
    depends_on: null

  - finding_id: TW-142
    surface_id: md-onb-109
    lens: technical-writer
    severity: P2
    issue: "'Application Not Approved' Title Case + euphemism. Inventory note: 'Rejected state heading'."
    evidence_excerpt: "Application Not Approved"
    recommendation: "'Application not approved' (sentence case). Note: aligns with delivery-portal 'Application Needs Changes' for re-applicable rejection (different state — confirm which applies here)."
    depends_on: null

  - finding_id: TW-143
    surface_id: md-onb-110
    lens: technical-writer
    severity: P2
    issue: "'Unfortunately your application was not approved at this time.' — 9 words. 'Unfortunately' is editorial sympathy ('No urgency tricks' Rule 1 — adjacent: no manufactured emotion). Passive 'was not approved'. Driver-glanceable should be direct."
    evidence_excerpt: "Unfortunately your application was not approved at this time."
    recommendation: "'Your application wasn't approved.' Direct, no euphemism, 5 words."
    depends_on: null

  - finding_id: TW-144
    surface_id: md-onb-113
    lens: technical-writer
    severity: P2
    issue: "'Application Under Review' Title Case."
    evidence_excerpt: "Application Under Review"
    recommendation: "'Application under review'."
    depends_on: null

  - finding_id: TW-145
    surface_id: md-onb-114
    lens: technical-writer
    severity: P2
    issue: "'Your application has been submitted and is being reviewed by our team.' — 12 words, driver borderline. Two passives ('has been submitted', 'is being reviewed')."
    evidence_excerpt: "Your application has been submitted and is being reviewed by our team."
    recommendation: "'We received your application. Our team is reviewing it.' 9 words, active."
    depends_on: null

  - finding_id: TW-146
    surface_id: md-onb-115
    lens: technical-writer
    severity: P2
    issue: "'Estimated review time: 24–48 hours' — colon after label (§4 form labels no colons; this is an info string). en-dash range is correct (✓ style guide §6)."
    evidence_excerpt: "Estimated review time: 24–48 hours"
    recommendation: "'Review time: 24–48 hours' — drop 'Estimated' (redundant). Alternative without colon: 'Review takes 24–48 hours.'"
    depends_on: null

  - finding_id: TW-147
    surface_id: md-onb-116
    lens: technical-writer
    severity: P2
    issue: "'We'll notify you once your application is approved. This page checks for updates automatically every 30 seconds.' — sentence 2 is 11 words, driver-OK. Sentence 1: 9 words. But inconsistent wording with delivery-portal (dp-auth-status-auto-refresh: 'Auto-refreshing every 30 seconds')."
    evidence_excerpt: "We'll notify you once your application is approved. This page checks for updates automatically every 30 seconds."
    recommendation: "Standardise: 'We'll notify you once approved. This page checks for updates every 30 seconds.' Aligns wording with delivery-portal (see TW-083)."
    depends_on: null

  - finding_id: TW-148
    surface_id: md-onb-001
    lens: technical-writer
    severity: P3
    issue: "Step header 'Step 1 of 6' is mobile-glanceable, driver-OK. Matches mobile-vendor pattern (TW-099) — 6 steps both apps. Internal consistency OK."
    evidence_excerpt: "Step 1 of 6"
    recommendation: "No change. Cross-link: confirm 6-step pattern matches delivery-portal web (dp-auth-onboarding-step-labels has 5 steps) — POTENTIAL DRIFT. Mobile delivery has 6 steps (Personal, Vehicle, Documents, Payout, Subscription, Review), web delivery-portal has 5 (Personal Info, Vehicle Details, Documents, Plan, Review). Web is missing Payout. CONFIRM intentional."
    depends_on: null

  - finding_id: TW-149
    surface_id: md-onb-003
    lens: technical-writer
    severity: P2
    issue: "'Personal Information' Title Case (matches mobile-vendor, vendor-portal, delivery-portal — same drift everywhere)."
    evidence_excerpt: "Personal Information"
    recommendation: "'About you' OR 'Personal info' — sentence case. Apply globally (see TW-041, TW-114, TW-118)."
    depends_on: null

  - finding_id: TW-150
    surface_id: md-onb-005
    lens: technical-writer
    severity: P2
    issue: "'City' OK single word. Other fields TW-151 onwards."
    evidence_excerpt: "City"
    recommendation: "Keep."
    depends_on: null

  - finding_id: TW-151
    surface_id: md-onb-008
    lens: technical-writer
    severity: P2
    issue: "'Vehicle Type' Title Case."
    evidence_excerpt: "Vehicle Type"
    recommendation: "'Vehicle type' or just 'Vehicle' (telegraphic)."
    depends_on: null

  - finding_id: TW-152
    surface_id: md-onb-010
    lens: technical-writer
    severity: P2
    issue: "'Emergency Contact Name' Title Case. Inconsistent with delivery-portal (which adds asterisk inline — TW-043)."
    evidence_excerpt: "Emergency Contact Name"
    recommendation: "'Emergency contact name' (sentence case). Asterisk via CSS, not inline."
    depends_on: null

  - finding_id: TW-153
    surface_id: md-onb-011
    lens: technical-writer
    severity: P3
    issue: "Placeholder 'Full name of emergency contact' restates label phrasing. Use example."
    evidence_excerpt: "Full name of emergency contact"
    recommendation: "Drop placeholder OR replace with example: 'e.g. Priya Sharma'."
    depends_on: null

  - finding_id: TW-154
    surface_id: md-onb-013
    lens: technical-writer
    severity: P2
    issue: "'Emergency Contact Phone' Title Case."
    evidence_excerpt: "Emergency Contact Phone"
    recommendation: "'Emergency contact phone'."
    depends_on: null

  - finding_id: TW-155
    surface_id: md-onb-018
    lens: technical-writer
    severity: P1
    issue: "DOB placeholder 'MM/DD/YYYY' is US format. INCONSISTENT with India-only validation elsewhere (zod patterns expect en-IN: '+91' phone, KA01AB1234 vehicle, etc.). Inventory note flags this drift."
    evidence_excerpt: "MM/DD/YYYY"
    recommendation: "'DD/MM/YYYY' (en-IN format). Align across all date fields. Style guide §6 implies en-IN locale primary."
    depends_on: null

  - finding_id: TW-156
    surface_id: md-onb-016
    lens: technical-writer
    severity: P2
    issue: "'Date of Birth' Title Case."
    evidence_excerpt: "Date of Birth"
    recommendation: "'Date of birth'."
    depends_on: null

  - finding_id: TW-157
    surface_id: md-onb-017
    lens: technical-writer
    severity: P3
    issue: "'(optional)' qualifier in label. See TW-010, TW-128. Optional fields should have NO qualifier."
    evidence_excerpt: "(optional)"
    recommendation: "Drop the qualifier. Required-only fields show asterisk; absence implies optional."
    depends_on: null

  - finding_id: TW-158
    surface_id: md-onb-020
    lens: technical-writer
    severity: P3
    issue: "'Failed to save personal info. Please try again.' — what happened + what to do. 'Please' is filler."
    evidence_excerpt: "Failed to save personal info. Please try again."
    recommendation: "'Couldn't save. Try again.' Driver-telegraphic."
    depends_on: null

  - finding_id: TW-159
    surface_id: md-onb-021
    lens: technical-writer
    severity: P2
    issue: "'Vehicle Details' Title Case."
    evidence_excerpt: "Vehicle Details"
    recommendation: "'Vehicle details' or 'Your vehicle'."
    depends_on: null

  - finding_id: TW-160
    surface_id: md-onb-022
    lens: technical-writer
    severity: P2
    issue: "'Vehicle type:' — trailing colon violates §4 form labels rule."
    evidence_excerpt: "Vehicle type:"
    recommendation: "'Vehicle type' (no colon)."
    depends_on: null

  - finding_id: TW-161
    surface_id: md-onb-023
    lens: technical-writer
    severity: P2
    issue: "'Vehicle Make' Title Case. Also 'Vehicle' prefix redundant in vehicle screen."
    evidence_excerpt: "Vehicle Make"
    recommendation: "'Make' (drops 'Vehicle' prefix, lower-case)."
    depends_on: null

  - finding_id: TW-162
    surface_id: md-onb-026
    lens: technical-writer
    severity: P2
    issue: "'Vehicle Model' Title Case + redundant prefix."
    evidence_excerpt: "Vehicle Model"
    recommendation: "'Model'."
    depends_on: null

  - finding_id: TW-163
    surface_id: md-onb-029
    lens: technical-writer
    severity: P2
    issue: "'Vehicle Year' Title Case + redundant prefix."
    evidence_excerpt: "Vehicle Year"
    recommendation: "'Year'."
    depends_on: null

  - finding_id: TW-164
    surface_id: md-onb-032
    lens: technical-writer
    severity: P2
    issue: "'Vehicle Color' Title Case + redundant prefix + American spelling."
    evidence_excerpt: "Vehicle Color"
    recommendation: "'Colour' (UK/IN spelling)."
    depends_on: null

  - finding_id: TW-165
    surface_id: md-onb-035
    lens: technical-writer
    severity: P2
    issue: "'Vehicle Registration Number' Title Case + verbose."
    evidence_excerpt: "Vehicle Registration Number"
    recommendation: "'Registration number'."
    depends_on: null

  - finding_id: TW-166
    surface_id: md-onb-038
    lens: technical-writer
    severity: P2
    issue: "'Driving License Number' Title Case + American spelling 'License'."
    evidence_excerpt: "Driving License Number"
    recommendation: "'Driving licence number' (UK/IN spelling, sentence case)."
    depends_on: null

  - finding_id: TW-167
    surface_id: md-onb-041
    lens: technical-writer
    severity: P3
    issue: "'Failed to save vehicle details. Please try again.' — 'Please' filler."
    evidence_excerpt: "Failed to save vehicle details. Please try again."
    recommendation: "'Couldn't save vehicle details. Try again.' Driver-direct."
    depends_on: null

  - finding_id: TW-168
    surface_id: md-onb-082
    lens: technical-writer
    severity: P2
    issue: "'Choose Your Plan' Title Case."
    evidence_excerpt: "Choose Your Plan"
    recommendation: "'Choose your plan'."
    depends_on: null

  - finding_id: TW-169
    surface_id: md-onb-083
    lens: technical-writer
    severity: P2
    issue: "'Select a subscription plan that works best for you' — 8 words. Driver-OK. 'that works best for you' is conversational filler."
    evidence_excerpt: "Select a subscription plan that works best for you"
    recommendation: "'Pick the plan that fits.' 5 words. Direct."
    depends_on: null

  - finding_id: TW-170
    surface_id: md-onb-086
    lens: technical-writer
    severity: P3
    issue: "'Up to {n} deliveries/month' uses '/month' suffix — fine. Pluralisation when n=1 unclear: 'Up to 1 deliveries/month'?"
    evidence_excerpt: "Up to {n} deliveries/month"
    recommendation: "Use ICU MessageFormat: '{n, plural, one {Up to # delivery/month} other {Up to # deliveries/month}}'. Style guide §6 plural rule."
    depends_on: null

  - finding_id: TW-171
    surface_id: md-onb-088
    lens: technical-writer
    severity: P2
    issue: "Alert title 'Select a Plan' Title Case + identical to button text — confusing. Body 'Please select a subscription plan to continue.' has 'Please' filler."
    evidence_excerpt: "Select a Plan + Please select a subscription plan to continue."
    recommendation: "Title: 'Choose a plan'. Body: 'Pick a plan to continue.'"
    depends_on: null

  - finding_id: TW-172
    surface_id: md-onb-093
    lens: technical-writer
    severity: P2
    issue: "'Review Your Application' Title Case."
    evidence_excerpt: "Review Your Application"
    recommendation: "'Review your application'."
    depends_on: null

  - finding_id: TW-173
    surface_id: md-onb-094
    lens: technical-writer
    severity: P3
    issue: "'Please review your details before submitting' — 6 words, OK. 'Please' is filler."
    evidence_excerpt: "Please review your details before submitting"
    recommendation: "'Review your details before submitting.' Drops 'Please'."
    depends_on: null

  - finding_id: TW-174
    surface_id: md-onb-099
    lens: technical-writer
    severity: P3
    issue: "Summary row value '{n}/{total} required + RC' is cryptic. 'RC' is acronym, needs context for drivers unfamiliar."
    evidence_excerpt: "{n}/{total} required + RC"
    recommendation: "'{n} of {total} required, plus RC'. Spells out 'of', adds 'plus' for clarity."
    depends_on: null

  - finding_id: TW-175
    surface_id: md-onb-103
    lens: technical-writer
    severity: P1
    issue: "'I accept the Terms of Service and Privacy Policy' — link wiring not confirmed in inventory. If 'Terms of Service' and 'Privacy Policy' are NOT linked, same problem as TW-075 (delivery-portal)."
    evidence_excerpt: "I accept the Terms of Service and Privacy Policy"
    recommendation: "Confirm links are wired. Wording also drifts from delivery-portal ('Terms & Conditions, Privacy Policy'). Align across portals: 'I agree to the [Terms and Conditions] and [Privacy Policy].'"
    depends_on: null

  - finding_id: TW-176
    surface_id: md-onb-104
    lens: technical-writer
    severity: P2
    issue: "Alert title 'Terms Required' Title Case."
    evidence_excerpt: "Terms Required"
    recommendation: "'Accept terms to continue' (more specific). Or 'Terms required' (sentence case)."
    depends_on: null

  - finding_id: TW-177
    surface_id: md-onb-107
    lens: technical-writer
    severity: P2
    issue: "Alert title 'Submission Error' Title Case + generic."
    evidence_excerpt: "Submission Error"
    recommendation: "'Submission failed' (what happened, not just 'error')."
    depends_on: null

  - finding_id: TW-178
    surface_id: md-onb-095
    lens: technical-writer
    severity: P2
    issue: "Review section headers Title Case: 'Personal Info', 'Vehicle Details', 'Documents', 'Documents Uploaded', 'Payout Method', 'Subscription Plan', 'Selected Plan'."
    evidence_excerpt: "Personal Info / Vehicle Details / Documents / Documents Uploaded / Payout Method / Subscription Plan / Selected Plan"
    recommendation: "All sentence case: 'Personal info', 'Vehicle details', 'Documents', 'Documents uploaded', 'Payout method', 'Subscription plan', 'Selected plan'."
    depends_on: null

  # ===========================================================
  # CROSS-CUTTING / CONSISTENCY
  # ===========================================================

  - finding_id: TW-179
    surface_id: dp-auth-step-payout-h
    lens: technical-writer
    severity: P1
    issue: "ORPHAN component: inventory notes the payout step file is present but 'not wired into OnboardingPage; orphan component'. Driver-onboarding web has no payout step in inventory list — mobile-delivery has full payout flow (md-onb-059 → md-onb-081). This is feature parity drift between web and mobile delivery, not just copy."
    evidence_excerpt: "dp-auth-step-payout-h — file present but not wired into OnboardingPage; orphan component"
    recommendation: "Surface this to product: mobile-delivery collects payout details during onboarding, web delivery-portal doesn't. Confirm intentional or wire the orphan component. From TW lens: if wired, all payout copy from delivery-portal needs to match mobile-delivery wording."
    depends_on: null

  - finding_id: TW-180
    surface_id: md-onb-018
    lens: technical-writer
    severity: P2
    issue: "CROSS-APP DATE FORMAT DRIFT: mobile-delivery uses 'MM/DD/YYYY' (US) for DOB placeholder; vendor-portal/web use no explicit format hint; mobile-vendor has no DOB. en-IN convention is DD/MM/YYYY."
    evidence_excerpt: "MM/DD/YYYY"
    recommendation: "Standardise: 'DD/MM/YYYY' on all DOB placeholders across portals. Document the decision."
    depends_on: null

  - finding_id: TW-181
    surface_id: vp-auth-register-hero-sub
    lens: technical-writer
    severity: P1
    issue: "AI-SLOP IN ENTRY SURFACES — comprehensive scan: vp-auth-register-hero-sub ('Join thousands') flagged TW-023. Cross-checking ap-auth-login-feature-list (TW-084 — marketing tone in admin), dp-auth-step4-zero-commission (TW-065 — 'here to help you succeed'), and dp-auth-step4-h ('keep every rupee you earn'). All hit unverified-metric or marketing-flourish anti-patterns. Pattern: Fe3dr marketing copy is leaking into functional surfaces."
    evidence_excerpt: "Join thousands of home chefs… / We're here to help you succeed, not to make profit from your hard work… / 100%…"
    recommendation: "Audit ALL hero/subhead strings across auth-onboarding for unverifiable claims and editorial sympathy. Replace with concrete, falsifiable copy. See individual findings TW-023, TW-065, TW-084 for specific rewrites."
    depends_on: null

  - finding_id: TW-182
    surface_id: dp-auth-onboarding-step-labels
    lens: technical-writer
    severity: P1
    issue: "STEPPER COUNT DRIFT: delivery-portal web has 5 steps (Personal Info / Vehicle Details / Documents / Plan / Review); mobile-delivery has 6 steps (adds Payout — see md-onb-001 + md-onb-059). Same product, two different onboarding flows. This is a TW lens flag because step labels and progress copy ('Step 1 of 5' vs 'Step 1 of 6') are user-facing and confusing for drivers switching devices."
    evidence_excerpt: "Personal Info / Vehicle Details / Documents / Plan / Review (web, 5 steps) vs Personal Info / Vehicle / Documents / Payout / Subscription / Review (mobile, 6 steps)"
    recommendation: "Reconcile: either wire payout step on web (TW-179) or remove from mobile. Step copy must match across portals. Same for vendor-portal (5 steps) vs mobile-vendor (6 steps) — TW-099."
    depends_on: null

  - finding_id: TW-183
    surface_id: web-auth-onboarding-heading
    lens: technical-writer
    severity: P2
    issue: "CUSTOMER ONBOARDING STEP COUNT: web shows 'Step {n} of 3', mobile-customer shows 'Step 1 of 3' (matches). OK consistency. Flagging only for completeness — customer parity holds."
    evidence_excerpt: "Step {n} of 3 (web) / Step 1 of 3 (mobile)"
    recommendation: "No change. Cross-app customer onboarding step count is consistent at 3 steps."
    depends_on: null

  - finding_id: TW-184
    surface_id: mc-auth-register-host
    lens: technical-writer
    severity: P3
    issue: "Mobile-customer register screen uses shared RegisterScreen from @homechef/mobile-shared with no local strings. Inventory notes 'flag for cross-app voice audit'. TW lens cannot audit shared screen body copy without separate inventory entry."
    evidence_excerpt: "(shared RegisterScreen — no local strings)"
    recommendation: "Request inventory expansion: extract all strings from @homechef/mobile-shared/screens/RegisterScreen and add to inventory as separate surface_ids. Cannot audit what isn't visible."
    depends_on: null

  - finding_id: TW-185
    surface_id: mc-auth-login-title
    lens: technical-writer
    severity: P3
    issue: "Mobile-customer login title 'Welcome back' is consistent across portals (web, vendor-portal, mobile-vendor, mobile-delivery all use 'Welcome back'). Good cross-app consistency."
    evidence_excerpt: "Welcome back"
    recommendation: "No change. Confirm same shared LoginScreen is in use to prevent future drift."
    depends_on: null

  - finding_id: TW-186
    surface_id: dp-auth-login-title
    lens: technical-writer
    severity: P3
    issue: "Brand wordmark 'Fe3dr Delivery' is consistent with admin-portal brand block. Brand wordmark OK."
    evidence_excerpt: "Fe3dr Delivery"
    recommendation: "No change. Brand wordmark formatting consistent."
    depends_on: null

  - finding_id: TW-187
    surface_id: dp-auth-login-subtitle-choose
    lens: technical-writer
    severity: P3
    issue: "'Sign in to get started' — inventory note V: '\"Sign in\" alone is enough'. 5 words; could be tighter for driver context."
    evidence_excerpt: "Sign in to get started"
    recommendation: "'Sign in.' Or keep current — both fine. Inventory note suggests dropping 'to get started'."
    depends_on: null

  - finding_id: TW-188
    surface_id: dp-auth-login-subtitle-staff
    lens: technical-writer
    severity: P3
    issue: "'Staff & fleet manager login' — ampersand + 'login' banned vocabulary."
    evidence_excerpt: "Staff & fleet manager login"
    recommendation: "'Staff and fleet manager sign-in.' Drops ampersand and banned 'login'."
    depends_on: null

  - finding_id: TW-189
    surface_id: ap-auth-login-brand-heading
    lens: technical-writer
    severity: P3
    issue: "'Platform Administration' is Title Case + abstract. Admin tone is direct/precise."
    evidence_excerpt: "Platform Administration"
    recommendation: "'Platform administration' (sentence case)."
    depends_on: null

  - finding_id: TW-190
    surface_id: ap-auth-login-brand-sub
    lens: technical-writer
    severity: P3
    issue: "'Manage users, chefs, orders, and analytics for the Fe3dr platform.' — 10 words. Admin-OK."
    evidence_excerpt: "Manage users, chefs, orders, and analytics for the Fe3dr platform."
    recommendation: "Keep. Mild: 'Manage users, chefs, orders, and analytics across Fe3dr.' (Drops 'for the…platform' which is filler.)"
    depends_on: null

  - finding_id: TW-191
    surface_id: ap-auth-login-portal-label
    lens: technical-writer
    severity: P3
    issue: "'Admin Portal' Title Case. Logo sub-label."
    evidence_excerpt: "Admin Portal"
    recommendation: "'Admin portal' (sentence case)."
    depends_on: null

  - finding_id: TW-192
    surface_id: md-onb-080
    lens: technical-writer
    severity: P3
    issue: "Inventory annotation says 'Select a Plan' alert appears in subscription screen but is filed under payout.tsx surface — possible inventory misclassification. TW lens: 'Select a Plan' is Title Case."
    evidence_excerpt: "Select a Plan"
    recommendation: "'Choose a plan' (sentence case). Confirm which screen it actually renders on."
    depends_on: null
```

## Legal findings

```yaml
# Legal lens findings — auth-onboarding category
# Jurisdiction: India + generic best-practice
# Auditor disclaimer: this is a content audit, NOT legal advice. Every finding
# requires lawyer review before binding text is drafted or shipped.

# =============================================================================
# WEB CUSTOMER — apps/web/src/features/auth/
# =============================================================================

findings:
  - finding_id: LEG-001
    surface_id: web-auth-register-form-fields
    lens: legal
    severity: P0
    issue: "Signup form collects personal data (name, email, password) with no DPDP §6 consent notice at the data-collection moment"
    evidence_excerpt: "First name / Last name / Email / Password / Confirm password / Min. 8 characters / Re-enter password"
    recommendation: "Add an explicit, granular consent notice immediately above the submit button: purposes of processing, identity of data fiduciary (legal entity name), retention period, and link to grievance officer. The post-submit footer 'By signing up, you agree to our Terms and Privacy Policy' is bundled blanket consent and does not satisfy DPDP §6 specificity requirements."
    citation: "DPDP Act 2023 §6 (consent must be free, specific, informed, unconditional, unambiguous, with clear affirmative action) + §5 (notice requirements)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-002
    surface_id: web-auth-register-submit
    lens: legal
    severity: P0
    issue: "Account creation submit button has no checkbox-gated affirmative-action consent; clicking 'Create account' is treated as implied consent"
    evidence_excerpt: "Create account / Creating account..."
    recommendation: "Per DPDP §6(1) consent must be by 'clear affirmative action'. A button click bundled with T&C acceptance after the fact does not satisfy this. Add at minimum an unchecked checkbox: 'I agree to the Terms of Service and Privacy Policy' that gates submission. Separate marketing-communications consent checkbox required if any opt-in to non-transactional email."
    citation: "DPDP Act 2023 §6(1); best-practice unbundled consent"
    depends_on: "needs lawyer review"

  - finding_id: LEG-003
    surface_id: web-auth-register-form-fields
    lens: legal
    severity: P0
    issue: "No children's-data screening at signup; platform has no age gate"
    evidence_excerpt: "First name / Last name / Email / Password / Confirm password"
    recommendation: "Per DPDP §9 (renumbered §7 in some drafts), data fiduciary must obtain verifiable parental consent before processing personal data of a child (<18). Add either (a) explicit age confirmation checkbox 'I confirm I am 18 or older' or (b) DOB collection at signup with backend rejection for <18. Food delivery is not exempted from §9."
    citation: "DPDP Act 2023 §9 (children's data) + Rule 10 of draft DPDP Rules 2025"
    depends_on: "needs lawyer review"

  - finding_id: LEG-004
    surface_id: web-auth-register-heading
    lens: legal
    severity: P1
    issue: "Signup heading does not identify the legal entity (data fiduciary) collecting the data"
    evidence_excerpt: "Create your account / Already have an account? Sign in"
    recommendation: "Per DPDP §5(i), notice must identify the data fiduciary. Add legal entity name (e.g., 'Fe3dr is operated by [Pvt Ltd entity name], registered office: [address]') either inline or via a 'Who we are' link in the consent notice."
    citation: "DPDP Act 2023 §5(i)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-005
    surface_id: web-auth-register-heading
    lens: legal
    severity: P0
    issue: "No grievance-officer contact disclosed at signup or in the visible consent footer"
    evidence_excerpt: "Create your account"
    recommendation: "Per DPDP §13, data fiduciary must publish grievance officer name and contact and make them accessible. Signup is the first contact; a 'Contact our Grievance Officer' link belongs in the consent notice or in the privacy policy linked from this screen — and lawyer must confirm the linked privacy policy actually lists the officer."
    citation: "DPDP Act 2023 §13(1)-(2)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-006
    surface_id: web-auth-register-heading
    lens: legal
    severity: P0
    issue: "No disclosure of user rights (access, correction, erasure, nomination, grievance redressal) at signup"
    evidence_excerpt: "Create your account / Already have an account? Sign in"
    recommendation: "Per DPDP §5(iii)–(iv), notice at consent must inform user of their rights (§11 access/correction, §12 erasure, §13 grievance) and the procedure to exercise them. Add a one-line summary with a link, or link the privacy policy that lawyer has reviewed for §11/12/13 compliance."
    citation: "DPDP Act 2023 §5(iii), §11, §12, §13"
    depends_on: "needs lawyer review"

  - finding_id: LEG-007
    surface_id: web-auth-register-form-fields
    lens: legal
    severity: P1
    issue: "Password collected and stored; no statement of security practices (encryption at rest/transit, bcrypt/argon2 hashing, breach notification commitment)"
    evidence_excerpt: "Password / Min. 8 characters"
    recommendation: "DPDP §8(5) requires data fiduciary to implement 'reasonable security safeguards'. Privacy policy linked from signup must describe these. Password minimum 8 chars is weak — consider documenting NIST SP 800-63B alignment (length over complexity, breach-list check) for binding policy text. Lawyer to confirm policy meets §8(5) standard."
    citation: "DPDP Act 2023 §8(5); NIST SP 800-63B best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-008
    surface_id: web-auth-register-social
    lens: legal
    severity: P0
    issue: "OAuth signup (Google/Facebook) does not disclose what data is shared with third-party identity provider before user clicks"
    evidence_excerpt: "Sign up with Google / Sign up with Facebook"
    recommendation: "Per DPDP §10 (if Fe3dr is significant data fiduciary) and §11 disclosure, user must know which fields are pulled from Google/Facebook (email, name, profile photo, locale, etc.) before authorising. Add a tooltip or 'What does Google share?' link below each OAuth button. Note also: Meta/Facebook OAuth has separate consent flow concerns — lawyer to confirm Fe3dr's data-handling addendum with each provider is current."
    citation: "DPDP Act 2023 §5, §10; best-practice OAuth disclosure"
    depends_on: "needs lawyer review"

  - finding_id: LEG-009
    surface_id: web-auth-register-social
    lens: legal
    severity: P1
    issue: "OAuth buttons identical for sign-up and login — different consent expectations; first-time OAuth signup needs DPDP consent capture, returning login does not"
    evidence_excerpt: "Sign up with Google / Sign up with Facebook (signup) vs Continue with Google (login)"
    recommendation: "On first OAuth signup, after callback, force user through a one-time consent screen capturing DPDP §6 affirmative action (purposes, retention, rights) before account is created. Currently the OAuth callback presumably creates the account silently with no consent record. Lawyer to confirm consent log/audit trail."
    citation: "DPDP Act 2023 §6(2) (record of consent)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-010
    surface_id: web-auth-login-social
    lens: legal
    severity: P2
    issue: "Login OAuth flow has no T&C/Privacy disclosure visible (unlike register page footer)"
    evidence_excerpt: "Continue with Google / Continue with Facebook"
    recommendation: "Even on returning-user login, jurisdictional best-practice is to keep T&C/Privacy links visible (e.g., footer) for users who may want to review before re-authenticating. Currently invisible on web-auth-login surface. Lawyer to confirm minimum disclosure requirement on returning-login screens."
    citation: "best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-011
    surface_id: web-auth-login-email-fields
    lens: legal
    severity: P1
    issue: "'Forgot password?' flow not inventoried for legal exposure — password reset emails carry security/notification obligations"
    evidence_excerpt: "Email / Password / Forgot password? / you@example.com / Enter your password"
    recommendation: "Password reset notification emails are a §8(6) breach-adjacent security control; if reset is triggered without user knowledge, DPDP arguably requires notice. Lawyer to confirm reset-email template text discloses (a) who initiated, (b) IP/device, (c) what to do if user didn't request."
    citation: "DPDP Act 2023 §8(5)–(6); best-practice account-security notification"
    depends_on: "needs lawyer review"

  - finding_id: LEG-012
    surface_id: web-auth-onboarding-step-basic
    lens: legal
    severity: P0
    issue: "Phone number collected post-signup without separate purpose-specific consent (transactional SMS vs marketing SMS)"
    evidence_excerpt: "Basic Information / Tell us a bit about yourself / Enter your first name / Enter your last name / +91 98765 43210"
    recommendation: "TRAI commercial-communications rules require explicit consent for promotional SMS/calls, separate from transactional. Per DPDP §6 'specific' consent requirement, bundling phone collection with general signup-consent is non-compliant. Add granular checkbox: 'Send me order updates by SMS (required)' (cannot legally make optional for transactional) and 'Send me offers and recommendations by SMS' (must be unchecked, opt-in)."
    citation: "DPDP Act 2023 §6; TRAI Telecom Commercial Communications Customer Preference Regulations (TCCCPR) 2018"
    depends_on: "needs lawyer review"

  - finding_id: LEG-013
    surface_id: web-auth-onboarding-step-address
    lens: legal
    severity: P1
    issue: "Delivery address collected with no retention statement or purpose limitation"
    evidence_excerpt: "Delivery Address / Enter PIN code / House / flat / building number, street / Landmark, area (optional)"
    recommendation: "Per DPDP §8(7) (storage limitation principle), specify retention period for address data. Per §5(ii), specify the purpose (delivery only? or also profiling/marketing recommendations?). Add helper text under section title: 'We use this to route deliveries. We keep it for [X period] after your last order.' Lawyer to confirm retention period aligns with Income Tax record-keeping (8 yrs) and FSSAI traceability (3 yrs)."
    citation: "DPDP Act 2023 §5(ii), §8(7); FSSAI Regulations §2.1.13 (traceability records); Income Tax §44AA"
    depends_on: "needs lawyer review"

  - finding_id: LEG-014
    surface_id: web-auth-onboarding-step-preferences
    lens: legal
    severity: P1
    issue: "Dietary preferences and allergies collected — potentially sensitive health data — no special-category consent flow"
    evidence_excerpt: "Dietary Preferences / What type of food do you prefer? / Food Allergies / Favourite Cuisines / Spice Tolerance / Household Size"
    recommendation: "Food allergies are health-related data. While DPDP 2023 does not formally create 'sensitive personal data' category (unlike GDPR Art. 9), §8(4) imposes accuracy obligations and best-practice treats health-related fields specially. Add explicit purpose statement: 'We use allergy info to flag risky dishes; you can change this anytime' and confirm with lawyer whether 'Skip' option (currently exists per web-auth-onboarding-heading) properly preserves DPDP §6 voluntariness."
    citation: "DPDP Act 2023 §8(4); best-practice — health-adjacent data"
    depends_on: "needs lawyer review"

  - finding_id: LEG-015
    surface_id: web-auth-onboarding-step-preferences
    lens: legal
    severity: P2
    issue: "'Household Size' collected — household composition is profile-enriching data, no purpose disclosed"
    evidence_excerpt: "Household Size"
    recommendation: "Household size goes beyond what's needed to deliver food. If used for portion recommendations, say so. If used for marketing segmentation (family-pack ads), separate consent is required. Either remove the field or add purpose justification."
    citation: "DPDP Act 2023 §5(ii) (purpose limitation), §6 (specific consent)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-016
    surface_id: web-auth-onboarding-heading
    lens: legal
    severity: P1
    issue: "'Skip' link on onboarding wizard implies users can bypass data collection — must confirm DPDP voluntariness; but be careful conditional consent isn't being collected"
    evidence_excerpt: "Complete Your Profile / Step {n} of 3 / Skip"
    recommendation: "Per DPDP §7(b) (lawful processing — note Act numbering varies), services cannot deny basic functionality for refusing consent to optional data. 'Skip' is good. But lawyer to confirm: does the platform actually function fully if user skips all preferences? If app blocks browsing/ordering until preferences are filled, that's conditional consent contrary to §7."
    citation: "DPDP Act 2023 §7 (conditional consent prohibition); §6(2)"
    depends_on: "needs lawyer review"

  # =============================================================================
  # VENDOR PORTAL — apps/vendor-portal/src/features/auth/
  # =============================================================================

  - finding_id: LEG-017
    surface_id: vp-auth-login-hero-heading
    lens: legal
    severity: P2
    issue: "Marketing claim on auth screen ('Grow your home kitchen business') is pre-contractual representation"
    evidence_excerpt: "Grow your home kitchen business"
    recommendation: "Pre-contractual marketing claims can be invoked in disputes under Indian Contract Act §17 (misrepresentation) and Consumer Protection Act 2019 §2(28). 'Grow your business' is sufficiently puffery to be defensible, but pair it with the binding T&C link to avoid implied warranty. Lawyer to confirm no binding promise read into hero copy."
    citation: "Indian Contract Act 1872 §17; Consumer Protection Act 2019 §2(28)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-018
    surface_id: vp-auth-login-features-list
    lens: legal
    severity: P1
    issue: "Pre-contract feature promise 'Document verification & compliance' implies platform handles regulatory compliance for chef — could shift liability"
    evidence_excerpt: "Real-time order management / Earnings & analytics dashboard / Menu management with categories / Customer reviews & ratings / Document verification & compliance"
    recommendation: "Saying 'Document verification & compliance' to a chef can be read as platform assuming FSSAI/tax compliance burden. FSSAI license under FSS Act §31 is the chef's obligation, not the platform's. Reword to 'Document upload for FSSAI and ID verification' or 'Compliance support' with explicit disclaimer in T&C that legal responsibility for FSSAI/GST/income-tax compliance rests with the chef."
    citation: "FSS Act 2006 §31; best-practice — pre-contractual misrepresentation"
    depends_on: "needs lawyer review"

  - finding_id: LEG-019
    surface_id: vp-auth-register-hero-sub
    lens: legal
    severity: P1
    issue: "'Join thousands of home chefs earning' is a verifiable factual claim — must be substantiated or removed"
    evidence_excerpt: "Join thousands of home chefs earning with Fe3dr. Your kitchen, your recipes, your rules."
    recommendation: "ASCI Code, MCA Consumer Protection (E-Commerce) Rules 2020 §6(1)(d), and Consumer Protection Act 2019 §2(28)(iv) prohibit misleading representations of fact. If 'thousands' (≥2,000) of active chefs aren't actually onboarded and earning, this is misleading. Lawyer to confirm actual numbers; if false, rewrite as 'Join home chefs across India' without quantitative claim."
    citation: "ASCI Code; Consumer Protection (E-Commerce) Rules 2020 §6(1)(d); Consumer Protection Act 2019 §2(28)(iv)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-020
    surface_id: vp-auth-register-hero-sub
    lens: legal
    severity: P2
    issue: "'Your kitchen, your recipes, your rules' contradicts platform T&C which inevitably impose menu, pricing, hygiene, and cancellation rules"
    evidence_excerpt: "Join thousands of home chefs earning with Fe3dr. Your kitchen, your recipes, your rules."
    recommendation: "Pre-contractual representation that contradicts subsequent T&C is a misrepresentation risk under Indian Contract Act §17. Vendor Terms (per StepPolicies.tsx) clearly impose platform commission, payout cadence, hygiene policy, and cancellation rules — not 'your rules'. Reword to 'Your kitchen, your menu, your schedule' or similar that is consistent with the actual contract."
    citation: "Indian Contract Act 1872 §17, §18; best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-021
    surface_id: vp-auth-register-login-link
    lens: legal
    severity: P0
    issue: "Vendor register page T&C disclosure says 'By registering, you agree...' — implies-consent pattern not §6 affirmative action"
    evidence_excerpt: "By registering, you agree to Fe3dr's Vendor Terms and Privacy Policy"
    recommendation: "Same as LEG-002 but vendor-side: button-click implies consent is not 'clear affirmative action' per DPDP §6(1). Vendor commits to Vendor Agreement, Privacy Policy, AND implicitly platform commission rate — needs explicit checkbox-gated affirmative consent at minimum. Lawyer to confirm vendor onboarding properly captures consent record per §6(2)."
    citation: "DPDP Act 2023 §6(1)–(2); Indian Contract Act 1872 §10 (consent must be free)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-022
    surface_id: vp-onb-personal-blurb
    lens: legal
    severity: P0
    issue: "'This information helps verify your identity' is KYC framing without disclosure of KYC obligations, retention, or refusal consequences"
    evidence_excerpt: "Tell us about yourself. This information helps verify your identity."
    recommendation: "KYC of vendors is required under RBI Payment Aggregator MD 2020 (Annex 2) for payout recipients. User must know: (a) data fiduciary identity, (b) KYC purpose specifically (payout disbursement + regulatory recordkeeping), (c) retention (5 yrs minimum per PMLA §12), (d) consequence of refusal (no payouts). Add a 'Why we need this' callout."
    citation: "RBI Payment Aggregator MD 2020 (Annex 2); PMLA 2002 §12; DPDP §5(ii)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-023
    surface_id: vp-onb-personal-fields
    lens: legal
    severity: P0
    issue: "Vendor full name, phone, email collected as KYC without explicit DPDP consent log capture"
    evidence_excerpt: "Full Name / Phone Number / Email Address / Pre-filled from your login / e.g. Meena Sharma / meena@example.com"
    recommendation: "Per DPDP §6(2), data fiduciary must keep a record of consent. Vendor KYC data is dual-purpose (account + payment compliance). Lawyer to confirm backend logs which version of T&C and Privacy Policy the vendor consented to, with timestamp, and that revocation per §6(4)–(6) is wired."
    citation: "DPDP Act 2023 §6(2)–(6); PMLA Record Maintenance Rules 2005"
    depends_on: "needs lawyer review"

  - finding_id: LEG-024
    surface_id: vp-onb-kitchen-blurb
    lens: legal
    severity: P1
    issue: "'Tell customers about your kitchen and what makes your food special' — chef-authored public-facing copy carries food-safety representation risk"
    evidence_excerpt: "Tell customers about your kitchen and what makes your food special."
    recommendation: "Chef-authored claims (e.g., 'organic', 'fresh', 'home-style', 'no preservatives') can attract FSSAI labelling and advertising violations (FSS Act §52, §53). Add a helper-text disclaimer: 'Avoid claims like organic, low-fat, or sugar-free unless certified — read our labelling guide.' Vendor Agreement (T&C) should explicitly transfer ad-claim liability to chef."
    citation: "FSS Act 2006 §52 (misleading advertisements), §53 (penalty); FSSAI Advertising and Claims Regulations 2018"
    depends_on: "needs lawyer review"

  - finding_id: LEG-025
    surface_id: vp-onb-personal-address-blurb
    lens: legal
    severity: P1
    issue: "Kitchen address collection does not disclose this becomes the FSSAI-registered premises address"
    evidence_excerpt: "Where your kitchen is located. This is used for delivery radius and customer visibility."
    recommendation: "Per FSS Act §31 and FSSAI Licensing & Registration Regulations 2011, a food business operator must register/license the actual premises where food is prepared. Address provided here likely flows to FSSAI license application. Clarify: 'This must be the actual kitchen address where you prepare food — it will appear on your FSSAI registration.' Critical for license validity and recall traceability."
    citation: "FSS Act 2006 §31; FSSAI Licensing & Registration Regulations 2011, Regulation 2.1.2"
    depends_on: "needs lawyer review"

  - finding_id: LEG-026
    surface_id: vp-onb-stepper-labels
    lens: legal
    severity: P1
    issue: "Stepper labels show 'Documents' generically — KYC documents required by RBI/PMLA must be enumerated upfront"
    evidence_excerpt: "Personal Info / Kitchen Details / Operations / Documents / Policies & Review (+ descriptions)"
    recommendation: "Before reaching Documents step, vendor should know what they will be asked for (PAN, Aadhaar/passport, FSSAI license, bank proof, cancelled cheque). Add a 'What you'll need' checklist at signup or before Documents step. Failure to disclose required documents upfront has been flagged by consumer courts as misleading pre-onboarding."
    citation: "RBI Payment Aggregator MD 2020 Annex 2 (KYC docs); CP(E-Commerce) Rules 2020 §6"
    depends_on: "needs lawyer review"

  - finding_id: LEG-027
    surface_id: vp-onb-ops-pricing-title
    lens: legal
    severity: P1
    issue: "'You can change these anytime' — pricing/prep-time promise without reference to platform commission deductions"
    evidence_excerpt: "Delivery & Pricing / Set your preparation time, delivery range, and pricing. You can change these anytime."
    recommendation: "Vendors set list price, but final payout reflects platform commission (per StepPolicies). 'You can change pricing anytime' creates expectation of full price control; needs callout: 'Your menu prices set what customers pay. See payout breakdown for the amount you receive after platform commission.' Lawyer to confirm Vendor Agreement properly discloses commission percentage upfront."
    citation: "Indian Contract Act 1872 §17; CP(E-Commerce) Rules 2020 §5(3) (transparency of fees)"
    depends_on: "needs lawyer review"

  # =============================================================================
  # DELIVERY PORTAL — apps/delivery-portal/
  # =============================================================================

  - finding_id: LEG-028
    surface_id: dp-auth-mode-driver-cta
    lens: legal
    severity: P1
    issue: "'I'm a Driver' / 'Login or sign up to deliver with Fe3dr' — driver-employment classification not addressed pre-onboarding"
    evidence_excerpt: "I'm a Driver / Login or sign up to deliver with Fe3dr"
    recommendation: "Gig worker classification under Code on Social Security 2020 §2(35) and Code on Wages 2019 has not been notified in many states, but Karnataka Platform-Based Gig Workers (Social Security and Welfare) Bill 2024 and Rajasthan Platform-Based Gig Workers Act 2023 impose specific disclosures pre-onboarding. Lawyer to confirm pre-onboarding state-specific gig-worker disclosures (welfare fee, ID issuance, termination process) are present."
    citation: "Code on Social Security 2020 §2(35) gig worker; Rajasthan PBGW Act 2023; Karnataka PBGW Bill 2024"
    depends_on: "needs lawyer review"

  - finding_id: LEG-029
    surface_id: dp-auth-google-button
    lens: legal
    severity: P1
    issue: "Driver OAuth signup (Google) — no separate driver-specific consent for KYC data sharing"
    evidence_excerpt: "Continue with Google"
    recommendation: "Driver onboarding is dual-purpose: account creation + gig-platform onboarding. OAuth signup pulls email/name; subsequent screens collect KYC, vehicle, license, Aadhaar, PAN. DPDP §6 'specific' consent requires separating these. Currently driver clicks Google OAuth then is funneled into KYC with no separate consent. Lawyer to confirm separate KYC-purpose consent screen exists between OAuth callback and StepPersonalInfo."
    citation: "DPDP Act 2023 §6(1) (specific consent); §5(ii)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-030
    surface_id: dp-auth-step3-doc-labels
    lens: legal
    severity: P0
    issue: "Aadhaar collected as document with no UIDAI offline e-KYC / consent framework disclosure"
    evidence_excerpt: "Profile Photo / Driving License / Aadhaar Card / PAN Card / Vehicle RC / Insurance / Vehicle — Front View / Back View / Left Side / Right Side / Top View / Number Plate (clear photo) / Bicycle — Front/Back/Left/Right View / Police Verification"
    recommendation: "Aadhaar Act 2016 §8 and Aadhaar Regulations 2016 (Authentication) restrict who can collect Aadhaar and how. Private platforms are NOT authorised to seek Aadhaar authentication except under specific licences. Acceptable mechanisms: UIDAI offline e-KYC (Aadhaar XML/QR) or DigiLocker-issued Aadhaar — NOT raw Aadhaar card upload. Raw Aadhaar upload also creates UIDAI §29(1) breach risk (sharing core biometric/identity number) and PMLA risk. URGENT: switch to Aadhaar offline e-KYC or DigiLocker integration; mask last 8 digits if photo is retained."
    citation: "Aadhaar Act 2016 §8, §29; Aadhaar Authentication Regulations 2016; UIDAI Circular K-11020/217/2018"
    depends_on: "needs lawyer review"

  - finding_id: LEG-031
    surface_id: dp-auth-step3-doc-labels
    lens: legal
    severity: P0
    issue: "PAN Card collection with no Income Tax §139A purpose disclosure or Rule 114B compliance"
    evidence_excerpt: "PAN Card"
    recommendation: "PAN collection requires §139A purpose disclosure (here: TDS on driver earnings under §194O e-commerce operator TDS). Form 60 substitute mandatory if driver has no PAN. Add explanatory text under PAN slot: 'We collect PAN to deduct TDS at 1% on your earnings, per Income Tax §194O.' Lawyer to confirm Form 60 fallback workflow exists for PAN-less drivers."
    citation: "Income Tax Act §139A; Rule 114B; §194O (TDS by e-commerce operator); Form 60"
    depends_on: "needs lawyer review"

  - finding_id: LEG-032
    surface_id: dp-auth-step3-doc-labels
    lens: legal
    severity: P1
    issue: "'Police Verification' document slot — uploading police verification certificate has DPDP §8(4) accuracy and §10 sharing implications"
    evidence_excerpt: "Police Verification"
    recommendation: "Police verification documents contain criminal-record-adjacent info; treatment as 'special category' is best-practice. Lawyer to confirm: (a) document retention duration, (b) access controls in admin portal, (c) whether absence is grounds for rejection (must be disclosed upfront), (d) handling if police verification reveals adverse history (driver classification + recourse). Currently no UI disclosure of any of this."
    citation: "DPDP Act 2023 §8(4) accuracy, §8(7) retention; best-practice criminal-record handling"
    depends_on: "needs lawyer review"

  - finding_id: LEG-033
    surface_id: dp-auth-step3-bicycle-hint
    lens: legal
    severity: P2
    issue: "'Since you're using a bicycle, driving license, vehicle RC, and insurance are not required.' — accepting bicycle drivers without insurance shifts third-party liability risk"
    evidence_excerpt: "Since you're using a bicycle, driving license, vehicle RC, and insurance are not required."
    recommendation: "Bicycle delivery riders are not subject to Motor Vehicles Act but are still subject to general tort liability (negligence causing injury). Platform liability under Consumer Protection Act 2019 §86 (e-commerce entity) and §94 (duty of due diligence) is heightened when there is no driver-side insurance. Lawyer to confirm platform-purchased third-party liability insurance covers bicycle riders; flag whether copy should disclose insurance arrangement."
    citation: "Consumer Protection Act 2019 §86, §94; Motor Vehicles Act 1988 (inapplicable); general tort"
    depends_on: "needs lawyer review"

  - finding_id: LEG-034
    surface_id: dp-auth-step3-section-headers
    lens: legal
    severity: P1
    issue: "'Vehicle Photos — All Angles / Optional' framing for photos that the platform later uses for verification creates ambiguous obligation"
    evidence_excerpt: "Personal Documents / Vehicle Documents / Vehicle Photos — All Angles / Optional"
    recommendation: "If photos are truly optional, no problem. If approval can be denied for missing photos, they aren't really optional — that's misrepresentation. Lawyer to confirm with PM whether photos affect approval; if yes, change copy to '(required for approval)' or remove 'Optional' label."
    citation: "Indian Contract Act 1872 §18; CP(E-Commerce) Rules 2020 §6(1)(d)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-035
    surface_id: dp-auth-step-payout-bank-fields
    lens: legal
    severity: P0
    issue: "Bank account collection (Account Holder Name, Account Number, IFSC) with no RBI penny-drop verification disclosure or escrow disclosure"
    evidence_excerpt: "Account Holder Name * / Name as on bank account / Account Number * / Enter account number / IFSC Code * / e.g., SBIN0001234"
    recommendation: "Per RBI Payment Aggregator MD 2020 §8.5 and §10, payout to driver requires: (a) name-match verification with PAN, (b) bank account validation (penny drop), (c) settlement timeline disclosure, (d) escrow/nodal account disclosure for funds awaiting settlement. None visible in onboarding UI. Lawyer to confirm whether penny-drop happens silently and whether disclosure obligation is met in Driver Agreement."
    citation: "RBI Payment Aggregator MD 2020 §8.5, §10; PMLA Record Maintenance Rules 2005 Rule 3"
    depends_on: "needs lawyer review"

  - finding_id: LEG-036
    surface_id: dp-auth-step-payout-upi
    lens: legal
    severity: P1
    issue: "UPI ID collection with no NPCI-mandated VPA verification disclosure"
    evidence_excerpt: "UPI ID * + e.g., name@upi"
    recommendation: "NPCI UPI Procedural Guidelines require name-resolution of VPA before payout to confirm beneficiary. Disclose to driver that 'We'll verify this UPI ID resolves to your registered name' to set expectation; refusal handling needed for VPA-name mismatch. Lawyer to confirm verification logic exists and matches the DPDP §8(4) accuracy obligation."
    citation: "NPCI UPI Procedural Guidelines; DPDP §8(4)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-037
    surface_id: dp-auth-step1-emergency-name
    lens: legal
    severity: P1
    issue: "Emergency contact name/phone collected — third-party personal data with no consent from that third party"
    evidence_excerpt: "Emergency Contact Name * + Full name / Emergency Phone * + +91 9876543210"
    recommendation: "Per DPDP §6(1), data fiduciary needs consent from each data principal. The driver cannot consent on behalf of their emergency contact. Add a notice to driver: 'You confirm your emergency contact has agreed to be reached by Fe3dr in case of safety incidents.' Best-practice (GDPR Art. 14 analogue) also recommends notice-to-third-party. Lawyer to confirm 'driver-attested consent' is sufficient under DPDP."
    citation: "DPDP Act 2023 §6(1); §5(i) (notice to data principal)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-038
    surface_id: dp-auth-step1-dob
    lens: legal
    severity: P0
    issue: "Date of Birth collected with no age-validation gate — driver could be a minor"
    evidence_excerpt: "Date of Birth"
    recommendation: "Drivers must be ≥18 (Motor Vehicles Act 1988 §4 for motorised vehicle, light-MV class — and platform's age requirement). Bicycle riders technically have no MV Act age floor, but platform may set its own. DPDP §9 imposes parental-consent requirement for <18. Add inline DOB validation: '(must be 18 or older)' helper, and backend enforcement."
    citation: "DPDP Act 2023 §9; Motor Vehicles Act 1988 §4"
    depends_on: "needs lawyer review"

  - finding_id: LEG-039
    surface_id: dp-auth-step1-referral
    lens: legal
    severity: P2
    issue: "Referral code field — pre-onboarding promise of referral reward must match Vendor/Driver Agreement"
    evidence_excerpt: "Referral Code + Enter referral code (optional) + Validate"
    recommendation: "Referral codes create contractual obligations to both referrer (referral bonus) and referee (signup bonus). Pre-disclose: 'What does this code do?' tooltip or info text. Lawyer to confirm referral T&C are linked from this field; otherwise referrer can sue for unpaid referral if rewarded inconsistently."
    citation: "Indian Contract Act 1872 §10; CP(E-Commerce) Rules 2020 §5(3)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-040
    surface_id: dp-auth-step1-referral-valid
    lens: legal
    severity: P2
    issue: "'Referred by {referrerName}' shows referrer's name — DPDP §5/§6 concern about disclosing one user's identity to another"
    evidence_excerpt: "Referred by {referrerName}"
    recommendation: "Showing referrer's full name to referee may exceed what referrer consented to. Consider showing initials only ('Referred by R. S.') unless referral T&C explicitly inform referrer their name is shown. Lawyer to confirm referral consent text addresses cross-disclosure."
    citation: "DPDP Act 2023 §6(1); §10 disclosure"
    depends_on: "needs lawyer review"

  - finding_id: LEG-041
    surface_id: dp-auth-step4-h
    lens: legal
    severity: P1
    issue: "'A small subscription to keep Fe3dr running — you keep every rupee you earn' — claim of '100% earnings' undermined by subsequent commission/deduction disclosures"
    evidence_excerpt: "Choose Your Plan + A small subscription to keep Fe3dr running — you keep every rupee you earn."
    recommendation: "Marketing promise of 'every rupee' is misrepresentation if any deductions exist (TDS under §194O, GST on subscription, late-cancellation deductions, etc.). Reword to 'You keep your delivery fees and tips — we charge a flat subscription, no per-order commission.' Lawyer to confirm no other deductions exist that contradict this claim."
    citation: "Indian Contract Act 1872 §17, §18; CP Act 2019 §2(28)(iv); Income Tax §194O TDS"
    depends_on: "needs lawyer review"

  - finding_id: LEG-042
    surface_id: dp-auth-step4-zero-commission
    lens: legal
    severity: P0
    issue: "Multiple repeated 'no commission' / '100%' promises in subscription marketing block — high misrepresentation exposure"
    evidence_excerpt: "We don't take any commission from your earnings + Every delivery fee and every tip goes directly to your account — 100%. We only charge a small subscription fee to keep the platform running and help you get more orders. We're here to help you succeed, not to make profit from your hard work."
    recommendation: "'100%' and 'no commission' must be literally true in every scenario including: surge pricing, refunds, customer cancellations after pickup, TDS deductions, GST on subscription. Add asterisked disclaimer with link to full payout terms. 'Not to make profit from your hard work' is a separate concern — implied non-profit promise creates fiduciary-style expectation that could be exploited in disputes. Strongly recommend removing or substantially softening."
    citation: "CP Act 2019 §2(28); CP(E-Commerce) Rules 2020 §6; Indian Contract Act §17–§18"
    depends_on: "needs lawyer review"

  - finding_id: LEG-043
    surface_id: dp-auth-step4-trial
    lens: legal
    severity: P1
    issue: "'{n}-day free trial / Try everything free. No card needed.' — auto-conversion to paid subscription must be disclosed at signup"
    evidence_excerpt: "{n}-day free trial + Try everything free. No card needed."
    recommendation: "Free trial that converts to paid is a 'dark pattern' per Guidelines for Prevention and Regulation of Dark Patterns 2023 (CCPA notification) — specifically 'Subscription Trap' (Annexure I.4). Required disclosure: (a) trial duration, (b) what happens after trial (auto-charge or auto-cancel), (c) cancellation steps. Currently '{n}-day free trial' is vague. Lawyer to verify against CCPA Dark Patterns Guidelines 2023."
    citation: "CCPA Guidelines for Prevention of Dark Patterns 2023 (Annexure I.4 Subscription Trap)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-044
    surface_id: dp-auth-step4-threshold
    lens: legal
    severity: P1
    issue: "'No charge until you earn {amount}' creates conditional billing promise — billing trigger logic must be deterministic and disclosed"
    evidence_excerpt: "No charge until you earn {amount} + We only bill after you start earning well."
    recommendation: "'Earn well' is non-deterministic; the {amount} threshold is the actual trigger. Reword: 'No charge until your total earnings reach ₹{amount}. After that, billing starts on your next cycle.' Lawyer to confirm threshold value and billing-cycle interaction with Vendor/Driver Agreement."
    citation: "CP(E-Commerce) Rules 2020 §5(3) transparency; Indian Contract Act §29 (certainty of contract terms)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-045
    surface_id: dp-auth-step4-billing-howto
    lens: legal
    severity: P1
    issue: "4-step billing explanation is dense legal-info copy presented inside onboarding — readability test for binding subscription terms"
    evidence_excerpt: "How billing works: + 4-step ordered list explaining trial, threshold, deduction, cancel-anytime"
    recommendation: "Per STYLE-GUIDE.md §5 'Legal-Page Tone', binding text must be plain language with one idea per paragraph and 15-25-word sentences. Without seeing actual text, lawyer to confirm: each step is a single sentence, defined terms (Free Trial, Earnings Threshold, Billing Cycle, Cancellation Window) are in bold first use, and the canonical version is also in Vendor Agreement. Inconsistency between onboarding summary and binding T&C is a contract-formation problem."
    citation: "Indian Contract Act §10, §29; best-practice — plain language"
    depends_on: "needs lawyer review"

  - finding_id: LEG-046
    surface_id: dp-auth-step4-secure-pay
    lens: legal
    severity: P1
    issue: "'Payments are securely processed via {gateway}' — gateway is a sub-processor; DPDP §10 third-party disclosure required"
    evidence_excerpt: "Payments are securely processed via {gateway}. You can change your plan or cancel anytime from your dashboard."
    recommendation: "Per DPDP §10 (Significant Data Fiduciary) and §5(ii)/(v), data shared with payment gateway (sub-processor) must be disclosed: which fields, for what purpose, retention by gateway. Privacy Policy must list gateway as a third-party processor with their privacy policy link. Lawyer to confirm Razorpay/Stripe/PayU is listed in privacy policy as named sub-processor."
    citation: "DPDP Act 2023 §10; §5(ii); §5(v) cross-border transfer if applicable"
    depends_on: "needs lawyer review"

  - finding_id: LEG-047
    surface_id: dp-auth-step5-terms
    lens: legal
    severity: P0
    issue: "'I agree to the Terms & Conditions, Privacy Policy' — links not wired (per inventory note 'no link wired')"
    evidence_excerpt: "I agree to the Terms & Conditions, Privacy Policy, and confirm that all information provided is accurate and up to date."
    recommendation: "Consent to T&C requires actual access to T&C. Per Indian Contract Act §13 (consent must be free and informed) and DPDP §6(1)(d) (consent must be 'informed'), unlinked T&C reference does NOT create binding consent. This is a contract-formation defect. URGENT: wire T&C and Privacy Policy to live, versioned documents before any production use; also bundle the 'accuracy of information' attestation into a separate checkbox (currently bundled with T&C — multiple-issue consent)."
    citation: "Indian Contract Act 1872 §13; DPDP Act 2023 §6(1)(d); §6 unbundled consent"
    depends_on: "needs lawyer review"

  - finding_id: LEG-048
    surface_id: dp-auth-step5-terms
    lens: legal
    severity: P0
    issue: "Bundled consent: 'Terms + Privacy Policy + accuracy attestation' is three obligations under one checkbox"
    evidence_excerpt: "I agree to the Terms & Conditions, Privacy Policy, and confirm that all information provided is accurate and up to date."
    recommendation: "DPDP §6(1) requires consent to be specific to each purpose. T&C acceptance (contract) + Privacy Policy consent (data processing) + factual-accuracy attestation (warranty) are three separate legal acts. Split into 3 checkboxes (or 2 minimum: 'I agree to T&C and Privacy Policy' + 'I confirm information is accurate'). Lawyer to confirm split structure."
    citation: "DPDP Act 2023 §6(1) (specific consent); Indian Contract Act §10"
    depends_on: "needs lawyer review"

  - finding_id: LEG-049
    surface_id: dp-auth-step5-plan-summary
    lens: legal
    severity: P1
    issue: "'Plan selected. Payments handled securely via Razorpay.' — gateway hardcoded; if gateway changes, this is a misrepresentation"
    evidence_excerpt: "Plan selected. Payments handled securely via Razorpay. + Billing starts only after you reach the minimum earnings threshold."
    recommendation: "Hardcoding Razorpay creates risk if platform switches gateways. Use parameterised {gateway} as elsewhere. Also: 'minimum earnings threshold' is vague; should reference the specific amount shown earlier and confirm consistency with Driver Agreement."
    citation: "Indian Contract Act §17–§18; DPDP §10 (third-party disclosure)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-050
    surface_id: dp-auth-onboarding-header
    lens: legal
    severity: P2
    issue: "'Become a Delivery Partner' — 'Delivery Partner' is gig-economy terminology that obscures contractor/employee classification"
    evidence_excerpt: "Become a Delivery Partner"
    recommendation: "'Partner' implies equity/profit-sharing relationship; here driver is an independent contractor. Karnataka and Rajasthan gig-worker laws use 'platform-based gig worker'. Inside the onboarding flow (where contractual relationship is being formed), 'Driver' is more accurate. Per Style Guide, customer-facing 'Delivery partner' is OK but driver-facing 'Driver' is preferred. Lawyer to confirm Driver Agreement is unambiguous on independent-contractor classification."
    citation: "Code on Social Security 2020 §2(35); Rajasthan PBGW Act 2023 §2(g); best-practice — classification clarity"
    depends_on: "needs lawyer review"

  - finding_id: LEG-051
    surface_id: dp-auth-status-descriptions
    lens: legal
    severity: P1
    issue: "'Our team is currently reviewing your application. This usually takes 1-2 business days.' — SLA promise without disclaimer or recourse"
    evidence_excerpt: "Your application requires some changes before it can be approved. / Our team is currently reviewing your application. This usually takes 1-2 business days. / Your application has been received and will be reviewed shortly."
    recommendation: "Stating '1-2 business days' creates expectation that, if breached, drives consumer complaints (CP Act §35). Add 'usually' (good — already there) and a 'What if it takes longer?' link. Lawyer to confirm whether breach of stated SLA gives rise to any liability."
    citation: "CP Act 2019 §35 (consumer complaints); §94 (e-commerce due diligence)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-052
    surface_id: dp-auth-status-rejection-reason
    lens: legal
    severity: P0
    issue: "'Reason for rejection: {rejectionReason}' — no appeal pathway shown, no DPDP §11(2) recourse"
    evidence_excerpt: "Reason for rejection: + {rejectionReason}"
    recommendation: "Per DPDP §11(2), data principal has the right to know how their data is being processed and to grieve adverse decisions. Per CP(E-Commerce) Rules 2020 §5(6), platform must have grievance redressal mechanism. Add 'Disagree with this decision? Contact us at [grievance officer]' below rejection reason. Also: per Algorithm Transparency Bill (draft), automated rejection must be flagged."
    citation: "DPDP Act 2023 §11(2), §13; CP(E-Commerce) Rules 2020 §5(6); draft Digital India Act"
    depends_on: "needs lawyer review"

  - finding_id: LEG-053
    surface_id: dp-auth-status-auto-refresh
    lens: legal
    severity: P3
    issue: "'Auto-refreshing every 30 seconds' — auto-refresh of status page is benign, but no privacy disclosure on background polling"
    evidence_excerpt: "Auto-refreshing every 30 seconds"
    recommendation: "Minor: background polling is normal app behavior and would not normally need disclosure. Listed for completeness in case automated decisions are made server-side during polling window. No action required unless polling triggers other data processing."
    citation: "best-practice"
    depends_on: "needs lawyer review"

  # =============================================================================
  # ADMIN PORTAL — apps/admin-portal/
  # =============================================================================

  - finding_id: LEG-054
    surface_id: ap-auth-login-feature-list
    lens: legal
    severity: P2
    issue: "Admin login left-pane features list contains sales-style copy on internal portal — unnecessary contractual surface"
    evidence_excerpt: "User & role management; Chef verification & approvals; Order monitoring & refunds; Platform analytics & insights; Revenue tracking & payouts; Content moderation tools"
    recommendation: "Admin portal is internal-only; these features don't need pre-contractual disclosure. But content moderation, refunds, payouts, verification are all areas with chef/customer-facing legal consequences — if this copy is ever seen by an admin who shares a screenshot externally, it forms an implied platform-capability statement. Recommend strip down to minimal 'Admin Sign In' surface. Low priority but advisable."
    citation: "best-practice — internal-tool minimal disclosure"
    depends_on: "needs lawyer review"

  - finding_id: LEG-055
    surface_id: ap-auth-login-instruction
    lens: legal
    severity: P1
    issue: "Admin login has no security warning notice (unauthorized access, monitoring) — best-practice for privileged accounts"
    evidence_excerpt: "Access the administration dashboard with your internal credentials"
    recommendation: "Internal/privileged admin login screens should display a 'This system is monitored. Unauthorized access prohibited under IT Act 2000 §66' style notice. Provides basis for prosecuting external unauthorized access attempts under IT Act §43, §66. Lawyer to confirm wording and whether session activity is logged for §66 evidentiary chain."
    citation: "IT Act 2000 §43, §66 unauthorized access; best-practice (NIST SP 800-53 AC-8)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-056
    surface_id: ap-auth-totp-enroll-help
    lens: legal
    severity: P1
    issue: "'Your organization requires 2FA for admins' — TOTP is correct security practice; but recovery-code disclosure is missing"
    evidence_excerpt: "Your organization requires 2FA for admins. Scan this QR with Google Authenticator (or similar), then enter the generated code."
    recommendation: "TOTP enrollment without recovery-code disclosure creates lockout risk. Best-practice (NIST SP 800-63B §5.1.2) requires single-use recovery codes shown once at enrollment. Lawyer concern: if admin is locked out and lacks recovery, what is the audit-trail-preserving recovery path? Must be documented in Admin Security Policy."
    citation: "NIST SP 800-63B §5.1.2; DPDP §8(5) reasonable security"
    depends_on: "needs lawyer review"

  - finding_id: LEG-057
    surface_id: ap-secsettings-2fa-disable
    lens: legal
    severity: P0
    issue: "Disabling 2FA is highly-sensitive privileged action — no co-approval or audit-trail disclosure"
    evidence_excerpt: "Disable 2FA; Requires your password and a current 6-digit code.; Password; Disabling...; Cancel"
    recommendation: "Disabling 2FA on an admin account is a security-policy event. Best-practice (and DPDP §8(5) reasonable security): require admin manager co-approval, log the event to audit-service immutably, and notify other admins. Currently only password + TOTP — same factors used for normal login. Lawyer to confirm internal Security Policy permits self-service 2FA disable, and that audit trail is sufficient."
    citation: "DPDP §8(5); ISO/IEC 27001 A.9.4.1; best-practice"
    depends_on: "needs lawyer review"

  # =============================================================================
  # MOBILE CUSTOMER — apps/mobile-customer/
  # =============================================================================

  - finding_id: LEG-058
    surface_id: mc-auth-login-title
    lens: legal
    severity: P1
    issue: "Mobile customer login 'Welcome back' — local title override but underlying shared LoginScreen may not surface mobile-specific DPDP/T&C disclosures"
    evidence_excerpt: '"Welcome back"'
    recommendation: "Inventory notes the rest of LoginScreen lives in @homechef/mobile-shared. Lawyer to confirm: (a) shared screen has T&C/Privacy footer same as web, (b) mobile-specific platform consent (Apple App Tracking Transparency, Google Play Data Safety, app permission consent) is captured before signin. Apple ATT prompt + Google Play DSA disclosure may need to fire before first data is processed."
    citation: "Apple App Tracking Transparency; Google Play Data Safety; DPDP §6(1)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-059
    surface_id: mc-auth-register-host
    lens: legal
    severity: P0
    issue: "Mobile customer register screen has zero local strings — all copy in shared screen; impossible to audit consent flow from local code"
    evidence_excerpt: "(shared RegisterScreen — no local strings)"
    recommendation: "Hidden in @homechef/mobile-shared/screens. Lawyer + auditor must inspect that shared package directly to verify DPDP §5/§6 disclosures are present. Currently invisible to per-app audit. Recommend exposing mobile-shared inventory in subsequent audit pass."
    citation: "DPDP Act 2023 §5, §6 (auditability)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-060
    surface_id: mc-onb-step3-title
    lens: legal
    severity: P2
    issue: "Mobile customer preferences subtitle 'Select your favourite cuisines to get personalised recommendations' explicitly says recommendation profiling — needs DPDP profiling notice"
    evidence_excerpt: "What do you love to eat? + Select your favourite cuisines to get personalised recommendations."
    recommendation: "Per DPDP §5(ii), purposes must be disclosed. 'Personalised recommendations' is profiling per draft DPDP Rules 2025. Add 'You can change preferences anytime; we don't share them outside Fe3dr.' Lawyer to confirm profiling-purpose disclosure aligns with privacy policy."
    citation: "DPDP Act 2023 §5(ii); draft DPDP Rules 2025 (profiling provisions); §6 consent"
    depends_on: "needs lawyer review"

  - finding_id: LEG-061
    surface_id: mc-onb-step1-labels
    lens: legal
    severity: P1
    issue: "Mobile-customer collects 'Phone Number' as part of onboarding — TRAI commercial-communications regime applies"
    evidence_excerpt: "First Name / Last Name / Phone Number"
    recommendation: "Same as LEG-012 (web equivalent). Mobile app must split transactional-SMS consent from promotional-SMS consent. Best-practice: explicit checkboxes."
    citation: "TRAI TCCCPR 2018; DPDP §6 specific consent"
    depends_on: "needs lawyer review"

  # =============================================================================
  # MOBILE VENDOR — apps/mobile-vendor/
  # =============================================================================

  - finding_id: LEG-062
    surface_id: mv-auth-login-title
    lens: legal
    severity: P1
    issue: "Mobile vendor login title 'Welcome back' — same shared-screen audit gap as mobile customer"
    evidence_excerpt: "Welcome back"
    recommendation: "Same as LEG-058. Vendor side has higher legal exposure because of KYC + commission terms. Lawyer to audit @homechef/mobile-shared/screens for vendor-specific disclosures."
    citation: "DPDP §6; RBI PA MD 2020"
    depends_on: "needs lawyer review"

  - finding_id: LEG-063
    surface_id: mv-onb-docs-sub
    lens: legal
    severity: P0
    issue: "'Upload your identity and FSSAI documents' — both are critical regulatory documents; no upload-purpose, retention, or sharing disclosure"
    evidence_excerpt: "Upload your identity and FSSAI documents"
    recommendation: "FSSAI license (FSS Act §31 mandatory for food business) and ID proof (KYC per RBI PA MD) are dual-purpose collections. Disclose: (a) FSSAI license number will be shown on customer order receipts (FSSAI mandates this), (b) ID proof stored encrypted for KYC retention period (5 yrs PMLA), (c) consequences of refusal (no payouts, no chef approval). Currently invisible to vendor."
    citation: "FSS Act 2006 §31; FSSAI Regulations Reg 2.1.2; PMLA Record Maintenance Rules 2005 Rule 3"
    depends_on: "needs lawyer review"

  - finding_id: LEG-064
    surface_id: mv-onb-docs-fssai-slot
    lens: legal
    severity: P0
    issue: "'FSSAI License' upload slot — if marked optional or accepted-as-blank, platform enables illegal food business"
    evidence_excerpt: "FSSAI License"
    recommendation: "Per FSS Act §31, no person can run a food business without FSSAI registration/license. Platform onboarding chefs without FSSAI exposes platform to vicarious liability under FSS Act §53 and IT Rules 2021 §3(1)(b) (publishing illegal info). Lawyer to confirm: (a) FSSAI license is hard-required (no submit without), (b) license number is validated against FSSAI database (or at least format check), (c) Vendor Agreement explicitly says chef warrants FSSAI is current."
    citation: "FSS Act 2006 §31, §53; FSSAI Licensing Regulations 2011; IT Rules 2021 §3(1)(b)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-065
    surface_id: mv-onb-docs-id-slot
    lens: legal
    severity: P0
    issue: "'ID Proof' is non-specific — must specify accepted IDs (PAN required for payouts; Aadhaar offline only)"
    evidence_excerpt: "ID Proof"
    recommendation: "Same Aadhaar concerns as LEG-030. Spec must say: 'Upload PAN card (required for payouts)' and separately 'Identity verification: choose Aadhaar offline e-KYC, DigiLocker, or Voter ID/Passport scan'. Generic 'ID Proof' invites uploads of raw Aadhaar (UIDAI §29 breach risk)."
    citation: "Aadhaar Act 2016 §29; Income Tax §139A; PMLA Rule 9 KYC"
    depends_on: "needs lawyer review"

  - finding_id: LEG-066
    surface_id: mv-onb-policies-heading
    lens: legal
    severity: P1
    issue: "'Policies / Review and accept terms' is bare-minimum policy-acceptance UI; deep concern about how policies are structured"
    evidence_excerpt: "Policies / Review and accept terms"
    recommendation: "Per vendor-portal StepPolicies.tsx (web equivalent), three checkboxes: Terms, Hygiene Policy, Cancellation Policy. Each must be individually checked. Inventory does not show whether mobile-vendor mirrors this; lawyer/dev to confirm parity. Bundled or pre-checked = §6 violation."
    citation: "DPDP §6(1); Indian Contract Act §13"
    depends_on: "needs lawyer review"

  - finding_id: LEG-067
    surface_id: mv-onb-review-section-policies
    lens: legal
    severity: P1
    issue: "Review screen shows 'Policies' section but inventory doesn't confirm read-only display of which policies + versions were accepted"
    evidence_excerpt: "Policies (section label)"
    recommendation: "Per DPDP §6(2) (consent record), Review screen should show: which policy versions were accepted, at what time, with consent token. Currently only labels visible in inventory. Lawyer to confirm consent record is preserved server-side and shown on Review for user verification."
    citation: "DPDP §6(2) record of consent"
    depends_on: "needs lawyer review"

  - finding_id: LEG-068
    surface_id: mv-onb-review-field-labels
    lens: legal
    severity: P1
    issue: "'Terms Accepted / Cancellation Policy' shown as a review row — but does not link out to the actual accepted version"
    evidence_excerpt: "Full Name / Phone / Email / Business Name / Cuisines / Description / Open Days / Prep Time / Service Radius / ID Proof / FSSAI License / Terms Accepted / Cancellation Policy"
    recommendation: "Reviewing 'Terms Accepted: ✓' on submission screen does not give user access to the version they accepted. Best-practice: link to the versioned policy document so user can confirm. DPDP §11(1) (access right) likely requires this on demand."
    citation: "DPDP Act 2023 §11(1); best-practice — versioned consent"
    depends_on: "needs lawyer review"

  # =============================================================================
  # MOBILE DELIVERY — apps/mobile-delivery/
  # =============================================================================

  - finding_id: LEG-069
    surface_id: md-auth-002
    lens: legal
    severity: P2
    issue: "Developer-shaped error 'Google sign-in failed: no ID token' could surface to user with technical detail"
    evidence_excerpt: 'error: "Google sign-in failed: no ID token"'
    recommendation: "Technical error strings exposed to users are not directly a legal issue, but they reveal internals (OAuth ID-token flow) which is a minor security-disclosure concern. Reword to 'Sign-in with Google didn't complete. Try again.' Lawyer concern is minimal but flagged for completeness."
    citation: "best-practice — minimum information disclosure"
    depends_on: "needs lawyer review"

  - finding_id: LEG-070
    surface_id: md-auth-003
    lens: legal
    severity: P2
    issue: "'Apple sign-in failed: no identity token' — same as above; Apple has additional Sign-In-with-Apple disclosure requirements"
    evidence_excerpt: 'error: "Apple sign-in failed: no identity token"'
    recommendation: "Apple Sign-In Review Guidelines §4.8 require equivalent prominence of SIWA when other third-party signin is offered. Lawyer to confirm SIWA is offered alongside Google. Also user-facing copy: 'Sign-in with Apple didn't complete. Try again.'"
    citation: "Apple App Store Review Guidelines §4.8; best-practice error UX"
    depends_on: "needs lawyer review"

  - finding_id: LEG-071
    surface_id: md-auth-004
    lens: legal
    severity: P1
    issue: "'Biometric authentication failed' — biometric data is sensitive; DPDP §8(4) accuracy obligation + Apple/Google biometric API constraints"
    evidence_excerpt: 'error: "Biometric authentication failed"'
    recommendation: "Biometric authentication on iOS (FaceID/TouchID) and Android (BiometricPrompt) keeps biometric data on-device — that's compliant. But platform must disclose to user pre-enrollment that biometric is locally stored, not server-transmitted. Add a one-time disclosure on first enrollment. Lawyer to confirm enrollment-disclosure copy exists in mobile-shared."
    citation: "DPDP §6 specific consent; Apple Privacy Manifests; Google Play Data Safety"
    depends_on: "needs lawyer review"

  - finding_id: LEG-072
    surface_id: md-auth-005
    lens: legal
    severity: P3
    issue: "'No saved session found. Please log in with email.' — benign UX message; no legal issue but mentioned for completeness"
    evidence_excerpt: 'error: "No saved session found. Please log in with email."'
    recommendation: "No action required from a legal perspective."
    citation: "n/a"
    depends_on: "needs lawyer review"

  - finding_id: LEG-073
    surface_id: md-onb-001
    lens: legal
    severity: P3
    issue: "'Step 1 of 6' header stepper — cosmetic; no legal concern except confirming user knows what's coming"
    evidence_excerpt: '"Step 1 of 6"'
    recommendation: "Best-practice consumer disclosure: tell user upfront what each step will collect. Currently steps are revealed sequentially. Recommend a pre-step preview list ('Here's what we'll ask for'). Lawyer notes this as best-practice not requirement."
    citation: "best-practice — informed consent UX"
    depends_on: "needs lawyer review"

  - finding_id: LEG-074
    surface_id: md-onb-042
    lens: legal
    severity: P0
    issue: "'Upload Documents' mobile screen — same Aadhaar/PAN/license concerns as dp-auth-step3-doc-labels"
    evidence_excerpt: "Upload Documents"
    recommendation: "Mobile parity with delivery-portal. All concerns LEG-030, LEG-031, LEG-032 apply. Lawyer to confirm UIDAI offline e-KYC integration in mobile flow, especially because mobile camera-based ID capture is more error-prone and creates higher PII-leak risk."
    citation: "Aadhaar Act §29; Income Tax §139A; PMLA Rule 9"
    depends_on: "needs lawyer review"

  - finding_id: LEG-075
    surface_id: md-onb-044
    lens: legal
    severity: P0
    issue: "'Driving License' slot — Motor Vehicles Act compliance and DL data treatment"
    evidence_excerpt: "Driving License"
    recommendation: "DL is regulated identity document. MV Act §3 requires valid DL for motor-vehicle delivery. Backend must verify DL validity (not expired, valid class) — for liability protection. Per DPDP §8(4), DL data must be accurate. Disclose: 'We verify your DL with the Ministry of Road Transport database' if mParivahan/DigiLocker is used. Lawyer to confirm verification mechanism."
    citation: "Motor Vehicles Act 1988 §3, §10; DPDP §8(4); DigiLocker Act"
    depends_on: "needs lawyer review"

  - finding_id: LEG-076
    surface_id: md-onb-045
    lens: legal
    severity: P0
    issue: "Mobile 'ID Proof' slot — generic label same concern as mv-onb-docs-id-slot (LEG-065)"
    evidence_excerpt: "ID Proof"
    recommendation: "Same as LEG-065. Specify accepted IDs; do not accept raw Aadhaar."
    citation: "Aadhaar Act §29; PMLA Rule 9"
    depends_on: "needs lawyer review"

  - finding_id: LEG-077
    surface_id: md-onb-046
    lens: legal
    severity: P1
    issue: "'Vehicle RC (optional)' — RC is regulatory document; making it optional in onboarding while required for motorised-vehicle delivery is contradiction"
    evidence_excerpt: "Vehicle RC (optional)"
    recommendation: "RC (Registration Certificate) is mandatory under MV Act §39 to operate any motor vehicle. Marking RC 'optional' is OK ONLY IF the driver-vehicle pair is bicycle (no MV registration). Lawyer to confirm conditional optionality is enforced; if motorised-vehicle driver omits RC, that is an MV Act §39 issue plus platform §94 (CP Act due-diligence) issue."
    citation: "Motor Vehicles Act 1988 §39; CP Act 2019 §94"
    depends_on: "needs lawyer review"

  - finding_id: LEG-078
    surface_id: md-onb-053
    lens: legal
    severity: P1
    issue: "'Upload PDF' as document-upload — PDF can carry embedded metadata/PII; no scrubbing disclosure"
    evidence_excerpt: "Upload PDF"
    recommendation: "PDFs often contain author name, software, timestamps in metadata. For KYC documents this can leak additional PII. Best-practice: server-side strip metadata before storage. Lawyer to confirm metadata-handling policy in Privacy Policy."
    citation: "DPDP §8(7) storage limitation; best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-079
    surface_id: md-onb-054
    lens: legal
    severity: P1
    issue: "'Camera permission is required to capture documents' — DPDP requires purpose-specific consent for sensor access; platform-specific permission strings also needed"
    evidence_excerpt: "Camera permission is required to capture documents."
    recommendation: "Apple Info.plist (NSCameraUsageDescription) and Android Manifest permission rationale must mirror this. The user-facing rationale should specifically say what photo is captured for (KYC docs only). Currently broad. Lawyer + dev to confirm app store privacy manifests match."
    citation: "Apple Info.plist NSCameraUsageDescription; Google Play permission policy; DPDP §6"
    depends_on: "needs lawyer review"

  - finding_id: LEG-080
    surface_id: md-onb-058
    lens: legal
    severity: P1
    issue: "'Please upload Driving License and ID Proof to continue' — gating requires DL but driver may be bicycle (DL not required)"
    evidence_excerpt: "Please upload Driving License and ID Proof to continue."
    recommendation: "Inconsistent with dp-auth-step3-bicycle-hint which says bicycle drivers don't need DL. Mobile flow must mirror this conditional logic. Currently gating is universal — would force bicycle drivers to upload DL or be blocked, which is misrepresentation of requirements."
    citation: "Indian Contract Act §17 misrepresentation; consistency requirement"
    depends_on: "needs lawyer review"

  - finding_id: LEG-081
    surface_id: md-onb-059
    lens: legal
    severity: P0
    issue: "'Payout Details' mobile screen — same RBI PA MD concerns as dp-auth-step-payout-bank-fields (LEG-035)"
    evidence_excerpt: "Payout Details"
    recommendation: "Same as LEG-035. RBI PA MD §8.5/§10 disclosures, name-match verification, escrow disclosure. Mobile parity required."
    citation: "RBI PA MD 2020 §8.5, §10"
    depends_on: "needs lawyer review"

  - finding_id: LEG-082
    surface_id: md-onb-063
    lens: legal
    severity: P0
    issue: "Mobile bank account collection without on-device security disclosure (autofill, screen recording, screenshot)"
    evidence_excerpt: "Account Number"
    recommendation: "On mobile, account number entry is exposed to clipboard managers, screenshot apps, screen-recording on Android. Set `secureTextEntry`, disable iOS keyboard autofill where appropriate, and disclose in Privacy Policy. Lawyer to confirm policy text. Note: copy 'Your payout details are encrypted and never stored on this device' (md-onb-079) is a binding factual claim — if app caches anything (form-state, Redux, AsyncStorage), this is a misrepresentation."
    citation: "DPDP §8(5) reasonable security; Indian Contract Act §17 misrepresentation"
    depends_on: "needs lawyer review"

  - finding_id: LEG-083
    surface_id: md-onb-079
    lens: legal
    severity: P0
    issue: "'Your payout details are encrypted and never stored on this device' — binding factual claim; if React state, Redux, or AsyncStorage caches the value, claim is false"
    evidence_excerpt: "Your payout details are encrypted and never stored on this device. They are used solely for processing your earnings."
    recommendation: "URGENT: dev to verify literal truth of this claim — React component state persists at least transiently; if it's saved through any form-draft mechanism or app-backgrounding restore, the claim is false. False security claims attract CP Act §2(28) misleading advertisement and DPDP §8(5) (claim of security practices that don't exist). Lawyer to review verbatim text against actual storage behavior."
    citation: "CP Act 2019 §2(28); DPDP §8(5); Indian Contract Act §17"
    depends_on: "needs lawyer review"

  - finding_id: LEG-084
    surface_id: md-onb-070
    lens: legal
    severity: P1
    issue: "IFSC Code collection — needs RBI/IFSC validation, no validation disclosure"
    evidence_excerpt: 'IFSC Code + "e.g. HDFC0001234"'
    recommendation: "IFSC validation (RBI lookup) typically happens server-side. If validation fails server-side after user submits, the per-field error display must be clear. Per DPDP §8(4) accuracy, IFSC should be validated, not just regex-checked. Lawyer to confirm penny-drop + IFSC lookup."
    citation: "RBI PA MD 2020 §8.5; DPDP §8(4)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-085
    surface_id: md-onb-076
    lens: legal
    severity: P1
    issue: "UPI ID collection — same NPCI VPA verification concern as dp-auth-step-payout-upi (LEG-036)"
    evidence_excerpt: "UPI ID"
    recommendation: "Same as LEG-036. Validate VPA resolves to expected name. Disclosure on mobile."
    citation: "NPCI UPI Procedural Guidelines; DPDP §8(4)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-086
    surface_id: md-onb-082
    lens: legal
    severity: P0
    issue: "Mobile 'Choose Your Plan' subscription screen — same dark-pattern subscription-trap concerns as dp-auth-step4-h"
    evidence_excerpt: "Choose Your Plan"
    recommendation: "Mobile flow inherits all subscription concerns LEG-041–LEG-046 — '100% earnings' claim, free-trial subscription trap, auto-billing trigger, gateway disclosure. Apple and Google Play also impose their own subscription disclosure rules (App Store §3.1.2 auto-renewable subscriptions, Google Play subscription policy). Lawyer + dev to verify mobile subscription terms also comply with store policies."
    citation: "CCPA Dark Patterns Guidelines 2023; Apple §3.1.2; Google Play subscription policy"
    depends_on: "needs lawyer review"

  - finding_id: LEG-087
    surface_id: md-onb-093
    lens: legal
    severity: P0
    issue: "Mobile review screen + Terms-acceptance gate parallels dp-auth-step5-terms (LEG-047, LEG-048)"
    evidence_excerpt: "Review Your Application"
    recommendation: "Same as LEG-047 (T&C links must be wired) and LEG-048 (unbundled consent). Mobile parity required."
    citation: "Indian Contract Act §13; DPDP §6(1)(d)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-088
    surface_id: md-onb-103
    lens: legal
    severity: P0
    issue: "'I accept the Terms of Service and Privacy Policy' — single checkbox bundles two consents"
    evidence_excerpt: "I accept the Terms of Service and Privacy Policy"
    recommendation: "Same as LEG-048: per DPDP §6, Terms of Service (contract) and Privacy Policy (data processing) must be separate consent items. Two checkboxes."
    citation: "DPDP Act 2023 §6(1); Indian Contract Act §13"
    depends_on: "needs lawyer review"

  - finding_id: LEG-089
    surface_id: md-onb-103
    lens: legal
    severity: P0
    issue: "T&C and Privacy Policy strings in mobile review have no hyperlink behavior confirmed — same wiring concern as web"
    evidence_excerpt: "I accept the Terms of Service and Privacy Policy"
    recommendation: "Mobile devs to confirm tappable links open in-app browser to live policies. Without tappable links, user has no access to the documents being accepted. Same Indian Contract Act §13 informed-consent issue."
    citation: "Indian Contract Act §13; DPDP §6(1)(d)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-090
    surface_id: md-onb-109
    lens: legal
    severity: P1
    issue: "'Application Not Approved' + 'Unfortunately your application was not approved at this time.' — no DPDP §11(2) reason-for-decision disclosure or appeal pathway"
    evidence_excerpt: "Application Not Approved / Unfortunately your application was not approved at this time."
    recommendation: "Driver has right to know reason for adverse decision. Currently 'Reason: {rejectionReason}' (md-onb-111) follows but inventory does not confirm grievance/appeal path. Add: 'Disagree? Contact our Grievance Officer at [contact].' DPDP §13(2) and CP Act §35."
    citation: "DPDP §11(2), §13(2); CP Act §35"
    depends_on: "needs lawyer review"

  - finding_id: LEG-091
    surface_id: md-onb-115
    lens: legal
    severity: P1
    issue: "'Estimated review time: 24–48 hours' — same SLA-promise concern as dp-auth-status-descriptions (LEG-051) but tighter window (24-48h vs 1-2 business days)"
    evidence_excerpt: "Estimated review time: 24–48 hours"
    recommendation: "Tighter SLA window (24-48 hrs) on mobile contradicts portal (1-2 business days). Inconsistency between portals is a misrepresentation risk. Pick one, align everywhere, and add 'usually' qualifier."
    citation: "Indian Contract Act §17; CP(E-Commerce) Rules 2020 §5(3) consistency"
    depends_on: "needs lawyer review"

  - finding_id: LEG-092
    surface_id: md-onb-116
    lens: legal
    severity: P2
    issue: "'We'll notify you once your application is approved' — notification commitment without channel disclosure"
    evidence_excerpt: "We'll notify you once your application is approved. This page checks for updates automatically every 30 seconds."
    recommendation: "What channel? Push notification, email, SMS, in-app? TRAI rules for SMS, DPDP §6 specificity for push. Add: 'We'll send a push notification and email.' Confirm with engineering."
    citation: "TRAI TCCCPR 2018; DPDP §6"
    depends_on: "needs lawyer review"

  - finding_id: LEG-093
    surface_id: md-onb-014
    lens: legal
    severity: P1
    issue: "Emergency contact phone '10-digit mobile number' — same third-party-consent concern as LEG-037"
    evidence_excerpt: "10-digit mobile number"
    recommendation: "Same as LEG-037. Mobile parity."
    citation: "DPDP §6(1); §5(i)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-094
    surface_id: md-onb-018
    lens: legal
    severity: P2
    issue: "DOB placeholder 'MM/DD/YYYY' inconsistent with India-only validation context — minor PII collection consistency issue"
    evidence_excerpt: "MM/DD/YYYY"
    recommendation: "India uses DD/MM/YYYY. Inconsistency is a UX bug but also a data-quality issue (DPDP §8(4) accuracy) — if users enter their date in their local-format expectation while UI shows MM/DD/YYYY, dates may be parsed wrong. Lawyer angle is minor; engineering fix primary."
    citation: "DPDP §8(4) accuracy"
    depends_on: "needs lawyer review"

  - finding_id: LEG-095
    surface_id: md-onb-015
    lens: legal
    severity: P3
    issue: "'Enter a valid 10-digit Indian mobile number' — explicit India-only validation; needs confirmation Indian-only is intended"
    evidence_excerpt: 'error: "Enter a valid 10-digit Indian mobile number"'
    recommendation: "If platform is India-only (per project CLAUDE.md), this is correct. If platform aspires to non-India, validation excludes valid foreign numbers. Lawyer to confirm jurisdictional scope statement in T&C matches India-only validation."
    citation: "consistency; CP Act §94"
    depends_on: "needs lawyer review"

  - finding_id: LEG-096
    surface_id: md-onb-038
    lens: legal
    severity: P1
    issue: "'Driving License Number' field — manual entry vs DigiLocker pull"
    evidence_excerpt: "Driving License Number"
    recommendation: "Manual DL number entry creates accuracy risk (DPDP §8(4)) and fraud risk. Best-practice: DigiLocker-backed pull or mParivahan API verification. Currently regex-only check ('Min 8 characters'). Lawyer to confirm backend verifies DL against authoritative source before driver approval."
    citation: "DPDP §8(4); Motor Vehicles Act §3"
    depends_on: "needs lawyer review"

  - finding_id: LEG-097
    surface_id: md-onb-035
    lens: legal
    severity: P1
    issue: "'Vehicle Registration Number' — regex-only check (e.g. MH12AB1234); no MV Act verification"
    evidence_excerpt: 'Vehicle Registration Number + "e.g. MH12AB1234"'
    recommendation: "Per MV Act §39 + §192, motor vehicle must be registered. Format check is insufficient — backend should verify with mParivahan/VAHAN before approval. Otherwise platform onboards drivers using fake/expired registration. Lawyer to confirm verification step."
    citation: "Motor Vehicles Act §39, §192; CP Act §94"
    depends_on: "needs lawyer review"

  # =============================================================================
  # CROSS-CUTTING / META
  # =============================================================================

  - finding_id: LEG-098
    surface_id: vp-auth-forgot-redirect
    lens: legal
    severity: P3
    issue: "'Redirecting to password reset...' has stale Keycloak code comment per inventory note; not directly user-visible but indicates auth-flow drift"
    evidence_excerpt: "Redirecting to password reset..."
    recommendation: "Stale references to deprecated auth provider (Keycloak) in code comments are not directly a legal issue but signal possible drift in auth flow. Lawyer angle: confirm password-reset emails accurately identify current data fiduciary, not Keycloak. Dev cleanup recommended."
    citation: "DPDP §5(i) data fiduciary identification; best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-099
    surface_id: dp-auth-login-title
    lens: legal
    severity: P3
    issue: "'Fe3dr Delivery' brand wordmark — confirm trademark protection and consistency"
    evidence_excerpt: "Fe3dr Delivery"
    recommendation: "Confirm 'Fe3dr' is registered trademark per Trade Marks Act 1999. Brand inconsistency between Fe3dr (apps) and Mark8ly (CLAUDE.md domain) is a brand-IP question. Lawyer to confirm both marks are registered/in-use and clear."
    citation: "Trade Marks Act 1999"
    depends_on: "needs lawyer review"

  - finding_id: LEG-100
    surface_id: dp-auth-login-title
    lens: legal
    severity: P0
    issue: "Across web, vendor-portal, delivery-portal, and three mobile apps — there is NO visible cookie/tracking consent banner on or before any auth-onboarding surface"
    evidence_excerpt: "(missing entirely)"
    recommendation: "DPDP §6 (consent before processing) + draft DPDP Rules 2025 require cookie/tracking consent before non-essential cookies fire. Auth pages typically load analytics (PostHog, OpenPanel, GA), session-replay (FullStory analogues), and OAuth provider tracking. Currently no cookie banner is inventoried on any auth screen. URGENT cross-app: implement DPDP-compliant cookie consent (granular: essential / analytics / marketing), pre-display, with reject-all as easy as accept-all. Lawyer to draft banner copy."
    citation: "DPDP Act 2023 §6; draft DPDP Rules 2025; CCPA Dark Patterns Guidelines 2023 (consent design)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-101
    surface_id: web-auth-register-form-fields
    lens: legal
    severity: P1
    issue: "No CAPTCHA/bot-protection disclosure on signup — bot signup creates compliance liability if fake accounts proliferate"
    evidence_excerpt: "(missing entirely)"
    recommendation: "Best-practice and DPDP §8(5) reasonable security imply protection against automated account creation. If reCAPTCHA/hCaptcha is used, Privacy Policy must disclose Google/hCaptcha as a third-party processor. If not used, lawyer concern around fake-account proliferation. Confirm with engineering."
    citation: "DPDP §8(5); §10 third-party disclosure"
    depends_on: "needs lawyer review"

  - finding_id: LEG-102
    surface_id: web-auth-register-form-fields
    lens: legal
    severity: P1
    issue: "Email verification flow not inventoried — confirming email controls account-takeover risk and DPDP accuracy"
    evidence_excerpt: "(missing entirely)"
    recommendation: "Best-practice: email verification before any meaningful action (especially payouts). Without verification, fraudster can create accounts at victim's email. Per DPDP §8(4) accuracy and §8(5) security, email verification is reasonable security. Confirm in code that verification email is sent and account remains gated until verified."
    citation: "DPDP §8(4), §8(5); best-practice account security"
    depends_on: "needs lawyer review"

  - finding_id: LEG-103
    surface_id: vp-onb-stepper-labels
    lens: legal
    severity: P0
    issue: "Across vendor-portal AND mobile-vendor onboarding — there is no visible disclosure of platform commission rate before the chef commits to onboarding"
    evidence_excerpt: "Personal Info / Kitchen Details / Operations / Documents / Policies & Review"
    recommendation: "Platform commission is the single most material economic term of the Vendor Agreement. Per CP(E-Commerce) Rules 2020 §5(3)(c) and Indian Contract Act §29 (certainty), this must be disclosed BEFORE acceptance. StepPolicies says 'Fe3dr charges a platform commission on each order' without naming the percentage. Lawyer to confirm: actual commission percentage is pre-disclosed in onboarding (Stepper preview or Step 0), not buried in T&C only."
    citation: "CP(E-Commerce) Rules 2020 §5(3)(c); Indian Contract Act §29"
    depends_on: "needs lawyer review"

  - finding_id: LEG-104
    surface_id: dp-auth-onboarding-header
    lens: legal
    severity: P0
    issue: "Cross-driver-onboarding: no disclosure of platform-side insurance arrangement (third-party liability for accidents during delivery)"
    evidence_excerpt: "Become a Delivery Partner / Complete your profile to start delivering"
    recommendation: "Karnataka PBGW Bill 2024 §13 and Rajasthan PBGW Act 2023 §11 require platforms to disclose insurance arrangements (accident, health) to gig workers. Currently no disclosure visible in driver onboarding. Lawyer to confirm: (a) does platform provide insurance, (b) what's covered (accidental death, medical, third-party liability), (c) what driver pays/contributes. Major gig-worker disclosure gap."
    citation: "Rajasthan PBGW Act 2023 §11; Karnataka PBGW Bill 2024 §13; Code on Social Security 2020 §114"
    depends_on: "needs lawyer review"

  - finding_id: LEG-105
    surface_id: web-auth-register-heading
    lens: legal
    severity: P1
    issue: "Across all six apps — no language-choice disclosure or vernacular T&C availability"
    evidence_excerpt: "(missing entirely)"
    recommendation: "India is multilingual. Per CP(E-Commerce) Rules 2020 §6(1)(f) and best-practice consumer law, key contractual terms should be available in the language a reasonable consumer understands. Currently English-only inventory. Lawyer to confirm: (a) policy in Hindi + major regional languages, (b) signup screen offers language toggle, (c) T&C acceptance in non-English language is enforceable. STYLE-GUIDE §7 notes i18n is out of scope for audit but flagged here as legal exposure."
    citation: "CP(E-Commerce) Rules 2020 §6(1)(f); CP Act §2(28); best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-106
    surface_id: web-auth-register-form-fields
    lens: legal
    severity: P1
    issue: "Account deletion / DPDP §12 erasure right — no visible path from signup/onboarding to deletion mechanism"
    evidence_excerpt: "(missing entirely)"
    recommendation: "Per DPDP §12, data principal has right to erasure. Per §5(iii)–(iv), this right must be disclosed at consent. Currently the signup flow does not mention erasure or how to exercise it. Lawyer to confirm Privacy Policy describes erasure procedure (in-app + email channel), and that the consent notice references it."
    citation: "DPDP Act 2023 §12; §5(iii)"
    depends_on: "needs lawyer review"
```

## Business Analyst findings

```yaml
# BA Lens — AUTH-ONBOARDING findings
# Auditor: business-analyst lens
# Date: 2026-05-13
# Scope: apps/web auth + vendor-portal auth+onboarding + delivery-portal auth+onboarding

findings:

  # ─────────────────────────────────────────────────────────────────────────
  # FAKE / UNVERIFIED TRUST SIGNALS — P0
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BA-001
    surface_id: web-auth-register-benefits
    lens: business-analyst
    severity: P0
    issue: "Hardcoded '500+ home chefs' claim on customer register page — fake metric"
    evidence_excerpt: "const BENEFITS = ['Access to 500+ home chefs', ...]"
    recommendation: >
      Replace with live count from API, e.g. "Order from home chefs near you." or
      render the count dynamically if the number is real. If the platform is early-stage
      and 500+ is aspirational, remove the number entirely. The framing "near you" is
      stronger — it sets delivery-zone expectations, not false scale.
    metric_hypothesis: "trust score; customers who discover 500+ is hardcoded will churn and leave negative reviews"
    depends_on: null

  - finding_id: BA-002
    surface_id: vendor-portal-register-benefits
    lens: business-analyst
    severity: P0
    issue: "Hardcoded 'Join thousands of home chefs' on vendor register page — fake social proof"
    evidence_excerpt: "Join thousands of home chefs earning with Fe3dr. Your kitchen, your recipes, your rules."
    recommendation: >
      Remove the unquantified "thousands" claim unless backed by live data. Replace with
      a concrete, verifiable value statement: "Start earning from your kitchen. Zero
      commission for your first month." This anchors the benefit (concrete, defensible)
      instead of manufacturing social proof.
    metric_hypothesis: "trust score; chef applicants who verify the claim is false will abandon and post negative word-of-mouth"
    depends_on: null

  - finding_id: BA-003
    surface_id: web-auth-login-testimonial
    lens: business-analyst
    severity: P0
    issue: "Fake testimonial ('Sarah M., Happy Customer') hardcoded on login page hero panel"
    evidence_excerpt: |
      "Fe3dr has changed how I eat. Finally, real homemade food that reminds me of my mom's cooking!"
      <footer><p>Sarah M.</p><p>Happy Customer</p></footer>
    recommendation: >
      Either (a) replace with a real verified customer review pulled from API with name,
      photo, and verified-purchase badge; or (b) remove the testimonial entirely and let
      the food photography speak. A fabricated quote is legally a false endorsement and
      undermines the brand's "trust" promise. "Happy Customer" as a role title is also
      content-lite — it signals nothing.
    metric_hypothesis: "trust score; customers who recognize the pattern of fake testimonials will abandon signup"
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # CUSTOMER REGISTER PAGE
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BA-004
    surface_id: web-auth-register-heading
    lens: business-analyst
    severity: P1
    issue: "Register page H1 'Create your account' is utility language — no value prop for a hungry customer"
    evidence_excerpt: "Create your account"
    recommendation: >
      For the customer register page, the headline should communicate why now, not what
      to do. Example: "Your neighbourhood home chef is 20 minutes away." with subline
      "Create an account to order." This gives the user a reason before asking for effort.
    metric_hypothesis: "register page completion rate; value-prop-first headlines improve form start rates"
    depends_on: null

  - finding_id: BA-005
    surface_id: web-auth-register-benefits
    lens: business-analyst
    severity: P2
    issue: "Benefits panel copy is generic — 'Authentic homemade food' and 'Support local home chefs' could appear on any food app"
    evidence_excerpt: "'Authentic homemade food', 'Fast & reliable delivery', 'Support local home chefs'"
    recommendation: >
      Replace generic bullets with differentiating proof points. Example:
      "Cook-to-order — no pre-made batches sitting on shelves."
      "Know exactly who made your food — every chef has a verified profile."
      "Delivery from 1 km away, not a central cloud kitchen."
      Each bullet should answer: why Fe3dr, not Swiggy?
    metric_hypothesis: "register page → first order conversion; differentiated value props increase trial intent"
    depends_on: null

  - finding_id: BA-006
    surface_id: web-auth-register-email-form
    lens: business-analyst
    severity: P2
    issue: "5-field email registration form (First name, Last name, Email, Password, Confirm password) creates unnecessary friction at signup"
    evidence_excerpt: "const [firstName, setFirstName] = useState(''); const [lastName, setLastName] = useState(''); const [email, setEmail] = useState('');"
    recommendation: >
      For customer signup, reduce to email + password (2 fields). Collect first/last name
      post-signup during the address/profile setup flow. Splitting first/last name into two
      fields adds friction for no immediate UX gain — the app can prompt for full name before
      first order. Standard benchmark: every additional required field drops completion ~8-12%.
    metric_hypothesis: "signup completion rate; reducing fields from 5 to 2 typically lifts completion 25-40%"
    depends_on: null

  - finding_id: BA-007
    surface_id: web-auth-register-email-form
    lens: business-analyst
    severity: P2
    issue: "Password confirmation field adds friction but no user-visible strength indicator or rule explanation before submission"
    evidence_excerpt: "'Min. 8 characters' (placeholder only), error only fires post-submit 'Password must be at least 8 characters'"
    recommendation: >
      Either (a) replace the confirm-password field with real-time password strength
      indicator (shows rules inline: green tick at 8+ chars) — industry data shows this
      increases completion vs. a second field; or (b) keep the field but add inline
      validation on blur so users don't discover errors only after hitting submit.
    metric_hypothesis: "signup form completion rate; inline validation reduces abandonment on password fields by 15-20%"
    depends_on: null

  - finding_id: BA-008
    surface_id: web-auth-register-tos
    lens: business-analyst
    severity: P2
    issue: "Terms of Service consent is a tiny 12px text block — no explicit 'data use' reassurance for signup"
    evidence_excerpt: "By signing up, you agree to our Terms of Service and Privacy Policy"
    recommendation: >
      Add a single reassurance line above or below: "We never sell your data or spam you.
      Your email is only used for order updates." This is not a legal disclaimer (Legal
      lens owns that) but a trust sentence that reduces signup abandonment driven by
      privacy concerns. Research consistently shows 15-30% of form abandonment is
      privacy-motivated.
    metric_hypothesis: "signup completion rate; explicit no-spam reassurance reduces abandonment at the consent step"
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # CUSTOMER LOGIN PAGE
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BA-009
    surface_id: web-auth-login-heading
    lens: business-analyst
    severity: P3
    issue: "'Welcome back' has no function pointer — nothing below it explains what the user is signing back in to do"
    evidence_excerpt: "Welcome back / Don't have an account? Sign up"
    recommendation: >
      Replace the subline "Don't have an account? Sign up" with a value-reminder for
      returning users: "Your orders are waiting." or "Your home chefs are ready." The
      signup link can live at the bottom of the form where it is lower-friction to discover.
      Showing it as the second sentence at the top promotes alternatives before the primary
      goal (login) is even attempted.
    metric_hypothesis: "login → order conversion; sub-headline optimisation for returning users"
    depends_on: null

  - finding_id: BA-010
    surface_id: web-auth-login-error
    lens: business-analyst
    severity: P2
    issue: "Generic auth error message 'Something went wrong. Please try again.' on non-session-expired OAuth failures gives no actionable path"
    evidence_excerpt: "Something went wrong. Please try again."
    recommendation: >
      Differentiate OAuth failure from credential failure. If OAuth fails: "We couldn't
      reach Google/Facebook. Try again, or sign in with email." This gives users a
      concrete workaround and stops them abandoning entirely. The current copy is a dead end.
    metric_hypothesis: "login completion rate after error; actionable error messages recover 20-30% of error-state users"
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # VENDOR PORTAL — REGISTER PAGE
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BA-011
    surface_id: vendor-portal-register-heading
    lens: business-analyst
    severity: P1
    issue: "'Register your kitchen' as H1 is process-oriented, not value-oriented — no income promise in the first fold"
    evidence_excerpt: "Register your kitchen / Create your vendor account to start selling home-cooked meals"
    recommendation: >
      Leads with income, not paperwork. Example H1: "Start earning from your kitchen."
      Subline: "Create your account — takes 2 minutes." This reframes registration as the
      first step to income, not a compliance form. For a chef deciding between Fe3dr and
      Swiggy Minis, income framing is the primary differentiator.
    metric_hypothesis: "vendor signup completion rate; income-first framing lifts chef supply acquisition"
    depends_on: null

  - finding_id: BA-012
    surface_id: vendor-portal-register-benefits
    lens: business-analyst
    severity: P1
    issue: "'Zero commission first month' benefit is buried 3rd item on the panel with no monetary anchor — chef doesn't know what normal commission is"
    evidence_excerpt: "{ title: 'Zero commission first month', desc: 'Get started completely risk-free' }"
    recommendation: >
      Make the zero-commission benefit the first item AND anchor the normal commission rate
      so the chef can calculate the savings. Example: "0% commission — for life. We charge
      a flat subscription, never a cut of your orders." If the platform does take eventual
      commission, be explicit: "0% commission for your first month, then X%." Hiding the
      transition erodes trust when discovered.
    metric_hypothesis: "vendor signup completion rate and D30 retention; income transparency drives quality chef supply"
    depends_on: null

  - finding_id: BA-013
    surface_id: vendor-portal-register-tos
    lens: business-analyst
    severity: P1
    issue: "Vendor registration TOS consent 'By registering, you agree to Fe3dr's Vendor Terms and Privacy Policy' — no links, no actionable text"
    evidence_excerpt: "By registering, you agree to Fe3dr's Vendor Terms and Privacy Policy"
    recommendation: >
      The words "Vendor Terms" and "Privacy Policy" must be hyperlinks (they are not — the
      text is plain). A chef who cannot read what they are agreeing to before registering
      may abandon. Add links and add a one-line reassurance: "We never share your personal
      details with customers." Missing links also create a legal exposure (Legal lens should
      flag separately).
    metric_hypothesis: "vendor signup completion rate; non-linked consent blocks legally cautious chefs"
    depends_on: null

  - finding_id: BA-014
    surface_id: vendor-portal-register-email-cta
    lens: business-analyst
    severity: P2
    issue: "Email registration CTA button reads 'Register with Email' — inconsistent with 'Sign up with Google/Facebook' pattern on same page"
    evidence_excerpt: "'Register with Email' vs 'Sign up with Google' / 'Sign up with Facebook'"
    recommendation: >
      Normalise to 'Sign up with email' across all three buttons. Style-guide mandates
      'Sign up' (verb form, two words). 'Register' is vocabulary that maps to bureaucratic
      process; 'Sign up' is lighter. Consistency across three CTA buttons also reduces
      cognitive friction when scanning the page.
    metric_hypothesis: "vendor signup click-through; consistent CTA vocabulary reduces decision paralysis"
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # VENDOR PORTAL — LOGIN PAGE
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BA-015
    surface_id: vendor-portal-login-heading
    lens: business-analyst
    severity: P2
    issue: "Vendor login subline 'Sign in to manage your menu, orders, and earnings' is task-list, not motivating"
    evidence_excerpt: "Sign in to manage your menu, orders, and earnings"
    recommendation: >
      Chef persona tone guide: functional, time-aware, operational. Better:
      "You have orders waiting." (if session had prior activity) or contextually:
      "Pick up where you left off." This is shorter, more urgent, and respects the
      chef's time pressure. The list of three things (menu, orders, earnings) is
      redundant — they know what the portal does.
    metric_hypothesis: "vendor login completion rate; time-aware microcopy for chefs reduces bounce"
    depends_on: null

  - finding_id: BA-016
    surface_id: vendor-portal-login-email-cta
    lens: business-analyst
    severity: P2
    issue: "Email login button reads 'Sign in with Email' — sentence-cased but capitalises 'Email' (title case inconsistency)"
    evidence_excerpt: "'Sign in with Email'"
    recommendation: >
      Style guide mandates sentence case for buttons: 'Sign in with email'. 'Email' is
      not a proper noun in this context. Consistent casing signals polish and reduces
      micro-friction that accumulates across a page.
    metric_hypothesis: "vendor login click-through; casing consistency is a polish signal that builds subconscious trust"
    depends_on: null

  - finding_id: BA-017
    surface_id: vendor-portal-login-access-denied
    lens: business-analyst
    severity: P1
    issue: "Access-denied error 'This portal is only for vendor accounts. Please use the Fe3dr customer app.' gives no link or recovery path"
    evidence_excerpt: "This portal is only for vendor accounts. Please use the Fe3dr customer app."
    recommendation: >
      Add a direct link to the customer app or the customer login URL. A chef who
      accidentally tried to log in with a customer account is still in the funnel — the
      current copy dead-ends them. Better: "This portal is for chefs only. Order food?
      [Go to the customer app →]" with a real link. This recovers cross-portal misdirects
      instead of losing them entirely.
    metric_hypothesis: "error-state recovery rate; dead-end errors are unrecoverable abandonment"
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # VENDOR ONBOARDING WIZARD — GENERAL
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BA-018
    surface_id: vendor-portal-onboarding-header
    lens: business-analyst
    severity: P1
    issue: "Wizard header copy 'Set Up Your Kitchen' + 'Step 3 of 5' gives no estimated completion time — chefs abandon unknown-length forms"
    evidence_excerpt: "'Set Up Your Kitchen' / 'Step ${currentStep + 1} of ${TOTAL_DISPLAY_STEPS}'"
    recommendation: >
      Add time estimate to the header or below the progress indicator: "About 8 minutes
      to complete." Disclosure of expected time is the single highest-leverage change for
      multi-step form completion — meta-analyses show 25-35% reduction in abandonment when
      time is disclosed. Chef persona is time-pressured; they need to know it's worth
      starting before the lunch rush.
    metric_hypothesis: "wizard start-to-complete rate; time-disclosure is the most consistently effective form-completion intervention"
    depends_on: null

  - finding_id: BA-019
    surface_id: vendor-portal-onboarding-stepper
    lens: business-analyst
    severity: P2
    issue: "Step 5 label 'Policies & Review' combines two distinct concerns in one step label — chef doesn't know which steps involve paperwork vs. review"
    evidence_excerpt: "{ title: 'Policies & Review', description: 'Agreements & submit', icon: Shield }"
    recommendation: >
      The wizard actually has a separate review screen (StepReview) after StepPolicies.
      The stepper should reflect this honestly: Step 5 = 'Agreements', Step 6 = 'Review
      & submit'. Or relabel Step 5 as 'Agreements' with description 'Food safety checklist',
      and 'Review' as a distinct step. Misrepresenting steps damages trust when chefs
      discover there is more than they expected.
    metric_hypothesis: "wizard completion rate; honest step labelling reduces drop-off on last step surprise"
    depends_on: null

  - finding_id: BA-020
    surface_id: vendor-portal-onboarding-wizard
    lens: business-analyst
    severity: P1
    issue: "Mobile step indicator shows numeric fraction '3/6' only — no step label on mobile, leaving chef without context for what they are completing"
    evidence_excerpt: "<span>{showReview ? `${TOTAL_DISPLAY_STEPS + 1}` : `${currentStep + 1}`}/{TOTAL_DISPLAY_STEPS + 1}</span>"
    recommendation: >
      Mobile indicator must show the current step name alongside the fraction.
      Current code shows only the fraction. The STEPS[currentStep]?.title is already
      available and is already rendered for the left side — but only the fraction is
      shown right-aligned. The fraction alone gives no semantic context: "3/6" is less
      meaningful than "Step 3 of 6 — Kitchen Details". Chef cannot orient themselves on
      mobile.
    metric_hypothesis: "mobile wizard completion rate; step labels reduce disorientation abandonment on mobile"
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # VENDOR ONBOARDING — STEP 0: PERSONAL INFO
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BA-021
    surface_id: vendor-portal-onboarding-step-personal
    lens: business-analyst
    severity: P1
    issue: "Personal Info step has 8 required fields visible at once (full name, phone, email, address line 1, country, state, city, PIN) — field overload in step 1"
    evidence_excerpt: "if (!data.fullName.trim()) errors.fullName = ... if (!data.phone.trim()) ... (6 required address subfields)"
    recommendation: >
      Split Step 1 into two sub-steps or progressive disclosure: (a) personal identity
      (name, phone, email — 3 fields), then (b) kitchen address (unlocked after personal
      info saves). Alternatively, default-fill country=India and auto-detect state/city
      from PIN code to collapse 4 address fields into 1. Reducing perceived field count
      on the first step is critical — first step abandonment is the highest-volume drop-off.
    metric_hypothesis: "wizard step-1 completion rate; first-step field count is the primary predictor of wizard abandonment"
    depends_on: null

  - finding_id: BA-022
    surface_id: vendor-portal-onboarding-step-personal
    lens: business-analyst
    severity: P2
    issue: "Address Line 2 and Landmark are separate optional fields — creates clutter alongside 3 required dropdowns"
    evidence_excerpt: "'Address Line 2 (Optional)', 'Landmark (Optional)', plus Country, State, City required dropdowns"
    recommendation: >
      Collapse into a single free-text "Landmark or area (optional)" field. Two optional
      address fields increase perceived form length without proportionate data quality gain.
      The landmark field is user-friendly (familiar in India) but two fields for the same
      concept signals poor UX design to a chef who is trying to cook, not fill government
      forms.
    metric_hypothesis: "step-1 completion rate; reducing optional field count decreases perceived form length"
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # VENDOR ONBOARDING — STEP 1: KITCHEN DETAILS
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BA-023
    surface_id: vendor-portal-onboarding-step-kitchen
    lens: business-analyst
    severity: P2
    issue: "Description field hint 'Min 20 characters. This is shown to customers on your profile.' provides no example of what good looks like"
    evidence_excerpt: "hint='Min 20 characters. This is shown to customers on your profile.'"
    recommendation: >
      Add a concrete example inline: "e.g. 'I've been cooking Gujarati thali for 15 years.
      Everything is freshly made the morning of your order.'" Chefs who stall on
      open-ended text fields are a major drop-off point. Showing what good looks like
      unlocks progress.
    metric_hypothesis: "step-2 completion rate; example copy in free-text fields reduces stall abandonment"
    depends_on: null

  - finding_id: BA-024
    surface_id: vendor-portal-onboarding-step-kitchen
    lens: business-analyst
    severity: P2
    issue: "'Years of Cooking Experience' select defaults to empty 'Select experience' — no business rationale communicated to chef why this is asked"
    evidence_excerpt: "<option value=''>Select experience</option>"
    recommendation: >
      Add a helper line below the field: "Shown on your profile — customers use this to
      find experienced chefs." This explains why the information is collected and what the
      chef gains by providing it (profile completeness = more orders). Without rationale,
      discretionary fields that feel nosy cause friction.
    metric_hypothesis: "step-2 completion rate; explaining why increases field fill rates on discretionary fields"
    depends_on: null

  - finding_id: BA-025
    surface_id: vendor-portal-onboarding-step-kitchen
    lens: business-analyst
    severity: P3
    issue: "'Meals You Can Prepare Daily' select — option '100+ meals' has no capacity implication explained"
    evidence_excerpt: "'50-100 meals', '100+ meals'"
    recommendation: >
      Selecting capacity affects how the platform routes orders. Add a brief note:
      "This helps us manage order volume so you're never overwhelmed." This framing is
      both reassuring (we protect you) and explains the business logic.
    metric_hypothesis: "chef D7 retention; chefs who are over-committed on capacity churn early"
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # VENDOR ONBOARDING — STEP 2: OPERATIONS
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BA-026
    surface_id: vendor-portal-onboarding-step-operations
    lens: business-analyst
    severity: P2
    issue: "Delivery Radius field hint 'How far can you deliver or allow pickup from' is ambiguous — chef doesn't know if they need to deliver themselves"
    evidence_excerpt: "hint='How far can you deliver or allow pickup from'"
    recommendation: >
      Clarify the logistics model before this field: "Fe3dr's delivery partners handle all
      deliveries — you just set your kitchen's service radius." Many chefs drop off here
      assuming they must personally deliver. Confirming platform delivery removes a major
      barrier.
    metric_hypothesis: "step-3 completion rate; logistics ambiguity is a primary chef onboarding drop-off driver"
    depends_on: null

  - finding_id: BA-027
    surface_id: vendor-portal-onboarding-step-operations
    lens: business-analyst
    severity: P3
    issue: "Delivery Fee field label says 'Delivery Fee' with no explanation of who sets it or whether Fe3dr overrides it"
    evidence_excerpt: "Delivery Fee / Per order delivery charge"
    recommendation: >
      Add helper text: "This is the fee customers pay per delivery. Fe3dr drivers are paid
      separately — you keep all of it." This clarifies the flow of money, which is a key
      trust question for a new chef deciding whether the platform is exploitative.
    metric_hypothesis: "chef signup completion and D30 retention; fee clarity is a top concern in seller-supply acquisition"
    depends_on: null

  - finding_id: BA-028
    surface_id: vendor-portal-onboarding-step-operations
    lens: business-analyst
    severity: P2
    issue: "Operating Hours grid shows 7 days with individual time pickers — all days start as Closed, forcing chef to click-enable every day they work"
    evidence_excerpt: "toggleDay function: if (data.operatingHours[day]) updateHours(day, undefined); else updateHours(day, { open: '09:00', close: '21:00' })"
    recommendation: >
      Pre-select common working days (Mon-Sat enabled by default) with a note "You can
      adjust or turn off any day." Most home chefs work Mon-Sat; forcing them to toggle
      6 days adds 6 unnecessary interactions. Alternatively add a "Select all weekdays"
      shortcut button.
    metric_hypothesis: "step-3 completion rate; pre-filled defaults reduce interaction count and time-on-step"
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # VENDOR ONBOARDING — STEP 3: DOCUMENTS (KYC)
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BA-029
    surface_id: vendor-portal-onboarding-step-documents
    lens: business-analyst
    severity: P1
    issue: "Documents step introduces PAN card, Aadhaar, kitchen photos — no upfront framing that explains the KYC obligation before the chef reaches this step"
    evidence_excerpt: "REQUIRED_DOCS: pan_card 'Required for tax purposes and payouts', aadhaar_card 'For identity verification'"
    recommendation: >
      Insert a single-sentence preview of this step in the Step 3 stepper description or
      on a modal before entering the step: "Step 4 requires identity documents (PAN +
      Aadhaar) and kitchen photos. Have them ready." Surfacing document requirements
      before the chef starts the wizard prevents mid-wizard abandonment when they discover
      they need to find their PAN card. The current stepper description 'ID & kitchen
      photos' is the only signal, which appears too late.
    metric_hypothesis: "wizard steps 1-3 completion rate and return rate; pre-announcing document needs allows preparation"
    depends_on: null

  - finding_id: BA-030
    surface_id: vendor-portal-onboarding-step-documents
    lens: business-analyst
    severity: P1
    issue: "Trust framing for Aadhaar collection is minimal — 'For identity verification' provides no security assurance for a sensitive government ID"
    evidence_excerpt: "aadhaar_card description: 'For identity verification'"
    recommendation: >
      Replace with: "Your Aadhaar is used only to verify you're who you say you are. It is
      encrypted at upload, never visible to customers, and not shared with third parties."
      Aadhaar is the most sensitive document an Indian citizen carries. Minimal framing
      signals either inexperience or indifference — both erode trust at the moment of
      highest friction. The existing info banner ('Your documents are encrypted and stored
      securely') exists but is above the fold; it must be adjacent to each sensitive field.
    metric_hypothesis: "step-4 document upload rate; trust framing adjacent to sensitive fields increases upload completion by 20-40%"
    depends_on: null

  - finding_id: BA-031
    surface_id: vendor-portal-onboarding-step-documents
    lens: business-analyst
    severity: P1
    issue: "Cancelled cheque optional document description 'For setting up direct payouts to your bank account (optional, can add later)' — payout setup is deferred with no clear timeline"
    evidence_excerpt: "'Cancelled Cheque / Bank Proof', description: 'For setting up direct payouts to your bank account (optional, can add later)'"
    recommendation: >
      The StepPolicies file also defers payout setup: 'You can set up your bank account
      or UPI details for receiving payouts from Settings after your kitchen is approved.'
      This creates a gap: a chef who finishes the entire wizard still does not know when
      they will receive their first payout. Add a clear SLA: "You can add bank details
      now or within 7 days of approval — your first payout processes once you add them."
      Unclear payout timelines are a top reason seller supply abandons platforms.
    metric_hypothesis: "chef D30 retention and first-payout completion rate; deferred payout setup delays first success event"
    depends_on: null

  - finding_id: BA-032
    surface_id: vendor-portal-onboarding-step-documents
    lens: business-analyst
    severity: P2
    issue: "PAN number text field validates only client-side; no inline format mask or validation feedback while typing"
    evidence_excerpt: "onChange={(e) => updateData({ panNumber: e.target.value.toUpperCase() })} hint='10-character alphanumeric PAN'"
    recommendation: >
      Add regex-based inline validation on blur: PAN format is `[A-Z]{5}[0-9]{4}[A-Z]{1}`.
      Show a green tick when the format is valid, or an inline error immediately on blur
      rather than waiting for the wizard Next button. Reducing end-of-step error surprises
      is critical for KYC step completion.
    metric_hypothesis: "step-4 validation error rate and completion rate; inline PAN validation reduces submission errors"
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # VENDOR ONBOARDING — STEP 4: POLICIES
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BA-033
    surface_id: vendor-portal-onboarding-step-policies
    lens: business-analyst
    severity: P1
    issue: "7-item compliance checklist (COMPLIANCE_ITEMS) displayed as a read-only list with no acknowledgement checkbox — chef cannot confirm they have read it"
    evidence_excerpt: "COMPLIANCE_ITEMS = ['My kitchen has access to clean running water', ...7 items] — no checkbox, just read-only text blocks"
    recommendation: >
      Either (a) consolidate the 7 items into a single acknowledgement checkbox: "I confirm
      my kitchen meets all food safety standards listed above", or (b) add a checkbox to
      each item so the chef actively ticks them. A read-only bullet list creates no
      commitment signal and no accountability. Also consider: 7 items + 3 policies = 10
      items on this step — break into two cards with sub-section labels 'Kitchen Standards'
      and 'Platform Agreements'.
    metric_hypothesis: "step-5 completion rate; active acknowledgement increases policy acceptance rate and reduces future disputes"
    depends_on: null

  - finding_id: BA-034
    surface_id: vendor-portal-onboarding-step-policies
    lens: business-analyst
    severity: P1
    issue: "Terms of Service policy checkbox description says 'Fe3dr charges a platform commission on each order' — contradicts the vendor register page promise of 'Zero commission first month'"
    evidence_excerpt: "'I understand that Fe3dr charges a platform commission on each order and that my payouts will be processed weekly'"
    recommendation: >
      This is a direct conversion-eroding conflict: the register page promises zero
      commission, the TOS checkbox says commissions are charged. The chef discovers this
      only at step 5 of a 6-step wizard, creating a trust collapse at the point of highest
      investment. Resolve the messaging conflict: either (a) update the policy description
      to reflect the actual commission model with specifics ("0% commission for your first
      month, then X% thereafter") or (b) update the register page to reflect commissions
      exist. This requires product/legal alignment but is a P1 conversion blocker.
    metric_hypothesis: "wizard final-step abandonment rate; discovering unexpected costs at step 5 causes high-intent drop-off"
    depends_on: BA-012

  - finding_id: BA-035
    surface_id: vendor-portal-onboarding-step-policies
    lens: business-analyst
    severity: P2
    issue: "Platform TOS policy checkbox links 'Terms of Service, Privacy Policy, and Vendor Agreement' are plain text — no hyperlinks to read before accepting"
    evidence_excerpt: "'I have read and agree to the Fe3dr Terms of Service, Privacy Policy, and Vendor Agreement.'"
    recommendation: >
      All three documents must be linked. A chef cannot meaningfully agree to a document
      they cannot read. This is also a legal risk (Legal lens owns this separately) but
      from a conversion perspective: legally cautious chefs will abandon rather than agree
      blind. Add links that open in a modal or new tab.
    metric_hypothesis: "step-5 acceptance rate; non-linked agreements block completion for legally aware chefs"
    depends_on: null

  - finding_id: BA-036
    surface_id: vendor-portal-onboarding-step-policies
    lens: business-analyst
    severity: P2
    issue: "Payout Details card says 'You can set up your bank account or UPI details... after your kitchen is approved' — deferred payout with no timeline creates uncertainty"
    evidence_excerpt: "'You can set up your bank account or UPI details for receiving payouts from Settings after your kitchen is approved.'"
    recommendation: >
      Add: "Payouts are processed weekly. Add your bank details within 7 days of approval
      to receive your first payout." Vague deferral ('can do it later') is less motivating
      than a concrete timeline. The chef has just completed 5 steps; giving them a clear
      success path for money receipt increases D7 activation.
    metric_hypothesis: "chef D7 activation and first-payout rate; specific payout timeline increases post-onboarding activation"
    depends_on: BA-031

  # ─────────────────────────────────────────────────────────────────────────
  # VENDOR ONBOARDING — REVIEW + SUBMIT
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BA-037
    surface_id: vendor-portal-onboarding-review
    lens: business-analyst
    severity: P2
    issue: "Review step missing documents show technical slug names 'pan_card', 'aadhaar_card' rather than human-readable names"
    evidence_excerpt: "Please go back to Step 4 and upload: {missingDocs.map((d) => d.replace(/_/g, ' ')).join(', ')}"
    recommendation: >
      The replace(/_/g, ' ') produces 'pan card' and 'aadhaar card' — lowercase, no
      capitalisation, not the field labels shown in step 4 ('PAN Card', 'Aadhaar Card').
      Use a label map (as already exists in delivery portal's StepReview docTypeLabels)
      to show 'PAN Card' and 'Aadhaar Card'. Inconsistent naming between steps creates
      cognitive mismatch.
    metric_hypothesis: "review step completion rate; clear document names reduce back-navigation confusion"
    depends_on: null

  - finding_id: BA-038
    surface_id: vendor-portal-onboarding-review
    lens: business-analyst
    severity: P1
    issue: "Submit Application toast 'Application submitted! We'll review and get back to you within 24-48 hours.' — no next action, no guidance for what to do while waiting"
    evidence_excerpt: "toast.success('Application submitted! We\\'ll review and get back to you within 24-48 hours.')"
    recommendation: >
      Redirect to an onboarding status page (like the delivery portal's OnboardingStatusPage)
      and populate it with: (a) confirmation of the 24-48 hour SLA, (b) an actionable
      step: "While you wait, start building your menu so you can launch the moment you're
      approved." (c) the payout setup CTA. A transient toast that disappears after 5
      seconds is insufficient for a high-stakes application event. The vendor portal
      lacks the status page the delivery portal already has.
    metric_hypothesis: "chef D1 and D7 activation; post-submission guidance drives pre-launch menu creation and reduces churn during wait period"
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # VENDOR ONBOARDING — REJECTION MESSAGING
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BA-039
    surface_id: vendor-portal-onboarding-rejection
    lens: business-analyst
    severity: P1
    issue: "Rejection banner heading 'Application Rejected' is blunt — no empathetic framing, no reframe that rejection is fixable"
    evidence_excerpt: "h3: 'Application Rejected' / p: 'Your previous application was not approved. Please review the feedback below and re-submit.'"
    recommendation: >
      Change heading to 'We need a few changes' and copy to: 'Your application needs a
      small update before we can approve it. Review the note below, fix the items, and
      resubmit — most applications are approved on second review.' This reframe reduces
      the finality of 'rejected', signals effort will succeed, and maintains chef in the
      funnel. 'Application Rejected' is demoralising language that guarantees a high
      proportion of rejected chefs exit permanently.
    metric_hypothesis: "rejected-chef resubmission rate; empathetic rejection framing increases resubmission by 30-50%"
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # DELIVERY PORTAL — REGISTER
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BA-040
    surface_id: delivery-portal-register-redirect
    lens: business-analyst
    severity: P0
    issue: "Driver registration page is a blank redirect to login — drivers cannot self-register, creating invisible supply acquisition failure"
    evidence_excerpt: "export default function RegisterPage() { return <Navigate to='/login' replace />; }"
    recommendation: >
      If driver registration is intentionally closed (invite-only or ops-managed),
      communicate this on the page rather than silently redirecting: "Driver applications
      are currently reviewed by our team. [Apply to become a driver →]" with a form or
      email link. A silent redirect causes drivers who reach the register URL (e.g. from
      a job board or a referral link) to land confused on the login page, with no path to
      apply. This is a zero-conversion dead end for driver supply.
    metric_hypothesis: "driver supply acquisition rate; silent redirect loses 100% of organic driver sign-up intent"
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # DELIVERY PORTAL — ONBOARDING WIZARD HEADER
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BA-041
    surface_id: delivery-portal-onboarding-header
    lens: business-analyst
    severity: P1
    issue: "'Become a Delivery Partner' as wizard H1 with subline 'Complete your profile to start delivering' provides no income framing"
    evidence_excerpt: "h1: 'Become a Delivery Partner' / p: 'Complete your profile to start delivering'"
    recommendation: >
      Driver's primary motivation is earnings, not completing a profile. Replace subline with
      earnings anchor: "Complete your profile — start earning from your first delivery."
      or "Most drivers complete this in 10 minutes." Time disclosure + income promise is the
      most effective header combination for driver supply acquisition wizards.
    metric_hypothesis: "driver wizard start-to-complete rate; income framing in wizard header reduces early-step abandonment"
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # DELIVERY PORTAL — STEP PROGRESS
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BA-042
    surface_id: delivery-portal-onboarding-step-progress
    lens: business-analyst
    severity: P2
    issue: "Step 4 is labelled 'Plan' in the StepProgress component — label is ambiguous; driver doesn't know it means a paid subscription plan"
    evidence_excerpt: "{ number: 4, label: 'Plan' }"
    recommendation: >
      Rename to 'Subscription' or 'Billing plan'. 'Plan' could mean a delivery route plan,
      a schedule plan, or a career plan. A driver who discovers step 4 requires paying for
      a subscription without being pre-warned may abandon at that step. The label should
      match the stake: 'Subscription' sets accurate expectations.
    metric_hypothesis: "driver wizard step-4 completion rate; unexpected billing at step 4 causes high-intent abandonment"
    depends_on: null

  - finding_id: BA-043
    surface_id: delivery-portal-onboarding-step-progress
    lens: business-analyst
    severity: P2
    issue: "StepProgress has no estimated time per step or total time indication"
    evidence_excerpt: "5-step progress bar with labels only, no time hints"
    recommendation: >
      Add a total time estimate to the wizard header or first step: "About 10 minutes total
      — have your driving license and Aadhaar ready." This is especially important for the
      delivery portal because Step 3 (Documents) requires physical documents that cannot be
      uploaded without preparation. A driver who starts without their license ends up
      abandoning mid-wizard.
    metric_hypothesis: "driver wizard completion rate; upfront document preparation instruction prevents mid-wizard abandonment"
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # DELIVERY PORTAL — SUBSCRIPTION STEP
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BA-044
    surface_id: delivery-portal-onboarding-subscription
    lens: business-analyst
    severity: P1
    issue: "Subscription step headline 'Choose Your Plan' appears at step 4 of 5 — driver has no prior warning in the wizard that the application requires payment"
    evidence_excerpt: "h2: 'Choose Your Plan' / p: 'A small subscription to keep Fe3dr running — you keep every rupee you earn.'"
    recommendation: >
      Disclose the subscription model in the wizard header or a pre-wizard briefing screen:
      "Heads up: once approved, a small monthly subscription starts after your first X days
      free." A driver who reaches step 4 with 80% invested and then discovers payment is
      required experiences a psychological bait-and-switch, dramatically increasing drop-off.
      The "zero commission" promise does not substitute for subscription disclosure.
    metric_hypothesis: "driver wizard step-4 completion rate; undisclosed payment at step 4 is the highest-risk drop-off in the funnel"
    depends_on: null

  - finding_id: BA-045
    surface_id: delivery-portal-onboarding-subscription
    lens: business-analyst
    severity: P2
    issue: "'No card needed' on trial CTA — trial card-skip promise is good but undermined because the plan selection button still shows payment framing"
    evidence_excerpt: "'Try everything free. No card needed.' / button: 'Start with Monthly Plan'"
    recommendation: >
      The CTA button should reinforce the trial, not jump to the plan interval: change
      'Start with Monthly Plan' to 'Start free trial' for first-time users during the trial
      period. After trial, it should read 'Confirm Monthly Plan'. This prevents cognitive
      dissonance between "no card needed" and a button that sounds like billing.
    metric_hypothesis: "driver subscription step completion rate; trust-coherent CTA copy reduces payment anxiety abandonment"
    depends_on: null

  - finding_id: BA-046
    surface_id: delivery-portal-onboarding-subscription
    lens: business-analyst
    severity: P2
    issue: "Subscription 'how billing works' list item 2 says 'we won't charge you until your earnings cross {thresholdFormatted}' — if threshold=0 from API, this renders incorrectly"
    evidence_excerpt: "'we won't charge you' until your earnings cross {thresholdFormatted} — if threshold=0, renders: 'until your earnings cross ₹0'"
    recommendation: >
      Guard the conditional: only render the threshold copy block when threshold > 0.
      Rendering "cross ₹0" is confusing and undermines the credibility of the payment
      model explanation. The existing code gates the UI card on threshold > 0 but the
      prose list item does not — add the same guard to step 3 of the list.
    metric_hypothesis: "driver subscription step comprehension rate; rendering nonsensical thresholds damages trust in the billing model"
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # DELIVERY PORTAL — REVIEW STEP
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BA-047
    surface_id: delivery-portal-onboarding-review
    lens: business-analyst
    severity: P2
    issue: "Review step Terms checkbox is plain text 'Terms & Conditions' and 'Privacy Policy' with no hyperlinks"
    evidence_excerpt: "'I agree to the Terms & Conditions, Privacy Policy, and confirm that all information provided is accurate'"
    recommendation: >
      'Terms & Conditions' and 'Privacy Policy' must be hyperlinked. This is consistent
      with the vendor portal TOS finding (BA-035). A driver accepting unlinked terms
      without being able to read them creates both a trust problem and a legal exposure.
    metric_hypothesis: "driver review step completion rate; linked terms increase acceptance confidence"
    depends_on: BA-035

  - finding_id: BA-048
    surface_id: delivery-portal-onboarding-review
    lens: business-analyst
    severity: P2
    issue: "Subscription Plan review card shows static copy 'Plan selected. Payments handled securely via Razorpay.' regardless of which plan was selected or current gateway"
    evidence_excerpt: "'Plan selected. Payments handled securely via Razorpay.' — hardcoded gateway name even though gateway is dynamic"
    recommendation: >
      Render the actual plan interval and amount selected in step 4: "Monthly plan — ₹X/month.
      Payments via [gateway]." Confirming the specific price and plan at review prevents
      surprise post-submission. Hardcoding 'Razorpay' when the gateway is configurable also
      creates a correctness bug for non-Razorpay deployments.
    metric_hypothesis: "driver review step confidence and post-approval satisfaction; showing specific billing details prevents billing surprise complaints"
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # DELIVERY PORTAL — STATUS PAGE (WAITING / REJECTION)
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BA-049
    surface_id: delivery-portal-onboarding-status
    lens: business-analyst
    severity: P1
    issue: "Status page 'Application Submitted' state shows no next action — driver is left in a pure waiting loop with no activation task"
    evidence_excerpt: "'Your application has been received and will be reviewed shortly.' — no CTA for submitted/in-review state"
    recommendation: >
      Add an activation task for the waiting period: "While we review your application,
      make sure your phone is ready to receive orders. [Download the driver app →]" or
      "Check your email at [email] for approval updates." An idle waiting screen drives
      7-day drop-off. Give drivers one meaningful action.
    metric_hypothesis: "driver D7 retention during review period; idle waiting screens produce cold applicant drop-off"
    depends_on: null

  - finding_id: BA-050
    surface_id: delivery-portal-onboarding-status
    lens: business-analyst
    severity: P1
    issue: "Rejection state heading 'Application Needs Changes' with description 'Your application requires some changes before it can be approved' — CTA 'Fix & Resubmit' with no specific guidance"
    evidence_excerpt: "'Application Needs Changes' / 'Fix & Resubmit' — no link to specific step or field to fix"
    recommendation: >
      The 'Fix & Resubmit' button takes the driver back to step 1 of the wizard
      (currentStep=1 on rejected status). Without specific guidance on what to fix, the
      driver must re-audit all 5 steps. The rejectionReason field is rendered if populated,
      which is good — but add a secondary CTA: "Not sure what to fix? [Contact support →]"
      Also rename 'Fix & Resubmit' to 'Update and resubmit' to match the platform's
      lower-urgency tone.
    metric_hypothesis: "rejected driver resubmission rate; specific fix guidance increases resubmission attempts"
    depends_on: null

  - finding_id: BA-051
    surface_id: delivery-portal-onboarding-status
    lens: business-analyst
    severity: P2
    issue: "'Auto-refreshing every 30 seconds' message is an internal system detail exposed to the driver — creates clock-watching anxiety"
    evidence_excerpt: "'Auto-refreshing every 30 seconds'"
    recommendation: >
      Remove or replace with: "We'll notify you by email as soon as your application is
      reviewed." The auto-refresh is a technical implementation detail. Telling the driver
      the page is polling every 30 seconds implicitly encourages them to sit and stare at
      the screen waiting for a status change — counterproductive for a multi-day review
      process.
    metric_hypothesis: "driver D7 retention; reducing screen-polling anxiety frees drivers to do other tasks while waiting"
    depends_on: null

  - finding_id: BA-052
    surface_id: delivery-portal-onboarding-status
    lens: business-analyst
    severity: P2
    issue: "Support contact for rejected driver shows only an email address — no expected response SLA"
    evidence_excerpt: "'Contact us at support@fe3dr.com'"
    recommendation: >
      Add response time SLA: "Contact us at support@fe3dr.com — we respond within 1
      business day." A bare email address without SLA gives no confidence that reaching
      out will be effective. For a driver who has been rejected, uncertainty about support
      responsiveness is a reason to not even try.
    metric_hypothesis: "rejected driver re-engagement rate; SLA disclosure increases support contact rate which increases resubmission pipeline"
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # DELIVERY PORTAL — PAYOUT STEP
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BA-053
    surface_id: delivery-portal-onboarding-payout
    lens: business-analyst
    severity: P1
    issue: "Payout Details step has no security framing for bank account / IFSC submission — sensitive financial data collected without encryption reassurance"
    evidence_excerpt: "h2: 'Payout Details' / p: 'How would you like to receive your earnings?' — no security statement"
    recommendation: >
      Add a security line before the form: "Your bank details are encrypted and only used
      to transfer your earnings. We never store your full account number in plain text."
      Bank detail collection without security framing is a leading cause of form abandonment
      in India, where banking fraud awareness is high. The delivery portal has no equivalent
      of the vendor portal's document info banner.
    metric_hypothesis: "driver payout step completion rate; security framing on banking forms increases completion by 15-30%"
    depends_on: null

  - finding_id: BA-054
    surface_id: delivery-portal-onboarding-payout
    lens: business-analyst
    severity: P2
    issue: "Payout form CTA button says 'Continue' — no confirmation of what the driver is proceeding to do with their bank details"
    evidence_excerpt: "'Continue' button after bank account form"
    recommendation: >
      Change to 'Save payout details' — matching the success toast 'Payout details saved'.
      'Continue' is verb-generic; it provides no confirmation that the driver understands
      their bank details are being saved. This also matches style-guide principle of
      verb-first, action-specific button copy.
    metric_hypothesis: "driver payout form completion rate; specific CTA copy reduces hesitation on financial data submission"
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # CROSS-CUTTING VOCABULARY / STYLE VIOLATIONS
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BA-055
    surface_id: vendor-portal-register-heading
    lens: business-analyst
    severity: P2
    issue: "Vendor portal uses 'vendor' terminology in UI — style guide mandates 'chef' for chef-facing surfaces"
    evidence_excerpt: "'Vendor Portal' (header label), 'Register as a vendor', 'Create your vendor account', 'Fe3dr's Vendor Terms'"
    recommendation: >
      Style guide: 'Chef' is the preferred term in chef-facing UI. 'Vendor' is internal/API
      terminology. Rename 'Vendor Portal' to 'Chef Portal' in all chef-facing copy (the
      portal sub-label, the register heading, the TOS reference). Calling chefs 'vendors'
      misaligns with the brand's positioning ("home chefs" not "vendors") and may alienate
      pride-of-craft cooks who don't identify as sellers.
    metric_hypothesis: "vendor signup completion rate; identity-aligned terminology improves conversion for culinary-identity chefs"
    depends_on: null

  - finding_id: BA-056
    surface_id: vendor-portal-onboarding-wizard
    lens: business-analyst
    severity: P3
    issue: "Wizard submit toast uses an escaped apostrophe in inline string: 'We\\'ll review' — suggests raw string concatenation risk"
    evidence_excerpt: "toast.success('Application submitted! We\\'ll review and get back to you within 24-48 hours.')"
    recommendation: >
      Use double quotes for the outer string: "Application submitted! We'll review and get
      back to you within 24-48 hours." — eliminates the escape. This is a code quality
      issue but surfaces in production if a linter doesn't catch it; log as minor.
    metric_hypothesis: "content correctness; rendering artefacts erode trust in platform quality"
    depends_on: null

  - finding_id: BA-057
    surface_id: delivery-portal-onboarding-step-progress
    lens: business-analyst
    severity: P2
    issue: "Payout step is listed in StepProgress labels but StepPayoutDetails is not in the delivery OnboardingPage step routing — step 4 routes to StepSubscriptionPlan, step 5 to StepReview"
    evidence_excerpt: "StepProgress shows steps: Personal Info, Vehicle Details, Documents, Plan, Review — no Payout step. StepPayoutDetails exists but is not wired into the wizard."
    recommendation: >
      The driver payout step (StepPayoutDetails.tsx) exists as a component but is not
      rendered in OnboardingPage.tsx — the wizard jumps from Documents (step 3) to
      Subscription Plan (step 4) to Review (step 5). Payout is either (a) missing from
      the wizard and must be added as step 5 between Plan and Review, or (b) has been
      intentionally moved to post-approval settings. If (b), remove the file or add a
      comment explaining the decision. If (a), this is a broken conversion path — drivers
      submit without providing payout details.
    metric_hypothesis: "driver first-payout rate; missing payout step means zero payout completions at onboarding"
    depends_on: BA-053

  - finding_id: BA-058
    surface_id: web-auth-login-divider
    lens: business-analyst
    severity: P3
    issue: "Social/email divider text is uppercase 'Or' on customer login, lowercase 'or' on vendor login and register — inconsistency across apps"
    evidence_excerpt: "web LoginPage: 'Or' (capitalised) vs vendor-portal LoginPage: 'or' (lowercase)"
    recommendation: >
      Normalise to lowercase 'or' across all three apps (web, vendor-portal, delivery-portal).
      Style guide implies sentence case; 'or' as a conjunction is not a proper noun.
      Consistency signals a unified product, not three separate prototypes.
    metric_hypothesis: "brand polish; inconsistency signals product immaturity to enterprise chefs comparing platforms"
    depends_on: null

  - finding_id: BA-059
    surface_id: vendor-portal-onboarding-step-policies
    lens: business-analyst
    severity: P2
    issue: "Completion state copy 'You're all set to submit your application!' uses an exclamation mark — violates style guide Rule 1 (no urgency exclamation marks except genuine celebration)"
    evidence_excerpt: "'You're all set to submit your application!'"
    recommendation: >
      Change to 'You're all set. Review your application and submit.' — period, no
      exclamation. The all-policies-accepted state is a functional milestone, not a
      celebration event. Style guide: exclamation marks only in genuine celebration
      (maximum 1 per page). This is a functional confirmation, not a celebration.
    metric_hypothesis: "voice consistency; exclamation marks outside celebration contexts undermine the brand's 'quietly confident' positioning"
    depends_on: null

  - finding_id: BA-060
    surface_id: delivery-portal-onboarding-review
    lens: business-analyst
    severity: P3
    issue: "Delivery portal review step submit success toast says 'Application submitted successfully!' — exclamation mark violates style guide"
    evidence_excerpt: "toast.success('Application submitted successfully!')"
    recommendation: >
      Change to 'Application submitted.' — plain past tense, no exclamation. The driver
      is then immediately navigated to OnboardingStatusPage where the status is clearly
      shown. The toast is transient and exclamation is redundant.
    metric_hypothesis: "voice consistency"
    depends_on: null
```

## Brand Voice findings

```yaml
# Brand Voice findings — auth-onboarding category
# Lens: brand-voice (cross-app, cross-surface voice consistency)
# Inputs: STYLE-GUIDE.md, .impeccable.md, CONTENT-INVENTORY.md auth-onboarding slice

# ─── P0: AI-slop / unverified social proof / brand-trust breakers ─────────────

findings:
  - finding_id: BV-001
    surface_id: web-auth-register-heading
    lens: brand-voice
    severity: P0
    issue: "Marketing-toned 'Access to 500+ home chefs' bullet on a customer register page — hardcoded fake metric on first auth surface."
    evidence_excerpt: "Access to 500+ home chefs"
    related_surfaces: ["web-home-hero", "web-home-stats"]
    recommendation: "Remove hardcoded '500+' bullets from RegisterPage. Either bind to live count from the chef-discovery API or delete the bullet entirely. Style-guide Rule 1 + 5: no unverified social-proof on register surfaces."
    depends_on: null

  - finding_id: BV-002
    surface_id: vp-auth-register-hero-sub
    lens: brand-voice
    severity: P0
    issue: "'Join thousands of home chefs earning with Fe3dr' — unverified 'thousands' claim on the chef-facing register hero, the very first impression for new vendors."
    evidence_excerpt: "Join thousands of home chefs earning with Fe3dr. Your kitchen, your recipes, your rules."
    related_surfaces: ["web-home-hero-cta", "vp-auth-register-heading"]
    recommendation: "Replace with a calm, factual line: 'Sell home-cooked food from your kitchen. Your menu, your hours, your earnings.' Drop 'thousands' until live count is wired."
    depends_on: null

  - finding_id: BV-003
    surface_id: vp-auth-register-hero-sub
    lens: brand-voice
    severity: P0
    issue: "Marketing bullets on chef register page — 'Zero commission first month' and 'Weekly payouts' belong on a marketing landing, not on a functional auth surface."
    evidence_excerpt: "Zero commission first month / Get started completely risk-free / Weekly payouts / Get paid directly to your bank account"
    related_surfaces: ["vp-auth-login-features-list", "dp-auth-step4-zero-commission"]
    recommendation: "Strip benefit bullets from RegisterPage (wrong persona-tone — chef wants to register, not be sold to). Move the same content to a separate /sell-with-us marketing page if needed. Persona-tone for chef = functional/operational, never hype."
    depends_on: null

  - finding_id: BV-004
    surface_id: dp-auth-step4-zero-commission
    lens: brand-voice
    severity: P0
    issue: "Two-paragraph 'We don't take any commission' block inside driver onboarding step 4 — marketing prose in the middle of a functional flow."
    evidence_excerpt: "We don't take any commission from your earnings. Every delivery fee and every tip goes directly to your account — 100%. We only charge a small subscription fee to keep the platform running and help you get more orders. We're here to help you succeed, not to make profit from your hard work."
    related_surfaces: ["vp-auth-register-hero-sub", "dp-auth-step4-h"]
    recommendation: "Cut to one sentence in driver tone: 'You keep 100% of delivery fees and tips. We charge only the plan fee below.' Glanceable driver copy ≤4 words where possible, ≤25 words for explanatory blocks. Current 38-word block violates persona-tone rule."
    depends_on: null

  - finding_id: BV-005
    surface_id: dp-auth-step4-billing-howto
    lens: brand-voice
    severity: P1
    issue: "Driver-facing 'How billing works' is a 50-word dense block with 4-step list inside onboarding — too verbose for driver persona (target ≤4 words / telegraphic per matrix)."
    evidence_excerpt: "How billing works: [4-step ordered list explaining trial, threshold, deduction, cancel-anytime]"
    related_surfaces: ["dp-auth-step4-trial", "dp-auth-step4-threshold"]
    recommendation: "Collapse into 3 short rows: 'Free for {n} days. Pay after you reach {amount}. Cancel anytime.' Move full detail to a 'Plan details' modal."
    depends_on: null

  # ─── P0: Cross-app verb drift on the core auth action ────────────────────────

  - finding_id: BV-006
    surface_id: web-auth-login-email-toggle
    lens: brand-voice
    severity: P0
    issue: "'Sign in with email' vs 'Sign in with Email' vs 'Sign in with Email' — capitalization of the same OAuth toggle drifts across 4 surfaces. Style-guide is explicit: sentence case for buttons."
    evidence_excerpt: "web: 'Sign in with email' / vendor-portal: 'Sign in with Email' / delivery-portal: 'Sign in with Email' / admin-portal: 'Sign in with Email'"
    related_surfaces: ["vp-auth-login-email-btn", "dp-auth-email-button", "ap-auth-login-heading"]
    recommendation: "Canonicalize as 'Sign in with email' (sentence case) per STYLE-GUIDE §4 Buttons. Apply to web, vendor-portal, delivery-portal, admin-portal."
    depends_on: null

  - finding_id: BV-007
    surface_id: web-auth-register-email-toggle
    lens: brand-voice
    severity: P0
    issue: "Register-with-email toggle wording drifts: 'Sign up with email' (web) vs 'Register with Email' (vendor-portal). Same action, two verbs and two casings."
    evidence_excerpt: "web: 'Sign up with email' / vendor-portal: 'Register with Email'"
    related_surfaces: ["vp-auth-register-email-btn"]
    recommendation: "Use 'Sign up with email' everywhere — style-guide bans 'Register' in favor of 'Sign up'. Apply lowercase 'email'."
    depends_on: null

  - finding_id: BV-008
    surface_id: dp-auth-login-subtitle-driver
    lens: brand-voice
    severity: P0
    issue: "'Driver login or sign up' and 'Login or sign up to deliver with Fe3dr' mix banned 'Login' with approved 'sign up' on the same surface."
    evidence_excerpt: "Driver login or sign up / Login or sign up to deliver with Fe3dr"
    related_surfaces: ["dp-auth-mode-driver-sub"]
    recommendation: "Use 'Sign in' (two words) — STYLE-GUIDE §3 explicitly bans 'Login' and 'Log in'. Rewrite as 'Sign in or sign up to deliver'."
    depends_on: null

  - finding_id: BV-009
    surface_id: ap-auth-login-heading
    lens: brand-voice
    severity: P1
    issue: "'Admin Sign In' uses Title Case for a heading + the canonical verb is one word in heading vs two in button. Inconsistent with sentence-case norm."
    evidence_excerpt: "Admin Sign In"
    related_surfaces: ["vp-auth-login-heading", "web-auth-login-heading"]
    recommendation: "Use sentence case: 'Admin sign in' or simply 'Sign in' (the page context already says Admin Portal). Pick one form for the verb across all four web auth surfaces."
    depends_on: null

  # ─── P1: Welcome-back consistency (good) but contextually thin ───────────────

  - finding_id: BV-010
    surface_id: web-auth-login-heading
    lens: brand-voice
    severity: P3
    issue: "'Welcome back' is consistent across web, vendor-portal, mobile-customer, mobile-vendor, mobile-delivery — flag as the standard. But delivery-portal and admin-portal LoginPage skip the welcome-back greeting entirely (drift by omission)."
    evidence_excerpt: "web/vp/mc/mv/md: 'Welcome back' / delivery-portal: 'Sign in to get started' / admin-portal: 'Admin Sign In'"
    related_surfaces: ["vp-auth-login-heading", "dp-auth-login-subtitle-choose", "ap-auth-login-heading", "mc-auth-login-title", "mv-auth-login-title", "md-auth-001"]
    recommendation: "Keep 'Welcome back' on customer + chef surfaces (warmer tone). Admin/driver default to 'Sign in' (Q+M persona tone matrix). Document the split in style guide."
    depends_on: null

  # ─── P0: Brand-name confusion — most damaging at first impression ────────────

  - finding_id: BV-011
    surface_id: dp-auth-login-title
    lens: brand-voice
    severity: P0
    issue: "Brand wordmark on delivery login is 'Fe3dr Delivery' — sub-brand applied to login page header creates inconsistency vs other portals that show just 'Fe3dr'."
    evidence_excerpt: "Fe3dr Delivery"
    related_surfaces: ["web-auth-login-heading", "vp-auth-login-heading", "ap-auth-login-portal-label"]
    recommendation: "Unify wordmark to 'Fe3dr' across all portals; use a small subordinate label ('Delivery portal' / 'Admin portal') below if needed. Sub-brand stacking is anti-pattern per .impeccable.md photo-forward / chrome-light principle."
    depends_on: null

  - finding_id: BV-012
    surface_id: vp-auth-login-features-list
    lens: brand-voice
    severity: P1
    issue: "'This portal is only for vendor accounts. Please use the Fe3dr customer app.' — refers to customer app, but the customer-facing brand surface is also Fe3dr. Distinction is unclear and may confuse users about which 'Fe3dr' is which."
    evidence_excerpt: "This portal is only for vendor accounts. Please use the Fe3dr customer app."
    related_surfaces: ["dp-auth-mode-driver-sub", "ap-auth-login-brand-sub"]
    recommendation: "Either name the surfaces ('Fe3dr Vendor', 'Fe3dr Driver', 'Fe3dr Customer') or refer by purpose ('To order food, sign in at fe3dr.com'). Avoid ambiguous self-reference."
    depends_on: null

  # ─── P0: Chef onboarding step labels drift between mobile-vendor and vendor-portal ──

  - finding_id: BV-013
    surface_id: vp-onb-personal-title
    lens: brand-voice
    severity: P0
    issue: "Same chef onboarding step labelled differently across mobile-vendor and vendor-portal: 'Personal Details' (vp StepPersonalInfo h3) vs 'Personal Information' (mv personal-info.tsx) vs 'Personal Info' (vp stepper)."
    evidence_excerpt: "vp StepPersonalInfo h3: 'Personal Details' / mv personal-info.tsx: 'Personal Information' / vp stepper: 'Personal Info'"
    related_surfaces: ["vp-onb-stepper-labels", "mv-onb-personal-heading", "mv-onb-review-section-personal"]
    recommendation: "Pick 'Personal information' (sentence case) as canonical — it matches the review section. Stepper label can shorten to 'Personal info' but the page heading should match across web + mobile."
    depends_on: null

  - finding_id: BV-014
    surface_id: vp-onb-kitchen-title
    lens: brand-voice
    severity: P1
    issue: "Same chef onboarding step labelled 'Kitchen Information' (vp StepKitchenDetails) vs 'Kitchen Details' (mv kitchen-details.tsx) vs 'Kitchen Details' (vp stepper). Inconsistent across web + mobile chef surfaces."
    evidence_excerpt: "vp StepKitchenDetails: 'Kitchen Information' / mv: 'Kitchen Details' / vp stepper: 'Kitchen Details'"
    related_surfaces: ["vp-onb-stepper-labels", "mv-onb-kitchen-heading"]
    recommendation: "Canonicalize as 'Kitchen details' (sentence case) on both web and mobile, both the heading and the stepper."
    depends_on: null

  - finding_id: BV-015
    surface_id: vp-onb-ops-pricing-title
    lens: brand-voice
    severity: P1
    issue: "Web chef step is 'Delivery & Pricing' / 'Operations' (stepper) but mobile-vendor uses 'Operations' as the screen heading. Step count differs too — vendor-portal has fewer steps than mobile-vendor (6)."
    evidence_excerpt: "vp StepOperations: 'Delivery & Pricing' / mv operations.tsx: 'Operations' / vp stepper: includes 'Operations'"
    related_surfaces: ["vp-onb-stepper-labels", "mv-onb-ops-heading"]
    recommendation: "Use 'Delivery & pricing' as the screen heading on mobile-vendor too (it's the human-readable summary). Keep stepper short label as 'Operations'. Reconcile step counts in a follow-up info-arch task."
    depends_on: null

  - finding_id: BV-016
    surface_id: vp-onb-header-title
    lens: brand-voice
    severity: P1
    issue: "Web chef onboarding header is 'Set Up Your Kitchen' (Title Case) while no mobile-vendor equivalent header exists; step counter 'Step 4 of 6' titles in mobile are stack-header strings only."
    evidence_excerpt: "vendor-portal: 'Set Up Your Kitchen' (Title Case) / mobile-vendor: just 'Step N of 6' in header"
    related_surfaces: ["mv-onb-step1-title", "vp-onb-stepper-labels"]
    recommendation: "Use sentence case 'Set up your kitchen' on web. Add same heading above the step counter on mobile-vendor for consistency."
    depends_on: null

  - finding_id: BV-017
    surface_id: vp-onb-stepper-labels
    lens: brand-voice
    severity: P1
    issue: "Web chef stepper uses Title Case ('Personal Info', 'Kitchen Details', 'Operations', 'Documents', 'Policies & Review') — banned outside proper nouns per STYLE-GUIDE §6."
    evidence_excerpt: "Personal Info / Kitchen Details / Operations / Documents / Policies & Review"
    related_surfaces: ["dp-auth-onboarding-step-labels"]
    recommendation: "Sentence case throughout: 'Personal info / Kitchen details / Operations / Documents / Policies & review'."
    depends_on: null

  # ─── P0: Driver onboarding step labels drift between mobile-delivery & delivery-portal ──

  - finding_id: BV-018
    surface_id: dp-auth-onboarding-step-labels
    lens: brand-voice
    severity: P0
    issue: "Driver step labels Title Case on delivery-portal ('Personal Info', 'Vehicle Details', 'Documents', 'Plan', 'Review') vs sentence case context expected; same step headings on mobile-delivery use yet another form ('Personal Information' as full heading, 'Personal Info' in review)."
    evidence_excerpt: "delivery-portal StepProgress: 'Personal Info / Vehicle Details / Documents / Plan / Review' / mobile-delivery personal.tsx heading: 'Personal Information' / mobile-delivery review.tsx section: 'Personal Info'"
    related_surfaces: ["md-onb-003", "md-onb-095", "dp-auth-step1-h", "dp-auth-step5-sections"]
    recommendation: "Canonicalize driver step labels: 'Personal information / Vehicle details / Documents / Plan / Review' (sentence case). Match between delivery-portal and mobile-delivery 1:1, including review section names."
    depends_on: null

  - finding_id: BV-019
    surface_id: dp-auth-step3-h
    lens: brand-voice
    severity: P1
    issue: "Step heading 'Documents & Photos' (delivery-portal) vs 'Upload Documents' (mobile-delivery) for the same step. Customer + chef + driver: each app names doc-upload step differently."
    evidence_excerpt: "delivery-portal StepDocuments: 'Documents & Photos' / mobile-delivery documents.tsx: 'Upload Documents' / mobile-vendor documents.tsx: 'Documents'"
    related_surfaces: ["md-onb-042", "mv-onb-docs-heading"]
    recommendation: "Use 'Documents' as the heading everywhere; subline can explain what's needed (e.g. 'Photos and PDFs of your ID and vehicle papers'). Removes Title Case, removes verbosity, unifies the step name across all three apps."
    depends_on: null

  - finding_id: BV-020
    surface_id: dp-auth-onboarding-header
    lens: brand-voice
    severity: P1
    issue: "Driver onboarding entry header is 'Become a Delivery Partner' on web (delivery-portal) but 'Driver Onboarding' on mobile-delivery stack header — same flow, two different framings."
    evidence_excerpt: "delivery-portal OnboardingPage: 'Become a Delivery Partner' / mobile-delivery layout: 'Driver Onboarding'"
    related_surfaces: ["md-onb-002", "dp-auth-status-h"]
    recommendation: "Use 'Become a driver' (Customer-facing brand uses 'Delivery partner', but driver-facing always uses 'Driver' per STYLE-GUIDE §3). Apply same heading on both. Status page 'Delivery Partner Onboarding' should also become 'Driver onboarding'."
    depends_on: null

  - finding_id: BV-021
    surface_id: dp-auth-status-h
    lens: brand-voice
    severity: P0
    issue: "Status page sub-heading is 'Delivery Partner Onboarding' on delivery-portal — driver should never see 'Delivery Partner' (that's the customer-facing variant per STYLE-GUIDE §3)."
    evidence_excerpt: "Delivery Partner Onboarding"
    related_surfaces: ["dp-auth-mode-driver-sub"]
    recommendation: "Rewrite as 'Driver onboarding'. STYLE-GUIDE: customer sees 'Delivery partner', driver always sees 'Driver'."
    depends_on: null

  - finding_id: BV-022
    surface_id: dp-auth-mode-driver-sub
    lens: brand-voice
    severity: P0
    issue: "'I'm a Driver' / 'I'm Staff' role-chooser uses Title Case + capitalised role — should be sentence case ('I'm a driver', 'I'm staff')."
    evidence_excerpt: "I'm a Driver / I'm Staff"
    related_surfaces: ["dp-auth-mode-staff-cta"]
    recommendation: "Sentence case: 'I'm a driver' / 'I'm staff'."
    depends_on: null

  # ─── P1: Persona-tone violations on driver/vendor surfaces ───────────────────

  - finding_id: BV-023
    surface_id: vp-auth-login-hero-heading
    lens: brand-voice
    severity: P1
    issue: "Chef login hero 'Grow your home kitchen business' is marketing-tone on a returning-user surface. Returning vendor already has a kitchen — they want to sign in, not be pitched."
    evidence_excerpt: "Grow your home kitchen business / Manage menus, track orders, view earnings - all from one dashboard."
    related_surfaces: ["vp-auth-login-features-list", "vp-auth-register-hero-heading"]
    recommendation: "Remove the marketing hero from LOGIN page (returning user). Keep brief subline only. Move 'Grow your home kitchen business' to RegisterPage hero where it's audience-appropriate."
    depends_on: null

  - finding_id: BV-024
    surface_id: vp-auth-login-features-list
    lens: brand-voice
    severity: P1
    issue: "5-bullet feature list ('Real-time order management', 'Earnings & analytics dashboard'…) on chef login page — sales-style copy where chef just wants to sign in. Persona-tone violation (chef should see functional/operational language)."
    evidence_excerpt: "Real-time order management / Earnings & analytics dashboard / Menu management with categories / Customer reviews & ratings / Document verification & compliance"
    related_surfaces: ["ap-auth-login-feature-list"]
    recommendation: "Drop the feature list from LoginPage. Either keep a single calm photo + subline, or move bullets to RegisterPage where they're audience-appropriate."
    depends_on: null

  - finding_id: BV-025
    surface_id: ap-auth-login-feature-list
    lens: brand-voice
    severity: P1
    issue: "Admin portal login left panel has 6 sales-style feature bullets ('User & role management; Chef verification & approvals…') — wrong persona-tone. Admin = 'precise, no fluff' per STYLE-GUIDE."
    evidence_excerpt: "User & role management; Chef verification & approvals; Order monitoring & refunds; Platform analytics & insights; Revenue tracking & payouts; Content moderation tools"
    related_surfaces: ["ap-auth-login-brand-sub", "ap-auth-login-instruction"]
    recommendation: "Strip the bullet list on admin login. Admin already knows what they're signing into. Replace left panel with a single line + quiet brand mark; or drop the split-pane entirely."
    depends_on: null

  - finding_id: BV-026
    surface_id: ap-auth-login-instruction
    lens: brand-voice
    severity: P2
    issue: "'Access the administration dashboard with your internal credentials' — verbose, slightly bureaucratic. Admin persona = neutral operator, plain English."
    evidence_excerpt: "Access the administration dashboard with your internal credentials"
    related_surfaces: ["ap-auth-login-heading"]
    recommendation: "Trim to 'Sign in with your work account.' or simply remove (the heading 'Admin sign in' is enough)."
    depends_on: null

  - finding_id: BV-027
    surface_id: ap-auth-login-brand-sub
    lens: brand-voice
    severity: P3
    issue: "'Manage users, chefs, orders, and analytics for the Fe3dr platform.' — slight redundancy with feature list above; Oxford comma is fine but tone is brochure-ish."
    evidence_excerpt: "Manage users, chefs, orders, and analytics for the Fe3dr platform."
    related_surfaces: ["ap-auth-login-feature-list"]
    recommendation: "Trim or remove once feature list is dropped per BV-025."
    depends_on: null

  # ─── P0: Mobile-vendor Title Case headings violate sentence-case rule ───────

  - finding_id: BV-028
    surface_id: mv-onb-personal-heading
    lens: brand-voice
    severity: P1
    issue: "Mobile-vendor onboarding screens use Title Case headings ('Personal Information', 'Kitchen Details', 'Personal Information', 'Documents'). STYLE-GUIDE §6 mandates sentence case for labels."
    evidence_excerpt: "Personal Information / Kitchen Details / Documents / Operations / Policies / Review Application"
    related_surfaces: ["mv-onb-kitchen-heading", "mv-onb-docs-heading", "mv-onb-ops-heading", "mv-onb-policies-heading", "mv-onb-review-heading"]
    recommendation: "Sentence case throughout: 'Personal information / Kitchen details / Documents / Operations / Policies / Review application'."
    depends_on: null

  - finding_id: BV-029
    surface_id: mv-onb-personal-fullname-label
    lens: brand-voice
    severity: P2
    issue: "Mobile-vendor field labels Title Case ('Full Name *', 'Phone Number *', 'Email Address', 'Business Name *', 'Cuisine Types *', 'Description *'). STYLE-GUIDE §4 form labels: sentence case, no colons."
    evidence_excerpt: "Full Name * / Phone Number * / Email Address / Business Name * / Cuisine Types * / Description *"
    related_surfaces: ["mv-onb-personal-phone-label", "mv-onb-kitchen-business-label", "mv-onb-kitchen-cuisines-label"]
    recommendation: "Sentence case: 'Full name', 'Phone number', 'Email address', 'Business name', 'Cuisine types', 'Description'. Move required asterisk to field-level visual rather than appending to label string."
    depends_on: null

  - finding_id: BV-030
    surface_id: mc-onb-step1-labels
    lens: brand-voice
    severity: P2
    issue: "Mobile-customer onboarding field labels Title Case ('First Name', 'Last Name', 'Phone Number'). Web equivalent (web-auth-register-form-fields) uses sentence case ('First name', 'Last name')."
    evidence_excerpt: "mobile-customer: First Name / Last Name / Phone Number / web: First name / Last name / Phone number"
    related_surfaces: ["web-auth-register-form-fields", "web-auth-onboarding-step-basic"]
    recommendation: "Mobile-customer should match web: sentence case. Casing drift across customer surfaces undermines unified brand voice."
    depends_on: null

  # ─── P1: Field-label drift across personas (Phone vs Email) ─────────────────

  - finding_id: BV-031
    surface_id: vp-onb-personal-fields
    lens: brand-voice
    severity: P2
    issue: "Email/Phone field labels drift: 'Email Address' (vp + mv), 'Email' (web auth), 'Phone Number' (vp + mc + mv), 'Phone number' (web onboarding). Casing AND verbosity drift."
    evidence_excerpt: "vp: Email Address / Phone Number / web register: Email / Password / web onboarding: Phone number / mc: Phone Number / mv: Email Address"
    related_surfaces: ["mc-onb-step1-labels", "mv-onb-personal-email-label", "web-auth-onboarding-step-basic"]
    recommendation: "Canonical labels: 'Email' (single word, sentence case) and 'Phone number' (when distinct field). Use 'Email address' only in legal/contact contexts. Apply across all 7 apps."
    depends_on: null

  # ─── P1: Onboarding subtitle drift  ─────────────────────────────────────────

  - finding_id: BV-032
    surface_id: mc-onb-step1-subtitle
    lens: brand-voice
    severity: P2
    issue: "Customer onboarding step 1 subtitle 'We need a few details to set up your account.' is functional/bland; mobile-vendor uses warmer 'Tell us about yourself'. Cross-app drift in the same persona function."
    evidence_excerpt: "mc step 1 sub: 'We need a few details to set up your account.' / mv personal-info sub: 'Tell us about yourself' / md personal sub: 'Tell us about yourself to get started' / dp StepPersonalInfo sub: 'Tell us about yourself to get started'"
    related_surfaces: ["mv-onb-personal-sub", "md-onb-004", "dp-auth-step1-sub"]
    recommendation: "Use 'Tell us about yourself.' on customer step 1 too — matches the warm/sensory customer persona-tone and is consistent with chef/driver flows. Drop the bland 'We need a few details' line."
    depends_on: null

  - finding_id: BV-033
    surface_id: mc-onb-step2-subtitle
    lens: brand-voice
    severity: P2
    issue: "'Where should we deliver your orders?' phrases first-person plural correctly but follows pattern 'Where should we…' that's slightly transactional. Inconsistent with the warmer 'Your delivery address' title above."
    evidence_excerpt: "Where should we deliver your orders?"
    related_surfaces: ["web-auth-onboarding-step-address"]
    recommendation: "Trim to 'We'll send your driver here.' (matches STYLE-GUIDE §4 helper-text example verbatim — pattern source)."
    depends_on: null

  - finding_id: BV-034
    surface_id: web-auth-onboarding-heading
    lens: brand-voice
    severity: P2
    issue: "Web onboarding header 'Complete Your Profile' (Title Case) drifts from chef header 'Set Up Your Kitchen' (also Title Case but different verb pattern) and driver 'Become a Delivery Partner'."
    evidence_excerpt: "web onboarding: 'Complete Your Profile' / vendor-portal: 'Set Up Your Kitchen' / delivery-portal: 'Become a Delivery Partner'"
    related_surfaces: ["vp-onb-header-title", "dp-auth-onboarding-header"]
    recommendation: "Sentence case all three: 'Complete your profile' / 'Set up your kitchen' / 'Become a driver'. Same verb pattern would also help — consider 'Complete your profile / Complete your kitchen profile / Complete your driver profile' as a parallel set."
    depends_on: null

  # ─── P1: Step counter drift ────────────────────────────────────────────────

  - finding_id: BV-035
    surface_id: web-auth-onboarding-heading
    lens: brand-voice
    severity: P3
    issue: "Step counter format drift: web onboarding 'Step {n} of 3' inline with mdash separator vs mobile-customer 'Step 2 of 3' as static label vs mobile-vendor 'Step 1 of 6' as stack title vs delivery-portal 'Step N of 5' implicit via StepProgress badge."
    evidence_excerpt: "web: 'Step 1 of 3 — {title}' / mc: 'Step 2 of 3' / mv: 'Step 1 of 6' / md: 'Step 1 of 6'"
    related_surfaces: ["mc-onb-step1-progress", "mv-onb-step1-title", "md-onb-001"]
    recommendation: "Standardize as 'Step {n} of {total}' (no mdash, no static title appended). Position consistently (above heading, muted text). Different totals (3 / 5 / 6) are correct per audience, but format must match."
    depends_on: null

  # ─── P1: Verification/waiting messages drift ────────────────────────────────

  - finding_id: BV-036
    surface_id: dp-auth-status-descriptions
    lens: brand-voice
    severity: P1
    issue: "Application-waiting copy drifts across driver and chef surfaces. Driver: 'Our team is currently reviewing your application. This usually takes 1-2 business days.' Chef mobile pending: 'Your application has been submitted and is being reviewed by our team.' Driver mobile pending: 'Estimated review time: 24–48 hours.'"
    evidence_excerpt: "dp web status: 'Our team is currently reviewing your application. This usually takes 1-2 business days.' / md pending: 'Your application has been submitted and is being reviewed by our team.' / 'Estimated review time: 24–48 hours' / mv pending: 'Application Submitted!'"
    related_surfaces: ["md-onb-114", "md-onb-115", "mv-onb-pending-title"]
    recommendation: "Standardize a single sentence: 'Your application is under review. We'll let you know within 24–48 hours.' Apply identically to driver web, driver mobile, and chef mobile pending screens. Drop the exclamation in 'Application Submitted!'."
    depends_on: null

  - finding_id: BV-037
    surface_id: dp-auth-status-titles
    lens: brand-voice
    severity: P0
    issue: "Same status labelled differently across surfaces: 'Under Review' (dp web title) / 'Application Under Review' (md mobile) / 'Submitted' / 'In Review' (dp web badge) / 'Application Needs Changes' (dp web) / 'Application Not Approved' (md, mv mobile)."
    evidence_excerpt: "dp web titles: 'Application Needs Changes / Under Review / Application Submitted' / dp web badges: 'Rejected / In Review / Submitted' / md pending: 'Application Under Review / Application Not Approved' / mv pending: 'Application Not Approved / Application Submitted!'"
    related_surfaces: ["dp-auth-status-badges", "md-onb-109", "md-onb-113", "mv-onb-pending-title"]
    recommendation: "Canonical three statuses: 'Under review' / 'Approved' / 'Needs changes'. Drop 'Application' prefix (context already implied), drop 'Not Approved' euphemism (use 'Needs changes' which is also more action-oriented). Standardize across web + mobile for driver and chef."
    depends_on: null

  - finding_id: BV-038
    surface_id: md-onb-109
    lens: brand-voice
    severity: P1
    issue: "'Application Not Approved' followed by 'Unfortunately your application was not approved at this time.' — apologetic + indirect. Plain English / restraint principle says lead with the action."
    evidence_excerpt: "Application Not Approved / Unfortunately your application was not approved at this time."
    related_surfaces: ["dp-auth-status-rejection-reason", "mv-onb-pending-title"]
    recommendation: "'Needs changes. Here's what to fix.' followed by the rejection reason. Reduces emotional load + leads with action. Apply across driver and chef pending screens."
    depends_on: null

  # ─── P1: OAuth/biometric button casing drift  ───────────────────────────────

  - finding_id: BV-039
    surface_id: dp-auth-google-button
    lens: brand-voice
    severity: P3
    issue: "Mobile shared LoginScreen 'Continue with Google' + 'Continue with Apple' is canonical, but admin-portal + vendor-portal + delivery-portal also use 'Continue with Google' — consistent on Google, ✓. Apple button only appears in mobile-shared."
    evidence_excerpt: "All web + mobile use 'Continue with Google'; Apple only on mobile-shared LoginScreen 'Continue with Apple'"
    related_surfaces: ["vp-auth-login-google-btn", "web-auth-login-social"]
    recommendation: "Document 'Continue with {provider}' as the canonical OAuth pattern in style guide. Apple button text already correct; this finding is to capture-and-enforce going forward."
    depends_on: null

  - finding_id: BV-040
    surface_id: web-auth-login-submit
    lens: brand-voice
    severity: P3
    issue: "Loading state strings drift: 'Signing in...' (web) vs 'Submitting...' (driver review) vs 'Creating account...' (web register) vs 'Saving...' (driver subscription) vs 'Uploading...' (driver docs). Some use ellipsis, some end without punctuation."
    evidence_excerpt: "web: 'Signing in...' / 'Creating account...' / dp: 'Submitting...' / 'Saving...' / md: 'Uploading...'"
    related_surfaces: ["web-auth-register-submit", "dp-auth-step5-submit", "dp-auth-step4-submit"]
    recommendation: "Loading state pattern: '<Verb>ing…' with proper ellipsis character (U+2026). Standardize across all auth/onboarding loading states."
    depends_on: null

  # ─── P1: Mobile error messages — developer-shaped, leak provider names ──────

  - finding_id: BV-041
    surface_id: md-auth-002
    lens: brand-voice
    severity: P1
    issue: "Developer-shaped error 'Google sign-in failed: no ID token' / 'Apple sign-in failed: no identity token' surfaces to user if uncaught. Style-guide §4: 'What happened → What to do.' No technical terms."
    evidence_excerpt: "Google sign-in failed: no ID token / Apple sign-in failed: no identity token / Biometric authentication failed"
    related_surfaces: ["md-auth-003", "md-auth-004", "md-auth-005"]
    recommendation: "User-facing strings: 'Couldn't sign in with Google. Try again or use email.' / 'Couldn't sign in with Apple. Try again or use email.' / 'Couldn't verify fingerprint. Try again or sign in with email.' Log technical detail separately."
    depends_on: null

  - finding_id: BV-042
    surface_id: md-onb-081
    lens: brand-voice
    severity: P2
    issue: "Generic 'Failed to save payout details. Please try again.' pattern repeated across md-onb-020, md-onb-041, md-onb-081, md-onb-092, md-onb-108 — identical phrasing is OK, but 'Please try again.' is bland."
    evidence_excerpt: "Failed to save personal info. Please try again. / Failed to save vehicle details. Please try again. / Failed to save payout details. Please try again. / Failed to submit application. Please try again. / Failed to select plan. Please try again."
    related_surfaces: ["md-onb-020", "md-onb-041", "md-onb-092", "md-onb-108"]
    recommendation: "Style-guide pattern 'What happened → What to do.' Replace with action-led: 'Couldn't save. Check your connection and try again.' Single utility function for these generic API failures."
    depends_on: null

  # ─── P0: Banned-vocab + AI-slop email-confirmation drift ─────────────────────

  - finding_id: BV-043
    surface_id: vp-auth-register-subheading
    lens: brand-voice
    severity: P2
    issue: "'Create your vendor account to start selling home-cooked meals' uses 'vendor account' (chef-facing should be 'chef account' per STYLE-GUIDE §3 'Vendor ❌')."
    evidence_excerpt: "Create your vendor account to start selling home-cooked meals"
    related_surfaces: ["vp-auth-login-register-link"]
    recommendation: "Use 'Create your chef account…' — 'Vendor' is banned externally; the chef self-identifies as a chef. Internal database can still use 'vendor', but UI strings use 'chef'."
    depends_on: null

  - finding_id: BV-044
    surface_id: vp-auth-login-register-link
    lens: brand-voice
    severity: P2
    issue: "'Want to start selling? Register as a vendor' — 'Register' is banned ('Sign up' ✓), 'vendor' is banned for chef-facing (should be 'chef')."
    evidence_excerpt: "Want to start selling? Register as a vendor"
    related_surfaces: ["vp-auth-register-subheading", "vp-onb-personal-title"]
    recommendation: "'New to Fe3dr? Sign up as a chef.'"
    depends_on: null

  # ─── P1: Mixed legal-tone drift in onboarding consent ───────────────────────

  - finding_id: BV-045
    surface_id: dp-auth-step5-terms
    lens: brand-voice
    severity: P1
    issue: "Driver consent string 'I agree to the Terms & Conditions, Privacy Policy, and confirm that all information provided is accurate and up to date.' uses '& Conditions' while customer/chef use 'Terms of Service'."
    evidence_excerpt: "dp web: 'Terms & Conditions, Privacy Policy' / md mobile: 'Terms of Service and Privacy Policy' / vp register: 'Vendor Terms and Privacy Policy' / vp login: 'Terms of Service and Privacy Policy'"
    related_surfaces: ["md-onb-103", "vp-auth-register-login-link"]
    recommendation: "Pick one: 'Terms of Service' (already most common). Drop '& Conditions' phrasing — sounds legalistic and varies."
    depends_on: null

  # ─── P2: Helper/placeholder voice  ──────────────────────────────────────────

  - finding_id: BV-046
    surface_id: vp-onb-personal-fields
    lens: brand-voice
    severity: P3
    issue: "'Pre-filled from your login' hint — mixes verb tense and is unclear. Better: 'From your login'."
    evidence_excerpt: "Pre-filled from your login"
    related_surfaces: []
    recommendation: "'From your login' — passive 'pre-filled' adds nothing."
    depends_on: null

  - finding_id: BV-047
    surface_id: web-auth-login-email-fields
    lens: brand-voice
    severity: P3
    issue: "Password placeholder 'Enter your password' is redundant with 'Password' label. STYLE-GUIDE §4 form-labels: placeholders show format/example, not restate label."
    evidence_excerpt: "label='Password' placeholder='Enter your password'"
    related_surfaces: ["mc-onb-step1-placeholders", "mv-onb-personal-fullname-ph"]
    recommendation: "Remove redundant placeholder (or use '••••••••' like mobile-shared LoginScreen). 'Enter your first name' / 'Enter your last name' / 'Enter your city' all show the same redundancy pattern across apps — replace with examples ('Meena', 'Bangalore') where possible."
    depends_on: null

  # ─── P2: Date / locale drift ────────────────────────────────────────────────

  - finding_id: BV-048
    surface_id: md-onb-018
    lens: brand-voice
    severity: P2
    issue: "Date placeholder 'MM/DD/YYYY' on driver mobile DOB field — inconsistent with India-locale validation elsewhere (10-digit Indian mobile, KA01AB1234 plate). en-IN should be DD/MM/YYYY."
    evidence_excerpt: "MM/DD/YYYY"
    related_surfaces: ["dp-auth-step1-dob"]
    recommendation: "Use 'DD/MM/YYYY' for en-IN. Or, better, use a native date picker and drop the placeholder string."
    depends_on: null

  - finding_id: BV-049
    surface_id: mc-onb-step3-subtitle
    lens: brand-voice
    severity: P3
    issue: "British spelling 'favourite cuisines to get personalised recommendations' on mobile-customer — web app uses 'Favourite Cuisines' in StepPreferences too. Consistent on en-GB, but document the en-GB choice in style guide."
    evidence_excerpt: "Select your favourite cuisines to get personalised recommendations."
    related_surfaces: ["web-auth-onboarding-step-preferences"]
    recommendation: "Lock en-GB (favourite, personalised, kilometre) as the project locale in STYLE-GUIDE §6. Today this is consistent but undocumented — it'll drift the moment a new screen is written."
    depends_on: null

  # ─── P3: Punctuation budget  ────────────────────────────────────────────────

  - finding_id: BV-050
    surface_id: mv-onb-pending-title
    lens: brand-voice
    severity: P2
    issue: "'Application Submitted!' on chef mobile pending screen uses an exclamation mark on a status surface. STYLE-GUIDE §1: exclamation-mark budget = 1 per page customer-facing, 0 vendor/driver."
    evidence_excerpt: "Application Submitted!"
    related_surfaces: ["md-onb-113"]
    recommendation: "Remove exclamation: 'Application submitted.' Mobile chef = chef persona = 0 exclamation marks."
    depends_on: null

  - finding_id: BV-051
    surface_id: vp-onb-personal-blurb
    lens: brand-voice
    severity: P3
    issue: "Two-sentence helper 'Tell us about yourself. This information helps verify your identity.' is verbose for an onboarding-step blurb. Plain English principle: one sentence, no jargon."
    evidence_excerpt: "Tell us about yourself. This information helps verify your identity."
    related_surfaces: ["vp-onb-personal-address-blurb"]
    recommendation: "'We use this to verify your identity.' (drop redundant 'Tell us about yourself' — the screen already shows that). Apply same trim to 'Kitchen Address' blurb."
    depends_on: null

  # ─── P1: Inconsistent CTA verbs on submit ───────────────────────────────────

  - finding_id: BV-052
    surface_id: dp-auth-step5-submit
    lens: brand-voice
    severity: P1
    issue: "Final submit CTA drift: driver web 'Submit Application' / driver mobile 'Submit Application' / chef mobile review uses '(no inline excerpt — needs verify)' / customer onboarding has no terminal submit CTA in inventory."
    evidence_excerpt: "dp web: 'Submit Application' / md mobile: 'Submit Application' / web onboarding final step: '?'"
    related_surfaces: ["md-onb-106"]
    recommendation: "Canonical final-step CTA: 'Submit' or 'Submit application' (sentence case). Customer onboarding likely uses 'Finish' or similar — pick a single closing verb pattern. Verify chef-portal final CTA."
    depends_on: null

  # ─── P1: Recommended-plan label is decorative, not voice  ───────────────────

  - finding_id: BV-053
    surface_id: md-onb-084
    lens: brand-voice
    severity: P3
    issue: "'Recommended' badge on subscription plans — fine voice-wise but flagged because the recommendation logic is unclear from copy; if hardcoded, this approaches the AI-slop pattern (looks dynamic, isn't)."
    evidence_excerpt: "Recommended"
    related_surfaces: []
    recommendation: "Either show 'Recommended for your delivery volume' (with reason) or drop the badge. Bare 'Recommended' without explanation is brand-trust-eroding."
    depends_on: null

  # ─── P2: Forgot-password flow ───────────────────────────────────────────────

  - finding_id: BV-054
    surface_id: vp-auth-forgot-redirect
    lens: brand-voice
    severity: P3
    issue: "'Redirecting to password reset…' interim screen — stale Keycloak reference per inventory notes. Voice itself is OK but the screen may not be hit post-GIP migration."
    evidence_excerpt: "Redirecting to password reset..."
    related_surfaces: []
    recommendation: "Verify if this screen is still reachable. If yes, rewrite as 'Taking you to password reset…' (warmer). If no, remove dead screen + comment."
    depends_on: null

  # ─── P2: Web register marketing-bullets cleanup follow-up  ───────────────────

  - finding_id: BV-055
    surface_id: web-auth-register-heading
    lens: brand-voice
    severity: P2
    issue: "'Join Fe3dr Today' heading on register left panel — 'Today' adds artificial urgency, Title Case used. Trend-chasing copy pattern flagged in lens brief."
    evidence_excerpt: "Join Fe3dr Today"
    related_surfaces: ["web-auth-register-social"]
    recommendation: "Sentence case + drop 'Today': 'Sign up to Fe3dr' or 'Order from home chefs' (food-forward). STYLE-GUIDE Rule 1: confident, not loud."
    depends_on: null

  # ─── P1: vendor-portal "vendor" leakage on customer-facing wording  ─────────

  - finding_id: BV-056
    surface_id: vp-auth-login-features-list
    lens: brand-voice
    severity: P2
    issue: "Document verification & compliance' bullet phrasing for chef login is enterprise-SaaS tone, anti-reference per .impeccable.md ('Generic SaaS dashboard')."
    evidence_excerpt: "Document verification & compliance"
    related_surfaces: ["ap-auth-login-feature-list"]
    recommendation: "If bullet list survives (it shouldn't, per BV-024), rewrite as 'Document checks' or simply 'Verified status'. Avoid SaaS-speak."
    depends_on: null

  - finding_id: BV-057
    surface_id: dp-auth-step4-secure-pay
    lens: brand-voice
    severity: P3
    issue: "'Payments are securely processed via {gateway}.' — passive voice, slightly trust-bait. STYLE-GUIDE §5 active voice."
    evidence_excerpt: "Payments are securely processed via {gateway}. You can change your plan or cancel anytime from your dashboard."
    related_surfaces: ["dp-auth-step5-plan-summary"]
    recommendation: "'{gateway} handles your payment. Change or cancel anytime from your dashboard.' — active voice, shorter, less hand-wavy."
    depends_on: null

  # ─── P1: Mobile-vendor verbose placeholder violates persona-tone ────────────

  - finding_id: BV-058
    surface_id: mv-onb-kitchen-desc-ph
    lens: brand-voice
    severity: P2
    issue: "'Describe your kitchen, specialties, and cooking style (min 50 characters)' — 12-word placeholder; chef persona-tone target is crisp (5-12 words). Min-char requirement embedded in placeholder is also UX anti-pattern."
    evidence_excerpt: "Describe your kitchen, specialties, and cooking style (min 50 characters)"
    related_surfaces: ["vp-onb-kitchen-blurb"]
    recommendation: "Placeholder: 'A short story about your kitchen.' Helper line below: 'At least 50 characters.' Same pattern across web and mobile vendor."
    depends_on: null

  # ─── P2: Driver-app "rupee" hardcoded  ──────────────────────────────────────

  - finding_id: BV-059
    surface_id: dp-auth-step4-h
    lens: brand-voice
    severity: P2
    issue: "'A small subscription to keep Fe3dr running — you keep every rupee you earn.' — 'rupee' hardcoded; not i18n-safe. Also slightly self-justifying tone."
    evidence_excerpt: "A small subscription to keep Fe3dr running — you keep every rupee you earn."
    related_surfaces: ["dp-auth-step4-zero-commission"]
    recommendation: "'You keep 100% of what you earn from each delivery.' Drops 'rupee', drops the 'keep Fe3dr running' defensive note (admit it via the plan price itself, don't apologize for it). i18n-safe."
    depends_on: null

  # ─── P3: AppRegister page metric inventory leakage ──────────────────────────

  - finding_id: BV-060
    surface_id: web-auth-register-heading
    lens: brand-voice
    severity: P3
    issue: "Confirmed: same '500+' AI-slop appears at HomePage line 88, HeroSection line 123, and RegisterPage line 10. Cross-surface drift of identical placeholder string indicates copy was duplicated without source-of-truth, not extracted to a shared metric. P0 already filed at BV-001; this is the trace."
    evidence_excerpt: "RegisterPage:10 'Access to 500+ home chefs' / HomePage:88 '500+ Home Chefs Near You' / HomePage:155 '500+ Home Chefs' / HeroSection:123 value: '500+'"
    related_surfaces: ["web-home-hero", "web-home-stats"]
    recommendation: "Once chef count is live, expose a useChefCount() hook from API. Until then, delete ALL three duplicates. This is the kind of finding that should converge with a marketing-lens finding — link to the marketing BV file."
    depends_on: null
```
