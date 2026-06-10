# Next session — pick up where we stopped (end of day 2026-06-10)

## TL;DR

Last session shipped **37 commits** taking the vendor app from Wave 1 kickoff through to Wave 3 partial: Wave 1 application code 100%, Dependabot 88→28, Wave 2 backend 100% + mobile 11/13, Wave 3 backend 6/9 + mobile 3/6, plus DPDP export/delete. Tree clean, all builds green, all pushed.

The next BIG thing is **finishing Wave 3** (weekly statements + TDS cert + refund history view + settlement reconciliation) and **starting Wave 4**. Plus a queue of operational items waiting on console / account access.

## State on `main` at session end

| Track | Status |
|---|---|
| Wave 1 application | ✅ 100% — Sentry mobile+backend, force-upgrade gate, rate-limit, idempotency, webhook HMAC |
| Wave 1 Dependabot | 88 → 28 vulns (60-vuln reduction via pnpm overrides + `golang.org/x/net` bump + Dockerfile pin to `golang:1.26.4-alpine`) |
| Wave 2 backend | ✅ 100% (9/9) |
| Wave 2 mobile | 11/13 — only **pause-receiving with auto-resume** + **FoSCoS** deferred |
| Wave 3 backend | 6/9 — invoice PDF (customer + chef), post-delivery refund, GSTIN+HSN, auto-email invoice, DPDP export/delete |
| Wave 3 mobile | 3/6 — GSTIN, HSN, Download invoice link |
| Wave 4 | Not started |

Live prod at session end: `homechef-api-00077` Ready; next CI rebuild lands all backend Wave 1/2/3 work + the `golang:1.26.4-alpine` Go stdlib CVE close.

## What got SHIPPED this session (commit ladder)

```
89c03ba feat(api): Wave 3 close-out - auto-email PDF invoice + DPDP /chef/me/export + /chef/me/delete
b19d51a feat(mobile): Wave 3 HSN input + Download invoice PDF on delivered orders
72a6a88 feat(mobile): Wave 3 GSTIN input in onboarding
08b08a3 feat(api):    Wave 3 backend foundation - GSTIN+HSN, maroto PDF invoice, post-delivery refund
c63f2d5 feat(mobile): More tab nav (admin-requests, documents/renew, notif-prefs)
90f57bf feat(wave2):  reviews summary endpoint + approval response screen + FSSAI license number/expiry
9e61087 feat:         Wave 2 per-line cancel (backend OrderItemResponse + mobile strikethrough/refunded UI)
3ae0fec feat(mobile): doc renewal screen
7c28712 fix(mobile):  reviews list adapter (was crashing on wire shape)
bf6fde2 feat(mobile): whole-order cancel UI with reason picker
c3b60a3 feat(mobile): notification-preferences + admin-requests inbox
508994a feat(api):    Wave 2 backend foundation (5 items)
9927661 feat(api):    Wave 2 chef order cancellation
1bad99f chore(api):   Dockerfile pin to golang:1.26.4-alpine
aa9a1f0 chore(deps):  pnpm overrides close 63 npm vulns
1e4ed20 chore(api):   x/net 0.51 → 0.55
97d63e8 fix(api):     version-check storeUrl fallback
435f004 feat(api):    Idempotency-Key middleware
07b8425 feat(api):    Redis token-bucket rate limit
46644a8 fix(api):     delivery webhook HMAC
eada225 feat(api):    Sentry-go init on homechef-api
f76e6fa feat:         Wave 1 mobile observability + force-upgrade gate
+ docs commits 16921e6, 804a215, b8e2a08
```

## What's pending in code (sorted by leverage)

### Wave 3 remaining (close out the wave)
- [ ] **Weekly settlement statements** — backend cron (Sunday 23:59 IST) + `GET /chef/statements/weekly` PDF + mobile Earnings tab. ~4–6h. Use existing `services.GenerateOrderInvoicePDF` pattern with maroto.
- [ ] **TDS Form 16A** annual PDF — `GET /chef/tax/certificate?year=YYYY`. ~2h. Same maroto pattern.
- [ ] **Refund history mobile view** — Earnings → Refunds section consuming `Order.RefundID/RefundAmount/RefundReason` (all already on the model). ~1h.
- [ ] **Stripe/Razorpay daily settlement reconciliation cron** — compare platform rows vs gateway records, alert on drift. ~3h.

### Wave 2 stragglers
- [ ] **Pause-receiving with auto-resume** — `Open / Closed / Back in {15,30,60} min`. Needs new backend support: scheduler that flips `accepting_orders` back on after the timer; mobile UI on the dashboard's status button. ~3h backend + ~1h mobile.
- [ ] **FoSCoS API integration** — external dep, multi-day; fallback is manual admin review queue (already exists).

