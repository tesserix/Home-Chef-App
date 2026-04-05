# Technology Stack — Mobile Apps (Expo / React Native)

**Project:** Home Chef Mobile Apps (Customer, Vendor, Delivery Driver)
**Researched:** 2026-04-05
**Research tools available:** Training data only (WebSearch + WebFetch blocked during this session)
**Overall confidence:** MEDIUM — versions match training data through August 2025; flag items marked VERIFY before pinning in package.json

---

## IMPORTANT: Version Verification Required

All version numbers below come from training data (cutoff August 2025). Before writing any
`package.json`, run:

```bash
npx expo install --fix          # after init, lets Expo pin compatible versions
npm info expo dist-tags         # confirm latest stable SDK tag
npm info expo-router version    # confirm router version
```

The Expo ecosystem uses a tight peer-dependency graph — never pin Expo library versions
independently. Always use `npx expo install <pkg>` so the Expo CLI resolves the correct
version for the SDK you've chosen.

---

## Recommended Stack

### Core Platform

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| Expo SDK | **52** (stable as of Aug 2025) | Core React Native managed workflow | Managed workflow eliminates native build toolchain from dev machines; New Architecture (Fabric + JSI) enabled by default in SDK 52; matches React 19 support. VERIFY: SDK 53 may be stable by Apr 2026 — run `npm info expo dist-tags` |
| React Native | **0.76.x** (bundled with SDK 52) | Native rendering layer | Do not pin independently; Expo SDK controls this version |
| React | **19.x** | UI rendering | Matches existing web apps (React 19.0.0) — consistent mental model for the team |
| TypeScript | **5.7.x** | Type safety | Matches monorepo root `typescript ^5.7.2` |
| Node.js | **22.x** | Build toolchain | Matches monorepo `engines.node >= 22.0.0` |

**Confidence:** MEDIUM — SDK 52 was the latest stable at training cutoff; SDK 53 roadmap was visible but release date was post-cutoff.

---

### Navigation

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| expo-router | **~4.0.x** | File-based routing | Official Expo navigation solution since SDK 50; built on React Navigation 7 under the hood; native deep linking via Expo's Universal Links without extra config; typed routes with TypeScript; aligns with Next.js App Router mental model the team already uses. **Use this, not react-navigation directly.** |
| react-navigation (peer) | **7.x** | Underlying stack/tab primitives | Pulled in automatically by expo-router; do not import directly unless expo-router lacks a needed primitive |

**Why not react-navigation directly:** expo-router gives file-based routing, typed `<Link>` components, automatic deep link configuration, and layout files — exactly the pattern the team uses in Next.js. Direct react-navigation requires manual linking config for both iOS and Android.

**Confidence:** MEDIUM — expo-router v4 was in beta/RC at training cutoff targeting SDK 52/53. VERIFY that v4 is stable; fall back to expo-router v3 if v4 is still RC.

---

### Maps and Location

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| react-native-maps | **~1.18.x** | Map display, route overlays, markers | The standard for Expo-managed workflow; uses Google Maps on Android and Apple Maps on iOS with a single API; supports live coordinate updates for delivery tracking; works with Expo Go in development |
| expo-location | **~17.x** | GPS coordinates, background location | Expo managed module; handles iOS/Android permission dialogs correctly; `startLocationUpdatesAsync` with background task for driver tracking |
| react-native-maps-directions | **~1.x** | Driving route polylines | Thin wrapper around Google Directions API; used for driver turn-by-turn route display |

**Why react-native-maps over MapLibre:** MapLibre-React-Native requires custom native builds (no Expo Go support without a dev client) and targets the offline/self-hosted tile case. For food delivery with Google Maps tiles and Directions API already in use, react-native-maps is the correct fit and much simpler to configure.

**Why react-native-maps over Mapbox:** Mapbox SDK v10+ dropped Expo managed workflow support and requires a bare workflow or a dev client build. Avoid for this project.

