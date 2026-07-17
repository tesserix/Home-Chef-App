// expo-secure-store wrappers for JWT token management
// Uses iOS Keychain / Android Keystore (NOT AsyncStorage — which is plaintext)

import * as SecureStore from 'expo-secure-store';

// Key constants — use these exact strings throughout all mobile apps
export const STORAGE_KEYS = {
  ACCESS_TOKEN: 'access_token',
  REFRESH_TOKEN: 'refresh_token',
  BIOMETRICS_ENABLED: 'biometrics_enabled',
  ONBOARDING_COMPLETE: 'onboarding_complete',
} as const;

// Auth tokens must survive app restarts and be readable on a launch that
// happens BEFORE the current session's first unlock (a push- or background-
// driven wake right after a reboot). The expo-secure-store default is
// `WHEN_UNLOCKED`, under which such a launch reads null and the app spuriously
// signs the user out. `AFTER_FIRST_UNLOCK` keeps the token readable for the rest
// of the boot once the device has been unlocked once — the correct accessibility
// for a "stay logged in until explicit logout" token. (#428)
export const TOKEN_KEYCHAIN_OPTIONS: SecureStore.SecureStoreOptions = {
  keychainAccessible: SecureStore.AFTER_FIRST_UNLOCK,
};

// In-memory fallback used when the Keychain is unavailable. Unsigned simulator
// builds have no application-identifier entitlement, so every SecureStore call
// throws "a required entitlement isn't present" — which would break sign-in
// (unguarded writes) and log the user out (reads treated as null). Signed device
// / EAS builds always take the real-Keychain path; the memory map is only ever
// touched when the Keychain throws (dev/sim), where a per-process copy is
// acceptable.
const memoryStore = new Map<string, string>();

// A Keychain read can also transiently fail (a race at cold start) by THROWING
// rather than returning null — treating that as "no token" is what spuriously
// signs the user out (#428). So: try the Keychain, retry once on throw, then
// fall back to the in-memory copy.
async function secureGet(key: string): Promise<string | null> {
  try {
    const v = await SecureStore.getItemAsync(key);
    if (v != null) return v;
  } catch {
    try {
      const v = await SecureStore.getItemAsync(key);
      if (v != null) return v;
    } catch {
      /* Keychain unavailable — fall through to the in-memory copy. */
    }
  }
  return memoryStore.get(key) ?? null;
}

async function secureSet(
  key: string,
  value: string,
  options?: SecureStore.SecureStoreOptions
): Promise<void> {
  try {
    await SecureStore.setItemAsync(key, value, options);
    memoryStore.delete(key); // the Keychain now owns it
  } catch {
    memoryStore.set(key, value);
  }
}

async function secureDelete(key: string): Promise<void> {
  memoryStore.delete(key);
  try {
    await SecureStore.deleteItemAsync(key);
  } catch {
    /* Keychain unavailable — the in-memory copy is already cleared. */
  }
}

export async function getAccessToken(): Promise<string | null> {
  return secureGet(STORAGE_KEYS.ACCESS_TOKEN);
}

export async function getRefreshToken(): Promise<string | null> {
  return secureGet(STORAGE_KEYS.REFRESH_TOKEN);
}

export async function setTokens(tokens: {
  accessToken: string;
  refreshToken?: string;
}): Promise<void> {
  // In the GIP/BFF model the BFF owns refresh; the client only persists the
  // session token. The refreshToken field is kept optional for any caller
  // that still has one (e.g. test fixtures, legacy flows). SecureStore
  // rejects empty strings, so skip the write when refreshToken is falsy.
  const writes: Promise<void>[] = [
    secureSet(STORAGE_KEYS.ACCESS_TOKEN, tokens.accessToken, TOKEN_KEYCHAIN_OPTIONS),
  ];
  if (tokens.refreshToken) {
    writes.push(
      secureSet(STORAGE_KEYS.REFRESH_TOKEN, tokens.refreshToken, TOKEN_KEYCHAIN_OPTIONS)
    );
  } else {
    // Clear any stale refresh token from a prior session so the api client's
    // 401 handler never reads a value that no longer matches the access token.
    writes.push(secureDelete(STORAGE_KEYS.REFRESH_TOKEN));
  }
  await Promise.all(writes);
}

export async function clearTokens(): Promise<void> {
  await Promise.all([
    secureDelete(STORAGE_KEYS.ACCESS_TOKEN),
    secureDelete(STORAGE_KEYS.REFRESH_TOKEN),
  ]);
}

export async function isBiometricsEnabled(): Promise<boolean> {
  return (await secureGet(STORAGE_KEYS.BIOMETRICS_ENABLED)) === 'true';
}

export async function setBiometricsEnabled(enabled: boolean): Promise<void> {
  await secureSet(STORAGE_KEYS.BIOMETRICS_ENABLED, enabled ? 'true' : 'false');
}

export async function isOnboardingComplete(): Promise<boolean> {
  return (await secureGet(STORAGE_KEYS.ONBOARDING_COMPLETE)) === 'true';
}

export async function setOnboardingCompleteInStore(complete: boolean): Promise<void> {
  await secureSet(STORAGE_KEYS.ONBOARDING_COMPLETE, complete ? 'true' : 'false');
}
