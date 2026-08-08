// device-identity.test.ts — the per-install id that lets the API tell one
// signed-in device from another (#1164). Push delivery and the new-device
// security email both key on it, so it must be stable across restarts and
// distinct per install.

import { beforeEach, describe, expect, it, vi } from 'vitest';

const store = new Map<string, string>();
vi.mock('expo-secure-store', () => ({
  AFTER_FIRST_UNLOCK: 'afterFirstUnlock',
  getItemAsync: vi.fn(async (k: string) => store.get(k) ?? null),
  setItemAsync: vi.fn(async (k: string, v: string) => {
    store.set(k, v);
  }),
  deleteItemAsync: vi.fn(async (k: string) => {
    store.delete(k);
  }),
}));
vi.mock('expo-device', () => ({
  isDevice: true,
  modelName: 'iPhone 17 Pro',
  osName: 'iOS',
}));

import * as SecureStore from 'expo-secure-store';

import {
  DEVICE_ID_KEY,
  describeDevice,
  deviceIdSync,
  getDeviceId,
  resetDeviceIdCache,
} from '../auth/device-identity';

beforeEach(() => {
  store.clear();
  resetDeviceIdCache();
  vi.clearAllMocks();
});

describe('getDeviceId', () => {
  it('returns the same id on every call within a session', async () => {
    const first = await getDeviceId();
    const second = await getDeviceId();

    expect(first).toBe(second);
    expect(first).not.toHaveLength(0);
  });

  it('persists the id so a restart keeps the same device', async () => {
    const first = await getDeviceId();
    resetDeviceIdCache();

    expect(await getDeviceId()).toBe(first);
    expect(store.get(DEVICE_ID_KEY)).toBe(first);
  });

  it('mints a distinct id for a fresh install', async () => {
    const first = await getDeviceId();
    store.clear();
    resetDeviceIdCache();

    expect(await getDeviceId()).not.toBe(first);
  });

  // Unsigned simulator builds throw on every Keychain call. Sign-in must not
  // depend on the device id, so a throwing store degrades to a per-process id
  // rather than an exception.
  it('still returns an id when the keychain is unavailable', async () => {
    vi.mocked(SecureStore.getItemAsync).mockRejectedValue(new Error('no entitlement'));
    vi.mocked(SecureStore.setItemAsync).mockRejectedValue(new Error('no entitlement'));

    const id = await getDeviceId();

    expect(id).not.toHaveLength(0);
    expect(await getDeviceId()).toBe(id);
  });
});

// The axios request interceptor is synchronous, so it reads the warmed copy —
// and warms it for the next request rather than skipping the header forever.
describe('deviceIdSync', () => {
  it('is empty before the id is resolved and set afterwards', async () => {
    expect(deviceIdSync()).toBe('');

    const id = await getDeviceId();

    expect(deviceIdSync()).toBe(id);
  });

  it('kicks off resolution so a later request carries the header', async () => {
    expect(deviceIdSync()).toBe('');

    await vi.waitFor(() => expect(deviceIdSync()).not.toBe(''));
  });
});

describe('describeDevice', () => {
  it('reports what the security email prints', () => {
    expect(describeDevice()).toMatchObject({ platform: 'ios', label: 'iPhone 17 Pro' });
  });
});
