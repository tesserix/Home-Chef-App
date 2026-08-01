// Raw FCM device token registration
// CRITICAL (D-09): Use getDevicePushTokenAsync() NOT getExpoPushTokenAsync()
// Reason: apps/api/services/push.go calls FCM HTTP v1 API directly with raw device tokens.
// Expo Push Tokens (ExponentPushToken[...]) are incompatible and cause silent FCM 404 errors.

import * as Notifications from 'expo-notifications';
import * as Device from 'expo-device';
import { Platform } from 'react-native';
import { AxiosInstance, AxiosRequestConfig } from 'axios';

/**
 * Get the raw FCM device token for this device.
 * Returns null on iOS simulators (cannot receive push) or when permission is denied.
 *
 * IMPORTANT: If the returned token starts with "ExponentPushToken", throw immediately —
 * that means getExpoPushTokenAsync() was called by mistake.
 */
export async function getRawFCMToken(): Promise<string | null> {
  // iOS-only: iOS simulators cannot receive push notifications at all. Android
  // emulators WITH Google Play Services CAN receive FCM pushes, so gating this
  // on `!Device.isDevice` alone (as before) unconditionally blocked every
  // Android emulator from ever registering a token (#870) — do not re-broaden
  // this back to blocking non-iOS emulators.
  if (Platform.OS === 'ios' && !Device.isDevice) {
    console.warn('[push] Skipped: iOS simulator cannot receive push notifications');
    return null;
  }

  const { status: existingStatus } = await Notifications.getPermissionsAsync();
  let finalStatus = existingStatus;

  if (existingStatus !== 'granted') {
    const { status } = await Notifications.requestPermissionsAsync();
    finalStatus = status;
  }

  if (finalStatus !== 'granted') {
    console.warn(`[push] Skipped: notification permission not granted (status=${finalStatus})`);
    return null;
  }

  // D-09: raw FCM token via getDevicePushTokenAsync(), NOT getExpoPushTokenAsync()
  const tokenData = await Notifications.getDevicePushTokenAsync();
  const token = tokenData.data;

  // Guard: detect if Expo Push Token was accidentally returned
  if (typeof token === 'string' && token.startsWith('ExponentPushToken')) {
    throw new Error(
      '[usePushToken] Received Expo Push Token instead of raw FCM token. ' +
        'Do NOT call getExpoPushTokenAsync(). See D-09 in CONTEXT.md.'
    );
  }

  return token;
}

/**
 * Resolve the device-token path for a given client.
 *
 * The apps mount their API client at DIFFERENT depths — customer's
 * EXPO_PUBLIC_API_URL ends at `/api`, while vendor's and delivery's already end
 * at `/api/v1` (their hooks call `/chef/...`, `/driver/...` with no version
 * prefix). A hardcoded `/v1/profile/device-token` therefore resolved to
 * `/api/v1/v1/profile/device-token` on vendor + delivery and 404'd, so their FCM
 * token was NEVER stored — every chef/driver push then logged
 * "Push skipped: user <id> has no FCM token" and no vendor notification could
 * ever arrive. It failed silently because registration is best-effort.
 *
 * Derive the prefix from the client instead of assuming one, so this is right
 * for every app and for any future one regardless of where it mounts.
 */
export function deviceTokenPath(client: AxiosInstance): string {
  const base = (client.defaults.baseURL ?? '').replace(/\/+$/, '');
  return /\/v1$/.test(base) ? '/profile/device-token' : '/v1/profile/device-token';
}

/**
 * Register the raw FCM token with the Go API.
 * Endpoint: PUT /api/v1/profile/device-token
 * Requires authenticated API client (Bearer token must be set).
 */
export async function registerDeviceToken(
  client: AxiosInstance,
  token: string
): Promise<void> {
  // skipAuthFailure: a 401 here (e.g. registration racing a session refresh on
  // cold start) must NOT clear the session and bounce the user to login — this
  // is a best-effort call. The client's response interceptor honours this flag.
  const config: AxiosRequestConfig & { skipAuthFailure: boolean } = {
    skipAuthFailure: true,
  };
  await client.put(deviceTokenPath(client), { token }, config);
}

/**
 * Register the raw FCM token with the Go API without ever throwing.
 *
 * Wraps registerDeviceToken so every call site (the initial registration in
 * _layout.tsx's push-setup effect and the token-rotation listener) can await
 * this instead of running its own try/catch. A failed or thrown registration
 * here must never prevent the rotation/badge/deep-link-tap listeners a few
 * lines below it from attaching — that was the core defect in #870, where one
 * failed PUT turned into total notification silence for the session.
 */
export async function registerDeviceTokenSafe(
  client: AxiosInstance,
  token: string
): Promise<boolean> {
  try {
    await registerDeviceToken(client, token);
    console.log('[push] Device token registered:', token.slice(0, 12) + '...');
    return true;
  } catch (err: unknown) {
    console.warn('[push] Device token registration failed', err);
    return false;
  }
}
