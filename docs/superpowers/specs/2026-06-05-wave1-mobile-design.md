# Wave 1 — Mobile Track Design

**Date:** 2026-06-05
**Status:** Drafted, awaiting execution
**Scope:** `apps/mobile-vendor` only (vendor app, iOS-first)
**Source plan:** `PROD-READINESS.md` → Wave 1 → Mobile (6 items)
**Estimated duration:** ~2 weeks solo, ~5 working days if subagent-parallelized
**Operator:** Mahesh (solo) or one subagent dispatched against this doc

## 1. Purpose & scope

Execution-ready plan for the **mobile track of Wave 1** — the six items under `PROD-READINESS.md` → Wave 1 → Mobile:

1. TestFlight pipeline (App Store Connect + `eas submit`)
2. Sentry crash reporting
3. Force-upgrade gate (min-version wall + `X-App-Version` header)
4. iOS 17+ privacy manifest (`PrivacyInfo.xcprivacy`)
5. APNs production cert in Firebase + EAS credentials
6. Bundle ID + production signing certs finalized in EAS

### Explicitly NOT in scope

- **Backend + infra tracks** — Sentry-go, rate limiting, idempotency, Razorpay HMAC, Dependabot, Trivy gate flip, Cloud SQL backups, `min_scale: 1` on auth-bff, Cloudflare WAF. Owned elsewhere.
- **Wave 2 mobile** — order cancellation, doc renewal, admin-requests inbox, image upload, reviews list, pause auto-resume, notification preferences, FSSAI date picker.
- **Wave 3 mobile** — GSTIN/HSN, statements, tax certificates, invoices, refund history.
- **Wave 4 mobile** — i18n + Hindi, locale picker, OTA via EAS Update, bundle audit, accessibility pass, App Store screenshots/listing/submission. This wave only builds the TestFlight rail; public submission is Wave 4.
- **Android** — vendor app is iOS-first. Android is touched only where free (e.g., `platform=android` query in the min-version endpoint). No Play Console work.

## 2. Prerequisites

Confirm before the first commit. Missing items become blockers (§8).

- **Apple Developer Program** membership for `tesserix-org`, active, operator has admin role.
- **App Store Connect** access for the same org, App Manager role minimum. App record for `com.homechef.vendor` may or may not exist — §4.1 handles both.
- **EAS** — `tesserix-org` is `owner` in `app.json`. Operator's Expo account is a member.
- **Sentry** project `homechef-vendor-mobile` (React Native platform) provisioned; DSN in hand.
- **Firebase** project access (owns the committed `GoogleService-Info.plist`); Editor role for the APNs upload.
- **APNs key (.p8)** from Apple Developer portal; Key ID + Team ID recorded. One key covers sandbox + production.
- **EAS CLI** `npx eas-cli@18.4.0` locally. Do not use 20.x (silent simulator-build failure per `NEXT-SESSION.md`).
- **Working tree clean** on `main` at or beyond `6d9f659`; two pending mobile fixes from `NEXT-SESSION.md` committed or stashed.

## 3. Cross-track contracts

These are **locked decisions** owned by other tracks. Repeated here so this doc stands alone.

### 3.1 Force-upgrade endpoint (backend ships, mobile consumes)

```
GET /api/v1/mobile/min-version?platform=ios|android&app=vendor

Response 200:
{
  "minVersion": "1.0.0",     // semver — block below this
  "latestVersion": "1.0.3",  // semver — informational, drives "update available" soft prompt later
  "storeUrl": "https://apps.apple.com/in/app/home-chef-vendor/id<APP_ID>"
}

Response 426 (Upgrade Required):
{ "error": "upgrade_required", "minVersion": "1.0.0", "storeUrl": "..." }
```

- No auth header required on the GET. Public.
- Backend may also return `426 Upgrade Required` on **any** authenticated endpoint when the client's `X-App-Version` is too old — mobile must handle that as a global axios interceptor, not only on the explicit poll.
- Mobile sends `X-App-Version: <expo.version>+<buildNumber>` on every API request as defense-in-depth (e.g., `1.0.0+12`).

### 3.2 Sentry env vars

