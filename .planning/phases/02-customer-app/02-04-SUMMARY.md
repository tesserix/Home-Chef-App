---
phase: 02-customer-app
plan: 04
subsystem: mobile-customer
tags: [tracking, maps, react-native-maps, polling, bottom-sheet, expo-router]
dependency_graph:
  requires: [02-03]
  provides: [live-order-tracking-screen, delivery-map-component, order-timeline]
  affects: [apps/mobile-customer/app/order]
tech_stack:
  added: []
  patterns: [react-query-polling, useIsFocused-guard, bottom-sheet-overlay, directory-based-routing]
key_files:
  created:
    - apps/mobile-customer/hooks/useOrderTracking.ts
    - apps/mobile-customer/components/tracking/DeliveryMap.tsx
    - apps/mobile-customer/components/orders/OrderTimeline.tsx
    - apps/mobile-customer/app/order/[id]/track.tsx
  modified:
    - apps/mobile-customer/types/customer.ts
    - apps/mobile-customer/app.json
    - apps/mobile-customer/app/order/[id]/index.tsx (renamed from [id].tsx)
decisions:
  - key: useIsFocused from expo-router not @react-navigation/native
    why: "@react-navigation/native is not installed; expo-router re-exports useIsFocused from it as a convenience export"
  - key: PROVIDER_DEFAULT for maps (not PROVIDER_GOOGLE)
    why: No GOOGLE_MAPS_API_KEY configured; PROVIDER_DEFAULT uses Apple Maps on iOS and Google Maps on Android without key
  - key: dropoff coords backend gap documented
    why: DeliveryResponse.ToResponse() omits DropoffLatitude/Longitude; fields added to TrackingResponse type with comment — falls back to chef coords until backend is updated
  - key: Converted app/order/[id].tsx to app/order/[id]/index.tsx directory
    why: expo-router requires directory-based routing to support nested route /order/:id/track alongside /order/:id
metrics:
  duration: 224s
  completed: 2026-04-05
  tasks_completed: 2
  files_changed: 7
---

# Phase 02 Plan 04: Live Order Tracking Screen Summary

**One-liner:** Live delivery tracking with react-native-maps MapView, 5s React Query polling, auto-stop on completion, and bottom-sheet order timeline.

## What Was Built

### Task 1 — TrackingResponse type, useOrderTracking hook, DeliveryMap, iOS location permission

- **`types/customer.ts`**: Added `dropoffLatitude` and `dropoffLongitude` to `TrackingResponse.delivery`. Documented backend gap (DeliveryResponse.ToResponse() omits coordinates for now).
- **`hooks/useOrderTracking.ts`**: React Query hook with `refetchInterval` as a function — returns `5000` for active statuses, `false` for `delivered`/`cancelled`. `refetchIntervalInBackground: false` stops polling when backgrounded.
- **`components/tracking/DeliveryMap.tsx`**: Full-screen MapView with driver marker (blue) and destination marker (red). Uses `dropoffLatitude`/`dropoffLongitude` from Delivery model for destination; falls back to chef coordinates when dropoff is 0/missing. Guards driver `0,0` before rendering driver pin.
- **`app.json`**: Added `NSLocationWhenInUseUsageDescription` to `expo.ios.infoPlist` — prevents blank map on iOS real device.

### Task 2 — Tracking screen, status timeline, directory routing

- **`components/orders/OrderTimeline.tsx`**: Four-step timeline (Confirmed → Preparing → On the Way → Delivered) with step completion state (grey/orange/green), progress connectors, and estimated arrival time display.
- **`app/order/[id]/track.tsx`**: Full-screen tracking screen. DeliveryMap fills 100% of screen; `@gorhom/bottom-sheet` overlaid with `snapPoints: ['30%', '60%']` showing order number, chef name, and OrderTimeline. Uses `useIsFocused` from `expo-router` as polling guard. Auto-navigates to order detail 2s after `delivered` status.
- **`app/order/[id]/index.tsx`**: Renamed from flat `app/order/[id].tsx` to directory-based routing so expo-router can resolve both `/order/:id` and `/order/:id/track` as sibling routes.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed broken relative imports after directory move**
- **Found during:** Task 2
- **Issue:** Moving `app/order/[id].tsx` → `app/order/[id]/index.tsx` broke `../../hooks/` and `../../types/` import paths
- **Fix:** Updated paths to `../../../hooks/useOrderHistory` and `../../../types/customer`
- **Files modified:** `apps/mobile-customer/app/order/[id]/index.tsx`
- **Commit:** 24dcf0a

**2. [Rule 1 - Bug] Fixed pre-existing numeric key prop type error in index.tsx**
- **Found during:** Task 2 TypeScript verification
- **Issue:** `key={index}` (number) passed to `View` caused `TS2769` — `key` must be string in React JSX
- **Fix:** Changed to `key={String(index)}`
- **Files modified:** `apps/mobile-customer/app/order/[id]/index.tsx`
- **Commit:** 24dcf0a

### Backend Gap Noted (not a deviation — documented in types)

The Go API `TrackOrder` handler uses `order.Delivery.ToResponse()` which maps to `DeliveryResponse` — this DTO omits `dropoffLatitude`, `dropoffLongitude`, `currentLatitude`, and `currentLongitude`. The `TrackingResponse` TypeScript type includes these fields as optional, and `DeliveryMap` falls back to chef coordinates when dropoff is absent. A backend fix is needed to include coordinates in `DeliveryResponse.ToResponse()`.

## Known Stubs

None — all props are wired from the API response. Coordinate fields will be `undefined` until the backend gap is resolved, but the fallback (chef coordinates) provides a functional map experience.

## Threat Flags

None — no new network endpoints or auth paths introduced. Existing `/v1/orders/:id/track` endpoint is used as-is.

## Self-Check: PASSED

| Check | Result |
|-------|--------|
| apps/mobile-customer/hooks/useOrderTracking.ts | FOUND |
| apps/mobile-customer/components/tracking/DeliveryMap.tsx | FOUND |
| apps/mobile-customer/components/orders/OrderTimeline.tsx | FOUND |
| apps/mobile-customer/app/order/[id]/track.tsx | FOUND |
| apps/mobile-customer/app/order/[id]/index.tsx | FOUND |
| commit 76b81ca | FOUND |
| commit 24dcf0a | FOUND |
