import { router } from 'expo-router';
import { createApiClient } from '@homechef/mobile-shared/api';
import { useAuthStore } from '../store/auth-store';
import { appPlatform, appVersion } from './app-version';

export const api = createApiClient({
  baseURL: process.env.EXPO_PUBLIC_API_URL!,
  getToken: () => useAuthStore.getState().accessToken,
  appVersion,
  platform: appPlatform,
  onAuthFailure: () => {
    // Token refresh failed — store clears itself, layout auth guard will redirect
    useAuthStore.getState().logout();
  },
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
