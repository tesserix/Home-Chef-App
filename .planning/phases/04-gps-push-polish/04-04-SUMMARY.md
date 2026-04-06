---
phase: 04-gps-push-polish
plan: "04"
subsystem: mobile-ux
tags: [ux, haptics, offline, skeleton, pull-to-refresh, react-native, expo]
dependency_graph:
  requires: [04-02, 04-03]
  provides: [UX-01, UX-02, UX-03, UX-04]
  affects: [mobile-customer, mobile-vendor, mobile-delivery, mobile-shared]
tech_stack:
  added:
    - "@react-native-community/netinfo ~11.4.1 — network state detection in all 3 apps and mobile-shared"
  patterns:
    - "RefreshControl on all FlatList/ScrollView list views (brand orange tintColor #f97316/#FF6B35)"
    - "Inline skeleton placeholder Views as StyleSheet rows (avoids JSX key type issues in missing-@types/react env)"
    - "OfflineBanner as shared component exported from packages/mobile-shared — NetInfo addEventListener pattern"
    - "Haptics fire-and-forget void calls in success callbacks only (not error paths, not awaited)"
key_files:
  created:
    - packages/mobile-shared/src/components/OfflineBanner.tsx
  modified:
    - apps/mobile-customer/package.json
    - apps/mobile-vendor/package.json
    - apps/mobile-delivery/package.json
    - packages/mobile-shared/package.json
    - packages/mobile-shared/src/index.ts
    - apps/mobile-customer/app/_layout.tsx
    - apps/mobile-vendor/app/_layout.tsx
    - apps/mobile-delivery/app/_layout.tsx
    - apps/mobile-vendor/app/(tabs)/orders.tsx
    - apps/mobile-customer/app/(tabs)/orders.tsx
    - apps/mobile-delivery/app/(tabs)/index.tsx
    - apps/mobile-delivery/app/(tabs)/available.tsx
    - apps/mobile-delivery/app/(tabs)/active.tsx
    - apps/mobile-vendor/hooks/useVendorOrders.ts
decisions:
  - "OfflineBanner placed in mobile-shared (not per-app) — single source of truth for offline UX across all 3 apps"
  - "Haptics for vendor accept/reject added to triggerAction in useVendorOrders hook (fires on user intent, before undo delay expires and API call fires)"
  - "Driver available.tsx haptic fires in acceptMutation onSuccess (not in handleAccept) so it only vibrates on server-confirmed accept"
  - "Skeleton in customer orders and driver screens uses StyleSheet named style skeletonRow to avoid JSX key-on-View TS2769 error caused by missing @types/react in this environment"
metrics:
  duration_seconds: 340
  completed_date: "2026-04-05"
  tasks_completed: 2
  tasks_total: 2
  files_created: 1
  files_modified: 13
---

# Phase 04 Plan 04: UX Polish — Pull-to-Refresh, Skeleton Screens, Offline Banner, Haptics Summary

**One-liner:** NetInfo-based OfflineBanner in shared package, skeleton placeholder screens on all loading states, RefreshControl on all 7 list views with brand orange, and expo-haptics fire-and-forget calls on every key user action across all three apps.

## Tasks Completed

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | Skeleton screens + pull-to-refresh + OfflineBanner | 202a852 | OfflineBanner.tsx, all 3 _layout.tsx, orders/available/index screens |
| 2 | Haptic feedback on all key actions | 01dd81e | active.tsx, available.tsx, useVendorOrders.ts |

## What Was Built

### Task 1: Skeleton screens, pull-to-refresh, OfflineBanner

**Audit findings (pre-existing — not added):**
- Customer home (`index.tsx`): RefreshControl + inline SkeletonCard already present
- Vendor dashboard: RefreshControl + SkeletonBox already present
- Vendor menu: RefreshControl + SkeletonCard already present
- Driver dashboard: RefreshControl already present
- Driver available: RefreshControl already present

**Added in this plan:**

1. `@react-native-community/netinfo ~11.4.1` added to all 3 app `package.json` files and `packages/mobile-shared/package.json` peer deps.

2. `packages/mobile-shared/src/components/OfflineBanner.tsx` — NetInfo-based non-blocking banner. Subscribes to OS network events via `NetInfo.addEventListener`, sets `isOffline` state, renders orange banner when offline, returns `null` when online (zero layout impact).

3. `packages/mobile-shared/src/index.ts` — Added `export { OfflineBanner } from './components/OfflineBanner'`.

4. All 3 app `_layout.tsx` files — Wrapped `Stack` with `<View style={{ flex: 1 }}>` containing `<OfflineBanner />` above the Stack. Customer layout wraps directly in `RootLayout`; vendor and delivery layouts wrap inside the `AppNavigator` return (inside QueryClientProvider/GestureHandlerRootView).

