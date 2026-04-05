---
phase: 01-foundation-auth
plan: "02"
subsystem: auth
tags: [axios, zustand, expo-secure-store, expo-notifications, react-native, typescript, jwt, fcm]

requires:
  - phase: 01-01
    provides: Expo SDK 55 monorepo scaffold with apps/mobile-customer, apps/mobile-vendor, apps/mobile-delivery and Metro config

provides:
  - "packages/mobile-shared: typed axios API client factory with Bearer injection and 401/refresh interceptor"
  - "packages/mobile-shared: auth API functions (login, register, oauth, forgotPassword, resetPassword, logout)"
  - "packages/mobile-shared: domain types matching Go API shapes (User, AuthResponse, LoginRequest, etc.)"
  - "packages/mobile-shared: expo-secure-store wrappers (getAccessToken, setTokens, clearTokens) using canonical key strings"
  - "packages/mobile-shared: useAuthStore Zustand v5 store with hydrateFromStorage, setAuthResponse, logout, setBiometricsEnabled"
  - "packages/mobile-shared: usePushToken with getRawFCMToken (raw FCM via getDevicePushTokenAsync) and registerDeviceToken"
  - "packages/mobile-shared: @tesserix/native design token bridge re-exporting colors, spacing, typography, nativeTokens"
  - "22 vitest unit tests covering storage, API client, auth API, and theme token bridge"

affects:
  - "01-03: email auth screens import from @homechef/mobile-shared"
  - "01-04: social/biometric auth builds on useAuthStore and API functions"
  - "all future mobile plans: consume @homechef/mobile-shared for all API/auth/token operations"

tech-stack:
  added:
    - "vitest ^2.1.8 (devDependency in mobile-shared)"
    - "@homechef/mobile-shared package (private, no build step)"
  patterns:
    - "API client factory pattern: createApiClient() takes baseURL + getToken() + onAuthFailure callbacks"
    - "Token deduplication: refreshPromise singleton prevents concurrent 401 refresh races"
    - "Zustand v5 store with async actions that sync to expo-secure-store (hydrateFromStorage on startup)"
    - "FCM D-09 guard: ExponentPushToken prefix check in getRawFCMToken throws immediately on misconfiguration"
    - "@tesserix/native token bridge: fallback ?? {} pattern handles missing package gracefully"

key-files:
  created:
    - "packages/mobile-shared/package.json"
    - "packages/mobile-shared/tsconfig.json"
    - "packages/mobile-shared/vitest.config.ts"
    - "packages/mobile-shared/src/index.ts"
    - "packages/mobile-shared/src/types/user.ts"
    - "packages/mobile-shared/src/types/index.ts"
    - "packages/mobile-shared/src/utils/storage.ts"
    - "packages/mobile-shared/src/utils/index.ts"
    - "packages/mobile-shared/src/api/client.ts"
    - "packages/mobile-shared/src/api/auth.ts"
    - "packages/mobile-shared/src/api/index.ts"
    - "packages/mobile-shared/src/theme/tokens.ts"
    - "packages/mobile-shared/src/theme/index.ts"
    - "packages/mobile-shared/src/hooks/useAuth.ts"
    - "packages/mobile-shared/src/hooks/usePushToken.ts"
    - "packages/mobile-shared/src/hooks/index.ts"
    - "packages/mobile-shared/src/__tests__/storage.test.ts"
    - "packages/mobile-shared/src/__tests__/api-client.test.ts"
    - "packages/mobile-shared/src/__tests__/auth-api.test.ts"
    - "packages/mobile-shared/src/__tests__/theme-tokens.test.ts"
    - "packages/mobile-shared/src/__mocks__/@tesserix/native.ts"
  modified: []

key-decisions:
  - "No build step: package.json main points to ./src/index.ts — TypeScript sources consumed directly by apps via tsconfig path aliases"
  - "@tesserix/native listed as peerDependency (installed per-app) not direct dependency — avoids version conflicts"
  - "Token bridge uses import * as NativeTokens with ?? {} fallback — satisfies FOUND-03 even before @tesserix/native is installed"
  - "refreshPromise singleton in createApiClient prevents concurrent refresh races on simultaneous 401 responses"
  - "useAuthStore: biometricsEnabled stored in secure store with setBiometricsEnabled action for auth screen to toggle"

