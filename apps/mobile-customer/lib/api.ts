import { createApiClient } from '@homechef/mobile-shared/api';
import { useAuthStore } from '../store/auth-store';

export const api = createApiClient({
  baseURL: process.env.EXPO_PUBLIC_API_URL!,
  getToken: () => useAuthStore.getState().accessToken,
  onAuthFailure: () => {
    // Token refresh failed — store clears itself, layout auth guard will redirect
    useAuthStore.getState().logout();
  },
});
