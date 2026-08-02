---
phase: quick-260802-kah
plan: 01
subsystem: mobile-realtime
tags: [websocket, backoff, react-native, notifications, vitest]

requires:
  - phase: quick-260726-892 (or equivalent #892 backoff work)
    provides: "socketReconnectDelayMs shared capped-exponential backoff helper in packages/mobile-shared/src/utils/socket-backoff.ts, already used by useOrderStatusWS, useOrderTrackingWS, vendor useLiveUpdates"
provides:
  - "useNotificationSocket onclose reconnects indefinitely with capped backoff instead of permanently giving up after 4 consecutive failures"
  - "shouldReconnectNotificationSocket() and notificationSocketReconnectDelayMs() exported pure seams, tested directly (no renderer/DOM)"
affects: [mobile-customer, mobile-vendor, notification-bell]

tech-stack:
  added: []
  patterns:
    - "Pure decision-seam extraction from a WebSocket onclose handler so tests exercise production logic directly, matching the shouldRetryNotificationQuery precedent already in this file"

key-files:
  created: []
  modified:
    - packages/mobile-shared/src/hooks/useNotifications.ts
    - packages/mobile-shared/src/__tests__/useNotifications.test.ts

key-decisions:
  - "Followed the plan's TDD flow exactly: failing tests first (RED, calling not-yet-exported functions), then the two-function extraction + onclose rewire (GREEN)."
  - "notificationSocketReconnectDelayMs is a thin named wrapper around the shared socketReconnectDelayMs, kept as its own export (not re-exporting the shared helper directly) so onclose and the tests import a name scoped to this hook, consistent with shouldRetryNotificationQuery's existing scoping."

patterns-established:
  - "Pure-function seam over renderer-based hook testing when a package has no DOM/rendering test infra (environment: 'node', no @testing-library/react, no react-test-renderer) — extract the decision logic the callback delegates to, test that directly."

requirements-completed: [GH-909]

duration: 3min
completed: 2026-08-02
---

# Quick Task 260802-kah: Fix #909 Notification Socket Retries Indefinitely Summary

**`useNotificationSocket`'s `onclose` now delegates to `shouldReconnectNotificationSocket()` and `notificationSocketReconnectDelayMs()`, retrying forever with the same capped exponential backoff (1s→2s→4s→8s→16s→30s) already applied to the other three sockets in #892 — the `MAX_WS_FAILURES = 4` permanent give-up is gone.**

## Performance

- **Duration:** 3 min
- **Started:** 2026-08-02T04:45:25Z
- **Completed:** 2026-08-02T04:48:38Z
- **Tasks:** 1 (TDD: RED + GREEN)
- **Files modified:** 2

## Accomplishments
- Removed the `MAX_WS_FAILURES = 4` / flat `RECONNECT_DELAY_MS = 3000` give-up path that caused the notification socket alone (of four sockets) to die permanently after ~12 seconds of failures, confirmed live on an Android emulator (`giving up after 7 consecutive failures — no further reconnects`).
- Extracted two exported pure functions — `shouldReconnectNotificationSocket(enabled, consecutiveFailures)` and `notificationSocketReconnectDelayMs(consecutiveFailures)` — that `onclose` itself calls, so the hook and its tests share one code path rather than the test re-deriving parallel logic.
- Added 6 new regression tests (11 total in the file) pinning both defects: reconnection continues past the old cap at 4, 7 (live-observed count), and 50 consecutive failures; and the exact delay curve (1000, 2000, 4000, 8000, 16000, 30000, 30000) asserted by literal value, not by re-deriving it from the same helper.
- REST polling (`useUnreadCount`, `useNotificationList`) untouched — remains the fallback while the socket is down or reconnecting.
- Updated JSDoc/comments on the hook and removed the stale "gives up" framing.

## Task Commits

TDD task, committed as RED → GREEN:

1. **Task 1 (RED): add failing regression tests** - `25f46ca8` (test)
2. **Task 1 (GREEN): extract pure seams, rewire onclose** - `7c5eefdc` (fix)

**Plan metadata:** committed separately by the orchestrator (docs commit not made by this executor per instructions)

## Files Created/Modified
- `packages/mobile-shared/src/hooks/useNotifications.ts` - Removed `MAX_WS_FAILURES`/`RECONNECT_DELAY_MS`; added `shouldReconnectNotificationSocket` and `notificationSocketReconnectDelayMs` exports; rewired `onclose` to use them; updated JSDoc on `useNotificationSocket` and the `onerror` log line
- `packages/mobile-shared/src/__tests__/useNotifications.test.ts` - Added `shouldReconnectNotificationSocket` describe block (4, 7, 50 failures + disabled case) and `notificationSocketReconnectDelayMs` describe block (literal curve values + monotonic growth)

## Decisions Made
- Followed the plan's exact TDD sequencing (RED commit, then GREEN commit) rather than a single combined commit, per this repo's TDD workflow.
- Kept `notificationSocketReconnectDelayMs` as a thin named wrapper (not a re-export) to match the existing `shouldRetryNotificationQuery` naming/scoping convention in this file.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

**Worktree/cwd resolution:** initial file reads and the first `Edit` call resolved against the shared checkout path (`/Users/Mahesh.Sangawar/personal/tesserix-new/Home-Chef-App/packages/...`) instead of this agent's worktree (`.../.claude/worktrees/agent-a948b7919bf3018c3/packages/...`). The `Edit` tool's worktree-isolation guard caught this before any write landed outside the worktree; re-read and re-edited using the worktree-rooted absolute path (`git rev-parse --show-toplevel`) for all subsequent operations. No files were modified outside the worktree.

**Base commit lag:** worktree HEAD was one commit behind the expected base (`5f720e70`, containing PLAN.md) at spawn time — `git merge-base` returned the parent commit instead. Per the `<worktree_branch_check>` step's explicit instruction, ran `git reset --hard 5f720e700050cd8faee79aff8b85684d6710e22a` (safe fast-forward, no divergent work existed) and re-verified merge-base equality before proceeding.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- All four real-time sockets (`useOrderStatusWS`, `useOrderTrackingWS`, vendor `useLiveUpdates`, `useNotificationSocket`) now share the same indefinite-retry-with-capped-backoff behavior from #892/#909.
- Not device-verified — the plan's test strategy was pure-function-only by design (no renderer/DOM infra in `packages/mobile-shared`); an on-device or emulator check that the notification bell recovers after a real socket drop (mirroring the original `[notif-ws] giving up after 7 consecutive failures` observation) would be the natural follow-up if desired, but is out of this plan's scope.

---
*Plan: quick-260802-kah*
*Completed: 2026-08-02*

## Self-Check: PASSED

- FOUND: packages/mobile-shared/src/hooks/useNotifications.ts
- FOUND: packages/mobile-shared/src/__tests__/useNotifications.test.ts
- FOUND: .planning/quick/260802-kah-fix-909-notification-socket-retries-inde/260802-kah-SUMMARY.md
- FOUND: commit 25f46ca8 (RED)
- FOUND: commit 7c5eefdc (GREEN)
