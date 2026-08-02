// useNotifications.test.ts — covers #869's retry predicate: a 401 from the
// notification REST queries must never be retried by React Query (the axios
// interceptor's own single refresh+retry already ran), while genuine
// transient errors (network, 5xx, 403) keep the app's existing retry cap.

import { describe, expect, it } from 'vitest';

import {
  notificationSocketReconnectDelayMs,
  shouldReconnectNotificationSocket,
  shouldRetryNotificationQuery,
} from '../hooks/useNotifications';

describe('shouldRetryNotificationQuery', () => {
  it('never retries a 401 at failureCount 0', () => {
    const error = { response: { status: 401 } };
    expect(shouldRetryNotificationQuery(0, error)).toBe(false);
  });

  it('never retries a 401 at failureCount 1', () => {
    const error = { response: { status: 401 } };
    expect(shouldRetryNotificationQuery(1, error)).toBe(false);
  });

  it('retries a network error (no response) while under the cap', () => {
    const error = new Error('Network Error');
    expect(shouldRetryNotificationQuery(0, error)).toBe(true);
    expect(shouldRetryNotificationQuery(1, error)).toBe(true);
  });

  it('stops retrying a 500 once the cap is reached', () => {
    const error = { response: { status: 500 } };
    expect(shouldRetryNotificationQuery(2, error)).toBe(false);
  });

  it('retries a 403 while under the cap (only 401 is special-cased)', () => {
    const error = { response: { status: 403 } };
    expect(shouldRetryNotificationQuery(0, error)).toBe(true);
  });
});

// #909 — the notification socket used to give up permanently after
// MAX_WS_FAILURES=4 consecutive failures, unlike the other three sockets
// already fixed in #892. These pin both defects directly against the pure
// seams `onclose` itself calls, so a regression here would also break
// production, not just a parallel re-derivation of the same logic.
describe('shouldReconnectNotificationSocket', () => {
  it('keeps reconnecting past the old 4-failure cap', () => {
    expect(shouldReconnectNotificationSocket(true, 4)).toBe(true);
  });

  it('keeps reconnecting at the live-observed 7 consecutive failures', () => {
    expect(shouldReconnectNotificationSocket(true, 7)).toBe(true);
  });

  it('keeps reconnecting indefinitely — no cap at all, not just a higher one', () => {
    expect(shouldReconnectNotificationSocket(true, 50)).toBe(true);
  });

  it('stops reconnecting when disabled, regardless of failure count', () => {
    expect(shouldReconnectNotificationSocket(false, 7)).toBe(false);
  });
});

describe('notificationSocketReconnectDelayMs', () => {
  it('follows the shared capped-exponential backoff curve', () => {
    expect(notificationSocketReconnectDelayMs(1)).toBe(1000);
    expect(notificationSocketReconnectDelayMs(2)).toBe(2000);
    expect(notificationSocketReconnectDelayMs(3)).toBe(4000);
    // Already exceeds the old flat RECONNECT_DELAY_MS=3000 — this is the
    // value that would fail if the flat delay ever crept back in.
    expect(notificationSocketReconnectDelayMs(4)).toBe(8000);
    expect(notificationSocketReconnectDelayMs(5)).toBe(16000);
    expect(notificationSocketReconnectDelayMs(6)).toBe(30000);
    expect(notificationSocketReconnectDelayMs(20)).toBe(30000);
  });

  it('grows monotonically as consecutive failures increase', () => {
    const delays = [1, 2, 3, 4, 5, 6].map(notificationSocketReconnectDelayMs);
    for (let i = 1; i < delays.length; i++) {
      expect(delays[i]).toBeGreaterThanOrEqual(delays[i - 1] as number);
    }
  });
});