### Wave 4 (entirely new — start when Wave 3 closes)
- i18n + Hindi translations + locale picker
- Deep links (`homechef://orders/<id>`) registered + push tap-through tested
- EAS Update OTA channel
- Bundle size audit, accessibility pass
- App Store screenshots + submission
- OTel tracing → Cloud Trace
- Structured logging + correlation IDs
- Audit log table
- `min_scale: 1` on homechef-api
- Connection pool tuning (5/2 → 20/5)
- Runbook + status page + concierge script for first-10-chefs

## What's pending out of code (operational — your hands)

| Action | Where |
|---|---|
| Apple Developer Program enrollment | apple.com/developer (24h–7d wait) |
| App Store Connect app record + TestFlight internal group | App Store Connect |
| APNs production cert in Firebase + EAS credentials | Firebase + EAS Cloud |
| Sentry account → create projects → `eas secret:create EAS_SECRET_SENTRY_DSN`, `EAS_SECRET_SENTRY_AUTH_TOKEN`, `EAS_SECRET_SENTRY_ORG` | sentry.io + EAS CLI (from `apps/mobile-vendor/`) |
| Cloudflare WAF managed ruleset + custom `/auth/auto-login` rate-limit | Cloudflare dashboard (Pro plan needed; ~$20/mo) |
| Cloud SQL backup retention + non-prod restore drill | gcloud + GCP Console |
| Apply `docs/base-image-go-1.26.4.patch` to `tesserix/base-docker-images` | Separate repo; I have read-only access |
| Set `SENTRY_DSN_API` + `SENTRY_ENVIRONMENT` env on homechef-api | tesserix-infra/k8s ExternalSecret |
| Set `MIN_VERSION_VENDOR_IOS/ANDROID`, `LATEST_VERSION_VENDOR_*`, `STORE_URL_VENDOR_*` env vars when ready to enforce the upgrade wall | tesserix-infra/k8s |
| Configure `webhook_secret` per enabled delivery provider | `PUT /api/v1/admin/delivery/providers/:id` |
| Triage remaining 6 Go stdlib vulns | rebuild homechef-api after the Dockerfile pin lands (CI may have already done this — check `kubectl get ksvc homechef-api`) |
| Verify FSSAI cron fires in prod | `kubectl logs` for `"fssai-reminder: scan complete"` |
| Legal: privacy policy + EULA URLs | wire into mobile app + onboarding consent capture |

## Known gotchas (still true)

- **Use `npx eas-cli@18.4.0`** — 20.x silently fails simulator builds (exit 0, no artifact)
- **`cd apps/mobile-vendor`** before any eas-cli command (else stray root `eas.json` is created)
- **Sim UUID:** `AD109A46-2F99-43C3-8AAA-FEE68DC8499E`, bundle `com.homechef.vendor`
- **Kargo overrides** `argocd/prod/apps/homechef/*.yaml` image tags — never roll back via manifest edits; use ArgoCD UI Rollback or push a revert
- **kube context for prod GKE:** `gke_tesseracthub-480811_asia-south1_tesseract-prod-in-gke`
- **iOS Pressable bug** — function-style returning array drops layout props on iOS. Use object-style returns OR push layout to inner View. See `~/.claude/projects/.../memory/feedback_ios_pressable_array_style.md`.
- **Pre-existing typecheck noise** — mobile-vendor has ~101 pre-existing TS errors (lucide-react-native missing exports + JSX type duplication from hoisted `@types/react` in `packages/mobile-shared/node_modules`). Filter on file paths you touched.
- **EMU permissions** — `gh pr create` fails on `tesserix` org; open PRs via the printed `Create a pull request` URL. `git push` works fine.

## Operational state in prod (verify on next session start)

- `homechef-api` Knative latest revision `00077` Ready (from `97d63e8` storeUrl fix)
- After CI cycles for the recent backend pushes, expect a fresh revision incorporating: Sentry, rate-limit, idempotency, webhook HMAC, cancel flows, doc replace, notif prefs, FCM topics, FSSAI cron, GSTIN/HSN, invoice PDFs, refund endpoint, DPDP endpoints, auto-email invoice, Go 1.26.4 base image
- `homechef-auth-bff` Deployment 2/2 (NOT Knative; already at minReplicas:2)
- `homechef-postgres-pooler-ro/rw` 2/2
- Dependabot: **28 vulns** at last push (5 critical / 9 high / 13 moderate / 1 low). The 6 Go stdlib among them close once the `golang:1.26.4-alpine` rebuild propagates → should drop to ~22.

