---
phase: 03-vendor-app-driver-core
plan: "04"
subsystem: apps/mobile-delivery
tags: [driver, onboarding, wizard, tabs, expo, zustand, react-hook-form, zod]
dependency_graph:
  requires:
    - 03-01 (vendor onboarding pattern — architecture mirror)
    - Phase 01 foundation (auth store, API client, mobile-shared hooks)
  provides:
    - Driver onboarding wizard (6 steps, API-backed)
    - Driver pending/rejected holding screen with 30s polling
    - 4-tab driver shell (Dashboard / Available / Active / More)
    - BottomSheetModalProvider at root (required by Plan 03-05 status updates)
  affects:
    - apps/mobile-delivery/app/_layout.tsx (onboarding gate added)
    - apps/mobile-delivery/app/(tabs)/ (full tab shell)
tech_stack:
  added:
    - "@gorhom/bottom-sheet ^5.2.8"
    - "expo-haptics ~55.0.11"
    - "expo-image ~55.0.8"
    - "lucide-react-native ^0.475.0"
    - "expo-document-picker ~13.0.3"
    - "expo-image-picker ~16.0.6"
  patterns:
    - Zustand store for in-memory wizard state (never persisted to device storage — security requirement)
    - React Hook Form + Zod for all form screens
    - useQuery with AxiosResponse<T> generic for typed API responses
    - 30s foreground polling on pending screen via refetchInterval
    - discriminatedUnion replaced with superRefine for payout schema (Zod type inference compatibility)
key_files:
  created:
    - apps/mobile-delivery/store/onboarding-store.ts
    - apps/mobile-delivery/nativewind-env.d.ts
    - apps/mobile-delivery/app/(onboarding)/_layout.tsx
    - apps/mobile-delivery/app/(onboarding)/personal.tsx
    - apps/mobile-delivery/app/(onboarding)/vehicle.tsx
    - apps/mobile-delivery/app/(onboarding)/documents.tsx
    - apps/mobile-delivery/app/(onboarding)/payout.tsx
    - apps/mobile-delivery/app/(onboarding)/subscription.tsx
    - apps/mobile-delivery/app/(onboarding)/review.tsx
    - apps/mobile-delivery/app/(onboarding)/pending.tsx
    - apps/mobile-delivery/app/(tabs)/available.tsx
    - apps/mobile-delivery/app/(tabs)/active.tsx
    - apps/mobile-delivery/app/(tabs)/more.tsx
  modified:
    - apps/mobile-delivery/package.json (added 6 new deps + expo-image-picker)
    - apps/mobile-delivery/tsconfig.json (added nativewind-env.d.ts to includes)
    - apps/mobile-delivery/app/_layout.tsx (onboarding gate + BottomSheetModalProvider)
    - apps/mobile-delivery/app/(tabs)/_layout.tsx (4-tab driver layout with lucide icons)
    - apps/mobile-delivery/app/(tabs)/index.tsx (NativeWind styled placeholder)
decisions:
  - "Used superRefine instead of discriminatedUnion for payout Zod schema — discriminatedUnion type inference is incompatible with react-hook-form generic constraint in strict mode"
  - "Added expo-image-picker explicitly to package.json — Expo managed workflow includes it but TypeScript requires explicit declaration for type resolution"
  - "useQuery generic typed as AxiosResponse<T> not T — api client returns AxiosResponse, not unwrapped data"
  - "AppNavigator sub-component pattern for root layout — useQuery hooks cannot be called outside QueryClientProvider, so the gate logic lives inside a child component"
metrics:
  duration: "~45 minutes"
  completed: "2026-04-06"
  tasks_completed: 2
  files_created: 13
  files_modified: 5
---

# Phase 03 Plan 04: Driver Onboarding Wizard and Tab Shell Summary

6-step driver onboarding wizard with correct document endpoint, subscription plan selection, 30s-polling pending screen, and 4-tab shell with BottomSheetModalProvider at root.

## What Was Built

### Onboarding Wizard Steps

| Step | File | API Endpoint | Key Fields |
|------|------|-------------|-----------|
| 1 — Personal | personal.tsx | POST /driver/onboarding/personal | city, vehicleType (radio), emergency contact, DOB |
| 2 — Vehicle | vehicle.tsx | POST /driver/onboarding/vehicle | make, model, year, color, reg number, license |
| 3 — Documents | documents.tsx | POST /driver/onboarding/documents | driving_license, id_proof (required), vehicle_rc (optional) |
| 4 — Payout | payout.tsx | POST /driver/onboarding/payout | bank (account+IFSC+name) or UPI |
| 5 — Subscription | subscription.tsx | GET /driver/subscription/plans + POST /driver/subscription/plan | plan cards with features, price |
| 6 — Review | review.tsx | POST /driver/onboarding/submit | read-only summary, terms checkbox |

### Document Upload Endpoint — Confirmed Correct

Step 3 posts to `/driver/onboarding/documents` (auth-only, no delivery role needed).
Post-onboarding document upload uses `/delivery/documents` (delivery role required) — different gates.

### Zustand Store

`useDriverOnboardingStore` exports:
- `updatePersonalInfo`, `updateVehicleDetails`, `updateDocuments`, `updatePayoutDetails`, `updateSubscriptionInfo`
- `setStep`, `reset`

