# Phase 4: GPS, Push + Polish - Research

**Researched:** 2026-04-06
**Domain:** Background GPS (expo-location + expo-task-manager), Push Notifications (FCM HTTP v1 actionable), WebSocket in Go (gorilla/websocket + NATS pub/sub), UX Polish (React Native)
**Confidence:** HIGH (verified primarily from codebase inspection + confirmed libraries already in package.json)

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Background GPS tracking starts when driver taps Accept, stops on `delivered` or `cancelled` — NOT on going online
- **D-02:** GPS update rate at Claude's discretion — 15 seconds recommended; rate limiting MUST be added to `PUT /delivery/location` (1 req/10s per user via gin middleware)
- **D-03:** Background location MUST show rationale screen before system permission prompt (App Store requirement)
- **D-04:** `expo-location.startLocationUpdatesAsync` + `expo-task-manager` for background task
- **D-05:** Android foreground service notification required for GPS tracking ("HomeChef Delivery is tracking your location")
- **D-06:** Direct FCM via `getDevicePushTokenAsync()` — raw FCM tokens, no Expo Push Service
- **D-07:** Vendor notifications MUST be actionable with lock-screen Accept/Reject buttons
- **D-08:** Deep link routing: notification taps open specific screen (order detail / delivery detail)
- **D-09:** Notification channels (Android) per app — Customer: Order Updates (high) + Promotions (default); Vendor: New Orders (max) + Order Updates (high); Driver: New Deliveries (max) + Delivery Updates (high)
- **D-10:** FCM token registration already in `usePushToken.ts` — Phase 4 wires into login flow
- **D-11:** Vendor sound alert now handled by push notification audio, not expo-av
- **D-12:** Customer tracking map upgrades polling → WebSocket
- **D-13:** New Go endpoint: `GET /api/v1/orders/:id/track/ws`
- **D-14:** Backend serialization fix: `DeliveryResponse.ToResponse()` must include `currentLatitude`, `currentLongitude`, `dropoffLatitude`, `dropoffLongitude`
- **D-15:** NATS broadcasting: driver GPS updates → `delivery.location.{deliveryId}` subject → WebSocket handler forwards to customers
- **D-16:** Mobile WebSocket client: automatic reconnect, fallback to polling after 3 failed reconnects
- **D-17:** Rate limiting on `PUT /delivery/location`: gin middleware, max 1 req/10s per user
- **D-18:** UX polish ALL 3 apps — audit first, add where missing
- **D-19:** Offline: NetInfo + non-blocking toast + cached React Query data browsing
- **D-20:** Haptic feedback: order placed, accept/reject, delivery status transitions, payment success
- **D-21:** Skeleton screens on all loading states
- **D-22:** Pull-to-refresh on all list views

### Claude's Discretion

- GPS update rate (D-02 resolved: 15s)
- Offline error state pattern (D-19 resolved: NetInfo + toast)
- Audit-based polish implementation (D-18 resolved: audit first)

### Deferred Ideas (OUT OF SCOPE)

- Apple Pay / Google Pay native payment sheet
- Full offline mode with sync
- Scheduled order reminders (push 30 min before delivery)
- Photo confirmation on delivery
- App Store rating prompt
- Shake to report issue
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DRIV-12 | Driver app sends background GPS location updates while on active delivery | expo-location 0.30.x + expo-task-manager confirmed in package.json; background task pattern documented below |
| DRIV-13 | Background location permission requested only when driver taps Accept (not first launch) | Rationale screen pattern + `requestBackgroundPermissionsAsync` gated on delivery accept action |
| PUSH-01 | Customer receives push for order status updates | `SendPushNotification` in push.go already exists; wire into order status change events |
| PUSH-02 | Vendor receives push for new incoming orders | `SubjectChefNewOrder` NATS event exists; wire push send into NATS consumer |
| PUSH-03 | Vendor can accept/reject from notification (lock screen) | iOS category + Android action payload additions to push.go `fcmMessageBody` |
| PUSH-04 | Driver receives push for new available deliveries | `SubjectDeliveryAssigned` NATS event exists; wire push send |
| PUSH-05 | App icon badge shows unread notification count | `expo-notifications setBadgeCountAsync` in `_layout.tsx` on notification receipt |
| PUSH-06 | FCM tokens registered via `PUT /profile/device-token` (raw FCM, not Expo Push) | `getRawFCMToken` + `registerDeviceToken` already built in `packages/mobile-shared/src/hooks/usePushToken.ts`; needs wiring in all 3 `_layout.tsx` files post-auth |
| UX-01 | Pull-to-refresh on all list views | `RefreshControl` from react-native; `refetch` from React Query |
| UX-02 | Skeleton screens on key views | NativeWind-styled placeholder components |
| UX-03 | Graceful offline error state | `@react-native-community/netinfo` + toast via `sonner-native` or custom toast |
| UX-04 | Haptic feedback on key actions | `expo-haptics` v55.0.11 already in all 3 package.json files |
</phase_requirements>

---

## Summary

Phase 4 is the final phase and touches all three mobile apps plus the Go backend. The research confirms the vast majority of infrastructure is already in place: `expo-notifications` v0.30.2 and `expo-haptics` v55.0.11 are already in all three app `package.json` files; `push.go` already sends FCM HTTP v1 notifications; the NATS `DELIVERY` stream already covers `delivery.*` subjects so `delivery.location.{deliveryId}` requires no new stream; and the `usePushToken.ts` hook already implements the correct `getDevicePushTokenAsync()` pattern. What's missing is wiring and extension, not greenfield building.

