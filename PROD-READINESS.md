# Home Chef Vendor App — Production Readiness Plan

**Target:** Open public launch in 8 weeks (start: 2026-06-05, ship: ~2026-07-31)
**Execution:** Solo dev (Mahesh) + Claude subagents
**Scope:** vendor-mobile + homechef-api + supporting infra. Customer + delivery apps tracked separately.

## Progress snapshot (last updated 2026-06-10)

| Wave | Application code | Operational |
|---|---|---|
| Wave 1 (Foundation) | **✅ 100%** — Sentry + force-upgrade gate + rate-limit + idempotency + webhook HMAC + Dependabot (88→28 vulns) + base-image Go 1.26.4 stopgap | ⏳ TestFlight, APNs cert, Sentry DSN secrets, Cloudflare WAF, Cloud SQL drill |
| Wave 2 (Critical workflows) | **✅ 100% backend, 11/13 mobile** — cancel flows, doc renewal, notif prefs + FCM topics, info_requested email, FSSAI cron, FSSAI license number + expiry inputs | ⏳ Pause-receiving (needs new scheduler), FoSCoS API access |
| Wave 3 (Financial + tax) | **✅ 100% in-scope (9/9 backend, 6/6 mobile)** — GSTIN + HSN, customer invoice PDF, post-delivery refund, auto-email invoice, DPDP export/delete, **weekly statements + cron, TDS Form 16A, refund history, settlement reconciliation** | ⏳ Privacy policy + EULA URLs (legal) |
| Wave 4 (Launch polish + scale) | **0%** — next up: observability trio (OTel + structured logging + audit log) | — |

**41 commits on `main`** since 2026-06-05 (4 this session closing Wave 3). Check `git log --oneline` for the full trail.

---

## Assumptions

- India-only launch. Multi-region deferred to v2.
- GKE Autopilot infra stays as-is. No platform migration.
- Cloud SQL `db-f1-micro` stays. No HA/replicas until traffic justifies (≥100 chefs).
- Sentry (free tier 5k errors/mo) for crash reporting on both mobile + backend.
- Will switch ONE service to `minScale: 1` (auth-bff) — others stay scale-to-zero.
- Trivy gate flips from warn-only to fail-on-critical in Wave 1.
- 4 waves × 2 weeks each. Each wave ships independently and is testable end-to-end.

## Locked deferrals (v2, NOT in scope)

- Customer + delivery apps (separate tracks)
- Multi-staff / sub-chef permissions
- Multi-kitchen locations
- Menu customizations + modifiers + combos
- Promo / discount codes
- Loyalty / referral program
- Chef community forum
- Time-windowed item availability
- Read replicas / multi-region
- E-invoicing (>₹5cr turnover) — none of our beta chefs will hit threshold

## Risks (and what eats which wave)

| Risk | Eats |
|---|---|
| Razorpay account onboarding for chefs blocked / slow | W3 |
| FoSCoS API integration delay | W2 |
| App Store review rejection (privacy, payment policy) | W4 |
| Cloud SQL connection-pool exhaustion at scale | needs W4 mitigation |
| GIP rate limits during onboarding burst | W1 |

---

## Wave 1 — Foundation (weeks 1–2)

**Goal:** Cannot ship without these. Make the system observable, securable, distributable.

### Mobile
- [ ] **TestFlight pipeline** — App Store Connect setup, distribution cert, `eas submit` flow, internal testing group _(blocked: Apple Developer enrollment)_
- [x] **Sentry** integration (`@sentry/react-native` + Expo plugin), source-maps upload on EAS build _(scaffold shipped — needs `EAS_SECRET_SENTRY_DSN/AUTH_TOKEN/ORG` set)_
- [x] **Force-upgrade gate** — `min_version` from API, hard wall + App Store link (`/upgrade-required` screen + axios 426 handler + useMinVersion polling)
- [x] **Privacy manifests** (iOS 17+) — `app.json` `ios.privacyManifests` declares 4 API categories + 7 data types
- [ ] **APNs production cert** in Firebase + EAS credentials _(blocked: Firebase + Apple Dev console access)_
- [x] **Bundle ID + signing certs** finalized for production profile in EAS _(eas.json `submit.production.ios` configured; cert upload pending)_