Payout details (`bankAccountNumber`, `bankIFSC`) stored in memory only — never persisted to AsyncStorage or SecureStore. Cleared on `reset()` after successful submit. Satisfies threat model T-03-04-02 and T-03-04-05.

### Pending/Rejected Holding Screen

- Polls `/driver/onboarding/status` every 30 seconds (foreground only — `refetchIntervalInBackground: false`)
- Auto-routes to `/(tabs)` when `onboardingComplete === true` or `verificationStatus === 'approved'`
- Rejected state shows rejection reason from `profile.rejectionReason` and a "Reapply" button → `/(onboarding)/personal`
- Logout button top-right on both states

### Root Layout Onboarding Gate

Gate logic reads `/driver/onboarding/status` via `useQuery` (enabled only when authenticated). Routes:
- `not_started` → `/(onboarding)/personal`
- `in_progress` / `pending_review` / `rejected` → `/(onboarding)/pending`
- `onboardingComplete: true` → `/(tabs)`
- Not authenticated → `/(auth)/login`

### 4-Tab Shell

| Tab | File | Icon | Route |
|-----|------|------|-------|
| Dashboard | index.tsx | LayoutDashboard | / (tabs) |
| Available | available.tsx | MapPin | /available |
| Active | active.tsx | Navigation | /active |
| More | more.tsx | MoreHorizontal | /more |

More tab items: Profile, Earnings, History, Fleet, Staff, Settings (stubs → Plan 03-06), plus Logout with confirmation alert.

### BottomSheetModalProvider

Wrapped at root layout level (inside `GestureHandlerRootView` and `QueryClientProvider`) — required for status update bottom sheet in Plan 03-05.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical Functionality] Added nativewind-env.d.ts**
- **Found during:** Task 1 TypeScript check
- **Issue:** `nativewind-env.d.ts` was missing from `apps/mobile-delivery/` causing all `className` props to produce TypeScript errors (pre-existing — also absent in mobile-vendor)
- **Fix:** Created `nativewind-env.d.ts` with `/// <reference types="nativewind/types" />` and added it to tsconfig includes. Pattern from `apps/mobile-customer/` which had the file correctly.
- **Files modified:** `apps/mobile-delivery/nativewind-env.d.ts`, `apps/mobile-delivery/tsconfig.json`
- **Commit:** 3267b31

**2. [Rule 2 - Missing Critical Functionality] Added expo-image-picker to package.json**
- **Found during:** Task 1 TypeScript check
- **Issue:** Research noted expo-image-picker is "part of Expo SDK" but TypeScript requires explicit package declaration for type resolution. Package was not in delivery app's package.json.
- **Fix:** Added `"expo-image-picker": "~16.0.6"` to dependencies; ran `pnpm install`.
- **Files modified:** `apps/mobile-delivery/package.json`
- **Commit:** 3267b31

**3. [Rule 1 - Bug] Replaced discriminatedUnion with superRefine in payout schema**
- **Found during:** Task 1 TypeScript check
- **Issue:** `z.discriminatedUnion` produces a type that is incompatible with react-hook-form's `UseFormReturn` generic constraint in strict mode — TS2740 error on the resolver argument.
- **Fix:** Replaced the two `z.discriminatedUnion` branches with a single `z.object().superRefine()` containing conditional validation logic.
- **Files modified:** `apps/mobile-delivery/app/(onboarding)/payout.tsx`
- **Commit:** 3267b31

**4. [Rule 2 - Missing Critical Functionality] AppNavigator sub-component for root layout**
- **Found during:** Task 2 implementation
- **Issue:** `useQuery` cannot be called before `QueryClientProvider` is mounted — plan spec showed the query directly in `RootLayout` but that component wraps `QueryClientProvider`.
- **Fix:** Extracted `AppNavigator` as a child component rendered inside `QueryClientProvider`, holding all hooks and navigation logic.
- **Files modified:** `apps/mobile-delivery/app/_layout.tsx`
- **Commit:** 746d4eb

### Pre-existing TypeScript Issues (Out of Scope)

The following errors exist in the delivery app's TypeScript check but pre-date this plan and are present in other apps too:
- `TS7016: Could not find a declaration file for module 'react'` — affects all app files; same error in `mobile-customer` (15 instances). Root cause: `@types/react` version mismatch in pnpm virtual store. Logged to deferred items.
- `../../packages/mobile-shared/src/` JSX component errors — pre-existing in the shared package, not caused by any changes in this plan.

## Known Stubs

| Stub | File | Reason |
|------|------|--------|
| Dashboard screen | app/(tabs)/index.tsx | Full dashboard (DRIV-01) built in Plan 03-05 |
| Available screen | app/(tabs)/available.tsx | Available deliveries list (DRIV-02) built in Plan 03-05 |
| Active screen | app/(tabs)/active.tsx | Active delivery (DRIV-03/04/05) built in Plan 03-05 |
| More tab nav targets | app/(tabs)/more.tsx | Profile/Earnings/History/Fleet/Staff/Settings screens built in Plan 03-06 |

All stubs display "coming soon" text and do not block plan's primary goal (onboarding wizard + tab shell).

## Threat Flags

No new security surface introduced beyond the plan's threat model. Payout details confirmed memory-only (not persisted). Document upload endpoint confirmed as `/driver/onboarding/documents` (auth-only gate).

## Self-Check: PASSED

All key files verified present. Both task commits confirmed in git history (3267b31, 746d4eb).
