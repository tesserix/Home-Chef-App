// device-token-path.test.ts — the FCM device-token endpoint must resolve
// correctly for EVERY app, whatever depth its API client is mounted at.
//
// The apps do not agree on where the client sits: the customer's
// EXPO_PUBLIC_API_URL ends at `/api` (its hooks call `/v1/...`), while the
// vendor's and the delivery app's already end at `/api/v1` (their hooks call
// `/chef/...`, `/driver/...` unversioned). registerDeviceToken hardcoded
// `/v1/profile/device-token`, so on vendor + delivery it resolved to
//
//     PUT /api/v1/v1/profile/device-token  ->  404
//
// and the token was never stored. Every chef push then logged "Push skipped:
// user <id> has no FCM token" and no vendor notification could ever arrive.
// Registration is best-effort, so nothing surfaced the 404.

import { describe, expect, it, vi } from 'vitest';
import type { AxiosInstance } from 'axios';

// usePushToken pulls in the Expo native modules at import time; stub them so the
// path logic can be exercised in the node test env (same approach as
// storage.test.ts).
vi.mock('expo-notifications', () => ({
  getPermissionsAsync: vi.fn(),
  requestPermissionsAsync: vi.fn(),
  getDevicePushTokenAsync: vi.fn(),
}));
vi.mock('expo-device', () => ({ isDevice: true }));

import { deviceTokenPath, registerDeviceToken } from '../hooks/usePushToken';

function clientWithBase(baseURL: string | undefined): AxiosInstance {
  return { defaults: { baseURL }, put: vi.fn().mockResolvedValue({}) } as unknown as AxiosInstance;
}

describe('deviceTokenPath', () => {
  it('adds the version prefix when the client stops at /api (customer)', () => {
    expect(deviceTokenPath(clientWithBase('https://fe3dr.com/api'))).toBe(
      '/v1/profile/device-token'
    );
  });

  it('omits the version prefix when the client already ends at /v1 (vendor)', () => {
    expect(deviceTokenPath(clientWithBase('https://vendors.fe3dr.com/api/v1'))).toBe(
      '/profile/device-token'
    );
  });

  it('omits the version prefix for the delivery app too', () => {
    expect(deviceTokenPath(clientWithBase('https://delivery.fe3dr.com/api/v1'))).toBe(
      '/profile/device-token'
    );
  });

  it('tolerates a trailing slash', () => {
    expect(deviceTokenPath(clientWithBase('https://vendors.fe3dr.com/api/v1/'))).toBe(
      '/profile/device-token'
    );
  });

  it('does not mistake a path merely containing v1 for a /v1 suffix', () => {
    // .../api/v10 and .../v1/orders must NOT be treated as ending in /v1.
    expect(deviceTokenPath(clientWithBase('https://x.test/api/v10'))).toBe(
      '/v1/profile/device-token'
    );
    expect(deviceTokenPath(clientWithBase('https://x.test/api/v1/orders'))).toBe(
      '/v1/profile/device-token'
    );
  });

  it('falls back to the versioned path when no baseURL is set', () => {
    expect(deviceTokenPath(clientWithBase(undefined))).toBe('/v1/profile/device-token');
  });
});

describe('registerDeviceToken', () => {
  // The regression: on the vendor client this used to PUT
  // /api/v1/v1/profile/device-token and 404.
  it('never doubles the version segment on a /api/v1 client', async () => {
    const client = clientWithBase('https://vendors.fe3dr.com/api/v1');

    await registerDeviceToken(client, 'fcm-token-abc');

    const [path] = (client.put as unknown as { mock: { calls: unknown[][] } }).mock.calls[0];
    expect(path).toBe('/profile/device-token');
    expect(String(path)).not.toContain('/v1/v1');
    // Resolved against the baseURL this is /api/v1/profile/device-token.
  });

  it('still resolves correctly on the customer client', async () => {
    const client = clientWithBase('https://fe3dr.com/api');

    await registerDeviceToken(client, 'fcm-token-abc');

    const [path] = (client.put as unknown as { mock: { calls: unknown[][] } }).mock.calls[0];
    expect(path).toBe('/v1/profile/device-token');
  });

  it('sends the token and skips auth-failure handling (best-effort call)', async () => {
    const client = clientWithBase('https://vendors.fe3dr.com/api/v1');

    await registerDeviceToken(client, 'fcm-token-xyz');

    const [, body, config] = (client.put as unknown as { mock: { calls: unknown[][] } }).mock
      .calls[0];
    expect(body).toEqual({ token: 'fcm-token-xyz' });
    // A 401 racing a cold-start session refresh must not bounce the user to login.
    expect(config).toMatchObject({ skipAuthFailure: true });
  });
});
