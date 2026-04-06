# Plan 04-03: Push Notifications - Summary

**Status:** Complete
**Commits:**
- `37061a2` feat(04-03): push notification wiring in all three mobile app layouts
- `f2803d4` feat(04-03): NATS-to-push bridge — vendor actionable push, driver new-delivery push, customer order-status push
- `d0bd69c` docs(04-03): complete push notification wiring plan summary

## What was built

**Mobile (all 3 apps):**
- `setNotificationHandler` at module level in each _layout.tsx
- Android notification channels created pre-notification (customer: order-updates/promotions; vendor: new-orders MAX, order-updates; driver: new-deliveries MAX, delivery-updates)
- FCM token registered post-auth via usePushToken
- Badge count increments on notification received (PUSH-05, verified for all 3 apps)
- Deep link routing on tap — notification data includes screen path

**Vendor (actionable notifications):**
- iOS notification category `new_order` with `ACCEPT_ORDER` and `REJECT_ORDER` actions, `opensAppToForeground: false`
- Android action buttons on new-orders channel
- Lock-screen Accept/Reject uses `SecureStore.getItemAsync` + `fetch` (NOT axios — no React context in background)

**Backend (NATS-to-push bridge):**
- `apps/api/handlers/notifications.go` — `RegisterPushConsumers()` with 3 NATS subscribers:
  - `SubjectChefNewOrder` → `SendActionablePush` to vendor FCM token (lock-screen actions)
  - `SubjectDeliveryAssigned` → `SendPushNotification` to driver FCM token
  - `SubjectOrderUpdated` → `SendPushNotification` to customer FCM token
- Wired into `main.go` startup
- All FCM calls in goroutines (threat T-04-15 mitigation)
- `go build ./...` passes
