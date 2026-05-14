import { create } from 'zustand';
import type { SessionUser } from '@/shared/types/auth';

interface AuthState {
  user: SessionUser | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  csrfToken: string | null;
  /** Firebase ID token; sent as X-Auth-Token bearer by api-client. */
  accessToken: string | null;
  /** Firebase refresh token; rotated internally by Firebase. */
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
   * Bootstrap auth state from the BFF session cookie. The cookie is the
   * single source of truth; Firebase auth-state changes flow in via
   * `subscribeAuth()` (see initialization below).
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

// Subscribe Firebase auth-state to the store so signOut from another tab (or
// token expiry detected by the SDK) clears local state automatically.
if (typeof window !== 'undefined') {
  // Lazy import to avoid pulling firebase into SSR bundles or initial JS chunk.
  void import('@/features/auth/services/auth-service').then(({ subscribeAuth, toSessionUser }) => {
    subscribeAuth((session) => {
      if (session) {
        useAuthStore.setState({
          user: toSessionUser(session),
          isAuthenticated: true,
          isLoading: false,
        });
      } else {
        useAuthStore.setState({
          user: null,
          isAuthenticated: false,
          csrfToken: null,
          accessToken: null,
          refreshToken: null,
          onboardingCompleted: null,
          isLoading: false,
        });
      }
    });
  });
}
