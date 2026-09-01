import { create } from 'zustand';
import type { SessionUser } from '@/shared/types/auth';

interface AuthState {
  user: SessionUser | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  csrfToken: string | null;
  /** Legacy bearer token slot; the BFF cookie is the real session. */
  accessToken: string | null;
  refreshToken: string | null;
  onboardingCompleted: boolean | null;
  setSession: (user: SessionUser, csrfToken?: string) => void;
  setApiAuth: (user: SessionUser, accessToken: string, refreshToken: string) => void;
  clearAuth: () => void;
  setLoading: (loading: boolean) => void;
  setOnboardingCompleted: (completed: boolean) => void;
  initialize: () => Promise<void>;
}

export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  isAuthenticated: false,
  isLoading: true,
  csrfToken: null,
  accessToken: null,
  refreshToken: null,
  onboardingCompleted: null,

  setSession: (user, csrfToken) =>
    set({ user, isAuthenticated: true, csrfToken: csrfToken ?? get().csrfToken }),

  setApiAuth: (user, accessToken, refreshToken) =>
    set({ user, isAuthenticated: true, accessToken, refreshToken }),

  clearAuth: () =>
    set({
      user: null,
      isAuthenticated: false,
      csrfToken: null,
      accessToken: null,
      refreshToken: null,
      onboardingCompleted: null,
    }),

  setLoading: (isLoading) => set({ isLoading }),

  setOnboardingCompleted: (completed) => set({ onboardingCompleted: completed }),

  /**
   * Bootstrap auth state from the BFF session cookie — the single source
   * of truth now that login happens on Zitadel's hosted pages.
   */
  initialize: async () => {
    try {
      const { authService } = await import('@/features/auth/services/auth-service');
      const session = await authService.getSession();

      if (session?.authenticated && session.user) {
        set({
          user: session.user,
          isAuthenticated: true,
          csrfToken: session.csrfToken ?? null,
          isLoading: false,
        });
      } else {
        set({ user: null, isAuthenticated: false, isLoading: false });
      }
    } catch {
      set({ user: null, isAuthenticated: false, isLoading: false });
    }
  },
}));
