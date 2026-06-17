---
phase: quick
plan: 260617-j7m
subsystem: api / security-middleware
tags: [testing, go, security, rate-limit, idempotency, hmac, force-upgrade]
key-files:
  created:
    - apps/api/services/redis_testseam.go
    - apps/api/middleware/version_check_test.go
    - apps/api/middleware/ratelimit_test.go
    - apps/api/middleware/redis_helper_test.go
    - apps/api/middleware/idempotency_test.go
    - apps/api/services/razorpay_webhook_test.go
    - apps/api/handlers/delivery_webhook_hmac_test.go
  modified:
    - apps/api/go.mod
    - apps/api/go.sum
metrics:
  tests-added: 46
  completed: 2026-06-17
---

# Quick 260617-j7m: Security Middleware Tests (Issue #10) Summary

Unit-test coverage for the Wave 1 security hardening — force-upgrade gate (HTTP
426), Redis-backed rate limiting (per-IP / per-user budgets + fail-open),
Idempotency-Key dedup (replay / 24h TTL / fail-open / in-flight 409), and
webhook HMAC verification (Razorpay + delivery). Redis-backed paths are driven
against an in-memory miniredis server via a single justified test seam.

## Tests Added Per Area

### Force-upgrade gate — `middleware/version_check_test.go` (9 tests)
Fully covered, no env-gating or skips.
- Stale client → 426 with `minVersion` + store link (`STORE_URL_*` set).
- Stale client with `STORE_URL_*` unset → default Apple/Play fallback link.
- Current / newer client (incl. `+build` suffix) → pass-through.
- No mobile headers (web/admin/curl) → no-op.
- Unknown platform → no-op.
- Excluded path prefix → bypass even for a stale client.
- Pure helpers: `semverLess` (numeric-not-lexical, `v`-strip, `+build`-strip,
  non-numeric → 0), `parseSemver`, `minVersionFromEnv` (env + default).

### Rate limiting — `middleware/ratelimit_test.go` (14 tests)
Fully covered (Redis paths via miniredis).
- `clientIP`: X-Forwarded-For (multi + single), CF-Connecting-IP, RemoteAddr
  fallback.
- `toString` helper (string / unsupported / Stringer).
- In-memory `RateLimit`: throttle after burst (429 + `Retry-After: 60`),
  per-IP isolation.
- In-memory `RateLimitByUser`: keyed by user ID across IPs (`Retry-After: 30`).
- `RateLimitRedis`: unauth budget exhaustion → 429 with `X-RateLimit-*` headers
  and counting-down `Remaining`; authed budget used when `userID` set; excluded
  path bypass; zero-config defaults (unauth 30/min); window roll-over via
  miniredis `FastForward` past the 70s key TTL.
- **Fail-open:** Redis disconnected → every request passes, no rate-limit
  headers emitted.

### Idempotency — `middleware/idempotency_test.go` (13 tests)
Fully covered (Redis paths via miniredis).
- Pure helpers: `isMutation`, `matchesAnyPrefix`, `idempotencyCacheKey`
  (stable, body-sensitive, user-namespaced, `anon` for empty user).
- Replay of cached response with `Idempotent-Replayed: true` header and handler
  executing **exactly once** (no double-write / double-charge).
- Different body + same key → fresh execution (body hash in cache key).
- No header / non-included path / GET → pass-through (opt-in semantics).
- 4xx response IS cached; 5xx is NOT cached and pending marker cleared so a
  retry re-executes.
- 24h `ResponseTTL` expiry via miniredis `FastForward(25h)` → handler runs again.
- **Fail-open:** Redis disconnected → handler runs uncached.
- In-flight collision (pre-seeded `:pending` marker) → 409 "already in flight".

### Webhook HMAC — `services/razorpay_webhook_test.go` (5) + `handlers/delivery_webhook_hmac_test.go` (5)
Fully covered, pure functions, no Redis/DB.
- Razorpay `VerifyWebhookSignature` (in-package `services` test, sets the
  unexported `razorpayClient` global under its mutex with restore): valid sig,
  bad sig / empty sig, tampered payload, wrong secret, no-secret-configured
  (nil client + empty secret both reject — never fail-open on auth).
- Delivery `verifyHMACSHA256` (in-package `handlers` test): valid sig, bad sig,
  tampered body, wrong secret, empty signature/secret all reject.

## Coverage Status

| Area | Status |
| --- | --- |
| Force-upgrade gate (426) | Fully covered |
| Rate limit — in-memory (`RateLimit`, `RateLimitByUser`) | Fully covered |
| Rate limit — Redis (`RateLimitRedis`) incl. fail-open + roll-over | Fully covered (miniredis) |
| Idempotency — replay / TTL / 4xx-cache / 5xx-clear / in-flight / fail-open | Fully covered (miniredis) |
| Webhook HMAC — Razorpay | Fully covered (pure fn) |
| Webhook HMAC — delivery | Fully covered (pure fn) |

No paths are env-gated or skipped. The miniredis dependency was successfully
added (network available), so the `TEST_REDIS_ADDR`-gated fallback described in
the constraints was **not needed** — all Redis-backed paths run hermetically
in-process.

## Test Seam Added

`apps/api/services/redis_testseam.go` — `SetRedisClientForTest(*redis.Client) *redis.Client`.
The Redis-backed rate-limit and idempotency middleware resolve their client
through the `services.GetRedisClient()` singleton, whose underlying
`*redis.Client` is unexported and otherwise only set by `Connect()` against a
real Redis URL. The seam assigns an injected client (miniredis-backed in tests)
and returns the prior client so tests restore global state in a `defer`. This is
the single justified production-adjacent seam; the Razorpay webhook test instead
sets the unexported `razorpayClient` global directly from an in-package
(`package services`) test, and the delivery HMAC test calls the unexported
`verifyHMACSHA256` directly from an in-package (`package handlers`) test, so no
seams were needed there.

Shared test helper `middleware/redis_helper_test.go` (`startMiniredis`,
`clearRedisClient`) wires miniredis through the seam and restores state on
`t.Cleanup`. Tests run serially (no `t.Parallel`) since they share the
process-global Redis singleton.

## Deviations from Plan

The PLAN.md file was not present on disk in this worktree. Scope was
reconstructed from GitHub Issue #10 (force-upgrade / rate-limit / idempotency /
HMAC) and the task constraints, then implemented exactly as the constraints
specified (new test files + the one justified `SetRedisClientForTest` seam,
miniredis added with `go mod tidy`). No production code under test was modified.

## Final Verification

- `cd apps/api && go test ./... -count=1` → all packages `ok`
  (handlers, logger, middleware, services).
- `cd apps/api && go test ./middleware/... ./services/... -race -count=1` → `ok`
  (no data races on the shared Redis singleton).
- `cd apps/api && go build ./...` → clean.
- `gofmt -l` on all new files → clean.
- Working tree clean; 3 atomic single-line commits (no signatures):
  - `6630b3c test(api): force-upgrade gate + webhook HMAC verification (issue #10)`
  - `0b5befa test(api): rate-limit fail-open + per-IP/user budgets via miniredis seam (issue #10)`
  - `bf126f1 test(api): idempotency replay/TTL/fail-open/in-flight dedup (issue #10)`
- Dependency added: `github.com/alicebob/miniredis/v2 v2.38.0` (test-only).

## Self-Check: PASSED

All 7 created files present on disk; all 3 commit hashes present in `git log`.
