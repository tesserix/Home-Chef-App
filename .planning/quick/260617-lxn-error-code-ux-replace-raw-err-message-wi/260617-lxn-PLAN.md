---
quick_id: 260617-lxn
title: "Error-code UX: stop surfacing raw err.message in auth screens"
status: ready
date: 2026-06-17
branch: fix/auth-error-code-ux
---

# Quick Task 260617-lxn — Error-code UX (auth screens)

## Goal
Users must never see raw error codes/messages (e.g. `auto_login_502`, Firebase
`auth/...` codes) during sign-in. Every user-facing auth error must pass through
`resolveAuthErrorMessage()` from `@homechef/mobile-shared/auth`.

## Investigation finding (scope is smaller than the handoff implied)
The shared `LoginScreen`, `RegisterScreen`, and `ForgotPasswordScreen`
(`packages/mobile-shared/src/screens/`) already wrap every callback
(`onLogin` / `onGoogleSignIn` / `onAppleSignIn` / `onBiometricLogin` /
`onRegister` / `onForgotPassword`) in try/catch and render
`resolveAuthErrorMessage(e)` in an inline banner.

Therefore:
- **mobile-customer** login/register/forgot — let errors propagate to the shared
  screen → already mapped. ✅ No change.
- **mobile-delivery** login/register — same. ✅ No change.
- **mobile-vendor** register/forgot — delegate to shared screens. ✅ No change.
- **mobile-vendor `app/(auth)/login.tsx`** — DIVERGES: it catches errors locally
  in all 4 handlers and shows `Alert.alert('Sign-in failed', err.message)` —
  leaking the raw code/message. ❌ This is the only real leak.

## Task 1 — Fix vendor login.tsx
- **files:** `apps/mobile-vendor/app/(auth)/login.tsx`
- **action:** Import `resolveAuthErrorMessage` from `@homechef/mobile-shared/auth`.
  In each of the 4 catch blocks (Google, Apple, biometric, email/password),
  replace `const msg = err instanceof Error ? err.message : '...'; Alert.alert('Sign-in failed', msg)`
  with `Alert.alert('Sign-in failed', resolveAuthErrorMessage(err))`.
- **verify:** `grep -n "err.message" apps/mobile-vendor/app/(auth)/login.tsx` → no matches;
  `tsc --noEmit` clean for the vendor app.
- **done:** No raw `err.message` reaches an Alert in vendor login; all 4 paths use the mapper.

## Out of scope (documented as PR follow-ups)
1. Optionally align vendor login to the shared inline-banner pattern (drop local
   try/catch entirely) so all 3 apps behave identically. Deferred — visible UX
   change (Alert → banner), not required to remove raw codes.
2. App-wide non-auth `err.message` Alerts (onboarding, settings, support, orders,
   documents). These are not auth flows; `resolveAuthErrorMessage` is auth-specific.
   A separate generic error-message resolver is needed — tracked as follow-up.

## Constraints
- Feature branch `fix/auth-error-code-ux` → PR to main. NEVER push to main directly.
</content>
</invoke>
