// install-guard.test.ts — fresh-install auth reset.
//
// RED phase: these fail to import until src/auth/install-guard.ts exists.
//
// The pure decision (`decideInstallReset`) maps "is the install sentinel
// present?" → the exact reset actions, with NO I/O, so it can be asserted in
// isolation. The runner (`ensureFreshInstallReset`) is exercised against mocked
// AsyncStorage / SecureStore / Firebase to prove it clears leftover Keychain
// auth material and signs Firebase out ONLY on a fresh (re)install.

import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('@react-native-async-storage/async-storage', () => ({
  default: {
    getItem: vi.fn(),
    setItem: vi.fn(),
    removeItem: vi.fn(),
  },
}));

vi.mock('expo-secure-store', () => ({
  getItemAsync: vi.fn(),
  setItemAsync: vi.fn(),
  deleteItemAsync: vi.fn(),
  AFTER_FIRST_UNLOCK: 'AFTER_FIRST_UNLOCK',
}));

vi.mock('@react-native-firebase/auth', () => {
  const signOut = vi.fn(async () => {});
  const authFn = Object.assign(vi.fn(() => ({ signOut })), {
    GoogleAuthProvider: { credential: vi.fn() },
    AppleAuthProvider: { credential: vi.fn() },
  });
  return { default: authFn };
});

import AsyncStorage from '@react-native-async-storage/async-storage';
import * as SecureStore from 'expo-secure-store';
import firebaseAuth from '@react-native-firebase/auth';
import {
  decideInstallReset,
  ensureFreshInstallReset,
  INSTALL_SENTINEL_KEY,
  FRESH_INSTALL_SECURE_KEYS,
} from '../auth/install-guard';

describe('decideInstallReset (pure)', () => {
  it('sentinel PRESENT → not a fresh install, do nothing', () => {
    const d = decideInstallReset(true);
    expect(d.isFreshInstall).toBe(false);
    expect(d.secureKeysToClear).toEqual([]);
    expect(d.signOutFirebase).toBe(false);
    expect(d.writeSentinel).toBe(false);
  });

  it('sentinel ABSENT → fresh install: clear all keys, sign out, write sentinel', () => {
    const d = decideInstallReset(false);
    expect(d.isFreshInstall).toBe(true);
    expect(d.secureKeysToClear).toEqual(FRESH_INSTALL_SECURE_KEYS);
    expect(d.signOutFirebase).toBe(true);
    expect(d.writeSentinel).toBe(true);
  });

  it('the cleared-key set covers every auth/session/credential key', () => {
    // These four are the ones hydrateFromStorage / the BFF session read to
    // decide "already signed in"; leaving any behind reintroduces the bug.
    expect(FRESH_INSTALL_SECURE_KEYS).toContain('access_token');
    expect(FRESH_INSTALL_SECURE_KEYS).toContain('refresh_token');
    expect(FRESH_INSTALL_SECURE_KEYS).toContain('hc_session_token');
    expect(FRESH_INSTALL_SECURE_KEYS).toContain('onboarding_complete');
  });

  it('sentinel key lives in AsyncStorage under a stable name', () => {
    expect(INSTALL_SENTINEL_KEY).toBe('hc_install_sentinel');
  });
});

describe('ensureFreshInstallReset (runner)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Mirror the real APIs, which resolve a Promise. Individual tests override.
    vi.mocked(SecureStore.deleteItemAsync).mockResolvedValue(undefined);
    vi.mocked(AsyncStorage.setItem).mockResolvedValue(undefined);
  });

  it('sentinel present: leaves the stored session untouched (app restart)', async () => {
    vi.mocked(AsyncStorage.getItem).mockResolvedValueOnce('1700000000000');

    const d = await ensureFreshInstallReset();

    expect(d.isFreshInstall).toBe(false);
    expect(SecureStore.deleteItemAsync).not.toHaveBeenCalled();
    expect(firebaseAuth().signOut).not.toHaveBeenCalled();
    expect(AsyncStorage.setItem).not.toHaveBeenCalled();
  });

  it('sentinel absent: clears every key, signs Firebase out, writes the sentinel', async () => {
    vi.mocked(AsyncStorage.getItem).mockResolvedValueOnce(null);

    const d = await ensureFreshInstallReset();

    expect(d.isFreshInstall).toBe(true);
    for (const key of FRESH_INSTALL_SECURE_KEYS) {
      expect(SecureStore.deleteItemAsync).toHaveBeenCalledWith(key);
    }
    expect(firebaseAuth().signOut).toHaveBeenCalledTimes(1);
    expect(AsyncStorage.setItem).toHaveBeenCalledWith(
      INSTALL_SENTINEL_KEY,
      expect.any(String),
    );
  });

  it('fails SAFE: if AsyncStorage read throws, do NOT wipe the session', async () => {
    vi.mocked(AsyncStorage.getItem).mockRejectedValueOnce(new Error('nope'));

    const d = await ensureFreshInstallReset();

    expect(d.isFreshInstall).toBe(false);
    expect(SecureStore.deleteItemAsync).not.toHaveBeenCalled();
    expect(firebaseAuth().signOut).not.toHaveBeenCalled();
  });

  it('best-effort: a failing key delete does not abort the reset', async () => {
    vi.mocked(AsyncStorage.getItem).mockResolvedValueOnce(null);
    vi.mocked(SecureStore.deleteItemAsync).mockRejectedValue(
      new Error('keychain unavailable'),
    );

    const d = await ensureFreshInstallReset();

    expect(d.isFreshInstall).toBe(true);
    // Even though every delete rejected, the sentinel is still written so the
    // guard doesn't loop-reset on every launch.
    expect(AsyncStorage.setItem).toHaveBeenCalledWith(
      INSTALL_SENTINEL_KEY,
      expect.any(String),
    );
  });
});
