# Phase 4: GPS, Push + Polish - Context

**Gathered:** 2026-04-06
**Status:** Ready for planning

<domain>
## Phase Boundary

Final phase: background GPS tracking for driver app, push notifications across all 3 apps, customer map upgrade to real-time WebSocket (with backend changes), and UX polish (pull-to-refresh, skeletons, offline states, haptics) across all 3 apps. This phase includes Go API backend changes — it is NOT mobile-only.

</domain>

<decisions>
## Implementation Decisions

### Background GPS (Driver)
- **D-01:** Background GPS tracking starts when driver taps Accept on a delivery, stops automatically when delivery status becomes `delivered` or `cancelled` — NOT on going online (saves battery, only tracks during actual deliveries)
- **D-02:** GPS update rate at Claude's discretion — recommend 15 seconds (balance between real-time feel and battery/DB load). Research flagged DB saturation risk; Phase 4 must also add basic rate limiting to `PUT /delivery/location` on backend (Phase 1 concern CONCERNS.md).
- **D-03:** Background location permission MUST be requested via rationale screen first, then system prompt — required to pass App Store review (Apple rejects apps that request background location without context)
- **D-04:** Use expo-location `startLocationUpdatesAsync` with `expo-task-manager` for background task — confirmed in Phase 1 research
- **D-05:** Android foreground service notification required ("HomeChef Delivery is tracking your location") — iOS handles this automatically via background modes

### Push Notifications
- **D-06:** Direct FCM via `getDevicePushTokenAsync()` — raw FCM tokens sent to existing `push.go` backend (matches Phase 1 D-09). NO Expo Push Service.
- **D-07:** Vendor notifications MUST be actionable with lock-screen Accept/Reject buttons — this is the primary vendor value, makes vendor app usable without unlocking phone
- **D-08:** Deep link routing: notification taps open the specific screen (order detail for order updates, delivery detail for driver available deliveries), NOT just launching the app
- **D-09:** Notification channels (Android) per app:
  - Customer: "Order Updates" (high importance), "Promotions" (default)
  - Vendor: "New Orders" (max importance, sound+vibration), "Order Updates" (high)
  - Driver: "New Deliveries" (max importance), "Delivery Updates" (high)
- **D-10:** FCM device token registration already implemented in Phase 1 `usePushToken.ts` — Phase 4 wires it into actual login flow and subscribes to notifications
- **D-11:** Vendor sound alert (deferred from Phase 3 D-02) is now handled by push notification sound (native notification audio), not in-app `expo-av` chime

### Customer Map Upgrade (WebSocket + Backend Changes)
- **D-12:** Customer tracking map upgrades from polling to WebSocket for real-time driver location updates — BACKEND CHANGES REQUIRED
- **D-13:** New Go WebSocket endpoint: `GET /api/v1/orders/:id/track/ws` — upgrades HTTP connection, streams driver location updates from NATS events
- **D-14:** Backend serialization fix: `DeliveryResponse.ToResponse()` must include `currentLatitude`, `currentLongitude`, `dropoffLatitude`, `dropoffLongitude` fields (currently omitted — Phase 2 workaround used chef coords as fallback)
- **D-15:** NATS broadcasting: driver GPS updates published to NATS subject `delivery.location.{deliveryId}` — WebSocket handler subscribes and forwards to connected customers
- **D-16:** Mobile client: react-native WebSocket with automatic reconnect on disconnect, fallback to polling after 3 failed reconnects
- **D-17:** Rate limiting on `PUT /delivery/location`: use gin middleware, limit to max 1 request per 10 seconds per user (prevents DB saturation)

### UX Polish
- **D-18:** Polish ALL UX requirements (UX-01..04) across all 3 apps — audit first, add where missing
- **D-19:** Offline error state at Claude's discretion — recommend NetInfo + non-blocking toast + cached React Query data browsing (not full-screen blocker)
- **D-20:** Haptic feedback on key actions: order placed, accept/reject, delivery status transitions, payment success
- **D-21:** Skeleton screens on all loading states — home feeds, order lists, menu, profile
- **D-22:** Pull-to-refresh on all list views

### Claude's Discretion
- GPS update rate (D-02)
- Offline error state pattern (D-19)
- Audit-based polish implementation (D-18)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project Context
- `.planning/PROJECT.md` — Project vision, constraints
- `.planning/REQUIREMENTS.md` — DRIV-12, DRIV-13, PUSH-01..06, UX-01..04 (12 requirements)