- `EXPO_PUBLIC_SENTRY_DSN` — DSN string from Sentry project. Public is fine; DSNs are not secrets.
- `EXPO_PUBLIC_SENTRY_ENVIRONMENT` — `production` for the `production` EAS profile, `preview` for `preview`, `prod-sim` for the simulator-distributed build, `development` for dev client.
- **Release ID format:** `homechef-vendor@<expo.version>+<buildNumber>` — e.g., `homechef-vendor@1.0.0+12`. Set on the SDK init AND passed to the source-maps upload script so symbolication matches.
- **Tier:** Sentry free (5k errors/month). Sample rate 1.0 for errors; `tracesSampleRate: 0` (no perf transactions) to stay under the free quota.

## 4. Task-by-task spec

Each task lists: goal · files · code shape · validation · effort · risks.

### 4.1 TestFlight pipeline

**Goal:** A signed `.ipa` from the `production` EAS profile lands in TestFlight via `eas submit`, available to an internal testing group.

**Touched / new files:**

- `apps/mobile-vendor/eas.json` — populate empty `submit.production: {}`.
- App Store Connect (web) — create app record if missing; create internal testing group "Home Chef — Internal".
- Do NOT create `credentials.json`. EAS-managed credentials only; no local `.p8` checked in.

**Code shape:**

```jsonc
// eas.json — submit.production
"submit": {
  "production": {
    "ios": {
      "appleId": "<operator-apple-id@example.com>",       // resolved from EXPO_APPLE_ID env
      "ascAppId": "<App Store Connect numeric app ID>",   // from ASC after app record creation
      "appleTeamId": "<10-char Apple Team ID>",
      "language": "en-IN",
      "sku": "homechef-vendor-ios",
      "companyName": "Tesserix"
    }
  }
}
```

`appleId` and any App-Specific Password are NOT committed — pulled from local env / EAS secrets at submit time.

**Operational steps (in order, all from `apps/mobile-vendor/` cwd):**

1. App Store Connect: confirm/create app record for `com.homechef.vendor`. Region: India. Category: Food & Drink.
2. `npx eas-cli@18.4.0 credentials --platform ios --profile production` → "Let EAS manage" for dist cert + provisioning profile. Subsumes §4.6.
3. `npx eas-cli@18.4.0 build --platform ios --profile production` → store-signed `.ipa`.
4. `npx eas-cli@18.4.0 submit --platform ios --profile production --latest` → uploads to ASC; appears in TestFlight after 5–20 min processing.
5. ASC → TestFlight → Internal Testing → create group "Home Chef — Internal" → add operator + 1–2 seed chefs by Apple ID.

**Validation:**

- `eas build:list --platform ios --profile production --limit 1` shows status `finished` with an `.ipa` artifact.
- Build appears in TestFlight "iOS Builds" tab with status `Ready to Test` (export compliance answered: encryption = standard https only).
- Operator's iPhone installs via TestFlight app, launches against `https://vendors.fe3dr.com/api/v1`, can sign in as the seeded test chef.

**Effort:** 4–6h (Apple account paperwork is the unknown).

**Risks + mitigations:**

- **Apple Team ID + ASC App ID unknown until ASC record exists** → record creation is step 2; do not write `eas.json` until those IDs are in hand.
- **Export compliance** prompt blocks the build from going to testers → answer "Yes, uses encryption" + "Only standard https/TLS" once; `app.json` gets `ios.config.usesNonExemptEncryption: false` to skip future prompts.
- **Bundle ID collision** if another Tesserix app already claims `com.homechef.vendor` → check ASC app inventory first.

### 4.2 Sentry integration

**Goal:** Unhandled JS errors and native crashes in the vendor app land in Sentry within 30s with symbolicated stack traces.

**Decision: `@sentry/react-native` with the Expo config plugin (`@sentry/react-native/expo`), NOT the deprecated `sentry-expo`.** Sentry's current Expo guide uses this path; native code is already present (Firebase, expo-local-authentication) so the plugin slots in via the existing EAS prebuild flow. Source-map upload runs automatically on EAS Build when the plugin is configured.

**Touched / new files:**

