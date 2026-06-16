# Home Chef Vendor App — Production Readiness Plan

**Target:** Open public launch in 8 weeks (start: 2026-06-05, ship: ~2026-07-31)
**Execution:** Solo dev (Mahesh) + Claude subagents
**Scope:** vendor-mobile + homechef-api + supporting infra. **Customer app (Wave 6)**, **delivery via 3PL — own fleet retired (Wave 7)**, and **tesserix-home admin integration (Wave 5B)** are captured below for separate implementation sessions.

## Progress snapshot (last updated 2026-06-11)

| Wave | Application code | Operational |
|---|---|---|
| Wave 1 (Foundation) | **✅ 100%** — Sentry + force-upgrade gate + rate-limit + idempotency + webhook HMAC + Dependabot (88→28 vulns) + base-image Go 1.26.4 stopgap | ⏳ TestFlight, APNs cert, Sentry DSN secrets, Cloudflare WAF, Cloud SQL drill |
| Wave 2 (Critical workflows) | **✅ 100% in-scope (9/9 backend, 12/13 mobile)** — cancel flows, doc renewal, notif prefs + FCM topics, info_requested email, FSSAI cron, FSSAI inputs, **pause-receiving with auto-resume** | ⏳ FoSCoS API access (external) |
| Wave 3 (Financial + tax) | **✅ 100% in-scope (9/9 backend, 6/6 mobile)** — GSTIN + HSN, invoice PDF, refund, auto-email, DPDP, weekly statements + cron, TDS 16A, refund history, settlement reconciliation | ✅ Privacy/Terms/Refund/EULA URLs drafted + linked (2026-06-15); ⏳ counsel sign-off only |
| Wave 4 (Launch polish + scale) | **✅ codeable items done** — OTel→Cloud Trace, structured logging + correlation IDs, audit log, pool tuning, automaxprocs (min-scale:1 already set); deep links, EAS OTA, i18n+Hindi+locale picker; runbook/on-call/status/concierge docs | ⏳ App Store enrollment+submission, full a11y/bundle QA (needs build), auth-bff OTel |

**51 commits on `main`** since 2026-06-05 (~14 this session: Wave 3 close-out + Wave 4 + Wave 2 pause-receiving). Check `git log --oneline` for the full trail.

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

- Customer app — now IN scope, see **Wave 6** (separate session)
- **Own delivery fleet / driver app** — RETIRED. Deliveries go through 3PL providers (Shadowfax-class). See **Wave 7**.
- Multi-staff / sub-chef permissions
- Multi-kitchen locations
- Menu customizations + modifiers + combos
- Promo / discount codes
- Loyalty / referral program
- Chef community forum
- Time-windowed item availability
- Read replicas / multi-region
- E-invoicing (>₹5cr turnover) — none of our beta chefs will hit threshold

## Platform integration — tesserix-home + tickets (planned)

The admin↔vendor loop spans two surfaces; the vendor app holds only the chef-facing half.

- [ ] **Platform-admin approval UI in `tesserix-home`** — admins review/approve/`info_requested` against the existing chef approval API (`/admin/...`, `RequireAdmin`). The vendor app keeps its **Admin Requests** page as the chef's *respond* surface (`GET /chef/admin-requests` + `PUT /chef/admin-requests/:id/respond`) — do NOT remove it; it's the chef side of this flow.
  - **Deferred (Phase B, 2026-06-12):** the read side (list pending) is a clean direct-DB query, but **approve/reject/info_requested have side effects** (chef email + NATS event + FCM) that only fire through homechef-api's approval handler — a direct-DB write would skip them. So this needs either cross-service auth from tesserix-home → homechef-api `/admin` (the documented design), or a homechef-api action endpoint. Left for a follow-up; the vendor-app respond side is unaffected.
