import {
  createContext,
  useContext,
  useEffect,
  useCallback,
  type ReactNode,
} from 'react';
import { useNavigate, useLocation } from 'react-router';
import { useAuthStore } from '../store/auth-store';
import { useFavoritesStore } from '../store/favorites-store';
import { useCurrencyStore } from '../store/currency-store';
import { apiClient, ACCOUNT_BLOCKED_EVENT, AUTH_EXPIRED_EVENT } from '@/shared/services/api-client';
import type { SessionUser, SocialProvider } from '@/shared/types/auth';
import type { OnboardingStatus, CustomerProfile } from '@/shared/types';

interface AuthContextValue {
  user: SessionUser | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  csrfToken: string | null;
  login: (provider?: SocialProvider) => Promise<void>;
  register: () => Promise<void>;
  loginWithEmail: (email: string, password: string) => Promise<void>;
  registerWithEmail: (data: { email: string; password: string; firstName: string; lastName: string; marketingConsent?: boolean }) => Promise<void>;
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

  // Global 401 recovery. apiClient dispatches `auth:expired` on a 401 from
  // any non-auth endpoint; we clear local auth state and bounce to /login
  // with a flag so the page can show "Your session has expired."
  // The endpoint detail is used to avoid bouncing on /auth/* endpoints
  // (login attempts naturally 401 on bad credentials — that's a user error,
  // not session expiry).
  useEffect(() => {
    const handler = () => {
      if (!useAuthStore.getState().isAuthenticated) return;
      clearAuth();
      // Preserve the page user was on so they return after re-auth.
      const returnTo = encodeURIComponent(location.pathname + location.search);
      navigate(`/login?error=session_expired&returnTo=${returnTo}`);
    };
    window.addEventListener(AUTH_EXPIRED_EVENT, handler);
    return () => window.removeEventListener(AUTH_EXPIRED_EVENT, handler);
  }, [clearAuth, navigate, location.pathname, location.search]);

  // A paused account 403s on every path except /me/reactivate — the session
  // itself is still valid (pausing is not a sign-out), so unlike auth:expired
  // above this does NOT clear auth. It only routes a user who is deep on some
  // other page to where reactivation is offered, rather than leaving every
  // query on that page failing silently. DataPrivacyPage itself listens for
  // the same event to render the paused/deleted state — this is only for
  // the case where the 403 arrives somewhere else.
  useEffect(() => {
    const handler = (e: Event) => {
      const status = (e as CustomEvent<{ status?: string }>).detail?.status;
      if (status !== 'account_deactivated') return;
      if (!useAuthStore.getState().isAuthenticated) return;
      if (location.pathname === '/data-privacy') return;
      navigate('/data-privacy');
    };
    window.addEventListener(ACCOUNT_BLOCKED_EVENT, handler);
    return () => window.removeEventListener(ACCOUNT_BLOCKED_EVENT, handler);
  }, [navigate, location.pathname]);

  // Load favorite chef + dish IDs once authenticated (#237)
  useEffect(() => {
    if (isAuthenticated) {
      useFavoritesStore.getState().load();
      useFavoritesStore.getState().loadDishes();
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

  const login = useCallback(async (provider?: SocialProvider) => {
    const svc = await import('@/features/auth/services/auth-service');
    const session =
      provider === 'apple' ? await svc.signInWithApple() : await svc.signInWithGoogle();
    const { setApiAuth } = useAuthStore.getState();
    setApiAuth(svc.toSessionUser(session), '', '');
  }, []);

  const register = useCallback(async () => {
    // Registration via social provider mirrors login; email/password
    // registration is handled by RegisterPage via `registerWithEmail`.
    const svc = await import('@/features/auth/services/auth-service');
    const session = await svc.signInWithGoogle();
    const { setApiAuth } = useAuthStore.getState();
    setApiAuth(svc.toSessionUser(session), '', '');
  }, []);

  const loginWithEmail = useCallback(async (email: string, password: string) => {
    const { authService } = await import('@/features/auth/services/auth-service');
    const result = await authService.loginWithEmail(email, password);
    const { setApiAuth } = useAuthStore.getState();
    setApiAuth(result.user, result.accessToken, result.refreshToken);
  }, []);

  const registerWithEmail = useCallback(async (data: { email: string; password: string; firstName: string; lastName: string; marketingConsent?: boolean }) => {
    const { authService } = await import('@/features/auth/services/auth-service');
    const result = await authService.registerWithEmail(data);
    const { setApiAuth } = useAuthStore.getState();
    setApiAuth(result.user, result.accessToken, result.refreshToken);
  }, []);

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
    register,
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
