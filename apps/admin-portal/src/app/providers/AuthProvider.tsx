import {
  createContext,
  useContext,
  useEffect,
  useCallback,
  type ReactNode,
} from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '../store/auth-store';
import type { SessionUser, SocialProvider } from '@/shared/types/auth';

interface AuthContextValue {
  user: SessionUser | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  csrfToken: string | null;
  login: (provider?: SocialProvider) => Promise<void>;
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

  const login = useCallback(async (_provider?: SocialProvider) => {
    // Admin portal supports Google only; the BFF allowlist enforces
    // membership server-side, so a successful exchange always means the
    // signed-in account is authorized.
    const svc = await import('@/features/auth/services/auth-service');
    const session = await svc.signInWithGoogle();
    const sessionUser = svc.toSessionUser(session);
    finalizeSuccess(sessionUser, '', '');
  }, [finalizeSuccess]);

  const loginWithEmail = useCallback(
    async (email: string, password: string): Promise<void> => {
      // MFA is deferred per the migration spec; a successful BFF exchange
      // already enforces the admin allowlist server-side.
      const svc = await import('@/features/auth/services/auth-service');
      const session = await svc.authService.loginWithEmail(email, password);
      const sessionUser = svc.toSessionUser(session);
      finalizeSuccess(sessionUser, '', '');
    },
    [finalizeSuccess],
  );

  const logout = useCallback(async () => {
    const { authService } = await import('@/features/auth/services/auth-service');
    await authService.logout();
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
