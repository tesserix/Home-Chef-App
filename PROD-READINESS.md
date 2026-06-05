# Home Chef Vendor App — Production Readiness Plan

**Target:** Open public launch in 8 weeks (start: 2026-06-05, ship: ~2026-07-31)
**Execution:** Solo dev (Mahesh) + Claude subagents
**Scope:** vendor-mobile + homechef-api + supporting infra. Customer + delivery apps tracked separately.

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
- [ ] **TestFlight pipeline** — App Store Connect setup, distribution cert, `eas submit` flow, internal testing group
- [ ] **Sentry** integration (`sentry-expo`), source-maps upload on EAS build
- [ ] **Force-upgrade gate** — `min_version` from API, hard wall + App Store link
- [ ] **Privacy manifests** (iOS 17+) — declare camera/photos/location/notifications usage
- [ ] **APNs production cert** in Firebase + EAS credentials
- [ ] **Bundle ID + signing certs** finalized for production profile in EAS

### Backend
- [ ] **Sentry-go** integration on `homechef-api` and `auth-bff` (panics + 5xx capture)
- [ ] **Rate limiting middleware** — Redis token-bucket, per chef (auth user_id) + per IP. `60req/min` chef, `30req/min` unauth.
- [ ] **Idempotency-Key** middleware on POST/PUT/PATCH mutations — Redis-backed, 24h TTL
- [ ] **Razorpay webhook signature verification** — audit all webhook endpoints, add HMAC check
- [ ] **Dependabot triage** — apt-get critical → high → moderate, ship bumps in a single PR each
- [ ] **Trivy gate flip** — from warn-only to fail on critical+high once Dependabot is clean

### Infra
- [ ] **Cloud SQL automated backups + restore drill** — verify retention policy, run a non-prod restore, document RTO/RPO
- [ ] **`min_scale: 1` on auth-bff** — kills the cold-start 503 race that bit us
- [ ] **Cloudflare WAF rules** — managed ruleset on `*.fe3dr.com`, custom rules for `/auth/auto-login` brute force

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
- [ ] **FSSAI expiry date picker** on docs step (onboarding + renewal)
- [ ] **Doc renewal screen** for verified chefs (`/documents/renew`) — list docs with expiry status, swap file + new expiry
- [ ] **Admin-requests inbox** — consume `/chef/admin-requests`, dashboard "ACTION REQUIRED" card per item
- [ ] **Approval-request response flow** — chef can reply to info-requested with text + attach a re-uploaded doc
- [ ] **Order partial fulfillment + cancellation** — TWO affordances on the accepted-order detail screen:
  - **Per-line "Can't fulfill this item"** — mark one line item as unavailable, system auto-refunds that line, remaining items continue prep. Critical UX: chef discovers mid-prep that one of N items can't be made (out of ingredient, ran out).
  - **Whole-order cancel** with reason picker (`out_of_ingredient`, `equipment_failure`, `customer_request`, `other`) — triggers full refund.
- [ ] **Pause receiving with auto-resume** — replace binary Open/Closed with `Open / Closed / Back in {15,30,60} min`
- [ ] **Menu item image upload UI** — `expo-image-picker` → POST `/chef/menu/items/:id/images` (endpoint exists)
- [ ] **Reviews list / inbox** — read endpoint exists, no list view today
- [ ] **Notification preferences screen** — categories (new order / payout / customer message / promo) + quiet hours

### Backend
- [ ] **`POST /chef/orders/:id/cancel`** with reason enum (`out_of_ingredient`, `equipment_failure`, `customer_request`, `other`) — triggers full Razorpay refund + status → `cancelled`
- [ ] **`POST /chef/orders/:id/items/:itemId/cancel`** with reason — partial Razorpay refund equal to that line's subtotal, atomically recomputes order subtotal/tax/total, mark order_item `is_cancelled = true`. Order status stays `accepted`/`preparing` so chef proceeds with the rest.
- [ ] **`is_cancelled` + `cancelled_reason` + `refund_id` + `cancelled_at` columns** on `order_items` — migration via AutoMigrate
- [ ] **Razorpay partial refund** integration in `services/razorpay.go` — wrap the gateway's `/payments/:id/refund` with our domain shape; idempotent on `(order_item.id, refund_id)`
- [ ] **`POST /chef/documents/:id/replace`** — atomically swap file URL + reset expiry + reset status to `pending` for re-verification
- [ ] **`GET /chef/notification-preferences` + `PUT`** — per-chef preference flags
- [ ] **FCM topic subscriptions** wired to preferences — chefs unsubscribe from `promo` channel etc.
- [ ] **Email notification on info_requested approval** (SendGrid) — chef sees the in-app card AND gets an email
- [ ] **FSSAI expiry cron** — daily job in `audit-service` or homechef-api that fires push at 30/15/7 days

### Compliance
- [ ] **FSSAI validation** — `FoSCoS` API integration (or manual review queue if FoSCoS access blocked). License number + expiry verified against the registry, not just photo upload.

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
- [ ] **GSTIN capture** on chef profile (optional but encouraged) + validation against GST portal format
- [ ] **HSN code per menu item** (optional, defaults to 996331 for restaurant services if blank)
- [ ] **Earnings → Statements tab** — list of weekly settlement statements, downloadable as PDF
- [ ] **Earnings → Tax certificates** — TDS year-end certificate download (Form 16A equivalent)
- [ ] **Order detail → Download invoice (PDF)** for the chef
- [ ] **Customer invoice auto-email** on order delivery (with chef's GSTIN + HSN line items)
- [ ] **Refund history view** on Earnings — see what got refunded, who paid, why

### Backend
- [ ] **`GET /chef/statements/weekly?cycle=...`** — returns PDF (or signed URL to GCS) of the weekly settlement
- [ ] **`GET /chef/tax/certificate?year=2026`** — TDS Form 16A PDF generation (use `maroto` lib — already a dep)
- [ ] **`GET /orders/:id/invoice.pdf`** — customer-facing GSTIN invoice
- [ ] **`POST /chef/orders/:id/refund`** with amount + reason — partial or full
- [ ] **GSTIN + HSN fields** on `chef_profiles` and `menu_items` respectively
- [ ] **Cycle close + statement generation cron** — weekly Sunday 23:59 IST, generates statement, sends NATS event

### Compliance
- [ ] **DPDP Act 2023** compliance pass — explicit consent capture on registration, data subject access endpoints (`/chef/me/export`, `/chef/me/delete`), retention policy in privacy policy
- [ ] **Privacy policy + EULA URLs** finalized + linked from app
- [ ] **Stripe/Razorpay settlement reconciliation** — automated daily comparison of platform records vs gateway records

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
