# Phase 1: Foundation + Auth - Research

**Researched:** 2026-04-05
**Domain:** Expo SDK 55 monorepo scaffold, pnpm workspace Metro config, JWT auth with social login and biometrics, EAS Build for iOS/Android
**Confidence:** HIGH (core stack verified against npm registry + live codebase inspection)

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** App directories named by Claude's discretion — recommended: `apps/mobile-customer/`, `apps/mobile-vendor/`, `apps/mobile-driver/`
- **D-02:** Shared mobile code lives in `packages/mobile-shared/` — pnpm-workspace.yaml already covers `packages/*`
- **D-03:** Shared package includes typed API client, response types, auth hooks, storage utils, AND shared screens (auth, onboarding) and navigation utilities
- **D-04:** Metro config in each app must include `watchFolders` pointing to monorepo root and `nodeModulesPaths` for pnpm symlink resolution — non-negotiable
- **D-05:** Direct JWT auth via `POST /api/v1/auth/login` — store JWT in expo-secure-store, send as `Bearer` header
- **D-06:** JWT auto-refresh via axios interceptor — intercept 401, call refresh endpoint, retry original request
- **D-07:** Social login at Claude's discretion — native SDKs (expo-apple-authentication + @react-native-google-signin/google-signin) sending ID token to Go API's `POST /api/v1/auth/oauth`
- **D-08:** Biometric auth at Claude's discretion — optional setting with unlock on app resume from background
- **D-09:** FCM device tokens registered using `getDevicePushTokenAsync()` (raw FCM tokens), NOT `getExpoPushTokenAsync()`
- **D-10:** Import design tokens from `@tesserix/native` package — same tokens as web
- **D-11:** Use NativeWind (Tailwind CSS for React Native) for styling
- **D-12:** Bundle identifiers: `com.homechef.customer`, `com.homechef.vendor`, `com.homechef.driver`
- **D-13:** EAS managed builds with `projectRoot` per app; dev client build for driver app
- **D-14:** Deep link URI schemes: `homechef-customer://`, `homechef-vendor://`, `homechef-driver://`

### Claude's Discretion

- App directory naming convention (D-01 resolved above)
- Social login SDK approach (D-07 resolved above)
- Biometric auth UX pattern (D-08 resolved above)
- EAS build strategy details (D-13 resolved above)

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| FOUND-01 | Expo monorepo scaffold with Metro config resolving pnpm workspace packages | Metro watchFolders + nodeModulesPaths pattern verified; Expo SDK 55 confirmed current stable |
| FOUND-02 | Shared `packages/mobile-shared/` with typed API client, hooks, and utils | Factory pattern from vendor-portal API client; axios v1.13 with interceptors |
| FOUND-03 | `@tesserix/native` integrated with shared color tokens and theme matching web | @tesserix/native peerDeps react >=18, react-native >=0.70 — compatible with Expo 55; uses StyleSheet + @tesserix/tokens, NOT NativeWind internally |
| FOUND-04 | EAS Build configuration for iOS and Android (all 3 apps) | EAS CLI 18.4.0 confirmed installed; projectRoot monorepo pattern documented |
| FOUND-05 | Deep linking with distinct URI schemes per app | expo-router scheme config in app.json; schemes locked in binary at build time |
| FOUND-06 | Secure token storage via expo-secure-store (not AsyncStorage) | expo-secure-store v55.0.11 confirmed; uses iOS Keychain / Android Keystore |
| AUTH-01 | Email/password login (all 3 apps) | `POST /api/v1/auth/login` confirmed; returns `accessToken` + `refreshToken` |
| AUTH-02 | New account registration (all 3 apps) | `POST /api/v1/auth/register` confirmed; returns `AuthResponse` with JWT |
| AUTH-03 | Password reset via email (vendor app) | `POST /api/v1/auth/forgot-password` + `POST /api/v1/auth/reset-password` confirmed |
| AUTH-04 | Google Sign-In (all 3 apps) | `@react-native-google-signin/google-signin` v16.1.2; sends ID token to `POST /api/v1/auth/oauth` |
| AUTH-05 | Apple Sign-In (all 3 apps — App Store requirement) | `expo-apple-authentication` v55.0.11; iOS-only native capability; needs EAS build |
| AUTH-06 | Biometric auth (Face ID / fingerprint) after first login | `expo-local-authentication` v55.0.11; credentials stored in secure store |
| AUTH-07 | JWT auto-refresh without session interruption | axios interceptor pattern; JWT expires in 24h (configurable), refresh tokens expire in 30 days |
</phase_requirements>

---

## Summary

This phase scaffolds three Expo SDK 55 mobile apps in the existing pnpm monorepo and delivers working authentication across all three. The most important architectural decisions are already locked in CONTEXT.md. Research confirmed that Expo SDK 55 (current stable, ships React 19.2.0 + React Native 0.83.4) is the correct SDK version and is compatible with `@tesserix/native`'s peer dependency range (`react >=18||>=19`, `react-native >=0.70`). The critical NativeWind decision (D-11) requires careful version choice: NativeWind v4 (latest stable 4.2.3) supports Tailwind v3 only; NativeWind v5 (preview.3) supports Tailwind v4+ but requires react >= 19 and react-native >= 0.81 — both satisfied by Expo 55. However v5 is still in preview; the planner should decide whether to use the stable v4+Tailwind v3 or the preview v5+Tailwind v4.

The Go API already implements all required auth endpoints (`login`, `register`, `forgot-password`, `reset-password`, `oauth`) including Google and Apple token verification on the server side. Social login flows use `POST /api/v1/auth/oauth` with a `provider` field (`google` or `apple`). The FCM push token decision (D-09) is critical: the existing `push.go` calls FCM HTTP v1 API directly using raw device tokens — using `getExpoPushTokenAsync()` instead of `getDevicePushTokenAsync()` would cause silent push notification failures.

