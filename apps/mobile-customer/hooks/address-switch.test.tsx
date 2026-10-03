import React from 'react';
import { act, create } from 'react-test-renderer';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { afterEach, beforeEach, expect, it, jest } from '@jest/globals';

jest.mock('react-native-css-interop/jsx-runtime', () =>
  require('react/jsx-runtime'),
);
jest.mock('react-native-css-interop/jsx-dev-runtime', () =>
  require('react/jsx-dev-runtime'),
);
jest.mock('../lib/api', () => ({ api: { get: jest.fn(), put: jest.fn() } }));
jest.mock('../store/auth-store', () => ({
  useAuthStore: (select: any) => select({ isAuthenticated: true }),
}));
import { api } from '../lib/api';
import { useAddresses, useSetDefaultAddress } from './useAddresses';
import { useChefs } from './useChefs';
import type { AxiosRequestConfig } from 'axios';
import type { Address } from '../types/customer';

(globalThis as any).IS_REACT_ACT_ENVIRONMENT = true;
const work: Address = {
  id: 'work',
  label: 'Work',
  addressLine1: '12 Queen Street',
  city: 'Auckland',
  state: 'Auckland',
  pincode: '1010',
  country: 'NZ',
  latitude: -36.844,
  longitude: 174.767,
  isDefault: true,
};
const home: Address = {
  id: 'home',
  label: 'Home',
  addressLine1: 'Domlur',
  city: 'Bengaluru',
  state: 'Karnataka',
  pincode: '560071',
  country: 'IN',
  latitude: 12.9611,
  longitude: 77.6387,
  isDefault: false,
};
const get = api.get as jest.Mock<
  (path: string, config?: AxiosRequestConfig) => Promise<any>
>;
let client: QueryClient;
let screen: ReturnType<typeof create>;
let select: ReturnType<typeof useSetDefaultAddress>;
let active: Address | undefined;
let kitchens: string[];
function Consumer() {
  const addresses = useAddresses();
  active = addresses.data?.data.find((a) => a.isDefault);
  select = useSetDefaultAddress();
  const chefs = useChefs({
    lat: active?.latitude,
    lng: active?.longitude,
    state: active?.state,
  });
  kitchens = chefs.data?.data.map((c) => c.name) ?? [];
  return null;
}
async function flush() {
  await act(async () => {
    await jest.runAllTimersAsync();
  });
}
beforeEach(async () => {
  jest.useFakeTimers();
  jest.clearAllMocks();
  client = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: Infinity },
      mutations: { retry: false },
    },
  });
  client.setQueryData(['addresses'], { data: [work, home] });
  get.mockImplementation(async (path, config) => {
    if (path === '/v1/addresses')
      return {
        data: [work, home].map((a) => ({
          ...a,
          line1: a.addressLine1,
          postalCode: a.pincode,
        })),
      };
    return {
      data: {
        data: [
          {
            id: 'chef',
            businessName:
              config?.params?.state === 'Karnataka'
                ? 'Bengaluru Kitchen'
                : 'NZ Kitchen',
          },
        ],
      },
    };
  });
  await act(async () => {
    screen = create(
      <QueryClientProvider client={client}>
        <Consumer />
      </QueryClientProvider>,
    );
  });
  await flush();
});
afterEach(async () => {
  await act(async () => screen.unmount());
  client.clear();
  jest.useRealTimers();
});

it('switches the address and nearby kitchens while the save is pending', async () => {
  jest.mocked(api.put).mockImplementation(() => new Promise(() => {}));
  expect(kitchens).toEqual(['NZ Kitchen']);
  await act(async () => {
    select.mutate(home);
  });
  await flush();
  expect(active?.id).toBe('home');
  expect(kitchens).toEqual(['Bengaluru Kitchen']);
});

it('restores the previous address and its kitchens when saving fails', async () => {
  let reject!: (error: Error) => void;
  jest.mocked(api.put).mockImplementation(
    () =>
      new Promise((_, fail) => {
        reject = fail;
      }),
  );
  await act(async () => {
    select.mutate(home);
  });
  await flush();
  expect(active?.id).toBe('home');
  await act(async () => {
    reject(new Error('offline'));
  });
  await flush();
  expect(active?.id).toBe('work');
  expect(kitchens).toEqual(['NZ Kitchen']);
});

it('does not let an older address response undo a confirmed switch', async () => {
  let finishOldRead!: (response: any) => void;
  get.mockImplementation((path, config) => {
    if (path === '/v1/addresses')
      return new Promise((resolve) => {
        finishOldRead = resolve;
      });
    return Promise.resolve({
      data: {
        data: [
          {
            id: 'chef',
            businessName:
              config?.params?.state === 'Karnataka'
                ? 'Bengaluru Kitchen'
                : 'NZ Kitchen',
          },
        ],
      },
    });
  });
  await act(async () => {
    void client.invalidateQueries({ queryKey: ['addresses'] });
  });
  jest.mocked(api.put).mockResolvedValue({
    data: {
      ...home,
      line1: home.addressLine1,
      postalCode: home.pincode,
      isDefault: true,
    },
  });
  await act(async () => {
    select.mutate(home);
  });
  await flush();
  await act(async () => {
    finishOldRead({
      data: [
        { ...work, line1: work.addressLine1, postalCode: work.pincode },
        { ...home, line1: home.addressLine1, postalCode: home.pincode },
      ],
    });
  });
  await flush();
  expect(active?.id).toBe('home');
  expect(kitchens).toEqual(['Bengaluru Kitchen']);
});

it('does not display the previous city when its slow kitchen request finishes late', async () => {
  let finishOldRead!: (response: any) => void;
  get.mockImplementation((_path, config) => {
    if (config?.params?.state === 'Auckland')
      return new Promise((resolve) => {
        finishOldRead = resolve;
      });
    return Promise.resolve({
      data: { data: [{ id: 'india', businessName: 'Bengaluru Kitchen' }] },
    });
  });
  await act(async () => {
    void client.invalidateQueries({ queryKey: ['chefs'] });
  });
  jest.mocked(api.put).mockImplementation(() => new Promise(() => {}));
  await act(async () => {
    select.mutate(home);
  });
  await flush();
  await act(async () => {
    finishOldRead({
      data: { data: [{ id: 'nz', businessName: 'NZ Kitchen' }] },
    });
  });
  await flush();
  expect(active?.id).toBe('home');
  expect(kitchens).toEqual(['Bengaluru Kitchen']);
});