- `apps/mobile-vendor/package.json` — add `@sentry/react-native`.
- `apps/mobile-vendor/app.json` — add Sentry config plugin (`organization`, `project`; no DSN — DSN is runtime env).
- `apps/mobile-vendor/app/_layout.tsx` — `Sentry.init` at module scope BEFORE any other import that might throw; wrap default export with `Sentry.wrap(RootLayout)`.
- `apps/mobile-vendor/eas.json` — add `SENTRY_AUTH_TOKEN` (EAS secret), `EXPO_PUBLIC_SENTRY_DSN`, `EXPO_PUBLIC_SENTRY_ENVIRONMENT` per profile.
- `apps/mobile-vendor/components/ErrorBoundary.tsx` — extend `componentDidCatch` to call `Sentry.captureException(error, { extra: errorInfo })`.

**Code shape — `_layout.tsx` init (module scope, BEFORE other imports that can throw):**

```typescript
import * as Sentry from '@sentry/react-native';
import Constants from 'expo-constants';

const version = Constants.expoConfig?.version ?? '0.0.0';
const buildNumber =
  Constants.expoConfig?.ios?.buildNumber ?? '0';

Sentry.init({
  dsn: process.env.EXPO_PUBLIC_SENTRY_DSN,
  environment: process.env.EXPO_PUBLIC_SENTRY_ENVIRONMENT ?? 'development',
  release: `homechef-vendor@${version}+${buildNumber}`,
  dist: buildNumber,
  enableAutoSessionTracking: true,
  tracesSampleRate: 0,                                  // perf off — free tier
  enabled: !!process.env.EXPO_PUBLIC_SENTRY_DSN,        // no DSN → no-op
});

export default Sentry.wrap(RootLayout);
```

**`app.json` plugin entry** (append to `plugins`): `["@sentry/react-native/expo", { "organization": "<slug>", "project": "homechef-vendor-mobile" }]`.

**`eas.json` env per profile:** add `EXPO_PUBLIC_SENTRY_DSN` (DSN is public, inline OK), `EXPO_PUBLIC_SENTRY_ENVIRONMENT`, and `SENTRY_AUTH_TOKEN: "$EAS_SECRET_SENTRY_AUTH_TOKEN"`. Provision the secret with `npx eas-cli@18.4.0 secret:create --scope project --name EAS_SECRET_SENTRY_AUTH_TOKEN --value <token>`.

**Validation:**

- Force a test error: temporary button in dev menu that calls `Sentry.nativeCrash()`. Crash report appears in Sentry within 30s with `release` matching the installed build.
- Source maps verified: stack trace shows real file paths (e.g., `app/_layout.tsx:412`), not `index.android.bundle:1:54321`.
- ErrorBoundary catches a forced render error → `captureException` lands in Sentry with `errorInfo` extras.
- `EXPO_PUBLIC_SENTRY_DSN` empty (dev client without env) → no network calls to Sentry; app starts normally.

**Effort:** 4h.

**Risks + mitigations:**

- **`Sentry.init` runs before fonts load — if init itself throws, the splash never appears.** Mitigation: wrap init in a try/catch; on failure log to console and continue.
- **Source-maps upload silently skipped** on EAS Build if `SENTRY_AUTH_TOKEN` is missing — log line in build output is the only signal. Always grep build log for `Uploaded source maps`.
- **PII leakage** in breadcrumbs — set `beforeSend` to scrub `Authorization` headers and any field named `phone` / `email` from event data before send.

### 4.3 Force-upgrade gate

**Goal:** Chef on an unsupported version cannot reach any screen except the upgrade wall, which links to the App Store.

**Touched / new files:**

- `packages/mobile-shared/src/api/client.ts` — extend `ApiClientOptions` with `appVersion` + `onUpgradeRequired` callbacks; inject `X-App-Version` on every request; on `426` call `onUpgradeRequired`.
- `apps/mobile-vendor/lib/api.ts` — pass the new options through.
- `apps/mobile-vendor/lib/version.ts` — new. Exports `currentAppVersion(): string` (`"1.0.0+12"` format) and `isBelowMinVersion(min, current): boolean` (semver compare, ignores build suffix for compare, includes it in header).
- `apps/mobile-vendor/hooks/useMinVersion.ts` — new React Query hook polling `/mobile/min-version?platform=ios&app=vendor` every 30 min while foregrounded.
- `apps/mobile-vendor/app/upgrade-required.tsx` — new Expo Router screen, modal-style, with brand copy + App Store button.
- `apps/mobile-vendor/app/_layout.tsx` — wire `onUpgradeRequired` to `router.replace('/upgrade-required')`; mount `useMinVersion` in `AppNavigator`; gate routing on min-version compare.