The two highest-risk items are: (1) the Go WebSocket endpoint — there is no WebSocket library in the existing `go.mod` so `gorilla/websocket` must be added; and (2) background location in the driver app — `expo-location` and `expo-task-manager` are NOT in the driver app `package.json` yet and are NOT in `app.json` plugins, meaning a new EAS Build is required before any GPS feature can be tested. The App Store rationale screen (D-03) must be implemented before submitting to Apple review, and the Android foreground service notification (D-05) is required for Google Play.

The `DeliveryResponse.ToResponse()` serialization bug is confirmed: the struct omits `currentLatitude`, `currentLongitude`, `dropoffLatitude`, `dropoffLongitude` despite those fields existing on the `Delivery` model. The tracking screen already references `tracking.delivery?.currentLatitude` and `tracking.delivery?.dropoffLatitude`, confirming the Phase 2 workaround used chef coordinates as a fallback. This fix is backend-only and unblocks real driver tracking.

**Primary recommendation:** Start with Wave 0 backend changes (serialization fix, rate limiter, WebSocket endpoint, NATS publish in UpdateLocation). Then add expo-location/task-manager to driver app and rebuild EAS. Then wire push notifications. Finish with UX polish audit.

---

## Standard Stack

### Already in Project (Confirmed)

| Library | Version | Purpose | Source |
|---------|---------|---------|--------|
| expo-notifications | ~0.30.2 | Push permissions, token registration, notification handling, badge count | [VERIFIED: all 3 package.json files] |
| expo-haptics | ~55.0.11 | Haptic feedback (impact, notification, selection) | [VERIFIED: all 3 package.json files] |
| expo-location | NOT installed | Background GPS, foreground location | [VERIFIED: absent from all 3 package.json files — must add] |
| expo-task-manager | NOT installed | Background task execution engine | [VERIFIED: absent from all 3 package.json files — must add] |
| @tanstack/react-query | ^5.83.0 | Server state, `refetchInterval`, `refetch` for pull-to-refresh | [VERIFIED: all 3 package.json files] |
| react-native-maps | 1.27.2 | Customer tracking map (existing) | [VERIFIED: mobile-customer package.json] |
| zustand | ^5.0.2 | GPS tracking state store | [VERIFIED: all 3 package.json files] |
| nats.go | v1.47.0 | NATS messaging (existing) | [VERIFIED: apps/api/go.mod] |
| golang.org/x/time | v0.15.0 | Rate limiting (`rate.Limiter`) — already in go.mod indirect | [VERIFIED: apps/api/go.mod] |

### Must Add

| Library | Version | Purpose | Why |
|---------|---------|---------|-----|
| expo-location | ~0.30.x | Background GPS (`startLocationUpdatesAsync`) + foreground | Required for DRIV-12/13; native plugin, requires EAS rebuild [ASSUMED compatible with Expo 55] |
| expo-task-manager | ~0.20.x | Background task registration (`defineTask`) | Required for expo-location background mode [ASSUMED compatible with Expo 55] |
| gorilla/websocket | ^1.5.3 | WebSocket upgrade in Go API | No WebSocket library in go.mod; gorilla/websocket is the standard Go choice [ASSUMED: verify current version] |
| @react-native-community/netinfo | ~11.x | Network connectivity status for offline detection | Required for UX-03 [ASSUMED compatible with Expo 55 / React Native 0.83] |

**Installation for driver app:**
```bash
cd apps/mobile-delivery
pnpm add expo-location expo-task-manager
```

**Installation for customer/vendor (NetInfo):**
```bash
cd apps/mobile-customer && pnpm add @react-native-community/netinfo
cd apps/mobile-vendor && pnpm add @react-native-community/netinfo
```

**Go API WebSocket:**
```bash
cd apps/api
go get github.com/gorilla/websocket@latest
```

---

## Architecture Patterns

### Pattern 1: Background GPS Task (Driver)

**What:** expo-task-manager defines a named background task that expo-location calls when a location update fires, even when the app is backgrounded.

**Critical:** Background location is a NATIVE capability. Adding expo-location + expo-task-manager to `app.json` plugins and `package.json` requires a new EAS Build. It cannot be OTA-updated.

**app.json additions required (driver app only):**
```json
{
  "expo": {
    "ios": {
      "infoPlist": {
        "NSLocationWhenInUseUsageDescription": "HomeChef Delivery needs your location to show your position on the delivery map.",
        "NSLocationAlwaysAndWhenInUseUsageDescription": "HomeChef Delivery tracks your location in the background to keep customers updated on their delivery status.",
        "UIBackgroundModes": ["location", "fetch"]
      }
    },
    "android": {
      "permissions": [
        "ACCESS_FINE_LOCATION",
        "ACCESS_COARSE_LOCATION",
        "ACCESS_BACKGROUND_LOCATION",
        "FOREGROUND_SERVICE",
        "FOREGROUND_SERVICE_LOCATION"
      ]
    },
    "plugins": [
      ["expo-location", {
        "locationAlwaysAndWhenInUsePermission": "HomeChef Delivery tracks your location during deliveries to keep customers updated.",
        "isAndroidBackgroundLocationEnabled": true,
        "isAndroidForegroundServiceEnabled": true
      }]
    ]
  }
}
```

**Background task pattern (define at module level, not inside a component):**
```typescript
// Source: expo-location docs [ASSUMED pattern — verify against expo-location 0.30 docs]
import * as Location from 'expo-location';
import * as TaskManager from 'expo-task-manager';

export const LOCATION_TASK_NAME = 'homechef-delivery-tracking';

TaskManager.defineTask(LOCATION_TASK_NAME, async ({ data, error }) => {
  if (error) return;
  const { locations } = data as { locations: Location.LocationObject[] };
  const latest = locations[locations.length - 1];
  // Call API — must use stored token from SecureStore (no React context in background)
  await updateDriverLocation(latest.coords.latitude, latest.coords.longitude);
});
```