patterns-established:
  - "Pattern 1: createApiClient factory — all apps create instance with app-specific getToken from their own useAuthStore"
  - "Pattern 2: hydrateFromStorage called at app startup (_layout.tsx useEffect) before any protected route renders"
  - "Pattern 3: FCM token registration — getRawFCMToken() called after login, then registerDeviceToken(client, token)"
  - "Pattern 4: Token keys — always use STORAGE_KEYS.ACCESS_TOKEN etc. never raw strings"

requirements-completed: [FOUND-02, FOUND-03, FOUND-06, AUTH-07]

duration: 25min
completed: "2026-04-05"
---

# Phase 01 Plan 02: mobile-shared Package Summary

**axios API client factory with Bearer/401-refresh interceptors, Zustand auth store persisted to expo-secure-store, raw FCM token hook with ExponentPushToken guard, and @tesserix/native design token bridge — all in a zero-build-step shared package**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-04-05T14:20:00Z
- **Completed:** 2026-04-05T14:45:00Z
- **Tasks:** 2
- **Files created:** 21

## Accomplishments

- Created `packages/mobile-shared/` — the shared package consumed by all three mobile apps — with no build step (TypeScript sources consumed directly via tsconfig paths)
- Implemented typed API client factory (`createApiClient`) with Bearer token injection and 401 auto-refresh with concurrent-request deduplication via refreshPromise singleton
- Built Zustand v5 auth store (`useAuthStore`) with expo-secure-store persistence — `hydrateFromStorage()` loads tokens on app start, `logout()` clears both access and refresh tokens atomically
- Implemented `getRawFCMToken()` using `getDevicePushTokenAsync()` (D-09 compliance) with ExponentPushToken guard to prevent silent push failures
- Created `@tesserix/native` design token bridge (FOUND-03) re-exporting `colors`, `spacing`, `typography`, and `nativeTokens` with `?? {}` fallback for pre-install safety
- Wrote 22 vitest unit tests covering all storage, API client, auth API functions, and theme token bridge behavior

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: failing tests** - `bbb6092` (test)
2. **Task 1 GREEN: types, storage, API client, token bridge** - `93a881b` (feat)
3. **Task 2: useAuthStore, usePushToken, root index** - `d747f28` (feat)
4. **package-lock.json** - `b508768` (chore)

## Files Created/Modified

- `packages/mobile-shared/package.json` - Package manifest, peerDeps, no build step (main: ./src/index.ts)
- `packages/mobile-shared/tsconfig.json` - TypeScript config with moduleResolution: bundler, jsx: react-native
- `packages/mobile-shared/vitest.config.ts` - Test config with @tesserix/native alias pointing to mock
- `packages/mobile-shared/src/types/user.ts` - User, AuthResponse, LoginRequest, RegisterRequest, OAuthLoginRequest, ApiError
- `packages/mobile-shared/src/utils/storage.ts` - expo-secure-store wrappers with STORAGE_KEYS constants
- `packages/mobile-shared/src/api/client.ts` - createApiClient factory with Bearer injection + 401 interceptor
- `packages/mobile-shared/src/api/auth.ts` - loginWithEmail, registerUser, oauthLogin, refreshAuthToken, forgotPassword, resetPassword, logoutUser
- `packages/mobile-shared/src/theme/tokens.ts` - @tesserix/native token bridge, re-exports colors, spacing, typography, nativeTokens
- `packages/mobile-shared/src/hooks/useAuth.ts` - useAuthStore Zustand v5 store
- `packages/mobile-shared/src/hooks/usePushToken.ts` - getRawFCMToken (raw FCM D-09), registerDeviceToken
- `packages/mobile-shared/src/index.ts` - Root barrel re-exporting all 5 subpackages
- `packages/mobile-shared/src/__tests__/*.test.ts` - 22 unit tests (storage, api-client, auth-api, theme-tokens)
- `packages/mobile-shared/src/__mocks__/@tesserix/native.ts` - Vitest mock with realistic token shape

