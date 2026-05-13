# Mobile Vendor Inventory

Generated: 2026-05-13
App: apps/mobile-vendor/
Total rows: 241

Note on excluded shared strings: All copy in `LoginScreen`, `RegisterScreen`, and `ForgotPasswordScreen` is rendered from `@homechef/mobile-shared/screens` (origin: `packages/mobile-shared`). Only the props passed in from mobile-vendor are inventoried here. The login screen passes a `title="Welcome back"` override (mv-auth-login-title). Toggle / segmented controls, lucide tab icons, and pure decorative emoji glyphs (e.g. inline ✓/✗/📄/🍳) are listed only when they convey a status word, never as decorative-only.

| surface_id | app | category | route_or_file | audience | word_count | current_text_excerpt | shared_component_origin | last_edited | notes |
|---|---|---|---|---|---|---|---|---|---|
| mv-auth-login-title | mobile-vendor | auth-onboarding | app/(auth)/login.tsx | chef | 2 | Welcome back | LoginScreen prop override (@homechef/mobile-shared) | 2026-04-06 | Mirrors vendor-portal sign-in title. Shared screen body copy NOT inventoried here. |
| mv-onb-step1-title | mobile-vendor | auth-onboarding | app/(onboarding)/_layout.tsx | chef | 3 | Step 1 of 6 | — | 2026-05-12 | Stack header title; verbose for glanceable mobile chef context but standard. Drift risk: vendor-portal onboarding may differ in step count. |
| mv-onb-step2-title | mobile-vendor | auth-onboarding | app/(onboarding)/_layout.tsx | chef | 3 | Step 2 of 6 | — | 2026-05-12 | Stack header title. |
| mv-onb-step3-title | mobile-vendor | auth-onboarding | app/(onboarding)/_layout.tsx | chef | 3 | Step 3 of 6 | — | 2026-05-12 | Stack header title. |
| mv-onb-step4-title | mobile-vendor | auth-onboarding | app/(onboarding)/_layout.tsx | chef | 3 | Step 4 of 6 | — | 2026-05-12 | Stack header title. |
| mv-onb-step5-title | mobile-vendor | auth-onboarding | app/(onboarding)/_layout.tsx | chef | 3 | Step 5 of 6 | — | 2026-05-12 | Stack header title. |
| mv-onb-step6-title | mobile-vendor | auth-onboarding | app/(onboarding)/_layout.tsx | chef | 3 | Step 6 of 6 | — | 2026-05-12 | Stack header title. |
| mv-onb-pending-title | mobile-vendor | auth-onboarding | app/(onboarding)/_layout.tsx | chef | 2 | Application Status | — | 2026-05-12 | Stack header for /pending route. |
| mv-onb-personal-heading | mobile-vendor | auth-onboarding | app/(onboarding)/personal-info.tsx | chef | 2 | Personal Information | — | 2026-05-12 | Display serif heading. |
| mv-onb-personal-sub | mobile-vendor | auth-onboarding | app/(onboarding)/personal-info.tsx | chef | 4 | Tell us about yourself | — | 2026-05-12 | Subheading. Friendly tone consistent with brand. |
| mv-onb-personal-fullname-label | mobile-vendor | auth-onboarding | app/(onboarding)/personal-info.tsx | chef | 3 | Full Name * | — | 2026-05-12 | Required-marker via asterisk. |
| mv-onb-personal-fullname-ph | mobile-vendor | auth-onboarding | app/(onboarding)/personal-info.tsx | chef | 4 | Enter your full name | — | 2026-05-12 | Input placeholder. |
| mv-onb-personal-fullname-err | mobile-vendor | errors-empty | app/(onboarding)/personal-info.tsx | chef | 7 | Full name must be at least 2 characters | — | 2026-05-12 | Zod validation message. |
| mv-onb-personal-phone-label | mobile-vendor | auth-onboarding | app/(onboarding)/personal-info.tsx | chef | 3 | Phone Number * | — | 2026-05-12 | Required field. |
| mv-onb-personal-phone-ph | mobile-vendor | auth-onboarding | app/(onboarding)/personal-info.tsx | chef | 3 | 10-digit mobile number | — | 2026-05-12 | India-specific format. |
| mv-onb-personal-phone-err | mobile-vendor | errors-empty | app/(onboarding)/personal-info.tsx | chef | 7 | Enter a valid 10-digit Indian mobile number | — | 2026-05-12 | Zod validation. India-specific. |
| mv-onb-personal-email-label | mobile-vendor | auth-onboarding | app/(onboarding)/personal-info.tsx | chef | 2 | Email Address | — | 2026-05-12 | Field label. |
| mv-onb-personal-email-helper | mobile-vendor | microcopy | app/(onboarding)/personal-info.tsx | chef | 6 | Email is pre-filled from your account | — | 2026-05-12 | Helper text below disabled field. |
| mv-onb-personal-email-err | mobile-vendor | errors-empty | app/(onboarding)/personal-info.tsx | chef | 3 | Invalid email address | — | 2026-05-12 | Zod validation. |
| mv-onb-personal-cta-next | mobile-vendor | core-ux | app/(onboarding)/personal-info.tsx | chef | 1 | Next | — | 2026-05-12 | Primary CTA (also on steps 2-5). |
| mv-onb-personal-validation-alert | mobile-vendor | errors-empty | app/(onboarding)/personal-info.tsx | chef | 2 | Validation Error | — | 2026-05-12 | Alert title on submit failure. |
| mv-onb-kitchen-heading | mobile-vendor | auth-onboarding | app/(onboarding)/kitchen-details.tsx | chef | 2 | Kitchen Details | — | 2026-05-12 | Step 2 heading. Matches vendor-portal kitchen onboarding step. |
| mv-onb-kitchen-sub | mobile-vendor | auth-onboarding | app/(onboarding)/kitchen-details.tsx | chef | 5 | Tell us about your kitchen | — | 2026-05-12 | Subheading. |
| mv-onb-kitchen-business-label | mobile-vendor | auth-onboarding | app/(onboarding)/kitchen-details.tsx | chef | 3 | Business Name * | — | 2026-05-12 | Field label. |
| mv-onb-kitchen-business-ph | mobile-vendor | auth-onboarding | app/(onboarding)/kitchen-details.tsx | chef | 5 | Your kitchen / business name | — | 2026-05-12 | Placeholder. |
| mv-onb-kitchen-business-err | mobile-vendor | errors-empty | app/(onboarding)/kitchen-details.tsx | chef | 7 | Business name must be at least 3 characters | — | 2026-05-12 | Validation. |
| mv-onb-kitchen-cuisines-label | mobile-vendor | auth-onboarding | app/(onboarding)/kitchen-details.tsx | chef | 3 | Cuisine Types * | — | 2026-05-12 | Field label. |
| mv-onb-kitchen-cuisines-err | mobile-vendor | errors-empty | app/(onboarding)/kitchen-details.tsx | chef | 6 | Select at least one cuisine type | — | 2026-05-12 | Validation. |
| mv-onb-kitchen-cuisine-options | mobile-vendor | core-ux | app/(onboarding)/kitchen-details.tsx | chef | 12 | North Indian / South Indian / Chinese / Continental / Bakery / Snacks / Beverages / Other | — | 2026-05-12 | Cuisine chip labels (8 options). India-centric. |
| mv-onb-kitchen-desc-label | mobile-vendor | auth-onboarding | app/(onboarding)/kitchen-details.tsx | chef | 2 | Description * | — | 2026-05-12 | Field label. |
| mv-onb-kitchen-desc-ph | mobile-vendor | auth-onboarding | app/(onboarding)/kitchen-details.tsx | chef | 12 | Describe your kitchen, specialties, and cooking style (min 50 characters) | — | 2026-05-12 | Verbose placeholder — flag for TW lens: glanceable chef may skim past. |
| mv-onb-kitchen-desc-min-err | mobile-vendor | errors-empty | app/(onboarding)/kitchen-details.tsx | chef | 7 | Description must be at least 50 characters | — | 2026-05-12 | Validation. |
| mv-onb-kitchen-desc-max-err | mobile-vendor | errors-empty | app/(onboarding)/kitchen-details.tsx | chef | 7 | Description must be at most 500 characters | — | 2026-05-12 | Validation. |
| mv-onb-ops-heading | mobile-vendor | auth-onboarding | app/(onboarding)/operations.tsx | chef | 1 | Operations | — | 2026-05-12 | Step 3 heading. |
| mv-onb-ops-sub | mobile-vendor | auth-onboarding | app/(onboarding)/operations.tsx | chef | 8 | Set your working hours and service details | — | 2026-05-12 | Subheading. |
| mv-onb-ops-hours-label | mobile-vendor | auth-onboarding | app/(onboarding)/operations.tsx | chef | 2 | Operating Hours | — | 2026-05-12 | Section label. |
| mv-onb-ops-days | mobile-vendor | core-ux | app/(onboarding)/operations.tsx | chef | 7 | Monday / Tuesday / Wednesday / Thursday / Friday / Saturday / Sunday | — | 2026-05-12 | Day-of-week labels (sliced to 3 chars Mon/Tue/etc in UI). |
| mv-onb-ops-closed-state | mobile-vendor | core-ux | app/(onboarding)/operations.tsx | chef | 1 | Closed | — | 2026-05-12 | State label when day toggled off. |
| mv-onb-ops-preptime-label | mobile-vendor | auth-onboarding | app/(onboarding)/operations.tsx | chef | 2 | Prep Time | — | 2026-05-12 | Section label. |
| mv-onb-ops-preptime-options | mobile-vendor | core-ux | app/(onboarding)/operations.tsx | chef | 5 | 15min / 30min / 45min / 60min / 90min | — | 2026-05-12 | Chip options — no space between number and unit. Drift vs new.tsx which uses "{n} min" with space. |
| mv-onb-ops-radius-label | mobile-vendor | auth-onboarding | app/(onboarding)/operations.tsx | chef | 3 | Service Radius (km) | — | 2026-05-12 | Field label. |
| mv-onb-ops-radius-ph | mobile-vendor | auth-onboarding | app/(onboarding)/operations.tsx | chef | 3 | 1–50 km | — | 2026-05-12 | Placeholder using en-dash. |
| mv-onb-ops-radius-err | mobile-vendor | errors-empty | app/(onboarding)/operations.tsx | chef | 9 | Service radius must be between 1 and 50 km | — | 2026-05-12 | Alert body. |
| mv-onb-docs-heading | mobile-vendor | auth-onboarding | app/(onboarding)/documents.tsx | chef | 1 | Documents | — | 2026-05-12 | Step 4 heading. |
| mv-onb-docs-sub | mobile-vendor | auth-onboarding | app/(onboarding)/documents.tsx | chef | 6 | Upload your identity and FSSAI documents | — | 2026-05-12 | Subheading. FSSAI = India food safety regulator. |
| mv-onb-docs-id-slot | mobile-vendor | auth-onboarding | app/(onboarding)/documents.tsx | chef | 2 | ID Proof | — | 2026-05-12 | Slot label. |
| mv-onb-docs-fssai-slot | mobile-vendor | auth-onboarding | app/(onboarding)/documents.tsx | chef | 2 | FSSAI License | — | 2026-05-12 | Slot label. |
| mv-onb-docs-pdf-uploaded | mobile-vendor | core-ux | app/(onboarding)/documents.tsx | chef | 2 | PDF uploaded | — | 2026-05-12 | State label after upload. |
| mv-onb-docs-upload-success | mobile-vendor | transactional | app/(onboarding)/documents.tsx | chef | 2 | Uploaded successfully | — | 2026-05-12 | Confirmation under preview. |
| mv-onb-docs-upload-progress | mobile-vendor | transactional | app/(onboarding)/documents.tsx | chef | 1 | Uploading... | — | 2026-05-12 | In-flight state. |
| mv-onb-docs-source-camera | mobile-vendor | core-ux | app/(onboarding)/documents.tsx | chef | 1 | Camera | — | 2026-05-12 | Upload source button. |
| mv-onb-docs-source-gallery | mobile-vendor | core-ux | app/(onboarding)/documents.tsx | chef | 1 | Gallery | — | 2026-05-12 | Upload source button. |
| mv-onb-docs-source-pdf | mobile-vendor | core-ux | app/(onboarding)/documents.tsx | chef | 1 | PDF | — | 2026-05-12 | Upload source button. |
| mv-onb-docs-camera-perm | mobile-vendor | errors-empty | app/(onboarding)/documents.tsx | chef | 7 | Camera permission is needed to take photos. | — | 2026-05-12 | Permission denial alert body. Same string in menu/new.tsx and menu/edit.tsx. |
| mv-onb-docs-gallery-perm | mobile-vendor | errors-empty | app/(onboarding)/documents.tsx | chef | 7 | Gallery permission is needed to select photos. | — | 2026-05-12 | Permission alert body. |
| mv-onb-docs-perm-title | mobile-vendor | errors-empty | app/(onboarding)/documents.tsx | chef | 2 | Permission Required | — | 2026-05-12 | Alert title (reused across screens). |
| mv-onb-docs-required-alert | mobile-vendor | errors-empty | app/(onboarding)/documents.tsx | chef | 11 | Please upload both ID proof and FSSAI license to continue. | — | 2026-05-12 | Alert body with title "Documents Required". |
| mv-onb-docs-required-title | mobile-vendor | errors-empty | app/(onboarding)/documents.tsx | chef | 2 | Documents Required | — | 2026-05-12 | Alert title. |
| mv-onb-docs-upload-fail | mobile-vendor | errors-empty | app/(onboarding)/documents.tsx | chef | 4 | Upload failed. Please try again. | — | 2026-05-12 | Fallback error message. |
| mv-onb-policies-heading | mobile-vendor | auth-onboarding | app/(onboarding)/policies.tsx | chef | 1 | Policies | — | 2026-05-12 | Step 5 heading. |
| mv-onb-policies-sub | mobile-vendor | auth-onboarding | app/(onboarding)/policies.tsx | chef | 4 | Review and accept terms | — | 2026-05-12 | Subheading. |
| mv-onb-policies-terms-body | mobile-vendor | legal | constants/terms.ts (VENDOR_TERMS_TEXT) | chef | 60 | By joining HomeChef as a vendor you agree to maintain food hygiene standards per FSSAI regulations, ensure accurate menu descriptions, prepare orders within your stated prep time, and comply with all applicable local food safety laws. HomeChef reserves the right to suspend accounts that receive repeated hygiene complaints or fail document verification. Full terms available at homechef.in/vendor-terms. | — | 2026-04-06 | Full vendor-terms block. Drift risk: full terms URL `homechef.in/vendor-terms` only referenced from mobile here. Verify same TOS on vendor-portal. |
| mv-onb-policies-checkbox | mobile-vendor | legal | app/(onboarding)/policies.tsx | chef | 5 | I accept the terms and conditions | — | 2026-05-12 | Checkbox label. |
| mv-onb-policies-cancel-label | mobile-vendor | legal | app/(onboarding)/policies.tsx | chef | 3 | Cancellation Policy * | — | 2026-05-12 | Section label. |
| mv-onb-policies-cancel-option-no | mobile-vendor | legal | constants/terms.ts | chef | 5 | No cancellations after order accepted | — | 2026-04-06 | Cancellation policy option label. |
| mv-onb-policies-cancel-option-1h | mobile-vendor | legal | constants/terms.ts | chef | 6 | Up to 1 hour before prep start | — | 2026-04-06 | Cancellation policy option label. |
| mv-onb-policies-cancel-option-30m | mobile-vendor | legal | constants/terms.ts | chef | 6 | Up to 30 mins before prep start | — | 2026-04-06 | Cancellation policy option label. |
| mv-onb-policies-terms-required-alert | mobile-vendor | errors-empty | app/(onboarding)/policies.tsx | chef | 9 | Please accept the terms and conditions to continue. | — | 2026-05-12 | Alert body. |
| mv-onb-policies-terms-required-title | mobile-vendor | errors-empty | app/(onboarding)/policies.tsx | chef | 2 | Terms Required | — | 2026-05-12 | Alert title. |
| mv-onb-policies-policy-required-alert | mobile-vendor | errors-empty | app/(onboarding)/policies.tsx | chef | 5 | Please select a cancellation policy. | — | 2026-05-12 | Alert body. |
| mv-onb-policies-policy-required-title | mobile-vendor | errors-empty | app/(onboarding)/policies.tsx | chef | 2 | Policy Required | — | 2026-05-12 | Alert title. |
| mv-onb-review-heading | mobile-vendor | auth-onboarding | app/(onboarding)/review.tsx | chef | 2 | Review Application | — | 2026-05-12 | Step 6 heading. |
| mv-onb-review-sub | mobile-vendor | auth-onboarding | app/(onboarding)/review.tsx | chef | 5 | Confirm your details before submitting | — | 2026-05-12 | Subheading. |
| mv-onb-review-section-personal | mobile-vendor | auth-onboarding | app/(onboarding)/review.tsx | chef | 2 | Personal Information | — | 2026-05-12 | Section label (uppercase styled). |
| mv-onb-review-section-kitchen | mobile-vendor | auth-onboarding | app/(onboarding)/review.tsx | chef | 2 | Kitchen Details | — | 2026-05-12 | Section label. |
| mv-onb-review-section-ops | mobile-vendor | auth-onboarding | app/(onboarding)/review.tsx | chef | 1 | Operations | — | 2026-05-12 | Section label. |
| mv-onb-review-section-docs | mobile-vendor | auth-onboarding | app/(onboarding)/review.tsx | chef | 1 | Documents | — | 2026-05-12 | Section label. |
| mv-onb-review-section-policies | mobile-vendor | auth-onboarding | app/(onboarding)/review.tsx | chef | 1 | Policies | — | 2026-05-12 | Section label. |
| mv-onb-review-field-labels | mobile-vendor | auth-onboarding | app/(onboarding)/review.tsx | chef | 14 | Full Name / Phone / Email / Business Name / Cuisines / Description / Open Days / Prep Time / Service Radius / ID Proof / FSSAI License / Terms Accepted / Cancellation Policy | — | 2026-05-12 | Read-only review row labels. |
| mv-onb-review-doc-uploaded | mobile-vendor | core-ux | app/(onboarding)/review.tsx | chef | 1 | Uploaded | — | 2026-05-12 | Document status. |
| mv-onb-review-doc-not-uploaded | mobile-vendor | core-ux | app/(onboarding)/review.tsx | chef | 2 | Not uploaded | — | 2026-05-12 | Document status. |
| mv-onb-review-terms-yes | mobile-vendor | core-ux | app/(onboarding)/review.tsx | chef | 1 | Yes | — | 2026-05-12 | Terms accepted indicator. |
| mv-onb-review-cta | mobile-vendor | core-ux | app/(onboarding)/review.tsx | chef | 2 | Submit Application | — | 2026-05-12 | Primary CTA. |
| mv-onb-review-submit-error-title | mobile-vendor | errors-empty | app/(onboarding)/review.tsx | chef | 2 | Submission Error | — | 2026-05-12 | Alert title. |
| mv-onb-review-submit-error-body | mobile-vendor | errors-empty | app/(onboarding)/review.tsx | chef | 4 | Submission failed. Please try again. | — | 2026-05-12 | Fallback error body. |
| mv-onb-pending-logout-btn | mobile-vendor | core-ux | app/(onboarding)/pending.tsx | chef | 1 | Logout | — | 2026-05-12 | Top-right action. |
| mv-onb-pending-submitted-title | mobile-vendor | transactional | app/(onboarding)/pending.tsx | chef | 2 | Application Submitted! | — | 2026-05-12 | Display serif title; lone exclamation in app — flag if brand voice avoids emphatic punctuation. |
| mv-onb-pending-submitted-body | mobile-vendor | transactional | app/(onboarding)/pending.tsx | chef | 17 | Our team will review your application within 24-48 hours. We will notify you once the review is complete. | — | 2026-05-12 | Body copy. Verbose for glanceable mobile — flag for TW. |
| mv-onb-pending-status-pending | mobile-vendor | transactional | app/(onboarding)/pending.tsx | chef | 3 | Status: Pending Review | — | 2026-05-12 | Status pill. |
| mv-onb-pending-status-submitted | mobile-vendor | transactional | app/(onboarding)/pending.tsx | chef | 2 | Status: Submitted | — | 2026-05-12 | Status pill. |
| mv-onb-pending-rejected-title | mobile-vendor | transactional | app/(onboarding)/pending.tsx | chef | 3 | Application Not Approved | — | 2026-05-12 | Reject state heading. Softer than "Rejected" — good. |
| mv-onb-pending-rejected-reason-prefix | mobile-vendor | transactional | app/(onboarding)/pending.tsx | chef | 1 | Reason: | — | 2026-05-12 | Inline label. |
| mv-onb-pending-rejected-body | mobile-vendor | transactional | app/(onboarding)/pending.tsx | chef | 8 | Please review the feedback and resubmit your application. | — | 2026-05-12 | Body copy. |
| mv-onb-pending-rejected-cta | mobile-vendor | core-ux | app/(onboarding)/pending.tsx | chef | 1 | Reapply | — | 2026-05-12 | CTA. |
| mv-tabs-dashboard-label | mobile-vendor | core-ux | app/(tabs)/_layout.tsx | chef | 1 | Dashboard | — | 2026-05-12 | Tab bar label. |
| mv-tabs-orders-label | mobile-vendor | core-ux | app/(tabs)/_layout.tsx | chef | 1 | Orders | — | 2026-05-12 | Tab bar label. Matches vendor-portal. |
| mv-tabs-menu-label | mobile-vendor | core-ux | app/(tabs)/_layout.tsx | chef | 1 | Menu | — | 2026-05-12 | Tab bar label. Matches vendor-portal. |
| mv-tabs-more-label | mobile-vendor | core-ux | app/(tabs)/_layout.tsx | chef | 1 | More | — | 2026-05-12 | Tab bar label. |
| mv-dash-greeting-morning | mobile-vendor | microcopy | app/(tabs)/index.tsx | chef | 2 | Good morning | — | 2026-05-12 | Time-based greeting. |
| mv-dash-greeting-afternoon | mobile-vendor | microcopy | app/(tabs)/index.tsx | chef | 2 | Good afternoon | — | 2026-05-12 | Time-based greeting. |
| mv-dash-greeting-evening | mobile-vendor | microcopy | app/(tabs)/index.tsx | chef | 2 | Good evening | — | 2026-05-12 | Time-based greeting. |
| mv-dash-greeting-fallback | mobile-vendor | microcopy | app/(tabs)/index.tsx | chef | 1 | Chef | — | 2026-05-12 | Name fallback when no profile name/email. |
| mv-dash-subhead | mobile-vendor | core-ux | app/(tabs)/index.tsx | chef | 4 | Here's your kitchen overview | — | 2026-05-12 | Dashboard subhead. |
| mv-dash-stats-today-orders | mobile-vendor | core-ux | app/(tabs)/index.tsx | chef | 2 | Today's Orders | — | 2026-05-12 | Stats card title. Matches vendor-portal terminology. |
| mv-dash-stats-today-earnings | mobile-vendor | core-ux | app/(tabs)/index.tsx | chef | 2 | Today's Earnings | — | 2026-05-12 | Stats card title. Note: vendor-portal uses "Earnings", drift to track. |
| mv-dash-stats-rating | mobile-vendor | core-ux | app/(tabs)/index.tsx | chef | 1 | Rating | — | 2026-05-12 | Stats card title. |
| mv-dash-stats-reviews | mobile-vendor | core-ux | app/(tabs)/index.tsx | chef | 1 | Reviews | — | 2026-05-12 | Stats card title. |
| mv-dash-stats-reviews-sub | mobile-vendor | microcopy | app/(tabs)/index.tsx | chef | 2 | total reviews | — | 2026-05-12 | Subtitle under Reviews card. |
| mv-dash-accepting-title | mobile-vendor | core-ux | app/(tabs)/index.tsx | chef | 2 | Accepting Orders | — | 2026-05-12 | Toggle label. Matches Settings screen and vendor-portal. |
| mv-dash-accepting-on | mobile-vendor | core-ux | app/(tabs)/index.tsx | chef | 4 | Customers can place orders | — | 2026-05-12 | Toggle ON state helper. |
| mv-dash-accepting-off | mobile-vendor | core-ux | app/(tabs)/index.tsx | chef | 3 | Kitchen is closed | — | 2026-05-12 | Toggle OFF state helper. |
| mv-dash-recent-orders | mobile-vendor | core-ux | app/(tabs)/index.tsx | chef | 2 | Recent Orders | — | 2026-05-12 | Section header. Matches vendor-portal (line 265). |
| mv-dash-fail-error | mobile-vendor | errors-empty | app/(tabs)/index.tsx | chef | 4 | Failed to load dashboard | — | 2026-05-12 | Error state. |
| mv-dash-retry-cta | mobile-vendor | errors-empty | app/(tabs)/index.tsx | chef | 1 | Retry | — | 2026-05-12 | Retry button (repeated on analytics/earnings/profile/reviews/menu error states). |
| mv-orders-heading | mobile-vendor | core-ux | app/(tabs)/orders.tsx | chef | 1 | Orders | — | 2026-05-12 | Tab screen heading. |
| mv-orders-tab-live | mobile-vendor | core-ux | app/(tabs)/orders.tsx | chef | 2 | Live Queue | — | 2026-05-12 | Segmented control tab. |
| mv-orders-tab-history | mobile-vendor | core-ux | app/(tabs)/orders.tsx | chef | 1 | History | — | 2026-05-12 | Segmented control tab. |
| mv-orders-empty-title | mobile-vendor | errors-empty | app/(tabs)/orders.tsx | chef | 3 | No pending orders | — | 2026-05-12 | Empty queue title. Drift vs vendor-portal "No pending orders right now". |
| mv-orders-empty-body | mobile-vendor | errors-empty | app/(tabs)/orders.tsx | chef | 6 | New orders will appear here automatically | — | 2026-05-12 | Empty queue body. |
| mv-orders-history-empty | mobile-vendor | errors-empty | app/(tabs)/orders.tsx | chef | 4 | No order history yet | — | 2026-05-12 | History tab empty state. |
| mv-ordercard-accept | mobile-vendor | core-ux | components/vendor/OrderCard.tsx | chef | 1 | Accept | — | 2026-05-12 | Primary action on order card; also iOS notification action button title (_layout.tsx line 66). |
| mv-ordercard-reject | mobile-vendor | core-ux | components/vendor/OrderCard.tsx | chef | 1 | Reject | — | 2026-05-12 | Secondary action; also iOS notification action (_layout.tsx line 73). |
| mv-undo-accepted | mobile-vendor | transactional | components/vendor/UndoSnackbar.tsx | chef | 2 | Order accepted | — | 2026-05-12 | Snackbar after accept action. |
| mv-undo-rejected | mobile-vendor | transactional | components/vendor/UndoSnackbar.tsx | chef | 2 | Order rejected | — | 2026-05-12 | Snackbar after reject action. |
| mv-undo-cta | mobile-vendor | core-ux | components/vendor/UndoSnackbar.tsx | chef | 1 | UNDO | — | 2026-05-12 | Snackbar action. ALL CAPS — flag against brand "no ALL CAPS body" guidance, though acceptable as button microcopy. |
| mv-push-channel-neworders | mobile-vendor | microcopy | app/_layout.tsx | chef | 2 | New Orders | — | 2026-05-12 | Android notification channel name (visible in OS settings). |
| mv-push-channel-orderupdates | mobile-vendor | microcopy | app/_layout.tsx | chef | 2 | Order Updates | — | 2026-05-12 | Android notification channel name. |
| mv-menu-heading | mobile-vendor | core-ux | app/(tabs)/menu.tsx | chef | 2 | My Menu | — | 2026-05-12 | Tab screen heading. |
| mv-menu-category-all | mobile-vendor | core-ux | app/(tabs)/menu.tsx | chef | 1 | All | — | 2026-05-12 | Default filter chip. |
| mv-menu-load-error | mobile-vendor | errors-empty | app/(tabs)/menu.tsx | chef | 3 | Failed to load menu | — | 2026-05-12 | Error state. |
| mv-menu-empty | mobile-vendor | errors-empty | app/(tabs)/menu.tsx | chef | 8 | No menu items yet. Tap + to add your first item. | — | 2026-05-12 | Empty state. |
| mv-menuitem-available | mobile-vendor | core-ux | components/vendor/MenuItemCard.tsx | chef | 1 | Available | — | 2026-05-12 | Toggle state label. |
| mv-menuitem-unavailable | mobile-vendor | core-ux | components/vendor/MenuItemCard.tsx | chef | 1 | Unavailable | — | 2026-05-12 | Toggle state label. |
| mv-menuitem-delete-alert | mobile-vendor | errors-empty | components/vendor/MenuItemCard.tsx | chef | 9 | Are you sure you want to delete this menu item? | — | 2026-05-12 | Delete confirmation body. Title: "Delete Item". |
| mv-menunew-heading | mobile-vendor | core-ux | app/menu/new.tsx | chef | 3 | Add Menu Item | — | 2026-05-12 | Screen heading. |
| mv-menunew-photo-section | mobile-vendor | core-ux | app/menu/new.tsx | chef | 2 | Food Photo | — | 2026-05-12 | Section label. |
| mv-menunew-photo-empty | mobile-vendor | core-ux | app/menu/new.tsx | chef | 3 | No photo selected | — | 2026-05-12 | Empty preview text. |
| mv-menunew-photo-take | mobile-vendor | core-ux | app/menu/new.tsx | chef | 2 | Take Photo | — | 2026-05-12 | Action. |
| mv-menunew-photo-gallery | mobile-vendor | core-ux | app/menu/new.tsx | chef | 1 | Gallery | — | 2026-05-12 | Action. |
| mv-menunew-section-details | mobile-vendor | core-ux | app/menu/new.tsx | chef | 2 | Item Details | — | 2026-05-12 | Section label (also on edit). |
| mv-menunew-name-label | mobile-vendor | core-ux | app/menu/new.tsx | chef | 3 | Item Name * | — | 2026-05-12 | Field label. |
| mv-menunew-name-ph | mobile-vendor | core-ux | app/menu/new.tsx | chef | 3 | e.g. Butter Chicken | — | 2026-05-12 | India-centric example. |
| mv-menunew-name-err | mobile-vendor | errors-empty | app/menu/new.tsx | chef | 7 | Name must be at least 3 characters | — | 2026-05-12 | Validation. |
| mv-menunew-desc-ph | mobile-vendor | core-ux | app/menu/new.tsx | chef | 6 | Describe your dish (at least 20 characters) | — | 2026-05-12 | Placeholder. |
| mv-menunew-desc-err | mobile-vendor | errors-empty | app/menu/new.tsx | chef | 7 | Description must be at least 20 characters | — | 2026-05-12 | Validation. |
| mv-menunew-price-label | mobile-vendor | core-ux | app/menu/new.tsx | chef | 3 | Price (₹) * | — | 2026-05-12 | Field label. India rupee. |
| mv-menunew-price-required | mobile-vendor | errors-empty | app/menu/new.tsx | chef | 3 | Price is required | — | 2026-05-12 | Validation. |
| mv-menunew-price-range | mobile-vendor | errors-empty | app/menu/new.tsx | chef | 8 | Price must be between ₹1 and ₹10,000 | — | 2026-05-12 | Validation. |
| mv-menunew-category-err | mobile-vendor | errors-empty | app/menu/new.tsx | chef | 4 | Please select a category | — | 2026-05-12 | Validation. |
| mv-menunew-type-veg | mobile-vendor | core-ux | app/menu/new.tsx | chef | 1 | Veg | — | 2026-05-12 | Toggle option. India-specific veg/non-veg. |
| mv-menunew-type-nonveg | mobile-vendor | core-ux | app/menu/new.tsx | chef | 1 | Non-Veg | — | 2026-05-12 | Toggle option. |
| mv-menunew-preptime-label | mobile-vendor | core-ux | app/menu/new.tsx | chef | 2 | Preparation Time | — | 2026-05-12 | Section label. |
| mv-menunew-preptime-unit | mobile-vendor | core-ux | app/menu/new.tsx | chef | 2 | {n} min | — | 2026-05-12 | Per-chip unit. Drift: onboarding/operations.tsx uses "{n}min" no space. |
| mv-menunew-cta | mobile-vendor | core-ux | app/menu/new.tsx | chef | 2 | Add Item | — | 2026-05-12 | Primary CTA. |
| mv-menunew-create-fail | mobile-vendor | errors-empty | app/menu/new.tsx | chef | 6 | Failed to create menu item. Please try again. | — | 2026-05-12 | Error alert body. |
| mv-menuedit-heading | mobile-vendor | core-ux | app/menu/[itemId]/edit.tsx | chef | 3 | Edit Menu Item | — | 2026-05-12 | Screen heading. |
| mv-menuedit-photos-section | mobile-vendor | core-ux | app/menu/[itemId]/edit.tsx | chef | 1 | Photos | — | 2026-05-12 | Section label. |
| mv-menuedit-uploading-progress | mobile-vendor | transactional | app/menu/[itemId]/edit.tsx | chef | 2 | Uploading photo... | — | 2026-05-12 | Inline progress text. |
| mv-menuedit-price-change-banner | mobile-vendor | microcopy | app/menu/[itemId]/edit.tsx | chef | 14 | Price changes are submitted for admin review and may take 24 hours to reflect. | — | 2026-05-12 | Inline warning banner; long for glanceable mobile — flag for TW. Important business rule chefs must understand. |
| mv-menuedit-delete-photo-alert | mobile-vendor | errors-empty | app/menu/[itemId]/edit.tsx | chef | 3 | Remove this photo? | — | 2026-05-12 | Confirm body. Title: "Delete Photo". |
| mv-menuedit-delete-photo-fail | mobile-vendor | errors-empty | app/menu/[itemId]/edit.tsx | chef | 3 | Failed to delete photo. | — | 2026-05-12 | Error body. |
| mv-menuedit-photo-upload-fail | mobile-vendor | errors-empty | app/menu/[itemId]/edit.tsx | chef | 6 | Failed to upload photo. Please try again. | — | 2026-05-12 | Error body. |
| mv-menuedit-save-cta | mobile-vendor | core-ux | app/menu/[itemId]/edit.tsx | chef | 2 | Save Changes | — | 2026-05-12 | Primary CTA. |
| mv-menuedit-update-fail | mobile-vendor | errors-empty | app/menu/[itemId]/edit.tsx | chef | 7 | Failed to update menu item. Please try again. | — | 2026-05-12 | Error alert body. |
| mv-more-account-title | mobile-vendor | core-ux | app/(tabs)/more.tsx | chef | 2 | My Account | — | 2026-05-12 | Header text. |
| mv-more-nav-profile | mobile-vendor | core-ux | app/(tabs)/more.tsx | chef | 1 | Profile | — | 2026-05-12 | Nav item. |
| mv-more-nav-earnings | mobile-vendor | core-ux | app/(tabs)/more.tsx | chef | 1 | Earnings | — | 2026-05-12 | Nav item. |
| mv-more-nav-analytics | mobile-vendor | core-ux | app/(tabs)/more.tsx | chef | 1 | Analytics | — | 2026-05-12 | Nav item. |
| mv-more-nav-reviews | mobile-vendor | core-ux | app/(tabs)/more.tsx | chef | 1 | Reviews | — | 2026-05-12 | Nav item. |
| mv-more-nav-settings | mobile-vendor | core-ux | app/(tabs)/more.tsx | chef | 1 | Settings | — | 2026-05-12 | Nav item. |
| mv-more-logout-row | mobile-vendor | core-ux | app/(tabs)/more.tsx | chef | 1 | Logout | — | 2026-05-12 | Destructive action row. |
| mv-more-logout-confirm-body | mobile-vendor | errors-empty | app/(tabs)/more.tsx | chef | 7 | Are you sure you want to log out? | — | 2026-05-12 | Confirm body. Title: "Logout". |
| mv-more-logout-confirm-cancel | mobile-vendor | core-ux | app/(tabs)/more.tsx | chef | 1 | Cancel | — | 2026-05-12 | Confirm action (reused across all confirm dialogs). |
| mv-analytics-heading | mobile-vendor | core-ux | app/analytics.tsx | chef | 1 | Analytics | — | 2026-05-12 | Screen heading. |
| mv-analytics-period-week | mobile-vendor | core-ux | app/analytics.tsx | chef | 1 | Week | — | 2026-05-12 | Period tab. |
| mv-analytics-period-month | mobile-vendor | core-ux | app/analytics.tsx | chef | 1 | Month | — | 2026-05-12 | Period tab. |
| mv-analytics-period-year | mobile-vendor | core-ux | app/analytics.tsx | chef | 1 | Year | — | 2026-05-12 | Period tab. |
| mv-analytics-total-orders | mobile-vendor | core-ux | app/analytics.tsx | chef | 2 | Total Orders | — | 2026-05-12 | Summary card. |
| mv-analytics-total-revenue | mobile-vendor | core-ux | app/analytics.tsx | chef | 2 | Total Revenue | — | 2026-05-12 | Summary card. Drift: earnings.tsx uses "Total Earnings" — verify mental model is consistent. |
| mv-analytics-popular | mobile-vendor | core-ux | app/analytics.tsx | chef | 2 | Popular Items | — | 2026-05-12 | Section label. |
| mv-analytics-orders-suffix | mobile-vendor | core-ux | app/analytics.tsx | chef | 2 | {n} orders | — | 2026-05-12 | Inline numerals. |
| mv-analytics-revenue-trend | mobile-vendor | core-ux | app/analytics.tsx | chef | 2 | Revenue Trend | — | 2026-05-12 | Chart section label. |
| mv-analytics-load-error | mobile-vendor | errors-empty | app/analytics.tsx | chef | 4 | Failed to load analytics | — | 2026-05-12 | Error state. |
| mv-earnings-heading | mobile-vendor | core-ux | app/earnings.tsx | chef | 1 | Earnings | — | 2026-05-12 | Screen heading. |
| mv-earnings-total | mobile-vendor | core-ux | app/earnings.tsx | chef | 2 | Total Earnings | — | 2026-05-12 | Summary card. |
| mv-earnings-pending-payout | mobile-vendor | core-ux | app/earnings.tsx | chef | 2 | Pending Payout | — | 2026-05-12 | Summary card. |
| mv-earnings-last-payout | mobile-vendor | core-ux | app/earnings.tsx | chef | 2 | Last Payout | — | 2026-05-12 | Section label. |
| mv-earnings-payout-account | mobile-vendor | core-ux | app/earnings.tsx | chef | 2 | Payout Account | — | 2026-05-12 | Section label. |
| mv-earnings-payout-fields | mobile-vendor | core-ux | app/earnings.tsx | chef | 6 | Bank / Account Number / IFSC / UPI ID | — | 2026-05-12 | Field labels. India-specific (IFSC, UPI). |
| mv-earnings-no-payout-account | mobile-vendor | errors-empty | app/earnings.tsx | chef | 4 | No payout account configured | — | 2026-05-12 | Empty state. |
| mv-earnings-weekly | mobile-vendor | core-ux | app/earnings.tsx | chef | 2 | Weekly Earnings | — | 2026-05-12 | Chart section. |
| mv-earnings-load-error | mobile-vendor | errors-empty | app/earnings.tsx | chef | 4 | Failed to load earnings | — | 2026-05-12 | Error state. |
| mv-profile-heading | mobile-vendor | core-ux | app/profile.tsx | chef | 1 | Profile | — | 2026-05-12 | Screen heading. |
| mv-profile-section-personal | mobile-vendor | core-ux | app/profile.tsx | chef | 2 | Personal Info | — | 2026-05-12 | Section label. |
| mv-profile-edit-cta | mobile-vendor | core-ux | app/profile.tsx | chef | 1 | Edit | — | 2026-05-12 | Inline action. |
| mv-profile-field-display-name | mobile-vendor | core-ux | app/profile.tsx | chef | 2 | Display Name | — | 2026-05-12 | Field label. |
| mv-profile-field-bio | mobile-vendor | core-ux | app/profile.tsx | chef | 1 | Bio | — | 2026-05-12 | Field label. |
| mv-profile-field-phone | mobile-vendor | core-ux | app/profile.tsx | chef | 1 | Phone | — | 2026-05-12 | Field label. |
| mv-profile-save | mobile-vendor | core-ux | app/profile.tsx | chef | 1 | Save | — | 2026-05-12 | Primary CTA in edit mode. |
| mv-profile-section-kitchen | mobile-vendor | core-ux | app/profile.tsx | chef | 2 | Kitchen Info | — | 2026-05-12 | Section label. |
| mv-profile-field-kitchen-name | mobile-vendor | core-ux | app/profile.tsx | chef | 2 | Kitchen Name | — | 2026-05-12 | Field label. |
| mv-profile-field-cuisine-types | mobile-vendor | core-ux | app/profile.tsx | chef | 2 | Cuisine Types | — | 2026-05-12 | Field label. |
| mv-profile-section-photos | mobile-vendor | core-ux | app/profile.tsx | chef | 2 | Kitchen Photos | — | 2026-05-12 | Section label. |
| mv-profile-displayname-required | mobile-vendor | errors-empty | app/profile.tsx | chef | 4 | Display name is required. | — | 2026-05-12 | Alert body. Title: "Validation". |
| mv-profile-update-success-body | mobile-vendor | transactional | app/profile.tsx | chef | 3 | Profile updated successfully. | — | 2026-05-12 | Success alert body. Title: "Success". |
| mv-profile-update-fail | mobile-vendor | errors-empty | app/profile.tsx | chef | 7 | Failed to update profile. Please try again. | — | 2026-05-12 | Error alert body. |
| mv-profile-load-error | mobile-vendor | errors-empty | app/profile.tsx | chef | 3 | Failed to load profile | — | 2026-05-12 | Error state. |
| mv-profile-photo-upload-fail | mobile-vendor | errors-empty | app/profile.tsx | chef | 4 | Failed to upload profile photo. | — | 2026-05-12 | Alert body. |
| mv-profile-kitchenphoto-upload-fail | mobile-vendor | errors-empty | app/profile.tsx | chef | 4 | Failed to upload kitchen photo. | — | 2026-05-12 | Alert body. |
| mv-reviews-heading | mobile-vendor | core-ux | app/reviews.tsx | chef | 2 | Customer Reviews | — | 2026-05-12 | Screen heading. |
| mv-reviews-count-suffix | mobile-vendor | core-ux | app/reviews.tsx | chef | 2 | {n} reviews | — | 2026-05-12 | Numeral suffix. |
| mv-reviewcard-your-reply | mobile-vendor | core-ux | app/reviews.tsx | chef | 2 | Your Reply | — | 2026-05-12 | Reply block label. |
| mv-reviewcard-reply-cta | mobile-vendor | core-ux | app/reviews.tsx | chef | 1 | Reply | — | 2026-05-12 | Action button. |
| mv-reviews-load-error | mobile-vendor | errors-empty | app/reviews.tsx | chef | 3 | Failed to load reviews | — | 2026-05-12 | Error state. |
| mv-reviews-empty | mobile-vendor | errors-empty | app/reviews.tsx | chef | 9 | No reviews yet. Your first review will appear here. | — | 2026-05-12 | Empty state. |
| mv-reviewdetail-heading | mobile-vendor | core-ux | app/review/[reviewId].tsx | chef | 3 | Reply to Review | — | 2026-05-12 | Screen heading. |
| mv-reviewdetail-form-label | mobile-vendor | core-ux | app/review/[reviewId].tsx | chef | 2 | Your Reply | — | 2026-05-12 | Section label. |
| mv-reviewdetail-input-ph | mobile-vendor | core-ux | app/review/[reviewId].tsx | chef | 7 | Write a thoughtful reply to this review... | — | 2026-05-12 | Placeholder. |
| mv-reviewdetail-min-err | mobile-vendor | errors-empty | app/review/[reviewId].tsx | chef | 6 | Reply must be at least 10 characters | — | 2026-05-12 | Validation. |
| mv-reviewdetail-min-fallback | mobile-vendor | errors-empty | app/review/[reviewId].tsx | chef | 4 | Reply is too short | — | 2026-05-12 | Fallback validation. |
| mv-reviewdetail-success-title | mobile-vendor | transactional | app/review/[reviewId].tsx | chef | 2 | Reply Sent | — | 2026-05-12 | Alert title. |
| mv-reviewdetail-success-body | mobile-vendor | transactional | app/review/[reviewId].tsx | chef | 5 | Your reply has been posted. | — | 2026-05-12 | Alert body. |
| mv-reviewdetail-error-body | mobile-vendor | errors-empty | app/review/[reviewId].tsx | chef | 6 | Failed to send reply. Please try again. | — | 2026-05-12 | Alert body. |
| mv-reviewdetail-send-cta | mobile-vendor | core-ux | app/review/[reviewId].tsx | chef | 2 | Send Reply | — | 2026-05-12 | Primary CTA. |
| mv-settings-heading | mobile-vendor | core-ux | app/settings.tsx | chef | 1 | Settings | — | 2026-05-12 | Screen heading. |
| mv-settings-section-notifs | mobile-vendor | core-ux | app/settings.tsx | chef | 2 | Notification Preferences | — | 2026-05-12 | Section label. |
| mv-settings-notif-neworders | mobile-vendor | core-ux | app/settings.tsx | chef | 3 | New Order Notifications | — | 2026-05-12 | Toggle label. |
| mv-settings-notif-neworders-helper | mobile-vendor | microcopy | app/settings.tsx | chef | 7 | Get notified when a new order arrives | — | 2026-05-12 | Helper text. |
| mv-settings-notif-payouts | mobile-vendor | core-ux | app/settings.tsx | chef | 2 | Payout Notifications | — | 2026-05-12 | Toggle label. |
| mv-settings-notif-payouts-helper | mobile-vendor | microcopy | app/settings.tsx | chef | 5 | Get notified when payouts are processed | — | 2026-05-12 | Helper text. |
| mv-settings-notif-reviews | mobile-vendor | core-ux | app/settings.tsx | chef | 2 | Review Notifications | — | 2026-05-12 | Toggle label. |
| mv-settings-notif-reviews-helper | mobile-vendor | microcopy | app/settings.tsx | chef | 6 | Get notified when customers leave reviews | — | 2026-05-12 | Helper text. |
| mv-settings-section-availability | mobile-vendor | core-ux | app/settings.tsx | chef | 1 | Availability | — | 2026-05-12 | Section label. |
| mv-settings-accepting-helper | mobile-vendor | microcopy | app/settings.tsx | chef | 7 | Toggle to start or pause accepting orders | — | 2026-05-12 | Helper text. |
| mv-settings-section-account | mobile-vendor | core-ux | app/settings.tsx | chef | 1 | Account | — | 2026-05-12 | Section label. |
| mv-settings-change-password | mobile-vendor | core-ux | app/settings.tsx | chef | 2 | Change Password | — | 2026-05-12 | Nav action — routes back to forgot-password screen (mismatch: label says "change", action triggers reset flow). Flag for TW lens. |
| mv-settings-delete-account | mobile-vendor | core-ux | app/settings.tsx | chef | 2 | Delete Account | — | 2026-05-12 | Destructive action. |
| mv-settings-delete-confirm-body | mobile-vendor | legal | app/settings.tsx | chef | 16 | To delete your account, please contact our support team. This action cannot be undone. | — | 2026-05-12 | Confirm alert body. |
| mv-settings-delete-contact-action | mobile-vendor | legal | app/settings.tsx | chef | 2 | Contact Support | — | 2026-05-12 | Alert action button. |
| mv-settings-delete-contact-email | mobile-vendor | legal | app/settings.tsx | chef | 8 | Please email support@homechef.app to delete your account. | — | 2026-05-12 | Follow-up alert body. Hardcoded email — verify against domain registration. |
| mv-settings-load-error | mobile-vendor | errors-empty | app/settings.tsx | chef | 4 | Failed to load settings | — | 2026-05-12 | Error state. |
