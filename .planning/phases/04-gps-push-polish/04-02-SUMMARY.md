---
phase: 04-gps-push-polish
plan: "02"
subsystem: mobile-delivery + mobile-customer
tags: [gps, background-location, websocket, expo-location, expo-task-manager, react-native]
dependency_graph:
  requires:
    - websocket-tracking-endpoint  # from 04-01: /api/v1/orders/:id/track/ws
    - delivery-response-lat-lng    # from 04-01: currentLatitude/currentLongitude in DeliveryResponse
  provides:
    - driver-background-gps-tracking
    - customer-websocket-map
  affects:
    - apps/mobile-delivery/package.json
    - apps/mobile-delivery/app.json
    - apps/mobile-delivery/app/_layout.tsx
    - apps/mobile-delivery/app/(tabs)/active.tsx
    - apps/mobile-delivery/app/(tabs)/available.tsx
    - apps/mobile-delivery/store/delivery-store.ts
    - apps/mobile-delivery/lib/background-location.ts
    - apps/mobile-delivery/components/LocationRationaleModal.tsx
    - apps/mobile-customer/hooks/useOrderTrackingWS.ts
    - apps/mobile-customer/app/order/[id]/track.tsx
tech_stack:
  added:
    - expo-location ~55.0.0 (background GPS updates via Expo SDK 55)
    - expo-task-manager ~55.0.0 (background task registration)
  patterns:
    - TaskManager.defineTask at module level (imported in _layout.tsx before React tree mounts)
    - SecureStore.getItemAsync('access_token') in background task (no React context available)
    - Rationale modal shown before requestBackgroundPermissionsAsync (App Store D-03 compliance)
    - WebSocket with 3-failure cap + automatic polling fallback (T-04-10 mitigation)
    - Parallel REST polling for order metadata + WS override for driver coordinates
key_files:
  created:
    - apps/mobile-delivery/lib/background-location.ts
    - apps/mobile-delivery/components/LocationRationaleModal.tsx
    - apps/mobile-delivery/store/delivery-store.ts
    - apps/mobile-customer/hooks/useOrderTrackingWS.ts
  modified:
    - apps/mobile-delivery/package.json
    - apps/mobile-delivery/app.json
    - apps/mobile-delivery/app/_layout.tsx
    - apps/mobile-delivery/app/(tabs)/active.tsx
    - apps/mobile-delivery/app/(tabs)/available.tsx
    - apps/mobile-customer/app/order/[id]/track.tsx
decisions:
  - "GPS start wired into available.tsx (accept action), not active.tsx — accept mutation lives there; active.tsx handles stop-on-terminal-status only"
  - "expo-location/task-manager versions pinned to ~55.0.0 (SDK 55 compatible) — plan specified ~0.30.0/~0.20.0 which are SDK 50/51 versions"
  - "track.tsx retains useOrderTracking for order metadata (status, chef, timeline); WS hook provides driver coordinate override — avoids loading spinner in primary WS path"
  - "useDeliveryStore.getState() used for imperative store updates inside async callbacks and background task — avoids stale closure issues"
metrics:
  duration: ~25min
  completed: 2026-04-06
  tasks_completed: 2
  files_changed: 10
  files_created: 4
---

# Phase 4 Plan 2: Driver GPS + Customer WebSocket Tracking Summary

**One-liner:** Driver app background GPS tracking via expo-location/task-manager with SecureStore JWT auth and App Store-compliant rationale modal; customer tracking map upgraded to WebSocket with 3-failure polling fallback.

## What Was Built

### Task 1: Driver app native setup

**apps/mobile-delivery/package.json**
- Added `expo-location ~55.0.0` and `expo-task-manager ~55.0.0`

**apps/mobile-delivery/app.json**
- `ios.infoPlist`: added `NSLocationWhenInUseUsageDescription`, `NSLocationAlwaysAndWhenInUseUsageDescription`, `UIBackgroundModes: ["location", "fetch"]`
- `android.permissions`: added `ACCESS_FINE_LOCATION`, `ACCESS_COARSE_LOCATION`, `ACCESS_BACKGROUND_LOCATION`, `FOREGROUND_SERVICE`, `FOREGROUND_SERVICE_LOCATION`
- `plugins`: added expo-location plugin with `isAndroidBackgroundLocationEnabled: true`, `isAndroidForegroundServiceEnabled: true`

**apps/mobile-delivery/lib/background-location.ts** (new)
- Exports `LOCATION_TASK_NAME = 'homechef-delivery-tracking'`
- `TaskManager.defineTask` at module level — reads JWT via `SecureStore.getItemAsync('access_token')`, calls `PUT /api/v1/delivery/location` with fetch (no React context)
- `startTracking()`: starts with 15s interval, 30m distance filter, Android foreground service notification
- `stopTracking()`: guards with `hasStartedLocationUpdatesAsync` before stopping

**apps/mobile-delivery/components/LocationRationaleModal.tsx** (new)
- Modal with explanation text shown before `requestBackgroundPermissionsAsync`
- `onAllow` / `onDeny` callbacks; `onDeny` lets delivery proceed without background tracking

