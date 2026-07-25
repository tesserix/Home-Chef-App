// Fresh-install auth reset.
//
// iOS Keychain (expo-secure-store) AND the native Firebase Auth session both
// SURVIVE an app uninstall+reinstall; AsyncStorage does NOT. That asymmetry is
// the fresh-install detector: on the first launch after a (re)install there is
// no AsyncStorage sentinel, yet the Keychain may still hold a previous run's
// access/session tokens — which hydrateFromStorage() reads as "already signed
// in" (isAuthenticated = !!token), so the login screen never mounts and a
// returning user is dead-ended in the onboarding wizard.
//
// So BEFORE the first hydrate/routing decision we detect the missing sentinel,
// clear every SecureStore auth key + sign Firebase out, then write the sentinel.
// Once written, every subsequent normal launch (and app RESTART — #428/#436) is
// a cheap no-op that leaves the stored session intact, preserving the
// intentional "stay logged in across restarts" behavior.

import AsyncStorage from '@react-native-async-storage/async-storage';
import * as SecureStore from 'expo-secure-store';
import { signOut as firebaseSignOut } from './sign-in';

// AsyncStorage sentinel. Present ⇒ this install has booted before ⇒ keep the
// stored session. Absent ⇒ fresh (re)install ⇒ reset. Deliberately in
// AsyncStorage, not SecureStore, because only AsyncStorage is wiped on uninstall
// — a SecureStore sentinel would survive and defeat the detector.
export const INSTALL_SENTINEL_KEY = 'hc_install_sentinel';

// Every SecureStore key holding auth / session / credential material. All are
// cleared on a fresh install so a prior owner's Keychain residue can't
// masquerade as a live session. One list = one source of truth for the reset:
//   - access_token / refresh_token   → utils/storage (feeds hydrateFromStorage)
//   - hc_session_token               → auth/bff-session (BFF session)
//   - onboarding_complete            → utils/storage (device-local wizard flag)
//   - biometrics_enabled             → utils/storage (device-local unlock pref;
//                                       a reinstalled device shouldn't inherit a
//                                       prior owner's biometric-unlock consent)
//   - mfa_device_token               → mfa/device-token (remembered-device
//                                       credential; bearer material, must not
//                                       outlive the user who earned it)
export const FRESH_INSTALL_SECURE_KEYS = [
  'access_token',
  'refresh_token',
  'hc_session_token',
  'onboarding_complete',
  'biometrics_enabled',
  'mfa_device_token',
] as const;

export interface InstallResetDecision {
  /** True on the first launch after a (re)install (sentinel absent). */
  isFreshInstall: boolean;
  /** SecureStore keys to delete (empty when not a fresh install). */
  secureKeysToClear: readonly string[];
  /** Whether to call Firebase signOut() (only on a fresh install). */
  signOutFirebase: boolean;
  /** Whether to write the sentinel (only on a fresh install). */
  writeSentinel: boolean;
}

/**
 * decideInstallReset maps "is the install sentinel present?" to exactly what the
 * reset should do. Pure — no I/O — so the decision is unit-testable in isolation.
 * @param sentinelPresent - true if the AsyncStorage sentinel already exists
 */
export function decideInstallReset(sentinelPresent: boolean): InstallResetDecision {
  if (sentinelPresent) {
    return {
      isFreshInstall: false,
      secureKeysToClear: [],
      signOutFirebase: false,
      writeSentinel: false,
    };
  }
  return {
    isFreshInstall: true,
    secureKeysToClear: FRESH_INSTALL_SECURE_KEYS,
    signOutFirebase: true,
    writeSentinel: true,
  };
}

/**
 * ensureFreshInstallReset runs the fresh-install guard. MUST be awaited before
 * hydrateFromStorage() and the first routing decision in every app's root
 * layout. Idempotent: after the first run the sentinel exists, so later launches
 * are a no-op that leaves the stored session intact (#428/#436).
 * @returns the decision that was applied (useful for tests / diagnostics).
 */
export async function ensureFreshInstallReset(): Promise<InstallResetDecision> {
  let sentinelPresent: boolean;
  try {
    sentinelPresent = (await AsyncStorage.getItem(INSTALL_SENTINEL_KEY)) != null;
  } catch {
    // If AsyncStorage itself is unreadable we cannot prove this is a fresh
    // install; treat it as NOT fresh so a transient storage error never wipes a
    // legitimately logged-in user's session. Fail safe.
    return decideInstallReset(true);
  }

  const decision = decideInstallReset(sentinelPresent);
  if (!decision.isFreshInstall) return decision;

  // Clear leftover Keychain auth material. Each delete is independent and
  // best-effort — one failing/absent key must not abort the rest.
  await Promise.all(
    decision.secureKeysToClear.map((key) =>
      SecureStore.deleteItemAsync(key).catch(() => {
        /* best-effort: key may be absent or the Keychain briefly unavailable */
      })
    )
  );

  if (decision.signOutFirebase) {
    try {
      await firebaseSignOut();
    } catch {
      // No current user, or Firebase not yet initialised — nothing to sign out.
    }
  }

  if (decision.writeSentinel) {
    try {
      await AsyncStorage.setItem(INSTALL_SENTINEL_KEY, String(Date.now()));
    } catch {
      // If the sentinel write fails the guard simply re-runs next launch; the
      // reset is idempotent (the user is already signed out), so it's harmless.
    }
  }

  return decision;
}