**Background location note:** Driver app requires background location. This requires:
1. `expo-location` `startLocationUpdatesAsync` with `expo-task-manager`
2. EAS Build (not Expo Go) for the driver app specifically
3. iOS `UIBackgroundModes: location` in `app.json`

**Confidence:** MEDIUM — react-native-maps compatibility with SDK 52 is well established; version numbers need `npx expo install` to confirm.

---

### Push Notifications

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| expo-notifications | **~0.29.x** | Push token registration, local/remote notification display | Expo's managed notification module; abstracts FCM (Android) and APNs (iOS) behind a single API; ExponentPushToken works with Expo Push Service as a free relay |
| expo-device | **~6.x** | Check if running on physical device before requesting push permissions | Required guard — simulators cannot receive push notifications |

**Push flow for this project:**
1. App registers with `expo-notifications`, gets an ExponentPushToken
2. Token is sent to the Go API and stored per-user
3. Go API sends push via Expo Push API (`https://exp.host/--/api/v2/push/send`) — no FCM/APNs credentials required for Expo Push Service
4. Expo Push Service relays to FCM/APNs

**Vendor app critical:** New order notifications must wake the vendor app reliably. Use `priority: "high"` and `channelId` (Android) in all order notification payloads. Test on physical devices early — simulators lie about notification delivery.

**Direct FCM/APNs (alternative):** If Expo Push Service latency becomes a problem in production, the Go backend already uses NATS; the API could bypass Expo Push Service and send directly to FCM/APNs using `firebase.google.com/go/v4` (already in the Tesserix shared stack). But for v1, use Expo Push Service — zero credential management.

**Confidence:** MEDIUM — expo-notifications API is stable; version needs `npx expo install` verification.

---

### State Management

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| Zustand | **^5.0.x** | Client/UI state (cart, auth session, active delivery state) | Matches web apps (Zustand v5.0.2); minimal boilerplate; works identically in React Native; no context provider wrapping needed |
| TanStack React Query | **^5.x** | Server state, API data fetching, caching | Matches web apps (v5.62.8); `useQuery`/`useMutation` work identically in React Native; built-in background refetch (critical for live order status); `staleTime` config handles the connectivity variability of mobile |

**Why not Redux:** Overkill for three focused apps; team already uses Zustand + React Query pattern.

**Why not SWR:** React Query is already established in the project. Use one server state library.

**State split guideline:**
- Zustand: cart contents, auth token/user, selected delivery, onboarding wizard step, UI toggles
- React Query: all API data — orders, menu items, chef listings, delivery status, earnings

**Confidence:** HIGH — both libraries are framework-agnostic; versions confirmed stable in training data.

---

### API Client

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| axios | **^1.13.x** | HTTP client for all Go API calls | Matches web apps (axios v1.13.0); interceptors for JWT injection and 401 refresh are straightforward; team already knows the pattern |

**Shared API client pattern (important):**
Create a shared package `packages/api-client/` in the monorepo that all three mobile apps import. This package contains:
- Axios instance with base URL + JWT interceptor
- TypeScript types for all request/response shapes (can be shared with web apps too)
- One function per API endpoint

Do not duplicate axios config in each of the three mobile app directories.

**JWT storage for mobile:** Use `expo-secure-store` (not `AsyncStorage`) for access tokens and refresh tokens. AsyncStorage is unencrypted; tokens in it can be extracted from unrooted Android devices via ADB backup.

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| expo-secure-store | **~13.x** | Encrypted token storage (Keychain on iOS, Keystore on Android) | Security requirement; AsyncStorage is plaintext |

**Confidence:** HIGH for axios pattern; MEDIUM for expo-secure-store version.

---

### Local Storage and Persistence

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| expo-secure-store | **~13.x** | Auth tokens, sensitive preferences | See above |
| AsyncStorage (`@react-native-async-storage/async-storage`) | **^2.x** | Non-sensitive persistent data: cached chef listings, onboarding wizard draft, user preferences | Expo managed module; used by React Query's `createAsyncStoragePersister` for offline cache |
| expo-file-system | **~17.x** | Image downloads/caching for chef photos and menu images | Expo managed module; needed for image caching strategy |

