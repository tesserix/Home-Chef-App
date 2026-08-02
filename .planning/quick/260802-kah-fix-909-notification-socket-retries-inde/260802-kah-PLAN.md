---
phase: quick-260802-kah
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - packages/mobile-shared/src/hooks/useNotifications.ts
  - packages/mobile-shared/src/__tests__/useNotifications.test.ts
autonomous: true
requirements: [GH-909]

must_haves:
  truths:
    - "The notification socket keeps attempting to reconnect past 4 consecutive failures — it never permanently gives up for the rest of the session."
    - "Each reconnect delay is computed from the shared capped-exponential backoff curve (1s, 2s, 4s, 8s, 16s, capped 30s), not a flat 3000ms."
    - "REST polling (useUnreadCount / useNotificationList) remains untouched as the fallback while the socket is down — this change only removes the permanent give-up, it does not remove or alter the fallback."
    - "Code comments on the hook describe indefinite-retry-with-backoff, not the old 4-failure give-up (a stale comment asserting the old behavior is exactly the trap called out in verified_findings)."
  artifacts:
    - path: "packages/mobile-shared/src/hooks/useNotifications.ts"
      provides: "onclose delegates to two exported pure seams — shouldReconnectNotificationSocket() and notificationSocketReconnectDelayMs() — so the hook and its tests exercise the same decision logic; MAX_WS_FAILURES/RECONNECT_DELAY_MS removed"
      contains: "shouldReconnectNotificationSocket"
    - path: "packages/mobile-shared/src/__tests__/useNotifications.test.ts"
      provides: "Pure-function regression coverage (no renderer, no DOM, no new dependency) pinning both defects: reconnection past the old 4-failure cap (asserted at 4, 7, 50), and delay growth+cap per the shared backoff curve"
  key_links:
    - from: "packages/mobile-shared/src/hooks/useNotifications.ts"
      to: "packages/mobile-shared/src/utils/socket-backoff.ts"
      via: "import { socketReconnectDelayMs } from '../utils/socket-backoff'"
      pattern: "socketReconnectDelayMs\\("
---

<objective>
Fix GitHub #909: `useNotificationSocket` in `packages/mobile-shared/src/hooks/useNotifications.ts` gives up permanently after `MAX_WS_FAILURES = 4` consecutive failures, reconnecting on a flat `RECONNECT_DELAY_MS = 3000` until then. #892 added the shared `socketReconnectDelayMs` capped-backoff helper and applied it to `useOrderStatusWS`, `useOrderTrackingWS`, and vendor `useLiveUpdates` — it never touched this fourth socket, which is why it alone still dies after ~12 seconds of failures and stays dead for the rest of the session (confirmed live on an Android emulator: `giving up after 7 consecutive failures` while the sibling order socket kept reconnecting).

Purpose: bring the notification socket in line with the other three — retry indefinitely with the same capped exponential backoff (1s → 2s → 4s → 8s → 16s → 30s ceiling) — while leaving its existing REST-polling fallback (`useUnreadCount` / `useNotificationList`) exactly as-is. Unlike `useOrderStatusWS` (no fallback), this socket already degrades gracefully today; the bug is that the degradation is permanent instead of recoverable.
Output: `useNotificationSocket`'s `onclose` delegates to two exported pure functions — `shouldReconnectNotificationSocket()` and `notificationSocketReconnectDelayMs()` — so it reconnects forever with growing backoff, and a new test in the existing `useNotifications.test.ts` calls those same two functions directly (no rendering, no new dependency) to pin both defects: no permanent give-up, and delay that actually grows and caps.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@packages/mobile-shared/src/utils/socket-backoff.ts
@packages/mobile-shared/src/__tests__/socket-backoff.test.ts
@packages/mobile-shared/src/__tests__/useNotifications.test.ts
@apps/mobile-customer/hooks/useOrderStatusWS.ts

<interfaces>
<!-- Existing exports this plan builds on. No further codebase exploration needed. -->

From packages/mobile-shared/src/utils/socket-backoff.ts (already used by the other three sockets, already unit-tested):
```typescript
export function socketReconnectDelayMs(consecutiveFailures: number): number;
// Wraps outbox's backoffMs: min(30_000, 1_000 * 2 ** max(0, attempts - 1))
// consecutiveFailures=1 -> 1000, 2 -> 2000, 3 -> 4000, 4 -> 8000, 5 -> 16000, 6+ -> 30000
// Never 0/negative, including for consecutiveFailures <= 0.
```
Import path from within `packages/mobile-shared/src/hooks/useNotifications.ts`: `../utils/socket-backoff`.

