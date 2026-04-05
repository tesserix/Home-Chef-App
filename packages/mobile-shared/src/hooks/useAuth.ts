// Zustand auth store with expo-secure-store persistence
// This is the single source of truth for auth state across all three apps
// Pattern: matches Zustand v5 API (no deprecated createStore pattern)

import { create } from 'zustand';
import {
  getAccessToken,
  getRefreshToken,
  setTokens,
  clearTokens,
  isBiometricsEnabled,
  setBiometricsEnabled,
} from '../utils/storage';
import { User, AuthResponse } from '../types/user';

interface AuthState {
  // State
  user: User | null;
  accessToken: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  biometricsEnabled: boolean;

  // Actions
  /** Load tokens from expo-secure-store into memory on app start */
  hydrateFromStorage: () => Promise<void>;
  /** Store tokens after successful login/register */
  setAuthResponse: (response: AuthResponse) => Promise<void>;
  /** Clear tokens and reset state on logout */
  logout: () => Promise<void>;
  /** Update biometrics preference */
  setBiometricsEnabled: (enabled: boolean) => Promise<void>;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  accessToken: null,
  isAuthenticated: false,
  isLoading: true,
  biometricsEnabled: false,

  hydrateFromStorage: async () => {
    try {
      const [token, biometrics] = await Promise.all([
        getAccessToken(),
        isBiometricsEnabled(),
      ]);
      set({
        accessToken: token,
        isAuthenticated: !!token,
        biometricsEnabled: biometrics,
        isLoading: false,
      });
    } catch {
      set({ isLoading: false });
    }
  },

  setAuthResponse: async (response: AuthResponse) => {
    await setTokens({
      accessToken: response.accessToken,
      refreshToken: response.refreshToken,
    });
    set({
      user: response.user,
      accessToken: response.accessToken,
      isAuthenticated: true,
      isLoading: false,
    });
  },

  logout: async () => {
    await clearTokens();
    set({
      user: null,
      accessToken: null,
      isAuthenticated: false,
      isLoading: false,
    });
  },

  setBiometricsEnabled: async (enabled: boolean) => {
    await setBiometricsEnabled(enabled);
    set({ biometricsEnabled: enabled });
  },
}));
