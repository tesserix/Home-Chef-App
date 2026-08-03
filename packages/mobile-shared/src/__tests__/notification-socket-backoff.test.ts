import { describe, expect, it } from 'vitest';

import {
  NOTIFICATION_SOCKET_STABLE_MS,
  nextConsecutiveFailures,
  notificationSocketReconnectDelayMs,
} from '../hooks/useNotifications';

describe('nextConsecutiveFailures (#928)', () => {
  it('counts a socket that never opened as a failure', () => {
    expect(nextConsecutiveFailures(0, null)).toBe(1);
    expect(nextConsecutiveFailures(3, null)).toBe(4);
  });

  it('counts a socket that opened and dropped immediately as a failure', () => {
    expect(nextConsecutiveFailures(0, 40)).toBe(1);
    expect(nextConsecutiveFailures(4, 900)).toBe(5);
  });

  it('clears the count once a connection survives the stability window', () => {
    expect(nextConsecutiveFailures(7, NOTIFICATION_SOCKET_STABLE_MS)).toBe(0);
    expect(nextConsecutiveFailures(7, 5 * 60_000)).toBe(0);
  });

  it('still counts a connection that drops just short of the window', () => {
    expect(nextConsecutiveFailures(1, NOTIFICATION_SOCKET_STABLE_MS - 1)).toBe(2);
  });

  it('escalates the delay through the observed 1/sec production flap', () => {
    // Reproduces #928: connect → error → close ~1s later, repeatedly. Under the
    // old onopen-resets-on-connect behaviour every iteration re-armed at 1s.
    let failures = 0;
    const delays: number[] = [];
    for (let i = 0; i < 6; i++) {
      failures = nextConsecutiveFailures(failures, 1_000);
      delays.push(notificationSocketReconnectDelayMs(failures));
    }
    expect(delays).toEqual([1_000, 2_000, 4_000, 8_000, 16_000, 30_000]);
  });

  it('re-arms a fast retry after a genuinely healthy session ends', () => {
    let failures = 0;
    for (let i = 0; i < 4; i++) failures = nextConsecutiveFailures(failures, 1_000);
    expect(notificationSocketReconnectDelayMs(failures)).toBe(8_000);

    failures = nextConsecutiveFailures(failures, 10 * 60_000);
    expect(notificationSocketReconnectDelayMs(failures)).toBe(1_000);
  });
});