Current buggy code in `packages/mobile-shared/src/hooks/useNotifications.ts` (lines 180-257), to be replaced:
```typescript
const MAX_WS_FAILURES = 4;
const RECONNECT_DELAY_MS = 3000;

export function useNotificationSocket(opts: {
  apiBaseUrl: string | undefined;
  getToken: () => string | null | undefined;
  enabled?: boolean;
}): void {
  const { apiBaseUrl, getToken, enabled = true } = opts;
  const qc = useQueryClient();
  const wsRef = useRef<WebSocket | null>(null);
  const failures = useRef(0);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const connect = useCallback(() => {
    // ... token/url setup unchanged ...
    ws.onopen = () => { failures.current = 0; /* log */ };
    ws.onmessage = () => { failures.current = 0; /* invalidate queries */ };
    ws.onerror = () => {
      failures.current += 1;
      console.warn(`[notif-ws] error (${failures.current}/${MAX_WS_FAILURES} failures)`);
    };
    ws.onclose = () => {
      wsRef.current = null;
      if (!enabled) return;
      if (failures.current >= MAX_WS_FAILURES) {
        console.error(`[notif-ws] giving up after ${failures.current} consecutive failures — no further reconnects, REST polling remains the fallback`);
        return;
      }
      console.warn(`[notif-ws] closed, reconnecting in ${RECONNECT_DELAY_MS}ms`);
      reconnectTimer.current = setTimeout(connect, RECONNECT_DELAY_MS);
    };
  }, [apiBaseUrl, getToken, enabled, qc]);

  useEffect(() => { connect(); return () => { /* cleanup */ }; }, [connect]);
}
```

Reference — `apps/mobile-customer/hooks/useOrderStatusWS.ts`'s already-fixed `ws.onclose` (the *decision shape* to mirror — unconditional reconnect while enabled, delay from the shared curve — minus its AppState foreground-reconnect addition which is out of scope here):
```typescript
ws.onclose = () => {
  if (!enabled) return;
  const delay = socketReconnectDelayMs(failureCount.current);
  console.warn(`[order-ws] closed, reconnecting in ${delay}ms`);
  reconnectTimer.current = setTimeout(connect, delay);
};
```
Its accompanying comment block (mirror the *framing*, adapted to the fact this socket — unlike order-ws — does have a REST fallback):
```typescript
// This stream has no fallback (unlike order-tracking's polling or the
// vendor app's SSE), so giving up permanently here is the worst case of
// #892 — a customer could go a whole session with no live updates.
// Retry indefinitely with backoff instead; there is no cap to hit.
```

**Test strategy — pure functions only, no renderer.** `packages/mobile-shared` has no DOM/rendering test infrastructure: `vitest.config.ts` sets `environment: 'node'`, and `@testing-library/react` / `react-test-renderer` are NOT in the package's `devDependencies` (only `@hookform/resolvers`, `@types/react`, `axios`, `react-hook-form`, `typescript`, `vitest`, `zod`) or `peerDependencies`. They are resolvable today purely because pnpm's `node-linker=hoisted` flattens the whole monorepo's `node_modules` — but `pnpm-lock.yaml` cannot currently be regenerated (`apps/web` pins `@tesserix/otto-widget@^0.5.4`, which no longer resolves on npmjs, #897), so nothing not already declared for this package can be honestly added or relied on. This is the same trap already reworked out of a `#881` plan today for `react-test-renderer`. **Do not use `renderHook`, `act`, `QueryClientProvider`, or a mock `WebSocket`/jsdom override.**