**Starting tracking on delivery accept:**
```typescript
// Source: expo-location docs [ASSUMED pattern]
await Location.startLocationUpdatesAsync(LOCATION_TASK_NAME, {
  accuracy: Location.Accuracy.Balanced,
  timeInterval: 15000,      // 15 seconds (D-02)
  distanceInterval: 30,     // 30 metres minimum movement
  foregroundService: {      // Android only (D-05)
    notificationTitle: 'HomeChef Delivery',
    notificationBody: 'Tracking your location during delivery',
    notificationColor: '#FF6B35',
  },
  pausesUpdatesAutomatically: false,
  showsBackgroundLocationIndicator: true, // iOS blue bar
});
```

**Stopping on delivery complete:**
```typescript
const isTracking = await Location.hasStartedLocationUpdatesAsync(LOCATION_TASK_NAME);
if (isTracking) {
  await Location.stopLocationUpdatesAsync(LOCATION_TASK_NAME);
}
```

### Pattern 2: Location Permission Rationale Screen (CRITICAL — App Store)

**What:** Show an in-app rationale screen BEFORE requesting background location permission. Apple rejects apps that request `Always` without user-facing justification.

**Trigger:** Only when driver taps Accept on a delivery for the first time (gate by checking existing permission status first).

**Flow:**
1. Check `Location.getPermissionsAsync()` — if already `granted (always)`, skip all UI
2. If `foreground` only, show rationale modal explaining background tracking
3. After user taps "Allow" in rationale: call `Location.requestForegroundPermissionsAsync()` (if not already foreground)
4. Then call `Location.requestBackgroundPermissionsAsync()` (shows system dialog)
5. If denied: continue delivery without background tracking (foreground-only fallback acceptable)

**Do NOT ask for background location in `_layout.tsx` or on app start.** Gate entirely on the delivery accept action.

### Pattern 3: Actionable Push Notifications (Vendor)

**What:** iOS notification categories and Android notification actions allow buttons on the lock screen notification.

**iOS requires:**
- Define a category with identifier (e.g., `"new_order"`) containing action identifiers
- Register categories via `Notifications.setNotificationCategoryAsync()` on app start
- Include `categoryIdentifier` in the FCM payload `apns.aps` field

**Android requires:**
- Include `actions` array in the FCM `android` payload

**Backend additions to `fcmMessageBody` in push.go:**

The current `fcmMessage` struct only has `Token`, `Notification`, and `Data`. Actionable notifications for vendor new orders require platform-specific additions:

```go
// Source: FCM HTTP v1 API docs [ASSUMED — verify field names against current FCM docs]
type fcmMessageBody struct {
    Token        string            `json:"token"`
    Notification *fcmNotification  `json:"notification,omitempty"`
    Data         map[string]string `json:"data,omitempty"`
    Android      *fcmAndroid       `json:"android,omitempty"`
    APNS         *fcmAPNS          `json:"apns,omitempty"`
}

type fcmAndroid struct {
    Priority       string              `json:"priority,omitempty"` // "high"
    Notification   *fcmAndroidNotif    `json:"notification,omitempty"`
}

type fcmAndroidNotif struct {
    ChannelID  string   `json:"channel_id,omitempty"`
    Actions    []fcmAndroidAction `json:"actions,omitempty"` // Android 13+ notification actions
    Sound      string   `json:"sound,omitempty"`
    Priority   string   `json:"notification_priority,omitempty"`
}

type fcmAndroidAction struct {
    Action      string `json:"action"`       // Deep link action
    Title       string `json:"title"`
    Icon        string `json:"icon,omitempty"`
}

type fcmAPNS struct {
    Payload *fcmAPNSPayload `json:"payload,omitempty"`
    Headers map[string]string `json:"headers,omitempty"`
}

type fcmAPNSPayload struct {
    APS *fcmAPS `json:"aps,omitempty"`
}

type fcmAPS struct {
    Category    string `json:"category,omitempty"` // maps to iOS category identifier
    Sound       string `json:"sound,omitempty"`    // "default" or custom sound
    Badge       int    `json:"badge,omitempty"`
    ContentAvailable int `json:"content-available,omitempty"`
}
```

**Mobile side — register iOS categories (in vendor app `_layout.tsx` after auth):**
```typescript
// Source: expo-notifications docs [ASSUMED pattern]
await Notifications.setNotificationCategoryAsync('new_order', [
  {
    identifier: 'ACCEPT_ORDER',
    buttonTitle: 'Accept',
    options: { opensAppToForeground: false }, // handles in background
  },
  {
    identifier: 'REJECT_ORDER',
    buttonTitle: 'Reject',
    options: { isDestructive: true, opensAppToForeground: false },
  },
]);
```

**Handling background notification response:**
```typescript
Notifications.addNotificationResponseReceivedListener(async (response) => {
  const actionId = response.actionIdentifier;
  const orderId = response.notification.request.content.data?.orderId as string;
  if (actionId === 'ACCEPT_ORDER') {
    await api.put(`/v1/chef/orders/${orderId}/status`, { status: 'accepted' });
  } else if (actionId === 'REJECT_ORDER') {
    await api.put(`/v1/chef/orders/${orderId}/status`, { status: 'rejected' });
  }
});
```

### Pattern 4: WebSocket Endpoint in Go (Customer Tracking)

**What:** Gin route upgrades HTTP to WebSocket. Handler subscribes to NATS subject `delivery.location.{deliveryID}`, forwards location messages to connected client, handles client disconnect.

**Critical:** `gorilla/websocket` is not in `go.mod`. Must `go get github.com/gorilla/websocket`.

**Key observations from existing code:**
- NATS `DELIVERY` stream already covers `delivery.*` subjects — `delivery.location.{id}` is within scope with NO new stream required [VERIFIED: nats.go:200-210 — stream subjects: `"delivery.*"`]
- NATS `NATSClient.Subscribe(subject, handler)` is the correct non-JetStream subscribe call for transient location fan-out (not persisted, not durable consumer needed)

