# Wave 1 — Backend Track Implementation Plan

**Date:** 2026-06-05
**Owner:** Mahesh (solo) + subagents
**Duration:** ~2 weeks (weeks 1–2 of the 8-week launch push)
**Source plan:** `PROD-READINESS.md` → "Wave 1 — Foundation" → "Backend" + "Dependabot/Trivy"
**Target service:** `apps/api` (`homechef-api`). `auth-bff` is a sibling Go service and gets the same Sentry recipe in a follow-up PR — out of scope here.

---

## 1. Purpose & scope

This plan covers the **backend half** of Wave 1: making `homechef-api` observable (Sentry), defendable (rate-limit + idempotency + webhook HMAC), and distributable to the mobile track (force-upgrade endpoint). It also covers the security-hygiene sub-track (Dependabot sweep + Trivy gate flip).

**In scope (7 items):**

1. Sentry-go integration (panics + 5xx capture)
2. Redis-backed rate-limit middleware (per chef + per IP)
3. Redis-backed `Idempotency-Key` middleware on chef mutations
4. Webhook HMAC verification — middleware-level for Razorpay, Stripe, delivery providers
5. `GET /api/v1/mobile/min-version` + `X-App-Version` enforcement middleware
6. Dependabot triage (82 open vulns → 0 critical/high)
7. Trivy gate flip (warn-only → fail-on-critical+high)

**Out of scope (deferred):**

- Order cancellation (full + per-line), `is_cancelled` columns, partial-refund wrapper, `POST /chef/documents/:id/replace`, notif-preferences endpoints, FCM topics, SendGrid info-requested email, FSSAI expiry cron — **Wave 2**
- Statement/tax/invoice PDFs, refund flow, GSTIN+HSN columns, cycle-close cron, DPDP `/me/export`, settlement reconciliation — **Wave 3**
- OpenTelemetry, slog convergence, audit-log table, `min_scale: 1` on api, pool tuning, `gomaxprocs` — **Wave 4**
- `auth-bff` Sentry init (same recipe, separate PR)
- Schema migrations (none required for Wave 1 backend)

---

## 2. Prerequisites

- Sentry account, two projects: `homechef-api` + `homechef-auth-bff`. DSNs captured. Free tier (5k errors/mo).
- Razorpay webhook secret already in SM as `prod-homechef-razorpay-webhook-secret` (`services/razorpay.go:35`); Stripe same (`handlers/admin.go:854`). No new secret work.
- Redis healthy in prod, fail-open on init (`main.go:82-89`). Middleware mirrors that posture.
- Dependabot enabled — verify `gh api repos/<owner>/Home-Chef-App/vulnerability-alerts` → 204.
- k8s ExternalSecret pipeline for `SENTRY_DSN_API`, `SENTRY_ENVIRONMENT` (same `prod-homechef-*` pattern).
- Mobile track agreed on the force-upgrade contract below.

---

## 3. Cross-track contracts

These are the wire-level commitments other tracks (mobile, infra) consume. Self-contained so the mobile subagent can build against them without reading the rest of this doc.

### 3.1 Force-upgrade endpoint

```
GET /api/v1/mobile/min-version?platform=ios|android&app=vendor

Response 200:
{
  "minVersion":    "1.0.0",
  "latestVersion": "1.0.3",
  "storeUrl":      "https://apps.apple.com/in/app/homechef-vendor/id..."
}

Response 400:
{ "error": "platform must be ios or android" }
{ "error": "app must be vendor" }   // customer/delivery added in W2+
```

