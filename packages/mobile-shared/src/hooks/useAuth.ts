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
  isOnboardingComplete,
  setOnboardingCompleteInStore,
} from '../utils/storage';
import { User, AuthResponse } from '../types/user';

interface AuthState {
  // State
  user: User | null;
  accessToken: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  biometricsEnabled: boolean;
  onboardingComplete: boolean;

  // Actions
  /** Load tokens from expo-secure-store into memory on app start */
  hydrateFromStorage: () => Promise<void>;
  /** Store tokens after successful login/register */
  setAuthResponse: (response: AuthResponse) => Promise<void>;
  /** Clear tokens and reset state on logout */
  logout: () => Promise<void>;
  /** Update biometrics preference */
  setBiometricsEnabled: (enabled: boolean) => Promise<void>;
  /** Mark onboarding as complete (persisted to SecureStore) */
  setOnboardingComplete: (complete: boolean) => Promise<void>;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  accessToken: null,
  isAuthenticated: false,
  isLoading: true,
  biometricsEnabled: false,
  onboardingComplete: false,

  hydrateFromStorage: async () => {
    try {
      const [token, biometrics, onboarding] = await Promise.all([
        getAccessToken(),
        isBiometricsEnabled(),
        isOnboardingComplete(),
      ]);
      set({
        accessToken: token,
        isAuthenticated: !!token,
        biometricsEnabled: biometrics,
        onboardingComplete: onboarding,
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
    // Reset the PERSISTED onboarding flag too — it's a device-local flag, so
    // without this the next user to sign in on this device inherits the
    // previous user's "onboarding complete" state and skips the wizard.
    await setOnboardingCompleteInStore(false);
    set({
      user: null,
      accessToken: null,
      isAuthenticated: false,
      onboardingComplete: false,
      isLoading: false,
    });
  },

  setBiometricsEnabled: async (enabled: boolean) => {
    await setBiometricsEnabled(enabled);
    set({ biometricsEnabled: enabled });
  },

  setOnboardingComplete: async (complete: boolean) => {
    await setOnboardingCompleteInStore(complete);
    set({ onboardingComplete: complete });
  },
}));