### Backend
- [x] **Sentry-go** integration on `homechef-api` (gated on `SENTRY_DSN_API`; panics + 5xx capture, 10% trace sample) _(auth-bff same recipe pending)_
- [x] **Rate limiting middleware** — Redis token-bucket, per chef (auth user_id) + per IP. `60req/min` chef, `30req/min` unauth. Fail-open on Redis down. X-RateLimit-* headers.
- [x] **Idempotency-Key** middleware on POST/PUT/PATCH mutations — Redis-backed, 24h TTL, opt-in via header
- [x] **Razorpay webhook signature verification** — audit complete (Razorpay+Stripe were already verified; delivery provider webhook HMAC added)
- [x] **Dependabot triage** — partial: 88 → 28 vulns (60-vuln reduction via pnpm overrides + x/net bump); 6 Go stdlib remain pending base-image rebuild
- [ ] **Trivy gate flip** — from warn-only to fail on critical+high once Dependabot is clean _(blocked: 6 remaining critical/high)_

### Infra
- [ ] **Cloud SQL automated backups + restore drill** — verify retention policy, run a non-prod restore, document RTO/RPO _(operational)_
- [x] **`min_scale: 1` on auth-bff** — confirmed already satisfied (auth-bff is Deployment+KEDA at 2/2 replicas, not Knative; flagged in `docs/superpowers/specs/2026-06-05-wave1-infra-design.md` §8)
- [ ] **Cloudflare WAF rules** — managed ruleset on `*.fe3dr.com`, custom rules for `/auth/auto-login` brute force _(operational; Pro plan needed)_

### Definition of done
- A real device installs via TestFlight (single chef test)
- Any panic in either Go service shows up in Sentry within 30s
- A 70-req/min flood from one IP gets HTTP 429 after the first 60
- `gh run list` shows no failing builds; Trivy gate is fail-on-critical
- Cold-start 503 on auth-bff is reproducibly gone (load test)

### Parallelization
- Subagent A (mobile): TestFlight + Sentry + force-upgrade + privacy manifests
- Subagent B (backend): Sentry + rate limit + idempotency + webhook verification
- Subagent C (deps): Dependabot sweep + Trivy gate
- Infra tasks (Cloud SQL + min_scale + WAF) — sequential, by hand, ~half-day

---

## Wave 2 — Critical chef workflows (weeks 3–4)

**Goal:** A real chef can run a full week of orders without needing manual ops intervention.

### Mobile
- [x] **FSSAI expiry date picker** on docs step (onboarding) — TextInput w/ YYYY-MM-DD format validation, sent as `expiryDate` multipart field on FSSAI upload
- [x] **Doc renewal screen** for verified chefs (`/documents/renew`) — list docs with expiry status, swap file (camera/gallery/PDF) → `POST /chef/documents/:id/replace`
- [x] **Admin-requests inbox** — dashboard "ACTION REQUIRED" card + `/admin-requests` list with status badges + `/admin-requests/[id]` detail
- [x] **Approval-request response flow** — chef replies with text + optional file attach; sends to `PUT /chef/admin-requests/:id/respond`
- [x] **Order partial fulfillment + cancellation** — TWO affordances on the accepted-order detail screen:
  - [x] **Per-line "Can't fulfill this item"** — backend partial-refunds the line + atomically recomputes order totals; mobile renders cancelled lines with strikethrough + "Refunded ₹X" badge
  - [x] **Whole-order cancel** with reason picker (iOS ActionSheet + Android chained Alert)
- [ ] **Pause receiving with auto-resume** — replace binary Open/Closed with `Open / Closed / Back in {15,30,60} min` _(deferred — needs new backend auto-resume scheduler)_
- [x] **Menu item image upload UI** — already wired pre-session via `MenuItemForm.handlePickPhoto` → `useUploadMenuPhoto` → `POST /chef/menu/items/:id/images`
- [x] **Reviews list / inbox** — `app/reviews.tsx` with adapter fix (was crashing on wire shape) + `/chef/reviews/summary` for averageRating
- [x] **Notification preferences screen** — `app/notification-preferences.tsx` with toggles + quiet hours + optimistic save; backend FCM topic reconcile on each flip

### Bonus (not in original spec)
- [x] **FSSAI license number capture** — 14-digit TextInput in onboarding, persisted to new `chef_profiles.fssai_license_number` column (unblocks Wave 3 invoicing + future FoSCoS lookup)
- [x] **More tab nav** — discoverable entries for admin-requests, documents/renew, notification-preferences

