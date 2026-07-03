---
gsd_state_version: 1.0
milestone: v1.0
milestone_name: milestone
status: executing
stopped_at: Phase 4 context gathered
last_updated: "2026-04-06T09:58:27.872Z"
last_activity: 2026-04-06
progress:
  total_phases: 4
  completed_phases: 4
  total_plans: 19
  completed_plans: 19
  percent: 100
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-04-05)

**Core value:** Customers can browse home chefs, order food, and track delivery on their phones — while chefs manage orders and drivers navigate deliveries natively.
**Current focus:** Phase 04 — GPS, Push + Polish

## Current Position

Phase: 04
Plan: Not started
Status: Executing Phase 04
Last activity: 2026-06-17 - Completed quick task 260617-mh6: DPDP export/delete tests (#12); branch test/dpdp-export-delete-issue-12, PR pending

Session continuity (2026-06-17 pm): repo `tesserix/Home-Chef-App` made **PUBLIC** (to get free GitHub Actions — org Actions billing was blocking all CI). Deployed `auth-bff` (registry cleanup) + `api` (social-profile/email_verified) via the CI→GHCR→tesserix-k8s argocd bump→ArgoCD chain — required refreshing the expired `TESSERIX_K8S_BOT` PAT (see reference_deploy_pipeline_and_repo_public memory). Going public unleashed Dependabot: **auto-merge PAUSED** (`dependabot-auto-merge.yml` disabled) after it merged a partial expo-updates 56 bump. SDK 56 upgrade done on local main but **NOT pushed** — owner must run a native build (Xcode ≥26.4) first, then push. Redundant dependabot expo-* PRs to close: #79/#84/#85/#86/#87. Android prod builds (customer e6bcc99e + vendor 8c868fa3, versionCode 2) submitted to Play internal earlier. iOS #9 closed (TestFlight launch verified).

Progress: [░░░░░░░░░░] 0%

## Performance Metrics

**Velocity:**

- Total plans completed: 19
- Average duration: —
- Total execution time: 0 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01 | 4 | - | - |
| 02 | 5 | - | - |
| 03 | 6 | - | - |
| 04 | 4 | - | - |

**Recent Trend:**

- Last 5 plans: —
- Trend: —

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Phase 1: Use raw FCM tokens (`getDevicePushTokenAsync`) not Expo Push Tokens — existing `push.go` calls FCM directly and the two token types are incompatible
- Phase 1: Must verify `@tesserix/native` peer deps and current Expo SDK stable version (52 vs 53) before pinning versions — do not scaffold until confirmed
- Phase 1: Each app needs `metro.config.js` with `watchFolders` and `resolver.nodeModulesPaths` — pnpm symlinks are invisible to Metro by default
- Phase 1: GPS location watcher must use `distanceInterval: 20–50m` and `timeInterval: 5–10s` to avoid flooding `db-f1-micro` (architectural constraint)
- Phase 4: Background location permission must be gated on driver going online, not on first launch — App Store rejection risk otherwise

### Pending Todos

- Dark mode for mobile apps (from 2026-06-10 vendor UX critique): `.impeccable.md` requires "light-first, dark supported" but `packages/mobile-shared/src/theme/tokens.ts` has no dark palette and all screens use static StyleSheets. Phase-sized: dark token set + theme-reactive styling across all 3 apps. Promote via /gsd-add-phase.
- Bulk availability toggle for vendor menu categories (from same critique) — needs UX decision on placement.

### Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 260610-grk | Vendor app UX improvements from design critique | 2026-06-10 | 2dee223 | [260610-grk-vendor-app-ux-improvements-from-design-c](./quick/260610-grk-vendor-app-ux-improvements-from-design-c/) |
| 260610-ica | Vendor app v2 visual redesign (canvas+cards, dark hero, motion) | 2026-06-10 | f2a5814 | [260610-ica-vendor-app-v2-visual-redesign-canvas-car](./quick/260610-ica-vendor-app-v2-visual-redesign-canvas-car/) |
| 260610-iy5 | Customer app v2 Airbnb redesign + data-layer fixes; checkout address mapping fix + per-item special instructions | 2026-06-10 | 55258a5 | [260610-iy5-customer-app-v2-redesign-plan-airbnb-pal](./quick/260610-iy5-customer-app-v2-redesign-plan-airbnb-pal/) |
| 260612-n0g | Polish batch: landing wordmark swap + SVG logo + mobile splash/adaptive-icons + doc-drift checkbox sync | 2026-06-12 | 111084f | [260612-n0g-polish-batch-landing-wordmark-swap-mobil](./quick/260612-n0g-polish-batch-landing-wordmark-swap-mobil/) |
| 260615-i1h | Legal docs → reviewer-ready: full landing policies + vendor legal screens + customer/vendor EULA + consistency sweep + COUNSEL-REVIEW.md; entity reconciled to Tesserix Pty Ltd (mark8ly parent precedent) | 2026-06-15 | bd26470 | [260615-i1h-finish-prod-readiness-point-2-legal-docs](./quick/260615-i1h-finish-prod-readiness-point-2-legal-docs/) |
| fast | Customer app iOS-submission config — privacy manifest (5 data types, no crash/perf) + ITSAppUsesNonExemptEncryption=false + eas submit appleTeamId/ascAppId | 2026-06-15 | 22cf787 | _(inline /gsd-fast)_ |
| fast | Customer app EAS Update OTA wiring — expo-updates@~55.0.24 + updates.url + appVersion runtimeVersion (now matches vendor) | 2026-06-15 | 4ff472c | _(inline)_ |
| fast | EAS Workflow CI scaffold (production iOS+Android build+submit, both apps, inert until creds) | 2026-06-15 | e7c5ca3 | _(inline)_ |
| fast | Bundle-ID migration → com.tesserix.homechef.{vendor,customer} (com.homechef.* taken globally); app.json+eas.json+new Firebase configs+iOS OAuth clients, verified | 2026-06-15 | 4ed26ee, b211b6b | _(inline)_ |
| fast | Android production builds (both apps) green on EAS; vendor required disabling Sentry source-map upload (EU-region sentry-cli routing) + Play android submit profile + EAS Workflow CI scaffold | 2026-06-15 | _(EAS builds + eas.json)_ | _(inline)_ |
| fast | iOS: both apps built + **submitted to TestFlight**; Apple dist cert + ASC API key set up on EAS (interactive once, now CI-ready); appleTeamId(2CRHRRYBPL)+ascAppId hardcoded in submit profiles (eas submit doesn't interpolate $EAS_SECRET) | 2026-06-16 | _(EAS submit + eas.json)_ | _(inline)_ |
| fast | Android: both apps published to Play **internal testing** (first manual AAB upload, Play's rule); **Google Play service account** (eas-play-submit@tesseracthub-480811) wired to EAS for BOTH apps + verified via eas submit (auth OK, rejected only on dup versionCode). **Release pipeline now fully automated both stores** — eas submit/release.yml zero-touch; versionCode auto-increments | 2026-06-16 | _(EAS credentials/submit)_ | _(inline)_ |
| 260617-due | Social-login profile (name+avatar, backfill-only) via GIP claims through verifier→autologin→apiclient→API upsert; email_verified-gated same-email re-bind (anti-hijack hardening); Apple first-auth name→Firebase displayName; **fixed "profile edits don't persist" — mobile read-side `{data}` envelope mismatch** (customer useProfile/profile.tsx read flat; vendor/delivery already flat). Go tests green both services | 2026-06-17 | 5d8e3fc | [260617-due-social-profile-and-profile-save-fix](./quick/260617-due-social-profile-and-profile-save-fix/) |
| fast | Issue #22 auth-bff registry cleanup — dropped dead web OIDC entries (web/vendor-portal/delivery-portal) from `homechef-products.yaml`; kept admin-portal + mobileTenantAllowlist (mobile uses /auth/auto-login, not these app entries — ResolveByHost only in oidc handlers); oidc+registry tests repointed to admin-portal, `go test ./...` green. Owner confirmed web portals already down | 2026-06-17 | 0fd4e5c | _(inline /gsd-fast)_ |
| 260617-j7m | Issue #10 — automated Go tests for Wave 1 security middleware (apps/api): 46 test fns / 5 files + 1 redis test seam. Covers force-upgrade 426, rate-limit 429 + X-RateLimit-* + fail-open, idempotency replay-once + TTL + fail-open, webhook HMAC (razorpay/stripe/delivery) accept/reject. miniredis added (no skips). `go test ./... -race` + build green. On local main (test-only; pushing redeploys api harmlessly) | 2026-06-17 | (local) | [260617-j7m-security-middleware-tests-issue-10](./quick/260617-j7m-security-middleware-tests-issue-10/) |
| 260617-gmq | Expo SDK 55→56 upgrade (all 3 apps + mobile-shared) — fixes the broken split from auto-merged expo-updates 56. expo ~56.0.12 / RN 0.85.3 / React 19.2.3, single coherent lockfile (frozen install passes); vendor bottom-tabs type reconciled via expo-router/tabs; delivery newArchEnabled→true; SDK-56 config plugins added. Static verify green (expo-doctor 19/21, 0 NEW tsc errors, tests baseline). **NOT build-verified, NOT pushed** — native build (Xcode ≥26.4 / iOS 16.4, prebuild --clean) is the owner's remaining gate. Research+plan-check+validate run | 2026-06-17 | a81d234 | [260617-gmq-expo-sdk-55-to-56-upgrade](./quick/260617-gmq-expo-sdk-55-to-56-upgrade/) |
| 260617-lxn | Error-code UX (owner priority) — route auth errors through `resolveAuthErrorMessage` so users never see raw codes (`auto_login_502`, Firebase `auth/*`). **Scope finding: only mobile-vendor `(auth)/login.tsx` leaked** (4 `Alert.alert(..., err.message)` blocks); customer/delivery login + all register/forgot-password already delegate to shared screens that map+banner. Fixed vendor login (−12/+5). PR #100 MERGED. Follow-ups: align vendor to inline-banner pattern; app-wide non-auth `err.message` audit needs a generic resolver | 2026-06-17 | d987f8d | [260617-lxn-error-code-ux-replace-raw-err-message-wi](./quick/260617-lxn-error-code-ux-replace-raw-err-message-wi/) |
| 260617-mp6 | Fix vendor "~48 lucide type errors" — root cause was NOT lucide drift: `apps/mobile-vendor/types/expo-modules.d.ts` was a hand-written `declare module` stub file ("packages not yet in node_modules") that, post-SDK-56, **shadowed** the real installed types; its lucide stub declared only 16 icons → every other icon `TS2305`. All 5 stubbed pkgs now installed w/ real types → deleted the obsolete file (−139). Vendor tsc **48→5**; the 5 remainders are pre-existing `expo-router` TS2345 from a stale auto-generated `.expo/types/router.d.ts` (regenerates at build; on main too). Customer/delivery had no stub. PR #105 MERGED | 2026-06-17 | (merged) | [260617-mp6-fix-lucide-react-native-type-errors-in-m](./quick/260617-mp6-fix-lucide-react-native-type-errors-in-m/) |
| 260617-mh6 | Issue #12 — automated Go tests for DPDP export/delete (apps/api): 11 tests, +403 lines, test-only. Covers export (attachment+§11 notice+user block, sensitive-field omission, chef+menu inclusion, 404), delete (email-confirm gate, soft-delete+30d retention, chef hard-delete + menu offline cascade, PII-safe audit row), retry behaviour, `sanitizeUserForExport` unit. In-memory sqlite + hand-rolled tables (AutoMigrate breaks on `gen_random_uuid()`), global `database.DB` swap w/ t.Cleanup. `go test ./... -race` + build green. **Findings (not fixed): `already_deleted` branch unreachable** (default-scoped First hides soft-deleted → retry 404 not idempotent 200; fix=`.Unscoped()`); ChefProfile hard-deleted (no gorm.DeletedAt); handler log.Printf prints email. Branch `test/dpdp-export-delete-issue-12` → PR #102 | 2026-06-17 | (branch) | [260617-mh6-issue-12-automated-go-tests-for-dpdp-exp](./quick/260617-mh6-issue-12-automated-go-tests-for-dpdp-exp/) |
| 260703-e4y | Flat 6% runtime platform commission from PlatformSettings (`payout.commission_rate`, default 0.06) + premium reduced-rate tier removed; GST 18% / TDS 1% unchanged; money-conservation test at 6% (`commission+tds+netPayout==gross`). Economics half of payout ADR #441 / #390. Razorpay Route split at checkout untouched (lands with #387/#388 control-plane). `go build ./...` + services/handlers tests green. Branch `feat/payout-6pct-commission` → PR #445 | 2026-07-03 | 451824a | [260703-e4y-runtime-commission-6pct-flat-from-platfo](./quick/260703-e4y-runtime-commission-6pct-flat-from-platfo/) |
| 260703-fqs | GH #391 (P0 HIGH) — gate chef `→Delivered` by fulfilment type: `chef_delivery`/`pickup` allowed, `delivery`(3PL)+unset blocked 403 (deny-by-default) at single choke point before release side-effects; audit-log via `services.LogAudit`. Courier/webhook + payout-release untouched. TDD, go build + handlers/services tests green. Branch `fix/chef-delivered-fulfillment-gate` → PR #449. Follow-ups: OTP PoD, #387 payout-hook decoupling | 2026-07-03 | 8c911b2 | [260703-fqs-gate-chef-mark-delivered-to-self-deliver](./quick/260703-fqs-gate-chef-mark-delivered-to-self-deliver/) |
| 260703-g3x | GH #387 backend — payout **hold state machine** (`awaiting_customer_confirmation→release_eligible→released/disputed`) as fields on Order+MealPlanDay (AutoMigrate + audit SQL); **decouple delivery from release at ALL 5 completion paths** (provider/shadowfax/temporal-saga/delivery/chef + meal-plan chokepoint) — plan-check caught 3 missed LIVE 3PL sites; ReleaseOrderPayouts now zero prod callers (grep-gated). Customer confirm endpoints (owner-scoped, idempotent, dispute-gated) + `payout.customer_confirm_window_hours` (24). All behind escrow flags (no live change). TDD 17 tests, build/vet/gofmt/test green. Branch `feat/payout-hold-state-machine` → PR #452. Follow-ups: sweep+NATS events, #388 admin queue, group-order decouple, mobile UI | 2026-07-03 | 72194f2 | [260703-g3x-payout-hold-state-machine-customer-confi](./quick/260703-g3x-payout-hold-state-machine-customer-confi/) |
| 260703-hyd | GH #387 follow-up — payout **auto-confirm sweep** cron (`payout-auto-confirm`, 15m; advances awaiting→release_eligible past the 24h window, dispute-free only) + **NATS events** `payments.hold_release_eligible`/`hold_disputed` emitted inside the confirm transaction (RowsAffected>0 emit-once guard) so endpoints + sweep both fire. All behind escrow flags; no Razorpay/release call (that's #388). TDD, build/vet/gofmt/test green (incl. existing #387 confirm tests after shared applyHoldConfirm refactor). Stacked on #387 → PR base `feat/payout-hold-state-machine`, branch `feat/payout-auto-confirm-sweep`. Follow-ups: #388 admin queue, #400 freeze-linkage, group-order, mobile | 2026-07-03 | accb871 | [260703-hyd-payout-auto-confirm-sweep-cron-nats-hold](./quick/260703-hyd-payout-auto-confirm-sweep-cron-nats-hold/) |
| 260703-jef | GH #388 backend — admin payout **release queue actuator**: `GET /admin/payouts/pending` + `POST /admin/payouts/:aggType/:id/{release,withhold,reverse}` + `release-bulk`. Race-safe conditional flip (RowsAffected→409) BEFORE Razorpay `ReleaseTransfer`/`ReverseTransfer`; new terminal enum `withheld`/`reversed`; audit every action; auto-approve getter (default 0=manual-first). All behind escrow flags (no money when OFF). **Known gap (must-fix-before-flags-ON): post-commit seam failure → `released`-but-unpaid drift, no in-slice re-drive.** TDD, build/vet/gofmt/test green. Branch `feat/payout-admin-release-queue` → PR #455. Follow-ups: tesserix-home UI, auto-approve sweep, drift reconcile, #400 freeze, group-order | 2026-07-03 | f4610ef | [260703-jef-admin-payout-release-queue-backend-pendi](./quick/260703-jef-admin-payout-release-queue-backend-pendi/) |
| 260703-km0 | GH #459 — payout **drift reconcile**: `payout_settled_at`+`payout_settle_attempts` cols (stamped ONLY after the money seam returns nil, via shared `settleRelease`/`settleReverse`); `ReleaseDayPayout` idempotent (tolerates already-released); new `payout-reconcile` cron (10m) re-drives `released`/`reversed` holds with `payout_settled_at IS NULL`, flag-gated no-op when dark, bounded+attempt-capped(5)+ALERT. TDD, build/vet/gofmt/test green. Branch `feat/payout-drift-reconcile` → PR pending. Follow-up: FetchTransfer for true day idempotency. Surfaced by the 2026-07-03 payout audit (see #456-462) | 2026-07-03 | 32926d2 | [260703-km0-payout-drift-reconcile-settled-at-column](./quick/260703-km0-payout-drift-reconcile-settled-at-column/) |
| 260703-le5 | GH #456 (P0) — route **group-order payouts through the hold state machine + flag-gate** (was leaking live money: released chef transfer on delivery ungated). 3 group seams (Hold/Release/Reverse GroupChefPayout) now gated on `payoutMovementEnabled()`; `MarkGroupOrderDelivered` PARKS `awaiting_customer_confirmation` (parkGroupOrderOnDelivery) instead of releasing; new `group-order` aggType wired through release/reverse/sweep/reconcile/admin-queue; host `POST /group-orders/:id/confirm-received`. No phantom double-hold (consolidated order has no razorpay_order_id). TDD, build/vet/gofmt/`-race` test green. Branch `feat/group-order-payout-hold` → PR pending. Follow-up (before flags-ON): W-A cancel-reverse through ReverseHold; tesserix-home group row label+confirm UI | 2026-07-03 | b9a82cb | [260703-le5-group-order-payouts-through-hold-state-m](./quick/260703-le5-group-order-payouts-through-hold-state-m/) |
| 260703-m3c | GH #457 (P0) — **refund↔payout cross-guard** (was: order could be refunded AND paid to chef). New `WithholdOrReverseOrderHoldForRefund` (eligible/awaiting→withheld; released→reversed+reverseMoney) wired into all 5 order-refund sites (RefundIssueToWallet, CompensateOrderRefund, chef RefundOrder+CancelOrder, PaymentHandler.InitiateRefund ×2 commits). Release-side backstop: `ReleaseHold` pre-check + `listPendingOrders` exclude refunded/cancelled AND `refunded_at IS NULL` (issue path leaves status=delivered); all-aggregate `releaseBlockedForAgg` (order/day/group resolve underlying order + pending-issue). `HasOpenIssue` on queue DTO. TDD 14 tests, build/vet/`-race` green. Branch `feat/refund-payout-hold-crossguard` → PR pending. Follow-ups: CancelOrderItem/handleRefundProcessed webhook/RefundGroupParticipant→hold, day/group queue exclusion | 2026-07-03 | 0e5bf5e | [260703-m3c-refund-dispute-cross-guard-refund-drives](./quick/260703-m3c-refund-dispute-cross-guard-refund-drives/) |

### Blockers/Concerns

- **✅ RESOLVED on-device (2026-06-16): iOS launch crash + Google Sign-In + Apple Sign-In.** All three verified working on a local signed device build (iPhone 17 Pro), both apps. Arch-mismatch theory was WRONG (RN 0.83 is new-arch-only; `newArchEnabled` no-op). Fixes: (1) **launch crash** resolved in current code (crashed TestFlight builds were an older commit; sim couldn't reproduce). (2) **Google Sign-In** — stale pre-migration iOS OAuth client IDs in `.env.local` didn't match `GoogleService-Info.plist`/URL scheme; FIX commit dda168e: aligned `.env.local` iosClientId, added `@react-native-google-signin/google-signin` plugin `iosUrlScheme` to both app.json, added Google client IDs + `GIP_API_KEY` to `eas.json` production env. (3) **Apple Sign-In** — was `auth/operation-not-allowed`; FIX = enabled Apple provider in both GIP tenants (HomeChef-Customer-rqg8a + HomeChef-Business-8s8ql; iOS platform + bundle ID, server-side, no app change). **EAS production build #3 (vendor `fe77a977` + customer `3c45c479`) submitted to TestFlight** (carries launch+Google fixes; Apple works via the now-enabled GIP provider). REMAINING: confirm build #3 from TestFlight; run Android via `release.yml`. Detail in `project_ios_launch_crash_debug` + `project_apple_signin_gip_not_enabled` memories.

- **Sentry source-map upload (vendor) DISABLED** via `SENTRY_DISABLE_AUTO_UPLOAD=true` — EU-region (`tesserix` org, `de.sentry.io`) + sentry-cli/Expo-gradle can't reconcile control-silo auth (sentry.io) vs EU data region: `SENTRY_URL=de`→401 invalid token, no `SENTRY_URL`→"organization not found". Runtime crash reporting via DSN is UNAFFECTED. **Post-launch follow-up:** re-enable upload with correct EU endpoint config (or a region-matched token) for de-minified stack traces.
- Confirm Go API has `PATCH /v1/delivery/:id/location` endpoint before Phase 4 GPS work begins
- Confirm whether real-time driver location for Customer map is REST polling or WebSocket (v1 fallback: 5-second polling)
- Verify EAS Build free tier build minute limits for 3 apps before Phase 1 ends

## Session Continuity

Last session: 2026-06-16 — iOS auth debugging
Stopped at: all iOS auth fixed on-device (launch + Google + Apple Sign-In); EAS prod build #3 submitted to TestFlight (vendor `fe77a977` + customer `3c45c479`)

Next session TODO:
1. Confirm TestFlight build #3 (both apps) once Apple processing finishes — verify launch + Google + Apple sign-in from TestFlight on-device.
2. Ship Android: `eas workflow:run release.yml` (or `eas build -p android --profile production --auto-submit`) per app — same Google client-ID fixes already in `eas.json` apply.
3. Account linking by email across password/Google/Apple — backend/identity work, see `project_account_linking_by_email` memory (outside mobile "no backend changes" scope; needs scoping).
4. Local-build reminder: build locally for SIM/device only, no EAS during debug (see `feedback_local_sim_builds_only`); device builds need `SENTRY_DISABLE_AUTO_UPLOAD=true` for vendor + ad-hoc sign for sim.

Key context memories: `project_ios_launch_crash_debug`, `project_apple_signin_gip_not_enabled`, `reference_apple_eas_identifiers`.
