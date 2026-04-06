---
phase: 04-gps-push-polish
plan: "03"
subsystem: mobile-push + go-api
tags: [push-notifications, expo-notifications, fcm, nats, vendor, driver, customer, android-channels, ios-categories, lock-screen-actions]
dependency_graph:
  requires:
    - actionable-push-structs       # from 04-01 (push.go SendActionablePush)
    - nats-location-publish         # from 04-01 (nats.go subjects)
  provides:
    - mobile-push-wiring-customer
    - mobile-push-wiring-vendor
    - mobile-push-wiring-driver
    - nats-to-push-bridge
  affects:
    - apps/mobile-customer/app/_layout.tsx
    - apps/mobile-vendor/app/_layout.tsx
    - apps/mobile-delivery/app/_layout.tsx
    - apps/api/handlers/notifications.go
    - apps/api/main.go
tech_stack:
  added: []
  patterns:
    - module-level Notifications.setNotificationHandler (before first notification)
    - iOS notification categories registered unconditionally at mount (before auth gate)
    - Android channels created before first notification inside post-auth useEffect
    - SecureStore + fetch for lock-screen Accept/Reject (no React context in background)
    - useRef cleanup pattern for push subscription teardown on unmount
    - NATS QueueSubscribe with "push-workers" queue group alongside existing "notification-workers"
    - goroutine-wrapped push calls to prevent FCM failures from crashing NATS consumers
key_files:
  created: []
  modified:
    - apps/mobile-customer/app/_layout.tsx
    - apps/mobile-vendor/app/_layout.tsx
    - apps/mobile-delivery/app/_layout.tsx
    - apps/api/handlers/notifications.go
    - apps/api/main.go
decisions:
  - "RegisterPushConsumers placed in handlers/notifications.go (not services/) so plan grep criteria are met and concerns are separated — HTTP handlers + push consumers share the notification domain"
  - "Used 'push-workers' queue group for new NATS subscribers so both groups receive all messages; no duplicate processing within each group alongside existing 'notification-workers'"
  - "Vendor Accept/Reject uses SecureStore.getItemAsync('access_token') + fetch — not the axios instance — because notification response listener fires in backgrounded/killed app state where React context is unavailable"
  - "iOS notification categories registered outside auth check (unconditional useEffect) so Accept/Reject buttons appear even on the first-ever notification before user has logged in this session"
  - "delivery app _layout.tsx retains the background-location import added by parallel agent 04-02 — merged cleanly without conflict"
metrics:
  duration: ~30min
  completed: 2026-04-06
  tasks_completed: 2
  files_changed: 5
  files_created: 0
---

# Phase 4 Plan 3: Push Notification Wiring Summary

**One-liner:** FCM push wired end-to-end — all three apps register Android channels and iOS categories at launch, token registered post-auth via usePushToken, vendor lock-screen Accept/Reject uses SecureStore+fetch, and backend NATS consumers fire SendActionablePush/SendPushNotification on order/delivery events.

## What Was Built

### Task 1: Push notification setup in all three _layout.tsx files

**apps/mobile-customer/app/_layout.tsx**
- `Notifications.setNotificationHandler` added at module level (outside component)
- Android channels created post-auth: `order-updates` (HIGH), `promotions` (DEFAULT)
- `getRawFCMToken` + `registerDeviceToken(api, token)` called post-auth
- `addPushTokenListener` handles FCM token rotation
- `addNotificationReceivedListener` increments badge count on foreground receipt
- `addNotificationResponseReceivedListener` routes `order_update` type to `/order/:orderId`
- Cleanup via `useRef` — all subscriptions removed on unmount

**apps/mobile-vendor/app/_layout.tsx**
- `Notifications.setNotificationHandler` added at module level
- iOS category `new_order` registered unconditionally at mount (before auth), with `ACCEPT_ORDER` and `REJECT_ORDER` actions (`opensAppToForeground: false`)
- Android channels post-auth: `new-orders` (MAX), `order-updates` (HIGH)
- `getRawFCMToken` + `registerDeviceToken` post-auth
- Token rotation handled via `addPushTokenListener`
- Badge count incremented on foreground receipt
- Lock-screen Accept/Reject handler: reads `access_token` from `SecureStore`, calls `fetch` directly to `PUT /chef/orders/:orderId/status` — no axios, no React context dependency
- Retains existing `background-location` import from parallel agent 04-02

