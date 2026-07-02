// silent-refresh.test.ts — #428 increment 2: on a 401 the api client silently
// refreshes (re-mint from Firebase) and retries once, instead of logging the
// user out. Covers the refreshSession registry (single-flight) and the client's
// 401 → refresh → retry vs. clear behavior.

import { describe, it, expect, vi, beforeEach } from 'vitest';

// A CALLABLE axios instance mock so the interceptor's retry `instance(cfg)` works.
vi.mock('axios', () => {
  const instance: unknown = Object.assign(vi.fn(), {
    interceptors: {
      request: { use: vi.fn() },
      response: { use: vi.fn() },
    },
    post: vi.fn(),
  });
  return { default: { create: vi.fn(() => instance) } };
});

vi.mock('expo-secure-store', () => ({
  getItemAsync: vi.fn(),
  setItemAsync: vi.fn(),
  deleteItemAsync: vi.fn(),
  AFTER_FIRST_UNLOCK: 'AFTER_FIRST_UNLOCK',
}));

import axios from 'axios';
import * as SecureStore from 'expo-secure-store';
import { createApiClient } from '../api/client';
import { setSessionRefresher, refreshSession } from '../auth/bff-session';

type ErrHandler = (error: unknown) => Promise<unknown>;

// Build a client and return its response-interceptor rejection handler.
function makeClientAndHandler(onAuthFailure?: () => void) {
  const client = createApiClient({
    baseURL: 'https://api.test/v1',
    getToken: () => 'old-token',
    onAuthFailure,
  });
  const useMock = client.interceptors.response.use as unknown as {
    mock: { calls: unknown[][] };
  };
  const lastCall = useMock.mock.calls[useMock.mock.calls.length - 1];
  return { client, onRejected: lastCall[1] as ErrHandler };
}

describe('refreshSession registry (#428)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setSessionRefresher(null);
  });

  it('returns null when no refresher is registered', async () => {
    expect(await refreshSession()).toBeNull();
  });

  it('returns the token from the registered refresher', async () => {
    setSessionRefresher(async () => 'fresh-token');
    expect(await refreshSession()).toBe('fresh-token');
  });

  it('is single-flight: concurrent 401s mint at most one new session', async () => {
    let calls = 0;
    setSessionRefresher(async () => {
      calls++;
      await new Promise((r) => setTimeout(r, 10));
      return 'tok';
    });
    const results = await Promise.all([refreshSession(), refreshSession(), refreshSession()]);
    expect(results).toEqual(['tok', 'tok', 'tok']);
    expect(calls).toBe(1);
  });

  it('returns null (does not throw) when the refresher fails', async () => {
    setSessionRefresher(async () => {
      throw new Error('firebase unavailable');
    });
    expect(await refreshSession()).toBeNull();
  });
});

describe('api client 401 handling (#428)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setSessionRefresher(null);
  });

  it('refreshes + retries the request once with the new token (no logout)', async () => {
    setSessionRefresher(async () => 'new-token');
    const onAuthFailure = vi.fn();
    const { client, onRejected } = makeClientAndHandler(onAuthFailure);
    const cfg = { headers: {} as Record<string, string> };

    await onRejected({ response: { status: 401 }, config: cfg });

    expect((cfg as { _retried?: boolean })._retried).toBe(true);
    expect(cfg.headers.Authorization).toBe('Bearer new-token');
    expect(client).toHaveBeenCalledWith(cfg); // retried
    expect(onAuthFailure).not.toHaveBeenCalled();
  });

  it('clears the session + calls onAuthFailure only when refresh yields nothing', async () => {
    setSessionRefresher(null); // no refresher → refresh returns null
    const onAuthFailure = vi.fn();
    const { onRejected } = makeClientAndHandler(onAuthFailure);

    await expect(
      onRejected({ response: { status: 401 }, config: { headers: {} } }),
    ).rejects.toBeDefined();

    expect(vi.mocked(SecureStore.deleteItemAsync)).toHaveBeenCalled();
    expect(onAuthFailure).toHaveBeenCalledTimes(1);
  });

  it('does not retry a request already marked _retried', async () => {
    setSessionRefresher(async () => 'new-token');
    const onAuthFailure = vi.fn();
    const { onRejected } = makeClientAndHandler(onAuthFailure);

    await expect(
      onRejected({ response: { status: 401 }, config: { headers: {}, _retried: true } }),
    ).rejects.toBeDefined();
    expect(onAuthFailure).toHaveBeenCalledTimes(1); // fell through to logout
  });

  it('skips refresh + logout for skipAuthFailure requests', async () => {
    setSessionRefresher(async () => 'new-token');
    const onAuthFailure = vi.fn();
    const { onRejected } = makeClientAndHandler(onAuthFailure);

    await expect(
      onRejected({ response: { status: 401 }, config: { headers: {}, skipAuthFailure: true } }),
    ).rejects.toBeDefined();
    expect(onAuthFailure).not.toHaveBeenCalled();
  });
});