## Cross-cutting design contracts in effect

- Force-upgrade: `GET /api/v1/mobile/min-version?platform=ios|android&app=vendor` → `{ minVersion, latestVersion, storeUrl }`. Mobile sends `X-App-Version` + `X-Platform` on every request; backend returns `426 Upgrade Required` when too old. Excludes `/health`, `/metrics`, the min-version endpoint itself.
- Rate-limit: 60req/min per chef, 30req/min unauth-by-IP. Fail-open on Redis down. `X-RateLimit-*` headers on every response.
- Idempotency: Opt-in via `Idempotency-Key` header on POST/PUT/PATCH for `/chef/orders/*`, `/chef/menu/*`, `/chef/documents/*`, `/chef/payments/*`. 24h TTL. 5xx not cached. Fail-open.
- Sentry: backend env `SENTRY_DSN_API` + `SENTRY_ENVIRONMENT`; mobile env `EXPO_PUBLIC_SENTRY_DSN` + `EXPO_PUBLIC_SENTRY_ENVIRONMENT`. Both no-op cleanly when unset.
- Redis fail-mode: ALL middleware fails OPEN (rate-limit, idempotency, FSSAI dedup). Documented tradeoff: burst risk during Redis outage is preferred over service downtime.
- Cancellation: reason enum `out_of_ingredient | equipment_failure | customer_request | other`. Whole-order = full refund + status flip. Per-line = partial refund + line strikethrough + order subtotal/tax/total recompute, status unchanged.
- Invoice PDF: maroto v0.46.2 generates GSTIN-formatted tax invoice with HSN-coded line items; auto-emailed via SendGrid attachment on `order.delivered` NATS event; chef can also download from order detail screen via `expo-file-system` + `expo-sharing`.

## First actions next session (suggested)

1. **Check prod state:** `kubectl --context gke_tesseracthub-480811_asia-south1_tesseract-prod-in-gke -n homechef get ksvc homechef-api` — confirm latest revision is rolled and serving
2. **Smoke-test the new endpoints** against `vendors.fe3dr.com`:
   - `curl -s vendors.fe3dr.com/api/v1/mobile/min-version?platform=ios&app=vendor` → 200 JSON
   - With auth: `curl -H "Idempotency-Key: test-1" ...` to verify middleware
   - With `X-App-Version: 0.0.1` + `X-Platform: ios` → 426
3. **Set Sentry secrets** if not done (unblocks crash reporting on next build):
   ```
   cd apps/mobile-vendor
   npx eas-cli@18.4.0 secret:create --scope project --name EAS_SECRET_SENTRY_DSN --value <dsn>
   npx eas-cli@18.4.0 secret:create --scope project --name EAS_SECRET_SENTRY_AUTH_TOKEN --value <token>
   npx eas-cli@18.4.0 secret:create --scope project --name EAS_SECRET_SENTRY_ORG --value <org>
   ```
4. **Pick next batch.** Suggested top picks:
   - **Close Wave 3:** weekly statements + TDS Form 16A + refund history (4 items, ~8h focused) → wave fully done
   - **OR Wave 2 pause-receiving:** lower priority but a smaller, cleaner unit (~4h)
   - **OR start Wave 4:** OTel + structured logging + audit log table is the most leverage-positive trio for launch readiness

## Suggested next-session prompt

```
Continuing the Home Chef vendor app. Read NEXT-SESSION.md + PROD-READINESS.md first (don't re-discover state — read the progress snapshot at top of PROD-READINESS).

State: Wave 1 application 100%, Wave 2 100% backend + 11/13 mobile, Wave 3 6/9 backend + 3/6 mobile, Dependabot 88→28. 37 commits since 2026-06-05. Tree clean.

First action: verify prod (kubectl ksvc homechef-api on the prod GKE context per NEXT-SESSION gotchas) + smoke test /mobile/min-version + /chef/orders/:id/invoice.pdf.

Then: [PICK ONE]
  - Close Wave 3: weekly statements + TDS Form 16A + refund history view
  - Wave 2 pause-receiving with auto-resume
  - Start Wave 4: OTel + structured logging + audit log

Operational gotchas: see NEXT-SESSION "Known gotchas". Specifically EAS CLI 18.4.0, cd into apps/mobile-vendor first, Kargo manages prod image tags, GKE context gke_tesseracthub-480811_asia-south1_tesseract-prod-in-gke.
```
