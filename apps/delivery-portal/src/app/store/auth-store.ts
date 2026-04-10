import { create } from 'zustand';
import type { SessionUser } from '@/shared/types/auth';

const TOKEN_KEY = 'fe3dr_delivery_access_token';
const REFRESH_KEY = 'fe3dr_delivery_refresh_token';

interface AuthState {
  user: SessionUser | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  csrfToken: string | null;
  accessToken: string | null;
  refreshToken: string | null;
  authMode: 'staff' | 'driver' | null;
  setSession: (user: SessionUser, csrfToken?: string) => void;
  setApiAuth: (user: SessionUser, accessToken: string, refreshToken: string) => void;
  clearAuth: () => void;
  setLoading: (loading: boolean) => void;
  initialize: () => Promise<void>;
}

// Both driver and staff use the same customer realm BFF (/bff/).
// Staff are 3rd party fleet managers, not platform admins.
const BFF_URL = (() => {
  const env = import.meta.env.VITE_BFF_URL;
  if (env) return env;
  if (typeof window !== 'undefined' && window.location.hostname !== 'localhost') {
    return `${window.location.origin}/bff`;
  }
  return '/bff';
})();

async function trySession(): Promise<{ authenticated: boolean; user?: SessionUser; csrfToken?: string } | null> {
  try {
    const res = await fetch(`${BFF_URL}/auth/session`, { credentials: 'include' });
    if (!res.ok) return null;
    const data = await res.json();
    if (data?.authenticated && data.user) return data;
    return null;
  } catch {
    return null;
  }
}

export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  isAuthenticated: false,
  isLoading: true,
  csrfToken: null,
  accessToken: null,
  refreshToken: null,
  authMode: null,

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
    set({ user: null, isAuthenticated: false, csrfToken: null, accessToken: null, refreshToken: null, authMode: null });
  },

  setLoading: (isLoading) => set({ isLoading }),

  initialize: async () => {
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

    // 2. Check BFF session (social login via customer realm — single realm)
    const savedMode = localStorage.getItem('fe3dr-auth-mode') as 'staff' | 'driver' | null;
    const session = await trySession();

    if (session) {
      localStorage.setItem('fe3dr-auth-mode', savedMode || 'driver');
      set({
        user: session.user!,
        isAuthenticated: true,
        csrfToken: session.csrfToken ?? null,
        authMode: savedMode || 'driver',
        isLoading: false,
      });
    } else {
      set({ user: null, isAuthenticated: false, isLoading: false, authMode: null });
    }
  },
}));