### Prior Phase Decisions
- `.planning/phases/01-foundation-auth/01-CONTEXT.md` — D-09 FCM token type (getDevicePushTokenAsync)
- `.planning/phases/02-customer-app/02-CONTEXT.md` — D-05 5s polling (now upgrading to WebSocket)
- `.planning/phases/03-vendor-app-driver-core/03-CONTEXT.md` — Deferred sound alert now handled via push

### Phase 1 Research
- `.planning/phases/01-foundation-auth/01-RESEARCH.md` — FCM token type warning, Apple background location rejection risk

### Project Research
- `.planning/research/PITFALLS.md` — DB saturation on location updates (CRITICAL — must add rate limiting)
- `.planning/research/STACK.md` — expo-location + expo-task-manager + react-native-maps confirmed

### Mobile Infrastructure (Phase 1)
- `packages/mobile-shared/src/hooks/usePushToken.ts` — FCM token registration (wire into login)
- `apps/mobile-delivery/` — Driver app that gets background GPS
- `apps/mobile-customer/hooks/useOrderTracking.ts` — Currently uses polling, upgrade to WebSocket
- `apps/mobile-customer/components/tracking/DeliveryMap.tsx` — Currently uses chef fallback, upgrade to real driver coords

### Go API Backend
- `apps/api/services/push.go` — FCM push sender (already exists)
- `apps/api/handlers/delivery.go` — UpdateLocation endpoint (needs rate limiting)
- `apps/api/handlers/orders.go` — TrackOrder endpoint (needs serialization fix)
- `apps/api/models/delivery.go` — Delivery model with DropoffLatitude/DropoffLongitude fields
- `apps/api/services/nats.go` — NATS messaging (broadcast driver location updates)
- `apps/api/routes/routes.go` — Add new WebSocket route
- `.planning/codebase/CONCERNS.md` — Rate limiting gap flagged

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `packages/mobile-shared/src/hooks/usePushToken.ts` — FCM token registration already built, just needs wiring to login + notification permission request
- `apps/api/services/push.go` — Backend FCM sender already built, just needs actionable notification metadata
- `apps/api/services/nats.go` — NATS messaging exists, add location.broadcast subject
- `apps/mobile-customer/hooks/useOrderTracking.ts` — Polling hook to upgrade to WebSocket
- `apps/mobile-customer/components/tracking/DeliveryMap.tsx` — Map component already renders driver marker

### Established Patterns
- React Query for server state (keep for fallback)
- Zustand for GPS tracking state (track which delivery is being tracked)
- NativeWind for all styling
- expo-router file-based routing with deep links (already configured in app.json)

### Integration Points
- Driver active delivery screen (`apps/mobile-delivery/app/(tabs)/active.tsx`) — trigger GPS start on accept
- Customer order tracking screen (`apps/mobile-customer/app/order/[id]/track.tsx`) — swap polling hook for WebSocket hook
- All app `_layout.tsx` files — add push notification permission request + FCM token registration after auth
- Go API routes — add `/orders/:id/track/ws` WebSocket endpoint
- Go API middleware — add rate limiter to `/delivery/location`

### Backend Changes Required (NEW for Phase 4)
1. Rate limiting middleware on PUT /delivery/location (1 req/10s per user)
2. Add lat/lng fields to DeliveryResponse JSON output
3. NATS publisher in UpdateLocation handler
4. New WebSocket endpoint with NATS subscriber
5. Actionable notification payload format in push.go (iOS categories + Android actions)

</code_context>

<specifics>
## Specific Ideas

- **Critical:** Background location rationale screen before system prompt — App Store rejection risk is real
- **Critical:** Rate limiting on PUT /delivery/location — research flagged DB saturation as CRITICAL concern
- Vendor lock-screen accept/reject is the primary vendor value — do not compromise on this
- WebSocket upgrade is the only backend-touching mobile change — use it as opportunity to fix the Phase 2 serialization workaround

</specifics>

<deferred>
## Deferred Ideas

- Apple Pay / Google Pay native payment sheet — out of scope per PROJECT.md
- Offline mode with full sync — out of scope per PROJECT.md
- Scheduled order reminders (push 30min before delivery) — v2 differentiator
- Photo confirmation on delivery — v2 differentiator
- App Store rating prompt — v2 differentiator
- Shake to report issue — v2 differentiator

</deferred>

---

*Phase: 04-gps-push-polish*
*Context gathered: 2026-04-06*
