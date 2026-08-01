import { describe, expect, it } from 'vitest';

import { socketReconnectDelayMs } from '../utils/socket-backoff';

describe('socketReconnectDelayMs', () => {
  it('returns a sane, non-zero delay for the first attempt', () => {
    const delay = socketReconnectDelayMs(1);
    expect(delay).toBeGreaterThan(0);
    expect(delay).toBeLessThanOrEqual(3000);
  });

  it('grows monotonically as consecutive failures increase', () => {
    const delays = [1, 2, 3, 4, 5].map(socketReconnectDelayMs);
    for (let i = 1; i < delays.length; i++) {
      expect(delays[i]).toBeGreaterThanOrEqual(delays[i - 1] as number);
    }
  });

  it('respects the ~30s ceiling and never exceeds it, however many failures pile up', () => {
    expect(socketReconnectDelayMs(6)).toBe(30_000);
    expect(socketReconnectDelayMs(20)).toBe(30_000);
    expect(socketReconnectDelayMs(1000)).toBe(30_000);
  });

  it('never returns 0 or a negative delay, including for non-positive input', () => {
    expect(socketReconnectDelayMs(0)).toBeGreaterThan(0);
    expect(socketReconnectDelayMs(-5)).toBeGreaterThan(0);
  });
});