**New subject constant to add in nats.go:**
```go
SubjectDeliveryLocation = "delivery.location" // Base; full subject: delivery.location.{deliveryID}
```

**UpdateLocation handler additions (publish location update):**
```go
// After saving partner location to DB, publish to NATS (non-blocking)
go func() {
    locationEvent := map[string]interface{}{
        "deliveryId": deliveryID,   // Must look up active delivery for this partner
        "latitude":   req.Latitude,
        "longitude":  req.Longitude,
        "timestamp":  time.Now().UTC(),
    }
    subject := fmt.Sprintf("delivery.location.%s", deliveryID)
    if err := services.GetNATSClient().Publish(subject, locationEvent); err != nil {
        log.Printf("Failed to publish location event: %v", err)
    }
}()
```

**WebSocket handler pattern:**
```go
// Source: gorilla/websocket docs [ASSUMED pattern — verify against gorilla/websocket v1.5.x]
import "github.com/gorilla/websocket"

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool { return true }, // CORS — restrict in prod
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
}

func (h *OrderHandler) TrackOrderWS(c *gin.Context) {
    orderID := c.Param("id")
    userID, _ := middleware.GetUserID(c)

    // Verify order ownership
    var order models.Order
    if err := database.DB.Preload("Delivery").
        Where("id = ? AND customer_id = ?", orderID, userID).
        First(&order).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
        return
    }
    if order.Delivery == nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "No active delivery"})
        return
    }
    deliveryID := order.Delivery.ID.String()

    // Upgrade connection
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        log.Printf("WebSocket upgrade failed: %v", err)
        return
    }
    defer conn.Close()

    // Subscribe to NATS location events for this delivery
    subject := fmt.Sprintf("delivery.location.%s", deliveryID)
    sub, err := services.GetNATSClient().Subscribe(subject, func(msg *nats.Msg) {
        if err := conn.WriteMessage(websocket.TextMessage, msg.Data); err != nil {
            log.Printf("WS write failed: %v", err)
        }
    })
    if err != nil {
        log.Printf("NATS subscribe failed: %v", err)
        return
    }
    defer sub.Unsubscribe()

    // Keep connection alive — read pump (handles pings and client-side close)
    for {
        if _, _, err := conn.ReadMessage(); err != nil {
            break // Client disconnected
        }
    }
}
```

**Route addition in routes.go:**
```go
// Inside orders group (already authenticated)
orders.GET("/:id/track/ws", orderHandler.TrackOrderWS)
```

### Pattern 5: Rate Limiting on UpdateLocation (Backend)

**What:** In-memory per-user rate limiter using `golang.org/x/time/rate` (already in go.mod as indirect dependency). Limits PUT /delivery/location to 1 request per 10 seconds per user.

**Why not a gin middleware here:** The rate limiter needs per-user state. Approach: add rate limiter map to `DeliveryHandler` struct (same pattern as `ChatHandler.rateLimit`).

```go
// In DeliveryHandler struct
type DeliveryHandler struct {
    locationRateLimit map[uuid.UUID]*rate.Limiter
    locationRateMu    sync.RWMutex
}

func (h *DeliveryHandler) getLocationLimiter(userID uuid.UUID) *rate.Limiter {
    h.locationRateMu.RLock()
    limiter, exists := h.locationRateLimit[userID]
    h.locationRateMu.RUnlock()
    if !exists {
        // 1 event per 10 seconds
        limiter = rate.NewLimiter(rate.Every(10*time.Second), 1)
        h.locationRateMu.Lock()
        h.locationRateLimit[userID] = limiter
        h.locationRateMu.Unlock()
    }
    return limiter
}

// In UpdateLocation handler, before the DB save:
limiter := h.getLocationLimiter(userID)
if !limiter.Allow() {
    c.JSON(http.StatusTooManyRequests, gin.H{"error": "Location update rate limit exceeded"})
    return
}
```

**Note:** This shares the same in-memory limitation as the chat rate limiter (CONCERNS.md) — won't work across multiple API replicas. However, for MVP with Knative scale-to-zero (typically 1 replica active), this is acceptable. Document the known limitation.

### Pattern 6: WebSocket Client in React Native (Customer)

**What:** Replace React Query polling in `useOrderTracking.ts` with native WebSocket + automatic reconnect. Fall back to polling after 3 consecutive failures.

**New `useOrderTrackingWS.ts` hook pattern:**
```typescript
// Source: React Native WebSocket docs [ASSUMED — standard RN WebSocket API]
import { useEffect, useRef, useCallback } from 'react';
import { useOrderTracking } from './useOrderTracking'; // polling fallback

const MAX_WS_FAILURES = 3;

export function useOrderTrackingWS(orderId: string, enabled: boolean = true) {
  const wsRef = useRef<WebSocket | null>(null);
  const failureCount = useRef(0);
  const [usePolling, setUsePolling] = useState(false);
  const [driverLocation, setDriverLocation] = useState<{latitude: number; longitude: number} | null>(null);

  const connect = useCallback(() => {
    if (!orderId || !enabled || usePolling) return;
    const ws = new WebSocket(`${WS_BASE_URL}/api/v1/orders/${orderId}/track/ws`);
    wsRef.current = ws;

    ws.onmessage = (event) => {
      failureCount.current = 0;
      const data = JSON.parse(event.data);
      setDriverLocation({ latitude: data.latitude, longitude: data.longitude });
    };

    ws.onerror = () => {
      failureCount.current += 1;
      if (failureCount.current >= MAX_WS_FAILURES) {
        setUsePolling(true); // Fall back to polling
      }
    };

    ws.onclose = () => {
      // Reconnect after 2s unless we've given up
      if (failureCount.current < MAX_WS_FAILURES) {
        setTimeout(connect, 2000);
      }
    };
  }, [orderId, enabled, usePolling]);

  useEffect(() => {
    connect();
    return () => wsRef.current?.close();
  }, [connect]);

  // Polling fallback
  const pollingResult = useOrderTracking(orderId, enabled && usePolling);

  return { driverLocation, isPolling: usePolling, pollingResult };
}
```

