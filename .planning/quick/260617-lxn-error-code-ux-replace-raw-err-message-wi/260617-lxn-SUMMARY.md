---
quick_id: 260617-lxn
title: "Error-code UX: stop surfacing raw err.message in auth screens"
status: complete
date: 2026-06-17
branch: fix/auth-error-code-ux
commit: d987f8d
---

# Summary — 260617-lxn

## What changed
`apps/mobile-vendor/app/(auth)/login.tsx` — imported `resolveAuthErrorMessage`
from `@homechef/mobile-shared/auth` and replaced the raw
`err instanceof Error ? err.message : '...'` + `Alert.alert('Sign-in failed', msg)`
pattern in all 4 catch blocks (Google, Apple, biometric, email/password) with
`Alert.alert('Sign-in failed', resolveAuthErrorMessage(err))`.

Net: −12 / +5 lines, 1 file.

## Key finding — scope was smaller than the handoff implied
The shared `LoginScreen` / `RegisterScreen` / `ForgotPasswordScreen`
(`packages/mobile-shared/src/screens/`) already wrap every auth callback in
try/catch and render `resolveAuthErrorMessage(e)` in an inline banner. So:
- mobile-customer login/register/forgot — already safe (delegate to shared). No change.
- mobile-delivery login/register — already safe. No change.
- mobile-vendor register/forgot — already safe (delegate to shared). No change.
- **mobile-vendor login.tsx** was the ONLY screen catching errors locally and
  surfacing the raw message via `Alert.alert`. Fixed.

## Verification
- `grep "err.message"` in vendor login → no matches.
- `tsc --noEmit` (vendor app): no errors referencing the changed file. (Pre-existing
  ~48 lucide-react-native type errors are unrelated — tracked separately.)

## Behavior note
The biometric handler previously threw friendly English strings
("No saved session found. Please log in with email."). These now route through
the mapper's generic fallback ("We couldn't sign you in. Please try again.").
Acceptable per the owner mandate (never surface raw text; prefer a safe generic).

## Follow-ups (out of scope for this PR)
1. Optionally align vendor login to the shared inline-banner pattern (drop local
   try/catch) so all 3 apps behave identically — visible UX change (Alert→banner).
2. App-wide non-auth `err.message` Alerts (onboarding/settings/support/orders/
   documents). Not auth flows; need a generic resolver, not the auth-specific one.
</content>