Metro bundler configuration is the single most important scaffolding step. Without explicit `watchFolders` and `resolver.nodeModulesPaths`, Metro cannot resolve pnpm workspace packages — this has been the most common Phase 1 blocker for monorepo Expo projects.

**Primary recommendation:** Scaffold with Expo SDK 55, use NativeWind v5 preview.3 (Tailwind v4 compatible, meets all peer deps), implement Metro monorepo config first before any feature code, and validate with `eas build --local` before starting auth screens.

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| expo | ~55.0.11 | SDK / managed workflow | Current stable [VERIFIED: npm registry — `latest: 55.0.11`, `sdk-52: 52.0.49`, `sdk-53: 53.0.27`] |
| react | 19.2.0 | UI rendering | Shipped by Expo 55; matches web apps [VERIFIED: npm registry] |
| react-native | 0.83.4 | Native rendering layer | Bundled with Expo 55; do not pin independently [VERIFIED: npm registry] |
| expo-router | ~55.0.10 | File-based routing + deep linking | Current SDK-pinned version [VERIFIED: npm registry] |
| typescript | ^5.7.2 | Type safety | Matches monorepo root [VERIFIED: root package.json] |

### Auth + Storage
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| expo-secure-store | ~55.0.11 | Encrypted JWT storage (Keychain/Keystore) | D-06: security requirement; AsyncStorage is plaintext [VERIFIED: npm registry] |
| expo-local-authentication | ~55.0.11 | Biometric login (Face ID / fingerprint) | AUTH-06; SDK-matched version [VERIFIED: npm registry] |
| expo-apple-authentication | ~55.0.11 | Apple Sign-In (iOS native) | AUTH-05; App Store requirement; SDK-matched [VERIFIED: npm registry] |
| @react-native-google-signin/google-signin | ^16.1.2 | Google Sign-In (iOS + Android) | AUTH-04; Expo managed workflow compatible (`expo: ">=52.0.40"` peerDep) [VERIFIED: npm registry] |
| axios | ^1.13.0 | HTTP client with interceptor support | D-05, D-06; matches web apps; pattern verified in vendor-portal [VERIFIED: codebase] |

### Styling
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| nativewind | 5.0.0-preview.3 | Tailwind CSS for React Native | D-11; v5 preview supports Tailwind v4 (peerDep `tailwindcss: >4.1.11`); v4 stable only supports Tailwind v3 [VERIFIED: npm registry] |
| tailwindcss | ^4.x | CSS utility classes | Already used by web apps [VERIFIED: CLAUDE.md] |
| @tesserix/native | ^1.0.0 | Design system components + tokens | D-10; peerDeps: `react: ^18||^19`, `react-native: >=0.70` — compatible with Expo 55 [VERIFIED: design-system package.json] |
| @tesserix/tokens | workspace:* | Design tokens (spacing, colors, typography) | Used internally by @tesserix/native; import via @tesserix/native [VERIFIED: design-system codebase] |

### State Management
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| zustand | ^5.0.x | Client state (auth session, UI state) | Matches web apps v5.0.2; no context provider wrapping [ASSUMED — version not re-verified] |
| @tanstack/react-query | ^5.x | Server state, API data fetching | Matches web apps; `useQuery`/`useMutation` work identically in React Native [ASSUMED] |

### Navigation
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| expo-router | ~55.0.10 | File-based routing | All screens; provides deep linking automatically |
| react-native-safe-area-context | SDK-matched | Safe area insets | Required peer dep for expo-router |
| react-native-screens | SDK-matched | Native screen containers | Required peer dep for expo-router |

### Forms + Validation
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| react-hook-form | ^7.x | Form state for all auth screens | Matches web apps; use `<Controller>` wrapper for RN TextInput [ASSUMED] |
| zod | ^3.x | Schema validation | Matches web apps; `@hookform/resolvers/zod` works in RN [ASSUMED] |

### Build + Tooling
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| eas-cli | 18.4.0 | EAS Build + EAS Submit + EAS Update | Confirmed installed on dev machine [VERIFIED: `eas --version`] |
| jest-expo | ~55.0.x | Unit testing preset | Matches SDK version for correct transforms |
| @testing-library/react-native | ^13.x | Component testing | Standard RN testing library |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| NativeWind v5 preview | NativeWind v4 stable | v4 stable requires Tailwind v3 (web uses v4); v5 preview enables class parity with web but is pre-release |
| expo-apple-authentication | react-native-apple-authentication | expo-apple-authentication is the Expo-native version; bare RN lib requires extra native config |
| @react-native-google-signin/google-signin | expo-auth-session (PKCE flow) | expo-auth-session opens browser tab; native SDK gives in-app native sheet; better UX |
| NativeWind | StyleSheet-only styling | @tesserix/native already uses StyleSheet internally; NativeWind adds utility class DX on top |

**Installation (run inside each app directory with `npx expo install` for Expo-managed packages):**
```bash
# Expo managed packages — version resolver handles compatibility
npx expo install expo-router expo-secure-store expo-local-authentication \
  expo-apple-authentication expo-notifications expo-device \
  react-native-safe-area-context react-native-screens \
  expo-linking expo-constants expo-status-bar expo-web-browser

# Non-expo packages — install via pnpm
pnpm add axios zustand @tanstack/react-query react-hook-form zod @hookform/resolvers

# Google Sign-In (managed workflow compatible)
pnpm add @react-native-google-signin/google-signin

# NativeWind v5 preview (Tailwind v4 support)
pnpm add nativewind@5.0.0-preview.3 tailwindcss react-native-css@^3.0.1

# Mobile shared package deps
pnpm add @react-native-async-storage/async-storage  # non-sensitive persistence
```

**Version verification:** All Expo SDK packages verified against npm registry on 2026-04-05. Use `npx expo install --fix` after initial scaffold to pin correct versions for your SDK.