### Pattern 7: DeliveryResponse Serialization Fix

**What:** `Delivery.ToResponse()` in `models/delivery.go` currently omits lat/lng fields. The `Delivery` model has all required fields; they just aren't in the DTO.

**Current `DeliveryResponse` struct (lines 276-286):**
- Has: `ID`, `OrderID`, `Status`, `Distance`, `EstimatedDuration`, `DeliveryFee`, `AssignedAt`, `PickedUpAt`, `DeliveredAt`
- Missing: `CurrentLatitude`, `CurrentLongitude`, `DropoffLatitude`, `DropoffLongitude`, `PickupLatitude`, `PickupLongitude`

**Fix — add to `DeliveryResponse` struct and `ToResponse()` method:**
```go
type DeliveryResponse struct {
    // ... existing fields ...
    CurrentLatitude   float64 `json:"currentLatitude"`
    CurrentLongitude  float64 `json:"currentLongitude"`
    DropoffLatitude   float64 `json:"dropoffLatitude"`
    DropoffLongitude  float64 `json:"dropoffLongitude"`
    PickupLatitude    float64 `json:"pickupLatitude"`
    PickupLongitude   float64 `json:"pickupLongitude"`
}
```

The `DeliveryPartner.CurrentLatitude/Longitude` lives on `DeliveryPartner`, not on `Delivery`. This means `ToResponse()` needs access to the partner's current position. The `UpdateLocation` handler already stores driver coords in `DeliveryPartner.CurrentLatitude/Longitude` — these need to be included when constructing the response. The `Delivery` model has a `DeliveryPartner` relationship field, so Preloading it gives access.

**Note for planner:** `TrackOrder` in orders.go uses `order.Delivery.ToResponse()` — it Preloads `Delivery` but NOT `Delivery.DeliveryPartner`. That Preload needs to be added for the serialization fix to work end-to-end.

### Pattern 8: Push Notification Wiring (All Apps)

**What:** Wire `getRawFCMToken` + `registerDeviceToken` from `usePushToken.ts` into each app's `_layout.tsx` after successful authentication. Also configure notification channels on Android.

**Current state (confirmed by codebase inspection):**
- `usePushToken.ts` is built and correct [VERIFIED]
- `expo-notifications` v0.30.2 is in all 3 package.json files [VERIFIED]
- NO notification wiring exists in any `_layout.tsx` (grep confirmed zero matches) [VERIFIED]

**Pattern for `_layout.tsx` (all 3 apps):**
```typescript
import * as Notifications from 'expo-notifications';
import { getRawFCMToken, registerDeviceToken } from '@homechef/mobile-shared';

// Set notification handler (show while app is foregrounded)
Notifications.setNotificationHandler({
  handleNotification: async () => ({
    shouldShowAlert: true,
    shouldPlaySound: true,
    shouldSetBadge: true,
  }),
});

// After auth success:
async function setupPushNotifications(apiClient: AxiosInstance) {
  // Android channels (must be created before first notification)
  if (Platform.OS === 'android') {
    await Notifications.setNotificationChannelAsync('order-updates', {
      name: 'Order Updates',
      importance: Notifications.AndroidImportance.HIGH,
      sound: 'default',
    });
    // ... other channels per D-09
  }

  const token = await getRawFCMToken();
  if (token) {
    await registerDeviceToken(apiClient, token);
  }

  // Listen for token rotation (re-register on change)
  const subscription = Notifications.addPushTokenListener((event) => {
    registerDeviceToken(apiClient, event.data);
  });
  return () => subscription.remove();
}
```

**Badge count update:**
```typescript
Notifications.addNotificationReceivedListener(async () => {
  const currentCount = await Notifications.getBadgeCountAsync();
  await Notifications.setBadgeCountAsync(currentCount + 1);
});
```

### Pattern 9: UX Polish Patterns

**Pull-to-refresh (all list views):**
```typescript
// Source: React Native docs [VERIFIED: standard RN API]
import { RefreshControl, FlatList } from 'react-native';
const { data, refetch, isRefetching } = useQuery(...);

<FlatList
  refreshControl={
    <RefreshControl
      refreshing={isRefetching}
      onRefresh={refetch}
      colors={['#FF6B35']} // brand color
    />
  }
/>
```

**Skeleton screens:**
```typescript
// NativeWind shimmer pattern (no library needed)
// Use animated opacity or a simple gray placeholder with NativeWind classes
<View className="bg-gray-200 rounded-lg h-20 w-full animate-pulse" />
```

**Haptic feedback — existing `expo-haptics` usage pattern:**
```typescript
// Source: expo-haptics docs [VERIFIED: package in all 3 apps]
import * as Haptics from 'expo-haptics';

// Order placed / delivery complete — heavy impact
await Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Heavy);

// Accept/reject — medium impact
await Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium);

// Success notification style
await Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
```

