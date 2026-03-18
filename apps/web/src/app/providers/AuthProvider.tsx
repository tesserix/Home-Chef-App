import {
  createContext,
  useContext,
  useEffect,
  useCallback,
  type ReactNode,
} from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { useAuthStore } from '../store/auth-store';
import { useFavoritesStore } from '../store/favorites-store';
import { useCurrencyStore } from '../store/currency-store';
import { apiClient } from '@/shared/services/api-client';
import type { SessionUser, SocialProvider } from '@/shared/types/auth';
import type { OnboardingStatus, CustomerProfile } from '@/shared/types';

const BFF_URL = import.meta.env.VITE_BFF_URL || 'https://identity.fe3dr.com';
// Same-origin proxy for fetch calls — browser redirects still use full BFF_URL
const _isLocal = typeof window !== 'undefined' && window.location.hostname === 'localhost';
const BFF_FETCH = _isLocal ? BFF_URL : '/bff';

interface AuthContextValue {
  user: SessionUser | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  csrfToken: string | null;
  login: (provider?: SocialProvider) => void;
  register: () => void;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const navigate = useNavigate();
  const location = useLocation();
  const {
    user,
    isAuthenticated,
    isLoading,
    csrfToken,
    onboardingCompleted,
    clearAuth,
    setOnboardingCompleted,
    initialize,
  } = useAuthStore();

  useEffect(() => {
    initialize();
  }, [initialize]);

  // Load favorite chef IDs once authenticated
  useEffect(() => {
    if (isAuthenticated) {
      useFavoritesStore.getState().load();
    } else {
      useFavoritesStore.getState().clear();
    }
  }, [isAuthenticated]);

  // Initialize currency: fetch rates/currencies, detect or restore preference
  useEffect(() => {
    const cs = useCurrencyStore.getState();
    cs.fetchCurrencies().then(() => {
      cs.fetchRates();
      if (isAuthenticated && !isLoading) {
        // Load preferred currency from profile
        apiClient
          .get<CustomerProfile>('/customer/profile')
          .then((profile) => {
            if (profile.preferredCurrency) {
              cs.initFromProfile(profile.preferredCurrency);
            }
          })
          .catch(() => {});
      } else if (!isLoading && !cs.detected) {
        // Guest: auto-detect from IP
        cs.detectCurrency();
      }
    });
  }, [isAuthenticated, isLoading]);

  // Check onboarding status after auth resolves
  useEffect(() => {
    if (!isAuthenticated || isLoading || onboardingCompleted !== null) return;

    apiClient
      .get<OnboardingStatus>('/customer/onboarding/status')
      .then((status) => {
        setOnboardingCompleted(status.onboardingCompleted);
        if (!status.onboardingCompleted && location.pathname !== '/user-info') {
          navigate('/user-info', { replace: true });
        }
      })
      .catch(() => {
        // If the check fails, don't block the user
        setOnboardingCompleted(true);
      });
  }, [isAuthenticated, isLoading, onboardingCompleted, navigate, location.pathname, setOnboardingCompleted]);

  const login = useCallback((provider?: SocialProvider) => {
    const params = new URLSearchParams();
    params.set('returnTo', window.location.origin);
    if (provider) {
      params.set('kc_idp_hint', provider);
    }
    window.location.href = `${BFF_URL}/auth/login?${params.toString()}`;
  }, []);

  const register = useCallback(() => {
    const params = new URLSearchParams();
    params.set('returnTo', window.location.origin);
    params.set('kc_action', 'register');
    window.location.href = `${BFF_URL}/auth/login?${params.toString()}`;
  }, []);

  const logout = useCallback(async () => {
    try {
      await fetch(`${BFF_FETCH}/auth/logout`, {
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
    register,
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
