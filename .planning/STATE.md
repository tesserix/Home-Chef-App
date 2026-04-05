# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-04-05)

**Core value:** Customers can browse home chefs, order food, and track delivery on their phones — while chefs manage orders and drivers navigate deliveries natively.
**Current focus:** Phase 1 — Foundation + Auth

## Current Position

Phase: 1 of 4 (Foundation + Auth)
Plan: 0 of TBD in current phase
Status: Ready to plan
Last activity: 2026-04-05 — Roadmap created, phases defined, ready to plan Phase 1

Progress: [░░░░░░░░░░] 0%

## Performance Metrics

**Velocity:**
- Total plans completed: 0
- Average duration: —
- Total execution time: 0 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| - | - | - | - |

**Recent Trend:**
- Last 5 plans: —
- Trend: —

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Phase 1: Use raw FCM tokens (`getDevicePushTokenAsync`) not Expo Push Tokens — existing `push.go` calls FCM directly and the two token types are incompatible
- Phase 1: Must verify `@tesserix/native` peer deps and current Expo SDK stable version (52 vs 53) before pinning versions — do not scaffold until confirmed
- Phase 1: Each app needs `metro.config.js` with `watchFolders` and `resolver.nodeModulesPaths` — pnpm symlinks are invisible to Metro by default
- Phase 1: GPS location watcher must use `distanceInterval: 20–50m` and `timeInterval: 5–10s` to avoid flooding `db-f1-micro` (architectural constraint)
- Phase 4: Background location permission must be gated on driver going online, not on first launch — App Store rejection risk otherwise

### Pending Todos

None yet.

### Blockers/Concerns

- Confirm Go API has `PATCH /v1/delivery/:id/location` endpoint before Phase 4 GPS work begins
- Confirm whether real-time driver location for Customer map is REST polling or WebSocket (v1 fallback: 5-second polling)
- Verify EAS Build free tier build minute limits for 3 apps before Phase 1 ends

## Session Continuity

Last session: 2026-04-05
Stopped at: Roadmap written, STATE.md initialized, REQUIREMENTS.md traceability updated
Resume file: None