5. `vendor/app/(tabs)/orders.tsx` — Added `RefreshControl` to the `LiveQueue` FlatList (the only list view missing it). Destructured `isRefetching` and `refetch` from `useVendorPendingOrders()` which already returns the full query object.

6. `customer/app/(tabs)/orders.tsx` — Replaced `ActivityIndicator` loading state with 4 skeleton `skeletonRow` Views using StyleSheet (not JSX-mapped array, to avoid TS2769 key-on-View error in env without @types/react).

7. `delivery/app/(tabs)/index.tsx` — Replaced `ActivityIndicator` with structured skeleton layout matching the dashboard grid (toggle row + 2x2 stats cards).

8. `delivery/app/(tabs)/available.tsx` — Replaced `ActivityIndicator` with 3 skeleton delivery card rows.

### Task 2: Haptic feedback

**Audit findings (pre-existing — not added):**
- `apps/mobile-customer/app/checkout.tsx` — `Haptics.notificationAsync(Success)` already present in payment polling success callback (line 108). `expo-haptics` already imported.
- `apps/mobile-vendor/hooks/useVendorOrders.ts` — `Haptics.notificationAsync(Warning)` already present for new-order detection (line 46).

**Added in this plan:**

1. `apps/mobile-vendor/hooks/useVendorOrders.ts` — Added `void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium)` at the top of `triggerAction()`. Fires on user intent (accept or reject) before the 3s undo timer starts. Fire-and-forget `void` call.

2. `apps/mobile-delivery/app/(tabs)/active.tsx` — Added `import * as Haptics from 'expo-haptics'`. Modified `SlideToConfirm onConfirm` callback to fire haptics based on `nextStatus`:
   - `delivered` → `Haptics.notificationAsync(Success)` (strongest confirmation)
   - all other transitions → `Haptics.impactAsync(Medium)`

3. `apps/mobile-delivery/app/(tabs)/available.tsx` — Added `import * as Haptics from 'expo-haptics'`. Added `void Haptics.impactAsync(Medium)` in `acceptMutation.onSuccess` callback — fires only on server-confirmed delivery accept.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Restored ActivityIndicator import in customer orders and delivery available**
- **Found during:** Task 1 — removing ActivityIndicator from imports broke the footer spinner in orders.tsx and the Go-Online button spinner in available.tsx
- **Fix:** Restored ActivityIndicator to both import lists; only replaced the full-screen loading state with skeletons
- **Files modified:** apps/mobile-customer/app/(tabs)/orders.tsx, apps/mobile-delivery/app/(tabs)/available.tsx
- **Commit:** 202a852

**2. [Rule 1 - Bug] Skeleton rows use StyleSheet instead of JSX-mapped array**
- **Found during:** Task 1 — TypeScript TS2769 error when mapping `View` components with `key` prop due to missing `@types/react` in this project environment
- **Fix:** Used 4 explicit `<View style={styles.skeletonRow} />` elements and added `skeletonRow` to StyleSheet instead of `.map()` with key prop
- **Files modified:** apps/mobile-customer/app/(tabs)/orders.tsx
- **Commit:** 202a852

**3. [Audit] Haptics in vendor orders — placed in hook, not screen**
- **Context:** Plan called for haptics in `apps/mobile-vendor/app/(tabs)/orders.tsx` but the accept/reject action is entirely managed by `triggerAction` in `useVendorOrders.ts`. The hook already had Haptics imported for new-order detection.
- **Decision:** Added haptic to `triggerAction` in the hook (correct location for side effects tied to the action) rather than in the UI component's `onAccept`/`onReject` props.
- **Files modified:** apps/mobile-vendor/hooks/useVendorOrders.ts
- **Commit:** 01dd81e

## Known Stubs

None. All features are fully wired to live data sources.

## Threat Flags

None. OfflineBanner reads OS network state (read-only), no new trust boundaries introduced.

## Self-Check: PASSED

- [x] `packages/mobile-shared/src/components/OfflineBanner.tsx` — exists
- [x] Commit `202a852` — exists (`feat(04-04): skeleton screens, pull-to-refresh on all list views, OfflineBanner in all 3 apps`)
- [x] Commit `01dd81e` — exists (`feat(04-04): haptic feedback on accept/reject, delivery status transitions, and delivery complete`)
- [x] RefreshControl in all 7 list views confirmed
- [x] OfflineBanner in all 3 app layouts confirmed
- [x] Haptics in checkout (pre-existing), vendor accept/reject (hook), driver accept (available), driver transitions (active) confirmed
- [x] No new TypeScript errors introduced in modified files
