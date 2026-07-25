import { router } from 'expo-router';
import { getDeviceToken, emitMFARequired } from '@homechef/mobile-shared/mfa';
import { createApiClient, type AccountBlockedStatus } from '@homechef/mobile-shared/api';
import { useAuthStore } from '../store/auth-store';
import { appPlatform, appVersion } from './app-version';

// Guards against a navigation storm: a paused account 403s on EVERY in-flight
// query, and each one would otherwise push the screen again.
let routingToBlockedScreen = false;

function handleAccountBlocked(status: AccountBlockedStatus) {
  if (routingToBlockedScreen) return;
  routingToBlockedScreen = true;
  router.replace(`/account-paused?status=${status}` as never);
  setTimeout(() => {
    routingToBlockedScreen = false;
  }, 3000);
}

export const api = createApiClient({
  // Two-factor: scope device trust to this app, present the remembered-device
  // token, and hand a challenge to MFAGateProvider when the API demands one.
  clientApp: 'vendor',
  getDeviceToken,
  onMFARequired: emitMFARequired,
  baseURL: process.env.EXPO_PUBLIC_API_URL!,
  getToken: () => useAuthStore.getState().accessToken,
  appVersion,
  platform: appPlatform,
  onAuthFailure: () => {
    // Token refresh failed — store clears itself, layout auth guard will redirect
    useAuthStore.getState().logout();
  },
  // Paused/deleted accounts 403 on every protected call; route to a screen that
  // explains why and offers reactivation, rather than leaving a dead app.
  onAccountBlocked: handleAccountBlocked,
  onUpgradeRequired: (payload) => {
    // Backend returned 426 — pin the upgrade wall regardless of the
    // current route. The screen itself reads minVersion/storeUrl from
    // search params so the chef sees the same numbers the server sent.
    const qs = new URLSearchParams();
    if (payload.minVersion) qs.set('minVersion', payload.minVersion);
    if (payload.storeUrl) qs.set('storeUrl', payload.storeUrl);
    const href = qs.toString()
      ? `/upgrade-required?${qs.toString()}`
      : '/upgrade-required';
    router.replace(href as never);
  },
});
