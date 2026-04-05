# Phase 1: Foundation + Auth - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-05
**Phase:** 01-Foundation + Auth
**Areas discussed:** Monorepo structure, Auth strategy, Design system setup, EAS Build config

---

## Monorepo Structure

| Option | Description | Selected |
|--------|-------------|----------|
| mobile-customer, mobile-vendor, mobile-driver | Clear prefix groups them together in apps/ | |
| customer-app, chef-app, driver-app | Shorter, role-focused naming | |
| You decide | Claude picks the best convention | ✓ |

**User's choice:** You decide
**Notes:** Claude has discretion on directory naming convention

---

| Option | Description | Selected |
|--------|-------------|----------|
| packages/mobile-shared | New workspace package in packages/ — pnpm-workspace.yaml already covers packages/* | ✓ |
| apps/mobile-shared | Keep all mobile code together under apps/ | |
| You decide | Claude picks based on monorepo conventions | |

**User's choice:** packages/mobile-shared

---

| Option | Description | Selected |
|--------|-------------|----------|
| API client + types + hooks | Typed HTTP client, response types matching Go API, auth/storage hooks | |
| Everything reusable | API client, types, hooks, plus shared screens (auth, onboarding) and navigation utils | ✓ |
| You decide | Claude determines the right split | |

**User's choice:** Everything reusable

---

## Auth Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Direct JWT (Recommended) | Login via POST /api/v1/auth/login, store JWT in expo-secure-store, Bearer header on all requests | ✓ |
| Via auth-bff | Route through auth-bff OIDC gateway like Tesserix platform apps | |
| You decide | Claude picks based on existing API patterns | |

**User's choice:** Direct JWT (Recommended)

---

| Option | Description | Selected |
|--------|-------------|----------|
| Native SDKs + API | Use expo-apple-authentication + Google Sign-In SDK, send ID token to Go API | |
| Web-based OAuth | Use expo-auth-session with browser redirect flow | |
| You decide | Claude picks the best approach | ✓ |

**User's choice:** You decide
**Notes:** Claude has discretion on social login implementation

---

| Option | Description | Selected |
|--------|-------------|----------|
| On app resume | Require Face ID/fingerprint when returning to app after background | |
| On token refresh only | Only require biometric when JWT expires | |
| Optional setting | User toggles biometric on/off in settings | |
| You decide | Claude picks the best UX | ✓ |

**User's choice:** You decide
**Notes:** Claude has discretion on biometric auth trigger pattern

---

## Design System Setup

| Option | Description | Selected |
|--------|-------------|----------|
| Import from @tesserix/native | @tesserix/native already has tokens — import colors, spacing, typography directly | ✓ |
| Shared tokens package | Extract tokens into a new @tesserix/tokens package that both web and native import | |
| You decide | Claude picks based on existing design system structure | |

**User's choice:** Import from @tesserix/native

---

| Option | Description | Selected |
|--------|-------------|----------|
| NativeWind (Tailwind for RN) | Same Tailwind class names as web — team already knows Tailwind v4 | ✓ |
| StyleSheet + tokens | Standard React Native StyleSheet with design tokens imported | |
| You decide | Claude picks the best approach | |

**User's choice:** NativeWind (Tailwind for RN)

---

## EAS Build Config

| Option | Description | Selected |
|--------|-------------|----------|
| com.tesserix.homechef.* | com.tesserix.homechef.customer, .vendor, .driver | |
| com.homechef.* | com.homechef.customer, .vendor, .driver | ✓ |
| You decide | Claude picks a standard convention | |

**User's choice:** com.homechef.*

---

| Option | Description | Selected |
|--------|-------------|----------|
| EAS managed builds | Use eas.json per app with projectRoot — builds on Expo's cloud | |
| Local builds | Use eas build --local for full control | |
| You decide | Claude picks based on project constraints | ✓ |

**User's choice:** You decide
**Notes:** Claude has discretion on build approach. Research flagged driver app needs dev client for background location.

---

## Claude's Discretion

- App directory naming convention
- Social login SDK approach (native vs web-based OAuth)
- Biometric auth UX pattern (when to trigger)
- EAS build strategy (managed vs local)

## Deferred Ideas

None — discussion stayed within phase scope