- [x] **Vendor support/feature tickets** — chefs raise platform issues + feature requests from the vendor app (2026-06-11). Mobile: `hooks/useSupport.ts` + `app/support/{index,new,[id]}.tsx` + `components/vendor/TicketStatus{Chip,Stepper}.tsx` + More-tab entry, wired to the existing Home Chef `/api/v1/support/tickets` API (4-stage stepper, Uber-monochrome). Full create→reply→close cycle verified live. Admin triage UI lives in tesserix-home (still pending — see Platform integration).
- [x] **Vendor cover image** — `bannerImage` upload in vendor profile (`POST /chef/banner-image`) + customer `ChefCard` hero now prefers the cover (banner → kitchen photo → avatar). Backend + model already supported it; this session added the vendor upload UI + flipped the customer hero ordering.
- [ ] **Chef payout/settlement admin in `tesserix-home`** — admins view + configure **all** chefs' payouts (weekly settlement statements, net payouts, status, commission %, schedule). The **chef-facing half is DONE** in the vendor app (Earnings → Statements + settlement-statement PDF + TDS cert; `GET /chef/statements/weekly`, `/chef/earnings/breakdown`, `/chef/payout`). Admin oversight + actual disbursement land with the tesserix-home batch — see Wave 5B.
- Future (deferred): **OCR** to pre-fill FSSAI license number + expiry on doc re-upload (chef confirms/edits) — see the doc-reupload decision.

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
- [x] **TestFlight pipeline** — **DONE 2026-06-16**: org Apple Developer account (Team `2CRHRRYBPL`), ASC app records (vendor `6780689641` / customer `6780689976`), distribution cert + provisioning profiles, ASC API key stored on EAS → **both apps built (`.ipa`) + submitted to TestFlight** via `eas submit`. Bundle IDs migrated to `com.tesserix.homechef.*` (com.homechef.* taken globally). ⏳ Remaining: add internal testers in App Store Connect to install on-device.
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
- [x] **Pause receiving with auto-resume** — dashboard status pill → Open / Close / Pause {15,30,60} min menu; `PausedUntil` on chef_profiles + `POST /chef/availability/pause|resume` + 1-min auto-resume cron (reopens + push); shows "Back HH:MM"
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
- [~] **Privacy policy + EULA URLs** finalized + linked from app — **code-done 2026-06-15** (quick-260615-i1h, `bd26470`): canonical fe3dr.com/{privacy,terms,refund,vendor-terms,eula} now render FULL policy bodies; customer + vendor apps have in-app Privacy/Terms/Refund/EULA (+ vendor Chef Agreement) wired into nav. Operator/email/date consistent. _(legal artifact: counsel sign-off still pending — see Wave 6 "Legal content review" + COUNSEL-REVIEW.md)_
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
- [x] **i18n setup** — `react-i18next` + `expo-localization`, device-locale detection + AsyncStorage persistence (`lib/i18n.ts`)
- [x] **Hindi translation** — `en`/`hi` resource bundles for chef-facing strings; More tab migrated to `t()` as the reference _(full per-screen string coverage is mechanical/incremental from here)_
- [x] **Locale picker** in More tab — System / English / हिन्दी (`app/language.tsx`), auto-detects device locale
- [x] **Deep links** — `homechef-vendor://` scheme + push tap-through generalized across all categories (order/FSSAI/statement/availability) via `resolvePushRoute` + cold-start replay
- [x] **OTA updates** via EAS Update — `expo-updates` + `updates.url` (u.expo.dev), `appVersion` runtimeVersion policy, Hermes locked; `production`/`preview` channels in eas.json
- [~] **Bundle size audit** — Hermes engine locked (`jsEngine: hermes`); the size *measurement* + asset-compression pass needs a production build _(QA — run on a build)_
- [~] **Accessibility pass** — all new components ship with `accessibilityRole`/`Label`; full VoiceOver sweep + contrast audit needs the app running on device _(QA)_
- [ ] **App Store screenshots + listing** for IN store _(blocked: needs Apple Dev account + built app)_
- [ ] **App Store submission + review wait** _(blocked: Apple Developer enrollment)_

