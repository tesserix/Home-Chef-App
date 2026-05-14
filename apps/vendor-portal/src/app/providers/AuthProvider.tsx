import {
  createContext,
  useContext,
  useEffect,
  useCallback,
  useState,
  type ReactNode,
} from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { useAuthStore } from '../store/auth-store';
import type { SessionUser, SocialProvider } from '@/shared/types/auth';

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
  needsOnboarding: boolean;
  onboardingStatus: string;
  adminNotes: string;
  login: (provider?: SocialProvider) => Promise<void>;
  register: () => Promise<void>;
  loginWithEmail: (email: string, password: string) => Promise<void>;
  registerWithEmail: (data: { email: string; password: string; firstName: string; lastName: string }) => Promise<void>;
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
    clearAuth,
    initialize,
  } = useAuthStore();
  const [needsOnboarding, setNeedsOnboarding] = useState(false);
  const [onboardingChecked, setOnboardingChecked] = useState(false);
  const [onboardingStatus, setOnboardingStatus] = useState('');
  const [adminNotes, setAdminNotes] = useState('');

  useEffect(() => {
    initialize();
  }, [initialize]);

  // After auth, check if chef profile exists and hydrate form data from server
  useEffect(() => {
    if (!isAuthenticated || isLoading || onboardingChecked) return;

    const checkOnboarding = async () => {
      try {
        // Attach the Firebase ID token (email/password login) the same way
        // api-client does. Without this header, the BFF returns 401 for users
        // who logged in with email/password (no Firebase cookie session),
        // which used to cascade into AuthProvider routing them back to
        // /onboarding forever.
        const { accessToken } = useAuthStore.getState();
        const headers: Record<string, string> = {};
        if (accessToken) headers['X-Auth-Token'] = accessToken;

        const res = await fetch(`${BFF_URL}/api/v1/chef/onboarding/status`, {
          credentials: 'include',
          headers,
        });
        if (!res.ok) {
          setNeedsOnboarding(true);
          if (!location.pathname.startsWith('/onboarding')) {
            navigate('/onboarding', { replace: true });
          }
          return;
        }
        const status = await res.json();
        setOnboardingStatus(status.status || '');
        setAdminNotes(status.adminNotes || '');
        if (status.completed) {
          setNeedsOnboarding(false);
        } else {
          // Only redirect to onboarding for first-time users who haven't submitted yet.
          // For rejected/info_requested, the chef has already submitted their profile —
          // let them stay on dashboard and see the notification in Admin Requests.
          const shouldRedirectToOnboarding =
            status.status === 'not_started' || status.status === 'in_progress';

          if (shouldRedirectToOnboarding) {
            setNeedsOnboarding(true);

            // Hydrate the onboarding form with server data.
            // First check if the local store has data from a different user and reset if so.
            const { useOnboardingStore } = await import('@/app/store/onboarding-store');
            const localEmail = useOnboardingStore.getState().data.email;
            const currentEmail = user?.email || status.profile?.email || '';
            if (localEmail && currentEmail && localEmail !== currentEmail) {
              // Different user logged in - clear stale data
              useOnboardingStore.getState().reset();
            }
            if (status.profile) {
              useOnboardingStore.getState().hydrateFromServer(status.step || 0, status.profile);
            }

            if (!location.pathname.startsWith('/onboarding')) {
              navigate('/onboarding', { replace: true });
            }
          } else {
            // rejected or info_requested — profile already submitted, no redirect
            setNeedsOnboarding(false);
          }
        }
      } catch {
        setNeedsOnboarding(false);
      } finally {
        setOnboardingChecked(true);
      }
    };

    checkOnboarding();
  }, [isAuthenticated, isLoading, onboardingChecked, navigate, location.pathname, user?.email]);

  const login = useCallback(async (provider?: SocialProvider) => {
    const svc = await import('@/features/auth/services/auth-service');
    const session =
      provider === 'apple'
        ? await svc.signInWithApple()
        : await svc.signInWithGoogle();
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

  const registerWithEmail = useCallback(async (data: { email: string; password: string; firstName: string; lastName: string }) => {
    const { authService } = await import('@/features/auth/services/auth-service');
    const result = await authService.registerWithEmail(data);
    const { setApiAuth } = useAuthStore.getState();
    setApiAuth(result.user, result.accessToken, result.refreshToken);
  }, []);

  const logout = useCallback(async () => {
    const { authService } = await import('@/features/auth/services/auth-service');
    await authService.logout();
    clearAuth();
    setOnboardingChecked(false);
    setNeedsOnboarding(false);
    // Clear onboarding form data to prevent cross-user contamination
    import('@/app/store/onboarding-store').then(({ useOnboardingStore }) => {
      useOnboardingStore.getState().reset();
    });
    navigate('/login');
  }, [clearAuth, navigate]);

  const value: AuthContextValue = {
    user,
    isAuthenticated,
    isLoading,
    csrfToken,
    needsOnboarding,
    onboardingStatus,
    adminNotes,
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