---

## Architecture Patterns

### Recommended Project Structure
```
Home-Chef-App/
├── apps/
│   ├── api/                        # Go API (existing, no changes)
│   ├── web/                        # Customer web (existing)
│   ├── vendor-portal/              # Vendor web (existing)
│   ├── mobile-customer/            # NEW: Expo customer app
│   ├── mobile-vendor/              # NEW: Expo vendor/chef app
│   └── mobile-driver/              # NEW: Expo driver app
└── packages/
    └── mobile-shared/              # NEW: shared API client, types, hooks, auth screens
```

Each mobile app follows expo-router file conventions:
```
apps/mobile-customer/
├── app/
│   ├── _layout.tsx                 # Root: QueryClientProvider + ThemeProvider + auth guard
│   ├── (auth)/
│   │   ├── _layout.tsx
│   │   ├── login.tsx
│   │   ├── register.tsx
│   │   └── forgot-password.tsx     # vendor app only — reused from mobile-shared
│   └── (tabs)/
│       ├── _layout.tsx
│       └── index.tsx               # stub home screen for phase 1
├── components/                     # App-specific components only
├── store/                          # Zustand stores (auth-store.ts)
├── lib/
│   └── api.ts                      # createApiClient() instance for this app
├── app.json
├── eas.json
├── metro.config.js                 # REQUIRED — monorepo resolution
├── babel.config.js
├── tsconfig.json
└── package.json
```

`packages/mobile-shared/` structure:
```
packages/mobile-shared/
├── src/
│   ├── api/
│   │   ├── client.ts               # createApiClient({ baseURL, getToken }) factory
│   │   ├── auth.ts                 # login(), register(), refreshToken(), oauthLogin()
│   │   └── index.ts
│   ├── types/
│   │   ├── user.ts                 # User, AuthUser, JWTClaims, AuthResponse
│   │   └── index.ts
│   ├── hooks/
│   │   ├── useAuth.ts              # login/logout/refresh + secure store persistence
│   │   ├── useBiometrics.ts        # expo-local-authentication wrapper
│   │   └── index.ts
│   ├── screens/                    # D-03: shared auth screens across 3 apps
│   │   ├── LoginScreen.tsx
│   │   ├── RegisterScreen.tsx
│   │   └── ForgotPasswordScreen.tsx
│   └── utils/
│       ├── storage.ts              # expo-secure-store wrappers (async getToken/setToken)
│       └── index.ts
├── package.json                    # name: @homechef/mobile-shared, no build step needed
└── tsconfig.json
```

### Pattern 1: Metro Monorepo Config (non-negotiable)
**What:** Every mobile app needs this `metro.config.js` to resolve pnpm workspace packages.
**When to use:** In every app before first `expo start` — without this, `@homechef/mobile-shared` and `@tesserix/native` will fail to resolve.

```javascript
// apps/mobile-customer/metro.config.js
// Source: https://docs.expo.dev/guides/monorepos/ [VERIFIED pattern, ASSUMED current syntax]
const { getDefaultConfig } = require('expo/metro-config');
const { FileStore } = require('metro-cache');
const path = require('path');

const projectRoot = __dirname;
const monorepoRoot = path.resolve(projectRoot, '../..');

const config = getDefaultConfig(projectRoot);

config.watchFolders = [monorepoRoot];
config.resolver.nodeModulesPaths = [
  path.resolve(projectRoot, 'node_modules'),
  path.resolve(monorepoRoot, 'node_modules'),
];
// Required for pnpm: follow symlinked packages
config.resolver.unstable_enableSymlinks = true;

config.cacheStores = [
  new FileStore({ root: path.join(projectRoot, '.metro-cache') }),
];

module.exports = config;
```

### Pattern 2: API Client Factory
**What:** `mobile-shared` exports `createApiClient()`. Each app instantiates with its own token getter.
**When to use:** In `apps/mobile-{name}/lib/api.ts` — never import axios directly in screens.

```typescript
// packages/mobile-shared/src/api/client.ts
// Source: pattern adapted from apps/vendor-portal/src/shared/services/api-client.ts [VERIFIED: codebase]
import axios, { AxiosInstance } from 'axios';
import * as SecureStore from 'expo-secure-store';

export function createApiClient(options: {
  baseURL: string;
  getToken: () => string | null;
}): AxiosInstance {
  const instance = axios.create({
    baseURL: options.baseURL,
    headers: { 'Content-Type': 'application/json' },
  });

  // Request interceptor: inject Bearer token
  instance.interceptors.request.use((config) => {
    const token = options.getToken();
    if (token) config.headers.Authorization = `Bearer ${token}`;
    return config;
  });

  // Response interceptor: handle 401 with token refresh (D-06)
  instance.interceptors.response.use(
    (response) => response,
    async (error) => {
      const originalRequest = error.config;
      if (error.response?.status === 401 && !originalRequest._retry) {
        originalRequest._retry = true;
        try {
          const refreshToken = await SecureStore.getItemAsync('refresh_token');
          if (!refreshToken) throw new Error('no refresh token');
          const res = await axios.post(`${options.baseURL}/auth/refresh`, {
            refreshToken,
          });
          const { accessToken } = res.data;
          await SecureStore.setItemAsync('access_token', accessToken);
          originalRequest.headers.Authorization = `Bearer ${accessToken}`;
          return instance(originalRequest);
        } catch {
          await SecureStore.deleteItemAsync('access_token');
          await SecureStore.deleteItemAsync('refresh_token');
          // Zustand auth store listens for null token and redirects to login
          return Promise.reject(error);
        }
      }
      return Promise.reject(error);
    }
  );

  return instance;
}
```