**Why not MMKV:** `react-native-mmkv` is faster than AsyncStorage for large stores, but requires a dev client build (not compatible with Expo Go in managed workflow). MMKV is worth considering post-v1 if React Query's AsyncStorage persistence becomes a bottleneck. For v1, stick with AsyncStorage.

**Confidence:** MEDIUM — versions need `npx expo install` confirmation.

---

### Image Handling

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| expo-image | **~2.x** | Performant image display with caching | Expo's replacement for React Native's built-in `<Image>`; uses `SDWebImage` on iOS and `Glide` on Android under the hood; critical for chef profile photos and menu item images loading smoothly |

**Why not `react-native-fast-image`:** Requires native build (incompatible with Expo Go in managed workflow). expo-image provides equivalent performance in managed workflow.

**Confidence:** MEDIUM — expo-image v2 was in RC at training cutoff; verify stable status.

---

### Forms and Validation

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| React Hook Form | **^7.x** | Form state for login, registration, onboarding, order forms | Matches web apps (v7.54.1); fully compatible with React Native via `Controller` wrapper |
| Zod | **^3.x** | Schema validation | Matches web apps (v3.24.1); `@hookform/resolvers/zod` works identically in RN |

**Pattern difference from web:** React Native has no native `<input>` — wrap all form fields with RHF's `<Controller>` component pointing to React Native `TextInput` or `@tesserix/native` input components.

**Confidence:** HIGH — both libraries are framework-agnostic.

---

### Payments

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| expo-web-browser | **~14.x** | Opens Razorpay checkout in an in-app browser | PROJECT.md explicitly defers Apple Pay/Google Pay to post-v1; use existing Razorpay web checkout via `expo-web-browser.openBrowserAsync(razorpayUrl)` — zero native SDK integration required for v1 |

**Why not Razorpay React Native SDK:** Razorpay's official RN SDK (`react-native-razorpay`) requires bare workflow or a dev client because it links native code. PROJECT.md already decided to use web checkout for v1 — honor that constraint.

**Post-v1 upgrade path:** Once EAS dev client is adopted, swap to `react-native-razorpay` for a native checkout sheet.

**Confidence:** HIGH — expo-web-browser is stable; the constraint is project-level, not library-level.

---

### Real-Time Updates (Live Order / Delivery Tracking)

The Go API uses NATS for event publishing but mobile apps are clients, not NATS subscribers. Options:

| Approach | Recommended | Reason |
|----------|-------------|--------|
| React Query polling (`refetchInterval`) | **YES for v1** | Simplest; no extra infrastructure; `refetchInterval: 5000` on order status queries is sufficient for tracking updates |
| WebSockets (`expo-modules-core` + native WS) | Post-v1 | Needed for sub-second delivery tracking; Go API already knows how to serve WS (standard `net/http` upgrade); implement when 5s polling feels too slow |
| Server-Sent Events | Not recommended | Poor mobile network recovery behavior; harder to implement in Go than WS |

**Polling is the correct v1 decision.** The customer tracking screen and driver active delivery screen use React Query's `refetchInterval`. If the team validates that 5s feels laggy for customers watching the map, upgrade to WebSockets in a targeted follow-on.

**Confidence:** HIGH — this is an architectural recommendation, not a library version claim.

---

### Build, CI/CD, and OTA

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| EAS Build | Current (cloud service, no version to pin) | Cloud-based iOS + Android binary builds | PROJECT.md constraint; eliminates need for macOS build machines for Android; required for iOS production builds from Linux CI; handles code signing certificates and provisioning profiles |
| EAS Submit | Bundled with EAS | App Store + Play Store submission | Automates `xcrun altool` / `bundletool` submission flows |
| EAS Update | Bundled with EAS | Over-the-air JS bundle updates | Critical for food delivery apps — push bug fixes between store reviews; does NOT update native code (maps, notifications) — only JS bundle |

**EAS Build profile recommendation:**

