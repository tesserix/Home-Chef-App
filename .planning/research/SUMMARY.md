# Project Research Summary

**Project:** Home Chef Mobile Apps (Customer, Vendor, Delivery Driver)
**Domain:** Food delivery mobile applications — Expo/React Native
**Researched:** 2026-04-05
**Confidence:** MEDIUM (training data through August 2025; no live web access during research)

## Executive Summary

This project adds three Expo (React Native) mobile apps — Customer, Vendor/Chef, Delivery Driver — to an existing pnpm monorepo that already contains a Go REST API and four Next.js web portals. The recommended approach is Expo managed workflow with EAS Build, using expo-router for file-based navigation, and a new shared workspace package (`packages/mobile-shared/`) for typed API client code and domain hooks. This mirrors how the existing web apps share `@tesserix/web` and minimises the learning curve for the team, which already uses Zustand v5, TanStack React Query v5, axios, React Hook Form, and Zod across the web portals.

The largest risk is infrastructure correctness before any feature work begins. Four Phase 1 decisions are non-negotiable: Metro bundler symlink configuration for pnpm workspaces, EAS Build monorepo root configuration, push notification token type resolution (Expo Push Token vs raw FCM token — the existing `push.go` uses raw FCM and the two are incompatible), and distinct deep link URI schemes per app baked into each binary. Getting any of these wrong causes silent failures that are expensive to diagnose later and some (URI schemes, native permissions) cannot be patched via OTA update.

The standard Expo managed workflow pattern is well-established and high-confidence for all core feature work. The main unknowns are the exact Expo SDK version to target (verify SDK 52 vs 53 at scaffold time), the internals of the `@tesserix/native` package (peer dependency version must be checked before pinning Expo SDK), and whether the Go API currently exposes a delivery location endpoint or WebSocket for real-time tracking (polling at 5-second intervals is the safe v1 assumption). None of these gaps block starting development, but all must be resolved before Phase 1 ends.

---

## Key Findings

### Recommended Stack

The team should use Expo SDK 52 (verify — SDK 53 may be stable by April 2026) with expo-router v3+ for file-based navigation, React 18.3.1 (Expo SDK 52 ships with React 18; the web apps use React 19 — pnpm isolates these without conflict), and TypeScript 5.7.x. State management follows the existing web pattern exactly: Zustand v5 for session/UI state, TanStack React Query v5 for server state. Axios is the API client, wrapped in a factory in `packages/mobile-shared/`.

All Expo-managed packages must be installed via `npx expo install`, never `pnpm add`, to preserve SDK version compatibility. Background location (Driver app) and native payment UI (post-v1) are the only capabilities that require EAS Build from day one — the Customer and Vendor apps can be developed in Expo Go for most of the feature work.

**Core technologies:**
- Expo SDK 52 (managed workflow): Core React Native platform — eliminates macOS/Xcode on dev machines, EAS Build handles iOS builds from CI
- expo-router v3+: File-based routing — matches Next.js App Router mental model, automatic deep linking, typed routes
- react-native-maps ~1.18.x: Map display and delivery route overlays — only Expo-managed option with Google Maps parity on both platforms
- expo-location ~17.x + expo-task-manager: GPS coordinates and background location for Driver app — handles iOS/Android permission dialogs
- expo-notifications ~0.29.x: Push token registration, FCM/APNs abstraction
- expo-secure-store ~13.x: Encrypted JWT storage — iOS Keychain/Android Keystore; AsyncStorage is plaintext and must not hold tokens
- EAS Build + EAS Update: Cloud iOS/Android builds from CI; OTA JS-only updates for bug fixes between store releases
- Maestro: E2E testing — no native build changes required, runs against real app binary
- jest-expo: Unit/component testing, version must match SDK version

### Expected Features

Research identified features by combining what the web portals already have with mobile-native capabilities that users expect on mobile. Web parity features (browse chefs, cart, checkout, order history, vendor menu management, driver earnings) are assumed as baseline and are not listed below.

