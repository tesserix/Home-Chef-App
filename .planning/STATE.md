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
Last activity: 2026-06-12 - Completed quick task 260612-n0g: polish batch (wordmark, splash/adaptive icons, doc drift, SVG logo)

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