```json
// eas.json
{
  "build": {
    "development": {
      "developmentClient": true,
      "distribution": "internal"
    },
    "preview": {
      "distribution": "internal",
      "channel": "preview"
    },
    "production": {
      "channel": "production"
    }
  }
}
```

**Dev client requirement:** The driver app needs background location — this cannot be tested in Expo Go. Set up a dev client build (`developmentClient: true`) for the driver app from the start of development.

**Monorepo EAS configuration:** Each of the three apps needs its own `app.json` with a unique `slug` and `bundleIdentifier`/`package`. EAS supports monorepos via the `projectRoot` field in `eas.json`.

**Confidence:** MEDIUM — EAS Build is a cloud service; pricing tiers and build minute limits should be verified at https://expo.dev/pricing before committing to the free tier for CI.

---

### Development Tooling

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| Expo Go | Latest (app store) | Development on physical devices without a build step | Use for customer and vendor apps during most of development; NOT sufficient for driver app (background location) |
| expo-dev-client | **~4.x** | Custom dev client with native modules | Required for driver app; recommended for all three apps once background location and any other native-only module is added |
| Maestro | **^1.x** | Mobile E2E testing | The standard Expo-compatible E2E framework; does not require Detox's complex native setup; tests against Expo Go or dev client builds; write flows for login, checkout, and order acceptance early |
| Jest + jest-expo | **~52.x** | Unit and component testing | `jest-expo` preset handles React Native transforms; matches `jest-expo` version to SDK version |

**Why not Detox:** Detox requires building a special test binary and complex native configuration. Maestro runs against your actual app binary via accessibility IDs — zero native build changes needed.

**Confidence:** MEDIUM — Maestro version and jest-expo preset version need verification.

---

### Monorepo Integration

The three mobile apps live at `apps/mobile-customer/`, `apps/mobile-vendor/`, `apps/mobile-delivery/` inside the existing pnpm monorepo.

**Key integration points:**

1. **pnpm workspace:** Add the three mobile app directories to `pnpm-workspace.yaml`. The shared API client package goes in `packages/api-client/`.

2. **Metro bundler + pnpm symlinks:** Expo's Metro bundler does not follow symlinks by default. Each mobile app's `metro.config.js` must include:

```javascript
// metro.config.js (each mobile app)
const { getDefaultConfig } = require("expo/metro-config");
const path = require("path");

const projectRoot = __dirname;
const workspaceRoot = path.resolve(projectRoot, "../..");

const config = getDefaultConfig(projectRoot);
config.watchFolders = [workspaceRoot];
config.resolver.nodeModulesPaths = [
  path.resolve(projectRoot, "node_modules"),
  path.resolve(workspaceRoot, "node_modules"),
];

module.exports = config;
```

This is non-negotiable — without this, Metro will not resolve packages from `packages/api-client/` or `packages/design-system/`.

3. **`@tesserix/native` resolution:** The design system package must be listed as a `watchFolder` in Metro config. Import it the same way as in any other app in the monorepo — no special handling needed beyond the metro.config.js above.

4. **TypeScript path aliases:** Add a `tsconfig.json` to each mobile app extending the root tsconfig, with `paths` mapping `@homechef/api-client` to the shared package.

5. **Do NOT hoist all packages to root:** React Native requires specific versions of `react` and `react-native` to be resolvable from each app's own `node_modules`. Add `shamefully-hoist=false` guard and use `dependenciesMeta` in pnpm to prevent hoisting conflicts.

**Confidence:** HIGH — Metro + pnpm monorepo setup is well-documented; the metro.config.js pattern is standard Expo monorepo practice.