**Must have (table stakes):**
- Push notifications — order lifecycle (accepted, ready, picked up, delivered) for all three apps
- Push notifications — actionable accept/reject from lock screen for Vendor app (critical: Vendors must be able to act on orders without opening the app)
- Biometric authentication (Face ID / Fingerprint) — users refuse to type passwords on mobile
- Background GPS + live delivery tracking — Driver app cannot function without it; Customer expects to see driver on map
- Deep links from notification tap to order detail screen
- App icon badge counts (pending orders for Vendor, unread notifications for Customer)
- Secure token storage + silent token refresh on 401
- Social login — Google + Apple Sign-In (Apple Sign-In is App Store mandatory when any social login is offered)
- Camera access — food photo upload (Vendor), document upload (Driver onboarding), proof-of-delivery photo (Driver)
- Pull-to-refresh, skeleton loading states, graceful offline error state (no full offline mode)
- Haptic feedback at key moments (order placed, accept/reject, delivery complete)

**Should have (competitive):**
- Native navigation handoff — "Navigate" button opens Google Maps/Apple Maps/Waze with destination pre-loaded (safer and more accurate than in-app nav)
- Live order tracking with animated driver pin on Customer map (requires backend real-time location broadcasting — deferred until backend capability confirmed)
- Rating prompt (expo-store-review) shown after first delivery confirmation
- Shake-to-report issue (Driver app)
- Notification preference management (reduce opt-out rate)
- Reorder shortcut — one-tap from order history

**Defer (v2+):**
- Animated live driver pin on Customer map (backend WebSocket/real-time location broadcast not confirmed present)
- Chef home screen widget (iOS WidgetKit — high complexity)
- Scheduled push reminders for catering pre-orders (requires backend cron job)
- Native payment sheet (Apple Pay / Google Pay) — PROJECT.md explicitly deferred; use Razorpay web checkout via expo-web-browser for v1
- Full offline mode with sync

### Architecture Approach

Three independent Expo apps (`apps/mobile-customer/`, `apps/mobile-vendor/`, `apps/mobile-delivery/`) live alongside existing web apps in `apps/`. A new shared workspace package `packages/mobile-shared/` holds the typed API client factory, domain types, shared hooks (useAuth, useLocation, usePushNotifications), and secure storage utilities. The three mobile apps do not import from each other — all cross-app shared code is routed through `packages/mobile-shared/`. Each app follows expo-router's file-system conventions with a root layout that sets up providers and auth guards, then branches into `(auth)/`, `(onboarding)/`, and `(tabs)/` route groups. State follows the established web pattern: React Query for server state, Zustand for session/UI state.

The single most critical infrastructure piece is the Metro bundler configuration. Each mobile app needs `metro.config.js` with `watchFolders: [monorepoRoot]`, `resolver.nodeModulesPaths` pointing to both the app's and monorepo root's `node_modules`, and `resolver.unstable_enableSymlinks: true`. Without this, pnpm symlinks are invisible to Metro and the app crashes at launch with cryptic errors.

**Major components:**
1. `packages/mobile-shared/` — API client factory, domain types, auth hook, location hook, push notification hook, secure storage utils; build first, unblocks all app work
2. `apps/mobile-customer/` — Chef discovery, cart, checkout (Razorpay web checkout), order tracking map, social feed, favorites, catering
3. `apps/mobile-vendor/` — Live order queue with actionable push notifications, menu management with camera upload, earnings, kitchen settings
4. `apps/mobile-delivery/` — Available delivery assignments, active delivery GPS map, native nav handoff, proof-of-delivery camera, driver earnings
5. Go API (`apps/api/`) — No changes required for v1 except: FCM token column may need schema migration to a device_tokens join table; delivery location endpoint must support rate-limited updates

### Critical Pitfalls

1. **FCM token type mismatch** — The existing `push.go` calls FCM directly with raw device tokens. Expo's `getExpoPushTokenAsync()` returns an Expo Push Token (incompatible with FCM direct). Use `getDevicePushTokenAsync()` instead to get the raw FCM token, which works with the existing backend unchanged. Decide this in Phase 1 before any notification wiring.

2. **Metro symlink resolution failure** — pnpm's symlinked `node_modules` are invisible to Metro by default. Add `metro.config.js` with `watchFolders` and `resolver.nodeModulesPaths` to every mobile app before running `expo start` for the first time. Validate in Phase 1, not after feature work starts.

3. **EAS Build monorepo root not configured** — Local dev works but every CI build fails with "module not found" unless each app's `eas.json` sets `"projectRoot": "../.."`. Validate with `eas build --local` before the first remote build.