**Offline detection with NetInfo:**
```typescript
// Source: @react-native-community/netinfo docs [ASSUMED — verify API shape]
import NetInfo from '@react-native-community/netinfo';

useEffect(() => {
  const unsubscribe = NetInfo.addEventListener(state => {
    if (!state.isConnected) {
      showToast('You are offline. Showing cached data.');
    }
  });
  return unsubscribe;
}, []);
```

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Background GPS task | Custom timer-based location fetch | `expo-location.startLocationUpdatesAsync` + `expo-task-manager` | OS-managed wakeup; survives app kill; required for iOS background modes |
| WebSocket in Go | Raw `net/http` hijack + manual frame handling | `gorilla/websocket` | Frame parsing, ping/pong, concurrent writes, connection upgrades — all handled |
| Actionable push notification UI | Custom in-app action sheet | iOS categories + Android notification actions via FCM payload | Native lock-screen buttons; works when app is killed |
| Rate limiting token bucket | Custom `time.Ticker` approach | `golang.org/x/time/rate.Limiter` | Already in go.mod (indirect); correct token bucket implementation |
| Haptic patterns | Custom vibration sequences | `expo-haptics` | Already installed; platform-correct feedback intensities |
| Network connectivity detection | WebSocket/HTTP polling for connectivity | `@react-native-community/netinfo` | Listens to native network events; works offline without network calls |

---

## Common Pitfalls

### Pitfall 1: expo-location/expo-task-manager Missing from app.json — Silent Failure
**What goes wrong:** Adding expo-location to `package.json` without adding it to `app.json` plugins means the native module is not compiled into the binary. `startLocationUpdatesAsync` will throw or fail silently at runtime. Background location will never fire.
**How to avoid:** Update `app.json` plugins AND rebuild with EAS (`eas build`) before testing background GPS. `expo start` / `expo go` will not be enough — background location requires a development build or production build.
**Warning signs:** `startLocationUpdatesAsync` returns without error but task never fires.

### Pitfall 2: Background Task API Calls Can't Use React Context
**What goes wrong:** The background task callback (`TaskManager.defineTask`) runs in a separate JS context without React state, providers, or hooks. Calling `api.get(...)` from the task will fail if the axios instance is created inside a React component.
**How to avoid:** In the background task, create a standalone API call using `expo-secure-store` to read the stored JWT directly and use `fetch` or a standalone axios instance (not the React-context-bound one).
**Warning signs:** Background task fires but API calls return 401 (no auth header).

### Pitfall 3: WebSocket and Gin Middleware Incompatibility
**What goes wrong:** Gin's response writer is buffered. After calling `upgrader.Upgrade(c.Writer, c.Request, nil)`, Gin's middleware (particularly response-wrapping middleware like the Prometheus metrics middleware) will attempt to finalize the response, interfering with the WebSocket connection.
**How to avoid:** Hijack the connection early; ensure no response-writing middleware runs after the WebSocket upgrade. Use `c.Set("websocket", true)` before the upgrade and skip response-wrapping middleware for WebSocket routes. Or register the WebSocket route without middleware wrappers.
**Warning signs:** WebSocket upgrade returns HTTP 200 but immediately closes; Prometheus middleware logs errors.

### Pitfall 4: NATS Subscribe (Core) vs JetStream Consumer for Location Fan-Out
**What goes wrong:** Using `NATSClient.js.Subscribe` (JetStream) for `delivery.location.*` will require a durable consumer. Location events are ephemeral fan-out (broadcast to all connected WebSocket clients for a delivery) — JetStream durable consumers are for reliable delivery to exactly one consumer, not fan-out.
**How to avoid:** Use `NATSClient.conn.Subscribe()` (core NATS, not JetStream) for the WebSocket handler. Core NATS supports pub/sub fan-out natively. The `NATSClient.Subscribe(subject, handler)` method already wraps `conn.Subscribe` correctly [VERIFIED: nats.go:348-358].
**Warning signs:** Only one WebSocket client receives updates even when multiple customers are tracking the same order.

### Pitfall 5: iOS Requires Category Registration Before Notification Arrives
**What goes wrong:** iOS notification categories (`Notifications.setNotificationCategoryAsync`) MUST be registered before the first notification arrives, not lazily. If the vendor app receives a `new_order` notification before categories are registered, the lock-screen buttons will not appear.
**How to avoid:** Register all notification categories in `_layout.tsx` on app launch (before auth), not after auth completes. Categories are safe to register even when user is not logged in.
**Warning signs:** Vendor receives notification but no Accept/Reject buttons appear on iOS.

### Pitfall 6: DeliveryPartner CurrentLatitude/Longitude Not Preloaded in TrackOrder
**What goes wrong:** `TrackOrder` in orders.go currently Preloads `Delivery` and `Chef` but NOT `Delivery.DeliveryPartner`. After the serialization fix adds `currentLatitude`/`currentLongitude` to `DeliveryResponse`, they will always be 0.0 because the partner's live location isn't loaded.
**How to avoid:** Update the TrackOrder DB query to also Preload `Delivery.DeliveryPartner` (or query partner separately). For the WebSocket endpoint, location comes from NATS events so this is less critical for the WS path — but the initial HTTP response before WS connect must include the current location.
**Warning signs:** Map initially shows driver at 0,0 (ocean off Africa) before WebSocket updates arrive.

### Pitfall 7: `gorilla/websocket` Write Concurrency
**What goes wrong:** `gorilla/websocket` connections are NOT safe for concurrent writes. If multiple NATS messages arrive quickly and the handler calls `conn.WriteMessage` from the NATS callback goroutine AND a ping handler goroutine simultaneously, there will be data races and connection corruption.
**How to avoid:** Use a write channel or mutex. Simple approach: send NATS messages to a buffered channel, have a single goroutine drain the channel and write to the WebSocket.
**Warning signs:** WebSocket connection drops randomly under load; race detector flags the connection.

### Pitfall 8: Android Notification Channel Must Be Created BEFORE First Notification
**What goes wrong:** Android notification channels must exist before any notification is sent to that channel. If the app is freshly installed and a push arrives before the user opens the app (which registers the channels), the notification is dropped or shows with wrong importance level.
**How to avoid:** Create channels in `Notifications.setNotificationCategoryAsync` at the very start of `_layout.tsx`, before auth, not after. Channels are idempotent — calling `setNotificationChannelAsync` on an already-created channel with the same ID is a no-op.