---

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| Framework | Expo (managed) | Bare React Native | Bare workflow requires macOS + Xcode + Android Studio on every dev machine; Expo managed + EAS Build eliminates this; managed workflow cost is negligible for this feature set |
| Framework | Expo | Flutter | PROJECT.md already decided Expo — team uses React/TypeScript, reuse is real |
| Navigation | expo-router | react-navigation directly | expo-router provides deep linking auto-config, typed routes, and file-based routing that the team already understands from Next.js |
| Maps | react-native-maps | Mapbox RN SDK v10 | Mapbox dropped managed workflow support; requires bare workflow |
| Maps | react-native-maps | MapLibre-React-Native | MapLibre targets self-hosted tiles + offline; adds complexity for no benefit here |
| Push | Expo Push Service | Direct FCM/APNs | Expo Push Service is zero-credential for v1; Go backend already has Firebase SDK if direct push is needed later |
| Payments | expo-web-browser (Razorpay web checkout) | react-native-razorpay | Native SDK requires bare workflow; PROJECT.md explicitly deferred native payments to post-v1 |
| Storage (sensitive) | expo-secure-store | AsyncStorage for tokens | AsyncStorage is unencrypted; tokens can be extracted via ADB backup |
| E2E Testing | Maestro | Detox | Detox requires a special test binary and native build changes; Maestro runs against the real app |
| Real-time | React Query polling (v1) | WebSockets | Polling at 5s interval is sufficient for v1 delivery tracking; WS upgrade path is clear |

---

## Installation

```bash
# Initialize each mobile app (run once per app directory)
npx create-expo-app apps/mobile-customer --template blank-typescript
npx create-expo-app apps/mobile-vendor --template blank-typescript
npx create-expo-app apps/mobile-delivery --template blank-typescript

# Install navigation (run inside each app)
npx expo install expo-router react-native-safe-area-context react-native-screens \
  expo-linking expo-constants expo-status-bar

# Install maps and location
npx expo install react-native-maps expo-location expo-task-manager

# Install notifications
npx expo install expo-notifications expo-device

# Install storage
npx expo install expo-secure-store @react-native-async-storage/async-storage

# Install image handling
npx expo install expo-image

# Install payments gateway
npx expo install expo-web-browser

# Install shared dependencies (same as web apps — no version conflicts)
pnpm add zustand @tanstack/react-query axios react-hook-form zod @hookform/resolvers

# Dev dependencies
npx expo install jest-expo --save-dev
pnpm add -D maestro  # or install Maestro CLI globally: curl -Ls "https://get.maestro.mobile.dev" | bash
```

**Critical:** Always use `npx expo install` for Expo-managed packages. This runs the Expo version resolver to pick the correct version for your SDK. Using `pnpm add` directly for Expo packages will likely install an incompatible version.

---

## Confidence Summary

| Decision | Confidence | Reason |
|----------|------------|--------|
| Expo managed workflow | HIGH | Stable, well-documented; the right choice for this team |
| expo-router for navigation | MEDIUM | v4 stable status needs verification; v3 is the safe fallback |
| react-native-maps for GPS | MEDIUM | Stable library; version needs `npx expo install` |
| expo-location + task-manager for background GPS | MEDIUM | API stable; needs physical device testing early |
| Expo Push Service + expo-notifications | MEDIUM | Service is stable; push reliability on physical devices must be tested |
| Zustand v5 + React Query v5 | HIGH | Framework-agnostic; already confirmed in web apps |
| axios for API client | HIGH | Framework-agnostic; already in use |
| expo-secure-store for tokens | HIGH | Security requirement; API is stable |
| EAS Build for CI/CD | MEDIUM | Service-level; verify current pricing tier limits |
| React Query polling for real-time v1 | HIGH | Architectural decision; implementation straightforward |
| Maestro for E2E testing | MEDIUM | Growing adoption; verify current version |
| Metro config for pnpm monorepo | HIGH | Documented pattern; deterministic behavior |

---

## Version Verification Checklist

Run these before writing any `package.json` or `app.json`:

```bash
# Confirm latest stable SDK
npm info expo dist-tags

# Confirm expo-router stable version
npm info expo-router version

# Confirm expo-image stable (was RC at training cutoff)
npm info expo-image version

# Confirm expo-dev-client version
npm info expo-dev-client version

# Check EAS CLI version
npm info eas-cli version
```

---

*Research date: 2026-04-05 | Training data cutoff: August 2025 | Web research tools: unavailable during this session*