4. **GPS location polling floods the DB** — `watchPositionAsync` without throttling produces thousands of writes per hour against `db-f1-micro`. Use `distanceInterval: 20–50m` and `timeInterval: 5–10s` in the location watcher, plus server-side rate limiting on the location update endpoint. Architectural decision must be made in Phase 1, before GPS feature work.

5. **Apple App Store rejection for background location** — Requesting `Always` location permission on first launch (rather than gating it on "Go Online") causes App Store rejection with a 1–3 week re-review cycle. Request foreground permission first; escalate to background only when the driver initiates a delivery. Build this into the permission flow from the start — it cannot be OTA-patched.

6. **OTA updates cannot ship native changes** — Any new `expo-*` package, `app.json` permission, or plugin addition requires a new EAS Build and store submission. This affects background location (Driver), push notification entitlements, and camera permissions — all of which must be in the initial binary.

---

## Implications for Roadmap

Based on combined research, the following phase structure is recommended. The hard constraint is that Phase 1 must validate all infrastructure before any feature work begins — several pitfalls are silent and expensive to discover mid-feature.

### Phase 1: Foundation — Monorepo Scaffolding + Shared Infrastructure
**Rationale:** Every subsequent phase depends on Metro resolution, EAS Build, push token type, URI schemes, and the shared API client being correct. These are binary-level or near-binary-level decisions that cannot be patched via OTA. Discovering problems here costs hours; discovering them in Phase 3 costs weeks.
**Delivers:** Three scaffolded Expo apps with verified Metro config; `packages/mobile-shared/` with typed API client, auth hook, secure token storage, and push token registration; EAS Build validated with a hello-world build for each app; distinct URI schemes set; `@tesserix/native` peer dependency pinned.
**Addresses:** Secure token storage + auto-refresh (shared API client), push token infrastructure (raw FCM token vs Expo token decision), biometric auth setup (expo-local-authentication declared in app.json)
**Avoids:** Pitfalls 1 (FCM token mismatch), 2 (Metro symlinks), 4 (EAS Build monorepo root), 9 (URI scheme collision), 11 (OTA native changes), 15 (JWT refresh)
**Research flag:** Needs `/gsd-research-phase` — verify `@tesserix/native` peer deps and current Expo SDK stable version before pinning versions.

### Phase 2: Auth + Onboarding (All Three Apps)
**Rationale:** No feature screen can be built without a working auth flow. Build the Customer app auth first (simplest), validate the full login → secure token storage → onboarding → main tabs stack, then replicate the pattern to Vendor and Driver apps (same structure, different steps). Apple Sign-In entitlement must be configured in this phase — it is an EAS build-time entitlement.
**Delivers:** Login, registration, forgot password, biometric auth, Google Sign-In, Apple Sign-In, onboarding wizard, and auth guard in root layout for all three apps.
**Uses:** expo-local-authentication, expo-auth-session, expo-apple-authentication, expo-secure-store, React Hook Form + Zod
**Implements:** Auth guard pattern in root layout, token persistence in mobile-shared/utils/storage.ts
**Avoids:** Pitfall 5 (Apple background location — foreground permission rationale screens also established here as a pattern for later GPS work)

### Phase 3: Core Feature Screens (Customer + Vendor in Parallel)
**Rationale:** After auth works, Customer and Vendor core screens can be built in parallel — they share no state. Customer needs chef discovery, menu browsing, cart, and checkout. Vendor needs live order queue and menu management with camera upload. These are the highest user value features and represent the core commerce loop.
**Delivers:** Customer: chef grid, chef detail + menu, cart, Razorpay web checkout, order history, order detail. Vendor: live orders list, order detail with accept/reject, menu management with camera upload.
**Uses:** react-native-maps (static, no live tracking yet), expo-image-picker, expo-web-browser (Razorpay), TanStack React Query for server state
**Implements:** Query-first screen architecture (thin screens, hooks in mobile-shared)
**Avoids:** Pitfall 10 (Razorpay deep link — register return URLs in Razorpay dashboard and Android intent filters at phase start), Pitfall 13 (keyboard avoiding view on iOS vs Android — establish platform pattern on first form screen)