## Decisions Made

- No build step: package.json `main` points to `./src/index.ts` so apps consume TypeScript sources directly via tsconfig path aliases — eliminates build step complexity for a private monorepo package
- `@tesserix/native` as peerDependency: installed per-app, not inside mobile-shared, to avoid version mismatches across the three apps
- Token bridge uses `import * as NativeTokens` with `?? {}` fallback so the `nativeTokens` export is always a defined object even if `@tesserix/native` hasn't been installed yet — FOUND-03 satisfied without a hard install prerequisite for this plan
- `refreshPromise` singleton in `createApiClient` prevents race condition where three simultaneous 401 responses each try to call `/auth/refresh` — only one refresh fires, others await the same promise

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added vitest infrastructure and @tesserix/native mock**
- **Found during:** Task 1 (TDD RED phase)
- **Issue:** Plan specified TDD but no test framework was set up in mobile-shared. Without vitest config and an @tesserix/native mock, tests could not run at all.
- **Fix:** Added `vitest.config.ts` with `resolve.alias` pointing `@tesserix/native` to a mock file; added `src/__mocks__/@tesserix/native.ts` with realistic token shape; added vitest as devDependency.
- **Files modified:** `packages/mobile-shared/vitest.config.ts`, `packages/mobile-shared/src/__mocks__/@tesserix/native.ts`, `packages/mobile-shared/package.json`
- **Verification:** All 22 tests pass including theme-tokens tests that import from the mocked @tesserix/native
- **Committed in:** `93a881b` (Task 1 GREEN commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical infrastructure)
**Impact on plan:** Required for TDD approach to work. No scope creep — mock is only used in tests, production runtime uses the real @tesserix/native installed by each consuming app.

## Issues Encountered

- `@tesserix/native` not installed in node_modules (it's a peerDependency) — handled correctly via the `?? {}` fallback pattern and vitest alias mock. The production token bridge will receive real tokens once each mobile app installs the peer.

## Known Stubs

None — all exports are wired to real implementations. The `@tesserix/native` token bridge uses `?? {}` fallbacks but these are intentional design (graceful degradation when peer not installed), not stubs. The actual token values will be non-empty when each app installs `@tesserix/native`.

## Threat Flags

None — no new network endpoints, auth paths, file access patterns, or schema changes introduced. Token storage uses expo-secure-store (OS-level encryption). The ExponentPushToken guard (T-02-06) is implemented as required by the threat model.

## User Setup Required

None — no external service configuration required. Apps consuming `@homechef/mobile-shared` need to add the package to their tsconfig paths (handled in plan 01-03).

## Next Phase Readiness

- `@homechef/mobile-shared` is complete and ready for plan 01-03 (email auth screens) to import from it
- Apps need `tsconfig.json` path alias `@homechef/mobile-shared` → `../../packages/mobile-shared/src/index.ts` (this is added in plan 01-03 when building auth screens)
- `useAuthStore.getState().accessToken` is the correct call site for `getToken` in `createApiClient` — plan 01-03 will wire this up

---
## Self-Check: PASSED

All key files exist and all commits are present in the git log.

| Item | Status |
|------|--------|
| packages/mobile-shared/src/api/client.ts | FOUND |
| packages/mobile-shared/src/api/auth.ts | FOUND |
| packages/mobile-shared/src/hooks/useAuth.ts | FOUND |
| packages/mobile-shared/src/hooks/usePushToken.ts | FOUND |
| packages/mobile-shared/src/theme/tokens.ts | FOUND |
| packages/mobile-shared/src/utils/storage.ts | FOUND |
| packages/mobile-shared/src/index.ts | FOUND |
| .planning/phases/01-foundation-auth/01-02-SUMMARY.md | FOUND |
| Commit bbb6092 (test RED) | FOUND |
| Commit 93a881b (feat GREEN task 1) | FOUND |
| Commit d747f28 (feat task 2) | FOUND |
| Commit b508768 (chore package-lock) | FOUND |

---
*Phase: 01-foundation-auth*
*Completed: 2026-04-05*
