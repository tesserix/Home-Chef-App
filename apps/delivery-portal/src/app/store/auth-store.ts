import { create } from 'zustand';
import type { SessionUser } from '@/shared/types/auth';

interface AuthState {
  user: SessionUser | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  csrfToken: string | null;
  authMode: 'staff' | 'driver' | null;
  setSession: (user: SessionUser, csrfToken?: string) => void;
  clearAuth: () => void;
  setLoading: (loading: boolean) => void;
  initialize: () => Promise<void>;
}

const STAFF_BFF_URL = (() => {
  const env = import.meta.env.VITE_BFF_URL;
  if (env) return env;
  if (typeof window !== 'undefined' && window.location.hostname !== 'localhost') {
    return `${window.location.origin}/bff`;
  }
  return '/bff';
})();

const DRIVER_BFF_URL = (() => {
  const env = import.meta.env.VITE_DRIVER_BFF_URL;
  if (env) return env;
  if (typeof window !== 'undefined' && window.location.hostname !== 'localhost') {
    return `${window.location.origin}/driver-bff`;
  }
  return '/driver-bff';
})();

async function trySession(bffUrl: string): Promise<{ authenticated: boolean; user?: SessionUser; csrfToken?: string } | null> {
  try {
    const res = await fetch(`${bffUrl}/auth/session`, { credentials: 'include' });
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
  authMode: null,

  setSession: (user, csrfToken) =>
    set({ user, isAuthenticated: true, csrfToken: csrfToken ?? get().csrfToken }),

  clearAuth: () =>
    set({ user: null, isAuthenticated: false, csrfToken: null, authMode: null }),

  setLoading: (isLoading) => set({ isLoading }),

  initialize: async () => {
    // Check which mode was last used
    const savedMode = localStorage.getItem('fe3dr-auth-mode') as 'staff' | 'driver' | null;

    // Try the saved mode first, then the other
    const primaryBff = savedMode === 'driver' ? DRIVER_BFF_URL : STAFF_BFF_URL;
    const primaryMode = savedMode === 'driver' ? 'driver' : 'staff';
    const secondaryBff = savedMode === 'driver' ? STAFF_BFF_URL : DRIVER_BFF_URL;
    const secondaryMode = savedMode === 'driver' ? 'staff' : 'driver';

    // Try primary BFF
    let session = await trySession(primaryBff);
    let activeMode: 'staff' | 'driver' | null = null;

    if (session) {
      activeMode = primaryMode;
    } else {
      // Try secondary BFF
      session = await trySession(secondaryBff);
      if (session) {
        activeMode = secondaryMode;
      }
    }

    if (session && activeMode) {
      // For staff mode, verify the user has an authorized role
      if (activeMode === 'staff') {
        const roles = session.user?.roles || [];
        const isAuthorized =
          roles.includes('admin') ||
          roles.includes('super_admin') ||
          roles.includes('delivery') ||
          roles.includes('fleet_manager');

        if (!isAuthorized) {
          set({ user: null, isAuthenticated: false, isLoading: false, authMode: null });
          return;
        }
      }

      localStorage.setItem('fe3dr-auth-mode', activeMode);
      set({
        user: session.user!,
        isAuthenticated: true,
        csrfToken: session.csrfToken ?? null,
        authMode: activeMode,
        isLoading: false,
      });
    } else {
      set({ user: null, isAuthenticated: false, isLoading: false, authMode: null });
    }
  },
}));