### Phase 4: GPS + Real-Time Tracking (Driver App, then Customer Map)
**Rationale:** Driver GPS requires background location, which is a native capability and must be included in the binary — it cannot be added via OTA. This phase requires physical device testing; simulators cannot test background GPS. Customer live tracking map is included here because it depends on driver location being published to the API.
**Delivers:** Driver: available deliveries, active delivery GPS map, native navigation handoff (Google Maps/Apple Maps deep link), background location tracking with battery-optimised polling. Customer: live order tracking screen with react-native-maps driver pin.
**Uses:** expo-location + expo-task-manager (background GPS), react-native-maps, react-native-maps-directions
**Implements:** GPS location service pattern (Pattern 4 from ARCHITECTURE.md), active-delivery-store Zustand store, customer order-tracking-store
**Avoids:** Pitfall 3 (GPS polling DB flood — distanceInterval + timeInterval + server rate limit), Pitfall 5 (Apple background location rejection — gate on "Go Online" action), Pitfall 7 (react-native-maps iOS Google Maps API key)
**Research flag:** Needs `/gsd-research-phase` — confirm whether Go API has a delivery location PATCH endpoint; confirm whether real-time driver location broadcasting (for Customer map) is REST polling or WebSocket. This determines implementation approach.

### Phase 5: Push Notifications + EAS Build CI
**Rationale:** Push notification actionability (Vendor lock screen accept/reject) depends on the notification categories being registered in the binary — another native-only capability. This phase also finalises the CI/CD pipeline and sets up TestFlight/Internal Testing, enabling beta distribution.
**Delivers:** Order lifecycle push notifications for all three apps; Vendor actionable notifications (accept/reject from lock screen); app icon badges; EAS Update (OTA) configured; GitHub Actions CI for EAS Build per app on path changes; TestFlight and Google Play Internal Testing configured.
**Uses:** expo-notifications, expo-device, GitHub Actions
**Implements:** Push notification data flow (push token registration at login → backend stores token → backend sends on order events → Expo relay or direct FCM)
**Avoids:** Pitfall 1 (FCM token type confirmed in Phase 1, wired here), Pitfall 6 (re-register token on foreground, handle FCM UNREGISTERED error in push.go)

### Phase 6: Polish + Differentiators
**Rationale:** With the full commerce loop working across all three apps, polish features and differentiators can be layered in without risk of blocking core functionality.
**Delivers:** Haptic feedback at key moments, pull-to-refresh, skeleton loading states, offline error states, driver shake-to-report, photo proof of delivery, reorder shortcut, App Store rating prompt.
**Uses:** expo-haptics, expo-sensors, @react-native-community/netinfo, expo-store-review
**Avoids:** Pitfall 14 (image caching — expo-image with blurhash should be in place from Phase 3, but confirmed here)

### Phase Ordering Rationale

- Phase 1 must precede everything: Metro config, EAS Build, push token type, URI schemes, and SDK version pinning are all binary-level decisions. Any rework here invalidates subsequent phases.
- Auth before features: No screen can function without working token storage and refresh. Apple Sign-In entitlement requires a binary rebuild — including it in Phase 2 avoids a later EAS Build cycle.
- Customer and Vendor features in parallel (Phase 3): The two apps share no runtime state. After auth is validated, their feature screens are independent.
- GPS deferred to Phase 4: Background location is a native capability requiring EAS Build and physical device testing. Deferring it avoids blocking the majority of feature work that can be done in Expo Go.
- Push notifications in Phase 5: Actionable notifications also require native binary capabilities. They are deferred until the core commerce screens are working so notification payloads have real order data to test against.
- This ordering means the team has a working Razorpay checkout loop (Customer + Vendor) before needing to solve the harder GPS and push notification infrastructure problems.

### Research Flags

Phases needing deeper research during planning:
- **Phase 1:** Verify current Expo SDK stable version (52 vs 53) and check `@tesserix/native`'s `package.json` for exact `expo` and `react-native` peer deps before any scaffolding. Also verify EAS Build free tier build minute limits for 3 apps.
- **Phase 4:** Confirm Go API delivery location endpoint exists (`PATCH /v1/delivery/:id/location`) and whether it supports rate limiting; confirm whether a real-time location broadcast mechanism exists for Customer tracking (REST polling vs WebSocket). These are architectural unknowns that affect implementation choices.
- **Phase 5:** Verify Expo Push Service latency is acceptable for Vendor new-order notifications. If latency exceeds 3–5 seconds, plan to use `getDevicePushTokenAsync()` with direct FCM (the backend already has `firebase.google.com/go/v4`).

