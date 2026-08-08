// Every authenticated request names the install it comes from, so the device
// registry stays current and the device-token PUT lands on this device's row
// rather than replacing the account's only one (#1164).

import { beforeEach, describe, expect, it, vi } from 'vitest';
import axios from 'axios';

vi.mock('axios', () => ({
  default: {
    create: vi.fn(() => ({
      interceptors: { request: { use: vi.fn() }, response: { use: vi.fn() } },
      post: vi.fn(),
    })),
  },
}));

const store = new Map<string, string>();
vi.mock('expo-secure-store', () => ({
  AFTER_FIRST_UNLOCK: 'AFTER_FIRST_UNLOCK',
  getItemAsync: vi.fn(async (k: string) => store.get(k) ?? null),
  setItemAsync: vi.fn(async (k: string, v: string) => {
    store.set(k, v);
  }),
  deleteItemAsync: vi.fn(),
}));

import { createApiClient } from '../api/client';
import { getDeviceId, resetDeviceIdCache } from '../auth/device-identity';

type Cfg = { headers: Record<string, string> };

function requestInterceptor(): (c: Cfg) => Cfg {
  let captured: ((c: Cfg) => Cfg) | null = null;
  vi.mocked(axios.create).mockReturnValueOnce({
    interceptors: {
      request: {
        use: vi.fn((fn: (c: Cfg) => Cfg) => {
          captured = fn;
        }),
      },
      response: { use: vi.fn() },
    },
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
  } as any);
  createApiClient({ baseURL: 'https://api.test.com/api', getToken: () => null });
  return captured!;
}

beforeEach(() => {
  store.clear();
  resetDeviceIdCache();
  vi.clearAllMocks();
});

describe('X-Device-Id', () => {
  it('rides on every request once the id is known', async () => {
    const id = await getDeviceId();

    const config = requestInterceptor()({ headers: {} });

    expect(config.headers['X-Device-Id']).toBe(id);
  });

  // The interceptor is synchronous and cold start may beat the keychain read.
  // A missing id must send no header rather than an empty one the API would
  // have to special-case.
  it('is omitted while the id is still unresolved', () => {
    const config = requestInterceptor()({ headers: {} });

    expect(config.headers).not.toHaveProperty('X-Device-Id');
  });
});