### Backend
- [x] **`POST /chef/orders/:id/cancel`** with reason enum — full Razorpay refund + status → `cancelled` + NATS `order.cancelled` event. Idempotent on retry.
- [x] **`POST /chef/orders/:id/items/:itemId/cancel`** with reason — partial Razorpay refund (line subtotal + proportional tax share); atomic GORM transaction for line flip + order recompute; order status stays accepted/preparing
- [x] **`is_cancelled` + `cancelled_reason` + `refund_id` + `cancelled_at` + `refund_amount` columns** on `order_items` via AutoMigrate
- [x] **Razorpay partial refund** — `services.GetRazorpay().CreateRefund` already existed; wired into both cancel paths
- [x] **`POST /chef/documents/:id/replace`** — atomic file URL + (optional) expiry + status flip to `pending`; cancels in-flight approval and creates a fresh one
- [x] **`GET /chef/notification-preferences` + `PUT`** — new `chef_notification_preferences` table; defaults applied when no row exists
- [x] **FCM topic subscriptions** wired to preferences — `services.SubscribeToFCMTopic` / `UnsubscribeFromFCMTopic` via Firebase IID API; reconciles on each PUT diff
- [x] **Email notification on info_requested approval** — `EmailService.SendApprovalInfoRequested` fires inline with the existing NATS event publish
- [x] **FSSAI expiry cron** — `services.StartFSSAIReminderCron` launched from `main.go`; 24h ticker, Redis-deduped via SETNX (26h TTL), fail-open on Redis down; fires push at 30/15/7 day windows

### Compliance
- [ ] **FSSAI validation** — `FoSCoS` API integration _(external dep — manual review queue is the fallback per risk callout)_

### Definition of done
- A chef whose FSSAI expires next month gets a banner + push at 30 days
- A chef can re-upload an expired doc and the system flips them back to `pending_review`
- A chef cancels an accepted order and the customer's full refund is in-flight within 10s
- A chef marks 1 of 2 items as "can't fulfill" → that line's refund is in-flight within 10s, order subtotal/tax/total recompute correctly, the other item continues prep
- A chef who toggles "no promo pings" stops getting promo pushes within one cycle

### Parallelization
- Subagent A (mobile): FSSAI date picker + doc renewal + admin inbox + approval response
- Subagent B (mobile): order cancellation + pause auto-resume + image upload + reviews list + notif prefs
- Subagent C (backend): all 5 backend tickets + cron job
- Compliance is single-thread (FoSCoS integration is a research + paperwork lift)

---

## Wave 3 — Financial + tax (weeks 5–6)

**Goal:** A chef can run their accounting for the quarter from inside the app. Legally sufficient invoices to customers.

### Mobile
- [x] **GSTIN capture** on chef profile (optional) — 15-char auto-uppercase TextInput in onboarding, sent to `/chef/onboarding`
- [x] **HSN code per menu item** — 8-digit numeric TextInput in MenuItemForm, defaults to 996331 server-side when blank
- [x] **Earnings → Statements section** — lists issued weekly statements, tap → downloads settlement PDF (shared `lib/download-pdf` helper)
- [x] **Earnings → Tax certificates** — TDS-certificate download row (current FY) on Earnings
- [x] **Order detail → Download invoice (PDF)** — `expo-file-system/legacy` + `expo-sharing`; auth'd fetch → system share sheet; surfaced on delivered orders
- [x] **Customer invoice auto-email** — backend hooks `order.delivered` NATS event, generates PDF, sends SendGrid attachment to customer
- [x] **Refund history view** on Earnings — Refunds section, one entry per refunded order (item breakdown), tap → opens order

### Backend
- [x] **`GET /chef/statements/weekly`** — lists immutable `WeeklyStatement` rows; `GET /chef/statements/:id/statement.pdf` streams the maroto settlement PDF
- [x] **`GET /chef/tax/certificate?year=FY`** — annual §194-O TDS summary (Form 16A style, quarterly breakdown, PAN/deductor block, TRACES disclaimer)
- [x] **`GET /orders/:id/invoice.pdf`** — customer-facing GSTIN invoice via `services.GenerateOrderInvoicePDF` (maroto v0.46)
- [x] **`GET /chef/orders/:orderId/invoice.pdf`** — chef-side copy of the same invoice _(bonus)_
- [x] **`POST /chef/orders/:id/refund`** with amount + reason — post-delivery goodwill refund (distinct from in-flight cancel); idempotent via remaining-balance check
- [x] **`GET /chef/refunds`** — refund history (authoritative `Order.RefundAmount`, no double-count vs per-line)
- [x] **GSTIN + HSN fields** on `chef_profiles` and `menu_items` respectively
- [x] **Weekly statement generation cron** — daily Mon–Sun IST close → immutable statement + push; idempotent (Redis SETNX + unique index); shared earnings math extracted to `services` with unit tests

