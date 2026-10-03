// push-observability.test.ts — covers #870's fix: the iOS-only simulator
// guard (Android emulators with Play Services CAN receive FCM and must not
// be blocked), warnings for failed registration rather than expected simulator
// behavior, and registerDeviceTokenSafe never throwing past
// a failed registration so listener setup in _layout.tsx is unaffected.

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { AxiosInstance } from 'axios';

// This package's global vitest alias (vitest.config.ts) maps `react-native`
// to a fixed Platform.OS: 'ios' mock, which can't be mutated per test — so
// this file overrides `react-native` itself via a hoisted, mutable mock
// (mirroring the existing vi.mock('expo-device', ...) / vi.mock('expo-notifications', ...)
// pattern already used in device-token-path.test.ts).
const { mockPlatform, mockDevice, mockNotifications } = vi.hoisted(() => ({
  mockPlatform: { OS: 'android' as string },
  mockDevice: { isDevice: false as boolean },
  mockNotifications: {
    getPermissionsAsync: vi.fn(),
    requestPermissionsAsync: vi.fn(),
    getDevicePushTokenAsync: vi.fn(),
  },
}));

vi.mock('react-native', () => ({ Platform: mockPlatform }));
vi.mock('expo-device', () => mockDevice);
vi.mock('expo-notifications', () => mockNotifications);

import { getRawFCMToken, registerDeviceTokenSafe } from '../hooks/usePushToken';

function clientWith(put: AxiosInstance['put']): AxiosInstance {
  return { defaults: { baseURL: 'https://fe3dr.com/api' }, put } as unknown as AxiosInstance;
}

describe('getRawFCMToken', () => {
  let warnSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    mockPlatform.OS = 'android';
    mockDevice.isDevice = false;
    mockNotifications.getPermissionsAsync.mockReset();
    mockNotifications.requestPermissionsAsync.mockReset();
    mockNotifications.getDevicePushTokenAsync.mockReset();
    warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => undefined);
  });

  afterEach(() => {
    warnSpy.mockRestore();
  });

  it('skips iOS simulators without calling getPermissionsAsync', async () => {
    mockPlatform.OS = 'ios';
    mockDevice.isDevice = false;

    const result = await getRawFCMToken();

    expect(result).toBeNull();
    expect(mockNotifications.getPermissionsAsync).not.toHaveBeenCalled();
    expect(warnSpy).not.toHaveBeenCalled();
  });

  it('does NOT short-circuit an Android emulator — resolves a real token', async () => {
    mockPlatform.OS = 'android';
    mockDevice.isDevice = false; // Android emulator, no Play Services device flag
    mockNotifications.getPermissionsAsync.mockResolvedValue({ status: 'granted' });
    mockNotifications.getDevicePushTokenAsync.mockResolvedValue({ data: 'fcm-token-abc' });

    const result = await getRawFCMToken();

    expect(mockNotifications.getPermissionsAsync).toHaveBeenCalled();
    expect(result).toBe('fcm-token-abc');
  });

  it('resolves null and warns with the denied status when permission is refused', async () => {
    mockPlatform.OS = 'android';
    mockDevice.isDevice = false;
    mockNotifications.getPermissionsAsync.mockResolvedValue({ status: 'undetermined' });
    mockNotifications.requestPermissionsAsync.mockResolvedValue({ status: 'denied' });

    const result = await getRawFCMToken();

    expect(result).toBeNull();
    expect(warnSpy).toHaveBeenCalledWith(expect.stringContaining('denied'));
  });
});

describe('registerDeviceTokenSafe', () => {
  let logSpy: ReturnType<typeof vi.spyOn>;
  let warnSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    logSpy = vi.spyOn(console, 'log').mockImplementation(() => undefined);
    warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => undefined);
  });

  afterEach(() => {
    logSpy.mockRestore();
    warnSpy.mockRestore();
  });

  it('returns true and logs a truncated token on success', async () => {
    const put = vi.fn().mockResolvedValue({});
    const client = clientWith(put);

    const result = await registerDeviceTokenSafe(client, 'fcm-token-abcdefghijklmnop');

    expect(result).toBe(true);
    expect(logSpy).toHaveBeenCalledWith(
      '[push] Device token registered:',
      expect.stringContaining('fcm-token-ab')
    );
  });

  it('returns false, warns, and does not throw when the PUT rejects', async () => {
    const put = vi.fn().mockRejectedValue(new Error('network down'));
    const client = clientWith(put);

    const result = await registerDeviceTokenSafe(client, 'fcm-token-xyz');

    expect(result).toBe(false);
    expect(warnSpy).toHaveBeenCalledWith(
      '[push] Device token registration failed',
      expect.any(Error)
    );
  });
});
