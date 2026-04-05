# Domain Pitfalls

**Domain:** Expo/React Native food delivery mobile apps (Customer, Vendor, Driver)
**Researched:** 2026-04-05
**Overall confidence:** MEDIUM (training data up to Aug 2025; web tools unavailable for live verification)

---

## Critical Pitfalls

Mistakes that cause rewrites, App Store rejections, or user-facing regressions.

---

### Pitfall 1: FCM Token Mismatch — Expo Push Token vs Raw FCM Token

**What goes wrong:** The existing API stores a raw FCM device token in `users.fcm_token` and sends via the FCM HTTP v1 API directly (`apps/api/services/push.go`). Expo's `expo-notifications` library by default gives you an **Expo Push Token** (format: `ExponentPushToken[xxxxxx]`), not a raw FCM token. If the mobile app registers an Expo Push Token into the `PUT /v1/profile/device-token` endpoint that calls `sendToToken(user.FCMToken, ...)` directly against FCM, every push notification silently fails. FCM rejects tokens it doesn't recognise with a 404, but the current `sendToToken` only logs the error — it doesn't surface it to the caller.

**Why it happens:** Teams assume "push token" means the same thing across FCM and Expo. Expo Push Tokens are valid only for the Expo Push Notification service (a relay). Raw FCM tokens come from `getDevicePushTokenAsync()` (not `getExpoPushTokenAsync()`). The two APIs look identical but return completely different token formats.

**Consequences:** Vendors never get new-order push alerts. Drivers miss delivery assignments. Customer order status notifications silently fail. No error shown anywhere because the push service is fire-and-forget with a log-only error path.

**Warning signs:**
- Push notifications work on Expo Go but not on production standalone builds
- `push.go` logs "Push notification sent" but nothing arrives on device
- FCM dashboard shows 0 deliveries despite non-zero sends

**Prevention:**
- Option A (recommended for this project): Use `getDevicePushTokenAsync()` from `expo-notifications` to obtain a raw FCM/APNs token. Register this in the existing API endpoint. No backend changes needed.
- Option B: Switch to Expo Push Notification Service relay and use `getExpoPushTokenAsync()`. Requires backend to call `https://exp.host/--/api/v2/push/send` instead of FCM directly.
- Option A preserves the existing backend unchanged. Option B requires a backend rewrite of `push.go`.
- Confirm which token type you're registering in Phase 1 (infrastructure setup) before any notification wiring.

**Phase:** Address in Phase 1 (Expo/EAS setup + infrastructure). Do not defer to the notification feature phase.

---

### Pitfall 2: Monorepo Metro Bundler Symlink Resolution Failure

