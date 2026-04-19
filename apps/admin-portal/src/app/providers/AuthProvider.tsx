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
import type { LoginResult } from '@/features/auth/services/auth-service';

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
  loginWithEmail: (email: string, password: string) => Promise<LoginResult>;
  completeTotpLogin: (
    challengeToken: string,
    code: string,
    mode: 'verify' | 'enroll',
  ) => Promise<void>;
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

  const finalizeSuccess = useCallback(
    (user: SessionUser, accessToken: string, refreshToken: string) => {
      // API returns role as singular string (user.role) not array (user.roles).
      const role = (user as unknown as { role?: string }).role || '';
      const roles = user.roles || [];
      const isAdmin =
        role === 'admin' ||
        role === 'super_admin' ||
        roles.includes('admin') ||
        roles.includes('super_admin');
      if (!isAdmin) {
        throw new Error('Access denied. Only administrators can sign in to this portal.');
      }
      const { setApiAuth } = useAuthStore.getState();
      setApiAuth(user, accessToken, refreshToken);
    },
    [],
  );

  const loginWithEmail = useCallback(
    async (email: string, password: string): Promise<LoginResult> => {
      const { authService } = await import('@/features/auth/services/auth-service');
      const result = await authService.loginWithEmail(email, password);
      if (result.kind === 'success') {
        finalizeSuccess(result.user, result.accessToken, result.refreshToken);
      }
      return result;
    },
    [finalizeSuccess],
  );

  // completeTotpLogin finishes a login that needed a 2FA code. `mode` picks
  // which backend path we hit:
  //   - verify: the user already has TOTP enrolled; just validate the code
  //   - enroll: forced enrollment flow; the code is from their newly-scanned QR
  const completeTotpLogin = useCallback(
    async (challengeToken: string, code: string, mode: 'verify' | 'enroll') => {
      const { authService } = await import('@/features/auth/services/auth-service');
      const result =
        mode === 'verify'
          ? await authService.verifyTotp(challengeToken, code)
          : await authService.confirmTotpDuringLogin(challengeToken, code);
      finalizeSuccess(result.user, result.accessToken, result.refreshToken);
    },
    [finalizeSuccess],
  );

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
    completeTotpLogin,
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
