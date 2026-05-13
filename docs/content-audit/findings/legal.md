# Findings — Legal

Category: legal
Lenses applied: technical-writer, legal, business-analyst, brand-voice
Inventory slice size: 47 surfaces
Total findings: 154

## Severity summary

| Severity | TW | Legal | BA | BV | Total |
|---|---|---|---|---|---|
| P0 | 1 | 16 | 3 | 5 | 25 |
| P1 | 9 | 24 | 9 | 8 | 50 |
| P2 | 17 | 12 | 8 | 8 | 45 |
| P3 | 21 | 8 | 3 | 2 | 34 |

## Cross-lens consensus surfaces (flagged by multiple lenses)

Surfaces flagged by 4 lenses (highest priority — every lens agreed there's a problem):

- **`vp-legal-kitchen-payout-warning`** — flagged by TW(P0), Legal(P0), BA(P0), BV(P1) — "Saved locally for now" banking copy
- **`vp-legal-settings-danger`** — flagged by TW(P1), Legal(P0×2), BA(P0), BV(P1) — "Demo mode" deactivation toast + retention disclosure
- **`mv-onb-policies-terms-body`** — flagged by TW(P1), Legal(P1×2), BA(P0), BV(P0, P1) — HomeChef vs Fe3dr brand drift + due-process + voice
- **`api-email-base-footer`** — flagged by TW(P1), Legal(P1×2, P2), BA(P1), BV(P0, P2) — "Fe3dr by HomeChef" footer + allergen disclosure + tagline drift
- **`vp-legal-profile-doc-types`** — flagged by TW(P1), Legal(P0), BA(P1), BV(P1) — FSSAI required/optional contradiction across surfaces
- **`vp-legal-policy-tos`** — flagged by TW(P1), Legal(P0, P1, P2), BA(P1), BV(P0) — commission/payout disclosure + document naming drift
- **`vp-legal-docs-banner-body`** — flagged by TW(P1), Legal(P0×2), BA(P1), BV(P2) — "encrypted and stored securely" claim + retention
- **`ap-chefs-reject-reason`** — flagged by TW(P1), Legal(P0), BA(P1), BV(P1) — hardcoded "Rejected by admin" with no actionable reason
- **`mv-settings-delete-contact-email`** — flagged by TW(P2), Legal(P1), BA(P1), BV(P0) — homechef.app vs fe3dr.com support email drift
- **`vp-legal-register-footer`** — flagged by TW(P2), Legal(P0), BA(P2), BV(P0) — "Vendor Terms" dead link at signup
- **`md-leg-001`** — flagged by TW(P2), Legal(P0), BA(P2), BV(P1) — driver onboarding ToS link not wired
- **`web-mkt-brand-copyright`** — flagged by TW(P3), Legal(P2), BA(P2), BV(P3) — copyright entity naming
- **`mv-onb-policies-checkbox`** — flagged by TW(P2), Legal(P1), BA(P2), BV(P1) — generic "terms and conditions" checkbox
- **`vp-legal-policy-hygiene`** — flagged by TW(P1), Legal(P1, P2), BA(P2), BV(P2) — "I commit" voice + inspection clause + measurable standard

Surfaces flagged by 3 lenses (also high-priority rewrites):

- **`vp-legal-docs-aadhaar`** — flagged by TW(P2), Legal(P0, P1), BA(P1) — Aadhaar consent and disclosure gaps
- **`web-legal-checkout-terms-note`** — flagged by TW(P2), Legal(P1×2), BA(P1) — refund/cancellation/GST disclosure at checkout
- **`vp-legal-policy-cancellation`** — flagged by TW(P1), Legal(P1, P2), BA(P1) — vague penalty thresholds + SLA reciprocity
- **`vp-legal-policies-compliance-confirm`** — flagged by TW(P3), Legal(P1), BA(P2) — bundled attestation needs per-item consent
- **`vp-legal-settings-stripe`** — flagged by TW(P2×2), Legal(P1×2), BA(P2) — gateway-switch consequences + KYC data flow disclosure
- **`vp-legal-login-footer`** — flagged by TW(P2), Legal(P1), BA(P2) — dead links + terms-version disclosure
- **`vp-legal-docs-cheque`** — flagged by TW(P3), Legal(P1), BA(P3) — bank-detail handling disclosure + skip consequence
- **`ap-secsettings-2fa-exempt`** — flagged by TW(P2), Legal(P1), BA(P3) — exemption policy + hardcoded service email
- **`vp-legal-docs-banner-title`** — flagged by TW(P3), Legal(P1), BV(P2) — DPDP §5 notice + Title Case
- **`vp-legal-policies-compliance-items`** — flagged by TW(P2), Legal(P0, P1), BV(P3) — vague hygiene standard + parallel structure
- **`vp-legal-docs-fssai`** — flagged by TW(P2), Legal(P0), BV(P3) — FSSAI optional vs mandatory under FSS Act

## Technical Writer findings

```yaml
findings:
  - finding_id: TW-001
    surface_id: web-legal-register-terms
    lens: technical-writer
    severity: P3
    issue: "Signup legal acknowledgment uses 'signing up' verb form, OK, but missing descriptive link context — bare 'Terms of Service' and 'Privacy Policy' phrases double as links and as defined terms with no surface-level explanation."
    evidence_excerpt: "By signing up, you agree to our Terms of Service and Privacy Policy"
    recommendation: "Acceptable as-is for short legal acknowledgment. If a tooltip is added, summarize each doc in one line (e.g., 'how we handle your data', 'rules for using Fe3dr'). Ensure same wording used on checkout (currently inconsistent — checkout omits Privacy Policy)."
    depends_on: null

  - finding_id: TW-002
    surface_id: web-legal-checkout-terms-note
    lens: technical-writer
    severity: P2
    issue: "Checkout legal acknowledgment is inconsistent with signup — references only 'Terms of Service' but signup references both Terms and Privacy. Same legal concept (consent) named/scoped differently across surfaces (Sec 3 inconsistency rule)."
    evidence_excerpt: "By placing this order, you agree to our Terms of Service"
    recommendation: "Align with signup: 'By placing this order, you agree to our Terms of Service and Privacy Policy.' Or document why Privacy is omitted here. Within-surface clarity is fine; cross-surface drift is the issue."
    depends_on: null

  - finding_id: TW-003
    surface_id: web-mkt-brand-copyright
    lens: technical-writer
    severity: P3
    issue: "Copyright footer uses 'All rights reserved.' — vestigial legal boilerplate that adds no value in modern footers. Plain-language test (Sec 5) flags this as ceremonial phrasing."
    evidence_excerpt: "© {year} Fe3dr. All rights reserved."
    recommendation: "Trim to '© {year} Fe3dr.' OR keep if Legal lens requires it — defer to Legal lens. TW flags as polish-level redundancy."
    depends_on: null

  - finding_id: TW-004
    surface_id: vp-legal-login-footer
    lens: technical-writer
    severity: P2
    issue: "Vendor portal login uses 'Terms of Service' but vendor portal register uses 'Vendor Terms'. Inconsistent naming of same legal concept across two adjacent screens in the same app (Sec 3 inconsistency rule)."
    evidence_excerpt: "By continuing, you agree to Fe3dr's Terms of Service and Privacy Policy"
    recommendation: "Decide on canonical name: 'Vendor Terms' (chef-specific) OR 'Terms of Service' (umbrella). Use the same on login and register. Recommend 'Vendor Terms' since vendors are signing a different agreement than customers."
    depends_on: null

  - finding_id: TW-005
    surface_id: vp-legal-register-footer
    lens: technical-writer
    severity: P2
    issue: "References 'Vendor Terms' but, per inventory note, has no link target. Either dead reference or undefined term. Surfaces a defined term without definition — violates Sec 5 'defined terms in bold first-use' principle when target doesn't exist."
    evidence_excerpt: "By registering, you agree to Fe3dr's Vendor Terms and Privacy Policy"
    recommendation: "Wire link target before shipping. If no Vendor Terms doc exists, change to 'Terms of Service' (or whatever single canonical doc covers vendor onboarding). Also: 'By registering' → 'By signing up' (Sec 3 vocab — 'Sign up' is preferred verb)."
    depends_on: null

  - finding_id: TW-006
    surface_id: vp-legal-docs-banner-title
    lens: technical-writer
    severity: P3
    issue: "Header conflates two distinct purposes ('verify your kitchen' AND 'enable payouts') in one sentence. Reads OK but a vendor scanning may miss either purpose."
    evidence_excerpt: "Documents help us verify your kitchen and enable payouts"
    recommendation: "Split or reorder: 'We use these to verify your kitchen and set up payouts.' Active voice, 'we'/'you' per Sec 5."
    depends_on: null

  - finding_id: TW-007
    surface_id: vp-legal-docs-banner-body
    lens: technical-writer
    severity: P1
    issue: "Two sentences, 26 words total — exceeds vendor-facing max of 20 words per sentence rule when combined as displayed. Also uses passive voice ('are encrypted', 'are only used') which Sec 5 forbids in legal copy."
    evidence_excerpt: "Your documents are encrypted and stored securely. They are only used for verification. FSSAI license is optional - many home chefs start without one and add it later."
    recommendation: "Active voice + plain language: 'We encrypt and store your documents securely, and we use them only to verify your kitchen. FSSAI license is optional — many chefs add it later.' Allergen/refund/encryption claim should also be a callout, not inline body text (Sec 5)."
    depends_on: null

  - finding_id: TW-008
    surface_id: vp-legal-docs-identity-title
    lens: technical-writer
    severity: P3
    issue: "Heading uses Title Case ('Identity Documents') in a UI that elsewhere uses sentence case. Helper text passive voice ('These are required to verify...')."
    evidence_excerpt: "Identity Documents / These are required to verify your identity and set up payouts."
    recommendation: "Sentence case: 'Identity documents'. Active voice: 'We need these to verify your identity and set up payouts.'"
    depends_on: null

  - finding_id: TW-009
    surface_id: vp-legal-docs-pan
    lens: technical-writer
    severity: P3
    issue: "Helper 'Required for tax purposes and payouts' uses vague phrase 'tax purposes'. Plain-language test — could be 'income tax reporting' for specificity."
    evidence_excerpt: "PAN Card / Required for tax purposes and payouts / PAN Number / ABCDE1234F / 10-character alphanumeric PAN"
    recommendation: "Helper: 'Required for income tax reporting and payouts.' Also: label is 'PAN Number' (Title Case) — should be sentence case 'PAN number' per Sec 4 form-label rule."
    depends_on: null

  - finding_id: TW-010
    surface_id: vp-legal-docs-aadhaar
    lens: technical-writer
    severity: P2
    issue: "For one of India's most sensitive IDs, helper text is just 'For identity verification' — 6 words with no plain-English explanation of what happens to it, who sees it, or why it's needed beyond the abstract noun. Missing helper text where field is non-obvious for legal/privacy implications."
    evidence_excerpt: "Aadhaar Card / For identity verification"
    recommendation: "Expand: 'Aadhaar card — we mask all but the last 4 digits after upload and use it only to verify you. Required by Indian KYC rules.' Aadhaar is highly sensitive; vague helper text is a P2 transparency issue."
    depends_on: null

  - finding_id: TW-011
    surface_id: vp-legal-docs-fssai
    lens: technical-writer
    severity: P2
    issue: "FSSAI is labeled 'Optional' here but is labeled 'Required' on the profile-level doc types (vp-legal-profile-doc-types). Same legal/regulatory concept named differently across two surfaces in the SAME app. Major within-app inconsistency."
    evidence_excerpt: "FSSAI License / If you have one - gives your profile a verified badge. You can add this later. / FSSAI License Number (Optional) / 14-digit FSSAI number."
    recommendation: "Reconcile with vp-legal-profile-doc-types where FSSAI is marked 'Required'. Decide: is it optional at onboarding and required later? If so, say so explicitly: 'Optional during onboarding. Required to keep selling after [X days].' Don't ship contradicting copy."
    depends_on: TW-021

  - finding_id: TW-012
    surface_id: vp-legal-docs-food-safety
    lens: technical-writer
    severity: P3
    issue: "Helper 'Any food handling or safety certification (optional)' ends with parenthetical '(optional)'. Sec 4 form-label rule: required indicator is asterisk on field, label stays clean — parallel rule should apply to optional state. Inconsistent label-state convention."
    evidence_excerpt: "Food Safety Training Certificate / Any food handling or safety certification (optional)"
    recommendation: "Move 'optional' out of helper text and into a label-state badge or omit (assume optional unless asterisked). Helper: 'Any food handling or safety certification you have.' Also Title Case → sentence case: 'Food safety training certificate.'"
    depends_on: null

  - finding_id: TW-013
    surface_id: vp-legal-docs-cheque
    lens: technical-writer
    severity: P3
    issue: "Label is 'Cancelled Cheque / Bank Proof' — dual label is ambiguous (is it one document with two names, or two acceptable documents?). Title Case violates Sec 4 form-label rule."
    evidence_excerpt: "Cancelled Cheque / Bank Proof / For setting up direct payouts to your bank account (optional, can add later)"
    recommendation: "Pick one canonical label: 'Cancelled cheque' (most specific). Helper: 'We use this to verify your bank account for payouts. You can add it later.' Active voice ('we use this') per Sec 5."
    depends_on: null

  - finding_id: TW-014
    surface_id: vp-legal-policies-compliance-title
    lens: technical-writer
    severity: P2
    issue: "Second sentence is 18 words and uses 'maintain quality standards' — vague phrasing. Combined block is 20 words, at the vendor-facing sentence max. Reading-ease borderline."
    evidence_excerpt: "Kitchen Compliance Checklist / Please confirm that your kitchen meets these basic food safety requirements. This ensures all home kitchens on Fe3dr maintain quality standards."
    recommendation: "Simplify: 'Confirm your kitchen meets these food safety basics. Every Fe3dr kitchen agrees to the same standards.' Drops to 5+11 words. Active voice, no filler ('Please confirm' → 'Confirm')."
    depends_on: null

  - finding_id: TW-015
    surface_id: vp-legal-policies-compliance-items
    lens: technical-writer
    severity: P2
    issue: "Compliance checklist items are inconsistent in grammatical form — mix of noun phrases ('clean water', 'waste disposal area') and verb phrases ('store raw and cooked separately', 'label allergens and ingredients'). Parallel structure principle for checklists."
    evidence_excerpt: "clean water / separate cutting boards veg vs non-veg / store raw and cooked separately / clean attire and personal hygiene / waste disposal area / freshly prepared / label allergens and ingredients"
    recommendation: "Pick one form. Recommend imperative-verb form for attestation: 'Use clean water', 'Keep separate cutting boards for veg and non-veg', 'Store raw and cooked food separately', 'Wear clean attire and maintain personal hygiene', 'Use a dedicated waste disposal area', 'Prepare food fresh for each order', 'Label allergens and ingredients on every dish.'"
    depends_on: null

  - finding_id: TW-016
    surface_id: vp-legal-policies-compliance-confirm
    lens: technical-writer
    severity: P3
    issue: "Attestation gate uses 'By proceeding' — passive framing. Plain-language test prefers direct verb. 14 words is fine for vendor."
    evidence_excerpt: "By proceeding, you confirm that all the above conditions are met in your kitchen."
    recommendation: "Active: 'By proceeding, you confirm your kitchen meets all the above.' (10 words). Or as checkbox label: 'My kitchen meets all the above conditions.'"
    depends_on: null

  - finding_id: TW-017
    surface_id: vp-legal-policies-agreements-title
    lens: technical-writer
    severity: P3
    issue: "Title 'Agreements & Policies' is Title Case. Helper 'Please read and accept each policy' is polite filler. Sec 5 active-voice preference."
    evidence_excerpt: "Agreements & Policies / Please read and accept each policy to complete your registration."
    recommendation: "Sentence case + plainer: 'Agreements and policies / Read and accept each one to finish signing up.' Also vocab: 'registration' → 'signing up' (Sec 3)."
    depends_on: null

  - finding_id: TW-018
    surface_id: vp-legal-policy-hygiene
    lens: technical-writer
    severity: P1
    issue: "52-word policy block uses formal first-person legal voice ('I commit to maintaining...'). Sec 5 requires 'we'/'you' in legal copy — not 'I'. Mixing chef-as-signer perspective into UI body text reads as a checkbox-disguised-as-contract. Also likely exceeds 25-word sentence max."
    evidence_excerpt: "Kitchen Hygiene & Food Safety Commitment / I commit to maintaining a clean and hygienic kitchen at all times... Fe3dr may conduct periodic kitchen checks."
    recommendation: "Reframe as plain 'you' statement, attest via checkbox: 'You agree to: keep your kitchen clean and hygienic, follow food safety practices, and allow Fe3dr to do periodic kitchen checks.' Bullet the obligations; one idea per paragraph per Sec 5."
    depends_on: null

  - finding_id: TW-019
    surface_id: vp-legal-policy-cancellation
    lens: technical-writer
    severity: P1
    issue: "56-word block in formal 'I' voice. Buries operational consequences in legal prose ('Repeated late cancellations may affect my kitchen rating and visibility') — Sec 5 says consequences get callout treatment, not buried. Likely also >25 word sentences."
    evidence_excerpt: "Order & Cancellation Policy / accept or reject orders within 5 minutes / prepare within estimated time / Repeated late cancellations may affect my kitchen rating and visibility."
    recommendation: "Reframe in 'you' voice. Bullet the obligations. Pull consequences into a callout: 'Heads up: repeated late cancellations lower your kitchen rating and reduce visibility in search.' Per Sec 5 (allergen/refund clauses get callouts) — operational penalties should too."
    depends_on: null

  - finding_id: TW-020
    surface_id: vp-legal-policy-tos
    lens: technical-writer
    severity: P1
    issue: "46-word TOS block in 'I' voice commits chef to commission + weekly payouts. Two material business terms (commission %, payout cadence) buried in one sentence with no callout. Sec 5 says financial commitments need callout treatment."
    evidence_excerpt: "Fe3dr Platform Terms of Service / I have read and agree... Fe3dr charges a platform commission on each order and that my payouts will be processed weekly to my registered bank account."
    recommendation: "Pull payment terms into a callout: '**Money basics.** Fe3dr keeps a platform commission on each order. We pay out the rest to your bank account weekly.' Body becomes 'you have read and agree...'. Per Sec 5, defined terms ('Platform Commission', 'Payout') in bold on first use."
    depends_on: null

  - finding_id: TW-021
    surface_id: vp-legal-kitchen-payout-warning
    lens: technical-writer
    severity: P0
    issue: "Banking-data collection warning says 'saved locally for now' — factually misleading copy that suggests data lives only on-device when the field is in a web app form (i.e., 'locally' is ambiguous — local storage? local to demo? local to backend?). Sec 5 plain-language test + P0 severity guide (factually misleading copy)."
    evidence_excerpt: "Payout Details / Bank account details for receiving your earnings / Payout integration coming soon. These details are saved locally for now."
    recommendation: "Clarify exactly where the data lives. If demo-only: 'Payouts coming soon. We're not collecting bank details yet — this form is preview only.' If saved to backend awaiting wiring: 'Payouts coming soon. We're storing these details securely and will use them once payouts go live.' Never say 'locally' without specifying."
    depends_on: null

  - finding_id: TW-022
    surface_id: vp-legal-profile-doc-types
    lens: technical-writer
    severity: P1
    issue: "FSSAI marked 'Required' here but 'Optional' on onboarding (vp-legal-docs-fssai). Same regulatory document in same app named with opposing legal weight. Direct contradiction visible to the same chef across two screens — P1 conversion-critical + Sec 3 inconsistency."
    evidence_excerpt: "FSSAI License (Food Safety and Standards Authority certificate) / PAN Card (Permanent Account Number card) / Aadhaar Card / Food Safety Certificate / Cancelled Cheque / Required"
    recommendation: "Reconcile required/optional state across onboarding + profile. Pick one truth and propagate. See TW-011. Also: parenthetical full-name expansions ('Food Safety and Standards Authority certificate') only needed on first reference per page; consider tooltip pattern instead."
    depends_on: null

  - finding_id: TW-023
    surface_id: vp-legal-settings-stripe
    lens: technical-writer
    severity: P2
    issue: "75-word block is a wall of mixed concepts: positioning ('for chefs outside India'), feature description, KYC disclosure, action buttons ('Resume Onboarding', 'Make Stripe My Primary Gateway'), status labels. Multiple Sec 4 violations: 'Make Stripe My Primary Gateway' is a 5-word button (max 3); 'Resume Onboarding' is Title Case button (should be sentence case 'Resume onboarding')."
    evidence_excerpt: "Stripe (International Payouts) / For chefs outside India, or as an alternative to Razorpay. / ... / Make Stripe My Primary Gateway / Switch Back to Razorpay / Refresh Status"
    recommendation: "Button rewrites (verb-first, ≤3 words, sentence case): 'Make Stripe My Primary Gateway' → 'Set as primary'; 'Switch Back to Razorpay' → 'Switch to Razorpay'; 'Refresh Status' → 'Refresh status'; 'Resume Onboarding' → 'Resume setup'; 'Connect with Stripe' → 'Connect Stripe'. Break body into 'About Stripe' (1 sentence) + 'How KYC works' (1 sentence) + status labels."
    depends_on: null

  - finding_id: TW-024
    surface_id: vp-legal-settings-stripe
    lens: technical-writer
    severity: P2
    issue: "'Stripe needs more information before you can accept payments.' — vague pronoun referent ('more information' — what kind?). Plain-language test: tell the chef what's missing."
    evidence_excerpt: "Stripe needs more information before you can accept payments. Resume onboarding to finish."
    recommendation: "If Stripe API tells us what's missing, surface it: 'Stripe needs a few more details (ID and bank info). Resume setup to finish.' If not, at least: 'Stripe needs more details before you can accept payments. Resume setup to finish.'"
    depends_on: null

  - finding_id: TW-025
    surface_id: vp-legal-settings-danger
    lens: technical-writer
    severity: P1
    issue: "'Account deactivation is not available in demo mode' shipped to vendor portal settings. If this is reaching production, it's a P1 trust break — chefs see 'demo mode' on a live app. Also: 'Danger Zone' is a Github-ism that doesn't match Fe3dr's calm/quietly-modern brand voice (Sec 1 Rule 5)."
    evidence_excerpt: "Danger Zone / Deactivating your account will hide your kitchen from customers and pause all orders. / Deactivate Account / Account deactivation is not available in demo mode"
    recommendation: "Verify 'demo mode' is not in production. If staging-only, gate the string. 'Danger Zone' → 'Deactivate account' (the section IS the action). Button 'Deactivate Account' → 'Deactivate account' (sentence case). Body is fine — clear consequence statement per Sec 4 modal subtitle rule."
    depends_on: null

  - finding_id: TW-026
    surface_id: ap-approvaldetail-notes-helper
    lens: technical-writer
    severity: P3
    issue: "Helper text is clear and active-voice — borderline within admin tone. Could be tighter."
    evidence_excerpt: "These notes will be recorded in the history and visible to other admins."
    recommendation: "Tighten: 'Saved to history. Visible to other admins.' (8 words) — matches admin tone matrix (Sec 2: precise, no fluff)."
    depends_on: null

  - finding_id: TW-027
    surface_id: ap-auth-restricted-heading
    lens: technical-writer
    severity: P3
    issue: "'Restricted Access' is Title Case; admin pages should match the sentence-case convention used elsewhere. Also could be more informative."
    evidence_excerpt: "Restricted Access"
    recommendation: "Sentence case: 'Restricted access' or more specific 'Admin portal — sign-in required'."
    depends_on: null

  - finding_id: TW-028
    surface_id: ap-auth-restricted-body
    lens: technical-writer
    severity: P3
    issue: "'Only pre-authorized administrators can sign in. New registrations are not allowed.' — second sentence uses banned form 'registrations' (Sec 3: 'Sign up' is preferred). Two sentences saying the same thing."
    evidence_excerpt: "Only pre-authorized administrators can sign in. New registrations are not allowed."
    recommendation: "Combine: 'Only pre-authorized admins can sign in. You can't sign up here.' Or just 'Only pre-authorized admins can sign in.'"
    depends_on: null

  - finding_id: TW-029
    surface_id: ap-auth-footer-legal
    lens: technical-writer
    severity: P3
    issue: "'Fe3dr Administration Portal · Internal Use Only' — 'Administration' is jargon for 'Admin'. Plain-language test."
    evidence_excerpt: "Fe3dr Administration Portal · Internal Use Only"
    recommendation: "'Fe3dr Admin · Internal use only' (sentence case on second clause)."
    depends_on: null

  - finding_id: TW-030
    surface_id: ap-chefs-reject-reason
    lens: technical-writer
    severity: P1
    issue: "'Rejected by admin' is the default reason sent to a chef when no specific reason is given — it tells the chef nothing about why, what to do next, or how to appeal. Sec 4 error formula: 'what happened → what to do.' This is what happened only."
    evidence_excerpt: "Rejected by admin"
    recommendation: "Replace generic with structured fallback: 'Application rejected. Contact support@fe3dr.com to learn why or appeal.' If a real reason should be required, gate the action behind a required-reason field — don't ship a copy-only fallback."
    depends_on: null

  - finding_id: TW-031
    surface_id: ap-auditlogs-subtitle
    lens: technical-writer
    severity: P3
    issue: "Subtitle is admin-precise and works. Minor: 'before/after values' is jargon-adjacent for non-engineer admins."
    evidence_excerpt: "Admin actions recorded with actor, IP, user agent, and before/after values."
    recommendation: "Acceptable for admin tone. Optional: 'Admin actions recorded with the admin's identity, IP, browser, and what changed.'"
    depends_on: null

  - finding_id: TW-032
    surface_id: ap-secsettings-2fa-enforce
    lens: technical-writer
    severity: P2
    issue: "25-word combined string is at the customer-facing max but applied to admin (where 20-word vendor cap is more typical). Two sentences sharing the toggle label is awkward; helper text should be separate and shorter."
    evidence_excerpt: "Require 2FA for all admins; When enabled, admins without 2FA are prompted to enroll on their next login."
    recommendation: "Label: 'Require 2FA for all admins'. Helper: 'Admins without 2FA enroll on next sign-in.' (8 words, active voice, 'sign-in' per Sec 3 not 'login')."
    depends_on: null

  - finding_id: TW-033
    surface_id: ap-secsettings-2fa-exempt
    lens: technical-writer
    severity: P2
    issue: "35-word helper text exceeds even customer-facing 25-word cap. Contains nested parenthetical ('(E2E tests, automation)') and three separate ideas: format ('one per line'), use case, and default behavior. Sec 5 'one idea per paragraph.'"
    evidence_excerpt: "2FA exempt emails; One per line. Useful for service accounts (E2E tests, automation) that can't scan a QR. Leave empty to enforce 2FA on every admin."
    recommendation: "Split: Label 'Exempt emails'. Helper line 1: 'One email per line.' Helper line 2: 'Use for service accounts that can't scan a QR.' Placeholder text or footnote: 'Leave empty to require 2FA for every admin.'"
    depends_on: null

  - finding_id: TW-034
    surface_id: ap-secsettings-apikey-warn
    lens: technical-writer
    severity: P2
    issue: "Warning is one-time-secret disclosure but uses em-dash mid-sentence and lacks a what-to-do. Sec 4 error pattern: what happened → what to do."
    evidence_excerpt: "Copy this key now — you won't see it again"
    recommendation: "'Copy this key now. We won't show it again.' (active voice, period not em-dash, parallel sentence structure)."
    depends_on: null

  - finding_id: TW-035
    surface_id: ap-settings-payment-helpers
    lens: technical-writer
    severity: P2
    issue: "Helper text is duplicated verbatim across two payment fields — Sec 3 inconsistency check passes (they match each other) but they're redundant and 'GCP Secret Manager' is jargon for a non-engineer admin who might handle payment keys."
    evidence_excerpt: "Keys are stored securely in GCP Secret Manager. Leave a field empty to keep its current value."
    recommendation: "De-jargon: 'Stored securely in our secret manager. Leave empty to keep current value.' Only one of the two fields needs the storage disclosure — put it once at the section level, not per field."
    depends_on: null

  - finding_id: TW-036
    surface_id: ap-staffdetail-role-descriptions
    lens: technical-writer
    severity: P3
    issue: "Role descriptions are short noun phrases, mostly parallel, but 'Full access to everything' is colloquial vs 'Admin portal management' (formal). Mixed register."
    evidence_excerpt: "Full access to everything; Admin portal management; Delivery fleet management; Delivery operations; Read-only support access"
    recommendation: "Parallel structure: 'Full platform access'; 'Admin portal management'; 'Delivery fleet management'; 'Delivery operations'; 'Read-only support access'. All noun-phrase, similar length."
    depends_on: null

  - finding_id: TW-037
    surface_id: mv-onb-policies-checkbox
    lens: technical-writer
    severity: P2
    issue: "'I accept the terms and conditions' uses 'terms and conditions' — banned generic phrasing. App uses 'Terms of Service' elsewhere. Sec 3 inconsistency + Sec 5 'we/you' principle ('I accept' vs other apps' 'you agree')."
    evidence_excerpt: "I accept the terms and conditions"
    recommendation: "'I agree to the Vendor Terms and Privacy Policy.' Match the named documents used elsewhere (vp-legal-register-footer). Or stay in 'you' voice if the checkbox sits under a 'By signing up, you agree to...' line."
    depends_on: TW-004

  - finding_id: TW-038
    surface_id: mv-onb-policies-cancel-label
    lens: technical-writer
    severity: P3
    issue: "'Cancellation Policy *' — Title Case label and uses asterisk inline (Sec 4 form-label rule: required indicator is the asterisk on field, label stays clean — should not be in label text)."
    evidence_excerpt: "Cancellation Policy *"
    recommendation: "'Cancellation policy' (sentence case, no inline asterisk; required state via field-level asterisk indicator)."
    depends_on: null

  - finding_id: TW-039
    surface_id: mv-settings-delete-confirm-body
    lens: technical-writer
    severity: P2
    issue: "'To delete your account, please contact our support team. This action cannot be undone.' — second sentence is the exact pattern Sec 4 modal-subtitle rule bans ('Are you sure? This action cannot be undone' — vague). Doesn't explain WHAT can't be undone in concrete terms."
    evidence_excerpt: "To delete your account, please contact our support team. This action cannot be undone."
    recommendation: "'Email support to delete your account. Deletion is permanent — you'll lose your menu, orders, and payouts history.' (concrete consequences per Sec 4)."
    depends_on: null

  - finding_id: TW-040
    surface_id: mv-settings-delete-contact-action
    lens: technical-writer
    severity: P3
    issue: "'Contact Support' button — Title Case. Sec 4 button formula: verb-first, ≤3 words, sentence case."
    evidence_excerpt: "Contact Support"
    recommendation: "'Email support' (verb-first, action-explicit, sentence case)."
    depends_on: null

  - finding_id: TW-041
    surface_id: mv-settings-delete-contact-email
    lens: technical-writer
    severity: P2
    issue: "Email is 'support@homechef.app' but inventory notes brand drift — other surfaces use Fe3dr (fe3dr.com). Within-surface fine but cross-app brand inconsistency (Sec 3)."
    evidence_excerpt: "Please email support@homechef.app to delete your account."
    recommendation: "Verify canonical support email (support@fe3dr.com vs support@homechef.app). Pick one and propagate. Also: drop 'Please' (Sec 1 Rule 1, confident not deferential): 'Email support@fe3dr.com to delete your account.'"
    depends_on: null

  - finding_id: TW-042
    surface_id: mv-onb-policies-cancel-option-no
    lens: technical-writer
    severity: P3
    issue: "'No cancellations after order accepted' — fine but ambiguous tense ('order accepted' is past participle hanging). Parallel structure with other cancel options."
    evidence_excerpt: "No cancellations after order accepted"
    recommendation: "'No cancellations after I accept the order' OR (more consistent with other options): 'Not allowed after order is accepted'. Match parallel form across cancel-option-1h, cancel-option-30m."
    depends_on: null

  - finding_id: TW-043
    surface_id: mv-onb-policies-cancel-option-1h
    lens: technical-writer
    severity: P3
    issue: "'Up to 1 hour before prep start' — terse, parallel with -30m, OK. Minor: 'prep start' is jargon — could be 'cooking starts.'"
    evidence_excerpt: "Up to 1 hour before prep start"
    recommendation: "Acceptable. Optional: 'Up to 1 hour before cooking starts' (plain English, Sec 5)."
    depends_on: null

  - finding_id: TW-044
    surface_id: mv-onb-policies-cancel-option-30m
    lens: technical-writer
    severity: P3
    issue: "'Up to 30 mins before prep start' — uses 'mins' abbreviation; elsewhere app uses 'min' or 'minutes'. Sec 6 numerals/formatting expects consistent unit notation."
    evidence_excerpt: "Up to 30 mins before prep start"
    recommendation: "'Up to 30 minutes before cooking starts.' Spell out 'minutes' for clarity (and translation-readiness — Sec 7)."
    depends_on: TW-043

  - finding_id: TW-045
    surface_id: mv-onb-policies-terms-body
    lens: technical-writer
    severity: P1
    issue: "60-word block with two very long sentences (one is ~38 words). Sec 5 says max 25 words per sentence in legal copy. Uses 'reserves the right to suspend' — formal legal jargon. Also references 'homechef.in/vendor-terms' which is brand-inconsistent with fe3dr.com elsewhere. Sec 3 'we/you' principle: uses 'HomeChef' as third-party rather than 'we'."
    evidence_excerpt: "By joining HomeChef as a vendor you agree to maintain food hygiene standards per FSSAI regulations... HomeChef reserves the right to suspend accounts that receive repeated hygiene complaints or fail document verification."
    recommendation: "Rewrite into short sentences, 'we'/'you' voice: 'By joining Fe3dr as a chef, you agree to: keep food hygiene standards per FSSAI rules, write accurate menu descriptions, prepare orders within your stated prep time, and follow local food safety laws. We may suspend accounts with repeated hygiene complaints or failed document checks. Full terms: fe3dr.com/vendor-terms.' Also resolve brand drift (HomeChef vs Fe3dr)."
    depends_on: null

  - finding_id: TW-046
    surface_id: md-leg-001
    lens: technical-writer
    severity: P2
    issue: "'Terms of Service' on driver onboarding review screen — note says no navigation wired. Same legal concept named differently than mobile-vendor ('terms and conditions') and vendor portal ('Vendor Terms'). Three-app drift. Also: dead link is a P2 trust break."
    evidence_excerpt: "Terms of Service"
    recommendation: "Decide naming convention by audience: 'Driver Terms' (driver-specific), 'Vendor Terms' (chef-specific), 'Terms of Service' (customer umbrella). Wire link before ship — TW won't flag a real link, but a label-only legal reference is misleading."
    depends_on: TW-004

  - finding_id: TW-047
    surface_id: md-leg-002
    lens: technical-writer
    severity: P3
    issue: "'Privacy Policy' on driver onboarding review screen — assuming link wires, OK. Driver-facing copy generally fine at 2 words per Sec 2 driver tone (telegraphic)."
    evidence_excerpt: "Privacy Policy"
    recommendation: "No change needed if linked. If not linked, see TW-046."
    depends_on: TW-046

  - finding_id: TW-048
    surface_id: api-email-base-footer
    lens: technical-writer
    severity: P1
    issue: "'Fe3dr by HomeChef · Authentic homemade food, delivered · fe3dr.com · Privacy · Terms' — brand drift: header says 'Fe3dr' only; footer says 'Fe3dr by HomeChef'. Also 'Authentic homemade food' uses 'Authentic' which is brand-drift-adjacent (similar register to banned 'Artisanal' per Sec 3 banned brand-drift terms). Plus inconsistent legal-link names ('Privacy' vs 'Privacy Policy' elsewhere; 'Terms' vs 'Terms of Service')."
    evidence_excerpt: "Fe3dr by HomeChef · Authentic homemade food, delivered · fe3dr.com · Privacy · Terms"
    recommendation: "'Fe3dr · Home-cooked food, delivered · fe3dr.com · Privacy Policy · Terms of Service'. Drops brand drift, drops 'Authentic' (Sec 3 banned-adjacent), aligns link names with web/vp surfaces."
    depends_on: null
```

## Legal findings

```yaml
# Legal lens findings — Home Chef Content Audit
# Scope: India regulatory exposure (DPDP Act 2023, FSSAI, RBI Payment Aggregator, GST) + generic best-practice
# IMPORTANT: All findings carry depends_on: "needs lawyer review" — this audit is not legal advice.

findings:
  # ============================================================================
  # CROSS-CUTTING / MISSING-ENTIRELY FINDINGS (apps with no legal pages at all)
  # ============================================================================

  - finding_id: LEG-001
    surface_id: web-legal-register-terms
    lens: legal
    severity: P0
    issue: "Signup links to 'Terms of Service' and 'Privacy Policy' but no such pages exist in apps/web; clicking either link will 404 or break the agreement"
    evidence_excerpt: "By signing up, you agree to our Terms of Service and Privacy Policy"
    recommendation: "Ship actual T&C and Privacy Policy pages at /terms and /privacy before any production signup; under DPDP Act §5 (Notice) and Indian Contract Act, consent referencing a missing document is unenforceable. Also add granular DPDP-grade notice listing purposes of data collection, retention period, data fiduciary identity, and grievance officer contact."
    citation: "DPDP Act 2023 §5 (Notice), §6 (Consent); Indian Contract Act 1872 §10"
    depends_on: "needs lawyer review"

  - finding_id: LEG-002
    surface_id: web-legal-register-terms
    lens: legal
    severity: P0
    issue: "Bundled consent — single 'by signing up' clause covers both T&C and Privacy in one act, with no granular opt-in for marketing/analytics/third-party sharing"
    evidence_excerpt: "By signing up, you agree to our Terms of Service and Privacy Policy"
    recommendation: "DPDP Act requires consent to be 'free, specific, informed, unconditional and unambiguous' (§6). Split into: (a) T&C acceptance (contractual, can be implied), (b) Privacy Notice acknowledgment, (c) optional purpose-by-purpose opt-ins (marketing emails, analytics, third-party sharing). Pre-checked boxes are prohibited."
    citation: "DPDP Act 2023 §6(1)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-003
    surface_id: web-legal-checkout-terms-note
    lens: legal
    severity: P1
    issue: "Order placement references T&C only, not Refund Policy or Cancellation Policy — RBI PA Master Direction requires refund timeline disclosure at point of payment"
    evidence_excerpt: "By placing this order, you agree to our Terms of Service"
    recommendation: "Add explicit reference to Refund Policy + Cancellation Policy at checkout. State maximum refund timeline (commonly ≤7 working days for digital refunds per RBI PA framework). Also link to FSSAI-mandated allergen/ingredient disclosure for the order."
    citation: "RBI Payment Aggregator Master Direction (17 Mar 2020, updated 2024); FSSAI Regulations 2011 Ch. 2"
    depends_on: "needs lawyer review"

  - finding_id: LEG-004
    surface_id: md-leg-001
    lens: legal
    severity: P0
    issue: "Driver onboarding shows 'Terms of Service' link but inventory notes 'no navigation wired in code' — driver is being asked to accept terms they cannot read"
    evidence_excerpt: "Terms of Service"
    recommendation: "Wire navigation to actual driver-specific T&C page covering gig-worker classification (independent contractor vs employee), insurance disclosure, earnings/deductions transparency, and termination clause. Without a readable contract, the agreement is voidable under Indian Contract Act §10."
    citation: "Indian Contract Act 1872 §10; Code on Social Security 2020 (gig worker provisions)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-005
    surface_id: md-leg-002
    lens: legal
    severity: P0
    issue: "Driver onboarding shows 'Privacy Policy' link with no navigation wired — driver location tracking, ID docs, banking info are collected without a readable privacy notice"
    evidence_excerpt: "Privacy Policy"
    recommendation: "Wire link to a driver-specific Privacy Notice that discloses: continuous GPS tracking, retention period for location history, biometric/ID handling, payment-data flow, and grievance officer contact. Drivers are data principals under DPDP and require notice at the time of data collection."
    citation: "DPDP Act 2023 §5; §13 (Grievance redressal)"
    depends_on: "needs lawyer review"

  # ============================================================================
  # FSSAI / FOOD SAFETY (HIGHEST EXPOSURE)
  # ============================================================================

  - finding_id: LEG-006
    surface_id: vp-legal-docs-fssai
    lens: legal
    severity: P0
    issue: "FSSAI license labeled '(Optional)' during chef onboarding — but FSSAI registration is mandatory for any food business, including home-based, with turnover obligations under FSS Act 2006"
    evidence_excerpt: "FSSAI License / If you have one - gives your profile a verified badge. You can add this later. / FSSAI License Number (Optional)"
    recommendation: "Per FSS Act 2006 §31, no food business can operate without an FSSAI registration/license. Home chefs with annual turnover ≤ ₹12 lakh need Basic Registration; above that, State/Central licence. 'Optional' framing exposes platform to aiding unlicensed food business operation. Reframe as 'FSSAI Registration is required by Indian law — Basic Registration is free and takes ~7 days'. Block listing/payouts until provided OR add a verified disclosure that the chef holds at least Basic Registration."
    citation: "FSS Act 2006 §31; FSS (Licensing & Registration) Regulations 2011 Reg. 2.1"
    depends_on: "needs lawyer review"

  - finding_id: LEG-007
    surface_id: vp-legal-profile-doc-types
    lens: legal
    severity: P0
    issue: "FSSAI labeled 'Required' on Profile DOCUMENT_TYPES but 'Optional' on onboarding — internally inconsistent contract; chef can complete onboarding without FSSAI but is told it's required at profile level"
    evidence_excerpt: "FSSAI License (Food Safety and Standards Authority certificate) ... Required"
    recommendation: "Resolve contradiction across all surfaces. If FSSAI is required (correct under FSS Act 2006), enforce it at onboarding too. Inconsistent disclosures across screens create estoppel risk and consumer-protection exposure for misrepresentation."
    citation: "FSS Act 2006 §31; Consumer Protection Act 2019 §2(28) (misleading representation)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-008
    surface_id: vp-legal-docs-food-safety
    lens: legal
    severity: P1
    issue: "Food Safety Training Certificate marked optional — FoSTaC training is mandated by FSSAI for at least one Food Safety Supervisor per food business (Schedule 4 of FSS Regulations)"
    evidence_excerpt: "Food Safety Training Certificate / Any food handling or safety certification (optional)"
    recommendation: "Verify with counsel whether home-chef setups fall under FoSTaC mandate. If yes, change framing from 'optional' to 'required for chefs preparing >X meals/day' with clear threshold disclosure. Reference: Schedule 4 of FSS (Licensing & Registration) Regulations 2011."
    citation: "FSS (Licensing & Registration) Regulations 2011 Schedule 4 (FoSTaC)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-009
    surface_id: vp-legal-policies-compliance-items
    lens: legal
    severity: P0
    issue: "7-item food-safety self-attestation list is load-bearing for FSSAI compliance, platform liability, and consumer-protection defence — but uses vague language ('clean water', 'clean attire') with no measurable standard"
    evidence_excerpt: "clean water / separate cutting boards veg vs non-veg / store raw and cooked separately / clean attire and personal hygiene / waste disposal area / freshly prepared / label allergens and ingredients"
    recommendation: "Each item should map to a specific clause in FSS (Licensing & Registration) Regulations 2011 Schedule 4 (Part II for home-based) so attestation is legally meaningful. E.g., 'potable water meeting IS 10500:2012'. Without measurable criteria, attestation provides no liability shield and may be unenforceable as too vague."
    citation: "FSS (Licensing & Registration) Regulations 2011 Schedule 4; IS 10500:2012 (potable water)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-010
    surface_id: vp-legal-policies-compliance-confirm
    lens: legal
    severity: P1
    issue: "Single tick-box attestation for 7 distinct compliance items — combined consent prevents partial truthful answers; encourages 'click-through' behavior"
    evidence_excerpt: "By proceeding, you confirm that all the above conditions are met in your kitchen."
    recommendation: "Require individual attestation per item (separate checkbox per row). Combined attestation is harder to enforce in a Consumer Forum action and easier to dismiss as boilerplate. Per FSSAI good-practice and DPDP §6 spirit (specific consent), break apart bundled affirmations."
    citation: "best-practice; DPDP Act 2023 §6 (specificity by analogy)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-011
    surface_id: vp-legal-policy-hygiene
    lens: legal
    severity: P1
    issue: "Hygiene policy commits chef to 'periodic kitchen checks' by Fe3dr — but does not state inspection frequency, notice period, audit rights of refusal, or who bears cost"
    evidence_excerpt: "I commit to maintaining a clean and hygienic kitchen at all times... Fe3dr may conduct periodic kitchen checks."
    recommendation: "Specify: notice required before inspection, frequency cap, recordable findings, chef's right to be present, and remediation window before delisting. Vague inspection rights may be unconscionable under Consumer Protection Act for the chef-as-platform-user; also creates ambiguity for platform's own auditors."
    citation: "Consumer Protection Act 2019; best-practice (contract clarity)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-012
    surface_id: vp-legal-policy-cancellation
    lens: legal
    severity: P1
    issue: "'Repeated late cancellations may affect my kitchen rating and visibility' — penalty terms are vague ('may'), with no defined threshold, appeal process, or proportionality"
    evidence_excerpt: "Repeated late cancellations may affect my kitchen rating and visibility."
    recommendation: "Define: count threshold ('3 late cancellations in 30 days'), visibility penalty magnitude, appeal/grievance pathway (mandated by DPDP §13 for data-based decisions, and CCPA-style for platform decisions), and reset period. Indeterminate enforcement clauses are challengeable in Consumer Forums."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 5(3) (grievance mechanism)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-013
    surface_id: vp-legal-policy-tos
    lens: legal
    severity: P0
    issue: "Vendor TOS clause commits chef to 'commission' and 'weekly payouts' without disclosing commission percentage, TDS withholding, GST treatment, or settlement cycle definition"
    evidence_excerpt: "Fe3dr charges a platform commission on each order and that my payouts will be processed weekly to my registered bank account."
    recommendation: "Disclose: (a) commission rate or rate-card link, (b) GST treatment (platform charges GST on commission to chef under reverse charge), (c) TDS withholding under §194-O of Income Tax Act (1% TDS on e-commerce participants), (d) payout cycle definition (which day, what cutoff). Failure to disclose §194-O TDS at point of agreement is a known dispute trigger."
    citation: "Income Tax Act 1961 §194-O; CGST Act 2017 §9(5); RBI Payment Aggregator Master Direction (settlement timelines)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-014
    surface_id: mv-onb-policies-terms-body
    lens: legal
    severity: P1
    issue: "Vendor terms reference URL 'homechef.in/vendor-terms' but inventory notes URL is unverified and only referenced from mobile vendor app — drift risk: same TOS likely not mirrored on web vendor-portal"
    evidence_excerpt: "Full terms available at homechef.in/vendor-terms."
    recommendation: "Verify the URL resolves, is owned by the same legal entity, and serves identical text to what is binding on vendor-portal users. Differing TOS across surfaces is unenforceable as to the version not actually seen by user. Also check domain registration matches the brand identity used elsewhere (Fe3dr vs HomeChef inconsistency — see LEG-046)."
    citation: "Indian Contract Act 1872 §10; best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-015
    surface_id: mv-onb-policies-terms-body
    lens: legal
    severity: P1
    issue: "Mobile vendor terms claim platform may 'suspend accounts that receive repeated hygiene complaints' — no due-process step, no opportunity to respond, no appeal"
    evidence_excerpt: "HomeChef reserves the right to suspend accounts that receive repeated hygiene complaints or fail document verification."
    recommendation: "Add procedural fairness: notice of complaint, opportunity to respond, defined threshold ('repeated' = how many?), and appeal/grievance officer route. Per Consumer Protection (E-Commerce) Rules 2020 Rule 5(3), grievance officer must respond within 48 hours of complaint and resolve within 30 days. Unilateral suspension without process risks Consumer Forum reversal."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 4(5), 5(3)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-016
    surface_id: mv-onb-policies-checkbox
    lens: legal
    severity: P1
    issue: "Single checkbox 'I accept the terms and conditions' bundles entire vendor agreement — no acknowledgment of specific load-bearing clauses (commission, suspension rights, food-safety liability)"
    evidence_excerpt: "I accept the terms and conditions"
    recommendation: "Break out high-stakes clauses for separate scroll-and-acknowledge: (a) commission %, (b) suspension grounds, (c) food-safety attestation, (d) DPDP consent. Indian courts have shown willingness to strike unconscionable clauses in click-through contracts where no specific attention was drawn."
    citation: "LIC of India v. Consumer Education & Research Centre (1995); best-practice"
    depends_on: "needs lawyer review"

  # ============================================================================
  # DPDP / DATA PROTECTION
  # ============================================================================

  - finding_id: LEG-017
    surface_id: vp-legal-docs-banner-body
    lens: legal
    severity: P0
    issue: "Claim 'Your documents are encrypted and stored securely' is a content claim that requires technical backing — under DPDP §8(5), data fiduciary must implement reasonable security safeguards AND not misrepresent them"
    evidence_excerpt: "Your documents are encrypted and stored securely. They are only used for verification."
    recommendation: "Verify with engineering: (a) at-rest encryption (KMS-managed keys, algorithm), (b) in-transit (TLS 1.2+), (c) access controls (who can decrypt), (d) retention period after verification. Then either: (i) make the claim specific ('AES-256 at rest, TLS 1.3 in transit, deleted 90 days after rejection') or (ii) soften to non-claim ('stored securely per industry standards'). False security representations under DPDP §33 attract penalties up to ₹250 crore."
    citation: "DPDP Act 2023 §8(5) (security safeguards), §33 (penalty for failure of safeguards)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-018
    surface_id: vp-legal-docs-banner-body
    lens: legal
    severity: P0
    issue: "Claim 'They are only used for verification' is a purpose-limitation promise but no retention period disclosed — DPDP requires explicit retention statement"
    evidence_excerpt: "They are only used for verification."
    recommendation: "Add: 'Retained for [X] years after kitchen deactivation as required by [tax/FSSAI/anti-money-laundering law cite]; then deleted within [Y] days'. DPDP §8(7) requires deletion once purpose is satisfied unless retention is required by law. Document retention without legal-basis disclosure is a §33 violation risk."
    citation: "DPDP Act 2023 §8(7) (retention limitation); §5(1)(ii) (retention notice in consent)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-019
    surface_id: vp-legal-docs-aadhaar
    lens: legal
    severity: P0
    issue: "Aadhaar Card collected with only 'For identity verification' framing — Aadhaar is governed by Aadhaar Act 2016 and UIDAI rules, NOT just DPDP; private entities have strict limits on holding Aadhaar"
    evidence_excerpt: "Aadhaar Card / For identity verification"
    recommendation: "Per Aadhaar Act §57 (post-2018 Supreme Court ruling in K.S. Puttaswamy II), private entities cannot mandate Aadhaar for service. Use Offline e-KYC (XML/QR) instead of storing Aadhaar number directly, OR use UIDAI-approved e-KYC AUA/KUA. Storing Aadhaar number or scans without UIDAI registration may violate Aadhaar (Authentication & Offline Verification) Regulations 2021. Also mask Aadhaar in any UI showing it back."
    citation: "Aadhaar Act 2016 §57; K.S. Puttaswamy v. Union of India (2018) 1 SCC 809; Aadhaar (Authentication & Offline Verification) Regulations 2021"
    depends_on: "needs lawyer review"

  - finding_id: LEG-020
    surface_id: vp-legal-docs-aadhaar
    lens: legal
    severity: P1
    issue: "Aadhaar collection alongside PAN and bank details — no separate, prominent Aadhaar consent or 'why we need Aadhaar specifically' disclosure"
    evidence_excerpt: "Aadhaar Card / For identity verification"
    recommendation: "Aadhaar collection requires distinct, written informed consent disclosing: (a) specific purpose, (b) alternatives available, (c) consequences of refusal. Bundling Aadhaar consent with general document upload violates Aadhaar Regulation 6(2). Offer Voter ID/Passport/Driving Licence as alternatives — Aadhaar must be optional for private platforms per Puttaswamy II."
    citation: "Aadhaar (Authentication & Offline Verification) Regulations 2021 Reg. 6; K.S. Puttaswamy II (2018)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-021
    surface_id: vp-legal-docs-pan
    lens: legal
    severity: P1
    issue: "PAN Card framed as 'Required for tax purposes and payouts' but no disclosure of TDS §194-O (1% TDS on e-commerce participants from gross sales over ₹5 lakh)"
    evidence_excerpt: "PAN Card / Required for tax purposes and payouts"
    recommendation: "Disclose at collection: 'We need your PAN to deduct 1% TDS on your gross sales above ₹5 lakh per year under Income Tax Act §194-O. Form 26AS will reflect this credit.' Chefs need to understand the tax-withholding implication BEFORE they sign up, not at first payout."
    citation: "Income Tax Act 1961 §194-O (e-commerce TDS); DPDP Act 2023 §5 (Notice)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-022
    surface_id: vp-legal-docs-banner-title
    lens: legal
    severity: P1
    issue: "'Documents help us verify your kitchen and enable payouts' — does not identify the data fiduciary, grievance officer, or rights of the data principal (chef)"
    evidence_excerpt: "Documents help us verify your kitchen and enable payouts"
    recommendation: "Add DPDP §5-compliant notice at point of collection (or prominent link to one): (a) data fiduciary entity name + registered address, (b) purposes (verify kitchen, payouts, TDS, FSSAI), (c) right to access/correct/erase (§11-13), (d) grievance officer name + contact + 30-day SLA, (e) right to file complaint with Data Protection Board (§28)."
    citation: "DPDP Act 2023 §5, §11-13, §28"
    depends_on: "needs lawyer review"

  - finding_id: LEG-023
    surface_id: vp-legal-docs-identity-title
    lens: legal
    severity: P2
    issue: "'These are required to verify your identity and set up payouts' — broad single-purpose framing; doesn't clarify which doc serves which purpose"
    evidence_excerpt: "Identity Documents / These are required to verify your identity and set up payouts."
    recommendation: "Per-document purpose disclosure: PAN → tax compliance and payouts; Aadhaar → identity verification (optional, alternatives available); Cheque → bank verification for payouts; FSSAI → food-business licence display. DPDP §5 requires purpose specificity — vague global purpose is non-compliant."
    citation: "DPDP Act 2023 §5(1)(i) (purpose of processing)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-024
    surface_id: vp-legal-docs-cheque
    lens: legal
    severity: P1
    issue: "Cancelled cheque collected with no disclosure of how bank details are stored, who has access, or PCI/RBI safeguards around banking data"
    evidence_excerpt: "Cancelled Cheque / Bank Proof / For setting up direct payouts to your bank account (optional, can add later)"
    recommendation: "Add: 'Your cancelled cheque is encrypted at rest and used only to set up payouts via [Razorpay/Stripe]. We do not store your IFSC/account number in plain text outside this verification.' Banking data falls under DPDP §6(7) sensitive-personal-data-like treatment (per draft Rules) and may require explicit consent."
    citation: "DPDP Act 2023 §8(5); RBI Master Direction on Digital Payment Security Controls 2021"
    depends_on: "needs lawyer review"

  # ============================================================================
  # BANKING / PAYMENTS / RBI
  # ============================================================================

  - finding_id: LEG-025
    surface_id: vp-legal-kitchen-payout-warning
    lens: legal
    severity: P0
    issue: "'Payout integration coming soon. These details are saved locally for now' — collecting banking data without clear handling disclosure; 'saved locally' may mean unencrypted browser storage, which is a data breach waiting to happen"
    evidence_excerpt: "Payout integration coming soon. These details are saved locally for now."
    recommendation: "STOP collecting bank account details until payout integration ships. 'Saved locally' is ambiguous — if it means localStorage/IndexedDB in browser, banking data sits in plain text exposed to XSS. If it means server-side stub, it must be encrypted with KMS keys. Either way, do not collect data you cannot yet process. Per RBI Master Direction on Digital Payment Security Controls 2021, banking data requires AES-256 encryption and tokenization."
    citation: "RBI Master Direction on Digital Payment Security Controls 2021; DPDP Act 2023 §8(5); §6 (purpose limitation — can't process for a purpose that doesn't yet exist)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-026
    surface_id: vp-legal-settings-stripe
    lens: legal
    severity: P1
    issue: "Stripe Connect block routes chef KYC through Stripe's hosted onboarding but doesn't disclose data flow to chef — Stripe is a cross-border processor (USA), so DPDP §16 cross-border transfer rules apply"
    evidence_excerpt: "Stripe handles KYC and bank verification on their hosted pages - just pick your country and follow the flow."
    recommendation: "Disclose: (a) Stripe is a data processor; chef's data leaves India; (b) Stripe's privacy policy applies in addition to ours; (c) data is transferred to USA. Under DPDP §16, cross-border transfers are allowed but require notification. Add link to Stripe's privacy policy at point of redirect. Also: Stripe is not a registered Payment Aggregator in India under RBI's PA framework — verify the legal structure (is Stripe acting as PA, or merchant-of-record outside India?)."
    citation: "DPDP Act 2023 §16; RBI Payment Aggregator Master Direction (Mar 2020, updated 2024) — PA licence requirement"
    depends_on: "needs lawyer review"

  - finding_id: LEG-027
    surface_id: vp-legal-settings-stripe
    lens: legal
    severity: P1
    issue: "Gateway switch text 'Make Stripe My Primary Gateway / Switch Back to Razorpay' — no disclosure of refund-in-flight handling, pending payout impact, or settlement-cycle change"
    evidence_excerpt: "Make Stripe My Primary Gateway / Switch Back to Razorpay / Active gateway for your orders: ..."
    recommendation: "Add a consequence callout before switch: 'Switching gateways will: pause new payouts for [X] hours; existing pending settlements complete via [old gateway]; refunds for past orders process through [old gateway] for [Y] days.' RBI PA Master Direction §9 requires settlement transparency; ambiguous gateway switching during open settlement windows is a customer-protection issue."
    citation: "RBI Payment Aggregator Master Direction §9 (settlement); Consumer Protection (E-Commerce) Rules 2020 Rule 5"
    depends_on: "needs lawyer review"

  - finding_id: LEG-028
    surface_id: vp-legal-policy-tos
    lens: legal
    severity: P1
    issue: "'Payouts will be processed weekly' — RBI PA Master Direction defines specific settlement cycles (T+1 to T+3 standard); 'weekly' without anchor day or holiday handling is operationally ambiguous"
    evidence_excerpt: "my payouts will be processed weekly to my registered bank account."
    recommendation: "State the specific cycle: 'Every Monday for orders completed up to the previous Friday' or similar. Disclose: what happens on bank holidays, what cutoff time defines 'completed', and what an order failure does to the cycle. Verify the cycle is consistent with the PA's settlement obligations to merchants under the RBI Master Direction."
    citation: "RBI Payment Aggregator Master Direction §9 (settlement to merchant)"
    depends_on: "needs lawyer review"

  # ============================================================================
  # ADMIN PORTAL — DUE PROCESS / TRANSPARENCY
  # ============================================================================

  - finding_id: LEG-029
    surface_id: ap-chefs-reject-reason
    lens: legal
    severity: P0
    issue: "Hardcoded rejection reason 'Rejected by admin' sent to chef with no specific reason — fails Consumer Protection (E-Commerce) Rules 2020 Rule 4(5) requirement to communicate reasons for actions affecting users"
    evidence_excerpt: "Rejected by admin"
    recommendation: "Require admin to select a coded reason from a structured list (FSSAI docs missing, kitchen photos insufficient, identity mismatch, prior policy violations, etc.) and append a free-text justification visible to the chef. Vague rejections are: (a) ripe for discrimination claims if patterns emerge, (b) non-compliant with E-Commerce Rules 4(5)/(11), (c) impossible to appeal under §13 DPDP grievance rights for automated/admin decisions."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 4(5), 4(11); DPDP Act 2023 §13 (grievance redressal)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-030
    surface_id: ap-approvaldetail-notes-helper
    lens: legal
    severity: P2
    issue: "'These notes will be recorded in the history and visible to other admins' — does not state whether notes are also disclosable to the affected chef under DPDP §11 (right of access)"
    evidence_excerpt: "These notes will be recorded in the history and visible to other admins."
    recommendation: "Add disclosure that admin notes about a data principal (chef) may be requested by that chef under DPDP §11 right of access — unless they fall under §17 exemption (e.g., prevention of fraud). This sets correct admin expectations and prevents tortious-interference / defamation exposure from off-record comments mistakenly thought to be private."
    citation: "DPDP Act 2023 §11 (right of access); §17 (exemptions)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-031
    surface_id: ap-auditlogs-subtitle
    lens: legal
    severity: P3
    issue: "Audit log subtitle states 'IP, user agent' captured — admin's own personal data is being processed; admin DPDP notice should cover this"
    evidence_excerpt: "Admin actions recorded with actor, IP, user agent, and before/after values."
    recommendation: "Cross-check: admin onboarding / employment contract should include notice that admin activities are logged with IP/UA for tamper-evidence and breach-investigation purposes. Without notice at hire, surveillance of admins may breach DPDP §5 even where employment-context exemptions narrow under §17."
    citation: "DPDP Act 2023 §5; §17"
    depends_on: "needs lawyer review"

  - finding_id: LEG-032
    surface_id: ap-secsettings-2fa-exempt
    lens: legal
    severity: P1
    issue: "2FA-exempt email list defines a carve-out for service accounts — without policy text explaining why exemptions are granted, this is a security/compliance gap and audit-trail weakness"
    evidence_excerpt: "2FA exempt emails ... Useful for service accounts (E2E tests, automation) that can't scan a QR. Leave empty to enforce 2FA on every admin. ... service@fe3dr.com"
    recommendation: "Policy gap: write a documented exemption policy with: (a) approval authority for adding an exemption, (b) review cadence, (c) compensating controls (IP allowlist, restricted scope), (d) audit log of additions/removals. Production-hardcoded 'service@fe3dr.com' must be verified as a real, secured service account, not a backdoor. Per CERT-In Directions Apr 2022, admin account compromise is reportable in 6 hours."
    citation: "CERT-In Direction No. 20(3)/2022-CERT-In dated 28 Apr 2022; best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-033
    surface_id: ap-settings-payment-helpers
    lens: legal
    severity: P2
    issue: "'Keys are stored securely in GCP Secret Manager' — specific technical claim that creates liability if false; also discloses internal infrastructure choice (GCP) which some security policies flag"
    evidence_excerpt: "Keys are stored securely in GCP Secret Manager. Leave a field empty to keep its current value."
    recommendation: "Verify the claim is true (engineering confirmation). If true, leave as-is (it's accurate and reassures admins). If GCP Secret Manager use is partial/migrating, soften to 'stored encrypted in our secrets manager'. Disclosing infrastructure to internal admins is fine; but copy this claim does not leak to chefs/customers."
    citation: "best-practice (truthful claims)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-034
    surface_id: ap-secsettings-2fa-enforce
    lens: legal
    severity: P3
    issue: "2FA enforcement toggle — no record of when/by whom the policy was last changed visible in the helper text"
    evidence_excerpt: "Require 2FA for all admins; When enabled, admins without 2FA are prompted to enroll on their next login."
    recommendation: "Cross-reference with audit log: any change to authentication policy should be recorded in audit log with actor, timestamp, before/after state — and ideally surfaced as last-changed banner here. CERT-In Directions require certain security events to be retained 180 days."
    citation: "CERT-In Direction 28 Apr 2022 (log retention); best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-035
    surface_id: ap-secsettings-apikey-warn
    lens: legal
    severity: P3
    issue: "'Copy this key now — you won't see it again' — operationally correct but doesn't disclose that the key, once leaked, identifies admin actions in audit logs"
    evidence_excerpt: "Copy this key now — you won't see it again"
    recommendation: "Add follow-up sentence: 'If you lose this key, regenerate it — don't share or store it in source code or screenshots. Compromised keys must be reported to security within 24 hours.' This aligns admin behaviour with CERT-In incident reporting (6-hour rule for major incidents)."
    citation: "CERT-In Direction 28 Apr 2022 §2(ii) (6-hour reporting); best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-036
    surface_id: ap-staffdetail-role-descriptions
    lens: legal
    severity: P2
    issue: "Role descriptions are one-line summaries ('Full access to everything', 'Read-only support access') — these are RBAC labels that drive admin permissions; vague labels make access-control review harder for an auditor or DPDP investigation"
    evidence_excerpt: "Full access to everything; Admin portal management; Delivery fleet management; Delivery operations; Read-only support access"
    recommendation: "Map each role to a documented permissions matrix (what it can read/write/delete, with examples). Per DPDP §8(4), data fiduciary must implement appropriate technical and organisational measures — vague RBAC labels undercut that. 'Full access to everything' especially needs explicit scoping for principle of least privilege."
    citation: "DPDP Act 2023 §8(4); ISO/IEC 27001 A.9 (Access Control) — best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-037
    surface_id: ap-auth-restricted-body
    lens: legal
    severity: P3
    issue: "'New registrations are not allowed' — informational but combined with 'Restricted Access' could be perceived as discouraging legitimate access requests from authorised staff who haven't been onboarded"
    evidence_excerpt: "Only pre-authorized administrators can sign in. New registrations are not allowed."
    recommendation: "Add contact for legitimate access requests: 'If you need admin access, contact your manager or security@fe3dr.com.' Clarifies the gate is procedural, not adversarial, and provides legitimate-purpose channel."
    citation: "best-practice"
    depends_on: "needs lawyer review"

  # ============================================================================
  # ACCOUNT DELETION / DPDP RIGHTS
  # ============================================================================

  - finding_id: LEG-038
    surface_id: vp-legal-settings-danger
    lens: legal
    severity: P0
    issue: "'Account deactivation is not available in demo mode' — production-shipped string referencing 'demo mode'; either (a) demo branch leaked to prod, OR (b) deactivation is disabled in prod under demo label, blocking DPDP §12 right to erasure"
    evidence_excerpt: "Account deactivation is not available in demo mode"
    recommendation: "URGENT: verify this is not blocking real account deletions. DPDP §12(1) gives data principal explicit right to erasure of personal data. Demo-mode strings in production are a P0 trust signal + a regulatory time-bomb. Remove the demo-mode gate, implement real deactivation, or clearly state deactivation is unavailable for legal reasons (e.g., open tax obligations) and provide alternative via grievance officer."
    citation: "DPDP Act 2023 §12 (right to correction and erasure); §13 (grievance)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-039
    surface_id: vp-legal-settings-danger
    lens: legal
    severity: P1
    issue: "'Deactivating your account will hide your kitchen from customers and pause all orders' — 'deactivate' is not 'delete'; doesn't disclose data retention after deactivation"
    evidence_excerpt: "Deactivating your account will hide your kitchen from customers and pause all orders. Deactivate Account"
    recommendation: "Distinguish: 'Deactivate' (hide profile, pause orders, data retained for X reason) vs 'Delete account' (erasure under DPDP §12). Disclose retention period and legal basis for each retention category (e.g., tax records 8 years, FSSAI complaint records X years). Currently 'deactivate' obscures whether data is actually erased."
    citation: "DPDP Act 2023 §8(7), §12"
    depends_on: "needs lawyer review"

  - finding_id: LEG-040
    surface_id: mv-settings-delete-confirm-body
    lens: legal
    severity: P0
    issue: "'To delete your account, please contact our support team' — under DPDP §12, deletion request must be actionable by data principal directly or via a clearly-defined process; routing through an unidentified support team without SLA fails the right-to-erasure standard"
    evidence_excerpt: "To delete your account, please contact our support team. This action cannot be undone."
    recommendation: "Implement self-service deletion OR provide: (a) grievance officer name + email per DPDP §10(2), (b) 30-day response SLA per §13(1), (c) escalation route to Data Protection Board. 'Contact support' without SLA is not a compliant erasure pathway."
    citation: "DPDP Act 2023 §10(2), §12, §13"
    depends_on: "needs lawyer review"

  - finding_id: LEG-041
    surface_id: mv-settings-delete-contact-email
    lens: legal
    severity: P1
    issue: "Support email 'support@homechef.app' — inconsistent with brand 'Fe3dr' (see LEG-046), AND inventory notes a different domain 'support@homechef.in' is referenced in mobile-delivery — two different support emails undermines grievance pathway"
    evidence_excerpt: "Please email support@homechef.app to delete your account."
    recommendation: "Pick one canonical grievance email and use it everywhere. DPDP §10(2) requires data fiduciary to publish a single grievance officer contact. Multiple emails create plausible-deniability risk where one mailbox is unmonitored. Verify domain ownership: homechef.app and homechef.in must both be owned by the same legal entity, or split clearly which serves which jurisdiction."
    citation: "DPDP Act 2023 §10(2); best-practice (grievance pathway clarity)"
    depends_on: "needs lawyer review"

  # ============================================================================
  # MARKETING / EMAIL / BRAND CONSISTENCY
  # ============================================================================

  - finding_id: LEG-042
    surface_id: api-email-base-footer
    lens: legal
    severity: P1
    issue: "Email footer says 'Fe3dr by HomeChef' — brand identity inconsistency across product (Fe3dr vs HomeChef) creates legal entity ambiguity, especially for receipt/notice/grievance emails that must clearly identify the data fiduciary"
    evidence_excerpt: "Fe3dr by HomeChef · Authentic homemade food, delivered · fe3dr.com · Privacy · Terms"
    recommendation: "Pick one. If Fe3dr is a brand of HomeChef Pvt Ltd (or similar), state in footer: 'Fe3dr is a service of HomeChef Pvt Ltd, [registered address], CIN: [X]'. Email-marketing rules (TRAI commercial communications regs) and DPDP §5 require clear data fiduciary identification. Brand ambiguity also makes it hard to know which entity is the merchant of record for refunds (RBI PA framework)."
    citation: "DPDP Act 2023 §5; TRAI Telecom Commercial Communications Customer Preference Regulations 2018; Companies Act 2013 §12(3) (display of company name)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-043
    surface_id: api-email-base-footer
    lens: legal
    severity: P1
    issue: "Footer links to '/privacy' and '/terms' hardcoded — same risk as LEG-001 (pages must exist) plus no unsubscribe / preferences link for marketing-class emails"
    evidence_excerpt: "fe3dr.com · Privacy · Terms"
    recommendation: "For transactional emails: include privacy + terms + grievance officer contact. For marketing emails: also include unsubscribe link (one-click), preference centre, and explicit identification of sender. Mixing classes (transactional + marketing) without separate footers violates DLT/TRAI norms for SMS/email commercial communications."
    citation: "TRAI Telecom Commercial Communications Customer Preference Regulations 2018; DPDP Act 2023 §6 (withdrawal of consent must be as easy as giving it)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-044
    surface_id: web-mkt-brand-copyright
    lens: legal
    severity: P2
    issue: "Copyright © {year} Fe3dr — no entity name beyond brand; for footer copyright to function as a legal notice, full company name should appear"
    evidence_excerpt: "© {year} Fe3dr. All rights reserved."
    recommendation: "Replace with full legal entity: '© {year} HomeChef Pvt Ltd. All rights reserved. Fe3dr is a registered trademark of HomeChef Pvt Ltd.' Companies Act 2013 §12(3) requires registered company name on all business communications. 'All rights reserved' is largely vestigial post-Berne Convention but harmless."
    citation: "Companies Act 2013 §12(3); Trade Marks Act 1999 §29"
    depends_on: "needs lawyer review"

  - finding_id: LEG-045
    surface_id: vp-legal-login-footer
    lens: legal
    severity: P1
    issue: "Sign-in screen footer 'By continuing, you agree to Fe3dr's Terms of Service and Privacy Policy' — but no link target verified; chef logs in routinely and is bound each time without notice of changes"
    evidence_excerpt: "By continuing, you agree to Fe3dr's Terms of Service and Privacy Policy"
    recommendation: "Verify links resolve. Add version/date stamp ('last updated: DD Mon YYYY') so chefs can identify if terms changed since their last accepted version. Major term changes (commission %, payout cycle, data-handling) require fresh consent per DPDP §6, not click-wrap acceptance via login."
    citation: "DPDP Act 2023 §6 (informed consent on changes); Indian Contract Act 1872 §10"
    depends_on: "needs lawyer review"

  - finding_id: LEG-046
    surface_id: vp-legal-register-footer
    lens: legal
    severity: P0
    issue: "'You agree to Fe3dr's Vendor Terms' — but inventory notes 'no link target' for Vendor Terms; chef is being asked to accept a document they cannot access during registration"
    evidence_excerpt: "By registering, you agree to Fe3dr's Vendor Terms and Privacy Policy"
    recommendation: "Wire the link to actual Vendor Terms (currently only mobile-vendor app references 'homechef.in/vendor-terms' — verify same URL works here). Without a readable contract at the moment of acceptance, the agreement may be voidable. Vendor Terms is the highest-stakes contract on the platform (commission, suspension, food-safety liability) — must be accessible."
    citation: "Indian Contract Act 1872 §10; Consumer Protection Act 2019 §2(11) (unfair contract)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-047
    surface_id: ap-auth-footer-legal
    lens: legal
    severity: P3
    issue: "'Fe3dr Administration Portal · Internal Use Only' — clarifies portal scope but doesn't reference confidentiality obligations on admins"
    evidence_excerpt: "Fe3dr Administration Portal · Internal Use Only"
    recommendation: "Admins handling chef/customer PII should see a confidentiality notice at login: 'Information accessed via this portal is confidential and protected under the DPDP Act 2023. Unauthorised access, disclosure, or use may result in disciplinary action and legal penalties.' Sets expectations and supports later enforcement."
    citation: "DPDP Act 2023 §33 (penalties); best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-048
    surface_id: mv-onb-policies-cancel-label
    lens: legal
    severity: P3
    issue: "'Cancellation Policy *' — asterisk on label conflicts with style guide which says required indicator should be on field, not label; also a contractual term being framed as a form field rather than a clause"
    evidence_excerpt: "Cancellation Policy *"
    recommendation: "Cancellation policy is a binding term, not a preference. Reframe as 'Select your cancellation policy — this becomes part of your kitchen's customer commitment.' Plus full preview of consequences for customers (what they'll see at checkout) so chef makes an informed selection."
    citation: "best-practice (informed contracting); Style guide §4 (Form labels)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-049
    surface_id: mv-onb-policies-cancel-option-no
    lens: legal
    severity: P1
    issue: "Cancellation option 'No cancellations after order accepted' — may conflict with Consumer Protection (E-Commerce) Rules 2020 which require reasonable cancellation windows for consumers"
    evidence_excerpt: "No cancellations after order accepted"
    recommendation: "Under E-Commerce Rules Rule 5(7), seller must accept cancellation 'before shipping/delivery'. For perishable home-cooked food, 'before prep starts' is the analogous moment. A blanket 'no cancellations after order accepted' is likely unenforceable against a consumer making a reasonable, timely cancellation. Frame as 'No refunds after prep starts; cancellations free until chef begins cooking.'"
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 5(7)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-050
    surface_id: mv-onb-policies-cancel-option-1h
    lens: legal
    severity: P2
    issue: "'Up to 1 hour before prep start' — consumer-facing implication of this option not surfaced; chef may pick without realising customer-side impact"
    evidence_excerpt: "Up to 1 hour before prep start"
    recommendation: "Show alongside each option a customer-side preview: 'Customers will see: \"Cancel free up to 1 hour before the chef starts cooking. After that, refunds may not apply.\"' Per Consumer Protection (E-Commerce) Rules 2020 Rule 5(3) and DPDP §5 transparency, customers need to know cancellation terms BEFORE paying."
    citation: "Consumer Protection (E-Commerce) Rules 2020 Rule 5; best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-051
    surface_id: mv-onb-policies-cancel-option-30m
    lens: legal
    severity: P2
    issue: "'Up to 30 mins before prep start' — same as LEG-050, no consumer-side preview, plus a very short refund window for digital food orders"
    evidence_excerpt: "Up to 30 mins before prep start"
    recommendation: "Verify with counsel whether 30-minute pre-prep cutoff meets the Consumer Protection Act 'reasonable opportunity to cancel' standard. Add customer-side preview and ensure refund timeline (≤7 days per RBI PA Master Direction) is stated separately."
    citation: "Consumer Protection Act 2019; RBI Payment Aggregator Master Direction (refund timelines)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-052
    surface_id: mv-settings-delete-contact-action
    lens: legal
    severity: P3
    issue: "'Contact Support' as the only action for account deletion — button text doesn't reflect the legal weight of the request (DPDP §12 right of erasure)"
    evidence_excerpt: "Contact Support"
    recommendation: "Label more precisely: 'Request account deletion' (verb-first per style guide). Sets correct user expectation that this is a DPDP §12 request with 30-day SLA, not a casual support ticket."
    citation: "DPDP Act 2023 §12; Style guide §4 (Buttons)"
    depends_on: "needs lawyer review"

  # ============================================================================
  # CROSS-CUTTING — PROHIBITED / MISSING DISCLOSURES (jurisdiction-agnostic)
  # ============================================================================

  - finding_id: LEG-053
    surface_id: api-email-base-footer
    lens: legal
    severity: P2
    issue: "No allergen disclaimer in email order confirmations — FSSAI requires allergen information to travel with the product to the consumer"
    evidence_excerpt: "Fe3dr by HomeChef · Authentic homemade food, delivered · fe3dr.com · Privacy · Terms"
    recommendation: "Add to order-confirmation email template (separate from footer): 'Allergens: [list from menu item]. If you have allergies, contact the chef before delivery: [chef contact]. We aren't responsible for allergic reactions unless we hid this information.' FSS (Packaging and Labelling) Regulations 2011 Reg. 2.4.5 mandates allergen disclosure."
    citation: "FSS (Packaging and Labelling) Regulations 2011 Reg. 2.4.5; FSS (Labelling and Display) Regulations 2020"
    depends_on: "needs lawyer review"

  - finding_id: LEG-054
    surface_id: vp-legal-docs-pan
    lens: legal
    severity: P3
    issue: "PAN placeholder 'ABCDE1234F' — using a realistic-looking PAN as placeholder could be confused with a real one; better to use clearly fake format like 'AAAPL1234C' (UIDAI/Income-Tax docs use this convention)"
    evidence_excerpt: "PAN Number / ABCDE1234F / 10-character alphanumeric PAN"
    recommendation: "Cosmetic but worth checking: ensure 'ABCDE1234F' is not actually a registered PAN (PAN structure: 5 letters + 4 digits + 1 letter, fourth letter is entity type). Use government-published example format. Low-risk but tidy."
    citation: "best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-055
    surface_id: vp-legal-policies-agreements-title
    lens: legal
    severity: P2
    issue: "'Please read and accept each policy to complete your registration' — implies user will read each policy, but inventory shows policies are presented as short paragraphs followed by checkboxes; no scroll-to-bottom or time-spent gating"
    evidence_excerpt: "Agreements & Policies / Please read and accept each policy to complete your registration."
    recommendation: "For high-stakes policies (commission, suspension, food-safety attestation), require scroll-to-bottom before checkbox enables. Indian courts have looked favourably on platforms that demonstrably ensured the user saw the term. Pure click-wrap with no engagement evidence is weaker."
    citation: "best-practice (click-wrap enforceability); LIC v. CERC (1995) — unconscionable terms"
    depends_on: "needs lawyer review"

  - finding_id: LEG-056
    surface_id: vp-legal-policy-tos
    lens: legal
    severity: P2
    issue: "Vendor TOS uses 'I' first-person ('I have read and agree') — style-guide says legal pages should use 'we' / 'you'; first-person attestation is ok for the act of attesting but should resolve into 'you' terms in the body"
    evidence_excerpt: "I have read and agree..."
    recommendation: "'I' is acceptable as the attestation verb; the underlying policy text should be in 'we'/'you' form per style guide §5. Verify the linked-full-policy uses 'we'/'you'. Mixing 'the Platform', 'the Company', 'the Vendor' across documents is a drift signal."
    citation: "Style guide §5 (Legal-page tone); best-practice"
    depends_on: "needs lawyer review"

  - finding_id: LEG-057
    surface_id: web-legal-checkout-terms-note
    lens: legal
    severity: P1
    issue: "Missing GST disclosure at checkout — under CGST Act and standard e-commerce practice, customers must see tax breakup before payment, not only on invoice"
    evidence_excerpt: "By placing this order, you agree to our Terms of Service"
    recommendation: "Verify checkout shows: subtotal, delivery, GST line (with rate, e.g., '5% GST on food'), total. Inclusive vs exclusive must be unambiguous. Under §31 CGST Act, tax invoice is mandatory; under E-Commerce Rules 4(3), price breakup including taxes must be shown. Missing GST disclosure pre-payment is an audit/refund liability."
    citation: "CGST Act 2017 §31; Consumer Protection (E-Commerce) Rules 2020 Rule 4(3)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-058
    surface_id: vp-legal-policies-compliance-items
    lens: legal
    severity: P1
    issue: "'label allergens and ingredients' is item #7 in compliance checklist but no enforcement mechanism shown — chef attests once at onboarding, not per dish; FSSAI requires per-product disclosure"
    evidence_excerpt: "label allergens and ingredients"
    recommendation: "Onboarding-time attestation is not sufficient. Each menu item must carry allergen + ingredient information. Make the chef-side menu form enforce this (required field), and surface allergens prominently to customers on item pages and at checkout. Per FSS (Labelling and Display) Regulations 2020, allergens must be highlighted (bold, italics, or different colour)."
    citation: "FSS (Labelling and Display) Regulations 2020 Reg. 5(7)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-059
    surface_id: vp-legal-policy-hygiene
    lens: legal
    severity: P2
    issue: "'I commit to maintaining a clean and hygienic kitchen at all times' — open-ended commitment with no measurable bar; combined with 'periodic kitchen checks' creates a one-sided contract where platform defines the standard at audit time"
    evidence_excerpt: "I commit to maintaining a clean and hygienic kitchen at all times..."
    recommendation: "Reference an objective standard (FSSAI Schedule 4 home-based food business hygiene norms) so both chef and platform know the bar. Open-ended hygiene clauses are vulnerable to challenge and don't actually protect platform if a hygiene incident triggers litigation."
    citation: "FSS (Licensing & Registration) Regulations 2011 Schedule 4 Part V"
    depends_on: "needs lawyer review"

  - finding_id: LEG-060
    surface_id: vp-legal-policy-cancellation
    lens: legal
    severity: P2
    issue: "'Accept or reject orders within 5 minutes' — chef-side SLA; platform doesn't disclose its own SLA to chefs (e.g., dispute response time, payout settlement time)"
    evidence_excerpt: "accept or reject orders within 5 minutes / prepare within estimated time"
    recommendation: "Contract reciprocity: if chef has 5-minute order-accept SLA, platform should commit to corresponding SLAs (dispute response, payout, support). One-sided SLAs are a hallmark of unfair contracts under Consumer Protection Act §2(46) and may be unenforceable."
    citation: "Consumer Protection Act 2019 §2(46) (unfair contract); best-practice (contract reciprocity)"
    depends_on: "needs lawyer review"
```

## Business Analyst findings

```yaml
findings:

  - finding_id: BA-001
    surface_id: vp-legal-kitchen-payout-warning
    lens: business-analyst
    severity: P0
    issue: '"Saved locally" on payout bank-detail form is a trust-destroying production leak'
    evidence_excerpt: "Payout integration coming soon. These details are saved locally for now."
    recommendation: >
      Remove the warning entirely until payout integration is live. If collection must precede
      integration, replace with: "Your bank details are stored securely. Payouts go live once your
      kitchen is approved — we'll notify you when your first payout is scheduled." This reframes
      collection as onboarding progress, not a dead-end. Expected impact: eliminate the drop-off
      spike at KitchenSetupPage for chefs who reach payout section; estimated 15-25% reduction
      in abandonment at that step.
    metric_hypothesis: "chef onboarding completion (KitchenSetupPage → submission)"
    depends_on: null

  - finding_id: BA-002
    surface_id: vp-legal-settings-danger
    lens: business-analyst
    severity: P0
    issue: '"Demo mode" toast on Deactivate Account button leaks placeholder copy to production'
    evidence_excerpt: "onClick={() => toast.error('Account deactivation is not available in demo mode')}"
    recommendation: >
      This is a shipped placeholder. Either wire real deactivation logic or replace the toast
      with: "To deactivate your kitchen, contact support@homechef.app — we'll pause your orders
      and hide your profile within 1 hour." Do not surface "demo mode" language to any real chef.
      Trust catastrophe: a chef who sees "demo mode" on their live account concludes the platform
      is not production-ready and churns immediately.
    metric_hypothesis: "chef trust score; chef D30 retention"
    depends_on: null

  - finding_id: BA-003
    surface_id: mv-onb-policies-terms-body
    lens: business-analyst
    severity: P0
    issue: 'Mobile-vendor terms use brand name "HomeChef" while web onboarding uses "Fe3dr" — same legal document, two different brands'
    evidence_excerpt: "By joining HomeChef as a vendor you agree to... HomeChef reserves the right to suspend accounts... Full terms available at homechef.in/vendor-terms."
    recommendation: >
      Resolve brand identity before next onboarding release. Pick one name and apply it
      consistently across constants/terms.ts, StepPolicies.tsx, all email templates, and all
      legal footers. The URL homechef.in/vendor-terms does not match the vendor-portal TOS
      referenced on the web (Fe3dr Platform Terms of Service). A chef who reads the mobile terms
      (HomeChef) and the web terms (Fe3dr) will believe they are agreeing to different contracts.
      This is both a trust failure and a legal exposure.
    metric_hypothesis: "chef onboarding completion (mobile); chef trust score"
    depends_on: null

  - finding_id: BA-004
    surface_id: api-email-base-footer
    lens: business-analyst
    severity: P1
    issue: 'Email footer uses hybrid brand "Fe3dr by HomeChef" — customers receive an identity-confused legal footer on every transactional email'
    evidence_excerpt: "Fe3dr by HomeChef · Authentic homemade food, delivered · fe3dr.com · Privacy · Terms"
    recommendation: >
      Unify to single brand name matching the registered legal entity. The footer is present on
      every transactional email (order confirmation, verification, payout, chef approval) — this
      dual-brand attribution creates confusion about who the customer has contracted with.
      Suggested: "HomeChef · fe3dr.com · Privacy · Terms" OR "Fe3dr · homechef.in · Privacy ·
      Terms" — choose one, update email_templates.go emailBase function. Expected impact: improved
      open-and-click trust on transactional emails; reduced support queries about "who is Fe3dr".
    metric_hypothesis: "transactional email trust; customer support contact rate"
    depends_on: null

  - finding_id: BA-005
    surface_id: vp-legal-docs-aadhaar
    lens: business-analyst
    severity: P1
    issue: 'Aadhaar collection described only as "For identity verification" — no trust framing causes drop-off on India most-sensitive government ID'
    evidence_excerpt: "Aadhaar Card / For identity verification"
    recommendation: >
      Add a trust-framing helper line beneath the label: "Required by UIDAI regulations for
      home-based food businesses. Your Aadhaar number is masked after verification and never
      shared with customers." Two sentences that explain the legal basis (reduces anxiety) and
      the data-minimisation commitment (rebuilds trust). Aadhaar is the single highest-friction
      document in Indian consumer KYC; without framing it is the most common abandonment trigger
      in the StepDocuments flow.
    metric_hypothesis: "chef onboarding completion (StepDocuments step); KYC drop-off rate"
    depends_on: null

  - finding_id: BA-006
    surface_id: vp-legal-profile-doc-types
    lens: business-analyst
    severity: P1
    issue: 'FSSAI License labelled "Required" on ProfilePage but "Optional" during onboarding — contradictory requirement creates chef confusion and support burden'
    evidence_excerpt: >
      ProfilePage DOCUMENT_TYPES: { type: 'fssai_license', label: 'FSSAI License', required: true }
      vs StepDocuments OPTIONAL_DOCS: { type: 'fssai_license', label: 'FSSAI License',
      description: 'If you have one — gives your profile a verified badge. You can add this later.', required: false }
    recommendation: >
      Align required status to one source of truth. If FSSAI is legally optional (many home
      chefs operate on Basic Registration below ₹12L turnover), mark it optional everywhere with
      a badge upsell: "Add FSSAI for a Verified Kitchen badge." If it is required, update
      onboarding to gate submission on it. The current contradiction causes chefs who sailed
      through onboarding to hit a "Required" blocker on ProfilePage and call support.
    metric_hypothesis: "chef profile completion rate; support ticket volume (FSSAI-related)"
    depends_on: null

  - finding_id: BA-007
    surface_id: web-legal-checkout-terms-note
    lens: business-analyst
    severity: P1
    issue: 'Checkout terms acknowledgment has no hyperlink to Terms of Service — legal consent is uncollectable and creates friction without the reassurance of transparency'
    evidence_excerpt: "By placing this order, you agree to our Terms of Service"
    recommendation: >
      Link "Terms of Service" to /terms. Without the link, the acknowledgment creates friction
      (implies a contract) but delivers no trust benefit (customer cannot read what they're
      agreeing to). A clickable link shows transparency and is standard checkout practice.
      Suggested text: "By placing this order, you agree to our Terms of Service and our
      Cancellation Policy." Adding the Cancellation Policy link at checkout sets expectations
      that reduce post-order refund disputes.
    metric_hypothesis: "checkout completion rate; post-order dispute rate"
    depends_on: null

  - finding_id: BA-008
    surface_id: vp-legal-policy-tos
    lens: business-analyst
    severity: P1
    issue: 'TOS acceptance checkbox embeds commission disclosure as inline prose with no link to full terms — chef cannot verify the commission rate before agreeing'
    evidence_excerpt: "I have read and agree to the Fe3dr Terms of Service, Privacy Policy, and Vendor Agreement. I understand that Fe3dr charges a platform commission on each order and that my payouts will be processed weekly to my registered bank account."
    recommendation: >
      The title "Fe3dr Platform Terms of Service" must link to the actual TOS document. The
      phrase "a platform commission" is dangerously vague — chefs will later dispute the actual
      percentage. Replace with the concrete rate: "Fe3dr charges a X% commission on each order"
      or link to a Pricing page. A chef who discovers a different commission rate post-signup
      churns and generates negative word-of-mouth. Expected impact: reduce chef churn attributed
      to pricing surprise; reduce commission dispute support tickets.
    metric_hypothesis: "chef D30 retention; chef support ticket volume (commission-related)"
    depends_on: null

  - finding_id: BA-009
    surface_id: vp-legal-policy-cancellation
    lens: business-analyst
    severity: P1
    issue: 'Cancellation policy agreement mentions rating/visibility penalties as consequence but without specificity — vague penalty language both frightens and misleads chefs'
    evidence_excerpt: "Repeated late cancellations may affect my kitchen rating and visibility."
    recommendation: >
      Replace vague consequence copy with a defined threshold: "Three or more cancellations in
      a week reduces your search ranking. Five or more in a week pauses new orders for 24 hours."
      Specific thresholds convert a threatening clause into a navigable rule. A chef who knows
      the threshold can manage their kitchen; a chef who reads "may affect" interprets it as
      arbitrary punishment and reduces order acceptance rates pre-emptively.
    metric_hypothesis: "chef order acceptance rate; chef D7 activation"
    depends_on: null

  - finding_id: BA-010
    surface_id: vp-legal-docs-banner-body
    lens: business-analyst
    severity: P1
    issue: '"Encrypted and stored securely" is an unqualified trust claim that cannot be substantiated without naming the standard — if false, it is a compliance liability; if true, it is weak without specificity'
    evidence_excerpt: "Your documents are encrypted and stored securely. They are only used for verification."
    recommendation: >
      Replace with a commitment that is both true and trust-building: "Documents are encrypted
      at rest in Google Cloud Storage. Only our verification team can access them. We delete
      uploaded images within 30 days of verification." If 30-day deletion is not the policy,
      state the actual retention period. Vague encryption claims score poorly in user trust
      research; specific, named storage with a retention timeline converts significantly better
      for KYC step completion.
    metric_hypothesis: "chef onboarding completion (StepDocuments); KYC conversion rate"
    depends_on: null

  - finding_id: BA-011
    surface_id: ap-chefs-reject-reason
    lens: business-analyst
    severity: P1
    issue: 'Hardcoded rejection reason "Rejected by admin" is sent to chef via API — vague rejection with no actionable reason causes chef re-apply abandonment'
    evidence_excerpt: "mutationFn: (id: string) => apiClient.put(`/admin/chefs/${id}/reject`, { reason: 'Rejected by admin' })"
    recommendation: >
      The rejection payload must be human-authored per case. Replace the hardcoded string with
      an admin-entered reason field (modal with a free-text input, required before reject button
      activates). The reason text should be surfaced to the chef in their rejection notification
      email. A chef who receives "Rejected by admin" has no path to re-apply; a chef who
      receives "Your Aadhaar scan was illegible — please re-upload a clear photo" reapplies
      within 48 hours. Impact: re-apply conversion rate; chef acquisition funnel recovery.
    metric_hypothesis: "rejected-chef re-apply rate; chef acquisition funnel"
    depends_on: null

  - finding_id: BA-012
    surface_id: mv-settings-delete-contact-email
    lens: business-analyst
    severity: P1
    issue: 'Account deletion flow sends mobile chef to email support@homechef.app — domain does not match the Fe3dr brand on web portal, and friction makes GDPR/DPDPA right-to-erasure requests adversarial'
    evidence_excerpt: "Please email support@homechef.app to delete your account."
    recommendation: >
      Resolve the domain inconsistency (homechef.app vs fe3dr.com). More importantly, replace
      the email-to-delete pattern with a self-service in-app deletion request that is queued
      for admin action. Under India's Digital Personal Data Protection Act 2023, data principals
      have a right to erasure — routing this through an unmonitored email creates compliance
      risk. From a conversion standpoint, a chef who cannot easily exit will distrust the
      platform when joining. Suggested UX: "Request account deletion" button → confirmation
      → "We'll delete your account within 30 days and email confirmation to [email]."
    metric_hypothesis: "chef trust score; DPDPA compliance risk; support ticket volume"
    depends_on: null

  - finding_id: BA-013
    surface_id: vp-legal-register-footer
    lens: business-analyst
    severity: P2
    issue: '"Vendor Terms" linked in chef registration footer points to no destination — dead link at the conversion gate undermines legal consent and trust'
    evidence_excerpt: "By registering, you agree to Fe3dr's Vendor Terms and Privacy Policy"
    recommendation: >
      Wire the "Vendor Terms" text to the actual vendor terms URL (same as what is referenced
      in the TOS acceptance checkbox on StepPolicies). If the terms document does not yet exist
      at a public URL, temporarily link to the inline terms shown in the onboarding wizard.
      A dead or missing link on the registration legal footer erodes trust and renders the
      consent collection legally questionable.
    metric_hypothesis: "chef signup completion rate; legal consent validity"
    depends_on: null

  - finding_id: BA-014
    surface_id: vp-legal-policies-compliance-confirm
    lens: business-analyst
    severity: P2
    issue: 'Kitchen Compliance Checklist confirmation is a silent passive attestation — no affirmative action is required from the chef before proceeding'
    evidence_excerpt: "By proceeding, you confirm that all the above conditions are met in your kitchen."
    recommendation: >
      Add an explicit checkbox with label "I confirm my kitchen meets all 7 requirements above."
      The current passive confirmation (merely scrolling past triggers it) is not a meaningful
      affirmative consent for food-safety self-attestation. From a BA lens: without the
      checkbox, chefs who skip reading the checklist proceed anyway, then fail kitchen inspections,
      leading to suspension and negative reviews. An explicit gate adds 3 seconds of friction
      but increases compliance quality and reduces suspension rates.
    metric_hypothesis: "chef kitchen inspection pass rate; chef suspension rate"
    depends_on: null

  - finding_id: BA-015
    surface_id: vp-legal-settings-stripe
    lens: business-analyst
    severity: P2
    issue: '"Make Stripe My Primary Gateway" and "Switch Back to Razorpay" are high-stakes financial switches with no consequence explanation — a chef who taps the wrong button cannot predict what happens to in-flight orders'
    evidence_excerpt: "Make Stripe My Primary Gateway / Switch Back to Razorpay / Active gateway for your orders: ..."
    recommendation: >
      Add a one-line consequence note beneath each switch button: "Switching to Stripe will
      process all new orders through Stripe. Orders already placed will complete via Razorpay."
      Without this, a chef mid-service who accidentally switches gateways risks payment
      disruption. This is a trust and financial safety issue that can generate high-severity
      support escalations. A modal confirmation with the consequence displayed before the action
      is committed would eliminate the risk.
    metric_hypothesis: "chef payment incident rate; chef support escalation volume (gateway-related)"
    depends_on: null

  - finding_id: BA-016
    surface_id: md-leg-001
    lens: business-analyst
    severity: P2
    issue: 'Driver onboarding Terms of Service and Privacy Policy are styled as links but have no navigation wired — tapping them does nothing, which invalidates the consent gate'
    evidence_excerpt: >
      <Text className="text-herb font-medium">Terms of Service</Text> and
      <Text className="text-herb font-medium">Privacy Policy</Text>
      (no onPress handler, no Linking.openURL)
    recommendation: >
      Wire both terms links to Linking.openURL pointing to the canonical terms URLs
      (fe3dr.com/terms and fe3dr.com/privacy, or homechef.in equivalents — resolve brand
      first per BA-003). A driver who taps "Terms of Service" and nothing happens will either
      distrust the app or assume consent was not real. This is also a legal risk: consent
      collected through a non-functional link is uncollectable consent.
    metric_hypothesis: "driver onboarding completion; driver trust score"
    depends_on: BA-003

  - finding_id: BA-017
    surface_id: web-mkt-brand-copyright
    lens: business-analyst
    severity: P2
    issue: 'Web footer copyright reads "Fe3dr" while email footer reads "Fe3dr by HomeChef" — inconsistent legal entity attribution across a single user session'
    evidence_excerpt: "© {year} Fe3dr. All rights reserved."
    recommendation: >
      The copyright entity must match the registered legal name exactly — neither "Fe3dr" nor
      "Fe3dr by HomeChef" is necessarily correct until confirmed against company registration.
      Resolve the legal entity name and apply consistently across: LogoFooter (web), email
      footer (API), vendor-portal legal footers (vp-legal-login-footer, vp-legal-register-footer),
      admin-portal footer (ap-auth-footer-legal), and mobile-vendor terms (mv-onb-policies-terms-body).
      A customer who sees different entity names in the same week loses trust in the platform's
      legitimacy.
    metric_hypothesis: "customer trust score; brand clarity"
    depends_on: null

  - finding_id: BA-018
    surface_id: vp-legal-login-footer
    lens: business-analyst
    severity: P2
    issue: 'Chef portal login-page legal footer has no clickable links — terms and privacy mentioned but not accessible at the exact moment chef is committing to the platform'
    evidence_excerpt: "By continuing, you agree to Fe3dr's Terms of Service and Privacy Policy"
    recommendation: >
      Link "Terms of Service" and "Privacy Policy" to their respective URLs. Login is a
      re-consent moment — a returning chef who has a concern about terms should be able to
      access them. The absence of links on a consent statement is a common reason for
      legal challenges to collected consent. From a conversion standpoint, a chef who cannot
      verify the terms is more likely to abandon login if they have heard negative things
      about the platform.
    metric_hypothesis: "chef login completion rate; legal consent validity"
    depends_on: null

  - finding_id: BA-019
    surface_id: ap-auth-restricted-heading
    lens: business-analyst
    severity: P3
    issue: '"Restricted Access" heading on admin login is security-framing language that creates anxiety for legitimate admins who mistype their password'
    evidence_excerpt: "Restricted Access / Only pre-authorized administrators can sign in. New registrations are not allowed."
    recommendation: >
      Reframe without reducing security signal: "Admin sign-in / Access is limited to authorised
      team members." This communicates the same access restriction without the threat tone.
      Per style guide: Admin persona is "neutral operator" not "warned intruder." Current copy
      makes new admins on their first login feel they may be doing something wrong.
    metric_hypothesis: "admin onboarding friction; admin first-login completion"
    depends_on: null

  - finding_id: BA-020
    surface_id: vp-legal-docs-cheque
    lens: business-analyst
    severity: P3
    issue: 'Cancelled Cheque described as "optional, can add later" with no explanation of the consequence of not adding it — chef does not know payouts will be blocked without it'
    evidence_excerpt: "Cancelled Cheque / Bank Proof / For setting up direct payouts to your bank account (optional, can add later)"
    recommendation: >
      Add a consequence clause: "Without bank proof, payouts default to UPI. Add later in
      Settings → Payout Details." This converts "optional" from a reassuring skip to an
      informed skip. A chef who skips and then cannot receive a bank transfer will blame
      the platform, not their earlier choice. Informed consent at the skip moment reduces
      payout-related support contacts.
    metric_hypothesis: "chef payout setup completion rate; payout support ticket volume"
    depends_on: null

  - finding_id: BA-021
    surface_id: ap-secsettings-2fa-exempt
    lens: business-analyst
    severity: P3
    issue: 'Placeholder email "service@fe3dr.com" is hardcoded in the 2FA exempt list textarea — if shipped to production, a non-admin account is pre-exempted from 2FA'
    evidence_excerpt: "placeholder=\"service@fe3dr.com\""
    recommendation: >
      Change placeholder to a generic example format: "placeholder=\"service-account@yourdomain.com\"".
      This is a BA severity P3 because the placeholder value does not persist unless an admin
      saves it, but the specific internal email in a placeholder normalises the idea of this
      being a valid entry and risks a copy-paste error that exempts a real service account.
      From a trust lens: admins see a pre-populated example and may assume it belongs there.
    metric_hypothesis: "admin security compliance; 2FA enforcement rate"
    depends_on: null

  - finding_id: BA-022
    surface_id: mv-onb-policies-checkbox
    lens: business-analyst
    severity: P2
    issue: 'Mobile-vendor policies checkbox reads "I accept the terms and conditions" — generic legal boilerplate with no link to what terms are being accepted'
    evidence_excerpt: "I accept the terms and conditions"
    recommendation: >
      Replace with: "I accept the Vendor Terms at homechef.in/vendor-terms." Make the URL
      a tappable link via Linking.openURL. The generic "terms and conditions" label is the
      lowest-trust variant of consent copy — it signals that the terms are hidden, not
      transparent. After brand resolution (BA-003), the URL and brand name should be updated
      to match the canonical domain.
    metric_hypothesis: "chef mobile onboarding completion; legal consent validity"
    depends_on: BA-003

  - finding_id: BA-023
    surface_id: vp-legal-policy-hygiene
    lens: business-analyst
    severity: P2
    issue: '"Fe3dr may conduct periodic kitchen checks" buried inside the hygiene policy checkbox description — consent to physical inspection hidden in prose, not surfaced as a distinct commitment'
    evidence_excerpt: "I commit to maintaining a clean and hygienic kitchen at all times... I understand Fe3dr may conduct periodic kitchen checks."
    recommendation: >
      Elevate kitchen inspection consent to a standalone sentence or callout: "By accepting,
      you agree that HomeChef may request a kitchen visit with 48 hours notice." Hiding
      inspection consent inside a hygiene commitment paragraph means chefs who skim will
      be surprised by inspection requests, generating complaints and trust erosion. A
      prominent but plainly-worded inspection notice is both fairer and more legally defensible.
    metric_hypothesis: "chef kitchen inspection consent rate; chef complaint rate post-inspection"
    depends_on: null
```

## Brand Voice findings

```yaml
# Brand-Voice lens findings — LEGAL category
# Auditor: brand-voice lens agent
# Scope: 47 legal-category surfaces across web, vendor-portal, admin-portal, mobile-vendor, mobile-delivery, api
# Date: 2026-05-13

findings:
  # ─────────────────────────────────────────────────────────────────────────
  # P0 — Brand name drift (single biggest brand-trust break in legal copy)
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BV-001
    surface_id: api-email-base-footer
    lens: brand-voice
    severity: P0
    issue: "Brand-name drift in legal footer of every transactional email — 'Fe3dr by HomeChef' contradicts every other surface which uses only 'Fe3dr'"
    evidence_excerpt: "Fe3dr by HomeChef · Authentic homemade food, delivered · fe3dr.com · Privacy · Terms"
    related_surfaces:
      - web-mkt-brand-copyright
      - vp-legal-login-footer
      - vp-legal-register-footer
      - vp-legal-policy-tos
      - mv-onb-policies-terms-body
      - ap-auth-footer-legal
    recommendation: "Decide on ONE legal brand name and use it everywhere. Recommend 'Fe3dr' as the consumer brand and drop 'HomeChef' entirely from user-facing copy (or invert if HomeChef is the legal entity). Footer rewrite: 'Fe3dr · Authentic homemade food, delivered · fe3dr.com · Privacy · Terms'. Document the legal-entity-vs-brand distinction in STYLE-GUIDE.md section 3."
    depends_on: null

  - finding_id: BV-002
    surface_id: mv-onb-policies-terms-body
    lens: brand-voice
    severity: P0
    issue: "Mobile vendor onboarding uses 'HomeChef' as brand name and links to 'homechef.in' domain — every other surface uses 'Fe3dr' / 'fe3dr.com'. A chef onboarding on mobile and then logging into vendor-portal on web will see two different brand names."
    evidence_excerpt: "By joining HomeChef as a vendor you agree to maintain food hygiene standards per FSSAI regulations... HomeChef reserves the right to suspend accounts... Full terms available at homechef.in/vendor-terms."
    related_surfaces:
      - vp-legal-login-footer
      - vp-legal-register-footer
      - vp-legal-policy-tos
      - api-email-base-footer
    recommendation: "Replace all 'HomeChef' with 'Fe3dr' and 'homechef.in/vendor-terms' with 'fe3dr.com/vendor-terms'. This is a binding legal block — the brand name in it must match the brand the chef is actually signing up for."
    depends_on: null

  - finding_id: BV-003
    surface_id: mv-settings-delete-contact-email
    lens: brand-voice
    severity: P0
    issue: "Account-deletion support email uses 'support@homechef.app' — third brand-name variant (the .app TLD doesn't appear anywhere else in the product). Customer/chef has no way to verify this is a legitimate Fe3dr address."
    evidence_excerpt: "Please email support@homechef.app to delete your account."
    related_surfaces:
      - api-email-base-footer
      - mv-onb-policies-terms-body
    recommendation: "Use support@fe3dr.com (matching the canonical brand and domain). Add a 'Support email' constant in shared config so this can't drift again."
    depends_on: BV-001

  # ─────────────────────────────────────────────────────────────────────────
  # P0 — Terms naming drift (same legal concept, four different names)
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BV-004
    surface_id: vp-legal-register-footer
    lens: brand-voice
    severity: P0
    issue: "Chef sees 'Vendor Terms' on register page but 'Terms of Service' on login page (same vendor-portal app, adjacent screens) — the document name drifts within a single sign-up flow"
    evidence_excerpt: "By registering, you agree to Fe3dr's Vendor Terms and Privacy Policy"
    related_surfaces:
      - vp-legal-login-footer
      - vp-legal-policy-tos
      - web-legal-register-terms
      - md-leg-001
    recommendation: "Pick one canonical name for the chef-facing terms document and use it everywhere. Recommendation: 'Vendor Terms' (specific, accurate for the audience) on every chef-facing surface; 'Terms of Service' on customer-facing surfaces. Never use both names for the same document."
    depends_on: null

  - finding_id: BV-005
    surface_id: vp-legal-policy-tos
    lens: brand-voice
    severity: P0
    issue: "Onboarding step uses 'Fe3dr Platform Terms of Service' — a third name for the same document. Internally the body refers to 'Fe3dr Terms of Service, Privacy Policy, and Vendor Agreement' (a fourth unnamed document)."
    evidence_excerpt: "Fe3dr Platform Terms of Service / I have read and agree to the Fe3dr Terms of Service, Privacy Policy, and Vendor Agreement."
    related_surfaces:
      - vp-legal-register-footer
      - vp-legal-login-footer
      - mv-onb-policies-checkbox
    recommendation: "Heading and body must use the same document names. If there is a 'Vendor Agreement' separate from 'Terms of Service', it must appear in the inventory and be linked. Otherwise drop the phrase. Recommended canonical set: 'Vendor Terms', 'Privacy Policy' — two documents, two links."
    depends_on: BV-004

  - finding_id: BV-006
    surface_id: md-leg-001
    lens: brand-voice
    severity: P1
    issue: "Driver onboarding review screen shows only 'Terms of Service' / 'Privacy Policy' labels with no brand prefix and no actual navigation wired up — the driver agrees to a document we don't show them. The 4th naming variant (no prefix) across the product."
    evidence_excerpt: "\"Terms of Service\" / \"Privacy Policy\""
    related_surfaces:
      - md-leg-002
      - vp-legal-login-footer
      - web-legal-register-terms
    recommendation: "Use 'Fe3dr Driver Terms' (matching the Vendor Terms naming pattern for the driver persona). Wire the link to the actual document. Drivers should never sign an unnamed/unlinked legal document on first launch."
    depends_on: BV-004

  - finding_id: BV-007
    surface_id: mv-onb-policies-checkbox
    lens: brand-voice
    severity: P1
    issue: "Generic 'terms and conditions' label — fifth document-name variant. Voice principle Rule 2 (plain English) + Rule 4 (one accent) violated: vague legal phrasing with no brand reference and no link to what's being accepted."
    evidence_excerpt: "I accept the terms and conditions"
    related_surfaces:
      - md-leg-001
      - vp-legal-policy-tos
    recommendation: "Rewrite to 'I accept the Fe3dr Vendor Terms and Privacy Policy' with both phrases as tappable links. Matches the vendor-portal web register footer and resolves the drift."
    depends_on: BV-004

  # ─────────────────────────────────────────────────────────────────────────
  # P1 — Pronoun / persona-tone violations (style-guide section 5)
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BV-008
    surface_id: mv-onb-policies-terms-body
    lens: brand-voice
    severity: P1
    issue: "Legal block addresses chef in third person ('vendor', 'accounts') instead of 'you' — violates STYLE-GUIDE.md section 5 ('We/you, never the Company/the User/the Service Provider')"
    evidence_excerpt: "By joining HomeChef as a vendor you agree... HomeChef reserves the right to suspend accounts that receive repeated hygiene complaints..."
    related_surfaces:
      - vp-legal-policy-hygiene
      - vp-legal-policy-cancellation
      - vp-legal-policy-tos
    recommendation: "Rewrite in we/you voice: 'By joining Fe3dr as a chef, you agree to maintain food hygiene per FSSAI rules, describe your menu accurately, prepare orders within your stated prep time, and follow local food safety laws. We may suspend your account if we receive repeated hygiene complaints or if your documents fail verification. Full terms: fe3dr.com/vendor-terms.'"
    depends_on: null

  - finding_id: BV-009
    surface_id: vp-legal-docs-banner-title
    lens: brand-voice
    severity: P2
    issue: "Mixed voice: 'Documents help us verify your kitchen and enable payouts' — uses 'us' / 'your' correctly here, but the surrounding StepDocuments labels ('PAN Card', 'Aadhaar Card', 'Identity Documents') are written as inventory labels rather than addressing the chef. Tone violates persona matrix: chef-facing should be 'functional, time-aware' — currently reads like a government form."
    evidence_excerpt: "Identity Documents / These are required to verify your identity and set up payouts."
    related_surfaces:
      - vp-legal-docs-pan
      - vp-legal-docs-aadhaar
      - vp-legal-docs-identity-title
    recommendation: "Section heading sentence case: 'Identity documents' (not 'Identity Documents' — Title Case banned per STYLE-GUIDE section 4). Keep document-name proper nouns ('PAN Card', 'Aadhaar Card', 'FSSAI License') capitalized since they are proper nouns for Indian regulatory documents."
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # P1 — Dev/internal language leaking to users
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BV-010
    surface_id: vp-legal-settings-danger
    lens: brand-voice
    severity: P1
    issue: "Toast leaks dev-environment language to chef in production: 'Account deactivation is not available in demo mode'. Chef has no idea what 'demo mode' is. Breaks brand-personality goal of 'confident · quietly modern' — sounds like an unfinished prototype."
    evidence_excerpt: "Account deactivation is not available in demo mode"
    related_surfaces:
      - vp-legal-kitchen-payout-warning
      - mv-settings-delete-confirm-body
    recommendation: "Either ship the deactivation flow or write a chef-facing message: 'Account deactivation isn't ready yet. Email support@fe3dr.com to deactivate your kitchen.' Never use 'demo mode' anywhere user-facing."
    depends_on: null

  - finding_id: BV-011
    surface_id: vp-legal-kitchen-payout-warning
    lens: brand-voice
    severity: P1
    issue: "Banking-data collection helper text leaks implementation state and uses legally fraught wording: 'These details are saved locally for now.' Voice rule 'Plain English' is met but the message is alarming and ambiguous (local to browser? local to chef's machine? not transmitted?) — chefs entering bank details deserve a clear privacy statement, not a TODO note."
    evidence_excerpt: "Payout integration coming soon. These details are saved locally for now."
    related_surfaces:
      - vp-legal-settings-danger
      - vp-legal-docs-banner-body
    recommendation: "If payouts aren't wired, don't collect bank details. If they are, write: 'Your bank details are encrypted and used only to send your payouts.' Never use 'saved locally' for financial data — it reads as either confused or careless."
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # P2 — Title Case drift in legal headings
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BV-012
    surface_id: vp-legal-policies-compliance-title
    lens: brand-voice
    severity: P2
    issue: "Title Case in legal heading 'Kitchen Compliance Checklist' violates STYLE-GUIDE section 4 (sentence case for labels and headings, Title Case banned outside proper nouns)"
    evidence_excerpt: "Kitchen Compliance Checklist"
    related_surfaces:
      - vp-legal-policies-agreements-title
      - vp-legal-policy-hygiene
      - vp-legal-policy-cancellation
      - vp-legal-policy-tos
      - vp-legal-docs-identity-title
      - vp-legal-settings-danger
      - mv-onb-policies-cancel-label
      - ap-auth-restricted-heading
      - ap-secsettings-2fa-enforce
    recommendation: "Sentence case throughout: 'Kitchen compliance checklist', 'Agreements and policies', 'Kitchen hygiene and food safety commitment', 'Order and cancellation policy', 'Fe3dr platform terms of service', 'Identity documents', 'Danger zone', 'Cancellation policy', 'Restricted access', 'Require 2FA for all admins'. Proper nouns (Fe3dr, FSSAI, PAN, Aadhaar, 2FA) stay capitalized."
    depends_on: null

  - finding_id: BV-013
    surface_id: ap-auth-footer-legal
    lens: brand-voice
    severity: P2
    issue: "Title Case in admin-portal footer 'Fe3dr Administration Portal · Internal Use Only' — also slightly stiff ('Administration' could be 'Admin'). Admin persona is 'neutral operator, precise', so the stiffness is acceptable, but the Title Case is not."
    evidence_excerpt: "Fe3dr Administration Portal · Internal Use Only"
    related_surfaces:
      - ap-auth-restricted-heading
      - ap-auth-restricted-body
    recommendation: "Sentence case: 'Fe3dr Admin · Internal use only'. Shortens to fit admin's telegraphic voice and removes the Title Case violation. Proper noun 'Fe3dr' stays cased."
    depends_on: BV-012

  - finding_id: BV-014
    surface_id: ap-auth-restricted-body
    lens: brand-voice
    severity: P3
    issue: "Admin-portal restricted-access message uses 'administrators' rather than 'admins' — minor verbosity violation for the admin persona ('precise, no fluff')"
    evidence_excerpt: "Only pre-authorized administrators can sign in. New registrations are not allowed."
    related_surfaces:
      - ap-auth-restricted-heading
    recommendation: "Tighten to: 'Only pre-authorized admins can sign in. Registration isn't open.' Saves three words, drops passive voice, matches the persona-tone matrix."
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # P2 — Cross-surface inconsistency in FSSAI required/optional + naming
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BV-015
    surface_id: vp-legal-profile-doc-types
    lens: brand-voice
    severity: P1
    issue: "Same document labeled inconsistently across two surfaces of the same vendor-portal app: onboarding calls FSSAI License 'optional' with chef-friendly framing ('If you have one — gives your profile a verified badge'); profile-page DOCUMENT_TYPES marks it required:true. Also two different document names — 'Food Safety Training Certificate' (onboarding) vs 'Food Safety Certificate' (profile)."
    evidence_excerpt: "FSSAI License (Food Safety and Standards Authority certificate) ... required: true. Food Safety Certificate ... description: 'Food safety training certificate'"
    related_surfaces:
      - vp-legal-docs-fssai
      - vp-legal-docs-food-safety
      - vp-legal-docs-banner-body
    recommendation: "Reconcile in one place: define document set as a shared constant. Use the same name everywhere — recommend 'FSSAI License' and 'Food Safety Training Certificate'. Decide once whether FSSAI is required or optional and stop saying both. Brand-voice impact: a chef who saw 'optional' during onboarding and now sees 'required' on their profile loses trust."
    depends_on: null

  - finding_id: BV-016
    surface_id: vp-legal-docs-fssai
    lens: brand-voice
    severity: P3
    issue: "Inconsistent em-dash vs hyphen in onboarding description ('If you have one - gives your profile…' uses hyphen as parenthetical, should be em-dash per editorial typography). Same pattern in cheque/optional copy."
    evidence_excerpt: "If you have one - gives your profile a verified badge. You can add this later."
    related_surfaces:
      - vp-legal-docs-cheque
      - vp-legal-docs-food-safety
    recommendation: "Use em-dash (—) for parentheticals: 'If you have one — gives your profile a verified badge.' Quietly modern editorial typography is part of brand voice."
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # P2 — Operational/cold tone where chef voice should be warmer-functional
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BV-017
    surface_id: ap-chefs-reject-reason
    lens: brand-voice
    severity: P1
    issue: "Hardcoded rejection reason 'Rejected by admin' is sent to the chef as legal-grade justification. Tone is cold-bureaucratic and content-empty. Violates brand-personality goal of trust + calm; reads as Soviet-era stamp rather than confident operator."
    evidence_excerpt: "Rejected by admin"
    related_surfaces:
      - ap-approvaldetail-notes-helper
    recommendation: "Never send a hardcoded reason. Require the admin to write a specific reason before submitting (form validation). Chef-facing rewrite once admin provides input: 'Your application needs more information: <reason>. Reply to support@fe3dr.com to resubmit.'"
    depends_on: null

  - finding_id: BV-018
    surface_id: vp-legal-policy-hygiene
    lens: brand-voice
    severity: P2
    issue: "Hygiene policy mixes warm self-pledge tone ('I commit to maintaining a clean and hygienic kitchen at all times') with operational obligation ('Fe3dr may conduct periodic kitchen checks') — the persona swings from earnest first-person to platform-as-watchdog within one paragraph. Plus 'sanitised' (British) appears next to American spellings elsewhere."
    evidence_excerpt: "I commit to maintaining a clean and hygienic kitchen at all times. I will use fresh ingredients, follow proper food handling practices, store food at safe temperatures, and ensure all cooking utensils and surfaces are sanitised regularly. I understand Fe3dr may conduct periodic kitchen checks."
    related_surfaces:
      - vp-legal-policy-cancellation
      - vp-legal-policies-compliance-confirm
      - vp-legal-policies-compliance-items
    recommendation: "Pick one stance and hold it. Recommend functional second-person ('you'): 'You agree to keep your kitchen clean, use fresh ingredients, handle and store food at safe temperatures, and sanitize utensils and surfaces regularly. We may run periodic kitchen checks.' Standardize on 'sanitize' (US spelling) per the rest of the codebase."
    depends_on: BV-008

  - finding_id: BV-019
    surface_id: vp-legal-policies-compliance-items
    lens: brand-voice
    severity: P3
    issue: "Compliance items mix sentence styles within a single 7-item attestation list. Three use full sentences ('My kitchen has access to clean running water'); others are partial ('I will label allergens and ingredients for each dish' future tense vs 'I store raw and cooked food separately' present). Inconsistent verb mood breaks list rhythm."
    evidence_excerpt: "My kitchen has access to clean running water / I use separate cutting boards for vegetarian and non-vegetarian items / I store raw and cooked food separately / I wear clean attire and maintain personal hygiene while cooking / I have a designated waste disposal area away from cooking space / I ensure all food is freshly prepared on the day of delivery / I will label allergens and ingredients for each dish"
    related_surfaces:
      - vp-legal-policy-hygiene
    recommendation: "Rewrite to consistent present-tense or imperative voice. Imperative is cleaner for a checklist (matches driver-app telegraphic style for any attestation list): 'Clean running water in the kitchen', 'Separate cutting boards for veg and non-veg', 'Raw and cooked food stored separately', 'Clean attire and personal hygiene while cooking', 'Designated waste disposal area away from cooking space', 'Food prepared fresh on the day of delivery', 'Allergens and ingredients labeled on each dish'."
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # P2 — Footer / tagline brand-voice issues
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BV-020
    surface_id: api-email-base-footer
    lens: brand-voice
    severity: P2
    issue: "Tagline 'Authentic homemade food, delivered' uses 'authentic' — adjacent to banned brand-drift terms ('artisanal', 'handcrafted with love', 'curated' per STYLE-GUIDE section 3). 'Authentic' carries the same artisanal-craft kitsch the brand is explicitly trying to avoid per .impeccable.md ('Appetizing, not artisanal')."
    evidence_excerpt: "Fe3dr by HomeChef · Authentic homemade food, delivered · fe3dr.com · Privacy · Terms"
    related_surfaces:
      - web-mkt-brand-copyright
    recommendation: "Drop 'Authentic'. Try 'Home-cooked food, delivered' or just 'Home cooks, delivered' (matches the brand's confident-restrained tone). 'Authentic' is the food-DTC tell the .impeccable.md anti-references warn against."
    depends_on: BV-001

  - finding_id: BV-021
    surface_id: web-mkt-brand-copyright
    lens: brand-voice
    severity: P3
    issue: "Copyright line says only 'Fe3dr' while email footer says 'Fe3dr by HomeChef'. Web register page heading separately says 'Join Fe3dr Today' (Title Case + filler word). Brand surfaces drift across three legal touchpoints customers see on day one."
    evidence_excerpt: "© {year} Fe3dr. All rights reserved."
    related_surfaces:
      - api-email-base-footer
      - web-legal-register-terms
    recommendation: "Standardize on 'Fe3dr' (no 'by HomeChef', no 'Today'). 'All rights reserved' is acceptable boilerplate but lower-case 'rights reserved' is more on-brand: '© {year} Fe3dr. All rights reserved.' is fine; ensure year is computed not hardcoded."
    depends_on: BV-001

  # ─────────────────────────────────────────────────────────────────────────
  # P2 — Cancellation-policy option labels (cross-surface verb-mood drift)
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BV-022
    surface_id: mv-onb-policies-cancel-option-no
    lens: brand-voice
    severity: P3
    issue: "Cancellation option labels mix noun-phrase ('No cancellations after order accepted') with preposition-phrase ('Up to 1 hour before prep start' / 'Up to 30 mins before prep start') — different grammar shapes for what should be parallel radio options. Also 'mins' abbreviation drifts from 'hour' (full word) within the same option set."
    evidence_excerpt: "No cancellations after order accepted / Up to 1 hour before prep start / Up to 30 mins before prep start"
    related_surfaces:
      - mv-onb-policies-cancel-option-1h
      - mv-onb-policies-cancel-option-30m
    recommendation: "Parallel grammar and consistent units: 'No cancellations after acceptance' / 'Up to 1 hour before prep' / 'Up to 30 minutes before prep'. Drop redundant 'start'; use full word 'minutes'."
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # P2 — Encryption / privacy claims need brand-voice + legal alignment
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BV-023
    surface_id: vp-legal-docs-banner-body
    lens: brand-voice
    severity: P2
    issue: "Privacy promise is grammatically passive and over-stated for brand voice. 'Your documents are encrypted and stored securely. They are only used for verification' uses 'are encrypted/stored/used' passive constructions where STYLE-GUIDE section 5 mandates active voice ('We refund within 7 days', not 'Refunds are processed within 7 days')."
    evidence_excerpt: "Your documents are encrypted and stored securely. They are only used for verification. FSSAI license is optional — many home chefs start without one and add it later."
    related_surfaces:
      - vp-legal-kitchen-payout-warning
      - vp-legal-docs-banner-body
    recommendation: "Active voice + we/you: 'We encrypt and store your documents securely. We only use them to verify your kitchen. FSSAI license is optional — many home chefs start without one and add it later.' Also verify the encryption claim is technically accurate before shipping (separate legal-lens concern)."
    depends_on: BV-008

  # ─────────────────────────────────────────────────────────────────────────
  # Summary
  # ─────────────────────────────────────────────────────────────────────────

summary:
  total_findings: 23
  by_severity:
    P0: 5   # brand-name and terms-naming drift (BV-001..BV-005)
    P1: 8   # pronoun violations, dev-language leaks, document required/optional drift, hardcoded rejection (BV-006, BV-007, BV-008, BV-010, BV-011, BV-015, BV-017)
    P2: 8   # title-case, tone drift, encryption claim phrasing, tagline (BV-009, BV-012, BV-013, BV-018, BV-020, BV-023)
    P3: 5   # em-dash, list grammar, copyright, admin verbosity, cancellation option parallelism (BV-014, BV-016, BV-019, BV-021, BV-022)
  top_cross_app_drift_patterns:
    - "Brand name drift: 'Fe3dr' (web, vendor-portal web, admin-portal, mobile-delivery, most API), 'Fe3dr by HomeChef' (every transactional email footer), 'HomeChef' (mobile-vendor vendor-terms constant + homechef.in URL), 'homechef.app' (mobile-vendor support email) — four variants across 6 surfaces"
    - "Terms-document naming drift: 'Terms of Service' (web register, vp login), 'Vendor Terms' (vp register), 'Fe3dr Platform Terms of Service' (vp onboarding heading), 'terms and conditions' (mv onboarding checkbox), 'Terms of Service' unprefixed link (mobile-delivery review) — five names for what appears to be the same document"
    - "Pronoun voice: vendor-portal onboarding policies (StepPolicies.tsx) use first-person 'I commit / I agree' (warm self-pledge), but mobile-vendor VENDOR_TERMS_TEXT uses third-person 'HomeChef reserves the right' (cold platform-watchdog), and admin-portal restricted-access copy uses agentless passive 'New registrations are not allowed' — three pronoun stances within one product"
    - "Title Case bleed into legal headings: 'Kitchen Compliance Checklist', 'Identity Documents', 'Danger Zone', 'Restricted Access', 'Cancellation Policy', 'Internal Use Only' — should be sentence case per STYLE-GUIDE section 4"
    - "FSSAI / Food Safety document naming inconsistency within vendor-portal: 'FSSAI License' optional during onboarding, 'FSSAI License' required on profile; 'Food Safety Training Certificate' becomes 'Food Safety Certificate' between the two screens"
    - "Dev-state language leaking to chefs: 'demo mode', 'saved locally for now', 'coming soon' in legally-sensitive contexts (account deactivation, bank-detail capture) — breaks 'confident, quietly modern' brand goal"
  output_file: docs/content-audit/.work/findings-legal-bv.yaml
```