### Compliance
- [x] **DPDP Act 2023** compliance pass — `GET /chef/me/export` (JSON dump of all chef data) + `POST /chef/me/delete` (soft-delete + 30d retention, requires email confirmation)
- [ ] **Privacy policy + EULA URLs** finalized + linked from app _(legal artifact)_
- [x] **Stripe/Razorpay settlement reconciliation** — daily cron cross-checks previous IST day's payments vs gateways (capture status + refund-amount drift), read-only, ERROR log + Sentry alert

### Definition of done
- Chef downloads weekly statement → opens cleanly in any PDF reader, line-itemized
- Customer receives a GSTIN-formatted invoice email within 5 min of delivery
- A test refund is reflected in chef's earnings within 1 hour
- `/chef/me/export` returns a zip of all chef data (DPDP requirement)

### Parallelization
- Subagent A (mobile): GSTIN/HSN capture + statements tab + tax certs + invoice download
- Subagent B (backend): PDF generation endpoints (3 of them) + refund flow + cycle close cron
- Compliance is sequential (legal review needed on DPDP + privacy policy)

---

## Wave 4 — Launch polish + scale (weeks 7–8)

**Goal:** Smooth public launch. App Store submission. India-vernacular reach. Observability for the support pager.

### Mobile
- [ ] **i18n setup** — `i18n-js` or `react-i18next`, Hindi locale at minimum
- [ ] **Hindi translation** of all chef-facing strings (use a service: Translated.com, Smartling, or human translator)
- [ ] **Locale picker** in More tab — English, Hindi, with auto-detection from device locale
- [ ] **Deep links** — `homechef://orders/<id>` registered + tested for push tap-through across 4 push categories
- [ ] **OTA updates** via EAS Update — channel `production`, runtimeVersion locked
- [ ] **Bundle size audit** — Hermes verify, dead-code elimination, asset compression
- [ ] **Accessibility pass** — VoiceOver across every screen, contrast audit, dynamic type lock-down where appropriate
- [ ] **App Store screenshots + listing** for IN store
- [ ] **App Store submission + review wait** — submit by week 7 Friday for week 8 review

### Backend
- [ ] **OpenTelemetry tracing** — instrument `homechef-api` + `auth-bff`, export to Cloud Trace (free up to 25M/mo)
- [ ] **Structured logging with correlation IDs** — converge slog (some services use logrus today), inject request ID from middleware
- [ ] **Audit log for sensitive mutations** — payout updates, doc downloads, approval responses → separate `audit_logs` table with retention
- [ ] **`min_scale: 1` on homechef-api** (auth-bff was done in W1) — kills cold-start latency for the chef-facing API
- [ ] **Connection pool tuning** — bump from 5 open / 2 idle to 20/5 with proper monitoring
- [ ] **`gomaxprocs`** alignment with Knative CPU limits (Uber automaxprocs lib)

### Launch ops
- [ ] **Runbook** for top-5 incident types — auth-bff down, DB connection pool exhausted, Razorpay outage, GIP outage, Knative scale-from-zero stuck
- [ ] **On-call rotation** (even if solo — define what triggers a page)
- [ ] **Status page** (Statuspage.io free tier or roll-your-own) at `status.fe3dr.com`
- [ ] **First-10-chefs concierge script** — onboard them by hand, watch logs live, fix in real-time

### Definition of done
- App is live on App Store IN region
- Hindi locale renders all chef-facing copy without truncation
- A trace from a chef's tap on "Accept order" reaches the DB write in Cloud Trace
- 10 real chefs onboarded with concierge support, zero critical bugs in 48h
- Runbook is tested (drill the auth-bff-down scenario at least once)

### Parallelization
- Subagent A (mobile): i18n + Hindi + deep links + OTA + accessibility
- Subagent B (mobile): bundle audit + App Store screenshots + submission flow
- Subagent C (backend): OTel + structured logs + audit log + scale tuning
- Launch ops + runbook + concierge — single-thread, by hand

---

## Tracking

This file is the source of truth. Update checkboxes as items ship. New blockers / scope changes get added to the appropriate wave OR explicitly punted to "Locked deferrals" with a one-line reason.

Weekly review: every Friday, walk through the current wave, mark items, surface blockers, re-plan if behind.
