// Storage for the two-factor device token.
//
// The token is a bearer credential: whoever holds it skips the challenge on this
// app. So it goes in SecureStore (Keychain / Keystore) alongside the session
// token, never AsyncStorage.
//
// Two kinds share one slot, deliberately. A "remember me" token persists until
// revoked; a plain elevation token expires server-side after 12 hours. The
// client cannot tell them apart and does not need to — it stores whatever the
// verify call returned and presents it until the server stops accepting it.

import * as SecureStore from 'expo-secure-store';

import { TOKEN_KEYCHAIN_OPTIONS } from '../utils/storage';

const DEVICE_TOKEN_KEY = 'mfa_device_token';

// Mirrored in memory because the axios request interceptor is synchronous and
// runs on every call — an await per request would be absurd.
let cached: string | null = null;

/** Load the stored token into memory. Call once at start-up, before the first request. */
export async function loadDeviceToken(): Promise<string | null> {
  try {
    cached = await SecureStore.getItemAsync(DEVICE_TOKEN_KEY, TOKEN_KEYCHAIN_OPTIONS);
  } catch {
    // A keychain read can fail on a locked device; treat it as "no token" and
    // let the user re-challenge rather than crashing the launch path.
    cached = null;
  }
  return cached;
}

/** The in-memory token. Synchronous, for the request interceptor. */
export function getDeviceToken(): string | null {
  return cached;
}

/** Persist a token returned by /auth/mfa/verify. */
export async function setDeviceToken(token: string): Promise<void> {
  cached = token;
  try {
    await SecureStore.setItemAsync(DEVICE_TOKEN_KEY, token, TOKEN_KEYCHAIN_OPTIONS);
  } catch {
    // Keep the in-memory copy so the current session still works; the user is
    // challenged again next launch, which is the safe direction to fail.
  }
}

/**
 * Forget the token. Called on sign-out and when the user revokes this device
 * from another one — keeping a token the server has already rejected only
 * produces a confusing loop of failed requests.
 */
export async function clearDeviceToken(): Promise<void> {
  cached = null;
  try {
    await SecureStore.deleteItemAsync(DEVICE_TOKEN_KEY, TOKEN_KEYCHAIN_OPTIONS);
  } catch {
    // best-effort
  }
}
