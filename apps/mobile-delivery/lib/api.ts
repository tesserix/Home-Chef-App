import { router } from 'expo-router';
import { getDeviceToken, emitMFARequired } from '@homechef/mobile-shared/mfa';
import { createApiClient, type AccountBlockedStatus } from '@homechef/mobile-shared/api';
import { useAuthStore } from '../store/auth-store';

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
  clientApp: 'delivery',
  getDeviceToken,
  onMFARequired: emitMFARequired,
  baseURL: process.env.EXPO_PUBLIC_API_URL!,
  getToken: () => useAuthStore.getState().accessToken,
  onAuthFailure: () => {
    // Token refresh failed — store clears itself, layout auth guard will redirect
    useAuthStore.getState().logout();
  },
  // Paused/deleted accounts 403 on every protected call; route to a screen that
  // explains why and offers reactivation, rather than leaving a dead app.
  onAccountBlocked: handleAccountBlocked,
});
