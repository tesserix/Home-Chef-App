// useNotifications.test.ts — covers #869's retry predicate: a 401 from the
// notification REST queries must never be retried by React Query (the axios
// interceptor's own single refresh+retry already ran), while genuine
// transient errors (network, 5xx, 403) keep the app's existing retry cap.

import { describe, expect, it } from 'vitest';

import { shouldRetryNotificationQuery } from '../hooks/useNotifications';

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
