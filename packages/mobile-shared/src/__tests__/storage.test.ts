// storage.test.ts — RED phase: these tests FAIL until storage.ts is implemented

import { describe, it, expect, vi, beforeEach } from 'vitest';

// Mock expo-secure-store before importing storage
vi.mock('expo-secure-store', () => ({
  getItemAsync: vi.fn(),
  setItemAsync: vi.fn(),
  deleteItemAsync: vi.fn(),
  AFTER_FIRST_UNLOCK: 'AFTER_FIRST_UNLOCK',
}));

import * as SecureStore from 'expo-secure-store';
import {
  getAccessToken,
  getRefreshToken,
  setTokens,
  clearTokens,
  isBiometricsEnabled,
  STORAGE_KEYS,
} from '../utils/storage';

describe('storage utils', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('getAccessToken returns string when SecureStore has access_token', async () => {
    vi.mocked(SecureStore.getItemAsync).mockResolvedValueOnce('token123');
    const result = await getAccessToken();
    expect(result).toBe('token123');
    expect(SecureStore.getItemAsync).toHaveBeenCalledWith(STORAGE_KEYS.ACCESS_TOKEN);
  });

  it('getAccessToken returns null when not set', async () => {
    vi.mocked(SecureStore.getItemAsync).mockResolvedValueOnce(null);
    const result = await getAccessToken();
    expect(result).toBeNull();
  });

  // #428 — a Keychain read that THROWS (transient cold-start race) must not be
  // treated as "no token" (which logs the user out). Retry once.
  it('getAccessToken retries once when the read throws, then returns the value', async () => {
    vi.mocked(SecureStore.getItemAsync)
      .mockRejectedValueOnce(new Error('keychain unavailable'))
      .mockResolvedValueOnce('token123');
    const result = await getAccessToken();
    expect(result).toBe('token123');
    expect(SecureStore.getItemAsync).toHaveBeenCalledTimes(2);
  });

  it('getAccessToken returns null only after the read throws twice', async () => {
    vi.mocked(SecureStore.getItemAsync).mockRejectedValue(new Error('keychain unavailable'));
    const result = await getAccessToken();
    expect(result).toBeNull();
    expect(SecureStore.getItemAsync).toHaveBeenCalledTimes(2);
  });

  it('setTokens writes both tokens with AFTER_FIRST_UNLOCK keychain accessibility', async () => {
    vi.mocked(SecureStore.setItemAsync).mockResolvedValue(undefined);
    await setTokens({ accessToken: 'access123', refreshToken: 'refresh456' });
    expect(SecureStore.setItemAsync).toHaveBeenCalledWith(STORAGE_KEYS.ACCESS_TOKEN, 'access123', {
      keychainAccessible: 'AFTER_FIRST_UNLOCK',
    });
    expect(SecureStore.setItemAsync).toHaveBeenCalledWith(STORAGE_KEYS.REFRESH_TOKEN, 'refresh456', {
      keychainAccessible: 'AFTER_FIRST_UNLOCK',
    });
    expect(SecureStore.setItemAsync).toHaveBeenCalledTimes(2);
  });

  it('clearTokens deletes both keys and resolves without error', async () => {
    vi.mocked(SecureStore.deleteItemAsync).mockResolvedValue(undefined);
    await expect(clearTokens()).resolves.toBeUndefined();
    expect(SecureStore.deleteItemAsync).toHaveBeenCalledWith(STORAGE_KEYS.ACCESS_TOKEN);
    expect(SecureStore.deleteItemAsync).toHaveBeenCalledWith(STORAGE_KEYS.REFRESH_TOKEN);
  });

  it('STORAGE_KEYS uses exact key strings', () => {
    expect(STORAGE_KEYS.ACCESS_TOKEN).toBe('access_token');
    expect(STORAGE_KEYS.REFRESH_TOKEN).toBe('refresh_token');
    expect(STORAGE_KEYS.BIOMETRICS_ENABLED).toBe('biometrics_enabled');
  });

  it('isBiometricsEnabled returns true when stored value is "true"', async () => {
    vi.mocked(SecureStore.getItemAsync).mockResolvedValueOnce('true');
    const result = await isBiometricsEnabled();
    expect(result).toBe(true);
  });

  it('isBiometricsEnabled returns false when stored value is not "true"', async () => {
    vi.mocked(SecureStore.getItemAsync).mockResolvedValueOnce(null);
    const result = await isBiometricsEnabled();
    expect(result).toBe(false);
  });
});