```typescript
// apps/mobile-customer/lib/api.ts
import { createApiClient } from '@homechef/mobile-shared/api';
import { useAuthStore } from '../store/auth-store';

export const api = createApiClient({
  baseURL: process.env.EXPO_PUBLIC_API_URL!,
  getToken: () => useAuthStore.getState().accessToken,
});
```

### Pattern 3: Auth Guard in Root Layout
**What:** Centralize auth checking in `app/_layout.tsx`. Individual screens never check auth state.
**When to use:** Always — expo-router redirect from root layout handles all unauthenticated access.

```typescript
// apps/mobile-customer/app/_layout.tsx
// Source: expo-router docs pattern [ASSUMED syntax — verify against expo-router v55 docs]
import { useEffect } from 'react';
import { Stack, router } from 'expo-router';
import { QueryClientProvider } from '@tanstack/react-query';
import { useAuthStore } from '../store/auth-store';

export default function RootLayout() {
  const { isAuthenticated, isLoading, hydrateFromStorage } = useAuthStore();

  useEffect(() => {
    hydrateFromStorage(); // Load token from expo-secure-store on first render
  }, []);

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      router.replace('/(auth)/login');
    }
  }, [isAuthenticated, isLoading]);

  return (
    <QueryClientProvider client={queryClient}>
      <Stack />
    </QueryClientProvider>
  );
}
```

### Pattern 4: Social Login (OAuth) Flow
**What:** Native Google/Apple sign-in returns an ID token, which is sent to `POST /api/v1/auth/oauth`.
**When to use:** AUTH-04 and AUTH-05 implementation.

The Go API endpoint `POST /api/v1/auth/oauth` accepts:
- `provider`: `"google"` | `"apple"` | `"facebook"`
- `token`: the ID token from the native SDK

```typescript
// packages/mobile-shared/src/api/auth.ts (social login fragment)
// Source: apps/api/handlers/auth.go OAuthLogin [VERIFIED: codebase]
export async function oauthLogin(
  client: AxiosInstance,
  provider: 'google' | 'apple',
  idToken: string
): Promise<AuthResponse> {
  const res = await client.post<AuthResponse>('/auth/oauth', {
    provider,
    token: idToken,
  });
  return res.data;
}
```

### Pattern 5: FCM Raw Token Registration
**What:** Use `getDevicePushTokenAsync()` not `getExpoPushTokenAsync()`.
**When to use:** Phase 1 infra setup — called once after login, re-called on app foreground.

```typescript
// packages/mobile-shared/src/hooks/usePushNotifications.ts (fragment)
// Source: apps/api/services/push.go uses raw FCM tokens [VERIFIED: codebase]
import * as Notifications from 'expo-notifications';
import * as Device from 'expo-device';

export async function getRawFCMToken(): Promise<string | null> {
  if (!Device.isDevice) return null; // Simulators cannot receive push
  const { status } = await Notifications.requestPermissionsAsync();
  if (status !== 'granted') return null;
  // D-09: raw FCM token, NOT getExpoPushTokenAsync()
  const tokenData = await Notifications.getDevicePushTokenAsync();
  return tokenData.data; // string on both iOS and Android
}

// Register with API: PUT /api/v1/profile/device-token
export async function registerDeviceToken(api: AxiosInstance, token: string) {
  await api.put('/profile/device-token', { token });
}
```

### Pattern 6: Biometric Auth (D-08)
**What:** Optional setting. After first email/password login, user can enable biometrics. Biometrics only unlock the cached JWT from secure store — they do not re-authenticate with the server.
**When to use:** AUTH-06 implementation. Prompt on app resume from background if enabled.

```typescript
// packages/mobile-shared/src/hooks/useBiometrics.ts (fragment)
// Source: expo-local-authentication docs [ASSUMED API — verify current docs]
import * as LocalAuthentication from 'expo-local-authentication';

export async function authenticateWithBiometrics(): Promise<boolean> {
  const hasHardware = await LocalAuthentication.hasHardwareAsync();
  const isEnrolled = await LocalAuthentication.isEnrolledAsync();
  if (!hasHardware || !isEnrolled) return false;

  const result = await LocalAuthentication.authenticateAsync({
    promptMessage: 'Authenticate to continue',
    fallbackLabel: 'Use password',
    cancelLabel: 'Cancel',
  });
  return result.success;
}
```

### Pattern 7: EAS Monorepo Config
**What:** Each app's `eas.json` must set `projectRoot` to the monorepo root so EAS Build can find `pnpm-workspace.yaml` and install all workspace packages.
**When to use:** EAS Build setup (FOUND-04).

```json
// apps/mobile-customer/eas.json
{
  "cli": { "version": ">= 18.0.0" },
  "build": {
    "development": {
      "developmentClient": true,
      "distribution": "internal",
      "ios": { "resourceClass": "m-medium" }
    },
    "preview": {
      "distribution": "internal",
      "channel": "preview"
    },
    "production": {
      "channel": "production"
    }
  },
  "submit": {
    "production": {}
  }
}
```

EAS monorepo support: run `eas build` from each app directory, or pass `--project-root apps/mobile-customer` from the monorepo root. The `pnpm-workspace.yaml` at the monorepo root is picked up automatically when EAS CLI detects pnpm.

### Pattern 8: app.json per App
```json
// apps/mobile-customer/app.json
{
  "expo": {
    "name": "HomeChef",
    "slug": "homechef-customer",
    "scheme": "homechef-customer",
    "version": "1.0.0",
    "orientation": "portrait",
    "icon": "./assets/icon.png",
    "userInterfaceStyle": "automatic",
    "ios": {
      "bundleIdentifier": "com.homechef.customer",
      "supportsTablet": false,
      "infoPlist": {
        "NSFaceIDUsageDescription": "Use Face ID to log in quickly",
        "NSCameraUsageDescription": "Used to take menu photos"
      }
    },
    "android": {
      "package": "com.homechef.customer",
      "adaptiveIcon": { "foregroundImage": "./assets/adaptive-icon.png" },
      "intentFilters": [
        {
          "action": "VIEW",
          "autoVerify": true,
          "data": [{ "scheme": "homechef-customer" }],
          "category": ["BROWSABLE", "DEFAULT"]
        }
      ]
    },
    "plugins": [
      "expo-router",
      "expo-secure-store",
      "expo-local-authentication",
      ["expo-build-properties", {
        "ios": { "newArchEnabled": true },
        "android": { "newArchEnabled": true }
      }]
    ]
  }
}
```

