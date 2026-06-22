import { Alert } from 'react-native';
import { createApiClient } from '@homechef/mobile-shared/api';
import { useAuthStore } from '../store/auth-store';
import { appPlatform, appVersion } from './app-version';

// Single axios instance for the admin app. Injects the BFF session token as a
// Bearer header, stamps the app version/platform, and handles 401 (session
// dead → logout) + 426 (client too old). Admin has no in-app upgrade wall, so
// 426 surfaces as an alert rather than routing to a dedicated screen.
export const api = createApiClient({
  baseURL: process.env.EXPO_PUBLIC_API_URL!,
  getToken: () => useAuthStore.getState().accessToken,
  appVersion,
  platform: appPlatform,
  onAuthFailure: () => {
    // Token rejected — clear the store; the root layout's auth gate redirects
    // to /(auth)/login on the next render.
    useAuthStore.getState().logout();
  },
  onUpgradeRequired: (payload) => {
    Alert.alert(
      'Update required',
      payload.minVersion
        ? `This version of the admin app is no longer supported. Please update to ${payload.minVersion} or later.`
        : 'This version of the admin app is no longer supported. Please update to continue.'
    );
  },
});
