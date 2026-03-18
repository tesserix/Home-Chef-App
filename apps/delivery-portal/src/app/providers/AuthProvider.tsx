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

// Staff BFF — internal Keycloak realm (tesserix-internal) via x-auth-context: admin
const STAFF_BFF_URL = (() => {
  const env = import.meta.env.VITE_BFF_URL;
  if (env) return env;
  if (typeof window !== 'undefined' && window.location.hostname !== 'localhost') {
    return `${window.location.origin}/bff`;
  }
  return '/bff';
})();

// Driver BFF — customer Keycloak realm (homechef) via /driver-bff (no x-auth-context)
const DRIVER_BFF_URL = (() => {
  const env = import.meta.env.VITE_DRIVER_BFF_URL;
  if (env) return env;
  if (typeof window !== 'undefined' && window.location.hostname !== 'localhost') {
    return `${window.location.origin}/driver-bff`;
  }
  return '/driver-bff';
})();

interface AuthContextValue {
  user: SessionUser | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  csrfToken: string | null;
  authMode: 'staff' | 'driver' | null;
  loginStaff: (provider?: 'google' | 'facebook') => void;
  loginDriver: (provider?: 'google' | 'facebook') => void;
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
    window.location.href = `${STAFF_BFF_URL}/auth/login?${params.toString()}`;
  }, []);

  const loginDriver = useCallback((provider?: 'google' | 'facebook') => {
    localStorage.setItem('fe3dr-auth-mode', 'driver');
    const params = new URLSearchParams();
    params.set('returnTo', `${window.location.origin}/dashboard`);
    if (provider) {
      params.set('kc_idp_hint', provider);
    }
    window.location.href = `${DRIVER_BFF_URL}/auth/login?${params.toString()}`;
  }, []);

  const logout = useCallback(async () => {
    const bffUrl = authMode === 'driver' ? DRIVER_BFF_URL : STAFF_BFF_URL;
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