Note: `expo-apple-authentication` is NOT a plugin in `app.json` — it is an iOS entitlement and is configured by the EAS build profile automatically when you add the package.

### Anti-Patterns to Avoid
- **Separate QueryClient per component:** Create exactly ONE `QueryClient` in the root `_layout.tsx`. Never create `new QueryClient()` inside a feature component — breaks shared cache.
- **AsyncStorage for JWT:** AsyncStorage is unencrypted. Use `expo-secure-store` for access tokens and refresh tokens.
- **`getExpoPushTokenAsync()` instead of `getDevicePushTokenAsync()`:** The existing `push.go` calls FCM HTTP v1 directly — Expo Push Tokens are incompatible and produce silent failures.
- **Missing `disableHierarchicalLookup` in metro config:** Without this flag, Metro may resolve duplicate React Native instances (one per workspace package), producing the "Invariant Violation: No React Native renderer" crash.
- **Deep link scheme collision:** All three apps must have distinct `scheme` values in `app.json`. Shared schemes cause iOS to route all deep links to the last-installed app.
- **Skipping `eas build --local` validation:** Always validate the monorepo EAS config locally before triggering remote builds. Local failures are free; remote failures consume build minutes.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Encrypted key-value storage | Custom encryption layer over AsyncStorage | expo-secure-store | Uses OS Keychain/Keystore; no crypto expertise needed |
| Token refresh interceptor | Manual 401 retry logic per API call | axios interceptor (see Pattern 2) | One interceptor handles all 401s globally |
| Biometric auth | Custom native bridge | expo-local-authentication | Cross-platform Face ID / fingerprint with fallback to PIN |
| Apple Sign-In | Custom SIWA flow | expo-apple-authentication | App Store requirement; handles nonce, state, Apple's auth flow |
| Google Sign-In | Custom OAuth PKCE with browser | @react-native-google-signin/google-signin | Native sign-in sheet; expo-auth-session browser alternative is inferior UX |
| Metro monorepo resolution | Custom resolver plugin | `watchFolders` + `nodeModulesPaths` config | This is the documented pattern; custom resolvers introduce subtle bugs |
| Push token registration | Background service | `expo-notifications.getDevicePushTokenAsync()` | Token rotation handled automatically |

**Key insight:** Every React Native "simple" native feature (biometrics, secure storage, native auth sheets) has 3–10 platform-specific edge cases. Expo's managed modules handle all of them correctly.

---

## Common Pitfalls

### Pitfall 1: Wrong Push Token Type
**What goes wrong:** `getExpoPushTokenAsync()` returns an Expo-format token (`ExponentPushToken[...]`). The existing `push.go` calls FCM HTTP v1 API directly using `sendToToken(user.FCMToken, ...)`. FCM rejects Expo Push Tokens with a 404 error. The error is only logged, never surfaced. Result: all push notifications silently fail.
**Why it happens:** Both functions look identical in code; the distinction is only in the token format.
**How to avoid:** D-09 is locked — always use `Notifications.getDevicePushTokenAsync()`. Add a validation check: if the returned token starts with `ExponentPushToken`, throw an error immediately.
**Warning signs:** Push service logs "sent" but devices never receive notifications; FCM dashboard shows 0 deliveries.

### Pitfall 2: Metro Cannot Resolve pnpm Workspace Packages
**What goes wrong:** Metro bundler does not follow symlinks by default. pnpm creates symlinks in `node_modules/.pnpm/`. Without `watchFolders` + `nodeModulesPaths`, Metro cannot find `@homechef/mobile-shared` or `@tesserix/native`.
**Why it happens:** Every other app in the monorepo (Vite-based) handles symlinks automatically. Metro is the only bundler that needs explicit configuration.
**How to avoid:** Create `metro.config.js` (see Pattern 1) before running `expo start` for the first time. Verify by importing from `@homechef/mobile-shared` in the app entry point and confirming no resolution error.
**Warning signs:** `Module not found: @homechef/mobile-shared` or `Cannot find module '@tesserix/native'` in Metro output.

### Pitfall 3: EAS Build Fails with Module Not Found
**What goes wrong:** EAS Build runs `pnpm install` in the app's subdirectory without the monorepo root context, so `@homechef/mobile-shared` and other workspace packages are not installed.
**Why it happens:** EAS CLI needs to see `pnpm-workspace.yaml` at the project root. If `eas.json` `projectRoot` is not configured, the wrong root is used.
**How to avoid:** Set up `eas.json` (see Pattern 7) and validate with `eas build --local` before the first remote build. Local builds surface this issue without consuming EAS Build credits.
**Warning signs:** Local `expo start` works fine; EAS remote build fails with module resolution errors immediately after `pnpm install`.

