---
phase: 01-foundation-auth
plan: 04
subsystem: mobile-auth
tags: [google-signin, apple-signin, biometrics, expo-local-authentication, social-auth]
dependency_graph:
  requires: [01-03]
  provides: [social-login, biometric-lock]
  affects: [mobile-customer, mobile-vendor, mobile-delivery, mobile-shared]
tech_stack:
  added:
    - "@react-native-google-signin/google-signin v16 — native Google Sign-In"
    - "expo-apple-authentication v55 — native Apple Sign-In (iOS only)"
    - "expo-local-authentication v55 — Face ID / Touch ID biometric auth"
  patterns:
    - "AppState listener gated on isLoading === false to prevent race with hydrateFromStorage"
    - "Social auth passes native SDK ID tokens opaquely to Go API POST /api/v1/auth/oauth"
    - "Biometric unlock reads cached JWT from secure store — no server re-auth"
key_files:
  created:
    - packages/mobile-shared/src/hooks/useBiometrics.ts
  modified:
    - packages/mobile-shared/src/hooks/index.ts
    - packages/mobile-shared/src/screens/LoginScreen.tsx
    - apps/mobile-customer/app/(auth)/login.tsx
    - apps/mobile-vendor/app/(auth)/login.tsx
    - apps/mobile-delivery/app/(auth)/login.tsx
    - apps/mobile-customer/app/_layout.tsx
    - apps/mobile-vendor/app/_layout.tsx
    - apps/mobile-delivery/app/_layout.tsx
    - apps/mobile-customer/.env.example
    - apps/mobile-vendor/.env.example
    - apps/mobile-delivery/.env.example
decisions:
  - "useBiometricLock gates AppState listener on isLoading === false — prevents race condition where prompt fires before hydrateFromStorage resolves"
  - "Apple Sign-In guarded with Platform.OS === 'ios' — expo-apple-authentication is unavailable on Android"
  - "EXPO_PUBLIC_GOOGLE_WEB_CLIENT_ID is bundled in app binary (EXPO_PUBLIC_ prefix) — accepted per T-04-04 as Web Client ID is a public identifier, not a secret"
  - "Biometric auth unlocks cached JWT only — does not re-issue tokens; expired JWT handled by axios interceptor refresh flow"
metrics:
  duration: ~25 minutes
  completed: "2026-04-05"
  tasks_completed: 2
  files_changed: 11
---

# Phase 01 Plan 04: Social Login and Biometric Auth Summary

Google Sign-In (all platforms), Apple Sign-In (iOS only), and Face ID / Touch ID biometric lock wired into all three mobile apps using native SDKs with a hydration-gated AppState listener race condition fix.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Create useBiometrics hook and add social login buttons to LoginScreen | bb2bb9d | useBiometrics.ts, hooks/index.ts, LoginScreen.tsx |
| 2 | Wire Google, Apple, and biometric auth into all three app login routes | e0c6c0d | 6 app files (login.tsx, _layout.tsx x3), 3 .env.example |

## What Was Built

### useBiometrics hook (`packages/mobile-shared/src/hooks/useBiometrics.ts`)

Three exports:
- `checkBiometricCapability()` — checks hardware and enrollment via expo-local-authentication
- `authenticateWithBiometrics()` — prompts for Face ID / Touch ID, returns success boolean
- `useBiometricLock({ onLockout })` — registers AppState listener that prompts on app resume from background

Race condition fix: the `useEffect` in `useBiometricLock` is gated on `isLoading === false`. Since `isLoading` starts as `true` and only flips after `hydrateFromStorage()` resolves, the AppState listener is never registered before the store has read `isAuthenticated` and `biometricsEnabled` from secure store. `isLoading` is included in the useEffect dependency array alongside `isAuthenticated` and `biometricsEnabled`.

### LoginScreen updates (`packages/mobile-shared/src/screens/LoginScreen.tsx`)

Three new optional props added to `LoginScreenProps`:
- `onGoogleSignIn?: () => Promise<void>` — renders "Continue with Google" outline button
- `onAppleSignIn?: () => Promise<void>` — renders "Continue with Apple" button
- `onBiometricLogin?: () => Promise<void>` — renders "Use Face ID / Touch ID" button

All three appear below a divider after the sign-in button. Each handler catches errors and shows them in the existing top-level error banner. Props are optional — screens without them show no social buttons (backward compatible).

### Login routes (all three apps)

Pattern applied identically to `mobile-customer`, `mobile-vendor`, and `mobile-delivery`:
- `GoogleSignin.configure({ webClientId: EXPO_PUBLIC_GOOGLE_WEB_CLIENT_ID })` in `useEffect([], [])`
- `handleGoogleSignIn` — calls `GoogleSignin.signIn()`, sends `idToken` to `oauthLogin(api, { provider: 'google', token })`
- `handleAppleSignIn` — calls `AppleAuthentication.signInAsync()`, sends `identityToken` to `oauthLogin(api, { provider: 'apple', token })`
- `handleBiometricLogin` — calls `authenticateWithBiometrics()`, verifies cached `accessToken` exists, navigates to tabs
- Both social flows call `setAuthResponse(response)` and attempt FCM token registration (non-fatal)
- `onAppleSignIn` prop passed only when `Platform.OS === 'ios'`
- `onBiometricLogin` prop passed only when `biometricsEnabled === true`

### Root layouts (all three apps)

`useBiometricLock({ onLockout: () => router.replace('/(auth)/login') })` added after `hydrateFromStorage()` useEffect. The hook self-gates on `isLoading` — no extra guards needed in the layout.

### .env.example (all three apps)

`EXPO_PUBLIC_GOOGLE_WEB_CLIENT_ID=YOUR_GOOGLE_WEB_CLIENT_ID_HERE` added. Note: this is the Web Client ID from Google Cloud Console (not iOS/Android client ID).

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — all social and biometric flows are fully wired to real native SDKs and the Go API.

## Threat Flags

No new threat surface beyond what the plan's threat model covers (T-04-01 through T-04-07).

## Self-Check: PASSED

Files verified present:
- packages/mobile-shared/src/hooks/useBiometrics.ts — FOUND
- apps/mobile-customer/app/(auth)/login.tsx — FOUND (GoogleSignin, oauthLogin, Platform.OS)
- apps/mobile-vendor/app/_layout.tsx — FOUND (useBiometricLock)
- apps/mobile-delivery/app/_layout.tsx — FOUND (useBiometricLock)

Commits verified:
- bb2bb9d — FOUND (Task 1)
- e0c6c0d — FOUND (Task 2)
