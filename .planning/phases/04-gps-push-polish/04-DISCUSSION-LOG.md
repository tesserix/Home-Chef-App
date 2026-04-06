# Phase 4: GPS, Push + Polish - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.

**Date:** 2026-04-06
**Phase:** 04-GPS, Push + Polish
**Areas discussed:** Background GPS, Push notifications, Order tracking integration, UX polish

---

## Background GPS (Driver)

- **Start condition:** On delivery accept (not on going online) — user-locked
- **Update rate:** You decide — Claude picks 15s recommended
- **Permission flow:** Rationale screen → system prompt — user-locked (App Store requirement)

## Push Notifications

- **Setup:** Direct FCM via getDevicePushTokenAsync — user-locked (matches Phase 1 D-09)
- **Vendor actions:** Yes, actionable lock-screen accept/reject — user-locked
- **Deep links:** Deep link to specific screen — user-locked

## Order Tracking Integration

- **Customer map:** Switch to WebSocket — user-locked, significant scope expansion
- **Backend changes:** Check and fix TrackingResponse serialization + add rate limiting + new WebSocket endpoint — user confirmed proceeding with scope expansion

### Scope Expansion Note
Phase 4 now includes Go API backend changes. User explicitly approved expanding scope to include:
- New WebSocket endpoint
- Serialization fix (DeliveryResponse lat/lng)
- NATS broadcasting
- Rate limiting middleware

## UX Polish

- **Scope:** All requirements UX-01..04 across all 3 apps — user-locked
- **Offline state:** You decide — Claude picks NetInfo + toast + cached data

---

## Claude's Discretion

- GPS update rate (recommend 15s)
- Offline error state pattern (recommend non-blocking toast)
- UX polish audit approach

## Deferred Ideas

All v2 differentiators (DIFF-01..09): live animated driver pin smoothing, shake to report, rating prompt, scheduled reminders, proof-of-delivery photo, reorder shortcut, smart ETA, notification prefs, chef availability widget.