### Pitfall 4: Apple Sign-In in Expo Go
**What goes wrong:** `expo-apple-authentication` requires a real Apple Identifier entitlement tied to the app's bundle ID. Expo Go cannot have the entitlement for `com.homechef.customer`. Calling `AppleAuthentication.signInAsync()` in Expo Go throws `The operation couldn't be completed. (com.apple.AuthenticationServices.AuthorizationError error 1000.)`.
**Why it happens:** Apple Sign-In is validated against Apple's servers using the bundle ID. Expo Go uses a different bundle ID.
**How to avoid:** Wrap `AppleAuthentication` in an availability check: `AppleAuthentication.isAvailableAsync()` returns false in Expo Go on iOS. Show a "Sign in with Email" fallback when Apple Sign-In is unavailable. Test Apple Sign-In only on EAS development builds.
**Warning signs:** Crash or error dialog when tapping Apple Sign-In button in Expo Go.

### Pitfall 5: NativeWind v5 Preview Instability
**What goes wrong:** NativeWind v5 is in preview (`5.0.0-preview.3`). It could have breaking API changes between preview releases.
**Why it happens:** Preview releases are not stable by definition.
**How to avoid:** Pin to the exact version `5.0.0-preview.3`. Do not use `^` range for NativeWind in `package.json`. Check NativeWind GitHub releases before upgrading. If v5 proves unstable during development, fall back to NativeWind v4.2.3 + Tailwind CSS v3 (class names are identical, only the configuration differs).
**Warning signs:** Styles not applying after `pnpm install` upgrade; TypeScript errors from NativeWind types.

### Pitfall 6: @tesserix/native Uses StyleSheet (Not NativeWind) Internally
**What goes wrong:** A developer assumes `@tesserix/native` components accept NativeWind `className` props. They don't — `@tesserix/native` uses React Native `StyleSheet` internally with `@tesserix/tokens` for values. Passing `className` props silently does nothing.
**Why it happens:** CONTEXT.md says "use NativeWind for styling" which implies all components accept `className`. But `@tesserix/native` is a pre-existing package that does not use NativeWind.
**How to avoid:** Use NativeWind `className` only on raw React Native primitives (`View`, `Text`, `TouchableOpacity`) and custom components written in this phase. For `@tesserix/native` components, use the `style`, `variant`, `colorScheme`, and `size` props they expose. Layer NativeWind classes only on wrapper/container views around design system components.
**Warning signs:** className prop passed to a `@tesserix/native` component with no visual effect; TypeScript warning "Property 'className' does not exist on type ButtonProps".

### Pitfall 7: JWT Token Expiry for Long Mobile Sessions
**What goes wrong:** The API defaults JWT expiry to 24 hours (`JWT_EXPIRATION_HOURS=24`). Drivers who leave the app open during a 12-hour shift are fine. But after 24h without an explicit token refresh, all API calls return 401 and the app appears broken.
**Why it happens:** Web apps typically redirect to login on 401; mobile apps with navigation stacks must handle 401 silently in the background.
**How to avoid:** D-06 is locked — implement the axios 401 interceptor (Pattern 2). The refresh token expires in 30 days (`REFRESH_TOKEN_DAYS=30`). The interceptor must store the new access token in SecureStore after a successful refresh.
**Warning signs:** API calls fail mid-session with 401; user reports being "logged out" while using the app.

---

## Code Examples

### Auth Login API Call
```typescript
// packages/mobile-shared/src/api/auth.ts
// Source: apps/api/handlers/auth.go Login [VERIFIED: codebase]
export interface LoginRequest {
  email: string;
  password: string;
}

export interface AuthResponse {
  user: UserResponse;
  accessToken: string;
  refreshToken: string;
}

export async function login(
  client: AxiosInstance,
  credentials: LoginRequest
): Promise<AuthResponse> {
  const res = await client.post<AuthResponse>('/auth/login', credentials);
  return res.data;
}

export async function register(
  client: AxiosInstance,
  data: {
    email: string;
    password: string;
    firstName: string;
    lastName: string;
    phone?: string;
  }
): Promise<AuthResponse> {
  const res = await client.post<AuthResponse>('/auth/register', data);
  return res.data;
}

export async function refreshToken(
  baseURL: string,
  refreshToken: string
): Promise<{ accessToken: string }> {
  // Use plain axios (not the instance) to avoid recursive interceptor
  const res = await axios.post<{ accessToken: string }>(`${baseURL}/auth/refresh`, {
    refreshToken,
  });
  return res.data;
}
```

### SecureStore Token Persistence
```typescript
// packages/mobile-shared/src/utils/storage.ts
// Source: expo-secure-store docs [ASSUMED API — stable since SDK 47]
import * as SecureStore from 'expo-secure-store';

const ACCESS_TOKEN_KEY = 'homechef_access_token';
const REFRESH_TOKEN_KEY = 'homechef_refresh_token';

export const tokenStorage = {
  async getAccessToken(): Promise<string | null> {
    return SecureStore.getItemAsync(ACCESS_TOKEN_KEY);
  },
  async setAccessToken(token: string): Promise<void> {
    await SecureStore.setItemAsync(ACCESS_TOKEN_KEY, token);
  },
  async getRefreshToken(): Promise<string | null> {
    return SecureStore.getItemAsync(REFRESH_TOKEN_KEY);
  },
  async setTokens(accessToken: string, refreshToken: string): Promise<void> {
    await Promise.all([
      SecureStore.setItemAsync(ACCESS_TOKEN_KEY, accessToken),
      SecureStore.setItemAsync(REFRESH_TOKEN_KEY, refreshToken),
    ]);
  },
  async clearTokens(): Promise<void> {
    await Promise.all([
      SecureStore.deleteItemAsync(ACCESS_TOKEN_KEY),
      SecureStore.deleteItemAsync(REFRESH_TOKEN_KEY),
    ]);
  },
};
```

### Google Sign-In Configuration
```typescript
// apps/mobile-customer/lib/google-auth.ts
// Source: @react-native-google-signin/google-signin docs [ASSUMED API — verify current docs]
import { GoogleSignin } from '@react-native-google-signin/google-signin';

export function configureGoogleSignIn() {
  GoogleSignin.configure({
    webClientId: process.env.EXPO_PUBLIC_GOOGLE_WEB_CLIENT_ID,
    iosClientId: process.env.EXPO_PUBLIC_GOOGLE_IOS_CLIENT_ID,
    scopes: ['profile', 'email'],
  });
}