**apps/mobile-delivery/app/_layout.tsx**
- `Notifications.setNotificationHandler` at module level
- Android channels post-auth: `new-deliveries` (MAX), `delivery-updates` (HIGH)
- `getRawFCMToken` + `registerDeviceToken` post-auth
- Token rotation handled
- Badge count incremented on foreground receipt
- `addNotificationResponseReceivedListener` routes `new_delivery` type to `/available` tab

### Task 2: Backend NATS-to-push bridge

**apps/api/handlers/notifications.go — RegisterPushConsumers()**
- `SubjectChefNewOrder` subscriber ("push-workers" queue group): resolves `ChefProfile.UserID` from `event.ChefID`, calls `services.SendActionablePush(userID, ..., "new-orders", "new_order", ...)` — actionable push with iOS category matching vendor app registration
- `SubjectDeliveryAssigned` subscriber: resolves `DeliveryPartner.UserID` from `driver_id` in event data, calls `services.SendPushNotification(userID, "New Delivery Available", ..., {"type": "new_delivery", "deliveryId": ...})`
- `SubjectOrderUpdated` subscriber: calls `services.SendPushNotification(event.CustomerID, "Order Update", ..., {"type": "order_update", "orderId": ..., "status": ...})`
- All FCM calls wrapped in goroutines — consumer loop never blocks on push failure (T-04-15)
- `humanReadableOrderStatus` helper maps status strings to customer-friendly labels

**apps/api/main.go**
- Added `handlers` import
- `handlers.RegisterPushConsumers()` called after `notificationService.Start()` inside the NATS-available block

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue] delivery app _layout.tsx modified by parallel agent 04-02**
- **Found during:** Task 1, writing delivery layout
- **Issue:** The file had been updated by the parallel 04-02 agent to add `import '../lib/background-location'` — a write-conflict that caused the runtime to reject the edit
- **Fix:** Re-read the updated file and merged push notification additions with the background-location import intact
- **Files modified:** `apps/mobile-delivery/app/_layout.tsx`
- **Commit:** 37061a2

**2. [Rule 3 - Blocking issue] services.NATSMsg does not exist**
- **Found during:** Task 2, first notifications.go write
- **Issue:** Used `*services.NATSMsg` as the callback type — this type does not exist; the correct type is `*nats.Msg` from `github.com/nats-io/nats.go`
- **Fix:** Added `natsclient "github.com/nats-io/nats.go"` import and used `*natsclient.Msg` in all three subscribe callbacks
- **Files modified:** `apps/api/handlers/notifications.go`
- **Commit:** f2803d4

## Known Stubs

None — all push calls reference live DB data (FCM tokens from User.fcm_token via SendActionablePush/SendPushNotification, which query the DB). No placeholder values flow to any UI.

## Threat Flags

No new network endpoints, auth paths, or schema changes introduced beyond what the plan's threat model covers.

| Threat | File | Status |
|--------|------|--------|
| T-04-13 Spoofing — vendor lock-screen Accept/Reject | apps/mobile-vendor/app/_layout.tsx | SecureStore+fetch with Bearer JWT; backend ChefHandler.UpdateOrderStatus verifies chef_id = current user |
| T-04-14 Info disclosure — push content | apps/api/handlers/notifications.go | Notification body contains no PII; only order status and UUID |
| T-04-15 DoS — NATS push consumer | apps/api/handlers/notifications.go | All FCM calls in goroutines; failures logged, consumer loop continues |
| T-04-17 EoP — opensAppToForeground: false | apps/mobile-vendor/app/_layout.tsx | Action handled in background; orderId verified server-side |

## Self-Check

Verified files modified:
- `apps/mobile-customer/app/_layout.tsx` — contains `getRawFCMToken`, `setBadgeCountAsync`, `setNotificationHandler`, `order-updates` channel
- `apps/mobile-vendor/app/_layout.tsx` — contains `getRawFCMToken`, `setNotificationCategoryAsync`, `ACCEPT_ORDER`, `new-orders` channel, `SecureStore.getItemAsync`
- `apps/mobile-delivery/app/_layout.tsx` — contains `getRawFCMToken`, `setBadgeCountAsync`, `new-deliveries` channel, background-location import preserved
- `apps/api/handlers/notifications.go` — contains `SubjectChefNewOrder` QueueSubscribe, `SendActionablePush`, `SubjectDeliveryAssigned` QueueSubscribe, `SendPushNotification`, `"new_order"` iOS category string
- `apps/api/main.go` — contains `handlers.RegisterPushConsumers()`

Verified commits exist:
- `37061a2` — Task 1: push wiring in all three layouts
- `f2803d4` — Task 2: NATS-to-push bridge

Build: `cd apps/api && go build ./...` exits 0

## Self-Check: PASSED