- **No auth.** Public endpoint. Excluded from rate-limit, idempotency, and `X-App-Version` middleware (would be circular).
- **Caching:** `Cache-Control: public, max-age=300` — 5 min. Mobile clients re-fetch on app foreground.
- **Source of truth (v1):** env vars on the pod — `MIN_VERSION_VENDOR_IOS`, `MIN_VERSION_VENDOR_ANDROID`, `LATEST_VERSION_VENDOR_IOS`, `LATEST_VERSION_VENDOR_ANDROID`, `STORE_URL_VENDOR_IOS`, `STORE_URL_VENDOR_ANDROID`. Flipping min-version requires a pod restart (acceptable for v1 because we'll rev <1x/month).
- **Source of truth (v2, deferred):** `mobile_versions` table (`platform, app, min_version, latest_version, store_url, updated_at`) with an admin UI to flip min-version without deploy. Decision deferred until the first time we need a no-deploy bump.

### 3.2 `X-App-Version` header enforcement

- Mobile clients send `X-App-Version: 1.0.3` on every request.
- Middleware compares against `minVersion` (resolved per-platform via `X-Platform: ios|android` header, also mobile-set).
- If `X-App-Version < minVersion` → `426 Upgrade Required` with body `{ "error": "App version <x> is below required <y>. Please update.", "minVersion": "<y>", "storeUrl": "<z>" }`.
- If either header missing → **pass through** (web/admin/delivery portals + curl users don't set them; we only enforce when explicitly mobile).
- Exclusions: `/health*`, `/metrics`, `/api/v1/mobile/min-version`, `/webhooks/*`.

### 3.3 Sentry env vars (backend)

| Var | Required | Purpose |
|---|---|---|
| `SENTRY_DSN_API` | yes in prod, optional in dev | Project DSN for `homechef-api` |
| `SENTRY_DSN_AUTH_BFF` | only on auth-bff pod | Project DSN for `homechef-auth-bff` |
| `SENTRY_ENVIRONMENT` | yes | `production`, `staging`, or `development` |
| `SENTRY_TRACES_SAMPLE_RATE` | optional | float 0–1, default `0.1` |
| `SENTRY_RELEASE` | optional | git SHA, set by CI; aids per-deploy attribution |

If `SENTRY_DSN_API` is unset, the SDK init logs a warning and is skipped — service still boots. Same fail-open posture as Redis/NATS.

### 3.4 Redis fail-open contract (rate-limit + idempotency)

On Redis error or timeout: log WARN (no PII), increment `homechef_middleware_redis_failures_total{middleware,op}`, set `X-RateLimit-Degraded: true` / `X-Idempotency-Degraded: true`, **pass the request through**.

**Acknowledged risks:**

- Rate-limit fail-open: burst attacker bypasses the 60req/min ceiling. Mitigated by Cloudflare WAF (Wave 1 infra item) handling L7 floods upstream.
- Idempotency fail-open: a retried Razorpay charge-create could create a duplicate Razorpay *order* row. Capture is still single because of Razorpay's own idempotency (we pass `receipt = order.OrderNumber`) plus the `payment_status` guard at `handlers/payment.go:55`.

Both accepted; the alternative — failing closed and blocking 100% of traffic on a Redis outage — is worse.

---

## 4. Task-by-task spec

### 4.1 Sentry-go integration

**Goal:** Every panic in `homechef-api`, and every 5xx response, surfaces in Sentry within 30s. Mobile + backend errors live in the same org for cross-stack triage.

**Touched / new files:**

- New: `apps/api/services/sentry.go` — init + flush helpers
- Edit: `apps/api/config/config.go` — add 5 Sentry fields
- Edit: `apps/api/main.go` — init Sentry before router setup, defer flush
- New: `apps/api/middleware/sentry.go` — recovery + 5xx capture handler
- Edit: `apps/api/routes/routes.go` — register Sentry middleware *first* in the chain (before `gin.Default()` recovery)
- New k8s ExternalSecret entry: `SENTRY_DSN_API`, `SENTRY_ENVIRONMENT`

**Code shape:**

```
services/sentry.go:
  InitSentry() error            // reads config.AppConfig.Sentry{DSN,Environment,Release,TracesSampleRate}
                                // BeforeSend hook calls scrubPII (strips Authorization, X-Auth-Token,
                                // cookies, email/phone query params; reuse services/pii_filter.go)
                                // returns nil + log warn if DSN empty (fail-open boot)
  FlushSentry()                 // sentry.Flush(2s) — defer in main

middleware/sentry.go:
  SentryMiddleware() gin.HandlerFunc
    - clone hub per-request, attach c.Request + userID (if set on context)
    - defer recover() → hub.RecoverWithContext + AbortWithStatus(500)
    - after c.Next(), if c.Writer.Status() >= 500: hub.CaptureMessage("<status> on <path>")
```

Wiring:

- `main.go`: call `services.InitSentry()` right after `config.Load()`; `defer services.FlushSentry()`.
- `routes.go`: switch `gin.Default()` → `gin.New()` (we own recovery now) and register `middleware.SentryMiddleware()` as the very first `r.Use(...)`, before Prometheus/CORS/security-headers.

**Sampling:** `TracesSampleRate: 0.1`. **All errors (panic + 5xx) captured unsampled.** At ~50 req/s and 0.1% error rate that's ~4.3k events/mo, under the 5k cap. If we breach: drop traces to 0, keep errors.

**Validation:**

- `POST /__test/panic` (dev-mode-only handler) → event in Sentry within 30s.
- Force a 5xx → event with level=error and request context.
- Confirm `Authorization` header is scrubbed in the captured payload.

**Effort:** 4–6 hours (4 if `services/pii_filter.go` patterns drop in, 6 if extended).

**Risks + mitigations:**

- PII leakage → `BeforeSend` scrubs Authorization, X-Auth-Token, cookies, email/phone in query. Reuse `pii_filter.go`. Unit test required.
- Free-tier overrun → Prometheus `homechef_sentry_events_total` counter + alert at 4k/mo.
- Init failure → non-fatal log, service still boots (same posture as Redis/NATS).

---

### 4.2 Rate-limiting middleware (Redis-backed)

**Goal:** A 70-req/min flood from one IP gets HTTP 429 after the first 60. Per-user (chef) limit for authenticated mutation traffic; per-IP for unauthenticated.

**Touched / new files:**

- New: `apps/api/middleware/ratelimit_redis.go` (do NOT delete the existing in-memory `ratelimit.go` — keep it for the `/webhooks/*` and `/auth/*` groups, which want process-local limits regardless of Redis health)
- Edit: `apps/api/routes/routes.go` — register new middleware on the `v1` group with skip-list

**Algorithm:** Redis token bucket via Lua `EVAL` (atomic). Per Redis cluster, one round-trip per request. Lua script returns `(allowed bool, retry_after_seconds int)`.

```
KEY: rl:{scope}:{id}            scope = "ip" | "chef", id = "1.2.3.4" or "<userID>"
TTL: ceil(window_seconds)
SCRIPT: token-bucket — atomic check + decrement + EXPIRE
LIMITS:
  chef:    60 req / 60s, burst 80
  unauth:  30 req / 60s, burst 50  (keyed by IP via the same X-Forwarded-For logic in middleware/ratelimit.go:63)
```

**Code shape:**

```
middleware/ratelimit_redis.go:
  type RedisLimitConfig struct { Rate int; Burst int; Window time.Duration }
  RedisRateLimit(cfg, scope string) gin.HandlerFunc
    - key = "rl:" + scope + ":" + resolveKey(c, scope)
      // scope="ip"   → resolveKey uses clientIP() from existing ratelimit.go:63
      // scope="chef" → resolveKey reads c.Get("userID"), falls back to clientIP
    - EvalRateLimit(ctx, key, cfg) → calls token-bucket Lua via redis.EVAL, atomic
    - on redis error: redisFailures.WithLabelValues("ratelimit","eval").Inc(),
      set "X-RateLimit-Degraded: true", c.Next() (fail open)
    - on !allowed: set Retry-After, abort 429
    - skipPaths checked at top; constant map in same file
```

Route registration (in `routes.go`):

- Per-IP gate on `v1` group before `bffAuth`: `RedisRateLimit({Rate:30,Burst:50,Window:60s}, "ip")`.
- Per-chef gate after `bffAuth` on each authenticated subtree (`chefDashboard`, `chefMenu`, `chefOnboarding`, `orderPayments`, `chefSubscription`, `chefPromotion`): `RedisRateLimit({Rate:60,Burst:80,Window:60s}, "chef")`.

**Exclusions:** `/health*`, `/metrics`, `/api/v1/mobile/min-version`. One `skipPaths` map at top of middleware file.

**Validation:**

- Flood 80x against `/api/v1/locations/countries` → ~50 of `200`, rest `429` with `Retry-After: 60`.
- Authed chef → 70x against any `/api/v1/chef/*` → first 60 `200`, rest `429`.
- Kill Redis → all return `200` with `X-RateLimit-Degraded: true`; counter `homechef_middleware_redis_failures_total{middleware="ratelimit"}` increments.

**Effort:** 6–8 hours (Lua + tests + skip-list).

**Risks + mitigations:**

- Hot-key contention → token-bucket is single-key per client; chef traffic shards naturally. Monitor Redis CPU.
- Skip-list drift → centralize in one `var rateLimitSkipPaths` constant; reviewers grep.
- IP spoofing via XFF → reuse `clientIP()` in `middleware/ratelimit.go:63`; Gin `SetTrustedProxies` at `routes.go:89` already locks XFF to loopback/private.

---

### 4.3 Idempotency-Key middleware

**Goal:** A mobile client that retries a `POST /chef/orders/.../status` after a network blip does not double-process the request. The original response is replayed.

**Touched / new files:**

- New: `apps/api/middleware/idempotency.go`
- Edit: `apps/api/routes/routes.go` — apply to chef mutations

**Scope:** `POST`, `PUT`, `PATCH` on:

- `/api/v1/chef/orders/*`
- `/api/v1/chef/menu/*` (includes `/menu/items/:itemId/availability`)
- `/api/v1/chef/documents/*` (covers `POST /chef/documents`)
- `/api/v1/payments/order/:orderId/*` (the `create` / `verify` / `refund` triplet)

If the header `Idempotency-Key` is missing, **pass through** (mobile sends one, web/admin may not — we don't force the world).

**Key format & Redis layout:**

```
KEY: idem:{userID}:{path-with-params-resolved}:{idempotencyKey}
TTL: 24h
VALUE: gob/json {
    status int
    headers map[string]string
    body    []byte
    requestHash string  // sha256 of req body, for collision detection
}
```

**Flow:**

```
1. Read Idempotency-Key header. If empty → next.
2. Look up KEY in Redis.
   - HIT + same requestHash → replay stored response (200/4xx/5xx) verbatim.
   - HIT + different requestHash → 409 {"error": "Idempotency-Key reused with different body"}.
   - MISS → continue to handler.
3. After handler runs, capture response (status + headers + body),
   store in Redis with 24h TTL.
4. On Redis error at step 2 or step 3 → log + counter + X-Idempotency-Degraded: true + pass through.
```

**Code shape:**

```
middleware/idempotency.go:
  type storedResp struct { Status int; Headers http.Header; Body []byte; RequestHash string }
  Idempotency() gin.HandlerFunc
    - skip if header empty OR method not in {POST,PUT,PATCH}
    - read+rebuffer body, compute sha256 reqHash, build redisKey = "idem:{userID}:{c.FullPath()}:{key}"
    - GET redisKey:
        hit + matching reqHash → replay stored Status/Headers/Body, abort
        hit + different reqHash → 409 "Idempotency-Key reused with different body"
        miss → wrap c.Writer with responseRecorder, c.Next(), then SET on 2xx only (24h TTL)
        redis error → counter inc, "X-Idempotency-Degraded: true" header, c.Next() (fail open)
    - hard cap captured body at 1MB; refuse to cache larger + log
```

Route registration: `idem := middleware.Idempotency()` then `chefDashboard.Use(idem); chefMenu.Use(idem); chefOnboarding.Use(idem); orderPayments.Use(idem)`. `chefOnboarding` covers `POST /chef/documents`.

**Validation:**

- `PUT /chef/menu/items/<id>/availability` twice with same `Idempotency-Key` → both `200`, `menu_item.updated_at` unchanged on second call.
- Same key, different body → `409`.
- Kill Redis → both execute, `X-Idempotency-Degraded: true` set.

**Effort:** 8–10 hours (response-capture wrapper + tests are the long pole).

**Risks + mitigations:**

- Large bodies → 1MB cap on captured response; refuse and log over.
- Streaming/SSE → only `POST/PUT/PATCH` covered; WS (GET upgrade) excluded by method filter.
- Lock-out on transient failures → only `2xx` stored; 4xx/5xx skipped so client can re-submit.

---

### 4.4 Webhook HMAC verification (middleware-level)

**Goal:** No webhook handler can be added in the future without HMAC verification. Reject mismatches with 401 + Sentry alert.

**Current state audit:**

- `POST /webhooks/razorpay` — verifies in handler (`payment.go:573`). Good but in-handler.
- `POST /webhooks/stripe` — verifies in handler (`payment.go:758`). Good but in-handler.
- `POST /webhooks/delivery/:provider` — **does NOT verify HMAC** (`delivery_provider.go:611`). Currently any caller can POST a fake provider webhook. Critical fix.

**Touched / new files:**

- New: `apps/api/middleware/webhook_hmac.go`
- Edit: `apps/api/routes/routes.go` — register per webhook
- Edit: `apps/api/handlers/payment.go` — strip the in-handler HMAC check (it becomes a no-op since middleware runs first; keep for defense-in-depth or remove for clarity — pick one and document)
- Edit: `apps/api/handlers/delivery_provider.go:611` — read raw body via `c.GetRawData()` (already does) and trust middleware result

**Code shape:**

```
middleware/webhook_hmac.go:
  type HMACConfig struct {
      SecretFn  func(c *gin.Context) ([]byte, error)  // per-provider secret resolution
      Header    string                                // "X-Razorpay-Signature" | "Stripe-Signature" | "X-Webhook-Signature"
      Algorithm string                                // "sha256" | "stripe-v1" (t=,v1= with 5min tolerance)
  }
  WebhookHMAC(cfg HMACConfig) gin.HandlerFunc
    - read+rebuffer body so handler can re-read via c.GetRawData()
    - resolve secret via cfg.SecretFn; empty/err → 401 "webhook unconfigured"
    - verify(algorithm, secret, body, c.GetHeader(cfg.Header)); mismatch → sentry.CaptureMessage + 401
    - dev escape hatch: if ENVIRONMENT != "production" AND DEV_ALLOW_UNSIGNED_WEBHOOKS=true, log + pass
```

Route registration uses three pre-built `HMACConfig` instances:

- **Razorpay** — `SecretFn` calls `services.GetRazorpay().WebhookSecret()` (new getter; today only `HasWebhookSecret()` exists). `Algorithm: "sha256"`.
- **Stripe** — `SecretFn` returns `config.AppConfig.StripeWebhookSecret`. `Algorithm: "stripe-v1"` (port the algorithm from existing `services/stripe.go:VerifyStripeWebhookSignature` verbatim).
- **Delivery provider** — `SecretFn` looks up `DeliveryProvider.WebhookSecret` by `c.Param("provider")` (column already exists per `handlers/delivery_provider.go:332`). `Algorithm: "sha256"`.

Wire as `r.POST("/webhooks/razorpay", webhookLimit, razorpaySig, paymentHandler.RazorpayWebhook)` etc. (chain inserted between rate-limit and the handler). Remove or downgrade-to-comment the in-handler signature check in `payment.go:573` and `:758` — middleware-only is the contract.

**Pre-merge grep:** `grep -rn "POST.*webhook\|/webhooks" apps/api/routes/ apps/api/handlers/` must show only the 3 routes above. Any 4th MUST have HMAC middleware.

**Validation:**

- Unsigned `POST /webhooks/razorpay` → `401`.
- Valid HMAC → `200`.
- Garbage signature → `401` + Sentry event with `path` tag.
- Same drill for `/webhooks/stripe` and `/webhooks/delivery/:provider`.

**Effort:** 4–6 hours.

**Risks + mitigations:**

- Body double-read → middleware re-buffers; handlers continue to read via `c.GetRawData()`. Test by replaying Stripe CLI payloads.
- Stripe replay window → reuse the 5-min tolerance already in `services/stripe.go`.
- Dev unset secret → `DEV_ALLOW_UNSIGNED_WEBHOOKS=true` escape hatch, refused in production.

---

### 4.5 `GET /api/v1/mobile/min-version` + `X-App-Version` middleware

**Goal:** Mobile clients can fetch the minimum supported version on launch and get hard-walled with a store-link if they're below. Backend can flip the floor by editing an env var (and restarting the pod) without a code deploy.

**Touched / new files:**

- New: `apps/api/handlers/mobile.go`
- New: `apps/api/middleware/app_version.go`
- Edit: `apps/api/config/config.go` — add 6 env vars (3 versions × 2 platforms = 6, technically 12 with `STORE_URL_*`; bundle into a struct)
- Edit: `apps/api/routes/routes.go` — register handler + middleware

**Config additions (in `config.go`):**

```go
// MobileVersions holds per-platform, per-app distribution metadata.
// Sourced from env vars; v2 will move to a config table for no-deploy bumps.
type MobileVersions struct {
    VendorIOS     VersionInfo
    VendorAndroid VersionInfo
}
type VersionInfo struct {
    Min, Latest, StoreURL string
}
```

Env vars: `MIN_VERSION_VENDOR_IOS`, `LATEST_VERSION_VENDOR_IOS`, `STORE_URL_VENDOR_IOS` (× android = 6). For W1, only `vendor` app is supported; customer + delivery added in later waves.

**Handler shape:**

```
handlers/mobile.go:
  (h *MobileHandler) GetMinVersion(c)
    - validate query: app == "vendor"; platform in {ios, android}; else 400
    - resolve config.AppConfig.MobileVersions.{VendorIOS|VendorAndroid}
    - set Cache-Control: public, max-age=300
    - return {minVersion, latestVersion, storeUrl}
```

**Middleware shape:**

```
middleware/app_version.go:
  var appVersionSkipPaths = {/health*, /metrics, /api/v1/mobile/min-version}
  AppVersionCheck() gin.HandlerFunc
    - skip if path in skipPaths OR strings.HasPrefix(path, "/webhooks/")
    - read X-App-Version + X-Platform; if EITHER missing → pass (not a mobile client)
    - use golang.org/x/mod/semver with "v" prefix to Compare clientVer vs info.Min
    - on <: abort 426 with {error, minVersion, storeUrl}
```

Register globally on `r` (after Sentry, before rate-limit). Reads no Redis, no DB — pure config + header parse.

**Validation:**

- `GET /api/v1/mobile/min-version?platform=ios&app=vendor` → expected JSON shape + cache header.
- `X-App-Version: 0.9.0 + X-Platform: ios` on a guarded endpoint → `426`.
- `X-App-Version: 1.0.5` → `200`.
- No headers (web/curl) → `200`.

**Effort:** 3–4 hours.

**Risks + mitigations:**

- Semver edges → `golang.org/x/mod/semver` (indirect dep already); prefix `v` before `Compare`.
- Mid-shift lockout → only bump `MIN_VERSION_*` after announcement window. Wave 4 runbook entry.
- CDN caching → Cloudflare honors `max-age=300`. Fine.

---

### 4.6 Dependabot triage (82 → 0 critical/high)

**Goal:** Clear the 82 open Dependabot vulns to the point where flipping the Trivy gate (next item) doesn't break the build.

**Strategy:** Critical → High → Moderate. One PR per severity tier per package family.

**Workflow:**

1. `gh api repos/:owner/:repo/dependabot/alerts --paginate -q '.[] | {n:.number, s:.security_advisory.severity, p:.security_vulnerability.package.name, e:.security_vulnerability.package.ecosystem}' | sort -u` → spreadsheet.
2. Group by `(ecosystem, package)`. Tier order: critical → high → moderate (low optional).
3. Per tier: `git checkout -b deps/<tier>`, pull Dependabot PRs locally, `go mod tidy`, `go test ./...`, squash into one PR.

**Per-PR acceptance:** Dependabot alerts auto-closed on merge; tests pass; CI green (still warn-only Trivy); smoke-test changed-transitive areas (Razorpay/Stripe/GCS clients = highest blast radius).

**Effort:** 8–16 hours total. Variable on `gin`/`gorm`/`golang-jwt` breaking changes.

**Risks + mitigations:**

- Major-version bumps with breaking APIs (e.g., `gorm` v1→v2) → file backlog item, use `replace` directives as temporary patch.
- JS/mobile deps → out of scope; mobile subagent owns `package.json`.
- Keep auto-merge OFF; manual merge after CI green.

---

### 4.7 Trivy gate flip

**Goal:** Builds fail when a new CRITICAL or HIGH CVE lands in our container image. Catches the next 82-vuln drift before it accumulates.

**Touched / new files:**

- Edit: `.github/workflows/homechef-api-build.yml` line 113

**Change:**

```yaml
# Before
exit-code: '0'      # warn-only

# After
exit-code: '1'      # fail the build on CRITICAL or HIGH
```

Also remove the SARIF upload `continue-on-error: true` workaround if we keep the SARIF step. The `severity: CRITICAL,HIGH` filter and `ignore-unfixed: true` flag stay as-is.

**Sequencing:**

- **Hard dependency:** do NOT merge this until §4.6 has closed all CRITICAL and HIGH alerts. Otherwise the very next build red-X's prod deploys.

**Validation:**

- After flip, push a no-op commit. `gh run watch <run-id>`. Build should pass green.
- Re-introduce a known-vulnerable package version on a throwaway branch (`github.com/dgrijalva/jwt-go v3.2.0` is reliably CVE-laden) and confirm the build fails with Trivy output naming the CVE.

**Effort:** 30 minutes (the flip itself) + ~1 hour validation.

**Risks + mitigations:**

- *Trivy false positives blocking emergency hotfix* — fallback is to revert the YAML change in a single-line PR. Document in the runbook.
- *Base image drift* — `base-image-refresh.yml` workflow exists; ensure it runs weekly so base CVEs get bumped before they're flagged.

---

## 5. Sequencing within the track

| # | Item | Rationale |
|---|---|---|
| 1 | Sentry (§4.1) | First, so the next 6 items' failures land in Sentry. Independent, low-risk. |
| 2 | Webhook HMAC (§4.4) | Closes open hole on `/webhooks/delivery/:provider`. Independent. |
| 3 | Force-upgrade endpoint + version check (§4.5) | Unblocks mobile track. Independent. |
| 4 | Rate-limit (§4.2) | Soft-coupled to idempotency (same chain). Land first to verify skip-list + chef-key. |
| 5 | Idempotency (§4.3) | Right after §4.2 in a separate PR. |
| 6 | Dependabot triage (§4.6) | Long-running, parallel with items 1–5 (no middleware touch). |
| 7 | Trivy gate flip (§4.7) | **Last.** Hard-blocked by §4.6 critical+high being clean. |

**Parallelism:** items 1, 2, 3 = three concurrent branches. Items 4 + 5 sequential. Item 6 alongside everything. Item 7 closer.

---

## 6. Acceptance criteria mapped to PROD-READINESS Wave 1 DoD

From `PROD-READINESS.md` "Definition of done", backend-track only:

| DoD line | Verification |
|---|---|
| Panic shows up in Sentry within 30s | `POST /__test/panic` → Sentry event within 30s. Repeat on auth-bff after follow-up PR. |
| 70-req/min flood from one IP → 429 after the first 60 | `seq 1 80 \| xargs -P 8 curl ...` against an unauth endpoint → ~50 of `200`, rest `429`. (Note: unauth limit is 30/min burst 50 per the locked decision — the DoD's "60" applies to the chef-keyed path; verify both in the PR.) |
| `gh run list` clean; Trivy gate fail-on-critical | Most recent build green; `exit-code: '1'` in `homechef-api-build.yml`. |

**Additional backend-only acceptance:**

- All 3 webhooks reject unsigned/bad-sig with `401`; rejection emits a Sentry event tagged with the path.
- `/api/v1/mobile/min-version?platform=ios&app=vendor` returns the contracted shape and `Cache-Control: public, max-age=300`.
- `X-App-Version: 0.0.1` on any non-excluded endpoint → `426`.
- Idempotent retry of `PUT /chef/menu/items/<id>/availability` with same key → no double mutation (`updated_at` unchanged on second call).
- Redis off: requests still serve with `X-*-Degraded: true`; `homechef_middleware_redis_failures_total` increments.
- Dependabot {critical, high} count on backend module = 0.

---

## 7. Rollback strategy

Every middleware ships behind an env-var kill switch (`if os.Getenv("DISABLE_X") == "true" { c.Next(); return }` at the top). Flip via `kubectl set env deploy/homechef-api ... -n homechef`; new pod rolls in ~30s.

| Component | Disable | Notes |
|---|---|---|
| Sentry | Unset `SENTRY_DSN_API`, restart | Init logs warning, skips |
| Rate-limit | `DISABLE_RATELIMIT=true` | Middleware short-circuits |
| Idempotency | `DISABLE_IDEMPOTENCY=true` | Same pattern |
| Webhook HMAC | `DISABLE_WEBHOOK_HMAC_{RAZORPAY,STRIPE,DELIVERY}=true` | Last resort; leaves auth gap, prefer revert |
| `X-App-Version` check | `DISABLE_APP_VERSION_CHECK=true` | Endpoint stays up |
| `/mobile/min-version` | Set `MIN_VERSION_VENDOR_IOS=0.0.0` to lower the floor | Can't disable mid-flight (clients fetch on launch) |
| Trivy gate | Revert the one-line PR | <5 min |

**Nuclear rollback:** Argo CD UI → roll Knative back to `homechef-api-00076` (current good per `NEXT-SESSION.md`). Discards all Wave 1 middleware in ~30s.

---

## 8. Open questions / blockers

1. **Dependabot severity breakdown unknown.** `NEXT-SESSION.md` says 82 open, no split. **Action before §4.6:** `gh api repos/:owner/:repo/dependabot/alerts --paginate -q '.[].security_advisory.severity' | sort | uniq -c`. If `critical + high > ~20`, §4.6 effort doubles and §4.7 risks slipping out of Wave 1. **This is the top blocker that needs user input.**

2. **Razorpay webhook secret getter** — `services/razorpay.go` exposes only `HasWebhookSecret()` (bool). Middleware needs the raw value: add `WebhookSecret() string` getter. Trivial, but call out in PR so reviewer doesn't flag the new export.

3. **Stripe webhook secret** lives in `config.AppConfig.StripeWebhookSecret` (env var, not Secret Manager). Inconsistent with Razorpay's SM-first pattern. **Recommendation: leave** — Stripe is secondary; SM migration is Wave 4 cleanup.

4. **Delivery-provider webhook secret** — plaintext column on `delivery_providers.webhook_secret` (already exists per `delivery_provider.go:332`). Acceptable v1; flag for Wave 3 `pgcrypto` hardening.

5. **Sentry release tagging** — inject `SENTRY_RELEASE=<git-sha>` from CI in `homechef-api-release.yml`. 15-min addition; ship in Sentry PR.

6. **Local dev webhook testing** — no real secret available. Use `DEV_ALLOW_UNSIGNED_WEBHOOKS=true` escape hatch; middleware refuses to honor it when `ENVIRONMENT=production`.