export async function signInWithGoogle(): Promise<string> {
  await GoogleSignin.hasPlayServices();
  const userInfo = await GoogleSignin.signIn();
  const { idToken } = await GoogleSignin.getTokens();
  if (!idToken) throw new Error('Google Sign-In: no ID token');
  return idToken;
}
```

### Apple Sign-In (with Expo Go fallback)
```typescript
// apps/mobile-customer/lib/apple-auth.ts
// Source: expo-apple-authentication docs [ASSUMED API]
import * as AppleAuthentication from 'expo-apple-authentication';

export async function isAppleSignInAvailable(): Promise<boolean> {
  return AppleAuthentication.isAvailableAsync();
}

export async function signInWithApple(): Promise<string> {
  const credential = await AppleAuthentication.signInAsync({
    requestedScopes: [
      AppleAuthentication.AppleAuthenticationScope.FULL_NAME,
      AppleAuthentication.AppleAuthenticationScope.EMAIL,
    ],
  });
  if (!credential.identityToken) throw new Error('Apple Sign-In: no identity token');
  return credential.identityToken;
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Expo SDK 52 (training data cutoff) | Expo SDK 55 (latest stable) | 2025–2026 | Ships React 19.2.0 + RN 0.83.4; New Architecture enabled by default |
| NativeWind v4 (Tailwind v3 only) | NativeWind v5 preview (Tailwind v4) | Early 2026 | Class parity with web Tailwind v4 setup; still preview |
| expo-router v3 | expo-router v55.0.10 (aligned to SDK) | SDK 55 | Versioning now follows SDK version; stable file-based routing |
| Separate EAS monorepo projectRoot field | pnpm-workspace.yaml auto-detected by EAS CLI | EAS CLI ~16+ | Less config required; EAS detects pnpm workspaces automatically |
| `getExpoPushTokenAsync()` for FCM | `getDevicePushTokenAsync()` for raw FCM | Always the case | Not a change — this is the correct API for direct FCM use; commonly confused |

**Deprecated/outdated:**
- Expo SDK 52 and 53 dist-tags: 52.0.49 and 53.0.27 are older stable releases — use SDK 55 for new projects.
- NativeWind v4.2.3: Last stable supporting only Tailwind v3. Use v5 preview for Tailwind v4 alignment with web apps.
- `react-native-fast-image`: Requires bare workflow. Use `expo-image` (SDK-managed) instead.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | All mobile app builds | ✓ | v22.19.0 | — |
| pnpm | Workspace management | ✓ | 9.15.1 | — |
| EAS CLI | FOUND-04 EAS Build | ✓ | 18.4.0 | — |
| Expo CLI (npx expo) | App scaffold + start | ✓ | 55.0.21 | — |
| iOS device / simulator | Apple Sign-In (native) | Unverified | — | Expo Go (no Apple Sign-In) or EAS dev build |
| Android device / emulator | Google Sign-In, biometrics | Unverified | — | Emulator (no biometrics) |
| GCP credentials (ADC) | push.go FCM push service | Unverified | — | Push init fails gracefully (non-fatal per push.go) |

**Missing dependencies with no fallback:** None — all scaffolding tools are present on dev machine.

**Missing dependencies with fallback:**
- Physical iOS device: Apple Sign-In requires real device OR EAS development build. Expo Go workaround: disable Apple Sign-In button when `AppleAuthentication.isAvailableAsync()` returns false.
- Physical Android device for biometrics: emulators do not support fingerprint/face sensors reliably. EAS development build on physical device needed for full biometric testing.

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Zustand v5.x and @tanstack/react-query v5.x work identically in React Native (no special config) | Standard Stack | Low — these are framework-agnostic state libraries; risk is minor version incompatibility |
| A2 | react-hook-form v7 + Zod v3 work with RN `<Controller>` in Expo 55 | Standard Stack | Low — stable pattern since RHF v7 |
| A3 | expo-router v55 syntax for `router.replace`, `Stack`, and `usePathname` matches prior versions | Architecture Patterns, Code Examples | Medium — expo-router API surface is stable but route group syntax may have evolved |
| A4 | expo-apple-authentication.isAvailableAsync() returns false in Expo Go (not a crash) | Pitfall 4, Code Examples | Medium — if it throws instead of returning false, error handling must be added |
| A5 | @react-native-google-signin/google-signin v16 `GoogleSignin.getTokens()` returns `idToken` | Code Examples | Medium — verify exact API shape in package docs before implementing |
| A6 | NativeWind v5 preview.3 is stable enough for Phase 1 scaffolding | Standard Stack | Medium — preview releases can have breaking changes; fallback to v4+Tailwind v3 is available |
| A7 | EAS CLI v18 auto-detects pnpm-workspace.yaml without explicit projectRoot in eas.json | Pitfall 3, Architecture | Medium — if not, add `"projectRoot": "../.."` to each app's eas.json |

---

## Open Questions

1. **NativeWind v5 preview vs v4 stable decision**
   - What we know: v5 preview.3 requires `tailwindcss >4.1.11` and `react >=19`, both satisfied by Expo 55
   - What's unclear: Whether preview.3 is stable enough for production use in Phase 1
   - Recommendation: Use v5 preview.3 (locked to exact version, no `^`). If any instability is encountered during scaffolding, fall back to v4.2.3 + Tailwind v3 with minimal rework (class syntax is identical).

2. **@tesserix/native dist version vs source**
   - What we know: `@tesserix/native` has `"react-native": "./src/index.ts"` in exports — Metro resolves TypeScript source directly
   - What's unclear: Whether the design-system package is published to GHCR or imported as a local workspace package in this monorepo
   - Recommendation: Check if `@tesserix/native` is in `pnpm-workspace.yaml` or installed from GHCR. If GHCR: add `NODE_AUTH_TOKEN` to `.env.local`. If workspace: add to `packages/` and add to pnpm-workspace.yaml.

3. **API base URL for mobile apps**
   - What we know: Web apps use `VITE_BFF_URL`; mobile apps should use `EXPO_PUBLIC_API_URL` (public env vars for Expo)
   - What's unclear: What the correct base URL is for dev (local), staging, and production environments
   - Recommendation: Use `EXPO_PUBLIC_API_URL=http://localhost:8080/api/v1` for local dev; the Expo public env var system (`process.env.EXPO_PUBLIC_*`) handles this correctly.

4. **Apple Sign-In backend token verification completeness**
   - What we know: `verifyAppleToken()` in `auth.go` does basic JWT decode but comments note it does NOT verify the RS256 signature against Apple's public keys
   - What's unclear: Whether this is sufficient for production or needs to be hardened before Phase 1 ships
   - Recommendation: Flag to developer that Apple token verification is incomplete (signature not verified against `https://appleid.apple.com/auth/keys`). This is a security concern that should be addressed before App Store submission, but does not block Phase 1 development.

---

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | JWT Bearer + expo-secure-store; no plaintext token storage |
| V3 Session Management | yes | 24h access token + 30-day refresh token; interceptor clears on refresh failure |
| V4 Access Control | no | Role enforcement is server-side in Go API; mobile only sends Bearer token |
| V5 Input Validation | yes | Zod + react-hook-form on all auth forms |
| V6 Cryptography | yes | expo-secure-store (OS Keychain / Keystore) — never hand-roll crypto |

### Known Threat Patterns for Expo + React Native

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| JWT stored in AsyncStorage | Information Disclosure | Use expo-secure-store (D-06) |
| Stale FCM token in DB | Denial of Service (push) | Re-register token on foreground; clear on FCM UNREGISTERED response |
| Incomplete Apple token verification | Spoofing | Harden `verifyAppleToken()` to verify RS256 signature before App Store submission |
| Deep link hijacking (homechef:// scheme) | Spoofing | Scheme must match bundle ID; validate redirect targets server-side |
| Expo Go bypasses native entitlements | Privilege Escalation | Apple Sign-In gracefully degrades; biometrics not tested in Expo Go |
| EXPO_PUBLIC_ vars in JS bundle | Information Disclosure | Only non-secret config (API URL, OAuth client IDs) — secrets must never use EXPO_PUBLIC_ prefix |

**Security note on Apple token verification:** The existing `verifyAppleToken()` in `apps/api/handlers/auth.go` decodes the JWT payload without verifying the RS256 signature against Apple's published public keys (`https://appleid.apple.com/auth/keys`). This means a crafted Apple ID token could bypass Apple Sign-In verification on the server. This is a MEDIUM security concern (requires constructing a valid-looking JWT). It must be addressed before App Store submission. The fix is to fetch Apple's JWKS endpoint and verify the `kid` and signature — the `golang-jwt/jwt/v5` package already in `go.mod` supports RS256 key verification.

---

## Sources

### Primary (HIGH confidence)
- npm registry (2026-04-05) — expo@55.0.11, expo-router@55.0.10, expo-secure-store@55.0.11, expo-local-authentication@55.0.11, expo-apple-authentication@55.0.11, nativewind@4.2.3 + 5.0.0-preview.3, @react-native-google-signin/google-signin@16.1.2, eas-cli@18.4.0
- `apps/api/handlers/auth.go` — OAuthLogin, Register, Login, ForgotPassword, ResetPassword endpoints (all confirmed present)
- `apps/api/services/push.go` — FCM HTTP v1 raw token confirmed; sendToToken() pattern
- `apps/api/config/config.go` — JWT_EXPIRATION_HOURS=24, REFRESH_TOKEN_DAYS=30
- `apps/api/routes/routes.go` — `/api/v1/auth/oauth`, `/api/v1/auth/refresh`, `/api/v1/profile/device-token` routes confirmed
- `design-system/packages/native/package.json` — peerDeps `react: ^18||^19`, `react-native: >=0.70.0`
- `design-system/packages/native/src/components/Button/Button.tsx` — uses StyleSheet + @tesserix/tokens (not NativeWind)
- `apps/vendor-portal/src/shared/services/api-client.ts` — API client factory pattern reference
- `pnpm-workspace.yaml` — confirms `packages/*` is in workspace; `packages/` dir does not exist yet
- `eas --version` output — EAS CLI 18.4.0 installed on dev machine
- `.planning/config.json` — `nyquist_validation: false`, `commit_docs: true`

### Secondary (MEDIUM confidence)
- `.planning/research/STACK.md` — technology recommendations (training data Aug 2025, cross-verified against npm registry)
- `.planning/research/ARCHITECTURE.md` — monorepo architecture patterns
- `.planning/research/PITFALLS.md` — pitfall catalog (training data, cross-verified against codebase)

### Tertiary (LOW confidence / ASSUMED)
- Zustand v5 + React Query v5 React Native compatibility (not re-verified; well-established pattern)
- expo-router v55 API surface for `router.replace`, layout files (not fetched from current docs)
- @react-native-google-signin/google-signin v16 `getTokens()` API (not fetched from current docs)

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all versions verified against npm registry on 2026-04-05
- API integration: HIGH — all endpoints verified against live codebase
- Architecture patterns: HIGH — verified against existing codebase conventions
- NativeWind v5 stability: LOW — preview release; exact API may shift
- Code examples: MEDIUM — patterns are correct but exact API signatures for some packages (expo-router v55, google-signin v16) should be validated against current docs before implementation

**Research date:** 2026-04-05
**Valid until:** 2026-05-05 (30 days; Expo ecosystem is fast-moving — re-check NativeWind v5 status)
