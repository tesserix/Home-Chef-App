// Zustand auth store with expo-secure-store persistence
// This is the single source of truth for auth state across all three apps
// Pattern: matches Zustand v5 API (no deprecated createStore pattern)

import { create } from 'zustand';
import { clearDeviceToken } from '../mfa/device-token';
import {
  getAccessToken,
  getRefreshToken,
  setTokens,
  clearTokens,
  isBiometricsEnabled,
  setBiometricsEnabled,
  isOnboardingComplete,
  setOnboardingCompleteInStore,
  isGuestMode,
  setGuestModeInStore,
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
  /**
   * The user chose to browse without an account.
   *
   * App Review guideline 5.1.1(iv): an app may only require an account for
   * features that genuinely need one. Browsing chefs and menus does not, so the
   * customer app lets people in and asks for an account at the point of ordering.
   *
   * Distinct from isAuthenticated on purpose — a guest has no token, so every
   * write path must check this rather than assume "in the app" means "signed in".
   */
  isGuest: boolean;

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
  /** Enter or leave browse-without-an-account mode (persisted). */
  setGuest: (guest: boolean) => Promise<void>;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  accessToken: null,
  isAuthenticated: false,
  isLoading: true,
  biometricsEnabled: false,
  onboardingComplete: false,
  isGuest: false,

  hydrateFromStorage: async () => {
    try {
      const [token, biometrics, onboarding, guest] = await Promise.all([
        getAccessToken(),
        isBiometricsEnabled(),
        isOnboardingComplete(),
        isGuestMode(),
      ]);
      set({
        accessToken: token,
        isAuthenticated: !!token,
        biometricsEnabled: biometrics,
        onboardingComplete: onboarding,
        // A real session always wins: a guest who signs in is no longer a guest,
        // and a stale flag must not shadow their account.
        isGuest: !token && guest,
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
    // Signing in ends guest mode — otherwise the flag survives in the keychain
    // and a later sign-out silently drops the user back into a browse session
    // instead of the login screen.
    await setGuestModeInStore(false);
    set({
      user: response.user,
      accessToken: response.accessToken,
      isAuthenticated: true,
      isGuest: false,
      isLoading: false,
    });
  },

  logout: async () => {
    await clearTokens();
    // Drop the two-factor device token for the same reason: it is credential
    // material scoped to the user who just left. The server would refuse it for
    // anyone else, but leaving it in the keychain after sign-out is exactly the
    // kind of residue that turns into a bug when a device changes hands.
    await clearDeviceToken();
    // Reset the PERSISTED onboarding flag too — it's a device-local flag, so
    // without this the next user to sign in on this device inherits the
    // previous user's "onboarding complete" state and skips the wizard.
    await setOnboardingCompleteInStore(false);
    await setGuestModeInStore(false);
    set({
      user: null,
      accessToken: null,
      isAuthenticated: false,
      isGuest: false,
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

  setGuest: async (guest: boolean) => {
    await setGuestModeInStore(guest);
    set({ isGuest: guest });
  },
}));
