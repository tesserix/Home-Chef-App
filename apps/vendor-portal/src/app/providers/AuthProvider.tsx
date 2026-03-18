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

const BFF_URL = import.meta.env.VITE_BFF_URL || 'https://identity.fe3dr.com';

interface AuthContextValue {
  user: SessionUser | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  csrfToken: string | null;
  needsOnboarding: boolean;
  onboardingStatus: string;
  adminNotes: string;
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
        const res = await fetch(`${BFF_URL}/api/v1/chef/onboarding/status`, {
          credentials: 'include',
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
  }, [isAuthenticated, isLoading, onboardingChecked, navigate, location.pathname]);

  const login = useCallback((provider?: SocialProvider) => {
    const params = new URLSearchParams();
    params.set('returnTo', `${window.location.origin}/dashboard`);
    if (provider) {
      params.set('kc_idp_hint', provider);
    }
    window.location.href = `${BFF_URL}/auth/login?${params.toString()}`;
  }, []);

  const register = useCallback(() => {
    const params = new URLSearchParams();
    params.set('returnTo', `${window.location.origin}/dashboard`);
    params.set('kc_action', 'register');
    window.location.href = `${BFF_URL}/auth/login?${params.toString()}`;
  }, []);

  const logout = useCallback(async () => {
    try {
      await fetch(`${BFF_URL}/auth/logout`, {
        method: 'POST',
        credentials: 'include',
      });
    } catch {
      // Ignore - clear local state regardless
    }
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