**apps/mobile-delivery/store/delivery-store.ts** (new)
- Zustand store: `isTrackingLocation: boolean`, `activeDeliveryId: string | null`, `setTrackingLocation(isTracking, deliveryId)` action

**apps/mobile-delivery/app/_layout.tsx**
- Added `import '../lib/background-location'` at top — ensures `defineTask` runs at app startup

### Task 2: GPS start/stop wiring + customer WebSocket hook

**apps/mobile-delivery/app/(tabs)/available.tsx**
- `requestGPSAndStartTracking(deliveryId)`: checks foreground/background permission status, shows `LocationRationaleModal` before `requestBackgroundPermissionsAsync`
- `handleRationaleAllow` / `handleRationaleDeny` handlers
- `handleAccept` calls `acceptMutation.mutate` then calls GPS start in `onSuccess` callback
- `LocationRationaleModal` rendered in JSX return

**apps/mobile-delivery/app/(tabs)/active.tsx**
- Added `useEffect` watching `currentDelivery?.status` — calls `stopTracking()` and clears store when status is `'delivered'` or `'cancelled'`

**apps/mobile-customer/hooks/useOrderTrackingWS.ts** (new)
- `MAX_WS_FAILURES = 3` constant
- Builds `wss://` URL from `EXPO_PUBLIC_API_URL` env var
- `failureCount` ref incremented on `onerror`; at 3 failures sets `useFallback = true` (polling via `useOrderTracking`)
- Reconnects on `onclose` only when `failureCount < MAX_WS_FAILURES` — no infinite loop
- Returns `{ driverLocation, isPollingFallback, pollingResult }`

**apps/mobile-customer/app/order/[id]/track.tsx**
- Imports both `useOrderTrackingWS` (driver coordinates) and `useOrderTracking` (order metadata)
- `effectiveDriverLat/Lng`: uses WS `driverLocation` when available and not in fallback; falls back to `tracking.delivery?.currentLatitude`
- `DeliveryMap` receives the effective driver coords

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] expo-location/task-manager version corrected to SDK 55**
- **Found during:** Task 1, Step 1
- **Issue:** Plan specified `expo-location@~0.30.0` and `expo-task-manager@~0.20.0` (SDK 50/51 versions); app uses Expo SDK 55 which requires `~55.0.0`
- **Fix:** Installed `expo-location@~55.0.0` and `expo-task-manager@~55.0.0`
- **Commits:** 2bb3385

**2. [Rule 1 - Bug] GPS start wired in available.tsx instead of active.tsx**
- **Found during:** Task 2, Step 1
- **Issue:** Plan listed `active.tsx` as the accept location but `active.tsx` has no accept action — `useAcceptDelivery` mutation and delivery card rendering live in `available.tsx`
- **Fix:** Placed `requestGPSAndStartTracking` and `LocationRationaleModal` in `available.tsx`; kept `stopTracking` useEffect in `active.tsx` as planned
- **Commits:** 40fe833

**3. [Rule 1 - Bug] track.tsx retains useOrderTracking for order metadata**
- **Found during:** Task 2, Step 3
- **Issue:** Original plan replaced `useOrderTracking` entirely with `useOrderTrackingWS`, but `useOrderTrackingWS` only provides driver coordinates — the bottom sheet needs order number, chef name, and status from the REST endpoint
- **Fix:** track.tsx uses both: `useOrderTrackingWS` for real-time driver position + `useOrderTracking` for order metadata; WS coordinates override polling coordinates when available
- **Commits:** 40fe833

## Known Stubs

None — all driver location fields are wired to live data sources (WS or REST polling fallback).

## Threat Flags

No new threat surface introduced beyond the plan's threat register.

| Threat | File | Status |
|--------|------|--------|
| T-04-08 Privacy: tracking start gate | available.tsx | Tracking starts only on delivery accept `onSuccess`, never on app launch |
| T-04-09 Spoofing: JWT in background task | lib/background-location.ts | `SecureStore.getItemAsync('access_token')` — no React context, no AsyncStorage |
| T-04-10 DoS: WS reconnect loop | hooks/useOrderTrackingWS.ts | Reconnect blocked when `failureCount >= MAX_WS_FAILURES`; fallback to polling |

## Self-Check

Verified files exist:
- `apps/mobile-delivery/lib/background-location.ts` — contains `LOCATION_TASK_NAME`, `SecureStore`, `startTracking`, `stopTracking`
- `apps/mobile-delivery/components/LocationRationaleModal.tsx` — contains `LocationRationaleModal`
- `apps/mobile-delivery/store/delivery-store.ts` — contains `isTrackingLocation`, `setTrackingLocation`
- `apps/mobile-customer/hooks/useOrderTrackingWS.ts` — contains `useOrderTrackingWS`, `MAX_WS_FAILURES = 3`

Verified commits exist:
- `2bb3385` — Task 1: native GPS setup
- `40fe833` — Task 2: GPS wiring + WebSocket hook

TypeScript: zero GPS/WS-specific type errors in new files; only pre-existing project-wide TS7016 (`@types/react` missing) affects all files including new ones.

## Self-Check: PASSED