**Code shape — `client.ts` additions:**

```typescript
export interface ApiClientOptions {
  baseURL: string;
  getToken: () => string | null;
  onAuthFailure?: () => void;
  appVersion: string;                              // NEW — "1.0.0+12"
  onUpgradeRequired?: (storeUrl?: string) => void; // NEW
}

// request interceptor: config.headers['X-App-Version'] = appVersion;
// response interceptor, BEFORE the 401 branch:
if (error.response?.status === 426) {
  const storeUrl = (error.response.data as any)?.storeUrl;
  onUpgradeRequired?.(storeUrl);
  return Promise.reject(error);
}
```

**`useMinVersion.ts`** — React Query against `/mobile/min-version?platform=ios&app=vendor` using bare `fetch` (skips the 401 interceptor since the endpoint is public). `refetchInterval: 30 * 60 * 1000`, `refetchOnWindowFocus: true`, `retry: 1`.

**`upgrade-required.tsx` — Pressable per repo memory `feedback_ios_pressable_array_style.md`:** use object-style return + inner `<View>` for layout (function-returning-array drops flex/bg/padding on iOS).

```typescript
<Pressable
  onPress={() => Linking.openURL(storeUrl)}
  style={({ pressed }) => ({ opacity: pressed ? 0.85 : 1 })}
>
  <View style={styles.button}>
    <Text style={styles.buttonText}>Update on the App Store</Text>
  </View>
</Pressable>
```

**Behavior rules:**

- **Hard block:** while `minVersion > currentAppVersion`, `_layout.tsx` routing short-circuits `expectedPath` and replaces to `/upgrade-required`. No back-gesture escape.
- **Soft prompt** (`latestVersion > current`): NOT in this wave.
- **Endpoint unreachable:** fail open. The per-request `426` interceptor is the second line of defense — a chef offline can still read their dashboard.
- **426 from any other endpoint** → `onUpgradeRequired` → same screen.

**Validation:**

- Manually call `EXPO_PUBLIC_API_URL/mobile/min-version` from a curl on the sim's network → JSON shape matches §3.1.
- Set `app.json` version to `0.9.0` locally, rebuild, run against an endpoint returning `minVersion: 1.0.0` → upgrade wall blocks every screen including dashboard tap-through; back gesture does not escape.
- App Store button on the wall opens `apps.apple.com` in Safari (or App Store app if linkable).
- `X-App-Version` header present on every request — verified by tcpdump / proxy capture or by `console.log` in dev.

**Effort:** 6–8h.

**Risks + mitigations:**

- **Backend endpoint not ready when mobile lands** — guard with `enabled: !!process.env.EXPO_PUBLIC_MIN_VERSION_ENABLED` feature flag during the gap; flip to always-on once backend ships.
- **Semver edge case:** `1.0.0-beta.1` vs `1.0.0` — we control our own versions, so pin to non-prerelease semver only. Helper rejects prerelease strings and logs to Sentry.
- **Loop risk:** if `/upgrade-required` itself somehow triggers a 426, the interceptor fires again — break the cycle by checking `pathname === '/upgrade-required'` before re-routing.

### 4.4 iOS privacy manifest

**Goal:** Vendor app passes Apple's iOS 17+ privacy-manifest review by declaring camera, photo library, location, notifications, and any required-reason API usage.

**Touched / new files:**