**What goes wrong:** Metro (React Native's bundler) does not follow symlinks by default. pnpm uses symlinks heavily in `node_modules/.pnpm/` to share packages across workspace members. When `apps/mobile-customer/` is a workspace package that depends on `@tesserix/native` (another workspace package in `packages/`), Metro silently fails to resolve imports from `@tesserix/native` — or worse, resolves two conflicting copies of React Native (one per workspace package), producing the infamous "Invariant Violation: No React Native renderer" crash at runtime.

**Why it happens:** The monorepo uses `pnpm@9.15.1` with `packages: ["apps/*", "packages/*"]` in `pnpm-workspace.yaml`. Every other app in this repo (web, vendor-portal, etc.) uses Vite, which handles symlinks fine. Metro's resolver needs explicit configuration: `watchFolders` must include the monorepo root, and `resolver.nodeModulesPaths` must be set. Without this, Metro only looks in `apps/mobile-customer/node_modules/`.

**Consequences:** Build succeeds but app crashes immediately on launch. Or `@tesserix/native` imports resolve to `undefined`. The error messages are cryptic ("cannot find module" or "null is not an object (evaluating '_reactNative.StyleSheet.create')").

**Warning signs:**
- Works in Expo Go but crashes in standalone build
- `Metro bundler` errors mentioning symlinks or `ENOENT` on pnpm-managed packages
- Duplicate React Native instances in bundle analysis

**Prevention:**
- Add `metro.config.js` to each mobile app with explicit `watchFolders: [path.resolve(__dirname, '../..')]` (monorepo root) and `resolver.nodeModulesPaths` pointing to root `node_modules`.
- Enable `resolver.unstable_enableSymlinks: true` (Expo SDK 50+, HIGH confidence this exists).
- Consider using `pnpm` with `shamefully-hoist=true` in an `.npmrc` at the monorepo root to flatten the dependency tree for Metro — but test this carefully as it can break other workspace members.
- EAS Build requires `eas.json` `"projectRoot"` and `"buildArtifactPaths"` to be set relative to the monorepo root, not the app directory.

**Phase:** Address in Phase 1 (monorepo scaffolding). Non-negotiable — this must be validated before any feature work begins.

---

### Pitfall 3: GPS Location Polling Floods the API and Drains Battery

**What goes wrong:** The existing delivery tracking model is HTTP polling: driver app calls `PUT /v1/delivery/location` with coordinates, customers poll `GET /v1/delivery/:id` to see the updated position. If the driver app calls `PUT /location` every second using `expo-location`'s `watchPositionAsync`, that is 86,400 API calls per driver per active delivery day. Each call also does a full `DB.First + DB.Save` (the full `DeliveryPartner` row), which is a DB write per ping. At 10 active drivers this is 864,000 DB writes/day on a single `db-f1-micro` Cloud SQL instance — which will saturate the connection pool.

**Why it happens:** `watchPositionAsync` is easy to wire up and developers assume "more frequent = more accurate map". The API endpoint `PUT /delivery/location` has no rate limiting (CONCERNS.md confirms no rate limiting on most endpoints). The existing in-memory chat rate limiter pattern won't scale; applying that same pattern to location updates won't work with multiple mobile replicas either.

**Consequences:** Cloud SQL CPU pinned at 100%. API response times degrade for all users during active delivery windows. Battery drain on driver devices (background location + constant network). App Store review may flag excessive background location usage.

**Warning signs:**
- DB CPU spikes during delivery test scenarios
- Driver device battery drops faster than expected during active delivery
- Background fetch wakeups exceeding iOS/Android background app refresh budgets

**Prevention:**
- Use `distanceInterval` (minimum 20–50 metres) and `timeInterval` (minimum 5–10 seconds) in `watchPositionAsync` options. This reduces updates by ~10–50x.
- Apply server-side rate limiting on `PUT /delivery/location`: one write per 5 seconds per driver. Return 200 immediately if the interval hasn't elapsed (this is a backend task, but mobile must gracefully handle the 429 without crashing).
- For the customer-facing tracking map, implement client-side polling (every 5s GET for delivery position) rather than having the server push — keeps the existing REST architecture intact without WebSocket refactoring.
- Use `expo-task-manager` + `expo-location` background task for driver location — but set `accuracy: Location.Accuracy.Balanced` not `Location.Accuracy.BestForNavigation` when the driver is stationary (detect via speed < 1 m/s).

**Phase:** Architecture decision must be made in Phase 1 (infrastructure). Rate limiting implementation is a backend task that must be unblocked before the driver app goes to beta.

---

### Pitfall 4: EAS Build Fails Because `eas.json` Does Not Account for Monorepo Root

**What goes wrong:** EAS Build runs from the app directory by default. In a monorepo, the app's `package.json` is at `apps/mobile-customer/package.json` but shared dependencies (like `@tesserix/native`) are hoisted to the root `node_modules/`. EAS Build creates a clean build environment and only uploads the app subdirectory unless `eas.json` explicitly sets `"projectRoot": "../.."`. Missing this means the build environment has no `@tesserix/native`, no shared types, and fails with "Cannot find module" during the Metro bundle step on EAS servers.

**Why it happens:** EAS CLI detects `app.json` or `app.config.js` in the directory you run it from, not the monorepo root. The monorepo root `pnpm-workspace.yaml` is invisible to EAS Build unless explicitly included.

**Consequences:** Local dev works fine (`npx expo start` from the app dir can find hoisted packages), but every CI/CD build fails. Teams spend days debugging EAS Build log output that says "module not found" without pointing to the real cause.

**Warning signs:**
- Local `expo start` works; EAS Build fails with module resolution errors
- EAS Build log shows `npm install` or `pnpm install` running only in the app subdirectory
- `@tesserix/native` is not in the app's own `package.json` dependencies

**Prevention:**
- Set `"projectRoot": "../.."` in each app's `eas.json` build profiles.
- Add an `"install"` hook in `eas.json` that runs `pnpm install --frozen-lockfile` from the monorepo root.
- Verify with `eas build --local` before first remote build — this surfaces resolver issues without consuming EAS Build credits.
- Keep `pnpm-lock.yaml` at the monorepo root committed and up to date; EAS Build uses it for reproducible installs.

**Phase:** Phase 1 (EAS Build setup). Validate with a dummy "Hello World" build before any feature work.

---

### Pitfall 5: Apple App Store Rejection for Background Location Without Justification

**What goes wrong:** The Driver app requires `background location` permission (iOS `NSLocationAlwaysUsageDescription` + `UIBackgroundModes: location`). Apple's review team rejects apps that request `Always` location permission without a clear in-app UI showing the user why — and specifically requires that users can choose `When In Use` as an alternative. Apps that ask for `Always` permission immediately on first launch (before the user has started a delivery) are rejected.

**Why it happens:** Developers add `"permissions": ["LOCATION", "LOCATION_BACKGROUND"]` to `app.json` globally and request `requestForegroundPermissionsAsync` + `requestBackgroundPermissionsAsync` on app start, without gating on the user's current workflow state.

**Consequences:** 1–3 week App Store re-review cycle. Revenue delay. The fix (showing a pre-permission rationale screen) is simple, but the round-trip is expensive.

**Warning signs:**
- `app.json` requests `LOCATION_BACKGROUND` at app level rather than per workflow
- Background location is requested unconditionally on Driver app first launch
- No in-app explanation screen before the system permission prompt

**Prevention:**
- Request foreground location (`requestForegroundPermissionsAsync`) first — only escalate to background (`requestBackgroundPermissionsAsync`) when the driver taps "Go Online" / "Start Delivery".
- Show a native-style rationale screen before every permission prompt explaining exactly what the app does with location data. This is required by Apple, not optional.
- In `app.json`, scope `UIBackgroundModes: ["location"]` only to the Driver app, not the Customer or Vendor apps.
- Customer app only needs `When In Use` location (for address detection / nearby chefs). Never request `Always` in the Customer or Vendor apps.

**Phase:** Phase 2 (Driver app GPS/navigation feature). Must be designed into the permission flow from the start, not retrofitted before submission.

---

### Pitfall 6: Single FCM Token Per User Breaks Multi-Device and Re-login Scenarios

**What goes wrong:** `users.fcm_token` is a single string column. When a user logs in on a second device (e.g., a Vendor using their tablet as backup), the new device token overwrites the old one. The first device stops receiving notifications. When a user logs out and logs back in, the token may change (FCM rotates tokens after app reinstalls or token refresh events). If the mobile app does not re-register the token on every app foreground, the stored token becomes stale and all notifications silently fail.

**Why it happens:** The existing web apps don't need device tokens (they use web push or in-app polling). The single-column schema was sufficient. Mobile apps introduce token lifecycle complexity that web doesn't have.

**Consequences:** Vendors miss new order notifications after reinstalling the app. Drivers miss delivery assignments after device swap. Silent failure — no error is visible anywhere.

**Warning signs:**
- Push notifications stop arriving after app reinstall
- `push.go` logs "Push notification sent" but device never receives anything
- FCM returns `UNREGISTERED` error (token invalid) — current code logs this but doesn't clean up the stale token

**Prevention:**
- Register the Expo/FCM token every time the app comes to foreground, not just on first login. Use `expo-notifications` `addPushTokenListener` to detect token rotation.
- For MVP: Accept the single-device limitation but add server-side cleanup: when FCM returns `UNREGISTERED` (HTTP 404 from FCM), clear `fcm_token` to prevent repeated failed sends. Currently `sendToToken` logs the error but does not clear the stale token in the DB.
- Medium-term: Migrate `fcm_token` to a `device_tokens` join table (user_id, token, platform, created_at) to support multiple devices.
- This is a backend schema change — flag it as a backend task to complete before the Vendor app ships (Vendors are most sensitive to missed notifications).

**Phase:** Phase 1 (infrastructure) for the re-registration logic on mobile. Backend schema migration can be Phase 2 but must be complete before Vendor beta.

---

## Moderate Pitfalls

---

### Pitfall 7: `react-native-maps` on iOS Requires Google Maps API Key in `app.json` or MapKit Setup

**What goes wrong:** `react-native-maps` on iOS defaults to Apple MapKit. If you want Google Maps on iOS (for consistency with Android), you must provide a `GOOGLE_MAPS_API_KEY` in `app.json` under `ios.config.googleMapsApiKey`. If you use MapKit but forget to add the `NSLocationWhenInUseUsageDescription` key in `app.json` `infoPlist`, the map renders blank on real devices.

**Why it happens:** The simulator doesn't enforce location permission descriptions. Developers test on simulator, pass, then get a blank map (or crash) on the first TestFlight install.

**Warning signs:**
- Map renders correctly in simulator but shows blank tiles on real device
- Xcode console shows "Trying to start MapKit session without valid entitlements"

**Prevention:**
- Decide at Phase 2 start whether to use Google Maps (both platforms, unified SDK) or Apple MapKit on iOS + Google on Android. Google Maps is recommended for delivery apps (better routing data, same API surface on both platforms).
- Add `ios.config.googleMapsApiKey` to `app.json` and ensure the key has both "Maps SDK for iOS" and "Directions API" enabled in Google Cloud Console.
- Add `NSLocationWhenInUseUsageDescription` to `app.json` `infoPlist` before first TestFlight build.

**Phase:** Phase 2 (GPS/maps feature).

---

### Pitfall 8: Expo SDK Version Lock-in vs. `@tesserix/native` Peer Dependency Mismatch

**What goes wrong:** `@tesserix/native` is an existing internal design system package. If it declares `react-native` or `expo` peer dependencies pinned to a specific SDK version (e.g., SDK 51) but the mobile apps use SDK 53, Expo's compatibility layer may silently downgrade or skip components. Worse, if `@tesserix/native` internally uses `expo-*` packages that were moved or renamed between SDK versions, the build silently produces wrong behavior.

**Why it happens:** Internal packages are rarely kept in sync with Expo SDK upgrade cadence. The web design system (`@tesserix/web`) doesn't have this problem because browser APIs are stable.

**Warning signs:**
- `@tesserix/native` components render differently than the design spec
- Peer dependency warnings during `pnpm install` in mobile app directories
- Expo upgrade guide lists breaking changes in packages that `@tesserix/native` uses

**Prevention:**
- Before scaffolding the mobile apps, read `@tesserix/native`'s `package.json` and identify the exact `expo` and `react-native` peer deps it requires. Pin all three mobile apps to that same SDK version.
- Do not upgrade Expo SDK independently in mobile apps without also upgrading `@tesserix/native`.
- Document the locked Expo SDK version in each `app.json`.

**Phase:** Phase 1 (scaffolding). Pin Expo SDK version before writing any feature code.

---

### Pitfall 9: Deep Link Scheme Conflicts Across the Three Apps

**What goes wrong:** If all three apps use the same URI scheme (e.g., `homechef://`) in `app.json`, iOS will route deep links to whichever app was installed last, silently breaking the others. This is especially problematic for payment redirect flows (Razorpay webhook redirects back to the app via deep link).

**Why it happens:** Developers copy `app.json` from one app to the next without changing the `scheme`.

**Warning signs:**
- Tapping a payment return link opens the wrong app
- Universal Links (HTTPS-based deep links) configured on a single domain but all three apps claim the same AASA (Apple App Site Association) paths

**Prevention:**
- Assign distinct URI schemes: `homechef://` (Customer), `homechef-vendor://` (Vendor), `homechef-driver://` (Driver).
- For Universal Links / App Links (HTTPS), use distinct path prefixes per app: `/app/order/*` → Customer, `/app/vendor/*` → Vendor, `/app/driver/*` → Driver.
- Set this in Phase 1 scaffolding and never change it — URI schemes are embedded in the binary and cannot be changed in an OTA update.

**Phase:** Phase 1 (app scaffolding). Non-negotiable — baked into the binary.

---

### Pitfall 10: Razorpay Mobile Payment — WebView vs Native SDK

**What goes wrong:** The PROJECT.md explicitly defers Apple Pay/Google Pay and uses "existing Razorpay web checkout for v1." If this is implemented as `WebBrowser.openBrowserAsync()` or a `WebView`, the payment page opens in a browser, Razorpay charges complete, but the redirect back to the app via deep link often fails silently on Android if the deep link scheme is not registered as an Intent Filter. The user is left on a browser success page with no way back to the order confirmation screen.

**Why it happens:** The web checkout flow assumes the browser handles redirects. Mobile deep link redirect requires explicit Intent Filter (Android) and URL Scheme (iOS) handling, plus the Razorpay dashboard must have the mobile deep link URL registered as a valid callback URL.

**Warning signs:**
- Payment completes successfully but order confirmation screen never appears
- Android: user sees "No app found to open this link" after Razorpay redirects back
- iOS: Safari/Chrome opens the app link as a URL bar, not as a deep link

**Prevention:**
- Register mobile deep link return URLs in the Razorpay dashboard: `homechef://payment/success` and `homechef://payment/failure`.
- Use `expo-linking` to handle the incoming deep link in the payment result screen.
- Add explicit Android Intent Filters for the `homechef://` scheme in `app.json` `android.intentFilters`.
- Test on a real Android device — the simulator's deep link handling is more forgiving than real devices.

**Phase:** Phase 3 (checkout/payment). Brief architecture decision needed at the start of that phase.

---

### Pitfall 11: OTA Update (Expo Updates / EAS Update) Cannot Update Native Code

**What goes wrong:** Expo's OTA update mechanism (EAS Update / `expo-updates`) can only update the JavaScript bundle — it cannot update native modules, `app.json` config, or binary entitlements. If a Phase 2 update adds `expo-location` background task support, that requires a native rebuild and a new App Store/Play Store submission. Deploying it as an OTA update silently does nothing — the background task is missing from the binary.

**Why it happens:** Teams assume OTA = hotfix for everything. The boundary between "JS change" and "native change" is non-obvious.

**Warning signs:**
- Feature "works in dev" after `eas update` but is absent on production builds
- New `expo-*` package added to `package.json` but no new EAS Build triggered

**Prevention:**
- Rule of thumb: any new `expo-*` package, any `app.json` permission or plugin addition, any native module addition → requires a new EAS Build and store submission, not just OTA.
- Maintain a checklist of "native changes since last store release" at the start of every phase.
- For the Driver app specifically: background location is a native capability. It must be in the initial binary before any OTA delivery cadence is established.

**Phase:** Ongoing. Document the rule in phase plans. Especially relevant in Phase 2 (GPS) and Phase 3 (notifications).

---

## Minor Pitfalls

---

### Pitfall 12: Android Back Button Handling in Custom Navigation Stacks

**What goes wrong:** On Android, the hardware back button behavior must be explicitly handled in React Navigation. Food delivery apps have complex navigation (order tracking modal over home screen, etc.). If not handled, the back button exits the app instead of going back one screen, or pops through modal stacks unexpectedly.

**Prevention:** Add `BackHandler` listeners or use React Navigation's `useFocusEffect` + `BackHandler` for modal screens. Test on a real Android device from Phase 2 onward.

---

### Pitfall 13: Keyboard Avoiding View Inconsistency Between iOS and Android

**What goes wrong:** Forms (login, checkout, address entry) look fine on iOS with `KeyboardAvoidingView behavior="padding"` but break on Android where the correct behavior is `behavior="height"`. Both platforms need different values. The `@tesserix/native` design system may already handle this — but verify before assuming.

**Prevention:** Wrap all form screens in `KeyboardAvoidingView` with `Platform.OS === 'ios' ? 'padding' : undefined`. Test on both platforms from the first form screen built.

---

### Pitfall 14: Image Caching Not Configured for Chef/Menu Photos

**What goes wrong:** Chef profile photos and menu item images are served from the existing GCS bucket (based on `apps/api/services/storage.go`). Without image caching (`expo-image` or `react-native-fast-image`), every screen mount re-downloads the same images. Food delivery apps are image-heavy (chef grids, menu carousels) — uncached images produce visible jank and inflated bandwidth costs.

**Prevention:** Use `expo-image` (Expo-managed, replaces the deprecated `Image` component) which has built-in disk + memory caching. Add `blurhash` placeholders for chef/menu images to eliminate layout shift on load. This should be a standard library decision in Phase 1 (stack selection), not a retrofit.

---

### Pitfall 15: Auth Token Refresh Not Handled for Long-Running Sessions

**What goes wrong:** Mobile apps remain open for much longer than web sessions. A driver who starts the app in the morning may still have it open 12 hours later. The existing API uses JWT tokens (Bearer header). If the JWT expiry is short (e.g., 1 hour — common for web apps), the driver's API calls will start returning 401 at hour 1. The mobile app must implement silent token refresh. If it doesn't, the driver sees a blank screen or confusing error mid-delivery.

**Why it happens:** Web portals typically handle this via cookie-based sessions or short-lived page loads. Mobile apps have indefinite foreground sessions.

**Warning signs:**
- API calls fail with 401 after the user has been in the app for a while without refreshing
- No refresh token flow implemented in the existing API auth handlers

**Prevention:**
- Audit the JWT expiry configured in `apps/api/config/`. If < 8 hours, implement refresh token flow.
- On mobile: intercept 401 responses in the shared API client layer, attempt silent refresh, retry the original request. Only redirect to login if the refresh also fails.
- This is a shared API client concern — build it in the shared API client layer (Phase 1) so all three apps benefit.

**Phase:** Phase 1 (shared API client). Non-negotiable for the Driver app which has the longest active sessions.

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Phase 1: Monorepo scaffolding | Metro symlink resolution failure (#2) | Add `metro.config.js` with `watchFolders` before first `expo start` |
| Phase 1: EAS Build setup | Monorepo root not in EAS project root (#4) | Validate with `eas build --local` before first remote build |
| Phase 1: Push infra | FCM token type mismatch (#1) | Decide Expo Push Token vs raw FCM token before wiring `PUT /device-token` |
| Phase 1: Push infra | Single token per user (#6) | Re-register token on every foreground, add FCM error cleanup in `push.go` |
| Phase 1: Scaffolding | Deep link scheme collision (#9) | Set distinct schemes per app in `app.json` before any feature code |
| Phase 1: Shared API client | JWT expiry / token refresh (#15) | Audit JWT expiry config and implement refresh interceptor |
| Phase 1: Stack decision | `@tesserix/native` SDK version lock (#8) | Pin Expo SDK version matching `@tesserix/native` peer deps |
| Phase 2: GPS/maps | Battery drain from high-frequency polling (#3) | Use `distanceInterval` + `timeInterval`, add server-side rate limit |
| Phase 2: GPS/maps | Apple background location rejection (#5) | Gate `requestBackgroundPermissionsAsync` on "Go Online" action only |
| Phase 2: GPS/maps | Google Maps API key missing on iOS (#7) | Decide MapKit vs Google Maps at phase start, configure in `app.json` |
| Phase 2: Driver app | Background location is native, not OTA (#11) | Include in initial binary; document native-change rule |
| Phase 3: Checkout/payment | Razorpay deep link return broken on Android (#10) | Register `homechef://` in Android Intent Filters, test on real device |
| Ongoing | OTA updates cannot ship native changes (#11) | Maintain native-change checklist per phase |

---

## Sources

- Project codebase analysis: `apps/api/services/push.go`, `apps/api/models/delivery.go`, `apps/api/handlers/delivery.go`, `apps/api/routes/routes.go` — HIGH confidence (direct code inspection)
- `.planning/codebase/CONCERNS.md` — HIGH confidence (generated from codebase analysis)
- `.planning/PROJECT.md` — HIGH confidence (project requirements document)
- Expo documentation patterns (Metro monorepo config, EAS Build projectRoot, Expo SDK peer deps) — MEDIUM confidence (training data, Aug 2025 cutoff; verify against current Expo docs before implementation)
- React Native / Expo iOS App Store review requirements (background location, permission rationale) — MEDIUM confidence (Apple review guidelines are stable but verify current wording)
- FCM HTTP v1 API token formats (Expo Push Token vs raw FCM token distinction) — HIGH confidence (fundamental FCM architecture, stable)
