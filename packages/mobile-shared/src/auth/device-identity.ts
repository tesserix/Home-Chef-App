// A stable per-install identifier (#1164). The API keys push tokens and the
// new-device sign-in email on it, so two devices on one account stay distinct
// instead of overwriting each other.

import Constants from 'expo-constants';
import * as Device from 'expo-device';

import { secureGet, secureSet, TOKEN_KEYCHAIN_OPTIONS } from '../utils/storage';

export const DEVICE_ID_KEY = 'hc_device_id';

let cached: string | null = null;

/** Test seam: forget the in-process copy so a "restart" can be exercised. */
export function resetDeviceIdCache(): void {
  cached = null;
}

export async function getDeviceId(): Promise<string> {
  if (cached) return cached;
  const stored = await secureGet(DEVICE_ID_KEY);
  if (stored) {
    cached = stored;
    return stored;
  }
  const minted = mintDeviceId();
  cached = minted;
  await secureSet(DEVICE_ID_KEY, minted, TOKEN_KEYCHAIN_OPTIONS);
  return minted;
}

/**
 * The device id for synchronous callers (the axios request interceptor).
 * Returns '' until the stored id has been read, and starts that read so the
 * next request carries the header.
 */
export function deviceIdSync(): string {
  if (!cached) void getDeviceId();
  return cached ?? '';
}

export interface DeviceDescription {
  platform: string;
  label: string;
  appVersion: string;
}

/** The human-readable parts of this install, for the new-device email. */
export function describeDevice(): DeviceDescription {
  return {
    platform: (Device.osName ?? '').toLowerCase(),
    label: Device.modelName ?? '',
    appVersion: Constants.expoConfig?.version ?? '',
  };
}

function mintDeviceId(): string {
  const rand = () => Math.random().toString(36).slice(2, 10);
  return `hc-${Date.now().toString(36)}-${rand()}${rand()}`;
}