Phases with standard, well-documented patterns (skip research-phase):
- **Phase 2:** Auth with expo-router + expo-secure-store + biometric auth is the canonical Expo auth pattern with abundant documentation.
- **Phase 3:** Cart, checkout, menu screens follow the query-first screen architecture pattern — standard React Query + Zustand usage, no novel patterns.
- **Phase 6:** Polish features (haptics, skeletons, pull-to-refresh) are all trivial Expo/React Native additions with no meaningful research required.

---

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | MEDIUM | Core choices (Expo managed, expo-router, Zustand, React Query, axios) are HIGH confidence. Specific version numbers need verification via `npx expo install` at scaffold time. `@tesserix/native` internals are LOW confidence (package confirmed to exist but peer deps unknown). |
| Features | HIGH | Based on direct codebase analysis of existing API handlers, push infrastructure, and established food delivery app patterns (Swiggy, Zomato, Uber Eats, DoorDash). Feature list is grounded in what the API already supports. |
| Architecture | HIGH | Metro monorepo config, expo-router file-based routing, pnpm workspace integration, Zustand + React Query for state — all well-established patterns unchanged since Expo SDK 49. API client factory pattern is standard. |
| Pitfalls | MEDIUM-HIGH | FCM token type distinction is HIGH (fundamental FCM architecture, stable). Metro symlinks, EAS Build monorepo config, Apple background location review requirements are HIGH based on stable platform behavior. GPS polling DB impact is HIGH (directly observed from codebase analysis of `push.go` and `db-f1-micro` constraint). |

**Overall confidence:** MEDIUM — architecture and patterns are solid; exact version numbers and two API capability gaps need resolution before Phase 1 ends.

### Gaps to Address

- **`@tesserix/native` peer deps:** Read the package's `package.json` before scaffolding. Pin Expo SDK version to match. Do not proceed with Phase 1 scaffold until this is confirmed.
- **Expo SDK 52 vs 53:** Run `npm info expo dist-tags` at scaffold time. If SDK 53 is stable, evaluate whether to start there rather than SDK 52.
- **Go API delivery location endpoint:** Confirm `PATCH /v1/delivery/:id/location` exists and what rate limiting it supports. If absent, this is a backend task that must be planned before Phase 4 begins.
- **Real-time location broadcast for Customer tracking:** Determine whether the Go API can push driver location to customers (NATS → WebSocket or polling endpoint). The v1 fallback is 5-second polling by the Customer app — validate this is acceptable UX before committing to it.
- **JWT expiry duration:** Audit `apps/api/config/` for JWT token expiry. If shorter than 8 hours, silent token refresh in the shared API client is critical (especially for Driver app with long active sessions).
- **EAS Build free tier limits:** Verify current pricing at https://expo.dev/pricing. GitHub Free plan is already a constraint (no org secrets, no SARIF upload per `CLAUDE.md`).

---

## Sources

### Primary (HIGH confidence)
- Project codebase — `apps/api/services/push.go`, `apps/api/models/delivery.go`, `apps/api/handlers/delivery.go`, `apps/api/routes/routes.go` — FCM token handling, delivery model, location update endpoint
- `.planning/PROJECT.md` — Feature scope, platform constraints, deferred items
- `.planning/codebase/CONCERNS.md` — Known issues including rate limiting gaps

### Secondary (MEDIUM confidence)
- Expo SDK documentation patterns (Metro monorepo config, EAS Build projectRoot, expo-router, expo-secure-store) — training data, August 2025 cutoff; verify against current Expo docs before implementation
- Food delivery app industry patterns (Swiggy, Zomato, Uber Eats, DoorDash) — feature expectations, driver app UX conventions
- Apple App Store Review Guidelines — background location justification requirements; guidelines are stable but verify current wording

### Tertiary (LOW confidence)
- Expo SDK 53 roadmap — post training-data-cutoff release; verify at scaffold time
- `@tesserix/native` package internals — package existence confirmed, internal peer deps unknown until read at scaffold time

---
*Research completed: 2026-04-05*
*Ready for roadmap: yes*
