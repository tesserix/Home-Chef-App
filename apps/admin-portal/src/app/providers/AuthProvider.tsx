import {
  createContext,
  useContext,
  useEffect,
  useCallback,
  type ReactNode,
} from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '../store/auth-store';
import type { SessionUser } from '@/shared/types/auth';

// Same-origin /bff/ proxy — Istio routes /bff/* to auth-bff with x-auth-context: admin
const BFF_URL = (() => {
  const env = import.meta.env.VITE_BFF_URL;
  if (env) return env;
  if (typeof window !== 'undefined' && window.location.hostname !== 'localhost') {
    return `${window.location.origin}/bff`;
  }
  return '/bff';
})();

interface AuthContextValue {
  user: SessionUser | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  csrfToken: string | null;
  login: (provider?: 'google' | 'facebook') => void;
  loginWithEmail: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const navigate = useNavigate();
  const {
    user,
    isAuthenticated,
    isLoading,
    csrfToken,
    clearAuth,
    initialize,
  } = useAuthStore();

  useEffect(() => {
    initialize();
  }, [initialize]);

  const login = useCallback((provider?: 'google' | 'facebook') => {
    const params = new URLSearchParams();
    params.set('returnTo', `${window.location.origin}/dashboard`);
    if (provider) {
      params.set('kc_idp_hint', provider);
    }
    window.location.href = `${BFF_URL}/auth/login?${params.toString()}`;
  }, []);

  const loginWithEmail = useCallback(async (email: string, password: string) => {
    const { authService } = await import('@/features/auth/services/auth-service');
    const result = await authService.loginWithEmail(email, password);

    // Verify admin role before allowing login
    // API returns role as singular string (user.role) not array (user.roles)
    const role = (result.user as Record<string, unknown>).role as string || '';
    const roles = result.user.roles || [];
    const isAdmin = role === 'admin' || role === 'super_admin'
      || roles.includes('admin') || roles.includes('super_admin');
    if (!isAdmin) {
      throw new Error('Access denied. Only administrators can sign in to this portal.');
    }

    const { setApiAuth } = useAuthStore.getState();
    setApiAuth(result.user, result.accessToken, result.refreshToken);
  }, []);

  const logout = useCallback(async () => {
    // Revoke API JWT if present
    const { refreshToken: rt } = useAuthStore.getState();
    if (rt) {
      const { authService } = await import('@/features/auth/services/auth-service');
      await authService.logoutApi(rt);
    }
    try {
      await fetch(`${BFF_URL}/auth/logout`, {
        method: 'POST',
        credentials: 'include',
      });
    } catch {
      // Ignore - clear local state regardless
    }
    clearAuth();
    navigate('/login');
  }, [clearAuth, navigate]);

  const value: AuthContextValue = {
    user,
    isAuthenticated,
    isLoading,
    csrfToken,
    login,
    loginWithEmail,
    logout,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
