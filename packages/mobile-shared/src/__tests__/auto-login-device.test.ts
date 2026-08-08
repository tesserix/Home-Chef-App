// The BFF can only register this install — and warn about an unrecognised one
// — if sign-in says which install it is (#1164).

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

import { autoLogin } from '../auth/bff-session';
import { getDeviceId, resetDeviceIdCache } from '../auth/device-identity';

function okResponse() {
  return {
    ok: true,
    json: async () => ({
      session_token: 'sess',
      expires_at: 1,
      user: { id: 'u1', email: 'a@b.com', role: 'customer', pool: 'customer' },
    }),
  };
}

function sentBody(): Record<string, unknown> {
  const call = vi.mocked(global.fetch).mock.calls[0];
  return JSON.parse((call?.[1] as RequestInit).body as string);
}

beforeEach(() => {
  store.clear();
  resetDeviceIdCache();
  global.fetch = vi.fn().mockResolvedValue(okResponse()) as unknown as typeof fetch;
});

describe('autoLogin', () => {
  it('describes the install it is signing in from', async () => {
    await autoLogin('https://bff.test', 'id-token', 'HomeChef-Customer-x');

    const body = sentBody();
    expect(body.device_id).toBe(await getDeviceId());
    expect(body.platform).toBe('ios');
    expect(body.device_label).toBe('Test Device');
  });

  it('reuses the same device id on the next sign-in', async () => {
    await autoLogin('https://bff.test', 'id-token', 'HomeChef-Customer-x');
    const first = sentBody().device_id;
    resetDeviceIdCache();
    vi.mocked(global.fetch).mockClear();

    await autoLogin('https://bff.test', 'id-token', 'HomeChef-Customer-x');

    expect(sentBody().device_id).toBe(first);
  });
});
