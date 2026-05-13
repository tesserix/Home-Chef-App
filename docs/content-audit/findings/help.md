# Findings — Help

Category: help
Lenses applied: technical-writer, legal, business-analyst, brand-voice
Inventory slice size: 12 surfaces
Total findings: 42

## Severity summary

| Severity | TW | Legal | BA | BV | Total |
|---|---|---|---|---|---|
| P0 | 0 | 4 | 0 | 0 | 4 |
| P1 | 0 | 5 | 3 | 2 | 10 |
| P2 | 7 | 2 | 4 | 4 | 17 |
| P3 | 5 | 1 | 3 | 2 | 11 |

## Cross-lens consensus surfaces (flagged by multiple lenses)

Surfaces flagged by 4 lenses (highest priority — every lens agreed there's a problem):

- **`web-help-otto-launcher`** — flagged by TW(P3), Legal(P2), BA(P3), BV(P1) — Otto launcher aria-label "Open support chat" + AI/grievance disclosure gap + three-name brand drift (`support chat` / `HomeChef support` / `Otto`)
- **`web-help-otto-empty`** — flagged by TW(P3), Legal(P0), BA(P2), BV(P3) — "Ask anything — order status, delivery, refunds, chef availability." AI/refund disclosure gap, unstructured intake, empty-state formula drift
- **`dp-help-new-driver-body`** — flagged by TW(P2), Legal(P0), BA(P1), BV(P2) — "Sign up with your email or Google account. You'll go through a quick onboarding process to start delivering." 18-word driver tone violation + "quick onboarding" misleading-claim + Fe3dr brand drift
- **`dp-help-staff-access-body`** — flagged by TW(P2), Legal(P1), BA(P3), BV(P2) — "Only pre-authorized fleet managers and delivery operations staff can sign in here. Contact an admin for access." DPDP §13 escalation gap + 17-word tone bust + vague "an admin"
- **`dp-help-status-contact`** — flagged by TW(P2), Legal(P0), BA(P1, P2), BV(P1) — "Need Help? / Contact us at support@fe3dr.com" Grievance Officer/timeline absence + Fe3dr vs HomeChef brand contradiction + no SLA on high-anxiety wait

Surfaces flagged by 3 lenses (also high-priority rewrites):

- **`web-help-otto-input`** — flagged by TW(P3), Legal(P2), BV(P2) — "Type a message..." / "Connecting..." DPDP §6 in-context notice gap + bland infrastructure microcopy
- **`dp-help-new-driver`** — flagged by TW(P2), BA(P1), BV(P2) — "New Driver?" Title Case + question framing + missing sign-up CTA dead-ends supply funnel
- **`dp-help-settings-help`** — flagged by TW(P2, P3), Legal(P1), BA(P2) — "Help & Support / Get help with your deliveries" dead-button + narrow scope misses earnings/data/account grievances
- **`ap-settings-payment-test-cards`** — flagged by TW(P2, P3), Legal(P0), BA(P3) — "Test Cards / Visa / Mastercard / UPI / 3D Secure" production-exposure risk under RBI PA MD §8 + missing test-mode label + missing helper text

Surfaces flagged by 2 lenses:

- **`web-help-otto-sender`** — flagged by Legal(P1), BV(P3) — "Otto · AI assistant / Support" persistent-AI-disclosure absence + dual sender label voice mix
- **`web-help-otto-chat-header`** — flagged by Legal(P1×2), BA(P2) — "HomeChef support chat / HomeChef support / Connecting..." conflates AI session with statutory grievance channel + no AI subtitle + no SLA disclosure
- **`dp-help-staff-access-title`** — flagged by TW(P2), Legal(P3) — "Staff Access" Title Case + internal-role disclosure surface needs paired auth/data note

## Technical Writer findings

```yaml
findings:

  # ===========================================================
  # WEB — Otto support chat (now external widget)
  # ===========================================================

  - finding_id: TW-001
    surface_id: web-help-otto-launcher
    lens: technical-writer
    severity: P3
    issue: "Inventory captures aria-label and chat strings at lines 146-260 of OttoChat.tsx, but the file is now 47 lines — a thin wrapper around @tesserix/otto-widget. All Otto chat copy ('Open support chat', 'HomeChef support chat', 'Connecting...', 'Ask anything — order status, delivery, refunds, chef availability.', 'Otto · AI assistant', 'Type a message...', 'Send message') is owned by the external widget package, not this repo. TW findings on those rows must be filed against slm-support-platform, not HomeChef."
    evidence_excerpt: "OttoChat.tsx is now 47 lines wrapping <OttoWidget apiBaseUrl=\"/api/otto\" productName=\"HomeChef Support\" .../>"
    recommendation: "Update CONTENT-INVENTORY.md to mark web-help-otto-* rows as 'external — @tesserix/otto-widget' and route TW review to the widget repo. The only HomeChef-owned string is productName=\"HomeChef Support\" passed as a prop. Style-guide says 'HomeChef' should be one word (already correct) — no rewrite needed here, but confirm widget renders 'HomeChef Support' as a heading vs label."
    depends_on: null

  - finding_id: TW-002
    surface_id: web-help-otto-empty
    lens: technical-writer
    severity: P3
    issue: "Empty-state copy 'Ask anything — order status, delivery, refunds, chef availability.' (owned by external widget, see TW-001) follows style-guide empty-state formula well — concise, plain, action-oriented. Borderline 'Ask anything' could be read as overpromising (the bot is scoped). If/when widget copy is audited, tighten to set realistic scope."
    evidence_excerpt: "Ask anything — order status, delivery, refunds, chef availability."
    recommendation: "Forwarded suggestion for widget owner: 'Ask Otto about your order, delivery, refunds, or a chef.' Sets persona name and tightens scope. Non-blocking for HomeChef repo."
    depends_on: TW-001

  - finding_id: TW-003
    surface_id: web-help-otto-input
    lens: technical-writer
    severity: P3
    issue: "Loading-state 'Connecting...' uses three ASCII dots, not Unicode ellipsis. Same drift flagged on auth surfaces (TW-003 in findings-auth-tw.yaml). Owned by external widget — track for widget audit."
    evidence_excerpt: "Connecting..."
    recommendation: "Forwarded suggestion for widget owner: 'Connecting…' (single Unicode U+2026). Non-blocking for HomeChef repo."
    depends_on: TW-001

  # ===========================================================
  # DELIVERY-PORTAL — Login help cards
  # ===========================================================

  - finding_id: TW-004
    surface_id: dp-help-new-driver
    lens: technical-writer
    severity: P2
    issue: "Title 'New Driver?' uses Title Case + interrogative tone on what is a labeled info card, not a question prompt. Style guide §4 form labels: 'noun, sentence case, no colons'. Driver tone matrix calls for telegraphic ≤4 words. 'New Driver?' is a teaser pattern (chatty, marketing) — out of voice for the driver persona who skims login. Inconsistent with sibling card 'Staff Access' which uses Title Case noun, no question mark."
    evidence_excerpt: "New Driver?"
    recommendation: "'New driver' (sentence case, no question mark) to match the operational label pattern. If a question framing is intentional to invite signup, replace with 'Need to sign up?' — but the simpler label is more on-voice."
    depends_on: null

  - finding_id: TW-005
    surface_id: dp-help-new-driver-body
    lens: technical-writer
    severity: P2
    issue: "Body sentence is 18 words — driver persona max is 12 per style-guide §2 tone matrix. Compound construction ('Sign up with your email or Google account. You'll go through a quick onboarding process to start delivering.') buries the call-to-action behind procedural framing. 'Quick onboarding process' is vague filler — what does 'quick' mean to a driver? Banned-list adjacent (urgency-ish without being literal urgency)."
    evidence_excerpt: "Sign up with your email or Google account. You'll go through a quick onboarding process to start delivering."
    recommendation: "'Sign up with email or Google. Onboarding takes about 10 minutes.' (15 words, two crisp sentences, concrete time anchor instead of 'quick'). If true onboarding duration is unknown, drop sentence two and stop at 'Sign up with email or Google to start delivering.' (10 words)."
    depends_on: null

  - finding_id: TW-006
    surface_id: dp-help-staff-access-title
    lens: technical-writer
    severity: P2
    issue: "Title Case 'Staff Access' violates style-guide §4: 'noun, sentence case'. Same pattern as the customer-facing 'Sign In' / 'Sign Up' Title Case drift flagged elsewhere. Driver/admin audience accepts terse, but sentence case is the system rule."
    evidence_excerpt: "Staff Access"
    recommendation: "'Staff access' (sentence case). Apply identically to sibling 'New Driver' card title for consistency."
    depends_on: null

  - finding_id: TW-007
    surface_id: dp-help-staff-access-body
    lens: technical-writer
    severity: P2
    issue: "Body is 17 words — over the 12-word ceiling for driver-facing copy. While 'Staff Access' audience is technically admin (per inventory), the surface is rendered inside the driver-portal LoginPage and competes with the driver card for the same scannable space — driver tone applies. 'Pre-authorized' is jargon for a driver-side reader and hyphenated unnecessarily; 'fleet managers and delivery operations staff' is a noun stack that's hard to parse at a glance. 'Contact an admin' is also vague — which admin? How?"
    evidence_excerpt: "Only pre-authorized fleet managers and delivery operations staff can sign in here. Contact an admin for access."
    recommendation: "'For approved fleet and ops staff. Ask your manager for access.' (12 words). If contact channel is known (email, Slack), name it: 'For approved fleet and ops staff. Email ops@fe3dr.com for access.'"
    depends_on: null

  # ===========================================================
  # DELIVERY-PORTAL — Onboarding status + settings
  # ===========================================================

  - finding_id: TW-008
    surface_id: dp-help-status-contact
    lens: technical-writer
    severity: P2
    issue: "Title 'Need Help?' (Title Case + question mark) again breaks sentence-case noun rule. Question framing on a help label is the same chatty pattern flagged in TW-004. Also: 'Contact us at support@fe3dr.com' uses the @fe3dr.com domain, which is inconsistent with HomeChef branding elsewhere in copy ('HomeChef support', etc.). Either the support domain is wrong or HomeChef brand is rendered through fe3dr — flag for product owner."
    evidence_excerpt: "Need Help? / Contact us at support@fe3dr.com"
    recommendation: "Title: 'Need help' (sentence case, no question mark) or 'Support' (single-word telegraphic label, matches driver tone). Body: 'Email support@fe3dr.com.' (3 words, period, verb-first). Separately: confirm support@fe3dr.com is the correct customer-facing address vs a HomeChef-branded one — out of TW scope to decide, but flag."
    depends_on: null

  - finding_id: TW-009
    surface_id: dp-help-settings-help
    lens: technical-writer
    severity: P2
    issue: "Title 'Help & Support' uses Title Case (style-guide §4 requires sentence case). Also redundant — 'Help' and 'Support' are synonyms in this context. Body 'Get help with your deliveries' is acceptable but generic; a driver tapping this row already knows it's about deliveries. 6-word ceiling is fine."
    evidence_excerpt: "Help & Support / Get help with your deliveries"
    recommendation: "Title: 'Support' (single word, sentence case, telegraphic — matches driver tone). Body: 'Contact the ops team.' or remove body entirely; the icon + label carry it."
    depends_on: null

  - finding_id: TW-010
    surface_id: dp-help-settings-help
    lens: technical-writer
    severity: P3
    issue: "Inconsistency: settings row title 'Help & Support' (Title Case, ampersand) vs status-page title 'Need Help?' (Title Case, question, no ampersand) vs the row's own sibling 'Privacy & Security' (also Title Case + ampersand). Same concept ('get help') is named three different ways across the driver portal."
    evidence_excerpt: "Help & Support / Need Help? / Privacy & Security"
    recommendation: "Pick one label and apply consistently. Recommended: 'Support' for the row, 'Support' for the contact card on status page. 'Privacy & Security' should also become 'Privacy and security' (sentence case, no ampersand) but that's out of help-category scope — file under settings audit."
    depends_on: TW-009

  # ===========================================================
  # ADMIN-PORTAL — Payment sandbox helper labels
  # ===========================================================

  - finding_id: TW-011
    surface_id: ap-settings-payment-test-cards
    lens: technical-writer
    severity: P2
    issue: "Section label 'Test Cards' is Title Case (violates §4 sentence-case rule for labels). Brand names (Visa, Mastercard, UPI) are correctly cased — those are proper nouns, exempt. '3D Secure' is a feature/standard name (also proper-noun-ish, retain). The issue is the section header only."
    evidence_excerpt: "Test Cards"
    recommendation: "'Test cards' (sentence case). Tiny fix; appears twice in SettingsPage.tsx (line 362 and 652) — change both."
    depends_on: null

  - finding_id: TW-012
    surface_id: ap-settings-payment-test-cards
    lens: technical-writer
    severity: P3
    issue: "Missing helper text. 'Test cards' label drops the user into a list of card numbers with no context for what they do, whether to copy them, or which payment mode they apply to. Admin tone allows precision — add one helper line."
    evidence_excerpt: "Test Cards / Visa 4111 1111 1111 1111 ..."
    recommendation: "Add helper under the section header: 'Use these card numbers to test payments in sandbox mode. Any future expiry and any 3-digit CVV will work.' (Adjust to match actual gateway behavior — confirm with engineering before shipping.)"
    depends_on: TW-011
```

## Legal findings

```yaml
findings:

  # ============================================================================
  # AI ASSISTANT DISCLOSURE — Otto chatbot grievance / accuracy / data-processing
  # DPDP Act §6 (notice at point of data collection), IT Rules 2021 Rule 3(1)(b)(ii)
  # (misleading information), Consumer Protection (E-Commerce) Rules 2020 §5(3)
  # ============================================================================

  - finding_id: LEG-HLP-001
    surface_id: web-help-otto-empty
    lens: legal
    severity: P0
    issue: "Otto AI chat invites customers to ask about 'order status, delivery, refunds, chef availability' but discloses nothing about (a) the assistant being AI, (b) that conversations may be logged/processed, (c) accuracy limitations, (d) the legally binding fallback to a human grievance officer. Refunds in particular are a binding commitment under RBI PA MD §7 + Consumer Protection (E-Commerce) Rules 2020 §5(4) — if Otto promises a refund and the platform later denies it under T&C, that is misleading information under IT Rules 2021 Rule 3(1)(b)(ii) and an unfair trade practice under CPA 2019 §2(47)."
    evidence_excerpt: "Ask anything — order status, delivery, refunds, chef availability."
    recommendation: "Add a one-line disclosure in the empty-state and/or first AI message stating Otto is an AI assistant, that it may not be accurate on refund / payment / legal questions, and a link to the human grievance officer (DPDP §13 / IT Rules 2021). Refund eligibility statements from Otto must be flagged non-binding ('Final eligibility is decided per our Refund Policy'). Conversation-retention disclosure (DPDP §6 notice) must be reachable from the chat surface."
    citation: "DPDP Act 2023 §6 (Notice); IT Rules 2021 Rule 3(1)(b)(ii) (misleading information); Consumer Protection (E-Commerce) Rules 2020 §5(3)–(4) (grievance redress + accurate information); CPA 2019 §2(47) (unfair trade practice)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-HLP-002
    surface_id: web-help-otto-sender
    lens: legal
    severity: P1
    issue: "Sender label 'Otto · AI assistant / Support' is the only place the user is told this is AI. Per DPDP §6 notice principles and the Consumer Protection (E-Commerce) Rules 2020 §5(3) grievance-redress transparency expectation, AI disclosure should be unmistakable at the conversation entry point — not only in a sender chip mid-thread. Risk of misleading information if a user assumes a human agent responded."
    evidence_excerpt: "Otto · AI assistant / Support"
    recommendation: "Promote AI disclosure to a persistent banner or first system message: 'You're chatting with Otto, our AI assistant. For binding decisions on refunds, payments or complaints, ask to be connected to a human or use our Grievance Officer link.' Reinforce in the empty state and in the chat header subtitle."
    citation: "DPDP Act 2023 §6; Consumer Protection (E-Commerce) Rules 2020 §5(3); IT Rules 2021 Rule 3(1)(b)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-HLP-003
    surface_id: web-help-otto-launcher
    lens: legal
    severity: P2
    issue: "Floating launcher aria-label 'Open support chat' positions the AI chatbot as the primary support channel without exposing a parallel grievance/human-escalation route. Under IT Rules 2021 Rule 3(2) and Consumer Protection (E-Commerce) Rules 2020 §5(3), the grievance officer's contact must be conspicuously displayed and easily accessible — channelling all support through an AI bot risks an inaccessibility claim."
    evidence_excerpt: "Open support chat"
    recommendation: "Either rename launcher to 'Open help' and surface both Otto and a 'Contact grievance officer / human support' option in the opened panel, or add a separate 'Grievance' footer/header link visible on every help surface. Document this in the surface's a11y notes alongside the legal requirement."
    citation: "IT Rules 2021 Rule 3(2) (Grievance Officer publication); Consumer Protection (E-Commerce) Rules 2020 §5(3)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-HLP-004
    surface_id: web-help-otto-chat-header
    lens: legal
    severity: P1
    issue: "Chat header strings ('HomeChef support chat', 'HomeChef support', 'Close support chat', 'Conversation messages', 'Connecting...') brand the AI session as 'HomeChef support' without distinguishing automated vs human support. This conflates the AI assistant with the platform's statutory grievance-redress channel (IT Rules 2021 Rule 3(2)). If a user sends a grievance here expecting the §3(2) timelines (acknowledge ≤24h, resolve ≤15 days), and the AI does not record/escalate it, the platform is exposed."
    evidence_excerpt: "HomeChef support chat / HomeChef support / Close support chat / Conversation messages / Connecting..."
    recommendation: "Re-label the dialog to 'Otto — AI support' (or similar) so it is not branded as the platform's grievance channel. Add a footer line inside the dialog: 'Filing a formal grievance? Contact our Grievance Officer at <link>'. Ensure Otto conversation logs are retained and routable to a human within IT Rules timelines."
    citation: "IT Rules 2021 Rule 3(2)(a)–(b) (Grievance Officer publication + 24h ack / 15-day resolution); DPDP Act 2023 §13 (Grievance redress)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-HLP-005
    surface_id: web-help-otto-input
    lens: legal
    severity: P2
    issue: "Input placeholder 'Type a message...' and send button 'Send message' collect free-text from the customer with zero in-context notice about how the message and any PII inside it will be processed, retained, or whether it leaves the platform (e.g., to an LLM vendor). DPDP §6 requires notice at or before data collection; sending a chat message that may contain order IDs, addresses, payment info, or grievances is a data-collection event."
    evidence_excerpt: "Message / Type a message... / Connecting... / Send message"
    recommendation: "Add a single-line disclosure under the input (or as a persistent first-time banner) stating that the message will be processed by an AI system, may be retained for support quality, and linking to the Privacy Policy section on AI/chat. If a third-party LLM provider is used, that must be disclosed per DPDP §8(7) data-processor transparency. Discourage users from typing card numbers / OTPs in the placeholder copy."
    citation: "DPDP Act 2023 §6 (Notice); DPDP Act 2023 §8(7) (Data Processor transparency); RBI PA MD §8 (handling of payment data)"
    depends_on: "needs lawyer review"

  # ============================================================================
  # GRIEVANCE OFFICER ACCESS FROM HELP SURFACES — DPDP §13, IT Rules 2021 Rule 3(2)
  # Every help touchpoint must surface a route to the Grievance Officer
  # ============================================================================

  - finding_id: LEG-HLP-006
    surface_id: dp-help-status-contact
    lens: legal
    severity: P0
    issue: "Onboarding status help block 'Need Help? Contact us at support@fe3dr.com' is the driver's primary support touchpoint during onboarding but (a) does not name the Grievance Officer required under IT Rules 2021 Rule 3(2) and DPDP §13, (b) does not list response/ack timelines, (c) uses an unfamiliar 'fe3dr.com' domain that contradicts the HomeChef brand identity — a possible misleading-information issue under IT Rules 2021 Rule 3(1)(b)(ii)."
    evidence_excerpt: "Need Help? + Contact us at support@fe3dr.com"
    recommendation: "Resolve domain mismatch first (this looks like dev placeholder copy in production; confirm support email is on the registered brand domain or disclose the entity that owns 'fe3dr.com'). Then add: Grievance Officer name + email + acknowledgement timeline (≤24h), resolution timeline (≤15 days) per IT Rules 2021 Rule 3(2). DPDP §13 requires this be readily available — onboarding status is high-stakes (driver income depends on it)."
    citation: "IT Rules 2021 Rule 3(2); DPDP Act 2023 §13; IT Rules 2021 Rule 3(1)(b)(ii) (misleading information)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-HLP-007
    surface_id: dp-help-settings-help
    lens: legal
    severity: P1
    issue: "Driver settings tile 'Help & Support / Get help with your deliveries' is the steady-state support entry for drivers, but the label scopes help narrowly to 'deliveries'. Driver grievances also cover earnings, deductions, suspensions, KYC and data rights under DPDP §11–§13. The framing risks deterring drivers from raising the very grievances the statute requires the platform to acknowledge."
    evidence_excerpt: "Help & Support + Get help with your deliveries"
    recommendation: "Broaden the body line to cover the statutory grievance topics (earnings, account, data, deliveries) and ensure the destination page exposes Grievance Officer name + email + DPDP §13 / IT Rules 2021 Rule 3(2) timelines. Style guide allows e.g. 'Earnings, account, deliveries — and how to raise a complaint.'"
    citation: "DPDP Act 2023 §11 (Rights of Data Principal), §13 (Grievance Redress); IT Rules 2021 Rule 3(2); Consumer Protection (E-Commerce) Rules 2020 §5(3)"
    depends_on: "needs lawyer review"

  # ============================================================================
  # HELP CLAIMS THAT CONTRADICT T&C / CHEF / DRIVER ONBOARDING TERMS
  # ============================================================================

  - finding_id: LEG-HLP-008
    surface_id: dp-help-new-driver-body
    lens: legal
    severity: P0
    issue: "Driver-recruitment help text 'Sign up with your email or Google account. You'll go through a quick onboarding process to start delivering.' makes two factual claims that must be reconciled with the binding driver agreement and the platform's stated KYC obligations: (a) 'quick onboarding' may contradict the real KYC/document-collection flow the driver T&C requires (PAN, Aadhaar, DL, vehicle RC, insurance, bank account) per RBI PA KYC + MV Act 1988 / 2019; (b) 'to start delivering' implies platform-controlled work allocation, which can tilt the independent-contractor vs employee classification analysis under the Code on Social Security 2020 §2(35) (aggregator definition) and the Karnataka/Rajasthan gig-worker bills. Misleading-claim risk under IT Rules 2021 Rule 3(1)(b)(ii)."
    evidence_excerpt: "Sign up with your email or Google account. You'll go through a quick onboarding process to start delivering."
    recommendation: "Replace 'quick onboarding' with a neutral, accurate description ('Verify your details, then start delivering once approved'). Reframe to preserve independent-contractor positioning ('Take deliveries on your schedule' rather than 'start delivering' as if employment). Add a sentence that links to the driver agreement summary so the user opts in with notice of what onboarding entails (DPDP §6 notice + Consumer Protection (E-Commerce) Rules 2020 §5(2) seller-information transparency for the platform-driver relationship)."
    citation: "IT Rules 2021 Rule 3(1)(b)(ii) (misleading info); Code on Social Security 2020 §2(35) + §114 (aggregator/gig-worker classification); Motor Vehicles Act 1988 (driver licensing); RBI PA MD §6 (KYC for merchant onboarding when driver receives payouts); DPDP Act 2023 §6"
    depends_on: "needs lawyer review"

  - finding_id: LEG-HLP-009
    surface_id: dp-help-staff-access-body
    lens: legal
    severity: P1
    issue: "'Only pre-authorized fleet managers and delivery operations staff can sign in here. Contact an admin for access.' is a help/access-control disclosure but lacks the audit-trail / authorisation-record commitment required of a Data Fiduciary (DPDP §8(4)–(5)) and the access-control transparency expected under IT Rules 2021 Rule 3(1)(g). 'Contact an admin' is not a grievance channel — staff who are wrongfully denied access have no DPDP §13 redress route on this screen."
    evidence_excerpt: "Only pre-authorized fleet managers and delivery operations staff can sign in here. Contact an admin for access."
    recommendation: "Add (or link) a route for staff to escalate access denial to the Grievance Officer per DPDP §13. Specify what 'pre-authorized' means in policy terms (which role list, which approver) so the disclosure is auditable. If admin grant/revoke is logged, say so plainly — closes a §8(4) accountability gap."
    citation: "DPDP Act 2023 §8(4)–(5), §13; IT Rules 2021 Rule 3(1)(g)–(j)"
    depends_on: "needs lawyer review"

  - finding_id: LEG-HLP-010
    surface_id: dp-help-staff-access-title
    lens: legal
    severity: P3
    issue: "'Staff Access' heading on a public-facing login page reveals the existence of an internal staff role and entry point without a corresponding privacy/auth disclosure. Low risk on its own but combined with finding LEG-HLP-009 it forms an information-disclosure surface that should either be hidden behind a less obvious affordance or accompanied by an access-control / data-handling note."
    evidence_excerpt: "Staff Access"
    recommendation: "Acceptable to keep the label, but pair it (in the body text per LEG-HLP-009) with a privacy/grievance note and ensure access attempts are logged per IT Rules 2021 Rule 3(1)(g)–(j) reasonable security practices."
    citation: "IT Rules 2021 Rule 3(1)(g)–(j); DPDP Act 2023 §8(5)"
    depends_on: "needs lawyer review"

  # ============================================================================
  # MISLEADING HELP CLAIMS — sandbox/test-mode disclosure to admins (live exposure risk)
  # ============================================================================

  - finding_id: LEG-HLP-011
    surface_id: ap-settings-payment-test-cards
    lens: legal
    severity: P0
    issue: "Admin help block 'Test Cards; Visa; Mastercard; UPI; 3D Secure' lists sandbox/test payment credentials inside the admin portal settings page. Two risks: (a) if displayed in production (live merchant onboarding) this is a misleading-information surface and a security/operational lapse — RBI PA MD 2020/2024 §8 prohibits exposure of test artefacts in production payment flows that could be confused with live PAN/UPI; (b) listing 'UPI' as a test card variant alongside Visa/Mastercard mixes RBI-regulated UPI semantics with card-network semantics, risking customer/admin confusion if these strings appear outside a clearly-flagged sandbox UI."
    evidence_excerpt: "Test Cards; Visa; Mastercard; UPI; 3D Secure"
    recommendation: "Confine the test-card help block strictly to a sandbox environment behind an explicit 'Sandbox / Test mode' badge. Remove or guard the strings from production builds. Add a one-line disclaimer 'For test transactions only. Never use these in live payment flows.' Verify with RBI PA MD §8 and the payment-aggregator's developer T&C that the credentials listed are permitted to be shown to merchants/admins at all."
    citation: "RBI Payment Aggregator MD 17 Mar 2020 §8 (data security); RBI PA Update 31 Jul 2024 (PA-CB & ongoing PA reporting); IT Rules 2021 Rule 3(1)(b)(ii) (misleading information)"
    depends_on: "needs lawyer review"

  # ============================================================================
  # SUPPORT FLOW DISCLOSURES — DPDP §13 acknowledgement + response timelines absent
  # ============================================================================

  - finding_id: LEG-HLP-012
    surface_id: web-help-otto-chat-header
    lens: legal
    severity: P1
    issue: "'Connecting...' state in the Otto chat header is the only visible signal of support availability but communicates nothing about (a) expected response time, (b) what happens if Otto cannot answer (escalation path), (c) statutory grievance timelines (IT Rules 2021 Rule 3(2): acknowledge ≤24h, resolve ≤15 days; DPDP §13 similar). Without these disclosures the customer cannot tell whether their query has been received as a grievance for §13 purposes."
    evidence_excerpt: "Connecting..."
    recommendation: "Add a connection-state disclosure: 'Otto usually replies in a minute. For formal grievances we acknowledge within 24 hours and resolve within 15 days — start a grievance.' Style guide says: empty/loading states should be plain and short; cite the timeline only where the user is filing a complaint, not every loading state."
    citation: "IT Rules 2021 Rule 3(2)(a)–(b); DPDP Act 2023 §13; Consumer Protection (E-Commerce) Rules 2020 §5(3)"
    depends_on: "needs lawyer review"
```

## Business Analyst findings

```yaml
findings:

  # ─────────────────────────────────────────────────────────────────────────
  # P1 — CONVERSION-CRITICAL PATHWAY BROKEN
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BA-HLP-001
    surface_id: dp-help-new-driver
    lens: business-analyst
    severity: P1
    issue: "\"New Driver?\" sign-up hint has no CTA button — driver activation pathway dead-ends in text"
    evidence_excerpt: >
      <p class="text-sm font-medium">New Driver?</p>
      <p class="mt-1 text-xs text-muted-foreground">
        Sign up with your email or Google account. You'll go through
        a quick onboarding process to start delivering.
      </p>
    recommendation: >
      Add a tappable CTA immediately below the body copy: "Sign up to deliver" linking to
      /register or the onboarding entry point. Without it, a prospective driver who taps
      "I'm a Driver", reads this panel, and realises they have no account has no forward path
      — they must guess to use the Google or Email sign-in buttons (which serve existing
      drivers). Separating sign-in from sign-up with a clear CTA reduces drop-off on driver
      acquisition, which is a direct supply-side conversion metric.
    metric_hypothesis: "driver sign-up completion rate; dead-end copy leaves prospective drivers with no forward action and they abandon"
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # P1 — BRAND CONSISTENCY / WRONG PRODUCT NAME ON HELP SURFACE
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BA-HLP-002
    surface_id: dp-help-status-contact
    lens: business-analyst
    severity: P1
    issue: "Support email domain \"fe3dr.com\" contradicts HomeChef brand — driver who Googles \"fe3dr\" finds nothing, trust collapses"
    evidence_excerpt: "Contact us at support@fe3dr.com"
    recommendation: >
      Replace with a HomeChef-branded support address (e.g., support@homechef.in or
      drivers@homechef.in). If Fe3dr is a legitimate sub-brand or white-label, add a one-line
      explanation: "Fe3dr is HomeChef's delivery partner platform." Without that, a driver
      already anxious about an approval delay sees an unfamiliar domain and questions whether
      they submitted documents to the right company. That credibility gap increases abandonment
      at a critical drop-off point (post-submission wait state).
    metric_hypothesis: "driver application completion + trust; unrecognised domain on a help surface during high-anxiety wait erodes confidence and increases CS escalations"
    depends_on: null

  - finding_id: BA-HLP-003
    surface_id: dp-help-new-driver-body
    lens: business-analyst
    severity: P1
    issue: "\"Login or sign up to deliver with Fe3dr\" on choose-mode screen mixes brand names — drivers signed up for HomeChef, not Fe3dr"
    evidence_excerpt: "Login or sign up to deliver with Fe3dr"
    recommendation: >
      Align to the HomeChef brand: "Sign in or sign up to deliver with HomeChef." Use the
      vocabulary-correct "Sign in" (not "Login" — banned per Style Guide). If Fe3dr is
      intentional as the fleet brand, it must be introduced earlier in the driver recruitment
      funnel; its first appearance cannot be on the authentication screen.
    metric_hypothesis: "driver login completion; brand-name mismatch at point of authentication triggers distrust and search-engine abandonment"
    depends_on: null

  # ─────────────────────────────────────────────────────────────────────────
  # P2 — MISSED DEFLECTION OPPORTUNITY
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BA-HLP-004
    surface_id: dp-help-settings-help
    lens: business-analyst
    severity: P2
    issue: "\"Help & Support\" settings entry is a dead-end button with no destination — tapping it does nothing actionable"
    evidence_excerpt: >
      <p class="text-sm font-medium">Help & Support</p>
      <p class="text-xs text-muted-foreground">Get help with your deliveries</p>
    recommendation: >
      Either (a) link this button to a real help destination — FAQ page, in-app chat (Otto
      equivalent for drivers), or a mailto link — or (b) remove it until that destination
      exists. A settings entry that does nothing trains drivers to distrust the UI and reach
      for the phone instead, increasing inbound CS volume. "Get help with your deliveries"
      is the correct scope framing; the CTA just needs to go somewhere. Minimum viable fix:
      href="mailto:support@homechef.in" with subject pre-filled "Driver support request".
    metric_hypothesis: "CS ticket deflection rate; tappable dead ends funnel drivers to phone/email support rather than self-service"
    depends_on: null

  - finding_id: BA-HLP-005
    surface_id: web-help-otto-empty
    lens: business-analyst
    severity: P2
    issue: "Otto empty-state prompt mixes four topic categories with no intake routing — customers start typing blind rather than picking a topic"
    evidence_excerpt: "Ask anything — order status, delivery, refunds, chef availability."
    recommendation: >
      Replace the open prompt with 3–4 quick-reply chips: "Track my order", "Request a
      refund", "Chef availability", "Something else." These map directly to the four topics
      already listed in the prompt. Structured intake reduces misclassification in the support
      pipeline, speeds time-to-resolution, and means fewer conversations that bounce to a
      human agent. The freeform fallback "Something else" preserves the open channel.
      Expected metric: first-response resolution rate up, avg handle time down.
    metric_hypothesis: "support deflection rate and CS cost; unstructured intake produces more human-escalated tickets than topic-scoped intake"
    depends_on: null

  - finding_id: BA-HLP-006
    surface_id: web-help-otto-chat-header
    lens: business-analyst
    severity: P2
    issue: "Chat header labels bot as \"HomeChef support\" without signalling AI — customers expecting a human agent are surprised on escalation"
    evidence_excerpt: "HomeChef support chat / HomeChef support / Otto · AI assistant / Support"
    recommendation: >
      The sender label correctly says "Otto · AI assistant" (BA-HLP-005 area), but the
      dialog header reads "HomeChef support" with no AI qualifier. Add a subtitle:
      "AI assistant — type your question to get started" in the header aria-description or
      visible subtext. Customers who discover mid-conversation that they were talking to
      AI feel deceived, which is a trust signal that damages repeat order rate more than
      any marketing copy can recover. This is especially acute for refund and complaint flows.
    metric_hypothesis: "customer trust score and chat CSAT; unlabelled AI support inflates complaint escalations when customers realise they were not talking to a human"
    depends_on: null

  - finding_id: BA-HLP-007
    surface_id: dp-help-status-contact
    lens: business-analyst
    severity: P2
    issue: "Onboarding status page offers only email contact — no SLA promise, no chat, no phone — during high-anxiety approval-wait state"
    evidence_excerpt: >
      Need Help?
      Contact us at support@fe3dr.com
    recommendation: >
      Drivers waiting for approval are the highest-churn moment in the supply funnel. Add:
      (1) an expected response time — "We respond within 1 business day." — so drivers
      do not repeatedly email wondering if anyone received their message; (2) a direct link
      to any available chat (if Otto is extended to drivers) or a WhatsApp number if phone
      support exists. Even a plain "We respond within 24 hours" reduces inbound repeat
      contacts by setting expectations. Fix the email domain to HomeChef brand per BA-HLP-002.
    metric_hypothesis: "driver application abandonment rate during approval wait; no SLA promise + unknown brand email = drivers assume the process is broken and stop waiting"
    depends_on: BA-HLP-002

  # ─────────────────────────────────────────────────────────────────────────
  # P3 — MINOR FRICTION / INTERNAL SURFACES
  # ─────────────────────────────────────────────────────────────────────────

  - finding_id: BA-HLP-008
    surface_id: dp-help-staff-access-body
    lens: business-analyst
    severity: P3
    issue: "\"Contact an admin for access\" on staff login has no email, link, or next step — locked-out staff must hunt for who \"an admin\" is"
    evidence_excerpt: >
      Only pre-authorized fleet managers and delivery operations
      staff can sign in here. Contact an admin for access.
    recommendation: >
      Replace generic "Contact an admin" with a specific escalation path: an email address
      (e.g., ops@homechef.in), a Slack channel name, or an internal ticket URL. The current
      copy is only actionable if the locked-out staff already knows who to contact — which
      defeats the purpose of the help text. Low-traffic surface (internal staff login), so
      P3, but the fix is one line.
    metric_hypothesis: "internal ops efficiency; ambiguous access-denial messaging increases IT/admin support load when staff can't self-route"
    depends_on: null

  - finding_id: BA-HLP-009
    surface_id: ap-settings-payment-test-cards
    lens: business-analyst
    severity: P3
    issue: "Test-card reference panel in admin settings has no \"test mode only\" warning label — risk that an admin pastes a test card into production config"
    evidence_excerpt: >
      Test Cards
      Visa: 4111 1111 1111 1111
      Mastercard: 5267 3181 8797 5449
      UPI: success@razorpay
    recommendation: >
      Add a prominent label at the top of the panel: "Test mode only — these cards will
      decline in live mode." The panel already appears conditionally when mode === 'test',
      so the information is correct — but the copy does not tell the admin why they are
      seeing it or what to do when they switch to live. A two-sentence context note reduces
      accidental misconfiguration. Also: the Razorpay panel shows a 3D-Secure card is
      missing from its test-card list (Stripe has 4000 0027 6000 3184; Razorpay omits 3DS).
      Add the Razorpay 3DS test card (5267 3181 8797 5449 with OTP simulation) for
      completeness.
    metric_hypothesis: "payment gateway misconfiguration rate; absent mode-context label increases likelihood of admins misapplying test credentials"
    depends_on: null

  - finding_id: BA-HLP-010
    surface_id: web-help-otto-launcher
    lens: business-analyst
    severity: P3
    issue: "Floating chat launcher aria-label is \"Open support chat\" — no product name or AI signal, low discoverability for first-time customers"
    evidence_excerpt: "Open support chat"
    recommendation: >
      Upgrade the launcher label or visible tooltip to "Chat with Otto — AI support"
      (or simply "Need help? Chat with us"). The aria-label is correct for accessibility but
      the visible floating button (presumably an icon-only FAB) gives no textual cue to
      first-time visitors that support is available. Adding a visible "Help?" label on the
      FAB increases chat-open rate and reduces customers abandoning because they could not
      find support. Low effort, measurable via chat-open click-through.
    metric_hypothesis: "support chat open rate; icon-only FAB with no label has lower discoverability than labelled trigger, leading customers to email instead"
    depends_on: null
```

## Brand Voice findings

```yaml
findings:

# ─── P1: Brand-name drift and cross-surface support naming inconsistency ──────

- finding_id: BV-HLP-001
  surface_id: web-help-otto-launcher
  lens: brand-voice
  severity: P1
  issue: >-
    Support is exposed under two names on the SAME customer app: the launcher
    aria-label says 'Open support chat' (generic) while the chat header brands
    it 'HomeChef support'. The chef-facing message label is 'Otto · AI
    assistant'. Customers see three names ('support chat', 'HomeChef support',
    'Otto') for one feature. Brand-voice rule: same concept = one name
    everywhere.
  evidence_excerpt: |
    Launcher: "Open support chat"
    Header: "HomeChef support chat" / "HomeChef support"
    Sender label: "Otto · AI assistant"
  related_surfaces: ["web-help-otto-chat-header", "web-help-otto-sender", "dp-help-status-contact", "dp-help-settings-help"]
  recommendation: >-
    Pick one customer-facing name and use it consistently. Recommend dropping
    'Otto' (internal tool name, not a brand) and using 'Home Chef support'
    everywhere: launcher 'Open Home Chef support', header 'Home Chef support',
    sender label 'Home Chef support' or simply 'Support'. Keep 'AI assistant'
    out of the sender label — it makes the response feel less trustworthy
    (Rule 1: confident, not loud about being AI).
  depends_on: null

# ─── P1: Brand-name contradiction across apps (web vs delivery-portal) ────────

- finding_id: BV-HLP-002
  surface_id: dp-help-status-contact
  lens: brand-voice
  severity: P1
  issue: >-
    Cross-app brand-name contradiction. The web app brands itself 'HomeChef'
    in the Otto support chat header, while the delivery-portal footer/support
    surfaces brand the same platform 'Fe3dr' ('support@fe3dr.com', 'Fe3dr
    Delivery Portal'). One platform, two names — a driver hitting both apps
    will not know it is the same company. P1 brand-trust hit on entry surfaces.
  evidence_excerpt: |
    web: "HomeChef support"
    delivery-portal: "Contact us at support@fe3dr.com" + "Fe3dr Delivery Portal"
  related_surfaces: ["web-help-otto-chat-header", "dp-help-new-driver-body"]
  recommendation: >-
    Decide canonical brand name (style guide implies 'Home Chef' two words).
    Migrate all 'Fe3dr' references — support email, footer labels, body
    copy — to the canonical brand. If the legal entity is Fe3dr Pvt Ltd,
    surface it ONLY in legal/footer fine print, never in support copy.
  depends_on: null

# ─── P2: Title Case banned outside proper nouns (STYLE-GUIDE §5) ──────────────

- finding_id: BV-HLP-003
  surface_id: dp-help-new-driver
  lens: brand-voice
  severity: P2
  issue: >-
    'New Driver?' uses Title Case for a non-proper-noun. Style guide enforces
    sentence case for buttons and labels. Same issue on 'Staff Access', 'Need
    Help?', 'Help & Support', 'Test Cards'. Five Title Case labels on
    secondary support surfaces violate the same rule.
  evidence_excerpt: |
    "New Driver?"   (LoginPage.tsx:221)
    "Staff Access"  (LoginPage.tsx:336)
    "Need Help?"    (OnboardingStatusPage.tsx:169)
    "Help & Support" + "Get help with your deliveries" (SettingsPage.tsx:38)
    "Test Cards"    (admin SettingsPage.tsx:362, 652)
  related_surfaces: ["dp-help-staff-access-title", "dp-help-status-contact", "dp-help-settings-help", "ap-settings-payment-test-cards"]
  recommendation: >-
    Rewrite in sentence case: 'New to driving?' / 'Staff access' / 'Need
    help?' / 'Help and support' / 'Test cards'. Replace the ampersand in
    'Help & Support' with 'and' to match the style-guide phrasing rules.
  depends_on: null

# ─── P2: Persona-tone drift — driver copy too verbose (STYLE-GUIDE §2) ────────

- finding_id: BV-HLP-004
  surface_id: dp-help-new-driver-body
  lens: brand-voice
  severity: P2
  issue: >-
    Driver-facing help body is 18 words across two sentences. Style-guide
    persona matrix caps driver copy at ≤4 words where possible, and otherwise
    crisp telegraphic phrasing. 'You'll go through a quick onboarding process
    to start delivering' is conversational customer-tone copy living in a
    driver surface.
  evidence_excerpt: >-
    "Sign up with your email or Google account. You'll go through a quick
    onboarding process to start delivering."
  related_surfaces: ["dp-help-staff-access-body"]
  recommendation: >-
    Rewrite for driver tone: 'Sign up with email or Google. Quick onboarding,
    then you start delivering.' (15 words → 11 words; two short sentences).
    Or shorter still: 'Sign up. Quick onboarding. Start delivering.'
  depends_on: null

- finding_id: BV-HLP-005
  surface_id: dp-help-staff-access-body
  lens: brand-voice
  severity: P2
  issue: >-
    Same persona-tone drift as BV-HLP-004 on the staff-access help card. 17
    words, conversational, with bureaucratic phrasing ('pre-authorized fleet
    managers and delivery operations staff'). Audience here is admin/staff, so
    persona is 'precise operator', not delivery driver — but copy is still
    over-stuffed and breaks Rule 2 (plain English over jargon).
  evidence_excerpt: >-
    "Only pre-authorized fleet managers and delivery operations staff can
    sign in here. Contact an admin for access."
  related_surfaces: ["dp-help-new-driver-body"]
  recommendation: >-
    'Staff sign-in is invite-only. Ask an admin if you need access.' (12
    words, plain English, no 'pre-authorized'/'delivery operations staff'
    SaaS-speak).
  depends_on: null

# ─── P2: Empty / placeholder microcopy ('Connecting...') ───────────────────────

- finding_id: BV-HLP-006
  surface_id: web-help-otto-input
  lens: brand-voice
  severity: P2
  issue: >-
    'Connecting...' as the message-input placeholder is bland infrastructure
    chatter that violates Rule 1 ('confident, not loud' includes 'confident,
    not whiny about plumbing'). Customers do not need to be told the
    WebSocket is reconnecting in the input field. Same generic
    'Connecting...' also appears as the send-button disabled label —
    duplicated, redundant.
  evidence_excerpt: |
    placeholder: "Connecting..."
    send-button: "Connecting..."
  related_surfaces: ["web-help-otto-chat-header"]
  recommendation: >-
    Use a calm, branded placeholder regardless of socket state ('Ask Home
    Chef support'). Surface connection state only via the send button being
    disabled with a small spinner — no copy needed. If a status string is
    required, use 'Reconnecting' (no ellipsis, the spinner is the affordance).
  depends_on: null

# ─── P3: Capitalization drift on Otto sender label ────────────────────────────

- finding_id: BV-HLP-007
  surface_id: web-help-otto-sender
  lens: brand-voice
  severity: P3
  issue: >-
    Two sender labels appear back-to-back: 'Otto · AI assistant' (humanised
    bot name) and 'Support' (generic). Same chat thread, two voices. Even if
    'Otto' is kept (see BV-HLP-001), 'AI assistant' should be sentence-cased
    only and never paired with a separate 'Support' label — pick one.
  evidence_excerpt: "Otto · AI assistant / Support"
  related_surfaces: ["web-help-otto-launcher", "web-help-otto-chat-header"]
  recommendation: >-
    Single sender label per side of the conversation: 'Home Chef support' for
    inbound, 'You' for the customer. Drop 'Otto · AI assistant' entirely or
    move to a one-time disclosure ('You are chatting with an AI assistant.
    Type "human" to reach a person.') above the input.
  depends_on: null

# ─── P3: Punctuation / style on Otto empty-state prompt ───────────────────────

- finding_id: BV-HLP-008
  surface_id: web-help-otto-empty
  lens: brand-voice
  severity: P3
  issue: >-
    Em-dash separator + comma-separated topic list ('Ask anything — order
    status, delivery, refunds, chef availability.') is fine in body copy but
    feels marketing-y as an empty-state prompt. Style-guide empty-state
    formula is 'Why it's empty → One action.' The current copy skips the
    'why' and lists four scopes — closer to a hero subtitle.
  evidence_excerpt: "Ask anything — order status, delivery, refunds, chef availability."
  related_surfaces: []
  recommendation: >-
    Match the empty-state formula: 'No messages yet. Ask about an order,
    delivery, refund, or chef.' Two short sentences, no em-dash, action verb
    leads.
  depends_on: null
```