### Backend
- [x] **OpenTelemetry tracing** — `homechef-api` TracerProvider + Cloud Trace exporter (graceful no-op without creds, ratio sampler via `OTEL_SAMPLING_RATE`), otelgin span-per-request + trace_id bridged into logs _(auth-bff still pending — separate repo)_
- [x] **Structured logging with correlation IDs** — slog JSON `logger` pkg + `X-Request-ID` middleware + structured access log; trace_id joined in
- [x] **Audit log for sensitive mutations** — `LogAudit` wired into payout updates, doc downloads (invoice/statement/TDS), approval responses, order cancel/item-cancel/refund, DPDP delete; indexed `CorrelationID` on `audit_logs` (PII-safe)
- [x] **`min_scale: 1` on homechef-api** — verified already set in prod (min-scale 1, max-scale 10); keeps a warm pod + makes the crons reliable
- [x] **Connection pool tuning** — right-sized to env-configurable 20/5 + `ConnMaxIdleTime` (was an unsafe 100/10 on the shared db-f1-micro)
- [x] **`gomaxprocs`** — `go.uber.org/automaxprocs` aligns GOMAXPROCS with the Knative CPU limit

### Launch ops
- [x] **Runbook** for top-5 incident types — `docs/ops/RUNBOOK.md` (auth-bff / DB-pool / Razorpay / GIP / Knative + cron health)
- [x] **On-call** paging policy (solo) — `docs/ops/ON-CALL.md` (P1 page triggers vs. P2 morning)
- [x] **Status page** plan at `status.fe3dr.com` — `docs/ops/STATUS-PAGE.md` _(hosting setup is operational)_
- [x] **First-10-chefs concierge script** — `docs/ops/CONCIERGE.md`

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

## Wave 5 — Platform consolidation (added 2026-06-10, runs parallel to W4 ops; ships after App Store approval)

**Goal:** Mobile-first like Uber — the web apps go away in favor of a single landing page with
store links, and Home Chef becomes a first-class Tesserix product visible in tesserix-home.

### 5A. Web sunset → landing page

- [x] **Landing page** at `fe3dr.com` — **BUILT + DEPLOYED LIVE 2026-06-11.** New `apps/web-landing` (Next.js 16 static export), Airbnb-**coral** consumer palette (owner chose coral over persimmon), Geist/Inter, authentic **Indian** home-cooking imagery (Pune launch), scroll-driven motion (signature animated order-route, parallax, dish marquee, FSSAI marks), live App/Play store badges (placeholder URLs — TODO), SEO (metadata/OG/sitemap/robots/JSON-LD), a11y AA. Deployed into the **existing `homechef-web` slot** (same GHCR image + ArgoCD app + fe3dr.com routing, port 80) via new `homechef-web-landing-build.yml` CI → no routing/DNS change. Old `apps/web` SPA de-wired (docker-compose + CI removed, `SUNSET.md`, dir kept). `vendors.fe3dr.com` API host verified unaffected (401 = alive). See [[project_ios_native_build_from_source]]-style memory `web-sunset-app-only`.
  - ⏳ Follow-ups: real store-listing URLs + official badge artwork; lawyer-reviewed legal content for `/privacy`,`/terms`,`/refund` stubs; owned photography + 1200×630 OG image; chef-onboarding link (currently `mailto:`).
- [~] **Decommission `apps/vendor-portal` + `apps/delivery-portal` web UIs** — **repo side done 2026-06-12** (`303e543`): `SUNSET.md` added to both + CI removed (`homechef-vendor-portal-build.yml`, `homechef-delivery-portal-build.yml`). **Still owner/infra:** retire the ArgoCD apps + 301 `vendors.fe3dr.com`/`delivery.fe3dr.com` web routes → landing (tesserix-infra + Cloudflare).
  - ⚠️ **`vendors.fe3dr.com` is the mobile app's API host** (`EXPO_PUBLIC_API_URL=https://vendors.fe3dr.com/api/v1` + BFF auth routes). Only the web UI dies — API/auth-bff routing on that host MUST keep serving (their CI is untouched). Same check for any API traffic on the other hosts before removal.
- [ ] **auth-bff registry cleanup** — drop the dead web app entries (`web`, `vendor-portal`, `delivery-portal` blocks in `apps/auth-bff/homechef-products.yaml`) **once portals are gone**; keep mobile-facing entries. **Intentionally deferred** — left in place (mirrors how `apps/web` was sunset) because deployments are still live and `vendors.fe3dr.com` auth must keep serving the mobile vendor app until the infra cutover.
- [~] **Repo cleanup** — `SUNSET.md` now on all retired app dirs (`web` + `vendor-portal` + `delivery-portal` + `mobile-delivery`); web CI removed. Dirs kept for history (not deleted), per the `apps/web` precedent. README/monorepo-docs refresh still pending.
- **Sequencing:** build the landing now; flip DNS/routing only after both store listings are live (a landing with dead store links is worse than the current web app).
- **Decision (owner, 2026-06-10):** customer web ordering dies entirely — app-only (Uber model), fewer apps to manage. All three web apps (`web`, `vendor-portal`, `delivery-portal`) sunset; the landing page is the only web surface.

