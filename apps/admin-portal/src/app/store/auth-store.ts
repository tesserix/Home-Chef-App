import { create } from 'zustand';
import type { SessionUser } from '@/shared/types/auth';

const TOKEN_KEY = 'fe3dr_admin_access_token';
const REFRESH_KEY = 'fe3dr_admin_refresh_token';

interface AuthState {
  user: SessionUser | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  csrfToken: string | null;
  accessToken: string | null;
  refreshToken: string | null;
  setSession: (user: SessionUser, csrfToken?: string) => void;
  setApiAuth: (user: SessionUser, accessToken: string, refreshToken: string) => void;
  clearAuth: () => void;
  setLoading: (loading: boolean) => void;
  initialize: () => Promise<void>;
}

export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  isAuthenticated: false,
  isLoading: true,
  csrfToken: null,
  accessToken: null,
  refreshToken: null,

  // BFF session-based auth (social login via Keycloak)
  setSession: (user, csrfToken) =>
    set({ user, isAuthenticated: true, csrfToken: csrfToken ?? get().csrfToken }),

  // Direct API auth (email/password login — JWT-based)
  setApiAuth: (user, accessToken, refreshToken) => {
    localStorage.setItem(TOKEN_KEY, accessToken);
    localStorage.setItem(REFRESH_KEY, refreshToken);
    set({ user, isAuthenticated: true, accessToken, refreshToken });
  },

  clearAuth: () => {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(REFRESH_KEY);
    set({
      user: null,
      isAuthenticated: false,
      csrfToken: null,
      accessToken: null,
      refreshToken: null,
    });
  },

  setLoading: (isLoading) => set({ isLoading }),

  initialize: async () => {
    try {
      // 1. Check for stored API JWT (email/password login)
      const storedToken = localStorage.getItem(TOKEN_KEY);
      const storedRefresh = localStorage.getItem(REFRESH_KEY);

      if (storedToken && storedRefresh) {
        try {
          const { authService } = await import(
            '@/features/auth/services/auth-service'
          );
          const result = await authService.refreshApiToken(storedRefresh);
          const payload = JSON.parse(atob(result.accessToken.split('.')[1] ?? ''));
          const user: SessionUser = {
            id: payload.userId || payload.sub,
            email: payload.email,
            firstName: payload.firstName,
            lastName: payload.lastName,
            roles: payload.role ? [payload.role] : [],
          };

          // Verify the user has admin or super_admin role
          const roles = user.roles || [];
          const isAdmin = roles.includes('admin') || roles.includes('super_admin');
          if (!isAdmin) {
            localStorage.removeItem(TOKEN_KEY);
            localStorage.removeItem(REFRESH_KEY);
            set({ user: null, isAuthenticated: false, isLoading: false });
            return;
          }

          localStorage.setItem(TOKEN_KEY, result.accessToken);
          localStorage.setItem(REFRESH_KEY, result.refreshToken);
          set({
            user,
            isAuthenticated: true,
            accessToken: result.accessToken,
            refreshToken: result.refreshToken,
            isLoading: false,
          });
          return;
        } catch {
          localStorage.removeItem(TOKEN_KEY);
          localStorage.removeItem(REFRESH_KEY);
        }
      }

      // 2. Check BFF session (social login via Keycloak)
      const { authService } = await import('@/features/auth/services/auth-service');
      const session = await authService.getSession();

      if (session?.authenticated && session.user) {
        // Verify the user has admin or super_admin role
        const roles = session.user.roles || [];
        const isAdmin = roles.includes('admin') || roles.includes('super_admin');

        if (!isAdmin) {
          set({ user: null, isAuthenticated: false, isLoading: false });
          return;
        }

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