---

## Code Examples

### FCM Data Payload Fields (for Deep Linking and Actions)

The existing `push.go` `sendToToken` sends a `data` map. Include these keys for mobile-side routing:
```go
// Source: codebase inspection of push.go [VERIFIED]
data := map[string]string{
    "type":    "new_order",        // for routing switch in notification listener
    "orderId": order.ID.String(),  // for deep link
    "action":  "vendor_new_order", // for category assignment
}
```

### NATS Stream Coverage for delivery.location.*

The DELIVERY stream is already configured (nats.go line 199):
```go
// Source: apps/api/services/nats.go [VERIFIED]
jetstream.StreamConfig{
    Name:     "DELIVERY",
    Subjects: []string{"delivery.*"},  // covers delivery.location.{id}
    // WorkQueuePolicy — note: location events are ephemeral, consider not persisting
}
```

Note for planner: `delivery.location.{deliveryID}` will be persisted in the JetStream DELIVERY stream. For high-frequency GPS updates (15s interval, multiple drivers), this may accumulate significant storage. Consider either:
- Publishing location events to core NATS (not JetStream) so they are NOT persisted
- Or adding a short MaxAge to location events

The `NATSClient.Publish()` method uses `n.conn.Publish()` (core NATS, not JetStream) [VERIFIED: nats.go:315-329]. This means location events published via `Publish()` will NOT be stored in JetStream unless there's also a JetStream stream with matching subjects. The DELIVERY stream DOES match `delivery.*` — so if JetStream is configured, events WILL be stored. **Use core NATS `Publish` for location updates to avoid JetStream storage accumulation.**

### Existing AcceptDelivery Handler (GPS Start Trigger Point)

```
apps/mobile-delivery/app/(tabs)/active.tsx  — useUpdateDeliveryStatus mutation
apps/mobile-delivery/hooks/useDriverDeliveries.ts — mutation calls PUT /:id/status
```
The GPS start should be triggered in the `useUpdateDeliveryStatus` success callback when `nextStatus === 'assigned'` (i.e., when the delivery is accepted and assigned to the driver). The acceptance itself is via `POST /:id/accept` in `delivery.go`.

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Customer polling every 5s | WebSocket with polling fallback | Phase 4 (this phase) | Eliminates 86,400 GET calls/driver/day under continuous tracking |
| No rate limit on PUT /delivery/location | Per-user token bucket (1 req/10s) | Phase 4 (this phase) | 6x reduction in DB writes at same GPS update interval |
| `DeliveryResponse` omits lat/lng | Includes current + dropoff + pickup coords | Phase 4 (this phase) | Enables real driver pin on map (was using chef fallback) |
| No push notifications | FCM HTTP v1 with actionable vendor notifications | Phase 4 (this phase) | Vendor can accept/reject from lock screen without unlocking phone |

---

## Open Questions (RESOLVED)

1. **NATS location event JetStream persistence** — RESOLVED: Use core NATS (not JetStream)
   - Decision: `NATSClient.Publish()` uses `conn.Publish()` (core NATS) which is NOT intercepted by JetStream on most server configs. Confirmed: use core NATS `Publish` for location events. Location events are ephemeral fan-out and must NOT accumulate in JetStream storage. This is the existing behavior of `NATSClient.Publish()` — no change needed.

2. **Driver active delivery ID for GPS location publish** — RESOLVED: WHERE clause query with DB index
   - Decision: In `UpdateLocation`, query `SELECT id FROM deliveries WHERE delivery_partner_id = ? AND status IN ('assigned','picked_up','in_transit') LIMIT 1`. Add a DB migration creating an index on `deliveries(delivery_partner_id, status)` to keep this lookup fast. Plan 04-01 Task 1 includes this migration (see below).

3. **gorilla/websocket vs nhooyr.io/websocket** — RESOLVED: Use gorilla/websocket
   - Decision: `gorilla/websocket` v1.5.3. Battle-tested, abundant Gin examples, maintenance-mode does not affect correctness for MVP. Can migrate to `nhooyr.io/websocket` later if needed.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| expo-location | DRIV-12/13 | Not installed | — | None — required for background GPS; must add + EAS rebuild |
| expo-task-manager | DRIV-12 | Not installed | — | None — required for expo-location background mode; must add + EAS rebuild |
| gorilla/websocket | D-13 WebSocket endpoint | Not in go.mod | — | None — required for WebSocket upgrade; must `go get` |
| @react-native-community/netinfo | UX-03 | Not installed in customer/vendor | — | Manual polling fallback (not recommended) |
| golang.org/x/time/rate | D-17 rate limiting | Available (indirect in go.mod) | v0.15.0 | — |
| expo-haptics | UX-04 | Installed in all 3 apps | ~55.0.11 | — |
| expo-notifications | PUSH-01..06 | Installed in all 3 apps | ~0.30.2 | — |
| NATS server | D-15 location broadcast | Assumed running (existing NATS config) | — | — |

**Missing dependencies with no fallback (block execution):**
- `expo-location` + `expo-task-manager` in `apps/mobile-delivery` — without these, background GPS is impossible. Wave 0 must add to package.json AND app.json plugins AND trigger EAS Build.
- `gorilla/websocket` (or equivalent) in `apps/api` — WebSocket endpoint cannot be written without it.

**Missing dependencies with fallback:**
- `@react-native-community/netinfo` — offline detection (UX-03) can be deferred or implemented with a simpler `fetch`-based connectivity check as fallback, but the proper library is strongly preferred.