### 5B. Tesserix platform integration (mark8ly pattern)

> **Owner cutover steps (DB role, secret mgmt, ArgoCD/Cloudflare teardown, infra hardening) are written up with concrete Home Chef values in [`docs/ops/CUTOVER-RUNBOOK.md`](docs/ops/CUTOVER-RUNBOOK.md).**

Reference: mark8ly's registration spans 4 touchpoints. Home Chef replicates each:

- [ ] **Platform auth-bff `products.yaml`** (`tesserix-new/auth-bff/products.yaml`) — add `homechef` product entry (domain `fe3dr.com`). GIP tenants already exist (`HomeChef-Customer-rqg8a`, `HomeChef-Business-8s8ql`) — reference them; do NOT create new ones.
  - **Deliberately deferred (2026-06-12):** this only matters for **platform SSO consolidation**, which is a separate (deferrable) effort — tesserix-home's admin access to Home Chef is via the **direct-DB platform-admin role** (above), not the platform auth-bff. Home-Chef-App keeps running its OWN auth-bff (`homechef-products.yaml`) for the mobile apps — untouched. Adding a half-wired entry here (clientSecretEnv not set) risks the platform auth-bff failing to load its config at boot, so it's left until SSO consolidation is actually scoped.
- [~] **`tesserix-infra/services.yaml`** — **NOT APPLICABLE (verified 2026-06-12).** That registry is consumed by Terraform (GCP **Pub/Sub** topic provisioning), `generate-overlays.sh` (k8s overlay generation), and go-shared dispatch. Home Chef uses **NATS not Pub/Sub**, ships its **own `tesserix-k8s/charts/apps/homechef-*` charts**, has its **own CI** (`homechef-*-build.yml`), and doesn't use the platform go-shared — so adding it there would trigger unwanted Pub/Sub topics + conflicting overlays. Home Chef is registered with the platform via the **apps-registry DB row** (runbook §1) + its own charts, not this file. _(Monitoring already covers it via the homechef namespace/CNPG Prometheus metrics the overview uses.)_
- [x] **tesserix-home product config** — **DONE 2026-06-12** (tesserix-home `c2ef673`): `homechef` ProductConfig in `lib/products/configs.ts` (namespace/cluster + KPI tiles active chefs / orders today / GMV today / pending approvals). KPIs are served by a new product-scoped route `/api/admin/apps/[product]/kpis` reading homechef_db directly via `lib/db/homechef.ts` (mark8ly's dashboard path untouched). `rowCountTables` left empty in v1 (DB-size on the overview comes from CNPG Prometheus metrics).
- [x] **tesserix-home UI** — **DONE 2026-06-12**: `app/admin/apps/homechef/page.tsx` (one-liner) + `homechefNav` + `RailContext`/`getActiveContext`/`getSecondaryNav` + product-switcher button in `sidebar.tsx`. `public/homechef-icon.png` is a **placeholder** (copy of generic icon — replace with real art). Admin Apps **grid** tile is DB-driven (`tesserix_admin.apps` row — the runbook INSERT), not code. _(Phase A = registration + overview; the deeper admin sub-pages below are Phase B.)_
- [ ] **OpenFGA stores** — register `hc-internal` / `hc-customer` stores via `go-shared/authz` MultiStoreClient at service startup if/when homechef services adopt platform FGA checks _(optional for v1 — current JWT role model keeps working; flag as fast-follow)_.
- [x] **Chef payout/settlement management (admin)** — **DONE 2026-06-12** (tesserix-home `974300f`): `/admin/apps/homechef/payouts` — all-chef weekly statements list (filter week/chef/status) + CSV export + **mark-paid** (direct homechef_db write in a txn + audit row, idempotent). Direct-DB per the mark8ly pattern; commission % is `platform_settings.service_fee_percent`.
  - **Access pattern (per mark8ly):** tesserix-home reads/writes the Home Chef DB **directly** via a `pg.Pool` (see `tesserix-home/lib/db/mark8ly.ts`) as a dedicated platform-admin role with **CRUD on all tables but NO DDL** — it does NOT call the homechef-api. So list-statements / mark-paid / configure-commission / CSV export are **direct DB** in the tesserix-home session, not API endpoints.
  - **homechef-api owns only the DDL** (the admin role can't `ALTER`): **DONE 2026-06-12** — `status / paidAt / payoutRef` columns added to `WeeklyStatement` (commit `63753bd`). Commission % is already `platform_settings.service_fee_percent`. **Infra prereq:** grant a `homechef_platform_admin` DB role (CRUD, no DDL) + wire `HOMECHEF_DB_*` env into tesserix-home (mirror the `MARK8LY_DB_*` setup). **DONE 2026-06-12** (role + grants live, `HOMECHEF_DB_*` wired via ExternalSecret + `database.homechef.enabled: true`). _(An earlier homechef-api `/admin/payouts/*` endpoint attempt was reverted — wrong layer.)_
  - **Already shipped:** weekly settlement **calculation** is automated (`statement_cron.go` → idempotent Mon–Sun IST statements w/ commission + CGST/SGST/IGST + TDS + NetPayout). Chef self-service view + receipt/invoice PDF is live in the vendor app.
  - **Disbursement gated** on RazorpayX + an Indian entity (see payment-structuring decision: stick with Razorpay, Indian subsidiary deferred, low volume). Until then payouts are computed weekly and **paid manually**; the RazorpayX auto-payout adapter (cron → createPayout per pending statement → webhook marks paid) is a fast-follow once the entity lands.
- [x] **Delivery (3PL) admin in tesserix-home** — **DONE 2026-06-12** (tesserix-home `c10d9aa`): `/admin/apps/homechef/delivery` — provider list + **enable/disable** (direct DB toggle + audit) + **cost reconciliation** (provider_cost vs collected delivery_fee → margin/subsidy). Provider **key config + "test connection"** intentionally stay in homechef-api's `/admin/delivery/providers` CRUD (it owns the keys + makes the outbound test call) — not duplicated direct-DB. See Wave 7E.
- [x] **Customer-ops oversight in tesserix-home** — **DONE 2026-06-12** (tesserix-home `1848d24`): overview GMV/orders KPI tiles (Phase A) + `/admin/apps/homechef/orders` — recent orders (filter by status) with GMV summary, read-only direct DB.
- [ ] **Secret management from the admin dashboard** — admins set/rotate platform secrets from tesserix-home; **values stay in GCP Secret Manager** (source of truth — unchanged). **Access pattern:** tesserix-home calls GCP Secret Manager **directly** via `@google-cloud/secret-manager` (already in its stack) — GCP SM is platform infra, not product-owned, so **no homechef-api endpoint**. tesserix-home holds the curated allowlist of GCP secret IDs (Razorpay test/live key+secret+webhook, SendGrid, Firebase, JWT) and does `addSecretVersion` (set/rotate) + `getSecretVersion` (is-set/last-rotated, never `access` the payload for status). **Hard requirements:** platform-super-admin only, audited (**never log the value**), UI **write-only / masked** (set-vs-unset + last-rotated, never plaintext). **Propagation:** new GCP SM version → ESO sync → k8s secret → env var → **pod rollout required** for env-injected secrets. _(`services/secrets.go` in homechef-api stays as-is — it's the service's own runtime read path, not the admin write path.)_
  - **Deferred (Phase B blockers, 2026-06-12):** not built yet — three things must be sorted first: (1) `@google-cloud/secret-manager` isn't a tesserix-home dependency; (2) the **real GCP secret IDs** + homechef ESO `remoteRefs` must be confirmed (guessing → "is-set" always false); (3) the tesserix-home pod needs **GCP SM access** wired (mark8ly reaches secrets via ESO, not direct SM). Best done alongside the runbook's GCP provisioning step.
  - **Propagation:** new GCP SM version → ESO sync → k8s secret → env var. **Env-injected secrets need a pod rollout** (or hot-reload) to take effect; secrets read live from DB (e.g. `DeliveryProvider` API keys, currently encrypted in-table) apply instantly — decide per secret whether it lives in GCP SM (env) or DB (live).
  - **Curated allowlist** of known keys with set/rotate (not arbitrary creation): Razorpay (test/live key id + secret + webhook secret — this becomes the mechanism for the Wave-6 test→live switch), 3PL provider keys (Shadowfax etc., Wave 7), SendGrid, Firebase, JWT.
- **Known friction (from mark8ly audit):** sidebar nav arrays + icon assets + per-product route dir are hardcoded — 3–4 file edits, no architectural blocker; "marketplace" strings leak in a couple of shared components.

### Definition of done
- `fe3dr.com` renders the landing with working store links on mobile + desktop; old portal URLs 301 to it; the vendor mobile app's API calls are unaffected (smoke: login + dashboard fetch from a device after DNS flip)
- Home Chef tile appears in tesserix-home products grid → opens a working product overview with live KPI tiles
- Platform admin can reach Home Chef ops data without leaving tesserix-home

### Parallelization
- Subagent A: landing page build (can start immediately)
- Subagent B: tesserix-home + registries (separate repos — needs a session in `tesserix-new/` root, not Home-Chef-App)
- DNS/routing flip + portal teardown: by hand, after store approval

---

## Wave 6 — Customer app production close-out (separate session)

**Goal:** the customer ordering app is launch-ready alongside the vendor app.

**Status:** Phase-02 verified **feature-complete** (11/11 CUST requirements, no stubs/TODO blockers). Shipped this session (2026-06-11): address fetch/create + **address labels** + **per-item special instructions**; **in-app Razorpay payment** (Standard Checkout in a WebView, server-side verify — replaces the external-browser flow, adds `react-native-webview`); **legal screens** (Refund/Terms/Privacy); order-shape mapper + **chef-on-orders** API field (deployed); place-order crash fix + **error surfacing**; **18 misconfigured delivery zones deactivated** to unblock ordering.

### Pending
- [x] **Live driver location on the tracking map** — **code shipped via Wave 7C** (2026-06-11): `Delivery` carries rider coords/identity, the provider webhook persists them, and `DeliveryResponse.ToResponse()` surfaces the real 3PL rider. Goes live the moment a real provider sends location webhooks (needs Shadowfax creds). No own-driver fix needed given fleet retirement.
- [ ] **Payment happy-path verification** — real cart → Razorpay **test card** (`4111 1111 1111 1111`) → success, tapped through on device. Then switch `rzp_test_*`→`rzp_live_*` in `homechef-api-secrets` and register the **live Razorpay webhook** at `https://api.fe3dr.com/webhooks/razorpay` (backend handler verified live; dashboard config is owner action).
- [x] **Surface Privacy in the UI** — **shipped 2026-06-12**: Profile → **Legal** section (Terms · Privacy · Refund) added, surfacing the previously-unlinked `/privacy` (`app/(tabs)/profile.tsx`).
- [~] **Legal content review** — **docs now reviewer-ready 2026-06-15** (quick-260615-i1h): full Refund/Terms/Privacy/EULA drafted across landing + customer + vendor, `COUNSEL-REVIEW.md` at repo root lists every decision counsel must confirm. **Entity question RESOLVED** — "Tesserix Pty Ltd" (ACN 694 070 865 / ABN 59 694 070 865, NSW Australia) is the genuine parent (same as mark8ly); Home Chef now mirrors mark8ly's cross-border structure (NSW governing law + consumer-protection carve-out + APP/DPDP-aligned + India grievance officer). **Still pending:** actual counsel sign-off; narrowed residual (Indian GST contracting entity / GSTIN, NSW-jurisdiction enforceability for Indian food consumers) + placeholders (GSTIN, named Grievance Officer, `dpo@fe3dr.com` mailbox) per COUNSEL-REVIEW.md. Confirmed: support **support@fe3dr.com**, effective date 11 Jun 2026.
- [x] **Customer unit tests** — **shipped 2026-06-12**: 22 tests / 3 suites (jest-expo, green) — `cart-store` (add/cross-chef/increment/remove/qty/instructions/totals/immutability), chef mappers (`mapChef`/`mapMenuItem` — exported), and the tracking poll-interval helper (`trackingRefetchInterval` — extracted pure). Jest globals via `@jest/globals` import (no `@types/jest` install; tsc baseline held at 77). _(Component/render tests still future.)_
- [ ] **EAS production build + App Store submission** (customer track; separate from vendor) — blocked on Apple Developer enrollment (same blocker as vendor).
- [~] **Address coordinate capture** — **shipped 2026-06-12** via the **geocoder** (no `expo-location`/native rebuild): the backend autocomplete now surfaces Photon `geometry` lat/lon, and the client threads the picked-suggestion coords through onboarding + `useCreateAddress` into the `Address` row (`addressLatitude/Longitude` already bound server-side). Manual-typed addresses (no suggestion picked) still have no coords → flat-fee + zone-skip fallback. _(Optional future: an `expo-location` "use current location" GPS button — needs the native rebuild.)_
- [ ] **Delivery zones** — still **OFF** (ordering unrestricted). Coord capture is now in (above), but zones still have **NULL bounds** and re-enabling would block any coord-less (manually-typed) address. Re-enable only after seeding **real zone bounding boxes** AND confirming coord capture is reliable on device.

---

## Wave 7 — Delivery via 3PL providers (Shadowfax-class) + own-fleet retirement

**Decision (owner, 2026-06-11):** Home Chef will **not run its own delivery fleet** — managing delivery agents is overkill at this scale. Deliveries are fulfilled by **third-party logistics providers** (Shadowfax / Dunzo / Porter class). The **own delivery app (`apps/mobile-delivery`) + own-fleet backend are retired.** Mirrors the web-sunset decision: fewer surfaces to operate.

### 7A. Retire own-fleet
- [~] **Sunset `apps/mobile-delivery`** + the web `delivery-portal` (already in Wave 5A). **Repo side done 2026-06-12** (`303e543`): `SUNSET.md` on both; web `delivery-portal` CI removed (`mobile-delivery` has no GH CI — EAS-built). **Still owner/infra:** ArgoCD teardown + `delivery.fe3dr.com` 301 → landing; pull any published EAS build.
- [x] **Deprecate own-driver code paths** — **routes retired 2026-06-11** (`routes/routes.go`): removed the driver onboarding groups (`/delivery/onboarding/*`, `/driver/onboarding/*` incl. `DriverOnboardingPayout`), the driver-app group (`/delivery/*`: profile/online/location/current/available/accept/status/earnings + driver Stripe Connect), `/driver/referral/*`, `/driver/subscription/*`, and `ManualAssignDelivery`. Admin + staff "delivery partners" views are now **read-only** (GET list/detail/stats/overview kept; `verify`/`suspend`/`assign` mutations removed). **Handler files, models, and DB tables left intact** (keep until data is irrelevant) — fully git-reversible. 3PL provider CRUD + delivery zones + staff management unaffected.
- ⚠️ Confirm nothing API-critical rides `delivery.fe3dr.com` before teardown (same caution as the vendor host in 5A).

### 7B. 3PL integration — what EXISTS (framework, scaffolded ✅)
- `models.DeliveryProvider` — full config: name/code (Dunzo/Porter/**Shadowfax**), `APIBaseURL/APIKey/APISecret/WebhookSecret`, `StatusMapping` (provider→Fe3dr `DeliveryStatus`), `SupportedCities/Countries`, `MaxDistance`, `PricingModel` + `BaseCost` (what Fe3dr pays per delivery).
- Admin CRUD: `/admin/delivery/providers` — List/Get/Create/Update/Delete/Toggle/**TestConnection**/Stats (`handlers/delivery_provider.go`).
- Inbound webhook: `POST /webhooks/delivery/:provider` — HMAC-verified (`X-Webhook-Signature`), `services.HandleProviderWebhook(code, body)` maps provider status → `DeliveryStatus`.

### 7C. 3PL integration — what's MISSING (build this)
- [x] **Per-provider OUTBOUND adapter** (Shadowfax) — `DeliveryProviderClient` interface (`GetQuote`/`CreateTask`/`CancelTask`/`TrackTask`) keyed by `provider.Code` via `ClientFor`; swap-ready Shadowfax adapter (`services/shadowfax_client.go`) built against the documented hyperlocal REST shape with auth/endpoints/field-mapping isolated behind `TODO(shadowfax-creds)` markers. Providers without an adapter fall back to the existing mock path. Interface + registry in `services/delivery_client.go`. _(2026-06-11; activates on real Shadowfax sandbox creds — only the marked helpers change.)_
- [x] **Serviceability + quote at checkout** — `services.QuoteCheckoutDeliveryFee` replaces the flat `policy.BaseDeliveryFee` at checkout; live 3PL quote (fee + ETA via `GetProviderQuote`) when drop coords + a serviceable provider exist, else **falls back to the flat policy fee**. Non-fatal: checkout never blocks on the quote. Drop coords now flow (2026-06-12 geocoder capture, see Wave 6), so quotes activate as soon as a provider is enabled (needs Shadowfax creds).
- [x] **Auto-dispatch** — `services.DispatchOrderDelivery` fires on the order→`ready` transition (`chefHandler.UpdateOrderStatus`), off the request path + **idempotent**. Selects provider by serviceable city/distance/cost (`FindAvailableProvider`), books via `CreateProviderDelivery`, persists `provider_id` + external task id + cost on a new `Delivery` row, links `order.delivery_id`. Own fleet retired → **no internal-driver fallback**; unserviceable orders are logged for manual handling.
- [x] **Tracking → closes the Wave-6 live-map gap** — `Delivery` now carries `RiderName/Phone/Latitude/Longitude` + raw `ProviderStatus`; `HandleProviderWebhook` persists rider location/identity from the inbound webhook (common field aliases); `DeliveryResponse.ToResponse()` **prefers the 3PL rider coords** (falls back to own-fleet partner). `DeliveryPartnerID` made nullable for 3PL rows. _(Map goes live as soon as a real provider sends location webhooks.)_
- [x] **Cancellation** — `services.CancelOrderDelivery` cancels any booked 3PL task (`CancelTask`) on order cancel from **both** the customer and chef paths; no-op when no/terminal delivery. 3PL cancellation-fee reconciliation tagged `TODO(shadowfax-creds)` (Wave 7E admin view).

### 7D. Delivery payments & payouts (3PL model — much simpler than own-fleet)
- **Customer side:** delivery fee (3PL quote) is collected in the order total via Razorpay — **no change to the customer payment rail.**
- **Platform → 3PL settlement:** Fe3dr pays the provider per `PricingModel`/`BaseCost`, typically a **prepaid wallet** or **postpaid weekly invoice** with the 3PL. **No per-driver payouts** — that entire complexity disappears with the fleet. Reconcile collected delivery fees vs 3PL charges (margin/subsidy).
  - ⚠️ Same constraint as chef payouts: paying an Indian 3PL needs **INR funded from an Indian account/entity** (see payment-structuring decision). At low volume, top up the 3PL wallet manually from the interim entity account; automate later.
- **Tax:** capture the 3PL's GST tax invoice for input credit (entity-dependent).

### 7E. tesserix-home admin (batches with Wave 5B)
- [x] Manage delivery providers (config/keys/toggle/test), view delivery list + stats, **reconcile 3PL cost vs collected delivery fee**. Built in the single tesserix-home integration effort alongside chef-payout admin (5B). **DONE 2026-06-12** (tesserix-home `c10d9aa`: provider list/toggle + cost reconciliation; key-config + test-connection intentionally stay in homechef-api).

### Definition of done
- A customer order **auto-dispatches to Shadowfax**, the customer sees the **real rider** move on the live map, status flows pickup→delivered via webhook, and the platform's 3PL cost is recorded + reconcilable — with **zero own-fleet code in the hot path**.

---

## Tracking

This file is the source of truth. Update checkboxes as items ship. New blockers / scope changes get added to the appropriate wave OR explicitly punted to "Locked deferrals" with a one-line reason.

Weekly review: every Friday, walk through the current wave, mark items, surface blockers, re-plan if behind.