- `apps/mobile-vendor/assets/PrivacyInfo.xcprivacy` — new file (XML plist).
- `apps/mobile-vendor/plugins/with-privacy-manifest.js` — new custom Expo config plugin that copies the file into the iOS project during prebuild (we're not ejecting).
- `apps/mobile-vendor/app.json` — register the plugin in the `plugins` array.

**Manifest contents (declare):**

- `NSPrivacyTracking: false` — we do not track across apps.
- `NSPrivacyAccessedAPITypes`:
  - `CategoryUserDefaults` reason `CA92.1` (AsyncStorage / SecureStore fallback)
  - `CategoryFileTimestamp` reason `C617.1` (image-picker, document-picker)
  - `CategorySystemBootTime` reason `35F9.1` (Sentry session attribution)
  - `CategoryDiskSpace` reason `E174.1` (Sentry + RN runtime)
- `NSPrivacyCollectedDataTypes`:
  - `EmailAddress` — Linked: true, Tracking: false, Purpose: `AppFunctionality`
  - `PreciseLocation` — Linked: true, Tracking: false, Purpose: `AppFunctionality`
  - `CrashData` — Linked: false, Tracking: false, Purpose: `AppFunctionality`

**Validation:**

- Build via `eas build --platform ios --profile production`; on the resulting `.ipa`, verify `PrivacyInfo.xcprivacy` is present at the top level of the app bundle: `unzip -p HomeChefVendor.ipa Payload/HomeChefVendor.app/PrivacyInfo.xcprivacy | head`.
- Submit to App Store Connect; the "App Privacy" section auto-populates from the manifest. No manual entries needed (or they match).
- TestFlight build does not warn `Missing Privacy Manifest` in build processing email.

**Effort:** 3h (most cost is iterating on the config plugin until prebuild emits the file).

**Risks + mitigations:**

- **SDK-level manifest mismatch:** Firebase, Sentry, expo-image, etc. each ship their own `PrivacyInfo.xcprivacy`. Apple aggregates these. We only declare app-level. If Apple flags missing reasons, add them post-hoc — no need to pre-declare third-party SDK reasons.
- **Config plugin fails silently** → verify `unzip -p` step is in the validation checklist; do not assume the file made it.

### 4.5 APNs production cert in Firebase + EAS

**Goal:** Production push notifications work end-to-end `homechef-api` → FCM → APNs → device.

**Files:** none — ops only.

**Steps:**

1. Apple Developer portal → Keys → "+" → APNs → download `.p8` (one-time download). Record Key ID + Team ID.
2. Firebase Console → Project Settings → Cloud Messaging → iOS app `com.homechef.vendor` → APNs Authentication Key → upload `.p8` + Key ID + Team ID.
3. The project uses `@react-native-firebase/messaging` (FCM token registration in `_layout.tsx`); EAS doesn't need a separate push key. Verify push capability is enabled in the provisioning profile via `eas credentials`.
4. Real-device test on TestFlight: send a `new_order` from `homechef-api`'s test endpoint, or send from Firebase Console → Cloud Messaging using the FCM token in device logs.

**Validation:** real-device TestFlight build receives push; Accept/Reject lockscreen actions fire (existing code); background-app push increments badge.

**Effort:** 2h.

**Risks:** Wrong Team ID → key rejected (cross-check against `eas.json` submit block). Bundle ID mismatch — `GoogleService-Info.plist` must match `com.homechef.vendor` (already verified in NEXT-SESSION.md).

### 4.6 Bundle ID + production signing certs finalized in EAS

**Goal:** EAS holds a managed distribution certificate + App Store provisioning profile for `com.homechef.vendor` on the `production` profile.

**Files:** none — EAS credential operation only.

**Steps:** from `apps/mobile-vendor/`: `npx eas-cli@18.4.0 credentials --platform ios --profile production` → "Build credentials" → "Set up a new build certificate" → "Let EAS handle it". EAS creates dist cert + App Store provisioning profile server-side. Confirm push capability is in the profile (overlaps §4.5). Run §4.1's build to validate.

**Validation:** `eas credentials --platform ios --profile production` lists a Distribution Certificate AND App Store Provisioning Profile, both `valid`, both bound to `com.homechef.vendor`. Build log shows `Using EAS-managed credentials`.

**Effort:** 1h (subsumed into §4.1 step 2; called out because PROD-READINESS DoD treats it as discrete).

**Risks:** Apple cert limit (3 dist certs/team) — EAS prompts to revoke oldest; fine for solo. Provisioning profile expires in 1 year — calendar reminder for 2027-06; EAS auto-regenerates on next build.

## 5. Sequencing within the track

Dependency graph: §4.2 Sentry, §4.3 force-upgrade, §4.4 privacy manifest are independent code work. §4.6 EAS credentials → §4.1 TestFlight build → §4.5 APNs (concurrent with §4.1).

**Solo order:** §4.2 → §4.3 → §4.4 → §4.6 → §4.1 → §4.5. Code first so the first prod build (§4.1) carries everything.

**Parallelized:** one subagent owns §4.2 + §4.3 (~2 days code). Operator owns §4.4 + §4.6 + §4.1 + §4.5 sequentially (~1.5 days console). Wall-clock ~3 days end-to-end.

## 6. Acceptance criteria mapped to Wave 1 DoD

The `PROD-READINESS.md` Wave 1 DoD lists five checks; only the first three apply to this track.

| DoD item | Verification |
|---|---|
| A real device installs via TestFlight (single chef test) | Operator's iPhone runs the prod-signed build from TestFlight, signs in as seeded chef, opens dashboard, sees the 2 demo orders. |
| Any panic in either Go service shows up in Sentry within 30s | (Backend track owns Go panics. **Mobile equivalent:** force `Sentry.nativeCrash()` from a dev menu button, see event in Sentry within 30s with `release: homechef-vendor@1.0.0+<n>`.) |
| 70-req/min flood → HTTP 429 after first 60 | (Backend track — mobile not involved.) |
| `gh run list` shows no failing builds; Trivy fail-on-critical | (Backend track.) |
| Cold-start 503 on auth-bff reproducibly gone | (Infra track.) |

**Mobile-specific additions to local DoD (not in PROD-READINESS but required to call this track done):**

- Chef on app version `0.9.0` cannot reach any screen, sees upgrade wall, taps button, lands in App Store.
- TestFlight build does not generate an Apple "Missing Privacy Manifest" warning email.
- `X-App-Version` header captured on a server-side log line for every authenticated request from a TestFlight install.

## 7. Rollback strategy

| Failure | Detection | Rollback |
|---|---|---|
| `Sentry.init` breaks startup | TestFlight build won't launch | Try/catch (in §4.2 risks); emergency OTA via EAS Update `production` that no-ops Sentry; worst case resubmit `1.0.1` with Sentry stubbed. |
| Min-version endpoint unreachable | API calls succeed, no min-version data | Fail open — already coded. No rollback. |
| Min-version returns wrong value (e.g., `1.0.5` when fleet is `1.0.0`) → fleet locked out | Sentry events from chefs drop to zero | Backend hotfix lowers `minVersion`; mobile needs no redeploy. Mitigation: server-side feature flag gating the wall for first 48h. |
| Privacy manifest rejected at ASC processing | Apple email within 15 min | Drop the config-plugin entry from `app.json`, rebuild, resubmit. SDK-bundled manifests cover the baseline. |
| EAS dist cert revoked | Next prod build fails `No valid distribution certificate` | `eas credentials` → recreate. Existing TestFlight builds stay valid until provisioning profile expiry. |
| APNs key wrong in Firebase | Real-device test push never arrives | Re-upload `.p8`. Propagates in ~5 min. |

## 8. Open questions / blockers

Confirm before §4 starts.

1. **Apple Developer Program** — `tesserix-org` membership active, operator has admin? **Hard blocker** — enrollment is 24h–7 days if missing.
2. **App Store Connect** — app record for `com.homechef.vendor` already exists? If not, who creates it (legal entity name + tax forms required). **Hard blocker.**
3. **Firebase project ownership** — operator has Editor role on the project that owns the committed `GoogleService-Info.plist`? **Hard blocker** for §4.5.
4. **Internal testing group composition** — operator-only for week 1, chefs in week 2? Or the two seed chefs from `NEXT-SESSION.md` from day one?
5. **Sentry org slug** — assumed `tesserix`. Override if different.
6. **`storeUrl` before the App Store listing exists** — for the first weeks the backend has no `apps.apple.com/...` URL. Recommended: return empty string + mobile shows "Update from TestFlight" copy; switch to real URL at Wave 4 submission.
7. **`X-App-Version` on `/auth/*` BFF routes** — auth-bff exempt from 426 so users can log in to receive the wall? Recommended yes; backend track owns the call.