The existing precedent in `packages/mobile-shared/src/__tests__/useNotifications.test.ts` is exactly the right pattern: `shouldRetryNotificationQuery` is a pure predicate exported straight from the hook module and tested with plain `describe`/`it`/`expect` — no rendering, no DOM. This plan extracts the same kind of seam from `onclose`'s decision so the new tests can call the *actual* production logic without a renderer, and so `onclose` and the tests share one code path instead of the test re-deriving parallel logic.
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Fix indefinite backoff retry via pure seam + regression test for #909</name>
  <files>packages/mobile-shared/src/hooks/useNotifications.ts, packages/mobile-shared/src/__tests__/useNotifications.test.ts</files>
  <behavior>
    New `describe('shouldReconnectNotificationSocket', ...)` block in `useNotifications.test.ts`:
    - `shouldReconnectNotificationSocket(true, 4)` -> `true` — past the old `MAX_WS_FAILURES = 4` cap, the exact boundary the old code gave up at.
    - `shouldReconnectNotificationSocket(true, 7)` -> `true` — matches the live-observed failure count from the verified findings (`giving up after 7 consecutive failures`).
    - `shouldReconnectNotificationSocket(true, 50)` -> `true` — proves there is no cap at all, not just a higher one.
    - `shouldReconnectNotificationSocket(false, 7)` -> `false` — `enabled=false` (hook torn down / disabled) must still stop reconnecting regardless of failure count; this is the one condition that legitimately halts reconnection.

    New `describe('notificationSocketReconnectDelayMs', ...)` block:
    - Hardcode the expected curve directly in the test (do NOT derive expectations by calling `socketReconnectDelayMs` a second time inside the assertion — that would make the test tautological against its own implementation): `notificationSocketReconnectDelayMs(1)` -> `1000`, `(2)` -> `2000`, `(3)` -> `4000`, `(4)` -> `8000` (already exceeds the old flat `RECONNECT_DELAY_MS = 3000` — this is the value that would fail if the flat delay ever crept back in), `(5)` -> `16000`, `(6)` -> `30000`, `(20)` -> `30000` (cap holds well past the curve's nominal end).
    - One monotonic-growth assertion across `[1,2,3,4,5,6]` mirroring `socket-backoff.test.ts`'s existing style, to document intent even though the hardcoded values above already prove it.
  </behavior>
  <action>
    In `packages/mobile-shared/src/hooks/useNotifications.ts`: add `import { socketReconnectDelayMs } from '../utils/socket-backoff';` near the top with the other imports. Delete the `MAX_WS_FAILURES` and `RECONNECT_DELAY_MS` constants (lines 180-181). Add two new exported pure functions in their place:
    `export function shouldReconnectNotificationSocket(enabled: boolean, _consecutiveFailures: number): boolean` — returns `enabled` unconditionally (parameter prefixed `_consecutiveFailures` per this repo's `noUnusedParameters` convention since it is intentionally unused — the whole point of the function is that failure count no longer gates reconnection; document that in a one-line comment directly above it, e.g. "Accepts consecutiveFailures so the signature itself documents that #909 removed the cap — it never gates on failure count, only on enabled.");
    `export function notificationSocketReconnectDelayMs(consecutiveFailures: number): number` — returns `socketReconnectDelayMs(consecutiveFailures)` (thin named wrapper, kept as its own export so `onclose` and the tests import a name scoped to this hook, consistent with how `shouldRetryNotificationQuery` is already named/scoped rather than imported generically).
    In `ws.onerror`, drop the `/${MAX_WS_FAILURES}` from the log line (just `error (${failures.current} consecutive failures)`) since there is no longer a cap to report against. In `ws.onclose`, delete the `if (failures.current >= MAX_WS_FAILURES) { ...; return; }` give-up branch entirely; replace the enabled check and delay/schedule logic with: `if (!shouldReconnectNotificationSocket(enabled, failures.current)) return;` then `const delay = notificationSocketReconnectDelayMs(failures.current);`, log `[notif-ws] closed, reconnecting in ${delay}ms`, and `reconnectTimer.current = setTimeout(connect, delay);`. Update the JSDoc above `useNotificationSocket` (currently: "The REST queries above remain the fallback if the socket can't connect.") to state the retry is indefinite with capped backoff, and that REST polling now covers the gap while the socket is *reconnecting* rather than being needed as a permanent replacement after a give-up — do not change anything about the REST hooks themselves, only this comment and the functions/branch above. Note `Fixes #909; mirrors #892's fix already applied to the other three sockets` in the comment near the two new functions.

    In `packages/mobile-shared/src/__tests__/useNotifications.test.ts`: add `shouldReconnectNotificationSocket` and `notificationSocketReconnectDelayMs` to the existing `import { shouldRetryNotificationQuery } from '../hooks/useNotifications';` line (single merged import, do not add a second import line or any new package import — no `@testing-library/react`, no `react-test-renderer`, no `@tanstack/react-query`, no `vi`/fake timers/mock `WebSocket` needed since nothing is rendered). Add the two new `describe` blocks per the `<behavior>` block above, following the same flat `describe`/`it`/`expect` style already used for `shouldRetryNotificationQuery` in this file and for `socketReconnectDelayMs` in `socket-backoff.test.ts`.
  </action>
  <verify>
    <automated>cd packages/mobile-shared && npx vitest run src/__tests__/useNotifications.test.ts</automated>
  </verify>
  <done>New `shouldReconnectNotificationSocket` and `notificationSocketReconnectDelayMs` describe blocks pass; `shouldRetryNotificationQuery` tests in the same file still pass unmodified; `useNotifications.ts` no longer references `MAX_WS_FAILURES` or a flat `RECONNECT_DELAY_MS`; `grep -c MAX_WS_FAILURES packages/mobile-shared/src/hooks/useNotifications.ts` returns 0; `packages/mobile-shared/package.json` and `pnpm-lock.yaml` are untouched (no new dependency).</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|--------------|
| None new | Pure client-side reconnect-scheduling change to an already-authenticated WebSocket (Bearer token unchanged) plus two pure functions. No new network endpoint, no new input parsing, no new stored data. |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|------------------|
| T-909-01 | Denial of Service | `useNotificationSocket` reconnect loop | accept | Backoff is capped at 30s (same curve already proven safe for the other three sockets in #892) — indefinite retry cannot become a hot loop; each attempt is a single authenticated WS handshake, not amplifiable by an attacker. |
| T-909-02 | Tampering | npm install | accept | No new dependency added or relied upon anywhere in this plan, including in tests (`packages/mobile-shared/package.json` and `pnpm-lock.yaml` untouched) — the lockfile cannot currently be regenerated (#897), so this is enforced by construction, not just by intent. |

No package-manager installs in this plan — Package Legitimacy Gate not applicable.
</threat_model>

<verification>
1. `cd packages/mobile-shared && npx vitest run` — new `shouldReconnectNotificationSocket`/`notificationSocketReconnectDelayMs` tests pass; total failures do not exceed the 4 pre-existing failures (`auth-screens` x3, `resolve-auth-error` x1); total pass count is >= 120 + new assertions.
2. `cd apps/mobile-customer && npx tsc --noEmit` — error count does not exceed the pre-existing 1 (`lib/payment.ts`); re-measure rather than assume, per the stated constraint.
3. `grep -c MAX_WS_FAILURES packages/mobile-shared/src/hooks/useNotifications.ts` returns 0; `grep -c socketReconnectDelayMs packages/mobile-shared/src/hooks/useNotifications.ts` returns >= 1 (import + call site inside `notificationSocketReconnectDelayMs`).
4. `git diff --stat -- packages/mobile-shared/package.json pnpm-lock.yaml` is empty — confirms no dependency was added.
5. Manual read-through of the updated JSDoc/comments on `useNotificationSocket` and its `onclose` handler: no remaining text asserting the socket gives up after N failures.
</verification>

<success_criteria>
- `useNotificationSocket`'s `onclose` delegates to `shouldReconnectNotificationSocket()` and `notificationSocketReconnectDelayMs()`; the `MAX_WS_FAILURES` give-up path is fully removed.
- REST polling (`useUnreadCount`, `useNotificationList`) is untouched — still the fallback while the socket is down or reconnecting, exactly as before.
- New automated tests call the same two exported pure functions the hook itself calls (no renderer, no DOM, no new dependency) and pin both defects: reconnection past the old cap (asserted at 4, 7, 50 consecutive failures) and delay growth+cap per the shared curve.
- vitest failure count in `packages/mobile-shared` does not exceed the pre-existing 4; `apps/mobile-customer` tsc error count does not exceed the pre-existing 1.
- No dependency added, no lockfile touched, no `prettier` run, ROADMAP.md untouched, the other three already-fixed sockets and the WS auth gate (#869) untouched.
</success_criteria>

<output>
Create `.planning/quick/260802-kah-fix-909-notification-socket-retries-inde/260802-kah-SUMMARY.md` when done
</output>
