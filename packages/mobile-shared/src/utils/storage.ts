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

// A Keychain read can transiently fail (e.g. a race at cold start) by THROWING
// rather than returning null. Treating that throw as "no token" is what logs the
// user out. Retry once, then fall back to null. (#428)
async function readTokenResilient(key: string): Promise<string | null> {
  try {
    return await SecureStore.getItemAsync(key);
  } catch {
    try {
      return await SecureStore.getItemAsync(key);
    } catch {
      return null;
    }
  }
}

export async function getAccessToken(): Promise<string | null> {
  return readTokenResilient(STORAGE_KEYS.ACCESS_TOKEN);
}

export async function getRefreshToken(): Promise<string | null> {
  return readTokenResilient(STORAGE_KEYS.REFRESH_TOKEN);
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
    SecureStore.setItemAsync(
      STORAGE_KEYS.ACCESS_TOKEN,
      tokens.accessToken,
      TOKEN_KEYCHAIN_OPTIONS
    ),
  ];
  if (tokens.refreshToken) {
    writes.push(
      SecureStore.setItemAsync(
        STORAGE_KEYS.REFRESH_TOKEN,
        tokens.refreshToken,
        TOKEN_KEYCHAIN_OPTIONS
      )
    );
  } else {
    // Clear any stale refresh token from a prior session so the api client's
    // 401 handler never reads a value that no longer matches the access token.
    writes.push(SecureStore.deleteItemAsync(STORAGE_KEYS.REFRESH_TOKEN));
  }
  await Promise.all(writes);
}

export async function clearTokens(): Promise<void> {
  await Promise.all([
    SecureStore.deleteItemAsync(STORAGE_KEYS.ACCESS_TOKEN),
    SecureStore.deleteItemAsync(STORAGE_KEYS.REFRESH_TOKEN),
  ]);
}

export async function isBiometricsEnabled(): Promise<boolean> {
  const val = await SecureStore.getItemAsync(STORAGE_KEYS.BIOMETRICS_ENABLED);
  return val === 'true';
}

export async function setBiometricsEnabled(enabled: boolean): Promise<void> {
  await SecureStore.setItemAsync(
    STORAGE_KEYS.BIOMETRICS_ENABLED,
    enabled ? 'true' : 'false'
  );
}

export async function isOnboardingComplete(): Promise<boolean> {
  const val = await SecureStore.getItemAsync(STORAGE_KEYS.ONBOARDING_COMPLETE);
  return val === 'true';
}

export async function setOnboardingCompleteInStore(complete: boolean): Promise<void> {
  await SecureStore.setItemAsync(
    STORAGE_KEYS.ONBOARDING_COMPLETE,
    complete ? 'true' : 'false'
  );
}