---

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes (push token registration) | JWT Bearer token required on `PUT /profile/device-token`; existing `AuthMiddleware()` [VERIFIED] |
| V3 Session Management | no | WebSocket session tied to authenticated HTTP upgrade; no separate session |
| V4 Access Control | yes (WebSocket endpoint) | `AuthMiddleware()` + ownership check (customer can only track their own orders) [pattern from existing TrackOrder] |
| V5 Input Validation | yes | lat/lng bounds validation in UpdateLocation; WebSocket message parsing |
| V6 Cryptography | no | No new crypto; FCM uses GCP Application Default Credentials [VERIFIED: push.go] |

### Known Threat Patterns for Phase 4 Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| WebSocket connection to another user's delivery | Spoofing | Verify `customer_id = ?` in initial HTTP upgrade before upgrading — pattern from existing `TrackOrder` |
| GPS coordinate spoofing (driver sends fake location) | Tampering | Out of scope for v1 — rate limiting and auth are sufficient MVP controls |
| Notification token exfiltration | Information Disclosure | Token stored in `users.fcm_token` column (existing), only sent to FCM — no change to exposure |
| Push notification action spoofing (fake accept/reject) | Tampering | Action handler calls authenticated API with user's JWT — same auth as UI button |
| Rate limit bypass (multiple connections per user) | Elevation of Privilege | Per-user in-memory limiter; single-replica Knative deployment for MVP acceptable |

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `expo-location ~0.30.x` is compatible with Expo 55 / React Native 0.83.4 | Standard Stack | Wrong SDK match causes native module crash; verify against expo-location CHANGELOG before installing |
| A2 | `expo-task-manager ~0.20.x` is compatible with Expo 55 | Standard Stack | Same risk as A1 |
| A3 | `gorilla/websocket ^1.5.3` is current stable version | Standard Stack | Minor version difference acceptable; verify with `go get github.com/gorilla/websocket@latest` |
| A4 | `@react-native-community/netinfo ~11.x` is compatible with React Native 0.83.4 | Standard Stack | API shape may differ; check peerDeps in netinfo package.json before installing |
| A5 | FCM HTTP v1 `android.notification.actions` field structure | Architecture Patterns (actionable push) | FCM payload structure may differ; verify against FCM HTTP v1 API docs before implementing |
| A6 | iOS `setNotificationCategoryAsync` with `opensAppToForeground: false` allows background API calls | Architecture Patterns (actionable push) | If false assumption: action always foregrounds the app, breaking lock-screen UX |
| A7 | `golang.org/x/time/rate` (already in go.mod indirect) is importable as direct dependency | Don't Hand-Roll | If module graph resolution fails, must explicitly add it; `go get golang.org/x/time` resolves this |
| A8 | Core NATS `Publish()` events to `delivery.location.*` are NOT intercepted by the DELIVERY JetStream stream | Code Examples | If wrong, location events accumulate in JetStream storage; mitigation: monitor DELIVERY stream message count after implementing |

---

## Sources

### Primary (HIGH confidence — codebase verified)
- `apps/api/services/push.go` — FCM sender structure, no actionable notification fields present
- `apps/api/services/nats.go` — NATS subjects, stream config, Subscribe/Publish methods
- `apps/api/handlers/delivery.go` — UpdateLocation handler (no rate limit, no NATS publish)
- `apps/api/handlers/orders.go` — TrackOrder handler (serialization gap confirmed)
- `apps/api/models/delivery.go` — DeliveryResponse struct (missing lat/lng confirmed)
- `apps/api/routes/routes.go` — All routes; WebSocket endpoint absent; delivery routes confirmed
- `apps/api/middleware/auth.go` — AuthMiddleware pattern for WebSocket auth
- `apps/api/go.mod` — All Go dependencies; gorilla/websocket absent; x/time/rate present indirect
- `apps/mobile-delivery/package.json` — expo-location/task-manager absent confirmed
- `apps/mobile-customer/package.json` — expo-notifications, expo-haptics, react-native-maps present
- `apps/mobile-vendor/package.json` — expo-notifications, expo-haptics present
- `packages/mobile-shared/src/hooks/usePushToken.ts` — getRawFCMToken + registerDeviceToken built
- `apps/mobile-customer/hooks/useOrderTracking.ts` — 5s polling hook (upgrade target)
- `apps/mobile-customer/app/order/[id]/track.tsx` — WebSocket integration point
- `apps/mobile-delivery/app/(tabs)/active.tsx` — GPS start trigger point (delivery accept screen)
- `apps/mobile-delivery/app.json` — No background location permissions/plugins currently
- `.planning/research/PITFALLS.md` — DB saturation analysis, App Store rejection risk
- `.planning/phases/04-gps-push-polish/04-CONTEXT.md` — All locked decisions

### Secondary (MEDIUM confidence — prior research phase)
- Phase 1 RESEARCH.md — expo-notifications v0.30.2, getDevicePushTokenAsync pattern, Expo SDK 55 compatibility

### Tertiary (LOW confidence — training knowledge, needs verification)
- gorilla/websocket API patterns (tag [ASSUMED] in code examples)
- expo-location 0.30.x background task API shape (tag [ASSUMED] in code examples)
- FCM HTTP v1 actionable notification payload format (tag [ASSUMED] in code examples)
- @react-native-community/netinfo API shape

---

## Metadata

**Confidence breakdown:**
- Standard Stack: HIGH (codebase verified for existing; ASSUMED version tags for new libraries)
- Architecture: HIGH for Go changes (codebase directly inspected); MEDIUM for mobile patterns (confirmed library presence; specific API calls ASSUMED)
- Pitfalls: HIGH (7 of 8 from direct codebase inspection; 1 from general knowledge)

**Research date:** 2026-04-06
**Valid until:** 2026-05-06 (libraries stable; Go FCM payload format may evolve)
