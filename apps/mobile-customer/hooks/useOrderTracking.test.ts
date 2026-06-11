import { describe, it, expect, jest } from '@jest/globals';

// Pure helper under test; mock lib/api so the import graph stays light.
jest.mock('../lib/api', () => ({ api: { get: jest.fn() } }));

import { trackingRefetchInterval } from './useOrderTracking';

describe('trackingRefetchInterval', () => {
  it('stops polling once the order is delivered or cancelled', () => {
    expect(trackingRefetchInterval('delivered')).toBe(false);
    expect(trackingRefetchInterval('cancelled')).toBe(false);
  });

  it('polls every 5s while the order is in flight', () => {
    expect(trackingRefetchInterval('pending')).toBe(5000);
    expect(trackingRefetchInterval('preparing')).toBe(5000);
    expect(trackingRefetchInterval('ready')).toBe(5000);
    expect(trackingRefetchInterval('picked_up')).toBe(5000);
  });

  it('polls when status is not yet known', () => {
    expect(trackingRefetchInterval(undefined)).toBe(5000);
  });
});
