import { create } from 'zustand';
import type { SessionUser } from '@/shared/types/auth';

interface AuthState {
  user: SessionUser | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  csrfToken: string | null;
  setSession: (user: SessionUser, csrfToken?: string) => void;
  clearAuth: () => void;
  setLoading: (loading: boolean) => void;
  initialize: () => Promise<void>;
}

export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  isAuthenticated: false,
  isLoading: true,
  csrfToken: null,

  setSession: (user, csrfToken) =>
    set({ user, isAuthenticated: true, csrfToken: csrfToken ?? get().csrfToken }),

  clearAuth: () =>
    set({ user: null, isAuthenticated: false, csrfToken: null }),

  setLoading: (isLoading) => set({ isLoading }),

  initialize: async () => {
    try {
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
