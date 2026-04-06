---
phase: 04-gps-push-polish
plan: "01"
subsystem: go-api
tags: [gps, websocket, push, rate-limiting, nats, backend]
dependency_graph:
  requires: []
  provides:
    - delivery-response-lat-lng
    - location-rate-limiter
    - nats-location-publish
    - websocket-tracking-endpoint
    - actionable-push-structs
  affects:
    - apps/api/models/delivery.go
    - apps/api/handlers/delivery.go
    - apps/api/handlers/orders.go
    - apps/api/services/nats.go
    - apps/api/services/push.go
    - apps/api/routes/routes.go
tech_stack:
  added:
    - github.com/gorilla/websocket v1.5.3
    - golang.org/x/time v0.15.0 (promoted to direct)
  patterns:
    - per-user rate.Limiter map with sync.RWMutex
    - write-channel pattern for concurrent-safe WebSocket writes
    - non-blocking NATS goroutine publish after DB save
    - GORM Preload chaining for nested relationships
key_files:
  created:
    - apps/api/migrations/20260405000000_add_delivery_partner_status_index.up.sql
    - apps/api/migrations/20260405000000_add_delivery_partner_status_index.down.sql
  modified:
    - apps/api/models/delivery.go
    - apps/api/handlers/delivery.go
    - apps/api/services/nats.go
    - apps/api/handlers/orders.go
    - apps/api/routes/routes.go
    - apps/api/services/push.go
    - apps/api/go.mod
    - apps/api/go.sum
decisions:
  - "sendFCMMessage extracted as internal PushService method so both SendPushNotification and SendActionablePush share the same FCM HTTP v1 send path"
  - "gorilla/websocket promoted to direct dependency in go.mod since it is used in handlers/orders.go"
  - "SQL migration files created under apps/api/migrations/ for documentation; project uses GORM AutoMigrate so the index must be applied manually or via a psql migration step before deploying"
  - "NATS publish in UpdateLocation queries deliveries WHERE status NOT IN (delivered, cancelled, failed) — returns early without error if no active delivery found"
metrics:
  duration: ~25min
  completed: 2026-04-05
  tasks_completed: 2
  files_changed: 8
  files_created: 2
---

# Phase 4 Plan 1: GPS + Push API Backend Summary

**One-liner:** Go API backend hardened with per-user GPS rate limiting, NATS location fan-out, WebSocket tracking endpoint, and actionable push notification structs for lock-screen vendor actions.

## What Was Built

### Task 1: DeliveryResponse serialization fix + rate limiter + NATS publish

**apps/api/models/delivery.go**
- Added 6 lat/lng fields to `DeliveryResponse`: `currentLatitude`, `currentLongitude`, `dropoffLatitude`, `dropoffLongitude`, `pickupLatitude`, `pickupLongitude`
- Updated `ToResponse()` to populate all 6 fields; `CurrentLatitude/CurrentLongitude` come from the preloaded `DeliveryPartner` relationship (nil-safe: defaults to 0.0 if not preloaded)

**apps/api/services/nats.go**
- Added `SubjectDeliveryLocation = "delivery.location"` constant to existing const block
- Full subject at runtime: `delivery.location.{deliveryID}` — uses core NATS (not JetStream), so messages are live-only and not stored

**apps/api/handlers/delivery.go**
- `DeliveryHandler` struct expanded with `locationRateLimit map[uuid.UUID]*rate.Limiter` and `locationRateMu sync.RWMutex`
- `NewDeliveryHandler()` initializes the map
- `getLocationLimiter(userID)` returns or creates a per-user `rate.Limiter` (1 token per 10 seconds)
- `UpdateLocation` now rejects excess requests with HTTP 429 before touching the DB
- After a successful DB save, a goroutine queries the active delivery for the partner and publishes to `delivery.location.{deliveryID}` via core NATS; returns early if no active delivery

