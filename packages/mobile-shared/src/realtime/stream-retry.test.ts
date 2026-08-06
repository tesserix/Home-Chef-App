import { describe, expect, it } from 'vitest';

import { streamRetryPlan } from './stream-retry';

describe('streamRetryPlan', () => {
  it('retries a transport failure on the backoff curve', () => {
    const plan = streamRetryPlan(0, 2);
    expect(plan.degrade).toBe(false);
    expect(plan.delayMs).toBeLessThanOrEqual(30_000);
  });

  it('retries a server fault, which may well be transient', () => {
    expect(streamRetryPlan(500, 1).degrade).toBe(false);
    expect(streamRetryPlan(503, 1).degrade).toBe(false);
  });

  // An order with no driver assigned yet answers 400 no_active_delivery on
  // every attempt. Retrying that on the transport curve is a hot loop against
  // an answer that cannot change until the kitchen dispatches.
  it('degrades and slows down when the server refused the stream', () => {
    for (const status of [400, 401, 403, 404]) {
      const plan = streamRetryPlan(status, 1);
      expect(plan.degrade).toBe(true);
      expect(plan.delayMs).toBe(30_000);
    }
  });

  it('holds the slow cadence however many times it is refused', () => {
    expect(streamRetryPlan(400, 50).delayMs).toBe(30_000);
  });
});
