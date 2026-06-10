# Next session — pick up where we stopped (end of day 2026-06-10, session 2)

## TL;DR

This session closed **Wave 3** (financial/tax), completed all **codeable Wave 4** items (observability + reliability + mobile platform + launch-ops docs), and shipped the **Wave 2 pause-receiving** straggler. All deployed to prod and verified live (trace IDs + correlation IDs flowing). Tree clean, all builds green, all pushed.

What's left is **operational / needs-a-build / blocked-on-Apple** — not code: App Store enrollment + submission, full a11y + bundle QA on a real build, status-page hosting, auth-bff OTel (separate repo), and the mechanical full-string Hindi migration.

## State on `main` at session end

| Track | Status |
|---|---|
| Wave 1 application | ✅ 100% (Sentry, force-upgrade, rate-limit, idempotency, webhook HMAC) |
| Wave 2 | ✅ 100% in-scope (9/9 backend, 12/13 mobile) — **pause-receiving + auto-resume** done; only FoSCoS (external) left |
| Wave 3 | ✅ 100% in-scope (9/9 backend, 6/6 mobile) — statements + cron, TDS 16A, refund history, reconciliation |
| Wave 4 backend | ✅ OTel→Cloud Trace, structured logging + correlation IDs, audit log, pool tuning, automaxprocs (min-scale:1 already set) |
| Wave 4 mobile | ✅ deep links + tap-through, EAS OTA, i18n+Hindi+locale picker · ⏳ a11y/bundle QA (needs build), App Store (blocked) |
| Wave 4 launch ops | ✅ runbook / on-call / status-page plan / concierge (`docs/ops/`) |

Live prod: **`homechef-api-00088`** (`main-53535ee`) Ready, 100% traffic. Verified: `x-request-id` + `x-trace-id` echoed → logging + OpenTelemetry are live with real Cloud Trace creds.

## What got SHIPPED this session (commit ladder, newest first)

```
53535ee docs:  Wave 4 + Wave 2 pause-receiving complete — PROD-READINESS checkboxes
09471d9 feat:  Wave 4 i18n + Hindi + locale picker (react-i18next + expo-localization)
4d1880c docs:  Wave 4 launch ops — runbook / on-call / status-page / concierge (docs/ops/)
349674b feat:  Wave 4 EAS Update OTA — expo-updates + u.expo.dev + Hermes lock
afe742b feat:  Wave 4 deep-link push tap-through — resolvePushRoute + cold-start replay
babe161 feat:  Wave 2 pause-receiving with auto-resume — PausedUntil + endpoints + 1-min cron
1d0fda1 perf:  Wave 4 reliability — DB pool 100/10 → env 20/5 + automaxprocs
60d2fe9 feat:  Wave 4 audit log for sensitive mutations — LogAudit wiring + CorrelationID
2612d49 feat:  Wave 4 OpenTelemetry → Cloud Trace — otelgin + TraceContext bridge
e1dae8f feat:  Wave 4 structured logging foundation — slog + X-Request-ID middleware
926a928 docs:  Wave 3 closed (100% in-scope)
f7894c5 feat:  Wave 3 settlement reconciliation cron (Razorpay/Stripe drift)
382b37e feat:  Wave 3 refund history — GET /chef/refunds
85dd913 feat:  Wave 3 TDS certificate — GET /chef/tax/certificate?year=FY
27b9eff feat:  Wave 3 weekly settlement statements — cron + PDF + Earnings section
```
(Plus earlier-in-day vendor v2 visual redesign commits: 115e4ff, f2a5814, 0523910, …)

## What's pending in code (sorted by leverage)

### Backend
- [ ] **auth-bff OpenTelemetry** — mirror the `homechef-api` tracing setup in the `auth-bff` service (separate repo) so login flows show in the same Cloud Trace.
- [ ] **Audit-log retention job** — `audit_logs` grows unbounded; add a prune cron (keep e.g. 13 months for DPDP) or a Cloud SQL TTL policy.

### Mobile
- [ ] **Full Hindi string migration** — infra is done (`lib/i18n.ts`, `locales/en|hi.json`); the More tab is the reference. Mechanically migrate the remaining screens' strings to `t()` and add `hi` translations. Highest-value screens first: dashboard, orders, earnings, onboarding.
- [ ] **Accessibility sweep** — run VoiceOver on every screen + contrast audit on a device/sim (new components are already labeled; this is the verification pass).
- [ ] **Bundle-size audit** — build production, measure with `npx expo export` / source-map-explorer; Hermes is already locked (`jsEngine: hermes`).

## What's pending out of code (operational — your hands)