**apps/api/migrations/**
- `20260405000000_add_delivery_partner_status_index.up.sql` — `CREATE INDEX IF NOT EXISTS idx_deliveries_partner_status ON deliveries (delivery_partner_id, status)`
- `20260405000000_add_delivery_partner_status_index.down.sql` — `DROP INDEX IF EXISTS idx_deliveries_partner_status`
- Note: project uses GORM AutoMigrate; this index must be applied manually (e.g., `psql -c`) before first deployment of this change

### Task 2: WebSocket tracking + gorilla/websocket + actionable push structs

**apps/api/handlers/orders.go**
- Added `gorilla/websocket` and `nats.go` imports plus package-level `wsUpgrader`
- `TrackOrder` now includes `.Preload("Delivery.DeliveryPartner")` so the initial HTTP response contains the driver's current coordinates
- `TrackOrderWS` handler added:
  - Verifies customer owns the order (`WHERE id = ? AND customer_id = ?`) before upgrading
  - Upgrades HTTP → WebSocket using `wsUpgrader`
  - Uses a buffered write channel (cap 32) so only one goroutine calls `conn.WriteMessage` (gorilla is not concurrent-write-safe)
  - Subscribes to `delivery.location.{deliveryID}` on core NATS; non-blocking send to write channel drops messages when full
  - `SetReadLimit(512)` caps inbound frame size; read pump blocks until client disconnects

**apps/api/routes/routes.go**
- Registered `orders.GET("/:id/track/ws", orderHandler.TrackOrderWS)` in the authenticated orders group

**apps/api/services/push.go**
- Added `Android *fcmAndroid` and `APNS *fcmAPNS` fields to `fcmMessageBody`
- Added structs: `fcmAndroid`, `fcmAndroidNotif`, `fcmAPNS`, `fcmAPNSPayload`, `fcmAPS`
- Extracted `sendFCMMessage(*fcmMessageBody)` as an internal `PushService` method (avoids code duplication)
- Added `SendActionablePush(userID, title, body, androidChannelID, iosCategory, data)` — looks up user FCM token, builds full platform block, calls `sendFCMMessage`

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical functionality] sendFCMMessage extracted as shared internal method**
- **Found during:** Task 2, Step 5
- **Issue:** Plan specified `sendFCMMessage` as the internal call target for `SendActionablePush`, but no such method existed — only `sendToToken` which builds its own `fcmMessage` without Android/APNS blocks
- **Fix:** Extracted `sendFCMMessage(*fcmMessageBody)` on `PushService` that accepts a fully-constructed body; updated `sendToToken` to use it; `SendActionablePush` calls it directly
- **Files modified:** `apps/api/services/push.go`
- **Commit:** 578c10d

**2. [Rule 3 - Blocking issue] golang.org/x/time promoted from indirect to direct**
- **Found during:** Task 1, Step 4
- **Issue:** `golang.org/x/time` was only an indirect dependency; needed as direct for `rate.Limiter` usage in handlers
- **Fix:** Added to direct require block in go.mod, removed from indirect block
- **Files modified:** `apps/api/go.mod`
- **Commit:** 03a655f

## Known Stubs

None — all lat/lng fields are wired to live DB data; no placeholder values.

## Threat Flags

All threats in the plan's threat register were mitigated as specified:

| Threat | File | Status |
|--------|------|--------|
| T-04-02 DoS GPS flooding | handlers/delivery.go | rate.Limiter 1/10s per user, HTTP 429 |
| T-04-03 Order ownership | handlers/orders.go TrackOrderWS | WHERE customer_id = ? before upgrade |
| T-04-05 WebSocket DoS | handlers/orders.go TrackOrderWS | SetReadLimit(512) + non-blocking write channel |

## Self-Check

Verified files exist:
- `apps/api/models/delivery.go` — contains `CurrentLatitude` in DeliveryResponse
- `apps/api/handlers/delivery.go` — contains `locationRateLimit`, `StatusTooManyRequests`, NATS publish goroutine
- `apps/api/services/nats.go` — contains `SubjectDeliveryLocation`
- `apps/api/handlers/orders.go` — contains `TrackOrderWS`, `writeCh`, `Preload("Delivery.DeliveryPartner")`
- `apps/api/routes/routes.go` — contains `track/ws` route
- `apps/api/services/push.go` — contains `fcmAndroid`, `SendActionablePush`
- `apps/api/migrations/20260405000000_add_delivery_partner_status_index.up.sql`

Verified commits exist:
- `03a655f` — Task 1
- `578c10d` — Task 2

Build: `cd apps/api && go build ./...` exits 0

## Self-Check: PASSED
