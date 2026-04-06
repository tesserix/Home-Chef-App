---
phase: 03-vendor-app-driver-core
plan: "02"
subsystem: mobile-vendor
tags: [react-query, polling, optimistic-update, haptics, vendor-dashboard, order-queue]
dependency_graph:
  requires: [03-01]
  provides: [vendor-dashboard, vendor-orders-live-queue, vendor-order-history]
  affects: []
tech_stack:
  added: []
  patterns:
    - React Query 10s polling with refetchIntervalInBackground:false
    - Optimistic update with 3-second undo timer (setTimeout + queryClient.setQueryData)
    - expo-haptics NotificationFeedbackType.Warning on new order arrival
    - Segmented control (two Pressable pills) for Live/History tab toggle
    - Animated.spring slide-up for UndoSnackbar entrance
key_files:
  created:
    - apps/mobile-vendor/hooks/useVendorDashboard.ts
    - apps/mobile-vendor/hooks/useVendorOrders.ts
    - apps/mobile-vendor/components/vendor/UndoSnackbar.tsx
    - apps/mobile-vendor/components/vendor/DashboardStatsCard.tsx
    - apps/mobile-vendor/components/vendor/OrderCard.tsx
  modified:
    - apps/mobile-vendor/app/(tabs)/index.tsx
    - apps/mobile-vendor/app/(tabs)/orders.tsx
decisions:
  - "UndoSnackbar uses Animated.spring (react-native core) not reanimated — keeps dependency surface minimal"
  - "Order history pagination implemented with page state in HistoryList sub-component to isolate re-renders"
  - "triggerAction does local optimistic remove first, then schedules API call — distinct from onMutate pattern to allow clean undo cancellation"
metrics:
  duration: "~25 minutes"
  completed: "2026-04-05"
  tasks_completed: 2
  files_changed: 7
---

# Phase 03 Plan 02: Vendor Dashboard and Orders Screen Summary

Vendor dashboard (VEND-01) and orders screen with live queue + history (VEND-04, VEND-05) are fully implemented. A chef can view today's stats, toggle accepting-orders, see recent orders, manage a live order queue with haptic alerts, and optimistically accept/reject with a 3-second undo. Order history is paginated with pull-to-refresh.

## Hooks Exported and Signatures

### useVendorDashboard.ts
- `useVendorDashboard(): UseQueryResult<DashboardData>` — staleTime 30s, queries `GET /chef/dashboard`
- `useToggleAcceptingOrders(): UseMutationResult` — calls `PUT /chef/settings`, invalidates dashboard cache

### useVendorOrders.ts
- `useVendorPendingOrders(): UseQueryResult<OrdersResponse>` — `refetchInterval: 10_000`, `refetchIntervalInBackground: false`; triggers `Haptics.notificationAsync(Warning)` when order count increases
- `useVendorOrderHistory(page?: number): UseQueryResult<OrdersResponse>` — staleTime 30s, paginated
- `useOrderAction(): { triggerAction, handleUndo, pendingUndo, isLoading }` — optimistic remove + 3-second undo timer; API call fires after delay or is cancelled on undo

## Polling Configuration

| Property | Value |
|---|---|
| refetchInterval | 10_000 ms (10 seconds) |
| refetchIntervalInBackground | false (foreground polling only, Phase 4 adds push) |
| Haptic trigger | When `orders.length > previousCountRef.current` |
| Haptic type | `Haptics.NotificationFeedbackType.Warning` |

## Undo Mechanism

`triggerAction(orderId, action, reason?)`:
1. Immediately calls `queryClient.setQueryData` to remove card from UI (optimistic, outside `useMutation.onMutate`)
2. Sets `pendingUndo` state — causes `UndoSnackbar` to slide up
3. Starts a `setTimeout` for `UNDO_DELAY_MS = 3000` ms
4. On timeout: calls `mutation.mutate(...)`, clears `pendingUndo`

`handleUndo()`:
1. `clearTimeout` cancels the pending API call
2. `queryClient.invalidateQueries` re-fetches to restore the card
3. Clears `pendingUndo` — snackbar slides out

This approach differs from `onMutate` optimistic updates: the mutation never fires if undo is pressed, so there is no `onError` rollback needed for the undo path.

## API Response Shapes

All shapes matched the plan interfaces exactly. Types defined in `useVendorOrders.ts`:
- `Order` — id, customerName, items, total, status, createdAt, deliveryAddress, specialInstructions?
- `OrdersResponse` — orders, total, page, limit
- `DashboardData` — todayOrders, todayEarnings, rating, totalReviews, acceptingOrders, recentOrders

## Commits

| Hash | Description |
|---|---|
| ef0d3e2 | feat(03-02): useVendorDashboard, useVendorOrders hooks, and UndoSnackbar component |
| 5938513 | feat(03-02): vendor dashboard screen, orders screen, DashboardStatsCard, OrderCard |

## Deviations from Plan

None — plan executed exactly as written. Audio chime deferred to Phase 4 per plan note (D-02).

## Known Stubs

None — all data wired to real API calls via React Query.

## Threat Flags

No new threat surface introduced beyond what is documented in the plan's threat model.

## Self-Check: PASSED

Files exist:
- apps/mobile-vendor/hooks/useVendorDashboard.ts — FOUND
- apps/mobile-vendor/hooks/useVendorOrders.ts — FOUND
- apps/mobile-vendor/components/vendor/UndoSnackbar.tsx — FOUND
- apps/mobile-vendor/components/vendor/DashboardStatsCard.tsx — FOUND
- apps/mobile-vendor/components/vendor/OrderCard.tsx — FOUND
- apps/mobile-vendor/app/(tabs)/index.tsx — FOUND
- apps/mobile-vendor/app/(tabs)/orders.tsx — FOUND

Commits exist: ef0d3e2, 5938513 — FOUND