| Action | Where |
|---|---|
| Apple Developer enrollment → App Store Connect record + TestFlight | apple.com/developer |
| APNs production cert in Firebase + EAS credentials | Firebase + EAS Cloud |
| App Store screenshots + IN listing + submission | App Store Connect (needs the build) |
| Sentry projects → `eas secret:create` DSN/auth-token/org | sentry.io + EAS CLI (from `apps/mobile-vendor/`) |
| Status page hosting (BetterStack/Instatus) → CNAME `status.fe3dr.com` | per `docs/ops/STATUS-PAGE.md` |
| Cloudflare WAF managed ruleset + `/auth/auto-login` rate-limit | Cloudflare (Pro plan) |
| Cloud SQL backup retention + restore drill | gcloud + GCP Console |
| Privacy policy + EULA URLs → wire into app + onboarding consent | legal |
| Triage remaining 30 Dependabot vulns (6 critical / 9 high) | repo security tab |
| First push of an OTA update once a build exists | `cd apps/mobile-vendor && npx eas-cli@18.4.0 update --channel production` |

## New surface added this session (for quick reference)

**New routes** (all under `/api/v1/chef`, BFF-auth + RequireChef):
- `GET /statements/weekly` · `GET /statements/:id/statement.pdf`
- `GET /tax/certificate?year=FY`
- `GET /refunds`
- `POST /availability/pause` `{minutes:15|30|60}` · `POST /availability/resume`

**New crons** (in-process on homechef-api, reliable under min-scale:1, all fire on startup):
- `weekly-statement` (daily, Mon–Sun IST close) · `reconciliation` (daily) · `availability-resume` (1-min) · existing `fssai-reminder` (daily)

**New env vars** (all have safe defaults):
- `OTEL_SAMPLING_RATE` (default 0.1) · `DB_MAX_OPEN_CONNS` (20) · `DB_MAX_IDLE_CONNS` (5) · `APP_VERSION` (trace/release tag)

**Observability**: every response carries `X-Request-ID` (+ `X-Trace-ID` when tracing on). Logs are JSON via `logger.FromContext(ctx)` and carry `correlation_id`/`trace_id`. Audit rows carry `correlation_id` — pivot from a log line to the mutation it caused.

## Known gotchas (still true)

- **Use `npx eas-cli@18.4.0`** — 20.x silently fails simulator builds.
- **`cd apps/mobile-vendor`** before any eas-cli command.
- **kube context (prod):** `gke_tesseracthub-480811_asia-south1_tesseract-prod-in-gke`, namespace `homechef`.
- **Never** `kubectl edit/patch` prod image tags — ArgoCD self-heals. Roll back via ArgoCD UI or revert the `tesserix-k8s` bump commit. (CI `bump-k8s` writes the tag there.)
- **Deploy = push to main** (paths `apps/api/**`) → CI build → `bump-k8s` → ArgoCD sync (~2 min) → new ksvc revision.
- **Pre-existing TS noise** — mobile-vendor has ~100 pre-existing lucide-react-native + JSX-dup errors. Filter on files you touched.
- **EMU permissions** — `gh pr create` fails on `tesserix` org; `git push` + `gh run watch` work fine.
- **Tracing needs GCP creds** — works in prod (Workload Identity); locally it no-ops cleanly without a project/creds.

## First actions next session (suggested)

1. **Verify prod:** `kubectl --context <prod> -n homechef get ksvc homechef-api` (expect a ≥00088 revision Ready) + `curl -D - https://vendors.fe3dr.com/api/v1/mobile/min-version?platform=ios&app=vendor` (expect `x-request-id` + `x-trace-id`).
2. **Confirm a trace** end-to-end in Cloud Trace (GCP console → Trace) for a chef request.
3. **Pick next batch:**
   - **auth-bff OTel** (close the tracing gap), or
   - **Full Hindi migration** of dashboard/orders/earnings (high user-facing value), or
   - **Operational unblock**: Apple Dev enrollment → TestFlight → first real-device QA pass (a11y + bundle).

## Suggested next-session prompt

```
Continuing the Home Chef vendor app. Read NEXT-SESSION.md + PROD-READINESS.md first (don't re-discover state).

State: Waves 1–3 100% in-scope, Wave 4 codeable items done (OTel/logging/audit/pool/automaxprocs + deep-links/EAS-OTA/i18n + launch-ops docs), Wave 2 pause-receiving done. Prod on homechef-api-00088, trace+correlation IDs verified live. Tree clean.

First action: verify prod (kubectl ksvc on the prod GKE context + curl min-version, expect x-request-id/x-trace-id headers).

Then [PICK ONE]:
  - auth-bff OpenTelemetry (close the tracing gap, separate repo)
  - Full Hindi migration of dashboard/orders/earnings strings to t()
  - Operational: Apple Dev enrollment → TestFlight → on-device a11y + bundle QA

Gotchas: EAS CLI 18.4.0, cd apps/mobile-vendor first, prod context gke_tesseracthub-480811_asia-south1_tesseract-prod-in-gke, deploy = push to main (paths apps/api/**) → ArgoCD ~2min.
```
