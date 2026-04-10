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

// Both driver and staff use the customer Keycloak realm (homechef).
// Staff are 3rd party fleet managers, NOT platform admins — they register
// alongside drivers in the same realm. Platform admins use admin.fe3dr.com.
// The /bff/ and /driver-bff/ routes on delivery.fe3dr.com both default to
// the customer realm (no x-auth-context header).
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
  authMode: 'staff' | 'driver' | null;
  loginStaff: (provider?: 'google' | 'facebook') => void;
  loginDriver: (provider?: 'google' | 'facebook') => void;
  loginWithEmail: (email: string, password: string) => Promise<void>;
  registerWithEmail: (data: { email: string; password: string; firstName: string; lastName: string }) => Promise<void>;
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
    authMode,
    clearAuth,
    initialize,
  } = useAuthStore();

  useEffect(() => {
    initialize();
  }, [initialize]);

  const loginStaff = useCallback((provider?: 'google' | 'facebook') => {
    localStorage.setItem('fe3dr-auth-mode', 'staff');
    const params = new URLSearchParams();
    params.set('returnTo', `${window.location.origin}/dashboard`);
    if (provider) {
      params.set('kc_idp_hint', provider);
    }
    window.location.href = `${BFF_URL}/auth/login?${params.toString()}`;
  }, []);

  const loginDriver = useCallback((provider?: 'google' | 'facebook') => {
    localStorage.setItem('fe3dr-auth-mode', 'driver');
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
    const { setApiAuth } = useAuthStore.getState();
    setApiAuth(result.user, result.accessToken, result.refreshToken);
  }, []);

  const registerWithEmail = useCallback(async (data: { email: string; password: string; firstName: string; lastName: string }) => {
    const { authService } = await import('@/features/auth/services/auth-service');
    const result = await authService.registerWithEmail(data);
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
    const bffUrl = BFF_URL;
    try {
      await fetch(`${bffUrl}/auth/logout`, {
        method: 'POST',
        credentials: 'include',
      });
    } catch {
      // Ignore - clear local state regardless
    }
    localStorage.removeItem('fe3dr-auth-mode');
    clearAuth();
    navigate('/login');
  }, [authMode, clearAuth, navigate]);

  const value: AuthContextValue = {
    user,
    isAuthenticated,
    isLoading,
    csrfToken,
    authMode,
    loginStaff,
    loginDriver,
    loginWithEmail,
    registerWithEmail,
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
